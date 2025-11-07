package services

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// MediaService handles video metadata extraction
type MediaService struct{}

// NewMediaService creates a new media service
func NewMediaService() *MediaService {
	return &MediaService{}
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
		if stream.CodecType == "video" {
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
		} else if stream.CodecType == "audio" {
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

	// Generate thumbnail at specified timestamp
	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", filePath,
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-y",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return nil
}
