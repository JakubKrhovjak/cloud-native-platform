package projectclient

import (
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"

	"github.com/gorilla/mux"
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

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/projects", h.GetAllProjects).Methods("GET")
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
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch projects via gRPC")
			return
		}
	} else {
		h.logger.Info("fetching all projects from project-service via HTTP")
		projects, err = h.httpClient.GetAllProjects(r.Context())
		if err != nil {
			h.logger.Error("failed to fetch projects via HTTP", "error", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to fetch projects via HTTP")
			return
		}
	}

	respondWithJSON(w, http.StatusOK, projects)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
