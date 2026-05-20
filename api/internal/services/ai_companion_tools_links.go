package services

import "fmt"

func (s *AICompanionService) registerLinksTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_video_link_stats",
			Description: "Get statistics about Edited↔Backup video pairing: how many confirmed links exist, how many are pending review, which library has the most unlinked videos, and overall link coverage percentage.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		var confirmed, suggested, rejected int
		s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'confirmed'").Scan(&confirmed)
		s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'suggested'").Scan(&suggested)
		s.db.QueryRow("SELECT COUNT(*) FROM video_links WHERE status = 'rejected'").Scan(&rejected)

		type libLinkStat struct {
			Name     string
			Total    int
			Linked   int
			Coverage float64
		}
		libRows, err := s.db.Query(`
			SELECT l.name, COUNT(v.id) AS total,
			       SUM(CASE WHEN EXISTS (
			           SELECT 1 FROM video_links vl
			           WHERE (vl.edited_video_id = v.id OR vl.backup_video_id = v.id)
			             AND vl.status = 'confirmed'
			       ) THEN 1 ELSE 0 END) AS linked
			FROM libraries l
			LEFT JOIN videos v ON v.library_id = l.id
			GROUP BY l.id ORDER BY l.name`)
		libStats := []libLinkStat{}
		if err == nil {
			defer libRows.Close()
			for libRows.Next() {
				var ls libLinkStat
				if libRows.Scan(&ls.Name, &ls.Total, &ls.Linked) == nil {
					if ls.Total > 0 {
						ls.Coverage = float64(ls.Linked) / float64(ls.Total) * 100
					}
					libStats = append(libStats, ls)
				}
			}
		}
		return map[string]interface{}{
			"confirmed_links":   confirmed,
			"suggested_links":   suggested,
			"rejected_links":    rejected,
			"total_links":       confirmed + suggested + rejected,
			"library_breakdown": libStats,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_unlinked_videos",
			Description: "List videos in a specific library that do not have a confirmed link to a video in the paired library. Use this to find which Edited videos are missing a Backup counterpart (or vice versa).",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name to check for unlinked videos (e.g. 'Edited' or 'Backup')"},
				"limit":        map[string]interface{}{"type": "number", "description": "Max results to return (default: 20)"},
			}, "required": []string{"library_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		limit := intArg(args, "limit", 20)
		if limit > 100 {
			limit = 100
		}
		if libraryName == "" {
			return map[string]interface{}{"error": "library_name is required"}, nil
		}
		rows, err := s.db.Query(`
			SELECT v.id, v.title, v.duration, COALESCE(v.is_missing, 0)
			FROM videos v
			INNER JOIN libraries l ON l.id = v.library_id
			WHERE LOWER(l.name) LIKE LOWER(?)
			  AND NOT EXISTS (
			      SELECT 1 FROM video_links vl
			      WHERE (vl.edited_video_id = v.id OR vl.backup_video_id = v.id)
			        AND vl.status = 'confirmed'
			  )
			ORDER BY v.title ASC
			LIMIT ?`, "%"+libraryName+"%", limit)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		defer rows.Close()

		type unlinkedEntry struct {
			ID        int64   `json:"id"`
			Title     string  `json:"title"`
			Duration  float64 `json:"duration"`
			IsMissing bool    `json:"is_missing"`
		}
		var unlinked []unlinkedEntry
		for rows.Next() {
			var e unlinkedEntry
			if rows.Scan(&e.ID, &e.Title, &e.Duration, &e.IsMissing) == nil {
				unlinked = append(unlinked, e)
			}
		}
		if unlinked == nil {
			unlinked = []unlinkedEntry{}
		}
		return map[string]interface{}{
			"library":        libraryName,
			"unlinked_count": len(unlinked),
			"videos":         unlinked,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "auto_link_by_path",
			Description: "Automatically link performers, tags, and studios to videos by analysing their file path and folder names. Safe to run without user confirmation since it only ADDS links, never removes anything. Run dry_run=true first to preview what would be applied.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name":  map[string]interface{}{"type": "string", "description": "Restrict to a specific library name (optional — omit to process all libraries)"},
				"only_unlinked": map[string]interface{}{"type": "boolean", "description": "Only process videos that have no performers yet (default: true)"},
				"dry_run":       map[string]interface{}{"type": "boolean", "description": "Preview what would be applied without making changes (default: false)"},
			}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		onlyUnlinked := false
		if v, ok := args["only_unlinked"].(bool); ok {
			onlyUnlinked = v
		}
		dryRun := false
		if v, ok := args["dry_run"].(bool); ok {
			dryRun = v
		}

		svc := NewPathAnalysisService()
		var libraryID int64
		if libraryName != "" {
			s.db.QueryRow(`SELECT id FROM libraries WHERE LOWER(name) LIKE LOWER(?)`, "%"+libraryName+"%").Scan(&libraryID)
		}

		report, err := svc.RunAutoLink(libraryID, onlyUnlinked, dryRun)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}

		action := "Applied"
		if dryRun {
			action = "Would apply (dry run)"
		}
		return map[string]interface{}{
			"dry_run":           dryRun,
			"videos_analyzed":   report.VideosAnalyzed,
			"performers_linked": report.PerformersLinked,
			"tags_applied":      report.TagsApplied,
			"studios_linked":    report.StudiosLinked,
			"summary": fmt.Sprintf("%s: %d performer links, %d tags, %d studio links across %d videos",
				action, report.PerformersLinked, report.TagsApplied, report.StudiosLinked, report.VideosAnalyzed),
		}, nil
	})
}
