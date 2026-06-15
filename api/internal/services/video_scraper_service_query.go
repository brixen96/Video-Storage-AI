package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// saveVideo saves a scraped video to the database
func (s *VideoScraperService) saveVideo(video *ScrapedVideo) error {
	tagsJSON, err := json.Marshal(video.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	// Helper function to convert empty strings to NULL
	toNullString := func(s string) interface{} {
		if s == "" {
			return nil
		}
		return s
	}

	// First check if video already exists
	var existingID int64
	checkQuery := `SELECT id FROM scraped_videos WHERE url = ? AND source = ?`
	err = s.db.QueryRow(checkQuery, video.URL, video.Source).Scan(&existingID)

	if err == nil {
		// Video already exists, update it with new data (e.g., duration if it was missing)
		updateQuery := `
			UPDATE scraped_videos SET
				title = ?,
				thumbnail_url = COALESCE(?, thumbnail_url),
				duration = CASE WHEN ? > 0 THEN ? ELSE duration END,
				view_count = CASE WHEN ? > 0 THEN ? ELSE view_count END,
				rating = CASE WHEN ? > 0 THEN ? ELSE rating END
			WHERE id = ?
		`
		_, err := s.db.Exec(
			updateQuery,
			video.Title,
			toNullString(video.ThumbnailURL),
			video.Duration, video.Duration,
			video.ViewCount, video.ViewCount,
			video.Rating, video.Rating,
			existingID,
		)
		if err != nil {
			return fmt.Errorf("failed to update existing video (url=%s): %w", video.URL, err)
		}
		video.ID = existingID
		// Return a special "error" to indicate this was an update (not a real error)
		return fmt.Errorf("Updated existing video (ID=%d)", existingID)
	}

	// Video doesn't exist, insert new one
	query := `
		INSERT INTO scraped_videos (
			source, site_type, content_type, external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?
		)
	`

	contentType := video.ContentType
	if contentType == "" {
		contentType = "regular"
	}

	result, err := s.db.Exec(
		query,
		video.Source, video.SiteType, contentType, toNullString(video.ExternalID), video.Title, video.URL, toNullString(video.Description),
		video.Duration, video.FileSize, toNullString(video.ThumbnailURL), toNullString(video.ThumbnailPath), toNullString(video.VideoPath),
		toNullString(video.Author), toNullString(video.Uploader), video.ViewCount, video.LikeCount, video.Rating, toNullString(video.Category), string(tagsJSON),
		video.Metadata, video.Status, toNullString(video.ErrorMessage), video.DiscoveredAt, video.ScrapedAt,
		video.DownloadedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert video (url=%s): %w", video.URL, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	video.ID = id
	return nil
}

// GetScrapedVideos retrieves all scraped videos
func (s *VideoScraperService) GetScrapedVideos() ([]*ScrapedVideo, error) {
	query := `
		SELECT
			id, source, site_type, COALESCE(content_type, 'regular'), external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at, created_at
		FROM scraped_videos
		ORDER BY discovered_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	var videos []*ScrapedVideo
	for rows.Next() {
		video := &ScrapedVideo{}
		var tagsJSON string
		var externalID, description, thumbnailURL, thumbnailPath, videoPath sql.NullString
		var author, uploader, category, errorMessage sql.NullString
		var scrapedAt, downloadedAt sql.NullTime

		err := rows.Scan(
			&video.ID, &video.Source, &video.SiteType, &video.ContentType, &externalID, &video.Title, &video.URL, &description,
			&video.Duration, &video.FileSize, &thumbnailURL, &thumbnailPath, &videoPath,
			&author, &uploader, &video.ViewCount, &video.LikeCount, &video.Rating, &category, &tagsJSON,
			&video.Metadata, &video.Status, &errorMessage, &video.DiscoveredAt, &scrapedAt,
			&downloadedAt, &video.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}

		// Handle nullable fields
		if externalID.Valid {
			video.ExternalID = externalID.String
		}
		if description.Valid {
			video.Description = description.String
		}
		if thumbnailURL.Valid {
			video.ThumbnailURL = thumbnailURL.String
		}
		if thumbnailPath.Valid {
			video.ThumbnailPath = thumbnailPath.String
		}
		if videoPath.Valid {
			video.VideoPath = videoPath.String
		}
		if author.Valid {
			video.Author = author.String
		}
		if uploader.Valid {
			video.Uploader = uploader.String
		}
		if category.Valid {
			video.Category = category.String
		}
		if errorMessage.Valid {
			video.ErrorMessage = errorMessage.String
		}
		if scrapedAt.Valid {
			video.ScrapedAt = &scrapedAt.Time
		}
		if downloadedAt.Valid {
			video.DownloadedAt = &downloadedAt.Time
		}

		// Unmarshal tags
		if err := json.Unmarshal([]byte(tagsJSON), &video.Tags); err != nil {
			video.Tags = []string{}
		}

		// Filter out blocked content
		if IsVideoBlocked(video) {
			continue
		}

		// Filter out Sexuria items - they should only be shown in the Sexuria section
		if video.SiteType == "sexuria" || video.Source == "sexuria.net" {
			continue
		}

		videos = append(videos, video)
	}

	return videos, nil
}

// GetScrapedVideosPaginated retrieves scraped videos with pagination and filtering
func (s *VideoScraperService) GetScrapedVideosPaginated(q ScrapedVideosQuery) (*ScrapedVideosResult, error) {
	// Set defaults
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 24
	} else if q.Limit > 500000 {
		q.Limit = 500000 // Safety cap to prevent unbounded queries
	}
	if q.SortBy == "" {
		q.SortBy = "discovered_at"
	}
	if q.SortOrder == "" {
		q.SortOrder = "desc"
	}

	// Validate sort field to prevent SQL injection
	validSortFields := map[string]bool{
		"discovered_at": true,
		"title":         true,
		"view_count":    true,
		"like_count":    true,
		"rating":        true,
		"duration":      true,
		"created_at":    true,
		"file_size":     true,
	}
	if !validSortFields[q.SortBy] {
		q.SortBy = "discovered_at"
	}
	if q.SortOrder != "asc" && q.SortOrder != "desc" {
		q.SortOrder = "desc"
	}

	// Build WHERE clause
	var conditions []string
	var args []interface{}

	if q.Source != "" {
		conditions = append(conditions, "source = ?")
		args = append(args, q.Source)
	}
	if q.SourceLike != "" {
		conditions = append(conditions, "source LIKE ?")
		args = append(args, q.SourceLike)
	}
	if q.SiteType != "" {
		conditions = append(conditions, "site_type = ?")
		args = append(args, q.SiteType)
	}
	if q.ContentType != "" {
		conditions = append(conditions, "COALESCE(content_type, 'regular') = ?")
		args = append(args, q.ContentType)
	}
	if q.Category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, q.Category)
	}
	if q.CategorySearch != "" {
		conditions = append(conditions, "category LIKE ?")
		args = append(args, "%"+q.CategorySearch+"%")
	}
	if q.Search != "" {
		for _, word := range strings.Fields(q.Search) {
			conditions = append(conditions, "title LIKE ?")
			args = append(args, "%"+word+"%")
		}
	}
	if q.Keyword != "" {
		conditions = append(conditions, "tags LIKE ?")
		args = append(args, "%\""+q.Keyword+"\"%")
	}
	for _, kw := range q.IncludeKeywords {
		if kw != "" {
			conditions = append(conditions, "tags LIKE ?")
			args = append(args, "%\""+kw+"\"%")
		}
	}
	for _, kw := range q.ExcludeKeywords {
		if kw != "" {
			conditions = append(conditions, "tags NOT LIKE ?")
			args = append(args, "%\""+kw+"\"%")
		}
	}
	if q.AuthorSearch != "" {
		conditions = append(conditions, "author LIKE ?")
		args = append(args, "%"+q.AuthorSearch+"%")
	}
	if q.PerformerID > 0 {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM json_each(json_extract(metadata, '$.performers')) p
			WHERE CAST(json_extract(p.value, '$.id') AS INTEGER) = ?
		)`)
		args = append(args, q.PerformerID)
	}
	if q.TagID > 0 {
		conditions = append(conditions, `EXISTS (
			SELECT 1 FROM json_each(json_extract(metadata, '$.matched_tags')) t
			WHERE CAST(json_extract(t.value, '$.id') AS INTEGER) = ?
		)`)
		args = append(args, q.TagID)
	}
	switch q.DurationFilter {
	case "short":
		conditions = append(conditions, "duration > 0 AND duration < 300")
	case "medium":
		conditions = append(conditions, "duration >= 300 AND duration <= 1200")
	case "long":
		conditions = append(conditions, "duration > 1200")
	}
	if q.OnlyDownloaded {
		conditions = append(conditions, "downloaded_at IS NOT NULL")
	}
	if q.MinRating > 0 {
		conditions = append(conditions, "COALESCE(rating,0) >= ?")
		args = append(args, q.MinRating)
	}
	if q.MinViews > 0 {
		conditions = append(conditions, "COALESCE(view_count,0) >= ?")
		args = append(args, q.MinViews)
	}
	if q.MinLikes > 0 {
		conditions = append(conditions, "COALESCE(like_count,0) >= ?")
		args = append(args, q.MinLikes)
	}
	if q.DateFrom != "" {
		conditions = append(conditions, "discovered_at >= ?")
		args = append(args, q.DateFrom)
	}
	if q.DateTo != "" {
		conditions = append(conditions, "discovered_at <= ?")
		args = append(args, q.DateTo+" 23:59:59")
	}
	if q.MinDuration > 0 {
		conditions = append(conditions, "duration >= ?")
		args = append(args, q.MinDuration)
	}
	if q.MaxDuration > 0 {
		conditions = append(conditions, "duration <= ?")
		args = append(args, q.MaxDuration)
	}
	if q.MinFileSize > 0 {
		conditions = append(conditions, "COALESCE(file_size,0) >= ?")
		args = append(args, q.MinFileSize)
	}
	if q.MaxFileSize > 0 {
		conditions = append(conditions, "COALESCE(file_size,0) <= ?")
		args = append(args, q.MaxFileSize)
	}
	if q.HasThumbnail != nil {
		if *q.HasThumbnail {
			conditions = append(conditions, "COALESCE(thumbnail_url, thumbnail_path, '') != ''")
		} else {
			conditions = append(conditions, "(thumbnail_url IS NULL OR thumbnail_url = '') AND (thumbnail_path IS NULL OR thumbnail_path = '')")
		}
	}
	if q.OnlyWithAuthor {
		conditions = append(conditions, "author IS NOT NULL AND author != ''")
	}
	if q.OnlyWithRating {
		conditions = append(conditions, "COALESCE(rating,0) > 0")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM scraped_videos %s", whereClause)
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count videos: %w", err)
	}

	// Calculate offset
	offset := (q.Page - 1) * q.Limit

	// Build main query with pagination
	query := fmt.Sprintf(`
		SELECT
			id, source, site_type, COALESCE(content_type, 'regular'), external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at, created_at
		FROM scraped_videos
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, whereClause, q.SortBy, strings.ToUpper(q.SortOrder))

	// Add pagination args
	args = append(args, q.Limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	var videos []*ScrapedVideo
	for rows.Next() {
		video := &ScrapedVideo{}
		var tagsJSON string
		var externalID, description, thumbnailURL, thumbnailPath, videoPath sql.NullString
		var author, uploader, category, errorMessage sql.NullString
		var scrapedAt, downloadedAt sql.NullTime

		err := rows.Scan(
			&video.ID, &video.Source, &video.SiteType, &video.ContentType, &externalID, &video.Title, &video.URL, &description,
			&video.Duration, &video.FileSize, &thumbnailURL, &thumbnailPath, &videoPath,
			&author, &uploader, &video.ViewCount, &video.LikeCount, &video.Rating, &category, &tagsJSON,
			&video.Metadata, &video.Status, &errorMessage, &video.DiscoveredAt, &scrapedAt,
			&downloadedAt, &video.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}

		// Handle nullable fields
		if externalID.Valid {
			video.ExternalID = externalID.String
		}
		if description.Valid {
			video.Description = description.String
		}
		if thumbnailURL.Valid {
			video.ThumbnailURL = thumbnailURL.String
		}
		if thumbnailPath.Valid {
			video.ThumbnailPath = thumbnailPath.String
		}
		if videoPath.Valid {
			video.VideoPath = videoPath.String
		}
		if author.Valid {
			video.Author = author.String
		}
		if uploader.Valid {
			video.Uploader = uploader.String
		}
		if category.Valid {
			video.Category = category.String
		}
		if errorMessage.Valid {
			video.ErrorMessage = errorMessage.String
		}
		if scrapedAt.Valid {
			video.ScrapedAt = &scrapedAt.Time
		}
		if downloadedAt.Valid {
			video.DownloadedAt = &downloadedAt.Time
		}

		// Unmarshal tags
		if err := json.Unmarshal([]byte(tagsJSON), &video.Tags); err != nil {
			video.Tags = []string{}
		}

		// On-the-fly thumbnail_frames derivation for scrapers that don't store frames
		metaJSON := video.Metadata
		if metaJSON == "" {
			metaJSON = "{}"
		}
		if !strings.Contains(metaJSON, "thumbnail_frames") && video.ThumbnailURL != "" {
			var frames []string
			switch video.SiteType {
			case "motherless":
				// Thumbnails stored as {hash}-strip.jpg or {hash}-small-{N}.jpg.
				// Derive 8 individual frames: {hash}-small-1.jpg through {hash}-small-8.jpg
				reMotherless := regexp.MustCompile(`-(strip|small-\d+)\.(jpg|jpeg|png|webp)$`)
				if reMotherless.MatchString(video.ThumbnailURL) {
					motherlessM := reMotherless.FindStringSubmatch(video.ThumbnailURL)
					ext := motherlessM[2]
					base := reMotherless.ReplaceAllString(video.ThumbnailURL, "")
					for n := 1; n <= 8; n++ {
						frames = append(frames, base+"-small-"+strconv.Itoa(n)+"."+ext)
					}
				}
			case "leakvids", "rule34video":
				// Pattern: .../N.jpg (last path segment is a frame number)
				// LeakVids:    https://leakvids.com/contents/videos_screenshots/{a}/{id}/{res}/{N}.jpg
				// Rule34Video: https://rule34video.com/contents/videos_screenshots/{a}/{id}/{res}/{N}.jpg
				reLeakVids := regexp.MustCompile(`/(\d+)\.(jpg|jpeg|png|webp)$`)
				if reLeakVids.MatchString(video.ThumbnailURL) {
					m := reLeakVids.FindStringSubmatch(video.ThumbnailURL)
					ext := m[2]
					base := video.ThumbnailURL[:strings.LastIndex(video.ThumbnailURL, "/")]
					for n := 1; n <= 10; n++ {
						frames = append(frames, base+"/"+strconv.Itoa(n)+"."+ext)
					}
				}
			case "luxuretv":
				// Pattern: https://thumb.luxuretv.com/.../filename.mp4-{N}.jpg
				// The startm() JS on the listing page cycles -1.jpg, -2.jpg, ... for animation.
				reLuxure := regexp.MustCompile(`-(\d+)\.(jpg|jpeg|png|webp)$`)
				if reLuxure.MatchString(video.ThumbnailURL) {
					m := reLuxure.FindStringSubmatch(video.ThumbnailURL)
					ext := m[2]
					base := reLuxure.ReplaceAllString(video.ThumbnailURL, "")
					for n := 1; n <= 10; n++ {
						frames = append(frames, base+"-"+strconv.Itoa(n)+"."+ext)
					}
				}
			case "pornhub":
				// Pattern: https://ei.phncdn.com/.../original/(m=...)(mh=...)N.jpg
				// The frame number follows the closing ")" of the quality params.
				rePH := regexp.MustCompile(`\)(\d+)\.(jpg|jpeg|png|webp)$`)
				if rePH.MatchString(video.ThumbnailURL) {
					m := rePH.FindStringSubmatch(video.ThumbnailURL)
					ext := m[2]
					maxFrame, _ := strconv.Atoi(m[1])
					if maxFrame > 20 {
						maxFrame = 20
					}
					closingParen := strings.LastIndex(video.ThumbnailURL, ")")
					base := video.ThumbnailURL[:closingParen+1]
					for n := 0; n <= maxFrame; n++ {
						frames = append(frames, base+strconv.Itoa(n)+"."+ext)
					}
				}
			}
			if len(frames) > 0 {
				var meta map[string]interface{}
				if json.Unmarshal([]byte(metaJSON), &meta) != nil {
					meta = map[string]interface{}{}
				}
				meta["thumbnail_frames"] = frames
				if b, err2 := json.Marshal(meta); err2 == nil {
					video.Metadata = string(b)
				}
			}
		}

		// Filter out blocked content
		if IsVideoBlocked(video) {
			total-- // Adjust total count
			continue
		}

		videos = append(videos, video)
	}

	totalPages := (total + q.Limit - 1) / q.Limit

	// Optionally populate SourceTabName from scraper_tabs
	if q.IncludeTabInfo && len(videos) > 0 {
		type tabInfo struct {
			Name        string
			ContentType string
		}
		tabMap := make(map[string]tabInfo)
		tabRows, err := s.db.Query(`SELECT base_url, name, content_type FROM scraper_tabs WHERE is_active = 1`)
		if err == nil {
			defer tabRows.Close()
			for tabRows.Next() {
				var baseURL, name, ct string
				if tabRows.Scan(&baseURL, &name, &ct) == nil {
					if parsedURL, err := url.Parse(baseURL); err == nil {
						tabMap[parsedURL.Host] = tabInfo{Name: name, ContentType: ct}
					}
				}
			}
			for _, v := range videos {
				if info, ok := tabMap[v.Source]; ok {
					if v.SourceTabName == "" {
						v.SourceTabName = info.Name
					}
					if v.ContentType == "" || v.ContentType == "regular" {
						if info.ContentType != "" {
							v.ContentType = info.ContentType
						}
					}
				}
			}
		}
	}

	return &ScrapedVideosResult{
		Videos:     videos,
		Total:      total,
		Page:       q.Page,
		Limit:      q.Limit,
		TotalPages: totalPages,
	}, nil
}

// GetScrapedVideosBySiteType retrieves all scraped videos for a specific site type
func (s *VideoScraperService) GetScrapedVideosBySiteType(siteType string) ([]*ScrapedVideo, error) {
	query := `
		SELECT
			id, source, site_type, COALESCE(content_type, 'regular'), external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at, created_at
		FROM scraped_videos
		WHERE site_type = ?
		ORDER BY discovered_at DESC
	`

	rows, err := s.db.Query(query, siteType)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	var videos []*ScrapedVideo
	for rows.Next() {
		video := &ScrapedVideo{}
		var tagsJSON string
		var externalID, description, thumbnailURL, thumbnailPath, videoPath sql.NullString
		var author, uploader, category, errorMessage sql.NullString
		var scrapedAt, downloadedAt sql.NullTime

		err := rows.Scan(
			&video.ID, &video.Source, &video.SiteType, &video.ContentType, &externalID, &video.Title, &video.URL, &description,
			&video.Duration, &video.FileSize, &thumbnailURL, &thumbnailPath, &videoPath,
			&author, &uploader, &video.ViewCount, &video.LikeCount, &video.Rating, &category, &tagsJSON,
			&video.Metadata, &video.Status, &errorMessage, &video.DiscoveredAt, &scrapedAt,
			&downloadedAt, &video.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}

		// Handle nullable fields
		if externalID.Valid {
			video.ExternalID = externalID.String
		}
		if description.Valid {
			video.Description = description.String
		}
		if thumbnailURL.Valid {
			video.ThumbnailURL = thumbnailURL.String
		}
		if thumbnailPath.Valid {
			video.ThumbnailPath = thumbnailPath.String
		}
		if videoPath.Valid {
			video.VideoPath = videoPath.String
		}
		if author.Valid {
			video.Author = author.String
		}
		if uploader.Valid {
			video.Uploader = uploader.String
		}
		if category.Valid {
			video.Category = category.String
		}
		if errorMessage.Valid {
			video.ErrorMessage = errorMessage.String
		}
		if scrapedAt.Valid {
			video.ScrapedAt = &scrapedAt.Time
		}
		if downloadedAt.Valid {
			video.DownloadedAt = &downloadedAt.Time
		}

		// Unmarshal tags
		if err := json.Unmarshal([]byte(tagsJSON), &video.Tags); err != nil {
			video.Tags = []string{}
		}

		// Filter out blocked content
		if IsVideoBlocked(video) {
			continue
		}

		videos = append(videos, video)
	}

	return videos, nil
}

// GetScrapedVideosBySource retrieves all scraped videos for a specific source (e.g., "sexuria", "pornhub")
func (s *VideoScraperService) GetScrapedVideosBySource(source string) ([]*ScrapedVideo, error) {
	query := `
		SELECT
			id, source, site_type, COALESCE(content_type, 'regular'), external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at, created_at
		FROM scraped_videos
		WHERE source = ?
		ORDER BY discovered_at DESC
	`

	rows, err := s.db.Query(query, source)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	var videos []*ScrapedVideo
	for rows.Next() {
		video := &ScrapedVideo{}
		var tagsJSON string
		var externalID, description, thumbnailURL, thumbnailPath, videoPath sql.NullString
		var author, uploader, category, errorMessage sql.NullString
		var scrapedAt, downloadedAt sql.NullTime

		err := rows.Scan(
			&video.ID, &video.Source, &video.SiteType, &video.ContentType, &externalID, &video.Title, &video.URL, &description,
			&video.Duration, &video.FileSize, &thumbnailURL, &thumbnailPath, &videoPath,
			&author, &uploader, &video.ViewCount, &video.LikeCount, &video.Rating, &category, &tagsJSON,
			&video.Metadata, &video.Status, &errorMessage, &video.DiscoveredAt, &scrapedAt,
			&downloadedAt, &video.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}

		// Handle nullable fields
		if externalID.Valid {
			video.ExternalID = externalID.String
		}
		if description.Valid {
			video.Description = description.String
		}
		if thumbnailURL.Valid {
			video.ThumbnailURL = thumbnailURL.String
		}
		if thumbnailPath.Valid {
			video.ThumbnailPath = thumbnailPath.String
		}
		if videoPath.Valid {
			video.VideoPath = videoPath.String
		}
		if author.Valid {
			video.Author = author.String
		}
		if uploader.Valid {
			video.Uploader = uploader.String
		}
		if category.Valid {
			video.Category = category.String
		}
		if errorMessage.Valid {
			video.ErrorMessage = errorMessage.String
		}
		if scrapedAt.Valid {
			video.ScrapedAt = &scrapedAt.Time
		}
		if downloadedAt.Valid {
			video.DownloadedAt = &downloadedAt.Time
		}

		// Unmarshal tags
		if err := json.Unmarshal([]byte(tagsJSON), &video.Tags); err != nil {
			video.Tags = []string{}
		}

		// Filter out blocked content
		if IsVideoBlocked(video) {
			continue
		}

		videos = append(videos, video)
	}

	return videos, nil
}

// GetScrapedVideosBySourceLike retrieves all scraped videos matching a source pattern (e.g., "sexuria%")
func (s *VideoScraperService) GetScrapedVideosBySourceLike(sourcePattern string) ([]*ScrapedVideo, error) {
	query := `
		SELECT
			id, source, site_type, COALESCE(content_type, 'regular'), external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at, created_at
		FROM scraped_videos
		WHERE source LIKE ?
		ORDER BY discovered_at DESC
	`

	rows, err := s.db.Query(query, sourcePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	var videos []*ScrapedVideo
	for rows.Next() {
		video := &ScrapedVideo{}
		var tagsJSON string
		var externalID, description, thumbnailURL, thumbnailPath, videoPath sql.NullString
		var author, uploader, category, errorMessage sql.NullString
		var scrapedAt, downloadedAt sql.NullTime

		err := rows.Scan(
			&video.ID, &video.Source, &video.SiteType, &video.ContentType, &externalID, &video.Title, &video.URL, &description,
			&video.Duration, &video.FileSize, &thumbnailURL, &thumbnailPath, &videoPath,
			&author, &uploader, &video.ViewCount, &video.LikeCount, &video.Rating, &category, &tagsJSON,
			&video.Metadata, &video.Status, &errorMessage, &video.DiscoveredAt, &scrapedAt,
			&downloadedAt, &video.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}

		// Handle nullable fields
		if externalID.Valid {
			video.ExternalID = externalID.String
		}
		if description.Valid {
			video.Description = description.String
		}
		if thumbnailURL.Valid {
			video.ThumbnailURL = thumbnailURL.String
		}
		if thumbnailPath.Valid {
			video.ThumbnailPath = thumbnailPath.String
		}
		if videoPath.Valid {
			video.VideoPath = videoPath.String
		}
		if author.Valid {
			video.Author = author.String
		}
		if uploader.Valid {
			video.Uploader = uploader.String
		}
		if category.Valid {
			video.Category = category.String
		}
		if errorMessage.Valid {
			video.ErrorMessage = errorMessage.String
		}
		if scrapedAt.Valid {
			video.ScrapedAt = &scrapedAt.Time
		}
		if downloadedAt.Valid {
			video.DownloadedAt = &downloadedAt.Time
		}

		// Unmarshal tags
		if err := json.Unmarshal([]byte(tagsJSON), &video.Tags); err != nil {
			video.Tags = []string{}
		}

		// Filter out blocked content
		if IsVideoBlocked(video) {
			continue
		}

		videos = append(videos, video)
	}

	return videos, nil
}

// GetScrapedVideoByID retrieves a single scraped video by ID
func (s *VideoScraperService) GetScrapedVideoByID(id int64) (*ScrapedVideo, error) {
	query := `
		SELECT
			id, source, site_type, COALESCE(content_type, 'regular'), external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at, created_at
		FROM scraped_videos
		WHERE id = ?
	`

	video := &ScrapedVideo{}
	var tagsJSON string
	var externalID, description, thumbnailURL, thumbnailPath, videoPath sql.NullString
	var author, uploader, category, errorMessage sql.NullString
	var scrapedAt, downloadedAt sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&video.ID, &video.Source, &video.SiteType, &video.ContentType, &externalID, &video.Title, &video.URL, &description,
		&video.Duration, &video.FileSize, &thumbnailURL, &thumbnailPath, &videoPath,
		&author, &uploader, &video.ViewCount, &video.LikeCount, &video.Rating, &category, &tagsJSON,
		&video.Metadata, &video.Status, &errorMessage, &video.DiscoveredAt, &scrapedAt,
		&downloadedAt, &video.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("video not found")
		}
		return nil, fmt.Errorf("failed to query video: %w", err)
	}

	// Handle nullable fields
	if externalID.Valid {
		video.ExternalID = externalID.String
	}
	if description.Valid {
		video.Description = description.String
	}
	if thumbnailURL.Valid {
		video.ThumbnailURL = thumbnailURL.String
	}
	if thumbnailPath.Valid {
		video.ThumbnailPath = thumbnailPath.String
	}
	if videoPath.Valid {
		video.VideoPath = videoPath.String
	}
	if author.Valid {
		video.Author = author.String
	}
	if uploader.Valid {
		video.Uploader = uploader.String
	}
	if category.Valid {
		video.Category = category.String
	}
	if errorMessage.Valid {
		video.ErrorMessage = errorMessage.String
	}
	if scrapedAt.Valid {
		video.ScrapedAt = &scrapedAt.Time
	}
	if downloadedAt.Valid {
		video.DownloadedAt = &downloadedAt.Time
	}

	// Unmarshal tags
	if err := json.Unmarshal([]byte(tagsJSON), &video.Tags); err != nil {
		video.Tags = []string{}
	}

	return video, nil
}

// MarkScrapedVideoDownloaded sets downloaded_at on a scraped video
func (s *VideoScraperService) MarkScrapedVideoDownloaded(id int64) error {
	_, err := s.db.Exec(
		`UPDATE scraped_videos SET downloaded_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	return err
}

// UnmarkScrapedVideoDownloaded clears downloaded_at on a scraped video
func (s *VideoScraperService) UnmarkScrapedVideoDownloaded(id int64) error {
	_, err := s.db.Exec(
		`UPDATE scraped_videos SET downloaded_at = NULL WHERE id = ?`,
		id,
	)
	return err
}

// GetScrapedVideosWithTabInfo retrieves all scraped videos with their source tab name
// This joins scraped_videos with scraper_tabs based on the source URL domain
func (s *VideoScraperService) GetScrapedVideosWithTabInfo() ([]*ScrapedVideo, error) {
	// Get all scraper tabs to map source domains to tab names
	tabsQuery := `SELECT base_url, name, content_type FROM scraper_tabs WHERE is_active = 1`
	tabRows, err := s.db.Query(tabsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query scraper tabs: %w", err)
	}
	defer tabRows.Close()

	// Build a map of domain -> tab info
	type tabInfo struct {
		Name        string
		ContentType string
	}
	tabMap := make(map[string]tabInfo)
	for tabRows.Next() {
		var baseURL, name, contentType string
		if err := tabRows.Scan(&baseURL, &name, &contentType); err != nil {
			continue
		}
		// Extract domain from base URL
		if parsedURL, err := url.Parse(baseURL); err == nil {
			domain := parsedURL.Host
			tabMap[domain] = tabInfo{Name: name, ContentType: contentType}
		}
	}

	// Get all scraped videos
	query := `
		SELECT
			id, source, site_type, COALESCE(content_type, 'regular'), external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at, created_at
		FROM scraped_videos
		ORDER BY discovered_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	var videos []*ScrapedVideo
	for rows.Next() {
		video := &ScrapedVideo{}
		var tagsJSON string
		var externalID, description, thumbnailURL, thumbnailPath, videoPath sql.NullString
		var author, uploader, category, errorMessage sql.NullString
		var scrapedAt, downloadedAt sql.NullTime

		err := rows.Scan(
			&video.ID, &video.Source, &video.SiteType, &video.ContentType, &externalID, &video.Title, &video.URL, &description,
			&video.Duration, &video.FileSize, &thumbnailURL, &thumbnailPath, &videoPath,
			&author, &uploader, &video.ViewCount, &video.LikeCount, &video.Rating, &category, &tagsJSON,
			&video.Metadata, &video.Status, &errorMessage, &video.DiscoveredAt, &scrapedAt,
			&downloadedAt, &video.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}

		// Handle nullable fields
		if externalID.Valid {
			video.ExternalID = externalID.String
		}
		if description.Valid {
			video.Description = description.String
		}
		if thumbnailURL.Valid {
			video.ThumbnailURL = thumbnailURL.String
		}
		if thumbnailPath.Valid {
			video.ThumbnailPath = thumbnailPath.String
		}
		if videoPath.Valid {
			video.VideoPath = videoPath.String
		}
		if author.Valid {
			video.Author = author.String
		}
		if uploader.Valid {
			video.Uploader = uploader.String
		}
		if category.Valid {
			video.Category = category.String
		}
		if errorMessage.Valid {
			video.ErrorMessage = errorMessage.String
		}
		if scrapedAt.Valid {
			video.ScrapedAt = &scrapedAt.Time
		}
		if downloadedAt.Valid {
			video.DownloadedAt = &downloadedAt.Time
		}

		// Unmarshal tags
		if err := json.Unmarshal([]byte(tagsJSON), &video.Tags); err != nil {
			video.Tags = []string{}
		}

		// Filter out blocked content
		if IsVideoBlocked(video) {
			continue
		}

		// Filter out Sexuria items - they should only be shown in the Sexuria section
		if video.SiteType == "sexuria" || video.Source == "sexuria.net" {
			continue
		}

		// Try to match the video's source to a scraper tab
		if info, exists := tabMap[video.Source]; exists {
			video.SourceTabName = info.Name
			// If the video doesn't have a content type set, use the tab's content type
			if video.ContentType == "" || video.ContentType == "regular" {
				video.ContentType = info.ContentType
			}
		}

		videos = append(videos, video)
	}

	return videos, nil
}

// CheckForNewVideos checks for new videos on a generic scraper tab since last checkpoint
func (s *VideoScraperService) CheckForNewVideos(scraperType string, baseURL string, maxPages int, activityID int) (*models.NewContentResult, error) {
	result := &models.NewContentResult{
		ScraperType: scraperType,
		CheckedAt:   time.Now(),
		Details:     map[string]interface{}{},
		NewItems:    []models.NewItemInfo{},
	}

	autoUpdateSvc := NewAutoUpdateService(nil, nil)

	// Get checkpoint for this tab
	checkpoint, _ := autoUpdateSvc.GetCheckpoint(scraperType, "last_video_url", "")
	lastSeenURL := ""
	if checkpoint != nil {
		lastSeenURL = checkpoint.CheckpointValue
	}

	log.Printf("Checking generic scraper %s for new videos (last seen: %s)", scraperType, lastSeenURL)

	// Parse the base URL to get the source
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	source := parsedURL.Host

	totalNewVideos := 0
	firstVideoURL := ""

	// Detect pagination style
	paginationStyle := "path"
	cleanBaseURL := strings.TrimRight(baseURL, "/")

	if strings.Contains(cleanBaseURL, "?page=") || strings.Contains(cleanBaseURL, "&page=") {
		paginationStyle = "query"
		if idx := strings.Index(cleanBaseURL, "?page="); idx != -1 {
			cleanBaseURL = cleanBaseURL[:idx]
		} else if idx := strings.Index(cleanBaseURL, "&page="); idx != -1 {
			cleanBaseURL = cleanBaseURL[:idx]
		}
	} else if strings.Contains(cleanBaseURL, "index.php") || strings.Contains(source, "fapzoo") {
		paginationStyle = "query"
	}

	// Check if URL ends with a page number
	hasPageNumber := false
	parts := strings.Split(cleanBaseURL, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if _, err := fmt.Sscanf(lastPart, "%d", new(int)); err == nil {
			cleanBaseURL = strings.Join(parts[:len(parts)-1], "/")
			hasPageNumber = true
		}
	}

	// Scrape up to maxPages looking for new videos
	for page := 1; page <= maxPages; page++ {
		var pageURL string
		if paginationStyle == "query" {
			if page == 1 {
				pageURL = cleanBaseURL
			} else {
				if strings.Contains(cleanBaseURL, "?") {
					pageURL = cleanBaseURL + "&page=" + fmt.Sprintf("%d", page)
				} else {
					pageURL = cleanBaseURL + "?page=" + fmt.Sprintf("%d", page)
				}
			}
		} else {
			if page == 1 && !hasPageNumber {
				pageURL = cleanBaseURL
			} else {
				pageURL = cleanBaseURL + "/" + fmt.Sprintf("%d", page)
			}
		}

		log.Printf("Checking page %d: %s", page, pageURL)

		videos, err := s.scrapeListingPage(pageURL, source, "custom")
		if err != nil {
			log.Printf("Error scraping page %d: %v", page, err)
			break
		}

		if len(videos) == 0 {
			log.Printf("No videos found on page %d, stopping", page)
			break
		}

		foundCheckpoint := false
		for _, video := range videos {
			// Save first video URL as new checkpoint
			if firstVideoURL == "" {
				firstVideoURL = video.URL
			}

			// If we hit the last seen video, stop
			if video.URL == lastSeenURL {
				foundCheckpoint = true
				log.Printf("Found checkpoint video, stopping at page %d", page)
				break
			}

			// Try to save the video
			saveErr := s.saveVideo(video)
			if saveErr == nil {
				// Genuine new insert — count it
				totalNewVideos++
				if len(result.NewItems) < 10 {
					result.NewItems = append(result.NewItems, models.NewItemInfo{
						URL:       video.URL,
						Title:     video.Title,
						Thumbnail: video.ThumbnailURL,
						Type:      "video",
						Duration:  video.Duration,
						Views:     video.ViewCount,
					})
				}
			} else if strings.Contains(saveErr.Error(), "Updated existing") {
				// Video already in DB — caught up to known content, stop scanning
				foundCheckpoint = true
				log.Printf("Hit existing video, stopping at page %d (caught up)", page)
				break
			}
		}

		if foundCheckpoint {
			break
		}

		time.Sleep(800 * time.Millisecond)
	}

	// Update checkpoint with newest video
	if firstVideoURL != "" {
		autoUpdateSvc.SaveCheckpoint(&models.ScraperCheckpoint{
			ScraperType:     scraperType,
			CheckpointType:  "last_video_url",
			CheckpointValue: firstVideoURL,
			TargetID:        "",
		})
	}

	result.NewItemsFound = totalNewVideos
	result.Details["total_new_videos"] = totalNewVideos
	result.Details["base_url"] = baseURL
	log.Printf("Found %d new videos from generic scraper %s", totalNewVideos, scraperType)

	return result, nil
}
