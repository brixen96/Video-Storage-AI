package services

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// AutoUpdateService manages auto-update configurations and coordinates updates
type AutoUpdateService struct {
	db                  *sql.DB
	activityService     *ActivityService
	notificationService *NotificationService
}

// NewAutoUpdateService creates a new AutoUpdateService
func NewAutoUpdateService(activitySvc *ActivityService, notificationSvc *NotificationService) *AutoUpdateService {
	return &AutoUpdateService{
		db:                  database.GetDB(),
		activityService:     activitySvc,
		notificationService: notificationSvc,
	}
}

// GetAllConfigs returns all scraper auto-update configurations
func (s *AutoUpdateService) GetAllConfigs() ([]*models.ScraperAutoUpdateConfig, error) {
	rows, err := s.db.Query(`
		SELECT id, scraper_type, enabled, interval_minutes, max_pages_to_check,
			   last_checked_at, last_found_count, total_found_count, check_count,
			   config_json, created_at, updated_at
		FROM scraper_auto_update_configs
		ORDER BY scraper_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.ScraperAutoUpdateConfig
	for rows.Next() {
		config := &models.ScraperAutoUpdateConfig{}
		err := rows.Scan(
			&config.ID, &config.ScraperType, &config.Enabled,
			&config.IntervalMinutes, &config.MaxPagesToCheck,
			&config.LastCheckedAt, &config.LastFoundCount,
			&config.TotalFoundCount, &config.CheckCount,
			&config.ConfigJSON, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			continue
		}
		config.ParseConfig()
		configs = append(configs, config)
	}

	return configs, nil
}

// GetConfig returns config for a specific scraper type
func (s *AutoUpdateService) GetConfig(scraperType string) (*models.ScraperAutoUpdateConfig, error) {
	config := &models.ScraperAutoUpdateConfig{}
	err := s.db.QueryRow(`
		SELECT id, scraper_type, enabled, interval_minutes, max_pages_to_check,
			   last_checked_at, last_found_count, total_found_count, check_count,
			   config_json, created_at, updated_at
		FROM scraper_auto_update_configs
		WHERE scraper_type = ?
	`, scraperType).Scan(
		&config.ID, &config.ScraperType, &config.Enabled,
		&config.IntervalMinutes, &config.MaxPagesToCheck,
		&config.LastCheckedAt, &config.LastFoundCount,
		&config.TotalFoundCount, &config.CheckCount,
		&config.ConfigJSON, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("config not found for scraper type: %s", scraperType)
		}
		return nil, err
	}
	config.ParseConfig()
	return config, nil
}

// GetOrCreateConfig returns config for a scraper type, creating a default one if it doesn't exist
func (s *AutoUpdateService) GetOrCreateConfig(scraperType string) (*models.ScraperAutoUpdateConfig, error) {
	config, err := s.GetConfig(scraperType)
	if err == nil {
		return config, nil
	}

	// Config doesn't exist, create a new one with defaults
	_, err = s.db.Exec(`
		INSERT INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json)
		VALUES (?, 0, 60, 3, '{}')
	`, scraperType)
	if err != nil {
		return nil, fmt.Errorf("failed to create config for %s: %w", scraperType, err)
	}

	return s.GetConfig(scraperType)
}

// UpdateConfig updates a scraper's auto-update configuration
func (s *AutoUpdateService) UpdateConfig(scraperType string, update *models.AutoUpdateConfigUpdate) error {
	// Build dynamic update query
	setClauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{}

	if update.Enabled != nil {
		setClauses = append(setClauses, "enabled = ?")
		args = append(args, *update.Enabled)
	}
	if update.IntervalMinutes != nil {
		setClauses = append(setClauses, "interval_minutes = ?")
		args = append(args, *update.IntervalMinutes)
	}
	if update.MaxPagesToCheck != nil {
		setClauses = append(setClauses, "max_pages_to_check = ?")
		args = append(args, *update.MaxPagesToCheck)
	}
	if update.ConfigJSON != nil {
		setClauses = append(setClauses, "config_json = ?")
		args = append(args, *update.ConfigJSON)
	}

	args = append(args, scraperType)

	query := fmt.Sprintf("UPDATE scraper_auto_update_configs SET %s WHERE scraper_type = ?",
		joinStrings(setClauses, ", "))

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("config not found for scraper type: %s", scraperType)
	}

	return nil
}

// ToggleEnabled enables or disables auto-update for a scraper
func (s *AutoUpdateService) ToggleEnabled(scraperType string, enabled bool) error {
	result, err := s.db.Exec(`
		UPDATE scraper_auto_update_configs
		SET enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE scraper_type = ?
	`, enabled, scraperType)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("config not found for scraper type: %s", scraperType)
	}

	log.Printf("Auto-update for %s: %s", scraperType, map[bool]string{true: "enabled", false: "disabled"}[enabled])
	return nil
}

// RecordCheckResult records the result of an auto-update check
func (s *AutoUpdateService) RecordCheckResult(scraperType string, foundCount int) error {
	_, err := s.db.Exec(`
		UPDATE scraper_auto_update_configs
		SET last_checked_at = CURRENT_TIMESTAMP,
			last_found_count = ?,
			total_found_count = total_found_count + ?,
			check_count = check_count + 1,
			updated_at = CURRENT_TIMESTAMP
		WHERE scraper_type = ?
	`, foundCount, foundCount, scraperType)
	return err
}

// GetCheckpoint returns the checkpoint for a scraper/target combination
func (s *AutoUpdateService) GetCheckpoint(scraperType, checkpointType, targetID string) (*models.ScraperCheckpoint, error) {
	checkpoint := &models.ScraperCheckpoint{}

	var query string
	var args []interface{}

	if targetID == "" {
		query = `SELECT id, scraper_type, checkpoint_type, checkpoint_value, target_id, created_at, updated_at
				 FROM scraper_checkpoints WHERE scraper_type = ? AND checkpoint_type = ? AND target_id IS NULL`
		args = []interface{}{scraperType, checkpointType}
	} else {
		query = `SELECT id, scraper_type, checkpoint_type, checkpoint_value, target_id, created_at, updated_at
				 FROM scraper_checkpoints WHERE scraper_type = ? AND checkpoint_type = ? AND target_id = ?`
		args = []interface{}{scraperType, checkpointType, targetID}
	}

	var targetIDNullable sql.NullString
	err := s.db.QueryRow(query, args...).Scan(
		&checkpoint.ID, &checkpoint.ScraperType, &checkpoint.CheckpointType,
		&checkpoint.CheckpointValue, &targetIDNullable,
		&checkpoint.CreatedAt, &checkpoint.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No checkpoint exists yet
		}
		return nil, err
	}

	if targetIDNullable.Valid {
		checkpoint.TargetID = targetIDNullable.String
	}

	return checkpoint, nil
}

// SaveCheckpoint saves or updates a checkpoint
func (s *AutoUpdateService) SaveCheckpoint(checkpoint *models.ScraperCheckpoint) error {
	var targetID interface{}
	if checkpoint.TargetID == "" {
		targetID = nil
	} else {
		targetID = checkpoint.TargetID
	}

	_, err := s.db.Exec(`
		INSERT INTO scraper_checkpoints (scraper_type, checkpoint_type, checkpoint_value, target_id, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(scraper_type, checkpoint_type, target_id) DO UPDATE SET
			checkpoint_value = excluded.checkpoint_value,
			updated_at = CURRENT_TIMESTAMP
	`, checkpoint.ScraperType, checkpoint.CheckpointType, checkpoint.CheckpointValue, targetID)

	return err
}

// GetEnabledConfigs returns all enabled auto-update configurations
func (s *AutoUpdateService) GetEnabledConfigs() ([]*models.ScraperAutoUpdateConfig, error) {
	rows, err := s.db.Query(`
		SELECT id, scraper_type, enabled, interval_minutes, max_pages_to_check,
			   last_checked_at, last_found_count, total_found_count, check_count,
			   config_json, created_at, updated_at
		FROM scraper_auto_update_configs
		WHERE enabled = 1
		ORDER BY scraper_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.ScraperAutoUpdateConfig
	for rows.Next() {
		config := &models.ScraperAutoUpdateConfig{}
		err := rows.Scan(
			&config.ID, &config.ScraperType, &config.Enabled,
			&config.IntervalMinutes, &config.MaxPagesToCheck,
			&config.LastCheckedAt, &config.LastFoundCount,
			&config.TotalFoundCount, &config.CheckCount,
			&config.ConfigJSON, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			continue
		}
		config.ParseConfig()
		configs = append(configs, config)
	}

	return configs, nil
}

// GetDueConfigs returns configs that are due for a check
func (s *AutoUpdateService) GetDueConfigs() ([]*models.ScraperAutoUpdateConfig, error) {
	rows, err := s.db.Query(`
		SELECT id, scraper_type, enabled, interval_minutes, max_pages_to_check,
			   last_checked_at, last_found_count, total_found_count, check_count,
			   config_json, created_at, updated_at
		FROM scraper_auto_update_configs
		WHERE enabled = 1
		  AND (last_checked_at IS NULL
			   OR datetime(last_checked_at, '+' || interval_minutes || ' minutes') <= datetime('now'))
		ORDER BY last_checked_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.ScraperAutoUpdateConfig
	for rows.Next() {
		config := &models.ScraperAutoUpdateConfig{}
		err := rows.Scan(
			&config.ID, &config.ScraperType, &config.Enabled,
			&config.IntervalMinutes, &config.MaxPagesToCheck,
			&config.LastCheckedAt, &config.LastFoundCount,
			&config.TotalFoundCount, &config.CheckCount,
			&config.ConfigJSON, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			continue
		}
		config.ParseConfig()
		configs = append(configs, config)
	}

	return configs, nil
}

// CreateNotification creates a notification for new content found
func (s *AutoUpdateService) CreateNotification(scraperType string, foundCount int, details map[string]interface{}) error {
	if s.notificationService == nil {
		return nil // Notification service not available
	}

	displayName, _, _ := models.GetScraperDisplayInfo(scraperType)

	title := fmt.Sprintf("New content from %s", displayName)
	message := fmt.Sprintf("Found %d new items", foundCount)

	notification := &models.CreateNotificationRequest{
		Type:     "new_content",
		Priority: "normal",
		Title:    title,
		Message:  message,
		Category: "scraper",
		Metadata: details,
	}

	_, err := s.notificationService.Create(notification)
	return err
}

// GetScraperStatusCards returns status cards for all scrapers
func (s *AutoUpdateService) GetScraperStatusCards() ([]*models.ScraperStatusCard, error) {
	configs, err := s.GetAllConfigs()
	if err != nil {
		return nil, err
	}

	var cards []*models.ScraperStatusCard
	for _, config := range configs {
		displayName, icon, color := models.GetScraperDisplayInfo(config.ScraperType)

		// Check if this is a custom tab and look up its name
		if strings.HasPrefix(config.ScraperType, "custom_") {
			tabIDStr := strings.TrimPrefix(config.ScraperType, "custom_")
			tabID, err := strconv.ParseInt(tabIDStr, 10, 64)
			if err == nil {
				var tabName, tabIcon, tabColor string
				err := s.db.QueryRow("SELECT name, icon, color FROM scraper_tabs WHERE id = ?", tabID).Scan(&tabName, &tabIcon, &tabColor)
				if err == nil {
					displayName = tabName
					if tabIcon != "" {
						icon = tabIcon
					}
					if tabColor != "" {
						color = tabColor
					}
				}
			}
		}

		card := &models.ScraperStatusCard{
			ScraperType:     config.ScraperType,
			DisplayName:     displayName,
			Icon:            icon,
			Color:           color,
			Enabled:         config.Enabled,
			LastCheckedAt:   config.LastCheckedAt,
			LastFoundCount:  config.LastFoundCount,
			TotalFoundCount: config.TotalFoundCount,
			IntervalMinutes: config.IntervalMinutes,
			Status:          "idle",
		}

		// Calculate next scheduled run
		if config.Enabled && config.LastCheckedAt != nil {
			nextRun := config.LastCheckedAt.Add(time.Duration(config.IntervalMinutes) * time.Minute)
			card.NextScheduledRun = &nextRun
		}

		// Get total items count for this scraper
		card.TotalItems = s.getScraperItemCount(config.ScraperType)

		cards = append(cards, card)
	}

	return cards, nil
}

// getScraperItemCount returns the total number of items for a scraper
func (s *AutoUpdateService) getScraperItemCount(scraperType string) int {
	var count int

	switch scraperType {
	case "simpcity":
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_threads WHERE source = 'simpcity'").Scan(&count)
	case "pornhub":
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'pornhub'").Scan(&count)
	case "rule34video":
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'rule34video'").Scan(&count)
	case "luxuretv":
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'luxuretv'").Scan(&count)
	case "sexuria":
		s.db.QueryRow("SELECT COUNT(*) FROM scraped_videos WHERE source = 'sexuria'").Scan(&count)
	}

	return count
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
