package middleware

import (
	"fmt"
	"time"

	"github.com/brixen96/video-storage-ai/internal/logger"
	"github.com/gin-gonic/gin"
)

// Logger returns a gin middleware for logging requests
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request details
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Format path with query string
		if raw != "" {
			path = path + "?" + raw
		}

		// Log to console with custom format
		fmt.Printf("[%s] | %s | %3d | %13v | %15s | %-7s %s %s\n",
			time.Now().Format("2006/01/02 - 15:04:05"),
			c.Request.Proto,
			statusCode,
			latency,
			clientIP,
			method,
			path,
			errorMessage,
		)

		// Log to database via custom logger
		logger.LogHTTPRequest(method, path, statusCode, latency.String(), clientIP, errorMessage)
	}
}

// Recovery returns a gin middleware that recovers from panics
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.JSON(500, gin.H{
				"error":     "Internal Server Error",
				"message":   err,
				"timestamp": time.Now().Format(time.RFC3339),
			})
		}
		c.AbortWithStatus(500)
	})
}
