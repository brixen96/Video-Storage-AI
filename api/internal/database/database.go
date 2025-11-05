package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

// DB is the global database instance
var DB *sql.DB

// Initialize sets up the database connection and creates tables
func Initialize(cfg *config.Config) error {
	// Ensure the database directory exists
	dbDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxIdleConns(cfg.Database.MaxIdleConn)
	db.SetMaxOpenConns(cfg.Database.MaxOpenConn)
	db.SetConnMaxLifetime(time.Hour)

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

	return nil
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

// runMigrations runs database migrations
func runMigrations() error {
	// Migration 1: Add primary_lib column to libraries table if it doesn't exist
	_, err := DB.Exec(`
		ALTER TABLE libraries ADD COLUMN primary_lib BOOLEAN DEFAULT 0
	`)
	// Ignore error if column already exists
	if err != nil && !isColumnExistsError(err) {
		return fmt.Errorf("failed to add primary_lib column: %w", err)
	}

	// Migration 2: Add not_interested column to videos table
	_, err = DB.Exec(`
		ALTER TABLE videos ADD COLUMN not_interested BOOLEAN DEFAULT 0
	`)
	if err != nil && !isColumnExistsError(err) {
		return fmt.Errorf("failed to add not_interested column: %w", err)
	}

	// Migration 3: Add in_edit_list column to videos table
	_, err = DB.Exec(`
		ALTER TABLE videos ADD COLUMN in_edit_list BOOLEAN DEFAULT 0
	`)
	if err != nil && !isColumnExistsError(err) {
		return fmt.Errorf("failed to add in_edit_list column: %w", err)
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
