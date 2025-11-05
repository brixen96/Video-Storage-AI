package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

// CORS returns a configured CORS middleware for Vue frontend
func CORS() gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:8081", "http://localhost:8082", "http://localhost:8080", "http://127.0.0.1:8081", "http://127.0.0.1:8082"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	return cors.New(config)
}
