package api

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var videoService *services.VideoService

// ensureVideoService initializes the service if needed
func ensureVideoService() *services.VideoService {
	if videoService == nil {
		activitySvc := ensureActivityService()
		librarySvc := ensureLibraryService()
		performerSvc := ensurePerformerService()
		videoService = services.NewVideoService(activitySvc, librarySvc, performerSvc)
	}
	return videoService
}

// getVideos handles GET /api/v1/videos
func getVideos(c *gin.Context) {
	svc := ensureVideoService()

	// Parse query parameters
	var query models.VideoSearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get videos
	videos, total, err := svc.GetAll(&query)
	if err != nil {
		log.Printf("Failed to get videos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve videos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  videos,
		"total": total,
		"page":  query.Page,
		"limit": query.Limit,
	})
}

// getVideo handles GET /api/v1/videos/:id
func getVideo(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	video, err := svc.GetByID(id)
	if err != nil {
		log.Printf("Failed to get video %d: %v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	c.JSON(http.StatusOK, video)
}

// createVideo handles POST /api/v1/videos
func createVideo(c *gin.Context) {
	svc := ensureVideoService()

	var create models.VideoCreate
	if err := c.ShouldBindJSON(&create); err != nil {
		log.Printf("Failed to bind JSON in createVideo: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Creating video with data: %+v", create)

	video, err := svc.Create(&create)
	if err != nil {
		log.Printf("Failed to create video: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create video: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, video)
}

// updateVideo handles PUT /api/v1/videos/:id
func updateVideo(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	var update models.VideoUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	video, err := svc.Update(id, &update)
	if err != nil {
		log.Printf("Failed to update video %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update video"})
		return
	}

	c.JSON(http.StatusOK, video)
}

// deleteVideo handles DELETE /api/v1/videos/:id
func deleteVideo(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid video ID"})
		return
	}

	err = svc.Delete(id)
	if err != nil {
		log.Printf("Failed to delete video %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete video"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Video deleted successfully"})
}

// searchVideos handles GET /api/v1/videos/search
func searchVideos(c *gin.Context) {
	svc := ensureVideoService()

	// Parse query parameters
	var query models.VideoSearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		log.Printf("Failed to bind query parameters: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid query parameters: %v", err)})
		return
	}

	log.Printf("searchVideos: content_type=%s, category=%s, library_id=%d", query.ContentType, query.Category, query.LibraryID)

	// Get videos
	videos, total, err := svc.GetAll(&query)
	if err != nil {
		log.Printf("Failed to search videos: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to search videos: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  videos,
		"total": total,
		"page":  query.Page,
		"limit": query.Limit,
	})
}

// scanVideos handles POST /api/v1/videos/scan
func scanVideos(c *gin.Context) {
	svc := ensureVideoService()

	// Get library ID from request body
	var request struct {
		LibraryID int64 `json:"library_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "library_id is required"})
		return
	}

	// Start scanning in background
	go func() {
		err := svc.ScanLibrary(request.LibraryID)
		if err != nil {
			log.Printf("Failed to scan library %d: %v", request.LibraryID, err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": fmt.Sprintf("Video scan started for library %d", request.LibraryID),
		"status":  "scanning",
	})
}

// scanAllVideosParallel handles POST /api/v1/videos/scan-all-parallel
// Scans all libraries in parallel with drive-aware optimization
func scanAllVideosParallel(c *gin.Context) {
	videoSvc := ensureVideoService()

	// Get configuration from request body
	var request struct {
		ServerDrives    []string `json:"server_drives"`    // e.g., ["Z:", "Y:"]
		LocalDrives     []string `json:"local_drives"`     // e.g., ["C:", "D:"]
		ServerMaxConcurrent int  `json:"server_max_concurrent"` // Max parallel scans for server drives
		LocalMaxConcurrent  int  `json:"local_max_concurrent"`  // Max parallel scans for local drives
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		// Use defaults if not provided
		request.ServerDrives = []string{"Z:", "Y:"}
		request.LocalDrives = []string{"C:", "D:"}
		request.ServerMaxConcurrent = 2  // Conservative for server
		request.LocalMaxConcurrent = 8   // Aggressive for local PC
	}

	// Start parallel scanning in background
	go func() {
		err := videoSvc.ScanAllLibrariesParallel(services.ParallelScanConfig{
			ServerDrives:        request.ServerDrives,
			LocalDrives:         request.LocalDrives,
			ServerMaxConcurrent: request.ServerMaxConcurrent,
			LocalMaxConcurrent:  request.LocalMaxConcurrent,
		})
		if err != nil {
			log.Printf("Failed to scan libraries in parallel: %v", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Parallel library scan started",
		"status":  "scanning",
		"config": gin.H{
			"server_drives":         request.ServerDrives,
			"local_drives":          request.LocalDrives,
			"server_max_concurrent": request.ServerMaxConcurrent,
			"local_max_concurrent":  request.LocalMaxConcurrent,
		},
	})
}

// generateAllPreviews handles POST /api/v1/videos/generate-previews
// Generates preview storyboards for all videos with drive-aware optimization
func generateAllPreviews(c *gin.Context) {
	videoSvc := ensureVideoService()

	// Get configuration from request body
	var request struct {
		ServerDrives        []string `json:"server_drives"`
		LocalDrives         []string `json:"local_drives"`
		ServerMaxConcurrent int      `json:"server_max_concurrent"`
		LocalMaxConcurrent  int      `json:"local_max_concurrent"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		// Use defaults if not provided
		request.ServerDrives = []string{"Z:", "Y:"}
		request.LocalDrives = []string{"C:", "D:"}
		request.ServerMaxConcurrent = 2 // Conservative for server
		request.LocalMaxConcurrent = 8  // Aggressive for local PC
	}

	// Start preview generation in background
	go func() {
		err := videoSvc.GenerateAllPreviews(services.ParallelScanConfig{
			ServerDrives:        request.ServerDrives,
			LocalDrives:         request.LocalDrives,
			ServerMaxConcurrent: request.ServerMaxConcurrent,
			LocalMaxConcurrent:  request.LocalMaxConcurrent,
		})
		if err != nil {
			log.Printf("Failed to generate previews: %v", err)
		}
	}()

	respondSuccess(c, nil, "Preview generation started")
}

// openInExplorer handles POST /api/v1/videos/:id/open-in-explorer
// Note: when the backend runs as a Windows service (session 0), the explorer window
// will not be visible on the user's desktop. The frontend shows the path as a fallback.
func openInExplorer(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "Invalid video ID", err)
		return
	}

	video, err := svc.GetByID(id)
	if err != nil {
		respondInternalError(c, "Video not found", err)
		return
	}

	if video.FilePath == "" {
		respondBadRequest(c, "Video file path not available", nil)
		return
	}

	dir := filepath.Dir(video.FilePath)

	// Best-effort: try to open the file location. May not work from a Windows service context.
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		winPath := filepath.FromSlash(video.FilePath)
		cmd = exec.Command("explorer", "/select,"+winPath)
	case "darwin":
		cmd = exec.Command("open", "-R", video.FilePath)
	case "linux":
		cmd = exec.Command("xdg-open", dir)
	}
	if cmd != nil {
		if startErr := cmd.Start(); startErr != nil {
			log.Printf("[Explorer] launch attempt failed: %v", startErr)
		}
	}

	respondSuccess(c, gin.H{"path": video.FilePath, "dir": dir}, "")
}

// updateVideoMarksByPath handles PATCH /api/v1/videos/marks-by-path
func updateVideoMarksByPath(c *gin.Context) {
	svc := ensureVideoService()

	var request struct {
		FilePath      string  `json:"file_path" binding:"required"`
		NotInterested *bool   `json:"not_interested,omitempty"`
		InEditList    *bool   `json:"in_edit_list,omitempty"`
		ContentType   *string `json:"content_type,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// First, get or create the video record
	video, err := svc.GetByFilePath(request.FilePath)
	if err != nil {
		log.Printf("Failed to get video by path %s: %v", request.FilePath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve video"})
		return
	}

	// Update the marks
	update := models.VideoUpdate{
		NotInterested: request.NotInterested,
		InEditList:    request.InEditList,
		ContentType:   request.ContentType,
	}

	updatedVideo, err := svc.Update(video.ID, &update)
	if err != nil {
		log.Printf("Failed to update video marks for %s: %v", request.FilePath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update video marks"})
		return
	}

	c.JSON(http.StatusOK, updatedVideo)
}

// updateFolderContentType updates content type for all videos in a folder recursively
func updateFolderContentType(c *gin.Context) {
	svc := ensureVideoService()

	var request struct {
		FolderPath  string `json:"folder_path" binding:"required"`
		ContentType string `json:"content_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", err.Error()))
		return
	}

	// Validate content type
	validTypes := map[string]bool{"regular": true, "zoo": true, "3d": true}
	if !validTypes[request.ContentType] {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid content type", "Must be one of: regular, zoo, 3d"))
		return
	}

	// Update all videos in the folder (using LIKE for recursive matching)
	updated, err := svc.UpdateContentTypeByFolder(request.FolderPath, request.ContentType)
	if err != nil {
		log.Printf("Failed to update folder content type: %v", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to update content type", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"updated":      updated,
		"folder_path":  request.FolderPath,
		"content_type": request.ContentType,
	}, fmt.Sprintf("Updated %d videos to %s", updated, request.ContentType)))
}

// linkPerformerToFolder links all videos in a folder to a performer
func linkPerformerToFolder(c *gin.Context) {
	svc := ensureVideoService()

	var request struct {
		FolderPath  string `json:"folder_path" binding:"required"`
		PerformerID int64  `json:"performer_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid request", err.Error()))
		return
	}

	linked, err := svc.LinkPerformerToFolderVideos(request.FolderPath, request.PerformerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to link performer to folder", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"linked":       linked,
		"folder_path":  request.FolderPath,
		"performer_id": request.PerformerID,
	}, fmt.Sprintf("Linked performer to %d videos", linked)))
}

// generateVideoThumbnails generates thumbnails for all videos that don't have them
func generateVideoThumbnails(c *gin.Context) {
	svc := ensureVideoService()

	// Start thumbnail generation asynchronously
	go func() {
		if err := svc.GenerateAllThumbnails(); err != nil {
			log.Printf("Failed to generate video thumbnails: %v", err)
		}
	}()

	respondSuccess(c, nil, "Video thumbnail generation started")
}

// runPathAnalysis handles POST /api/v1/videos/path-analysis
// Runs path-based auto-linking for performers, tags, and studios.
func runPathAnalysis(c *gin.Context) {
	var req struct {
		LibraryID    int64 `json:"library_id"`
		OnlyUnlinked bool  `json:"only_unlinked"`
		DryRun       bool  `json:"dry_run"`
	}
	c.ShouldBindJSON(&req)

	go func() {
		svc := services.NewPathAnalysisService()
		report, err := svc.RunAutoLink(req.LibraryID, req.OnlyUnlinked, req.DryRun)
		if err != nil {
			log.Printf("Path analysis failed: %v", err)
			return
		}
		log.Printf("Path analysis complete: %d videos, %d performers, %d tags, %d studios",
			report.VideosAnalyzed, report.PerformersLinked, report.TagsApplied, report.StudiosLinked)
	}()

	respondSuccess(c, nil, "Path analysis started")
}

// analyzeSingleVideoPath handles POST /api/v1/videos/:id/analyze-path
// Runs path analysis for a single video and returns what was linked.
func analyzeSingleVideoPath(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "invalid video id", err)
		return
	}
	svc := services.NewPathAnalysisService()
	report, err := svc.RunAutoLinkForVideo(id)
	if err != nil {
		respondInternalError(c, "path analysis failed", err)
		return
	}
	respondSuccess(c, report, fmt.Sprintf("Linked %d performers, %d tags, %d studios", report.PerformersLinked, report.TagsApplied, report.StudiosLinked))
}

// locateVideoFile handles GET /api/v1/videos/:id/locate
// Searches the video's library folder for files matching its filename.
func locateVideoFile(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "Invalid video ID", err)
		return
	}

	candidates, err := svc.LocateFile(id)
	if err != nil {
		respondInternalError(c, "Locate failed", err)
		return
	}

	respondSuccess(c, gin.H{"candidates": candidates}, "")
}

// remapVideoPath handles PATCH /api/v1/videos/:id/remap
// Updates the video's file_path and clears is_missing.
func remapVideoPath(c *gin.Context) {
	svc := ensureVideoService()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "Invalid video ID", err)
		return
	}

	var body struct {
		FilePath string `json:"file_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondBadRequest(c, "file_path is required", err)
		return
	}

	if err := svc.RemapVideoPath(id, body.FilePath); err != nil {
		respondInternalError(c, "Remap failed", err)
		return
	}

	respondSuccess(c, nil, "Video path updated")
}

// deleteMissingVideos handles POST /api/v1/videos/missing/cleanup
// Bulk-deletes all is_missing=1 records, optionally filtered by library_id.
func deleteMissingVideos(c *gin.Context) {
	svc := ensureVideoService()

	var body struct {
		LibraryID int64 `json:"library_id"`
	}
	_ = c.ShouldBindJSON(&body)

	count, err := svc.DeleteMissingVideos(body.LibraryID)
	if err != nil {
		respondInternalError(c, "Cleanup failed", err)
		return
	}

	respondSuccess(c, gin.H{"deleted": count}, fmt.Sprintf("Deleted %d missing video records", count))
}

// refreshVideoMetadata handles POST /api/v1/videos/:id/fetch
// Re-reads file metadata from disk using ffprobe and updates the DB.
func refreshVideoMetadata(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "invalid video id", err)
		return
	}

	var filePath string
	if err := database.GetDB().QueryRow(`SELECT file_path FROM videos WHERE id = ?`, id).Scan(&filePath); err != nil {
		respondInternalError(c, "video not found", err)
		return
	}

	mediaSvc := services.NewMediaService()
	meta, err := mediaSvc.ExtractMetadata(filePath)
	if err != nil {
		respondInternalError(c, "failed to read file metadata", err)
		return
	}

	resolution := ""
	if meta.Width > 0 && meta.Height > 0 {
		resolution = fmt.Sprintf("%dx%d", meta.Width, meta.Height)
	}

	if _, err := database.GetDB().Exec(
		`UPDATE videos SET duration = ?, file_size = ?, codec = ?, resolution = ?, bitrate = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		meta.Duration, meta.Size, meta.Codec, resolution, meta.Bitrate, id,
	); err != nil {
		respondInternalError(c, "failed to update metadata", err)
		return
	}

	respondSuccess(c, meta, "Metadata refreshed from file")
}

// bulkVideoOperation handles POST /api/v1/videos/bulk
// Supported operations: "delete", "fetch_metadata", "analyze_path"
func bulkVideoOperation(c *gin.Context) {
	var req struct {
		Operation string  `json:"operation"`
		VideoIDs  []int64 `json:"video_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBadRequest(c, "invalid request", err)
		return
	}
	if len(req.VideoIDs) == 0 {
		respondBadRequest(c, "no video IDs provided", nil)
		return
	}

	switch req.Operation {
	case "delete":
		svc := ensureVideoService()
		deleted := 0
		for _, id := range req.VideoIDs {
			if err := svc.Delete(id); err != nil {
				log.Printf("[Bulk] delete video %d: %v", id, err)
			} else {
				deleted++
			}
		}
		respondSuccess(c, gin.H{"deleted": deleted}, fmt.Sprintf("Deleted %d of %d videos", deleted, len(req.VideoIDs)))

	case "fetch_metadata":
		go func() {
			mediaSvc := services.NewMediaService()
			db := database.GetDB()
			updated := 0
			for _, id := range req.VideoIDs {
				var filePath string
				if err := db.QueryRow(`SELECT file_path FROM videos WHERE id = ?`, id).Scan(&filePath); err != nil {
					continue
				}
				meta, err := mediaSvc.ExtractMetadata(filePath)
				if err != nil {
					log.Printf("[Bulk] fetch_metadata video %d: %v", id, err)
					continue
				}
				resolution := ""
				if meta.Width > 0 && meta.Height > 0 {
					resolution = fmt.Sprintf("%dx%d", meta.Width, meta.Height)
				}
				db.Exec(`UPDATE videos SET duration = ?, file_size = ?, codec = ?, resolution = ?, bitrate = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
					meta.Duration, meta.Size, meta.Codec, resolution, meta.Bitrate, id)
				updated++
			}
			log.Printf("[Bulk] fetch_metadata: updated %d/%d videos", updated, len(req.VideoIDs))
		}()
		respondSuccess(c, nil, fmt.Sprintf("Metadata refresh started for %d videos", len(req.VideoIDs)))

	case "analyze_path":
		go func() {
			pathSvc := services.NewPathAnalysisService()
			perfs, tags, studios := 0, 0, 0
			for _, id := range req.VideoIDs {
				report, err := pathSvc.RunAutoLinkForVideo(id)
				if err != nil {
					log.Printf("[Bulk] analyze_path video %d: %v", id, err)
					continue
				}
				perfs += report.PerformersLinked
				tags += report.TagsApplied
				studios += report.StudiosLinked
			}
			log.Printf("[Bulk] analyze_path: %d videos → %d performers, %d tags, %d studios", len(req.VideoIDs), perfs, tags, studios)
		}()
		respondSuccess(c, nil, fmt.Sprintf("Path analysis started for %d videos", len(req.VideoIDs)))

	default:
		respondBadRequest(c, "unknown operation: "+req.Operation, nil)
	}
}

// getVideoDuplicates handles GET /api/v1/videos/duplicates
func getVideoDuplicates(c *gin.Context) {
	svc := ensureVideoService()
	groups, totalDups, wastedBytes, err := svc.FindDuplicates()
	if err != nil {
		respondInternalError(c, "Failed to find duplicates", err)
		return
	}
	respondSuccess(c, gin.H{
		"groups":           groups,
		"total_groups":     len(groups),
		"total_duplicates": totalDups,
		"wasted_bytes":     wastedBytes,
	}, "")
}


// markWatchedHandler handles PATCH /api/v1/videos/:id/watched
func markWatchedHandler(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondBadRequest(c, "invalid video id", err)
		return
	}
	var body struct {
		Watched bool `json:"watched"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondBadRequest(c, "invalid request body", err)
		return
	}
	if err := ensureVideoService().MarkWatched(id, body.Watched); err != nil {
		respondInternalError(c, "Failed to update watched status", err)
		return
	}
	respondSuccess(c, gin.H{"watched": body.Watched}, "")
}
