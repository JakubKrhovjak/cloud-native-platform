package project

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

type Handler struct {
	service  Service
	validate *validator.Validate
	logger   *slog.Logger
}

func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
		logger:   logger,
	}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/projects", h.CreateProject).Methods("POST")
	router.HandleFunc("/api/projects", h.GetAllProjects).Methods("GET")
	router.HandleFunc("/api/projects/{id}", h.GetProject).Methods("GET")
	router.HandleFunc("/api/projects/{id}", h.UpdateProject).Methods("PUT")
	router.HandleFunc("/api/projects/{id}", h.DeleteProject).Methods("DELETE")
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var project Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil || h.validate.Struct(&project) != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	h.logger.Info("creating project", "name", project.Name)
	if err := h.service.CreateProject(r.Context(), &project); err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, project)
}

func (h *Handler) GetAllProjects(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("fetching all projects")

	projects, err := h.service.GetAllProjects(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, projects)
}

func (h *Handler) GetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	h.logger.Info("fetching project by ID")
	project, err := h.service.GetProjectByID(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, project)
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	var project Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil || h.validate.Struct(&project) != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	project.ID = id

	h.logger.Info("updating project", "name", project.Name)
	if err := h.service.UpdateProject(r.Context(), &project); err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Fetch updated project from DB to get all fields including timestamps
	updatedProject, err := h.service.GetProjectByID(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedProject)
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid project ID")
		return
	}

	h.logger.Info("deleting project")
	if err := h.service.DeleteProject(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrProjectNotFound) {
		h.logger.Info("project not found")
		respondWithError(w, http.StatusNotFound, "Project not found")
		return
	}
	if errors.Is(err, ErrInvalidInput) {
		h.logger.Info("invalid input")
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.logger.Error("internal error")
	respondWithError(w, http.StatusInternalServerError, err.Error())
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
