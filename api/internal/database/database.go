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
	dbPath := cfg.Database.Path + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=10000&_foreign_keys=ON"
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
		`ALTER TABLE performers ADD COLUMN category TEXT DEFAULT 'regular'`,
		`UPDATE performers SET category = 'zoo' WHERE zoo = 1`,
		`UPDATE performers SET category = 'regular' WHERE zoo = 0 OR zoo IS NULL`,
		// Migration 16: Add category to studios
		`ALTER TABLE studios ADD COLUMN category TEXT DEFAULT 'regular'`,
		// Migration 17: Add category to groups
		`ALTER TABLE groups ADD COLUMN category TEXT DEFAULT 'regular'`,
		// Migration 18: Add indexes for category fields
		`CREATE INDEX IF NOT EXISTS idx_performers_category ON performers(category)`,
		`CREATE INDEX IF NOT EXISTS idx_studios_category ON studios(category)`,
		`CREATE INDEX IF NOT EXISTS idx_groups_category ON groups(category)`,
		// Migration 19: Update tags table to allow duplicate names across categories
		// Step 1: Create new tags table with composite unique constraint
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
		// Step 2: Copy data from old table
		`INSERT INTO tags_new (id, name, color, icon, category, created_at, updated_at)
		 SELECT id, name, color, icon, category, created_at, updated_at FROM tags`,
		// Step 3: Drop old table
		`DROP TABLE tags`,
		// Step 4: Rename new table
		`ALTER TABLE tags_new RENAME TO tags`,
		// Step 5: Recreate index
		`CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_category ON tags(category)`,
		// Migration 20: Add video_count to performers and populate from existing data
		`ALTER TABLE performers ADD COLUMN video_count INTEGER DEFAULT 0`,
		// Update video_count based on actual video relationships
		`UPDATE performers SET video_count = (
			SELECT COUNT(*) FROM video_performers WHERE performer_id = performers.id
		)`,
		// Migration 21: Remove scene_count column (deprecated, use video_count)
		// SQLite doesn't support DROP COLUMN, so we recreate the table
		`CREATE TABLE IF NOT EXISTS performers_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			preview_path TEXT,
			folder_path TEXT,
			video_count INTEGER DEFAULT 0,
			category TEXT DEFAULT 'regular' CHECK(category IN ('regular', 'zoo', '3d')),
			metadata TEXT DEFAULT '{}',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO performers_new (id, name, preview_path, folder_path, video_count, category, metadata, created_at, updated_at)
		 SELECT id, name, preview_path, folder_path, video_count, category, metadata, created_at, updated_at FROM performers`,
		`DROP TABLE performers`,
		`ALTER TABLE performers_new RENAME TO performers`,
		// Recreate indexes
		`CREATE INDEX IF NOT EXISTS idx_performers_name ON performers(name)`,
		`CREATE INDEX IF NOT EXISTS idx_performers_category ON performers(category)`,
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

	// Migration 32: Notifications System
	migration32 := []string{
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

	for _, migration := range migration32 {
		if _, err := DB.Exec(migration); err != nil {
			if !isColumnExistsError(err) {
				return fmt.Errorf("migration 32 failed: %w", err)
			}
		}
	}

	for _, migration := range migrations {
		if _, err := DB.Exec(migration); err != nil {
			// Ignore error if column already exists
			if !isColumnExistsError(err) {
				return fmt.Errorf("migration failed: %w", err)
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
