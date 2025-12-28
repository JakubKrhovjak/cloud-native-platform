package message

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"student-service/internal/auth"
	"student-service/internal/metrics"

	"grud/common/httputil"

	"github.com/go-chi/chi/v5"
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

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/messages", h.SendMessage)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	// Get email from auth context
	email, ok := auth.GetEmail(r.Context())
	if !ok {
		h.logger.WarnContext(r.Context(), "email not found in context")
		httputil.RespondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Validate request
	if err := h.validate.Struct(&req); err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "message is required")
		return
	}

	h.logger.InfoContext(r.Context(), "sending message", "email", email, "message", req.Message)

	// Send message via service
	if err := h.service.SendMessage(r.Context(), email, req.Message); err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	// Record metric
	h.metrics.RecordMessageSent(r.Context())

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "message sent successfully",
	})
}
