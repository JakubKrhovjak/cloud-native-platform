package student

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
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
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := h.service.CreateStudent(r.Context(), &student); err != nil {
		if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrInvalidEmail) {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, student)
}

func (h *Handler) GetAllStudents(w http.ResponseWriter, r *http.Request) {
	students, err := h.service.GetAllStudents(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, students)
}

func (h *Handler) GetStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	student, err := h.service.GetStudentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			respondWithError(w, http.StatusNotFound, "Student not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, student)
}

func (h *Handler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid student ID")
		return
	}

	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	student.ID = id

	if err := h.service.UpdateStudent(r.Context(), &student); err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			respondWithError(w, http.StatusNotFound, "Student not found")
			return
		}
		if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrInvalidEmail) {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
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

	if err := h.service.DeleteStudent(r.Context(), id); err != nil {
		if errors.Is(err, ErrStudentNotFound) {
			respondWithError(w, http.StatusNotFound, "Student not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
