package api

import (
	"net/http"
	"strconv"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var aiMonitorService *services.AIMonitorService

func getAIMonitorService() *services.AIMonitorService {
	if aiMonitorService == nil {
		aiMonitorService = services.NewAIMonitorService()
	}
	return aiMonitorService
}

// getAIDashboard returns aggregated metrics for the monitoring dashboard
// GET /api/v1/ai-monitor/dashboard
func getAIDashboard(c *gin.Context) {
	svc := getAIMonitorService()
	metrics, err := svc.GetDashboardMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// getAICallHistory returns paginated call history with filters
// GET /api/v1/ai-monitor/calls?limit=50&offset=0&model=deepseek&success=true
func getAICallHistory(c *gin.Context) {
	svc := getAIMonitorService()

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	model := c.Query("model")
	successOnly := c.Query("success") == "true"

	calls, total, err := svc.GetCallHistory(limit, offset, model, successOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    calls,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// getAIToolStats returns tool usage statistics
// GET /api/v1/ai-monitor/tools
func getAIToolStats(c *gin.Context) {
	svc := getAIMonitorService()
	stats, err := svc.GetToolStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// getAICostSummary returns cost breakdown by day and model
// GET /api/v1/ai-monitor/costs?days=30
func getAICostSummary(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Database not available",
		})
		return
	}

	// Cost by day
	rows, err := db.Query(`
		SELECT DATE(created_at) as date, model, SUM(cost_usd) as cost, COUNT(*) as count
		FROM ai_call_logs
		WHERE created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY DATE(created_at), model
		ORDER BY date DESC
	`, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	defer rows.Close()

	type dailyModelCost struct {
		Date      string  `json:"date"`
		Model     string  `json:"model"`
		CostUSD   float64 `json:"cost_usd"`
		CallCount int     `json:"call_count"`
	}
	var costs []dailyModelCost
	for rows.Next() {
		var dc dailyModelCost
		rows.Scan(&dc.Date, &dc.Model, &dc.CostUSD, &dc.CallCount)
		costs = append(costs, dc)
	}

	// Total cost summary
	var totalCost float64
	var totalCalls int
	db.QueryRow(`SELECT COALESCE(SUM(cost_usd), 0), COUNT(*) FROM ai_call_logs WHERE created_at >= datetime('now', '-'||?||' days')`, days).Scan(&totalCost, &totalCalls)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"costs":      costs,
			"total_cost": totalCost,
			"total_calls": totalCalls,
			"days":       days,
		},
	})
}
