package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// ThreadEnhancer handles AI-powered thread content enhancement
type ThreadEnhancer struct {
	db          *sql.DB
	aiService   *AIService
	activitySvc *ActivityService
}

// NewThreadEnhancer creates a new thread enhancer service
func NewThreadEnhancer(activitySvc *ActivityService) *ThreadEnhancer {
	return &ThreadEnhancer{
		db:          database.GetDB(),
		aiService:   NewAIService(),
		activitySvc: activitySvc,
	}
}

// EnhancementResult represents the result of thread enhancement
type EnhancementResult struct {
	ThreadID        int64               `json:"thread_id"`
	PostsProcessed  int                 `json:"posts_processed"`
	PostsCleaned    int                 `json:"posts_cleaned"`
	PerformersFound []string            `json:"performers_found"`
	TagsExtracted   []string            `json:"tags_extracted"`
	Summary         string              `json:"summary"`
	Duration        time.Duration       `json:"duration"`
	Details         []PostEnhancement   `json:"details,omitempty"`
	ImagesValidated int                 `json:"images_validated"`
	ImagesValid     int                 `json:"images_valid"`
	ImagesInvalid   int                 `json:"images_invalid"`
}

// PostEnhancement represents changes made to a single post
type PostEnhancement struct {
	PostID          int64    `json:"post_id"`
	OriginalLength  int      `json:"original_length"`
	CleanedLength   int      `json:"cleaned_length"`
	PerformersFound []string `json:"performers_found,omitempty"`
	TagsFound       []string `json:"tags_found,omitempty"`
	ChangesMade     []string `json:"changes_made"`
}

// EnhanceThread performs AI-powered enhancement on a thread's posts
func (s *ThreadEnhancer) EnhanceThread(threadID int64, options EnhancementOptions) (*EnhancementResult, error) {
	startTime := time.Now()

	// Create activity log
	activity, err := s.activitySvc.Create(&models.ActivityLogCreate{
		TaskType: "thread_enhancement",
		Status:   "running",
		Message:  fmt.Sprintf("Enhancing thread %d", threadID),
		Details: map[string]interface{}{
			"thread_id": threadID,
			"options":   options,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create activity: %w", err)
	}

	// Get thread
	var thread models.ScrapedThread
	err = s.db.QueryRow(`
		SELECT id, title, author, category, metadata
		FROM scraped_threads WHERE id = ?
	`, threadID).Scan(&thread.ID, &thread.Title, &thread.Author, &thread.Category, &thread.Metadata)
	if err != nil {
		s.activitySvc.FailTask(activity.ID, fmt.Sprintf("Thread not found: %v", err))
		return nil, fmt.Errorf("thread not found: %w", err)
	}

	// Get all posts for this thread
	posts, err := s.getThreadPosts(threadID)
	if err != nil {
		s.activitySvc.FailTask(activity.ID, fmt.Sprintf("Failed to get posts: %v", err))
		return nil, fmt.Errorf("failed to get posts: %w", err)
	}

	result := &EnhancementResult{
		ThreadID:        threadID,
		PostsProcessed:  len(posts),
		PerformersFound: []string{},
		TagsExtracted:   []string{},
		Details:         []PostEnhancement{},
	}

	// Process each post
	for i, post := range posts {
		// Update progress
		progress := int(float64(i+1) / float64(len(posts)) * 100)
		s.activitySvc.UpdateProgress(activity.ID, progress, 
			fmt.Sprintf("Processing post %d/%d", i+1, len(posts)))

		enhancement := s.enhancePost(post, options)
		if enhancement != nil {
			result.Details = append(result.Details, *enhancement)
			result.PostsCleaned++

			// Collect unique performers and tags
			for _, p := range enhancement.PerformersFound {
				if !contains(result.PerformersFound, p) {
					result.PerformersFound = append(result.PerformersFound, p)
				}
			}
			for _, t := range enhancement.TagsFound {
				if !contains(result.TagsExtracted, t) {
					result.TagsExtracted = append(result.TagsExtracted, t)
				}
			}
		}
	}

	// Update thread metadata with extracted performers
	if options.ExtractPerformers && len(result.PerformersFound) > 0 {
		s.updateThreadPerformers(threadID, result.PerformersFound)
	}

	// Validate images if requested
	if options.ValidateImages {
		imageCount, validCount, invalidCount := s.validateThreadImages(threadID, thread.Title)
		result.ImagesValidated = imageCount
		result.ImagesValid = validCount
		result.ImagesInvalid = invalidCount
	}

	// Generate summary
	result.Summary = s.generateSummary(result, thread)
	result.Duration = time.Since(startTime)

	// Complete activity
	s.activitySvc.CompleteTask(int64(activity.ID), result.Summary)

	return result, nil
}

// EnhancementOptions controls what enhancements to apply
type EnhancementOptions struct {
	CleanContent      bool `json:"clean_content"`      // Remove forum boilerplate
	ExtractPerformers bool `json:"extract_performers"` // Extract performer names
	ExtractTags       bool `json:"extract_tags"`       // Extract content tags
	GenerateSummary   bool `json:"generate_summary"`   // Generate thread summary
	ValidateLinks     bool `json:"validate_links"`     // Check link health
	ValidateImages    bool `json:"validate_images"`    // Validate images using AI vision
}

// getThreadPosts retrieves all posts for a thread
func (s *ThreadEnhancer) getThreadPosts(threadID int64) ([]*models.ScrapedPost, error) {
	rows, err := s.db.Query(`
		SELECT id, thread_id, external_id, source, author, content, plain_text,
			   post_number, like_count, metadata, is_read, posted_at, scraped_at, created_at
		FROM scraped_posts
		WHERE thread_id = ?
		ORDER BY post_number ASC
	`, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*models.ScrapedPost
	for rows.Next() {
		post := &models.ScrapedPost{}
		err := rows.Scan(
			&post.ID, &post.ThreadID, &post.ExternalID, &post.Source,
			&post.Author, &post.Content, &post.PlainText, &post.PostNumber,
			&post.LikeCount, &post.Metadata, &post.IsRead, &post.PostedAt,
			&post.ScrapedAt, &post.CreatedAt,
		)
		if err != nil {
			log.Printf("Failed to scan post: %v", err)
			continue
		}
		// Unmarshal metadata
		if post.Metadata != "" {
			var meta models.ScrapedPostMetadata
			if err := json.Unmarshal([]byte(post.Metadata), &meta); err == nil {
				post.MetadataObj = &meta
			}
		}
		posts = append(posts, post)
	}
	return posts, nil
}

// enhancePost applies enhancements to a single post
func (s *ThreadEnhancer) enhancePost(post *models.ScrapedPost, options EnhancementOptions) *PostEnhancement {
	enhancement := &PostEnhancement{
		PostID:         post.ID,
		OriginalLength: len(post.PlainText),
		ChangesMade:    []string{},
	}

	// Clean content - remove common forum boilerplate
	if options.CleanContent {
		cleaned := s.cleanPostContent(post.Content, post.PlainText)
		if cleaned != post.PlainText {
			post.PlainText = cleaned
			enhancement.ChangesMade = append(enhancement.ChangesMade, "cleaned_content")
		}
	}

	// Extract performers from content
	if options.ExtractPerformers {
		performers := s.extractPerformers(post)
		if len(performers) > 0 {
			enhancement.PerformersFound = performers
			enhancement.ChangesMade = append(enhancement.ChangesMade, "extracted_performers")
		}
	}

	// Extract tags
	if options.ExtractTags {
		tags := s.extractTags(post)
		if len(tags) > 0 {
			enhancement.TagsFound = tags
			enhancement.ChangesMade = append(enhancement.ChangesMade, "extracted_tags")
		}
	}

	enhancement.CleanedLength = len(post.PlainText)

	// Save changes if any
	if len(enhancement.ChangesMade) > 0 {
		s.savePostEnhancements(post)
	}

	// Only return if changes were made
	if len(enhancement.ChangesMade) == 0 {
		return nil
	}
	return enhancement
}

// cleanPostContent removes forum boilerplate and cleans up text
func (s *ThreadEnhancer) cleanPostContent(html, plainText string) string {
	cleaned := plainText

	// Common forum signatures and boilerplate patterns
	boilerplatePatterns := []string{
		`(?i)sent from my .+`,
		`(?i)posted from .+`,
		`(?i)via tapatalk`,
		`(?i)click to expand`,
		`(?i)show\s*full\s*quote`,
		`(?i)reply\s*with\s*quote`,
		`(?i)edit\s*\|\s*delete\s*\|\s*report`,
		`(?i)\d+\s*(posts?|comments?)\s*liked`,
		`(?i)like\s*-\s*\d+`,
		`(?i)post\s*#\d+`,
		`(?i)quote\s*Originally\s*Posted\s*by\s*.+`,
		`(?i)\[img\]\s*\[\/img\]`,
		`(?i)\[url\]\s*\[\/url\]`,
	}

	for _, pattern := range boilerplatePatterns {
		re := regexp.MustCompile(pattern)
		cleaned = re.ReplaceAllString(cleaned, "")
	}

	// Clean up excessive whitespace
	cleaned = regexp.MustCompile(`\n{3,}`).ReplaceAllString(cleaned, "\n\n")
	cleaned = regexp.MustCompile(`[ \t]{2,}`).ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}

// extractPerformers extracts performer names from post content
func (s *ThreadEnhancer) extractPerformers(post *models.ScrapedPost) []string {
	performers := []string{}

	// Check if performers already exist in metadata
	if post.MetadataObj != nil && len(post.MetadataObj.ExtractedLinks) > 0 {
		return performers
	}

	// Only extract from explicit "starring/featuring" patterns - much stricter
	patterns := []string{
		`(?i)(?:starring|featuring|with|models?|performers?)\s*:\s*([A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+)+)`,
		`(?i)(?:girl|boy|male|female)\s*:\s*([A-Z][a-zA-Z]+(?:\s+[A-Z][a-zA-Z]+)?)`,
	}

	content := post.PlainText
	if content == "" {
		content = post.Content
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				if isValidPerformerName(name) && !contains(performers, name) {
					performers = append(performers, name)
				}
			}
		}
	}

	return performers
}

// extractTags extracts content tags from post
func (s *ThreadEnhancer) extractTags(post *models.ScrapedPost) []string {
	tags := []string{}

	content := post.PlainText
	if content == "" {
		content = post.Content
	}

	// Only specific content tags - avoid generic terms
	tagPatterns := map[string]string{
		"bbc":         `(?i)\bbc\b`,
		"interracial": `(?i)\binterracial\b`,
		"anal":        `(?i)\banal\b`,
		"blowjob":     `(?i)\bblowjob\b`,
		"creampie":    `(?i)\bcreampie\b`,
		"threesome":   `(?i)\b threesome\b`,
		"milf":        `(?i)\bmilf\b`,
		"lesbian":     `(?i)\blesbian\b`,
		" gangbang":   `(?i)\bgangbang\b`,
		"hardcore":    `(?i)\bhardcore\b`,
	}

	for tag, pattern := range tagPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			tags = append(tags, tag)
		}
	}

	return tags
}

// savePostEnhancements persists changes to the database
func (s *ThreadEnhancer) savePostEnhancements(post *models.ScrapedPost) error {
	_, err := s.db.Exec(`
		UPDATE scraped_posts 
		SET plain_text = ?, content = ?, metadata = ?
		WHERE id = ?
	`, post.PlainText, post.Content, post.Metadata, post.ID)
	return err
}

// updateThreadPerformers updates thread metadata with extracted performers
func (s *ThreadEnhancer) updateThreadPerformers(threadID int64, performers []string) error {
	// Get current metadata
	var metadataJSON string
	err := s.db.QueryRow("SELECT metadata FROM scraped_threads WHERE id = ?", threadID).Scan(&metadataJSON)
	if err != nil {
		return err
	}

	var metadata models.ScrapedThreadMetadata
	if metadataJSON != "" {
		json.Unmarshal([]byte(metadataJSON), &metadata)
	}

	// Merge performers
	existingPerformers := make(map[string]bool)
	for _, p := range metadata.PerformerNames {
		existingPerformers[p] = true
	}
	for _, p := range performers {
		if !existingPerformers[p] {
			metadata.PerformerNames = append(metadata.PerformerNames, p)
		}
	}

	// Save back
	newMetadata, _ := json.Marshal(metadata)
	_, err = s.db.Exec("UPDATE scraped_threads SET metadata = ? WHERE id = ?", string(newMetadata), threadID)
	return err
}

// generateSummary creates a human-readable summary of the enhancement
func (s *ThreadEnhancer) generateSummary(result *EnhancementResult, thread models.ScrapedThread) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Enhanced thread \"%s\"", thread.Title))
	parts = append(parts, fmt.Sprintf("Processed %d posts", result.PostsProcessed))

	if result.PostsCleaned > 0 {
		parts = append(parts, fmt.Sprintf("Cleaned %d posts", result.PostsCleaned))
	}
	if len(result.PerformersFound) > 0 {
		parts = append(parts, fmt.Sprintf("Found %d performers: %s", 
			len(result.PerformersFound), strings.Join(result.PerformersFound, ", ")))
	}
	if len(result.TagsExtracted) > 0 {
		parts = append(parts, fmt.Sprintf("Extracted tags: %s", 
			strings.Join(result.TagsExtracted, ", ")))
	}
	if result.ImagesValidated > 0 {
		parts = append(parts, fmt.Sprintf("Validated %d images: %d valid, %d invalid", 
			result.ImagesValidated, result.ImagesValid, result.ImagesInvalid))
	}

	return strings.Join(parts, ". ")
}

// isValidPerformerName validates a potential performer name
func isValidPerformerName(name string) bool {
	// Must be exactly 2 words (first name + last name)
	words := strings.Fields(name)
	if len(words) != 2 {
		return false
	}

	// Must not be too long
	if len(name) > 50 {
		return false
	}

	// Must not contain common non-performer words
	invalidWords := []string{
		"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with", "by",
		"hot", "big", "huge", "sexy", "hot", "new", "old", "free", "best", "top",
		"video", "photo", "image", "clip", "scene", "movie", "film",
		"watch", "download", "stream", "play", "view", "see",
		"folder", "file", "link", "url", "com", "net", "org",
		"reupload", "reup", "backup", "copy", "original",
		"hd", "sd", "4k", "720p", "1080p",
		"starring", "featuring", "with", "model", "star", "porn",
		"girl", "boy", "male", "female", "teen", "milf",
		"amateur", "professional", "studio", "production",
		"lesbian", "gay", "straight", "bisexual",
	}

	// Check if any word is in the invalid list
	for _, word := range words {
		for _, invalid := range invalidWords {
			if strings.EqualFold(word, invalid) {
				return false
			}
		}
	}

	// First letter of each word should be capitalized (proper name)
	for _, word := range words {
		if len(word) > 0 && !strings.HasPrefix(word, strings.ToUpper(string(word[0]))) {
			return false
		}
	}

	return true
}

// validateThreadImages validates all images in a thread using AI vision
func (s *ThreadEnhancer) validateThreadImages(threadID int64, threadTitle string) (imageCount, validCount, invalidCount int) {
	// Get API key from config
	apiKey := s.getOpenRouterAPIKey()
	if apiKey == "" {
		log.Printf("[ImageValidator] No OpenRouter API key configured - skipping image validation")
		return 0, 0, 0
	}

	// Get all image URLs from thread posts
	rows, err := s.db.Query(`
		SELECT DISTINCT metadata 
		FROM scraped_posts 
		WHERE thread_id = ? AND metadata != ''
	`, threadID)
	if err != nil {
		log.Printf("Failed to query posts for image validation: %v", err)
		return 0, 0, 0
	}
	defer rows.Close()

	var imageURLs []string
	for rows.Next() {
		var metaStr string
		if err := rows.Scan(&metaStr); err == nil {
			var meta struct {
				Attachments []struct {
					Type         string `json:"type"`
					URL          string `json:"url"`
					ThumbnailURL string `json:"thumbnail_url"`
				} `json:"attachments"`
			}
			if err := json.Unmarshal([]byte(metaStr), &meta); err == nil {
				for _, att := range meta.Attachments {
					if att.Type == "image" {
						if att.ThumbnailURL != "" {
							imageURLs = append(imageURLs, att.ThumbnailURL)
						} else if att.URL != "" {
							imageURLs = append(imageURLs, att.URL)
						}
					}
				}
			}
		}
	}

	if len(imageURLs) == 0 {
		log.Printf("[ImageValidator] No images found in thread %d", threadID)
		return 0, 0, 0
	}

	log.Printf("[ImageValidator] Validating %d images in thread %d", len(imageURLs), threadID)

	// Validate images using AI
	validator := NewImageValidator(apiKey)
	results, err := validator.ValidateThreadImages(threadTitle, imageURLs)
	if err != nil {
		log.Printf("Image validation failed: %v", err)
		return len(imageURLs), 0, len(imageURLs)
	}

	validCount = 0
	invalidCount = 0
	for _, r := range results.Results {
		if r.IsValid {
			validCount++
		} else {
			invalidCount++
			if r.Error != "" {
				log.Printf("[ImageValidator] Invalid image in thread %d: %s - Error: %s", threadID, r.ImageURL, r.Error)
			}
		}
	}

	log.Printf("[ImageValidator] Validation complete: %d valid, %d invalid out of %d images", validCount, invalidCount, len(imageURLs))
	return len(imageURLs), validCount, invalidCount
}

// getOpenRouterAPIKey retrieves the OpenRouter API key from ai_config table
// (same config used by AI Companion)
func (s *ThreadEnhancer) getOpenRouterAPIKey() string {
	var apiKey string
	
	// Use the same query as AI Companion's GetAIConfig()
	err := s.db.QueryRow(`
		SELECT COALESCE(api_key, '') FROM ai_config WHERE id = 1
	`).Scan(&apiKey)
	if err != nil {
		log.Printf("[ImageValidator] Failed to fetch API key from ai_config: %v", err)
		return ""
	}
	
	if apiKey == "" {
		log.Printf("[ImageValidator] No OpenRouter API key configured in ai_config")
		return ""
	}
	
	return apiKey
}
