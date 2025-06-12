package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/tools"
	"github.com/stretchr/testify/assert"
)

// Test ExecuteWithTools method to improve coverage
func TestWorkerAgent_ExecuteWithTools(t *testing.T) {
	tests := []struct {
		name         string
		setupAgent   func() *WorkerAgent
		request      string
		allowedTools []string
		expectErr    bool
		errContains  string
	}{
		{
			name: "successful execution with tools",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "LLM response"},
					CostManager:  newCostManager(),
					ToolRegistry: &mockWorkingToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{"test-tool"},
			expectErr:    false,
		},
		{
			name: "execution with no tools",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "LLM response"},
					CostManager:  newCostManager(),
					ToolRegistry: &mockWorkingToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{},
			expectErr:    false,
		},
		{
			name: "execution with nil cost manager",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "LLM response"},
					CostManager:  nil,
					ToolRegistry: &mockWorkingToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{"test-tool"},
			expectErr:    true,
			errContains:  "no cost manager configured",
		},
		{
			name: "execution with budget exceeded",
			setupAgent: func() *WorkerAgent {
				costMgr := newCostManager()
				// Set very low budget
				costMgr.SetBudget(CostTypeTool, 0.0001)
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "LLM response"},
					CostManager:  costMgr,
					ToolRegistry: &mockWorkingToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{"tool1", "tool2", "tool3"}, // Multiple tools to exceed budget
			expectErr:    true,
			errContains:  "tool budget exceeded",
		},
		{
			name: "execution with LLM error",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{shouldError: true, errorMsg: "LLM failed"},
					CostManager:  newCostManager(),
					ToolRegistry: &mockWorkingToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{"test-tool"},
			expectErr:    true,
			errContains:  "LLM failed",
		},
		{
			name: "execution with tool not found",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "LLM response"},
					CostManager:  newCostManager(),
					ToolRegistry: &mockEmptyToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{"non-existent-tool"},
			expectErr:    true,
			errContains:  "tool not found",
		},
		{
			name: "execution with tool execution error",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "LLM response"},
					CostManager:  newCostManager(),
					ToolRegistry: &mockFailingToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{"failing-tool"},
			expectErr:    true,
			errContains:  "tool execution failed",
		},
		{
			name: "execution with cost tracking error",
			setupAgent: func() *WorkerAgent {
				costMgr := &mockErrorCostManager{}
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "LLM response"},
					CostManager:  costMgr,
					ToolRegistry: &mockWorkingToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{"test-tool"},
			expectErr:    true,
			errContains:  "tool budget exceeded",
		},
		{
			name: "execution with multiple tools",
			setupAgent: func() *WorkerAgent {
				costMgr := newCostManager()
				costMgr.SetBudget(CostTypeTool, 10.0) // High budget
				return &WorkerAgent{
					ID:           "test-agent",
					Name:         "Test Agent",
					LLMClient:    &mockLLMClient{response: "LLM response"},
					CostManager:  costMgr,
					ToolRegistry: &mockWorkingToolRegistry{},
				}
			},
			request:      "test request",
			allowedTools: []string{"tool1", "tool2", "tool3"},
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := tt.setupAgent()
			ctx := context.Background()

			response, err := agent.ExecuteWithTools(ctx, tt.request, tt.allowedTools)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Empty(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, response)

				// If tools were allowed, response should include tool result
				if len(tt.allowedTools) > 0 {
					assert.Contains(t, response, "Tool Result")
				}
			}
		})
	}
}

// Mock working tool registry for ExecuteWithTools tests
type mockWorkingToolRegistry struct{}

func (m *mockWorkingToolRegistry) RegisterTool(name string, tool tools.Tool) error {
	return nil
}

func (m *mockWorkingToolRegistry) GetTool(name string) (tools.Tool, error) {
	return &mockWorkingTool{name: name}, nil
}

func (m *mockWorkingToolRegistry) ListTools() []string {
	return []string{"test-tool", "tool1", "tool2", "tool3"}
}

func (m *mockWorkingToolRegistry) HasTool(name string) bool {
	return true
}

func (m *mockWorkingToolRegistry) UnregisterTool(name string) error {
	return nil
}

func (m *mockWorkingToolRegistry) Clear() {
	// No-op
}

// Mock failing tool registry
type mockFailingToolRegistry struct{}

func (m *mockFailingToolRegistry) RegisterTool(name string, tool tools.Tool) error {
	return nil
}

func (m *mockFailingToolRegistry) GetTool(name string) (tools.Tool, error) {
	return &mockFailingTool{name: name}, nil
}

func (m *mockFailingToolRegistry) ListTools() []string {
	return []string{"failing-tool"}
}

func (m *mockFailingToolRegistry) HasTool(name string) bool {
	return true
}

func (m *mockFailingToolRegistry) UnregisterTool(name string) error {
	return nil
}

func (m *mockFailingToolRegistry) Clear() {
	// No-op
}

// Mock working tool
type mockWorkingTool struct {
	name string
}

func (t *mockWorkingTool) Name() string {
	return t.name
}

func (t *mockWorkingTool) Description() string {
	return fmt.Sprintf("Working tool: %s", t.name)
}

func (t *mockWorkingTool) Schema() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *mockWorkingTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	return &tools.ToolResult{Output: fmt.Sprintf("result from %s", t.name), Success: true}, nil
}

func (t *mockWorkingTool) Examples() []string {
	return []string{"example input"}
}

func (t *mockWorkingTool) Category() string {
	return "working"
}

func (t *mockWorkingTool) RequiresAuth() bool {
	return false
}

// Mock failing tool
type mockFailingTool struct {
	name string
}

func (t *mockFailingTool) Name() string {
	return t.name
}

func (t *mockFailingTool) Description() string {
	return fmt.Sprintf("Failing tool: %s", t.name)
}

func (t *mockFailingTool) Schema() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *mockFailingTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	return nil, fmt.Errorf("tool execution failed for %s", t.name)
}

func (t *mockFailingTool) Examples() []string {
	return []string{"example input"}
}

func (t *mockFailingTool) Category() string {
	return "failing"
}

func (t *mockFailingTool) RequiresAuth() bool {
	return false
}

// Mock error cost manager
type mockErrorCostManager struct{}

func (m *mockErrorCostManager) TrackCost(costType CostType, amount float64) error {
	return fmt.Errorf("cost tracking failed")
}

func (m *mockErrorCostManager) GetCostReport() map[string]interface{} {
	return map[string]interface{}{}
}

func (m *mockErrorCostManager) SetBudget(costType CostType, amount float64) {
	// No-op
}

func (m *mockErrorCostManager) GetBudget(costType CostType) float64 {
	return 1000.0
}

func (m *mockErrorCostManager) CanAfford(costType CostType, amount float64) bool {
	return false // Always return false to trigger budget exceeded
}

func (m *mockErrorCostManager) GetBudgetRemaining(costType CostType) float64 {
	return 0.0
}

func (m *mockErrorCostManager) GetTotalCost() float64 {
	return 0.0
}

func (m *mockErrorCostManager) Reset() {
	// No-op
}

func (m *mockErrorCostManager) ExceedsBudget(costType CostType, amount float64) bool {
	return true
}

func (m *mockErrorCostManager) EstimateLLMCost(model string, estimatedTokens int) float64 {
	return 0.0
}

func (m *mockErrorCostManager) RecordLLMCost(model string, promptTokens, completionTokens int, metadata map[string]string) error {
	return fmt.Errorf("failed to record LLM cost")
}

// Test edge cases for ExecuteWithTools
func TestWorkerAgent_ExecuteWithTools_EdgeCases(t *testing.T) {
	agent := &WorkerAgent{
		ID:           "test-agent",
		Name:         "Test Agent",
		LLMClient:    &mockLLMClient{response: "LLM response"},
		CostManager:  newCostManager(),
		ToolRegistry: &mockWorkingToolRegistry{},
	}

	ctx := context.Background()

	// Test with empty request
	response, err := agent.ExecuteWithTools(ctx, "", []string{"test-tool"})
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Contains(t, response, "Tool Result")

	// Test with very long request
	longRequest := generateLongString(5000)
	response, err = agent.ExecuteWithTools(ctx, longRequest, []string{"test-tool"})
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Contains(t, response, "Tool Result")

	// Test with many tools (but only first one should execute)
	manyTools := []string{"tool1", "tool2", "tool3", "tool4", "tool5"}
	response, err = agent.ExecuteWithTools(ctx, "test request", manyTools)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
	assert.Contains(t, response, "Tool Result")
	// Should only contain result from first tool (tool1)
	assert.Contains(t, response, "tool1")
	assert.NotContains(t, response, "tool2") // Other tools shouldn't execute
}

// Test cost tracking behavior
func TestWorkerAgent_ExecuteWithTools_CostTracking(t *testing.T) {
	costMgr := newCostManager()
	costMgr.SetBudget(CostTypeTool, 10.0)

	agent := &WorkerAgent{
		ID:           "test-agent",
		Name:         "Test Agent",
		LLMClient:    &mockLLMClient{response: "LLM response"},
		CostManager:  costMgr,
		ToolRegistry: &mockWorkingToolRegistry{},
	}

	ctx := context.Background()

	// Execute with tools
	response, err := agent.ExecuteWithTools(ctx, "test request", []string{"test-tool"})
	assert.NoError(t, err)
	assert.NotEmpty(t, response)

	// Verify cost tracking worked
	report := costMgr.GetCostReport()
	assert.NotNil(t, report)

	// Should have tracked some tool cost
	totalCosts, ok := report["total_costs"].(map[string]float64)
	assert.True(t, ok)
	assert.Greater(t, totalCosts[string(CostTypeTool)], 0.0)
}
