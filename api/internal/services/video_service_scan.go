package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	appconfig "github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// ParallelScanConfig holds configuration for parallel library scanning
type ParallelScanConfig struct {
	ServerDrives        []string
	LocalDrives         []string
	ServerMaxConcurrent int
	LocalMaxConcurrent  int
}

// thumbnailJobHierarchical represents a queued thumbnail generation job
type thumbnailJobHierarchical struct {
	videoID int64
	config  ThumbnailConfig
}

// ScanLibrary scans a single library, adds new videos, generates thumbnails, and reconciles missing files.
func (s *VideoService) ScanLibrary(libraryID int64) error {
	consoleLogSvc := NewConsoleLogService()

	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		consoleLogSvc.LogAPI("error", "Failed to get library for scan", map[string]interface{}{
			"library_id": libraryID,
			"error":      err.Error(),
		})
		return fmt.Errorf("library not found: %w", err)
	}

	consoleLogSvc.LogAPI("info", fmt.Sprintf("Library scan started: %s", library.Name), map[string]interface{}{
		"library_id":   libraryID,
		"library_name": library.Name,
		"library_path": library.Path,
	})

	activity, err := s.activityService.StartTask(
		"video_scan",
		fmt.Sprintf("Scanning library: %s", library.Name),
		map[string]interface{}{
			"library_id":   libraryID,
			"library_name": library.Name,
			"library_path": library.Path,
		},
	)
	if err != nil {
		consoleLogSvc.LogAPI("error", "Failed to create activity log for library scan", map[string]interface{}{
			"library_id": libraryID,
			"error":      err.Error(),
		})
		return fmt.Errorf("failed to create activity log: %w", err)
	}

	videoFiles, skippedDirs, err := s.findVideoFiles(library.Path)
	if err != nil {
		failMsg := fmt.Sprintf("Failed to scan directory: %v", err)
		if len(library.Path) >= 2 && library.Path[1] == ':' {
			failMsg += " — mapped drive letters are not accessible to Windows services. Use a UNC path (\\\\server\\share\\...) or change the service logon account to your user in services.msc."
		}
		consoleLogSvc.LogAPI("error", "Failed to scan directory for videos", map[string]interface{}{
			"library_id":   libraryID,
			"library_name": library.Name,
			"library_path": library.Path,
			"error":        err.Error(),
			"hint":         "If path uses a mapped drive letter, use a UNC path instead",
		})
		if err := s.activityService.FailTask(activity.ID, failMsg); err != nil {
			log.Printf("Failed to fail task: %v", err)
		}
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	total := len(videoFiles)
	logFields := map[string]interface{}{
		"library_id":   libraryID,
		"library_name": library.Name,
		"total_files":  total,
	}
	progressMsg := fmt.Sprintf("Found %d video files", total)
	if len(skippedDirs) > 0 {
		logFields["skipped_dirs"] = skippedDirs
		progressMsg += fmt.Sprintf(" (%d folder(s) skipped — access denied, check SMB credentials)", len(skippedDirs))
	}
	consoleLogSvc.LogAPI("info", fmt.Sprintf("Found %d video files in library: %s", total, library.Name), logFields)
	if err := s.activityService.UpdateProgress(activity.ID, 0, progressMsg); err != nil {
		log.Printf("Failed to update progress: %v", err)
	}

	processed := 0
	added := 0
	skipped := 0

	mediaService := NewMediaService()

	thumbnailDir := os.Getenv("THUMBNAIL_DIR")
	if thumbnailDir == "" {
		thumbnailDir = filepath.Join(appconfig.DataDir(), "thumbnails")
	}

	numWorkers := runtime.NumCPU() * 2
	thumbnailJobs := make(chan thumbnailJobHierarchical, numWorkers*2)
	var wg sync.WaitGroup
	var thumbnailMutex sync.Mutex

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range thumbnailJobs {
				result, err := mediaService.GenerateThumbnailHierarchical(job.config)
				if err != nil {
					log.Printf("Worker %d: Failed to generate thumbnail for video ID %d: %v", workerID, job.videoID, err)
				} else {
					log.Printf("Worker %d: Generated thumbnail for video ID %d at %s", workerID, job.videoID, result.RelativePath)
					thumbnailMutex.Lock()
					s.updateVideoThumbnailPath(job.videoID, result.RelativePath)
					thumbnailMutex.Unlock()
				}
			}
		}(i)
	}

	for _, filePath := range videoFiles {
		if IsActivityPaused(int64(activity.ID)) {
			log.Printf("Library scan cancelled/paused at %d/%d files", processed, total)
			break
		}

		processed++
		progress := int((float64(processed) / float64(total)) * 100)
		currentFile := filepath.Base(filePath)
		progressMsg := fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d)\nCurrent: %s", processed, total, skipped, added, currentFile)

		exists, err := s.videoExists(filePath)
		if err == nil && exists {
			skipped++
			if err := s.activityService.UpdateProgress(activity.ID, progress, progressMsg); err != nil {
				log.Printf("Failed to update progress: %v", err)
			}
			continue
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			skipped++
			progressMsg = fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d) - Error: %v\nCurrent: %s", processed, total, skipped, added, err, currentFile)
			if err := s.activityService.UpdateProgress(activity.ID, progress, progressMsg); err != nil {
				log.Printf("Failed to update progress: %v", err)
			}
			continue
		}

		metadata, err := mediaService.ExtractMetadata(filePath)
		if err != nil {
			metadata = &VideoMetadata{
				Duration: 0,
				Size:     fileInfo.Size(),
			}
		}

		resolution := ""
		if metadata.Width > 0 && metadata.Height > 0 {
			resolution = fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
		}

		thumbnailConfig := ThumbnailConfig{
			LibraryID:     libraryID,
			LibraryPath:   library.Path,
			VideoFilePath: filePath,
			Duration:      metadata.Duration,
			ThumbnailDir:  thumbnailDir,
		}

		// thumbnail_path is intentionally empty here; the worker sets it only after successful generation
		create := &models.VideoCreate{
			LibraryID:  libraryID,
			Title:      filepath.Base(filePath),
			FilePath:   filePath,
			FileSize:   metadata.Size,
			Duration:   metadata.Duration,
			Codec:      metadata.Codec,
			Resolution: resolution,
			Bitrate:    metadata.Bitrate,
			FPS:        metadata.FrameRate,
		}

		video, err := s.Create(create)
		if err != nil {
			skipped++
			progressMsg = fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d) - Error: %v\nCurrent: %s", processed, total, skipped, added, err, currentFile)
			if err := s.activityService.UpdateProgress(activity.ID, progress, progressMsg); err != nil {
				log.Printf("Failed to update progress: %v", err)
			}
			continue
		}

		thumbnailJobs <- thumbnailJobHierarchical{videoID: video.ID, config: thumbnailConfig}

		thumbnailMutex.Lock()
		added++
		thumbnailMutex.Unlock()

		progressMsg = fmt.Sprintf("Processing %d/%d (Skipped: %d, Added: %d)\nCurrent: %s", processed, total, skipped, added, currentFile)
		if err := s.activityService.UpdateProgress(activity.ID, progress, progressMsg); err != nil {
			log.Printf("Failed to update progress: %v", err)
		}
	}

	close(thumbnailJobs)
	log.Println("Waiting for thumbnail generation workers to complete...")
	wg.Wait()
	log.Println("All thumbnail generation workers completed")

	// Reconciliation pass — mark missing / recover returned files
	if err := s.activityService.UpdateProgress(activity.ID, 99, "Reconciling file paths…"); err != nil {
		log.Printf("Failed to update progress: %v", err)
	}

	nowMissing := 0
	nowRecovered := 0
	libRows, err := s.db.Query(`SELECT id, file_path, COALESCE(is_missing,0) FROM videos WHERE library_id = ?`, libraryID)
	if err == nil {
		type libVideo struct {
			id        int64
			path      string
			isMissing bool
		}
		var libVideos []libVideo
		for libRows.Next() {
			var lv libVideo
			if libRows.Scan(&lv.id, &lv.path, &lv.isMissing) == nil {
				libVideos = append(libVideos, lv)
			}
		}
		libRows.Close()

		for _, lv := range libVideos {
			_, statErr := os.Stat(lv.path)
			fileGone := statErr != nil
			if fileGone && !lv.isMissing {
				s.db.Exec(`UPDATE videos SET is_missing = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, lv.id)
				log.Printf("[SCAN] Marked missing: %s", lv.path)
				nowMissing++
			} else if !fileGone && lv.isMissing {
				s.db.Exec(`UPDATE videos SET is_missing = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, lv.id)
				log.Printf("[SCAN] Recovered: %s", lv.path)
				nowRecovered++
			}
		}
	}

	consoleLogSvc.LogAPI("info", fmt.Sprintf("Library scan completed: %s", library.Name), map[string]interface{}{
		"library_id":     libraryID,
		"library_name":   library.Name,
		"total_files":    total,
		"videos_added":   added,
		"videos_skipped": skipped,
		"now_missing":    nowMissing,
		"now_recovered":  nowRecovered,
	})

	msg := fmt.Sprintf("Scan complete: %d added, %d skipped", added, skipped)
	if nowMissing > 0 {
		msg += fmt.Sprintf(", %d missing", nowMissing)
	}
	if nowRecovered > 0 {
		msg += fmt.Sprintf(", %d recovered", nowRecovered)
	}
	_ = s.activityService.CompleteTask(int64(activity.ID), msg)

	if OnScanComplete != nil {
		go OnScanComplete(libraryID, int(added))
	}

	return nil
}

// ScanAllLibrariesParallel scans all libraries in parallel with drive-aware concurrency limits.
func (s *VideoService) ScanAllLibrariesParallel(config ParallelScanConfig) error {
	log.Println("Starting parallel library scan with drive-aware optimization...")

	consoleLogSvc := NewConsoleLogService()

	libraries, err := s.libraryService.GetAll()
	if err != nil {
		consoleLogSvc.LogAPI("error", "Failed to get libraries for parallel scan", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to get libraries: %w", err)
	}

	if len(libraries) == 0 {
		log.Println("No libraries found to scan")
		consoleLogSvc.LogAPI("warning", "No libraries found to scan", nil)
		return nil
	}

	consoleLogSvc.LogAPI("info", fmt.Sprintf("Starting parallel scan of %d libraries", len(libraries)), map[string]interface{}{
		"library_count":         len(libraries),
		"server_max_concurrent": config.ServerMaxConcurrent,
		"local_max_concurrent":  config.LocalMaxConcurrent,
	})

	activity, err := s.activityService.StartTask(
		"library_scan_all",
		fmt.Sprintf("Scanning all %d libraries with drive-aware optimization", len(libraries)),
		map[string]interface{}{
			"library_count":         len(libraries),
			"server_max_concurrent": config.ServerMaxConcurrent,
			"local_max_concurrent":  config.LocalMaxConcurrent,
		},
	)
	if err != nil {
		log.Printf("Failed to create activity log: %v", err)
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

	if activity != nil {
		s.activityService.UpdateProgress(
			activity.ID,
			10,
			fmt.Sprintf("Grouped %d libraries (Server: %d, Local: %d)", len(libraries), len(serverLibraries), len(localLibraries)),
		)
	}

	var wg sync.WaitGroup

	if len(serverLibraries) > 0 {
		log.Printf("Starting server library scans (max concurrent: %d)...", config.ServerMaxConcurrent)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scanLibrariesConcurrent(serverLibraries, config.ServerMaxConcurrent, "SERVER")
		}()
	}

	if len(localLibraries) > 0 {
		log.Printf("Starting local library scans (max concurrent: %d)...", config.LocalMaxConcurrent)
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scanLibrariesConcurrent(localLibraries, config.LocalMaxConcurrent, "LOCAL")
		}()
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		progress := 20
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if activity != nil && progress < 90 {
					progress += 10
					s.activityService.UpdateProgress(
						activity.ID,
						progress,
						fmt.Sprintf("Scanning in progress... (%d%%)", progress),
					)
				}
			}
		}
	}()

	wg.Wait()
	close(done)
	log.Println("All parallel library scans completed")

	consoleLogSvc.LogAPI("info", fmt.Sprintf("Parallel scan completed: %d libraries", len(libraries)), map[string]interface{}{
		"library_count":    len(libraries),
		"server_libraries": len(serverLibraries),
		"local_libraries":  len(localLibraries),
	})

	if activity != nil {
		s.activityService.CompleteTask(
			int64(activity.ID),
			fmt.Sprintf("All %d libraries scanned successfully", len(libraries)),
		)
	}

	return nil
}

// scanLibrariesConcurrent scans a slice of libraries with a bounded worker pool.
func (s *VideoService) scanLibrariesConcurrent(libraries []models.Library, maxConcurrent int, driveType string) {
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, library := range libraries {
		wg.Add(1)
		sem <- struct{}{}

		go func(lib models.Library) {
			defer wg.Done()
			defer func() { <-sem }()

			log.Printf("[%s] Scanning library: %s (ID: %d, Path: %s)", driveType, lib.Name, lib.ID, lib.Path)
			startTime := time.Now()

			err := s.ScanLibrary(lib.ID)
			duration := time.Since(startTime)

			if err != nil {
				log.Printf("[%s] Failed to scan library %s: %v (Duration: %s)", driveType, lib.Name, err, duration)
			} else {
				log.Printf("[%s] Successfully scanned library %s (Duration: %s)", driveType, lib.Name, duration)
			}
		}(library)
	}

	wg.Wait()
	log.Printf("[%s] All %d library scans completed", driveType, len(libraries))
}

// isServerDrive reports whether a path resides on one of the configured server drives.
func (s *VideoService) isServerDrive(path string, serverDrives []string) bool {
	if len(path) < 2 {
		return false
	}
	drive := strings.ToUpper(path[:2])
	for _, serverDrive := range serverDrives {
		if strings.ToUpper(serverDrive) == drive {
			return true
		}
	}
	return false
}

// isLocalDrive reports whether a path resides on one of the configured local drives.
func (s *VideoService) isLocalDrive(path string, localDrives []string) bool {
	if len(path) < 2 {
		return false
	}
	drive := strings.ToUpper(path[:2])
	for _, localDrive := range localDrives {
		if strings.ToUpper(localDrive) == drive {
			return true
		}
	}
	return false
}

// findVideoFiles recursively walks dir and returns (videoFiles, skippedPaths, error).
// Inaccessible subdirectories are skipped rather than aborting the scan; only an
// inaccessible root returns a hard error.
func (s *VideoService) findVideoFiles(dir string) ([]string, []string, error) {
	var videoFiles []string
	var skippedPaths []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if path == dir {
				return err
			}
			log.Printf("[SCAN] Skipping inaccessible path %q: %v", path, err)
			skippedPaths = append(skippedPaths, path)
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, supportedExt := range SupportedVideoExtensions {
			if ext == supportedExt {
				videoFiles = append(videoFiles, path)
				break
			}
		}

		return nil
	})

	if len(skippedPaths) > 0 {
		log.Printf("[SCAN] Walk completed with %d inaccessible path(s) skipped in %s", len(skippedPaths), dir)
	}

	return videoFiles, skippedPaths, err
}

// videoExists reports whether a video with the given file path is already in the database.
func (s *VideoService) videoExists(filePath string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM videos WHERE file_path = ? LIMIT 1)`, filePath).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
