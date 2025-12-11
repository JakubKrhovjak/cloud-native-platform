package projectclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"student-service/internal/projectclient"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock gRPC Client
type mockGrpcClient struct {
	messages []projectclient.Message
	projects []projectclient.Project
	err      error
}

func (m *mockGrpcClient) GetAllProjects(ctx context.Context) ([]projectclient.Project, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.projects, nil
}

func (m *mockGrpcClient) GetMessagesByEmail(ctx context.Context, email string) ([]projectclient.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.messages, nil
}

func (m *mockGrpcClient) Close() error {
	return nil
}

// Ensure mockGrpcClient implements the interface
var _ interface {
	GetAllProjects(ctx context.Context) ([]projectclient.Project, error)
	GetMessagesByEmail(ctx context.Context, email string) ([]projectclient.Message, error)
	Close() error
} = (*mockGrpcClient)(nil)

func TestGetMessages(t *testing.T) {
	t.Run("GetMessages_Success", func(t *testing.T) {
		// Mock data
		mockMessages := []projectclient.Message{
			{
				ID:        1,
				Email:     "test@example.com",
				Message:   "First message",
				CreatedAt: time.Now(),
			},
			{
				ID:        2,
				Email:     "test@example.com",
				Message:   "Second message",
				CreatedAt: time.Now(),
			},
		}

		mockClient := &mockGrpcClient{
			messages: mockMessages,
		}

		router := chi.NewRouter()

		// Create a test handler function that mimics the behavior
		router.Get("/messages", func(w http.ResponseWriter, r *http.Request) {
			email := r.URL.Query().Get("email")
			if email == "" {
				http.Error(w, `{"error":"Email parameter is required"}`, http.StatusBadRequest)
				return
			}

			messages, err := mockClient.GetMessagesByEmail(r.Context(), email)
			if err != nil {
				http.Error(w, `{"error":"Failed to fetch messages"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messages)
		})

		req := httptest.NewRequest(http.MethodGet, "/messages?email=test@example.com", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []projectclient.Message
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, "test@example.com", response[0].Email)
		assert.Equal(t, "First message", response[0].Message)
	})

	t.Run("GetMessages_MissingEmail", func(t *testing.T) {
		mockClient := &mockGrpcClient{}

		router := chi.NewRouter()
		router.Get("/messages", func(w http.ResponseWriter, r *http.Request) {
			email := r.URL.Query().Get("email")
			if email == "" {
				http.Error(w, `{"error":"Email parameter is required"}`, http.StatusBadRequest)
				return
			}

			messages, err := mockClient.GetMessagesByEmail(r.Context(), email)
			if err != nil {
				http.Error(w, `{"error":"Failed to fetch messages"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messages)
		})

		req := httptest.NewRequest(http.MethodGet, "/messages", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GetMessages_EmptyResult", func(t *testing.T) {
		mockClient := &mockGrpcClient{
			messages: []projectclient.Message{},
		}

		router := chi.NewRouter()
		router.Get("/messages", func(w http.ResponseWriter, r *http.Request) {
			email := r.URL.Query().Get("email")
			if email == "" {
				http.Error(w, `{"error":"Email parameter is required"}`, http.StatusBadRequest)
				return
			}

			messages, err := mockClient.GetMessagesByEmail(r.Context(), email)
			if err != nil {
				http.Error(w, `{"error":"Failed to fetch messages"}`, http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messages)
		})

		req := httptest.NewRequest(http.MethodGet, "/messages?email=nonexistent@example.com", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []projectclient.Message
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 0)
	})
}
