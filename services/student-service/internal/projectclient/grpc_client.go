package projectclient

import (
	"context"
	"fmt"
	"time"

	messagepb "grud/api/gen/message/v1"
	projectpb "grud/api/gen/project/v1"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcClient struct {
	conn          *grpc.ClientConn
	projectClient projectpb.ProjectServiceClient
	messageClient messagepb.MessageServiceClient
}

func NewGrpcClient(address string) (*GrpcClient, error) {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return &GrpcClient{
		conn:          conn,
		projectClient: projectpb.NewProjectServiceClient(conn),
		messageClient: messagepb.NewMessageServiceClient(conn),
	}, nil
}

func (c *GrpcClient) GetAllProjects(ctx context.Context) ([]Project, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.projectClient.GetAllProjects(ctx, &projectpb.GetAllProjectsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to call GetAllProjects: %w", err)
	}

	projects := make([]Project, len(resp.Projects))
	for i, pbProj := range resp.Projects {
		projects[i] = Project{
			ID:        int(pbProj.Id),
			Name:      pbProj.Name,
			CreatedAt: pbProj.CreatedAt.AsTime(),
			UpdatedAt: pbProj.UpdatedAt.AsTime(),
		}
	}

	return projects, nil
}

func (c *GrpcClient) GetMessagesByEmail(ctx context.Context, email string) ([]Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.messageClient.GetMessagesByEmail(ctx, &messagepb.GetMessagesByEmailRequest{
		Email: email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call GetMessagesByEmail: %w", err)
	}

	messages := make([]Message, len(resp.Messages))
	for i, pbMsg := range resp.Messages {
		messages[i] = Message{
			ID:        int(pbMsg.Id),
			Email:     pbMsg.Email,
			Message:   pbMsg.Message,
			CreatedAt: pbMsg.CreatedAt.AsTime(),
		}
	}

	return messages, nil
}

func (c *GrpcClient) Close() error {
	return c.conn.Close()
}

// HealthCheck verifies gRPC connection is healthy
func (c *GrpcClient) HealthCheck(ctx context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("grpc connection is nil")
	}

	// Try a simple RPC call with short timeout to verify connectivity
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.projectClient.GetAllProjects(ctx, &projectpb.GetAllProjectsRequest{})
	return err
}
