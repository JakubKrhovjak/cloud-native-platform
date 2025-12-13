package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"

	"project-service/internal/config"
	"project-service/internal/db"
	"project-service/internal/message"
	"project-service/internal/messaging"
	"project-service/internal/project"

	"grud/common/logger"

	messagepb "grud/api/gen/message/v1"
	projectpb "grud/api/gen/project/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type App struct {
	config       *config.Config
	grpcServer   *grpc.Server
	natsConsumer *messaging.Consumer
	logger       *slog.Logger
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

	messageRepo := message.NewRepository(database)
	messageService := message.NewService(messageRepo)
	natsConsumer, err := messaging.NewConsumer(cfg.NATS.URL, cfg.NATS.Subject, messageRepo, slogLogger)
	if err != nil {
		log.Fatal("failed to create NATS consumer:", err)
	}
	slogLogger.Info("NATS consumer initialized", "url", cfg.NATS.URL, "subject", cfg.NATS.Subject)

	app.natsConsumer = natsConsumer

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
	// Start NATS consumer
	go func() {
		a.logger.Info("NATS consumer starting", "subject", a.config.NATS.Subject)
		ctx := context.Background()
		if err := a.natsConsumer.Start(ctx); err != nil {
			a.logger.Error("NATS consumer error", "error", err)
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

	// Close NATS consumer
	if err := a.natsConsumer.Close(); err != nil {
		a.logger.Error("NATS consumer close error", "error", err)
	}

	return nil
}
