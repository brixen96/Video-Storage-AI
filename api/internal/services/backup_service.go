package services

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// BackupService handles database backup and restore operations
type BackupService struct {
	db              *sql.DB
	backupDir       string
	notificationSvc *NotificationService
}

// BackupInfo contains information about a backup
type BackupInfo struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	Type      string    `json:"type"` // manual or automatic
}

// NewBackupService creates a new backup service
func NewBackupService(backupDir string) *BackupService {
	db := database.GetDB()

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Printf("Failed to create backup directory: %v", err)
	}

	return &BackupService{
		db:              db,
		backupDir:       backupDir,
		notificationSvc: NewNotificationService(db),
	}
}

// CreateBackup creates a new database backup
func (s *BackupService) CreateBackup(backupType string) (*BackupInfo, error) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("video-storage-ai_%s_%s.db", backupType, timestamp)
	backupPath := filepath.Join(s.backupDir, filename)

	log.Printf("Creating %s backup: %s", backupType, filename)

	// Get the database file path
	var dbPath string
	err := s.db.QueryRow("PRAGMA database_list").Scan(nil, nil, &dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %w", err)
	}

	// Close any open transactions and checkpoint WAL
	if _, err := s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Printf("Warning: WAL checkpoint failed: %v", err)
	}

	// Copy the database file
	if err := s.copyFile(dbPath, backupPath); err != nil {
		return nil, fmt.Errorf("failed to copy database: %w", err)
	}

	// Get backup file info
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	backup := &BackupInfo{
		Filename:  filename,
		Path:      backupPath,
		Size:      fileInfo.Size(),
		CreatedAt: time.Now(),
		Type:      backupType,
	}

	log.Printf("Backup created successfully: %s (%.2f MB)", filename, float64(backup.Size)/1024/1024)

	// Send notification
	if s.notificationSvc != nil {
		s.notificationSvc.NotifyBackupCompleted(backup)
	}

	return backup, nil
}

// ListBackups returns a list of all available backups
func (s *BackupService) ListBackups() ([]*BackupInfo, error) {
	files, err := os.ReadDir(s.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []*BackupInfo
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".db") {
			continue
		}

		fullPath := filepath.Join(s.backupDir, file.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			log.Printf("Failed to stat backup file %s: %v", file.Name(), err)
			continue
		}

		// Parse backup type from filename
		backupType := "manual"
		if strings.Contains(file.Name(), "_automatic_") {
			backupType = "automatic"
		}

		backups = append(backups, &BackupInfo{
			Filename:  file.Name(),
			Path:      fullPath,
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Type:      backupType,
		})
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// GetBackup retrieves information about a specific backup
func (s *BackupService) GetBackup(filename string) (*BackupInfo, error) {
	backupPath := filepath.Join(s.backupDir, filename)

	info, err := os.Stat(backupPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("backup not found: %s", filename)
		}
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	backupType := "manual"
	if strings.Contains(filename, "_automatic_") {
		backupType = "automatic"
	}

	return &BackupInfo{
		Filename:  filename,
		Path:      backupPath,
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
		Type:      backupType,
	}, nil
}

// RestoreBackup restores the database from a backup
func (s *BackupService) RestoreBackup(filename string) error {
	backupPath := filepath.Join(s.backupDir, filename)

	// Verify backup exists
	if _, err := os.Stat(backupPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("backup not found: %s", filename)
		}
		return fmt.Errorf("failed to access backup file: %w", err)
	}

	log.Printf("Restoring database from backup: %s", filename)

	// Get the current database path
	var dbPath string
	err := s.db.QueryRow("PRAGMA database_list").Scan(nil, nil, &dbPath)
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Close the database connection
	if err := s.db.Close(); err != nil {
		log.Printf("Warning: Failed to close database: %v", err)
	}

	// Create a safety backup of current database before restore
	safetyBackupPath := dbPath + ".pre-restore-" + time.Now().Format("2006-01-02_15-04-05")
	if err := s.copyFile(dbPath, safetyBackupPath); err != nil {
		return fmt.Errorf("failed to create safety backup: %w", err)
	}

	log.Printf("Created safety backup: %s", safetyBackupPath)

	// Restore the backup
	if err := s.copyFile(backupPath, dbPath); err != nil {
		// Attempt to restore from safety backup
		log.Printf("Restore failed, attempting to recover from safety backup...")
		if recoverErr := s.copyFile(safetyBackupPath, dbPath); recoverErr != nil {
			return fmt.Errorf("restore failed and recovery failed: restore error: %w, recovery error: %v", err, recoverErr)
		}
		return fmt.Errorf("restore failed but database recovered: %w", err)
	}

	log.Printf("Database restored successfully from: %s", filename)

	// Note: Database connection will need to be manually reinitialized after restore
	// The application should be restarted after a restore operation

	// Send notification
	if s.notificationSvc != nil {
		s.notificationSvc.NotifyBackupRestored(filename)
	}

	return nil
}

// DeleteBackup deletes a backup file
func (s *BackupService) DeleteBackup(filename string) error {
	backupPath := filepath.Join(s.backupDir, filename)

	// Verify backup exists
	if _, err := os.Stat(backupPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("backup not found: %s", filename)
		}
		return fmt.Errorf("failed to access backup file: %w", err)
	}

	log.Printf("Deleting backup: %s", filename)

	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// CleanupOldBackups removes backups older than the retention period
func (s *BackupService) CleanupOldBackups(retentionDays int, keepMinimum int) (int, error) {
	backups, err := s.ListBackups()
	if err != nil {
		return 0, fmt.Errorf("failed to list backups: %w", err)
	}

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	deletedCount := 0

	// Separate automatic and manual backups
	var automaticBackups []*BackupInfo
	for _, backup := range backups {
		if backup.Type == "automatic" {
			automaticBackups = append(automaticBackups, backup)
		}
	}

	// Only delete automatic backups, keep manual backups
	for i, backup := range automaticBackups {
		// Keep minimum number of backups even if they're old
		if i < keepMinimum {
			continue
		}

		if backup.CreatedAt.Before(cutoffDate) {
			if err := s.DeleteBackup(backup.Filename); err != nil {
				log.Printf("Failed to delete old backup %s: %v", backup.Filename, err)
				continue
			}
			deletedCount++
			log.Printf("Deleted old backup: %s", backup.Filename)
		}
	}

	if deletedCount > 0 {
		log.Printf("Cleaned up %d old backups", deletedCount)
	}

	return deletedCount, nil
}

// GetBackupStats returns statistics about backups
func (s *BackupService) GetBackupStats() (map[string]interface{}, error) {
	backups, err := s.ListBackups()
	if err != nil {
		return nil, err
	}

	var totalSize int64
	var automaticCount, manualCount int
	var oldestBackup, newestBackup time.Time

	for i, backup := range backups {
		totalSize += backup.Size

		if backup.Type == "automatic" {
			automaticCount++
		} else {
			manualCount++
		}

		if i == 0 || backup.CreatedAt.After(newestBackup) {
			newestBackup = backup.CreatedAt
		}
		if i == 0 || backup.CreatedAt.Before(oldestBackup) {
			oldestBackup = backup.CreatedAt
		}
	}

	stats := map[string]interface{}{
		"total_backups":    len(backups),
		"automatic_count":  automaticCount,
		"manual_count":     manualCount,
		"total_size_bytes": totalSize,
		"total_size_mb":    float64(totalSize) / 1024 / 1024,
	}

	if len(backups) > 0 {
		stats["newest_backup"] = newestBackup
		stats["oldest_backup"] = oldestBackup
	}

	return stats, nil
}

// copyFile copies a file from src to dst
func (s *BackupService) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

// NotifyBackupCompleted sends a notification when a backup is completed
func (s *NotificationService) NotifyBackupCompleted(backup *BackupInfo) error {
	req := &models.CreateNotificationRequest{
		Type:        "backup_completed",
		Priority:    "normal",
		Title:       "Database Backup Completed",
		Message:     fmt.Sprintf("Backup created: %s (%.2f MB)", backup.Filename, float64(backup.Size)/1024/1024),
		Category:    "system",
		ActionURL:   "/settings",
		ActionLabel: "View Backups",
		Metadata: map[string]interface{}{
			"filename": backup.Filename,
			"size":     backup.Size,
			"type":     backup.Type,
		},
	}

	_, err := s.Create(req)
	return err
}

// NotifyBackupRestored sends a notification when a backup is restored
func (s *NotificationService) NotifyBackupRestored(filename string) error {
	req := &models.CreateNotificationRequest{
		Type:        "backup_restored",
		Priority:    "high",
		Title:       "Database Restored",
		Message:     fmt.Sprintf("Database restored from backup: %s", filename),
		Category:    "system",
		ActionURL:   "/settings",
		ActionLabel: "View Settings",
		Metadata: map[string]interface{}{
			"filename": filename,
		},
	}

	_, err := s.Create(req)
	return err
}
