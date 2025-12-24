package project_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	pb "grud/api/gen/project/v1"
	commonmetrics "grud/common/metrics"
	"grud/testing/testdb"
	projectmetrics "project-service/internal/metrics"
	"project-service/internal/project"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectGrpcServer_Shared(t *testing.T) {
	pgContainer := testdb.SetupSharedPostgres(t)
	defer pgContainer.Cleanup(t)

	pgContainer.RunMigrations(t, (*project.Project)(nil))
	pgContainer.CreateUpdateTrigger(t, "projects")

	mockServiceMetrics := projectmetrics.NewMock()
	mockRepoMetrics := commonmetrics.NewMock()
	repo := project.NewRepository(pgContainer.DB, mockRepoMetrics)
	service := project.NewService(repo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	grpcServer := project.NewGrpcServer(service, logger, mockServiceMetrics)

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

		req := &pb.GetAllProjectsRequest{}
		resp, err := grpcServer.GetAllProjects(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Projects, 3)

		assert.Equal(t, "Project One", resp.Projects[0].Name)
		assert.Equal(t, "Project Two", resp.Projects[1].Name)
		assert.Equal(t, "Project Three", resp.Projects[2].Name)

		for i := range resp.Projects {
			assert.NotZero(t, resp.Projects[i].Id)
			assert.NotZero(t, resp.Projects[i].CreatedAt)
			assert.NotZero(t, resp.Projects[i].UpdatedAt)
		}
	})

	t.Run("GetAllProjects_EmptyResult", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		req := &pb.GetAllProjectsRequest{}
		resp, err := grpcServer.GetAllProjects(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Projects, 0)
	})

	t.Run("CreateProject", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		req := &pb.CreateProjectRequest{
			Name: "New Project",
		}
		resp, err := grpcServer.CreateProject(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Project)
		assert.NotZero(t, resp.Project.Id)
		assert.Equal(t, "New Project", resp.Project.Name)
		assert.NotZero(t, resp.Project.CreatedAt)
		assert.NotZero(t, resp.Project.UpdatedAt)
	})

	t.Run("GetProject", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		p := &project.Project{Name: "Test Project"}
		_, err := pgContainer.DB.NewInsert().Model(p).Exec(ctx)
		require.NoError(t, err)

		req := &pb.GetProjectRequest{Id: int32(p.ID)}
		resp, err := grpcServer.GetProject(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Project)
		assert.Equal(t, int32(p.ID), resp.Project.Id)
		assert.Equal(t, "Test Project", resp.Project.Name)
		assert.NotZero(t, resp.Project.CreatedAt)
		assert.NotZero(t, resp.Project.UpdatedAt)
	})

	t.Run("UpdateProject", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		p := &project.Project{Name: "Old Name"}
		_, err := pgContainer.DB.NewInsert().Model(p).Exec(ctx)
		require.NoError(t, err)

		req := &pb.UpdateProjectRequest{
			Id:   int32(p.ID),
			Name: "Updated Name",
		}
		resp, err := grpcServer.UpdateProject(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Project)
		assert.Equal(t, int32(p.ID), resp.Project.Id)
		assert.Equal(t, "Updated Name", resp.Project.Name)
		assert.NotZero(t, resp.Project.CreatedAt)
		assert.NotZero(t, resp.Project.UpdatedAt)
	})

	t.Run("DeleteProject", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "projects")

		ctx := context.Background()
		p := &project.Project{Name: "To Delete"}
		_, err := pgContainer.DB.NewInsert().Model(p).Exec(ctx)
		require.NoError(t, err)

		req := &pb.DeleteProjectRequest{Id: int32(p.ID)}
		resp, err := grpcServer.DeleteProject(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify project is deleted
		var count int
		count, err = pgContainer.DB.NewSelect().Model((*project.Project)(nil)).Where("id = ?", p.ID).Count(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

}
