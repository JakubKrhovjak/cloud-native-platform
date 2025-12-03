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
	"student-service/internal/projectclient"
	"student-service/internal/student"

	"grud/common/logger"

	"github.com/gorilla/mux"
)

type App struct {
	config *config.Config
	router *mux.Router
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
		router: mux.NewRouter(),
		logger: slogLogger,
	}

	database := db.New(cfg.Database)

	ctx := context.Background()
	if err := db.RunMigrations(ctx, database, (*student.Student)(nil), (*auth.RefreshToken)(nil)); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

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

	// Create protected subrouter for student endpoints
	protectedRouter := app.router.PathPrefix("/api").Subrouter()
	protectedRouter.Use(auth.AuthMiddleware(slogLogger))
	studentHandler.RegisterRoutes(protectedRouter)

	// Project client endpoints (auth required)
	httpClient := projectclient.NewClient(cfg.ProjectService.BaseURL)

	grpcClient, err := projectclient.NewGrpcClient(cfg.ProjectService.GrpcAddress)
	if err != nil {
		slogLogger.Warn("failed to initialize gRPC client", "error", err)
		grpcClient = nil
	} else {
		slogLogger.Info("gRPC client initialized successfully")
	}

	projectHandler := projectclient.NewHandler(httpClient, grpcClient, slogLogger)
	projectHandler.RegisterRoutes(protectedRouter)

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
