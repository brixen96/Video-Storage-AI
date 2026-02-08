package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// VideoListService handles operations on video lists
type VideoListService struct {
	db *sql.DB
}

// NewVideoListService creates a new VideoListService
func NewVideoListService(db *sql.DB) *VideoListService {
	return &VideoListService{db: db}
}

// GetAllLists returns all video lists
func (s *VideoListService) GetAllLists() ([]models.VideoList, error) {
	query := `
		SELECT id, name, description, color, icon, is_default, position, video_count, created_at, updated_at
		FROM video_lists
		ORDER BY position ASC, created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query lists: %w", err)
	}
	defer rows.Close()

	var lists []models.VideoList
	for rows.Next() {
		var list models.VideoList
		err := rows.Scan(
			&list.ID, &list.Name, &list.Description, &list.Color, &list.Icon,
			&list.IsDefault, &list.Position, &list.VideoCount, &list.CreatedAt, &list.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan list: %w", err)
		}
		lists = append(lists, list)
	}

	return lists, nil
}

// GetListByID returns a single list by ID
func (s *VideoListService) GetListByID(id int64) (*models.VideoList, error) {
	query := `
		SELECT id, name, description, color, icon, is_default, position, video_count, created_at, updated_at
		FROM video_lists
		WHERE id = ?
	`

	var list models.VideoList
	err := s.db.QueryRow(query, id).Scan(
		&list.ID, &list.Name, &list.Description, &list.Color, &list.Icon,
		&list.IsDefault, &list.Position, &list.VideoCount, &list.CreatedAt, &list.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("list not found")
		}
		return nil, fmt.Errorf("failed to query list: %w", err)
	}

	return &list, nil
}

// CreateList creates a new video list
func (s *VideoListService) CreateList(create *models.VideoListCreate) (*models.VideoList, error) {
	// Get max position
	var maxPosition int
	err := s.db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM video_lists").Scan(&maxPosition)
	if err != nil {
		return nil, fmt.Errorf("failed to get max position: %w", err)
	}

	// Set defaults
	color := create.Color
	if color == "" {
		color = "#007bff"
	}
	icon := create.Icon
	if icon == "" {
		icon = "list"
	}

	query := `
		INSERT INTO video_lists (name, description, color, icon, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := s.db.Exec(query, create.Name, create.Description, color, icon, maxPosition+1, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get insert ID: %w", err)
	}

	return s.GetListByID(id)
}

// UpdateList updates an existing video list
func (s *VideoListService) UpdateList(id int64, update *models.VideoListUpdate) (*models.VideoList, error) {
	// Check if list exists
	_, err := s.GetListByID(id)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	setClauses := []string{"updated_at = ?"}
	args := []interface{}{time.Now()}

	if update.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *update.Name)
	}
	if update.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *update.Description)
	}
	if update.Color != nil {
		setClauses = append(setClauses, "color = ?")
		args = append(args, *update.Color)
	}
	if update.Icon != nil {
		setClauses = append(setClauses, "icon = ?")
		args = append(args, *update.Icon)
	}
	if update.Position != nil {
		setClauses = append(setClauses, "position = ?")
		args = append(args, *update.Position)
	}

	args = append(args, id)

	query := "UPDATE video_lists SET "
	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += " WHERE id = ?"

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update list: %w", err)
	}

	return s.GetListByID(id)
}

// DeleteList deletes a video list
func (s *VideoListService) DeleteList(id int64) error {
	// Check if it's the default list
	var isDefault bool
	err := s.db.QueryRow("SELECT is_default FROM video_lists WHERE id = ?", id).Scan(&isDefault)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("list not found")
		}
		return fmt.Errorf("failed to check list: %w", err)
	}

	if isDefault {
		return fmt.Errorf("cannot delete the default list")
	}

	// Delete list (cascade will delete items)
	_, err = s.db.Exec("DELETE FROM video_lists WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete list: %w", err)
	}

	return nil
}

// AddVideosToList adds videos to a list
func (s *VideoListService) AddVideosToList(listID int64, request *models.AddToListRequest) error {
	// Check if list exists
	_, err := s.GetListByID(listID)
	if err != nil {
		return err
	}

	// Get current max position
	var maxPosition int
	err = s.db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM video_list_items WHERE list_id = ?", listID).Scan(&maxPosition)
	if err != nil {
		return fmt.Errorf("failed to get max position: %w", err)
	}

	position := maxPosition + 1

	// Add local videos
	for _, videoID := range request.VideoIDs {
		_, err := s.db.Exec(`
			INSERT OR IGNORE INTO video_list_items (list_id, video_id, position, notes, added_at)
			VALUES (?, ?, ?, ?, ?)
		`, listID, videoID, position, request.Notes, time.Now())
		if err != nil {
			return fmt.Errorf("failed to add video %d: %w", videoID, err)
		}
		position++
	}

	// Add scraped videos
	for _, scrapedID := range request.ScrapedVideoIDs {
		_, err := s.db.Exec(`
			INSERT OR IGNORE INTO video_list_items (list_id, scraped_video_id, position, notes, added_at)
			VALUES (?, ?, ?, ?, ?)
		`, listID, scrapedID, position, request.Notes, time.Now())
		if err != nil {
			return fmt.Errorf("failed to add scraped video %d: %w", scrapedID, err)
		}
		position++
	}

	// Update video count
	s.updateVideoCount(listID)

	return nil
}

// RemoveVideosFromList removes videos from a list
func (s *VideoListService) RemoveVideosFromList(listID int64, request *models.RemoveFromListRequest) error {
	// Remove by item IDs
	for _, itemID := range request.ItemIDs {
		_, err := s.db.Exec("DELETE FROM video_list_items WHERE id = ? AND list_id = ?", itemID, listID)
		if err != nil {
			return fmt.Errorf("failed to remove item %d: %w", itemID, err)
		}
	}

	// Remove by video IDs
	for _, videoID := range request.VideoIDs {
		_, err := s.db.Exec("DELETE FROM video_list_items WHERE video_id = ? AND list_id = ?", videoID, listID)
		if err != nil {
			return fmt.Errorf("failed to remove video %d: %w", videoID, err)
		}
	}

	// Remove by scraped video IDs
	for _, scrapedID := range request.ScrapedVideoIDs {
		_, err := s.db.Exec("DELETE FROM video_list_items WHERE scraped_video_id = ? AND list_id = ?", scrapedID, listID)
		if err != nil {
			return fmt.Errorf("failed to remove scraped video %d: %w", scrapedID, err)
		}
	}

	// Update video count
	s.updateVideoCount(listID)

	return nil
}

// GetListItems returns all items in a list with video details
func (s *VideoListService) GetListItems(listID int64) ([]models.VideoListItem, error) {
	query := `
		SELECT
			vli.id, vli.list_id, vli.video_id, vli.scraped_video_id, vli.position, COALESCE(vli.notes, ''), vli.added_at,
			COALESCE(v.title, sv.title, '') as title,
			COALESCE(v.thumbnail_path, '') as thumbnail_path,
			COALESCE(sv.thumbnail_url, '') as thumbnail_url,
			COALESCE(v.duration, sv.duration, 0) as duration,
			COALESCE(v.file_size, 0) as file_size,
			COALESCE(v.resolution, '') as resolution,
			COALESCE(sv.source, 'local') as source,
			CASE WHEN vli.scraped_video_id IS NOT NULL THEN 1 ELSE 0 END as is_scraped,
			COALESCE(v.file_path, '') as file_path,
			COALESCE(v.library_id, 0) as library_id,
			COALESCE(l.name, '') as library_name
		FROM video_list_items vli
		LEFT JOIN videos v ON vli.video_id = v.id
		LEFT JOIN scraped_videos sv ON vli.scraped_video_id = sv.id
		LEFT JOIN libraries l ON v.library_id = l.id
		WHERE vli.list_id = ?
		ORDER BY vli.position ASC
	`

	rows, err := s.db.Query(query, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to query items: %w", err)
	}
	defer rows.Close()

	var items []models.VideoListItem
	for rows.Next() {
		var item models.VideoListItem
		var videoID, scrapedVideoID sql.NullInt64
		err := rows.Scan(
			&item.ID, &item.ListID, &videoID, &scrapedVideoID, &item.Position, &item.Notes, &item.AddedAt,
			&item.Title, &item.ThumbnailPath, &item.ThumbnailURL, &item.Duration, &item.FileSize, &item.Resolution, &item.Source, &item.IsScraped,
			&item.FilePath, &item.LibraryID, &item.LibraryName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		if videoID.Valid {
			item.VideoID = &videoID.Int64
		}
		if scrapedVideoID.Valid {
			item.ScrapedVideoID = &scrapedVideoID.Int64
		}
		items = append(items, item)
	}

	return items, nil
}

// GetListStats returns statistics about a list
func (s *VideoListService) GetListStats(listID int64) (*models.VideoListStats, error) {
	query := `
		SELECT
			COUNT(*) as total_videos,
			SUM(CASE WHEN vli.video_id IS NOT NULL THEN 1 ELSE 0 END) as local_videos,
			SUM(CASE WHEN vli.scraped_video_id IS NOT NULL THEN 1 ELSE 0 END) as scraped_videos,
			COALESCE(SUM(COALESCE(v.duration, sv.duration, 0)), 0) as total_duration,
			COALESCE(SUM(COALESCE(v.file_size, 0)), 0) as total_size,
			COALESCE(AVG(COALESCE(v.rating, sv.rating, 0)), 0) as average_rating
		FROM video_list_items vli
		LEFT JOIN videos v ON vli.video_id = v.id
		LEFT JOIN scraped_videos sv ON vli.scraped_video_id = sv.id
		WHERE vli.list_id = ?
	`

	var stats models.VideoListStats
	err := s.db.QueryRow(query, listID).Scan(
		&stats.TotalVideos, &stats.LocalVideos, &stats.ScrapedVideos,
		&stats.TotalDuration, &stats.TotalSize, &stats.AverageRating,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// updateVideoCount updates the video_count for a list
func (s *VideoListService) updateVideoCount(listID int64) {
	s.db.Exec(`
		UPDATE video_lists
		SET video_count = (SELECT COUNT(*) FROM video_list_items WHERE list_id = ?),
		    updated_at = ?
		WHERE id = ?
	`, listID, time.Now(), listID)
}

// GetListsForVideo returns all lists that contain a specific video
func (s *VideoListService) GetListsForVideo(videoID int64, isScraped bool) ([]models.VideoList, error) {
	var query string
	if isScraped {
		query = `
			SELECT vl.id, vl.name, vl.description, vl.color, vl.icon, vl.is_default, vl.position, vl.video_count, vl.created_at, vl.updated_at
			FROM video_lists vl
			INNER JOIN video_list_items vli ON vl.id = vli.list_id
			WHERE vli.scraped_video_id = ?
			ORDER BY vl.position ASC
		`
	} else {
		query = `
			SELECT vl.id, vl.name, vl.description, vl.color, vl.icon, vl.is_default, vl.position, vl.video_count, vl.created_at, vl.updated_at
			FROM video_lists vl
			INNER JOIN video_list_items vli ON vl.id = vli.list_id
			WHERE vli.video_id = ?
			ORDER BY vl.position ASC
		`
	}

	rows, err := s.db.Query(query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lists: %w", err)
	}
	defer rows.Close()

	var lists []models.VideoList
	for rows.Next() {
		var list models.VideoList
		err := rows.Scan(
			&list.ID, &list.Name, &list.Description, &list.Color, &list.Icon,
			&list.IsDefault, &list.Position, &list.VideoCount, &list.CreatedAt, &list.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan list: %w", err)
		}
		lists = append(lists, list)
	}

	return lists, nil
}

// ReorderListItems reorders items within a list
func (s *VideoListService) ReorderListItems(listID int64, itemIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i, itemID := range itemIDs {
		_, err := tx.Exec("UPDATE video_list_items SET position = ? WHERE id = ? AND list_id = ?", i+1, itemID, listID)
		if err != nil {
			return fmt.Errorf("failed to update position: %w", err)
		}
	}

	return tx.Commit()
}

// UpdateItemNotes updates the notes for a list item
func (s *VideoListService) UpdateItemNotes(listID, itemID int64, notes string) error {
	result, err := s.db.Exec("UPDATE video_list_items SET notes = ? WHERE id = ? AND list_id = ?", notes, itemID, listID)
	if err != nil {
		return fmt.Errorf("failed to update notes: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("item not found in list")
	}

	return nil
}
