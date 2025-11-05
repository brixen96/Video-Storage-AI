package api

import (
	"net/http"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var scannerService *services.ScannerService

// ScanRequest represents a request to scan a directory
type ScanRequest struct {
	Directory string `json:"directory" binding:"required"`
	Type      string `json:"type" binding:"required"` // "performers" or "videos"
	Recursive bool   `json:"recursive"`
}

// scanDirectory scans a directory for performers or videos
func scanDirectory(c *gin.Context) {
	if scannerService == nil {
		scannerService = services.NewScannerService()
	}

	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	switch req.Type {
	case "performers":
		result, err := scannerService.ScanPerformerDirectory(req.Directory)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to scan performer directory",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusOK, models.SuccessResponse(result, "Performer directory scanned successfully"))

	case "videos":
		videos, err := scannerService.ScanVideosDirectory(req.Directory, req.Recursive)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
				"Failed to scan video directory",
				err.Error(),
			))
			return
		}
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"videos_found": len(videos),
			"videos":       videos,
		}, "Video directory scanned successfully"))

	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid scan type",
			"type must be 'performers' or 'videos'",
		))
	}
}

var fileService *services.FileService

// ensureFileService initializes the service if needed
func ensureFileService() *services.FileService {
	if fileService == nil {
		fileService = services.NewFileService()
	}
	return fileService
}

// RenameFileRequest represents a request to rename a file
type RenameFileRequest struct {
	LibraryID int64  `json:"library_id" binding:"required"`
	FilePath  string `json:"file_path" binding:"required"`
	NewName   string `json:"new_name" binding:"required"`
}

// renameFile renames a file
func renameFile(c *gin.Context) {
	svc := ensureFileService()

	var req RenameFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := svc.RenameFile(req.LibraryID, req.FilePath, req.NewName); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to rename file",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		gin.H{"file_path": req.FilePath, "new_name": req.NewName},
		"File renamed successfully",
	))
}

// MoveFileRequest represents a request to move a file
type MoveFileRequest struct {
	LibraryID       int64  `json:"library_id" binding:"required"`
	SourcePath      string `json:"source_path" binding:"required"`
	DestinationPath string `json:"destination_path" binding:"required"`
}

// moveFile moves a file to a new location
func moveFile(c *gin.Context) {
	svc := ensureFileService()

	var req MoveFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := svc.MoveFile(req.LibraryID, req.SourcePath, req.DestinationPath); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to move file",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		gin.H{"source": req.SourcePath, "destination": req.DestinationPath},
		"File moved successfully",
	))
}

// DeleteFileRequest represents a request to delete a file
type DeleteFileRequest struct {
	LibraryID int64  `json:"library_id" binding:"required"`
	FilePath  string `json:"file_path" binding:"required"`
}

// deleteFile deletes a file
func deleteFile(c *gin.Context) {
	svc := ensureFileService()

	var req DeleteFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid request body",
			err.Error(),
		))
		return
	}

	if err := svc.DeleteFile(req.LibraryID, req.FilePath); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to delete file",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(
		gin.H{"file_path": req.FilePath},
		"File deleted successfully",
	))
}
