package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var autoUpdateService *services.AutoUpdateService

// ensureAutoUpdateService initializes the service if needed
func ensureAutoUpdateService() *services.AutoUpdateService {
	if autoUpdateService == nil {
		activitySvc := ensureActivityService()
		notificationSvc := ensureNotificationService()
		autoUpdateService = services.NewAutoUpdateService(activitySvc, notificationSvc)
	}
	return autoUpdateService
}

// getAutoUpdateConfigs returns all scraper auto-update configurations
func getAutoUpdateConfigs(c *gin.Context) {
	svc := ensureAutoUpdateService()

	configs, err := svc.GetAllConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get auto-update configs",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(configs, "Auto-update configs retrieved"))
}

// getAutoUpdateConfig returns config for a specific scraper
func getAutoUpdateConfig(c *gin.Context) {
	svc := ensureAutoUpdateService()
	scraperType := c.Param("type")

	// Use GetOrCreateConfig to auto-create configs for custom tabs
	config, err := svc.GetOrCreateConfig(scraperType)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Config not found",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(config, "Config retrieved"))
}

// updateAutoUpdateConfig updates a scraper's auto-update configuration
func updateAutoUpdateConfig(c *gin.Context) {
	svc := ensureAutoUpdateService()
	scraperType := c.Param("type")

	var update models.AutoUpdateConfigUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Ensure config exists first (for custom tabs)
	_, err := svc.GetOrCreateConfig(scraperType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get/create config",
			err.Error(),
		))
		return
	}

	if err := svc.UpdateConfig(scraperType, &update); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to update config",
			err.Error(),
		))
		return
	}

	// Return updated config
	config, _ := svc.GetOrCreateConfig(scraperType)
	c.JSON(http.StatusOK, models.SuccessResponse(config, "Config updated"))
}

// toggleAutoUpdate enables or disables auto-update for a scraper
func toggleAutoUpdate(c *gin.Context) {
	svc := ensureAutoUpdateService()
	scraperType := c.Param("type")

	var request struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	// Ensure config exists first (for custom tabs)
	_, err := svc.GetOrCreateConfig(scraperType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get/create config",
			err.Error(),
		))
		return
	}

	if err := svc.ToggleEnabled(scraperType, request.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to toggle auto-update",
			err.Error(),
		))
		return
	}

	// Return updated config
	config, _ := svc.GetOrCreateConfig(scraperType)
	c.JSON(http.StatusOK, models.SuccessResponse(config, "Auto-update toggled"))
}

// checkNowAutoUpdate triggers an immediate check for new content
func checkNowAutoUpdate(c *gin.Context) {
	scraperType := c.Param("type")

	// Get or create config to ensure it exists
	svc := ensureAutoUpdateService()
	config, err := svc.GetOrCreateConfig(scraperType)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Config not found",
			err.Error(),
		))
		return
	}

	// Trigger check in background
	go func() {
		triggerAutoUpdateCheck(scraperType, config)
	}()

	c.JSON(http.StatusAccepted, models.SuccessResponse(nil, "Check started. Monitor activity for progress."))
}

// triggerAutoUpdateCheck runs an auto-update check for a scraper
func triggerAutoUpdateCheck(scraperType string, config *models.ScraperAutoUpdateConfig) {
	// This will be implemented by each scraper's CheckForNew method
	// For now, just log that it was triggered
	activitySvc := ensureActivityService()

	activity, err := activitySvc.Create(&models.ActivityLogCreate{
		TaskType: "auto_update_check",
		Status:   "running",
		Message:  "Checking for new content from " + scraperType,
		Details: map[string]interface{}{
			"scraper_type":       scraperType,
			"max_pages_to_check": config.MaxPagesToCheck,
		},
	})

	if err != nil {
		return
	}

	// Call the appropriate scraper's check method based on type
	var result *models.NewContentResult
	var checkErr error

	// Check if this is a custom tab scraper type
	if strings.HasPrefix(scraperType, "custom_") {
		tabIDStr := strings.TrimPrefix(scraperType, "custom_")
		tabID, parseErr := strconv.ParseInt(tabIDStr, 10, 64)
		if parseErr != nil {
			checkErr = fmt.Errorf("invalid custom tab ID: %s", tabIDStr)
		} else {
			scraperTabsSvc := ensureScraperTabsService()
			tab, tabErr := scraperTabsSvc.GetScraperTabByID(tabID)
			if tabErr != nil {
				checkErr = fmt.Errorf("custom tab not found: %v", tabErr)
			} else {
				// Route to appropriate scraper based on tab's site type
				switch tab.SiteType {
				case "pornhub":
					pornhubSvc := ensurePornHubScraperService()
					keywords := []string{}
					if config.Config != nil {
						keywords = config.Config.Keywords
					}
					result, checkErr = pornhubSvc.CheckForNewVideos(keywords, config.MaxPagesToCheck, activity.ID)
				case "luxuretv":
					luxureSvc := ensureLuxureTVScraperService()
					// For custom tabs, use the tab's baseUrl as the category
					categories := []string{}
					if tab.BaseURL != "" {
						categories = []string{tab.BaseURL}
					}
					result, checkErr = luxureSvc.CheckForNewVideos(categories, config.MaxPagesToCheck, activity.ID)
				case "sexuria":
					sexuriaSvc := ensureSexuriaScraperService()
					// For custom tabs, use the tab's baseUrl (can be category or advanced search URL)
					categories := []string{}
					if tab.BaseURL != "" {
						categories = []string{tab.BaseURL}
					}
					result, checkErr = sexuriaSvc.CheckForNewItems(categories, config.MaxPagesToCheck, activity.ID)
				case "rule34video":
					rule34Svc := ensureRule34VideoScraperService()
					result, checkErr = rule34Svc.CheckForNewVideos(config.MaxPagesToCheck, activity.ID)
				case "generic", "":
					// Generic tabs use the tab's baseUrl for auto-update
					if tab.BaseURL == "" {
						checkErr = fmt.Errorf("no base URL configured for generic tab")
					} else {
						videoScraperSvc := ensureVideoScraperService()
						result, checkErr = videoScraperSvc.CheckForNewVideos(scraperType, tab.BaseURL, config.MaxPagesToCheck, activity.ID)
					}
				default:
					checkErr = fmt.Errorf("unsupported site type '%s'. Supported types: pornhub, luxuretv, sexuria, rule34video", tab.SiteType)
				}
			}
		}
	} else {
		// Handle built-in scraper types
		switch scraperType {
		case "simpcity":
			scraperSvc := ensureScraperService()
			urls := []string{}
			if config.Config != nil && len(config.Config.URLs) > 0 {
				urls = config.Config.URLs
			}
			result, checkErr = scraperSvc.CheckForNewContent(urls, config.MaxPagesToCheck, activity.ID)
		case "pornhub":
			pornhubSvc := ensurePornHubScraperService()
			keywords := []string{}
			if config.Config != nil {
				keywords = config.Config.Keywords
			}
			result, checkErr = pornhubSvc.CheckForNewVideos(keywords, config.MaxPagesToCheck, activity.ID)
		case "rule34video":
			rule34Svc := ensureRule34VideoScraperService()
			result, checkErr = rule34Svc.CheckForNewVideos(config.MaxPagesToCheck, activity.ID)
		case "luxuretv":
			luxureSvc := ensureLuxureTVScraperService()
			categories := []string{}
			if config.Config != nil {
				categories = config.Config.Categories
			}
			result, checkErr = luxureSvc.CheckForNewVideos(categories, config.MaxPagesToCheck, activity.ID)
		case "sexuria":
			sexuriaSvc := ensureSexuriaScraperService()
			categories := []string{}
			if config.Config != nil {
				categories = config.Config.Categories
			}
			result, checkErr = sexuriaSvc.CheckForNewItems(categories, config.MaxPagesToCheck, activity.ID)
		case "motherless":
			motherlessSvc := ensureMotherlessScraperService()
			result, checkErr = motherlessSvc.CheckForNewVideos(config.MaxPagesToCheck, activity.ID)
		case "leakvids":
			leakVidsSvc := ensureLeakVidsScraperService()
			result, checkErr = leakVidsSvc.CheckForNewVideos(config.MaxPagesToCheck, activity.ID)
		case "hqporner":
			hqpornerSvc := ensureHQPornerScraperService()
			result, checkErr = hqpornerSvc.CheckForNewVideos("", config.MaxPagesToCheck, activity.ID)
		default:
			checkErr = fmt.Errorf("unknown scraper type: %s", scraperType)
		}
	}

	autoUpdateSvc := ensureAutoUpdateService()
	dashboardSvc := ensureDashboardService()

	if checkErr != nil {
		activitySvc.FailTask(activity.ID, checkErr.Error())
		// Disable so it doesn't keep retrying a failing site
		autoUpdateSvc.ToggleEnabled(scraperType, false)
		// Add a visible error entry to the dashboard feed
		displayName, icon, _ := models.GetScraperDisplayInfo(scraperType)
		if displayName == scraperType {
			displayName = scraperType
		}
		dashboardSvc.AddFeedItem(&models.DashboardFeedItem{
			ItemType:    "error",
			Title:       displayName + ": Auto-Scraper Disabled",
			Message:     "Scraper failed and was automatically disabled — " + checkErr.Error(),
			Icon:        icon,
			Color:       "danger",
			ScraperType: scraperType,
		})
		if wsHub != nil {
			wsHub.BroadcastFeedUpdate(scraperType, 0)
		}
		return
	}

	// Record the check result
	foundCount := 0
	if result != nil {
		foundCount = result.NewItemsFound
	}
	autoUpdateSvc.RecordCheckResult(scraperType, foundCount)

	// Add to dashboard feed if new content found
	if foundCount > 0 {
		// Include video data in feed item
		details := result.Details
		if details == nil {
			details = map[string]interface{}{}
		}

		// Convert NewItems to videos array for the feed
		if len(result.NewItems) > 0 {
			videos := make([]map[string]interface{}, 0, len(result.NewItems))
			for _, item := range result.NewItems {
				v := map[string]interface{}{
					"title":         item.Title,
					"url":           item.URL,
					"thumbnail_url": item.Thumbnail,
					"duration":      item.Duration,
					"views":         item.Views,
					"type":          item.Type,
				}
				if item.ID > 0 {
					v["id"] = item.ID
				}
				videos = append(videos, v)
			}
			details["videos"] = videos
		}

		dashboardSvc.AddNewContentFeedItem(scraperType, foundCount, details)
		autoUpdateSvc.CreateNotification(scraperType, foundCount, result.Details)

		// Broadcast feed update via WebSocket for live UI updates
		if wsHub != nil {
			wsHub.BroadcastFeedUpdate(scraperType, foundCount)
		}
	}

	activitySvc.CompleteTask(int64(activity.ID), "Check completed. Found "+itoa(foundCount)+" new items.")
}

// getAutoUpdateStatus returns overall auto-update status
func getAutoUpdateStatus(c *gin.Context) {
	svc := ensureAutoUpdateService()

	configs, err := svc.GetAllConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to get status",
			err.Error(),
		))
		return
	}

	enabledCount := 0
	for _, config := range configs {
		if config.Enabled {
			enabledCount++
		}
	}

	status := map[string]interface{}{
		"enabled_count": enabledCount,
		"total_count":   len(configs),
		"configs":       configs,
	}

	c.JSON(http.StatusOK, models.SuccessResponse(status, "Status retrieved"))
}

// Helper function
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if negative {
		s = "-" + s
	}
	return s
}
