package student

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
	router.HandleFunc("/api/students", h.CreateStudent).Methods("POST")
	router.HandleFunc("/api/students", h.GetAllStudents).Methods("GET")
	router.HandleFunc("/api/students/{id}", h.GetStudent).Methods("GET")
	router.HandleFunc("/api/students/{id}", h.UpdateStudent).Methods("PUT")
	router.HandleFunc("/api/students/{id}", h.DeleteStudent).Methods("DELETE")
}

func (h *Handler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil || h.validate.Struct(&student) != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	h.logger.Info("creating student", "email", student.Email)
	if err := h.service.CreateStudent(r.Context(), &student); err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, student)
}

func (h *Handler) GetAllStudents(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("fetching all students")

	students, err := h.service.GetAllStudents(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, students)
}

func (h *Handler) GetStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	h.logger.Info("fetching student by ID")
	student, err := h.service.GetStudentByID(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, student)
}

func (h *Handler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil || h.validate.Struct(&student) != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	student.ID = id

	h.logger.Info("updating student", "email", student.Email)
	if err := h.service.UpdateStudent(r.Context(), &student); err != nil {
		h.handleServiceError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, student)
}

func (h *Handler) DeleteStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	h.logger.Info("deleting student")
	if err := h.service.DeleteStudent(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrStudentNotFound) {
		h.logger.Info("student not found")
		respondWithError(w, http.StatusNotFound, "Student not found")
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
