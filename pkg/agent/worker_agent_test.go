package agent_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/blockhead-consulting/guild/pkg/agent"
	"github.com/blockhead-consulting/guild/pkg/agent/mocks"
	"github.com/blockhead-consulting/guild/pkg/kanban"
	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/blockhead-consulting/guild/tools"
)

// TestWorkerAgentImplementation tests that WorkerAgent implements the Agent interface
func TestWorkerAgentImplementation(t *testing.T) {
	var _ agent.Agent = &agent.WorkerAgent{}
}

// setupWorkerAgent creates a worker agent for testing
func setupWorkerAgent(t *testing.T) (*agent.WorkerAgent, *mocks.MockLLMClient, *mocks.MockChainManager, *mocks.MockToolRegistry, *mocks.MockObjectiveManager) {
	llmClient := mocks.NewMockLLMClient()
	chainManager := mocks.NewMockChainManager()
	toolRegistry := mocks.NewMockToolRegistry()
	objectiveManager := mocks.NewMockObjectiveManager()

	// Add a tool to the registry
	tool := mocks.NewMockTool("test-tool", "A tool for testing")
	tool.WithResult(&tools.ToolResult{
		Success: true,
		Output:  "Tool executed successfully",
	})
	toolRegistry.WithTool(tool)

	config := &agent.AgentConfig{
		ID:          "test-worker",
		Name:        "Test Worker",
		Description: "A worker agent for testing",
		Type:        "worker",
		Provider:    "openai",
		Model:       "gpt-4",
		MaxTokens:   4096,
		Temperature: 0.7,
		Tools:       []string{"test-tool"},
		Metadata:    map[string]string{"key": "value"},
	}

	workerAgent := agent.NewWorkerAgent(config, llmClient, chainManager, toolRegistry, objectiveManager)

	return workerAgent, llmClient, chainManager, toolRegistry, objectiveManager
}

// TestNewWorkerAgent tests creating a new worker agent
func TestNewWorkerAgent(t *testing.T) {
	workerAgent, _, _, _, _ := setupWorkerAgent(t)

	// Check agent properties
	if workerAgent.ID() != "test-worker" {
		t.Errorf("Expected agent ID 'test-worker', got '%s'", workerAgent.ID())
	}

	if workerAgent.Name() != "Test Worker" {
		t.Errorf("Expected agent name 'Test Worker', got '%s'", workerAgent.Name())
	}

	if workerAgent.Type() != "worker" {
		t.Errorf("Expected agent type 'worker', got '%s'", workerAgent.Type())
	}

	if workerAgent.Status() != agent.StatusIdle {
		t.Errorf("Expected agent status '%s', got '%s'", agent.StatusIdle, workerAgent.Status())
	}
}

// TestWorkerAgentExecuteWithoutTask tests executing the agent without a task
func TestWorkerAgentExecuteWithoutTask(t *testing.T) {
	workerAgent, _, _, _, _ := setupWorkerAgent(t)
	ctx := context.Background()

	// Execute without a task
	err := workerAgent.Execute(ctx)
	if err == nil {
		t.Error("Expected error when executing without a task, got nil")
	}

	if err.Error() != "no task assigned" {
		t.Errorf("Expected error 'no task assigned', got '%v'", err)
	}
}

// TestWorkerAgentExecuteWithTask tests executing the agent with a task
func TestWorkerAgentExecuteWithTask(t *testing.T) {
	workerAgent, llmClient, chainManager, _, objectiveManager := setupWorkerAgent(t)
	ctx := context.Background()

	// Create a task
	task := &kanban.Task{
		ID:          "test-task",
		Title:       "Test Task",
		Description: "A task for testing",
		Status:      kanban.StatusTodo,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Metadata:    map[string]string{"objective_id": "test-objective"},
	}

	// Create an objective
	obj := objective.NewObjective("Test Objective", "An objective for testing")
	obj.ID = "test-objective"
	obj.Parts = []objective.Part{
		{
			ID:      "part-1",
			Title:   "Context",
			Type:    "context",
			Content: "This is the context for the objective",
		},
		{
			ID:      "part-2",
			Title:   "Goal",
			Type:    "goal",
			Content: "This is the goal of the objective",
		},
	}
	objectiveManager.WithObjective(obj)

	// Setup LLM response for successful completion
	llmClient.WithDefaultResponse(providers.CompletionResponse{
		Text:         `{"thoughts": "I've completed the task successfully", "final_answer": "The task has been completed"}`,
		TokensUsed:   100,
		TokensInput:  50,
		TokensOutput: 50,
		FinishReason: "stop",
		ModelUsed:    "gpt-4",
	})

	// Assign the task
	err := workerAgent.AssignTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Execute the agent
	err = workerAgent.Execute(ctx)
	if err != nil {
		t.Fatalf("Failed to execute agent: %v", err)
	}

	// Check agent status
	if workerAgent.Status() != agent.StatusIdle {
		t.Errorf("Expected agent status '%s', got '%s'", agent.StatusIdle, workerAgent.Status())
	}

	// Check task status
	if task.Status != "done" {
		t.Errorf("Expected task status 'done', got '%s'", task.Status)
	}

	if task.CompletedAt == nil {
		t.Error("Expected non-nil completed time")
	}

	// Check task metadata
	if task.Metadata["completion_summary"] != "The task has been completed" {
		t.Errorf("Expected completion summary 'The task has been completed', got '%s'", task.Metadata["completion_summary"])
	}
}

// TestWorkerAgentExecuteWithToolAction tests executing the agent with a tool action
func TestWorkerAgentExecuteWithToolAction(t *testing.T) {
	workerAgent, llmClient, chainManager, toolRegistry, _ := setupWorkerAgent(t)
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

	// Setup memory for conversation context
	chainManager.WithBuildContext([]memory.Message{
		{
			Role:      "system",
			Content:   "You are a worker agent.",
			Timestamp: time.Now().UTC(),
		},
	})

	// Setup LLM responses - first use tool, then complete
	toolActionResponse := providers.CompletionResponse{
		Text: `{"thoughts": "I need to use a tool", "action": {"tool": "test-tool", "input": {"param": "value"}}}`,
	}
	completionResponse := providers.CompletionResponse{
		Text: `{"thoughts": "I've completed the task after using the tool", "final_answer": "Tool executed successfully"}`,
	}

	// Make the LLM client return the tool action response first, then the completion response
	llmClient.WithResponse("System: You are a worker agent.", toolActionResponse)

	// Setup tool result
	toolRegistry.WithToolResult(&tools.ToolResult{
		Success: true,
		Output:  "Tool executed successfully",
	})

	// Need to update the LLM client after the tool is used
	// This is a bit hacky but works for the test
	executedTool := false
	oldExecuteTool := toolRegistry.ExecuteToolWithParams
	toolRegistry.ExecuteToolWithParams = func(ctx context.Context, name string, params map[string]interface{}) (*tools.ToolResult, error) {
		result, err := oldExecuteTool(ctx, name, params)
		if err == nil && !executedTool {
			executedTool = true
			llmClient.WithDefaultResponse(completionResponse)
		}
		return result, err
	}

	// Assign the task
	err := workerAgent.AssignTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Execute the agent
	err = workerAgent.Execute(ctx)
	if err != nil {
		t.Fatalf("Failed to execute agent: %v", err)
	}

	// Check agent status
	if workerAgent.Status() != agent.StatusIdle {
		t.Errorf("Expected agent status '%s', got '%s'", agent.StatusIdle, workerAgent.Status())
	}

	// Check task status
	if task.Status != "done" {
		t.Errorf("Expected task status 'done', got '%s'", task.Status)
	}

	// Check task metadata
	if task.Metadata["completion_summary"] != "Tool executed successfully" {
		t.Errorf("Expected completion summary 'Tool executed successfully', got '%s'", task.Metadata["completion_summary"])
	}
}

// TestWorkerAgentExecuteWithError tests executing the agent with errors
func TestWorkerAgentExecuteWithError(t *testing.T) {
	workerAgent, llmClient, _, _, _ := setupWorkerAgent(t)
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

	// Setup LLM to return an error
	llmClient.WithError(errors.New("test error"))

	// Assign the task
	err := workerAgent.AssignTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Execute the agent - should return error after max retries
	err = workerAgent.Execute(ctx)
	if err == nil {
		t.Fatal("Expected error executing agent, got nil")
	}

	// Check agent status
	if workerAgent.Status() != agent.StatusError {
		t.Errorf("Expected agent status '%s', got '%s'", agent.StatusError, workerAgent.Status())
	}

	// Check agent state
	state := workerAgent.GetState()
	if state.LastError == "" {
		t.Error("Expected non-empty last error")
	}
}

// TestWorkerAgentStop tests stopping the agent
func TestWorkerAgentStop(t *testing.T) {
	workerAgent, _, _, _, _ := setupWorkerAgent(t)
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
	err := workerAgent.AssignTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Stop the agent
	err = workerAgent.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop agent: %v", err)
	}

	// Check agent status
	if workerAgent.Status() != agent.StatusPaused {
		t.Errorf("Expected agent status '%s', got '%s'", agent.StatusPaused, workerAgent.Status())
	}
}

// TestParseAgentResponse tests parsing agent responses
func TestParseAgentResponse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantThoughts string
		wantAction   bool
		wantToolName string
		wantFinal    string
		wantQuestion string
		wantError    bool
	}{
		{
			name: "final answer",
			input: `I'm thinking about what to do next.
			{"thoughts": "Task is complete", "final_answer": "The task has been completed"}`,
			wantThoughts: "Task is complete",
			wantFinal:    "The task has been completed",
		},
		{
			name: "tool action",
			input: `{"thoughts": "I need to use a tool", "action": {"tool": "test-tool", "input": {"param": "value"}}}`,
			wantThoughts: "I need to use a tool",
			wantAction:   true,
			wantToolName: "test-tool",
		},
		{
			name: "question",
			input: `{"thoughts": "I need more information", "question": "What should I do next?"}`,
			wantThoughts: "I need more information",
			wantQuestion: "What should I do next?",
		},
		{
			name:      "invalid JSON",
			input:     `This is not valid JSON`,
			wantError: true,
		},
		{
			name:      "incomplete JSON",
			input:     `{"thoughts": "Incomplete JSON"`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't access the parseAgentResponse function directly,
			// so we'll test it through the executeLoop method by mocking
			// the dependencies and checking the behavior.
			workerAgent, llmClient, chainManager, _, _ := setupWorkerAgent(t)
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

			// Set up LLM response
			llmClient.WithDefaultResponse(providers.CompletionResponse{
				Text: tt.input,
			})

			// If expecting a final answer, we should see the task complete
			if tt.wantFinal != "" {
				task.Metadata = make(map[string]string)
			}

			// Assign the task
			err := workerAgent.AssignTask(ctx, task)
			if err != nil {
				t.Fatalf("Failed to assign task: %v", err)
			}

			// Execute the agent - for invalid JSON, it will retry
			if !tt.wantError {
				err = workerAgent.Execute(ctx)
				if err != nil {
					if tt.wantFinal == "" && tt.wantAction == false && tt.wantQuestion == "" {
						// This is expected - other response types will cause the agent to retry
						// until it gets a final answer, which will timeout in this test
					} else {
						t.Fatalf("Failed to execute agent: %v", err)
					}
				}

				// For final answer, check task status
				if tt.wantFinal != "" {
					if task.Status != "done" {
						t.Errorf("Expected task status 'done', got '%s'", task.Status)
					}

					if task.Metadata["completion_summary"] != tt.wantFinal {
						t.Errorf("Expected completion summary '%s', got '%s'", tt.wantFinal, task.Metadata["completion_summary"])
					}
				}
			}
		})
	}
}

// TestBuildPromptFromMessages tests building a prompt from messages
func TestBuildPromptFromMessages(t *testing.T) {
	// Create messages
	messages := []memory.Message{
		{
			Role:      "system",
			Content:   "You are a worker agent.",
			Timestamp: time.Now().UTC(),
		},
		{
			Role:      "user",
			Content:   "Hello, agent!",
			Timestamp: time.Now().UTC(),
		},
		{
			Role:      "assistant",
			Content:   "Hello! How can I help you?",
			Timestamp: time.Now().UTC(),
		},
		{
			Role:      "tool",
			Name:      "test-tool",
			Content:   "Tool result",
			Timestamp: time.Now().UTC(),
		},
	}

	// Since buildPromptFromMessages is internal, we'll test it indirectly
	// by using the worker agent and checking the LLM client calls
	workerAgent, llmClient, chainManager, _, _ := setupWorkerAgent(t)
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

	// Setup context with these messages
	chainManager.WithBuildContext(messages)

	// Setup LLM to return a final answer
	llmClient.WithDefaultResponse(providers.CompletionResponse{
		Text: `{"thoughts": "Task complete", "final_answer": "Done"}`,
	})

	// Assign the task
	err := workerAgent.AssignTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Execute the agent
	err = workerAgent.Execute(ctx)
	if err != nil {
		t.Fatalf("Failed to execute agent: %v", err)
	}

	// Check that the LLM client was called with a prompt containing all message contents
	if len(llmClient.RequestHistory) == 0 {
		t.Fatal("Expected at least one LLM request")
	}

	prompt := llmClient.RequestHistory[0].Prompt
	expectedContents := []string{
		"System: You are a worker agent.",
		"User: Hello, agent!",
		"Assistant: Hello! How can I help you?",
		"Tool test-tool: Tool result",
	}

	for _, expected := range expectedContents {
		if !contains(prompt, expected) {
			t.Errorf("Expected prompt to contain '%s'", expected)
			fmt.Println("Actual prompt:", prompt)
		}
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return s != "" && substr != "" && (s == substr || s[:len(s)] != "" && (substr == s[:len(substr)] || s[len(s)-len(substr):] == substr))
}