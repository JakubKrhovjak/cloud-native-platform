package messaging

import (
	"context"
	"encoding/json"
	"log/slog"

	"project-service/internal/message"
	"project-service/internal/metrics"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type Consumer struct {
	conn       *nats.Conn
	sub        *nats.Subscription
	subject    string
	repository message.Repository
	logger     *slog.Logger
	metrics    *metrics.Metrics
}

func NewConsumer(url string, subject string, repository message.Repository, logger *slog.Logger, metrics *metrics.Metrics) (*Consumer, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		conn:       nc,
		subject:    subject,
		repository: repository,
		logger:     logger,
		metrics:    metrics,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	sub, err := c.conn.Subscribe(c.subject, func(msg *nats.Msg) {
		// Extract trace context from NATS headers
		msgCtx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.HeaderCarrier(msg.Header))

		c.logger.InfoContext(msgCtx, "received message from NATS", "subject", msg.Subject)

		var event message.MessageEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			c.logger.ErrorContext(msgCtx, "failed to unmarshal message", "error", err)
			return
		}

		dbMessage := &message.Message{
			Email:   event.Email,
			Message: event.Message,
		}

		if err := c.repository.Create(msgCtx, dbMessage); err != nil {
			c.logger.ErrorContext(msgCtx, "failed to save message to database", "error", err)
			return
		}

		// Record metric
		c.metrics.RecordMessageReceived(msgCtx)

		c.logger.InfoContext(msgCtx, "message saved to database",
			"email", event.Email,
			"message", event.Message,
			"id", dbMessage.ID,
		)
	})

	if err != nil {
		return err
	}

	c.sub = sub
	c.logger.Info("NATS consumer started", "subject", c.subject)

	<-ctx.Done()
	return ctx.Err()
}

func (c *Consumer) Close() error {
	if c.sub != nil {
		c.sub.Unsubscribe()
	}
	c.conn.Close()
	return nil
}

// HealthCheck verifies NATS connection is healthy
func (c *Consumer) HealthCheck() error {
	if c.conn == nil {
		return nats.ErrConnectionClosed
	}

	if !c.conn.IsConnected() {
		return nats.ErrDisconnected
	}

	return nil
}
