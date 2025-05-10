package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/blockhead-consulting/guild/pkg/kanban"
	"github.com/blockhead-consulting/guild/pkg/memory"
	"github.com/blockhead-consulting/guild/pkg/objective"
	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/blockhead-consulting/guild/tools"
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

// Craftsman implements a worker artisan that executes specific tasks
type Craftsman struct {
	*GuildMember
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

// NewCraftsman creates a new craftsman artisan
func NewCraftsman(
	config *AgentConfig,
	llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry *tools.ToolRegistry,
	objectiveMgr *objective.Manager,
) *Craftsman {
	member := NewGuildMember(config, llmClient, memoryManager, toolRegistry, objectiveMgr)

	return &Craftsman{
		GuildMember:          member,
		maxConsecutiveErrors: 3, // Default max errors before giving up
	}
}

// CraftSolution runs the craftsman's execution cycle
func (a *Craftsman) CraftSolution(ctx context.Context) error {
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

	// Add cost-awareness if budget is limited
	llmBudget := a.costManager.GetBudget(CostTypeLLM)
	toolBudget := a.costManager.GetBudget(CostTypeTool)
	if llmBudget > 0 || toolBudget > 0 {
		costContext := "\n\n## Cost Awareness\n"
		costContext += "You must optimize for cost efficiency while completing this task.\n"

		if llmBudget > 0 {
			llmCost := a.costManager.GetTotalCost(CostTypeLLM)
			costContext += fmt.Sprintf("LLM Budget: $%.4f (Used: $%.4f, Remaining: $%.4f)\n",
				llmBudget, llmCost, llmBudget - llmCost)
		}

		if toolBudget > 0 {
			toolCost := a.costManager.GetTotalCost(CostTypeTool)
			costContext += fmt.Sprintf("Tool Budget: $%.4f (Used: $%.4f, Remaining: $%.4f)\n",
				toolBudget, toolCost, toolBudget - toolCost)
		}

		costContext += "\nCost-saving strategies:\n"
		costContext += "1. Break complex tasks into smaller steps to avoid long completions\n"
		costContext += "2. Use tools and memory efficiently to avoid redundant LLM calls\n"
		costContext += "3. Be concise in your reasoning to minimize token usage\n"

		promptContext += costContext
	}

	// Build the prompt
	prompt := a.buildPrompt(promptContext)

	// Estimate prompt tokens for cost tracking
	promptTokens := estimateTokens(prompt)

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
	return a.executeLoop(ctx, chainID, promptTokens)
}

// buildPrompt builds the agent's prompt
func (a *Craftsman) buildPrompt(context string) string {
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
func (a *Craftsman) buildPromptContext(ctx context.Context) (string, error) {
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
func (a *Craftsman) getTaskHistory(ctx context.Context) (string, error) {
	if a.currentTask == nil {
		return "", fmt.Errorf("no task assigned")
	}
	
	// Check if there's a task history in the metadata
	if history, ok := a.currentTask.Metadata["history"]; ok {
		return history, nil
	}
	
	return "", nil
}

// estimateTokens provides a simple estimation of token count from text length
// This is a very rough approximation - in a real system, you'd use a proper tokenizer
func estimateTokens(text string) int {
	// Rough approximation: 1 token ≈ 4 characters for English text
	return len(text) / 4
}

// executeLoop executes the agent's main loop
func (a *Craftsman) executeLoop(ctx context.Context, chainID string, promptTokens int) error {
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

		// Estimate LLM cost before making the API call
		estimatedPromptTokens := estimateTokens(req.Prompt)
		estimatedMaxCost := a.costManager.EstimateLLMCost(a.config.Model, estimatedPromptTokens, a.config.MaxTokens)

		// Check if we're within budget before making the API call
		if !a.costManager.CanAfford(CostTypeLLM, estimatedMaxCost) {
			// Add budget warning to memory
			budgetMsg := "LLM budget exceeded. The agent will attempt to complete the task with reduced API calls."
			a.memoryManager.AddMessage(ctx, chainID, memory.Message{
				Role:      "system",
				Content:   budgetMsg,
				Timestamp: time.Now().UTC(),
			})

			// If we're completely out of budget, stop execution
			if a.costManager.GetTotalCost(CostTypeLLM) >= a.costManager.GetBudget(CostTypeLLM) * 1.1 { // Allow 10% overage
				a.state.Status = StatusError
				a.state.LastError = "LLM budget exhausted"
				a.SaveState(ctx)
				return fmt.Errorf("LLM budget exhausted")
			}
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

		// Record LLM cost
		metadata := map[string]string{
			"agent_id": a.config.ID,
			"task_id":  a.currentTask.ID,
			"chain_id": chainID,
		}
		a.costManager.RecordLLMCost(a.config.Model, promptTokens, resp.TokensUsed, metadata)

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
			// Check if we can afford the tool usage
			toolName := agentResp.Action.Tool
			toolCost := a.costManager.GetToolCost(toolName)
			if !a.costManager.CanAfford(CostTypeTool, toolCost) {
				// Add budget warning to memory
				budgetMsg := fmt.Sprintf("Tool budget exceeded for %s. Trying to proceed without this tool.", toolName)
				a.memoryManager.AddMessage(ctx, chainID, memory.Message{
					Role:      "system",
					Content:   budgetMsg,
					Timestamp: time.Now().UTC(),
				})
				continue
			}

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

			// Record tool cost
			metadata := map[string]string{
				"agent_id": a.config.ID,
				"task_id":  a.currentTask.ID,
				"chain_id": chainID,
			}
			a.costManager.RecordToolCost(toolName, metadata)

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
func (a *Craftsman) executeTool(ctx context.Context, action *AgentAction) (string, error) {
	if action.Tool == "" {
		return "", fmt.Errorf("tool name is required")
	}

	// Check if we're tracking tool costs
	toolBudget := a.costManager.GetBudget(CostTypeTool)
	if toolBudget > 0 {
		// Check if we're within budget before executing the tool
		toolCost := a.costManager.GetToolCost(action.Tool)
		if !a.costManager.CanAfford(CostTypeTool, toolCost) {
			return "", fmt.Errorf("tool budget exceeded for %s", action.Tool)
		}
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
func (a *Craftsman) completeTask(ctx context.Context, summary string) {
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

	// Add cost information to the task metadata
	costReport := a.costManager.GetCostReport()

	// Add LLM cost if available
	if llmCost, ok := costReport["total_costs"].(map[string]float64)[string(CostTypeLLM)]; ok {
		a.currentTask.Metadata["llm_cost"] = fmt.Sprintf("%.6f", llmCost)
	}

	// Add tool cost if available
	if toolCost, ok := costReport["total_costs"].(map[string]float64)[string(CostTypeTool)]; ok {
		a.currentTask.Metadata["tool_cost"] = fmt.Sprintf("%.6f", toolCost)
	}

	// Add total cost
	totalCost := 0.0
	if costs, ok := costReport["total_costs"].(map[string]float64); ok {
		for _, cost := range costs {
			totalCost += cost
		}
	}
	a.currentTask.Metadata["total_cost"] = fmt.Sprintf("%.6f", totalCost)

	// Update agent state
	a.state.Status = StatusIdle
	a.state.CurrentTask = ""
	a.state.UpdatedAt = time.Now().UTC()

	// Save agent state
	a.SaveState(ctx)
}

// Stop stops the craftsman's execution
func (a *Craftsman) Stop(ctx context.Context) error {
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