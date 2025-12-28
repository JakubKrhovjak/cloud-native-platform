package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/trace"
)

// New creates a new slog.Logger with trace context support
// Kubernetes/Production: JSONHandler (structured logging for log aggregation)
// Local development: TextHandler with colored output
// All handlers are wrapped with traceContextHandler to add trace_id/span_id
func New() *slog.Logger {
	_, inK8s := os.LookupEnv("KUBERNETES_SERVICE_HOST")

	env := os.Getenv("ENV")
	useJSON := inK8s || env == "prod" || env == "dev"

	var handler slog.Handler
	if useJSON {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		})
	} else {
		handler = newColorTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	return slog.New(newTraceContextHandler(handler))
}

func NewWithServiceContext(serviceName, version string) *slog.Logger {
	return New().With(
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

// traceContextHandler wraps any slog.Handler to add trace_id and span_id from OTel context
type traceContextHandler struct {
	handler slog.Handler
}

func newTraceContextHandler(h slog.Handler) *traceContextHandler {
	return &traceContextHandler{handler: h}
}

func (h *traceContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *traceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}
	return h.handler.Handle(ctx, r)
}

func (h *traceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceContextHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *traceContextHandler) WithGroup(name string) slog.Handler {
	return &traceContextHandler{handler: h.handler.WithGroup(name)}
}
