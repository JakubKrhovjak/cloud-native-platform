package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"project-service/internal/message"

	"github.com/IBM/sarama"
)

type Consumer struct {
	consumer   sarama.ConsumerGroup
	topic      string
	repository message.Repository
	logger     *slog.Logger
}

func NewConsumer(brokers []string, topic string, repository message.Repository, logger *slog.Logger) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	consumerGroup, err := sarama.NewConsumerGroup(brokers, "project-service-group", config)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		consumer:   consumerGroup,
		topic:      topic,
		repository: repository,
		logger:     logger,
	}, nil
}

func (c *Consumer) Start(ctx context.Context) error {
	handler := &consumerGroupHandler{
		repository: c.repository,
		logger:     c.logger,
	}

	for {
		if err := c.consumer.Consume(ctx, []string{c.topic}, handler); err != nil {
			c.logger.Error("error consuming messages", "error", err)
			return err
		}

		// Check if context was cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (c *Consumer) Close() error {
	return c.consumer.Close()
}

type consumerGroupHandler struct {
	repository message.Repository
	logger     *slog.Logger
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.logger.Info("received message from Kafka",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
		)

		var event message.MessageEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			h.logger.Error("failed to unmarshal message", "error", err)
			session.MarkMessage(msg, "")
			continue
		}

		// Save message to database
		dbMessage := &message.Message{
			Email:   event.Email,
			Message: event.Message,
		}

		if err := h.repository.Create(context.Background(), dbMessage); err != nil {
			h.logger.Error("failed to save message to database", "error", err)
			// Still mark as consumed to avoid reprocessing
			session.MarkMessage(msg, "")
			continue
		}

		h.logger.Info("message saved to database",
			"email", event.Email,
			"message", event.Message,
			"id", dbMessage.ID,
		)

		session.MarkMessage(msg, "")
	}

	return nil
}
