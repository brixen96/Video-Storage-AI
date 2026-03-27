package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// checkSiteHealth does a HEAD request to a URL and reports reachability
func checkSiteHealth(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "url parameter required"})
		return
	}

	online, statusCode, latencyMs := probeSite(url)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"url":        url,
			"online":     online,
			"status":     statusCode,
			"latency_ms": latencyMs,
		},
	})
}

// probeSite performs a HEAD (fallback GET) request and returns (online, statusCode, latencyMs)
func probeSite(url string) (bool, int, int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, 0, 0
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SiteHealthCheck/1.0)")

	resp, err := client.Do(req)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		return false, 0, latencyMs
	}
	defer resp.Body.Close()

	// HEAD might return 405 on some servers — fall back to GET
	if resp.StatusCode == http.StatusMethodNotAllowed {
		req2, err2 := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err2 != nil {
			return false, resp.StatusCode, latencyMs
		}
		req2.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SiteHealthCheck/1.0)")
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return false, resp.StatusCode, latencyMs
		}
		defer resp2.Body.Close()
		online := resp2.StatusCode < 500
		return online, resp2.StatusCode, time.Since(start).Milliseconds()
	}

	online := resp.StatusCode < 500
	return online, resp.StatusCode, latencyMs
}
