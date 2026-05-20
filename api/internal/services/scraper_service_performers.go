package services

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// LinkThreadToPerformer creates a link between a scraped thread and a performer
func (s *ScraperService) LinkThreadToPerformer(threadID int64, performerID int64, confidence float64) error {
	_, err := s.db.Exec(`
		INSERT INTO performer_scraped_threads (performer_id, thread_id, confidence)
		VALUES (?, ?, ?)
		ON CONFLICT(performer_id, thread_id) DO UPDATE SET confidence = ?
	`, performerID, threadID, confidence, confidence)

	if err != nil {
		return fmt.Errorf("failed to link thread to performer: %w", err)
	}

	return nil
}

// FindPerformer finds an existing performer by name (case-insensitive)
func (s *ScraperService) FindPerformer(name string) (int64, error) {
	var performerID int64
	err := s.db.QueryRow(`
		SELECT id FROM performers WHERE LOWER(name) = LOWER(?)
	`, name).Scan(&performerID)

	if err == sql.ErrNoRows {
		return 0, nil // Return 0 if not found, no error
	}

	if err != nil {
		return 0, fmt.Errorf("failed to query performer: %w", err)
	}

	return performerID, nil
}

// CreatePerformerFromThread creates a new performer from a thread's extracted name
func (s *ScraperService) CreatePerformerFromThread(name string) (int64, error) {
	// Check if performer already exists
	existingID, err := s.FindPerformer(name)
	if err != nil {
		return 0, err
	}
	if existingID != 0 {
		return existingID, nil // Return existing performer
	}

	// Create new performer
	result, err := s.db.Exec(`
		INSERT INTO performers (name, category, metadata, created_at, updated_at)
		VALUES (?, 'regular', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, name)

	if err != nil {
		return 0, fmt.Errorf("failed to create performer: %w", err)
	}

	performerID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get performer ID: %w", err)
	}

	log.Printf("Created new performer: %s (ID: %d)", name, performerID)
	return performerID, nil
}

// GetThreadsByPerformer retrieves all scraped threads linked to a performer
func (s *ScraperService) GetThreadsByPerformer(performerID int64) ([]*models.ScrapedThread, error) {
	query := `
		SELECT
			t.id, t.external_id, t.source, t.title, t.url, t.author, t.category,
			t.view_count, t.reply_count, t.post_count, t.download_count,
			t.metadata, t.first_scraped_at, t.last_scraped_at, t.last_updated_at, t.created_at
		FROM scraped_threads t
		INNER JOIN performer_scraped_threads pt ON t.id = pt.thread_id
		WHERE pt.performer_id = ?
		ORDER BY t.last_scraped_at DESC
	`

	rows, err := s.db.Query(query, performerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query threads: %w", err)
	}
	defer rows.Close()

	var threads []*models.ScrapedThread
	for rows.Next() {
		thread := &models.ScrapedThread{}
		err := rows.Scan(
			&thread.ID, &thread.ExternalID, &thread.Source, &thread.Title, &thread.URL,
			&thread.Author, &thread.Category, &thread.ViewCount, &thread.ReplyCount,
			&thread.PostCount, &thread.DownloadCount, &thread.Metadata,
			&thread.FirstScrapedAt, &thread.LastScrapedAt, &thread.LastUpdatedAt, &thread.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan thread: %w", err)
		}

		// Unmarshal metadata
		thread.UnmarshalMetadata()
		threads = append(threads, thread)
	}

	return threads, nil
}

// CheckLinkStatus checks if a download link is still active
func (s *ScraperService) CheckLinkStatus(linkURL string) string {
	// Create HEAD request to check if link is alive
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Allow redirects
		},
	}

	req, err := http.NewRequest("HEAD", linkURL, nil)
	if err != nil {
		return "unknown"
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		// Network error or timeout
		return "dead"
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "active"
	} else if resp.StatusCode == 404 || resp.StatusCode == 410 {
		return "dead"
	} else if resp.StatusCode == 403 || resp.StatusCode == 401 {
		return "restricted"
	} else if resp.StatusCode >= 500 {
		return "unavailable"
	}

	return "unknown"
}

// CheckAllLinkStatuses checks the status of all download links
func (s *ScraperService) CheckAllLinkStatuses() error {
	query := `SELECT id, url FROM scraped_download_links WHERE status != 'dead'`

	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query links: %w", err)
	}
	defer rows.Close()

	checkedCount := 0
	deadCount := 0

	for rows.Next() {
		var linkID int64
		var linkURL string

		if err := rows.Scan(&linkID, &linkURL); err != nil {
			continue
		}

		// Check link status
		status := s.CheckLinkStatus(linkURL)

		// Update database
		_, err := s.db.Exec(`
			UPDATE scraped_download_links
			SET status = ?, last_checked_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, status, linkID)

		if err != nil {
			log.Printf("Error updating link status: %v", err)
			continue
		}

		checkedCount++
		if status == "dead" {
			deadCount++
		}

		// Rate limit: wait 1 second between checks
		time.Sleep(1 * time.Second)

		// Log progress every 10 links
		if checkedCount%10 == 0 {
			log.Printf("Checked %d links, found %d dead", checkedCount, deadCount)
		}
	}

	log.Printf("Link status check complete. Checked: %d, Dead: %d", checkedCount, deadCount)
	return nil
}

// AutoLinkThreadsToPerformers automatically links threads to performers based on title matching
func (s *ScraperService) AutoLinkThreadsToPerformers() error {
	// Get all threads (using a large limit to get all threads)
	threads, _, err := s.GetAllThreads(10000, 0, "", "", "", "", "")
	if err != nil {
		return fmt.Errorf("failed to get threads: %w", err)
	}

	linkedCount := 0
	for _, thread := range threads {
		// Extract performer names from title
		performerNames := s.ExtractPerformerNameFromTitle(thread.Title)

		for _, name := range performerNames {
			// Find existing performer (don't create new ones automatically)
			performerID, err := s.FindPerformer(name)
			if err != nil {
				log.Printf("Error finding performer %s: %v", name, err)
				continue
			}

			// Skip if performer doesn't exist
			if performerID == 0 {
				log.Printf("Skipping thread '%s' - performer '%s' not found (create manually if needed)", thread.Title, name)
				continue
			}

			// Link thread to performer
			err = s.LinkThreadToPerformer(thread.ID, performerID, 0.8) // 0.8 confidence for auto-linking
			if err != nil {
				log.Printf("Error linking thread %d to performer %d: %v", thread.ID, performerID, err)
				continue
			}

			linkedCount++
			log.Printf("Linked thread '%s' to performer '%s'", thread.Title, name)
		}
	}

	log.Printf("Auto-linked %d thread-performer relationships", linkedCount)
	return nil
}
