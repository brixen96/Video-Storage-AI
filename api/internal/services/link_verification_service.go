package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
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

// VerifyLink checks if a single download link is still active
func (s *LinkVerificationService) VerifyLink(link *models.ScrapedDownloadLink) (string, error) {
	// Create HEAD request
	req, err := http.NewRequestWithContext(s.ctx, "HEAD", link.URL, nil)
	if err != nil {
		return "dead", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "dead", nil // Network error or timeout = dead link
	}
	defer resp.Body.Close()

	// Determine status based on HTTP status code
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return "active", nil
	case resp.StatusCode == 404:
		return "dead", nil
	case resp.StatusCode == 410: // Gone
		return "expired", nil
	case resp.StatusCode == 403 || resp.StatusCode == 401:
		// Forbidden/Unauthorized - might need auth, mark as unknown
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
		SELECT id, url, status
		FROM scraped_download_links
		WHERE thread_id = ?
	`, threadID)
	if err != nil {
		s.activityService.FailTask(activity.ID, fmt.Sprintf("Failed to query links: %v", err))
		return err
	}
	defer rows.Close()

	var links []struct {
		ID     int64
		URL    string
		Status string
	}

	for rows.Next() {
		var link struct {
			ID     int64
			URL    string
			Status string
		}
		if err := rows.Scan(&link.ID, &link.URL, &link.Status); err != nil {
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
			ID:  link.ID,
			URL: link.URL,
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
		SELECT id, thread_id, url, status
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
		if err := rows.Scan(&link.ID, &link.ThreadID, &link.URL, &link.Status); err != nil {
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

// GetThreadLinkStats returns link statistics for a specific thread
func (s *LinkVerificationService) GetThreadLinkStats(threadID int64) (map[string]int, error) {
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

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		stats[status] = count
	}

	return stats, nil
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
