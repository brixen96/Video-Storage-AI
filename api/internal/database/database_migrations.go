package database

import (
	"fmt"
	"log"
	"strings"
)

// runMigrations applies all schema migrations in order.
// Migrations are idempotent — duplicate-column and already-exists errors are silently ignored.
func runMigrations() error {
	migrations := []string{
		// Migration 1: Add primary_lib column to libraries table if it doesn't exist
		`
		ALTER TABLE libraries ADD COLUMN primary_lib BOOLEAN DEFAULT 0
	`,
		// Migration 2: Add not_interested column to videos table
		`
		ALTER TABLE videos ADD COLUMN not_interested BOOLEAN DEFAULT 0
	`,
		// Migration 3: Add in_edit_list column to videos table
		`
		ALTER TABLE videos ADD COLUMN in_edit_list BOOLEAN DEFAULT 0
	`,
		// Migration 4: Add zoo column to performers table
		`
		ALTER TABLE performers ADD COLUMN zoo BOOLEAN DEFAULT 0
	`,
		// Migration 5: Add conversion link columns to videos table
		`
		ALTER TABLE videos ADD COLUMN converted_from INTEGER
	`,
		`
		ALTER TABLE videos ADD COLUMN converted_to INTEGER
	`,
		`CREATE TABLE IF NOT EXISTS activities (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            task_type TEXT NOT NULL,
            status TEXT NOT NULL,
            message TEXT,
            details TEXT,
            progress INTEGER DEFAULT 0,
            started_at DATETIME NOT NULL,
            completed_at DATETIME,
            error TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE INDEX IF NOT EXISTS idx_activities_status ON activities(status)`,
		`CREATE INDEX IF NOT EXISTS idx_activities_task_type ON activities(task_type)`,
		`CREATE INDEX IF NOT EXISTS idx_activities_started_at ON activities(started_at)`,

		`CREATE TABLE IF NOT EXISTS activity_logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            task_type TEXT NOT NULL,
            status TEXT NOT NULL,
            message TEXT,
            progress INTEGER DEFAULT 0,
            started_at DATETIME NOT NULL,
            completed_at DATETIME,
            details TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_status ON activity_logs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_task_type ON activity_logs(task_type)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_started_at ON activity_logs(started_at)`,
		// Migration 5: Add updated_at column to activity_logs table
		`ALTER TABLE activity_logs ADD COLUMN updated_at DATETIME`,
		// Migration 6: Add error column to activity_logs table
		`ALTER TABLE activity_logs ADD COLUMN error TEXT`,
		// Migration 7: Add date column to videos table
		`ALTER TABLE videos ADD COLUMN date TEXT`,
		// Migration 8: Add rating column to videos table
		`ALTER TABLE videos ADD COLUMN rating INTEGER DEFAULT 0`,
		// Migration 9: Add description column to videos table
		`ALTER TABLE videos ADD COLUMN description TEXT`,
		// Migration 10: Add is_favorite column to videos table
		`ALTER TABLE videos ADD COLUMN is_favorite BOOLEAN DEFAULT 0`,
		// Migration 11: Add is_pinned column to videos table
		`ALTER TABLE videos ADD COLUMN is_pinned BOOLEAN DEFAULT 0`,
		// Migration 12: Add preview_path column to videos table
		`ALTER TABLE videos ADD COLUMN preview_path TEXT`,
		// Migration 13: Create memories table for AI Companion
		`CREATE TABLE IF NOT EXISTS memories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			value TEXT NOT NULL,
			category TEXT NOT NULL,
			importance INTEGER DEFAULT 5,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_importance ON memories(importance)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_key ON memories(key)`,
		// Migration 14: Add category column to tags table
		`ALTER TABLE tags ADD COLUMN category TEXT DEFAULT 'regular'`,
		// Migration 15: Add category to performers and migrate zoo data
		`ALTER TABLE performers ADD COLUMN category TEXT DEFAULT 'regular'`,
		`UPDATE performers SET category = 'zoo' WHERE zoo = 1 AND category = 'regular'`,
		`UPDATE performers SET category = 'regular' WHERE category IS NULL`,
		// Migration 16: Add category to studios
		`ALTER TABLE studios ADD COLUMN category TEXT DEFAULT 'regular'`,
		// Migration 17: Add category to groups
		`ALTER TABLE groups ADD COLUMN category TEXT DEFAULT 'regular'`,
		// Migration 18: Add indexes for category fields
		`CREATE INDEX IF NOT EXISTS idx_performers_category ON performers(category)`,
		`CREATE INDEX IF NOT EXISTS idx_studios_category ON studios(category)`,
		`CREATE INDEX IF NOT EXISTS idx_groups_category ON groups(category)`,
		// Migration 19: Moved to conditional block below
		`CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(category)`,
		// Migration 20: Add video_count to performers
		`ALTER TABLE performers ADD COLUMN video_count INTEGER DEFAULT 0`,
		`UPDATE performers SET video_count = (
			SELECT COUNT(*) FROM video_performers WHERE performer_id = performers.id
		)`,
		// Migration 21: Moved to conditional block below
		// Migration 22: Add thumbnail_path to performers
		`ALTER TABLE performers ADD COLUMN thumbnail_path TEXT DEFAULT ''`,
		// Migration 23: Create scraper tables
		`CREATE TABLE IF NOT EXISTS scraped_threads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			external_id TEXT NOT NULL,
			source TEXT NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			author TEXT,
			category TEXT,
			view_count INTEGER DEFAULT 0,
			reply_count INTEGER DEFAULT 0,
			post_count INTEGER DEFAULT 0,
			download_count INTEGER DEFAULT 0,
			metadata TEXT DEFAULT '{}',
			first_scraped_at DATETIME NOT NULL,
			last_scraped_at DATETIME NOT NULL,
			last_updated_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(external_id, source)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_threads_source ON scraped_threads(source)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_threads_category ON scraped_threads(category)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_threads_last_scraped ON scraped_threads(last_scraped_at)`,
		`CREATE TABLE IF NOT EXISTS scraped_posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			thread_id INTEGER NOT NULL,
			external_id TEXT NOT NULL,
			source TEXT NOT NULL,
			author TEXT,
			content TEXT,
			plain_text TEXT,
			post_number INTEGER DEFAULT 0,
			like_count INTEGER DEFAULT 0,
			metadata TEXT DEFAULT '{}',
			posted_at DATETIME,
			scraped_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (thread_id) REFERENCES scraped_threads(id) ON DELETE CASCADE,
			UNIQUE(external_id, source)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_posts_thread ON scraped_posts(thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_posts_source ON scraped_posts(source)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_posts_author ON scraped_posts(author)`,
		`CREATE TABLE IF NOT EXISTS scraped_download_links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			thread_id INTEGER NOT NULL,
			post_id INTEGER NOT NULL,
			source TEXT NOT NULL,
			provider TEXT NOT NULL,
			url TEXT NOT NULL,
			original_url TEXT,
			filename TEXT,
			file_size INTEGER DEFAULT 0,
			file_type TEXT,
			status TEXT DEFAULT 'active',
			metadata TEXT DEFAULT '{}',
			discovered_at DATETIME NOT NULL,
			last_checked_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (thread_id) REFERENCES scraped_threads(id) ON DELETE CASCADE,
			FOREIGN KEY (post_id) REFERENCES scraped_posts(id) ON DELETE CASCADE,
			UNIQUE(url, source)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_links_thread ON scraped_download_links(thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_links_post ON scraped_download_links(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_links_provider ON scraped_download_links(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_links_status ON scraped_download_links(status)`,
		`CREATE TABLE IF NOT EXISTS scraper_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_type TEXT NOT NULL,
			source TEXT NOT NULL,
			target_url TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			progress INTEGER DEFAULT 0,
			items_processed INTEGER DEFAULT 0,
			items_total INTEGER DEFAULT 0,
			error_message TEXT,
			result TEXT DEFAULT '{}',
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_jobs_status ON scraper_jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_jobs_source ON scraper_jobs(source)`,
		`CREATE TABLE IF NOT EXISTS ai_companion_memory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			content TEXT NOT NULL,
			metadata TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_memory_type ON ai_companion_memory(type)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_memory_created ON ai_companion_memory(created_at)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Migration 24: Create performer_scraped_threads linking table
		`CREATE TABLE IF NOT EXISTS performer_scraped_threads (
			performer_id INTEGER NOT NULL,
			thread_id INTEGER NOT NULL,
			confidence REAL DEFAULT 1.0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (performer_id, thread_id),
			FOREIGN KEY (performer_id) REFERENCES performers(id) ON DELETE CASCADE,
			FOREIGN KEY (thread_id) REFERENCES scraped_threads(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_performer_threads_performer ON performer_scraped_threads(performer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_performer_threads_thread ON performer_scraped_threads(thread_id)`,
		// Migration 25: Create console_logs table for system-wide logging
		`CREATE TABLE IF NOT EXISTS console_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			details TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_console_logs_source ON console_logs(source)`,
		`CREATE INDEX IF NOT EXISTS idx_console_logs_level ON console_logs(level)`,
		`CREATE INDEX IF NOT EXISTS idx_console_logs_created ON console_logs(created_at DESC)`,
		// Migration 26: Add pause/resume support to activities
		`ALTER TABLE activities ADD COLUMN is_paused BOOLEAN DEFAULT 0`,
		`ALTER TABLE activities ADD COLUMN paused_at DATETIME`,
		`ALTER TABLE activities ADD COLUMN checkpoint TEXT DEFAULT '{}'`,
		`ALTER TABLE activity_logs ADD COLUMN is_paused BOOLEAN DEFAULT 0`,
		`ALTER TABLE activity_logs ADD COLUMN paused_at DATETIME`,
		`ALTER TABLE activity_logs ADD COLUMN checkpoint TEXT DEFAULT '{}'`,
		// Migration 27: Performance indexes for pause/resume and AI search
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_is_paused ON activity_logs(is_paused) WHERE is_paused = 1`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_status_paused ON activity_logs(status, is_paused)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_posts_plain_text ON scraped_posts(plain_text)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_threads_title ON scraped_threads(title)`,
		// Migration 28: Additional performance indexes
		`CREATE INDEX IF NOT EXISTS idx_video_performers_performer ON video_performers(performer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_tags_tag ON video_tags(tag_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_studios_studio ON video_studios(studio_id)`,
		`CREATE INDEX IF NOT EXISTS idx_performer_scraped_threads_thread ON performer_scraped_threads(thread_id)`,
		// Migration 29: Download tracking fields
		`ALTER TABLE scraped_download_links ADD COLUMN download_status TEXT DEFAULT 'pending'`,
		`ALTER TABLE scraped_download_links ADD COLUMN downloaded_at DATETIME`,
		`ALTER TABLE scraped_download_links ADD COLUMN download_path TEXT`,
		`ALTER TABLE scraped_download_links ADD COLUMN download_notes TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_links_download_status ON scraped_download_links(download_status)`,
		// Migration 30: AI Audit Logging System
		`CREATE TABLE IF NOT EXISTS ai_audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			interaction_type TEXT NOT NULL,
			operation TEXT NOT NULL,
			user_query TEXT,
			ai_prompt TEXT,
			ai_response TEXT,
			context_data TEXT DEFAULT '{}',
			performer_id INTEGER,
			thread_id INTEGER,
			video_id INTEGER,
			tokens_used INTEGER DEFAULT 0,
			cost_usd REAL DEFAULT 0.0,
			response_time_ms INTEGER DEFAULT 0,
			success BOOLEAN DEFAULT 1,
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (performer_id) REFERENCES performers(id) ON DELETE SET NULL,
			FOREIGN KEY (thread_id) REFERENCES scraped_threads(id) ON DELETE SET NULL,
			FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_audit_interaction_type ON ai_audit_logs(interaction_type)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_audit_operation ON ai_audit_logs(operation)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_audit_created ON ai_audit_logs(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_audit_performer ON ai_audit_logs(performer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_audit_thread ON ai_audit_logs(thread_id)`,
		// Migration 31: Scheduled Jobs System
		`CREATE TABLE IF NOT EXISTS scheduled_jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_type TEXT NOT NULL,
			schedule_type TEXT NOT NULL,
			schedule_config TEXT DEFAULT '{}',
			target_type TEXT,
			target_id INTEGER,
			enabled BOOLEAN DEFAULT 1,
			last_run_at DATETIME,
			next_run_at DATETIME,
			run_count INTEGER DEFAULT 0,
			success_count INTEGER DEFAULT 0,
			failure_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_enabled ON scheduled_jobs(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_next_run ON scheduled_jobs(next_run_at)`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_jobs_type ON scheduled_jobs(job_type)`,
		`CREATE TABLE IF NOT EXISTS job_execution_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			started_at DATETIME NOT NULL,
			completed_at DATETIME,
			duration_ms INTEGER DEFAULT 0,
			result_data TEXT DEFAULT '{}',
			error_message TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (job_id) REFERENCES scheduled_jobs(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_job_history_job ON job_execution_history(job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_job_history_status ON job_execution_history(status)`,
		`CREATE INDEX IF NOT EXISTS idx_job_history_created ON job_execution_history(created_at DESC)`,
	}

	for i, migration := range migrations {
		if _, err := DB.Exec(migration); err != nil {
			errMsg := err.Error()
			if !isColumnExistsError(err) && !strings.Contains(errMsg, "no such column") {
				return fmt.Errorf("migration %d failed: %w", i+1, err)
			}
		}
	}

	// Migration 32: Generic Video Scraper Tables
	migration32VideoScraper := []string{
		`CREATE TABLE IF NOT EXISTS scraped_videos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			site_type TEXT NOT NULL,
			external_id TEXT,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			description TEXT,
			duration INTEGER DEFAULT 0,
			file_size INTEGER DEFAULT 0,
			thumbnail_url TEXT,
			thumbnail_path TEXT,
			video_path TEXT,
			author TEXT,
			uploader TEXT,
			view_count INTEGER DEFAULT 0,
			like_count INTEGER DEFAULT 0,
			category TEXT,
			tags TEXT DEFAULT '[]',
			metadata TEXT DEFAULT '{}',
			status TEXT DEFAULT 'pending',
			error_message TEXT,
			discovered_at DATETIME NOT NULL,
			scraped_at DATETIME,
			downloaded_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(url, source)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_videos_source ON scraped_videos(source)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_videos_site_type ON scraped_videos(site_type)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_videos_status ON scraped_videos(status)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_videos_discovered ON scraped_videos(discovered_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_videos_title ON scraped_videos(title)`,
	}
	for _, migration := range migration32VideoScraper {
		if _, err := DB.Exec(migration); err != nil {
			if !isColumnExistsError(err) {
				return fmt.Errorf("migration 32 (video scraper) failed: %w", err)
			}
		}
	}

	// Migration 32b: Add rating column to scraped_videos
	if _, err := DB.Exec(`ALTER TABLE scraped_videos ADD COLUMN rating INTEGER DEFAULT 0`); err != nil {
		if !isColumnExistsError(err) {
			return fmt.Errorf("migration 32b (add rating to scraped_videos) failed: %w", err)
		}
	}

	// Migration 33: Notifications System
	migration33 := []string{
		`CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			priority TEXT NOT NULL DEFAULT 'normal',
			title TEXT NOT NULL,
			message TEXT NOT NULL,
			category TEXT,
			action_url TEXT,
			action_label TEXT,
			metadata TEXT DEFAULT '{}',
			is_read BOOLEAN DEFAULT 0,
			is_archived BOOLEAN DEFAULT 0,
			related_entity_type TEXT,
			related_entity_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			read_at DATETIME,
			expires_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(is_read)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_archived ON notifications(is_archived)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_priority ON notifications(priority)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_entity ON notifications(related_entity_type, related_entity_id)`,
		`CREATE TABLE IF NOT EXISTS notification_preferences (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			notification_type TEXT UNIQUE NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			priority_filter TEXT DEFAULT 'all',
			delivery_methods TEXT DEFAULT '["in_app"]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, migration := range migration33 {
		if _, err := DB.Exec(migration); err != nil {
			if !isColumnExistsError(err) {
				return fmt.Errorf("migration 33 (notifications) failed: %w", err)
			}
		}
	}

	// Migration 34: Scraper Tabs Table
	if _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS scraper_tabs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		tab_type TEXT NOT NULL DEFAULT 'video',
		base_url TEXT NOT NULL,
		site_type TEXT NOT NULL,
		icon TEXT DEFAULT 'video',
		color TEXT DEFAULT '#007bff',
		scrape_options TEXT DEFAULT '{}',
		is_active BOOLEAN DEFAULT 1,
		position INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		if !isColumnExistsError(err) {
			return fmt.Errorf("migration 34 (scraper tabs table) failed: %w", err)
		}
	}
	for _, idx := range []string{
		`CREATE INDEX IF NOT EXISTS idx_scraper_tabs_active ON scraper_tabs(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_tabs_position ON scraper_tabs(position)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_tabs_type ON scraper_tabs(tab_type)`,
	} {
		if _, err := DB.Exec(idx); err != nil {
			if !isColumnExistsError(err) {
				return fmt.Errorf("migration 34 (scraper tabs indexes) failed: %w", err)
			}
		}
	}

	// Migration 35: Scraper Tab Jobs Table
	if _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS scraper_tab_jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tab_id INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		url TEXT NOT NULL,
		total_pages INTEGER DEFAULT 0,
		pages_completed INTEGER DEFAULT 0,
		videos_found INTEGER DEFAULT 0,
		videos_saved INTEGER DEFAULT 0,
		errors_count INTEGER DEFAULT 0,
		error_messages TEXT DEFAULT '[]',
		started_at DATETIME,
		completed_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (tab_id) REFERENCES scraper_tabs(id) ON DELETE CASCADE
	)`); err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "duplicate") {
			return fmt.Errorf("migration 35 (scraper tab jobs table) failed: %w", err)
		}
	}
	scraperTabJobsIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_scraper_tab_jobs_tab ON scraper_tab_jobs(tab_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_tab_jobs_status ON scraper_tab_jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_tab_jobs_created ON scraper_tab_jobs(created_at DESC)`,
	}
	for _, idx := range scraperTabJobsIndexes {
		if _, err := DB.Exec(idx); err != nil {
			errMsg := err.Error()
			if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "no such column") {
				return fmt.Errorf("migration 35 (scraper jobs indexes) failed: %w", err)
			}
		}
	}

	// Migration 36: Add job_type column to scraper_tab_jobs
	DB.Exec(`ALTER TABLE scraper_tab_jobs ADD COLUMN job_type TEXT DEFAULT 'tab'`)

	// Migration 37: Recreate scraper_tab_jobs with nullable tab_id
	var needsMigration bool
	if _, err := DB.Exec(`INSERT INTO scraper_tab_jobs (tab_id, status, url, job_type) VALUES (NULL, 'test', 'test', 'test')`); err != nil {
		needsMigration = true
	} else {
		DB.Exec(`DELETE FROM scraper_tab_jobs WHERE status = 'test' AND url = 'test'`)
	}
	if needsMigration {
		migrationSteps := []string{
			`CREATE TABLE IF NOT EXISTS scraper_tab_jobs_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				tab_id INTEGER,
				job_type TEXT DEFAULT 'tab',
				status TEXT NOT NULL DEFAULT 'pending',
				url TEXT NOT NULL,
				total_pages INTEGER DEFAULT 0,
				pages_completed INTEGER DEFAULT 0,
				videos_found INTEGER DEFAULT 0,
				videos_saved INTEGER DEFAULT 0,
				errors_count INTEGER DEFAULT 0,
				error_messages TEXT DEFAULT '[]',
				started_at DATETIME,
				completed_at DATETIME,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (tab_id) REFERENCES scraper_tabs(id) ON DELETE CASCADE
			)`,
			`INSERT INTO scraper_tab_jobs_new SELECT id, tab_id, COALESCE(job_type, 'tab'), status, url, total_pages, pages_completed, videos_found, videos_saved, errors_count, error_messages, started_at, completed_at, created_at, updated_at FROM scraper_tab_jobs`,
			`DROP TABLE scraper_tab_jobs`,
			`ALTER TABLE scraper_tab_jobs_new RENAME TO scraper_tab_jobs`,
		}
		for i, step := range migrationSteps {
			if _, err := DB.Exec(step); err != nil {
				errMsg := err.Error()
				if !strings.Contains(errMsg, "no such table") && !strings.Contains(errMsg, "already exists") {
					return fmt.Errorf("migration 37 step %d failed: %w", i+1, err)
				}
			}
		}
		for _, idx := range scraperTabJobsIndexes {
			DB.Exec(idx)
		}
	}

	// Migration 38–40: content_type columns
	DB.Exec(`ALTER TABLE scraper_tabs ADD COLUMN content_type TEXT DEFAULT 'regular'`)
	DB.Exec(`ALTER TABLE scraped_videos ADD COLUMN content_type TEXT DEFAULT 'regular'`)
	DB.Exec(`ALTER TABLE videos ADD COLUMN content_type TEXT DEFAULT 'regular'`)

	// Migration 41: Video Lists System
	migration41VideoLists := []string{
		`CREATE TABLE IF NOT EXISTS video_lists (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			color TEXT DEFAULT '#007bff',
			icon TEXT DEFAULT 'list',
			is_default BOOLEAN DEFAULT 0,
			position INTEGER DEFAULT 0,
			video_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS video_list_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			list_id INTEGER NOT NULL,
			video_id INTEGER,
			scraped_video_id INTEGER,
			position INTEGER DEFAULT 0,
			notes TEXT,
			added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (list_id) REFERENCES video_lists(id) ON DELETE CASCADE,
			FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE,
			FOREIGN KEY (scraped_video_id) REFERENCES scraped_videos(id) ON DELETE CASCADE,
			UNIQUE(list_id, video_id),
			UNIQUE(list_id, scraped_video_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_video_lists_name ON video_lists(name)`,
		`CREATE INDEX IF NOT EXISTS idx_video_lists_position ON video_lists(position)`,
		`CREATE INDEX IF NOT EXISTS idx_video_list_items_list ON video_list_items(list_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_list_items_video ON video_list_items(video_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_list_items_scraped ON video_list_items(scraped_video_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_list_items_position ON video_list_items(position)`,
		`INSERT OR IGNORE INTO video_lists (name, description, icon, is_default, position)
		 VALUES ('Edit List', 'Default list for videos queued for editing', 'edit', 1, 0)`,
	}
	for _, migration := range migration41VideoLists {
		if _, err := DB.Exec(migration); err != nil {
			errMsg := err.Error()
			if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "UNIQUE constraint failed") {
				return fmt.Errorf("migration 41 (video lists) failed: %w", err)
			}
		}
	}
	DB.Exec(`
		INSERT OR IGNORE INTO video_list_items (list_id, video_id, added_at)
		SELECT (SELECT id FROM video_lists WHERE is_default = 1 LIMIT 1), id, CURRENT_TIMESTAMP
		FROM videos WHERE in_edit_list = 1
	`)
	DB.Exec(`UPDATE video_lists SET video_count = (SELECT COUNT(*) FROM video_list_items WHERE list_id = video_lists.id)`)

	// Migration 42: Auto-Update System & Dashboard
	migration42AutoUpdate := []string{
		`CREATE TABLE IF NOT EXISTS scraper_auto_update_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scraper_type TEXT UNIQUE NOT NULL,
			enabled BOOLEAN DEFAULT 0,
			interval_minutes INTEGER DEFAULT 60,
			max_pages_to_check INTEGER DEFAULT 3,
			last_checked_at DATETIME,
			last_found_count INTEGER DEFAULT 0,
			total_found_count INTEGER DEFAULT 0,
			check_count INTEGER DEFAULT 0,
			config_json TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_auto_update_enabled ON scraper_auto_update_configs(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_auto_update_last_checked ON scraper_auto_update_configs(last_checked_at)`,
		`CREATE TABLE IF NOT EXISTS scraper_checkpoints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scraper_type TEXT NOT NULL,
			checkpoint_type TEXT NOT NULL,
			checkpoint_value TEXT NOT NULL,
			target_id TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_checkpoint_unique ON scraper_checkpoints(scraper_type, checkpoint_type, target_id)`,
		`CREATE INDEX IF NOT EXISTS idx_checkpoint_type ON scraper_checkpoints(scraper_type)`,
		`CREATE TABLE IF NOT EXISTS dashboard_feed_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_type TEXT NOT NULL,
			title TEXT NOT NULL,
			message TEXT NOT NULL,
			icon TEXT DEFAULT 'bell',
			color TEXT DEFAULT 'primary',
			scraper_type TEXT,
			metadata TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_feed_created ON dashboard_feed_items(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_feed_type ON dashboard_feed_items(item_type)`,
		`ALTER TABLE scraped_threads ADD COLUMN is_watched BOOLEAN DEFAULT 0`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json) VALUES ('simpcity', 0, 60, 3, '{}')`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json) VALUES ('pornhub', 0, 120, 3, '{"keywords": []}')`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json) VALUES ('rule34video', 0, 120, 3, '{}')`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json) VALUES ('luxuretv', 0, 180, 3, '{"categories": []}')`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json) VALUES ('sexuria', 0, 180, 3, '{"categories": []}')`,
	}
	for _, migration := range migration42AutoUpdate {
		if _, err := DB.Exec(migration); err != nil {
			errMsg := err.Error()
			if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "duplicate column name") && !strings.Contains(errMsg, "UNIQUE constraint failed") {
				return fmt.Errorf("migration 42 (auto-update) failed: %w", err)
			}
		}
	}
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_threads_watched ON scraped_threads(is_watched)`)

	// Migration 43: Sexuria Download Links Table
	migration43SexuriaLinks := []string{
		`CREATE TABLE IF NOT EXISTS sexuria_download_links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			video_id INTEGER NOT NULL,
			source TEXT NOT NULL DEFAULT 'sexuria',
			provider TEXT NOT NULL,
			url TEXT NOT NULL,
			original_url TEXT,
			filename TEXT,
			file_size TEXT,
			file_type TEXT,
			status TEXT DEFAULT 'unknown',
			last_checked_at DATETIME,
			last_status_code INTEGER,
			metadata TEXT DEFAULT '{}',
			discovered_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (video_id) REFERENCES scraped_videos(id) ON DELETE CASCADE,
			UNIQUE(url, video_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sexuria_links_video ON sexuria_download_links(video_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sexuria_links_provider ON sexuria_download_links(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_sexuria_links_status ON sexuria_download_links(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sexuria_links_last_checked ON sexuria_download_links(last_checked_at)`,
		`ALTER TABLE scraped_videos ADD COLUMN links_extracted BOOLEAN DEFAULT 0`,
		`ALTER TABLE scraped_videos ADD COLUMN links_last_extracted_at DATETIME`,
	}
	for _, migration := range migration43SexuriaLinks {
		if _, err := DB.Exec(migration); err != nil {
			errMsg := err.Error()
			if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "duplicate column name") && !strings.Contains(errMsg, "UNIQUE constraint failed") {
				return fmt.Errorf("migration 43 (sexuria links) failed: %w", err)
			}
		}
	}

	// Migration 44: JDownloader 2 Integration
	migration44JDownloader := []string{
		`CREATE TABLE IF NOT EXISTS jdownloader_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			connection_type TEXT NOT NULL DEFAULT 'local',
			local_host TEXT DEFAULT '192.168.1.10',
			local_port INTEGER DEFAULT 3128,
			local_password TEXT DEFAULT '',
			local_use_auth BOOLEAN DEFAULT 0,
			cloud_email TEXT DEFAULT '',
			cloud_password TEXT DEFAULT '',
			cloud_device_id TEXT DEFAULT '',
			cloud_device_name TEXT DEFAULT '',
			default_package_name TEXT DEFAULT '',
			auto_start_downloads BOOLEAN DEFAULT 0,
			enabled BOOLEAN DEFAULT 1,
			last_connected_at DATETIME,
			connection_status TEXT DEFAULT 'disconnected',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS jdownloader_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			video_id INTEGER,
			link_url TEXT NOT NULL,
			package_name TEXT,
			status TEXT DEFAULT 'queued',
			error_message TEXT,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE SET NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jd_history_video ON jdownloader_history(video_id)`,
		`CREATE INDEX IF NOT EXISTS idx_jd_history_status ON jdownloader_history(status)`,
		`CREATE INDEX IF NOT EXISTS idx_jd_history_sent ON jdownloader_history(sent_at DESC)`,
	}
	for _, migration := range migration44JDownloader {
		if _, err := DB.Exec(migration); err != nil {
			errMsg := err.Error()
			if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "duplicate column name") {
				return fmt.Errorf("migration 44 (jdownloader) failed: %w", err)
			}
		}
	}
	DB.Exec(`INSERT OR IGNORE INTO jdownloader_config (id, connection_type) VALUES (1, 'local')`)
	DB.Exec(`UPDATE jdownloader_config SET local_port = 3128 WHERE local_port = 3129`)

	// Migration 45: AI Config table for OpenRouter integration
	if _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS ai_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		lm_studio_url TEXT DEFAULT 'http://localhost:1234',
		model_name TEXT DEFAULT '',
		temperature REAL DEFAULT 0.7,
		max_tokens INTEGER DEFAULT 2000,
		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "already exists") {
			return fmt.Errorf("migration 45 (ai_config) failed: %w", err)
		}
	}
	DB.Exec(`INSERT OR IGNORE INTO ai_config (id, lm_studio_url) VALUES (1, 'http://localhost:1234')`)

	// Migration 46: Backfill Sexuria category from metadata JSON
	DB.Exec(`
		UPDATE scraped_videos
		SET category = json_extract(metadata, '$.subcategory')
		WHERE source LIKE 'sexuria%'
		AND (category IS NULL OR category = '')
		AND metadata IS NOT NULL
		AND json_extract(metadata, '$.subcategory') IS NOT NULL
		AND json_extract(metadata, '$.subcategory') != ''
	`)

	// Migration 47: Fix stuck auto_update_check tasks
	DB.Exec(`
		UPDATE activity_logs
		SET status = 'completed', completed_at = datetime('now')
		WHERE task_type = 'auto_update_check'
		AND status = 'running'
		AND progress = 100
	`)

	// Migration 48: Performance indexes and startup cleanup
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_console_logs_level_created ON console_logs(level, created_at DESC)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_activity_logs_status_created ON activity_logs(status, created_at DESC)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_dashboard_feed_type_created ON dashboard_feed_items(item_type, created_at DESC)`)
	DB.Exec(`DELETE FROM activity_logs WHERE status IN ('completed', 'failed') AND created_at < datetime('now', '-14 days')`)
	DB.Exec(`DELETE FROM console_logs WHERE created_at < datetime('now', '-7 days')`)

	// Migration 19 (conditional): Update tags table unique constraint
	var hasTagsNew int
	DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tags_new'").Scan(&hasTagsNew)
	var needsTagsMigration bool
	rows, err := DB.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name='tags'")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sqlStr string
			rows.Scan(&sqlStr)
			if !strings.Contains(sqlStr, "UNIQUE(name, category)") && !strings.Contains(sqlStr, "UNIQUE (name, category)") {
				needsTagsMigration = true
			}
		}
	}
	if needsTagsMigration {
		log.Println("Migration 19: Updating tags table unique constraint...")
		for _, step := range []string{
			`CREATE TABLE IF NOT EXISTS tags_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				color TEXT,
				icon TEXT,
				category TEXT DEFAULT 'regular',
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(name, category)
			)`,
			`INSERT INTO tags_new (id, name, color, icon, category, created_at, updated_at) SELECT id, name, color, icon, category, created_at, updated_at FROM tags`,
			`DROP TABLE tags`,
			`ALTER TABLE tags_new RENAME TO tags`,
		} {
			if _, err := DB.Exec(step); err != nil {
				if !isColumnExistsError(err) {
					return fmt.Errorf("migration 19 (tags unique constraint) failed: %w", err)
				}
			}
		}
	} else {
		DB.Exec("DROP TABLE IF EXISTS tags_new")
	}

	// Migration 21 (conditional): Remove scene_count column from performers
	var hasSceneCount int
	if err := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('performers') WHERE name='scene_count'").Scan(&hasSceneCount); err == nil && hasSceneCount > 0 {
		log.Println("Migration 21: Removing scene_count column from performers table...")
		for _, step := range []string{
			`CREATE TABLE IF NOT EXISTS performers_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				preview_path TEXT,
				thumbnail_path TEXT DEFAULT '',
				folder_path TEXT,
				video_count INTEGER DEFAULT 0,
				category TEXT DEFAULT 'regular' CHECK(category IN ('regular', 'zoo', '3d')),
				metadata TEXT DEFAULT '{}',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)`,
			`INSERT INTO performers_new (id, name, preview_path, thumbnail_path, folder_path, video_count, category, metadata, created_at, updated_at)
			 SELECT id, name, preview_path, COALESCE(thumbnail_path, ''), folder_path, video_count, category, metadata, created_at, updated_at FROM performers`,
			`DROP TABLE performers`,
			`ALTER TABLE performers_new RENAME TO performers`,
			`CREATE INDEX IF NOT EXISTS idx_performers_name ON performers(name)`,
			`CREATE INDEX IF NOT EXISTS idx_performers_category ON performers(category)`,
		} {
			if _, err := DB.Exec(step); err != nil {
				if !isColumnExistsError(err) {
					return fmt.Errorf("migration 21 (remove scene_count) failed: %w", err)
				}
			}
		}
	} else {
		DB.Exec("DROP TABLE IF EXISTS performers_new")
	}

	// Migration 49: Add is_read column to scraped_posts
	DB.Exec(`ALTER TABLE scraped_posts ADD COLUMN is_read BOOLEAN DEFAULT 0`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scraped_posts_thread_read ON scraped_posts(thread_id, is_read)`)

	// Migration 50: Performance indexes for scraped_videos
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scraped_videos_source_discovered ON scraped_videos(source, discovered_at DESC)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scraped_videos_view_count ON scraped_videos(view_count DESC)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scraped_videos_rating ON scraped_videos(rating DESC)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scraped_videos_duration ON scraped_videos(duration DESC)`)

	// Migration 51: Persistent AI chat history
	DB.Exec(`CREATE TABLE IF NOT EXISTS ai_chat_messages (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		role       TEXT NOT NULL,
		content    TEXT NOT NULL,
		source     TEXT NOT NULL DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_created ON ai_chat_messages(created_at DESC)`)

	// Migration 52: Track missing video files
	DB.Exec(`ALTER TABLE videos ADD COLUMN is_missing BOOLEAN DEFAULT 0`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_videos_is_missing ON videos(is_missing) WHERE is_missing = 1`)

	// Migration 53: Edited↔Backup video links
	DB.Exec(`CREATE TABLE IF NOT EXISTS video_links (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		edited_video_id INTEGER NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
		backup_video_id INTEGER NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
		confidence      REAL    NOT NULL DEFAULT 0.0,
		status          TEXT    NOT NULL DEFAULT 'suggested',
		match_reasons   TEXT    NOT NULL DEFAULT '[]',
		created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(edited_video_id, backup_video_id)
	)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_video_links_edited  ON video_links(edited_video_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_video_links_backup  ON video_links(backup_video_id)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_video_links_status  ON video_links(status)`)

	// Migration 54: Switch AI provider to OpenRouter
	DB.Exec(`ALTER TABLE ai_config ADD COLUMN api_key TEXT DEFAULT ''`)
	DB.Exec(`ALTER TABLE ai_config ADD COLUMN api_url TEXT DEFAULT 'https://openrouter.ai/api'`)
	DB.Exec(`UPDATE ai_config SET api_url = lm_studio_url WHERE lm_studio_url != 'http://localhost:1234' AND lm_studio_url != ''`)
	DB.Exec(`UPDATE ai_config SET api_url = 'https://openrouter.ai/api', model_name = CASE WHEN model_name = '' THEN 'mistralai/mistral-nemo' ELSE model_name END WHERE id = 1 AND (api_url IS NULL OR api_url = '')`)

	// Migration 55: Watched status tracking
	DB.Exec(`ALTER TABLE videos ADD COLUMN watched BOOLEAN NOT NULL DEFAULT 0`)
	DB.Exec(`ALTER TABLE videos ADD COLUMN last_watched_at DATETIME`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_videos_watched ON videos(watched)`)

	// Migration 56: FTS5 full-text search for memories
	var hasFTS int
	DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='memories_fts'`).Scan(&hasFTS)
	if hasFTS == 0 {
		log.Println("Migration 56: Creating FTS5 full-text search index for memories...")
		for _, step := range []string{
			`CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(key, value, category, content='memories', content_rowid='id')`,
			`INSERT INTO memories_fts(rowid, key, value, category) SELECT id, key, value, category FROM memories`,
			`CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
				INSERT INTO memories_fts(rowid, key, value, category) VALUES (new.id, new.key, new.value, new.category);
			END`,
			`CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
				INSERT INTO memories_fts(memories_fts, rowid, key, value, category) VALUES ('delete', old.id, old.key, old.value, old.category);
			END`,
			`CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
				INSERT INTO memories_fts(memories_fts, rowid, key, value, category) VALUES ('delete', old.id, old.key, old.value, old.category);
				INSERT INTO memories_fts(rowid, key, value, category) VALUES (new.id, new.key, new.value, new.category);
			END`,
		} {
			if _, err := DB.Exec(step); err != nil {
				log.Printf("Migration 56 (FTS5 memories): %v", err)
			}
		}
	}

	// Migration 57: Companion improvement plans
	DB.Exec(`CREATE TABLE IF NOT EXISTS companion_plans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		library_id INTEGER NOT NULL,
		plan_name TEXT NOT NULL,
		plan_data TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL DEFAULT 'draft',
		total_actions INTEGER DEFAULT 0,
		completed_actions INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		executed_at DATETIME,
		FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE CASCADE
	)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_companion_plans_status ON companion_plans(status)`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_companion_plans_library ON companion_plans(library_id)`)

	// Migration 58: Fix Simpcity domain from .is to .cr
	log.Println("Migration 58: Fixing Simpcity URLs from .is to .cr...")
	// Fix forum URLs in auto-update configs (JSON field)
	if _, err := DB.Exec(`UPDATE scraper_auto_update_configs SET config_json = REPLACE(config_json, 'simpcity.is', 'simpcity.cr') WHERE scraper_type = 'simpcity' AND config_json LIKE '%simpcity.is%'`); err != nil {
		log.Printf("Migration 58 (configs): %v", err)
	}
	// Fix thread URLs in scraped_threads
	if _, err := DB.Exec(`UPDATE scraped_threads SET url = REPLACE(url, 'simpcity.is', 'simpcity.cr') WHERE source = 'simpcity' AND url LIKE '%simpcity.is%'`); err != nil {
		log.Printf("Migration 58 (threads): %v", err)
	}
	// Fix download links that reference simpcity.is
	if _, err := DB.Exec(`UPDATE scraped_download_links SET url = REPLACE(url, 'simpcity.is', 'simpcity.cr') WHERE provider = 'simpcity' AND url LIKE '%simpcity.is%'`); err != nil {
		log.Printf("Migration 58 (links): %v", err)
	}
	log.Println("Migration 58: Simpcity URLs fixed")

	return nil
}

// isColumnExistsError returns true when the error indicates a column or table already exists.
func isColumnExistsError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "duplicate column name") ||
		strings.Contains(errMsg, "already exists")
}
