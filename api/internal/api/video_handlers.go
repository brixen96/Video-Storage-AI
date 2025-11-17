package api

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
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
		performerSvc := ensurePerformerService()
		videoService = services.NewVideoService(activitySvc, librarySvc, performerSvc)
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
		log.Printf("Failed to bind JSON in createVideo: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Creating video with data: %+v", create)

	video, err := svc.Create(&create)
	if err != nil {
		log.Printf("Failed to create video: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create video: %v", err)})
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
		log.Printf("Failed to bind query parameters: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid query parameters: %v", err)})
		return
	}

	// Get videos
	videos, total, err := svc.GetAll(&query)
	if err != nil {
		log.Printf("Failed to search videos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to search videos: %v", err)})
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

// openInExplorer handles POST /api/v1/videos/:id/open-in-explorer
func openInExplorer(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	// Get video to retrieve file path
	video, err := svc.GetByID(id)
	if err != nil {
		log.Printf("Failed to get video %d: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	if video.FilePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Video file path not available"})
		return
	}

	// Open file location based on OS
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows: Use explorer.exe with /select parameter to highlight the file
		cmd = exec.Command("explorer", "/select,", video.FilePath)
	case "darwin":
		// macOS: Use open command with -R flag to reveal in Finder
		cmd = exec.Command("open", "-R", video.FilePath)
	case "linux":
		// Linux: Use xdg-open to open the directory (varies by desktop environment)
		// Extract directory from file path
		cmd = exec.Command("xdg-open", video.FilePath[:len(video.FilePath)-len(video.Title)])
	default:
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Unsupported operating system"})
		return
	}

	err = cmd.Start()
	if err != nil {
		log.Printf("Failed to open file location: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to open file location: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File location opened successfully",
		"path":    video.FilePath,
	})
}

// updateVideoMarksByPath handles PATCH /api/v1/videos/marks-by-path
func updateVideoMarksByPath(c *gin.Context) {
	svc := ensureVideoService()

	var request struct {
		FilePath      string `json:"file_path" binding:"required"`
		NotInterested *bool  `json:"not_interested,omitempty"`
		InEditList    *bool  `json:"in_edit_list,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// First, get or create the video record
	video, err := svc.GetByFilePath(request.FilePath)
	if err != nil {
		log.Printf("Failed to get video by path %s: %v", request.FilePath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video"})
		return
	}

	// Update the marks
	update := models.VideoUpdate{
		NotInterested: request.NotInterested,
		InEditList:    request.InEditList,
	}

	updatedVideo, err := svc.Update(video.ID, &update)
	if err != nil {
		log.Printf("Failed to update video marks for %s: %v", request.FilePath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update video marks"})
		return
	}

	c.JSON(http.StatusOK, updatedVideo)
}
