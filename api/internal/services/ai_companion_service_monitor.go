package services

import (
	"fmt"
	"log"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// monitorActivityLogs monitors activity_logs table for task completions
func (s *AICompanionService) monitorActivityLogs() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkActivityLogUpdates()
		}
	}
}

// checkActivityLogUpdates checks for new completed activities
func (s *AICompanionService) checkActivityLogUpdates() {
	// Query for activities that completed since last check
	query := `
		SELECT id, task_type, status, message, progress, started_at, completed_at, details
		FROM activity_logs
		WHERE status = 'completed'
		AND completed_at IS NOT NULL
		AND completed_at > ?
		ORDER BY completed_at ASC
	`

	rows, err := s.db.Query(query, s.lastActivityCheck)
	if err != nil {
		log.Printf("Failed to query activity logs: %v", err)
		return
	}
	defer rows.Close()

	s.mu.Lock()
	currentTime := time.Now()
	s.mu.Unlock()

	for rows.Next() {
		var activity models.ActivityLog
		err := rows.Scan(
			&activity.ID,
			&activity.TaskType,
			&activity.Status,
			&activity.Message,
			&activity.Progress,
			&activity.StartedAt,
			&activity.CompletedAt,
			&activity.Details,
		)
		if err != nil {
			log.Printf("Failed to scan activity log: %v", err)
			continue
		}

		// Unmarshal details
		if err := activity.UnmarshalDetails(); err != nil {
			log.Printf("Failed to unmarshal activity details: %v", err)
		}

		// Process the completed activity
		s.processCompletedActivity(&activity)
	}

	// Update last check time
	s.mu.Lock()
	s.lastActivityCheck = currentTime
	s.mu.Unlock()
}

// processCompletedActivity processes a completed activity and reacts if it matches a recommendation
func (s *AICompanionService) processCompletedActivity(activity *models.ActivityLog) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if this activity matches any of our recommendations
	var matchedRecommendation *Recommendation
	for id, rec := range s.recommendations {
		if rec.TaskType == activity.TaskType {
			matchedRecommendation = rec
			delete(s.recommendations, id) // Remove once matched
			break
		}
	}

	// Generate contextual response based on task type and whether it was recommended
	var responseMessage string
	var severity string = "info"

	if matchedRecommendation != nil {
		// User followed our recommendation!
		responseMessage = s.generateFollowUpResponse(activity, matchedRecommendation)
		severity = "info"
	} else {
		// User initiated this task on their own - still acknowledge
		responseMessage = s.generateAcknowledgmentResponse(activity)
		severity = "info"
	}

	// Emit the response event
	if responseMessage != "" {
		s.emitEvent(CompanionEvent{
			Type:    "notification",
			Source:  "activity_monitor",
			Message: responseMessage,
			Data: map[string]interface{}{
				"activity_id":       activity.ID,
				"task_type":         activity.TaskType,
				"was_recommended":   matchedRecommendation != nil,
				"completion_time":   activity.CompletedAt,
				"details":           activity.DetailsObj,
			},
			Severity:  severity,
			Timestamp: time.Now(),
		})
	}

	// Perform post-completion analysis
	s.analyzeCompletionImpact(activity)
}

// generateFollowUpResponse creates a contextual response when user follows a recommendation
func (s *AICompanionService) generateFollowUpResponse(activity *models.ActivityLog, rec *Recommendation) string {
	switch activity.TaskType {
	case "performer_thumbnail_generation":
		totalCount := 0
		if val, ok := activity.DetailsObj["total_count"].(float64); ok {
			totalCount = int(val)
		}
		return fmt.Sprintf("🎉 Great! I see you generated thumbnails for %d performers. The Performers page should load much faster now. Your library performance is improving!", totalCount)

	case "video_thumbnail_generation":
		return "🎉 Excellent! Video thumbnails have been generated. Your library browsing experience should be much smoother now!"

	case "thumbnail_generation_batch":
		totalCount := 0
		if val, ok := activity.DetailsObj["total_count"].(float64); ok {
			totalCount = int(val)
		}
		return fmt.Sprintf("✅ Excellent! Video thumbnails generated for %d files. Browse performance is now optimized.", totalCount)

	case "library_scan":
		return "🔍 Library scan completed! I've indexed all the new content. Your library is now up to date."

	case "performer_scan":
		scannedCount := 0
		if val, ok := activity.DetailsObj["scanned_count"].(float64); ok {
			scannedCount = int(val)
		}
		return fmt.Sprintf("👤 Performer scan finished! Processed %d performers. All performer previews are now updated.", scannedCount)

	case "metadata_fetch":
		return "📋 Metadata fetching complete! Your content now has enriched information from external sources."

	case "ai_tagging":
		return "🏷️ AI tagging finished! Your videos are now automatically categorized and easier to discover."

	default:
		return fmt.Sprintf("✅ Task '%s' completed successfully! Thanks for keeping your library optimized.", activity.TaskType)
	}
}

// generateAcknowledgmentResponse creates a response for user-initiated tasks
func (s *AICompanionService) generateAcknowledgmentResponse(activity *models.ActivityLog) string {
	switch activity.TaskType {
	case "performer_thumbnail_generation":
		return "✅ Performer thumbnails generated! Your Performers page should load significantly faster now."

	case "video_thumbnail_generation":
		return "✅ Video thumbnails generated successfully! Your videos now have preview thumbnails."

	case "thumbnail_generation_batch":
		return "✅ Video thumbnails generated successfully! Browse performance improved."

	case "library_scan":
		return "✅ Library scan complete! All content has been indexed."

	case "performer_scan":
		return "✅ Performer scan complete! All performer previews updated."

	default:
		// Only acknowledge major tasks, return empty for minor ones
		return ""
	}
}

// analyzeCompletionImpact analyzes the impact of a completed task and suggests next steps
func (s *AICompanionService) analyzeCompletionImpact(activity *models.ActivityLog) {
	// After thumbnail generation, check if there are other optimization opportunities
	if activity.TaskType == "performer_thumbnail_generation" {
		// Check if video thumbnails also need generation
		var videosNeedingThumbnails int
		s.db.QueryRow(`
			SELECT COUNT(*)
			FROM videos
			WHERE (thumbnail_path IS NULL OR thumbnail_path = '')
		`).Scan(&videosNeedingThumbnails)

		if videosNeedingThumbnails > 100 {
			s.emitEvent(CompanionEvent{
				Type:    "notification",
				Source:  "activity_monitor",
				Message: fmt.Sprintf("💡 Next optimization: %d videos don't have thumbnails. Consider generating them for even better performance!", videosNeedingThumbnails),
				Data: map[string]interface{}{
					"videos_needing_thumbnails": videosNeedingThumbnails,
					"action":                    "generate_video_thumbnails",
				},
				Severity:  "info",
				Timestamp: time.Now(),
			})
		}
	}

	// After library scan, suggest metadata enrichment
	if activity.TaskType == "library_scan" {
		var videosWithoutMetadata int
		s.db.QueryRow(`
			SELECT COUNT(*)
			FROM videos
			WHERE (metadata IS NULL OR metadata = '' OR metadata = '{}')
		`).Scan(&videosWithoutMetadata)

		if videosWithoutMetadata > 50 {
			s.emitEvent(CompanionEvent{
				Type:    "notification",
				Source:  "activity_monitor",
				Message: fmt.Sprintf("📋 Suggestion: %d videos have minimal metadata. Fetch metadata to enrich your library!", videosWithoutMetadata),
				Data: map[string]interface{}{
					"videos_without_metadata": videosWithoutMetadata,
					"action":                  "fetch_metadata",
				},
				Severity:  "info",
				Timestamp: time.Now(),
			})
		}
	}
}

// trackRecommendation stores a recommendation that was given to the user
func (s *AICompanionService) trackRecommendation(recType, taskType, message string, context map[string]interface{}) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("%s_%d", recType, time.Now().UnixNano())
	rec := &Recommendation{
		ID:          id,
		Type:        recType,
		TaskType:    taskType,
		Message:     message,
		SuggestedAt: time.Now(),
		Context:     context,
	}

	s.recommendations[id] = rec
	return id
}
