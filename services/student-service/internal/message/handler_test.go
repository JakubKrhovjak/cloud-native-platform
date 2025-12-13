package message_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"student-service/internal/auth"
	"student-service/internal/message"
	"student-service/internal/messaging"

	"grud/testing/testnats"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestWithNATSContainer(t *testing.T, natsContainer *testnats.NATSContainer, subject string) (chi.Router, *nats.Conn) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	producer, err := messaging.NewProducer(natsContainer.URL, subject, logger)
	require.NoError(t, err)

	service := message.NewService(producer, logger)
	handler := message.NewHandler(service, logger)

	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	nc := natsContainer.Connect(t)

	return router, nc
}

func TestMessageHandlerWithNATSContainer(t *testing.T) {
	natsContainer := testnats.SetupSharedNATS(t)
	defer natsContainer.Cleanup(t)

	t.Run("SendMessage_Success", func(t *testing.T) {
		subject := "test.messages." + strings.ReplaceAll(t.Name(), "/", ".")
		router, nc := setupTestWithNATSContainer(t, natsContainer, subject)

		// Subscribe to verify message was sent
		received := make(chan *nats.Msg, 1)
		_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
			received <- msg
		})
		require.NoError(t, err)

		// Create request
		payload := message.SendMessageRequest{
			Message: "Hello from NATS testcontainer!",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), auth.EmailKey, "test@example.com")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assertions
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "success", response["status"])

		// Verify message was published to NATS
		select {
		case msg := <-received:
			var event message.MessageEvent
			err = json.Unmarshal(msg.Data, &event)
			require.NoError(t, err)
			assert.Equal(t, "test@example.com", event.Email)
			assert.Equal(t, "Hello from NATS testcontainer!", event.Message)
		case <-time.After(2 * time.Second):
			t.Fatal("Message not received on NATS within timeout")
		}
	})

	t.Run("SendMessage_MultipleMessages", func(t *testing.T) {
		subject := "test.messages." + strings.ReplaceAll(t.Name(), "/", ".")
		router, nc := setupTestWithNATSContainer(t, natsContainer, subject)

		received := make(chan *nats.Msg, 3)
		_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
			received <- msg
		})
		require.NoError(t, err)

		// Ensure subscriber is ready
		require.NoError(t, nc.Flush())
		time.Sleep(50 * time.Millisecond)

		emails := []string{"user1@example.com", "user2@example.com", "user3@example.com"}

		for i, email := range emails {
			payload := message.SendMessageRequest{
				Message: "Message " + string(rune(i)),
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			ctx := context.WithValue(req.Context(), auth.EmailKey, email)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}

		// Verify all messages received
		for i := 0; i < 3; i++ {
			select {
			case msg := <-received:
				var event message.MessageEvent
				err = json.Unmarshal(msg.Data, &event)
				require.NoError(t, err)
				assert.Contains(t, emails, event.Email)
			case <-time.After(2 * time.Second):
				t.Fatalf("Message %d not received within timeout", i)
			}
		}
	})

	t.Run("SendMessage_Unauthorized", func(t *testing.T) {
		subject := "test.messages." + strings.ReplaceAll(t.Name(), "/", ".")
		router, _ := setupTestWithNATSContainer(t, natsContainer, subject)

		payload := message.SendMessageRequest{
			Message: "This should fail",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("SendMessage_EmptyMessage", func(t *testing.T) {
		subject := "test.messages." + strings.ReplaceAll(t.Name(), "/", ".")
		router, _ := setupTestWithNATSContainer(t, natsContainer, subject)

		payload := message.SendMessageRequest{
			Message: "",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		ctx := context.WithValue(req.Context(), auth.EmailKey, "test@example.com")
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
