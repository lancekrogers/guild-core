package tools

import (
	"context"
	"encoding/json"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Output    string                 `json:"output"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Success   bool                   `json:"success"`
	ExtraData map[string]interface{} `json:"extra_data,omitempty"`
}

// Tool is an interface for tools that can be used by agents
type Tool interface {
	// Name returns the name of the tool
	Name() string

	// Description returns a description of what the tool does
	Description() string

	// Schema returns the JSON schema for the tool's input parameters
	Schema() map[string]interface{}

	// Execute runs the tool with the given input and returns the result
	Execute(ctx context.Context, input string) (*ToolResult, error)

	// Examples returns a list of example inputs for the tool
	Examples() []string

	// Category returns the category of the tool (e.g., "file", "web", "code")
	Category() string

	// RequiresAuth returns whether the tool requires authentication
	RequiresAuth() bool
}

// BaseTool provides a base implementation of the Tool interface
type BaseTool struct {
	name        string
	description string
	schema      map[string]interface{}
	category    string
	needsAuth   bool
	examples    []string
}

// NewBaseTool creates a new base tool
func NewBaseTool(name, description string, schema map[string]interface{}, category string, needsAuth bool, examples []string) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
		category:    category,
		needsAuth:   needsAuth,
		examples:    examples,
	}
}

// Name returns the name of the tool
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns a description of what the tool does
func (t *BaseTool) Description() string {
	return t.description
}

// Schema returns the JSON schema for the tool's input parameters
func (t *BaseTool) Schema() map[string]interface{} {
	return t.schema
}

// Category returns the category of the tool
func (t *BaseTool) Category() string {
	return t.category
}

// RequiresAuth returns whether the tool requires authentication
func (t *BaseTool) RequiresAuth() bool {
	return t.needsAuth
}

// Examples returns a list of example inputs for the tool
func (t *BaseTool) Examples() []string {
	return t.examples
}

// Execute implements the Tool interface but should be overridden by concrete tools
func (t *BaseTool) Execute(ctx context.Context, input string) (*ToolResult, error) {
	return nil, gerror.New(gerror.ErrCodeInternal, "Execute not implemented for BaseTool, must be implemented by concrete tool", nil).
		WithComponent("tools").
		WithOperation("execute")
}

// NewToolResult creates a new tool result
func NewToolResult(output string, metadata map[string]string, err error, extraData map[string]interface{}) *ToolResult {
	result := &ToolResult{
		Output:    output,
		Metadata:  metadata,
		Success:   err == nil,
		ExtraData: extraData,
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// RegisterTool registers a tool with the registry
func (r *ToolRegistry) RegisterTool(tool Tool) error {
	if tool == nil {
		return gerror.New(gerror.ErrCodeValidation, "tool cannot be nil", nil).
			WithComponent("tools").
			WithOperation("register_tool")
	}

	name := tool.Name()
	if name == "" {
		return gerror.New(gerror.ErrCodeValidation, "tool name cannot be empty", nil).
			WithComponent("tools").
			WithOperation("register_tool")
	}

	if _, exists := r.tools[name]; exists {
		return gerror.Newf(gerror.ErrCodeAlreadyExists, "tool with name '%s' already registered", name).
			WithComponent("tools").
			WithOperation("register_tool")
	}

	r.tools[name] = tool
	return nil
}

// GetTool gets a tool by name
func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// ListTools returns a list of all registered tools
func (r *ToolRegistry) ListTools() []Tool {
	var tools []Tool
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListToolsByCategory returns a list of tools in a specific category
func (r *ToolRegistry) ListToolsByCategory(category string) []Tool {
	var tools []Tool
	for _, tool := range r.tools {
		if tool.Category() == category {
			tools = append(tools, tool)
		}
	}
	return tools
}

// ExecuteTool executes a tool by name with the given input
func (r *ToolRegistry) ExecuteTool(ctx context.Context, name string, input string) (*ToolResult, error) {
	tool, exists := r.GetTool(name)
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "tool '%s' not found", name).
			WithComponent("tools").
			WithOperation("get_tool")
	}

	return tool.Execute(ctx, input)
}

// ExecuteToolWithParams executes a tool by name with the given parameters as a JSON object
func (r *ToolRegistry) ExecuteToolWithParams(ctx context.Context, name string, params map[string]interface{}) (*ToolResult, error) {
	// Convert params to JSON
	inputJSON, err := json.Marshal(params)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal parameters").
			WithComponent("tools").
			WithOperation("get_schema")
	}

	return r.ExecuteTool(ctx, name, string(inputJSON))
}