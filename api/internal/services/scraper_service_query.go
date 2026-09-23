package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// GetThreadByURL checks if a thread exists and returns its current state
func (s *ScraperService) GetThreadByURL(url string, source string) (*models.ScrapedThread, error) {
	cleanURL := strings.TrimSuffix(url, "/")
	cleanURL = strings.TrimSuffix(cleanURL, "/unread")
	cleanURL = strings.TrimSuffix(cleanURL, "/latest")
	cleanURL = strings.TrimSuffix(cleanURL, "/")

	var thread models.ScrapedThread
	err := s.db.QueryRow(`
		SELECT id, external_id, source, title, url, author, category,
			   view_count, reply_count, post_count, download_count,
			   metadata, first_scraped_at, last_scraped_at, last_updated_at, created_at
		FROM scraped_threads
		WHERE url = ? AND source = ?
	`, cleanURL, source).Scan(
		&thread.ID, &thread.ExternalID, &thread.Source, &thread.Title,
		&thread.URL, &thread.Author, &thread.Category, &thread.ViewCount,
		&thread.ReplyCount, &thread.PostCount, &thread.DownloadCount,
		&thread.Metadata, &thread.FirstScrapedAt, &thread.LastScrapedAt,
		&thread.LastUpdatedAt, &thread.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Thread doesn't exist yet
	} else if err != nil {
		return nil, fmt.Errorf("failed to query thread: %w", err)
	}

	return &thread, nil
}

// ShouldRescrapeThread determines if a thread needs to be re-scraped
// Returns true if thread is new or has new activity
func (s *ScraperService) ShouldRescrapeThread(threadURL string, currentReplyCount int) (bool, *models.ScrapedThread, error) {
	existingThread, err := s.GetThreadByURL(threadURL, "simpcity")
	if err != nil {
		return false, nil, err
	}

	// Thread doesn't exist - needs scraping
	if existingThread == nil {
		return true, nil, nil
	}

	// Thread exists - check if there are new posts
	hasNewPosts := currentReplyCount > existingThread.ReplyCount

	return hasNewPosts, existingThread, nil
}

// GetStats returns scraper statistics
func (s *ScraperService) GetStats() (*models.ScraperStats, error) {
	stats := &models.ScraperStats{
		SourceBreakdown:   make(map[string]int),
		ProviderBreakdown: make(map[string]int),
		CategoryBreakdown: make(map[string]int),
	}

	// Total threads
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads").Scan(&stats.TotalThreads)

	// Total posts
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts").Scan(&stats.TotalPosts)

	// Total download links
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links").Scan(&stats.TotalDownloadLinks)

	// Active/dead links
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links WHERE status = 'active'").Scan(&stats.ActiveLinks)
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links WHERE status = 'dead'").Scan(&stats.DeadLinks)

	// Last scraped
	s.db.QueryRow("SELECT MAX(last_scraped_at) FROM scraped_threads").Scan(&stats.LastScrapedAt)

	// Source breakdown
	rows, _ := s.db.Query("SELECT source, COUNT(*) FROM scraped_threads GROUP BY source")
	defer rows.Close()
	for rows.Next() {
		var source string
		var count int
		rows.Scan(&source, &count)
		stats.SourceBreakdown[source] = count
	}

	// Provider breakdown
	rows, _ = s.db.Query("SELECT provider, COUNT(*) FROM scraped_download_links GROUP BY provider")
	defer rows.Close()
	for rows.Next() {
		var provider string
		var count int
		rows.Scan(&provider, &count)
		stats.ProviderBreakdown[provider] = count
	}

	// Category breakdown
	catRows, _ := s.db.Query("SELECT COALESCE(category,''), COUNT(*) FROM scraped_threads WHERE category != '' AND category IS NOT NULL GROUP BY category ORDER BY COUNT(*) DESC")
	defer catRows.Close()
	for catRows.Next() {
		var cat string
		var count int
		catRows.Scan(&cat, &count)
		if cat != "" {
			stats.CategoryBreakdown[cat] = count
		}
	}

	return stats, nil
}

// GetAllThreads retrieves all scraped threads with pagination, sorting, and filtering
func (s *ScraperService) GetAllThreads(limit, offset int, sortBy, provider, filter, author, category string) ([]*models.ScrapedThread, int, error) {
	// Build WHERE clause for filters
	whereClause := "WHERE 1=1"
	var args []interface{}

	// Provider filter (check download links for specific provider)
	if provider != "" {
		whereClause += " AND id IN (SELECT DISTINCT thread_id FROM scraped_download_links WHERE provider = ?)"
		args = append(args, provider)
	}

	// Content filter
	if filter == "has_downloads" {
		whereClause += " AND download_count > 0"
	} else if filter == "no_downloads" {
		whereClause += " AND download_count = 0"
	} else if filter == "watched" {
		whereClause += " AND COALESCE(is_watched, 0) = 1"
	} else if filter == "performer_in_db" {
		whereClause += " AND LOWER(author) IN (SELECT LOWER(name) FROM performers WHERE category = 'regular')"
	}

	// Author filter
	if author != "" {
		whereClause += " AND author LIKE ?"
		args = append(args, "%"+author+"%")
	}

	// Category filter
	if category != "" {
		whereClause += " AND category = ?"
		args = append(args, category)
	}

	// Get total count with filters
	var total int
	countQuery := "SELECT COUNT(*) FROM scraped_threads " + whereClause
	countArgs := args
	s.db.QueryRow(countQuery, countArgs...).Scan(&total)

	// Build ORDER BY clause
	orderClause := "ORDER BY last_scraped_at DESC"
	switch sortBy {
	case "date_desc":
		orderClause = "ORDER BY last_scraped_at DESC"
	case "date_asc":
		orderClause = "ORDER BY last_scraped_at ASC"
	case "title_asc":
		orderClause = "ORDER BY title COLLATE NOCASE ASC"
	case "title_desc":
		orderClause = "ORDER BY title COLLATE NOCASE DESC"
	case "views_desc":
		orderClause = "ORDER BY COALESCE(view_count, 0) DESC"
	case "views_asc":
		orderClause = "ORDER BY COALESCE(view_count, 0) ASC"
	case "replies_desc":
		orderClause = "ORDER BY COALESCE(reply_count, 0) DESC"
	case "replies_asc":
		orderClause = "ORDER BY COALESCE(reply_count, 0) ASC"
	case "downloads_desc":
		orderClause = "ORDER BY COALESCE(download_count, 0) DESC"
	case "downloads_asc":
		orderClause = "ORDER BY COALESCE(download_count, 0) ASC"
	}

	// Build final query
	query := "SELECT id, external_id, source, title, url, author, category, " +
		"view_count, reply_count, post_count, download_count, " +
		"metadata, first_scraped_at, last_scraped_at, last_updated_at, created_at, " +
		"COALESCE(is_watched, 0) " +
		"FROM scraped_threads " + whereClause + " " +
		orderClause + " " +
		"LIMIT ? OFFSET ?"

	args = append(args, limit, offset)

	// Debug logging
	log.Printf("GetAllThreads Query: %s", query)
	log.Printf("GetAllThreads Args: %v", args)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var threads []*models.ScrapedThread
	for rows.Next() {
		thread := &models.ScrapedThread{}
		err := rows.Scan(
			&thread.ID, &thread.ExternalID, &thread.Source, &thread.Title,
			&thread.URL, &thread.Author, &thread.Category, &thread.ViewCount,
			&thread.ReplyCount, &thread.PostCount, &thread.DownloadCount,
			&thread.Metadata, &thread.FirstScrapedAt, &thread.LastScrapedAt,
			&thread.LastUpdatedAt, &thread.CreatedAt, &thread.IsWatched,
		)
		if err != nil {
			log.Printf("Error scanning thread: %v", err)
			continue
		}
		thread.UnmarshalMetadata()
		threads = append(threads, thread)

		// Debug: Log first 10 threads when sorting by replies
		if sortBy == "replies_desc" && len(threads) <= 10 {
			titlePreview := thread.Title
			if len(titlePreview) > 50 {
				titlePreview = titlePreview[:50] + "..."
			}
			log.Printf("Thread #%d: ID=%d, ReplyCount=%d, ViewCount=%d, Title=%s",
				len(threads), thread.ID, thread.ReplyCount, thread.ViewCount, titlePreview)
		}
	}

	if sortBy == "replies_desc" && len(threads) > 0 {
		log.Printf("Total threads returned: %d", len(threads))
	}

	return threads, total, nil
}

// GetThreadByID retrieves a specific thread with its posts
func (s *ScraperService) GetThreadByID(id int64) (*models.ScrapedThread, error) {
	thread := &models.ScrapedThread{}
	err := s.db.QueryRow(`
		SELECT id, external_id, source, title, url, author, category,
			   view_count, reply_count, post_count, download_count,
			   metadata, first_scraped_at, last_scraped_at, last_updated_at, created_at,
			   COALESCE(is_watched, 0)
		FROM scraped_threads
		WHERE id = ?
	`, id).Scan(
		&thread.ID, &thread.ExternalID, &thread.Source, &thread.Title,
		&thread.URL, &thread.Author, &thread.Category, &thread.ViewCount,
		&thread.ReplyCount, &thread.PostCount, &thread.DownloadCount,
		&thread.Metadata, &thread.FirstScrapedAt, &thread.LastScrapedAt,
		&thread.LastUpdatedAt, &thread.CreatedAt, &thread.IsWatched,
	)
	if err != nil {
		return nil, err
	}

	thread.UnmarshalMetadata()
	return thread, nil
}

// ToggleThreadWatch toggles the watch status for a thread
func (s *ScraperService) ToggleThreadWatch(id int64) (bool, error) {
	// Get current watch status
	var currentStatus bool
	err := s.db.QueryRow("SELECT COALESCE(is_watched, 0) FROM scraped_threads WHERE id = ?", id).Scan(&currentStatus)
	if err != nil {
		return false, fmt.Errorf("thread not found: %w", err)
	}

	// Toggle the status
	newStatus := !currentStatus
	_, err = s.db.Exec("UPDATE scraped_threads SET is_watched = ? WHERE id = ?", newStatus, id)
	if err != nil {
		return false, err
	}

	return newStatus, nil
}

// GetWatchedThreads returns all threads marked as watched
func (s *ScraperService) GetWatchedThreads() ([]*models.ScrapedThread, error) {
	rows, err := s.db.Query(`
		SELECT id, external_id, source, title, url, author, category,
			   view_count, reply_count, post_count, download_count,
			   metadata, first_scraped_at, last_scraped_at, last_updated_at, created_at,
			   COALESCE(is_watched, 0)
		FROM scraped_threads
		WHERE is_watched = 1
		ORDER BY last_scraped_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []*models.ScrapedThread
	for rows.Next() {
		thread := &models.ScrapedThread{}
		err := rows.Scan(
			&thread.ID, &thread.ExternalID, &thread.Source, &thread.Title,
			&thread.URL, &thread.Author, &thread.Category, &thread.ViewCount,
			&thread.ReplyCount, &thread.PostCount, &thread.DownloadCount,
			&thread.Metadata, &thread.FirstScrapedAt, &thread.LastScrapedAt,
			&thread.LastUpdatedAt, &thread.CreatedAt, &thread.IsWatched,
		)
		if err != nil {
			continue
		}
		thread.UnmarshalMetadata()
		threads = append(threads, thread)
	}

	return threads, nil
}

// CheckWatchedThreadsForNewPosts checks all watched threads for new posts
func (s *ScraperService) CheckWatchedThreadsForNewPosts(activityID int) (*models.NewContentResult, error) {
	watchedThreads, err := s.GetWatchedThreads()
	if err != nil {
		return nil, err
	}

	if len(watchedThreads) == 0 {
		return &models.NewContentResult{
			ScraperType:   "simpcity",
			NewItemsFound: 0,
			CheckedAt:     time.Now(),
			Details:       map[string]interface{}{"message": "No watched threads"},
		}, nil
	}

	result := &models.NewContentResult{
		ScraperType: "simpcity",
		CheckedAt:   time.Now(),
		Details:     map[string]interface{}{},
		NewItems:    []models.NewItemInfo{},
	}

	totalNewPosts := 0
	threadsWithNewPosts := 0

	for _, thread := range watchedThreads {
		// Update activity progress
		if activityID > 0 {
			s.activityService.UpdateProgressLog(int64(activityID), 0,
				fmt.Sprintf("Checking thread: %s", thread.Title))
		}

		// Fetch current thread info to get latest reply count
		currentThread, err := s.ScrapeThread(thread.URL)
		if err != nil {
			log.Printf("Error checking thread %s: %v", thread.Title, err)
			continue
		}

		// Compare reply counts
		if currentThread.ReplyCount > thread.ReplyCount {
			newPosts := currentThread.ReplyCount - thread.ReplyCount
			totalNewPosts += newPosts
			threadsWithNewPosts++

			result.NewItems = append(result.NewItems, models.NewItemInfo{
				ID:    thread.ID,
				URL:   thread.URL,
				Title: thread.Title,
				Type:  "thread",
			})

			// Auto-scrape the new content
			log.Printf("Found %d new posts in thread: %s. Auto-scraping...", newPosts, thread.Title)
			go func(url string, aid int) {
				s.ScrapeThreadComplete(url, aid)
			}(thread.URL, activityID)
		}

		// Small delay between checks to avoid rate limiting
		time.Sleep(500 * time.Millisecond)
	}

	result.NewItemsFound = totalNewPosts
	result.Details["threads_checked"] = len(watchedThreads)
	result.Details["threads_with_new_posts"] = threadsWithNewPosts
	result.Details["total_new_posts"] = totalNewPosts

	return result, nil
}

// CheckForNewContent checks watched threads AND configured forum URLs for new content
func (s *ScraperService) CheckForNewContent(urls []string, maxPages int, activityID int) (*models.NewContentResult, error) {
	// Step 1: Always check watched threads
	result, err := s.CheckWatchedThreadsForNewPosts(activityID)
	if err != nil {
		return nil, err
	}

	// Step 2: Also check configured forum URLs for new threads
	if len(urls) > 0 {
		for _, forumURL := range urls {
			if activityID > 0 {
				s.activityService.UpdateProgressLog(int64(activityID), 0,
					fmt.Sprintf("Checking forum URL: %s", forumURL))
			}

			threads, err := s.ScrapeForumCategory(forumURL, activityID)
			if err != nil {
				log.Printf("Error scraping forum URL %s: %v", forumURL, err)
				continue
			}

			// Check each thread - see if it's new or has new posts
			newFromURL := 0
			for _, threadInfo := range threads {
				// Check if thread already exists in DB
				existing, _ := s.GetThreadByURL(threadInfo.URL, "simpcity")
				if existing == nil {
					// New thread found - auto-scrape it
					newFromURL++
					result.NewItems = append(result.NewItems, models.NewItemInfo{
						URL:   threadInfo.URL,
						Title: threadInfo.Title,
						Type:  "thread",
					})
					go func(url string, aid int) {
						s.ScrapeThreadComplete(url, aid)
					}(threadInfo.URL, activityID)
				}
			}

			if newFromURL > 0 {
				result.NewItemsFound += newFromURL
				log.Printf("Found %d new threads from forum URL: %s", newFromURL, forumURL)
			}

			time.Sleep(1 * time.Second)
		}

		result.Details["forum_urls_checked"] = len(urls)
	}

	return result, nil
}

// SearchThreads searches threads by title or content with sorting and filtering
func (s *ScraperService) SearchThreads(query string, limit, offset int, sortBy, provider, filter, author, category string) ([]*models.ScrapedThread, int, error) {
	searchPattern := "%" + query + "%"

	// Build WHERE clause
	whereClause := "WHERE (title LIKE ? OR author LIKE ?)"
	args := []interface{}{searchPattern, searchPattern}

	if provider != "" {
		whereClause += " AND id IN (SELECT DISTINCT thread_id FROM scraped_download_links WHERE provider = ?)"
		args = append(args, provider)
	}
	if filter == "has_downloads" {
		whereClause += " AND download_count > 0"
	} else if filter == "no_downloads" {
		whereClause += " AND download_count = 0"
	} else if filter == "watched" {
		whereClause += " AND COALESCE(is_watched, 0) = 1"
	} else if filter == "performer_in_db" {
		whereClause += " AND LOWER(author) IN (SELECT LOWER(name) FROM performers WHERE category = 'regular')"
	}
	if author != "" {
		whereClause += " AND author LIKE ?"
		args = append(args, "%"+author+"%")
	}
	if category != "" {
		whereClause += " AND category = ?"
		args = append(args, category)
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM scraped_threads " + whereClause
	s.db.QueryRow(countQuery, args...).Scan(&total)

	// Build ORDER BY clause
	orderClause := "ORDER BY last_scraped_at DESC"
	switch sortBy {
	case "date_desc":
		orderClause = "ORDER BY last_scraped_at DESC"
	case "date_asc":
		orderClause = "ORDER BY last_scraped_at ASC"
	case "title_asc":
		orderClause = "ORDER BY title COLLATE NOCASE ASC"
	case "title_desc":
		orderClause = "ORDER BY title COLLATE NOCASE DESC"
	case "views_desc":
		orderClause = "ORDER BY COALESCE(view_count, 0) DESC"
	case "views_asc":
		orderClause = "ORDER BY COALESCE(view_count, 0) ASC"
	case "replies_desc":
		orderClause = "ORDER BY COALESCE(reply_count, 0) DESC"
	case "replies_asc":
		orderClause = "ORDER BY COALESCE(reply_count, 0) ASC"
	case "downloads_desc":
		orderClause = "ORDER BY COALESCE(download_count, 0) DESC"
	case "downloads_asc":
		orderClause = "ORDER BY COALESCE(download_count, 0) ASC"
	}

	selectQuery := "SELECT id, external_id, source, title, url, author, category, " +
		"view_count, reply_count, post_count, download_count, " +
		"metadata, first_scraped_at, last_scraped_at, last_updated_at, created_at, " +
		"COALESCE(is_watched, 0) " +
		"FROM scraped_threads " + whereClause + " " +
		orderClause + " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var threads []*models.ScrapedThread
	for rows.Next() {
		thread := &models.ScrapedThread{}
		err := rows.Scan(
			&thread.ID, &thread.ExternalID, &thread.Source, &thread.Title,
			&thread.URL, &thread.Author, &thread.Category, &thread.ViewCount,
			&thread.ReplyCount, &thread.PostCount, &thread.DownloadCount,
			&thread.Metadata, &thread.FirstScrapedAt, &thread.LastScrapedAt,
			&thread.LastUpdatedAt, &thread.CreatedAt, &thread.IsWatched,
		)
		if err != nil {
			continue
		}
		thread.UnmarshalMetadata()
		threads = append(threads, thread)
	}

	return threads, total, nil
}

// GetPostsByThreadID retrieves all posts for a thread
func (s *ScraperService) GetPostsByThreadID(threadID int64) ([]*models.ScrapedPost, error) {
	rows, err := s.db.Query(`
		SELECT id, thread_id, external_id, author, content, plain_text, posted_at,
		       metadata, created_at
		FROM scraped_posts
		WHERE thread_id = ?
		ORDER BY posted_at ASC
	`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.ScrapedPost
	for rows.Next() {
		post := &models.ScrapedPost{}
		err := rows.Scan(
			&post.ID, &post.ThreadID, &post.ExternalID, &post.Author,
			&post.Content, &post.PlainText, &post.PostedAt, &post.Metadata, &post.CreatedAt,
		)
		if err != nil {
			continue
		}
		post.UnmarshalMetadata()
		posts = append(posts, post)
	}

	return posts, nil
}

// GetRecentPostsByThreadID retrieves the most recent posts for a thread, including read status
func (s *ScraperService) GetRecentPostsByThreadID(threadID int64, limit int) ([]*models.ScrapedPost, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
		SELECT id, thread_id, external_id, author, plain_text, metadata, is_read, posted_at, created_at
		FROM scraped_posts
		WHERE thread_id = ?
		ORDER BY posted_at DESC
		LIMIT ?
	`, threadID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.ScrapedPost
	for rows.Next() {
		post := &models.ScrapedPost{}
		var metadataStr string
		err := rows.Scan(
			&post.ID, &post.ThreadID, &post.ExternalID, &post.Author,
			&post.PlainText, &metadataStr, &post.IsRead, &post.PostedAt, &post.CreatedAt,
		)
		if err != nil {
			continue
		}
		// Parse metadata to expose attachments and extracted links
		if metadataStr != "" {
			var meta models.ScrapedPostMetadata
			if jsonErr := json.Unmarshal([]byte(metadataStr), &meta); jsonErr == nil {
				post.MetadataObj = &meta
			}
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// MarkPostAsRead marks a single post as read
func (s *ScraperService) MarkPostAsRead(postID int64) error {
	_, err := s.db.Exec("UPDATE scraped_posts SET is_read = 1 WHERE id = ?", postID)
	return err
}

// GetDownloadLinksByThreadID retrieves all download links for a thread
func (s *ScraperService) GetDownloadLinksByThreadID(threadID int64) ([]*models.ScrapedDownloadLink, error) {
	rows, err := s.db.Query(`
		SELECT id, thread_id, post_id, source, provider, url, original_url,
			   filename, file_size, file_type, status, download_status,
			   metadata, discovered_at, last_checked_at, created_at
		FROM scraped_download_links
		WHERE thread_id = ?
		ORDER BY discovered_at ASC
	`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*models.ScrapedDownloadLink
	for rows.Next() {
		link := &models.ScrapedDownloadLink{}
		err := rows.Scan(
			&link.ID, &link.ThreadID, &link.PostID, &link.Source,
			&link.Provider, &link.URL, &link.OriginalURL,
			&link.Filename, &link.FileSize, &link.FileType,
			&link.Status, &link.DownloadStatus,
			&link.Metadata, &link.DiscoveredAt, &link.LastCheckedAt, &link.CreatedAt,
		)
		if err != nil {
			continue
		}
		link.UnmarshalMetadata()
		links = append(links, link)
	}

	return links, nil
}
