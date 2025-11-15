package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var studioService *services.StudioService

// ensureStudioService initializes the service if needed
func ensureStudioService() *services.StudioService {
	if studioService == nil {
		studioService = services.NewStudioService()
	}
	return studioService
}

// getStudios handles GET /api/v1/studios
func getStudios(c *gin.Context) {
	svc := ensureStudioService()

	studios, err := svc.GetAll()
	if err != nil {
		log.Printf("Failed to get studios: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve studios"})
		return
	}

	c.JSON(http.StatusOK, studios)
}

// getStudio handles GET /api/v1/studios/:id
func getStudio(c *gin.Context) {
	svc := ensureStudioService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid studio ID"})
		return
	}

	// Check if we should include groups
	includeGroups := c.Query("include_groups") == "true"

	if includeGroups {
		studioWithGroups, err := svc.GetWithGroups(id)
		if err != nil {
			log.Printf("Failed to get studio %d with groups: %v", id, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Studio not found"})
			return
		}
		c.JSON(http.StatusOK, studioWithGroups)
	} else {
		studio, err := svc.GetByID(id)
		if err != nil {
			log.Printf("Failed to get studio %d: %v", id, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Studio not found"})
			return
		}
		c.JSON(http.StatusOK, studio)
	}
}

// createStudio handles POST /api/v1/studios
func createStudio(c *gin.Context) {
	svc := ensureStudioService()

	var create models.StudioCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studio, err := svc.Create(&create)
	if err != nil {
		log.Printf("Failed to create studio: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create studio"})
		return
	}

	c.JSON(http.StatusCreated, studio)
}

// updateStudio handles PUT /api/v1/studios/:id
func updateStudio(c *gin.Context) {
	svc := ensureStudioService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid studio ID"})
		return
	}

	var update models.StudioUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	studio, err := svc.Update(id, &update)
	if err != nil {
		log.Printf("Failed to update studio %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update studio"})
		return
	}

	c.JSON(http.StatusOK, studio)
}

// deleteStudio handles DELETE /api/v1/studios/:id
func deleteStudio(c *gin.Context) {
	svc := ensureStudioService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid studio ID"})
		return
	}

	err = svc.Delete(id)
	if err != nil {
		log.Printf("Failed to delete studio %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Studio deleted successfully"})
}

// resetStudioMetadata handles POST /api/v1/studios/:id/reset-metadata
func resetStudioMetadata(c *gin.Context) {
	svc := ensureStudioService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid studio ID"})
		return
	}

	studio, err := svc.ResetMetadata(id)
	if err != nil {
		log.Printf("Failed to reset metadata for studio %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset metadata"})
		return
	}

	c.JSON(http.StatusOK, studio)
}
