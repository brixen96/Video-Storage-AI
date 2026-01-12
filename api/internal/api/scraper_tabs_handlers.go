package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var scraperTabsService *services.ScraperTabsService

// ensureScraperTabsService initializes the service if needed
func ensureScraperTabsService() *services.ScraperTabsService {
	if scraperTabsService == nil {
		scraperTabsService = services.NewScraperTabsService(database.GetDB())
	}
	return scraperTabsService
}

// getScraperTabsHandler retrieves all scraper tabs
func getScraperTabsHandler(c *gin.Context) {
	svc := ensureScraperTabsService()

	tabs, err := svc.GetScraperTabs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve scraper tabs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(tabs, "Scraper tabs retrieved successfully"))
}

// getScraperTabHandler retrieves a single scraper tab by ID
func getScraperTabHandler(c *gin.Context) {
	svc := ensureScraperTabsService()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Tab ID is required",
			"",
		))
		return
	}

	var tabID int64
	if _, err := fmt.Sscanf(id, "%d", &tabID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid tab ID",
			err.Error(),
		))
		return
	}

	tab, err := svc.GetScraperTabByID(tabID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Tab not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(tab, "Scraper tab retrieved successfully"))
}

// createScraperTabHandler creates a new scraper tab
func createScraperTabHandler(c *gin.Context) {
	svc := ensureScraperTabsService()

	var tab services.ScraperTab
	if err := c.ShouldBindJSON(&tab); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Validate required fields
	if tab.Name == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Tab name is required",
			"",
		))
		return
	}

	if tab.BaseURL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Base URL is required",
			"",
		))
		return
	}

	// Set defaults
	if tab.TabType == "" {
		tab.TabType = "video"
	}
	tab.IsActive = true // Always active when created

	createdTab, err := svc.CreateScraperTab(&tab)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create scraper tab",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(createdTab, "Scraper tab created successfully"))
}

// updateScraperTabHandler updates an existing scraper tab
func updateScraperTabHandler(c *gin.Context) {
	svc := ensureScraperTabsService()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Tab ID is required",
			"",
		))
		return
	}

	var tabID int64
	if _, err := fmt.Sscanf(id, "%d", &tabID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid tab ID",
			err.Error(),
		))
		return
	}

	var tab services.ScraperTab
	if err := c.ShouldBindJSON(&tab); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	updatedTab, err := svc.UpdateScraperTab(tabID, &tab)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update scraper tab",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(updatedTab, "Scraper tab updated successfully"))
}

// deleteScraperTabHandler deletes a scraper tab
func deleteScraperTabHandler(c *gin.Context) {
	svc := ensureScraperTabsService()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Tab ID is required",
			"",
		))
		return
	}

	var tabID int64
	if _, err := fmt.Sscanf(id, "%d", &tabID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid tab ID",
			err.Error(),
		))
		return
	}

	if err := svc.DeleteScraperTab(tabID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete scraper tab",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Scraper tab deleted successfully"))
}

// getScraperJobsHandler retrieves all jobs for a specific tab
func getScraperJobsHandler(c *gin.Context) {
	svc := ensureScraperTabsService()

	tabID := c.Query("tabId")
	if tabID == "" {
		// Get all active jobs
		jobs, err := svc.GetActiveScraperJobs()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to retrieve scraper jobs",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusOK, models.SuccessResponse(jobs, "Scraper jobs retrieved successfully"))
		return
	}

	var id int64
	if _, err := fmt.Sscanf(tabID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid tab ID",
			err.Error(),
		))
		return
	}

	jobs, err := svc.GetScraperJobsByTabID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve scraper jobs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(jobs, "Scraper jobs retrieved successfully"))
}

// getScraperJobHandler retrieves a single job by ID
func getScraperJobHandler(c *gin.Context) {
	svc := ensureScraperTabsService()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Job ID is required",
			"",
		))
		return
	}

	var jobID int64
	if _, err := fmt.Sscanf(id, "%d", &jobID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid job ID",
			err.Error(),
		))
		return
	}

	job, err := svc.GetScraperJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Job not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(job, "Scraper job retrieved successfully"))
}

// startScraperJobHandler starts a new scraping job for a tab
func startScraperJobHandler(c *gin.Context) {
	svc := ensureScraperTabsService()
	videoScraperSvc := ensureVideoScraperService()
	consoleLogSvc := ensureConsoleLogService()

	var request struct {
		TabID int64  `json:"tabId"`
		URL   string `json:"url"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Get the tab configuration
	tab, err := svc.GetScraperTabByID(request.TabID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Tab not found",
			err.Error(),
		))
		return
	}

	// Create a new job
	job := &services.ScraperJob{
		TabID:          request.TabID,
		Status:         "pending",
		URL:            request.URL,
		ErrorMessages:  []string{},
	}

	createdJob, err := svc.CreateScraperJob(job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create scraper job",
			err.Error(),
		))
		return
	}

	// Start the scraping in a goroutine
	go func() {
		// Update job status to running
		now := time.Now()
		createdJob.Status = "running"
		createdJob.StartedAt = &now
		svc.UpdateScraperJob(createdJob.ID, createdJob)

		consoleLogSvc.LogAPI("info", "Scraper job started", map[string]interface{}{
			"job_id": createdJob.ID,
			"tab_id": request.TabID,
			"url":    request.URL,
		})

		// Build scrape request from tab configuration
		scrapeRequest := services.ScrapeVideoRequest{
			URL:      request.URL,
			SiteType: tab.SiteType,
			Options: services.ScrapeOptions{
				DownloadVideo:    false,
				ExtractMetadata:  true,
				ScrapeThumbnails: true,
				MaxPages:         0, // Unlimited
			},
		}

		// Override with tab-specific options if available
		if opts, ok := tab.ScrapeOptions["downloadVideo"].(bool); ok {
			scrapeRequest.Options.DownloadVideo = opts
		}
		if opts, ok := tab.ScrapeOptions["extractMetadata"].(bool); ok {
			scrapeRequest.Options.ExtractMetadata = opts
		}
		if opts, ok := tab.ScrapeOptions["scrapeThumbnails"].(bool); ok {
			scrapeRequest.Options.ScrapeThumbnails = opts
		}

		// Execute the scrape with progress callback
		result, err := videoScraperSvc.ScrapePageWithProgress(scrapeRequest, func(progress services.ScrapeProgress) {
			// Update job progress in real-time
			createdJob.PagesCompleted = progress.PagesCompleted
			createdJob.VideosFound = progress.VideosFound
			createdJob.VideosSaved = progress.VideosSaved
			createdJob.ErrorsCount = len(progress.Errors)
			if len(progress.Errors) > 0 {
				createdJob.ErrorMessages = progress.Errors
			}
			svc.UpdateScraperJob(createdJob.ID, createdJob)
		})

		completedAt := time.Now()
		createdJob.CompletedAt = &completedAt

		if err != nil {
			createdJob.Status = "failed"
			createdJob.ErrorMessages = append(createdJob.ErrorMessages, err.Error())
			createdJob.ErrorsCount++

			consoleLogSvc.LogAPI("error", "Scraper job failed", map[string]interface{}{
				"job_id": createdJob.ID,
				"error":  err.Error(),
			})
		} else {
			createdJob.Status = "completed"
			createdJob.TotalPages = result.PagesCrawled
			createdJob.PagesCompleted = result.PagesCrawled
			createdJob.VideosFound = result.TotalScraped
			createdJob.VideosSaved = result.TotalScraped

			if len(result.Errors) > 0 {
				createdJob.ErrorMessages = result.Errors
				createdJob.ErrorsCount = len(result.Errors)
			}

			consoleLogSvc.LogAPI("info", "Scraper job completed", map[string]interface{}{
				"job_id":        createdJob.ID,
				"pages_crawled": result.PagesCrawled,
				"videos_found":  result.TotalScraped,
			})
		}

		// Update final job status
		svc.UpdateScraperJob(createdJob.ID, createdJob)
	}()

	c.JSON(http.StatusOK, models.SuccessResponse(createdJob, "Scraper job started successfully"))
}

// cancelScraperJobHandler cancels a running job
func cancelScraperJobHandler(c *gin.Context) {
	svc := ensureScraperTabsService()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Job ID is required",
			"",
		))
		return
	}

	var jobID int64
	if _, err := fmt.Sscanf(id, "%d", &jobID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid job ID",
			err.Error(),
		))
		return
	}

	job, err := svc.GetScraperJobByID(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Job not found",
			err.Error(),
		))
		return
	}

	// Update job status to cancelled
	now := time.Now()
	job.Status = "cancelled"
	job.CompletedAt = &now

	if err := svc.UpdateScraperJob(jobID, job); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to cancel job",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(job, "Job cancelled successfully"))
}
