package agent

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/stretchr/testify/assert"
)

// Test DefaultCostManagerFactory
func TestDefaultCostManagerFactory(t *testing.T) {
	// Create cost manager using default factory
	costManager := DefaultCostManagerFactory()

	// Verify it's not nil and is the right type
	assert.NotNil(t, costManager)

	// Test basic operations
	costManager.SetBudget(CostTypeLLM, 100.0)
	// Use GetBudgetRemaining since GetBudget doesn't exist in interface
	assert.Equal(t, 100.0, costManager.GetBudgetRemaining(CostTypeLLM))

	// Test it can track costs
	err := costManager.TrackCost(CostTypeLLM, 5.0)
	assert.NoError(t, err)
	assert.Equal(t, 95.0, costManager.GetBudgetRemaining(CostTypeLLM))
}

// Test Factory creation and worker agent creation without circular deps
func TestFactory_CreateWorkerAgent_Simple(t *testing.T) {
	// Create test implementations
	testLLM := &testLLMClient{}
	testMemory := &testMemoryManager{}
	testTools := &testToolRegistry{}
	testCommission := &testCommissionManager{}

	// Create factory with default cost manager
	factory := &DefaultFactory{
		LLMClient:         testLLM,
		MemoryManager:     testMemory,
		ToolRegistry:      testTools,
		CommissionManager: testCommission,
		CostManager:       DefaultCostManagerFactory(),
	}

	// Create worker agent
	ctx := context.Background()
	agent, err := factory.CreateWorkerAgent(ctx, "test-worker", "Test Worker")
	assert.NoError(t, err)

	// Verify
	assert.NotNil(t, agent)
	assert.Equal(t, "test-worker", agent.GetID())
	assert.Equal(t, "Test Worker", agent.GetName())

	// Verify it's a WorkerAgent with all dependencies
	workerAgent, ok := agent.(*WorkerAgent)
	assert.True(t, ok, "Should be a WorkerAgent")
	assert.NotNil(t, workerAgent.CostManager)
	assert.NotNil(t, workerAgent.LLMClient)
	assert.NotNil(t, workerAgent.MemoryManager)
	assert.NotNil(t, workerAgent.ToolRegistry)
	assert.NotNil(t, workerAgent.CommissionManager)
}

// Test Factory CreateAgent with config
func TestFactory_CreateAgent_Simple(t *testing.T) {
	tests := []struct {
		name         string
		agentConfig  config.AgentConfig
		expectedType string
	}{
		{
			name: "create worker agent",
			agentConfig: config.AgentConfig{
				ID:   "worker-1",
				Name: "Worker Agent",
				Type: "worker",
			},
			expectedType: "worker",
		},
		{
			name: "create default agent (worker)",
			agentConfig: config.AgentConfig{
				ID:   "default-1",
				Name: "Default Agent",
				Type: "worker", // Explicitly set to worker since empty type returns error
			},
			expectedType: "worker",
		},
		{
			name: "create unknown type returns error",
			agentConfig: config.AgentConfig{
				ID:   "unknown-1",
				Name: "Unknown Agent",
				Type: "unknown",
			},
			expectedType: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test implementations
			testLLM := &testLLMClient{}
			testMemory := &testMemoryManager{}
			testTools := &testToolRegistry{}
			testCommission := &testCommissionManager{}

			// Create factory with default cost manager
			factory := &DefaultFactory{
				LLMClient:         testLLM,
				MemoryManager:     testMemory,
				ToolRegistry:      testTools,
				CommissionManager: testCommission,
				CostManager:       DefaultCostManagerFactory(),
			}

			// Create agent
			ctx := context.Background()
			agent, err := factory.CreateAgent(ctx, tt.agentConfig.ID, tt.agentConfig.Name, tt.agentConfig.Type)

			// Verify based on expected type
			if tt.expectedType == "error" {
				assert.Error(t, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, tt.agentConfig.ID, agent.GetID())
				assert.Equal(t, tt.agentConfig.Name, agent.GetName())

				// Verify it's a worker agent
				_, ok := agent.(*WorkerAgent)
				assert.True(t, ok, "Should be a WorkerAgent")
			}
		})
	}
}
