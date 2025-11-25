package project_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"grud/testing/testdb"
	"project-service/internal/logger"
	"project-service/internal/project"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	pgContainer *testdb.PostgresContainer
	router      *mux.Router
	handler     *project.Handler
}

func setupTest(t *testing.T) *testEnv {
	t.Helper()

	// Setup PostgreSQL testcontainer
	pgContainer := testdb.SetupPostgres(t)

	// Run migrations
	pgContainer.RunMigrations(t, (*project.Project)(nil))
	pgContainer.CreateUpdateTrigger(t, "projects")

	// Setup service and handler
	repo := project.NewRepository(pgContainer.DB)
	service := project.NewService(repo)
	handler := project.NewHandler(service, logger.New())

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
	repo := project.NewRepository(env.pgContainer.DB)
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
	repo := project.NewRepository(env.pgContainer.DB)
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
	repo := project.NewRepository(env.pgContainer.DB)
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
	repo := project.NewRepository(env.pgContainer.DB)
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
	count, err = env.pgContainer.DB.NewSelect().Model((*project.Project)(nil)).Where("id = ?", testProject.ID).Count(ctx)
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
