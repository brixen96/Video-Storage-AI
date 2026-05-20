package api

import (
	"encoding/json"

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

// analyzeVideos handles POST /videos/analyze
// Accepts multipart/form-data with:
//   - video_ids: JSON-encoded int64 array (e.g. "[1,2,3]") — optional
//   - file: one or more uploaded video files — optional
//
// At least 2 total entries (IDs + files) are required.
func analyzeVideos(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		respondBadRequest(c, "invalid multipart form", err)
		return
	}

	var videoIDs []int64
	if vals := form.Value["video_ids"]; len(vals) > 0 && vals[0] != "" {
		if err := json.Unmarshal([]byte(vals[0]), &videoIDs); err != nil {
			respondBadRequest(c, "invalid video_ids format — expected JSON array", err)
			return
		}
	}

	files := form.File["file"]

	if len(videoIDs)+len(files) < 2 {
		respondBadRequest(c, "at least 2 videos are required for comparison", nil)
		return
	}

	result, err := ensureQualitySvc().CompareFromMixed(videoIDs, files)
	if err != nil {
		respondInternalError(c, "failed to analyze videos", err)
		return
	}

	respondSuccess(c, result, "analysis complete")
}
