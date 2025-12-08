package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"

	"project-service/internal/config"
	"project-service/internal/db"
	"project-service/internal/health"
	"project-service/internal/kafka"
	"project-service/internal/message"
	"project-service/internal/project"

	"grud/common/logger"

	pb "grud/api/gen/project/v1"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
)

type App struct {
	config        *config.Config
	router        *mux.Router
	httpServer    *http.Server
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
		router: mux.NewRouter(),
		logger: slogLogger,
	}

	database := db.New(cfg.Database)

	ctx := context.Background()
	if err := db.RunMigrations(ctx, database, (*project.Project)(nil), (*message.Message)(nil)); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

	projectRepo := project.NewRepository(database)
	projectService := project.NewService(projectRepo)

	// Initialize message repository and Kafka consumer
	messageRepo := message.NewRepository(database)
	kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, messageRepo, slogLogger)
	if err != nil {
		log.Fatal("failed to create Kafka consumer:", err)
	}
	slogLogger.Info("kafka consumer initialized", "brokers", cfg.Kafka.Brokers, "topic", cfg.Kafka.Topic)

	app.kafkaConsumer = kafkaConsumer

	// Health endpoints
	healthHandler := health.NewHandler()
	healthHandler.RegisterRoutes(app.router)

	// HTTP Handler
	projectHandler := project.NewHandler(projectService, slogLogger)
	projectHandler.RegisterRoutes(app.router)

	// gRPC Server
	app.grpcServer = grpc.NewServer()
	grpcHandler := project.NewGrpcServer(projectService, slogLogger)
	pb.RegisterProjectServiceServer(app.grpcServer, grpcHandler)

	slogLogger.Info("application initialized successfully")

	return app
}

func (a *App) Run() error {
	// Start HTTP server
	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", a.config.Server.Port),
		Handler: a.router,
	}

	go func() {
		a.logger.Info("HTTP server starting", "port", a.config.Server.Port)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server failed:", err)
		}
	}()

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

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down servers")

	// Shutdown HTTP server
	if err := a.httpServer.Shutdown(ctx); err != nil {
		a.logger.Error("HTTP server shutdown error", "error", err)
	}

	// Shutdown gRPC server
	a.grpcServer.GracefulStop()

	// Close Kafka consumer
	if err := a.kafkaConsumer.Close(); err != nil {
		a.logger.Error("Kafka consumer close error", "error", err)
	}

	return nil
}
