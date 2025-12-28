package messaging

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type Producer struct {
	conn    *nats.Conn
	subject string
	logger  *slog.Logger
}

func NewProducer(url string, subject string, logger *slog.Logger) (*Producer, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	logger.Info("NATS producer initialized", "url", url, "subject", subject)

	return &Producer{
		conn:    nc,
		subject: subject,
		logger:  logger,
	}, nil
}

func (p *Producer) SendMessage(ctx context.Context, value interface{}) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to marshal message", "error", err)
		return err
	}

	// Create NATS message with headers for trace propagation
	msg := nats.NewMsg(p.subject)
	msg.Data = valueBytes

	// Inject trace context into NATS headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(msg.Header))

	if err := p.conn.PublishMsg(msg); err != nil {
		p.logger.ErrorContext(ctx, "failed to send message to NATS", "error", err)
		return err
	}

	p.logger.InfoContext(ctx, "message sent to NATS", "subject", p.subject)
	return nil
}

func (p *Producer) Close() error {
	p.conn.Close()
	return nil
}

// HealthCheck verifies NATS connection is healthy
func (p *Producer) HealthCheck() error {
	if p.conn == nil {
		return nats.ErrConnectionClosed
	}

	if !p.conn.IsConnected() {
		return nats.ErrDisconnected
	}

	return nil
}
