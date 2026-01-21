package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/gin-gonic/gin"
)

// exportThreadLinks exports download links in various formats
func exportThreadLinks(c *gin.Context) {
	threadIDStr := c.Param("id")
	threadID, err := strconv.ParseInt(threadIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid thread ID",
			err.Error(),
		))
		return
	}

	// Get query parameters
	format := c.DefaultQuery("format", "txt")       // txt, json, csv
	status := c.DefaultQuery("status", "active")    // active, all
	provider := c.Query("provider")                  // optional filter

	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Database unavailable",
			"Could not connect to database",
		))
		return
	}

	// Build query
	query := `
		SELECT
			sdl.id, sdl.url, sdl.provider, sdl.filename,
			sdl.file_size, sdl.file_type, sdl.status,
			st.title as thread_title
		FROM scraped_download_links sdl
		INNER JOIN scraped_threads st ON sdl.thread_id = st.id
		WHERE sdl.thread_id = ?
	`
	args := []interface{}{threadID}

	// Add status filter
	if status == "active" {
		query += " AND sdl.status = 'active'"
	}

	// Add provider filter
	if provider != "" {
		query += " AND LOWER(sdl.provider) = LOWER(?)"
		args = append(args, provider)
	}

	query += " ORDER BY sdl.id ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to query links",
			err.Error(),
		))
		return
	}
	defer rows.Close()

	// Collect links
	type ExportLink struct {
		ID          int64  `json:"id"`
		URL         string `json:"url"`
		Provider    string `json:"provider"`
		Filename    string `json:"filename"`
		FileSize    int64  `json:"file_size"`
		FileType    string `json:"file_type"`
		Status      string `json:"status"`
		ThreadTitle string `json:"thread_title"`
	}

	var links []ExportLink
	for rows.Next() {
		var link ExportLink
		if err := rows.Scan(
			&link.ID, &link.URL, &link.Provider, &link.Filename,
			&link.FileSize, &link.FileType, &link.Status, &link.ThreadTitle,
		); err != nil {
			continue
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"count":   0,
			"message": "No links found matching criteria",
		}, "No links to export"))
		return
	}

	// Export based on format
	switch format {
	case "json":
		exportJSON(c, links)
	case "csv":
		exportCSV(c, links)
	case "txt":
		exportTXT(c, links)
	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid format",
			"Supported formats: txt, json, csv",
		))
	}
}

// exportJSON exports links as JSON
func exportJSON(c *gin.Context, links interface{}) {
	c.Header("Content-Disposition", "attachment; filename=links.json")
	c.Header("Content-Type", "application/json")

	encoder := json.NewEncoder(c.Writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(links); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to encode JSON",
			err.Error(),
		))
	}
}

// exportCSV exports links as CSV
func exportCSV(c *gin.Context, linksInterface interface{}) {
	links, ok := linksInterface.([]ExportLink)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Invalid data type",
			"Expected ExportLink slice",
		))
		return
	}

	c.Header("Content-Disposition", "attachment; filename=links.csv")
	c.Header("Content-Type", "text/csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"URL", "Provider", "Filename", "File Size", "File Type", "Status", "Thread"})

	// Write rows
	for _, link := range links {
		writer.Write([]string{
			link.URL,
			link.Provider,
			link.Filename,
			formatFileSize(link.FileSize),
			link.FileType,
			link.Status,
			link.ThreadTitle,
		})
	}
}

// exportTXT exports links as plain text (one URL per line)
func exportTXT(c *gin.Context, linksInterface interface{}) {
	links, ok := linksInterface.([]ExportLink)
	if !ok {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Invalid data type",
			"Expected ExportLink slice",
		))
		return
	}

	c.Header("Content-Disposition", "attachment; filename=links.txt")
	c.Header("Content-Type", "text/plain")

	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Download Links Export\n"))
	output.WriteString(fmt.Sprintf("# Thread: %s\n", links[0].ThreadTitle))
	output.WriteString(fmt.Sprintf("# Total Links: %d\n\n", len(links)))

	for _, link := range links {
		output.WriteString(link.URL + "\n")
	}

	c.String(http.StatusOK, output.String())
}

// exportAllThreadsLinks exports links from multiple threads
func exportAllThreadsLinks(c *gin.Context) {
	// Get query parameters
	format := c.DefaultQuery("format", "txt")
	status := c.DefaultQuery("status", "active")
	provider := c.Query("provider")
	limitStr := c.DefaultQuery("limit", "1000")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 10000 {
		limit = 1000
	}

	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Database unavailable",
			"Could not connect to database",
		))
		return
	}

	// Build query for all threads
	query := `
		SELECT
			sdl.id, sdl.url, sdl.provider, sdl.filename,
			sdl.file_size, sdl.file_type, sdl.status,
			st.title as thread_title
		FROM scraped_download_links sdl
		INNER JOIN scraped_threads st ON sdl.thread_id = st.id
		WHERE 1=1
	`
	args := []interface{}{}

	if status == "active" {
		query += " AND sdl.status = 'active'"
	}

	if provider != "" {
		query += " AND LOWER(sdl.provider) = LOWER(?)"
		args = append(args, provider)
	}

	query += " ORDER BY st.id, sdl.id ASC LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to query links",
			err.Error(),
		))
		return
	}
	defer rows.Close()

	type ExportLink struct {
		ID          int64  `json:"id"`
		URL         string `json:"url"`
		Provider    string `json:"provider"`
		Filename    string `json:"filename"`
		FileSize    int64  `json:"file_size"`
		FileType    string `json:"file_type"`
		Status      string `json:"status"`
		ThreadTitle string `json:"thread_title"`
	}

	var links []ExportLink
	for rows.Next() {
		var link ExportLink
		if err := rows.Scan(
			&link.ID, &link.URL, &link.Provider, &link.Filename,
			&link.FileSize, &link.FileType, &link.Status, &link.ThreadTitle,
		); err != nil {
			continue
		}
		links = append(links, link)
	}

	if len(links) == 0 {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"count":   0,
			"message": "No links found",
		}, "No links to export"))
		return
	}

	// Export based on format
	switch format {
	case "json":
		exportJSON(c, links)
	case "csv":
		exportCSV(c, links)
	case "txt":
		exportTXT(c, links)
	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid format",
			"Supported formats: txt, json, csv",
		))
	}
}

// formatFileSize formats bytes to human readable size
func formatFileSize(bytes int64) string {
	if bytes == 0 {
		return "Unknown"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Define ExportLink type at package level for reuse
type ExportLink struct {
	ID          int64  `json:"id"`
	URL         string `json:"url"`
	Provider    string `json:"provider"`
	Filename    string `json:"filename"`
	FileSize    int64  `json:"file_size"`
	FileType    string `json:"file_type"`
	Status      string `json:"status"`
	ThreadTitle string `json:"thread_title"`
}
