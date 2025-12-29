package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var consoleLogService *services.ConsoleLogService

// ensureConsoleLogService initializes the service if needed
func ensureConsoleLogService() *services.ConsoleLogService {
	if consoleLogService == nil {
		consoleLogService = services.NewConsoleLogService()
	}
	return consoleLogService
}

// getConsoleLogs retrieves all console logs with optional filtering and pagination
func getConsoleLogs(c *gin.Context) {
	svc := ensureConsoleLogService()

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	source := c.Query("source")
	level := c.Query("level")
	search := c.Query("search")

	// Validate pagination
	if limit < 1 || limit > 1000 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * limit

	// Get logs
	logs, total, err := svc.GetAll(limit, offset, source, level, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve console logs",
			err.Error(),
		))
		return
	}

	// Calculate total pages
	totalPages := (total + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Console logs retrieved successfully",
		"data":    logs,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// getConsoleLog retrieves a single console log by ID
func getConsoleLog(c *gin.Context) {
	svc := ensureConsoleLogService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid console log ID",
			err.Error(),
		))
		return
	}

	log, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Console log not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(log, "Console log retrieved successfully"))
}

// createConsoleLog creates a new console log entry
func createConsoleLog(c *gin.Context) {
	svc := ensureConsoleLogService()

	var request struct {
		Source  string                 `json:"source" binding:"required"`
		Level   string                 `json:"level" binding:"required"`
		Message string                 `json:"message" binding:"required"`
		Details map[string]interface{} `json:"details"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Validate source
	validSources := map[string]bool{"api": true, "ai_companion": true, "frontend": true}
	if !validSources[request.Source] {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid source",
			"Source must be one of: api, ai_companion, frontend",
		))
		return
	}

	// Validate level
	validLevels := map[string]bool{"debug": true, "info": true, "warning": true, "error": true}
	if !validLevels[request.Level] {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid level",
			"Level must be one of: debug, info, warning, error",
		))
		return
	}

	if err := svc.LogEntry(request.Source, request.Level, request.Message, request.Details); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create console log",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(nil, "Console log created successfully"))
}

// deleteConsoleLog deletes a console log by ID
func deleteConsoleLog(c *gin.Context) {
	svc := ensureConsoleLogService()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid console log ID",
			err.Error(),
		))
		return
	}

	if err := svc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete console log",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Console log deleted successfully"))
}

// clearConsoleLogs deletes all console logs
func clearConsoleLogs(c *gin.Context) {
	svc := ensureConsoleLogService()

	count, err := svc.DeleteAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to clear console logs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"deleted_count": count,
	}, "Console logs cleared successfully"))
}

// cleanOldConsoleLogs deletes console logs older than specified days
func cleanOldConsoleLogs(c *gin.Context) {
	svc := ensureConsoleLogService()

	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid days parameter",
			"Days must be a positive integer",
		))
		return
	}

	count, err := svc.DeleteOlderThan(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to clean old console logs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"deleted_count": count,
	}, "Old console logs cleaned successfully"))
}

// getConsoleLogStats retrieves statistics about console logs
func getConsoleLogStats(c *gin.Context) {
	svc := ensureConsoleLogService()

	stats, err := svc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve console log stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Console log stats retrieved successfully"))
}
