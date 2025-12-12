package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"

	"project-service/internal/config"
	"project-service/internal/db"
	"project-service/internal/kafka"
	"project-service/internal/message"
	"project-service/internal/project"

	"grud/common/logger"

	messagepb "grud/api/gen/message/v1"
	projectpb "grud/api/gen/project/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type App struct {
	config        *config.Config
	grpcServer    *grpc.Server
	kafkaConsumer *kafka.Consumer
	logger        *slog.Logger
}

func New() *App {
	slogLogger := logger.NewWithServiceContext("project-service", "1.0.0")

	// Set as default logger so slog.Info() uses JSON format
	slog.SetDefault(slogLogger)

	slogLogger.Info("initializing application")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	slogLogger.Info("config loaded", "env", cfg.Env)

	app := &App{
		config: cfg,
		logger: slogLogger,
	}

	database := db.New(cfg.Database)

	ctx := context.Background()
	if err := db.RunMigrations(ctx, database, (*project.Project)(nil), (*message.Message)(nil)); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

	projectRepo := project.NewRepository(database)
	projectService := project.NewService(projectRepo)

	// Initialize message repository, service and Kafka consumer
	messageRepo := message.NewRepository(database)
	messageService := message.NewService(messageRepo)
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, messageRepo, slogLogger)
	if err != nil {
		log.Fatal("failed to create Kafka consumer:", err)
	}
	slogLogger.Info("kafka consumer initialized", "brokers", cfg.Kafka.Brokers, "topic", cfg.Kafka.Topic)

	app.kafkaConsumer = kafkaConsumer

	// gRPC Server
	app.grpcServer = grpc.NewServer()
	projectGrpcHandler := project.NewGrpcServer(projectService, slogLogger)
	projectpb.RegisterProjectServiceServer(app.grpcServer, projectGrpcHandler)

	messageGrpcHandler := message.NewGrpcServer(messageService, slogLogger)
	messagepb.RegisterMessageServiceServer(app.grpcServer, messageGrpcHandler)

	// Register gRPC health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(app.grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("project.v1.ProjectService", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("message.v1.MessageService", grpc_health_v1.HealthCheckResponse_SERVING)

	slogLogger.Info("application initialized successfully")

	return app
}

func (a *App) Run() error {
	// Start Kafka consumer
	go func() {
		a.logger.Info("Kafka consumer starting", "topic", a.config.Kafka.Topic)
		ctx := context.Background()
		if err := a.kafkaConsumer.Start(ctx); err != nil {
			a.logger.Error("Kafka consumer error", "error", err)
		}
	}()

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", a.config.Grpc.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port: %w", err)
	}

	a.logger.Info("gRPC server starting", "port", a.config.Grpc.Port)
	return a.grpcServer.Serve(lis)
}

func (a *App) Shutdown(_ context.Context) error {
	a.logger.Info("shutting down servers")

	// Shutdown gRPC server
	a.grpcServer.GracefulStop()

	// Close Kafka consumer
	if err := a.kafkaConsumer.Close(); err != nil {
		a.logger.Error("Kafka consumer close error", "error", err)
	}

	return nil
}
