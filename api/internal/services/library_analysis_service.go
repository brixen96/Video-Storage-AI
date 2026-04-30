package services

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/brixen96/video-storage-ai/internal/database"
)

// LibraryAnalysisService analyzes library folder structure and produces
// content-type, tag, and performer assignment plans based on folder conventions:
//
//   {Root}/RL/   → content_type = regular
//   {Root}/A/    → content_type = zoo
//   {Root}/3D/   → content_type = 3d
//
// Within a content-type folder:
//   TagName/video.mp4                                  → tag "TagName"
//   TagName/Pornstars/PerformerName/video.mp4          → tag + performer
//   TagName/Pornstars/Female|Male/PerformerName/video  → tag + performer
type LibraryAnalysisService struct {
	db *sql.DB
}

func NewLibraryAnalysisService() *LibraryAnalysisService {
	return &LibraryAnalysisService{db: database.GetDB()}
}

// rootFolderToContentType maps top-level folder names to content_type values.
var rootFolderToContentType = map[string]string{
	"RL": "regular",
	"A":  "zoo",
	"3D": "3d",
}

// StructureAnalysisPlan holds all proposed changes from analyzing a library's folder structure.
type StructureAnalysisPlan struct {
	LibraryID          int64               `json:"library_id"`
	LibraryName        string              `json:"library_name"`
	LibraryRoot        string              `json:"library_root"`
	ContentTypeChanges []ContentTypeChange `json:"content_type_changes"`
	TagAssignments     []TagAssignment     `json:"tag_assignments"`
	PerformerLinks     []PerformerLink     `json:"performer_links"`
	Stats              AnalysisStats       `json:"stats"`
}

type ContentTypeChange struct {
	VideoID  int64  `json:"video_id"`
	Title    string `json:"title"`
	FilePath string `json:"file_path"`
	OldType  string `json:"old_type"`
	NewType  string `json:"new_type"`
}

type TagAssignment struct {
	VideoID  int64  `json:"video_id"`
	Title    string `json:"title"`
	FilePath string `json:"file_path"`
	TagID    int64  `json:"tag_id"`
	TagName  string `json:"tag_name"`
}

type PerformerLink struct {
	VideoID       int64  `json:"video_id"`
	Title         string `json:"title"`
	FilePath      string `json:"file_path"`
	PerformerID   int64  `json:"performer_id"`
	PerformerName string `json:"performer_name"`
}

type AnalysisStats struct {
	TotalVideos          int `json:"total_videos"`
	ContentTypeWillChange int `json:"content_type_will_change"`
	TagsWillAdd          int `json:"tags_will_add"`
	PerformersWillLink   int `json:"performers_will_link"`
	VideosUnknownRoot    int `json:"videos_unknown_root"`
}

// ApplyPlanRequest is sent by the frontend to apply a (possibly filtered) subset of the plan.
type ApplyPlanRequest struct {
	LibraryID          int64               `json:"library_id"`
	ContentTypeChanges []ContentTypeChange `json:"content_type_changes"`
	TagAssignments     []TagAssignment     `json:"tag_assignments"`
	PerformerLinks     []PerformerLink     `json:"performer_links"`
}

type ApplyPlanResult struct {
	ContentTypeChanged int `json:"content_type_changed"`
	TagsAdded          int `json:"tags_added"`
	PerformersLinked   int `json:"performers_linked"`
	Total              int `json:"total"`
}

// AnalyzeLibraryStructure scans a library's video paths and returns a plan of changes.
func (s *LibraryAnalysisService) AnalyzeLibraryStructure(libraryID int64) (*StructureAnalysisPlan, error) {
	// Get library
	var libName, libPath string
	err := s.db.QueryRow(`SELECT name, path FROM libraries WHERE id = ?`, libraryID).Scan(&libName, &libPath)
	if err != nil {
		return nil, fmt.Errorf("library not found: %w", err)
	}

	// Load all tags: lowercased name → {id, canonical name}
	type nameID struct {
		ID   int64
		Name string
	}
	tagMap := make(map[string]nameID)
	tagRows, _ := s.db.Query(`SELECT id, name FROM tags`)
	if tagRows != nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var id int64
			var name string
			if tagRows.Scan(&id, &name) == nil {
				tagMap[strings.ToLower(name)] = nameID{id, name}
			}
		}
		tagRows.Close()
	}

	// Load all performers: lowercased name → {id, canonical name}
	perfMap := make(map[string]nameID)
	perfRows, _ := s.db.Query(`SELECT id, name FROM performers`)
	if perfRows != nil {
		defer perfRows.Close()
		for perfRows.Next() {
			var id int64
			var name string
			if perfRows.Scan(&id, &name) == nil {
				perfMap[strings.ToLower(name)] = nameID{id, name}
			}
		}
		perfRows.Close()
	}

	// Load existing video_tags for this library to skip duplicates
	existingTags := make(map[string]bool) // "videoID:tagID"
	etRows, _ := s.db.Query(`
		SELECT vt.video_id, vt.tag_id
		FROM video_tags vt
		INNER JOIN videos v ON v.id = vt.video_id
		WHERE v.library_id = ?`, libraryID)
	if etRows != nil {
		defer etRows.Close()
		for etRows.Next() {
			var vid, tid int64
			if etRows.Scan(&vid, &tid) == nil {
				existingTags[fmt.Sprintf("%d:%d", vid, tid)] = true
			}
		}
		etRows.Close()
	}

	// Load existing video_performers for this library
	existingPerfs := make(map[string]bool) // "videoID:performerID"
	epRows, _ := s.db.Query(`
		SELECT vp.video_id, vp.performer_id
		FROM video_performers vp
		INNER JOIN videos v ON v.id = vp.video_id
		WHERE v.library_id = ?`, libraryID)
	if epRows != nil {
		defer epRows.Close()
		for epRows.Next() {
			var vid, pid int64
			if epRows.Scan(&vid, &pid) == nil {
				existingPerfs[fmt.Sprintf("%d:%d", vid, pid)] = true
			}
		}
		epRows.Close()
	}

	// Load all non-missing videos for this library
	videoRows, err := s.db.Query(`
		SELECT id, title, file_path, COALESCE(content_type, 'regular')
		FROM videos
		WHERE library_id = ? AND COALESCE(is_missing, 0) = 0`, libraryID)
	if err != nil {
		return nil, fmt.Errorf("failed to load videos: %w", err)
	}
	defer videoRows.Close()

	plan := &StructureAnalysisPlan{
		LibraryID:   libraryID,
		LibraryName: libName,
		LibraryRoot: libPath,
	}
	libRoot := strings.TrimRight(filepath.ToSlash(libPath), "/")

	for videoRows.Next() {
		var id int64
		var title, filePath, currentType string
		if err := videoRows.Scan(&id, &title, &filePath, &currentType); err != nil {
			continue
		}
		plan.Stats.TotalVideos++

		contentType, tagFolderName, performerFolderName := parseLibraryPath(libRoot, filePath)
		if contentType == "" {
			plan.Stats.VideosUnknownRoot++
			continue
		}

		// Content type change?
		if contentType != currentType {
			plan.ContentTypeChanges = append(plan.ContentTypeChanges, ContentTypeChange{
				VideoID:  id,
				Title:    title,
				FilePath: filePath,
				OldType:  currentType,
				NewType:  contentType,
			})
		}

		// Tag assignment?
		if tagFolderName != "" {
			if tag, ok := tagMap[strings.ToLower(tagFolderName)]; ok {
				key := fmt.Sprintf("%d:%d", id, tag.ID)
				if !existingTags[key] {
					plan.TagAssignments = append(plan.TagAssignments, TagAssignment{
						VideoID:  id,
						Title:    title,
						FilePath: filePath,
						TagID:    tag.ID,
						TagName:  tag.Name,
					})
					existingTags[key] = true // prevent double-adding same video/tag within the plan
				}
			}
		}

		// Performer link?
		if performerFolderName != "" {
			if perf, ok := perfMap[strings.ToLower(performerFolderName)]; ok {
				key := fmt.Sprintf("%d:%d", id, perf.ID)
				if !existingPerfs[key] {
					plan.PerformerLinks = append(plan.PerformerLinks, PerformerLink{
						VideoID:       id,
						Title:         title,
						FilePath:      filePath,
						PerformerID:   perf.ID,
						PerformerName: perf.Name,
					})
					existingPerfs[key] = true
				}
			}
		}
	}
	videoRows.Close()

	plan.Stats.ContentTypeWillChange = len(plan.ContentTypeChanges)
	plan.Stats.TagsWillAdd = len(plan.TagAssignments)
	plan.Stats.PerformersWillLink = len(plan.PerformerLinks)

	return plan, nil
}

// ApplyPlan applies the given subset of changes to the database.
func (s *LibraryAnalysisService) ApplyPlan(req *ApplyPlanRequest) (*ApplyPlanResult, error) {
	result := &ApplyPlanResult{}

	for _, change := range req.ContentTypeChanges {
		if _, err := s.db.Exec(
			`UPDATE videos SET content_type = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
			change.NewType, change.VideoID,
		); err != nil {
			return result, fmt.Errorf("failed to update content type for video %d: %w", change.VideoID, err)
		}
		result.ContentTypeChanged++
	}

	for _, ta := range req.TagAssignments {
		if _, err := s.db.Exec(
			`INSERT OR IGNORE INTO video_tags (video_id, tag_id) VALUES (?, ?)`,
			ta.VideoID, ta.TagID,
		); err != nil {
			return result, fmt.Errorf("failed to assign tag: %w", err)
		}
		result.TagsAdded++
	}

	for _, pl := range req.PerformerLinks {
		if _, err := s.db.Exec(
			`INSERT OR IGNORE INTO video_performers (video_id, performer_id) VALUES (?, ?)`,
			pl.VideoID, pl.PerformerID,
		); err != nil {
			return result, fmt.Errorf("failed to link performer: %w", err)
		}
		// Update performer video_count
		s.db.Exec(`UPDATE performers SET video_count = (SELECT COUNT(*) FROM video_performers WHERE performer_id = ?) WHERE id = ?`,
			pl.PerformerID, pl.PerformerID)
		result.PerformersLinked++
	}

	result.Total = result.ContentTypeChanged + result.TagsAdded + result.PerformersLinked
	return result, nil
}

// parseLibraryPath extracts content_type, tag folder, and performer folder from a video path.
//
// Supported structures (relative to library root):
//
//	RL/TagName/video.mp4
//	RL/TagName/Pornstars/PerformerName/video.mp4
//	RL/TagName/Pornstars/Female|Male/PerformerName/video.mp4
//	A/TagName/video.mp4   (etc.)
func parseLibraryPath(libraryRoot, filePath string) (contentType, tagFolder, performerFolder string) {
	fp := filepath.ToSlash(filePath)
	root := strings.TrimRight(filepath.ToSlash(libraryRoot), "/")

	if !strings.HasPrefix(fp, root+"/") {
		return
	}
	rel := fp[len(root)+1:]
	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return
	}

	ct, ok := rootFolderToContentType[strings.ToUpper(parts[0])]
	if !ok {
		return
	}
	contentType = ct

	if len(parts) < 3 {
		// Video sits directly inside the content-type folder — no tag
		return
	}

	folder1 := parts[1]

	// If second folder is "Pornstars" (no tag folder in path)
	if strings.EqualFold(folder1, "Pornstars") {
		performerFolder = extractPerformerFromPornstars(parts, 2)
		return
	}

	// Second folder is a tag
	tagFolder = folder1

	// Third folder may be "Pornstars"
	if len(parts) >= 4 && strings.EqualFold(parts[2], "Pornstars") {
		performerFolder = extractPerformerFromPornstars(parts, 3)
	}

	return
}

// extractPerformerFromPornstars resolves the performer folder name starting at index i in parts.
// Handles both direct: parts[i] = PerformerName
// and gendered:        parts[i] = Female/Male, parts[i+1] = PerformerName
func extractPerformerFromPornstars(parts []string, i int) string {
	if i >= len(parts) {
		return ""
	}
	if strings.EqualFold(parts[i], "Female") || strings.EqualFold(parts[i], "Male") {
		if i+1 < len(parts) {
			// parts[i+1] is the performer name folder (parts after that are files)
			candidate := parts[i+1]
			// Make sure it's not a file (has an extension)
			if !strings.Contains(candidate, ".") {
				return candidate
			}
		}
		return ""
	}
	// Direct performer folder — make sure it's not a file
	candidate := parts[i]
	if !strings.Contains(candidate, ".") {
		return candidate
	}
	return ""
}
