package main

import (
	"context"
	systemLog "log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project-service/internal/app"

	"github.com/rs/zerolog/log"
)

func main() {
	// Initialize application with gRPC on port 50052
	application := app.New()

	// Start dependency health checks in background
	healthCtx, healthCancel := context.WithCancel(context.Background())
	defer healthCancel()
	go application.StartHealthChecks(healthCtx)

	go func() {
		if err := application.Run(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		systemLog.Fatal("Server forced to shutdown:", err)
	}

	systemLog.Println("Server exited gracefully")
}
