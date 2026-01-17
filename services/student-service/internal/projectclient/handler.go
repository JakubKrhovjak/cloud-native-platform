package projectclient

import (
	"log/slog"
	"net/http"

	"student-service/internal/metrics"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	grpcClient *GrpcClient
	logger     *slog.Logger
	metrics    *metrics.Metrics
}

func NewHandler(grpcClient *GrpcClient, logger *slog.Logger, metrics *metrics.Metrics) *Handler {
	return &Handler{
		grpcClient: grpcClient,
		logger:     logger,
		metrics:    metrics,
	}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/projects", h.GetAllProjects)
	router.GET("/messages", h.GetMessages)
}

func (h *Handler) GetAllProjects(c *gin.Context) {
	if h.grpcClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC client not available"})
		return
	}

	h.logger.InfoContext(c.Request.Context(), "fetching all projects from project-service via gRPC")
	projects, err := h.grpcClient.GetAllProjects(c.Request.Context())
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to fetch projects via gRPC", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	// Record metric
	h.metrics.RecordProjectsListViewedByStudent(c.Request.Context())

	c.JSON(http.StatusOK, projects)
}

func (h *Handler) GetMessages(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email parameter is required"})
		return
	}

	if h.grpcClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gRPC client not available"})
		return
	}

	h.logger.InfoContext(c.Request.Context(), "fetching messages from project-service via gRPC", "email", email)
	messages, err := h.grpcClient.GetMessagesByEmail(c.Request.Context(), email)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "failed to fetch messages via gRPC", "error", err, "email", email)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}
