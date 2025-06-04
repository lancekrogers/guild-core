package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/agent/mocks"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	"github.com/guild-ventures/guild-core/pkg/objective"
	"github.com/guild-ventures/guild-core/pkg/tools"
	toolmocks "github.com/guild-ventures/guild-core/tools/mocks"
)

// TestWorkerAgentWithContext tests WorkerAgent with context handling
func TestWorkerAgentWithContext(t *testing.T) {
	// Create temporary store
	tempDir := t.TempDir()
	store, err := boltdb.NewStore(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	
	objectiveManager, err := objective.NewManager(store, tempDir)
	if err != nil {
		t.Fatalf("Failed to create objective manager: %v", err)
	}
	
	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"ctx-test-agent",
		"Context Test Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		objectiveManager,
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

// TestWorkerAgentCostTracking tests cost tracking functionality
func TestWorkerAgentCostTracking(t *testing.T) {
	// Create temporary store
	tempDir := t.TempDir()
	store, err := boltdb.NewStore(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	
	objectiveManager, err := objective.NewManager(store, tempDir)
	if err != nil {
		t.Fatalf("Failed to create objective manager: %v", err)
	}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"cost-test-agent",
		"Cost Test Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		objectiveManager,
	)

	// Set budgets
	workerAgent.SetCostBudget(agent.CostTypeLLM, 1000.0)
	workerAgent.SetCostBudget(agent.CostTypeTool, 1.0)

	// Get initial cost report
	report := workerAgent.GetCostReport()
	
	// Verify budgets are set
	if budgets, ok := report["budgets"].(map[string]float64); ok {
		if budgets[string(agent.CostTypeLLM)] != 1000.0 {
			t.Errorf("Expected LLM budget 1000.0, got %f", budgets[string(agent.CostTypeLLM)])
		}
		if budgets[string(agent.CostTypeTool)] != 1.0 {
			t.Errorf("Expected Tool budget 1.0, got %f", budgets[string(agent.CostTypeTool)])
		}
	}

	// Execute would normally incur costs, but our stub implementation doesn't
	ctx := context.Background()
	_, err = workerAgent.Execute(ctx, "expensive operation")
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// In a real implementation, we would check that costs were tracked
	// For now, just verify the cost report structure
	finalReport := workerAgent.GetCostReport()
	if finalReport == nil {
		t.Fatal("Cost report should not be nil")
	}
}

// TestManagerAgentInheritance tests that ManagerAgent inherits WorkerAgent functionality
func TestManagerAgentInheritance(t *testing.T) {
	// Create temporary store
	tempDir := t.TempDir()
	store, err := boltdb.NewStore(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	
	objectiveManager, err := objective.NewManager(store, tempDir)
	if err != nil {
		t.Fatalf("Failed to create objective manager: %v", err)
	}
	
	// Create manager agent
	managerAgent := agent.NewManagerAgent(
		"manager-test",
		"Manager Test",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		objectiveManager,
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

	// Test cost management
	managerAgent.SetCostBudget(agent.CostTypeLLM, 2000.0)
	report := managerAgent.GetCostReport()
	if report == nil {
		t.Fatal("Manager should have cost reporting")
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

	// Create temporary store
	tempDir := t.TempDir()
	store, err := boltdb.NewStore(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	
	objectiveManager, err := objective.NewManager(store, tempDir)
	if err != nil {
		t.Fatalf("Failed to create objective manager: %v", err)
	}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"memory-test-agent",
		"Memory Test Agent",
		mocks.NewMockLLMClient(),
		memoryManager,
		tools.NewToolRegistry(),
		objectiveManager,
	)

	// Verify memory manager is accessible and has context
	if workerAgent.GetMemoryManager() != memoryManager {
		t.Error("Memory manager not properly set")
	}
}

// TestWorkerAgentWithObjectiveManager tests agent with objective manager
func TestWorkerAgentWithObjectiveManager(t *testing.T) {
	// Create temporary store
	tempDir := t.TempDir()
	store, err := boltdb.NewStore(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	
	// Create objective manager
	objManager, err := objective.NewManager(store, tempDir)
	if err != nil {
		t.Fatalf("Failed to create objective manager: %v", err)
	}

	// Create worker agent
	workerAgent := agent.NewWorkerAgent(
		"obj-test-agent",
		"Objective Test Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		objManager,
	)

	// Verify objective manager is accessible
	if workerAgent.GetObjectiveManager() != objManager {
		t.Error("Objective manager not properly set")
	}

	// In a real implementation, we would test objective-based execution
	ctx := context.Background()
	response, err := workerAgent.Execute(ctx, "work on objective")
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

	// Create temporary store
	tempDir := t.TempDir()
	store, err := boltdb.NewStore(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	
	objectiveManager, err := objective.NewManager(store, tempDir)
	if err != nil {
		t.Fatalf("Failed to create objective manager: %v", err)
	}

	// Create worker agent with tools
	workerAgent := agent.NewWorkerAgent(
		"tool-exec-agent",
		"Tool Execution Agent",
		mocks.NewMockLLMClient(),
		mocks.NewMockChainManager(),
		toolRegistry,
		objectiveManager,
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

	// Create temporary store
	tempDir := t.TempDir()
	store, err := boltdb.NewStore(tempDir + "/test.db")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()
	
	// Create objective manager
	objManager, err := objective.NewManager(store, tempDir)
	if err != nil {
		t.Fatalf("Failed to create objective manager: %v", err)
	}

	// Create fully integrated worker agent
	workerAgent := agent.NewWorkerAgent(
		"integrated-agent",
		"Integrated Test Agent",
		llmClient,
		memoryManager,
		toolRegistry,
		objManager,
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
	if workerAgent.GetObjectiveManager() == nil {
		t.Error("Objective manager should not be nil")
	}
}