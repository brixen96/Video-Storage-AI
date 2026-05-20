package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	appconfig "github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// VideoMarks holds mark flags for a single video (not_interested, in_edit_list, content_type, lists).
type VideoMarks struct {
	ID            int64
	NotInterested bool
	InEditList    bool
	ContentType   string
	Lists         []VideoListInfo
}

// VideoListInfo holds basic info about a video list membership.
type VideoListInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

// updateVideoThumbnailPath sets the thumbnail_path column for a video after successful generation.
func (s *VideoService) updateVideoThumbnailPath(videoID int64, thumbnailPath string) error {
	_, err := s.db.Exec(`UPDATE videos SET thumbnail_path = ? WHERE id = ?`, thumbnailPath, videoID)
	if err != nil {
		log.Printf("Failed to update thumbnail path for video %d: %v", videoID, err)
		return err
	}
	return nil
}

// updateVideoPreviewPath sets the preview_path column for a video after successful generation.
func (s *VideoService) updateVideoPreviewPath(videoID int64, previewPath string) error {
	_, err := s.db.Exec(`UPDATE videos SET preview_path = ? WHERE id = ?`, previewPath, videoID)
	if err != nil {
		log.Printf("Failed to update preview path for video %d: %v", videoID, err)
		return err
	}
	return nil
}

// GenerateAllThumbnails generates thumbnails for every video that is missing one (or whose
// thumbnail file no longer exists on disk).
func (s *VideoService) GenerateAllThumbnails() error {
	log.Println("Starting batch video thumbnail generation...")

	activity, err := s.activityService.StartTask(
		"video_thumbnail_generation",
		"Generating thumbnails for videos without thumbnails",
		map[string]interface{}{},
	)
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	rows, err := s.db.Query(`
		SELECT id, library_id, file_path, duration, COALESCE(thumbnail_path,'')
		FROM videos
		ORDER BY id ASC
	`)
	if err != nil {
		if activity != nil {
			s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Failed: %v", err))
		}
		return fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	thumbnailBaseDir := os.Getenv("THUMBNAIL_DIR")
	if thumbnailBaseDir == "" {
		thumbnailBaseDir = filepath.Join(appconfig.DataDir(), "thumbnails")
	}

	type videoInfo struct {
		ID        int64
		LibraryID int64
		FilePath  string
		Duration  float64
	}

	var videos []videoInfo
	for rows.Next() {
		var v videoInfo
		var thumbPath string
		if err := rows.Scan(&v.ID, &v.LibraryID, &v.FilePath, &v.Duration, &thumbPath); err != nil {
			log.Printf("Failed to scan video: %v", err)
			continue
		}
		if thumbPath == "" {
			videos = append(videos, v)
			continue
		}
		fullThumbPath := filepath.Join(thumbnailBaseDir, strings.TrimPrefix(thumbPath, "thumbnails/"))
		if _, statErr := os.Stat(fullThumbPath); os.IsNotExist(statErr) {
			videos = append(videos, v)
		}
	}

	if len(videos) == 0 {
		log.Println("No videos need thumbnails")
		if activity != nil {
			s.activityService.CompleteTask(int64(activity.ID), "All videos already have thumbnails")
		}
		return nil
	}

	log.Printf("Found %d videos without thumbnails", len(videos))

	numWorkers := runtime.NumCPU()
	thumbnailJobs := make(chan videoInfo, len(videos))
	var wg sync.WaitGroup
	var generated, failed int
	var mu sync.Mutex

	mediaService := NewMediaService()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for video := range thumbnailJobs {
				library, err := s.libraryService.GetByID(video.LibraryID)
				if err != nil {
					log.Printf("Worker %d: Failed to get library for video %d: %v", workerID, video.ID, err)
					mu.Lock()
					failed++
					mu.Unlock()
					continue
				}

				config := ThumbnailConfig{
					LibraryID:     video.LibraryID,
					LibraryPath:   library.Path,
					VideoFilePath: video.FilePath,
					Duration:      video.Duration,
					ThumbnailDir:  thumbnailBaseDir,
				}

				result, err := mediaService.GenerateThumbnailHierarchical(config)
				if err != nil {
					log.Printf("Worker %d: Failed to generate thumbnail for video %d: %v", workerID, video.ID, err)
					mu.Lock()
					failed++
					mu.Unlock()
					continue
				}

				if err := s.updateVideoThumbnailPath(video.ID, result.RelativePath); err != nil {
					log.Printf("Worker %d: Failed to update thumbnail path for video %d: %v", workerID, video.ID, err)
					mu.Lock()
					failed++
					mu.Unlock()
					continue
				}

				mu.Lock()
				generated++
				if generated%10 == 0 {
					log.Printf("Progress: %d/%d thumbnails generated, %d failed", generated, len(videos), failed)
					if activity != nil {
						progress := int((float64(generated+failed) / float64(len(videos))) * 100)
						s.activityService.UpdateProgress(
							activity.ID,
							progress,
							fmt.Sprintf("Generated %d/%d thumbnails (%d failed)", generated, len(videos), failed),
						)
					}
				}
				mu.Unlock()
			}
		}(i)
	}

	for _, video := range videos {
		if activity != nil && IsActivityPaused(int64(activity.ID)) {
			log.Printf("Thumbnail generation cancelled/paused, stopping queue")
			break
		}
		thumbnailJobs <- video
	}

	close(thumbnailJobs)
	wg.Wait()

	log.Printf("Thumbnail generation complete: %d generated, %d failed", generated, failed)

	if activity != nil {
		s.activityService.CompleteTask(
			int64(activity.ID),
			fmt.Sprintf("Completed: %d thumbnails generated, %d failed", generated, failed),
		)
	}

	return nil
}

// GenerateAllPreviews generates preview storyboards for all videos across all libraries.
func (s *VideoService) GenerateAllPreviews(config ParallelScanConfig) error {
	log.Println("Starting preview generation for all videos...")

	activity, err := s.activityService.StartTask(
		"preview_generation",
		"Generating preview storyboards for all videos",
		map[string]interface{}{},
	)
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	libraries, err := s.libraryService.GetAll()
	if err != nil {
		if activity != nil {
			s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Failed: %v", err))
		}
		return fmt.Errorf("failed to get libraries: %w", err)
	}

	if len(libraries) == 0 {
		log.Println("No libraries found")
		if activity != nil {
			s.activityService.CompleteTask(int64(activity.ID), "No libraries found")
		}
		return nil
	}

	previewDir := os.Getenv("PREVIEW_DIR")
	if previewDir == "" {
		previewDir = filepath.Join(appconfig.DataDir(), "previews")
	}

	serverLibraries := []models.Library{}
	localLibraries := []models.Library{}
	for _, lib := range libraries {
		if s.isServerDrive(lib.Path, config.ServerDrives) {
			serverLibraries = append(serverLibraries, lib)
		} else if s.isLocalDrive(lib.Path, config.LocalDrives) {
			localLibraries = append(localLibraries, lib)
		} else {
			localLibraries = append(localLibraries, lib)
		}
	}

	log.Printf("Grouped libraries - Server: %d, Local: %d", len(serverLibraries), len(localLibraries))

	totalLibraries := int64(len(libraries))
	var completedLibraries int64
	onLibraryDone := func(libName string) {
		completed := atomic.AddInt64(&completedLibraries, 1)
		if activity != nil {
			progress := int(completed * 100 / totalLibraries)
			s.activityService.UpdateProgressLog(
				int64(activity.ID), progress,
				fmt.Sprintf("Completed %d/%d libraries (last: %s)", completed, totalLibraries, libName),
			)
		}
	}

	var wg sync.WaitGroup
	if len(serverLibraries) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.generatePreviewsForLibraries(serverLibraries, config.ServerMaxConcurrent, previewDir, "SERVER", onLibraryDone)
		}()
	}
	if len(localLibraries) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.generatePreviewsForLibraries(localLibraries, config.LocalMaxConcurrent, previewDir, "LOCAL", onLibraryDone)
		}()
	}

	wg.Wait()
	log.Println("All preview generation completed")

	if activity != nil {
		s.activityService.CompleteTask(int64(activity.ID), fmt.Sprintf("Completed preview generation for %d libraries", len(libraries)))
	}

	return nil
}

// generatePreviewsForLibraries processes a slice of libraries with a bounded worker pool.
func (s *VideoService) generatePreviewsForLibraries(libraries []models.Library, maxConcurrent int, previewDir string, driveType string, onDone func(libName string)) {
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, library := range libraries {
		wg.Add(1)
		sem <- struct{}{}

		go func(lib models.Library) {
			defer wg.Done()
			defer func() { <-sem }()

			log.Printf("[%s] Generating previews for library: %s (ID: %d)", driveType, lib.Name, lib.ID)
			startTime := time.Now()

			err := s.generatePreviewsForLibrary(lib.ID, previewDir)
			duration := time.Since(startTime)

			if err != nil {
				log.Printf("[%s] Failed to generate previews for library %s: %v (Duration: %s)", driveType, lib.Name, err, duration)
			} else {
				log.Printf("[%s] Successfully generated previews for library %s (Duration: %s)", driveType, lib.Name, duration)
			}
			onDone(lib.Name)
		}(library)
	}

	wg.Wait()
	log.Printf("[%s] All %d library preview generations completed", driveType, len(libraries))
}

// generatePreviewsForLibrary generates preview storyboards for all videos in a single library.
func (s *VideoService) generatePreviewsForLibrary(libraryID int64, previewDir string) error {
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return fmt.Errorf("library not found: %w", err)
	}

	query := models.VideoSearchQuery{LibraryID: libraryID, Limit: 10000}
	videos, _, err := s.GetAll(&query)
	if err != nil {
		return fmt.Errorf("failed to get videos: %w", err)
	}

	if len(videos) == 0 {
		log.Printf("No videos found in library %s", library.Name)
		return nil
	}

	log.Printf("Generating previews for %d videos in library: %s", len(videos), library.Name)

	activityService := NewActivityService()
	activityLog, err := activityService.CreateLog(&models.ActivityLogCreate{
		TaskType: "preview_generation",
		Status:   models.TaskStatusRunning,
		Message:  fmt.Sprintf("Generating previews for library: %s", library.Name),
		Progress: 0,
		Details:  map[string]interface{}{"library_id": libraryID, "library_name": library.Name, "total_videos": len(videos)},
	})
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
	}

	mediaService := NewMediaService()

	numWorkers := 2
	if runtime.NumCPU() >= 8 {
		numWorkers = 3
	}
	log.Printf("Using %d workers for preview generation (CPU cores: %d)", numWorkers, runtime.NumCPU())

	previewJobs := make(chan struct {
		video   models.Video
		library models.Library
	}, numWorkers*2)
	var wg sync.WaitGroup
	var previewMutex sync.Mutex

	generated := 0
	skipped := 0

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range previewJobs {
				if job.video.PreviewPath != "" {
					firstFrame := filepath.Join(previewDir, job.video.PreviewPath, "frame_001.jpg")
					if _, statErr := os.Stat(firstFrame); !os.IsNotExist(statErr) {
						previewMutex.Lock()
						skipped++
						previewMutex.Unlock()
						continue
					}
				}

				if job.video.Duration <= 0 {
					previewMutex.Lock()
					skipped++
					previewMutex.Unlock()
					log.Printf("Worker %d: Skipping video %s - no duration", workerID, job.video.Title)
					continue
				}

				previewConfig := PreviewConfig{
					LibraryID:      job.library.ID,
					LibraryPath:    job.library.Path,
					VideoFilePath:  job.video.FilePath,
					Duration:       job.video.Duration,
					PreviewDir:     previewDir,
					FrameCount:     10,
					ThumbnailWidth: 320,
				}

				result, err := mediaService.GeneratePreviewStoryboard(previewConfig)
				if err != nil {
					log.Printf("Worker %d: Failed to generate preview for video ID %d: %v", workerID, job.video.ID, err)
					previewMutex.Lock()
					skipped++
					previewMutex.Unlock()
				} else {
					previewMutex.Lock()
					err := s.updateVideoPreviewPath(job.video.ID, result.RelativePath)
					if err == nil {
						generated++
						if generated%10 == 0 {
							log.Printf("Progress: Generated %d previews, skipped %d", generated, skipped)
							if activityLog != nil {
								progress := int((float64(generated+skipped) / float64(len(videos))) * 100)
								activityService.UpdateProgressLog(activityLog.ID, progress,
									fmt.Sprintf("Generated %d/%d previews", generated, len(videos)))
							}
						}
					} else {
						skipped++
					}
					previewMutex.Unlock()
				}

				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	for _, video := range videos {
		previewJobs <- struct {
			video   models.Video
			library models.Library
		}{video: video, library: *library}
	}

	close(previewJobs)
	wg.Wait()

	log.Printf("Preview generation complete for library %s: %d generated, %d skipped", library.Name, generated, skipped)

	if activityLog != nil {
		activityService.CompleteTaskLog(activityLog.ID,
			fmt.Sprintf("Completed: Generated %d previews, skipped %d", generated, skipped))
	}

	return nil
}

// GetVideoMarksByPath retrieves mark flags for a video by its file path.
// Returns nil (no error) if the video is not yet in the database.
func (s *VideoService) GetVideoMarksByPath(filePath string) (*VideoMarks, error) {
	var marks VideoMarks
	var contentType sql.NullString
	err := s.db.QueryRow(`SELECT not_interested, in_edit_list, content_type FROM videos WHERE file_path = ?`, filePath).
		Scan(&marks.NotInterested, &marks.InEditList, &contentType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get video marks: %w", err)
	}
	if contentType.Valid {
		marks.ContentType = contentType.String
	}
	return &marks, nil
}

// GetVideoMarksBatch retrieves mark flags for multiple file paths in a single query.
func (s *VideoService) GetVideoMarksBatch(filePaths []string) (map[string]*VideoMarks, error) {
	if len(filePaths) == 0 {
		return make(map[string]*VideoMarks), nil
	}

	placeholders := make([]string, len(filePaths))
	args := make([]interface{}, len(filePaths))
	for i, path := range filePaths {
		placeholders[i] = "?"
		args[i] = path
	}
	placeholderStr := strings.Join(placeholders, ",")

	query := fmt.Sprintf(`
		SELECT id, file_path, not_interested, in_edit_list, content_type
		FROM videos
		WHERE file_path IN (%s)
	`, placeholderStr)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query video marks: %w", err)
	}
	defer rows.Close()

	marksMap := make(map[string]*VideoMarks)
	var videoIDs []int64
	videoIDToPath := make(map[int64]string)

	for rows.Next() {
		var filePath string
		var marks VideoMarks
		var contentType sql.NullString
		if err := rows.Scan(&marks.ID, &filePath, &marks.NotInterested, &marks.InEditList, &contentType); err != nil {
			log.Printf("Warning: Failed to scan video marks: %v", err)
			continue
		}
		if contentType.Valid {
			marks.ContentType = contentType.String
		}
		marksMap[filePath] = &marks
		videoIDs = append(videoIDs, marks.ID)
		videoIDToPath[marks.ID] = filePath
	}

	if len(videoIDs) == 0 {
		return marksMap, nil
	}

	idPlaceholders := make([]string, len(videoIDs))
	listArgs := make([]interface{}, len(videoIDs))
	for i, id := range videoIDs {
		idPlaceholders[i] = "?"
		listArgs[i] = id
	}
	idPlaceholderStr := strings.Join(idPlaceholders, ",")

	listQuery := `
		SELECT vli.video_id, vl.id, vl.name, vl.color, vl.icon
		FROM video_list_items vli
		INNER JOIN video_lists vl ON vli.list_id = vl.id
		WHERE vli.video_id IN (` + idPlaceholderStr + `)
		ORDER BY vl.position ASC
	`

	listRows, err := s.db.Query(listQuery, listArgs...)
	if err != nil {
		log.Printf("Warning: Failed to query video list membership: %v", err)
	} else {
		defer listRows.Close()
		for listRows.Next() {
			var videoID int64
			var listInfo VideoListInfo
			if err := listRows.Scan(&videoID, &listInfo.ID, &listInfo.Name, &listInfo.Color, &listInfo.Icon); err != nil {
				log.Printf("Warning: Failed to scan list info: %v", err)
				continue
			}
			if filePath, ok := videoIDToPath[videoID]; ok {
				if marks, ok := marksMap[filePath]; ok {
					marks.Lists = append(marks.Lists, listInfo)
				}
			}
		}
	}

	return marksMap, nil
}
