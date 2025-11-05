package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileService handles file operations
type FileService struct {
	libraryService *LibraryService
}

// NewFileService creates a new file service
func NewFileService() *FileService {
	return &FileService{
		libraryService: NewLibraryService(),
	}
}

// MoveFile moves a file within a library
func (s *FileService) MoveFile(libraryID int64, sourcePath, destPath string) error {
	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return fmt.Errorf("library not found: %w", err)
	}

	// Construct full paths
	sourceFullPath := filepath.Join(library.Path, sourcePath)
	destFullPath := filepath.Join(library.Path, destPath)

	// Normalize paths
	sourceFullPath = filepath.Clean(sourceFullPath)
	destFullPath = filepath.Clean(destFullPath)

	// Security check: ensure paths are within library
	if !strings.HasPrefix(sourceFullPath, filepath.Clean(library.Path)) {
		return fmt.Errorf("source path traversal detected")
	}
	if !strings.HasPrefix(destFullPath, filepath.Clean(library.Path)) {
		return fmt.Errorf("destination path traversal detected")
	}

	// Check if source exists
	if _, err := os.Stat(sourceFullPath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist")
	}

	// Check if destination already exists
	if _, err := os.Stat(destFullPath); err == nil {
		return fmt.Errorf("destination file already exists")
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destFullPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Move file
	if err := os.Rename(sourceFullPath, destFullPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// RenameFile renames a file within a library
func (s *FileService) RenameFile(libraryID int64, filePath, newName string) error {
	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return fmt.Errorf("library not found: %w", err)
	}

	// Construct full path
	fullPath := filepath.Join(library.Path, filePath)
	fullPath = filepath.Clean(fullPath)

	// Security check: ensure path is within library
	if !strings.HasPrefix(fullPath, filepath.Clean(library.Path)) {
		return fmt.Errorf("path traversal detected")
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist")
	}

	// Construct new path
	dir := filepath.Dir(fullPath)
	newPath := filepath.Join(dir, newName)

	// Check if new name already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("file with new name already exists")
	}

	// Rename file
	if err := os.Rename(fullPath, newPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// DeleteFile deletes a file from a library
func (s *FileService) DeleteFile(libraryID int64, filePath string) error {
	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return fmt.Errorf("library not found: %w", err)
	}

	// Construct full path
	fullPath := filepath.Join(library.Path, filePath)
	fullPath = filepath.Clean(fullPath)

	// Security check: ensure path is within library
	if !strings.HasPrefix(fullPath, filepath.Clean(library.Path)) {
		return fmt.Errorf("path traversal detected")
	}

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist")
	}

	// Don't allow deleting directories with this method
	if fileInfo.IsDir() {
		return fmt.Errorf("cannot delete directories")
	}

	// Delete file
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// CopyFile copies a file within a library
func (s *FileService) CopyFile(libraryID int64, sourcePath, destPath string) error {
	// Get library
	library, err := s.libraryService.GetByID(libraryID)
	if err != nil {
		return fmt.Errorf("library not found: %w", err)
	}

	// Construct full paths
	sourceFullPath := filepath.Join(library.Path, sourcePath)
	destFullPath := filepath.Join(library.Path, destPath)

	// Normalize paths
	sourceFullPath = filepath.Clean(sourceFullPath)
	destFullPath = filepath.Clean(destFullPath)

	// Security check: ensure paths are within library
	if !strings.HasPrefix(sourceFullPath, filepath.Clean(library.Path)) {
		return fmt.Errorf("source path traversal detected")
	}
	if !strings.HasPrefix(destFullPath, filepath.Clean(library.Path)) {
		return fmt.Errorf("destination path traversal detected")
	}

	// Check if source exists
	if _, err := os.Stat(sourceFullPath); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist")
	}

	// Check if destination already exists
	if _, err := os.Stat(destFullPath); err == nil {
		return fmt.Errorf("destination file already exists")
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destFullPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read source file
	sourceData, err := os.ReadFile(sourceFullPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(destFullPath, sourceData, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}
