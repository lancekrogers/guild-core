// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package edit

import (
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// RegisterEditTools registers all edit tools with the provided registry
func RegisterEditTools(registry *tools.ToolRegistry) error {
	// Create edit tools
	multiEditTool := NewMultiEditTool()
	applyDiffTool := NewApplyDiffTool()
	multiRefactorTool := NewMultiFileRefactorTool()

	// Register multi-edit tool
	if err := registry.RegisterTool(multiEditTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register multi-edit tool").
			WithComponent("tools.edit").
			WithOperation("register_tools").
			WithDetails("tool", "multi_edit")
	}

	// Register apply diff tool
	if err := registry.RegisterTool(applyDiffTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register apply diff tool").
			WithComponent("tools.edit").
			WithOperation("register_tools").
			WithDetails("tool", "apply_diff")
	}

	// Register multi-refactor tool
	if err := registry.RegisterTool(multiRefactorTool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register multi-refactor tool").
			WithComponent("tools.edit").
			WithOperation("register_tools").
			WithDetails("tool", "multi_refactor")
	}

	return nil
}

// GetEditTools returns all edit tools for registration with cost-aware registries
func GetEditTools() []EditToolInfo {
	return []EditToolInfo{
		{
			Tool:          NewMultiEditTool(),
			CostMagnitude: 0, // Local file operation, zero cost
			Capabilities:  []string{"file_operations", "edit", "find_replace", "atomic"},
		},
		{
			Tool:          NewApplyDiffTool(),
			CostMagnitude: 0, // Local file operation, zero cost
			Capabilities:  []string{"file_operations", "edit", "diff", "patch"},
		},
		{
			Tool:          NewMultiFileRefactorTool(),
			CostMagnitude: 1, // More complex operation with cross-file analysis
			Capabilities:  []string{"file_operations", "edit", "refactor", "analysis"},
		},
	}
}

// EditToolInfo contains tool information for cost-aware registration
type EditToolInfo struct {
	Tool          tools.Tool
	CostMagnitude int
	Capabilities  []string
}
