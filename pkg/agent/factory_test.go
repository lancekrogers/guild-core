package agent

import (
	"fmt"
	"testing"

	// "github.com/guild-ventures/guild-core/pkg/agent/manager" // Commented to avoid import cycle
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock components for factory testing
type mockPromptManager struct{}

func (m *mockPromptManager) GetPrompt(name string) (string, error) {
	return "mock prompt", nil
}

func (m *mockPromptManager) RegisterPrompt(name, prompt string) error {
	return nil
}

func (m *mockPromptManager) ListPrompts() []string {
	return []string{"prompt1", "prompt2"}
}

// mockToolRegistry is defined in execute_with_tools_test.go

// Clear is defined in execute_with_tools_test.go

type mockMemoryManager struct{}

func (m *mockMemoryManager) Store(key string, value interface{}) error {
	return nil
}

func (m *mockMemoryManager) Retrieve(key string) (interface{}, error) {
	return "mock value", nil
}

func (m *mockMemoryManager) Delete(key string) error {
	return nil
}

func (m *mockMemoryManager) List() ([]string, error) {
	return []string{"key1", "key2"}, nil
}

func (m *mockMemoryManager) Clear() error {
	return nil
}

// Test newFactory
func TestNewFactory(t *testing.T) {
	tests := []struct {
		name          string
		llmClient     LLMClient
		promptManager PromptManager
		toolRegistry  ToolRegistry
		memoryManager MemoryManager
		expectPanic   bool
		validateFactory func(t *testing.T, f *factory)
	}{
		{
			name:          "successful creation with all components",
			llmClient:     &mockLLMClient{response: "test"},
			promptManager: &mockPromptManager{},
			toolRegistry:  &mockToolRegistry{tools: map[string]bool{"tool1": true}},
			memoryManager: &mockMemoryManager{},
			expectPanic:   false,
			validateFactory: func(t *testing.T, f *factory) {
				assert.NotNil(t, f)
				assert.NotNil(t, f.llmClient)
				assert.NotNil(t, f.promptManager)
				assert.NotNil(t, f.toolRegistry)
				assert.NotNil(t, f.memoryManager)
			},
		},
		{
			name:          "creation with nil LLM client",
			llmClient:     nil,
			promptManager: &mockPromptManager{},
			toolRegistry:  &mockToolRegistry{},
			memoryManager: &mockMemoryManager{},
			expectPanic:   false,
			validateFactory: func(t *testing.T, f *factory) {
				assert.NotNil(t, f)
				assert.Nil(t, f.llmClient)
			},
		},
		{
			name:          "creation with minimal components",
			llmClient:     &mockLLMClient{response: "test"},
			promptManager: nil,
			toolRegistry:  nil,
			memoryManager: nil,
			expectPanic:   false,
			validateFactory: func(t *testing.T, f *factory) {
				assert.NotNil(t, f)
				assert.NotNil(t, f.llmClient)
				assert.Nil(t, f.promptManager)
				assert.Nil(t, f.toolRegistry)
				assert.Nil(t, f.memoryManager)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.Panics(t, func() {
					newFactory(tt.llmClient, tt.promptManager, tt.toolRegistry, tt.memoryManager)
				})
			} else {
				f := newFactory(tt.llmClient, tt.promptManager, tt.toolRegistry, tt.memoryManager)
				tt.validateFactory(t, f)
			}
		})
	}
}

// Test CreateAgent
func TestFactory_CreateAgent(t *testing.T) {
	tests := []struct {
		name         string
		setupFactory func() *factory
		agentConfig  config.AgentConfig
		expectError  bool
		validateAgent func(t *testing.T, agent Agent)
	}{
		{
			name: "create worker agent",
			setupFactory: func() *factory {
				return newFactory(
					&mockLLMClient{response: "test"},
					&mockPromptManager{},
					&mockToolRegistry{tools: map[string]bool{"tool1": true}},
					&mockMemoryManager{},
				)
			},
			agentConfig: config.AgentConfig{
				ID:           "worker-1",
				Name:         "Worker Agent 1",
				Type:         "worker",
				Capabilities: []string{"general", "code"},
			},
			expectError: false,
			validateAgent: func(t *testing.T, agent Agent) {
				assert.NotNil(t, agent)
				assert.Equal(t, "worker-1", agent.GetID())
				assert.Equal(t, "Worker Agent 1", agent.GetName())
				workerAgent, ok := agent.(*WorkerAgent)
				require.True(t, ok)
				assert.NotNil(t, workerAgent.LLMClient)
			},
		},
		{
			name: "create agent with cost manager",
			setupFactory: func() *factory {
				f := newFactory(
					&mockLLMClient{response: "test"},
					&mockPromptManager{},
					&mockToolRegistry{},
					&mockMemoryManager{},
				)
				f.costManager = newCostManager()
				return f
			},
			agentConfig: config.AgentConfig{
				ID:   "cost-aware-agent",
				Name: "Cost Aware Agent",
				Type: "worker",
			},
			expectError: false,
			validateAgent: func(t *testing.T, agent Agent) {
				workerAgent, ok := agent.(*WorkerAgent)
				require.True(t, ok)
				assert.NotNil(t, workerAgent.CostManager)
			},
		},
		{
			name: "create agent without LLM client",
			setupFactory: func() *factory {
				return newFactory(nil, nil, nil, nil)
			},
			agentConfig: config.AgentConfig{
				ID:   "no-llm-agent",
				Name: "No LLM Agent",
				Type: "worker",
			},
			expectError: false,
			validateAgent: func(t *testing.T, agent Agent) {
				workerAgent, ok := agent.(*WorkerAgent)
				require.True(t, ok)
				assert.Nil(t, workerAgent.LLMClient)
			},
		},
		{
			name: "create agent with all components",
			setupFactory: func() *factory {
				f := newFactory(
					&mockLLMClient{response: "test"},
					&mockPromptManager{},
					&mockToolRegistry{tools: map[string]bool{"tool1": true, "tool2": true}},
					&mockMemoryManager{},
				)
				f.costManager = newCostManager()
				return f
			},
			agentConfig: config.AgentConfig{
				ID:           "full-agent",
				Name:         "Full Agent",
				Type:         "worker",
				Capabilities: []string{"analysis", "code", "documentation"},
			},
			expectError: false,
			validateAgent: func(t *testing.T, agent Agent) {
				workerAgent, ok := agent.(*WorkerAgent)
				require.True(t, ok)
				assert.NotNil(t, workerAgent.LLMClient)
				assert.NotNil(t, workerAgent.PromptManager)
				assert.NotNil(t, workerAgent.ToolRegistry)
				assert.NotNil(t, workerAgent.MemoryManager)
				assert.NotNil(t, workerAgent.CostManager)
				assert.Equal(t, 3, len(workerAgent.Capabilities))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.setupFactory()
			
			agent, err := f.CreateAgent(tt.agentConfig)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.validateAgent(t, agent)
			}
		})
	}
}

// Test CreateManagerAgent
// TODO: Fix import cycle - this test imports manager package which imports agent package
/*
func TestFactory_CreateManagerAgent(t *testing.T) {
	tests := []struct {
		name         string
		setupFactory func() *factory
		intelligence manager.IntelligenceService
		expectError  bool
		validateAgent func(t *testing.T, agent Agent)
	}{
		{
			name: "create manager with intelligence service",
			setupFactory: func() *factory {
				return newFactory(
					&mockLLMClient{response: "manager response"},
					&mockPromptManager{},
					nil,
					nil,
				)
			},
			intelligence: &mockIntelligenceService{},
			expectError:  false,
			validateAgent: func(t *testing.T, agent Agent) {
				assert.NotNil(t, agent)
				managerAgent, ok := agent.(*ManagerAgent)
				require.True(t, ok)
				assert.Equal(t, ManagerAgentID, managerAgent.ID)
				assert.NotNil(t, managerAgent.intelligenceService)
			},
		},
		{
			name: "create manager without intelligence service",
			setupFactory: func() *factory {
				return newFactory(
					&mockLLMClient{response: "manager response"},
					nil,
					nil,
					nil,
				)
			},
			intelligence: nil,
			expectError:  false,
			validateAgent: func(t *testing.T, agent Agent) {
				managerAgent, ok := agent.(*ManagerAgent)
				require.True(t, ok)
				assert.Nil(t, managerAgent.intelligenceService)
			},
		},
		{
			name: "create manager without LLM client",
			setupFactory: func() *factory {
				return newFactory(nil, nil, nil, nil)
			},
			intelligence: &mockIntelligenceService{},
			expectError:  false,
			validateAgent: func(t *testing.T, agent Agent) {
				managerAgent, ok := agent.(*ManagerAgent)
				require.True(t, ok)
				assert.Nil(t, managerAgent.llmClient)
				assert.NotNil(t, managerAgent.intelligenceService)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.setupFactory()
			
			agent, err := f.CreateManagerAgent(tt.intelligence)
			
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.validateAgent(t, agent)
			}
		})
	}
}
*/

// Test DefaultFactoryFactory
func TestDefaultFactoryFactory(t *testing.T) {
	tests := []struct {
		name          string
		llmClient     LLMClient
		promptManager PromptManager
		toolRegistry  ToolRegistry
		memoryManager MemoryManager
		expectNil     bool
	}{
		{
			name:          "create with all components",
			llmClient:     &mockLLMClient{response: "test"},
			promptManager: &mockPromptManager{},
			toolRegistry:  &mockToolRegistry{},
			memoryManager: &mockMemoryManager{},
			expectNil:     false,
		},
		{
			name:          "create with minimal components",
			llmClient:     &mockLLMClient{response: "test"},
			promptManager: nil,
			toolRegistry:  nil,
			memoryManager: nil,
			expectNil:     false,
		},
		{
			name:          "create with nil LLM client",
			llmClient:     nil,
			promptManager: &mockPromptManager{},
			toolRegistry:  nil,
			memoryManager: nil,
			expectNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factoryFunc := DefaultFactoryFactory(tt.llmClient, tt.promptManager, tt.toolRegistry, tt.memoryManager)
			
			if tt.expectNil {
				assert.Nil(t, factoryFunc)
			} else {
				assert.NotNil(t, factoryFunc)
				
				// Test that the function returns a valid AgentFactory
				agentFactory := factoryFunc()
				assert.NotNil(t, agentFactory)
				
				// Verify it can create agents
				agent, err := agentFactory.CreateAgent(config.AgentConfig{
					ID:   "test",
					Name: "Test",
					Type: "worker",
				})
				assert.NoError(t, err)
				assert.NotNil(t, agent)
			}
		})
	}
}

// Test factory with cost manager
func TestFactory_WithCostManager(t *testing.T) {
	f := newFactory(
		&mockLLMClient{response: "test"},
		&mockPromptManager{},
		&mockToolRegistry{},
		&mockMemoryManager{},
	)
	
	// Initially no cost manager
	agent1, err := f.CreateAgent(config.AgentConfig{
		ID:   "agent1",
		Name: "Agent 1",
		Type: "worker",
	})
	assert.NoError(t, err)
	workerAgent1 := agent1.(*WorkerAgent)
	assert.Nil(t, workerAgent1.CostManager)
	
	// Add cost manager
	f.costManager = newCostManager()
	
	// New agent should have cost manager
	agent2, err := f.CreateAgent(config.AgentConfig{
		ID:   "agent2",
		Name: "Agent 2",
		Type: "worker",
	})
	assert.NoError(t, err)
	workerAgent2 := agent2.(*WorkerAgent)
	assert.NotNil(t, workerAgent2.CostManager)
}

// Test concurrent agent creation
func TestFactory_ConcurrentCreation(t *testing.T) {
	f := newFactory(
		&mockLLMClient{response: "test"},
		&mockPromptManager{},
		&mockToolRegistry{tools: map[string]bool{"tool1": true}},
		&mockMemoryManager{},
	)
	f.costManager = newCostManager()
	
	// Create multiple agents concurrently
	numAgents := 10
	agents := make(chan Agent, numAgents)
	errors := make(chan error, numAgents)
	
	for i := 0; i < numAgents; i++ {
		go func(id int) {
			config := config.AgentConfig{
				ID:           fmt.Sprintf("concurrent-%d", id),
				Name:         fmt.Sprintf("Concurrent Agent %d", id),
				Type:         "worker",
				Capabilities: []string{"general"},
			}
			
			agent, err := f.CreateAgent(config)
			if err != nil {
				errors <- err
			} else {
				agents <- agent
			}
		}(i)
	}
	
	// Collect results
	createdAgents := make([]Agent, 0, numAgents)
	for i := 0; i < numAgents; i++ {
		select {
		case err := <-errors:
			t.Errorf("Failed to create agent: %v", err)
		case agent := <-agents:
			createdAgents = append(createdAgents, agent)
		}
	}
	
	assert.Equal(t, numAgents, len(createdAgents))
	
	// Verify all agents are unique and properly configured
	seenIDs := make(map[string]bool)
	for _, agent := range createdAgents {
		id := agent.GetID()
		assert.False(t, seenIDs[id], "Duplicate agent ID found: %s", id)
		seenIDs[id] = true
		
		// Verify agent has all components
		workerAgent := agent.(*WorkerAgent)
		assert.NotNil(t, workerAgent.LLMClient)
		assert.NotNil(t, workerAgent.PromptManager)
		assert.NotNil(t, workerAgent.ToolRegistry)
		assert.NotNil(t, workerAgent.MemoryManager)
		assert.NotNil(t, workerAgent.CostManager)
	}
}