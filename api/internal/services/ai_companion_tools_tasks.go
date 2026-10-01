package services

import (
	"fmt"

	"github.com/brixen96/video-storage-ai/internal/models"
)

func (s *AICompanionService) registerTaskTools() {
	taskService := NewCompanionTaskService(s.db)

	// create_task - Create a new task on the shared board
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "create_task",
			Description: "Create a new task on the shared task board. Use this to report bugs, suggest improvements, or track work items that need attention.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title":       map[string]interface{}{"type": "string", "description": "Brief task title"},
					"description": map[string]interface{}{"type": "string", "description": "Detailed description of the task"},
					"priority":    map[string]interface{}{"type": "number", "description": "Priority 1-10 (default: 5, higher = more important)"},
					"tags":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Tags for categorization"},
					"related_file": map[string]interface{}{"type": "string", "description": "Related source file path"},
					"related_line": map[string]interface{}{"type": "number", "description": "Related line number in source file"},
					"error_context": map[string]interface{}{"type": "string", "description": "Error message or stack trace if reporting a bug"},
				},
				"required": []string{"title"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		title, _ := args["title"].(string)
		description, _ := args["description"].(string)
		priority := intArg(args, "priority", 5)
		relatedFile, _ := args["related_file"].(string)
		relatedLine := intArg(args, "related_line", 0)
		errorContext, _ := args["error_context"].(string)

		var tags []string
		if tagList, ok := args["tags"].([]interface{}); ok {
			for _, t := range tagList {
				if tag, ok := t.(string); ok {
					tags = append(tags, tag)
				}
			}
		}

		task := &models.CompanionTask{
			Title:        title,
			Description:  description,
			Status:       "open",
			Priority:     priority,
			CreatedBy:    "companion",
			Tags:         tags,
			RelatedFile:  relatedFile,
			RelatedLine:  relatedLine,
			ErrorContext: errorContext,
		}

		if err := taskService.CreateTask(task); err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}

		return map[string]interface{}{
			"success": true,
			"task_id": task.ID,
			"message": fmt.Sprintf("Created task #%d: %s", task.ID, task.Title),
		}, nil
	})

	// list_tasks - List tasks on the board
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "list_tasks",
			Description: "List tasks from the shared task board. Optionally filter by status or creator.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status":     map[string]interface{}{"type": "string", "enum": []string{"open", "in_progress", "completed", "archived", "all"}, "description": "Filter by status (default: all)"},
					"created_by": map[string]interface{}{"type": "string", "enum": []string{"companion", "developer", "system"}, "description": "Filter by who created the task"},
					"limit":      map[string]interface{}{"type": "number", "description": "Max tasks to return (default: 20)"},
				},
				"required": []string{},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		status, _ := args["status"].(string)
		createdBy, _ := args["created_by"].(string)
		limit := intArg(args, "limit", 20)

		tasks, err := taskService.ListTasks(status, createdBy, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}

		return map[string]interface{}{
			"tasks": tasks,
			"count": len(tasks),
		}, nil
	})

	// update_task - Update a task's status or details
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "update_task",
			Description: "Update a task's status, priority, or other fields.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id":     map[string]interface{}{"type": "number", "description": "Task ID to update"},
					"status":      map[string]interface{}{"type": "string", "enum": []string{"open", "in_progress", "completed", "archived"}, "description": "New status"},
					"priority":    map[string]interface{}{"type": "number", "description": "New priority 1-10"},
					"resolution":  map[string]interface{}{"type": "string", "description": "Resolution notes when completing"},
					"assigned_to": map[string]interface{}{"type": "string", "description": "Who this task is assigned to"},
				},
				"required": []string{"task_id"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		taskID := int64(args["task_id"].(float64))
		status, _ := args["status"].(string)
		resolution, _ := args["resolution"].(string)

		updates := map[string]interface{}{}
		if status != "" {
			updates["status"] = status
		}
		if v, ok := args["priority"].(float64); ok {
			updates["priority"] = v
		}
		if resolution != "" {
			updates["resolution"] = resolution
		}
		if v, ok := args["assigned_to"].(string); ok {
			updates["assigned_to"] = v
		}

		if status != "" {
			if err := taskService.UpdateTaskStatus(taskID, status, resolution); err != nil {
				return nil, fmt.Errorf("failed to update task: %w", err)
			}
		}

		if len(updates) > 1 { // more than just status
			if err := taskService.UpdateTask(taskID, updates); err != nil {
				return nil, fmt.Errorf("failed to update task: %w", err)
			}
		}

		task, err := taskService.GetTask(taskID)
		if err != nil {
			return map[string]interface{}{"success": true, "message": "Task updated"}, nil
		}

		return map[string]interface{}{
			"success": true,
			"task":    task,
			"message": fmt.Sprintf("Updated task #%d: %s", task.ID, task.Title),
		}, nil
	})

	// get_task_stats - Get task board statistics
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "get_task_stats",
			Description: "Get statistics about the shared task board - counts by status, recent activity, etc.",
			Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		stats, err := taskService.GetTaskStats()
		if err != nil {
			return nil, fmt.Errorf("failed to get stats: %w", err)
		}
		return stats, nil
	})

	// delete_task - Remove a task from the board
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "delete_task",
			Description: "Delete a task from the board permanently.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "number", "description": "Task ID to delete"},
				},
				"required": []string{"task_id"},
			},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		taskID := int64(args["task_id"].(float64))
		if err := taskService.DeleteTask(taskID); err != nil {
			return nil, fmt.Errorf("failed to delete task: %w", err)
		}
		return map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Deleted task #%d", taskID),
		}, nil
	})
}