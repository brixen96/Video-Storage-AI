package services

import (
	"fmt"
	"time"
)

func (s *AICompanionService) registerScraperTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_scraper_status",
			Description: "Get status of all scrapers including video counts per source, recent scrape activity, and auto-update configs.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		result := map[string]interface{}{}
		sources := []string{"pornhub", "luxuretv", "sexuria", "rule34video"}
		for _, src := range sources {
			var cnt int
			s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = ?", src).Scan(&cnt)
			result[src+"_count"] = cnt
		}
		var threadCount, postCount int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads").Scan(&threadCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts").Scan(&postCount)
		result["simpcity_threads"] = threadCount
		result["simpcity_posts"] = postCount

		rows, err := s.db.Query(`SELECT task_type, status, message, created_at FROM activity_logs WHERE task_type LIKE '%scrape%' OR task_type LIKE '%scraper%' ORDER BY created_at DESC LIMIT 5`)
		if err == nil {
			defer rows.Close()
			recent := []map[string]interface{}{}
			for rows.Next() {
				var taskType, status, message string
				var createdAt time.Time
				if rows.Scan(&taskType, &status, &message, &createdAt) == nil {
					recent = append(recent, map[string]interface{}{"task_type": taskType, "status": status, "message": message, "when": formatDuration(time.Since(createdAt)) + " ago"})
				}
			}
			result["recent_activity"] = recent
		}
		return result, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_jdownloader_status",
			Description: "Get JDownloader connection status, queue info, and recent download history.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		result := map[string]interface{}{}
		var connType, connStatus string
		err := s.db.QueryRow("SELECT connection_type, connection_status FROM jdownloader_config LIMIT 1").Scan(&connType, &connStatus)
		if err != nil {
			result["configured"] = false
			return result, nil
		}
		result["configured"] = true
		result["connection_type"] = connType
		result["connection_status"] = connStatus

		var sentCount, failedCount int
		s.db.QueryRow("SELECT COUNT(*) FROM jdownloader_history WHERE status = 'sent'").Scan(&sentCount)
		s.db.QueryRow("SELECT COUNT(*) FROM jdownloader_history WHERE status = 'failed'").Scan(&failedCount)
		result["total_sent"] = sentCount
		result["total_failed"] = failedCount

		rows, err := s.db.Query(`SELECT link_url, package_name, status, sent_at FROM jdownloader_history ORDER BY sent_at DESC LIMIT 5`)
		if err == nil {
			defer rows.Close()
			recent := []map[string]interface{}{}
			for rows.Next() {
				var url, pkg, status string
				var sentAt time.Time
				if rows.Scan(&url, &pkg, &status, &sentAt) == nil {
					recent = append(recent, map[string]interface{}{"url": url, "package": pkg, "status": status, "when": formatDuration(time.Since(sentAt)) + " ago"})
				}
			}
			result["recent_sends"] = recent
		}
		return result, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "search_scraped_content",
			Description: "Search scraped videos and forum posts by keyword.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"query":  map[string]interface{}{"type": "string", "description": "Search keyword"},
				"source": map[string]interface{}{"type": "string", "description": "Filter by source: pornhub, luxuretv, sexuria, rule34video, simpcity"},
				"limit":  map[string]interface{}{"type": "number", "description": "Max results (default: 10)"},
			}, "required": []string{"query"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		source, _ := args["source"].(string)
		limit := intArg(args, "limit", 10)
		if query == "" {
			return map[string]interface{}{"error": "query is required"}, nil
		}
		sqlQuery := `SELECT source, title, performers, url FROM scraped_videos WHERE (title LIKE ? OR performers LIKE ? OR tags LIKE ?)`
		sqlArgs := []interface{}{"%" + query + "%", "%" + query + "%", "%" + query + "%"}
		if source != "" {
			sqlQuery += " AND source = ?"
			sqlArgs = append(sqlArgs, source)
		}
		sqlQuery += " ORDER BY discovered_at DESC LIMIT ?"
		sqlArgs = append(sqlArgs, limit)
		rows, err := s.db.Query(sqlQuery, sqlArgs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		results := []map[string]interface{}{}
		for rows.Next() {
			var src, title, performers, url string
			if rows.Scan(&src, &title, &performers, &url) == nil {
				results = append(results, map[string]interface{}{"source": src, "title": title, "performers": performers, "url": url})
			}
		}
		return map[string]interface{}{"results": results, "count": len(results)}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "suggest_scraper_config",
			Description: "Fetch and analyze a website to suggest scraper configuration including site type and pagination pattern. Use this when the user wants to add a new scraper tab for a website and provides one or two page URLs.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page1_url": map[string]interface{}{"type": "string", "description": "URL of the first/listing page of the site"},
					"page2_url": map[string]interface{}{"type": "string", "description": "URL of the second page (optional but helps detect pagination pattern)"},
				},
				"required": []string{"page1_url"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		page1URL, _ := args["page1_url"].(string)
		page2URL, _ := args["page2_url"].(string)
		if page1URL == "" {
			return map[string]interface{}{"error": "page1_url is required"}, nil
		}
		return s.fetchSiteInfo(page1URL, page2URL)
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "fix_scraped_urls",
			Description: "Find and fix broken/old URLs in scraped data. Replaces deprecated domains (e.g. simpcity.is → simpcity.cr) across threads, posts, and download links. Use when a scraper reports fetch failures due to domain changes.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"old_domain":  map[string]interface{}{"type": "string", "description": "The old/broken domain to replace (e.g. 'simpcity.is')"},
					"new_domain":  map[string]interface{}{"type": "string", "description": "The new domain to use (e.g. 'simpcity.cr')"},
					"dry_run":     map[string]interface{}{"type": "boolean", "description": "If true, only count affected rows without changing them (default: true)"},
					"source":      map[string]interface{}{"type": "string", "description": "Filter by source (e.g. 'simpcity'). Leave empty for all sources."},
				},
				"required": []string{"old_domain", "new_domain"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		oldDomain, _ := args["old_domain"].(string)
		newDomain, _ := args["new_domain"].(string)
		dryRun, _ := args["dry_run"].(bool)
		source, _ := args["source"].(string)

		if oldDomain == "" || newDomain == "" {
			return map[string]interface{}{"error": "old_domain and new_domain are required"}, nil
		}
		// Default to dry run for safety
		if !args["dry_run"].(bool) && args["dry_run"] == nil {
			dryRun = true
		}

		type fixResult struct {
			Table       string `json:"table"`
			Affected    int    `json:"affected"`
			SampleURLs  []string `json:"sample_urls,omitempty"`
		}
		var fixes []fixResult
		tables := []struct {
			name   string
			urlCol string
			srcCol string
		}{
			{"scraped_threads", "url", "source"},
			{"scraped_posts", "source_url", "source"},
			{"scraped_download_links", "url", "provider"},
		}

		for _, t := range tables {
			// Count affected rows
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s LIKE ?",
				t.name, t.urlCol)
			countArgs := []interface{}{"%" + oldDomain + "%"}
			if source != "" && t.srcCol != "" {
				countQuery += fmt.Sprintf(" AND %s = ?", t.srcCol)
				countArgs = append(countArgs, source)
			}
			var count int
			s.db.QueryRow(countQuery, countArgs...).Scan(&count)

			fix := fixResult{Table: t.name, Affected: count}

			// Get sample URLs
			if count > 0 {
				sampleQuery := fmt.Sprintf("SELECT %s FROM %s WHERE %s LIKE ? LIMIT 3",
					t.urlCol, t.name, t.urlCol)
				sampleArgs := []interface{}{"%" + oldDomain + "%"}
				rows, err := s.db.Query(sampleQuery, sampleArgs...)
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var u string
						if rows.Scan(&u) == nil {
							fix.SampleURLs = append(fix.SampleURLs, u)
						}
					}
				}
			}

			// Apply fix if not dry run
			if !dryRun && count > 0 {
				updateQuery := fmt.Sprintf("UPDATE %s SET %s = REPLACE(%s, ?, ?)",
					t.name, t.urlCol, t.urlCol)
				updateArgs := []interface{}{oldDomain, newDomain}
				if source != "" && t.srcCol != "" {
					updateQuery += fmt.Sprintf(" WHERE %s LIKE ? AND %s = ?", t.urlCol, t.srcCol)
					updateArgs = append(updateArgs, "%"+oldDomain+"%", source)
				} else {
					updateQuery += fmt.Sprintf(" WHERE %s LIKE ?", t.urlCol)
					updateArgs = append(updateArgs, "%"+oldDomain+"%")
				}
				s.db.Exec(updateQuery, updateArgs...)
			}

			fixes = append(fixes, fix)
		}

		// Also fix auto-update configs
		var configCount int
		configQuery := "SELECT COUNT(*) FROM scraper_auto_update_configs WHERE config_json LIKE ?"
		configArgs := []interface{}{"%" + oldDomain + "%"}
		if source != "" {
			configQuery += " AND scraper_type = ?"
			configArgs = append(configArgs, source)
		}
		s.db.QueryRow(configQuery, configArgs...).Scan(&configCount)

		if !dryRun && configCount > 0 {
			configUpdate := "UPDATE scraper_auto_update_configs SET config_json = REPLACE(config_json, ?, ?) WHERE config_json LIKE ?"
			configUpdateArgs := []interface{}{oldDomain, newDomain, "%" + oldDomain + "%"}
			if source != "" {
				configUpdate += " AND scraper_type = ?"
				configUpdateArgs = append(configUpdateArgs, source)
			}
			s.db.Exec(configUpdate, configUpdateArgs...)
		}

		fixes = append(fixes, fixResult{Table: "scraper_auto_update_configs (config_json)", Affected: configCount})

		totalAffected := 0
		for _, f := range fixes {
			totalAffected += f.Affected
		}

		return map[string]interface{}{
			"action":         map[bool]string{true: "preview", false: "applied"}[dryRun],
			"old_domain":     oldDomain,
			"new_domain":     newDomain,
			"total_affected": totalAffected,
			"tables":         fixes,
		}, nil
	})
}
