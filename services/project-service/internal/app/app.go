package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"time"

	"project-service/internal/config"
	"project-service/internal/db"
	"project-service/internal/message"
	"project-service/internal/messaging"
	localmetrics "project-service/internal/metrics"
	"project-service/internal/project"

	"grud/common/logger"
	"grud/common/metrics"
	"grud/common/telemetry"

	messagepb "grud/api/gen/message/v1"
	projectpb "grud/api/gen/project/v1"

	"github.com/uptrace/bun"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type App struct {
	config         *config.Config
	grpcServer     *grpc.Server
	natsConsumer   *messaging.Consumer
	database       *bun.DB
	logger         *slog.Logger
	telemetry      *telemetry.Telemetry
	metrics        *metrics.Metrics
	serviceMetrics *localmetrics.Metrics
}

func New() *App {
	slogLogger := logger.NewWithServiceContext(ServiceName, Version)

	// Set as default logger so slog.Info() uses JSON format
	slog.SetDefault(slogLogger)

	slogLogger.Info("initializing application",
		"service", ServiceName,
		"version", Version,
		"commit", GitCommit,
		"build_time", BuildTime,
	)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	slogLogger.Info("config loaded", "env", cfg.Env)

	// Initialize OTel telemetry and metrics
	ctx := context.Background()
	telem, _ := telemetry.Init(ctx, ServiceName, Version, cfg.Env, slogLogger)

	app := &App{
		config:    cfg,
		logger:    slogLogger,
		telemetry: telem,
	}

	if telem != nil {
		app.metrics = telem.Metrics

		meter := otel.Meter(ServiceName)
		serviceMetrics, err := localmetrics.New(meter)
		if err != nil {
			slogLogger.Warn("failed to initialize service metrics", "error", err)
		}
		app.serviceMetrics = serviceMetrics
	}

	database := db.New(cfg.Database)
	app.database = database
	if err := db.RunMigrations(ctx, database, (*project.Project)(nil), (*message.Message)(nil)); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

	// Register database for metrics collection
	if app.metrics != nil {
		meter := otel.Meter(ServiceName)
		sqlDB := database.DB
		if err := app.metrics.Database.RegisterDB(sqlDB, meter); err != nil {
			slogLogger.Warn("failed to register database metrics", "error", err)
		}

		// Register dependencies for health monitoring
		dependencies := []string{"postgres", "nats"}
		if err := app.metrics.Health.RegisterDependencies(ctx, meter, dependencies); err != nil {
			slogLogger.Warn("failed to register dependencies", "error", err)
		}
	}

	projectRepo := project.NewRepository(database, app.metrics)
	projectService := project.NewService(projectRepo)

	messageRepo := message.NewRepository(database, app.metrics)
	messageService := message.NewService(messageRepo)
	natsConsumer, err := messaging.NewConsumer(cfg.NATS.URL, cfg.NATS.Subject, messageRepo, slogLogger, app.serviceMetrics)
	if err != nil {
		log.Fatal("failed to create NATS consumer:", err)
	}
	slogLogger.Info("NATS consumer initialized", "url", cfg.NATS.URL, "subject", cfg.NATS.Subject)

	app.natsConsumer = natsConsumer

	// gRPC Server with OTel instrumentation
	var grpcOpts []grpc.ServerOption
	if telem != nil {
		grpcOpts = append(grpcOpts,
			grpc.StatsHandler(otelgrpc.NewServerHandler()),
		)
	}
	app.grpcServer = grpc.NewServer(grpcOpts...)
	projectGrpcHandler := project.NewGrpcServer(projectService, slogLogger, app.serviceMetrics)
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

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down servers")

	// Shutdown gRPC server
	a.grpcServer.GracefulStop()

	// Close NATS consumer
	if err := a.natsConsumer.Close(); err != nil {
		a.logger.Error("NATS consumer close error", "error", err)
	}

	// Shutdown OTel meter provider
	if a.telemetry != nil && a.telemetry.MeterProvider != nil {
		if err := telemetry.Shutdown(ctx, a.telemetry.MeterProvider, a.logger); err != nil {
			a.logger.Error("failed to shutdown OTel", "error", err)
		}
	}

	return nil
}

// StartHealthChecks periodically checks dependencies and reports status
func (a *App) StartHealthChecks(ctx context.Context) {
	if a.metrics == nil {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	a.logger.Info("starting dependency health checks", "interval", "30s")

	// Run initial check immediately
	a.checkDependencies(ctx)

	for {
		select {
		case <-ticker.C:
			a.checkDependencies(ctx)
		case <-ctx.Done():
			a.logger.Info("stopping dependency health checks")
			return
		}
	}
}

func (a *App) checkDependencies(ctx context.Context) {
	// Check PostgreSQL
	if a.database != nil {
		start := time.Now()
		err := a.database.PingContext(ctx)
		a.metrics.Health.RecordDependencyCheck(ctx, "postgres", time.Since(start), err)
	}

	// Check NATS
	if a.natsConsumer != nil {
		start := time.Now()
		err := a.natsConsumer.HealthCheck()
		a.metrics.Health.RecordDependencyCheck(ctx, "nats", time.Since(start), err)
	}
}
