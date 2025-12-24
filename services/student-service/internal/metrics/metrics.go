package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	studentsRegistered          metric.Int64Counter
	messagesSent                metric.Int64Counter
	studentsViewed              metric.Int64Counter
	studentsListViewed          metric.Int64Counter
	projectsListViewedByStudent metric.Int64Counter
}

func New(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error

	m.studentsRegistered, err = meter.Int64Counter(
		"student_service.students.registered",
		metric.WithDescription("Total number of students registered"),
		metric.WithUnit("{student}"),
	)
	if err != nil {
		return nil, err
	}

	m.messagesSent, err = meter.Int64Counter(
		"student_service.messages.sent",
		metric.WithDescription("Total number of messages sent"),
		metric.WithUnit("{message}"),
	)
	if err != nil {
		return nil, err
	}

	m.studentsViewed, err = meter.Int64Counter(
		"student_service.students.viewed",
		metric.WithDescription("Total number of students viewed"),
		metric.WithUnit("{view}"),
	)
	if err != nil {
		return nil, err
	}

	m.studentsListViewed, err = meter.Int64Counter(
		"student_service.students.list_viewed",
		metric.WithDescription("Total number of times students list was viewed"),
		metric.WithUnit("{view}"),
	)
	if err != nil {
		return nil, err
	}

	m.projectsListViewedByStudent, err = meter.Int64Counter(
		"student_service.projects.list_viewed_by_student",
		metric.WithDescription("Total number of times students viewed projects list"),
		metric.WithUnit("{view}"),
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Metrics) RecordStudentRegistration(ctx context.Context) {
	if m != nil && m.studentsRegistered != nil {
		m.studentsRegistered.Add(ctx, 1)
	}
}

func (m *Metrics) RecordMessageSent(ctx context.Context) {
	if m != nil && m.messagesSent != nil {
		m.messagesSent.Add(ctx, 1)
	}
}

func (m *Metrics) RecordStudentViewed(ctx context.Context) {
	if m != nil && m.studentsViewed != nil {
		m.studentsViewed.Add(ctx, 1)
	}
}

func (m *Metrics) RecordStudentsListViewed(ctx context.Context) {
	if m != nil && m.studentsListViewed != nil {
		m.studentsListViewed.Add(ctx, 1)
	}
}

func (m *Metrics) RecordProjectsListViewedByStudent(ctx context.Context) {
	if m != nil && m.projectsListViewedByStudent != nil {
		m.projectsListViewedByStudent.Add(ctx, 1)
	}
}

// NewMock creates a no-op Metrics instance for testing
// The returned Metrics will safely ignore all Record* calls
func NewMock() *Metrics {
	return &Metrics{}
}
