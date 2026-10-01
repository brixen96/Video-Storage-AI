package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

// getCompanionTasks lists tasks with optional filters
func getCompanionTasks(c *gin.Context) {
	taskService := services.NewCompanionTaskService(database.GetDB())
	status := c.DefaultQuery("status", "all")
	createdBy := c.Query("created_by")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	tasks, err := taskService.ListTasks(status, createdBy, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tasks,
		"count":   len(tasks),
	})
}

// createCompanionTask creates a new task
func createCompanionTask(c *gin.Context) {
	var task models.CompanionTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if task.CreatedBy == "" {
		task.CreatedBy = "developer"
	}

	taskService := services.NewCompanionTaskService(database.GetDB())
	if err := taskService.CreateTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    task,
	})
}

// getCompanionTask gets a single task by ID
func getCompanionTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	taskService := services.NewCompanionTaskService(database.GetDB())
	task, err := taskService.GetTask(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    task,
	})
}

// updateCompanionTask updates a task
func updateCompanionTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskService := services.NewCompanionTaskService(database.GetDB())

	// Handle status update separately if provided
	if status, ok := updates["status"].(string); ok {
		resolution, _ := updates["resolution"].(string)
		if err := taskService.UpdateTaskStatus(id, status, resolution); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		delete(updates, "status")
		delete(updates, "resolution")
	}

	// Update remaining fields
	if len(updates) > 0 {
		if err := taskService.UpdateTask(id, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	task, err := taskService.GetTask(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Task updated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    task,
	})
}

// deleteCompanionTask deletes a task
func deleteCompanionTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	taskService := services.NewCompanionTaskService(database.GetDB())
	if err := taskService.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Task deleted",
	})
}

// getCompanionTaskStats gets task board statistics
func getCompanionTaskStats(c *gin.Context) {
	taskService := services.NewCompanionTaskService(database.GetDB())
	stats, err := taskService.GetTaskStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}