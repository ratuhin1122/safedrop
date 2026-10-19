package observability

import (
	"context"
	"log/slog"
	"sync"
)

type loggerKey struct{}
type requestLogger struct {
	logger *slog.Logger
	mu     sync.RWMutex
	userID string
}

func withRequestLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, &requestLogger{logger: logger})
}

func SetUserID(ctx context.Context, id string) {
	if state, _ := ctx.Value(loggerKey{}).(*requestLogger); state != nil {
		state.mu.Lock()
		state.userID = id
		state.mu.Unlock()
	}
}

func UserID(ctx context.Context) string {
	if state, _ := ctx.Value(loggerKey{}).(*requestLogger); state != nil {
		state.mu.RLock()
		defer state.mu.RUnlock()
		return state.userID
	}
	return ""
}

// LoggerFrom binds context even when the caller uses Info/Warn without Context.
func LoggerFrom(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	if state, _ := ctx.Value(loggerKey{}).(*requestLogger); state != nil {
		logger = state.logger
	}
	return slog.New(fixedContextHandler{Handler: contextHandler{logger.Handler()}, ctx: ctx})
}

type fixedContextHandler struct {
	slog.Handler
	ctx context.Context
}

func (h fixedContextHandler) Handle(_ context.Context, record slog.Record) error {
	return h.Handler.Handle(h.ctx, record)
}
func (h fixedContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return fixedContextHandler{Handler: h.Handler.WithAttrs(attrs), ctx: h.ctx}
}
func (h fixedContextHandler) WithGroup(name string) slog.Handler {
	return fixedContextHandler{Handler: h.Handler.WithGroup(name), ctx: h.ctx}
}
