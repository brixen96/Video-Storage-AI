package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var videoScraperService *services.VideoScraperService

// ensureVideoScraperService initializes the service if needed
func ensureVideoScraperService() *services.VideoScraperService {
	if videoScraperService == nil {
		videoScraperService = services.NewVideoScraperService(database.GetDB())
	}
	return videoScraperService
}

// scrapeVideoHandler handles scraping a video from a URL
func scrapeVideoHandler(c *gin.Context) {
	svc := ensureVideoScraperService()
	consoleLogSvc := ensureConsoleLogService()

	var request services.ScrapeVideoRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Validate URL
	if request.URL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"URL is required",
			"",
		))
		return
	}

	// Log scrape start
	consoleLogSvc.LogAPI("info", "Video scraping started", map[string]interface{}{
		"url":       request.URL,
		"site_type": request.SiteType,
		"max_pages": request.Options.MaxPages,
	})

	// Determine if this is a listing page or single video based on URL pattern
	// If MaxPages is not set (0), default to unlimited pagination
	if request.Options.MaxPages == 0 {
		request.Options.MaxPages = 9999 // Effectively unlimited
	}

	// Check if URL looks like a listing page (contains /videos, /videos/, /videos/2/, etc.)
	isListingPage := false
	if parsedURL, err := url.Parse(request.URL); err == nil {
		path := parsedURL.Path
		// Listing page patterns: /videos, /videos/, /videos/2/, etc.
		if strings.Contains(path, "/videos") {
			isListingPage = true
		}
	}

	// If it's a listing page, use ScrapePage
	if isListingPage {
		result, err := svc.ScrapePage(request)
		if err != nil {
			errMsg := fmt.Sprintf("Page scraping failed: %v", err)
			consoleLogSvc.LogAPI("error", errMsg, map[string]interface{}{
				"url":   request.URL,
				"error": err.Error(),
			})
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to scrape page",
				err.Error(),
			))
			return
		}

		consoleLogSvc.LogAPI("info", "Page scraped successfully", map[string]interface{}{
			"url":           request.URL,
			"videos_found":  result.TotalScraped,
			"pages_crawled": result.PagesCrawled,
		})

		c.JSON(http.StatusOK, models.SuccessResponse(result, fmt.Sprintf("Scraped %d videos from %d pages", result.TotalScraped, result.PagesCrawled)))
		return
	}

	// Otherwise, scrape a single video
	video, err := svc.ScrapeVideo(request)
	if err != nil {
		consoleLogSvc.LogAPI("error", "Video scraping failed", map[string]interface{}{
			"url":   request.URL,
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to scrape video",
			err.Error(),
		))
		return
	}

	consoleLogSvc.LogAPI("info", "Video scraped successfully", map[string]interface{}{
		"url":   request.URL,
		"title": video.Title,
	})

	c.JSON(http.StatusOK, models.SuccessResponse(video, "Video scraped successfully"))
}

// getScrapedVideosHandler retrieves all scraped videos
func getScrapedVideosHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	// Check if we want to include tab info
	includeTabInfo := c.Query("includeTabInfo") == "true"

	// Check if filtering by source
	source := c.Query("source")

	var videos []*services.ScrapedVideo
	var err error

	if source != "" {
		// Filter by source for better performance
		videos, err = svc.GetScrapedVideosBySource(source)
	} else if includeTabInfo {
		videos, err = svc.GetScrapedVideosWithTabInfo()
	} else {
		videos, err = svc.GetScrapedVideos()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve scraped videos",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(videos, "Scraped videos retrieved successfully"))
}

// testScraperHandler tests the scraper on a URL without saving to database
func testScraperHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"URL parameter is required",
			"",
		))
		return
	}

	result, err := svc.TestScrapeListingPage(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to test scraper",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result, "Test completed"))
}

// deleteAllScrapedVideosHandler deletes all scraped videos from database
func deleteAllScrapedVideosHandler(c *gin.Context) {
	db := database.GetDB()

	result, err := db.Exec("DELETE FROM scraped_videos")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete scraped videos",
			err.Error(),
		))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted": rowsAffected,
	}, fmt.Sprintf("Deleted %d scraped videos", rowsAffected)))
}

// deleteScrapedVideosBySiteTypeHandler deletes all scraped videos for a specific site type
func deleteScrapedVideosBySiteTypeHandler(c *gin.Context) {
	db := database.GetDB()

	siteType := c.Param("siteType")
	if siteType == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Site type is required",
			"",
		))
		return
	}

	result, err := db.Exec("DELETE FROM scraped_videos WHERE site_type = ?", siteType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete scraped videos",
			err.Error(),
		))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted":   rowsAffected,
		"site_type": siteType,
	}, fmt.Sprintf("Deleted %d %s videos", rowsAffected, siteType)))
}

// bulkDeleteScrapedVideosHandler deletes multiple scraped videos by IDs
func bulkDeleteScrapedVideosHandler(c *gin.Context) {
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Database connection not available",
			"",
		))
		return
	}

	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"No IDs provided",
			"",
		))
		return
	}

	// Build query with placeholders
	placeholders := make([]string, len(req.IDs))
	args := make([]interface{}, len(req.IDs))
	for i, id := range req.IDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM scraped_videos WHERE id IN (%s)", strings.Join(placeholders, ","))
	result, err := db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete scraped videos",
			err.Error(),
		))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted":   rowsAffected,
		"requested": len(req.IDs),
	}, fmt.Sprintf("Deleted %d videos", rowsAffected)))
}

// findDuplicatesHandler finds duplicate scraped videos
func findDuplicatesHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	stats, err := svc.FindDuplicates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to find duplicates",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, fmt.Sprintf("Found %d potential duplicates", stats.TotalDuplicates)))
}

// deleteDuplicatesHandler deletes duplicate scraped videos
func deleteDuplicatesHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	// Get strategy from query param (keep_first, keep_last, keep_best)
	strategy := c.DefaultQuery("strategy", "keep_best")

	deleted, err := svc.DeleteDuplicates(strategy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete duplicates",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted":  deleted,
		"strategy": strategy,
	}, fmt.Sprintf("Deleted %d duplicate videos using %s strategy", deleted, strategy)))
}

// getScrapedVideoHandler retrieves a single scraped video by ID
func getScrapedVideoHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Video ID is required",
			"",
		))
		return
	}

	// Convert ID to int64
	var videoID int64
	if _, err := fmt.Sscanf(id, "%d", &videoID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid video ID",
			err.Error(),
		))
		return
	}

	video, err := svc.GetScrapedVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Video not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(video, "Video retrieved successfully"))
}
