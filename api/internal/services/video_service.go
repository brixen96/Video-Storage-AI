package services

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// VideoService handles video CRUD, search, and relationship loading.
// Scan, thumbnail, preview, metadata-bulk, and missing-file operations
// live in their respective video_service_*.go files in this package.
type VideoService struct {
	db               *sql.DB
	activityService  *ActivityService
	libraryService   *LibraryService
	performerService *PerformerService
}

// NewVideoService creates a new VideoService.
func NewVideoService(activityService *ActivityService, libraryService *LibraryService, performerService *PerformerService) *VideoService {
	return &VideoService{
		db:               database.GetDB(),
		activityService:  activityService,
		libraryService:   libraryService,
		performerService: performerService,
	}
}

// SupportedVideoExtensions is the list of file extensions treated as video files during scans.
var SupportedVideoExtensions = []string{
	".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".mpg", ".mpeg",
}

// GetAll retrieves videos matching the given search/filter query with pagination.
func (s *VideoService) GetAll(query *models.VideoSearchQuery) ([]models.Video, int, error) {
	baseQuery := `
		SELECT DISTINCT v.id, v.library_id, v.title, v.file_path, v.file_size, v.duration, v.codec,
		       v.resolution, v.bitrate, v.fps, v.thumbnail_path, v.preview_path, v.date, v.rating, v.description,
		       v.is_favorite, v.is_pinned, v.not_interested, v.in_edit_list, v.content_type, v.created_at, v.updated_at, v.last_played_at, v.play_count,
		       COALESCE(v.is_missing, 0),
		       (SELECT CASE WHEN vl.edited_video_id = v.id THEN vl.backup_video_id ELSE vl.edited_video_id END
		        FROM video_links vl
		        WHERE (vl.edited_video_id = v.id OR vl.backup_video_id = v.id) AND vl.status = 'confirmed'
		        LIMIT 1) AS linked_video_id,
		       COALESCE(v.watched, 0), v.last_watched_at
		FROM videos v
	`

	var conditions []string
	var args []interface{}
	var joins []string

	if query.LibraryID > 0 {
		conditions = append(conditions, "v.library_id = ?")
		args = append(args, query.LibraryID)
	}

	if query.Query != "" {
		// Each word must appear somewhere in title/path/performers/tags/studios (AND logic).
		words := strings.Fields(query.Query)
		for _, word := range words {
			like := "%" + word + "%"
			conditions = append(conditions, `(
				v.title LIKE ? OR v.file_path LIKE ?
				OR EXISTS (SELECT 1 FROM video_performers vp JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = v.id AND p.name LIKE ?)
				OR EXISTS (SELECT 1 FROM video_tags vt JOIN tags t ON t.id = vt.tag_id WHERE vt.video_id = v.id AND t.name LIKE ?)
				OR EXISTS (SELECT 1 FROM video_studios vs JOIN studios s ON s.id = vs.studio_id WHERE vs.video_id = v.id AND s.name LIKE ?)
			)`)
			args = append(args, like, like, like, like, like)
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

	// Multi-tag: AND logic — video must have ALL selected tags
	for _, tagID := range query.TagIDs {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM video_tags vt WHERE vt.video_id = v.id AND vt.tag_id = ?)")
		args = append(args, tagID)
	}
	if query.TagID > 0 {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM video_tags vt WHERE vt.video_id = v.id AND vt.tag_id = ?)")
		args = append(args, query.TagID)
	}

	if query.Category != "" {
		joins = append(joins, "INNER JOIN video_performers vp2 ON v.id = vp2.video_id")
		joins = append(joins, "INNER JOIN performers p ON vp2.performer_id = p.id")
		conditions = append(conditions, "p.category = ?")
		args = append(args, query.Category)
	}
	if query.ContentType != "" {
		log.Printf("GetAll: Filtering by content_type=%s", query.ContentType)
		conditions = append(conditions, "v.content_type = ?")
		args = append(args, query.ContentType)
	}
	if query.NotInterested != nil {
		conditions = append(conditions, "v.not_interested = ?")
		args = append(args, *query.NotInterested)
	}
	if query.InEditList != nil {
		conditions = append(conditions, "v.in_edit_list = ?")
		args = append(args, *query.InEditList)
	}
	if !query.ShowMissing {
		conditions = append(conditions, "COALESCE(v.is_missing, 0) = 0")
	}
	if query.NoPerformer {
		conditions = append(conditions, "NOT EXISTS (SELECT 1 FROM video_performers vp_np WHERE vp_np.video_id = v.id)")
	}
	if query.Watched != nil {
		conditions = append(conditions, "COALESCE(v.watched, 0) = ?")
		args = append(args, *query.Watched)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(DISTINCT v.id) FROM videos v"
	if len(joins) > 0 {
		countQuery += " " + strings.Join(joins, " ")
	}
	countQuery += whereClause

	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Count query failed: %v", err)
		return nil, 0, fmt.Errorf("failed to count videos: %w", err)
	}

	mainQuery := baseQuery
	if len(joins) > 0 {
		mainQuery += " " + strings.Join(joins, " ")
	}
	mainQuery += whereClause

	sortBy := query.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := query.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	validSortColumns := map[string]bool{
		"id": true, "created_at": true, "updated_at": true, "title": true,
		"duration": true, "file_size": true, "play_count": true,
	}
	if !validSortColumns[sortBy] {
		sortBy = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	mainQuery += fmt.Sprintf(" ORDER BY v.%s %s", sortBy, strings.ToUpper(sortOrder))

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

	videos := make([]models.Video, 0)
	for rows.Next() {
		var video models.Video
		var lastPlayedAt, lastWatchedAt sql.NullTime
		var date, description, previewPath, contentType sql.NullString
		var linkedVideoID sql.NullInt64
		err := rows.Scan(
			&video.ID, &video.LibraryID, &video.Title, &video.FilePath, &video.FileSize, &video.Duration,
			&video.Codec, &video.Resolution, &video.Bitrate, &video.FPS, &video.ThumbnailPath, &previewPath,
			&date, &video.Rating, &description, &video.IsFavorite, &video.IsPinned, &video.NotInterested, &video.InEditList,
			&contentType, &video.CreatedAt, &video.UpdatedAt, &lastPlayedAt, &video.PlayCount,
			&video.IsMissing, &linkedVideoID, &video.Watched, &lastWatchedAt,
		)
		if err != nil {
			log.Printf("Failed to scan video row: %v", err)
			return nil, 0, fmt.Errorf("failed to scan video: %w", err)
		}
		if lastPlayedAt.Valid {
			video.LastPlayedAt = &lastPlayedAt.Time
		}
		if lastWatchedAt.Valid {
			video.LastWatchedAt = &lastWatchedAt.Time
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

	if err = rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
		return nil, 0, fmt.Errorf("error iterating videos: %w", err)
	}

	if len(videos) > 0 {
		if err := s.loadVideoRelationshipsBatch(videos); err != nil {
			log.Printf("Warning: Failed to batch load relationships: %v", err)
		}
	}

	return videos, total, nil
}

// GetByID retrieves a single video by its primary key, including relationships.
func (s *VideoService) GetByID(id int64) (*models.Video, error) {
	var video models.Video
	var lastPlayedAt sql.NullTime
	var date, description, contentType sql.NullString
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

	if err := s.loadVideoRelationships(&video); err != nil {
		log.Printf("Warning: Failed to load relationships for video %d: %v", video.ID, err)
	}

	return &video, nil
}

// GetByPerformer retrieves all videos featuring a specific performer.
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

// Create inserts a new video record. thumbnail_path should be empty here;
// set it via updateVideoThumbnailPath only after successful generation.
func (s *VideoService) Create(create *models.VideoCreate) (*models.Video, error) {
	now := time.Now()
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
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	result, err := s.db.Exec(`
		INSERT INTO videos (library_id, title, file_path, file_size, duration, codec, resolution, bitrate, fps, thumbnail_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, video.LibraryID, video.Title, video.FilePath, video.FileSize, video.Duration, video.Codec,
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

// Update modifies an existing video's fields and relationships.
func (s *VideoService) Update(id int64, update *models.VideoUpdate) (*models.Video, error) {
	video, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

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

	_, err = s.db.Exec(`
		UPDATE videos
		SET title = ?, date = ?, rating = ?, description = ?, is_favorite = ?, is_pinned = ?,
		    play_count = ?, not_interested = ?, in_edit_list = ?, content_type = ?, updated_at = ?
		WHERE id = ?
	`, video.Title, video.Date, video.Rating, video.Description,
		video.IsFavorite, video.IsPinned, video.PlayCount, video.NotInterested, video.InEditList, video.ContentType, video.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update video: %w", err)
	}

	if update.StudioID != nil {
		if _, err := s.db.Exec("DELETE FROM video_studios WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear studios: %w", err)
		}
		if *update.StudioID > 0 {
			if _, err := s.db.Exec("INSERT INTO video_studios (video_id, studio_id) VALUES (?, ?)", id, *update.StudioID); err != nil {
				return nil, fmt.Errorf("failed to add studio: %w", err)
			}
		}
	}

	if update.GroupID != nil {
		if _, err := s.db.Exec("DELETE FROM video_groups WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear groups: %w", err)
		}
		if *update.GroupID > 0 {
			if _, err := s.db.Exec("INSERT INTO video_groups (video_id, group_id) VALUES (?, ?)", id, *update.GroupID); err != nil {
				return nil, fmt.Errorf("failed to add group: %w", err)
			}
		}
	}

	if update.PerformerIDs != nil {
		existingRows, err := s.db.Query(`SELECT performer_id FROM video_performers WHERE video_id = ?`, id)
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

		if _, err := s.db.Exec("DELETE FROM video_performers WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear performers: %w", err)
		}

		for oldPerformerID := range existingPerformers {
			if s.performerService != nil {
				if err := s.performerService.DecrementVideoCount(oldPerformerID); err != nil {
					log.Printf("Warning: failed to decrement video count for performer %d: %v", oldPerformerID, err)
				}
			}
		}

		for _, performerID := range update.PerformerIDs {
			if _, err := s.db.Exec("INSERT INTO video_performers (video_id, performer_id) VALUES (?, ?)", id, performerID); err != nil {
				return nil, fmt.Errorf("failed to add performer: %w", err)
			}
			if s.performerService != nil {
				if err := s.performerService.IncrementVideoCount(performerID); err != nil {
					log.Printf("Warning: failed to increment video count for performer %d: %v", performerID, err)
				}
				if err := s.performerService.ApplyMasterTagsToVideo(performerID, id); err != nil {
					log.Printf("Warning: failed to apply master tags from performer %d to video %d: %v", performerID, id, err)
				}
			}
		}
	}

	if update.TagIDs != nil {
		if _, err := s.db.Exec("DELETE FROM video_tags WHERE video_id = ?", id); err != nil {
			return nil, fmt.Errorf("failed to clear tags: %w", err)
		}
		for _, tagID := range update.TagIDs {
			if _, err := s.db.Exec("INSERT INTO video_tags (video_id, tag_id) VALUES (?, ?)", id, tagID); err != nil {
				return nil, fmt.Errorf("failed to add tag: %w", err)
			}
		}
	}

	return s.GetByID(id)
}

// Delete removes a video record and updates performer video counts.
func (s *VideoService) Delete(id int64) error {
	rows, _ := s.db.Query(`SELECT performer_id FROM video_performers WHERE video_id = ?`, id)
	var performerIDs []int64
	for rows != nil && rows.Next() {
		var pid int64
		if rows.Scan(&pid) == nil {
			performerIDs = append(performerIDs, pid)
		}
	}
	if rows != nil {
		rows.Close()
	}

	if _, err := s.db.Exec(`DELETE FROM videos WHERE id = ?`, id); err != nil {
		return fmt.Errorf("failed to delete video: %w", err)
	}

	for _, pid := range performerIDs {
		s.db.Exec(`UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = ?) WHERE id = ?`, pid, pid)
	}
	return nil
}

// MarkWatched sets or clears the watched flag on a video, updating last_watched_at on mark.
func (s *VideoService) MarkWatched(id int64, watched bool) error {
	if watched {
		_, err := s.db.Exec(`UPDATE videos SET watched = 1, last_watched_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
		return err
	}
	_, err := s.db.Exec(`UPDATE videos SET watched = 0, last_watched_at = NULL WHERE id = ?`, id)
	return err
}

// GetByFilePath retrieves a video by its file_path, including relationships.
func (s *VideoService) GetByFilePath(filePath string) (*models.Video, error) {
	var video models.Video
	var date, description, contentType sql.NullString
	var lastPlayedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, library_id, title, file_path, file_size, duration, codec, resolution,
		       bitrate, fps, thumbnail_path, date, rating, description, is_favorite, is_pinned,
		       not_interested, in_edit_list, content_type, created_at, updated_at, last_played_at, play_count
		FROM videos WHERE file_path = ?
	`, filePath).Scan(
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

	if err := s.loadVideoRelationships(&video); err != nil {
		return nil, err
	}

	return &video, nil
}

// loadVideoRelationshipsBatch batch-loads performers, tags, studios, and groups
// for a slice of videos using one query per relation type.
func (s *VideoService) loadVideoRelationshipsBatch(videos []models.Video) error {
	if len(videos) == 0 {
		return nil
	}

	videoIDs := make([]int64, len(videos))
	videoMap := make(map[int64]*models.Video)
	for i := range videos {
		videoIDs[i] = videos[i].ID
		videoMap[videos[i].ID] = &videos[i]
	}

	placeholders := make([]string, len(videoIDs))
	args := make([]interface{}, len(videoIDs))
	for i, id := range videoIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	placeholderStr := strings.Join(placeholders, ",")

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
					performer.UnmarshalMetadata()
				}
				if video, exists := videoMap[videoID]; exists {
					video.Performers = append(video.Performers, performer)
				}
			}
		}
	}

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

// loadVideoRelationships loads performers, tags, studios, and groups for a single video.
func (s *VideoService) loadVideoRelationships(video *models.Video) error {
	performerRows, err := s.db.Query(`
		SELECT p.id, p.name, p.preview_path, p.folder_path, p.video_count, p.category, p.metadata, p.created_at, p.updated_at
		FROM performers p
		INNER JOIN video_performers vp ON p.id = vp.performer_id
		WHERE vp.video_id = ?
	`, video.ID)
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
					performer.UnmarshalMetadata()
				}
				video.Performers = append(video.Performers, performer)
			}
		}
	}

	tagRows, err := s.db.Query(`
		SELECT t.id, t.name, t.color, t.icon, t.created_at, t.updated_at
		FROM tags t
		INNER JOIN video_tags vt ON t.id = vt.tag_id
		WHERE vt.video_id = ?
	`, video.ID)
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

	studioRows, err := s.db.Query(`
		SELECT s.id, s.name, s.logo_path, s.description, s.founded_date, s.country, s.metadata, s.created_at, s.updated_at
		FROM studios s
		INNER JOIN video_studios vs ON s.id = vs.studio_id
		WHERE vs.video_id = ?
	`, video.ID)
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

	groupRows, err := s.db.Query(`
		SELECT g.id, g.studio_id, g.name, g.logo_path, g.description, g.metadata, g.created_at, g.updated_at
		FROM groups g
		INNER JOIN video_groups vg ON g.id = vg.group_id
		WHERE vg.video_id = ?
	`, video.ID)
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
