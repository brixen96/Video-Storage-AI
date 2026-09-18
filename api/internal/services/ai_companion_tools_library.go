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
			Description: "List the folder/directory structure of a library using database records. Use this to understand how a library is organized before suggesting placement. Use max_depth=1 for a quick overview.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name"},
				"max_depth":    map[string]interface{}{"type": "number", "description": "How many levels deep to scan (default: 1, max: 3)"},
			}, "required": []string{"library_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		maxDepth := intArg(args, "max_depth", 1)
		if maxDepth > 3 {
			maxDepth = 3
		}
		var libID int64
		var libPath string
		err := s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libPath)
		if err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}
		libPathSlash := filepath.ToSlash(libPath)

		// Build folder structure from database records — much faster than filesystem walk
		folderCounts := map[string]int{}

		rows, err := s.db.Query("SELECT file_path FROM videos WHERE library_id = ?", libID)
		if err != nil {
			return map[string]interface{}{"error": "query failed: " + err.Error()}, nil
		}
		defer rows.Close()

		for rows.Next() {
			var fp string
			if rows.Scan(&fp) != nil {
				continue
			}
			rel := filepath.ToSlash(fp)
			if strings.HasPrefix(rel, libPathSlash) {
				rel = strings.TrimPrefix(rel, libPathSlash)
				rel = strings.TrimPrefix(rel, "/")
			}
			parts := strings.Split(rel, "/")
			// Count video at each folder level up to maxDepth
			for d := 1; d <= maxDepth && d < len(parts); d++ {
				folderKey := strings.Join(parts[:d], "/")
				folderCounts[folderKey]++
			}
		}

		folders := make([]map[string]interface{}, 0, len(folderCounts))
		for path, count := range folderCounts {
			folders = append(folders, map[string]interface{}{
				"path": path, "depth": len(strings.Split(path, "/")), "video_count": count,
			})
		}
		// Sort by video count descending
		sort.Slice(folders, func(i, j int) bool {
			return folders[i]["video_count"].(int) > folders[j]["video_count"].(int)
		})
		if len(folders) > 100 {
			folders = folders[:100]
		}
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

		// Single query: join videos with performers, tags, and studios using GROUP_CONCAT
		rows, err := s.db.Query(`
			SELECT v.file_path,
			       COALESCE(GROUP_CONCAT(DISTINCT p.name), '') as performers,
			       COALESCE(GROUP_CONCAT(DISTINCT t.name), '') as tags,
			       COALESCE(GROUP_CONCAT(DISTINCT st.name), '') as studios
			FROM videos v
			LEFT JOIN video_performers vp ON v.id = vp.video_id
			LEFT JOIN performers p ON vp.performer_id = p.id
			LEFT JOIN video_tags vt ON v.id = vt.video_id
			LEFT JOIN tags t ON vt.tag_id = t.id
			LEFT JOIN video_studios vs ON v.id = vs.video_id
			LEFT JOIN studios st ON vs.studio_id = st.id
			WHERE v.library_id = ?
			GROUP BY v.id`, libID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var fp, perfCSV, tagCSV, studioCSV string
			if rows.Scan(&fp, &perfCSV, &tagCSV, &studioCSV) == nil {
				vm := videoMeta{fileDir: filepath.ToSlash(filepath.Dir(fp))}
				if perfCSV != "" {
					vm.performers = strings.Split(perfCSV, ",")
				}
				if tagCSV != "" {
					vm.tags = strings.Split(tagCSV, ",")
				}
				if studioCSV != "" {
					vm.studios = strings.Split(studioCSV, ",")
				}
				allVideos = append(allVideos, vm)
			}
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
			Name:        "review_library_structure",
			Description: "Review a library's folder layout and recommend practical structure improvements based on detected patterns, folder depth, and dominant entities.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name to review"},
				"max_depth":    map[string]interface{}{"type": "number", "description": "How many folder levels to analyze (default: 3)"},
			}, "required": []string{"library_name"}},
		}}, s.toolReviewLibraryStructure)

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

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "score_library_structure",
			Description: "Score a library's folder structure from 0-100. Evaluates consistency, naming, depth, clutter, and coverage. Returns a grade (A-F), breakdown, and specific improvement suggestions.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name to score"},
			}, "required": []string{"library_name"}},
		},
	}, s.toolScoreLibraryStructure)

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_all_library_scores",
			Description: "Score ALL libraries in one call. Returns a ranked list with grades, scores, video counts, and top issues for each. Use this for quick cross-library comparison instead of scoring each library individually.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, s.toolGetAllLibraryScores)
}

func (s *AICompanionService) toolGetAllLibraryScores(args map[string]interface{}) (interface{}, error) {
	// Get all libraries
	rows, err := s.db.Query("SELECT id, name, path FROM libraries ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type libInfo struct{ id int64; name, path string }
	var libs []libInfo
	for rows.Next() {
		var l libInfo
		if rows.Scan(&l.id, &l.name, &l.path) == nil {
			libs = append(libs, l)
		}
	}

	type libScore struct {
		Name         string         `json:"name"`
		Score        int            `json:"score"`
		Grade        string         `json:"grade"`
		Videos       int            `json:"videos"`
		Folders      int            `json:"folders"`
		MaxDepth     int            `json:"max_depth"`
		CatchAllPct  string         `json:"catch_all_pct"`
		Suggestions  []string       `json:"top_issues"`
		Scores       map[string]int `json:"scores"`
	}

	var results []libScore
	totalVideos := 0

	for _, lib := range libs {
		var videoCount int
		s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ?", lib.id).Scan(&videoCount)
		totalVideos += videoCount

		libPathSlash := sql.NullString{String: lib.path, Valid: true}
		_ = libPathSlash

		// Folder analysis
		folderMap := map[string]int{}
		var maxDepth int
		var videosAtRoot, videosInCatchAll int

		s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path = ?", lib.id, lib.path).Scan(&videosAtRoot)
		s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND (file_path LIKE ? OR file_path LIKE ?)",
			lib.id, "%/Downloads%Torrents%", "%/Users%").Scan(&videosInCatchAll)

		vRows, err := s.db.Query("SELECT file_path FROM videos WHERE library_id = ?", lib.id)
		if err == nil {
			for vRows.Next() {
				var fp string
				if vRows.Scan(&fp) != nil { continue }
				rel := fp
				if strings.HasPrefix(rel, lib.path) {
					rel = strings.TrimPrefix(rel, lib.path)
					rel = strings.TrimPrefix(rel, "/")
				}
				rel = strings.ReplaceAll(rel, "\\", "/")
				parts := strings.Split(rel, "/")
				for d := 1; d < len(parts); d++ {
					key := strings.Join(parts[:d], "/")
					folderMap[key]++
					if d > maxDepth { maxDepth = d }
				}
			}
			vRows.Close()
		}

		totalFolders := len(folderMap)
		singleVideoFolders := 0
		for _, cnt := range folderMap {
			if cnt == 1 { singleVideoFolders++ }
		}

		// Scores
		scores := map[string]int{}

		// Coverage (0-30)
		catchAllPct := 0.0
		if videoCount > 0 { catchAllPct = float64(videosInCatchAll) / float64(videoCount) * 100 }
		coverageScore := int(30 * (1 - catchAllPct/100))
		if coverageScore < 0 { coverageScore = 0 }
		scores["coverage"] = coverageScore

		// Consistency (0-25)
		consistencyScore := 25
		if maxDepth > 3 { consistencyScore -= (maxDepth - 3) * 5 }
		if totalFolders > 0 {
			singlePct := float64(singleVideoFolders) / float64(totalFolders) * 100
			if singlePct > 30 {
				consistencyScore -= 10
			} else if singlePct > 15 {
				consistencyScore -= 5
			}
		}
		if consistencyScore < 0 { consistencyScore = 0 }
		scores["consistency"] = consistencyScore

		// Naming (0-20) — simplified check
		namingScore := 20
		badNames := 0
		for name := range folderMap {
			base := filepath.Base(name)
			if len(base) > 60 { badNames++ }
		}
		if totalFolders > 0 {
			badPct := float64(badNames) / float64(totalFolders) * 100
			if badPct > 20 {
				namingScore -= 15
			} else if badPct > 10 {
				namingScore -= 10
			} else if badPct > 5 {
				namingScore -= 5
			}
		}
		if namingScore < 0 { namingScore = 0 }
		scores["naming"] = namingScore

		// Clutter (0-15) — simplified
		clutterScore := 15
		if singleVideoFolders > 20 { clutterScore -= 10 } else if singleVideoFolders > 10 { clutterScore -= 5 }
		if clutterScore < 0 { clutterScore = 0 }
		scores["clutter"] = clutterScore

		// Root (0-10)
		rootScore := 10
		if videoCount > 0 {
			rootPct := float64(videosAtRoot) / float64(videoCount) * 100
			if rootPct > 20 {
				rootScore -= 8
			} else if rootPct > 5 {
				rootScore -= 4
			}
		}
		if rootScore < 0 { rootScore = 0 }
		scores["root_cleanliness"] = rootScore

		total := 0
		for _, v := range scores { total += v }

		grade := "F"
		switch {
		case total >= 90: grade = "A"
		case total >= 80: grade = "B"
		case total >= 70: grade = "C"
		case total >= 50: grade = "D"
		}

		var issues []string
		if catchAllPct > 20 {
			issues = append(issues, fmt.Sprintf("%.0f%% in catch-all folders", catchAllPct))
		}
		if maxDepth > 3 {
			issues = append(issues, fmt.Sprintf("Nested %d levels deep", maxDepth))
		}
		if singleVideoFolders > 10 {
			issues = append(issues, fmt.Sprintf("%d single-video folders", singleVideoFolders))
		}
		if videosAtRoot > 5 {
			issues = append(issues, fmt.Sprintf("%d videos at root", videosAtRoot))
		}
		if len(issues) == 0 {
			issues = append(issues, "Looks good!")
		}

		results = append(results, libScore{
			Name: lib.name, Score: total, Grade: grade, Videos: videoCount,
			Folders: totalFolders, MaxDepth: maxDepth,
			CatchAllPct: fmt.Sprintf("%.1f%%", catchAllPct),
			Suggestions: issues, Scores: scores,
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })

	return map[string]interface{}{
		"total_videos":  totalVideos,
		"libraries":     results,
		"library_count": len(results),
	}, nil
}

func (s *AICompanionService) toolScoreLibraryStructure(args map[string]interface{}) (interface{}, error) {
	libraryName, _ := args["library_name"].(string)
	var libID int64
	var libPath, libNameFull string
	err := s.db.QueryRow("SELECT id, name, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libNameFull, &libPath)
	if err != nil {
		return map[string]interface{}{"error": "library not found: " + libraryName}, nil
	}

	// Gather metrics from DB
	var totalVideos, videosAtRoot, videosInCatchAll int
	libPathSlash := filepath.ToSlash(libPath)

	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ?", libID).Scan(&totalVideos)
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path = ?", libID, libPath).Scan(&videosAtRoot)
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND (file_path LIKE ? OR file_path LIKE ?)",
		libID, "%"+string(filepath.Separator)+"Downloads%Torrents%", "%"+string(filepath.Separator)+"Users%").Scan(&videosInCatchAll)

	// Folder analysis
	type folderInfo struct{ depth, count int; names []string }
	folderMap := map[string]*folderInfo{}
	var maxDepth int
	var totalFolders int
	var emptyFolders int

	rows, err := s.db.Query("SELECT file_path FROM videos WHERE library_id = ?", libID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var fp string
			if rows.Scan(&fp) != nil { continue }
			rel := filepath.ToSlash(fp)
			if strings.HasPrefix(rel, libPathSlash) {
				rel = strings.TrimPrefix(rel, libPathSlash)
				rel = strings.TrimPrefix(rel, "/")
			}
			parts := strings.Split(rel, "/")
			for d := 1; d < len(parts); d++ {
				folderKey := strings.Join(parts[:d], "/")
				if folderMap[folderKey] == nil {
					folderMap[folderKey] = &folderInfo{names: []string{filepath.Base(folderKey)}}
					totalFolders++
				}
				folderMap[folderKey].count++
				folderMap[folderKey].depth = d
				if d > maxDepth { maxDepth = d }
			}
		}
	}

	// Count folders with very few videos (clutter)
	singleVideoFolders := 0
	for _, fi := range folderMap {
		if fi.count == 1 { singleVideoFolders++ }
	}

	// Naming quality — check for special chars, very long names
	badNames := 0
	for name := range folderMap {
		base := filepath.Base(name)
		if len(base) > 60 { badNames++ }
		for _, ch := range []string{"|", "!", "@", "#", "&", "³", "⁄", "¦"} {
			if strings.Contains(base, ch) { badNames++; break }
		}
	}

	// New folder / empty folder detection on disk
	_ = filepath.Walk(libPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || !info.IsDir() || path == libPath { return nil }
		base := filepath.Base(path)
		if base == "New folder" || base == ".thumb" || base == "Proxy" {
			emptyFolders++
		}
		entries, _ := os.ReadDir(path)
		if len(entries) == 0 { emptyFolders++ }
		return nil
	})

	// ── Scoring ──
	scores := map[string]int{}

	// Coverage (0-30): what % of videos are NOT in catch-all folders
	catchAllPct := 0.0
	if totalVideos > 0 { catchAllPct = float64(videosInCatchAll) / float64(totalVideos) * 100 }
	coverageScore := int(30 * (1 - catchAllPct/100))
	if coverageScore < 0 { coverageScore = 0 }
	scores["coverage"] = coverageScore

	// Consistency (0-25): penalize deep nesting and single-video folders
	consistencyScore := 25
	if maxDepth > 3 { consistencyScore -= (maxDepth - 3) * 5 }
	if totalFolders > 0 {
		singlePct := float64(singleVideoFolders) / float64(totalFolders) * 100
		if singlePct > 30 {
			consistencyScore -= 10
		} else if singlePct > 15 {
			consistencyScore -= 5
		}
	}
	if consistencyScore < 0 { consistencyScore = 0 }
	scores["consistency"] = consistencyScore

	// Naming (0-20): penalize bad folder names
	namingScore := 20
	if totalFolders > 0 {
		badPct := float64(badNames) / float64(totalFolders) * 100
		if badPct > 20 {
			namingScore -= 15
		} else if badPct > 10 {
			namingScore -= 10
		} else if badPct > 5 {
			namingScore -= 5
		}
	}
	if namingScore < 0 { namingScore = 0 }
	scores["naming"] = namingScore

	// Clutter (0-15): penalize empty/New folders
	clutterScore := 15
	if emptyFolders > 10 {
		clutterScore -= 10
	} else if emptyFolders > 5 {
		clutterScore -= 5
	}
	if clutterScore < 0 { clutterScore = 0 }
	scores["clutter"] = clutterScore

	// Root cleanliness (0-10): penalize videos at library root
	rootScore := 10
	if totalVideos > 0 {
		rootPct := float64(videosAtRoot) / float64(totalVideos) * 100
		if rootPct > 20 {
			rootScore -= 8
		} else if rootPct > 5 {
			rootScore -= 4
		}
	}
	if rootScore < 0 { rootScore = 0 }
	scores["root_cleanliness"] = rootScore

	totalScore := 0
	for _, s := range scores { totalScore += s }

	// Grade
	grade := "F"
	switch {
	case totalScore >= 90: grade = "A"
	case totalScore >= 80: grade = "B"
	case totalScore >= 70: grade = "C"
	case totalScore >= 50: grade = "D"
	}

	// Suggestions
	var suggestions []string
	if catchAllPct > 20 {
		suggestions = append(suggestions, fmt.Sprintf("%.0f%% of videos are in catch-all folders — use batch_move_by_pattern to restructure", catchAllPct))
	}
	if maxDepth > 3 {
		suggestions = append(suggestions, fmt.Sprintf("Folders go %d levels deep — flatten to max 3 levels", maxDepth))
	}
	if singleVideoFolders > 10 {
		suggestions = append(suggestions, fmt.Sprintf("%d folders with only 1 video — consolidate into parent folders", singleVideoFolders))
	}
	if emptyFolders > 0 {
		suggestions = append(suggestions, fmt.Sprintf("%d empty/New folders detected — clean up with plan_structure_improvements", emptyFolders))
	}
	if badNames > 0 {
		suggestions = append(suggestions, fmt.Sprintf("%d folders with problematic names (special chars, too long) — use rename_folder", badNames))
	}
	if videosAtRoot > 5 {
		suggestions = append(suggestions, fmt.Sprintf("%d videos sitting at library root — move into subfolders", videosAtRoot))
	}
	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Library structure looks good!")
	}

	return map[string]interface{}{
		"library":       libNameFull,
		"score":         totalScore,
		"grade":         grade,
		"scores":        scores,
		"total_videos":  totalVideos,
		"total_folders": totalFolders,
		"max_depth":     maxDepth,
		"empty_folders": emptyFolders,
		"catch_all_pct": fmt.Sprintf("%.1f%%", catchAllPct),
		"suggestions":   suggestions,
	}, nil
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

func (s *AICompanionService) toolReviewLibraryStructure(args map[string]interface{}) (interface{}, error) {
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

	folders := []map[string]interface{}{}
	totalVideos := 0
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
		folders = append(folders, map[string]interface{}{
			"path": relPath,
			"depth": depth,
			"video_count": count,
		})
		totalVideos += count
		return nil
	})

	// Build summary of root organization and suggestions.
	rootCount := 0
	for _, folder := range folders {
		if folder["depth"].(int) == 1 {
			rootCount++
		}
	}

	summary := map[string]interface{}{
		"library":      libNameFull,
		"root":         libPath,
		"total_folders": len(folders),
		"root_folders":  rootCount,
		"scanned_depth": maxDepth,
		"total_videos":  totalVideos,
	}

	// Analyze organization pattern using the library analysis service.
	analysisSvc := NewLibraryAnalysisService()
	plan, err := analysisSvc.AnalyzeLibraryStructure(libID)
	var analysis map[string]interface{}
	if err == nil && plan != nil {
		analysis = map[string]interface{}{
			"library_id":    plan.LibraryID,
			"library_name":  plan.LibraryName,
			"library_root":  plan.LibraryRoot,
			"total_videos":  plan.Stats.TotalVideos,
			"content_type_changes": plan.ContentTypeChanges,
			"tag_assignments":      plan.TagAssignments,
			"performer_links":      plan.PerformerLinks,
			"stats":               plan.Stats,
		}
		summary["analysis"] = analysis
	}

	// Recommendations based on structure and depth.
	recommendations := []string{}
	if rootCount > 8 {
		recommendations = append(recommendations, "The library has many first-level folders. Consider grouping by performer, studio, or category to reduce root clutter.")
	}
	if totalVideos > 0 && len(folders) > 30 {
		recommendations = append(recommendations, "There are many folders. Standardize on 2-3 folder levels and avoid deeply nested subfolders where possible.")
	}
	if analysis != nil {
		if det, ok := analysis["detected_pattern"].(string); ok && det == "mixed" {
			recommendations = append(recommendations, "The current structure appears mixed. Choose a dominant pattern such as performer-based or tag-based organization for easier navigation.")
		}
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "The library structure looks reasonably organized. Focus on consistent naming and complete metadata so video placement remains reliable.")
	}

	summary["recommendations"] = recommendations
	summary["folders"] = folders
	return summary, nil
}
