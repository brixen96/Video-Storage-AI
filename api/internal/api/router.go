package api

import (
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/middleware"
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

	// Add config to context for handlers
	router.Use(func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	})

	// Serve static files (performer previews, thumbnails)
	router.Static("/assets", cfg.Paths.AssetsBaseDir)

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
			libraries.GET("", getLibraries)              // List all libraries
			libraries.GET("/primary", getPrimaryLibrary) // Get primary library
			libraries.GET("/:id", getLibrary)            // Get single library
			libraries.GET("/:id/browse", browseLibrary)  // Browse library filesystem
			libraries.GET("/:id/stream", streamVideo)    // Stream video file
			libraries.POST("", createLibrary)            // Create library
			libraries.PUT("/:id", updateLibrary)         // Update library
			libraries.DELETE("/:id", deleteLibrary)      // Delete library
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
			videos.POST("/:id/open-in-explorer", openInExplorer)   // Open video location in file explorer
			videos.GET("/:id/stream", streamVideoByID)             // Stream video by ID
			videos.PATCH("/marks-by-path", updateVideoMarksByPath) // Update marks by file path
		}

		// Performers endpoints
		performers := v1.Group("/performers")
		{
			performers.GET("", getPerformers)                     // List all performers
			performers.GET("/:id", getPerformer)                  // Get single performer
			performers.GET("/:id/previews", getPerformerPreviews) // Get all preview videos
			performers.POST("", createPerformer)                  // Create performer
			performers.POST("/scan", scanPerformers)              // Scan performer folders
			performers.PUT("/:id", updatePerformer)               // Update performer
			performers.DELETE("/:id", deletePerformer)            // Delete performer
			performers.POST("/:id/fetch-metadata", fetchMetadata) // Fetch from AdultDataLink
			performers.POST("/:id/reset-metadata", resetMetadata) // Reset metadata
			performers.POST("/:id/reset-previews", resetPreviews) // Reset previews
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
		}

		// File operations endpoints
		files := v1.Group("/files")
		{
			files.POST("/scan", scanDirectory)  // Scan directory for videos
			files.POST("/rename", renameFile)   // Rename file
			files.POST("/move", moveFile)       // Move file
			files.DELETE("/delete", deleteFile) // Delete file
		}

		// AI Assistant endpoints
		ai := v1.Group("/ai")
		{
			ai.POST("/chat", aiChat)                      // Chat with AI assistant
			ai.POST("/suggest-tags", aiSuggestTags)       // AI auto-tagging
			ai.POST("/suggest-naming", aiSuggestNaming)   // AI naming suggestions
			ai.POST("/analyze-library", aiAnalyzeLibrary) // AI library analysis
		}

		// WebSocket endpoint
		v1.GET("/ws", handleWebSocket)
	}

	return router
}

// Video handlers are implemented in video_handlers.go
// Performer handlers are implemented in performer_handlers.go
// Studio handlers are implemented in studio_handlers.go
// Group handlers are implemented in group_handlers.go
// Tag handlers are implemented in tag_handlers.go

// Activity handlers are implemented in activity_handlers.go

// File operation handlers are implemented in file_handlers.go

func aiChat(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"message": "AI chat"}) }
func aiSuggestTags(c *gin.Context)    { c.JSON(http.StatusOK, gin.H{"message": "AI suggest tags"}) }
func aiSuggestNaming(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"message": "AI suggest naming"}) }
func aiAnalyzeLibrary(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "AI analyze library"}) }
