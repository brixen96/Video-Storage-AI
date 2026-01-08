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

var notificationService *services.NotificationService

// ensureNotificationService initializes the notification service if needed
func ensureNotificationService() *services.NotificationService {
	if notificationService == nil {
		dbInstance := database.GetDB()
		if dbInstance != nil {
			notificationService = services.NewNotificationService(dbInstance)
			log.Println("📬 Notification Service initialized")
		}
	}
	return notificationService
}

// getNotifications returns all notifications with filters
func getNotifications(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
		))
		return
	}

	unreadOnly := c.Query("unread_only") == "true"
	priority := c.Query("priority")
	category := c.Query("category")
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

	notifications, err := svc.GetAll(unreadOnly, priority, category, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get notifications",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"notifications": notifications,
		"count":         len(notifications),
		"limit":         limit,
		"offset":        offset,
	}, "Notifications retrieved successfully"))
}

// getNotification returns a single notification by ID
func getNotification(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid notification ID",
			err.Error(),
		))
		return
	}

	notification, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Notification not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(notification, "Notification retrieved successfully"))
}

// createNotification creates a new notification
func createNotification(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
		))
		return
	}

	var req models.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	notification, err := svc.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create notification",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(notification, "Notification created successfully"))
}

// markNotificationAsRead marks a notification as read
func markNotificationAsRead(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid notification ID",
			err.Error(),
		))
		return
	}

	if err := svc.MarkAsRead(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to mark as read",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Notification marked as read"))
}

// markAllNotificationsAsRead marks all notifications as read
func markAllNotificationsAsRead(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
		))
		return
	}

	if err := svc.MarkAllAsRead(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to mark all as read",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "All notifications marked as read"))
}

// archiveNotification archives a notification
func archiveNotification(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid notification ID",
			err.Error(),
		))
		return
	}

	if err := svc.Archive(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to archive notification",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Notification archived successfully"))
}

// deleteNotification deletes a notification
func deleteNotification(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
		))
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid notification ID",
			err.Error(),
		))
		return
	}

	if err := svc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete notification",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Notification deleted successfully"))
}

// getNotificationStats returns notification statistics
func getNotificationStats(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
		))
		return
	}

	stats, err := svc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get notification stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Statistics retrieved successfully"))
}

// deleteOldNotifications deletes old archived notifications
func deleteOldNotifications(c *gin.Context) {
	svc := ensureNotificationService()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Service unavailable",
			"Notification service not initialized",
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

	count, err := svc.DeleteOld(req.DaysToKeep)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete old notifications",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"deleted_count": count,
		"days_kept":     req.DaysToKeep,
	}, "Old notifications deleted successfully"))
}
