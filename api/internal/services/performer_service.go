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
		db: database.DB,
	}
}

// GetAll retrieves all performers
func (s *PerformerService) GetAll() ([]models.Performer, error) {
	query := `
		SELECT id, name, preview_path, folder_path, scene_count, zoo, metadata, created_at, updated_at
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
			&p.ID, &p.Name, &p.PreviewPath, &p.FolderPath,
			&p.SceneCount, &p.Zoo, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
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
		SELECT id, name, preview_path, folder_path, scene_count, zoo, metadata, created_at, updated_at
		FROM performers
		WHERE id = ?
	`

	var p models.Performer
	err := s.db.QueryRow(query, id).Scan(
		&p.ID, &p.Name, &p.PreviewPath, &p.FolderPath,
		&p.SceneCount, &p.Zoo, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
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
		SELECT id, name, preview_path, folder_path, scene_count, zoo, metadata, created_at, updated_at
		FROM performers
		WHERE name = ?
	`

	var p models.Performer
	err := s.db.QueryRow(query, name).Scan(
		&p.ID, &p.Name, &p.PreviewPath, &p.FolderPath,
		&p.SceneCount, &p.Zoo, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
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
		PreviewPath: create.PreviewPath,
		FolderPath:  create.FolderPath,
		SceneCount:  0,
		MetadataObj: create.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Marshal metadata
	if err := performer.MarshalMetadata(); err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Insert into database
	query := `
		INSERT INTO performers (name, preview_path, folder_path, scene_count, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(
		query,
		performer.Name, performer.PreviewPath, performer.FolderPath,
		performer.SceneCount, performer.Metadata, performer.CreatedAt, performer.UpdatedAt,
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
		performer.PreviewPath = *update.PreviewPath
	}
	if update.FolderPath != nil {
		performer.FolderPath = *update.FolderPath
	}
	if update.SceneCount != nil {
		performer.SceneCount = *update.SceneCount
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
		SET name = ?, preview_path = ?, folder_path = ?, scene_count = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(
		query,
		performer.Name, performer.PreviewPath, performer.FolderPath,
		performer.SceneCount, performer.Metadata, performer.UpdatedAt, id,
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
		SELECT id, name, preview_path, folder_path, scene_count, metadata, created_at, updated_at
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
			&p.ID, &p.Name, &p.PreviewPath, &p.FolderPath,
			&p.SceneCount, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
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

// UpdateSceneCount updates the scene count for a performer
func (s *PerformerService) UpdateSceneCount(id int64) error {
	query := `
		UPDATE performers
		SET scene_count = (
			SELECT COUNT(*) FROM video_performers WHERE performer_id = ?
		)
		WHERE id = ?
	`

	_, err := s.db.Exec(query, id, id)
	if err != nil {
		return fmt.Errorf("failed to update scene count: %w", err)
	}

	return nil
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
        SELECT id, name, preview_path, folder_path, scene_count, metadata, created_at, updated_at
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
			&p.ID, &p.Name, &p.PreviewPath, &p.FolderPath,
			&p.SceneCount, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
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
        SELECT id, name, preview_path, folder_path, scene_count, metadata, created_at, updated_at
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
			&p.ID, &p.Name, &p.PreviewPath, &p.FolderPath,
			&p.SceneCount, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
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
