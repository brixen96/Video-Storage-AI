package api

import (
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var qualitySvc *services.VideoQualityService

func ensureQualitySvc() *services.VideoQualityService {
	if qualitySvc == nil {
		qualitySvc = services.NewVideoQualityService()
	}
	return qualitySvc
}

// qualityCompareVideos handles POST /videos/quality-compare
// Body: { "video_ids": [1, 2, 3] }
func qualityCompareVideos(c *gin.Context) {
	var body struct {
		VideoIDs []int64 `json:"video_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondBadRequest(c, "video_ids array is required", err)
		return
	}
	if len(body.VideoIDs) < 2 {
		respondBadRequest(c, "at least 2 video IDs are required for comparison", nil)
		return
	}

	result, err := ensureQualitySvc().CompareByIDs(body.VideoIDs)
	if err != nil {
		respondInternalError(c, "failed to compare video quality", err)
		return
	}

	respondSuccess(c, result, "quality comparison complete")
}
