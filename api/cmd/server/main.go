package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/brixen96/video-storage-ai/internal/api"
	"github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/logger"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/kardianos/service"
)

// apiProgram implements service.Interface for the API server.
type apiProgram struct {
	server *http.Server
	stopCh chan struct{}
}

func (p *apiProgram) Start(s service.Service) error {
	p.stopCh = make(chan struct{})
	go p.run()
	return nil
}

func (p *apiProgram) Stop(s service.Service) error {
	// Signal the run goroutine to begin shutdown
	if p.stopCh != nil {
		close(p.stopCh)
	}

	// Stop AI Companion
	if companion := api.GetAICompanionService(); companion != nil {
		if err := companion.Stop(); err != nil {
			log.Printf("Error stopping AI Companion: %v", err)
		}
	}

	// Graceful HTTP shutdown
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.server.Shutdown(ctx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}
	}

	log.Println("Server exited")
	return nil
}

func (p *apiProgram) run() {
	// Load configuration (also resolves and creates the AppData base dir)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Ensure all required subdirectories exist under AppData
	for _, dir := range []string{
		config.DataDir(),
		cfg.Paths.ThumbnailDir,
		cfg.Paths.PerformerDir,
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Warning: could not create directory %s: %v", dir, err)
		}
	}
	log.Printf("Data directory: %s", config.DataDir())

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

	// Initialize custom logger to capture all console output
	consoleLogService := services.NewConsoleLogService()
	logger.SetupLogger(consoleLogService)
	log.Println("Console logger initialized - all logs will be captured")

	// Check for paused tasks and notify
	activityService := services.NewActivityService()
	pausedActivities, err := activityService.GetAll("running", "", 100)
	if err == nil {
		pausedCount := 0
		for _, activity := range pausedActivities {
			if activity.IsPaused {
				pausedCount++
				log.Printf("Found paused task: [%s] %s (ID: %d)", activity.TaskType, activity.Message, activity.ID)
			}
		}
		if pausedCount > 0 {
			log.Printf("Found %d paused task(s). You can resume them from the Activity page.", pausedCount)
		}
	}

	// Run startup performer scan
	log.Println("Running startup performer scan...")
	scanService := services.NewPerformerScanService()
	scanResult, err := scanService.ScanPerformerFolders()
	if err != nil {
		log.Printf("Warning: Performer scan failed: %v", err)
	} else {
		log.Printf("Performer scan complete: %d folders found, %d new created, %d existing",
			scanResult.TotalFolders, scanResult.NewCreated, scanResult.Existing)
		for _, errMsg := range scanResult.Errors {
			log.Printf("  Scan error: %s", errMsg)
		}
	}

	// Verify performer categories after startup
	performerSvc := services.NewPerformerService()
	allPerformers, err := performerSvc.GetAll()
	if err == nil {
		catCounts := map[string]int{}
		missingThumbnails := 0
		for _, p := range allPerformers {
			catCounts[p.Category]++
			if p.PreviewPath.Valid && p.PreviewPath.String != "" && (!p.ThumbnailPath.Valid || p.ThumbnailPath.String == "") {
				missingThumbnails++
			}
		}
		log.Printf("Performer categories: %v", catCounts)

		// Auto-generate thumbnails for performers missing them (async, non-blocking)
		if missingThumbnails > 0 {
			log.Printf("Found %d performers with previews but no thumbnails - generating in background...", missingThumbnails)
			mediaSvc := services.NewMediaService()
			thumbSvc := services.NewPerformerThumbnailService(mediaSvc, activityService)
			go func() {
				if err := thumbSvc.GenerateAllThumbnails(); err != nil {
					log.Printf("Warning: Auto thumbnail generation failed: %v", err)
				}
			}()
		}
	}

	// Setup router
	router := api.SetupRouter(cfg)

	// Initialize WebSocket hub
	hub := api.InitWebSocket()
	services.SetWebSocketHub(hub)
	services.SetNotificationWebSocketHub(hub)

	// Initialize AI Companion Service
	log.Println("Initializing AI Companion...")
	companionSvc := api.InitAICompanion()
	log.Println("AI Companion initialized successfully")

	// Bridge AI Companion events to WebSocket for proactive messages
	if companionSvc != nil {
		eventChan, _ := companionSvc.SubscribeToEvents()
		go func() {
			for event := range eventChan {
				hub.BroadcastCompanionEvent(event.Type, event.Source, event.Message, event.Severity, event.Data)
			}
		}()
		log.Println("AI Companion events bridged to WebSocket")
	}

	// Initialize and start the scheduler (auto-update scrapers, scheduled jobs)
	api.InitScheduler()

	// Create HTTP server
	p.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      api.CacheControlMiddleware(router),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	log.Printf("Starting server on http://%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Mode: %s", cfg.Server.Mode)

	if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func main() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	svcConfig := &service.Config{
		Name:             "VideoStorageAI-API",
		DisplayName:      "Video Storage AI - API Server",
		Description:      "Video Storage AI backend API server (port 8080 by default).",
		WorkingDirectory: exeDir,
		Option: service.KeyValue{
			"StartType":              "automatic",
			"OnFailure":              "restart",
			"OnFailureDelayDuration": "5s",
			"OnFailureResetPeriod":   10,
		},
	}

	prg := &apiProgram{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	// Handle service control commands: install, uninstall, start, stop, restart
	if len(os.Args) > 1 {
		cmd := os.Args[1]
		switch cmd {
		case "install", "uninstall", "start", "stop", "restart":
			if err := service.Control(s, cmd); err != nil {
				log.Fatalf("Service %s failed: %v", cmd, err)
			}
			log.Printf("Service %s succeeded.", cmd)
			return
		case "status":
			status, err := s.Status()
			if err != nil {
				log.Fatalf("Could not get service status: %v", err)
			}
			switch status {
			case service.StatusRunning:
				log.Println("Service status: Running")
			case service.StatusStopped:
				log.Println("Service status: Stopped")
			default:
				log.Println("Service status: Unknown")
			}
			return
		}
	}

	// Run as service (or interactively if not invoked by SCM)
	if err := s.Run(); err != nil {
		log.Fatalf("Service run failed: %v", err)
	}
}
