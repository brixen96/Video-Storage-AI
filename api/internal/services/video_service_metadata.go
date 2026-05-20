package services

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// UpdateConversionLink links a video to its converted counterpart.
func (s *VideoService) UpdateConversionLink(videoID int64, linkedVideoID int64, isOriginal bool) error {
	column := "converted_from"
	if isOriginal {
		column = "converted_to"
	}
	query := fmt.Sprintf("UPDATE videos SET %s = ?, updated_at = ? WHERE id = ?", column)
	_, err := s.db.Exec(query, linkedVideoID, time.Now(), videoID)
	return err
}

// CopyPerformers copies all performer associations from one video to another.
func (s *VideoService) CopyPerformers(fromVideoID, toVideoID int64) error {
	_, err := s.db.Exec(`
		INSERT INTO video_performers (video_id, performer_id)
		SELECT ?, performer_id FROM video_performers WHERE video_id = ?
	`, toVideoID, fromVideoID)
	return err
}

// CopyTags copies all tag associations from one video to another.
func (s *VideoService) CopyTags(fromVideoID, toVideoID int64) error {
	_, err := s.db.Exec(`
		INSERT INTO video_tags (video_id, tag_id)
		SELECT ?, tag_id FROM video_tags WHERE video_id = ?
	`, toVideoID, fromVideoID)
	return err
}

// CopyStudios copies all studio associations from one video to another.
func (s *VideoService) CopyStudios(fromVideoID, toVideoID int64) error {
	_, err := s.db.Exec(`
		INSERT INTO video_studios (video_id, studio_id)
		SELECT ?, studio_id FROM video_studios WHERE video_id = ?
	`, toVideoID, fromVideoID)
	return err
}

// CopyGroups copies all group associations from one video to another.
func (s *VideoService) CopyGroups(fromVideoID, toVideoID int64) error {
	_, err := s.db.Exec(`
		INSERT INTO video_groups (video_id, group_id)
		SELECT ?, group_id FROM video_groups WHERE video_id = ?
	`, toVideoID, fromVideoID)
	return err
}

// CopyMetadata copies date, rating, and description from one video to another.
func (s *VideoService) CopyMetadata(fromVideoID, toVideoID int64) error {
	_, err := s.db.Exec(`
		UPDATE videos
		SET date        = (SELECT date        FROM videos WHERE id = ?),
		    rating      = (SELECT rating      FROM videos WHERE id = ?),
		    description = (SELECT description FROM videos WHERE id = ?),
		    updated_at  = ?
		WHERE id = ?
	`, fromVideoID, fromVideoID, fromVideoID, time.Now(), toVideoID)
	return err
}

// UpdateContentTypeByFolder sets content_type on all videos whose file_path is under folderPath.
func (s *VideoService) UpdateContentTypeByFolder(folderPath string, contentType string) (int64, error) {
	searchPath := folderPath
	if !strings.HasSuffix(searchPath, "\\") && !strings.HasSuffix(searchPath, "/") {
		searchPath += "\\"
	}

	log.Printf("UpdateContentTypeByFolder: folderPath=%s pattern=%s", folderPath, searchPath+"%")

	result, err := s.db.Exec(`
		UPDATE videos SET content_type = ?, updated_at = ?
		WHERE file_path LIKE ?
	`, contentType, time.Now(), searchPath+"%")
	if err != nil {
		log.Printf("UpdateContentTypeByFolder: error executing query: %v", err)
		return 0, fmt.Errorf("failed to update folder content type: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	log.Printf("UpdateContentTypeByFolder: updated %d videos with content_type=%s", rowsAffected, contentType)
	return rowsAffected, nil
}

// LinkPerformerToFolderVideos links all videos under folderPath to performerID (skipping existing links).
func (s *VideoService) LinkPerformerToFolderVideos(folderPath string, performerID int64) (int64, error) {
	searchPath := folderPath
	if !strings.HasSuffix(searchPath, "\\") && !strings.HasSuffix(searchPath, "/") {
		searchPath += "\\"
	}

	result, err := s.db.Exec(`
		INSERT INTO video_performers (video_id, performer_id)
		SELECT v.id, ?
		FROM videos v
		WHERE v.file_path LIKE ?
		AND NOT EXISTS (
			SELECT 1 FROM video_performers vp
			WHERE vp.video_id = v.id AND vp.performer_id = ?
		)
	`, performerID, searchPath+"%", performerID)
	if err != nil {
		return 0, fmt.Errorf("failed to link performer to folder: %w", err)
	}

	linked, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	s.db.Exec(`UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = ?) WHERE id = ?`, performerID, performerID)

	log.Printf("LinkPerformerToFolderVideos: linked performer %d to %d videos in folder %s", performerID, linked, folderPath)
	return linked, nil
}
