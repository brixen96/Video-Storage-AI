package api

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

// extractEmbeddedNumber finds the single digit sequence inside a path segment (e.g. "page2.html" → 2, "page{page}.html").
// Returns (number, template, true) if exactly one number is found; otherwise (0, "", false).
var reDigits = regexp.MustCompile(`\d+`)

func extractEmbeddedNumber(seg string) (number int, template string, ok bool) {
	matches := reDigits.FindAllStringIndex(seg, -1)
	if len(matches) != 1 {
		return 0, "", false
	}
	start, end := matches[0][0], matches[0][1]
	n, err := strconv.Atoi(seg[start:end])
	if err != nil {
		return 0, "", false
	}
	return n, seg[:start] + "{page}" + seg[end:], true
}

var scraperService *services.ScraperService

// ensureScraperService initializes the service if needed
func ensureScraperService() *services.ScraperService {
	if scraperService == nil {
		activitySvc := ensureActivityService()
		aiCompanionSvc := GetAICompanionService()
		scraperService = services.NewScraperService(activitySvc, aiCompanionSvc)
	}
	return scraperService
}

// scrapeThread handles scraping a single thread
func scrapeThread(c *gin.Context) {
	svc := ensureScraperService()
	consoleLogSvc := ensureConsoleLogService()

	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Log scrape start
	consoleLogSvc.LogAPI("info", "Thread scraping started", map[string]interface{}{
		"url": request.URL,
	})

	// Start scraping in the background (tracked by activity service)
	go func() {
		if err := svc.ScrapeThreadComplete(request.URL); err != nil {
			log.Printf("Error scraping thread: %v", err)
			consoleLogSvc.LogAPI("error", "Thread scraping failed", map[string]interface{}{
				"url":   request.URL,
				"error": err.Error(),
			})
		} else {
			consoleLogSvc.LogAPI("info", "Thread scraping completed successfully", map[string]interface{}{
				"url": request.URL,
			})
		}
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Thread scraping started. Check activity logs for progress."))
}

// getScraperStats returns scraper statistics
func getScraperStats(c *gin.Context) {
	svc := ensureScraperService()

	stats, err := svc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve scraper stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Scraper stats retrieved successfully"))
}

// getScrapedThreads retrieves all scraped threads with pagination, sorting, and filtering
func getScrapedThreads(c *gin.Context) {
	svc := ensureScraperService()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	sortBy := c.DefaultQuery("sort", "date_desc")
	provider := c.Query("provider")
	filter := c.Query("filter")
	author := c.Query("author")
	category := c.Query("category")

	if limit < 1 || limit > 200 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	threads, total, err := svc.GetAllThreads(limit, offset, sortBy, provider, filter, author, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve threads",
			err.Error(),
		))
		return
	}

	totalPages := (total + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Threads retrieved successfully",
		"data":    threads,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// getScrapedThread retrieves a specific thread with posts and download links
func getScrapedThread(c *gin.Context) {
	svc := ensureScraperService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	thread, err := svc.GetThreadByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Thread not found",
			err.Error(),
		))
		return
	}

	// Get posts for this thread
	posts, err := svc.GetPostsByThreadID(id)
	if err != nil {
		posts = []*models.ScrapedPost{} // Empty array if error
	}

	// Get download links for this thread
	downloadLinks, err := svc.GetDownloadLinksByThreadID(id)
	if err != nil {
		downloadLinks = []*models.ScrapedDownloadLink{} // Empty array if error
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"thread":         thread,
		"posts":          posts,
		"download_links": downloadLinks,
	}, "Thread retrieved successfully"))
}

// searchScrapedThreads searches threads
func searchScrapedThreads(c *gin.Context) {
	svc := ensureScraperService()

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Search query required",
			"Please provide a search query using the 'q' parameter",
		))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	sortBy := c.DefaultQuery("sort", "date_desc")
	provider := c.Query("provider")
	filter := c.Query("filter")
	author := c.Query("author")
	category := c.Query("category")

	if limit < 1 || limit > 200 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	threads, total, err := svc.SearchThreads(query, limit, offset, sortBy, provider, filter, author, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to search threads",
			err.Error(),
		))
		return
	}

	totalPages := (total + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Search completed successfully",
		"data":    threads,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// rescrapeThread rescrapers an existing thread for updates
func rescrapeThread(c *gin.Context) {
	svc := ensureScraperService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	// Get thread to find its URL
	thread, err := svc.GetThreadByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Thread not found",
			err.Error(),
		))
		return
	}

	// Start rescraping in the background (force=true to bypass smart caching)
	go func() {
		if err := svc.ScrapeThreadCompleteForce(thread.URL, true); err != nil {
			log.Printf("Error rescraping thread: %v", err)
		}
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Thread rescraping started. Check activity logs for progress."))
}

// toggleThreadWatch toggles the watch status for a thread
func toggleThreadWatch(c *gin.Context) {
	svc := ensureScraperService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	// Toggle watch status
	isWatched, err := svc.ToggleThreadWatch(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to toggle watch status",
			err.Error(),
		))
		return
	}

	message := "Thread is now being watched"
	if !isWatched {
		message = "Thread is no longer being watched"
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		map[string]interface{}{
			"is_watched": isWatched,
		},
		message,
	))
}

// deleteThread deletes a scraped thread and all its associated data
func deleteThread(c *gin.Context) {
	svc := ensureScraperService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	// Delete the thread (this will cascade to posts and links)
	if err := svc.DeleteThread(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete thread",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Thread deleted successfully"))
}

// setSessionCookie sets the authentication cookie for scraping
func setSessionCookie(c *gin.Context) {
	svc := ensureScraperService()

	var request struct {
		Cookie string `json:"cookie" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Set the cookie first
	svc.SetSessionCookie(request.Cookie)

	// Verify the cookie by trying to access a protected page
	verified, username, err := svc.VerifySessionCookie()

	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"verified": false,
			"error":    err.Error(),
			"message":  "Cookie saved but verification failed. The cookie may be invalid or expired.",
		}, "Cookie saved but verification failed"))
		return
	}

	if !verified {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"verified": false,
			"message":  "Cookie saved but appears to be invalid. Please check you copied all cookies correctly.",
		}, "Cookie saved but not verified"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"verified": true,
		"username": username,
		"message":  "Cookie verified successfully! You are logged in.",
	}, "Session cookie set and verified successfully"))
}

// getSessionCookie returns the current session cookie status
func getSessionCookie(c *gin.Context) {
	svc := ensureScraperService()

	cookie := svc.GetSessionCookie()
	isSet := cookie != ""

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"is_set": isSet,
		"length": len(cookie),
	}, "Session cookie status retrieved"))
}

// scrapeForumCategory scrapes all threads from a forum category listing
func scrapeForumCategory(c *gin.Context) {
	svc := ensureScraperService()

	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Start scraping in the background
	go func() {
		threads, err := svc.ScrapeForumCategory(request.URL)
		if err != nil {
			log.Printf("Error scraping forum category: %v", err)
		} else {
			log.Printf("Found %d threads in forum category", len(threads))
		}
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Forum category scraping started. Check logs for progress."))
}

// scrapeForumAndSaveAll scrapes all threads from a forum and saves complete content
func scrapeForumAndSaveAll(c *gin.Context) {
	svc := ensureScraperService()

	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Start full forum scraping in the background
	go func() {
		if err := svc.ScrapeForumAndSaveAll(request.URL); err != nil {
			log.Printf("Error scraping forum: %v", err)
		}
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Full forum scraping started. This may take a while. Check activity logs for progress."))
}

// autoLinkThreadsToPerformers automatically links scraped threads to performers
func autoLinkThreadsToPerformers(c *gin.Context) {
	svc := ensureScraperService()

	// Start auto-linking in the background
	go func() {
		if err := svc.AutoLinkThreadsToPerformers(); err != nil {
			log.Printf("Error auto-linking threads to performers: %v", err)
		}
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Auto-linking started. Check logs for progress."))
}

// getThreadsByPerformer retrieves all scraped threads linked to a specific performer
func getThreadsByPerformer(c *gin.Context) {
	svc := ensureScraperService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	threads, err := svc.GetThreadsByPerformer(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve threads for performer",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(threads, "Threads retrieved successfully"))
}

// linkThreadToPerformer manually links a thread to a performer
func linkThreadToPerformer(c *gin.Context) {
	svc := ensureScraperService()

	var request struct {
		ThreadID    int64   `json:"thread_id" binding:"required"`
		PerformerID int64   `json:"performer_id" binding:"required"`
		Confidence  float64 `json:"confidence"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Default confidence if not provided
	if request.Confidence == 0 {
		request.Confidence = 1.0
	}

	if err := svc.LinkThreadToPerformer(request.ThreadID, request.PerformerID, request.Confidence); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to link thread to performer",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Thread linked to performer successfully"))
}

// createPerformerFromThread creates a new performer from a thread's extracted name
func createPerformerFromThread(c *gin.Context) {
	svc := ensureScraperService()

	var request struct {
		ThreadID int64  `json:"thread_id" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Create the performer
	performerID, err := svc.CreatePerformerFromThread(request.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create performer",
			err.Error(),
		))
		return
	}

	// Automatically link the thread to the newly created performer
	err = svc.LinkThreadToPerformer(request.ThreadID, performerID, 1.0)
	if err != nil {
		log.Printf("Created performer %d but failed to link thread %d: %v", performerID, request.ThreadID, err)
	}

	// Get the performer details to return
	performerSvc := ensurePerformerService()
	performer, err := performerSvc.GetByID(performerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Performer created but failed to retrieve details",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(performer, "Performer created and linked to thread successfully"))
}

// checkLinkStatuses checks the status of all download links
func checkLinkStatuses(c *gin.Context) {
	svc := ensureScraperService()

	// Start link checking in the background
	go func() {
		if err := svc.CheckAllLinkStatuses(); err != nil {
			log.Printf("Error checking link statuses: %v", err)
		}
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Link status check started. This will take a while. Check logs for progress."))
}

// deleteMultipleThreads deletes multiple threads by their IDs
func deleteMultipleThreads(c *gin.Context) {
	svc := ensureScraperService()

	var request struct {
		ThreadIDs []int64 `json:"thread_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if len(request.ThreadIDs) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"No thread IDs provided",
			"Please provide at least one thread ID to delete",
		))
		return
	}

	if err := svc.DeleteThreads(request.ThreadIDs); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete threads",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, fmt.Sprintf("Successfully deleted %d threads", len(request.ThreadIDs))))
}

// deleteAllThreads deletes all scraped threads
func deleteAllThreads(c *gin.Context) {
	svc := ensureScraperService()

	if err := svc.DeleteAllThreads(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete all threads",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Successfully deleted all scraped threads"))
}

// getWatchedThreads returns all watched threads for the tab interface
func getWatchedThreads(c *gin.Context) {
	svc := ensureScraperService()

	threads, err := svc.GetWatchedThreads()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get watched threads", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(threads, fmt.Sprintf("Found %d watched threads", len(threads))))
}

// enhanceThread triggers AI-powered thread enhancement
func enhanceThread(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	var request struct {
		CleanContent      bool `json:"clean_content"`
		ExtractPerformers bool `json:"extract_performers"`
		ExtractTags       bool `json:"extract_tags"`
		GenerateSummary   bool `json:"generate_summary"`
	}

	// Bind with defaults
	if err := c.ShouldBindJSON(&request); err != nil {
		// Use defaults if no body provided
		request = struct {
			CleanContent      bool `json:"clean_content"`
			ExtractPerformers bool `json:"extract_performers"`
			ExtractTags       bool `json:"extract_tags"`
			GenerateSummary   bool `json:"generate_summary"`
		}{
			CleanContent:      true,
			ExtractPerformers: true,
			ExtractTags:       true,
			GenerateSummary:   true,
		}
	}

	// Set defaults
	if !request.CleanContent && !request.ExtractPerformers && !request.ExtractTags && !request.GenerateSummary {
		request = struct {
			CleanContent      bool `json:"clean_content"`
			ExtractPerformers bool `json:"extract_performers"`
			ExtractTags       bool `json:"extract_tags"`
			GenerateSummary   bool `json:"generate_summary"`
		}{
			CleanContent:      true,
			ExtractPerformers: true,
			ExtractTags:       true,
			GenerateSummary:   true,
		}
	}

	options := services.EnhancementOptions{
		CleanContent:      request.CleanContent,
		ExtractPerformers: request.ExtractPerformers,
		ExtractTags:       request.ExtractTags,
		GenerateSummary:   request.GenerateSummary,
	}

	enhancer := services.NewThreadEnhancer(ensureActivityService())

	// Run enhancement in background
	go func() {
		result, err := enhancer.EnhanceThread(id, options)
		if err != nil {
			log.Printf("Thread enhancement failed for thread %d: %v", id, err)
		} else {
			log.Printf("Thread enhancement completed: %s", result.Summary)
		}
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Thread enhancement started. Check activity logs for progress."))
}

// getThreadEnhancementStatus returns enhancement status for a thread
func getThreadEnhancementStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	// Query recent enhancement activities
	db := database.GetDB()
	rows, err := db.Query(`
		SELECT id, status, message, progress, started_at, completed_at
		FROM activity_logs
		WHERE task_type = 'thread_enhancement' 
		AND details LIKE ?
		ORDER BY started_at DESC
		LIMIT 10
	`, fmt.Sprintf(`%%"thread_id":%d%%`, id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get enhancement status",
			err.Error(),
		))
		return
	}
	defer rows.Close()

	jobs := []map[string]interface{}{}
	for rows.Next() {
		var jobID int64
		var status, message string
		var progress int
		var startedAt, completedAt time.Time
		if err := rows.Scan(&jobID, &status, &message, &progress, &startedAt, &completedAt); err == nil {
			jobs = append(jobs, map[string]interface{}{
				"id":           jobID,
				"status":       status,
				"message":      message,
				"progress":     progress,
				"started_at":   startedAt,
				"completed_at": completedAt,
			})
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"thread_id": id,
		"jobs":      jobs,
	}, "Enhancement status retrieved"))
}

// getThreadRecentPosts returns recent posts for a thread (for hover preview)
func getThreadRecentPosts(c *gin.Context) {
	svc := ensureScraperService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid thread ID", err.Error()))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	posts, err := svc.GetRecentPostsByThreadID(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get posts", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(posts, fmt.Sprintf("Found %d posts", len(posts))))
}

// markPostAsRead marks a post as read
func markPostAsRead(c *gin.Context) {
	svc := ensureScraperService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid post ID", err.Error()))
		return
	}

	if err := svc.MarkPostAsRead(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to mark as read", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Post marked as read"))
}

// paginationDetectResult holds the result of pagination analysis
type paginationDetectResult struct {
	PageUrlTemplate string `json:"page_url_template"`
	PageStartsAt    int    `json:"page_starts_at"`
	SiteType        string `json:"site_type"`
	Confidence      string `json:"confidence"` // "high" or "low"
	Notes           string `json:"notes"`
}

// detectPaginationConfig compares two page URLs to detect the pagination pattern and site type
func detectPaginationConfig(c *gin.Context) {
	var req struct {
		Page1URL string `json:"page1_url" binding:"required"`
		Page2URL string `json:"page2_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", err.Error()))
		return
	}
	result := analyzePaginationURLs(req.Page1URL, req.Page2URL)
	c.JSON(http.StatusOK, models.SuccessResponse(result, "Pagination config detected"))
}

// analyzePaginationURLs compares two page URLs and returns the detected pagination template + site type
func analyzePaginationURLs(rawURL1, rawURL2 string) paginationDetectResult {
	u1, err1 := url.Parse(rawURL1)
	u2, err2 := url.Parse(rawURL2)
	if err1 != nil || err2 != nil {
		return paginationDetectResult{SiteType: "generic", Confidence: "low", Notes: "Could not parse URLs"}
	}

	siteType := detectSiteTypeFromURL(u1.Hostname())

	// --- Check query parameter changes ---
	q1 := u1.Query()
	q2 := u2.Query()

	// Case A: same param exists in both with sequential numeric values
	for key := range q1 {
		vals1 := q1[key]
		vals2 := q2[key]
		if len(vals1) > 0 && len(vals2) > 0 {
			n1, e1 := strconv.Atoi(vals1[0])
			n2, e2 := strconv.Atoi(vals2[0])
			if e1 == nil && e2 == nil && n2 == n1+1 {
				template := strings.Replace(rawURL1, key+"="+vals1[0], key+"={page}", 1)
				return paginationDetectResult{
					PageUrlTemplate: template,
					PageStartsAt:    n1,
					SiteType:        siteType,
					Confidence:      "high",
					Notes:           fmt.Sprintf("Query parameter '%s' increments per page (starting at %d)", key, n1),
				}
			}
		}
	}

	// Case B: page param appears in URL2 only (page 1 = base URL, page 2+ = ?param=N)
	for key := range q2 {
		if _, exists := q1[key]; !exists {
			vals2 := q2[key]
			if len(vals2) > 0 {
				n2, e2 := strconv.Atoi(vals2[0])
				if e2 == nil {
					sep := "?"
					if strings.Contains(rawURL1, "?") {
						sep = "&"
					}
					template := strings.TrimRight(rawURL1, "/") + sep + key + "={page}"
					return paginationDetectResult{
						PageUrlTemplate: template,
						PageStartsAt:    n2,
						SiteType:        siteType,
						Confidence:      "high",
						Notes:           fmt.Sprintf("Page 1 uses base URL; pages %d+ add ?%s=N", n2, key),
					}
				}
			}
		}
	}

	// --- Check path segment changes ---
	path1 := strings.TrimRight(u1.Path, "/")
	path2 := strings.TrimRight(u2.Path, "/")
	parts1 := strings.Split(path1, "/")
	parts2 := strings.Split(path2, "/")

	// Case C: same number of segments, one segment changes (pure number or embedded number)
	// e.g. /page/1 → /page/2  OR  /page-1.html → /page-2.html
	if len(parts1) == len(parts2) {
		for i, p1 := range parts1 {
			p2 := parts2[i]
			if p1 != p2 {
				// Try pure numeric segment
				n1, e1 := strconv.Atoi(p1)
				n2, e2 := strconv.Atoi(p2)
				if e1 == nil && e2 == nil && n2 == n1+1 {
					templateParts := make([]string, len(parts1))
					copy(templateParts, parts1)
					templateParts[i] = "{page}"
					template := u1.Scheme + "://" + u1.Host + strings.Join(templateParts, "/")
					if u1.RawQuery != "" {
						template += "?" + u1.RawQuery
					}
					return paginationDetectResult{
						PageUrlTemplate: template,
						PageStartsAt:    n1,
						SiteType:        siteType,
						Confidence:      "high",
						Notes:           fmt.Sprintf("Page number in URL path, starting at %d", n1),
					}
				}
				// Try embedded number (e.g. "page-1.html" → "page-2.html")
				num1, _, ok1 := extractEmbeddedNumber(p1)
				num2, tmpl2, ok2 := extractEmbeddedNumber(p2)
				if ok1 && ok2 && num2 == num1+1 {
					templateParts := make([]string, len(parts1))
					copy(templateParts, parts1)
					templateParts[i] = tmpl2
					template := u1.Scheme + "://" + u1.Host + strings.Join(templateParts, "/")
					if u1.RawQuery != "" {
						template += "?" + u1.RawQuery
					}
					return paginationDetectResult{
						PageUrlTemplate: template,
						PageStartsAt:    num1,
						SiteType:        siteType,
						Confidence:      "high",
						Notes:           fmt.Sprintf("Embedded page number in '%s', starting at %d", tmpl2, num1),
					}
				}
			}
		}
	}

	// Case D: URL2 has one extra segment at end — pure number or embedded number
	// e.g. /videos → /videos/2  OR  / → /page2.html
	if len(parts2) == len(parts1)+1 {
		lastPart := parts2[len(parts2)-1]
		// D1: pure numeric segment (e.g. /videos/2)
		n2, e2 := strconv.Atoi(lastPart)
		if e2 == nil {
			template := u1.Scheme + "://" + u1.Host + path1 + "/{page}"
			if u1.RawQuery != "" {
				template += "?" + u1.RawQuery
			}
			return paginationDetectResult{
				PageUrlTemplate: template,
				PageStartsAt:    n2,
				SiteType:        siteType,
				Confidence:      "high",
				Notes:           fmt.Sprintf("Page 1 uses base URL; pages %d+ append /%d to path", n2, n2),
			}
		}
		// D2: embedded number in segment (e.g. /page2.html or /page-2/)
		num2, segTemplate, ok2 := extractEmbeddedNumber(lastPart)
		if ok2 {
			basePath := path1
			if basePath == "" {
				template := u1.Scheme + "://" + u1.Host + "/" + segTemplate
				if u1.RawQuery != "" {
					template += "?" + u1.RawQuery
				}
				return paginationDetectResult{
					PageUrlTemplate: template,
					PageStartsAt:    num2,
					SiteType:        siteType,
					Confidence:      "high",
					Notes:           fmt.Sprintf("Page 1 uses base URL; pages %d+ use pattern '%s'", num2, segTemplate),
				}
			}
			template := u1.Scheme + "://" + u1.Host + basePath + "/" + segTemplate
			if u1.RawQuery != "" {
				template += "?" + u1.RawQuery
			}
			return paginationDetectResult{
				PageUrlTemplate: template,
				PageStartsAt:    num2,
				SiteType:        siteType,
				Confidence:      "high",
				Notes:           fmt.Sprintf("Page 1 uses base URL; pages %d+ use pattern '%s'", num2, segTemplate),
			}
		}
	}

	return paginationDetectResult{
		PageUrlTemplate: "",
		PageStartsAt:    1,
		SiteType:        siteType,
		Confidence:      "low",
		Notes:           "Could not auto-detect pagination pattern from the two URLs provided. Please check that both URLs differ only by page number.",
	}
}

// detectSiteTypeFromURL returns the appropriate site type based on the hostname
func detectSiteTypeFromURL(hostname string) string {
	hostname = strings.ToLower(strings.TrimPrefix(hostname, "www."))
	customDomains := []string{"pornhub.com", "luxuretv.com", "sexuria.com", "rule34video.com", "xhamster.com", "xvideos.com"}
	for _, d := range customDomains {
		if strings.Contains(hostname, d) {
			return "custom"
		}
	}
	if strings.Contains(hostname, "youtube.com") || hostname == "youtu.be" {
		return "youtube"
	}
	if strings.Contains(hostname, "vimeo.com") {
		return "vimeo"
	}
	return "generic"
}
