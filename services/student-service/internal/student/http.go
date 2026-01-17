package student

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"student-service/internal/metrics"

	"github.com/gin-gonic/gin"
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

func (h *Handler) RegisterRoutes(router gin.IRouter) {
	router.POST("/students", h.CreateStudent)
	router.GET("/students", h.GetAllStudents)
	router.GET("/students/:id", h.GetStudent)
	router.PUT("/students/:id", h.UpdateStudent)
	router.DELETE("/students/:id", h.DeleteStudent)
}

func (h *Handler) CreateStudent(c *gin.Context) {
	var student Student
	if err := c.ShouldBindJSON(&student); err != nil || h.validate.Struct(&student) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Set default password for students created via API
	// In production, students should be created via /auth/register
	if student.Password == "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("DefaultPassword123!"), bcrypt.DefaultCost)
		if err != nil {
			h.logger.ErrorContext(c.Request.Context(), "failed to hash password", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}
		student.Password = string(hashedPassword)
	}

	h.logger.InfoContext(c.Request.Context(), "creating student", "email", student.Email)
	createdStudent, err := h.service.CreateStudent(c.Request.Context(), &student)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Record metric
	h.metrics.RecordStudentRegistration(c.Request.Context())

	c.JSON(http.StatusCreated, createdStudent)
}

func (h *Handler) GetAllStudents(c *gin.Context) {
	h.logger.InfoContext(c.Request.Context(), "fetching all students")

	students, err := h.service.GetAllStudents(c.Request.Context())
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Record metric
	h.metrics.RecordStudentsListViewed(c.Request.Context())

	c.JSON(http.StatusOK, students)
}

func (h *Handler) GetStudent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student ID"})
		return
	}

	h.logger.InfoContext(c.Request.Context(), "fetching student by ID")
	student, err := h.service.GetStudentByID(c.Request.Context(), id)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	// Record metric
	h.metrics.RecordStudentViewed(c.Request.Context())

	c.JSON(http.StatusOK, student)
}

func (h *Handler) UpdateStudent(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var student Student
	if err := c.ShouldBindJSON(&student); err != nil || h.validate.Struct(&student) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	student.ID = id

	h.logger.InfoContext(c.Request.Context(), "updating student", "email", student.Email)
	if err := h.service.UpdateStudent(c.Request.Context(), &student); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, student)
}

func (h *Handler) DeleteStudent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid student ID"})
		return
	}

	h.logger.InfoContext(c.Request.Context(), "deleting student")
	if err := h.service.DeleteStudent(c.Request.Context(), id); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) handleServiceError(c *gin.Context, err error) {
	if errors.Is(err, ErrStudentNotFound) {
		h.logger.Info("student not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}
	if errors.Is(err, ErrInvalidInput) {
		h.logger.Info("invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.logger.Error("internal error")
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
