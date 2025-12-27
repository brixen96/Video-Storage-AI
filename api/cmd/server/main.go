package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brixen96/video-storage-ai/internal/api"
	"github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/services"
)

// main is the entry point of the application. It initializes the configuration,
// database, and services, then starts the HTTP server. It also handles
// graceful shutdown on interrupt signals.
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("Failed to close database: %v", err)
		}
	}()

	log.Println("Database initialized successfully")

	// Run startup performer scan
	log.Println("Running startup performer scan...")
	scanService := services.NewPerformerScanService()
	scanResult, err := scanService.ScanPerformerFolders()
	if err != nil {
		log.Printf("Warning: Performer scan failed: %v", err)
	} else {
		log.Printf("Performer scan complete: %d folders found, %d new created, %d existing",
			scanResult.TotalFolders, scanResult.NewCreated, scanResult.Existing)
		if len(scanResult.Errors) > 0 {
			for _, errMsg := range scanResult.Errors {
				log.Printf("  Scan error: %s", errMsg)
			}
		}
	}

	// Setup router
	router := api.SetupRouter(cfg)

	// Initialize WebSocket hub
	hub := api.InitWebSocket()
	services.SetWebSocketHub(hub)

	// Initialize AI Companion Service
	log.Println("Initializing AI Companion...")
	api.InitAICompanion()
	log.Println("AI Companion initialized successfully")

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      api.CacheControlMiddleware(router),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on http://%s:%s", cfg.Server.Host, cfg.Server.Port)
		log.Printf("Mode: %s", cfg.Server.Mode)
		log.Printf("Health check: http://%s:%s/health", cfg.Server.Host, cfg.Server.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop AI Companion
	if companion := api.GetAICompanionService(); companion != nil {
		if err := companion.Stop(); err != nil {
			log.Printf("Error stopping AI Companion: %v", err)
		}
	}

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
