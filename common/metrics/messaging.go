package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type MessagingMetrics struct {
	messagesPublished         metric.Int64Counter
	messagesConsumed          metric.Int64Counter
	messageProcessingDuration metric.Float64Histogram
	messageErrors             metric.Int64Counter
	connectionsActive         metric.Int64UpDownCounter
	publishDuration           metric.Float64Histogram
}

func NewMessagingMetrics(meter metric.Meter) (*MessagingMetrics, error) {
	mm := &MessagingMetrics{}

	var err error

	// Messages published
	mm.messagesPublished, err = meter.Int64Counter(
		"messaging.messages.published",
		metric.WithDescription("Total number of messages published"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, err
	}

	// Messages consumed
	mm.messagesConsumed, err = meter.Int64Counter(
		"messaging.messages.consumed",
		metric.WithDescription("Total number of messages consumed"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, err
	}

	// Message processing duration with custom buckets for p95, p99
	// Buckets: 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
	mm.messageProcessingDuration, err = meter.Float64Histogram(
		"messaging.message.processing_duration",
		metric.WithDescription("Time spent processing a message"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.001, // 1ms
			0.005, // 5ms
			0.01,  // 10ms
			0.025, // 25ms
			0.05,  // 50ms
			0.1,   // 100ms
			0.25,  // 250ms
			0.5,   // 500ms
			1.0,   // 1s
			2.5,   // 2.5s
			5.0,   // 5s
			10.0,  // 10s
		),
	)
	if err != nil {
		return nil, err
	}

	// Publish duration with custom buckets optimized for network operations
	// Buckets: 100µs, 500µs, 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s
	mm.publishDuration, err = meter.Float64Histogram(
		"messaging.message.publish_duration",
		metric.WithDescription("Time spent publishing a message"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.0001, // 100µs
			0.0005, // 500µs
			0.001,  // 1ms
			0.005,  // 5ms
			0.01,   // 10ms
			0.025,  // 25ms
			0.05,   // 50ms
			0.1,    // 100ms
			0.25,   // 250ms
			0.5,    // 500ms
			1.0,    // 1s
		),
	)
	if err != nil {
		return nil, err
	}

	// Message errors
	mm.messageErrors, err = meter.Int64Counter(
		"messaging.message.errors",
		metric.WithDescription("Total number of message processing errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	// Active connections
	mm.connectionsActive, err = meter.Int64UpDownCounter(
		"messaging.connections.active",
		metric.WithDescription("Current number of active messaging connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}

	return mm, nil
}

func (mm *MessagingMetrics) RecordPublish(ctx context.Context, subject string, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("subject", subject),
	}

	mm.messagesPublished.Add(ctx, 1, metric.WithAttributes(attrs...))
	mm.publishDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil {
		errAttrs := append(attrs, attribute.String("error", err.Error()))
		mm.messageErrors.Add(ctx, 1, metric.WithAttributes(errAttrs...))
	}
}

func (mm *MessagingMetrics) RecordConsume(ctx context.Context, subject string, processingDuration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("subject", subject),
	}

	mm.messagesConsumed.Add(ctx, 1, metric.WithAttributes(attrs...))
	mm.messageProcessingDuration.Record(ctx, processingDuration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil {
		errAttrs := append(attrs,
			attribute.String("error", err.Error()),
			attribute.String("error_type", "processing"),
		)
		mm.messageErrors.Add(ctx, 1, metric.WithAttributes(errAttrs...))
	}
}

func (mm *MessagingMetrics) RecordConnectionChange(ctx context.Context, delta int64) {
	mm.connectionsActive.Add(ctx, delta)
}
