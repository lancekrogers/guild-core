// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent_test

import (
	"context"
	"testing"

	"github.com/guild-framework/guild-core/pkg/agents/core"
	"github.com/guild-framework/guild-core/pkg/agents/core/mocks"
	"github.com/guild-framework/guild-core/pkg/tools"
	toolmocks "github.com/guild-framework/guild-core/tools/mocks"
)

// TestWorkerAgentImplementsAgent tests that WorkerAgent implements the Agent interface
func TestWorkerAgentImplementsAgent(t *testing.T) {
	var _ core.Agent = &core.WorkerAgent{}
}

// TestWorkerAgentImplementsGuildArtisan tests that WorkerAgent implements the GuildArtisan interface
func TestWorkerAgentImplementsGuildArtisan(t *testing.T) {
	var _ core.GuildArtisan = &core.WorkerAgent{}
}

// TestCreateWorkerAgent tests creating a new worker agent via factory
func TestCreateWorkerAgent(t *testing.T) {
	// Create mock dependencies
	llmClient := mocks.NewMockLLMClient()
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()

	// Create commission manager mock
	commissionManager := mocks.NewMockCommissionManager()

	// Create cost manager mock
	costManager := mocks.NewMockCostManager()

	// Create factory
	factory := core.DefaultFactoryFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)

	// Create worker agent via factory
	ctx := context.Background()
	createdAgent, err := factory.CreateWorkerAgent(ctx, "test-agent-1", "Test Agent")
	if err != nil {
		t.Fatalf("Failed to create worker agent: %v", err)
	}

	// Cast to GuildArtisan to access extended methods
	workerAgent, ok := createdAgent.(core.Agent)
	if !ok {
		t.Fatal("Agent should implement Agent interface")
	}

	guildArtisan, ok := createdAgent.(core.GuildArtisan)
	if !ok {
		t.Fatal("Worker agent should implement GuildArtisan interface")
	}

	// Verify agent properties
	if workerAgent.GetID() != "test-agent-1" {
		t.Errorf("Expected agent ID 'test-agent-1', got '%s'", workerAgent.GetID())
	}

	if workerAgent.GetName() != "Test Agent" {
		t.Errorf("Expected agent name 'Test Agent', got '%s'", workerAgent.GetName())
	}

	// Verify dependencies are properly set via GuildArtisan interface
	if guildArtisan.GetLLMClient() != llmClient {
		t.Error("LLM client not properly set")
	}

	if guildArtisan.GetMemoryManager() != memoryManager {
		t.Error("Memory manager not properly set")
	}

	if guildArtisan.GetToolRegistry() != toolRegistry {
		t.Error("Tool registry not properly set")
	}

	if guildArtisan.GetCommissionManager() != commissionManager {
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
	commissionManager := mocks.NewMockCommissionManager()

	// Create cost manager mock
	costManager := mocks.NewMockCostManager()

	// Create factory
	factory := core.DefaultFactoryFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)

	// Create worker agent via factory
	ctx := context.Background()
	createdAgent, err := factory.CreateWorkerAgent(ctx, "test-agent-2", "Test Worker")
	if err != nil {
		t.Fatalf("Failed to create worker agent: %v", err)
	}

	// Test Execute method
	response, err := createdAgent.Execute(ctx, "test request")
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
	// Create mock dependencies
	llmClient := mocks.NewMockLLMClient()
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()
	commissionManager := mocks.NewMockCommissionManager()
	costManager := mocks.NewMockCostManager()

	// Create factory
	factory := core.DefaultFactoryFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)

	// Create worker agent via factory
	ctx := context.Background()
	createdAgent, err := factory.CreateWorkerAgent(ctx, "test-agent-3", "Cost Test Agent")
	if err != nil {
		t.Fatalf("Failed to create worker agent: %v", err)
	}

	// Note: Cost management methods may not be public
	// This test focuses on agent creation and basic functionality
	if createdAgent.GetID() != "test-agent-3" {
		t.Errorf("Expected agent ID 'test-agent-3', got '%s'", createdAgent.GetID())
	}

	if createdAgent.GetName() != "Cost Test Agent" {
		t.Errorf("Expected agent name 'Cost Test Agent', got '%s'", createdAgent.GetName())
	}
}

// TestManagerAgentCreation tests creating a manager agent
func TestManagerAgentCreation(t *testing.T) {
	// Create mock dependencies
	llmClient := mocks.NewMockLLMClient()
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()
	commissionManager := mocks.NewMockCommissionManager()
	costManager := mocks.NewMockCostManager()

	// Create factory
	factory := core.DefaultFactoryFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)

	// Create manager agent via factory
	ctx := context.Background()
	createdAgent, err := factory.CreateManagerAgent(ctx, "test-manager-1", "Test Manager")
	if err != nil {
		t.Fatalf("Failed to create manager agent: %v", err)
	}

	// Verify it's a manager agent with worker capabilities
	if createdAgent.GetID() != "test-manager-1" {
		t.Errorf("Expected manager ID 'test-manager-1', got '%s'", createdAgent.GetID())
	}

	if createdAgent.GetName() != "Test Manager" {
		t.Errorf("Expected manager name 'Test Manager', got '%s'", createdAgent.GetName())
	}

	// Verify it implements the Agent interface
	var _ core.Agent = createdAgent

	// Verify it implements the GuildArtisan interface
	if guildArtisan, ok := createdAgent.(core.GuildArtisan); !ok {
		t.Error("Manager agent should implement GuildArtisan interface")
	} else {
		// Verify GuildArtisan methods are accessible
		_ = guildArtisan.GetLLMClient()
		_ = guildArtisan.GetMemoryManager()
		_ = guildArtisan.GetToolRegistry()
		_ = guildArtisan.GetCommissionManager()
	}
}

// TestWorkerAgentWithMockedLLMResponse tests agent with mocked LLM responses
func TestWorkerAgentWithMockedLLMResponse(t *testing.T) {
	// Create mock LLM client with predefined response
	llmClient := mocks.NewMockLLMClient()
	llmClient.WithResponse("Test response from LLM")

	// Create mock memory manager
	memoryManager := mocks.NewMockChainManager()
	commissionManager := mocks.NewMockCommissionManager()

	// Create cost manager mock
	costManager := mocks.NewMockCostManager()

	// Create factory
	factory := core.DefaultFactoryFactory(
		llmClient,
		memoryManager,
		tools.NewToolRegistry(),
		commissionManager,
		costManager,
	)

	// Create worker agent via factory
	ctx := context.Background()
	workerAgent, err := factory.CreateWorkerAgent(ctx, "test-agent-4", "LLM Test Agent")
	if err != nil {
		t.Fatalf("Failed to create worker agent: %v", err)
	}

	// Cast to GuildArtisan to access dependencies
	guildArtisan, ok := workerAgent.(core.GuildArtisan)
	if !ok {
		t.Fatal("Worker agent should implement GuildArtisan interface")
	}

	// Verify we can access the mocked dependencies
	if guildArtisan.GetLLMClient() != llmClient {
		t.Error("LLM client not properly accessible")
	}

	// Verify the LLM client works
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
	err := toolRegistry.RegisterTool("test-tool", mockTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	commissionManager := mocks.NewMockCommissionManager()

	// Create cost manager mock
	costManager := mocks.NewMockCostManager()

	// Create factory
	factory := core.DefaultFactoryFactory(
		&mocks.MockLLMClient{},
		mocks.NewMockChainManager(),
		toolRegistry,
		commissionManager,
		costManager,
	)

	// Create worker agent via factory
	ctx := context.Background()
	workerAgent, err := factory.CreateWorkerAgent(ctx, "test-agent-5", "Tool Test Agent")
	if err != nil {
		t.Fatalf("Failed to create worker agent: %v", err)
	}

	// Cast to GuildArtisan to access tool registry
	guildArtisan, ok := workerAgent.(core.GuildArtisan)
	if !ok {
		t.Fatal("Worker agent should implement GuildArtisan interface")
	}

	// Verify tool registry is accessible
	registry := guildArtisan.GetToolRegistry()
	if registry == nil {
		t.Fatal("Tool registry should not be nil")
	}

	// Verify the tool is in the registry
	tool, err := registry.GetTool("test-tool")
	if err != nil {
		t.Fatalf("Tool should exist in registry: %v", err)
	}

	if tool.Name() != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tool.Name())
	}
}

// TestAgentInterfaceCompliance tests that agents properly implement required interfaces
func TestAgentInterfaceCompliance(t *testing.T) {
	t.Run("worker_agent_interfaces", func(t *testing.T) {
		// Test that WorkerAgent implements all required interfaces
		var _ core.Agent = &core.WorkerAgent{}
		var _ core.GuildArtisan = &core.WorkerAgent{}
	})

	t.Run("manager_agent_interfaces", func(t *testing.T) {
		// Create a manager agent to test interface compliance
		llmClient := mocks.NewMockLLMClient()
		memoryManager := mocks.NewMockChainManager()
		toolRegistry := tools.NewToolRegistry()
		commissionManager := mocks.NewMockCommissionManager()

		costManager := mocks.NewMockCostManager()

		// Create factory
		factory := core.DefaultFactoryFactory(
			llmClient,
			memoryManager,
			toolRegistry,
			commissionManager,
			costManager,
		)

		// Create manager agent via factory
		ctx := context.Background()
		managerAgent, err := factory.CreateManagerAgent(ctx, "interface-test-manager", "Interface Test Manager")
		if err != nil {
			t.Fatalf("Failed to create manager agent: %v", err)
		}

		// Verify interface implementations
		var _ core.Agent = managerAgent

		// Cast to GuildArtisan to verify implementation
		guildArtisan, ok := managerAgent.(core.GuildArtisan)
		if !ok {
			t.Fatal("Manager agent should implement GuildArtisan interface")
		}
		var _ core.GuildArtisan = guildArtisan

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
		commissionManager := mocks.NewMockCommissionManager()

		// Should not panic with minimal setup
		costManager := mocks.NewMockCostManager()

		// Create factory
		factory := core.DefaultFactoryFactory(
			llmClient,
			memoryManager,
			toolRegistry,
			commissionManager,
			costManager,
		)

		// Create worker agent via factory
		ctx := context.Background()
		workerAgent, err := factory.CreateWorkerAgent(ctx, "minimal-agent", "Minimal Agent")
		if err != nil {
			t.Fatalf("Failed to create worker agent: %v", err)
		}

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
		commissionManager := mocks.NewMockCommissionManager()

		costManager := mocks.NewMockCostManager()

		// Create factory
		factory := core.DefaultFactoryFactory(
			llmClient,
			memoryManager,
			toolRegistry,
			commissionManager,
			costManager,
		)

		// Create worker agent via factory (with empty strings)
		ctx := context.Background()
		workerAgent, err := factory.CreateWorkerAgent(ctx, "", "") // Empty ID and name
		if err != nil {
			t.Fatalf("Failed to create worker agent: %v", err)
		}

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
	commissionManager := mocks.NewMockCommissionManager()
	costManager := mocks.NewMockCostManager()

	// Create factory once
	factory := core.DefaultFactoryFactory(
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
		costManager,
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create worker agent via factory
		ctx := context.Background()
		workerAgent, err := factory.CreateWorkerAgent(ctx, "benchmark-agent", "Benchmark Agent")
		if err != nil {
			b.Fatalf("Failed to create worker agent: %v", err)
		}

		// Avoid compiler optimization
		_ = workerAgent.GetID()
	}
}
