package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blockhead-consulting/Guild/pkg/kanban"
	"github.com/blockhead-consulting/Guild/pkg/memory"
	"github.com/blockhead-consulting/Guild/pkg/objective"
	"github.com/blockhead-consulting/Guild/pkg/providers"
	"github.com/blockhead-consulting/Guild/tools"
)

const (
	// Default prompt template for worker agents
	workerAgentPrompt = `You are {{.AgentName}}, an autonomous AI agent working in a Guild of coordinated agents.

Your current task is:

Title: {{.TaskTitle}}
Description: {{.TaskDescription}}

Additional Context:
{{.Context}}

You have the following tools available:
{{.ToolDescriptions}}

To use a tool, respond with a JSON message in this format:
{
  "thoughts": "your step-by-step reasoning about what to do next",
  "action": {
    "tool": "tool_name",
    "input": {
      // tool-specific parameters
    }
  }
}

If you believe the task is complete, respond with:
{
  "thoughts": "why you believe the task is complete",
  "final_answer": "comprehensive summary of what you accomplished"
}

If you need more information or are stuck, respond with:
{
  "thoughts": "what information you need or what obstacle you're facing",
  "question": "specific question to help you proceed"
}

Think step-by-step and break down complex tasks. Always provide thorough thoughts to explain your reasoning.
`
)

// WorkerAgent implements a worker agent that executes specific tasks
type WorkerAgent struct {
	*BaseAgent
	maxConsecutiveErrors int
	errorCount           int
}

// AgentResponse represents a structured response from the agent
type AgentResponse struct {
	Thoughts    string          `json:"thoughts"`
	Action      *AgentAction    `json:"action,omitempty"`
	FinalAnswer string          `json:"final_answer,omitempty"`
	Question    string          `json:"question,omitempty"`
}

// AgentAction represents a tool action the agent wants to take
type AgentAction struct {
	Tool  string                 `json:"tool"`
	Input map[string]interface{} `json:"input"`
}

// NewWorkerAgent creates a new worker agent
func NewWorkerAgent(
	config *AgentConfig,
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveMgr *objective.Manager,
) *WorkerAgent {
	baseAgent := NewBaseAgent(config, llmClient, memoryManager, toolRegistry, objectiveMgr)
	
	return &WorkerAgent{
		BaseAgent:            baseAgent,
		maxConsecutiveErrors: 3, // Default max errors before giving up
	}
}

// Execute runs the worker agent's execution cycle
func (a *WorkerAgent) Execute(ctx context.Context) error {
	// Check if the agent has a task
	if a.currentTask == nil {
		return fmt.Errorf("no task assigned")
	}
	
	// Update status
	a.state.Status = StatusWorking
	a.state.UpdatedAt = time.Now().UTC()
	
	// Create a memory chain for this execution if needed
	var chainID string
	var err error
	
	if len(a.state.Memory) == 0 {
		// No existing memory chain, create one
		chainID, err = a.memoryManager.CreateChain(ctx, a.config.ID, a.currentTask.ID)
		if err != nil {
			return fmt.Errorf("failed to create memory chain: %w", err)
		}
		a.state.Memory = append(a.state.Memory, chainID)
	} else {
		// Use the last memory chain
		chainID = a.state.Memory[len(a.state.Memory)-1]
	}
	
	// Prepare context for the prompt
	promptContext, err := a.buildPromptContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to build prompt context: %w", err)
	}
	
	// Add task history to prompt context if available
	taskHistory, err := a.getTaskHistory(ctx)
	if err == nil && taskHistory != "" {
		promptContext += "\n\nTask History:\n" + taskHistory
	}
	
	// Build the prompt
	prompt := a.buildPrompt(promptContext)
	
	// Add the prompt to the memory chain
	err = a.memoryManager.AddMessage(ctx, chainID, memory.Message{
		Role:      "system",
		Content:   prompt,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("failed to add prompt to memory: %w", err)
	}
	
	// Execute the agent loop
	return a.executeLoop(ctx, chainID)
}

// buildPrompt builds the agent's prompt
func (a *WorkerAgent) buildPrompt(context string) string {
	// In a real implementation, you would use a proper template engine
	// For simplicity, we'll do simple replacements
	prompt := workerAgentPrompt
	
	// Replace placeholders
	prompt = strings.Replace(prompt, "{{.AgentName}}", a.config.Name, -1)
	prompt = strings.Replace(prompt, "{{.TaskTitle}}", a.currentTask.Title, -1)
	prompt = strings.Replace(prompt, "{{.TaskDescription}}", a.currentTask.Description, -1)
	prompt = strings.Replace(prompt, "{{.Context}}", context, -1)
	
	// Build tool descriptions
	var toolDescriptions strings.Builder
	for _, tool := range a.GetAvailableTools() {
		toolDescriptions.WriteString("- " + tool.Name() + ": " + tool.Description() + "\n")
	}
	prompt = strings.Replace(prompt, "{{.ToolDescriptions}}", toolDescriptions.String(), -1)
	
	return prompt
}

// buildPromptContext builds the context for the agent's prompt
func (a *WorkerAgent) buildPromptContext(ctx context.Context) (string, error) {
	var contextBuilder strings.Builder
	
	// Add relevant metadata from the task
	for key, value := range a.currentTask.Metadata {
		contextBuilder.WriteString(key + ": " + value + "\n")
	}
	
	// Add objective context if this task is part of an objective
	if objectiveID, ok := a.currentTask.Metadata["objective_id"]; ok {
		obj, err := a.objectiveMgr.GetObjective(ctx, objectiveID)
		if err == nil {
			contextBuilder.WriteString("\nObjective: " + obj.Title + "\n")
			contextBuilder.WriteString("Description: " + obj.Description + "\n")
			
			// Add relevant parts from the objective
			for _, part := range obj.Parts {
				if part.Type == "context" || part.Type == "goal" {
					contextBuilder.WriteString("\n## " + part.Title + "\n")
					contextBuilder.WriteString(part.Content + "\n")
				}
			}
		}
	}
	
	return contextBuilder.String(), nil
}

// getTaskHistory retrieves the task's history
func (a *WorkerAgent) getTaskHistory(ctx context.Context) (string, error) {
	if a.currentTask == nil {
		return "", fmt.Errorf("no task assigned")
	}
	
	// Check if there's a task history in the metadata
	if history, ok := a.currentTask.Metadata["history"]; ok {
		return history, nil
	}
	
	return "", nil
}

// executeLoop executes the agent's main loop
func (a *WorkerAgent) executeLoop(ctx context.Context, chainID string) error {
	for {
		// Check if the context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue execution
		}
		
		// Get the most recent messages as context
		messages, err := a.memoryManager.BuildContext(ctx, a.config.ID, a.currentTask.ID, a.config.MaxTokens/2)
		if err != nil {
			return fmt.Errorf("failed to build context: %w", err)
		}
		
		// Create a completion request
		req := &providers.CompletionRequest{
			Prompt:      buildPromptFromMessages(messages),
			MaxTokens:   a.config.MaxTokens,
			Temperature: a.config.Temperature,
		}
		
		// Call the LLM
		resp, err := a.llmClient.Complete(ctx, req)
		if err != nil {
			a.errorCount++
			if a.errorCount >= a.maxConsecutiveErrors {
				a.state.Status = StatusError
				a.state.LastError = fmt.Sprintf("Too many consecutive errors: %v", err)
				a.SaveState(ctx)
				return fmt.Errorf("too many consecutive errors: %w", err)
			}
			
			// Log error and continue
			a.state.LastError = err.Error()
			a.SaveState(ctx)
			continue
		}
		
		// Reset error count
		a.errorCount = 0
		
		// Add the response to memory
		err = a.memoryManager.AddMessage(ctx, chainID, memory.Message{
			Role:       "assistant",
			Content:    resp.Text,
			Timestamp:  time.Now().UTC(),
			TokenUsage: resp.TokensUsed,
		})
		if err != nil {
			return fmt.Errorf("failed to add response to memory: %w", err)
		}
		
		// Parse the response
		agentResp, err := parseAgentResponse(resp.Text)
		if err != nil {
			// Add error message to memory
			errorMsg := fmt.Sprintf("Failed to parse response: %v\nPlease respond with valid JSON in the required format.", err)
			a.memoryManager.AddMessage(ctx, chainID, memory.Message{
				Role:      "system",
				Content:   errorMsg,
				Timestamp: time.Now().UTC(),
			})
			continue
		}
		
		// Check if the agent has a final answer
		if agentResp.FinalAnswer != "" {
			// Task completed
			a.completeTask(ctx, agentResp.FinalAnswer)
			return nil
		}
		
		// Check if the agent has a question
		if agentResp.Question != "" {
			// Add the question to memory
			a.memoryManager.AddMessage(ctx, chainID, memory.Message{
				Role:      "system",
				Content:   "You asked: " + agentResp.Question + "\nPlease try to proceed with the information you have or use available tools to find what you need.",
				Timestamp: time.Now().UTC(),
			})
			continue
		}
		
		// Check if the agent wants to use a tool
		if agentResp.Action != nil {
			toolResult, err := a.executeTool(ctx, agentResp.Action)
			if err != nil {
				// Add error message to memory
				errorMsg := fmt.Sprintf("Error executing tool: %v", err)
				a.memoryManager.AddMessage(ctx, chainID, memory.Message{
					Role:      "system",
					Content:   errorMsg,
					Timestamp: time.Now().UTC(),
				})
				continue
			}
			
			// Add tool result to memory
			a.memoryManager.AddMessage(ctx, chainID, memory.Message{
				Role:      "tool",
				Name:      agentResp.Action.Tool,
				Content:   toolResult,
				Timestamp: time.Now().UTC(),
			})
		}
	}
}

// executeTool executes a tool and returns the result
func (a *WorkerAgent) executeTool(ctx context.Context, action *AgentAction) (string, error) {
	if action.Tool == "" {
		return "", fmt.Errorf("tool name is required")
	}
	
	// Execute the tool
	result, err := a.toolRegistry.ExecuteToolWithParams(ctx, action.Tool, action.Input)
	if err != nil {
		return "", fmt.Errorf("failed to execute tool %s: %w", action.Tool, err)
	}
	
	if !result.Success {
		return "", fmt.Errorf("tool execution failed: %s", result.Error)
	}
	
	return result.Output, nil
}

// completeTask marks the task as complete
func (a *WorkerAgent) completeTask(ctx context.Context, summary string) {
	// Update the task status
	a.currentTask.Status = "done"
	a.currentTask.UpdatedAt = time.Now().UTC()
	now := time.Now().UTC()
	a.currentTask.CompletedAt = &now
	
	// Add the summary to the task metadata
	if a.currentTask.Metadata == nil {
		a.currentTask.Metadata = make(map[string]string)
	}
	a.currentTask.Metadata["completion_summary"] = summary
	
	// Update agent state
	a.state.Status = StatusIdle
	a.state.CurrentTask = ""
	a.state.UpdatedAt = time.Now().UTC()
	
	// Save agent state
	a.SaveState(ctx)
}

// Stop stops the agent's execution
func (a *WorkerAgent) Stop(ctx context.Context) error {
	// Implementation depends on execution model
	// For simplicity, just update the state
	a.state.Status = StatusPaused
	a.state.UpdatedAt = time.Now().UTC()
	return a.SaveState(ctx)
}

// parseAgentResponse parses the agent's response
func parseAgentResponse(text string) (*AgentResponse, error) {
	// Find the JSON block in the response
	jsonStart := strings.Index(text, "{")
	jsonEnd := strings.LastIndex(text, "}")
	
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return nil, fmt.Errorf("no valid JSON found in response")
	}
	
	jsonStr := text[jsonStart : jsonEnd+1]
	
	var resp AgentResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return &resp, nil
}

// buildPromptFromMessages builds a prompt from a list of messages
func buildPromptFromMessages(messages []memory.Message) string {
	var prompt strings.Builder
	
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			prompt.WriteString("System: " + msg.Content + "\n\n")
		case "user":
			prompt.WriteString("User: " + msg.Content + "\n\n")
		case "assistant":
			prompt.WriteString("Assistant: " + msg.Content + "\n\n")
		case "tool":
			prompt.WriteString("Tool " + msg.Name + ": " + msg.Content + "\n\n")
		}
	}
	
	return prompt.String()
}