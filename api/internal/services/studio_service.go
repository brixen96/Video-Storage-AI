package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// StudioService handles studio-related business logic
type StudioService struct {
	db *sql.DB
}

// NewStudioService creates a new studio service
func NewStudioService() *StudioService {
	return &StudioService{
		db: database.GetDB(),
	}
}

// GetAll retrieves all studios with video counts
func (s *StudioService) GetAll() ([]models.Studio, error) {
	query := `
		SELECT s.id, s.name, s.logo_path, s.description, s.founded_date, s.country,
		       s.metadata, s.created_at, s.updated_at
		FROM studios s
		ORDER BY s.name ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query studios: %w", err)
	}
	defer rows.Close()

	var studios []models.Studio
	for rows.Next() {
		var studio models.Studio
		err := rows.Scan(
			&studio.ID, &studio.Name, &studio.LogoPath, &studio.Description,
			&studio.FoundedDate, &studio.Country, &studio.Metadata,
			&studio.CreatedAt, &studio.UpdatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan studio: %v", err)
			continue
		}

		// Unmarshal metadata JSON
		if err := studio.UnmarshalMetadata(); err != nil {
			log.Printf("Failed to unmarshal metadata for studio %d: %v", studio.ID, err)
		}

		studios = append(studios, studio)
	}

	return studios, rows.Err()
}

// GetByID retrieves a single studio by ID
func (s *StudioService) GetByID(id int64) (*models.Studio, error) {
	query := `
		SELECT id, name, logo_path, description, founded_date, country,
		       metadata, created_at, updated_at
		FROM studios
		WHERE id = ?
	`

	var studio models.Studio
	err := s.db.QueryRow(query, id).Scan(
		&studio.ID, &studio.Name, &studio.LogoPath, &studio.Description,
		&studio.FoundedDate, &studio.Country, &studio.Metadata,
		&studio.CreatedAt, &studio.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("studio not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get studio: %w", err)
	}

	// Unmarshal metadata JSON
	if err := studio.UnmarshalMetadata(); err != nil {
		log.Printf("Failed to unmarshal metadata for studio %d: %v", studio.ID, err)
	}

	return &studio, nil
}

// GetWithGroups retrieves a studio with its groups
func (s *StudioService) GetWithGroups(id int64) (*models.StudioWithGroups, error) {
	studio, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Get groups for this studio
	groupQuery := `
		SELECT id, studio_id, name, logo_path, description, metadata, created_at, updated_at
		FROM groups
		WHERE studio_id = ?
		ORDER BY name ASC
	`

	rows, err := s.db.Query(groupQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var group models.Group
		err := rows.Scan(
			&group.ID, &group.StudioID, &group.Name, &group.LogoPath,
			&group.Description, &group.Metadata, &group.CreatedAt, &group.UpdatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan group: %v", err)
			continue
		}

		// Unmarshal metadata JSON
		if err := group.UnmarshalMetadata(); err != nil {
			log.Printf("Failed to unmarshal metadata for group %d: %v", group.ID, err)
		}

		groups = append(groups, group)
	}

	return &models.StudioWithGroups{
		Studio: *studio,
		Groups: groups,
	}, nil
}

// Create creates a new studio
func (s *StudioService) Create(create *models.StudioCreate) (*models.Studio, error) {
	now := time.Now()

	// Marshal metadata if provided
	metadataJSON := "{}"
	if create.Metadata != nil {
		studio := &models.Studio{MetadataObj: create.Metadata}
		if err := studio.MarshalMetadata(); err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = studio.Metadata
	}

	query := `
		INSERT INTO studios (name, logo_path, description, founded_date, country, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(query, create.Name, create.LogoPath, create.Description,
		create.FoundedDate, create.Country, metadataJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create studio: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetByID(id)
}

// Update updates an existing studio
func (s *StudioService) Update(id int64, update *models.StudioUpdate) (*models.Studio, error) {
	// Get existing studio
	studio, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if update.Name != nil {
		studio.Name = *update.Name
	}
	if update.LogoPath != nil {
		studio.LogoPath = *update.LogoPath
	}
	if update.Description != nil {
		studio.Description = *update.Description
	}
	if update.FoundedDate != nil {
		studio.FoundedDate = *update.FoundedDate
	}
	if update.Country != nil {
		studio.Country = *update.Country
	}
	if update.Metadata != nil {
		studio.MetadataObj = update.Metadata
	}

	studio.UpdatedAt = time.Now()

	// Marshal metadata
	if err := studio.MarshalMetadata(); err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE studios
		SET name = ?, logo_path = ?, description = ?, founded_date = ?, country = ?,
		    metadata = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = s.db.Exec(query, studio.Name, studio.LogoPath, studio.Description,
		studio.FoundedDate, studio.Country, studio.Metadata, studio.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update studio: %w", err)
	}

	return studio, nil
}

// Delete deletes a studio
func (s *StudioService) Delete(id int64) error {
	// First check if studio has any groups
	var groupCount int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM groups WHERE studio_id = ?`, id).Scan(&groupCount)
	if err != nil {
		return fmt.Errorf("failed to check groups: %w", err)
	}

	if groupCount > 0 {
		return fmt.Errorf("cannot delete studio with %d groups", groupCount)
	}

	// Delete the studio
	result, err := s.db.Exec(`DELETE FROM studios WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete studio: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("studio not found")
	}

	return nil
}

// ResetMetadata resets studio metadata to defaults
func (s *StudioService) ResetMetadata(id int64) (*models.Studio, error) {
	studio, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	studio.MetadataObj = nil
	studio.Metadata = "{}"
	studio.UpdatedAt = time.Now()

	query := `UPDATE studios SET metadata = ?, updated_at = ? WHERE id = ?`
	_, err = s.db.Exec(query, studio.Metadata, studio.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to reset metadata: %w", err)
	}

	return studio, nil
}
