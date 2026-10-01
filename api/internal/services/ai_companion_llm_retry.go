package services

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// maxRetries is the maximum number of retry attempts for transient failures.
const maxRetries = 3

// isRetryableStatus returns true for HTTP status codes that are safe to retry.
func isRetryableStatus(code int) bool {
	return code == 429 || code == 500 || code == 502 || code == 503 || code == 504
}

// retryDelay computes an exponential back-off delay with jitter.
// attempt is zero-indexed (0 = first retry).
func retryDelay(attempt int) time.Duration {
	base := time.Duration(math.Pow(2, float64(attempt))) * time.Second
	// Add ±25% jitter
	jitter := time.Duration(float64(base) * 0.25)
	return base + jitter
}

// parseRetryAfter extracts the Retry-After header value (seconds) or returns 0.
func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 0
}

// doLLMRequest performs an HTTP POST to the OpenRouter chat completions endpoint
// with exponential back-off retry for transient failures (429, 5xx).
// Returns the response body bytes and status code, or a final error after retries are exhausted.
func doLLMRequest(url, apiKey string, body []byte) ([]byte, int, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt - 1)
			// Respect Retry-After for 429s
			log.Printf("[AI] Retry %d/%d after %v delay", attempt, maxRetries, delay)
			time.Sleep(delay)
		}

		req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		client := &http.Client{Timeout: 120 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("connection failed: %w", err)
			log.Printf("[AI] Request failed (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			log.Printf("[AI] Response read failed (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return respBody, resp.StatusCode, nil
		}

		if isRetryableStatus(resp.StatusCode) {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
			log.Printf("[AI] Retryable error %d (attempt %d/%d): %s", resp.StatusCode, attempt+1, maxRetries+1, truncate(string(respBody), 100))
			// For 429, respect Retry-After
			if resp.StatusCode == 429 {
				if ra := parseRetryAfter(resp); ra > 0 {
					log.Printf("[AI] Respect Retry-After: %v", ra)
					time.Sleep(ra)
				}
			}
			continue
		}

		// Non-retryable error (4xx except 429)
		return respBody, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	return nil, 0, fmt.Errorf("all %d retries exhausted: %w", maxRetries+1, lastErr)
}

// openLLMStream opens an SSE streaming connection with retry for transient failures.
// Returns the response body (streaming io.ReadCloser) or a final error.
func openLLMStream(url, apiKey string, body []byte) (io.ReadCloser, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt - 1)
			log.Printf("[AI] Stream retry %d/%d after %v delay", attempt, maxRetries, delay)
			time.Sleep(delay)
		}

		req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
		if err != nil {
			return nil, fmt.Errorf("failed to build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		client := &http.Client{Timeout: 180 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("connection failed: %w", err)
			log.Printf("[AI] Stream connection failed (attempt %d/%d): %v", attempt+1, maxRetries+1, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}

			// Read error response body for debugging
		var errBody []byte
		if resp.Body != nil {
			errBody, _ = io.ReadAll(io.LimitReader(resp.Body, 2048))
			resp.Body.Close()
		}
		if isRetryableStatus(resp.StatusCode) {
			lastErr = fmt.Errorf("HTTP %d on stream open: %s", resp.StatusCode, string(errBody))
			log.Printf("[AI] Stream retryable error %d (attempt %d/%d): %s", resp.StatusCode, attempt+1, maxRetries+1, truncate(string(errBody), 300))
			if resp.StatusCode == 429 {
				if ra := parseRetryAfter(resp); ra > 0 {
					time.Sleep(ra)
				}
			}
			continue
		}

		log.Printf("[AI] Stream HTTP %d response: %s", resp.StatusCode, truncate(string(errBody), 500))
		return nil, fmt.Errorf("HTTP %d on stream open: %s", resp.StatusCode, truncate(string(errBody), 200))
	}

	return nil, fmt.Errorf("all %d stream retries exhausted: %w", maxRetries+1, lastErr)
}

// truncate shortens a string to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
