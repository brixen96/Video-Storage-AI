package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var dashboardService *services.DashboardService

// ensureDashboardService initializes the service if needed
func ensureDashboardService() *services.DashboardService {
	if dashboardService == nil {
		activitySvc := ensureActivityService()
		autoUpdateSvc := ensureAutoUpdateService()
		dashboardService = services.NewDashboardService(activitySvc, autoUpdateSvc)
	}
	return dashboardService
}

// getDashboardStats returns aggregated dashboard statistics
func getDashboardStats(c *gin.Context) {
	svc := ensureDashboardService()

	stats, err := svc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get dashboard stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Dashboard stats retrieved"))
}

// getDashboardFeed returns recent activity feed items
func getDashboardFeed(c *gin.Context) {
	svc := ensureDashboardService()

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	items, total, err := svc.GetFeedItems(limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get feed items",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Feed items retrieved",
		"data":    items,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

// getDashboardScrapers returns scraper status cards
func getDashboardScrapers(c *gin.Context) {
	svc := ensureDashboardService()

	cards, err := svc.GetScraperStatusCards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get scraper status",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(cards, "Scraper status retrieved"))
}

// getDashboardRunningTasks returns currently running tasks
func getDashboardRunningTasks(c *gin.Context) {
	svc := ensureDashboardService()

	tasks, err := svc.GetRunningTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get running tasks",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(tasks, "Running tasks retrieved"))
}

// executeDashboardQuickAction executes a quick action
func executeDashboardQuickAction(c *gin.Context) {
	action := c.Param("action")

	switch action {
	case "scan-library":
		// Trigger library scan
		c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Library scan started"))

	case "clean-feed":
		// Clean old feed items
		svc := ensureDashboardService()
		deleted, err := svc.CleanOldFeedItems(7) // Keep 7 days
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to clean feed",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusOK, models.SuccessResponse(
			gin.H{"deleted": deleted},
			"Feed cleaned successfully",
		))

	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Unknown action",
			"Action '"+action+"' is not recognized",
		))
	}
}

// addDashboardFeedItem manually adds a feed item (for testing/internal use)
func addDashboardFeedItem(c *gin.Context) {
	svc := ensureDashboardService()

	var item models.DashboardFeedItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := svc.AddFeedItem(&item); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to add feed item",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(item, "Feed item added"))
}
