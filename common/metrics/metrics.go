package metrics

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
)

type Metrics struct {
	Runtime   *RuntimeMetrics
	Database  *DatabaseMetrics
	Messaging *MessagingMetrics
	Health    *HealthMetrics
	logger    *slog.Logger
}

func New(ctx context.Context, serviceName string, logger *slog.Logger) (*Metrics, error) {
	meter := otel.Meter(serviceName)

	runtime, err := NewRuntimeMetrics(ctx, meter)
	if err != nil {
		return nil, err
	}

	database, err := NewDatabaseMetrics(meter)
	if err != nil {
		return nil, err
	}

	messaging, err := NewMessagingMetrics(meter)
	if err != nil {
		return nil, err
	}

	health, err := NewHealthMetrics(meter)
	if err != nil {
		return nil, err
	}

	logger.Info("metrics collectors initialized successfully")

	return &Metrics{
		Runtime:   runtime,
		Database:  database,
		Messaging: messaging,
		Health:    health,
		logger:    logger,
	}, nil
}

// NewMock creates a no-op Metrics instance for testing
// The returned Metrics will safely ignore all Record* calls
func NewMock() *Metrics {
	return &Metrics{
		Database:  &DatabaseMetrics{},
		Messaging: &MessagingMetrics{},
		Health:    &HealthMetrics{},
		Runtime:   &RuntimeMetrics{},
	}
}
