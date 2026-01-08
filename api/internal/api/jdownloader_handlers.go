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

var jdownloaderService *services.JDownloaderService

// ensureJDownloaderService initializes the JDownloader service if needed
func ensureJDownloaderService() *services.JDownloaderService {
	if jdownloaderService == nil {
		jdownloaderService = services.NewJDownloaderService("")
		log.Println("📥 JDownloader Service initialized")
	}
	return jdownloaderService
}

// checkJDownloaderStatus checks if JDownloader is available
func checkJDownloaderStatus(c *gin.Context) {
	svc := ensureJDownloaderService()

	available := svc.IsAvailable()
	var version string
	var err error

	if available {
		version, err = svc.GetVersion()
		if err != nil {
			log.Printf("Error getting JDownloader version: %v", err)
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"available": available,
		"version":   version,
		"url":       "http://localhost:3128",
	}, "JDownloader status retrieved"))
}

// sendLinksToJDownloader sends links to JDownloader
func sendLinksToJDownloader(c *gin.Context) {
	svc := ensureJDownloaderService()

	if !svc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"JDownloader not available",
			"Make sure JDownloader is running and Direct Connection is enabled",
		))
		return
	}

	var req struct {
		Links          []string `json:"links" binding:"required"`
		PackageName    string   `json:"package_name"`
		DestinationDir string   `json:"destination_dir"`
		AutoStart      bool     `json:"auto_start"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if len(req.Links) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"No links provided",
			"At least one link is required",
		))
		return
	}

	var err error
	if req.AutoStart {
		err = svc.AddLinksAndStart(req.Links, req.PackageName, req.DestinationDir)
	} else {
		err = svc.AddLinks(req.Links, req.PackageName, req.DestinationDir)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to send links to JDownloader",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"links_sent": len(req.Links),
		"auto_start": req.AutoStart,
	}, "Links sent to JDownloader successfully"))
}

// sendThreadToJDownloader sends all active links from a thread to JDownloader
func sendThreadToJDownloader(c *gin.Context) {
	svc := ensureJDownloaderService()

	if !svc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"JDownloader not available",
			"Make sure JDownloader is running and Direct Connection is enabled",
		))
		return
	}

	threadIDStr := c.Param("id")
	threadID, err := strconv.ParseInt(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	var req struct {
		DestinationDir string `json:"destination_dir"`
		AutoStart      bool   `json:"auto_start"`
		OnlyActive     bool   `json:"only_active"`     // Only send active (verified) links
		OnlyPending    bool   `json:"only_pending"`    // Only send not-yet-downloaded links
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		req.OnlyActive = true
		req.OnlyPending = true
	}

	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Database unavailable",
			"Could not connect to database",
		))
		return
	}

	// Get thread info
	var threadTitle string
	err = db.QueryRow("SELECT title FROM scraped_threads WHERE id = ?", threadID).Scan(&threadTitle)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Thread not found",
			err.Error(),
		))
		return
	}

	// Build query to get links
	query := `
		SELECT url
		FROM scraped_download_links
		WHERE thread_id = ?
	`
	args := []interface{}{threadID}

	if req.OnlyActive {
		query += " AND status = 'active'"
	}

	if req.OnlyPending {
		query += " AND (download_status IS NULL OR download_status = 'pending')"
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to query links",
			err.Error(),
		))
		return
	}
	defer rows.Close()

	var links []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			continue
		}
		links = append(links, url)
	}

	if len(links) == 0 {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"links_sent": 0,
			"message":    "No links matching criteria found",
		}, "No links to send"))
		return
	}

	// Send to JDownloader
	var sendErr error
	if req.AutoStart {
		sendErr = svc.AddLinksAndStart(links, threadTitle, req.DestinationDir)
	} else {
		sendErr = svc.AddLinks(links, threadTitle, req.DestinationDir)
	}

	if sendErr != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to send links to JDownloader",
			sendErr.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"links_sent":   len(links),
		"thread_title": threadTitle,
		"auto_start":   req.AutoStart,
		"filters": gin.H{
			"only_active":  req.OnlyActive,
			"only_pending": req.OnlyPending,
		},
	}, "Thread links sent to JDownloader successfully"))
}

// getJDownloaderStatus returns current JDownloader download status
func getJDownloaderStatus(c *gin.Context) {
	svc := ensureJDownloaderService()

	if !svc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"JDownloader not available",
			"Make sure JDownloader is running",
		))
		return
	}

	status, err := svc.GetDownloadStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get download status",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(status, "JDownloader status retrieved"))
}

// startJDownloaderDownloads starts all downloads
func startJDownloaderDownloads(c *gin.Context) {
	svc := ensureJDownloaderService()

	if !svc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"JDownloader not available",
			"Make sure JDownloader is running",
		))
		return
	}

	if err := svc.StartDownloads(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to start downloads",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Downloads started"))
}

// stopJDownloaderDownloads stops all downloads
func stopJDownloaderDownloads(c *gin.Context) {
	svc := ensureJDownloaderService()

	if !svc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"JDownloader not available",
			"Make sure JDownloader is running",
		))
		return
	}

	if err := svc.StopDownloads(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to stop downloads",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Downloads stopped"))
}

// getJDownloaderDownloads returns all downloads in JDownloader
func getJDownloaderDownloads(c *gin.Context) {
	svc := ensureJDownloaderService()

	if !svc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"JDownloader not available",
			"Make sure JDownloader is running",
		))
		return
	}

	downloads, err := svc.GetDownloadList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get download list",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"downloads": downloads,
		"count":     len(downloads),
	}, "Download list retrieved"))
}
