package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// LibraryService handles library-related business logic
type LibraryService struct {
	db *sql.DB
}

// NewLibraryService creates a new library service
func NewLibraryService() *LibraryService {
	return &LibraryService{
		db: database.DB,
	}
}

// GetAll retrieves all libraries
func (s *LibraryService) GetAll() ([]models.Library, error) {
	query := `
		SELECT id, name, path, primary_lib, created_at, updated_at
		FROM libraries
		ORDER BY primary_lib DESC, name ASC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query libraries: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var libraries []models.Library
	for rows.Next() {
		var lib models.Library
		err := rows.Scan(&lib.ID, &lib.Name, &lib.Path, &lib.Primary, &lib.CreatedAt, &lib.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan library: %w", err)
		}
		libraries = append(libraries, lib)
	}

	return libraries, nil
}

// GetByID retrieves a library by ID
func (s *LibraryService) GetByID(id int64) (*models.Library, error) {
	query := `
		SELECT id, name, path, primary_lib, created_at, updated_at
		FROM libraries
		WHERE id = ?
	`

	var lib models.Library
	err := s.db.QueryRow(query, id).Scan(
		&lib.ID, &lib.Name, &lib.Path, &lib.Primary, &lib.CreatedAt, &lib.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("library not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query library: %w", err)
	}

	return &lib, nil
}

// GetByName retrieves a library by name
func (s *LibraryService) GetByName(name string) (*models.Library, error) {
	query := `
		SELECT id, name, path, primary_lib, created_at, updated_at
		FROM libraries
		WHERE name = ?
	`

	var lib models.Library
	err := s.db.QueryRow(query, name).Scan(
		&lib.ID, &lib.Name, &lib.Path, &lib.Primary, &lib.CreatedAt, &lib.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("library not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query library: %w", err)
	}

	return &lib, nil
}

// GetPrimary retrieves the primary library
func (s *LibraryService) GetPrimary() (*models.Library, error) {
	query := `
		SELECT id, name, path, primary_lib, created_at, updated_at
		FROM libraries
		WHERE primary_lib = 1
		LIMIT 1
	`

	var lib models.Library
	err := s.db.QueryRow(query).Scan(
		&lib.ID, &lib.Name, &lib.Path, &lib.Primary, &lib.CreatedAt, &lib.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no primary library found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query primary library: %w", err)
	}

	return &lib, nil
}

// Create creates a new library
func (s *LibraryService) Create(create *models.LibraryCreate) (*models.Library, error) {
	// Check if library already exists
	existing, _ := s.GetByName(create.Name)
	if existing != nil {
		return nil, fmt.Errorf("library with name '%s' already exists", create.Name)
	}

	// Normalize path for the current OS
	normalizedPath := filepath.Clean(create.Path)

	// Validate path exists
	if _, err := os.Stat(normalizedPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("path does not exist: %s", normalizedPath)
	}

	// Use normalized path
	create.Path = normalizedPath

	// If setting this as primary, unset all other libraries as primary
	if create.Primary {
		_, err := s.db.Exec(`UPDATE libraries SET primary_lib = 0 WHERE primary_lib = 1`)
		if err != nil {
			return nil, fmt.Errorf("failed to unset existing primary library: %w", err)
		}
	}

	library := &models.Library{
		Name:      create.Name,
		Path:      create.Path,
		Primary:   create.Primary,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO libraries (name, path, primary_lib, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query, library.Name, library.Path, library.Primary, library.CreatedAt, library.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert library: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	library.ID = id
	return library, nil
}

// Update updates an existing library
func (s *LibraryService) Update(id int64, update *models.LibraryUpdate) (*models.Library, error) {
	// Get existing library
	library, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if update.Name != nil {
		library.Name = *update.Name
	}
	if update.Path != nil {
		// Normalize path for the current OS
		normalizedPath := filepath.Clean(*update.Path)

		// Validate new path exists
		if _, err := os.Stat(normalizedPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", normalizedPath)
		}
		library.Path = normalizedPath
	}
	if update.Primary != nil {
		// If setting this library as primary, unset all other libraries
		if *update.Primary {
			_, err := s.db.Exec(`UPDATE libraries SET primary_lib = 0 WHERE primary_lib = 1 AND id != ?`, id)
			if err != nil {
				return nil, fmt.Errorf("failed to unset existing primary library: %w", err)
			}
		}
		library.Primary = *update.Primary
	}

	library.UpdatedAt = time.Now()

	query := `
		UPDATE libraries
		SET name = ?, path = ?, primary_lib = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = s.db.Exec(query, library.Name, library.Path, library.Primary, library.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update library: %w", err)
	}

	return library, nil
}

// Delete deletes a library
func (s *LibraryService) Delete(id int64) error {
	// Check if library exists
	_, err := s.GetByID(id)
	if err != nil {
		return err
	}

	query := `DELETE FROM libraries WHERE id = ?`
	_, err = s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete library: %w", err)
	}

	return nil
}

// GetWithStats retrieves a library with statistics
func (s *LibraryService) GetWithStats(id int64) (*models.LibraryWithStats, error) {
	library, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	stats := &models.LibraryWithStats{
		Library: *library,
	}

	// Get video count
	var videoCount int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE library_id = ?`, id).Scan(&videoCount)
	if err != nil {
		videoCount = 0
	}
	stats.VideoCount = videoCount

	return stats, nil
}

// GetVideoCount returns the number of videos in a library
func (s *LibraryService) GetVideoCount(id int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM videos WHERE library_id = ?`
	err := s.db.QueryRow(query, id).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get video count: %w", err)
	}
	return count, nil
}
