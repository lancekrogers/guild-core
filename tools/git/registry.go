// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package git

import (
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// RegisterGitTools registers all git tools with the provided registry
func RegisterGitTools(registry *tools.ToolRegistry, workspacePath string) error {
	// Create git tools
	gitLogTool := NewGitLogTool(workspacePath)
	gitBlameTool := NewGitBlameTool(workspacePath)
	gitMergeConflictsTool := NewGitMergeConflictsTool(workspacePath)

	// Register git log tool
	if err := registry.RegisterTool(gitLogTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register git log tool").
			WithComponent("tools.git").
			WithOperation("register_tools").
			WithDetails("tool", "git_log")
	}

	// Register git blame tool
	if err := registry.RegisterTool(gitBlameTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register git blame tool").
			WithComponent("tools.git").
			WithOperation("register_tools").
			WithDetails("tool", "git_blame")
	}

	// Register git merge conflicts tool
	if err := registry.RegisterTool(gitMergeConflictsTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register git merge conflicts tool").
			WithComponent("tools.git").
			WithOperation("register_tools").
			WithDetails("tool", "git_merge_conflicts")
	}

	return nil
}

// GetGitTools returns all git tools for registration with cost-aware registries
func GetGitTools(workspacePath string) []GitToolInfo {
	return []GitToolInfo{
		{
			Tool:          NewGitLogTool(workspacePath),
			CostMagnitude: 0, // Local operation, zero cost
			Capabilities:  []string{"version_control", "history", "search"},
		},
		{
			Tool:          NewGitBlameTool(workspacePath),
			CostMagnitude: 0, // Local operation, zero cost
			Capabilities:  []string{"version_control", "authorship", "analysis"},
		},
		{
			Tool:          NewGitMergeConflictsTool(workspacePath),
			CostMagnitude: 0, // Local operation, zero cost
			Capabilities:  []string{"version_control", "conflict_resolution", "merge"},
		},
	}
}

// GitToolInfo contains tool information for cost-aware registration
type GitToolInfo struct {
	Tool          tools.Tool
	CostMagnitude int
	Capabilities  []string
}
