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

// MoveFileAcrossLibraries moves a file from one library to another
func (s *FileService) MoveFileAcrossLibraries(sourceLibraryID int64, sourcePath string, targetLibraryID int64, targetPath string) error {
	// Get source library
	sourceLibrary, err := s.libraryService.GetByID(sourceLibraryID)
	if err != nil {
		return fmt.Errorf("source library not found: %w", err)
	}

	// Get target library
	targetLibrary, err := s.libraryService.GetByID(targetLibraryID)
	if err != nil {
		return fmt.Errorf("target library not found: %w", err)
	}

	// Construct full paths
	sourceFullPath := filepath.Join(sourceLibrary.Path, sourcePath)
	targetFullPath := filepath.Join(targetLibrary.Path, targetPath)

	// Normalize paths
	sourceFullPath = filepath.Clean(sourceFullPath)
	targetFullPath = filepath.Clean(targetFullPath)

	// Security check: ensure paths are within their respective libraries
	if !strings.HasPrefix(sourceFullPath, filepath.Clean(sourceLibrary.Path)) {
		return fmt.Errorf("source path traversal detected")
	}
	if !strings.HasPrefix(targetFullPath, filepath.Clean(targetLibrary.Path)) {
		return fmt.Errorf("target path traversal detected")
	}

	// Check if source exists
	sourceInfo, err := os.Stat(sourceFullPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("source file or folder does not exist")
	}

	// If target path is a directory, append source filename
	if targetInfo, err := os.Stat(targetFullPath); err == nil && targetInfo.IsDir() {
		targetFullPath = filepath.Join(targetFullPath, filepath.Base(sourceFullPath))
	}

	// Check if destination already exists
	if _, err := os.Stat(targetFullPath); err == nil {
		return fmt.Errorf("destination file or folder already exists")
	}

	// Create destination directory if it doesn't exist
	targetDir := filepath.Dir(targetFullPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Move file or folder
	// Try os.Rename first (works if on same filesystem)
	if err := os.Rename(sourceFullPath, targetFullPath); err != nil {
		// If rename fails (likely different filesystem), copy and delete
		if sourceInfo.IsDir() {
			// For directories, we need to copy recursively
			if err := copyDir(sourceFullPath, targetFullPath); err != nil {
				return fmt.Errorf("failed to copy directory: %w", err)
			}
			// Remove source directory after successful copy
			if err := os.RemoveAll(sourceFullPath); err != nil {
				return fmt.Errorf("failed to remove source directory after copy: %w", err)
			}
		} else {
			// For files, read and write
			sourceData, err := os.ReadFile(sourceFullPath)
			if err != nil {
				return fmt.Errorf("failed to read source file: %w", err)
			}
			if err := os.WriteFile(targetFullPath, sourceData, 0644); err != nil {
				return fmt.Errorf("failed to write target file: %w", err)
			}
			// Remove source file after successful copy
			if err := os.Remove(sourceFullPath); err != nil {
				return fmt.Errorf("failed to remove source file after copy: %w", err)
			}
		}
	}

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read directory contents
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
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
