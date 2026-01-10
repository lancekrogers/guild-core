// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/tools"
	"github.com/lancekrogers/guild-core/tools/jump"
)

// RegisterJumpTools registers the jump navigation tool with the given tool registry
func RegisterJumpTools(registry interface{}) error {
	// Type assert to get the tool registry
	toolRegistry, ok := registry.(*tools.ToolRegistry)
	if !ok {
		// Try pkg/tools registry
		if pkgRegistry, ok := registry.(interface{ RegisterTool(tools.Tool) error }); ok {
			return registerJumpWithPkgRegistry(pkgRegistry)
		}
		return nil
	}

	// Create and register jump tool
	jumpTool, err := jump.NewJumpTool()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create jump tool").
			WithComponent("registry").
			WithOperation("RegisterJumpTools")
	}

	if err := toolRegistry.RegisterTool(jumpTool); err != nil {
		// Clean up on failure
		jumpTool.Close()
		return err
	}

	return nil
}

func registerJumpWithPkgRegistry(registry interface{ RegisterTool(tools.Tool) error }) error {
	// Create and register jump tool
	jumpTool, err := jump.NewJumpTool()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create jump tool").
			WithComponent("registry").
			WithOperation("registerJumpWithPkgRegistry")
	}

	if err := registry.RegisterTool(jumpTool); err != nil {
		// Clean up on failure
		jumpTool.Close()
		return err
	}

	return nil
}

// GetJumpToolNames returns the names of all registered jump tools
func GetJumpToolNames() []string {
	return []string{
		"jump",
	}
}

// GetJumpToolsByCategory returns jump tools grouped by category
func GetJumpToolsByCategory() map[string][]string {
	return map[string][]string{
		"navigation": {
			"jump",
		},
	}
}
