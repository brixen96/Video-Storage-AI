package api

import (
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var videoLinksSvc *services.VideoLinksService

func ensureVideoLinksSvc() *services.VideoLinksService {
	if videoLinksSvc == nil {
		videoLinksSvc = services.NewVideoLinksService()
	}
	return videoLinksSvc
}

// suggestVideoLinksHandler runs the matching engine and saves suggestions.
// POST /api/v1/video-links/suggest  body: {"edited_library_id": 1, "backup_library_id": 2}
func suggestVideoLinksHandler(c *gin.Context) {
	var req struct {
		EditedLibraryID int64 `json:"edited_library_id" binding:"required"`
		BackupLibraryID int64 `json:"backup_library_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "edited_library_id and backup_library_id are required", err)
		return
	}

	suggestions, err := ensureVideoLinksSvc().RunAndSaveSuggestions(req.EditedLibraryID, req.BackupLibraryID)
	if err != nil {
		respondInternalError(c, "failed to generate suggestions", err)
		return
	}
	if suggestions == nil {
		suggestions = []*models.VideoLink{}
	}

	respondSuccess(c, gin.H{"suggestions": suggestions, "count": len(suggestions)}, "")
}

// getVideoLinksHandler returns links for an edited library.
// GET /api/v1/video-links?edited_library_id=1&status=suggested
func getVideoLinksHandler(c *gin.Context) {
	editedLibID, err := strconv.ParseInt(c.Query("edited_library_id"), 10, 64)
	if err != nil || editedLibID == 0 {
		respondBadRequest(c, "edited_library_id query param required", nil)
		return
	}
	status := c.DefaultQuery("status", "")

	links, err := ensureVideoLinksSvc().GetLinks(editedLibID, status)
	if err != nil {
		respondInternalError(c, "failed to fetch links", err)
		return
	}
	if links == nil {
		links = []*models.VideoLink{}
	}

	respondSuccess(c, gin.H{"links": links, "count": len(links)}, "")
}

// updateVideoLinkStatusHandler updates a single link status.
// PATCH /api/v1/video-links/:id  body: {"status": "confirmed"}
func updateVideoLinkStatusHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "invalid id", err)
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "status field required", err)
		return
	}
	if req.Status != "confirmed" && req.Status != "rejected" && req.Status != "suggested" {
		respondBadRequest(c, "status must be confirmed, rejected, or suggested", nil)
		return
	}
	if err := ensureVideoLinksSvc().UpdateLinkStatus(id, req.Status); err != nil {
		respondInternalError(c, "failed to update link status", err)
		return
	}
	respondSuccess(c, gin.H{"ok": true}, "")
}

// bulkUpdateVideoLinkStatusHandler updates multiple links at once.
// POST /api/v1/video-links/bulk-status  body: {"ids": [1,2,3], "status": "confirmed"}
func bulkUpdateVideoLinkStatusHandler(c *gin.Context) {
	var req struct {
		IDs    []int64 `json:"ids" binding:"required"`
		Status string  `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "ids and status are required", err)
		return
	}
	if req.Status != "confirmed" && req.Status != "rejected" && req.Status != "suggested" {
		respondBadRequest(c, "status must be confirmed, rejected, or suggested", nil)
		return
	}
	if err := ensureVideoLinksSvc().BulkUpdateStatus(req.IDs, req.Status); err != nil {
		respondInternalError(c, "failed to bulk update", err)
		return
	}
	respondSuccess(c, gin.H{"ok": true, "updated": len(req.IDs)}, "")
}

// getVideoLinkForVideoHandler returns the confirmed link for a given video.
// GET /api/v1/video-links/for-video/:id
func getVideoLinkForVideoHandler(c *gin.Context) {
	videoID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "invalid video id", err)
		return
	}
	link, err := ensureVideoLinksSvc().GetLinkForVideo(videoID)
	if err != nil {
		respondInternalError(c, "failed to fetch link", err)
		return
	}
	respondSuccess(c, gin.H{"link": link}, "")
}
