package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// TagService handles tag-related business logic
type TagService struct {
	db *sql.DB
}

// NewTagService creates a new tag service
func NewTagService() *TagService {
	return &TagService{
		db: database.GetDB(),
	}
}

// GetAll retrieves all tags with video counts
func (s *TagService) GetAll() ([]models.TagWithCount, error) {
	query := `
		SELECT t.id, t.name, t.color, t.icon, t.category, t.created_at, t.updated_at,
		       COALESCE(COUNT(vt.video_id), 0) as video_count
		FROM tags t
		LEFT JOIN video_tags vt ON t.id = vt.tag_id
		GROUP BY t.id
		ORDER BY t.name ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	var tags []models.TagWithCount
	for rows.Next() {
		var tag models.TagWithCount
		err := rows.Scan(
			&tag.ID, &tag.Name, &tag.Color, &tag.Icon, &tag.Category,
			&tag.CreatedAt, &tag.UpdatedAt, &tag.VideoCount,
		)
		if err != nil {
			log.Printf("Failed to scan tag: %v", err)
			continue
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// GetByID retrieves a single tag by ID
func (s *TagService) GetByID(id int64) (*models.Tag, error) {
	query := `SELECT id, name, color, icon, category, created_at, updated_at FROM tags WHERE id = ?`

	var tag models.Tag
	err := s.db.QueryRow(query, id).Scan(
		&tag.ID, &tag.Name, &tag.Color, &tag.Icon, &tag.Category,
		&tag.CreatedAt, &tag.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tag not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	return &tag, nil
}

// Create creates a new tag
func (s *TagService) Create(create *models.TagCreate) (*models.Tag, error) {
	now := time.Now()

	// Default to 'regular' if category is empty
	category := create.Category
	if category == "" {
		category = "regular"
	}

	query := `INSERT INTO tags (name, color, icon, category, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := s.db.Exec(query, create.Name, create.Color, create.Icon, category, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetByID(id)
}

// Update updates an existing tag
func (s *TagService) Update(id int64, update *models.TagUpdate) (*models.Tag, error) {
	// Get existing tag
	tag, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if update.Name != nil {
		tag.Name = *update.Name
	}
	if update.Color != nil {
		tag.Color = *update.Color
	}
	if update.Icon != nil {
		tag.Icon = *update.Icon
	}
	if update.Category != nil {
		tag.Category = *update.Category
	}

	tag.UpdatedAt = time.Now()

	query := `UPDATE tags SET name = ?, color = ?, icon = ?, category = ?, updated_at = ? WHERE id = ?`
	_, err = s.db.Exec(query, tag.Name, tag.Color, tag.Icon, tag.Category, tag.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	return tag, nil
}

// Delete deletes a tag
func (s *TagService) Delete(id int64) error {
	// First, remove all video-tag associations
	_, err := s.db.Exec(`DELETE FROM video_tags WHERE tag_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete video-tag associations: %w", err)
	}

	// Then delete the tag
	result, err := s.db.Exec(`DELETE FROM tags WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tag not found")
	}

	return nil
}

// Merge merges multiple source tags into a target tag
func (s *TagService) Merge(request *models.TagMergeRequest) error {
	// Verify target tag exists
	_, err := s.GetByID(request.TargetTagID)
	if err != nil {
		return fmt.Errorf("target tag not found: %w", err)
	}

	// For each source tag
	for _, sourceID := range request.SourceTagIDs {
		if sourceID == request.TargetTagID {
			continue // Skip if source is same as target
		}

		// Move all video associations to target tag
		query := `
			INSERT OR IGNORE INTO video_tags (video_id, tag_id)
			SELECT video_id, ? FROM video_tags WHERE tag_id = ?
		`
		_, err := s.db.Exec(query, request.TargetTagID, sourceID)
		if err != nil {
			log.Printf("Failed to merge tag %d: %v", sourceID, err)
			continue
		}

		// Delete the source tag
		err = s.Delete(sourceID)
		if err != nil {
			log.Printf("Failed to delete source tag %d: %v", sourceID, err)
		}
	}

	return nil
}
