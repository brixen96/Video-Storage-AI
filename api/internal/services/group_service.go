package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// GroupService handles group-related business logic
type GroupService struct {
	db *sql.DB
}

// NewGroupService creates a new group service
func NewGroupService() *GroupService {
	return &GroupService{
		db: database.GetDB(),
	}
}

// GetAll retrieves all groups
func (s *GroupService) GetAll() ([]models.Group, error) {
	query := `
		SELECT id, studio_id, name, logo_path, description, category, metadata, created_at, updated_at
		FROM groups
		ORDER BY name ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var group models.Group
		err := rows.Scan(
			&group.ID, &group.StudioID, &group.Name, &group.LogoPath,
			&group.Description, &group.Category, &group.Metadata, &group.CreatedAt, &group.UpdatedAt,
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

	return groups, rows.Err()
}

// GetByID retrieves a single group by ID
func (s *GroupService) GetByID(id int64) (*models.Group, error) {
	query := `
		SELECT id, studio_id, name, logo_path, description, category, metadata, created_at, updated_at
		FROM groups
		WHERE id = ?
	`

	var group models.Group
	err := s.db.QueryRow(query, id).Scan(
		&group.ID, &group.StudioID, &group.Name, &group.LogoPath,
		&group.Description, &group.Category, &group.Metadata, &group.CreatedAt, &group.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}

	// Unmarshal metadata JSON
	if err := group.UnmarshalMetadata(); err != nil {
		log.Printf("Failed to unmarshal metadata for group %d: %v", group.ID, err)
	}

	return &group, nil
}

// GetByStudioID retrieves all groups for a studio
func (s *GroupService) GetByStudioID(studioID int64) ([]models.Group, error) {
	query := `
		SELECT id, studio_id, name, logo_path, description, category, metadata, created_at, updated_at
		FROM groups
		WHERE studio_id = ?
		ORDER BY name ASC
	`

	rows, err := s.db.Query(query, studioID)
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var group models.Group
		err := rows.Scan(
			&group.ID, &group.StudioID, &group.Name, &group.LogoPath,
			&group.Description, &group.Category, &group.Metadata, &group.CreatedAt, &group.UpdatedAt,
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

	return groups, rows.Err()
}

// Create creates a new group
func (s *GroupService) Create(create *models.GroupCreate) (*models.Group, error) {
	now := time.Now()

	// Default category to 'regular' if not provided
	category := create.Category
	if category == "" {
		category = "regular"
	}

	// Marshal metadata if provided
	metadataJSON := "{}"
	if create.Metadata != nil {
		group := &models.Group{MetadataObj: create.Metadata}
		if err := group.MarshalMetadata(); err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = group.Metadata
	}

	query := `
		INSERT INTO groups (studio_id, name, logo_path, description, category, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(query, create.StudioID, create.Name, create.LogoPath,
		create.Description, category, metadataJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return s.GetByID(id)
}

// Update updates an existing group
func (s *GroupService) Update(id int64, update *models.GroupUpdate) (*models.Group, error) {
	// Get existing group
	group, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if update.Name != nil {
		group.Name = *update.Name
	}
	if update.LogoPath != nil {
		group.LogoPath = *update.LogoPath
	}
	if update.Description != nil {
		group.Description = *update.Description
	}
	if update.Category != nil {
		group.Category = *update.Category
	}
	if update.Metadata != nil {
		group.MetadataObj = update.Metadata
	}

	group.UpdatedAt = time.Now()

	// Marshal metadata
	if err := group.MarshalMetadata(); err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		UPDATE groups
		SET name = ?, logo_path = ?, description = ?, category = ?, metadata = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = s.db.Exec(query, group.Name, group.LogoPath, group.Description,
		group.Category, group.Metadata, group.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	return group, nil
}

// Delete deletes a group
func (s *GroupService) Delete(id int64) error {
	result, err := s.db.Exec(`DELETE FROM groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("group not found")
	}

	return nil
}

// ResetMetadata resets group metadata to defaults
func (s *GroupService) ResetMetadata(id int64) (*models.Group, error) {
	group, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	group.MetadataObj = nil
	group.Metadata = "{}"
	group.UpdatedAt = time.Now()

	query := `UPDATE groups SET metadata = ?, updated_at = ? WHERE id = ?`
	_, err = s.db.Exec(query, group.Metadata, group.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to reset metadata: %w", err)
	}

	return group, nil
}
