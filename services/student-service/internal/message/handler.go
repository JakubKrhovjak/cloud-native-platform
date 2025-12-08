package message

import (
	"log/slog"
	"net/http"

	"student-service/internal/auth"

	"grud/common/httputil"

	"encoding/json"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service  *Service
	validate *validator.Validate
	logger   *slog.Logger
}

func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
		logger:   logger,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/messages", h.SendMessage)
}

func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	// Get email from auth context
	email, ok := auth.GetEmail(r.Context())
	if !ok {
		h.logger.Warn("email not found in context")
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

	h.logger.Info("sending message", "email", email, "message", req.Message)

	// Send message via service
	if err := h.service.SendMessage(email, req.Message); err != nil {
		httputil.RespondWithError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "message sent successfully",
	})
}
