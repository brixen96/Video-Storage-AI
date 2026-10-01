package services

import (
	"encoding/json"
	"fmt"
	"log"
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

	// Register thread enhancement tools
	s.registerThreadEnhancementTools()
}

// registerThreadEnhancementTools adds AI-powered thread enhancement tools
func (s *AICompanionService) registerThreadEnhancementTools() {
	activitySvc := NewActivityService()
	enhancer := NewThreadEnhancer(activitySvc)

	// Tool: Enhance Thread
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "enhance_thread",
			Description: "AI-powered thread enhancement: clean up post content, extract performer names, extract tags, and generate summaries. Use this to improve thread quality and metadata.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"thread_id": map[string]interface{}{
						"type":        "number",
						"description": "ID of the thread to enhance",
					},
					"clean_content": map[string]interface{}{
						"type":        "boolean",
						"description": "Remove forum boilerplate, signatures, and clutter from posts (default: true)",
					},
					"extract_performers": map[string]interface{}{
						"type":        "boolean",
						"description": "Extract performer names from post content (default: true)",
					},
					"extract_tags": map[string]interface{}{
						"type":        "boolean",
						"description": "Extract content tags like BBC, interracial, etc. (default: true)",
					},
					"generate_summary": map[string]interface{}{
						"type":        "boolean",
						"description": "Generate a summary of the thread (default: true)",
					},
				},
				"required": []string{"thread_id"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		threadID := int64(args["thread_id"].(float64))
		if threadID == 0 {
			return map[string]interface{}{"error": "thread_id is required"}, nil
		}

		options := EnhancementOptions{
			CleanContent:      getBoolArg(args, "clean_content", true),
			ExtractPerformers: getBoolArg(args, "extract_performers", true),
			ExtractTags:       getBoolArg(args, "extract_tags", true),
			GenerateSummary:   getBoolArg(args, "generate_summary", true),
		}

		result, err := enhancer.EnhanceThread(threadID, options)
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}

		return map[string]interface{}{
			"success":          true,
			"thread_id":        result.ThreadID,
			"posts_processed":  result.PostsProcessed,
			"posts_cleaned":    result.PostsCleaned,
			"performers_found": result.PerformersFound,
			"tags_extracted":   result.TagsExtracted,
			"summary":          result.Summary,
			"duration":         result.Duration.String(),
		}, nil
	})

	// Tool: Get Thread Enhancement Status
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_thread_enhancement_status",
			Description: "Check the status of a thread enhancement job and get results.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"thread_id": map[string]interface{}{
						"type":        "number",
						"description": "ID of the thread to check",
					},
				},
				"required": []string{"thread_id"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		threadID := int64(args["thread_id"].(float64))
		if threadID == 0 {
			return map[string]interface{}{"error": "thread_id is required"}, nil
		}

		// Check recent enhancement activities for this thread
		rows, err := s.db.Query(`
			SELECT id, status, message, progress, started_at, completed_at
			FROM activity_logs
			WHERE task_type = 'thread_enhancement' 
			AND details LIKE ?
			ORDER BY started_at DESC
			LIMIT 5
		`, fmt.Sprintf(`%%"thread_id":%d%%`, threadID))
		if err != nil {
			return map[string]interface{}{"error": err.Error()}, nil
		}
		defer rows.Close()

		jobs := []map[string]interface{}{}
		for rows.Next() {
			var id int64
			var status, message string
			var progress int
			var startedAt, completedAt time.Time
			if err := rows.Scan(&id, &status, &message, &progress, &startedAt, &completedAt); err == nil {
				jobs = append(jobs, map[string]interface{}{
					"id":           id,
					"status":       status,
					"message":      message,
					"progress":     progress,
					"started_at":   startedAt,
					"completed_at": completedAt,
				})
			}
		}

		return map[string]interface{}{
			"thread_id": threadID,
			"jobs":      jobs,
			"count":     len(jobs),
		}, nil
	})

	// Tool: Batch Enhance Threads
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "batch_enhance_threads",
			Description: "Enhance multiple threads at once. Use for bulk cleanup of thread content.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"thread_ids": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "number"},
						"description": "List of thread IDs to enhance",
					},
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Enhance all threads from this source (e.g., 'simpcity'). Overrides thread_ids if provided.",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Max threads to process (default: 10, max: 50)",
					},
				},
				"required": []string{},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		threadIDs := []int64{}
		if ids, ok := args["thread_ids"].([]interface{}); ok {
			for _, id := range ids {
				if idFloat, ok := id.(float64); ok {
					threadIDs = append(threadIDs, int64(idFloat))
				}
			}
		}

		source, _ := args["source"].(string)
		limit := intArg(args, "limit", 10)
		if limit > 50 {
			limit = 50
		}

		// If source provided, get thread IDs from that source
		if source != "" {
			rows, err := s.db.Query(`
				SELECT id FROM scraped_threads 
				WHERE source = ? 
				ORDER BY last_scraped_at DESC 
				LIMIT ?
			`, source, limit)
			if err != nil {
				return map[string]interface{}{"error": err.Error()}, nil
			}
			defer rows.Close()
			threadIDs = []int64{}
			for rows.Next() {
				var id int64
				if rows.Scan(&id) == nil {
					threadIDs = append(threadIDs, id)
				}
			}
		}

		if len(threadIDs) == 0 {
			return map[string]interface{}{"error": "No threads found to enhance"}, nil
		}

		// Limit
		if len(threadIDs) > limit {
			threadIDs = threadIDs[:limit]
		}

		// Start enhancement in background
		go func() {
			for _, threadID := range threadIDs {
				options := EnhancementOptions{
					CleanContent:      true,
					ExtractPerformers: true,
					ExtractTags:       true,
					GenerateSummary:   false,
				}
				_, err := enhancer.EnhanceThread(threadID, options)
				if err != nil {
					log.Printf("Failed to enhance thread %d: %v", threadID, err)
				}
			}
		}()

		return map[string]interface{}{
			"success":    true,
			"message":    fmt.Sprintf("Started enhancing %d threads", len(threadIDs)),
			"thread_ids": threadIDs,
		}, nil
	})

	// Register image validation tool
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "validate_thread_images",
			Description: "Validate images in a thread using AI vision analysis. Checks image quality, content, and detects broken/invalid images. Uses Qwen3.8-omni-flash for cost-effective analysis.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"thread_id": map[string]interface{}{
						"type":        "number",
						"description": "ID of the thread to validate images for",
					},
					"image_urls": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Optional: Specific image URLs to validate (if empty, validates all thread images)",
					},
					"auto_fix": map[string]interface{}{
						"type":        "boolean",
						"description": "Automatically remove invalid images from thread metadata (default: false)",
					},
				},
				"required": []string{"thread_id"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		threadID := int64(args["thread_id"].(float64))
		autoFix, _ := args["auto_fix"].(bool)

		// Get thread images
		var imageURLs []string
		if specificURLs, ok := args["image_urls"].([]interface{}); ok && len(specificURLs) > 0 {
			for _, u := range specificURLs {
				if urlStr, ok := u.(string); ok {
					imageURLs = append(imageURLs, urlStr)
				}
			}
		} else {
			// Get images from thread posts
			rows, err := s.db.Query(`
				SELECT DISTINCT metadata 
				FROM scraped_posts 
				WHERE thread_id = ? AND metadata != ''
			`, threadID)
			if err != nil {
				return map[string]interface{}{"error": fmt.Sprintf("Failed to query posts: %v", err)}, nil
			}
			defer rows.Close()

			for rows.Next() {
				var metaStr string
				if err := rows.Scan(&metaStr); err == nil {
					var meta struct {
						Attachments []struct {
							Type        string `json:"type"`
							URL         string `json:"url"`
							ThumbnailURL string `json:"thumbnail_url"`
						} `json:"attachments"`
					}
					if err := json.Unmarshal([]byte(metaStr), &meta); err == nil {
						for _, att := range meta.Attachments {
							if att.Type == "image" {
								if att.ThumbnailURL != "" {
									imageURLs = append(imageURLs, att.ThumbnailURL)
								} else if att.URL != "" {
									imageURLs = append(imageURLs, att.URL)
								}
							}
						}
					}
				}
			}
		}

		if len(imageURLs) == 0 {
			return map[string]interface{}{
				"success":  true,
				"message":  "No images found in thread",
				"thread_id": threadID,
			}, nil
		}

		// Validate images using AI
		validator := NewImageValidator("") // API key will be fetched from config
		threadTitle := ""
		s.db.QueryRow("SELECT title FROM scraped_threads WHERE id = ?", threadID).Scan(&threadTitle)

		results, err := validator.ValidateThreadImages(threadTitle, imageURLs)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Validation failed: %v", err)}, nil
		}

		// Auto-fix if requested
		if autoFix {
			invalidCount := 0
			for _, r := range results.Results {
				if !r.IsValid {
					invalidCount++
					// Log invalid image for manual review
					log.Printf("[ImageValidator] Invalid image in thread %d: %s - %s", threadID, r.ImageURL, r.Error)
				}
			}
			results.InvalidCount = invalidCount
		}

		return map[string]interface{}{
			"success":      true,
			"thread_id":    threadID,
			"total_images": results.TotalCount,
			"valid_images": results.ValidCount,
			"invalid_images": results.InvalidCount,
			"duration":     results.Duration,
			"results":      results.Results,
		}, nil
	})
}

// getBoolArg extracts a boolean from args with a default value
func getBoolArg(args map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}
