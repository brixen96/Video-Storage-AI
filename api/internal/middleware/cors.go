package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// allowedWebOrigins are the local frontend origins (Vue dev servers + served dist).
var allowedWebOrigins = []string{
	"http://localhost:3000",
	"http://localhost:8080",
	"http://localhost:8081",
	"http://localhost:8082",
	"http://127.0.0.1:3000",
	"http://127.0.0.1:8080",
	"http://127.0.0.1:8081",
	"http://127.0.0.1:8082",
}

// CORS returns a configured CORS middleware for Vue frontend and the
// Firefox extension bridge. The extension's WebSocket handshake carries an
// Origin of moz-extension://<uuid> - without an explicit allowance here,
// gin-contrib/cors aborts the upgrade with 403 before any handler runs.
func CORS() gin.HandlerFunc {
	config := cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if origin == "" {
				return true // non-browser clients (curl, Go tests) send no Origin
			}
			if strings.HasPrefix(origin, "moz-extension://") {
				return true // VideoStorageAI Companion Bridge extension
			}
			for _, o := range allowedWebOrigins {
				if o == origin {
					return true
				}
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	return cors.New(config)
}
