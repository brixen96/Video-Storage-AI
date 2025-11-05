package api

import (
	"net/http"
	"strconv"

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

	var performers []models.Performer
	var err error

	if searchTerm != "" {
		performers, err = svc.Search(searchTerm)
	} else {
		performers, err = svc.GetAll()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve performers",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(performers, "Performers retrieved successfully"))
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
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	err = svc.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete performer",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Performer deleted successfully"))
}

// fetchMetadata fetches metadata from AdultDataLink API (placeholder for now)
func fetchMetadata(c *gin.Context) {
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

	// TODO: Implement AdultDataLink API integration
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
		"Metadata fetch not yet implemented. Coming soon!",
	))
}

// resetMetadata clears all metadata for a performer
func resetMetadata(c *gin.Context) {
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

	err = svc.ResetMetadata(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to reset metadata",
			err.Error(),
		))
		return
	}

	performer, _ := svc.GetByID(id)
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

	result, err := scanService.ScanPerformerFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to scan performer folders",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		result,
		"Performer scan completed successfully",
	))
}
