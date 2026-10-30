package services

import (
	"fmt"
	"path/filepath"
	"strings"
)

// registerImportTools closes the download→organize loop: new files land in a
// library (fsnotify watcher fires, scan adds them to the DB), and this tool
// surfaces them with scored placement suggestions so the Companion can propose
// moves the moment content arrives.

func (s *AICompanionService) registerImportTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "get_new_imports",
			Description: "List videos recently added to the library (default: last 24h) with performer resolution and a suggested destination folder for each. " +
				"Use when Brix asks what's new / what just landed / what needs organizing, or proactively after a download completes. " +
				"confidence 1.0 = performer fully resolved (safe to move), 0.0 = no performer (needs metadata or manual placement). " +
				"Follow up with batch_move_by_pattern (min_confidence=1.0) to file the confident ones.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"hours":        map[string]interface{}{"type": "number", "description": "Look back this many hours (default: 24, max: 720)"},
				"library_name": map[string]interface{}{"type": "string", "description": "Restrict to one library (optional)"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max videos to return (default: 50, max: 200)"},
				"only_unplaced": map[string]interface{}{"type": "boolean", "description": "Only videos with no performer linked yet (default: true)"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		hours := intArg(args, "hours", 24)
		if hours > 720 {
			hours = 720
		}
		limit := intArg(args, "limit", 50)
		if limit > 200 {
			limit = 200
		}
		libName, _ := args["library_name"].(string)
		onlyUnplaced := true
		if v, ok := args["only_unplaced"].(bool); ok {
			onlyUnplaced = v
		}

		// Performers for resolution
		var performers []pathEntity
		if pRows, err := s.db.Query(`SELECT id, name FROM performers ORDER BY video_count DESC`); err == nil {
			for pRows.Next() {
				var p pathEntity
				if pRows.Scan(&p.id, &p.name) == nil {
					performers = append(performers, p)
				}
			}
			pRows.Close()
		}

		query := `SELECT v.id, v.title, v.file_path, v.library_id, l.name as lib_name,
			COALESCE((SELECT GROUP_CONCAT(DISTINCT p.name) FROM video_performers vp
				JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = v.id), '') as performers
			FROM videos v JOIN libraries l ON l.id = v.library_id
			WHERE v.created_at > datetime('now', ?)`
		argsList := []interface{}{fmt.Sprintf("-%d hours", hours)}

		if libName != "" {
			query += " AND l.name LIKE ?"
			argsList = append(argsList, "%"+libName+"%")
		}
		if onlyUnplaced {
			query += ` AND NOT EXISTS (SELECT 1 FROM video_performers vp WHERE vp.video_id = v.id)`
		}
		query += " ORDER BY v.created_at DESC LIMIT ?"
		argsList = append(argsList, limit)

		rows, err := s.db.Query(query, argsList...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		imports := []map[string]interface{}{}
		confident := 0
		needsMetadata := 0
		ambiguousCount := 0

		for rows.Next() {
			var id int64
			var title, filePath, libName2, existingPerformers string
			var libID int64
			if rows.Scan(&id, &title, &filePath, &libID, &libName2, &existingPerformers) != nil {
				continue
			}

			filename := filepath.Base(filePath)
			entry := map[string]interface{}{
				"video_id":   id,
				"title":      title,
				"filename":   filename,
				"library":    libName2,
				"path":       filePath,
			}

			if existingPerformers != "" {
				// Already linked — placement resolves from the DB
				entry["performers"] = existingPerformers
				entry["suggested_dest"] = filepath.Join("Performers", strings.Split(existingPerformers, ",")[0])
				entry["confidence"] = 1.0
				entry["status"] = "ready_to_file"
				confident++
			} else {
				cands := rankPerformersForFilename(filename, performers, 0.6)
				if len(cands) == 0 {
					entry["confidence"] = 0.0
					entry["status"] = "needs_metadata"
					entry["note"] = "No performer resolved from filename — link a performer first or place manually."
					needsMetadata++
				} else if isAmbiguous(cands) {
					entry["confidence"] = cands[0].Score
					entry["candidates"] = topN(cands, 3)
					entry["status"] = "ambiguous"
					entry["note"] = "Top candidates too close — ask Brix which performer this is."
					ambiguousCount++
				} else {
					entry["suggested_performer"] = cands[0].Name
					entry["confidence"] = cands[0].Score
					entry["match_type"] = cands[0].MatchType
					entry["candidates"] = topN(cands, 3)
					if cands[0].Score >= 0.9 {
						entry["suggested_dest"] = filepath.Join("Performers", cands[0].Name)
						entry["status"] = "ready_to_file"
						confident++
					} else {
						entry["status"] = "low_confidence"
						needsMetadata++
					}
				}
			}
			imports = append(imports, entry)
		}

		return map[string]interface{}{
			"window_hours":       hours,
			"count":              len(imports),
			"ready_to_file":      confident,
			"needs_metadata":     needsMetadata,
			"ambiguous":          ambiguousCount,
			"imports":            imports,
			"note":               "ready_to_file items can be moved with batch_move_by_pattern (min_confidence=1.0). Link ambiguous ones with link_performers_from_filenames after confirming with Brix.",
		}, nil
	})
}