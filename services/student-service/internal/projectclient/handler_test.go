package projectclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"student-service/internal/projectclient"

	"github.com/gin-gonic/gin"
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
	gin.SetMode(gin.TestMode)

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

		router := gin.New()

		// Create a test handler function that mimics the behavior
		router.GET("/messages", func(c *gin.Context) {
			email := c.Query("email")
			if email == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter is required"})
				return
			}

			messages, err := mockClient.GetMessagesByEmail(c.Request.Context(), email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
				return
			}

			c.JSON(http.StatusOK, messages)
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

		router := gin.New()
		router.GET("/messages", func(c *gin.Context) {
			email := c.Query("email")
			if email == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter is required"})
				return
			}

			messages, err := mockClient.GetMessagesByEmail(c.Request.Context(), email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
				return
			}

			c.JSON(http.StatusOK, messages)
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

		router := gin.New()
		router.GET("/messages", func(c *gin.Context) {
			email := c.Query("email")
			if email == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter is required"})
				return
			}

			messages, err := mockClient.GetMessagesByEmail(c.Request.Context(), email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
				return
			}

			c.JSON(http.StatusOK, messages)
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

func TestGetAllProjects(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("GetAllProjects_Success", func(t *testing.T) {
		// Mock data
		mockProjects := []projectclient.Project{
			{
				ID:        1,
				Name:      "Project One",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        2,
				Name:      "Project Two",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}

		mockClient := &mockGrpcClient{
			projects: mockProjects,
		}

		router := gin.New()

		// Create a test handler function that mimics the behavior
		router.GET("/projects", func(c *gin.Context) {
			projects, err := mockClient.GetAllProjects(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
				return
			}

			c.JSON(http.StatusOK, projects)
		})

		req := httptest.NewRequest(http.MethodGet, "/projects", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []projectclient.Project
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
		assert.Equal(t, "Project One", response[0].Name)
		assert.Equal(t, "Project Two", response[1].Name)
	})

	t.Run("GetAllProjects_EmptyResult", func(t *testing.T) {
		mockClient := &mockGrpcClient{
			projects: []projectclient.Project{},
		}

		router := gin.New()
		router.GET("/projects", func(c *gin.Context) {
			projects, err := mockClient.GetAllProjects(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
				return
			}

			c.JSON(http.StatusOK, projects)
		})

		req := httptest.NewRequest(http.MethodGet, "/projects", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []projectclient.Project
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Len(t, response, 0)
	})
}
