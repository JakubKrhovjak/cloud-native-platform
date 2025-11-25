package student_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"grud/testing/testdb"
	"student-service/internal/logger"
	"student-service/internal/student"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	pgContainer *testdb.PostgresContainer
	router      *mux.Router
	handler     *student.Handler
}

func setupTest(t *testing.T) *testEnv {
	t.Helper()

	// Setup PostgreSQL testcontainer
	pgContainer := testdb.SetupPostgres(t)

	// Run migrations
	pgContainer.RunMigrations(t, (*student.Student)(nil))

	// Setup service and handler
	repo := student.NewRepository(pgContainer.DB)
	service := student.NewService(repo)
	handler := student.NewHandler(service, logger.New())

	// Setup router
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	return &testEnv{
		pgContainer: pgContainer,
		router:      router,
		handler:     handler,
	}
}

func (env *testEnv) cleanup(t *testing.T) {
	t.Helper()
	env.pgContainer.Cleanup(t)
}

func TestCreateStudent(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	payload := map[string]interface{}{
		"firstName": "John",
		"lastName":  "Doe",
		"email":     "john.doe@example.com",
		"major":     "Computer Science",
		"year":      2,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/students", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response student.Student
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotZero(t, response.ID)

	expectedJSON := `{
		"firstName": "John",
		"lastName": "Doe",
		"email": "john.doe@example.com",
		"major": "Computer Science",
		"year": 2
	}`

	actualJSON, _ := json.Marshal(map[string]interface{}{
		"firstName": response.FirstName,
		"lastName":  response.LastName,
		"email":     response.Email,
		"major":     response.Major,
		"year":      response.Year,
	})

	assert.JSONEq(t, expectedJSON, string(actualJSON))
}

func TestGetStudent(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create a student first
	ctx := context.Background()
	testStudent := &student.Student{
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     "jane.doe@example.com",
		Major:     "Mathematics",
		Year:      3,
	}
	_, err := env.pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
	require.NoError(t, err)

	// Get the student
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/students/%d", testStudent.ID), nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expectedJSON := fmt.Sprintf(`{
		"id": %d,
		"firstName": "Jane",
		"lastName": "Doe",
		"email": "jane.doe@example.com",
		"major": "Mathematics",
		"year": 3
	}`, testStudent.ID)

	assert.JSONEq(t, expectedJSON, w.Body.String())
}

func TestGetStudentNotFound(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/students/99999", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAllStudents(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create test students
	ctx := context.Background()
	students := []*student.Student{
		{FirstName: "Student", LastName: "One", Email: "s1@example.com", Major: "Physics", Year: 1},
		{FirstName: "Student", LastName: "Two", Email: "s2@example.com", Major: "Chemistry", Year: 2},
		{FirstName: "Student", LastName: "Three", Email: "s3@example.com", Major: "Biology", Year: 3},
	}

	for _, s := range students {
		_, err := env.pgContainer.DB.NewInsert().Model(s).Exec(ctx)
		require.NoError(t, err)
	}

	// Get all students
	req := httptest.NewRequest(http.MethodGet, "/api/students", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expectedJSON := fmt.Sprintf(`[
		{
			"id": %d,
			"firstName": "Student",
			"lastName": "One",
			"email": "s1@example.com",
			"major": "Physics",
			"year": 1
		},
		{
			"id": %d,
			"firstName": "Student",
			"lastName": "Two",
			"email": "s2@example.com",
			"major": "Chemistry",
			"year": 2
		},
		{
			"id": %d,
			"firstName": "Student",
			"lastName": "Three",
			"email": "s3@example.com",
			"major": "Biology",
			"year": 3
		}
	]`, students[0].ID, students[1].ID, students[2].ID)

	assert.JSONEq(t, expectedJSON, w.Body.String())
}

func TestUpdateStudent(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create a student first
	ctx := context.Background()
	testStudent := &student.Student{
		FirstName: "Original",
		LastName:  "Name",
		Email:     "original@example.com",
		Major:     "Engineering",
		Year:      1,
	}
	_, err := env.pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
	require.NoError(t, err)

	// Update the student
	payload := map[string]interface{}{
		"firstName": "Updated",
		"lastName":  "Name",
		"email":     "updated@example.com",
		"major":     "Computer Science",
		"year":      4,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/students/%d", testStudent.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expectedJSON := fmt.Sprintf(`{
		"id": %d,
		"firstName": "Updated",
		"lastName": "Name",
		"email": "updated@example.com",
		"major": "Computer Science",
		"year": 4
	}`, testStudent.ID)

	assert.JSONEq(t, expectedJSON, w.Body.String())
}

func TestUpdateStudentNotFound(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	payload := map[string]interface{}{
		"firstName": "Updated",
		"lastName":  "Name",
		"email":     "updated@example.com",
		"major":     "Computer Science",
		"year":      4,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/students/99999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteStudent(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create a student first
	ctx := context.Background()
	testStudent := &student.Student{
		FirstName: "To",
		LastName:  "Delete",
		Email:     "delete@example.com",
		Major:     "History",
		Year:      2,
	}
	_, err := env.pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
	require.NoError(t, err)

	// Delete the student
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/students/%d", testStudent.ID), nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify deletion
	var count int
	count, err = env.pgContainer.DB.NewSelect().Model((*student.Student)(nil)).Where("id = ?", testStudent.ID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestDeleteStudentNotFound(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/students/99999", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInvalidJSON(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/students", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInvalidStudentID(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/students/invalid", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Mock project service for testing GET /api/projects endpoint
func TestGetAllProjects_WithMockProjectService(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create mock project-service
	mockProjectService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/projects", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		projects := []map[string]interface{}{
			{"id": 1, "name": "Mock Project 1", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-01T00:00:00Z"},
			{"id": 2, "name": "Mock Project 2", "created_at": "2024-01-02T00:00:00Z", "updated_at": "2024-01-02T00:00:00Z"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projects)
	}))
	defer mockProjectService.Close()

	// Note: This test shows the structure, but we'd need to inject the mock URL into the handler
	// For a real implementation, the handler would need to accept the project service URL as a dependency
	t.Log("Mock project service URL:", mockProjectService.URL)
	t.Skip("Skipping - requires dependency injection of project service URL")
}

func TestGetAllProjects_ProjectServiceDown(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create mock that returns error
	mockProjectService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockProjectService.Close()

	t.Log("Mock project service URL (error):", mockProjectService.URL)
	t.Skip("Skipping - requires dependency injection of project service URL")
}
