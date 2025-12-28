package message

import (
	"context"
	"log/slog"

	pb "grud/api/gen/message/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcServer struct {
	pb.UnimplementedMessageServiceServer
	service Service
	logger  *slog.Logger
}

func NewGrpcServer(service Service, logger *slog.Logger) *GrpcServer {
	return &GrpcServer{
		service: service,
		logger:  logger,
	}
}

func (s *GrpcServer) GetMessagesByEmail(ctx context.Context, req *pb.GetMessagesByEmailRequest) (*pb.GetMessagesByEmailResponse, error) {
	s.logger.InfoContext(ctx, "gRPC: fetching messages by email", "email", req.Email)

	messages, err := s.service.GetMessagesByEmail(ctx, req.Email)
	if err != nil {
		s.logger.ErrorContext(ctx, "gRPC: failed to fetch messages by email", "error", err, "email", req.Email)
		return nil, err
	}

	pbMessages := make([]*pb.Message, len(messages))
	for i, msg := range messages {
		pbMessages[i] = &pb.Message{
			Id:        int32(msg.ID),
			Email:     msg.Email,
			Message:   msg.Message,
			CreatedAt: timestamppb.New(msg.CreatedAt),
		}
	}

	return &pb.GetMessagesByEmailResponse{
		Messages: pbMessages,
	}, nil
}
