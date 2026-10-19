package jobs

import (
	"context"
	"errors"
	"fmt"

	"safedrop/internal/db"
)

var ErrPipelineUnavailable = errors.New("scan and image processing pipeline is not configured")

// Processor runs under a per-file row lock. It must use the supplied transaction
// queries for status updates and honor context cancellation. Success requires a
// terminal file status, or ErrProcessingPending after a clean scan. External
// storage operations must themselves be idempotent.
type Processor interface {
	Process(context.Context, *db.Queries, db.File) error
}

func runProcessor(ctx context.Context, processor Processor, queries *db.Queries, file db.File) (err error) {
	if processor == nil {
		return ErrPipelineUnavailable
	}
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("processor panic: %v", value)
		}
	}()
	return processor.Process(ctx, queries, file)
}
