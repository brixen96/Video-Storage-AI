package services

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
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

// aliasToContentType maps the canonical alias result to videos.content_type.
// Content type is inferred ONLY from contentTypeAliases keys (RL/A/3D) — never from
// general tag names — so "Amateur" or other tags shared across categories never affect it.
var aliasToContentType = map[string]string{
	"regular": "regular",
	"zoo":     "zoo",
	"3d":      "3d",
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

// tagEntry holds a tag id plus its category so we can pick the right variant
// when multiple tags share the same name (e.g. "Amateur" in REGULAR vs ZOO).
type tagEntry struct {
	id       int64
	category string // "", "regular", "zoo", "3d"
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
	ContentTypeSet   int  `json:"content_type_set"`
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
	tagsByName, err := s.loadTagMap()
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

		perfs, tagsMatched, studs, inferredCT := s.analyzePath(v.filePath, libRoot, performers, tagsByName, studios)

		if dryRun {
			report.PerformersLinked += len(perfs)
			report.TagsApplied += len(tagsMatched)
			report.StudiosLinked += len(studs)
			if inferredCT != "" {
				report.ContentTypeSet++
			}
			continue
		}

		// Reset all path-derived links for this video so stale entries from previous runs
		// don't accumulate (e.g. old wrong tag replaced by correct one, not duplicated).
		tx.Exec(`DELETE FROM video_performers WHERE video_id = ?`, v.id)
		tx.Exec(`DELETE FROM video_tags WHERE video_id = ?`, v.id)
		tx.Exec(`DELETE FROM video_studios WHERE video_id = ?`, v.id)
		for _, e := range perfs {
			if _, err := tx.Exec(`INSERT INTO video_performers (video_id, performer_id) VALUES (?, ?)`, v.id, e.id); err == nil {
				report.PerformersLinked++
			}
		}
		for _, e := range tagsMatched {
			if _, err := tx.Exec(`INSERT INTO video_tags (video_id, tag_id) VALUES (?, ?)`, v.id, e.id); err == nil {
				report.TagsApplied++
			}
		}
		for _, e := range studs {
			if _, err := tx.Exec(`INSERT INTO video_studios (video_id, studio_id) VALUES (?, ?)`, v.id, e.id); err == nil {
				report.StudiosLinked++
			}
		}
		// Set content_type only from explicit alias inference (RL/A/3D) — never from tag names
		if inferredCT != "" {
			if res, err := tx.Exec(`UPDATE videos SET content_type = ? WHERE id = ? AND content_type != ?`, inferredCT, v.id, inferredCT); err == nil {
				if n, _ := res.RowsAffected(); n > 0 {
					report.ContentTypeSet++
				}
			}
		}

		// Commit every 50 videos so the write lock is released frequently enough
		// for other API requests (activity updates, video queries) to proceed.
		if (i+1)%50 == 0 {
			if err := tx.Commit(); err != nil {
				if actLog != nil {
					actSvc.FailTaskLog(actLog.ID, err.Error())
				}
				return nil, fmt.Errorf("commit batch at %d: %w", i+1, err)
			}
			tx = nil

			// Brief pause between batches so other goroutines (API handlers,
			// activity-log writes) can acquire the SQLite write lock.
			time.Sleep(5 * time.Millisecond)

			if actLog != nil {
				progress := int(float64(i+1) / float64(total) * 100)
				actSvc.UpdateProgressLog(actLog.ID, progress,
					fmt.Sprintf("Processed %d/%d — %d performers, %d tags, %d studios linked so far",
						i+1, total, report.PerformersLinked, report.TagsApplied, report.StudiosLinked))
			}

			tx, err = s.db.Begin()
			if err != nil {
				if actLog != nil {
					actSvc.FailTaskLog(actLog.ID, err.Error())
				}
				return nil, fmt.Errorf("begin batch transaction at %d: %w", i+1, err)
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

		// Recalculate performer video_count to reflect newly linked videos
		if _, recalcErr := s.db.Exec(`UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = performers.id)`); recalcErr != nil {
			log.Printf("[PathAnalysis] video_count recalculate failed: %v", recalcErr)
		}

		if actLog != nil {
			actSvc.CompleteTaskLog(actLog.ID,
				fmt.Sprintf("Linked %d performers, %d tags, %d studios, set content_type on %d videos across %d total",
					report.PerformersLinked, report.TagsApplied, report.StudiosLinked, report.ContentTypeSet, total))
		}
	}

	return report, nil
}

// RunAutoLinkForVideo runs path analysis for a single video and applies results immediately.
// Returns the report and an error if the video is not found or the DB write fails.
func (s *PathAnalysisService) RunAutoLinkForVideo(videoID int64) (*PathAnalysisReport, error) {
	performers, err := s.loadEntityMap("SELECT id, name FROM performers")
	if err != nil {
		return nil, fmt.Errorf("load performers: %w", err)
	}
	tagsByName, err := s.loadTagMap()
	if err != nil {
		return nil, fmt.Errorf("load tags: %w", err)
	}
	studios, err := s.loadEntityMap("SELECT id, name FROM studios")
	if err != nil {
		return nil, fmt.Errorf("load studios: %w", err)
	}
	libPaths, err := s.loadLibraryPaths()
	if err != nil {
		return nil, fmt.Errorf("load libraries: %w", err)
	}

	var v videoRow
	err = s.db.QueryRow(
		`SELECT v.id, v.file_path, COALESCE(l.path,'') FROM videos v LEFT JOIN libraries l ON l.id = v.library_id WHERE v.id = ?`,
		videoID,
	).Scan(&v.id, &v.filePath, &v.libPath)
	if err != nil {
		return nil, fmt.Errorf("video not found: %w", err)
	}

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

	perfs, tagsMatched, studs, inferredCT := s.analyzePath(v.filePath, libRoot, performers, tagsByName, studios)

	report := &PathAnalysisReport{VideosAnalyzed: 1}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			tx.Rollback()
		}
	}()

	// Collect previously linked performers so we can update their video_count after removal.
	allAffectedPerformers := map[int64]bool{}
	if prevRows, qErr := s.db.Query(`SELECT performer_id FROM video_performers WHERE video_id = ?`, v.id); qErr == nil {
		for prevRows.Next() {
			var pid int64
			if prevRows.Scan(&pid) == nil {
				allAffectedPerformers[pid] = true
			}
		}
		prevRows.Close()
	}

	// Reset all path-derived links so stale entries from previous runs are replaced.
	tx.Exec(`DELETE FROM video_performers WHERE video_id = ?`, v.id)
	tx.Exec(`DELETE FROM video_tags WHERE video_id = ?`, v.id)
	tx.Exec(`DELETE FROM video_studios WHERE video_id = ?`, v.id)
	for _, e := range perfs {
		if _, err := tx.Exec(`INSERT INTO video_performers (video_id, performer_id) VALUES (?, ?)`, v.id, e.id); err == nil {
			report.PerformersLinked++
			allAffectedPerformers[e.id] = true
		}
	}
	for _, e := range tagsMatched {
		if _, err := tx.Exec(`INSERT INTO video_tags (video_id, tag_id) VALUES (?, ?)`, v.id, e.id); err == nil {
			report.TagsApplied++
		}
	}
	for _, e := range studs {
		if _, err := tx.Exec(`INSERT INTO video_studios (video_id, studio_id) VALUES (?, ?)`, v.id, e.id); err == nil {
			report.StudiosLinked++
		}
	}
	if inferredCT != "" {
		if res, err := tx.Exec(`UPDATE videos SET content_type = ? WHERE id = ? AND content_type != ?`, inferredCT, v.id, inferredCT); err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				report.ContentTypeSet++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	tx = nil

	// Update video_count for all affected performers (both newly linked and previously linked but now removed).
	for pid := range allAffectedPerformers {
		s.db.Exec(`UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = ?) WHERE id = ?`, pid, pid)
	}

	return report, nil
}

// analyzePath extracts performers, tags, and studios from a video's file path.
// Each path component is matched against the DB. The only hardcoded logic is
// the RL/A/3D → Regular/Zoo/3D content-type alias translation.
func (s *PathAnalysisService) analyzePath(
	filePath, libRoot string,
	performers map[string]int64,
	tagsByName map[string][]tagEntry,
	studios map[string]int64,
) (matchedPerfs, matchedTags, matchedStudios []pathEntity, contentType string) {

	// Compute path relative to library root for entity (performer/tag/studio) matching.
	rel := filepath.ToSlash(filePath)
	if libRoot != "" {
		lrSlash := filepath.ToSlash(libRoot)
		if strings.HasPrefix(strings.ToLower(rel), strings.ToLower(lrSlash)) {
			rel = rel[len(lrSlash):]
		}
	}
	rel = strings.TrimPrefix(rel, "/")
	parts := strings.Split(rel, "/")

	// Determine content type by scanning path segments for the known alias keys (RL/A/3D).
	// Relative path is tried first so that a content-type folder inside the library root wins
	// over any alias that might appear in the library root path itself (e.g. /NAS/RL/ library
	// containing an A subfolder must yield "zoo", not "regular").
	// Full path is used as a fallback for libraries whose root IS the alias folder
	// (e.g. library at /NAS/A/ — after stripping, the relative path has no alias).
	scanAlias := func(p string) string {
		for _, fp := range strings.Split(p, "/") {
			if fp == "" {
				continue
			}
			if alias, ok := contentTypeAliases[strings.ToUpper(fp)]; ok {
				if ct, ok := aliasToContentType[strings.ToLower(alias)]; ok {
					return ct
				}
			}
		}
		return ""
	}
	contentType = scanAlias(rel)
	if contentType == "" {
		contentType = scanAlias(filepath.ToSlash(filePath))
	}

	seenPerfs := map[int64]bool{}
	seenTags := map[int64]bool{}
	seenStudios := map[int64]bool{}

	for i, part := range parts {
		if part == "" {
			continue
		}
		isFilename := i == len(parts)-1

		// Translate alias for tag lookup (RL→"Regular" tag, A→"Zoo" tag, 3D→"3D" tag).
		// Content type was already determined above from the full path.
		lookup := part
		if alias, ok := contentTypeAliases[strings.ToUpper(part)]; ok {
			lookup = alias
		}

		// Tag match — prefer the tag whose category matches the video's content type.
		// This resolves same-name duplicates like "Amateur (ZOO)" vs "Amateur (REGULAR)".
		if entries, ok := tagsByName[strings.ToLower(lookup)]; ok {
			var chosen *tagEntry
			for i := range entries {
				if entries[i].category == contentType {
					chosen = &entries[i]
					break
				}
			}
			if chosen == nil {
				// Fall back: prefer uncategorised tag, then first available.
				for i := range entries {
					if entries[i].category == "" {
						chosen = &entries[i]
						break
					}
				}
				if chosen == nil && len(entries) > 0 {
					chosen = &entries[0]
				}
			}
			if chosen != nil && !seenTags[chosen.id] {
				matchedTags = append(matchedTags, pathEntity{chosen.id, lookup})
				seenTags[chosen.id] = true
			}
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
			// Normalize separators (_, -, .) to spaces so "alex_blake_video.mp4" matches "Alex Blake".
			// Skip names shorter than 4 chars to avoid false positives.
			nameWithoutExt := strings.TrimSuffix(part, filepath.Ext(part))
			normalized := normalizeSeparators(strings.ToLower(nameWithoutExt))
			for perfNameLower, id := range performers {
				if len(perfNameLower) < 4 || seenPerfs[id] {
					continue
				}
				if containsExactName(normalized, normalizeSeparators(perfNameLower)) {
					matchedPerfs = append(matchedPerfs, pathEntity{id, perfNameLower})
					seenPerfs[id] = true
				}
			}
		}
	}

	return matchedPerfs, matchedTags, matchedStudios, contentType
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

// normalizeSeparators splits on any non-alphanumeric character and rejoins with single spaces.
// "alex_blake-video.part1" → "alex blake video part1"
func normalizeSeparators(s string) string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.Join(fields, " ")
}

// loadTagMap loads all tags grouped by lowercased name, preserving category.
// When multiple tags share a name (e.g. "Amateur" in REGULAR and ZOO), analyzePath
// picks the one whose category matches the video's inferred content type.
func (s *PathAnalysisService) loadTagMap() (map[string][]tagEntry, error) {
	rows, err := s.db.Query(`SELECT id, name, COALESCE(category, '') FROM tags`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string][]tagEntry{}
	for rows.Next() {
		var id int64
		var name, category string
		if rows.Scan(&id, &name, &category) == nil {
			key := strings.ToLower(strings.TrimSpace(name))
			m[key] = append(m[key], tagEntry{id, category})
		}
	}
	return m, nil
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
