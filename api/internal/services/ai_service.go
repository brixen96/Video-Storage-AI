package services

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

type AIService struct {
	db *sql.DB
}

func NewAIService() *AIService {
	return &AIService{
		db: database.DB,
	}
}

// PerformerMatch represents a potential match between a video and performer
type PerformerMatch struct {
	VideoID      int64   `json:"video_id"`
	VideoTitle   string  `json:"video_title"`
	PerformerID  int64   `json:"performer_id"`
	PerformerName string `json:"performer_name"`
	Confidence   float64 `json:"confidence"`
	MatchType    string  `json:"match_type"` // "exact", "partial", "fuzzy"
}

// PerformerLinkSuggestion represents suggestions for a single video
type PerformerLinkSuggestion struct {
	VideoID    int64             `json:"video_id"`
	VideoTitle string            `json:"video_title"`
	FilePath   string            `json:"file_path"`
	Matches    []PerformerMatch  `json:"matches"`
}

// CommonPrefix represents studio/site prefixes to ignore
var commonPrefixes = []string{
	"Brazzers", "RealityKings", "Mofos", "TeamSkeet", "BangBros",
	"Naughty America", "Digital Playground", "Evil Angel",
	"Inside", "Tushy", "Blacked", "Vixen", "Deeper",
}

// AutoLinkPerformers analyzes video filenames and suggests performer links
func (s *AIService) AutoLinkPerformers(videoIDs []int64, autoApply bool) ([]PerformerLinkSuggestion, error) {
	log.Printf("Starting auto-link analysis for %d videos (auto-apply: %v)", len(videoIDs), autoApply)

	// Get all performers
	performers, err := s.getAllPerformers()
	if err != nil {
		return nil, fmt.Errorf("failed to get performers: %w", err)
	}

	if len(performers) == 0 {
		return []PerformerLinkSuggestion{}, nil
	}

	// Get videos to analyze
	videos, err := s.getVideosForAnalysis(videoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	suggestions := []PerformerLinkSuggestion{}

	for _, video := range videos {
		matches := s.findPerformerMatches(video, performers)

		if len(matches) > 0 {
			suggestion := PerformerLinkSuggestion{
				VideoID:    video.ID,
				VideoTitle: video.Title,
				FilePath:   video.FilePath,
				Matches:    matches,
			}
			suggestions = append(suggestions, suggestion)

			// Auto-apply if requested and confidence is high
			if autoApply {
				s.applyHighConfidenceMatches(video.ID, matches)
			}
		}
	}

	log.Printf("Found %d videos with performer matches", len(suggestions))
	return suggestions, nil
}

// findPerformerMatches finds all potential performer matches for a video
func (s *AIService) findPerformerMatches(video models.Video, performers []models.Performer) []PerformerMatch {
	// Extract filename without extension
	filename := filepath.Base(video.FilePath)
	filenameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Clean the filename
	cleanedName := s.cleanFilename(filenameNoExt)

	matches := []PerformerMatch{}

	for _, performer := range performers {
		confidence, matchType := s.calculateMatch(cleanedName, performer.Name)

		if confidence >= 0.6 { // Minimum 60% confidence
			match := PerformerMatch{
				VideoID:       video.ID,
				VideoTitle:    video.Title,
				PerformerID:   performer.ID,
				PerformerName: performer.Name,
				Confidence:    confidence,
				MatchType:     matchType,
			}
			matches = append(matches, match)
		}
	}

	return matches
}

// cleanFilename removes common prefixes and cleans up the filename
func (s *AIService) cleanFilename(filename string) string {
	// Remove common prefixes
	for _, prefix := range commonPrefixes {
		// Case-insensitive removal
		re := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(prefix) + `\s*[-–—]\s*`)
		filename = re.ReplaceAllString(filename, "")
	}

	// Remove common patterns like dates, resolution, etc.
	patterns := []string{
		`\d{4}[-._]\d{2}[-._]\d{2}`, // Dates: 2024-01-15
		`\d{3,4}p`,                   // Resolution: 1080p, 720p
		`[hH][dD]`,                   // HD
		`[xX]264`,                    // x264
		`[hH]265`,                    // h265
		`HEVC`,                       // HEVC
		`\[.*?\]`,                    // Anything in brackets
		`\(.*?\)`,                    // Anything in parentheses
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		filename = re.ReplaceAllString(filename, " ")
	}

	// Normalize spacing and special characters
	filename = strings.ReplaceAll(filename, "_", " ")
	filename = strings.ReplaceAll(filename, ".", " ")
	filename = regexp.MustCompile(`\s+`).ReplaceAllString(filename, " ")
	filename = strings.TrimSpace(filename)

	return filename
}

// calculateMatch calculates the confidence score for a performer match
func (s *AIService) calculateMatch(filename, performerName string) (float64, string) {
	filenameLower := strings.ToLower(filename)
	performerLower := strings.ToLower(performerName)

	// Exact match (case-insensitive)
	if strings.Contains(filenameLower, performerLower) {
		return 1.0, "exact"
	}

	// Split performer name into parts
	nameParts := strings.Fields(performerName)
	if len(nameParts) < 2 {
		return 0.0, "none"
	}

	firstName := strings.ToLower(nameParts[0])
	lastName := strings.ToLower(nameParts[len(nameParts)-1])

	// Check for first + last name match
	hasFirst := strings.Contains(filenameLower, firstName)
	hasLast := strings.Contains(filenameLower, lastName)

	if hasFirst && hasLast {
		// Both names present - high confidence
		return 0.95, "partial"
	}

	if hasLast {
		// Last name only - medium confidence
		// Check if it's a reasonably unique last name (length > 4)
		if len(lastName) > 4 {
			return 0.75, "partial"
		}
		return 0.65, "partial"
	}

	if hasFirst && len(firstName) > 4 {
		// First name only with decent length
		return 0.70, "partial"
	}

	// Fuzzy match - check for similar words
	for _, part := range nameParts {
		partLower := strings.ToLower(part)
		if len(part) > 5 && strings.Contains(filenameLower, partLower) {
			return 0.65, "fuzzy"
		}
	}

	return 0.0, "none"
}

// applyHighConfidenceMatches automatically applies matches with high confidence
func (s *AIService) applyHighConfidenceMatches(videoID int64, matches []PerformerMatch) {
	for _, match := range matches {
		if match.Confidence >= 0.90 {
			// Check if link already exists
			exists, err := s.performerLinkExists(videoID, match.PerformerID)
			if err != nil || exists {
				continue
			}

			// Create the link
			err = s.linkPerformerToVideo(videoID, match.PerformerID)
			if err != nil {
				log.Printf("Failed to auto-link performer %d to video %d: %v", match.PerformerID, videoID, err)
			} else {
				log.Printf("Auto-linked performer '%s' to video %d (confidence: %.2f)", match.PerformerName, videoID, match.Confidence)
			}
		}
	}
}

// ApplySuggestions applies selected performer links
func (s *AIService) ApplySuggestions(suggestions []PerformerMatch) error {
	for _, match := range suggestions {
		// Check if link already exists
		exists, err := s.performerLinkExists(match.VideoID, match.PerformerID)
		if err != nil {
			return fmt.Errorf("failed to check existing link: %w", err)
		}
		if exists {
			continue
		}

		// Create the link
		err = s.linkPerformerToVideo(match.VideoID, match.PerformerID)
		if err != nil {
			return fmt.Errorf("failed to link performer %d to video %d: %w", match.PerformerID, match.VideoID, err)
		}
	}

	return nil
}

// Helper functions

func (s *AIService) getAllPerformers() ([]models.Performer, error) {
	query := `SELECT id, name FROM performers ORDER BY name`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	performers := []models.Performer{}
	for rows.Next() {
		var p models.Performer
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		performers = append(performers, p)
	}

	return performers, nil
}

func (s *AIService) getVideosForAnalysis(videoIDs []int64) ([]models.Video, error) {
	var query string
	var rows *sql.Rows
	var err error

	if len(videoIDs) > 0 {
		// Specific videos
		query = `SELECT id, title, file_path FROM videos WHERE id IN (?` + strings.Repeat(",?", len(videoIDs)-1) + `)`
		args := make([]interface{}, len(videoIDs))
		for i, id := range videoIDs {
			args[i] = id
		}
		rows, err = s.db.Query(query, args...)
	} else {
		// All videos
		query = `SELECT id, title, file_path FROM videos ORDER BY id`
		rows, err = s.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	videos := []models.Video{}
	for rows.Next() {
		var v models.Video
		if err := rows.Scan(&v.ID, &v.Title, &v.FilePath); err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}

	return videos, nil
}

func (s *AIService) performerLinkExists(videoID, performerID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM video_performers WHERE video_id = ? AND performer_id = ?`
	var count int
	err := s.db.QueryRow(query, videoID, performerID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *AIService) linkPerformerToVideo(videoID, performerID int64) error {
	query := `INSERT INTO video_performers (video_id, performer_id) VALUES (?, ?)`
	_, err := s.db.Exec(query, videoID, performerID)
	return err
}

// ================== Scene Detection ==================

// SceneDetectionResult represents detected scenes in a video
type SceneDetectionResult struct {
	VideoID    int64         `json:"video_id"`
	VideoTitle string        `json:"video_title"`
	Scenes     []VideoScene  `json:"scenes"`
	TotalScenes int          `json:"total_scenes"`
}

type VideoScene struct {
	StartTime   float64 `json:"start_time"`   // in seconds
	EndTime     float64 `json:"end_time"`     // in seconds
	Duration    float64 `json:"duration"`     // in seconds
	SceneType   string  `json:"scene_type"`   // "intro", "main", "outro", "credits"
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
}

// DetectScenes analyzes videos and detects scene boundaries
func (s *AIService) DetectScenes(videoIDs []int64) ([]SceneDetectionResult, error) {
	log.Printf("Starting scene detection for %d videos", len(videoIDs))

	videos, err := s.getVideosForAnalysis(videoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	results := []SceneDetectionResult{}
	for _, video := range videos {
		// Simulate scene detection (in production, this would use FFmpeg or ML models)
		scenes := s.analyzeVideoScenes(video)

		result := SceneDetectionResult{
			VideoID:     video.ID,
			VideoTitle:  video.Title,
			Scenes:      scenes,
			TotalScenes: len(scenes),
		}
		results = append(results, result)
	}

	log.Printf("Detected scenes in %d videos", len(results))
	return results, nil
}

func (s *AIService) analyzeVideoScenes(video models.Video) []VideoScene {
	// Simulated scene detection logic
	// In production, this would analyze video frames, audio, and content changes
	duration := 1800.0 // Default 30 minutes if not available

	scenes := []VideoScene{
		{
			StartTime:   0,
			EndTime:     30,
			Duration:    30,
			SceneType:   "intro",
			Confidence:  0.85,
			Description: "Opening/branding sequence",
		},
		{
			StartTime:   30,
			EndTime:     duration - 60,
			Duration:    duration - 90,
			SceneType:   "main",
			Confidence:  0.95,
			Description: "Main content",
		},
		{
			StartTime:   duration - 60,
			EndTime:     duration,
			Duration:    60,
			SceneType:   "outro",
			Confidence:  0.80,
			Description: "Ending/credits",
		},
	}

	return scenes
}

// ================== Content Classification ==================

// ContentClassification represents the classification of video content
type ContentClassification struct {
	VideoID    int64               `json:"video_id"`
	VideoTitle string              `json:"video_title"`
	Categories []ContentCategory   `json:"categories"`
	MainCategory string            `json:"main_category"`
}

type ContentCategory struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Tags       []string `json:"tags"`
}

// ClassifyContent analyzes and categorizes video content
func (s *AIService) ClassifyContent(videoIDs []int64) ([]ContentClassification, error) {
	log.Printf("Starting content classification for %d videos", len(videoIDs))

	videos, err := s.getVideosForAnalysis(videoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	results := []ContentClassification{}
	for _, video := range videos {
		classification := s.classifyVideoContent(video)
		results = append(results, classification)
	}

	log.Printf("Classified %d videos", len(results))
	return results, nil
}

func (s *AIService) classifyVideoContent(video models.Video) ContentClassification {
	// Analyze filename and existing tags for classification
	filename := strings.ToLower(video.Title + " " + video.FilePath)

	categories := []ContentCategory{}

	// Check for various categories based on filename analysis
	if strings.Contains(filename, "amateur") || strings.Contains(filename, "homemade") {
		categories = append(categories, ContentCategory{
			Name:       "Amateur",
			Confidence: 0.85,
			Tags:       []string{"amateur", "homemade", "real"},
		})
	}

	if strings.Contains(filename, "professional") || strings.Contains(filename, "studio") {
		categories = append(categories, ContentCategory{
			Name:       "Professional",
			Confidence: 0.90,
			Tags:       []string{"professional", "studio", "high-quality"},
		})
	}

	if strings.Contains(filename, "4k") || strings.Contains(filename, "uhd") || strings.Contains(filename, "2160p") {
		categories = append(categories, ContentCategory{
			Name:       "Ultra HD",
			Confidence: 0.95,
			Tags:       []string{"4k", "uhd", "high-resolution"},
		})
	}

	mainCategory := "Uncategorized"
	if len(categories) > 0 {
		mainCategory = categories[0].Name
	}

	return ContentClassification{
		VideoID:      video.ID,
		VideoTitle:   video.Title,
		Categories:   categories,
		MainCategory: mainCategory,
	}
}

// ================== Quality Analysis ==================

// QualityAnalysis represents video quality metrics
type QualityAnalysis struct {
	VideoID      int64   `json:"video_id"`
	VideoTitle   string  `json:"video_title"`
	Resolution   string  `json:"resolution"`    // "1080p", "720p", etc.
	Bitrate      int     `json:"bitrate"`       // in kbps
	FrameRate    float64 `json:"frame_rate"`    // fps
	Codec        string  `json:"codec"`         // "h264", "h265", etc.
	FileSize     int64   `json:"file_size"`     // in bytes
	Duration     float64 `json:"duration"`      // in seconds
	QualityScore float64 `json:"quality_score"` // 0-100
	Issues       []string `json:"issues"`
}

// AnalyzeQuality analyzes video technical quality
func (s *AIService) AnalyzeQuality(videoIDs []int64) ([]QualityAnalysis, error) {
	log.Printf("Starting quality analysis for %d videos", len(videoIDs))

	videos, err := s.getVideosForAnalysis(videoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	results := []QualityAnalysis{}
	for _, video := range videos {
		analysis := s.analyzeVideoQuality(video)
		results = append(results, analysis)
	}

	log.Printf("Analyzed quality of %d videos", len(results))
	return results, nil
}

func (s *AIService) analyzeVideoQuality(video models.Video) QualityAnalysis {
	// In production, this would use FFprobe to get actual video metadata
	// For now, we'll simulate based on filename patterns
	filename := strings.ToLower(video.FilePath)

	analysis := QualityAnalysis{
		VideoID:    video.ID,
		VideoTitle: video.Title,
		Resolution: "1080p",
		Bitrate:    5000,
		FrameRate:  30.0,
		Codec:      "h264",
		FileSize:   video.FileSize,
		Duration:   1800.0,
		Issues:     []string{},
	}

	// Detect resolution from filename
	if strings.Contains(filename, "4k") || strings.Contains(filename, "2160p") {
		analysis.Resolution = "2160p"
		analysis.QualityScore = 95
	} else if strings.Contains(filename, "1080p") {
		analysis.Resolution = "1080p"
		analysis.QualityScore = 85
	} else if strings.Contains(filename, "720p") {
		analysis.Resolution = "720p"
		analysis.QualityScore = 70
	} else if strings.Contains(filename, "480p") {
		analysis.Resolution = "480p"
		analysis.QualityScore = 50
		analysis.Issues = append(analysis.Issues, "Low resolution")
	}

	// Check for codec indicators
	if strings.Contains(filename, "h265") || strings.Contains(filename, "hevc") {
		analysis.Codec = "h265"
		analysis.QualityScore += 5
	}

	// Check file size relative to duration
	if video.FileSize > 0 && video.FileSize < 100*1024*1024 { // Less than 100MB for a full video
		analysis.Issues = append(analysis.Issues, "Unusually small file size for quality")
		analysis.QualityScore -= 10
	}

	return analysis
}

// ================== Missing Metadata Detection ==================

// MissingMetadata represents videos missing important metadata
type MissingMetadata struct {
	VideoID       int64    `json:"video_id"`
	VideoTitle    string   `json:"video_title"`
	MissingFields []string `json:"missing_fields"`
	Severity      string   `json:"severity"` // "high", "medium", "low"
	Suggestions   []string `json:"suggestions"`
}

// DetectMissingMetadata finds videos with incomplete metadata
func (s *AIService) DetectMissingMetadata(videoIDs []int64) ([]MissingMetadata, error) {
	log.Printf("Starting missing metadata detection for %d videos", len(videoIDs))

	query := `
		SELECT v.id, v.title, v.file_path,
			   (SELECT COUNT(*) FROM video_performers vp WHERE vp.video_id = v.id) as performer_count,
			   (SELECT COUNT(*) FROM video_tags vt WHERE vt.video_id = v.id) as tag_count,
			   (SELECT COUNT(*) FROM video_studios vs WHERE vs.video_id = v.id) as studio_count
		FROM videos v
	`

	if len(videoIDs) > 0 {
		placeholders := make([]string, len(videoIDs))
		args := make([]interface{}, len(videoIDs))
		for i, id := range videoIDs {
			placeholders[i] = "?"
			args[i] = id
		}
		query += " WHERE v.id IN (" + strings.Join(placeholders, ",") + ")"

		rows, err := s.db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return s.processMetadataResults(rows)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.processMetadataResults(rows)
}

func (s *AIService) processMetadataResults(rows *sql.Rows) ([]MissingMetadata, error) {
	results := []MissingMetadata{}

	for rows.Next() {
		var id int64
		var title, filePath string
		var performerCount, tagCount, studioCount int

		if err := rows.Scan(&id, &title, &filePath, &performerCount, &tagCount, &studioCount); err != nil {
			return nil, err
		}

		missingFields := []string{}
		suggestions := []string{}

		if performerCount == 0 {
			missingFields = append(missingFields, "performers")
			suggestions = append(suggestions, "Use Auto-Link Performers feature")
		}

		if tagCount == 0 {
			missingFields = append(missingFields, "tags")
			suggestions = append(suggestions, "Use Smart Tagging feature")
		}

		if studioCount == 0 {
			missingFields = append(missingFields, "studio")
			suggestions = append(suggestions, "Manually assign studio")
		}

		// Only include if something is missing
		if len(missingFields) > 0 {
			severity := "low"
			if len(missingFields) >= 3 {
				severity = "high"
			} else if len(missingFields) == 2 {
				severity = "medium"
			}

			results = append(results, MissingMetadata{
				VideoID:       id,
				VideoTitle:    title,
				MissingFields: missingFields,
				Severity:      severity,
				Suggestions:   suggestions,
			})
		}
	}

	log.Printf("Found %d videos with missing metadata", len(results))
	return results, nil
}

// ================== Duplicate Detection ==================

// DuplicateGroup represents a group of potentially duplicate videos
type DuplicateGroup struct {
	GroupID    int              `json:"group_id"`
	Videos     []DuplicateVideo `json:"videos"`
	Similarity float64          `json:"similarity"` // 0-1
	Reason     string           `json:"reason"`     // "exact_match", "similar_name", "same_size"
}

type DuplicateVideo struct {
	VideoID    int64  `json:"video_id"`
	VideoTitle string `json:"video_title"`
	FilePath   string `json:"file_path"`
	FileSize   int64  `json:"file_size"`
	Duration   float64 `json:"duration"`
}

// DetectDuplicates finds duplicate or similar videos
func (s *AIService) DetectDuplicates(videoIDs []int64) ([]DuplicateGroup, error) {
	log.Printf("Starting duplicate detection for %d videos", len(videoIDs))

	videos, err := s.getVideosWithMetadata(videoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	groups := []DuplicateGroup{}
	groupID := 1
	processed := make(map[int64]bool)

	for i := 0; i < len(videos); i++ {
		if processed[videos[i].ID] {
			continue
		}

		duplicates := []DuplicateVideo{
			{
				VideoID:    videos[i].ID,
				VideoTitle: videos[i].Title,
				FilePath:   videos[i].FilePath,
				FileSize:   videos[i].FileSize,
			},
		}

		// Track the highest similarity and reason for this group
		highestSimilarity := 0.0
		groupReason := "no_match"

		// Find similar videos
		for j := i + 1; j < len(videos); j++ {
			if processed[videos[j].ID] {
				continue
			}

			similarity, reason := s.calculateSimilarity(videos[i], videos[j])
			// Increased threshold to reduce false positives
			if similarity >= 0.95 {
				duplicates = append(duplicates, DuplicateVideo{
					VideoID:    videos[j].ID,
					VideoTitle: videos[j].Title,
					FilePath:   videos[j].FilePath,
					FileSize:   videos[j].FileSize,
				})
				processed[videos[j].ID] = true

				// Track the highest similarity for this group
				if similarity > highestSimilarity {
					highestSimilarity = similarity
					groupReason = reason
				}
			}
		}

		if len(duplicates) > 1 {
			groups = append(groups, DuplicateGroup{
				GroupID:    groupID,
				Videos:     duplicates,
				Similarity: highestSimilarity,
				Reason:     groupReason,
			})
			groupID++
		}

		processed[videos[i].ID] = true
	}

	log.Printf("Found %d duplicate groups", len(groups))
	return groups, nil
}

func (s *AIService) getVideosWithMetadata(videoIDs []int64) ([]models.Video, error) {
	query := `SELECT id, title, file_path, file_size FROM videos`

	if len(videoIDs) > 0 {
		placeholders := make([]string, len(videoIDs))
		args := make([]interface{}, len(videoIDs))
		for i, id := range videoIDs {
			placeholders[i] = "?"
			args[i] = id
		}
		query += " WHERE id IN (" + strings.Join(placeholders, ",") + ")"

		rows, err := s.db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return s.scanVideoRows(rows)
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanVideoRows(rows)
}

func (s *AIService) scanVideoRows(rows *sql.Rows) ([]models.Video, error) {
	videos := []models.Video{}
	for rows.Next() {
		var v models.Video
		if err := rows.Scan(&v.ID, &v.Title, &v.FilePath, &v.FileSize); err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, nil
}

func (s *AIService) calculateSimilarity(v1, v2 models.Video) (float64, string) {
	// Strict duplicate detection - only finds actual duplicates, not similar videos
	title1 := strings.ToLower(v1.Title)
	title2 := strings.ToLower(v2.Title)

	// Remove common file extensions for better comparison
	title1 = strings.TrimSuffix(title1, ".mp4")
	title1 = strings.TrimSuffix(title1, ".mkv")
	title1 = strings.TrimSuffix(title1, ".avi")
	title1 = strings.TrimSuffix(title1, ".mov")
	title2 = strings.TrimSuffix(title2, ".mp4")
	title2 = strings.TrimSuffix(title2, ".mkv")
	title2 = strings.TrimSuffix(title2, ".avi")
	title2 = strings.TrimSuffix(title2, ".mov")

	// Exact title match (after removing extensions)
	if title1 == title2 {
		return 1.0, "exact_match"
	}

	// Remove common prefixes like "converted -", "converted-", etc.
	title1Clean := strings.TrimPrefix(title1, "converted - ")
	title1Clean = strings.TrimPrefix(title1Clean, "converted-")
	title2Clean := strings.TrimPrefix(title2, "converted - ")
	title2Clean = strings.TrimPrefix(title2Clean, "converted-")

	// Check if titles are very similar after removing common patterns
	// This catches things like "video_final.mp4" and "video_final (1).mp4"
	similarity := calculateLevenshteinSimilarity(title1Clean, title2Clean)

	// Very high title similarity (95%+) = likely duplicate or version
	if similarity >= 0.95 {
		// Also check file size to confirm
		if v1.FileSize > 0 && v2.FileSize > 0 {
			sizeDiff := float64(abs(v1.FileSize-v2.FileSize)) / float64(max(v1.FileSize, v2.FileSize))
			// Same size within 5% = likely same video, different encode
			if sizeDiff < 0.05 {
				return 0.99, "duplicate_version"
			}
			// Very similar name but different size = probably different videos
			return 0.90, "similar_name_different_size"
		}
		return 0.95, "similar_name"
	}

	// Exact file size (within 1KB) with some title similarity
	if v1.FileSize > 0 && v2.FileSize > 0 {
		absoluteDiff := abs(v1.FileSize - v2.FileSize)
		// Exact same size (within 1KB) for any file
		if absoluteDiff <= 1024 {
			// Require at least 50% title similarity to avoid random matches
			if similarity >= 0.50 {
				return 0.97, "same_size_similar_name"
			}
		}
	}

	return 0.0, "no_match"
}

// calculateLevenshteinSimilarity calculates similarity between two strings using Levenshtein distance
func calculateLevenshteinSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Use Levenshtein distance for better string matching
	distance := levenshteinDistance(s1, s2)
	maxLen := max(int64(len(s1)), int64(len(s2)))

	return 1.0 - (float64(distance) / float64(maxLen))
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = minInt(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func minInt(a, b, c int) int {
	min := a
	if b < min {
		min = b
	}
	if c < min {
		min = c
	}
	return min
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ================== Auto-Naming ==================

// NamingSuggestion represents a suggested filename for a video
type NamingSuggestion struct {
	VideoID         int64  `json:"video_id"`
	CurrentName     string `json:"current_name"`
	SuggestedName   string `json:"suggested_name"`
	Confidence      float64 `json:"confidence"`
	Reason          string `json:"reason"`
	PerformerNames  []string `json:"performer_names"`
	StudioName      string `json:"studio_name"`
}

// SuggestNaming suggests better filenames for videos
func (s *AIService) SuggestNaming(videoIDs []int64) ([]NamingSuggestion, error) {
	log.Printf("Starting naming suggestions for %d videos", len(videoIDs))

	query := `
		SELECT v.id, v.title, v.file_path,
			   GROUP_CONCAT(DISTINCT p.name) as performers,
			   GROUP_CONCAT(DISTINCT st.name) as studio
		FROM videos v
		LEFT JOIN video_performers vp ON v.id = vp.video_id
		LEFT JOIN performers p ON vp.performer_id = p.id
		LEFT JOIN video_studios vs ON v.id = vs.video_id
		LEFT JOIN studios st ON vs.studio_id = st.id
	`

	if len(videoIDs) > 0 {
		placeholders := make([]string, len(videoIDs))
		args := make([]interface{}, len(videoIDs))
		for i, id := range videoIDs {
			placeholders[i] = "?"
			args[i] = id
		}
		query += " WHERE v.id IN (" + strings.Join(placeholders, ",") + ") GROUP BY v.id"

		return s.processNamingSuggestions(query, args...)
	}

	query += " GROUP BY v.id"
	return s.processNamingSuggestions(query)
}

func (s *AIService) processNamingSuggestions(query string, args ...interface{}) ([]NamingSuggestion, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	suggestions := []NamingSuggestion{}

	for rows.Next() {
		var id int64
		var title, filePath string
		var performers, studio sql.NullString

		if err := rows.Scan(&id, &title, &filePath, &performers, &studio); err != nil {
			return nil, err
		}

		performerList := []string{}
		if performers.Valid && performers.String != "" {
			performerList = strings.Split(performers.String, ",")
		}

		studioName := ""
		if studio.Valid {
			studioName = studio.String
		}

		// Generate suggested name
		suggestedName := s.generateSuggestedName(title, performerList, studioName)

		// Only suggest if it's different and better
		if suggestedName != title && len(suggestedName) > 0 {
			suggestions = append(suggestions, NamingSuggestion{
				VideoID:        id,
				CurrentName:    title,
				SuggestedName:  suggestedName,
				Confidence:     0.85,
				Reason:         "Better structure with metadata",
				PerformerNames: performerList,
				StudioName:     studioName,
			})
		}
	}

	log.Printf("Generated %d naming suggestions", len(suggestions))
	return suggestions, nil
}

func (s *AIService) generateSuggestedName(currentTitle string, performers []string, studio string) string {
	parts := []string{}

	// Add studio if available
	if studio != "" {
		parts = append(parts, studio)
	}

	// Add performers (limit to 3)
	if len(performers) > 0 {
		maxPerformers := 3
		if len(performers) < maxPerformers {
			maxPerformers = len(performers)
		}
		performerPart := strings.Join(performers[:maxPerformers], ", ")
		if len(performers) > maxPerformers {
			performerPart += " & more"
		}
		parts = append(parts, performerPart)
	}

	// Add cleaned current title (remove common junk)
	cleanTitle := s.cleanTitle(currentTitle)
	if cleanTitle != "" {
		parts = append(parts, cleanTitle)
	}

	return strings.Join(parts, " - ")
}

func (s *AIService) cleanTitle(title string) string {
	// Remove common junk patterns
	patterns := []string{
		`\[.*?\]`,           // Remove bracketed text
		`\(.*?\)`,           // Remove parenthesized text
		`\d{3,4}p`,          // Remove resolution
		`[xh]\.?264`,        // Remove codec
		`[xh]\.?265`,
		`hevc`,
		`mp4|mkv|avi|wmv`,   // Remove extensions
		`[-_.]+`,            // Collapse multiple separators
	}

	cleaned := title
	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?i)` + pattern)
		cleaned = re.ReplaceAllString(cleaned, " ")
	}

	// Clean up whitespace
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimSpace(cleaned)
}

// ================== Library Analytics ==================

// LibraryStats represents comprehensive library statistics
type LibraryStats struct {
	TotalVideos       int            `json:"total_videos"`
	TotalSize         int64          `json:"total_size"`
	TotalDuration     float64        `json:"total_duration"`
	TotalPerformers   int            `json:"total_performers"`
	TotalStudios      int            `json:"total_studios"`
	TotalTags         int            `json:"total_tags"`
	TopPerformers     []PerformerStat `json:"top_performers"`
	TopStudios        []StudioStat    `json:"top_studios"`
	TopTags           []TagStat       `json:"top_tags"`
	QualityDistribution map[string]int `json:"quality_distribution"`
	MissingMetadataCount int          `json:"missing_metadata_count"`
	Recommendations   []string        `json:"recommendations"`
}

type PerformerStat struct {
	Name       string `json:"name"`
	VideoCount int    `json:"video_count"`
}

type StudioStat struct {
	Name       string `json:"name"`
	VideoCount int    `json:"video_count"`
}

type TagStat struct {
	Name       string `json:"name"`
	VideoCount int    `json:"video_count"`
}

// GetLibraryAnalytics provides comprehensive library statistics and insights
func (s *AIService) GetLibraryAnalytics() (*LibraryStats, error) {
	log.Printf("Generating library analytics")

	stats := &LibraryStats{
		QualityDistribution: make(map[string]int),
		Recommendations:     []string{},
	}

	// Get basic counts
	if err := s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&stats.TotalVideos); err != nil {
		return nil, err
	}

	if err := s.db.QueryRow("SELECT COUNT(DISTINCT id) FROM performers").Scan(&stats.TotalPerformers); err != nil {
		return nil, err
	}

	if err := s.db.QueryRow("SELECT COUNT(DISTINCT id) FROM studios").Scan(&stats.TotalStudios); err != nil {
		return nil, err
	}

	if err := s.db.QueryRow("SELECT COUNT(DISTINCT id) FROM tags").Scan(&stats.TotalTags); err != nil {
		return nil, err
	}

	// Get total size
	s.db.QueryRow("SELECT COALESCE(SUM(size), 0) FROM videos").Scan(&stats.TotalSize)

	// Get top performers
	stats.TopPerformers = s.getTopPerformers(10)

	// Get top studios
	stats.TopStudios = s.getTopStudios(10)

	// Get top tags
	stats.TopTags = s.getTopTags(10)

	// Quality distribution (simulated)
	stats.QualityDistribution["4K"] = stats.TotalVideos / 10
	stats.QualityDistribution["1080p"] = stats.TotalVideos / 2
	stats.QualityDistribution["720p"] = stats.TotalVideos / 3
	stats.QualityDistribution["Other"] = stats.TotalVideos / 10

	// Generate recommendations
	stats.Recommendations = s.generateRecommendations(stats)

	log.Printf("Library analytics generated successfully")
	return stats, nil
}

func (s *AIService) getTopPerformers(limit int) []PerformerStat {
	query := `
		SELECT p.name, COUNT(vp.video_id) as count
		FROM performers p
		LEFT JOIN video_performers vp ON p.id = vp.performer_id
		GROUP BY p.id
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return []PerformerStat{}
	}
	defer rows.Close()

	stats := []PerformerStat{}
	for rows.Next() {
		var stat PerformerStat
		if err := rows.Scan(&stat.Name, &stat.VideoCount); err == nil {
			stats = append(stats, stat)
		}
	}

	return stats
}

func (s *AIService) getTopStudios(limit int) []StudioStat {
	query := `
		SELECT st.name, COUNT(vs.video_id) as count
		FROM studios st
		LEFT JOIN video_studios vs ON st.id = vs.studio_id
		GROUP BY st.id
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return []StudioStat{}
	}
	defer rows.Close()

	stats := []StudioStat{}
	for rows.Next() {
		var stat StudioStat
		if err := rows.Scan(&stat.Name, &stat.VideoCount); err == nil {
			stats = append(stats, stat)
		}
	}

	return stats
}

func (s *AIService) getTopTags(limit int) []TagStat {
	query := `
		SELECT t.name, COUNT(vt.video_id) as count
		FROM tags t
		LEFT JOIN video_tags vt ON t.id = vt.tag_id
		GROUP BY t.id
		ORDER BY count DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return []TagStat{}
	}
	defer rows.Close()

	stats := []TagStat{}
	for rows.Next() {
		var stat TagStat
		if err := rows.Scan(&stat.Name, &stat.VideoCount); err == nil {
			stats = append(stats, stat)
		}
	}

	return stats
}

func (s *AIService) generateRecommendations(stats *LibraryStats) []string {
	recommendations := []string{}

	if stats.TotalVideos > 100 && stats.TotalTags < 10 {
		recommendations = append(recommendations, "Consider using Smart Tagging to better organize your library")
	}

	if stats.TotalVideos > 50 && stats.TotalPerformers < 10 {
		recommendations = append(recommendations, "Use Auto-Link Performers to identify performers in your videos")
	}

	if stats.TotalSize > 1024*1024*1024*100 { // > 100GB
		recommendations = append(recommendations, "Your library is large. Consider using Quality Analysis to identify low-quality duplicates")
	}

	return recommendations
}

// ================== Thumbnail Quality Analysis ==================

// ThumbnailQuality represents thumbnail quality assessment
type ThumbnailQuality struct {
	VideoID       int64    `json:"video_id"`
	VideoTitle    string   `json:"video_title"`
	HasThumbnail  bool     `json:"has_thumbnail"`
	QualityScore  float64  `json:"quality_score"` // 0-100
	Issues        []string `json:"issues"`
	Suggestions   []string `json:"suggestions"`
	BestTimestamp float64  `json:"best_timestamp"` // Suggested time for better thumbnail
}

// AnalyzeThumbnailQuality evaluates thumbnail quality and suggests improvements
func (s *AIService) AnalyzeThumbnailQuality(videoIDs []int64) ([]ThumbnailQuality, error) {
	log.Printf("Starting thumbnail quality analysis for %d videos", len(videoIDs))

	videos, err := s.getVideosForAnalysis(videoIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	results := []ThumbnailQuality{}
	for _, video := range videos {
		analysis := s.analyzeThumbnail(video)
		results = append(results, analysis)
	}

	log.Printf("Analyzed thumbnails for %d videos", len(results))
	return results, nil
}

func (s *AIService) analyzeThumbnail(video models.Video) ThumbnailQuality {
	// Check if thumbnail exists
	hasThumbnail := video.ThumbnailPath != ""

	analysis := ThumbnailQuality{
		VideoID:      video.ID,
		VideoTitle:   video.Title,
		HasThumbnail: hasThumbnail,
		QualityScore: 0,
		Issues:       []string{},
		Suggestions:  []string{},
		BestTimestamp: 300.0, // Suggest 5 minutes in as default
	}

	if !hasThumbnail {
		analysis.Issues = append(analysis.Issues, "No thumbnail generated")
		analysis.Suggestions = append(analysis.Suggestions, "Generate thumbnail from video")
		analysis.QualityScore = 0
		return analysis
	}

	// Simulated quality assessment
	analysis.QualityScore = 75.0

	// Suggest improvements
	analysis.Suggestions = append(analysis.Suggestions, "Consider generating thumbnail from a more interesting scene")
	analysis.BestTimestamp = 180.0 // 3 minutes in

	return analysis
}
