package tools

import (
	"context"
	
	"github.com/blockhead-consulting/guild/tools"
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

// NewToolRegistry creates a new tool registry with cost tracking
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		ToolRegistry: tools.NewToolRegistry(),
		toolCosts:    make(map[string]float64),
	}
}

// RegisterTool registers a tool with the registry
func (r *ToolRegistry) RegisterTool(tool Tool) error {
	return r.ToolRegistry.RegisterTool(tool)
}

// RegisterToolWithCost registers a tool with a specific cost
func (r *ToolRegistry) RegisterToolWithCost(tool Tool, costPerUse float64) error {
	err := r.ToolRegistry.RegisterTool(tool)
	if err != nil {
		return err
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