package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// MediaService handles video metadata extraction
type MediaService struct{
	hwAccel string
	hwAccelOnce sync.Once
}

// NewMediaService creates a new media service
func NewMediaService() *MediaService {
	return &MediaService{}
}

// detectHardwareEncoder detects available hardware encoder
func (s *MediaService) detectHardwareEncoder() string {
	s.hwAccelOnce.Do(func() {
		// Check if ffmpeg is available
		_, err := exec.LookPath("ffmpeg")
		if err != nil {
			s.hwAccel = ""
			return
		}

		// Get list of available encoders
		cmd := exec.Command("ffmpeg", "-hide_banner", "-encoders")
		output, err := cmd.Output()
		if err != nil {
			s.hwAccel = ""
			return
		}

		encoders := string(output)

		// Check for hardware encoders in order of preference
		// NVIDIA NVENC
		if strings.Contains(encoders, "h264_nvenc") {
			s.hwAccel = "h264_nvenc"
			log.Println("Hardware acceleration: NVIDIA NVENC detected")
			return
		}

		// Intel Quick Sync
		if strings.Contains(encoders, "h264_qsv") {
			s.hwAccel = "h264_qsv"
			log.Println("Hardware acceleration: Intel Quick Sync detected")
			return
		}

		// AMD AMF
		if strings.Contains(encoders, "h264_amf") {
			s.hwAccel = "h264_amf"
			log.Println("Hardware acceleration: AMD AMF detected")
			return
		}

		// Fallback to software encoding
		s.hwAccel = ""
		log.Println("Hardware acceleration: Not available, using software encoding")
	})
	return s.hwAccel
}

// VideoMetadata represents extracted video metadata
type VideoMetadata struct {
	Duration   float64 `json:"duration"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Bitrate    int64   `json:"bitrate"`
	Codec      string  `json:"codec"`
	FrameRate  float64 `json:"frame_rate"`
	Size       int64   `json:"size"`
	HasAudio   bool    `json:"has_audio"`
	AudioCodec string  `json:"audio_codec,omitempty"`
}

// FFProbeOutput represents the output from ffprobe
type FFProbeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
	} `json:"streams"`
}

// ExtractMetadata extracts metadata from a video file using ffprobe
func (s *MediaService) ExtractMetadata(filePath string) (*VideoMetadata, error) {
	// Check if ffprobe is available
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found in PATH: %w", err)
	}

	// Run ffprobe to get video metadata
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var probeOutput FFProbeOutput
	if err := json.Unmarshal(output, &probeOutput); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	metadata := &VideoMetadata{}

	// Extract duration
	if duration, err := strconv.ParseFloat(probeOutput.Format.Duration, 64); err == nil {
		metadata.Duration = duration
	}

	// Extract file size
	if size, err := strconv.ParseInt(probeOutput.Format.Size, 10, 64); err == nil {
		metadata.Size = size
	}

	// Extract bitrate
	if bitrate, err := strconv.ParseInt(probeOutput.Format.BitRate, 10, 64); err == nil {
		metadata.Bitrate = bitrate
	}

	// Extract video and audio stream information
	for _, stream := range probeOutput.Streams {
		switch stream.CodecType {
		case "video":
			metadata.Width = stream.Width
			metadata.Height = stream.Height
			metadata.Codec = stream.CodecName

			// Parse frame rate
			if stream.RFrameRate != "" {
				parts := strings.Split(stream.RFrameRate, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den != 0 {
						metadata.FrameRate = num / den
					}
				}
			}
		case "audio":
			metadata.HasAudio = true
			metadata.AudioCodec = stream.CodecName
		}
	}

	return metadata, nil
}

// GenerateThumbnail generates a thumbnail for a video file
func (s *MediaService) GenerateThumbnail(filePath, outputPath string, timestamp float64) error {
	// Check if ffmpeg is available
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	// Detect hardware encoder
	hwEncoder := s.detectHardwareEncoder()

	// Build ffmpeg command with optional hardware acceleration
	args := []string{
		"-ss", fmt.Sprintf("%.2f", timestamp),
	}

	// Add hardware acceleration flags if available
	if hwEncoder != "" {
		switch hwEncoder {
		case "h264_nvenc":
			args = append(args, "-hwaccel", "cuda", "-hwaccel_output_format", "cuda")
		case "h264_qsv":
			args = append(args, "-hwaccel", "qsv", "-hwaccel_output_format", "qsv")
		case "h264_amf":
			args = append(args, "-hwaccel", "d3d11va")
		}
	}

	args = append(args,
		"-i", filePath,
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-q:v", "2", // Higher quality thumbnails
		"-y",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)

	if err := cmd.Run(); err != nil {
		// If hardware acceleration fails, retry with software encoding
		if hwEncoder != "" {
			log.Printf("Hardware acceleration failed for %s, retrying with software encoding: %v", filePath, err)
			cmd := exec.Command("ffmpeg",
				"-ss", fmt.Sprintf("%.2f", timestamp),
				"-i", filePath,
				"-vframes", "1",
				"-vf", "scale=320:-1",
				"-q:v", "2",
				"-y",
				outputPath,
			)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to generate thumbnail: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return nil
}

// ThumbnailConfig holds configuration for thumbnail generation
type ThumbnailConfig struct {
	LibraryID      int64   // Library ID for folder hierarchy
	LibraryPath    string  // Base library path
	VideoFilePath  string  // Full path to video file
	Duration       float64 // Video duration for timestamp calculation
	ThumbnailDir   string  // Base thumbnail directory (e.g., "./assets/thumbnails")
}

// ThumbnailResult holds the result of thumbnail generation
type ThumbnailResult struct {
	RelativePath string // Relative path for database storage (e.g., "thumbnails/1/folder/video.jpg")
	FullPath     string // Full filesystem path (e.g., "./assets/thumbnails/1/folder/video.jpg")
	URLPath      string // URL path for frontend access (e.g., "thumbnails/1/folder/video.jpg")
}

// GenerateThumbnailHierarchical generates a thumbnail using folder hierarchy structure
// This creates thumbnails in: {thumbnailDir}/{libraryID}/{relativePath}/{filename}.jpg
func (s *MediaService) GenerateThumbnailHierarchical(config ThumbnailConfig) (*ThumbnailResult, error) {
	// Calculate relative path from library to video file
	relativeVideoPath, err := filepath.Rel(config.LibraryPath, config.VideoFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate relative path: %w", err)
	}

	// Get the directory path and filename
	relativeDir := filepath.Dir(relativeVideoPath)
	videoFileName := filepath.Base(config.VideoFilePath)
	videoExt := filepath.Ext(videoFileName)
	thumbnailName := strings.TrimSuffix(videoFileName, videoExt) + ".jpg"

	// Build the hierarchical thumbnail directory structure
	// Format: {thumbnailDir}/{libraryID}/{relativeDir}
	var libraryThumbnailDir string
	if relativeDir == "." || relativeDir == "" {
		// Video is in root of library
		libraryThumbnailDir = filepath.Join(config.ThumbnailDir, fmt.Sprintf("%d", config.LibraryID))
	} else {
		// Video is in subdirectory
		libraryThumbnailDir = filepath.Join(config.ThumbnailDir, fmt.Sprintf("%d", config.LibraryID), relativeDir)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(libraryThumbnailDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Full path to thumbnail file
	thumbnailFullPath := filepath.Join(libraryThumbnailDir, thumbnailName)

	// Calculate thumbnail timestamp (10% of duration, capped at 5 seconds)
	timestamp := config.Duration * 0.1
	if timestamp > 5.0 {
		timestamp = 5.0
	}
	if timestamp < 0 {
		timestamp = 1.0 // Fallback to 1 second if duration is invalid
	}

	// Generate the thumbnail using existing method
	if err := s.GenerateThumbnail(config.VideoFilePath, thumbnailFullPath, timestamp); err != nil {
		return nil, err
	}

	// Build the relative path for database storage
	// Format: thumbnails/{libraryID}/{relativeDir}/{filename}
	var dbPath string
	if relativeDir == "." || relativeDir == "" {
		dbPath = fmt.Sprintf("thumbnails/%d/%s", config.LibraryID, thumbnailName)
	} else {
		// Convert backslashes to forward slashes for URL compatibility
		urlRelativeDir := filepath.ToSlash(relativeDir)
		dbPath = fmt.Sprintf("thumbnails/%d/%s/%s", config.LibraryID, urlRelativeDir, thumbnailName)
	}

	result := &ThumbnailResult{
		RelativePath: dbPath,
		FullPath:     thumbnailFullPath,
		URLPath:      dbPath,
	}

	return result, nil
}

// GetThumbnailPath returns the expected thumbnail path without generating it
// Useful for checking if a thumbnail already exists
func (s *MediaService) GetThumbnailPath(config ThumbnailConfig) *ThumbnailResult {
	// Calculate relative path from library to video file
	relativeVideoPath, err := filepath.Rel(config.LibraryPath, config.VideoFilePath)
	if err != nil {
		log.Printf("Failed to calculate relative path: %v", err)
		return nil
	}

	// Get the directory path and filename
	relativeDir := filepath.Dir(relativeVideoPath)
	videoFileName := filepath.Base(config.VideoFilePath)
	videoExt := filepath.Ext(videoFileName)
	thumbnailName := strings.TrimSuffix(videoFileName, videoExt) + ".jpg"

	// Build the hierarchical thumbnail directory structure
	var libraryThumbnailDir string
	if relativeDir == "." || relativeDir == "" {
		libraryThumbnailDir = filepath.Join(config.ThumbnailDir, fmt.Sprintf("%d", config.LibraryID))
	} else {
		libraryThumbnailDir = filepath.Join(config.ThumbnailDir, fmt.Sprintf("%d", config.LibraryID), relativeDir)
	}

	thumbnailFullPath := filepath.Join(libraryThumbnailDir, thumbnailName)

	// Build the relative path for database storage
	var dbPath string
	if relativeDir == "." || relativeDir == "" {
		dbPath = fmt.Sprintf("thumbnails/%d/%s", config.LibraryID, thumbnailName)
	} else {
		urlRelativeDir := filepath.ToSlash(relativeDir)
		dbPath = fmt.Sprintf("thumbnails/%d/%s/%s", config.LibraryID, urlRelativeDir, thumbnailName)
	}

	return &ThumbnailResult{
		RelativePath: dbPath,
		FullPath:     thumbnailFullPath,
		URLPath:      dbPath,
	}
}

// ThumbnailExists checks if a thumbnail file already exists
func (s *MediaService) ThumbnailExists(thumbnailPath string) bool {
	_, err := os.Stat(thumbnailPath)
	return err == nil
}
