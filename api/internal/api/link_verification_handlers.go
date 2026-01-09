package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var linkVerificationService *services.LinkVerificationService

// ensureLinkVerificationService initializes the link verification service if needed
func ensureLinkVerificationService() *services.LinkVerificationService {
	if linkVerificationService == nil {
		activitySvc := ensureActivityService()
		dbInstance := database.GetDB()
		if dbInstance != nil && activitySvc != nil {
			linkVerificationService = services.NewLinkVerificationService(dbInstance, activitySvc)
			if err := linkVerificationService.Start(); err != nil {
				log.Printf("Failed to start link verification service: %v", err)
			}
		}
	}
	return linkVerificationService
}

// verifyThreadLinks verifies all download links for a specific thread
func verifyThreadLinks(c *gin.Context) {
	svc := ensureLinkVerificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Link verification service not initialized",
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

	// Start verification in background
	go func() {
		if err := svc.VerifyThreadLinks(threadID); err != nil {
			log.Printf("Error verifying thread %d links: %v", threadID, err)
		}
	}()

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Link verification started in background"))
}

// getVerificationStats returns overall link verification statistics
func getVerificationStats(c *gin.Context) {
	svc := ensureLinkVerificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Link verification service not initialized",
		))
		return
	}

	stats := svc.GetStats()
	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Statistics retrieved successfully"))
}

// getThreadLinkStats returns link statistics for a specific thread
func getThreadLinkStats(c *gin.Context) {
	svc := ensureLinkVerificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Link verification service not initialized",
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

	stats, err := svc.GetThreadLinkStats(threadID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get thread link stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Thread link statistics retrieved successfully"))
}

// verifyAllLinks verifies all links in the database (with optional limit)
func verifyAllLinks(c *gin.Context) {
	svc := ensureLinkVerificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Link verification service not initialized",
		))
		return
	}

	// Get optional limit from query param
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		limit = 100
	}

	// Verify old links (not checked in last 7 days)
	cutoffTime := c.Query("cutoff_days")
	days := 7
	if cutoffTime != "" {
		if d, err := strconv.Atoi(cutoffTime); err == nil && d > 0 {
			days = d
		}
	}

	// Start verification in background
	go func() {
		cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
		if err := svc.VerifyOldLinks(cutoff, limit); err != nil {
			log.Printf("Error verifying old links: %v", err)
		}
	}()

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"limit":       limit,
		"cutoff_days": days,
		"message":     "Link verification started in background",
	}, "Verification started"))
}

// getProviderHealthStats returns health statistics for all download providers
func getProviderHealthStats(c *gin.Context) {
	svc := ensureLinkVerificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Link verification service not initialized",
		))
		return
	}

	stats, err := svc.GetProviderHealthStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get provider health stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"providers": stats,
		"count":     len(stats),
	}, "Provider health statistics retrieved successfully"))
}

// getProviderHealth returns health statistics for a specific provider
func getProviderHealth(c *gin.Context) {
	svc := ensureLinkVerificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Link verification service not initialized",
		))
		return
	}

	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			"Provider name is required",
		))
		return
	}

	health, err := svc.GetProviderHealth(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get provider health",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(health, "Provider health retrieved successfully"))
}
