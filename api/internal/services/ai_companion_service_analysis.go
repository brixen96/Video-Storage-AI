package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// RunPathAnalysisAndReport is called after a library scan to auto-link by path
// and emit a proactive companion event with the results.
func (s *AICompanionService) RunPathAnalysisAndReport(libraryID int64, addedCount int) {
	svc := NewPathAnalysisService()
	report, err := svc.RunAutoLink(libraryID, true, false)
	if err != nil {
		log.Printf("[PathAnalysis] Error: %v", err)
		return
	}
	if report.PerformersLinked == 0 && report.TagsApplied == 0 && report.StudiosLinked == 0 {
		return // nothing to report
	}
	parts := []string{}
	if report.PerformersLinked > 0 {
		parts = append(parts, fmt.Sprintf("**%d** performer links", report.PerformersLinked))
	}
	if report.TagsApplied > 0 {
		parts = append(parts, fmt.Sprintf("**%d** tags", report.TagsApplied))
	}
	if report.StudiosLinked > 0 {
		parts = append(parts, fmt.Sprintf("**%d** studio links", report.StudiosLinked))
	}
	msg := fmt.Sprintf("🗂️ Path analysis complete after scan (%d new videos): applied %s based on folder structure.",
		addedCount, strings.Join(parts, ", "))
	s.emitEvent(CompanionEvent{
		Type:      "insight",
		Source:    "path_analysis",
		Message:   msg,
		Severity:  "info",
		Timestamp: time.Now(),
	})
}

// RunVideoHealthCheckAndReport runs a full health check and emits a proactive companion event.
// Pass emitAlways=true for on-demand triggers; false skips the event when everything is already healthy.
func (s *AICompanionService) RunVideoHealthCheckAndReport(emitAlways bool) {
	svc := NewVideoHealthService()
	report, err := svc.Run(s.ctx)
	if err != nil {
		log.Printf("[AI HealthCheck] error: %v", err)
		return
	}
	msg := BuildHealthSummary(report)
	if !emitAlways && report.MetadataFixed == 0 && report.ThumbnailsFixed == 0 &&
		report.PerformersLinked == 0 && report.ContentTypeSet == 0 {
		return // nothing to report
	}
	s.emitEvent(CompanionEvent{
		Type:     "insight",
		Source:   "health_monitor",
		Message:  "🩺 " + msg,
		Data:     map[string]interface{}{"report": report},
		Severity: "info",
		Timestamp: time.Now(),
	})
}

// runPeriodicHealthCheck runs a full video health check every 6 hours, starting 90s after startup.
func (s *AICompanionService) runPeriodicHealthCheck() {
	select {
	case <-s.ctx.Done():
		return
	case <-time.After(90 * time.Second):
	}

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		// Cooldown guard — skip if ran in the last 4 hours (handles restart edge cases)
		var lastStr string
		s.db.QueryRow("SELECT value FROM memories WHERE key = 'last_health_check'").Scan(&lastStr)
		shouldRun := true
		if lastStr != "" {
			if t, err := time.Parse(time.RFC3339, lastStr); err == nil {
				if time.Since(t) < 4*time.Hour {
					shouldRun = false
				}
			}
		}
		if shouldRun {
			s.db.Exec(`INSERT OR REPLACE INTO memories (key, value, category, importance, created_at, updated_at)
				VALUES ('last_health_check', ?, 'system', 1, datetime('now'), datetime('now'))`,
				time.Now().Format(time.RFC3339))
			go s.RunVideoHealthCheckAndReport(false)
		}

		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// fetchSiteInfo fetches a page and returns structural info for scraper config suggestions
func (s *AICompanionService) fetchSiteInfo(page1URL, page2URL string) (interface{}, error) {
	client := &http.Client{Timeout: 12 * time.Second}

	// Detect pagination if page2 provided
	paginationInfo := map[string]interface{}{
		"template":    "",
		"starts_at":   1,
		"confidence":  "unknown",
		"notes":       "No second page URL provided",
	}
	if page2URL != "" {
		// Inline pagination detection (mirrors the handler logic)
		detected := detectPaginationForAI(page1URL, page2URL)
		paginationInfo = detected
	}

	// Fetch page1 HTML
	req, err := http.NewRequest("GET", page1URL, nil)
	if err != nil {
		return map[string]interface{}{
			"error":          fmt.Sprintf("Invalid URL: %v", err),
			"pagination":     paginationInfo,
		}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"error":          fmt.Sprintf("Failed to fetch page: %v", err),
			"pagination":     paginationInfo,
		}, nil
	}
	defer resp.Body.Close()

	// Read up to 80KB
	limitReader := io.LimitReader(resp.Body, 80*1024)
	bodyBytes, err := io.ReadAll(limitReader)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Failed to read page: %v", err)}, nil
	}
	html := string(bodyBytes)

	// Extract page title
	title := ""
	titleRe := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	if m := titleRe.FindStringSubmatch(html); len(m) > 1 {
		title = strings.TrimSpace(m[1])
	}

	// Strip all tags for content summary
	stripTags := regexp.MustCompile(`<[^>]+>`)
	text := stripTags.ReplaceAllString(html, " ")
	// Collapse whitespace
	wsRe := regexp.MustCompile(`\s+`)
	text = strings.TrimSpace(wsRe.ReplaceAllString(text, " "))
	if len(text) > 2000 {
		text = text[:2000]
	}

	// Count structural elements
	linkCount := strings.Count(strings.ToLower(html), "<a ")
	imgCount := strings.Count(strings.ToLower(html), "<img ")
	videoCount := strings.Count(strings.ToLower(html), "<video") + strings.Count(strings.ToLower(html), "iframe")

	return map[string]interface{}{
		"url":               page1URL,
		"page_title":        title,
		"status_code":       resp.StatusCode,
		"link_count":        linkCount,
		"image_count":       imgCount,
		"video_embed_count": videoCount,
		"content_preview":   text,
		"pagination":        paginationInfo,
	}, nil
}

// detectPaginationForAI is a lightweight version of analyzePaginationURLs for use in the AI service
func detectPaginationForAI(url1, url2 string) map[string]interface{} {
	// Compare the two URLs to find a numeric difference
	// Simple approach: find the token that changed
	if len(url1) == 0 || len(url2) == 0 {
		return map[string]interface{}{"template": "", "confidence": "low", "notes": "Missing URLs"}
	}

	// Find common prefix
	prefix := ""
	for i := 0; i < len(url1) && i < len(url2); i++ {
		if url1[i] == url2[i] {
			prefix += string(url1[i])
		} else {
			break
		}
	}

	// Find common suffix (reversed)
	suffix := ""
	for i := 0; i < len(url1) && i < len(url2); i++ {
		if url1[len(url1)-1-i] == url2[len(url2)-1-i] {
			suffix = string(url1[len(url1)-1-i]) + suffix
		} else {
			break
		}
	}

	middle1 := url1[len(prefix) : len(url1)-len(suffix)]
	middle2 := url2[len(prefix) : len(url2)-len(suffix)]

	n1, e1 := strconv.Atoi(middle1)
	n2, e2 := strconv.Atoi(middle2)
	if e1 == nil && e2 == nil && n2 == n1+1 {
		template := prefix + "{page}" + suffix
		return map[string]interface{}{
			"template":   template,
			"starts_at":  n1,
			"confidence": "high",
			"notes":      fmt.Sprintf("Page number changes from %d to %d between the two URLs", n1, n2),
		}
	}

	return map[string]interface{}{
		"template":   "",
		"starts_at":  1,
		"confidence": "low",
		"notes":      "Could not auto-detect pagination pattern",
	}
}

// ================== Advanced Query Functions ==================

// SearchVideos searches for videos by various criteria
func (s *AICompanionService) SearchVideos(query string) ([]string, error) {
	searchQuery := `
		SELECT v.title, v.file_path, p.name as performer_name
		FROM videos v
		LEFT JOIN video_performers vp ON v.id = vp.video_id
		LEFT JOIN performers p ON vp.performer_id = p.id
		WHERE v.title LIKE ? OR p.name LIKE ?
		GROUP BY v.id
		LIMIT 10
	`

	rows, err := s.db.Query(searchQuery, "%"+query+"%", "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var title, filePath string
		var performerName sql.NullString
		if err := rows.Scan(&title, &filePath, &performerName); err != nil {
			continue
		}

		result := fmt.Sprintf("📹 %s", title)
		if performerName.Valid && performerName.String != "" {
			result += fmt.Sprintf(" (featuring %s)", performerName.String)
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return []string{fmt.Sprintf("No videos found matching '%s'", query)}, nil
	}

	return results, nil
}

// FindPerformer searches for a performer and returns detailed info
func (s *AICompanionService) FindPerformer(name string) (string, error) {
	query := `
		SELECT p.id, p.name, p.category, COUNT(vp.video_id) as video_count,
		       p.thumbnail_path, p.preview_path
		FROM performers p
		LEFT JOIN video_performers vp ON p.id = vp.performer_id
		WHERE p.name LIKE ?
		GROUP BY p.id
		LIMIT 1
	`

	var id int64
	var performerName, category string
	var videoCount int
	var thumbnailPath, previewPath sql.NullString

	err := s.db.QueryRow(query, "%"+name+"%").Scan(&id, &performerName, &category, &videoCount, &thumbnailPath, &previewPath)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("I couldn't find a performer matching '%s'. Try checking the Performers page or scanning for new performers.", name), nil
	}
	if err != nil {
		return "", err
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("👤 Performer: %s\n", performerName))
	response.WriteString(fmt.Sprintf("Category: %s\n", category))
	response.WriteString(fmt.Sprintf("Videos: %d\n", videoCount))

	hasThumbnail := thumbnailPath.Valid && thumbnailPath.String != ""
	hasPreview := previewPath.Valid && previewPath.String != ""

	response.WriteString(fmt.Sprintf("Thumbnail: %s\n", map[bool]string{true: "✅", false: "❌"}[hasThumbnail]))
	response.WriteString(fmt.Sprintf("Preview: %s\n", map[bool]string{true: "✅", false: "❌"}[hasPreview]))

	return response.String(), nil
}

// GetTaggedVideos finds videos with specific tags
func (s *AICompanionService) GetTaggedVideos(tagName string) ([]string, error) {
	query := `
		SELECT v.title, t.name as tag_name
		FROM videos v
		INNER JOIN video_tags vt ON v.id = vt.video_id
		INNER JOIN tags t ON vt.tag_id = t.id
		WHERE t.name LIKE ?
		LIMIT 10
	`

	rows, err := s.db.Query(query, "%"+tagName+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var title, tag string
		if err := rows.Scan(&title, &tag); err != nil {
			continue
		}
		results = append(results, fmt.Sprintf("🏷️ %s [%s]", title, tag))
	}

	if len(results) == 0 {
		return []string{fmt.Sprintf("No videos found with tag '%s'", tagName)}, nil
	}

	return results, nil
}

// CountPostsWithKeyword counts posts containing a specific keyword
func (s *AICompanionService) CountPostsWithKeyword(keyword string, threadID ...int64) (int, error) {
	var query string
	var args []interface{}
	var count int

	if len(threadID) > 0 && threadID[0] > 0 {
		query = "SELECT COUNT(*) FROM scraped_posts WHERE thread_id = ? AND plain_text LIKE ?"
		args = []interface{}{threadID[0], "%" + keyword + "%"}
	} else {
		query = "SELECT COUNT(*) FROM scraped_posts WHERE plain_text LIKE ?"
		args = []interface{}{"%" + keyword + "%"}
	}

	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// SearchThreadPosts searches for text in scraped forum posts
func (s *AICompanionService) SearchThreadPosts(searchTerm string, threadID ...int64) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}

	if len(threadID) > 0 && threadID[0] > 0 {
		// Search within specific thread
		query = `
			SELECT sp.id, sp.plain_text, sp.author, sp.post_number, st.title, st.id as thread_id
			FROM scraped_posts sp
			INNER JOIN scraped_threads st ON sp.thread_id = st.id
			WHERE sp.thread_id = ? AND sp.plain_text LIKE ?
			ORDER BY sp.post_number ASC
			LIMIT 50
		`
		args = []interface{}{threadID[0], "%" + searchTerm + "%"}
	} else {
		// Search across all posts
		query = `
			SELECT sp.id, sp.plain_text, sp.author, sp.post_number, st.title, st.id as thread_id
			FROM scraped_posts sp
			INNER JOIN scraped_threads st ON sp.thread_id = st.id
			WHERE sp.plain_text LIKE ?
			ORDER BY st.id, sp.post_number ASC
			LIMIT 50
		`
		args = []interface{}{"%" + searchTerm + "%"}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var postID, threadIDVal int64
		var plainText, author, threadTitle string
		var postNumber int

		err = rows.Scan(&postID, &plainText, &author, &postNumber, &threadTitle, &threadIDVal)
		if err != nil {
			continue
		}

		// Truncate text to 300 chars for display
		excerpt := plainText
		if len(excerpt) > 300 {
			excerpt = excerpt[:300] + "..."
		}

		results = append(results, map[string]interface{}{
			"post_id":      postID,
			"thread_id":    threadIDVal,
			"thread_title": threadTitle,
			"post_number":  postNumber,
			"author":       author,
			"excerpt":      excerpt,
			"full_text":    plainText,
		})
	}

	return results, nil
}

// GetThreadByName finds a thread by partial name match
func (s *AICompanionService) GetThreadByName(threadName string) (int64, string, error) {
	var threadID int64
	var title string

	query := `
		SELECT id, title
		FROM scraped_threads
		WHERE title LIKE ?
		LIMIT 1
	`

	err := s.db.QueryRow(query, "%"+threadName+"%").Scan(&threadID, &title)
	if err != nil {
		return 0, "", err
	}

	return threadID, title, nil
}

// GetRecentVideos returns recently added videos
func (s *AICompanionService) GetRecentVideos(limit int) ([]string, error) {
	if limit <= 0 || limit > 20 {
		limit = 10
	}

	query := `
		SELECT title, created_at
		FROM videos
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var title string
		var createdAt time.Time
		if err := rows.Scan(&title, &createdAt); err != nil {
			continue
		}

		timeAgo := formatDuration(time.Since(createdAt))
		results = append(results, fmt.Sprintf("📹 %s (%s ago)", title, timeAgo))
	}

	return results, nil
}

// AnalyzeStorageUsage provides insights into storage usage
func (s *AICompanionService) AnalyzeStorageUsage() (string, error) {
	var totalSize, videoCount int64
	var avgSize float64

	s.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(file_size), 0), COALESCE(AVG(file_size), 0) FROM videos").Scan(&videoCount, &totalSize, &avgSize)

	// Get size by library
	libraryQuery := `
		SELECT l.name, COUNT(v.id) as count, COALESCE(SUM(v.file_size), 0) as size
		FROM libraries l
		LEFT JOIN videos v ON l.id = v.library_id
		GROUP BY l.id
		ORDER BY size DESC
		LIMIT 5
	`

	rows, err := s.db.Query(libraryQuery)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var response strings.Builder
	response.WriteString("💾 Storage Analysis:\n\n")
	response.WriteString(fmt.Sprintf("Total Videos: %d\n", videoCount))
	response.WriteString(fmt.Sprintf("Total Size: %.2f GB\n", float64(totalSize)/(1024*1024*1024)))
	response.WriteString(fmt.Sprintf("Average Video Size: %.2f MB\n\n", avgSize/(1024*1024)))

	response.WriteString("Top Libraries by Size:\n")
	for rows.Next() {
		var name string
		var count, size int64
		if err := rows.Scan(&name, &count, &size); err != nil {
			continue
		}
		response.WriteString(fmt.Sprintf("- %s: %d videos, %.2f GB\n", name, count, float64(size)/(1024*1024*1024)))
	}

	return response.String(), nil
}

// ================== Predictive Analytics & Insights ==================

// PredictLibraryGrowth analyzes library growth trends and predicts future growth
func (s *AICompanionService) PredictLibraryGrowth() (string, error) {
	// Get video creation timestamps
	query := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM videos
		WHERE created_at >= DATE('now', '-30 days')
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	dailyCounts := []int{}
	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			continue
		}
		dailyCounts = append(dailyCounts, count)
	}

	if len(dailyCounts) == 0 {
		return "📊 Not enough data to predict library growth. Add more videos over time to see trends!", nil
	}

	// Calculate average daily growth
	total := 0
	for _, count := range dailyCounts {
		total += count
	}
	avgPerDay := float64(total) / float64(len(dailyCounts))

	// Get total videos
	var totalVideos int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&totalVideos)

	// Predict next 30 days
	predicted30Days := int(avgPerDay * 30)
	predicted90Days := int(avgPerDay * 90)

	var response strings.Builder
	response.WriteString("📈 Library Growth Prediction:\n\n")
	response.WriteString(fmt.Sprintf("Current Library Size: %d videos\n", totalVideos))
	response.WriteString(fmt.Sprintf("Average Daily Growth: %.1f videos/day\n\n", avgPerDay))
	response.WriteString("Predictions:\n")
	response.WriteString(fmt.Sprintf("• In 30 days: ~%d videos (+%d)\n", totalVideos+predicted30Days, predicted30Days))
	response.WriteString(fmt.Sprintf("• In 90 days: ~%d videos (+%d)\n", totalVideos+predicted90Days, predicted90Days))

	if avgPerDay > 10 {
		response.WriteString("\n🚀 Your library is growing rapidly! Consider scheduling regular maintenance tasks.")
	} else if avgPerDay > 5 {
		response.WriteString("\n📊 Steady growth detected. Your library is expanding at a healthy pace.")
	} else if avgPerDay > 0 {
		response.WriteString("\n🌱 Slow but steady growth. Perfect for manageable library maintenance.")
	}

	return response.String(), nil
}

// AnalyzePerformerQuality provides quality scoring for performers based on metadata completeness
func (s *AICompanionService) AnalyzePerformerQuality() (string, error) {
	query := `
		SELECT
			p.name,
			CASE WHEN p.thumbnail_path IS NOT NULL AND p.thumbnail_path != '' THEN 1 ELSE 0 END as has_thumb,
			CASE WHEN p.preview_path IS NOT NULL AND p.preview_path != '' THEN 1 ELSE 0 END as has_preview,
			CASE WHEN p.metadata IS NOT NULL AND p.metadata != '{}' AND p.metadata != '' THEN 1 ELSE 0 END as has_metadata,
			COUNT(vp.video_id) as video_count
		FROM performers p
		LEFT JOIN video_performers vp ON p.id = vp.performer_id
		WHERE p.category = 'regular'
		GROUP BY p.id
		HAVING video_count > 0
		ORDER BY video_count DESC
		LIMIT 10
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type PerformerScore struct {
		Name       string
		Score      int
		VideoCount int
	}

	performers := []PerformerScore{}
	for rows.Next() {
		var name string
		var hasThumb, hasPreview, hasMetadata, videoCount int
		if err := rows.Scan(&name, &hasThumb, &hasPreview, &hasMetadata, &videoCount); err != nil {
			continue
		}

		// Calculate quality score (out of 100)
		score := 0
		score += hasThumb * 30      // 30 points for thumbnail
		score += hasPreview * 30    // 30 points for preview
		score += hasMetadata * 40   // 40 points for metadata

		performers = append(performers, PerformerScore{
			Name:       name,
			Score:      score,
			VideoCount: videoCount,
		})
	}

	if len(performers) == 0 {
		return "No performer data available for quality analysis.", nil
	}

	var response strings.Builder
	response.WriteString("⭐ Top Performers - Quality Score:\n\n")

	for i, p := range performers {
		emoji := "❌"
		if p.Score >= 90 {
			emoji = "🌟"
		} else if p.Score >= 70 {
			emoji = "✅"
		} else if p.Score >= 50 {
			emoji = "⚠️"
		}

		response.WriteString(fmt.Sprintf("%d. %s %s - %d/100 (%d videos)\n", i+1, emoji, p.Name, p.Score, p.VideoCount))
	}

	return response.String(), nil
}

// GenerateInsights provides AI-powered insights about library usage and patterns
func (s *AICompanionService) GenerateInsights() ([]string, error) {
	insights := []string{}

	// Insight 1: Collection focus
	var topPerformerCount int
	s.db.QueryRow(`
		SELECT COUNT(vp.video_id)
		FROM video_performers vp
		INNER JOIN performers p ON vp.performer_id = p.id
		GROUP BY p.id
		ORDER BY COUNT(vp.video_id) DESC
		LIMIT 1
	`).Scan(&topPerformerCount)

	if topPerformerCount > 20 {
		insights = append(insights, fmt.Sprintf("🎯 Collection Focus: You have a strong collection focus with %d+ videos of your top performer!", topPerformerCount))
	}

	// Insight 2: Organization level
	var totalVideos, taggedVideos int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&totalVideos)
	s.db.QueryRow("SELECT COUNT(DISTINCT video_id) FROM video_tags").Scan(&taggedVideos)

	if totalVideos > 0 {
		orgLevel := float64(taggedVideos) / float64(totalVideos) * 100
		if orgLevel > 70 {
			insights = append(insights, fmt.Sprintf("🏆 Excellent Organization: %.0f%% of your videos are tagged!", orgLevel))
		} else if orgLevel < 20 && totalVideos > 50 {
			insights = append(insights, "📋 Organization Opportunity: Most videos are untagged. Try Smart Tagging for better discovery!")
		}
	}

	// Insight 3: Recent activity
	var recentScans int
	s.db.QueryRow(`
		SELECT COUNT(*)
		FROM activity_logs
		WHERE task_type LIKE '%scan%'
		AND created_at >= DATETIME('now', '-7 days')
	`).Scan(&recentScans)

	if recentScans > 5 {
		insights = append(insights, "🔄 Active Management: You're frequently scanning your library. Great job staying organized!")
	}

	// Insight 4: Storage efficiency
	var avgSize, totalSize int64
	s.db.QueryRow("SELECT COALESCE(AVG(file_size), 0), COALESCE(SUM(file_size), 0) FROM videos").Scan(&avgSize, &totalSize)

	if avgSize > 0 {
		avgGB := float64(avgSize) / (1024 * 1024 * 1024)
		if avgGB > 2 {
			insights = append(insights, fmt.Sprintf("💾 High Quality: Average video size is %.2f GB - you're collecting high-quality content!", avgGB))
		}
	}

	// Insight 5: Metadata richness
	var performersWithMetadata, totalPerformers int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular'").Scan(&totalPerformers)
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular' AND metadata IS NOT NULL AND metadata != '{}' AND metadata != ''").Scan(&performersWithMetadata)

	if totalPerformers > 0 {
		metadataPercent := float64(performersWithMetadata) / float64(totalPerformers) * 100
		if metadataPercent > 80 {
			insights = append(insights, "📚 Rich Metadata: Your library has excellent metadata coverage!")
		}
	}

	if len(insights) == 0 {
		insights = append(insights, "🌱 Growing Library: Keep adding content and organizing your library to unlock more insights!")
	}

	return insights, nil
}

// ================== Duplicate Detection & Content Analysis ==================

// DetectPotentialDuplicates finds videos that might be duplicates based on various criteria
func (s *AICompanionService) DetectPotentialDuplicates() ([]string, error) {
	duplicates := []string{}

	// Find videos with very similar titles
	query := `
		SELECT v1.title, v2.title, v1.file_size, v2.file_size
		FROM videos v1
		INNER JOIN videos v2 ON v1.id < v2.id
		WHERE (
			-- Similar titles (case insensitive)
			LOWER(v1.title) = LOWER(v2.title)
			OR
			-- Same file size
			v1.file_size = v2.file_size
		)
		LIMIT 20
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var title1, title2 string
		var size1, size2 int64
		if err := rows.Scan(&title1, &title2, &size1, &size2); err != nil {
			continue
		}

		if title1 == title2 {
			duplicates = append(duplicates, fmt.Sprintf("📹 Duplicate Title: '%s' (%.2f MB)", title1, float64(size1)/(1024*1024)))
		} else if size1 == size2 {
			duplicates = append(duplicates, fmt.Sprintf("🔍 Same Size: '%s' vs '%s' (%.2f MB)", title1, title2, float64(size1)/(1024*1024)))
		}
	}

	if len(duplicates) == 0 {
		return []string{"✅ No obvious duplicates detected!"}, nil
	}

	return duplicates, nil
}

// AnalyzeLibraryHealth provides a comprehensive health score
func (s *AICompanionService) AnalyzeLibraryHealth() (string, error) {
	var response strings.Builder
	response.WriteString("🥼 Library Health Report:\n\n")

	score := 100
	issues := []string{}

	// Check video coverage
	var totalVideos, videosWithThumbs, videosWithPreviews int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&totalVideos)
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&videosWithThumbs)
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&videosWithPreviews)

	thumbCoverage := 0.0
	if totalVideos > 0 {
		thumbCoverage = float64(videosWithThumbs) / float64(totalVideos) * 100
	}

	if thumbCoverage < 50 {
		score -= 20
		issues = append(issues, fmt.Sprintf("❌ Low thumbnail coverage: %.0f%%", thumbCoverage))
	} else if thumbCoverage < 80 {
		score -= 10
		issues = append(issues, fmt.Sprintf("⚠️ Medium thumbnail coverage: %.0f%%", thumbCoverage))
	} else {
		issues = append(issues, fmt.Sprintf("✅ Good thumbnail coverage: %.0f%%", thumbCoverage))
	}

	// Check performer metadata
	var totalPerformers, performersWithMetadata int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular'").Scan(&totalPerformers)
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular' AND metadata IS NOT NULL AND metadata != '{}' AND metadata != ''").Scan(&performersWithMetadata)

	metadataCoverage := 0.0
	if totalPerformers > 0 {
		metadataCoverage = float64(performersWithMetadata) / float64(totalPerformers) * 100
	}

	if metadataCoverage < 30 {
		score -= 15
		issues = append(issues, fmt.Sprintf("❌ Low metadata coverage: %.0f%%", metadataCoverage))
	} else if metadataCoverage < 70 {
		score -= 5
		issues = append(issues, fmt.Sprintf("⚠️ Medium metadata coverage: %.0f%%", metadataCoverage))
	} else {
		issues = append(issues, fmt.Sprintf("✅ Good metadata coverage: %.0f%%", metadataCoverage))
	}

	// Check for recent errors
	consoleLogSvc := NewConsoleLogService()
	errors, _, _ := consoleLogSvc.GetAll(10, 0, "", "error", "")
	if len(errors) > 5 {
		score -= 15
		issues = append(issues, fmt.Sprintf("❌ %d recent errors detected", len(errors)))
	} else if len(errors) > 0 {
		score -= 5
		issues = append(issues, fmt.Sprintf("⚠️ %d minor errors", len(errors)))
	} else {
		issues = append(issues, "✅ No recent errors")
	}

	// Check organization (tags)
	var videoCount, taggedVideoCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(DISTINCT video_id) FROM video_tags").Scan(&taggedVideoCount)

	tagCoverage := 0.0
	if videoCount > 0 {
		tagCoverage = float64(taggedVideoCount) / float64(videoCount) * 100
	}

	if tagCoverage < 20 {
		score -= 10
		issues = append(issues, fmt.Sprintf("❌ Poor organization: %.0f%% videos tagged", tagCoverage))
	} else if tagCoverage < 50 {
		score -= 5
		issues = append(issues, fmt.Sprintf("⚠️ Moderate organization: %.0f%% videos tagged", tagCoverage))
	} else {
		issues = append(issues, fmt.Sprintf("✅ Well organized: %.0f%% videos tagged", tagCoverage))
	}

	// Determine health rating
	rating := "Excellent"
	emoji := "🌟"
	if score < 60 {
		rating = "Poor"
		emoji = "⚠️"
	} else if score < 75 {
		rating = "Fair"
		emoji = "📊"
	} else if score < 90 {
		rating = "Good"
		emoji = "👍"
	}

	response.WriteString(fmt.Sprintf("Overall Health: %s %d/100 %s\n\n", emoji, score, rating))
	response.WriteString("Details:\n")
	for _, issue := range issues {
		response.WriteString(fmt.Sprintf("• %s\n", issue))
	}

	if score < 90 {
		response.WriteString("\n💡 Run 'optimization suggestions' for improvement tips!")
	}

	// suppress unused variable warning
	_ = videosWithPreviews

	return response.String(), nil
}

// RecommendContent suggests content based on library analysis
func (s *AICompanionService) RecommendContent() ([]string, error) {
	recommendations := []string{}

	// Find top performers
	query := `
		SELECT p.name, COUNT(vp.video_id) as video_count
		FROM performers p
		INNER JOIN video_performers vp ON p.id = vp.performer_id
		GROUP BY p.id
		ORDER BY video_count DESC
		LIMIT 3
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topPerformers := []string{}
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}
		topPerformers = append(topPerformers, name)
	}

	if len(topPerformers) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("⭐ You seem to enjoy content featuring: %s", strings.Join(topPerformers, ", ")))
		recommendations = append(recommendations, fmt.Sprintf("💡 Tip: Search for more videos with these performers!"))
	}

	// Find popular tags
	tagQuery := `
		SELECT t.name, COUNT(vt.video_id) as usage_count
		FROM tags t
		INNER JOIN video_tags vt ON t.id = vt.tag_id
		GROUP BY t.id
		ORDER BY usage_count DESC
		LIMIT 3
	`

	tagRows, err := s.db.Query(tagQuery)
	if err != nil {
		return recommendations, nil
	}
	defer tagRows.Close()

	topTags := []string{}
	for tagRows.Next() {
		var name string
		var count int
		if err := tagRows.Scan(&name, &count); err != nil {
			continue
		}
		topTags = append(topTags, name)
	}

	if len(topTags) > 0 {
		recommendations = append(recommendations, fmt.Sprintf("🏷️ Your most used tags: %s", strings.Join(topTags, ", ")))
	}

	if len(recommendations) == 0 {
		return []string{"📚 Start organizing your library with tags and performers to get personalized recommendations!"}, nil
	}

	return recommendations, nil
}

// ================== Learning & Pattern Recognition ==================

// LearnFromUserAction records user actions to learn patterns
func (s *AICompanionService) LearnFromUserAction(actionType, context string, details map[string]interface{}) error {
	// Store as a memory for pattern recognition
	memory := &models.Memory{
		Key:        fmt.Sprintf("action_%s_%d", actionType, time.Now().Unix()),
		Value:      context,
		Category:   "user_action",
		Importance: 1, // Actions are low importance unless they reveal patterns
	}

	// Add details as JSON if provided
	if len(details) > 0 {
		detailsJSON, _ := json.Marshal(details)
		memory.Value = fmt.Sprintf("%s|%s", context, string(detailsJSON))
	}

	return s.SaveMemory(memory)
}

// DetectUsagePatterns analyzes user behavior and suggests workflows
func (s *AICompanionService) DetectUsagePatterns() ([]string, error) {
	patterns := []string{}

	// Check library scan frequency
	var lastScanTime time.Time
	err := s.db.QueryRow(`
		SELECT MAX(completed_at)
		FROM activity_logs
		WHERE task_type = 'video_scan' AND status = 'completed'
	`).Scan(&lastScanTime)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if !lastScanTime.IsZero() {
		daysSinceLastScan := int(time.Since(lastScanTime).Hours() / 24)
		if daysSinceLastScan > 7 {
			patterns = append(patterns, fmt.Sprintf("📅 Pattern: It's been %d days since your last library scan. Consider scanning to find new content!", daysSinceLastScan))
		}
	}

	// Check if user frequently views certain categories
	var favoriteTagRows int
	s.db.QueryRow(`
		SELECT COUNT(DISTINCT tag_id)
		FROM video_tags
		GROUP BY tag_id
		HAVING COUNT(*) > 10
		LIMIT 1
	`).Scan(&favoriteTagRows)

	if favoriteTagRows > 0 {
		patterns = append(patterns, "🏷️ Pattern: You have well-organized tags! Consider using Smart Tagging to automatically categorize new videos.")
	}

	// Check performer engagement
	var performersViewed int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM performers WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''
	`).Scan(&performersViewed)

	if performersViewed < 5 {
		patterns = append(patterns, "👤 Pattern: Limited performer thumbnails detected. Generate thumbnails for faster browsing!")
	}

	return patterns, nil
}

// SuggestNextAction provides contextual suggestions based on recent activity
func (s *AICompanionService) SuggestNextAction() (string, error) {
	// Check most recent completed task
	var recentTask string
	var recentTime time.Time

	err := s.db.QueryRow(`
		SELECT task_type, completed_at
		FROM activity_logs
		WHERE status = 'completed'
		ORDER BY completed_at DESC
		LIMIT 1
	`).Scan(&recentTask, &recentTime)

	if err == sql.ErrNoRows {
		return "💡 Start by scanning a library to index your videos!", nil
	}
	if err != nil {
		return "", err
	}

	// Suggest logical next steps based on recent task
	switch recentTask {
	case "video_scan", "library_scan":
		// Check if thumbnails need generation
		var videosWithoutThumbnails int
		s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE thumbnail_path IS NULL OR thumbnail_path = ''").Scan(&videosWithoutThumbnails)

		if videosWithoutThumbnails > 20 {
			return fmt.Sprintf("💡 Next Step: You just scanned videos! Consider generating thumbnails for %d videos to improve browsing.", videosWithoutThumbnails), nil
		}
		return "✅ Library scan complete! You might want to check the Videos page to browse your content.", nil

	case "performer_thumbnail_generation":
		return "✅ Performer thumbnails generated! Visit the Performers page to see the improved loading speed.", nil

	case "metadata_fetch":
		return "✅ Metadata fetched! Your content is now enriched with detailed information.", nil

	default:
		return "💡 Everything looks good! Check 'optimization suggestions' for improvement ideas.", nil
	}
}

// ScheduleAutomatedTasks suggests optimal times for maintenance tasks
func (s *AICompanionService) ScheduleAutomatedTasks() ([]string, error) {
	suggestions := []string{}

	// Suggest library scan if not done recently
	var lastScan sql.NullTime
	s.db.QueryRow(`
		SELECT MAX(completed_at)
		FROM activity_logs
		WHERE task_type LIKE '%scan%' AND status = 'completed'
	`).Scan(&lastScan)

	if !lastScan.Valid || time.Since(lastScan.Time) > 7*24*time.Hour {
		suggestions = append(suggestions, "📅 Weekly Task: Schedule a library scan to index new content")
	}

	// Suggest thumbnail maintenance
	var thumbailCoverage float64
	s.db.QueryRow(`
		SELECT CAST(COUNT(CASE WHEN thumbnail_path IS NOT NULL AND thumbnail_path != '' THEN 1 END) AS REAL) / COUNT(*)
		FROM videos
	`).Scan(&thumbailCoverage)

	if thumbailCoverage < 0.8 {
		suggestions = append(suggestions, fmt.Sprintf("🖼️ Maintenance Task: Only %.0f%% of videos have thumbnails. Run thumbnail generation.", thumbailCoverage*100))
	}

	// Suggest metadata cleanup
	var performersNeedingMetadata int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM performers
		WHERE category = 'regular' AND (metadata IS NULL OR metadata = '{}')
	`).Scan(&performersNeedingMetadata)

	if performersNeedingMetadata > 20 {
		suggestions = append(suggestions, fmt.Sprintf("📊 Maintenance Task: %d performers missing metadata. Fetch from AdultDataLink.", performersNeedingMetadata))
	}

	if len(suggestions) == 0 {
		return []string{"✅ No maintenance tasks needed! Your library is well maintained."}, nil
	}

	return suggestions, nil
}

// ================== Task Execution Functions ==================

// ExecuteLibraryScan triggers a library scan for a specific library
func (s *AICompanionService) ExecuteLibraryScan(libraryID int64) error {
	consoleLogSvc := NewConsoleLogService()
	consoleLogSvc.LogAICompanion("info", "AI Companion initiated library scan", map[string]interface{}{
		"library_id": libraryID,
		"trigger":    "ai_companion",
	})

	videoSvc := NewVideoService(NewActivityService(), NewLibraryService(), NewPerformerService())
	go func() {
		if err := videoSvc.ScanLibrary(libraryID); err != nil {
			consoleLogSvc.LogAICompanion("error", "AI Companion library scan failed", map[string]interface{}{
				"library_id": libraryID,
				"error":      err.Error(),
			})
		}
	}()

	return nil
}

// ExecutePerformerScan triggers a performer scan
func (s *AICompanionService) ExecutePerformerScan() error {
	consoleLogSvc := NewConsoleLogService()
	consoleLogSvc.LogAICompanion("info", "AI Companion initiated performer scan", map[string]interface{}{
		"trigger": "ai_companion",
	})

	performerScanSvc := NewPerformerScanService()
	go func() {
		if _, err := performerScanSvc.ScanPerformerFolders(); err != nil {
			consoleLogSvc.LogAICompanion("error", "AI Companion performer scan failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	return nil
}

// SuggestOptimizations analyzes the system and suggests optimizations
func (s *AICompanionService) SuggestOptimizations() ([]string, error) {
	suggestions := []string{}

	// Check for performers without thumbnails
	var performersWithoutThumbnails int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != '' AND (thumbnail_path IS NULL OR thumbnail_path = '')").Scan(&performersWithoutThumbnails)
	if performersWithoutThumbnails > 5 {
		suggestions = append(suggestions, fmt.Sprintf("⚡ Performance: Generate thumbnails for %d performers to improve page load times", performersWithoutThumbnails))
	}

	// Check for videos without thumbnails
	var videosWithoutThumbnails int
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE thumbnail_path IS NULL OR thumbnail_path = ''").Scan(&videosWithoutThumbnails)
	if videosWithoutThumbnails > 20 {
		suggestions = append(suggestions, fmt.Sprintf("📸 Media: Generate thumbnails for %d videos for better browsing", videosWithoutThumbnails))
	}

	// Check for videos without previews
	var videosWithoutPreviews int
	s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE preview_path IS NULL OR preview_path = ''").Scan(&videosWithoutPreviews)
	if videosWithoutPreviews > 50 {
		suggestions = append(suggestions, fmt.Sprintf("🎬 Media: Generate preview storyboards for %d videos", videosWithoutPreviews))
	}

	// Check for performers without metadata
	var performersWithoutMetadata int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE category = 'regular' AND (metadata IS NULL OR metadata = '{}' OR metadata = '')").Scan(&performersWithoutMetadata)
	if performersWithoutMetadata > 10 {
		suggestions = append(suggestions, fmt.Sprintf("📊 Metadata: Fetch metadata for %d performers from AdultDataLink", performersWithoutMetadata))
	}

	// Check console logs for recent errors
	consoleLogSvc := NewConsoleLogService()
	errors, _, _ := consoleLogSvc.GetAll(10, 0, "", "error", "")
	if len(errors) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("🚨 Errors: %d recent errors detected - run 'check for errors' to analyze", len(errors)))
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "✅ Your system is running smoothly! No optimizations needed at this time.")
	}

	return suggestions, nil
}

// ================== Console Log Integration ==================

// GetRecentConsoleLogs retrieves recent console logs with optional filtering
func (s *AICompanionService) GetRecentConsoleLogs(limit int, source, level string) ([]*models.ConsoleLog, error) {
	consoleLogSvc := NewConsoleLogService()
	logs, _, err := consoleLogSvc.GetAll(limit, 0, source, level, "")
	return logs, err
}

// AnalyzeConsoleErrors checks console logs for errors and provides insights
func (s *AICompanionService) AnalyzeConsoleErrors() (string, error) {
	consoleLogSvc := NewConsoleLogService()

	// Get recent errors
	allErrors, _, err := consoleLogSvc.GetAll(50, 0, "", "error", "")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve error logs: %w", err)
	}

	// Filter out noisy/benign errors
	errors := make([]*models.ConsoleLog, 0)
	for _, e := range allErrors {
		if !isNoisyError(e.Message) {
			errors = append(errors, e)
		}
	}

	if len(errors) == 0 {
		return "✅ No significant errors detected! Your system is running smoothly.", nil
	}

	// Analyze error patterns
	errorPatterns := make(map[string]int)
	sources := make(map[string]int)

	for _, logEntry := range errors {
		// Count by source
		sources[logEntry.Source]++

		// Categorize errors
		msg := strings.ToLower(logEntry.Message)
		if strings.Contains(msg, "library") || strings.Contains(msg, "scan") {
			errorPatterns["library_scan"]++
		} else if strings.Contains(msg, "scraper") || strings.Contains(msg, "thread") {
			errorPatterns["scraper"]++
		} else if strings.Contains(msg, "database") || strings.Contains(msg, "sql") {
			errorPatterns["database"]++
		} else if strings.Contains(msg, "api") || strings.Contains(msg, "request") {
			errorPatterns["api"]++
		} else {
			errorPatterns["other"]++
		}
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("⚠️ Found %d significant errors (filtered %d noisy). Analysis:\n\n", len(errors), len(allErrors)-len(errors)))

	response.WriteString("Error Sources:\n")
	for source, count := range sources {
		response.WriteString(fmt.Sprintf("- %s: %d errors\n", source, count))
	}

	response.WriteString("\nError Categories:\n")
	for category, count := range errorPatterns {
		response.WriteString(fmt.Sprintf("- %s: %d errors\n", category, count))
	}

	// Provide recommendations
	if errorPatterns["library_scan"] > 0 {
		response.WriteString("\n💡 Recommendation: Library scan errors detected. Check library paths and permissions.")
	}
	if errorPatterns["scraper"] > 0 {
		response.WriteString("\n💡 Recommendation: Scraper errors detected. Check network connectivity and target URLs.")
	}
	if errorPatterns["database"] > 0 {
		response.WriteString("\n💡 Recommendation: Database errors detected. Check database integrity and disk space.")
	}

	return response.String(), nil
}

// isNoisyError returns true if the error message is benign/expected and should not trigger alerts
func isNoisyError(message string) bool {
	noisyPatterns := []string{
		"Error closing WebSocket connection",
		"use of closed network connection",
		"broken pipe",
		"connection reset by peer",
		"websocket: close",
		"context canceled",
		"http: Server closed",
		"[AI] Tool call",
		"[AI] LLM",
		"[AI] ChatLLM",
		"Network Error",
		"Failed to load dashboard",
		"Failed to load running tasks",
	}
	lowerMsg := strings.ToLower(message)
	for _, pattern := range noisyPatterns {
		if strings.Contains(lowerMsg, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// MonitorConsoleLogsForErrors periodically checks for new errors
func (s *AICompanionService) MonitorConsoleLogsForErrors() {
	ticker := time.NewTicker(2 * time.Minute) // Check every 2 minutes
	defer ticker.Stop()

	lastErrorCount := 0

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			consoleLogSvc := NewConsoleLogService()
			errors, _, err := consoleLogSvc.GetAll(10, 0, "", "error", "")
			if err != nil {
				continue
			}

			// Filter out noisy/benign errors
			significantErrors := make([]*models.ConsoleLog, 0)
			for _, e := range errors {
				if !isNoisyError(e.Message) {
					significantErrors = append(significantErrors, e)
				}
			}

			// Alert if new significant errors appeared
			if len(significantErrors) > lastErrorCount && len(significantErrors) > 0 {
				newErrors := len(significantErrors) - lastErrorCount
				recentError := significantErrors[0]

				s.emitEvent(CompanionEvent{
					Type:    "notification",
					Source:  "error_monitor",
					Message: fmt.Sprintf("🚨 %d new error(s) detected! Latest: %s", newErrors, recentError.Message),
					Data: map[string]interface{}{
						"new_errors":    newErrors,
						"total_errors":  len(significantErrors),
						"latest_source": recentError.Source,
						"latest_level":  recentError.Level,
					},
					Severity:  "error",
					Timestamp: time.Now(),
				})
			}

			lastErrorCount = len(significantErrors)
		}
	}
}

// ================== Chat Helper Methods ==================

// generateAppUpdate creates a comprehensive status update of the app
func (s *AICompanionService) generateAppUpdate() (string, error) {
	var response strings.Builder
	response.WriteString("📊 **App Status Update**\n\n")

	// Library stats
	var videoCount, performerCount, tagCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
	s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
	response.WriteString(fmt.Sprintf("**Library:** %d videos, %d performers, %d tags\n\n", videoCount, performerCount, tagCount))

	// Running tasks
	rows, err := s.db.Query(`SELECT task_type, message, created_at FROM activity_logs WHERE status = 'running' ORDER BY created_at DESC LIMIT 5`)
	if err == nil {
		defer rows.Close()
		running := []string{}
		for rows.Next() {
			var taskType, message string
			var createdAt time.Time
			if rows.Scan(&taskType, &message, &createdAt) == nil {
				running = append(running, fmt.Sprintf("  - [%s] %s (%s ago)", taskType, message, formatDuration(time.Since(createdAt))))
			}
		}
		if len(running) > 0 {
			response.WriteString("**Running Tasks:**\n")
			for _, r := range running {
				response.WriteString(r + "\n")
			}
			response.WriteString("\n")
		} else {
			response.WriteString("**Running Tasks:** None - system is idle\n\n")
		}
	}

	// Recent activity
	recentRows, err := s.db.Query(`SELECT task_type, status, message, created_at FROM activity_logs WHERE status != 'running' ORDER BY created_at DESC LIMIT 5`)
	if err == nil {
		defer recentRows.Close()
		response.WriteString("**Recent Activity:**\n")
		hasActivity := false
		for recentRows.Next() {
			var taskType, status, message string
			var createdAt time.Time
			if recentRows.Scan(&taskType, &status, &message, &createdAt) == nil {
				hasActivity = true
				response.WriteString(fmt.Sprintf("  - [%s] %s - %s (%s ago)\n", taskType, status, message, formatDuration(time.Since(createdAt))))
			}
		}
		if !hasActivity {
			response.WriteString("  No recent activity.\n")
		}
		response.WriteString("\n")
	}

	// Scraper summary
	var scrapedTotal int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos").Scan(&scrapedTotal)
	if scrapedTotal > 0 {
		response.WriteString(fmt.Sprintf("**Scraped Content:** %d total items\n", scrapedTotal))
	}

	// JDownloader
	var jdStatus string
	err = s.db.QueryRow("SELECT connection_status FROM jdownloader_config LIMIT 1").Scan(&jdStatus)
	if err == nil {
		response.WriteString(fmt.Sprintf("**JDownloader:** %s\n", jdStatus))
	}

	// Errors
	var errorCount int
	s.db.QueryRow("SELECT COUNT(*) FROM console_logs WHERE level = 'error' AND created_at > datetime('now', '-1 hour')").Scan(&errorCount)
	if errorCount > 0 {
		response.WriteString(fmt.Sprintf("\n⚠️ **%d error(s)** in the last hour\n", errorCount))
	} else {
		response.WriteString("\n✅ No errors in the last hour\n")
	}

	return response.String(), nil
}

// looksLikePerformerName checks if a message looks like a performer name (2-4 capitalized words, no common keywords)
func (s *AICompanionService) looksLikePerformerName(message string) bool {
	words := strings.Fields(message)
	if len(words) < 1 || len(words) > 5 {
		return false
	}

	// Must have at least one word starting with uppercase
	hasUppercase := false
	for _, w := range words {
		if len(w) > 0 && w[0] >= 'A' && w[0] <= 'Z' {
			hasUppercase = true
		}
	}
	if !hasUppercase {
		return false
	}

	// Reject if contains common keywords
	lower := strings.ToLower(message)
	commonWords := []string{"how", "what", "where", "when", "why", "show", "find", "search", "get", "list", "help", "status", "update", "check", "error", "video", "tag", "scraper", "library", "jdownloader", "hello", "hi", "hey", "please", "thanks", "yes", "no"}
	for _, w := range commonWords {
		if strings.Contains(lower, w) {
			return false
		}
	}

	// Check if this name exists in the performers table
	var exists int
	s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE name LIKE ?", "%"+message+"%").Scan(&exists)
	return exists > 0
}

// searchPerformerAcrossAll searches for a performer name in both library and scraped content
func (s *AICompanionService) searchPerformerAcrossAll(name string) (string, error) {
	var response strings.Builder

	// Search scraped videos
	var scrapedCount int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE performers LIKE ? OR title LIKE ?", "%"+name+"%", "%"+name+"%").Scan(&scrapedCount)

	if scrapedCount > 0 {
		response.WriteString(fmt.Sprintf("\n🔍 **Scraped Content:** Found %d items mentioning \"%s\"\n", scrapedCount, name))

		rows, err := s.db.Query(`SELECT source, title FROM scraped_videos WHERE performers LIKE ? OR title LIKE ? ORDER BY discovered_at DESC LIMIT 5`, "%"+name+"%", "%"+name+"%")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var source, title string
				if rows.Scan(&source, &title) == nil {
					response.WriteString(fmt.Sprintf("  - [%s] %s\n", strings.ToUpper(source), title))
				}
			}
			if scrapedCount > 5 {
				response.WriteString(fmt.Sprintf("  _... and %d more_\n", scrapedCount-5))
			}
		}
	}

	// Search forum posts
	var postCount int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts WHERE plain_text LIKE ?", "%"+name+"%").Scan(&postCount)
	if postCount > 0 {
		response.WriteString(fmt.Sprintf("\n💬 **Forum Posts:** %d posts mention \"%s\"\n", postCount, name))
	}

	if response.Len() == 0 {
		return "", nil
	}
	return response.String(), nil
}

// searchPerformerWithTag searches for a performer's content with a specific tag/keyword
func (s *AICompanionService) searchPerformerWithTag(performerName, tag string) (string, error) {
	var response strings.Builder
	response.WriteString(fmt.Sprintf("🔍 **Searching \"%s\" + \"%s\"**\n\n", performerName, tag))

	// Search in library videos
	rows, err := s.db.Query(`
		SELECT v.title
		FROM videos v
		JOIN video_performers vp ON v.id = vp.video_id
		JOIN performers p ON vp.performer_id = p.id
		WHERE p.name LIKE ? AND v.title LIKE ?
		LIMIT 10
	`, "%"+performerName+"%", "%"+tag+"%")

	libraryCount := 0
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var title string
			if rows.Scan(&title) == nil {
				libraryCount++
				response.WriteString(fmt.Sprintf("📹 %s\n", title))
			}
		}
	}

	// Search scraped content
	scrapedRows, err := s.db.Query(`
		SELECT source, title FROM scraped_videos
		WHERE (performers LIKE ? OR title LIKE ?) AND (title LIKE ? OR tags LIKE ?)
		ORDER BY discovered_at DESC LIMIT 10
	`, "%"+performerName+"%", "%"+performerName+"%", "%"+tag+"%", "%"+tag+"%")

	scrapedCount := 0
	if err == nil {
		defer scrapedRows.Close()
		for scrapedRows.Next() {
			var source, title string
			if scrapedRows.Scan(&source, &title) == nil {
				scrapedCount++
				response.WriteString(fmt.Sprintf("🔗 [%s] %s\n", strings.ToUpper(source), title))
			}
		}
	}

	// Search forum posts
	var postCount int
	s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts WHERE plain_text LIKE ? AND plain_text LIKE ?", "%"+performerName+"%", "%"+tag+"%").Scan(&postCount)

	if libraryCount == 0 && scrapedCount == 0 && postCount == 0 {
		response.WriteString(fmt.Sprintf("No results found for \"%s\" with \"%s\" in library, scraped videos, or forum posts.", performerName, tag))
	} else {
		response.WriteString(fmt.Sprintf("\n**Totals:** %d library videos, %d scraped items, %d forum posts\n", libraryCount, scrapedCount, postCount))
	}

	return response.String(), nil
}

// extractNameAfterPhrase extracts a name/subject after a trigger phrase
func extractNameAfterPhrase(message string, phrases []string) string {
	lower := strings.ToLower(message)
	for _, phrase := range phrases {
		idx := strings.Index(lower, phrase)
		if idx >= 0 {
			after := strings.TrimSpace(message[idx+len(phrase):])
			// Remove surrounding quotes if present
			after = strings.Trim(after, "\"'")
			if after != "" {
				return after
			}
		}
	}
	return ""
}

// listForumPosts lists actual forum posts matching a performer name and optional keyword
func (s *AICompanionService) listForumPosts(performerName, keyword string, limit int) (string, error) {
	var response strings.Builder
	var rows *sql.Rows
	var err error

	if keyword != "" {
		response.WriteString(fmt.Sprintf("💬 **Forum posts mentioning \"%s\" + \"%s\":**\n\n", performerName, keyword))
		rows, err = s.db.Query(`
			SELECT sp.plain_text, sp.author, st.title as thread_title
			FROM scraped_posts sp
			LEFT JOIN scraped_threads st ON sp.thread_id = st.id
			WHERE sp.plain_text LIKE ? AND sp.plain_text LIKE ?
			ORDER BY sp.scraped_at DESC LIMIT ?
		`, "%"+performerName+"%", "%"+keyword+"%", limit)
	} else {
		response.WriteString(fmt.Sprintf("💬 **Forum posts mentioning \"%s\":**\n\n", performerName))
		rows, err = s.db.Query(`
			SELECT sp.plain_text, sp.author, st.title as thread_title
			FROM scraped_posts sp
			LEFT JOIN scraped_threads st ON sp.thread_id = st.id
			WHERE sp.plain_text LIKE ?
			ORDER BY sp.scraped_at DESC LIMIT ?
		`, "%"+performerName+"%", limit)
	}

	if err != nil {
		return "Failed to search forum posts.", err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var text, author, threadTitle string
		if rows.Scan(&text, &author, &threadTitle) == nil {
			count++
			// Truncate long post text
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			response.WriteString(fmt.Sprintf("**%d.** [%s] by **%s**\n> %s\n\n", count, threadTitle, author, text))
		}
	}

	if count == 0 {
		return fmt.Sprintf("No forum posts found matching those criteria."), nil
	}

	response.WriteString(fmt.Sprintf("_Showing %d results_", count))
	return response.String(), nil
}

// searchScrapedContentByKeyword searches all scraped content (videos + posts) by keyword
func (s *AICompanionService) searchScrapedContentByKeyword(keyword string, limit int) (string, error) {
	var response strings.Builder
	response.WriteString(fmt.Sprintf("🔍 **Scraped content matching \"%s\":**\n\n", keyword))

	// Search scraped videos
	videoRows, err := s.db.Query(`
		SELECT source, title, performers, url
		FROM scraped_videos
		WHERE title LIKE ? OR performers LIKE ? OR tags LIKE ?
		ORDER BY discovered_at DESC LIMIT ?
	`, "%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%", limit)

	videoCount := 0
	if err == nil {
		defer videoRows.Close()
		for videoRows.Next() {
			var source, title, performers, url string
			if videoRows.Scan(&source, &title, &performers, &url) == nil {
				videoCount++
				response.WriteString(fmt.Sprintf("🎬 [%s] %s", strings.ToUpper(source), title))
				if performers != "" {
					response.WriteString(fmt.Sprintf(" (%s)", performers))
				}
				response.WriteString("\n")
			}
		}
	}

	// Search forum posts
	postRows, err := s.db.Query(`
		SELECT sp.plain_text, sp.author, st.title as thread_title
		FROM scraped_posts sp
		LEFT JOIN scraped_threads st ON sp.thread_id = st.id
		WHERE sp.plain_text LIKE ?
		ORDER BY sp.scraped_at DESC LIMIT ?
	`, "%"+keyword+"%", limit)

	postCount := 0
	if err == nil {
		defer postRows.Close()
		if videoCount > 0 {
			response.WriteString("\n")
		}
		for postRows.Next() {
			var text, author, threadTitle string
			if postRows.Scan(&text, &author, &threadTitle) == nil {
				postCount++
				if len(text) > 150 {
					text = text[:150] + "..."
				}
				response.WriteString(fmt.Sprintf("💬 [%s] by %s: %s\n", threadTitle, author, text))
			}
		}
	}

	if videoCount == 0 && postCount == 0 {
		return fmt.Sprintf("No scraped content found matching \"%s\".", keyword), nil
	}

	response.WriteString(fmt.Sprintf("\n**Found:** %d videos, %d forum posts", videoCount, postCount))
	return response.String(), nil
}

