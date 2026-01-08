package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/gin-gonic/gin"
)

// getSystemHealth returns comprehensive system health metrics
func getSystemHealth(c *gin.Context) {
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Database unavailable",
			"Could not connect to database",
		))
		return
	}

	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	}

	// Database health
	dbHealth := checkDatabaseHealth(db)
	health["database"] = dbHealth

	// System resources
	health["system"] = getSystemResources()

	// Application stats
	appStats, err := getApplicationStats(db)
	if err == nil {
		health["application"] = appStats
	}

	// Service health checks
	health["services"] = checkServiceHealth()

	c.JSON(http.StatusOK, models.SuccessResponse(health, "System health retrieved"))
}

// checkDatabaseHealth checks database health and performance
func checkDatabaseHealth(db *sql.DB) gin.H {
	health := gin.H{
		"status": "unknown",
	}

	// Check connection
	start := time.Now()
	err := db.Ping()
	pingDuration := time.Since(start).Milliseconds()

	if err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
		return health
	}

	health["status"] = "healthy"
	health["ping_ms"] = pingDuration

	// Get database stats
	stats := db.Stats()
	health["open_connections"] = stats.OpenConnections
	health["in_use"] = stats.InUse
	health["idle"] = stats.Idle
	health["wait_count"] = stats.WaitCount
	health["wait_duration_ms"] = stats.WaitDuration.Milliseconds()
	health["max_idle_closed"] = stats.MaxIdleClosed
	health["max_lifetime_closed"] = stats.MaxLifetimeClosed

	// Get database size
	var dbSize int64
	err = db.QueryRow(`
		SELECT page_count * page_size as size
		FROM pragma_page_count(), pragma_page_size()
	`).Scan(&dbSize)
	if err == nil {
		health["size_bytes"] = dbSize
		health["size_mb"] = float64(dbSize) / 1024 / 1024
	}

	// Get table counts
	tableCounts := make(map[string]int64)
	tables := []string{
		"videos", "performers", "studios", "tags", "libraries",
		"scraped_threads", "scraped_posts", "scraped_download_links",
		"activities", "ai_audit_logs", "scheduled_jobs",
	}

	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		err := db.QueryRow(query).Scan(&count)
		if err == nil {
			tableCounts[table] = count
		}
	}
	health["table_counts"] = tableCounts

	return health
}

// getSystemResources returns system resource usage
func getSystemResources() gin.H {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return gin.H{
		"go_version":      runtime.Version(),
		"num_goroutines":  runtime.NumGoroutine(),
		"num_cpu":         runtime.NumCPU(),
		"memory": gin.H{
			"alloc_bytes":       m.Alloc,
			"alloc_mb":          float64(m.Alloc) / 1024 / 1024,
			"total_alloc_bytes": m.TotalAlloc,
			"total_alloc_mb":    float64(m.TotalAlloc) / 1024 / 1024,
			"sys_bytes":         m.Sys,
			"sys_mb":            float64(m.Sys) / 1024 / 1024,
			"num_gc":            m.NumGC,
			"gc_cpu_fraction":   m.GCCPUFraction,
		},
		"process_id": os.Getpid(),
	}
}

// getApplicationStats returns application-level statistics
func getApplicationStats(db *sql.DB) (gin.H, error) {
	stats := gin.H{}

	// Total videos
	var totalVideos, totalSize int64
	err := db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(size), 0)
		FROM videos
	`).Scan(&totalVideos, &totalSize)
	if err == nil {
		stats["total_videos"] = totalVideos
		stats["total_video_size_bytes"] = totalSize
		stats["total_video_size_gb"] = float64(totalSize) / 1024 / 1024 / 1024
	}

	// Total performers
	var totalPerformers int64
	db.QueryRow(`SELECT COUNT(*) FROM performers`).Scan(&totalPerformers)
	stats["total_performers"] = totalPerformers

	// Total studios
	var totalStudios int64
	db.QueryRow(`SELECT COUNT(*) FROM studios`).Scan(&totalStudios)
	stats["total_studios"] = totalStudios

	// Total tags
	var totalTags int64
	db.QueryRow(`SELECT COUNT(*) FROM tags`).Scan(&totalTags)
	stats["total_tags"] = totalTags

	// Scraper stats
	var totalThreads, activeThreads int64
	db.QueryRow(`SELECT COUNT(*) FROM scraped_threads`).Scan(&totalThreads)
	db.QueryRow(`SELECT COUNT(*) FROM scraped_threads WHERE is_active = 1`).Scan(&activeThreads)
	stats["scraper"] = gin.H{
		"total_threads":  totalThreads,
		"active_threads": activeThreads,
	}

	// Download link stats
	var totalLinks, activeLinks, deadLinks int64
	db.QueryRow(`SELECT COUNT(*) FROM scraped_download_links`).Scan(&totalLinks)
	db.QueryRow(`SELECT COUNT(*) FROM scraped_download_links WHERE status = 'active'`).Scan(&activeLinks)
	db.QueryRow(`SELECT COUNT(*) FROM scraped_download_links WHERE status = 'dead'`).Scan(&deadLinks)

	var downloadedLinks int64
	db.QueryRow(`SELECT COUNT(*) FROM scraped_download_links WHERE download_status = 'downloaded'`).Scan(&downloadedLinks)

	stats["download_links"] = gin.H{
		"total":      totalLinks,
		"active":     activeLinks,
		"dead":       deadLinks,
		"downloaded": downloadedLinks,
	}

	// Activity stats
	var runningActivities, pausedActivities int64
	db.QueryRow(`SELECT COUNT(*) FROM activities WHERE status = 'running'`).Scan(&runningActivities)
	db.QueryRow(`SELECT COUNT(*) FROM activities WHERE is_paused = 1`).Scan(&pausedActivities)
	stats["activities"] = gin.H{
		"running": runningActivities,
		"paused":  pausedActivities,
	}

	// AI usage stats (last 24 hours)
	var aiInteractions24h int64
	var aiCost24h float64
	db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(cost_usd), 0)
		FROM ai_audit_logs
		WHERE created_at >= datetime('now', '-1 day')
	`).Scan(&aiInteractions24h, &aiCost24h)
	stats["ai_usage_24h"] = gin.H{
		"interactions": aiInteractions24h,
		"cost_usd":     aiCost24h,
	}

	// Scheduled jobs
	var totalJobs, enabledJobs int64
	db.QueryRow(`SELECT COUNT(*) FROM scheduled_jobs`).Scan(&totalJobs)
	db.QueryRow(`SELECT COUNT(*) FROM scheduled_jobs WHERE enabled = 1`).Scan(&enabledJobs)
	stats["scheduled_jobs"] = gin.H{
		"total":   totalJobs,
		"enabled": enabledJobs,
	}

	// Recent activity timeline
	type RecentActivity struct {
		Hour  string
		Count int64
	}
	var recentActivities []RecentActivity
	rows, err := db.Query(`
		SELECT
			strftime('%Y-%m-%d %H:00', created_at) as hour,
			COUNT(*) as count
		FROM activities
		WHERE created_at >= datetime('now', '-24 hours')
		GROUP BY hour
		ORDER BY hour DESC
		LIMIT 24
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ra RecentActivity
			if err := rows.Scan(&ra.Hour, &ra.Count); err == nil {
				recentActivities = append(recentActivities, ra)
			}
		}
		stats["recent_activity_timeline"] = recentActivities
	}

	return stats, nil
}

// checkServiceHealth checks health of various services
func checkServiceHealth() gin.H {
	services := gin.H{}

	// Check scheduler
	schedulerSvc := ensureSchedulerService()
	services["scheduler"] = gin.H{
		"status":    getServiceStatus(schedulerSvc != nil),
		"available": schedulerSvc != nil,
	}

	// Check scraper
	scraperSvc := ensureScraperService()
	services["scraper"] = gin.H{
		"status":    getServiceStatus(scraperSvc != nil),
		"available": scraperSvc != nil,
	}

	// Check activity service
	activitySvc := ensureActivityService()
	services["activity"] = gin.H{
		"status":    getServiceStatus(activitySvc != nil),
		"available": activitySvc != nil,
	}

	// Check AI audit service
	aiAuditSvc := ensureAIAuditService()
	services["ai_audit"] = gin.H{
		"status":    getServiceStatus(aiAuditSvc != nil),
		"available": aiAuditSvc != nil,
	}

	// Check JDownloader availability
	jdSvc := ensureJDownloaderService()
	if jdSvc != nil {
		available := jdSvc.IsAvailable()
		var version string
		if available {
			v, _ := jdSvc.GetVersion()
			version = v
		}
		services["jdownloader"] = gin.H{
			"status":    getServiceStatus(available),
			"available": available,
			"version":   version,
		}
	} else {
		services["jdownloader"] = gin.H{
			"status":    "unavailable",
			"available": false,
		}
	}

	return services
}

// getServiceStatus returns a status string based on availability
func getServiceStatus(available bool) string {
	if available {
		return "healthy"
	}
	return "unavailable"
}

// getSystemMetrics returns time-series metrics for monitoring
func getSystemMetrics(c *gin.Context) {
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponseMsg(
			"Database unavailable",
			"Could not connect to database",
		))
		return
	}

	timeRange := c.DefaultQuery("range", "24h")
	var hoursBack int

	switch timeRange {
	case "1h":
		hoursBack = 1
	case "6h":
		hoursBack = 6
	case "24h":
		hoursBack = 24
	case "7d":
		hoursBack = 168
	default:
		hoursBack = 24
	}

	metrics := gin.H{}

	// Activity metrics over time
	type TimeSeriesPoint struct {
		Timestamp string  `json:"timestamp"`
		Count     int64   `json:"count"`
		Value     float64 `json:"value,omitempty"`
	}

	// Activities created over time
	var activityTimeline []TimeSeriesPoint
	rows, err := db.Query(`
		SELECT
			strftime('%Y-%m-%d %H:%M', created_at) as timestamp,
			COUNT(*) as count
		FROM activities
		WHERE created_at >= datetime('now', ?)
		GROUP BY timestamp
		ORDER BY timestamp ASC
	`, fmt.Sprintf("-%d hours", hoursBack))

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var point TimeSeriesPoint
			if err := rows.Scan(&point.Timestamp, &point.Count); err == nil {
				activityTimeline = append(activityTimeline, point)
			}
		}
	}
	metrics["activity_timeline"] = activityTimeline

	// AI cost over time
	var aiCostTimeline []TimeSeriesPoint
	rows2, err := db.Query(`
		SELECT
			strftime('%Y-%m-%d %H:00', created_at) as timestamp,
			COUNT(*) as count,
			COALESCE(SUM(cost_usd), 0) as value
		FROM ai_audit_logs
		WHERE created_at >= datetime('now', ?)
		GROUP BY timestamp
		ORDER BY timestamp ASC
	`, fmt.Sprintf("-%d hours", hoursBack))

	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var point TimeSeriesPoint
			if err := rows2.Scan(&point.Timestamp, &point.Count, &point.Value); err == nil {
				aiCostTimeline = append(aiCostTimeline, point)
			}
		}
	}
	metrics["ai_cost_timeline"] = aiCostTimeline

	// Download link verification over time
	var linkVerificationTimeline []TimeSeriesPoint
	rows3, err := db.Query(`
		SELECT
			strftime('%Y-%m-%d %H:00', last_checked_at) as timestamp,
			COUNT(*) as count
		FROM scraped_download_links
		WHERE last_checked_at >= datetime('now', ?)
		GROUP BY timestamp
		ORDER BY timestamp ASC
	`, fmt.Sprintf("-%d hours", hoursBack))

	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var point TimeSeriesPoint
			if err := rows3.Scan(&point.Timestamp, &point.Count); err == nil {
				linkVerificationTimeline = append(linkVerificationTimeline, point)
			}
		}
	}
	metrics["link_verification_timeline"] = linkVerificationTimeline

	c.JSON(http.StatusOK, models.SuccessResponse(metrics, "System metrics retrieved"))
}
