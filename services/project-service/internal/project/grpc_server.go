package project

import (
	"context"
	"log/slog"

	pb "grud/api/gen/project/v1"
	"project-service/internal/metrics"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcServer struct {
	pb.UnimplementedProjectServiceServer
	service Service
	logger  *slog.Logger
	metrics *metrics.Metrics
}

func NewGrpcServer(service Service, logger *slog.Logger, metrics *metrics.Metrics) *GrpcServer {
	return &GrpcServer{
		service: service,
		logger:  logger,
		metrics: metrics,
	}
}

func (s *GrpcServer) GetAllProjects(ctx context.Context, req *pb.GetAllProjectsRequest) (*pb.GetAllProjectsResponse, error) {
	s.logger.InfoContext(ctx, "gRPC: fetching all projects")

	projects, err := s.service.GetAllProjects(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "gRPC: failed to fetch projects", "error", err)
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

	// Record metric
	s.metrics.RecordProjectsListViewed(ctx)

	return &pb.GetAllProjectsResponse{
		Projects: pbProjects,
	}, nil
}

func (s *GrpcServer) GetProject(ctx context.Context, req *pb.GetProjectRequest) (*pb.GetProjectResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be greater than 0")
	}

	s.logger.InfoContext(ctx, "gRPC: fetching project by ID", "id", req.Id)

	project, err := s.service.GetProjectByID(ctx, int(req.Id))
	if err != nil {
		s.logger.ErrorContext(ctx, "gRPC: failed to fetch project", "error", err, "id", req.Id)
		return nil, err
	}

	// Record metric
	s.metrics.RecordProjectViewed(ctx)

	return &pb.GetProjectResponse{
		Project: &pb.Project{
			Id:        int32(project.ID),
			Name:      project.Name,
			CreatedAt: timestamppb.New(project.CreatedAt),
			UpdatedAt: timestamppb.New(project.UpdatedAt),
		},
	}, nil
}

func (s *GrpcServer) CreateProject(ctx context.Context, req *pb.CreateProjectRequest) (*pb.CreateProjectResponse, error) {
	s.logger.InfoContext(ctx, "gRPC: creating project", "name", req.Name)

	project := &Project{
		Name: req.Name,
	}

	if err := s.service.CreateProject(ctx, project); err != nil {
		s.logger.ErrorContext(ctx, "gRPC: failed to create project", "error", err)
		return nil, err
	}

	// Record metric
	s.metrics.RecordProjectCreation(ctx)

	return &pb.CreateProjectResponse{
		Project: &pb.Project{
			Id:        int32(project.ID),
			Name:      project.Name,
			CreatedAt: timestamppb.New(project.CreatedAt),
			UpdatedAt: timestamppb.New(project.UpdatedAt),
		},
	}, nil
}

func (s *GrpcServer) UpdateProject(ctx context.Context, req *pb.UpdateProjectRequest) (*pb.UpdateProjectResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be greater than 0")
	}

	s.logger.InfoContext(ctx, "gRPC: updating project", "id", req.Id, "name", req.Name)

	project := &Project{
		ID:   int(req.Id),
		Name: req.Name,
	}

	if err := s.service.UpdateProject(ctx, project); err != nil {
		s.logger.ErrorContext(ctx, "gRPC: failed to update project", "error", err, "id", req.Id)
		return nil, err
	}

	// Fetch updated project to get all fields including timestamps
	updatedProject, err := s.service.GetProjectByID(ctx, int(req.Id))
	if err != nil {
		s.logger.ErrorContext(ctx, "gRPC: failed to fetch updated project", "error", err, "id", req.Id)
		return nil, err
	}

	return &pb.UpdateProjectResponse{
		Project: &pb.Project{
			Id:        int32(updatedProject.ID),
			Name:      updatedProject.Name,
			CreatedAt: timestamppb.New(updatedProject.CreatedAt),
			UpdatedAt: timestamppb.New(updatedProject.UpdatedAt),
		},
	}, nil
}

func (s *GrpcServer) DeleteProject(ctx context.Context, req *pb.DeleteProjectRequest) (*pb.DeleteProjectResponse, error) {
	if req.Id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be greater than 0")
	}

	s.logger.InfoContext(ctx, "gRPC: deleting project", "id", req.Id)

	if err := s.service.DeleteProject(ctx, int(req.Id)); err != nil {
		s.logger.ErrorContext(ctx, "gRPC: failed to delete project", "error", err, "id", req.Id)
		return nil, err
	}

	return &pb.DeleteProjectResponse{}, nil
}
