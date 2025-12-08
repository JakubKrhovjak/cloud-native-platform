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
	handler := &ConsumerGroupHandler{
		Repository: c.repository,
		Logger:     c.logger,
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

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler interface
type ConsumerGroupHandler struct {
	Repository message.Repository
	Logger     *slog.Logger
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.Logger.Info("received message from Kafka",
			"topic", msg.Topic,
			"partition", msg.Partition,
			"offset", msg.Offset,
		)

		var event message.MessageEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			h.Logger.Error("failed to unmarshal message", "error", err)
			session.MarkMessage(msg, "")
			continue
		}

		// Save message to database
		dbMessage := &message.Message{
			Email:   event.Email,
			Message: event.Message,
		}

		if err := h.Repository.Create(context.Background(), dbMessage); err != nil {
			h.Logger.Error("failed to save message to database", "error", err)
			// Still mark as consumed to avoid reprocessing
			session.MarkMessage(msg, "")
			continue
		}

		h.Logger.Info("message saved to database",
			"email", event.Email,
			"message", event.Message,
			"id", dbMessage.ID,
		)

		session.MarkMessage(msg, "")
	}

	return nil
}
