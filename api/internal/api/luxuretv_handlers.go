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

// LuxureTV scraper service singleton
var luxureTVScraperService *services.LuxureTVScraperService

// ensureLuxureTVScraperService returns the LuxureTV scraper service singleton
func ensureLuxureTVScraperService() *services.LuxureTVScraperService {
	if luxureTVScraperService == nil {
		videoScraperSvc := ensureVideoScraperService()
		luxureTVScraperService = services.NewLuxureTVScraperService(database.GetDB(), videoScraperSvc)
	}
	return luxureTVScraperService
}

// LuxureTVScrapeRequestBody represents the request body for LuxureTV scrape
type LuxureTVScrapeRequestBody struct {
	CategoryURL string `json:"categoryUrl" binding:"required"`
	MaxPages    int    `json:"maxPages"`
}

// startLuxureTVScrapeHandler starts a LuxureTV scrape job by category URL
func startLuxureTVScrapeHandler(c *gin.Context) {
	var request LuxureTVScrapeRequestBody
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if request.CategoryURL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Category URL is required",
			"",
		))
		return
	}

	luxureTVSvc := ensureLuxureTVScraperService()
	scraperTabsSvc := ensureScraperTabsService()
	activitySvc := ensureActivityService()

	// Create a new job to track progress
	job := &services.ScraperJob{
		TabID:         nil,
		JobType:       "luxuretv",
		Status:        "pending",
		URL:           request.CategoryURL,
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
			TaskType: "luxuretv_scraping",
			Status:   "running",
			Message:  fmt.Sprintf("Scraping LuxureTV category: %s", request.CategoryURL),
			Progress: 0,
			Details: map[string]interface{}{
				"job_id":       createdJob.ID,
				"category_url": request.CategoryURL,
				"max_pages":    request.MaxPages,
			},
		})

		// Get activity ID for pause checking
		var activityID int64 = 0
		if activity != nil {
			activityID = int64(activity.ID)
		}

		// Execute scrape with progress callback
		result, err := luxureTVSvc.ScrapeCategoryWithProgress(
			services.LuxureTVScrapeRequest{
				CategoryURL: request.CategoryURL,
				MaxPages:    request.MaxPages,
				ActivityID:  activityID,
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
					// that contain category_url needed for resume
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
						Progress: &progressPercent,
					})

					// Save checkpoint for resume capability (separate from details)
					// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
					activitySvc.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
						"current_page":   progress.CurrentPage, // Actual page number for correct resume
						"videos_found":   progress.VideosFound,
						"videos_saved":   progress.VideosSaved,
						"category_url":   request.CategoryURL,
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
				log.Printf("✓ LuxureTV scrape paused successfully, can be resumed")
			} else {
				createdJob.Status = "failed"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				if activity != nil {
					failedStatus := "failed"
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed to scrape LuxureTV: %s", err.Error())),
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
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV", result.PagesCrawled, result.TotalScraped)),
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
			"job_id":       createdJob.ID,
			"category_url": request.CategoryURL,
		},
		fmt.Sprintf("Started LuxureTV scrape for: %s", request.CategoryURL),
	))
}

// getLuxureTVCategoriesHandler returns available LuxureTV categories
func getLuxureTVCategoriesHandler(c *gin.Context) {
	luxureTVSvc := ensureLuxureTVScraperService()
	categories := luxureTVSvc.GetLuxureTVCategories()

	c.JSON(http.StatusOK, models.SuccessResponse(categories, "Categories retrieved successfully"))
}

// getLuxureTVVideosHandler returns scraped LuxureTV videos with pagination
func getLuxureTVVideosHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "discovered_at")
	sortOrder := c.DefaultQuery("order", "desc")
	performerID, _ := strconv.ParseInt(c.DefaultQuery("performer_id", "0"), 10, 64)
	tagID, _ := strconv.ParseInt(c.DefaultQuery("tag_id", "0"), 10, 64)
	durationFilter := c.DefaultQuery("duration_filter", "")
	category := c.Query("category")
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
		SiteType:        "luxuretv",
		Search:          search,
		Page:            page,
		Limit:           limit,
		SortBy:          sortBy,
		SortOrder:       sortOrder,
		PerformerID:     performerID,
		TagID:           tagID,
		DurationFilter:  durationFilter,
		Category:        category,
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
		"LuxureTV videos retrieved successfully",
	))
}

// getLuxureTVPerformersHandler returns all unique matched performers in LuxureTV videos
func getLuxureTVPerformersHandler(c *gin.Context) {
	luxureTVSvc := ensureLuxureTVScraperService()
	performers, err := luxureTVSvc.GetMatchedPerformers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get performers", err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(performers, "LuxureTV performers retrieved successfully"))
}

// getLuxureTVTagsHandler returns all unique matched tags in LuxureTV videos
func getLuxureTVTagsHandler(c *gin.Context) {
	luxureTVSvc := ensureLuxureTVScraperService()
	tags, err := luxureTVSvc.GetMatchedTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get tags", err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(tags, "LuxureTV tags retrieved successfully"))
}

// getLuxureTVStatsHandler returns statistics about scraped LuxureTV videos
func getLuxureTVStatsHandler(c *gin.Context) {
	videoScraperSvc := ensureVideoScraperService()

	videos, err := videoScraperSvc.GetScrapedVideos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get videos",
			err.Error(),
		))
		return
	}

	// Count LuxureTV videos
	totalVideos := 0
	totalViews := 0
	avgRating := 0
	ratingCount := 0

	for _, v := range videos {
		if v.SiteType == "luxuretv" || v.Source == "en.luxuretv.com" {
			totalVideos++
			totalViews += v.ViewCount
			if v.Rating > 0 {
				avgRating += v.Rating
				ratingCount++
			}
		}
	}

	if ratingCount > 0 {
		avgRating = avgRating / ratingCount
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		map[string]interface{}{
			"total_videos": totalVideos,
			"total_views":  totalViews,
			"avg_rating":   avgRating,
		},
		"LuxureTV stats retrieved successfully",
	))
}

// fetchLuxureTVEmbedIDsHandler fetches embed IDs for LuxureTV videos that don't have them
func fetchLuxureTVEmbedIDsHandler(c *gin.Context) {
	luxureTVSvc := ensureLuxureTVScraperService()
	videoScraperSvc := ensureVideoScraperService()

	// Get all LuxureTV videos
	videos, err := videoScraperSvc.GetScrapedVideos()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get videos",
			err.Error(),
		))
		return
	}

	// Filter to LuxureTV videos only
	var luxureTVVideos []*services.ScrapedVideo
	for _, v := range videos {
		if v.SiteType == "luxuretv" || v.Source == "en.luxuretv.com" {
			luxureTVVideos = append(luxureTVVideos, v)
		}
	}

	if len(luxureTVVideos) == 0 {
		c.JSON(http.StatusOK, models.SuccessResponse(
			map[string]interface{}{
				"updated": 0,
				"message": "No LuxureTV videos to process",
			},
			"No videos to process",
		))
		return
	}

	// Start fetching embed IDs in background
	go func() {
		updated, err := luxureTVSvc.FetchEmbedIDsForVideos(luxureTVVideos)
		if err != nil {
			fmt.Printf("Error fetching embed IDs: %v\n", err)
		}
		fmt.Printf("Fetched embed IDs for %d videos\n", updated)
	}()

	c.JSON(http.StatusOK, models.SuccessResponse(
		map[string]interface{}{
			"videos_to_process": len(luxureTVVideos),
			"message":           "Started fetching embed IDs in background",
		},
		"Started fetching embed IDs",
	))
}
