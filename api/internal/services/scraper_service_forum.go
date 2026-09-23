package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// ForumThreadInfo represents basic thread information from forum listing
type ForumThreadInfo struct {
	Title      string
	URL        string
	ExternalID string
	Author     string
	ReplyCount int
	ViewCount  int
}

// ScrapeCheckpoint stores progress for resuming interrupted scrapes
type ScrapeCheckpoint struct {
	CurrentPage         int                           `json:"current_page"`
	PostNumber          int                           `json:"post_number"`
	EstimatedTotalPages int                           `json:"estimated_total_pages"`
	PostsCollected      int                           `json:"posts_collected"`
	LinksCollected      int                           `json:"links_collected"`
	LastPageURL         string                        `json:"last_page_url"`
	CleanURL            string                        `json:"clean_url"`
	SavedPosts          []*models.ScrapedPost         `json:"-"` // Not serialized
	SavedLinks          []*models.ScrapedDownloadLink `json:"-"` // Not serialized
}

// ScrapeForumCategory scrapes all threads from a forum category (with pagination)
func (s *ScraperService) ScrapeForumCategory(forumURL string, activityID ...int) ([]ForumThreadInfo, error) {
	// Auto-migrate broken domains (e.g. simpcity.is → simpcity.cr)
	forumURL = migrateURL(forumURL)

	log.Printf("Starting forum category scrape: %s", forumURL)

	var allThreads []ForumThreadInfo
	currentPage := 1
	baseURL := strings.TrimSuffix(forumURL, "/")

	for {
		// Check if task has been paused (if activity ID provided)
		if len(activityID) > 0 && activityID[0] > 0 {
			currentActivity, err := s.activityService.GetByID(int64(activityID[0]))
			if err == nil && currentActivity.IsPaused {
				log.Printf("⏸️ Task paused during forum listing at page %d", currentPage)
				return allThreads, fmt.Errorf("task paused by user at page %d", currentPage)
			}
		}

		pageURL := baseURL
		if currentPage > 1 {
			pageURL = fmt.Sprintf("%s/page-%d", baseURL, currentPage)
		}

		log.Printf("Scraping forum page %d: %s", currentPage, pageURL)

		// Retry logic for server errors (500, 502, 503) and rate limiting
		var resp *http.Response
		var doc *goquery.Document
		var err error
		var req *http.Request
		maxRetries := 3
		retryDelay := 10 * time.Second

		for attempt := 0; attempt <= maxRetries; attempt++ {
			// Create request
			req, err = http.NewRequest("GET", pageURL, nil)
			if err != nil {
				return allThreads, fmt.Errorf("failed to create request: %w", err)
			}

			// Add session cookie if available
			if s.sessionCookie != "" {
				req.Header.Set("Cookie", s.sessionCookie)
			}

			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

			// Fetch the page
			resp, err = s.httpClient.Do(req)
			if err != nil {
				if attempt < maxRetries {
					waitTime := retryDelay * time.Duration(attempt+1)
					log.Printf("Request error on forum page %d. Waiting %v before retry %d/%d", currentPage, waitTime, attempt+1, maxRetries)
					time.Sleep(waitTime)
					continue
				}
				return allThreads, fmt.Errorf("failed to fetch forum page after %d retries: %w", maxRetries, err)
			}

			// Handle server errors with retry
			if resp.StatusCode >= 500 && resp.StatusCode < 600 {
				resp.Body.Close()
				if attempt < maxRetries {
					waitTime := retryDelay * time.Duration(attempt+1)
					log.Printf("Server error (%d) on forum page %d. Waiting %v before retry %d/%d", resp.StatusCode, currentPage, waitTime, attempt+1, maxRetries)
					time.Sleep(waitTime)
					continue
				}
				return allThreads, fmt.Errorf("server error after %d retries on forum page %d: %d", maxRetries, currentPage, resp.StatusCode)
			}

			// Handle rate limiting with retry
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				if attempt < maxRetries {
					waitTime := retryDelay * time.Duration(attempt+1)
					log.Printf("Rate limited (429) on forum page %d. Waiting %v before retry %d/%d", currentPage, waitTime, attempt+1, maxRetries)
					time.Sleep(waitTime)
					continue
				}
				return allThreads, fmt.Errorf("rate limited after %d retries on forum page %d (429)", maxRetries, currentPage)
			}

			// Check for other non-OK status codes
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return allThreads, fmt.Errorf("unexpected status code on forum page %d: %d", currentPage, resp.StatusCode)
			}

			// Parse HTML
			doc, err = goquery.NewDocumentFromReader(resp.Body)
			resp.Body.Close()
			if err != nil {
				if attempt < maxRetries {
					waitTime := retryDelay * time.Duration(attempt+1)
					log.Printf("Failed to parse HTML on forum page %d. Waiting %v before retry %d/%d", currentPage, waitTime, attempt+1, maxRetries)
					time.Sleep(waitTime)
					continue
				}
				return allThreads, fmt.Errorf("failed to parse HTML after %d retries: %w", maxRetries, err)
			}

			// Success - break out of retry loop
			break
		}

		threadsOnPage := 0

		// Extract thread information from listing
		doc.Find(".structItem--thread").Each(func(i int, sel *goquery.Selection) {
			thread := ForumThreadInfo{}

			// Extract title and URL
			titleEl := sel.Find(".structItem-title a[data-tp-primary]").First()
			thread.Title = strings.TrimSpace(titleEl.Text())
			threadURL, exists := titleEl.Attr("href")
			if exists {
				// Convert relative URL to absolute
				if strings.HasPrefix(threadURL, "/") {
					parsedURL, _ := url.Parse(baseURL)
					thread.URL = fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, threadURL)
				} else {
					thread.URL = threadURL
				}

				// Extract thread ID from URL
				thread.ExternalID, _ = s.extractThreadID(thread.URL)
			}

			// Extract author
			thread.Author = strings.TrimSpace(sel.Find(".structItem-cell--meta .username").First().Text())

			// Extract reply count
			replyText := strings.TrimSpace(sel.Find(".structItem-cell--meta dd").First().Text())
			fmt.Sscanf(replyText, "%d", &thread.ReplyCount)

			// Extract view count (not always available)
			// This is typically in a tooltip or data attribute

			if thread.Title != "" && thread.URL != "" {
				allThreads = append(allThreads, thread)
				threadsOnPage++
			}
		})

		log.Printf("Found %d threads on page %d (total so far: %d)", threadsOnPage, currentPage, len(allThreads))

		// Check if there's a next page
		hasNextPage := false
		if doc.Find(".pageNav-jump--next").Length() > 0 {
			hasNextPage = true
		}

		if !hasNextPage || threadsOnPage == 0 {
			log.Printf("No more pages to scrape. Total threads found: %d", len(allThreads))
			break
		}

		currentPage++
		time.Sleep(2 * time.Second) // Be respectful to the server
	}

	return allThreads, nil
}

// ScrapeForumAndSaveAll scrapes all threads from a forum and saves them with full content
// Uses concurrent workers for significantly faster scraping
func (s *ScraperService) ScrapeForumAndSaveAll(forumURL string) error {
	// Create activity log for tracking
	activity, err := s.activityService.StartTask("forum_scrape", fmt.Sprintf("Scraping forum: %s", forumURL), map[string]interface{}{
		"forum_url": forumURL,
	})
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	log.Printf("Starting multi-threaded forum scrape: %s", forumURL)

	// First, get all thread URLs from the forum listing
	if activity != nil {
		s.activityService.UpdateProgress(activity.ID, 5, "Scanning forum pages for threads...")
	}

	var threads []ForumThreadInfo
	if activity != nil {
		threads, err = s.ScrapeForumCategory(forumURL, activity.ID)
	} else {
		threads, err = s.ScrapeForumCategory(forumURL)
	}
	if err != nil {
		if activity != nil {
			// Check if it was paused
			if strings.Contains(err.Error(), "paused by user") {
				s.activityService.UpdateProgressLog(int64(activity.ID), 0, "⏸️ Task paused during forum listing scan")
				return nil
			}
			s.activityService.FailTask(activity.ID, fmt.Sprintf("Failed to scrape forum category: %v", err))
		}
		return fmt.Errorf("failed to scrape forum category: %w", err)
	}

	totalThreads := len(threads)
	log.Printf("Found %d threads to scrape sequentially", totalThreads)
	if activity != nil {
		s.activityService.UpdateProgress(activity.ID, 10, fmt.Sprintf("Found %d threads. Starting sequential scrape...", totalThreads))
	}

	// Check if resuming from checkpoint
	startIndex := 0
	if activity != nil {
		currentActivity, err := s.activityService.GetByID(int64(activity.ID))
		if err == nil {
			// Unmarshal checkpoint data
			if err := currentActivity.UnmarshalCheckpoint(); err == nil && currentActivity.CheckpointObj["thread_index"] != nil {
				startIndex = int(currentActivity.CheckpointObj["thread_index"].(float64))
				log.Printf("📍 Resuming forum scrape from thread %d/%d", startIndex+1, totalThreads)
				s.activityService.UpdateProgressLog(int64(activity.ID), 0,
					fmt.Sprintf("📍 Resumed from checkpoint: thread %d/%d", startIndex+1, totalThreads))
			}
		}
	}

	// Sequential scraping with adaptive rate limiting
	successCount := 0
	errorCount := 0
	consecutiveErrors := 0
	baseDelay := 8 * time.Second

	for i := startIndex; i < totalThreads; i++ {
		threadInfo := threads[i]

		// Check if task has been paused
		if activity != nil {
			currentActivity, err := s.activityService.GetByID(int64(activity.ID))
			if err == nil && currentActivity.IsPaused {
				log.Printf("⏸️ Task paused at thread %d/%d", i+1, totalThreads)
				s.activityService.UpdateProgressLog(int64(activity.ID), 0,
					fmt.Sprintf("⏸️ Task paused at thread %d/%d. Progress saved.", i+1, totalThreads))
				return nil // Exit gracefully
			}
		}

		log.Printf("Scraping thread %d/%d: %s", i+1, totalThreads, threadInfo.Title)

		err := s.ScrapeThreadComplete(threadInfo.URL, activity.ID)

		if err != nil {
			log.Printf("Error scraping thread %s: %v", threadInfo.URL, err)
			errorCount++
			consecutiveErrors++

			// Adaptive backoff: increase delay after consecutive errors
			if consecutiveErrors >= 3 {
				extraDelay := time.Duration(consecutiveErrors-2) * 10 * time.Second
				log.Printf("⚠️ Multiple consecutive errors (%d). Adding %v extra delay to help server recover", consecutiveErrors, extraDelay)
				if activity != nil {
					s.activityService.UpdateProgressLog(int64(activity.ID), 0,
						fmt.Sprintf("⚠️ Server struggling (%d errors). Slowing down scrape by %v", consecutiveErrors, extraDelay))
				}
				time.Sleep(extraDelay)
			}
		} else {
			successCount++
			consecutiveErrors = 0 // Reset on success
		}

		// Update progress
		if activity != nil {
			progress := 10 + (85 * (i + 1) / totalThreads)
			s.activityService.UpdateProgress(activity.ID, progress,
				fmt.Sprintf("Scraped %d/%d threads (Success: %d, Errors: %d)", i+1, totalThreads, successCount, errorCount))

			// Save checkpoint every 10 threads
			if (i+1)%10 == 0 {
				s.activityService.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
					"thread_index":      i + 1,
					"threads_completed": i + 1,
					"total_threads":     totalThreads,
					"success_count":     successCount,
					"error_count":       errorCount,
				})
			}
		}

		// Base delay to avoid rate limiting and server overload
		if i < totalThreads-1 { // Don't delay after the last thread
			time.Sleep(baseDelay)
		}
	}

	log.Printf("Multi-threaded forum scrape complete. Success: %d, Errors: %d", successCount, errorCount)

	if activity != nil {
		s.activityService.CompleteTask(int64(activity.ID),
			fmt.Sprintf("Forum scrape complete. Success: %d, Errors: %d", successCount, errorCount))
	}

	// Notify AI Companion about completion
	if s.aiCompanionService != nil {
		// Trigger auto-link after forum scrape
		go func() {
			time.Sleep(2 * time.Second)
			log.Println("🤖 Auto-linking threads to performers...")
			if err := s.AutoLinkThreadsToPerformers(); err != nil {
				log.Printf("Error in auto-link: %v", err)
			}
		}()
	}

	return nil
}

// ResumeForumScrape resumes a paused forum scrape from checkpoint
func (s *ScraperService) ResumeForumScrape(forumURL string, activityID int) error {
	log.Printf("📍 Resuming forum scrape for activity %d: %s", activityID, forumURL)

	activity := &models.Activity{ID: activityID}

	// Update progress to show resume
	s.activityService.UpdateProgress(activity.ID, 5, "Resuming... Scanning forum pages for threads...")

	// Get all thread URLs from the forum listing
	threads, err := s.ScrapeForumCategory(forumURL, activity.ID)
	if err != nil {
		if strings.Contains(err.Error(), "paused by user") {
			s.activityService.UpdateProgressLog(int64(activity.ID), 0, "⏸️ Task paused during forum listing scan")
			return nil
		}
		s.activityService.FailTask(activity.ID, fmt.Sprintf("Failed to scrape forum category: %v", err))
		return fmt.Errorf("failed to scrape forum category: %w", err)
	}

	totalThreads := len(threads)
	log.Printf("Found %d threads to scrape sequentially", totalThreads)
	s.activityService.UpdateProgress(activity.ID, 10, fmt.Sprintf("Found %d threads. Resuming sequential scrape...", totalThreads))

	// Check if resuming from checkpoint
	startIndex := 0
	currentActivity, err := s.activityService.GetByID(int64(activity.ID))
	if err == nil {
		if err := currentActivity.UnmarshalCheckpoint(); err == nil && currentActivity.CheckpointObj["thread_index"] != nil {
			startIndex = int(currentActivity.CheckpointObj["thread_index"].(float64))
			log.Printf("📍 Resuming forum scrape from thread %d/%d", startIndex+1, totalThreads)
			s.activityService.UpdateProgressLog(int64(activity.ID), 0,
				fmt.Sprintf("📍 Resumed from checkpoint: thread %d/%d", startIndex+1, totalThreads))
		}
	}

	// Sequential scraping with adaptive rate limiting
	successCount := 0
	errorCount := 0
	consecutiveErrors := 0
	baseDelay := 8 * time.Second

	for i := startIndex; i < totalThreads; i++ {
		threadInfo := threads[i]

		// Check if task has been paused
		currentActivity, err := s.activityService.GetByID(int64(activity.ID))
		if err == nil && currentActivity.IsPaused {
			log.Printf("⏸️ Task paused at thread %d/%d", i+1, totalThreads)
			s.activityService.UpdateProgressLog(int64(activity.ID), 0,
				fmt.Sprintf("⏸️ Task paused at thread %d/%d. Progress saved.", i+1, totalThreads))
			return nil
		}

		log.Printf("Scraping thread %d/%d: %s", i+1, totalThreads, threadInfo.Title)
		err = s.ScrapeThreadComplete(threadInfo.URL, activity.ID)

		if err != nil {
			log.Printf("Error scraping thread %s: %v", threadInfo.URL, err)
			errorCount++
			consecutiveErrors++

			if consecutiveErrors >= 3 {
				extraDelay := time.Duration(consecutiveErrors-2) * 10 * time.Second
				log.Printf("⚠️ Multiple consecutive errors (%d). Adding %v extra delay to help server recover", consecutiveErrors, extraDelay)
				s.activityService.UpdateProgressLog(int64(activity.ID), 0,
					fmt.Sprintf("⚠️ Server struggling (%d errors). Slowing down scrape by %v", consecutiveErrors, extraDelay))
				time.Sleep(extraDelay)
			}
		} else {
			successCount++
			consecutiveErrors = 0
		}

		// Update progress
		progress := 10 + (85 * (i + 1) / totalThreads)
		s.activityService.UpdateProgress(activity.ID, progress,
			fmt.Sprintf("Scraped %d/%d threads (Success: %d, Errors: %d)", i+1, totalThreads, successCount, errorCount))

		// Save checkpoint every 10 threads
		if (i+1)%10 == 0 {
			s.activityService.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
				"thread_index":      i + 1,
				"threads_completed": i + 1,
				"total_threads":     totalThreads,
				"success_count":     successCount,
				"error_count":       errorCount,
			})
		}

		// Base delay to avoid rate limiting
		if i < totalThreads-1 {
			time.Sleep(baseDelay)
		}
	}

	log.Printf("Forum scrape resumed and completed. Success: %d, Errors: %d", successCount, errorCount)
	s.activityService.CompleteTask(int64(activity.ID),
		fmt.Sprintf("Forum scrape complete. Success: %d, Errors: %d", successCount, errorCount))

	return nil
}

// saveCheckpoint saves the current scraping progress to the database
func (s *ScraperService) saveCheckpoint(activityID int64, checkpoint *ScrapeCheckpoint) error {
	// Serialize posts and links separately to avoid huge JSON
	postsJSON, err := json.Marshal(checkpoint.SavedPosts)
	if err != nil {
		return fmt.Errorf("failed to serialize posts: %w", err)
	}

	linksJSON, err := json.Marshal(checkpoint.SavedLinks)
	if err != nil {
		return fmt.Errorf("failed to serialize links: %w", err)
	}

	// Create checkpoint data with serialized posts/links
	checkpointData := map[string]interface{}{
		"current_page":          checkpoint.CurrentPage,
		"post_number":           checkpoint.PostNumber,
		"estimated_total_pages": checkpoint.EstimatedTotalPages,
		"posts_collected":       checkpoint.PostsCollected,
		"links_collected":       checkpoint.LinksCollected,
		"last_page_url":         checkpoint.LastPageURL,
		"clean_url":             checkpoint.CleanURL,
		"saved_posts":           string(postsJSON),
		"saved_links":           string(linksJSON),
	}

	checkpointJSON, err := json.Marshal(checkpointData)
	if err != nil {
		return fmt.Errorf("failed to serialize checkpoint: %w", err)
	}

	// Update activity checkpoint
	_, err = s.db.Exec(`
		UPDATE activities
		SET checkpoint = ?
		WHERE id = ?
	`, string(checkpointJSON), activityID)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	log.Printf("💾 Checkpoint saved: page %d, %d posts, %d links",
		checkpoint.CurrentPage, checkpoint.PostsCollected, checkpoint.LinksCollected)
	return nil
}

// loadCheckpoint loads a saved scraping checkpoint from the database
func (s *ScraperService) loadCheckpoint(activityID int64) (*ScrapeCheckpoint, error) {
	var checkpointJSON string
	err := s.db.QueryRow(`
		SELECT checkpoint
		FROM activities
		WHERE id = ?
	`, activityID).Scan(&checkpointJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No checkpoint found
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	if checkpointJSON == "" || checkpointJSON == "{}" {
		return nil, nil // Empty checkpoint
	}

	// Deserialize checkpoint data
	var checkpointData map[string]interface{}
	if err := json.Unmarshal([]byte(checkpointJSON), &checkpointData); err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint: %w", err)
	}

	checkpoint := &ScrapeCheckpoint{}

	// Extract basic fields
	if val, ok := checkpointData["current_page"].(float64); ok {
		checkpoint.CurrentPage = int(val)
	}
	if val, ok := checkpointData["post_number"].(float64); ok {
		checkpoint.PostNumber = int(val)
	}
	if val, ok := checkpointData["estimated_total_pages"].(float64); ok {
		checkpoint.EstimatedTotalPages = int(val)
	}
	if val, ok := checkpointData["posts_collected"].(float64); ok {
		checkpoint.PostsCollected = int(val)
	}
	if val, ok := checkpointData["links_collected"].(float64); ok {
		checkpoint.LinksCollected = int(val)
	}
	if val, ok := checkpointData["last_page_url"].(string); ok {
		checkpoint.LastPageURL = val
	}
	if val, ok := checkpointData["clean_url"].(string); ok {
		checkpoint.CleanURL = val
	}

	// Deserialize posts
	if postsStr, ok := checkpointData["saved_posts"].(string); ok && postsStr != "" {
		var posts []*models.ScrapedPost
		if err := json.Unmarshal([]byte(postsStr), &posts); err != nil {
			log.Printf("⚠️ Failed to deserialize saved posts: %v", err)
		} else {
			checkpoint.SavedPosts = posts
		}
	}

	// Deserialize links
	if linksStr, ok := checkpointData["saved_links"].(string); ok && linksStr != "" {
		var links []*models.ScrapedDownloadLink
		if err := json.Unmarshal([]byte(linksStr), &links); err != nil {
			log.Printf("⚠️ Failed to deserialize saved links: %v", err)
		} else {
			checkpoint.SavedLinks = links
		}
	}

	return checkpoint, nil
}

// clearCheckpoint removes the checkpoint data from the database
func (s *ScraperService) clearCheckpoint(activityID int64) error {
	_, err := s.db.Exec(`
		UPDATE activities
		SET checkpoint = '{}'
		WHERE id = ?
	`, activityID)

	if err != nil {
		return fmt.Errorf("failed to clear checkpoint: %w", err)
	}

	log.Printf("🗑️ Checkpoint cleared for activity %d", activityID)
	return nil
}
