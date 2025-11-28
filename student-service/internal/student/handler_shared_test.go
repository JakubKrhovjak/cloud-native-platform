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

// Example of using shared container for faster tests
// Run with: go test -run TestStudentService_Shared

type sharedTestEnv struct {
	pgContainer *testdb.PostgresContainer
	router      *mux.Router
	handler     *student.Handler
}

func setupSharedTest(t *testing.T) *sharedTestEnv {
	t.Helper()

	// Use shared PostgreSQL container (created once)
	pgContainer := testdb.SetupSharedPostgres(t)

	// Clean tables before each test (ensures clean state)
	testdb.CleanupTables(t, pgContainer.DB, "students")

	// Setup service and handler
	repo := student.NewRepository(pgContainer.DB)
	service := student.NewService(repo)
	handler := student.NewHandler(service, logger.New())

	// Setup router
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	return &sharedTestEnv{
		pgContainer: pgContainer,
		router:      router,
		handler:     handler,
	}
}

// Example: Run all CRUD tests with shared container
// This is ~10x faster than individual containers
func TestStudentService_Shared(t *testing.T) {
	// Setup shared container once for all subtests
	pgContainer := testdb.SetupSharedPostgres(t)
	defer pgContainer.Cleanup(t)

	// Run migrations once
	pgContainer.RunMigrations(t, (*student.Student)(nil))

	// NOTE: These subtests CANNOT run in parallel because they share a DB

	t.Run("CreateStudent", func(t *testing.T) {
		env := setupSharedTest(t)

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
	})

	t.Run("GetStudent", func(t *testing.T) {
		env := setupSharedTest(t)

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
		req := httptest.NewRequest(http.MethodGet, "/api/students/1", nil)
		w := httptest.NewRecorder()

		env.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("GetAllStudents", func(t *testing.T) {
		env := setupSharedTest(t)

		// Create test students
		ctx := context.Background()
		students := []*student.Student{
			{FirstName: "Student", LastName: "One", Email: "s1@example.com", Major: "Physics", Year: 1},
			{FirstName: "Student", LastName: "Two", Email: "s2@example.com", Major: "Chemistry", Year: 2},
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

		var response []student.Student
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Len(t, response, 2)
	})
}

// Benchmark comparison: shared vs isolated containers
func BenchmarkSharedContainer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		t := &testing.T{}
		env := setupSharedTest(t)

		req := httptest.NewRequest(http.MethodGet, "/api/students", nil)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)
	}
}

func BenchmarkIsolatedContainer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		t := &testing.T{}
		env := setupTest(t)

		req := httptest.NewRequest(http.MethodGet, "/api/students", nil)
		w := httptest.NewRecorder()
		env.router.ServeHTTP(w, req)

		env.cleanup(t)
	}
}
