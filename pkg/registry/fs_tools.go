// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package registry

import (
	"os"

	"github.com/guild-ventures/guild-core/tools"
	"github.com/guild-ventures/guild-core/tools/fs"
)

// RegisterFSTools registers all filesystem tools with the given tool registry
func RegisterFSTools(registry interface{}) error {
	// Type assert to get the tool registry
	toolRegistry, ok := registry.(*tools.ToolRegistry)
	if !ok {
		// Try pkg/tools registry
		if pkgRegistry, ok := registry.(interface{ RegisterTool(tools.Tool) error }); ok {
			return registerFSWithPkgRegistry(pkgRegistry)
		}
		return nil
	}

	// Get current working directory as default base path
	basePath, err := os.Getwd()
	if err != nil {
		basePath = "." // fallback to current directory
	}

	// Register file tool
	fileTool := fs.NewFileTool(basePath)
	if err := toolRegistry.RegisterTool(fileTool); err != nil {
		return err
	}

	// Register glob tool
	globTool := fs.NewGlobTool(basePath)
	if err := toolRegistry.RegisterTool(globTool); err != nil {
		return err
	}

	// Register grep tool
	grepTool := fs.NewGrepTool(basePath)
	if err := toolRegistry.RegisterTool(grepTool); err != nil {
		return err
	}

	return nil
}

func registerFSWithPkgRegistry(registry interface{ RegisterTool(tools.Tool) error }) error {
	// Get current working directory as default base path
	basePath, err := os.Getwd()
	if err != nil {
		basePath = "." // fallback to current directory
	}

	// Register file tool
	fileTool := fs.NewFileTool(basePath)
	if err := registry.RegisterTool(fileTool); err != nil {
		return err
	}

	// Register glob tool
	globTool := fs.NewGlobTool(basePath)
	if err := registry.RegisterTool(globTool); err != nil {
		return err
	}

	// Register grep tool
	grepTool := fs.NewGrepTool(basePath)
	if err := registry.RegisterTool(grepTool); err != nil {
		return err
	}

	return nil
}

// GetFSToolNames returns the names of all registered filesystem tools
func GetFSToolNames() []string {
	return []string{
		"file",
		"glob",
		"grep",
	}
}

// GetFSToolsByCategory returns filesystem tools grouped by category
func GetFSToolsByCategory() map[string][]string {
	return map[string][]string{
		"filesystem": {
			"file",
			"glob",
			"grep",
		},
	}
}
