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
