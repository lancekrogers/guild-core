package rag

import (
	"testing"

	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/agent/mocks"
	"github.com/blockhead-consulting/guild/pkg/corpus"
	memorymocks "github.com/blockhead-consulting/guild/pkg/memory/mocks"
	"github.com/blockhead-consulting/guild/pkg/memory/vector"
	objectivemocks "github.com/blockhead-consulting/guild/pkg/objective/mocks"
	"github.com/blockhead-consulting/guild/pkg/providers"
	toolsmocks "github.com/blockhead-consulting/guild/tools/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestFactory tests the factory methods
func TestFactory(t *testing.T) {
	// Create mock dependencies
	mockMemoryManager := new(memorymocks.MockMemoryManager)
	mockToolRegistry := new(toolsmocks.MockToolRegistry)
	mockObjectiveManager := new(objectivemocks.MockObjectiveManager)
	mockVectorStore := new(mocks.MockVectorStore)
	mockAgent := new(mocks.MockAgent)

	// Configure mocks
	mockVectorStore.On("Close").Return(nil)
	
	mockAgent.On("ID").Return("test-agent")
	mockAgent.On("Name").Return("Test Agent")
	mockAgent.On("Type").Return("worker")
	mockAgent.On("Status").Return(agent.StatusIdle)
	mockAgent.On("GetMemoryManager").Return(mockMemoryManager)
	
	// Create factory options
	options := DefaultFactoryOptions()
	
	// Test creating a factory
	factory, err := NewFactory(
		mockMemoryManager,
		mockToolRegistry,
		mockObjectiveManager,
		options,
		mockVectorStore,
	)
	
	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, factory)
	
	// Test enhancing an agent
	enhancedAgent := factory.EnhanceAgent(mockAgent)
	assert.NotNil(t, enhancedAgent)
	assert.Equal(t, "test-agent", enhancedAgent.ID())
	
	// Test getting retriever
	retriever := factory.GetRetriever()
	assert.NotNil(t, retriever)
	
	// Test closing
	err = factory.Close()
	assert.NoError(t, err)
	mockVectorStore.AssertCalled(t, "Close")
}

// MockAgentFactory is a mock implementation of agent.Factory
type MockAgentFactory struct {
	mock.Mock
}

// CreateAgent implements the agent.Factory interface
func (m *MockAgentFactory) CreateAgent(config *agent.AgentConfig, llmClient providers.LLMClient) (agent.GuildArtisan, error) {
	args := m.Called(config, llmClient)
	return args.Get(0).(agent.GuildArtisan), args.Error(1)
}

// TestCreateRagAgent tests the agent creation function
func TestCreateRagAgent(t *testing.T) {
	// Create mock dependencies
	mockMemoryManager := new(memorymocks.MockMemoryManager)
	mockToolRegistry := new(toolsmocks.MockToolRegistry)
	mockObjectiveManager := new(objectivemocks.MockObjectiveManager)
	mockVectorStore := new(mocks.MockVectorStore)
	mockLlmClient := new(mocks.MockLLMClient)
	mockAgentFactory := new(MockAgentFactory)
	mockAgent := new(mocks.MockAgent)
	
	// Configure mocks
	mockVectorStore.On("Close").Return(nil)
	
	mockAgent.On("ID").Return("test-agent")
	mockAgent.On("Name").Return("Test Agent")
	mockAgent.On("Type").Return("worker")
	mockAgent.On("Status").Return(agent.StatusIdle)
	mockAgent.On("GetMemoryManager").Return(mockMemoryManager)
	
	// Create config and factory
	config := &agent.AgentConfig{
		ID:   "test-agent",
		Name: "Test Agent",
		Type: "worker",
	}
	
	mockAgentFactory.On("CreateAgent", config, mockLlmClient).Return(mockAgent, nil)
	
	options := DefaultFactoryOptions()
	factory, err := NewFactory(
		mockMemoryManager,
		mockToolRegistry,
		mockObjectiveManager,
		options,
		mockVectorStore,
	)
	assert.NoError(t, err)
	
	// Test creating a RAG agent
	ragAgent, err := factory.CreateRagAgent(mockAgentFactory, config, mockLlmClient)
	
	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, ragAgent)
	assert.Equal(t, "test-agent", ragAgent.ID())
	mockAgentFactory.AssertCalled(t, "CreateAgent", config, mockLlmClient)
}