// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/tools"
)

// MockToolRegistry implements the tools.ToolRegistry interface for testing
type MockToolRegistry struct {
	mu              sync.RWMutex
	tools           map[string]tools.Tool
	executionResult *tools.ToolResult
	executionError  error
}

// NewMockToolRegistry creates a new mock tool registry
func NewMockToolRegistry() *MockToolRegistry {
	return &MockToolRegistry{
		tools: make(map[string]tools.Tool),
		executionResult: &tools.ToolResult{
			Success: true,
			Output:  "Tool executed successfully",
		},
	}
}

// WithTool adds a tool to the registry
func (m *MockToolRegistry) WithTool(tool tools.Tool) *MockToolRegistry {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools[tool.Name()] = tool
	return m
}

// WithToolResult sets the result to return from tool execution
func (m *MockToolRegistry) WithToolResult(result *tools.ToolResult) *MockToolRegistry {
	m.executionResult = result
	return m
}

// WithExecutionError sets an error to return from tool execution
func (m *MockToolRegistry) WithExecutionError(err error) *MockToolRegistry {
	m.executionError = err
	return m
}

// RegisterTool implements tools.ToolRegistry.RegisterTool
func (m *MockToolRegistry) RegisterTool(tool tools.Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tool == nil {
		return gerror.New(gerror.ErrCodeValidation, "tool cannot be nil", nil).
			WithComponent("tools").
			WithOperation("RegisterTool")
	}

	m.tools[tool.Name()] = tool
	return nil
}

// GetTool implements tools.ToolRegistry.GetTool
func (m *MockToolRegistry) GetTool(name string) (tools.Tool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, ok := m.tools[name]
	return tool, ok
}

// ListTools implements tools.ToolRegistry.ListTools
func (m *MockToolRegistry) ListTools() []tools.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var toolList []tools.Tool
	for _, tool := range m.tools {
		toolList = append(toolList, tool)
	}

	return toolList
}

// ExecuteTool implements tools.ToolRegistry.ExecuteTool
func (m *MockToolRegistry) ExecuteTool(ctx context.Context, name string, input string) (*tools.ToolResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.executionError != nil {
		return nil, m.executionError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if tool exists
	_, ok := m.tools[name]
	if !ok {
		return nil, gerror.New(gerror.ErrCodeNotFound, "tool not found", nil).
			WithComponent("tools").
			WithOperation("ExecuteTool").
			WithDetails("tool_name", name)
	}

	// Return predefined result
	return m.executionResult, nil
}

// ExecuteToolWithParams implements tools.ToolRegistry.ExecuteToolWithParams
func (m *MockToolRegistry) ExecuteToolWithParams(ctx context.Context, name string, params map[string]interface{}) (*tools.ToolResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.executionError != nil {
		return nil, m.executionError
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if tool exists
	_, ok := m.tools[name]
	if !ok {
		return nil, gerror.New(gerror.ErrCodeNotFound, "tool not found", nil).
			WithComponent("tools").
			WithOperation("ExecuteTool").
			WithDetails("tool_name", name)
	}

	// Return predefined result
	return m.executionResult, nil
}

// MockTool implements the tools.Tool interface for testing
type MockTool struct {
	name        string
	description string
	params      map[string]string
	result      *tools.ToolResult
	err         error
}

// NewMockTool creates a new mock tool
func NewMockTool(name, description string) *MockTool {
	return &MockTool{
		name:        name,
		description: description,
		params:      make(map[string]string),
		result: &tools.ToolResult{
			Success: true,
			Output:  "Tool executed successfully",
		},
	}
}

// WithResult sets the result to return from Execute
func (t *MockTool) WithResult(result *tools.ToolResult) *MockTool {
	t.result = result
	return t
}

// WithError sets an error to return from Execute
func (t *MockTool) WithError(err error) *MockTool {
	t.err = err
	return t
}

// Name implements tools.Tool.Name
func (t *MockTool) Name() string {
	return t.name
}

// Description implements tools.Tool.Description
func (t *MockTool) Description() string {
	return t.description
}

// Parameters implements tools.Tool.Parameters
func (t *MockTool) Parameters() map[string]string {
	return t.params
}

// Execute implements tools.Tool.Execute
func (t *MockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if t.err != nil {
		return nil, t.err
	}

	return t.result, nil
}

// ExecuteWithParams implements tools.Tool.ExecuteWithParams
func (t *MockTool) ExecuteWithParams(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if t.err != nil {
		return nil, t.err
	}

	return t.result, nil
}
