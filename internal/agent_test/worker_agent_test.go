package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/agent/mocks"
	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/tools"
	toolmocks "github.com/guild-ventures/guild-core/tools/mocks"
)

// TestWorkerAgentWithContext tests WorkerAgent with context handling
func TestWorkerAgentWithContext(t *testing.T) {
	commissionManager := &commission.Manager{}
	
	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"ctx-test-agent",
		"Context Test Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		commissionManager,
	)

	// Test with context
	ctx := context.Background()
	response, err := workerAgent.Execute(ctx, "test with context")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if response != "Mock response" {
		t.Errorf("Unexpected response: %s", response)
	}

	// Test with cancelled context
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The implementation should handle cancelled context
	_, err = workerAgent.Execute(cancelCtx, "test with cancelled context")
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
	// The error should indicate context cancellation, either directly or wrapped
	if !strings.Contains(err.Error(), "context canceled") && err != context.Canceled {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

// TestWorkerAgentCostTracking tests cost tracking functionality if available
func TestWorkerAgentCostTracking(t *testing.T) {
	commissionManager := &commission.Manager{}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"cost-test-agent",
		"Cost Test Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		commissionManager,
	)

	// Test basic execution (cost tracking may not be publicly available)
	ctx := context.Background()
	response, err := workerAgent.Execute(ctx, "test operation")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}

	// Verify agent properties remain accessible
	if workerAgent.GetID() != "cost-test-agent" {
		t.Error("Agent ID should remain accessible")
	}
}

// TestManagerAgentInheritance tests that ManagerAgent inherits WorkerAgent functionality
func TestManagerAgentInheritance(t *testing.T) {
	commissionManager := &commission.Manager{}
	
	// Create manager agent
	managerAgent := agent.NewManagerAgent(
		"manager-test",
		"Manager Test",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		commissionManager,
	)

	// Test that it has all WorkerAgent functionality
	ctx := context.Background()
	response, err := managerAgent.Execute(ctx, "manager request")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if response != "Mock response" {
		t.Errorf("Unexpected response: %s", response)
	}

	// Verify manager agent has all expected properties
	if managerAgent.GetID() != "manager-test" {
		t.Error("Manager agent ID not properly set")
	}

	if managerAgent.GetName() != "Manager Test" {
		t.Error("Manager agent name not properly set")
	}
}

// TestWorkerAgentWithFullMemoryContext tests agent with complete memory context
func TestWorkerAgentWithFullMemoryContext(t *testing.T) {
	// Create memory manager with conversation history
	memoryManager := mocks.NewMockChainManager()
	messages := []memory.Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant",
		},
		{
			Role:    "user",
			Content: "Hello",
		},
		{
			Role:    "assistant",
			Content: "Hello! How can I help you?",
		},
		{
			Role:    "user",
			Content: "What's 2+2?",
		},
	}
	memoryManager.WithBuildContext(messages)

	commissionManager := &commission.Manager{}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"memory-test-agent",
		"Memory Test Agent",
		mocks.NewMockLLMClient(),
		memoryManager,
		tools.NewToolRegistry(),
		commissionManager,
	)

	// Verify memory manager is accessible and has context
	if workerAgent.GetMemoryManager() != memoryManager {
		t.Error("Memory manager not properly set")
	}

	// Test execution with memory context
	ctx := context.Background()
	response, err := workerAgent.Execute(ctx, "continue conversation")
	if err != nil {
		t.Fatalf("Execute with memory context failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response with memory context")
	}
}

// TestWorkerAgentWithCommissionManager tests agent with commission manager
func TestWorkerAgentWithCommissionManager(t *testing.T) {
	// Create commission manager
	commissionManager := &commission.Manager{}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"commission-test-agent",
		"Commission Test Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		commissionManager,
	)

	// Verify commission manager is accessible
	if workerAgent.GetCommissionManager() != commissionManager {
		t.Error("Commission manager not properly set")
	}

	// Test commission-based execution
	ctx := context.Background()
	response, err := workerAgent.Execute(ctx, "work on commission")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}
}

// TestWorkerAgentToolExecution tests agent executing tools
func TestWorkerAgentToolExecution(t *testing.T) {
	// Create tool registry with multiple tools
	toolRegistry := tools.NewToolRegistry()
	
	// Add calculation tool
	calcTool := toolmocks.NewMockTool("calculator", "Performs calculations")
	err := toolRegistry.RegisterTool(calcTool)
	if err != nil {
		t.Fatalf("Failed to register calculator tool: %v", err)
	}
	
	// Add search tool
	searchTool := toolmocks.NewMockTool("search", "Searches the web")
	err = toolRegistry.RegisterTool(searchTool)
	if err != nil {
		t.Fatalf("Failed to register search tool: %v", err)
	}

	commissionManager := &commission.Manager{}

	// Create worker agent with tools
	workerAgent := agent.NewWorkerAgent(
		"tool-exec-agent",
		"Tool Execution Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		toolRegistry,
		commissionManager,
	)

	// Verify tools are available
	registry := workerAgent.GetToolRegistry()
	
	calcToolRetrieved, exists := registry.GetTool("calculator")
	if !exists {
		t.Fatal("Calculator tool should exist in registry")
	}
	if calcToolRetrieved.Name() != "calculator" {
		t.Error("Calculator tool not properly registered")
	}

	searchToolRetrieved, exists := registry.GetTool("search")
	if !exists {
		t.Fatal("Search tool should exist in registry")
	}
	if searchToolRetrieved.Name() != "search" {
		t.Error("Search tool not properly registered")
	}

	// List all tools
	allTools := registry.ListTools()
	if len(allTools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(allTools))
	}

	// Test execution with tools available
	ctx := context.Background()
	response, err := workerAgent.Execute(ctx, "use tools for calculation")
	if err != nil {
		t.Fatalf("Execute with tools failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}
}

// TestWorkerAgentIntegration tests WorkerAgent with all components integrated
func TestWorkerAgentIntegration(t *testing.T) {
	// Create fully configured LLM client
	llmClient := mocks.NewMockLLMClient()
	llmClient.WithResponse("Integrated response")

	// Create memory manager with context
	memoryManager := mocks.NewMockChainManager()
	memoryManager.WithBuildContext([]memory.Message{
		{
			Role:    "system",
			Content: "You are an efficient task executor",
		},
	})

	// Create tool registry with tools
	toolRegistry := tools.NewToolRegistry()
	err := toolRegistry.RegisterTool(toolmocks.NewMockTool("executor", "Executes tasks"))
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	commissionManager := &commission.Manager{}

	// Create fully integrated worker agent
	workerAgent := agent.NewWorkerAgent(
		"integrated-agent",
		"Integrated Test Agent",
		llmClient,
		memoryManager,
		toolRegistry,
		commissionManager,
	)

	// Execute a complex task
	ctx := context.Background()
	response, err := workerAgent.Execute(ctx, "Execute complex integration test")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify response
	if response == "" {
		t.Error("Expected non-empty response")
	}

	// Verify all components are accessible
	if workerAgent.GetLLMClient() == nil {
		t.Error("LLM client should not be nil")
	}
	if workerAgent.GetMemoryManager() == nil {
		t.Error("Memory manager should not be nil")
	}
	if workerAgent.GetToolRegistry() == nil {
		t.Error("Tool registry should not be nil")
	}
	if workerAgent.GetCommissionManager() == nil {
		t.Error("Commission manager should not be nil")
	}
}

// TestWorkerAgentErrorHandling tests error handling scenarios
func TestWorkerAgentErrorHandling(t *testing.T) {
	t.Run("llm_client_error", func(t *testing.T) {
		// Create mock client that returns an error
		llmClient := mocks.NewMockLLMClient()
		llmClient.WithError("LLM service unavailable")

		commissionManager := &commission.Manager{}
		workerAgent := agent.NewWorkerAgent(
			"error-test-agent",
			"Error Test Agent",
			llmClient,
			mocks.NewMockChainManager(),
			tools.NewToolRegistry(),
			commissionManager,
		)

		ctx := context.Background()
		_, err := workerAgent.Execute(ctx, "test request")
		if err == nil {
			t.Error("Expected error when LLM client fails")
		}

		if !strings.Contains(err.Error(), "LLM service unavailable") {
			t.Errorf("Expected LLM error message, got: %v", err)
		}
	})

	t.Run("nil_dependencies", func(t *testing.T) {
		// Test behavior with nil dependencies (should not panic)
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Worker agent creation should not panic with nil dependencies: %v", r)
			}
		}()

		// This might not work depending on the implementation,
		// but shouldn't cause a panic
		workerAgent := agent.NewWorkerAgent(
			"nil-test-agent",
			"Nil Test Agent",
			nil, // nil LLM client
			nil, // nil memory manager
			nil, // nil tool registry
			nil, // nil commission manager
		)

		// Basic property access should work even with nil dependencies
		if workerAgent.GetID() != "nil-test-agent" {
			t.Error("Agent ID should be accessible even with nil dependencies")
		}
	})
}

// TestWorkerAgentConcurrency tests concurrent access to worker agent
func TestWorkerAgentConcurrency(t *testing.T) {
	commissionManager := &commission.Manager{}
	workerAgent := agent.NewWorkerAgent(
		"concurrent-agent",
		"Concurrent Test Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		commissionManager,
	)

	// Test concurrent execution
	ctx := context.Background()
	numGoroutines := 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			response, err := workerAgent.Execute(ctx, "concurrent request")
			if err != nil {
				errors <- err
				return
			}
			
			if response == "" {
				errors <- testing.Errorf("Empty response from goroutine %d", id)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case err := <-errors:
			t.Errorf("Concurrent execution error: %v", err)
		}
	}
}

// BenchmarkWorkerAgentExecution benchmarks worker agent execution
func BenchmarkWorkerAgentExecution(b *testing.B) {
	commissionManager := &commission.Manager{}
	workerAgent := agent.NewWorkerAgent(
		"benchmark-agent",
		"Benchmark Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		commissionManager,
	)

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := workerAgent.Execute(ctx, "benchmark request")
		if err != nil {
			b.Fatalf("Benchmark execution failed: %v", err)
		}
	}
}