package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// init registers the universal scraper tools.
func init() {
	// Ensure buildToolRegistry picks up the new register function.
}

// registerUniversalScraperTools adds AI-powered universal scraping tools.
func (s *AICompanionService) registerUniversalScraperTools() {
	videoScraperSvc := NewVideoScraperService(s.db)

	// ── Tool: scrape_url ───────────────────────────────────────────────
	// AI fetches any page, analyzes the HTML structure, extracts video
	// metadata via LLM, and saves results to the scraped_videos table.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "scrape_url",
			Description: "AI-powered universal web scraper. Fetches a URL, analyzes its HTML structure with the LLM, extracts video metadata (title, thumbnail, duration, tags, performers), and saves results to the database. Works on any website without pre-configured selectors. Use for single video pages or listing/category pages.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to scrape (single video page or listing/category page)",
					},
					"content_type": map[string]interface{}{
						"type":        "string",
						"description": "Content type tag: regular, zoo, 3d (default: regular)",
						"enum":        []string{"regular", "zoo", "3d"},
					},
					"site_label": map[string]interface{}{
						"type":        "string",
						"description": "Optional friendly name for the source (e.g. 'fapzoo', 'custom-site'). Used as the 'source' field in the database.",
					},
				},
				"required": []string{"url"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rawURL, _ := args["url"].(string)
		contentType, _ := args["content_type"].(string)
		siteLabel, _ := args["site_label"].(string)
		if rawURL == "" {
			return map[string]interface{}{"error": "url is required"}, nil
		}
		if contentType == "" {
			contentType = "regular"
		}

		// ── Step 1: Fetch HTML ──────────────────────────────────────────
		client := &http.Client{Timeout: 20 * time.Second}
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Invalid URL: %v", err)}, nil
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")

		resp, err := client.Do(req)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to fetch page: %v", err)}, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return map[string]interface{}{"error": fmt.Sprintf("HTTP %d from %s", resp.StatusCode, rawURL)}, nil
		}

		// Read up to 200KB
		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 200*1024))
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to read response: %v", err)}, nil
		}

		// ── Step 2: Parse with goquery ──────────────────────────────────
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyBytes)))
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to parse HTML: %v", err)}, nil
		}

		// Extract structural info for the LLM
		structure := s.extractPageStructure(doc, rawURL)

		// ── Step 3: Call LLM to extract video data ─────────────────────
		config := s.GetAIConfig()
		if !config.Enabled {
			return map[string]interface{}{"error": "AI is disabled in config"}, nil
		}

		extractionPrompt := buildScrapeExtractionPrompt(structure)
		llmBody := map[string]interface{}{
			"model":       config.ModelName,
			"max_tokens":  4096,
			"temperature": 0.1,
			"messages": []map[string]string{
				{"role": "system", "content": `You are an expert web scraper. Analyze the HTML structure provided and extract all video content. Return ONLY valid JSON, no explanation.

For a SINGLE VIDEO PAGE, return:
{"type":"single_video","videos":[{"title":"...","url":"video page url","thumbnail_url":"...","duration":123,"tags":["tag1"],"performers":["name1"],"category":"...","author":"...","view_count":0,"like_count":0,"rating":0}]}

For a LISTING/CATEGORY PAGE, return:
{"type":"listing","videos":[{"title":"...","url":"video page url","thumbnail_url":"...","duration":123,"tags":["tag1"],"category":"..."}]}

Rules:
- url must be the VIDEO PAGE URL (not embed or player URL)
- thumbnail_url should be the largest available image
- duration in seconds (estimate if shown as MM:SS)
- tags should be lowercase content descriptors
- performers should be real person names only
- Return empty array if no videos found
- Never return more than 50 videos`},
				{"role": "user", "content": extractionPrompt},
			},
		}
		llmBodyBytes, _ := json.Marshal(llmBody)

		llmResp, _, err := doLLMRequest(config.APIURL+"/v1/chat/completions", config.APIKey, llmBodyBytes)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("LLM request failed: %v", err)}, nil
		}

		// Parse LLM response
		var llmResult struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(llmResp, &llmResult); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to parse LLM response: %v", err)}, nil
		}
		if len(llmResult.Choices) == 0 {
			return map[string]interface{}{"error": "LLM returned empty response"}, nil
		}

		// Parse the extracted video data from LLM content
		rawContent := llmResult.Choices[0].Message.Content
		rawContent = cleanLLMJSON(rawContent)

		var extracted struct {
			Type   string `json:"type"`
			Videos []struct {
				Title        string   `json:"title"`
				URL          string   `json:"url"`
				ThumbnailURL string   `json:"thumbnail_url"`
				Duration     int      `json:"duration"`
				Tags         []string `json:"tags"`
				Performers   []string `json:"performers"`
				Category     string   `json:"category"`
				Author       string   `json:"author"`
				ViewCount    int      `json:"view_count"`
				LikeCount    int      `json:"like_count"`
				Rating       int      `json:"rating"`
			} `json:"videos"`
		}
		if err := json.Unmarshal([]byte(rawContent), &extracted); err != nil {
			return map[string]interface{}{
				"error":        fmt.Sprintf("Failed to parse LLM extraction: %v", err),
				"raw_response": truncateStr(rawContent, 500),
			}, nil
		}

		if len(extracted.Videos) == 0 {
			return map[string]interface{}{
				"success":    true,
				"url":        rawURL,
				"type":       extracted.Type,
				"videos":     0,
				"message":    "No video content found on this page",
				"structural": structure,
			}, nil
		}

		// ── Step 4: Save to database ───────────────────────────────────
		parsedURL, _ := url.Parse(rawURL)
		source := parsedURL.Host
		if siteLabel != "" {
			source = siteLabel
		}

		saved := 0
		updated := 0
		errors := []string{}

		for _, v := range extracted.Videos {
			videoURL := v.URL
			if videoURL == "" {
				continue
			}
			// Make absolute if relative
			if !strings.HasPrefix(videoURL, "http") {
				videoURL = resolveRelativeURL(rawURL, videoURL)
			}

			// Check content filter
			if ContainsBlockedContent(v.Title) || ContainsBlockedContent(v.Category) {
				continue
			}

			video := &ScrapedVideo{
				Source:       source,
				SiteType:     "ai-scraped",
				ContentType:  contentType,
				Title:        v.Title,
				URL:          videoURL,
				ThumbnailURL: v.ThumbnailURL,
				Duration:     v.Duration,
				Author:       v.Author,
				ViewCount:    v.ViewCount,
				LikeCount:    v.LikeCount,
				Rating:       v.Rating,
				Category:     v.Category,
				Tags:         v.Tags,
				Status:       "pending",
				DiscoveredAt: time.Now(),
			}

			if err := videoScraperSvc.saveVideo(video); err != nil {
				if strings.Contains(err.Error(), "Updated existing") {
					updated++
				} else {
					errors = append(errors, fmt.Sprintf("%s: %v", v.Title, err))
				}
			} else {
				saved++
			}
		}

		// Report progress
		s.ReportProgress(map[string]interface{}{
			"tool":       "scrape_url",
			"status":     "completed",
			"url":        rawURL,
			"new_videos": saved,
			"updated":    updated,
		})

		return map[string]interface{}{
			"success":       true,
			"url":           rawURL,
			"page_type":     extracted.Type,
			"videos_found":  len(extracted.Videos),
			"videos_saved":  saved,
			"videos_updated": updated,
			"source":        source,
			"errors":        errors,
		}, nil
	})

	// ── Tool: create_scraper_tab ───────────────────────────────────────
	// Creates a new scraper tab from AI chat — replaces the UI "Add Tab".
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "create_scraper_tab",
			Description: "Create a new scraper tab configuration. Use this when the user wants to add a new website to scrape, or when setting up a blank tab for a new source. The tab appears in the Scraper page and supports the generic video scraper.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Display name for the tab (e.g. 'FapZoo', 'My Site')",
					},
					"base_url": map[string]interface{}{
						"type":        "string",
						"description": "The base URL for the site's video listing (e.g. 'https://example.com/videos')",
					},
					"site_type": map[string]interface{}{
						"type":        "string",
						"description": "Site type classification (default: 'generic')",
						"enum":        []string{"generic", "youtube", "vimeo", "custom"},
					},
					"icon": map[string]interface{}{
						"type":        "string",
						"description": "FontAwesome icon name (default: 'video')",
					},
					"color": map[string]interface{}{
						"type":        "string",
						"description": "Tab accent color as hex (e.g. '#ff6600')",
					},
					"content_type": map[string]interface{}{
						"type":        "string",
						"description": "Content type: regular, zoo, 3d (default: regular)",
						"enum":        []string{"regular", "zoo", "3d"},
					},
					"page_url_template": map[string]interface{}{
						"type":        "string",
						"description": "URL template for pagination with {page} placeholder (e.g. 'https://example.com/page/{page}' or 'https://example.com/videocat/p{page}/'). Derive this from page 1 and page 2 URLs.",
					},
					"page_starts_at": map[string]interface{}{
						"type":        "number",
						"description": "The starting page number for the template (default: 1). If page 1 is the base URL, use 1.",
					},
				},
				"required": []string{"name", "base_url"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		name, _ := args["name"].(string)
		baseURL, _ := args["base_url"].(string)
		siteType, _ := args["site_type"].(string)
		icon, _ := args["icon"].(string)
		color, _ := args["color"].(string)
		contentType, _ := args["content_type"].(string)
		pageURLTemplate, _ := args["page_url_template"].(string)
		pageStartsAt := intArg(args, "page_starts_at", 1)

		if name == "" || baseURL == "" {
			return map[string]interface{}{"error": "name and base_url are required"}, nil
		}
		if siteType == "" {
			siteType = "generic"
		}
		if icon == "" {
			icon = "video"
		}
		if color == "" {
			color = "#6c757d"
		}
		if contentType == "" {
			contentType = "regular"
		}

		scrapeOpts := map[string]interface{}{
			"extractMetadata":  true,
			"scrapeThumbnails": true,
		}
		if pageURLTemplate != "" {
			scrapeOpts["pageUrlTemplate"] = pageURLTemplate
			scrapeOpts["pageStartsAt"] = pageStartsAt
		}

		tabsSvc := NewScraperTabsService(s.db)
		tab := &ScraperTab{
			Name:          name,
			TabType:       "custom",
			BaseURL:       baseURL,
			SiteType:      siteType,
			Icon:          icon,
			Color:         color,
			ContentType:   contentType,
			ScrapeOptions: scrapeOpts,
			IsActive:      true,
		}

		created, err := tabsSvc.CreateScraperTab(tab)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to create tab: %v", err)}, nil
		}

		result := map[string]interface{}{
			"success": true,
			"tab_id":  created.ID,
			"name":    created.Name,
			"url":     created.BaseURL,
			"message": fmt.Sprintf("Created scraper tab '%s' for %s", created.Name, created.BaseURL),
		}
		if pageURLTemplate != "" {
			result["page_url_template"] = pageURLTemplate
			result["page_starts_at"] = pageStartsAt
		}
		return result, nil
	})

	// ── Tool: analyze_page ─────────────────────────────────────────────
	// Lightweight page analysis without saving — returns structure info.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "analyze_page",
			Description: "Analyze a web page's structure without scraping. Returns HTML structure info (title, links, images, video embeds, content type). Useful for understanding a site before scraping.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL to analyze",
					},
				},
				"required": []string{"url"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		rawURL, _ := args["url"].(string)
		if rawURL == "" {
			return map[string]interface{}{"error": "url is required"}, nil
		}

		client := &http.Client{Timeout: 15 * time.Second}
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Invalid URL: %v", err)}, nil
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to fetch: %v", err)}, nil
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 200*1024))
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to read: %v", err)}, nil
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(bodyBytes)))
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("Failed to parse: %v", err)}, nil
		}

		structure := s.extractPageStructure(doc, rawURL)
		structure["status_code"] = resp.StatusCode
		structure["content_length"] = len(bodyBytes)

		return structure, nil
	})
}

// extractPageStructure uses goquery to pull structural info from a parsed page.
func (s *AICompanionService) extractPageStructure(doc *goquery.Document, pageURL string) map[string]interface{} {
	result := map[string]interface{}{}

	// Title
	title := doc.Find("title").Text()
	result["title"] = strings.TrimSpace(title)

	// Meta description
	if desc, exists := doc.Find("meta[name='description']").Attr("content"); exists {
		result["meta_description"] = truncateStr(strings.TrimSpace(desc), 300)
	}

	// Canonical URL
	if canonical, exists := doc.Find("link[rel='canonical']").Attr("href"); exists {
		result["canonical_url"] = canonical
	}

	// Open Graph data
	ogTags := map[string]string{}
	doc.Find("meta[property^='og:']").Each(func(_ int, sel *goquery.Selection) {
		if prop, exists := sel.Attr("property"); exists {
			if content, exists := sel.Attr("content"); exists {
				ogTags[prop] = content
			}
		}
	})
	if len(ogTags) > 0 {
		result["open_graph"] = ogTags
	}

	// Count elements
	result["link_count"] = doc.Find("a").Length()
	result["image_count"] = doc.Find("img").Length()

	// Video embeds
	videoEmbeds := []string{}
	doc.Find("video source, video[src]").Each(func(_ int, sel *goquery.Selection) {
		if src, exists := sel.Attr("src"); exists {
			videoEmbeds = append(videoEmbeds, src)
		}
	})
	doc.Find("iframe").Each(func(_ int, sel *goquery.Selection) {
		if src, exists := sel.Attr("src"); exists {
			videoEmbeds = append(videoEmbeds, src)
		}
	})
	result["video_embeds"] = videoEmbeds
	result["video_embed_count"] = len(videoEmbeds)

	// Sample links (up to 30)
	links := []map[string]string{}
	seenLinks := map[string]bool{}
	doc.Find("a[href]").Each(func(_ int, sel *goquery.Selection) {
		if len(links) >= 30 {
			return
		}
		href, _ := sel.Attr("href")
		text := strings.TrimSpace(sel.Text())
		if href == "" || text == "" || seenLinks[href] {
			return
		}
		seenLinks[href] = true
		links = append(links, map[string]string{
			"href":  truncateStr(href, 200),
			"text":  truncateStr(text, 100),
			"class": sel.AttrOr("class", ""),
		})
	})
	result["sample_links"] = links

	// Sample images (up to 15)
	images := []map[string]string{}
	doc.Find("img[src]").Each(func(_ int, sel *goquery.Selection) {
		if len(images) >= 15 {
			return
		}
		src, _ := sel.Attr("src")
		alt, _ := sel.Attr("alt")
		if src == "" {
			return
		}
		images = append(images, map[string]string{
			"src": truncateStr(src, 200),
			"alt": truncateStr(alt, 100),
		})
	})
	result["sample_images"] = images

	// Article/main content text (trimmed)
	mainContent := ""
	if main := doc.Find("article, main, .content, .video-info, .video-detail, .entry-content"); main.Length() > 0 {
		mainContent = main.Text()
	} else {
		mainContent = doc.Find("body").Text()
	}
	mainContent = cleanWhitespace(mainContent)
	if len(mainContent) > 3000 {
		mainContent = mainContent[:3000]
	}
	result["content_text"] = mainContent

	return result
}

// buildScrapeExtractionPrompt creates a concise prompt for the LLM from page structure.
func buildScrapeExtractionPrompt(structure map[string]interface{}) string {
	var b strings.Builder

	b.WriteString("Analyze this web page and extract video content.\n\n")

	if title, ok := structure["title"].(string); ok && title != "" {
		b.WriteString(fmt.Sprintf("Page Title: %s\n", title))
	}
	if desc, ok := structure["meta_description"].(string); ok && desc != "" {
		b.WriteString(fmt.Sprintf("Meta Description: %s\n", desc))
	}
	if canonical, ok := structure["canonical_url"].(string); ok && canonical != "" {
		b.WriteString(fmt.Sprintf("Canonical URL: %s\n", canonical))
	}

	// Open Graph
	if og, ok := structure["open_graph"].(map[string]string); ok && len(og) > 0 {
		b.WriteString("Open Graph:\n")
		for k, v := range og {
			b.WriteString(fmt.Sprintf("  %s: %s\n", k, truncateStr(v, 150)))
		}
	}

	// Counts
	b.WriteString(fmt.Sprintf("\nElement Counts: %d links, %d images, %d video embeds\n",
		structure["link_count"], structure["image_count"], structure["video_embed_count"]))

	// Video embeds
	if embeds, ok := structure["video_embeds"].([]string); ok && len(embeds) > 0 {
		b.WriteString("\nVideo Embeds Found:\n")
		for _, e := range embeds {
			b.WriteString(fmt.Sprintf("  - %s\n", truncateStr(e, 200)))
		}
	}

	// Sample links
	if links, ok := structure["sample_links"].([]map[string]string); ok && len(links) > 0 {
		b.WriteString("\nSample Links:\n")
		for _, l := range links {
			text := l["text"]
			href := l["href"]
			if text == "" {
				text = "(no text)"
			}
			b.WriteString(fmt.Sprintf("  [%s] → %s\n", truncateStr(text, 60), truncateStr(href, 120)))
		}
	}

	// Sample images
	if images, ok := structure["sample_images"].([]map[string]string); ok && len(images) > 0 {
		b.WriteString("\nSample Images:\n")
		for _, img := range images {
			b.WriteString(fmt.Sprintf("  src=%s alt=%s\n", truncateStr(img["src"], 100), truncateStr(img["alt"], 60)))
		}
	}

	// Content text
	if content, ok := structure["content_text"].(string); ok && content != "" {
		b.WriteString(fmt.Sprintf("\nPage Content Text:\n%s\n", truncateStr(content, 2000)))
	}

	return b.String()
}

// cleanLLMJSON strips markdown code fences from LLM JSON output.
func cleanLLMJSON(s string) string {
	s = strings.TrimSpace(s)
	// Remove ```json ... ``` wrappers
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		started := false
		var cleaned []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") && !started {
				started = true
				continue
			}
			if strings.TrimSpace(trimmed) == "```" && started {
				break
			}
			if started {
				cleaned = append(cleaned, line)
			}
		}
		s = strings.Join(cleaned, "\n")
	}
	return strings.TrimSpace(s)
}

// resolveRelativeURL resolves a relative URL against a base URL.
func resolveRelativeURL(base, ref string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}

// cleanWhitespace collapses multiple whitespace/newlines into single spaces.
func cleanWhitespace(s string) string {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(s, " ")
}

// truncateStr truncates a string to maxLen characters, appending "..." if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
