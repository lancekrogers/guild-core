package rag

import (
	"context"
	"strings"
	"testing"
	
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/objective"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// MockAgent implements the GuildArtisan interface for testing
type MockAgent struct {
	id              string
	name            string
	toolRegistry    *tools.ToolRegistry
	objectiveManager *objective.Manager
	llmClient       providers.LLMClient
	memoryManager   memory.ChainManager
	executeFunc     func(ctx context.Context, request string) (string, error)
}

func (m *MockAgent) Execute(ctx context.Context, request string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, request)
	}
	return "mock response: " + request, nil
}

func (m *MockAgent) GetID() string {
	return m.id
}

func (m *MockAgent) GetName() string {
	return m.name
}

func (m *MockAgent) GetToolRegistry() *tools.ToolRegistry {
	return m.toolRegistry
}

func (m *MockAgent) GetObjectiveManager() *objective.Manager {
	return m.objectiveManager
}

func (m *MockAgent) GetLLMClient() providers.LLMClient {
	return m.llmClient
}

func (m *MockAgent) GetMemoryManager() memory.ChainManager {
	return m.memoryManager
}

func TestAgentWrapper_BasicDelegation(t *testing.T) {
	// Create mock agent
	mockAgent := &MockAgent{
		id:   "test-agent",
		name: "Test Agent",
	}
	
	// Create wrapper with nil retriever for delegation test
	config := Config{
		MaxResults: 5,
		ChunkSize:  1000,
	}
	wrapper := &AgentWrapper{
		agent:  mockAgent,
		config: config,
	}
	
	// Test delegation methods
	if wrapper.GetID() != "test-agent" {
		t.Errorf("Expected ID 'test-agent', got '%s'", wrapper.GetID())
	}
	
	if wrapper.GetName() != "Test Agent" {
		t.Errorf("Expected name 'Test Agent', got '%s'", wrapper.GetName())
	}
}

func TestAgentWrapper_ExecuteWithoutRetriever(t *testing.T) {
	// Create mock agent
	var capturedRequest string
	mockAgent := &MockAgent{
		id:   "test-agent",
		name: "Test Agent",
		executeFunc: func(ctx context.Context, request string) (string, error) {
			capturedRequest = request
			return "executed: " + request, nil
		},
	}
	
	// Create wrapper with nil retriever
	config := Config{
		MaxResults: 5,
		ChunkSize:  1000,
	}
	wrapper := &AgentWrapper{
		agent:  mockAgent,
		config: config,
	}
	
	// Execute without RAG enhancement
	ctx := context.Background()
	originalRequest := "What is the capital of France?"
	response, err := wrapper.Execute(ctx, originalRequest)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Verify response
	if !strings.Contains(response, "executed:") {
		t.Errorf("Expected response to contain 'executed:', got '%s'", response)
	}
	
	// Verify the request was not enhanced (no retriever)
	if capturedRequest != originalRequest {
		t.Errorf("Request should not be enhanced when no retriever, got '%s'", capturedRequest)
	}
}

// TestAgentWrapper_InterfaceCompliance verifies the wrapper properly implements interfaces
func TestAgentWrapper_InterfaceCompliance(t *testing.T) {
	// Create mock agent
	mockAgent := &MockAgent{
		id:            "test-agent",
		name:          "Test Agent",
		toolRegistry:  tools.NewToolRegistry(),
		objectiveManager: &objective.Manager{},
		llmClient:     nil, // Would use a mock in real test
		memoryManager: nil, // Would use a mock in real test
	}
	
	// Create wrapper
	config := Config{
		MaxResults: 5,
		ChunkSize:  1000,
	}
	wrapper := &AgentWrapper{
		agent:  mockAgent,
		config: config,
	}
	
	// Verify interface implementations
	var _ agent.Agent = wrapper
	var _ agent.GuildArtisan = wrapper
	
	// Test that all methods properly delegate
	if wrapper.GetToolRegistry() != mockAgent.toolRegistry {
		t.Error("GetToolRegistry should delegate to wrapped agent")
	}
	
	if wrapper.GetObjectiveManager() != mockAgent.objectiveManager {
		t.Error("GetObjectiveManager should delegate to wrapped agent")
	}
}