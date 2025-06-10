package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/lsp"
	"github.com/guild-ventures/guild-core/pkg/observability"
	"github.com/guild-ventures/guild-core/tools"
	lsptools "github.com/guild-ventures/guild-core/pkg/lsp/tools"
)

// LSPAwareExecutor is a task executor that leverages LSP for intelligent code operations
type LSPAwareExecutor struct {
	*TaskExecutor
	lspManager *lsp.Manager
	lspTools   map[string]tools.Tool
}

// NewLSPAwareExecutor creates a new LSP-aware task executor
func NewLSPAwareExecutor(baseExecutor *TaskExecutor, lspManager *lsp.Manager) *LSPAwareExecutor {
	executor := &LSPAwareExecutor{
		TaskExecutor: baseExecutor,
		lspManager:   lspManager,
		lspTools:     make(map[string]tools.Tool),
	}
	
	// Register LSP tools
	executor.registerLSPTools()
	
	return executor
}

// registerLSPTools registers all LSP-based tools
func (e *LSPAwareExecutor) registerLSPTools() {
	// Create LSP tools
	completionTool := lsptools.NewCompletionTool(e.lspManager)
	definitionTool := lsptools.NewDefinitionTool(e.lspManager)
	referencesTool := lsptools.NewReferencesTool(e.lspManager)
	hoverTool := lsptools.NewHoverTool(e.lspManager)
	
	// Adapt and store in our map
	e.lspTools[completionTool.Name()] = lsptools.ToRegistryTool(completionTool)
	e.lspTools[definitionTool.Name()] = lsptools.ToRegistryTool(definitionTool)
	e.lspTools[referencesTool.Name()] = lsptools.ToRegistryTool(referencesTool)
	e.lspTools[hoverTool.Name()] = lsptools.ToRegistryTool(hoverTool)
	
	// Register with the base tool registry
	for _, tool := range e.lspTools {
		e.toolRegistry.RegisterTool(tool.Name(), tool)
	}
}

// ExecuteWithContext executes a task with LSP context enhancement
func (e *LSPAwareExecutor) ExecuteWithContext(ctx context.Context, task Task) (TaskResult, error) {
	logger := observability.GetLogger(ctx)
	
	// Check if this is a code-related task
	if e.isCodeRelatedTask(task) {
		// Enhance task with LSP context
		enhancedTask, err := e.enhanceTaskWithLSP(ctx, task)
		if err != nil {
			logger.WarnContext(ctx, "Failed to enhance task with LSP context",
				"error", err,
				"task", task.ID)
			// Continue with original task
		} else {
			task = enhancedTask
		}
	}
	
	// Execute with the base executor
	return e.TaskExecutor.Execute(ctx, task)
}

// isCodeRelatedTask determines if a task is code-related
func (e *LSPAwareExecutor) isCodeRelatedTask(task Task) bool {
	// Check tool name
	codeTools := []string{
		"read_file", "write_file", "edit_file",
		"search_code", "analyze_code", "refactor_code",
		"lsp_completion", "lsp_definition", "lsp_references", "lsp_hover",
	}
	
	for _, tool := range codeTools {
		if strings.Contains(task.Tool, tool) {
			return true
		}
	}
	
	// Check task description for code-related keywords
	keywords := []string{
		"code", "function", "method", "class", "variable",
		"implement", "refactor", "fix", "bug", "error",
		"compile", "build", "test", "debug",
	}
	
	lowerDesc := strings.ToLower(task.Description)
	for _, keyword := range keywords {
		if strings.Contains(lowerDesc, keyword) {
			return true
		}
	}
	
	return false
}

// enhanceTaskWithLSP enhances a task with LSP-derived context
func (e *LSPAwareExecutor) enhanceTaskWithLSP(ctx context.Context, task Task) (Task, error) {
	logger := observability.GetLogger(ctx)
	
	// Extract file information from task
	fileInfo := e.extractFileInfo(task)
	if fileInfo.FilePath == "" {
		return task, nil // No file context to enhance
	}
	
	// Get language server for the file
	server, err := e.lspManager.GetServerForFile(ctx, fileInfo.FilePath)
	if err != nil {
		return task, gerror.Wrap(err, gerror.ErrCodeExternal, "failed to get language server").
			WithComponent("lsp_aware_executor").
			WithOperation("enhance_task")
	}
	
	// Create enhanced context
	enhancedContext := make(map[string]interface{})
	
	// Add file metadata
	enhancedContext["file"] = fileInfo.FilePath
	enhancedContext["language"] = server.Language
	
	// If we have position information, get additional context
	if fileInfo.Line > 0 && fileInfo.Column > 0 {
		// Get hover information for context
		hover, err := e.lspManager.GetHover(ctx, fileInfo.FilePath, fileInfo.Line-1, fileInfo.Column-1)
		if err == nil && hover != nil {
			enhancedContext["symbol_info"] = e.extractHoverInfo(hover)
		}
		
		// Get definition for navigation context
		definitions, err := e.lspManager.GetDefinition(ctx, fileInfo.FilePath, fileInfo.Line-1, fileInfo.Column-1)
		if err == nil && len(definitions) > 0 {
			var defLocations []string
			for _, def := range definitions {
				defLocations = append(defLocations, fmt.Sprintf("%s:%d:%d", 
					strings.TrimPrefix(def.URI, "file://"),
					def.Range.Start.Line+1,
					def.Range.Start.Character+1))
			}
			enhancedContext["definitions"] = defLocations
		}
	}
	
	// Add context to task
	if task.Context == nil {
		task.Context = make(map[string]interface{})
	}
	task.Context["lsp_context"] = enhancedContext
	
	logger.DebugContext(ctx, "Enhanced task with LSP context",
		"task_id", task.ID,
		"file", fileInfo.FilePath,
		"language", server.Language)
	
	return task, nil
}

// fileInfo contains extracted file information
type fileInfo struct {
	FilePath string
	Line     int
	Column   int
}

// extractFileInfo extracts file information from a task
func (e *LSPAwareExecutor) extractFileInfo(task Task) fileInfo {
	info := fileInfo{}
	
	// Check input parameters
	if task.Input != nil {
		if filePath, ok := task.Input["file"].(string); ok {
			info.FilePath = filePath
		} else if filePath, ok := task.Input["path"].(string); ok {
			info.FilePath = filePath
		}
		
		if line, ok := task.Input["line"].(int); ok {
			info.Line = line
		} else if line, ok := task.Input["line"].(float64); ok {
			info.Line = int(line)
		}
		
		if column, ok := task.Input["column"].(int); ok {
			info.Column = column
		} else if column, ok := task.Input["column"].(float64); ok {
			info.Column = int(column)
		}
	}
	
	// Check context for file information
	if info.FilePath == "" && task.Context != nil {
		if filePath, ok := task.Context["current_file"].(string); ok {
			info.FilePath = filePath
		}
	}
	
	return info
}

// extractHoverInfo extracts useful information from hover results
func (e *LSPAwareExecutor) extractHoverInfo(hover *lsp.Hover) map[string]interface{} {
	info := make(map[string]interface{})
	
	// Extract content based on type
	switch content := hover.Contents.(type) {
	case string:
		info["content"] = content
		info["type"] = "plain"
	case map[string]interface{}:
		if kind, ok := content["kind"].(string); ok {
			info["type"] = kind
		}
		if value, ok := content["value"].(string); ok {
			info["content"] = value
		}
	}
	
	return info
}

// SelectTool selects the most appropriate tool for a task, preferring LSP tools when available
func (e *LSPAwareExecutor) SelectTool(ctx context.Context, task Task) (tools.Tool, error) {
	// If task explicitly requests an LSP tool, use it
	if strings.HasPrefix(task.Tool, "lsp_") {
		if tool, exists := e.lspTools[task.Tool]; exists {
			return tool, nil
		}
	}
	
	// For code intelligence tasks, prefer LSP tools
	if e.shouldUseLSPTool(ctx, task) {
		lspTool := e.mapToLSPTool(task.Tool)
		if lspTool != "" {
			if tool, exists := e.lspTools[lspTool]; exists {
				observability.GetLogger(ctx).DebugContext(ctx, "Using LSP tool instead of regular tool",
					"original_tool", task.Tool,
					"lsp_tool", lspTool)
				return tool, nil
			}
		}
	}
	
	// Fall back to regular tool selection
	return e.TaskExecutor.selectTool(task)
}

// shouldUseLSPTool determines if an LSP tool should be used instead of a regular tool
func (e *LSPAwareExecutor) shouldUseLSPTool(ctx context.Context, task Task) bool {
	// Check if we have file context
	fileInfo := e.extractFileInfo(task)
	if fileInfo.FilePath == "" {
		return false
	}
	
	// Check if we have an LSP server for this file type
	language := lsp.DetectLanguage(fileInfo.FilePath)
	if language == "" {
		return false
	}
	
	// Check if the language server is available
	_, err := e.lspManager.GetServerForFile(ctx, fileInfo.FilePath)
	return err == nil
}

// mapToLSPTool maps regular tool names to LSP tool equivalents
func (e *LSPAwareExecutor) mapToLSPTool(toolName string) string {
	mappings := map[string]string{
		"get_completions":     "lsp_completion",
		"find_definition":     "lsp_definition",
		"find_references":     "lsp_references",
		"get_documentation":   "lsp_hover",
		"get_type_info":       "lsp_hover",
	}
	
	if lspTool, exists := mappings[toolName]; exists {
		return lspTool
	}
	
	return ""
}

// GetAvailableLSPTools returns the list of available LSP tools
func (e *LSPAwareExecutor) GetAvailableLSPTools() []string {
	var tools []string
	for name := range e.lspTools {
		tools = append(tools, name)
	}
	return tools
}

// GetLSPServerStatus returns the status of active LSP servers
func (e *LSPAwareExecutor) GetLSPServerStatus() []lsp.ActiveServerInfo {
	return e.lspManager.GetActiveServers()
}