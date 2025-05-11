package tools

// ToolRegistry manages tools for agents
type ToolRegistry struct {
	tools map[string]interface{}
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]interface{}),
	}
}

// RegisterTool registers a tool with the registry
func (r *ToolRegistry) RegisterTool(name string, tool interface{}) {
	r.tools[name] = tool
}

// GetTool returns a tool by name
func (r *ToolRegistry) GetTool(name string) (interface{}, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// ListTools returns a list of registered tool names
func (r *ToolRegistry) ListTools() []string {
	var names []string
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}