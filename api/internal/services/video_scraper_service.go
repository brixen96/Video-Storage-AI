package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/brixen96/video-storage-ai/internal/httpclient"
)

// VideoScraperService handles scraping videos from various sites
type VideoScraperService struct {
	db *sql.DB
}

// GlobalBlockedContentTags contains tags/keywords that should be filtered out across all scrapers
var GlobalBlockedContentTags = []string{
	"transsexual",
	"shemale",
	"gay",
	"transgender",
	"solo",
	"ts",
	"tranny",
	"ladyboy",
	"femboy",
	"sissy",
	"trans",
}

// ContainsBlockedContent checks if text contains any blocked content keywords
func ContainsBlockedContent(text string) bool {
	lowerText := strings.ToLower(text)
	for _, blocked := range GlobalBlockedContentTags {
		// Check for whole word match or as part of compound words
		if strings.Contains(lowerText, blocked) {
			return true
		}
	}
	return false
}

// HasBlockedTags checks if any of the tags are in the blocked list
func HasBlockedTags(tags []string) bool {
	for _, tag := range tags {
		lowerTag := strings.ToLower(strings.TrimSpace(tag))
		for _, blocked := range GlobalBlockedContentTags {
			if lowerTag == blocked || strings.Contains(lowerTag, blocked) {
				return true
			}
		}
	}
	return false
}

// isPlaceholderURL returns true if the URL looks like a lazy-load placeholder
func isPlaceholderURL(u string) bool {
	lower := strings.ToLower(u)
	for _, ph := range []string{"placeholder", "blank.gif", "blank.png", "lazy.gif", "loading.gif", "transparent.gif", "spacer.gif", "no-image", "noimage", "default.jpg", "nophoto", "noimage.gif", "empty.gif"} {
		if strings.Contains(lower, ph) {
			return true
		}
	}
	return false
}

// IsVideoBlocked checks if a video should be filtered out based on title, tags, category
func IsVideoBlocked(video *ScrapedVideo) bool {
	if video == nil {
		return false
	}
	if ContainsBlockedContent(video.Title) {
		return true
	}
	if ContainsBlockedContent(video.Category) {
		return true
	}
	if ContainsBlockedContent(video.Description) {
		return true
	}
	if HasBlockedTags(video.Tags) {
		return true
	}
	return false
}

// NewVideoScraperService creates a new video scraper service
func NewVideoScraperService(db *sql.DB) *VideoScraperService {
	return &VideoScraperService{db: db}
}

// ScrapedVideo represents a scraped video
type ScrapedVideo struct {
	ID            int64      `json:"id"`
	Source        string     `json:"source"`
	SourceTabName string     `json:"source_tab_name,omitempty"` // Name of the scraper tab
	SiteType      string     `json:"site_type"`
	ContentType   string     `json:"content_type,omitempty"` // regular, zoo, 3d, etc.
	ExternalID    string     `json:"external_id,omitempty"`
	Title         string     `json:"title"`
	URL           string     `json:"url"`
	Description   string     `json:"description,omitempty"`
	Duration      int        `json:"duration"`
	FileSize      int64      `json:"file_size"`
	ThumbnailURL  string     `json:"thumbnail_url,omitempty"`
	ThumbnailPath string     `json:"thumbnail_path,omitempty"`
	VideoPath     string     `json:"video_path,omitempty"`
	Author        string     `json:"author,omitempty"`
	Uploader      string     `json:"uploader,omitempty"`
	ViewCount     int        `json:"view_count"`
	LikeCount     int        `json:"like_count"`
	Rating        int        `json:"rating"` // Rating percentage (0-100)
	Category      string     `json:"category,omitempty"`
	Tags          []string   `json:"tags"`
	Metadata      string     `json:"metadata"`
	Status        string     `json:"status"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	DiscoveredAt  time.Time  `json:"discovered_at"`
	ScrapedAt     *time.Time `json:"scraped_at,omitempty"`
	DownloadedAt  *time.Time `json:"downloaded_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ScrapeOptions contains scraping configuration
type ScrapeOptions struct {
	DownloadVideo    bool   `json:"downloadVideo"`
	ExtractMetadata  bool   `json:"extractMetadata"`
	ScrapeThumbnails bool   `json:"scrapeThumbnails"`
	MaxPages         int    `json:"maxPages"`        // 0 = unlimited, scrapes all pages
	PageUrlTemplate  string `json:"pageUrlTemplate"` // e.g. "https://site.com/page/{page}" — overrides auto-detection
	PageStartsAt     int    `json:"pageStartsAt"`    // First page number used in template (1 or 2; 2 means page 1 = base URL)
}

// ScrapeProgress represents the current progress of a scraping operation
type ScrapeProgress struct {
	PagesCompleted int      `json:"pagesCompleted"` // Count of pages scraped (for display)
	CurrentPage    int      `json:"currentPage"`    // Actual page number just completed (for checkpoint/resume)
	VideosFound    int      `json:"videosFound"`
	VideosSaved    int      `json:"videosSaved"`
	Errors         []string `json:"errors"`
}

// ProgressCallback is called after each page is scraped
type ProgressCallback func(progress ScrapeProgress)

// ScrapeVideoRequest represents a video scrape request
type ScrapeVideoRequest struct {
	URL               string        `json:"url"`
	SiteType          string        `json:"site_type"`
	ContentType       string        `json:"content_type"`      // Content type for all scraped videos (regular, zoo, 3d, etc.)
	Options           ScrapeOptions `json:"options"`
	StartPage         int           `json:"startPage"`         // Page to start from (for resume)
	ActivityID        int64         `json:"-"`                 // Activity ID for pause checking (not JSON serialized)
	InitialVideoCount int           `json:"-"`                 // Initial video count from checkpoint (for resume)
	InitialSavedCount int           `json:"-"`                 // Initial saved count from checkpoint (for resume)
}

// ScrapeResult contains the result of a scrape operation
type ScrapeResult struct {
	Videos       []*ScrapedVideo `json:"videos"`
	TotalScraped int             `json:"total_scraped"`
	PagesCrawled int             `json:"pages_crawled"`
	Errors       []string        `json:"errors,omitempty"`
}

// ScrapedVideosQuery represents query parameters for paginated scraped video retrieval
type ScrapedVideosQuery struct {
	Source          string   // Filter by source (e.g., "www.pornhub.com", "luxuretv.com")
	SourceLike      string   // Filter by source using LIKE (e.g., "sexuria%" to match both "sexuria" and "sexuria.net")
	SiteType        string   // Filter by site_type (e.g., "pornhub", "luxuretv", "sexuria")
	ContentType     string   // Filter by content_type (e.g., "regular", "zoo")
	Category        string   // Exact match on category column
	CategorySearch  string   // LIKE search on category column (partial match)
	Keyword         string   // Filter by tag/keyword (LIKE in tags)
	IncludeKeywords []string // All must be present in tags (AND conditions)
	ExcludeKeywords []string // None may be present in tags (AND NOT conditions)
	AuthorSearch    string   // LIKE search on author/artist column
	Search          string   // Search in title
	Page            int      // Page number (1-indexed)
	Limit           int      // Items per page
	SortBy          string   // Sort field (discovered_at, title, view_count, like_count, rating, duration, file_size)
	SortOrder       string   // Sort order (asc, desc)
	PerformerID     int64    // Filter to videos containing this performer (matched via metadata JSON)
	TagID           int64    // Filter to videos containing this tag (matched via metadata JSON)
	DurationFilter  string   // "short" (<5 min), "medium" (5-20 min), "long" (>20 min), or ""
	OnlyDownloaded  bool     // Only return videos that have been downloaded
	MinRating       int      // Minimum rating (0–100); 0 means no filter
	MinViews        int      // Minimum view count; 0 means no filter
	MinLikes        int      // Minimum like count; 0 means no filter
	DateFrom        string   // Minimum discovered_at date (YYYY-MM-DD)
	DateTo          string   // Maximum discovered_at date (YYYY-MM-DD)
	MinDuration     float64  // Minimum duration in seconds; 0 means no filter
	MaxDuration     float64  // Maximum duration in seconds; 0 means no filter
	MinFileSize     int64    // Minimum file size in bytes; 0 means no filter
	MaxFileSize     int64    // Maximum file size in bytes; 0 means no filter
	HasThumbnail    *bool    // Filter by thumbnail existence (true = has thumbnail, false = no thumbnail)
	OnlyWithAuthor  bool     // Only return videos that have an author
	OnlyWithRating  bool     // Only return videos that have a rating > 0
	IncludeTabInfo  bool     // Populate SourceTabName from scraper_tabs
}

// ScrapedVideosResult represents a paginated result of scraped videos
type ScrapedVideosResult struct {
	Videos     []*ScrapedVideo `json:"videos"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	TotalPages int             `json:"totalPages"`
}

// ScrapedDuplicateMatch represents a potential duplicate scraped video match
type ScrapedDuplicateMatch struct {
	Video1ID     int64   `json:"video1_id"`
	Video1Title  string  `json:"video1_title"`
	Video1URL    string  `json:"video1_url"`
	Video1Source string  `json:"video1_source"`
	Video2ID     int64   `json:"video2_id"`
	Video2Title  string  `json:"video2_title"`
	Video2URL    string  `json:"video2_url"`
	Video2Source string  `json:"video2_source"`
	MatchType    string  `json:"match_type"` // "exact_url", "similar_title", "external_id"
	Similarity   float64 `json:"similarity"` // 0.0 - 1.0
}

// ScrapedDuplicateStats contains statistics about duplicates
type ScrapedDuplicateStats struct {
	TotalDuplicates int                      `json:"total_duplicates"`
	ByMatchType     map[string]int           `json:"by_match_type"`
	BySiteType      map[string]int           `json:"by_site_type"`
	Duplicates      []*ScrapedDuplicateMatch `json:"duplicates"`
}

// ScrapeVideo scrapes a video from a URL
func (s *VideoScraperService) ScrapeVideo(req ScrapeVideoRequest) (*ScrapedVideo, error) {
	// Parse URL to determine source
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	source := parsedURL.Host
	siteType := req.SiteType
	if siteType == "generic" || siteType == "" {
		siteType = s.detectSiteType(source)
	}

	// Create initial video record
	video := &ScrapedVideo{
		Source:       source,
		SiteType:     siteType,
		URL:          req.URL,
		Status:       "scraping",
		DiscoveredAt: time.Now(),
		Tags:         []string{},
		Metadata:     "{}",
	}

	// Scrape based on site type
	switch siteType {
	case "youtube":
		if err := s.scrapeYouTube(video, req.URL); err != nil {
			video.Status = "failed"
			video.ErrorMessage = err.Error()
		}
	case "vimeo":
		if err := s.scrapeVimeo(video, req.URL); err != nil {
			video.Status = "failed"
			video.ErrorMessage = err.Error()
		}
	case "custom":
		if err := s.scrapeGenericSite(video, req.URL); err != nil {
			video.Status = "failed"
			video.ErrorMessage = err.Error()
		}
	default:
		if err := s.scrapeGenericSite(video, req.URL); err != nil {
			video.Status = "failed"
			video.ErrorMessage = err.Error()
		}
	}

	// Mark as completed if no error
	if video.Status != "failed" {
		video.Status = "completed"
		now := time.Now()
		video.ScrapedAt = &now
	}

	// Save to database
	if err := s.saveVideo(video); err != nil {
		return nil, fmt.Errorf("failed to save video: %w", err)
	}

	return video, nil
}

// detectSiteType detects the site type from the hostname
func (s *VideoScraperService) detectSiteType(host string) string {
	host = strings.ToLower(host)

	if strings.Contains(host, "youtube.com") || strings.Contains(host, "youtu.be") {
		return "youtube"
	}
	if strings.Contains(host, "vimeo.com") {
		return "vimeo"
	}

	return "custom"
}

// scrapeYouTube scrapes video metadata from YouTube
func (s *VideoScraperService) scrapeYouTube(video *ScrapedVideo, videoURL string) error {
	// For YouTube, we would typically use yt-dlp or YouTube API
	// For now, this is a placeholder that extracts basic metadata

	req, err := httpclient.NewRequest("GET", videoURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := httpclient.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch YouTube page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title from meta tags
	if title, exists := doc.Find("meta[property='og:title']").Attr("content"); exists {
		video.Title = title
	} else {
		video.Title = doc.Find("title").Text()
	}

	// Extract description
	if desc, exists := doc.Find("meta[property='og:description']").Attr("content"); exists {
		video.Description = desc
	}

	// Extract thumbnail
	if thumb, exists := doc.Find("meta[property='og:image']").Attr("content"); exists {
		video.ThumbnailURL = thumb
	}

	// Extract video ID
	if strings.Contains(videoURL, "v=") {
		parts := strings.Split(videoURL, "v=")
		if len(parts) > 1 {
			video.ExternalID = strings.Split(parts[1], "&")[0]
		}
	}

	// Extract channel name
	if channel, exists := doc.Find("link[itemprop='name']").Attr("content"); exists {
		video.Author = channel
	}

	return nil
}

// scrapeVimeo scrapes video metadata from Vimeo
func (s *VideoScraperService) scrapeVimeo(video *ScrapedVideo, videoURL string) error {
	req, err := httpclient.NewRequest("GET", videoURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := httpclient.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch Vimeo page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title
	if title, exists := doc.Find("meta[property='og:title']").Attr("content"); exists {
		video.Title = title
	}

	// Extract description
	if desc, exists := doc.Find("meta[property='og:description']").Attr("content"); exists {
		video.Description = desc
	}

	// Extract thumbnail
	if thumb, exists := doc.Find("meta[property='og:image']").Attr("content"); exists {
		video.ThumbnailURL = thumb
	}

	// Extract video ID from URL
	parts := strings.Split(videoURL, "/")
	if len(parts) > 0 {
		video.ExternalID = parts[len(parts)-1]
	}

	return nil
}

// scrapeGenericSite scrapes video metadata from a generic site
func (s *VideoScraperService) scrapeGenericSite(video *ScrapedVideo, videoURL string) error {
	req, err := httpclient.NewRequest("GET", videoURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := httpclient.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Try to extract title from various sources (site-specific first, then OG meta)
	if title := strings.TrimSpace(doc.Find("h1.main-h1").First().Text()); title != "" {
		video.Title = title // HQPorner
	} else if title, exists := doc.Find("meta[property='og:title']").Attr("content"); exists {
		video.Title = title
	} else if title, exists := doc.Find("meta[name='title']").Attr("content"); exists {
		video.Title = title
	} else {
		video.Title = doc.Find("title").Text()
	}

	// Extract description
	if desc, exists := doc.Find("meta[property='og:description']").Attr("content"); exists {
		video.Description = desc
	} else if desc, exists := doc.Find("meta[name='description']").Attr("content"); exists {
		video.Description = desc
	}

	// Extract thumbnail
	if thumb, exists := doc.Find("meta[property='og:image']").Attr("content"); exists {
		video.ThumbnailURL = thumb
	} else if thumb, exists := doc.Find("meta[property='og:image:url']").Attr("content"); exists {
		video.ThumbnailURL = thumb
	}

	// Extract duration from video page (site-specific selectors)
	// HQPorner: li.icon.fa-clock-o inside ul.meta; generic: [class*=duration], time[datetime]
	if video.Duration == 0 {
		durationCandidates := []string{
			"li.fa-clock-o", "li.icon.fa-clock-o",
			".duration", "[class*='duration']", "[class*='runtime']",
			"li.fa-clock-o", ".fa-clock-o",
			"time[datetime]",
		}
		for _, sel := range durationCandidates {
			doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
				if video.Duration > 0 {
					return
				}
				// For <time datetime="PT37M7S"> extract from attribute
				if dt, ex := s.Attr("datetime"); ex && dt != "" {
					if d := parseDuration(dt); d > 0 {
						video.Duration = d
						return
					}
				}
				if d := parseDuration(strings.TrimSpace(s.Text())); d > 0 {
					video.Duration = d
				}
			})
			if video.Duration > 0 {
				break
			}
		}
	}

	// Extract performers and categories as tags
	// HQPorner: li.icon.fa-star-o a = performers, a.tag-link = categories
	tags := []string{}
	doc.Find("li.fa-star-o a, li.icon.fa-star-o a").Each(func(_ int, s *goquery.Selection) {
		if name := strings.TrimSpace(s.Text()); name != "" {
			tags = append(tags, name)
		}
	})
	if len(tags) == 0 {
		// Generic performer selectors
		doc.Find(".pornstars a, .performers a, .models a, a[href*='/actress/'], a[href*='/pornstar/'], a[href*='/model/']").Each(func(_ int, s *goquery.Selection) {
			if name := strings.TrimSpace(s.Text()); name != "" && len(name) < 60 {
				tags = append(tags, name)
			}
		})
	}
	// Categories
	doc.Find("a.tag-link, a[href*='/category/'], a[href*='/categories/'], a[href*='/tag/']").Each(func(_ int, s *goquery.Selection) {
		if name := strings.TrimSpace(s.Text()); name != "" && len(name) < 40 {
			tags = append(tags, name)
		}
	})
	if len(tags) > 0 {
		video.Tags = tags
	}

	// Try to find video elements and extract resolution
	metadata := map[string]interface{}{}
	resolutions := []string{}

	doc.Find("video").Each(func(i int, s *goquery.Selection) {
		// Get main video src
		if src, exists := s.Attr("src"); exists {
			metadata["video_src"] = src
			// Try to extract resolution from filename (e.g., "video_720p.mp4")
			if res := extractResolutionFromURL(src); res != "" {
				resolutions = append(resolutions, res)
			}
		}

		// Get all source elements for different quality options
		sources := []map[string]string{}
		s.Find("source").Each(func(j int, source *goquery.Selection) {
			srcURL, _ := source.Attr("src")
			title, _ := source.Attr("title")
			srcType, _ := source.Attr("type")

			if srcURL != "" {
				sourceInfo := map[string]string{
					"url":  srcURL,
					"type": srcType,
				}
				if title != "" {
					sourceInfo["quality"] = title
					resolutions = append(resolutions, title)
				} else if res := extractResolutionFromURL(srcURL); res != "" {
					sourceInfo["quality"] = res
					resolutions = append(resolutions, res)
				}
				sources = append(sources, sourceInfo)
			}
		})

		if len(sources) > 0 {
			metadata["video_sources"] = sources
		}
	})

	// Store available resolutions
	if len(resolutions) > 0 {
		metadata["resolutions"] = resolutions
		// Store the highest resolution found
		metadata["max_resolution"] = findHighestResolution(resolutions)
	}

	// Merge with any pre-existing metadata (e.g., thumbnail_frames from listing page)
	if video.Metadata != "" {
		existingMeta := map[string]interface{}{}
		if err := json.Unmarshal([]byte(video.Metadata), &existingMeta); err == nil {
			for k, v := range existingMeta {
				if _, exists := metadata[k]; !exists {
					metadata[k] = v
				}
			}
		}
	}

	if len(metadata) > 0 {
		metadataJSON, _ := json.Marshal(metadata)
		video.Metadata = string(metadataJSON)
	}

	// If no title was found, use a default
	if video.Title == "" {
		video.Title = "Untitled Video from " + video.Source
	}

	return nil
}

// preloadSeenURLs seeds a seenURLs map with all URLs already in the DB for the
// given source hostname. This allows ScrapePage/ScrapePageWithProgress to stop
// early during a re-scrape once they hit pages whose videos are all known.
func (s *VideoScraperService) preloadSeenURLs(seenURLs map[string]bool, source string) {
	if source == "" {
		return
	}
	rows, err := s.db.Query("SELECT url FROM scraped_videos WHERE source = ?", source)
	if err != nil {
		return
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var u string
		if rows.Scan(&u) == nil {
			seenURLs[u] = true
			count++
		}
	}
	if count > 0 {
		log.Printf("[SCRAPER] Pre-loaded %d existing URLs for source '%s' — will stop early when re-scraping known pages", count, source)
	}
}
