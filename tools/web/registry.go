// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package web

import (
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/tools"
)

// RegisterWebTools registers all web-related tools with the given registry
func RegisterWebTools(toolRegistry *tools.ToolRegistry, aiProvider providers.AIProvider) error {
	// Register WebSearch tool
	webSearchTool := NewWebSearchTool()
	if err := toolRegistry.RegisterTool(webSearchTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register WebSearch tool").
			WithComponent("web_tools").
			WithOperation("RegisterWebTools")
	}

	// Register WebFetch tool
	webFetchTool := NewWebFetchTool(aiProvider)
	if err := toolRegistry.RegisterTool(webFetchTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register WebFetch tool").
			WithComponent("web_tools").
			WithOperation("RegisterWebTools")
	}

	return nil
}

// RegisterWebToolsWithRegistry registers web tools with the DefaultToolRegistry
func RegisterWebToolsWithRegistry(registry registry.ToolRegistry, aiProvider providers.AIProvider) error {
	// Register WebSearch tool with cost information
	webSearchTool := NewWebSearchTool()
	err := registry.RegisterToolWithCost(
		webSearchTool.Name(),
		webSearchTool,
		1, // Low cost - basic web search
		[]string{"web", "search", "information_gathering"},
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register WebSearch tool with cost").
			WithComponent("web_tools").
			WithOperation("RegisterWebToolsWithRegistry")
	}

	// Register WebFetch tool with cost information
	webFetchTool := NewWebFetchTool(aiProvider)
	err = registry.RegisterToolWithCost(
		webFetchTool.Name(),
		webFetchTool,
		2, // Medium cost - involves AI analysis
		[]string{"web", "fetch", "analysis", "content_extraction"},
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register WebFetch tool with cost").
			WithComponent("web_tools").
			WithOperation("RegisterWebToolsWithRegistry")
	}

	return nil
}

// GetWebSearchTool creates and returns a WebSearch tool instance
func GetWebSearchTool() *WebSearchTool {
	return NewWebSearchTool()
}

// GetWebFetchTool creates and returns a WebFetch tool instance
func GetWebFetchTool(aiProvider providers.AIProvider) *WebFetchTool {
	return NewWebFetchTool(aiProvider)
}

// ListWebTools returns the names of all web tools
func ListWebTools() []string {
	return []string{
		"web_search",
		"web_fetch",
	}
}

// GetWebToolsInfo returns information about web tools for documentation
func GetWebToolsInfo() map[string]ToolInfo {
	return map[string]ToolInfo{
		"web_search": {
			Name:         "web_search",
			Description:  "Search the web using multiple search engines with domain filtering",
			Category:     "web",
			CostLevel:    1,
			Capabilities: []string{"web", "search", "information_gathering"},
			RequiresAuth: false,
			RequiresAI:   false,
		},
		"web_fetch": {
			Name:         "web_fetch",
			Description:  "Fetch and analyze web content using AI",
			Category:     "web",
			CostLevel:    2,
			Capabilities: []string{"web", "fetch", "analysis", "content_extraction"},
			RequiresAuth: false,
			RequiresAI:   true,
		},
	}
}

// ToolInfo represents information about a web tool
type ToolInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Category     string   `json:"category"`
	CostLevel    int      `json:"cost_level"`
	Capabilities []string `json:"capabilities"`
	RequiresAuth bool     `json:"requires_auth"`
	RequiresAI   bool     `json:"requires_ai"`
}
