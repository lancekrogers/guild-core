// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package tools provides tool management for MCP
package tools

import (
	"context"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/mcp/protocol"
)

// Tool represents a callable tool in the MCP system
type Tool interface {
	// ID returns the unique tool identifier
	ID() string

	// Name returns the human-readable tool name
	Name() string

	// Description returns the tool description
	Description() string

	// Capabilities returns the tool's capabilities
	Capabilities() []string

	// Execute executes the tool with given parameters
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)

	// HealthCheck checks if the tool is healthy
	HealthCheck() error

	// GetCostProfile returns the tool's cost profile
	GetCostProfile() protocol.CostProfile

	// GetParameters returns the tool's parameter definitions
	GetParameters() []protocol.ToolParameter

	// GetReturns returns the tool's return value definitions
	GetReturns() []protocol.ToolParameter
}

// Registry manages tool registration and discovery
type Registry interface {
	// RegisterTool registers a tool
	RegisterTool(tool Tool) error

	// DeregisterTool removes a tool
	DeregisterTool(toolID string) error

	// GetTool retrieves a tool by ID
	GetTool(toolID string) (Tool, error)

	// DiscoverTools finds tools matching criteria
	DiscoverTools(criteria protocol.ToolQuery) ([]Tool, error)

	// ListTools returns all registered tools
	ListTools() []Tool

	// UpdateToolStatus updates a tool's availability
	UpdateToolStatus(toolID string, available bool) error
}

// MemoryRegistry implements an in-memory tool registry
type MemoryRegistry struct {
	tools     map[string]Tool
	status    map[string]bool // tool availability status
	mu        sync.RWMutex
	indexCaps map[string][]string // capability -> tool IDs index
	indexTags map[string][]string // tag -> tool IDs index
}

// NewMemoryRegistry creates a new in-memory registry
func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{
		tools:     make(map[string]Tool),
		status:    make(map[string]bool),
		indexCaps: make(map[string][]string),
		indexTags: make(map[string][]string),
	}
}

// RegisterTool registers a tool in the registry
func (r *MemoryRegistry) RegisterTool(tool Tool) error {
	if tool == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "tool cannot be nil", nil).
			WithComponent("mcp.tools.registry").
			WithOperation("RegisterTool")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	toolID := tool.ID()
	if _, exists := r.tools[toolID]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "tool already registered", nil).
			WithComponent("mcp.tools.registry").
			WithOperation("RegisterTool").
			WithDetails("tool_id", toolID)
	}

	// Register the tool
	r.tools[toolID] = tool
	r.status[toolID] = true // Available by default

	// Update capability index
	for _, cap := range tool.Capabilities() {
		r.indexCaps[cap] = append(r.indexCaps[cap], toolID)
	}

	return nil
}

// DeregisterTool removes a tool from the registry
func (r *MemoryRegistry) DeregisterTool(toolID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tool, exists := r.tools[toolID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "tool not found", nil).
			WithComponent("mcp.tools.registry").
			WithOperation("DeregisterTool").
			WithDetails("tool_id", toolID)
	}

	// Remove from tools map
	delete(r.tools, toolID)
	delete(r.status, toolID)

	// Update capability index
	for _, cap := range tool.Capabilities() {
		if capTools, exists := r.indexCaps[cap]; exists {
			r.indexCaps[cap] = r.removeFromSlice(capTools, toolID)
			if len(r.indexCaps[cap]) == 0 {
				delete(r.indexCaps, cap)
			}
		}
	}

	return nil
}

// GetTool retrieves a tool by ID
func (r *MemoryRegistry) GetTool(toolID string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[toolID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "tool not found", nil).
			WithComponent("mcp.tools.registry").
			WithOperation("GetTool").
			WithDetails("tool_id", toolID)
	}

	return tool, nil
}

// DiscoverTools finds tools matching the given criteria
func (r *MemoryRegistry) DiscoverTools(criteria protocol.ToolQuery) ([]Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Start with all tools
	candidates := make(map[string]Tool)

	// Filter by required capabilities
	if len(criteria.RequiredCapabilities) > 0 {
		// Find tools that have ALL required capabilities
		for i, cap := range criteria.RequiredCapabilities {
			toolIDs := r.indexCaps[cap]
			if i == 0 {
				// First capability - add all tools
				for _, id := range toolIDs {
					if tool, exists := r.tools[id]; exists {
						candidates[id] = tool
					}
				}
			} else {
				// Subsequent capabilities - keep only tools that have this cap too
				newCandidates := make(map[string]Tool)
				for _, id := range toolIDs {
					if tool, exists := candidates[id]; exists {
						newCandidates[id] = tool
					}
				}
				candidates = newCandidates
			}
		}

		// If no tools have all capabilities, return empty
		if len(candidates) == 0 {
			return []Tool{}, nil
		}
	} else {
		// No capability filter - include all tools
		for id, tool := range r.tools {
			candidates[id] = tool
		}
	}

	// Apply additional filters
	var result []Tool
	for id, tool := range candidates {
		// Check availability
		if available, exists := r.status[id]; exists && !available {
			continue
		}

		// Check cost constraints
		profile := tool.GetCostProfile()
		if criteria.MaxCost > 0 && profile.FinancialCost > criteria.MaxCost {
			continue
		}
		if criteria.MaxLatency > 0 && profile.LatencyCost > criteria.MaxLatency {
			continue
		}

		// Tool passed all filters
		result = append(result, tool)
	}

	return result, nil
}

// ListTools returns all registered tools
func (r *MemoryRegistry) ListTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// UpdateToolStatus updates a tool's availability status
func (r *MemoryRegistry) UpdateToolStatus(toolID string, available bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[toolID]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "tool not found", nil).
			WithComponent("mcp.tools.registry").
			WithOperation("UpdateToolStatus").
			WithDetails("tool_id", toolID)
	}

	r.status[toolID] = available
	return nil
}

// removeFromSlice removes an element from a slice
func (r *MemoryRegistry) removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// BaseTool provides a base implementation of the Tool interface
type BaseTool struct {
	id           string
	name         string
	description  string
	capabilities []string
	costProfile  protocol.CostProfile
	parameters   []protocol.ToolParameter
	returns      []protocol.ToolParameter
	executor     func(context.Context, map[string]interface{}) (interface{}, error)
}

// NewBaseTool creates a new base tool
func NewBaseTool(
	id, name, description string,
	capabilities []string,
	costProfile protocol.CostProfile,
	parameters []protocol.ToolParameter,
	returns []protocol.ToolParameter,
	executor func(context.Context, map[string]interface{}) (interface{}, error),
) *BaseTool {
	return &BaseTool{
		id:           id,
		name:         name,
		description:  description,
		capabilities: capabilities,
		costProfile:  costProfile,
		parameters:   parameters,
		returns:      returns,
		executor:     executor,
	}
}

// ID returns the tool ID
func (t *BaseTool) ID() string {
	return t.id
}

// Name returns the tool name
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *BaseTool) Description() string {
	return t.description
}

// Capabilities returns the tool capabilities
func (t *BaseTool) Capabilities() []string {
	return t.capabilities
}

// Execute executes the tool
func (t *BaseTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if t.executor == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "tool has no executor", nil).
			WithComponent("mcp.tools.basetool").
			WithOperation("Execute").
			WithDetails("tool_id", t.id)
	}

	// Validate required parameters
	for _, param := range t.parameters {
		if param.Required {
			if _, exists := params[param.Name]; !exists {
				return nil, gerror.New(gerror.ErrCodeMissingRequired, "required parameter missing", nil).
					WithComponent("mcp.tools.basetool").
					WithOperation("Execute").
					WithDetails("parameter_name", param.Name).
					WithDetails("tool_id", t.id)
			}
		}
	}

	return t.executor(ctx, params)
}

// HealthCheck performs a health check
func (t *BaseTool) HealthCheck() error {
	// Basic implementation - can be overridden
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := t.Execute(ctx, map[string]interface{}{
		"_health_check": true,
	})

	return err
}

// GetCostProfile returns the cost profile
func (t *BaseTool) GetCostProfile() protocol.CostProfile {
	return t.costProfile
}

// GetParameters returns parameter definitions
func (t *BaseTool) GetParameters() []protocol.ToolParameter {
	return t.parameters
}

// GetReturns returns return value definitions
func (t *BaseTool) GetReturns() []protocol.ToolParameter {
	return t.returns
}
