package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/tools"
	"github.com/stretchr/testify/assert"
)

// Test executeWithTools method to improve coverage
func TestWorkerAgent_executeWithTools(t *testing.T) {
	tests := []struct {
		name        string
		setupAgent  func() *WorkerAgent
		request     string
		expectErr   bool
		errContains string
		expectTools bool
	}{
		{
			name: "execution with no tool registry",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "response without tools"},
					ToolRegistry: nil,
				}
			},
			request:     "test request",
			expectErr:   false,
			expectTools: false,
		},
		{
			name: "execution with empty tool registry",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "response with empty tools"},
					ToolRegistry: &mockEmptyToolRegistry{},
				}
			},
			request:     "test request",
			expectErr:   false,
			expectTools: false,
		},
		{
			name: "execution with populated tool registry",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "response with tools"},
					ToolRegistry: &mockToolRegistry{},
				}
			},
			request:     "test request",
			expectErr:   false,
			expectTools: true,
		},
		{
			name: "execution with tool registry error",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "response with error tools"},
					ToolRegistry: &mockErrorToolRegistry{},
				}
			},
			request:     "test request",
			expectErr:   false, // Tool errors shouldn't fail the whole request
			expectTools: false,
		},
		{
			name: "execution with LLM error",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{shouldError: true, errorMsg: "LLM failed"},
					ToolRegistry: &mockToolRegistry{},
				}
			},
			request:     "test request",
			expectErr:   true,
			errContains: "LLM completion failed with tool context",
		},
		{
			name: "execution with empty request",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "response to empty"},
					ToolRegistry: &mockToolRegistry{},
				}
			},
			request:     "",
			expectErr:   false,
			expectTools: true,
		},
		{
			name: "execution with very long request",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "response to long request"},
					ToolRegistry: &mockToolRegistry{},
				}
			},
			request:     generateLongString(10000),
			expectErr:   false,
			expectTools: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := tt.setupAgent()
			ctx := context.Background()

			response, err := agent.executeWithTools(ctx, tt.request)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Empty(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, response)
			}
		})
	}
}

// Mock tool registry that returns empty tools
type mockEmptyToolRegistry struct{}

func (m *mockEmptyToolRegistry) RegisterTool(name string, tool tools.Tool) error {
	return nil
}

func (m *mockEmptyToolRegistry) GetTool(name string) (tools.Tool, error) {
	return nil, fmt.Errorf("tool not found: %s", name)
}

func (m *mockEmptyToolRegistry) ListTools() []string {
	return []string{}
}

func (m *mockEmptyToolRegistry) HasTool(name string) bool {
	return false
}

func (m *mockEmptyToolRegistry) UnregisterTool(name string) error {
	return nil
}

func (m *mockEmptyToolRegistry) Clear() {
	// No-op
}

// Mock tool registry with tools
type mockToolRegistry struct{}

func (m *mockToolRegistry) RegisterTool(name string, tool tools.Tool) error {
	return nil
}

func (m *mockToolRegistry) GetTool(name string) (tools.Tool, error) {
	return &mockTool{name: name}, nil
}

func (m *mockToolRegistry) ListTools() []string {
	return []string{"file-tool", "web-tool", "calc-tool"}
}

func (m *mockToolRegistry) HasTool(name string) bool {
	return true
}

func (m *mockToolRegistry) UnregisterTool(name string) error {
	return nil
}

func (m *mockToolRegistry) Clear() {
	// No-op
}

// Mock tool registry that returns errors for tools
type mockErrorToolRegistry struct{}

func (m *mockErrorToolRegistry) RegisterTool(name string, tool tools.Tool) error {
	return fmt.Errorf("register error")
}

func (m *mockErrorToolRegistry) GetTool(name string) (tools.Tool, error) {
	return nil, fmt.Errorf("get tool error for %s", name)
}

func (m *mockErrorToolRegistry) ListTools() []string {
	return []string{"error-tool-1", "error-tool-2"}
}

func (m *mockErrorToolRegistry) HasTool(name string) bool {
	return false
}

func (m *mockErrorToolRegistry) UnregisterTool(name string) error {
	return fmt.Errorf("unregister error")
}

func (m *mockErrorToolRegistry) Clear() {
	// No-op
}

// Mock tool implementation
type mockTool struct {
	name string
}

func (t *mockTool) Name() string {
	return t.name
}

func (t *mockTool) Description() string {
	return fmt.Sprintf("Mock tool: %s", t.name)
}

func (t *mockTool) Schema() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *mockTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	return &tools.ToolResult{
		Output:  fmt.Sprintf("result from %s", t.name),
		Success: true,
	}, nil
}

func (t *mockTool) Examples() []string {
	return []string{"example input"}
}

func (t *mockTool) Category() string {
	return "mock"
}

func (t *mockTool) RequiresAuth() bool {
	return false
}

// Test the tool context enhancement specifically
func TestWorkerAgent_executeWithTools_ToolContext(t *testing.T) {
	agent := &WorkerAgent{
		ID:           "test-agent",
		Name:         "Test Agent",
		LLMClient:    &mockCapturingLLMClient{},
		ToolRegistry: &mockToolRegistry{},
	}

	ctx := context.Background()
	request := "test request"

	response, err := agent.executeWithTools(ctx, request)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)

	// Verify that the LLM client received enhanced request with tool context
	capturingClient := agent.LLMClient.(*mockCapturingLLMClient)
	assert.NotEqual(t, request, capturingClient.lastPrompt)
	assert.Contains(t, capturingClient.lastPrompt, request)
	assert.Contains(t, capturingClient.lastPrompt, "Available tools:")
	assert.Contains(t, capturingClient.lastPrompt, "file-tool")
	assert.Contains(t, capturingClient.lastPrompt, "web-tool")
	assert.Contains(t, capturingClient.lastPrompt, "calc-tool")
}

// Mock LLM client that captures the last prompt for verification
type mockCapturingLLMClient struct {
	lastPrompt string
	response   string
}

func (m *mockCapturingLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	m.lastPrompt = prompt
	if m.response == "" {
		return "captured response", nil
	}
	return m.response, nil
}

// Test edge cases for tool context building
func TestWorkerAgent_executeWithTools_ToolContextEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		toolRegistry interface{}
		expectTools  bool
	}{
		{
			name:         "nil tool registry",
			toolRegistry: nil,
			expectTools:  false,
		},
		{
			name:         "empty tool registry",
			toolRegistry: &mockEmptyToolRegistry{},
			expectTools:  false,
		},
		{
			name:         "error tool registry",
			toolRegistry: &mockErrorToolRegistry{},
			expectTools:  false,
		},
		{
			name:         "working tool registry",
			toolRegistry: &mockToolRegistry{},
			expectTools:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &WorkerAgent{
				ID:        "test-agent",
				Name:      "Test Agent",
				LLMClient: &mockCapturingLLMClient{},
			}

			// Set tool registry based on test case
			if tt.toolRegistry != nil {
				if registry, ok := tt.toolRegistry.(tools.Registry); ok {
					agent.ToolRegistry = registry
				}
			}

			ctx := context.Background()
			request := "test request"

			response, err := agent.executeWithTools(ctx, request)
			assert.NoError(t, err)
			assert.NotEmpty(t, response)

			capturingClient := agent.LLMClient.(*mockCapturingLLMClient)
			if tt.expectTools {
				assert.Contains(t, capturingClient.lastPrompt, "Available tools:")
			} else {
				assert.NotContains(t, capturingClient.lastPrompt, "Available tools:")
			}
		})
	}
}
