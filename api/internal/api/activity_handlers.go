package api

import (
	"log"
	"net/http"
	"strconv"

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

	// Get updated activity to broadcast via WebSocket
	updatedActivity, _ := svc.GetByID(id)
	if updatedActivity != nil && wsHub != nil {
		wsHub.BroadcastActivityUpdate(updatedActivity)
	}

	// Actually restart the task based on its type
	if activity.TaskType == "forum_scrape" {
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
	}

	// For other task types, just return the checkpoint
	result := map[string]interface{}{
		"checkpoint": checkpoint,
		"message":    "Activity resumed - manual restart may be required",
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result, "Activity resumed successfully"))
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
