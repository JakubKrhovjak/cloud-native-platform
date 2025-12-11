package message_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	pb "grud/api/gen/message/v1"
	"grud/testing/testdb"
	"project-service/internal/message"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageGrpcServer_Shared(t *testing.T) {
	pgContainer := testdb.SetupSharedPostgres(t)
	defer pgContainer.Cleanup(t)

	pgContainer.RunMigrations(t, (*message.Message)(nil))

	repo := message.NewRepository(pgContainer.DB)
	service := message.NewService(repo)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	grpcServer := message.NewGrpcServer(service, logger)

	t.Run("GetMessagesByEmail", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "messages")

		ctx := context.Background()
		messages := []*message.Message{
			{Email: "test@example.com", Message: "First message"},
			{Email: "test@example.com", Message: "Second message"},
			{Email: "other@example.com", Message: "Other message"},
		}

		for _, msg := range messages {
			_, err := pgContainer.DB.NewInsert().Model(msg).Exec(ctx)
			require.NoError(t, err)
		}

		req := &pb.GetMessagesByEmailRequest{
			Email: "test@example.com",
		}
		resp, err := grpcServer.GetMessagesByEmail(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Messages, 2)

		assert.Equal(t, "test@example.com", resp.Messages[0].Email)
		assert.Equal(t, "test@example.com", resp.Messages[1].Email)
		assert.NotZero(t, resp.Messages[0].Id)
		assert.NotZero(t, resp.Messages[0].CreatedAt)
	})

	t.Run("GetMessagesByEmail_NoResults", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "messages")

		ctx := context.Background()
		req := &pb.GetMessagesByEmailRequest{
			Email: "nonexistent@example.com",
		}
		resp, err := grpcServer.GetMessagesByEmail(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Messages, 0)
	})

	t.Run("GetMessagesByEmail_EmptyEmail", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "messages")

		ctx := context.Background()
		req := &pb.GetMessagesByEmailRequest{
			Email: "",
		}
		resp, err := grpcServer.GetMessagesByEmail(ctx, req)

		require.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("GetMessagesByEmail_OrderedByCreatedAtDesc", func(t *testing.T) {
		testdb.CleanupTables(t, pgContainer.DB, "messages")

		ctx := context.Background()
		messages := []*message.Message{
			{Email: "test@example.com", Message: "Oldest message"},
			{Email: "test@example.com", Message: "Middle message"},
			{Email: "test@example.com", Message: "Newest message"},
		}

		for _, msg := range messages {
			_, err := pgContainer.DB.NewInsert().Model(msg).Exec(ctx)
			require.NoError(t, err)
		}

		req := &pb.GetMessagesByEmailRequest{
			Email: "test@example.com",
		}
		resp, err := grpcServer.GetMessagesByEmail(ctx, req)

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Messages, 3)

		// Should be ordered by created_at DESC (newest first)
		assert.Equal(t, "Newest message", resp.Messages[0].Message)
		assert.Equal(t, "Middle message", resp.Messages[1].Message)
		assert.Equal(t, "Oldest message", resp.Messages[2].Message)
	})
}
