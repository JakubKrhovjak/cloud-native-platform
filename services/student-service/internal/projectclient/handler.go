package projectclient

import (
	"log/slog"
	"net/http"

	"student-service/internal/metrics"

	"grud/common/httputil"

	"github.com/go-chi/chi/v5"
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

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/projects", h.GetAllProjects)
	router.Get("/messages", h.GetMessages)
}

func (h *Handler) GetAllProjects(w http.ResponseWriter, r *http.Request) {
	if h.grpcClient == nil {
		httputil.RespondWithError(w, http.StatusServiceUnavailable, "gRPC client not available")
		return
	}

	h.logger.Info("fetching all projects from project-service via gRPC")
	projects, err := h.grpcClient.GetAllProjects(r.Context())
	if err != nil {
		h.logger.Error("failed to fetch projects via gRPC", "error", err)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch projects")
		return
	}

	// Record metric
	h.metrics.RecordProjectsListViewedByStudent(r.Context())

	httputil.RespondWithJSON(w, http.StatusOK, projects)
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		httputil.RespondWithError(w, http.StatusBadRequest, "Email parameter is required")
		return
	}

	if h.grpcClient == nil {
		httputil.RespondWithError(w, http.StatusServiceUnavailable, "gRPC client not available")
		return
	}

	h.logger.Info("fetching messages from project-service via gRPC", "email", email)
	messages, err := h.grpcClient.GetMessagesByEmail(r.Context(), email)
	if err != nil {
		h.logger.Error("failed to fetch messages via gRPC", "error", err, "email", email)
		httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, messages)
}
