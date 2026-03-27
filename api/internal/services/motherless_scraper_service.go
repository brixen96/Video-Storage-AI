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

// MotherlessScraperService handles scraping from Motherless.com
type MotherlessScraperService struct {
	db                  *sql.DB
	videoScraperService *VideoScraperService
}

// MotherlessScrapeRequest represents a scrape request for Motherless
type MotherlessScrapeRequest struct {
	SearchURL         string `json:"searchUrl"`
	MaxPages          int    `json:"maxPages"`
	StartPage         int    `json:"startPage"`
	ActivityID        int64  `json:"-"`
	InitialVideoCount int    `json:"-"`
	InitialSavedCount int    `json:"-"`
}

// MotherlessScrapeResult contains the results of a Motherless scrape
type MotherlessScrapeResult struct {
	SearchURL     string   `json:"searchUrl"`
	SearchTerm    string   `json:"searchTerm"`
	TotalScraped  int      `json:"totalScraped"`
	NewVideos     int      `json:"newVideos"`
	UpdatedVideos int      `json:"updatedVideos"`
	PagesCrawled  int      `json:"pagesCrawled"`
	Errors        []string `json:"errors"`
}

// NewMotherlessScraperService creates a new Motherless scraper service
func NewMotherlessScraperService(db *sql.DB, videoScraperService *VideoScraperService) *MotherlessScraperService {
	return &MotherlessScraperService{
		db:                  db,
		videoScraperService: videoScraperService,
	}
}

// ExtractSearchTerm extracts the search term from a Motherless search URL
// e.g., "https://motherless.com/term/videos/mom+incest?..." → "mom incest"
func ExtractSearchTerm(searchURL string) string {
	parsedURL, err := url.Parse(searchURL)
	if err != nil {
		return ""
	}

	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	for i, part := range parts {
		if part == "videos" && i+1 < len(parts) {
			term := parts[i+1]
			term = strings.ReplaceAll(term, "+", " ")
			// URL decode in case of %20 etc
			decoded, err := url.PathUnescape(term)
			if err == nil {
				term = decoded
			}
			return term
		}
	}
	return ""
}

// BuildMotherlessPageURL constructs a page URL by setting the page query parameter
func BuildMotherlessPageURL(baseURL string, page int) string {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := parsedURL.Query()
	q.Set("page", strconv.Itoa(page))
	parsedURL.RawQuery = q.Encode()
	return parsedURL.String()
}

// ScrapeSearch scrapes Motherless videos by search URL
func (s *MotherlessScraperService) ScrapeSearch(req MotherlessScrapeRequest) (*MotherlessScrapeResult, error) {
	return s.ScrapeSearchWithProgress(req, nil)
}

// ScrapeSearchWithProgress scrapes Motherless videos by search URL with progress callback
func (s *MotherlessScraperService) ScrapeSearchWithProgress(req MotherlessScrapeRequest, progressCallback ProgressCallback) (*MotherlessScrapeResult, error) {
	if req.SearchURL == "" {
		return nil, fmt.Errorf("search URL is required")
	}

	searchTerm := ExtractSearchTerm(req.SearchURL)
	if searchTerm == "" {
		searchTerm = "unknown"
	}

	result := &MotherlessScrapeResult{
		SearchURL:     req.SearchURL,
		SearchTerm:    searchTerm,
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
		// Try to extract starting page from the URL
		parsedURL, err := url.Parse(req.SearchURL)
		if err == nil {
			if pageStr := parsedURL.Query().Get("page"); pageStr != "" {
				if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
					startPage = p
				}
			}
		}
		if startPage <= 0 {
			startPage = 1
		}
	}

	seenURLs := make(map[string]bool)
	consecutiveEmptyPages := 0

	for page := startPage; page < startPage+maxPages; page++ {
		// Check if task has been paused
		if IsActivityPaused(req.ActivityID) {
			log.Printf("⏸️ Motherless scrape paused by user at page %d", page)
			return result, fmt.Errorf("task paused by user at page %d", page)
		}

		pageURL := BuildMotherlessPageURL(req.SearchURL, page)
		log.Printf("\n=== Scraping Motherless Page %d: %s ===", page, pageURL)

		videos, err := s.scrapeMotherlessPage(pageURL, searchTerm)
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

	log.Printf("\n=== Motherless Scrape Complete ===")
	log.Printf("Search term: %s", searchTerm)
	log.Printf("Pages crawled: %d", result.PagesCrawled)
	log.Printf("Total scraped: %d", result.TotalScraped)
	log.Printf("New videos: %d", result.NewVideos)
	log.Printf("Updated videos: %d", result.UpdatedVideos)
	log.Printf("Errors: %d", len(result.Errors))

	return result, nil
}

// scrapeMotherlessPage scrapes a single Motherless search results page
func (s *MotherlessScraperService) scrapeMotherlessPage(pageURL string, searchTerm string) ([]*ScrapedVideo, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var videos []*ScrapedVideo

	doc.Find(".thumb-container.video").Each(func(i int, sel *goquery.Selection) {
		// Get codename (external ID)
		thumbEl := sel.Find(".desktop-thumb")
		codename, _ := thumbEl.Attr("data-codename")
		if codename == "" {
			return
		}

		videoURL := fmt.Sprintf("https://motherless.com/%s", codename)

		// Title
		title := strings.TrimSpace(sel.Find("a.caption.title").Text())
		if title == "" {
			title = strings.TrimSpace(sel.Find("a.caption").Text())
		}
		if title == "" {
			title = "Motherless Video " + codename
		}

		// Thumbnail
		thumbnailURL := ""
		if img := sel.Find("img.static"); img.Length() > 0 {
			if src, exists := img.Attr("src"); exists && !strings.Contains(src, "plc.gif") {
				thumbnailURL = src
			}
		}
		if thumbnailURL == "" {
			thumbnailURL = fmt.Sprintf("https://cdn5-thumbs.motherlessmedia.com/thumbs/%s-small-7.jpg", codename)
		}

		// Duration
		durationStr := strings.TrimSpace(sel.Find("span.size").Text())
		duration := parseDurationMMSS(durationStr)

		// Uploader
		uploader := strings.TrimSpace(sel.Find("a.uploader").Text())

		// View count
		viewStr := strings.TrimSpace(sel.Find("span.hits span.value").Text())
		if viewStr == "" {
			viewStr = strings.TrimSpace(sel.Find(".hits .value").Text())
		}
		viewCount := parseViewCount(viewStr)

		// Build metadata
		metadataMap := map[string]interface{}{
			"search_term": searchTerm,
			"codename":    codename,
		}
		metadataJSON, _ := json.Marshal(metadataMap)

		video := &ScrapedVideo{
			Source:       "motherless.com",
			SiteType:     "motherless",
			ContentType:  "regular",
			ExternalID:   codename,
			URL:          videoURL,
			Title:        title,
			ThumbnailURL: thumbnailURL,
			Duration:     duration,
			ViewCount:    viewCount,
			Uploader:     uploader,
			Category:     searchTerm,
			Status:       "discovered",
			DiscoveredAt: time.Now(),
			Tags:         []string{searchTerm},
			Metadata:     string(metadataJSON),
		}

		videos = append(videos, video)
	})

	return videos, nil
}

// parseDurationMMSS parses "MM:SS" or "H:MM:SS" to seconds
func parseDurationMMSS(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	switch len(parts) {
	case 2:
		m, _ := strconv.Atoi(parts[0])
		sec, _ := strconv.Atoi(parts[1])
		return m*60 + sec
	case 3:
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		sec, _ := strconv.Atoi(parts[2])
		return h*3600 + m*60 + sec
	}
	return 0
}

// GetMotherlessStats returns statistics about scraped Motherless videos
func (s *MotherlessScraperService) GetMotherlessStats() (map[string]interface{}, error) {
	totalVideos := 0
	searchTermCounts := make(map[string]int)

	rows, err := s.db.Query(`SELECT category, COUNT(*) as cnt FROM scraped_videos WHERE source = 'motherless.com' GROUP BY category`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			continue
		}
		if category != "" {
			searchTermCounts[category] = count
		}
		totalVideos += count
	}

	return map[string]interface{}{
		"total_videos":       totalVideos,
		"search_term_counts": searchTermCounts,
	}, nil
}

// GetMotherlessSearchTerms returns distinct search terms with counts
func (s *MotherlessScraperService) GetMotherlessSearchTerms() (map[string]int, error) {
	searchTerms := make(map[string]int)

	rows, err := s.db.Query(`SELECT category, COUNT(*) as cnt FROM scraped_videos WHERE source = 'motherless.com' AND category != '' GROUP BY category ORDER BY cnt DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			continue
		}
		searchTerms[category] = count
	}

	return searchTerms, nil
}

// CheckForNewVideos checks for new Motherless videos across all known search terms (used by auto-update system)
func (s *MotherlessScraperService) CheckForNewVideos(maxPages int, activityID int) (*models.NewContentResult, error) {
	result := &models.NewContentResult{
		ScraperType: "motherless",
		CheckedAt:   time.Now(),
		Details:     map[string]interface{}{},
	}

	if maxPages <= 0 {
		maxPages = 2
	}

	// Get all known search terms to check
	searchTerms, err := s.GetMotherlessSearchTerms()
	if err != nil || len(searchTerms) == 0 {
		return result, nil
	}

	for term := range searchTerms {
		baseURL := fmt.Sprintf("https://motherless.com/term/videos/%s?range=0&size=0&sort=relevance&page=1", strings.ReplaceAll(term, " ", "+"))
		for page := 1; page <= maxPages; page++ {
			pageURL := BuildMotherlessPageURL(baseURL, page)
			videos, err := s.scrapeMotherlessPage(pageURL, term)
			if err != nil {
				log.Printf("Error checking Motherless page %d for '%s': %v", page, term, err)
				continue
			}

			for _, video := range videos {
				var count int
				s.db.QueryRow(`SELECT COUNT(*) FROM scraped_videos WHERE url = ? AND source = 'motherless.com'`, video.URL).Scan(&count)
				if count == 0 {
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

			time.Sleep(1500 * time.Millisecond)
		}
	}

	result.Details["search_terms_checked"] = len(searchTerms)
	return result, nil
}
