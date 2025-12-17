package middleware

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// SPAMiddleware handles single-page application routing
// Serves index.html for non-API routes and non-existent files
func SPAMiddleware(distPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip API routes, assets, health check (these should never reach NoRoute anyway)
		if strings.HasPrefix(path, "/api/") ||
			strings.HasPrefix(path, "/assets/") ||
			path == "/health" {
			c.JSON(404, gin.H{"error": "Not found"})
			return
		}

		// Try to serve the file from dist folder
		fullPath := filepath.Join(distPath, path)

		// Check if file exists
		if info, err := os.Stat(fullPath); err == nil {
			// If it's a directory, try to serve index.html from it
			if info.IsDir() {
				indexPath := filepath.Join(fullPath, "index.html")
				if _, err := os.Stat(indexPath); err == nil {
					c.File(indexPath)
					return
				}
			} else {
				// It's a file, serve it
				c.File(fullPath)
				return
			}
		}

		// File doesn't exist - fallback to root index.html for SPA routing
		// This handles client-side routes like /videos, /performers, etc.
		indexPath := filepath.Join(distPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			c.File(indexPath)
			return
		}

		// If dist folder or index.html doesn't exist, return 404
		c.JSON(404, gin.H{
			"error":   "Not found",
			"message": "Frontend files not found. Run 'yarn build' to generate dist folder.",
		})
	}
}

// FrontendCacheMiddleware adds appropriate cache headers for frontend static files
func FrontendCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Get file extension
		ext := filepath.Ext(path)

		switch ext {
		case ".js", ".css":
			// JavaScript and CSS with hash: cache for 1 year (immutable)
			// Files without hash: cache for 1 hour
			if strings.Contains(path, ".") && len(strings.Split(filepath.Base(path), ".")) > 2 {
				// Has hash (e.g., app.abc123.js)
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				// No hash (e.g., app.js)
				c.Header("Cache-Control", "public, max-age=3600")
			}
		case ".woff", ".woff2", ".ttf", ".eot", ".otf":
			// Fonts: cache for 1 year
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".ico":
			// Images: cache for 1 week
			c.Header("Cache-Control", "public, max-age=604800")
		case ".html":
			// HTML files: no cache (always check for updates)
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		default:
			// Other files: cache for 1 hour
			c.Header("Cache-Control", "public, max-age=3600")
		}

		c.Next()
	}
}
