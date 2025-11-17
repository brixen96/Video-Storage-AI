package services

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
)

type DatabaseService struct {
	db *sql.DB
}

func NewDatabaseService() *DatabaseService {
	return &DatabaseService{
		db: database.DB,
	}
}

// DatabaseStats holds statistics about the database
type DatabaseStats struct {
	Size           int64  `json:"size"`
	PageCount      int64  `json:"page_count"`
	PageSize       int64  `json:"page_size"`
	FreePages      int64  `json:"free_pages"`
	VideoCount     int64  `json:"video_count"`
	PerformerCount int64  `json:"performer_count"`
	StudioCount    int64  `json:"studio_count"`
	TagCount       int64  `json:"tag_count"`
	GroupCount     int64  `json:"group_count"`
}

// GetStats retrieves database statistics
func (s *DatabaseService) GetStats() (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Get file size
	dbPath := "./video-storage.db"
	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database file info: %w", err)
	}
	stats.Size = fileInfo.Size()

	// Get SQLite page info
	err = s.db.QueryRow("PRAGMA page_count").Scan(&stats.PageCount)
	if err != nil {
		log.Printf("Failed to get page count: %v", err)
	}

	err = s.db.QueryRow("PRAGMA page_size").Scan(&stats.PageSize)
	if err != nil {
		log.Printf("Failed to get page size: %v", err)
	}

	err = s.db.QueryRow("PRAGMA freelist_count").Scan(&stats.FreePages)
	if err != nil {
		log.Printf("Failed to get free pages: %v", err)
	}

	// Get record counts
	err = s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&stats.VideoCount)
	if err != nil {
		log.Printf("Failed to count videos: %v", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM performers").Scan(&stats.PerformerCount)
	if err != nil {
		log.Printf("Failed to count performers: %v", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM studios").Scan(&stats.StudioCount)
	if err != nil {
		log.Printf("Failed to count studios: %v", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&stats.TagCount)
	if err != nil {
		log.Printf("Failed to count tags: %v", err)
	}

	err = s.db.QueryRow("SELECT COUNT(*) FROM groups").Scan(&stats.GroupCount)
	if err != nil {
		log.Printf("Failed to count groups: %v", err)
	}

	return stats, nil
}

// Optimize runs VACUUM to optimize the database
func (s *DatabaseService) Optimize() error {
	log.Println("Starting database optimization (VACUUM)...")

	// VACUUM reclaims space from deleted records and defragments the database
	_, err := s.db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("failed to optimize database: %w", err)
	}

	// Also run ANALYZE to update query optimizer statistics
	_, err = s.db.Exec("ANALYZE")
	if err != nil {
		log.Printf("Warning: ANALYZE failed: %v", err)
	}

	log.Println("Database optimization completed successfully")
	return nil
}

// BackupResult holds information about a backup operation
type BackupResult struct {
	BackupPath string    `json:"backup_path"`
	Size       int64     `json:"size"`
	Timestamp  time.Time `json:"timestamp"`
}

// Backup creates a backup of the database
func (s *DatabaseService) Backup() (*BackupResult, error) {
	log.Println("Starting database backup...")

	// Create backups directory if it doesn't exist
	backupDir := "./backups"
	err := os.MkdirAll(backupDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupFilename := fmt.Sprintf("video-storage_%s.db", timestamp)
	backupPath := filepath.Join(backupDir, backupFilename)

	// Close current connections and perform backup
	// SQLite doesn't support online backup easily, so we'll use file copy
	dbPath := "./video-storage.db"

	// Open source file
	sourceFile, err := os.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer destFile.Close()

	// Copy the file
	written, err := io.Copy(destFile, sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to copy database: %w", err)
	}

	// Sync to ensure data is written
	err = destFile.Sync()
	if err != nil {
		return nil, fmt.Errorf("failed to sync backup file: %w", err)
	}

	log.Printf("Database backup created: %s (%d bytes)", backupPath, written)

	result := &BackupResult{
		BackupPath: backupPath,
		Size:       written,
		Timestamp:  time.Now(),
	}

	return result, nil
}

// RestoreFromBackup restores the database from a backup file
func (s *DatabaseService) RestoreFromBackup(backupPath string) error {
	log.Printf("Starting database restore from: %s", backupPath)

	// Verify backup file exists
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}
	if backupInfo.IsDir() {
		return fmt.Errorf("backup path is a directory, not a file")
	}

	// Create a safety backup of current database first
	_, err = s.Backup()
	if err != nil {
		log.Printf("Warning: Failed to create pre-restore backup: %v", err)
	}

	dbPath := "./video-storage.db"

	// Note: Database restore requires restarting the application to properly close
	// and reopen the database connection. For now, we'll perform a file copy
	// but the user will need to restart the application.

	// Verify the backup file can be opened
	backupFile, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer backupFile.Close()

	// Create a temporary restore file
	tempPath := dbPath + ".restore_temp"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temporary restore file: %w", err)
	}
	defer tempFile.Close()

	// Copy backup to temp file
	_, err = io.Copy(tempFile, backupFile)
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to copy backup: %w", err)
	}

	err = tempFile.Sync()
	if err != nil {
		log.Printf("Warning: Failed to sync restore file: %v", err)
	}

	tempFile.Close()
	backupFile.Close()

	// Close database connection
	err = database.DB.Close()
	if err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	// Replace database file with restore file
	err = os.Rename(tempPath, dbPath)
	if err != nil {
		return fmt.Errorf("failed to replace database file: %w", err)
	}

	log.Println("Database restore completed. Please restart the application for changes to take effect.")
	return fmt.Errorf("database restored successfully - please restart the application")

}

// ListBackups returns a list of available backup files
func (s *DatabaseService) ListBackups() ([]BackupResult, error) {
	backupDir := "./backups"

	// Check if backups directory exists
	_, err := os.Stat(backupDir)
	if os.IsNotExist(err) {
		return []BackupResult{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to access backups directory: %w", err)
	}

	// Read directory
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backups directory: %w", err)
	}

	var backups []BackupResult
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".db" {
			continue
		}

		fullPath := filepath.Join(backupDir, file.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			log.Printf("Warning: Failed to stat backup file %s: %v", file.Name(), err)
			continue
		}

		backups = append(backups, BackupResult{
			BackupPath: fullPath,
			Size:       info.Size(),
			Timestamp:  info.ModTime(),
		})
	}

	return backups, nil
}
