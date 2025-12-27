package api

import (
	"fmt"
	"log"
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
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5000"))

	// Ensure valid pagination values
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 5000
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

	// Mark the first item on the first page as an LCP candidate
	if page == 1 && len(performers) > 0 {
		performers[0].IsLCPCandidate = true
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

	// Use UpdateWithThumbnailCheck to automatically regenerate thumbnail if preview changes
	performer, err := svc.UpdateWithThumbnailCheck(id, &update)
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
		log.Printf("Failed to create activity log: %v\n", err)
	}

	err = svc.Delete(id)
	if err != nil {
		if activity != nil {
			if err := activitySvc.FailTask(activity.ID, fmt.Sprintf("Delete failed: %v", err)); err != nil {
				log.Printf("Failed to fail task: %v", err)
			}
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete performer",
			err.Error(),
		))
		return
	}

	// Mark task as completed
	if activity != nil {
		if err := activitySvc.CompleteTask(int64(activity.ID), fmt.Sprintf("Successfully deleted %s", performer.Name)); err != nil {
			log.Printf("Failed to complete task: %v", err)
		}
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
		log.Printf("Failed to create activity log: %v\n", err)
	}

	// Get config from context (set by router)
	cfg, exists := c.Get("config")
	if !exists {
		if activity != nil {
			if err := activitySvc.FailTask(activity.ID, "Configuration not available"); err != nil {
				log.Printf("Failed to fail task: %v", err)
			}
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
			if err := activitySvc.FailTask(activity.ID, fmt.Sprintf("Failed to fetch from AdultDataLink: %v", err)); err != nil {
				log.Printf("Failed to fail task: %v", err)
			}
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
			if err := activitySvc.FailTask(activity.ID, fmt.Sprintf("Failed to update performer: %v", err)); err != nil {
				log.Printf("Failed to fail task: %v", err)
			}
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update performer with fetched metadata",
			err.Error(),
		))
		return
	}

	// Mark task as completed
	if activity != nil {
		if err := activitySvc.CompleteTask(int64(activity.ID), fmt.Sprintf("Successfully fetched metadata for %s", performer.Name)); err != nil {
			log.Printf("Failed to complete task: %v", err)
		}
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
		log.Printf("Failed to create activity log: %v\n", err)
	}

	err = svc.ResetMetadata(id)
	if err != nil {
		if activity != nil {
			if err := activitySvc.FailTask(activity.ID, fmt.Sprintf("Reset failed: %v", err)); err != nil {
				log.Printf("Failed to fail task: %v", err)
			}
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to reset metadata",
			err.Error(),
		))
		return
	}

	// Mark task as completed
	if activity != nil {
		if err := activitySvc.CompleteTask(int64(activity.ID), fmt.Sprintf("Successfully reset metadata for %s", performer.Name)); err != nil {
			log.Printf("Failed to complete task: %v", err)
		}
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
		log.Printf("Failed to create activity log: %v\n", err)
	}

	result, err := scanService.ScanPerformerFolders()
	if err != nil {
		if activity != nil {
			if err := activitySvc.FailTask(activity.ID, fmt.Sprintf("Scan failed: %v", err)); err != nil {
				log.Printf("Failed to fail task: %v", err)
			}
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to scan performer folders",
			err.Error(),
		))
		return
	}

	// Mark task as completed
	if activity != nil {
		if err := activitySvc.CompleteTask(int64(activity.ID), fmt.Sprintf("Scan completed: %d new, %d existing, %d errors",
			result.NewCreated, result.Existing, len(result.Errors))); err != nil {
			log.Printf("Failed to complete task: %v", err)
		}
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

// getPerformerTags retrieves all master tags for a performer
func getPerformerTags(c *gin.Context) {
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

	tags, err := svc.GetPerformerTags(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve performer tags",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		tags,
		"Performer tags retrieved successfully",
	))
}

// addPerformerTag adds a master tag to a performer
func addPerformerTag(c *gin.Context) {
	svc := ensurePerformerService()

	idStr := c.Param("id")
	performerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	var request struct {
		TagID int64 `json:"tag_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	err = svc.AddPerformerTag(performerID, request.TagID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to add performer tag",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Master tag added to performer"))
}

// removePerformerTag removes a master tag from a performer
func removePerformerTag(c *gin.Context) {
	svc := ensurePerformerService()

	idStr := c.Param("id")
	performerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	tagIDStr := c.Param("tagId")
	tagID, err := strconv.ParseInt(tagIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid tag ID",
			err.Error(),
		))
		return
	}

	err = svc.RemovePerformerTag(performerID, tagID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to remove performer tag",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Master tag removed from performer"))
}

// syncPerformerTags syncs a performer's master tags to all their videos
func syncPerformerTags(c *gin.Context) {
	svc := ensurePerformerService()

	idStr := c.Param("id")
	performerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	videosUpdated, err := svc.SyncPerformerTagsToVideos(performerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to sync performer tags",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		gin.H{"videos_updated": videosUpdated},
		fmt.Sprintf("Synced master tags to %d videos", videosUpdated),
	))
}

// generatePerformerThumbnails generates thumbnails for all performers with preview videos
func generatePerformerThumbnails(c *gin.Context) {
	// Create services
	mediaService := services.NewMediaService()
	activityService := services.NewActivityService()
	thumbnailService := services.NewPerformerThumbnailService(mediaService, activityService)

	// Start thumbnail generation (runs asynchronously)
	err := thumbnailService.GenerateAllThumbnails()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to start thumbnail generation",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusAccepted, models.SuccessResponse(
		nil,
		"Performer thumbnail generation started",
	))
}

// generatePerformerThumbnail generates a thumbnail for a specific performer
func generatePerformerThumbnail(c *gin.Context) {
	idStr := c.Param("id")
	performerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	// Create services
	mediaService := services.NewMediaService()
	activityService := services.NewActivityService()
	thumbnailService := services.NewPerformerThumbnailService(mediaService, activityService)

	// Generate thumbnail for this specific performer
	thumbnailPath, err := thumbnailService.GenerateThumbnailForPerformer(performerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to generate thumbnail",
			err.Error(),
		))
		return
	}

	if thumbnailPath == "" {
		c.JSON(http.StatusOK, models.SuccessResponse(
			nil,
			"Performer has no preview video, skipping",
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		gin.H{"thumbnail_path": thumbnailPath},
		"Thumbnail generated successfully",
	))
}

// getPerformerVideos retrieves all videos featuring a specific performer
func getPerformerVideos(c *gin.Context) {
	idStr := c.Param("id")
	performerID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	// Get videos featuring this performer
	activityService := services.NewActivityService()
	libraryService := services.NewLibraryService()
	performerService := services.NewPerformerService()
	videoService := services.NewVideoService(activityService, libraryService, performerService)
	videos, err := videoService.GetByPerformer(performerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve performer videos",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		videos,
		"Performer videos retrieved successfully",
	))
}
