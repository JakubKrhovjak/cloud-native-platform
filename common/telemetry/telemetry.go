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
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Telemetry struct {
	MeterProvider  *metric.MeterProvider
	TracerProvider *sdktrace.TracerProvider
	Metrics        *metrics.Metrics
}

func InitMeterProvider(ctx context.Context, serviceName, serviceVersion string, logger *slog.Logger) (*metric.MeterProvider, error) {
	otelEndpoint := getOtelEndpoint()
	logger.Info("initializing OTel metrics", "endpoint", otelEndpoint)

	res, err := createResource(ctx, serviceName, serviceVersion)
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

func InitTracerProvider(ctx context.Context, serviceName, serviceVersion string, logger *slog.Logger) (*sdktrace.TracerProvider, error) {
	otelEndpoint := getOtelEndpoint()
	logger.Info("initializing OTel tracing", "endpoint", otelEndpoint)

	res, err := createResource(ctx, serviceName, serviceVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OTel tracing initialized successfully")
	return tracerProvider, nil
}

func Init(ctx context.Context, serviceName, serviceVersion, env string, logger *slog.Logger) (*Telemetry, error) {
	// Initialize tracing
	tracerProvider, err := InitTracerProvider(ctx, serviceName, serviceVersion, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize metrics
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
		MeterProvider:  meterProvider,
		TracerProvider: tracerProvider,
		Metrics:        m,
	}, nil
}

func (t *Telemetry) Shutdown(ctx context.Context, logger *slog.Logger) error {
	logger.Info("shutting down OTel providers")

	if t.TracerProvider != nil {
		if err := t.TracerProvider.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown tracer provider", "error", err)
		}
	}

	if t.MeterProvider != nil {
		if err := t.MeterProvider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown meter provider: %w", err)
		}
	}

	return nil
}

func getOtelEndpoint() string {
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "alloy.infra.svc.cluster.local:4317"
	}
	return otelEndpoint
}

func createResource(ctx context.Context, serviceName, serviceVersion string) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
}

// Shutdown is deprecated, use Telemetry.Shutdown instead
func Shutdown(ctx context.Context, meterProvider *metric.MeterProvider, logger *slog.Logger) error {
	logger.Info("shutting down OTel meter provider")
	if err := meterProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown meter provider: %w", err)
	}
	return nil
}
