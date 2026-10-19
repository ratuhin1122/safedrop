package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/observability"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type ProcessFile struct {
	river.WorkerDefaults[ProcessFileArgs]
	Pool       *pgxpool.Pool
	Processor  Processor
	Logger     *slog.Logger
	JobTimeout time.Duration
}

func (w *ProcessFile) Timeout(*river.Job[ProcessFileArgs]) time.Duration { return w.JobTimeout }

func (w *ProcessFile) NextRetry(job *river.Job[ProcessFileArgs]) time.Time {
	delay := time.Second << min(max(job.Attempt, 1), 6)
	return time.Now().Add(min(delay, time.Minute))
}

func (w *ProcessFile) work(ctx context.Context, job *river.Job[ProcessFileArgs]) error {
	var id pgtype.UUID
	if err := id.Scan(job.Args.FileID); err != nil || !id.Valid || id.Bytes == [16]byte{} {
		return river.JobCancel(errors.New("invalid process_file file ID"))
	}
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = tx.Rollback(cleanup)
	}()
	queries := db.New(tx)
	file, err := queries.LockFile(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		w.Logger.InfoContext(ctx, "process_file skipped", "file_id", job.Args.FileID, "reason", "missing or deleted")
		return nil
	}
	if err != nil {
		return err
	}
	if file.Status == "ready" || file.Status == "rejected" {
		w.Logger.InfoContext(ctx, "process_file skipped", "file_id", job.Args.FileID, "status", file.Status)
		_ = tx.Rollback(ctx)
		if file.Status == "ready" {
			return finalize(ctx, w.Processor, file)
		}
		if cleaner, ok := w.Processor.(interface {
			CleanupRejected(context.Context, db.File) error
		}); ok && file.Status == "rejected" {
			return cleaner.CleanupRejected(ctx, file)
		}
		return nil
	}
	w.Logger.InfoContext(ctx, "process_file started", "job_id", job.ID, "file_id", job.Args.FileID,
		"attempt", job.Attempt, "max_attempts", job.MaxAttempts, "status", file.Status)
	if file.Status == "uploaded" {
		if _, err := queries.AdvanceFile(ctx, db.AdvanceFileParams{ID: id, ExpectedStatus: "uploaded", NextStatus: "scanning"}); err != nil {
			return err
		}
		file.Status = "scanning"
	}
	processErr := runProcessor(ctx, w.Processor, queries, file)
	if errors.Is(processErr, ErrProcessingPending) {
		current, err := queries.GetFile(ctx, id)
		if err != nil {
			return err
		}
		if current.Status != "processing" {
			return errors.New("pending processing requires a clean scan")
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		w.Logger.InfoContext(ctx, "file awaiting image processing", "file_id", job.Args.FileID)
		return river.JobSnooze(time.Minute)
	}
	rejected := false
	if processErr == nil {
		current, err := queries.GetFile(ctx, id)
		if err != nil {
			return err
		}
		if current.Status != "ready" && current.Status != "rejected" {
			processErr = errors.New("processor returned before a terminal file status")
		}
		rejected = current.Status == "rejected"
	}
	if processErr != nil && job.Attempt >= job.MaxAttempts {
		if _, err := queries.RejectFile(ctx, id); err != nil {
			return errors.Join(processErr, err)
		}
		rejected = true
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Join(processErr, err)
	}
	if rejected {
		observability.DefaultMetrics.Rejected.Add(1)
	}
	if processErr != nil {
		w.Logger.ErrorContext(ctx, "process_file failed", "file_id", job.Args.FileID,
			"attempt", job.Attempt, "exhausted", job.Attempt >= job.MaxAttempts, "error", processErr)
		return fmt.Errorf("process file: %w", processErr)
	}
	w.Logger.InfoContext(ctx, "process_file completed", "file_id", job.Args.FileID)
	return finalize(ctx, w.Processor, file)
}

func finalize(ctx context.Context, processor Processor, file db.File) error {
	if finalizer, ok := processor.(interface {
		Finalize(context.Context, db.File) error
	}); ok {
		return finalizer.Finalize(ctx, file)
	}
	return nil
}
