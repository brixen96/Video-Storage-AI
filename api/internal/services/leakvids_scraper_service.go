package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// LeakVidsScraperService handles scraping from LeakVids.com
type LeakVidsScraperService struct {
	db                  *sql.DB
	videoScraperService *VideoScraperService
}

// LeakVidsScrapeRequest represents a scrape request for LeakVids
type LeakVidsScrapeRequest struct {
	BaseURL           string `json:"baseUrl"`
	MaxPages          int    `json:"maxPages"`
	StartPage         int    `json:"startPage"`
	ActivityID        int64  `json:"-"`
	InitialVideoCount int    `json:"-"`
	InitialSavedCount int    `json:"-"`
}

// LeakVidsScrapeResult contains the results of a LeakVids scrape
type LeakVidsScrapeResult struct {
	BaseURL       string   `json:"baseUrl"`
	TotalScraped  int      `json:"totalScraped"`
	NewVideos     int      `json:"newVideos"`
	UpdatedVideos int      `json:"updatedVideos"`
	PagesCrawled  int      `json:"pagesCrawled"`
	Errors        []string `json:"errors"`
}

// NewLeakVidsScraperService creates a new LeakVids scraper service
func NewLeakVidsScraperService(db *sql.DB, videoScraperService *VideoScraperService) *LeakVidsScraperService {
	return &LeakVidsScraperService{
		db:                  db,
		videoScraperService: videoScraperService,
	}
}

// BuildLeakVidsPageURL constructs the async API URL for a given page number
func BuildLeakVidsPageURL(baseURL string, page int) string {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	q := parsedURL.Query()
	q.Set("mode", "async")
	q.Set("function", "get_block")
	q.Set("block_id", "list_videos_latest_videos_list")
	q.Set("sort_by", "post_date")
	q.Set("from", strconv.Itoa(page))
	q.Set("_", strconv.FormatInt(time.Now().UnixMilli(), 10))
	parsedURL.RawQuery = q.Encode()

	return parsedURL.String()
}

// ScrapeWithProgress scrapes LeakVids videos with progress callback
func (s *LeakVidsScraperService) ScrapeWithProgress(req LeakVidsScrapeRequest, progressCallback ProgressCallback) (*LeakVidsScrapeResult, error) {
	if req.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	result := &LeakVidsScrapeResult{
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

	seenURLs := make(map[string]bool)
	consecutiveEmptyPages := 0

	for page := startPage; page < startPage+maxPages; page++ {
		// Check if task has been paused
		if IsActivityPaused(req.ActivityID) {
			log.Printf("⏸️ LeakVids scrape paused by user at page %d", page)
			return result, fmt.Errorf("task paused by user at page %d", page)
		}

		pageURL := BuildLeakVidsPageURL(req.BaseURL, page)
		log.Printf("\n=== Scraping LeakVids Page %d: %s ===", page, pageURL)

		videos, err := s.scrapeLeakVidsPage(pageURL)
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

		// Check for duplicates
		allDuplicates := true
		for _, video := range videos {
			if !seenURLs[video.URL] {
				allDuplicates = false
				seenURLs[video.URL] = true
			}
		}

		if allDuplicates {
			log.Printf("🛑 Stopping - all videos on page %d are duplicates", page)
			break
		}

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
				CurrentPage:    page,
				VideosFound:    result.TotalScraped,
				VideosSaved:    result.NewVideos + result.UpdatedVideos,
				Errors:         result.Errors,
			})
		}

		// Polite delay between pages
		time.Sleep(1500 * time.Millisecond)
	}

	log.Printf("\n=== LeakVids Scrape Complete ===")
	log.Printf("Pages crawled: %d", result.PagesCrawled)
	log.Printf("Total scraped: %d", result.TotalScraped)
	log.Printf("New videos: %d", result.NewVideos)
	log.Printf("Updated videos: %d", result.UpdatedVideos)
	log.Printf("Errors: %d", len(result.Errors))

	return result, nil
}

// scrapeLeakVidsPage scrapes a single LeakVids async API response (HTML fragment)
func (s *LeakVidsScraperService) scrapeLeakVidsPage(pageURL string) ([]*ScrapedVideo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Connection", "keep-alive")

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

	var videos []*ScrapedVideo

	doc.Find(".item").Each(func(i int, sel *goquery.Selection) {
		// Get the link element
		link := sel.Find("a").First()
		if link.Length() == 0 {
			return
		}

		videoURL, _ := link.Attr("href")
		if videoURL == "" {
			return
		}

		// Title from strong.title or link title attribute
		title := strings.TrimSpace(sel.Find("strong.title").Text())
		if title == "" {
			title, _ = link.Attr("title")
			title = strings.TrimSpace(title)
		}
		if title == "" {
			return
		}

		// External ID from data-fav-video-id attribute
		externalID := ""
		sel.Find("[data-fav-video-id]").Each(func(_ int, el *goquery.Selection) {
			if id, exists := el.Attr("data-fav-video-id"); exists && externalID == "" {
				externalID = id
			}
		})
		if externalID == "" {
			// Try to extract from URL path (e.g., /video/2327/...)
			parts := strings.Split(strings.Trim(videoURL, "/"), "/")
			for i, part := range parts {
				if part == "video" && i+1 < len(parts) {
					externalID = parts[i+1]
					break
				}
			}
		}

		// Thumbnail — try multiple attributes since lazy-load may put real URL in data-src/data-original
		thumbnailURL := ""
		if img := sel.Find("img.thumb"); img.Length() > 0 {
			// Try data-src first (most common lazy-load pattern)
			if src, exists := img.Attr("data-src"); exists && src != "" {
				thumbnailURL = src
			}
			// Then data-original
			if thumbnailURL == "" {
				if src, exists := img.Attr("data-original"); exists && src != "" {
					thumbnailURL = src
				}
			}
			// Then src (may be real or placeholder)
			if thumbnailURL == "" {
				if src, exists := img.Attr("src"); exists && src != "" && !strings.Contains(src, "blank.") && !strings.Contains(src, "spacer.") && !strings.Contains(src, "placeholder") && !strings.Contains(src, "data:image") {
					thumbnailURL = src
				}
			}
			// Fallback: data-webp
			if thumbnailURL == "" {
				if src, exists := img.Attr("data-webp"); exists && src != "" {
					thumbnailURL = src
				}
			}
		}

		// Log first video's thumbnail info for debugging
		if i == 0 {
			if img := sel.Find("img.thumb"); img.Length() > 0 {
				srcAttr, _ := img.Attr("src")
				dataSrc, _ := img.Attr("data-src")
				dataOrig, _ := img.Attr("data-original")
				dataWebp, _ := img.Attr("data-webp")
				log.Printf("🔍 LeakVids img attrs - src: %q, data-src: %q, data-original: %q, data-webp: %q, resolved: %q",
					srcAttr, dataSrc, dataOrig, dataWebp, thumbnailURL)
			}
		}

		// Duration
		durationStr := strings.TrimSpace(sel.Find(".duration").Text())
		duration := parseDurationMMSS(durationStr)

		// Rating (e.g., "82%" → 82)
		ratingStr := strings.TrimSpace(sel.Find(".rating").Text())
		rating := parseRatingPercent(ratingStr)

		// Views (e.g., "1.6m", "911.5k")
		viewStr := strings.TrimSpace(sel.Find(".views").Text())
		viewCount := parseViewCount(viewStr)

		// Build metadata
		metadataMap := map[string]interface{}{
			"external_id": externalID,
			"rating":      rating,
		}
		metadataJSON, _ := json.Marshal(metadataMap)

		video := &ScrapedVideo{
			Source:       "leakvids.com",
			SiteType:     "leakvids",
			ContentType:  "regular",
			ExternalID:   externalID,
			URL:          videoURL,
			Title:        title,
			ThumbnailURL: thumbnailURL,
			Duration:     duration,
			ViewCount:    viewCount,
			Rating:       rating,
			Status:       "discovered",
			DiscoveredAt: time.Now(),
			Tags:         []string{},
			Metadata:     string(metadataJSON),
		}

		videos = append(videos, video)
	})

	return videos, nil
}

// parseRatingPercent parses "82%" to int 82
func parseRatingPercent(s string) int {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

// GetLeakVidsStats returns statistics about scraped LeakVids videos
func (s *LeakVidsScraperService) GetLeakVidsStats() (map[string]interface{}, error) {
	var totalVideos int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM scraped_videos WHERE source = 'leakvids.com'`).Scan(&totalVideos)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_videos": totalVideos,
	}, nil
}

// CheckForNewVideos checks for new LeakVids videos (used by auto-update system)
func (s *LeakVidsScraperService) CheckForNewVideos(maxPages int, activityID int) (*models.NewContentResult, error) {
	result := &models.NewContentResult{
		ScraperType: "leakvids",
		CheckedAt:   time.Now(),
		Details:     map[string]interface{}{},
	}

	if maxPages <= 0 {
		maxPages = 3
	}

	baseURL := "https://leakvids.com/latest-updates/"

	for page := 1; page <= maxPages; page++ {
		pageURL := BuildLeakVidsPageURL(baseURL, page)
		videos, err := s.scrapeLeakVidsPage(pageURL)
		if err != nil {
			log.Printf("Error checking LeakVids page %d: %v", page, err)
			continue
		}

		for _, video := range videos {
			// Check if video already exists
			var count int
			s.db.QueryRow(`SELECT COUNT(*) FROM scraped_videos WHERE url = ? AND source = 'leakvids.com'`, video.URL).Scan(&count)
			if count == 0 {
				// Save new video
				if err := s.videoScraperService.saveVideo(video); err == nil {
					result.NewItemsFound++
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

		time.Sleep(1 * time.Second)
	}

	result.Details["pages_checked"] = maxPages
	return result, nil
}
