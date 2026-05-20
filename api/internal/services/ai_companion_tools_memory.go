package services

import "github.com/brixen96/video-storage-ai/internal/models"

func (s *AICompanionService) registerMemoryTools() {
	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "save_memory",
			Description: "Store important information in long-term memory that persists forever.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"key":        map[string]interface{}{"type": "string", "description": "Unique identifier for this memory"},
				"value":      map[string]interface{}{"type": "string", "description": "The information to remember"},
				"category":   map[string]interface{}{"type": "string", "enum": []string{"preference", "fact", "insight", "task", "note"}, "description": "Type of memory"},
				"importance": map[string]interface{}{"type": "number", "description": "Importance level 1-10"},
			}, "required": []string{"key", "value", "category"}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		key, _ := args["key"].(string)
		value, _ := args["value"].(string)
		category, _ := args["category"].(string)
		importance := intArg(args, "importance", 5)
		if key == "" || value == "" {
			return map[string]interface{}{"error": "key and value are required"}, nil
		}
		mem := &models.Memory{Key: key, Value: value, Category: category, Importance: importance}
		if err := s.SaveMemory(mem); err != nil {
			return nil, err
		}
		return map[string]interface{}{"saved": true, "key": key}, nil
	})

	s.registerTool(lmToolDefinition{
		Type: "function", Function: lmToolFunction{
			Name:        "recall_memories",
			Description: "Search and retrieve stored memories.",
			Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string", "description": "Search query to find relevant memories"},
			}, "required": []string{}},
		},
	}, func(args map[string]interface{}) (interface{}, error) {
		query, _ := args["query"].(string)
		if query == "" {
			return s.GetMemories("", 20)
		}
		return s.SearchMemories(query)
	})
}
