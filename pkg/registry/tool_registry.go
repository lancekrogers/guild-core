package registry

import (
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/tools"
	basetools "github.com/guild-ventures/guild-core/tools"
)

// DefaultToolRegistry implements the ToolRegistry interface by wrapping the existing tool registry
type DefaultToolRegistry struct {
	registry *tools.ToolRegistry
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() ToolRegistry {
	return &DefaultToolRegistry{
		registry: tools.NewToolRegistry(),
	}
}

// RegisterTool registers a tool with the registry
func (r *DefaultToolRegistry) RegisterTool(name string, tool Tool) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	// Convert the registry Tool interface to the actual tool interface
	// This is a wrapper to adapt between interfaces
	actualTool, ok := tool.(basetools.Tool)
	if !ok {
		return fmt.Errorf("tool does not implement the expected Tool interface")
	}

	return r.registry.RegisterTool(actualTool)
}

// GetTool retrieves a registered tool by name
func (r *DefaultToolRegistry) GetTool(name string) (Tool, error) {
	tool, exists := r.registry.GetTool(name)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	// Return the tool (it already implements the Tool interface)
	return tool, nil
}

// ListTools returns all registered tool names
func (r *DefaultToolRegistry) ListTools() []string {
	tools := r.registry.ListTools()
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}
	return names
}

// GetToolsByCapability returns tools that have a specific capability
func (r *DefaultToolRegistry) GetToolsByCapability(capability string) []Tool {
	allTools := r.registry.ListTools()
	var matchingTools []Tool

	for _, tool := range allTools {
		// Check if tool has the capability
		// This assumes tools have a way to report their capabilities
		// You might need to adapt this based on your actual Tool interface
		if tool.Category() == capability {
			matchingTools = append(matchingTools, tool)
		}
	}

	return matchingTools
}

// HasTool checks if a tool is registered
func (r *DefaultToolRegistry) HasTool(name string) bool {
	_, exists := r.registry.GetTool(name)
	return exists
}

// GetUnderlyingRegistry returns the underlying tool registry for direct access
// This is useful for components that need the full tool registry functionality
func (r *DefaultToolRegistry) GetUnderlyingRegistry() *tools.ToolRegistry {
	return r.registry
}

// RegisterToolWithCost registers a tool with a specific cost (uses the existing cost tracking)
func (r *DefaultToolRegistry) RegisterToolWithCost(tool Tool, costPerUse float64) error {
	actualTool, ok := tool.(basetools.Tool)
	if !ok {
		return fmt.Errorf("tool does not implement the expected Tool interface")
	}

	return r.registry.RegisterToolWithCost(actualTool, costPerUse)
}

// GetToolCost returns the cost for using a specific tool
func (r *DefaultToolRegistry) GetToolCost(toolName string) float64 {
	return r.registry.GetToolCost(toolName)
}

// SetToolCost sets the cost for using a specific tool
func (r *DefaultToolRegistry) SetToolCost(toolName string, cost float64) {
	r.registry.SetToolCost(toolName, cost)
}