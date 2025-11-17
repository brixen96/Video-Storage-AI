package api

import (
	"log"
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var databaseService *services.DatabaseService

// ensureDatabaseService initializes the service if needed
func ensureDatabaseService() *services.DatabaseService {
	if databaseService == nil {
		databaseService = services.NewDatabaseService()
	}
	return databaseService
}

// getDatabaseStats retrieves database statistics
func getDatabaseStats(c *gin.Context) {
	svc := ensureDatabaseService()

	stats, err := svc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve database statistics",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database statistics retrieved successfully",
		"data":    stats,
	})
}

// optimizeDatabase runs VACUUM and ANALYZE on the database
func optimizeDatabase(c *gin.Context) {
	svc := ensureDatabaseService()

	log.Println("Database optimization requested via API")

	err := svc.Optimize()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to optimize database",
			err.Error(),
		))
		return
	}

	// Get updated stats after optimization
	stats, err := svc.GetStats()
	if err != nil {
		log.Printf("Warning: Failed to get stats after optimization: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database optimized successfully",
		"data":    stats,
	})
}

// backupDatabase creates a backup of the database
func backupDatabase(c *gin.Context) {
	svc := ensureDatabaseService()

	log.Println("Database backup requested via API")

	result, err := svc.Backup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to create database backup",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database backup created successfully",
		"data":    result,
	})
}

// listBackups returns a list of available backup files
func listBackups(c *gin.Context) {
	svc := ensureDatabaseService()

	backups, err := svc.ListBackups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to list database backups",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database backups retrieved successfully",
		"data":    backups,
	})
}

// restoreDatabase restores the database from a backup file
func restoreDatabase(c *gin.Context) {
	svc := ensureDatabaseService()

	var request struct {
		BackupPath string `json:"backup_path" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request",
			err.Error(),
		))
		return
	}

	log.Printf("Database restore requested via API: %s", request.BackupPath)

	err := svc.RestoreFromBackup(request.BackupPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to restore database",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database restored successfully",
	})
}
