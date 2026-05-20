package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

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
			fmt.Printf("Scraping paused by user at page %d\n", page)
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
			fmt.Printf("All %d videos on page %d are duplicates (seen before)\n", len(videos), page)
			if consecutiveDuplicates >= 2 {
				fmt.Printf("Stopping pagination - same videos found on 2 consecutive pages (likely reached the end)\n")
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
			fmt.Printf("Scraping paused by user at page %d\n", page)
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
			fmt.Printf("All %d videos on page %d are duplicates (seen before)\n", len(videos), page)
			if consecutiveDuplicates >= 2 {
				fmt.Printf("Stopping pagination - same videos found on 2 consecutive pages (likely reached the end)\n")
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
		"url":             pageURL,
		"source":          source,
		"selectors_tried": []map[string]interface{}{},
		"videos_found":    []map[string]string{},
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

				fmt.Printf("  [TEST] Accepted: %s (title: %s, thumb: %v)\n", fullURL, strings.TrimSpace(title), thumbnail != "")

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
			fmt.Printf("  [TEST fallback] Accepted: %s (title: %s, thumb: %v)\n", fullURL, strings.TrimSpace(title), thumbnail != "")
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
			videoPreviewURL := ""         // For animated video previews (PornHub uses data-mediabook)
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

			fmt.Printf("  Accepted video URL: %s (title: %s)\n", fullURL, strings.TrimSpace(title))

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
					fmt.Printf("  Failed to save video: %v\n", saveErr)
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

			fmt.Printf("  [fallback] Accepted: %s (title: %s, duration: %d, preview: %v)\n",
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
		fmt.Printf("\nSummary: %d videos processed (%d new, %d updated, %d errors)\n",
			len(videos), newCount, updatedCount, errorCount)
	}

	return videos, nil
}
