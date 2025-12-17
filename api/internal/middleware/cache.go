package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticFileCache adds cache headers for static files
func StaticFileCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the file extension
		path := c.Request.URL.Path
		ext := strings.ToLower(path[strings.LastIndex(path, "."):])

		// Set cache headers based on file type
		switch ext {
		case ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".webm", ".m4v":
			// Video files: cache for 7 days (604800 seconds)
			// These don't change often, so we can cache aggressively
			c.Header("Cache-Control", "public, max-age=604800, immutable")
			c.Header("Expires", "")
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg":
			// Image files: cache for 7 days
			c.Header("Cache-Control", "public, max-age=604800, immutable")
			c.Header("Expires", "")
		case ".js", ".css":
			// JavaScript and CSS: cache for 1 hour (in case of updates)
			c.Header("Cache-Control", "public, max-age=3600")
			c.Header("Expires", "")
		default:
			// Other files: cache for 1 hour
			c.Header("Cache-Control", "public, max-age=3600")
		}

		c.Next()
	}
}
