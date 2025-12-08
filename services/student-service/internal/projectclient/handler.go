package projectclient

import (
	"log/slog"
	"math/rand"
	"net/http"

	"grud/common/httputil"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	httpClient *Client
	grpcClient *GrpcClient
	logger     *slog.Logger
}

func NewHandler(httpClient *Client, grpcClient *GrpcClient, logger *slog.Logger) *Handler {
	return &Handler{
		httpClient: httpClient,
		grpcClient: grpcClient,
		logger:     logger,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Get("/projects", h.GetAllProjects)
}

func (h *Handler) GetAllProjects(w http.ResponseWriter, r *http.Request) {
	// Randomly choose between HTTP and gRPC (50/50 chance)
	useGrpc := rand.Intn(2) == 0

	var projects []Project
	var err error

	if useGrpc && h.grpcClient != nil {
		h.logger.Info("fetching all projects from project-service via gRPC")
		projects, err = h.grpcClient.GetAllProjects(r.Context())
		if err != nil {
			h.logger.Error("failed to fetch projects via gRPC", "error", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch projects via gRPC")
			return
		}
	} else {
		h.logger.Info("fetching all projects from project-service via HTTP")
		projects, err = h.httpClient.GetAllProjects(r.Context())
		if err != nil {
			h.logger.Error("failed to fetch projects via HTTP", "error", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Failed to fetch projects via HTTP")
			return
		}
	}

	httputil.RespondWithJSON(w, http.StatusOK, projects)
}
