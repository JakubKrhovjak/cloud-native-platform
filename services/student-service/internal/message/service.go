package message

import (
	"log/slog"
	"student-service/internal/kafka"
)

// Producer interface for Kafka producer (for testing)
type Producer interface {
	SendMessage(key string, value interface{}) error
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

// NewServiceWithKafka creates a service with kafka.Producer
func NewServiceWithKafka(producer *kafka.Producer, logger *slog.Logger) *Service {
	return NewService(producer, logger)
}

func (s *Service) SendMessage(email string, message string) error {
	event := MessageEvent{
		Email:   email,
		Message: message,
	}

	s.logger.Info("sending message to kafka", "email", email)

	if err := s.producer.SendMessage(email, event); err != nil {
		s.logger.Error("failed to send message", "error", err)
		return err
	}

	return nil
}
