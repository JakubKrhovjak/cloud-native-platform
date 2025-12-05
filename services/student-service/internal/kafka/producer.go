package kafka

import (
	"encoding/json"
	"log/slog"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
	logger   *slog.Logger
}

func NewProducer(brokers []string, topic string, logger *slog.Logger) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	logger.Info("kafka producer initialized", "brokers", brokers, "topic", topic)

	return &Producer{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}, nil
}

func (p *Producer) SendMessage(key string, value interface{}) error {
	// Marshal value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		p.logger.Error("failed to marshal message", "error", err)
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(valueBytes),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error("failed to send message to kafka", "error", err)
		return err
	}

	p.logger.Info("message sent to kafka", "topic", p.topic, "partition", partition, "offset", offset, "key", key)
	return nil
}

func (p *Producer) Close() error {
	return p.producer.Close()
}
