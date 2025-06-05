package agent_test

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/agent/mocks"
	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
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
	
	// Create temporary store for objective manager
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
		"test-agent-1",
		"Test Agent",
		llmClient,
		memoryManager,
		toolRegistry,
		objectiveManager,
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

	if workerAgent.GetCommissionManager() != objectiveManager {
		t.Error("Objective manager not properly set")
	}

	// Verify cost manager is initialized
	if workerAgent.CostManager == nil {
		t.Error("Cost manager should be initialized")
	}
}

// TestWorkerAgentExecute tests the Execute method
func TestWorkerAgentExecute(t *testing.T) {
	// Create mock dependencies
	llmClient := mocks.NewMockLLMClient()
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()
	
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
		"test-agent-2",
		"Test Worker",
		llmClient,
		memoryManager,
		toolRegistry,
		objectiveManager,
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

// TestWorkerAgentCostManagement tests cost budget and reporting
func TestWorkerAgentCostManagement(t *testing.T) {
	// Create worker agent with minimal dependencies
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
	
	workerAgent := agent.NewWorkerAgent(
		"test-agent-3",
		"Cost Test Agent",
		&mocks.MockLLMClient{},
		mocks.NewMockChainManager(),
		tools.NewToolRegistry(),
		objectiveManager,
	)

	// Set cost budget using the correct CostType constants
	workerAgent.SetCostBudget(agent.CostTypeLLM, 1000.0)
	workerAgent.SetCostBudget(agent.CostTypeTool, 50.0)

	// Get cost report
	report := workerAgent.GetCostReport()
	if report == nil {
		t.Fatal("Cost report should not be nil")
	}

	// Verify report contains budget info
	if budgets, ok := report["budgets"].(map[string]float64); ok {
		if budgets[string(agent.CostTypeLLM)] != 1000.0 {
			t.Errorf("Expected LLM budget 1000.0, got %f", budgets[string(agent.CostTypeLLM)])
		}
		if budgets[string(agent.CostTypeTool)] != 50.0 {
			t.Errorf("Expected Tool budget 50.0, got %f", budgets[string(agent.CostTypeTool)])
		}
	} else {
		t.Error("Cost report should contain budgets")
	}
}

// TestManagerAgentCreation tests creating a manager agent
func TestManagerAgentCreation(t *testing.T) {
	// Create mock dependencies
	llmClient := mocks.NewMockLLMClient()
	memoryManager := mocks.NewMockChainManager()
	toolRegistry := tools.NewToolRegistry()
	
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
		"test-manager-1",
		"Test Manager",
		llmClient,
		memoryManager,
		toolRegistry,
		objectiveManager,
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
		"test-agent-4",
		"LLM Test Agent",
		llmClient,
		memoryManager,
		tools.NewToolRegistry(),
		objectiveManager,
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
		"test-agent-5",
		"Tool Test Agent",
		&mocks.MockLLMClient{},
		mocks.NewMockChainManager(),
		toolRegistry,
		objectiveManager,
	)

	// Verify tool registry is accessible
	registry := workerAgent.GetToolRegistry()
	if registry == nil {
		t.Fatal("Tool registry should not be nil")
	}

	// Verify the tool is in the registry (using the correct method name)
	tool, exists := registry.GetTool("test-tool")
	if !exists {
		t.Fatal("Tool should exist in registry")
	}

	if tool.Name() != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tool.Name())
	}
}