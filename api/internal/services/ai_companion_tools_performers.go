package services

import (
	"fmt"
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
}
