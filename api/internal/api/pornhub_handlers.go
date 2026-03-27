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

// PornHub scraper service singleton
var pornHubScraperService *services.PornHubScraperService

// ensurePornHubScraperService returns the PornHub scraper service singleton
func ensurePornHubScraperService() *services.PornHubScraperService {
	if pornHubScraperService == nil {
		videoScraperSvc := ensureVideoScraperService()
		pornHubScraperService = services.NewPornHubScraperService(database.GetDB(), videoScraperSvc)
	}
	return pornHubScraperService
}

// PornHubSearchRequestBody represents the request body for PornHub search
type PornHubSearchRequestBody struct {
	Keyword  string `json:"keyword" binding:"required"`
	MaxPages int    `json:"maxPages"`
}

// startPornHubScrapeHandler starts a PornHub scrape job by keyword
func startPornHubScrapeHandler(c *gin.Context) {
	var request PornHubSearchRequestBody
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if request.Keyword == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Keyword is required",
			"",
		))
		return
	}

	pornHubSvc := ensurePornHubScraperService()
	scraperTabsSvc := ensureScraperTabsService()
	activitySvc := ensureActivityService()

	// Create a new job to track progress
	job := &services.ScraperJob{
		TabID:         nil, // No tab for PornHub keyword searches
		JobType:       "pornhub",
		Status:        "pending",
		URL:           fmt.Sprintf("https://www.pornhub.com/video/search?search=%s", request.Keyword),
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
			TaskType: "pornhub_scraping",
			Status:   "running",
			Message:  fmt.Sprintf("Scraping PornHub for keyword: %s", request.Keyword),
			Progress: 0,
			Details: map[string]interface{}{
				"job_id":    createdJob.ID,
				"keyword":   request.Keyword,
				"max_pages": request.MaxPages,
			},
		})

		// Get activity ID for pause checking
		var activityID int64 = 0
		if activity != nil {
			activityID = int64(activity.ID)
		}

		// Execute scrape with progress callback
		result, err := pornHubSvc.ScrapeByKeywordWithProgress(
			services.PornHubSearchRequest{
				Keyword:    request.Keyword,
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

				// Update activity and save checkpoint for resume
				if activity != nil {
					progressPercent := 0
					// NOTE: Do NOT update Details here - it would overwrite the original details
					// that contain keyword needed for resume
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s' (page %d)", progress.PagesCompleted, progress.VideosFound, request.Keyword, progress.CurrentPage)),
						Progress: &progressPercent,
					})

					// Save checkpoint for resume capability (separate from details)
					// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
					activitySvc.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
						"current_page":  progress.CurrentPage, // Actual page number for correct resume
						"videos_found":  progress.VideosFound,
						"videos_saved":  progress.VideosSaved,
						"keyword":       request.Keyword,
					})
				}
			},
		)

		// Update job with final status
		if err != nil {
			// Check if this was a pause (not a real error)
			if strings.Contains(err.Error(), "paused by user") {
				createdJob.Status = "paused"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				// Don't update activity - it's already marked as paused
				log.Printf("✓ PornHub scrape paused successfully, can be resumed")
			} else {
				createdJob.Status = "failed"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				if activity != nil {
					failedStatus := "failed"
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed to scrape PornHub: %s", err.Error())),
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
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s'", result.PagesCrawled, result.TotalScraped, request.Keyword)),
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
			"job_id":  createdJob.ID,
			"keyword": request.Keyword,
		},
		fmt.Sprintf("Started PornHub scrape for keyword: %s", request.Keyword),
	))
}

// getPornHubKeywordsHandler returns suggested keywords for PornHub scraping
func getPornHubKeywordsHandler(c *gin.Context) {
	pornHubSvc := ensurePornHubScraperService()
	keywords := pornHubSvc.GetPornHubKeywords()

	c.JSON(http.StatusOK, models.SuccessResponse(keywords, "Keywords retrieved successfully"))
}

// getPornHubVideosHandler returns scraped PornHub videos with pagination
func getPornHubVideosHandler(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	keyword := c.Query("keyword")
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

	videoScraperSvc := ensureVideoScraperService()

	// Use paginated query
	result, err := videoScraperSvc.GetScrapedVideosPaginated(services.ScrapedVideosQuery{
		Source:          "www.pornhub.com",
		Keyword:         keyword,
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
		"PornHub videos retrieved successfully",
	))
}

// getPornHubStatsHandler returns statistics about scraped PornHub videos
func getPornHubStatsHandler(c *gin.Context) {
	videoScraperSvc := ensureVideoScraperService()

	videos, err := videoScraperSvc.GetScrapedVideos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get videos",
			err.Error(),
		))
		return
	}

	// Count PornHub videos and collect keywords
	totalVideos := 0
	keywordCounts := make(map[string]int)

	for _, v := range videos {
		if v.Source == "www.pornhub.com" {
			totalVideos++
			for _, tag := range v.Tags {
				keywordCounts[tag]++
			}
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		map[string]interface{}{
			"total_videos":   totalVideos,
			"keyword_counts": keywordCounts,
		},
		"PornHub stats retrieved successfully",
	))
}
