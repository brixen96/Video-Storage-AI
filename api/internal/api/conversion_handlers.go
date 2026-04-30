package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var conversionService *services.ConversionService

func ensureConversionService() *services.ConversionService {
	if conversionService == nil {
		activitySvc := services.NewActivityService()
		librarySvc := services.NewLibraryService()
		performerSvc := services.NewPerformerService()
		videoSvc := services.NewVideoService(activitySvc, librarySvc, performerSvc)
		mediaSvc := services.NewMediaService()
		conversionService = services.NewConversionService(videoSvc, mediaSvc, activitySvc)
	}
	return conversionService
}

// convertVideoToMP4 handles POST /api/videos/:id/convert
func convertVideoToMP4(c *gin.Context) {
	svc := ensureConversionService()

	idStr := c.Param("id")
	videoID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	if !svc.CheckFFmpegInstalled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "FFmpeg is not installed or not available in PATH"})
		return
	}

	activity, err := svc.StartConversionAsync(videoID)
	if err != nil {
		log.Printf("Failed to start conversion for video %d: %v", videoID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, models.SuccessResponse(
		gin.H{"activity_id": activity.ID},
		"Conversion started",
	))
}

// checkFFmpegStatus handles GET /api/conversion/status
func checkFFmpegStatus(c *gin.Context) {
	svc := ensureConversionService()

	isInstalled := svc.CheckFFmpegInstalled()

	c.JSON(http.StatusOK, models.SuccessResponse(
		gin.H{
			"ffmpeg_installed": isInstalled,
		},
		"FFmpeg status checked",
	))
}
