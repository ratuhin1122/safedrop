package observability

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
)

// An upload rejected before persistence has an attempt UUID, not a database row.
func LogRejection(ctx context.Context, logger *slog.Logger, fileID, reason string) {
	if fileID == "" {
		var id [16]byte
		_, _ = rand.Read(id[:])
		id[6] = id[6]&15 | 64
		id[8] = id[8]&63 | 128
		fileID = fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:])
	}
	LoggerFrom(withRequestLogger(ctx, logger)).Warn("file rejected", "file_id", fileID, "reason", reason)
}
