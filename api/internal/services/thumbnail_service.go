package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// ThumbnailService handles background thumbnail generation
type ThumbnailService struct {
	mediaService    *MediaService
	activityService *ActivityService
	libraryService  *LibraryService
}

// NewThumbnailService creates a new thumbnail service
func NewThumbnailService(mediaService *MediaService, activityService *ActivityService, libraryService *LibraryService) *ThumbnailService {
	return &ThumbnailService{
		mediaService:    mediaService,
		activityService: activityService,
		libraryService:  libraryService,
	}
}

// GenerateThumbnailsForFolder generates thumbnails for all videos in a folder
func (s *ThumbnailService) GenerateThumbnailsForFolder(libraryID int64, folderPath string) error {
	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return fmt.Errorf("failed to get library: %w", err)
	}

	// Build full path
	fullPath := filepath.Join(library.Path, folderPath)

	// Read directory
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Filter video files
	videoFiles := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if isVideoFile(ext) {
			videoFiles = append(videoFiles, entry.Name())
		}
	}

	if len(videoFiles) == 0 {
		return nil
	}

	// Get thumbnail directory
	thumbnailDir := os.Getenv("THUMBNAIL_DIR")
	if thumbnailDir == "" {
		thumbnailDir = filepath.Join("assets", "thumbnails")
	}

	// Filter out videos that already have thumbnails
	videosNeedingThumbnails := []string{}
	for _, videoFile := range videoFiles {
		videoPath := filepath.Join(fullPath, videoFile)

		// Quick check: create a minimal config to check if thumbnail exists
		// We'll extract metadata later only for videos that need thumbnails
		quickConfig := ThumbnailConfig{
			LibraryID:     libraryID,
			LibraryPath:   library.Path,
			VideoFilePath: videoPath,
			Duration:      1.0, // Placeholder, we'll use proper duration later
			ThumbnailDir:  thumbnailDir,
		}

		expectedThumbnail := s.mediaService.GetThumbnailPath(quickConfig)
		if expectedThumbnail == nil || !s.mediaService.ThumbnailExists(expectedThumbnail.FullPath) {
			videosNeedingThumbnails = append(videosNeedingThumbnails, videoFile)
		}
	}

	// If all thumbnails already exist, don't start a new activity
	if len(videosNeedingThumbnails) == 0 {
		log.Printf("All thumbnails already exist for folder: %s", folderPath)
		return nil
	}

	log.Printf("Found %d videos needing thumbnails (out of %d total)", len(videosNeedingThumbnails), len(videoFiles))

	// Create activity log for the batch
	activity, err := s.activityService.StartTask(
		"thumbnail_generation_batch",
		fmt.Sprintf("Generating thumbnails for %d videos", len(videosNeedingThumbnails)),
		map[string]interface{}{
			"library_id":  libraryID,
			"folder_path": folderPath,
			"total_count": len(videosNeedingThumbnails),
		},
	)
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	// Process videos asynchronously
	go func() {
		processed := 0
		failed := 0
		skipped := 0

		for _, videoFile := range videosNeedingThumbnails {
			videoPath := filepath.Join(fullPath, videoFile)

			// Extract metadata
			metadata, err := s.mediaService.ExtractMetadata(videoPath)
			if err != nil {
				log.Printf("Failed to extract metadata for %s: %v", videoFile, err)
				failed++
				continue
			}

			// Create thumbnail configuration with proper duration
			thumbnailConfig := ThumbnailConfig{
				LibraryID:     libraryID,
				LibraryPath:   library.Path,
				VideoFilePath: videoPath,
				Duration:      metadata.Duration,
				ThumbnailDir:  thumbnailDir,
			}

			// Generate thumbnail using hierarchical structure
			thumbnailResult, err := s.mediaService.GenerateThumbnailHierarchical(thumbnailConfig)
			if err != nil {
				log.Printf("Failed to generate thumbnail for %s: %v", videoFile, err)
				failed++
			} else if thumbnailResult != nil {
				processed++
			} else {
				skipped++
			}

			// Update activity progress
			if activity != nil {
				progress := int(float64(processed+failed+skipped) / float64(len(videosNeedingThumbnails)) * 100)
				s.activityService.UpdateProgress(
					int(activity.ID),
					progress,
					fmt.Sprintf("Processed %d/%d videos (%d failed)", processed+failed+skipped, len(videosNeedingThumbnails), failed),
				)
			}
		}

		// Complete activity
		if activity != nil {
			if failed > 0 {
				s.activityService.CompleteTask(
					int64(activity.ID),
					fmt.Sprintf("Completed with errors: %d successful, %d failed", processed, failed),
				)
			} else {
				s.activityService.CompleteTask(
					int64(activity.ID),
					fmt.Sprintf("Successfully generated %d thumbnails", processed),
				)
			}
		}
	}()

	return nil
}
