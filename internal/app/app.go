package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"grud/internal/config"
	"grud/internal/db"
	"grud/internal/student"

	"github.com/gorilla/mux"
)

type App struct {
	config *config.Config
	router *mux.Router
	server *http.Server
}

func New() *App {
	cfg := config.Load()

	app := &App{
		config: cfg,
		router: mux.NewRouter(),
	}

	database := db.New(cfg.Database)

	ctx := context.Background()
	if err := db.RunMigrations(ctx, database, (*student.Student)(nil)); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	studentRepo := student.NewRepository(database)
	studentService := student.NewService(studentRepo)
	studentHandler := student.NewHandler(studentService)
	studentHandler.RegisterRoutes(app.router)

	return app
}

func (a *App) Run() error {
	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", a.config.Server.Port),
		Handler: a.router,
	}

	log.Printf("Server starting on port %s...", a.config.Server.Port)
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return a.server.Shutdown(ctx)
}
