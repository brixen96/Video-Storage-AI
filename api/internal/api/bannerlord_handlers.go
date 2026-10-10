package api

import (
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

// ── Bannerlord Campaign endpoints ──
// Per-campaign views of the AIInfluence mod data (read-only).

// bannerlordListCampaigns returns all campaign folders found in save_data/
func bannerlordListCampaigns(c *gin.Context) {
	campaigns, err := services.ListBannerlordCampaigns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to list campaigns", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": campaigns, "count": len(campaigns)})
}

// bannerlordCampaignOverview returns the full overview for one campaign
func bannerlordCampaignOverview(c *gin.Context) {
	overview, err := services.GetBannerlordCampaignOverview(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Failed to load campaign", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

// bannerlordNpcDialogue returns the full conversation history for one NPC file
func bannerlordNpcDialogue(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", "path query param required"))
		return
	}
	data, err := services.GetBannerlordNPCDialogue(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Failed to load dialogue", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// bannerlordRpContext returns the full roleplay writing context for one NPC
func bannerlordRpContext(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", "path query param required"))
		return
	}
	data, err := services.GetBannerlordRPContext(path)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Failed to load RP context", err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}
