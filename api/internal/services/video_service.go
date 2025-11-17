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
	"time"

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
		db:               database.DB,
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

// GetAll retrieves all videos with optional filters
func (s *VideoService) GetAll(query *models.VideoSearchQuery) ([]models.Video, int, error) {
	// Build base query - include library_id in SELECT
	baseQuery := `
		SELECT DISTINCT v.id, v.library_id, v.title, v.file_path, v.file_size, v.duration, v.codec,
		       v.resolution, v.bitrate, v.fps, v.thumbnail_path, v.date, v.rating, v.description,
		       v.is_favorite, v.is_pinned, v.not_interested, v.in_edit_list, v.created_at, v.updated_at, v.last_played_at, v.play_count
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
		conditions = append(conditions, "v.title LIKE ?")
		args = append(args, "%"+query.Query+"%")
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

	// Zoo content filter (checks if video has any performers with zoo = true)
	if query.Zoo != nil {
		joins = append(joins, "INNER JOIN video_performers vp2 ON v.id = vp2.video_id")
		joins = append(joins, "INNER JOIN performers p ON vp2.performer_id = p.id")
		if *query.Zoo {
			// Show only zoo content
			conditions = append(conditions, "p.zoo = 1")
		} else {
			// Show only non-zoo content (no performers with zoo = true)
			conditions = append(conditions, "p.zoo = 0")
		}
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
		err := rows.Scan(
			&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
			&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath,
			&date, &video.Rating, &description, &video.IsFavorite, &video.IsPinned, &video.NotInterested, &video.InEditList,
			&video.CreatedAt, &video.UpdatedAt, &lastPlayedAt, &video.PlayCount,
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

		videos = append(videos, video)
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
		return nil, 0, fmt.Errorf("error iterating videos: %w", err)
	}

	// Load relationships for each video
	for i := range videos {
		if err := s.loadVideoRelationships(&videos[i]); err != nil {
			log.Printf("Warning: Failed to load relationships for video %d: %v", videos[i].ID, err)
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

	query := `
		SELECT id, library_id, title, file_path, file_size, duration, codec, resolution,
		       bitrate, fps, thumbnail_path, date, rating, description, is_favorite, is_pinned,
		       not_interested, in_edit_list, created_at, updated_at, last_played_at, play_count
		FROM videos
		WHERE id = ?
	`

	err := s.db.QueryRow(query, id).Scan(
		&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
		&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath,
		&date, &video.Rating, &description, &video.IsFavorite, &video.IsPinned, &video.NotInterested, &video.InEditList,
		&video.CreatedAt, &video.UpdatedAt, &lastPlayedAt, &video.PlayCount,
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

	// Load relationships
	if err := s.loadVideoRelationships(&video); err != nil {
		log.Printf("Warning: Failed to load relationships for video %d: %v", video.ID, err)
	}

	return &video, nil
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

	video.UpdatedAt = time.Now()

	query := `
		UPDATE videos
		SET title = ?, date = ?, rating = ?, description = ?, is_favorite = ?, is_pinned = ?,
		    play_count = ?, not_interested = ?, in_edit_list = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(query, video.Title, video.Date, video.Rating, video.Description,
		video.IsFavorite, video.IsPinned, video.PlayCount, video.NotInterested, video.InEditList, video.UpdatedAt, id)
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
		// Clear existing performers
		if _, err := s.db.Exec("DELETE FROM video_performers WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear performers: %w", err)
		}
		// Add new performers
		for _, performerID := range update.PerformerIDs {
			if _, err := s.db.Exec("INSERT INTO video_performers (video_id, performer_id) VALUES (?, ?)",
				id, performerID); err != nil {
				return nil, fmt.Errorf("failed to add performer: %w", err)
			}

			// Auto-apply performer's master tags to this video
			if s.performerService != nil {
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
	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return fmt.Errorf("library not found: %w", err)
	}

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
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	// Scan for video files
	videoFiles, err := s.findVideoFiles(library.Path)
	if err != nil {
		if err := s.activityService.FailTask(activity.ID, fmt.Sprintf("Failed to scan directory: %v", err)); err != nil {
			log.Printf("Failed to fail task: %v", err)
		}
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	total := len(videoFiles)
	if err := s.activityService.UpdateProgress(activity.ID, 0, fmt.Sprintf("Found %d video files", total)); err != nil {
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
		thumbnailDir = filepath.Join("assets", "thumbnails")
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

		expectedThumbnail := mediaService.GetThumbnailPath(thumbnailConfig)
		thumbnailPath := ""
		if expectedThumbnail != nil {
			thumbnailPath = expectedThumbnail.RelativePath
		}

		// Create video record with library ID and thumbnail path
		create := &models.VideoCreate{
			LibraryID:     libraryID,
			Title:         filepath.Base(filePath),
			FilePath:      filePath,
			FileSize:      metadata.Size,
			Duration:      metadata.Duration,
			Codec:         metadata.Codec,
			Resolution:    resolution,
			Bitrate:       metadata.Bitrate,
			FPS:           metadata.FrameRate,
			ThumbnailPath: thumbnailPath,
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

	// Complete activity
	_ = s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Scan complete: %d videos added, %d skipped", added, skipped))

	return nil
}

// findVideoFiles recursively finds all video files in a directory
func (s *VideoService) findVideoFiles(dir string) ([]string, error) {
	var videoFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
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

	return videoFiles, err
}

// videoExists checks if a video with the given file path already exists
func (s *VideoService) videoExists(filePath string) (bool, error) {
	query := `SELECT COUNT(*) FROM videos WHERE file_path = ?`
	var count int
	err := s.db.QueryRow(query, filePath).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
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

// VideoMarks holds marking information for a video
type VideoMarks struct {
	NotInterested bool
	InEditList    bool
}

// GetVideoMarksByPath retrieves video marks (not_interested, in_edit_list) by file path
func (s *VideoService) GetVideoMarksByPath(filePath string) (*VideoMarks, error) {
	var marks VideoMarks
	query := `SELECT not_interested, in_edit_list FROM videos WHERE file_path = ?`
	err := s.db.QueryRow(query, filePath).Scan(&marks.NotInterested, &marks.InEditList)
	if err == sql.ErrNoRows {
		// Video doesn't exist in database yet, return nil (no marks)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get video marks: %w", err)
	}
	return &marks, nil
}

// GetByFilePath retrieves a video by file path
func (s *VideoService) GetByFilePath(filePath string) (*models.Video, error) {
	var video models.Video
	var date, description sql.NullString
	var lastPlayedAt sql.NullTime

	query := `SELECT id, library_id, title, file_path, file_size, duration, codec, resolution,
	          bitrate, fps, thumbnail_path, date, rating, description, is_favorite, is_pinned,
	          not_interested, in_edit_list, created_at, updated_at, last_played_at, play_count
	          FROM videos WHERE file_path = ?`

	err := s.db.QueryRow(query, filePath).Scan(
		&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
		&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath,
		&date, &video.Rating, &description, &video.IsFavorite, &video.IsPinned,
		&video.NotInterested, &video.InEditList, &video.CreatedAt, &video.UpdatedAt,
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

	// Load related data
	if err := s.loadVideoRelationships(&video); err != nil {
		return nil, err
	}

	return &video, nil
}

// loadVideoRelationships loads related data for a video
func (s *VideoService) loadVideoRelationships(video *models.Video) error {
	// Load performers
	performerQuery := `
		SELECT p.id, p.name, p.preview_path, p.folder_path, p.scene_count, p.zoo, p.metadata, p.created_at, p.updated_at
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
			var zooVal interface{}
			err := performerRows.Scan(&performer.ID, &performer.Name, &performer.PreviewPath, &performer.FolderPath, &performer.SceneCount, &zooVal, &metadata, &performer.CreatedAt, &performer.UpdatedAt)
			if err == nil {
				// Convert to bool - handle both int64 and bool types
				switch v := zooVal.(type) {
				case int64:
					performer.Zoo = v != 0
				case bool:
					performer.Zoo = v
				default:
					performer.Zoo = false
				}

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
