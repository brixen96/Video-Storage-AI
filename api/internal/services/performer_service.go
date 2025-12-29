package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// PerformerService handles performer-related business logic
type PerformerService struct {
	db *sql.DB
}

// NewPerformerService creates a new performer service
func NewPerformerService() *PerformerService {
	return &PerformerService{
		db: database.GetDB(),
	}
}

// GetAll retrieves all performers
func (s *PerformerService) GetAll() ([]models.Performer, error) {
	query := `
		SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
		FROM performers
		ORDER BY name ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query performers: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var performers []models.Performer
	for rows.Next() {
		var p models.Performer
		err := rows.Scan(
			&p.ID, &p.Name, &p.PreviewPath, &p.ThumbnailPath, &p.FolderPath,
			&p.VideoCount, &p.Category, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan performer: %w", err)
		}

		// Parse metadata JSON
		if err := p.UnmarshalMetadata(); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata for performer %d: %w", p.ID, err)
		}

		performers = append(performers, p)
	}

	return performers, nil
}

// GetByID retrieves a performer by ID
func (s *PerformerService) GetByID(id int64) (*models.Performer, error) {
	query := `
		SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
		FROM performers
		WHERE id = ?
	`

	var p models.Performer
	err := s.db.QueryRow(query, id).Scan(
		&p.ID, &p.Name, &p.PreviewPath, &p.ThumbnailPath, &p.FolderPath,
		&p.VideoCount, &p.Category, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("performer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query performer: %w", err)
	}

	// Parse metadata JSON
	if err := p.UnmarshalMetadata(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &p, nil
}

// GetPerformerByName retrieves a performer by name
func (s *PerformerService) GetPerformerByName(name string) (*models.Performer, error) {
	query := `
		SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
		FROM performers
		WHERE name = ?
	`

	var p models.Performer
	err := s.db.QueryRow(query, name).Scan(
		&p.ID, &p.Name, &p.PreviewPath, &p.ThumbnailPath, &p.FolderPath,
		&p.VideoCount, &p.Category, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("performer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query performer: %w", err)
	}

	// Parse metadata JSON
	if err := p.UnmarshalMetadata(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &p, nil
}

// Create creates a new performer
func (s *PerformerService) Create(create *models.PerformerCreate) (*models.Performer, error) {
	// Check if performer already exists
	existing, _ := s.GetPerformerByName(create.Name)
	if existing != nil {
		return nil, fmt.Errorf("performer with name '%s' already exists", create.Name)
	}

	// Prepare performer
	performer := &models.Performer{
		Name:        create.Name,
		PreviewPath: sql.NullString{String: create.PreviewPath, Valid: create.PreviewPath != ""},
		FolderPath:  sql.NullString{String: create.FolderPath, Valid: create.FolderPath != ""},
		VideoCount:  0,
		MetadataObj: create.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Marshal metadata
	if err := performer.MarshalMetadata(); err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Default category to 'regular' if not provided
	if performer.Category == "" {
		performer.Category = "regular"
	}

	// Insert into database
	query := `
		INSERT INTO performers (name, preview_path, folder_path, video_count, category, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(
		query,
		performer.Name, performer.PreviewPath, performer.FolderPath,
		performer.VideoCount, performer.Category, performer.Metadata, performer.CreatedAt, performer.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert performer: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	performer.ID = id
	return performer, nil
}

// Update updates an existing performer
func (s *PerformerService) Update(id int64, update *models.PerformerUpdate) (*models.Performer, error) {
	// Get existing performer
	performer, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if update.Name != nil {
		performer.Name = *update.Name
	}
	if update.PreviewPath != nil {
		performer.PreviewPath = sql.NullString{String: *update.PreviewPath, Valid: *update.PreviewPath != ""}
	}
	if update.ThumbnailPath != nil {
		performer.ThumbnailPath = sql.NullString{String: *update.ThumbnailPath, Valid: *update.ThumbnailPath != ""}
	}
	if update.FolderPath != nil {
		performer.FolderPath = sql.NullString{String: *update.FolderPath, Valid: *update.FolderPath != ""}
	}
	if update.Category != nil {
		performer.Category = *update.Category
	}
	if update.Metadata != nil {
		performer.MetadataObj = update.Metadata
	}

	performer.UpdatedAt = time.Now()

	// Marshal metadata
	if err := performer.MarshalMetadata(); err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Update database
	query := `
		UPDATE performers
		SET name = ?, preview_path = ?, thumbnail_path = ?, folder_path = ?, category = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(
		query,
		performer.Name, performer.PreviewPath, performer.ThumbnailPath, performer.FolderPath,
		performer.Category, performer.Metadata, performer.UpdatedAt, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update performer: %w", err)
	}

	return performer, nil
}

// Delete deletes a performer
func (s *PerformerService) Delete(id int64) error {
	// Check if performer exists
	_, err := s.GetByID(id)
	if err != nil {
		return err
	}

	query := `DELETE FROM performers WHERE id = ?`
	_, err = s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete performer: %w", err)
	}

	return nil
}

// Search searches performers by name
func (s *PerformerService) Search(searchTerm string) ([]models.Performer, error) {
	query := `
		SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
		FROM performers
		WHERE name LIKE ?
		ORDER BY name ASC
	`

	rows, err := s.db.Query(query, "%"+searchTerm+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search performers: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var performers []models.Performer
	for rows.Next() {
		var p models.Performer
		err := rows.Scan(
			&p.ID, &p.Name, &p.PreviewPath, &p.ThumbnailPath, &p.FolderPath,
			&p.VideoCount, &p.Category, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan performer: %w", err)
		}

		// Parse metadata JSON
		if err := p.UnmarshalMetadata(); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		performers = append(performers, p)
	}

	return performers, nil
}

// ResetMetadata clears all metadata for a performer
func (s *PerformerService) ResetMetadata(id int64) error {
	query := `
		UPDATE performers
		SET metadata = '{}', updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to reset metadata: %w", err)
	}

	return nil
}

// GetAllPaginated retrieves performers with pagination
func (s *PerformerService) GetAllPaginated(limit, offset int) ([]models.Performer, int64, error) {
	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM performers"
	err := s.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count performers: %w", err)
	}

	// Get paginated results
	query := `
        SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
        FROM performers
        ORDER BY name ASC
        LIMIT ? OFFSET ?
    `

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query performers: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var performers []models.Performer
	for rows.Next() {
		var p models.Performer
		err := rows.Scan(
			&p.ID, &p.Name, &p.PreviewPath, &p.ThumbnailPath, &p.FolderPath,
			&p.VideoCount, &p.Category, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan performer: %w", err)
		}

		// Parse metadata JSON
		if err := p.UnmarshalMetadata(); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		performers = append(performers, p)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating performers: %w", err)
	}

	return performers, total, nil
}

// SearchPaginated searches performers with pagination
func (s *PerformerService) SearchPaginated(searchTerm string, limit, offset int) ([]models.Performer, int64, error) {
	searchPattern := "%" + searchTerm + "%"

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM performers WHERE name LIKE ?"
	err := s.db.QueryRow(countQuery, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count performers: %w", err)
	}

	// Get paginated results
	query := `
        SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
        FROM performers
        WHERE name LIKE ?
        ORDER BY name ASC
        LIMIT ? OFFSET ?
    `

	rows, err := s.db.Query(query, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query performers: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var performers []models.Performer
	for rows.Next() {
		var p models.Performer
		err := rows.Scan(
			&p.ID, &p.Name, &p.PreviewPath, &p.ThumbnailPath, &p.FolderPath,
			&p.VideoCount, &p.Category, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan performer: %w", err)
		}

		// Parse metadata JSON
		if err := p.UnmarshalMetadata(); err != nil {
			return nil, 0, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		performers = append(performers, p)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating performers: %w", err)
	}

	return performers, total, nil
}

// GetPerformerTags retrieves all tags for a performer (master tags)
func (s *PerformerService) GetPerformerTags(performerID int64) ([]models.Tag, error) {
	query := `
        SELECT t.id, t.name, t.color, t.icon, t.created_at, t.updated_at
        FROM tags t
        INNER JOIN performer_tags pt ON t.id = pt.tag_id
        WHERE pt.performer_id = ?
        ORDER BY t.name ASC
    `

	rows, err := s.db.Query(query, performerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query performer tags: %w", err)
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Icon, &tag.CreatedAt, &tag.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// AddPerformerTag adds a master tag to a performer
func (s *PerformerService) AddPerformerTag(performerID, tagID int64) error {
	// Check if relationship already exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM performer_tags WHERE performer_id = ? AND tag_id = ?", performerID, tagID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing performer tag: %w", err)
	}

	if count > 0 {
		return nil // Already exists
	}

	// Create the relationship
	query := "INSERT INTO performer_tags (performer_id, tag_id) VALUES (?, ?)"
	_, err = s.db.Exec(query, performerID, tagID)
	if err != nil {
		return fmt.Errorf("failed to add performer tag: %w", err)
	}

	log.Printf("Added tag %d to performer %d", tagID, performerID)
	return nil
}

// RemovePerformerTag removes a master tag from a performer
func (s *PerformerService) RemovePerformerTag(performerID, tagID int64) error {
	query := "DELETE FROM performer_tags WHERE performer_id = ? AND tag_id = ?"
	_, err := s.db.Exec(query, performerID, tagID)
	if err != nil {
		return fmt.Errorf("failed to remove performer tag: %w", err)
	}

	log.Printf("Removed tag %d from performer %d", tagID, performerID)
	return nil
}

// SyncPerformerTagsToVideos applies a performer's master tags to all their videos
func (s *PerformerService) SyncPerformerTagsToVideos(performerID int64) (int, error) {
	// Get all master tags for this performer
	performerTags, err := s.GetPerformerTags(performerID)
	if err != nil {
		return 0, fmt.Errorf("failed to get performer tags: %w", err)
	}

	if len(performerTags) == 0 {
		return 0, nil // No tags to sync
	}

	// Get all videos featuring this performer
	query := `
        SELECT video_id
        FROM video_performers
        WHERE performer_id = ?
    `

	rows, err := s.db.Query(query, performerID)
	if err != nil {
		return 0, fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	var videoIDs []int64
	for rows.Next() {
		var videoID int64
		if err := rows.Scan(&videoID); err != nil {
			return 0, fmt.Errorf("failed to scan video ID: %w", err)
		}
		videoIDs = append(videoIDs, videoID)
	}

	// Add each master tag to each video (if not already present)
	tagsAdded := 0
	videosUpdated := make(map[int64]bool)
	for _, videoID := range videoIDs {
		for _, tag := range performerTags {
			// Check if video already has this tag
			var count int
			err := s.db.QueryRow("SELECT COUNT(*) FROM video_tags WHERE video_id = ? AND tag_id = ?", videoID, tag.ID).Scan(&count)
			if err != nil {
				log.Printf("Warning: Failed to check existing video tag: %v", err)
				continue
			}

			if count == 0 {
				// Add the tag
				_, err = s.db.Exec("INSERT INTO video_tags (video_id, tag_id) VALUES (?, ?)", videoID, tag.ID)
				if err != nil {
					log.Printf("Warning: Failed to add tag %d to video %d: %v", tag.ID, videoID, err)
					continue
				}
				tagsAdded++
				videosUpdated[videoID] = true
			}
		}
	}

	log.Printf("Synced %d tags to %d videos for performer %d", tagsAdded, len(videosUpdated), performerID)
	return len(videosUpdated), nil
}

// ApplyMasterTagsToVideo automatically adds a performer's master tags to a video
// This should be called whenever a performer is linked to a video
func (s *PerformerService) ApplyMasterTagsToVideo(performerID, videoID int64) error {
	// Get all master tags for this performer
	performerTags, err := s.GetPerformerTags(performerID)
	if err != nil {
		return fmt.Errorf("failed to get performer tags: %w", err)
	}

	// Add each master tag to the video (if not already present)
	for _, tag := range performerTags {
		// Check if video already has this tag
		var count int
		err := s.db.QueryRow("SELECT COUNT(*) FROM video_tags WHERE video_id = ? AND tag_id = ?", videoID, tag.ID).Scan(&count)
		if err != nil {
			log.Printf("Warning: Failed to check existing video tag: %v", err)
			continue
		}

		if count == 0 {
			// Add the tag
			_, err = s.db.Exec("INSERT INTO video_tags (video_id, tag_id) VALUES (?, ?)", videoID, tag.ID)
			if err != nil {
				log.Printf("Warning: Failed to add master tag %d to video %d: %v", tag.ID, videoID, err)
				continue
			}
			log.Printf("Applied master tag '%s' from performer %d to video %d", tag.Name, performerID, videoID)
		}
	}

	return nil
}

// UpdateVideoCount updates the video_count for a performer based on actual video relationships
func (s *PerformerService) UpdateVideoCount(performerID int64) error {
	query := `
		UPDATE performers
		SET video_count = (
			SELECT COUNT(*) FROM video_performers WHERE performer_id = ?
		)
		WHERE id = ?
	`
	_, err := s.db.Exec(query, performerID, performerID)
	if err != nil {
		return fmt.Errorf("failed to update video count: %w", err)
	}
	return nil
}

// RecalculateAllVideoCounts recalculates video_count for all performers
func (s *PerformerService) RecalculateAllVideoCounts() error {
	query := `
		UPDATE performers
		SET video_count = (
			SELECT COUNT(*) FROM video_performers WHERE performer_id = performers.id
		)
	`
	result, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to recalculate video counts: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Recalculated video counts for %d performers", rowsAffected)
	return nil
}

// IncrementVideoCount increments the video_count for a performer
func (s *PerformerService) IncrementVideoCount(performerID int64) error {
	query := `UPDATE performers SET video_count = video_count + 1 WHERE id = ?`
	_, err := s.db.Exec(query, performerID)
	if err != nil {
		return fmt.Errorf("failed to increment video count: %w", err)
	}
	return nil
}

// DecrementVideoCount decrements the video_count for a performer
func (s *PerformerService) DecrementVideoCount(performerID int64) error {
	query := `UPDATE performers SET video_count = MAX(0, video_count - 1) WHERE id = ?`
	_, err := s.db.Exec(query, performerID)
	if err != nil {
		return fmt.Errorf("failed to decrement video count: %w", err)
	}
	return nil
}
