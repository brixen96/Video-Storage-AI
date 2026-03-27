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

// LeakVids scraper service singleton
var leakVidsScraperService *services.LeakVidsScraperService

// ensureLeakVidsScraperService returns the LeakVids scraper service singleton
func ensureLeakVidsScraperService() *services.LeakVidsScraperService {
	if leakVidsScraperService == nil {
		videoScraperSvc := ensureVideoScraperService()
		leakVidsScraperService = services.NewLeakVidsScraperService(database.GetDB(), videoScraperSvc)
	}
	return leakVidsScraperService
}

// LeakVidsScrapeRequestBody represents the request body for LeakVids scraping
type LeakVidsScrapeRequestBody struct {
	BaseURL  string `json:"baseUrl" binding:"required"`
	MaxPages int    `json:"maxPages"`
}

// startLeakVidsScrapeHandler starts a LeakVids scrape job
func startLeakVidsScrapeHandler(c *gin.Context) {
	var request LeakVidsScrapeRequestBody
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if request.BaseURL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Base URL is required",
			"",
		))
		return
	}

	leakVidsSvc := ensureLeakVidsScraperService()
	scraperTabsSvc := ensureScraperTabsService()
	activitySvc := ensureActivityService()

	// Create a new job to track progress
	job := &services.ScraperJob{
		TabID:         nil,
		JobType:       "leakvids",
		Status:        "pending",
		URL:           request.BaseURL,
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
			TaskType: "leakvids_scraping",
			Status:   "running",
			Message:  fmt.Sprintf("Scraping LeakVids: %s", request.BaseURL),
			Progress: 0,
			Details: map[string]interface{}{
				"job_id":    createdJob.ID,
				"base_url":  request.BaseURL,
				"max_pages": request.MaxPages,
			},
		})

		var activityID int64 = 0
		if activity != nil {
			activityID = int64(activity.ID)
		}

		// Execute scrape with progress callback
		result, err := leakVidsSvc.ScrapeWithProgress(
			services.LeakVidsScrapeRequest{
				BaseURL:    request.BaseURL,
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
						Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
						Progress: &progressPercent,
					})

					activitySvc.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
						"current_page": progress.CurrentPage,
						"videos_found": progress.VideosFound,
						"videos_saved": progress.VideosSaved,
						"base_url":     request.BaseURL,
					})
				}
			},
		)

		// Update job with final status
		if err != nil {
			if strings.Contains(err.Error(), "paused by user") {
				createdJob.Status = "paused"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				log.Printf("✓ LeakVids scrape paused successfully, can be resumed")
			} else {
				createdJob.Status = "failed"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				if activity != nil {
					failedStatus := "failed"
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed to scrape LeakVids: %s", err.Error())),
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
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LeakVids", result.PagesCrawled, result.TotalScraped)),
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
			"job_id":   createdJob.ID,
			"base_url": request.BaseURL,
		},
		fmt.Sprintf("Started LeakVids scrape: %s", request.BaseURL),
	))
}

// getLeakVidsVideosHandler returns scraped LeakVids videos with pagination
func getLeakVidsVideosHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "discovered_at")
	sortOrder := c.DefaultQuery("order", "desc")
	durationFilter := c.DefaultQuery("duration_filter", "")
	onlyDownloaded := c.DefaultQuery("downloaded_only", "false") == "true"
	minRating, _ := strconv.Atoi(c.DefaultQuery("min_rating", "0"))
	minViews, _ := strconv.Atoi(c.DefaultQuery("min_views", "0"))
	var includeTags, excludeTags []string
	if v := c.Query("include_tags"); v != "" {
		includeTags = strings.Split(v, ",")
	}
	if v := c.Query("exclude_tags"); v != "" {
		excludeTags = strings.Split(v, ",")
	}

	videoScraperSvc := ensureVideoScraperService()

	result, err := videoScraperSvc.GetScrapedVideosPaginated(services.ScrapedVideosQuery{
		Source:          "leakvids.com",
		Search:          search,
		Page:            page,
		Limit:           limit,
		SortBy:          sortBy,
		SortOrder:       sortOrder,
		DurationFilter:  durationFilter,
		OnlyDownloaded:  onlyDownloaded,
		MinRating:       minRating,
		MinViews:        minViews,
		IncludeKeywords: includeTags,
		ExcludeKeywords: excludeTags,
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
		"LeakVids videos retrieved successfully",
	))
}

// getLeakVidsStatsHandler returns statistics about scraped LeakVids videos
func getLeakVidsStatsHandler(c *gin.Context) {
	leakVidsSvc := ensureLeakVidsScraperService()

	stats, err := leakVidsSvc.GetLeakVidsStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "LeakVids stats retrieved successfully"))
}
