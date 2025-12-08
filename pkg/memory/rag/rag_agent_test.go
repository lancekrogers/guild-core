// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"
	"testing"

	"github.com/guild-framework/guild-core/pkg/commission"
	"github.com/guild-framework/guild-core/pkg/memory"
	"github.com/guild-framework/guild-core/pkg/memory/vector"
	"github.com/guild-framework/guild-core/pkg/providers"
	"github.com/guild-framework/guild-core/pkg/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test NewAgentWrapper
func TestNewAgentWrapper(t *testing.T) {
	baseAgent := &testBaseAgent{
		id:   "test-agent",
		name: "Test Agent",
	}
	retriever := &Retriever{}
	config := Config{MaxResults: 5}

	ragAgent := NewAgentWrapper(baseAgent, retriever, config)

	assert.NotNil(t, ragAgent)
	assert.Equal(t, baseAgent, ragAgent.agent)
	assert.Equal(t, retriever, ragAgent.retriever)
	assert.Equal(t, config, ragAgent.config)
}

// Test AgentWrapper Execute
func TestAgentWrapper_Execute(t *testing.T) {
	tests := []struct {
		name        string
		request     string
		setupMocks  func(*testBaseAgent, *Retriever)
		wantResp    string
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful execution with fallback",
			request: "How do I implement authentication?",
			setupMocks: func(agent *testBaseAgent, retriever *Retriever) {
				// Mock agent execution - will be called with original request due to nil retriever
				agent.executeFunc = func(ctx context.Context, request string) (string, error) {
					assert.Equal(t, "How do I implement authentication?", request)
					return "Based on the query, you should use JWT tokens for authentication.", nil
				}
			},
			wantResp: "Based on the query, you should use JWT tokens for authentication.",
			wantErr:  false,
		},
		{
			name:    "agent error",
			request: "test query",
			setupMocks: func(agent *testBaseAgent, retriever *Retriever) {
				agent.executeFunc = func(ctx context.Context, request string) (string, error) {
					return "", assert.AnError
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseAgent := &testBaseAgent{
				id:   "test-agent",
				name: "Test Agent",
			}

			// Create a retriever with nil vector store (enhancement will fail gracefully)
			retriever := (*Retriever)(nil)
			config := Config{MaxResults: 5}

			if tt.setupMocks != nil {
				tt.setupMocks(baseAgent, retriever)
			}

			wrapper := &AgentWrapper{
				agent:     baseAgent,
				retriever: retriever,
				config:    config,
			}

			ctx := context.Background()
			resp, err := wrapper.Execute(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResp, resp)
			}
		})
	}
}

// Test AgentWrapper interface methods
func TestAgentWrapper_InterfaceMethods(t *testing.T) {
	baseAgent := &testBaseAgent{
		id:            "test-agent",
		name:          "Test Agent",
		toolRegistry:  &testToolRegistry{},
		commissionMgr: &testCommissionManager{},
		llmClient:     &testLLMClient{},
		memoryManager: &testMemoryManager{},
	}
	retriever := &Retriever{Config: Config{MaxResults: 5}}
	config := Config{MaxResults: 5}

	wrapper := &AgentWrapper{
		agent:     baseAgent,
		retriever: retriever,
		config:    config,
	}

	// Test GetID
	assert.Equal(t, "test-agent", wrapper.GetID())

	// Test GetName
	assert.Equal(t, "Test Agent", wrapper.GetName())

	// Test GetToolRegistry
	assert.Equal(t, baseAgent.toolRegistry, wrapper.GetToolRegistry())

	// Test GetCommissionManager
	assert.Equal(t, baseAgent.commissionMgr, wrapper.GetCommissionManager())

	// Test GetLLMClient
	assert.Equal(t, baseAgent.llmClient, wrapper.GetLLMClient())

	// Test GetMemoryManager
	assert.Equal(t, baseAgent.memoryManager, wrapper.GetMemoryManager())
}

// Test EnhancePrompt
func TestAgentWrapper_EnhancePrompt(t *testing.T) {
	// Create mock vector store
	mockStore := &mockVectorStore{}
	mockStore.On("QueryEmbeddings", mock.Anything, "test query", 10).
		Return([]vector.EmbeddingMatch{}, nil) // Return empty results

	retriever := &Retriever{
		Config:      Config{MaxResults: 5},
		vectorStore: mockStore,
	}
	config := Config{MaxResults: 5}

	wrapper := &AgentWrapper{
		retriever: retriever,
		config:    config,
	}

	ctx := context.Background()
	retrievalConfig := RetrievalConfig{
		MaxResults: 5,
		MinScore:   0.5,
	}

	// Test with empty results (should return original prompt)
	enhanced, err := wrapper.EnhancePrompt(ctx, "test prompt", "test query", retrievalConfig)

	assert.NoError(t, err)
	assert.Equal(t, "test prompt", enhanced) // No enhancement with empty results

	mockStore.AssertExpectations(t)
}

// Test implementations

type testBaseAgent struct {
	id            string
	name          string
	executeFunc   func(context.Context, string) (string, error)
	toolRegistry  tools.Registry
	commissionMgr commission.CommissionManager
	llmClient     providers.LLMClient
	memoryManager memory.ChainManager
}

func (t *testBaseAgent) Execute(ctx context.Context, request string) (string, error) {
	if t.executeFunc != nil {
		return t.executeFunc(ctx, request)
	}
	return "default response", nil
}

func (t *testBaseAgent) GetID() string                                      { return t.id }
func (t *testBaseAgent) GetName() string                                    { return t.name }
func (t *testBaseAgent) GetToolRegistry() tools.Registry                    { return t.toolRegistry }
func (t *testBaseAgent) GetCommissionManager() commission.CommissionManager { return t.commissionMgr }
func (t *testBaseAgent) GetLLMClient() providers.LLMClient                  { return t.llmClient }
func (t *testBaseAgent) GetMemoryManager() memory.ChainManager              { return t.memoryManager }
func (t *testBaseAgent) GetType() string                                    { return "test" }
func (t *testBaseAgent) GetCapabilities() []string                          { return []string{"testing"} }

type testRetriever struct {
	enhanceFunc func(context.Context, string) (string, error)
}

func (t *testRetriever) RetrieveContext(ctx context.Context, query string) ([]SearchResult, error) {
	return []SearchResult{}, nil
}

func (t *testRetriever) AddDocument(ctx context.Context, id, content string, metadata map[string]interface{}) error {
	return nil
}

func (t *testRetriever) AddCorpusDocument(ctx context.Context, doc interface{}) error {
	return nil
}

func (t *testRetriever) EnhancePrompt(ctx context.Context, prompt string) (string, error) {
	if t.enhanceFunc != nil {
		return t.enhanceFunc(ctx, prompt)
	}
	return prompt, nil
}

func (t *testRetriever) RemoveDocument(ctx context.Context, id string) error {
	return nil
}

func (t *testRetriever) Close() error {
	return nil
}

// Simple test implementations for interfaces

type testToolRegistry struct{}

func (t *testToolRegistry) RegisterTool(name string, tool tools.Tool) error { return nil }
func (t *testToolRegistry) GetTool(name string) (tools.Tool, error)         { return nil, nil }
func (t *testToolRegistry) ListTools() []string                             { return []string{} }
func (t *testToolRegistry) HasTool(name string) bool                        { return false }
func (t *testToolRegistry) UnregisterTool(name string) error                { return nil }
func (t *testToolRegistry) Clear()                                          {}

type testCommissionManager struct{}

func (t *testCommissionManager) CreateCommission(ctx context.Context, commission commission.Commission) (*commission.Commission, error) {
	return &commission, nil
}
func (t *testCommissionManager) GetCommission(ctx context.Context, id string) (*commission.Commission, error) {
	return nil, nil
}
func (t *testCommissionManager) UpdateCommission(ctx context.Context, commission commission.Commission) error {
	return nil
}
func (t *testCommissionManager) DeleteCommission(ctx context.Context, id string) error { return nil }
func (t *testCommissionManager) ListCommissions(ctx context.Context) ([]*commission.Commission, error) {
	return []*commission.Commission{}, nil
}
func (t *testCommissionManager) SaveCommission(ctx context.Context, commission *commission.Commission) error {
	return nil
}
func (t *testCommissionManager) LoadCommissionFromFile(ctx context.Context, path string) (*commission.Commission, error) {
	return nil, nil
}
func (t *testCommissionManager) GetCommissionsByTag(ctx context.Context, tag string) ([]*commission.Commission, error) {
	return []*commission.Commission{}, nil
}
func (t *testCommissionManager) SetCommission(ctx context.Context, commissionID string) error {
	return nil
}

type testLLMClient struct{}

func (t *testLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	return "test response", nil
}

type testMemoryManager struct{}

func (t *testMemoryManager) CreateChain(ctx context.Context, agentID, taskID string) (string, error) {
	return "chain-id", nil
}
func (t *testMemoryManager) GetChain(ctx context.Context, chainID string) (*memory.PromptChain, error) {
	return nil, nil
}
func (t *testMemoryManager) AddMessage(ctx context.Context, chainID string, message memory.Message) error {
	return nil
}
func (t *testMemoryManager) GetChainsByAgent(ctx context.Context, agentID string) ([]*memory.PromptChain, error) {
	return []*memory.PromptChain{}, nil
}
func (t *testMemoryManager) GetChainsByTask(ctx context.Context, taskID string) ([]*memory.PromptChain, error) {
	return []*memory.PromptChain{}, nil
}
func (t *testMemoryManager) BuildContext(ctx context.Context, agentID, taskID string, maxTokens int) ([]memory.Message, error) {
	return []memory.Message{}, nil
}
func (t *testMemoryManager) DeleteChain(ctx context.Context, chainID string) error {
	return nil
}
func (t *testMemoryManager) AddInteraction(ctx context.Context, userInput, assistantResponse string) error {
	return nil
}
func (t *testMemoryManager) GetRecentMessages(ctx context.Context, limit int) ([]memory.Message, error) {
	return []memory.Message{}, nil
}
func (t *testMemoryManager) Clear(ctx context.Context) error {
	return nil
}
func (t *testMemoryManager) GetMessageCount(ctx context.Context) (int, error) {
	return 0, nil
}

// Mock vector store for testing
type mockVectorStore struct {
	mock.Mock
}

func (m *mockVectorStore) SaveEmbedding(ctx context.Context, embedding vector.Embedding) error {
	args := m.Called(ctx, embedding)
	return args.Error(0)
}

func (m *mockVectorStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]vector.EmbeddingMatch, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]vector.EmbeddingMatch), args.Error(1)
}

func (m *mockVectorStore) QueryCollection(ctx context.Context, collectionName, query string, limit int) ([]vector.EmbeddingMatch, error) {
	args := m.Called(ctx, collectionName, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]vector.EmbeddingMatch), args.Error(1)
}

func (m *mockVectorStore) DeleteEmbedding(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockVectorStore) Close() error {
	args := m.Called()
	return args.Error(0)
}
