package registry

import (
	"github.com/guild-ventures/guild-core/tools"
	"github.com/guild-ventures/guild-core/tools/code"
	"github.com/guild-ventures/guild-core/tools/edit"
)

// RegisterCodeTools registers all code analysis and editing tools with the given tool registry
func RegisterCodeTools(registry interface{}) error {
	// Type assert to get the tool registry
	toolRegistry, ok := registry.(*tools.ToolRegistry)
	if !ok {
		// Try pkg/tools registry
		if pkgRegistry, ok := registry.(interface{ RegisterTool(tools.Tool) error }); ok {
			return registerWithPkgRegistry(pkgRegistry)
		}
		return nil
	}

	// Code analysis tools
	astTool := code.NewASTTool()
	if err := toolRegistry.RegisterTool(astTool); err != nil {
		return err
	}

	dependenciesTool := code.NewDependenciesTool()
	if err := toolRegistry.RegisterTool(dependenciesTool); err != nil {
		return err
	}

	metricsTool := code.NewMetricsTool()
	if err := toolRegistry.RegisterTool(metricsTool); err != nil {
		return err
	}

	searchReplaceTool := code.NewSearchReplaceTool()
	if err := toolRegistry.RegisterTool(searchReplaceTool); err != nil {
		return err
	}

	// Edit tools
	applyDiffTool := edit.NewApplyDiffTool()
	if err := toolRegistry.RegisterTool(applyDiffTool); err != nil {
		return err
	}

	cursorPositionTool := edit.NewCursorPositionTool()
	if err := toolRegistry.RegisterTool(cursorPositionTool); err != nil {
		return err
	}

	multiRefactorTool := edit.NewMultiFileRefactorTool()
	if err := toolRegistry.RegisterTool(multiRefactorTool); err != nil {
		return err
	}

	return nil
}

func registerWithPkgRegistry(registry interface{ RegisterTool(tools.Tool) error }) error {
	// Code analysis tools
	astTool := code.NewASTTool()
	if err := registry.RegisterTool(astTool); err != nil {
		return err
	}

	dependenciesTool := code.NewDependenciesTool()
	if err := registry.RegisterTool(dependenciesTool); err != nil {
		return err
	}

	metricsTool := code.NewMetricsTool()
	if err := registry.RegisterTool(metricsTool); err != nil {
		return err
	}

	searchReplaceTool := code.NewSearchReplaceTool()
	if err := registry.RegisterTool(searchReplaceTool); err != nil {
		return err
	}

	// Edit tools
	applyDiffTool := edit.NewApplyDiffTool()
	if err := registry.RegisterTool(applyDiffTool); err != nil {
		return err
	}

	cursorPositionTool := edit.NewCursorPositionTool()
	if err := registry.RegisterTool(cursorPositionTool); err != nil {
		return err
	}

	multiRefactorTool := edit.NewMultiFileRefactorTool()
	if err := registry.RegisterTool(multiRefactorTool); err != nil {
		return err
	}

	return nil
}

// GetCodeToolNames returns the names of all registered code tools
func GetCodeToolNames() []string {
	return []string{
		"ast",
		"dependencies",
		"metrics",
		"search_replace",
		"apply_diff",
		"cursor_position",
		"multi_refactor",
	}
}

// GetCodeToolsByCategory returns tools grouped by category
func GetCodeToolsByCategory() map[string][]string {
	return map[string][]string{
		"code_analysis": {
			"ast",
			"dependencies",
			"metrics",
		},
		"code_search": {
			"search_replace",
		},
		"code_editing": {
			"apply_diff",
			"cursor_position",
			"multi_refactor",
		},
	}
}
