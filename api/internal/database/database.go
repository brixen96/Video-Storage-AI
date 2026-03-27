package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

// DB is the global database instance
var DB *sql.DB

// dbPath stores the path to the database file
var dbPath string

// Initialize sets up the database connection and creates tables
func Initialize(cfg *config.Config) error {
	// Store the database path
	dbPath = cfg.Database.Path

	// Ensure the database directory exists
	dbDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection with optimized settings
	// Enable WAL mode for better concurrent read performance
	dbPath := cfg.Database.Path + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=FULL&_cache_size=10000&_foreign_keys=ON"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings optimized for SQLite with WAL mode
	// WAL mode allows multiple concurrent readers with one writer
	db.SetMaxOpenConns(25)  // Allow up to 25 concurrent connections (mostly readers)
	db.SetMaxIdleConns(5)   // Keep 5 idle connections ready for fast reuse
	db.SetConnMaxLifetime(time.Hour) // Recycle connections every hour
	db.SetConnMaxIdleTime(10 * time.Minute) // Close idle connections after 10 minutes

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db

	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database connection established")
	return nil
}

// GetDB returns the database instance
func GetDB() *sql.DB {
	return DB
}

// GetDBPath returns the database file path
func GetDBPath() string {
	return dbPath
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// createTables creates all necessary database tables
func createTables() error {
	schema := `
	-- Libraries table
	CREATE TABLE IF NOT EXISTS libraries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		path TEXT NOT NULL,
		primary_lib BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Videos table
	CREATE TABLE IF NOT EXISTS videos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		library_id INTEGER,
		title TEXT NOT NULL,
		file_path TEXT UNIQUE NOT NULL,
		file_size INTEGER,
		duration REAL,
		codec TEXT,
		resolution TEXT,
		bitrate INTEGER,
		fps REAL,
		thumbnail_path TEXT,
		preview_path TEXT,
		date TEXT,
		rating INTEGER DEFAULT 0,
		description TEXT,
		is_favorite BOOLEAN DEFAULT 0,
		is_pinned BOOLEAN DEFAULT 0,
		not_interested BOOLEAN DEFAULT 0,
		in_edit_list BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_played_at DATETIME,
		play_count INTEGER DEFAULT 0,
		FOREIGN KEY (library_id) REFERENCES libraries(id) ON DELETE SET NULL
	);

	-- Performers table
	CREATE TABLE IF NOT EXISTS performers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		preview_path TEXT,
		folder_path TEXT,
		scene_count INTEGER DEFAULT 0,
		metadata TEXT, -- JSON field for additional metadata
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Studios table
	CREATE TABLE IF NOT EXISTS studios (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		logo_path TEXT,
		description TEXT,
		founded_date DATE,
		country TEXT,
		metadata TEXT, -- JSON field for additional metadata
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Groups table (sub-labels under studios)
	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		studio_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		logo_path TEXT,
		description TEXT,
		metadata TEXT, -- JSON field for additional metadata
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (studio_id) REFERENCES studios(id) ON DELETE CASCADE,
		UNIQUE(studio_id, name)
	);

	-- Tags table
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		color TEXT,
		icon TEXT,
		category TEXT DEFAULT 'regular',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Video-Performer relationship (many-to-many)
	CREATE TABLE IF NOT EXISTS video_performers (
		video_id INTEGER NOT NULL,
		performer_id INTEGER NOT NULL,
		PRIMARY KEY (video_id, performer_id),
		FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE,
		FOREIGN KEY (performer_id) REFERENCES performers(id) ON DELETE CASCADE
	);

	-- Video-Tag relationship (many-to-many)
	CREATE TABLE IF NOT EXISTS video_tags (
		video_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (video_id, tag_id),
		FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);

	-- Performer-Tag relationship (many-to-many) - Master Tags
	CREATE TABLE IF NOT EXISTS performer_tags (
		performer_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (performer_id, tag_id),
		FOREIGN KEY (performer_id) REFERENCES performers(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);

	-- Video-Studio relationship
	CREATE TABLE IF NOT EXISTS video_studios (
		video_id INTEGER NOT NULL,
		studio_id INTEGER NOT NULL,
		PRIMARY KEY (video_id, studio_id),
		FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE,
		FOREIGN KEY (studio_id) REFERENCES studios(id) ON DELETE CASCADE
	);

	-- Video-Group relationship
	CREATE TABLE IF NOT EXISTS video_groups (
		video_id INTEGER NOT NULL,
		group_id INTEGER NOT NULL,
		PRIMARY KEY (video_id, group_id),
		FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE,
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
	);

	-- Activity Monitor logs
	CREATE TABLE IF NOT EXISTS activity_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_type TEXT NOT NULL, -- scanning, indexing, ai_tagging, metadata_fetch, etc.
		status TEXT NOT NULL, -- pending, running, completed, failed
		message TEXT,
		progress INTEGER DEFAULT 0, -- 0-100
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		details TEXT -- JSON field for additional details
	);

	-- Indexes for better performance
	CREATE INDEX IF NOT EXISTS idx_libraries_name ON libraries(name);
	CREATE INDEX IF NOT EXISTS idx_videos_library_id ON videos(library_id);
	CREATE INDEX IF NOT EXISTS idx_videos_file_path ON videos(file_path);
	CREATE INDEX IF NOT EXISTS idx_videos_created_at ON videos(created_at);

	-- Composite indexes for common filtered queries
	CREATE INDEX IF NOT EXISTS idx_videos_library_created ON videos(library_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_videos_duration ON videos(duration);
	CREATE INDEX IF NOT EXISTS idx_videos_file_size ON videos(file_size);
	CREATE INDEX IF NOT EXISTS idx_videos_play_count ON videos(play_count DESC);
	CREATE INDEX IF NOT EXISTS idx_videos_marks ON videos(not_interested, in_edit_list);
	CREATE INDEX IF NOT EXISTS idx_videos_rating ON videos(rating DESC);

	-- Relationship table indexes (both directions for JOIN performance)
	CREATE INDEX IF NOT EXISTS idx_video_performers_video ON video_performers(video_id);
	CREATE INDEX IF NOT EXISTS idx_video_performers_performer ON video_performers(performer_id, video_id);
	CREATE INDEX IF NOT EXISTS idx_video_tags_video ON video_tags(video_id);
	CREATE INDEX IF NOT EXISTS idx_video_tags_tag ON video_tags(tag_id, video_id);
	CREATE INDEX IF NOT EXISTS idx_video_studios_video ON video_studios(video_id);
	CREATE INDEX IF NOT EXISTS idx_video_studios_studio ON video_studios(studio_id, video_id);
	CREATE INDEX IF NOT EXISTS idx_video_groups_video ON video_groups(video_id);
	CREATE INDEX IF NOT EXISTS idx_video_groups_group ON video_groups(group_id, video_id);

	CREATE INDEX IF NOT EXISTS idx_performers_name ON performers(name);
	CREATE INDEX IF NOT EXISTS idx_studios_name ON studios(name);
	CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);
	CREATE INDEX IF NOT EXISTS idx_activity_logs_task_type ON activity_logs(task_type);
	CREATE INDEX IF NOT EXISTS idx_activity_logs_status ON activity_logs(status);
	`

	_, err := DB.Exec(schema)
	return err
}

// HealthCheck performs a database health check
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Ping()
}

// runMigrations executes database migrations
func runMigrations() error {
	migrations := []string{
		// Migration 1: Add primary_lib column to libraries table if it doesn't exist
		`
		ALTER TABLE libraries ADD COLUMN primary_lib BOOLEAN DEFAULT 0
	`,
		// Ignore error if column already exists
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
		// NOTE: The zoo column was removed by migration 21. Migration 4 re-adds it on every restart
		// with default 0, which caused the UPDATE below to reset ALL categories to 'regular'.
		// The UPDATEs are now safe: only migrate rows where category hasn't been set yet (IS NULL).
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
		// Migration 19: Moved to conditional block (see below) to avoid cascade-deleting video_tags/performer_tags on every restart
		`CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(category)`,
		// Migration 20: Add video_count to performers and populate from existing data
		`ALTER TABLE performers ADD COLUMN video_count INTEGER DEFAULT 0`,
		// Update video_count based on actual video relationships
		`UPDATE performers SET video_count = (
			SELECT COUNT(*) FROM video_performers WHERE performer_id = performers.id
		)`,
		// Migration 21: Moved to conditional block (see below) to avoid cascade-deleting performer_tags on every restart
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
		`ALTER TABLE activities ADD COLUMN checkpoint TEXT DEFAULT '{}'`, // JSON checkpoint data
		`ALTER TABLE activity_logs ADD COLUMN is_paused BOOLEAN DEFAULT 0`,
		`ALTER TABLE activity_logs ADD COLUMN paused_at DATETIME`,
		`ALTER TABLE activity_logs ADD COLUMN checkpoint TEXT DEFAULT '{}'`,
		// Migration 27: Performance indexes for pause/resume and AI search
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_is_paused ON activity_logs(is_paused) WHERE is_paused = 1`,
		`CREATE INDEX IF NOT EXISTS idx_activity_logs_status_paused ON activity_logs(status, is_paused)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_posts_plain_text ON scraped_posts(plain_text)`,
		`CREATE INDEX IF NOT EXISTS idx_scraped_threads_title ON scraped_threads(title)`,
		// Migration 28: Additional performance indexes for foreign keys and common queries
		`CREATE INDEX IF NOT EXISTS idx_video_performers_performer ON video_performers(performer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_tags_tag ON video_tags(tag_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_studios_studio ON video_studios(studio_id)`,
		`CREATE INDEX IF NOT EXISTS idx_performer_scraped_threads_thread ON performer_scraped_threads(thread_id)`,
		// Migration 29: Download tracking fields
		`ALTER TABLE scraped_download_links ADD COLUMN download_status TEXT DEFAULT 'pending'`, // pending, downloaded, failed, in_progress
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
	migration32bRating := `ALTER TABLE scraped_videos ADD COLUMN rating INTEGER DEFAULT 0`
	if _, err := DB.Exec(migration32bRating); err != nil {
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

	// Create indexes for scraper_tabs
	scraperTabsIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_scraper_tabs_active ON scraper_tabs(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_tabs_position ON scraper_tabs(position)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_tabs_type ON scraper_tabs(tab_type)`,
	}
	for _, idx := range scraperTabsIndexes {
		if _, err := DB.Exec(idx); err != nil {
			if !isColumnExistsError(err) {
				return fmt.Errorf("migration 34 (scraper tabs indexes) failed: %w", err)
			}
		}
	}

	// Migration 35: Scraper Tab Jobs Table (renamed to avoid conflict with old scraper_jobs)
	migration35ScraperTabJobs := `CREATE TABLE IF NOT EXISTS scraper_tab_jobs (
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
	)`

	if _, err := DB.Exec(migration35ScraperTabJobs); err != nil {
		errMsg := err.Error()
		// Ignore "already exists" errors
		if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "duplicate") {
			return fmt.Errorf("migration 35 (scraper tab jobs table) failed: %w", err)
		}
	}

	// Create indexes for scraper_tab_jobs
	scraperTabJobsIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_scraper_tab_jobs_tab ON scraper_tab_jobs(tab_id)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_tab_jobs_status ON scraper_tab_jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_scraper_tab_jobs_created ON scraper_tab_jobs(created_at DESC)`,
	}
	for _, idx := range scraperTabJobsIndexes {
		if _, err := DB.Exec(idx); err != nil {
			errMsg := err.Error()
			// Ignore "already exists" and "no such column" errors (table might not exist yet)
			if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "no such column") {
				return fmt.Errorf("migration 35 (scraper jobs indexes) failed: %w", err)
			}
		}
	}

	// Migration 36: Add job_type column to scraper_tab_jobs for source-specific jobs (like PornHub)
	// and make tab_id nullable
	migration36JobType := `ALTER TABLE scraper_tab_jobs ADD COLUMN job_type TEXT DEFAULT 'tab'`
	if _, err := DB.Exec(migration36JobType); err != nil {
		if !isColumnExistsError(err) {
			// Ignore if column exists
		}
	}

	// Migration 37: Recreate scraper_tab_jobs with nullable tab_id (SQLite workaround)
	// Check if migration is needed by checking if we can insert with NULL tab_id
	var needsMigration bool
	testQuery := `INSERT INTO scraper_tab_jobs (tab_id, status, url, job_type) VALUES (NULL, 'test', 'test', 'test')`
	if _, err := DB.Exec(testQuery); err != nil {
		needsMigration = true
	} else {
		// Clean up test row
		DB.Exec(`DELETE FROM scraper_tab_jobs WHERE status = 'test' AND url = 'test'`)
	}

	if needsMigration {
		// Recreate the table with nullable tab_id
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
				// Allow some errors during migration (like table doesn't exist)
				if !strings.Contains(errMsg, "no such table") && !strings.Contains(errMsg, "already exists") {
					return fmt.Errorf("migration 37 step %d failed: %w", i+1, err)
				}
			}
		}

		// Recreate indexes after table rename
		for _, idx := range scraperTabJobsIndexes {
			DB.Exec(idx)
		}
	}

	// Migration 38: Add content_type column to scraper_tabs for filtering (Regular/Zoo/3D/etc)
	migration38ContentType := `ALTER TABLE scraper_tabs ADD COLUMN content_type TEXT DEFAULT 'regular'`
	if _, err := DB.Exec(migration38ContentType); err != nil {
		// Ignore if column exists
		if !isColumnExistsError(err) {
			// Allow to continue
		}
	}

	// Migration 39: Add content_type column to scraped_videos
	migration39ContentType := `ALTER TABLE scraped_videos ADD COLUMN content_type TEXT DEFAULT 'regular'`
	if _, err := DB.Exec(migration39ContentType); err != nil {
		// Ignore if column exists
		if !isColumnExistsError(err) {
			// Allow to continue
		}
	}

	// Migration 40: Add content_type column to videos table
	migration40ContentType := `ALTER TABLE videos ADD COLUMN content_type TEXT DEFAULT 'regular'`
	if _, err := DB.Exec(migration40ContentType); err != nil {
		// Ignore if column exists
		if !isColumnExistsError(err) {
			// Allow to continue
		}
	}

	// Migration 41: Video Lists System - Custom named lists for videos
	migration41VideoLists := []string{
		// video_lists table - stores named lists
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
		// video_list_items junction table - links videos to lists
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
		// Indexes for video_lists
		`CREATE INDEX IF NOT EXISTS idx_video_lists_name ON video_lists(name)`,
		`CREATE INDEX IF NOT EXISTS idx_video_lists_position ON video_lists(position)`,
		// Indexes for video_list_items
		`CREATE INDEX IF NOT EXISTS idx_video_list_items_list ON video_list_items(list_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_list_items_video ON video_list_items(video_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_list_items_scraped ON video_list_items(scraped_video_id)`,
		`CREATE INDEX IF NOT EXISTS idx_video_list_items_position ON video_list_items(position)`,
		// Create default "Edit List" to migrate existing in_edit_list videos
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

	// Migrate existing in_edit_list videos to the default Edit List
	migrateEditListSQL := `
		INSERT OR IGNORE INTO video_list_items (list_id, video_id, added_at)
		SELECT
			(SELECT id FROM video_lists WHERE is_default = 1 LIMIT 1),
			id,
			CURRENT_TIMESTAMP
		FROM videos
		WHERE in_edit_list = 1
	`
	if _, err := DB.Exec(migrateEditListSQL); err != nil {
		// Ignore errors - migration might already be done
	}

	// Update video_count for the default list
	updateCountSQL := `
		UPDATE video_lists SET video_count = (
			SELECT COUNT(*) FROM video_list_items WHERE list_id = video_lists.id
		)
	`
	if _, err := DB.Exec(updateCountSQL); err != nil {
		// Ignore errors
	}

	// Migration 42: Auto-Update System & Dashboard
	migration42AutoUpdate := []string{
		// Scraper auto-update configurations
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
		// Index for enabled configs
		`CREATE INDEX IF NOT EXISTS idx_auto_update_enabled ON scraper_auto_update_configs(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_auto_update_last_checked ON scraper_auto_update_configs(last_checked_at)`,

		// Track seen content to detect NEW items only
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

		// Dashboard activity feed cache
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

		// Add is_watched column to scraped_threads for simpcity auto-update
		`ALTER TABLE scraped_threads ADD COLUMN is_watched BOOLEAN DEFAULT 0`,

		// Insert default auto-update configs for each scraper type
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json)
		 VALUES ('simpcity', 0, 60, 3, '{}')`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json)
		 VALUES ('pornhub', 0, 120, 3, '{"keywords": []}')`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json)
		 VALUES ('rule34video', 0, 120, 3, '{}')`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json)
		 VALUES ('luxuretv', 0, 180, 3, '{"categories": []}')`,
		`INSERT OR IGNORE INTO scraper_auto_update_configs (scraper_type, enabled, interval_minutes, max_pages_to_check, config_json)
		 VALUES ('sexuria', 0, 180, 3, '{"categories": []}')`,
	}

	for _, migration := range migration42AutoUpdate {
		if _, err := DB.Exec(migration); err != nil {
			errMsg := err.Error()
			if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "duplicate column name") && !strings.Contains(errMsg, "UNIQUE constraint failed") {
				return fmt.Errorf("migration 42 (auto-update) failed: %w", err)
			}
		}
	}

	// Create index on is_watched if not exists
	if _, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_threads_watched ON scraped_threads(is_watched)`); err != nil {
		// Ignore if already exists
	}

	// Migration 43: Sexuria Download Links Table for link verification
	migration43SexuriaLinks := []string{
		// Sexuria download links table
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
		// Indexes for efficient queries
		`CREATE INDEX IF NOT EXISTS idx_sexuria_links_video ON sexuria_download_links(video_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sexuria_links_provider ON sexuria_download_links(provider)`,
		`CREATE INDEX IF NOT EXISTS idx_sexuria_links_status ON sexuria_download_links(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sexuria_links_last_checked ON sexuria_download_links(last_checked_at)`,
		// Add links_extracted flag to scraped_videos for Sexuria items
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
		// JDownloader configuration table
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
		// JDownloader download history table
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
		// Indexes for JDownloader tables
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

	// Insert default JDownloader config if not exists
	if _, err := DB.Exec(`INSERT OR IGNORE INTO jdownloader_config (id, connection_type) VALUES (1, 'local')`); err != nil {
		// Ignore if already exists
	}

	// Fix JDownloader port from 3129 to 3128 (correct default port)
	DB.Exec(`UPDATE jdownloader_config SET local_port = 3128 WHERE local_port = 3129`)

	// Migration 45: AI Config table for LM Studio integration
	migration45AIConfig := []string{
		`CREATE TABLE IF NOT EXISTS ai_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			lm_studio_url TEXT DEFAULT 'http://localhost:1234',
			model_name TEXT DEFAULT '',
			temperature REAL DEFAULT 0.7,
			max_tokens INTEGER DEFAULT 2000,
			enabled BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, migration := range migration45AIConfig {
		if _, err := DB.Exec(migration); err != nil {
			errMsg := err.Error()
			if !strings.Contains(errMsg, "already exists") && !strings.Contains(errMsg, "duplicate column name") {
				return fmt.Errorf("migration 45 (ai_config) failed: %w", err)
			}
		}
	}

	// Insert default AI config if not exists
	if _, err := DB.Exec(`INSERT OR IGNORE INTO ai_config (id, lm_studio_url) VALUES (1, 'http://localhost:1234')`); err != nil {
		// Ignore if already exists
	}

	// Migration 46: Backfill category column for Sexuria items from metadata JSON
	// The subcategory was previously only stored in metadata JSON, now we populate the category column
	DB.Exec(`
		UPDATE scraped_videos
		SET category = json_extract(metadata, '$.subcategory')
		WHERE source LIKE 'sexuria%'
		AND (category IS NULL OR category = '')
		AND metadata IS NOT NULL
		AND json_extract(metadata, '$.subcategory') IS NOT NULL
		AND json_extract(metadata, '$.subcategory') != ''
	`)

	// Migration 47: Fix stuck auto_update_check tasks at 100% progress but still "running"
	// The scheduler was calling UpdateProgress(100) instead of CompleteTask(), leaving tasks stuck
	DB.Exec(`
		UPDATE activity_logs
		SET status = 'completed', completed_at = datetime('now')
		WHERE task_type = 'auto_update_check'
		AND status = 'running'
		AND progress = 100
	`)

	// Migration 48: Performance indexes and startup cleanup
	// Composite index for console_logs error monitoring queries (level + created_at)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_console_logs_level_created ON console_logs(level, created_at DESC)`)
	// Composite index for activity_logs status + created_at queries
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_activity_logs_status_created ON activity_logs(status, created_at DESC)`)
	// Index for dashboard feed recent items query
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_dashboard_feed_type_created ON dashboard_feed_items(item_type, created_at DESC)`)
	// Startup cleanup: delete completed/failed activity_logs older than 14 days
	DB.Exec(`DELETE FROM activity_logs WHERE status IN ('completed', 'failed') AND created_at < datetime('now', '-14 days')`)
	// Startup cleanup: delete console_logs older than 7 days
	DB.Exec(`DELETE FROM console_logs WHERE created_at < datetime('now', '-7 days')`)

	// Migration 19 (conditional): Update tags table to allow duplicate names across categories.
	// Only run if tags table still has the old UNIQUE(name) constraint (no category in unique).
	// DROP TABLE with foreign_keys=ON triggers implicit DELETE FROM, which cascades to video_tags/performer_tags.
	var hasTagsNew int
	DB.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='tags_new'").Scan(&hasTagsNew)
	// Check if tags table has the old unique constraint (name only, not name+category composite)
	var needsTagsMigration bool
	rows, err := DB.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name='tags'")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sql string
			rows.Scan(&sql)
			// Old schema has UNIQUE on name only; new schema has UNIQUE(name, category)
			if !strings.Contains(sql, "UNIQUE(name, category)") && !strings.Contains(sql, "UNIQUE (name, category)") {
				needsTagsMigration = true
			}
		}
	}
	if needsTagsMigration {
		log.Println("Migration 19: Updating tags table unique constraint...")
		migration19Steps := []string{
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
			`INSERT INTO tags_new (id, name, color, icon, category, created_at, updated_at)
			 SELECT id, name, color, icon, category, created_at, updated_at FROM tags`,
			`DROP TABLE tags`,
			`ALTER TABLE tags_new RENAME TO tags`,
		}
		for _, step := range migration19Steps {
			if _, err := DB.Exec(step); err != nil {
				if !isColumnExistsError(err) {
					return fmt.Errorf("migration 19 (tags unique constraint) failed: %w", err)
				}
			}
		}
	} else {
		DB.Exec("DROP TABLE IF EXISTS tags_new")
	}

	// Migration 21 (conditional): Remove scene_count column by recreating performers table.
	// Only run if scene_count column still exists to avoid cascade-deleting performer_tags on every restart.
	// DROP TABLE with foreign_keys=ON triggers implicit DELETE FROM, which cascades to performer_tags.
	var hasSceneCount int
	if err := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('performers') WHERE name='scene_count'").Scan(&hasSceneCount); err == nil && hasSceneCount > 0 {
		log.Println("Migration 21: Removing scene_count column from performers table...")
		migration21Steps := []string{
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
		}
		for _, step := range migration21Steps {
			if _, err := DB.Exec(step); err != nil {
				if !isColumnExistsError(err) {
					return fmt.Errorf("migration 21 (remove scene_count) failed: %w", err)
				}
			}
		}
	} else {
		// Clean up leftover temp table if it exists
		DB.Exec("DROP TABLE IF EXISTS performers_new")
	}

	// Migration 49: Add is_read column to scraped_posts for read tracking
	DB.Exec(`ALTER TABLE scraped_posts ADD COLUMN is_read BOOLEAN DEFAULT 0`)
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scraped_posts_thread_read ON scraped_posts(thread_id, is_read)`)

	// Migration 50: Performance indexes for scraped_videos sort/filter queries
	// Composite index covers the most common query: WHERE source = ? ORDER BY discovered_at DESC
	DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scraped_videos_source_discovered ON scraped_videos(source, discovered_at DESC)`)
	// Sort indexes for ORDER BY view_count, rating, duration queries
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

	for i, migration := range migrations {
		if _, err := DB.Exec(migration); err != nil {
			errMsg := err.Error()
			// Ignore error if column already exists or if it's a "no such column" error
			// (which can happen with ALTER TABLE or CREATE INDEX on non-existent columns)
			if !isColumnExistsError(err) && !strings.Contains(errMsg, "no such column") {
				return fmt.Errorf("migration %d failed: %w", i+1, err)
			}
		}
	}

	return nil
}

// isColumnExistsError checks if error is due to duplicate column
func isColumnExistsError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "duplicate column name") ||
		strings.Contains(errMsg, "already exists")
}
