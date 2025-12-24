package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"student-service/internal/auth"
	"student-service/internal/config"
	"student-service/internal/db"
	"student-service/internal/health"
	"student-service/internal/message"
	"student-service/internal/messaging"
	localmetrics "student-service/internal/metrics"
	"student-service/internal/middleware"
	"student-service/internal/projectclient"
	"student-service/internal/student"

	"grud/common/logger"
	"grud/common/metrics"
	"grud/common/telemetry"

	"github.com/go-chi/chi/v5"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

type App struct {
	config         *config.Config
	router         chi.Router
	server         *http.Server
	logger         *slog.Logger
	telemetry      *telemetry.Telemetry
	metrics        *metrics.Metrics
	serviceMetrics *localmetrics.Metrics
	database       *bun.DB
	natsProducer   *messaging.Producer
	grpcClient     *projectclient.GrpcClient
}

func New() *App {
	slogLogger := logger.NewWithServiceContext(ServiceName, Version)
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
		router:    chi.NewRouter(),
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
	if err := db.RunMigrations(ctx, database, (*student.Student)(nil), (*auth.RefreshToken)(nil)); err != nil {
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
		dependencies := []string{"postgres", "nats", "project-service"}
		if err := app.metrics.Health.RegisterDependencies(ctx, meter, dependencies); err != nil {
			slogLogger.Warn("failed to register dependencies", "error", err)
		}
	}

	// Apply CORS middleware globally
	app.router.Use(middleware.CORS)

	// Apply OTel HTTP instrumentation middleware (if available)
	if telem != nil {
		app.router.Use(func(next http.Handler) http.Handler {
			return otelhttp.NewHandler(next, ServiceName)
		})
	}

	// Health endpoints (no auth required)
	healthHandler := health.NewHandler()
	healthHandler.RegisterRoutes(app.router)

	// Auth setup
	studentRepo := student.NewRepository(database, app.metrics)
	authRepo := auth.NewRepository(database, app.metrics)
	authService := auth.NewService(authRepo, studentRepo)
	authHandler := auth.NewHandler(authService, slogLogger)
	authHandler.RegisterRoutes(app.router)

	// Student endpoints (auth required)
	studentService := student.NewService(studentRepo)
	studentHandler := student.NewHandler(studentService, slogLogger, app.serviceMetrics)

	// Project client endpoints (auth required)
	grpcClient, err := projectclient.NewGrpcClient(cfg.ProjectService.GrpcAddress)
	if err != nil {
		slogLogger.Warn("failed to initialize gRPC client", "error", err)
		grpcClient = nil
	} else {
		slogLogger.Info("gRPC client initialized successfully")
	}
	app.grpcClient = grpcClient

	projectHandler := projectclient.NewHandler(grpcClient, slogLogger, app.serviceMetrics)

	// NATS producer setup
	natsProducer, err := messaging.NewProducer(cfg.NATS.URL, cfg.NATS.Subject, slogLogger)
	if err != nil {
		slogLogger.Warn("failed to initialize NATS producer", "error", err)
		natsProducer = nil
	} else {
		slogLogger.Info("NATS producer initialized successfully")
	}
	app.natsProducer = natsProducer

	// Create protected routes group for /api endpoints
	app.router.Route("/api", func(r chi.Router) {
		r.Use(auth.AuthMiddleware(slogLogger))
		studentHandler.RegisterRoutes(r)
		projectHandler.RegisterRoutes(r)

		// Message handler (only if NATS is available)
		if natsProducer != nil {
			messageService := message.NewService(natsProducer, slogLogger)
			messageHandler := message.NewHandler(messageService, slogLogger, app.serviceMetrics)
			messageHandler.RegisterRoutes(r)
		}
	})

	slogLogger.Info("application initialized successfully")

	return app
}

func (a *App) Run() error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", a.config.Server.Port),
		Handler: a.router,
	}

	a.logger.Info("server starting", "port", a.config.Server.Port)
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down server")

	// Shutdown HTTP server
	if err := a.server.Shutdown(ctx); err != nil {
		return err
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
	if a.natsProducer != nil {
		start := time.Now()
		// NATS doesn't have a direct Ping, but we can check if connection exists
		err := a.natsProducer.HealthCheck()
		a.metrics.Health.RecordDependencyCheck(ctx, "nats", time.Since(start), err)
	}

	// Check gRPC (project-service)
	if a.grpcClient != nil {
		start := time.Now()
		err := a.grpcClient.HealthCheck(ctx)
		a.metrics.Health.RecordDependencyCheck(ctx, "project-service", time.Since(start), err)
	}
}
