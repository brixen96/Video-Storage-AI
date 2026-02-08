package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	DownloadVideo    bool `json:"downloadVideo"`
	ExtractMetadata  bool `json:"extractMetadata"`
	ScrapeThumbnails bool `json:"scrapeThumbnails"`
	MaxPages         int  `json:"maxPages"` // For pagination support (0 = unlimited, scrapes all pages)
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

	// Try to extract title from various sources
	if title, exists := doc.Find("meta[property='og:title']").Attr("content"); exists {
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

	// Track seen URLs to detect when we're scraping the same page over and over
	seenURLs := make(map[string]bool)
	consecutiveDuplicates := 0

	// Support resume from a specific page
	startPage := req.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	// Scrape pages
	for page := startPage; page <= maxPages; page++ {
		// Check if task has been paused
		if req.ActivityID > 0 {
			activitySvc := NewActivityService()
			currentActivity, err := activitySvc.GetByID(req.ActivityID)
			if err == nil && currentActivity.IsPaused {
				fmt.Printf("⏸️ Scraping paused by user at page %d\n", page)
				return result, fmt.Errorf("task paused by user at page %d", page)
			}
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
		videos, err := s.scrapeListingPage(pageURL, source)
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

	// Track seen URLs to detect when we're scraping the same page over and over
	seenURLs := make(map[string]bool)
	consecutiveDuplicates := 0

	// Support resume from a specific page
	startPage := req.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	// Scrape pages
	for page := startPage; page <= maxPages; page++ {
		// Check if task has been paused
		if req.ActivityID > 0 {
			activitySvc := NewActivityService()
			currentActivity, err := activitySvc.GetByID(req.ActivityID)
			if err == nil && currentActivity.IsPaused {
				fmt.Printf("⏸️ Scraping paused by user at page %d\n", page)
				return result, fmt.Errorf("task paused by user at page %d", page)
			}
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
		videos, err := s.scrapeListingPage(pageURL, source)
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

				title := sel.Find("img").AttrOr("alt", "")
				if title == "" {
					title = sel.AttrOr("title", "")
				}
				if title == "" {
					title = sel.Text()
				}

				fmt.Printf("  [TEST] ✓ Accepted: %s (title: %s)\n", fullURL, strings.TrimSpace(title))

				videos = append(videos, map[string]string{
					"url":   fullURL,
					"title": strings.TrimSpace(title),
				})
			})

			// Stop after first successful selector
			if len(videos) > 0 {
				break
			}
		}
	}

	result["videos_found"] = videos
	result["total_found"] = len(videos)
	return result, nil
}

// scrapeListingPage extracts video links from a listing page
func (s *VideoScraperService) scrapeListingPage(pageURL, source string) ([]*ScrapedVideo, error) {
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
				// Try various image source attributes
				if thumbnail == "" {
					if src, exists := img.Attr("src"); exists && src != "" && !strings.Contains(src, "placeholder") {
						thumbnail = src
					} else if dataSrc, exists := img.Attr("data-src"); exists && dataSrc != "" {
						thumbnail = dataSrc
					} else if dataOriginal, exists := img.Attr("data-original"); exists && dataOriginal != "" {
						thumbnail = dataOriginal
					} else if dataLazySrc, exists := img.Attr("data-lazy-src"); exists && dataLazySrc != "" {
						thumbnail = dataLazySrc
					}
				}

				// Extract video preview URL (PornHub uses data-mediabook for webm preview)
				if mediabook, exists := img.Attr("data-mediabook"); exists && mediabook != "" {
					videoPreviewURL = mediabook
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
			// Try finding a title/heading element within the selection or parent
			if title == "" {
				// Look in parent container for title elements (PornHub has .thumbnailTitle in sibling)
				if parent := sel.Parent().Parent(); parent.Length() > 0 {
					if titleEl := parent.Find(".thumbnailTitle, .title a, .video-title"); titleEl.Length() > 0 {
						title = strings.TrimSpace(titleEl.First().Text())
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

			// Extract duration if available
			duration := 0
			if durationEl := sel.Find(".duration, .time, [class*='duration']"); durationEl.Length() > 0 {
				durationText := strings.TrimSpace(durationEl.First().Text())
				if durationText != "" {
					duration = parseDuration(durationText)
				}
			}
			// Also try looking for duration in parent or sibling elements
			if duration == 0 {
				if parent := sel.Parent(); parent.Length() > 0 {
					// PornHub uses var.duration, also check standard duration classes
					if durationEl := parent.Find("var.duration, .duration, .time, span.duration"); durationEl.Length() > 0 {
						durationText := strings.TrimSpace(durationEl.First().Text())
						if durationText != "" {
							duration = parseDuration(durationText)
						}
					}
				}
			}
			// Also search in grandparent for PornHub's deeper nested structure
			if duration == 0 {
				if grandparent := sel.Parent().Parent(); grandparent.Length() > 0 {
					if durationEl := grandparent.Find("var.duration, .duration"); durationEl.Length() > 0 {
						durationText := strings.TrimSpace(durationEl.First().Text())
						if durationText != "" {
							duration = parseDuration(durationText)
						}
					}
				}
			}

			// Extract resolution if available (look for quality indicators or extract from URL)
			resolution := ""

			// Check if duration element contains HD/quality indicators (e.g., "<strong>HD</strong> 00:03")
			if durationEl := sel.Find(".duration, .time, [class*='duration']"); durationEl.Length() > 0 {
				// Check for <strong> tag inside duration
				if strongEl := durationEl.Find("strong"); strongEl.Length() > 0 {
					qualityText := strings.TrimSpace(strongEl.First().Text())
					resolution = extractResolutionFromText(qualityText)
				}
				// Also check the full duration text
				if resolution == "" {
					durationFullText := strings.TrimSpace(durationEl.First().Text())
					resolution = extractResolutionFromText(durationFullText)
				}
			}

			// Look for quality/resolution elements
			if resolution == "" {
				if qualityEl := sel.Find(".quality, .hd, [class*='quality'], [class*='resolution']"); qualityEl.Length() > 0 {
					qualityText := strings.TrimSpace(qualityEl.First().Text())
					resolution = extractResolutionFromText(qualityText)
				}
			}

			// Also try parent element
			if resolution == "" {
				if parent := sel.Parent(); parent.Length() > 0 {
					// Check duration in parent
					if durationEl := parent.Find(".duration strong"); durationEl.Length() > 0 {
						qualityText := strings.TrimSpace(durationEl.First().Text())
						resolution = extractResolutionFromText(qualityText)
					}
					// Check other quality elements in parent
					if resolution == "" {
						if qualityEl := parent.Find(".quality, .hd, span.quality"); qualityEl.Length() > 0 {
							qualityText := strings.TrimSpace(qualityEl.First().Text())
							resolution = extractResolutionFromText(qualityText)
						}
					}
				}
			}

			// Try to extract from video URL itself
			if resolution == "" {
				resolution = extractResolutionFromURL(fullURL)
			}

			// Extract view count from parent element
			viewCount := 0
			if parent := sel.Parent(); parent.Length() > 0 {
				// Look for views element: <span class="views">148164 views</span>
				if viewsEl := parent.Find(".views, span.views, [class*='views']"); viewsEl.Length() > 0 {
					viewsText := strings.TrimSpace(viewsEl.First().Text())
					viewCount = parseViewCount(viewsText)
				}
			}
			// Also search in grandparent for PornHub's structure
			if viewCount == 0 {
				if grandparent := sel.Parent().Parent(); grandparent.Length() > 0 {
					if viewsEl := grandparent.Find(".views var, span.views var, .views"); viewsEl.Length() > 0 {
						viewsText := strings.TrimSpace(viewsEl.First().Text())
						viewCount = parseViewCount(viewsText)
					}
				}
			}

			// Extract rating from parent element
			rating := 0
			if parent := sel.Parent(); parent.Length() > 0 {
				// Look for rating element: <span class="rating up"><i class="fa fa-thumbs-up"></i> 86%</span>
				if ratingEl := parent.Find(".rating, span.rating, [class*='rating']"); ratingEl.Length() > 0 {
					ratingText := strings.TrimSpace(ratingEl.First().Text())
					rating = parseRating(ratingText)
				}
			}

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
				SiteType:     "custom",
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
				rating = CASE WHEN ? > 0 THEN ? ELSE rating END,
				discovered_at = ?
			WHERE id = ?
		`
		_, err := s.db.Exec(
			updateQuery,
			video.Title,
			toNullString(video.ThumbnailURL),
			video.Duration, video.Duration,
			video.ViewCount, video.ViewCount,
			video.Rating, video.Rating,
			video.DiscoveredAt,
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

	// Default content_type to "regular" if not set
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
	Source      string // Filter by source (e.g., "www.pornhub.com", "luxuretv.com")
	SourceLike  string // Filter by source using LIKE (e.g., "sexuria%" to match both "sexuria" and "sexuria.net")
	SiteType    string // Filter by site_type (e.g., "pornhub", "luxuretv", "sexuria")
	ContentType string // Filter by content_type (e.g., "regular", "zoo")
	Keyword     string // Filter by tag/keyword
	Search      string // Search in title
	Page        int    // Page number (1-indexed)
	Limit       int    // Items per page
	SortBy      string // Sort field (discovered_at, title, view_count, rating, duration)
	SortOrder   string // Sort order (asc, desc)
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
	} else if q.Limit > 50000 {
		q.Limit = 50000 // Cap at 50000 for performance
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
	if q.Search != "" {
		conditions = append(conditions, "title LIKE ?")
		args = append(args, "%"+q.Search+"%")
	}
	if q.Keyword != "" {
		// Search for keyword in tags JSON array
		conditions = append(conditions, "tags LIKE ?")
		args = append(args, "%\""+q.Keyword+"\"%")
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

		// Filter out blocked content
		if IsVideoBlocked(video) {
			total-- // Adjust total count
			continue
		}

		videos = append(videos, video)
	}

	totalPages := (total + q.Limit - 1) / q.Limit

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

	parts := strings.Split(durationStr, ":")
	totalSeconds := 0

	switch len(parts) {
	case 1:
		// Just seconds: "45"
		if seconds, err := fmt.Sscanf(parts[0], "%d", new(int)); err == nil {
			totalSeconds = seconds
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

		videos, err := s.scrapeListingPage(pageURL, source)
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

			// This is a new video - save it
			saveErr := s.saveVideo(video)
			// saveVideo returns error for updates too, so we check if it's a real error
			if saveErr == nil || strings.HasPrefix(saveErr.Error(), "Updated existing video") {
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
			}
		}

		if foundCheckpoint {
			break
		}

		time.Sleep(800 * time.Millisecond)
	}

	// Update checkpoint with newest video
	if firstVideoURL != "" && totalNewVideos > 0 {
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
