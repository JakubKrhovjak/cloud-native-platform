package message

import (
	"log/slog"
)

// Producer interface for messaging (NATS/Kafka)
type Producer interface {
	SendMessage(value interface{}) error
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

func (s *Service) SendMessage(email string, message string) error {
	event := MessageEvent{
		Email:   email,
		Message: message,
	}

	s.logger.Info("sending message to NATS", "email", email)

	if err := s.producer.SendMessage(event); err != nil {
		s.logger.Error("failed to send message", "error", err)
		return err
	}

	return nil
}
