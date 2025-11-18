package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var thumbnailService *services.ThumbnailService

func ensureThumbnailService() *services.ThumbnailService {
	if thumbnailService == nil {
		mediaSvc := services.NewMediaService()
		activitySvc := services.NewActivityService()
		librarySvc := services.NewLibraryService()
		thumbnailService = services.NewThumbnailService(mediaSvc, activitySvc, librarySvc)
	}
	return thumbnailService
}

// generateThumbnailsForFolder handles POST /api/libraries/:id/generate-thumbnails
func generateThumbnailsForFolder(c *gin.Context) {
	svc := ensureThumbnailService()

	// Get library ID from URL
	idStr := c.Param("id")
	libraryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid library ID"})
		return
	}

	// Get folder path from query params
	folderPath := c.Query("path")

	// Start background thumbnail generation
	if err := svc.GenerateThumbnailsForFolder(libraryID, folderPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Thumbnail generation started",
		"status":  "processing",
	})
}
