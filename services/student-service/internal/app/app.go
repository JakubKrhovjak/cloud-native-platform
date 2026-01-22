package app

import (
	"context"
	"fmt"
	systemLog "log"
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

	"github.com/gin-gonic/gin"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

type App struct {
	config         *config.Config
	router         *gin.Engine
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
	log := logger.NewWithServiceContext(ServiceName, Version)
	slog.SetDefault(log)

	log.Info("initializing application",
		"service", ServiceName,
		"version", Version,
		"commit", GitCommit,
		"build_time", BuildTime,
	)

	cfg, err := config.Load()
	if err != nil {
		systemLog.Fatalf("failed to load config: %v", err)
	}

	log.Info("config loaded", "env", cfg.Env)

	// Initialize OTel telemetry and metrics
	ctx := context.Background()
	telem, _ := telemetry.Init(ctx, ServiceName, Version, cfg.Env, log)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	app := &App{
		config:    cfg,
		router:    router,
		logger:    log,
		telemetry: telem,
	}

	if telem != nil {
		app.metrics = telem.Metrics

		meter := otel.Meter(ServiceName)
		serviceMetrics, err := localmetrics.New(meter)
		if err != nil {
			log.Warn("failed to initialize service metrics", "error", err)
		}
		app.serviceMetrics = serviceMetrics
	}

	database := db.New(cfg.Database)
	app.database = database
	if err := db.RunMigrations(ctx, database, (*student.Student)(nil), (*auth.RefreshToken)(nil)); err != nil {
		systemLog.Fatal("failed to run migrations:", err)
	}

	// Register database for metrics collection
	if app.metrics != nil {
		meter := otel.Meter(ServiceName)
		sqlDB := database.DB
		if err := app.metrics.Database.RegisterDB(sqlDB, meter); err != nil {
			log.Warn("failed to register database metrics", "error", err)
		}

		// Register dependencies for health monitoring
		dependencies := []string{"postgres", "nats", "project-service"}
		if err := app.metrics.Health.RegisterDependencies(ctx, meter, dependencies); err != nil {
			log.Warn("failed to register dependencies", "error", err)
		}
	}

	// Apply CORS middleware globally (origins from config)
	if len(cfg.Server.CORSOrigins) > 0 {
		app.router.Use(middleware.CORS(cfg.Server.CORSOrigins))
	}

	// Apply OTel HTTP instrumentation middleware (if available)
	if telem != nil {
		app.router.Use(func(c *gin.Context) {
			handler := otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Next()
			}), ServiceName)
			handler.ServeHTTP(c.Writer, c.Request)
		})
	}

	// Health endpoints (no auth required)
	healthHandler := health.NewHandler()
	healthHandler.RegisterRoutes(app.router)

	// Auth setup
	studentRepo := student.NewRepository(database, app.metrics)
	authRepo := auth.NewRepository(database, app.metrics)
	authService := auth.NewService(authRepo, studentRepo)
	authHandler := auth.NewHandler(authService, log)
	authHandler.RegisterRoutes(app.router)

	// Student endpoints (auth required)
	studentService := student.NewService(studentRepo)
	studentHandler := student.NewHandler(studentService, log, app.serviceMetrics)

	// Project client endpoints (auth required)
	grpcClient, err := projectclient.NewGrpcClient(cfg.ProjectService.GrpcAddress)
	if err != nil {
		log.Warn("failed to initialize gRPC client", "error", err)
		grpcClient = nil
	} else {
		log.Info("gRPC client initialized successfully")
	}
	app.grpcClient = grpcClient

	projectHandler := projectclient.NewHandler(grpcClient, log, app.serviceMetrics)

	// NATS producer setup
	natsProducer, err := messaging.NewProducer(cfg.NATS.URL, cfg.NATS.Subject, log)
	if err != nil {
		log.Warn("failed to initialize NATS producer", "error", err)
		natsProducer = nil
	} else {
		log.Info("NATS producer initialized successfully")
	}
	app.natsProducer = natsProducer

	// Create protected routes group for /api endpoints
	apiGroup := app.router.Group("/api")
	apiGroup.Use(auth.AuthMiddleware(log))
	studentHandler.RegisterRoutes(apiGroup)
	projectHandler.RegisterRoutes(apiGroup)

	// Message handler (only if NATS is available)
	if natsProducer != nil {
		messageService := message.NewService(natsProducer, log)
		messageHandler := message.NewHandler(messageService, log, app.serviceMetrics)
		messageHandler.RegisterRoutes(apiGroup)
	}

	log.Info("application initialized successfully")

	return app
}

func (a *App) Run() error {
	readTimeout := a.config.Server.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 30
	}

	writeTimeout := a.config.Server.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 30
	}

	idleTimeout := a.config.Server.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = 120
	}

	a.server = &http.Server{
		Addr:         fmt.Sprintf(":%s", a.config.Server.Port),
		Handler:      a.router,
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
		IdleTimeout:  time.Duration(idleTimeout) * time.Second,
	}

	a.logger.Info("server starting",
		"port", a.config.Server.Port,
		"read_timeout_seconds", readTimeout,
		"write_timeout_seconds", writeTimeout,
		"idle_timeout_seconds", idleTimeout,
	)
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

// StartPeriodicLogging logs a message every second
func (a *App) StartPeriodicLogging(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	a.logger.Info("starting periodic logging", "interval", "1s")

	for {
		select {
		case <-ticker.C:
			a.logger.Info("student service periodical log")
		case <-ctx.Done():
			a.logger.Info("stopping periodic logging")
			return
		}
	}
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
