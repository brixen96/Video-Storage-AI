package services

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/brixen96/video-storage-ai/internal/database"
	"github.com/brixen96/video-storage-ai/internal/models"
)

// OnScanComplete is called by VideoService after a scan adds new videos.
// AICompanionService sets this to trigger path analysis automatically.
var OnScanComplete func(libraryID int64, addedCount int)

// contentTypeAliases maps folder names to canonical tag names.
// These three are the only hardcoded mappings — user confirmed they never change.
var contentTypeAliases = map[string]string{
	"RL": "Regular",
	"A":  "Zoo",
	"3D": "3D",
}

// PathAnalysisService links performers, tags, and studios to videos
// by matching folder and filename components against the database.
type PathAnalysisService struct {
	db *sql.DB
}

func NewPathAnalysisService() *PathAnalysisService {
	return &PathAnalysisService{db: database.GetDB()}
}

type pathEntity struct {
	id   int64
	name string
}

type videoRow struct {
	id       int64
	filePath string
	libPath  string
}

// PathAnalysisReport summarises what was (or would be) applied.
type PathAnalysisReport struct {
	VideosAnalyzed   int  `json:"videos_analyzed"`
	PerformersLinked int  `json:"performers_linked"`
	TagsApplied      int  `json:"tags_applied"`
	StudiosLinked    int  `json:"studios_linked"`
	DryRun           bool `json:"dry_run"`
}

// RunAutoLink analyses video paths and applies matches to the database.
// If libraryID == 0, all libraries are processed.
// If onlyUnlinked is true, only videos with no performers yet are processed.
// If dryRun is true, nothing is written — only counts are returned.
func (s *PathAnalysisService) RunAutoLink(libraryID int64, onlyUnlinked, dryRun bool) (*PathAnalysisReport, error) {
	performers, err := s.loadEntityMap("SELECT id, name FROM performers")
	if err != nil {
		return nil, fmt.Errorf("load performers: %w", err)
	}
	tags, err := s.loadEntityMap("SELECT id, name FROM tags")
	if err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	studios, err := s.loadEntityMap("SELECT id, name FROM studios")
	if err != nil {
		return nil, fmt.Errorf("load studios: %w", err)
	}

	// Load library root paths sorted longest-first so we strip the most specific match
	libPaths, err := s.loadLibraryPaths()
	if err != nil {
		return nil, fmt.Errorf("load libraries: %w", err)
	}

	videos, err := s.loadVideos(libraryID, onlyUnlinked)
	if err != nil {
		return nil, fmt.Errorf("load videos: %w", err)
	}

	report := &PathAnalysisReport{DryRun: dryRun}
	total := len(videos)

	// Track progress in the Activity page (skip for dry runs)
	var actSvc *ActivityService
	var actLog *models.ActivityLog
	if !dryRun && total > 0 {
		actSvc = NewActivityService()
		scope := "all libraries"
		if libraryID > 0 {
			scope = fmt.Sprintf("library %d", libraryID)
		}
		actLog, _ = actSvc.StartTaskLog("path_analysis",
			fmt.Sprintf("Auto-linking by path — %d videos (%s)", total, scope),
			map[string]interface{}{"library_id": libraryID, "total_videos": total, "only_unlinked": onlyUnlinked},
		)
	}

	var tx *sql.Tx
	if !dryRun {
		tx, err = s.db.Begin()
		if err != nil {
			if actLog != nil {
				actSvc.FailTaskLog(actLog.ID, err.Error())
			}
			return nil, fmt.Errorf("begin transaction: %w", err)
		}
		defer func() {
			if tx != nil {
				tx.Rollback()
			}
		}()
	}

	for i, v := range videos {
		report.VideosAnalyzed++

		// Update activity progress every 1000 videos
		if actLog != nil && total > 0 && (i+1)%1000 == 0 {
			progress := int(float64(i+1) / float64(total) * 100)
			if logErr := actSvc.UpdateProgressLog(actLog.ID, progress,
				fmt.Sprintf("Processed %d/%d — %d performers, %d tags, %d studios linked so far",
					i+1, total, report.PerformersLinked, report.TagsApplied, report.StudiosLinked)); logErr != nil {
				log.Printf("[PathAnalysis] progress update failed: %v", logErr)
			}
		}

		// Find the best (longest-matching) library root for this video
		libRoot := v.libPath
		if libRoot == "" {
			for _, lp := range libPaths {
				if strings.HasPrefix(strings.ToLower(filepath.ToSlash(v.filePath)),
					strings.ToLower(filepath.ToSlash(lp))) {
					libRoot = lp
					break
				}
			}
		}

		perfs, tagsMatched, studs := s.analyzePath(v.filePath, libRoot, performers, tags, studios)

		if dryRun {
			report.PerformersLinked += len(perfs)
			report.TagsApplied += len(tagsMatched)
			report.StudiosLinked += len(studs)
			continue
		}

		for _, e := range perfs {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO video_performers (video_id, performer_id) VALUES (?, ?)`, v.id, e.id); err == nil {
				report.PerformersLinked++
			}
		}
		for _, e := range tagsMatched {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO video_tags (video_id, tag_id) VALUES (?, ?)`, v.id, e.id); err == nil {
				report.TagsApplied++
			}
		}
		for _, e := range studs {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO video_studios (video_id, studio_id) VALUES (?, ?)`, v.id, e.id); err == nil {
				report.StudiosLinked++
			}
		}
	}

	if !dryRun {
		if err := tx.Commit(); err != nil {
			if actLog != nil {
				actSvc.FailTaskLog(actLog.ID, err.Error())
			}
			return nil, fmt.Errorf("commit: %w", err)
		}
		tx = nil
		if actLog != nil {
			actSvc.CompleteTaskLog(actLog.ID,
				fmt.Sprintf("Linked %d performers, %d tags, %d studios across %d videos",
					report.PerformersLinked, report.TagsApplied, report.StudiosLinked, total))
		}
	}

	return report, nil
}

// analyzePath extracts performers, tags, and studios from a video's file path.
// Each path component is matched against the DB. The only hardcoded logic is
// the RL/A/3D → Regular/Zoo/3D content-type alias translation.
func (s *PathAnalysisService) analyzePath(
	filePath, libRoot string,
	performers, tags, studios map[string]int64,
) (matchedPerfs, matchedTags, matchedStudios []pathEntity) {

	// Compute path relative to library root for cleaner parsing
	rel := filepath.ToSlash(filePath)
	if libRoot != "" {
		lrSlash := filepath.ToSlash(libRoot)
		if strings.HasPrefix(strings.ToLower(rel), strings.ToLower(lrSlash)) {
			rel = rel[len(lrSlash):]
		}
	}
	rel = strings.TrimPrefix(rel, "/")
	parts := strings.Split(rel, "/")

	seenPerfs := map[int64]bool{}
	seenTags := map[int64]bool{}
	seenStudios := map[int64]bool{}

	for i, part := range parts {
		if part == "" {
			continue
		}
		isFilename := i == len(parts)-1

		// Apply content-type alias (RL→Regular, A→Zoo, 3D→3D)
		lookup := part
		if alias, ok := contentTypeAliases[part]; ok {
			lookup = alias
		}

		// Tag match — use translated name so "RL" finds the "Regular" tag
		if id, ok := tags[strings.ToLower(lookup)]; ok && !seenTags[id] {
			matchedTags = append(matchedTags, pathEntity{id, lookup})
			seenTags[id] = true
		}

		if !isFilename {
			// Folder: exact name match for performers and studios
			if id, ok := performers[strings.ToLower(part)]; ok && !seenPerfs[id] {
				matchedPerfs = append(matchedPerfs, pathEntity{id, part})
				seenPerfs[id] = true
			}
			if id, ok := studios[strings.ToLower(part)]; ok && !seenStudios[id] {
				matchedStudios = append(matchedStudios, pathEntity{id, part})
				seenStudios[id] = true
			}
		} else {
			// Filename: scan for performer names using word-boundary matching.
			// Skip names shorter than 4 chars to avoid false positives.
			nameWithoutExt := strings.TrimSuffix(part, filepath.Ext(part))
			lower := strings.ToLower(nameWithoutExt)
			for perfNameLower, id := range performers {
				if len(perfNameLower) < 4 || seenPerfs[id] {
					continue
				}
				if containsExactName(lower, perfNameLower) {
					matchedPerfs = append(matchedPerfs, pathEntity{id, perfNameLower})
					seenPerfs[id] = true
				}
			}
		}
	}

	return matchedPerfs, matchedTags, matchedStudios
}

// containsExactName returns true if name appears in text surrounded by non-letter/digit boundaries.
func containsExactName(text, name string) bool {
	idx := strings.Index(text, name)
	if idx == -1 {
		return false
	}
	if idx > 0 && isNameChar(rune(text[idx-1])) {
		return false
	}
	end := idx + len(name)
	if end < len(text) && isNameChar(rune(text[end])) {
		return false
	}
	return true
}

func isNameChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (s *PathAnalysisService) loadEntityMap(query string) (map[string]int64, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]int64{}
	for rows.Next() {
		var id int64
		var name string
		if rows.Scan(&id, &name) == nil {
			m[strings.ToLower(strings.TrimSpace(name))] = id
		}
	}
	return m, nil
}

func (s *PathAnalysisService) loadLibraryPaths() ([]string, error) {
	rows, err := s.db.Query(`SELECT path FROM libraries ORDER BY LENGTH(path) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if rows.Scan(&p) == nil {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

func (s *PathAnalysisService) loadVideos(libraryID int64, onlyUnlinked bool) ([]videoRow, error) {
	var conds []string
	var args []interface{}

	if onlyUnlinked {
		conds = append(conds, `NOT EXISTS (SELECT 1 FROM video_performers vp WHERE vp.video_id = v.id)`)
	}
	if libraryID > 0 {
		conds = append(conds, `v.library_id = ?`)
		args = append(args, libraryID)
	}

	q := `SELECT v.id, v.file_path, COALESCE(l.path,'') FROM videos v LEFT JOIN libraries l ON l.id = v.library_id`
	if len(conds) > 0 {
		q += ` WHERE ` + strings.Join(conds, ` AND `)
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []videoRow
	for rows.Next() {
		var r videoRow
		if rows.Scan(&r.id, &r.filePath, &r.libPath) == nil {
			result = append(result, r)
		}
	}
	return result, nil
}
