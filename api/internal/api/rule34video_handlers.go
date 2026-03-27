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

// Rule34Video scraper service singleton
var rule34VideoScraperService *services.Rule34VideoScraperService

// ensureRule34VideoScraperService returns the Rule34Video scraper service singleton
func ensureRule34VideoScraperService() *services.Rule34VideoScraperService {
	if rule34VideoScraperService == nil {
		videoScraperSvc := ensureVideoScraperService()
		rule34VideoScraperService = services.NewRule34VideoScraperService(database.GetDB(), videoScraperSvc)
	}
	return rule34VideoScraperService
}

// Rule34VideoScrapeRequestBody represents the request body for Rule34Video scraping
type Rule34VideoScrapeRequestBody struct {
	MaxPages     int      `json:"maxPages"`
	SortBy       string   `json:"sortBy"`       // post_date, video_viewed, rating, duration, pseudo_rand
	DateFilter   string   `json:"dateFilter"`   // post_date_past_24_hours, etc.
	CategorySlug string   `json:"categorySlug"` // e.g. "3d"
	ModelSlug    string   `json:"modelSlug"`    // e.g. "anianiboy"
	IncludeTags  []string `json:"includeTags"`
	ExcludeTags  []string `json:"excludeTags"`
	FetchDetails bool     `json:"fetchDetails"` // fetch detail page per video
}

// startRule34VideoScrapeHandler starts a Rule34Video scrape job
func startRule34VideoScrapeHandler(c *gin.Context) {
	var request Rule34VideoScrapeRequestBody
	if err := c.ShouldBindJSON(&request); err != nil {
		// Use defaults if no body provided
		request.MaxPages = 0
		request.SortBy = "post_date"
	}

	rule34Svc := ensureRule34VideoScraperService()
	scraperTabsSvc := ensureScraperTabsService()
	activitySvc := ensureActivityService()

	// Build job URL based on filters
	jobURL := "https://rule34video.com/"
	if request.CategorySlug != "" {
		jobURL = "https://rule34video.com/categories/" + request.CategorySlug + "/"
	} else if request.ModelSlug != "" {
		jobURL = "https://rule34video.com/models/" + request.ModelSlug + "/"
	}

	// Create a new job to track progress
	job := &services.ScraperJob{
		TabID:         nil,
		JobType:       "rule34video",
		Status:        "pending",
		URL:           jobURL,
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

	sortLabel := "Most Recent"
	for _, opt := range rule34Svc.GetSortOptions() {
		if opt["value"] == request.SortBy {
			sortLabel = opt["label"]
			break
		}
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
			TaskType: "rule34video_scraping",
			Status:   "running",
			Message:  fmt.Sprintf("Scraping Rule34Video (Straight filter, %s)", sortLabel),
			Progress: 0,
			Details: map[string]interface{}{
				"job_id":    createdJob.ID,
				"sort_by":   request.SortBy,
				"max_pages": request.MaxPages,
				"filter":    "straight",
			},
		})

		// Get activity ID for pause checking
		var activityID int64 = 0
		if activity != nil {
			activityID = int64(activity.ID)
		}

		// Execute scrape with progress callback
		result, err := rule34Svc.ScrapeWithProgress(
			services.Rule34VideoScrapeRequest{
				MaxPages:     request.MaxPages,
				SortBy:       request.SortBy,
				DateFilter:   request.DateFilter,
				CategorySlug: request.CategorySlug,
				ModelSlug:    request.ModelSlug,
				IncludeTags:  request.IncludeTags,
				ExcludeTags:  request.ExcludeTags,
				FetchDetails: request.FetchDetails,
				ActivityID:   activityID,
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
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
						Progress: &progressPercent,
					})

					// Save checkpoint for resume capability
					activitySvc.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
						"current_page": progress.CurrentPage,
						"videos_found": progress.VideosFound,
						"videos_saved": progress.VideosSaved,
						"sort_by":      request.SortBy,
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
				log.Printf("✓ Rule34Video scrape paused successfully, can be resumed")
			} else {
				createdJob.Status = "failed"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				if activity != nil {
					failedStatus := "failed"
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed to scrape Rule34Video: %s", err.Error())),
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
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from Rule34Video", result.PagesCrawled, result.TotalScraped)),
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
			"sort_by": request.SortBy,
			"filter":  "straight",
		},
		fmt.Sprintf("Started Rule34Video scrape (%s, Straight filter)", sortLabel),
	))
}

// getRule34VideoSortOptionsHandler returns available sort options
func getRule34VideoSortOptionsHandler(c *gin.Context) {
	rule34Svc := ensureRule34VideoScraperService()
	options := rule34Svc.GetSortOptions()

	c.JSON(http.StatusOK, models.SuccessResponse(options, "Sort options retrieved successfully"))
}

// getRule34VideoVideosHandler returns scraped Rule34Video videos with pagination
func getRule34VideoVideosHandler(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort", "discovered_at")
	sortOrder := c.DefaultQuery("order", "desc")
	durationFilter := c.DefaultQuery("duration_filter", "")
	onlyDownloaded := c.DefaultQuery("downloaded_only", "false") == "true"
	authorSearch := c.Query("author")
	categorySearch := c.Query("category")
	includeTagsStr := c.Query("include_tags")
	excludeTagsStr := c.Query("exclude_tags")

	var includeTags, excludeTags []string
	if includeTagsStr != "" {
		for _, t := range strings.Split(includeTagsStr, ",") {
			if t = strings.TrimSpace(t); t != "" {
				includeTags = append(includeTags, t)
			}
		}
	}
	if excludeTagsStr != "" {
		for _, t := range strings.Split(excludeTagsStr, ",") {
			if t = strings.TrimSpace(t); t != "" {
				excludeTags = append(excludeTags, t)
			}
		}
	}
	minRating, _ := strconv.Atoi(c.DefaultQuery("min_rating", "0"))
	minViews, _ := strconv.Atoi(c.DefaultQuery("min_views", "0"))

	videoScraperSvc := ensureVideoScraperService()

	// Use paginated query
	result, err := videoScraperSvc.GetScrapedVideosPaginated(services.ScrapedVideosQuery{
		Source:          "rule34video.com",
		Search:          search,
		Page:            page,
		Limit:           limit,
		SortBy:          sortBy,
		SortOrder:       sortOrder,
		DurationFilter:  durationFilter,
		OnlyDownloaded:  onlyDownloaded,
		AuthorSearch:    authorSearch,
		CategorySearch:  categorySearch,
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
		"Rule34Video videos retrieved successfully",
	))
}

// getRule34VideoStatsHandler returns statistics about scraped Rule34Video videos
func getRule34VideoStatsHandler(c *gin.Context) {
	rule34Svc := ensureRule34VideoScraperService()

	stats, err := rule34Svc.GetRule34Stats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Rule34Video stats retrieved successfully"))
}

// getRule34VideoDownloadURLHandler fetches the download URL for a Rule34Video video
func getRule34VideoDownloadURLHandler(c *gin.Context) {
	videoURL := c.Query("url")
	if videoURL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Missing parameter",
			"Video URL parameter is required",
		))
		return
	}

	rule34Svc := ensureRule34VideoScraperService()

	downloadURL, err := rule34Svc.FetchDownloadURL(videoURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get download URL",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		map[string]string{
			"download_url": downloadURL,
		},
		"Download URL retrieved successfully",
	))
}

// getRule34VideoDetailsHandler fetches full details (download URLs, tags, categories) for a video
func getRule34VideoDetailsHandler(c *gin.Context) {
	videoURL := c.Query("url")
	if videoURL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Missing parameter", "url is required"))
		return
	}

	rule34Svc := ensureRule34VideoScraperService()
	details, err := rule34Svc.FetchVideoDetails(videoURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to fetch video details", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(details, "Video details retrieved"))
}

// getRule34VideoCategoriesHandler returns top categories from Rule34Video
func getRule34VideoCategoriesHandler(c *gin.Context) {
	rule34Svc := ensureRule34VideoScraperService()
	categories, err := rule34Svc.GetTopCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to fetch categories", err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(categories, "Categories retrieved"))
}

// getRule34VideoModelsHandler returns top models/artists from Rule34Video
func getRule34VideoModelsHandler(c *gin.Context) {
	rule34Svc := ensureRule34VideoScraperService()
	models2, err := rule34Svc.GetTopModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to fetch models", err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(models2, "Models retrieved"))
}

// getRule34VideoFilterOptionsHandler returns all filter/sort options
func getRule34VideoFilterOptionsHandler(c *gin.Context) {
	rule34Svc := ensureRule34VideoScraperService()
	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"sort_options":        rule34Svc.GetSortOptions(),
		"date_filter_options": rule34Svc.GetDateFilterOptions(),
	}, "Filter options retrieved"))
}

// getRule34VideoLocalCategoriesHandler returns distinct categories from locally scraped Rule34Video videos
func getRule34VideoLocalCategoriesHandler(c *gin.Context) {
	db := database.GetDB()
	rows, err := db.Query(`
		SELECT DISTINCT category FROM scraped_videos
		WHERE source = 'rule34video.com' AND category != '' AND category IS NOT NULL
		ORDER BY category ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get local categories", err.Error()))
		return
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var cat string
		if err := rows.Scan(&cat); err == nil && cat != "" {
			categories = append(categories, cat)
		}
	}
	if categories == nil {
		categories = []string{}
	}
	c.JSON(http.StatusOK, models.SuccessResponse(categories, "Local categories retrieved"))
}
