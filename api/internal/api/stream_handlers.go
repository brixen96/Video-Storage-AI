package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
	"github.com/brixen96/video-storage-ai/internal/services"
	"github.com/gin-gonic/gin"
)

var streamLibraryService *services.LibraryService

// ensureStreamLibraryService initializes the service if needed
func ensureStreamLibraryService() *services.LibraryService {
	if streamLibraryService == nil {
		streamLibraryService = services.NewLibraryService()
	}
	return streamLibraryService
}

// streamVideo streams a video file from a library
func streamVideo(c *gin.Context) {
	svc := ensureStreamLibraryService()

	// Get library ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid library ID",
			err.Error(),
		))
		return
	}

	// Get path parameter
	relativePath := c.Query("path")
	if relativePath == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Missing path parameter",
			"",
		))
		return
	}

	// Get library
	library, err := svc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Library not found",
			err.Error(),
		))
		return
	}

	// Construct full path
	fullPath := filepath.Join(library.Path, relativePath)
	fullPath = filepath.Clean(fullPath)

	// Security check: ensure path is within library
	if !strings.HasPrefix(fullPath, filepath.Clean(library.Path)) {
		c.JSON(http.StatusForbidden, models.ErrorResponseMsg(
			"Path traversal detected",
			"",
		))
		return
	}

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"File not found",
			err.Error(),
		))
		return
	}

	if fileInfo.IsDir() {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Path is a directory",
			"",
		))
		return
	}

	// Open file
	file, err := os.Open(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to open file",
			err.Error(),
		))
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	// Get file size
	fileSize := fileInfo.Size()

	// Set content type based on extension
	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := getContentType(ext)

	// Handle range requests for video seeking
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		handleRangeRequest(c, file, fileSize, contentType, rangeHeader)
	} else {
		// Serve entire file
		c.Header("Content-Type", contentType)
		c.Header("Content-Length", fmt.Sprintf("%d", fileSize))
		c.Header("Accept-Ranges", "bytes")
		c.Status(http.StatusOK)
		if _, err := io.Copy(c.Writer, file); err != nil {
			log.Printf("Failed to copy file to response: %v", err)
		}
	}
}

// handleRangeRequest handles HTTP range requests for video seeking
func handleRangeRequest(c *gin.Context, file *os.File, fileSize int64, contentType string, rangeHeader string) {
	// Parse range header (e.g., "bytes=0-1023")
	rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeHeader, "-")

	var start, end int64
	var err error

	if len(parts) != 2 {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Parse start
	if parts[0] != "" {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
	}

	// Parse end
	if parts[1] != "" {
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			c.Status(http.StatusRequestedRangeNotSatisfiable)
			return
		}
	} else {
		end = fileSize - 1
	}

	// Validate range
	if start < 0 || start >= fileSize || end >= fileSize || start > end {
		c.Status(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Seek to start position
	_, err = file.Seek(start, 0)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// Calculate content length
	contentLength := end - start + 1

	// Set headers for partial content
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", contentLength))
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.Header("Accept-Ranges", "bytes")
	c.Status(http.StatusPartialContent)

	// Stream the requested range
	if _, err := io.CopyN(c.Writer, file, contentLength); err != nil {
		log.Printf("Failed to copy file to response: %v", err)
	}
}

// streamVideoByID streams a video file by video ID
func streamVideoByID(c *gin.Context) {
	videoSvc := ensureVideoService()

	// Get video ID
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Invalid video ID",
			err.Error(),
		))
		return
	}

	// Get video
	video, err := videoSvc.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"Video not found",
			err.Error(),
		))
		return
	}

	fullPath := video.FilePath

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponseMsg(
			"File not found",
			err.Error(),
		))
		return
	}

	if fileInfo.IsDir() {
		c.JSON(http.StatusBadRequest, models.ErrorResponseMsg(
			"Path is a directory",
			"",
		))
		return
	}

	// Open file
	file, err := os.Open(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponseMsg(
			"Failed to open file",
			err.Error(),
		))
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	// Get file size
	fileSize := fileInfo.Size()

	// Set content type based on extension
	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := getContentType(ext)

	// Handle range requests for video seeking
	rangeHeader := c.GetHeader("Range")
	if rangeHeader != "" {
		handleRangeRequest(c, file, fileSize, contentType, rangeHeader)
	} else {
		// Serve entire file
		c.Header("Content-Type", contentType)
		c.Header("Content-Length", fmt.Sprintf("%d", fileSize))
		c.Header("Accept-Ranges", "bytes")
		c.Status(http.StatusOK)
		if _, err := io.Copy(c.Writer, file); err != nil {
			log.Printf("Failed to copy file to response: %v", err)
		}
	}
}

// getContentType returns the MIME type for a video file extension
func getContentType(ext string) string {
	mimeTypes := map[string]string{
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".ogg":  "video/ogg",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".m4v":  "video/mp4",
		".mpg":  "video/mpeg",
		".mpeg": "video/mpeg",
		".3gp":  "video/3gpp",
		".ts":   "video/mp2t",
		".m2ts": "video/mp2t",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "video/mp4" // default
}
