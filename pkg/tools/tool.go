package tools

import (
	"context"
	"fmt"
	
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/tools"
)

// Re-export the Tool interface from the tools package
type Tool = tools.Tool

// Re-export the ToolResult type from the tools package
type ToolResult = tools.ToolResult

// ToolRegistry manages tools for agents with cost tracking
type ToolRegistry struct {
	// Embed the original tool registry
	*tools.ToolRegistry
	
	// Tool costs (tool name -> cost per use)
	toolCosts map[string]float64
}

// newToolRegistry creates a new tool registry with cost tracking (private constructor)
func newToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		ToolRegistry: tools.NewToolRegistry(),
		toolCosts:    make(map[string]float64),
	}
}

// NewToolRegistry creates a new tool registry with cost tracking (public constructor)
func NewToolRegistry() *ToolRegistry {
	return newToolRegistry()
}

// DefaultToolRegistryFactory creates a tool registry for registry use
func DefaultToolRegistryFactory() Registry {
	return newToolRegistry()
}

// RegisterTool registers a tool with the registry (implements Registry interface)
func (r *ToolRegistry) RegisterTool(name string, tool Tool) error {
	// The underlying registry uses the tool's Name() method, 
	// so we validate that it matches the provided name
	if tool.Name() != name {
		return fmt.Errorf("tool name mismatch: provided '%s', tool reports '%s'", name, tool.Name())
	}
	return r.ToolRegistry.RegisterTool(tool)
}

// Register registers a tool using its own name
// This provides compatibility with the base registry's signature
func (r *ToolRegistry) Register(tool Tool) error {
	return r.ToolRegistry.RegisterTool(tool)
}

// GetTool retrieves a tool by name (implements Registry interface)
func (r *ToolRegistry) GetTool(name string) (Tool, error) {
	tool, exists := r.ToolRegistry.GetTool(name)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return tool, nil
}

// ListTools returns the names of all registered tools (implements Registry interface)
func (r *ToolRegistry) ListTools() []string {
	tools := r.ToolRegistry.ListTools()
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}
	return names
}

// HasTool checks if a tool is registered (implements Registry interface)
func (r *ToolRegistry) HasTool(name string) bool {
	_, exists := r.ToolRegistry.GetTool(name)
	return exists
}

// UnregisterTool removes a tool from the registry (implements Registry interface)
func (r *ToolRegistry) UnregisterTool(name string) error {
	if !r.HasTool(name) {
		return fmt.Errorf("tool '%s' not found", name)
	}
	// The underlying registry doesn't have UnregisterTool, so we need to manage this
	// For now, return an error indicating it's not supported
	return fmt.Errorf("unregister not supported by underlying registry")
}

// Clear removes all tools from the registry (implements Registry interface)
func (r *ToolRegistry) Clear() {
	// The underlying registry doesn't have Clear, so we need to recreate it
	r.ToolRegistry = tools.NewToolRegistry()
	r.toolCosts = make(map[string]float64)
}

// RegisterToolWithCost registers a tool with a specific cost
func (r *ToolRegistry) RegisterToolWithCost(tool Tool, costPerUse float64) error {
	err := r.ToolRegistry.RegisterTool(tool)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register tool with cost").
			WithComponent("tools").
			WithOperation("RegisterToolWithCost").
			WithDetails("tool_name", tool.Name()).
			WithDetails("cost_per_use", fmt.Sprintf("%.2f", costPerUse))
	}
	
	r.toolCosts[tool.Name()] = costPerUse
	return nil
}

// GetToolCost returns the cost for using a specific tool
func (r *ToolRegistry) GetToolCost(toolName string) float64 {
	if cost, ok := r.toolCosts[toolName]; ok {
		return cost
	}
	// Default cost if not specified
	return 0.01
}

// SetToolCost sets the cost for using a specific tool
func (r *ToolRegistry) SetToolCost(toolName string, cost float64) {
	r.toolCosts[toolName] = cost
}

// ExecuteToolWithCostTracking executes a tool and returns the result with cost information
func (r *ToolRegistry) ExecuteToolWithCostTracking(ctx context.Context, name string, input string) (*ToolResult, float64, error) {
	result, err := r.ExecuteTool(ctx, name, input)
	cost := r.GetToolCost(name)
	
	return result, cost, err
}

// ExecuteToolWithParams executes a tool by name with the given parameters as a JSON object
func (r *ToolRegistry) ExecuteToolWithParams(ctx context.Context, name string, params map[string]interface{}) (*ToolResult, error) {
	return r.ToolRegistry.ExecuteToolWithParams(ctx, name, params)
}

// ExecuteToolWithParamsAndCostTracking executes a tool with params and returns the result with cost information
func (r *ToolRegistry) ExecuteToolWithParamsAndCostTracking(ctx context.Context, name string, params map[string]interface{}) (*ToolResult, float64, error) {
	result, err := r.ExecuteToolWithParams(ctx, name, params)
	cost := r.GetToolCost(name)
	
	return result, cost, err
}