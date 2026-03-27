package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var jdownloaderService *services.JDownloaderService

// ensureJDownloaderService initializes the JDownloader service if needed
func ensureJDownloaderService() *services.JDownloaderService {
	if jdownloaderService == nil {
		jdownloaderService = services.NewJDownloaderService()
		log.Println("📥 JDownloader Service initialized")
	}
	return jdownloaderService
}

// getJDownloaderConfig handles GET /api/v1/jdownloader/config
func getJDownloaderConfig(c *gin.Context) {
	svc := ensureJDownloaderService()

	config, err := svc.GetConfig()
	if err != nil {
		log.Printf("Failed to get JDownloader config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get configuration"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// updateJDownloaderConfig handles PUT /api/v1/jdownloader/config
func updateJDownloaderConfig(c *gin.Context) {
	svc := ensureJDownloaderService()

	var config models.JDownloaderConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.UpdateConfig(&config); err != nil {
		log.Printf("Failed to update JDownloader config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration"})
		return
	}

	// Return updated config
	updatedConfig, _ := svc.GetConfig()
	c.JSON(http.StatusOK, updatedConfig)
}

// testJDownloaderConnection handles POST /api/v1/jdownloader/test
func testJDownloaderConnection(c *gin.Context) {
	svc := ensureJDownloaderService()

	result, err := svc.TestConnection()
	if err != nil {
		log.Printf("Failed to test JDownloader connection: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to test connection"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// sendToJDownloader handles POST /api/v1/jdownloader/send
func sendToJDownloader(c *gin.Context) {
	svc := ensureJDownloaderService()
	activitySvc := ensureActivityService()
	videoScraperSvc := ensureVideoScraperService()

	var request models.JDownloaderSendLinksRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.Links) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No links provided"})
		return
	}

	// Extract scraped video IDs and clear them from the link objects before sending to JD.
	// jdownloader_history.video_id has a FK to the regular "videos" table, not scraped_videos,
	// so passing a scraped video ID causes a FK constraint failure.
	scrapedVideoIDs := make([]*int64, len(request.Links))
	for i, link := range request.Links {
		scrapedVideoIDs[i] = link.VideoID
		request.Links[i].VideoID = nil
	}

	// Use autoStart=false so we can rename the file before JD starts downloading
	for i := range request.Links {
		request.Links[i].AutoStart = false
	}

	results, err := svc.SendLinks(request.Links)
	if err != nil {
		log.Printf("Failed to send links to JDownloader: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Count successes/failures, then post-process successful ones
	successCount := 0
	failCount := 0
	for i, r := range results {
		if !r.Success {
			failCount++
			continue
		}
		successCount++
		link := request.Links[i]
		videoID := scrapedVideoIDs[i]

		// Mark video as downloaded immediately (it's been queued in JD)
		if videoID != nil {
			if err := videoScraperSvc.MarkScrapedVideoDownloaded(*videoID); err != nil {
				log.Printf("[JD] Failed to mark video %d as downloaded: %v", *videoID, err)
			}
		}

		// Rename + start in background (3s delay to let JD process the link fully)
		packageName := link.PackageName
		videoTitle := link.PackageName // packageName is the video title
		go func(pkgName, title string) {
			time.Sleep(3 * time.Second)
			if err := svc.RenameAndStartLink(pkgName, title); err != nil {
				log.Printf("[JD] RenameAndStartLink failed for %q: %v", pkgName, err)
			}
		}(packageName, videoTitle)

		// Create activity task and poll download progress
		title := link.PackageName
		if title == "" {
			title = link.URL
		}
		go func(pkgName, taskTitle string, videoID *int64) {
			activity, err := activitySvc.StartTask("jdownloader_download",
				fmt.Sprintf("Downloading: %s", taskTitle),
				map[string]interface{}{"package_name": pkgName},
			)
			if err != nil || activity == nil {
				log.Printf("[JD] Failed to create activity for %q: %v", pkgName, err)
				return
			}
			activityID := int64(activity.ID)

			// Poll download progress every 5s, timeout after 2h
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			deadline := time.Now().Add(2 * time.Hour)
			lastProgress := 0
			nilCount := 0 // consecutive polls where package was not found

			for range ticker.C {
				if time.Now().After(deadline) {
					activitySvc.CompleteTask(activityID, fmt.Sprintf("Download timed out: %s", taskTitle))
					return
				}

				pp, err := svc.GetDownloadPackageProgress(pkgName)
				if err != nil {
					log.Printf("[JD] Progress poll error for %q: %v", pkgName, err)
					continue
				}

				if pp == nil {
					nilCount++
					// Package not visible in download list — treat as complete if:
					//   a) download was in progress (lastProgress > 0), or
					//   b) package absent for 20s (nilCount >= 4) — either it finished fast
					//      and JD removed it, or moveToDownloadlist failed; either way, done.
					if lastProgress > 0 || nilCount >= 4 {
						activitySvc.CompleteTask(activityID, fmt.Sprintf("Downloaded: %s", taskTitle))
						return
					}
					continue
				}
				nilCount = 0 // reset on successful lookup

				if pp.Finished {
					activitySvc.CompleteTask(activityID, fmt.Sprintf("Downloaded: %s", taskTitle))
					return
				}

				if pp.BytesTotal > 0 {
					progress := int(pp.BytesLoaded * 100 / pp.BytesTotal)
					if progress != lastProgress {
						lastProgress = progress
						msg := fmt.Sprintf("Downloading: %s (%d%%)", taskTitle, progress)
						activitySvc.UpdateProgress(int(activityID), progress, msg)
					}
				}
			}
		}(packageName, title, videoID)
	}

	c.JSON(http.StatusOK, gin.H{
		"results":      results,
		"successCount": successCount,
		"failCount":    failCount,
		"total":        len(results),
	})
}

// getJDownloaderConnectionStatus handles GET /api/v1/jdownloader/status
func getJDownloaderConnectionStatus(c *gin.Context) {
	svc := ensureJDownloaderService()

	status, err := svc.GetConnectionStatus()
	if err != nil {
		log.Printf("Failed to get JDownloader status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// getJDownloaderHistory handles GET /api/v1/jdownloader/history
func getJDownloaderHistory(c *gin.Context) {
	svc := ensureJDownloaderService()

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	history, total, err := svc.GetHistory(limit, offset)
	if err != nil {
		log.Printf("Failed to get JDownloader history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   history,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// getJDPackageAddedTimes returns locally-tracked timestamps for packages added via our app
func getJDPackageAddedTimes(c *gin.Context) {
	svc := ensureJDownloaderService()

	times, err := svc.GetPackageAddedTimes()
	if err != nil {
		log.Printf("Failed to get package added times: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get package timestamps"})
		return
	}

	c.JSON(http.StatusOK, times)
}

// clearJDownloaderHistory handles DELETE /api/v1/jdownloader/history
func clearJDownloaderHistory(c *gin.Context) {
	svc := ensureJDownloaderService()

	if err := svc.ClearHistory(); err != nil {
		log.Printf("Failed to clear JDownloader history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "History cleared"})
}

// startJDownloaderDownloads handles POST /api/v1/jdownloader/start
func startJDownloaderDownloads(c *gin.Context) {
	svc := ensureJDownloaderService()

	if err := svc.StartDownloads(); err != nil {
		log.Printf("Failed to start JDownloader downloads: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Downloads started"})
}

// stopJDownloaderDownloads handles POST /api/v1/jdownloader/stop
func stopJDownloaderDownloads(c *gin.Context) {
	svc := ensureJDownloaderService()

	if err := svc.StopDownloads(); err != nil {
		log.Printf("Failed to stop JDownloader downloads: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Downloads stopped"})
}

// checkJDownloaderAvailability handles GET /api/v1/jdownloader/available
func checkJDownloaderAvailability(c *gin.Context) {
	svc := ensureJDownloaderService()

	available := svc.IsAvailable()

	c.JSON(http.StatusOK, gin.H{"available": available})
}

// Legacy handlers for backward compatibility

// checkJDownloaderStatus checks if JDownloader is available (legacy)
func checkJDownloaderStatus(c *gin.Context) {
	svc := ensureJDownloaderService()

	available := svc.IsAvailable()
	config, _ := svc.GetConfig()

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"available": available,
		"host":      config.LocalHost,
		"port":      config.LocalPort,
	}, "JDownloader status retrieved"))
}

// sendLinksToJDownloader sends links to JDownloader (legacy format)
func sendLinksToJDownloader(c *gin.Context) {
	svc := ensureJDownloaderService()

	if !svc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"JDownloader not available",
			"Make sure JDownloader is running and Remote API is enabled",
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

	// Convert to new format
	linkRequests := make([]models.JDownloaderLinkRequest, len(req.Links))
	for i, link := range req.Links {
		linkRequests[i] = models.JDownloaderLinkRequest{
			URL:         link,
			PackageName: req.PackageName,
			AutoStart:   req.AutoStart,
		}
	}

	results, err := svc.SendLinks(linkRequests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to send links to JDownloader",
			err.Error(),
		))
		return
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"links_sent": successCount,
		"auto_start": req.AutoStart,
	}, "Links sent to JDownloader successfully"))
}

// sendThreadToJDownloader sends all active links from a thread to JDownloader
func sendThreadToJDownloader(c *gin.Context) {
	svc := ensureJDownloaderService()

	if !svc.IsAvailable() {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"JDownloader not available",
			"Make sure JDownloader is running and Remote API is enabled",
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
		OnlyActive     bool   `json:"only_active"`
		OnlyPending    bool   `json:"only_pending"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
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

	var linkRequests []models.JDownloaderLinkRequest
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			continue
		}
		linkRequests = append(linkRequests, models.JDownloaderLinkRequest{
			URL:         url,
			PackageName: threadTitle,
			AutoStart:   req.AutoStart,
		})
	}

	if len(linkRequests) == 0 {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"links_sent": 0,
			"message":    "No links matching criteria found",
		}, "No links to send"))
		return
	}

	results, err := svc.SendLinks(linkRequests)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to send links to JDownloader",
			err.Error(),
		))
		return
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"links_sent":   successCount,
		"thread_title": threadTitle,
		"auto_start":   req.AutoStart,
		"filters": gin.H{
			"only_active":  req.OnlyActive,
			"only_pending": req.OnlyPending,
		},
	}, "Thread links sent to JDownloader successfully"))
}

// getJDownloaderStatus returns current JDownloader download status (legacy)
func getJDownloaderStatus(c *gin.Context) {
	svc := ensureJDownloaderService()

	status, err := svc.GetConnectionStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get status",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(status, "JDownloader status retrieved"))
}

// ============ Extended JDownloader Control Handlers ============

// getJDSystemInfo returns JDownloader system information
func getJDSystemInfo(c *gin.Context) {
	svc := ensureJDownloaderService()

	info, err := svc.GetSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// pauseJDDownloads pauses/resumes downloads
func pauseJDDownloads(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		Pause bool `json:"pause"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.PauseDownloads(req.Pause); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Downloads paused state updated", "paused": req.Pause})
}

// getJDSpeed returns current download speed
func getJDSpeed(c *gin.Context) {
	svc := ensureJDownloaderService()

	speed, err := svc.GetDownloadSpeed()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"speed": speed})
}

// setJDSpeedLimit sets the download speed limit
func setJDSpeedLimit(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		Limit int64 `json:"limit"` // bytes per second, 0 = unlimited
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.SetSpeedLimit(req.Limit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Speed limit updated", "limit": req.Limit})
}

// getLinkGrabberLinks returns LinkGrabber links
func getLinkGrabberLinks(c *gin.Context) {
	svc := ensureJDownloaderService()

	links, err := svc.GetLinkGrabberLinks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": links, "total": len(links)})
}

// getLinkGrabberPackages returns LinkGrabber packages
func getLinkGrabberPackages(c *gin.Context) {
	svc := ensureJDownloaderService()

	packages, err := svc.GetLinkGrabberPackages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": packages, "total": len(packages)})
}

// getDownloadLinks returns download list links
func getDownloadLinks(c *gin.Context) {
	svc := ensureJDownloaderService()

	links, err := svc.GetDownloadLinks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": links, "total": len(links)})
}

// getDownloadPackages returns download list packages
func getDownloadPackages(c *gin.Context) {
	svc := ensureJDownloaderService()

	packages, err := svc.GetDownloadPackages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": packages, "total": len(packages)})
}

// moveToDownloads moves links from LinkGrabber to download list
func moveToDownloads(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		LinkIDs    []int64 `json:"linkIds"`
		PackageIDs []int64 `json:"packageIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.MoveToDownloads(req.LinkIDs, req.PackageIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Links moved to downloads"})
}

// removeLinkGrabberLinks removes links from LinkGrabber
func removeLinkGrabberLinks(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		LinkIDs    []int64 `json:"linkIds"`
		PackageIDs []int64 `json:"packageIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.RemoveLinkGrabberLinks(req.LinkIDs, req.PackageIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Links removed from LinkGrabber"})
}

// removeDownloadLinks removes links from download list
func removeDownloadLinks(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		LinkIDs    []int64 `json:"linkIds"`
		PackageIDs []int64 `json:"packageIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.RemoveDownloadLinks(req.LinkIDs, req.PackageIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Links removed from downloads"})
}

// clearLinkGrabber clears all links from LinkGrabber
func clearLinkGrabber(c *gin.Context) {
	svc := ensureJDownloaderService()

	if err := svc.ClearLinkGrabber(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "LinkGrabber cleared"})
}

// setLinkEnabled enables/disables links
func setLinkEnabled(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		Enabled    bool    `json:"enabled"`
		LinkIDs    []int64 `json:"linkIds"`
		PackageIDs []int64 `json:"packageIds"`
		IsDownload bool    `json:"isDownload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.SetLinkEnabled(req.Enabled, req.LinkIDs, req.PackageIDs, req.IsDownload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Link enabled state updated"})
}

// forceDownload forces download of specific links
func forceDownload(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		LinkIDs    []int64 `json:"linkIds"`
		PackageIDs []int64 `json:"packageIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.ForceDownload(req.LinkIDs, req.PackageIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Download forced"})
}

// cleanupFinished removes finished downloads
func cleanupFinished(c *gin.Context) {
	svc := ensureJDownloaderService()

	if err := svc.CleanupFinished(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Finished downloads cleaned up"})
}

// resetLinks resets failed/skipped links
func resetLinks(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		LinkIDs    []int64 `json:"linkIds"`
		PackageIDs []int64 `json:"packageIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.ResetLinks(req.LinkIDs, req.PackageIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Links reset"})
}

// setLinkPriority sets download priority
func setLinkPriority(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		Priority   string  `json:"priority"` // HIGHEST, HIGHER, HIGH, DEFAULT, LOW, LOWER, LOWEST
		LinkIDs    []int64 `json:"linkIds"`
		PackageIDs []int64 `json:"packageIds"`
		IsDownload bool    `json:"isDownload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.SetPriority(req.Priority, req.LinkIDs, req.PackageIDs, req.IsDownload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Priority updated"})
}

// renamePackage renames a package
func renamePackage(c *gin.Context) {
	svc := ensureJDownloaderService()

	var req struct {
		PackageID  int64  `json:"packageId"`
		NewName    string `json:"newName"`
		IsDownload bool   `json:"isDownload"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := svc.RenamePackage(req.PackageID, req.NewName, req.IsDownload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Package renamed"})
}
