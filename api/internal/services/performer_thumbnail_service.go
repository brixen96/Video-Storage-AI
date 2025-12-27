package services

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// PerformerThumbnailService handles performer thumbnail generation
type PerformerThumbnailService struct {
	performerService *PerformerService
	mediaService     *MediaService
	activityService  *ActivityService
	assetsBaseDir    string
}

// NewPerformerThumbnailService creates a new performer thumbnail service
func NewPerformerThumbnailService(mediaService *MediaService, activityService *ActivityService) *PerformerThumbnailService {
	// Get absolute path to assets directory
	assetsBase, err := filepath.Abs("./assets")
	if err != nil {
		assetsBase = "./assets"
	}

	return &PerformerThumbnailService{
		performerService: NewPerformerService(),
		mediaService:     mediaService,
		activityService:  activityService,
		assetsBaseDir:    assetsBase,
	}
}

// GenerateAllThumbnails generates thumbnails for all performers with preview videos
func (s *PerformerThumbnailService) GenerateAllThumbnails() error {
	// Get all performers with preview videos
	performers, err := s.getPerformersWithPreviews()
	if err != nil {
		return fmt.Errorf("failed to get performers: %w", err)
	}

	if len(performers) == 0 {
		log.Println("No performers with preview videos found")
		return nil
	}

	log.Printf("Found %d performers with preview videos", len(performers))

	// Create activity log
	activity, err := s.activityService.StartTask(
		"performer_thumbnail_generation",
		fmt.Sprintf("Generating thumbnails for %d performers", len(performers)),
		map[string]interface{}{
			"total_count": len(performers),
		},
	)
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	// Process performers asynchronously
	go func() {
		processed := 0
		failed := 0
		skipped := 0

		for _, performer := range performers {
			// Generate thumbnail
			thumbnailPath, err := s.GenerateThumbnailForPerformer(performer.ID)
			if err != nil {
				log.Printf("Failed to generate thumbnail for performer %s (ID: %d): %v", performer.Name, performer.ID, err)
				failed++
			} else if thumbnailPath != "" {
				processed++
				log.Printf("Generated thumbnail for performer: %s (ID: %d) -> %s", performer.Name, performer.ID, thumbnailPath)
			} else {
				skipped++
			}

			// Update activity progress
			if activity != nil {
				progress := int(float64(processed+failed+skipped) / float64(len(performers)) * 100)
				s.activityService.UpdateProgress(
					int(activity.ID),
					progress,
					fmt.Sprintf("Processed %d/%d performers (%d failed)", processed+failed+skipped, len(performers), failed),
				)
			}
		}

		// Complete activity
		if activity != nil {
			if failed > 0 {
				s.activityService.CompleteTask(
					int64(activity.ID),
					fmt.Sprintf("Completed with errors: %d successful, %d failed, %d skipped", processed, failed, skipped),
				)
			} else {
				s.activityService.CompleteTask(
					int64(activity.ID),
					fmt.Sprintf("Successfully generated %d thumbnails (%d skipped)", processed, skipped),
				)
			}
		}
	}()

	return nil
}

// GenerateThumbnailForPerformer generates a thumbnail for a specific performer
func (s *PerformerThumbnailService) GenerateThumbnailForPerformer(performerID int64) (string, error) {
	// Get performer
	performer, err := s.performerService.GetByID(performerID)
	if err != nil {
		return "", fmt.Errorf("failed to get performer: %w", err)
	}

	// Check if performer has a preview video
	if performer.PreviewPath == "" {
		return "", nil // Skip performers without preview videos
	}

	// Convert preview path URL to filesystem path
	// PreviewPath is like "/assets/performers/Name/preview.mp4"
	// Remove "/assets/" prefix and join with assetsBaseDir
	previewPath := performer.PreviewPath
	if len(previewPath) > 8 && previewPath[:8] == "/assets/" {
		previewPath = previewPath[8:] // Remove "/assets/" prefix
	}
	// Convert forward slashes to OS-specific path separators
	previewPath = filepath.FromSlash(previewPath)
	videoPath := filepath.Join(s.assetsBaseDir, previewPath)

	// Verify video file exists
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return "", fmt.Errorf("preview video not found: %s", videoPath)
	}

	// Create thumbnail directory for performer
	// Structure: assets/previews/{performer_id}/thumbnail.jpg
	thumbnailDir := filepath.Join(s.assetsBaseDir, "previews", fmt.Sprintf("%d", performerID))
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Full path to thumbnail file
	thumbnailFullPath := filepath.Join(thumbnailDir, "thumbnail.jpg")

	// Generate thumbnail using FFmpeg - extract frame at 2 seconds with high quality
	if err := s.generateHighQualityThumbnail(videoPath, thumbnailFullPath, 2.0); err != nil {
		return "", fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	// Calculate relative path for database storage
	relPath, err := filepath.Rel(s.assetsBaseDir, thumbnailFullPath)
	if err != nil {
		relPath = filepath.Join("previews", fmt.Sprintf("%d", performerID), "thumbnail.jpg")
	}

	// Convert to URL path
	thumbnailURL := "/assets/" + filepath.ToSlash(relPath)

	// Update performer's thumbnail_path in database
	_, err = s.performerService.Update(performerID, &models.PerformerUpdate{
		ThumbnailPath: &thumbnailURL,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update performer thumbnail path: %w", err)
	}

	return thumbnailURL, nil
}

// generateHighQualityThumbnail generates a high-quality thumbnail from a video
func (s *PerformerThumbnailService) generateHighQualityThumbnail(videoPath, outputPath string, timestamp float64) error {
	// Check if ffmpeg is available
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	// Build ffmpeg command for high-quality thumbnail
	// -ss: seek to timestamp
	// -i: input file
	// -vframes 1: extract one frame
	// -vf scale: resize to reasonable size while maintaining aspect ratio (480p height)
	// -q:v 2: high quality (1-31, lower is better)
	args := []string{
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", videoPath,
		"-vframes", "1",
		"-vf", "scale=-1:480",
		"-q:v", "2",
		"-y", // Overwrite output file
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// getPerformersWithPreviews retrieves all performers that have preview videos
func (s *PerformerThumbnailService) getPerformersWithPreviews() ([]models.Performer, error) {
	query := `
		SELECT id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at
		FROM performers
		WHERE preview_path IS NOT NULL AND preview_path != ''
		ORDER BY name
	`

	rows, err := database.GetDB().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	performers := []models.Performer{}
	for rows.Next() {
		var p models.Performer
		err := rows.Scan(
			&p.ID, &p.Name, &p.PreviewPath, &p.ThumbnailPath,
			&p.FolderPath, &p.VideoCount, &p.Category, &p.Metadata,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Unmarshal metadata
		if err := p.UnmarshalMetadata(); err != nil {
			log.Printf("Warning: failed to unmarshal metadata for performer %s: %v", p.Name, err)
		}

		performers = append(performers, p)
	}

	return performers, rows.Err()
}

// RegenerateThumbnailOnPreviewChange regenerates thumbnail when preview path changes
func (s *PerformerThumbnailService) RegenerateThumbnailOnPreviewChange(performerID int64, oldPreviewPath, newPreviewPath string) error {
	// Only regenerate if the preview path actually changed
	if oldPreviewPath == newPreviewPath {
		return nil
	}

	// Generate new thumbnail
	_, err := s.GenerateThumbnailForPerformer(performerID)
	if err != nil {
		return fmt.Errorf("failed to regenerate thumbnail: %w", err)
	}

	return nil
}

// Helper function to update performer with thumbnail regeneration check
func (ps *PerformerService) UpdateWithThumbnailCheck(id int64, update *models.PerformerUpdate) (*models.Performer, error) {
	// Get current performer to check if preview path is changing
	current, err := ps.GetByID(id)
	if err != nil {
		return nil, err
	}

	oldPreviewPath := current.PreviewPath

	// Perform the update
	performer, err := ps.Update(id, update)
	if err != nil {
		return nil, err
	}

	// If preview path changed, regenerate thumbnail
	if update.PreviewPath != nil && *update.PreviewPath != oldPreviewPath {
		// Create thumbnail service and regenerate
		thumbnailService := NewPerformerThumbnailService(NewMediaService(), NewActivityService())
		go func() {
			err := thumbnailService.RegenerateThumbnailOnPreviewChange(id, oldPreviewPath, *update.PreviewPath)
			if err != nil {
				log.Printf("Failed to regenerate thumbnail for performer %d: %v", id, err)
			} else {
				log.Printf("Successfully regenerated thumbnail for performer %d", id)
			}
		}()
	}

	return performer, nil
}
