package services

import (
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"os"
	"os/exec"
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
	UpscaleRisk          string  `json:"upscale_risk,omitempty"` // "low", "medium", "high" — empty means none detected
	SharpnessScore       float64 `json:"sharpness_score,omitempty"`       // mean Laplacian variance across sampled frames
	PerceptualEfficiency float64 `json:"perceptual_efficiency,omitempty"` // sharpness normalised by sqrt(pixels) — cross-resolution comparable
	BestForTopaz         bool    `json:"best_for_topaz,omitempty"`        // highest perceptual efficiency among non-Topaz files
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
		w, h := vqParseResolution(v.Resolution)
		if sh, eff, err := vqDeepAnalyzeFile(v.FilePath, w, h, v.Duration); err == nil {
			v.SharpnessScore = sh
			v.PerceptualEfficiency = eff
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
		w, h := vqParseResolution(v.Resolution)
		if sh, eff, err := vqDeepAnalyzeFile(v.FilePath, w, h, v.Duration); err == nil {
			v.SharpnessScore = sh
			v.PerceptualEfficiency = eff
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
		if err != nil {
			os.Remove(tmpPath)
			continue
		}

		res := ""
		if meta.Width > 0 && meta.Height > 0 {
			res = fmt.Sprintf("%dx%d", meta.Width, meta.Height)
		}
		entry := VideoQualityScore{
			VideoID:    0, // not in DB
			Title:      fh.Filename,
			FilePath:   fh.Filename, // use original name for Topaz/ext detection
			FileSize:   meta.Size,
			Duration:   meta.Duration,
			Resolution: res,
			Codec:      meta.Codec,
			Bitrate:    meta.Bitrate,
			FPS:        meta.FrameRate,
		}
		if meta.Width > 0 && meta.Height > 0 {
			if sh, eff, err := vqDeepAnalyzeFile(tmpPath, meta.Width, meta.Height, meta.Duration); err == nil {
				entry.SharpnessScore = sh
				entry.PerceptualEfficiency = eff
			}
		}
		os.Remove(tmpPath)
		scores = append(scores, entry)
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
	// IsRecommended: highest PerceptualEfficiency among non-Topaz files (deep analysis always runs).
	// Falls back to highest OverallScore non-Topaz file when efficiency data is unavailable.
	recIdx := -1
	bestEff := 0.0
	for i, sc := range scores {
		if !sc.IsTopazEnhanced && sc.PerceptualEfficiency > bestEff {
			bestEff = sc.PerceptualEfficiency
			recIdx = i
		}
	}
	if recIdx == -1 { // no efficiency data — fall back to first non-Topaz by score
		for i := range scores {
			if !scores[i].IsTopazEnhanced {
				recIdx = i
				break
			}
		}
	}
	if recIdx == -1 && len(scores) > 0 {
		recIdx = 0
	}
	if recIdx >= 0 {
		scores[recIdx].IsRecommended = true
		if bestEff > 0 {
			scores[recIdx].BestForTopaz = true
		}
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

		// ── Upscale risk (skip Topaz — those are intentionally upscaled) ─────
		if !v.IsTopazEnhanced {
			v.UpscaleRisk = vqUpscaleRisk(*v)
		}

		// ── Video score ──────────────────────────────────────────────────────
		// Adjust effective pixel count down when upscale risk is detected so
		// the resolution component reflects real content detail, not declared size.
		w, h := vqParseResolution(v.Resolution)
		effectivePixels := float64(w * h)
		switch v.UpscaleRisk {
		case "low":
			effectivePixels *= 0.80
		case "medium":
			effectivePixels *= 0.50
		case "high":
			effectivePixels *= 0.25
		}
		res := vqResolutionScoreFromPixels(effectivePixels)
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

// buildRecommendation returns a human-readable explanation of the recommended choice.
func (s *VideoQualityService) buildRecommendation(ranked []VideoQualityScore) string {
	if len(ranked) == 0 {
		return "No videos could be scored."
	}
	if len(ranked) == 1 {
		return fmt.Sprintf("%s is the only video analysed.", ranked[0].Title)
	}

	// Find the recommended file (set by finalizeScores — may not be ranked[0])
	var winner VideoQualityScore
	for _, v := range ranked {
		if v.IsRecommended {
			winner = v
			break
		}
	}
	if winner.Title == "" {
		winner = ranked[0]
	}

	// Separate Topaz-enhanced from source files
	var sourceFiles, topazFiles []VideoQualityScore
	for _, v := range ranked {
		if v.IsTopazEnhanced {
			topazFiles = append(topazFiles, v)
		} else {
			sourceFiles = append(sourceFiles, v)
		}
	}

	if len(topazFiles) > 0 && len(sourceFiles) > 0 {
		var descs []string
		for _, t := range topazFiles {
			descs = append(descs, fmt.Sprintf("%q (%s)", t.Title, t.TopazType))
		}
		return fmt.Sprintf("Topaz AI Enhanced: %s  •  Best source file: %q (%.0f%% overall score)",
			strings.Join(descs, ", "), winner.Title, winner.OverallScore)
	}

	if len(topazFiles) > 0 {
		w := ranked[0]
		return fmt.Sprintf("All files are Topaz AI enhanced. Best version: %q (%s, %.0f%% overall).",
			w.Title, w.TopazType, w.OverallScore)
	}

	// Build upscale warning suffix (shared by both recommendation paths)
	var upscaleWarnings []string
	for _, v := range ranked {
		if v.UpscaleRisk == "medium" || v.UpscaleRisk == "high" {
			upscaleWarnings = append(upscaleWarnings, fmt.Sprintf("%q (%s risk)", v.Title, v.UpscaleRisk))
		}
	}
	upscaleSuffix := ""
	if len(upscaleWarnings) > 0 {
		upscaleSuffix = fmt.Sprintf("  •  ⚠ Suspected upscale: %s", strings.Join(upscaleWarnings, ", "))
	}

	// Sharpness-based recommendation (deep analysis ran)
	if winner.PerceptualEfficiency > 0 {
		var runner VideoQualityScore
		for _, v := range ranked {
			if !v.IsTopazEnhanced && v.Title != winner.Title && v.PerceptualEfficiency > runner.PerceptualEfficiency {
				runner = v
			}
		}
		rec := fmt.Sprintf("Best source: \"%s\"  —  Sharpest at native resolution", winner.Title)
		if runner.Title != "" {
			rec += fmt.Sprintf(" (%.2f vs %.2f efficiency)", winner.PerceptualEfficiency, runner.PerceptualEfficiency)
		}
		rec += "  •  Recommended for Topaz upscaling"
		return rec + upscaleSuffix
	}

	// Score-based fallback
	var runner VideoQualityScore
	for _, v := range ranked {
		if v.Title != winner.Title && !v.IsTopazEnhanced {
			runner = v
			break
		}
	}
	if runner.Title == "" && len(ranked) > 1 {
		runner = ranked[1]
	}

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
	return fmt.Sprintf("Recommended: \"%s\"  —  %s", winner.Title, strings.Join(reasons, "  •  ")) + upscaleSuffix
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

// minBPPPerCodec is the minimum bits-per-pixel-per-frame for genuine content at acceptable quality.
// BPP = bitrate(bps) / (width × height × fps). Below threshold → resolution may be inflated.
var minBPPByCodec = []struct {
	substr    string
	threshold float64
}{
	{"av1", 0.015},
	{"hevc", 0.020},
	{"h265", 0.020},
	{"vp9", 0.025},
	{"vp8", 0.035},
	{"avc", 0.040},
	{"h264", 0.040},
	{"mpeg4", 0.045},
	{"wmv", 0.055},
	{"mpeg2", 0.055},
	{"mpeg1", 0.065},
}

// vqUpscaleRisk returns "low", "medium", or "high" when BPP is suspiciously low for the
// declared resolution+codec, suggesting the video was upscaled from a lower-res source.
// Returns "" when risk is negligible or metadata is incomplete.
// Topaz-enhanced files should skip this check (intentionally upscaled).
func vqUpscaleRisk(v VideoQualityScore) string {
	if v.Bitrate <= 0 || v.FPS <= 0 {
		return ""
	}
	w, h := vqParseResolution(v.Resolution)
	if w <= 0 || h <= 0 {
		return ""
	}
	bpp := float64(v.Bitrate) / (float64(w) * float64(h) * v.FPS)
	threshold := 0.040 // default H.264-like
	lower := strings.ToLower(v.Codec)
	for _, entry := range minBPPByCodec {
		if strings.Contains(lower, entry.substr) {
			threshold = entry.threshold
			break
		}
	}
	ratio := bpp / threshold
	switch {
	case ratio >= 0.80:
		return ""
	case ratio >= 0.40:
		return "low"
	case ratio >= 0.15:
		return "medium"
	default:
		return "high"
	}
}

func vqResolutionScoreFromPixels(pixels float64) float64 {
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

func vqResolutionScore(resolution string) float64 {
	w, h := vqParseResolution(resolution)
	return vqResolutionScoreFromPixels(float64(w * h))
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

// ── Deep frame-sharpness analysis ────────────────────────────────────────────

const deepAnalysisFrames = 8
const deepFrameW = 160
const deepFrameH = 90

// vqDeepAnalyzeFile extracts deepAnalysisFrames evenly-spaced frames from the
// video, computes Laplacian variance per frame, and returns:
//   - sharpness: mean Laplacian variance (higher = genuinely sharper edges)
//   - perceptualEfficiency: sharpness / sqrt(width*height) — cross-resolution comparable
//
// This lets a sharp 480p original outrank a blurry 720p upscale when choosing
// the best source for Topaz Video AI.
func vqDeepAnalyzeFile(filePath string, width, height int, duration float64) (sharpness, efficiency float64, err error) {
	if duration <= 0 || width <= 0 || height <= 0 {
		log.Printf("[DeepAnalysis] skip %q: bad params w=%d h=%d dur=%.1f", filePath, width, height, duration)
		return 0, 0, fmt.Errorf("invalid dimensions or duration")
	}
	log.Printf("[DeepAnalysis] start %q (%dx%d, %.1fs)", filePath, width, height, duration)
	var total float64
	var count int
	for i := 0; i < deepAnalysisFrames; i++ {
		// Sample from 10% → 90% of the video to skip intros/credits
		offset := duration * (0.10 + float64(i)/float64(deepAnalysisFrames)*0.80)
		pixels, ferr := extractGrayFrame(filePath, offset)
		if ferr != nil {
			log.Printf("[DeepAnalysis] frame %d at %.1fs FAILED: %v", i, offset, ferr)
			continue
		}
		total += laplacianVariance(pixels, deepFrameW, deepFrameH)
		count++
	}
	if count == 0 {
		log.Printf("[DeepAnalysis] %q — no frames extracted", filePath)
		return 0, 0, fmt.Errorf("could not extract any frames from %q", filePath)
	}
	sharpness = total / float64(count)
	efficiency = sharpness / math.Sqrt(float64(width*height))
	log.Printf("[DeepAnalysis] %q — %d frames, sharpness=%.1f, efficiency=%.4f", filePath, count, sharpness, efficiency)
	return sharpness, efficiency, nil
}

// extractGrayFrame runs FFmpeg with input-seeking to grab a single grayscale
// frame at offsetSecs, scaled to deepFrameW×deepFrameH, returned as raw bytes.
// Writes to a temp file rather than stdout to avoid pipe-wiring issues on Windows.
func extractGrayFrame(filePath string, offsetSecs float64) ([]byte, error) {
	expected := deepFrameW * deepFrameH

	tmp, err := os.CreateTemp("", "vsai-frame-*.raw")
	if err != nil {
		return nil, fmt.Errorf("create temp frame file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.3f", offsetSecs),
		"-i", filePath,
		"-frames:v", "1",
		"-an",
		"-vf", fmt.Sprintf("scale=%d:%d", deepFrameW, deepFrameH),
		"-f", "rawvideo",
		"-pix_fmt", "gray",
		"-y",
		"-v", "quiet",
		tmpPath,
	)
	cmd.Stderr = io.Discard
	cmd.Run() // exit code ignored — validate by file size

	data, readErr := os.ReadFile(tmpPath)
	if readErr != nil || len(data) < expected {
		return nil, fmt.Errorf("frame extract at %.1fs: got %d bytes, want %d", offsetSecs, len(data), expected)
	}
	return data[:expected], nil
}

// laplacianVariance applies a 3×3 Laplacian kernel to a grayscale frame and
// returns the variance of the response. Higher values indicate more genuine
// edge detail (sharper image); blurry or upscaled content scores near zero.
func laplacianVariance(pixels []byte, w, h int) float64 {
	if len(pixels) < w*h || w < 3 || h < 3 {
		return 0
	}
	responses := make([]float64, 0, (w-2)*(h-2))
	var sum float64
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			idx := y*w + x
			v := 8.0*float64(pixels[idx]) -
				float64(pixels[idx-w-1]) - float64(pixels[idx-w]) - float64(pixels[idx-w+1]) -
				float64(pixels[idx-1]) - float64(pixels[idx+1]) -
				float64(pixels[idx+w-1]) - float64(pixels[idx+w]) - float64(pixels[idx+w+1])
			responses = append(responses, v)
			sum += v
		}
	}
	n := float64(len(responses))
	if n == 0 {
		return 0
	}
	mean := sum / n
	var variance float64
	for _, v := range responses {
		d := v - mean
		variance += d * d
	}
	return variance / n
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
