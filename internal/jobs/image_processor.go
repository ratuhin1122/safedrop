package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"safedrop/internal/db"
	"safedrop/internal/imageproc"
	"safedrop/internal/observability"
	"safedrop/internal/storage"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImageProcessor struct {
	Engine  *imageproc.Engine
	Storage storage.Store
	Pool    *pgxpool.Pool
}

func (p *ImageProcessor) Process(ctx context.Context, q *db.Queries, file db.File) error {
	if file.Status != "processing" {
		return errors.New("image processing requires a clean scan")
	}
	started := time.Now()
	variants, err := p.Engine.Generate(ctx, p.Storage, file.StorageKey, "final/"+file.ID.String())
	observability.DefaultMetrics.ObserveProcessing("convert", time.Since(started))
	if err != nil {
		return err
	}
	for _, variant := range variants {
		if err := q.UpsertFileVariant(ctx, db.UpsertFileVariantParams{FileID: file.ID,
			TargetWidth: variant.TargetWidth, Width: variant.Width, Height: variant.Height,
			Size: variant.Size, Mime: variant.MIME, StorageKey: variant.Key}); err != nil {
			return err
		}
	}
	if err := q.ScheduleFileCleanup(ctx, db.ScheduleFileCleanupParams{FileID: file.ID, StorageKey: file.StorageKey}); err != nil {
		return err
	}
	primary := variants[len(variants)-1]
	count, err := q.CompleteFile(ctx, db.CompleteFileParams{ID: file.ID, StorageKey: primary.Key,
		Mime: primary.MIME, Size: primary.Size, Width: pgtype.Int4{Int32: primary.Width, Valid: true},
		Height: pgtype.Int4{Int32: primary.Height, Valid: true}})
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("file did not transition to ready")
	}
	return nil
}
