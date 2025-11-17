package services

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// ConversionService handles video format conversion using FFmpeg
type ConversionService struct {
	videoService    *VideoService
	mediaService    *MediaService
	activityService *ActivityService
}

// NewConversionService creates a new conversion service
func NewConversionService(videoService *VideoService, mediaService *MediaService, activityService *ActivityService) *ConversionService {
	return &ConversionService{
		videoService:    videoService,
		mediaService:    mediaService,
		activityService: activityService,
	}
}

// ConvertVideoToMP4 converts a video file to MP4 format
func (s *ConversionService) ConvertVideoToMP4(videoID int64) (*models.Video, error) {
	// Get the original video
	video, err := s.videoService.GetByID(videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %w", err)
	}

	// Check if already MP4
	if strings.ToLower(filepath.Ext(video.FilePath)) == ".mp4" {
		return nil, fmt.Errorf("video is already in MP4 format")
	}

	// Check if already has a conversion
	if video.ConvertedTo != nil {
		convertedVideo, err := s.videoService.GetByID(*video.ConvertedTo)
		if err == nil {
			return convertedVideo, nil
		}
	}

	// Create output file path
	ext := filepath.Ext(video.FilePath)
	outputPath := strings.TrimSuffix(video.FilePath, ext) + "_converted.mp4"

	// Check if output file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return nil, fmt.Errorf("converted file already exists at: %s", outputPath)
	}

	// Create activity for tracking
	activity, err := s.activityService.Create(&models.ActivityLogCreate{
		TaskType: "video_conversion",
		Status:   "running",
		Message:  fmt.Sprintf("Converting %s to MP4", filepath.Base(video.FilePath)),
	})
	if err != nil {
		log.Printf("Failed to create conversion activity: %v", err)
	}

	// Build FFmpeg command
	// Using H.264 codec with high quality settings for broad compatibility
	cmd := exec.Command("ffmpeg",
		"-i", video.FilePath,                    // Input file
		"-c:v", "libx264",                       // Video codec
		"-preset", "medium",                     // Encoding speed/quality trade-off
		"-crf", "23",                            // Quality (lower = better, 18-28 is good range)
		"-c:a", "aac",                           // Audio codec
		"-b:a", "192k",                          // Audio bitrate
		"-movflags", "+faststart",               // Enable streaming
		"-y",                                    // Overwrite output file if exists
		outputPath,                              // Output file
	)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := fmt.Sprintf("FFmpeg conversion failed: %v\nOutput: %s", err, string(output))
		log.Printf(errMsg)

		if activity != nil {
			s.activityService.FailTask(int(activity.ID), errMsg)
		}

		return nil, fmt.Errorf("conversion failed: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		if activity != nil {
			s.activityService.FailTask(int(activity.ID), fmt.Sprintf("Failed to stat output file: %v", err))
		}
		return nil, fmt.Errorf("failed to stat output file: %w", err)
	}

	// Extract metadata from converted file
	metadata, err := s.mediaService.ExtractMetadata(outputPath)
	if err != nil {
		log.Printf("Warning: Failed to extract metadata from converted file: %v", err)
		// Continue anyway, we can create the video record without full metadata
	}

	// Create new video record
	newVideo := &models.VideoCreate{
		LibraryID:  video.LibraryID,
		Title:      video.Title + " (Converted)",
		FilePath:   outputPath,
		FileSize:   fileInfo.Size(),
	}

	if metadata != nil {
		newVideo.Duration = metadata.Duration
		newVideo.Codec = metadata.Codec
		newVideo.Resolution = fmt.Sprintf("%dx%d", metadata.Width, metadata.Height)
		newVideo.Bitrate = metadata.Bitrate
		newVideo.FPS = metadata.FrameRate
	}

	// Create the video in database
	createdVideo, err := s.videoService.Create(newVideo)
	if err != nil {
		if activity != nil {
			s.activityService.FailTask(int(activity.ID), fmt.Sprintf("Failed to create video record: %v", err))
		}
		return nil, fmt.Errorf("failed to create video record: %w", err)
	}

	// Link the videos together
	video.ConvertedTo = &createdVideo.ID
	createdVideo.ConvertedFrom = &video.ID

	// Update original video with conversion link
	if err := s.videoService.UpdateConversionLink(video.ID, createdVideo.ID, true); err != nil {
		log.Printf("Warning: Failed to update original video conversion link: %v", err)
	}

	// Update converted video with original link
	if err := s.videoService.UpdateConversionLink(createdVideo.ID, video.ID, false); err != nil {
		log.Printf("Warning: Failed to update converted video original link: %v", err)
	}

	// Copy relationships from original video
	// Copy performers
	if err := s.videoService.CopyPerformers(video.ID, createdVideo.ID); err != nil {
		log.Printf("Warning: Failed to copy performers: %v", err)
	}

	// Copy tags
	if err := s.videoService.CopyTags(video.ID, createdVideo.ID); err != nil {
		log.Printf("Warning: Failed to copy tags: %v", err)
	}

	// Copy studios
	if err := s.videoService.CopyStudios(video.ID, createdVideo.ID); err != nil {
		log.Printf("Warning: Failed to copy studios: %v", err)
	}

	// Copy groups
	if err := s.videoService.CopyGroups(video.ID, createdVideo.ID); err != nil {
		log.Printf("Warning: Failed to copy groups: %v", err)
	}

	// Copy other metadata
	if err := s.videoService.CopyMetadata(video.ID, createdVideo.ID); err != nil {
		log.Printf("Warning: Failed to copy metadata: %v", err)
	}

	// Complete activity
	if activity != nil {
		s.activityService.CompleteTask(
			int64(activity.ID),
			fmt.Sprintf("Successfully converted to MP4: %s", filepath.Base(outputPath)),
		)
	}

	return createdVideo, nil
}

// CheckFFmpegInstalled checks if FFmpeg is installed and available
func (s *ConversionService) CheckFFmpegInstalled() bool {
	cmd := exec.Command("ffmpeg", "-version")
	err := cmd.Run()
	return err == nil
}
