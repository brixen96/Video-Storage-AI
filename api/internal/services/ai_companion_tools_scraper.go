package services

import (
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
}
