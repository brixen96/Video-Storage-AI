package api

import (
	"net/http"
	"path/filepath"

	"github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/middleware"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// SetupRouter configures and returns the Gin router
func SetupRouter(cfg *config.Config) *gin.Engine {
	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Create router
	router := gin.New()

	// Apply middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())

	// Add gzip compression for API responses
	router.Use(gzip.Gzip(gzip.DefaultCompression))

	// Add config to context for handlers
	router.Use(func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	})

	// Serve static files (performer previews, thumbnails) with caching
	// Create assets group with cache middleware
	assets := router.Group("/assets")
	assets.Use(middleware.StaticFileCache())
	assets.StaticFS("", gin.Dir(cfg.Paths.AssetsBaseDir, false))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		if err := database.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unhealthy",
				"database": "disconnected",
				"error":    err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "connected",
			"version":  "0.1.0",
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Libraries endpoints
		libraries := v1.Group("/libraries")
		{
			libraries.GET("", getLibraries)                            // List all libraries
			libraries.GET("/primary", getPrimaryLibrary)               // Get primary library
			libraries.GET("/:id", getLibrary)                          // Get single library
			libraries.GET("/:id/browse", browseLibrary)                // Browse library filesystem
			libraries.GET("/:id/stream", streamVideo)                  // Stream video file
			libraries.POST("", createLibrary)                          // Create library
			libraries.PUT("/:id", updateLibrary)                       // Update library
			libraries.DELETE("/:id", deleteLibrary)                    // Delete library
			libraries.POST("/:id/generate-thumbnails", generateThumbnailsForFolder) // Generate thumbnails for folder
		}

		// Videos endpoints
		videos := v1.Group("/videos")
		{
			videos.GET("", getVideos)                              // List all videos
			videos.GET("/:id", getVideo)                           // Get single video
			videos.POST("", createVideo)                           // Create video entry
			videos.PUT("/:id", updateVideo)                        // Update video
			videos.DELETE("/:id", deleteVideo)                     // Delete video
			videos.GET("/search", searchVideos)                    // Search videos
			videos.POST("/scan", scanVideos)                       // Scan library for videos
			videos.POST("/scan-all-parallel", scanAllVideosParallel) // Scan all libraries in parallel
			videos.POST("/generate-previews", generateAllPreviews) // Generate preview storyboards for all videos
			videos.POST("/generate-thumbnails", generateVideoThumbnails) // Generate thumbnails for all videos
			videos.POST("/:id/open-in-explorer", openInExplorer)   // Open video location in file explorer
			videos.GET("/:id/stream", streamVideoByID)             // Stream video by ID
			videos.PATCH("/marks-by-path", updateVideoMarksByPath) // Update marks by file path
			videos.PATCH("/content-type/folder", updateFolderContentType) // Update content type for folder recursively
			videos.PATCH("/performer/folder", linkPerformerToFolder)       // Link performer to all videos in a folder
			videos.POST("/:id/convert", convertVideoToMP4)         // Convert video to MP4
		}

		// Conversion endpoints
		conversion := v1.Group("/conversion")
		{
			conversion.GET("/status", checkFFmpegStatus) // Check FFmpeg installation status
		}

		// Performers endpoints
		performers := v1.Group("/performers")
		{
			performers.GET("", getPerformers)                       // List all performers
			performers.GET("/:id", getPerformer)                    // Get single performer
			performers.GET("/:id/previews", getPerformerPreviews)   // Get all preview videos
			performers.GET("/:id/tags", getPerformerTags)           // Get performer master tags
			performers.GET("/:id/videos", getPerformerVideos)       // Get videos featuring performer
			performers.GET("/:id/scraped-threads", getThreadsByPerformer) // Get scraped threads for performer
			performers.POST("", createPerformer)                    // Create performer
			performers.POST("/scan", scanPerformers)                // Scan performer folders
			performers.POST("/cleanup", cleanupEmptyPerformers)     // Cleanup empty performers
			performers.POST("/generate-thumbnails", generatePerformerThumbnails) // Generate thumbnails for all performers
			performers.PUT("/:id", updatePerformer)                 // Update performer
			performers.DELETE("/:id", deletePerformer)              // Delete performer
			performers.POST("/:id/fetch-metadata", fetchMetadata)   // Fetch from AdultDataLink
			performers.POST("/:id/reset-metadata", resetMetadata)   // Reset metadata
			performers.POST("/:id/reset-previews", resetPreviews)   // Reset previews
			performers.POST("/:id/generate-thumbnail", generatePerformerThumbnail) // Generate thumbnail for single performer
			performers.POST("/:id/tags", addPerformerTag)           // Add master tag to performer
			performers.DELETE("/:id/tags/:tagId", removePerformerTag) // Remove master tag from performer
			performers.POST("/:id/sync-tags", syncPerformerTags)    // Sync master tags to all videos
		}

		// Studios endpoints
		studios := v1.Group("/studios")
		{
			studios.GET("", getStudios)                             // List all studios
			studios.GET("/:id", getStudio)                          // Get single studio
			studios.POST("", createStudio)                          // Create studio
			studios.PUT("/:id", updateStudio)                       // Update studio
			studios.DELETE("/:id", deleteStudio)                    // Delete studio
			studios.POST("/:id/reset-metadata", resetStudioMetadata) // Reset metadata
		}

		// Groups endpoints
		groups := v1.Group("/groups")
		{
			groups.GET("", getGroups)                              // List all groups
			groups.GET("/:id", getGroup)                           // Get single group
			groups.POST("", createGroup)                           // Create group
			groups.PUT("/:id", updateGroup)                        // Update group
			groups.DELETE("/:id", deleteGroup)                     // Delete group
			groups.POST("/:id/reset-metadata", resetGroupMetadata) // Reset metadata
		}

		// Tags endpoints
		tags := v1.Group("/tags")
		{
			tags.GET("", getTags)          // List all tags
			tags.GET("/:id", getTag)       // Get single tag
			tags.POST("", createTag)       // Create tag
			tags.PUT("/:id", updateTag)    // Update tag
			tags.DELETE("/:id", deleteTag) // Delete tag
			tags.POST("/merge", mergeTags) // Merge multiple tags
		}

		// Activity Monitor endpoints
		activity := v1.Group("/activity")
		{
			activity.GET("", getActivities)              // List all activities with filters
			activity.GET("/recent", getRecentActivities) // Get recent activities
			activity.GET("/status", getActivityStatus)   // Get current running tasks
			activity.GET("/stats", getActivityStats)     // Get statistics by type
			activity.GET("/paused", getPausedActivities) // Get all paused tasks
			activity.GET("/:id", getActivity)            // Get single activity
			activity.POST("", createActivity)            // Create activity log
			activity.PUT("/:id", updateActivity)         // Update activity
			activity.DELETE("/:id", deleteActivity)      // Delete activity
			activity.POST("/clean", cleanOldActivities)  // Clean old activities
			activity.POST("/clear-all", clearAllActivities) // Clear all activities
			activity.POST("/:id/pause", pauseActivity)   // Pause a running task
			activity.POST("/:id/resume", resumeActivity) // Resume a paused task
			activity.POST("/:id/cancel", cancelActivity) // Cancel a running or paused task
			activity.POST("/:id/retry", retryActivity)   // Retry a failed or cancelled task
		}

		// File operations endpoints
		files := v1.Group("/files")
		{
			files.POST("/scan", scanDirectory)                    // Scan directory for videos
			files.POST("/rename", renameFile)                     // Rename file
			files.POST("/move", moveFile)                         // Move file within library
			files.POST("/move-across-libraries", moveFileAcrossLibraries) // Move file between libraries
			files.DELETE("/delete", deleteFile)                   // Delete file
		}

		// Database management endpoints
		database := v1.Group("/database")
		{
			database.GET("/stats", getDatabaseStats)       // Get database statistics
			database.POST("/optimize", optimizeDatabase)   // Optimize database (VACUUM)
			database.POST("/backup", backupDatabase)       // Create database backup
			database.GET("/backups", listBackups)          // List available backups
			database.POST("/restore", restoreDatabase)     // Restore from backup
		}

		// AI Assistant endpoints
		ai := v1.Group("/ai")
		{
			ai.POST("/link-performers", autoLinkPerformers)  // Auto-link performers to videos
			ai.POST("/apply-links", applyPerformerLinks)     // Apply selected performer links
			ai.POST("/suggest-tags", suggestTags)            // AI smart tagging
			ai.POST("/apply-tag-suggestions", applyTagSuggestions) // Apply tag suggestions
			ai.POST("/detect-scenes", detectScenes)          // Detect scene boundaries in videos
			ai.POST("/classify-content", classifyContent)    // Classify video content types
			ai.POST("/analyze-quality", analyzeQuality)      // Analyze video quality
			ai.POST("/detect-missing-metadata", detectMissingMetadata) // Find videos with missing metadata
			ai.POST("/detect-duplicates", detectDuplicates)  // Find duplicate/similar videos
			ai.POST("/suggest-naming", suggestNaming)        // Generate better filename suggestions
			ai.GET("/library-analytics", getLibraryAnalytics) // Get comprehensive library statistics
			ai.POST("/analyze-thumbnail-quality", analyzeThumbnailQuality) // Analyze thumbnail quality

			// AI Companion endpoints
			ai.POST("/chat", aiCompanionChat)                // Chat with AI Companion
			ai.POST("/chat/stream", aiCompanionChatStream)   // Chat with streaming SSE response
			ai.GET("/chat/history", getChatHistory)          // Get persistent chat history
			ai.DELETE("/chat/history", clearChatHistory)     // Clear chat history
			ai.GET("/status", getAICompanionStatus)          // Get AI Companion status
			ai.POST("/memories", saveAIMemory)               // Save memory
			ai.GET("/memories", getAIMemories)               // Get memories
			ai.GET("/memories/search", searchAIMemories)     // Search memories
			ai.PUT("/memories/:id", updateAIMemory)          // Update memory
			ai.DELETE("/memories/:id", deleteAIMemory)       // Delete memory
			ai.GET("/config", getAIConfig)                   // Get AI/LM Studio config
			ai.PUT("/config", updateAIConfig)                // Update AI/LM Studio config
			ai.POST("/test-connection", testAIConnection)    // Test LM Studio connection
		}

		// Scraper endpoints
		scraper := v1.Group("/scraper")
		{
			scraper.GET("/stats", getScraperStats)                 // Get scraper statistics
			scraper.GET("/threads", getScrapedThreads)             // List all scraped threads
			scraper.GET("/threads/watched", getWatchedThreads)     // List watched threads only
			scraper.GET("/threads/search", searchScrapedThreads)   // Search threads
			scraper.GET("/threads/:id", getScrapedThread)          // Get single thread
			scraper.POST("/threads/scrape", scrapeThread)          // Scrape a thread
			scraper.POST("/threads/:id/rescrape", rescrapeThread)  // Rescrape an existing thread
			scraper.POST("/threads/:id/watch", toggleThreadWatch)  // Toggle watch status for thread
			scraper.GET("/threads/:id/recent-posts", getThreadRecentPosts) // Get recent posts for thread hover
			scraper.POST("/posts/:id/mark-read", markPostAsRead)          // Mark a post as read
			scraper.DELETE("/threads/:id", deleteThread)           // Delete a thread
			scraper.DELETE("/threads", deleteMultipleThreads)      // Delete multiple threads
			scraper.DELETE("/threads/all", deleteAllThreads)       // Delete all threads
			scraper.POST("/session", setSessionCookie)             // Set session cookie for authentication
			scraper.GET("/session", getSessionCookie)              // Get session cookie status

			// Forum scraping endpoints
			scraper.POST("/forum/scrape-listing", scrapeForumCategory)     // Scrape forum category listing (get all thread URLs)
			scraper.POST("/forum/scrape-all", scrapeForumAndSaveAll)       // Scrape entire forum with full content
			scraper.POST("/performers/auto-link", autoLinkThreadsToPerformers) // Auto-link threads to performers
			scraper.POST("/performers/link", linkThreadToPerformer)        // Manually link thread to performer
			scraper.POST("/performers/create-from-thread", createPerformerFromThread) // Create performer from thread
			scraper.POST("/links/check-status", checkLinkStatuses)         // Check all download link statuses

			// Link verification endpoints
			scraper.POST("/threads/:id/verify-links", verifyThreadLinks)   // Verify all links for a thread
			scraper.GET("/threads/:id/link-stats", getThreadLinkStats)     // Get link statistics for a thread
			scraper.POST("/links/verify-all", verifyAllLinks)              // Verify old/unchecked links
			scraper.GET("/links/verification-stats", getVerificationStats) // Get overall verification stats

			// Provider health analytics endpoints
			scraper.GET("/providers/health", getProviderHealthStats)       // Get health stats for all providers
			scraper.GET("/providers/:provider/health", getProviderHealth)  // Get health stats for specific provider

			// Link export endpoints
			scraper.GET("/threads/:id/export-links", exportThreadLinks)    // Export links from a thread (formats: txt, json, csv)
			scraper.GET("/links/export-all", exportAllThreadsLinks)        // Export links from all threads

			// Video scraper endpoints
			scraper.GET("/extract-video-url", extractVideoURLHandler)      // Extract direct video URL from a page
			scraper.POST("/video", scrapeVideoHandler)                     // Scrape a video from URL
			scraper.GET("/video/test", testScraperHandler)                 // Test scraper without saving (GET /video/test?url=...)
			scraper.GET("/videos", getScrapedVideosHandler)                // Get all scraped videos
			scraper.DELETE("/videos", deleteAllScrapedVideosHandler)       // Delete all scraped videos
			scraper.DELETE("/videos/site/:siteType", deleteScrapedVideosBySiteTypeHandler)    // Delete videos by site type
			scraper.DELETE("/videos/source/:source", deleteScrapedVideosBySourceHandler)      // Delete videos by source hostname
			scraper.POST("/videos/bulk-delete", bulkDeleteScrapedVideosHandler) // Bulk delete scraped videos by IDs
			scraper.GET("/videos/:id", getScrapedVideoHandler)             // Get single scraped video
			scraper.POST("/videos/:id/mark-downloaded", markScrapedVideoDownloadedHandler)     // Mark video as downloaded
			scraper.DELETE("/videos/:id/mark-downloaded", unmarkScrapedVideoDownloadedHandler) // Remove downloaded mark
			scraper.GET("/proxy-image", proxyScraperImageHandler)             // Proxy CDN image with correct Referer
			scraper.GET("/videos/duplicates", findDuplicatesHandler)       // Find duplicate scraped videos
			scraper.DELETE("/videos/duplicates", deleteDuplicatesHandler)  // Delete duplicate scraped videos

			// Scraper tabs endpoints
			scraper.GET("/tabs", getScraperTabsHandler)                    // Get all scraper tabs
			scraper.GET("/tabs/:id", getScraperTabHandler)                 // Get single scraper tab
			scraper.POST("/tabs", createScraperTabHandler)                 // Create scraper tab
			scraper.PUT("/tabs/:id", updateScraperTabHandler)              // Update scraper tab
			scraper.DELETE("/tabs/:id", deleteScraperTabHandler)           // Delete scraper tab
			scraper.POST("/detect-pagination", detectPaginationConfig)     // Detect pagination pattern from two page URLs

			// Scraper jobs endpoints
			scraper.GET("/jobs", getScraperJobsHandler)                    // Get scraper jobs (all or by tabId query param)
			scraper.GET("/jobs/:id", getScraperJobHandler)                 // Get single scraper job
			scraper.POST("/jobs/start", startScraperJobHandler)            // Start a new scraper job
			scraper.POST("/jobs/:id/cancel", cancelScraperJobHandler)      // Cancel a scraper job

			// PornHub scraper endpoints
			scraper.POST("/pornhub/scrape", startPornHubScrapeHandler)     // Start PornHub scrape by keyword
			scraper.GET("/pornhub/keywords", getPornHubKeywordsHandler)    // Get suggested keywords
			scraper.GET("/pornhub/videos", getPornHubVideosHandler)        // Get scraped PornHub videos
			scraper.GET("/pornhub/stats", getPornHubStatsHandler)          // Get PornHub scraping stats

			// LuxureTV scraper endpoints
			scraper.POST("/luxuretv/scrape", startLuxureTVScrapeHandler)       // Start LuxureTV scrape by category
			scraper.GET("/luxuretv/categories", getLuxureTVCategoriesHandler)  // Get available categories
			scraper.GET("/luxuretv/videos", getLuxureTVVideosHandler)          // Get scraped LuxureTV videos
			scraper.GET("/luxuretv/performers", getLuxureTVPerformersHandler)  // Get matched performers
			scraper.GET("/luxuretv/tags", getLuxureTVTagsHandler)              // Get matched tags
			scraper.GET("/luxuretv/stats", getLuxureTVStatsHandler)            // Get LuxureTV scraping stats
			scraper.POST("/luxuretv/fetch-embeds", fetchLuxureTVEmbedIDsHandler) // Fetch embed IDs for videos

			// Sexuria scraper endpoints
			scraper.POST("/sexuria/scrape", startSexuriaScrapeHandler)                   // Start Sexuria scrape by category
			scraper.GET("/sexuria/categories", getSexuriaCategoriesHandler)              // Get available categories
			scraper.GET("/sexuria/items", getSexuriaItemsHandler)                        // Get scraped Sexuria items
			scraper.GET("/sexuria/stats", getSexuriaStatsHandler)                        // Get Sexuria scraping stats
			scraper.GET("/sexuria/item-categories", getSexuriaItemCategoriesHandler)      // Get distinct item categories
			scraper.POST("/sexuria/item-details", getSexuriaItemDetailsHandler)          // Get detailed info for an item
			scraper.GET("/sexuria/search-options", getSexuriaSearchOptionsHandler)       // Get advanced search options
			scraper.POST("/sexuria/advanced-search", startSexuriaAdvancedSearchHandler)  // Start advanced search scrape

			// Sexuria link verification endpoints
			scraper.GET("/sexuria/links/stats", getSexuriaLinkStats)                     // Get Sexuria link statistics
			scraper.POST("/sexuria/items/:id/extract-links", extractSexuriaItemLinks)    // Extract links from item page
			scraper.GET("/sexuria/items/:id/links", getSexuriaItemLinks)                 // Get links for item
			scraper.POST("/sexuria/items/:id/verify-links", verifySexuriaItemLinks)      // Verify links for item
			scraper.POST("/sexuria/links/verify-all", verifyAllSexuriaLinks)             // Batch verify all links

			// Rule34Video scraper endpoints
			scraper.POST("/rule34video/scrape", startRule34VideoScrapeHandler)             // Start Rule34Video scrape
			scraper.GET("/rule34video/sort-options", getRule34VideoSortOptionsHandler)     // Get available sort options
			scraper.GET("/rule34video/filter-options", getRule34VideoFilterOptionsHandler) // Get all filter/sort options
			scraper.GET("/rule34video/videos", getRule34VideoVideosHandler)                // Get scraped Rule34Video videos
			scraper.GET("/rule34video/stats", getRule34VideoStatsHandler)                  // Get Rule34Video scraping stats
			scraper.GET("/rule34video/download-url", getRule34VideoDownloadURLHandler)     // Get download URL for a video
			scraper.GET("/rule34video/video-details", getRule34VideoDetailsHandler)        // Get full video details (tags, download URLs)
			scraper.GET("/rule34video/categories", getRule34VideoCategoriesHandler)             // Get top categories from site
			scraper.GET("/rule34video/local-categories", getRule34VideoLocalCategoriesHandler) // Distinct categories from local DB
			scraper.GET("/rule34video/models", getRule34VideoModelsHandler)                    // Get top models/artists

			// Motherless scraper endpoints
			scraper.POST("/motherless/scrape", startMotherlessScrapeHandler)               // Start Motherless scrape by search URL
			scraper.GET("/motherless/videos", getMotherlessVideosHandler)                  // Get scraped Motherless videos (paginated)
			scraper.GET("/motherless/stats", getMotherlessStatsHandler)                    // Get Motherless scraping stats
			scraper.GET("/motherless/search-terms", getMotherlessSearchTermsHandler)       // Get distinct search terms

			// LeakVids scraper endpoints
			scraper.POST("/leakvids/scrape", startLeakVidsScrapeHandler)                   // Start LeakVids scrape
			scraper.GET("/leakvids/videos", getLeakVidsVideosHandler)                      // Get scraped LeakVids videos (paginated)
			scraper.GET("/leakvids/stats", getLeakVidsStatsHandler)                        // Get LeakVids scraping stats

			// HQPorner scraper endpoints
			scraper.POST("/hqporner/scrape", startHQPornerScrapeHandler)                   // Start HQPorner scrape
			scraper.GET("/hqporner/videos", getHQPornerVideosHandler)                      // Get scraped HQPorner videos (paginated)
			scraper.GET("/hqporner/stats", getHQPornerStatsHandler)                        // Get HQPorner scraping stats
			scraper.GET("/hqporner/performers", getHQPornerPerformersHandler)              // Get matched performers from HQPorner videos
			scraper.GET("/hqporner/tags", getHQPornerTagsHandler)                          // Get matched tags from HQPorner videos
		}

		// Download Tracking endpoints
		downloads := v1.Group("/downloads")
		{
			downloads.POST("/links/:id/mark-downloaded", markLinkAsDownloaded)    // Mark link as downloaded
			downloads.POST("/links/:id/mark-failed", markLinkAsFailed)            // Mark link as failed
			downloads.POST("/links/:id/reset", resetLinkDownloadStatus)           // Reset download status
			downloads.POST("/links/bulk-mark-downloaded", bulkMarkAsDownloaded)   // Bulk mark as downloaded
			downloads.GET("/threads/:id/stats", getThreadDownloadStats)           // Get thread download stats
			downloads.GET("/performers/:id/stats", getPerformerDownloadStats)     // Get performer download stats
			downloads.GET("/stats", getGlobalDownloadStats)                       // Get global download stats
			downloads.GET("/threads", getThreadsByDownloadStatus)                 // Filter threads by download status
			downloads.GET("/recent", getRecentDownloads)                          // Get recent downloads
		}

		// JDownloader Integration endpoints
		jdownloader := v1.Group("/jdownloader")
		{
			// Configuration
			jdownloader.GET("/config", getJDownloaderConfig)                // Get JDownloader configuration
			jdownloader.PUT("/config", updateJDownloaderConfig)             // Update JDownloader configuration
			jdownloader.POST("/test", testJDownloaderConnection)            // Test connection to JDownloader
			jdownloader.GET("/available", checkJDownloaderAvailability)     // Quick availability check

			// System info
			jdownloader.GET("/system", getJDSystemInfo)                     // Get system info (version, uptime, state)
			jdownloader.GET("/speed", getJDSpeed)                           // Get current download speed
			jdownloader.POST("/speed-limit", setJDSpeedLimit)               // Set speed limit

			// Send links
			jdownloader.POST("/send", sendToJDownloader)                    // Send links to JDownloader (new format)
			jdownloader.POST("/send-links", sendLinksToJDownloader)         // Send custom links (legacy)
			jdownloader.POST("/threads/:id/send", sendThreadToJDownloader)  // Send entire thread to JDownloader

			// History
			jdownloader.GET("/history", getJDownloaderHistory)              // Get send history
			jdownloader.DELETE("/history", clearJDownloaderHistory)         // Clear history
			jdownloader.GET("/package-timestamps", getJDPackageAddedTimes)  // Get locally-tracked package add times

			// Download control
			jdownloader.GET("/status", getJDownloaderConnectionStatus)      // Get connection status
			jdownloader.POST("/start", startJDownloaderDownloads)           // Start all downloads
			jdownloader.POST("/stop", stopJDownloaderDownloads)             // Stop all downloads
			jdownloader.POST("/pause", pauseJDDownloads)                    // Pause/resume downloads

			// LinkGrabber
			jdownloader.GET("/linkgrabber/links", getLinkGrabberLinks)      // Get LinkGrabber links
			jdownloader.GET("/linkgrabber/packages", getLinkGrabberPackages) // Get LinkGrabber packages
			jdownloader.POST("/linkgrabber/move", moveToDownloads)          // Move to download list
			jdownloader.POST("/linkgrabber/remove", removeLinkGrabberLinks) // Remove from LinkGrabber
			jdownloader.POST("/linkgrabber/clear", clearLinkGrabber)        // Clear LinkGrabber

			// Downloads
			jdownloader.GET("/downloads/links", getDownloadLinks)           // Get download links
			jdownloader.GET("/downloads/packages", getDownloadPackages)     // Get download packages
			jdownloader.POST("/downloads/remove", removeDownloadLinks)      // Remove from downloads
			jdownloader.POST("/downloads/force", forceDownload)             // Force download
			jdownloader.POST("/downloads/cleanup", cleanupFinished)         // Cleanup finished
			jdownloader.POST("/downloads/reset", resetLinks)                // Reset failed links

			// Common operations
			jdownloader.POST("/enable", setLinkEnabled)                     // Enable/disable links
			jdownloader.POST("/priority", setLinkPriority)                  // Set priority
			jdownloader.POST("/rename", renamePackage)                      // Rename package
		}

		// AI Audit endpoints
		aiAudit := v1.Group("/ai-audit")
		{
			aiAudit.GET("", getAIAuditLogs)                              // Get all audit logs with filters
			aiAudit.GET("/stats", getAIAuditStats)                       // Get AI usage statistics
			aiAudit.GET("/search", searchAIAuditLogs)                    // Search audit logs
			aiAudit.GET("/:id", getAIAuditLog)                           // Get single audit log
			aiAudit.GET("/performer/:id", getAIAuditLogsByPerformer)     // Get logs for performer
			aiAudit.GET("/thread/:id", getAIAuditLogsByThread)           // Get logs for thread
			aiAudit.POST("/clean", deleteOldAIAuditLogs)                 // Delete old audit logs
		}

		// Scheduler endpoints
		scheduler := v1.Group("/scheduler")
		{
			scheduler.GET("/status", getSchedulerStatus)                  // Get scheduler status
			scheduler.GET("/jobs", getAllJobs)                            // Get all jobs
			scheduler.GET("/jobs/:id", getJob)                            // Get single job
			scheduler.POST("/jobs", createJob)                            // Create new job
			scheduler.PUT("/jobs/:id", updateJob)                         // Update job
			scheduler.DELETE("/jobs/:id", deleteJob)                      // Delete job
			scheduler.POST("/jobs/:id/toggle", toggleJob)                 // Toggle job enabled/disabled
			scheduler.GET("/jobs/:id/history", getJobExecutionHistory)    // Get job execution history
		}

		// Auto-Update endpoints
		autoUpdate := v1.Group("/auto-update")
		{
			autoUpdate.GET("/configs", getAutoUpdateConfigs)              // Get all scraper auto-update configs
			autoUpdate.GET("/configs/:type", getAutoUpdateConfig)         // Get config for specific scraper
			autoUpdate.PUT("/configs/:type", updateAutoUpdateConfig)      // Update scraper config
			autoUpdate.POST("/configs/:type/toggle", toggleAutoUpdate)    // Enable/disable auto-update
			autoUpdate.POST("/configs/:type/check-now", checkNowAutoUpdate) // Trigger immediate check
			autoUpdate.GET("/status", getAutoUpdateStatus)                // Get overall auto-update status
		}

		// Dashboard endpoints
		dashboard := v1.Group("/dashboard")
		{
			dashboard.GET("/stats", getDashboardStats)                    // Get aggregated dashboard stats
			dashboard.GET("/feed", getDashboardFeed)                      // Get activity feed items
			dashboard.GET("/scrapers", getDashboardScrapers)              // Get scraper status cards
			dashboard.GET("/running-tasks", getDashboardRunningTasks)     // Get currently running tasks
			dashboard.POST("/quick-action/:action", executeDashboardQuickAction) // Execute quick action
			dashboard.DELETE("/feed", clearDashboardFeed)                // Clear all feed items
			dashboard.POST("/feed", addDashboardFeedItem)                 // Add feed item (internal)
		}

		// System Health endpoints
		system := v1.Group("/system")
		{
			system.GET("/health", getSystemHealth)             // Get comprehensive system health
			system.GET("/metrics", getSystemMetrics)           // Get time-series metrics
			system.POST("/open-incognito", openInIncognito)    // Open URL in incognito browser window
			system.POST("/open-login-helper", openLoginHelper) // Open login page for cookie capture
			system.POST("/open-url", openUrl)                  // Open any URL in Firefox
		}

		// Notification endpoints
		notifications := v1.Group("/notifications")
		{
			notifications.GET("", getNotifications)                         // Get all notifications with filters
			notifications.GET("/stats", getNotificationStats)               // Get notification statistics
			notifications.GET("/:id", getNotification)                      // Get single notification
			notifications.POST("", createNotification)                      // Create new notification
			notifications.POST("/:id/read", markNotificationAsRead)         // Mark notification as read
			notifications.POST("/read-all", markAllNotificationsAsRead)     // Mark all as read
			notifications.POST("/:id/archive", archiveNotification)         // Archive notification
			notifications.DELETE("", deleteAllNotifications)                 // Delete all notifications
			notifications.DELETE("/:id", deleteNotification)                // Delete notification
			notifications.POST("/clean", deleteOldNotifications)            // Delete old notifications
		}

		// Backup endpoints
		backups := v1.Group("/backups")
		{
			backups.POST("", createDatabaseBackup)                   // Create new backup
			backups.GET("", listDatabaseBackups)                     // List all backups
			backups.GET("/stats", getBackupStats)                    // Get backup statistics
			backups.GET("/:filename", getBackup)                     // Get backup info
			backups.GET("/:filename/download", downloadBackup)       // Download backup file
			backups.POST("/:filename/restore", restoreBackup)        // Restore from backup
			backups.DELETE("/:filename", deleteBackup)               // Delete backup
			backups.POST("/cleanup", cleanupOldBackups)              // Cleanup old backups
		}

		// Console Logs endpoints
		consoleLogs := v1.Group("/console-logs")
		{
			consoleLogs.GET("", getConsoleLogs)              // Get all console logs with filters
			consoleLogs.GET("/stats", getConsoleLogStats)    // Get console log statistics
			consoleLogs.GET("/:id", getConsoleLog)           // Get single console log
			consoleLogs.POST("", createConsoleLog)           // Create a new console log
			consoleLogs.DELETE("/:id", deleteConsoleLog)     // Delete a console log
			consoleLogs.POST("/clear", clearConsoleLogs)     // Clear all console logs
			consoleLogs.POST("/clean", cleanOldConsoleLogs)  // Clean old console logs
		}

		// Video Lists endpoints
		lists := v1.Group("/lists")
		{
			lists.GET("", getAllListsHandler)                    // Get all video lists
			lists.GET("/for-video", getListsForVideoHandler)     // Get lists containing a video
			lists.GET("/:id", getListByIDHandler)                // Get single list
			lists.GET("/:id/items", getListItemsHandler)         // Get list items with video details
			lists.GET("/:id/stats", getListStatsHandler)         // Get list statistics
			lists.POST("", createListHandler)                    // Create new list
			lists.PUT("/:id", updateListHandler)                 // Update list
			lists.DELETE("/:id", deleteListHandler)              // Delete list
			lists.POST("/:id/videos", addToListHandler)          // Add videos to list
			lists.DELETE("/:id/videos", removeFromListHandler)   // Remove videos from list
			lists.POST("/:id/reorder", reorderListItemsHandler)  // Reorder items in list
			lists.PUT("/:id/items/:itemId/notes", updateItemNotesHandler) // Update item notes
		}

		// Site health check
		v1.GET("/sites/status", checkSiteHealth) // Check if an external site is reachable

		// WebSocket endpoint
		v1.GET("/ws", handleWebSocket)
	}

	// Serve frontend static files from dist folder (production build)
	// This allows the API to serve the entire application from a single server
	distPath := filepath.Join(".", "dist")

	// Apply cache middleware globally for frontend files
	router.Use(middleware.FrontendCacheMiddleware())

	// Handle all unmatched routes (serve frontend SPA)
	router.NoRoute(middleware.SPAMiddleware(distPath))

	return router
}

// Video handlers are implemented in video_handlers.go
// Performer handlers are implemented in performer_handlers.go
// Studio handlers are implemented in studio_handlers.go
// Group handlers are implemented in group_handlers.go
// Tag handlers are implemented in tag_handlers.go

// Activity handlers are implemented in activity_handlers.go

// File operation handlers are implemented in file_handlers.go

// AI handlers are implemented in ai_handlers.go
