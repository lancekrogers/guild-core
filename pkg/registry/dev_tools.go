// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/tools"
	"github.com/guild-ventures/guild-core/tools/dev"
	"github.com/guild-ventures/guild-core/tools/git"
	"github.com/guild-ventures/guild-core/tools/search"
	"github.com/guild-ventures/guild-core/tools/shell"
)

// RegisterDevTools registers development workflow tools with the given tool registry
func RegisterDevTools(registry interface{}, llmProvider interfaces.AIProvider) error {
	// Type assert to get the tool registry
	toolRegistry, ok := registry.(*tools.ToolRegistry)
	if !ok {
		// Try pkg/tools registry
		if pkgRegistry, ok := registry.(interface{ RegisterTool(tools.Tool) error }); ok {
			return registerDevWithPkgRegistry(pkgRegistry, llmProvider)
		}
		return nil
	}

	// Test Runner Tool
	testRunner := dev.NewTestRunnerTool()
	if err := toolRegistry.RegisterTool(testRunner); err != nil {
		return err
	}

	// Streaming Shell Tool
	streamingShell := shell.NewStreamingShellTool(shell.ShellToolOptions{})
	if err := toolRegistry.RegisterTool(streamingShell); err != nil {
		return err
	}

	// Smart Git Commit Tool (only if LLM provider available)
	if llmProvider != nil {
		smartCommit := git.NewSmartCommitTool(llmProvider)
		if err := toolRegistry.RegisterTool(smartCommit); err != nil {
			return err
		}
	}

	// Silver Searcher (ag) Tool
	agTool := search.NewAgTool("")
	if err := toolRegistry.RegisterTool(agTool); err != nil {
		return err
	}

	return nil
}

func registerDevWithPkgRegistry(registry interface{ RegisterTool(tools.Tool) error }, llmProvider interfaces.AIProvider) error {
	// Test Runner Tool
	testRunner := dev.NewTestRunnerTool()
	if err := registry.RegisterTool(testRunner); err != nil {
		return err
	}

	// Streaming Shell Tool
	streamingShell := shell.NewStreamingShellTool(shell.ShellToolOptions{})
	if err := registry.RegisterTool(streamingShell); err != nil {
		return err
	}

	// Smart Git Commit Tool (only if LLM provider available)
	if llmProvider != nil {
		smartCommit := git.NewSmartCommitTool(llmProvider)
		if err := registry.RegisterTool(smartCommit); err != nil {
			return err
		}
	}

	// Silver Searcher (ag) Tool
	agTool := search.NewAgTool("")
	if err := registry.RegisterTool(agTool); err != nil {
		return err
	}

	return nil
}

// GetDevToolNames returns the names of all development tools
func GetDevToolNames() []string {
	return []string{
		"test_runner",
		"streaming_shell",
		"smart_commit",
		"ag",
	}
}

// GetDevToolsByCategory returns development tools grouped by category
func GetDevToolsByCategory() map[string][]string {
	return map[string][]string{
		"testing": {
			"test_runner",
		},
		"shell": {
			"streaming_shell",
		},
		"git": {
			"smart_commit",
		},
		"search": {
			"ag",
		},
	}
}
