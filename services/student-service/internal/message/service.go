package message

import (
	"context"
	"log/slog"
)

// Producer interface for messaging (NATS/Kafka)
type Producer interface {
	SendMessage(ctx context.Context, value interface{}) error
	Close() error
}

type Service struct {
	producer Producer
	logger   *slog.Logger
}

func NewService(producer Producer, logger *slog.Logger) *Service {
	return &Service{
		producer: producer,
		logger:   logger,
	}
}

func (s *Service) SendMessage(ctx context.Context, email string, message string) error {
	event := MessageEvent{
		Email:   email,
		Message: message,
	}

	s.logger.InfoContext(ctx, "sending message to NATS", "email", email)

	if err := s.producer.SendMessage(ctx, event); err != nil {
		s.logger.ErrorContext(ctx, "failed to send message", "error", err)
		return err
	}

	return nil
}
