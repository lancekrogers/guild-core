// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"context"

	"github.com/lancekrogers/guild-core/tools"
)

// LSPTool interface that all LSP tools implement
type LSPTool interface {
	tools.Tool
}

// registryAdapter adapts an LSP tool to the registry tool interface
type registryAdapter struct {
	tool tools.Tool
}

// ToRegistryTool adapts an LSP tool to work with the tool registry
func ToRegistryTool(tool tools.Tool) tools.Tool {
	return &registryAdapter{tool: tool}
}

// Name returns the name of the tool
func (a *registryAdapter) Name() string {
	return a.tool.Name()
}

// Description returns a description of what the tool does
func (a *registryAdapter) Description() string {
	return a.tool.Description()
}

// Schema returns the JSON schema for the tool's input parameters
func (a *registryAdapter) Schema() map[string]interface{} {
	return a.tool.Schema()
}

// Execute runs the tool with the given input and returns the result
func (a *registryAdapter) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	return a.tool.Execute(ctx, input)
}

// Examples returns a list of example inputs for the tool
func (a *registryAdapter) Examples() []string {
	return a.tool.Examples()
}

// HealthCheck verifies the tool is functional
func (a *registryAdapter) HealthCheck() error {
	return a.tool.HealthCheck()
}

// Category returns the category of the tool
func (a *registryAdapter) Category() string {
	return a.tool.Category()
}

// RequiresAuth returns whether the tool requires authentication
func (a *registryAdapter) RequiresAuth() bool {
	return a.tool.RequiresAuth()
}

// RequiresConfirmation returns whether the tool requires user confirmation
func (a *registryAdapter) RequiresConfirmation() bool {
	// Extend the interface if needed
	if confirmable, ok := a.tool.(interface{ RequiresConfirmation() bool }); ok {
		return confirmable.RequiresConfirmation()
	}
	return false
}
