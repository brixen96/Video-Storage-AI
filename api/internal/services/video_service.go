package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// VideoService handles video-related business logic
type VideoService struct {
	db              *sql.DB
	activityService *ActivityService
	libraryService  *LibraryService
}

// NewVideoService creates a new video service
func NewVideoService(activityService *ActivityService, libraryService *LibraryService) *VideoService {
	return &VideoService{
		db:              database.DB,
		activityService: activityService,
		libraryService:  libraryService,
	}
}

// SupportedVideoExtensions lists supported video file extensions
var SupportedVideoExtensions = []string{
	".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".mpg", ".mpeg",
}

// GetAll retrieves all videos with optional filters
func (s *VideoService) GetAll(query *models.VideoSearchQuery) ([]models.Video, int, error) {
	// Build base query - include library_id in SELECT
	baseQuery := `
		SELECT DISTINCT v.id, v.library_id, v.title, v.file_path, v.file_size, v.duration, v.codec,
		       v.resolution, v.bitrate, v.fps, v.thumbnail_path,
		       v.created_at, v.updated_at, v.last_played_at, v.play_count
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
		err := rows.Scan(
			&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
			&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath,
			&video.CreatedAt, &video.UpdatedAt, &lastPlayedAt, &video.PlayCount,
		)
		if err != nil {
			log.Printf("Failed to scan video row: %v", err)
			return nil, 0, fmt.Errorf("failed to scan video: %w", err)
		}

		if lastPlayedAt.Valid {
			video.LastPlayedAt = &lastPlayedAt.Time
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

	query := `
		SELECT id, library_id, title, file_path, file_size, duration, codec, resolution,
		       bitrate, fps, thumbnail_path, created_at, updated_at, last_played_at, play_count
		FROM videos
		WHERE id = ?
	`

	err := s.db.QueryRow(query, id).Scan(
		&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
		&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath,
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

	if update.Title != nil {
		video.Title = *update.Title
	}
	if update.ThumbnailPath != nil {
		video.ThumbnailPath = *update.ThumbnailPath
	}
	if update.PlayCount != nil {
		video.PlayCount = *update.PlayCount
	}

	video.UpdatedAt = time.Now()

	query := `
		UPDATE videos
		SET title = ?, thumbnail_path = ?, play_count = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(query, video.Title, video.ThumbnailPath, video.PlayCount, video.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	return video, nil
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

	for _, filePath := range videoFiles {
		processed++
		progress := int((float64(processed) / float64(total)) * 100)

		// Check if video already exists
		exists, err := s.videoExists(filePath)
		if err == nil && exists {
			skipped++
			if err := s.activityService.UpdateProgress(activity.ID, progress, fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d)", processed, total, skipped, added)); err != nil {
				log.Printf("Failed to update progress: %v", err)
			}
			continue
		}

		// Get file size
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			skipped++
			if err := s.activityService.UpdateProgress(activity.ID, progress, fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d) - Error: %v", processed, total, skipped, added, err)); err != nil {
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

		// Create video record with library ID
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

		_, err = s.Create(create)
		if err != nil {
			skipped++
			if err := s.activityService.UpdateProgress(activity.ID, progress, fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d) - Error: %v", processed, total, skipped, added, err)); err != nil {
				log.Printf("Failed to update progress: %v", err)
			}
			continue
		}

		added++
		if err := s.activityService.UpdateProgress(activity.ID, progress, fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d)", processed, total, skipped, added)); err != nil {
			log.Printf("Failed to update progress: %v", err)
		}
	}

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

// loadVideoRelationships loads related data for a video
func (s *VideoService) loadVideoRelationships(video *models.Video) error {
	// Load performers
	performerQuery := `
		SELECT p.id, p.name, p.preview_path, p.folder_path, p.scene_count, p.metadata, p.created_at, p.updated_at
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
			err := performerRows.Scan(&performer.ID, &performer.Name, &performer.PreviewPath, &performer.FolderPath, &performer.SceneCount, &metadata, &performer.CreatedAt, &performer.UpdatedAt)
			if err == nil {
				if metadata.Valid {
					performer.Metadata = metadata.String
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
