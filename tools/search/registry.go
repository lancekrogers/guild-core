// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package search

import (
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// RegisterSearchTools registers all search tools with the provided registry
func RegisterSearchTools(registry *tools.ToolRegistry, workspacePath string) error {
	// Create search tools
	agTool := NewAgTool(workspacePath)

	// Register ag tool
	if err := registry.RegisterTool(agTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register ag tool").
			WithComponent("tools.search").
			WithOperation("register_tools").
			WithDetails("tool", "ag")
	}

	return nil
}

// GetSearchTools returns all search tools for registration with cost-aware registries
func GetSearchTools(workspacePath string) []SearchToolInfo {
	return []SearchToolInfo{
		{
			Tool:          NewAgTool(workspacePath),
			CostMagnitude: 0, // Local operation, zero cost
			Capabilities:  []string{"search", "text_search", "code_search", "pattern_matching", "file_filtering"},
		},
	}
}

// SearchToolInfo contains tool information for cost-aware registration
type SearchToolInfo struct {
	Tool          tools.Tool
	CostMagnitude int
	Capabilities  []string
}

// RegisterSearchToolsWithCost registers search tools with cost information
func RegisterSearchToolsWithCost(registry interface{}, workspacePath string) error {
	// Type assert to cost-aware registry interface
	type CostAwareRegistry interface {
		RegisterToolWithCost(name string, tool tools.Tool, costMagnitude int, capabilities []string) error
	}

	costRegistry, ok := registry.(CostAwareRegistry)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidInput, "registry does not support cost-aware registration", nil).
			WithComponent("tools.search").
			WithOperation("register_tools_with_cost")
	}

	// Get all search tools with cost information
	searchTools := GetSearchTools(workspacePath)

	// Register each tool with cost information
	for _, toolInfo := range searchTools {
		err := costRegistry.RegisterToolWithCost(
			toolInfo.Tool.Name(),
			toolInfo.Tool,
			toolInfo.CostMagnitude,
			toolInfo.Capabilities,
		)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register search tool with cost").
				WithComponent("tools.search").
				WithOperation("register_tools_with_cost").
				WithDetails("tool", toolInfo.Tool.Name())
		}
	}

	return nil
}

// GetAgTool creates and returns a configured ag tool
func GetAgTool(workspacePath string) *AgTool {
	return NewAgTool(workspacePath)
}

// ValidateAgInstallation checks if ag (The Silver Searcher) is properly installed
func ValidateAgInstallation() error {
	agTool := NewAgTool("")
	if !agTool.isAgInstalled() {
		return gerror.New(gerror.ErrCodeNotFound, "The Silver Searcher (ag) is not installed. Please install it using: brew install the_silver_searcher (macOS) or apt-get install silversearcher-ag (Ubuntu)", nil).
			WithComponent("tools.search").
			WithOperation("validate_ag_installation")
	}
	return nil
}

// GetSupportedFileTypes returns commonly supported file types for ag searches
func GetSupportedFileTypes() []string {
	return []string{
		"go", "js", "ts", "py", "java", "c", "cpp", "h", "hpp",
		"rb", "php", "html", "htm", "css", "scss", "sass", "less",
		"xml", "json", "yaml", "yml", "toml", "ini", "cfg", "conf",
		"sh", "bash", "zsh", "fish", "ps1", "bat", "cmd",
		"sql", "md", "txt", "rst", "tex", "r", "R", "m", "mm",
		"swift", "kt", "scala", "clj", "cljs", "edn", "hs", "lhs",
		"elm", "ex", "exs", "erl", "hrl", "fs", "fsx", "ml", "mli",
		"rs", "dart", "lua", "pl", "pm", "t", "vim", "vimrc",
	}
}

// GetCommonIgnorePatterns returns commonly used ignore patterns
func GetCommonIgnorePatterns() []string {
	return []string{
		"*.min.js", "*.min.css", "*.map",
		"node_modules", "bower_components", "vendor",
		"*.log", "*.tmp", "*.temp", "*.cache",
		"build", "dist", "target", "bin", "obj",
		".git", ".svn", ".hg", ".bzr",
		"*.pyc", "*.pyo", "__pycache__",
		"*.class", "*.jar", "*.war", "*.ear",
		"*.o", "*.so", "*.dylib", "*.dll", "*.exe",
		"*.zip", "*.tar.gz", "*.rar", "*.7z",
		"*.jpg", "*.jpeg", "*.png", "*.gif", "*.bmp", "*.ico",
		"*.mp3", "*.mp4", "*.avi", "*.mov", "*.wmv", "*.flv",
		"*.pdf", "*.doc", "*.docx", "*.xls", "*.xlsx", "*.ppt", "*.pptx",
	}
}
