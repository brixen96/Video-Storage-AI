package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	appconfig "github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// VideoService handles video-related business logic
type VideoService struct {
	db               *sql.DB
	activityService  *ActivityService
	libraryService   *LibraryService
	performerService *PerformerService
}

// NewVideoService creates a new video service
func NewVideoService(activityService *ActivityService, libraryService *LibraryService, performerService *PerformerService) *VideoService {
	return &VideoService{
		db:               database.GetDB(),
		activityService:  activityService,
		libraryService:   libraryService,
		performerService: performerService,
	}
}

// SupportedVideoExtensions lists supported video file extensions
var SupportedVideoExtensions = []string{
	".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".mpg", ".mpeg",
}

// thumbnailJobHierarchical represents a hierarchical thumbnail generation job
type thumbnailJobHierarchical struct {
	videoID int64
	config  ThumbnailConfig
}

// ParallelScanConfig holds configuration for parallel library scanning
type ParallelScanConfig struct {
	ServerDrives        []string
	LocalDrives         []string
	ServerMaxConcurrent int
	LocalMaxConcurrent  int
}

// GetAll retrieves all videos with optional filters
func (s *VideoService) GetAll(query *models.VideoSearchQuery) ([]models.Video, int, error) {
	// Build base query - include library_id in SELECT
	baseQuery := `
		SELECT DISTINCT v.id, v.library_id, v.title, v.file_path, v.file_size, v.duration, v.codec,
		       v.resolution, v.bitrate, v.fps, v.thumbnail_path, v.preview_path, v.date, v.rating, v.description,
		       v.is_favorite, v.is_pinned, v.not_interested, v.in_edit_list, v.content_type, v.created_at, v.updated_at, v.last_played_at, v.play_count,
		       COALESCE(v.is_missing, 0),
		       (SELECT CASE WHEN vl.edited_video_id = v.id THEN vl.backup_video_id ELSE vl.edited_video_id END
		        FROM video_links vl
		        WHERE (vl.edited_video_id = v.id OR vl.backup_video_id = v.id) AND vl.status = 'confirmed'
		        LIMIT 1) AS linked_video_id
		FROM videos v
	`

	// Build WHERE conditions
	var conditions []string
	var args []interface{}
	var joins []string

	// Add library filter if specified
	if query.LibraryID > 0 {
		conditions = append(conditions, "v.library_id = ?")
		args = append(args, query.LibraryID)
	}

	if query.Query != "" {
		// Split into words so every word must appear somewhere in the title/path (AND logic).
		// e.g. "milf big ass" matches "MILF with a big ass" because each word is present.
		// SQLite LIKE is case-insensitive for ASCII by default.
		words := strings.Fields(query.Query)
		for _, word := range words {
			conditions = append(conditions, "(v.title LIKE ? OR v.file_path LIKE ?)")
			args = append(args, "%"+word+"%", "%"+word+"%")
		}
	}

	if query.Resolution != "" {
		conditions = append(conditions, "v.resolution = ?")
		args = append(args, query.Resolution)
	}

	if query.MinDuration > 0 {
		conditions = append(conditions, "v.duration >= ?")
		args = append(args, query.MinDuration)
	}

	if query.MaxDuration > 0 {
		conditions = append(conditions, "v.duration <= ?")
		args = append(args, query.MaxDuration)
	}

	if query.MinSize > 0 {
		conditions = append(conditions, "v.file_size >= ?")
		args = append(args, query.MinSize)
	}

	if query.MaxSize > 0 {
		conditions = append(conditions, "v.file_size <= ?")
		args = append(args, query.MaxSize)
	}

	if query.DateFrom != "" {
		conditions = append(conditions, "v.created_at >= ?")
		args = append(args, query.DateFrom)
	}

	if query.DateTo != "" {
		conditions = append(conditions, "v.created_at <= ?")
		args = append(args, query.DateTo)
	}

	// Add JOIN conditions for relationships
	if query.PerformerID > 0 {
		joins = append(joins, "INNER JOIN video_performers vp ON v.id = vp.video_id")
		conditions = append(conditions, "vp.performer_id = ?")
		args = append(args, query.PerformerID)
	}

	if query.StudioID > 0 {
		joins = append(joins, "INNER JOIN video_studios vs ON v.id = vs.video_id")
		conditions = append(conditions, "vs.studio_id = ?")
		args = append(args, query.StudioID)
	}

	if query.GroupID > 0 {
		joins = append(joins, "INNER JOIN video_groups vg ON v.id = vg.video_id")
		conditions = append(conditions, "vg.group_id = ?")
		args = append(args, query.GroupID)
	}

	if len(query.TagIDs) > 0 {
		joins = append(joins, "INNER JOIN video_tags vt ON v.id = vt.video_id")
		placeholders := make([]string, len(query.TagIDs))
		for i, tagID := range query.TagIDs {
			placeholders[i] = "?"
			args = append(args, tagID)
		}
		conditions = append(conditions, fmt.Sprintf("vt.tag_id IN (%s)", strings.Join(placeholders, ",")))
	}

	// Single tag filter
	if query.TagID > 0 {
		joins = append(joins, "INNER JOIN video_tags vt2 ON v.id = vt2.video_id")
		conditions = append(conditions, "vt2.tag_id = ?")
		args = append(args, query.TagID)
	}

	// Category filter (checks if video has any performers with specified category)
	if query.Category != "" {
		joins = append(joins, "INNER JOIN video_performers vp2 ON v.id = vp2.video_id")
		joins = append(joins, "INNER JOIN performers p ON vp2.performer_id = p.id")
		conditions = append(conditions, "p.category = ?")
		args = append(args, query.Category)
	}

	// Content type filter (filters by video's content_type field)
	if query.ContentType != "" {
		log.Printf("GetAll: Filtering by content_type=%s", query.ContentType)
		conditions = append(conditions, "v.content_type = ?")
		args = append(args, query.ContentType)
	}

	// Add marking filters
	if query.NotInterested != nil {
		conditions = append(conditions, "v.not_interested = ?")
		args = append(args, *query.NotInterested)
	}

	if query.InEditList != nil {
		conditions = append(conditions, "v.in_edit_list = ?")
		args = append(args, *query.InEditList)
	}

	// By default hide missing-file videos; only show them when explicitly requested
	if !query.ShowMissing {
		conditions = append(conditions, "COALESCE(v.is_missing, 0) = 0")
	}

	// Build the WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Build count query separately
	countQuery := "SELECT COUNT(DISTINCT v.id) FROM videos v"
	if len(joins) > 0 {
		countQuery += " " + strings.Join(joins, " ")
	}
	countQuery += whereClause

	// Count total results
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Count query failed: %v", err)
		return nil, 0, fmt.Errorf("failed to count videos: %w", err)
	}

	// Build the main query
	mainQuery := baseQuery
	if len(joins) > 0 {
		mainQuery += " " + strings.Join(joins, " ")
	}
	mainQuery += whereClause

	// Add sorting
	sortBy := query.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := query.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	// Validate sort column to prevent SQL injection
	validSortColumns := map[string]bool{
		"id": true, "created_at": true, "updated_at": true, "title": true,
		"duration": true, "file_size": true, "play_count": true,
	}
	if !validSortColumns[sortBy] {
		sortBy = "created_at"
	}

	// Validate sort order
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}

	mainQuery += fmt.Sprintf(" ORDER BY v.%s %s", sortBy, strings.ToUpper(sortOrder))

	// Add pagination
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	mainQuery += " LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)

	// Execute query
	rows, err := s.db.Query(mainQuery, queryArgs...)
	if err != nil {
		log.Printf("Query execution failed: %v", err)
		return nil, 0, fmt.Errorf("failed to query videos: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	// Initialize empty slice instead of nil
	videos := make([]models.Video, 0)

	for rows.Next() {
		var video models.Video
		var lastPlayedAt sql.NullTime
		var date sql.NullString
		var description sql.NullString
		var previewPath sql.NullString
		var contentType sql.NullString
		var linkedVideoID sql.NullInt64
		err := rows.Scan(
			&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
			&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath, &previewPath,
			&date, &video.Rating, &description, &video.IsFavorite, &video.IsPinned, &video.NotInterested, &video.InEditList,
			&contentType, &video.CreatedAt, &video.UpdatedAt, &lastPlayedAt, &video.PlayCount,
			&video.IsMissing, &linkedVideoID,
		)
		if err != nil {
			log.Printf("Failed to scan video row: %v", err)
			return nil, 0, fmt.Errorf("failed to scan video: %w", err)
		}

		if lastPlayedAt.Valid {
			video.LastPlayedAt = &lastPlayedAt.Time
		}
		if date.Valid {
			video.Date = date.String
		}
		if description.Valid {
			video.Description = description.String
		}
		if previewPath.Valid {
			video.PreviewPath = previewPath.String
		}
		if contentType.Valid {
			video.ContentType = contentType.String
		}
		if linkedVideoID.Valid {
			video.LinkedVideoID = &linkedVideoID.Int64
		}

		videos = append(videos, video)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
		return nil, 0, fmt.Errorf("error iterating videos: %w", err)
	}

	// Load relationships for all videos in batch
	if len(videos) > 0 {
		if err := s.loadVideoRelationshipsBatch(videos); err != nil {
			log.Printf("Warning: Failed to batch load relationships: %v", err)
			// Don't fail the entire request, just log the warning
		}
	}

	return videos, total, nil
}

// GetByID retrieves a video by ID
func (s *VideoService) GetByID(id int64) (*models.Video, error) {
	var video models.Video
	var lastPlayedAt sql.NullTime
	var date sql.NullString
	var description sql.NullString
	var contentType sql.NullString
	var linkedVideoID sql.NullInt64

	query := `
		SELECT id, library_id, title, file_path, file_size, duration, codec, resolution,
		       bitrate, fps, thumbnail_path, date, rating, description, is_favorite, is_pinned,
		       not_interested, in_edit_list, content_type, created_at, updated_at, last_played_at, play_count,
		       (SELECT CASE WHEN vl.edited_video_id = videos.id THEN vl.backup_video_id ELSE vl.edited_video_id END
		        FROM video_links vl
		        WHERE (vl.edited_video_id = videos.id OR vl.backup_video_id = videos.id) AND vl.status = 'confirmed'
		        LIMIT 1) AS linked_video_id
		FROM videos
		WHERE id = ?
	`

	err := s.db.QueryRow(query, id).Scan(
		&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
		&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath,
		&date, &video.Rating, &description, &video.IsFavorite, &video.IsPinned, &video.NotInterested, &video.InEditList,
		&contentType, &video.CreatedAt, &video.UpdatedAt, &lastPlayedAt, &video.PlayCount,
		&linkedVideoID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("video not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	if lastPlayedAt.Valid {
		video.LastPlayedAt = &lastPlayedAt.Time
	}
	if date.Valid {
		video.Date = date.String
	}
	if description.Valid {
		video.Description = description.String
	}
	if contentType.Valid {
		video.ContentType = contentType.String
	}
	if linkedVideoID.Valid {
		video.LinkedVideoID = &linkedVideoID.Int64
	}

	// Load relationships
	if err := s.loadVideoRelationships(&video); err != nil {
		log.Printf("Warning: Failed to load relationships for video %d: %v", video.ID, err)
	}

	return &video, nil
}

// GetByPerformer retrieves all videos featuring a specific performer
func (s *VideoService) GetByPerformer(performerID int64) ([]models.Video, error) {
	query := `
		SELECT DISTINCT v.id, v.library_id, v.title, v.file_path, v.file_size, v.duration, v.codec, v.resolution,
		       v.bitrate, v.fps, v.thumbnail_path, v.date, v.rating, v.description, v.is_favorite, v.is_pinned,
		       v.not_interested, v.in_edit_list, v.content_type, v.created_at, v.updated_at, v.last_played_at, v.play_count
		FROM videos v
		INNER JOIN video_performers vp ON v.id = vp.video_id
		WHERE vp.performer_id = ?
		ORDER BY v.created_at DESC
	`

	rows, err := s.db.Query(query, performerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos by performer: %w", err)
	}
	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video
		var lastPlayedAt sql.NullTime
		var date, description, contentType sql.NullString

		err := rows.Scan(
			&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
			&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath,
			&date, &video.Rating, &description, &video.IsFavorite, &video.IsPinned, &video.NotInterested, &video.InEditList,
			&contentType, &video.CreatedAt, &video.UpdatedAt, &lastPlayedAt, &video.PlayCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}

		if lastPlayedAt.Valid {
			video.LastPlayedAt = &lastPlayedAt.Time
		}
		if date.Valid {
			video.Date = date.String
		}
		if description.Valid {
			video.Description = description.String
		}
		if contentType.Valid {
			video.ContentType = contentType.String
		}

		videos = append(videos, video)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating video rows: %w", err)
	}

	return videos, nil
}

// Create creates a new video
func (s *VideoService) Create(create *models.VideoCreate) (*models.Video, error) {
	video := &models.Video{
		LibraryID:     create.LibraryID,
		Title:         create.Title,
		FilePath:      create.FilePath,
		FileSize:      create.FileSize,
		Duration:      create.Duration,
		Codec:         create.Codec,
		Resolution:    create.Resolution,
		Bitrate:       create.Bitrate,
		FPS:           create.FPS,
		ThumbnailPath: create.ThumbnailPath,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	query := `
		INSERT INTO videos (library_id, title, file_path, file_size, duration, codec, resolution, bitrate, fps, thumbnail_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query,
		video.LibraryID, video.Title, video.FilePath, video.FileSize, video.Duration, video.Codec,
		video.Resolution, video.Bitrate, video.FPS, video.ThumbnailPath, video.CreatedAt, video.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert video: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	video.ID = id
	return video, nil
}

// Update updates an existing video
func (s *VideoService) Update(id int64, update *models.VideoUpdate) (*models.Video, error) {
	video, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update basic fields
	if update.Title != nil {
		video.Title = *update.Title
	}
	if update.Date != nil {
		video.Date = *update.Date
	}
	if update.Rating != nil {
		video.Rating = *update.Rating
	}
	if update.Description != nil {
		video.Description = *update.Description
	}
	if update.IsFavorite != nil {
		video.IsFavorite = *update.IsFavorite
	}
	if update.IsPinned != nil {
		video.IsPinned = *update.IsPinned
	}
	if update.PlayCount != nil {
		video.PlayCount = *update.PlayCount
	}
	if update.NotInterested != nil {
		video.NotInterested = *update.NotInterested
	}
	if update.InEditList != nil {
		video.InEditList = *update.InEditList
	}
	if update.ContentType != nil {
		video.ContentType = *update.ContentType
	}

	video.UpdatedAt = time.Now()

	query := `
		UPDATE videos
		SET title = ?, date = ?, rating = ?, description = ?, is_favorite = ?, is_pinned = ?,
		    play_count = ?, not_interested = ?, in_edit_list = ?, content_type = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(query, video.Title, video.Date, video.Rating, video.Description,
		video.IsFavorite, video.IsPinned, video.PlayCount, video.NotInterested, video.InEditList, video.ContentType, video.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	// Update relationships
	// Update studio relationship
	if update.StudioID != nil {
		// Clear existing studios
		if _, err := s.db.Exec("DELETE FROM video_studios WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear studios: %w", err)
		}
		// Add new studio if provided
		if *update.StudioID > 0 {
			if _, err := s.db.Exec("INSERT INTO video_studios (video_id, studio_id) VALUES (?, ?)",
				id, *update.StudioID); err != nil {
				return nil, fmt.Errorf("failed to add studio: %w", err)
			}
		}
	}

	// Update group relationship
	if update.GroupID != nil {
		// Clear existing groups
		if _, err := s.db.Exec("DELETE FROM video_groups WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear groups: %w", err)
		}
		// Add new group if provided
		if *update.GroupID > 0 {
			if _, err := s.db.Exec("INSERT INTO video_groups (video_id, group_id) VALUES (?, ?)",
				id, *update.GroupID); err != nil {
				return nil, fmt.Errorf("failed to add group: %w", err)
			}
		}
	}

	// Update performer relationships
	if update.PerformerIDs != nil {
		// Get existing performers for this video to track changes
		existingQuery := `SELECT performer_id FROM video_performers WHERE video_id = ?`
		existingRows, err := s.db.Query(existingQuery, id)
		if err != nil {
			return nil, fmt.Errorf("failed to query existing performers: %w", err)
		}

		existingPerformers := make(map[int64]bool)
		for existingRows.Next() {
			var performerID int64
			if err := existingRows.Scan(&performerID); err == nil {
				existingPerformers[performerID] = true
			}
		}
		existingRows.Close()

		// Clear existing performers
		if _, err := s.db.Exec("DELETE FROM video_performers WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear performers: %w", err)
		}

		// Decrement counts for removed performers
		for oldPerformerID := range existingPerformers {
			if s.performerService != nil {
				if err := s.performerService.DecrementVideoCount(oldPerformerID); err != nil {
					log.Printf("Warning: failed to decrement video count for performer %d: %v", oldPerformerID, err)
				}
			}
		}

		// Add new performers
		for _, performerID := range update.PerformerIDs {
			if _, err := s.db.Exec("INSERT INTO video_performers (video_id, performer_id) VALUES (?, ?)",
				id, performerID); err != nil {
				return nil, fmt.Errorf("failed to add performer: %w", err)
			}

			// Increment count for newly added performer
			if s.performerService != nil {
				if err := s.performerService.IncrementVideoCount(performerID); err != nil {
					log.Printf("Warning: failed to increment video count for performer %d: %v", performerID, err)
				}

				// Auto-apply performer's master tags to this video
				if err := s.performerService.ApplyMasterTagsToVideo(performerID, id); err != nil {
					// Log error but don't fail the entire operation
					log.Printf("Warning: failed to apply master tags from performer %d to video %d: %v", performerID, id, err)
				}
			}
		}
	}

	// Update tag relationships
	if update.TagIDs != nil {
		// Clear existing tags
		if _, err := s.db.Exec("DELETE FROM video_tags WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear tags: %w", err)
		}
		// Add new tags
		for _, tagID := range update.TagIDs {
			if _, err := s.db.Exec("INSERT INTO video_tags (video_id, tag_id) VALUES (?, ?)",
				id, tagID); err != nil {
				return nil, fmt.Errorf("failed to add tag: %w", err)
			}
		}
	}

	// Reload video with updated relationships
	return s.GetByID(id)
}

// Delete deletes a video
func (s *VideoService) Delete(id int64) error {
	query := `DELETE FROM videos WHERE id = ?`
	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}
	return nil
}

// ScanLibrary scans a library for video files
func (s *VideoService) ScanLibrary(libraryID int64) error {
	// Initialize console log service
	consoleLogSvc := NewConsoleLogService()

	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		consoleLogSvc.LogAPI("error", "Failed to get library for scan", map[string]interface{}{
			"library_id": libraryID,
			"error":      err.Error(),
		})
		return fmt.Errorf("library not found: %w", err)
	}

	// Log scan start
	consoleLogSvc.LogAPI("info", fmt.Sprintf("Library scan started: %s", library.Name), map[string]interface{}{
		"library_id":   libraryID,
		"library_name": library.Name,
		"library_path": library.Path,
	})

	// Create activity log
	activity, err := s.activityService.StartTask(
		"video_scan",
		fmt.Sprintf("Scanning library: %s", library.Name),
		map[string]interface{}{
			"library_id":   libraryID,
			"library_name": library.Name,
			"library_path": library.Path,
		},
	)
	if err != nil {
		consoleLogSvc.LogAPI("error", "Failed to create activity log for library scan", map[string]interface{}{
			"library_id": libraryID,
			"error":      err.Error(),
		})
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	// Scan for video files
	videoFiles, skippedDirs, err := s.findVideoFiles(library.Path)
	if err != nil {
		// Give a helpful hint when the path looks like a mapped network drive.
		// Windows services run as SYSTEM and have no access to user-mapped drives (Y:, Z:, etc.).
		// The fix is to use UNC paths (\\server\share\...) or run the service as the user account.
		failMsg := fmt.Sprintf("Failed to scan directory: %v", err)
		if len(library.Path) >= 2 && library.Path[1] == ':' {
			failMsg += " — mapped drive letters are not accessible to Windows services. Use a UNC path (\\\\server\\share\\...) or change the service logon account to your user in services.msc."
		}
		consoleLogSvc.LogAPI("error", "Failed to scan directory for videos", map[string]interface{}{
			"library_id":   libraryID,
			"library_name": library.Name,
			"library_path": library.Path,
			"error":        err.Error(),
			"hint":         "If path uses a mapped drive letter, use a UNC path instead",
		})
		if err := s.activityService.FailTask(activity.ID, failMsg); err != nil {
			log.Printf("Failed to fail task: %v", err)
		}
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	total := len(videoFiles)
	logFields := map[string]interface{}{
		"library_id":   libraryID,
		"library_name": library.Name,
		"total_files":  total,
	}
	progressMsg := fmt.Sprintf("Found %d video files", total)
	if len(skippedDirs) > 0 {
		logFields["skipped_dirs"] = skippedDirs
		progressMsg += fmt.Sprintf(" (%d folder(s) skipped — access denied, check SMB credentials)", len(skippedDirs))
	}
	consoleLogSvc.LogAPI("info", fmt.Sprintf("Found %d video files in library: %s", total, library.Name), logFields)
	if err := s.activityService.UpdateProgress(activity.ID, 0, progressMsg); err != nil {
		log.Printf("Failed to update progress: %v", err)
	}

	// Process each video file
	processed := 0
	added := 0
	skipped := 0

	// Create media service for metadata extraction
	mediaService := NewMediaService()

	// Get thumbnail base directory
	thumbnailDir := os.Getenv("THUMBNAIL_DIR")
	if thumbnailDir == "" {
		thumbnailDir = filepath.Join(appconfig.DataDir(), "thumbnails")
	}

	// Setup parallel thumbnail generation using new hierarchical structure
	numWorkers := runtime.NumCPU() * 2 // Use 2x CPU cores for I/O bound work
	thumbnailJobs := make(chan thumbnailJobHierarchical, numWorkers*2)
	var wg sync.WaitGroup
	var thumbnailMutex sync.Mutex

	// Start thumbnail worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range thumbnailJobs {
				result, err := mediaService.GenerateThumbnailHierarchical(job.config)
				if err != nil {
					log.Printf("Worker %d: Failed to generate thumbnail for video ID %d: %v", workerID, job.videoID, err)
				} else {
					log.Printf("Worker %d: Generated thumbnail for video ID %d at %s", workerID, job.videoID, result.RelativePath)
					// Update video thumbnail path in database
					thumbnailMutex.Lock()
					s.updateVideoThumbnailPath(job.videoID, result.RelativePath)
					thumbnailMutex.Unlock()
				}
			}
		}(i)
	}

	// Process videos sequentially, but queue thumbnails for parallel generation
	for _, filePath := range videoFiles {
		// Respect pause/cancel — check before each file so the task stops promptly
		if IsActivityPaused(int64(activity.ID)) {
			log.Printf("Library scan cancelled/paused at %d/%d files", processed, total)
			break
		}

		processed++
		progress := int((float64(processed) / float64(total)) * 100)
		currentFile := filepath.Base(filePath)

		// Update progress with current file being processed
		progressMsg := fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d)\nCurrent: %s", processed, total, skipped, added, currentFile)

		// Check if video already exists
		exists, err := s.videoExists(filePath)
		if err == nil && exists {
			skipped++
			if err := s.activityService.UpdateProgress(activity.ID, progress, progressMsg); err != nil {
				log.Printf("Failed to update progress: %v", err)
			}
			continue
		}

		// Get file size
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			skipped++
			progressMsg = fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d) - Error: %v\nCurrent: %s", processed, total, skipped, added, err, currentFile)
			if err := s.activityService.UpdateProgress(activity.ID, progress, progressMsg); err != nil {
				log.Printf("Failed to update progress: %v", err)
			}
			continue
		}

		// Extract metadata using MediaService
		metadata, err := mediaService.ExtractMetadata(filePath)
		if err != nil {
			// If metadata extraction fails, still create the video with basic info
			metadata = &VideoMetadata{
				Duration: 0,
				Size:     fileInfo.Size(),
			}
		}

		// Build resolution string
		resolution := ""
		if metadata.Width > 0 && metadata.Height > 0 {
			resolution = fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
		}

		// Get expected thumbnail path using hierarchical structure (but don't generate yet)
		thumbnailConfig := ThumbnailConfig{
			LibraryID:     libraryID,
			LibraryPath:   library.Path,
			VideoFilePath: filePath,
			Duration:      metadata.Duration,
			ThumbnailDir:  thumbnailDir,
		}

		// Create video record — thumbnail_path is intentionally empty here.
		// The parallel thumbnail worker sets it only after successful generation,
		// so the frontend never sees a path that points to a non-existent file.
		create := &models.VideoCreate{
			LibraryID:  libraryID,
			Title:      filepath.Base(filePath),
			FilePath:   filePath,
			FileSize:   metadata.Size,
			Duration:   metadata.Duration,
			Codec:      metadata.Codec,
			Resolution: resolution,
			Bitrate:    metadata.Bitrate,
			FPS:        metadata.FrameRate,
		}

		video, err := s.Create(create)
		if err != nil {
			skipped++
			progressMsg = fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d) - Error: %v\nCurrent: %s", processed, total, skipped, added, err, currentFile)
			if err := s.activityService.UpdateProgress(activity.ID, progress, progressMsg); err != nil {
				log.Printf("Failed to update progress: %v", err)
			}
			continue
		}

		// Queue thumbnail generation for parallel processing using hierarchical structure
		thumbnailJobs <- thumbnailJobHierarchical{
			videoID: video.ID,
			config:  thumbnailConfig,
		}

		thumbnailMutex.Lock()
		added++
		thumbnailMutex.Unlock()

		progressMsg = fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d)\nCurrent: %s", processed, total, skipped, added, currentFile)
		if err := s.activityService.UpdateProgress(activity.ID, progress, progressMsg); err != nil {
			log.Printf("Failed to update progress: %v", err)
		}
	}

	// Close thumbnail jobs channel and wait for all workers to finish
	close(thumbnailJobs)
	log.Println("Waiting for thumbnail generation workers to complete...")
	wg.Wait()
	log.Println("All thumbnail generation workers completed")

	// ── Reconciliation pass ──────────────────────────────────────────────────
	// Check every video in this library against the filesystem.
	// Mark files that no longer exist as is_missing=1; clear the flag for any
	// that have come back (e.g. re-mounted network share).
	if err := s.activityService.UpdateProgress(activity.ID, 99, "Reconciling file paths…"); err != nil {
		log.Printf("Failed to update progress: %v", err)
	}

	nowMissing := 0
	nowRecovered := 0
	libRows, err := s.db.Query(`SELECT id, file_path, COALESCE(is_missing,0) FROM videos WHERE library_id = ?`, libraryID)
	if err == nil {
		type libVideo struct {
			id        int64
			path      string
			isMissing bool
		}
		var libVideos []libVideo
		for libRows.Next() {
			var lv libVideo
			if libRows.Scan(&lv.id, &lv.path, &lv.isMissing) == nil {
				libVideos = append(libVideos, lv)
			}
		}
		libRows.Close()

		for _, lv := range libVideos {
			_, statErr := os.Stat(lv.path)
			fileGone := statErr != nil
			if fileGone && !lv.isMissing {
				s.db.Exec(`UPDATE videos SET is_missing = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, lv.id)
				log.Printf("[SCAN] Marked missing: %s", lv.path)
				nowMissing++
			} else if !fileGone && lv.isMissing {
				s.db.Exec(`UPDATE videos SET is_missing = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, lv.id)
				log.Printf("[SCAN] Recovered: %s", lv.path)
				nowRecovered++
			}
		}
	}
	// ─────────────────────────────────────────────────────────────────────────

	// Log scan completion
	consoleLogSvc.LogAPI("info", fmt.Sprintf("Library scan completed: %s", library.Name), map[string]interface{}{
		"library_id":      libraryID,
		"library_name":    library.Name,
		"total_files":     total,
		"videos_added":    added,
		"videos_skipped":  skipped,
		"now_missing":     nowMissing,
		"now_recovered":   nowRecovered,
	})

	// Complete activity
	msg := fmt.Sprintf("Scan complete: %d added, %d skipped", added, skipped)
	if nowMissing > 0 {
		msg += fmt.Sprintf(", %d missing", nowMissing)
	}
	if nowRecovered > 0 {
		msg += fmt.Sprintf(", %d recovered", nowRecovered)
	}
	_ = s.activityService.CompleteTask(int64(activity.ID), msg)

	// Trigger path-based auto-linking (always — even re-scans catch new DB performers)
	if OnScanComplete != nil {
		go OnScanComplete(libraryID, int(added))
	}

	return nil
}

// ScanAllLibrariesParallel scans all libraries in parallel with drive-aware optimization
func (s *VideoService) ScanAllLibrariesParallel(config ParallelScanConfig) error {
	log.Println("Starting parallel library scan with drive-aware optimization...")

	// Initialize console log service
	consoleLogSvc := NewConsoleLogService()

	// Get all libraries
	libraries, err := s.libraryService.GetAll()
	if err != nil {
		consoleLogSvc.LogAPI("error", "Failed to get libraries for parallel scan", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to get libraries: %w", err)
	}

	if len(libraries) == 0 {
		log.Println("No libraries found to scan")
		consoleLogSvc.LogAPI("warning", "No libraries found to scan", nil)
		return nil
	}

	// Log parallel scan start
	consoleLogSvc.LogAPI("info", fmt.Sprintf("Starting parallel scan of %d libraries", len(libraries)), map[string]interface{}{
		"library_count":         len(libraries),
		"server_max_concurrent": config.ServerMaxConcurrent,
		"local_max_concurrent":  config.LocalMaxConcurrent,
	})

	// Create overall activity log for the entire parallel scan
	activity, err := s.activityService.StartTask(
		"library_scan_all",
		fmt.Sprintf("Scanning all %d libraries with drive-aware optimization", len(libraries)),
		map[string]interface{}{
			"library_count":         len(libraries),
			"server_max_concurrent": config.ServerMaxConcurrent,
			"local_max_concurrent":  config.LocalMaxConcurrent,
		},
	)
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	// Group libraries by drive type
	serverLibraries := []models.Library{}
	localLibraries := []models.Library{}

	for _, lib := range libraries {
		if s.isServerDrive(lib.Path, config.ServerDrives) {
			serverLibraries = append(serverLibraries, lib)
		} else if s.isLocalDrive(lib.Path, config.LocalDrives) {
			localLibraries = append(localLibraries, lib)
		} else {
			// Default to local for unknown drives
			localLibraries = append(localLibraries, lib)
		}
	}

	log.Printf("Grouped libraries - Server: %d, Local: %d", len(serverLibraries), len(localLibraries))

	if activity != nil {
		s.activityService.UpdateProgress(
			activity.ID,
			10,
			fmt.Sprintf("Grouped %d libraries (Server: %d, Local: %d)", len(libraries), len(serverLibraries), len(localLibraries)),
		)
	}

	// Create wait group for all scans
	var wg sync.WaitGroup

	// Scan server libraries with limited concurrency
	if len(serverLibraries) > 0 {
		log.Printf("Starting server library scans (max concurrent: %d)...", config.ServerMaxConcurrent)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scanLibrariesConcurrent(serverLibraries, config.ServerMaxConcurrent, "SERVER")
		}()
	}

	// Scan local libraries with higher concurrency
	if len(localLibraries) > 0 {
		log.Printf("Starting local library scans (max concurrent: %d)...", config.LocalMaxConcurrent)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scanLibrariesConcurrent(localLibraries, config.LocalMaxConcurrent, "LOCAL")
		}()
	}

	// Update progress periodically while scans are running
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		progress := 20
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if activity != nil && progress < 90 {
					progress += 10
					s.activityService.UpdateProgress(
						activity.ID,
						progress,
						fmt.Sprintf("Scanning in progress... (%d%%)", progress),
					)
				}
			}
		}
	}()

	// Wait for all scans to complete
	wg.Wait()
	close(done)
	log.Println("All parallel library scans completed")

	// Log parallel scan completion
	consoleLogSvc.LogAPI("info", fmt.Sprintf("Parallel scan completed: %d libraries", len(libraries)), map[string]interface{}{
		"library_count":    len(libraries),
		"server_libraries": len(serverLibraries),
		"local_libraries":  len(localLibraries),
	})

	// Complete activity
	if activity != nil {
		s.activityService.CompleteTask(
			int64(activity.ID),
			fmt.Sprintf("All %d libraries scanned successfully", len(libraries)),
		)
	}

	return nil
}

// scanLibrariesConcurrent scans multiple libraries with controlled concurrency
func (s *VideoService) scanLibrariesConcurrent(libraries []models.Library, maxConcurrent int, driveType string) {
	// Create semaphore to limit concurrency
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, library := range libraries {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(lib models.Library) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			log.Printf("[%s] Scanning library: %s (ID: %d, Path: %s)", driveType, lib.Name, lib.ID, lib.Path)
			startTime := time.Now()

			err := s.ScanLibrary(lib.ID)
			duration := time.Since(startTime)

			if err != nil {
				log.Printf("[%s] Failed to scan library %s: %v (Duration: %s)", driveType, lib.Name, err, duration)
			} else {
				log.Printf("[%s] Successfully scanned library %s (Duration: %s)", driveType, lib.Name, duration)
			}
		}(library)
	}

	wg.Wait()
	log.Printf("[%s] All %d library scans completed", driveType, len(libraries))
}

// isServerDrive checks if a path is on a server drive
func (s *VideoService) isServerDrive(path string, serverDrives []string) bool {
	if len(path) < 2 {
		return false
	}

	// Get drive letter (e.g., "C:", "Z:")
	drive := strings.ToUpper(path[:2])

	for _, serverDrive := range serverDrives {
		if strings.ToUpper(serverDrive) == drive {
			return true
		}
	}

	return false
}

// isLocalDrive checks if a path is on a local drive
func (s *VideoService) isLocalDrive(path string, localDrives []string) bool {
	if len(path) < 2 {
		return false
	}

	// Get drive letter (e.g., "C:", "D:")
	drive := strings.ToUpper(path[:2])

	for _, localDrive := range localDrives {
		if strings.ToUpper(localDrive) == drive {
			return true
		}
	}

	return false
}

// findVideoFiles recursively finds all video files in a directory
func (s *VideoService) findVideoFiles(dir string) ([]string, []string, error) {
	var videoFiles []string
	var skippedPaths []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if path == dir {
				// Root library path itself is inaccessible — fail hard so the caller surfaces a clear error.
				return err
			}
			// Subdirectory or file is inaccessible (e.g. expired SMB session, permission denied).
			// Log and skip rather than aborting the entire scan.
			log.Printf("[SCAN] Skipping inaccessible path %q: %v", path, err)
			skippedPaths = append(skippedPaths, path)
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, supportedExt := range SupportedVideoExtensions {
			if ext == supportedExt {
				videoFiles = append(videoFiles, path)
				break
			}
		}

		return nil
	})

	if len(skippedPaths) > 0 {
		log.Printf("[SCAN] Walk completed with %d inaccessible path(s) skipped in %s", len(skippedPaths), dir)
	}

	return videoFiles, skippedPaths, err
}

// videoExists checks if a video with the given file path already exists
func (s *VideoService) videoExists(filePath string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM videos WHERE file_path = ? LIMIT 1)`
	var exists bool
	err := s.db.QueryRow(query, filePath).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// updateVideoThumbnailPath updates the thumbnail path for a video
func (s *VideoService) updateVideoThumbnailPath(videoID int64, thumbnailPath string) error {
	query := `UPDATE videos SET thumbnail_path = ? WHERE id = ?`
	_, err := s.db.Exec(query, thumbnailPath, videoID)
	if err != nil {
		log.Printf("Failed to update thumbnail path for video %d: %v", videoID, err)
		return err
	}
	return nil
}

// updateVideoPreviewPath updates the preview path for a video
func (s *VideoService) updateVideoPreviewPath(videoID int64, previewPath string) error {
	query := `UPDATE videos SET preview_path = ? WHERE id = ?`
	_, err := s.db.Exec(query, previewPath, videoID)
	if err != nil {
		log.Printf("Failed to update preview path for video %d: %v", videoID, err)
		return err
	}
	return nil
}

// GenerateAllPreviews generates preview storyboards for all videos in all libraries
func (s *VideoService) GenerateAllPreviews(config ParallelScanConfig) error {
	log.Println("Starting preview generation for all videos...")

	// Create activity log
	activity, err := s.activityService.StartTask(
		"preview_generation",
		"Generating preview storyboards for all videos",
		map[string]interface{}{},
	)
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	// Get all libraries
	libraries, err := s.libraryService.GetAll()
	if err != nil {
		if activity != nil {
			s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Failed: %v", err))
		}
		return fmt.Errorf("failed to get libraries: %w", err)
	}

	if len(libraries) == 0 {
		log.Println("No libraries found")
		if activity != nil {
			s.activityService.CompleteTask(int64(activity.ID), "No libraries found")
		}
		return nil
	}

	// Get preview base directory
	previewDir := os.Getenv("PREVIEW_DIR")
	if previewDir == "" {
		previewDir = filepath.Join(appconfig.DataDir(), "previews")
	}

	// Create wait group for all library processing
	var wg sync.WaitGroup

	// Group libraries by drive type
	serverLibraries := []models.Library{}
	localLibraries := []models.Library{}

	for _, lib := range libraries {
		if s.isServerDrive(lib.Path, config.ServerDrives) {
			serverLibraries = append(serverLibraries, lib)
		} else if s.isLocalDrive(lib.Path, config.LocalDrives) {
			localLibraries = append(localLibraries, lib)
		} else {
			localLibraries = append(localLibraries, lib)
		}
	}

	log.Printf("Grouped libraries - Server: %d, Local: %d", len(serverLibraries), len(localLibraries))

	// Track per-library completion for outer activity progress
	totalLibraries := int64(len(libraries))
	var completedLibraries int64
	onLibraryDone := func(libName string) {
		completed := atomic.AddInt64(&completedLibraries, 1)
		if activity != nil {
			progress := int(completed * 100 / totalLibraries)
			s.activityService.UpdateProgressLog(
				int64(activity.ID), progress,
				fmt.Sprintf("Completed %d/%d libraries (last: %s)", completed, totalLibraries, libName),
			)
		}
	}

	// Process server libraries with limited concurrency
	if len(serverLibraries) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.generatePreviewsForLibraries(serverLibraries, config.ServerMaxConcurrent, previewDir, "SERVER", onLibraryDone)
		}()
	}

	// Process local libraries with higher concurrency
	if len(localLibraries) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.generatePreviewsForLibraries(localLibraries, config.LocalMaxConcurrent, previewDir, "LOCAL", onLibraryDone)
		}()
	}

	wg.Wait()
	log.Println("All preview generation completed")

	// Complete activity log
	if activity != nil {
		s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Completed preview generation for %d libraries", len(libraries)))
	}

	return nil
}

// generatePreviewsForLibraries generates previews for all videos in the given libraries with controlled concurrency
func (s *VideoService) generatePreviewsForLibraries(libraries []models.Library, maxConcurrent int, previewDir string, driveType string, onDone func(libName string)) {
	// Create semaphore to limit concurrency
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, library := range libraries {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(lib models.Library) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			log.Printf("[%s] Generating previews for library: %s (ID: %d)", driveType, lib.Name, lib.ID)
			startTime := time.Now()

			err := s.generatePreviewsForLibrary(lib.ID, previewDir)
			duration := time.Since(startTime)

			if err != nil {
				log.Printf("[%s] Failed to generate previews for library %s: %v (Duration: %s)", driveType, lib.Name, err, duration)
			} else {
				log.Printf("[%s] Successfully generated previews for library %s (Duration: %s)", driveType, lib.Name, duration)
			}
			onDone(lib.Name)
		}(library)
	}

	wg.Wait()
	log.Printf("[%s] All %d library preview generations completed", driveType, len(libraries))
}

// generatePreviewsForLibrary generates previews for all videos in a specific library
func (s *VideoService) generatePreviewsForLibrary(libraryID int64, previewDir string) error {
	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return fmt.Errorf("library not found: %w", err)
	}

	// Get all videos in this library
	query := models.VideoSearchQuery{
		LibraryID: libraryID,
		Limit:     10000, // Process all videos
	}
	videos, _, err := s.GetAll(&query)
	if err != nil {
		return fmt.Errorf("failed to get videos: %w", err)
	}

	if len(videos) == 0 {
		log.Printf("No videos found in library %s", library.Name)
		return nil
	}

	log.Printf("Generating previews for %d videos in library: %s", len(videos), library.Name)

	// Create activity log for this library
	activityService := NewActivityService()
	activityLog, err := activityService.CreateLog(&models.ActivityLogCreate{
		TaskType: "preview_generation",
		Status:   models.TaskStatusRunning,
		Message:  fmt.Sprintf("Generating previews for library: %s", library.Name),
		Progress: 0,
		Details:  map[string]interface{}{"library_id": libraryID, "library_name": library.Name, "total_videos": len(videos)},
	})
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	// Create media service
	mediaService := NewMediaService()

	// Setup parallel preview generation with worker pool
	// CONSERVATIVE: Use only 2-4 workers to prevent system overload
	// Each worker runs ffmpeg which is resource-intensive
	numWorkers := 2
	if runtime.NumCPU() >= 8 {
		numWorkers = 3 // Maximum 3 workers even on powerful systems
	}
	log.Printf("Using %d workers for preview generation (CPU cores: %d)", numWorkers, runtime.NumCPU())

	previewJobs := make(chan struct {
		video   models.Video
		library models.Library
	}, numWorkers*2)
	var wg sync.WaitGroup
	var previewMutex sync.Mutex

	generated := 0
	skipped := 0

	// Start preview worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range previewJobs {
				// Skip if preview already exists AND frames are on disk
				if job.video.PreviewPath != "" {
					firstFrame := filepath.Join(previewDir, job.video.PreviewPath, "frame_001.jpg")
					if _, statErr := os.Stat(firstFrame); !os.IsNotExist(statErr) {
						previewMutex.Lock()
						skipped++
						previewMutex.Unlock()
						continue
					}
					// Path set in DB but file missing — regenerate
				}

				// Skip if video has no duration
				if job.video.Duration <= 0 {
					previewMutex.Lock()
					skipped++
					previewMutex.Unlock()
					log.Printf("Worker %d: Skipping video %s - no duration", workerID, job.video.Title)
					continue
				}

				// Generate preview
				previewConfig := PreviewConfig{
					LibraryID:      job.library.ID,
					LibraryPath:    job.library.Path,
					VideoFilePath:  job.video.FilePath,
					Duration:       job.video.Duration,
					PreviewDir:     previewDir,
					FrameCount:     10,
					ThumbnailWidth: 320,
				}

				result, err := mediaService.GeneratePreviewStoryboard(previewConfig)
				if err != nil {
					log.Printf("Worker %d: Failed to generate preview for video ID %d: %v", workerID, job.video.ID, err)
					previewMutex.Lock()
					skipped++
					previewMutex.Unlock()
				} else {
					// Update video preview path in database
					previewMutex.Lock()
					err := s.updateVideoPreviewPath(job.video.ID, result.RelativePath)
					if err == nil {
						generated++
						if generated%10 == 0 {
							log.Printf("Progress: Generated %d previews, skipped %d", generated, skipped)
							// Update activity progress every 10 videos
							if activityLog != nil {
								progress := int((float64(generated+skipped) / float64(len(videos))) * 100)
								activityService.UpdateProgressLog(activityLog.ID, progress,
									fmt.Sprintf("Generated %d/%d previews", generated, len(videos)))
							}
						}
					} else {
						skipped++
					}
					previewMutex.Unlock()
				}

				// Small delay to prevent overwhelming the system
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	// Queue videos for preview generation
	for _, video := range videos {
		previewJobs <- struct {
			video   models.Video
			library models.Library
		}{video: video, library: *library}
	}

	// Close jobs channel and wait for all workers to finish
	close(previewJobs)
	wg.Wait()

	log.Printf("Preview generation complete for library %s: %d generated, %d skipped", library.Name, generated, skipped)

	// Mark activity as complete
	if activityLog != nil {
		activityService.CompleteTaskLog(activityLog.ID,
			fmt.Sprintf("Completed: Generated %d previews, skipped %d", generated, skipped))
	}

	return nil
}

// GenerateAllThumbnails generates thumbnails for all videos that don't have them
func (s *VideoService) GenerateAllThumbnails() error {
	log.Println("Starting batch video thumbnail generation...")

	// Create activity log
	activity, err := s.activityService.StartTask(
		"video_thumbnail_generation",
		"Generating thumbnails for videos without thumbnails",
		map[string]interface{}{},
	)
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	// Get all videos — we'll check thumbnail file existence in Go to also catch
	// records that have a path set but whose file was never actually generated.
	query := `
		SELECT id, library_id, file_path, duration, COALESCE(thumbnail_path,'')
		FROM videos
		ORDER BY id ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		if activity != nil {
			s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Failed: %v", err))
		}
		return fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	// Get the configured thumbnail directory so we can check file existence.
	thumbnailBaseDir := os.Getenv("THUMBNAIL_DIR")
	if thumbnailBaseDir == "" {
		thumbnailBaseDir = filepath.Join(appconfig.DataDir(), "thumbnails")
	}

	type videoInfo struct {
		ID        int64
		LibraryID int64
		FilePath  string
		Duration  float64
	}

	var videos []videoInfo
	for rows.Next() {
		var v videoInfo
		var thumbPath string
		if err := rows.Scan(&v.ID, &v.LibraryID, &v.FilePath, &v.Duration, &thumbPath); err != nil {
			log.Printf("Failed to scan video: %v", err)
			continue
		}
		// Only queue if thumbnail is missing or the file doesn't actually exist on disk.
		if thumbPath == "" {
			videos = append(videos, v)
			continue
		}
		fullThumbPath := filepath.Join(thumbnailBaseDir, strings.TrimPrefix(thumbPath, "thumbnails/"))
		if _, statErr := os.Stat(fullThumbPath); os.IsNotExist(statErr) {
			videos = append(videos, v)
		}
	}

	if len(videos) == 0 {
		log.Println("No videos need thumbnails")
		if activity != nil {
			s.activityService.CompleteTask(int64(activity.ID), "All videos already have thumbnails")
		}
		return nil
	}

	log.Printf("Found %d videos without thumbnails", len(videos))

	// Generate thumbnails in parallel
	numWorkers := runtime.NumCPU()
	thumbnailJobs := make(chan videoInfo, len(videos))
	var wg sync.WaitGroup
	var generated, failed int
	var mu sync.Mutex

	mediaService := NewMediaService()

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for video := range thumbnailJobs {
				// Get library
				library, err := s.libraryService.GetByID(video.LibraryID)
				if err != nil {
					log.Printf("Worker %d: Failed to get library for video %d: %v", workerID, video.ID, err)
					mu.Lock()
					failed++
					mu.Unlock()
					continue
				}

				// Generate thumbnail
				config := ThumbnailConfig{
					LibraryID:     video.LibraryID,
					LibraryPath:   library.Path,
					VideoFilePath: video.FilePath,
					Duration:      video.Duration,
					ThumbnailDir:  thumbnailBaseDir,
				}

				result, err := mediaService.GenerateThumbnailHierarchical(config)
				if err != nil {
					log.Printf("Worker %d: Failed to generate thumbnail for video %d: %v", workerID, video.ID, err)
					mu.Lock()
					failed++
					mu.Unlock()
					continue
				}

				// Update video thumbnail path
				if err := s.updateVideoThumbnailPath(video.ID, result.RelativePath); err != nil {
					log.Printf("Worker %d: Failed to update thumbnail path for video %d: %v", workerID, video.ID, err)
					mu.Lock()
					failed++
					mu.Unlock()
					continue
				}

				mu.Lock()
				generated++
				if generated%10 == 0 {
					log.Printf("Progress: %d/%d thumbnails generated, %d failed", generated, len(videos), failed)
					if activity != nil {
						progress := int((float64(generated+failed) / float64(len(videos))) * 100)
						s.activityService.UpdateProgress(
							activity.ID,
							progress,
							fmt.Sprintf("Generated %d/%d thumbnails (%d failed)", generated, len(videos), failed),
						)
					}
				}
				mu.Unlock()
			}
		}(i)
	}

	// Queue all videos, stopping early if the task is cancelled
	for _, video := range videos {
		if activity != nil && IsActivityPaused(int64(activity.ID)) {
			log.Printf("Thumbnail generation cancelled/paused, stopping queue")
			break
		}
		thumbnailJobs <- video
	}

	close(thumbnailJobs)
	wg.Wait()

	log.Printf("Thumbnail generation complete: %d generated, %d failed", generated, failed)

	if activity != nil {
		s.activityService.CompleteTask(
			int64(activity.ID),
			fmt.Sprintf("Completed: %d thumbnails generated, %d failed", generated, failed),
		)
	}

	return nil
}

// VideoMarks holds marking information for a video
type VideoMarks struct {
	ID            int64
	NotInterested bool
	InEditList    bool
	ContentType   string
	Lists         []VideoListInfo
}

// VideoListInfo holds basic info about a list a video belongs to
type VideoListInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

// GetVideoMarksByPath retrieves video marks (not_interested, in_edit_list, content_type) by file path
func (s *VideoService) GetVideoMarksByPath(filePath string) (*VideoMarks, error) {
	var marks VideoMarks
	var contentType sql.NullString
	query := `SELECT not_interested, in_edit_list, content_type FROM videos WHERE file_path = ?`
	err := s.db.QueryRow(query, filePath).Scan(&marks.NotInterested, &marks.InEditList, &contentType)
	if err == sql.ErrNoRows {
		// Video doesn't exist in database yet, return nil (no marks)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get video marks: %w", err)
	}
	if contentType.Valid {
		marks.ContentType = contentType.String
	}
	return &marks, nil
}

// GetVideoMarksBatch retrieves video marks for multiple file paths in a single query
func (s *VideoService) GetVideoMarksBatch(filePaths []string) (map[string]*VideoMarks, error) {
	if len(filePaths) == 0 {
		return make(map[string]*VideoMarks), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(filePaths))
	args := make([]interface{}, len(filePaths))
	for i, path := range filePaths {
		placeholders[i] = "?"
		args[i] = path
	}
	placeholderStr := strings.Join(placeholders, ",")

	query := fmt.Sprintf(`
		SELECT id, file_path, not_interested, in_edit_list, content_type
		FROM videos
		WHERE file_path IN (%s)
	`, placeholderStr)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query video marks: %w", err)
	}
	defer rows.Close()

	marksMap := make(map[string]*VideoMarks)
	var videoIDs []int64
	videoIDToPath := make(map[int64]string)

	for rows.Next() {
		var filePath string
		var marks VideoMarks
		var contentType sql.NullString
		if err := rows.Scan(&marks.ID, &filePath, &marks.NotInterested, &marks.InEditList, &contentType); err != nil {
			log.Printf("Warning: Failed to scan video marks: %v", err)
			continue
		}
		if contentType.Valid {
			marks.ContentType = contentType.String
		}
		marksMap[filePath] = &marks
		videoIDs = append(videoIDs, marks.ID)
		videoIDToPath[marks.ID] = filePath
	}

	// Load list membership for all videos in one query
	if len(videoIDs) > 0 {
		// Build proper placeholders for video IDs
		idPlaceholders := make([]string, len(videoIDs))
		listArgs := make([]interface{}, len(videoIDs))
		for i, id := range videoIDs {
			idPlaceholders[i] = "?"
			listArgs[i] = id
		}
		idPlaceholderStr := strings.Join(idPlaceholders, ",")

		listQuery := `
			SELECT vli.video_id, vl.id, vl.name, vl.color, vl.icon
			FROM video_list_items vli
			INNER JOIN video_lists vl ON vli.list_id = vl.id
			WHERE vli.video_id IN (` + idPlaceholderStr + `)
			ORDER BY vl.position ASC
		`

		listRows, err := s.db.Query(listQuery, listArgs...)
		if err != nil {
			log.Printf("Warning: Failed to query video list membership: %v", err)
		} else {
			defer listRows.Close()
			for listRows.Next() {
				var videoID int64
				var listInfo VideoListInfo
				if err := listRows.Scan(&videoID, &listInfo.ID, &listInfo.Name, &listInfo.Color, &listInfo.Icon); err != nil {
					log.Printf("Warning: Failed to scan list info: %v", err)
					continue
				}
				if filePath, ok := videoIDToPath[videoID]; ok {
					if marks, ok := marksMap[filePath]; ok {
						marks.Lists = append(marks.Lists, listInfo)
					}
				}
			}
		}
	}

	return marksMap, nil
}

// GetByFilePath retrieves a video by file path
func (s *VideoService) GetByFilePath(filePath string) (*models.Video, error) {
	var video models.Video
	var date, description, contentType sql.NullString
	var lastPlayedAt sql.NullTime

	query := `SELECT id, library_id, title, file_path, file_size, duration, codec, resolution,
	          bitrate, fps, thumbnail_path, date, rating, description, is_favorite, is_pinned,
	          not_interested, in_edit_list, content_type, created_at, updated_at, last_played_at, play_count
	          FROM videos WHERE file_path = ?`

	err := s.db.QueryRow(query, filePath).Scan(
		&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
		&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath,
		&date, &video.Rating, &description, &video.IsFavorite, &video.IsPinned,
		&video.NotInterested, &video.InEditList, &contentType, &video.CreatedAt, &video.UpdatedAt,
		&lastPlayedAt, &video.PlayCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("video not found with file path: %s", filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get video by file path: %w", err)
	}

	// Handle nullable fields
	if date.Valid {
		video.Date = date.String
	}
	if description.Valid {
		video.Description = description.String
	}
	if lastPlayedAt.Valid {
		video.LastPlayedAt = &lastPlayedAt.Time
	}
	if contentType.Valid {
		video.ContentType = contentType.String
	}

	// Load related data
	if err := s.loadVideoRelationships(&video); err != nil {
		return nil, err
	}

	return &video, nil
}

// loadVideoRelationshipsBatch loads related data for multiple videos in batch
func (s *VideoService) loadVideoRelationshipsBatch(videos []models.Video) error {
	if len(videos) == 0 {
		return nil
	}

	// Collect all video IDs
	videoIDs := make([]int64, len(videos))
	videoMap := make(map[int64]*models.Video)
	for i := range videos {
		videoIDs[i] = videos[i].ID
		videoMap[videos[i].ID] = &videos[i]
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(videoIDs))
	args := make([]interface{}, len(videoIDs))
	for i, id := range videoIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	placeholderStr := strings.Join(placeholders, ",")

	// Batch load performers
	performerQuery := fmt.Sprintf(`
		SELECT vp.video_id, p.id, p.name, p.preview_path, p.folder_path, p.video_count, p.category, p.metadata, p.created_at, p.updated_at
		FROM performers p
		INNER JOIN video_performers vp ON p.id = vp.performer_id
		WHERE vp.video_id IN (%s)
		ORDER BY vp.video_id, p.name
	`, placeholderStr)

	performerRows, err := s.db.Query(performerQuery, args...)
	if err == nil {
		defer func() {
			if err := performerRows.Close(); err != nil {
				log.Printf("failed to close performerRows: %v", err)
			}
		}()
		for performerRows.Next() {
			var videoID int64
			var performer models.Performer
			var metadata sql.NullString
			err := performerRows.Scan(&videoID, &performer.ID, &performer.Name, &performer.PreviewPath, &performer.FolderPath, &performer.VideoCount, &performer.Category, &metadata, &performer.CreatedAt, &performer.UpdatedAt)
			if err == nil {
				if metadata.Valid {
					performer.Metadata = metadata.String
					// Unmarshal metadata for frontend use
					performer.UnmarshalMetadata()
				}

				if video, exists := videoMap[videoID]; exists {
					video.Performers = append(video.Performers, performer)
				}
			}
		}
	}

	// Batch load tags
	tagQuery := fmt.Sprintf(`
		SELECT vt.video_id, t.id, t.name, t.color, t.icon, t.created_at, t.updated_at
		FROM tags t
		INNER JOIN video_tags vt ON t.id = vt.tag_id
		WHERE vt.video_id IN (%s)
		ORDER BY vt.video_id, t.name
	`, placeholderStr)

	tagRows, err := s.db.Query(tagQuery, args...)
	if err == nil {
		defer func() {
			if err := tagRows.Close(); err != nil {
				log.Printf("failed to close tagRows: %v", err)
			}
		}()
		for tagRows.Next() {
			var videoID int64
			var tag models.Tag
			err := tagRows.Scan(&videoID, &tag.ID, &tag.Name, &tag.Color, &tag.Icon, &tag.CreatedAt, &tag.UpdatedAt)
			if err == nil {
				if video, exists := videoMap[videoID]; exists {
					video.Tags = append(video.Tags, tag)
				}
			}
		}
	}

	// Batch load studios
	studioQuery := fmt.Sprintf(`
		SELECT vs.video_id, s.id, s.name, s.logo_path, s.description, s.founded_date, s.country, s.metadata, s.created_at, s.updated_at
		FROM studios s
		INNER JOIN video_studios vs ON s.id = vs.studio_id
		WHERE vs.video_id IN (%s)
		ORDER BY vs.video_id, s.name
	`, placeholderStr)

	studioRows, err := s.db.Query(studioQuery, args...)
	if err == nil {
		defer func() {
			if err := studioRows.Close(); err != nil {
				log.Printf("failed to close studioRows: %v", err)
			}
		}()
		for studioRows.Next() {
			var videoID int64
			var studio models.Studio
			var metadata, foundedDate sql.NullString
			err := studioRows.Scan(&videoID, &studio.ID, &studio.Name, &studio.LogoPath, &studio.Description, &foundedDate, &studio.Country, &metadata, &studio.CreatedAt, &studio.UpdatedAt)
			if err == nil {
				if metadata.Valid {
					studio.Metadata = metadata.String
				}
				if foundedDate.Valid {
					studio.FoundedDate = foundedDate.String
				}
				if video, exists := videoMap[videoID]; exists {
					video.Studios = append(video.Studios, studio)
				}
			}
		}
	}

	// Batch load groups
	groupQuery := fmt.Sprintf(`
		SELECT vg.video_id, g.id, g.studio_id, g.name, g.logo_path, g.description, g.metadata, g.created_at, g.updated_at
		FROM groups g
		INNER JOIN video_groups vg ON g.id = vg.group_id
		WHERE vg.video_id IN (%s)
		ORDER BY vg.video_id, g.name
	`, placeholderStr)

	groupRows, err := s.db.Query(groupQuery, args...)
	if err == nil {
		defer func() {
			if err := groupRows.Close(); err != nil {
				log.Printf("failed to close groupRows: %v", err)
			}
		}()
		for groupRows.Next() {
			var videoID int64
			var group models.Group
			var metadata sql.NullString
			err := groupRows.Scan(&videoID, &group.ID, &group.StudioID, &group.Name, &group.LogoPath, &group.Description, &metadata, &group.CreatedAt, &group.UpdatedAt)
			if err == nil {
				if metadata.Valid {
					group.Metadata = metadata.String
				}
				if video, exists := videoMap[videoID]; exists {
					video.Groups = append(video.Groups, group)
				}
			}
		}
	}

	return nil
}

// loadVideoRelationships loads related data for a video
func (s *VideoService) loadVideoRelationships(video *models.Video) error {
	// Load performers
	performerQuery := `
		SELECT p.id, p.name, p.preview_path, p.folder_path, p.video_count, p.category, p.metadata, p.created_at, p.updated_at
		FROM performers p
		INNER JOIN video_performers vp ON p.id = vp.performer_id
		WHERE vp.video_id = ?
	`
	performerRows, err := s.db.Query(performerQuery, video.ID)
	if err == nil {
		defer func() {
			if err := performerRows.Close(); err != nil {
				log.Printf("failed to close performerRows: %v", err)
			}
		}()
		for performerRows.Next() {
			var performer models.Performer
			var metadata sql.NullString
			err := performerRows.Scan(&performer.ID, &performer.Name, &performer.PreviewPath, &performer.FolderPath, &performer.VideoCount, &performer.Category, &metadata, &performer.CreatedAt, &performer.UpdatedAt)
			if err == nil {
				if metadata.Valid {
					performer.Metadata = metadata.String
					// Unmarshal metadata for frontend use
					performer.UnmarshalMetadata()
				}
				video.Performers = append(video.Performers, performer)
			}
		}
	}

	// Load tags
	tagQuery := `
		SELECT t.id, t.name, t.color, t.icon, t.created_at, t.updated_at
		FROM tags t
		INNER JOIN video_tags vt ON t.id = vt.tag_id
		WHERE vt.video_id = ?
	`
	tagRows, err := s.db.Query(tagQuery, video.ID)
	if err == nil {
		defer func() {
			if err := tagRows.Close(); err != nil {
				log.Printf("failed to close tagRows: %v", err)
			}
		}()
		for tagRows.Next() {
			var tag models.Tag
			err := tagRows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Icon, &tag.CreatedAt, &tag.UpdatedAt)
			if err == nil {
				video.Tags = append(video.Tags, tag)
			}
		}
	}

	// Load studios
	studioQuery := `
		SELECT s.id, s.name, s.logo_path, s.description, s.founded_date, s.country, s.metadata, s.created_at, s.updated_at
		FROM studios s
		INNER JOIN video_studios vs ON s.id = vs.studio_id
		WHERE vs.video_id = ?
	`
	studioRows, err := s.db.Query(studioQuery, video.ID)
	if err == nil {
		defer func() {
			if err := studioRows.Close(); err != nil {
				log.Printf("failed to close studioRows: %v", err)
			}
		}()
		for studioRows.Next() {
			var studio models.Studio
			var metadata, foundedDate sql.NullString
			err := studioRows.Scan(&studio.ID, &studio.Name, &studio.LogoPath, &studio.Description, &foundedDate, &studio.Country, &metadata, &studio.CreatedAt, &studio.UpdatedAt)
			if err == nil {
				if metadata.Valid {
					studio.Metadata = metadata.String
				}
				if foundedDate.Valid {
					studio.FoundedDate = foundedDate.String
				}
				video.Studios = append(video.Studios, studio)
			}
		}
	}

	// Load groups
	groupQuery := `
		SELECT g.id, g.studio_id, g.name, g.logo_path, g.description, g.metadata, g.created_at, g.updated_at
		FROM groups g
		INNER JOIN video_groups vg ON g.id = vg.group_id
		WHERE vg.video_id = ?
	`
	groupRows, err := s.db.Query(groupQuery, video.ID)
	if err == nil {
		defer func() {
			if err := groupRows.Close(); err != nil {
				log.Printf("failed to close groupRows: %v", err)
			}
		}()
		for groupRows.Next() {
			var group models.Group
			var metadata sql.NullString
			err := groupRows.Scan(&group.ID, &group.StudioID, &group.Name, &group.LogoPath, &group.Description, &metadata, &group.CreatedAt, &group.UpdatedAt)
			if err == nil {
				if metadata.Valid {
					group.Metadata = metadata.String
				}
				video.Groups = append(video.Groups, group)
			}
		}
	}

	return nil
}

// UpdateConversionLink updates the conversion link for a video
func (s *VideoService) UpdateConversionLink(videoID int64, linkedVideoID int64, isOriginal bool) error {
	var column string
	if isOriginal {
		column = "converted_to"
	} else {
		column = "converted_from"
	}

	query := fmt.Sprintf("UPDATE videos SET %s = ?, updated_at = ? WHERE id = ?", column)
	_, err := s.db.Exec(query, linkedVideoID, time.Now(), videoID)
	return err
}

// CopyPerformers copies performers from one video to another
func (s *VideoService) CopyPerformers(fromVideoID, toVideoID int64) error {
	query := `
		INSERT INTO video_performers (video_id, performer_id)
		SELECT ?, performer_id FROM video_performers WHERE video_id = ?
	`
	_, err := s.db.Exec(query, toVideoID, fromVideoID)
	return err
}

// CopyTags copies tags from one video to another
func (s *VideoService) CopyTags(fromVideoID, toVideoID int64) error {
	query := `
		INSERT INTO video_tags (video_id, tag_id)
		SELECT ?, tag_id FROM video_tags WHERE video_id = ?
	`
	_, err := s.db.Exec(query, toVideoID, fromVideoID)
	return err
}

// CopyStudios copies studios from one video to another
func (s *VideoService) CopyStudios(fromVideoID, toVideoID int64) error {
	query := `
		INSERT INTO video_studios (video_id, studio_id)
		SELECT ?, studio_id FROM video_studios WHERE video_id = ?
	`
	_, err := s.db.Exec(query, toVideoID, fromVideoID)
	return err
}

// CopyGroups copies groups from one video to another
func (s *VideoService) CopyGroups(fromVideoID, toVideoID int64) error {
	query := `
		INSERT INTO video_groups (video_id, group_id)
		SELECT ?, group_id FROM video_groups WHERE video_id = ?
	`
	_, err := s.db.Exec(query, toVideoID, fromVideoID)
	return err
}

// CopyMetadata copies metadata fields from one video to another
func (s *VideoService) CopyMetadata(fromVideoID, toVideoID int64) error {
	query := `
		UPDATE videos
		SET date = (SELECT date FROM videos WHERE id = ?),
		    rating = (SELECT rating FROM videos WHERE id = ?),
		    description = (SELECT description FROM videos WHERE id = ?),
		    updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.Exec(query, fromVideoID, fromVideoID, fromVideoID, time.Now(), toVideoID)
	return err
}

// UpdateContentTypeByFolder updates the content_type for all videos in a folder recursively
func (s *VideoService) UpdateContentTypeByFolder(folderPath string, contentType string) (int64, error) {
	// Use LIKE to match all videos whose file_path starts with the folder path
	// Ensure folder path ends with separator for proper matching
	searchPath := folderPath
	if !strings.HasSuffix(searchPath, "\\") && !strings.HasSuffix(searchPath, "/") {
		searchPath += "\\"
	}

	log.Printf("UpdateContentTypeByFolder: folderPath=%s, searchPath=%s, pattern=%s", folderPath, searchPath, searchPath+"%")

	query := `
		UPDATE videos
		SET content_type = ?, updated_at = ?
		WHERE file_path LIKE ?
	`

	result, err := s.db.Exec(query, contentType, time.Now(), searchPath+"%")
	if err != nil {
		log.Printf("UpdateContentTypeByFolder: error executing query: %v", err)
		return 0, fmt.Errorf("failed to update folder content type: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	log.Printf("UpdateContentTypeByFolder: updated %d videos with content_type=%s", rowsAffected, contentType)

	return rowsAffected, nil
}

// LinkPerformerToFolderVideos links all videos in a folder (recursively) to a performer
func (s *VideoService) LinkPerformerToFolderVideos(folderPath string, performerID int64) (int64, error) {
	searchPath := folderPath
	if !strings.HasSuffix(searchPath, "\\") && !strings.HasSuffix(searchPath, "/") {
		searchPath += "\\"
	}

	// Insert relationships for all folder videos that don't already have this performer
	query := `
		INSERT INTO video_performers (video_id, performer_id)
		SELECT v.id, ?
		FROM videos v
		WHERE v.file_path LIKE ?
		AND NOT EXISTS (
			SELECT 1 FROM video_performers vp
			WHERE vp.video_id = v.id AND vp.performer_id = ?
		)
	`
	result, err := s.db.Exec(query, performerID, searchPath+"%", performerID)
	if err != nil {
		return 0, fmt.Errorf("failed to link performer to folder: %w", err)
	}

	linked, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	// Update performer video count
	s.db.Exec(`UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = ?) WHERE id = ?`, performerID, performerID)

	log.Printf("LinkPerformerToFolderVideos: linked performer %d to %d videos in folder %s", performerID, linked, folderPath)
	return linked, nil
}

// LocateFile searches the video's library directory for files matching the video's filename.
// Returns up to 10 candidate paths (absolute) that share the same base filename (case-insensitive).
func (s *VideoService) LocateFile(videoID int64) ([]string, error) {
	// Load video and its library path
	var filePath string
	var libraryID int64
	err := s.db.QueryRow(`SELECT file_path, library_id FROM videos WHERE id = ?`, videoID).Scan(&filePath, &libraryID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}

	var libraryPath string
	err = s.db.QueryRow(`SELECT path FROM libraries WHERE id = ?`, libraryID).Scan(&libraryPath)
	if err != nil {
		return nil, fmt.Errorf("library not found: %w", err)
	}

	targetName := strings.ToLower(filepath.Base(filePath))
	var candidates []string
	const maxResults = 10

	_ = filepath.WalkDir(libraryPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if len(candidates) >= maxResults {
			return filepath.SkipAll
		}
		if strings.ToLower(d.Name()) == targetName {
			candidates = append(candidates, path)
		}
		return nil
	})

	return candidates, nil
}

// RemapVideoPath updates a video's file_path and clears is_missing.
// Returns an error if newPath does not exist on disk.
func (s *VideoService) RemapVideoPath(videoID int64, newPath string) error {
	if _, err := os.Stat(newPath); err != nil {
		return fmt.Errorf("file not found at new path: %w", err)
	}
	info, _ := os.Stat(newPath)
	newSize := info.Size()

	_, err := s.db.Exec(
		`UPDATE videos SET file_path = ?, file_size = ?, is_missing = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		newPath, newSize, videoID,
	)
	if err != nil {
		return fmt.Errorf("failed to remap video path: %w", err)
	}
	log.Printf("[MISSING] Remapped video %d → %s", videoID, newPath)
	return nil
}

// DeleteMissingVideos bulk-deletes all is_missing=1 videos.
// If libraryID > 0, only that library is affected.
// Returns the count of deleted records.
func (s *VideoService) DeleteMissingVideos(libraryID int64) (int64, error) {
	var result interface {
		RowsAffected() (int64, error)
	}
	var err error

	if libraryID > 0 {
		result, err = s.db.Exec(
			`DELETE FROM videos WHERE COALESCE(is_missing, 0) = 1 AND library_id = ?`, libraryID,
		)
	} else {
		result, err = s.db.Exec(
			`DELETE FROM videos WHERE COALESCE(is_missing, 0) = 1`,
		)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to delete missing videos: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	log.Printf("[MISSING] Deleted %d missing video records (library_id=%d)", count, libraryID)
	return count, nil
}
