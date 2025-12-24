package metrics

import (
	"context"
	"database/sql"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type DatabaseMetrics struct {
	connectionsOpen     metric.Int64ObservableGauge
	connectionsIdle     metric.Int64ObservableGauge
	connectionsInUse    metric.Int64ObservableGauge
	connectionsWaitTime metric.Float64Histogram
	maxOpenConnections  metric.Int64ObservableGauge
	queryDuration       metric.Float64Histogram
	queryErrors         metric.Int64Counter
	db                  *sql.DB
}

func NewDatabaseMetrics(meter metric.Meter) (*DatabaseMetrics, error) {
	dm := &DatabaseMetrics{}

	var err error

	// Connection pool metrics
	dm.connectionsOpen, err = meter.Int64ObservableGauge(
		"db.connections.open",
		metric.WithDescription("Current number of open database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}

	dm.connectionsIdle, err = meter.Int64ObservableGauge(
		"db.connections.idle",
		metric.WithDescription("Current number of idle database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}

	dm.connectionsInUse, err = meter.Int64ObservableGauge(
		"db.connections.in_use",
		metric.WithDescription("Current number of in-use database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}

	dm.maxOpenConnections, err = meter.Int64ObservableGauge(
		"db.connections.max_open",
		metric.WithDescription("Maximum number of open connections allowed"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, err
	}

	// Connection wait time with custom buckets for p95, p99 accuracy
	// Buckets: 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s
	dm.connectionsWaitTime, err = meter.Float64Histogram(
		"db.connections.wait_duration",
		metric.WithDescription("Time spent waiting for a database connection"),
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
		),
	)
	if err != nil {
		return nil, err
	}

	// Query duration with custom buckets optimized for database operations
	// Buckets: 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
	dm.queryDuration, err = meter.Float64Histogram(
		"db.query.duration",
		metric.WithDescription("Database query duration"),
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

	dm.queryErrors, err = meter.Int64Counter(
		"db.query.errors",
		metric.WithDescription("Database query errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	return dm, nil
}

func (dm *DatabaseMetrics) RegisterDB(db *sql.DB, meter metric.Meter) error {
	dm.db = db

	// Register callback for connection pool stats
	// Note: service_name label is automatically added from resource attributes (service.name)
	_, err := meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			if dm.db == nil {
				return nil
			}

			stats := dm.db.Stats()

			observer.ObserveInt64(dm.connectionsOpen, int64(stats.OpenConnections))
			observer.ObserveInt64(dm.connectionsIdle, int64(stats.Idle))
			observer.ObserveInt64(dm.connectionsInUse, int64(stats.InUse))
			observer.ObserveInt64(dm.maxOpenConnections, int64(stats.MaxOpenConnections))

			return nil
		},
		dm.connectionsOpen,
		dm.connectionsIdle,
		dm.connectionsInUse,
		dm.maxOpenConnections,
	)

	return err
}

func (dm *DatabaseMetrics) RecordWaitTime(duration time.Duration) {
	dm.connectionsWaitTime.Record(context.Background(), duration.Seconds())
}

func (dm *DatabaseMetrics) RecordQuery(ctx context.Context, operation string, table string, duration time.Duration, err error) {
	if dm == nil || dm.queryDuration == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
		attribute.String("table", table),
	}

	dm.queryDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil && dm.queryErrors != nil {
		errAttrs := append(attrs, attribute.String("error", err.Error()))
		dm.queryErrors.Add(ctx, 1, metric.WithAttributes(errAttrs...))
	}
}
