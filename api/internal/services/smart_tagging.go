package services

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// TagSuggestion represents a suggested tag for a video
type TagSuggestion struct {
	TagID      int64   `json:"tag_id"`
	TagName    string  `json:"tag_name"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"` // Why this tag was suggested
}

// VideoTagSuggestion represents tag suggestions for a single video
type VideoTagSuggestion struct {
	VideoID     int64           `json:"video_id"`
	VideoTitle  string          `json:"video_title"`
	FilePath    string          `json:"file_path"`
	Suggestions []TagSuggestion `json:"suggestions"`
}

// Common tag keywords and their associated tags
var tagKeywords = map[string][]string{
	// Positions
	"doggy":          {"Doggy Style", "Doggystyle"},
	"missionary":     {"Missionary"},
	"cowgirl":        {"Cowgirl", "Reverse Cowgirl"},
	"reverse cowgirl": {"Reverse Cowgirl"},
	"69":             {"69"},
	"standing":       {"Standing"},

	// Acts
	"anal":               {"Anal"},
	"dp":                 {"Double Penetration", "DP"},
	"double penetration": {"Double Penetration", "DP"},
	"oral":               {"Oral", "Blowjob"},
	"bj":                 {"Blowjob"},
	"blowjob":            {"Blowjob"},
	"deepthroat":         {"Deepthroat"},
	"threesome":          {"Threesome", "3some"},
	"3some":              {"Threesome", "3some"},
	"gangbang":           {"Gangbang"},
	"creampie":           {"Creampie"},
	"cumshot":            {"Cumshot"},
	"facial":             {"Facial"},
	"squirt":             {"Squirting"},
	"squirting":          {"Squirting"},

	// Settings/Scenarios
	"pov":     {"POV"},
	"public":  {"Public"},
	"outdoor": {"Outdoor"},
	"car":     {"Car Sex"},
	"office":  {"Office"},
	"bedroom": {"Bedroom"},
	"shower":  {"Shower"},
	"bath":    {"Bath", "Bathroom"},
	"pool":    {"Pool"},
	"massage": {"Massage"},

	// Attributes
	"lesbian":      {"Lesbian"},
	"solo":         {"Solo"},
	"masturbation": {"Masturbation"},
	"toys":         {"Toys"},
	"dildo":        {"Dildo"},
	"bbc":          {"BBC"},
	"interracial":  {"Interracial"},
	"rough":        {"Rough"},
	"hardcore":     {"Hardcore"},
	"softcore":     {"Softcore"},
	"romantic":     {"Romantic"},
	"sensual":      {"Sensual"},

	// Physical traits
	"big tits":   {"Big Tits"},
	"small tits": {"Small Tits"},
	"natural":    {"Natural"},
	"fake tits":  {"Fake Tits"},
	"milf":       {"MILF"},
	"teen":       {"Teen"},
	"mature":     {"Mature"},
	"bbw":        {"BBW"},
	"petite":     {"Petite"},
	"tall":       {"Tall"},

	// Clothing
	"lingerie":  {"Lingerie"},
	"stockings": {"Stockings"},
	"heels":     {"Heels"},
	"uniform":   {"Uniform"},
	"cosplay":   {"Cosplay"},
}

// SuggestTags analyzes videos and suggests relevant tags based on content analysis
func (s *AIService) SuggestTags(videoIDs []int64, autoApply bool, minConfidence float64) ([]VideoTagSuggestion, error) {
	log.Printf("Starting smart tagging analysis for %d videos (auto-apply: %v, min confidence: %.2f)", len(videoIDs), autoApply, minConfidence)

	// If no video IDs provided, analyze all videos
	query := `SELECT id, title, file_path, description FROM videos`
	args := []interface{}{}

	if len(videoIDs) > 0 {
		placeholders := strings.Repeat("?,", len(videoIDs)-1) + "?"
		query += " WHERE id IN (" + placeholders + ")"
		for _, id := range videoIDs {
			args = append(args, id)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query videos: %w", err)
	}
	defer rows.Close()

	var suggestions []VideoTagSuggestion

	// Get all existing tags
	allTags, err := s.getAllTags()
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	appliedCount := 0

	for rows.Next() {
		var videoID int64
		var title, filePath, description sql.NullString

		if err := rows.Scan(&videoID, &title, &filePath, &description); err != nil {
			log.Printf("Failed to scan video: %v", err)
			continue
		}

		// Get existing tags for this video
		existingTags, err := s.getVideoTags(videoID)
		if err != nil {
			log.Printf("Failed to get existing tags for video %d: %v", videoID, err)
			continue
		}
		existingTagSet := make(map[int64]bool)
		for _, tag := range existingTags {
			existingTagSet[tag.ID] = true
		}

		// Analyze video and suggest tags
		videoSuggestions := s.analyzeVideoForTags(
			videoID,
			title.String,
			filePath.String,
			description.String,
			allTags,
			existingTagSet,
		)

		// Auto-apply high confidence tags if requested
		if autoApply {
			for _, suggestion := range videoSuggestions.Suggestions {
				if suggestion.Confidence >= minConfidence {
					if err := s.addTagToVideo(videoID, suggestion.TagID); err != nil {
						log.Printf("Failed to auto-apply tag %d to video %d: %v", suggestion.TagID, videoID, err)
					} else {
						appliedCount++
					}
				}
			}
		}

		// Only include suggestions that aren't auto-applied or below threshold
		filteredSuggestions := []TagSuggestion{}
		for _, suggestion := range videoSuggestions.Suggestions {
			if !autoApply || suggestion.Confidence < minConfidence {
				filteredSuggestions = append(filteredSuggestions, suggestion)
			}
		}

		if len(filteredSuggestions) > 0 {
			videoSuggestions.Suggestions = filteredSuggestions
			suggestions = append(suggestions, videoSuggestions)
		}
	}

	log.Printf("Found suggestions for %d videos, auto-applied %d tags", len(suggestions), appliedCount)
	return suggestions, nil
}

// analyzeVideoForTags analyzes a video's metadata and suggests appropriate tags
func (s *AIService) analyzeVideoForTags(
	videoID int64,
	title, filePath, description string,
	allTags []models.Tag,
	existingTags map[int64]bool,
) VideoTagSuggestion {
	suggestion := VideoTagSuggestion{
		VideoID:     videoID,
		VideoTitle:  title,
		FilePath:    filePath,
		Suggestions: []TagSuggestion{},
	}

	// Combine all text for analysis
	analysisText := strings.ToLower(title + " " + filepath.Base(filePath) + " " + description)

	// Track suggested tags to avoid duplicates
	suggestedTags := make(map[int64]TagSuggestion)

	// Check for keyword matches
	for keyword, tagNames := range tagKeywords {
		if strings.Contains(analysisText, keyword) {
			// Find matching tags in the database
			for _, tag := range allTags {
				if existingTags[tag.ID] {
					continue // Skip tags already on video
				}

				for _, tagName := range tagNames {
					if strings.EqualFold(tag.Name, tagName) {
						confidence := 0.85 // High confidence for keyword matches

						// Boost confidence if keyword appears multiple times
						count := strings.Count(analysisText, keyword)
						if count > 1 {
							confidence = 0.95
						}

						// Add or update suggestion
						if existing, exists := suggestedTags[tag.ID]; !exists || confidence > existing.Confidence {
							suggestedTags[tag.ID] = TagSuggestion{
								TagID:      tag.ID,
								TagName:    tag.Name,
								Confidence: confidence,
								Reason:     fmt.Sprintf("Keyword '%s' found in video metadata", keyword),
							}
						}
						break
					}
				}
			}
		}
	}

	// Also check for direct tag name matches (fuzzy matching)
	for _, tag := range allTags {
		if existingTags[tag.ID] {
			continue
		}
		if _, alreadySuggested := suggestedTags[tag.ID]; alreadySuggested {
			continue
		}

		tagLower := strings.ToLower(tag.Name)

		// Exact match
		if strings.Contains(analysisText, tagLower) {
			suggestedTags[tag.ID] = TagSuggestion{
				TagID:      tag.ID,
				TagName:    tag.Name,
				Confidence: 0.80,
				Reason:     "Tag name found in video metadata",
			}
		}
	}

	// Convert map to slice and sort by confidence
	for _, tagSuggestion := range suggestedTags {
		suggestion.Suggestions = append(suggestion.Suggestions, tagSuggestion)
	}

	// Sort by confidence (highest first)
	for i := 0; i < len(suggestion.Suggestions); i++ {
		for j := i + 1; j < len(suggestion.Suggestions); j++ {
			if suggestion.Suggestions[j].Confidence > suggestion.Suggestions[i].Confidence {
				suggestion.Suggestions[i], suggestion.Suggestions[j] = suggestion.Suggestions[j], suggestion.Suggestions[i]
			}
		}
	}

	return suggestion
}

// getAllTags retrieves all tags from the database
func (s *AIService) getAllTags() ([]models.Tag, error) {
	query := `SELECT id, name, created_at FROM tags ORDER BY name`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// getVideoTags retrieves all tags currently assigned to a video
func (s *AIService) getVideoTags(videoID int64) ([]models.Tag, error) {
	query := `
		SELECT t.id, t.name, t.created_at
		FROM tags t
		JOIN video_tags vt ON vt.tag_id = t.id
		WHERE vt.video_id = ?
	`
	rows, err := s.db.Query(query, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// addTagToVideo adds a tag to a video if it doesn't already exist
func (s *AIService) addTagToVideo(videoID, tagID int64) error {
	// Check if tag already exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM video_tags WHERE video_id = ? AND tag_id = ?)", videoID, tagID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return nil // Tag already exists
	}

	_, err = s.db.Exec("INSERT INTO video_tags (video_id, tag_id) VALUES (?, ?)", videoID, tagID)
	return err
}

// ApplyTagSuggestions applies selected tag suggestions to a video
func (s *AIService) ApplyTagSuggestions(videoID int64, tagIDs []int64) error {
	for _, tagID := range tagIDs {
		if err := s.addTagToVideo(videoID, tagID); err != nil {
			return fmt.Errorf("failed to apply tag %d: %w", tagID, err)
		}
	}
	return nil
}
