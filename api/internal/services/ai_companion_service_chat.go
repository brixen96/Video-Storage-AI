package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Chat processes user messages with rule-based intelligence
func (s *AICompanionService) Chat(message string, history []map[string]string) (string, error) {
	messageLower := strings.ToLower(message)

	// ===== CONVERSATION HISTORY CONTEXT =====
	// Check last assistant response for context-aware follow-ups
	var lastAssistantMsg string
	if len(history) > 0 {
		for i := len(history) - 1; i >= 0; i-- {
			if history[i]["role"] == "assistant" {
				lastAssistantMsg = history[i]["content"]
				break
			}
		}
	}

	// ===== FOLLOW-UP HANDLING =====
	// Use conversation history to handle follow-up questions
	if lastAssistantMsg != "" {
		lastLower := strings.ToLower(lastAssistantMsg)

		// Extract quoted terms from previous response for context
		reQuoted := regexp.MustCompile(`"([^"]+)"`)
		prevQuotedTerms := reQuoted.FindAllStringSubmatch(lastAssistantMsg, -1)

		// "List/show the forum posts" after a response that mentioned forum posts
		if (strings.Contains(messageLower, "list") || strings.Contains(messageLower, "show") || strings.Contains(messageLower, "can you list") || strings.Contains(messageLower, "can you show")) &&
			(strings.Contains(messageLower, "forum") || strings.Contains(messageLower, "post")) &&
			strings.Contains(lastLower, "forum posts") {
			// Extract performer name and search term from previous response
			re := regexp.MustCompile(`(?i)Searching "([^"]+)" \+ "([^"]+)"`)
			matches := re.FindStringSubmatch(lastAssistantMsg)
			if len(matches) >= 3 {
				return s.listForumPosts(matches[1], matches[2], 10)
			}
			if len(prevQuotedTerms) > 0 {
				return s.listForumPosts(prevQuotedTerms[0][1], "", 10)
			}
		}

		// "Is this from X scraper?" / "Is that from Simpcity?" - source questions
		if (strings.Contains(messageLower, "is this from") || strings.Contains(messageLower, "is that from") ||
			strings.Contains(messageLower, "is the") && strings.Contains(messageLower, "from") ||
			strings.Contains(messageLower, "which scraper") || strings.Contains(messageLower, "what source") ||
			strings.Contains(messageLower, "where") && (strings.Contains(messageLower, "from") || strings.Contains(messageLower, "come from"))) {

			// Check if previous response has thread/source info
			if strings.Contains(lastLower, "forum posts") || strings.Contains(lastLower, "scraped") || strings.Contains(lastLower, "thread") {
				var response strings.Builder
				response.WriteString("The forum posts are from the **Simpcity Forum Scraper**. ")
				response.WriteString("This scraper collects posts and download links from forum threads.\n\n")

				// Show which threads contained the data
				if len(prevQuotedTerms) > 0 {
					searchTerm := prevQuotedTerms[0][1]
					rows, err := s.db.Query(`
						SELECT st.title, COUNT(*) as cnt
						FROM scraped_posts sp
						JOIN scraped_threads st ON sp.thread_id = st.id
						WHERE sp.plain_text LIKE ?
						GROUP BY st.id
						ORDER BY cnt DESC LIMIT 5
					`, "%"+searchTerm+"%")
					if err == nil {
						defer rows.Close()
						response.WriteString(fmt.Sprintf("**Threads containing \"%s\":**\n", searchTerm))
						for rows.Next() {
							var title string
							var cnt int
							if rows.Scan(&title, &cnt) == nil {
								response.WriteString(fmt.Sprintf("  - %s (%d posts)\n", title, cnt))
							}
						}
					}
				}
				return response.String(), nil
			}
		}

		// "Yes" / "Yes please" / "Sure" - affirmative follow-up with context
		if messageLower == "yes" || messageLower == "yes please" || messageLower == "sure" || messageLower == "yeah" || messageLower == "more" || messageLower == "go on" || messageLower == "tell me more" {
			// If previous response mentioned forum posts, list them
			if strings.Contains(lastLower, "forum posts") {
				re := regexp.MustCompile(`(?i)Searching "([^"]+)" \+ "([^"]+)"`)
				matches := re.FindStringSubmatch(lastAssistantMsg)
				if len(matches) >= 3 {
					return s.listForumPosts(matches[1], matches[2], 10)
				}
				if len(prevQuotedTerms) > 0 {
					return s.listForumPosts(prevQuotedTerms[0][1], "", 10)
				}
			}

			// If previous response mentioned scraped content, show more
			if strings.Contains(lastLower, "scraped content") || strings.Contains(lastLower, "scraped items") {
				if len(prevQuotedTerms) > 0 {
					return s.searchScrapedContentByKeyword(prevQuotedTerms[0][1], 15)
				}
			}

			// If previous response was about a performer, show full details
			rePerfName := regexp.MustCompile(`\*\*([A-Z][a-zA-Z]+(?: [A-Z][a-zA-Z]+)+)\*\*`)
			perfMatches := rePerfName.FindStringSubmatch(lastAssistantMsg)
			if len(perfMatches) >= 2 {
				result, err := s.FindPerformer(perfMatches[1])
				if err == nil {
					scrapedInfo, _ := s.searchPerformerAcrossAll(perfMatches[1])
					if scrapedInfo != "" {
						result += "\n" + scrapedInfo
					}
					return result, nil
				}
			}
		}

		// "Can you list/show" generic follow-up
		if (strings.Contains(messageLower, "can you") || strings.Contains(messageLower, "show me") || strings.Contains(messageLower, "what are")) &&
			!strings.Contains(messageLower, "scraper") && !strings.Contains(messageLower, "forum") {
			// Try to use context from previous response
			if strings.Contains(lastLower, "scraped content") || strings.Contains(lastLower, "scraped items") {
				if len(prevQuotedTerms) > 0 {
					return s.searchScrapedContentByKeyword(prevQuotedTerms[0][1], 15)
				}
			}
			if strings.Contains(lastLower, "forum posts") {
				if len(prevQuotedTerms) > 0 {
					return s.listForumPosts(prevQuotedTerms[0][1], "", 10)
				}
			}
		}
	}

	// ===== GREETINGS =====
	if messageLower == "hi" || messageLower == "hello" || messageLower == "hey" || messageLower == "yo" || messageLower == "sup" || strings.HasPrefix(messageLower, "hello ") || strings.HasPrefix(messageLower, "hi ") || strings.HasPrefix(messageLower, "hey ") {
		var videoCount, performerCount int
		s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
		var runningCount int
		s.db.QueryRow("SELECT COUNT(*) FROM activity_logs WHERE status = 'running'").Scan(&runningCount)
		greeting := fmt.Sprintf("Hey! 👋 Your library has **%d videos** and **%d performers**.", videoCount, performerCount)
		if runningCount > 0 {
			greeting += fmt.Sprintf(" There are **%d tasks** currently running.", runningCount)
		} else {
			greeting += " Everything is idle right now."
		}
		greeting += "\n\nWhat would you like to know?"
		return greeting, nil
	}

	// ===== UPDATE / WHAT'S HAPPENING =====
	if messageLower == "update" || messageLower == "give me an update" || messageLower == "what's new" || messageLower == "whats new" || strings.Contains(messageLower, "what's happening") || strings.Contains(messageLower, "whats happening") || (strings.Contains(messageLower, "give") && strings.Contains(messageLower, "update")) || messageLower == "overview" || messageLower == "summary" {
		return s.generateAppUpdate()
	}

	// ===== SHORT CONVERSATIONAL FOLLOW-UPS (no history context) =====
	if lastAssistantMsg == "" && (messageLower == "yes" || messageLower == "yes please" || messageLower == "sure" || messageLower == "ok" || messageLower == "yeah" || messageLower == "tell me more" || messageLower == "more" || messageLower == "go on") {
		return "Could you be more specific about what you'd like to know? I can help with:\n\n" +
			"- **Performer info**: Ask about any performer by name (e.g. \"Kendra Lust\")\n" +
			"- **Library stats**: \"How many videos?\", \"Top performers\"\n" +
			"- **Scrapers**: \"Scraper summary\", \"PornHub stats\"\n" +
			"- **System**: \"Status\", \"Check for errors\"\n" +
			"- **JDownloader**: \"JDownloader status\"", nil
	}

	// ===== DIRECT PERFORMER NAME LOOKUP =====
	// If the message looks like a performer name (2-4 words, capitalized, no common keywords)
	if s.looksLikePerformerName(message) {
		result, err := s.FindPerformer(message)
		if err == nil && !strings.Contains(result, "couldn't find") {
			return result, nil
		}
		// Also search scraped content for this name
		scrapedResult, _ := s.searchPerformerAcrossAll(message)
		if scrapedResult != "" {
			return scrapedResult, nil
		}
		// Fall through to other patterns
	}

	// ===== "ALL DATA ON X" / "EVERYTHING ABOUT X" =====
	if (strings.Contains(messageLower, "all data") || strings.Contains(messageLower, "everything about") || strings.Contains(messageLower, "tell me about") || strings.Contains(messageLower, "info on") || strings.Contains(messageLower, "details on") || strings.Contains(messageLower, "details about")) {
		// Extract the name after the keyword phrase
		name := extractNameAfterPhrase(message, []string{"all data on", "everything about", "tell me about", "info on", "details on", "details about"})
		if name != "" {
			result, err := s.FindPerformer(name)
			if err == nil {
				// Also check scraped content
				scrapedInfo, _ := s.searchPerformerAcrossAll(name)
				if scrapedInfo != "" {
					result += "\n" + scrapedInfo
				}
				return result, nil
			}
		}
	}

	// ===== "DOES X HAVE ANY Y" pattern =====
	if strings.Contains(messageLower, "does") && (strings.Contains(messageLower, "have any") || strings.Contains(messageLower, "have a")) {
		// Extract performer name and search term
		re := regexp.MustCompile(`(?i)does\s+["']?(.+?)["']?\s+have\s+(?:any|a)\s+["']?(.+?)["']?\s*(?:scenes?|videos?|content)?[?]?$`)
		matches := re.FindStringSubmatch(message)
		if len(matches) >= 3 {
			performerName := strings.TrimSpace(matches[1])
			searchTerm := strings.TrimSpace(matches[2])
			return s.searchPerformerWithTag(performerName, searchTerm)
		}
	}

	// Rule-based responses for common queries
	if strings.Contains(messageLower, "how many videos") {
		var count int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&count); err != nil {
			return "", err
		}
		return fmt.Sprintf("You have %d videos in your library.", count), nil
	}

	if strings.Contains(messageLower, "how many performers") {
		var count int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&count); err != nil {
			return "", err
		}
		return fmt.Sprintf("You have %d performers in your library.", count), nil
	}

	// Top performers query
	if strings.Contains(messageLower, "top performers") || strings.Contains(messageLower, "most videos") {
		query := `
			SELECT p.name, COUNT(vp.video_id) as video_count
			FROM performers p
			LEFT JOIN video_performers vp ON p.id = vp.performer_id
			GROUP BY p.id
			ORDER BY video_count DESC
			LIMIT 10
		`
		rows, err := s.db.Query(query)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		var result strings.Builder
		result.WriteString("Top 10 Performers:\n")
		count := 1
		for rows.Next() {
			var name string
			var videoCount int
			if err := rows.Scan(&name, &videoCount); err != nil {
				continue
			}
			result.WriteString(fmt.Sprintf("%d. %s (%d videos)\n", count, name, videoCount))
			count++
		}
		return result.String(), nil
	}

	// Console log queries
	if strings.Contains(messageLower, "console log") || strings.Contains(messageLower, "logs") && strings.Contains(messageLower, "error") {
		if strings.Contains(messageLower, "error") || strings.Contains(messageLower, "analyze") {
			return s.AnalyzeConsoleErrors()
		}

		// General console log stats
		consoleLogSvc := NewConsoleLogService()
		stats, err := consoleLogSvc.GetStats()
		if err != nil {
			return "Failed to retrieve console log statistics.", err
		}

		var response strings.Builder
		response.WriteString("📋 Console Log Statistics:\n\n")

		if totalLogs, ok := stats["total_logs"].(int64); ok {
			response.WriteString(fmt.Sprintf("Total Logs: %d\n", totalLogs))
		}
		if bySource, ok := stats["by_source"].(map[string]interface{}); ok {
			response.WriteString("\nBy Source:\n")
			for source, count := range bySource {
				response.WriteString(fmt.Sprintf("- %s: %v\n", source, count))
			}
		}
		if byLevel, ok := stats["by_level"].(map[string]interface{}); ok {
			response.WriteString("\nBy Level:\n")
			for level, count := range byLevel {
				response.WriteString(fmt.Sprintf("- %s: %v\n", level, count))
			}
		}

		return response.String(), nil
	}

	// Check for errors command
	if strings.Contains(messageLower, "check") && strings.Contains(messageLower, "error") {
		return s.AnalyzeConsoleErrors()
	}

	// Recent activity from console logs
	if strings.Contains(messageLower, "recent activity") || strings.Contains(messageLower, "what happened") {
		consoleLogSvc := NewConsoleLogService()
		recentLogs, _, err := consoleLogSvc.GetAll(10, 0, "", "info", "")
		if err != nil {
			return "Failed to retrieve recent activity.", err
		}

		if len(recentLogs) == 0 {
			return "No recent activity to report.", nil
		}

		var response strings.Builder
		response.WriteString("🔍 Recent Activity:\n\n")

		for i, logEntry := range recentLogs {
			if i >= 5 {
				break
			}
			timeAgo := time.Since(logEntry.CreatedAt)
			response.WriteString(fmt.Sprintf("%d. [%s] %s (%s ago)\n",
				i+1, logEntry.Source, logEntry.Message, formatDuration(timeAgo)))
		}

		return response.String(), nil
	}

	// Search forum posts for specific keywords
	// Enhanced to handle: "Does thread X have any Y posts?", "Does X contain Y?", etc.
	if (strings.Contains(messageLower, "post") || strings.Contains(messageLower, "thread")) &&
		(strings.Contains(messageLower, "contain") || strings.Contains(messageLower, "include") ||
			strings.Contains(messageLower, "have") || strings.Contains(messageLower, "with") ||
			strings.Contains(messageLower, "mention") || strings.Contains(messageLower, "about") ||
			strings.Contains(messageLower, "does") || strings.Contains(messageLower, "any")) {

		// Extract thread name if specified
		var threadName string
		var searchKeyword string

		// Look for quoted text first (thread name or keyword)
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		if len(matches) >= 2 {
			// Two quoted texts found
			// Check which comes first - if "thread" or "scraper" comes before first quote, it's the thread name
			firstQuoteIndex := strings.Index(message, matches[0][0])
			threadIndex := strings.Index(strings.ToLower(message[:firstQuoteIndex]), "thread")
			scraperIndex := strings.Index(strings.ToLower(message[:firstQuoteIndex]), "scraper")

			if threadIndex != -1 || scraperIndex != -1 {
				// First quoted text is thread name, second is keyword
				threadName = matches[0][1]
				searchKeyword = matches[1][1]
			} else {
				// First is keyword, second might be thread name
				searchKeyword = matches[0][1]
				threadName = matches[1][1]
			}
		} else if len(matches) == 1 {
			// Only one quoted text - determine if it's thread name or keyword
			// If "thread" or "scraper" appears before the quote, it's a thread name
			quoteIndex := strings.Index(message, matches[0][0])
			preText := strings.ToLower(message[:quoteIndex])
			if strings.Contains(preText, "thread") || strings.Contains(preText, "scraper") {
				threadName = matches[0][1]
			} else {
				searchKeyword = matches[0][1]
			}
		}

		// If no quotes, try to extract keyword from common patterns
		if searchKeyword == "" {
			words := strings.Fields(message)
			for i, word := range words {
				wordLower := strings.ToLower(word)
				if (wordLower == "containing" || wordLower == "include" || wordLower == "with" ||
					wordLower == "mention" || wordLower == "about" || wordLower == "have") && i+1 < len(words) {
					nextWord := words[i+1]
					// Remove punctuation
					searchKeyword = strings.TrimRight(nextWord, "?.!")
					break
				}
			}
		}

		if searchKeyword == "" {
			return "Please specify what to search for. Try: 'how many posts contain \"BBC\"' or 'search thread \"Marsha May\" for \"IR\"'", nil
		}

		var response strings.Builder

		// If thread name specified, search within that thread
		if threadName != "" {
			threadID, threadTitle, err := s.GetThreadByName(threadName)
			if err != nil {
				return fmt.Sprintf("Thread '%s' not found. Try checking the exact thread name on the Scraper page.", threadName), nil
			}

			count, err := s.CountPostsWithKeyword(searchKeyword, threadID)
			if err != nil {
				return "Failed to search posts.", err
			}

			response.WriteString(fmt.Sprintf("🔍 Search Results\n\n"))
			response.WriteString(fmt.Sprintf("Thread: \"%s\"\n", threadTitle))
			response.WriteString(fmt.Sprintf("Keyword: \"%s\"\n\n", searchKeyword))
			response.WriteString(fmt.Sprintf("**Found %d posts** containing this keyword.\n\n", count))

			// Show sample results if count is reasonable
			if count > 0 && count <= 20 {
				results, _ := s.SearchThreadPosts(searchKeyword, threadID)
				response.WriteString("Sample matches:\n\n")
				for i, result := range results {
					if i >= 5 {
						response.WriteString(fmt.Sprintf("_... and %d more posts_\n", count-5))
						break
					}
					response.WriteString(fmt.Sprintf("**Post #%d** by %s:\n%s\n\n",
						result["post_number"], result["author"], result["excerpt"]))
				}
			}

			return response.String(), nil
		}

		// Search across all threads
		count, err := s.CountPostsWithKeyword(searchKeyword)
		if err != nil {
			return "Failed to search posts.", err
		}

		response.WriteString(fmt.Sprintf("🔍 Global Search Results\n\n"))
		response.WriteString(fmt.Sprintf("Keyword: \"%s\"\n\n", searchKeyword))
		response.WriteString(fmt.Sprintf("**Found %d posts** across all scraped threads.\n\n", count))

		if count > 0 && count <= 20 {
			results, _ := s.SearchThreadPosts(searchKeyword)
			response.WriteString("Recent matches:\n\n")
			threadCounts := make(map[string]int)
			for _, result := range results {
				threadTitle := result["thread_title"].(string)
				threadCounts[threadTitle]++
			}

			response.WriteString("Breakdown by thread:\n")
			for thread, cnt := range threadCounts {
				response.WriteString(fmt.Sprintf("• %s: %d posts\n", thread, cnt))
			}
		}

		return response.String(), nil
	}

	// ===== SEARCH SCRAPED CONTENT (broad pattern) =====
	// Handles: "search for X from scrapers", "find X in scraped content", "search scrapers for X"
	if (strings.Contains(messageLower, "search") || strings.Contains(messageLower, "find") || strings.Contains(messageLower, "look")) &&
		(strings.Contains(messageLower, "scraper") || strings.Contains(messageLower, "scraped") || strings.Contains(messageLower, "from the scraper")) {

		// Extract keyword - try quotes first
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)
		keyword := ""
		if len(matches) > 0 {
			keyword = matches[0][1]
		}

		// If no quotes, extract the search subject
		if keyword == "" {
			// Try "search for X from", "find X from", "search X from"
			reFor := regexp.MustCompile(`(?i)(?:search|find|look)\s+(?:for\s+)?(.+?)\s+(?:from|in|on)\s+(?:the\s+)?scrap`)
			m := reFor.FindStringSubmatch(message)
			if len(m) >= 2 {
				keyword = strings.TrimSpace(m[1])
			}
		}

		// Try "search scrapers for X"
		if keyword == "" {
			reAfter := regexp.MustCompile(`(?i)(?:search|find|look)\s+(?:the\s+)?scrapers?\s+(?:for\s+)(.+)`)
			m := reAfter.FindStringSubmatch(message)
			if len(m) >= 2 {
				keyword = strings.TrimSpace(m[1])
			}
		}

		// Last resort: take everything after "search for" or "find"
		if keyword == "" {
			words := strings.Fields(message)
			for i, w := range words {
				wl := strings.ToLower(w)
				if (wl == "for" || wl == "find" || wl == "search") && i+1 < len(words) {
					remaining := strings.Join(words[i+1:], " ")
					// Remove "from scrapers", "in scraped", etc.
					remaining = regexp.MustCompile(`(?i)\s*(from|in|on)\s+(the\s+)?scrap\w+\s*$`).ReplaceAllString(remaining, "")
					remaining = strings.TrimSpace(remaining)
					if remaining != "" && remaining != "scrapers" && remaining != "scraped" {
						keyword = remaining
					}
					break
				}
			}
		}

		if keyword != "" {
			// Remove trailing "videos" or "content" from keyword
			keyword = regexp.MustCompile(`(?i)\s*(videos?|content|items?)\s*$`).ReplaceAllString(keyword, "")
			keyword = strings.TrimSpace(keyword)
			if keyword != "" {
				return s.searchScrapedContentByKeyword(keyword, 15)
			}
		}
	}

	// Scraper stats (only for stats/summary requests, not searches)
	if (strings.Contains(messageLower, "scraper") && (strings.Contains(messageLower, "stat") || strings.Contains(messageLower, "summary") || strings.Contains(messageLower, "overview") || messageLower == "scraper" || messageLower == "scrapers")) ||
		(strings.Contains(messageLower, "scraped") && !strings.Contains(messageLower, "search") && !strings.Contains(messageLower, "find")) ||
		(messageLower == "forum" || messageLower == "forum stats" || (strings.Contains(messageLower, "forum") && strings.Contains(messageLower, "stat"))) {
		var threadCount, postCount, linkCount, activeLinks int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads").Scan(&threadCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_posts").Scan(&postCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links").Scan(&linkCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_download_links WHERE status = 'active'").Scan(&activeLinks)

		// Get video scraper stats
		var totalScrapedVideos, pornhubCount, luxuretvCount, sexuriaCount int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos").Scan(&totalScrapedVideos)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'pornhub'").Scan(&pornhubCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'luxuretv'").Scan(&luxuretvCount)
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'sexuria'").Scan(&sexuriaCount)

		var response strings.Builder
		response.WriteString("🕷️ Web Scraper Statistics:\n\n")
		response.WriteString("**Forum Scraper:**\n")
		response.WriteString(fmt.Sprintf("📰 Threads Scraped: %d\n", threadCount))
		response.WriteString(fmt.Sprintf("💬 Posts Collected: %d\n", postCount))
		response.WriteString(fmt.Sprintf("🔗 Download Links: %d\n", linkCount))
		response.WriteString(fmt.Sprintf("✅ Active Links: %d\n\n", activeLinks))

		if totalScrapedVideos > 0 {
			response.WriteString("**Video Scrapers:**\n")
			response.WriteString(fmt.Sprintf("🎬 Total Scraped Videos: %d\n", totalScrapedVideos))
			if pornhubCount > 0 {
				response.WriteString(fmt.Sprintf("   • PornHub: %d videos\n", pornhubCount))
			}
			if luxuretvCount > 0 {
				response.WriteString(fmt.Sprintf("   • LuxureTV: %d videos\n", luxuretvCount))
			}
			if sexuriaCount > 0 {
				response.WriteString(fmt.Sprintf("   • Sexuria: %d items\n", sexuriaCount))
			}
			response.WriteString("\n")
		}

		response.WriteString("💡 Tip: Ask me about 'pornhub videos', 'luxuretv stats', or 'sexuria items' for detailed info!")
		return response.String(), nil
	}

	// PornHub specific queries
	if strings.Contains(messageLower, "pornhub") {
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'pornhub'").Scan(&count)

		if count == 0 {
			return "No PornHub videos have been scraped yet. Use the Scraper page to start scraping PornHub videos.", nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🟠 **PornHub Scraped Videos: %d**\n\n", count))

		// Get top categories
		categoryRows, err := s.db.Query(`
			SELECT site_type, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'pornhub'
			GROUP BY site_type
			ORDER BY cnt DESC
			LIMIT 5
		`)
		if err == nil {
			defer categoryRows.Close()
			response.WriteString("**Top Categories:**\n")
			for categoryRows.Next() {
				var siteType string
				var cnt int
				if categoryRows.Scan(&siteType, &cnt) == nil {
					response.WriteString(fmt.Sprintf("• %s: %d videos\n", siteType, cnt))
				}
			}
			response.WriteString("\n")
		}

		// Get top performers from scraped videos
		performerRows, err := s.db.Query(`
			SELECT performers, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'pornhub' AND performers != '' AND performers IS NOT NULL
			GROUP BY performers
			ORDER BY cnt DESC
			LIMIT 5
		`)
		if err == nil {
			defer performerRows.Close()
			response.WriteString("**Top Performers:**\n")
			hasPerformers := false
			for performerRows.Next() {
				var performers string
				var cnt int
				if performerRows.Scan(&performers, &cnt) == nil && performers != "" {
					hasPerformers = true
					response.WriteString(fmt.Sprintf("• %s: %d videos\n", performers, cnt))
				}
			}
			if !hasPerformers {
				response.WriteString("No performer data collected yet.\n")
			}
			response.WriteString("\n")
		}

		// Get average rating and views
		var avgRating float64
		var avgViews float64
		s.db.QueryRow("SELECT AVG(rating), AVG(views) FROM scraped_videos WHERE source = 'pornhub' AND rating > 0").Scan(&avgRating, &avgViews)
		if avgRating > 0 {
			response.WriteString(fmt.Sprintf("**Stats:**\n• Avg Rating: %.1f%%\n", avgRating))
			if avgViews > 0 {
				response.WriteString(fmt.Sprintf("• Avg Views: %.0f\n", avgViews))
			}
		}

		return response.String(), nil
	}

	// LuxureTV specific queries
	if strings.Contains(messageLower, "luxuretv") || strings.Contains(messageLower, "luxure") {
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'luxuretv'").Scan(&count)

		if count == 0 {
			return "No LuxureTV videos have been scraped yet. Use the Scraper page to start scraping LuxureTV videos.", nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🎬 **LuxureTV Scraped Videos: %d**\n\n", count))

		// Get categories breakdown
		categoryRows, err := s.db.Query(`
			SELECT site_type, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'luxuretv'
			GROUP BY site_type
			ORDER BY cnt DESC
			LIMIT 10
		`)
		if err == nil {
			defer categoryRows.Close()
			response.WriteString("**Categories:**\n")
			for categoryRows.Next() {
				var siteType string
				var cnt int
				if categoryRows.Scan(&siteType, &cnt) == nil {
					response.WriteString(fmt.Sprintf("• %s: %d videos\n", siteType, cnt))
				}
			}
			response.WriteString("\n")
		}

		// Get content type breakdown
		contentRows, err := s.db.Query(`
			SELECT COALESCE(content_type, 'regular') as ct, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'luxuretv'
			GROUP BY ct
			ORDER BY cnt DESC
		`)
		if err == nil {
			defer contentRows.Close()
			response.WriteString("**Content Types:**\n")
			for contentRows.Next() {
				var contentType string
				var cnt int
				if contentRows.Scan(&contentType, &cnt) == nil {
					response.WriteString(fmt.Sprintf("• %s: %d videos\n", contentType, cnt))
				}
			}
		}

		return response.String(), nil
	}

	// Sexuria specific queries
	if strings.Contains(messageLower, "sexuria") {
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'sexuria'").Scan(&count)

		if count == 0 {
			return "No Sexuria items have been scraped yet. Use the Scraper page to start scraping Sexuria galleries.", nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🔞 **Sexuria Scraped Items: %d**\n\n", count))

		// Get top pornstars
		performerRows, err := s.db.Query(`
			SELECT performers, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'sexuria' AND performers != '' AND performers IS NOT NULL
			GROUP BY performers
			ORDER BY cnt DESC
			LIMIT 10
		`)
		if err == nil {
			defer performerRows.Close()
			response.WriteString("**Top Performers:**\n")
			hasPerformers := false
			for performerRows.Next() {
				var performers string
				var cnt int
				if performerRows.Scan(&performers, &cnt) == nil && performers != "" {
					hasPerformers = true
					response.WriteString(fmt.Sprintf("• %s: %d galleries\n", performers, cnt))
				}
			}
			if !hasPerformers {
				response.WriteString("Performer data included in item details.\n")
			}
			response.WriteString("\n")
		}

		// Get top tags
		tagRows, err := s.db.Query(`
			SELECT tags, COUNT(*) as cnt
			FROM scraped_videos
			WHERE source = 'sexuria' AND tags != '' AND tags IS NOT NULL
			GROUP BY tags
			ORDER BY cnt DESC
			LIMIT 10
		`)
		if err == nil {
			defer tagRows.Close()
			response.WriteString("**Popular Tags:**\n")
			hasTags := false
			for tagRows.Next() {
				var tags string
				var cnt int
				if tagRows.Scan(&tags, &cnt) == nil && tags != "" {
					hasTags = true
					// Show first few tags if it's a comma-separated list
					if len(tags) > 50 {
						tags = tags[:47] + "..."
					}
					response.WriteString(fmt.Sprintf("• %s (%d)\n", tags, cnt))
				}
			}
			if !hasTags {
				response.WriteString("Tag data included in item details.\n")
			}
		}

		return response.String(), nil
	}

	// Search scraped videos
	if (strings.Contains(messageLower, "search") || strings.Contains(messageLower, "find")) &&
		(strings.Contains(messageLower, "scraped video") || strings.Contains(messageLower, "scraped content")) {

		// Extract search term from quotes
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		searchTerm := ""
		if len(matches) > 0 {
			searchTerm = matches[0][1]
		}

		if searchTerm == "" {
			return "Please specify what to search for. Try: 'search scraped videos for \"keyword\"'", nil
		}

		rows, err := s.db.Query(`
			SELECT source, title, performers, url
			FROM scraped_videos
			WHERE title LIKE ? OR performers LIKE ? OR tags LIKE ?
			ORDER BY discovered_at DESC
			LIMIT 10
		`, "%"+searchTerm+"%", "%"+searchTerm+"%", "%"+searchTerm+"%")

		if err != nil {
			return "Failed to search scraped videos.", err
		}
		defer rows.Close()

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🔍 **Search Results for '%s':**\n\n", searchTerm))

		count := 0
		for rows.Next() {
			var source, title, performers, url string
			if rows.Scan(&source, &title, &performers, &url) == nil {
				count++
				response.WriteString(fmt.Sprintf("%d. [%s] %s\n", count, strings.ToUpper(source), title))
				if performers != "" {
					response.WriteString(fmt.Sprintf("   Performers: %s\n", performers))
				}
			}
		}

		if count == 0 {
			return fmt.Sprintf("No scraped videos found matching '%s'.", searchTerm), nil
		}

		response.WriteString(fmt.Sprintf("\nFound %d results.", count))
		return response.String(), nil
	}

	// Videos by performer query
	if strings.Contains(messageLower, "video") && (strings.Contains(messageLower, "by") || strings.Contains(messageLower, "with") || strings.Contains(messageLower, "featuring")) &&
		(strings.Contains(messageLower, "performer") || strings.Contains(messageLower, "actor") || strings.Contains(messageLower, "model")) {

		// Extract performer name from quotes or after "by/with/featuring"
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		performerName := ""
		if len(matches) > 0 {
			performerName = matches[0][1]
		} else {
			words := strings.Fields(message)
			for i, word := range words {
				wordLower := strings.ToLower(word)
				if (wordLower == "by" || wordLower == "with" || wordLower == "featuring") && i+1 < len(words) {
					performerName = words[i+1]
					break
				}
			}
		}

		if performerName == "" {
			return "Please specify a performer name. Try: 'videos by \"Performer Name\"'", nil
		}

		query := `
			SELECT v.id, v.title, v.path
			FROM videos v
			JOIN video_performers vp ON v.id = vp.video_id
			JOIN performers p ON vp.performer_id = p.id
			WHERE p.name LIKE ?
			ORDER BY v.created_at DESC
			LIMIT 10
		`

		rows, err := s.db.Query(query, "%"+performerName+"%")
		if err != nil {
			return "Failed to search videos by performer.", err
		}
		defer rows.Close()

		var results []string
		count := 0
		for rows.Next() {
			var id int64
			var title, path string
			if err := rows.Scan(&id, &title, &path); err != nil {
				continue
			}
			results = append(results, fmt.Sprintf("- %s (ID: %d)", title, id))
			count++
		}

		if count == 0 {
			return fmt.Sprintf("No videos found featuring '%s'. Check the performer name and try again.", performerName), nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🎬 Videos featuring '%s' (%d results):\n\n", performerName, count))
		for _, result := range results {
			response.WriteString(result + "\n")
		}
		if count >= 10 {
			response.WriteString("\n(Showing first 10 results)")
		}

		return response.String(), nil
	}

	// Videos by tag query
	if strings.Contains(messageLower, "video") && (strings.Contains(messageLower, "tagged") || strings.Contains(messageLower, "tag")) {
		// Extract tag name from quotes or after "tagged/tag"
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		tagName := ""
		if len(matches) > 0 {
			tagName = matches[0][1]
		}

		if tagName == "" {
			return "Please specify a tag name. Try: 'videos tagged \"BBC\"'", nil
		}

		query := `
			SELECT v.id, v.title, COUNT(vt.tag_id) as tag_count
			FROM videos v
			JOIN video_tags vt ON v.id = vt.video_id
			JOIN tags t ON vt.tag_id = t.id
			WHERE t.name LIKE ?
			GROUP BY v.id
			ORDER BY v.created_at DESC
			LIMIT 10
		`

		rows, err := s.db.Query(query, "%"+tagName+"%")
		if err != nil {
			return "Failed to search videos by tag.", err
		}
		defer rows.Close()

		var results []string
		count := 0
		for rows.Next() {
			var id int64
			var title string
			var tagCount int
			if err := rows.Scan(&id, &title, &tagCount); err != nil {
				continue
			}
			results = append(results, fmt.Sprintf("- %s (ID: %d)", title, id))
			count++
		}

		if count == 0 {
			return fmt.Sprintf("No videos found with tag '%s'.", tagName), nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🏷️ Videos tagged '%s' (%d results):\n\n", tagName, count))
		for _, result := range results {
			response.WriteString(result + "\n")
		}
		if count >= 10 {
			response.WriteString("\n(Showing first 10 results)")
		}

		return response.String(), nil
	}

	// Search for videos
	if (strings.Contains(messageLower, "find") || strings.Contains(messageLower, "search")) && strings.Contains(messageLower, "video") {
		// Extract search term (simple extraction - everything after "for" or "video")
		words := strings.Fields(message)
		searchTerm := ""
		for i, word := range words {
			if strings.ToLower(word) == "for" && i+1 < len(words) {
				searchTerm = strings.Join(words[i+1:], " ")
				break
			} else if strings.ToLower(word) == "video" && i+1 < len(words) {
				searchTerm = strings.Join(words[i+1:], " ")
				break
			}
		}

		if searchTerm == "" {
			return "Please specify what video you're looking for. Try: 'search for video [name]'", nil
		}

		results, err := s.SearchVideos(searchTerm)
		if err != nil {
			return "Failed to search videos.", err
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("Search results for '%s':\n\n", searchTerm))
		for _, result := range results {
			response.WriteString(result + "\n")
		}

		return response.String(), nil
	}

	// Find performer
	if (strings.Contains(messageLower, "find") || strings.Contains(messageLower, "search") || strings.Contains(messageLower, "who is")) && strings.Contains(messageLower, "performer") {
		// Extract performer name
		words := strings.Fields(message)
		performerName := ""
		for i, word := range words {
			if (strings.ToLower(word) == "performer" || strings.ToLower(word) == "is") && i+1 < len(words) {
				performerName = strings.Join(words[i+1:], " ")
				break
			}
		}

		if performerName == "" {
			return "Please specify which performer you're looking for. Try: 'find performer [name]'", nil
		}

		return s.FindPerformer(performerName)
	}

	// Recent videos
	if strings.Contains(messageLower, "recent") && strings.Contains(messageLower, "video") {
		results, err := s.GetRecentVideos(10)
		if err != nil {
			return "Failed to retrieve recent videos.", err
		}

		var response strings.Builder
		response.WriteString("📅 Recently Added Videos:\n\n")
		for _, result := range results {
			response.WriteString(result + "\n")
		}

		return response.String(), nil
	}

	// Tagged videos
	if strings.Contains(messageLower, "tag") && (strings.Contains(messageLower, "video") || strings.Contains(messageLower, "with")) {
		// Extract tag name
		words := strings.Fields(message)
		tagName := ""
		for i, word := range words {
			if strings.ToLower(word) == "tag" && i+1 < len(words) {
				tagName = strings.Join(words[i+1:], " ")
				tagName = strings.TrimSuffix(tagName, " videos")
				tagName = strings.TrimSuffix(tagName, " video")
				break
			}
		}

		if tagName == "" {
			return "Please specify which tag you're looking for. Try: 'videos with tag [name]'", nil
		}

		results, err := s.GetTaggedVideos(tagName)
		if err != nil {
			return "Failed to search tagged videos.", err
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("Videos tagged '%s':\n\n", tagName))
		for _, result := range results {
			response.WriteString(result + "\n")
		}

		return response.String(), nil
	}

	// Storage analysis
	if strings.Contains(messageLower, "storage") || strings.Contains(messageLower, "disk space") || strings.Contains(messageLower, "how much space") {
		return s.AnalyzeStorageUsage()
	}

	// Pattern detection
	if strings.Contains(messageLower, "pattern") || strings.Contains(messageLower, "usage") && strings.Contains(messageLower, "habit") {
		patterns, err := s.DetectUsagePatterns()
		if err != nil {
			return "Failed to analyze usage patterns.", err
		}

		if len(patterns) == 0 {
			return "I haven't detected any specific usage patterns yet. Keep using the system and I'll learn your preferences!", nil
		}

		var response strings.Builder
		response.WriteString("🧠 Detected Usage Patterns:\n\n")
		for i, pattern := range patterns {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, pattern))
		}

		return response.String(), nil
	}

	// Next action suggestion
	if strings.Contains(messageLower, "what should i do") || strings.Contains(messageLower, "what next") || strings.Contains(messageLower, "next step") {
		return s.SuggestNextAction()
	}

	// Duplicate detection
	if strings.Contains(messageLower, "duplicate") || strings.Contains(messageLower, "find duplicates") {
		duplicates, err := s.DetectPotentialDuplicates()
		if err != nil {
			return "Failed to detect duplicates.", err
		}

		var response strings.Builder
		response.WriteString("🔍 Duplicate Detection Results:\n\n")
		for i, dup := range duplicates {
			if i >= 10 {
				response.WriteString(fmt.Sprintf("\n... and %d more potential duplicates", len(duplicates)-10))
				break
			}
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, dup))
		}

		return response.String(), nil
	}

	// Library health check
	if strings.Contains(messageLower, "health") || strings.Contains(messageLower, "health check") || strings.Contains(messageLower, "health score") {
		return s.AnalyzeLibraryHealth()
	}

	// Content recommendations
	if strings.Contains(messageLower, "recommend") && !strings.Contains(messageLower, "optimization") {
		recommendations, err := s.RecommendContent()
		if err != nil {
			return "Failed to generate recommendations.", err
		}

		var response strings.Builder
		response.WriteString("👍 Personalized Recommendations:\n\n")
		for _, rec := range recommendations {
			response.WriteString(fmt.Sprintf("• %s\n", rec))
		}

		return response.String(), nil
	}

	// Maintenance schedule
	if strings.Contains(messageLower, "maintenance") || strings.Contains(messageLower, "schedule") || strings.Contains(messageLower, "task") && strings.Contains(messageLower, "list") {
		tasks, err := s.ScheduleAutomatedTasks()
		if err != nil {
			return "Failed to generate maintenance schedule.", err
		}

		var response strings.Builder
		response.WriteString("🗓️ Recommended Maintenance Tasks:\n\n")
		for i, task := range tasks {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, task))
		}
		response.WriteString("\nTip: I'll automatically remind you when these tasks become important!")

		return response.String(), nil
	}

	// Enhanced performer information query with scraped threads
	if (strings.Contains(messageLower, "performer") || strings.Contains(messageLower, "about")) &&
		(strings.Contains(messageLower, "information") || strings.Contains(messageLower, "details") ||
			strings.Contains(messageLower, "tell me about") || strings.Contains(messageLower, "info")) {

		// Extract performer name from quotes or after keywords
		re := regexp.MustCompile(`["']([^"']+)["']`)
		matches := re.FindAllStringSubmatch(message, -1)

		performerName := ""
		if len(matches) > 0 {
			performerName = matches[0][1]
		} else {
			words := strings.Fields(message)
			for i, word := range words {
				wordLower := strings.ToLower(word)
				if (wordLower == "performer" || wordLower == "about") && i+1 < len(words) {
					performerName = words[i+1]
					break
				}
			}
		}

		if performerName == "" {
			return "Please specify a performer name. Try: 'tell me about performer \"Name\"'", nil
		}

		// Get performer details including metadata and scraped threads
		query := `
			SELECT p.id, p.name, p.metadata, p.preview_path, p.thumbnail_path,
				COUNT(DISTINCT vp.video_id) as video_count,
				COUNT(DISTINCT pst.thread_id) as thread_count
			FROM performers p
			LEFT JOIN video_performers vp ON p.id = vp.performer_id
			LEFT JOIN performer_scraped_threads pst ON p.id = pst.performer_id
			WHERE p.name LIKE ?
			GROUP BY p.id
			LIMIT 1
		`

		var id int64
		var name, metadata, previewPath, thumbnailPath string
		var videoCount, threadCount int

		err := s.db.QueryRow(query, "%"+performerName+"%").Scan(
			&id, &name, &metadata, &previewPath, &thumbnailPath,
			&videoCount, &threadCount)

		if err != nil {
			return fmt.Sprintf("I couldn't find information about performer '%s'. Try checking the exact name on the Performers page.", performerName), nil
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("👤 **%s**\n\n", name))
		response.WriteString(fmt.Sprintf("📹 Videos: %d\n", videoCount))
		response.WriteString(fmt.Sprintf("🕷️ Scraped Threads: %d\n", threadCount))

		// Parse and display metadata if available
		if metadata != "" && metadata != "{}" {
			var metaMap map[string]interface{}
			if err := json.Unmarshal([]byte(metadata), &metaMap); err == nil {
				if age, ok := metaMap["age"].(float64); ok && age > 0 {
					response.WriteString(fmt.Sprintf("🎂 Age: %.0f\n", age))
				}
				if birthdate, ok := metaMap["birthdate"].(string); ok && birthdate != "" {
					response.WriteString(fmt.Sprintf("📅 Birthdate: %s\n", birthdate))
				}
				if height, ok := metaMap["height"].(string); ok && height != "" {
					response.WriteString(fmt.Sprintf("📏 Height: %s\n", height))
				}
				if breast, ok := metaMap["breast"].(string); ok && breast != "" {
					response.WriteString(fmt.Sprintf("💎 Breast: %s\n", breast))
				}
			}
		}

		if previewPath != "" {
			response.WriteString("✅ Has preview video\n")
		}
		if thumbnailPath != "" {
			response.WriteString("✅ Has thumbnail\n")
		}

		// Get scraped thread titles if any
		if threadCount > 0 {
			threadQuery := `
				SELECT st.title
				FROM scraped_threads st
				JOIN performer_scraped_threads pst ON st.id = pst.thread_id
				WHERE pst.performer_id = ?
				LIMIT 5
			`
			rows, err := s.db.Query(threadQuery, id)
			if err == nil {
				defer rows.Close()
				response.WriteString("\n📋 Recent Scraped Threads:\n")
				count := 0
				for rows.Next() {
					var title string
					if rows.Scan(&title) == nil {
						count++
						response.WriteString(fmt.Sprintf("%d. %s\n", count, title))
					}
				}
				if threadCount > 5 {
					response.WriteString(fmt.Sprintf("... and %d more\n", threadCount-5))
				}
			}
		}

		return response.String(), nil
	}

	// Thumbnail status queries
	if strings.Contains(messageLower, "thumbnail") && (strings.Contains(messageLower, "status") || strings.Contains(messageLower, "coverage")) {
		var performersWithThumbnails, performersWithPreviews, performersWithoutThumbnails int
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&performersWithThumbnails)
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&performersWithPreviews)
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != '' AND (thumbnail_path IS NULL OR thumbnail_path = '')").Scan(&performersWithoutThumbnails)

		coverage := float64(0)
		if performersWithPreviews > 0 {
			coverage = float64(performersWithThumbnails) / float64(performersWithPreviews) * 100
		}

		var response strings.Builder
		response.WriteString("📸 Performer Thumbnail Status:\n")
		response.WriteString(fmt.Sprintf("- Performers with thumbnails: %d/%d (%.1f%%)\n", performersWithThumbnails, performersWithPreviews, coverage))
		if performersWithoutThumbnails > 0 {
			response.WriteString(fmt.Sprintf("- ⚡ Missing thumbnails: %d performers\n", performersWithoutThumbnails))
			response.WriteString("\nTip: Generate thumbnails on the Tasks page to improve page load performance!")
		} else {
			response.WriteString("\n✅ All performers with previews have thumbnails!")
		}
		return response.String(), nil
	}

	// Optimization suggestions
	if strings.Contains(messageLower, "optimize") || strings.Contains(messageLower, "suggestion") || strings.Contains(messageLower, "recommend") {
		suggestions, err := s.SuggestOptimizations()
		if err != nil {
			return "Failed to analyze system for optimizations.", err
		}

		var response strings.Builder
		response.WriteString("💡 System Optimization Suggestions:\n\n")
		for i, suggestion := range suggestions {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion))
		}
		response.WriteString("\nTip: Visit the Tasks page to execute these optimizations!")

		return response.String(), nil
	}

	// Performance suggestions
	if strings.Contains(messageLower, "performance") || strings.Contains(messageLower, "slow") || strings.Contains(messageLower, "faster") {
		var performersWithoutThumbnails int
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != '' AND (thumbnail_path IS NULL OR thumbnail_path = '')").Scan(&performersWithoutThumbnails)

		if performersWithoutThumbnails > 0 {
			msg := fmt.Sprintf("I've detected a performance optimization opportunity! You have %d performers with preview videos but no thumbnails. Generating thumbnails will significantly speed up the Performers page. Head to the Tasks page and click 'Generate Performer Thumbnails' to improve performance.", performersWithoutThumbnails)

			// Track this recommendation so we can react when user follows it
			s.trackRecommendation(
				"chat_performance_suggestion",
				"performer_thumbnail_generation",
				msg,
				map[string]interface{}{
					"performers_without_thumbnails": performersWithoutThumbnails,
					"suggested_via":                 "chat",
				},
			)

			return msg, nil
		}
		return "Your library is well-optimized! All performers with previews have thumbnails for fast loading.", nil
	}

	// Library statistics
	if strings.Contains(messageLower, "library stats") || strings.Contains(messageLower, "library statistics") || strings.Contains(messageLower, "overview") {
		var videoCount, performerCount, studioCount, tagCount int
		var performersWithThumbnails, performersWithPreviews int
		var videosWithPreviews int

		s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)
		s.db.QueryRow("SELECT COUNT(*) FROM studios").Scan(&studioCount)
		s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&tagCount)
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE thumbnail_path IS NOT NULL AND thumbnail_path != ''").Scan(&performersWithThumbnails)
		s.db.QueryRow("SELECT COUNT(*) FROM performers WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&performersWithPreviews)
		s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE preview_path IS NOT NULL AND preview_path != ''").Scan(&videosWithPreviews)

		thumbnailCoverage := float64(0)
		if performersWithPreviews > 0 {
			thumbnailCoverage = float64(performersWithThumbnails) / float64(performersWithPreviews) * 100
		}
		videoPreviewCoverage := float64(0)
		if videoCount > 0 {
			videoPreviewCoverage = float64(videosWithPreviews) / float64(videoCount) * 100
		}

		var response strings.Builder
		response.WriteString("📊 Library Overview:\n\n")
		response.WriteString("Content:\n")
		response.WriteString(fmt.Sprintf("- Videos: %d\n", videoCount))
		response.WriteString(fmt.Sprintf("- Performers: %d\n", performerCount))
		response.WriteString(fmt.Sprintf("- Studios: %d\n", studioCount))
		response.WriteString(fmt.Sprintf("- Tags: %d\n\n", tagCount))
		response.WriteString("Media Coverage:\n")
		response.WriteString(fmt.Sprintf("- Video previews: %.1f%%\n", videoPreviewCoverage))
		response.WriteString(fmt.Sprintf("- Performer thumbnails: %.1f%%\n", thumbnailCoverage))

		return response.String(), nil
	}

	// Predict library growth
	if strings.Contains(messageLower, "predict") || strings.Contains(messageLower, "growth") || strings.Contains(messageLower, "library growth") {
		return s.PredictLibraryGrowth()
	}

	// Performer quality analysis
	if strings.Contains(messageLower, "quality") && strings.Contains(messageLower, "performer") || strings.Contains(messageLower, "performer quality") {
		return s.AnalyzePerformerQuality()
	}

	// Generate insights
	if strings.Contains(messageLower, "insight") || strings.Contains(messageLower, "show insights") {
		insights, err := s.GenerateInsights()
		if err != nil {
			return "Failed to generate insights.", err
		}

		var response strings.Builder
		response.WriteString("💡 AI-Powered Insights:\n\n")
		for i, insight := range insights {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, insight))
		}

		if len(insights) == 0 {
			response.WriteString("I haven't gathered enough data to provide insights yet. Keep using the system!")
		}

		return response.String(), nil
	}

	// AI Task Commands - Auto-link performers
	if strings.Contains(messageLower, "auto") && (strings.Contains(messageLower, "link") || strings.Contains(messageLower, "performers")) {
		return `## 🔗 Auto-Link Performers

I can automatically detect and link performers to videos based on filename analysis!

**What I do:**
- Scan video filenames for performer names
- Calculate confidence scores for each match
- Auto-apply 100% confidence matches
- Provide suggestions for manual review

**To run this task:**
Navigate to the **AI Page** and click "Start Analysis" in the "Auto-Link Performers" card.

**Expected results:**
- Videos analyzed
- Matches found with confidence scores
- Automatic links applied for high-confidence matches

Would you like me to explain any other AI capabilities?`, nil
	}

	// AI Task Commands - Smart Tagging
	if strings.Contains(messageLower, "smart") && strings.Contains(messageLower, "tag") || strings.Contains(messageLower, "suggest tags") {
		return `## 🏷️ Smart Tagging

I can suggest relevant tags for your videos based on intelligent analysis!

**What I analyze:**
- Video titles and filenames
- Performer information
- Existing tag patterns
- Content keywords

**To run this task:**
Go to the **AI Page** and click "Start Analysis" in the "Smart Tagging" card.

**Features:**
- High confidence tags (85%+) auto-applied
- Tag suggestions with reasoning
- Bulk tag application

Try asking: "What tags should I use?" or "How do I organize my videos with tags?"`, nil
	}

	// AI Task Commands - Duplicate Detection
	if (strings.Contains(messageLower, "find") || strings.Contains(messageLower, "detect")) && strings.Contains(messageLower, "duplicate") {
		return `## 🔍 Duplicate Detection

I can find duplicate and similar videos across your library!

**Detection methods:**
- Exact file size matches
- Similar titles
- Identical durations
- Content fingerprinting

**What you'll get:**
- Duplicate groups with similarity scores
- File comparison details
- Recommendations on which to keep
- Easy deletion options

**To run this task:**
Visit the **AI Page** and use the "Duplicate Detection" feature.

**Tip:** I compare videos by size, title, and metadata to ensure accuracy!`, nil
	}

	// AI Task Commands - Scene Detection
	if strings.Contains(messageLower, "scene") && (strings.Contains(messageLower, "detect") || strings.Contains(messageLower, "detection")) {
		return `## 🎬 Scene Detection

I can analyze videos to detect scene changes and create chapter markers!

**Capabilities:**
- Detect scene transitions
- Identify chapter boundaries
- Timestamp each scene
- Categorize scene types

**Use cases:**
- Breaking down long videos
- Creating chapter navigation
- Content analysis

**To activate:**
Go to the **AI Page** → "Scene Detection" card → Click "Start Analysis"

This is perfect for multi-scene content!`, nil
	}

	// AI Task Commands - Quality Analysis
	if (strings.Contains(messageLower, "quality") && strings.Contains(messageLower, "analyz")) || strings.Contains(messageLower, "video quality") {
		return `## 📊 Video Quality Analysis

I can analyze video technical quality and identify issues!

**What I check:**
- **Resolution** - 4K, 1080p, 720p, etc.
- **Bitrate** - Video encoding quality
- **Codec** - Video compression format
- **Frame rate** - Smoothness
- **File integrity** - Corruption detection

**Issues detected:**
- Low resolution videos
- Poor encoding quality
- Corrupted files
- Suboptimal bitrates

**To run:**
Navigate to **AI Page** → "Quality Analysis" → "Start Analysis"

**Results:** Detailed quality scores and actionable recommendations!`, nil
	}

	// AI Task Commands - Content Classification
	if strings.Contains(messageLower, "classif") || strings.Contains(messageLower, "categoriz") {
		return `## 📁 Content Classification

I can automatically categorize your videos into intelligent groups!

**Classification criteria:**
- Content type analysis
- Performer patterns
- Tag associations
- Studio organization
- Genre detection

**Benefits:**
- Better organization
- Easier searching
- Pattern recognition
- Collection insights

**To use:**
**AI Page** → "Content Classification" → "Start Analysis"

I'll analyze your entire library and suggest categories!`, nil
	}

	// AI Task Commands - Auto-Naming
	if strings.Contains(messageLower, "auto") && strings.Contains(messageLower, "nam") || strings.Contains(messageLower, "rename") {
		return `## ✏️ Auto-Naming Suggestions

I can generate better filenames based on video metadata!

**Naming scheme includes:**
- Performer names
- Studio name
- Production date
- Content tags
- Sequential numbering

**Example format:**
` + "`[Studio] Performer1_Performer2 - Title (YYYY-MM-DD)`" + `

**To generate suggestions:**
**AI Page** → "Auto-Naming" → "Start Analysis"

**Features:**
- Consistent naming across library
- Metadata-driven names
- Batch renaming support
- Preview before applying

Keep your library organized with intelligent naming!`, nil
	}

	// What can you do
	if strings.Contains(messageLower, "what can you do") || strings.Contains(messageLower, "help") && strings.Contains(messageLower, "companion") {
		var response strings.Builder
		response.WriteString("🤖 I'm your AI Companion! Here's what I can help with:\n\n")
		response.WriteString("📊 Information & Stats:\n")
		response.WriteString("- Library stats, video counts, performer counts\n")
		response.WriteString("- Thumbnail coverage and performance analysis\n")
		response.WriteString("- Top performers list\n")
		response.WriteString("- Scraper statistics and forum data\n")
		response.WriteString("- Storage usage and disk space analysis\n\n")
		response.WriteString("🔍 Smart Search:\n")
		response.WriteString("- 'Search for video [name]' - Find videos\n")
		response.WriteString("- 'Find performer [name]' - Performer details\n")
		response.WriteString("- 'Videos with tag [name]' - Tagged content\n")
		response.WriteString("- 'Recent videos' - Latest additions\n\n")
		response.WriteString("📋 Console Logs & Monitoring:\n")
		response.WriteString("- 'Console log stats' - View log statistics\n")
		response.WriteString("- 'Check for errors' - Analyze recent errors\n")
		response.WriteString("- 'Recent activity' - See what's been happening\n")
		response.WriteString("- I automatically alert on new errors!\n\n")
		response.WriteString("⚡ System Optimization:\n")
		response.WriteString("- 'Optimization suggestions' - Get recommendations\n")
		response.WriteString("- I continuously monitor library health\n")
		response.WriteString("- Alert on performance opportunities\n")
		response.WriteString("- Provide actionable improvement tips\n\n")
		response.WriteString("🧠 Intelligent Learning:\n")
		response.WriteString("- 'Usage patterns' - See detected patterns\n")
		response.WriteString("- 'What should I do next?' - Get smart suggestions\n")
		response.WriteString("- 'Maintenance schedule' - View recommended tasks\n")
		response.WriteString("- I learn from your actions over time\n\n")
		response.WriteString("🔍 Advanced Analysis:\n")
		response.WriteString("- 'Find duplicates' - Detect duplicate videos\n")
		response.WriteString("- 'Library health' - Get comprehensive health score\n")
		response.WriteString("- 'Recommendations' - Personalized content suggestions\n")
		response.WriteString("- Smart duplicate detection by title and size\n\n")
		response.WriteString("📈 Predictive Analytics:\n")
		response.WriteString("- 'Predict growth' - Library growth predictions\n")
		response.WriteString("- 'Performer quality' - Quality scoring analysis\n")
		response.WriteString("- 'Show insights' - AI-powered insights\n")
		response.WriteString("- Analyze trends and predict future states\n\n")
		response.WriteString("🤖 AI Task Capabilities:\n")
		response.WriteString("- 'Auto-link performers' - Automatic performer detection\n")
		response.WriteString("- 'Smart tagging' - Intelligent tag suggestions\n")
		response.WriteString("- 'Detect duplicates' - Find duplicate videos\n")
		response.WriteString("- 'Scene detection' - Analyze video scenes\n")
		response.WriteString("- 'Quality analysis' - Check video quality\n")
		response.WriteString("- 'Content classification' - Categorize videos\n")
		response.WriteString("- 'Auto-naming' - Generate better filenames\n")
		response.WriteString("- Ask me about any AI task for details!\n\n")
		response.WriteString("💡 Try asking:\n")
		response.WriteString("- 'What's my library health?'\n")
		response.WriteString("- 'Find duplicates'\n")
		response.WriteString("- 'Predict library growth'\n")
		response.WriteString("- 'Performer quality'\n")
		response.WriteString("- 'Show insights'\n")
		response.WriteString("- 'Find performer [name]'\n")
		response.WriteString("- 'Recent videos'\n")
		response.WriteString("- 'Storage usage'\n")
		response.WriteString("- 'Check for errors'\n")
		response.WriteString("- 'What should I do next?'\n")
		response.WriteString("- 'Give me recommendations'\n")
		return response.String(), nil
	}

	// Status check
	if strings.Contains(messageLower, "status") || strings.Contains(messageLower, "how are you") {
		status := s.GetStatus()
		uptimeSeconds := status["uptime_seconds"].(float64)
		uptime := time.Duration(uptimeSeconds * float64(time.Second))
		return fmt.Sprintf("I'm running smoothly! Uptime: %s, Watching %d libraries, Processed %d events.",
			uptime.Round(time.Second), status["active_watchers"], status["events_processed"]), nil
	}

	// Task execution suggestions
	if strings.Contains(messageLower, "scan") || strings.Contains(messageLower, "task") {
		return "I can help you run tasks! However, tasks are best executed through the Task Center page in the app. You can find quick action buttons there for:\n- Auto-Link Performers\n- Smart Tagging\n- Duplicate Detection\n- Quality Analysis\n\nOr you can ask me specific questions about your library!", nil
	}

	// Help / capabilities
	if strings.Contains(messageLower, "help") || strings.Contains(messageLower, "what can you do") {
		return "I'm your AI Companion! I can help you with:\n\n" +
			"📊 Library Information:\n" +
			"- 'How many videos/performers do I have?'\n" +
			"- 'Library statistics'\n" +
			"- 'Top performers'\n\n" +
			"🤖 System Status:\n" +
			"- 'Status' or 'How are you?'\n\n" +
			"🧠 Memory:\n" +
			"- I remember our conversations\n" +
			"- I learn your preferences over time\n\n" +
			"For complex questions or natural conversations, I can connect to LM Studio for advanced AI capabilities!", nil
	}

	// Default: try to find anything matching the message as a search
	// First, try as a performer name (direct DB lookup)
	var performerMatch string
	s.db.QueryRow("SELECT name FROM performers WHERE name LIKE ? LIMIT 1", "%"+message+"%").Scan(&performerMatch)
	if performerMatch != "" {
		result, err := s.FindPerformer(performerMatch)
		if err == nil {
			scrapedInfo, _ := s.searchPerformerAcrossAll(performerMatch)
			if scrapedInfo != "" {
				result += "\n" + scrapedInfo
			}
			return result, nil
		}
	}

	// Try as video search
	var videoMatch string
	s.db.QueryRow("SELECT title FROM videos WHERE title LIKE ? LIMIT 1", "%"+message+"%").Scan(&videoMatch)
	if videoMatch != "" {
		results, err := s.SearchVideos(message)
		if err == nil && len(results) > 0 {
			var response strings.Builder
			response.WriteString(fmt.Sprintf("🎬 Found videos matching \"%s\":\n\n", message))
			for _, r := range results {
				response.WriteString(r + "\n")
			}
			return response.String(), nil
		}
	}

	// Default fallback with quick summary
	var videoCount, performerCount int
	s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&videoCount)
	s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&performerCount)

	return fmt.Sprintf("I'm not sure how to help with that. Here's what I can do:\n\n"+
		"📊 **Your Library:** %d videos, %d performers\n\n"+
		"**Try asking:**\n"+
		"- A performer name (e.g. \"Kendra Lust\")\n"+
		"- \"Give me an update\" for app status\n"+
		"- \"How many videos?\" for stats\n"+
		"- \"Top performers\" for rankings\n"+
		"- \"Scraper summary\" for scraping info\n"+
		"- \"Check for errors\" for system health\n"+
		"- \"Help\" for full capabilities list",
		videoCount, performerCount), nil
}
