package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"student-service/internal/auth"
	"student-service/internal/config"
	"student-service/internal/db"
	"student-service/internal/health"
	"student-service/internal/message"
	"student-service/internal/messaging"
	"student-service/internal/middleware"
	"student-service/internal/projectclient"
	"student-service/internal/student"

	"grud/common/logger"

	"github.com/go-chi/chi/v5"
)

type App struct {
	config *config.Config
	router chi.Router
	server *http.Server
	logger *slog.Logger
}

func New() *App {
	slogLogger := logger.NewWithServiceContext("student-service", "1.0.0")

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
		router: chi.NewRouter(),
		logger: slogLogger,
	}

	database := db.New(cfg.Database)

	ctx := context.Background()
	if err := db.RunMigrations(ctx, database, (*student.Student)(nil), (*auth.RefreshToken)(nil)); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

	// Apply CORS middleware globally
	app.router.Use(middleware.CORS)

	// Health endpoints (no auth required)
	healthHandler := health.NewHandler()
	healthHandler.RegisterRoutes(app.router)

	// Auth setup
	studentRepo := student.NewRepository(database)
	authRepo := auth.NewRepository(database)
	authService := auth.NewService(authRepo, studentRepo)
	authHandler := auth.NewHandler(authService, slogLogger)
	authHandler.RegisterRoutes(app.router)

	// Student endpoints (auth required)
	studentService := student.NewService(studentRepo)
	studentHandler := student.NewHandler(studentService, slogLogger)

	// Project client endpoints (auth required)
	grpcClient, err := projectclient.NewGrpcClient(cfg.ProjectService.GrpcAddress)
	if err != nil {
		slogLogger.Warn("failed to initialize gRPC client", "error", err)
		grpcClient = nil
	} else {
		slogLogger.Info("gRPC client initialized successfully")
	}

	projectHandler := projectclient.NewHandler(grpcClient, slogLogger)

	// NATS producer setup
	natsProducer, err := messaging.NewProducer(cfg.NATS.URL, cfg.NATS.Subject, slogLogger)
	if err != nil {
		slogLogger.Warn("failed to initialize NATS producer", "error", err)
		natsProducer = nil
	} else {
		slogLogger.Info("NATS producer initialized successfully")
	}

	// Create protected routes group for /api endpoints
	app.router.Route("/api", func(r chi.Router) {
		r.Use(auth.AuthMiddleware(slogLogger))
		studentHandler.RegisterRoutes(r)
		projectHandler.RegisterRoutes(r)

		// Message handler (only if NATS is available)
		if natsProducer != nil {
			messageService := message.NewService(natsProducer, slogLogger)
			messageHandler := message.NewHandler(messageService, slogLogger)
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
	return a.server.Shutdown(ctx)
}
