package messaging_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"project-service/internal/message"
	"project-service/internal/messaging"

	"grud/testing/testnats"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMessageRepository for testing
type MockMessageRepository struct {
	messages []*message.Message
}

func (m *MockMessageRepository) Create(ctx context.Context, msg *message.Message) error {
	msg.ID = len(m.messages) + 1
	m.messages = append(m.messages, msg)
	return nil
}

func (m *MockMessageRepository) GetByEmail(ctx context.Context, email string) ([]*message.Message, error) {
	var results []*message.Message
	for _, msg := range m.messages {
		if msg.Email == email {
			results = append(results, msg)
		}
	}
	return results, nil
}

func TestNATSConsumerIntegration(t *testing.T) {
	natsContainer := testnats.SetupSharedNATS(t)
	defer natsContainer.Cleanup(t)

	t.Run("Consumer_ReceivesAndStoresMessage", func(t *testing.T) {
		natsURL := natsContainer.URL

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

		subject := "test.messages." + strings.ReplaceAll(t.Name(), "/", ".")
		repo := &MockMessageRepository{messages: make([]*message.Message, 0)}

		// Create consumer
		consumer, err := messaging.NewConsumer(natsURL, subject, repo, logger)
		require.NoError(t, err)
		defer consumer.Close()

		// Start consumer in background
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = consumer.Start(ctx)
		}()

		// Wait for consumer to be ready
		time.Sleep(100 * time.Millisecond)

		// Publish a message
		nc, err := nats.Connect(natsURL)
		require.NoError(t, err)
		defer nc.Close()

		event := message.MessageEvent{
			Email:   "user@example.com",
			Message: "Test message from integration test",
		}
		data, err := json.Marshal(event)
		require.NoError(t, err)

		err = nc.Publish(subject, data)
		require.NoError(t, err)

		// Wait for message to be processed
		time.Sleep(200 * time.Millisecond)

		// Verify message was stored
		messages, err := repo.GetByEmail(context.Background(), "user@example.com")
		require.NoError(t, err)
		require.Len(t, messages, 1)
		assert.Equal(t, "user@example.com", messages[0].Email)
		assert.Equal(t, "Test message from integration test", messages[0].Message)
	})

	t.Run("Consumer_MultipleMessages", func(t *testing.T) {
		natsURL := natsContainer.URL

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

		subject := "test.messages." + strings.ReplaceAll(t.Name(), "/", ".")
		repo := &MockMessageRepository{messages: make([]*message.Message, 0)}

		consumer, err := messaging.NewConsumer(natsURL, subject, repo, logger)
		require.NoError(t, err)
		defer consumer.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = consumer.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		// Publish multiple messages
		nc, err := nats.Connect(natsURL)
		require.NoError(t, err)
		defer nc.Close()

		for i := 0; i < 5; i++ {
			event := message.MessageEvent{
				Email:   "user@example.com",
				Message: "Message " + string(rune(i)),
			}
			data, _ := json.Marshal(event)
			nc.Publish(subject, data)
		}

		// Wait for all messages to be processed
		time.Sleep(300 * time.Millisecond)

		messages, err := repo.GetByEmail(context.Background(), "user@example.com")
		require.NoError(t, err)
		assert.Len(t, messages, 5)
	})

	t.Run("Consumer_InvalidJSON", func(t *testing.T) {
		natsURL := natsContainer.URL

		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

		subject := "test.messages." + strings.ReplaceAll(t.Name(), "/", ".")
		repo := &MockMessageRepository{messages: make([]*message.Message, 0)}

		consumer, err := messaging.NewConsumer(natsURL, subject, repo, logger)
		require.NoError(t, err)
		defer consumer.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			_ = consumer.Start(ctx)
		}()

		time.Sleep(100 * time.Millisecond)

		// Publish invalid JSON
		nc, err := nats.Connect(natsURL)
		require.NoError(t, err)
		defer nc.Close()

		err = nc.Publish(subject, []byte("invalid json"))
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// No messages should be stored
		messages, err := repo.GetByEmail(context.Background(), "user@example.com")
		require.NoError(t, err)
		assert.Len(t, messages, 0)
	})
}
