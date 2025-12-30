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
			activity.GET("/:id", getActivity)            // Get single activity
			activity.POST("", createActivity)            // Create activity log
			activity.PUT("/:id", updateActivity)         // Update activity
			activity.DELETE("/:id", deleteActivity)      // Delete activity
			activity.POST("/clean", cleanOldActivities)  // Clean old activities
			activity.POST("/clear-all", clearAllActivities) // Clear all activities
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
			ai.GET("/status", getAICompanionStatus)          // Get AI Companion status
			ai.POST("/memories", saveAIMemory)               // Save memory
			ai.GET("/memories", getAIMemories)               // Get memories
			ai.GET("/memories/search", searchAIMemories)     // Search memories
			ai.PUT("/memories/:id", updateAIMemory)          // Update memory
			ai.DELETE("/memories/:id", deleteAIMemory)       // Delete memory
		}

		// Scraper endpoints
		scraper := v1.Group("/scraper")
		{
			scraper.GET("/stats", getScraperStats)                 // Get scraper statistics
			scraper.GET("/threads", getScrapedThreads)             // List all scraped threads
			scraper.GET("/threads/search", searchScrapedThreads)   // Search threads
			scraper.GET("/threads/:id", getScrapedThread)          // Get single thread
			scraper.POST("/threads/scrape", scrapeThread)          // Scrape a thread
			scraper.POST("/threads/:id/rescrape", rescrapeThread)  // Rescrape an existing thread
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
			scraper.POST("/links/check-status", checkLinkStatuses)         // Check all download link statuses
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
