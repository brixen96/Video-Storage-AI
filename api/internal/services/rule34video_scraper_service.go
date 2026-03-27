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

// Rule34VideoScraperService handles scraping from rule34video.com
type Rule34VideoScraperService struct {
	db                  *sql.DB
	videoScraperService *VideoScraperService
}

// Rule34VideoScrapeRequest represents a scrape request for Rule34Video
type Rule34VideoScrapeRequest struct {
	MaxPages          int      `json:"maxPages"`          // 0 = unlimited
	StartPage         int      `json:"startPage"`         // Page to start from (for resume)
	SortBy            string   `json:"sortBy"`            // post_date, video_viewed, rating, duration, pseudo_rand
	DateFilter        string   `json:"dateFilter"`        // post_date_past_24_hours, post_date_past_week, post_date_past_month, post_date_past_year
	CategorySlug      string   `json:"categorySlug"`      // e.g. "3d", "big-hero-6"
	ModelSlug         string   `json:"modelSlug"`         // e.g. "anianiboy"
	IncludeTags       []string `json:"includeTags"`       // tags to include
	ExcludeTags       []string `json:"excludeTags"`       // tags to exclude
	FetchDetails      bool     `json:"fetchDetails"`      // fetch video detail page for tags/download URLs
	ActivityID        int64    `json:"-"`                 // Activity ID for pause checking
	InitialVideoCount int      `json:"-"`                 // Initial video count from checkpoint
	InitialSavedCount int      `json:"-"`                 // Initial saved count from checkpoint
}

// Rule34VideoDetails contains detailed info from a video detail page
type Rule34VideoDetails struct {
	DownloadURLs map[string]string `json:"download_urls"` // "360p","480p","720p" -> URL
	Tags         []string          `json:"tags"`
	Categories   []string          `json:"categories"`
	CategoryURLs map[string]string `json:"category_urls"` // name -> URL
	Artist       string            `json:"artist"`
	ArtistURL    string            `json:"artist_url"`
	Uploader     string            `json:"uploader"`
}

// Rule34VideoCategory represents a category on Rule34Video
type Rule34VideoCategory struct {
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	URL   string `json:"url"`
	Count string `json:"count"`
}

// Rule34VideoModel represents a model/artist on Rule34Video
type Rule34VideoModel struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	URL  string `json:"url"`
}

// Rule34VideoScrapeResult contains the results of a Rule34Video scrape
type Rule34VideoScrapeResult struct {
	TotalScraped  int      `json:"totalScraped"`
	NewVideos     int      `json:"newVideos"`
	UpdatedVideos int      `json:"updatedVideos"`
	PagesCrawled  int      `json:"pagesCrawled"`
	Errors        []string `json:"errors"`
}

// NewRule34VideoScraperService creates a new Rule34Video scraper service
func NewRule34VideoScraperService(db *sql.DB, videoScraperService *VideoScraperService) *Rule34VideoScraperService {
	return &Rule34VideoScraperService{
		db:                  db,
		videoScraperService: videoScraperService,
	}
}

// GetSortOptions returns available sort options for Rule34Video
func (s *Rule34VideoScraperService) GetSortOptions() []map[string]string {
	return []map[string]string{
		{"value": "post_date", "label": "Most Recent"},
		{"value": "video_viewed", "label": "Most Viewed"},
		{"value": "rating", "label": "Top Rated"},
		{"value": "duration", "label": "Longest"},
		{"value": "pseudo_rand", "label": "Random"},
		{"value": "video_viewed_today", "label": "Most Viewed Today"},
		{"value": "video_viewed_week", "label": "Most Viewed This Week"},
		{"value": "video_viewed_month", "label": "Most Viewed This Month"},
	}
}

// GetDateFilterOptions returns available date filter options
func (s *Rule34VideoScraperService) GetDateFilterOptions() []map[string]string {
	return []map[string]string{
		{"value": "", "label": "All Time"},
		{"value": "post_date_past_24_hours", "label": "Past 24 Hours"},
		{"value": "post_date_past_week", "label": "Past Week"},
		{"value": "post_date_past_month", "label": "Past Month"},
		{"value": "post_date_past_year", "label": "Past Year"},
	}
}

// Scrape starts scraping Rule34Video with "Straight" filter applied
func (s *Rule34VideoScraperService) Scrape(req Rule34VideoScrapeRequest) (*Rule34VideoScrapeResult, error) {
	return s.ScrapeWithProgress(req, nil)
}

// ScrapeWithProgress scrapes Rule34Video with progress callback
func (s *Rule34VideoScraperService) ScrapeWithProgress(req Rule34VideoScrapeRequest, progressCallback ProgressCallback) (*Rule34VideoScrapeResult, error) {
	// Initialize result with counts from checkpoint (for resume)
	result := &Rule34VideoScrapeResult{
		TotalScraped:  req.InitialVideoCount,
		NewVideos:     0,
		UpdatedVideos: req.InitialSavedCount,
		PagesCrawled:  0,
		Errors:        []string{},
	}

	maxPages := req.MaxPages
	if maxPages <= 0 {
		maxPages = 9999 // Unlimited
	}

	startPage := req.StartPage
	if startPage <= 0 {
		startPage = 1
	}

	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "post_date" // Default to most recent
	}

	// Track seen URLs to detect duplicates
	seenURLs := make(map[string]bool)
	consecutiveEmptyPages := 0

	// Scrape pages
	for page := startPage; page <= maxPages; page++ {
		// Check if task has been paused
		if IsActivityPaused(req.ActivityID) {
			fmt.Printf("⏸️ Rule34Video scrape paused by user at page %d\n", page)
			return result, fmt.Errorf("task paused by user at page %d", page)
		}

		fmt.Printf("\n=== Scraping Rule34Video Page %d (sort: %s) ===\n", page, sortBy)

		videos, err := s.scrapeRule34VideoPage(page, sortBy, req.DateFilter, req.CategorySlug, req.ModelSlug, req.IncludeTags, req.ExcludeTags)
		if err != nil {
			errMsg := fmt.Sprintf("Error scraping page %d: %v", page, err)
			result.Errors = append(result.Errors, errMsg)
			fmt.Printf("❌ %s\n", errMsg)
			consecutiveEmptyPages++
			if consecutiveEmptyPages >= 3 {
				fmt.Printf("🛑 Stopping - 3 consecutive errors/empty pages\n")
				break
			}
			continue
		}

		result.PagesCrawled++
		fmt.Printf("Found %d videos on page %d\n", len(videos), page)

		if len(videos) == 0 {
			consecutiveEmptyPages++
			fmt.Printf("⚠️  No videos found on page %d\n", page)
			if consecutiveEmptyPages >= 2 {
				fmt.Printf("🛑 Stopping pagination - no more videos\n")
				break
			}
			continue
		}

		consecutiveEmptyPages = 0

		// Check for duplicates
		allDuplicates := true
		for _, video := range videos {
			if !seenURLs[video.URL] {
				allDuplicates = false
				seenURLs[video.URL] = true
			}
		}

		if allDuplicates {
			fmt.Printf("🛑 Stopping - all videos on page %d are duplicates\n", page)
			break
		}

		// Fetch details concurrently (up to 4 at a time), then save sequentially
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

				details, detailErr := s.FetchVideoDetails(video.URL)
				if detailErr != nil {
					log.Printf("  ⚠️  Could not fetch details for %s: %v", video.URL, detailErr)
					return
				}

				mu.Lock()
				defer mu.Unlock()
				if len(details.Tags) > 0 {
					video.Tags = details.Tags
				}
				if details.Artist != "" {
					video.Author = details.Artist
				}
				if details.Uploader != "" {
					video.Uploader = details.Uploader
				}
				var metaMap map[string]interface{}
				json.Unmarshal([]byte(video.Metadata), &metaMap)
				if metaMap == nil {
					metaMap = make(map[string]interface{})
				}
				if len(details.Categories) > 0 {
					metaMap["categories"] = details.Categories
					video.Category = strings.Join(details.Categories, ", ")
				}
				if len(details.DownloadURLs) > 0 {
					metaMap["download_urls"] = details.DownloadURLs
				}
				if metaJSON, merr := json.Marshal(metaMap); merr == nil {
					video.Metadata = string(metaJSON)
				}
			}()
		}
		wg.Wait()

		// Save enriched videos sequentially (SQLite doesn't handle concurrent writes well)
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

		// Small delay between pages to be polite
		time.Sleep(800 * time.Millisecond)
	}

	fmt.Printf("\n=== Rule34Video Scrape Complete ===\n")
	fmt.Printf("Pages crawled: %d\n", result.PagesCrawled)
	fmt.Printf("Total scraped: %d\n", result.TotalScraped)
	fmt.Printf("New videos: %d\n", result.NewVideos)
	fmt.Printf("Updated videos: %d\n", result.UpdatedVideos)
	fmt.Printf("Errors: %d\n", len(result.Errors))

	return result, nil
}

// scrapeRule34VideoPage scrapes a single page from Rule34Video using AJAX POST
func (s *Rule34VideoScraperService) scrapeRule34VideoPage(page int, sortBy, dateFilter, categorySlug, modelSlug string, includeTags, excludeTags []string) ([]*ScrapedVideo, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Build base URL based on context (all videos, category, or model)
	baseURL := "https://rule34video.com/"
	if categorySlug != "" {
		baseURL = fmt.Sprintf("https://rule34video.com/categories/%s/", categorySlug)
	} else if modelSlug != "" {
		baseURL = fmt.Sprintf("https://rule34video.com/models/%s/", modelSlug)
	}

	// Build form data for the AJAX request
	formData := fmt.Sprintf("mode=async&function=get_block&block_id=custom_list_videos_most_recent_videos&sort_by=%s", sortBy)

	// Date filter
	if dateFilter != "" {
		formData += "&post_date=" + dateFilter
	}

	// Tag inclusion/exclusion
	if len(includeTags) > 0 || len(excludeTags) > 0 {
		var tagParts []string
		for _, t := range includeTags {
			tagParts = append(tagParts, strings.ReplaceAll(t, " ", "_"))
		}
		for _, t := range excludeTags {
			tagParts = append(tagParts, "-"+strings.ReplaceAll(t, " ", "_"))
		}
		formData += "&tags=" + strings.Join(tagParts, "+")
	}

	// Add pagination - "from" parameter is the page number
	if page > 1 {
		formData += fmt.Sprintf("&from=%d", page)
	}

	fmt.Printf("POST to: %s\n", baseURL)
	fmt.Printf("Form data: %s\n", formData)

	req, err := http.NewRequest("POST", baseURL, strings.NewReader(formData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for AJAX POST request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Origin", "https://rule34video.com")
	req.Header.Set("Referer", "https://rule34video.com/")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %d\n", resp.StatusCode)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("received status code %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Debug: print the HTML length
	html, _ := doc.Html()
	fmt.Printf("Response HTML length: %d bytes\n", len(html))

	// Debug: try to find video containers with different selectors
	fmt.Printf("Looking for video items...\n")
	fmt.Printf("  .item.thumb count: %d\n", doc.Find(".item.thumb").Length())
	fmt.Printf("  div.item.thumb count: %d\n", doc.Find("div.item.thumb").Length())
	fmt.Printf("  a[href*='/video/'] count: %d\n", doc.Find("a[href*='/video/']").Length())

	var videos []*ScrapedVideo

	// Find all video items
	doc.Find(".item.thumb").Each(func(i int, sel *goquery.Selection) {
		// Extract video URL
		link := sel.Find("a.th.js-open-popup")
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		// Build full URL
		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = "https://rule34video.com" + href
		}

		// Extract video ID from URL (e.g., /video/12345/slug/ -> 12345)
		externalID := extractRule34VideoID(fullURL)

		// Extract title - try multiple sources
		title := ""
		// Try .thumb_title div first
		if titleDiv := sel.Find(".thumb_title"); titleDiv.Length() > 0 {
			title = strings.TrimSpace(titleDiv.Text())
		}
		// Fallback to link title attribute
		if title == "" {
			if titleAttr, exists := link.Attr("title"); exists && titleAttr != "" {
				title = strings.TrimSpace(titleAttr)
			}
		}
		// Fallback to image alt
		if title == "" {
			if img := sel.Find("img.thumb"); img.Length() > 0 {
				if alt, exists := img.Attr("alt"); exists && alt != "" {
					title = strings.TrimSpace(alt)
				}
			}
		}
		if title == "" {
			title = "Rule34Video Video"
		}

		// Extract thumbnail - Rule34Video uses data-original for lazy loading
		thumbnail := ""
		if img := sel.Find("img.thumb"); img.Length() > 0 {
			// Try data-original first (lazy loading)
			if dataOriginal, exists := img.Attr("data-original"); exists && dataOriginal != "" {
				thumbnail = dataOriginal
			} else if dataSrc, exists := img.Attr("data-src"); exists && dataSrc != "" {
				thumbnail = dataSrc
			} else if src, exists := img.Attr("src"); exists && src != "" && !strings.Contains(src, "data:image") {
				thumbnail = src
			}
		}

		// Extract preview video URL from data-preview attribute on .wrap_image or .img
		previewURL := ""
		if wrapImage := sel.Find(".wrap_image, .img"); wrapImage.Length() > 0 {
			if preview, exists := wrapImage.Attr("data-preview"); exists && preview != "" {
				previewURL = preview
			}
		}

		// Extract duration
		duration := 0
		if timeEl := sel.Find(".time"); timeEl.Length() > 0 {
			durationText := strings.TrimSpace(timeEl.Text())
			duration = parseDuration(durationText)
		}

		// Extract view count
		viewCount := 0
		if viewsEl := sel.Find(".views"); viewsEl.Length() > 0 {
			viewsText := strings.TrimSpace(viewsEl.Text())
			viewCount = parseRule34ViewCount(viewsText)
		}

		// Extract rating
		rating := 0
		if ratingEl := sel.Find(".rating"); ratingEl.Length() > 0 {
			ratingText := strings.TrimSpace(ratingEl.Text())
			rating = parseRule34Rating(ratingText)
		}

		// Build metadata
		metadata := "{}"
		metadataMap := make(map[string]interface{})
		if previewURL != "" {
			metadataMap["video_preview_url"] = previewURL
		}
		metadataMap["content_filter"] = "straight"
		if len(metadataMap) > 0 {
			if metadataJSON, err := json.Marshal(metadataMap); err == nil {
				metadata = string(metadataJSON)
			}
		}

		// Skip HMV (Hentai Music Video) videos only - Rule34Video handles content filtering
		titleLower := strings.ToLower(title)
		if strings.Contains(titleLower, "hmv") {
			fmt.Printf("  ✗ Skipped (HMV): %s\n", title)
			return
		}

		video := &ScrapedVideo{
			Source:       "rule34video.com",
			SiteType:     "rule34video",
			ContentType:  "3d", // Rule34Video is primarily 3D/hentai animated content
			ExternalID:   externalID,
			URL:          fullURL,
			Title:        title,
			ThumbnailURL: thumbnail,
			Duration:     duration,
			ViewCount:    viewCount,
			Rating:       rating,
			Status:       "discovered",
			DiscoveredAt: time.Now(),
			Tags:         []string{"3D"},
			Metadata:     metadata,
		}

		videos = append(videos, video)
		fmt.Printf("  ✓ Found: %s (duration: %ds, views: %d, rating: %d%%)\n", title, duration, viewCount, rating)
	})

	return videos, nil
}

// extractRule34VideoID extracts the numeric ID from a Rule34Video URL
func extractRule34VideoID(urlStr string) string {
	// URL format: https://rule34video.com/video/12345/slug-name/
	re := regexp.MustCompile(`/video/(\d+)/`)
	matches := re.FindStringSubmatch(urlStr)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// parseRule34ViewCount parses view count from Rule34Video format
func parseRule34ViewCount(viewsText string) int {
	// Remove any non-numeric characters except K, M
	viewsText = strings.TrimSpace(viewsText)
	viewsText = strings.ToUpper(viewsText)

	// Remove "views" text if present
	viewsText = strings.ReplaceAll(viewsText, "VIEWS", "")
	viewsText = strings.TrimSpace(viewsText)

	multiplier := 1.0
	if strings.HasSuffix(viewsText, "K") {
		multiplier = 1000
		viewsText = strings.TrimSuffix(viewsText, "K")
	} else if strings.HasSuffix(viewsText, "M") {
		multiplier = 1000000
		viewsText = strings.TrimSuffix(viewsText, "M")
	}

	// Remove commas and spaces
	viewsText = strings.ReplaceAll(viewsText, ",", "")
	viewsText = strings.ReplaceAll(viewsText, " ", "")

	if value, err := strconv.ParseFloat(viewsText, 64); err == nil {
		return int(value * multiplier)
	}
	return 0
}

// parseRule34Rating parses rating percentage from Rule34Video format
func parseRule34Rating(ratingText string) int {
	// Rating is usually shown as percentage like "95%"
	ratingText = strings.TrimSpace(ratingText)
	ratingText = strings.ReplaceAll(ratingText, "%", "")

	if value, err := strconv.Atoi(ratingText); err == nil {
		return value
	}
	return 0
}

// GetRule34Videos returns all scraped Rule34Video videos
func (s *Rule34VideoScraperService) GetRule34Videos() ([]*ScrapedVideo, error) {
	return s.videoScraperService.GetScrapedVideosBySource("rule34video.com")
}

// GetRule34Stats returns statistics about scraped Rule34Video videos
func (s *Rule34VideoScraperService) GetRule34Stats() (map[string]interface{}, error) {
	videos, err := s.GetRule34Videos()
	if err != nil {
		return nil, err
	}

	totalVideos := len(videos)
	totalDuration := 0
	totalViews := 0
	avgRating := 0

	for _, v := range videos {
		totalDuration += v.Duration
		totalViews += v.ViewCount
		avgRating += v.Rating
	}

	if totalVideos > 0 {
		avgRating = avgRating / totalVideos
	}

	return map[string]interface{}{
		"total_videos":   totalVideos,
		"total_duration": totalDuration,
		"total_views":    totalViews,
		"avg_rating":     avgRating,
	}, nil
}

// FetchDownloadURL fetches the highest quality download URL for a Rule34Video video
func (s *Rule34VideoScraperService) FetchDownloadURL(videoURL string) (string, error) {
	// Create HTTP client with headers
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch video page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("video page returned status %d", resp.StatusCode)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Find download links - Rule34Video typically has them in a download section
	// Look for the highest quality download link
	var downloadURLs []struct {
		url     string
		quality int
	}

	// Try to find download links in various formats
	doc.Find("a[href*='get_file'], a.download_link, .download_options a, a[download]").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists || href == "" {
			return
		}

		// Make absolute URL if relative
		if strings.HasPrefix(href, "/") {
			href = "https://rule34video.com" + href
		}

		// Try to determine quality from the link text or URL
		text := strings.ToLower(sel.Text())
		quality := 0

		if strings.Contains(href, "1080") || strings.Contains(text, "1080") {
			quality = 1080
		} else if strings.Contains(href, "720") || strings.Contains(text, "720") {
			quality = 720
		} else if strings.Contains(href, "480") || strings.Contains(text, "480") {
			quality = 480
		} else if strings.Contains(href, "360") || strings.Contains(text, "360") {
			quality = 360
		} else if strings.Contains(href, "240") || strings.Contains(text, "240") {
			quality = 240
		} else {
			quality = 100 // Default quality for unknown
		}

		downloadURLs = append(downloadURLs, struct {
			url     string
			quality int
		}{url: href, quality: quality})
	})

	// Also check for video source elements
	doc.Find("video source, source[src*='get_file']").Each(func(i int, sel *goquery.Selection) {
		src, exists := sel.Attr("src")
		if !exists || src == "" {
			return
		}

		// Make absolute URL if relative
		if strings.HasPrefix(src, "/") {
			src = "https://rule34video.com" + src
		}

		// Video source is typically highest quality
		downloadURLs = append(downloadURLs, struct {
			url     string
			quality int
		}{url: src, quality: 1080})
	})

	if len(downloadURLs) == 0 {
		return "", fmt.Errorf("no download links found on page")
	}

	// Return highest quality URL
	bestURL := downloadURLs[0]
	for _, dl := range downloadURLs {
		if dl.quality > bestURL.quality {
			bestURL = dl
		}
	}

	return bestURL.url, nil
}

// CheckForNewVideos checks for new videos since last checkpoint
func (s *Rule34VideoScraperService) CheckForNewVideos(maxPages int, activityID int) (*models.NewContentResult, error) {
	result := &models.NewContentResult{
		ScraperType: "rule34video",
		CheckedAt:   time.Now(),
		Details:     map[string]interface{}{},
		NewItems:    []models.NewItemInfo{},
	}

	autoUpdateSvc := NewAutoUpdateService(nil, nil)

	// Get checkpoint
	checkpoint, _ := autoUpdateSvc.GetCheckpoint("rule34video", "last_external_id", "")
	lastSeenID := ""
	if checkpoint != nil {
		lastSeenID = checkpoint.CheckpointValue
	}

	log.Printf("Checking Rule34Video for new videos (last seen ID: %s)", lastSeenID)

	totalNewVideos := 0
	firstVideoID := ""

	// Scrape up to maxPages looking for new videos
	for page := 1; page <= maxPages; page++ {
		videos, err := s.scrapeRule34VideoPage(page, "post_date", "", "", "", nil, nil)
		if err != nil {
			log.Printf("Error scraping page %d: %v", page, err)
			break
		}

		if len(videos) == 0 {
			break
		}

		foundCheckpoint := false
		for _, video := range videos {
			// Save first video ID as new checkpoint
			if firstVideoID == "" {
				firstVideoID = video.ExternalID
			}

			// If we hit the last seen video, stop
			if video.ExternalID == lastSeenID {
				foundCheckpoint = true
				break
			}

			// This is a new video - save it
			if err := s.videoScraperService.saveVideo(video); err == nil {
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
	if firstVideoID != "" {
		autoUpdateSvc.SaveCheckpoint(&models.ScraperCheckpoint{
			ScraperType:     "rule34video",
			CheckpointType:  "last_external_id",
			CheckpointValue: firstVideoID,
			TargetID:        "",
		})
	}

	result.NewItemsFound = totalNewVideos
	result.Details["total_new_videos"] = totalNewVideos
	log.Printf("Found %d new videos from Rule34Video", totalNewVideos)

	return result, nil
}

// FetchVideoDetails fetches full details from a Rule34Video video page
func (s *Rule34VideoScraperService) FetchVideoDetails(videoURL string) (*Rule34VideoDetails, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", videoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("page returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	details := &Rule34VideoDetails{
		DownloadURLs: make(map[string]string),
		CategoryURLs: make(map[string]string),
	}

	// Extract flashvars from inline script for download URLs and metadata
	doc.Find("script").Each(func(_ int, sel *goquery.Selection) {
		text := sel.Text()
		if !strings.Contains(text, "flashvars") {
			return
		}
		// Extract each quality URL
		for _, pair := range []struct{ key, quality string }{
			{"video_url", "360p"},
			{"video_alt_url", "480p"},
			{"video_alt_url2", "720p"},
		} {
			re := regexp.MustCompile(pair.key + `:\s*'([^']+)'`)
			if m := re.FindStringSubmatch(text); len(m) > 1 {
				details.DownloadURLs[pair.quality] = m[1]
			}
		}
	})

	// Extract tags: a.tag_item but not .tag_item_suggest
	doc.Find("a.tag_item").Each(func(_ int, sel *goquery.Selection) {
		if sel.HasClass("tag_item_suggest") {
			return
		}
		if href, _ := sel.Attr("href"); strings.Contains(href, "get_file") {
			// This is a download link, not a tag
			text := strings.TrimSpace(sel.Text())
			if strings.Contains(text, "720") {
				details.DownloadURLs["720p"] = href
			} else if strings.Contains(text, "480") {
				details.DownloadURLs["480p"] = href
			} else if strings.Contains(text, "360") {
				details.DownloadURLs["360p"] = href
			}
			return
		}
		tag := strings.TrimSpace(sel.Text())
		if tag != "" && tag != "+ | Suggest" {
			details.Tags = append(details.Tags, tag)
		}
	})

	// Extract categories, artist, uploader from the info cols
	doc.Find(".col").Each(func(_ int, col *goquery.Selection) {
		label := strings.TrimSpace(col.Find(".label").First().Text())
		switch label {
		case "Categories":
			col.Find("a.item.btn_link").Each(func(_ int, a *goquery.Selection) {
				name := strings.TrimSpace(a.Find("span").Text())
				if name == "" {
					name = strings.TrimSpace(a.Text())
				}
				href, _ := a.Attr("href")
				if name != "" {
					details.Categories = append(details.Categories, name)
					details.CategoryURLs[name] = href
				}
			})
		case "Artist":
			a := col.Find("a.item.btn_link").First()
			details.Artist = strings.TrimSpace(a.Find(".name").Text())
			if details.Artist == "" {
				details.Artist = strings.TrimSpace(a.Text())
			}
			details.ArtistURL, _ = a.Attr("href")
		case "Uploaded by":
			a := col.Find("a.item.btn_link").First()
			// Get text without child element text
			details.Uploader = strings.TrimSpace(a.Clone().Children().Remove().End().Text())
			if details.Uploader == "" {
				details.Uploader = strings.TrimSpace(a.Text())
			}
		}
	})

	return details, nil
}

// GetTopCategories fetches all categories from Rule34Video using pagination
func (s *Rule34VideoScraperService) GetTopCategories() ([]Rule34VideoCategory, error) {
	var allCategories []Rule34VideoCategory
	seen := make(map[string]bool)

	for page := 1; page <= 30; page++ {
		cats, err := s.fetchCategoriesPage(page)
		if err != nil {
			if page == 1 {
				return nil, err
			}
			break
		}
		if len(cats) == 0 {
			break
		}
		added := 0
		for _, c := range cats {
			if !seen[c.Slug] {
				seen[c.Slug] = true
				allCategories = append(allCategories, c)
				added++
			}
		}
		// If no new categories were found, we've reached the end
		if added == 0 {
			break
		}
		if page > 1 {
			time.Sleep(400 * time.Millisecond)
		}
	}

	return allCategories, nil
}

// fetchCategoriesPage fetches one page of categories from Rule34Video
func (s *Rule34VideoScraperService) fetchCategoriesPage(page int) ([]Rule34VideoCategory, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	var reqURL string
	if page == 1 {
		reqURL = "https://rule34video.com/categories/"
	} else {
		reqURL = fmt.Sprintf(
			"https://rule34video.com/categories/?mode=async&function=get_block&block_id=list_categories_categories_list&sort_by=total_videos&from=%d&_=%d",
			page, time.Now().UnixMilli(),
		)
	}

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,*/*;q=0.9")
	if page > 1 {
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Referer", "https://rule34video.com/categories/")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d for categories page %d", resp.StatusCode, page)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var categories []Rule34VideoCategory

	// For AJAX response and full page: find all category links
	doc.Find("a[href*='/categories/']").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		slug := extractRule34Slug(href, "categories")
		if slug == "" {
			return
		}

		// Extract name: try .name first (removing count sub-element), then img alt
		nameEl := sel.Find(".name").First()
		var name string
		if nameEl.Length() > 0 {
			name = strings.TrimSpace(nameEl.Clone().Children().Remove().End().Text())
			if name == "" {
				name = strings.TrimSpace(nameEl.Text())
			}
		}
		if name == "" {
			name, _ = sel.Find("img").First().Attr("alt")
		}
		if name == "" {
			return
		}

		count := strings.TrimSpace(sel.Find(".count span").Text())
		categories = append(categories, Rule34VideoCategory{
			Name:  name,
			Slug:  slug,
			URL:   href,
			Count: count,
		})
	})

	return categories, nil
}

// GetTopModels fetches top models/artists from Rule34Video
func (s *Rule34VideoScraperService) GetTopModels() ([]Rule34VideoModel, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "https://rule34video.com/models/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var models []Rule34VideoModel
	doc.Find(".aside_wrap a.item, .item.model, .models-list .item").Each(func(_ int, sel *goquery.Selection) {
		href, _ := sel.Attr("href")
		name := strings.TrimSpace(sel.Find(".name").First().Clone().Children().Remove().End().Text())
		if name == "" {
			name = strings.TrimSpace(sel.Find(".name").Text())
		}
		if name == "" {
			name, _ = sel.Find("img").First().Attr("alt")
		}
		slug := extractRule34Slug(href, "models")
		if name != "" && slug != "" {
			models = append(models, Rule34VideoModel{
				Name: name,
				Slug: slug,
				URL:  href,
			})
		}
	})

	return models, nil
}

// extractRule34Slug extracts a slug from a Rule34Video URL
// e.g. "/categories/3d/" -> "3d", "/models/anianiboy/" -> "anianiboy"
func extractRule34Slug(urlStr, segment string) string {
	re := regexp.MustCompile(`/` + segment + `/([^/]+)/`)
	if m := re.FindStringSubmatch(urlStr); len(m) > 1 {
		return m[1]
	}
	return ""
}
