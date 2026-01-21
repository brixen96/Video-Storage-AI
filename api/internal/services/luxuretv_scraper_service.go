package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// LuxureTVScraperService handles scraping from LuxureTV
type LuxureTVScraperService struct {
	db                  *sql.DB
	videoScraperService *VideoScraperService
}

// LuxureTVScrapeRequest represents a scrape request for LuxureTV
type LuxureTVScrapeRequest struct {
	CategoryURL       string `json:"categoryUrl"`       // e.g., https://en.luxuretv.com/channels/76/zoophilia/
	MaxPages          int    `json:"maxPages"`          // 0 = unlimited
	StartPage         int    `json:"startPage"`         // Page to start from (for resume)
	ActivityID        int64  `json:"-"`                 // Activity ID for pause checking (not JSON serialized)
	InitialVideoCount int    `json:"-"`                 // Initial video count from checkpoint (for resume)
	InitialSavedCount int    `json:"-"`                 // Initial saved count from checkpoint (for resume)
}

// LuxureTVScrapeResult contains the results of a LuxureTV scrape
type LuxureTVScrapeResult struct {
	CategoryURL   string   `json:"categoryUrl"`
	TotalScraped  int      `json:"totalScraped"`
	NewVideos     int      `json:"newVideos"`
	UpdatedVideos int      `json:"updatedVideos"`
	PagesCrawled  int      `json:"pagesCrawled"`
	Errors        []string `json:"errors"`
}

// NewLuxureTVScraperService creates a new LuxureTV scraper service
func NewLuxureTVScraperService(db *sql.DB, videoScraperService *VideoScraperService) *LuxureTVScraperService {
	return &LuxureTVScraperService{
		db:                  db,
		videoScraperService: videoScraperService,
	}
}

// ScrapeCategory scrapes LuxureTV videos from a category page
func (s *LuxureTVScraperService) ScrapeCategory(req LuxureTVScrapeRequest) (*LuxureTVScrapeResult, error) {
	return s.ScrapeCategoryWithProgress(req, nil)
}

// ScrapeCategoryWithProgress scrapes LuxureTV videos with progress callback
func (s *LuxureTVScraperService) ScrapeCategoryWithProgress(req LuxureTVScrapeRequest, progressCallback ProgressCallback) (*LuxureTVScrapeResult, error) {
	if req.CategoryURL == "" {
		return nil, fmt.Errorf("category URL is required")
	}

	// Normalize the base URL (remove trailing page info)
	baseURL := normalizeBaseURL(req.CategoryURL)

	// Initialize result with counts from checkpoint (for resume)
	result := &LuxureTVScrapeResult{
		CategoryURL:   baseURL,
		TotalScraped:  req.InitialVideoCount, // Start from checkpoint count
		NewVideos:     0,
		UpdatedVideos: req.InitialSavedCount, // Approximate - assume all previous were updates
		PagesCrawled:  0,
		Errors:        []string{},
	}

	maxPages := req.MaxPages
	if maxPages <= 0 {
		maxPages = 9999 // Unlimited
	}

	// Support resume from a specific page
	startPage := req.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	consecutiveEmptyPages := 0

	// Scrape pages
	for page := startPage; page <= maxPages; page++ {
		// Check if task has been paused
		if req.ActivityID > 0 {
			activitySvc := NewActivityService()
			currentActivity, err := activitySvc.GetByID(req.ActivityID)
			if err == nil && currentActivity.IsPaused {
				fmt.Printf("⏸️ LuxureTV scrape paused by user at page %d\n", page)
				return result, fmt.Errorf("task paused by user at page %d", page)
			}
		}

		pageURL := buildLuxureTVPageURL(baseURL, page)

		fmt.Printf("\n=== Scraping LuxureTV Page %d: %s ===\n", page, pageURL)

		videos, err := s.scrapeLuxureTVPage(pageURL)
		if err != nil {
			errMsg := fmt.Sprintf("Error scraping page %d: %v", page, err)
			result.Errors = append(result.Errors, errMsg)
			fmt.Printf("❌ %s\n", errMsg)
			consecutiveEmptyPages++
			if consecutiveEmptyPages >= 5 {
				fmt.Printf("🛑 Stopping - 5 consecutive errors/empty pages\n")
				break
			}
			continue
		}

		result.PagesCrawled++
		fmt.Printf("Found %d videos on page %d\n", len(videos), page)

		if len(videos) == 0 {
			consecutiveEmptyPages++
			fmt.Printf("⚠️  No videos found on page %d\n", page)
			if consecutiveEmptyPages >= 5 {
				fmt.Printf("🛑 Stopping pagination - 5 consecutive empty pages\n")
				break
			}
			continue
		}

		consecutiveEmptyPages = 0

		// Save videos
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
				CurrentPage:    page, // Actual page number for checkpoint/resume
				VideosFound:    result.TotalScraped,
				VideosSaved:    result.NewVideos + result.UpdatedVideos,
				Errors:         result.Errors,
			})
		}

		// Small delay between pages to be polite
		// Delay between pages to avoid rate limiting
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("\n=== LuxureTV Scrape Complete ===\n")
	fmt.Printf("Category: %s\n", baseURL)
	fmt.Printf("Pages crawled: %d\n", result.PagesCrawled)
	fmt.Printf("Total scraped: %d\n", result.TotalScraped)
	fmt.Printf("New videos: %d\n", result.NewVideos)
	fmt.Printf("Updated videos: %d\n", result.UpdatedVideos)
	fmt.Printf("Errors: %d\n", len(result.Errors))

	return result, nil
}

// normalizeBaseURL removes page info from the URL to get the base category URL
func normalizeBaseURL(urlStr string) string {
	// Remove page number from URL (e.g., page2.html)
	re := regexp.MustCompile(`/page\d+\.html$`)
	urlStr = re.ReplaceAllString(urlStr, "/")

	// Ensure trailing slash
	if !strings.HasSuffix(urlStr, "/") {
		urlStr += "/"
	}

	return urlStr
}

// buildLuxureTVPageURL constructs the page URL for a given page number
func buildLuxureTVPageURL(baseURL string, page int) string {
	if page == 1 {
		return baseURL
	}
	// Remove trailing slash and add page number
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/page%d.html", baseURL, page)
}

// scrapeLuxureTVPage scrapes a single LuxureTV page and extracts video information
func (s *LuxureTVScraperService) scrapeLuxureTVPage(pageURL string) ([]*ScrapedVideo, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to look like a real browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("  DEBUG: Response status: %d, URL: %s\n", resp.StatusCode, resp.Request.URL.String())

	if resp.StatusCode == 404 {
		// Page doesn't exist - return empty to signal end of pagination
		return []*ScrapedVideo{}, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("received status code %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var videos []*ScrapedVideo

	// Debug: count the content divs we find
	contentCount := doc.Find("div.contents div.content").Length()
	fmt.Printf("  DEBUG: Found %d div.content elements on page\n", contentCount)

	// LuxureTV video cards are in div.content elements
	doc.Find("div.contents div.content").Each(func(i int, sel *goquery.Selection) {
		// Find the video link
		link := sel.Find("a[href*='/videos/']").First()
		if link.Length() == 0 {
			return
		}

		href, exists := link.Attr("href")
		if !exists {
			return
		}

		// Ensure full URL
		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = "https://en.luxuretv.com" + href
		}

		// Extract title from the link or img alt
		title := ""
		if titleAttr, exists := link.Attr("title"); exists && titleAttr != "" {
			title = titleAttr
		} else if img := link.Find("img"); img.Length() > 0 {
			if alt, exists := img.Attr("alt"); exists && alt != "" {
				title = alt
			}
		}
		// Also try from vtitle div
		if title == "" || title == "LuxureTV Video" {
			if vtitle := sel.Find("div.vtitle a"); vtitle.Length() > 0 {
				title = strings.TrimSpace(vtitle.Text())
			}
		}
		if title == "" {
			title = "LuxureTV Video"
		}

		// Extract thumbnail URL from img
		thumbnail := ""
		if img := link.Find("img"); img.Length() > 0 {
			// Try data-src first (lazy loading)
			if dataSrc, exists := img.Attr("data-src"); exists && dataSrc != "" {
				thumbnail = dataSrc
			} else if src, exists := img.Attr("src"); exists && src != "" {
				thumbnail = src
			}
		}

		// Extract duration from div.time
		duration := 0
		if timeDiv := sel.Find("div.time b"); timeDiv.Length() > 0 {
			durationText := strings.TrimSpace(timeDiv.Text())
			duration = parseDuration(durationText)
		}

		// Extract view count from div.views
		viewCount := 0
		if viewsDiv := sel.Find("div.views b"); viewsDiv.Length() > 0 {
			viewsText := strings.TrimSpace(viewsDiv.Text())
			viewCount = parseLuxureTVViewCount(viewsText)
		}

		// Extract rating from star_on width (0-100px = 0-100%)
		rating := 0
		if starOn := sel.Find("div.star_on"); starOn.Length() > 0 {
			if style, exists := starOn.Attr("style"); exists {
				rating = parseStarRating(style)
			}
		}

		// Extract external ID from URL (the number at the end)
		externalID := extractLuxureTVID(fullURL)

		// Skip videos with blocked content in title
		if ContainsBlockedContent(title) {
			fmt.Printf("  ✗ Skipped (blocked content): %s\n", title)
			return
		}

		video := &ScrapedVideo{
			Source:       "en.luxuretv.com",
			SiteType:     "luxuretv",
			ContentType:  "zoo", // LuxureTV zoophilia category
			ExternalID:   externalID,
			URL:          fullURL,
			Title:        strings.TrimSpace(title),
			ThumbnailURL: thumbnail,
			Duration:     duration,
			ViewCount:    viewCount,
			Rating:       rating,
			Status:       "discovered",
			DiscoveredAt: time.Now(),
			Tags:         []string{"zoo", "luxuretv"},
			Metadata:     "{}",
		}

		videos = append(videos, video)
		fmt.Printf("  ✓ Found: %s (duration: %ds, views: %d, rating: %d%%)\n", title, duration, viewCount, rating)
	})

	return videos, nil
}

// parseLuxureTVViewCount parses view counts like "6", "5K", "15.1K"
func parseLuxureTVViewCount(viewsText string) int {
	viewsText = strings.TrimSpace(viewsText)
	viewsText = strings.ReplaceAll(viewsText, ",", "")

	// Handle K suffix (thousands)
	if strings.HasSuffix(strings.ToUpper(viewsText), "K") {
		numStr := strings.TrimSuffix(strings.ToUpper(viewsText), "K")
		if num, err := strconv.ParseFloat(numStr, 64); err == nil {
			return int(num * 1000)
		}
	}

	// Handle M suffix (millions)
	if strings.HasSuffix(strings.ToUpper(viewsText), "M") {
		numStr := strings.TrimSuffix(strings.ToUpper(viewsText), "M")
		if num, err := strconv.ParseFloat(numStr, 64); err == nil {
			return int(num * 1000000)
		}
	}

	// Plain number
	if num, err := strconv.Atoi(viewsText); err == nil {
		return num
	}

	return 0
}

// parseStarRating extracts rating percentage from style attribute
// e.g., "width:80px" -> 80 (assuming 100px = 100%)
func parseStarRating(style string) int {
	re := regexp.MustCompile(`width:\s*(\d+)px`)
	matches := re.FindStringSubmatch(style)
	if len(matches) >= 2 {
		if width, err := strconv.Atoi(matches[1]); err == nil {
			// The star bar is 100px wide, so width directly corresponds to percentage
			if width > 100 {
				return 100
			}
			return width
		}
	}
	return 0
}

// extractLuxureTVID extracts the video ID from a LuxureTV URL
// e.g., https://en.luxuretv.com/videos/some-video-title-392593.html -> 392593
func extractLuxureTVID(urlStr string) string {
	re := regexp.MustCompile(`-(\d+)\.html$`)
	matches := re.FindStringSubmatch(urlStr)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// GetLuxureTVCategories returns a list of available LuxureTV categories
func (s *LuxureTVScraperService) GetLuxureTVCategories() []map[string]string {
	return []map[string]string{
		{"name": "Zoophilia", "url": "https://en.luxuretv.com/channels/76/zoophilia/"},
	}
}

// FetchEmbedID fetches the embed ID for a single video by visiting its page
func (s *LuxureTVScraperService) FetchEmbedID(videoURL string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("received status code %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Find the embed code input field and extract the embed ID
	embedID := ""
	doc.Find("input.share-code-champ").Each(func(i int, sel *goquery.Selection) {
		if value, exists := sel.Attr("value"); exists {
			// Extract embed ID from: <iframe src="https://en.luxuretv.com/embed/91723" ...
			re := regexp.MustCompile(`/embed/(\d+)`)
			matches := re.FindStringSubmatch(value)
			if len(matches) >= 2 {
				embedID = matches[1]
			}
		}
	})

	if embedID == "" {
		return "", fmt.Errorf("embed ID not found on page")
	}

	return embedID, nil
}

// FetchEmbedIDsForVideos fetches embed IDs for videos that don't have them yet
func (s *LuxureTVScraperService) FetchEmbedIDsForVideos(videos []*ScrapedVideo) (int, error) {
	updated := 0

	for _, video := range videos {
		// Skip if already has embed ID in metadata
		if video.Metadata != "" && video.Metadata != "{}" {
			var meta map[string]interface{}
			if err := json.Unmarshal([]byte(video.Metadata), &meta); err == nil {
				if _, exists := meta["embed_id"]; exists {
					continue
				}
			}
		}

		// Fetch embed ID
		embedID, err := s.FetchEmbedID(video.URL)
		if err != nil {
			fmt.Printf("  ⚠️  Failed to fetch embed ID for %s: %v\n", video.Title, err)
			continue
		}

		// Update metadata with embed ID
		meta := make(map[string]interface{})
		if video.Metadata != "" && video.Metadata != "{}" {
			json.Unmarshal([]byte(video.Metadata), &meta)
		}
		meta["embed_id"] = embedID
		meta["embed_url"] = fmt.Sprintf("https://en.luxuretv.com/embed/%s", embedID)

		metaJSON, _ := json.Marshal(meta)
		video.Metadata = string(metaJSON)

		// Save updated video
		if err := s.videoScraperService.saveVideo(video); err != nil {
			fmt.Printf("  ⚠️  Failed to save video with embed ID: %v\n", err)
			continue
		}

		fmt.Printf("  ✓ Fetched embed ID %s for: %s\n", embedID, video.Title)
		updated++

		// Small delay to be polite
		time.Sleep(500 * time.Millisecond)
	}

	return updated, nil
}
