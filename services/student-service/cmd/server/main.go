package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"student-service/internal/app"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	application := app.New()

	go func() {
		if err := application.Run(); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited gracefully")
}
