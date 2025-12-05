package message

import (
	"context"
	"log/slog"
	"student-service/internal/kafka"
)

type Service struct {
	producer *kafka.Producer
	logger   *slog.Logger
}

func NewService(producer *kafka.Producer, logger *slog.Logger) *Service {
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

	s.logger.Info("sending message to kafka", "email", email)

	if err := s.producer.SendMessage(email, event); err != nil {
		s.logger.Error("failed to send message", "error", err)
		return err
	}

	return nil
}
