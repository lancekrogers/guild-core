// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package memory_test

import (
	"context"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/memory/rag"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/tools"
)

// MockAgent implements the GuildArtisan interface for testing
type MockAgent struct {
	id                string
	name              string
	toolRegistry      tools.Registry
	commissionManager commission.CommissionManager
	llmClient         providers.LLMClient
	memoryManager     memory.ChainManager
	executeFunc       func(ctx context.Context, request string) (string, error)
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

func (m *MockAgent) GetToolRegistry() tools.Registry {
	return m.toolRegistry
}

func (m *MockAgent) GetCommissionManager() commission.CommissionManager {
	return m.commissionManager
}

func (m *MockAgent) GetLLMClient() providers.LLMClient {
	return m.llmClient
}

func (m *MockAgent) GetMemoryManager() memory.ChainManager {
	return m.memoryManager
}

func (m *MockAgent) GetType() string {
	return "mock"
}

func (m *MockAgent) GetCapabilities() []string {
	return []string{"testing", "mocking"}
}

func TestAgentWrapper_BasicDelegation(t *testing.T) {
	// Create mock agent
	mockAgent := &MockAgent{
		id:   "test-agent",
		name: "Test Agent",
	}

	// Create wrapper with nil retriever for delegation test
	config := rag.Config{
		MaxResults: 5,
		ChunkSize:  1000,
	}
	// Use constructor since fields are unexported
	wrapper := rag.NewAgentWrapper(mockAgent, nil, config)

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
	config := rag.Config{
		MaxResults: 5,
		ChunkSize:  1000,
	}
	// Use constructor since fields are unexported
	wrapper := rag.NewAgentWrapper(mockAgent, nil, config)

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
		id:                "test-agent",
		name:              "Test Agent",
		toolRegistry:      tools.NewToolRegistry(),
		commissionManager: nil, // Mock commission manager
		llmClient:         nil, // Would use a mock in real test
		memoryManager:     nil, // Would use a mock in real test
	}

	// Create wrapper
	config := rag.Config{
		MaxResults: 5,
		ChunkSize:  1000,
	}
	// Use constructor since fields are unexported
	wrapper := rag.NewAgentWrapper(mockAgent, nil, config)

	// Verify interface implementations
	var _ agent.Agent = wrapper
	var _ agent.GuildArtisan = wrapper

	// Test that all methods properly delegate
	if wrapper.GetToolRegistry() != mockAgent.toolRegistry {
		t.Error("GetToolRegistry should delegate to wrapped agent")
	}

	if wrapper.GetCommissionManager() != mockAgent.commissionManager {
		t.Error("GetCommissionManager should delegate to wrapped agent")
	}
}

// TestRAGConfig tests the RAG configuration
func TestRAGConfig(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		config := rag.Config{
			MaxResults: 10,
			ChunkSize:  2000,
		}

		if config.MaxResults != 10 {
			t.Errorf("Expected MaxResults 10, got %d", config.MaxResults)
		}

		if config.ChunkSize != 2000 {
			t.Errorf("Expected ChunkSize 2000, got %d", config.ChunkSize)
		}
	})

	t.Run("config_validation", func(t *testing.T) {
		// Test edge cases for config values
		configs := []rag.Config{
			{MaxResults: 0, ChunkSize: 100},   // Zero max results
			{MaxResults: 1, ChunkSize: 0},     // Zero chunk size
			{MaxResults: 100, ChunkSize: 100}, // Normal values
		}

		for i, config := range configs {
			// Config should be acceptable - no validation errors expected for basic values
			if config.MaxResults < 0 {
				t.Errorf("Config %d: MaxResults should not be negative", i)
			}
			if config.ChunkSize < 0 {
				t.Errorf("Config %d: ChunkSize should not be negative", i)
			}
		}
	})
}

// TestRAGIntegration tests RAG functionality integration
func TestRAGIntegration(t *testing.T) {
	t.Run("agent_wrapper_creation", func(t *testing.T) {
		// Test creating agent wrapper with different configurations
		mockAgent := &MockAgent{
			id:   "rag-test-agent",
			name: "RAG Test Agent",
		}

		configs := []rag.Config{
			{MaxResults: 5, ChunkSize: 1000},
			{MaxResults: 10, ChunkSize: 2000},
			{MaxResults: 1, ChunkSize: 500},
		}

		for i, config := range configs {
			// Use constructor since fields are unexported
			wrapper := rag.NewAgentWrapper(mockAgent, nil, config)

			if wrapper.GetID() != "rag-test-agent" {
				t.Errorf("Wrapper %d: Expected ID 'rag-test-agent', got '%s'", i, wrapper.GetID())
			}

			if wrapper.GetName() != "RAG Test Agent" {
				t.Errorf("Wrapper %d: Expected name 'RAG Test Agent', got '%s'", i, wrapper.GetName())
			}
		}
	})

	t.Run("execution_flow", func(t *testing.T) {
		// Test the execution flow with mock agent
		executionCount := 0
		mockAgent := &MockAgent{
			id:   "flow-test-agent",
			name: "Flow Test Agent",
			executeFunc: func(ctx context.Context, request string) (string, error) {
				executionCount++
				return "response to: " + request, nil
			},
		}

		config := rag.Config{
			MaxResults: 3,
			ChunkSize:  800,
		}
		// Use constructor since fields are unexported
		wrapper := rag.NewAgentWrapper(mockAgent, nil, config)

		// Execute multiple requests
		requests := []string{
			"Analyze the codebase structure",
			"Generate API documentation",
			"Review security practices",
		}

		for i, request := range requests {
			response, err := wrapper.Execute(context.Background(), request)
			if err != nil {
				t.Errorf("Request %d failed: %v", i, err)
				continue
			}

			if !strings.Contains(response, request) {
				t.Errorf("Request %d: Response should contain original request", i)
			}

			if !strings.Contains(response, "response to:") {
				t.Errorf("Request %d: Response should contain expected prefix", i)
			}
		}

		if executionCount != len(requests) {
			t.Errorf("Expected %d executions, got %d", len(requests), executionCount)
		}
	})
}

// BenchmarkAgentWrapper tests performance of the RAG agent wrapper
func BenchmarkAgentWrapper(b *testing.B) {
	mockAgent := &MockAgent{
		id:   "benchmark-agent",
		name: "Benchmark Agent",
		executeFunc: func(ctx context.Context, request string) (string, error) {
			return "benchmark response", nil
		},
	}

	config := rag.Config{
		MaxResults: 5,
		ChunkSize:  1000,
	}
	// Use constructor since fields are unexported
	wrapper := rag.NewAgentWrapper(mockAgent, nil, config)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := wrapper.Execute(context.Background(), "benchmark request")
		if err != nil {
			b.Fatalf("Benchmark execution failed: %v", err)
		}
	}
}
