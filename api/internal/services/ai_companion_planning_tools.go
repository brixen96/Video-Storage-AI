package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/brixen96/video-storage-ai/internal/models"
)

// ImprovementAction represents a single action within a plan.
type ImprovementAction struct {
	ID            int    `json:"id"`
	Type          string `json:"type"`            // create_folder, move_video, tag_video, rename_folder
	Status        string `json:"status"`          // pending, completed, skipped, failed
	Description   string `json:"description"`
	VideoID       int64  `json:"video_id,omitempty"`
	SourcePath    string `json:"source_path,omitempty"`
	TargetPath    string `json:"target_path,omitempty"`
	LibraryID     int64  `json:"library_id,omitempty"`
	TagName       string `json:"tag_name,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ImprovementPlan represents a complete library improvement plan.
type ImprovementPlan struct {
	ID              int                `json:"id"`
	LibraryID       int64              `json:"library_id"`
	LibraryName     string             `json:"library_name"`
	PlanName        string             `json:"plan_name"`
	Status          string             `json:"status"` // draft, executing, completed, failed
	Actions         []ImprovementAction `json:"actions"`
	TotalActions    int                `json:"total_actions"`
	CompletedActions int              `json:"completed_actions"`
	CreatedAt       time.Time          `json:"created_at"`
	ExecutedAt      *time.Time         `json:"executed_at,omitempty"`
}

func (s *AICompanionService) registerPlanningTools() {
	// Tool 1: plan_structure_improvements
	// Scans a library deeply, identifies issues, and builds a concrete improvement plan.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "plan_structure_improvements",
			Description: `Analyze a library's file/folder structure and build a concrete improvement plan. This is READ-ONLY — no files are moved. The plan includes: folder creation, video reorganization, tagging, and cleanup actions. Use this in PLANNING mode to build plans before executing them.`,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"library_name": map[string]interface{}{
						"type":        "string",
						"description": "Library name or partial name to analyze",
					},
					"focus": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"full", "structure", "tagging", "duplicates", "orphans"},
						"description": "Focus area: full (default), structure (folder layout), tagging (missing tags), duplicates (duplicate files), orphans (DB entries without files)",
					},
				},
				"required": []string{"library_name"},
			},
		},
	}, s.toolPlanStructureImprovements)

	// Tool 2: execute_plan
	// Executes a previously saved improvement plan, one action at a time with safety checks.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "execute_plan",
			Description: `Execute a previously saved improvement plan by its ID. Runs all pending actions with safety checks. Returns progress after each batch. Only execute plans that the user has reviewed and approved.`,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plan_id": map[string]interface{}{
						"type":        "number",
						"description": "ID of the plan to execute",
					},
					"dry_run": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, simulate execution without making changes (default: true)",
					},
				},
				"required": []string{"plan_id"},
			},
		},
	}, s.toolExecutePlan)

	// Tool 3: list_plans
	// Lists all saved improvement plans, with optional status filter.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "list_plans",
			Description: "List all saved improvement plans. Shows plan name, library, status, and action counts.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"all", "draft", "executing", "completed", "failed"},
						"description": "Filter by status (default: all)",
					},
				},
				"required": []string{},
			},
		},
	}, s.toolListPlans)

	// Tool 4: get_plan
	// Retrieves a full plan with all its actions for review.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_plan",
			Description: "Get the full details of an improvement plan, including all actions. Use this to review a plan before executing it.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plan_id": map[string]interface{}{
						"type":        "number",
						"description": "ID of the plan to retrieve",
					},
				},
				"required": []string{"plan_id"},
			},
		},
	}, s.toolGetPlan)

	// Tool 5: delete_plan
	// Deletes a draft plan that the user no longer wants.
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "delete_plan",
			Description: "Delete a draft improvement plan. Cannot delete plans that are currently executing or already completed.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"plan_id": map[string]interface{}{
						"type":        "number",
						"description": "ID of the plan to delete",
					},
				},
				"required": []string{"plan_id"},
			},
		},
	}, s.toolDeletePlan)
}

// ==================== PLAN STRUCTURE IMPROVEMENTS ====================

func (s *AICompanionService) toolPlanStructureImprovements(args map[string]interface{}) (interface{}, error) {
	libraryName, _ := args["library_name"].(string)
	focus, _ := args["focus"].(string)
	if focus == "" {
		focus = "full"
	}

	// Resolve library
	var libID int64
	var libPath, libNameFull string
	err := s.db.QueryRow("SELECT id, name, path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libID, &libNameFull, &libPath)
	if err != nil {
		return map[string]interface{}{"error": "library not found: " + libraryName}, nil
	}

	var actions []ImprovementAction
	actionID := 0

	// ===== STRUCTURE ANALYSIS =====
	if focus == "full" || focus == "structure" {
		structActions, err := s.analyzeStructureForPlan(libID, libPath, libNameFull)
		if err != nil {
			log.Printf("[AI] Structure analysis error: %v", err)
		} else {
			for i := range structActions {
				actionID++
				structActions[i].ID = actionID
			}
			actions = append(actions, structActions...)
		}
	}

	// ===== TAGGING ANALYSIS =====
	if focus == "full" || focus == "tagging" {
		tagActions, err := s.analyzeTaggingForPlan(libID, libPath)
		if err != nil {
			log.Printf("[AI] Tagging analysis error: %v", err)
		} else {
			for i := range tagActions {
				actionID++
				tagActions[i].ID = actionID
			}
			actions = append(actions, tagActions...)
		}
	}

	// ===== DUPLICATE ANALYSIS =====
	if focus == "full" || focus == "duplicates" {
		dupActions, err := s.analyzeDuplicatesForPlan(libID, libPath)
		if err != nil {
			log.Printf("[AI] Duplicate analysis error: %v", err)
		} else {
			for i := range dupActions {
				actionID++
				dupActions[i].ID = actionID
			}
			actions = append(actions, dupActions...)
		}
	}

	// ===== ORPHAN ANALYSIS =====
	if focus == "full" || focus == "orphans" {
		orphActions, err := s.analyzeOrphansForPlan(libID)
		if err != nil {
			log.Printf("[AI] Orphan analysis error: %v", err)
		} else {
			for i := range orphActions {
				actionID++
				orphActions[i].ID = actionID
			}
			actions = append(actions, orphActions...)
		}
	}

	if len(actions) == 0 {
		return map[string]interface{}{
			"library":  libNameFull,
			"message":  "Library structure looks good — no improvements needed right now.",
			"actions":  []ImprovementAction{},
			"total":    0,
		}, nil
	}

	// Save plan to DB
	planName := fmt.Sprintf("%s improvement plan", libNameFull)
	planData, _ := json.Marshal(actions)
	var planID int64
	err = s.db.QueryRow(
		`INSERT INTO companion_plans (library_id, plan_name, plan_data, status, total_actions, created_at)
		 VALUES (?, ?, ?, 'draft', ?, ?) RETURNING id`,
		libID, planName, string(planData), len(actions), time.Now(),
	).Scan(&planID)
	if err != nil {
		// Fallback: try without RETURNING (SQLite < 3.35)
		res, insErr := s.db.Exec(
			`INSERT INTO companion_plans (library_id, plan_name, plan_data, status, total_actions, created_at)
			 VALUES (?, ?, ?, 'draft', ?, ?)`,
			libID, planName, string(planData), len(actions), time.Now(),
		)
		if insErr != nil {
			return map[string]interface{}{
				"library": libNameFull, "message": "Plan created but could not be saved.",
				"actions": actions, "total": len(actions),
			}, nil
		}
		planID, _ = res.LastInsertId()
	}

	// Group actions by type for summary
	typeCounts := map[string]int{}
	for _, a := range actions {
		typeCounts[a.Type]++
	}

	return map[string]interface{}{
		"plan_id":     planID,
		"library":     libNameFull,
		"plan_name":   planName,
		"status":      "draft",
		"total":       len(actions),
		"type_counts": typeCounts,
		"actions":     actions,
		"message":     fmt.Sprintf("Plan created with %d actions. Review it with get_plan, then execute with execute_plan.", len(actions)),
	}, nil
}

// analyzeStructureForPlan finds folder organization improvements.
func (s *AICompanionService) analyzeStructureForPlan(libID int64, libPath, libName string) ([]ImprovementAction, error) {
	var actions []ImprovementAction
	actionID := 0

	// Find videos at library root that should be in subfolders
	rows, err := s.db.Query(`
		SELECT v.id, v.title, v.file_path
		FROM videos v WHERE v.library_id = ? AND v.file_path = ?
		ORDER BY v.title LIMIT 200`, libID, libPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rootVideo struct {
		id    int64
		title string
		path  string
	}
	var rootVideos []rootVideo
	for rows.Next() {
		var rv rootVideo
		if rows.Scan(&rv.id, &rv.title, &rv.path) == nil {
			rootVideos = append(rootVideos, rv)
		}
	}

	// For each root-level video, try to find a good performer-based subfolder
	for _, rv := range rootVideos {
		// Find performers for this video
		perfRows, _ := s.db.Query(
			`SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?`, rv.id)
		var performers []string
		for perfRows.Next() {
			var n string
			perfRows.Scan(&n)
			performers = append(performers, n)
		}
		perfRows.Close()

		if len(performers) == 0 {
			continue
		}

		// Check if a folder for this performer already exists
		perfName := performers[0]
		perfFolder := filepath.Join(libPath, "Performers", perfName)
		if _, err := os.Stat(perfFolder); err == nil {
			// Folder exists — suggest move
			actionID++
			actions = append(actions, ImprovementAction{
				ID:          actionID,
				Type:        "move_video",
				Status:      "pending",
				Description: fmt.Sprintf("Move '%s' to Performers/%s/ (performer folder exists)", truncateTitle(rv.title, 40), perfName),
				VideoID:     rv.id,
				SourcePath:  rv.path,
				TargetPath:  filepath.Join("Performers", perfName),
				LibraryID:   libID,
			})
		}
	}

	// Find empty or near-empty folders that could be cleaned up — limit depth for performance
	maxFolders := 0
	_ = filepath.Walk(libPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || !info.IsDir() || path == libPath {
			return nil
		}
		relPath, _ := filepath.Rel(libPath, path)
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) > 2 {
			return filepath.SkipDir
		}
		maxFolders++
		if maxFolders > 500 {
			return filepath.SkipDir
		}
		var count int
		s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE library_id = ? AND file_path LIKE ?",
			libID, "%"+string(filepath.Separator)+relPath+"%").Scan(&count)
		if count == 0 {
			// Check if directory is actually empty on disk
			entries, err := os.ReadDir(path)
			if err == nil && len(entries) == 0 {
				actionID++
				actions = append(actions, ImprovementAction{
					ID:          actionID,
					Type:        "create_folder",
					Status:      "pending",
					Description: fmt.Sprintf("Remove empty folder: %s", filepath.ToSlash(relPath)),
					SourcePath:  filepath.ToSlash(relPath),
					LibraryID:   libID,
				})
			}
		}
		return nil
	})

	return actions, nil
}

// analyzeTaggingForPlan finds videos that are missing tags or performer links.
func (s *AICompanionService) analyzeTaggingForPlan(libID int64, libPath string) ([]ImprovementAction, error) {
	var actions []ImprovementAction
	actionID := 0

	// Find videos with no tags
	rows, err := s.db.Query(`
		SELECT v.id, v.title, v.file_path
		FROM videos v
		LEFT JOIN video_tags vt ON v.id = vt.video_id
		WHERE v.library_id = ? AND vt.video_id IS NULL
		ORDER BY v.created_at DESC LIMIT 50`, libID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var title, fp string
		if rows.Scan(&id, &title, &fp) == nil {
			actionID++
			actions = append(actions, ImprovementAction{
				ID:          actionID,
				Type:        "tag_video",
				Status:      "pending",
				Description: fmt.Sprintf("Tag '%s' based on performer history", truncateTitle(title, 40)),
				VideoID:     id,
				SourcePath:  fp,
				LibraryID:   libID,
			})
		}
	}

	// Find videos with no performers
	rows2, err := s.db.Query(`
		SELECT v.id, v.title, v.file_path
		FROM videos v
		LEFT JOIN video_performers vp ON v.id = vp.video_id
		WHERE v.library_id = ? AND vp.video_id IS NULL
		ORDER BY v.created_at DESC LIMIT 50`, libID)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var id int64
		var title, fp string
		if rows2.Scan(&id, &title, &fp) == nil {
			// Check if we already have a tag action for this video
			alreadyTagged := false
			for _, a := range actions {
				if a.VideoID == id {
					alreadyTagged = true
					break
				}
			}
			if !alreadyTagged {
				actionID++
				actions = append(actions, ImprovementAction{
					ID:          actionID,
					Type:        "tag_video",
					Status:      "pending",
					Description: fmt.Sprintf("Link performer to '%s' (no performers assigned)", truncateTitle(title, 40)),
					VideoID:     id,
					SourcePath:  fp,
					LibraryID:   libID,
				})
			}
		}
	}

	return actions, nil
}

// analyzeDuplicatesForPlan finds duplicate videos across libraries.
func (s *AICompanionService) analyzeDuplicatesForPlan(libID int64, libPath string) ([]ImprovementAction, error) {
	var actions []ImprovementAction
	actionID := 0

	// Find videos with the same title in different libraries
	rows, err := s.db.Query(`
		SELECT v1.id, v1.title, v1.file_path, v2.id, v2.file_path, l.name
		FROM videos v1
		JOIN videos v2 ON LOWER(v1.title) = LOWER(v2.title) AND v1.id != v2.id
		JOIN libraries l ON v2.library_id = l.id
		WHERE v1.library_id = ? AND v1.created_at <= v2.created_at
		ORDER BY v1.title
		LIMIT 30`, libID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[int64]bool{}
	for rows.Next() {
		var id1, id2 int64
		var title, fp1, fp2, libName string
		if rows.Scan(&id1, &title, &fp1, &id2, &fp2, &libName) == nil {
			if seen[id1] {
				continue
			}
			seen[id1] = true
			actionID++
			actions = append(actions, ImprovementAction{
				ID:          actionID,
				Type:        "move_video",
				Status:      "pending",
				Description: fmt.Sprintf("Duplicate: '%s' exists in %s — consider removing one", truncateTitle(title, 40), libName),
				VideoID:     id1,
				SourcePath:  fp1,
				TargetPath:  fp2,
				LibraryID:   libID,
			})
		}
	}

	return actions, nil
}

// analyzeOrphansForPlan finds DB entries with missing files on disk.
func (s *AICompanionService) analyzeOrphansForPlan(libID int64) ([]ImprovementAction, error) {
	var actions []ImprovementAction
	actionID := 0

	rows, err := s.db.Query(`
		SELECT id, title, file_path FROM videos
		WHERE library_id = ? AND is_missing = 1
		ORDER BY title LIMIT 50`, libID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var title, fp string
		if rows.Scan(&id, &title, &fp) == nil {
			actionID++
			actions = append(actions, ImprovementAction{
				ID:          actionID,
				Type:        "move_video",
				Status:      "pending",
				Description: fmt.Sprintf("Orphaned DB entry: '%s' — file missing from disk", truncateTitle(title, 40)),
				VideoID:     id,
				SourcePath:  fp,
				LibraryID:   libID,
			})
		}
	}

	return actions, nil
}

// ==================== EXECUTE PLAN ====================

func (s *AICompanionService) toolExecutePlan(args map[string]interface{}) (interface{}, error) {
	planID := int64(intArg(args, "plan_id", 0))
	dryRun := true
	if v, ok := args["dry_run"].(bool); ok {
		dryRun = v
	}
	if v, ok := args["dry_run"].(string); ok && v == "false" {
		dryRun = false
	}

	// Load plan
	var libID int64
	var planName, planDataStr, status string
	var totalActions, completedActions int
	var createdAt time.Time
	err := s.db.QueryRow(
		`SELECT library_id, plan_name, plan_data, status, total_actions, completed_actions, created_at
		 FROM companion_plans WHERE id = ?`, planID,
	).Scan(&libID, &planName, &planDataStr, &status, &totalActions, &completedActions, &createdAt)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("plan %d not found", planID)}, nil
	}

	if status == "completed" {
		return map[string]interface{}{"error": "plan is already completed"}, nil
	}

	var actions []ImprovementAction
	if err := json.Unmarshal([]byte(planDataStr), &actions); err != nil {
		return map[string]interface{}{"error": "failed to parse plan data"}, nil
	}

	// Update status to executing
	now := time.Now()
	s.db.Exec(`UPDATE companion_plans SET status = 'executing', executed_at = ? WHERE id = ?`, now, planID)

	// Get library path
	var libPath string
	s.db.QueryRow("SELECT path FROM libraries WHERE id = ?", libID).Scan(&libPath)

	// Execute pending actions
	results := []map[string]interface{}{}
	executed := 0
	maxBatch := 10 // Process up to 10 actions per call

	for i := range actions {
		if actions[i].Status != "pending" {
			continue
		}
		if executed >= maxBatch {
			break
		}

		result := map[string]interface{}{
			"action_id":   actions[i].ID,
			"type":        actions[i].Type,
			"description": actions[i].Description,
		}

		if dryRun {
			result["status"] = "would_execute"
			result["dry_run"] = true
			actions[i].Status = "completed" // Count as done for dry run
			completedActions++
			executed++
			results = append(results, result)
			continue
		}

		// Execute for real based on action type
		switch actions[i].Type {
		case "move_video":
			if actions[i].VideoID > 0 && actions[i].TargetPath != "" {
				// Check if target is a duplicate warning (target_path contains another file path)
				if strings.Contains(actions[i].TargetPath, ".") && filepath.Ext(actions[i].TargetPath) != "" {
					// This is a duplicate notification, not a move
					result["status"] = "skipped"
					result["reason"] = "duplicate detection — review manually"
					actions[i].Status = "skipped"
				} else {
					// Actually move the video
					err := s.executeMoveVideo(actions[i].VideoID, libPath, actions[i].TargetPath)
					if err != nil {
						result["status"] = "failed"
						result["error"] = err.Error()
						actions[i].Status = "failed"
						actions[i].Error = err.Error()
					} else {
						result["status"] = "completed"
						actions[i].Status = "completed"
						completedActions++
					}
				}
			} else {
				result["status"] = "skipped"
				result["reason"] = "missing video_id or target path"
				actions[i].Status = "skipped"
			}
			executed++

		case "tag_video":
			if actions[i].VideoID > 0 {
				// Auto-tag based on performer history
				tagged, err := s.autoTagVideoFromPerformers(actions[i].VideoID)
				if err != nil {
					result["status"] = "failed"
					result["error"] = err.Error()
					actions[i].Status = "failed"
				} else {
					result["status"] = "completed"
					result["tags_applied"] = tagged
					actions[i].Status = "completed"
					completedActions++
				}
			} else {
				result["status"] = "skipped"
				actions[i].Status = "skipped"
			}
			executed++

		case "create_folder":
			// Create folder on disk
			if actions[i].SourcePath != "" && !dryRun {
				folderPath := filepath.Join(libPath, filepath.FromSlash(actions[i].SourcePath))
				if err := os.MkdirAll(folderPath, 0755); err != nil {
					result["status"] = "failed"
					result["error"] = err.Error()
					actions[i].Status = "failed"
				} else {
					result["status"] = "completed"
					actions[i].Status = "completed"
					completedActions++
				}
			}
			executed++

		default:
			result["status"] = "skipped"
			result["reason"] = "unknown action type: " + actions[i].Type
			actions[i].Status = "skipped"
			executed++
		}

		results = append(results, result)
	}

	// Determine final status
	finalStatus := "executing"
	allDone := true
	anyFailed := false
	for _, a := range actions {
		if a.Status == "pending" {
			allDone = false
		}
		if a.Status == "failed" {
			anyFailed = true
		}
	}
	if allDone {
		finalStatus = "completed"
	} else if anyFailed {
		finalStatus = "executing" // Continue despite failures
	}

	// Save updated plan data
	updatedData, _ := json.Marshal(actions)
	s.db.Exec(`UPDATE companion_plans SET plan_data = ?, status = ?, completed_actions = ? WHERE id = ?`,
		string(updatedData), finalStatus, completedActions, planID)

	return map[string]interface{}{
		"plan_id":           planID,
		"plan_name":         planName,
		"status":            finalStatus,
		"dry_run":           dryRun,
		"actions_executed":  executed,
		"actions_remaining": totalActions - completedActions,
		"total_actions":     totalActions,
		"completed_actions": completedActions,
		"results":           results,
	}, nil
}

// executeMoveVideo moves a video to a subfolder within its library.
func (s *AICompanionService) executeMoveVideo(videoID int64, libPath, targetSubfolder string) error {
	var filePath string
	err := s.db.QueryRow("SELECT file_path FROM videos WHERE id = ?", videoID).Scan(&filePath)
	if err != nil {
		return fmt.Errorf("video %d not found", videoID)
	}

	srcFullPath := filePath
	if !filepath.IsAbs(filePath) {
		srcFullPath = filepath.Join(libPath, filePath)
	}

	if _, err := os.Stat(srcFullPath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", srcFullPath)
	}

	filename := filepath.Base(srcFullPath)
	tgtFullPath := filepath.Join(libPath, filepath.FromSlash(targetSubfolder), filename)

	if _, err := os.Stat(tgtFullPath); err == nil {
		return fmt.Errorf("destination already exists: %s", tgtFullPath)
	}

	if err := os.MkdirAll(filepath.Dir(tgtFullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.Rename(srcFullPath, tgtFullPath); err != nil {
		// Fallback: copy + delete
		data, readErr := os.ReadFile(srcFullPath)
		if readErr != nil {
			return fmt.Errorf("failed to read source: %w", readErr)
		}
		if writeErr := os.WriteFile(tgtFullPath, data, 0644); writeErr != nil {
			return fmt.Errorf("failed to write target: %w", writeErr)
		}
		os.Remove(srcFullPath)
	}

	// Update DB
	_, dbErr := s.db.Exec("UPDATE videos SET file_path = ?, updated_at = ? WHERE id = ?",
		tgtFullPath, time.Now(), videoID)
	if dbErr != nil {
		return fmt.Errorf("file moved but DB update failed: %w", dbErr)
	}

	// Learn placement for future suggestions
	perfRows, _ := s.db.Query("SELECT p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?", videoID)
	for perfRows.Next() {
		var perfName string
		if perfRows.Scan(&perfName) == nil {
			memKey := "placement_" + strings.ToLower(strings.ReplaceAll(perfName, " ", "_"))
			_ = s.SaveMemory(&models.Memory{
				Key:        memKey,
				Value:      fmt.Sprintf("Performer '%s' → %s", perfName, targetSubfolder),
				Category:   "insight",
				Importance: 8,
			})
		}
	}
	perfRows.Close()

	return nil
}

// autoTagVideoFromPerformers applies tags to a video based on what tags other videos by the same performers have.
func (s *AICompanionService) autoTagVideoFromPerformers(videoID int64) ([]string, error) {
	// Find performers for this video
	perfRows, err := s.db.Query(
		`SELECT p.id, p.name FROM performers p JOIN video_performers vp ON p.id = vp.performer_id WHERE vp.video_id = ?`, videoID)
	if err != nil {
		return nil, err
	}
	defer perfRows.Close()

	type performer struct {
		id   int64
		name string
	}
	var performers []performer
	for perfRows.Next() {
		var p performer
		if perfRows.Scan(&p.id, &p.name) == nil {
			performers = append(performers, p)
		}
	}

	if len(performers) == 0 {
		return nil, nil
	}

	// Find common tags across videos by these performers
	tagCounts := map[string]int64{}
	for _, p := range performers {
		tagRows, _ := s.db.Query(`
			SELECT t.id, t.name, COUNT(*) as cnt
			FROM tags t
			JOIN video_tags vt ON t.id = vt.tag_id
			JOIN video_performers vp ON vt.video_id = vp.video_id
			WHERE vp.performer_id = ? AND vt.video_id != ?
			GROUP BY t.id
			ORDER BY cnt DESC
			LIMIT 5`, p.id, videoID)
		if tagRows != nil {
			for tagRows.Next() {
				var tagID int64
				var tagName string
				var cnt int
				if tagRows.Scan(&tagID, &tagName, &cnt) == nil {
					if cnt >= 2 { // Only tags that appear 2+ times
						tagCounts[tagName] = tagID
					}
				}
			}
			tagRows.Close()
		}
	}

	var applied []string
	for tagName, tagID := range tagCounts {
		// Check if video already has this tag
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM video_tags WHERE video_id = ? AND tag_id = ?", videoID, tagID).Scan(&exists)
		if exists == 0 {
			s.db.Exec("INSERT OR IGNORE INTO video_tags (video_id, tag_id) VALUES (?, ?)", videoID, tagID)
			applied = append(applied, tagName)
		}
	}

	return applied, nil
}

// ==================== LIST / GET / DELETE PLANS ====================

func (s *AICompanionService) toolListPlans(args map[string]interface{}) (interface{}, error) {
	status, _ := args["status"].(string)
	if status == "" || status == "all" {
		status = ""
	}

	query := `SELECT cp.id, cp.plan_name, l.name, cp.status, cp.total_actions, cp.completed_actions, cp.created_at
	          FROM companion_plans cp
	          JOIN libraries l ON cp.library_id = l.id`
	args2 := []interface{}{}
	if status != "" {
		query += " WHERE cp.status = ?"
		args2 = append(args2, status)
	}
	query += " ORDER BY cp.created_at DESC LIMIT 20"

	rows, err := s.db.Query(query, args2...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plans := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var name, libName, planStatus string
		var total, completed int
		var createdAt time.Time
		if rows.Scan(&id, &name, &libName, &planStatus, &total, &completed, &createdAt) == nil {
			plans = append(plans, map[string]interface{}{
				"id":               id,
				"plan_name":        name,
				"library":          libName,
				"status":           planStatus,
				"total_actions":    total,
				"completed_actions": completed,
				"created_at":       createdAt.Format(time.RFC3339),
			})
		}
	}
	return map[string]interface{}{"plans": plans, "count": len(plans)}, nil
}

func (s *AICompanionService) toolGetPlan(args map[string]interface{}) (interface{}, error) {
	planID := int64(intArg(args, "plan_id", 0))

	var libID int64
	var planName, planDataStr, status string
	var totalActions, completedActions int
	var createdAt time.Time
	err := s.db.QueryRow(
		`SELECT library_id, plan_name, plan_data, status, total_actions, completed_actions, created_at
		 FROM companion_plans WHERE id = ?`, planID,
	).Scan(&libID, &planName, &planDataStr, &status, &totalActions, &completedActions, &createdAt)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("plan %d not found", planID)}, nil
	}

	var libName string
	s.db.QueryRow("SELECT name FROM libraries WHERE id = ?", libID).Scan(&libName)

	var actions []ImprovementAction
	json.Unmarshal([]byte(planDataStr), &actions)

	// Group by status
	statusCounts := map[string]int{}
	for _, a := range actions {
		statusCounts[a.Status]++
	}

	return map[string]interface{}{
		"plan_id":           planID,
		"plan_name":         planName,
		"library":           libName,
		"status":            status,
		"total_actions":     totalActions,
		"completed_actions": completedActions,
		"status_counts":     statusCounts,
		"created_at":        createdAt.Format(time.RFC3339),
		"actions":           actions,
	}, nil
}

func (s *AICompanionService) toolDeletePlan(args map[string]interface{}) (interface{}, error) {
	planID := int64(intArg(args, "plan_id", 0))

	var status string
	err := s.db.QueryRow("SELECT status FROM companion_plans WHERE id = ?", planID).Scan(&status)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("plan %d not found", planID)}, nil
	}
	if status == "executing" || status == "completed" {
		return map[string]interface{}{"error": fmt.Sprintf("cannot delete plan with status '%s'", status)}, nil
	}

	s.db.Exec("DELETE FROM companion_plans WHERE id = ?", planID)
	return map[string]interface{}{"deleted": true, "plan_id": planID}, nil
}

// ==================== HELPERS ====================

func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}

// sortActionsByPriority orders actions so structural changes come before moves,
// and moves before tagging.
func sortActionsByPriority(actions []ImprovementAction) {
	priority := map[string]int{
		"create_folder": 0,
		"move_video":    1,
		"tag_video":     2,
		"rename_folder": 3,
	}
	sort.Slice(actions, func(i, j int) bool {
		pi, oki := priority[actions[i].Type]
		if !oki {
			pi = 99
		}
		pj, okj := priority[actions[j].Type]
		if !okj {
			pj = 99
		}
		return pi < pj
	})
}
