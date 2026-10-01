package services

import (
	"fmt"
	"log"
	"time"
)

// ── Phase 3 agency tools ──
//
// Tools that extend the Companion's hands to what Brix can do in the UI:
// background scraper jobs, thread rescrapes, library scans, and switching its
// own autonomy mode. All operations are additive/idempotent — destructive
// actions (delete/move/mutate) are intentionally NOT exposed here.

// registerAgencyTools registers the Phase 3 agency tools. Called from buildToolRegistry.
func (s *AICompanionService) registerAgencyTools() {
	// ── start_scraper_job ──
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "start_scraper_job",
			Description: "Start a background scraping job for a custom scraper tab. Use when the user wants a site scraped as a queued/background job instead of a one-off scrape_url call.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"tab_id": map[string]interface{}{"type": "number", "description": "ID of the custom scraper tab"},
				"url":    map[string]interface{}{"type": "string", "description": "URL to scrape"},
			}, "required": []string{"tab_id", "url"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		tabID := int64(intArg(args, "tab_id", 0))
		url, _ := args["url"].(string)
		if tabID <= 0 || url == "" {
			return map[string]interface{}{"error": "tab_id and url are required"}, nil
		}
		tabsSvc := NewScraperTabsService(s.db)
		tab, err := tabsSvc.GetScraperTabByID(tabID)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("tab not found: %v", err)}, nil
		}
		job := &ScraperJob{
			TabID:         &tabID,
			JobType:       "tab",
			Status:        "pending",
			URL:           url,
			ErrorMessages: []string{},
		}
		created, err := tabsSvc.CreateScraperJob(job)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("failed to create job: %v", err)}, nil
		}

		go func() {
			now := time.Now()
			created.Status = "running"
			created.StartedAt = &now
			tabsSvc.UpdateScraperJob(created.ID, created)

			vSvc := NewVideoScraperService(s.db)
			result, err := vSvc.ScrapePageWithProgress(ScrapeVideoRequest{
				URL:         url,
				SiteType:    tab.SiteType,
				ContentType: tab.ContentType,
				Options: ScrapeOptions{
					DownloadVideo:    false,
					ExtractMetadata:  true,
					ScrapeThumbnails: true,
					MaxPages:         0, // unlimited
				},
			}, func(p ScrapeProgress) {
				created.PagesCompleted = p.PagesCompleted
				created.VideosFound = p.VideosFound
				created.VideosSaved = p.VideosSaved
				created.ErrorsCount = len(p.Errors)
				if len(p.Errors) > 0 {
					created.ErrorMessages = p.Errors
				}
				tabsSvc.UpdateScraperJob(created.ID, created)
			})

			completed := time.Now()
			created.CompletedAt = &completed
			if err != nil {
				created.Status = "failed"
				created.ErrorMessages = append(created.ErrorMessages, err.Error())
				created.ErrorsCount++
				log.Printf("[Agency] Scraper job %d failed: %v", created.ID, err)
			} else {
				created.Status = "completed"
				created.TotalPages = result.PagesCrawled
				created.PagesCompleted = result.PagesCrawled
				created.VideosFound = result.TotalScraped
				created.VideosSaved = result.TotalScraped
				if len(result.Errors) > 0 {
					created.ErrorMessages = result.Errors
					created.ErrorsCount = len(result.Errors)
				}
			}
			tabsSvc.UpdateScraperJob(created.ID, created)
		}()

		return map[string]interface{}{
			"started": true, "job_id": created.ID, "tab": tab.Name, "url": url,
			"note": "Scraping runs in the background; check the Scraper page for progress.",
		}, nil
	})

	// ── rescrape_thread ──
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "rescrape_thread",
			Description: "Force-rescrape an existing forum thread (bypasses smart caching). Use when a thread's data is stale or images need re-fetching.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"thread_id": map[string]interface{}{"type": "number", "description": "ID of the scraped thread"},
			}, "required": []string{"thread_id"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		threadID := int64(intArg(args, "thread_id", 0))
		if threadID <= 0 {
			return map[string]interface{}{"error": "thread_id is required"}, nil
		}
		scraperSvc := NewScraperService(NewActivityService(), s)
		thread, err := scraperSvc.GetThreadByID(threadID)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("thread not found: %v", err)}, nil
		}
		go func() {
			if err := scraperSvc.ScrapeThreadCompleteForce(thread.URL, true); err != nil {
				log.Printf("[Agency] Rescrape of thread %d failed: %v", threadID, err)
			}
		}()
		return map[string]interface{}{
			"started": true, "thread_id": threadID, "url": thread.URL,
			"note": "Rescrape running in the background.",
		}, nil
	})

	// ── scan_library ──
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "scan_library",
			Description: "Trigger a video library scan (finds new files on disk and adds them to the DB). Use when the user adds files manually and wants them cataloged.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_id": map[string]interface{}{"type": "number", "description": "ID of the library to scan (use get_all_libraries to list them)"},
			}, "required": []string{"library_id"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryID := int64(intArg(args, "library_id", 0))
		if libraryID <= 0 {
			return map[string]interface{}{"error": "library_id is required"}, nil
		}
		videoSvc := NewVideoService(NewActivityService(), NewLibraryService(), NewPerformerService())
		go func() {
			if err := videoSvc.ScanLibrary(libraryID); err != nil {
				log.Printf("[Agency] Library scan %d failed: %v", libraryID, err)
			}
		}()
		return map[string]interface{}{
			"started": true, "library_id": libraryID,
			"note": "Library scan running in the background; check the Activity page for progress.",
		}, nil
	})

	// ── set_autonomy_mode ──
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "set_autonomy_mode",
			Description: "Change the Companion's autonomy mode. assist = commentary only, copilot = proposes actions for approval, autopilot = runs safe actions automatically. Brix must ask for this explicitly.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"mode": map[string]interface{}{"type": "string", "enum": []string{"assist", "copilot", "autopilot"}, "description": "The new autonomy mode"},
			}, "required": []string{"mode"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		mode, _ := args["mode"].(string)
		if !IsValidAutonomyMode(mode) {
			return map[string]interface{}{"error": "mode must be one of: assist, copilot, autopilot"}, nil
		}
		now := time.Now()
		_, err := s.db.Exec(
			`INSERT INTO ai_config (id, autonomy_mode, updated_at) VALUES (1, ?, ?)
			 ON CONFLICT(id) DO UPDATE SET autonomy_mode = excluded.autonomy_mode, updated_at = excluded.updated_at`,
			mode, now,
		)
		if err != nil {
			return nil, err
		}
		log.Printf("[Agency] Autonomy mode changed to %s (via tool)", mode)
		return map[string]interface{}{"saved": true, "autonomy_mode": mode}, nil
	})
}
