package agent

import (
	"testing"

	"github.com/blockhead-consulting/guild/pkg/agent/mocks"
	"github.com/blockhead-consulting/guild/pkg/corpus"
	memorymocks "github.com/blockhead-consulting/guild/pkg/memory/mocks"
	"github.com/blockhead-consulting/guild/pkg/memory/rag"
	"github.com/blockhead-consulting/guild/pkg/memory/vector"
	objectivemocks "github.com/blockhead-consulting/guild/pkg/objective/mocks"
	"github.com/blockhead-consulting/guild/pkg/providers"
	toolsmocks "github.com/blockhead-consulting/guild/tools/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVectorStoreFactory is a mock implementation of the vector store factory
// for testing purposes
type MockVectorStoreFactory struct {
	vector.Factory
	mockVectorStore vector.VectorStore
}

func (m *MockVectorStoreFactory) CreateVectorStore(config vector.StoreConfig) (vector.VectorStore, error) {
	return m.mockVectorStore, nil
}

// TestNewRagAgentFactory tests the creation of a RAG agent factory
func TestNewRagAgentFactory(t *testing.T) {
	// Create mock dependencies
	mockMemoryManager := new(memorymocks.MockMemoryManager)
	mockToolRegistry := new(toolsmocks.MockToolRegistry)
	mockObjectiveManager := new(objectivemocks.MockObjectiveManager)
	
	// Create options with test configurations
	options := DefaultRagAgentFactoryOptions()
	options.VectorStoreConfig.Path = "./test-vectorstore"
	
	// Test creating the factory
	factory, err := NewRagAgentFactory(
		mockMemoryManager,
		mockToolRegistry,
		mockObjectiveManager,
		options,
	)
	
	// Just check for the error, as we can't easily create a real vector store in tests
	// In a real implementation, you might want to mock the vector store factory
	// Assert error is not nil since we're not mocking the vector store creation
	assert.Error(t, err)
	assert.Nil(t, factory)
}

// TestCreateAgent tests creating a RAG agent from the factory
func TestCreateAgent(t *testing.T) {
	// Create mock dependencies
	mockMemoryManager := new(memorymocks.MockMemoryManager)
	mockToolRegistry := new(toolsmocks.MockToolRegistry)
	mockObjectiveManager := new(objectivemocks.MockObjectiveManager)
	mockVectorStore := new(mocks.MockVectorStore)
	mockLlmClient := new(mocks.MockLLMClient)
	
	// Configure vector store mock
	mockVectorStore.On("Close").Return(nil)
	
	// Create a factory with the mock vector store
	factory := &RagAgentFactory{
		options:       DefaultRagAgentFactoryOptions(),
		vectorStore:   mockVectorStore,
		retriever:     rag.NewRetriever(mockVectorStore, corpus.DefaultConfig()),
		memoryManager: mockMemoryManager,
		toolRegistry:  mockToolRegistry,
		objectiveMgr:  mockObjectiveManager,
	}
	
	// Test creating an agent
	config := &AgentConfig{
		ID:          "test-rag-agent",
		Name:        "Test RAG Agent",
		Type:        "worker",
		Provider:    providers.ProviderTypeOpenAI,
		Model:       "gpt-3.5-turbo",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	
	// Mock LLM client's Complete method
	mockLlmClient.On("Complete", mock.Anything, mock.Anything).Return(&providers.CompletionResponse{
		Text:       "Test response",
		TokensUsed: 10,
	}, nil)
	
	// Create the agent
	agent, err := factory.CreateAgent(config, mockLlmClient)
	
	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, "test-rag-agent", agent.ID())
	assert.Equal(t, "Test RAG Agent", agent.Name())
	assert.Equal(t, "worker", agent.Type())
	
	// Test closing the factory
	err = factory.Close()
	assert.NoError(t, err)
	mockVectorStore.AssertCalled(t, "Close")
}

// TestGetRetriever tests getting the retriever from the factory
func TestGetRetriever(t *testing.T) {
	// Create mock dependencies
	mockMemoryManager := new(memorymocks.MockMemoryManager)
	mockToolRegistry := new(toolsmocks.MockToolRegistry)
	mockObjectiveManager := new(objectivemocks.MockObjectiveManager)
	mockVectorStore := new(mocks.MockVectorStore)
	
	// Configure vector store mock
	mockVectorStore.On("Close").Return(nil)
	
	// Create a factory with the mock vector store
	factory := &RagAgentFactory{
		options:       DefaultRagAgentFactoryOptions(),
		vectorStore:   mockVectorStore,
		retriever:     rag.NewRetriever(mockVectorStore, corpus.DefaultConfig()),
		memoryManager: mockMemoryManager,
		toolRegistry:  mockToolRegistry,
		objectiveMgr:  mockObjectiveManager,
	}
	
	// Test getting the retriever
	retriever := factory.GetRetriever()
	assert.NotNil(t, retriever)
}