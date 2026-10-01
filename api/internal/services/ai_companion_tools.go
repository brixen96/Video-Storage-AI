package services

import (
	"fmt"
	"sort"
)

// toolHandler is a function that executes a tool call from the LLM.
// It receives the parsed arguments map and returns a result or error.
type toolHandler func(args map[string]interface{}) (interface{}, error)

// registeredTool pairs an LLM tool definition with its Go implementation.
type registeredTool struct {
	def     lmToolDefinition
	handler toolHandler
}

// registerTool adds a tool to the service's registry.
// Call this inside a registerXxxTools() method during buildToolRegistry().
func (s *AICompanionService) registerTool(def lmToolDefinition, handler toolHandler) {
	s.tools[def.Function.Name] = &registeredTool{def: def, handler: handler}
}

// buildToolRegistry registers all tools by calling each domain's register function.
// Add a new registerXxxTools() call here when adding a new tool group.
func (s *AICompanionService) buildToolRegistry() {
	s.registerCoreTools()
	s.registerVideoTools()
	s.registerPerformerTools()
	s.registerLibraryTools()
	s.registerScraperTools()
	s.registerMemoryTools()
	s.registerLinksTools()
	s.registerFilesystemTools()
	s.registerPlanningTools()
	s.registerBulkMoveTool()
	s.registerRenameFolderTool()
	s.registerMoveFolderContentsTool()
	s.registerBatchMoveByPatternTool()
	s.registerUniversalScraperTools()
	s.registerTaskTools()
}

// getLMToolDefinitions returns all registered tool definitions, sorted by name
// for deterministic ordering in LLM requests.
func (s *AICompanionService) getLMToolDefinitions() []lmToolDefinition {
	defs := make([]lmToolDefinition, 0, len(s.tools))
	for _, t := range s.tools {
		defs = append(defs, t.def)
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Function.Name < defs[j].Function.Name
	})
	return defs
}

// ExecuteTool dispatches a tool call from the LLM to the registered handler.
func (s *AICompanionService) ExecuteTool(name string, args map[string]interface{}) (interface{}, error) {
	tool, ok := s.tools[name]
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf("Unknown tool: %s", name)}, nil
	}
	return tool.handler(args)
}
