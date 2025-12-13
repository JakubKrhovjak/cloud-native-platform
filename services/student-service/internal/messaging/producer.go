package messaging

import (
	"encoding/json"
	"log/slog"

	"github.com/nats-io/nats.go"
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

func (p *Producer) SendMessage(value interface{}) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		p.logger.Error("failed to marshal message", "error", err)
		return err
	}

	if err := p.conn.Publish(p.subject, valueBytes); err != nil {
		p.logger.Error("failed to send message to NATS", "error", err)
		return err
	}

	p.logger.Info("message sent to NATS", "subject", p.subject)
	return nil
}

func (p *Producer) Close() error {
	p.conn.Close()
	return nil
}
