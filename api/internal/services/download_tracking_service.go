package services

import (
	"database/sql"
	"fmt"
	"time"
)

// DownloadTrackingService handles download tracking operations
type DownloadTrackingService struct {
	db *sql.DB
}

// NewDownloadTrackingService creates a new download tracking service
func NewDownloadTrackingService(db *sql.DB) *DownloadTrackingService {
	return &DownloadTrackingService{
		db: db,
	}
}

// MarkAsDownloaded marks a link as successfully downloaded
func (s *DownloadTrackingService) MarkAsDownloaded(linkID int64, downloadPath string, notes string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE scraped_download_links
		SET download_status = 'downloaded',
			downloaded_at = ?,
			download_path = ?,
			download_notes = ?
		WHERE id = ?
	`, now, downloadPath, notes, linkID)

	return err
}

// MarkAsFailed marks a link as failed to download
func (s *DownloadTrackingService) MarkAsFailed(linkID int64, notes string) error {
	_, err := s.db.Exec(`
		UPDATE scraped_download_links
		SET download_status = 'failed',
			download_notes = ?
		WHERE id = ?
	`, notes, linkID)

	return err
}

// MarkAsInProgress marks a link as currently downloading
func (s *DownloadTrackingService) MarkAsInProgress(linkID int64) error {
	_, err := s.db.Exec(`
		UPDATE scraped_download_links
		SET download_status = 'in_progress'
		WHERE id = ?
	`, linkID)

	return err
}

// ResetDownloadStatus resets a link back to pending
func (s *DownloadTrackingService) ResetDownloadStatus(linkID int64) error {
	_, err := s.db.Exec(`
		UPDATE scraped_download_links
		SET download_status = 'pending',
			downloaded_at = NULL,
			download_path = NULL,
			download_notes = NULL
		WHERE id = ?
	`, linkID)

	return err
}

// BulkMarkAsDownloaded marks multiple links as downloaded
func (s *DownloadTrackingService) BulkMarkAsDownloaded(linkIDs []int64, downloadPath string, notes string) error {
	if len(linkIDs) == 0 {
		return nil
	}

	now := time.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		UPDATE scraped_download_links
		SET download_status = 'downloaded',
			downloaded_at = ?,
			download_path = ?,
			download_notes = ?
		WHERE id = ?
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, linkID := range linkIDs {
		if _, err := stmt.Exec(now, downloadPath, notes, linkID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetThreadDownloadStats returns download statistics for a thread
func (s *DownloadTrackingService) GetThreadDownloadStats(threadID int64) (map[string]interface{}, error) {
	var stats struct {
		TotalLinks      int
		Downloaded      int
		Pending         int
		Failed          int
		InProgress      int
		DownloadedBytes int64
	}

	// Get count by status
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN download_status = 'downloaded' THEN 1 ELSE 0 END), 0) as downloaded,
			COALESCE(SUM(CASE WHEN download_status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN download_status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN download_status = 'in_progress' THEN 1 ELSE 0 END), 0) as in_progress,
			COALESCE(SUM(CASE WHEN download_status = 'downloaded' THEN file_size ELSE 0 END), 0) as downloaded_bytes
		FROM scraped_download_links
		WHERE thread_id = ?
	`, threadID).Scan(
		&stats.TotalLinks,
		&stats.Downloaded,
		&stats.Pending,
		&stats.Failed,
		&stats.InProgress,
		&stats.DownloadedBytes,
	)

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"total_links":       stats.TotalLinks,
		"downloaded":        stats.Downloaded,
		"pending":           stats.Pending,
		"failed":            stats.Failed,
		"in_progress":       stats.InProgress,
		"downloaded_bytes":  stats.DownloadedBytes,
		"completion_percent": 0,
	}

	if stats.TotalLinks > 0 {
		result["completion_percent"] = int((float64(stats.Downloaded) / float64(stats.TotalLinks)) * 100)
	}

	return result, nil
}

// GetPerformerDownloadStats returns download statistics for a performer across all threads
func (s *DownloadTrackingService) GetPerformerDownloadStats(performerID int64) (map[string]interface{}, error) {
	var stats struct {
		TotalLinks      int
		Downloaded      int
		Pending         int
		Failed          int
		TotalThreads    int
		DownloadedBytes int64
	}

	err := s.db.QueryRow(`
		SELECT
			COUNT(DISTINCT sdl.id) as total_links,
			COALESCE(SUM(CASE WHEN sdl.download_status = 'downloaded' THEN 1 ELSE 0 END), 0) as downloaded,
			COALESCE(SUM(CASE WHEN sdl.download_status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN sdl.download_status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COUNT(DISTINCT sdl.thread_id) as total_threads,
			COALESCE(SUM(CASE WHEN sdl.download_status = 'downloaded' THEN sdl.file_size ELSE 0 END), 0) as downloaded_bytes
		FROM scraped_download_links sdl
		INNER JOIN performer_scraped_threads pst ON sdl.thread_id = pst.thread_id
		WHERE pst.performer_id = ?
	`, performerID).Scan(
		&stats.TotalLinks,
		&stats.Downloaded,
		&stats.Pending,
		&stats.Failed,
		&stats.TotalThreads,
		&stats.DownloadedBytes,
	)

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"total_links":       stats.TotalLinks,
		"downloaded":        stats.Downloaded,
		"pending":           stats.Pending,
		"failed":            stats.Failed,
		"total_threads":     stats.TotalThreads,
		"downloaded_bytes":  stats.DownloadedBytes,
		"completion_percent": 0,
	}

	if stats.TotalLinks > 0 {
		result["completion_percent"] = int((float64(stats.Downloaded) / float64(stats.TotalLinks)) * 100)
	}

	return result, nil
}

// GetGlobalDownloadStats returns overall download statistics
func (s *DownloadTrackingService) GetGlobalDownloadStats() (map[string]interface{}, error) {
	var stats struct {
		TotalLinks      int
		Downloaded      int
		Pending         int
		Failed          int
		InProgress      int
		DownloadedBytes int64
		TotalThreads    int
		CompleteThreads int
	}

	// Get link stats
	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN download_status = 'downloaded' THEN 1 ELSE 0 END), 0) as downloaded,
			COALESCE(SUM(CASE WHEN download_status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN download_status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(SUM(CASE WHEN download_status = 'in_progress' THEN 1 ELSE 0 END), 0) as in_progress,
			COALESCE(SUM(CASE WHEN download_status = 'downloaded' THEN file_size ELSE 0 END), 0) as downloaded_bytes
		FROM scraped_download_links
	`).Scan(
		&stats.TotalLinks,
		&stats.Downloaded,
		&stats.Pending,
		&stats.Failed,
		&stats.InProgress,
		&stats.DownloadedBytes,
	)

	if err != nil {
		return nil, err
	}

	// Get thread stats
	_ = s.db.QueryRow(`
		SELECT
			COUNT(DISTINCT thread_id) as total_threads,
			COUNT(DISTINCT CASE
				WHEN (SELECT COUNT(*) FROM scraped_download_links WHERE thread_id = st.id AND download_status != 'downloaded') = 0
				THEN thread_id
			END) as complete_threads
		FROM scraped_threads st
		WHERE EXISTS (SELECT 1 FROM scraped_download_links WHERE thread_id = st.id)
	`).Scan(&stats.TotalThreads, &stats.CompleteThreads)

	result := map[string]interface{}{
		"total_links":       stats.TotalLinks,
		"downloaded":        stats.Downloaded,
		"pending":           stats.Pending,
		"failed":            stats.Failed,
		"in_progress":       stats.InProgress,
		"downloaded_bytes":  stats.DownloadedBytes,
		"total_threads":     stats.TotalThreads,
		"complete_threads":  stats.CompleteThreads,
		"completion_percent": 0,
	}

	if stats.TotalLinks > 0 {
		result["completion_percent"] = int((float64(stats.Downloaded) / float64(stats.TotalLinks)) * 100)
	}

	return result, nil
}

// GetThreadsByDownloadStatus returns threads filtered by download completion
func (s *DownloadTrackingService) GetThreadsByDownloadStatus(status string, limit int, offset int) ([]map[string]interface{}, error) {
	// status can be: "complete", "partial", "none", "all"
	query := `
		SELECT
			st.id,
			st.title,
			st.url,
			st.download_count,
			COUNT(sdl.id) as total_links,
			COALESCE(SUM(CASE WHEN sdl.download_status = 'downloaded' THEN 1 ELSE 0 END), 0) as downloaded,
			COALESCE(SUM(CASE WHEN sdl.download_status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN sdl.download_status = 'failed' THEN 1 ELSE 0 END), 0) as failed
		FROM scraped_threads st
		LEFT JOIN scraped_download_links sdl ON st.id = sdl.thread_id
		WHERE 1=1
	`

	switch status {
	case "complete":
		query += ` AND (SELECT COUNT(*) FROM scraped_download_links WHERE thread_id = st.id AND download_status != 'downloaded') = 0`
	case "partial":
		query += ` AND EXISTS (SELECT 1 FROM scraped_download_links WHERE thread_id = st.id AND download_status = 'downloaded')
				   AND EXISTS (SELECT 1 FROM scraped_download_links WHERE thread_id = st.id AND download_status != 'downloaded')`
	case "none":
		query += ` AND NOT EXISTS (SELECT 1 FROM scraped_download_links WHERE thread_id = st.id AND download_status = 'downloaded')`
	}

	query += `
		GROUP BY st.id, st.title, st.url, st.download_count
		ORDER BY st.last_scraped_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []map[string]interface{}
	for rows.Next() {
		var t struct {
			ID            int64
			Title         string
			URL           string
			DownloadCount int
			TotalLinks    int
			Downloaded    int
			Pending       int
			Failed        int
		}

		if err := rows.Scan(&t.ID, &t.Title, &t.URL, &t.DownloadCount, &t.TotalLinks, &t.Downloaded, &t.Pending, &t.Failed); err != nil {
			continue
		}

		completionPercent := 0
		if t.TotalLinks > 0 {
			completionPercent = int((float64(t.Downloaded) / float64(t.TotalLinks)) * 100)
		}

		threads = append(threads, map[string]interface{}{
			"id":                 t.ID,
			"title":              t.Title,
			"url":                t.URL,
			"download_count":     t.DownloadCount,
			"total_links":        t.TotalLinks,
			"downloaded":         t.Downloaded,
			"pending":            t.Pending,
			"failed":             t.Failed,
			"completion_percent": completionPercent,
		})
	}

	return threads, nil
}

// GetRecentDownloads returns recently downloaded links
func (s *DownloadTrackingService) GetRecentDownloads(limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			sdl.id,
			sdl.url,
			sdl.filename,
			sdl.file_size,
			sdl.provider,
			sdl.downloaded_at,
			sdl.download_path,
			st.title as thread_title,
			st.id as thread_id
		FROM scraped_download_links sdl
		INNER JOIN scraped_threads st ON sdl.thread_id = st.id
		WHERE sdl.download_status = 'downloaded'
		ORDER BY sdl.downloaded_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var downloads []map[string]interface{}
	for rows.Next() {
		var d struct {
			ID           int64
			URL          string
			Filename     string
			FileSize     int64
			Provider     string
			DownloadedAt time.Time
			DownloadPath string
			ThreadTitle  string
			ThreadID     int64
		}

		if err := rows.Scan(&d.ID, &d.URL, &d.Filename, &d.FileSize, &d.Provider, &d.DownloadedAt, &d.DownloadPath, &d.ThreadTitle, &d.ThreadID); err != nil {
			continue
		}

		downloads = append(downloads, map[string]interface{}{
			"id":            d.ID,
			"url":           d.URL,
			"filename":      d.Filename,
			"file_size":     d.FileSize,
			"provider":      d.Provider,
			"downloaded_at": d.DownloadedAt,
			"download_path": d.DownloadPath,
			"thread_title":  d.ThreadTitle,
			"thread_id":     d.ThreadID,
		})
	}

	return downloads, nil
}

// FormatFileSize converts bytes to human-readable format
func FormatFileSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
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
