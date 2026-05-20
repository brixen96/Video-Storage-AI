package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
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

// Initialize sets up the database connection, creates tables, and runs migrations.
func Initialize(cfg *config.Config) error {
	dbPath = cfg.Database.Path

	dbDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Repair orphan indexes before the main open — can appear after DB copies
	// or improper schema changes and cause a fatal "malformed database schema" error.
	repairOrphanIndexes(cfg.Database.Path)

	// WAL mode for better concurrent read performance
	connStr := cfg.Database.Path + "?_journal_mode=WAL&_busy_timeout=5000&_synchronous=FULL&_cache_size=10000&_foreign_keys=ON"
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db

	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	backfillScrapedVideoContentType(DB)

	log.Println("Database connection established")
	return nil
}

// GetDB returns the global database instance.
func GetDB() *sql.DB {
	return DB
}

// GetDBPath returns the database file path.
func GetDBPath() string {
	return dbPath
}

// Close closes the database connection.
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// HealthCheck pings the database to verify the connection is alive.
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Ping()
}

// repairOrphanIndexes removes index/trigger entries from sqlite_master whose table no
// longer exists. This can cause a fatal "malformed database schema - orphan index" on startup.
func repairOrphanIndexes(filePath string) {
	db, err := sql.Open("sqlite3", filePath+"?_foreign_keys=OFF")
	if err != nil {
		return
	}
	defer db.Close()

	if _, err = db.Exec("PRAGMA writable_schema = ON"); err != nil {
		return
	}
	res, err := db.Exec(`
		DELETE FROM sqlite_master
		WHERE type IN ('index', 'trigger')
		  AND tbl_name NOT IN (SELECT name FROM sqlite_master WHERE type = 'table')
	`)
	db.Exec("PRAGMA writable_schema = OFF")
	if err != nil {
		log.Printf("[DB] Schema repair query failed: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("[DB] Repaired %d orphan schema entries", n)
	}
}

// createTables creates the initial database schema (tables + indexes).
// All subsequent schema changes go through runMigrations in database_migrations.go.
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
		metadata TEXT,
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
		metadata TEXT,
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
		metadata TEXT,
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
		task_type TEXT NOT NULL,
		status TEXT NOT NULL,
		message TEXT,
		progress INTEGER DEFAULT 0,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		details TEXT
	);

	-- Indexes for better performance
	CREATE INDEX IF NOT EXISTS idx_libraries_name ON libraries(name);
	CREATE INDEX IF NOT EXISTS idx_videos_library_id ON videos(library_id);
	CREATE INDEX IF NOT EXISTS idx_videos_file_path ON videos(file_path);
	CREATE INDEX IF NOT EXISTS idx_videos_created_at ON videos(created_at);
	CREATE INDEX IF NOT EXISTS idx_videos_library_created ON videos(library_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_videos_duration ON videos(duration);
	CREATE INDEX IF NOT EXISTS idx_videos_file_size ON videos(file_size);
	CREATE INDEX IF NOT EXISTS idx_videos_play_count ON videos(play_count DESC);
	CREATE INDEX IF NOT EXISTS idx_videos_marks ON videos(not_interested, in_edit_list);
	CREATE INDEX IF NOT EXISTS idx_videos_rating ON videos(rating DESC);
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

// backfillScrapedVideoContentType sets content_type on scraped_videos by matching
// their source domain against the scraper_tabs table. Also ensures LuxureTV is zoo.
func backfillScrapedVideoContentType(db *sql.DB) {
	db.Exec(`UPDATE scraper_tabs SET content_type = 'zoo' WHERE site_type = 'luxuretv' AND (content_type = '' OR content_type = 'regular' OR content_type IS NULL)`)

	tabRows, err := db.Query(`SELECT base_url, content_type FROM scraper_tabs WHERE is_active = 1 AND content_type != '' AND content_type != 'regular'`)
	if err != nil {
		log.Printf("[DB] Failed to query scraper_tabs for backfill: %v", err)
		return
	}
	tabMap := make(map[string]string)
	for tabRows.Next() {
		var baseURL, ct string
		if tabRows.Scan(&baseURL, &ct) == nil {
			if parsed, parseErr := url.Parse(baseURL); parseErr == nil {
				host := strings.TrimPrefix(parsed.Hostname(), "www.")
				if host != "" {
					tabMap[host] = ct
				}
			}
		}
	}
	tabRows.Close()

	if len(tabMap) == 0 {
		return
	}

	videoRows, err := db.Query(`SELECT id, source FROM scraped_videos WHERE (content_type IS NULL OR content_type = '' OR content_type = 'regular') AND source != ''`)
	if err != nil {
		log.Printf("[DB] Failed to query scraped_videos for backfill: %v", err)
		return
	}
	type videoUpdate struct {
		id int64
		ct string
	}
	var updates []videoUpdate
	for videoRows.Next() {
		var id int64
		var source string
		if videoRows.Scan(&id, &source) == nil {
			cleanSource := strings.TrimPrefix(source, "www.")
			if ct, ok := tabMap[cleanSource]; ok {
				updates = append(updates, videoUpdate{id, ct})
			}
		}
	}
	videoRows.Close()

	updated := 0
	for _, u := range updates {
		if _, execErr := db.Exec(`UPDATE scraped_videos SET content_type = ? WHERE id = ?`, u.ct, u.id); execErr == nil {
			updated++
		}
	}
	if updated > 0 {
		log.Printf("[DB] Backfilled content_type on %d scraped_videos from scraper_tabs", updated)
	}
}
