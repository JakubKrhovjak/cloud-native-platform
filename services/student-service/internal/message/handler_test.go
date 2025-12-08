package message_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"student-service/internal/auth"
	"student-service/internal/message"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProducer mocks the Kafka producer for testing
type MockProducer struct {
	SendMessageFunc func(key string, value interface{}) error
	messages        []MockMessage
}

type MockMessage struct {
	Key   string
	Value interface{}
}

func (m *MockProducer) SendMessage(key string, value interface{}) error {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(key, value)
	}
	m.messages = append(m.messages, MockMessage{Key: key, Value: value})
	return nil
}

func (m *MockProducer) Close() error {
	return nil
}

func TestMessageHandler(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	t.Run("SendMessage_Success", func(t *testing.T) {
		// Setup mock producer
		mockProducer := &MockProducer{}
		service := message.NewService(mockProducer, logger)
		handler := message.NewHandler(service, logger)

		router := chi.NewRouter()
		handler.RegisterRoutes(router)

		// Create request payload
		payload := message.SendMessageRequest{
			Message: "Hello from test!",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		// Create request with auth context
		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Add email to context (simulating auth middleware)
		ctx := context.WithValue(req.Context(), auth.EmailKey, "test@example.com")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		// Execute request
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "success", response["status"])
		assert.Equal(t, "message sent successfully", response["message"])

		// Verify message was sent to Kafka
		require.Len(t, mockProducer.messages, 1)
		assert.Equal(t, "test@example.com", mockProducer.messages[0].Key)

		messageEvent, ok := mockProducer.messages[0].Value.(message.MessageEvent)
		require.True(t, ok)
		assert.Equal(t, "test@example.com", messageEvent.Email)
		assert.Equal(t, "Hello from test!", messageEvent.Message)
	})

	t.Run("SendMessage_Unauthorized_NoEmail", func(t *testing.T) {
		mockProducer := &MockProducer{}
		service := message.NewService(mockProducer, logger)
		handler := message.NewHandler(service, logger)

		router := chi.NewRouter()
		handler.RegisterRoutes(router)

		payload := message.SendMessageRequest{
			Message: "Hello from test!",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		// Create request WITHOUT email in context
		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Verify no message was sent
		assert.Len(t, mockProducer.messages, 0)
	})

	t.Run("SendMessage_InvalidJSON", func(t *testing.T) {
		mockProducer := &MockProducer{}
		service := message.NewService(mockProducer, logger)
		handler := message.NewHandler(service, logger)

		router := chi.NewRouter()
		handler.RegisterRoutes(router)

		// Invalid JSON
		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), auth.EmailKey, "test@example.com")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify no message was sent
		assert.Len(t, mockProducer.messages, 0)
	})

	t.Run("SendMessage_EmptyMessage", func(t *testing.T) {
		mockProducer := &MockProducer{}
		service := message.NewService(mockProducer, logger)
		handler := message.NewHandler(service, logger)

		router := chi.NewRouter()
		handler.RegisterRoutes(router)

		// Empty message (should fail validation)
		payload := message.SendMessageRequest{
			Message: "",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), auth.EmailKey, "test@example.com")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 400 Bad Request (validation failed)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify no message was sent
		assert.Len(t, mockProducer.messages, 0)
	})

	t.Run("SendMessage_ProducerError", func(t *testing.T) {
		// Setup mock producer that returns error
		mockProducer := &MockProducer{
			SendMessageFunc: func(key string, value interface{}) error {
				return errors.New("kafka connection failed")
			},
		}
		service := message.NewService(mockProducer, logger)
		handler := message.NewHandler(service, logger)

		router := chi.NewRouter()
		handler.RegisterRoutes(router)

		payload := message.SendMessageRequest{
			Message: "Hello from test!",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), auth.EmailKey, "test@example.com")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should return 500 Internal Server Error
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("SendMessage_MultipleMessages", func(t *testing.T) {
		mockProducer := &MockProducer{}
		service := message.NewService(mockProducer, logger)
		handler := message.NewHandler(service, logger)

		router := chi.NewRouter()
		handler.RegisterRoutes(router)

		// Send first message
		payload1 := message.SendMessageRequest{
			Message: "First message",
		}
		body1, _ := json.Marshal(payload1)

		req1 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		ctx1 := context.WithValue(req1.Context(), auth.EmailKey, "user1@example.com")
		req1 = req1.WithContext(ctx1)

		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Send second message
		payload2 := message.SendMessageRequest{
			Message: "Second message",
		}
		body2, _ := json.Marshal(payload2)

		req2 := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		ctx2 := context.WithValue(req2.Context(), auth.EmailKey, "user2@example.com")
		req2 = req2.WithContext(ctx2)

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)

		// Verify both messages were sent
		require.Len(t, mockProducer.messages, 2)

		msg1 := mockProducer.messages[0].Value.(message.MessageEvent)
		assert.Equal(t, "user1@example.com", msg1.Email)
		assert.Equal(t, "First message", msg1.Message)

		msg2 := mockProducer.messages[1].Value.(message.MessageEvent)
		assert.Equal(t, "user2@example.com", msg2.Email)
		assert.Equal(t, "Second message", msg2.Message)
	})
}
