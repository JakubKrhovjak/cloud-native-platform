package projectclient

import (
	"context"
	"fmt"
	"time"

	pb "grud/api/gen/project/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GrpcClient struct {
	conn   *grpc.ClientConn
	client pb.ProjectServiceClient
}

func NewGrpcClient(address string) (*GrpcClient, error) {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	client := pb.NewProjectServiceClient(conn)

	return &GrpcClient{
		conn:   conn,
		client: client,
	}, nil
}

func (c *GrpcClient) GetAllProjects(ctx context.Context) ([]Project, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.GetAllProjects(ctx, &pb.GetAllProjectsRequest{})
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

func (c *GrpcClient) Close() error {
	return c.conn.Close()
}
