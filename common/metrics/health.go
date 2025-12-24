package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type HealthMetrics struct {
	dependencyUp           metric.Int64ObservableGauge
	dependencyResponseTime metric.Float64Histogram
	serviceInfo            metric.Int64ObservableGauge
	dependencies           map[string]*DependencyStatus
}

type DependencyStatus struct {
	Name      string
	Available bool
}

func NewHealthMetrics(meter metric.Meter) (*HealthMetrics, error) {
	hm := &HealthMetrics{
		dependencies: make(map[string]*DependencyStatus),
	}

	var err error

	// Dependency up status (1 = up, 0 = down)
	hm.dependencyUp, err = meter.Int64ObservableGauge(
		"dependency.up",
		metric.WithDescription("Dependency availability status (1=up, 0=down)"),
		metric.WithUnit("{status}"),
	)
	if err != nil {
		return nil, err
	}

	// Dependency response time with custom buckets for p95, p99
	// Buckets: 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
	hm.dependencyResponseTime, err = meter.Float64Histogram(
		"dependency.response_time",
		metric.WithDescription("Dependency health check response time"),
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

	// Service info (always 1, carries metadata as labels)
	hm.serviceInfo, err = meter.Int64ObservableGauge(
		"service.info",
		metric.WithDescription("Service metadata information"),
		metric.WithUnit("{info}"),
	)
	if err != nil {
		return nil, err
	}

	return hm, nil
}

// Service info for Prometheus
func (hm *HealthMetrics) RegisterServiceInfo(ctx context.Context, meter metric.Meter, serviceName, version, env string) error {
	_, err := meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			attrs := []attribute.KeyValue{
				attribute.String("service_name", serviceName),
				attribute.String("version", version),
				attribute.String("environment", env),
			}
			observer.ObserveInt64(hm.serviceInfo, 1, metric.WithAttributes(attrs...))
			return nil
		},
		hm.serviceInfo,
	)
	return err
}

func (hm *HealthMetrics) RegisterDependencies(ctx context.Context, meter metric.Meter, dependencies []string) error {
	for _, dep := range dependencies {
		hm.dependencies[dep] = &DependencyStatus{
			Name:      dep,
			Available: false,
		}
	}

	_, err := meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			for _, dep := range hm.dependencies {
				attrs := []attribute.KeyValue{
					attribute.String("dependency", dep.Name),
				}

				value := int64(0)
				if dep.Available {
					value = 1
				}

				observer.ObserveInt64(hm.dependencyUp, value, metric.WithAttributes(attrs...))
			}
			return nil
		},
		hm.dependencyUp,
	)

	return err
}

func (hm *HealthMetrics) UpdateDependencyStatus(name string, available bool) {
	if dep, ok := hm.dependencies[name]; ok {
		dep.Available = available
	}
}

func (hm *HealthMetrics) RecordDependencyCheck(ctx context.Context, dependency string, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("dependency", dependency),
	}

	hm.dependencyResponseTime.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if err != nil {
		hm.UpdateDependencyStatus(dependency, false)
	} else {
		hm.UpdateDependencyStatus(dependency, true)
	}
}
