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

// Sexuria scraper service singleton
var sexuriaScraperService *services.SexuriaScraperService

// ensureSexuriaScraperService returns the Sexuria scraper service singleton
func ensureSexuriaScraperService() *services.SexuriaScraperService {
	if sexuriaScraperService == nil {
		videoScraperSvc := ensureVideoScraperService()
		sexuriaScraperService = services.NewSexuriaScraperService(database.GetDB(), videoScraperSvc)
	}
	return sexuriaScraperService
}

// SexuriaScrapeRequestBody represents the request body for Sexuria scrape
type SexuriaScrapeRequestBody struct {
	CategoryURL string `json:"categoryUrl" binding:"required"`
	MaxPages    int    `json:"maxPages"`
}

// startSexuriaScrapeHandler starts a Sexuria scrape job by category URL
func startSexuriaScrapeHandler(c *gin.Context) {
	var request SexuriaScrapeRequestBody
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

	sexuriaSvc := ensureSexuriaScraperService()
	scraperTabsSvc := ensureScraperTabsService()
	activitySvc := ensureActivityService()

	// Create a new job to track progress
	job := &services.ScraperJob{
		TabID:         nil,
		JobType:       "sexuria",
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
		activity, activityErr := activitySvc.Create(&models.ActivityLogCreate{
			TaskType: "sexuria_scraping",
			Status:   "running",
			Message:  fmt.Sprintf("Scraping Sexuria category: %s", request.CategoryURL),
			Progress: 0,
			Details: map[string]interface{}{
				"job_id":       createdJob.ID,
				"category_url": request.CategoryURL,
				"max_pages":    request.MaxPages,
			},
		})
		if activityErr != nil {
			log.Printf("❌ Failed to create activity: %v", activityErr)
		} else {
			log.Printf("✓ Created activity with ID: %d", activity.ID)
		}

		// Get activity ID for pause checking
		var activityID int64 = 0
		if activity != nil {
			activityID = int64(activity.ID)
		}

		// Execute scrape with progress callback
		result, err := sexuriaSvc.ScrapeCategoryWithProgress(
			services.SexuriaScrapeRequest{
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
						Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d items from Sexuria (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
						Progress: &progressPercent,
					})

					// Save checkpoint for resume capability (separate from details)
					// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
					activitySvc.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
						"current_page":  progress.CurrentPage, // Actual page number for correct resume
						"videos_found":  progress.VideosFound,
						"videos_saved":  progress.VideosSaved,
						"category_url":  request.CategoryURL,
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
				log.Printf("✓ Sexuria scrape paused successfully, can be resumed")
			} else {
				createdJob.Status = "failed"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				if activity != nil {
					failedStatus := "failed"
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed to scrape Sexuria: %s", err.Error())),
						Completed: true,
					})
				}
			}
		} else {
			createdJob.Status = "completed"
			createdJob.PagesCompleted = result.PagesCrawled
			createdJob.VideosFound = result.TotalScraped
			createdJob.VideosSaved = result.NewItems + result.UpdatedItems
			createdJob.ErrorsCount = len(result.Errors)

			if activity != nil {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d items from Sexuria", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
					Details: map[string]interface{}{
						"pages_crawled": result.PagesCrawled,
						"total_scraped": result.TotalScraped,
						"new_items":     result.NewItems,
						"updated_items": result.UpdatedItems,
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
		fmt.Sprintf("Started Sexuria scrape for: %s", request.CategoryURL),
	))
}

// getSexuriaCategoriesHandler returns available Sexuria categories
func getSexuriaCategoriesHandler(c *gin.Context) {
	sexuriaSvc := ensureSexuriaScraperService()
	categories := sexuriaSvc.GetSexuriaCategories()

	c.JSON(http.StatusOK, models.SuccessResponse(categories, "Categories retrieved successfully"))
}

// getSexuriaSearchOptionsHandler returns all options for advanced search
func getSexuriaSearchOptionsHandler(c *gin.Context) {
	sexuriaSvc := ensureSexuriaScraperService()

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"categories":  sexuriaSvc.GetSexuriaSearchCategories(),
		"fileHosts":   sexuriaSvc.GetSexuriaFileHosts(),
		"resolutions": sexuriaSvc.GetSexuriaResolutions(),
	}, "Search options retrieved successfully"))
}

// SexuriaAdvancedSearchRequestBody represents the request body for advanced search
type SexuriaAdvancedSearchRequestBody struct {
	Categories  []string `json:"categories"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Pornstars   string   `json:"pornstars"`
	Production  string   `json:"production"`
	Tags        string   `json:"tags"`
	Resolutions []string `json:"resolutions"`
	FileHosts   []string `json:"fileHosts"`
	MaxPages    int      `json:"maxPages"`
}

// startSexuriaAdvancedSearchHandler starts an advanced search scrape job
func startSexuriaAdvancedSearchHandler(c *gin.Context) {
	var request SexuriaAdvancedSearchRequestBody
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	sexuriaSvc := ensureSexuriaScraperService()
	scraperTabsSvc := ensureScraperTabsService()
	activitySvc := ensureActivityService()

	// Build the search URL for display
	searchReq := services.SexuriaAdvancedSearchRequest{
		Categories:  request.Categories,
		Title:       request.Title,
		Description: request.Description,
		Pornstars:   request.Pornstars,
		Production:  request.Production,
		Tags:        request.Tags,
		Resolutions: request.Resolutions,
		FileHosts:   request.FileHosts,
		MaxPages:    request.MaxPages,
	}
	searchURL := sexuriaSvc.BuildAdvancedSearchURL(searchReq)

	// Create a new job to track progress
	job := &services.ScraperJob{
		TabID:         nil,
		JobType:       "sexuria_advanced",
		Status:        "pending",
		URL:           searchURL,
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
		activity, activityErr := activitySvc.Create(&models.ActivityLogCreate{
			TaskType: "sexuria_advanced_search",
			Status:   "running",
			Message:  fmt.Sprintf("Advanced search: %s", searchURL),
			Progress: 0,
			Details: map[string]interface{}{
				"job_id":      createdJob.ID,
				"search_url":  searchURL,
				"pornstars":   request.Pornstars,
				"tags":        request.Tags,
				"production":  request.Production,
				"resolutions": request.Resolutions,
				"max_pages":   request.MaxPages,
			},
		})
		if activityErr != nil {
			log.Printf("❌ Failed to create activity for advanced search: %v", activityErr)
		} else {
			log.Printf("✓ Created activity with ID: %d for advanced search", activity.ID)
			// Set activity ID for pause checking
			searchReq.ActivityID = int64(activity.ID)
		}

		// Execute scrape with progress callback
		result, err := sexuriaSvc.ScrapeAdvancedSearchWithProgress(
			searchReq,
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
					// that contain search_url and other parameters needed for resume
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d items (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
						Progress: &progressPercent,
					})

					// Save checkpoint for resume capability (separate from details)
					// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
					activitySvc.SaveCheckpoint(int64(activity.ID), map[string]interface{}{
						"current_page":  progress.CurrentPage, // Actual page number for correct resume
						"videos_found":  progress.VideosFound,
						"videos_saved":  progress.VideosSaved,
						"search_url":    searchURL,
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
				log.Printf("✓ Advanced search paused successfully, can be resumed")
			} else {
				createdJob.Status = "failed"
				createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
				if activity != nil {
					failedStatus := "failed"
					activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			}
		} else {
			createdJob.Status = "completed"
			createdJob.PagesCompleted = result.PagesCrawled
			createdJob.VideosFound = result.TotalScraped
			createdJob.VideosSaved = result.NewItems + result.UpdatedItems
			createdJob.ErrorsCount = len(result.Errors)

			if activity != nil {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(activity.ID, &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d items", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
					Details: map[string]interface{}{
						"pages_crawled": result.PagesCrawled,
						"total_scraped": result.TotalScraped,
						"new_items":     result.NewItems,
						"updated_items": result.UpdatedItems,
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
			"job_id":     createdJob.ID,
			"search_url": searchURL,
		},
		"Started Sexuria advanced search",
	))
}

// getSexuriaItemsHandler returns scraped Sexuria items with pagination
func getSexuriaItemsHandler(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "24"))
	search := c.Query("search")
	host := c.Query("host") // Filter by file host
	sortBy := c.DefaultQuery("sort", "discovered_at")
	sortOrder := c.DefaultQuery("order", "desc")

	videoScraperSvc := ensureVideoScraperService()

	// Use paginated query - Sexuria items are identified by source (match both "sexuria" and "sexuria.net")
	result, err := videoScraperSvc.GetScrapedVideosPaginated(services.ScrapedVideosQuery{
		SourceLike: "sexuria%",
		Keyword:    host, // File hosts are stored as tags
		Search:     search,
		Page:       page,
		Limit:      limit,
		SortBy:     sortBy,
		SortOrder:  sortOrder,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get items",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		map[string]interface{}{
			"items":      result.Videos,
			"count":      len(result.Videos),
			"total":      result.Total,
			"page":       result.Page,
			"limit":      result.Limit,
			"totalPages": result.TotalPages,
		},
		"Sexuria items retrieved successfully",
	))
}

// getSexuriaStatsHandler returns statistics about scraped Sexuria items
func getSexuriaStatsHandler(c *gin.Context) {
	sexuriaSvc := ensureSexuriaScraperService()

	stats, err := sexuriaSvc.GetSexuriaStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Sexuria stats retrieved successfully"))
}

// SexuriaItemDetailsRequest represents the request body for fetching item details
type SexuriaItemDetailsRequest struct {
	URL string `json:"url" binding:"required"`
}

// getSexuriaItemDetailsHandler fetches detailed information for a Sexuria item
func getSexuriaItemDetailsHandler(c *gin.Context) {
	var request SexuriaItemDetailsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if request.URL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"URL is required",
			"",
		))
		return
	}

	sexuriaSvc := ensureSexuriaScraperService()

	details, err := sexuriaSvc.FetchItemDetails(request.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to fetch item details",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(details, "Item details retrieved successfully"))
}
