package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// VideoScraperService handles scraping videos from various sites
type VideoScraperService struct {
	db *sql.DB
}

// NewVideoScraperService creates a new video scraper service
func NewVideoScraperService(db *sql.DB) *VideoScraperService {
	return &VideoScraperService{db: db}
}

// ScrapedVideo represents a scraped video
type ScrapedVideo struct {
	ID            int64     `json:"id"`
	Source        string    `json:"source"`
	SiteType      string    `json:"site_type"`
	ExternalID    string    `json:"external_id,omitempty"`
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	Description   string    `json:"description,omitempty"`
	Duration      int       `json:"duration"`
	FileSize      int64     `json:"file_size"`
	ThumbnailURL  string    `json:"thumbnail_url,omitempty"`
	ThumbnailPath string    `json:"thumbnail_path,omitempty"`
	VideoPath     string    `json:"video_path,omitempty"`
	Author        string    `json:"author,omitempty"`
	Uploader      string    `json:"uploader,omitempty"`
	ViewCount     int       `json:"view_count"`
	LikeCount     int       `json:"like_count"`
	Rating        int       `json:"rating"`           // Rating percentage (0-100)
	Category      string    `json:"category,omitempty"`
	Tags          []string  `json:"tags"`
	Metadata      string    `json:"metadata"`
	Status        string    `json:"status"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	DiscoveredAt  time.Time `json:"discovered_at"`
	ScrapedAt     *time.Time `json:"scraped_at,omitempty"`
	DownloadedAt  *time.Time `json:"downloaded_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
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
	PagesCompleted int      `json:"pagesCompleted"`
	VideosFound    int      `json:"videosFound"`
	VideosSaved    int      `json:"videosSaved"`
	Errors         []string `json:"errors"`
}

// ProgressCallback is called after each page is scraped
type ProgressCallback func(progress ScrapeProgress)

// ScrapeVideoRequest represents a video scrape request
type ScrapeVideoRequest struct {
	URL      string        `json:"url"`
	SiteType string        `json:"site_type"`
	Options  ScrapeOptions `json:"options"`
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
	// Remove trailing slash and page number if present
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/videos") {
		baseURL = baseURL + "/"
	}

	// Track seen URLs to detect when we're scraping the same page over and over
	seenURLs := make(map[string]bool)
	consecutiveDuplicates := 0

	// Scrape pages
	for page := 1; page <= maxPages; page++ {
		var pageURL string
		if page == 1 {
			// First page might not have /1/ suffix
			pageURL = baseURL
		} else {
			// Subsequent pages: /videos/2/, /videos/3/, etc.
			if strings.Contains(baseURL, "/videos") {
				pageURL = strings.TrimSuffix(baseURL, "/") + "/" + fmt.Sprintf("%d/", page)
			} else {
				pageURL = baseURL + "/videos/" + fmt.Sprintf("%d/", page)
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
	// Remove trailing slash and page number if present
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/videos") {
		baseURL = baseURL + "/"
	}

	// Track seen URLs to detect when we're scraping the same page over and over
	seenURLs := make(map[string]bool)
	consecutiveDuplicates := 0

	// Scrape pages
	for page := 1; page <= maxPages; page++ {
		var pageURL string
		if page == 1 {
			// First page might not have /1/ suffix
			pageURL = baseURL
		} else {
			// Subsequent pages: /videos/2/, /videos/3/, etc.
			if strings.Contains(baseURL, "/videos") {
				pageURL = strings.TrimSuffix(baseURL, "/") + "/" + fmt.Sprintf("%d/", page)
			} else {
				pageURL = baseURL + "/videos/" + fmt.Sprintf("%d/", page)
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
			if img := sel.Find("img"); img.Length() > 0 {
				// Try various image source attributes
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

			// Extract title - try multiple sources
			title := ""
			// First try image alt text (most reliable for video cards)
			if img := sel.Find("img"); img.Length() > 0 {
				if alt, exists := img.Attr("alt"); exists && alt != "" {
					title = alt
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
			// Try finding a title/heading element within the selection
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
					if durationEl := parent.Find(".duration, .time, span.duration"); durationEl.Length() > 0 {
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

			// Extract rating from parent element
			rating := 0
			if parent := sel.Parent(); parent.Length() > 0 {
				// Look for rating element: <span class="rating up"><i class="fa fa-thumbs-up"></i> 86%</span>
				if ratingEl := parent.Find(".rating, span.rating, [class*='rating']"); ratingEl.Length() > 0 {
					ratingText := strings.TrimSpace(ratingEl.First().Text())
					rating = parseRating(ratingText)
				}
			}

			// Build metadata with resolution if found
			metadata := "{}"
			if resolution != "" {
				metadataMap := map[string]interface{}{
					"max_resolution": resolution,
					"resolutions":    []string{resolution},
				}
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
			source, site_type, external_id, title, url, description,
			duration, file_size, thumbnail_url, thumbnail_path, video_path,
			author, uploader, view_count, like_count, rating, category, tags,
			metadata, status, error_message, discovered_at, scraped_at,
			downloaded_at
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?
		)
	`

	result, err := s.db.Exec(
		query,
		video.Source, video.SiteType, toNullString(video.ExternalID), video.Title, video.URL, toNullString(video.Description),
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
			id, source, site_type, external_id, title, url, description,
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
			&video.ID, &video.Source, &video.SiteType, &externalID, &video.Title, &video.URL, &description,
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

		videos = append(videos, video)
	}

	return videos, nil
}

// GetScrapedVideoByID retrieves a single scraped video by ID
func (s *VideoScraperService) GetScrapedVideoByID(id int64) (*ScrapedVideo, error) {
	query := `
		SELECT
			id, source, site_type, external_id, title, url, description,
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
		&video.ID, &video.Source, &video.SiteType, &externalID, &video.Title, &video.URL, &description,
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
