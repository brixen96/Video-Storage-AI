package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var groupService *services.GroupService

// ensureGroupService initializes the service if needed
func ensureGroupService() *services.GroupService {
	if groupService == nil {
		groupService = services.NewGroupService()
	}
	return groupService
}

// getGroups handles GET /api/v1/groups
func getGroups(c *gin.Context) {
	svc := ensureGroupService()

	// Check if filtering by studio_id
	studioIDStr := c.Query("studio_id")
	if studioIDStr != "" {
		studioID, err := strconv.ParseInt(studioIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid studio ID"})
			return
		}

		groups, err := svc.GetByStudioID(studioID)
		if err != nil {
			log.Printf("Failed to get groups for studio %d: %v", studioID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve groups"})
			return
		}

		c.JSON(http.StatusOK, groups)
		return
	}

	// Get all groups
	groups, err := svc.GetAll()
	if err != nil {
		log.Printf("Failed to get groups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve groups"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// getGroup handles GET /api/v1/groups/:id
func getGroup(c *gin.Context) {
	svc := ensureGroupService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	group, err := svc.GetByID(id)
	if err != nil {
		log.Printf("Failed to get group %d: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, group)
}

// createGroup handles POST /api/v1/groups
func createGroup(c *gin.Context) {
	svc := ensureGroupService()

	var create models.GroupCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := svc.Create(&create)
	if err != nil {
		log.Printf("Failed to create group: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// updateGroup handles PUT /api/v1/groups/:id
func updateGroup(c *gin.Context) {
	svc := ensureGroupService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	var update models.GroupUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := svc.Update(id, &update)
	if err != nil {
		log.Printf("Failed to update group %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update group"})
		return
	}

	c.JSON(http.StatusOK, group)
}

// deleteGroup handles DELETE /api/v1/groups/:id
func deleteGroup(c *gin.Context) {
	svc := ensureGroupService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	err = svc.Delete(id)
	if err != nil {
		log.Printf("Failed to delete group %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted successfully"})
}

// resetGroupMetadata handles POST /api/v1/groups/:id/reset-metadata
func resetGroupMetadata(c *gin.Context) {
	svc := ensureGroupService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	group, err := svc.ResetMetadata(id)
	if err != nil {
		log.Printf("Failed to reset metadata for group %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset metadata"})
		return
	}

	c.JSON(http.StatusOK, group)
}
