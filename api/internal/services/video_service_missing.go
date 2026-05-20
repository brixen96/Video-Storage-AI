package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// FileDuplicateVideo is a lightweight video record used in file-based duplicate groups.
type FileDuplicateVideo struct {
	ID              int64   `json:"id"`
	LibraryID       int64   `json:"library_id"`
	Title           string  `json:"title"`
	FilePath        string  `json:"file_path"`
	FileSize        int64   `json:"file_size"`
	Duration        float64 `json:"duration"`
	ThumbnailPath   string  `json:"thumbnail_path"`
	ContentType     string  `json:"content_type"`
	Resolution      string  `json:"resolution"`
	IsTopazEnhanced bool    `json:"is_topaz_enhanced"`
	TopazType       string  `json:"topaz_type,omitempty"`
}

// FileDuplicateGroup holds videos that share the same file size and rounded duration.
type FileDuplicateGroup struct {
	FileSize int64                `json:"file_size"`
	Duration float64              `json:"duration"`
	Videos   []FileDuplicateVideo `json:"videos"`
}

// LocateFile searches the video's library directory for files with the same base name.
// Returns up to 10 candidate absolute paths.
func (s *VideoService) LocateFile(videoID int64) ([]string, error) {
	var filePath string
	var libraryID int64
	err := s.db.QueryRow(`SELECT file_path, library_id FROM videos WHERE id = ?`, videoID).Scan(&filePath, &libraryID)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}

	var libraryPath string
	err = s.db.QueryRow(`SELECT path FROM libraries WHERE id = ?`, libraryID).Scan(&libraryPath)
	if err != nil {
		return nil, fmt.Errorf("library not found: %w", err)
	}

	targetName := strings.ToLower(filepath.Base(filePath))
	var candidates []string
	const maxResults = 10

	_ = filepath.WalkDir(libraryPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if len(candidates) >= maxResults {
			return filepath.SkipAll
		}
		if strings.ToLower(d.Name()) == targetName {
			candidates = append(candidates, path)
		}
		return nil
	})

	return candidates, nil
}

// RemapVideoPath updates a video's file_path and clears is_missing.
// Returns an error if newPath does not exist on disk.
func (s *VideoService) RemapVideoPath(videoID int64, newPath string) error {
	if _, err := os.Stat(newPath); err != nil {
		return fmt.Errorf("file not found at new path: %w", err)
	}
	info, _ := os.Stat(newPath)
	newSize := info.Size()

	_, err := s.db.Exec(
		`UPDATE videos SET file_path = ?, file_size = ?, is_missing = 0, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		newPath, newSize, videoID,
	)
	if err != nil {
		return fmt.Errorf("failed to remap video path: %w", err)
	}
	log.Printf("[MISSING] Remapped video %d → %s", videoID, newPath)
	return nil
}

// DeleteMissingVideos bulk-deletes all videos with is_missing=1.
// If libraryID > 0, only that library is affected.
// Returns the count of deleted records.
func (s *VideoService) DeleteMissingVideos(libraryID int64) (int64, error) {
	var result interface {
		RowsAffected() (int64, error)
	}
	var err error

	if libraryID > 0 {
		result, err = s.db.Exec(`DELETE FROM videos WHERE COALESCE(is_missing, 0) = 1 AND library_id = ?`, libraryID)
	} else {
		result, err = s.db.Exec(`DELETE FROM videos WHERE COALESCE(is_missing, 0) = 1`)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to delete missing videos: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	log.Printf("[MISSING] Deleted %d missing video records (library_id=%d)", count, libraryID)

	if count > 0 {
		s.db.Exec(`UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = performers.id)`)
	}
	return count, nil
}

// FindDuplicates returns groups of videos that share the same file_size and rounded duration.
// Returns (groups, totalDuplicates, wastedBytes, error).
func (s *VideoService) FindDuplicates() ([]FileDuplicateGroup, int, int64, error) {
	rows, err := s.db.Query(`
		SELECT v.id, v.library_id, v.title, v.file_path, v.file_size,
		       COALESCE(v.duration, 0.0)      AS duration,
		       COALESCE(v.thumbnail_path, '') AS thumbnail_path,
		       COALESCE(v.content_type, '')   AS content_type,
		       COALESCE(v.resolution, '')     AS resolution
		FROM videos v
		INNER JOIN (
		    SELECT file_size, CAST(ROUND(COALESCE(duration, 0)) AS INTEGER) AS dur_int
		    FROM videos
		    WHERE COALESCE(is_missing, 0) = 0
		      AND file_size > 0
		    GROUP BY file_size, CAST(ROUND(COALESCE(duration, 0)) AS INTEGER)
		    HAVING COUNT(*) > 1
		) dups ON dups.file_size = v.file_size
		       AND dups.dur_int = CAST(ROUND(COALESCE(v.duration, 0)) AS INTEGER)
		WHERE COALESCE(v.is_missing, 0) = 0
		  AND v.file_size > 0
		ORDER BY v.file_size DESC, v.duration, v.id
	`)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	type groupKey struct {
		fileSize int64
		durInt   int64
	}
	var keyOrder []groupKey
	groups := map[groupKey]*FileDuplicateGroup{}

	for rows.Next() {
		var v FileDuplicateVideo
		if err := rows.Scan(&v.ID, &v.LibraryID, &v.Title, &v.FilePath, &v.FileSize,
			&v.Duration, &v.ThumbnailPath, &v.ContentType, &v.Resolution); err != nil {
			continue
		}
		if t := detectTopazType(v.FilePath); t != "" {
			v.IsTopazEnhanced = true
			v.TopazType = t
		}
		key := groupKey{fileSize: v.FileSize, durInt: int64(v.Duration + 0.5)}
		if g, ok := groups[key]; ok {
			g.Videos = append(g.Videos, v)
		} else {
			groups[key] = &FileDuplicateGroup{FileSize: v.FileSize, Duration: v.Duration, Videos: []FileDuplicateVideo{v}}
			keyOrder = append(keyOrder, key)
		}
	}

	result := make([]FileDuplicateGroup, 0, len(keyOrder))
	var totalDups int
	var wastedBytes int64
	for _, k := range keyOrder {
		g := groups[k]
		result = append(result, *g)
		totalDups += len(g.Videos)
		wastedBytes += g.FileSize * int64(len(g.Videos)-1)
	}
	return result, totalDups, wastedBytes, nil
}
