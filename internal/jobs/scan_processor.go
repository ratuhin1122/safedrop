package jobs

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"safedrop/internal/db"
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
		result, scanErr := p.Scanner.Scan(ctx, body)
		closeErr := body.Close()
		if scanErr != nil {
			return errors.Join(scanErr, closeErr)
		}
		switch result.Verdict {
		case scan.Infected:
			if _, err := queries.RejectFile(ctx, file.ID); err != nil {
				return err
			}
			p.Logger.WarnContext(ctx, "file infected", "file_id", file.ID.String(), "signature", result.Signature)
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
