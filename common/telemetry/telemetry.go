package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"grud/common/metrics"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func InitMeterProvider(ctx context.Context, serviceName, serviceVersion string, logger *slog.Logger) (*metric.MeterProvider, error) {
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "otel-collector.infra.svc.cluster.local:4317"
	}

	logger.Info("initializing OTel metrics", "endpoint", otelEndpoint)

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otelEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(10*time.Second))),
	)

	otel.SetMeterProvider(meterProvider)
	logger.Info("OTel metrics initialized successfully")

	return meterProvider, nil
}

type Telemetry struct {
	MeterProvider *metric.MeterProvider
	Metrics       *metrics.Metrics
}

func Init(ctx context.Context, serviceName, serviceVersion, env string, logger *slog.Logger) (*Telemetry, error) {
	meterProvider, err := InitMeterProvider(ctx, serviceName, serviceVersion, logger)
	if err != nil {
		return nil, err
	}

	m, err := metrics.New(ctx, serviceName, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	meter := otel.Meter(serviceName)
	if err := m.Health.RegisterServiceInfo(ctx, meter, serviceName, serviceVersion, env); err != nil {
		logger.Warn("failed to register service info", "error", err)
	}

	return &Telemetry{
		MeterProvider: meterProvider,
		Metrics:       m,
	}, nil
}

func Shutdown(ctx context.Context, meterProvider *metric.MeterProvider, logger *slog.Logger) error {
	logger.Info("shutting down OTel meter provider")
	if err := meterProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown meter provider: %w", err)
	}
	return nil
}
