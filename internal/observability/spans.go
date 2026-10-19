package observability

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// Span returns an end function that records no raw error message or payload.
func Span(ctx context.Context, name string) (context.Context, func(error)) {
	ctx, span := otel.Tracer("safedrop").Start(ctx, name)
	return ctx, func(err error) {
		if err != nil {
			span.SetStatus(codes.Error, "operation failed")
		}
		span.End()
	}
}
