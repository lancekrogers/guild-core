package registry

import (
	"sort"
	"sync"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/tools"
	basetools "github.com/guild-ventures/guild-core/tools"
)

// DefaultToolRegistry implements the ToolRegistry interface by wrapping the existing tool registry
type DefaultToolRegistry struct {
	registry      *tools.ToolRegistry
	toolMetadata  map[string]ToolInfo // Cost and capability metadata for tools
	mu            sync.RWMutex
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() ToolRegistry {
	return &DefaultToolRegistry{
		registry:     tools.NewToolRegistry(),
		toolMetadata: make(map[string]ToolInfo),
	}
}

// RegisterTool registers a tool with the registry
func (r *DefaultToolRegistry) RegisterTool(name string, tool Tool) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "tool name cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterTool")
	}
	if tool == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "tool cannot be nil", nil).
			WithComponent("registry").
			WithOperation("RegisterTool")
	}

	// Convert the registry Tool interface to the actual tool interface
	// This is a wrapper to adapt between interfaces
	actualTool, ok := tool.(basetools.Tool)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidFormat, "tool does not implement the expected Tool interface", nil).
			WithComponent("registry").
			WithOperation("RegisterTool")
	}

	return r.registry.RegisterTool(actualTool)
}

// GetTool retrieves a registered tool by name
func (r *DefaultToolRegistry) GetTool(name string) (Tool, error) {
	tool, exists := r.registry.GetTool(name)
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "tool '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetTool").
			WithDetails("tool", name)
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

// RegisterToolWithLegacyCost registers a tool with a specific cost (uses the existing cost tracking)
// Deprecated: Use RegisterToolWithCost instead for Fibonacci cost magnitude
func (r *DefaultToolRegistry) RegisterToolWithLegacyCost(tool Tool, costPerUse float64) error {
	actualTool, ok := tool.(basetools.Tool)
	if !ok {
		return gerror.New(gerror.ErrCodeInvalidFormat, "tool does not implement the expected Tool interface", nil).
			WithComponent("registry").
			WithOperation("RegisterTool")
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

// Cost-based tool selection methods

// GetToolsByCost returns tools with cost magnitude <= maxCost, sorted by cost
func (r *DefaultToolRegistry) GetToolsByCost(maxCost int) []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tools []ToolInfo
	for _, toolInfo := range r.toolMetadata {
		if toolInfo.CostMagnitude <= maxCost && toolInfo.Available {
			tools = append(tools, toolInfo)
		}
	}

	// Sort by cost magnitude (ascending)
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].CostMagnitude < tools[j].CostMagnitude
	})

	return tools
}

// GetCheapestToolByCapability returns the lowest-cost tool with the given capability
func (r *DefaultToolRegistry) GetCheapestToolByCapability(capability string) (*ToolInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var cheapestTool *ToolInfo
	lowestCost := 999 // Higher than max Fibonacci value

	for _, toolInfo := range r.toolMetadata {
		if toolInfo.Available && r.hasToolCapability(toolInfo.Capabilities, capability) {
			if toolInfo.CostMagnitude < lowestCost {
				lowestCost = toolInfo.CostMagnitude
				toolCopy := toolInfo
				cheapestTool = &toolCopy
			}
		}
	}

	if cheapestTool == nil {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no tool found with capability '%s'", capability).
			WithComponent("registry").
			WithOperation("GetCheapestToolByCapability").
			WithDetails("capability", capability)
	}

	return cheapestTool, nil
}

// RegisterToolWithCost registers a tool with cost information and capabilities
func (r *DefaultToolRegistry) RegisterToolWithCost(name string, tool Tool, costMagnitude int, capabilities []string) error {
	if name == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "tool name cannot be empty", nil).
			WithComponent("registry").
			WithOperation("RegisterTool")
	}
	if tool == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "tool cannot be nil", nil).
			WithComponent("registry").
			WithOperation("RegisterTool")
	}
	
	// Validate cost magnitude (Fibonacci scale)
	if costMagnitude != 0 {
		validCosts := map[int]bool{1: true, 2: true, 3: true, 5: true, 8: true}
		if !validCosts[costMagnitude] {
			return gerror.Newf(gerror.ErrCodeInvalidInput, "invalid cost_magnitude: %d (must be 0 for free tools, or Fibonacci values: 1,2,3,5,8)", costMagnitude).
				WithComponent("registry").
				WithOperation("RegisterToolWithCost").
				WithDetails("costMagnitude", costMagnitude)
		}
	}

	// Register the tool with the underlying registry
	if err := r.RegisterTool(name, tool); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register tool").
			WithComponent("registry").
			WithOperation("RegisterToolWithCost").
			WithDetails("tool", name)
	}

	// Store metadata for cost-based selection
	r.mu.Lock()
	defer r.mu.Unlock()

	r.toolMetadata[name] = ToolInfo{
		Name:          name,
		Capabilities:  capabilities,
		CostMagnitude: costMagnitude,
		Available:     true,
		Tool:          tool,
	}

	return nil
}

// Helper methods

// hasToolCapability checks if the tool has a specific capability
func (r *DefaultToolRegistry) hasToolCapability(capabilities []string, target string) bool {
	for _, cap := range capabilities {
		if cap == target {
			return true
		}
	}
	return false
}

// GetToolInfo returns metadata for a specific tool
func (r *DefaultToolRegistry) GetToolInfo(name string) (*ToolInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	toolInfo, exists := r.toolMetadata[name]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "tool metadata for '%s' not found", name).
			WithComponent("registry").
			WithOperation("GetToolInfo").
			WithDetails("tool", name)
	}

	return &toolInfo, nil
}

// ListToolsWithMetadata returns all registered tools with their metadata
func (r *DefaultToolRegistry) ListToolsWithMetadata() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]ToolInfo, 0, len(r.toolMetadata))
	for _, toolInfo := range r.toolMetadata {
		tools = append(tools, toolInfo)
	}

	// Sort by cost magnitude for consistent ordering
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].CostMagnitude < tools[j].CostMagnitude
	})

	return tools
}

// SetToolAvailability sets whether a tool is currently available
func (r *DefaultToolRegistry) SetToolAvailability(name string, available bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	toolInfo, exists := r.toolMetadata[name]
	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "tool metadata for '%s' not found", name).
			WithComponent("registry").
			WithOperation("SetToolAvailability").
			WithDetails("tool", name)
	}

	toolInfo.Available = available
	r.toolMetadata[name] = toolInfo
	return nil
}

// RegisterBasicTools registers common tools with their cost information
func (r *DefaultToolRegistry) RegisterBasicTools() error {
	// This is a placeholder for registering common tools with cost metadata
	// In practice, you'd register actual tool implementations here
	
	// Example registrations (these would be actual tools):
	// Shell tools - zero cost
	// err := r.RegisterToolWithCost("shell", shellTool, 0, []string{"execution", "file_operations"})
	// File system tools - zero cost
	// err = r.RegisterToolWithCost("file_system", fsTool, 0, []string{"file_operations", "read", "write"})
	// Git tools - zero cost
	// err = r.RegisterToolWithCost("git", gitTool, 0, []string{"version_control", "collaboration"})
	// HTTP client - low cost
	// err = r.RegisterToolWithCost("http_client", httpTool, 1, []string{"network", "api"})
	
	return nil
}