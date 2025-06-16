// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"context"
)

// Registry defines the interface for tool registration and retrieval
type Registry interface {
	// RegisterTool registers a tool with the given name
	RegisterTool(name string, tool Tool) error

	// GetTool retrieves a tool by name
	GetTool(name string) (Tool, error)

	// ListTools returns the names of all registered tools
	ListTools() []string

	// HasTool checks if a tool is registered
	HasTool(name string) bool

	// UnregisterTool removes a tool from the registry
	UnregisterTool(name string) error

	// Clear removes all tools from the registry
	Clear()
}

// ContextRegistry extends Registry with context-aware operations
type ContextRegistry interface {
	Registry

	// RegisterToolWithContext registers a tool with context metadata
	RegisterToolWithContext(ctx context.Context, name string, tool Tool, metadata map[string]interface{}) error

	// GetToolWithContext retrieves a tool with context awareness
	GetToolWithContext(ctx context.Context, name string) (Tool, error)
}
