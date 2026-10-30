package services

import (
	"path/filepath"

	appconfig "github.com/brixen96/video-storage-ai/internal/config"
)

// registerBackupTools exposes database backup operations to the AI Companion.
// Pairs with bulk operations: "back up the DB before this bulk move" is now a
// single tool call instead of a manual trip to the Backups page.

func (s *AICompanionService) registerBackupTools() {
	// ── create_backup ─────────────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "create_backup",
			Description: "Create a database backup NOW. Use before risky bulk operations (mass moves, deletes, migrations) when Brix asks to back up, or when he says 'back it up first'. Returns the backup file info.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Short reason recorded with the backup (e.g. 'before New folder bulk move')",
				},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		reason, _ := args["reason"].(string)
		backupType := "manual"
		if reason != "" {
			backupType = "ai_" + reason
		}
		svc := NewBackupService(filepath.Join(appconfig.DataDir(), "backups"))
		info, err := svc.CreateBackup(backupType)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"created":   true,
			"filename":  info.Filename,
			"size_bytes": info.Size,
			"created_at": info.CreatedAt,
			"reason":    reason,
		}, nil
	})

	// ── list_backups ──────────────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "list_backups",
			Description: "List available database backups (newest first) with size and age. Use when Brix asks what backups exist or when the last backup was made.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "number", "description": "Max backups to return (default: 20)"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		limit := intArg(args, "limit", 20)
		svc := NewBackupService(filepath.Join(appconfig.DataDir(), "backups"))
		backups, err := svc.ListBackups()
		if err != nil {
			return nil, err
		}
		if len(backups) > limit {
			backups = backups[:limit]
		}
		return map[string]interface{}{
			"count":    len(backups),
			"backups":  backups,
		}, nil
	})

	// ── get_backup_stats ──────────────────────────────────────────────────────
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_backup_stats",
			Description: "Get backup statistics: total count, total size, oldest/newest backup. Use for status questions about backups.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		svc := NewBackupService(filepath.Join(appconfig.DataDir(), "backups"))
		stats, err := svc.GetBackupStats()
		if err != nil {
			return nil, err
		}
		return stats, nil
	})
}