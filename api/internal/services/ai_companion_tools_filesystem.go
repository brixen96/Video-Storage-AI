package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func (s *AICompanionService) registerFilesystemTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "list_folder_contents",
			Description: "List files and subfolders inside a library folder. Provide a library name and an optional relative subfolder path. Returns folders, video files, and non-video files.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name":    map[string]interface{}{"type": "string", "description": "Library name or partial name"},
				"subfolder":       map[string]interface{}{"type": "string", "description": "Relative path inside the library (optional — omit for library root)"},
				"include_files":   map[string]interface{}{"type": "boolean", "description": "Include individual files in the listing (default: true)"},
			}, "required": []string{"library_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		subfolder, _ := args["subfolder"].(string)
		includeFiles := true
		if v, ok := args["include_files"].(bool); ok {
			includeFiles = v
		}

		var libPath string
		if err := s.db.QueryRow("SELECT path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libPath); err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}

		targetPath := libPath
		if subfolder != "" {
			targetPath = filepath.Join(libPath, filepath.FromSlash(subfolder))
		}
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return map[string]interface{}{"error": "folder not found: " + targetPath}, nil
		}

		entries, err := os.ReadDir(targetPath)
		if err != nil {
			return map[string]interface{}{"error": "cannot read folder: " + err.Error()}, nil
		}

		var folders, videoFiles, otherFiles []map[string]interface{}
		videoExts := map[string]bool{".mp4": true, ".mkv": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true, ".m4v": true, ".webm": true}

		for _, e := range entries {
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			if e.IsDir() {
				folders = append(folders, map[string]interface{}{"name": e.Name(), "type": "folder"})
			} else if includeFiles {
				ext := strings.ToLower(filepath.Ext(e.Name()))
				entry := map[string]interface{}{"name": e.Name(), "size_bytes": size}
				if videoExts[ext] {
					videoFiles = append(videoFiles, entry)
				} else {
					otherFiles = append(otherFiles, entry)
				}
			}
		}

		relDisplay := subfolder
		if relDisplay == "" {
			relDisplay = "."
		}
		return map[string]interface{}{
			"library":      libraryName,
			"path":         relDisplay,
			"folders":      folders,
			"video_files":  videoFiles,
			"other_files":  otherFiles,
			"folder_count": len(folders),
			"video_count":  len(videoFiles),
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "create_folder",
			Description: "Create a new folder inside a library. Provide the library name and the relative path for the new folder.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name"},
				"folder_path":  map[string]interface{}{"type": "string", "description": "Relative path for the new folder inside the library (e.g. 'Performers/Jane Doe')"},
			}, "required": []string{"library_name", "folder_path"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		folderPath, _ := args["folder_path"].(string)
		if folderPath == "" {
			return map[string]interface{}{"error": "folder_path required"}, nil
		}

		var libPath string
		if err := s.db.QueryRow("SELECT path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libPath); err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}

		fullPath := filepath.Join(libPath, filepath.FromSlash(folderPath))
		if _, err := os.Stat(fullPath); err == nil {
			return map[string]interface{}{"error": "folder already exists: " + fullPath}, nil
		}
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return map[string]interface{}{"error": "failed to create folder: " + err.Error()}, nil
		}
		log.Printf("[AI] create_folder: created %s", fullPath)
		return map[string]interface{}{"created": true, "path": folderPath, "full_path": fullPath}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "rename_folder_fs",
			Description: "Rename a folder on disk and update all video file_path records in the database that fall under that folder. ALWAYS confirm with the user before calling — this affects all videos inside the folder.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name"},
				"old_path":     map[string]interface{}{"type": "string", "description": "Current relative folder path inside the library (e.g. 'Performers/Jane')"},
				"new_name":     map[string]interface{}{"type": "string", "description": "New name for the folder (just the final segment, not the full path)"},
			}, "required": []string{"library_name", "old_path", "new_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		oldRelPath, _ := args["old_path"].(string)
		newName, _ := args["new_name"].(string)
		if oldRelPath == "" || newName == "" {
			return map[string]interface{}{"error": "old_path and new_name are required"}, nil
		}

		var libPath string
		if err := s.db.QueryRow("SELECT path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libPath); err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}

		oldFullPath := filepath.Join(libPath, filepath.FromSlash(oldRelPath))
		newFullPath := filepath.Join(filepath.Dir(oldFullPath), newName)

		if _, err := os.Stat(oldFullPath); os.IsNotExist(err) {
			return map[string]interface{}{"error": "source folder not found: " + oldFullPath}, nil
		}
		if _, err := os.Stat(newFullPath); err == nil {
			return map[string]interface{}{"error": "destination already exists: " + newFullPath}, nil
		}

		if err := os.Rename(oldFullPath, newFullPath); err != nil {
			return map[string]interface{}{"error": "rename failed: " + err.Error()}, nil
		}

		// Update all videos under the old path in the DB
		rows, err := s.db.Query(`SELECT id, file_path FROM videos WHERE file_path LIKE ?`, oldFullPath+"%")
		if err != nil {
			return map[string]interface{}{"warning": "folder renamed on disk but DB update query failed: " + err.Error()}, nil
		}
		type vidUpdate struct {
			id      int64
			oldPath string
		}
		var updates []vidUpdate
		for rows.Next() {
			var id int64
			var fp string
			if rows.Scan(&id, &fp) == nil {
				updates = append(updates, vidUpdate{id, fp})
			}
		}
		rows.Close()

		updatedCount := 0
		for _, u := range updates {
			newFilePath := newFullPath + u.oldPath[len(oldFullPath):]
			if _, err := s.db.Exec("UPDATE videos SET file_path = ? WHERE id = ?", newFilePath, u.id); err == nil {
				updatedCount++
			}
		}

		log.Printf("[AI] rename_folder: %s → %s (%d video records updated)", oldFullPath, newFullPath, updatedCount)
		return map[string]interface{}{
			"renamed":        true,
			"old_path":       oldRelPath,
			"new_full_path":  newFullPath,
			"videos_updated": updatedCount,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "delete_folder",
			Description: "Delete an empty folder from disk. Refuses if the folder contains any files. ALWAYS confirm with the user before calling.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"library_name": map[string]interface{}{"type": "string", "description": "Library name or partial name"},
				"folder_path":  map[string]interface{}{"type": "string", "description": "Relative folder path inside the library"},
			}, "required": []string{"library_name", "folder_path"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		libraryName, _ := args["library_name"].(string)
		folderPath, _ := args["folder_path"].(string)
		if folderPath == "" {
			return map[string]interface{}{"error": "folder_path required"}, nil
		}

		var libPath string
		if err := s.db.QueryRow("SELECT path FROM libraries WHERE name LIKE ?", "%"+libraryName+"%").Scan(&libPath); err != nil {
			return map[string]interface{}{"error": "library not found: " + libraryName}, nil
		}

		fullPath := filepath.Join(libPath, filepath.FromSlash(folderPath))
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return map[string]interface{}{"error": "folder not found: " + fullPath}, nil
		}

		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return map[string]interface{}{"error": "cannot read folder: " + err.Error()}, nil
		}
		if len(entries) > 0 {
			return map[string]interface{}{
				"error": fmt.Sprintf("folder is not empty (%d items inside) — move or delete contents first", len(entries)),
			}, nil
		}

		if err := os.Remove(fullPath); err != nil {
			return map[string]interface{}{"error": "delete failed: " + err.Error()}, nil
		}
		log.Printf("[AI] delete_folder: deleted %s", fullPath)
		return map[string]interface{}{"deleted": true, "path": folderPath}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "rename_video_file",
			Description: "Rename a video file on disk and update its file_path in the database. The file stays in the same folder — only the filename changes. ALWAYS confirm with the user before calling.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id":  map[string]interface{}{"type": "number", "description": "ID of the video to rename"},
				"new_name":  map[string]interface{}{"type": "string", "description": "New filename including extension (e.g. 'MyVideo.mp4')"},
			}, "required": []string{"video_id", "new_name"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		newName, _ := args["new_name"].(string)
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		if newName == "" {
			return map[string]interface{}{"error": "new_name required"}, nil
		}

		var title, filePath string
		if err := s.db.QueryRow("SELECT title, file_path FROM videos WHERE id = ?", videoID).Scan(&title, &filePath); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return map[string]interface{}{"error": "file not found on disk: " + filePath}, nil
		}

		newPath := filepath.Join(filepath.Dir(filePath), newName)
		if _, err := os.Stat(newPath); err == nil {
			return map[string]interface{}{"error": "a file with that name already exists: " + newPath}, nil
		}

		if err := os.Rename(filePath, newPath); err != nil {
			return map[string]interface{}{"error": "rename failed: " + err.Error()}, nil
		}

		if _, err := s.db.Exec("UPDATE videos SET file_path = ? WHERE id = ?", newPath, videoID); err != nil {
			return map[string]interface{}{"warning": "file renamed on disk but DB update failed: " + err.Error(), "new_path": newPath}, nil
		}

		log.Printf("[AI] rename_video_file: video %d (%s) %s → %s", videoID, title, filepath.Base(filePath), newName)
		return map[string]interface{}{
			"video_id": videoID, "title": title,
			"old_name": filepath.Base(filePath), "new_name": newName,
			"new_path": newPath,
		}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "delete_video_file",
			Description: "Permanently delete a video file from disk AND remove its database record. This is irreversible. ALWAYS confirm explicitly with the user before calling this tool.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"video_id": map[string]interface{}{"type": "number", "description": "ID of the video to permanently delete"},
				"confirmed": map[string]interface{}{"type": "boolean", "description": "Must be true — explicit confirmation that the file should be permanently deleted"},
			}, "required": []string{"video_id", "confirmed"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		videoID := int64(floatArg(args, "video_id", 0))
		confirmed, _ := args["confirmed"].(bool)
		if videoID == 0 {
			return map[string]interface{}{"error": "video_id required"}, nil
		}
		if !confirmed {
			return map[string]interface{}{"error": "confirmed must be true — get explicit user confirmation before deleting a file permanently"}, nil
		}

		var title, filePath string
		if err := s.db.QueryRow("SELECT title, file_path FROM videos WHERE id = ?", videoID).Scan(&title, &filePath); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("video %d not found", videoID)}, nil
		}

		// Delete from disk
		fileDeleted := false
		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err != nil {
				return map[string]interface{}{"error": "failed to delete file: " + err.Error()}, nil
			}
			fileDeleted = true
		}

		// Remove DB record and junction table rows
		for _, q := range []string{
			"DELETE FROM video_performers WHERE video_id = ?",
			"DELETE FROM video_tags WHERE video_id = ?",
			"DELETE FROM video_studios WHERE video_id = ?",
		} {
			s.db.Exec(q, videoID)
		}
		s.db.Exec("DELETE FROM videos WHERE id = ?", videoID)

		log.Printf("[AI] delete_video_file: permanently deleted video %d (%s), file_deleted=%v", videoID, title, fileDeleted)
		return map[string]interface{}{
			"deleted":      true,
			"video_id":     videoID,
			"title":        title,
			"file_deleted": fileDeleted,
			"db_removed":   true,
		}, nil
	})
}
