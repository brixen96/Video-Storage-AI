package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var videoScraperService *services.VideoScraperService

// ensureVideoScraperService initializes the service if needed
func ensureVideoScraperService() *services.VideoScraperService {
	if videoScraperService == nil {
		videoScraperService = services.NewVideoScraperService(database.GetDB())
	}
	return videoScraperService
}

// scrapeVideoHandler handles scraping a video from a URL
func scrapeVideoHandler(c *gin.Context) {
	svc := ensureVideoScraperService()
	consoleLogSvc := ensureConsoleLogService()

	var request services.ScrapeVideoRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Validate URL
	if request.URL == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"URL is required",
			"",
		))
		return
	}

	// Log scrape start
	consoleLogSvc.LogAPI("info", "Video scraping started", map[string]interface{}{
		"url":       request.URL,
		"site_type": request.SiteType,
		"max_pages": request.Options.MaxPages,
	})

	// Determine if this is a listing page or single video based on URL pattern
	// If MaxPages is not set (0), default to unlimited pagination
	if request.Options.MaxPages == 0 {
		request.Options.MaxPages = 9999 // Effectively unlimited
	}

	// Check if URL looks like a listing page (contains /videos, /videos/, /videos/2/, etc.)
	isListingPage := false
	if parsedURL, err := url.Parse(request.URL); err == nil {
		path := parsedURL.Path
		// Listing page patterns: /videos, /videos/, /videos/2/, etc.
		if strings.Contains(path, "/videos") {
			isListingPage = true
		}
	}

	// If it's a listing page, use ScrapePage
	if isListingPage {
		result, err := svc.ScrapePage(request)
		if err != nil {
			errMsg := fmt.Sprintf("Page scraping failed: %v", err)
			consoleLogSvc.LogAPI("error", errMsg, map[string]interface{}{
				"url":   request.URL,
				"error": err.Error(),
			})
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to scrape page",
				err.Error(),
			))
			return
		}

		consoleLogSvc.LogAPI("info", "Page scraped successfully", map[string]interface{}{
			"url":           request.URL,
			"videos_found":  result.TotalScraped,
			"pages_crawled": result.PagesCrawled,
		})

		c.JSON(http.StatusOK, models.SuccessResponse(result, fmt.Sprintf("Scraped %d videos from %d pages", result.TotalScraped, result.PagesCrawled)))
		return
	}

	// Otherwise, scrape a single video
	video, err := svc.ScrapeVideo(request)
	if err != nil {
		consoleLogSvc.LogAPI("error", "Video scraping failed", map[string]interface{}{
			"url":   request.URL,
			"error": err.Error(),
		})
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to scrape video",
			err.Error(),
		))
		return
	}

	consoleLogSvc.LogAPI("info", "Video scraped successfully", map[string]interface{}{
		"url":   request.URL,
		"title": video.Title,
	})

	c.JSON(http.StatusOK, models.SuccessResponse(video, "Video scraped successfully"))
}

// getScrapedVideosHandler retrieves all scraped videos
func getScrapedVideosHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	// Check if we want to include tab info
	includeTabInfo := c.Query("includeTabInfo") == "true"

	// Check if filtering by source
	source := c.Query("source")

	var videos []*services.ScrapedVideo
	var err error

	if source != "" {
		// Filter by source for better performance
		videos, err = svc.GetScrapedVideosBySource(source)
	} else if includeTabInfo {
		videos, err = svc.GetScrapedVideosWithTabInfo()
	} else {
		videos, err = svc.GetScrapedVideos()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to retrieve scraped videos",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(videos, "Scraped videos retrieved successfully"))
}

// testScraperHandler tests the scraper on a URL without saving to database
func testScraperHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"URL parameter is required",
			"",
		))
		return
	}

	result, err := svc.TestScrapeListingPage(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to test scraper",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result, "Test completed"))
}

// deleteAllScrapedVideosHandler deletes all scraped videos from database
func deleteAllScrapedVideosHandler(c *gin.Context) {
	db := database.GetDB()

	result, err := db.Exec("DELETE FROM scraped_videos")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete scraped videos",
			err.Error(),
		))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted": rowsAffected,
	}, fmt.Sprintf("Deleted %d scraped videos", rowsAffected)))
}

// deleteScrapedVideosBySiteTypeHandler deletes all scraped videos for a specific site type
func deleteScrapedVideosBySiteTypeHandler(c *gin.Context) {
	db := database.GetDB()

	siteType := c.Param("siteType")
	if siteType == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Site type is required",
			"",
		))
		return
	}

	// Use source LIKE for sexuria since it stores multiple site_types ("sexuria" and "advanced_search")
	// This matches how items are loaded (source LIKE 'sexuria%')
	deleteQuery := "DELETE FROM scraped_videos WHERE site_type = ?"
	deleteArg := siteType
	if siteType == "sexuria" {
		// Delete related download links first
		db.Exec("DELETE FROM sexuria_download_links WHERE video_id IN (SELECT id FROM scraped_videos WHERE source LIKE 'sexuria%')")
		deleteQuery = "DELETE FROM scraped_videos WHERE source LIKE ?"
		deleteArg = "sexuria%"
	}

	result, err := db.Exec(deleteQuery, deleteArg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete scraped videos",
			err.Error(),
		))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted":   rowsAffected,
		"site_type": siteType,
	}, fmt.Sprintf("Deleted %d %s videos", rowsAffected, siteType)))
}

// deleteScrapedVideosBySourceHandler deletes all scraped videos from a specific source hostname
func deleteScrapedVideosBySourceHandler(c *gin.Context) {
	db := database.GetDB()

	source := c.Param("source")
	if source == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Source is required", ""))
		return
	}

	result, err := db.Exec("DELETE FROM scraped_videos WHERE source = ?", source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to delete scraped videos", err.Error()))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted": rowsAffected,
		"source":  source,
	}, fmt.Sprintf("Deleted %d videos from %s", rowsAffected, source)))
}

// bulkDeleteScrapedVideosHandler deletes multiple scraped videos by IDs
func bulkDeleteScrapedVideosHandler(c *gin.Context) {
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Database connection not available",
			"",
		))
		return
	}

	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"No IDs provided",
			"",
		))
		return
	}

	// Build query with placeholders
	placeholders := make([]string, len(req.IDs))
	args := make([]interface{}, len(req.IDs))
	for i, id := range req.IDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM scraped_videos WHERE id IN (%s)", strings.Join(placeholders, ","))
	result, err := db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete scraped videos",
			err.Error(),
		))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted":   rowsAffected,
		"requested": len(req.IDs),
	}, fmt.Sprintf("Deleted %d videos", rowsAffected)))
}

// findDuplicatesHandler finds duplicate scraped videos
func findDuplicatesHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	stats, err := svc.FindDuplicates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to find duplicates",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stats, fmt.Sprintf("Found %d potential duplicates", stats.TotalDuplicates)))
}

// deleteDuplicatesHandler deletes duplicate scraped videos
func deleteDuplicatesHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	// Get strategy from query param (keep_first, keep_last, keep_best)
	strategy := c.DefaultQuery("strategy", "keep_best")

	deleted, err := svc.DeleteDuplicates(strategy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete duplicates",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"deleted":  deleted,
		"strategy": strategy,
	}, fmt.Sprintf("Deleted %d duplicate videos using %s strategy", deleted, strategy)))
}

// getScrapedVideoHandler retrieves a single scraped video by ID
func getScrapedVideoHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Video ID is required",
			"",
		))
		return
	}

	// Convert ID to int64
	var videoID int64
	if _, err := fmt.Sscanf(id, "%d", &videoID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid video ID",
			err.Error(),
		))
		return
	}

	video, err := svc.GetScrapedVideoByID(videoID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Video not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(video, "Video retrieved successfully"))
}

// fetchAndParseVideoPage fetches a URL and extracts a direct video URL and any player iframe URL from the HTML.
func fetchAndParseVideoPage(pageURL string) (directURL, contentType, iframeURL string, doc *goquery.Document, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", pageURL)

	resp, fetchErr := client.Do(req)
	if fetchErr != nil {
		err = fetchErr
		return
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		err = readErr
		return
	}

	doc, err = goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return
	}

	// Try og:video meta tags first (most reliable)
	if u, ok := doc.Find("meta[property='og:video:url']").Attr("content"); ok && u != "" {
		directURL = u
		contentType = "video/mp4"
	} else if u, ok := doc.Find("meta[property='og:video']").Attr("content"); ok && u != "" {
		directURL = u
		contentType = "video/mp4"
	}

	// Try <video src="..."> or <video><source src="...">
	if directURL == "" {
		doc.Find("video").Each(func(_ int, s *goquery.Selection) {
			if directURL != "" {
				return
			}
			if src, ok := s.Attr("src"); ok && src != "" && !strings.HasPrefix(src, "blob:") {
				directURL = src
				contentType = "video/mp4"
				return
			}
			s.Find("source").Each(func(_ int, src *goquery.Selection) {
				if directURL != "" {
					return
				}
				if u, ok := src.Attr("src"); ok && u != "" && !strings.HasPrefix(u, "blob:") {
					directURL = u
					if ct, ok := src.Attr("type"); ok {
						contentType = ct
					} else {
						contentType = "video/mp4"
					}
				}
			})
		})
	}

	// Try iframes inside known player containers (e.g. MyZooWife embeds via <iframe>)
	if directURL == "" {
		playerSelectors := []string{
			"#player iframe", "#player-container iframe", ".player iframe",
			"div[id*='player'] iframe", ".video-player iframe", "#content iframe",
		}
		for _, sel := range playerSelectors {
			if src, ok := doc.Find(sel).First().Attr("src"); ok && src != "" && !strings.HasPrefix(src, "about:") {
				iframeURL = src
				break
			}
		}
	}

	return
}

// extractVideoURLHandler fetches a video page and extracts the direct MP4/video URL.
// If the page embeds via an <iframe>, it follows the iframe and extracts from there too.
// GET /api/v1/scraper/extract-video-url?url=<page_url>
func extractVideoURLHandler(c *gin.Context) {
	pageURL := c.Query("url")
	if pageURL == "" {
		respondBadRequest(c, "url query param required", nil)
		return
	}

	directURL, contentType, iframeURL, doc, err := fetchAndParseVideoPage(pageURL)
	if err != nil {
		respondInternalError(c, "failed to fetch page", err)
		return
	}

	// If no direct URL found but there's an iframe, follow it to get the real video URL
	if directURL == "" && iframeURL != "" {
		embedDirect, embedType, _, _, embedErr := fetchAndParseVideoPage(iframeURL)
		if embedErr == nil && embedDirect != "" {
			directURL = embedDirect
			contentType = embedType
			iframeURL = "" // resolved — no need to expose the intermediate iframe URL
		}
		// If embed page also failed to yield a direct URL, keep iframeURL as fallback
	}

	// Extract title: h1 → og:title → <title>
	var title string
	if doc != nil {
		if h1 := strings.TrimSpace(doc.Find("h1").First().Text()); h1 != "" {
			title = h1
		} else if t, ok := doc.Find("meta[property='og:title']").Attr("content"); ok {
			title = t
		} else {
			title = strings.TrimSpace(doc.Find("title").Text())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"direct_url":   directURL,
		"content_type": contentType,
		"iframe_url":   iframeURL,
		"title":        title,
		"page_url":     pageURL,
	})
}

// markScrapedVideoDownloadedHandler marks a scraped video as downloaded
// POST /api/v1/scraper/videos/:id/mark-downloaded
func markScrapedVideoDownloadedHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	id := c.Param("id")
	var videoID int64
	if _, err := fmt.Sscanf(id, "%d", &videoID); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid video ID", err.Error()))
		return
	}

	if err := svc.MarkScrapedVideoDownloaded(videoID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to mark video as downloaded", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{"id": videoID}, "Video marked as downloaded"))
}

// unmarkScrapedVideoDownloadedHandler clears the downloaded mark on a scraped video
// DELETE /api/v1/scraper/videos/:id/mark-downloaded
func unmarkScrapedVideoDownloadedHandler(c *gin.Context) {
	svc := ensureVideoScraperService()

	id := c.Param("id")
	var videoID int64
	if _, err := fmt.Sscanf(id, "%d", &videoID); err != nil {
		respondBadRequest(c, "invalid video ID", err)
		return
	}

	if err := svc.UnmarkScrapedVideoDownloaded(videoID); err != nil {
		respondInternalError(c, "failed to unmark video", err)
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(map[string]interface{}{"id": videoID}, "Downloaded mark removed"))
}

// proxyScraperImageHandler proxies scraped thumbnail images to bypass CDN hotlink protection.
// GET /api/v1/scraper/proxy-image?url=<encoded-url>
func proxyScraperImageHandler(c *gin.Context) {
	rawURL := c.Query("url")
	if rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter required"})
		return
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}

	referer := inferImageReferer(parsed.Host)

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") && !strings.HasPrefix(contentType, "video/") {
		contentType = "image/jpeg"
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("Content-Type", contentType)
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// inferImageReferer maps CDN hosts to the parent site's Referer for hotlink bypass.
func inferImageReferer(host string) string {
	switch {
	case strings.Contains(host, "hqporner") || strings.Contains(host, "fastporndelivery"):
		return "https://www.hqporner.com"
	case strings.Contains(host, "motherlessmedia") || strings.Contains(host, "motherless"):
		return "https://motherless.com"
	case strings.Contains(host, "leakvids"):
		return "https://leakvids.com"
	case strings.Contains(host, "phncdn") || strings.Contains(host, "pornhub"):
		return "https://www.pornhub.com"
	case strings.Contains(host, "r34xyz") || strings.Contains(host, "rule34video"):
		return "https://rule34video.com"
	case strings.Contains(host, "luxuretv"):
		return "https://luxuretv.com"
	case strings.Contains(host, "sexuria"):
		return "https://sexuria.com"
	default:
		return ""
	}
}
