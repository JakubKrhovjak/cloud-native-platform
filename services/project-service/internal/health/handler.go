package health

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/health", h.Health).Methods("GET")
	router.HandleFunc("/ready", h.Ready).Methods("GET")
}

type HealthResponse struct {
	Status string `json:"status"`
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{Status: "ok"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{Status: "ready"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
