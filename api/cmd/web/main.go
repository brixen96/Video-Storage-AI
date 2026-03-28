package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/config"
	"github.com/kardianos/service"
)

// webProgram implements service.Interface for the web/frontend server.
type webProgram struct {
	server *http.Server
}

func (p *webProgram) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *webProgram) Stop(s service.Service) error {
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := p.server.Shutdown(ctx); err != nil {
			log.Printf("Web server forced to shutdown: %v", err)
		}
	}
	log.Println("Web server exited")
	return nil
}

func (p *webProgram) run() {
	// Load config — sets up AppData dir and reads config.json
	if _, err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	userCfg := config.GetUserConfig()
	apiURL := fmt.Sprintf("http://localhost:%s", userCfg.ServerPort)
	webPort := userCfg.WebPort

	// Locate dist/ next to the executable (production layout)
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Cannot determine executable path: %v", err)
	}
	distPath := filepath.Join(filepath.Dir(exePath), "dist")

	// Fallback to ./dist for development (running via 'go run')
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		distPath = filepath.Join(".", "dist")
	}

	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		log.Fatalf("Frontend dist folder not found at %s — run 'npm run build' first", distPath)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveSPA(w, r, distPath, apiURL)
	})

	addr := fmt.Sprintf(":%s", webPort)
	p.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Video Storage AI — Web Server on http://localhost%s", addr)
	log.Printf("API backend: %s", apiURL)
	log.Printf("Serving: %s", distPath)

	if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Web server failed: %v", err)
	}
}

func main() {
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)

	svcConfig := &service.Config{
		Name:             "VideoStorageAI-Web",
		DisplayName:      "Video Storage AI - Web Server",
		Description:      "Video Storage AI frontend web server (port 3000 by default).",
		WorkingDirectory: exeDir,
		Option: service.KeyValue{
			"StartType":              "automatic",
			"OnFailure":              "restart",
			"OnFailureDelayDuration": "5s",
			"OnFailureResetPeriod":   10,
		},
	}

	prg := &webProgram{}
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

	if err := s.Run(); err != nil {
		log.Fatalf("Service run failed: %v", err)
	}
}

// serveSPA handles all HTTP requests:
//   - / or /index.html or unknown routes → inject VSAI_API_URL and serve index.html
//   - real files → served directly from disk
func serveSPA(w http.ResponseWriter, r *http.Request, distPath, apiURL string) {
	urlPath := r.URL.Path

	if urlPath == "/" || urlPath == "/index.html" {
		serveIndex(w, distPath, apiURL)
		return
	}

	filePath := filepath.Join(distPath, filepath.FromSlash(urlPath))
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		// No matching file → SPA client-side route, serve index.html
		serveIndex(w, distPath, apiURL)
		return
	}

	http.ServeFile(w, r, filePath)
}

// serveIndex reads index.html, injects window.VSAI_API_URL, and writes it to the response.
func serveIndex(w http.ResponseWriter, distPath, apiURL string) {
	data, err := os.ReadFile(filepath.Join(distPath, "index.html"))
	if err != nil {
		http.Error(w, "Frontend not ready — run 'npm run build' first", http.StatusServiceUnavailable)
		return
	}

	script := fmt.Sprintf(`<script>window.VSAI_API_URL = "%s";</script>`, apiURL)
	html := strings.Replace(string(data), "</head>", script+"\n</head>", 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	fmt.Fprint(w, html)
}
