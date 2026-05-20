package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// startFileWatchers initializes file system watchers for all libraries
func (s *AICompanionService) startFileWatchers() error {
	// Get all libraries from database
	rows, err := s.db.Query(`SELECT id, name, path FROM libraries`)
	if err != nil {
		return fmt.Errorf("failed to query libraries: %w", err)
	}
	defer rows.Close()

	libraries := []struct {
		ID   int64
		Name string
		Path string
	}{}

	for rows.Next() {
		var lib struct {
			ID   int64
			Name string
			Path string
		}
		if err := rows.Scan(&lib.ID, &lib.Name, &lib.Path); err != nil {
			log.Printf("Error scanning library: %v", err)
			continue
		}
		libraries = append(libraries, lib)
	}

	// Start watcher for each library
	for _, lib := range libraries {
		if err := s.addLibraryWatcher(lib.ID, lib.Name, lib.Path); err != nil {
			log.Printf("Warning: Failed to add watcher for library '%s': %v", lib.Name, err)
		} else {
			log.Printf("📂 Watching library: %s (%s)", lib.Name, lib.Path)
		}
	}

	return nil
}

// addLibraryWatcher adds a file watcher for a specific library
func (s *AICompanionService) addLibraryWatcher(id int64, name, path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Add the library path to the watcher
	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch path: %w", err)
	}

	libKey := fmt.Sprintf("%d", id)
	s.watchers[libKey] = watcher

	// Start monitoring this watcher
	go s.watchLibrary(watcher, name, path)

	return nil
}

// watchLibrary monitors file system events for a library
func (s *AICompanionService) watchLibrary(watcher *fsnotify.Watcher, libraryName, libraryPath string) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			s.handleFileEvent(event, libraryName, libraryPath)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error for %s: %v", libraryName, err)
		}
	}
}

// handleFileEvent processes file system events
func (s *AICompanionService) handleFileEvent(event fsnotify.Event, libraryName, libraryPath string) {
	// Filter for video files only
	ext := strings.ToLower(filepath.Ext(event.Name))
	videoExts := map[string]bool{
		".mp4": true, ".mkv": true, ".avi": true, ".mov": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
	}

	if !videoExts[ext] {
		return
	}

	var eventType string
	var message string

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = "file_added"
		message = fmt.Sprintf("New video detected: %s", filepath.Base(event.Name))
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = "file_removed"
		message = fmt.Sprintf("Video removed: %s", filepath.Base(event.Name))
	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = "file_modified"
		message = fmt.Sprintf("Video modified: %s", filepath.Base(event.Name))
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = "file_renamed"
		message = fmt.Sprintf("Video renamed: %s", filepath.Base(event.Name))
	default:
		return // Ignore other events
	}

	// Emit event
	s.emitEvent(CompanionEvent{
		Type:   eventType,
		Source: "file_watcher",
		Message: message,
		Data: map[string]interface{}{
			"library":   libraryName,
			"file_path": event.Name,
			"operation": event.Op.String(),
		},
		Severity:  "info",
		Timestamp: time.Now(),
	})

	// Autonomous decision: If new file, trigger analysis after short delay
	if eventType == "file_added" {
		go s.analyzeNewFile(event.Name, libraryName)
	}
}

// analyzeNewFile generates proactive placement suggestions when a video lands in a Downloads library
func (s *AICompanionService) analyzeNewFile(filePath, libraryName string) {
	// Wait for file to stabilize
	time.Sleep(5 * time.Second)

	filename := filepath.Base(filePath)

	// Only generate placement suggestions for download-type libraries
	if !strings.Contains(strings.ToLower(libraryName), "download") {
		s.emitEvent(CompanionEvent{
			Type:      "insight",
			Source:    "analysis_engine",
			Message:   fmt.Sprintf("New file in **%s**: `%s`", libraryName, filename),
			Severity:  "info",
			Timestamp: time.Now(),
		})
		return
	}

	// Find Backup library
	var tgtLibID int64
	var tgtLibPath, tgtLibName string
	if err := s.db.QueryRow("SELECT id, name, path FROM libraries WHERE LOWER(name) LIKE ?", "%backup%").Scan(&tgtLibID, &tgtLibName, &tgtLibPath); err != nil {
		s.emitEvent(CompanionEvent{
			Type:      "insight",
			Source:    "analysis_engine",
			Message:   fmt.Sprintf("🔥 **New download**: `%s`\n\nNo Backup library configured.", filename),
			Severity:  "info",
			Timestamp: time.Now(),
		})
		return
	}

	// Poll DB for up to 30s — the library scanner may not have run yet
	var videoID int64
	var performers, tags, studios []string
	for i := 0; i < 6; i++ {
		if i > 0 {
			time.Sleep(5 * time.Second)
		}
		var vid int64
		if s.db.QueryRow("SELECT id FROM videos WHERE file_path = ?", filePath).Scan(&vid) == nil && vid > 0 {
			videoID = vid
			perfRows, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", videoID)
			for perfRows.Next() {
				var n string
				perfRows.Scan(&n)
				performers = append(performers, n)
			}
			perfRows.Close()
			tagRows, _ := s.db.Query("SELECT t.name FROM tags t JOIN video_tags vt ON t.id = vt.tag_id WHERE vt.video_id = ?", videoID)
			for tagRows.Next() {
				var n string
				tagRows.Scan(&n)
				tags = append(tags, n)
			}
			tagRows.Close()
			studioRows, _ := s.db.Query("SELECT st.name FROM studios st JOIN video_studios vs ON st.id = vs.studio_id WHERE vs.video_id = ?", videoID)
			for studioRows.Next() {
				var n string
				studioRows.Scan(&n)
				studios = append(studios, n)
			}
			studioRows.Close()
			break
		}
	}

	// Check learned placement memories for known performers
	learnedFolders := map[string]string{} // subfolder → reason
	for _, p := range performers {
		if mems, err := s.SearchMemories("placement " + p); err == nil {
			for _, mem := range mems {
				if strings.Contains(mem.Value, "→") {
					parts := strings.SplitN(mem.Value, "→", 2)
					if len(parts) == 2 {
						subfolder := strings.TrimSpace(strings.Split(parts[1], " in ")[0])
						learnedFolders[subfolder] = "✅ learned: " + p
					}
				}
			}
		}
	}

	// Score Backup folders against metadata + filename
	type candidate struct {
		path    string
		score   float64
		reason  string
		learned bool
	}
	var candidates []candidate

	// Build a map of performer → existing folder in Backup (content-based, highest confidence)
	// This finds folders that ALREADY contain videos by the same performer
	performerFolders := map[string]string{} // performer name → subfolder
	for _, p := range performers {
		var existingPath string
		err := s.db.QueryRow(
			`SELECT v.file_path FROM videos v
			 JOIN video_performers vp ON v.id = vp.video_id
			 JOIN performers perf ON perf.id = vp.performer_id
			 WHERE v.library_id = ? AND LOWER(perf.name) = LOWER(?)
			 LIMIT 1`, tgtLibID, p).Scan(&existingPath)
		if err == nil && existingPath != "" {
			rel, err2 := filepath.Rel(tgtLibPath, filepath.Dir(existingPath))
			if err2 == nil {
				performerFolders[p] = filepath.ToSlash(rel)
			}
		}
	}

	_ = filepath.Walk(tgtLibPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || !info.IsDir() || path == tgtLibPath {
			return nil
		}
		relPath, _ := filepath.Rel(tgtLibPath, path)
		if len(strings.Split(filepath.ToSlash(relPath), "/")) > 3 {
			return filepath.SkipDir
		}
		folderSlash := filepath.ToSlash(relPath)
		folderLower := strings.ToLower(folderSlash)
		sc := 0.0
		var reasons []string
		isLearned := false

		if reason, ok := learnedFolders[folderSlash]; ok {
			sc += 20
			reasons = append(reasons, reason)
			isLearned = true
		}
		// Tier 1: folder already contains videos by this performer (content-based)
		for _, p := range performers {
			if existing, ok := performerFolders[p]; ok && strings.EqualFold(existing, folderSlash) {
				sc += 15
				reasons = append(reasons, "existing videos by "+p)
			}
		}
		// Tier 2: exact full performer name in folder path
		for _, p := range performers {
			folderBase := strings.ToLower(filepath.Base(filepath.FromSlash(folderSlash)))
			if strings.EqualFold(folderBase, p) {
				sc += 10
				reasons = append(reasons, "performer folder: "+p)
			} else if strings.Contains(folderLower, strings.ToLower(p)) && len(strings.Fields(p)) > 1 {
				// Only match multi-word full names (e.g. "Nicole Aniston") in path, not single words
				sc += 8
				reasons = append(reasons, "performer in path: "+p)
			}
		}
		for _, st := range studios {
			if strings.Contains(folderLower, strings.ToLower(st)) {
				sc += 7
				reasons = append(reasons, "studio: "+st)
			}
		}
		for _, t := range tags {
			if strings.Contains(folderLower, strings.ToLower(t)) {
				sc += 4
				reasons = append(reasons, "tag: "+t)
			}
		}
		// No filename word matching — that leads to false positives

		if sc > 0 {
			candidates = append(candidates, candidate{path: folderSlash, score: sc, reason: strings.Join(reasons, "; "), learned: isLearned})
		}
		return nil
	})

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	if len(candidates) > 3 {
		candidates = candidates[:3]
	}

	// Format message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("🔥 **New download detected**: `%s`\n\n", filename))
	if len(performers) > 0 {
		msg.WriteString(fmt.Sprintf("**Performers**: %s\n\n", strings.Join(performers, ", ")))
	}
	if len(candidates) > 0 {
		msg.WriteString(fmt.Sprintf("**Suggested placements in %s:**\n", tgtLibName))
		for i, c := range candidates {
			msg.WriteString(fmt.Sprintf("%d. `%s` — %s\n", i+1, c.path, c.reason))
		}
		if videoID != 0 {
			msg.WriteString(fmt.Sprintf("\n*Video ID: **%d*** — tell me which number to move it to.", videoID))
		} else {
			msg.WriteString("\n*Not indexed yet* — scan the Downloads library first, then ask me to move it.")
		}
	} else {
		msg.WriteString(fmt.Sprintf("No matching folders found in **%s**.\n", tgtLibName))
		msg.WriteString("Ask me to scan its structure to see what's available.")
	}

	eventData := map[string]interface{}{"file_path": filePath, "library": libraryName}
	if videoID != 0 {
		eventData["video_id"] = videoID
	}
	s.emitEvent(CompanionEvent{
		Type:      "suggestion",
		Source:    "analysis_engine",
		Message:   msg.String(),
		Data:      eventData,
		Severity:  "info",
		Timestamp: time.Now(),
	})
}

// processEvents processes events and broadcasts to subscribers
func (s *AICompanionService) processEvents() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.eventChan:
			s.mu.Lock()
			s.eventsProcessed++
			s.mu.Unlock()

			// Broadcast to all subscribers
			s.mu.RLock()
			for _, subscriber := range s.subscribers {
				// Use select with default to avoid blocking
				// Also wrap in recover to handle closed channel panics
				func() {
					defer func() {
						if r := recover(); r != nil {
							// Channel was closed, ignore
						}
					}()
					select {
					case subscriber <- event:
					default:
						// Skip if subscriber channel is full
					}
				}()
			}
			s.mu.RUnlock()

			// Log important events
			if event.Severity == "warning" || event.Severity == "error" || event.Severity == "critical" {
				log.Printf("[AI Companion] %s: %s", strings.ToUpper(event.Severity), event.Message)
			}
		}
	}
}

// emitEvent sends an event to the event channel
func (s *AICompanionService) emitEvent(event CompanionEvent) {
	select {
	case s.eventChan <- event:
	default:
		log.Printf("Event channel full, dropping event: %s", event.Message)
	}
}
