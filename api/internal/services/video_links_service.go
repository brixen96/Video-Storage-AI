package services

import (
	"database/sql"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// VideoLinksService manages Edited↔Backup video pairs.
type VideoLinksService struct {
	db *sql.DB
}

func NewVideoLinksService() *VideoLinksService {
	return &VideoLinksService{db: database.GetDB()}
}

// SuggestLinks computes link suggestions between two libraries without saving.
// editedLibID should be the "Edited" library, backupLibID the "Backup" library.
func (s *VideoLinksService) SuggestLinks(editedLibID, backupLibID int64) ([]*models.VideoLink, error) {
	edited, err := s.loadLibraryVideos(editedLibID)
	if err != nil {
		return nil, fmt.Errorf("load edited videos: %w", err)
	}
	backup, err := s.loadLibraryVideos(backupLibID)
	if err != nil {
		return nil, fmt.Errorf("load backup videos: %w", err)
	}

	// Load existing confirmed/suggested link pairs so we skip them
	existingPairs := make(map[string]bool)
	rows, _ := s.db.Query(`SELECT edited_video_id, backup_video_id FROM video_links WHERE status != 'rejected'`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var e, b int64
			if rows.Scan(&e, &b) == nil {
				existingPairs[fmt.Sprintf("%d:%d", e, b)] = true
			}
		}
		rows.Close()
	}

	var suggestions []*models.VideoLink
	for _, ev := range edited {
		best := scoreBestMatch(ev, backup, existingPairs)
		if best != nil {
			suggestions = append(suggestions, best)
		}
	}
	return suggestions, nil
}

// RunAndSaveSuggestions suggests links and upserts them as 'suggested' in the DB.
// Already-confirmed or already-rejected pairs are left untouched.
func (s *VideoLinksService) RunAndSaveSuggestions(editedLibID, backupLibID int64) ([]*models.VideoLink, error) {
	suggestions, err := s.SuggestLinks(editedLibID, backupLibID)
	if err != nil {
		return nil, err
	}

	for _, sug := range suggestions {
		_, err := s.db.Exec(`
			INSERT INTO video_links (edited_video_id, backup_video_id, confidence, status, match_reasons)
			VALUES (?, ?, ?, 'suggested', ?)
			ON CONFLICT(edited_video_id, backup_video_id) DO NOTHING`,
			sug.EditedVideoID, sug.BackupVideoID, sug.Confidence, sug.MatchReasons)
		if err != nil {
			return nil, fmt.Errorf("save suggestion: %w", err)
		}
	}
	return suggestions, nil
}

// GetLinks returns video links for review, with full video info populated.
// status filter: "suggested", "confirmed", "rejected", or "" for all.
func (s *VideoLinksService) GetLinks(editedLibID int64, status string) ([]*models.VideoLink, error) {
	query := `
		SELECT vl.id, vl.edited_video_id, vl.backup_video_id, vl.confidence, vl.status, vl.match_reasons,
		       vl.created_at, vl.updated_at,
		       ev.title, ev.file_path, COALESCE(ev.duration, 0), ev.library_id,
		       bv.title, bv.file_path, COALESCE(bv.duration, 0), bv.library_id,
		       el.name, bl.name
		FROM video_links vl
		INNER JOIN videos ev ON ev.id = vl.edited_video_id
		INNER JOIN videos bv ON bv.id = vl.backup_video_id
		INNER JOIN libraries el ON el.id = ev.library_id
		INNER JOIN libraries bl ON bl.id = bv.library_id
		WHERE ev.library_id = ?`
	args := []interface{}{editedLibID}
	if status != "" {
		query += ` AND vl.status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY vl.confidence DESC, vl.created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()

	var links []*models.VideoLink
	for rows.Next() {
		var vl models.VideoLink
		var ev, bv models.LinkedVideoEntry
		var elName, blName string
		if err := rows.Scan(
			&vl.ID, &vl.EditedVideoID, &vl.BackupVideoID, &vl.Confidence, &vl.Status, &vl.MatchReasons,
			&vl.CreatedAt, &vl.UpdatedAt,
			&ev.Title, &ev.FilePath, &ev.Duration, &ev.LibraryID,
			&bv.Title, &bv.FilePath, &bv.Duration, &bv.LibraryID,
			&elName, &blName,
		); err != nil {
			continue
		}
		ev.ID = vl.EditedVideoID
		ev.LibraryName = elName
		bv.ID = vl.BackupVideoID
		bv.LibraryName = blName
		vl.EditedVideo = &ev
		vl.BackupVideo = &bv
		links = append(links, &vl)
	}
	return links, nil
}

// UpdateLinkStatus sets the status of a single link.
func (s *VideoLinksService) UpdateLinkStatus(id int64, status string) error {
	_, err := s.db.Exec(
		`UPDATE video_links SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, id)
	return err
}

// BulkUpdateStatus sets the status for multiple links.
func (s *VideoLinksService) BulkUpdateStatus(ids []int64, status string) error {
	for _, id := range ids {
		if err := s.UpdateLinkStatus(id, status); err != nil {
			return err
		}
	}
	return nil
}

// GetLinkForVideo returns the confirmed link for a video (either side), or nil if none.
func (s *VideoLinksService) GetLinkForVideo(videoID int64) (*models.VideoLink, error) {
	row := s.db.QueryRow(`
		SELECT id, edited_video_id, backup_video_id, confidence, status, match_reasons, created_at, updated_at
		FROM video_links
		WHERE (edited_video_id = ? OR backup_video_id = ?) AND status = 'confirmed'
		LIMIT 1`, videoID, videoID)

	var vl models.VideoLink
	err := row.Scan(&vl.ID, &vl.EditedVideoID, &vl.BackupVideoID, &vl.Confidence, &vl.Status, &vl.MatchReasons, &vl.CreatedAt, &vl.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &vl, nil
}

// ─── Scoring helpers ────────────────────────────────────────────────────────

type libVideo struct {
	ID       int64
	Title    string
	FilePath string
	Duration float64
}

func (s *VideoLinksService) loadLibraryVideos(libraryID int64) ([]libVideo, error) {
	rows, err := s.db.Query(`
		SELECT id, title, file_path, COALESCE(duration, 0)
		FROM videos
		WHERE library_id = ? AND COALESCE(is_missing, 0) = 0`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vids []libVideo
	for rows.Next() {
		var v libVideo
		if rows.Scan(&v.ID, &v.Title, &v.FilePath, &v.Duration) == nil {
			vids = append(vids, v)
		}
	}
	return vids, nil
}

const minConfidence = 0.35

// scoreBestMatch returns the best-matching backup video for a single edited video.
func scoreBestMatch(ev libVideo, backups []libVideo, existingPairs map[string]bool) *models.VideoLink {
	var best *models.VideoLink
	var bestScore float64

	evWords := titleWords(ev.Title, ev.FilePath)

	for _, bv := range backups {
		pairKey := fmt.Sprintf("%d:%d", ev.ID, bv.ID)
		if existingPairs[pairKey] {
			continue
		}

		durScore := durationScore(ev.Duration, bv.Duration)
		nameScore := wordOverlapScore(evWords, titleWords(bv.Title, bv.FilePath))

		// Weighted: duration gets more weight when it's non-zero
		var combined float64
		if ev.Duration > 0 && bv.Duration > 0 {
			combined = 0.5*durScore + 0.5*nameScore
		} else {
			combined = nameScore
		}

		if combined < minConfidence {
			continue
		}

		if best == nil || combined > bestScore {
			reasons := buildReasons(durScore, nameScore)
			best = &models.VideoLink{
				EditedVideoID: ev.ID,
				BackupVideoID: bv.ID,
				Confidence:    math.Round(combined*100) / 100,
				Status:        "suggested",
				MatchReasons:  reasons,
				EditedVideo:   &models.LinkedVideoEntry{ID: ev.ID, Title: ev.Title, FilePath: ev.FilePath, Duration: ev.Duration},
				BackupVideo:   &models.LinkedVideoEntry{ID: bv.ID, Title: bv.Title, FilePath: bv.FilePath, Duration: bv.Duration},
			}
			bestScore = combined
		}
	}
	return best
}

// durationScore returns 0..1 based on how close two durations are.
func durationScore(a, b float64) float64 {
	if a == 0 || b == 0 {
		return 0
	}
	diff := math.Abs(a - b)
	switch {
	case diff <= 5:
		return 1.0
	case diff <= 30:
		return 0.85
	case diff <= 60:
		return 0.65
	case diff <= 120:
		return 0.4
	case diff <= 300:
		return 0.15
	default:
		return 0
	}
}

// stopWords are ignored when building word sets.
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "of": true,
	"in": true, "at": true, "to": true, "for": true, "on": true, "with": true,
	"mp4": true, "mkv": true, "avi": true, "wmv": true, "mov": true,
}

// titleWords extracts a normalised word set from title and filename.
func titleWords(title, filePath string) map[string]bool {
	stem := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	combined := strings.ToLower(title + " " + stem)
	// Replace common separators with spaces
	for _, ch := range []string{".", "_", "-", "(", ")", "[", "]"} {
		combined = strings.ReplaceAll(combined, ch, " ")
	}
	words := make(map[string]bool)
	for _, w := range strings.Fields(combined) {
		if len(w) >= 3 && !stopWords[w] {
			words[w] = true
		}
	}
	return words
}

// wordOverlapScore computes Jaccard similarity between two word sets.
func wordOverlapScore(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for w := range a {
		if b[w] {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

// buildReasons produces a JSON array string listing which signals matched well.
func buildReasons(durScore, nameScore float64) string {
	var parts []string
	if durScore >= 0.65 {
		parts = append(parts, `"duration"`)
	}
	if nameScore >= 0.4 {
		parts = append(parts, `"filename"`)
	}
	if len(parts) == 0 {
		parts = append(parts, `"partial"`)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
