package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// BrowseService handles filesystem browsing
type BrowseService struct {
	libraryService  *LibraryService
	mediaService    *MediaService
	activityService *ActivityService
	videoService    *VideoService
}

// NewBrowseService creates a new browse service
func NewBrowseService() *BrowseService {
	activitySvc := NewActivityService()
	librarySvc := NewLibraryService()
	return &BrowseService{
		libraryService:  librarySvc,
		mediaService:    NewMediaService(),
		activityService: activitySvc,
		videoService:    NewVideoService(activitySvc, librarySvc),
	}
}

// BrowseLibrary browses a library path
func (s *BrowseService) BrowseLibrary(libraryID int64, relativePath string, extractMetadata bool) (*models.BrowseResponse, error) {
	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return nil, fmt.Errorf("library not found: %w", err)
	}

	// Construct full path
	fullPath := filepath.Join(library.Path, relativePath)

	// Normalize path
	fullPath = filepath.Clean(fullPath)

	// Security check: ensure path is within library
	if !strings.HasPrefix(fullPath, filepath.Clean(library.Path)) {
		return nil, fmt.Errorf("path traversal detected")
	}

	// Check if path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	// Read directory contents
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var items []models.BrowseItem
	for _, entry := range entries {
		itemInfo, err := entry.Info()
		if err != nil {
			continue // Skip items we can't read
		}

		// Get full path for this item
		itemFullPath := filepath.Join(fullPath, entry.Name())

		item := models.BrowseItem{
			Name:     entry.Name(),
			Path:     filepath.Join(relativePath, entry.Name()),
			FullPath: itemFullPath,
			IsDir:    entry.IsDir(),
			Size:     itemInfo.Size(),
			Modified: itemInfo.ModTime(),
		}

		// Determine type
		if entry.IsDir() {
			item.Type = "folder"
		} else {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if isVideoFile(ext) {
				item.Type = "video"
				item.Extension = ext

				// Load existing marks from database (not_interested, in_edit_list)
				if videoMarks, err := s.videoService.GetVideoMarksByPath(itemFullPath); err == nil && videoMarks != nil {
					item.NotInterested = videoMarks.NotInterested
					item.InEditList = videoMarks.InEditList
				}

				// Extract metadata if requested
				if extractMetadata {
					if metadata, err := s.mediaService.ExtractMetadata(itemFullPath); err == nil {
						item.Duration = metadata.Duration
						item.Width = metadata.Width
						item.Height = metadata.Height
						item.FrameRate = metadata.FrameRate

						// Use centralized thumbnail generation with hierarchical structure
						thumbnailDir := os.Getenv("THUMBNAIL_DIR")
						if thumbnailDir == "" {
							thumbnailDir = filepath.Join("assets", "thumbnails")
						}

						// Create thumbnail configuration
						thumbnailConfig := ThumbnailConfig{
							LibraryID:     libraryID,
							LibraryPath:   library.Path,
							VideoFilePath: itemFullPath,
							Duration:      metadata.Duration,
							ThumbnailDir:  thumbnailDir,
						}

						// Get expected thumbnail path
						expectedThumbnail := s.mediaService.GetThumbnailPath(thumbnailConfig)
						if expectedThumbnail != nil {
							// Check if thumbnail exists
							if !s.mediaService.ThumbnailExists(expectedThumbnail.FullPath) {
								// Set thumbnail path immediately (will show placeholder until generated)
								item.Thumbnail = expectedThumbnail.URLPath

								// Generate thumbnail asynchronously
								go func(config ThumbnailConfig, videoName string) {
									// Create activity log for thumbnail generation
									activity, actErr := s.activityService.StartTask(
										"thumbnail_generation",
										fmt.Sprintf("Generating thumbnail for %s", videoName),
										map[string]interface{}{
											"video_name": videoName,
											"library_id": libraryID,
											"path":       config.VideoFilePath,
										},
									)
									if actErr != nil {
										log.Printf("Failed to create thumbnail activity: %v\n", actErr)
									}

									// Generate thumbnail using centralized method
									if result, err := s.mediaService.GenerateThumbnailHierarchical(config); err == nil {
										// Mark activity as completed
										if activity != nil {
											if err := s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Generated thumbnail at %s", result.RelativePath)); err != nil {
												log.Printf("Failed to complete task: %v", err)
											}
										}
									} else {
										// Mark activity as failed
										if activity != nil {
											if err := s.activityService.FailTask(activity.ID, fmt.Sprintf("Failed to generate thumbnail: %v", err)); err != nil {
												log.Printf("Failed to fail task: %v", err)
											}
										}
									}
								}(thumbnailConfig, entry.Name())
							} else {
								// Thumbnail already exists
								item.Thumbnail = expectedThumbnail.URLPath
							}
						}
					}
				}
			} else {
				item.Type = "file"
				item.Extension = ext
			}
		}

		items = append(items, item)
	}

	// Sort: folders first, then by name
	sortBrowseItems(items)

	response := &models.BrowseResponse{
		LibraryID:   libraryID,
		LibraryName: library.Name,
		CurrentPath: relativePath,
		FullPath:    fullPath,
		Items:       items,
		TotalItems:  len(items),
		FolderCount: countByType(items, "folder"),
		VideoCount:  countByType(items, "video"),
		FileCount:   countByType(items, "file"),
	}

	return response, nil
}

// isVideoFile checks if file extension is a video format
func isVideoFile(ext string) bool {
	videoExts := map[string]bool{
		".mp4":  true,
		".mkv":  true,
		".avi":  true,
		".mov":  true,
		".wmv":  true,
		".flv":  true,
		".webm": true,
		".m4v":  true,
		".mpg":  true,
		".mpeg": true,
		".3gp":  true,
		".ts":   true,
		".m2ts": true,
	}
	return videoExts[ext]
}

// sortBrowseItems sorts items: folders first, then alphabetically
func sortBrowseItems(items []models.BrowseItem) {
	// Simple bubble sort (for small lists this is fine)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			// Folders before files
			if items[i].Type != "folder" && items[j].Type == "folder" {
				items[i], items[j] = items[j], items[i]
			} else if items[i].Type == items[j].Type {
				// Same type, sort by name
				if strings.ToLower(items[i].Name) > strings.ToLower(items[j].Name) {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	}
}

// countByType counts items by type
func countByType(items []models.BrowseItem, itemType string) int {
	count := 0
	for _, item := range items {
		if item.Type == itemType {
			count++
		}
	}
	return count
}
