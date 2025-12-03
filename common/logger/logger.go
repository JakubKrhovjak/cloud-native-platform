package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// New creates a new slog.Logger
// Kubernetes/Production: JSONHandler (structured logging for log aggregation)
// Local development: TextHandler with colored output
func New() *slog.Logger {
	// Auto-detect Kubernetes environment
	_, inK8s := os.LookupEnv("KUBERNETES_SERVICE_HOST")

	// Or check ENV variable
	env := os.Getenv("ENV")
	useJSON := inK8s || env == "prod"

	var handler slog.Handler
	if useJSON {
		// JSON format for Kubernetes - structured logging
		// Benefits:
		// - Easy parsing by log aggregators (Loki, ELK, CloudWatch)
		// - Queryable fields (level, service, timestamp)
		// - Better performance than regex parsing
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true, // Include source file:line for debugging
		})
	} else {
		// Human-readable format for local development
		handler = newColorTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	return slog.New(handler)
}

// NewWithServiceContext creates a logger with service metadata
// Adds service, version, and environment fields to all log entries
func NewWithServiceContext(serviceName, version string) *slog.Logger {
	logger := New()

	// Add service context to all logs
	return logger.With(
		slog.String("service", serviceName),
		slog.String("version", version),
		slog.String("environment", os.Getenv("ENV")),
	)
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
