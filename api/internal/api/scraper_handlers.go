package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

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

	if limit < 1 || limit > 200 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	threads, total, err := svc.GetAllThreads(limit, offset, sortBy, provider, filter)
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

	if limit < 1 || limit > 200 {
		limit = 20
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	threads, total, err := svc.SearchThreads(query, limit, offset)
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

	// Start rescraping in the background
	go func() {
		if err := svc.ScrapeThreadComplete(thread.URL); err != nil {
			log.Printf("Error rescraping thread: %v", err)
		}
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Thread rescraping started. Check activity logs for progress."))
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

	svc.SetSessionCookie(request.Cookie)

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Session cookie set successfully"))
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
