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

var aiAuditService *services.AIAuditService

// ensureAIAuditService initializes the AI audit service if needed
func ensureAIAuditService() *services.AIAuditService {
	if aiAuditService == nil {
		dbInstance := database.GetDB()
		if dbInstance != nil {
			aiAuditService = services.NewAIAuditService(dbInstance)
			log.Println("🔍 AI Audit Service initialized")
		}
	}
	return aiAuditService
}

// getAIAuditLogs returns AI audit logs with pagination
func getAIAuditLogs(c *gin.Context) {
	svc := ensureAIAuditService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"AI audit service not initialized",
		))
		return
	}

	interactionType := c.Query("interaction_type")
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

	logs, err := svc.GetAll(interactionType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get audit logs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"logs":   logs,
		"count":  len(logs),
		"limit":  limit,
		"offset": offset,
	}, "Audit logs retrieved successfully"))
}

// getAIAuditLog returns a single audit log by ID
func getAIAuditLog(c *gin.Context) {
	svc := ensureAIAuditService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"AI audit service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid audit log ID",
			err.Error(),
		))
		return
	}

	log, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Audit log not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(log, "Audit log retrieved successfully"))
}

// getAIAuditStats returns aggregate AI usage statistics
func getAIAuditStats(c *gin.Context) {
	svc := ensureAIAuditService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"AI audit service not initialized",
		))
		return
	}

	stats, err := svc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get AI stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "AI statistics retrieved successfully"))
}

// getAIAuditLogsByPerformer returns audit logs for a specific performer
func getAIAuditLogsByPerformer(c *gin.Context) {
	svc := ensureAIAuditService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"AI audit service not initialized",
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

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	logs, err := svc.GetByPerformer(performerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get performer audit logs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"logs":  logs,
		"count": len(logs),
	}, "Performer audit logs retrieved successfully"))
}

// getAIAuditLogsByThread returns audit logs for a specific thread
func getAIAuditLogsByThread(c *gin.Context) {
	svc := ensureAIAuditService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"AI audit service not initialized",
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

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	logs, err := svc.GetByThread(threadID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get thread audit logs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"logs":  logs,
		"count": len(logs),
	}, "Thread audit logs retrieved successfully"))
}

// searchAIAuditLogs searches audit logs
func searchAIAuditLogs(c *gin.Context) {
	svc := ensureAIAuditService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"AI audit service not initialized",
		))
		return
	}

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Missing search query",
			"Query parameter 'q' is required",
		))
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 200 {
		limit = 50
	}

	logs, err := svc.Search(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to search audit logs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"logs":  logs,
		"count": len(logs),
		"query": query,
	}, "Search completed successfully"))
}

// deleteOldAIAuditLogs deletes old audit logs
func deleteOldAIAuditLogs(c *gin.Context) {
	svc := ensureAIAuditService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"AI audit service not initialized",
		))
		return
	}

	var req struct {
		DaysToKeep int `json:"days_to_keep" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if req.DaysToKeep < 1 {
		req.DaysToKeep = 30
	}

	count, err := svc.DeleteOldLogs(req.DaysToKeep)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete old logs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"deleted_count": count,
		"days_kept":     req.DaysToKeep,
	}, "Old audit logs deleted successfully"))
}
