package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// HQPornerScraperService handles scraping from HQPorner.com
type HQPornerScraperService struct {
	db                  *sql.DB
	videoScraperService *VideoScraperService
}

// HQPornerScrapeRequest represents a scrape request for HQPorner
type HQPornerScrapeRequest struct {
	BaseURL           string `json:"baseUrl"`
	MaxPages          int    `json:"maxPages"`
	StartPage         int    `json:"startPage"`
	ActivityID        int64  `json:"-"`
	InitialVideoCount int    `json:"-"`
	InitialSavedCount int    `json:"-"`
}

// HQPornerScrapeResult contains the results of an HQPorner scrape
type HQPornerScrapeResult struct {
	BaseURL       string   `json:"baseUrl"`
	TotalScraped  int      `json:"totalScraped"`
	NewVideos     int      `json:"newVideos"`
	UpdatedVideos int      `json:"updatedVideos"`
	PagesCrawled  int      `json:"pagesCrawled"`
	Errors        []string `json:"errors"`
}

// NewHQPornerScraperService creates a new HQPorner scraper service
func NewHQPornerScraperService(db *sql.DB, videoScraperService *VideoScraperService) *HQPornerScraperService {
	return &HQPornerScraperService{
		db:                  db,
		videoScraperService: videoScraperService,
	}
}

// BuildHQPornerPageURL returns the URL for the given page of the HQPorner main listing.
// Page 1 is the root: https://hqporner.com/
// Page 2+ is:        https://hqporner.com/hdporn/N  (no trailing slash)
func BuildHQPornerPageURL(page int) string {
	if page <= 1 {
		return "https://hqporner.com/"
	}
	return "https://hqporner.com/hdporn/" + strconv.Itoa(page)
}

// ScrapeWithProgress scrapes HQPorner videos with progress callback
func (s *HQPornerScraperService) ScrapeWithProgress(req HQPornerScrapeRequest, progressCallback ProgressCallback) (*HQPornerScrapeResult, error) {
	if req.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	result := &HQPornerScrapeResult{
		BaseURL:       req.BaseURL,
		TotalScraped:  req.InitialVideoCount,
		NewVideos:     0,
		UpdatedVideos: req.InitialSavedCount,
		PagesCrawled:  0,
		Errors:        []string{},
	}

	maxPages := req.MaxPages
	if maxPages <= 0 {
		maxPages = 9999
	}

	startPage := req.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	consecutiveEmptyPages := 0
	consecutiveAllExisting := 0

	for page := startPage; page < startPage+maxPages; page++ {
		// Check if task has been paused
		if IsActivityPaused(req.ActivityID) {
			log.Printf("⏸️ HQPorner scrape paused by user at page %d", page)
			return result, fmt.Errorf("task paused by user at page %d", page)
		}

		pageURL := BuildHQPornerPageURL(page)
		log.Printf("\n=== Scraping HQPorner Page %d: %s ===", page, pageURL)

		videos, err := s.scrapeHQPornerListingPage(pageURL)
		if err != nil {
			errMsg := fmt.Sprintf("Error scraping page %d: %v", page, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("❌ %s", errMsg)
			consecutiveEmptyPages++
			if consecutiveEmptyPages >= 3 {
				log.Printf("🛑 Stopping - 3 consecutive errors/empty pages")
				break
			}
			continue
		}

		result.PagesCrawled++
		log.Printf("Found %d videos on page %d", len(videos), page)

		if len(videos) == 0 {
			consecutiveEmptyPages++
			log.Printf("⚠️ No videos found on page %d", page)
			if consecutiveEmptyPages >= 2 {
				log.Printf("🛑 Stopping pagination - no more videos")
				break
			}
			continue
		}

		consecutiveEmptyPages = 0

		// Check if all videos on this page already exist in the DB.
		// This lets re-runs stop quickly once they reach already-scraped content,
		// while fresh runs run to completion (nothing in DB = never stops early).
		allExisting := true
		for _, video := range videos {
			var count int
			s.db.QueryRow(`SELECT COUNT(*) FROM scraped_videos WHERE url = ? AND source = 'hqporner.com'`, video.URL).Scan(&count)
			if count == 0 {
				allExisting = false
				break
			}
		}
		if allExisting {
			consecutiveAllExisting++
			log.Printf("ℹ️ All videos on page %d already in DB (%d consecutive)", page, consecutiveAllExisting)
			if consecutiveAllExisting >= 2 {
				log.Printf("🛑 Stopping - reached already-scraped content")
				break
			}
		} else {
			consecutiveAllExisting = 0
		}

		// Concurrently enrich each video with per-video page data (performers, tags, author)
		const detailWorkers = 4
		sem := make(chan struct{}, detailWorkers)
		var wg sync.WaitGroup
		var mu sync.Mutex
		for _, video := range videos {
			video := video
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				if enrichErr := s.scrapeHQPornerVideoPage(video); enrichErr != nil {
					log.Printf("  ⚠️ [hqporner] failed to enrich %s: %v", video.URL, enrichErr)
				}
				mu.Lock()
				_ = video // keep reference alive
				mu.Unlock()
			}()
		}
		wg.Wait()

		// Save sequentially to avoid SQLite write contention
		for _, video := range videos {
			err := s.videoScraperService.saveVideo(video)
			if err != nil {
				if strings.Contains(err.Error(), "Updated existing") {
					result.UpdatedVideos++
				} else {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to save video: %v", err))
				}
			} else {
				result.NewVideos++
			}
			result.TotalScraped++
		}

		// Call progress callback
		if progressCallback != nil {
			progressCallback(ScrapeProgress{
				PagesCompleted: result.PagesCrawled,
				CurrentPage:    page,
				VideosFound:    result.TotalScraped,
				VideosSaved:    result.NewVideos + result.UpdatedVideos,
				Errors:         result.Errors,
			})
		}

		// Polite delay between pages
		time.Sleep(1500 * time.Millisecond)
	}

	log.Printf("\n=== HQPorner Scrape Complete ===")
	log.Printf("Pages crawled: %d", result.PagesCrawled)
	log.Printf("Total scraped: %d", result.TotalScraped)
	log.Printf("New videos: %d", result.NewVideos)
	log.Printf("Updated: %d", result.UpdatedVideos)
	log.Printf("Errors: %d", len(result.Errors))

	return result, nil
}

// scrapeHQPornerListingPage scrapes a single HQPorner listing page for video links
func (s *HQPornerScraperService) scrapeHQPornerListingPage(pageURL string) ([]*ScrapedVideo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	foundURLs := make(map[string]bool)
	var videos []*ScrapedVideo

	doc.Find("section.box.feature a.image").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists || href == "" {
			return
		}

		// Normalise to absolute URL
		if strings.HasPrefix(href, "//") {
			href = "https:" + href
		} else if !strings.HasPrefix(href, "http") {
			href = "https://www.hqporner.com" + href
		}
		// Only keep video pages (skip category/performer links)
		if !strings.Contains(href, "/hdporn/") {
			return
		}
		// Deduplicate
		if foundURLs[href] {
			return
		}
		foundURLs[href] = true

		// Extract thumbnail from <img> inside the link
		thumbnail := ""
		sel.Find("img").First().Each(func(_ int, img *goquery.Selection) {
			for _, attr := range []string{"data-src", "src"} {
				if v, ok := img.Attr(attr); ok && v != "" &&
					!strings.Contains(v, "blank.") &&
					!strings.Contains(v, "spacer.") &&
					!strings.Contains(v, "data:image") {
					thumbnail = v
					break
				}
			}
		})
		// Normalize protocol-relative thumbnail URLs
		if strings.HasPrefix(thumbnail, "//") {
			thumbnail = "https:" + thumbnail
		}

		// Title is in the sibling h3.meta-data-title inside the parent section
		section := sel.Closest("section.box.feature")
		title := strings.TrimSpace(section.Find("h3.meta-data-title a").First().Text())
		if title == "" {
			title = strings.TrimSpace(sel.AttrOr("title", ""))
		}
		if title == "" {
			// Derive from URL slug: /hdporn/12345-some-title.html → "some title"
			slug := href
			if idx := strings.LastIndex(slug, "/"); idx >= 0 {
				slug = slug[idx+1:]
			}
			slug = strings.TrimSuffix(slug, ".html")
			if dashIdx := strings.Index(slug, "-"); dashIdx >= 0 {
				slug = slug[dashIdx+1:]
			}
			title = strings.ReplaceAll(slug, "-", " ")
		}
		if title == "" {
			title = "HQPorner Video"
		}

		// Duration is in sibling span.fa-clock-o.meta-data
		duration := 0
		section.Find(".fa-clock-o.meta-data, span.icon.fa-clock-o").Each(func(_ int, span *goquery.Selection) {
			if duration > 0 {
				return
			}
			if d := parseDuration(strings.TrimSpace(span.Text())); d > 0 {
				duration = d
			}
		})

		// View count (not typically on listing page, but try)
		viewCount := 0

		// Generate animated thumbnail frames: _main.jpg → _1.jpg ... _10.jpg
		thumbnailFrames := []string{}
		if thumbnail != "" {
			for n := 1; n <= 10; n++ {
				frame := regexp.MustCompile(`_main\.(jpg|jpeg|png|webp)`).ReplaceAllString(
					thumbnail, "_"+strconv.Itoa(n)+".$1",
				)
				thumbnailFrames = append(thumbnailFrames, frame)
			}
		}

		// Build metadata
		metadataMap := map[string]interface{}{}
		if len(thumbnailFrames) > 0 {
			metadataMap["thumbnail_frames"] = thumbnailFrames
		}
		metadata := "{}"
		if len(metadataMap) > 0 {
			if b, err := json.Marshal(metadataMap); err == nil {
				metadata = string(b)
			}
		}

		video := &ScrapedVideo{
			Source:       "hqporner.com",
			SiteType:     "hqporner",
			ContentType:  "regular",
			URL:          href,
			Title:        title,
			ThumbnailURL: thumbnail,
			Duration:     duration,
			ViewCount:    viewCount,
			Status:       "discovered",
			DiscoveredAt: time.Now(),
			Tags:         []string{},
			Metadata:     metadata,
		}

		videos = append(videos, video)
	})

	return videos, nil
}

// scrapeHQPornerVideoPage visits an individual HQPorner video page to extract
// title, duration, performers, and categories; matches them against the DB.
func (s *HQPornerScraperService) scrapeHQPornerVideoPage(video *ScrapedVideo) error {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", video.URL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch video page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("parse HTML: %w", err)
	}

	// Title — prefer h1 over listing-page fallback
	if title := strings.TrimSpace(doc.Find("h1.main-h1").First().Text()); title != "" {
		video.Title = title
	}

	// Duration — skip empty icon spans
	if video.Duration == 0 {
		doc.Find("li.fa-clock-o, li.icon.fa-clock-o").Each(func(_ int, s *goquery.Selection) {
			if video.Duration > 0 {
				return
			}
			if d := parseDuration(strings.TrimSpace(s.Text())); d > 0 {
				video.Duration = d
			}
		})
	}

	// Performers from video page
	performerNames := []string{}
	doc.Find("li.fa-star-o a, li.icon.fa-star-o a").Each(func(_ int, s *goquery.Selection) {
		if name := strings.TrimSpace(s.Text()); name != "" {
			performerNames = append(performerNames, name)
		}
	})

	// Categories / tags from video page
	categoryNames := []string{}
	doc.Find("a.tag-link").Each(func(_ int, s *goquery.Selection) {
		if name := strings.TrimSpace(s.Text()); name != "" {
			categoryNames = append(categoryNames, name)
		}
	})

	// Match against DB
	matchedPerformers, matchedTags := s.matchPerformersAndTags(performerNames, categoryNames)

	// Build full performers list: all found names, with DB id if matched
	allPerformers := []map[string]interface{}{}
	for _, name := range performerNames {
		entry := map[string]interface{}{"id": int64(0), "name": name}
		for _, mp := range matchedPerformers {
			if strings.EqualFold(mp["name"].(string), name) {
				entry["id"] = mp["id"]
				entry["name"] = mp["name"] // use DB casing
				break
			}
		}
		allPerformers = append(allPerformers, entry)
	}

	// Store tags as video.Tags (string names) for searchability
	allTagNames := []string{}
	for _, name := range categoryNames {
		allTagNames = append(allTagNames, name)
	}
	if len(allTagNames) > 0 {
		video.Tags = allTagNames
	}

	// Store first performer as Author for filtering
	if len(performerNames) > 0 {
		video.Author = strings.Join(performerNames, ", ")
	}

	// Thumbnail fallback: if listing page didn't find one, try og:image from video page
	if video.ThumbnailURL == "" {
		if ogImage, exists := doc.Find("meta[property='og:image']").Attr("content"); exists && ogImage != "" {
			if strings.HasPrefix(ogImage, "//") {
				ogImage = "https:" + ogImage
			}
			video.ThumbnailURL = ogImage
		}
	}

	// Merge into existing metadata (thumbnail_frames from listing page)
	existingMeta := map[string]interface{}{}
	if video.Metadata != "" && video.Metadata != "{}" {
		json.Unmarshal([]byte(video.Metadata), &existingMeta)
	}
	if len(allPerformers) > 0 {
		existingMeta["performers"] = allPerformers
	}
	if len(matchedTags) > 0 {
		existingMeta["matched_tags"] = matchedTags
	}
	if b, err := json.Marshal(existingMeta); err == nil {
		video.Metadata = string(b)
	}

	return nil
}

// matchPerformersAndTags matches names against the DB (regular category, exact case-insensitive).
// Returns matched performers and matched tags with their DB IDs.
func (s *HQPornerScraperService) matchPerformersAndTags(
	performerNames []string,
	categoryNames []string,
) ([]map[string]interface{}, []map[string]interface{}) {
	matchedPerformers := []map[string]interface{}{}
	matchedTags := []map[string]interface{}{}

	for _, name := range performerNames {
		var id int64
		var dbName string
		err := s.db.QueryRow(
			`SELECT id, name FROM performers WHERE LOWER(name) = LOWER(?) AND category = 'regular' LIMIT 1`,
			name,
		).Scan(&id, &dbName)
		if err == nil {
			matchedPerformers = append(matchedPerformers, map[string]interface{}{
				"id":   id,
				"name": dbName,
			})
		}
	}

	for _, name := range categoryNames {
		var id int64
		var dbName, color, icon string
		err := s.db.QueryRow(
			`SELECT id, name, COALESCE(color, '#6c757d'), COALESCE(icon, '') FROM tags WHERE LOWER(name) = LOWER(?) AND category = 'regular' LIMIT 1`,
			name,
		).Scan(&id, &dbName, &color, &icon)
		if err == nil {
			matchedTags = append(matchedTags, map[string]interface{}{
				"id":    id,
				"name":  dbName,
				"color": color,
				"icon":  icon,
			})
		}
	}

	return matchedPerformers, matchedTags
}

// HQPornerVideoQuery holds filter/sort/page parameters for listing HQPorner videos
type HQPornerVideoQuery struct {
	Page            int
	Limit           int
	Search          string
	SortBy          string
	SortOrder       string
	PerformerID     int64
	TagID           int64
	DurationFilter  string   // "short" (<300s), "medium" (300-1200s), "long" (>1200s), or ""
	AuthorSearch    string   // LIKE search on author column
	IncludeKeywords []string // AND (tags LIKE each)
	ExcludeKeywords []string // AND NOT (tags LIKE each)
	MinRating       int      // Minimum rating (0–100); 0 means no filter
	MinViews        int      // Minimum view count; 0 means no filter
	MinDuration     float64  // Minimum duration in seconds; 0 means no filter
	MaxDuration     float64  // Maximum duration in seconds; 0 means no filter
}

// HQPornerVideoResult holds a page of HQPorner videos and the total count
type HQPornerVideoResult struct {
	Videos     []ScrapedVideo `json:"videos"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"totalPages"`
}

// GetVideosPaginated returns a filtered, sorted, paginated list of HQPorner videos.
// Supports performer and tag filtering via JSON extraction on the metadata column.
func (s *HQPornerScraperService) GetVideosPaginated(query HQPornerVideoQuery) (*HQPornerVideoResult, error) {
	if query.Limit <= 0 {
		query.Limit = 24
	}
	if query.Page <= 0 {
		query.Page = 1
	}

	where := `WHERE source = 'hqporner.com'`
	args := []interface{}{}

	if query.Search != "" {
		where += ` AND LOWER(title) LIKE LOWER(?)`
		args = append(args, "%"+query.Search+"%")
	}
	if query.PerformerID > 0 {
		where += ` AND EXISTS (
			SELECT 1 FROM json_each(json_extract(metadata, '$.performers')) p
			WHERE CAST(json_extract(p.value, '$.id') AS INTEGER) = ?
		)`
		args = append(args, query.PerformerID)
	}
	if query.TagID > 0 {
		where += ` AND EXISTS (
			SELECT 1 FROM json_each(json_extract(metadata, '$.matched_tags')) t
			WHERE CAST(json_extract(t.value, '$.id') AS INTEGER) = ?
		)`
		args = append(args, query.TagID)
	}
	switch query.DurationFilter {
	case "short":
		where += ` AND duration > 0 AND duration < 300`
	case "medium":
		where += ` AND duration >= 300 AND duration <= 1200`
	case "long":
		where += ` AND duration > 1200`
	}
	if query.AuthorSearch != "" {
		where += ` AND LOWER(COALESCE(author,'')) LIKE LOWER(?)`
		args = append(args, "%"+query.AuthorSearch+"%")
	}
	for _, kw := range query.IncludeKeywords {
		where += ` AND LOWER(COALESCE(tags,'')) LIKE LOWER(?)`
		args = append(args, "%"+kw+"%")
	}
	for _, kw := range query.ExcludeKeywords {
		where += ` AND LOWER(COALESCE(tags,'')) NOT LIKE LOWER(?)`
		args = append(args, "%"+kw+"%")
	}
	if query.MinRating > 0 {
		where += ` AND COALESCE(rating,0) >= ?`
		args = append(args, query.MinRating)
	}
	if query.MinViews > 0 {
		where += ` AND COALESCE(view_count,0) >= ?`
		args = append(args, query.MinViews)
	}
	if query.MinDuration > 0 {
		where += ` AND duration >= ?`
		args = append(args, query.MinDuration)
	}
	if query.MaxDuration > 0 {
		where += ` AND duration <= ?`
		args = append(args, query.MaxDuration)
	}

	// Total count
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM scraped_videos `+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Sort clause
	validSorts := map[string]string{
		"discovered_at": "discovered_at",
		"title":         "title",
		"duration":      "duration",
		"view_count":    "view_count",
		"rating":        "rating",
	}
	sortField, ok := validSorts[query.SortBy]
	if !ok {
		sortField = "discovered_at"
	}
	sortOrder := "DESC"
	if strings.ToUpper(query.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}
	orderClause := fmt.Sprintf("ORDER BY %s %s", sortField, sortOrder)

	offset := (query.Page - 1) * query.Limit
	pageArgs := append(args, query.Limit, offset)

	rows, err := s.db.Query(
		`SELECT id, url, title, COALESCE(thumbnail_url,''), COALESCE(duration,0), COALESCE(view_count,0),
		        COALESCE(rating,0), COALESCE(tags,'[]'), COALESCE(metadata,'{}'), discovered_at
		 FROM scraped_videos `+where+` `+orderClause+` LIMIT ? OFFSET ?`,
		pageArgs...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []ScrapedVideo{}
	for rows.Next() {
		var v ScrapedVideo
		var tagsJSON, metaJSON string
		var discoveredAt string
		if err := rows.Scan(&v.ID, &v.URL, &v.Title, &v.ThumbnailURL, &v.Duration,
			&v.ViewCount, &v.Rating, &tagsJSON, &metaJSON, &discoveredAt); err != nil {
			continue
		}
		v.Source = "hqporner.com"
		v.SiteType = "hqporner"
		v.Tags = []string{}
		json.Unmarshal([]byte(tagsJSON), &v.Tags)

		// Compute thumbnail_frames on-the-fly if not already in metadata.
		// Frame URLs are deterministic: replace _main.ext with _1.ext .. _10.ext.
		if !strings.Contains(metaJSON, "thumbnail_frames") && strings.Contains(v.ThumbnailURL, "_main.") {
			var meta map[string]interface{}
			if json.Unmarshal([]byte(metaJSON), &meta) != nil {
				meta = map[string]interface{}{}
			}
			re := regexp.MustCompile(`_main\.(jpg|jpeg|png|webp)`)
			frames := make([]string, 0, 10)
			for n := 1; n <= 10; n++ {
				frames = append(frames, re.ReplaceAllString(v.ThumbnailURL, "_"+strconv.Itoa(n)+".$1"))
			}
			meta["thumbnail_frames"] = frames
			if b, err := json.Marshal(meta); err == nil {
				metaJSON = string(b)
			}
		}
		v.Metadata = metaJSON
		if t, err := time.Parse(time.RFC3339, discoveredAt); err == nil {
			v.DiscoveredAt = t
		}
		videos = append(videos, v)
	}

	totalPages := (total + query.Limit - 1) / query.Limit
	return &HQPornerVideoResult{
		Videos:     videos,
		Total:      total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

// GetMatchedPerformers returns all unique DB-matched performers appearing in HQPorner videos.
func (s *HQPornerScraperService) GetMatchedPerformers() ([]map[string]interface{}, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT
			CAST(json_extract(p.value, '$.id') AS INTEGER) as pid,
			json_extract(p.value, '$.name') as pname
		FROM scraped_videos,
		     json_each(json_extract(metadata, '$.performers')) p
		WHERE source = 'hqporner.com'
		  AND json_extract(p.value, '$.id') > 0
		ORDER BY pname
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{"id": id, "name": name})
	}
	return result, nil
}

// GetMatchedTags returns all unique DB-matched tags appearing in HQPorner videos.
func (s *HQPornerScraperService) GetMatchedTags() ([]map[string]interface{}, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT
			CAST(json_extract(t.value, '$.id') AS INTEGER) as tid,
			json_extract(t.value, '$.name') as tname,
			COALESCE(json_extract(t.value, '$.color'), '#6c757d') as color,
			COALESCE(json_extract(t.value, '$.icon'), '') as icon
		FROM scraped_videos,
		     json_each(json_extract(metadata, '$.matched_tags')) t
		WHERE source = 'hqporner.com'
		  AND json_extract(t.value, '$.id') > 0
		ORDER BY tname
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var name, color, icon string
		if err := rows.Scan(&id, &name, &color, &icon); err != nil {
			continue
		}
		result = append(result, map[string]interface{}{"id": id, "name": name, "color": color, "icon": icon})
	}
	return result, nil
}

// GetHQPornerStats returns statistics about scraped HQPorner videos
func (s *HQPornerScraperService) GetHQPornerStats() (map[string]interface{}, error) {
	var totalVideos int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM scraped_videos WHERE source = 'hqporner.com'`).Scan(&totalVideos)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_videos": totalVideos,
	}, nil
}

// CheckForNewVideos checks for new HQPorner videos (used by auto-update system)
func (s *HQPornerScraperService) CheckForNewVideos(baseURL string, maxPages int, activityID int) (*models.NewContentResult, error) {
	result := &models.NewContentResult{
		ScraperType: "hqporner",
		CheckedAt:   time.Now(),
		Details:     map[string]interface{}{},
		NewItems:    []models.NewItemInfo{},
	}

	if baseURL == "" {
		baseURL = "https://www.hqporner.com/"
	}
	if maxPages <= 0 {
		maxPages = 3
	}

	for page := 1; page <= maxPages; page++ {
		pageURL := BuildHQPornerPageURL(page)
		videos, err := s.scrapeHQPornerListingPage(pageURL)
		if err != nil {
			log.Printf("Error checking HQPorner page %d: %v", page, err)
			continue
		}

		for _, video := range videos {
			var count int
			s.db.QueryRow(`SELECT COUNT(*) FROM scraped_videos WHERE url = ? AND source = 'hqporner.com'`, video.URL).Scan(&count)
			if count == 0 {
				if enrichErr := s.scrapeHQPornerVideoPage(video); enrichErr != nil {
					log.Printf("  Warning: failed to enrich new video %s: %v", video.URL, enrichErr)
				}
				if err := s.videoScraperService.saveVideo(video); err == nil {
					result.NewItemsFound++
					result.NewItems = append(result.NewItems, models.NewItemInfo{
						URL:       video.URL,
						Title:     video.Title,
						Thumbnail: video.ThumbnailURL,
						Type:      "video",
						Duration:  video.Duration,
					})
				}
			}
		}

		time.Sleep(1 * time.Second)
	}

	result.Details["pages_checked"] = maxPages
	result.Details["base_url"] = baseURL
	return result, nil
}
