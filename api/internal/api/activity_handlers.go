package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var activityService *services.ActivityService

// ensureActivityService initializes the service if needed
func ensureActivityService() *services.ActivityService {
	if activityService == nil {
		activityService = services.NewActivityService()
	}
	return activityService
}

// getActivities retrieves all activity logs with optional filtering
func getActivities(c *gin.Context) {
	svc := ensureActivityService()

	status := c.Query("status")
	taskType := c.Query("task_type")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	activities, err := svc.GetAll(status, taskType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve activities",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(activities, "Activities retrieved successfully"))
}

// getActivity retrieves a single activity log by ID
func getActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid activity ID",
			err.Error(),
		))
		return
	}

	activity, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Activity not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(activity, "Activity retrieved successfully"))
}

// createActivity creates a new activity log
func createActivity(c *gin.Context) {
	svc := ensureActivityService()
	var create models.ActivityLogCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	activity, err := svc.Create(&create)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create activity",
			err.Error(),
		))
		return
	}

	if err := svc.BroadcastUpdate(activity); err != nil {
		log.Printf("Error broadcasting activity update: %v", err)
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(activity, "Activity created successfully"))
}

// updateActivity updates an existing activity log
func updateActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid activity ID",
			err.Error(),
		))
		return
	}

	var update models.ActivityLogUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	activity, err := svc.Update(int(id), &update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update activity",
			err.Error(),
		))
		return
	}

	if err := svc.BroadcastUpdate(activity); err != nil {
		log.Printf("Error broadcasting activity update: %v", err)
	}

	c.JSON(http.StatusOK, models.SuccessResponse(activity, "Activity updated successfully"))
}

// deleteActivity deletes an activity log
func deleteActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid activity ID",
			err.Error(),
		))
		return
	}

	err = svc.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete activity",
			err.Error(),
		))
		return
	}

	if err := svc.BroadcastStatusUpdate(); err != nil {
		log.Printf("Error broadcasting status update: %v", err)
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Activity deleted successfully"))
}

// getActivityStatus returns current activity status summary
func getActivityStatus(c *gin.Context) {
	svc := ensureActivityService()

	status, err := svc.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve activity status",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(status, "Status retrieved successfully"))
}

// getActivityStats returns statistics grouped by task type
func getActivityStats(c *gin.Context) {
	svc := ensureActivityService()

	stats, err := svc.GetStatsByType()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve activity stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Stats retrieved successfully"))
}

// cleanOldActivities removes old completed/failed activities
func cleanOldActivities(c *gin.Context) {
	svc := ensureActivityService()
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		days = 30
	}

	count, err := svc.CleanOld(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to clean old activities",
			err.Error(),
		))
		return
	}

	if err := svc.BroadcastStatusUpdate(); err != nil {
		log.Printf("Error broadcasting status update: %v", err)
	}

	result := map[string]interface{}{
		"deleted_count": count,
		"days_old":      days,
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result, "Old activities cleaned successfully"))
}

// clearAllActivities removes all activity logs
func clearAllActivities(c *gin.Context) {
	svc := ensureActivityService()

	count, err := svc.ClearAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to clear all activities",
			err.Error(),
		))
		return
	}

	if err := svc.BroadcastStatusUpdate(); err != nil {
		log.Printf("Error broadcasting status update: %v", err)
	}

	result := map[string]interface{}{
		"deleted_count": count,
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result, "All activities cleared successfully"))
}

// getRecentActivities retrieves the most recent activities
func getRecentActivities(c *gin.Context) {
	svc := ensureActivityService()
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 20
	}

	activities, err := svc.GetRecent(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve recent activities",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(activities, "Recent activities retrieved successfully"))
}

// pauseActivity pauses a running task
func pauseActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid activity ID",
			err.Error(),
		))
		return
	}

	// Get checkpoint data from request body (optional)
	var req struct {
		Checkpoint map[string]interface{} `json:"checkpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body provided, use empty checkpoint
		req.Checkpoint = make(map[string]interface{})
	}

	err = svc.PauseTask(id, req.Checkpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to pause activity",
			err.Error(),
		))
		return
	}

	// Get updated activity to return
	activity, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Activity paused successfully"))
		return
	}

	// Broadcast the paused activity via WebSocket for instant UI update
	if wsHub != nil {
		wsHub.BroadcastActivityUpdate(activity)
	}

	c.JSON(http.StatusOK, models.SuccessResponse(activity, "Activity paused successfully"))
}

// cancelActivity cancels a running or paused task permanently
func cancelActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid activity ID",
			err.Error(),
		))
		return
	}

	err = svc.CancelTask(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to cancel activity",
			err.Error(),
		))
		return
	}

	// Get updated activity to return
	activity, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Activity cancelled successfully"))
		return
	}

	// Broadcast the cancelled activity via WebSocket for instant UI update
	if wsHub != nil {
		wsHub.BroadcastActivityUpdate(activity)
	}

	c.JSON(http.StatusOK, models.SuccessResponse(activity, "Activity cancelled successfully"))
}

// retryActivity retries a failed or cancelled task from the beginning
func retryActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid activity ID",
			err.Error(),
		))
		return
	}

	// Reset the task for retry and get updated activity with details
	activity, err := svc.RetryTask(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retry activity",
			err.Error(),
		))
		return
	}

	// Parse activity details to restart the appropriate scraper
	if err := activity.UnmarshalDetails(); err != nil {
		log.Printf("Warning: failed to parse activity details: %v", err)
	}

	// Based on task type, restart the appropriate scraper
	// The actual restart logic is similar to resumeActivity but starts from the beginning
	switch activity.TaskType {
	case "pornhub_scraping":
		keyword, ok := activity.DetailsObj["keyword"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to retry activity",
				"Keyword not found in activity details",
			))
			return
		}

		pornHubSvc := ensurePornHubScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.PornHubSearchRequest{
				Keyword:    keyword,
				StartPage:  1, // Start from beginning
				ActivityID: id,
			}
			result, err := pornHubSvc.ScrapeByKeywordWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s' (page %d)", progress.PagesCompleted, progress.VideosFound, keyword, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page":  progress.CurrentPage,
					"videos_found":  progress.VideosFound,
					"videos_saved":  progress.VideosSaved,
					"keyword":       keyword,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s'", result.PagesCrawled, result.TotalScraped, keyword)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(activity, "PornHub scraping restarted"))
		return

	case "luxuretv_scraping":
		categoryURL, ok := activity.DetailsObj["category_url"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to retry activity",
				"Category URL not found in activity details",
			))
			return
		}

		luxureTVSvc := ensureLuxureTVScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.LuxureTVScrapeRequest{
				CategoryURL: categoryURL,
				StartPage:   1, // Start from beginning
				ActivityID:  id,
			}
			result, err := luxureTVSvc.ScrapeCategoryWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page":  progress.CurrentPage,
					"videos_found":  progress.VideosFound,
					"videos_saved":  progress.VideosSaved,
					"category_url":  categoryURL,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(activity, "LuxureTV scraping restarted"))
		return

	case "sexuria_scraping", "sexuria_advanced_search":
		categoryURL, _ := activity.DetailsObj["category_url"].(string)
		sexuriaSvc := ensureSexuriaScraperService()
		activitySvc := ensureActivityService()

		go func() {
			req := services.SexuriaScrapeRequest{
				CategoryURL: categoryURL,
				StartPage:   1, // Start from beginning
				ActivityID:  id,
			}
			result, err := sexuriaSvc.ScrapeCategoryWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d items from Sexuria (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page":  progress.CurrentPage,
					"videos_found":  progress.VideosFound,
					"videos_saved":  progress.VideosSaved,
					"category_url":  categoryURL,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d items from Sexuria", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(activity, "Sexuria scraping restarted"))
		return

	default:
		// For unknown task types, just return success with the reset activity
		c.JSON(http.StatusOK, models.SuccessResponse(activity, "Activity reset for retry - manual restart may be required"))
	}
}

// resumeActivity resumes a paused task
func resumeActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid activity ID",
			err.Error(),
		))
		return
	}

	// Get the activity to check its type and details
	activity, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get activity",
			err.Error(),
		))
		return
	}

	// Unmarshal details to get task-specific data
	if err := activity.UnmarshalDetails(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to parse activity details",
			err.Error(),
		))
		return
	}

	// Resume the task (clears pause flag)
	checkpoint, err := svc.ResumeTask(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to resume activity",
			err.Error(),
		))
		return
	}

	// Debug: Log checkpoint data
	log.Printf("📋 Resume task %d - checkpoint data: %+v", id, checkpoint)

	// Get updated activity to broadcast via WebSocket
	updatedActivity, _ := svc.GetByID(id)
	if updatedActivity != nil && wsHub != nil {
		wsHub.BroadcastActivityUpdate(updatedActivity)
	}

	// Actually restart the task based on its type
	switch activity.TaskType {
	case "forum_scrape":
		// Get forum URL from details
		forumURL, ok := activity.DetailsObj["forum_url"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to resume activity",
				"Forum URL not found in activity details",
			))
			return
		}

		// Restart the forum scraper in the background with existing activity ID
		scraperSvc := ensureScraperService()
		go func() {
			if err := scraperSvc.ResumeForumScrape(forumURL, int(id)); err != nil {
				log.Printf("Error resuming forum scrape: %v", err)
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Forum scraping resumed in background"))
		return

	case "sexuria_scraping":
		// Get category URL from details
		categoryURL, ok := activity.DetailsObj["category_url"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to resume activity",
				"Category URL not found in activity details",
			))
			return
		}

		// Get checkpoint page number and counts
		startPage := 1
		initialVideoCount := 0
		initialSavedCount := 0
		if checkpoint != nil {
			if page, ok := checkpoint["current_page"].(float64); ok {
				startPage = int(page) + 1 // Resume from the NEXT page after the last completed one
			}
			if found, ok := checkpoint["videos_found"].(float64); ok {
				initialVideoCount = int(found)
			}
			if saved, ok := checkpoint["videos_saved"].(float64); ok {
				initialSavedCount = int(saved)
			}
		}
		log.Printf("📍 sexuria_scraping: Resuming from page %d with %d videos found (checkpoint: %+v)", startPage, initialVideoCount, checkpoint)

		// Restart the Sexuria scraper in background with activity ID for pause support
		sexuriaSvc := ensureSexuriaScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.SexuriaScrapeRequest{
				CategoryURL:       categoryURL,
				StartPage:         startPage,
				ActivityID:        id, // Pass activity ID so it can be paused again
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
			}
			result, err := sexuriaSvc.ScrapeCategoryWithProgress(req, func(progress services.ScrapeProgress) {
				// Update activity progress
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d items from Sexuria (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				// Save checkpoint for resume capability
				// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page":  progress.CurrentPage, // Actual page number for correct resume
					"videos_found":  progress.VideosFound,
					"videos_saved":  progress.VideosSaved,
					"category_url":  categoryURL,
				})
			})

			// Update final status
			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d items", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Sexuria scraping resumed in background"))
		return

	case "sexuria_advanced_search":
		// Get search URL from details or checkpoint
		searchURL, ok := activity.DetailsObj["search_url"].(string)
		if !ok {
			// Try checkpoint
			if checkpoint != nil {
				searchURL, ok = checkpoint["search_url"].(string)
			}
			if !ok {
				c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
					"Failed to resume activity",
					"Search URL not found in activity details",
				))
				return
			}
		}

		// Get checkpoint page number and counts
		startPage := 1
		initialVideoCount := 0
		initialSavedCount := 0
		if checkpoint != nil {
			if page, ok := checkpoint["current_page"].(float64); ok {
				startPage = int(page) + 1 // Resume from the NEXT page after the last completed one
			}
			if found, ok := checkpoint["videos_found"].(float64); ok {
				initialVideoCount = int(found)
			}
			if saved, ok := checkpoint["videos_saved"].(float64); ok {
				initialSavedCount = int(saved)
			}
		}
		log.Printf("📍 sexuria_advanced_search: Resuming from page %d with %d videos found (checkpoint: %+v)", startPage, initialVideoCount, checkpoint)

		// Restart the Sexuria advanced search in background
		sexuriaSvc := ensureSexuriaScraperService()
		activitySvc := ensureActivityService()
		go func() {
			// Build request from stored URL - parse categories, resolutions, etc from details
			req := services.SexuriaAdvancedSearchRequest{
				StartPage:         startPage,
				ActivityID:        id, // Pass activity ID so it can be paused again
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
			}

			// Restore search parameters from details
			if cats, ok := activity.DetailsObj["categories"].([]interface{}); ok {
				for _, c := range cats {
					if s, ok := c.(string); ok {
						req.Categories = append(req.Categories, s)
					}
				}
			}
			if ps, ok := activity.DetailsObj["pornstars"].(string); ok {
				req.Pornstars = ps
			}
			if tags, ok := activity.DetailsObj["tags"].(string); ok {
				req.Tags = tags
			}
			if prod, ok := activity.DetailsObj["production"].(string); ok {
				req.Production = prod
			}
			if res, ok := activity.DetailsObj["resolutions"].([]interface{}); ok {
				for _, r := range res {
					if s, ok := r.(string); ok {
						req.Resolutions = append(req.Resolutions, s)
					}
				}
			}
			if maxPages, ok := activity.DetailsObj["max_pages"].(float64); ok {
				req.MaxPages = int(maxPages)
			}

			result, err := sexuriaSvc.ScrapeAdvancedSearchWithProgress(req, func(progress services.ScrapeProgress) {
				// Update activity progress
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d items (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				// Save checkpoint for resume capability
				// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page":  progress.CurrentPage, // Actual page number for correct resume
					"videos_found":  progress.VideosFound,
					"videos_saved":  progress.VideosSaved,
					"search_url":    searchURL,
				})
			})

			// Update final status
			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d items", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Sexuria advanced search resumed in background"))
		return

	case "pornhub_scraping":
		// Get keyword from details
		keyword, ok := activity.DetailsObj["keyword"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to resume activity",
				"Keyword not found in activity details",
			))
			return
		}

		// Get checkpoint page number and counts
		startPage := 1
		initialVideoCount := 0
		initialSavedCount := 0
		if checkpoint != nil {
			if page, ok := checkpoint["current_page"].(float64); ok {
				startPage = int(page) + 1 // Resume from the NEXT page after the last completed one
			}
			if found, ok := checkpoint["videos_found"].(float64); ok {
				initialVideoCount = int(found)
			}
			if saved, ok := checkpoint["videos_saved"].(float64); ok {
				initialSavedCount = int(saved)
			}
		}

		// Restart the PornHub scraper in background with activity ID for pause support
		pornhubSvc := ensurePornHubScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.PornHubSearchRequest{
				Keyword:           keyword,
				StartPage:         startPage,
				ActivityID:        id, // Pass activity ID so it can be paused again
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
			}
			result, err := pornhubSvc.ScrapeByKeywordWithProgress(req, func(progress services.ScrapeProgress) {
				// Update activity progress
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s' (page %d)", progress.PagesCompleted, progress.VideosFound, keyword, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				// Save checkpoint for resume capability
				// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page":  progress.CurrentPage, // Actual page number for correct resume
					"videos_found":  progress.VideosFound,
					"videos_saved":  progress.VideosSaved,
					"keyword":       keyword,
				})
			})

			// Update final status
			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s'", result.PagesCrawled, result.TotalScraped, keyword)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "PornHub scraping resumed in background"))
		return

	case "luxuretv_scraping":
		// Get category URL from details
		categoryURL, ok := activity.DetailsObj["category_url"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to resume activity",
				"Category URL not found in activity details",
			))
			return
		}

		// Get checkpoint page number and video counts
		startPage := 1
		initialVideoCount := 0
		initialSavedCount := 0
		if checkpoint != nil {
			if page, ok := checkpoint["current_page"].(float64); ok {
				startPage = int(page) + 1 // Resume from the NEXT page after the last completed one
			}
			if found, ok := checkpoint["videos_found"].(float64); ok {
				initialVideoCount = int(found)
			}
			if saved, ok := checkpoint["videos_saved"].(float64); ok {
				initialSavedCount = int(saved)
			}
		}

		// Restart the LuxureTV scraper in background with activity ID for pause support
		luxureTVSvc := ensureLuxureTVScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.LuxureTVScrapeRequest{
				CategoryURL:       categoryURL,
				StartPage:         startPage,
				ActivityID:        id, // Pass activity ID so it can be paused again
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
			}
			result, err := luxureTVSvc.ScrapeCategoryWithProgress(req, func(progress services.ScrapeProgress) {
				// Update activity progress
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				// Save checkpoint for resume capability
				// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page":   progress.CurrentPage, // Actual page number for correct resume
					"videos_found":   progress.VideosFound,
					"videos_saved":   progress.VideosSaved,
					"category_url":   categoryURL,
				})
			})

			// Update final status
			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "LuxureTV scraping resumed in background"))
		return

	case "scraper_thread":
		// Get thread URL from details
		threadURL, ok := activity.DetailsObj["url"].(string)
		if !ok {
			// Try checkpoint
			if checkpoint != nil {
				threadURL, ok = checkpoint["url"].(string)
			}
			if !ok {
				c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
					"Failed to resume activity",
					"Thread URL not found in activity details",
				))
				return
			}
		}

		// Restart the scraper thread in background with existing activity ID
		scraperSvc := ensureScraperService()
		go func() {
			if err := scraperSvc.ScrapeThreadComplete(threadURL, int(id)); err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					log.Printf("Error resuming scraper thread: %v", err)
					failedStatus := "failed"
					activitySvc := ensureActivityService()
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Scraper thread resumed in background"))
		return

	case "scraping":
		// Get URL from details
		url, ok := activity.DetailsObj["url"].(string)
		if !ok {
			// Try checkpoint
			if checkpoint != nil {
				url, ok = checkpoint["url"].(string)
			}
			if !ok {
				c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
					"Failed to resume activity",
					"URL not found in activity details",
				))
				return
			}
		}

		// Get tab_id from details (optional)
		var tabID int64 = 0
		if tid, ok := activity.DetailsObj["tab_id"].(float64); ok {
			tabID = int64(tid)
		}

		// Get checkpoint page number and video counts
		startPage := 1
		initialVideoCount := 0
		initialSavedCount := 0
		if checkpoint != nil {
			if page, ok := checkpoint["current_page"].(float64); ok {
				startPage = int(page) + 1 // Resume from the NEXT page after the last completed one
			}
			if found, ok := checkpoint["videos_found"].(float64); ok {
				initialVideoCount = int(found)
			}
			if saved, ok := checkpoint["videos_saved"].(float64); ok {
				initialSavedCount = int(saved)
			}
		}

		// Restart the generic scraper in background with activity ID for pause support
		videoScraperSvc := ensureVideoScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.ScrapeVideoRequest{
				URL:               url,
				StartPage:         startPage,
				ActivityID:        id, // Pass activity ID so it can be paused again
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
				Options: services.ScrapeOptions{
					DownloadVideo:    false,
					ExtractMetadata:  true,
					ScrapeThumbnails: true,
					MaxPages:         0, // Unlimited
				},
			}
			result, err := videoScraperSvc.ScrapePageWithProgress(req, func(progress services.ScrapeProgress) {
				// Update activity progress
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				// Save checkpoint for resume capability
				// Use CurrentPage (actual page number) NOT PagesCompleted (counter)
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page":  progress.CurrentPage, // Actual page number for correct resume
					"videos_found":  progress.VideosFound,
					"videos_saved":  progress.VideosSaved,
					"url":           url,
					"tab_id":        tabID,
				})
			})

			// Update final status
			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Scraping resumed in background"))
		return

	default:
		// For other task types, just return the checkpoint
		result := map[string]interface{}{
			"checkpoint": checkpoint,
			"message":    "Activity resumed - manual restart may be required",
		}

		c.JSON(http.StatusOK, models.SuccessResponse(result, "Activity resumed successfully"))
	}
}

// getPausedActivities retrieves all currently paused tasks
func getPausedActivities(c *gin.Context) {
	svc := ensureActivityService()

	// Get all running tasks that are paused
	activities, err := svc.GetAll("running", "", 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve paused activities",
			err.Error(),
		))
		return
	}

	// Filter for paused tasks
	pausedActivities := make([]*models.Activity, 0)
	for i := range activities {
		if activities[i].IsPaused {
			pausedActivities = append(pausedActivities, &activities[i])
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(pausedActivities, "Paused activities retrieved successfully"))
}
