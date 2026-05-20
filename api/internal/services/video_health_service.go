package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	appconfig "github.com/brixen96/video-storage-ai/internal/config"
	"github.com/brixen96/video-storage-ai/internal/database"
)

// HealthCheckReport summarises what the health check found and fixed.
type HealthCheckReport struct {
	TotalVideos      int `json:"total_videos"`
	MetadataFixed    int `json:"metadata_fixed"`
	MetadataFailed   int `json:"metadata_failed"`
	ThumbnailsFixed  int `json:"thumbnails_fixed"`
	ThumbnailsFailed int `json:"thumbnails_failed"`
	PerformersLinked int `json:"performers_linked"`
	TagsApplied      int `json:"tags_applied"`
	ContentTypeSet   int `json:"content_type_set"`
}

// VideoHealthService detects and auto-fixes common video library issues.
type VideoHealthService struct {
	db     *sql.DB
	actSvc *ActivityService
	libSvc *LibraryService
}

func NewVideoHealthService() *VideoHealthService {
	return &VideoHealthService{
		db:     database.GetDB(),
		actSvc: NewActivityService(),
		libSvc: NewLibraryService(),
	}
}

// Run performs a full three-phase health check and tracks progress in an activity log.
func (s *VideoHealthService) Run(ctx context.Context) (*HealthCheckReport, error) {
	report := &HealthCheckReport{}

	activity, err := s.actSvc.StartTask(
		"video_health_check",
		"AI Health Check: scanning for fixable issues",
		map[string]interface{}{},
	)
	if err != nil {
		log.Printf("[HealthCheck] Failed to create activity: %v", err)
	}

	progress := func(pct int, msg string) {
		if activity != nil {
			s.actSvc.UpdateProgress(activity.ID, pct, msg)
		}
		log.Printf("[HealthCheck] %d%% — %s", pct, msg)
	}

	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM videos WHERE COALESCE(is_missing,0) = 0`).Scan(&report.TotalVideos)

	// Phase 1: Fix missing metadata (0–33%)
	progress(5, "Phase 1/3: Fixing missing video metadata...")
	s.fixMissingMetadata(ctx, report)
	if ctx.Err() != nil {
		s.actSvc.CompleteTask(int64(activity.ID), "Cancelled")
		return report, ctx.Err()
	}
	progress(33, fmt.Sprintf("Phase 1/3 done — %d fixed, %d failed", report.MetadataFixed, report.MetadataFailed))

	// Phase 2: Fix missing thumbnails (33–66%)
	progress(38, "Phase 2/3: Generating missing thumbnails...")
	s.fixMissingThumbnails(ctx, report)
	if ctx.Err() != nil {
		s.actSvc.CompleteTask(int64(activity.ID), "Cancelled")
		return report, ctx.Err()
	}
	progress(66, fmt.Sprintf("Phase 2/3 done — %d generated, %d failed", report.ThumbnailsFixed, report.ThumbnailsFailed))

	// Phase 3a: Fix missing content_type from file path (66–75%)
	progress(68, "Phase 3/3: Patching missing content types from file paths...")
	s.fixMissingContentType(ctx, report)
	if ctx.Err() != nil {
		s.actSvc.CompleteTask(int64(activity.ID), "Cancelled")
		return report, ctx.Err()
	}
	progress(75, fmt.Sprintf("Content type phase done — %d set", report.ContentTypeSet))

	// Phase 3b: Path analysis for unlinked videos (75–100%)
	progress(78, "Running path analysis on unlinked videos...")
	s.fixUnlinkedVideos(ctx, report)

	summary := BuildHealthSummary(report)
	if activity != nil {
		s.actSvc.CompleteTask(int64(activity.ID), summary)
	}
	log.Printf("[HealthCheck] %s", summary)
	return report, ctx.Err()
}

// BuildHealthSummary produces the one-line result string (exported so ai_companion_service can use it).
func BuildHealthSummary(r *HealthCheckReport) string {
	if r.MetadataFixed == 0 && r.ThumbnailsFixed == 0 && r.PerformersLinked == 0 && r.ContentTypeSet == 0 {
		return fmt.Sprintf("Health check complete — all %d videos look healthy", r.TotalVideos)
	}
	var parts []string
	if r.MetadataFixed > 0 {
		parts = append(parts, fmt.Sprintf("%d metadata", r.MetadataFixed))
	}
	if r.ThumbnailsFixed > 0 {
		parts = append(parts, fmt.Sprintf("%d thumbnails", r.ThumbnailsFixed))
	}
	if r.PerformersLinked > 0 {
		parts = append(parts, fmt.Sprintf("%d performer links", r.PerformersLinked))
	}
	if r.ContentTypeSet > 0 {
		parts = append(parts, fmt.Sprintf("%d content types", r.ContentTypeSet))
	}
	return fmt.Sprintf("Health check fixed %s across %d videos", strings.Join(parts, ", "), r.TotalVideos)
}

// fixMissingMetadata finds videos with no duration/resolution/file_size and re-runs ffprobe.
func (s *VideoHealthService) fixMissingMetadata(ctx context.Context, report *HealthCheckReport) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_path FROM videos
		WHERE COALESCE(is_missing, 0) = 0
		  AND (duration IS NULL OR duration = 0
		       OR resolution IS NULL OR resolution = ''
		       OR file_size IS NULL OR file_size = 0)
		ORDER BY id ASC
	`)
	if err != nil {
		log.Printf("[HealthCheck] metadata query: %v", err)
		return
	}
	defer rows.Close()

	type vid struct {
		id   int64
		path string
	}
	var videos []vid
	for rows.Next() {
		var v vid
		if rows.Scan(&v.id, &v.path) == nil {
			videos = append(videos, v)
		}
	}
	if len(videos) == 0 {
		return
	}
	log.Printf("[HealthCheck] %d videos need metadata refresh", len(videos))

	mediaSvc := NewMediaService()
	for _, v := range videos {
		if ctx.Err() != nil {
			return
		}
		meta, err := mediaSvc.ExtractMetadata(v.path)
		if err != nil {
			log.Printf("[HealthCheck] ffprobe video %d: %v", v.id, err)
			report.MetadataFailed++
			continue
		}
		res := fmt.Sprintf("%dx%d", meta.Width, meta.Height)
		if _, err := s.db.ExecContext(ctx, `
			UPDATE videos SET duration=?, file_size=?, codec=?, resolution=?, bitrate=?, updated_at=datetime('now')
			WHERE id=?`, meta.Duration, meta.Size, meta.Codec, res, meta.Bitrate, v.id); err != nil {
			report.MetadataFailed++
		} else {
			report.MetadataFixed++
		}
	}
}

// fixMissingThumbnails generates thumbnails for videos that are missing them or whose file doesn't exist.
func (s *VideoHealthService) fixMissingThumbnails(ctx context.Context, report *HealthCheckReport) {
	thumbnailBaseDir := os.Getenv("THUMBNAIL_DIR")
	if thumbnailBaseDir == "" {
		thumbnailBaseDir = filepath.Join(appconfig.DataDir(), "thumbnails")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, library_id, file_path, COALESCE(duration,0), COALESCE(thumbnail_path,'')
		FROM videos
		WHERE COALESCE(is_missing, 0) = 0
		ORDER BY id ASC
	`)
	if err != nil {
		log.Printf("[HealthCheck] thumbnail query: %v", err)
		return
	}
	defer rows.Close()

	type vid struct {
		id        int64
		libraryID int64
		path      string
		duration  float64
		thumbPath string
	}
	var videos []vid
	for rows.Next() {
		var v vid
		if rows.Scan(&v.id, &v.libraryID, &v.path, &v.duration, &v.thumbPath) != nil {
			continue
		}
		if v.thumbPath == "" {
			videos = append(videos, v)
			continue
		}
		full := filepath.Join(thumbnailBaseDir, strings.TrimPrefix(v.thumbPath, "thumbnails/"))
		if _, statErr := os.Stat(full); os.IsNotExist(statErr) {
			videos = append(videos, v)
		}
	}
	if len(videos) == 0 {
		return
	}
	log.Printf("[HealthCheck] %d videos need thumbnails", len(videos))

	// Cache library lookups to avoid redundant DB queries.
	libCache := map[int64]string{}
	mediaSvc := NewMediaService()

	for _, v := range videos {
		if ctx.Err() != nil {
			return
		}
		libPath, ok := libCache[v.libraryID]
		if !ok {
			lib, err := s.libSvc.GetByID(v.libraryID)
			if err != nil {
				report.ThumbnailsFailed++
				continue
			}
			libPath = lib.Path
			libCache[v.libraryID] = libPath
		}

		config := ThumbnailConfig{
			LibraryID:     v.libraryID,
			LibraryPath:   libPath,
			VideoFilePath: v.path,
			Duration:      v.duration,
			ThumbnailDir:  thumbnailBaseDir,
		}
		result, err := mediaSvc.GenerateThumbnailHierarchical(config)
		if err != nil {
			log.Printf("[HealthCheck] thumbnail video %d: %v", v.id, err)
			report.ThumbnailsFailed++
			continue
		}
		s.db.ExecContext(ctx, `UPDATE videos SET thumbnail_path=?, updated_at=datetime('now') WHERE id=?`,
			result.RelativePath, v.id)
		report.ThumbnailsFixed++
	}
}

// fixMissingContentType infers content_type from the file path for videos that have it empty.
// This repairs videos that were analysed before the full-path alias fix was applied.
func (s *VideoHealthService) fixMissingContentType(ctx context.Context, report *HealthCheckReport) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, file_path FROM videos
		WHERE COALESCE(is_missing, 0) = 0
		  AND (content_type IS NULL OR content_type = '')
		ORDER BY id ASC
		LIMIT 1000
	`)
	if err != nil {
		log.Printf("[HealthCheck] content_type query: %v", err)
		return
	}
	defer rows.Close()

	type vid struct {
		id   int64
		path string
	}
	var videos []vid
	for rows.Next() {
		var v vid
		if rows.Scan(&v.id, &v.path) == nil {
			videos = append(videos, v)
		}
	}
	if len(videos) == 0 {
		return
	}
	log.Printf("[HealthCheck] %d videos missing content_type", len(videos))

	aliases := map[string]string{"RL": "Regular", "A": "Zoo", "3D": "3D"}
	aliasToType := map[string]string{"regular": "regular", "zoo": "zoo", "3d": "3d"}

	for _, v := range videos {
		if ctx.Err() != nil {
			return
		}
		ct := ""
		for _, part := range strings.Split(filepath.ToSlash(v.path), "/") {
			if part == "" {
				continue
			}
			if alias, ok := aliases[strings.ToUpper(part)]; ok {
				if mapped, ok := aliasToType[strings.ToLower(alias)]; ok {
					ct = mapped
					break
				}
			}
		}
		if ct == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE videos SET content_type=?, updated_at=datetime('now') WHERE id=?`,
			ct, v.id); err == nil {
			report.ContentTypeSet++
		}
	}
}

// fixUnlinkedVideos runs path analysis on videos with no linked performers (capped at 500/run).
func (s *VideoHealthService) fixUnlinkedVideos(ctx context.Context, report *HealthCheckReport) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM videos
		WHERE COALESCE(is_missing, 0) = 0
		  AND NOT EXISTS (SELECT 1 FROM video_performers WHERE video_id = videos.id)
		ORDER BY id ASC
		LIMIT 500
	`)
	if err != nil {
		log.Printf("[HealthCheck] unlinked query: %v", err)
		return
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	log.Printf("[HealthCheck] Running path analysis on %d unlinked videos", len(ids))

	pathSvc := NewPathAnalysisService()
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		rpt, err := pathSvc.RunAutoLinkForVideo(id)
		if err != nil || rpt == nil {
			continue
		}
		report.PerformersLinked += rpt.PerformersLinked
		report.TagsApplied += rpt.TagsApplied
		report.ContentTypeSet += rpt.ContentTypeSet
	}
}
