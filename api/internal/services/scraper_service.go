package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// ScraperService handles web scraping operations
type ScraperService struct {
	db                  *sql.DB
	activityService     *ActivityService
	aiCompanionService  *AICompanionService
	notificationService *NotificationService
	httpClient          *http.Client
	sessionCookie       string // Store session cookie for authenticated requests
}

// AdaptiveRateLimiter tracks server response times and adjusts delays
type AdaptiveRateLimiter struct {
	currentDelay    time.Duration
	minDelay        time.Duration
	maxDelay        time.Duration
	responseHistory []time.Duration
	historySize     int
}

// NewScraperService creates a new scraper service
func NewScraperService(activitySvc *ActivityService, aiCompanionSvc *AICompanionService) *ScraperService {
	db := database.GetDB()
	svc := &ScraperService{
		db:                  db,
		activityService:     activitySvc,
		aiCompanionService:  aiCompanionSvc,
		notificationService: NewNotificationService(db),
		sessionCookie:       "",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}

	// Load session cookie from database
	svc.loadSessionCookieFromDB()

	return svc
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

// ScrapeThread scrapes a single thread from simpcity.is
func (s *ScraperService) ScrapeThread(threadURL string) (*models.ScrapedThread, error) {
	// Clean the URL: remove common suffixes like /unread, /latest
	cleanURL := strings.TrimSuffix(threadURL, "/")
	cleanURL = strings.TrimSuffix(cleanURL, "/unread")
	cleanURL = strings.TrimSuffix(cleanURL, "/latest")
	cleanURL = strings.TrimSuffix(cleanURL, "/")

	log.Printf("Scraping thread: %s", cleanURL)

	// Parse URL to extract thread ID
	threadID, err := s.extractThreadID(cleanURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract thread ID: %w", err)
	}

	// Retry logic for server errors (500, 502, 503)
	var resp *http.Response
	var doc *goquery.Document
	maxRetries := 3
	retryDelay := 10 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create request with authentication
		req, reqErr := http.NewRequest("GET", cleanURL, nil)
		if reqErr != nil {
			return nil, fmt.Errorf("failed to create request: %w", reqErr)
		}

		// Add session cookie if available
		if s.sessionCookie != "" {
			req.Header.Set("Cookie", s.sessionCookie)
		}

		// Add user agent to mimic browser
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		// Fetch the page
		resp, err = s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch thread: %w", err)
		}

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			resp.Body.Close()
			return nil, fmt.Errorf("authentication required - please set session cookie")
		}

		// Handle server errors with retry
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			resp.Body.Close()
			if attempt < maxRetries {
				waitTime := retryDelay * time.Duration(attempt+1)
				log.Printf("Server error (%d). Waiting %v before retry %d/%d", resp.StatusCode, waitTime, attempt+1, maxRetries)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("server error after %d retries: %d", maxRetries, resp.StatusCode)
		}

		// Handle rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt < maxRetries {
				waitTime := retryDelay * time.Duration(attempt+1)
				log.Printf("Rate limited (429). Waiting %v before retry %d/%d", waitTime, attempt+1, maxRetries)
				time.Sleep(waitTime)
				continue
			}
			return nil, fmt.Errorf("rate limited after %d retries", maxRetries)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Success - parse HTML
		doc, err = goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTML: %w", err)
		}

		// Break out of retry loop on success
		break
	}

	// Extract thread information
	thread := &models.ScrapedThread{
		ExternalID:     threadID,
		Source:         "simpcity",
		URL:            cleanURL,
		FirstScrapedAt: time.Now(),
		LastScrapedAt:  time.Now(),
		MetadataObj:    &models.ScrapedThreadMetadata{},
	}

	// Extract title (raw)
	rawTitle := strings.TrimSpace(doc.Find("h1.p-title-value").First().Text())

	// Extract tags from title BEFORE cleaning
	extractedTags := s.ExtractTagsFromTitle(rawTitle)

	// Clean title from tags and set it
	thread.Title = s.CleanTitleFromTags(rawTitle)

	// Extract author
	thread.Author = strings.TrimSpace(doc.Find(".p-description .username").First().Text())

	// Extract category
	thread.Category = strings.TrimSpace(doc.Find(".p-breadcrumbs li:nth-last-child(2) a").First().Text())

	// Extract view count and reply count from meta info
	doc.Find("dl.pairs").Each(func(i int, sel *goquery.Selection) {
		dt := strings.TrimSpace(sel.Find("dt").Text())
		dd := strings.TrimSpace(sel.Find("dd").Text())

		if strings.Contains(dt, "Views") {
			fmt.Sscanf(dd, "%d", &thread.ViewCount)
		} else if strings.Contains(dt, "Replies") {
			fmt.Sscanf(dd, "%d", &thread.ReplyCount)
		}
	})

	// Extract all thumbnails (images) for fallback support
	var thumbnails []string
	doc.Find(".message-main img").Each(func(i int, sel *goquery.Selection) {
		if imgURL, exists := sel.Attr("src"); exists {
			// Filter out smilies, avatars, and other non-content images
			if !strings.Contains(imgURL, "/smilies/") &&
				!strings.Contains(imgURL, "/avatars/") &&
				!strings.Contains(imgURL, "data:image") {
				thumbnails = append(thumbnails, imgURL)
			}
		}
	})

	// Set primary thumbnail and all thumbnails
	if len(thumbnails) > 0 {
		thread.MetadataObj.ThumbnailURL = thumbnails[0]
		thread.MetadataObj.ThumbnailURLs = thumbnails
	}

	// Combine extracted tags with forum tags
	thread.MetadataObj.Tags = extractedTags

	// Also get tags from the forum's tag list
	doc.Find(".tagList a").Each(func(i int, sel *goquery.Selection) {
		tag := strings.TrimSpace(sel.Text())
		if tag != "" && !contains(thread.MetadataObj.Tags, tag) {
			thread.MetadataObj.Tags = append(thread.MetadataObj.Tags, tag)
		}
	})

	// Check if pinned or locked
	thread.MetadataObj.IsPinned = doc.Find(".structItem--sticky").Length() > 0
	thread.MetadataObj.IsLocked = doc.Find(".structItem--locked").Length() > 0

	// Extract performer and studio names from CLEANED title
	thread.MetadataObj.PerformerNames = s.extractPerformerNames(thread.Title)
	thread.MetadataObj.StudioNames = s.extractStudioNames(thread.Title)

	// Count posts
	thread.PostCount = doc.Find(".message--post").Length()

	return thread, nil
}

// ScrapePosts scrapes all posts from a thread, handling pagination
func (s *ScraperService) ScrapePosts(threadURL string, threadID int64, activityID ...int) ([]*models.ScrapedPost, []*models.ScrapedDownloadLink, error) {
	// Extract activity ID if provided (optional parameter)
	var aid int
	if len(activityID) > 0 {
		aid = activityID[0]
	}
	// Clean the URL first: remove trailing slash and common suffixes like /unread, /latest
	cleanURL := strings.TrimSuffix(threadURL, "/")
	cleanURL = strings.TrimSuffix(cleanURL, "/unread")
	cleanURL = strings.TrimSuffix(cleanURL, "/latest")
	cleanURL = strings.TrimSuffix(cleanURL, "/")

	log.Printf("Scraping posts from thread: %s", cleanURL)

	var allPosts []*models.ScrapedPost
	var allDownloadLinks []*models.ScrapedDownloadLink
	postNumber := 1
	currentPage := 1
	estimatedTotalPages := 0 // Will be updated after first page scrape
	var existingPostIDs map[string]bool // Track existing posts for incremental scraping

	// Initialize adaptive rate limiter
	rateLimiter := NewAdaptiveRateLimiter()
	log.Printf("⚡ Adaptive rate limiting enabled: %.1fs initial delay", rateLimiter.GetDelay().Seconds())

	// Check for existing posts in this thread for incremental scraping
	existingPostIDs, err := s.getExistingPostIDs(threadID)
	if err != nil {
		log.Printf("⚠️ Failed to load existing posts: %v (continuing anyway)", err)
		existingPostIDs = make(map[string]bool)
	} else if len(existingPostIDs) > 0 {
		log.Printf("📊 Found %d existing posts in database - will skip duplicates", len(existingPostIDs))
		if aid > 0 {
			s.activityService.UpdateProgressLog(int64(aid), 0,
				fmt.Sprintf("📊 Incremental mode: %d posts already in database", len(existingPostIDs)))
		}
	}

	// Check for existing checkpoint to resume from
	if aid > 0 {
		checkpoint, err := s.loadCheckpoint(int64(aid))
		if err == nil && checkpoint != nil {
			currentPage = checkpoint.CurrentPage
			postNumber = checkpoint.PostNumber
			estimatedTotalPages = checkpoint.EstimatedTotalPages
			allPosts = checkpoint.SavedPosts
			allDownloadLinks = checkpoint.SavedLinks
			log.Printf("🔄 Resuming scrape from page %d (had %d posts, %d links)", currentPage, len(allPosts), len(allDownloadLinks))
			s.activityService.UpdateProgressLog(int64(aid), 0,
				fmt.Sprintf("🔄 Resuming from page %d/%d (recovered %d posts, %d links)",
					currentPage, estimatedTotalPages, len(allPosts), len(allDownloadLinks)))
		}
	}

	for {
		// Check if task has been paused
		if aid > 0 {
			currentActivity, err := s.activityService.GetByID(int64(aid))
			if err == nil && currentActivity.IsPaused {
				log.Printf("⏸️ Task paused during post scraping at page %d", currentPage)
				return allPosts, allDownloadLinks, fmt.Errorf("task paused by user at page %d", currentPage)
			}
		}

		// Construct URL for current page
		pageURL := cleanURL
		if currentPage > 1 {
			pageURL = fmt.Sprintf("%s/page-%d", cleanURL, currentPage)
		}

		log.Printf("Scraping page %d: %s", currentPage, pageURL)
		if aid > 0 {
			s.activityService.UpdateProgressLog(int64(aid), 0, fmt.Sprintf("Scraping page %d", currentPage))
		}

		// Enhanced retry logic with exponential backoff
		var resp *http.Response
		var doc *goquery.Document
		var err error
		maxRetries := 5 // Increased from 3 to 5
		baseRetryDelay := 3 * time.Second

		for attempt := 0; attempt <= maxRetries; attempt++ {
			// Create request with authentication
			req, reqErr := http.NewRequest("GET", pageURL, nil)
			if reqErr != nil {
				return nil, nil, fmt.Errorf("failed to create request: %w", reqErr)
			}

			// Add session cookie if available
			if s.sessionCookie != "" {
				req.Header.Set("Cookie", s.sessionCookie)
			}

			// Add user agent
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

			// Track response time for adaptive rate limiting
			requestStart := time.Now()
			resp, err = s.httpClient.Do(req)
			responseTime := time.Since(requestStart)
			if err != nil {
				// Network error - retry with backoff
				if attempt < maxRetries {
					waitTime := baseRetryDelay * time.Duration(1<<uint(attempt)) // Exponential: 3s, 6s, 12s, 24s, 48s
					log.Printf("⚠️ Network error on page %d: %v. Retrying in %v (%d/%d)", currentPage, err, waitTime, attempt+1, maxRetries)
					if aid > 0 {
						s.activityService.UpdateProgressLog(int64(aid), 0, fmt.Sprintf("⚠️ Network error. Retrying in %v (%d/%d)", waitTime, attempt+1, maxRetries))
					}
					time.Sleep(waitTime)
					continue
				}
				return nil, nil, fmt.Errorf("network error after %d retries: %w", maxRetries, err)
			}

			if resp.StatusCode == http.StatusNotFound {
				// No more pages
				resp.Body.Close()
				break
			}

			// Handle rate limiting with exponential backoff
			if resp.StatusCode == http.StatusTooManyRequests {
				resp.Body.Close()
				// Record rate limit for adaptive adjustment
				rateLimiter.RecordResponse(responseTime, resp.StatusCode)
				if attempt < maxRetries {
					waitTime := baseRetryDelay * time.Duration(1<<uint(attempt)) // Exponential backoff
					log.Printf("⚠️ Rate limited (429) on page %d. Waiting %v before retry %d/%d", currentPage, waitTime, attempt+1, maxRetries)
					if aid > 0 {
						s.activityService.UpdateProgressLog(int64(aid), 0, fmt.Sprintf("⚠️ Rate limited. Waiting %v before retry %d/%d", waitTime, attempt+1, maxRetries))
					}
					time.Sleep(waitTime)
					continue
				}
				return nil, nil, fmt.Errorf("rate limited after %d retries (429)", maxRetries)
			}

			// Handle server errors (500, 502, 503) with retry
			if resp.StatusCode >= 500 && resp.StatusCode < 600 {
				resp.Body.Close()
				// Record server error for adaptive adjustment
				rateLimiter.RecordResponse(responseTime, resp.StatusCode)
				if attempt < maxRetries {
					waitTime := baseRetryDelay * time.Duration(1<<uint(attempt))
					log.Printf("⚠️ Server error %d on page %d. Retrying in %v (%d/%d)", resp.StatusCode, currentPage, waitTime, attempt+1, maxRetries)
					if aid > 0 {
						s.activityService.UpdateProgressLog(int64(aid), 0, fmt.Sprintf("⚠️ Server error %d. Retrying in %v (%d/%d)", resp.StatusCode, waitTime, attempt+1, maxRetries))
					}
					time.Sleep(waitTime)
					continue
				}
				return nil, nil, fmt.Errorf("server error %d after %d retries", resp.StatusCode, maxRetries)
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			// Record successful response for adaptive rate limiting
			rateLimiter.RecordResponse(responseTime, resp.StatusCode)

			// Success - break out of retry loop
			break
		}

		doc, err = goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse HTML: %w", err)
		}

		// Extract posts from current page with duplicate detection and validation
		postsOnPage := 0
		newPostsOnPage := 0
		skippedPostsOnPage := 0
		invalidPostsOnPage := 0
		doc.Find(".message--post").Each(func(i int, sel *goquery.Selection) {
			post := s.extractPost(sel, threadID, postNumber)
			if post != nil {
				postsOnPage++

				// Check if this post already exists (incremental scraping)
				if existingPostIDs[post.ExternalID] {
					skippedPostsOnPage++
					postNumber++
					return // Skip this post, already in database
				}

				// Validate post content quality
				if !s.isValidPost(post) {
					invalidPostsOnPage++
					postNumber++
					return // Skip invalid/empty post
				}

				// Extract download links from post content and store temporarily with post index
				links := s.extractDownloadLinks(post.Content, threadID, int64(len(allPosts))) // Use index as temporary marker

				// Deduplicate links within this post
				links = s.deduplicateLinks(links)

				allPosts = append(allPosts, post)
				allDownloadLinks = append(allDownloadLinks, links...)
				newPostsOnPage++
				postNumber++
			}
		})

		// Enhanced logging with validation and duplicate detection stats
		logMsg := fmt.Sprintf("Found %d posts on page %d", postsOnPage, currentPage)
		if skippedPostsOnPage > 0 || invalidPostsOnPage > 0 {
			logMsg = fmt.Sprintf("Found %d posts on page %d (%d new, %d duplicates, %d invalid)",
				postsOnPage, currentPage, newPostsOnPage, skippedPostsOnPage, invalidPostsOnPage)
		}
		log.Printf(logMsg)

		// Detect total pages from pagination (only on first page)
		if currentPage == 1 && estimatedTotalPages == 0 {
			// Look for the last page number in pagination
			maxPageNum := 1
			doc.Find(".pageNav-page").Each(func(i int, sel *goquery.Selection) {
				pageNumText := strings.TrimSpace(sel.Text())
				if pageNumText != "" {
					var num int
					_, err := fmt.Sscanf(pageNumText, "%d", &num)
					if err == nil && num > maxPageNum {
						maxPageNum = num
					}
				}
			})
			estimatedTotalPages = maxPageNum
			if estimatedTotalPages > 1 {
				log.Printf("📄 Detected %d total pages in thread", estimatedTotalPages)
				if aid > 0 {
					s.activityService.UpdateProgressLog(int64(aid), 0, fmt.Sprintf("📄 Thread has %d pages to scrape", estimatedTotalPages))
				}
			}
		}

		// Enhanced progress logging with page count
		progressMsg := fmt.Sprintf("Found %d posts on page %d", postsOnPage, currentPage)
		if estimatedTotalPages > 1 {
			progressMsg = fmt.Sprintf("Found %d posts on page %d/%d", postsOnPage, currentPage, estimatedTotalPages)
		}
		if aid > 0 {
			s.activityService.UpdateProgressLog(int64(aid), 0, progressMsg)
		}

		// Stop if no posts found on this page
		if postsOnPage == 0 {
			log.Printf("No posts found on page %d, stopping pagination", currentPage)
			break
		}

		// Check if there's a next page by looking for pagination elements
		hasNextPage := false

		// Method 1: Check for "Next" button (most reliable)
		nextButton := doc.Find(".pageNav-jump--next")
		if nextButton.Length() > 0 && !nextButton.HasClass("is-disabled") {
			hasNextPage = true
			log.Printf("Found active 'Next' button - more pages available")
		}

		// Method 2: Check for page numbers greater than current page
		if !hasNextPage {
			doc.Find(".pageNav-page").Each(func(i int, sel *goquery.Selection) {
				pageNumText := strings.TrimSpace(sel.Text())
				if pageNumText != "" {
					var num int
					_, err := fmt.Sscanf(pageNumText, "%d", &num)
					if err == nil && num > currentPage {
						hasNextPage = true
						log.Printf("Found page number %d (current: %d) - more pages available", num, currentPage)
					}
				}
			})
		}

		// Method 3: Check if current page is marked as "last" page
		if !hasNextPage {
			// Check if we're on the last page by looking for the current page in pagination
			isLastPage := false
			doc.Find(".pageNav-page").Each(func(i int, sel *goquery.Selection) {
				if sel.HasClass("pageNav-page--current") {
					// Found current page marker - check if there's no next page link after it
					pageNumText := strings.TrimSpace(sel.Text())
					var num int
					_, err := fmt.Sscanf(pageNumText, "%d", &num)
					if err == nil && num == currentPage {
						// We are on this page - if Next button is disabled, this is the last page
						if doc.Find(".pageNav-jump--next.is-disabled").Length() > 0 {
							isLastPage = true
							log.Printf("Current page %d is marked as last page", currentPage)
						}
					}
				}
			})

			if isLastPage {
				log.Printf("No more pages to scrape (last page: %d)", currentPage)
				break
			}
		}

		if !hasNextPage {
			// CRITICAL FIX: Don't trust HEAD requests - simpcity returns 200 for non-existent pages
			// If pagination UI doesn't show a next page, we should stop here
			log.Printf("No pagination indicator found - stopping at page %d", currentPage)
			break
		}

		// Save checkpoint after successful page scrape
		if aid > 0 {
			checkpoint := &ScrapeCheckpoint{
				CurrentPage:         currentPage + 1, // Next page to scrape
				PostNumber:          postNumber,
				EstimatedTotalPages: estimatedTotalPages,
				PostsCollected:      len(allPosts),
				LinksCollected:      len(allDownloadLinks),
				LastPageURL:         pageURL,
				CleanURL:            cleanURL,
				SavedPosts:          allPosts,
				SavedLinks:          allDownloadLinks,
			}
			if err := s.saveCheckpoint(int64(aid), checkpoint); err != nil {
				log.Printf("⚠️ Failed to save checkpoint: %v", err)
			}
		}

		currentPage++
		nextPageMsg := fmt.Sprintf("Moving to page %d", currentPage)
		if estimatedTotalPages > 1 {
			nextPageMsg = fmt.Sprintf("Moving to page %d/%d", currentPage, estimatedTotalPages)
		}
		log.Printf(nextPageMsg)

		// Adaptive delay based on server performance
		adaptiveDelay := rateLimiter.GetDelay()
		statusMsg := rateLimiter.GetStatus()
		log.Printf("%s - waiting %.1fs before next page", statusMsg, adaptiveDelay.Seconds())
		if aid > 0 && currentPage%10 == 0 { // Update status every 10 pages
			s.activityService.UpdateProgressLog(int64(aid), 0, statusMsg)
		}
		time.Sleep(adaptiveDelay)
	}

	// Final summary with duplicate detection statistics
	summaryMsg := fmt.Sprintf("✅ Scraping complete: %d posts, %d download links across %d pages", len(allPosts), len(allDownloadLinks), currentPage)
	if len(existingPostIDs) > 0 {
		summaryMsg = fmt.Sprintf("✅ Scraping complete: %d new posts, %d download links across %d pages (%d posts already in database)",
			len(allPosts), len(allDownloadLinks), currentPage, len(existingPostIDs))
	}
	log.Printf(summaryMsg)
	if aid > 0 {
		s.activityService.UpdateProgressLog(int64(aid), 0, summaryMsg)
		// Clear checkpoint on successful completion
		if err := s.clearCheckpoint(int64(aid)); err != nil {
			log.Printf("⚠️ Failed to clear checkpoint: %v", err)
		}
	}

	return allPosts, allDownloadLinks, nil
}

// extractPost extracts a single post from a goquery selection
func (s *ScraperService) extractPost(sel *goquery.Selection, threadID int64, postNumber int) *models.ScrapedPost {
	post := &models.ScrapedPost{
		ThreadID:    threadID,
		Source:      "simpcity",
		PostNumber:  postNumber,
		ScrapedAt:   time.Now(),
		MetadataObj: &models.ScrapedPostMetadata{},
	}

	// Extract post ID
	if postID, exists := sel.Attr("data-content"); exists {
		post.ExternalID = strings.TrimPrefix(postID, "post-")
	}

	// Extract author
	post.Author = strings.TrimSpace(sel.Find(".message-name .username").Text())

	// Extract content (HTML)
	contentHTML, _ := sel.Find(".message-body .bbWrapper").Html()
	post.Content = contentHTML

	// Extract plain text
	post.PlainText = strings.TrimSpace(sel.Find(".message-body .bbWrapper").Text())

	// Extract like count
	likeText := strings.TrimSpace(sel.Find(".reactionsBar-link").Text())
	fmt.Sscanf(likeText, "%d", &post.LikeCount)

	// Extract post date
	if dateStr, exists := sel.Find(".message-attribution-main time").Attr("datetime"); exists {
		if parsedTime, err := time.Parse(time.RFC3339, dateStr); err == nil {
			post.PostedAt = parsedTime
		}
	}

	// Extract attachments from attachment section
	sel.Find(".message-attachments .attachment").Each(func(i int, attSel *goquery.Selection) {
		attachment := models.Attachment{}
		if imgURL, exists := attSel.Find("img").Attr("src"); exists {
			attachment.Type = "image"
			attachment.URL = imgURL
			attachment.ThumbnailURL, _ = attSel.Find("img").Attr("data-thumbnail")
			post.MetadataObj.Attachments = append(post.MetadataObj.Attachments, attachment)
		}
	})

	// Extract images from post content
	sel.Find(".message-body .bbWrapper img").Each(func(i int, imgSel *goquery.Selection) {
		attachment := models.Attachment{}
		if imgURL, exists := imgSel.Attr("src"); exists {
			attachment.Type = "image"
			attachment.URL = imgURL
			// Check for data-url (full size) or use src
			if dataURL, hasData := imgSel.Attr("data-url"); hasData {
				attachment.URL = dataURL
				attachment.ThumbnailURL = imgURL
			}
			post.MetadataObj.Attachments = append(post.MetadataObj.Attachments, attachment)
		}
	})

	// Store raw HTML
	post.MetadataObj.RawHTML = contentHTML

	return post
}

// extractDownloadLinks extracts download links from post content
func (s *ScraperService) extractDownloadLinks(content string, threadID int64, postID int64) []*models.ScrapedDownloadLink {
	var links []*models.ScrapedDownloadLink

	// Regular expressions for different hosting providers - more flexible patterns
	providers := map[string]*regexp.Regexp{
		"gofile":     regexp.MustCompile(`https?://(?:www\.)?gofile\.io/d/[a-zA-Z0-9_-]+`),
		"pixeldrain": regexp.MustCompile(`https?://(?:www\.)?pixeldrain\.com/(?:u|l)/[a-zA-Z0-9_-]+`),
		"bunkr":      regexp.MustCompile(`https?://(?:www\.)?(?:bunkr|bunkrr)\.[a-z]+/[a-z]/[a-zA-Z0-9_-]+`),
		"cyberdrop":  regexp.MustCompile(`https?://(?:www\.)?cyberdrop\.(?:me|to|cc)/a/[a-zA-Z0-9_-]+`),
		"mediafire":  regexp.MustCompile(`https?://(?:www\.)?mediafire\.com/(?:file|folder)/[a-zA-Z0-9_/-]+`),
		"mega":       regexp.MustCompile(`https?://(?:www\.)?mega\.nz/(?:file|folder)/[a-zA-Z0-9#_-]+`),
	}

	for provider, regex := range providers {
		matches := regex.FindAllString(content, -1)
		for _, match := range matches {
			link := &models.ScrapedDownloadLink{
				ThreadID:      threadID,
				PostID:        postID,
				Source:        "simpcity",
				Provider:      provider,
				URL:           match,
				OriginalURL:   match,
				Status:        "active",
				DiscoveredAt:  time.Now(),
				MetadataObj:   &models.DownloadLinkMetadata{},
			}
			links = append(links, link)
		}
	}

	// Log if no links found
	// Note: Removed individual post logging here as it's too verbose
	// The page-level summary in ScrapePosts is sufficient

	return links
}

// extractThreadID extracts the thread ID from URL
func (s *ScraperService) extractThreadID(threadURL string) (string, error) {
	parsedURL, err := url.Parse(threadURL)
	if err != nil {
		return "", err
	}

	// Extract from path like /threads/thread-name.123456/
	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid thread URL format")
	}

	threadID := strings.TrimSuffix(parts[len(parts)-1], "/")
	return threadID, nil
}

// extractPerformerNames attempts to extract performer names from title
func (s *ScraperService) extractPerformerNames(title string) []string {
	var performers []string

	// Common patterns: [Name], (Name), Name - Content
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\[([^\]]+)\]`),
		regexp.MustCompile(`\(([^)]+)\)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(title, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				// Filter out common non-name terms
				if !s.isCommonTerm(name) {
					performers = append(performers, name)
				}
			}
		}
	}

	return performers
}

// extractStudioNames attempts to extract studio names from title
func (s *ScraperService) extractStudioNames(title string) []string {
	var studios []string

	// Common studio indicators
	studioKeywords := []string{"Official", "Network", "Productions", "Studios", "Entertainment"}

	words := strings.Fields(title)
	for i, word := range words {
		for _, keyword := range studioKeywords {
			if strings.Contains(word, keyword) && i > 0 {
				studios = append(studios, words[i-1]+" "+word)
			}
		}
	}

	return studios
}

// isCommonTerm checks if a term is a common non-name word
func (s *ScraperService) isCommonTerm(term string) bool {
	commonTerms := []string{"NEW", "HD", "4K", "LEAKED", "EXCLUSIVE", "UPDATED", "MEGA", "PACK"}
	termUpper := strings.ToUpper(term)
	for _, common := range commonTerms {
		if termUpper == common {
			return true
		}
	}
	return false
}

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

// SaveThread saves a scraped thread to the database
func (s *ScraperService) SaveThread(thread *models.ScrapedThread) error {
	// Marshal metadata
	if err := thread.MarshalMetadata(); err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Check if thread already exists
	var existingID int64
	err := s.db.QueryRow(`
		SELECT id FROM scraped_threads
		WHERE external_id = ? AND source = ?
	`, thread.ExternalID, thread.Source).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new thread
		result, err := s.db.Exec(`
			INSERT INTO scraped_threads (
				external_id, source, title, url, author, category,
				view_count, reply_count, post_count, download_count,
				metadata, first_scraped_at, last_scraped_at,
				last_updated_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, thread.ExternalID, thread.Source, thread.Title, thread.URL,
			thread.Author, thread.Category, thread.ViewCount, thread.ReplyCount,
			thread.PostCount, thread.DownloadCount, thread.Metadata,
			thread.FirstScrapedAt, thread.LastScrapedAt, thread.LastUpdatedAt,
			time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert thread: %w", err)
		}

		threadID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get thread ID: %w", err)
		}
		thread.ID = threadID
	} else if err != nil {
		return fmt.Errorf("failed to check existing thread: %w", err)
	} else {
		// Update existing thread
		_, err = s.db.Exec(`
			UPDATE scraped_threads SET
				title = ?, url = ?, author = ?, category = ?,
				view_count = ?, reply_count = ?, post_count = ?,
				download_count = ?, metadata = ?, last_scraped_at = ?,
				last_updated_at = ?
			WHERE id = ?
		`, thread.Title, thread.URL, thread.Author, thread.Category,
			thread.ViewCount, thread.ReplyCount, thread.PostCount,
			thread.DownloadCount, thread.Metadata, time.Now(),
			thread.LastUpdatedAt, existingID)

		if err != nil {
			return fmt.Errorf("failed to update thread: %w", err)
		}

		thread.ID = existingID
	}

	return nil
}

// SavePost saves a scraped post to the database
func (s *ScraperService) SavePost(post *models.ScrapedPost) error {
	// Marshal metadata
	if err := post.MarshalMetadata(); err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Check if post already exists
	var existingID int64
	err := s.db.QueryRow(`
		SELECT id FROM scraped_posts
		WHERE external_id = ? AND source = ?
	`, post.ExternalID, post.Source).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new post
		result, err := s.db.Exec(`
			INSERT INTO scraped_posts (
				thread_id, external_id, source, author, content,
				plain_text, post_number, like_count, metadata,
				posted_at, scraped_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, post.ThreadID, post.ExternalID, post.Source, post.Author,
			post.Content, post.PlainText, post.PostNumber, post.LikeCount,
			post.Metadata, post.PostedAt, post.ScrapedAt, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert post: %w", err)
		}

		postID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get post ID: %w", err)
		}
		post.ID = postID
	} else if err != nil {
		return fmt.Errorf("failed to check existing post: %w", err)
	} else {
		// Update existing post
		_, err = s.db.Exec(`
			UPDATE scraped_posts SET
				content = ?, plain_text = ?, like_count = ?,
				metadata = ?, scraped_at = ?
			WHERE id = ?
		`, post.Content, post.PlainText, post.LikeCount,
			post.Metadata, time.Now(), existingID)

		if err != nil {
			return fmt.Errorf("failed to update post: %w", err)
		}

		post.ID = existingID
	}

	return nil
}

// SaveDownloadLink saves a download link to the database
func (s *ScraperService) SaveDownloadLink(link *models.ScrapedDownloadLink) error {
	// Marshal metadata
	if err := link.MarshalMetadata(); err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Check if link already exists
	var existingID int64
	err := s.db.QueryRow(`
		SELECT id FROM scraped_download_links
		WHERE url = ? AND source = ?
	`, link.URL, link.Source).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new link
		result, err := s.db.Exec(`
			INSERT INTO scraped_download_links (
				thread_id, post_id, source, provider, url, original_url,
				filename, file_size, file_type, status, metadata,
				discovered_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, link.ThreadID, link.PostID, link.Source, link.Provider,
			link.URL, link.OriginalURL, link.Filename, link.FileSize,
			link.FileType, link.Status, link.Metadata, link.DiscoveredAt,
			time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert download link: %w", err)
		}

		linkID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("failed to get download link ID: %w", err)
		}
		link.ID = linkID
	} else if err != nil {
		return fmt.Errorf("failed to check existing link: %w", err)
	} else {
		// Update existing link
		link.ID = existingID
		// Don't update existing links, just track they were found again
	}

	return nil
}

// ScrapeThreadComplete scrapes a thread and all its posts in one operation
func (s *ScraperService) ScrapeThreadComplete(threadURL string, parentActivityID ...int) error {
	// Track scrape start time for duration calculation
	scrapeStartTime := time.Now()

	// Use parent activity if provided, otherwise create new activity
	var activityID int
	var shouldComplete bool = true

	if len(parentActivityID) > 0 && parentActivityID[0] > 0 {
		// Use parent activity for logging
		activityID = parentActivityID[0]
		shouldComplete = false // Don't complete parent activity
		s.activityService.UpdateProgressLog(int64(activityID), 0, fmt.Sprintf("Scraping thread: %s", threadURL))
	} else {
		// Create standalone activity log
		activity, err := s.activityService.Create(&models.ActivityLogCreate{
			TaskType: "scraper_thread",
			Status:   "running",
			Message:  fmt.Sprintf("Scraping thread: %s", threadURL),
			Details: map[string]interface{}{
				"url": threadURL,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create activity: %w", err)
		}
		activityID = activity.ID
	}

	// Update progress
	s.activityService.UpdateProgress(activityID, 10, "Fetching thread information...")

	// Scrape thread metadata
	thread, err := s.ScrapeThread(threadURL)
	if err != nil {
		s.activityService.FailTask(activityID, fmt.Sprintf("Failed to scrape thread: %v", err))
		return err
	}

	// Check if thread needs re-scraping (smart caching)
	shouldRescrape, existingThread, err := s.ShouldRescrapeThread(threadURL, thread.ReplyCount)
	if err != nil {
		log.Printf("Warning: Failed to check cache status: %v. Proceeding with scrape.", err)
	} else if !shouldRescrape && existingThread != nil {
		// Thread exists and has no new posts - skip scraping
		s.activityService.UpdateProgressLog(int64(activityID), 0, fmt.Sprintf("✓ Thread already up-to-date (%d posts, last scraped %v ago)",
			existingThread.PostCount, time.Since(existingThread.LastScrapedAt).Round(time.Minute)))
		if shouldComplete {
			s.activityService.CompleteTask(int64(activityID),
				fmt.Sprintf("Thread already up-to-date. %d posts, %d download links (no changes since last scrape)",
					existingThread.PostCount, existingThread.DownloadCount))
		}
		return nil
	} else if existingThread != nil {
		// Thread exists but has new posts - log incremental update
		newPostCount := thread.ReplyCount - existingThread.ReplyCount
		s.activityService.UpdateProgressLog(int64(activityID), 0,
			fmt.Sprintf("📝 Incremental update: %d new posts detected (was %d, now %d)",
				newPostCount, existingThread.ReplyCount, thread.ReplyCount))
	}

	// Save thread
	if err := s.SaveThread(thread); err != nil {
		s.activityService.FailTask(activityID, fmt.Sprintf("Failed to save thread: %v", err))
		return err
	}

	s.activityService.UpdateProgress(activityID, 30, "Thread saved. Scraping posts...")

	// Scrape posts and download links (pass activity ID for detailed logging)
	posts, downloadLinks, err := s.ScrapePosts(threadURL, thread.ID, activityID)
	if err != nil {
		// Check if task was paused
		if strings.Contains(err.Error(), "paused by user") {
			s.activityService.UpdateProgressLog(int64(activityID), 0, "⏸️ Task paused during post scraping")
			return nil // Exit gracefully without marking as failed
		}
		s.activityService.FailTask(activityID, fmt.Sprintf("Failed to scrape posts: %v", err))
		return err
	}

	s.activityService.UpdateProgress(activityID, 50, fmt.Sprintf("Found %d posts. Saving...", len(posts)))

	// Save posts and their associated links
	for i, post := range posts {
		if err := s.SavePost(post); err != nil {
			log.Printf("Failed to save post %d: %v", i, err)
			continue
		}

		// Save download links that belong to this post
		// Links were marked with post index (PostID field was used as temporary index)
		for _, link := range downloadLinks {
			// Check if this link belongs to this post (by matching the temporary index)
			if link.PostID == int64(i) {
				link.PostID = post.ID // Update with actual database post ID
				if err := s.SaveDownloadLink(link); err != nil {
					log.Printf("Failed to save download link for post %d: %v", i, err)
				}
			}
		}

		progress := 50 + (40 * (i + 1) / len(posts))
		s.activityService.UpdateProgress(activityID, progress, fmt.Sprintf("Saved %d/%d posts", i+1, len(posts)))
	}

	// Update thread counts
	thread.PostCount = len(posts)
	thread.DownloadCount = len(downloadLinks)
	s.SaveThread(thread)

	s.activityService.UpdateProgress(activityID, 90, "Notifying AI Companion...")

	// Notify AI Companion
	s.notifyAICompanion(thread, len(posts), len(downloadLinks))

	// Complete activity (only if standalone, not when using parent activity)
	if shouldComplete {
		s.activityService.CompleteTask(int64(activityID), fmt.Sprintf(
			"Successfully scraped thread: %d posts, %d download links found",
			len(posts), len(downloadLinks),
		))
	} else {
		s.activityService.UpdateProgressLog(int64(activityID), 0, fmt.Sprintf(
			"✓ Thread scraped: %d posts, %d download links",
			len(posts), len(downloadLinks),
		))
	}

	// Send enhanced notification with comprehensive stats
	if s.notificationService != nil {
		scrapeDuration := time.Since(scrapeStartTime)
		isIncremental := existingThread != nil
		err := s.notificationService.NotifyScrapeCompletedEnhanced(
			thread.ID,
			thread.Title,
			len(posts),
			len(downloadLinks),
			scrapeDuration,
			isIncremental,
		)
		if err != nil {
			log.Printf("Failed to send scrape notification: %v\n", err)
		}
	}

	return nil
}

// notifyAICompanion sends scraping results to AI Companion
func (s *ScraperService) notifyAICompanion(thread *models.ScrapedThread, postCount, linkCount int) {
	if s.aiCompanionService == nil {
		return
	}

	// Create memory entry
	memoryData := map[string]interface{}{
		"type":           "scraper_result",
		"source":         "simpcity",
		"thread_title":   thread.Title,
		"thread_url":     thread.URL,
		"author":         thread.Author,
		"category":       thread.Category,
		"post_count":     postCount,
		"download_count": linkCount,
		"performers":     thread.MetadataObj.PerformerNames,
		"studios":        thread.MetadataObj.StudioNames,
		"tags":           thread.MetadataObj.Tags,
		"timestamp":      time.Now(),
	}

	memoryJSON, _ := json.Marshal(memoryData)

	// Store in AI Companion memory
	s.db.Exec(`
		INSERT INTO ai_companion_memory (type, content, metadata, created_at)
		VALUES (?, ?, ?, ?)
	`, "scraper_result", thread.Title, string(memoryJSON), time.Now())

	log.Printf("AI Companion notified of scraping results: %s", thread.Title)
}

// GetStats returns scraper statistics
func (s *ScraperService) GetStats() (*models.ScraperStats, error) {
	stats := &models.ScraperStats{
		SourceBreakdown:   make(map[string]int),
		ProviderBreakdown: make(map[string]int),
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

	return stats, nil
}

// GetAllThreads retrieves all scraped threads with pagination, sorting, and filtering
func (s *ScraperService) GetAllThreads(limit, offset int, sortBy, provider, filter string) ([]*models.ScrapedThread, int, error) {
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
	}

	// Get total count with filters
	var total int
	countQuery := "SELECT COUNT(*) FROM scraped_threads " + whereClause
	countArgs := args
	s.db.QueryRow(countQuery, countArgs...).Scan(&total)

	// Build ORDER BY clause (cast numeric fields to ensure proper sorting with NULLS LAST)
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
	case "downloads_desc":
		orderClause = "ORDER BY COALESCE(download_count, 0) DESC"
	}

	// Build final query
	query := "SELECT id, external_id, source, title, url, author, category, " +
		"view_count, reply_count, post_count, download_count, " +
		"metadata, first_scraped_at, last_scraped_at, last_updated_at, created_at " +
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
			&thread.LastUpdatedAt, &thread.CreatedAt,
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
			   metadata, first_scraped_at, last_scraped_at, last_updated_at, created_at
		FROM scraped_threads
		WHERE id = ?
	`, id).Scan(
		&thread.ID, &thread.ExternalID, &thread.Source, &thread.Title,
		&thread.URL, &thread.Author, &thread.Category, &thread.ViewCount,
		&thread.ReplyCount, &thread.PostCount, &thread.DownloadCount,
		&thread.Metadata, &thread.FirstScrapedAt, &thread.LastScrapedAt,
		&thread.LastUpdatedAt, &thread.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	thread.UnmarshalMetadata()
	return thread, nil
}

// SearchThreads searches threads by title or content
func (s *ScraperService) SearchThreads(query string, limit, offset int) ([]*models.ScrapedThread, int, error) {
	searchPattern := "%" + query + "%"

	var total int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM scraped_threads
		WHERE title LIKE ? OR author LIKE ?
	`, searchPattern, searchPattern).Scan(&total)

	rows, err := s.db.Query(`
		SELECT id, external_id, source, title, url, author, category,
			   view_count, reply_count, post_count, download_count,
			   metadata, first_scraped_at, last_scraped_at, last_updated_at, created_at
		FROM scraped_threads
		WHERE title LIKE ? OR author LIKE ?
		ORDER BY last_scraped_at DESC
		LIMIT ? OFFSET ?
	`, searchPattern, searchPattern, limit, offset)
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
			&thread.LastUpdatedAt, &thread.CreatedAt,
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
		SELECT id, thread_id, external_id, author, content, posted_at,
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
			&post.Content, &post.PostedAt, &post.Metadata, &post.CreatedAt,
		)
		if err != nil {
			continue
		}
		post.UnmarshalMetadata()
		posts = append(posts, post)
	}

	return posts, nil
}

// GetDownloadLinksByThreadID retrieves all download links for a thread
func (s *ScraperService) GetDownloadLinksByThreadID(threadID int64) ([]*models.ScrapedDownloadLink, error) {
	rows, err := s.db.Query(`
		SELECT id, thread_id, post_id, source, provider, url, original_url,
			   status, metadata, discovered_at, last_checked_at, created_at
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
			&link.Provider, &link.URL, &link.OriginalURL, &link.Status,
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

// DeleteThread deletes a thread and all associated posts and download links
func (s *ScraperService) DeleteThread(threadID int64) error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete download links first
	_, err = tx.Exec("DELETE FROM scraped_download_links WHERE thread_id = ?", threadID)
	if err != nil {
		return fmt.Errorf("failed to delete download links: %w", err)
	}

	// Delete posts
	_, err = tx.Exec("DELETE FROM scraped_posts WHERE thread_id = ?", threadID)
	if err != nil {
		return fmt.Errorf("failed to delete posts: %w", err)
	}

	// Delete thread
	result, err := tx.Exec("DELETE FROM scraped_threads WHERE id = ?", threadID)
	if err != nil {
		return fmt.Errorf("failed to delete thread: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("thread not found")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Deleted thread %d and all associated data", threadID)
	return nil
}

// ForumThreadInfo represents basic thread information from forum listing
type ForumThreadInfo struct {
	Title      string
	URL        string
	ExternalID string
	Author     string
	ReplyCount int
	ViewCount  int
}

// ScrapeForumCategory scrapes all threads from a forum category (with pagination)
func (s *ScraperService) ScrapeForumCategory(forumURL string, activityID ...int) ([]ForumThreadInfo, error) {
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
			if (i+1) % 10 == 0 {
				s.activityService.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
					"thread_index":     i + 1,
					"threads_completed": i + 1,
					"total_threads":    totalThreads,
					"success_count":    successCount,
					"error_count":      errorCount,
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

// Known tags to extract from titles
var knownTags = []string{
	"XXX", "OnlyFans", "BBW", "T H I C C", "MILF", "Petite",
	"Teen", "Asian", "Indian", "Ebony", "Latina", "Feet", "Retired",
}

// ExtractTagsFromTitle extracts known tags from thread title
func (s *ScraperService) ExtractTagsFromTitle(title string) []string {
	var extractedTags []string
	titleUpper := strings.ToUpper(title)

	for _, tag := range knownTags {
		// Case-insensitive search
		tagUpper := strings.ToUpper(tag)
		if strings.Contains(titleUpper, tagUpper) {
			// Add tag if not already in list
			if !contains(extractedTags, tag) {
				extractedTags = append(extractedTags, tag)
			}
		}
	}

	return extractedTags
}

// CleanTitleFromTags removes known tags from title and cleans it up
func (s *ScraperService) CleanTitleFromTags(title string) string {
	cleanedTitle := title

	// Remove tags (case-insensitive)
	for _, tag := range knownTags {
		// Try various bracket formats: [TAG], (TAG), {TAG}
		patterns := []string{
			fmt.Sprintf("[%s]", tag),
			fmt.Sprintf("(%s)", tag),
			fmt.Sprintf("{%s}", tag),
			fmt.Sprintf("【%s】", tag), // Japanese brackets
		}

		for _, pattern := range patterns {
			// Case-insensitive replacement
			re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(pattern))
			cleanedTitle = re.ReplaceAllString(cleanedTitle, "")
		}

		// Also remove standalone tags with spaces
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(tag) + `\b`)
		cleanedTitle = re.ReplaceAllString(cleanedTitle, "")
	}

	// Clean up multiple spaces, dashes, and pipes
	cleanedTitle = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedTitle, " ")
	cleanedTitle = regexp.MustCompile(`\s*[-|]\s*$`).ReplaceAllString(cleanedTitle, "")
	cleanedTitle = regexp.MustCompile(`^\s*[-|]\s*`).ReplaceAllString(cleanedTitle, "")
	cleanedTitle = strings.TrimSpace(cleanedTitle)

	return cleanedTitle
}

// ExtractPerformerNameFromTitle extracts potential performer name from thread title
func (s *ScraperService) ExtractPerformerNameFromTitle(title string) []string {
	// First, clean the title from tags
	cleanedTitle := s.CleanTitleFromTags(title)

	// Common patterns for performer threads:
	// "Performer Name - Title"
	// "[Performer Name]"
	// "Performer Name | something"
	// "Performer Name (aka Other Name)"

	var names []string

	// Remove common prefixes
	cleanedTitle = strings.TrimSpace(cleanedTitle)

	// Pattern 1: Text before dash
	if strings.Contains(cleanedTitle, " - ") {
		parts := strings.Split(cleanedTitle, " - ")
		name := strings.TrimSpace(parts[0])
		// Remove brackets if present
		name = strings.Trim(name, "[]")
		if name != "" {
			names = append(names, name)
		}
	}

	// Pattern 2: Text in brackets
	bracketRegex := regexp.MustCompile(`\[(.*?)\]`)
	matches := bracketRegex.FindAllStringSubmatch(cleanedTitle, -1)
	for _, match := range matches {
		if len(match) > 1 {
			name := strings.TrimSpace(match[1])
			if name != "" && !contains(names, name) {
				names = append(names, name)
			}
		}
	}

	// Pattern 3: Text before pipe
	if strings.Contains(cleanedTitle, " | ") {
		parts := strings.Split(cleanedTitle, " | ")
		name := strings.TrimSpace(parts[0])
		name = strings.Trim(name, "[]")
		if name != "" && !contains(names, name) {
			names = append(names, name)
		}
	}

	// Pattern 4: Extract aka names
	akaRegex := regexp.MustCompile(`\(aka\s+(.*?)\)`)
	matches = akaRegex.FindAllStringSubmatch(cleanedTitle, -1)
	for _, match := range matches {
		if len(match) > 1 {
			name := strings.TrimSpace(match[1])
			if name != "" && !contains(names, name) {
				names = append(names, name)
			}
		}
	}

	return names
}

// contains checks if a string slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

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

// FindOrCreatePerformer finds an existing performer by name or creates a new one
func (s *ScraperService) FindOrCreatePerformer(name string) (int64, error) {
	// First, try to find existing performer (case-insensitive)
	var performerID int64
	err := s.db.QueryRow(`
		SELECT id FROM performers WHERE LOWER(name) = LOWER(?)
	`, name).Scan(&performerID)

	if err == nil {
		// Found existing performer
		return performerID, nil
	}

	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query performer: %w", err)
	}

	// Create new performer
	result, err := s.db.Exec(`
		INSERT INTO performers (name, category, metadata, created_at, updated_at)
		VALUES (?, 'regular', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, name)

	if err != nil {
		return 0, fmt.Errorf("failed to create performer: %w", err)
	}

	performerID, err = result.LastInsertId()
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
	threads, _, err := s.GetAllThreads(10000, 0, "", "", "")
	if err != nil {
		return fmt.Errorf("failed to get threads: %w", err)
	}

	linkedCount := 0
	for _, thread := range threads {
		// Extract performer names from title
		performerNames := s.ExtractPerformerNameFromTitle(thread.Title)

		for _, name := range performerNames {
			// Find or create performer
			performerID, err := s.FindOrCreatePerformer(name)
			if err != nil {
				log.Printf("Error finding/creating performer %s: %v", name, err)
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

// DeleteThreads deletes multiple threads by their IDs
func (s *ScraperService) DeleteThreads(threadIDs []int64) error {
	if len(threadIDs) == 0 {
		return fmt.Errorf("no thread IDs provided")
	}

	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete download links for these threads
	placeholders := make([]string, len(threadIDs))
	args := make([]interface{}, len(threadIDs))
	for i, id := range threadIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf("DELETE FROM scraped_download_links WHERE thread_id IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete download links: %w", err)
	}

	// Delete posts for these threads
	query = fmt.Sprintf("DELETE FROM scraped_posts WHERE thread_id IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete posts: %w", err)
	}

	// Delete performer-thread links
	query = fmt.Sprintf("DELETE FROM performer_scraped_threads WHERE thread_id IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete performer links: %w", err)
	}

	// Delete the threads themselves
	query = fmt.Sprintf("DELETE FROM scraped_threads WHERE id IN (%s)", strings.Join(placeholders, ","))
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete threads: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully deleted %d threads", len(threadIDs))
	return nil
}

// DeleteAllThreads deletes all scraped threads from the database
func (s *ScraperService) DeleteAllThreads() error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all download links
	_, err = tx.Exec("DELETE FROM scraped_download_links")
	if err != nil {
		return fmt.Errorf("failed to delete download links: %w", err)
	}

	// Delete all posts
	_, err = tx.Exec("DELETE FROM scraped_posts")
	if err != nil {
		return fmt.Errorf("failed to delete posts: %w", err)
	}

	// Delete all performer-thread links
	_, err = tx.Exec("DELETE FROM performer_scraped_threads")
	if err != nil {
		return fmt.Errorf("failed to delete performer links: %w", err)
	}

	// Delete all threads
	_, err = tx.Exec("DELETE FROM scraped_threads")
	if err != nil {
		return fmt.Errorf("failed to delete threads: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Println("Successfully deleted all scraped threads")
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

// getExistingPostIDs retrieves all existing post external IDs for a thread
func (s *ScraperService) getExistingPostIDs(threadID int64) (map[string]bool, error) {
	existingPosts := make(map[string]bool)

	rows, err := s.db.Query(`
		SELECT external_id
		FROM scraped_posts
		WHERE thread_id = ? AND source = 'simpcity'
	`, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing posts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var externalID string
		if err := rows.Scan(&externalID); err != nil {
			log.Printf("⚠️ Failed to scan post ID: %v", err)
			continue
		}
		existingPosts[externalID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating existing posts: %w", err)
	}

	return existingPosts, nil
}

// isValidPost validates post content quality before saving
func (s *ScraperService) isValidPost(post *models.ScrapedPost) bool {
	// Check if post has an external ID (required for uniqueness)
	if post.ExternalID == "" {
		log.Printf("⚠️ Skipping post without external ID")
		return false
	}

	// Check if post has any content (plain text or HTML)
	hasContent := false
	if post.PlainText != "" && len(strings.TrimSpace(post.PlainText)) > 0 {
		hasContent = true
	}
	if post.Content != "" && len(strings.TrimSpace(post.Content)) > 10 { // At least 10 chars of HTML
		hasContent = true
	}

	// Check if post has attachments (images count as content)
	if post.MetadataObj != nil && len(post.MetadataObj.Attachments) > 0 {
		hasContent = true
	}

	if !hasContent {
		log.Printf("⚠️ Skipping empty post %s (no content or attachments)", post.ExternalID)
		return false
	}

	// Minimum content length check (at least 5 characters of actual text)
	plainText := strings.TrimSpace(post.PlainText)
	if len(plainText) > 0 && len(plainText) < 5 && len(post.MetadataObj.Attachments) == 0 {
		log.Printf("⚠️ Skipping post %s with minimal content (%d chars)", post.ExternalID, len(plainText))
		return false
	}

	return true
}

// deduplicateLinks removes duplicate download links within a post
func (s *ScraperService) deduplicateLinks(links []*models.ScrapedDownloadLink) []*models.ScrapedDownloadLink {
	if len(links) == 0 {
		return links
	}

	seen := make(map[string]bool)
	deduplicated := make([]*models.ScrapedDownloadLink, 0, len(links))
	duplicateCount := 0

	for _, link := range links {
		// Normalize URL for comparison (remove trailing slashes, convert to lowercase)
		normalizedURL := strings.ToLower(strings.TrimSuffix(link.URL, "/"))

		if !seen[normalizedURL] {
			seen[normalizedURL] = true
			deduplicated = append(deduplicated, link)
		} else {
			duplicateCount++
		}
	}

	if duplicateCount > 0 {
		log.Printf("✂️ Deduplicated %d duplicate links (kept %d unique)", duplicateCount, len(deduplicated))
	}

	return deduplicated
}
