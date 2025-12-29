package services

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// PerformerScanService handles scanning performer asset folders
type PerformerScanService struct {
	performerService *PerformerService
	assetsPath       string
	assetsBaseDir    string // Full path to assets directory
}

// NewPerformerScanService creates a new performer scan service
func NewPerformerScanService() *PerformerScanService {
	// Get absolute path to assets directory (api/assets since server runs from api folder)
	assetsBase, err := filepath.Abs("./assets")
	if err != nil {
		// Fallback to relative path
		assetsBase = "./assets"
	}

	return &PerformerScanService{
		performerService: NewPerformerService(),
		assetsPath:       "assets/performers",
		assetsBaseDir:    assetsBase,
	}
}

// PerformerScanResult contains the results of a scan operation
type PerformerScanResult struct {
	TotalFolders int      `json:"total_folders"`
	NewCreated   int      `json:"new_created"`
	Existing     int      `json:"existing"`
	Errors       []string `json:"errors,omitempty"`
}

// ScanPerformerFolders scans the api/assets/performers directory
func (s *PerformerScanService) ScanPerformerFolders() (*PerformerScanResult, error) {
	result := &PerformerScanResult{
		Errors: []string{},
	}

	// Check if assets directory exists
	if _, err := os.Stat(s.assetsPath); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		if err := os.MkdirAll(s.assetsPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create performers directory: %w", err)
		}
		return result, nil
	}

	// Read all folders in the performers directory
	entries, err := os.ReadDir(s.assetsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read performers directory: %w", err)
	}

	// Process each folder
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		result.TotalFolders++
		performerName := entry.Name()
		folderPath := filepath.Join(s.assetsPath, performerName)

		// Check if performer exists in database
		existingPerformer, err := s.getPerformerByName(performerName)
		if err == nil && existingPerformer != nil {
			result.Existing++
			// Performer exists, update folder path if needed
			if existingPerformer.FolderPath.String != folderPath {
				existingPerformer.FolderPath = sql.NullString{String: folderPath, Valid: folderPath != ""}
				_, updateErr := s.performerService.Update(existingPerformer.ID, &models.PerformerUpdate{
					FolderPath: &folderPath,
				})
				if updateErr != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to update %s: %v", performerName, updateErr))
				}
			}
			continue
		}

		// Create new performer
		previewPath := s.findPrimaryPreview(folderPath)
		newPerformer := &models.PerformerCreate{
			Name:        performerName,
			FolderPath:  folderPath,
			PreviewPath: previewPath,
		}

		createdPerformer, createErr := s.performerService.Create(newPerformer)
		if createErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create %s: %v", performerName, createErr))
		} else {
			result.NewCreated++
			log.Printf("Created performer: %s (ID: %d)", performerName, createdPerformer.ID)
		}
	}

	return result, nil
}

// findPrimaryPreview finds the primary preview video in a performer folder
func (s *PerformerScanService) findPrimaryPreview(folderPath string) string {
	videoExts := []string{".mp4", ".mkv", ".avi", ".mov", ".webm", ".m4v"}

	// Look for a file named "preview" with video extension
	for _, ext := range videoExts {
		previewPath := filepath.Join(folderPath, "preview"+ext)
		if _, err := os.Stat(previewPath); err == nil {
			// Convert to absolute path first
			absPath, err := filepath.Abs(previewPath)
			if err != nil {
				absPath = previewPath
			}

			// Use the configured assets base directory
			relPath, err := filepath.Rel(s.assetsBaseDir, absPath)
			if err != nil {
				// Fallback to just the filename if we can't get relative path
				relPath = filepath.Base(previewPath)
			}
			// Prepend /assets/ to create the URL path
			return "/assets/" + filepath.ToSlash(relPath)
		}
	}

	// If no "preview" file, find the first video file
	var firstVideo string
	if err := filepath.WalkDir(folderPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, videoExt := range videoExts {
			if ext == videoExt {
				if firstVideo == "" {
					// Convert to absolute path first
					absPath, err := filepath.Abs(path)
					if err != nil {
						absPath = path
					}

					// Use the configured assets base directory
					relPath, err := filepath.Rel(s.assetsBaseDir, absPath)
					if err != nil {
						// Fallback to just the filename if we can't get relative path
						relPath = filepath.Base(path)
					}
					// Prepend /assets/ to create the URL path
					firstVideo = "/assets/" + filepath.ToSlash(relPath)
				}
				return filepath.SkipDir // Stop walking
			}
		}
		return nil
	}); err != nil {
		log.Printf("Error walking directory %s: %v", folderPath, err)
	}

	return firstVideo
}

// GetPerformerPreviews returns all preview videos for a performer
func (s *PerformerScanService) GetPerformerPreviews(performer *models.Performer) ([]string, error) {
	if !performer.FolderPath.Valid || performer.FolderPath.String == "" {
		return []string{}, nil
	}

	videoExts := []string{".mp4", ".mkv", ".avi", ".mov", ".webm", ".m4v"}
	previews := []string{}

	// Walk the performer's folder and find all videos
	err := filepath.WalkDir(performer.FolderPath.String, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, videoExt := range videoExts {
			if ext == videoExt {
				// Convert to absolute path first to handle relative paths
				absPath, err := filepath.Abs(path)
				if err != nil {
					absPath = path
				}

				// Use the configured assets base directory
				relPath, err := filepath.Rel(s.assetsBaseDir, absPath)
				if err != nil {
					// Fallback to just the filename if we can't get relative path
					relPath = filepath.Base(path)
				}
				// Prepend /assets/ to create the URL path
				previews = append(previews, "/assets/"+filepath.ToSlash(relPath))
				break
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan performer folder: %w", err)
	}

	return previews, nil
}

// GetByName retrieves a performer by name (helper method)
func (ps *PerformerService) GetByName(name string) (*models.Performer, error) {
	query := `
		SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
		FROM performers
		WHERE name = ?
	`

	var performer models.Performer
	err := database.GetDB().QueryRow(query, name).Scan(
		&performer.ID, &performer.Name, &performer.PreviewPath, &performer.ThumbnailPath,
		&performer.FolderPath, &performer.VideoCount, &performer.Category, &performer.Metadata,
		&performer.CreatedAt, &performer.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Unmarshal metadata
	if err := performer.UnmarshalMetadata(); err != nil {
		log.Printf("Warning: failed to unmarshal metadata for performer %s: %v", performer.Name, err)
	}

	return &performer, nil
}

// getPerformerByName retrieves a performer by name
func (s *PerformerScanService) getPerformerByName(name string) (*models.Performer, error) {
	query := `
        SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
        FROM performers
        WHERE name = ?
    `

	var performer models.Performer
	err := database.GetDB().QueryRow(query, name).Scan(
		&performer.ID,
		&performer.Name,
		&performer.PreviewPath,
		&performer.ThumbnailPath,
		&performer.FolderPath,
		&performer.VideoCount,
		&performer.Category,
		&performer.Metadata,
		&performer.CreatedAt,
		&performer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found, return nil without error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query performer: %w", err)
	}

	// Parse metadata JSON if needed
	if err := performer.UnmarshalMetadata(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &performer, nil
}
