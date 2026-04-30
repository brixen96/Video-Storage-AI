package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/brixen96/video-storage-ai/internal/models"
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

	resp, err := http.Get(videoURL)
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
	resp, err := http.Get(videoURL)
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
	resp, err := http.Get(videoURL)
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

// ScrapePage scrapes a listing page for videos with pagination support
func (s *VideoScraperService) ScrapePage(req ScrapeVideoRequest) (*ScrapeResult, error) {
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	source := parsedURL.Host
	result := &ScrapeResult{
		Videos:       []*ScrapedVideo{},
		TotalScraped: 0,
		PagesCrawled: 0,
		Errors:       []string{},
	}

	maxPages := req.Options.MaxPages
	if maxPages <= 0 {
		maxPages = 9999 // 0 means unlimited, set to high number
	}

	baseURL := req.URL
	// Remove trailing slash for consistency
	baseURL = strings.TrimRight(baseURL, "/")

	// Detect pagination style:
	// 1. Query parameter pagination (e.g., ?page=2)
	// 2. Path-based pagination (e.g., /videos/2)
	paginationStyle := "path" // default to path-based
	hasPageNumber := false

	// Check if URL already has query parameter pagination
	if strings.Contains(baseURL, "?page=") || strings.Contains(baseURL, "&page=") {
		paginationStyle = "query"
		// Remove existing page parameter to get base URL
		if idx := strings.Index(baseURL, "?page="); idx != -1 {
			baseURL = baseURL[:idx]
		} else if idx := strings.Index(baseURL, "&page="); idx != -1 {
			baseURL = baseURL[:idx]
		}
	} else if strings.Contains(baseURL, "index.php") || strings.Contains(source, "fapzoo") {
		// Sites like fapzoo.fun use query parameter pagination
		paginationStyle = "query"
	} else {
		// Check if the URL already ends with a page number (like /extreme-test/1)
		parts := strings.Split(baseURL, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			// Check if last part is a number (page number)
			if _, err := fmt.Sscanf(lastPart, "%d", new(int)); err == nil {
				// Remove the page number from the base URL
				baseURL = strings.Join(parts[:len(parts)-1], "/")
				hasPageNumber = true
			}
		}
	}

	// Track seen URLs to detect when we're scraping the same page over and over.
	// Pre-populate from DB so re-scrapes stop early once they hit already-known pages.
	seenURLs := make(map[string]bool)
	s.preloadSeenURLs(seenURLs, source)
	consecutiveDuplicates := 0

	// Support resume from a specific page
	startPage := req.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	// Scrape pages
	for page := startPage; page <= maxPages; page++ {
		// Check if task has been paused
		if IsActivityPaused(req.ActivityID) {
			fmt.Printf("⏸️ Scraping paused by user at page %d\n", page)
			return result, fmt.Errorf("task paused by user at page %d", page)
		}
		var pageURL string
		if paginationStyle == "query" {
			// Query parameter pagination (e.g., ?page=2)
			if page == 1 {
				pageURL = baseURL
			} else {
				if strings.Contains(baseURL, "?") {
					pageURL = baseURL + "&page=" + fmt.Sprintf("%d", page)
				} else {
					pageURL = baseURL + "?page=" + fmt.Sprintf("%d", page)
				}
			}
		} else {
			// Path-based pagination (e.g., /videos/2)
			if page == 1 && !hasPageNumber {
				// First page without explicit page number in original URL
				pageURL = baseURL
			} else {
				// Construct page URL by appending page number to base URL
				pageURL = baseURL + "/" + fmt.Sprintf("%d", page)
			}
		}

		fmt.Printf("\n=== Scraping Page %d: %s ===\n", page, pageURL)
		videos, err := s.scrapeListingPage(pageURL, source, req.SiteType)
		if err != nil {
			fmt.Printf("ERROR scraping page %d: %v\n", page, err)
			result.Errors = append(result.Errors, fmt.Sprintf("Page %d: %v", page, err))
			// Don't break - try next page
			continue
		}

		fmt.Printf("Found %d videos on page %d\n", len(videos), page)

		if len(videos) == 0 {
			// No more videos found, stop pagination
			fmt.Printf("No videos found on page %d, stopping pagination\n", page)
			break
		}

		// Apply content_type from request to videos that don't have one set
		if req.ContentType != "" && req.ContentType != "regular" {
			for _, v := range videos {
				if v.ContentType == "" || v.ContentType == "regular" {
					v.ContentType = req.ContentType
				}
			}
		}

		// Check if all videos on this page are duplicates (already seen)
		allDuplicates := true
		for _, video := range videos {
			if !seenURLs[video.URL] {
				allDuplicates = false
				seenURLs[video.URL] = true
			}
		}

		// If all videos are duplicates, we're likely on the last page repeating
		if allDuplicates {
			consecutiveDuplicates++
			fmt.Printf("⚠️  All %d videos on page %d are duplicates (seen before)\n", len(videos), page)
			if consecutiveDuplicates >= 2 {
				fmt.Printf("🛑 Stopping pagination - same videos found on 2 consecutive pages (likely reached the end)\n")
				break
			}
		} else {
			consecutiveDuplicates = 0
		}

		result.Videos = append(result.Videos, videos...)
		result.PagesCrawled++
		result.TotalScraped += len(videos)

		// Small delay to be respectful
		time.Sleep(1 * time.Second)
	}

	return result, nil
}

// ScrapePageWithProgress scrapes a listing page with real-time progress updates
func (s *VideoScraperService) ScrapePageWithProgress(req ScrapeVideoRequest, progressCallback ProgressCallback) (*ScrapeResult, error) {
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	source := parsedURL.Host
	// Initialize result with counts from checkpoint (for resume)
	result := &ScrapeResult{
		Videos:       []*ScrapedVideo{},
		TotalScraped: req.InitialVideoCount, // Start from checkpoint count
		PagesCrawled: 0,
		Errors:       []string{},
	}

	maxPages := req.Options.MaxPages
	if maxPages <= 0 {
		maxPages = 9999 // 0 means unlimited, set to high number
	}

	baseURL := req.URL
	// Remove trailing slash for consistency
	baseURL = strings.TrimRight(baseURL, "/")

	// If an explicit page URL template was provided, skip auto-detection
	useTemplate := req.Options.PageUrlTemplate != ""
	templateStartsAt := req.Options.PageStartsAt
	if useTemplate && templateStartsAt <= 0 {
		templateStartsAt = 1
	}

	// Detect pagination style (only when no explicit template is provided):
	// 1. Query parameter pagination (e.g., ?page=2)
	// 2. Path-based pagination (e.g., /videos/2)
	paginationStyle := "path" // default to path-based
	hasPageNumber := false

	if !useTemplate {
		// Check if URL already has query parameter pagination
		if strings.Contains(baseURL, "?page=") || strings.Contains(baseURL, "&page=") {
			paginationStyle = "query"
			// Remove existing page parameter to get base URL
			if idx := strings.Index(baseURL, "?page="); idx != -1 {
				baseURL = baseURL[:idx]
			} else if idx := strings.Index(baseURL, "&page="); idx != -1 {
				baseURL = baseURL[:idx]
			}
		} else if strings.Contains(baseURL, "index.php") || strings.Contains(source, "fapzoo") {
			// Sites like fapzoo.fun use query parameter pagination
			paginationStyle = "query"
		} else {
			// Check if the URL already ends with a page number (like /extreme-test/1)
			parts := strings.Split(baseURL, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				// Check if last part is a number (page number)
				if _, err := fmt.Sscanf(lastPart, "%d", new(int)); err == nil {
					// Remove the page number from the base URL
					baseURL = strings.Join(parts[:len(parts)-1], "/")
					hasPageNumber = true
				}
			}
		}
	}

	// Track seen URLs to detect when we're scraping the same page over and over.
	// Pre-populate from DB so re-scrapes stop early once they hit already-known pages.
	seenURLs := make(map[string]bool)
	s.preloadSeenURLs(seenURLs, source)
	consecutiveDuplicates := 0

	// Support resume from a specific page
	startPage := req.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	// Scrape pages
	for page := startPage; page <= maxPages; page++ {
		// Check if task has been paused
		if IsActivityPaused(req.ActivityID) {
			fmt.Printf("⏸️ Scraping paused by user at page %d\n", page)
			return result, fmt.Errorf("task paused by user at page %d", page)
		}
		var pageURL string
		if useTemplate {
			// Template-based pagination: replace {page} with actual page number.
			// If templateStartsAt > 1, page 1 uses the raw base URL; later pages use the template.
			if page == 1 && templateStartsAt > 1 {
				pageURL = strings.TrimRight(req.URL, "/")
			} else {
				pageURL = strings.Replace(req.Options.PageUrlTemplate, "{page}", fmt.Sprintf("%d", page), 1)
			}
		} else if paginationStyle == "query" {
			// Query parameter pagination (e.g., ?page=2)
			if page == 1 {
				pageURL = baseURL
			} else {
				if strings.Contains(baseURL, "?") {
					pageURL = baseURL + "&page=" + fmt.Sprintf("%d", page)
				} else {
					pageURL = baseURL + "?page=" + fmt.Sprintf("%d", page)
				}
			}
		} else {
			// Path-based pagination (e.g., /videos/2)
			if page == 1 && !hasPageNumber {
				// First page without explicit page number in original URL
				pageURL = baseURL
			} else {
				// Construct page URL by appending page number to base URL
				pageURL = baseURL + "/" + fmt.Sprintf("%d", page)
			}
		}

		fmt.Printf("\n=== Scraping Page %d: %s ===\n", page, pageURL)
		videos, err := s.scrapeListingPage(pageURL, source, req.SiteType)
		if err != nil {
			fmt.Printf("ERROR scraping page %d: %v\n", page, err)
			result.Errors = append(result.Errors, fmt.Sprintf("Page %d: %v", page, err))
			// Don't break - try next page
			continue
		}

		fmt.Printf("Found %d videos on page %d\n", len(videos), page)

		if len(videos) == 0 {
			// No more videos found, stop pagination
			fmt.Printf("No videos found on page %d, stopping pagination\n", page)
			break
		}

		// Apply content_type from request to videos that don't have one set
		if req.ContentType != "" && req.ContentType != "regular" {
			for _, v := range videos {
				if v.ContentType == "" || v.ContentType == "regular" {
					v.ContentType = req.ContentType
				}
			}
		}

		// Check if all videos on this page are duplicates (already seen)
		allDuplicates := true
		for _, video := range videos {
			if !seenURLs[video.URL] {
				allDuplicates = false
				seenURLs[video.URL] = true
			}
		}

		// If all videos are duplicates, we're likely on the last page repeating
		if allDuplicates {
			consecutiveDuplicates++
			fmt.Printf("⚠️  All %d videos on page %d are duplicates (seen before)\n", len(videos), page)
			if consecutiveDuplicates >= 2 {
				fmt.Printf("🛑 Stopping pagination - same videos found on 2 consecutive pages (likely reached the end)\n")
				break
			}
		} else {
			consecutiveDuplicates = 0
		}

		result.Videos = append(result.Videos, videos...)
		result.PagesCrawled++
		result.TotalScraped += len(videos)

		// Call progress callback after each page
		if progressCallback != nil {
			progressCallback(ScrapeProgress{
				PagesCompleted: result.PagesCrawled,
				CurrentPage:    page, // Actual page number for checkpoint/resume
				VideosFound:    result.TotalScraped,
				VideosSaved:    result.TotalScraped,
				Errors:         result.Errors,
			})
		}

		// Small delay to be respectful
		time.Sleep(1 * time.Second)
	}

	return result, nil
}

// TestScrapeListingPage extracts video links from a listing page WITHOUT saving to database (for testing)
func (s *VideoScraperService) TestScrapeListingPage(pageURL string) (map[string]interface{}, error) {
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	source := parsedURL.Host

	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := map[string]interface{}{
		"url":            pageURL,
		"source":         source,
		"selectors_tried": []map[string]interface{}{},
		"videos_found":   []map[string]string{},
	}

	selectors := []string{
		// HQPorner-specific
		"section.box.feature a.image",
		"a[href*='/hdporn/']",
		// Most specific first - for sites with li.video structure
		"li.video > a.image",
		"li.video > span.title > a",
		// Specific video item containers
		".video-item a", ".video-card a", ".video-block a", "article.video a",
		".thumb a", ".video-thumb a", ".video-container a", ".item a", ".grid-item a",
		"a[href*='/video/']", "a[href*='/watch/']", "a[href*='/v/']",
		"ytd-video-renderer a", "div[class*='video'] a:has(img)", "div[class*='thumb'] a:has(img)",
	}

	foundLinks := make(map[string]bool)
	var videos []map[string]string

	for _, selector := range selectors {
		matches := doc.Find(selector)
		selectorInfo := map[string]interface{}{
			"selector": selector,
			"matches":  matches.Length(),
		}
		result["selectors_tried"] = append(result["selectors_tried"].([]map[string]interface{}), selectorInfo)

		if matches.Length() > 0 {
			fmt.Printf("Testing selector '%s': found %d matches\n", selector, matches.Length())

			matches.Each(func(i int, sel *goquery.Selection) {
				href, _ := sel.Attr("href")
				if href == "" {
					return
				}

				videoURL, _ := url.Parse(href)
				if !videoURL.IsAbs() {
					baseURL, _ := url.Parse(pageURL)
					videoURL = baseURL.ResolveReference(videoURL)
				}
				fullURL := videoURL.String()

				if foundLinks[fullURL] {
					fmt.Printf("  [TEST] Skipped (duplicate): %s\n", fullURL)
					return
				}

				// Skip if the URL is the same as the page we're scraping (listing page itself)
				if fullURL == pageURL || strings.TrimSuffix(fullURL, "/") == strings.TrimSuffix(pageURL, "/") {
					fmt.Printf("  [TEST] Skipped (same as page URL): %s\n", fullURL)
					return
				}

				// Skip URLs that look like listing/category pages (not individual videos)
				normalizedURL := strings.TrimSuffix(fullURL, "/")
				if strings.HasSuffix(normalizedURL, "/videos") ||
				   strings.Contains(normalizedURL, "/videos/page") ||
				   strings.Contains(normalizedURL, "/category") ||
				   strings.Contains(normalizedURL, "/categories") ||
				   strings.Contains(normalizedURL, "/tag") ||
				   strings.Contains(normalizedURL, "/tags") {
					fmt.Printf("  [TEST] Skipped (listing/category page): %s\n", fullURL)
					return
				}

				// Skip pagination URLs like /videos/2/, /videos/3/, etc.
				// Check if URL matches pattern: ends with /videos/{number}
				urlPath := videoURL.Path
				if strings.Contains(urlPath, "/videos/") {
					parts := strings.Split(strings.Trim(urlPath, "/"), "/")
					if len(parts) >= 2 {
						lastPart := parts[len(parts)-1]
						// If last part is a number, it's likely a pagination URL
						if _, err := fmt.Sscanf(lastPart, "%d", new(int)); err == nil && parts[len(parts)-2] == "videos" {
							fmt.Printf("  [TEST] Skipped (pagination URL): %s\n", fullURL)
							return
						}
					}
					// Also check if the path is exactly "/videos" or "/videos/"
					if urlPath == "/videos" || urlPath == "/videos/" {
						fmt.Printf("  [TEST] Skipped (base videos path): %s\n", fullURL)
						return
					}
				}

				foundLinks[fullURL] = true

				img := sel.Find("img").First()
				thumbnail := ""
				if img.Length() > 0 {
					for _, attr := range []string{"data-mediumthumb", "data-src", "data-original", "data-lazy-src", "data-lazy", "src"} {
						if v, ex := img.Attr(attr); ex && v != "" && !isPlaceholderURL(v) {
							thumbnail = v
							break
						}
					}
					if thumbnail != "" && !strings.HasPrefix(thumbnail, "http") {
						if turl, err := url.Parse(thumbnail); err == nil {
							pageBase, _ := url.Parse(pageURL)
							thumbnail = pageBase.ResolveReference(turl).String()
						}
					}
				}
				title := ""
				if img.Length() > 0 {
					title = img.AttrOr("alt", "")
				}
				if title == "" {
					title = sel.AttrOr("title", "")
				}
				// Walk up to 4 levels for sibling title containers (MadnessPorn: .title-thumb a, HQPorner: .meta-data-title a)
				if title == "" {
					for _, container := range []*goquery.Selection{sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
						if container.Length() == 0 || title != "" {
							break
						}
						if titleEl := container.Find(".title-thumb a, .meta-data-title a, .thumbnailTitle, .title > a, .video-title a, h3 > a, h4 > a"); titleEl.Length() > 0 {
							if t := strings.TrimSpace(titleEl.First().Text()); t != "" {
								title = t
							}
						}
					}
				}
				if title == "" {
					title = strings.TrimSpace(sel.Text())
				}
				// Duration: search up to 4 ancestor levels; use .Each() to skip empty elements
				duration := ""
				durationSels := "var.duration, .duration, .time, span.duration, [class*='duration'], [class*='runtime'], [class*='clock'], .fa-clock-o, [class*='meta-data']"
				for _, c := range []*goquery.Selection{sel, sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
					if c.Length() == 0 || duration != "" {
						break
					}
					c.Find(durationSels).Each(func(_ int, el *goquery.Selection) {
						if duration != "" {
							return
						}
						if d := strings.TrimSpace(el.Text()); parseDuration(d) > 0 {
							duration = d
						}
					})
					// MadnessPorn: span.left
					if duration == "" {
						c.Find("span.left, .left").Each(func(_ int, s *goquery.Selection) {
							if duration != "" {
								return
							}
							if t := strings.TrimSpace(s.Text()); parseDuration(t) > 0 {
								duration = t
							}
						})
					}
				}

				fmt.Printf("  [TEST] ✓ Accepted: %s (title: %s, thumb: %v)\n", fullURL, strings.TrimSpace(title), thumbnail != "")

				videos = append(videos, map[string]string{
					"url":       fullURL,
					"title":     strings.TrimSpace(title),
					"thumbnail": thumbnail,
					"duration":  duration,
				})
			})

			// Stop after first successful selector
			if len(videos) > 0 {
				break
			}
		}
	}

	// Broad fallback: if no videos found by specific selectors, try any <a> containing an <img>
	if len(videos) == 0 {
		fmt.Printf("[TEST] No specific selector matched, trying broad a:has(img) fallback...\n")
		fallbackMatches := doc.Find("a:has(img)")
		fallbackMatches.Each(func(i int, sel *goquery.Selection) {
			if sel.Closest("header, nav, footer, [class*='nav'], [class*='menu'], [class*='header'], [class*='footer'], [class*='sidebar'], [class*='banner'], [class*='logo']").Length() > 0 {
				return
			}
			href, exists := sel.Attr("href")
			if !exists || href == "" || href == "#" || strings.HasPrefix(href, "javascript:") {
				return
			}
			videoURL, _ := url.Parse(href)
			if !videoURL.IsAbs() {
				baseURL, _ := url.Parse(pageURL)
				videoURL = baseURL.ResolveReference(videoURL)
			}
			fullURL := videoURL.String()
			if foundLinks[fullURL] {
				return
			}
			if fullURL == pageURL || strings.TrimSuffix(fullURL, "/") == strings.TrimSuffix(pageURL, "/") {
				return
			}
			normalizedURL := strings.TrimSuffix(fullURL, "/")
			if strings.HasSuffix(normalizedURL, "/videos") ||
				strings.Contains(normalizedURL, "/category") ||
				strings.Contains(normalizedURL, "/categories") ||
				strings.Contains(normalizedURL, "/tag") ||
				strings.Contains(normalizedURL, "/tags") {
				return
			}
			urlPath := videoURL.Path
			if urlPath == "/" || urlPath == "" {
				return
			}
			if strings.Contains(urlPath, "/login") || strings.Contains(urlPath, "/register") ||
				strings.Contains(urlPath, "/signup") || strings.Contains(urlPath, "/search") ||
				strings.Contains(urlPath, "/about") || strings.Contains(urlPath, "/contact") ||
				strings.Contains(urlPath, "/privacy") || strings.Contains(urlPath, "/terms") {
				return
			}
			foundLinks[fullURL] = true
			thumbnail := ""
			title := ""
			img := sel.Find("img").First()
			if img.Length() > 0 {
				for _, attr := range []string{"data-mediumthumb", "data-src", "data-original", "data-lazy-src", "data-lazy", "src"} {
					if v, ex := img.Attr(attr); ex && v != "" && !isPlaceholderURL(v) {
						thumbnail = v
						break
					}
				}
				if thumbnail != "" && !strings.HasPrefix(thumbnail, "http") {
					if turl, err := url.Parse(thumbnail); err == nil {
						pageBase, _ := url.Parse(pageURL)
						thumbnail = pageBase.ResolveReference(turl).String()
					}
				}
				title, _ = img.Attr("alt")
			}
			if title == "" {
				title, _ = sel.Attr("title")
			}
			// Walk up to 4 levels for sibling title containers
			if title == "" {
				for _, container := range []*goquery.Selection{sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
					if container.Length() == 0 || title != "" {
						break
					}
					if titleEl := container.Find(".title-thumb a, .meta-data-title a, .thumbnailTitle, .title > a, .video-title a, h3 > a, h4 > a"); titleEl.Length() > 0 {
						if t := strings.TrimSpace(titleEl.First().Text()); t != "" {
							title = t
						}
					}
				}
			}
			if title == "" {
				title = strings.TrimSpace(sel.Text())
			}
			// Duration: search up to 4 ancestor levels; use .Each() to skip empty elements
			duration := ""
			durationSels2 := "var.duration, .duration, .time, span.duration, [class*='duration'], [class*='runtime'], [class*='clock'], .fa-clock-o, [class*='meta-data']"
			for _, c := range []*goquery.Selection{sel, sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
				if c.Length() == 0 || duration != "" {
					break
				}
				c.Find(durationSels2).Each(func(_ int, el *goquery.Selection) {
					if duration != "" {
						return
					}
					if d := strings.TrimSpace(el.Text()); parseDuration(d) > 0 {
						duration = d
					}
				})
				if duration == "" {
					c.Find("span.left, .left").Each(func(_ int, s *goquery.Selection) {
						if duration != "" {
							return
						}
						if t := strings.TrimSpace(s.Text()); parseDuration(t) > 0 {
							duration = t
						}
					})
				}
			}
			fmt.Printf("  [TEST fallback] ✓ Accepted: %s (title: %s, thumb: %v)\n", fullURL, strings.TrimSpace(title), thumbnail != "")
			videos = append(videos, map[string]string{
				"url":       fullURL,
				"title":     strings.TrimSpace(title),
				"thumbnail": thumbnail,
				"duration":  duration,
			})
		})
		if len(videos) > 0 {
			result["selectors_tried"] = append(result["selectors_tried"].([]map[string]interface{}), map[string]interface{}{
				"selector": "a:has(img) [broad fallback]",
				"matches":  len(videos),
			})
		}
	}

	result["videos_found"] = videos
	result["total_found"] = len(videos)
	return result, nil
}

// scrapeListingPage extracts video links from a listing page
func (s *VideoScraperService) scrapeListingPage(pageURL, source, siteType string) ([]*ScrapedVideo, error) {
	if siteType == "" {
		siteType = "custom"
	}
	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var videos []*ScrapedVideo

	// Look for common video listing patterns
	// Try multiple selectors to find video items
	selectors := []string{
		// PornHub-specific selectors
		"a.js-linkVideoThumb",
		".phimage a[href*='view_video']",
		"a[href*='viewkey=']",
		// FapZoo-specific selectors (video-card-wrapper with a.video-card)
		".video-card-wrapper a.video-card",
		".video-grid a.video-card",
		// HQPorner-specific: section.box.feature contains a.image.featured with img inside
		"section.box.feature a.image",
		"a[href*='/hdporn/']",
		// Most specific first - for sites with li.video structure
		"li.video > a.image",
		"li.video > span.title > a",
		// Specific video item containers
		".video-item a",
		".video-card a",
		".video-block a",
		"article.video a",
		".thumb a",
		".video-thumb a",
		".video-container a",
		".item a",
		".grid-item a",
		// Links with video-related hrefs (broader match)
		"a[href*='/video/']",
		"a[href*='/watch/']",
		"a[href*='/v/']",
		// YouTube-like structure
		"ytd-video-renderer a",
		// Generic containers with images (video thumbnails)
		"div[class*='video'] a:has(img)",
		"div[class*='thumb'] a:has(img)",
	}

	foundLinks := make(map[string]bool)
	newCount := 0
	updatedCount := 0
	errorCount := 0

	for _, selector := range selectors {
		matches := doc.Find(selector)
		if matches.Length() > 0 {
			fmt.Printf("Found %d matches with selector: %s\n", matches.Length(), selector)
		}

		matches.Each(func(i int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if !exists {
				return
			}

			// Convert relative URLs to absolute
			videoURL, err := url.Parse(href)
			if err != nil {
				return
			}
			if !videoURL.IsAbs() {
				baseURL, _ := url.Parse(pageURL)
				videoURL = baseURL.ResolveReference(videoURL)
			}

			fullURL := videoURL.String()

			// Skip duplicates
			if foundLinks[fullURL] {
				return
			}

			// Skip if the URL is the same as the page we're scraping (listing page itself)
			if fullURL == pageURL || strings.TrimSuffix(fullURL, "/") == strings.TrimSuffix(pageURL, "/") {
				fmt.Printf("  Skipped (same as page URL): %s\n", fullURL)
				return
			}

			// Skip URLs that look like listing/category pages (not individual videos)
			normalizedURL := strings.TrimSuffix(fullURL, "/")
			if strings.HasSuffix(normalizedURL, "/videos") ||
			   strings.Contains(normalizedURL, "/videos/page") ||
			   strings.Contains(normalizedURL, "/category") ||
			   strings.Contains(normalizedURL, "/categories") ||
			   strings.Contains(normalizedURL, "/tag") ||
			   strings.Contains(normalizedURL, "/tags") {
				fmt.Printf("  Skipped (listing/category page): %s\n", fullURL)
				return
			}

			// Skip pagination URLs like /videos/2/, /videos/3/, etc.
			// Check if URL matches pattern: ends with /videos/{number}
			urlPath := videoURL.Path
			if strings.Contains(urlPath, "/videos/") {
				parts := strings.Split(strings.Trim(urlPath, "/"), "/")
				if len(parts) >= 2 {
					lastPart := parts[len(parts)-1]
					// If last part is a number, it's likely a pagination URL
					if _, err := fmt.Sscanf(lastPart, "%d", new(int)); err == nil && parts[len(parts)-2] == "videos" {
						fmt.Printf("  Skipped (pagination URL): %s\n", fullURL)
						return
					}
				}
				// Also check if the path is exactly "/videos" or "/videos/"
				if urlPath == "/videos" || urlPath == "/videos/" {
					fmt.Printf("  Skipped (base videos path): %s\n", fullURL)
					return
				}
			}

			foundLinks[fullURL] = true

			// Extract thumbnail - try multiple attributes for lazy-loaded images
			thumbnail := ""
			videoPreviewURL := "" // For animated video previews (PornHub uses data-mediabook)
			thumbnailFrames := []string{} // Store all thumbnail frames for hover preview
			if img := sel.Find("img"); img.Length() > 0 {
				// Try PornHub's data-mediumthumb first (higher quality)
				if mediumThumb, exists := img.Attr("data-mediumthumb"); exists && mediumThumb != "" {
					thumbnail = mediumThumb
				}
				// Try various image source attributes — prioritize lazy-load attrs over src
				// (most sites use src=placeholder.gif and store the real thumbnail in data-src etc.)
				if thumbnail == "" {
					for _, attr := range []string{"data-src", "data-original", "data-lazy-src", "data-lazy", "data-srcset"} {
						if v, ex := img.Attr(attr); ex && v != "" && !isPlaceholderURL(v) {
							thumbnail = strings.Fields(v)[0] // take first URL if srcset
							break
						}
					}
				}
				// Fall back to src only if no lazy-load attr found
				if thumbnail == "" {
					if src, ex := img.Attr("src"); ex && src != "" && !isPlaceholderURL(src) {
						thumbnail = src
					}
				}

				// Extract video preview URL — check <a>, <img>, and sibling <video> elements
				previewAttrs := []string{"data-mediabook", "data-preview", "data-webm", "data-mp4",
					"data-video", "data-video-preview", "data-src-preview", "data-hover",
					"data-animated", "data-trailer", "data-preview-url", "data-videopreview"}
				for _, attr := range previewAttrs {
					if v, ex := sel.Attr(attr); ex && v != "" {
						videoPreviewURL = v
						break
					}
					if v, ex := img.Attr(attr); ex && v != "" {
						videoPreviewURL = v
						break
					}
				}
				// Also check any <video> element inside the link (e.g. FapZoo: <video data-preview-url="...">)
				if videoPreviewURL == "" {
					if vidEl := sel.Find("video[data-preview-url]"); vidEl.Length() > 0 {
						if v, ex := vidEl.Attr("data-preview-url"); ex && v != "" {
							videoPreviewURL = v
						}
					}
					if videoPreviewURL == "" {
						if vidEl := sel.Find("video.video-preview"); vidEl.Length() > 0 {
							if v, ex := vidEl.Attr("src"); ex && v != "" && !isPlaceholderURL(v) {
								videoPreviewURL = v
							}
						}
					}
				}
				// Make preview URL absolute if needed
				if videoPreviewURL != "" && !strings.HasPrefix(videoPreviewURL, "http") {
					if previewParsed, err := url.Parse(videoPreviewURL); err == nil {
						pageBase, _ := url.Parse(pageURL)
						videoPreviewURL = pageBase.ResolveReference(previewParsed).String()
					}
				}

				// Convert relative thumbnail URL to absolute URL
				if thumbnail != "" && !strings.HasPrefix(thumbnail, "http://") && !strings.HasPrefix(thumbnail, "https://") {
					thumbURL, err := url.Parse(thumbnail)
					if err == nil {
						pageBase, _ := url.Parse(pageURL)
						absoluteThumb := pageBase.ResolveReference(thumbURL)
						thumbnail = absoluteThumb.String()
					}
				}

				// Extract thumbnail frames for hover preview
				// Check for data-thumbs attribute (total number of frames)
				if thumbsCount, exists := img.Attr("data-thumbs"); exists && thumbnail != "" {
					var totalFrames int
					if _, err := fmt.Sscanf(thumbsCount, "%d", &totalFrames); err == nil && totalFrames > 1 {
						// Generate URLs for all frames
						// Pattern: /media/videos/tmb/000/010/964/1.jpg -> /media/videos/tmb/000/010/964/2.jpg, etc.
						baseThumbURL := thumbnail
						// Replace the frame number in the URL
						// Common patterns: /1.jpg, /2.jpg or _1.jpg, _2.jpg
						for i := 1; i <= totalFrames; i++ {
							frameURL := baseThumbURL
							// Try replacing /1.jpg, /1., _1.jpg, _1. patterns
							if strings.Contains(baseThumbURL, "/1.jpg") {
								frameURL = strings.Replace(baseThumbURL, "/1.jpg", fmt.Sprintf("/%d.jpg", i), 1)
							} else if strings.Contains(baseThumbURL, "/1.") {
								frameURL = strings.Replace(baseThumbURL, "/1.", fmt.Sprintf("/%d.", i), 1)
							} else if strings.Contains(baseThumbURL, "_1.jpg") {
								frameURL = strings.Replace(baseThumbURL, "_1.jpg", fmt.Sprintf("_%d.jpg", i), 1)
							} else if strings.Contains(baseThumbURL, "_1.") {
								frameURL = strings.Replace(baseThumbURL, "_1.", fmt.Sprintf("_%d.", i), 1)
							}
							thumbnailFrames = append(thumbnailFrames, frameURL)
						}
					}
				}
			}

			// MadnessPorn: extract animated preview frames from onmouseover="startm(id,base_url,.ext)"
			// onmouseover is on the <img> element, not the <a>, so check both sel and its img child
			if len(thumbnailFrames) == 0 {
				for _, attrEl := range []*goquery.Selection{sel, sel.Find("img").First()} {
					if attrEl.Length() == 0 {
						continue
					}
					if onover, ex := attrEl.Attr("onmouseover"); ex && strings.Contains(onover, "startm(") {
						qParts := strings.Split(onover, `"`)
						if len(qParts) >= 6 {
							baseURL := qParts[3]
							ext := strings.TrimSuffix(strings.TrimSuffix(qParts[5], ")"), ";")
							if strings.HasPrefix(baseURL, "http") {
								for i := 1; i <= 10; i++ {
									thumbnailFrames = append(thumbnailFrames, fmt.Sprintf("%s%d%s", baseURL, i, ext))
								}
								break
							}
						}
					}
				}
			}

			// HQPorner: thumbnail xxx_main.jpg -> frames xxx_1.jpg through xxx_10.jpg
			if len(thumbnailFrames) == 0 && strings.Contains(thumbnail, "_main.jpg") {
				base := strings.Replace(thumbnail, "_main.jpg", "", 1)
				for i := 1; i <= 10; i++ {
					thumbnailFrames = append(thumbnailFrames, fmt.Sprintf("%s_%d.jpg", base, i))
				}
			}

			// Extract title - try multiple sources
			title := ""
			// First try data-title attribute (PornHub uses this)
			if dataTitle, exists := sel.Attr("data-title"); exists && dataTitle != "" {
				title = dataTitle
			}
			// Try image data-title attribute (PornHub also uses this on img)
			if title == "" {
				if img := sel.Find("img"); img.Length() > 0 {
					if dataTitle, exists := img.Attr("data-title"); exists && dataTitle != "" {
						title = dataTitle
					}
				}
			}
			// Try image alt text (most reliable for video cards)
			if title == "" {
				if img := sel.Find("img"); img.Length() > 0 {
					if alt, exists := img.Attr("alt"); exists && alt != "" {
						title = alt
					}
				}
			}
			// Try aria-label on the link
			if title == "" {
				if ariaLabel, exists := sel.Attr("aria-label"); exists && ariaLabel != "" {
					title = ariaLabel
				}
			}
			// Try title attribute
			if title == "" {
				if titleAttr, exists := sel.Attr("title"); exists && titleAttr != "" {
					title = titleAttr
				}
			}
			// Try finding a title/heading element within parent containers
			if title == "" {
				// Walk up to 4 levels to find title in sibling containers (MadnessPorn: .title-thumb a, HQPorner: .meta-data-title a)
				for _, container := range []*goquery.Selection{sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
					if container.Length() == 0 || title != "" {
						break
					}
					if titleEl := container.Find(".title-thumb a, .meta-data-title a, .thumbnailTitle, .title > a, .video-title a, h3 > a, h4 > a"); titleEl.Length() > 0 {
						if t := strings.TrimSpace(titleEl.First().Text()); t != "" {
							title = t
						}
					}
				}
			}
			if title == "" {
				if h := sel.Find("h1, h2, h3, h4, h5, h6, .title, .video-title"); h.Length() > 0 {
					title = h.First().Text()
				}
			}
			// Last resort: use link text (often not great for video cards)
			if title == "" {
				title = strings.TrimSpace(sel.Text())
			}
			// Fallback
			if title == "" {
				title = "Video from " + source
			}

			fmt.Printf("  ✓ Accepted video URL: %s (title: %s)\n", fullURL, strings.TrimSpace(title))

			// Extract duration — search up to 4 ancestor levels with broad selectors
			// Use .Each() not .First() to skip empty elements (e.g. HQPorner has empty fa-circle-o.meta-data before fa-clock-o.meta-data)
			duration := 0
			durationSelectors := "var.duration, .duration, .time, span.duration, [class*='duration'], [class*='runtime'], [class*='clock'], .fa-clock-o, [class*='meta-data']"
			for _, container := range []*goquery.Selection{sel, sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
				if container.Length() == 0 || duration > 0 {
					break
				}
				container.Find(durationSelectors).Each(func(_ int, el *goquery.Selection) {
					if duration > 0 {
						return
					}
					if d := parseDuration(strings.TrimSpace(el.Text())); d > 0 {
						duration = d
					}
				})
				// MadnessPorn uses span.left for duration — only accept if text parses as a valid time
				if duration == 0 {
					container.Find("span.left, .left").Each(func(_ int, s *goquery.Selection) {
						if duration > 0 {
							return
						}
						if d := parseDuration(strings.TrimSpace(s.Text())); d > 0 {
							duration = d
						}
					})
				}
			}
			// Fallback: scan all leaf elements within the link for text that looks exactly
			// like a duration (e.g. "1:57 Min", "12:34"). Catches Tailwind-based sites
			// (chezcathy.com) where the badge has no class matching standard selectors.
			if duration == 0 {
				durationOnlyRe := regexp.MustCompile(`^\s*\d{1,2}:\d{2}(?::\d{2})?(?:\s*(?:[Mm]in|[Ss]ec))?\s*$`)
				sel.Find("*").Each(func(_ int, el *goquery.Selection) {
					if duration > 0 || el.Children().Length() > 0 {
						return // only leaf nodes
					}
					text := strings.TrimSpace(el.Text())
					if durationOnlyRe.MatchString(text) {
						if d := parseDuration(text); d > 0 {
							duration = d
						}
					}
				})
			}

			// Extract view count — search up to 4 ancestor levels + raw text scan
			viewCount := 0
			for _, container := range []*goquery.Selection{sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
				if container.Length() == 0 || viewCount > 0 {
					break
				}
				if viewsEl := container.Find(".views, span.views, [class*='views'], [class*='view-count']"); viewsEl.Length() > 0 {
					viewCount = parseViewCount(strings.TrimSpace(viewsEl.First().Text()))
				}
				// Raw text scan: MadnessPorn keeps "642 views" as a text node in div.text
				if viewCount == 0 {
					container.Find(".text, .info, .meta, .details, .data").Each(func(_ int, s *goquery.Selection) {
						if viewCount > 0 {
							return
						}
						words := strings.Fields(strings.ToLower(s.Text()))
						for i, w := range words {
							if (w == "views" || w == "view") && i > 0 {
								if v := parseViewCount(words[i-1] + " views"); v > 0 {
									viewCount = v
								}
							}
						}
					})
				}
			}

			// Extract rating — search up to 4 ancestor levels
			rating := 0
			for _, container := range []*goquery.Selection{sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
				if container.Length() == 0 || rating > 0 {
					break
				}
				container.Find(".rating, span.rating, [class*='rating'], .score, [class*='score']").Each(func(_ int, r *goquery.Selection) {
					if rating > 0 {
						return
					}
					if v := parseRating(strings.TrimSpace(r.Text())); v > 0 {
						rating = v
					}
				})
			}

			resolution := extractResolutionFromURL(fullURL)

			// Build metadata with resolution, thumbnail frames, and video preview if found
			metadata := "{}"
			metadataMap := make(map[string]interface{})

			if resolution != "" {
				metadataMap["max_resolution"] = resolution
				metadataMap["resolutions"] = []string{resolution}
			}

			if len(thumbnailFrames) > 0 {
				metadataMap["thumbnail_frames"] = thumbnailFrames
			}

			// Store video preview URL for animated hover preview (e.g., PornHub's webm previews)
			if videoPreviewURL != "" {
				metadataMap["video_preview_url"] = videoPreviewURL
			}

			if len(metadataMap) > 0 {
				if metadataJSON, err := json.Marshal(metadataMap); err == nil {
					metadata = string(metadataJSON)
				}
			}

			video := &ScrapedVideo{
				Source:       source,
				SiteType:     siteType,
				URL:          fullURL,
				Title:        strings.TrimSpace(title),
				ThumbnailURL: thumbnail,
				Duration:     duration,
				ViewCount:    viewCount,
				Rating:       rating,
				Status:       "discovered",
				DiscoveredAt: time.Now(),
				Tags:         []string{},
				Metadata:     metadata,
			}


			// Save to database
			saveErr := s.saveVideo(video)
			if saveErr != nil {
				// Check if it's an actual error or just an update
				if strings.Contains(saveErr.Error(), "Updated existing") {
					updatedCount++
				} else {
					errorCount++
					fmt.Printf("  ✗ Failed to save video: %v\n", saveErr)
				}
			} else {
				newCount++
			}
			// Always add to videos array (even if it already existed and was updated)
			videos = append(videos, video)
		})

		// If we found videos with this selector, stop trying others
		if len(videos) > 0 {
			break
		}
	}

	// Broad fallback: if no videos found by specific selectors, try any <a> containing an <img>
	// that is not inside a header/nav/footer element (catches sites like HQPorner with non-standard HTML)
	if len(videos) == 0 {
		fmt.Printf("No videos found with specific selectors, trying broad a:has(img) fallback...\n")
		fallbackMatches := doc.Find("a:has(img)")
		fallbackMatches.Each(func(i int, sel *goquery.Selection) {
			// Skip links inside nav/header/footer/menu elements
			if sel.Closest("header, nav, footer, [class*='nav'], [class*='menu'], [class*='header'], [class*='footer'], [class*='sidebar'], [class*='banner'], [class*='logo']").Length() > 0 {
				return
			}

			href, exists := sel.Attr("href")
			if !exists || href == "" || href == "#" || strings.HasPrefix(href, "javascript:") {
				return
			}

			videoURL, err := url.Parse(href)
			if err != nil {
				return
			}
			if !videoURL.IsAbs() {
				baseURL, _ := url.Parse(pageURL)
				videoURL = baseURL.ResolveReference(videoURL)
			}
			fullURL := videoURL.String()

			if foundLinks[fullURL] {
				return
			}
			if fullURL == pageURL || strings.TrimSuffix(fullURL, "/") == strings.TrimSuffix(pageURL, "/") {
				return
			}

			normalizedURL := strings.TrimSuffix(fullURL, "/")
			if strings.HasSuffix(normalizedURL, "/videos") ||
				strings.Contains(normalizedURL, "/videos/page") ||
				strings.Contains(normalizedURL, "/category") ||
				strings.Contains(normalizedURL, "/categories") ||
				strings.Contains(normalizedURL, "/tag") ||
				strings.Contains(normalizedURL, "/tags") {
				return
			}

			urlPath := videoURL.Path
			if urlPath == "/" || urlPath == "" {
				return
			}
			if strings.Contains(urlPath, "/login") || strings.Contains(urlPath, "/register") ||
				strings.Contains(urlPath, "/signup") || strings.Contains(urlPath, "/search") ||
				strings.Contains(urlPath, "/about") || strings.Contains(urlPath, "/contact") ||
				strings.Contains(urlPath, "/privacy") || strings.Contains(urlPath, "/terms") ||
				strings.HasPrefix(urlPath, "/cdn-cgi/") || strings.Contains(urlPath, ".css") ||
				strings.Contains(urlPath, ".js") {
				return
			}

			if strings.Contains(urlPath, "/videos/") {
				parts := strings.Split(strings.Trim(urlPath, "/"), "/")
				if len(parts) >= 2 {
					lastPart := parts[len(parts)-1]
					if _, err := fmt.Sscanf(lastPart, "%d", new(int)); err == nil && parts[len(parts)-2] == "videos" {
						return
					}
				}
			}

			foundLinks[fullURL] = true

			// --- Full metadata extraction (same quality as main selector loop) ---
			thumbnail := ""
			videoPreviewURL := ""
			img := sel.Find("img").First()
			if img.Length() > 0 {
				// Thumbnail: prioritize lazy-load attrs; src is last resort (often a placeholder)
				for _, attr := range []string{"data-mediumthumb", "data-src", "data-original", "data-lazy-src", "data-lazy", "data-srcset", "src"} {
					if v, ex := img.Attr(attr); ex && v != "" && !isPlaceholderURL(v) {
						thumbnail = strings.Fields(v)[0] // take first URL if srcset
						break
					}
				}
				// Animated preview URL — check <a>, <img>, and sibling <video> elements
				previewAttrs := []string{"data-mediabook", "data-preview", "data-webm", "data-mp4",
					"data-video", "data-video-preview", "data-src-preview", "data-hover",
					"data-animated", "data-trailer", "data-preview-url", "data-videopreview"}
				for _, attr := range previewAttrs {
					if v, ex := sel.Attr(attr); ex && v != "" {
						videoPreviewURL = v
						break
					}
					if v, ex := img.Attr(attr); ex && v != "" {
						videoPreviewURL = v
						break
					}
				}
				if videoPreviewURL == "" {
					if vidEl := sel.Find("video[data-preview-url]"); vidEl.Length() > 0 {
						if v, ex := vidEl.Attr("data-preview-url"); ex && v != "" {
							videoPreviewURL = v
						}
					}
					if videoPreviewURL == "" {
						if vidEl := sel.Find("video.video-preview"); vidEl.Length() > 0 {
							if v, ex := vidEl.Attr("src"); ex && v != "" && !isPlaceholderURL(v) {
								videoPreviewURL = v
							}
						}
					}
				}
				// Make URLs absolute
				if thumbnail != "" && !strings.HasPrefix(thumbnail, "http") {
					if turl, err := url.Parse(thumbnail); err == nil {
						pageBase, _ := url.Parse(pageURL)
						thumbnail = pageBase.ResolveReference(turl).String()
					}
				}
				if videoPreviewURL != "" && !strings.HasPrefix(videoPreviewURL, "http") {
					if purl, err := url.Parse(videoPreviewURL); err == nil {
						pageBase, _ := url.Parse(pageURL)
						videoPreviewURL = pageBase.ResolveReference(purl).String()
					}
				}
			}

			// Title: try various attribute sources first
			title := ""
			if v, ex := sel.Attr("data-title"); ex && v != "" {
				title = v
			}
			if title == "" {
				if v, ex := img.Attr("data-title"); ex && v != "" {
					title = v
				}
			}
			if title == "" {
				if v, ex := img.Attr("alt"); ex && v != "" {
					title = v
				}
			}
			if title == "" {
				if v, ex := sel.Attr("aria-label"); ex && v != "" {
					title = v
				}
			}
			if title == "" {
				if v, ex := sel.Attr("title"); ex && v != "" {
					title = v
				}
			}
			// Walk up to 4 levels to find title in sibling containers (MadnessPorn: .title-thumb a, HQPorner: .meta-data-title a)
			if title == "" {
				for _, container := range []*goquery.Selection{sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
					if container.Length() == 0 || title != "" {
						break
					}
					if titleEl := container.Find(".title-thumb a, .meta-data-title a, .thumbnailTitle, .title > a, .video-title a, h3 > a, h4 > a"); titleEl.Length() > 0 {
						if t := strings.TrimSpace(titleEl.First().Text()); t != "" {
							title = t
						}
					}
				}
			}
			if title == "" {
				if h := sel.Find("h1, h2, h3, h4, h5, h6, .title, .video-title"); h.Length() > 0 {
					title = strings.TrimSpace(h.First().Text())
				}
			}
			if title == "" {
				title = strings.TrimSpace(sel.Text())
			}
			if title == "" {
				title = "Video from " + source
			}

			// Duration: search up to 4 ancestor levels; use .Each() to skip empty elements
			duration := 0
			durationSelectors := "var.duration, .duration, .time, span.duration, [class*='duration'], [class*='runtime'], [class*='clock'], .fa-clock-o, [class*='meta-data']"
			for _, container := range []*goquery.Selection{sel, sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
				if container.Length() == 0 || duration > 0 {
					break
				}
				container.Find(durationSelectors).Each(func(_ int, el *goquery.Selection) {
					if duration > 0 {
						return
					}
					if d := parseDuration(strings.TrimSpace(el.Text())); d > 0 {
						duration = d
					}
				})
				// MadnessPorn uses span.left for duration — only accept if text parses as a valid time
				if duration == 0 {
					container.Find("span.left, .left").Each(func(_ int, s *goquery.Selection) {
						if duration > 0 {
							return
						}
						if d := parseDuration(strings.TrimSpace(s.Text())); d > 0 {
							duration = d
						}
					})
				}
			}

			// View count: search up to 4 ancestor levels + raw text scan
			viewCount := 0
			for _, container := range []*goquery.Selection{sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
				if container.Length() == 0 || viewCount > 0 {
					break
				}
				if viewsEl := container.Find(".views, span.views, [class*='views'], [class*='view-count']"); viewsEl.Length() > 0 {
					viewCount = parseViewCount(strings.TrimSpace(viewsEl.First().Text()))
				}
				// Raw text scan: MadnessPorn keeps "642 views" as a text node in div.text
				if viewCount == 0 {
					container.Find(".text, .info, .meta, .details, .data").Each(func(_ int, s *goquery.Selection) {
						if viewCount > 0 {
							return
						}
						words := strings.Fields(strings.ToLower(s.Text()))
						for i, w := range words {
							if (w == "views" || w == "view") && i > 0 {
								if v := parseViewCount(words[i-1] + " views"); v > 0 {
									viewCount = v
								}
							}
						}
					})
				}
			}

			// Rating: search up to 4 ancestor levels
			rating := 0
			for _, container := range []*goquery.Selection{sel.Parent(), sel.Parent().Parent(), sel.Parent().Parent().Parent(), sel.Parent().Parent().Parent().Parent()} {
				if container.Length() == 0 || rating > 0 {
					break
				}
				container.Find(".rating, span.rating, [class*='rating'], .score, [class*='score']").Each(func(_ int, r *goquery.Selection) {
					if rating > 0 {
						return
					}
					if v := parseRating(strings.TrimSpace(r.Text())); v > 0 {
						rating = v
					}
				})
			}

			// Animated preview frames (MadnessPorn startm / HQPorner _main.jpg)
			// MadnessPorn puts onmouseover on the <img>, so check both sel (<a>) and its img child
			thumbnailFrames := []string{}
			for _, attrEl := range []*goquery.Selection{sel, sel.Find("img").First()} {
				if len(thumbnailFrames) > 0 || attrEl.Length() == 0 {
					break
				}
				if onover, ex := attrEl.Attr("onmouseover"); ex && strings.Contains(onover, "startm(") {
					qParts := strings.Split(onover, `"`)
					if len(qParts) >= 6 {
						baseURL := qParts[3]
						ext := strings.TrimSuffix(strings.TrimSuffix(qParts[5], ")"), ";")
						if strings.HasPrefix(baseURL, "http") {
							for i := 1; i <= 10; i++ {
								thumbnailFrames = append(thumbnailFrames, fmt.Sprintf("%s%d%s", baseURL, i, ext))
							}
						}
					}
				}
			}
			if len(thumbnailFrames) == 0 && strings.Contains(thumbnail, "_main.jpg") {
				base := strings.Replace(thumbnail, "_main.jpg", "", 1)
				for i := 1; i <= 10; i++ {
					thumbnailFrames = append(thumbnailFrames, fmt.Sprintf("%s_%d.jpg", base, i))
				}
			}

			// Build metadata JSON
			resolution := extractResolutionFromURL(fullURL)
			metadataMap := make(map[string]interface{})
			if resolution != "" {
				metadataMap["max_resolution"] = resolution
				metadataMap["resolutions"] = []string{resolution}
			}
			if len(thumbnailFrames) > 0 {
				metadataMap["thumbnail_frames"] = thumbnailFrames
			}
			if videoPreviewURL != "" {
				metadataMap["video_preview_url"] = videoPreviewURL
			}
			metadata := "{}"
			if len(metadataMap) > 0 {
				if b, err := json.Marshal(metadataMap); err == nil {
					metadata = string(b)
				}
			}

			fmt.Printf("  [fallback] ✓ Accepted: %s (title: %s, duration: %d, preview: %v)\n",
				fullURL, strings.TrimSpace(title), duration, videoPreviewURL != "")

			video := &ScrapedVideo{
				Source:       source,
				SiteType:     siteType,
				URL:          fullURL,
				Title:        strings.TrimSpace(title),
				ThumbnailURL: thumbnail,
				Duration:     duration,
				ViewCount:    viewCount,
				Rating:       rating,
				Status:       "discovered",
				DiscoveredAt: time.Now(),
				Tags:         []string{},
				Metadata:     metadata,
			}


			saveErr := s.saveVideo(video)
			if saveErr != nil {
				if strings.Contains(saveErr.Error(), "Updated existing") {
					updatedCount++
				} else {
					errorCount++
				}
			} else {
				newCount++
			}
			videos = append(videos, video)
		})
		if len(videos) > 0 {
			fmt.Printf("  [fallback] Found %d video links\n", len(videos))
		}
	}

	// Print summary
	if len(videos) > 0 {
		fmt.Printf("\n📊 Summary: %d videos processed (%d new, %d updated, %d errors)\n",
			len(videos), newCount, updatedCount, errorCount)
	}

	return videos, nil
}

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
	SortBy          string   // Sort field (discovered_at, title, view_count, rating, duration)
	SortOrder       string   // Sort order (asc, desc)
	PerformerID     int64    // Filter to videos containing this performer (matched via metadata JSON)
	TagID           int64    // Filter to videos containing this tag (matched via metadata JSON)
	DurationFilter  string   // "short" (<5 min), "medium" (5-20 min), "long" (>20 min), or ""
	OnlyDownloaded  bool     // Only return videos that have been downloaded
	MinRating       int      // Minimum rating (0–100); 0 means no filter
	MinViews        int      // Minimum view count; 0 means no filter
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
		"rating":        true,
		"duration":      true,
		"created_at":    true,
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

// ScrapedDuplicateMatch represents a potential duplicate scraped video match
type ScrapedDuplicateMatch struct {
	Video1ID       int64   `json:"video1_id"`
	Video1Title    string  `json:"video1_title"`
	Video1URL      string  `json:"video1_url"`
	Video1Source   string  `json:"video1_source"`
	Video2ID       int64   `json:"video2_id"`
	Video2Title    string  `json:"video2_title"`
	Video2URL      string  `json:"video2_url"`
	Video2Source   string  `json:"video2_source"`
	MatchType      string  `json:"match_type"` // "exact_url", "similar_title", "external_id"
	Similarity     float64 `json:"similarity"` // 0.0 - 1.0
}

// ScrapedDuplicateStats contains statistics about duplicates
type ScrapedDuplicateStats struct {
	TotalDuplicates   int                      `json:"total_duplicates"`
	ByMatchType       map[string]int           `json:"by_match_type"`
	BySiteType        map[string]int           `json:"by_site_type"`
	Duplicates        []*ScrapedDuplicateMatch `json:"duplicates"`
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
		ID       int64
		Title    string
		URL      string
		Source   string
		SiteType string
		Duration int
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
				if v1.ViewCount > 0 { score1++ }
				if v2.ViewCount > 0 { score2++ }
				if v1.Rating > 0 { score1++ }
				if v2.Rating > 0 { score2++ }
				if v1.Duration > 0 { score1++ }
				if v2.Duration > 0 { score2++ }
				if v1.ThumbnailURL != "" { score1++ }
				if v2.ThumbnailURL != "" { score2++ }

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
