package services

import (
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/database"
)

// videoCodecTiers maps codec names to quality scores (0–100).
// Matches are substring-based so "hevc" matches "hevc_main10" etc.
var videoCodecTiers = map[string]float64{
	"av1":        100,
	"hevc":        90,
	"h265":        90,
	"vp9":         80,
	"avc":         65,
	"h264":        65,
	"vp8":         50,
	"mpeg4":       40,
	"mpeg2video":  30,
	"mpeg1video":  20,
	"wmv":         25,
	"theora":      35,
}

// VideoQualityScore holds the quality scores for a single video in a comparison.
type VideoQualityScore struct {
	VideoID         int64   `json:"video_id"`
	Title           string  `json:"title"`
	FilePath        string  `json:"file_path"`
	ThumbnailPath   string  `json:"thumbnail_path,omitempty"`
	FileSize        int64   `json:"file_size"`
	Duration        float64 `json:"duration"`
	Resolution      string  `json:"resolution"`
	Codec           string  `json:"codec"`
	Bitrate         int64   `json:"bitrate"`
	FPS             float64 `json:"fps"`
	VideoScore      float64 `json:"video_score"`
	TechnicalScore  float64 `json:"technical_score"`
	OverallScore    float64 `json:"overall_score"`
	Rank            int     `json:"rank"`
	IsRecommended   bool    `json:"is_recommended"`
	IsTopazEnhanced bool    `json:"is_topaz_enhanced"`
	TopazType       string  `json:"topaz_type,omitempty"` // "upscaled", "interpolated", "upscaled+interpolated"
}

// VideoQualityResult holds the full comparison result for a group of videos.
type VideoQualityResult struct {
	Videos         []VideoQualityScore `json:"videos"`
	Recommendation string              `json:"recommendation"`
}

// VideoQualityService scores and ranks video files by technical quality.
type VideoQualityService struct{}

func NewVideoQualityService() *VideoQualityService {
	return &VideoQualityService{}
}

// CompareByIDs fetches the given videos from the DB, scores them, and returns
// a ranked VideoQualityResult. Requires at least 2 IDs.
func (s *VideoQualityService) CompareByIDs(videoIDs []int64) (*VideoQualityResult, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	if len(videoIDs) < 2 {
		return nil, fmt.Errorf("at least 2 video IDs are required for comparison")
	}

	var scores []VideoQualityScore
	for _, id := range videoIDs {
		row := db.QueryRow(
			`SELECT id, title, file_path, COALESCE(thumbnail_path,''), file_size, duration,
			        COALESCE(codec,''), COALESCE(resolution,''), COALESCE(bitrate,0), COALESCE(fps,0)
			 FROM videos WHERE id = ?`, id)
		var v VideoQualityScore
		if err := row.Scan(&v.VideoID, &v.Title, &v.FilePath, &v.ThumbnailPath,
			&v.FileSize, &v.Duration, &v.Codec, &v.Resolution, &v.Bitrate, &v.FPS); err != nil {
			continue
		}
		scores = append(scores, v)
	}

	if len(scores) < 2 {
		return nil, fmt.Errorf("could not load enough videos for comparison")
	}

	return s.finalizeScores(scores), nil
}

// CompareFromMixed scores library videos (by ID) and uploaded files together.
// Uploaded files are processed via ffprobe from the provided multipart headers.
// Requires at least 2 total entries.
func (s *VideoQualityService) CompareFromMixed(videoIDs []int64, files []*multipart.FileHeader) (*VideoQualityResult, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var scores []VideoQualityScore

	// Load library videos from DB
	for _, id := range videoIDs {
		row := db.QueryRow(
			`SELECT id, title, file_path, COALESCE(thumbnail_path,''), file_size, duration,
			        COALESCE(codec,''), COALESCE(resolution,''), COALESCE(bitrate,0), COALESCE(fps,0)
			 FROM videos WHERE id = ?`, id)
		var v VideoQualityScore
		if err := row.Scan(&v.VideoID, &v.Title, &v.FilePath, &v.ThumbnailPath,
			&v.FileSize, &v.Duration, &v.Codec, &v.Resolution, &v.Bitrate, &v.FPS); err != nil {
			continue
		}
		scores = append(scores, v)
	}

	// Process uploaded files via ffprobe
	mediaSvc := NewMediaService()
	for _, fh := range files {
		ext := filepath.Ext(fh.Filename)
		tmpFile, err := os.CreateTemp("", "vsai-upload-*"+ext)
		if err != nil {
			continue
		}
		tmpPath := tmpFile.Name()

		src, err := fh.Open()
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			continue
		}
		_, copyErr := io.Copy(tmpFile, src)
		src.Close()
		tmpFile.Close()
		if copyErr != nil {
			os.Remove(tmpPath)
			continue
		}

		meta, err := mediaSvc.ExtractMetadata(tmpPath)
		os.Remove(tmpPath)
		if err != nil {
			continue
		}

		res := ""
		if meta.Width > 0 && meta.Height > 0 {
			res = fmt.Sprintf("%dx%d", meta.Width, meta.Height)
		}
		scores = append(scores, VideoQualityScore{
			VideoID:    0, // not in DB
			Title:      fh.Filename,
			FilePath:   fh.Filename, // use original name for Topaz/ext detection
			FileSize:   meta.Size,
			Duration:   meta.Duration,
			Resolution: res,
			Codec:      meta.Codec,
			Bitrate:    meta.Bitrate,
			FPS:        meta.FrameRate,
		})
	}

	if len(scores) < 2 {
		return nil, fmt.Errorf("could not load enough videos for comparison (need at least 2)")
	}

	return s.finalizeScores(scores), nil
}

// finalizeScores detects Topaz, calculates scores, sorts/ranks, and builds the recommendation.
func (s *VideoQualityService) finalizeScores(scores []VideoQualityScore) *VideoQualityResult {
	for i := range scores {
		t := detectTopazType(scores[i].FilePath)
		scores[i].TopazType = t
		scores[i].IsTopazEnhanced = t != ""
	}

	s.calculateScores(scores)

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].OverallScore > scores[j].OverallScore
	})
	for i := range scores {
		scores[i].Rank = i + 1
	}
	// IsRecommended = best non-Topaz source; fall back to best overall if all are Topaz
	recommended := false
	for i := range scores {
		if !scores[i].IsTopazEnhanced {
			scores[i].IsRecommended = true
			recommended = true
			break
		}
	}
	if !recommended && len(scores) > 0 {
		scores[0].IsRecommended = true
	}

	return &VideoQualityResult{
		Videos:         scores,
		Recommendation: s.buildRecommendation(scores),
	}
}

// calculateScores populates VideoScore, TechnicalScore, and OverallScore on each entry.
func (s *VideoQualityService) calculateScores(scores []VideoQualityScore) {
	// Relative bitrate scoring across the group (highest = 100)
	bitrateRel := relativeScoresFloat(scores, func(v VideoQualityScore) float64 { return float64(v.Bitrate) })

	for i := range scores {
		v := &scores[i]

		// ── Video score ──────────────────────────────────────────────────────
		res := vqResolutionScore(v.Resolution)
		codec := vqCodecScore(v.Codec)
		fps := vqFPSScore(v.FPS)
		vbr := bitrateRel[i]

		v.VideoScore = round1dp(res*0.35 + codec*0.25 + vbr*0.25 + fps*0.15)

		// ── Technical score ──────────────────────────────────────────────────
		tech := 80.0
		ext := strings.ToLower(filepath.Ext(v.FilePath))
		switch ext {
		case ".mkv", ".mp4", ".mov":
			tech += 10.0
		case ".avi", ".wmv", ".flv", ".rm", ".rmvb":
			tech -= 10.0
		}

		// Penalise outlier durations (likely truncated / different edit)
		if len(scores) > 1 {
			var totalDur float64
			var count int
			for _, o := range scores {
				if o.Duration > 0 {
					totalDur += o.Duration
					count++
				}
			}
			if count > 0 && v.Duration > 0 {
				avgDur := totalDur / float64(count)
				if avgDur > 0 {
					deviation := math.Abs(v.Duration-avgDur) / avgDur
					if deviation > 0.10 {
						tech -= 20.0
					} else if deviation > 0.05 {
						tech -= 10.0
					}
				}
			}
		}

		v.TechnicalScore = round1dp(math.Min(100, math.Max(0, tech)))

		// ── Overall score (no audio: video 80%, technical 20%) ───────────────
		v.OverallScore = round1dp(v.VideoScore*0.80 + v.TechnicalScore*0.20)
	}
}

// buildRecommendation returns a human-readable explanation of the top-ranked choice.
func (s *VideoQualityService) buildRecommendation(ranked []VideoQualityScore) string {
	if len(ranked) == 0 {
		return "No videos could be scored."
	}
	if len(ranked) == 1 {
		return fmt.Sprintf("%s is the only video analysed.", ranked[0].Title)
	}

	// Separate Topaz-enhanced files from original source files (ranked slice is already sorted by score)
	var sourceFiles, topazFiles []VideoQualityScore
	for _, v := range ranked {
		if v.IsTopazEnhanced {
			topazFiles = append(topazFiles, v)
		} else {
			sourceFiles = append(sourceFiles, v)
		}
	}

	if len(topazFiles) > 0 && len(sourceFiles) > 0 {
		bestSource := sourceFiles[0]
		var descs []string
		for _, t := range topazFiles {
			descs = append(descs, fmt.Sprintf("%q (%s)", t.Title, t.TopazType))
		}
		return fmt.Sprintf("Topaz AI Enhanced: %s  •  Best source file: %q (%.0f%% overall score)",
			strings.Join(descs, ", "), bestSource.Title, bestSource.OverallScore)
	}

	if len(topazFiles) > 0 {
		winner := ranked[0]
		return fmt.Sprintf("All files are Topaz AI enhanced. Best version: %q (%s, %.0f%% overall).",
			winner.Title, winner.TopazType, winner.OverallScore)
	}

	winner := ranked[0]
	runner := ranked[1]

	var reasons []string

	if winner.VideoScore > runner.VideoScore+2 {
		var parts []string
		wW, wH := vqParseResolution(winner.Resolution)
		rW, rH := vqParseResolution(runner.Resolution)
		if wW*wH > rW*rH {
			parts = append(parts, fmt.Sprintf("higher resolution (%s vs %s)", winner.Resolution, runner.Resolution))
		}
		if vqCodecScore(winner.Codec) > vqCodecScore(runner.Codec) {
			parts = append(parts, fmt.Sprintf("better codec (%s vs %s)",
				strings.ToUpper(winner.Codec), strings.ToUpper(runner.Codec)))
		}
		if float64(winner.Bitrate) > float64(runner.Bitrate)*1.1 {
			parts = append(parts, fmt.Sprintf("higher bitrate (%s vs %s)",
				vqFormatBitrate(winner.Bitrate), vqFormatBitrate(runner.Bitrate)))
		}
		if len(parts) > 0 {
			reasons = append(reasons, "Video: "+strings.Join(parts, ", "))
		}
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "marginally better overall quality")
	}

	return fmt.Sprintf("Recommended: \"%s\"  —  %s", winner.Title, strings.Join(reasons, "  •  "))
}

// ── Helper functions ──────────────────────────────────────────────────────────

func relativeScoresFloat(scores []VideoQualityScore, get func(VideoQualityScore) float64) []float64 {
	maxVal := 0.0
	for _, v := range scores {
		if val := get(v); val > maxVal {
			maxVal = val
		}
	}
	result := make([]float64, len(scores))
	for i, v := range scores {
		val := get(v)
		if maxVal == 0 {
			result[i] = 50.0
		} else if val > 0 {
			result[i] = math.Min(100.0, val/maxVal*100)
		} else {
			result[i] = 20.0
		}
	}
	return result
}

func vqCodecScore(codec string) float64 {
	lower := strings.ToLower(codec)
	for key, score := range videoCodecTiers {
		if strings.Contains(lower, key) {
			return score
		}
	}
	return 35.0
}

func vqResolutionScore(resolution string) float64 {
	w, h := vqParseResolution(resolution)
	pixels := w * h
	switch {
	case pixels >= 8_000_000:
		return 100.0
	case pixels >= 3_500_000:
		return 85.0
	case pixels >= 2_000_000:
		return 70.0
	case pixels >= 900_000:
		return 50.0
	case pixels >= 400_000:
		return 30.0
	default:
		return 15.0
	}
}

func vqFPSScore(fps float64) float64 {
	switch {
	case fps >= 120:
		return 100.0
	case fps >= 60:
		return 90.0
	case fps >= 48:
		return 80.0
	case fps >= 30:
		return 70.0
	case fps >= 24:
		return 60.0
	default:
		return 40.0
	}
}

func vqParseResolution(res string) (int, int) {
	parts := strings.SplitN(res, "x", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	w, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return w, h
}

func vqFormatBitrate(bps int64) string {
	if bps >= 1_000_000 {
		return fmt.Sprintf("%.1f Mbps", float64(bps)/1_000_000)
	}
	return fmt.Sprintf("%d Kbps", bps/1000)
}

func round1dp(v float64) float64 {
	return math.Round(v*10) / 10
}

// detectTopazType returns the Topaz Video AI enhancement type based on filename suffixes.
// "_prob4" = upscaled, "_chr2" = interpolated (higher FPS), "_chr2_prob4" = both.
// Returns "" if the file has no Topaz suffix.
func detectTopazType(filePath string) string {
	base := filepath.Base(filePath)
	name := strings.ToLower(strings.TrimSuffix(base, filepath.Ext(base)))
	hasChr2 := strings.Contains(name, "_chr2")
	hasProb4 := strings.Contains(name, "_prob4")
	switch {
	case hasChr2 && hasProb4:
		return "upscaled+interpolated"
	case hasProb4:
		return "upscaled"
	case hasChr2:
		return "interpolated"
	default:
		return ""
	}
}
