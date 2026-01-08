package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var schedulerService *services.SchedulerService

// ensureSchedulerService initializes the scheduler service if needed
func ensureSchedulerService() *services.SchedulerService {
	if schedulerService == nil {
		dbInstance := database.GetDB()
		if dbInstance != nil {
			scraperSvc := ensureScraperService()
			activitySvc := ensureActivityService()
			if scraperSvc != nil && activitySvc != nil {
				schedulerService = services.NewSchedulerService(dbInstance, scraperSvc, activitySvc)
				// Auto-start scheduler
				if err := schedulerService.Start(); err != nil {
					log.Printf("Failed to start scheduler: %v\n", err)
				}
			}
		}
	}
	return schedulerService
}

// getSchedulerStatus returns the scheduler status
func getSchedulerStatus(c *gin.Context) {
	svc := ensureSchedulerService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Scheduler service not initialized",
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"running": true,
		"message": "Scheduler is running",
	}, "Scheduler status retrieved"))
}

// getAllJobs returns all scheduled jobs
func getAllJobs(c *gin.Context) {
	svc := ensureSchedulerService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Scheduler service not initialized",
		))
		return
	}

	jobs, err := svc.GetAllJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get jobs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"jobs":  jobs,
		"count": len(jobs),
	}, "Jobs retrieved successfully"))
}

// getJob returns a single job by ID
func getJob(c *gin.Context) {
	svc := ensureSchedulerService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Scheduler service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid job ID",
			err.Error(),
		))
		return
	}

	job, err := svc.GetJobByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Job not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(job, "Job retrieved successfully"))
}

// createJob creates a new scheduled job
func createJob(c *gin.Context) {
	svc := ensureSchedulerService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Scheduler service not initialized",
		))
		return
	}

	var req struct {
		JobType        string                 `json:"job_type" binding:"required"`
		ScheduleType   string                 `json:"schedule_type" binding:"required"`
		ScheduleConfig models.ScheduleConfig  `json:"schedule_config"`
		TargetType     string                 `json:"target_type"`
		TargetID       *int64                 `json:"target_id"`
		Enabled        bool                   `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Validate job type
	validJobTypes := []string{"scrape_thread", "verify_links", "cleanup_old_activities", "cleanup_old_audit_logs"}
	isValid := false
	for _, validType := range validJobTypes {
		if req.JobType == validType {
			isValid = true
			break
		}
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid job type",
			"Job type must be one of: scrape_thread, verify_links, cleanup_old_activities, cleanup_old_audit_logs",
		))
		return
	}

	// Validate schedule type
	validScheduleTypes := []string{"interval", "cron", "once"}
	isValid = false
	for _, validType := range validScheduleTypes {
		if req.ScheduleType == validType {
			isValid = true
			break
		}
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid schedule type",
			"Schedule type must be one of: interval, cron, once",
		))
		return
	}

	// Calculate next run time
	var nextRunAt *time.Time
	now := time.Now()
	switch req.ScheduleType {
	case "interval":
		if req.ScheduleConfig.IntervalMinutes > 0 {
			next := now.Add(time.Duration(req.ScheduleConfig.IntervalMinutes) * time.Minute)
			nextRunAt = &next
		}
	case "once":
		if req.ScheduleConfig.RunAt != nil {
			nextRunAt = req.ScheduleConfig.RunAt
		} else {
			// Default to immediate execution
			nextRunAt = &now
		}
	}

	// Serialize config
	configJSON, err := json.Marshal(req.ScheduleConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to serialize config",
			err.Error(),
		))
		return
	}

	job := &models.ScheduledJob{
		JobType:        req.JobType,
		ScheduleType:   req.ScheduleType,
		ScheduleConfig: string(configJSON),
		TargetType:     req.TargetType,
		TargetID:       req.TargetID,
		Enabled:        req.Enabled,
		NextRunAt:      nextRunAt,
	}

	createdJob, err := svc.CreateJob(job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create job",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(createdJob, "Job created successfully"))
}

// updateJob updates a scheduled job
func updateJob(c *gin.Context) {
	svc := ensureSchedulerService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Scheduler service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid job ID",
			err.Error(),
		))
		return
	}

	// Get existing job
	job, err := svc.GetJobByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Job not found",
			err.Error(),
		))
		return
	}

	var req struct {
		JobType        string                 `json:"job_type"`
		ScheduleType   string                 `json:"schedule_type"`
		ScheduleConfig *models.ScheduleConfig `json:"schedule_config"`
		TargetType     string                 `json:"target_type"`
		TargetID       *int64                 `json:"target_id"`
		Enabled        *bool                  `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Update fields if provided
	if req.JobType != "" {
		job.JobType = req.JobType
	}
	if req.ScheduleType != "" {
		job.ScheduleType = req.ScheduleType
	}
	if req.ScheduleConfig != nil {
		configJSON, err := json.Marshal(req.ScheduleConfig)
		if err == nil {
			job.ScheduleConfig = string(configJSON)
		}
	}
	if req.TargetType != "" {
		job.TargetType = req.TargetType
	}
	if req.TargetID != nil {
		job.TargetID = req.TargetID
	}
	if req.Enabled != nil {
		job.Enabled = *req.Enabled
	}

	if err := svc.UpdateJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update job",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(job, "Job updated successfully"))
}

// deleteJob deletes a scheduled job
func deleteJob(c *gin.Context) {
	svc := ensureSchedulerService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Scheduler service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid job ID",
			err.Error(),
		))
		return
	}

	if err := svc.DeleteJob(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete job",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Job deleted successfully"))
}

// toggleJob toggles a job's enabled status
func toggleJob(c *gin.Context) {
	svc := ensureSchedulerService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Scheduler service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid job ID",
			err.Error(),
		))
		return
	}

	job, err := svc.GetJobByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Job not found",
			err.Error(),
		))
		return
	}

	job.Enabled = !job.Enabled
	if err := svc.UpdateJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to toggle job",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(job, "Job toggled successfully"))
}

// getJobExecutionHistory returns execution history for a job
func getJobExecutionHistory(c *gin.Context) {
	svc := ensureSchedulerService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Scheduler service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid job ID",
			err.Error(),
		))
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 200 {
		limit = 50
	}

	history, err := svc.GetExecutionHistory(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get execution history",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"history": history,
		"count":   len(history),
	}, "Execution history retrieved successfully"))
}
