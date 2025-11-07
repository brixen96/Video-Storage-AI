package services

import (
	"fmt"
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
}

// NewBrowseService creates a new browse service
func NewBrowseService() *BrowseService {
	return &BrowseService{
		libraryService:  NewLibraryService(),
		mediaService:    NewMediaService(),
		activityService: NewActivityService(),
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

		item := models.BrowseItem{
			Name:     entry.Name(),
			Path:     filepath.Join(relativePath, entry.Name()),
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

				// Extract metadata if requested
				if extractMetadata {
					itemFullPath := filepath.Join(fullPath, entry.Name())
					if metadata, err := s.mediaService.ExtractMetadata(itemFullPath); err == nil {
						item.Duration = metadata.Duration
						item.Width = metadata.Width
						item.Height = metadata.Height
						item.FrameRate = metadata.FrameRate

						// Generate thumbnail if it doesn't exist
						thumbnailName := strings.TrimSuffix(entry.Name(), ext) + ".jpg"

						// Build thumbnail directory path
						var thumbnailDir = os.Getenv("THUMBNAIL_DIR")
						// Build thumbnail directory path with library ID
						libraryThumbnailDir := filepath.Join(thumbnailDir, fmt.Sprintf("%d", libraryID))
                        // Build thumbnail directory path with library ID and relative path
                        if relativePath != "" {
                            libraryThumbnailDir = filepath.Join(libraryThumbnailDir, relativePath)
                        }
                        // Build full thumbnail path with library ID and relative path
                        thumbnailFullPath := filepath.Join(libraryThumbnailDir, thumbnailName)
                        
						// Create thumbnail directory if it doesn't exist
						if err := os.MkdirAll(libraryThumbnailDir, 0755); err == nil {
							// Check if thumbnail exists
							if _, err := os.Stat(thumbnailFullPath); os.IsNotExist(err) {
								// Set thumbnail path immediately (will show placeholder until generated)
								if relativePath == "" {
									item.Thumbnail = fmt.Sprintf("thumbnails/%d/%s", libraryID, thumbnailName)
								} else {
									item.Thumbnail = fmt.Sprintf("thumbnails/%d/%s/%s", libraryID, relativePath, thumbnailName)
								}

								// Generate thumbnail asynchronously
								go func(videoPath, thumbPath, videoName string, duration float64) {
									// Create activity log for thumbnail generation
									activity, actErr := s.activityService.StartTask(
										"thumbnail_generation",
										fmt.Sprintf("Generating thumbnail for %s", videoName),
										map[string]interface{}{
											"video_name":  videoName,
											"library_id":  libraryID,
											"path":        thumbPath,
										},
									)
									if actErr != nil {
										fmt.Printf("Failed to create thumbnail activity: %v\n", actErr)
									}

									// Generate thumbnail at 10% of duration or 5 seconds
									timestamp := duration * 0.1
									if timestamp > 5 {
										timestamp = 5
									}

									if err := s.mediaService.GenerateThumbnail(videoPath, thumbPath, timestamp); err == nil {
										// Mark activity as completed
										if activity != nil {
											s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Generated thumbnail for %s", videoName))
										}
									} else {
										// Mark activity as failed
										if activity != nil {
											s.activityService.FailTask(activity.ID, fmt.Sprintf("Failed to generate thumbnail: %v", err))
										}
									}
								}(itemFullPath, thumbnailFullPath, entry.Name(), metadata.Duration)
							} else {
								// Thumbnail already exists
								if relativePath == "" {
									item.Thumbnail = fmt.Sprintf("thumbnails/%d/%s", libraryID, thumbnailName)
								} else {
									item.Thumbnail = fmt.Sprintf("thumbnails/%d/%s/%s", libraryID, relativePath, thumbnailName)
								}
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
		LibraryID:    libraryID,
		LibraryName:  library.Name,
		CurrentPath:  relativePath,
		FullPath:     fullPath,
		Items:        items,
		TotalItems:   len(items),
		FolderCount:  countByType(items, "folder"),
		VideoCount:   countByType(items, "video"),
		FileCount:    countByType(items, "file"),
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
