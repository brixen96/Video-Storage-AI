package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// ScannerService handles filesystem scanning operations
type ScannerService struct {
	performerService *PerformerService
}

// NewScannerService creates a new scanner service
func NewScannerService() *ScannerService {
	return &ScannerService{
		performerService: NewPerformerService(),
	}
}

// ScanResult contains the results of a directory scan
type ScanResult struct {
	PerformersFound int      `json:"performers_found"`
	PerformersAdded int      `json:"performers_added"`
	VideosFound     int      `json:"videos_found"`
	Errors          []string `json:"errors,omitempty"`
}

// ScanPerformerDirectory scans the performer directory and creates performer entries
func (s *ScannerService) ScanPerformerDirectory(baseDir string) (*ScanResult, error) {
	result := &ScanResult{
		Errors: []string{},
	}

	// Check if directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("performer directory does not exist: %s", baseDir)
	}

	// Read all subdirectories (each is a performer)
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read performer directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		performerName := entry.Name()
		performerPath := filepath.Join(baseDir, performerName)
		result.PerformersFound++

		// Check if performer already exists
		existing, _ := s.performerService.GetByName(performerName)
		if existing != nil {
			// Update folder path if needed
			if existing.FolderPath.String != performerPath {
				update := &models.PerformerUpdate{
					FolderPath: &performerPath,
				}
				_, err := s.performerService.Update(existing.ID, update)
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to update %s: %v", performerName, err))
				}
			}
			continue
		}

		// Find video files in performer directory
		previewPath, videoCount := s.findPerformerVideos(performerPath)
		result.VideosFound += videoCount

		// Create performer
		create := &models.PerformerCreate{
			Name:        performerName,
			FolderPath:  performerPath,
			PreviewPath: previewPath,
		}

		_, err := s.performerService.Create(create)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create %s: %v", performerName, err))
			continue
		}

		result.PerformersAdded++
	}

	return result, nil
}

// findPerformerVideos finds video files in a performer directory
// Returns the first video as preview and total video count
func (s *ScannerService) findPerformerVideos(performerDir string) (string, int) {
	videoExtensions := map[string]bool{
		".mkv":  true,
		".mp4":  true,
		".webm": true,
		".avi":  true,
		".mov":  true,
	}

	var firstVideo string
	videoCount := 0

	entries, err := os.ReadDir(performerDir)
	if err != nil {
		return "", 0
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if videoExtensions[ext] {
			videoCount++
			if firstVideo == "" {
				firstVideo = filepath.Join(performerDir, entry.Name())
			}
		}
	}

	return firstVideo, videoCount
}

// GetPerformerVideos returns all video files for a performer
func (s *ScannerService) GetPerformerVideos(performerDir string) ([]string, error) {
	videoExtensions := map[string]bool{
		".mkv":  true,
		".mp4":  true,
		".webm": true,
		".avi":  true,
		".mov":  true,
	}

	var videos []string

	entries, err := os.ReadDir(performerDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if videoExtensions[ext] {
			videos = append(videos, filepath.Join(performerDir, entry.Name()))
		}
	}

	return videos, nil
}

// ScanVideosDirectory scans a directory for video files (for video library)
func (s *ScannerService) ScanVideosDirectory(baseDir string, recursive bool) ([]string, error) {
	videoExtensions := map[string]bool{
		".mkv":  true,
		".mp4":  true,
		".webm": true,
		".avi":  true,
		".mov":  true,
		".flv":  true,
		".wmv":  true,
	}

	var videos []string

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if !recursive && path != baseDir {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if videoExtensions[ext] {
			videos = append(videos, path)
		}

		return nil
	}

	err := filepath.Walk(baseDir, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return videos, nil
}

// ValidateVideoPath checks if a video file exists and is accessible
func (s *ScannerService) ValidateVideoPath(videoPath string) error {
	info, err := os.Stat(videoPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("video file does not exist")
	}
	if err != nil {
		return fmt.Errorf("failed to access video file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file")
	}
	return nil
}
