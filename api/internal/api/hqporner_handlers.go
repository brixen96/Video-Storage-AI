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

// HQPorner scraper service singleton
var hqpornerScraperService *services.HQPornerScraperService

// ensureHQPornerScraperService returns the HQPorner scraper service singleton
func ensureHQPornerScraperService() *services.HQPornerScraperService {
	if hqpornerScraperService == nil {
		videoScraperSvc := ensureVideoScraperService()
		hqpornerScraperService = services.NewHQPornerScraperService(database.GetDB(), videoScraperSvc)
	}
	return hqpornerScraperService
}

// hqpornerStartURL is the fixed start URL for HQPorner scraping
const hqpornerStartURL = "https://hqporner.com/"

// HQPornerScrapeRequestBody represents the request body for HQPorner scraping
type HQPornerScrapeRequestBody struct {
	MaxPages int `json:"maxPages"`
}

// startHQPornerScrapeHandler starts an HQPorner scrape job
func startHQPornerScrapeHandler(c *gin.Context) {
	var request HQPornerScrapeRequestBody
	// body is optional — maxPages defaults to 0 (unlimited) if not provided
	c.ShouldBindJSON(&request)

	hqpornerSvc := ensureHQPornerScraperService()
	scraperTabsSvc := ensureScraperTabsService()
	activitySvc := ensureActivityService()

	// Create a job to track progress
	job := &services.ScraperJob{
		TabID:         nil,
		JobType:       "hqporner",
		Status:        "pending",
		URL:           hqpornerStartURL,
		ErrorMessages: []string{},
	}

	createdJob, err := scraperTabsSvc.CreateScraperJob(job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to create job", err.Error()))
		return
	}

	// Start scraping in background
	go func() {
		now := time.Now()
		createdJob.Status = "running"
		createdJob.StartedAt = &now
		scraperTabsSvc.UpdateScraperJob(createdJob.ID, createdJob)

		activity, _ := activitySvc.Create(&models.ActivityLogCreate{
			TaskType: "hqporner_scraping",
			Status:   "running",
			Message:  fmt.Sprintf("Scraping HQPorner: %s", hqpornerStartURL),
			Progress: 0,
			Details: map[string]interface{}{
				"job_id":    createdJob.ID,
				"base_url":  hqpornerStartURL,
				"max_pages": request.MaxPages,
			},
		})

		var activityID int64 = 0
		if activity != nil {
			activityID = int64(activity.ID)
		}

		result, err := hqpornerSvc.ScrapeWithProgress(
			services.HQPornerScrapeRequest{
				BaseURL:    hqpornerStartURL,
				MaxPages:   request.MaxPages,
				ActivityID: activityID,
			},
			func(progress services.ScrapeProgress) {
				createdJob.PagesCompleted = progress.PagesCompleted
				createdJob.VideosFound = progress.VideosFound
				createdJob.VideosSaved = progress.VideosSaved
				createdJob.ErrorsCount = len(progress.Errors)
				scraperTabsSvc.UpdateScraperJob(createdJob.ID, createdJob)

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
						"base_url":     hqpornerStartURL,
					})
				}
			},
		)

		if err != nil {
			if strings.Contains(err.Error(), "paused by user") {
				createdJob.Status = "paused"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				log.Printf("✓ HQPorner scrape paused successfully")
			} else {
				createdJob.Status = "failed"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				if activity != nil {
					failedStatus := "failed"
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed to scrape HQPorner: %s", err.Error())),
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
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from HQPorner", result.PagesCrawled, result.TotalScraped)),
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
			"base_url": hqpornerStartURL,
		},
		fmt.Sprintf("Started HQPorner scrape: %s", hqpornerStartURL),
	))
}

// getHQPornerVideosHandler returns scraped HQPorner videos with server-side pagination,
// search, sort, and optional performer/tag filtering via JSON metadata extraction.
func getHQPornerVideosHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	performerID, _ := strconv.ParseInt(c.DefaultQuery("performer_id", "0"), 10, 64)
	tagID, _ := strconv.ParseInt(c.DefaultQuery("tag_id", "0"), 10, 64)
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "discovered_at")
	sortOrder := c.DefaultQuery("order", "desc")
	durationFilter := c.DefaultQuery("duration_filter", "")
	authorSearch := c.Query("author")
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

	hqpornerSvc := ensureHQPornerScraperService()

	result, err := hqpornerSvc.GetVideosPaginated(services.HQPornerVideoQuery{
		Page:            page,
		Limit:           limit,
		Search:          search,
		SortBy:          sortBy,
		SortOrder:       sortOrder,
		PerformerID:     performerID,
		TagID:           tagID,
		DurationFilter:  durationFilter,
		AuthorSearch:    authorSearch,
		IncludeKeywords: includeTags,
		ExcludeKeywords: excludeTags,
		MinRating:       minRating,
		MinViews:        minViews,
		MinDuration:     minDuration,
		MaxDuration:     maxDuration,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get videos", err.Error()))
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
		"HQPorner videos retrieved successfully",
	))
}

// getHQPornerPerformersHandler returns all unique DB-matched performers from HQPorner videos
func getHQPornerPerformersHandler(c *gin.Context) {
	hqpornerSvc := ensureHQPornerScraperService()
	performers, err := hqpornerSvc.GetMatchedPerformers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get performers", err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(performers, "HQPorner performers retrieved successfully"))
}

// getHQPornerTagsHandler returns all unique DB-matched tags from HQPorner videos
func getHQPornerTagsHandler(c *gin.Context) {
	hqpornerSvc := ensureHQPornerScraperService()
	tags, err := hqpornerSvc.GetMatchedTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get tags", err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(tags, "HQPorner tags retrieved successfully"))
}

// getHQPornerStatsHandler returns statistics about scraped HQPorner videos
func getHQPornerStatsHandler(c *gin.Context) {
	hqpornerSvc := ensureHQPornerScraperService()

	stats, err := hqpornerSvc.GetHQPornerStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get stats", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "HQPorner stats retrieved successfully"))
}
