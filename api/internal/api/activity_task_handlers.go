package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

// pauseActivity pauses a running task
func pauseActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid activity ID", err.Error()))
		return
	}

	var req struct {
		Checkpoint map[string]interface{} `json:"checkpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Checkpoint = make(map[string]interface{})
	}

	err = svc.PauseTask(id, req.Checkpoint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to pause activity", err.Error()))
		return
	}

	activity, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Activity paused successfully"))
		return
	}

	if wsHub != nil {
		wsHub.BroadcastActivityUpdate(activity)
	}

	c.JSON(http.StatusOK, models.SuccessResponse(activity, "Activity paused successfully"))
}

// cancelActivity cancels a running or paused task permanently
func cancelActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid activity ID", err.Error()))
		return
	}

	if err = svc.CancelTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to cancel activity", err.Error()))
		return
	}

	activity, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Activity cancelled successfully"))
		return
	}

	if wsHub != nil {
		wsHub.BroadcastActivityUpdate(activity)
	}

	c.JSON(http.StatusOK, models.SuccessResponse(activity, "Activity cancelled successfully"))
}

// retryActivity retries a failed or cancelled task from the beginning
func retryActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid activity ID", err.Error()))
		return
	}

	activity, err := svc.RetryTask(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to retry activity", err.Error()))
		return
	}

	if err := activity.UnmarshalDetails(); err != nil {
		log.Printf("Warning: failed to parse activity details: %v", err)
	}

	switch activity.TaskType {
	case "pornhub_scraping":
		keyword, ok := activity.DetailsObj["keyword"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to retry activity", "Keyword not found in activity details"))
			return
		}

		pornHubSvc := ensurePornHubScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.PornHubSearchRequest{
				Keyword:    keyword,
				StartPage:  1,
				ActivityID: id,
			}
			result, err := pornHubSvc.ScrapeByKeywordWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s' (page %d)", progress.PagesCompleted, progress.VideosFound, keyword, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page": progress.CurrentPage,
					"videos_found": progress.VideosFound,
					"videos_saved": progress.VideosSaved,
					"keyword":      keyword,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s'", result.PagesCrawled, result.TotalScraped, keyword)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(activity, "PornHub scraping restarted"))
		return

	case "luxuretv_scraping":
		categoryURL, ok := activity.DetailsObj["category_url"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to retry activity", "Category URL not found in activity details"))
			return
		}

		luxureTVSvc := ensureLuxureTVScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.LuxureTVScrapeRequest{
				CategoryURL: categoryURL,
				StartPage:   1,
				ActivityID:  id,
			}
			result, err := luxureTVSvc.ScrapeCategoryWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page": progress.CurrentPage,
					"videos_found": progress.VideosFound,
					"videos_saved": progress.VideosSaved,
					"category_url": categoryURL,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(activity, "LuxureTV scraping restarted"))
		return

	case "sexuria_scraping", "sexuria_advanced_search":
		categoryURL, _ := activity.DetailsObj["category_url"].(string)
		sexuriaSvc := ensureSexuriaScraperService()
		activitySvc := ensureActivityService()

		go func() {
			req := services.SexuriaScrapeRequest{
				CategoryURL: categoryURL,
				StartPage:   1,
				ActivityID:  id,
			}
			result, err := sexuriaSvc.ScrapeCategoryWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d items from Sexuria (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page": progress.CurrentPage,
					"videos_found": progress.VideosFound,
					"videos_saved": progress.VideosSaved,
					"category_url": categoryURL,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d items from Sexuria", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(activity, "Sexuria scraping restarted"))
		return

	default:
		c.JSON(http.StatusOK, models.SuccessResponse(activity, "Activity reset for retry - manual restart may be required"))
	}
}

// resumeActivity resumes a paused task
func resumeActivity(c *gin.Context) {
	svc := ensureActivityService()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg("Invalid activity ID", err.Error()))
		return
	}

	activity, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to get activity", err.Error()))
		return
	}

	if err := activity.UnmarshalDetails(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to parse activity details", err.Error()))
		return
	}

	checkpoint, err := svc.ResumeTask(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to resume activity", err.Error()))
		return
	}

	log.Printf("Resume task %d - checkpoint data: %+v", id, checkpoint)

	updatedActivity, _ := svc.GetByID(id)
	if updatedActivity != nil && wsHub != nil {
		wsHub.BroadcastActivityUpdate(updatedActivity)
	}

	switch activity.TaskType {
	case "forum_scrape":
		forumURL, ok := activity.DetailsObj["forum_url"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to resume activity", "Forum URL not found in activity details"))
			return
		}

		scraperSvc := ensureScraperService()
		go func() {
			if err := scraperSvc.ResumeForumScrape(forumURL, int(id)); err != nil {
				log.Printf("Error resuming forum scrape: %v", err)
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Forum scraping resumed in background"))
		return

	case "sexuria_scraping":
		categoryURL, ok := activity.DetailsObj["category_url"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to resume activity", "Category URL not found in activity details"))
			return
		}

		startPage, initialVideoCount, initialSavedCount := extractCheckpointCounts(checkpoint)
		log.Printf("sexuria_scraping: Resuming from page %d with %d videos found (checkpoint: %+v)", startPage, initialVideoCount, checkpoint)

		sexuriaSvc := ensureSexuriaScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.SexuriaScrapeRequest{
				CategoryURL:       categoryURL,
				StartPage:         startPage,
				ActivityID:        id,
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
			}
			result, err := sexuriaSvc.ScrapeCategoryWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d items from Sexuria (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page": progress.CurrentPage,
					"videos_found": progress.VideosFound,
					"videos_saved": progress.VideosSaved,
					"category_url": categoryURL,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d items", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Sexuria scraping resumed in background"))
		return

	case "sexuria_advanced_search":
		searchURL, ok := activity.DetailsObj["search_url"].(string)
		if !ok {
			if checkpoint != nil {
				searchURL, ok = checkpoint["search_url"].(string)
			}
			if !ok {
				c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to resume activity", "Search URL not found in activity details"))
				return
			}
		}

		startPage, initialVideoCount, initialSavedCount := extractCheckpointCounts(checkpoint)
		log.Printf("sexuria_advanced_search: Resuming from page %d with %d videos found (checkpoint: %+v)", startPage, initialVideoCount, checkpoint)

		sexuriaSvc := ensureSexuriaScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.SexuriaAdvancedSearchRequest{
				StartPage:         startPage,
				ActivityID:        id,
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
			}

			if cats, ok := activity.DetailsObj["categories"].([]interface{}); ok {
				for _, cat := range cats {
					if s, ok := cat.(string); ok {
						req.Categories = append(req.Categories, s)
					}
				}
			}
			if ps, ok := activity.DetailsObj["pornstars"].(string); ok {
				req.Pornstars = ps
			}
			if tags, ok := activity.DetailsObj["tags"].(string); ok {
				req.Tags = tags
			}
			if prod, ok := activity.DetailsObj["production"].(string); ok {
				req.Production = prod
			}
			if res, ok := activity.DetailsObj["resolutions"].([]interface{}); ok {
				for _, r := range res {
					if s, ok := r.(string); ok {
						req.Resolutions = append(req.Resolutions, s)
					}
				}
			}
			if maxPages, ok := activity.DetailsObj["max_pages"].(float64); ok {
				req.MaxPages = int(maxPages)
			}

			result, err := sexuriaSvc.ScrapeAdvancedSearchWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d items (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page": progress.CurrentPage,
					"videos_found": progress.VideosFound,
					"videos_saved": progress.VideosSaved,
					"search_url":   searchURL,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d items", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Sexuria advanced search resumed in background"))
		return

	case "pornhub_scraping":
		keyword, ok := activity.DetailsObj["keyword"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to resume activity", "Keyword not found in activity details"))
			return
		}

		startPage, initialVideoCount, initialSavedCount := extractCheckpointCounts(checkpoint)

		pornhubSvc := ensurePornHubScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.PornHubSearchRequest{
				Keyword:           keyword,
				StartPage:         startPage,
				ActivityID:        id,
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
			}
			result, err := pornhubSvc.ScrapeByKeywordWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s' (page %d)", progress.PagesCompleted, progress.VideosFound, keyword, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page": progress.CurrentPage,
					"videos_found": progress.VideosFound,
					"videos_saved": progress.VideosSaved,
					"keyword":      keyword,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos for '%s'", result.PagesCrawled, result.TotalScraped, keyword)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "PornHub scraping resumed in background"))
		return

	case "luxuretv_scraping":
		categoryURL, ok := activity.DetailsObj["category_url"].(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to resume activity", "Category URL not found in activity details"))
			return
		}

		startPage, initialVideoCount, initialSavedCount := extractCheckpointCounts(checkpoint)

		luxureTVSvc := ensureLuxureTVScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.LuxureTVScrapeRequest{
				CategoryURL:       categoryURL,
				StartPage:         startPage,
				ActivityID:        id,
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
			}
			result, err := luxureTVSvc.ScrapeCategoryWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page": progress.CurrentPage,
					"videos_found": progress.VideosFound,
					"videos_saved": progress.VideosSaved,
					"category_url": categoryURL,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos from LuxureTV", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "LuxureTV scraping resumed in background"))
		return

	case "scraper_thread":
		threadURL, ok := activity.DetailsObj["url"].(string)
		if !ok {
			if checkpoint != nil {
				threadURL, ok = checkpoint["url"].(string)
			}
			if !ok {
				c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to resume activity", "Thread URL not found in activity details"))
				return
			}
		}

		scraperSvc := ensureScraperService()
		go func() {
			if err := scraperSvc.ScrapeThreadComplete(threadURL, int(id)); err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					log.Printf("Error resuming scraper thread: %v", err)
					failedStatus := "failed"
					activitySvc := ensureActivityService()
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Scraper thread resumed in background"))
		return

	case "scraping":
		url, ok := activity.DetailsObj["url"].(string)
		if !ok {
			if checkpoint != nil {
				url, ok = checkpoint["url"].(string)
			}
			if !ok {
				c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to resume activity", "URL not found in activity details"))
				return
			}
		}

		var tabID int64
		if tid, ok := activity.DetailsObj["tab_id"].(float64); ok {
			tabID = int64(tid)
		}

		startPage, initialVideoCount, initialSavedCount := extractCheckpointCounts(checkpoint)

		videoScraperSvc := ensureVideoScraperService()
		activitySvc := ensureActivityService()
		go func() {
			req := services.ScrapeVideoRequest{
				URL:               url,
				StartPage:         startPage,
				ActivityID:        id,
				InitialVideoCount: initialVideoCount,
				InitialSavedCount: initialSavedCount,
				Options: services.ScrapeOptions{
					DownloadVideo:    false,
					ExtractMetadata:  true,
					ScrapeThumbnails: true,
					MaxPages:         0,
				},
			}
			result, err := videoScraperSvc.ScrapePageWithProgress(req, func(progress services.ScrapeProgress) {
				progressPercent := 0
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Message:  stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos (page %d)", progress.PagesCompleted, progress.VideosFound, progress.CurrentPage)),
					Progress: &progressPercent,
				})
				activitySvc.SaveCheckpoint(id, map[string]interface{}{
					"current_page": progress.CurrentPage,
					"videos_found": progress.VideosFound,
					"videos_saved": progress.VideosSaved,
					"url":          url,
					"tab_id":       tabID,
				})
			})

			if err != nil {
				if !strings.Contains(err.Error(), "paused by user") {
					failedStatus := "failed"
					activitySvc.Update(int(id), &models.ActivityLogUpdate{
						Status:    &failedStatus,
						Message:   stringPtr(fmt.Sprintf("Failed: %s", err.Error())),
						Completed: true,
					})
				}
			} else {
				completedStatus := "completed"
				progress100 := 100
				activitySvc.Update(int(id), &models.ActivityLogUpdate{
					Status:    &completedStatus,
					Message:   stringPtr(fmt.Sprintf("Scraped %d pages, found %d videos", result.PagesCrawled, result.TotalScraped)),
					Progress:  &progress100,
					Completed: true,
				})
			}
		}()

		c.JSON(http.StatusOK, models.SuccessResponse(nil, "Scraping resumed in background"))
		return

	default:
		result := map[string]interface{}{
			"checkpoint": checkpoint,
			"message":    "Activity resumed - manual restart may be required",
		}
		c.JSON(http.StatusOK, models.SuccessResponse(result, "Activity resumed successfully"))
	}
}

// getPausedActivities retrieves all currently paused tasks
func getPausedActivities(c *gin.Context) {
	svc := ensureActivityService()

	activities, err := svc.GetAll("running", "", 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg("Failed to retrieve paused activities", err.Error()))
		return
	}

	pausedActivities := make([]*models.Activity, 0)
	for i := range activities {
		if activities[i].IsPaused {
			pausedActivities = append(pausedActivities, &activities[i])
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(pausedActivities, "Paused activities retrieved successfully"))
}

// extractCheckpointCounts pulls page/video counts from a checkpoint map.
// Returns startPage (next page after last completed), initialVideoCount, initialSavedCount.
func extractCheckpointCounts(checkpoint map[string]interface{}) (startPage, initialVideoCount, initialSavedCount int) {
	startPage = 1
	if checkpoint == nil {
		return
	}
	if page, ok := checkpoint["current_page"].(float64); ok {
		startPage = int(page) + 1
	}
	if found, ok := checkpoint["videos_found"].(float64); ok {
		initialVideoCount = int(found)
	}
	if saved, ok := checkpoint["videos_saved"].(float64); ok {
		initialSavedCount = int(saved)
	}
	return
}
