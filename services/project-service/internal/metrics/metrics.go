package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	projectsCreated    metric.Int64Counter
	projectsViewed     metric.Int64Counter
	projectsListViewed metric.Int64Counter
	messagesReceived   metric.Int64Counter
}

func New(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.projectsCreated, err = meter.Int64Counter(
		"project_service.projects.created",
		metric.WithDescription("Total number of projects created"),
		metric.WithUnit("{project}"),
	)
	if err != nil {
		return nil, err
	}

	m.projectsViewed, err = meter.Int64Counter(
		"project_service.projects.viewed",
		metric.WithDescription("Total number of projects viewed"),
		metric.WithUnit("{view}"),
	)
	if err != nil {
		return nil, err
	}

	m.projectsListViewed, err = meter.Int64Counter(
		"project_service.projects.list_viewed",
		metric.WithDescription("Total number of times projects list was viewed"),
		metric.WithUnit("{view}"),
	)
	if err != nil {
		return nil, err
	}

	m.messagesReceived, err = meter.Int64Counter(
		"project_service.messages.received",
		metric.WithDescription("Total number of messages received from NATS"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Metrics) RecordProjectCreation(ctx context.Context) {
	if m != nil && m.projectsCreated != nil {
		m.projectsCreated.Add(ctx, 1)
	}
}

func (m *Metrics) RecordProjectViewed(ctx context.Context) {
	if m != nil && m.projectsViewed != nil {
		m.projectsViewed.Add(ctx, 1)
	}
}

func (m *Metrics) RecordProjectsListViewed(ctx context.Context) {
	if m != nil && m.projectsListViewed != nil {
		m.projectsListViewed.Add(ctx, 1)
	}
}

func (m *Metrics) RecordMessageReceived(ctx context.Context) {
	if m != nil && m.messagesReceived != nil {
		m.messagesReceived.Add(ctx, 1)
	}
}

// NewMock creates a no-op Metrics instance for testing
// The returned Metrics will safely ignore all Record* calls
func NewMock() *Metrics {
	return &Metrics{}
}
