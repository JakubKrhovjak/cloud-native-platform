package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"student-service/internal/config"
	"student-service/internal/db"
	"student-service/internal/logger"
	"student-service/internal/student"

	"github.com/gorilla/mux"
)

type App struct {
	config *config.Config
	router *mux.Router
	server *http.Server
	logger *slog.Logger
}

func New() *App {
	slogLogger := logger.New()

	slogLogger.Info("initializing application")

	cfg := config.Load()

	app := &App{
		config: cfg,
		router: mux.NewRouter(),
		logger: slogLogger,
	}

	database := db.New(cfg.Database)

	ctx := context.Background()
	if err := db.RunMigrations(ctx, database, (*student.Student)(nil)); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

	studentRepo := student.NewRepository(database)
	studentService := student.NewService(studentRepo)
	studentHandler := student.NewHandler(studentService, slogLogger)
	studentHandler.RegisterRoutes(app.router)

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
