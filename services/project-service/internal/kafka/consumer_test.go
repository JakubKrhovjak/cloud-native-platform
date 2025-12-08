package kafka_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"project-service/internal/kafka"
	"project-service/internal/message"

	"grud/testing/testdb"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumer(t *testing.T) {
	// Setup PostgreSQL container
	pgContainer := testdb.SetupSharedPostgres(t)
	defer pgContainer.Cleanup(t)

	// Run migrations
	pgContainer.RunMigrations(t, (*message.Message)(nil))

	// Create repository
	repo := message.NewRepository(pgContainer.DB)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	handler := &kafka.ConsumerGroupHandler{
		Repository: repo,
		Logger:     logger,
	}

	t.Run("ConsumeClaim_Success_SavesToDB", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "messages")

		session := newMockSession()

		// Create mock claim with one message
		messageEvent := message.MessageEvent{
			Email:   "test@example.com",
			Message: "Hello from Kafka!",
		}
		messageBytes, _ := json.Marshal(messageEvent)

		claim := &mockConsumerGroupClaim{
			messages: []*sarama.ConsumerMessage{
				{
					Topic:     "test-topic",
					Partition: 0,
					Offset:    0,
					Key:       []byte("test@example.com"),
					Value:     messageBytes,
					Timestamp: time.Now(),
				},
			},
		}

		// Consume message
		err := handler.ConsumeClaim(session, claim)
		require.NoError(t, err)

		// Verify message was marked
		assert.True(t, session.MarkedMessages["0:0"])

		// Verify message was saved to database
		ctx := context.Background()
		var messages []message.Message
		err = pgContainer.DB.NewSelect().
			Model(&messages).
			Order("id ASC").
			Scan(ctx)
		require.NoError(t, err)

		require.Len(t, messages, 1)
		assert.Equal(t, "test@example.com", messages[0].Email)
		assert.Equal(t, "Hello from Kafka!", messages[0].Message)
		assert.NotZero(t, messages[0].ID)
		assert.NotZero(t, messages[0].CreatedAt)
	})

	t.Run("ConsumeClaim_MultipleMessages", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "messages")

		session := newMockSession()

		// Create multiple messages
		msg1 := message.MessageEvent{Email: "user1@example.com", Message: "Message 1"}
		msg2 := message.MessageEvent{Email: "user2@example.com", Message: "Message 2"}
		msg3 := message.MessageEvent{Email: "user3@example.com", Message: "Message 3"}

		msg1Bytes, _ := json.Marshal(msg1)
		msg2Bytes, _ := json.Marshal(msg2)
		msg3Bytes, _ := json.Marshal(msg3)

		claim := &mockConsumerGroupClaim{
			messages: []*sarama.ConsumerMessage{
				{Topic: "test-topic", Partition: 0, Offset: 0, Value: msg1Bytes},
				{Topic: "test-topic", Partition: 0, Offset: 1, Value: msg2Bytes},
				{Topic: "test-topic", Partition: 0, Offset: 2, Value: msg3Bytes},
			},
		}

		// Consume messages
		err := handler.ConsumeClaim(session, claim)
		require.NoError(t, err)

		// Verify all messages were marked
		assert.Len(t, session.MarkedMessages, 3)

		// Verify all messages were saved to database
		ctx := context.Background()
		var messages []message.Message
		err = pgContainer.DB.NewSelect().
			Model(&messages).
			Order("id ASC").
			Scan(ctx)
		require.NoError(t, err)

		require.Len(t, messages, 3)
		assert.Equal(t, "user1@example.com", messages[0].Email)
		assert.Equal(t, "Message 1", messages[0].Message)
		assert.Equal(t, "user2@example.com", messages[1].Email)
		assert.Equal(t, "Message 2", messages[1].Message)
		assert.Equal(t, "user3@example.com", messages[2].Email)
		assert.Equal(t, "Message 3", messages[2].Message)
	})

	t.Run("ConsumeClaim_InvalidJSON_MarksAndContinues", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "messages")

		session := newMockSession()

		// Invalid JSON + Valid message
		validMsg := message.MessageEvent{Email: "valid@example.com", Message: "Valid message"}
		validMsgBytes, _ := json.Marshal(validMsg)

		claim := &mockConsumerGroupClaim{
			messages: []*sarama.ConsumerMessage{
				{Topic: "test-topic", Partition: 0, Offset: 0, Value: []byte("invalid json")},
				{Topic: "test-topic", Partition: 0, Offset: 1, Value: validMsgBytes},
			},
		}

		// Consume messages
		err := handler.ConsumeClaim(session, claim)
		require.NoError(t, err)

		// Both should be marked (invalid one marked to avoid reprocessing)
		assert.Len(t, session.MarkedMessages, 2)

		// Only valid message should be in DB
		ctx := context.Background()
		var messages []message.Message
		err = pgContainer.DB.NewSelect().
			Model(&messages).
			Order("id ASC").
			Scan(ctx)
		require.NoError(t, err)

		require.Len(t, messages, 1)
		assert.Equal(t, "valid@example.com", messages[0].Email)
		assert.Equal(t, "Valid message", messages[0].Message)
	})

	t.Run("ConsumeClaim_EmptyMessages", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "messages")

		session := newMockSession()

		// Empty claim
		claim := &mockConsumerGroupClaim{
			messages: []*sarama.ConsumerMessage{},
		}

		// Should not error
		err := handler.ConsumeClaim(session, claim)
		require.NoError(t, err)

		// No messages marked
		assert.Len(t, session.MarkedMessages, 0)

		// No messages in DB
		ctx := context.Background()
		count, err := pgContainer.DB.NewSelect().
			Model((*message.Message)(nil)).
			Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// Mock implementations

func newMockSession() *mockConsumerGroupSession {
	return &mockConsumerGroupSession{
		MarkedMessages: make(map[string]bool),
	}
}

type mockConsumerGroupSession struct {
	MarkedMessages map[string]bool
}

func (m *mockConsumerGroupSession) Claims() map[string][]int32 {
	return nil
}

func (m *mockConsumerGroupSession) MemberID() string {
	return "test-member"
}

func (m *mockConsumerGroupSession) GenerationID() int32 {
	return 1
}

func (m *mockConsumerGroupSession) MarkOffset(_ string, _ int32, _ int64, _ string) {
	// Not used
}

func (m *mockConsumerGroupSession) ResetOffset(_ string, _ int32, _ int64, _ string) {
	// Not used
}

func (m *mockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, _ string) {
	key := fmt.Sprintf("%d:%d", msg.Partition, msg.Offset)
	m.MarkedMessages[key] = true
}

func (m *mockConsumerGroupSession) Commit() {
	// Not used in our tests
}

func (m *mockConsumerGroupSession) Context() context.Context {
	return context.Background()
}

type mockConsumerGroupClaim struct {
	messages []*sarama.ConsumerMessage
	index    int
}

func (m *mockConsumerGroupClaim) Topic() string {
	return "test-topic"
}

func (m *mockConsumerGroupClaim) Partition() int32 {
	return 0
}

func (m *mockConsumerGroupClaim) InitialOffset() int64 {
	return 0
}

func (m *mockConsumerGroupClaim) HighWaterMarkOffset() int64 {
	return int64(len(m.messages))
}

func (m *mockConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	ch := make(chan *sarama.ConsumerMessage)
	go func() {
		defer close(ch)
		for _, msg := range m.messages {
			ch <- msg
		}
	}()
	return ch
}
