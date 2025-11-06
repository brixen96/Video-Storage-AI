package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var performerService *services.PerformerService

// ensurePerformerService initializes the service if needed
func ensurePerformerService() *services.PerformerService {
	if performerService == nil {
		performerService = services.NewPerformerService()
	}
	return performerService
}

// getPerformers retrieves all performers
func getPerformers(c *gin.Context) {
	svc := ensurePerformerService()
	searchTerm := c.Query("search")
	
	// Pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	
	// Ensure valid pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	
	offset := (page - 1) * limit

	var performers []models.Performer
	var total int64
	var err error

	if searchTerm != "" {
		performers, total, err = svc.SearchPaginated(searchTerm, limit, offset)
	} else {
		performers, total, err = svc.GetAllPaginated(limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve performers",
			err.Error(),
		))
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Performers retrieved successfully",
		"data":    performers,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// getPerformer retrieves a single performer by ID
func getPerformer(c *gin.Context) {
	svc := ensurePerformerService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	performer, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Performer not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(performer, "Performer retrieved successfully"))
}

// createPerformer creates a new performer
func createPerformer(c *gin.Context) {
	svc := ensurePerformerService()
	var create models.PerformerCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	performer, err := svc.Create(&create)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create performer",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(performer, "Performer created successfully"))
}

// updatePerformer updates an existing performer
func updatePerformer(c *gin.Context) {
	svc := ensurePerformerService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	var update models.PerformerUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	performer, err := svc.Update(id, &update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update performer",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(performer, "Performer updated successfully"))
}

// deletePerformer deletes a performer
func deletePerformer(c *gin.Context) {
	svc := ensurePerformerService()
	activitySvc := services.NewActivityService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	// Get performer name before deleting
	performer, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Performer not found",
			err.Error(),
		))
		return
	}

	// Create activity log
	activity, err := activitySvc.StartTask(
		"delete_performer",
		fmt.Sprintf("Deleting performer: %s", performer.Name),
		map[string]interface{}{
			"performer_id":   performer.ID,
			"performer_name": performer.Name,
		},
	)
	if err != nil {
		fmt.Printf("Failed to create activity log: %v\n", err)
	}

	err = svc.Delete(id)
	if err != nil {
		if activity != nil {
			activitySvc.FailTask(activity.ID, fmt.Sprintf("Delete failed: %v", err))
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete performer",
			err.Error(),
		))
		return
	}

	// Mark task as completed
	if activity != nil {
		activitySvc.CompleteTask(int64(activity.ID), fmt.Sprintf("Successfully deleted %s", performer.Name))
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Performer deleted successfully"))
}

// fetchMetadata fetches metadata from AdultDataLink API
func fetchMetadata(c *gin.Context) {
	svc := ensurePerformerService()
	activitySvc := services.NewActivityService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	// Get the performer
	performer, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Performer not found",
			err.Error(),
		))
		return
	}

	// Create activity log
	activity, err := activitySvc.StartTask(
		"fetch_metadata",
		fmt.Sprintf("Fetching metadata for %s", performer.Name),
		map[string]interface{}{
			"performer_id":   performer.ID,
			"performer_name": performer.Name,
		},
	)
	if err != nil {
		// Log error but continue with the operation
		fmt.Printf("Failed to create activity log: %v\n", err)
	}

	// Get config from context (set by router)
	cfg, exists := c.Get("config")
	if !exists {
		if activity != nil {
			activitySvc.FailTask(activity.ID, "Configuration not available")
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Configuration not available",
			"",
		))
		return
	}

	// Fetch metadata from AdultDataLink API
	adlService := services.NewAdultDataLinkService(cfg.(*config.Config))
	metadata, err := adlService.FetchPerformerData(performer.Name)
	if err != nil {
		if activity != nil {
			activitySvc.FailTask(activity.ID, fmt.Sprintf("Failed to fetch from AdultDataLink: %v", err))
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to fetch metadata from AdultDataLink",
			err.Error(),
		))
		return
	}

	// Update performer with fetched metadata
	updateData := &models.PerformerUpdate{
		Metadata: metadata,
	}
	updatedPerformer, err := svc.Update(id, updateData)
	if err != nil {
		if activity != nil {
			activitySvc.FailTask(activity.ID, fmt.Sprintf("Failed to update performer: %v", err))
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update performer with fetched metadata",
			err.Error(),
		))
		return
	}

	// Mark task as completed
	if activity != nil {
		activitySvc.CompleteTask(int64(activity.ID), fmt.Sprintf("Successfully fetched metadata for %s", performer.Name))
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		updatedPerformer,
		"Metadata fetched and updated successfully",
	))
}

// resetMetadata clears all metadata for a performer
func resetMetadata(c *gin.Context) {
	svc := ensurePerformerService()
	activitySvc := services.NewActivityService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	// Get performer before resetting
	performer, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Performer not found",
			err.Error(),
		))
		return
	}

	// Create activity log
	activity, err := activitySvc.StartTask(
		"reset_metadata",
		fmt.Sprintf("Resetting metadata for %s", performer.Name),
		map[string]interface{}{
			"performer_id":   performer.ID,
			"performer_name": performer.Name,
		},
	)
	if err != nil {
		fmt.Printf("Failed to create activity log: %v\n", err)
	}

	err = svc.ResetMetadata(id)
	if err != nil {
		if activity != nil {
			activitySvc.FailTask(activity.ID, fmt.Sprintf("Reset failed: %v", err))
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to reset metadata",
			err.Error(),
		))
		return
	}

	// Mark task as completed
	if activity != nil {
		activitySvc.CompleteTask(int64(activity.ID), fmt.Sprintf("Successfully reset metadata for %s", performer.Name))
	}

	performer, _ = svc.GetByID(id)
	c.JSON(http.StatusOK, models.SuccessResponse(performer, "Metadata reset successfully"))
}

// resetPreviews resets performer preview videos (placeholder for now)
func resetPreviews(c *gin.Context) {
	svc := ensurePerformerService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	// TODO: Implement preview reset logic
	performer, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Performer not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		performer,
		"Preview reset not yet implemented. Coming soon!",
	))
}

// scanPerformers scans the performer asset folders and creates/updates performers
func scanPerformers(c *gin.Context) {
	scanService := services.NewPerformerScanService()
	activitySvc := services.NewActivityService()

	// Create activity log
	activity, err := activitySvc.StartTask(
		"scan_performers",
		"Scanning performer folders",
		map[string]interface{}{
			"source": "manual_trigger",
		},
	)
	if err != nil {
		fmt.Printf("Failed to create activity log: %v\n", err)
	}

	result, err := scanService.ScanPerformerFolders()
	if err != nil {
		if activity != nil {
			activitySvc.FailTask(activity.ID, fmt.Sprintf("Scan failed: %v", err))
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to scan performer folders",
			err.Error(),
		))
		return
	}

	// Mark task as completed
	if activity != nil {
		activitySvc.CompleteTask(int64(activity.ID), fmt.Sprintf("Scan completed: %d new, %d existing, %d errors",
			result.NewCreated, result.Existing, len(result.Errors)))
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		result,
		"Performer scan completed successfully",
	))
}

// getPerformerPreviews retrieves all preview videos for a performer
func getPerformerPreviews(c *gin.Context) {
	svc := ensurePerformerService()
	scanService := services.NewPerformerScanService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	// Get the performer first
	performer, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Performer not found",
			err.Error(),
		))
		return
	}

	// Get all previews from the performer's folder
	previews, err := scanService.GetPerformerPreviews(performer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get performer previews",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		gin.H{"previews": previews},
		"Previews retrieved successfully",
	))
}
