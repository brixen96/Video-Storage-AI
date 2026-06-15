package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"

	"github.com/gin-gonic/gin"
)

// Motherless scraper service singleton
var motherlessScraperService *services.MotherlessScraperService

// ensureMotherlessScraperService returns the Motherless scraper service singleton
func ensureMotherlessScraperService() *services.MotherlessScraperService {
	if motherlessScraperService == nil {
		videoScraperSvc := ensureVideoScraperService()
		motherlessScraperService = services.NewMotherlessScraperService(database.GetDB(), videoScraperSvc)
	}
	return motherlessScraperService
}

// MotherlessScrapeRequestBody represents the request body for Motherless scraping
type MotherlessScrapeRequestBody struct {
	SearchURL string `json:"searchUrl" binding:"required"`
	MaxPages  int    `json:"maxPages"`
}

// startMotherlessScrapeHandler starts a Motherless scrape job by search URL
func startMotherlessScrapeHandler(c *gin.Context) {
	var request MotherlessScrapeRequestBody
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if request.SearchURL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Search URL is required",
			"",
		))
		return
	}

	motherlessSvc := ensureMotherlessScraperService()
	scraperTabsSvc := ensureScraperTabsService()
	activitySvc := ensureActivityService()

	searchTerm := services.ExtractSearchTerm(request.SearchURL)

	// Create a new job to track progress
	job := &services.ScraperJob{
		TabID:         nil,
		JobType:       "motherless",
		Status:        "pending",
		URL:           request.SearchURL,
		ErrorMessages: []string{},
	}

	createdJob, err := scraperTabsSvc.CreateScraperJob(job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create job",
			err.Error(),
		))
		return
	}

	// Start scraping in background
	go func() {
		// Update job status to running
		now := time.Now()
		createdJob.Status = "running"
		createdJob.StartedAt = &now
		scraperTabsSvc.UpdateScraperJob(createdJob.ID, createdJob)

		// Create activity log entry
		activity, _ := activitySvc.Create(&models.ActivityLogCreate{
			TaskType: "motherless_scraping",
			Status:   "running",
			Message:  fmt.Sprintf("Scraping Motherless for: %s", searchTerm),
			Progress: 0,
			Details: map[string]interface{}{
				"job_id":      createdJob.ID,
				"search_url":  request.SearchURL,
				"search_term": searchTerm,
				"max_pages":   request.MaxPages,
			},
		})

		var activityID int64 = 0
		if activity != nil {
			activityID = int64(activity.ID)
		}

		// Execute scrape with progress callback
		result, err := motherlessSvc.ScrapeSearchWithProgress(
			services.MotherlessScrapeRequest{
				SearchURL:  request.SearchURL,
				MaxPages:   request.MaxPages,
				ActivityID: activityID,
			},
			func(progress services.ScrapeProgress) {
				// Update job progress
				createdJob.PagesCompleted = progress.PagesCompleted
				createdJob.VideosFound = progress.VideosFound
				createdJob.VideosSaved = progress.VideosSaved
				createdJob.ErrorsCount = len(progress.Errors)
				scraperTabsSvc.UpdateScraperJob(createdJob.ID, createdJob)

				// Update activity and save checkpoint
				if activity != nil {
					progressPercent := 0
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s' (page %d)", progress.PagesCompleted, progress.VideosFound, searchTerm, progress.CurrentPage)),
						Progress: &progressPercent,
					})

					activitySvc.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
						"current_page": progress.CurrentPage,
						"videos_found": progress.VideosFound,
						"videos_saved": progress.VideosSaved,
						"search_url":   request.SearchURL,
						"search_term":  searchTerm,
					})
				}
			},
		)

		// Update job with final status
		if err != nil {
			if strings.Contains(err.Error(), "paused by user") {
				createdJob.Status = "paused"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				log.Printf("✓ Motherless scrape paused successfully, can be resumed")
			} else {
				createdJob.Status = "failed"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				if activity != nil {
					failedStatus := "failed"
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed to scrape Motherless: %s", err.Error())),
						Completed: true,
					})
				}
			}
		} else {
			createdJob.Status = "completed"
			createdJob.PagesCompleted = result.PagesCrawled
			createdJob.VideosFound = result.TotalScraped
			createdJob.VideosSaved = result.NewVideos + result.UpdatedVideos
			createdJob.ErrorsCount = len(result.Errors)

			if activity != nil {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s'", result.PagesCrawled, result.TotalScraped, searchTerm)),
					Progress:  &progress100,
					Completed: true,
					Details: map[string]interface{}{
						"pages_crawled":  result.PagesCrawled,
						"total_scraped":  result.TotalScraped,
						"new_videos":     result.NewVideos,
						"updated_videos": result.UpdatedVideos,
					},
				})
			}
		}

		completedAt := time.Now()
		createdJob.CompletedAt = &completedAt
		scraperTabsSvc.UpdateScraperJob(createdJob.ID, createdJob)
	}()

	c.JSON(http.StatusOK, models.SuccessResponse(
		map[string]interface{}{
			"job_id":      createdJob.ID,
			"search_url":  request.SearchURL,
			"search_term": searchTerm,
		},
		fmt.Sprintf("Started Motherless scrape for: %s", searchTerm),
	))
}

// getMotherlessVideosHandler returns scraped Motherless videos with pagination
func getMotherlessVideosHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	category := c.Query("category")
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "discovered_at")
	sortOrder := c.DefaultQuery("order", "desc")
	durationFilter := c.DefaultQuery("duration_filter", "")
	onlyDownloaded := c.DefaultQuery("downloaded_only", "false") == "true"
	var includeTags, excludeTags []string
	if v := c.Query("include_tags"); v != "" {
		includeTags = strings.Split(v, ",")
	}
	if v := c.Query("exclude_tags"); v != "" {
		excludeTags = strings.Split(v, ",")
	}
	minRating, _ := strconv.Atoi(c.DefaultQuery("min_rating", "0"))
	minViews, _ := strconv.Atoi(c.DefaultQuery("min_views", "0"))
	minDuration, _ := strconv.ParseFloat(c.DefaultQuery("min_duration", "0"), 64)
	maxDuration, _ := strconv.ParseFloat(c.DefaultQuery("max_duration", "0"), 64)

	videoScraperSvc := ensureVideoScraperService()

	result, err := videoScraperSvc.GetScrapedVideosPaginated(services.ScrapedVideosQuery{
		Source:          "motherless.com",
		Category:        category,
		Search:          search,
		Page:            page,
		Limit:           limit,
		SortBy:          sortBy,
		SortOrder:       sortOrder,
		DurationFilter:  durationFilter,
		OnlyDownloaded:  onlyDownloaded,
		IncludeKeywords: includeTags,
		ExcludeKeywords: excludeTags,
		MinRating:       minRating,
		MinViews:        minViews,
		MinDuration:     minDuration,
		MaxDuration:     maxDuration,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get videos",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		map[string]interface{}{
			"videos":     result.Videos,
			"count":      len(result.Videos),
			"total":      result.Total,
			"page":       result.Page,
			"limit":      result.Limit,
			"totalPages": result.TotalPages,
		},
		"Motherless videos retrieved successfully",
	))
}

// getMotherlessStatsHandler returns statistics about scraped Motherless videos
func getMotherlessStatsHandler(c *gin.Context) {
	motherlessSvc := ensureMotherlessScraperService()

	stats, err := motherlessSvc.GetMotherlessStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Motherless stats retrieved successfully"))
}

// getMotherlessSearchTermsHandler returns distinct search terms with counts
func getMotherlessSearchTermsHandler(c *gin.Context) {
	motherlessSvc := ensureMotherlessScraperService()

	terms, err := motherlessSvc.GetMotherlessSearchTerms()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get search terms",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(terms, "Search terms retrieved successfully"))
}
