package services

import (
	"encoding/json"
	"time"
)

func (s *AICompanionService) registerCoreTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_library_stats",
			Description: "Get overall statistics about the video library including total videos, performers, tags, studios, and database size.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		var videoCount, performerCount, tagCount, libraryCount, studioCount int
		s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
		s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
		s.db.QueryRow("SELECT COUNT(*) FROM libraries").Scan(&libraryCount)
		s.db.QueryRow("SELECT COUNT(*) FROM studios").Scan(&studioCount)
		return map[string]interface{}{
			"total_videos": videoCount, "total_performers": performerCount,
			"total_tags": tagCount, "total_libraries": libraryCount, "total_studios": studioCount,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_daily_briefing",
			Description: "Generate a fresh daily briefing with library stats, new videos since yesterday, unorganized downloads, untagged count, and recent errors. Use this when the user asks for a status update, summary, or 'what happened today'.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		var vidCount, perfCount, downloadsCount, untaggedCount, errorCount, runningTasks int
		s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&vidCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&perfCount)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos v JOIN libraries l ON v.library_id = l.id WHERE LOWER(l.name) LIKE '%download%'`).Scan(&downloadsCount)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE id NOT IN (SELECT DISTINCT video_id FROM video_tags)`).Scan(&untaggedCount)
		s.db.QueryRow(`SELECT COUNT(*) FROM console_logs WHERE level = 'error' AND created_at > datetime('now', '-24 hours')`).Scan(&errorCount)
		s.db.QueryRow(`SELECT COUNT(*) FROM activity_logs WHERE status = 'running'`).Scan(&runningTasks)

		var snapshotJSON string
		s.db.QueryRow("SELECT value FROM memories WHERE key = 'daily_snapshot'").Scan(&snapshotJSON)
		prevCount := vidCount
		if snapshotJSON != "" {
			var snap map[string]int
			if json.Unmarshal([]byte(snapshotJSON), &snap) == nil {
				prevCount = snap["video_count"]
			}
		}
		newVideos := vidCount - prevCount

		type recentJob struct {
			TaskType  string `json:"task_type"`
			Message   string `json:"message"`
			Status    string `json:"status"`
			CreatedAt string `json:"created_at"`
		}
		var recentJobs []recentJob
		jobRows, _ := s.db.Query(`SELECT task_type, COALESCE(message,''), status, created_at FROM activity_logs
			WHERE created_at > datetime('now', '-24 hours') AND status IN ('completed','failed')
			ORDER BY created_at DESC LIMIT 5`)
		if jobRows != nil {
			for jobRows.Next() {
				var j recentJob
				jobRows.Scan(&j.TaskType, &j.Message, &j.Status, &j.CreatedAt)
				recentJobs = append(recentJobs, j)
			}
			jobRows.Close()
		}
		return map[string]interface{}{
			"total_videos":      vidCount,
			"total_performers":  perfCount,
			"new_videos_delta":  newVideos,
			"downloads_waiting": downloadsCount,
			"untagged_videos":   untaggedCount,
			"errors_24h":        errorCount,
			"running_tasks":     runningTasks,
			"recent_jobs":       recentJobs,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_library_health",
			Description: "Generate a comprehensive health report of the entire library: total counts, untagged %, unrated %, videos with no performers, recent additions, and a health score (0-100). Use when the user asks 'how's my library?', 'library health', or 'give me an overview'.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		var total, rated, untagged, noPerformers, recentAdded int
		s.db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&total)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE rating IS NOT NULL AND rating > 0`).Scan(&rated)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos v WHERE NOT EXISTS (SELECT 1 FROM video_tags vt WHERE vt.video_id = v.id)`).Scan(&untagged)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos v WHERE NOT EXISTS (SELECT 1 FROM video_performers vp WHERE vp.video_id = v.id)`).Scan(&noPerformers)
		s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE created_at > datetime('now', '-7 days')`).Scan(&recentAdded)

		var totalPerformers, totalTags, totalStudios, totalLibraries int
		s.db.QueryRow(`SELECT COUNT(*) FROM performers`).Scan(&totalPerformers)
		s.db.QueryRow(`SELECT COUNT(*) FROM tags`).Scan(&totalTags)
		s.db.QueryRow(`SELECT COUNT(*) FROM studios`).Scan(&totalStudios)
		s.db.QueryRow(`SELECT COUNT(*) FROM libraries`).Scan(&totalLibraries)

		libRows, _ := s.db.Query(`SELECT l.name, COUNT(v.id) as cnt FROM libraries l LEFT JOIN videos v ON v.library_id = l.id GROUP BY l.id, l.name ORDER BY cnt DESC`)
		var libraries []map[string]interface{}
		if libRows != nil {
			for libRows.Next() {
				var lname string
				var cnt int
				if libRows.Scan(&lname, &cnt) == nil {
					libraries = append(libraries, map[string]interface{}{"name": lname, "videos": cnt})
				}
			}
			libRows.Close()
		}

		score := 100
		if total > 0 {
			untaggedPct := untagged * 100 / total
			unratedPct := (total - rated) * 100 / total
			noPerformerPct := noPerformers * 100 / total
			if untaggedPct > 80 {
				score -= 30
			} else if untaggedPct > 50 {
				score -= 20
			} else if untaggedPct > 20 {
				score -= 10
			}
			if unratedPct > 80 {
				score -= 25
			} else if unratedPct > 60 {
				score -= 15
			} else if unratedPct > 40 {
				score -= 8
			}
			if noPerformerPct > 50 {
				score -= 20
			} else if noPerformerPct > 25 {
				score -= 10
			} else if noPerformerPct > 10 {
				score -= 5
			}
		}
		if score < 0 {
			score = 0
		}
		grade := "A"
		switch {
		case score < 40:
			grade = "D"
		case score < 55:
			grade = "C"
		case score < 70:
			grade = "B"
		}
		pct := func(n int) int {
			if total == 0 {
				return 0
			}
			return n * 100 / total
		}
		return map[string]interface{}{
			"health_score": score,
			"health_grade": grade,
			"totals": map[string]interface{}{
				"videos": total, "performers": totalPerformers,
				"tags": totalTags, "studios": totalStudios, "libraries": totalLibraries,
			},
			"untagged":         untagged,
			"untagged_pct":     pct(untagged),
			"unrated":          total - rated,
			"unrated_pct":      pct(total - rated),
			"no_performers":    noPerformers,
			"no_performer_pct": pct(noPerformers),
			"recent_additions": recentAdded,
			"libraries":        libraries,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_running_tasks",
			Description: "Get currently running tasks/activities in the app (scraping, scanning, etc).",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rows, err := s.db.Query(`SELECT id, task_type, status, message, created_at FROM activity_logs WHERE status = 'running' ORDER BY created_at DESC LIMIT 20`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var taskType, status, message string
			var createdAt time.Time
			if rows.Scan(&id, &taskType, &status, &message, &createdAt) == nil {
				results = append(results, map[string]interface{}{
					"id": id, "task_type": taskType, "status": status, "message": message,
					"started_ago": formatDuration(time.Since(createdAt)),
				})
			}
		}
		return map[string]interface{}{"running_tasks": results, "count": len(results)}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_recent_activity",
			Description: "Get recent activity log entries showing what happened in the app.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Max entries (default: 10)"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		limit := intArg(args, "limit", 10)
		rows, err := s.db.Query(`SELECT task_type, status, message, created_at FROM activity_logs ORDER BY created_at DESC LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var taskType, status, message string
			var createdAt time.Time
			if rows.Scan(&taskType, &status, &message, &createdAt) == nil {
				results = append(results, map[string]interface{}{"task_type": taskType, "status": status, "message": message, "when": formatDuration(time.Since(createdAt)) + " ago"})
			}
		}
		return results, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_recent_errors",
			Description: "Get recent errors from the console log (last hour).",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Max errors to return (default: 10)"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		limit := intArg(args, "limit", 10)
		rows, err := s.db.Query(`SELECT source, level, message, created_at FROM console_logs WHERE level = 'error' ORDER BY created_at DESC LIMIT ?`, limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var source, level, message string
			var createdAt time.Time
			if rows.Scan(&source, &level, &message, &createdAt) == nil {
				results = append(results, map[string]interface{}{"source": source, "level": level, "message": message, "when": formatDuration(time.Since(createdAt)) + " ago"})
			}
		}
		return map[string]interface{}{"errors": results, "count": len(results)}, nil
	})
}
