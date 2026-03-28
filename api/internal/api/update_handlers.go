package api

import (
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

// checkForUpdate queries GitHub releases and returns version comparison info.
func checkForUpdate(c *gin.Context) {
	svc := services.NewUpdateService()
	info, err := svc.CheckForUpdate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Update check failed", err.Error()))
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse(info, "Update check complete"))
}

// installUpdate downloads the release ZIP and spawns updater.bat, then the caller
// should stop the service (the updater.bat handles restart).
func installUpdate(c *gin.Context) {
	var req struct {
		DownloadURL string `json:"download_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", err.Error()))
		return
	}

	svc := services.NewUpdateService()
	if err := svc.InstallUpdate(req.DownloadURL); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Update failed", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Update downloaded. Services will restart momentarily."))
}
