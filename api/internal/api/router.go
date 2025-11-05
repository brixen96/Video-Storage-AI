package api

import (
	"github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/middleware"
	"github.com/gin-gonic/gin"
	"net/http"
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
			videos.GET("", getVideos)           // List all videos
			videos.GET("/:id", getVideo)        // Get single video
			videos.POST("", createVideo)        // Create video entry
			videos.PUT("/:id", updateVideo)     // Update video
			videos.DELETE("/:id", deleteVideo)  // Delete video
			videos.GET("/search", searchVideos) // Search videos
		}

		// Performers endpoints
		performers := v1.Group("/performers")
		{
			performers.GET("", getPerformers)                      // List all performers
			performers.GET("/:id", getPerformer)                   // Get single performer
			performers.POST("", createPerformer)                   // Create performer
			performers.POST("/scan", scanPerformers)               // Scan performer folders
			performers.PUT("/:id", updatePerformer)                // Update performer
			performers.DELETE("/:id", deletePerformer)             // Delete performer
			performers.POST("/:id/fetch-metadata", fetchMetadata)  // Fetch from AdultDataLink
			performers.POST("/:id/reset-metadata", resetMetadata)  // Reset metadata
			performers.POST("/:id/reset-previews", resetPreviews)  // Reset previews
		}

		// Studios endpoints
		studios := v1.Group("/studios")
		{
			studios.GET("", getStudios)         // List all studios
			studios.GET("/:id", getStudio)      // Get single studio
			studios.POST("", createStudio)      // Create studio
			studios.PUT("/:id", updateStudio)   // Update studio
			studios.DELETE("/:id", deleteStudio) // Delete studio
		}

		// Groups endpoints
		groups := v1.Group("/groups")
		{
			groups.GET("", getGroups)         // List all groups
			groups.GET("/:id", getGroup)      // Get single group
			groups.POST("", createGroup)      // Create group
			groups.PUT("/:id", updateGroup)   // Update group
			groups.DELETE("/:id", deleteGroup) // Delete group
		}

		// Tags endpoints
		tags := v1.Group("/tags")
		{
			tags.GET("", getTags)         // List all tags
			tags.GET("/:id", getTag)      // Get single tag
			tags.POST("", createTag)      // Create tag
			tags.PUT("/:id", updateTag)   // Update tag
			tags.DELETE("/:id", deleteTag) // Delete tag
			tags.POST("/merge", mergeTags) // Merge multiple tags
		}

		// Activity Monitor endpoints
		activity := v1.Group("/activity")
		{
			activity.GET("", getActivities)               // List all activities with filters
			activity.GET("/recent", getRecentActivities)  // Get recent activities
			activity.GET("/status", getActivityStatus)    // Get current running tasks
			activity.GET("/stats", getActivityStats)      // Get statistics by type
			activity.GET("/:id", getActivity)             // Get single activity
			activity.POST("", createActivity)             // Create activity log
			activity.PUT("/:id", updateActivity)          // Update activity
			activity.DELETE("/:id", deleteActivity)       // Delete activity
			activity.POST("/clean", cleanOldActivities)   // Clean old activities
		}

		// File operations endpoints
		files := v1.Group("/files")
		{
			files.POST("/scan", scanDirectory)     // Scan directory for videos
			files.POST("/rename", renameFile)      // Rename file
			files.POST("/move", moveFile)          // Move file
			files.DELETE("/delete", deleteFile)    // Delete file
		}

		// AI Assistant endpoints
		ai := v1.Group("/ai")
		{
			ai.POST("/chat", aiChat)                    // Chat with AI assistant
			ai.POST("/suggest-tags", aiSuggestTags)     // AI auto-tagging
			ai.POST("/suggest-naming", aiSuggestNaming) // AI naming suggestions
			ai.POST("/analyze-library", aiAnalyzeLibrary) // AI library analysis
		}
	}

	return router
}

// Placeholder handlers - will implement these next
func getVideos(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"message": "Get videos"}) }
func getVideo(c *gin.Context)            { c.JSON(http.StatusOK, gin.H{"message": "Get video"}) }
func createVideo(c *gin.Context)         { c.JSON(http.StatusOK, gin.H{"message": "Create video"}) }
func updateVideo(c *gin.Context)         { c.JSON(http.StatusOK, gin.H{"message": "Update video"}) }
func deleteVideo(c *gin.Context)         { c.JSON(http.StatusOK, gin.H{"message": "Delete video"}) }
func searchVideos(c *gin.Context)        { c.JSON(http.StatusOK, gin.H{"message": "Search videos"}) }

// Performer handlers are implemented in performer_handlers.go

func getStudios(c *gin.Context)          { c.JSON(http.StatusOK, gin.H{"message": "Get studios"}) }
func getStudio(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"message": "Get studio"}) }
func createStudio(c *gin.Context)        { c.JSON(http.StatusOK, gin.H{"message": "Create studio"}) }
func updateStudio(c *gin.Context)        { c.JSON(http.StatusOK, gin.H{"message": "Update studio"}) }
func deleteStudio(c *gin.Context)        { c.JSON(http.StatusOK, gin.H{"message": "Delete studio"}) }

func getGroups(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"message": "Get groups"}) }
func getGroup(c *gin.Context)            { c.JSON(http.StatusOK, gin.H{"message": "Get group"}) }
func createGroup(c *gin.Context)         { c.JSON(http.StatusOK, gin.H{"message": "Create group"}) }
func updateGroup(c *gin.Context)         { c.JSON(http.StatusOK, gin.H{"message": "Update group"}) }
func deleteGroup(c *gin.Context)         { c.JSON(http.StatusOK, gin.H{"message": "Delete group"}) }

func getTags(c *gin.Context)             { c.JSON(http.StatusOK, gin.H{"message": "Get tags"}) }
func getTag(c *gin.Context)              { c.JSON(http.StatusOK, gin.H{"message": "Get tag"}) }
func createTag(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"message": "Create tag"}) }
func updateTag(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"message": "Update tag"}) }
func deleteTag(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"message": "Delete tag"}) }
func mergeTags(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"message": "Merge tags"}) }

// Activity handlers are implemented in activity_handlers.go

// File operation handlers are implemented in file_handlers.go

func aiChat(c *gin.Context)              { c.JSON(http.StatusOK, gin.H{"message": "AI chat"}) }
func aiSuggestTags(c *gin.Context)       { c.JSON(http.StatusOK, gin.H{"message": "AI suggest tags"}) }
func aiSuggestNaming(c *gin.Context)     { c.JSON(http.StatusOK, gin.H{"message": "AI suggest naming"}) }
func aiAnalyzeLibrary(c *gin.Context)    { c.JSON(http.StatusOK, gin.H{"message": "AI analyze library"}) }
