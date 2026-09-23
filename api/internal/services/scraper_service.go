package services

import (
	"database/sql"
	"encoding/base64"
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

// domainMigrations maps old domains to their current replacements.
// When a scraper encounters a URL with an old domain, it automatically redirects.
var domainMigrations = map[string]string{
	"simpcity.is":  "simpcity.cr",
	"simpcity.su":  "simpcity.cr",
}

// migrateURL applies domain migrations to a URL, returning the updated URL.
// If the domain was changed, it also persists the fix to the database.
func migrateURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if newDomain, ok := domainMigrations[parsed.Host]; ok {
		oldHost := parsed.Host
		parsed.Host = newDomain
		fixed := parsed.String()
		log.Printf("🔄 Auto-migrating URL domain: %s → %s", oldHost, newDomain)
		// Persist fix to database in background
		go persistDomainFix(oldHost, newDomain)
		return fixed
	}
	return rawURL
}

// persistDomainFix updates all URLs in the database when a domain migration is detected.
func persistDomainFix(oldDomain, newDomain string) {
	db := database.DB
	if db == nil {
		return
	}
	tables := []struct {
		name   string
		urlCol string
	}{
		{"scraped_threads", "url"},
		{"scraped_posts", "source_url"},
		{"scraped_download_links", "url"},
	}
	for _, t := range tables {
		query := fmt.Sprintf("UPDATE %s SET %s = REPLACE(%s, ?, ?) WHERE %s LIKE ?",
			t.name, t.urlCol, t.urlCol, t.urlCol)
		db.Exec(query, oldDomain, newDomain, "%"+oldDomain+"%")
	}
	// Also fix auto-update configs
	db.Exec(`UPDATE scraper_auto_update_configs SET config_json = REPLACE(config_json, ?, ?) WHERE config_json LIKE ?`,
		oldDomain, newDomain, "%"+oldDomain+"%")
	log.Printf("✅ Persisted domain fix: %s → %s across all tables", oldDomain, newDomain)
}

// ScraperService handles web scraping operations
type ScraperService struct {
	db                  *sql.DB
	activityService     *ActivityService
	aiCompanionService  *AICompanionService
	notificationService *NotificationService
	httpClient          *http.Client
	sessionCookie       string // Store session cookie for authenticated requests
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

// ScrapeThread scrapes a single thread from simpcity.cr
func (s *ScraperService) ScrapeThread(threadURL string) (*models.ScrapedThread, error) {
	// Clean the URL: remove common suffixes like /unread, /latest
	cleanURL := strings.TrimSuffix(threadURL, "/")
	cleanURL = strings.TrimSuffix(cleanURL, "/unread")
	cleanURL = strings.TrimSuffix(cleanURL, "/latest")
	cleanURL = strings.TrimSuffix(cleanURL, "/")

	// Auto-migrate broken domains (e.g. simpcity.is → simpcity.cr)
	cleanURL = migrateURL(cleanURL)

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
		hasCookie := s.sessionCookie != ""
		if hasCookie {
			req.Header.Set("Cookie", s.sessionCookie)
		}

		// Add user agent to mimic browser
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		req.Header.Set("Accept-Encoding", "identity") // Prevent compressed responses Go can't decode (e.g. Brotli)

		// Fetch the page
		log.Printf("Making HTTP request to %s (attempt %d/%d, has cookie: %v)", cleanURL, attempt+1, maxRetries+1, hasCookie)
		requestStart := time.Now()
		resp, err = s.httpClient.Do(req)
		requestDuration := time.Since(requestStart)

		if err != nil {
			log.Printf("HTTP request failed after %v: %v", requestDuration, err)
			return nil, fmt.Errorf("failed to fetch thread: %w", err)
		}
		log.Printf("HTTP request completed in %v, status: %d", requestDuration, resp.StatusCode)

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
		finalURL := resp.Request.URL.String()
		log.Printf("HTTP %d from %s (requested: %s)", resp.StatusCode, finalURL, cleanURL)

		doc, err = goquery.NewDocumentFromReader(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to parse HTML: %w", err)
		}

		// Check for login redirect (URL contains /login)
		if strings.Contains(finalURL, "/login") {
			return nil, fmt.Errorf("redirected to login page - session cookie expired or invalid")
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

	// Validate that we got a valid page (not a login page or error)
	if rawTitle == "" {
		// Check if we got redirected to login
		if doc.Find("form[action*='login']").Length() > 0 || doc.Find(".formRow--input input[name='login']").Length() > 0 {
			return nil, fmt.Errorf("authentication required - session cookie expired or invalid. Please update your session cookie in Settings")
		}
		// Check for other error indicators
		if doc.Find(".blockMessage--error").Length() > 0 {
			errorMsg := strings.TrimSpace(doc.Find(".blockMessage--error").First().Text())
			return nil, fmt.Errorf("page error: %s", errorMsg)
		}
		return nil, fmt.Errorf("failed to extract thread title - page may be inaccessible or structure changed")
	}

	log.Printf("Extracted title (len=%d): %q", len(rawTitle), rawTitle)

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

	// Auto-migrate broken domains (e.g. simpcity.is → simpcity.cr)
	cleanURL = migrateURL(cleanURL)

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

	// First, decode any /redirect/?to=BASE64&m=b64 links in the HTML content
	// Simpcity wraps external links in redirect URLs with base64-encoded targets
	expandedContent := s.decodeRedirectLinks(content)

	// Regular expressions for different hosting providers - more flexible patterns
	providers := map[string]*regexp.Regexp{
		"gofile":     regexp.MustCompile(`https?://(?:www\.)?gofile\.io/d/[a-zA-Z0-9_-]+`),
		"pixeldrain": regexp.MustCompile(`https?://(?:www\.)?pixeldrain\.com/(?:u|l)/[a-zA-Z0-9_-]+`),
		"bunkr":      regexp.MustCompile(`https?://(?:www\.)?(?:bunkr|bunkrr)\.[a-z]+/[a-z]/[a-zA-Z0-9_-]+`),
		"cyberdrop":  regexp.MustCompile(`https?://(?:www\.)?cyberdrop\.(?:me|to|cc)/a/[a-zA-Z0-9_-]+`),
		"mediafire":  regexp.MustCompile(`https?://(?:www\.)?mediafire\.com/(?:file|folder)/[a-zA-Z0-9_/-]+`),
		"mega":       regexp.MustCompile(`https?://(?:www\.)?mega\.nz/(?:file|folder)/[a-zA-Z0-9#_-]+`),
		"cyberfile":  regexp.MustCompile(`https?://(?:www\.)?cyberfile\.(?:me|su)/[a-zA-Z0-9_.?=&/-]+`),
	}

	for provider, regex := range providers {
		matches := regex.FindAllString(expandedContent, -1)
		for _, match := range matches {
			link := &models.ScrapedDownloadLink{
				ThreadID:     threadID,
				PostID:       postID,
				Source:       "simpcity",
				Provider:     provider,
				URL:          match,
				OriginalURL:  match,
				Status:       "active",
				DiscoveredAt: time.Now(),
				MetadataObj:  &models.DownloadLinkMetadata{},
			}
			links = append(links, link)
		}
	}

	return links
}

// decodeRedirectLinks finds /redirect/?to=BASE64&m=b64 patterns in HTML and appends the decoded URLs
func (s *ScraperService) decodeRedirectLinks(content string) string {
	// Match redirect hrefs: /redirect/?to=ENCODED&e=1&m=b64 (or similar query param orders)
	redirectRegex := regexp.MustCompile(`/redirect/\?[^"']*?to=([A-Za-z0-9+/=]+)[^"']*?m=b64`)
	matches := redirectRegex.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return content
	}

	// Append decoded URLs to the content so the provider regexes can find them
	var decoded []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		encodedURL := match[1]
		// Try standard base64 decoding
		urlBytes, err := base64.StdEncoding.DecodeString(encodedURL)
		if err != nil {
			// Try URL-safe base64
			urlBytes, err = base64.URLEncoding.DecodeString(encodedURL)
			if err != nil {
				// Try without padding
				urlBytes, err = base64.RawStdEncoding.DecodeString(encodedURL)
				if err != nil {
					continue
				}
			}
		}
		decodedURL := string(urlBytes)
		if strings.HasPrefix(decodedURL, "http") {
			decoded = append(decoded, decodedURL)
		}
	}

	if len(decoded) > 0 {
		// Append decoded URLs as plain text so provider regexes can match them
		content += "\n" + strings.Join(decoded, "\n")
	}

	return content
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
	log.Printf("Thread scraped: %q, reply_count=%d. Checking if rescrape needed...", thread.Title, thread.ReplyCount)
	shouldRescrape, existingThread, err := s.ShouldRescrapeThread(threadURL, thread.ReplyCount)
	if err != nil {
		log.Printf("Warning: Failed to check cache status: %v. Proceeding with scrape.", err)
	} else if !shouldRescrape && existingThread != nil {
		// Thread exists and has no new posts - skip scraping
		log.Printf("Thread already up-to-date: %d posts, %d downloads (last scraped %v ago). Skipping.",
			existingThread.PostCount, existingThread.DownloadCount, time.Since(existingThread.LastScrapedAt).Round(time.Minute))
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
