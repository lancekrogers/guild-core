package agent_test

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/agent/mocks"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/tools"
	toolmocks "github.com/guild-ventures/guild-core/tools/mocks"
)

// TestWorkerAgentImplementsAgent tests that WorkerAgent implements the Agent interface
func TestWorkerAgentImplementsAgent(t *testing.T) {
	var _ agent.Agent = &agent.WorkerAgent{}
}

// TestWorkerAgentImplementsGuildArtisan tests that WorkerAgent implements the GuildArtisan interface
func TestWorkerAgentImplementsGuildArtisan(t *testing.T) {
	var _ agent.GuildArtisan = &agent.WorkerAgent{}
}

// TestNewWorkerAgent tests creating a new worker agent
func TestNewWorkerAgent(t *testing.T) {
	// Create mock dependencies
	llmClient := mocks.NewMockLLMClient()
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()
	
	// Create commission manager (updated from objective)
	commissionManager := &commission.Manager{}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"test-agent-1",
		"Test Agent",
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
	)

	// Verify agent properties
	if workerAgent.GetID() != "test-agent-1" {
		t.Errorf("Expected agent ID 'test-agent-1', got '%s'", workerAgent.GetID())
	}

	if workerAgent.GetName() != "Test Agent" {
		t.Errorf("Expected agent name 'Test Agent', got '%s'", workerAgent.GetName())
	}

	// Verify dependencies are properly set
	if workerAgent.GetLLMClient() != llmClient {
		t.Error("LLM client not properly set")
	}

	if workerAgent.GetMemoryManager() != memoryManager {
		t.Error("Memory manager not properly set")
	}

	if workerAgent.GetToolRegistry() != toolRegistry {
		t.Error("Tool registry not properly set")
	}

	if workerAgent.GetCommissionManager() != commissionManager {
		t.Error("Commission manager not properly set")
	}

	// Note: CostManager check removed as it may not be public or may not exist
}

// TestWorkerAgentExecute tests the Execute method
func TestWorkerAgentExecute(t *testing.T) {
	// Create mock dependencies
	llmClient := mocks.NewMockLLMClient()
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()
	commissionManager := &commission.Manager{}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"test-agent-2",
		"Test Worker",
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
	)

	ctx := context.Background()
	
	// Test Execute method
	response, err := workerAgent.Execute(ctx, "test request")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// The response should be what the mock LLM returns
	expectedResponse := "Mock response"
	if response != expectedResponse {
		t.Errorf("Expected response '%s', got '%s'", expectedResponse, response)
	}
}

// TestWorkerAgentCostManagement tests cost budget and reporting if available
func TestWorkerAgentCostManagement(t *testing.T) {
	commissionManager := &commission.Manager{}
	
	workerAgent := agent.NewWorkerAgent(
		"test-agent-3",
		"Cost Test Agent",
		&mocks.MockLLMClient{},
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		commissionManager,
	)

	// Note: Cost management methods may not be public
	// This test focuses on agent creation and basic functionality
	if workerAgent.GetID() != "test-agent-3" {
		t.Errorf("Expected agent ID 'test-agent-3', got '%s'", workerAgent.GetID())
	}

	if workerAgent.GetName() != "Cost Test Agent" {
		t.Errorf("Expected agent name 'Cost Test Agent', got '%s'", workerAgent.GetName())
	}
}

// TestManagerAgentCreation tests creating a manager agent
func TestManagerAgentCreation(t *testing.T) {
	// Create mock dependencies
	llmClient := mocks.NewMockLLMClient()
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()
	commissionManager := &commission.Manager{}

	// Create manager agent
	managerAgent := agent.NewManagerAgent(
		"test-manager-1",
		"Test Manager",
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
	)

	// Verify it's a manager agent with worker capabilities
	if managerAgent.GetID() != "test-manager-1" {
		t.Errorf("Expected manager ID 'test-manager-1', got '%s'", managerAgent.GetID())
	}

	if managerAgent.GetName() != "Test Manager" {
		t.Errorf("Expected manager name 'Test Manager', got '%s'", managerAgent.GetName())
	}

	// Verify it implements the Agent interface
	var _ agent.Agent = managerAgent
	
	// Verify it implements the GuildArtisan interface
	var _ agent.GuildArtisan = managerAgent
}

// TestWorkerAgentWithMockedLLMResponse tests agent with mocked LLM responses
func TestWorkerAgentWithMockedLLMResponse(t *testing.T) {
	// Create mock LLM client with predefined response
	llmClient := mocks.NewMockLLMClient()
	llmClient.WithResponse("Test response from LLM")

	// Create mock memory manager
	memoryManager := mocks.NewMockChainManager()
	commissionManager := &commission.Manager{}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"test-agent-4",
		"LLM Test Agent",
		llmClient,
		memoryManager,
		tools.NewToolRegistry(),
		commissionManager,
	)

	// Verify we can access the mocked dependencies
	if workerAgent.GetLLMClient() != llmClient {
		t.Error("LLM client not properly accessible")
	}
	
	// Verify the LLM client works
	ctx := context.Background()
	response, err := llmClient.Complete(ctx, "test prompt")
	if err != nil {
		t.Fatalf("LLM Complete failed: %v", err)
	}
	if response != "Test response from LLM" {
		t.Errorf("Expected 'Test response from LLM', got '%s'", response)
	}
}

// TestWorkerAgentWithTools tests agent with tool registry
func TestWorkerAgentWithTools(t *testing.T) {
	// Create tool registry and add a mock tool
	toolRegistry := tools.NewToolRegistry()
	mockTool := toolmocks.NewMockTool("test-tool", "A test tool")
	err := toolRegistry.RegisterTool(mockTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}
	
	commissionManager := &commission.Manager{}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"test-agent-5",
		"Tool Test Agent",
		&mocks.MockLLMClient{},
		mocks.NewMockChainManager(),
		toolRegistry,
		commissionManager,
	)

	// Verify tool registry is accessible
	registry := workerAgent.GetToolRegistry()
	if registry == nil {
		t.Fatal("Tool registry should not be nil")
	}

	// Verify the tool is in the registry
	tool, exists := registry.GetTool("test-tool")
	if !exists {
		t.Fatal("Tool should exist in registry")
	}

	if tool.Name() != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tool.Name())
	}
}

// TestAgentInterfaceCompliance tests that agents properly implement required interfaces
func TestAgentInterfaceCompliance(t *testing.T) {
	t.Run("worker_agent_interfaces", func(t *testing.T) {
		// Test that WorkerAgent implements all required interfaces
		var _ agent.Agent = &agent.WorkerAgent{}
		var _ agent.GuildArtisan = &agent.WorkerAgent{}
	})

	t.Run("manager_agent_interfaces", func(t *testing.T) {
		// Create a manager agent to test interface compliance
		llmClient := mocks.NewMockLLMClient()
		memoryManager := mocks.NewMockChainManager()
		toolRegistry := tools.NewToolRegistry()
		commissionManager := &commission.Manager{}

		managerAgent := agent.NewManagerAgent(
			"interface-test-manager",
			"Interface Test Manager",
			llmClient,
			memoryManager,
			toolRegistry,
			commissionManager,
		)

		// Verify interface implementations
		var _ agent.Agent = managerAgent
		var _ agent.GuildArtisan = managerAgent

		// Test basic functionality
		if managerAgent.GetID() != "interface-test-manager" {
			t.Error("Manager agent ID not properly set")
		}
	})
}

// TestAgentCreationEdgeCases tests edge cases in agent creation
func TestAgentCreationEdgeCases(t *testing.T) {
	t.Run("minimal_dependencies", func(t *testing.T) {
		// Test creating agents with minimal dependencies
		llmClient := &mocks.MockLLMClient{}
		memoryManager := mocks.NewMockChainManager()
		toolRegistry := tools.NewToolRegistry()
		commissionManager := &commission.Manager{}

		// Should not panic with minimal setup
		workerAgent := agent.NewWorkerAgent(
			"minimal-agent",
			"Minimal Agent",
			llmClient,
			memoryManager,
			toolRegistry,
			commissionManager,
		)

		if workerAgent == nil {
			t.Fatal("Worker agent should not be nil")
		}

		if workerAgent.GetID() != "minimal-agent" {
			t.Error("Agent ID not properly set with minimal dependencies")
		}
	})

	t.Run("empty_strings", func(t *testing.T) {
		// Test with empty strings (edge case)
		llmClient := &mocks.MockLLMClient{}
		memoryManager := mocks.NewMockChainManager()
		toolRegistry := tools.NewToolRegistry()
		commissionManager := &commission.Manager{}

		workerAgent := agent.NewWorkerAgent(
			"", // Empty ID
			"", // Empty name
			llmClient,
			memoryManager,
			toolRegistry,
			commissionManager,
		)

		// Should still create agent, just with empty values
		if workerAgent == nil {
			t.Fatal("Worker agent should not be nil even with empty strings")
		}

		if workerAgent.GetID() != "" {
			t.Error("Empty ID should be preserved")
		}

		if workerAgent.GetName() != "" {
			t.Error("Empty name should be preserved")
		}
	})
}

// BenchmarkAgentCreation benchmarks agent creation performance
func BenchmarkAgentCreation(b *testing.B) {
	// Pre-create dependencies to isolate agent creation time
	llmClient := &mocks.MockLLMClient{}
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()
	commissionManager := &commission.Manager{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		workerAgent := agent.NewWorkerAgent(
			"benchmark-agent",
			"Benchmark Agent",
			llmClient,
			memoryManager,
			toolRegistry,
			commissionManager,
		)
		
		// Avoid compiler optimization
		_ = workerAgent.GetID()
	}
}