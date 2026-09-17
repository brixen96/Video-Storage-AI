package services

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// AdaptiveRateLimiter tracks server response times and adjusts delays
type AdaptiveRateLimiter struct {
	currentDelay    time.Duration
	minDelay        time.Duration
	maxDelay        time.Duration
	responseHistory []time.Duration
	historySize     int
}

// loadSessionCookieFromDB loads the session cookie from database on startup
func (s *ScraperService) loadSessionCookieFromDB() {
	var cookie string
	err := s.db.QueryRow("SELECT value FROM app_settings WHERE key = 'scraper_session_cookie'").Scan(&cookie)
	if err == nil {
		s.sessionCookie = cookie
		log.Printf("Loaded session cookie from database (length: %d)", len(cookie))
	} else if err != sql.ErrNoRows {
		log.Printf("Error loading session cookie from database: %v", err)
	}
}

// SetSessionCookie sets the authentication cookie for scraping
func (s *ScraperService) SetSessionCookie(cookie string) {
	// Clean the cookie string - remove newlines, carriage returns, and extra spaces
	cleaned := strings.ReplaceAll(cookie, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.TrimSpace(cleaned)
	s.sessionCookie = cleaned
	log.Printf("Session cookie set (length: %d)", len(cleaned))

	// Save to database for persistence
	_, err := s.db.Exec(`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ('scraper_session_cookie', ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP
	`, cleaned, cleaned)
	if err != nil {
		log.Printf("Error saving session cookie to database: %v", err)
	} else {
		log.Printf("Session cookie saved to database")
	}
}

// GetSessionCookie returns the current session cookie
func (s *ScraperService) GetSessionCookie() string {
	return s.sessionCookie
}

// VerifySessionCookie verifies that the session cookie is valid by checking the user's profile
func (s *ScraperService) VerifySessionCookie() (bool, string, error) {
	if s.sessionCookie == "" {
		return false, "", fmt.Errorf("no session cookie set")
	}

	// Try to access the account page which requires authentication
	req, err := http.NewRequest("GET", "https://simpcity.cr/account/", nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Cookie", s.sessionCookie)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to verify cookie: %w", err)
	}
	defer resp.Body.Close()

	// If we get redirected to login, the cookie is invalid
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, "", nil
	}

	// Check if we're on the login page (redirect)
	finalURL := resp.Request.URL.String()
	if strings.Contains(finalURL, "/login") {
		return false, "", nil
	}

	// Parse the page to extract username
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		// Cookie worked but couldn't parse - still consider it valid
		return true, "", nil
	}

	// Try to find the username from the page
	username := ""

	// Look for username in various locations
	// Check navigation username display
	doc.Find(".p-navgroup-link--user .p-navgroup-linkText").Each(func(i int, sel *goquery.Selection) {
		if username == "" {
			username = strings.TrimSpace(sel.Text())
		}
	})

	// Alternative: check account page header
	if username == "" {
		doc.Find("h1.p-title-value").Each(func(i int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if strings.HasPrefix(text, "Your account") {
				// Username might be elsewhere
			}
		})
	}

	// Alternative: look for data-user-id or similar attributes
	if username == "" {
		doc.Find("[data-user-id]").Each(func(i int, sel *goquery.Selection) {
			if name, exists := sel.Attr("data-username"); exists && username == "" {
				username = name
			}
		})
	}

	return true, username, nil
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter() *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		currentDelay:    2500 * time.Millisecond, // Start at 2.5s (current default)
		minDelay:        1000 * time.Millisecond, // Minimum 1s delay
		maxDelay:        6000 * time.Millisecond, // Maximum 6s delay
		responseHistory: make([]time.Duration, 0, 5),
		historySize:     5, // Track last 5 response times
	}
}

// RecordResponse records a server response time and adjusts delay
func (rl *AdaptiveRateLimiter) RecordResponse(responseTime time.Duration, statusCode int) {
	// Add response time to history
	rl.responseHistory = append(rl.responseHistory, responseTime)
	if len(rl.responseHistory) > rl.historySize {
		rl.responseHistory = rl.responseHistory[1:]
	}

	// Calculate average response time
	var total time.Duration
	for _, rt := range rl.responseHistory {
		total += rt
	}
	avgResponseTime := total / time.Duration(len(rl.responseHistory))

	// Adjust delay based on server performance
	if statusCode == 429 || statusCode >= 500 {
		// Server is stressed - increase delay significantly
		rl.currentDelay = time.Duration(float64(rl.currentDelay) * 1.5)
		if rl.currentDelay > rl.maxDelay {
			rl.currentDelay = rl.maxDelay
		}
	} else if avgResponseTime > 3*time.Second {
		// Server is slow - increase delay moderately
		rl.currentDelay = time.Duration(float64(rl.currentDelay) * 1.2)
		if rl.currentDelay > rl.maxDelay {
			rl.currentDelay = rl.maxDelay
		}
	} else if avgResponseTime < 1*time.Second && rl.currentDelay > rl.minDelay {
		// Server is fast and responsive - decrease delay
		rl.currentDelay = time.Duration(float64(rl.currentDelay) * 0.9)
		if rl.currentDelay < rl.minDelay {
			rl.currentDelay = rl.minDelay
		}
	}
}

// GetDelay returns the current adaptive delay
func (rl *AdaptiveRateLimiter) GetDelay() time.Duration {
	return rl.currentDelay
}

// GetStatus returns a human-readable status message
func (rl *AdaptiveRateLimiter) GetStatus() string {
	if rl.currentDelay <= 1500*time.Millisecond {
		return fmt.Sprintf("🟢 Server responsive (%.1fs delay)", rl.currentDelay.Seconds())
	} else if rl.currentDelay <= 3000*time.Millisecond {
		return fmt.Sprintf("🟡 Normal pace (%.1fs delay)", rl.currentDelay.Seconds())
	} else {
		return fmt.Sprintf("🟠 Server slow (%.1fs delay)", rl.currentDelay.Seconds())
	}
}
