package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// LinkVerificationService handles verification of scraped download links
type LinkVerificationService struct {
	db              *sql.DB
	httpClient      *http.Client
	activityService *ActivityService
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	isRunning       bool
	stats           VerificationStats
	gofileToken     string
	gofileTokenMu   sync.Mutex
}

// VerificationStats tracks verification statistics
type VerificationStats struct {
	TotalChecked   int64     `json:"total_checked"`
	ActiveLinks    int64     `json:"active_links"`
	DeadLinks      int64     `json:"dead_links"`
	ExpiredLinks   int64     `json:"expired_links"`
	LastCheckTime  time.Time `json:"last_check_time"`
	ChecksInProgress int     `json:"checks_in_progress"`
}

// NewLinkVerificationService creates a new link verification service
func NewLinkVerificationService(db *sql.DB, activityService *ActivityService) *LinkVerificationService {
	ctx, cancel := context.WithCancel(context.Background())

	return &LinkVerificationService{
		db:              db,
		activityService: activityService,
		ctx:             ctx,
		cancel:          cancel,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 5 redirects
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// Start starts the background verification service
func (s *LinkVerificationService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("service already running")
	}

	s.isRunning = true
	log.Println("🔗 Link Verification Service started")

	// Start background scheduler for periodic checks
	go s.periodicVerification()

	return nil
}

// Stop stops the verification service
func (s *LinkVerificationService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.cancel()
	s.isRunning = false
	log.Println("🔗 Link Verification Service stopped")
}

// periodicVerification runs periodic link checks
func (s *LinkVerificationService) periodicVerification() {
	// Check old links every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Verify links that haven't been checked in 7 days
			cutoffTime := time.Now().Add(-7 * 24 * time.Hour)
			go s.VerifyOldLinks(cutoffTime, 100) // Check 100 links at a time
		}
	}
}

// gofileContentIDRegex extracts the content ID from GoFile URLs like https://gofile.io/d/ThF0BU
var gofileContentIDRegex = regexp.MustCompile(`gofile\.io/d/([a-zA-Z0-9]+)`)

// getGoFileToken creates a guest account and returns an API token (cached)
func (s *LinkVerificationService) getGoFileToken() (string, error) {
	s.gofileTokenMu.Lock()
	defer s.gofileTokenMu.Unlock()

	if s.gofileToken != "" {
		return s.gofileToken, nil
	}

	req, err := http.NewRequestWithContext(s.ctx, "POST", "https://api.gofile.io/accounts", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create gofile account request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create gofile guest account: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read gofile account response: %w", err)
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", fmt.Errorf("failed to parse gofile account response: %w", err)
	}

	if result.Status != "ok" || result.Data.Token == "" {
		return "", fmt.Errorf("gofile account creation failed: status=%s", result.Status)
	}

	s.gofileToken = result.Data.Token
	log.Printf("🔗 GoFile guest token acquired")
	return s.gofileToken, nil
}

// verifyGoFileLink uses GoFile's API to check if content exists
func (s *LinkVerificationService) verifyGoFileLink(link *models.ScrapedDownloadLink) (string, error) {
	matches := gofileContentIDRegex.FindStringSubmatch(link.URL)
	if len(matches) < 2 {
		return "dead", fmt.Errorf("could not extract content ID from GoFile URL: %s", link.URL)
	}
	contentID := matches[1]

	token, err := s.getGoFileToken()
	if err != nil {
		log.Printf("⚠️ GoFile token error, falling back to HEAD check: %v", err)
		return s.verifyLinkWithHEAD(link)
	}

	apiURL := fmt.Sprintf("https://api.gofile.io/contents/%s", contentID)
	req, err := http.NewRequestWithContext(s.ctx, "GET", apiURL, nil)
	if err != nil {
		return "dead", fmt.Errorf("failed to create gofile API request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Website-Token", "4fd6sg89d7s6")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "dead", nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "dead", nil
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "dead", nil
	}

	// GoFile API returns "ok" for existing content, error statuses for dead content
	if result.Status == "ok" {
		return "active", nil
	}

	// Any non-ok status means the content is dead/removed
	// Common: "error-notFound", "error-notPublic", etc.
	log.Printf("🔗 GoFile content %s status: %s", contentID, result.Status)

	// If token was rejected (expired), clear it and retry once
	if result.Status == "error-notAuthorized" {
		s.gofileTokenMu.Lock()
		s.gofileToken = ""
		s.gofileTokenMu.Unlock()

		token, err = s.getGoFileToken()
		if err != nil {
			return "dead", nil
		}

		req2, err := http.NewRequestWithContext(s.ctx, "GET", apiURL, nil)
		if err != nil {
			return "dead", nil
		}
		req2.Header.Set("Authorization", "Bearer "+token)
		req2.Header.Set("X-Website-Token", "4fd6sg89d7s6")

		resp2, err := s.httpClient.Do(req2)
		if err != nil {
			return "dead", nil
		}
		defer resp2.Body.Close()

		bodyBytes2, _ := io.ReadAll(resp2.Body)
		var result2 struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(bodyBytes2, &result2) == nil && result2.Status == "ok" {
			return "active", nil
		}
	}

	return "dead", nil
}

// VerifyLink checks if a single download link is still active
func (s *LinkVerificationService) VerifyLink(link *models.ScrapedDownloadLink) (string, error) {
	provider := strings.ToLower(link.Provider)

	// GoFile is a JavaScript SPA - server returns only HTML shell,
	// error content is rendered client-side. Must use their API instead.
	if provider == "gofile" {
		return s.verifyGoFileLink(link)
	}

	return s.verifyLinkWithHEAD(link)
}

// verifyLinkWithHEAD uses a HEAD request for efficient link verification
func (s *LinkVerificationService) verifyLinkWithHEAD(link *models.ScrapedDownloadLink) (string, error) {
	req, err := http.NewRequestWithContext(s.ctx, "HEAD", link.URL, nil)
	if err != nil {
		return "dead", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "dead", nil // Network error or timeout = dead link
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return "active", nil
	case resp.StatusCode == 404:
		return "dead", nil
	case resp.StatusCode == 410: // Gone
		return "expired", nil
	case resp.StatusCode == 403 || resp.StatusCode == 401:
		return "active", nil // Assume active but requires auth
	default:
		return "dead", nil
	}
}

// UpdateLinkStatus updates a link's status in the database
func (s *LinkVerificationService) UpdateLinkStatus(linkID int64, status string) error {
	_, err := s.db.Exec(`
		UPDATE scraped_download_links
		SET status = ?, last_checked_at = ?
		WHERE id = ?
	`, status, time.Now(), linkID)

	return err
}

// VerifyThreadLinks verifies all links for a specific thread
func (s *LinkVerificationService) VerifyThreadLinks(threadID int64) error {
	// Create activity log
	activity, err := s.activityService.StartTask("link_verification",
		fmt.Sprintf("Verifying download links for thread %d", threadID),
		map[string]interface{}{
			"thread_id": threadID,
		})
	if err != nil {
		return fmt.Errorf("failed to create activity: %w", err)
	}

	// Get all links for the thread
	rows, err := s.db.Query(`
		SELECT id, url, provider, status
		FROM scraped_download_links
		WHERE thread_id = ?
	`, threadID)
	if err != nil {
		s.activityService.FailTask(activity.ID, fmt.Sprintf("Failed to query links: %v", err))
		return err
	}
	defer rows.Close()

	var links []struct {
		ID       int64
		URL      string
		Provider string
		Status   string
	}

	for rows.Next() {
		var link struct {
			ID       int64
			URL      string
			Provider string
			Status   string
		}
		if err := rows.Scan(&link.ID, &link.URL, &link.Provider, &link.Status); err != nil {
			continue
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		s.activityService.CompleteTask(int64(activity.ID), "No links found to verify")
		return nil
	}

	s.activityService.UpdateProgress(activity.ID, 10, fmt.Sprintf("Found %d links to verify", len(links)))

	// Verify each link with rate limiting
	activeCount := 0
	deadCount := 0
	expiredCount := 0

	for i, link := range links {
		// Check if task was paused
		currentActivity, err := s.activityService.GetByID(int64(activity.ID))
		if err == nil && currentActivity.IsPaused {
			log.Printf("⏸️ Link verification paused at link %d/%d", i+1, len(links))
			s.activityService.UpdateProgressLog(int64(activity.ID), 0,
				fmt.Sprintf("⏸️ Paused at link %d/%d", i+1, len(links)))
			return nil
		}

		// Rate limiting: wait 500ms between checks to avoid hammering servers
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		// Verify the link
		linkModel := &models.ScrapedDownloadLink{
			ID:       link.ID,
			URL:      link.URL,
			Provider: link.Provider,
		}

		status, err := s.VerifyLink(linkModel)
		if err != nil {
			log.Printf("Error verifying link %d: %v", link.ID, err)
			continue
		}

		// Update status in database
		if err := s.UpdateLinkStatus(link.ID, status); err != nil {
			log.Printf("Error updating link %d status: %v", link.ID, err)
			continue
		}

		// Update counts
		switch status {
		case "active":
			activeCount++
		case "dead":
			deadCount++
		case "expired":
			expiredCount++
		}

		// Update progress
		progress := int((float64(i+1) / float64(len(links))) * 100)
		s.activityService.UpdateProgress(activity.ID, progress,
			fmt.Sprintf("Verified %d/%d links - Active: %d, Dead: %d, Expired: %d",
				i+1, len(links), activeCount, deadCount, expiredCount))
	}

	// Complete task
	resultMsg := fmt.Sprintf("✅ Verification complete - Active: %d, Dead: %d, Expired: %d out of %d total links",
		activeCount, deadCount, expiredCount, len(links))
	s.activityService.CompleteTask(int64(activity.ID), resultMsg)

	// Broadcast verification complete via WebSocket
	if wsHub != nil {
		wsHub.BroadcastLinkVerificationComplete(threadID, activeCount, deadCount, expiredCount, len(links))
	}

	// Update stats
	s.mu.Lock()
	s.stats.TotalChecked += int64(len(links))
	s.stats.ActiveLinks += int64(activeCount)
	s.stats.DeadLinks += int64(deadCount)
	s.stats.ExpiredLinks += int64(expiredCount)
	s.stats.LastCheckTime = time.Now()
	s.mu.Unlock()

	return nil
}

// VerifyOldLinks verifies links that haven't been checked recently
func (s *LinkVerificationService) VerifyOldLinks(cutoffTime time.Time, limit int) error {
	// Get links that haven't been checked since cutoffTime
	rows, err := s.db.Query(`
		SELECT id, thread_id, url, provider, status
		FROM scraped_download_links
		WHERE last_checked_at IS NULL OR last_checked_at < ?
		ORDER BY last_checked_at ASC NULLS FIRST
		LIMIT ?
	`, cutoffTime, limit)
	if err != nil {
		return fmt.Errorf("failed to query old links: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var link models.ScrapedDownloadLink
		if err := rows.Scan(&link.ID, &link.ThreadID, &link.URL, &link.Provider, &link.Status); err != nil {
			continue
		}

		// Rate limiting
		if count > 0 {
			time.Sleep(1 * time.Second)
		}

		status, err := s.VerifyLink(&link)
		if err != nil {
			log.Printf("Error verifying old link %d: %v", link.ID, err)
			continue
		}

		if err := s.UpdateLinkStatus(link.ID, status); err != nil {
			log.Printf("Error updating old link %d: %v", link.ID, err)
			continue
		}

		count++
	}

	log.Printf("🔗 Verified %d old links", count)
	return nil
}

// GetStats returns current verification statistics
func (s *LinkVerificationService) GetStats() VerificationStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// ThreadLinkStats contains link statistics and last verification time for a thread
type ThreadLinkStats struct {
	Stats       map[string]int `json:"stats"`
	LastChecked *time.Time     `json:"last_checked,omitempty"`
}

// GetThreadLinkStats returns link statistics for a specific thread
func (s *LinkVerificationService) GetThreadLinkStats(threadID int64) (*ThreadLinkStats, error) {
	rows, err := s.db.Query(`
		SELECT status, COUNT(*) as count
		FROM scraped_download_links
		WHERE thread_id = ?
		GROUP BY status
	`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &ThreadLinkStats{
		Stats: make(map[string]int),
	}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		result.Stats[status] = count
	}

	// Get last verification time (scan as string since SQLite MAX() returns TEXT)
	var lastCheckedStr sql.NullString
	s.db.QueryRow(`
		SELECT MAX(last_checked_at) FROM scraped_download_links WHERE thread_id = ?
	`, threadID).Scan(&lastCheckedStr)
	if lastCheckedStr.Valid && lastCheckedStr.String != "" {
		// Try multiple time formats SQLite might use
		formats := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02T15:04:05",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, lastCheckedStr.String); err == nil {
				result.LastChecked = &t
				break
			}
		}
	}

	return result, nil
}

// ProviderHealth represents health statistics for a download provider
type ProviderHealth struct {
	Provider      string  `json:"provider"`
	TotalLinks    int     `json:"total_links"`
	ActiveLinks   int     `json:"active_links"`
	DeadLinks     int     `json:"dead_links"`
	ExpiredLinks  int     `json:"expired_links"`
	UncheckedLinks int    `json:"unchecked_links"`
	HealthScore   float64 `json:"health_score"` // Percentage of active links
	LastChecked   *time.Time `json:"last_checked,omitempty"`
}

// GetProviderHealthStats returns health statistics for all providers
func (s *LinkVerificationService) GetProviderHealthStats() ([]*ProviderHealth, error) {
	// Query provider statistics grouped by provider and status
	rows, err := s.db.Query(`
		SELECT
			provider,
			status,
			COUNT(*) as count,
			MAX(last_checked_at) as last_checked
		FROM scraped_download_links
		GROUP BY provider, status
		ORDER BY provider, status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query provider stats: %w", err)
	}
	defer rows.Close()

	// Collect data grouped by provider
	providerData := make(map[string]*ProviderHealth)

	for rows.Next() {
		var provider, status string
		var count int
		var lastChecked sql.NullTime

		if err := rows.Scan(&provider, &status, &count, &lastChecked); err != nil {
			log.Printf("Error scanning provider stat: %v", err)
			continue
		}

		// Initialize provider entry if doesn't exist
		if _, exists := providerData[provider]; !exists {
			providerData[provider] = &ProviderHealth{
				Provider: provider,
			}
		}

		ph := providerData[provider]
		ph.TotalLinks += count

		// Update last checked time if this is more recent
		if lastChecked.Valid {
			if ph.LastChecked == nil || lastChecked.Time.After(*ph.LastChecked) {
				ph.LastChecked = &lastChecked.Time
			}
		}

		// Categorize by status
		switch status {
		case "active":
			ph.ActiveLinks += count
		case "dead":
			ph.DeadLinks += count
		case "expired":
			ph.ExpiredLinks += count
		default:
			ph.UncheckedLinks += count
		}
	}

	// Calculate health scores and convert to slice
	var results []*ProviderHealth
	for _, ph := range providerData {
		if ph.TotalLinks > 0 {
			ph.HealthScore = float64(ph.ActiveLinks) / float64(ph.TotalLinks) * 100
		}
		results = append(results, ph)
	}

	// Sort by total links (most popular first)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].TotalLinks > results[i].TotalLinks {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results, nil
}

// GetProviderHealth returns health statistics for a specific provider
func (s *LinkVerificationService) GetProviderHealth(provider string) (*ProviderHealth, error) {
	rows, err := s.db.Query(`
		SELECT
			status,
			COUNT(*) as count,
			MAX(last_checked_at) as last_checked
		FROM scraped_download_links
		WHERE provider = ?
		GROUP BY status
	`, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to query provider health: %w", err)
	}
	defer rows.Close()

	ph := &ProviderHealth{
		Provider: provider,
	}

	for rows.Next() {
		var status string
		var count int
		var lastChecked sql.NullTime

		if err := rows.Scan(&status, &count, &lastChecked); err != nil {
			continue
		}

		ph.TotalLinks += count

		if lastChecked.Valid {
			if ph.LastChecked == nil || lastChecked.Time.After(*ph.LastChecked) {
				ph.LastChecked = &lastChecked.Time
			}
		}

		switch status {
		case "active":
			ph.ActiveLinks += count
		case "dead":
			ph.DeadLinks += count
		case "expired":
			ph.ExpiredLinks += count
		default:
			ph.UncheckedLinks += count
		}
	}

	if ph.TotalLinks > 0 {
		ph.HealthScore = float64(ph.ActiveLinks) / float64(ph.TotalLinks) * 100
	}

	return ph, nil
}

// ===== SEXURIA LINK VERIFICATION =====

// SexuriaLinkResult represents the result of verifying a Sexuria link
type SexuriaLinkResult struct {
	LinkID     int64  `json:"link_id"`
	URL        string `json:"url"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
	StatusCode int    `json:"status_code"`
}

// VerifySexuriaLink verifies a single Sexuria download link
func (s *LinkVerificationService) VerifySexuriaLink(linkID int64, url string, provider ...string) (*SexuriaLinkResult, error) {
	result := &SexuriaLinkResult{
		LinkID: linkID,
		URL:    url,
	}

	// Determine provider
	prov := ""
	if len(provider) > 0 {
		prov = provider[0]
	}

	// GoFile uses API-based verification
	if strings.ToLower(prov) == "gofile" {
		link := &models.ScrapedDownloadLink{
			ID:       linkID,
			URL:      url,
			Provider: prov,
		}
		status, err := s.verifyGoFileLink(link)
		if err != nil {
			result.Status = "dead"
			return result, nil
		}
		result.Status = status
		return result, nil
	}

	// Default: HEAD request for efficiency
	req, err := http.NewRequestWithContext(s.ctx, "HEAD", url, nil)
	if err != nil {
		result.Status = "dead"
		return result, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		result.Status = "dead"
		return result, nil
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		result.Status = "active"
	case resp.StatusCode == 404:
		result.Status = "dead"
	case resp.StatusCode == 410:
		result.Status = "expired"
	case resp.StatusCode == 403 || resp.StatusCode == 401:
		result.Status = "active"
	case resp.StatusCode >= 500:
		result.Status = "unknown"
	default:
		result.Status = "dead"
	}

	return result, nil
}

// UpdateSexuriaLinkStatus updates a Sexuria link's status in the database
func (s *LinkVerificationService) UpdateSexuriaLinkStatus(linkID int64, status string, statusCode int) error {
	_, err := s.db.Exec(`
		UPDATE sexuria_download_links
		SET status = ?, last_checked_at = ?, last_status_code = ?
		WHERE id = ?
	`, status, time.Now(), statusCode, linkID)
	return err
}

// VerifySexuriaItemLinks verifies all links for a specific Sexuria item
func (s *LinkVerificationService) VerifySexuriaItemLinks(videoID int64) (map[string]int, error) {
	// Get all links for this item
	rows, err := s.db.Query(`
		SELECT id, url, provider
		FROM sexuria_download_links
		WHERE video_id = ?
	`, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []struct {
		ID       int64
		URL      string
		Provider string
	}

	for rows.Next() {
		var link struct {
			ID       int64
			URL      string
			Provider string
		}
		if err := rows.Scan(&link.ID, &link.URL, &link.Provider); err != nil {
			continue
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		return map[string]int{"total": 0}, nil
	}

	stats := map[string]int{
		"total":   len(links),
		"active":  0,
		"dead":    0,
		"expired": 0,
		"unknown": 0,
	}

	for i, link := range links {
		// Rate limiting
		if i > 0 {
			time.Sleep(500 * time.Millisecond)
		}

		result, err := s.VerifySexuriaLink(link.ID, link.URL, link.Provider)
		if err != nil {
			log.Printf("Error verifying Sexuria link %d: %v", link.ID, err)
			continue
		}

		// Update status in database
		s.UpdateSexuriaLinkStatus(link.ID, result.Status, result.StatusCode)

		// Update stats
		stats[result.Status]++
	}

	return stats, nil
}

// VerifyAllSexuriaLinks verifies Sexuria links that haven't been checked recently
func (s *LinkVerificationService) VerifyAllSexuriaLinks(limit int, cutoffDays int) error {
	cutoffTime := time.Now().Add(-time.Duration(cutoffDays) * 24 * time.Hour)

	rows, err := s.db.Query(`
		SELECT id, video_id, url, provider, status
		FROM sexuria_download_links
		WHERE last_checked_at IS NULL OR last_checked_at < ?
		ORDER BY last_checked_at ASC NULLS FIRST
		LIMIT ?
	`, cutoffTime, limit)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	activeCount := 0
	deadCount := 0

	for rows.Next() {
		var link struct {
			ID       int64
			VideoID  int64
			URL      string
			Provider string
			Status   string
		}
		if err := rows.Scan(&link.ID, &link.VideoID, &link.URL, &link.Provider, &link.Status); err != nil {
			continue
		}

		// Rate limiting
		if count > 0 {
			time.Sleep(800 * time.Millisecond)
		}

		result, err := s.VerifySexuriaLink(link.ID, link.URL, link.Provider)
		if err != nil {
			log.Printf("Error verifying Sexuria link %d: %v", link.ID, err)
			continue
		}

		s.UpdateSexuriaLinkStatus(link.ID, result.Status, result.StatusCode)

		if result.Status == "active" {
			activeCount++
		} else if result.Status == "dead" {
			deadCount++
		}

		count++
	}

	log.Printf("🔗 Verified %d Sexuria links (Active: %d, Dead: %d)", count, activeCount, deadCount)
	return nil
}

// GetSexuriaLinkStats returns overall statistics for Sexuria download links
func (s *LinkVerificationService) GetSexuriaLinkStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total links by status
	rows, err := s.db.Query(`
		SELECT status, COUNT(*) as count
		FROM sexuria_download_links
		GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalLinks := 0
	activeLinks := 0
	deadLinks := 0
	unknownLinks := 0

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}

		totalLinks += count
		switch status {
		case "active":
			activeLinks = count
		case "dead", "expired":
			deadLinks += count
		default:
			unknownLinks += count
		}
	}

	stats["total_links"] = totalLinks
	stats["active_links"] = activeLinks
	stats["dead_links"] = deadLinks
	stats["unknown_links"] = unknownLinks

	// Calculate health percentage
	if totalLinks > 0 {
		checkedLinks := activeLinks + deadLinks
		if checkedLinks > 0 {
			stats["health_percent"] = float64(activeLinks) / float64(checkedLinks) * 100
		} else {
			stats["health_percent"] = 0.0
		}
	} else {
		stats["health_percent"] = 0.0
	}

	// Provider breakdown
	providerRows, err := s.db.Query(`
		SELECT provider, status, COUNT(*) as count
		FROM sexuria_download_links
		GROUP BY provider, status
		ORDER BY provider
	`)
	if err == nil {
		defer providerRows.Close()

		providerStats := make(map[string]map[string]int)
		for providerRows.Next() {
			var provider, status string
			var count int
			if err := providerRows.Scan(&provider, &status, &count); err != nil {
				continue
			}
			if _, exists := providerStats[provider]; !exists {
				providerStats[provider] = make(map[string]int)
			}
			providerStats[provider][status] = count
		}
		stats["providers"] = providerStats
	}

	// Last verification time
	var lastChecked sql.NullTime
	s.db.QueryRow(`SELECT MAX(last_checked_at) FROM sexuria_download_links`).Scan(&lastChecked)
	if lastChecked.Valid {
		stats["last_verified"] = lastChecked.Time
	}

	return stats, nil
}
