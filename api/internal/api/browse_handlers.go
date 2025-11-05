package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var browseService *services.BrowseService

// ensureBrowseService initializes the service if needed
func ensureBrowseService() *services.BrowseService {
	if browseService == nil {
		browseService = services.NewBrowseService()
	}
	return browseService
}

// browseLibrary browses a library directory
func browseLibrary(c *gin.Context) {
	svc := ensureBrowseService()

	// Get library ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid library ID",
			err.Error(),
		))
		return
	}

	// Get path parameter (default to root)
	path := c.DefaultQuery("path", "")

	// Get metadata extraction flag
	extractMetadata := c.DefaultQuery("metadata", "false") == "true"

	// Browse
	response, err := svc.BrowseLibrary(id, path, extractMetadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to browse library",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(response, "Directory browsed successfully"))
}
