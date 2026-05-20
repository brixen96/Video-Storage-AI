package services

import (
	"fmt"
	"strings"
)

// parseDuration converts duration strings like "01:21", "12:34", "1:23:45" to seconds
// Also handles "HD 00:03" by stripping out quality indicators
func parseDuration(durationStr string) int {
	durationStr = strings.TrimSpace(durationStr)
	if durationStr == "" {
		return 0
	}

	// Remove quality indicators like "HD", "4K", "720p", etc.
	// Pattern: "HD 00:03" -> "00:03"
	qualityIndicators := []string{"HD", "4K", "UHD", "FHD", "2160p", "1440p", "1080p", "720p", "480p", "360p", "240p", "144p"}
	for _, indicator := range qualityIndicators {
		durationStr = strings.ReplaceAll(durationStr, indicator, "")
	}
	durationStr = strings.TrimSpace(durationStr)

	// Handle "NNm NNs" or "NNm" format (e.g., HQPorner: "35m 21s", "37m 7s", "1h 23m 45s")
	if strings.Contains(durationStr, "m") || (strings.Contains(durationStr, "h") && strings.Contains(durationStr, "s")) {
		totalSeconds := 0
		remaining := durationStr
		if hIdx := strings.Index(remaining, "h"); hIdx > 0 {
			var hours int
			if _, err := fmt.Sscanf(strings.TrimSpace(remaining[:hIdx]), "%d", &hours); err == nil {
				totalSeconds += hours * 3600
			}
			remaining = strings.TrimSpace(remaining[hIdx+1:])
		}
		if mIdx := strings.Index(remaining, "m"); mIdx >= 0 {
			var mins int
			if _, err := fmt.Sscanf(strings.TrimSpace(remaining[:mIdx]), "%d", &mins); err == nil {
				totalSeconds += mins * 60
			}
			remaining = strings.TrimSpace(remaining[mIdx+1:])
		}
		if sIdx := strings.Index(remaining, "s"); sIdx >= 0 {
			var secs int
			if _, err := fmt.Sscanf(strings.TrimSpace(remaining[:sIdx]), "%d", &secs); err == nil {
				totalSeconds += secs
			}
		}
		if totalSeconds > 0 {
			return totalSeconds
		}
	}

	parts := strings.Split(durationStr, ":")
	totalSeconds := 0

	switch len(parts) {
	case 1:
		// Just seconds: "45"
		var s int
		if _, err := fmt.Sscanf(parts[0], "%d", &s); err == nil {
			totalSeconds = s
		}
	case 2:
		// MM:SS format: "12:34"
		var minutes, seconds int
		if _, err := fmt.Sscanf(durationStr, "%d:%d", &minutes, &seconds); err == nil {
			totalSeconds = minutes*60 + seconds
		}
	case 3:
		// HH:MM:SS format: "1:23:45"
		var hours, minutes, seconds int
		if _, err := fmt.Sscanf(durationStr, "%d:%d:%d", &hours, &minutes, &seconds); err == nil {
			totalSeconds = hours*3600 + minutes*60 + seconds
		}
	}

	return totalSeconds
}

// extractResolutionFromURL extracts resolution from video URL (e.g., "video_720p.mp4" -> "720p")
func extractResolutionFromURL(urlStr string) string {
	// Common patterns: 720p, 1080p, 4K, 240p, 360p, etc.
	resolutionPatterns := []string{
		"2160p", "4K", "1440p", "1080p", "720p", "480p", "360p", "240p", "144p",
	}

	urlLower := strings.ToLower(urlStr)
	for _, pattern := range resolutionPatterns {
		if strings.Contains(urlLower, strings.ToLower(pattern)) {
			return pattern
		}
	}

	return ""
}

// extractResolutionFromText extracts resolution from text (e.g., "HD 720p" -> "720p")
func extractResolutionFromText(text string) string {
	// Common patterns: 720p, 1080p, 4K, 240p, 360p, etc.
	resolutionPatterns := []string{
		"2160p", "4K", "1440p", "1080p", "720p", "480p", "360p", "240p", "144p",
	}

	textLower := strings.ToLower(text)
	for _, pattern := range resolutionPatterns {
		if strings.Contains(textLower, strings.ToLower(pattern)) {
			return pattern
		}
	}

	// Also check for just "HD" or "4K" without 'p'
	if strings.Contains(textLower, "4k") || strings.Contains(textLower, "uhd") {
		return "4K"
	}
	if strings.Contains(textLower, "hd") && !strings.Contains(textLower, "720") {
		return "720p" // Assume HD means 720p
	}

	return ""
}

// parseViewCount extracts view count from text like "148164 views" or "1.5M views"
func parseViewCount(viewsText string) int {
	// Remove "views" text and trim
	viewsText = strings.ToLower(viewsText)
	viewsText = strings.ReplaceAll(viewsText, "views", "")
	viewsText = strings.ReplaceAll(viewsText, "view", "")
	viewsText = strings.TrimSpace(viewsText)

	if viewsText == "" {
		return 0
	}

	// Handle abbreviations like "1.5M", "148K", etc.
	multiplier := 1
	if strings.HasSuffix(viewsText, "k") {
		multiplier = 1000
		viewsText = strings.TrimSuffix(viewsText, "k")
	} else if strings.HasSuffix(viewsText, "m") {
		multiplier = 1000000
		viewsText = strings.TrimSuffix(viewsText, "m")
	} else if strings.HasSuffix(viewsText, "b") {
		multiplier = 1000000000
		viewsText = strings.TrimSuffix(viewsText, "b")
	}

	// Remove commas
	viewsText = strings.ReplaceAll(viewsText, ",", "")

	// Parse the number
	var count float64
	if _, err := fmt.Sscanf(viewsText, "%f", &count); err == nil {
		return int(count * float64(multiplier))
	}

	return 0
}

// parseRating extracts rating percentage from text like "86%" or "86% thumbs up"
func parseRating(ratingText string) int {
	// Look for percentage pattern
	ratingText = strings.TrimSpace(ratingText)

	// Find percentage sign and extract number before it
	if idx := strings.Index(ratingText, "%"); idx > 0 {
		// Extract everything before the %
		numStr := ratingText[:idx]
		// Remove any non-digit characters except decimal point
		numStr = strings.TrimSpace(numStr)

		// Try to find the last contiguous number before %
		var rating int
		// Scan backwards from % to find the number
		for i := len(numStr) - 1; i >= 0; i-- {
			if numStr[i] >= '0' && numStr[i] <= '9' {
				continue
			} else if numStr[i] == '.' || numStr[i] == ' ' {
				numStr = numStr[i+1:]
				break
			} else {
				numStr = numStr[i+1:]
				break
			}
		}

		if _, err := fmt.Sscanf(numStr, "%d", &rating); err == nil {
			if rating >= 0 && rating <= 100 {
				return rating
			}
		}
	}

	return 0
}

// findHighestResolution finds the highest quality resolution from a list
func findHighestResolution(resolutions []string) string {
	resolutionPriority := map[string]int{
		"2160p": 9, "4K": 9,
		"1440p": 8,
		"1080p": 7,
		"720p":  6,
		"480p":  5,
		"360p":  4,
		"240p":  3,
		"144p":  2,
	}

	highest := ""
	highestPriority := 0

	for _, res := range resolutions {
		priority, exists := resolutionPriority[res]
		if exists && priority > highestPriority {
			highestPriority = priority
			highest = res
		}
	}

	return highest
}

// FindDuplicates finds potential duplicate videos across the database
func (s *VideoScraperService) FindDuplicates() (*ScrapedDuplicateStats, error) {
	stats := &ScrapedDuplicateStats{
		ByMatchType: make(map[string]int),
		BySiteType:  make(map[string]int),
		Duplicates:  []*ScrapedDuplicateMatch{},
	}

	// 1. Find exact URL duplicates (same URL across different site_types)
	exactURLQuery := `
		SELECT
			v1.id, v1.title, v1.url, v1.source, v1.site_type,
			v2.id, v2.title, v2.url, v2.source, v2.site_type
		FROM scraped_videos v1
		JOIN scraped_videos v2 ON v1.url = v2.url AND v1.id < v2.id
		ORDER BY v1.id
	`
	rows, err := s.db.Query(exactURLQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to find URL duplicates: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v1ID, v2ID int64
		var v1Title, v1URL, v1Source, v1SiteType string
		var v2Title, v2URL, v2Source, v2SiteType string
		if err := rows.Scan(&v1ID, &v1Title, &v1URL, &v1Source, &v1SiteType, &v2ID, &v2Title, &v2URL, &v2Source, &v2SiteType); err != nil {
			continue
		}
		dup := &ScrapedDuplicateMatch{
			Video1ID:     v1ID,
			Video1Title:  v1Title,
			Video1URL:    v1URL,
			Video1Source: v1Source,
			Video2ID:     v2ID,
			Video2Title:  v2Title,
			Video2URL:    v2URL,
			Video2Source: v2Source,
			MatchType:    "exact_url",
			Similarity:   1.0,
		}
		stats.Duplicates = append(stats.Duplicates, dup)
		stats.ByMatchType["exact_url"]++
		stats.BySiteType[v1SiteType]++
	}
	rows.Close()

	// 2. Find duplicates by external_id (same external_id across sources)
	externalIDQuery := `
		SELECT
			v1.id, v1.title, v1.url, v1.source, v1.site_type,
			v2.id, v2.title, v2.url, v2.source, v2.site_type
		FROM scraped_videos v1
		JOIN scraped_videos v2 ON
			v1.external_id = v2.external_id
			AND v1.external_id IS NOT NULL
			AND v1.external_id != ''
			AND v1.id < v2.id
		ORDER BY v1.id
	`
	rows2, err := s.db.Query(externalIDQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to find external_id duplicates: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var v1ID, v2ID int64
		var v1Title, v1URL, v1Source, v1SiteType string
		var v2Title, v2URL, v2Source, v2SiteType string
		if err := rows2.Scan(&v1ID, &v1Title, &v1URL, &v1Source, &v1SiteType, &v2ID, &v2Title, &v2URL, &v2Source, &v2SiteType); err != nil {
			continue
		}
		dup := &ScrapedDuplicateMatch{
			Video1ID:     v1ID,
			Video1Title:  v1Title,
			Video1URL:    v1URL,
			Video1Source: v1Source,
			Video2ID:     v2ID,
			Video2Title:  v2Title,
			Video2URL:    v2URL,
			Video2Source: v2Source,
			MatchType:    "external_id",
			Similarity:   1.0,
		}
		stats.Duplicates = append(stats.Duplicates, dup)
		stats.ByMatchType["external_id"]++
		stats.BySiteType[v1SiteType]++
	}
	rows2.Close()

	// 3. Find similar titles using normalized comparison
	// Get all videos and compare titles in memory for fuzzy matching
	titleQuery := `
		SELECT id, title, url, source, site_type, duration
		FROM scraped_videos
		ORDER BY id
	`
	rows3, err := s.db.Query(titleQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos for title matching: %w", err)
	}
	defer rows3.Close()

	type videoInfo struct {
		ID        int64
		Title     string
		URL       string
		Source    string
		SiteType  string
		Duration  int
		NormTitle string
	}
	var allVideos []videoInfo

	for rows3.Next() {
		var v videoInfo
		if err := rows3.Scan(&v.ID, &v.Title, &v.URL, &v.Source, &v.SiteType, &v.Duration); err != nil {
			continue
		}
		v.NormTitle = normalizeTitle(v.Title)
		allVideos = append(allVideos, v)
	}
	rows3.Close()

	// Track already found duplicates to avoid reporting same pair multiple times
	foundPairs := make(map[string]bool)
	for _, dup := range stats.Duplicates {
		key := fmt.Sprintf("%d-%d", dup.Video1ID, dup.Video2ID)
		foundPairs[key] = true
	}

	// Compare titles for similar matches
	for i := 0; i < len(allVideos); i++ {
		for j := i + 1; j < len(allVideos); j++ {
			v1 := allVideos[i]
			v2 := allVideos[j]

			// Skip if same source (might be intentional variations)
			if v1.Source == v2.Source {
				continue
			}

			// Skip if already found as URL or external_id duplicate
			key := fmt.Sprintf("%d-%d", v1.ID, v2.ID)
			if foundPairs[key] {
				continue
			}

			// Check exact normalized title match
			if v1.NormTitle == v2.NormTitle && v1.NormTitle != "" && len(v1.NormTitle) > 10 {
				dup := &ScrapedDuplicateMatch{
					Video1ID:     v1.ID,
					Video1Title:  v1.Title,
					Video1URL:    v1.URL,
					Video1Source: v1.Source,
					Video2ID:     v2.ID,
					Video2Title:  v2.Title,
					Video2URL:    v2.URL,
					Video2Source: v2.Source,
					MatchType:    "exact_title",
					Similarity:   1.0,
				}
				stats.Duplicates = append(stats.Duplicates, dup)
				stats.ByMatchType["exact_title"]++
				stats.BySiteType[v1.SiteType]++
				foundPairs[key] = true
				continue
			}

			// Check fuzzy title match (if titles are long enough)
			if len(v1.NormTitle) > 15 && len(v2.NormTitle) > 15 {
				similarity := calculateTitleSimilarity(v1.NormTitle, v2.NormTitle)
				// Also check if duration matches (strong indicator of duplicate)
				durationMatch := v1.Duration > 0 && v2.Duration > 0 &&
					(v1.Duration == v2.Duration || absInt(v1.Duration-v2.Duration) <= 5)

				// Require high similarity, or moderate similarity + same duration
				if similarity > 0.85 || (similarity > 0.7 && durationMatch) {
					dup := &ScrapedDuplicateMatch{
						Video1ID:     v1.ID,
						Video1Title:  v1.Title,
						Video1URL:    v1.URL,
						Video1Source: v1.Source,
						Video2ID:     v2.ID,
						Video2Title:  v2.Title,
						Video2URL:    v2.URL,
						Video2Source: v2.Source,
						MatchType:    "similar_title",
						Similarity:   similarity,
					}
					stats.Duplicates = append(stats.Duplicates, dup)
					stats.ByMatchType["similar_title"]++
					stats.BySiteType[v1.SiteType]++
					foundPairs[key] = true
				}
			}
		}
	}

	stats.TotalDuplicates = len(stats.Duplicates)
	return stats, nil
}

// normalizeTitle normalizes a video title for comparison
func normalizeTitle(title string) string {
	// Convert to lowercase
	normalized := strings.ToLower(title)

	// Remove common patterns that vary between sites
	// Remove resolution indicators
	resPatterns := []string{"[4k]", "[1080p]", "[720p]", "[480p]", "[hd]", "4k", "1080p", "720p", "480p", "hd"}
	for _, pattern := range resPatterns {
		normalized = strings.ReplaceAll(normalized, pattern, "")
	}

	// Remove common suffixes/prefixes
	suffixes := []string{" - pornhub.com", " - xvideos.com", " porn", " xxx", " free porn", " video"}
	for _, suffix := range suffixes {
		normalized = strings.TrimSuffix(normalized, suffix)
	}

	// Remove special characters except spaces
	var result strings.Builder
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			result.WriteRune(r)
		}
	}
	normalized = result.String()

	// Collapse multiple spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}

	return strings.TrimSpace(normalized)
}

// calculateTitleSimilarity calculates similarity between two normalized titles
func calculateTitleSimilarity(title1, title2 string) float64 {
	if title1 == title2 {
		return 1.0
	}

	// Use Jaccard similarity on word sets
	words1 := strings.Fields(title1)
	words2 := strings.Fields(title2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Create word sets
	set1 := make(map[string]bool)
	for _, w := range words1 {
		if len(w) > 2 { // Only consider words > 2 chars
			set1[w] = true
		}
	}
	set2 := make(map[string]bool)
	for _, w := range words2 {
		if len(w) > 2 {
			set2[w] = true
		}
	}

	// Calculate intersection and union
	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	union := len(set1)
	for w := range set2 {
		if !set1[w] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// absInt returns absolute value of an int
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// DeleteDuplicates removes duplicate videos, keeping the one with more metadata
func (s *VideoScraperService) DeleteDuplicates(keepStrategy string) (int, error) {
	stats, err := s.FindDuplicates()
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, dup := range stats.Duplicates {
		// Determine which video to keep based on strategy
		var keepID, deleteID int64
		switch keepStrategy {
		case "keep_first":
			keepID = dup.Video1ID
			deleteID = dup.Video2ID
		case "keep_last":
			keepID = dup.Video2ID
			deleteID = dup.Video1ID
		default:
			// Default: keep the one with more views/rating
			v1, _ := s.GetScrapedVideoByID(dup.Video1ID)
			v2, _ := s.GetScrapedVideoByID(dup.Video2ID)
			if v1 != nil && v2 != nil {
				// Score based on metadata completeness
				score1 := 0
				score2 := 0
				if v1.ViewCount > 0 {
					score1++
				}
				if v2.ViewCount > 0 {
					score2++
				}
				if v1.Rating > 0 {
					score1++
				}
				if v2.Rating > 0 {
					score2++
				}
				if v1.Duration > 0 {
					score1++
				}
				if v2.Duration > 0 {
					score2++
				}
				if v1.ThumbnailURL != "" {
					score1++
				}
				if v2.ThumbnailURL != "" {
					score2++
				}

				if score1 >= score2 {
					keepID = dup.Video1ID
					deleteID = dup.Video2ID
				} else {
					keepID = dup.Video2ID
					deleteID = dup.Video1ID
				}
			} else {
				keepID = dup.Video1ID
				deleteID = dup.Video2ID
			}
		}

		// Delete the duplicate
		_, err := s.db.Exec("DELETE FROM scraped_videos WHERE id = ?", deleteID)
		if err == nil {
			deleted++
			fmt.Printf("Deleted duplicate video ID %d (kept ID %d)\n", deleteID, keepID)
		}
	}

	return deleted, nil
}
