package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var downloadTrackingService *services.DownloadTrackingService

// ensureDownloadTrackingService initializes the download tracking service if needed
func ensureDownloadTrackingService() *services.DownloadTrackingService {
	if downloadTrackingService == nil {
		dbInstance := database.GetDB()
		if dbInstance != nil {
			downloadTrackingService = services.NewDownloadTrackingService(dbInstance)
			log.Println("📥 Download Tracking Service initialized")
		}
	}
	return downloadTrackingService
}

// markLinkAsDownloaded marks a link as downloaded
func markLinkAsDownloaded(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	linkIDStr := c.Param("id")
	linkID, err := strconv.ParseInt(linkIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid link ID",
			err.Error(),
		))
		return
	}

	var req struct {
		DownloadPath string `json:"download_path"`
		Notes        string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := svc.MarkAsDownloaded(linkID, req.DownloadPath, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to mark link as downloaded",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Link marked as downloaded"))
}

// markLinkAsFailed marks a link as failed
func markLinkAsFailed(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	linkIDStr := c.Param("id")
	linkID, err := strconv.ParseInt(linkIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid link ID",
			err.Error(),
		))
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := svc.MarkAsFailed(linkID, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to mark link as failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Link marked as failed"))
}

// resetLinkDownloadStatus resets a link back to pending
func resetLinkDownloadStatus(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	linkIDStr := c.Param("id")
	linkID, err := strconv.ParseInt(linkIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid link ID",
			err.Error(),
		))
		return
	}

	if err := svc.ResetDownloadStatus(linkID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to reset download status",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Download status reset"))
}

// bulkMarkAsDownloaded marks multiple links as downloaded
func bulkMarkAsDownloaded(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	var req struct {
		LinkIDs      []int64 `json:"link_ids" binding:"required"`
		DownloadPath string  `json:"download_path"`
		Notes        string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := svc.BulkMarkAsDownloaded(req.LinkIDs, req.DownloadPath, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to mark links as downloaded",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"updated_count": len(req.LinkIDs),
	}, "Links marked as downloaded"))
}

// getThreadDownloadStats returns download statistics for a thread
func getThreadDownloadStats(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	threadIDStr := c.Param("id")
	threadID, err := strconv.ParseInt(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	stats, err := svc.GetThreadDownloadStats(threadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get thread download stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Thread download statistics retrieved"))
}

// getPerformerDownloadStats returns download statistics for a performer
func getPerformerDownloadStats(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	performerIDStr := c.Param("id")
	performerID, err := strconv.ParseInt(performerIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid performer ID",
			err.Error(),
		))
		return
	}

	stats, err := svc.GetPerformerDownloadStats(performerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get performer download stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Performer download statistics retrieved"))
}

// getGlobalDownloadStats returns overall download statistics
func getGlobalDownloadStats(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	stats, err := svc.GetGlobalDownloadStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get global download stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Global download statistics retrieved"))
}

// getThreadsByDownloadStatus returns threads filtered by download status
func getThreadsByDownloadStatus(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	status := c.DefaultQuery("status", "all") // complete, partial, none, all
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 200 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	threads, err := svc.GetThreadsByDownloadStatus(status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get threads by download status",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"threads": threads,
		"count":   len(threads),
		"status":  status,
		"limit":   limit,
		"offset":  offset,
	}, "Threads retrieved successfully"))
}

// getRecentDownloads returns recently downloaded links
func getRecentDownloads(c *gin.Context) {
	svc := ensureDownloadTrackingService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Download tracking service not initialized",
		))
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	downloads, err := svc.GetRecentDownloads(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get recent downloads",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"downloads": downloads,
		"count":     len(downloads),
	}, "Recent downloads retrieved successfully"))
}
