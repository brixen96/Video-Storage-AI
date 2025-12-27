package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var aiService *services.AIService
var aiCompanionService *services.AICompanionService

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

// detectScenes analyzes videos and detects scene boundaries
func detectScenes(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs []int64 `json:"video_ids"` // Empty array = all videos
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Scene detection request: %d videos", len(request.VideoIDs))

	results, err := svc.DetectScenes(request.VideoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to detect scenes",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Scene detection completed",
		"data": gin.H{
			"results": results,
			"total":   len(results),
		},
	})
}

// classifyContent analyzes videos and classifies content types
func classifyContent(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs []int64 `json:"video_ids"` // Empty array = all videos
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Content classification request: %d videos", len(request.VideoIDs))

	results, err := svc.ClassifyContent(request.VideoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to classify content",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Content classification completed",
		"data": gin.H{
			"results": results,
			"total":   len(results),
		},
	})
}

// analyzeQuality analyzes video quality metrics
func analyzeQuality(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs []int64 `json:"video_ids"` // Empty array = all videos
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Quality analysis request: %d videos", len(request.VideoIDs))

	results, err := svc.AnalyzeQuality(request.VideoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to analyze quality",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Quality analysis completed",
		"data": gin.H{
			"results": results,
			"total":   len(results),
		},
	})
}

// detectMissingMetadata finds videos with incomplete metadata
func detectMissingMetadata(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs []int64 `json:"video_ids"` // Empty array = all videos
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Missing metadata detection request: %d videos", len(request.VideoIDs))

	results, err := svc.DetectMissingMetadata(request.VideoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to detect missing metadata",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Missing metadata detection completed",
		"data": gin.H{
			"results": results,
			"total":   len(results),
		},
	})
}

// detectDuplicates finds duplicate or similar videos
func detectDuplicates(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs []int64 `json:"video_ids"` // Empty array = all videos
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Duplicate detection request: %d videos", len(request.VideoIDs))

	results, err := svc.DetectDuplicates(request.VideoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to detect duplicates",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Duplicate detection completed",
		"data": gin.H{
			"results": results,
			"total":   len(results),
		},
	})
}

// suggestNaming generates better filename suggestions
func suggestNaming(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs []int64 `json:"video_ids"` // Empty array = all videos
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Auto-naming request: %d videos", len(request.VideoIDs))

	results, err := svc.SuggestNaming(request.VideoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to suggest naming",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Naming suggestions completed",
		"data": gin.H{
			"results": results,
			"total":   len(results),
		},
	})
}

// getLibraryAnalytics provides comprehensive library statistics
func getLibraryAnalytics(c *gin.Context) {
	svc := ensureAIService()

	log.Printf("Library analytics request")

	stats, err := svc.GetLibraryAnalytics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get library analytics",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Library analytics completed",
		"data":    stats,
	})
}

// analyzeThumbnailQuality evaluates thumbnail quality
func analyzeThumbnailQuality(c *gin.Context) {
	svc := ensureAIService()

	var request struct {
		VideoIDs []int64 `json:"video_ids"` // Empty array = all videos
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Thumbnail quality analysis request: %d videos", len(request.VideoIDs))

	results, err := svc.AnalyzeThumbnailQuality(request.VideoIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to analyze thumbnail quality",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Thumbnail quality analysis completed",
		"data": gin.H{
			"results": results,
			"total":   len(results),
		},
	})
}

// ================== AI Companion Endpoints ==================

// InitAICompanion initializes the global AI Companion service
func InitAICompanion() *services.AICompanionService {
	if aiCompanionService == nil {
		aiCompanionService = services.NewAICompanionService()
		if err := aiCompanionService.Start(); err != nil {
			log.Printf("Failed to start AI Companion: %v", err)
		}
	}
	return aiCompanionService
}

// GetAICompanionService returns the global AI Companion instance
func GetAICompanionService() *services.AICompanionService {
	return aiCompanionService
}

// aiCompanionChat handles chat messages to the AI Companion
func aiCompanionChat(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"AI Companion not initialized",
			"The AI Companion service is not running",
		))
		return
	}

	var request struct {
		Message string              `json:"message"`
		History []map[string]string `json:"history"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	response, err := aiCompanionService.Chat(request.Message, request.History)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Chat failed",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": response,
	})
}

// getAICompanionStatus returns the AI Companion status
func getAICompanionStatus(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"running": false,
			"message": "AI Companion not initialized",
		})
		return
	}

	status := aiCompanionService.GetStatus()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// saveAIMemory stores a new memory
func saveAIMemory(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"AI Companion not initialized",
			"The AI Companion service is not running",
		))
		return
	}

	var memory models.Memory
	if err := c.ShouldBindJSON(&memory); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if err := aiCompanionService.SaveMemory(&memory); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to save memory",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Memory saved successfully",
		"data":    memory,
	})
}

// getAIMemories retrieves memories with filters
func getAIMemories(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"AI Companion not initialized",
			"The AI Companion service is not running",
		))
		return
	}

	category := c.Query("category")
	limit := 50
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	memories, err := aiCompanionService.GetMemories(category, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get memories",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    memories,
		"count":   len(memories),
	})
}

// searchAIMemories searches memories by query
func searchAIMemories(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"AI Companion not initialized",
			"The AI Companion service is not running",
		))
		return
	}

	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			"Query parameter is required",
		))
		return
	}

	memories, err := aiCompanionService.SearchMemories(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to search memories",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    memories,
		"count":   len(memories),
	})
}

// updateAIMemory updates an existing memory
func updateAIMemory(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"AI Companion not initialized",
			"The AI Companion service is not running",
		))
		return
	}

	var id int64
	if err := c.ShouldBindUri(&struct{ ID int64 }{ID: id}); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			err.Error(),
		))
		return
	}

	var request struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	if err := aiCompanionService.UpdateMemory(id, request.Value); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update memory",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Memory updated successfully",
	})
}

// deleteAIMemory deletes a memory
func deleteAIMemory(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"AI Companion not initialized",
			"The AI Companion service is not running",
		))
		return
	}

	var id int64
	if err := c.ShouldBindUri(&struct{ ID int64 }{ID: id}); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid ID",
			err.Error(),
		))
		return
	}

	if err := aiCompanionService.DeleteMemory(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete memory",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Memory deleted successfully",
	})
}
