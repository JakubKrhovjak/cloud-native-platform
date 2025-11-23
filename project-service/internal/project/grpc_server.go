package project

import (
	"context"
	"log/slog"

	pb "grud/api/gen/project/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcServer struct {
	pb.UnimplementedProjectServiceServer
	service Service
	logger  *slog.Logger
}

func NewGrpcServer(service Service, logger *slog.Logger) *GrpcServer {
	return &GrpcServer{
		service: service,
		logger:  logger,
	}
}

func (s *GrpcServer) GetAllProjects(ctx context.Context, req *pb.GetAllProjectsRequest) (*pb.GetAllProjectsResponse, error) {
	s.logger.Info("gRPC: fetching all projects")

	projects, err := s.service.GetAllProjects(ctx)
	if err != nil {
		s.logger.Error("gRPC: failed to fetch projects", "error", err)
		return nil, err
	}

	// Convert internal Project model to protobuf Project
	pbProjects := make([]*pb.Project, len(projects))
	for i, proj := range projects {
		pbProjects[i] = &pb.Project{
			Id:        int32(proj.ID),
			Name:      proj.Name,
			CreatedAt: timestamppb.New(proj.CreatedAt),
			UpdatedAt: timestamppb.New(proj.UpdatedAt),
		}
	}

	return &pb.GetAllProjectsResponse{
		Projects: pbProjects,
	}, nil
}
