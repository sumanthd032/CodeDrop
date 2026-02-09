package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sumanthd032/codedrop/internal/api"
	"github.com/sumanthd032/codedrop/internal/db"
	"github.com/sumanthd032/codedrop/internal/store"
)

func main() {
	// Initialize Database
	database, err := db.NewConnection()
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	if err := database.Migrate(); err != nil {
		log.Fatalf("Could not apply migrations: %v", err)
	}

	// Initialize Storage
	st, err := store.NewS3Store()
	if err != nil {
		log.Fatalf("Could not connect to storage: %v", err)
	}

	// Initialize API Server
	srv := api.NewServer(database, st)

	// Configure HTTP Server
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: srv.Router,
	}

	// Start Server in a Goroutine (Background)
	go func() {
		log.Println("CodeDrop Server listening on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	// Graceful Shutdown Logic
	// Wait for interrupt signal (Ctrl+C or Kubernetes SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block here until signal received

	log.Println("Shutting down server...")

	// Create a context with a 5-second timeout to allow active requests to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}