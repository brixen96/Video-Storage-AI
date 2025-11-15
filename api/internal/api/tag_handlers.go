package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var tagService *services.TagService

// ensureTagService initializes the service if needed
func ensureTagService() *services.TagService {
	if tagService == nil {
		tagService = services.NewTagService()
	}
	return tagService
}

// getTags handles GET /api/v1/tags
func getTags(c *gin.Context) {
	svc := ensureTagService()

	tags, err := svc.GetAll()
	if err != nil {
		log.Printf("Failed to get tags: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tags"})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// getTag handles GET /api/v1/tags/:id
func getTag(c *gin.Context) {
	svc := ensureTagService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	tag, err := svc.GetByID(id)
	if err != nil {
		log.Printf("Failed to get tag %d: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Tag not found"})
		return
	}

	c.JSON(http.StatusOK, tag)
}

// createTag handles POST /api/v1/tags
func createTag(c *gin.Context) {
	svc := ensureTagService()

	var create models.TagCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tag, err := svc.Create(&create)
	if err != nil {
		log.Printf("Failed to create tag: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag"})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

// updateTag handles PUT /api/v1/tags/:id
func updateTag(c *gin.Context) {
	svc := ensureTagService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	var update models.TagUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tag, err := svc.Update(id, &update)
	if err != nil {
		log.Printf("Failed to update tag %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tag"})
		return
	}

	c.JSON(http.StatusOK, tag)
}

// deleteTag handles DELETE /api/v1/tags/:id
func deleteTag(c *gin.Context) {
	svc := ensureTagService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag ID"})
		return
	}

	err = svc.Delete(id)
	if err != nil {
		log.Printf("Failed to delete tag %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tag"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted successfully"})
}

// mergeTags handles POST /api/v1/tags/merge
func mergeTags(c *gin.Context) {
	svc := ensureTagService()

	var request models.TagMergeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := svc.Merge(&request)
	if err != nil {
		log.Printf("Failed to merge tags: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to merge tags"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tags merged successfully"})
}
