package student

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"student-service/internal/metrics"

	"grud/common/httputil"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	service  Service
	validate *validator.Validate
	logger   *slog.Logger
	metrics  *metrics.Metrics
}

func NewHandler(service Service, logger *slog.Logger, metrics *metrics.Metrics) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
		logger:   logger,
		metrics:  metrics,
	}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Post("/students", h.CreateStudent)
	router.Get("/students", h.GetAllStudents)
	router.Get("/students/{id}", h.GetStudent)
	router.Put("/students/{id}", h.UpdateStudent)
	router.Delete("/students/{id}", h.DeleteStudent)
}

func (h *Handler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil || h.validate.Struct(&student) != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Set default password for students created via API
	// In production, students should be created via /auth/register
	if student.Password == "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("DefaultPassword123!"), bcrypt.DefaultCost)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "failed to hash password", "error", err)
			httputil.RespondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		student.Password = string(hashedPassword)
	}

	h.logger.InfoContext(r.Context(), "creating student", "email", student.Email)
	createdStudent, err := h.service.CreateStudent(r.Context(), &student)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Record metric
	h.metrics.RecordStudentRegistration(r.Context())

	httputil.RespondWithJSON(w, http.StatusCreated, createdStudent)
}

func (h *Handler) GetAllStudents(w http.ResponseWriter, r *http.Request) {
	h.logger.InfoContext(r.Context(), "fetching all students")

	students, err := h.service.GetAllStudents(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Record metric
	h.metrics.RecordStudentsListViewed(r.Context())

	httputil.RespondWithJSON(w, http.StatusOK, students)
}

func (h *Handler) GetStudent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	h.logger.InfoContext(r.Context(), "fetching student by ID")
	student, err := h.service.GetStudentByID(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Record metric
	h.metrics.RecordStudentViewed(r.Context())

	httputil.RespondWithJSON(w, http.StatusOK, student)
}

func (h *Handler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil || h.validate.Struct(&student) != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	student.ID = id

	h.logger.InfoContext(r.Context(), "updating student", "email", student.Email)
	if err := h.service.UpdateStudent(r.Context(), &student); err != nil {
		h.handleServiceError(w, err)
		return
	}

	httputil.RespondWithJSON(w, http.StatusOK, student)
}

func (h *Handler) DeleteStudent(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httputil.RespondWithError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	h.logger.InfoContext(r.Context(), "deleting student")
	if err := h.service.DeleteStudent(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrStudentNotFound) {
		h.logger.Info("student not found")
		httputil.RespondWithError(w, http.StatusNotFound, "Student not found")
		return
	}
	if errors.Is(err, ErrInvalidInput) {
		h.logger.Info("invalid input")
		httputil.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.logger.Error("internal error")
	httputil.RespondWithError(w, http.StatusInternalServerError, err.Error())
}
