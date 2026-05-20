package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// generateDailyBriefing emits a proactive morning summary once per ~20 hours.
// It also saves a video-count snapshot so it can report deltas the next day.
func (s *AICompanionService) generateDailyBriefing() {
	// Check when we last sent a briefing
	var lastBriefingStr string
	s.db.QueryRow("SELECT value FROM memories WHERE key = 'last_daily_briefing'").Scan(&lastBriefingStr)
	if lastBriefingStr != "" {
		if t, err := time.Parse(time.RFC3339, lastBriefingStr); err == nil {
			if time.Since(t) < 20*time.Hour {
				log.Printf("[AI] Daily briefing skipped — last sent %s ago", formatDuration(time.Since(t)))
				return
			}
		}
	}

	// Current stats
	var videoCount, performerCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)

	// Delta since last snapshot
	var snapshotJSON string
	s.db.QueryRow("SELECT value FROM memories WHERE key = 'daily_snapshot'").Scan(&snapshotJSON)
	prevVideoCount := videoCount
	if snapshotJSON != "" {
		var snap map[string]int
		if json.Unmarshal([]byte(snapshotJSON), &snap) == nil {
			prevVideoCount = snap["video_count"]
		}
	}
	newVideos := videoCount - prevVideoCount

	// Videos sitting in Downloads library
	var downloadsCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM videos v JOIN libraries l ON v.library_id = l.id WHERE LOWER(l.name) LIKE '%download%'`).Scan(&downloadsCount)

	// Untagged videos
	var untaggedCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE id NOT IN (SELECT DISTINCT video_id FROM video_tags)`).Scan(&untaggedCount)

	// Recent errors (last 24h)
	var errorCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM console_logs WHERE level = 'error' AND created_at > datetime('now', '-24 hours')`).Scan(&errorCount)

	// Currently running tasks
	var runningTasks int
	s.db.QueryRow(`SELECT COUNT(*) FROM activity_logs WHERE status = 'running'`).Scan(&runningTasks)

	// Quick orphan sample — check first 200 videos for missing files (fast, SSD-friendly)
	orphanCount := 0
	sampleRows, _ := s.db.Query(`SELECT file_path FROM videos ORDER BY created_at DESC LIMIT 200`)
	if sampleRows != nil {
		for sampleRows.Next() {
			var fp string
			sampleRows.Scan(&fp)
			if _, statErr := os.Stat(fp); os.IsNotExist(statErr) {
				orphanCount++
			}
		}
		sampleRows.Close()
	}

	// Build the briefing message
	var msg strings.Builder
	msg.WriteString("## 🌅 Daily Briefing\n\n")

	// Library headline
	if newVideos > 0 {
		msg.WriteString(fmt.Sprintf("📈 **%d new video(s)** added since last briefing — library now at **%d total**\n\n", newVideos, videoCount))
	} else if newVideos < 0 {
		msg.WriteString(fmt.Sprintf("📚 Library: **%d videos** across **%d performers**\n\n", videoCount, performerCount))
	} else {
		msg.WriteString(fmt.Sprintf("📚 Library unchanged: **%d videos** across **%d performers**\n\n", videoCount, performerCount))
	}

	// Downloads waiting
	if downloadsCount > 0 {
		msg.WriteString(fmt.Sprintf("🔥 **%d video(s)** in Downloads waiting to be organized into Backup\n\n", downloadsCount))
	}

	// Untagged
	if untaggedCount > 0 {
		msg.WriteString(fmt.Sprintf("🏷️ **%d video(s)** have no tags — ask me to help tag them\n\n", untaggedCount))
	}

	// Errors
	if errorCount > 0 {
		msg.WriteString(fmt.Sprintf("⚠️ **%d error(s)** logged in the last 24 hours — ask me to check for errors\n\n", errorCount))
	} else {
		msg.WriteString("✅ No errors logged in the last 24 hours\n\n")
	}

	// Running tasks
	if runningTasks > 0 {
		msg.WriteString(fmt.Sprintf("⚙️ **%d task(s)** currently running in the background\n\n", runningTasks))
	}

	// Orphan warning (sampled check)
	if orphanCount > 0 {
		msg.WriteString(fmt.Sprintf("🔍 **%d video(s)** in the last 200 sampled appear to be missing from disk — ask me to run a full orphan scan\n\n", orphanCount))
	}

	msg.WriteString("_Ask me anything, or say **\"help me organize my downloads\"** to get started!_")

	// Persist snapshot + timestamp
	newSnap, _ := json.Marshal(map[string]int{"video_count": videoCount})
	s.db.Exec(`INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at)
		VALUES ('daily_snapshot', ?, 'system', 1, datetime('now'), datetime('now'))`, string(newSnap))
	s.db.Exec(`INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at)
		VALUES ('last_daily_briefing', ?, 'system', 1, datetime('now'), datetime('now'))`, time.Now().Format(time.RFC3339))

	log.Printf("[AI] Emitting daily briefing (newVideos=%d, downloads=%d, errors=%d, orphanSample=%d)", newVideos, downloadsCount, errorCount, orphanCount)

	s.emitEvent(CompanionEvent{
		Type:      "insight",
		Source:    "daily_briefing",
		Message:   msg.String(),
		Severity:  "info",
		Timestamp: time.Now(),
	})
}

// monitorLibraryHealth periodically checks library health
func (s *AICompanionService) monitorLibraryHealth() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkLibraryHealth()
		}
	}
}

// checkLibraryHealth performs health checks on libraries
func (s *AICompanionService) checkLibraryHealth() {
	// Get comprehensive library stats
	var videoCount, performerCount, tagCount, studioCount int
	var performersWithThumbnails, performersWithPreviews, performersWithoutThumbnails int
	var videosWithThumbnails, videosWithPreviews int

	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
	s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
	s.db.QueryRow("SELECT COUNT(*) FROM studios").Scan(&studioCount)

	// Check performer thumbnail coverage
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&performersWithThumbnails)
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&performersWithPreviews)
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != '' AND (thumbnail_path IS NULL OR thumbnail_path = '')").Scan(&performersWithoutThumbnails)

	// Check video thumbnail coverage
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&videosWithThumbnails)
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&videosWithPreviews)

	// === PERFORMER THUMBNAIL MONITORING ===
	// Alert if performers have previews but no thumbnails (performance issue)
	if performersWithoutThumbnails > 5 {
		message := fmt.Sprintf("⚡ Performance Opportunity: %d performers have preview videos but no thumbnails. Generate thumbnails for faster page loading!", performersWithoutThumbnails)

		// Track this recommendation
		s.trackRecommendation(
			"performance_optimization",
			"performer_thumbnail_generation",
			message,
			map[string]interface{}{
				"performers_without_thumbnails": performersWithoutThumbnails,
				"performers_with_previews":      performersWithPreviews,
			},
		)

		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: message,
			Data: map[string]interface{}{
				"performers_without_thumbnails": performersWithoutThumbnails,
				"performers_with_previews":      performersWithPreviews,
				"action":                        "generate_performer_thumbnails",
			},
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}

	// === TAGGING MONITORING ===
	if videoCount > 50 && tagCount < 5 {
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("📋 Organization Tip: You have %d videos but only %d tags. Consider using Smart Tagging to better organize your library.", videoCount, tagCount),
			Data: map[string]interface{}{
				"video_count": videoCount,
				"tag_count":   tagCount,
				"action":      "smart_tagging",
			},
			Severity:  "warning",
			Timestamp: time.Now(),
		})
	}

	// === PERFORMER LINKING MONITORING ===
	if videoCount > 100 && performerCount < 10 {
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("🔗 Metadata Alert: %d videos detected with only %d performers. Try Auto-Link Performers to improve organization.", videoCount, performerCount),
			Data: map[string]interface{}{
				"video_count":     videoCount,
				"performer_count": performerCount,
				"action":          "auto_link_performers",
			},
			Severity:  "warning",
			Timestamp: time.Now(),
		})
	}

	// === VIDEO PREVIEW MONITORING ===
	videosWithoutPreviews := videoCount - videosWithPreviews
	if videosWithoutPreviews > 20 && videoCount > 50 {
		coveragePercent := float64(videosWithPreviews) / float64(videoCount) * 100
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("🎬 Preview Coverage: Only %.1f%% of videos have previews. Generate previews for better browsing experience.", coveragePercent),
			Data: map[string]interface{}{
				"videos_without_previews": videosWithoutPreviews,
				"total_videos":           videoCount,
				"coverage_percent":       coveragePercent,
				"action":                 "generate_previews",
			},
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}

	// === METADATA COMPLETENESS ===
	var performersWithoutMetadata int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular' AND (metadata IS NULL OR metadata = '{}' OR metadata = '')").Scan(&performersWithoutMetadata)
	if performersWithoutMetadata > 10 {
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("📊 Metadata Incomplete: %d performers are missing metadata. Fetch from AdultDataLink for richer information.", performersWithoutMetadata),
			Data: map[string]interface{}{
				"performers_without_metadata": performersWithoutMetadata,
				"action":                      "fetch_metadata",
			},
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}

	// === LIBRARY HEALTH SUMMARY (Every 24 hours) ===
	uptime := time.Since(s.startTime)
	if uptime > 24*time.Hour && int(uptime.Hours())%24 == 0 {
		thumbnailCoverage := float64(0)
		if performersWithPreviews > 0 {
			thumbnailCoverage = float64(performersWithThumbnails) / float64(performersWithPreviews) * 100
		}

		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "health_monitor",
			Message: fmt.Sprintf("📈 Daily Library Report: %d videos, %d performers, %d tags, %d studios. Performer thumbnail coverage: %.1f%%", videoCount, performerCount, tagCount, studioCount, thumbnailCoverage),
			Data: map[string]interface{}{
				"video_count":         videoCount,
				"performer_count":     performerCount,
				"tag_count":           tagCount,
				"studio_count":        studioCount,
				"thumbnail_coverage":  thumbnailCoverage,
				"uptime_hours":        int(uptime.Hours()),
			},
			Severity:  "info",
			Timestamp: time.Now(),
		})
	}

	// suppress unused variable warnings
	_ = videosWithThumbnails
}

// performPeriodicAnalysis runs automated analysis tasks
func (s *AICompanionService) performPeriodicAnalysis() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			log.Println("🔍 AI Companion: Running periodic library analysis...")
			s.runLibraryInsights()
		}
	}
}

// runLibraryInsights checks for actionable library conditions and emits proactive insights.
// Each insight type has its own cooldown key in the memories table to avoid spam.
func (s *AICompanionService) runLibraryInsights() {
	s.checkInsight(
		"insight_unrated_recent",
		24*time.Hour,
		func() (bool, string, map[string]interface{}) {
			var count int
			s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE (rating IS NULL OR rating = 0) AND created_at > datetime('now', '-14 days')`).Scan(&count)
			if count < 6 {
				return false, "", nil
			}
			msg := fmt.Sprintf("⭐ **%d videos** added in the last 2 weeks haven't been rated yet. Rating helps me give you better recommendations — ask me to help rate them!", count)
			return true, msg, map[string]interface{}{"unrated_recent": count, "action": "rate_videos"}
		},
	)

	s.checkInsight(
		"insight_untagged_videos",
		48*time.Hour,
		func() (bool, string, map[string]interface{}) {
			var untagged, total int
			s.db.QueryRow(`SELECT COUNT(*) FROM videos v WHERE NOT EXISTS (SELECT 1 FROM video_tags vt WHERE vt.video_id = v.id)`).Scan(&untagged)
			s.db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&total)
			if untagged < 15 || total == 0 {
				return false, "", nil
			}
			pct := untagged * 100 / total
			msg := fmt.Sprintf("🏷️ **%d videos (%d%%)** in your library have no tags. Ask me to suggest tags — I can analyse performers to recommend the best ones!", untagged, pct)
			return true, msg, map[string]interface{}{"untagged": untagged, "total": total, "action": "suggest_tags"}
		},
	)

	s.checkInsight(
		"insight_unorganized_downloads",
		12*time.Hour,
		func() (bool, string, map[string]interface{}) {
			var count int
			s.db.QueryRow(`
				SELECT COUNT(*) FROM videos v
				JOIN libraries l ON l.id = v.library_id
				WHERE LOWER(l.name) LIKE '%download%'
				  AND NOT EXISTS (SELECT 1 FROM video_performers vp WHERE vp.video_id = v.id)
			`).Scan(&count)
			if count < 8 {
				return false, "", nil
			}
			msg := fmt.Sprintf("🔥 **%d unlinked videos** in your Downloads library have no performers attached. These are likely freshly downloaded files that haven't been processed yet. Want me to help organise them?", count)
			return true, msg, map[string]interface{}{"unlinked_downloads": count, "action": "organize_downloads"}
		},
	)

	s.checkInsight(
		"insight_rating_coverage",
		72*time.Hour,
		func() (bool, string, map[string]interface{}) {
			var rated, total int
			s.db.QueryRow(`SELECT COUNT(*) FROM videos WHERE rating IS NOT NULL AND rating > 0`).Scan(&rated)
			s.db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&total)
			if total < 20 {
				return false, "", nil
			}
			pct := rated * 100 / total
			if pct >= 30 {
				return false, "", nil
			}
			msg := fmt.Sprintf("📊 Only **%d%% of your library** (%d / %d videos) has been rated. The more you rate, the smarter my recommendations get. Ask me to pull up your unrated videos!", pct, rated, total)
			return true, msg, map[string]interface{}{"rated": rated, "total": total, "rating_pct": pct, "action": "rate_videos"}
		},
	)
}

// checkInsight runs a single insight check with cooldown guard.
// fn returns (shouldFire, message, data). If shouldFire is false nothing is emitted.
func (s *AICompanionService) checkInsight(cooldownKey string, cooldown time.Duration, fn func() (bool, string, map[string]interface{})) {
	var lastStr string
	s.db.QueryRow("SELECT value FROM memories WHERE key = ?", cooldownKey).Scan(&lastStr)
	if lastStr != "" {
		if t, err := time.Parse(time.RFC3339, lastStr); err == nil {
			if time.Since(t) < cooldown {
				return
			}
		}
	}

	shouldFire, msg, data := fn()
	if !shouldFire {
		return
	}

	s.db.Exec(`INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at)
		VALUES (?, ?, 'system', 1, datetime('now'), datetime('now'))`, cooldownKey, time.Now().Format(time.RFC3339))

	s.emitEvent(CompanionEvent{
		Type:      "insight",
		Source:    "library_insights",
		Message:   msg,
		Data:      data,
		Severity:  "info",
		Timestamp: time.Now(),
	})
	log.Printf("[AI] Library insight fired: %s", cooldownKey)
}
