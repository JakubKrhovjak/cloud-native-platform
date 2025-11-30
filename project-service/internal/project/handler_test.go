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

func TestProjectService_Shared(t *testing.T) {
	pgContainer := testdb.SetupSharedPostgres(t)
	defer pgContainer.Cleanup(t)

	pgContainer.RunMigrations(t, (*project.Project)(nil))
	pgContainer.CreateUpdateTrigger(t, "projects")

	repo := project.NewRepository(pgContainer.DB)
	service := project.NewService(repo)
	handler := project.NewHandler(service, logger.New())
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	t.Run("CreateProject", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		payload := map[string]interface{}{
			"name": "Test Project",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response project.Project
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)
		assert.NotZero(t, response.ID)
		assert.NotZero(t, response.CreatedAt)
		assert.NotZero(t, response.UpdatedAt)
		assert.Equal(t, "Test Project", response.Name)
	})

	t.Run("GetProject", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		testProject := &project.Project{
			Name: "Sample Project",
		}
		_, err := pgContainer.DB.NewInsert().Model(testProject).Exec(ctx)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%d", testProject.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response project.Project
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, testProject.ID, response.ID)
		assert.Equal(t, "Sample Project", response.Name)
		assert.NotZero(t, response.CreatedAt)
		assert.NotZero(t, response.UpdatedAt)
	})

	t.Run("GetProjectNotFound", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		req := httptest.NewRequest(http.MethodGet, "/api/projects/99999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetAllProjects", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		projects := []*project.Project{
			{Name: "Project One"},
			{Name: "Project Two"},
			{Name: "Project Three"},
		}

		for _, p := range projects {
			_, err := pgContainer.DB.NewInsert().Model(p).Exec(ctx)
			require.NoError(t, err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []project.Project
		err := json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Len(t, response, 3)

		assert.Equal(t, "Project One", response[0].Name)
		assert.Equal(t, "Project Two", response[1].Name)
		assert.Equal(t, "Project Three", response[2].Name)

		for i := range response {
			assert.NotZero(t, response[i].ID)
			assert.NotZero(t, response[i].CreatedAt)
			assert.NotZero(t, response[i].UpdatedAt)
		}
	})

	t.Run("UpdateProject", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		testProject := &project.Project{
			Name: "Original Name",
		}
		_, err := pgContainer.DB.NewInsert().Model(testProject).Exec(ctx)
		require.NoError(t, err)

		payload := map[string]interface{}{
			"name": "Updated Name",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/projects/%d", testProject.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response project.Project
		err = json.NewDecoder(w.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, testProject.ID, response.ID)
		assert.Equal(t, "Updated Name", response.Name)
		assert.NotZero(t, response.CreatedAt)
		assert.NotZero(t, response.UpdatedAt)
	})

	t.Run("UpdateProjectNotFound", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		payload := map[string]interface{}{
			"name": "Updated Name",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPut, "/api/projects/99999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("DeleteProject", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		testProject := &project.Project{
			Name: "To Delete",
		}
		_, err := pgContainer.DB.NewInsert().Model(testProject).Exec(ctx)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/projects/%d", testProject.ID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		var count int
		count, err = pgContainer.DB.NewSelect().Model((*project.Project)(nil)).Where("id = ?", testProject.ID).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("DeleteProjectNotFound", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		req := httptest.NewRequest(http.MethodDelete, "/api/projects/99999", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("InvalidProjectID", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		req := httptest.NewRequest(http.MethodGet, "/api/projects/invalid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
