package jobs

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"safedrop/internal/db"
	"safedrop/internal/observability"
	"safedrop/internal/scan"
	"safedrop/internal/storage"
)

var ErrProcessingPending = errors.New("image processing stage is not configured")

type ScanProcessor struct {
	Scanner scan.Scanner
	Storage storage.Store
	Logger  *slog.Logger
	Next    Processor
}

func (p *ScanProcessor) Process(ctx context.Context, queries *db.Queries, file db.File) error {
	if !strings.HasPrefix(file.StorageKey, "incoming/") {
		return errors.New("active file must be in incoming storage")
	}
	if file.Status == "scanning" {
		body, err := p.Storage.Get(ctx, file.StorageKey)
		if errors.Is(err, storage.ErrNotFound) {
			_, err := queries.RejectFile(ctx, file.ID)
			return err
		}
		if err != nil {
			return err
		}
		started := time.Now()
		scanCtx, endScan := observability.Span(ctx, "scan.clamav")
		result, scanErr := p.Scanner.Scan(scanCtx, body)
		endScan(scanErr)
		observeStage(ctx, "scan", time.Since(started))
		closeErr := body.Close()
		if scanErr != nil {
			return errors.Join(scanErr, closeErr)
		}
		switch result.Verdict {
		case scan.Infected:
			if report, _ := ctx.Value(reportKey{}).(*processReport); report != nil {
				report.virus = true
			}
			if _, err := queries.RejectFile(ctx, file.ID); err != nil {
				return err
			}
			if err := queries.SetVirusRejection(ctx, file.ID); err != nil {
				return err
			}
			observability.LogRejection(ctx, p.Logger, file.ID.String(), "virus")
			return errors.Join(closeErr, p.CleanupRejected(ctx, file))
		case scan.Clean:
			if closeErr != nil {
				return closeErr
			}
			if _, err := queries.AdvanceFile(ctx, db.AdvanceFileParams{ID: file.ID, ExpectedStatus: "scanning", NextStatus: "processing"}); err != nil {
				return err
			}
			file.Status = "processing"
			p.Logger.InfoContext(ctx, "file scan clean", "file_id", file.ID.String())
		default:
			return errors.New("scanner returned no verdict")
		}
	}
	if file.Status != "processing" {
		return errors.New("unexpected scan pipeline status")
	}
	if p.Next == nil {
		return ErrProcessingPending
	}
	return p.Next.Process(ctx, queries, file)
}

// CleanupRejected is retried when a rejected file's job is redelivered. Delete is
// idempotent, and only private incoming objects are eligible for removal here.
func (p *ScanProcessor) CleanupRejected(ctx context.Context, file db.File) error {
	if !strings.HasPrefix(file.StorageKey, "incoming/") {
		return nil
	}
	return p.Storage.Delete(ctx, file.StorageKey)
}

func (p *ScanProcessor) Finalize(ctx context.Context, file db.File) error {
	if next, ok := p.Next.(interface {
		Finalize(context.Context, db.File) error
	}); ok {
		return next.Finalize(ctx, file)
	}
	return nil
}

func (p *ScanProcessor) Maintain(ctx context.Context, pool *pgxpool.Pool) error {
	if next, ok := p.Next.(Maintainer); ok {
		return next.Maintain(ctx, pool)
	}
	return nil
}
