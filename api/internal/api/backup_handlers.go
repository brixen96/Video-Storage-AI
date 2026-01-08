package api

import (
	"log"
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var backupService *services.BackupService

// ensureBackupService initializes the backup service if needed
func ensureBackupService() *services.BackupService {
	if backupService == nil {
		backupService = services.NewBackupService("./backups")
		log.Println("💾 Backup Service initialized")
	}
	return backupService
}

// createDatabaseBackup creates a new database backup
func createDatabaseBackup(c *gin.Context) {
	svc := ensureBackupService()

	var req struct {
		Type string `json:"type"` // manual or automatic
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Default to manual if not specified
		req.Type = "manual"
	}

	if req.Type == "" {
		req.Type = "manual"
	}

	backup, err := svc.CreateBackup(req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create backup",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(backup, "Backup created successfully"))
}

// listDatabaseBackups returns a list of all available backups
func listDatabaseBackups(c *gin.Context) {
	svc := ensureBackupService()

	backups, err := svc.ListBackups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to list backups",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"backups": backups,
		"count":   len(backups),
	}, "Backups retrieved successfully"))
}

// getBackup retrieves information about a specific backup
func getBackup(c *gin.Context) {
	svc := ensureBackupService()
	filename := c.Param("filename")

	backup, err := svc.GetBackup(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Backup not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(backup, "Backup retrieved successfully"))
}

// restoreBackup restores the database from a backup
func restoreBackup(c *gin.Context) {
	svc := ensureBackupService()
	filename := c.Param("filename")

	if err := svc.RestoreBackup(filename); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to restore backup",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Database restored successfully"))
}

// deleteBackup deletes a backup file
func deleteBackup(c *gin.Context) {
	svc := ensureBackupService()
	filename := c.Param("filename")

	if err := svc.DeleteBackup(filename); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete backup",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Backup deleted successfully"))
}

// cleanupOldBackups removes old backups based on retention policy
func cleanupOldBackups(c *gin.Context) {
	svc := ensureBackupService()

	var req struct {
		RetentionDays int `json:"retention_days" binding:"required"`
		KeepMinimum   int `json:"keep_minimum"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Default to keeping at least 3 backups
	if req.KeepMinimum == 0 {
		req.KeepMinimum = 3
	}

	deletedCount, err := svc.CleanupOldBackups(req.RetentionDays, req.KeepMinimum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to cleanup old backups",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"deleted_count":  deletedCount,
		"retention_days": req.RetentionDays,
		"kept_minimum":   req.KeepMinimum,
	}, "Old backups cleaned up successfully"))
}

// getBackupStats returns statistics about backups
func getBackupStats(c *gin.Context) {
	svc := ensureBackupService()

	stats, err := svc.GetBackupStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get backup stats",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, "Backup statistics retrieved successfully"))
}

// downloadBackup allows downloading a backup file
func downloadBackup(c *gin.Context) {
	svc := ensureBackupService()
	filename := c.Param("filename")

	backup, err := svc.GetBackup(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Backup not found",
			err.Error(),
		))
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+backup.Filename)
	c.Header("Content-Type", "application/octet-stream")
	c.File(backup.Path)
}
