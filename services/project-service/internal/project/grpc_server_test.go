package project_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	pb "grud/api/gen/project/v1"
	"grud/testing/testdb"
	"project-service/internal/project"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectGrpcServer_Shared(t *testing.T) {
	pgContainer := testdb.SetupSharedPostgres(t)
	defer pgContainer.Cleanup(t)

	pgContainer.RunMigrations(t, (*project.Project)(nil))
	pgContainer.CreateUpdateTrigger(t, "projects")

	repo := project.NewRepository(pgContainer.DB)
	service := project.NewService(repo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	grpcServer := project.NewGrpcServer(service, logger)

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

}
