// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test CostAwareExecute method to improve coverage
func TestWorkerAgent_CostAwareExecute(t *testing.T) {
	tests := []struct {
		name        string
		setupAgent  func() *WorkerAgent
		request     string
		expectErr   bool
		errContains string
	}{
		{
			name: "successful cost-aware execution",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:          "test-agent",
					Name:        "Test Agent",
					LLMClient:   &mockLLMClient{response: "test response"},
					CostManager: newCostManager(),
				}
			},
			request:   "test request",
			expectErr: false,
		},
		{
			name: "execution with empty request",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:          "test-agent",
					Name:        "Test Agent",
					LLMClient:   &mockLLMClient{response: "test response"},
					CostManager: newCostManager(),
				}
			},
			request:   "",
			expectErr: false,
		},
		{
			name: "execution with long request",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:          "test-agent",
					Name:        "Test Agent",
					LLMClient:   &mockLLMClient{response: "large response"},
					CostManager: newCostManager(),
				}
			},
			request:   "complex request requiring more tokens",
			expectErr: false,
		},
		{
			name: "execution with nil cost manager",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:          "test-agent",
					Name:        "Test Agent",
					LLMClient:   &mockLLMClient{response: "test response"},
					CostManager: nil,
				}
			},
			request:     "test request",
			expectErr:   true,
			errContains: "no cost manager configured",
		},
		{
			name: "execution with LLM error",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:          "test-agent",
					Name:        "Test Agent",
					LLMClient:   &mockLLMClient{shouldError: true, errorMsg: "LLM failed"},
					CostManager: newCostManager(),
				}
			},
			request:     "test request",
			expectErr:   true,
			errContains: "LLM failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := tt.setupAgent()
			ctx := context.Background()

			response, err := agent.CostAwareExecute(ctx, tt.request)

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

// Test CostAwareExecute with different cost scenarios
func TestWorkerAgent_CostAwareExecute_CostTracking(t *testing.T) {
	costMgr := newCostManager()
	agent := &WorkerAgent{
		ID:          "test-agent",
		Name:        "Test Agent",
		LLMClient:   &mockLLMClient{response: "test response"},
		CostManager: costMgr,
	}

	ctx := context.Background()

	// Set up initial budget
	agent.SetCostBudget(CostTypeLLM, 1000.0)

	// Execute first request
	response1, err1 := agent.CostAwareExecute(ctx, "first request")
	assert.NoError(t, err1)
	assert.NotEmpty(t, response1)

	// Execute second request
	response2, err2 := agent.CostAwareExecute(ctx, "second request")
	assert.NoError(t, err2)
	assert.NotEmpty(t, response2)

	// Verify cost tracking is working
	report := agent.GetCostReport()
	assert.NotNil(t, report)
	assert.IsType(t, map[string]interface{}{}, report)
}

// Test CostAwareExecute edge cases
func TestWorkerAgent_CostAwareExecute_EdgeCases(t *testing.T) {
	agent := &WorkerAgent{
		ID:          "test-agent",
		Name:        "Test Agent",
		LLMClient:   &mockLLMClient{response: "test response"},
		CostManager: newCostManager(),
	}

	ctx := context.Background()

	// Test with empty request
	response, err := agent.CostAwareExecute(ctx, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, response)

	// Test with very long request
	longRequest := generateLongString(5000)
	response, err = agent.CostAwareExecute(ctx, longRequest)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)

	// Test with short request
	response, err = agent.CostAwareExecute(ctx, "test request")
	assert.NoError(t, err) // Should still work
	assert.NotEmpty(t, response)
}
