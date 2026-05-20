package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var aiCompanionService *services.AICompanionService

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

// aiCompanionChatStream handles chat messages with SSE streaming of the final response
func aiCompanionChatStream(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", "The AI Companion service is not running"))
		return
	}

	var request struct {
		Message string              `json:"message"`
		History []map[string]string `json:"history"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", err.Error()))
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Streaming unsupported", ""))
		return
	}

	clientGone := c.Request.Context().Done()

	var fullResponse strings.Builder
	source := "llm"
	var videoCards []map[string]interface{}

	err := aiCompanionService.ChatLLMStream(request.Message, request.History, func(chunk string) bool {
		select {
		case <-clientGone:
			return false
		default:
		}
		fullResponse.WriteString(chunk)
		chunkJSON, _ := json.Marshal(chunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", chunkJSON)
		flusher.Flush()
		return true
	}, func(cards []map[string]interface{}) {
		videoCards = cards
	})

	if err != nil {
		// LLM unavailable — fall back to rule-based and send as a single chunk
		log.Printf("[AI] Stream LLM unavailable (%v), falling back to rule-based", err)
		source = "rule-based"
		response, rbErr := aiCompanionService.Chat(request.Message, request.History)
		if rbErr != nil {
			response = "Sorry, I’m currently unavailable."
		}
		fullResponse.WriteString(response)
		chunkJSON, _ := json.Marshal(response)
		fmt.Fprintf(c.Writer, "data: %s\n\n", chunkJSON)
		flusher.Flush()
	}

	// Persist the exchange to the database
	if fullResponse.Len() > 0 {
		aiCompanionService.SaveChatMessage("user", request.Message, "user")
		aiCompanionService.SaveChatMessage("assistant", fullResponse.String(), source)
	}

	// Send video cards event before [DONE] so frontend can render inline cards
	if len(videoCards) > 0 {
		cardsJSON, _ := json.Marshal(map[string]interface{}{"videos": videoCards})
		fmt.Fprintf(c.Writer, "event: video_cards\ndata: %s\n\n", string(cardsJSON))
		flusher.Flush()
	}

	// Signal end of stream
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}

// getChatHistory returns persisted chat messages
func getChatHistory(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", ""))
		return
	}
	limit := 100
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	messages, err := aiCompanionService.GetChatHistory(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get chat history", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": messages, "count": len(messages)})
}

// clearChatHistory deletes all persisted chat messages
func clearChatHistory(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", ""))
		return
	}
	if err := aiCompanionService.ClearChatHistory(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to clear chat history", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Chat history cleared"})
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

	// Try LLM (LM Studio) first, fall back to rule-based
	source := "llm"
	response, err := aiCompanionService.ChatLLM(request.Message, request.History)
	if err != nil {
		log.Printf("[AI] LLM unavailable (%v), falling back to rule-based", err)
		source = "rule-based"
		response, err = aiCompanionService.Chat(request.Message, request.History)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Chat failed",
				err.Error(),
			))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": response,
		"source":  source,
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

// getAIConfig returns the AI provider configuration
func getAIConfig(c *gin.Context) {
	db := database.GetDB()
	var config models.AIConfig
	err := db.QueryRow(`SELECT id, COALESCE(api_url, lm_studio_url, 'https://openrouter.ai/api'), COALESCE(api_key, ''), model_name, temperature, max_tokens, enabled, created_at, updated_at FROM ai_config WHERE id = 1`).Scan(
		&config.ID, &config.APIURL, &config.APIKey, &config.ModelName, &config.Temperature, &config.MaxTokens, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		config = models.AIConfig{
			ID:          1,
			APIURL:      "https://openrouter.ai/api",
			APIKey:      "",
			ModelName:   "mistralai/mistral-nemo",
			Temperature: 0.7,
			MaxTokens:   2000,
			Enabled:     true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}
	if config.APIURL == "" || config.APIURL == "http://localhost:1234" {
		config.APIURL = "https://openrouter.ai/api"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
	})
}

// updateAIConfig updates the AI provider configuration
func updateAIConfig(c *gin.Context) {
	var config models.AIConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	db := database.GetDB()
	_, err := db.Exec(`
		UPDATE ai_config SET
			api_url = ?, api_key = ?, model_name = ?, temperature = ?, max_tokens = ?, enabled = ?, updated_at = ?
		WHERE id = 1`,
		config.APIURL, config.APIKey, config.ModelName, config.Temperature, config.MaxTokens, config.Enabled, time.Now(),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update AI config",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "AI config updated successfully",
	})
}

// runVideoHealthCheck triggers an async video health check (fix metadata, thumbnails, unlinked videos).
func runVideoHealthCheck(c *gin.Context) {
	if aiCompanionService != nil {
		go aiCompanionService.RunVideoHealthCheckAndReport(true)
	} else {
		// AI companion not running — run standalone
		go func() {
			svc := services.NewVideoHealthService()
			svc.Run(c.Request.Context())
		}()
	}
	respondSuccess(c, nil, "Health check started — watch the Activity page for progress")
}

// testAIConnection tests the LM Studio connection
func testAIConnection(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"AI Companion not initialized",
			"The AI Companion service is not running",
		))
		return
	}

	result := aiCompanionService.TestLMStudioConnection()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
