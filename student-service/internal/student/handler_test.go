package student_test

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestStudentService_Shared(t *testing.T) {
	pgContainer := testdb.SetupSharedPostgres(t)
	defer pgContainer.Cleanup(t)

	pgContainer.RunMigrations(t, (*student.Student)(nil))

	// Create handler ONCE and reuse across all subtests
	repo := student.NewRepository(pgContainer.DB)
	service := student.NewService(repo)
	handler := student.NewHandler(service, logger.New())
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	t.Run("CreateStudent", func(t *testing.T) {
		// Only cleanup tables, reuse handler
		testdb.CleanupTables(t, pgContainer.DB, "students")

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

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response student.Student
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.NotZero(t, response.ID)
	})

	t.Run("GetStudent", func(t *testing.T) {
		// Only cleanup tables, reuse handler
		testdb.CleanupTables(t, pgContainer.DB, "students")

		// Create a student first
		ctx := context.Background()
		testStudent := &student.Student{
			FirstName: "Jane",
			LastName:  "Doe",
			Email:     "jane.doe@example.com",
			Major:     "Mathematics",
			Year:      3,
		}
		_, err := pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
		require.NoError(t, err)

		// Get the student
		req := httptest.NewRequest(http.MethodGet, "/api/students/1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify response body
		var response student.Student
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		// Verify correct data returned
		assert.Equal(t, "Jane", response.FirstName)
		assert.Equal(t, "Doe", response.LastName)
		assert.Equal(t, "jane.doe@example.com", response.Email)
		assert.Equal(t, "Mathematics", response.Major)
		assert.Equal(t, 3, response.Year)
		assert.NotZero(t, response.ID)
	})

	t.Run("GetAllStudents", func(t *testing.T) {
		// Only cleanup tables, reuse handler
		testdb.CleanupTables(t, pgContainer.DB, "students")

		// Create test students
		ctx := context.Background()
		students := []*student.Student{
			{FirstName: "Student", LastName: "One", Email: "s1@example.com", Major: "Physics", Year: 1},
			{FirstName: "Student", LastName: "Two", Email: "s2@example.com", Major: "Chemistry", Year: 2},
		}

		for _, s := range students {
			_, err := pgContainer.DB.NewInsert().Model(s).Exec(ctx)
			require.NoError(t, err)
		}

		// Get all students
		req := httptest.NewRequest(http.MethodGet, "/api/students", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []student.Student
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Len(t, response, 2)

		// Verify first student
		assert.Equal(t, "Student", response[0].FirstName)
		assert.Equal(t, "One", response[0].LastName)
		assert.Equal(t, "s1@example.com", response[0].Email)
		assert.Equal(t, "Physics", response[0].Major)
		assert.Equal(t, 1, response[0].Year)
		assert.NotZero(t, response[0].ID)

		// Verify second student
		assert.Equal(t, "Student", response[1].FirstName)
		assert.Equal(t, "Two", response[1].LastName)
		assert.Equal(t, "s2@example.com", response[1].Email)
		assert.Equal(t, "Chemistry", response[1].Major)
		assert.Equal(t, 2, response[1].Year)
		assert.NotZero(t, response[1].ID)
	})

	t.Run("GetStudentNotFound", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students")

		req := httptest.NewRequest(http.MethodGet, "/api/students/99999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("UpdateStudent", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students")

		// Create a student first
		ctx := context.Background()
		testStudent := &student.Student{
			FirstName: "Original",
			LastName:  "Name",
			Email:     "original@example.com",
			Major:     "Engineering",
			Year:      1,
		}
		_, err := pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
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

		req := httptest.NewRequest(http.MethodPut, "/api/students/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response student.Student
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Updated", response.FirstName)
		assert.Equal(t, "updated@example.com", response.Email)
		assert.Equal(t, "Computer Science", response.Major)
		assert.Equal(t, 4, response.Year)
	})

	t.Run("UpdateStudentNotFound", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students")

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

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("DeleteStudent", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students")

		// Create a student first
		ctx := context.Background()
		testStudent := &student.Student{
			FirstName: "To",
			LastName:  "Delete",
			Email:     "delete@example.com",
			Major:     "History",
			Year:      2,
		}
		_, err := pgContainer.DB.NewInsert().Model(testStudent).Exec(ctx)
		require.NoError(t, err)

		// Delete the student
		req := httptest.NewRequest(http.MethodDelete, "/api/students/1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify deletion
		var count int
		count, err = pgContainer.DB.NewSelect().Model((*student.Student)(nil)).Where("id = ?", testStudent.ID).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("DeleteStudentNotFound", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students")

		req := httptest.NewRequest(http.MethodDelete, "/api/students/99999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students")

		req := httptest.NewRequest(http.MethodPost, "/api/students", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("InvalidStudentID", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "students")

		req := httptest.NewRequest(http.MethodGet, "/api/students/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
