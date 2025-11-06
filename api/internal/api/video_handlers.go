package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var videoService *services.VideoService

// ensureVideoService initializes the service if needed
func ensureVideoService() *services.VideoService {
	if videoService == nil {
		activitySvc := ensureActivityService()
		librarySvc := ensureLibraryService()
		videoService = services.NewVideoService(activitySvc, librarySvc)
	}
	return videoService
}

// getVideos handles GET /api/v1/videos
func getVideos(c *gin.Context) {
	svc := ensureVideoService()

	// Parse query parameters
	var query models.VideoSearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get videos
	videos, total, err := svc.GetAll(&query)
	if err != nil {
		log.Printf("Failed to get videos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve videos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  videos,
		"total": total,
		"page":  query.Page,
		"limit": query.Limit,
	})
}

// getVideo handles GET /api/v1/videos/:id
func getVideo(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	video, err := svc.GetByID(id)
	if err != nil {
		log.Printf("Failed to get video %d: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	c.JSON(http.StatusOK, video)
}

// createVideo handles POST /api/v1/videos
func createVideo(c *gin.Context) {
	svc := ensureVideoService()

	var create models.VideoCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	video, err := svc.Create(&create)
	if err != nil {
		log.Printf("Failed to create video: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video"})
		return
	}

	c.JSON(http.StatusCreated, video)
}

// updateVideo handles PUT /api/v1/videos/:id
func updateVideo(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	var update models.VideoUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	video, err := svc.Update(id, &update)
	if err != nil {
		log.Printf("Failed to update video %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update video"})
		return
	}

	c.JSON(http.StatusOK, video)
}

// deleteVideo handles DELETE /api/v1/videos/:id
func deleteVideo(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	err = svc.Delete(id)
	if err != nil {
		log.Printf("Failed to delete video %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete video"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Video deleted successfully"})
}

// searchVideos handles GET /api/v1/videos/search
func searchVideos(c *gin.Context) {
	svc := ensureVideoService()

	// Parse query parameters
	var query models.VideoSearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get videos
	videos, total, err := svc.GetAll(&query)
	if err != nil {
		log.Printf("Failed to search videos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search videos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  videos,
		"total": total,
		"page":  query.Page,
		"limit": query.Limit,
	})
}

// scanVideos handles POST /api/v1/videos/scan
func scanVideos(c *gin.Context) {
	svc := ensureVideoService()

	// Get library ID from request body
	var request struct {
		LibraryID int64 `json:"library_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "library_id is required"})
		return
	}

	// Start scanning in background
	go func() {
		err := svc.ScanLibrary(request.LibraryID)
		if err != nil {
			log.Printf("Failed to scan library %d: %v", request.LibraryID, err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": fmt.Sprintf("Video scan started for library %d", request.LibraryID),
		"status":  "scanning",
	})
}
