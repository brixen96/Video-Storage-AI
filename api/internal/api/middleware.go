package api

import (
	"net/http"
	"strings"
)

// CacheControlMiddleware adds cache headers for static assets.
func CacheControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is for a static asset that can be cached.
		// You can customize this logic based on your URL structure.
		if strings.HasPrefix(r.URL.Path, "/api/assets/") ||
			strings.HasPrefix(r.URL.Path, "/api/thumbnails/") ||
			strings.HasSuffix(r.URL.Path, ".jpg") ||
			strings.HasSuffix(r.URL.Path, ".png") ||
			strings.HasSuffix(r.URL.Path, ".mkv") ||
			strings.HasPrefix(r.URL.Path, "/js/") ||
			strings.HasPrefix(r.URL.Path, "/css/") ||
			r.URL.Path == "/favicon.ico" {

			// Cache for 1 year
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}