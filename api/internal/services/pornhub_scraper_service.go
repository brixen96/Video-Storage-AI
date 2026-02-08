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

// PornHubScraperService handles scraping from PornHub
type PornHubScraperService struct {
	db                  *sql.DB
	videoScraperService *VideoScraperService
}

// PornHubSearchRequest represents a search request for PornHub
type PornHubSearchRequest struct {
	Keyword           string `json:"keyword"`
	MaxPages          int    `json:"maxPages"`          // 0 = unlimited
	StartPage         int    `json:"startPage"`         // Page to start from (for resume)
	ActivityID        int64  `json:"-"`                 // Activity ID for pause checking (not JSON serialized)
	InitialVideoCount int    `json:"-"`                 // Initial video count from checkpoint (for resume)
	InitialSavedCount int    `json:"-"`                 // Initial saved count from checkpoint (for resume)
}

// PornHubScrapeResult contains the results of a PornHub scrape
type PornHubScrapeResult struct {
	Keyword      string   `json:"keyword"`
	TotalScraped int      `json:"totalScraped"`
	NewVideos    int      `json:"newVideos"`
	UpdatedVideos int     `json:"updatedVideos"`
	PagesCrawled int      `json:"pagesCrawled"`
	Errors       []string `json:"errors"`
}

// NewPornHubScraperService creates a new PornHub scraper service
func NewPornHubScraperService(db *sql.DB, videoScraperService *VideoScraperService) *PornHubScraperService {
	return &PornHubScraperService{
		db:                  db,
		videoScraperService: videoScraperService,
	}
}

// ScrapeByKeyword scrapes PornHub videos by keyword search
func (s *PornHubScraperService) ScrapeByKeyword(req PornHubSearchRequest) (*PornHubScrapeResult, error) {
	return s.ScrapeByKeywordWithProgress(req, nil)
}

// ScrapeByKeywordWithProgress scrapes PornHub videos by keyword search with progress callback
func (s *PornHubScraperService) ScrapeByKeywordWithProgress(req PornHubSearchRequest, progressCallback ProgressCallback) (*PornHubScrapeResult, error) {
	if req.Keyword == "" {
		return nil, fmt.Errorf("keyword is required")
	}

	// Initialize result with counts from checkpoint (for resume)
	result := &PornHubScrapeResult{
		Keyword:       req.Keyword,
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

	// Track seen URLs to detect duplicates
	seenURLs := make(map[string]bool)
	consecutiveEmptyPages := 0

	// Scrape pages
	for page := startPage; page <= maxPages; page++ {
		// Check if task has been paused
		if req.ActivityID > 0 {
			activitySvc := NewActivityService()
			currentActivity, err := activitySvc.GetByID(req.ActivityID)
			if err == nil && currentActivity.IsPaused {
				fmt.Printf("⏸️ PornHub scrape paused by user at page %d\n", page)
				return result, fmt.Errorf("task paused by user at page %d", page)
			}
		}

		pageURL := buildPornHubSearchURL(req.Keyword, page)

		fmt.Printf("\n=== Scraping PornHub Page %d: %s ===\n", page, pageURL)

		videos, err := s.scrapePornHubPage(pageURL)
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

		// Save videos
		for _, video := range videos {
			video.Source = "www.pornhub.com"
			video.SiteType = "pornhub"

			// Add keyword to tags
			video.Tags = append(video.Tags, req.Keyword)

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
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\n=== PornHub Scrape Complete ===\n")
	fmt.Printf("Keyword: %s\n", req.Keyword)
	fmt.Printf("Pages crawled: %d\n", result.PagesCrawled)
	fmt.Printf("Total scraped: %d\n", result.TotalScraped)
	fmt.Printf("New videos: %d\n", result.NewVideos)
	fmt.Printf("Updated videos: %d\n", result.UpdatedVideos)
	fmt.Printf("Errors: %d\n", len(result.Errors))

	return result, nil
}

// buildPornHubSearchURL constructs the search URL for a given keyword and page
func buildPornHubSearchURL(keyword string, page int) string {
	encodedKeyword := url.QueryEscape(keyword)
	if page == 1 {
		return fmt.Sprintf("https://www.pornhub.com/video/search?search=%s", encodedKeyword)
	}
	return fmt.Sprintf("https://www.pornhub.com/video/search?search=%s&page=%d", encodedKeyword, page)
}

// scrapePornHubPage scrapes a single PornHub page and extracts video information
func (s *PornHubScraperService) scrapePornHubPage(pageURL string) ([]*ScrapedVideo, error) {
	// Create HTTP client with headers to mimic browser
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

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("received status code %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var videos []*ScrapedVideo

	// PornHub video card selectors
	// Look for the main video link with class js-linkVideoThumb
	doc.Find("a.js-linkVideoThumb, .phimage a[href*='view_video'], a[href*='viewkey=']").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists {
			return
		}

		// Build full URL
		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = "https://www.pornhub.com" + href
		}

		// Skip if not a video URL
		if !strings.Contains(fullURL, "view_video") && !strings.Contains(fullURL, "viewkey=") {
			return
		}

		// Extract video key for external ID
		externalID := ""
		if strings.Contains(fullURL, "viewkey=") {
			parts := strings.Split(fullURL, "viewkey=")
			if len(parts) > 1 {
				externalID = strings.Split(parts[1], "&")[0]
			}
		}

		// Extract thumbnail
		thumbnail := ""
		videoPreviewURL := ""
		if img := sel.Find("img.js-videoThumb, img"); img.Length() > 0 {
			// Try data-mediumthumb first (higher quality)
			if mediumThumb, exists := img.Attr("data-mediumthumb"); exists && mediumThumb != "" {
				thumbnail = mediumThumb
			} else if src, exists := img.Attr("src"); exists && src != "" {
				thumbnail = src
			}

			// Get video preview URL (webm)
			if mediabook, exists := img.Attr("data-mediabook"); exists && mediabook != "" {
				videoPreviewURL = mediabook
			}
		}

		// Extract title
		title := ""
		// Try data-title on the link
		if dataTitle, exists := sel.Attr("data-title"); exists && dataTitle != "" {
			title = dataTitle
		}
		// Try data-title on the image
		if title == "" {
			if img := sel.Find("img"); img.Length() > 0 {
				if dataTitle, exists := img.Attr("data-title"); exists && dataTitle != "" {
					title = dataTitle
				} else if alt, exists := img.Attr("alt"); exists && alt != "" {
					title = alt
				}
			}
		}
		// Try title attribute on link
		if title == "" {
			if titleAttr, exists := sel.Attr("title"); exists && titleAttr != "" {
				title = titleAttr
			}
		}
		// Try finding .thumbnailTitle in parent structure
		if title == "" {
			// Go up to find the wrapper and look for title
			wrapper := sel.Closest(".videoBox, .pcVideoListItem, .wrap")
			if wrapper.Length() > 0 {
				if titleEl := wrapper.Find(".thumbnailTitle, .title a"); titleEl.Length() > 0 {
					title = strings.TrimSpace(titleEl.First().Text())
				}
			}
		}

		if title == "" {
			title = "PornHub Video"
		}

		// Extract duration
		duration := 0
		// Look for duration in parent structure
		wrapper := sel.Closest(".videoBox, .pcVideoListItem, .wrap, .phimage").Parent()
		if wrapper.Length() > 0 {
			if durationEl := wrapper.Find("var.duration, .duration"); durationEl.Length() > 0 {
				durationText := strings.TrimSpace(durationEl.First().Text())
				duration = parseDuration(durationText)
			}
		}
		// Also check within the selection itself
		if duration == 0 {
			if durationEl := sel.Find("var.duration, .duration"); durationEl.Length() > 0 {
				durationText := strings.TrimSpace(durationEl.First().Text())
				duration = parseDuration(durationText)
			}
		}

		// Extract view count
		viewCount := 0
		if wrapper.Length() > 0 {
			if viewsEl := wrapper.Find(".views var, span.views var, .views"); viewsEl.Length() > 0 {
				viewsText := strings.TrimSpace(viewsEl.First().Text())
				viewCount = parseViewCount(viewsText)
			}
		}

		// Extract uploader
		uploader := ""
		if wrapper.Length() > 0 {
			if uploaderEl := wrapper.Find(".usernameWrap a, .videoUploaderBlock a"); uploaderEl.Length() > 0 {
				uploader = strings.TrimSpace(uploaderEl.First().Text())
			}
		}

		// Build metadata
		metadata := "{}"
		metadataMap := make(map[string]interface{})
		if videoPreviewURL != "" {
			metadataMap["video_preview_url"] = videoPreviewURL
		}
		if len(metadataMap) > 0 {
			if metadataJSON, err := json.Marshal(metadataMap); err == nil {
				metadata = string(metadataJSON)
			}
		}

		// Skip videos with blocked content in title
		if ContainsBlockedContent(title) {
			fmt.Printf("  ✗ Skipped (blocked content): %s\n", title)
			return
		}

		video := &ScrapedVideo{
			Source:       "www.pornhub.com",
			SiteType:     "pornhub",
			ContentType:  "regular",
			ExternalID:   externalID,
			URL:          fullURL,
			Title:        strings.TrimSpace(title),
			ThumbnailURL: thumbnail,
			Duration:     duration,
			ViewCount:    viewCount,
			Uploader:     uploader,
			Status:       "discovered",
			DiscoveredAt: time.Now(),
			Tags:         []string{},
			Metadata:     metadata,
		}

		videos = append(videos, video)
		fmt.Printf("  ✓ Found: %s (duration: %ds, views: %d)\n", title, duration, viewCount)
	})

	return videos, nil
}

// GetPornHubKeywords returns a list of suggested keywords for PornHub scraping
func (s *PornHubScraperService) GetPornHubKeywords() []string {
	return []string{
		"amateur",
		"milf",
		"teen",
		"asian",
		"ebony",
		"latina",
		"bbc",
		"anal",
		"lesbian",
		"threesome",
		"pov",
		"creampie",
		"blowjob",
		"hardcore",
		"interracial",
	}
}

// CheckForNewVideos checks for new videos since last checkpoint
func (s *PornHubScraperService) CheckForNewVideos(keywords []string, maxPages int, activityID int) (*models.NewContentResult, error) {
	if len(keywords) == 0 {
		return &models.NewContentResult{
			ScraperType:   "pornhub",
			NewItemsFound: 0,
			CheckedAt:     time.Now(),
			Details:       map[string]interface{}{"message": "No keywords configured"},
		}, nil
	}

	result := &models.NewContentResult{
		ScraperType: "pornhub",
		CheckedAt:   time.Now(),
		Details:     map[string]interface{}{},
		NewItems:    []models.NewItemInfo{},
	}

	autoUpdateSvc := NewAutoUpdateService(nil, nil)
	totalNewVideos := 0

	for _, keyword := range keywords {
		// Get checkpoint for this keyword
		checkpoint, _ := autoUpdateSvc.GetCheckpoint("pornhub", "last_video_url", keyword)
		lastSeenURL := ""
		if checkpoint != nil {
			lastSeenURL = checkpoint.CheckpointValue
		}

		log.Printf("Checking keyword: %s (last seen: %s)", keyword, lastSeenURL)

		newVideosForKeyword := 0
		firstVideoURL := ""

		// Scrape up to maxPages looking for new videos
		for page := 1; page <= maxPages; page++ {
			pageURL := fmt.Sprintf("https://www.pornhub.com/video/search?search=%s&page=%d", url.QueryEscape(keyword), page)
			videos, err := s.scrapePornHubPage(pageURL)
			if err != nil {
				log.Printf("Error scraping page %d for keyword %s: %v", page, keyword, err)
				break
			}

			if len(videos) == 0 {
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
					break
				}

				// This is a new video - save it
				if err := s.videoScraperService.saveVideo(video); err == nil {
					newVideosForKeyword++
					if len(result.NewItems) < 10 { // Limit to first 10 new items
						result.NewItems = append(result.NewItems, models.NewItemInfo{
							URL:       video.URL,
							Title:     video.Title,
							Thumbnail: video.ThumbnailURL,
							Type:      "video",
						})
					}
				}
			}

			if foundCheckpoint {
				break
			}

			time.Sleep(500 * time.Millisecond)
		}

		// Update checkpoint with newest video
		if firstVideoURL != "" && newVideosForKeyword > 0 {
			autoUpdateSvc.SaveCheckpoint(&models.ScraperCheckpoint{
				ScraperType:     "pornhub",
				CheckpointType:  "last_video_url",
				CheckpointValue: firstVideoURL,
				TargetID:        keyword,
			})
		}

		totalNewVideos += newVideosForKeyword
		log.Printf("Found %d new videos for keyword: %s", newVideosForKeyword, keyword)
	}

	result.NewItemsFound = totalNewVideos
	result.Details["keywords_checked"] = len(keywords)
	result.Details["total_new_videos"] = totalNewVideos

	return result, nil
}
