package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/agent/mocks"
	"github.com/blockhead-consulting/guild/pkg/kanban"
	"github.com/blockhead-consulting/guild/tools"
)

// TestBaseAgentImplementation tests that BaseAgent implements the Agent interface
func TestBaseAgentImplementation(t *testing.T) {
	var _ agent.Agent = &agent.BaseAgent{}
}

// setupBaseAgent creates a base agent for testing
func setupBaseAgent(t *testing.T) (*agent.BaseAgent, *mocks.MockLLMClient, *mocks.MockChainManager, *mocks.MockToolRegistry, *mocks.MockObjectiveManager) {
	llmClient := mocks.NewMockLLMClient()
	chainManager := mocks.NewMockChainManager()
	toolRegistry := mocks.NewMockToolRegistry()
	objectiveManager := mocks.NewMockObjectiveManager()

	// Add a tool to the registry
	tool := mocks.NewMockTool("test-tool", "A tool for testing")
	toolRegistry.WithTool(tool)

	config := &agent.AgentConfig{
		ID:          "test-agent",
		Name:        "Test Agent",
		Description: "An agent for testing",
		Type:        "worker",
		Provider:    "openai",
		Model:       "gpt-4",
		MaxTokens:   4096,
		Temperature: 0.7,
		Tools:       []string{"test-tool"},
		Metadata:    map[string]string{"key": "value"},
	}

	baseAgent := agent.NewBaseAgent(config, llmClient, chainManager, toolRegistry, objectiveManager)

	return baseAgent, llmClient, chainManager, toolRegistry, objectiveManager
}

// TestNewBaseAgent tests creating a new base agent
func TestNewBaseAgent(t *testing.T) {
	baseAgent, _, _, _, _ := setupBaseAgent(t)

	// Check agent properties
	if baseAgent.ID() != "test-agent" {
		t.Errorf("Expected agent ID 'test-agent', got '%s'", baseAgent.ID())
	}

	if baseAgent.Name() != "Test Agent" {
		t.Errorf("Expected agent name 'Test Agent', got '%s'", baseAgent.Name())
	}

	if baseAgent.Type() != "worker" {
		t.Errorf("Expected agent type 'worker', got '%s'", baseAgent.Type())
	}

	if baseAgent.Status() != agent.StatusIdle {
		t.Errorf("Expected agent status '%s', got '%s'", agent.StatusIdle, baseAgent.Status())
	}

	// Check config
	config := baseAgent.GetConfig()
	if config.ID != "test-agent" {
		t.Errorf("Expected config ID 'test-agent', got '%s'", config.ID)
	}

	// Check state
	state := baseAgent.GetState()
	if state.Status != agent.StatusIdle {
		t.Errorf("Expected state status '%s', got '%s'", agent.StatusIdle, state.Status)
	}

	if state.UpdatedAt.IsZero() {
		t.Error("Expected non-zero updated time")
	}
}

// TestBaseAgentAssignTask tests assigning a task to the agent
func TestBaseAgentAssignTask(t *testing.T) {
	baseAgent, _, _, _, _ := setupBaseAgent(t)
	ctx := context.Background()

	// Create a task
	task := &kanban.Task{
		ID:          "test-task",
		Title:       "Test Task",
		Description: "A task for testing",
		Status:      kanban.StatusTodo,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Assign the task
	err := baseAgent.AssignTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Check agent status
	if baseAgent.Status() != agent.StatusWorking {
		t.Errorf("Expected agent status '%s', got '%s'", agent.StatusWorking, baseAgent.Status())
	}

	// Check state
	state := baseAgent.GetState()
	if state.CurrentTask != "test-task" {
		t.Errorf("Expected current task 'test-task', got '%s'", state.CurrentTask)
	}

	if state.StartedAt.IsZero() {
		t.Error("Expected non-zero started time")
	}

	// Try to assign another task while the agent is busy
	task2 := &kanban.Task{
		ID:          "test-task-2",
		Title:       "Test Task 2",
		Description: "Another task for testing",
		Status:      kanban.StatusTodo,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	err = baseAgent.AssignTask(ctx, task2)
	if err != agent.ErrAgentBusy {
		t.Errorf("Expected error '%v', got '%v'", agent.ErrAgentBusy, err)
	}
}

// TestBaseAgentGetAvailableTools tests getting available tools
func TestBaseAgentGetAvailableTools(t *testing.T) {
	baseAgent, _, _, toolRegistry, _ := setupBaseAgent(t)

	// Add another tool to the registry
	anotherTool := mocks.NewMockTool("another-tool", "Another tool for testing")
	toolRegistry.WithTool(anotherTool)

	// Get available tools
	tools := baseAgent.GetAvailableTools()

	// Should only have the configured tool
	if len(tools) != 1 {
		t.Errorf("Expected 1 available tool, got %d", len(tools))
	}

	if len(tools) > 0 && tools[0].Name() != "test-tool" {
		t.Errorf("Expected tool 'test-tool', got '%s'", tools[0].Name())
	}

	// Create a new agent with no tools specified
	config := &agent.AgentConfig{
		ID:          "test-agent-2",
		Name:        "Test Agent 2",
		Description: "Another agent for testing",
		Type:        "worker",
		Provider:    "openai",
		Model:       "gpt-4",
		Tools:       []string{}, // Empty tools list
	}

	llmClient := mocks.NewMockLLMClient()
	chainManager := mocks.NewMockChainManager()
	objectiveManager := mocks.NewMockObjectiveManager()

	baseAgent2 := agent.NewBaseAgent(config, llmClient, chainManager, toolRegistry, objectiveManager)

	// Get available tools for agent with no tools specified
	tools2 := baseAgent2.GetAvailableTools()

	// Should have all tools in the registry
	if len(tools2) != 2 {
		t.Errorf("Expected 2 available tools, got %d", len(tools2))
	}
}

// TestBaseAgentReset tests resetting the agent
func TestBaseAgentReset(t *testing.T) {
	baseAgent, _, _, _, _ := setupBaseAgent(t)
	ctx := context.Background()

	// Create a task
	task := &kanban.Task{
		ID:          "test-task",
		Title:       "Test Task",
		Description: "A task for testing",
		Status:      kanban.StatusTodo,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Assign the task
	err := baseAgent.AssignTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Reset the agent
	err = baseAgent.Reset(ctx)
	if err != nil {
		t.Fatalf("Failed to reset agent: %v", err)
	}

	// Check agent status
	if baseAgent.Status() != agent.StatusIdle {
		t.Errorf("Expected agent status '%s', got '%s'", agent.StatusIdle, baseAgent.Status())
	}

	// Check state
	state := baseAgent.GetState()
	if state.CurrentTask != "" {
		t.Errorf("Expected empty current task, got '%s'", state.CurrentTask)
	}
}

// TestAgentError tests the AgentError type
func TestAgentError(t *testing.T) {
	err := agent.AgentError{Message: "test error"}
	
	if err.Error() != "test error" {
		t.Errorf("Expected error message 'test error', got '%s'", err.Error())
	}
	
	if agent.ErrAgentBusy.Error() != "agent is busy with another task" {
		t.Errorf("Expected error message 'agent is busy with another task', got '%s'", agent.ErrAgentBusy.Error())
	}
}

// TestBaseAgentGetMemoryManager tests getting the memory manager
func TestBaseAgentGetMemoryManager(t *testing.T) {
	baseAgent, _, chainManager, _, _ := setupBaseAgent(t)
	
	memoryManager := baseAgent.GetMemoryManager()
	if memoryManager != chainManager {
		t.Error("Expected memory manager to be the same as chain manager")
	}
}