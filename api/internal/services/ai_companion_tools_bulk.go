package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// ── bulk_move_videos ──────────────────────────────────────────────────────────
// Moves multiple videos to a target subfolder within a library in a single call.
// Cap at 50 per call to stay within streaming timeout limits.

func (s *AICompanionService) registerBulkMoveTool() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "bulk_move_videos",
			Description: `Move up to 50 videos in one call. For each video provide its ID and target subfolder (relative to the library root). Returns a summary of successes/failures. Use dry_run=true first to preview.`,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"moves": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"video_id": map[string]interface{}{
									"type":        "number",
									"description": "ID of the video to move",
								},
								"target_subfolder": map[string]interface{}{
									"type":        "string",
									"description": "Relative subfolder inside the library (e.g. 'Performers/Alice')",
								},
							},
							"required": []string{"video_id", "target_subfolder"},
						},
						"description": "List of {video_id, target_subfolder} moves",
					},
					"target_library": map[string]interface{}{
						"type":        "string",
						"description": "Target library name (default: same library the video is already in)",
					},
					"dry_run": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, simulate without moving files (default: true)",
					},
				},
				"required": []string{"moves"},
			},
		},
	}, s.toolBulkMoveVideos)
}

type moveResult struct {
	VideoID   int64  `json:"video_id"`
	Title     string `json:"title"`
	Status    string `json:"status"` // success, skipped, failed
	Source    string `json:"source,omitempty"`
	Target    string `json:"target,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (s *AICompanionService) toolBulkMoveVideos(args map[string]interface{}) (interface{}, error) {
	movesRaw, ok := args["moves"].([]interface{})
	if !ok || len(movesRaw) == 0 {
		return map[string]interface{}{"error": "moves array is required"}, nil
	}
	if len(movesRaw) > 50 {
		movesRaw = movesRaw[:50]
	}

	targetLibName, _ := args["target_library"].(string)
	dryRun := true
	if v, ok := args["dry_run"].(bool); ok {
		dryRun = v
	}
	if v, ok := args["dry_run"].(string); ok && v == "false" {
		dryRun = false
	}

	results := make([]moveResult, 0, len(movesRaw))
	successCount, failCount, skipCount := 0, 0, 0

	for _, m := range movesRaw {
		move, _ := m.(map[string]interface{})
		videoID := int64(floatArg(move, "video_id", 0))
		targetSub, _ := move["target_subfolder"].(string)

		if videoID == 0 || targetSub == "" {
			skipCount++
			results = append(results, moveResult{VideoID: videoID, Status: "skipped", Error: "missing video_id or target_subfolder"})
			continue
		}

		// Resolve source
		var filePath string
		var srcLibID int64
		err := s.db.QueryRow("SELECT file_path, library_id FROM videos WHERE id = ?", videoID).Scan(&filePath, &srcLibID)
		if err != nil {
			failCount++
			results = append(results, moveResult{VideoID: videoID, Status: "failed", Error: "video not found"})
			continue
		}

		// Resolve target library
		var tgtLibID int64
		var tgtLibPath string
		if targetLibName != "" {
			err = s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+targetLibName+"%").Scan(&tgtLibID, &tgtLibPath)
		} else {
			err = s.db.QueryRow("SELECT id, path FROM libraries WHERE id = ?", srcLibID).Scan(&tgtLibID, &tgtLibPath)
		}
		if err != nil {
			failCount++
			results = append(results, moveResult{VideoID: videoID, Status: "failed", Error: "target library not found"})
			continue
		}

		var title string
		s.db.QueryRow("SELECT title FROM videos WHERE id = ?", videoID).Scan(&title)

		srcFullPath := filePath
		if !filepath.IsAbs(filePath) {
			var srcLibPath string
			s.db.QueryRow("SELECT path FROM libraries WHERE id = ?", srcLibID).Scan(&srcLibPath)
			srcFullPath = filepath.Join(srcLibPath, filePath)
		}

		filename := filepath.Base(srcFullPath)
		tgtFullPath := filepath.Join(tgtLibPath, filepath.FromSlash(targetSub), filename)

		if dryRun {
			successCount++
			results = append(results, moveResult{
				VideoID: videoID, Title: title, Status: "dry_run",
				Source: srcFullPath, Target: tgtFullPath,
			})
			continue
		}

		// Actual move
		if _, statErr := os.Stat(srcFullPath); os.IsNotExist(statErr) {
			failCount++
			results = append(results, moveResult{VideoID: videoID, Title: title, Status: "failed", Error: "source file missing"})
			continue
		}
		if _, statErr := os.Stat(tgtFullPath); statErr == nil {
			failCount++
			results = append(results, moveResult{VideoID: videoID, Title: title, Status: "failed", Error: "destination already exists"})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(tgtFullPath), 0755); err != nil {
			failCount++
			results = append(results, moveResult{VideoID: videoID, Title: title, Status: "failed", Error: err.Error()})
			continue
		}
		if err := os.Rename(srcFullPath, tgtFullPath); err != nil {
			data, readErr := os.ReadFile(srcFullPath)
			if readErr != nil {
				failCount++
				results = append(results, moveResult{VideoID: videoID, Title: title, Status: "failed", Error: readErr.Error()})
				continue
			}
			if writeErr := os.WriteFile(tgtFullPath, data, 0644); writeErr != nil {
				failCount++
				results = append(results, moveResult{VideoID: videoID, Title: title, Status: "failed", Error: writeErr.Error()})
				continue
			}
			os.Remove(srcFullPath)
		}

		_, dbErr := s.db.Exec("UPDATE videos SET file_path = ?, library_id = ?, updated_at = ? WHERE id = ?",
			tgtFullPath, tgtLibID, time.Now(), videoID)
		if dbErr != nil {
			log.Printf("[AI bulk_move] File moved but DB update failed for video %d: %v", videoID, dbErr)
		}

		// Learn placement
		perfRows, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", videoID)
		for perfRows.Next() {
			var perfName string
			if perfRows.Scan(&perfName) == nil {
				memKey := "placement_" + strings.ToLower(strings.ReplaceAll(perfName, " ", "_"))
				_ = s.SaveMemory(&models.Memory{
					Key: memKey, Value: fmt.Sprintf("Performer '%s' → %s", perfName, targetSub),
					Category: "insight", Importance: 8,
				})
			}
		}
		perfRows.Close()

		successCount++
		results = append(results, moveResult{
			VideoID: videoID, Title: title, Status: "success",
			Source: srcFullPath, Target: tgtFullPath,
		})
	}

	return map[string]interface{}{
		"dry_run":          dryRun,
		"total":            len(movesRaw),
		"success":          successCount,
		"failed":           failCount,
		"skipped":          skipCount,
		"results":          results,
	}, nil
}

// ── rename_folder ─────────────────────────────────────────────────────────────
// Renames a subfolder within a library and updates all affected video paths in the DB.

func (s *AICompanionService) registerRenameFolderTool() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "rename_folder",
			Description: `Rename a folder inside a library. All videos pointing to files inside that folder will have their paths updated in the database. The folder must exist on disk.`,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"library_name": map[string]interface{}{
						"type": "string", "description": "Library name or partial name",
					},
					"old_path": map[string]interface{}{
						"type": "string", "description": "Current relative path inside the library (e.g. 'A/D/BJ & Knot')",
					},
					"new_path": map[string]interface{}{
						"type": "string", "description": "New relative path inside the library (e.g. 'Tags/Amateur')",
					},
					"dry_run": map[string]interface{}{
						"type": "boolean", "description": "Simulate only (default: true)",
					},
				},
				"required": []string{"library_name", "old_path", "new_path"},
			},
		},
	}, s.toolRenameFolder)
}

func (s *AICompanionService) toolRenameFolder(args map[string]interface{}) (interface{}, error) {
	libName, _ := args["library_name"].(string)
	oldRel, _ := args["old_path"].(string)
	newRel, _ := args["new_path"].(string)
	dryRun := true
	if v, ok := args["dry_run"].(bool); ok { dryRun = v }
	if v, ok := args["dry_run"].(string); ok && v == "false" { dryRun = false }

	if oldRel == "" || newRel == "" {
		return map[string]interface{}{"error": "old_path and new_path are required"}, nil
	}

	var libID int64
	var libPath string
	err := s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+libName+"%").Scan(&libID, &libPath)
	if err != nil {
		return map[string]interface{}{"error": "library not found: " + libName}, nil
	}

	oldAbs := filepath.Join(libPath, filepath.FromSlash(oldRel))
	newAbs := filepath.Join(libPath, filepath.FromSlash(newRel))

	if _, err := os.Stat(oldAbs); os.IsNotExist(err) {
		return map[string]interface{}{"error": "source folder does not exist: " + oldRel}, nil
	}
	if _, err := os.Stat(newAbs); err == nil {
		return map[string]interface{}{"error": "destination already exists: " + newRel}, nil
	}

	// Count affected videos
	var affectedCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path LIKE ?",
		libID, "%"+string(filepath.Separator)+oldRel+"%").Scan(&affectedCount)

	// Build video ID list for DB update
	rows, err := s.db.Query("SELECT id, file_path FROM videos WHERE library_id = ? AND file_path LIKE ?",
		libID, "%"+string(filepath.Separator)+oldRel+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type vidPath struct { id int64; fp string }
	var vids []vidPath
	for rows.Next() {
		var v vidPath
		if rows.Scan(&v.id, &v.fp) == nil { vids = append(vids, v) }
	}

	if dryRun {
		return map[string]interface{}{
			"dry_run": true, "library": libName,
			"old_path": oldRel, "new_path": newRel,
			"affected_videos": affectedCount,
			"message": fmt.Sprintf("Would rename '%s' → '%s' and update %d video paths", oldRel, newRel, affectedCount),
		}, nil
	}

	// Create destination parent
	if err := os.MkdirAll(filepath.Dir(newAbs), 0755); err != nil {
		return map[string]interface{}{"error": "failed to create directory: " + err.Error()}, nil
	}
	// Rename on disk
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return map[string]interface{}{"error": "failed to rename folder: " + err.Error()}, nil
	}

	// Update DB paths
	now := time.Now()
	updated := 0
	for _, v := range vids {
		newFP := strings.Replace(v.fp, filepath.FromSlash(oldRel), filepath.FromSlash(newRel), 1)
		if _, err := s.db.Exec("UPDATE videos SET file_path = ?, updated_at = ? WHERE id = ?", newFP, now, v.id); err == nil {
			updated++
		}
	}

	return map[string]interface{}{
		"success": true, "library": libName,
		"old_path": oldRel, "new_path": newRel,
		"affected_videos": affectedCount, "db_updated": updated,
	}, nil
}

// ── move_folder_contents ──────────────────────────────────────────────────────
// Moves all videos from one subfolder into another (or flat into library root).

func (s *AICompanionService) registerMoveFolderContentsTool() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "move_folder_contents",
			Description: `Move all videos from one subfolder into another subfolder within the same library. Use this to consolidate catch-all folders into proper structure. Set target_subfolder to empty string to move to library root.`,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"library_name": map[string]interface{}{
						"type": "string", "description": "Library name or partial name",
					},
					"source_subfolder": map[string]interface{}{
						"type": "string", "description": "Relative path of the folder to empty (e.g. 'Z/Downloads/Torrents')",
					},
					"target_subfolder": map[string]interface{}{
						"type":        "string",
						"description": "Destination relative path (empty string = library root)",
					},
					"max_videos": map[string]interface{}{
						"type":        "number",
						"description": "Max videos to move per call (default: 50, max: 100)",
					},
					"dry_run": map[string]interface{}{
						"type": "boolean", "description": "Simulate only (default: true)",
					},
				},
				"required": []string{"library_name", "source_subfolder"},
			},
		},
	}, s.toolMoveFolderContents)
}

func (s *AICompanionService) toolMoveFolderContents(args map[string]interface{}) (interface{}, error) {
	libName, _ := args["library_name"].(string)
	srcRel, _ := args["source_subfolder"].(string)
	tgtRel, _ := args["target_subfolder"].(string)
	maxVids := intArg(args, "max_videos", 50)
	if maxVids > 100 { maxVids = 100 }
	dryRun := true
	if v, ok := args["dry_run"].(bool); ok { dryRun = v }
	if v, ok := args["dry_run"].(string); ok && v == "false" { dryRun = false }

	var libID int64
	var libPath string
	err := s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+libName+"%").Scan(&libID, &libPath)
	if err != nil {
		return map[string]interface{}{"error": "library not found: " + libName}, nil
	}

	// Find videos in the source folder
	rows, err := s.db.Query(
		"SELECT id, title, file_path FROM videos WHERE library_id = ? AND file_path LIKE ? LIMIT ?",
		libID, "%"+string(filepath.Separator)+srcRel+"%", maxVids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type vidInfo struct { id int64; title, fp string }
	var vids []vidInfo
	for rows.Next() {
		var v vidInfo
		if rows.Scan(&v.id, &v.title, &v.fp) == nil { vids = append(vids, v) }
	}

	totalInFolder := 0
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path LIKE ?",
		libID, "%"+string(filepath.Separator)+srcRel+"%").Scan(&totalInFolder)

	results := make([]moveResult, 0, len(vids))
	success, fail := 0, 0
	tgtFullPath := ""

	for _, v := range vids {
		srcFullPath := v.fp
		if !filepath.IsAbs(v.fp) {
			srcFullPath = filepath.Join(libPath, v.fp)
		}
		filename := filepath.Base(srcFullPath)
		if tgtRel != "" {
			tgtFullPath = filepath.Join(libPath, filepath.FromSlash(tgtRel), filename)
		} else {
			tgtFullPath = filepath.Join(libPath, filename)
		}

		if dryRun {
			success++
			results = append(results, moveResult{VideoID: v.id, Title: v.title, Status: "dry_run", Source: srcFullPath, Target: tgtFullPath})
			continue
		}

		if _, err := os.Stat(srcFullPath); os.IsNotExist(err) {
			fail++
			results = append(results, moveResult{VideoID: v.id, Title: v.title, Status: "failed", Error: "source missing"})
			continue
		}
		if _, err := os.Stat(tgtFullPath); err == nil {
			fail++
			results = append(results, moveResult{VideoID: v.id, Title: v.title, Status: "failed", Error: "destination exists"})
			continue
		}
		os.MkdirAll(filepath.Dir(tgtFullPath), 0755)
		if err := os.Rename(srcFullPath, tgtFullPath); err != nil {
			data, re := os.ReadFile(srcFullPath)
			if re != nil { fail++; results = append(results, moveResult{VideoID: v.id, Title: v.title, Status: "failed", Error: re.Error()}); continue }
			os.WriteFile(tgtFullPath, data, 0644)
			os.Remove(srcFullPath)
		}
		s.db.Exec("UPDATE videos SET file_path = ?, updated_at = ? WHERE id = ?", tgtFullPath, time.Now(), v.id)
		success++
		results = append(results, moveResult{VideoID: v.id, Title: v.title, Status: "success", Source: srcFullPath, Target: tgtFullPath})
	}

	return map[string]interface{}{
		"dry_run":              dryRun,
		"library":              libName,
		"source_subfolder":     srcRel,
		"total_in_source":      totalInFolder,
		"processed_this_call":  len(vids),
		"success":              success,
		"failed":               fail,
		"remaining":            totalInFolder - len(vids),
		"results":              results,
	}, nil
}

// ── batch_move_by_pattern ─────────────────────────────────────────────────────
// Moves videos from a source folder to performer-based subfolders automatically.
// Uses the priority rule: Performer > Tag > Studio > _Unsorted.
// Processes up to 200 videos per call — for a 16k catch-all, the AI calls this ~80 times.

func (s *AICompanionService) registerBatchMoveByPatternTool() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "batch_move_by_pattern",
			Description: `Move videos from a source folder into performer-based subfolders automatically. For each video, looks up its performers in the database and places it into 'Performers/<Name>/'. Videos with no performers go to '_Unsorted/'. Processes up to 200 per call. Use dry_run=true first to preview the plan.`,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"library_name": map[string]interface{}{
						"type": "string", "description": "Library name or partial name",
					},
					"source_subfolder": map[string]interface{}{
						"type": "string", "description": "Relative path of the folder to restructure (e.g. 'Z/Downloads/Torrents')",
					},
					"target_base": map[string]interface{}{
						"type":        "string",
						"description": "Base folder for organized output (default: 'Performers'). Videos go to <target_base>/<PerformerName>/",
					},
					"fallback_folder": map[string]interface{}{
						"type":        "string",
						"description": "Folder for videos with no performers (default: '_Unsorted')",
					},
					"max_videos": map[string]interface{}{
						"type":        "number",
						"description": "Max videos to process per call (default: 200, max: 500)",
					},
					"dry_run": map[string]interface{}{
						"type": "boolean", "description": "Simulate only (default: true)",
					},
				},
				"required": []string{"library_name", "source_subfolder"},
			},
		},
	}, s.toolBatchMoveByPattern)
}

func (s *AICompanionService) toolBatchMoveByPattern(args map[string]interface{}) (interface{}, error) {
	libName, _ := args["library_name"].(string)
	srcRel, _ := args["source_subfolder"].(string)
	targetBase, _ := args["target_base"].(string)
	fallbackFolder, _ := args["fallback_folder"].(string)
	maxVids := intArg(args, "max_videos", 200)
	if maxVids > 500 { maxVids = 500 }
	dryRun := true
	if v, ok := args["dry_run"].(bool); ok { dryRun = v }
	if v, ok := args["dry_run"].(string); ok && v == "false" { dryRun = false }
	if targetBase == "" { targetBase = "Performers" }
	if fallbackFolder == "" { fallbackFolder = "_Unsorted" }

	var libID int64
	var libPath string
	err := s.db.QueryRow("SELECT id, path FROM libraries WHERE name LIKE ?", "%"+libName+"%").Scan(&libID, &libPath)
	if err != nil {
		return map[string]interface{}{"error": "library not found: " + libName}, nil
	}

	// Find videos in source folder with their performers in one query
	rows, err := s.db.Query(`
		SELECT v.id, v.title, v.file_path,
		       COALESCE(GROUP_CONCAT(DISTINCT p.name), '') as performers
		FROM videos v
		LEFT JOIN video_performers vp ON v.id = vp.video_id
		LEFT JOIN performers p ON vp.performer_id = p.id
		WHERE v.library_id = ? AND v.file_path LIKE ?
		GROUP BY v.id
		ORDER BY v.title
		LIMIT ?`, libID, "%"+string(filepath.Separator)+srcRel+"%", maxVids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type batchVideo struct {
		id         int64
		title      string
		filePath   string
		performers string
	}
	var vids []batchVideo
	for rows.Next() {
		var v batchVideo
		if rows.Scan(&v.id, &v.title, &v.filePath, &v.performers) == nil {
			vids = append(vids, v)
		}
	}

	// Count total in source
	var totalCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path LIKE ?",
		libID, "%"+string(filepath.Separator)+srcRel+"%").Scan(&totalCount)

	// Build placement plan
	type placement struct {
		videoID   int64
		title     string
		srcPath   string
		tgtPath   string
		performer string
		status    string
		error     string
	}
	var placements []placement
	performerCounts := map[string]int{}

	for _, v := range vids {
		srcFullPath := v.filePath
		if !filepath.IsAbs(v.filePath) {
			srcFullPath = filepath.Join(libPath, v.filePath)
		}
		filename := filepath.Base(srcFullPath)

		var tgtSub string
		var performer string
		if v.performers != "" {
			// Use first performer for folder placement
			parts := strings.Split(v.performers, ",")
			performer = strings.TrimSpace(parts[0])
			tgtSub = filepath.Join(targetBase, performer)
			performerCounts[performer]++
		} else {
			tgtSub = fallbackFolder
			performerCounts[fallbackFolder]++
		}
		tgtFullPath := filepath.Join(libPath, filepath.FromSlash(tgtSub), filename)

		placements = append(placements, placement{
			videoID: v.id, title: v.title, srcPath: srcFullPath,
			tgtPath: tgtFullPath, performer: performer,
		})
	}

	if dryRun {
		// Show top performers and what would happen
		type perfCount struct { name string; count int }
		var sorted []perfCount
		for name, count := range performerCounts {
			sorted = append(sorted, perfCount{name, count})
		}
		// Simple sort by count desc
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].count > sorted[i].count {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}
		if len(sorted) > 20 { sorted = sorted[:20] }

		return map[string]interface{}{
			"dry_run":           true,
			"library":           libName,
			"source":            srcRel,
			"target_base":       targetBase,
			"total_in_source":   totalCount,
			"processed_this":    len(vids),
			"would_move":        len(placements),
			"unique_performers": len(performerCounts),
			"top_destinations":  sorted,
			"sample_placements": placements,
		}, nil
	}

	// Execute moves — report progress every 25 videos
	success, fail, skip := 0, 0, 0
	for i := range placements {
		p := &placements[i]

		// Report progress every 25 videos
		if (i+1)%25 == 0 || i == len(placements)-1 {
			s.ReportProgress(map[string]interface{}{
				"type":         "batch_move_progress",
				"current":      i + 1,
				"total":        len(placements),
				"success":      success,
				"failed":       fail,
				"skipped":      skip,
				"remaining":    totalCount - (i + 1),
				"source":       srcRel,
				"target_base":  targetBase,
			})
		}

		if _, statErr := os.Stat(p.srcPath); os.IsNotExist(statErr) {
			p.status = "skipped"
			p.error = "source missing"
			skip++
			continue
		}
		if _, statErr := os.Stat(p.tgtPath); statErr == nil {
			p.status = "skipped"
			p.error = "destination exists"
			skip++
			continue
		}
		os.MkdirAll(filepath.Dir(p.tgtPath), 0755)

		if err := os.Rename(p.srcPath, p.tgtPath); err != nil {
			// Fallback: copy + delete
			data, re := os.ReadFile(p.srcPath)
			if re != nil {
				p.status = "failed"; p.error = re.Error(); fail++; continue
			}
			if we := os.WriteFile(p.tgtPath, data, 0644); we != nil {
				p.status = "failed"; p.error = we.Error(); fail++; continue
			}
			os.Remove(p.srcPath)
		}

		// Update DB
		s.db.Exec("UPDATE videos SET file_path = ?, updated_at = ? WHERE id = ?",
			p.tgtPath, time.Now(), p.videoID)

		// Learn placement
		if p.performer != "" {
			memKey := "placement_" + strings.ToLower(strings.ReplaceAll(p.performer, " ", "_"))
			_ = s.SaveMemory(&models.Memory{
				Key: memKey, Value: fmt.Sprintf("Performer '%s' → %s/%s", p.performer, targetBase, p.performer),
				Category: "insight", Importance: 8,
			})
		}

		p.status = "success"
		success++
	}

	return map[string]interface{}{
		"dry_run":           false,
		"library":           libName,
		"source":            srcRel,
		"target_base":       targetBase,
		"total_in_source":   totalCount,
		"processed_this":    len(vids),
		"success":           success,
		"failed":            fail,
		"skipped":           skip,
		"remaining":         totalCount - len(vids),
		"performers_used":   len(performerCounts),
		"summary":           performerCounts,
	}, nil
}
