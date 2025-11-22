package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"grud/internal/config"
	"grud/internal/db"
	"grud/internal/logger"
	"grud/internal/student"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type App struct {
	config *config.Config
	router *mux.Router
	server *http.Server
	logger *zap.Logger
}

func New() *App {
	zapLogger, err := logger.New()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	zapLogger.Info("initializing application")

	cfg := config.Load()

	app := &App{
		config: cfg,
		router: mux.NewRouter(),
		logger: zapLogger,
	}

	database := db.New(cfg.Database)

	ctx := context.Background()
	if err := db.RunMigrations(ctx, database, (*student.Student)(nil)); err != nil {
		zapLogger.Fatal("failed to run migrations", zap.Error(err))
	}

	studentRepo := student.NewRepository(database)
	studentService := student.NewService(studentRepo, zapLogger)
	studentHandler := student.NewHandler(studentService)
	studentHandler.RegisterRoutes(app.router)

	zapLogger.Info("application initialized successfully")

	return app
}

func (a *App) Run() error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", a.config.Server.Port),
		Handler: a.router,
	}

	a.logger.Info("server starting", zap.String("port", a.config.Server.Port))
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.logger.Info("shutting down server")
	defer a.logger.Sync()
	return a.server.Shutdown(ctx)
}
