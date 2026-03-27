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
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// SexuriaScraperService handles scraping from Sexuria.net
type SexuriaScraperService struct {
	db                  *sql.DB
	videoScraperService *VideoScraperService
}

// SexuriaScrapeRequest represents a scrape request for Sexuria
type SexuriaScrapeRequest struct {
	CategoryURL       string `json:"categoryUrl"`
	MaxPages          int    `json:"maxPages"`          // 0 = unlimited
	StartPage         int    `json:"startPage"`         // Page to start from (for resume)
	ActivityID        int64  `json:"-"`                 // Activity ID for pause checking (not JSON serialized)
	InitialVideoCount int    `json:"-"`                 // Initial video count from checkpoint (for resume)
	InitialSavedCount int    `json:"-"`                 // Initial saved count from checkpoint (for resume)
}

// SexuriaAdvancedSearchRequest represents an advanced search request for Sexuria
type SexuriaAdvancedSearchRequest struct {
	Categories        []string `json:"categories"`        // Category IDs (e.g., "2", "12", "1")
	Title             string   `json:"title"`             // Post title search
	Description       string   `json:"description"`       // Publication description search
	Pornstars         string   `json:"pornstars"`         // Pornstar names (comma-separated supported)
	Production        string   `json:"production"`        // Studio/Production name
	Tags              string   `json:"tags"`              // Tags (comma-separated)
	Resolutions       []string `json:"resolutions"`       // Resolution values (e.g., "1080", "720")
	FileHosts         []string `json:"fileHosts"`         // File hosts to filter by
	MaxPages          int      `json:"maxPages"`          // 0 = unlimited
	StartPage         int      `json:"startPage"`         // Page to start from (for resume)
	ActivityID        int64    `json:"-"`                 // Activity ID for pause checking (not JSON serialized)
	InitialVideoCount int      `json:"-"`                 // Initial video count from checkpoint (for resume)
	InitialSavedCount int      `json:"-"`                 // Initial saved count from checkpoint (for resume)
}

// SexuriaFileHost represents available file hosts for filtering
type SexuriaFileHost struct {
	Name  string `json:"name"`
	Param string `json:"param"`
}

// GetSexuriaFileHosts returns available file hosts for filtering
func (s *SexuriaScraperService) GetSexuriaFileHosts() []SexuriaFileHost {
	return []SexuriaFileHost{
		{Name: "Rapidgator", Param: "g.link1-rapidgator"},
		{Name: "FastFile.cc", Param: "g.links1-share-online"},
		{Name: "Keep2share", Param: "g.keep2share1"},
		{Name: "KatFile", Param: "g.katfile1"},
		{Name: "File.al", Param: "g.fileal"},
		{Name: "Dropapk.to", Param: "g.dropapk"},
		{Name: "Filespace.com", Param: "g.filespace"},
		{Name: "RapidCloud.cc", Param: "g.rapidcloud"},
	}
}

// SexuriaSearchCategory represents a category option for advanced search
type SexuriaSearchCategory struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsParent bool   `json:"isParent"`
}

// GetSexuriaSearchCategories returns all categories available for advanced search
func (s *SexuriaScraperService) GetSexuriaSearchCategories() []SexuriaSearchCategory {
	return []SexuriaSearchCategory{
		{ID: "2", Name: "Movies", IsParent: true},
		{ID: "12", Name: "  - DVD-Rips", IsParent: false},
		{ID: "13", Name: "  - HD-DVDs", IsParent: false},
		{ID: "18", Name: "  - Virtual Reality", IsParent: false},
		{ID: "19", Name: "  - DVD Untouched", IsParent: false},
		{ID: "1", Name: "Videos", IsParent: true},
		{ID: "11", Name: "  - Scenes", IsParent: false},
		{ID: "10", Name: "  - HD-Clips", IsParent: false},
		{ID: "17", Name: "  - Virtual Reality", IsParent: false},
		{ID: "34", Name: "JAV", IsParent: true},
		{ID: "35", Name: "  - Censored", IsParent: false},
		{ID: "36", Name: "  - Uncensored", IsParent: false},
		{ID: "3", Name: "Site-Rips", IsParent: true},
		{ID: "8", Name: "Pic-Sets", IsParent: true},
		{ID: "14", Name: "eBooks", IsParent: true},
		{ID: "15", Name: "Games", IsParent: true},
		{ID: "20", Name: "Comics", IsParent: true},
		{ID: "32", Name: "3D Animation", IsParent: true},
		{ID: "33", Name: "Hentai", IsParent: true},
	}
}

// GetSexuriaResolutions returns available resolution options
func (s *SexuriaScraperService) GetSexuriaResolutions() []string {
	return []string{"360", "480", "540", "720", "1080", "1440", "2160"}
}

// BuildAdvancedSearchURL builds the URL for advanced search
func (s *SexuriaScraperService) BuildAdvancedSearchURL(req SexuriaAdvancedSearchRequest) string {
	baseURL := "https://sexuria.net/f/"
	var parts []string

	// Add categories (p.cat parameter)
	if len(req.Categories) > 0 {
		parts = append(parts, fmt.Sprintf("p.cat=%s", strings.Join(req.Categories, ",")))
	}

	// Add file hosts
	for _, host := range req.FileHosts {
		parts = append(parts, fmt.Sprintf("%s=1", host))
	}

	// Add title search
	if req.Title != "" {
		parts = append(parts, fmt.Sprintf("j.x.name-released=%s", strings.ReplaceAll(req.Title, " ", "+")))
	}

	// Add description search
	if req.Description != "" {
		parts = append(parts, fmt.Sprintf("l.full_story=%s", strings.ReplaceAll(req.Description, " ", "+")))
	}

	// Add pornstars search
	if req.Pornstars != "" {
		parts = append(parts, fmt.Sprintf("j.x.pornstars;p.full_story=%s", strings.ReplaceAll(req.Pornstars, " ", "+")))
	}

	// Add production search
	if req.Production != "" {
		parts = append(parts, fmt.Sprintf("j.x.production;p.full_story=%s", strings.ReplaceAll(req.Production, " ", "+")))
	}

	// Add tags search (comma-separated with +)
	if req.Tags != "" {
		// Convert "tag1, tag2" to "tag1,+tag2"
		tags := strings.ReplaceAll(req.Tags, ", ", ",+")
		tags = strings.ReplaceAll(tags, " ", "+")
		parts = append(parts, fmt.Sprintf("n.m.tags=%s", tags))
	}

	// Add resolutions
	if len(req.Resolutions) > 0 {
		for _, res := range req.Resolutions {
			parts = append(parts, fmt.Sprintf("resolution=%s", res))
		}
	}

	// Add sorting (always sort by date descending)
	parts = append(parts, "sort=date")
	parts = append(parts, "order=desc")

	return baseURL + strings.Join(parts, "/") + "/"
}

// ScrapeAdvancedSearch scrapes Sexuria using advanced search parameters
func (s *SexuriaScraperService) ScrapeAdvancedSearch(req SexuriaAdvancedSearchRequest) (*SexuriaScrapeResult, error) {
	return s.ScrapeAdvancedSearchWithProgress(req, nil)
}

// ScrapeAdvancedSearchWithProgress scrapes Sexuria using advanced search with progress callback
func (s *SexuriaScraperService) ScrapeAdvancedSearchWithProgress(req SexuriaAdvancedSearchRequest, progressCallback ProgressCallback) (*SexuriaScrapeResult, error) {
	searchURL := s.BuildAdvancedSearchURL(req)

	// Initialize result with counts from checkpoint (for resume)
	result := &SexuriaScrapeResult{
		CategoryURL:  searchURL,
		TotalScraped: req.InitialVideoCount, // Start from checkpoint count
		NewItems:     0,
		UpdatedItems: req.InitialSavedCount, // Approximate - assume all previous were updates
		PagesCrawled: 0,
		Errors:       []string{},
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
		if IsActivityPaused(req.ActivityID) {
			fmt.Printf("⏸️ Sexuria advanced search paused by user at page %d\n", page)
			return result, fmt.Errorf("task paused by user at page %d", page)
		}

		pageURL := buildSexuriaAdvancedSearchPageURL(searchURL, page)

		fmt.Printf("\n=== Scraping Sexuria Advanced Search Page %d: %s ===\n", page, pageURL)

		items, err := s.scrapeSexuriaPage(pageURL)
		if err != nil {
			errMsg := fmt.Sprintf("Error scraping page %d: %v", page, err)
			result.Errors = append(result.Errors, errMsg)
			fmt.Println(errMsg)

			consecutiveEmptyPages++
			if consecutiveEmptyPages >= 2 {
				fmt.Println("Too many consecutive errors, stopping")
				break
			}
			continue
		}

		if len(items) == 0 {
			fmt.Printf("No items found on page %d\n", page)
			consecutiveEmptyPages++
			if consecutiveEmptyPages >= 2 {
				fmt.Println("Two consecutive empty pages, stopping")
				break
			}
			continue
		}

		consecutiveEmptyPages = 0
		result.PagesCrawled++
		result.TotalScraped += len(items)

		// Save items to database
		for _, item := range items {
			item.Source = "sexuria.net"
			item.SiteType = "advanced_search"
			err := s.videoScraperService.saveVideo(item)
			if err != nil {
				if strings.Contains(err.Error(), "Updated existing") {
					result.UpdatedItems++
				} else {
					errMsg := fmt.Sprintf("Error saving item %s: %v", item.Title, err)
					result.Errors = append(result.Errors, errMsg)
				}
				continue
			}
			result.NewItems++
		}

		fmt.Printf("Page %d: Found %d items, %d new, %d updated\n",
			page, len(items), result.NewItems, result.UpdatedItems)

		// Emit progress
		if progressCallback != nil {
			progressCallback(ScrapeProgress{
				PagesCompleted: result.PagesCrawled,
				CurrentPage:    page, // Actual page number for checkpoint/resume
				VideosFound:    result.TotalScraped,
				VideosSaved:    result.NewItems + result.UpdatedItems,
				Errors:         result.Errors,
			})
		}

		// Delay between pages to be polite
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("\n=== Sexuria Advanced Search Complete ===\n")
	fmt.Printf("Search URL: %s\n", searchURL)
	fmt.Printf("Pages crawled: %d\n", result.PagesCrawled)
	fmt.Printf("Total scraped: %d\n", result.TotalScraped)
	fmt.Printf("New items: %d\n", result.NewItems)
	fmt.Printf("Updated items: %d\n", result.UpdatedItems)
	fmt.Printf("Errors: %d\n", len(result.Errors))

	return result, nil
}

// buildSexuriaAdvancedSearchPageURL constructs the page URL for advanced search pagination
func buildSexuriaAdvancedSearchPageURL(baseURL string, page int) string {
	if page == 1 {
		return baseURL
	}
	// For advanced search, pagination is added before the trailing slash
	// e.g., https://sexuria.net/f/.../sort=date/order=desc/page/2/
	if strings.HasSuffix(baseURL, "/") {
		return fmt.Sprintf("%spage/%d/", baseURL, page)
	}
	return fmt.Sprintf("%s/page/%d/", baseURL, page)
}

// SexuriaScrapeResult contains the results of a Sexuria scrape
type SexuriaScrapeResult struct {
	CategoryURL   string   `json:"categoryUrl"`
	TotalScraped  int      `json:"totalScraped"`
	NewItems      int      `json:"newItems"`
	UpdatedItems  int      `json:"updatedItems"`
	PagesCrawled  int      `json:"pagesCrawled"`
	Errors        []string `json:"errors"`
}

// SexuriaCategory represents a category on Sexuria
type SexuriaCategory struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Slug string `json:"slug"`
}

// SexuriaItemDetails contains detailed information about a Sexuria item
type SexuriaItemDetails struct {
	Screenshots []string `json:"screenshots"`
	Tags        []string `json:"tags"`
	PornStars   []string `json:"porn_stars"`
}

// NewSexuriaScraperService creates a new Sexuria scraper service
func NewSexuriaScraperService(db *sql.DB, videoScraperService *VideoScraperService) *SexuriaScraperService {
	return &SexuriaScraperService{
		db:                  db,
		videoScraperService: videoScraperService,
	}
}

// GetSexuriaCategories returns available Sexuria categories
func (s *SexuriaScraperService) GetSexuriaCategories() []SexuriaCategory {
	return []SexuriaCategory{
		{Name: "Movies", URL: "https://sexuria.net/movies/", Slug: "movies"},
		{Name: "Videos", URL: "https://sexuria.net/videos/", Slug: "videos"},
		{Name: "3D Animation", URL: "https://sexuria.net/3d-animation/", Slug: "3d-animation"},
	}
}

// ScrapeCategory scrapes a Sexuria category
func (s *SexuriaScraperService) ScrapeCategory(req SexuriaScrapeRequest) (*SexuriaScrapeResult, error) {
	return s.ScrapeCategoryWithProgress(req, nil)
}

// ScrapeCategoryWithProgress scrapes a Sexuria category with progress callback
func (s *SexuriaScraperService) ScrapeCategoryWithProgress(req SexuriaScrapeRequest, progressCallback ProgressCallback) (*SexuriaScrapeResult, error) {
	if req.CategoryURL == "" {
		return nil, fmt.Errorf("category URL is required")
	}

	// Initialize result with counts from checkpoint (for resume)
	result := &SexuriaScrapeResult{
		CategoryURL:  req.CategoryURL,
		TotalScraped: req.InitialVideoCount, // Start from checkpoint count
		NewItems:     0,
		UpdatedItems: req.InitialSavedCount, // Approximate - assume all previous were updates
		PagesCrawled: 0,
		Errors:       []string{},
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
		if IsActivityPaused(req.ActivityID) {
			fmt.Printf("⏸️ Sexuria scrape paused by user at page %d\n", page)
			return result, fmt.Errorf("task paused by user at page %d", page)
		}

		pageURL := buildSexuriaPageURL(req.CategoryURL, page)

		fmt.Printf("\n=== Scraping Sexuria Page %d: %s ===\n", page, pageURL)

		items, err := s.scrapeSexuriaPage(pageURL)
		if err != nil {
			errMsg := fmt.Sprintf("Error scraping page %d: %v", page, err)
			result.Errors = append(result.Errors, errMsg)
			fmt.Printf("Error: %s\n", errMsg)
			consecutiveEmptyPages++
			if consecutiveEmptyPages >= 5 {
				fmt.Printf("Stopping - 5 consecutive errors/empty pages\n")
				break
			}
			continue
		}

		result.PagesCrawled++
		fmt.Printf("Found %d items on page %d\n", len(items), page)

		if len(items) == 0 {
			consecutiveEmptyPages++
			fmt.Printf("Warning: No items found on page %d\n", page)
			if consecutiveEmptyPages >= 5 {
				fmt.Printf("Stopping pagination - no more items\n")
				break
			}
			continue
		}

		consecutiveEmptyPages = 0

		// Save items
		for _, item := range items {
			err := s.videoScraperService.saveVideo(item)
			if err != nil {
				if strings.Contains(err.Error(), "Updated existing") {
					result.UpdatedItems++
				} else {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to save item: %v", err))
				}
			} else {
				result.NewItems++
			}
			result.TotalScraped++
		}

		// Call progress callback
		if progressCallback != nil {
			progressCallback(ScrapeProgress{
				PagesCompleted: result.PagesCrawled,
				CurrentPage:    page, // Actual page number for checkpoint/resume
				VideosFound:    result.TotalScraped,
				VideosSaved:    result.NewItems + result.UpdatedItems,
				Errors:         result.Errors,
			})
		}

		// Delay between pages to be polite
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("\n=== Sexuria Scrape Complete ===\n")
	fmt.Printf("Category: %s\n", req.CategoryURL)
	fmt.Printf("Pages crawled: %d\n", result.PagesCrawled)
	fmt.Printf("Total scraped: %d\n", result.TotalScraped)
	fmt.Printf("New items: %d\n", result.NewItems)
	fmt.Printf("Updated items: %d\n", result.UpdatedItems)
	fmt.Printf("Errors: %d\n", len(result.Errors))

	return result, nil
}

// buildSexuriaPageURL constructs the page URL for pagination
func buildSexuriaPageURL(baseURL string, page int) string {
	// Ensure base URL ends with /
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	if page == 1 {
		return baseURL
	}
	return fmt.Sprintf("%spage/%d/", baseURL, page)
}

// scrapeSexuriaPage scrapes a single Sexuria page and extracts item information
func (s *SexuriaScraperService) scrapeSexuriaPage(pageURL string) ([]*ScrapedVideo, error) {
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

	fmt.Printf("Response status: %d\n", resp.StatusCode)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("received status code %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var items []*ScrapedVideo

	// Find all article.short elements
	doc.Find("article.short").Each(func(i int, sel *goquery.Selection) {
		// Extract URL and title from header link
		headerLink := sel.Find(".header a[title]")
		href, exists := headerLink.Attr("href")
		if !exists {
			return
		}

		title := ""
		if titleAttr, exists := headerLink.Attr("title"); exists {
			title = strings.TrimSpace(titleAttr)
		}
		if title == "" {
			title = strings.TrimSpace(headerLink.Find("h2").Text())
		}
		if title == "" {
			title = "Sexuria Item"
		}

		// Extract external ID from URL (e.g., 674712-brunette-babes-1080p.html -> 674712)
		externalID := extractSexuriaID(href)

		// Extract thumbnail from a.img > img (the actual HTML structure on sexuria.net)
		// Example: <a href="..." class="img"><img src="/uploads/posts/..." /></a>
		thumbnail := ""
		const baseURL = "https://sexuria.net"

		// Primary method: Look for a.img img - this is the actual structure
		if aImg := sel.Find("a.img img"); aImg.Length() > 0 {
			// Try src first (the actual loaded image)
			if src, exists := aImg.Attr("src"); exists && src != "" && !strings.Contains(src, "data:image") {
				thumbnail = src
			}
			// Fallback to data-src for lazy loading
			if thumbnail == "" {
				if dataSrc, exists := aImg.Attr("data-src"); exists && dataSrc != "" {
					thumbnail = dataSrc
				}
			}
		}

		// Fallback: Try .img img (class selector)
		if thumbnail == "" {
			if img := sel.Find(".img img"); img.Length() > 0 {
				if src, exists := img.Attr("src"); exists && src != "" && !strings.Contains(src, "data:image") {
					thumbnail = src
				}
			}
		}

		// Last resort: Try any img in the article
		if thumbnail == "" {
			sel.Find("img").Each(func(j int, imgSel *goquery.Selection) {
				if thumbnail != "" {
					return
				}
				if src, exists := imgSel.Attr("src"); exists && src != "" && !strings.Contains(src, "data:image") && !strings.Contains(src, "blank") {
					thumbnail = src
				}
			})
		}

		// Add base URL if the thumbnail is a relative path
		if thumbnail != "" && strings.HasPrefix(thumbnail, "/") {
			thumbnail = baseURL + thumbnail
		}

		// Extract file hosts from .released div span elements
		var fileHosts []string
		sel.Find(".released div span").Each(func(j int, spanSel *goquery.Selection) {
			host := strings.TrimSpace(spanSel.Text())
			if host != "" && host != "Video size:" {
				fileHosts = append(fileHosts, host)
			}
		})

		// Extract file size
		fileSize := ""
		sel.Find(".released div").Each(func(j int, divSel *goquery.Selection) {
			text := strings.TrimSpace(divSel.Text())
			if strings.Contains(text, "Video size:") {
				// Extract the size value after "Video size:"
				parts := strings.Split(text, "Video size:")
				if len(parts) > 1 {
					// The size might be in a nested element or after the text
					sizeText := strings.TrimSpace(parts[1])
					// Clean up - remove any host names that might be mixed in
					for _, host := range fileHosts {
						sizeText = strings.Replace(sizeText, host, "", -1)
					}
					fileSize = strings.TrimSpace(sizeText)
				}
			}
		})

		// Try to extract file size from the specific structure
		if fileSize == "" {
			// Look for pattern like "Video size: 1.9 GB"
			releasedText := sel.Find(".released").Text()
			sizeRegex := regexp.MustCompile(`Video size:\s*([\d.]+\s*(?:GB|MB|TB))`)
			if matches := sizeRegex.FindStringSubmatch(releasedText); len(matches) > 1 {
				fileSize = strings.TrimSpace(matches[1])
			}
		}

		// Extract author
		author := ""
		sel.Find(".author-cat a").Each(func(j int, aSel *goquery.Selection) {
			// Skip the category link, get the author link
			if href, exists := aSel.Attr("href"); exists && strings.Contains(href, "/author/") {
				author = strings.TrimSpace(aSel.Text())
			}
		})

		// Extract date
		dateStr := ""
		month := strings.TrimSpace(sel.Find(".date .month").Text())
		day := strings.TrimSpace(sel.Find(".date .day").Text())
		if month != "" && day != "" {
			dateStr = fmt.Sprintf("%s %s", month, day)
		}

		// Extract subcategory
		subcategory := ""
		sel.Find(".author-cat a").Each(func(j int, aSel *goquery.Selection) {
			if href, exists := aSel.Attr("href"); exists && !strings.Contains(href, "/author/") {
				subcategory = strings.TrimSpace(aSel.Text())
			}
		})

		// Build metadata
		metadataMap := map[string]interface{}{
			"file_hosts":  fileHosts,
			"file_size":   fileSize,
			"author":      author,
			"date":        dateStr,
			"subcategory": subcategory,
			"site_type":   "download_links",
		}
		metadataJSON, _ := json.Marshal(metadataMap)

		// Skip items with blocked content in title or subcategory
		if ContainsBlockedContent(title) || ContainsBlockedContent(subcategory) {
			fmt.Printf("  Skipped (blocked content): %s\n", title)
			return // Skip this item
		}

		item := &ScrapedVideo{
			Source:       "sexuria.net",
			SiteType:     "sexuria",
			ContentType:  "regular", // Sexuria is regular content
			ExternalID:   externalID,
			URL:          href,
			Title:        title,
			ThumbnailURL: thumbnail,
			Duration:     0, // Not applicable for download links
			ViewCount:    0,
			Uploader:     author,
			Category:     subcategory, // Store subcategory for filtering
			Status:       "discovered",
			DiscoveredAt: time.Now(),
			Tags:         fileHosts, // Store file hosts as tags for easy filtering
			Metadata:     string(metadataJSON),
		}

		items = append(items, item)
		fmt.Printf("  Found: %s (hosts: %v, size: %s)\n", title, fileHosts, fileSize)
	})

	return items, nil
}

// extractSexuriaID extracts the numeric ID from a Sexuria URL
func extractSexuriaID(url string) string {
	// URL format: https://sexuria.net/movies/674712-brunette-babes-1080p.html
	// Extract: 674712
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove .html extension
		lastPart = strings.TrimSuffix(lastPart, ".html")
		// Extract the numeric ID before the first dash
		if idx := strings.Index(lastPart, "-"); idx > 0 {
			return lastPart[:idx]
		}
		return lastPart
	}
	return ""
}

// GetSexuriaItems returns all scraped Sexuria items (filtered for blocked content)
func (s *SexuriaScraperService) GetSexuriaItems() ([]*ScrapedVideo, error) {
	// Query by source - match both "sexuria" and "sexuria.net" for backwards compatibility
	videos, err := s.videoScraperService.GetScrapedVideosBySourceLike("sexuria%")
	if err != nil {
		return nil, err
	}

	return videos, nil
}

// GetSexuriaStats returns statistics about scraped Sexuria items
func (s *SexuriaScraperService) GetSexuriaStats() (map[string]interface{}, error) {
	items, err := s.GetSexuriaItems()
	if err != nil {
		return nil, err
	}

	// Count items and collect file host stats
	totalItems := len(items)
	hostCounts := make(map[string]int)

	for _, item := range items {
		// Tags contain file hosts
		for _, host := range item.Tags {
			hostCounts[host]++
		}
	}

	return map[string]interface{}{
		"total_items": totalItems,
		"host_counts": hostCounts,
	}, nil
}

// ParseFileSize parses a file size string and returns bytes
func ParseFileSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))

	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "TB") {
		multiplier = 1024 * 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "TB")
	}

	sizeStr = strings.TrimSpace(sizeStr)
	if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
		return int64(size * float64(multiplier))
	}

	return 0
}

// FetchItemDetails fetches detailed information for a Sexuria item from its page
func (s *SexuriaScraperService) FetchItemDetails(itemURL string) (*SexuriaItemDetails, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", itemURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

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

	details := &SexuriaItemDetails{
		Screenshots: []string{},
		Tags:        []string{},
		PornStars:   []string{},
	}

	// Extract screenshots from div.screenshot a.highslide href
	doc.Find("div.screenshot a.highslide").Each(func(i int, sel *goquery.Selection) {
		if href, exists := sel.Attr("href"); exists && href != "" {
			details.Screenshots = append(details.Screenshots, href)
		}
	})

	// Extract tags - look for links to /tags/ URLs specifically
	// These are the actual content tags, not file hosts or other metadata
	doc.Find("a[href*='/tags/']").Each(func(i int, sel *goquery.Selection) {
		tag := strings.TrimSpace(sel.Text())
		// Skip empty tags and duplicates
		if tag != "" {
			// Check if we already have this tag
			isDuplicate := false
			for _, existingTag := range details.Tags {
				if existingTag == tag {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				details.Tags = append(details.Tags, tag)
			}
		}
	})

	// Helper to add unique porn star
	addUniqueStar := func(star string) {
		star = strings.TrimSpace(star)
		if star == "" || star == "..." {
			return
		}
		// Check for duplicates
		for _, existing := range details.PornStars {
			if existing == star {
				return
			}
		}
		details.PornStars = append(details.PornStars, star)
	}

	// Extract porn stars - look for links to xfsearch/pornstars URLs
	doc.Find("a[href*='xfsearch/pornstars']").Each(func(i int, sel *goquery.Selection) {
		addUniqueStar(sel.Text())
	})

	// Also try stars-porn class if no stars found yet
	if len(details.PornStars) == 0 {
		doc.Find(".stars-porn a").Each(func(i int, sel *goquery.Selection) {
			addUniqueStar(sel.Text())
		})
	}

	return details, nil
}

// GetItemByID returns a single Sexuria item by its database ID
func (s *SexuriaScraperService) GetItemByID(id int64) (*ScrapedVideo, error) {
	videos, err := s.videoScraperService.GetScrapedVideos()
	if err != nil {
		return nil, err
	}

	for _, v := range videos {
		if v.ID == id && (v.SiteType == "sexuria" || v.Source == "sexuria.net") {
			return v, nil
		}
	}

	return nil, fmt.Errorf("item not found")
}

// CheckForNewItems checks for new items since last checkpoint
func (s *SexuriaScraperService) CheckForNewItems(categories []string, maxPages int, activityID int) (*models.NewContentResult, error) {
	result := &models.NewContentResult{
		ScraperType: "sexuria",
		CheckedAt:   time.Now(),
		Details:     map[string]interface{}{},
		NewItems:    []models.NewItemInfo{},
	}

	// If no categories specified, use default
	if len(categories) == 0 {
		categories = []string{"https://sexuria.net/v1/"}
	}

	autoUpdateSvc := NewAutoUpdateService(nil, nil)
	totalNewItems := 0

	for _, category := range categories {
		// Get checkpoint for this category
		checkpoint, _ := autoUpdateSvc.GetCheckpoint("sexuria", "last_item_url", category)
		lastSeenURL := ""
		if checkpoint != nil {
			lastSeenURL = checkpoint.CheckpointValue
		}

		log.Printf("Checking Sexuria category: %s (last seen: %s)", category, lastSeenURL)

		newItemsForCategory := 0
		firstItemURL := ""

		// Scrape up to maxPages looking for new items
		for page := 1; page <= maxPages; page++ {
			pageURL := category
			if page > 1 {
				pageURL = fmt.Sprintf("%spage/%d/", category, page)
			}

			items, err := s.scrapeSexuriaPage(pageURL)
			if err != nil {
				log.Printf("Error scraping page %d: %v", page, err)
				break
			}

			if len(items) == 0 {
				break
			}

			foundCheckpoint := false
			for _, item := range items {
				if firstItemURL == "" {
					firstItemURL = item.URL
				}

				if item.URL == lastSeenURL {
					foundCheckpoint = true
					break
				}

				if err := s.videoScraperService.saveVideo(item); err == nil {
					newItemsForCategory++
					if len(result.NewItems) < 10 {
						result.NewItems = append(result.NewItems, models.NewItemInfo{
							URL:       item.URL,
							Title:     item.Title,
							Thumbnail: item.ThumbnailURL,
							Type:      "item",
						})
					}
				}
			}

			if foundCheckpoint {
				break
			}

			time.Sleep(1 * time.Second)
		}

		if firstItemURL != "" {
			autoUpdateSvc.SaveCheckpoint(&models.ScraperCheckpoint{
				ScraperType:     "sexuria",
				CheckpointType:  "last_item_url",
				CheckpointValue: firstItemURL,
				TargetID:        category,
			})
		}

		totalNewItems += newItemsForCategory
		log.Printf("Found %d new items for category: %s", newItemsForCategory, category)
	}

	result.NewItemsFound = totalNewItems
	result.Details["categories_checked"] = len(categories)
	result.Details["total_new_items"] = totalNewItems

	return result, nil
}

// SexuriaDownloadLink represents a download link extracted from a Sexuria item page
type SexuriaDownloadLink struct {
	ID            int64      `json:"id"`
	VideoID       int64      `json:"video_id"`
	Source        string     `json:"source"`
	Provider      string     `json:"provider"`
	URL           string     `json:"url"`
	OriginalURL   string     `json:"original_url,omitempty"`
	Filename      string     `json:"filename,omitempty"`
	FileSize      string     `json:"file_size,omitempty"`
	FileType      string     `json:"file_type,omitempty"`
	Status        string     `json:"status"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	StatusCode    int        `json:"status_code,omitempty"`
	DiscoveredAt  time.Time  `json:"discovered_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ExtractDownloadLinks fetches and extracts download links from a Sexuria item page
func (s *SexuriaScraperService) ExtractDownloadLinks(itemURL string) ([]*SexuriaDownloadLink, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", itemURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

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

	var links []*SexuriaDownloadLink
	now := time.Now()

	// Known file host patterns
	hostPatterns := map[string]string{
		"rapidgator":    "rapidgator",
		"keep2share":    "keep2share",
		"k2s.cc":        "keep2share",
		"katfile":       "katfile",
		"filespace":     "filespace",
		"file.al":       "file.al",
		"nitro.download": "nitroflare",
		"nitroflare":    "nitroflare",
		"uploaded":      "uploaded",
		"turbobit":      "turbobit",
		"filejoker":     "filejoker",
		"mexashare":     "mexashare",
		"fastfile":      "fastfile",
		"rapidcloud":    "rapidcloud",
		"dropapk":       "dropapk",
	}

	// Extract download links from the page
	// Look for links in the download section
	doc.Find("a[href]").Each(func(i int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		if !exists || href == "" {
			return
		}

		// Check if URL matches known file hosts
		lowerHref := strings.ToLower(href)
		for pattern, provider := range hostPatterns {
			if strings.Contains(lowerHref, pattern) {
				// Try to extract filename from link text or URL
				filename := strings.TrimSpace(sel.Text())
				if filename == "" || len(filename) > 200 {
					// Try to extract from URL
					parts := strings.Split(href, "/")
					if len(parts) > 0 {
						filename = parts[len(parts)-1]
					}
				}

				link := &SexuriaDownloadLink{
					Source:       "sexuria",
					Provider:     provider,
					URL:          href,
					Filename:     filename,
					Status:       "unknown",
					DiscoveredAt: now,
				}
				links = append(links, link)
				break
			}
		}
	})

	return links, nil
}

// SaveDownloadLinks saves extracted download links to the database
func (s *SexuriaScraperService) SaveDownloadLinks(videoID int64, links []*SexuriaDownloadLink) error {
	for _, link := range links {
		link.VideoID = videoID

		_, err := s.db.Exec(`
			INSERT OR IGNORE INTO sexuria_download_links
			(video_id, source, provider, url, original_url, filename, file_size, file_type, status, discovered_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, link.VideoID, link.Source, link.Provider, link.URL, link.OriginalURL,
			link.Filename, link.FileSize, link.FileType, link.Status, link.DiscoveredAt)

		if err != nil {
			log.Printf("Error saving download link: %v", err)
		}
	}

	// Mark video as having links extracted
	_, err := s.db.Exec(`
		UPDATE scraped_videos
		SET links_extracted = 1, links_last_extracted_at = ?
		WHERE id = ?
	`, time.Now(), videoID)

	return err
}

// GetDownloadLinks returns all download links for a Sexuria item
func (s *SexuriaScraperService) GetDownloadLinks(videoID int64) ([]*SexuriaDownloadLink, error) {
	rows, err := s.db.Query(`
		SELECT id, video_id, source, provider, url, original_url, filename,
		       file_size, file_type, status, last_checked_at, last_status_code,
		       discovered_at, created_at
		FROM sexuria_download_links
		WHERE video_id = ?
		ORDER BY provider, created_at
	`, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*SexuriaDownloadLink
	for rows.Next() {
		link := &SexuriaDownloadLink{}
		var originalURL, filename, fileSize, fileType sql.NullString
		var lastCheckedAt sql.NullTime
		var statusCode sql.NullInt64

		err := rows.Scan(
			&link.ID, &link.VideoID, &link.Source, &link.Provider, &link.URL,
			&originalURL, &filename, &fileSize, &fileType, &link.Status,
			&lastCheckedAt, &statusCode, &link.DiscoveredAt, &link.CreatedAt,
		)
		if err != nil {
			continue
		}

		if originalURL.Valid {
			link.OriginalURL = originalURL.String
		}
		if filename.Valid {
			link.Filename = filename.String
		}
		if fileSize.Valid {
			link.FileSize = fileSize.String
		}
		if fileType.Valid {
			link.FileType = fileType.String
		}
		if lastCheckedAt.Valid {
			link.LastCheckedAt = &lastCheckedAt.Time
		}
		if statusCode.Valid {
			link.StatusCode = int(statusCode.Int64)
		}

		links = append(links, link)
	}

	return links, nil
}

// ExtractAndSaveLinks extracts links from item page and saves them
func (s *SexuriaScraperService) ExtractAndSaveLinks(videoID int64, itemURL string) ([]*SexuriaDownloadLink, error) {
	links, err := s.ExtractDownloadLinks(itemURL)
	if err != nil {
		return nil, err
	}

	if len(links) > 0 {
		if err := s.SaveDownloadLinks(videoID, links); err != nil {
			return nil, err
		}
	}

	return links, nil
}

// GetLinkStats returns link statistics for Sexuria items
func (s *SexuriaScraperService) GetLinkStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total links by status
	rows, err := s.db.Query(`
		SELECT status, COUNT(*) as count
		FROM sexuria_download_links
		GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	totalLinks := 0
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		statusCounts[status] = count
		totalLinks += count
	}

	stats["total_links"] = totalLinks
	stats["status_counts"] = statusCounts

	// Links by provider
	providerRows, err := s.db.Query(`
		SELECT provider, COUNT(*) as count
		FROM sexuria_download_links
		GROUP BY provider
		ORDER BY count DESC
	`)
	if err != nil {
		return stats, nil // Return partial stats
	}
	defer providerRows.Close()

	providerCounts := make(map[string]int)
	for providerRows.Next() {
		var provider string
		var count int
		if err := providerRows.Scan(&provider, &count); err != nil {
			continue
		}
		providerCounts[provider] = count
	}
	stats["provider_counts"] = providerCounts

	// Items with/without links extracted
	var extractedCount, notExtractedCount int
	s.db.QueryRow(`
		SELECT COUNT(*) FROM scraped_videos
		WHERE (site_type = 'sexuria' OR source = 'sexuria.net') AND links_extracted = 1
	`).Scan(&extractedCount)
	s.db.QueryRow(`
		SELECT COUNT(*) FROM scraped_videos
		WHERE (site_type = 'sexuria' OR source = 'sexuria.net') AND (links_extracted = 0 OR links_extracted IS NULL)
	`).Scan(&notExtractedCount)

	stats["items_with_links"] = extractedCount
	stats["items_without_links"] = notExtractedCount

	return stats, nil
}
