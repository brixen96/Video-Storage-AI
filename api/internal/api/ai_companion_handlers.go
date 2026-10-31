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

// truncateErr returns a short, user-friendly error string.
func truncateErr(err error) string {
	if err == nil {
		return "unknown error"
	}
	s := err.Error()
	if len(s) > 120 {
		s = s[:120] + "..."
	}
	return s
}

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
		Scope   string              `json:"scope"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", err.Error()))
		return
	}
	// "rp" = Bannerlord mini-chat thread; anything else is the main companion chat
	scope := request.Scope
	if scope == "" {
		scope = "main"
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
	var toolSteps []string

	// SSE keepalive: send a comment every 15s to prevent connection drops
	// during long LLM processing or tool execution.
	keepAliveDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-keepAliveDone:
				return
			case <-clientGone:
				return
			case <-ticker.C:
				fmt.Fprintf(c.Writer, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	}()
	defer close(keepAliveDone)

	// Set up progress callback so tools can report incremental progress via SSE
	aiCompanionService.SetProgressCallback(func(data map[string]interface{}) {
		evt, _ := json.Marshal(data)
		if data["type"] == "request_info" {
			fmt.Fprintf(c.Writer, "event: request_info\ndata: %s\n\n", string(evt))
		} else {
			fmt.Fprintf(c.Writer, "event: tool_progress\ndata: %s\n\n", string(evt))
		}
		flusher.Flush()
	})
	defer aiCompanionService.SetProgressCallback(nil)

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
	}, func(toolName string) {
		toolSteps = append(toolSteps, toolName)
		evt, _ := json.Marshal(map[string]string{"tool": toolName})
		fmt.Fprintf(c.Writer, "event: tool_call\ndata: %s\n\n", string(evt))
		flusher.Flush()
	}, func(usage map[string]interface{}) {
		evt, _ := json.Marshal(usage)
		fmt.Fprintf(c.Writer, "event: token_usage\ndata: %s\n\n", string(evt))
		flusher.Flush()
	})

	if err != nil {
		// LLM unavailable — fall back to rule-based and send as a single chunk
		log.Printf("[AI] Stream LLM unavailable (%v), falling back to rule-based", err)
		source = "rule-based"
		response, rbErr := aiCompanionService.Chat(request.Message, request.History)
		if rbErr != nil {
			log.Printf("[AI] Rule-based fallback also failed: %v", rbErr)
			response = "Sorry, I'm currently unavailable."
		} else {
			// Prepend a notice so the user knows the LLM wasn't used
			response = fmt.Sprintf("⚠️ *LLM unavailable (%s). Using rule-based responses.*\n\n%s", truncateErr(err), response)
		}
		fullResponse.WriteString(response)
		chunkJSON, _ := json.Marshal(response)
		fmt.Fprintf(c.Writer, "data: %s\n\n", chunkJSON)
		flusher.Flush()
	}

	// NOTE: the V3.2 draft review is NO LONGER a hardcoded post-stream pass.
	// The master model (configured V4-Flash) orchestrates it itself via the
	// review_rp_draft / ask_model tools inside the tool loop — generate →
	// review → revise, per draft. See ai_companion_tools_models.go.

	// Persist the exchange to the database
	if fullResponse.Len() > 0 {
		log.Printf("[AI] Persisting response: source=%s scope=%s len=%d tools=%d", source, scope, fullResponse.Len(), len(toolSteps))
		aiCompanionService.SaveChatMessageScoped("user", request.Message, "user", "", scope)
		toolStepsJSON := ""
		if len(toolSteps) > 0 {
			b, _ := json.Marshal(toolSteps)
			toolStepsJSON = string(b)
		}
		aiCompanionService.SaveChatMessageScoped("assistant", fullResponse.String(), source, toolStepsJSON, scope)

		// Auto-capture (memory layer 3): extract durable facts after every MAIN
		// turn. RP scope is roleplay dialogue, not real taste — never extracted.
		// Runs async so it never delays the stream or the next message.
		if scope == "main" {
			go aiCompanionService.ExtractMemoriesAfterTurn(request.Message, fullResponse.String())
		}
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
	scope := c.Query("scope")
	if scope == "" {
		scope = "main"
	}
	messages, err := aiCompanionService.GetChatHistoryScoped(limit, scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get chat history", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": messages, "count": len(messages)})
}

// clearChatHistory deletes persisted chat messages
func clearChatHistory(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", ""))
		return
	}
	scope := c.Query("scope")
	if scope == "" {
		scope = "main"
	}
	if err := aiCompanionService.ClearChatHistoryScoped(scope); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to clear chat history", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Chat history cleared"})
}

// resetChat resets the companion to defaults: clears chat messages and
// non-app memories, then restores the default app identity memory.
func resetChat(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", ""))
		return
	}
	summary := aiCompanionService.ResetChat()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Companion reset to defaults", "data": summary})
}

// recordUserAction records UI actions for the Companion Supervisor to observe
func recordUserAction(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", ""))
		return
	}
	var request struct {
		Actions []struct {
			Type    string `json:"type"`
			Summary string `json:"summary"`
			Details string `json:"details"`
		} `json:"actions"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || len(request.Actions) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", "non-empty actions array required"))
		return
	}
	if len(request.Actions) > 20 {
		request.Actions = request.Actions[:20]
	}
	recorded := 0
	for _, a := range request.Actions {
		if a.Type == "" || a.Summary == "" {
			continue
		}
		aiCompanionService.RecordUserAction(a.Type, a.Summary, a.Details)
		recorded++
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "recorded": recorded})
}

// getCompanionEvaluations returns recent supervisor evaluations for the Monitor tab
func getCompanionEvaluations(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", ""))
		return
	}
	limit := 20
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	evals, err := aiCompanionService.GetEvaluations(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get evaluations", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": evals, "count": len(evals)})
}

// approveProposal executes a supervisor-proposed action after user approval
func approveProposal(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", ""))
		return
	}
	var id int64
	fmt.Sscanf(c.Param("id"), "%d", &id)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid proposal id", ""))
		return
	}
	result, err := aiCompanionService.ExecuteProposal(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Cannot execute proposal", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// rejectProposal marks a supervisor proposal as rejected
func rejectProposal(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg("AI Companion not initialized", ""))
		return
	}
	var id int64
	fmt.Sscanf(c.Param("id"), "%d", &id)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid proposal id", ""))
		return
	}
	result, err := aiCompanionService.RejectProposal(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Cannot reject proposal", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// ── Roleplay Watcher (AIInfluence NPC files) ──

// getRoleplayWatches returns all NPC file watches
func getRoleplayWatches(c *gin.Context) {
	watches, err := services.GetRoleplayWatches()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get watches", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": watches})
}

// addRoleplayWatch registers an NPC file for watching
func addRoleplayWatch(c *gin.Context) {
	var request struct {
		FilePath string `json:"file_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", err.Error()))
		return
	}
	result, err := services.AddRoleplayWatch(request.FilePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Failed to add watch", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// removeRoleplayWatch stops watching an NPC file
func removeRoleplayWatch(c *gin.Context) {
	var id int64
	fmt.Sscanf(c.Param("id"), "%d", &id)
	if id <= 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid watch id", ""))
		return
	}
	if err := services.RemoveRoleplayWatch(id); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Failed to remove watch", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Watch removed"})
}

// getRoleplaySnapshots returns the latest NPC state per watched file
func getRoleplaySnapshots(c *gin.Context) {
	snapshots, err := services.GetRoleplaySnapshots()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get snapshots", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": snapshots})
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

	// Try LLM (OpenRouter) first, fall back to rule-based
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
		response = fmt.Sprintf("⚠️ *LLM unavailable (%s). Using rule-based responses.*\n\n%s", truncateErr(err), response)
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
	err := db.QueryRow(`SELECT id, COALESCE(api_url, 'https://openrouter.ai/api'), COALESCE(api_key, ''), model_name, temperature, max_tokens, COALESCE(reasoning_effort, 'medium'), COALESCE(autonomy_mode, 'assist'), enabled, created_at, updated_at FROM ai_config WHERE id = 1`).Scan(
		&config.ID, &config.APIURL, &config.APIKey, &config.ModelName, &config.Temperature, &config.MaxTokens, &config.ReasoningEffort, &config.AutonomyMode, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		config = models.AIConfig{
			ID:              1,
			APIURL:          "https://openrouter.ai/api",
			APIKey:          "",
			ModelName:       "mistralai/mistral-nemo",
			Temperature:     0.7,
			MaxTokens:       8192,
			ReasoningEffort: "medium",
			AutonomyMode:    "assist",
			Enabled:         true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
	}
	if config.APIURL == "" {
		config.APIURL = "https://openrouter.ai/api"
	}
	if !services.IsValidReasoningEffort(config.ReasoningEffort) {
		config.ReasoningEffort = "medium"
	}
	if !services.IsValidAutonomyMode(config.AutonomyMode) {
		config.AutonomyMode = "assist"
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
	if !services.IsValidReasoningEffort(config.ReasoningEffort) {
		config.ReasoningEffort = "medium"
	}
	if !services.IsValidAutonomyMode(config.AutonomyMode) {
		config.AutonomyMode = "assist"
	}

	db := database.GetDB()
	_, err := db.Exec(`
		INSERT INTO ai_config (id, api_url, api_key, model_name, temperature, max_tokens, reasoning_effort, autonomy_mode, enabled, updated_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			api_url = excluded.api_url,
			api_key = excluded.api_key,
			model_name = excluded.model_name,
			temperature = excluded.temperature,
			max_tokens = excluded.max_tokens,
			reasoning_effort = excluded.reasoning_effort,
			autonomy_mode = excluded.autonomy_mode,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at`,
		config.APIURL, config.APIKey, config.ModelName, config.Temperature, config.MaxTokens, config.ReasoningEffort, config.AutonomyMode, config.Enabled, time.Now(),
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

// testAIConnection tests the OpenRouter connection
func testAIConnection(c *gin.Context) {
	if aiCompanionService == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"AI Companion not initialized",
			"The AI Companion service is not running",
		))
		return
	}

	result := aiCompanionService.TestAIConnection()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}
