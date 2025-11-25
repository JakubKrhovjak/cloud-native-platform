package project_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-service/internal/db"
	"project-service/internal/logger"
	"project-service/internal/project"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
)

type testEnv struct {
	container *postgres.PostgresContainer
	db        *bun.DB
	router    *mux.Router
	handler   *project.Handler
}

func setupTest(t *testing.T) *testEnv {
	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	database := db.NewWithDSN(connStr)
	require.NotNil(t, database)

	// Run migrations
	err = db.RunMigrations(ctx, database, (*project.Project)(nil))
	require.NoError(t, err)

	// Setup service and handler
	repo := project.NewRepository(database)
	service := project.NewService(repo)
	handler := project.NewHandler(service, logger.New())

	// Setup router
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	return &testEnv{
		container: pgContainer,
		db:        database,
		router:    router,
		handler:   handler,
	}
}

func (env *testEnv) cleanup(t *testing.T) {
	ctx := context.Background()
	if env.db != nil {
		env.db.Close()
	}
	if env.container != nil {
		if err := env.container.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}
}

func TestCreateProject(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	payload := map[string]interface{}{
		"name": "Test Project",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response project.Project
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.NotZero(t, response.ID)
	assert.NotZero(t, response.CreatedAt)
	assert.NotZero(t, response.UpdatedAt)

	expectedJSON := `{
		"name": "Test Project"
	}`

	actualJSON, _ := json.Marshal(map[string]interface{}{
		"name": response.Name,
	})

	assert.JSONEq(t, expectedJSON, string(actualJSON))
}

func TestGetProject(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create a project first
	ctx := context.Background()
	repo := project.NewRepository(env.db)
	testProject := &project.Project{
		Name: "Sample Project",
	}
	err := repo.Create(ctx, testProject)
	require.NoError(t, err)

	// Get the project
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%d", testProject.ID), nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Save body for later comparison
	responseBody := w.Body.String()

	var response project.Project
	err = json.Unmarshal([]byte(responseBody), &response)
	require.NoError(t, err)

	// Verify timestamps are set
	assert.NotZero(t, response.CreatedAt)
	assert.NotZero(t, response.UpdatedAt)

	// Compare full JSON
	expectedJSON := fmt.Sprintf(`{
		"id": %d,
		"name": "Sample Project",
		"createdAt": "%s",
		"updatedAt": "%s"
	}`, response.ID, response.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), response.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))

	assert.JSONEq(t, expectedJSON, responseBody)
}

func TestGetProjectNotFound(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/99999", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAllProjects(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create test projects
	ctx := context.Background()
	repo := project.NewRepository(env.db)
	projects := []*project.Project{
		{Name: "Project One"},
		{Name: "Project Two"},
		{Name: "Project Three"},
	}

	for _, p := range projects {
		err := repo.Create(ctx, p)
		require.NoError(t, err)
	}

	// Get all projects
	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Save body for later comparison
	responseBody := w.Body.String()

	var response []project.Project
	err := json.Unmarshal([]byte(responseBody), &response)
	require.NoError(t, err)

	assert.Len(t, response, 3)

	// Verify timestamps
	for i := range response {
		assert.NotZero(t, response[i].CreatedAt)
		assert.NotZero(t, response[i].UpdatedAt)
	}

	// Compare full JSON
	expectedJSON := fmt.Sprintf(`[
		{
			"id": %d,
			"name": "Project One",
			"createdAt": "%s",
			"updatedAt": "%s"
		},
		{
			"id": %d,
			"name": "Project Two",
			"createdAt": "%s",
			"updatedAt": "%s"
		},
		{
			"id": %d,
			"name": "Project Three",
			"createdAt": "%s",
			"updatedAt": "%s"
		}
	]`,
		response[0].ID, response[0].CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), response[0].UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		response[1].ID, response[1].CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), response[1].UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		response[2].ID, response[2].CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), response[2].UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))

	assert.JSONEq(t, expectedJSON, responseBody)
}

func TestUpdateProject(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create a project first using repository to ensure timestamps are set
	ctx := context.Background()
	repo := project.NewRepository(env.db)
	testProject := &project.Project{
		Name: "Original Name",
	}
	err := repo.Create(ctx, testProject)
	require.NoError(t, err)

	// Update the project
	payload := map[string]interface{}{
		"name": "Updated Name",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/projects/%d", testProject.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Save body for later comparison
	responseBody := w.Body.String()

	var response project.Project
	err = json.Unmarshal([]byte(responseBody), &response)
	require.NoError(t, err)

	// Verify timestamps
	assert.NotZero(t, response.CreatedAt)
	assert.NotZero(t, response.UpdatedAt)

	// Compare full JSON
	expectedJSON := fmt.Sprintf(`{
		"id": %d,
		"name": "Updated Name",
		"createdAt": "%s",
		"updatedAt": "%s"
	}`, response.ID, response.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), response.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"))

	assert.JSONEq(t, expectedJSON, responseBody)
}

func TestUpdateProjectNotFound(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	payload := map[string]interface{}{
		"name": "Updated Name",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/projects/99999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteProject(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	// Create a project first
	ctx := context.Background()
	repo := project.NewRepository(env.db)
	testProject := &project.Project{
		Name: "To Delete",
	}
	err := repo.Create(ctx, testProject)
	require.NoError(t, err)

	// Delete the project
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/projects/%d", testProject.ID), nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify deletion
	var count int
	count, err = env.db.NewSelect().Model((*project.Project)(nil)).Where("id = ?", testProject.ID).Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestDeleteProjectNotFound(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/projects/99999", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestInvalidJSON(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestInvalidProjectID(t *testing.T) {
	env := setupTest(t)
	defer env.cleanup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/projects/invalid", nil)
	w := httptest.NewRecorder()

	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
