package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// Verifies the extension bridge origin (moz-extension://) passes CORS while
// unknown origins are still rejected. Regression for the 403 WS handshake.
func TestCORSOriginPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	cases := []struct {
		origin string
		want   int
	}{
		{"moz-extension://d3ac6365-05ea-4b01-9bd8-ad4b62ea5bbd", http.StatusOK},
		{"http://localhost:3000", http.StatusOK},
		{"http://127.0.0.1:8080", http.StatusOK},
		{"", http.StatusOK},
		{"http://evil.example.com", http.StatusForbidden},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("origin %q: got status %d, want %d", tc.origin, w.Code, tc.want)
		}
	}
}