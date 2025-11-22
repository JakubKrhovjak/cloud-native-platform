package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// New creates a new slog.Logger
// Development mode: TextHandler with colored ERROR level
// Production mode: JSONHandler
func New() *slog.Logger {
	env := os.Getenv("ENV")

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = newColorTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	return slog.New(handler)
}

// colorTextHandler wraps TextHandler to add red color to ERROR level
type colorTextHandler struct {
	handler slog.Handler
}

func newColorTextHandler(w io.Writer, opts *slog.HandlerOptions) *colorTextHandler {
	return &colorTextHandler{
		handler: slog.NewTextHandler(w, opts),
	}
}

func (h *colorTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *colorTextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add red color for ERROR level
	if r.Level >= slog.LevelError {
		// Create a new record with colored level
		newRecord := slog.NewRecord(r.Time, r.Level, fmt.Sprintf("\x1b[31m%s\x1b[0m", r.Message), r.PC)
		r.Attrs(func(a slog.Attr) bool {
			newRecord.AddAttrs(a)
			return true
		})
		return h.handler.Handle(ctx, newRecord)
	}

	return h.handler.Handle(ctx, r)
}

func (h *colorTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &colorTextHandler{
		handler: h.handler.WithAttrs(attrs),
	}
}

func (h *colorTextHandler) WithGroup(name string) slog.Handler {
	return &colorTextHandler{
		handler: h.handler.WithGroup(name),
	}
}
