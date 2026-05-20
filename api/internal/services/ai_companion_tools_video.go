package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (s *AICompanionService) registerVideoTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "search_videos",
			Description: "Rich video search supporting natural-language criteria: filter by title, performer name, tag, studio, library, duration range, minimum rating, and favorite status. Also supports sort order. Use this to answer queries like 'Nicole Aniston videos over 30 minutes', 'my favorites rated 4+', 'newest videos in Backup', or 'random video suggestion'.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"query":        map[string]interface{}{"type": "string", "description": "Title keyword search (optional)"},
				"performer":    map[string]interface{}{"type": "string", "description": "Filter by performer full name (optional)"},
				"tag":          map[string]interface{}{"type": "string", "description": "Filter by tag name (optional)"},
				"studio":       map[string]interface{}{"type": "string", "description": "Filter by studio name (optional)"},
				"library":      map[string]interface{}{"type": "string", "description": "Filter by library name, e.g. 'Backup' (optional)"},
				"min_duration": map[string]interface{}{"type": "number", "description": "Minimum duration in seconds, e.g. 1800 for 30 min (optional)"},
				"max_duration": map[string]interface{}{"type": "number", "description": "Maximum duration in seconds (optional)"},
				"min_rating":   map[string]interface{}{"type": "number", "description": "Minimum rating 1-5 (optional)"},
				"is_favorite":  map[string]interface{}{"type": "boolean", "description": "If true, return only favorites (optional)"},
				"sort_by":      map[string]interface{}{"type": "string", "description": "Sort order: 'recent' (default), 'rating', 'duration', 'title', 'random'"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max results (default: 20, max: 50)"},
			}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		titleQ, _ := args["query"].(string)
		performer, _ := args["performer"].(string)
		tag, _ := args["tag"].(string)
		studio, _ := args["studio"].(string)
		library, _ := args["library"].(string)
		minDur := floatArg(args, "min_duration", 0)
		maxDur := floatArg(args, "max_duration", 0)
		minRating := intArg(args, "min_rating", 0)
		isFavOnly, _ := args["is_favorite"].(bool)
		sortBy, _ := args["sort_by"].(string)
		limit := intArg(args, "limit", 20)
		if limit > 50 {
			limit = 50
		}

		var conds []string
		var qArgs []interface{}
		if titleQ != "" {
			conds = append(conds, "v.title LIKE ?")
			qArgs = append(qArgs, "%"+titleQ+"%")
		}
		if performer != "" {
			conds = append(conds, `EXISTS (SELECT 1 FROM video_performers vp JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = v.id AND LOWER(p.name) LIKE LOWER(?))`)
			qArgs = append(qArgs, "%"+performer+"%")
		}
		if tag != "" {
			conds = append(conds, `EXISTS (SELECT 1 FROM video_tags vt JOIN tags t ON t.id = vt.tag_id WHERE vt.video_id = v.id AND LOWER(t.name) LIKE LOWER(?))`)
			qArgs = append(qArgs, "%"+tag+"%")
		}
		if studio != "" {
			conds = append(conds, `EXISTS (SELECT 1 FROM video_studios vs JOIN studios s ON s.id = vs.studio_id WHERE vs.video_id = v.id AND LOWER(s.name) LIKE LOWER(?))`)
			qArgs = append(qArgs, "%"+studio+"%")
		}
		if library != "" {
			conds = append(conds, "LOWER(l.name) LIKE LOWER(?)")
			qArgs = append(qArgs, "%"+library+"%")
		}
		if minDur > 0 {
			conds = append(conds, "v.duration >= ?")
			qArgs = append(qArgs, minDur)
		}
		if maxDur > 0 {
			conds = append(conds, "v.duration <= ?")
			qArgs = append(qArgs, maxDur)
		}
		if minRating > 0 {
			conds = append(conds, "COALESCE(v.rating, 0) >= ?")
			qArgs = append(qArgs, minRating)
		}
		if isFavOnly {
			conds = append(conds, "v.is_favorite = 1")
		}

		where := ""
		if len(conds) > 0 {
			where = "WHERE " + strings.Join(conds, " AND ")
		}
		orderClause := "ORDER BY v.created_at DESC"
		switch strings.ToLower(sortBy) {
		case "rating":
			orderClause = "ORDER BY COALESCE(v.rating, 0) DESC"
		case "duration":
			orderClause = "ORDER BY COALESCE(v.duration, 0) DESC"
		case "title":
			orderClause = "ORDER BY v.title ASC"
		case "random":
			orderClause = "ORDER BY RANDOM()"
		}

		searchQ := fmt.Sprintf(`
			SELECT v.id, v.title,
			       COALESCE(v.duration, 0),
			       COALESCE(v.rating, 0),
			       COALESCE(v.is_favorite, 0),
			       COALESCE(v.thumbnail_path, ''),
			       COALESCE(l.name, ''),
			       COALESCE((SELECT GROUP_CONCAT(p2.name, ', ')
			                 FROM video_performers vp2 JOIN performers p2 ON p2.id = vp2.performer_id
			                 WHERE vp2.video_id = v.id), '') AS performers,
			       COALESCE((SELECT GROUP_CONCAT(t2.name, ', ')
			                 FROM video_tags vt2 JOIN tags t2 ON t2.id = vt2.tag_id
			                 WHERE vt2.video_id = v.id), '') AS tags
			FROM videos v
			LEFT JOIN libraries l ON v.library_id = l.id
			%s
			%s
			LIMIT ?`, where, orderClause)

		qArgs = append(qArgs, limit)
		rows, err := s.db.Query(searchQ, qArgs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath, libName, performers, tags string
			var duration float64
			var rating, isFav int
			if rows.Scan(&id, &title, &duration, &rating, &isFav, &thumbPath, &libName, &performers, &tags) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title,
					"duration":       formatSeconds(duration),
					"duration_s":     int(duration),
					"rating":         rating,
					"is_favorite":    isFav > 0,
					"thumbnail_path": thumbPath,
					"library":        libName,
					"performers":     performers,
					"tags":           tags,
				})
			}
		}
		return map[string]interface{}{"count": len(results), "videos": results}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_video_details",
			Description: "Get full details for a specific video by ID: title, file path, duration, rating, favorite status, performers, tags, studios, and library.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to look up"},
			}, "required": []string{"video_id"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		var title, filePath, thumbPath, libName string
		var duration float64
		var rating, isFav int
		err := s.db.QueryRow(`
			SELECT v.title, v.file_path, COALESCE(v.thumbnail_path,''),
			       COALESCE(v.duration,0), COALESCE(v.rating,0), COALESCE(v.is_favorite,0),
			       COALESCE(l.name,'')
			FROM videos v LEFT JOIN libraries l ON v.library_id = l.id
			WHERE v.id = ?`, videoID).Scan(&title, &filePath, &thumbPath, &duration, &rating, &isFav, &libName)
		if err == sql.ErrNoRows {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		}
		if err != nil {
			return nil, err
		}

		// Performers
		perfRows, _ := s.db.Query(`SELECT p.id, p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?`, videoID)
		var performers []map[string]interface{}
		if perfRows != nil {
			for perfRows.Next() {
				var pid int64
				var pname string
				if perfRows.Scan(&pid, &pname) == nil {
					performers = append(performers, map[string]interface{}{"id": pid, "name": pname})
				}
			}
			perfRows.Close()
		}

		// Tags
		tagRows, _ := s.db.Query(`SELECT t.id, t.name FROM tags t JOIN video_tags vt ON t.id = vt.tag_id WHERE vt.video_id = ?`, videoID)
		var tags []map[string]interface{}
		if tagRows != nil {
			for tagRows.Next() {
				var tid int64
				var tname string
				if tagRows.Scan(&tid, &tname) == nil {
					tags = append(tags, map[string]interface{}{"id": tid, "name": tname})
				}
			}
			tagRows.Close()
		}

		// Studios
		stRows, _ := s.db.Query(`SELECT st.id, st.name FROM studios st JOIN video_studios vs ON st.id = vs.studio_id WHERE vs.video_id = ?`, videoID)
		var studios []map[string]interface{}
		if stRows != nil {
			for stRows.Next() {
				var sid int64
				var sname string
				if stRows.Scan(&sid, &sname) == nil {
					studios = append(studios, map[string]interface{}{"id": sid, "name": sname})
				}
			}
			stRows.Close()
		}

		return map[string]interface{}{
			"id":             videoID,
			"title":          title,
			"file_path":      filePath,
			"thumbnail_path": thumbPath,
			"duration":       formatSeconds(duration),
			"duration_s":     int(duration),
			"rating":         rating,
			"is_favorite":    isFav > 0,
			"library":        libName,
			"performers":     performers,
			"tags":           tags,
			"studios":        studios,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "update_video_metadata",
			Description: "Update the title and/or notes for a specific video. Use when the user asks to rename a video or add notes to it.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to update"},
				"title":    map[string]interface{}{"type": "string", "description": "New title (optional — omit to leave unchanged)"},
				"notes":    map[string]interface{}{"type": "string", "description": "Notes to set on the video (optional — omit to leave unchanged)"},
			}, "required": []string{"video_id"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		newTitle, hasTitle := args["title"].(string)
		newNotes, hasNotes := args["notes"].(string)
		if !hasTitle && !hasNotes {
			return map[string]interface{}{"error": "provide at least one of: title, notes"}, nil
		}
		var oldTitle string
		if err := s.db.QueryRow("SELECT title FROM videos WHERE id = ?", videoID).Scan(&oldTitle); err == sql.ErrNoRows {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		}
		if hasTitle && strings.TrimSpace(newTitle) != "" {
			s.db.Exec("UPDATE videos SET title = ? WHERE id = ?", strings.TrimSpace(newTitle), videoID)
		}
		if hasNotes {
			s.db.Exec("UPDATE videos SET notes = ? WHERE id = ?", newNotes, videoID)
		}
		return map[string]interface{}{"video_id": videoID, "updated": true, "old_title": oldTitle}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_favorites",
			Description: "Get the user's favorited videos with titles, ratings, and performers.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Max results (default: 20)"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		limit := intArg(args, "limit", 20)
		rows, err := s.db.Query(`
			SELECT v.id, v.title, COALESCE(v.duration, 0), COALESCE(v.rating, 0),
			       COALESCE(v.thumbnail_path, ''),
			       GROUP_CONCAT(p.name, ', ') as performers
			FROM videos v
			LEFT JOIN video_performers vp ON v.id = vp.video_id
			LEFT JOIN performers p ON vp.performer_id = p.id
			WHERE v.is_favorite = 1
			GROUP BY v.id
			ORDER BY v.updated_at DESC
			LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath string
			var duration float64
			var rating int
			var performers sql.NullString
			if rows.Scan(&id, &title, &duration, &rating, &thumbPath, &performers) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title, "duration": formatSeconds(duration),
					"rating": rating, "is_favorite": true,
					"thumbnail_path": thumbPath,
					"performers":     performers.String,
				})
			}
		}
		return map[string]interface{}{"favorites": results, "count": len(results)}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_top_rated",
			Description: "Get the highest-rated videos sorted by rating descending, with performer names.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit":      map[string]interface{}{"type": "number", "description": "Max results (default: 20)"},
				"min_rating": map[string]interface{}{"type": "number", "description": "Minimum rating to include (1-5, default: 1)"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		limit := intArg(args, "limit", 20)
		minRating := intArg(args, "min_rating", 1)
		rows, err := s.db.Query(`
			SELECT v.id, v.title, COALESCE(v.duration, 0), v.rating,
			       COALESCE(v.thumbnail_path, ''),
			       GROUP_CONCAT(p.name, ', ') as performers
			FROM videos v
			LEFT JOIN video_performers vp ON v.id = vp.video_id
			LEFT JOIN performers p ON vp.performer_id = p.id
			WHERE v.rating >= ?
			GROUP BY v.id
			ORDER BY v.rating DESC, v.updated_at DESC
			LIMIT ?`, minRating, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var title, thumbPath string
			var duration float64
			var rating int
			var performers sql.NullString
			if rows.Scan(&id, &title, &duration, &rating, &thumbPath, &performers) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "title": title, "duration": formatSeconds(duration),
					"rating":         rating,
					"thumbnail_path": thumbPath,
					"performers":     performers.String,
				})
			}
		}
		return map[string]interface{}{"top_rated": results, "count": len(results)}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "set_video_rating",
			Description: "Set the star rating of a video (0-5). Use when the user asks to rate a video or says something like 'give this a 4 star rating'. 0 clears the rating.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to rate"},
				"rating":   map[string]interface{}{"type": "number", "description": "Rating from 0 (clear) to 5 (max)"},
			}, "required": []string{"video_id", "rating"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		rating := intArg(args, "rating", -1)
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		if rating < 0 || rating > 5 {
			return map[string]interface{}{"error": "rating must be 0-5"}, nil
		}
		var oldRating int
		var title string
		err := s.db.QueryRow(`SELECT COALESCE(rating, 0), title FROM videos WHERE id = ?`, videoID).Scan(&oldRating, &title)
		if err == sql.ErrNoRows {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		} else if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		_, err = s.db.Exec(`UPDATE videos SET rating = ? WHERE id = ?`, rating, videoID)
		if err != nil {
			return map[string]interface{}{"error": "update failed: " + err.Error()}, nil
		}
		log.Printf("[AI] set_video_rating: video %d (%s) rated %d (was %d)", videoID, title, rating, oldRating)
		return map[string]interface{}{"video_id": videoID, "title": title, "old_rating": oldRating, "new_rating": rating}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "toggle_video_favorite",
			Description: "Toggle the favorite flag on a video — marks it as a favorite if not already, or removes the favorite if it is. Use when the user says 'favorite this video', 'add to favorites', or 'remove from favorites'.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to toggle favorite on"},
			}, "required": []string{"video_id"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		var isFav int
		var title string
		err := s.db.QueryRow(`SELECT COALESCE(is_favorite, 0), title FROM videos WHERE id = ?`, videoID).Scan(&isFav, &title)
		if err == sql.ErrNoRows {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		} else if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		newFav := 0
		if isFav == 0 {
			newFav = 1
		}
		_, err = s.db.Exec(`UPDATE videos SET is_favorite = ? WHERE id = ?`, newFav, videoID)
		if err != nil {
			return map[string]interface{}{"error": "update failed: " + err.Error()}, nil
		}
		action := "added to favorites"
		if newFav == 0 {
			action = "removed from favorites"
		}
		log.Printf("[AI] toggle_video_favorite: video %d (%s) %s", videoID, title, action)
		return map[string]interface{}{"video_id": videoID, "title": title, "is_favorite": newFav == 1, "action": action}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "find_duplicate_videos",
			Description: "Scan the video library for likely duplicate files — same filename in multiple places, same title across libraries, or same performers with similar duration. Returns suspect pairs grouped by confidence level. Use this when the user asks to find duplicates or clean up the library.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Restrict search to a specific library (optional — omit to search all libraries)"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max number of duplicate groups to return (default: 30)"},
			}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		targetLib, _ := args["library_name"].(string)
		limit := intArg(args, "limit", 30)

		type vidInfo struct {
			ID        int64
			Title     string
			FilePath  string
			Duration  float64
			LibID     int64
			LibName   string
			Filename  string
			NormTitle string
			PerfKey   string
		}

		baseQ := `SELECT v.id, v.title, v.file_path, COALESCE(v.duration, 0), v.library_id, l.name
		           FROM videos v JOIN libraries l ON v.library_id = l.id`
		var qArgs []interface{}
		if targetLib != "" {
			baseQ += " WHERE LOWER(l.name) LIKE ?"
			qArgs = append(qArgs, "%"+strings.ToLower(targetLib)+"%")
		}
		rows, err := s.db.Query(baseQ, qArgs...)
		if err != nil {
			return nil, err
		}
		var videos []vidInfo
		for rows.Next() {
			var v vidInfo
			rows.Scan(&v.ID, &v.Title, &v.FilePath, &v.Duration, &v.LibID, &v.LibName)
			v.Filename = strings.ToLower(filepath.Base(v.FilePath))
			v.NormTitle = normalizeVideoTitle(v.Title)
			videos = append(videos, v)
		}
		rows.Close()

		perfsByVideo := map[int64][]string{}
		perfRows, _ := s.db.Query(`SELECT vp.video_id, p.name FROM video_performers vp
			JOIN performers p ON p.id = vp.performer_id ORDER BY vp.video_id, p.name`)
		if perfRows != nil {
			for perfRows.Next() {
				var vid int64
				var name string
				perfRows.Scan(&vid, &name)
				perfsByVideo[vid] = append(perfsByVideo[vid], name)
			}
			perfRows.Close()
		}
		for i := range videos {
			if perfs := perfsByVideo[videos[i].ID]; len(perfs) > 0 {
				videos[i].PerfKey = strings.Join(perfs, "|")
			}
		}

		type dupGroup struct {
			Reason     string                   `json:"reason"`
			Confidence float64                  `json:"confidence"`
			Videos     []map[string]interface{} `json:"videos"`
		}
		seen := map[string]bool{}
		var groups []dupGroup

		toEntry := func(v vidInfo) map[string]interface{} {
			return map[string]interface{}{
				"id": v.ID, "title": v.Title,
				"library": v.LibName, "file": v.Filename,
				"duration_s": int(v.Duration),
			}
		}
		pairKey := func(a, b int64) string {
			if a < b {
				return fmt.Sprintf("%d:%d", a, b)
			}
			return fmt.Sprintf("%d:%d", b, a)
		}
		addPair := func(a, b vidInfo, reason string, conf float64) {
			pk := pairKey(a.ID, b.ID)
			if seen[pk] {
				return
			}
			seen[pk] = true
			groups = append(groups, dupGroup{
				Reason:     reason,
				Confidence: conf,
				Videos:     []map[string]interface{}{toEntry(a), toEntry(b)},
			})
		}

		filenameIdx := map[string][]vidInfo{}
		for _, v := range videos {
			if v.Filename == "" {
				continue
			}
			filenameIdx[v.Filename] = append(filenameIdx[v.Filename], v)
		}
		for _, vids := range filenameIdx {
			if len(vids) < 2 {
				continue
			}
			for i := 0; i < len(vids); i++ {
				for j := i + 1; j < len(vids); j++ {
					addPair(vids[i], vids[j], "identical filename", 0.95)
				}
			}
		}

		titleIdx := map[string][]vidInfo{}
		for _, v := range videos {
			if len(v.NormTitle) < 5 {
				continue
			}
			titleIdx[v.NormTitle] = append(titleIdx[v.NormTitle], v)
		}
		for _, vids := range titleIdx {
			if len(vids) < 2 {
				continue
			}
			for i := 0; i < len(vids); i++ {
				for j := i + 1; j < len(vids); j++ {
					if vids[i].LibID == vids[j].LibID {
						continue
					}
					durDiff := vids[i].Duration - vids[j].Duration
					if durDiff < 0 {
						durDiff = -durDiff
					}
					conf := 0.75
					reason := "same title (different libraries)"
					if durDiff <= 10 {
						conf = 0.88
						reason = fmt.Sprintf("same title, duration matches (±%.0fs)", durDiff)
					}
					addPair(vids[i], vids[j], reason, conf)
				}
			}
		}

		perfDurIdx := map[string][]vidInfo{}
		for _, v := range videos {
			if v.PerfKey == "" {
				continue
			}
			bucket := (int(v.Duration) / 15) * 15
			key := fmt.Sprintf("%s@%d", v.PerfKey, bucket)
			perfDurIdx[key] = append(perfDurIdx[key], v)
		}
		for _, vids := range perfDurIdx {
			if len(vids) < 2 {
				continue
			}
			for i := 0; i < len(vids); i++ {
				for j := i + 1; j < len(vids); j++ {
					if vids[i].LibID == vids[j].LibID {
						continue
					}
					durDiff := vids[i].Duration - vids[j].Duration
					if durDiff < 0 {
						durDiff = -durDiff
					}
					if durDiff > 30 {
						continue
					}
					addPair(vids[i], vids[j], fmt.Sprintf("same performers, similar duration (±%.0fs)", durDiff), 0.70)
				}
			}
		}

		sort.Slice(groups, func(i, j int) bool { return groups[i].Confidence > groups[j].Confidence })
		if len(groups) > limit {
			groups = groups[:limit]
		}
		return map[string]interface{}{
			"total_suspects": len(groups),
			"duplicates":     groups,
			"note":           "Review each pair — same filename/title is a strong signal, same performers+duration is weaker. Use video IDs to move or delete.",
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "find_orphaned_videos",
			Description: "Check which videos in the database no longer have their file on disk — the file was moved, renamed, or deleted outside the app. Returns a list of orphaned DB entries. Always run this before delete_video_records.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Restrict check to a specific library (optional)"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max videos to scan (default: 2000)"},
			}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libFilter, _ := args["library_name"].(string)
		limit := intArg(args, "limit", 2000)

		baseQ := `SELECT v.id, v.title, v.file_path, COALESCE(l.name, '') FROM videos v LEFT JOIN libraries l ON v.library_id = l.id`
		var qArgs []interface{}
		if libFilter != "" {
			baseQ += " WHERE LOWER(l.name) LIKE LOWER(?)"
			qArgs = append(qArgs, "%"+libFilter+"%")
		}
		baseQ += " LIMIT ?"
		qArgs = append(qArgs, limit)

		rows, err := s.db.Query(baseQ, qArgs...)
		if err != nil {
			return nil, err
		}
		type orphan struct {
			ID       int64  `json:"id"`
			Title    string `json:"title"`
			FilePath string `json:"file_path"`
			Library  string `json:"library"`
		}
		var orphans []orphan
		var scanned int
		for rows.Next() {
			var o orphan
			rows.Scan(&o.ID, &o.Title, &o.FilePath, &o.Library)
			scanned++
			if _, statErr := os.Stat(o.FilePath); os.IsNotExist(statErr) {
				orphans = append(orphans, o)
			}
		}
		rows.Close()
		return map[string]interface{}{
			"scanned":        scanned,
			"orphaned_count": len(orphans),
			"orphans":        orphans,
			"note":           "These DB entries have no file on disk. Use delete_video_records with the listed IDs to remove them — files are NOT touched.",
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "delete_video_records",
			Description: "Delete video entries from the database by ID. Only deletes the DB record — never touches files on disk. Use this to clean up orphaned entries found by find_orphaned_videos. ALWAYS confirm with the user before calling this tool.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_ids": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "List of video IDs to remove from the database",
				},
			}, "required": []string{"video_ids"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rawIDs, ok := args["video_ids"]
		if !ok {
			return map[string]interface{}{"error": "video_ids required"}, nil
		}
		idSlice, ok := rawIDs.([]interface{})
		if !ok || len(idSlice) == 0 {
			return map[string]interface{}{"error": "video_ids must be a non-empty array"}, nil
		}
		var ids []int64
		for _, raw := range idSlice {
			switch v := raw.(type) {
			case float64:
				ids = append(ids, int64(v))
			case int64:
				ids = append(ids, v)
			case int:
				ids = append(ids, int64(v))
			}
		}
		if len(ids) == 0 {
			return map[string]interface{}{"error": "no valid video IDs provided"}, nil
		}
		ph := make([]string, len(ids))
		iArgs := make([]interface{}, len(ids))
		for i, id := range ids {
			ph[i] = "?"
			iArgs[i] = id
		}
		inClause := strings.Join(ph, ",")
		tables := []string{"video_performers", "video_tags", "video_studios", "dashboard_feed_items"}
		colMap := map[string]string{
			"video_performers": "video_id", "video_tags": "video_id",
			"video_studios": "video_id", "dashboard_feed_items": "item_id",
		}
		for _, tbl := range tables {
			col := colMap[tbl]
			_, _ = s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE %s IN (%s)", tbl, col, inClause), iArgs...)
		}
		res, err := s.db.Exec(fmt.Sprintf("DELETE FROM videos WHERE id IN (%s)", inClause), iArgs...)
		if err != nil {
			return map[string]interface{}{"error": "delete failed: " + err.Error()}, nil
		}
		deleted, _ := res.RowsAffected()
		log.Printf("[AI] delete_video_records: removed %d DB entries (IDs: %v)", deleted, ids)
		return map[string]interface{}{
			"deleted_count": deleted,
			"requested_ids": ids,
			"note":          "Database records removed. No files were touched on disk.",
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "find_untagged_videos",
			Description: "Find videos that have no tags, optionally filtered by library or performer. Returns video IDs, titles, and performer names so you can call suggest_tags_for_video then apply_tags_to_video on each.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Restrict to a specific library (optional)"},
				"performer":    map[string]interface{}{"type": "string", "description": "Restrict to videos featuring a specific performer (optional)"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max results (default: 20, max: 50)"},
			}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libFilter, _ := args["library_name"].(string)
		perfFilter, _ := args["performer"].(string)

		var conds []string
		var qArgs []interface{}
		conds = append(conds, `NOT EXISTS (SELECT 1 FROM video_tags vt WHERE vt.video_id = v.id)`)
		if libFilter != "" {
			conds = append(conds, "LOWER(l.name) LIKE LOWER(?)")
			qArgs = append(qArgs, "%"+libFilter+"%")
		}
		if perfFilter != "" {
			conds = append(conds, `EXISTS (SELECT 1 FROM video_performers vp JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = v.id AND LOWER(p.name) LIKE LOWER(?))`)
			qArgs = append(qArgs, "%"+perfFilter+"%")
		}
		where := "WHERE " + strings.Join(conds, " AND ")

		var totalCount int
		s.db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM videos v LEFT JOIN libraries l ON l.id=v.library_id %s`, where), qArgs...).Scan(&totalCount)

		idRows, err := s.db.Query(fmt.Sprintf(`SELECT v.id FROM videos v LEFT JOIN libraries l ON l.id=v.library_id %s ORDER BY v.created_at DESC LIMIT 200`, where), qArgs...)
		if err != nil {
			return nil, err
		}
		defer idRows.Close()
		var videoIDs []int64
		for idRows.Next() {
			var id int64
			if idRows.Scan(&id) == nil {
				videoIDs = append(videoIDs, id)
			}
		}

		sampleArgs := append(qArgs, 5)
		sampleRows, _ := s.db.Query(fmt.Sprintf(`SELECT v.id, v.title FROM videos v LEFT JOIN libraries l ON l.id=v.library_id %s ORDER BY v.created_at DESC LIMIT ?`, where), sampleArgs...)
		var sample []map[string]interface{}
		if sampleRows != nil {
			defer sampleRows.Close()
			for sampleRows.Next() {
				var id int64
				var title string
				if sampleRows.Scan(&id, &title) == nil {
					sample = append(sample, map[string]interface{}{"id": id, "title": title})
				}
			}
		}
		return map[string]interface{}{
			"total_count": totalCount,
			"video_ids":   videoIDs,
			"sample":      sample,
			"note":        "Pass video_ids to bulk_apply_tags to tag all at once. Confirm with the user first.",
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "bulk_apply_tags",
			Description: "Automatically suggest and apply performer-based tags to a batch of videos in one operation. For each video, finds the most common tags from other videos featuring the same performers and applies them. ALWAYS tell the user how many videos will be tagged and confirm before running.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_ids": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "List of video IDs to auto-tag",
				},
				"max_tags_per_video": map[string]interface{}{"type": "number", "description": "Max tags to apply per video (default: 5)"},
			}, "required": []string{"video_ids"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rawIDs, ok := args["video_ids"]
		if !ok {
			return map[string]interface{}{"error": "video_ids required"}, nil
		}
		idSlice, _ := rawIDs.([]interface{})
		maxTags := intArg(args, "max_tags_per_video", 5)
		if maxTags < 1 {
			maxTags = 1
		}
		if maxTags > 15 {
			maxTags = 15
		}
		var videoIDs []int64
		for _, r := range idSlice {
			switch v := r.(type) {
			case float64:
				videoIDs = append(videoIDs, int64(v))
			case int64:
				videoIDs = append(videoIDs, v)
			case int:
				videoIDs = append(videoIDs, int64(v))
			}
		}
		if len(videoIDs) == 0 {
			return map[string]interface{}{"error": "no valid video IDs"}, nil
		}

		type bulkResult struct {
			ID          int64    `json:"id"`
			Title       string   `json:"title"`
			TagsApplied []string `json:"tags_applied"`
			Skipped     bool     `json:"skipped,omitempty"`
		}
		var results []bulkResult
		totalApplied := 0
		perfPH := func(n int) string {
			p := make([]string, n)
			for i := range p {
				p[i] = "?"
			}
			return strings.Join(p, ",")
		}

		for _, vid := range videoIDs {
			var title string
			s.db.QueryRow("SELECT title FROM videos WHERE id=?", vid).Scan(&title)

			perfRows, _ := s.db.Query("SELECT performer_id FROM video_performers WHERE video_id=?", vid)
			var perfArgs []interface{}
			for perfRows.Next() {
				var pid int64
				if perfRows.Scan(&pid) == nil {
					perfArgs = append(perfArgs, pid)
				}
			}
			perfRows.Close()

			if len(perfArgs) == 0 {
				results = append(results, bulkResult{ID: vid, Title: title, Skipped: true})
				continue
			}

			qArgs := append(perfArgs, vid, vid, maxTags)
			tagRows, err := s.db.Query(fmt.Sprintf(`
				SELECT t.id, t.name, COUNT(*) as freq
				FROM video_performers vp
				JOIN video_tags vt ON vt.video_id = vp.video_id
				JOIN tags t ON t.id = vt.tag_id
				WHERE vp.performer_id IN (%s)
				  AND vp.video_id != ?
				  AND NOT EXISTS (SELECT 1 FROM video_tags ex WHERE ex.video_id=? AND ex.tag_id=t.id)
				GROUP BY t.id, t.name
				ORDER BY freq DESC
				LIMIT ?`, perfPH(len(perfArgs))), qArgs...)

			var applied []string
			if err == nil {
				for tagRows.Next() {
					var tagID int64
					var tagName string
					var freq int
					if tagRows.Scan(&tagID, &tagName, &freq) == nil {
						if _, err := s.db.Exec("INSERT OR IGNORE INTO video_tags (video_id, tag_id) VALUES (?,?)", vid, tagID); err == nil {
							applied = append(applied, tagName)
						}
					}
				}
				tagRows.Close()
			}
			results = append(results, bulkResult{ID: vid, Title: title, TagsApplied: applied})
			totalApplied += len(applied)
		}

		skipped := 0
		for _, r := range results {
			if r.Skipped {
				skipped++
			}
		}
		log.Printf("[AI] bulk_apply_tags: processed %d videos, applied %d tags total", len(videoIDs), totalApplied)
		return map[string]interface{}{
			"processed":             len(videoIDs),
			"total_tags_applied":    totalApplied,
			"skipped_no_performers": skipped,
			"videos":                results,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "suggest_tags_for_video",
			Description: "Suggest tags for a video based on what tags are commonly applied to other videos featuring the same performers. Excludes tags already applied.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to suggest tags for"},
				"limit":    map[string]interface{}{"type": "number", "description": "Max suggestions to return (default: 10)"},
			}, "required": []string{"video_id"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		limit := intArg(args, "limit", 10)
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		perfRows, err := s.db.Query(`SELECT p.id FROM video_performers vp JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = ?`, videoID)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		var perfIDs []interface{}
		for perfRows.Next() {
			var pid int64
			if perfRows.Scan(&pid) == nil {
				perfIDs = append(perfIDs, pid)
			}
		}
		perfRows.Close()
		if len(perfIDs) == 0 {
			return map[string]interface{}{"suggestions": []interface{}{}, "note": "No performers linked to this video — cannot suggest tags"}, nil
		}

		existingRows, _ := s.db.Query(`SELECT tag_id FROM video_tags WHERE video_id = ?`, videoID)
		existingTags := map[int64]bool{}
		for existingRows.Next() {
			var tid int64
			if existingRows.Scan(&tid) == nil {
				existingTags[tid] = true
			}
		}
		existingRows.Close()

		ph := make([]string, len(perfIDs))
		for i := range perfIDs {
			ph[i] = "?"
		}
		inClause := strings.Join(ph, ",")
		qArgs := append(perfIDs, videoID)
		tagRows, err := s.db.Query(fmt.Sprintf(`
			SELECT t.id, t.name, COUNT(*) as freq
			FROM video_performers vp
			JOIN video_tags vt ON vt.video_id = vp.video_id
			JOIN tags t ON t.id = vt.tag_id
			WHERE vp.performer_id IN (%s)
			  AND vp.video_id != ?
			GROUP BY t.id, t.name
			ORDER BY freq DESC
			LIMIT ?`, inClause), append(qArgs, limit)...)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		defer tagRows.Close()

		var suggestions []map[string]interface{}
		for tagRows.Next() {
			var tid int64
			var tname string
			var freq int
			if tagRows.Scan(&tid, &tname, &freq) == nil && !existingTags[tid] {
				suggestions = append(suggestions, map[string]interface{}{"tag_id": tid, "name": tname, "frequency": freq})
			}
		}
		if suggestions == nil {
			suggestions = []map[string]interface{}{}
		}
		return map[string]interface{}{"video_id": videoID, "suggestions": suggestions, "count": len(suggestions)}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "apply_tags_to_video",
			Description: "Apply one or more tags to a video by name. Creates tags that don't exist yet.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to tag"},
				"tags": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "List of tag names to apply",
				},
			}, "required": []string{"video_id", "tags"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		rawTags, ok := args["tags"]
		if !ok {
			return map[string]interface{}{"error": "tags array required"}, nil
		}
		tagSlice, ok := rawTags.([]interface{})
		if !ok || len(tagSlice) == 0 {
			return map[string]interface{}{"error": "tags must be a non-empty array"}, nil
		}
		var applied, skipped []string
		for _, rawTag := range tagSlice {
			tagName, ok := rawTag.(string)
			if !ok || strings.TrimSpace(tagName) == "" {
				continue
			}
			tagName = strings.TrimSpace(tagName)
			var tagID int64
			err := s.db.QueryRow(`SELECT id FROM tags WHERE LOWER(name) = LOWER(?)`, tagName).Scan(&tagID)
			if err == sql.ErrNoRows {
				res, err := s.db.Exec(`INSERT INTO tags (name) VALUES (?)`, tagName)
				if err != nil {
					skipped = append(skipped, tagName+" (create failed)")
					continue
				}
				tagID, _ = res.LastInsertId()
			} else if err != nil {
				skipped = append(skipped, tagName+" (lookup failed)")
				continue
			}
			_, err = s.db.Exec(`INSERT OR IGNORE INTO video_tags (video_id, tag_id) VALUES (?, ?)`, videoID, tagID)
			if err != nil {
				skipped = append(skipped, tagName+" (link failed)")
			} else {
				applied = append(applied, tagName)
			}
		}
		return map[string]interface{}{
			"video_id": videoID,
			"applied":  applied,
			"skipped":  skipped,
			"note":     fmt.Sprintf("Applied %d tag(s) to video %d", len(applied), videoID),
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "remove_tags_from_video",
			Description: "Remove specific named tags from a video. Does not delete the tag itself — only removes the link between this video and the named tags.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video"},
				"tags": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Tag names to remove from this video",
				},
			}, "required": []string{"video_id", "tags"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		rawTags, ok := args["tags"]
		if !ok {
			return map[string]interface{}{"error": "tags array required"}, nil
		}
		tagSlice, _ := rawTags.([]interface{})
		var removed, notFound []string
		for _, rawTag := range tagSlice {
			tagName, ok := rawTag.(string)
			if !ok || strings.TrimSpace(tagName) == "" {
				continue
			}
			tagName = strings.TrimSpace(tagName)
			var tagID int64
			if err := s.db.QueryRow(`SELECT id FROM tags WHERE LOWER(name) = LOWER(?)`, tagName).Scan(&tagID); err != nil {
				notFound = append(notFound, tagName)
				continue
			}
			res, _ := s.db.Exec(`DELETE FROM video_tags WHERE video_id = ? AND tag_id = ?`, videoID, tagID)
			n, _ := res.RowsAffected()
			if n > 0 {
				removed = append(removed, tagName)
			} else {
				notFound = append(notFound, tagName+" (not applied)")
			}
		}
		return map[string]interface{}{"video_id": videoID, "removed": removed, "not_found": notFound}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "suggest_videos_to_watch",
			Description: "Recommend videos the user hasn't favorited or rated highly yet, based on their taste profile built from favorites and 4★+ rated videos. Matches by shared performers, tags, and studios. Use when the user asks 'what should I watch?', 'suggest something', or 'recommend a video'.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"count":   map[string]interface{}{"type": "number", "description": "How many suggestions to return (default: 5)"},
				"library": map[string]interface{}{"type": "string", "description": "Restrict suggestions to a specific library (optional)"},
			}},
		},
	}, s.toolSuggestVideosToWatch)

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_all_tags",
			Description: "Get all tags with their names and video counts.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rows, err := s.db.Query(`SELECT t.id, t.name, COUNT(vt.video_id) as cnt FROM tags t LEFT JOIN video_tags vt ON t.id = vt.tag_id GROUP BY t.id ORDER BY cnt DESC LIMIT 100`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var name string
			var cnt int
			if rows.Scan(&id, &name, &cnt) == nil {
				results = append(results, map[string]interface{}{"id": id, "name": name, "video_count": cnt})
			}
		}
		return results, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name: "compare_video_quality",
			Description: "Compare the technical quality of multiple videos by scoring resolution, codec, bitrate, and FPS. Returns a ranked list with scores (0–100) and a plain-English recommendation for which file to keep. Ideal when the same scene was downloaded from several sources or when deciding which copy to delete.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_ids": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "number"},
					"description": "IDs of the videos to compare (minimum 2, maximum 10)",
				},
			}, "required": []string{"video_ids"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rawIDs, _ := args["video_ids"].([]interface{})
		if len(rawIDs) < 2 {
			return nil, fmt.Errorf("at least 2 video IDs are required")
		}
		ids := make([]int64, 0, len(rawIDs))
		for _, raw := range rawIDs {
			switch v := raw.(type) {
			case float64:
				ids = append(ids, int64(v))
			case int64:
				ids = append(ids, v)
			}
		}
		svc := NewVideoQualityService()
		return svc.CompareByIDs(ids)
	})
}

// toolSuggestVideosToWatch is extracted as a named function due to its length.
func (s *AICompanionService) toolSuggestVideosToWatch(args map[string]interface{}) (interface{}, error) {
	count := intArg(args, "count", 5)
	if count > 20 {
		count = 20
	}
	libFilter, _ := args["library"].(string)

	profRows, err := s.db.Query(`SELECT id FROM videos WHERE is_favorite = 1 OR COALESCE(rating, 0) >= 4`)
	if err != nil {
		return nil, err
	}
	var profileIDs []int64
	profileSet := map[int64]bool{}
	for profRows.Next() {
		var id int64
		profRows.Scan(&id)
		profileIDs = append(profileIDs, id)
		profileSet[id] = true
	}
	profRows.Close()

	if len(profileIDs) == 0 {
		return map[string]interface{}{"message": "No favorites or 4★+ rated videos found. Rate or favorite some videos first so I can learn your taste."}, nil
	}

	perfFreq := map[int64]int{}
	tagFreq := map[int64]int{}
	studioFreq := map[int64]int{}

	buildPH := func(n int) string {
		ph := make([]string, n)
		for i := range ph {
			ph[i] = "?"
		}
		return strings.Join(ph, ",")
	}
	toIface := func(ids []int64) []interface{} {
		out := make([]interface{}, len(ids))
		for i, id := range ids {
			out[i] = id
		}
		return out
	}

	ph := buildPH(len(profileIDs))
	iargs := toIface(profileIDs)
	if pr, _ := s.db.Query(fmt.Sprintf("SELECT performer_id FROM video_performers WHERE video_id IN (%s)", ph), iargs...); pr != nil {
		for pr.Next() {
			var pid int64; pr.Scan(&pid); perfFreq[pid]++
		}
		pr.Close()
	}
	if tr, _ := s.db.Query(fmt.Sprintf("SELECT tag_id FROM video_tags WHERE video_id IN (%s)", ph), iargs...); tr != nil {
		for tr.Next() {
			var tid int64; tr.Scan(&tid); tagFreq[tid]++
		}
		tr.Close()
	}
	if sr, _ := s.db.Query(fmt.Sprintf("SELECT studio_id FROM video_studios WHERE video_id IN (%s)", ph), iargs...); sr != nil {
		for sr.Next() {
			var sid int64; sr.Scan(&sid); studioFreq[sid]++
		}
		sr.Close()
	}

	type kv64 struct{ k int64; v int }
	toSortedIDs := func(m map[int64]int, n int) []int64 {
		var pairs []kv64
		for k, v := range m {
			pairs = append(pairs, kv64{k, v})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].v > pairs[j].v })
		out := make([]int64, 0, n)
		for i, p := range pairs {
			if i >= n {
				break
			}
			out = append(out, p.k)
		}
		return out
	}
	perfIDList := toSortedIDs(perfFreq, 20)
	tagIDList := toSortedIDs(tagFreq, 20)
	studioIDList := toSortedIDs(studioFreq, 10)

	if len(perfIDList) == 0 && len(tagIDList) == 0 && len(studioIDList) == 0 {
		return map[string]interface{}{"message": "Your favorite videos don't have enough metadata to build recommendations. Try adding performer and tag data to your videos."}, nil
	}

	var scoreExpr strings.Builder
	var scoreArgs []interface{}
	if len(perfIDList) > 0 {
		scoreExpr.WriteString(fmt.Sprintf(`(SELECT COUNT(*) FROM video_performers vp2 WHERE vp2.video_id = v.id AND vp2.performer_id IN (%s)) * 3`, buildPH(len(perfIDList))))
		for _, id := range perfIDList {
			scoreArgs = append(scoreArgs, id)
		}
	}
	if len(tagIDList) > 0 {
		if scoreExpr.Len() > 0 {
			scoreExpr.WriteString(" + ")
		}
		scoreExpr.WriteString(fmt.Sprintf(`(SELECT COUNT(*) FROM video_tags vt2 WHERE vt2.video_id = v.id AND vt2.tag_id IN (%s)) * 1`, buildPH(len(tagIDList))))
		for _, id := range tagIDList {
			scoreArgs = append(scoreArgs, id)
		}
	}
	if len(studioIDList) > 0 {
		if scoreExpr.Len() > 0 {
			scoreExpr.WriteString(" + ")
		}
		scoreExpr.WriteString(fmt.Sprintf(`(SELECT COUNT(*) FROM video_studios vs2 WHERE vs2.video_id = v.id AND vs2.studio_id IN (%s)) * 2`, buildPH(len(studioIDList))))
		for _, id := range studioIDList {
			scoreArgs = append(scoreArgs, id)
		}
	}

	excludeIDs := profileIDs
	if len(excludeIDs) > 200 {
		excludeIDs = excludeIDs[:200]
	}
	libCond := ""
	var libArgs []interface{}
	if libFilter != "" {
		libCond = "AND LOWER(l.name) LIKE LOWER(?)"
		libArgs = append(libArgs, "%"+libFilter+"%")
	}

	candQ := fmt.Sprintf(`
		SELECT v.id, v.title, COALESCE(v.duration, 0), COALESCE(v.rating, 0), COALESCE(l.name, ''),
		       COALESCE(v.thumbnail_path, ''),
		       COALESCE((SELECT GROUP_CONCAT(p.name, ', ') FROM video_performers vp JOIN performers p ON p.id = vp.performer_id WHERE vp.video_id = v.id), '') AS performers,
		       (%s) AS score
		FROM videos v
		LEFT JOIN libraries l ON v.library_id = l.id
		WHERE v.id NOT IN (%s)
		  AND COALESCE(v.is_favorite, 0) = 0
		  %s
		HAVING score > 0
		ORDER BY score DESC, RANDOM()
		LIMIT ?`, scoreExpr.String(), buildPH(len(excludeIDs)), libCond)

	var allArgs []interface{}
	allArgs = append(allArgs, scoreArgs...)
	allArgs = append(allArgs, toIface(excludeIDs)...)
	allArgs = append(allArgs, libArgs...)
	allArgs = append(allArgs, count*3)

	candRows, err := s.db.Query(candQ, allArgs...)
	if err != nil {
		return nil, err
	}
	defer candRows.Close()

	type suggestion struct {
		ID            int64
		Title         string
		Duration      float64
		Rating        int
		Library       string
		ThumbnailPath string
		Performers    string
		Score         float64
	}
	var suggestions []suggestion
	for candRows.Next() {
		var sg suggestion
		candRows.Scan(&sg.ID, &sg.Title, &sg.Duration, &sg.Rating, &sg.Library, &sg.ThumbnailPath, &sg.Performers, &sg.Score)
		suggestions = append(suggestions, sg)
	}
	if len(suggestions) > count {
		suggestions = suggestions[:count]
	}

	results := make([]map[string]interface{}, 0, len(suggestions))
	for _, sg := range suggestions {
		entry := map[string]interface{}{
			"id": sg.ID, "title": sg.Title,
			"duration": formatSeconds(sg.Duration), "rating": sg.Rating,
			"thumbnail_path": sg.ThumbnailPath, "library": sg.Library,
			"performers": sg.Performers, "score": sg.Score,
		}
		if len(perfIDList) > 0 {
			mArgs := append([]interface{}{sg.ID}, toIface(perfIDList)...)
			mArgs = append(mArgs, 3)
			mRows, mErr := s.db.Query(fmt.Sprintf(`SELECT p.name FROM video_performers vp JOIN performers p ON p.id=vp.performer_id WHERE vp.video_id=? AND vp.performer_id IN (%s) LIMIT ?`, buildPH(len(perfIDList))), mArgs...)
			if mErr == nil {
				var mp []string
				for mRows.Next() {
					var n string
					if mRows.Scan(&n) == nil {
						mp = append(mp, n)
					}
				}
				mRows.Close()
				if len(mp) > 0 {
					entry["matched_performers"] = mp
				}
			}
		}
		if len(tagIDList) > 0 {
			tArgs := append([]interface{}{sg.ID}, toIface(tagIDList)...)
			tArgs = append(tArgs, 3)
			tRows, tErr := s.db.Query(fmt.Sprintf(`SELECT t.name FROM video_tags vt JOIN tags t ON t.id=vt.tag_id WHERE vt.video_id=? AND vt.tag_id IN (%s) LIMIT ?`, buildPH(len(tagIDList))), tArgs...)
			if tErr == nil {
				var mt []string
				for tRows.Next() {
					var n string
					if tRows.Scan(&n) == nil {
						mt = append(mt, n)
					}
				}
				tRows.Close()
				if len(mt) > 0 {
					entry["matched_tags"] = mt
				}
			}
		}
		results = append(results, entry)
	}
	return map[string]interface{}{
		"suggestions":    results,
		"based_on":       fmt.Sprintf("taste profile built from %d favorite/4★+ videos", len(profileIDs)),
		"top_performers": len(perfIDList),
		"top_tags":       len(tagIDList),
	}, nil
}

