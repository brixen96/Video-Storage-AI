package services

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

func (s *AICompanionService) registerLibraryTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_all_libraries",
			Description: "Get all video libraries with names, paths, and video counts.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rows, err := s.db.Query(`SELECT l.id, l.name, l.path, COUNT(v.id) as cnt FROM libraries l LEFT JOIN videos v ON l.id = v.library_id GROUP BY l.id`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var name, path string
			var cnt int
			if rows.Scan(&id, &name, &path, &cnt) == nil {
				results = append(results, map[string]interface{}{"id": id, "name": name, "path": path, "video_count": cnt})
			}
		}
		return results, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_library_videos",
			Description: "List recent videos in a specific library (e.g. 'Downloads - Local', 'Backup'). Returns title, path, performers, and tags.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name to search for"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max results (default: 20)"},
			}, "required": []string{"library_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		limit := intArg(args, "limit", 20)
		var libID int64
		var libPath string
		err := s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libPath)
		if err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}
		rows, err := s.db.Query(`
			SELECT v.id, v.title, COALESCE(v.thumbnail_path,''), COALESCE(v.rating,0), COALESCE(v.is_favorite,0),
			       GROUP_CONCAT(p.name, ', ') as performers
			FROM videos v
			LEFT JOIN video_performers vp ON v.id = vp.video_id
			LEFT JOIN performers p ON vp.performer_id = p.id
			WHERE v.library_id = ?
			GROUP BY v.id
			ORDER BY v.created_at DESC
			LIMIT ?`, libID, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath string
			var rating, isFav int
			var performers sql.NullString
			if rows.Scan(&id, &title, &thumbPath, &rating, &isFav, &performers) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title,
					"thumbnail_path": thumbPath,
					"rating": rating, "is_favorite": isFav > 0,
					"performers": performers.String,
				})
			}
		}
		return map[string]interface{}{"library": libraryName, "videos": results, "count": len(results)}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "scan_library_structure",
			Description: "List the folder/directory structure of a library up to 3 levels deep. Use this to understand how a library is organized before suggesting placement.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name"},
				"max_depth":    map[string]interface{}{"type": "number", "description": "How many levels deep to scan (default: 2, max: 4)"},
			}, "required": []string{"library_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		maxDepth := intArg(args, "max_depth", 2)
		if maxDepth > 4 {
			maxDepth = 4
		}
		var libID int64
		var libPath string
		err := s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libPath)
		if err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}
		folders := []map[string]interface{}{}
		_ = filepath.Walk(libPath, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || !info.IsDir() || path == libPath {
				return nil
			}
			relPath, _ := filepath.Rel(libPath, path)
			parts := strings.Split(filepath.ToSlash(relPath), "/")
			depth := len(parts)
			if depth > maxDepth {
				return filepath.SkipDir
			}
			var count int
			s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path LIKE ?",
				libID, "%"+string(filepath.Separator)+relPath+string(filepath.Separator)+"%").Scan(&count)
			var direct int
			s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path LIKE ?",
				libID, "%"+string(filepath.Separator)+relPath+string(filepath.Separator)+"%.%").Scan(&direct)
			folders = append(folders, map[string]interface{}{
				"path": filepath.ToSlash(relPath), "depth": depth, "video_count": count + direct,
			})
			return nil
		})
		return map[string]interface{}{
			"library": libraryName, "root": libPath,
			"folders": folders, "total_folders": len(folders),
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "analyze_library_structure",
			Description: "Deep-analyze a library's organizational structure by examining what performers, tags, and studios already exist in each folder. Returns the detected pattern and improvement suggestions.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name to analyze"},
				"max_depth":    map[string]interface{}{"type": "number", "description": "Max folder depth to analyze (default: 3)"},
			}, "required": []string{"library_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		maxDepth := intArg(args, "max_depth", 3)
		if maxDepth > 5 {
			maxDepth = 5
		}
		var libID int64
		var libPath, libNameFull string
		err := s.db.QueryRow("SELECT id, name, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libNameFull, &libPath)
		if err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}

		type videoMeta struct {
			fileDir    string
			performers []string
			tags       []string
			studios    []string
		}
		var allVideos []videoMeta
		rows, err := s.db.Query("SELECT id, file_path FROM videos WHERE library_id = ?", libID)
		if err != nil {
			return nil, err
		}
		type idPath struct {
			id   int64
			path string
		}
		var vidList []idPath
		for rows.Next() {
			var id int64
			var fp string
			if rows.Scan(&id, &fp) == nil {
				vidList = append(vidList, idPath{id, fp})
			}
		}
		rows.Close()

		for _, v := range vidList {
			vm := videoMeta{fileDir: filepath.ToSlash(filepath.Dir(v.path))}
			pr, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", v.id)
			for pr.Next() {
				var n string
				pr.Scan(&n)
				vm.performers = append(vm.performers, n)
			}
			pr.Close()
			tr, _ := s.db.Query("SELECT t.name FROM tags t JOIN video_tags vt ON t.id = vt.tag_id WHERE vt.video_id = ?", v.id)
			for tr.Next() {
				var n string
				tr.Scan(&n)
				vm.tags = append(vm.tags, n)
			}
			tr.Close()
			sr, _ := s.db.Query("SELECT st.name FROM studios st JOIN video_studios vs ON st.id = vs.studio_id WHERE vs.video_id = ?", v.id)
			for sr.Next() {
				var n string
				sr.Scan(&n)
				vm.studios = append(vm.studios, n)
			}
			sr.Close()
			allVideos = append(allVideos, vm)
		}

		type folderStats struct {
			videoCount      int
			performerCounts map[string]int
			tagCounts       map[string]int
			studioCounts    map[string]int
		}
		folderMap := map[string]*folderStats{}
		libPathSlash := filepath.ToSlash(libPath)

		for _, vm := range allVideos {
			rel := vm.fileDir
			if strings.HasPrefix(rel, libPathSlash) {
				rel = strings.TrimPrefix(rel, libPathSlash)
				rel = strings.TrimPrefix(rel, "/")
			}
			parts := strings.Split(rel, "/")
			if len(parts) > maxDepth {
				parts = parts[:maxDepth]
			}
			folderKey := strings.Join(parts, "/")
			if folderKey == "" {
				folderKey = "."
			}
			if folderMap[folderKey] == nil {
				folderMap[folderKey] = &folderStats{
					performerCounts: map[string]int{},
					tagCounts:       map[string]int{},
					studioCounts:    map[string]int{},
				}
			}
			fs := folderMap[folderKey]
			fs.videoCount++
			for _, p := range vm.performers {
				fs.performerCounts[p]++
			}
			for _, t := range vm.tags {
				fs.tagCounts[t]++
			}
			for _, st := range vm.studios {
				fs.studioCounts[st]++
			}
		}

		type folderAnalysis struct {
			Folder       string   `json:"folder"`
			VideoCount   int      `json:"video_count"`
			DominantType string   `json:"dominant_type"`
			TopEntities  []string `json:"top_entities"`
		}
		var analyzed []folderAnalysis
		performerPatternCount, tagPatternCount, studioPatternCount := 0, 0, 0

		for folder, fs := range folderMap {
			if folder == "." {
				continue
			}
			topPerf := topNFromMap(fs.performerCounts, 3)
			topTag := topNFromMap(fs.tagCounts, 3)
			topStudio := topNFromMap(fs.studioCounts, 3)
			maxPerf := 0
			for _, c := range fs.performerCounts {
				if c > maxPerf {
					maxPerf = c
				}
			}
			maxTag := 0
			for _, c := range fs.tagCounts {
				if c > maxTag {
					maxTag = c
				}
			}
			maxStudio := 0
			for _, c := range fs.studioCounts {
				if c > maxStudio {
					maxStudio = c
				}
			}
			domType := "mixed"
			var topEntities []string
			switch {
			case maxPerf >= maxTag && maxPerf >= maxStudio && maxPerf > 0:
				domType = "performer"
				topEntities = topPerf
				performerPatternCount++
			case maxTag >= maxPerf && maxTag >= maxStudio && maxTag > 0:
				domType = "tag"
				topEntities = topTag
				tagPatternCount++
			case maxStudio > 0:
				domType = "studio"
				topEntities = topStudio
				studioPatternCount++
			}
			analyzed = append(analyzed, folderAnalysis{
				Folder: folder, VideoCount: fs.videoCount,
				DominantType: domType, TopEntities: topEntities,
			})
		}
		sort.Slice(analyzed, func(i, j int) bool { return analyzed[i].VideoCount > analyzed[j].VideoCount })

		detectedPattern := "mixed"
		if performerPatternCount > tagPatternCount && performerPatternCount > studioPatternCount {
			detectedPattern = "performer-based (Performers/<Name>/)"
		} else if tagPatternCount > performerPatternCount && tagPatternCount > studioPatternCount {
			detectedPattern = "category-based (Categories/<Tag>/)"
		} else if studioPatternCount > 0 {
			detectedPattern = "studio-based (Studios/<Name>/)"
		}
		if len(analyzed) > 50 {
			analyzed = analyzed[:50]
		}
		return map[string]interface{}{
			"library":          libNameFull,
			"total_videos":     len(allVideos),
			"total_folders":    len(folderMap),
			"detected_pattern": detectedPattern,
			"folders":          analyzed,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "suggest_video_placement",
			Description: "Analyze a video's metadata (performers, tags, title) and suggest the best matching subfolders in the Backup library. Call scan_library_structure first if you haven't already.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id":       map[string]interface{}{"type": "number", "description": "ID of the video to place"},
				"target_library": map[string]interface{}{"type": "string", "description": "Target library name (default: 'Backup')"},
			}, "required": []string{"video_id"}},
		},
	}, s.toolSuggestVideoPlacement)

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "move_video_to_library",
			Description: "Move a video file to a subfolder in another library and update the database. Use suggest_video_placement first to get the right subfolder path.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id":         map[string]interface{}{"type": "number", "description": "ID of the video to move"},
				"target_library":   map[string]interface{}{"type": "string", "description": "Target library name (e.g. 'Backup')"},
				"target_subfolder": map[string]interface{}{"type": "string", "description": "Relative subfolder path inside target library (e.g. 'Performers/Alice'). Empty = library root."},
			}, "required": []string{"video_id", "target_library"}},
		},
	}, s.toolMoveVideoToLibrary)
}

func (s *AICompanionService) toolSuggestVideoPlacement(args map[string]interface{}) (interface{}, error) {
	videoID := int64(intArg(args, "video_id", 0))
	targetLibraryName, _ := args["target_library"].(string)
	if targetLibraryName == "" {
		targetLibraryName = "Backup"
	}

	var title, filePath string
	var srcLibID int64
	err := s.db.QueryRow("SELECT title, file_path, library_id FROM videos WHERE id = ?", videoID).Scan(&title, &filePath, &srcLibID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
	}
	var srcLibName string
	s.db.QueryRow("SELECT name FROM libraries WHERE id = ?", srcLibID).Scan(&srcLibName)

	perfRows, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", videoID)
	var performers []string
	for perfRows.Next() {
		var n string
		perfRows.Scan(&n)
		performers = append(performers, n)
	}
	perfRows.Close()

	tagRows, _ := s.db.Query("SELECT t.name FROM tags t JOIN video_tags vt ON t.id = vt.tag_id WHERE vt.video_id = ?", videoID)
	var tags []string
	for tagRows.Next() {
		var n string
		tagRows.Scan(&n)
		tags = append(tags, n)
	}
	tagRows.Close()

	studioRows, _ := s.db.Query("SELECT s.name FROM studios s JOIN video_studios vs ON s.id = vs.studio_id WHERE vs.video_id = ?", videoID)
	var studios []string
	for studioRows.Next() {
		var n string
		studioRows.Scan(&n)
		studios = append(studios, n)
	}
	studioRows.Close()

	var tgtLibID int64
	var tgtLibPath string
	err = s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+targetLibraryName+"%").Scan(&tgtLibID, &tgtLibPath)
	if err != nil {
		return map[string]interface{}{"error": "target library not found: " + targetLibraryName}, nil
	}

	learnedFolders := map[string]string{}
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

	performerFolders := map[string]string{}
	for _, p := range performers {
		var existingPath string
		err2 := s.db.QueryRow(
			`SELECT v.file_path FROM videos v
			 JOIN video_performers vp ON v.id = vp.video_id
			 JOIN performers perf ON perf.id = vp.performer_id
			 WHERE v.library_id = ? AND LOWER(perf.name) = LOWER(?)
			 LIMIT 1`, tgtLibID, p).Scan(&existingPath)
		if err2 == nil && existingPath != "" {
			rel, err3 := filepath.Rel(tgtLibPath, filepath.Dir(existingPath))
			if err3 == nil {
				performerFolders[p] = filepath.ToSlash(rel)
			}
		}
	}

	type scored struct {
		path    string
		score   float64
		reason  string
		learned bool
	}
	var candidates []scored
	_ = filepath.Walk(tgtLibPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || !info.IsDir() || path == tgtLibPath {
			return nil
		}
		relPath, _ := filepath.Rel(tgtLibPath, path)
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) > 3 {
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
		for _, p := range performers {
			if existing, ok := performerFolders[p]; ok && strings.EqualFold(existing, folderSlash) {
				sc += 15
				reasons = append(reasons, "existing videos by "+p)
			}
		}
		for _, p := range performers {
			folderBase := strings.ToLower(filepath.Base(filepath.FromSlash(folderSlash)))
			if strings.EqualFold(folderBase, p) {
				sc += 10
				reasons = append(reasons, "performer folder: "+p)
			} else if strings.Contains(folderLower, strings.ToLower(p)) && len(strings.Fields(p)) > 1 {
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
		if sc > 0 {
			candidates = append(candidates, scored{path: folderSlash, score: sc, reason: strings.Join(reasons, "; "), learned: isLearned})
		}
		return nil
	})
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	suggestions := []map[string]interface{}{}
	for i, c := range candidates {
		if i >= 5 {
			break
		}
		suggestions = append(suggestions, map[string]interface{}{
			"subfolder": c.path, "score": c.score, "reason": c.reason, "learned": c.learned,
		})
	}
	return map[string]interface{}{
		"video_id": videoID, "title": title,
		"source_library": srcLibName, "current_path": filepath.Base(filePath),
		"performers": performers, "tags": tags, "studios": studios,
		"target_library": targetLibraryName, "suggestions": suggestions,
	}, nil
}

func (s *AICompanionService) toolMoveVideoToLibrary(args map[string]interface{}) (interface{}, error) {
	videoID := int64(intArg(args, "video_id", 0))
	targetLibraryName, _ := args["target_library"].(string)
	targetSubfolder, _ := args["target_subfolder"].(string)

	var title, filePath string
	var srcLibID int64
	err := s.db.QueryRow("SELECT title, file_path, library_id FROM videos WHERE id = ?", videoID).Scan(&title, &filePath, &srcLibID)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
	}
	var srcLibPath, srcLibName string
	s.db.QueryRow("SELECT name, path FROM libraries WHERE id = ?", srcLibID).Scan(&srcLibName, &srcLibPath)

	var tgtLibID int64
	var tgtLibPath string
	err = s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+targetLibraryName+"%").Scan(&tgtLibID, &tgtLibPath)
	if err != nil {
		return map[string]interface{}{"error": "target library not found: " + targetLibraryName}, nil
	}

	srcFullPath := filePath
	if !filepath.IsAbs(filePath) {
		srcFullPath = filepath.Join(srcLibPath, filePath)
	}
	if _, statErr := os.Stat(srcFullPath); os.IsNotExist(statErr) {
		return map[string]interface{}{"error": "source file not found on disk: " + srcFullPath}, nil
	}
	filename := filepath.Base(srcFullPath)
	var tgtFullPath string
	if targetSubfolder != "" {
		tgtFullPath = filepath.Join(tgtLibPath, filepath.FromSlash(targetSubfolder), filename)
	} else {
		tgtFullPath = filepath.Join(tgtLibPath, filename)
	}
	if _, statErr := os.Stat(tgtFullPath); statErr == nil {
		return map[string]interface{}{"error": "a file with that name already exists at the destination: " + tgtFullPath}, nil
	}
	if err := os.MkdirAll(filepath.Dir(tgtFullPath), 0755); err != nil {
		return map[string]interface{}{"error": "failed to create target directory: " + err.Error()}, nil
	}
	if err := os.Rename(srcFullPath, tgtFullPath); err != nil {
		srcData, readErr := os.ReadFile(srcFullPath)
		if readErr != nil {
			return map[string]interface{}{"error": "failed to read source: " + readErr.Error()}, nil
		}
		if writeErr := os.WriteFile(tgtFullPath, srcData, 0644); writeErr != nil {
			return map[string]interface{}{"error": "failed to write target: " + writeErr.Error()}, nil
		}
		os.Remove(srcFullPath)
	}
	_, dbErr := s.db.Exec("UPDATE videos SET file_path = ?, library_id = ?, updated_at = ? WHERE id = ?",
		tgtFullPath, tgtLibID, time.Now(), videoID)
	if dbErr != nil {
		return map[string]interface{}{"warning": "file moved but DB update failed: " + dbErr.Error(), "new_path": tgtFullPath}, nil
	}

	if targetSubfolder != "" {
		perfRows, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", videoID)
		for perfRows.Next() {
			var perfName string
			if perfRows.Scan(&perfName) == nil {
				memKey := "placement_" + strings.ToLower(strings.ReplaceAll(perfName, " ", "_"))
				_ = s.SaveMemory(&models.Memory{
					Key:        memKey,
					Value:      fmt.Sprintf("Performer '%s' → %s in %s", perfName, targetSubfolder, targetLibraryName),
					Category:   "insight",
					Importance: 8,
				})
			}
		}
		perfRows.Close()
	}
	return map[string]interface{}{
		"success": true, "video_id": videoID, "title": title,
		"from_library": srcLibName, "to_library": targetLibraryName,
		"destination_path": tgtFullPath,
	}, nil
}
