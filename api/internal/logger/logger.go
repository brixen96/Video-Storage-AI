package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/brixen96/video-storage-ai/internal/services"
)

var (
	consoleLogService *services.ConsoleLogService
	loggerMutex       sync.Mutex
)

// ConsoleWriter is a custom io.Writer that captures log output and stores it in the database
type ConsoleWriter struct {
	originalWriter io.Writer
}

// Write implements io.Writer interface
func (cw *ConsoleWriter) Write(p []byte) (n int, err error) {
	// Write to original output (stdout)
	n, err = cw.originalWriter.Write(p)

	// Also store in database if service is available
	if consoleLogService != nil {
		message := strings.TrimSpace(string(p))
		if message != "" {
			// Determine log level based on message content
			level := "info"
			if strings.Contains(strings.ToLower(message), "error") || strings.Contains(strings.ToLower(message), "failed") {
				level = "error"
			} else if strings.Contains(strings.ToLower(message), "warn") || strings.Contains(strings.ToLower(message), "warning") {
				level = "warning"
			}

			// Store in database (non-blocking to avoid slowdown)
			go func() {
				loggerMutex.Lock()
				defer loggerMutex.Unlock()
				_ = consoleLogService.LogAPI(level, message, nil)
			}()
		}
	}

	return n, err
}

// SetupLogger initializes the custom logger that captures all output
func SetupLogger(service *services.ConsoleLogService) {
	consoleLogService = service

	// Create custom writer that captures output
	writer := &ConsoleWriter{
		originalWriter: os.Stdout,
	}

	// Set the log package to use our custom writer
	log.SetOutput(writer)
	log.SetFlags(log.Ldate | log.Ltime)
}

// LogAPI logs an API-related message directly
func LogAPI(level, message string, details map[string]interface{}) {
	if consoleLogService != nil {
		loggerMutex.Lock()
		defer loggerMutex.Unlock()
		_ = consoleLogService.LogAPI(level, message, details)
	}
	// Also print to console
	log.Printf("[%s] %s", strings.ToUpper(level), message)
}

// LogHTTPRequest logs an HTTP request
func LogHTTPRequest(method, path string, statusCode int, latency string, clientIP string, errorMsg string) {
	message := fmt.Sprintf("%s %s - %d - %s - %s", method, path, statusCode, latency, clientIP)
	if errorMsg != "" {
		message += fmt.Sprintf(" - Error: %s", errorMsg)
	}

	details := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"latency":     latency,
		"client_ip":   clientIP,
	}
	if errorMsg != "" {
		details["error"] = errorMsg
	}

	level := "info"
	if statusCode >= 500 {
		level = "error"
	} else if statusCode >= 400 {
		level = "warning"
	}

	if consoleLogService != nil {
		loggerMutex.Lock()
		defer loggerMutex.Unlock()
		_ = consoleLogService.LogAPI(level, message, details)
	}
}
