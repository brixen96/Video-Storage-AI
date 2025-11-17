package api

import (
	"log"
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var aiService *services.AIService

// ensureAIService initializes the service if needed
func ensureAIService() *services.AIService {
	if aiService == nil {
		aiService = services.NewAIService()
	}
	return aiService
}

// autoLinkPerformers analyzes videos and suggests performer links
func autoLinkPerformers(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs  []int64 `json:"video_ids"`  // Empty array = all videos
		AutoApply bool    `json:"auto_apply"` // Auto-apply high confidence matches
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Auto-link performers request: %d videos, auto-apply: %v", len(request.VideoIDs), request.AutoApply)

	suggestions, err := svc.AutoLinkPerformers(request.VideoIDs, request.AutoApply)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to analyze videos",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Video analysis completed",
		"data": gin.H{
			"suggestions": suggestions,
			"total":       len(suggestions),
		},
	})
}

// applyPerformerLinks applies selected performer link suggestions
func applyPerformerLinks(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		Matches []services.PerformerMatch `json:"matches"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Applying %d performer links", len(request.Matches))

	err := svc.ApplySuggestions(request.Matches)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to apply performer links",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Performer links applied successfully",
		"data": gin.H{
			"applied": len(request.Matches),
		},
	})
}

// suggestTags analyzes videos and suggests relevant tags
func suggestTags(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs      []int64 `json:"video_ids"`       // Empty array = all videos
		AutoApply     bool    `json:"auto_apply"`      // Auto-apply high confidence tags
		MinConfidence float64 `json:"min_confidence"`  // Minimum confidence for auto-apply (default 0.85)
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	// Default min confidence to 0.85
	if request.MinConfidence == 0 {
		request.MinConfidence = 0.85
	}

	log.Printf("Smart tagging request: %d videos, auto-apply: %v, min confidence: %.2f",
		len(request.VideoIDs), request.AutoApply, request.MinConfidence)

	suggestions, err := svc.SuggestTags(request.VideoIDs, request.AutoApply, request.MinConfidence)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to analyze videos for tags",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tag analysis completed",
		"data": gin.H{
			"suggestions": suggestions,
			"total":       len(suggestions),
		},
	})
}

// applyTagSuggestions applies selected tag suggestions to a video
func applyTagSuggestions(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoID int64   `json:"video_id"`
		TagIDs  []int64 `json:"tag_ids"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Applying %d tags to video %d", len(request.TagIDs), request.VideoID)

	err := svc.ApplyTagSuggestions(request.VideoID, request.TagIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to apply tag suggestions",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tags applied successfully",
		"data": gin.H{
			"applied": len(request.TagIDs),
		},
	})
}
