package services

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (s *AICompanionService) registerPerformerTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "search_performers",
			Description: "Search for performers by name. Returns performer details.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query to filter performers by name"},
				"limit": map[string]interface{}{"type": "number", "description": "Max results (default: 50)"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		limit := intArg(args, "limit", 50)
		rows, err := s.db.Query(`SELECT p.id, p.name, p.video_count FROM performers p WHERE p.name LIKE ? ORDER BY p.video_count DESC LIMIT ?`, "%"+query+"%", limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var name string
			var videoCount int
			if rows.Scan(&id, &name, &videoCount) == nil {
				results = append(results, map[string]interface{}{"id": id, "name": name, "video_count": videoCount})
			}
		}
		return results, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_top_performers",
			Description: "Get performers with the most videos.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Number of top performers (default: 10)"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		limit := intArg(args, "limit", 10)
		rows, err := s.db.Query(`SELECT p.name, COUNT(vp.video_id) as cnt FROM performers p LEFT JOIN video_performers vp ON p.id = vp.performer_id GROUP BY p.id ORDER BY cnt DESC LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var name string
			var cnt int
			if rows.Scan(&name, &cnt) == nil {
				results = append(results, map[string]interface{}{"name": name, "video_count": cnt})
			}
		}
		return results, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_performer_stats",
			Description: "Get a comprehensive profile for a performer: total videos, total watch time, avg rating, rating distribution, favorite count, library breakdown, top tags, top studios, recent additions, and date range.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"performer": map[string]interface{}{"type": "string", "description": "Performer name to look up (partial match supported)"},
			}, "required": []string{"performer"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		performerQuery, _ := args["performer"].(string)
		if performerQuery == "" {
			return map[string]interface{}{"error": "performer name required"}, nil
		}
		var perfID int64
		var perfName string
		err := s.db.QueryRow("SELECT id, name FROM performers WHERE LOWER(name) = LOWER(?) LIMIT 1", performerQuery).Scan(&perfID, &perfName)
		if err != nil {
			err = s.db.QueryRow("SELECT id, name FROM performers WHERE LOWER(name) LIKE LOWER(?) ORDER BY LENGTH(name) ASC LIMIT 1", "%"+performerQuery+"%").Scan(&perfID, &perfName)
		}
		if err != nil {
			return map[string]interface{}{"error": "performer not found: " + performerQuery}, nil
		}

		var videoCount, favCount int
		var totalDuration, avgRating float64
		s.db.QueryRow(`
			SELECT COUNT(*), COALESCE(SUM(v.duration), 0),
			       COALESCE(AVG(CASE WHEN v.rating > 0 THEN CAST(v.rating AS FLOAT) END), 0)
			FROM videos v JOIN video_performers vp ON v.id = vp.video_id
			WHERE vp.performer_id = ?`, perfID).Scan(&videoCount, &totalDuration, &avgRating)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos v JOIN video_performers vp ON v.id = vp.video_id WHERE vp.performer_id = ? AND v.is_favorite = 1`, perfID).Scan(&favCount)

		ratingDist := map[string]int{"0★ (unrated)": 0, "1★": 0, "2★": 0, "3★": 0, "4★": 0, "5★": 0}
		rdRows, _ := s.db.Query(`SELECT COALESCE(v.rating, 0), COUNT(*) FROM videos v JOIN video_performers vp ON v.id = vp.video_id WHERE vp.performer_id = ? GROUP BY COALESCE(v.rating, 0)`, perfID)
		if rdRows != nil {
			for rdRows.Next() {
				var r, cnt int
				rdRows.Scan(&r, &cnt)
				key := fmt.Sprintf("%d★", r)
				if r == 0 {
					key = "0★ (unrated)"
				}
				ratingDist[key] = cnt
			}
			rdRows.Close()
		}

		type nameCount struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		var libBreakdown []nameCount
		lbRows, _ := s.db.Query(`
			SELECT l.name, COUNT(v.id) FROM videos v
			JOIN video_performers vp ON v.id = vp.video_id JOIN libraries l ON v.library_id = l.id
			WHERE vp.performer_id = ? GROUP BY l.id ORDER BY COUNT(v.id) DESC`, perfID)
		if lbRows != nil {
			for lbRows.Next() {
				var nc nameCount
				lbRows.Scan(&nc.Name, &nc.Count)
				libBreakdown = append(libBreakdown, nc)
			}
			lbRows.Close()
		}

		var topTags []nameCount
		tagRows, _ := s.db.Query(`
			SELECT t.name, COUNT(DISTINCT vt.video_id) FROM tags t
			JOIN video_tags vt ON t.id = vt.tag_id JOIN video_performers vp ON vt.video_id = vp.video_id
			WHERE vp.performer_id = ? GROUP BY t.id ORDER BY COUNT(DISTINCT vt.video_id) DESC LIMIT 10`, perfID)
		if tagRows != nil {
			for tagRows.Next() {
				var nc nameCount
				tagRows.Scan(&nc.Name, &nc.Count)
				topTags = append(topTags, nc)
			}
			tagRows.Close()
		}

		var topStudios []nameCount
		stRows, _ := s.db.Query(`
			SELECT s.name, COUNT(DISTINCT vs.video_id) FROM studios s
			JOIN video_studios vs ON s.id = vs.studio_id JOIN video_performers vp ON vs.video_id = vp.video_id
			WHERE vp.performer_id = ? GROUP BY s.id ORDER BY COUNT(DISTINCT vs.video_id) DESC LIMIT 5`, perfID)
		if stRows != nil {
			for stRows.Next() {
				var nc nameCount
				stRows.Scan(&nc.Name, &nc.Count)
				topStudios = append(topStudios, nc)
			}
			stRows.Close()
		}

		type recentVid struct {
			ID      int64  `json:"id"`
			Title   string `json:"title"`
			AddedAt string `json:"added_at"`
			Rating  int    `json:"rating"`
			Library string `json:"library"`
		}
		var recentVideos []recentVid
		recRows, _ := s.db.Query(`
			SELECT v.id, v.title, v.created_at, COALESCE(v.rating, 0), COALESCE(l.name, '')
			FROM videos v JOIN video_performers vp ON v.id = vp.video_id LEFT JOIN libraries l ON v.library_id = l.id
			WHERE vp.performer_id = ? ORDER BY v.created_at DESC LIMIT 5`, perfID)
		if recRows != nil {
			for recRows.Next() {
				var rv recentVid
				recRows.Scan(&rv.ID, &rv.Title, &rv.AddedAt, &rv.Rating, &rv.Library)
				recentVideos = append(recentVideos, rv)
			}
			recRows.Close()
		}

		var firstAdded, lastAdded string
		s.db.QueryRow(`SELECT MIN(v.created_at), MAX(v.created_at) FROM videos v JOIN video_performers vp ON v.id = vp.video_id WHERE vp.performer_id = ?`, perfID).Scan(&firstAdded, &lastAdded)

		return map[string]interface{}{
			"performer_id":         perfID,
			"performer_name":       perfName,
			"total_videos":         videoCount,
			"favorite_count":       favCount,
			"total_watch_time":     formatSeconds(totalDuration),
			"total_watch_time_hrs": fmt.Sprintf("%.1fh", totalDuration/3600),
			"avg_rating":           fmt.Sprintf("%.2f", avgRating),
			"rating_distribution":  ratingDist,
			"libraries":            libBreakdown,
			"top_tags":             topTags,
			"top_studios":          topStudios,
			"recent_videos":        recentVideos,
			"first_added":          firstAdded,
			"last_added":           lastAdded,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "link_performer_to_video",
			Description: "Add a performer to a video by performer name. Creates the performer if they don't exist yet. Use when the user asks to add a performer to a specific video.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id":       map[string]interface{}{"type": "number", "description": "ID of the video"},
				"performer_name": map[string]interface{}{"type": "string", "description": "Name of the performer to link"},
			}, "required": []string{"video_id", "performer_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		performerName, _ := args["performer_name"].(string)
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		if performerName == "" {
			return map[string]interface{}{"error": "performer_name required"}, nil
		}

		var videoTitle string
		if err := s.db.QueryRow("SELECT title FROM videos WHERE id = ?", videoID).Scan(&videoTitle); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		}

		var perfID int64
		err := s.db.QueryRow("SELECT id FROM performers WHERE LOWER(name) = LOWER(?)", performerName).Scan(&perfID)
		if err != nil {
			res, err := s.db.Exec("INSERT INTO performers (name, video_count) VALUES (?, 0)", performerName)
			if err != nil {
				return map[string]interface{}{"error": "failed to create performer: " + err.Error()}, nil
			}
			perfID, _ = res.LastInsertId()
		}

		_, err = s.db.Exec("INSERT OR IGNORE INTO video_performers (video_id, performer_id) VALUES (?, ?)", videoID, perfID)
		if err != nil {
			return map[string]interface{}{"error": "failed to link: " + err.Error()}, nil
		}
		s.db.Exec("UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = ?) WHERE id = ?", perfID, perfID)

		return map[string]interface{}{
			"video_id":       videoID,
			"video_title":    videoTitle,
			"performer_id":   perfID,
			"performer_name": performerName,
			"linked":         true,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "unlink_performer_from_video",
			Description: "Remove a performer from a video. Does not delete the performer record itself — only removes the association with this video.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id":       map[string]interface{}{"type": "number", "description": "ID of the video"},
				"performer_name": map[string]interface{}{"type": "string", "description": "Name of the performer to unlink"},
			}, "required": []string{"video_id", "performer_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		performerName, _ := args["performer_name"].(string)
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		if performerName == "" {
			return map[string]interface{}{"error": "performer_name required"}, nil
		}

		var perfID int64
		if err := s.db.QueryRow("SELECT id FROM performers WHERE LOWER(name) = LOWER(?)", performerName).Scan(&perfID); err != nil {
			return map[string]interface{}{"error": "performer not found: " + performerName}, nil
		}

		res, _ := s.db.Exec("DELETE FROM video_performers WHERE video_id = ? AND performer_id = ?", videoID, perfID)
		n, _ := res.RowsAffected()
		if n == 0 {
			return map[string]interface{}{"error": fmt.Sprintf("performer '%s' was not linked to video %d", performerName, videoID)}, nil
		}
		s.db.Exec("UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = ?) WHERE id = ?", perfID, perfID)

		return map[string]interface{}{
			"video_id":       videoID,
			"performer_name": performerName,
			"unlinked":       true,
		}, nil
	})

	// ── create_performers ──────────────────────────────────────────────────────
	// Registers NEW performers by name (batch). The resolver can only match
	// against performers that EXIST in the DB — when filenames clearly contain
	// unregistered names (e.g. a fresh download folder full of new stars), the
	// Companion reads them itself and registers them here first, then re-runs
	// link_performers_from_filenames / batch_move_by_pattern.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "create_performers",
			Description: "Register one or more NEW performers by name (case-insensitive dedupe against existing). " +
				"Use when filenames clearly contain performer names that are NOT in the database yet — the filename resolver can only match against known performers. " +
				"Only pass names you are CONFIDENT are real performer names from filenames/titles (e.g. 'Marsha May', 'Madison Ivy'). " +
				"Never invent names. After creating, re-run link_performers_from_filenames to link them to videos.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"names": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Performer names to create (e.g. ['Marsha May', 'Madison Ivy'])",
				},
			}, "required": []string{"names"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rawNames, _ := args["names"].([]interface{})
		if len(rawNames) == 0 {
			return map[string]interface{}{"error": "names array required"}, nil
		}

		created := []map[string]interface{}{}
		skipped := []string{}
		for _, n := range rawNames {
			name, _ := n.(string)
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			// Case-insensitive dedupe
			var existingID int64
			if err := s.db.QueryRow(`SELECT id FROM performers WHERE LOWER(name) = LOWER(?)`, name).Scan(&existingID); err == nil {
				skipped = append(skipped, name+" (exists)")
				continue
			}
			res, err := s.db.Exec(`INSERT INTO performers (name, video_count) VALUES (?, 0)`, name)
			if err != nil {
				skipped = append(skipped, name+" (error: "+err.Error()+")")
				continue
			}
			id, _ := res.LastInsertId()
			created = append(created, map[string]interface{}{"name": name, "id": id})
		}

		return map[string]interface{}{
			"created_count": len(created),
			"created":       created,
			"skipped":       skipped,
			"note":          "Now re-run link_performers_from_filenames — these names will resolve against the new entries.",
		}, nil
	})

	// ── resolve_performer_from_filename ────────────────────────────────────────
	// The Companion's requested #4: an AGGRESSIVE filename→performer resolver.
	// Normalizes separators (Crystal.Rush → crystal rush), matches whole words,
	// falls back to Levenshtein for typos, and returns scored candidates so the
	// AI can decide what is "100% sure" vs needs human review.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "resolve_performer_from_filename",
			Description: "Score every known performer against a filename (or folder name) and return ranked candidates. " +
				"Normalizes dots/underscores/dashes, matches whole words, and tolerates typos (Levenshtein). " +
				"Use before batch moves to know which videos can be placed with confidence: score 1.0 = all name tokens present (safe), " +
				"0.95 = first+last present, <0.9 = ambiguous or partial. Check 'ambiguous' — if true, do NOT auto-link, ask Brix.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"filename":  map[string]interface{}{"type": "string", "description": "Filename or folder name to resolve (e.g. 'Crystal.Rush.2024.1080p.mp4')"},
				"min_score": map[string]interface{}{"type": "number", "description": "Only return candidates at or above this score (default: 0.6)"},
				"limit":     map[string]interface{}{"type": "number", "description": "Max candidates to return (default: 5)"},
			}, "required": []string{"filename"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		filename, _ := args["filename"].(string)
		if filename == "" {
			return map[string]interface{}{"error": "filename required"}, nil
		}
		minScore := floatArg(args, "min_score", 0.6)
		limit := intArg(args, "limit", 5)

		var performers []pathEntity
		rows, err := s.db.Query(`SELECT id, name FROM performers ORDER BY video_count DESC`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var p pathEntity
			if rows.Scan(&p.id, &p.name) == nil {
				performers = append(performers, p)
			}
		}

		cands := rankPerformersForFilename(filename, performers, minScore)
		if len(cands) > limit {
			cands = cands[:limit]
		}
		return map[string]interface{}{
			"filename":          filename,
			"normalized":        normalizeForMatch(filename),
			"candidates":        cands,
			"match_count":       len(cands),
			"ambiguous":         isAmbiguous(cands),
			"performers_known":  len(performers),
			"note":              "ambiguous=true means top 2 candidates are within 0.05 — do not auto-link, ask Brix which one.",
		}, nil
	})

	// ── link_performers_from_filenames ─────────────────────────────────────────
	// Batch version: scans a folder's videos, resolves performers from filenames,
	// and links ONLY high-confidence, unambiguous matches. Dry-run by default.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "link_performers_from_filenames",
			Description: "Scan videos in a folder and link performers resolved from their filenames. Only links matches at or above min_score that are NOT ambiguous (top-2 within 0.05 = skipped). " +
				"Safe: only ADDS links, never removes. Run dry_run=true first to see per-video candidates. " +
				"This is the missing step before batch_move_by_pattern — once performers are linked, move previews resolve to real destinations.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name":     map[string]interface{}{"type": "string", "description": "Library name or partial name (required)"},
				"source_subfolder": map[string]interface{}{"type": "string", "description": "Restrict to a relative subfolder (optional, e.g. 'New folder')"},
				"min_score":        map[string]interface{}{"type": "number", "description": "Minimum confidence to auto-link (default: 0.9 — first+last or all tokens present)"},
				"max_videos":       map[string]interface{}{"type": "number", "description": "Max videos to process per call (default: 200, max: 500)"},
				"dry_run":          map[string]interface{}{"type": "boolean", "description": "Preview only, no writes (default: true)"},
			}, "required": []string{"library_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libName, _ := args["library_name"].(string)
		srcRel, _ := args["source_subfolder"].(string)
		minScore := floatArg(args, "min_score", 0.9)
		maxVids := intArg(args, "max_videos", 200)
		if maxVids > 500 {
			maxVids = 500
		}
		dryRun := true
		if v, ok := args["dry_run"].(bool); ok {
			dryRun = v
		}

		var libID int64
		if err := s.db.QueryRow("SELECT id FROM libraries WHERE name LIKE ?", "%"+libName+"%").Scan(&libID); err != nil {
			return map[string]interface{}{"error": "library not found: " + libName}, nil
		}

		// Performers ranked by video_count (popular names first)
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
		if len(performers) == 0 {
			return map[string]interface{}{"error": "no performers in database to match against"}, nil
		}

		// Videos in scope, skipping ones that already have performers
		query := `SELECT v.id, v.title, v.file_path FROM videos v
			WHERE v.library_id = ? AND v.file_path LIKE ?
			AND NOT EXISTS (SELECT 1 FROM video_performers vp WHERE vp.video_id = v.id)
			ORDER BY v.title LIMIT ?`
		like := "%" + string(filepath.Separator) + srcRel + "%"
		if srcRel == "" {
			like = "%"
		}
		rows, err := s.db.Query(query, libID, like, maxVids)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		type videoRow2 struct {
			id    int64
			title string
			path  string
		}
		var vids []videoRow2
		for rows.Next() {
			var v videoRow2
			if rows.Scan(&v.id, &v.title, &v.path) == nil {
				vids = append(vids, v)
			}
		}

		linked := 0
		skippedAmbiguous := 0
		skippedLow := 0
		alreadyLinked := 0
		results := []map[string]interface{}{}

		for _, v := range vids {
			filename := filepath.Base(v.path)
			cands := rankPerformersForFilename(filename, performers, minScore)
			if len(cands) == 0 {
				skippedLow++
				continue
			}
			if isAmbiguous(cands) {
				skippedAmbiguous++
				if dryRun {
					results = append(results, map[string]interface{}{
						"video_id": v.id, "title": v.title, "filename": filename,
						"status": "ambiguous", "candidates": topN(cands, 3),
					})
				}
				continue
			}
			best := cands[0]

			// Already linked?
			var exists int
			s.db.QueryRow(`SELECT 1 FROM video_performers WHERE video_id = ? AND performer_id = ?`, v.id, best.PerformerID).Scan(&exists)
			if exists == 1 {
				alreadyLinked++
				continue
			}

			if dryRun {
				results = append(results, map[string]interface{}{
					"video_id": v.id, "title": v.title, "filename": filename,
					"status": "would_link", "performer": best.Name,
					"score": best.Score, "match_type": best.MatchType,
				})
				linked++
				continue
			}

			if _, err := s.db.Exec(`INSERT OR IGNORE INTO video_performers (video_id, performer_id) VALUES (?, ?)`, v.id, best.PerformerID); err == nil {
				s.db.Exec(`UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = ?) WHERE id = ?`, best.PerformerID, best.PerformerID)
				linked++
				results = append(results, map[string]interface{}{
					"video_id": v.id, "title": v.title, "filename": filename,
					"status": "linked", "performer": best.Name,
					"score": best.Score, "match_type": best.MatchType,
				})
			}
		}

		return map[string]interface{}{
			"dry_run":           dryRun,
			"library":           libName,
			"source":            srcRel,
			"videos_scanned":    len(vids),
			"linked":            linked,
			"skipped_ambiguous": skippedAmbiguous,
			"skipped_low_score": skippedLow,
			"already_linked":    alreadyLinked,
			"min_score":         minScore,
			"results":           results,
			"note":              "After linking, re-run batch_move_by_pattern dry_run — destinations will now resolve to real performers.",
		}, nil
	})
}

// topN returns the first n candidates (already sorted best-first).
func topN(cands []performerCandidate, n int) []performerCandidate {
	if len(cands) > n {
		return cands[:n]
	}
	return cands
}
