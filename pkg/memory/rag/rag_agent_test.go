package rag

import (
	"context"
	"testing"
	"time"

	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/agent/mocks"
	"github.com/blockhead-consulting/guild/pkg/corpus"
	"github.com/blockhead-consulting/guild/pkg/kanban"
	"github.com/blockhead-consulting/guild/pkg/memory"
	memorymocks "github.com/blockhead-consulting/guild/pkg/memory/mocks"
	"github.com/blockhead-consulting/guild/pkg/memory/vector"
	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// createMockChain creates a mock memory chain for testing
func createMockChain(id, agentID, taskID string) *memory.PromptChain {
	return &memory.PromptChain{
		ID:        id,
		AgentID:   agentID,
		TaskID:    taskID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Messages: []memory.Message{
			{
				Role:      "system",
				Content:   "You are an AI assistant. Your task is:\n\nTitle: Test Task\nDescription: This is a test task for RAG enhancement",
				Timestamp: time.Now().UTC(),
			},
		},
	}
}

// createMockAgent creates a mock agent for testing
func createMockAgent(t *testing.T) (*mocks.MockAgent, *memorymocks.MockMemoryManager) {
	mockAgent := new(mocks.MockAgent)
	mockMemoryManager := new(memorymocks.MockMemoryManager)

	// Set up agent config
	config := &agent.AgentConfig{
		ID:          "test-agent",
		Name:        "Test Agent",
		Type:        "worker",
		Provider:    providers.ProviderTypeOpenAI,
		Model:       "gpt-3.5-turbo",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	// Set up agent state
	state := &agent.AgentState{
		Status:      agent.StatusIdle,
		CurrentTask: "test-task",
		UpdatedAt:   time.Now().UTC(),
		Memory:      []string{"chain-1"},
	}

	// Configure mock methods
	mockAgent.On("ID").Return("test-agent")
	mockAgent.On("Name").Return("Test Agent")
	mockAgent.On("Type").Return("worker")
	mockAgent.On("Status").Return(agent.StatusIdle)
	mockAgent.On("GetConfig").Return(config)
	mockAgent.On("GetState").Return(state)
	mockAgent.On("GetMemoryManager").Return(mockMemoryManager)
	mockAgent.On("CraftSolution", mock.Anything).Return(nil)
	mockAgent.On("SaveState", mock.Anything).Return(nil)

	return mockAgent, mockMemoryManager
}

// createMockRetriever creates a mock retriever for testing
func createMockRetriever() *Retriever {
	// Mock vector store just for testing
	mockVectorStore := &mockVectorStore{}
	corpusConfig := corpus.DefaultConfig()
	return NewRetriever(mockVectorStore, corpusConfig)
}

// mockVectorStore is a simple mock implementation of vector.VectorStore
type mockVectorStore struct{}

func (m *mockVectorStore) SaveEmbedding(ctx context.Context, embedding vector.Embedding) error {
	return nil
}

func (m *mockVectorStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]vector.EmbeddingMatch, error) {
	return []vector.EmbeddingMatch{
		{
			ID:       "test-doc-1",
			Text:     "This is a test document that matches the query",
			Source:   "Test Source",
			Score:    0.9,
			Metadata: map[string]interface{}{"title": "Test Document"},
		},
	}, nil
}

func (m *mockVectorStore) Close() error {
	return nil
}

func TestAgentWrapper_CraftSolution(t *testing.T) {
	mockAgent, mockMemoryManager := createMockAgent(t)
	retriever := createMockRetriever()
	config := DefaultRetrievalConfig()

	wrapper := NewAgentWrapper(mockAgent, retriever, config)

	// Mock the memory chain retrieval
	testChain := createMockChain("chain-1", "test-agent", "test-task")
	mockMemoryManager.On("GetChainsByTask", mock.Anything, "test-task").Return([]*memory.PromptChain{testChain}, nil)

	// Mock the creation of a new chain
	mockMemoryManager.On("CreateChain", mock.Anything, "test-agent", "test-task").Return("chain-2", nil)

	// Mock adding messages to the new chain
	mockMemoryManager.On("AddMessage", mock.Anything, "chain-2", mock.Anything).Return(nil)

	// Test the craft solution method
	err := wrapper.CraftSolution(context.Background())
	assert.NoError(t, err)

	// Verify that the underlying agent's CraftSolution was called
	mockAgent.AssertCalled(t, "CraftSolution", mock.Anything)

	// Verify that memory operations were performed
	mockMemoryManager.AssertCalled(t, "GetChainsByTask", mock.Anything, "test-task")
	mockMemoryManager.AssertCalled(t, "CreateChain", mock.Anything, "test-agent", "test-task")
	mockMemoryManager.AssertCalled(t, "AddMessage", mock.Anything, "chain-2", mock.Anything)
}

func TestAgentWrapper_ExtractQueryFromTask(t *testing.T) {
	mockAgent, mockMemoryManager := createMockAgent(t)
	retriever := createMockRetriever()
	config := DefaultRetrievalConfig()

	wrapper := NewAgentWrapper(mockAgent, retriever, config)

	// Mock the memory chain retrieval
	testChain := createMockChain("chain-1", "test-agent", "test-task")
	mockMemoryManager.On("GetChainsByTask", mock.Anything, "test-task").Return([]*memory.PromptChain{testChain}, nil)

	// Test query extraction
	query := wrapper.extractQueryFromTask("test-task")
	
	// Expect the query to contain task title and description
	assert.Contains(t, query, "Test Task")
	assert.Contains(t, query, "This is a test task for RAG enhancement")
}

func TestNewRagAgent(t *testing.T) {
	// Create test dependencies
	mockMemoryManager := new(memorymocks.MockMemoryManager)
	retriever := createMockRetriever()
	
	// Create a test config
	config := &agent.AgentConfig{
		ID:          "test-rag-agent",
		Name:        "Test RAG Agent",
		Type:        "worker",
		Provider:    providers.ProviderTypeOpenAI,
		Model:       "gpt-3.5-turbo",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	
	// Create mock LLM client
	mockLlmClient := new(mocks.MockLLMClient)
	mockLlmClient.On("Complete", mock.Anything, mock.Anything).Return(&providers.CompletionResponse{
		Text:       "Test response",
		TokensUsed: 10,
	}, nil)
	
	// Create mock tool registry
	mockToolRegistry := new(mocks.MockToolRegistry)
	
	// Create mock objective manager
	mockObjectiveManager := new(mocks.MockObjectiveManager)
	
	// Test creating a RAG agent
	options := DefaultRagAgentOptions()
	ragAgent := NewRagAgent(
		config,
		mockLlmClient,
		mockMemoryManager,
		mockToolRegistry,
		mockObjectiveManager,
		retriever,
		options,
	)
	
	// Assert the agent was created successfully
	assert.NotNil(t, ragAgent)
	assert.Equal(t, "test-rag-agent", ragAgent.ID())
	assert.Equal(t, "Test RAG Agent", ragAgent.Name())
	assert.Equal(t, "worker", ragAgent.Type())
}