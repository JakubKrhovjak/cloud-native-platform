package metrics

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/metric"
)

type RuntimeMetrics struct {
	goroutines      metric.Int64ObservableGauge
	heapAlloc       metric.Int64ObservableGauge
	heapInuse       metric.Int64ObservableGauge
	heapObjects     metric.Int64ObservableGauge
	stackInuse      metric.Int64ObservableGauge
	gcPauseDuration metric.Float64Histogram
	gcCount         metric.Int64ObservableCounter
	uptimeSeconds   metric.Float64ObservableCounter
	startTime       time.Time
}

func NewRuntimeMetrics(ctx context.Context, meter metric.Meter) (*RuntimeMetrics, error) {
	rm := &RuntimeMetrics{
		startTime: time.Now(),
	}

	var err error

	// Goroutines
	rm.goroutines, err = meter.Int64ObservableGauge(
		"runtime.go.goroutines",
		metric.WithDescription("Number of goroutines"),
		metric.WithUnit("{goroutine}"),
	)
	if err != nil {
		return nil, err
	}

	// Heap memory allocated
	rm.heapAlloc, err = meter.Int64ObservableGauge(
		"runtime.go.mem.heap_alloc",
		metric.WithDescription("Bytes of allocated heap objects"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	// Heap memory in use
	rm.heapInuse, err = meter.Int64ObservableGauge(
		"runtime.go.mem.heap_inuse",
		metric.WithDescription("Bytes in in-use spans"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	// Heap objects
	rm.heapObjects, err = meter.Int64ObservableGauge(
		"runtime.go.mem.heap_objects",
		metric.WithDescription("Number of allocated heap objects"),
		metric.WithUnit("{object}"),
	)
	if err != nil {
		return nil, err
	}

	// Stack memory in use
	rm.stackInuse, err = meter.Int64ObservableGauge(
		"runtime.go.mem.stack_inuse",
		metric.WithDescription("Bytes in stack spans"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, err
	}

	// GC count
	rm.gcCount, err = meter.Int64ObservableCounter(
		"runtime.go.gc.count",
		metric.WithDescription("Number of completed GC cycles"),
		metric.WithUnit("{gc}"),
	)
	if err != nil {
		return nil, err
	}

	// Service uptime
	rm.uptimeSeconds, err = meter.Float64ObservableCounter(
		"service.uptime",
		metric.WithDescription("Service uptime in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// GC pause duration histogram with custom buckets for better percentile accuracy
	// Buckets: 0.1µs, 0.5µs, 1µs, 5µs, 10µs, 50µs, 100µs, 500µs, 1ms, 5ms, 10ms, 50ms, 100ms
	rm.gcPauseDuration, err = meter.Float64Histogram(
		"runtime.go.gc.pause_duration",
		metric.WithDescription("GC pause duration in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.0000001, // 0.1µs
			0.0000005, // 0.5µs
			0.000001,  // 1µs
			0.000005,  // 5µs
			0.00001,   // 10µs
			0.00005,   // 50µs
			0.0001,    // 100µs
			0.0005,    // 500µs
			0.001,     // 1ms
			0.005,     // 5ms
			0.01,      // 10ms
			0.05,      // 50ms
			0.1,       // 100ms
		),
	)
	if err != nil {
		return nil, err
	}

	// Register callbacks for observable metrics
	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			observer.ObserveInt64(rm.goroutines, int64(runtime.NumGoroutine()))
			observer.ObserveInt64(rm.heapAlloc, int64(m.HeapAlloc))
			observer.ObserveInt64(rm.heapInuse, int64(m.HeapInuse))
			observer.ObserveInt64(rm.heapObjects, int64(m.HeapObjects))
			observer.ObserveInt64(rm.stackInuse, int64(m.StackInuse))
			observer.ObserveInt64(rm.gcCount, int64(m.NumGC))
			observer.ObserveFloat64(rm.uptimeSeconds, time.Since(rm.startTime).Seconds())

			return nil
		},
		rm.goroutines,
		rm.heapAlloc,
		rm.heapInuse,
		rm.heapObjects,
		rm.stackInuse,
		rm.gcCount,
		rm.uptimeSeconds,
	)
	if err != nil {
		return nil, err
	}

	return rm, nil
}

func (rm *RuntimeMetrics) RecordGCPause(duration time.Duration) {
	rm.gcPauseDuration.Record(context.Background(), duration.Seconds())
}
