package message

import (
	"log/slog"
	"net/http"

	"student-service/internal/auth"
	"student-service/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service  *Service
	validate *validator.Validate
	logger   *slog.Logger
	metrics  *metrics.Metrics
}

func NewHandler(service *Service, logger *slog.Logger, metrics *metrics.Metrics) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
		logger:   logger,
		metrics:  metrics,
	}
}

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.POST("/messages", h.SendMessage)
}

func (h *Handler) SendMessage(c *gin.Context) {
	// Get email from auth context
	email, ok := auth.GetEmail(c.Request.Context())
	if !ok {
		h.logger.WarnContext(c.Request.Context(), "email not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Parse request
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Validate request
	if err := h.validate.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	h.logger.InfoContext(c.Request.Context(), "sending message", "email", email, "message", req.Message)

	// Send message via service
	if err := h.service.SendMessage(c.Request.Context(), email, req.Message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message"})
		return
	}

	// Record metric
	h.metrics.RecordMessageSent(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "message sent successfully",
	})
}
