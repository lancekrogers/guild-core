package agent

import (
	"context"
	"fmt"
	"time"
	
	"github.com/guild-ventures/guild-core/pkg/memory"
)

// CostAwareExecute is an enhanced execution method that tracks costs
func (a *WorkerAgent) CostAwareExecute(ctx context.Context, request string) (string, error) {
	// Check if we have a valid LLM client
	if a.LLMClient == nil {
		return "", fmt.Errorf("no LLM client configured")
	}
	
	// Estimate cost for this request
	// Rough estimation: 1 character ≈ 0.25 tokens
	estimatedPromptTokens := len(request) / 4
	estimatedCompletionTokens := 500 // Assume a moderate response
	
	// Get model from LLM client (this would need to be added to the interface)
	model := "gpt-3.5-turbo" // Default model, would get from config
	
	// Estimate cost
	estimatedCost := a.CostManager.EstimateLLMCost(model, estimatedPromptTokens, estimatedCompletionTokens)
	
	// Check if we can afford it
	if !a.CostManager.CanAfford(CostTypeLLM, estimatedCost) {
		return "", fmt.Errorf("LLM budget exceeded: estimated cost $%.4f exceeds available budget", estimatedCost)
	}
	
	// Create or get memory chain
	var chainID string
	var err error
	
	if a.MemoryManager != nil {
		// Create a new chain for this execution
		chainID, err = a.MemoryManager.CreateChain(ctx, a.ID, "task-" + time.Now().Format("20060102150405"))
		if err != nil {
			return "", fmt.Errorf("failed to create memory chain: %w", err)
		}
		
		// Add the request to memory
		err = a.MemoryManager.AddMessage(ctx, chainID, memory.Message{
			Role:      "user",
			Content:   request,
			Timestamp: time.Now().UTC(),
		})
		if err != nil {
			return "", fmt.Errorf("failed to add request to memory: %w", err)
		}
	}
	
	// Execute the request with the LLM
	response, err := a.LLMClient.Complete(ctx, request)
	if err != nil {
		return "", fmt.Errorf("LLM execution failed: %w", err)
	}
	
	// Calculate actual cost (would get actual token counts from response)
	// In reality, the response would include usage information
	actualPromptTokens := len(request) / 4
	actualCompletionTokens := len(response) / 4
	
	// Record the cost
	cost := a.CostManager.RecordLLMCost(model, actualPromptTokens, actualCompletionTokens, map[string]string{
		"agent_id":  a.ID,
		"chain_id":  chainID,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	
	// Add response to memory
	if a.MemoryManager != nil && chainID != "" {
		err = a.MemoryManager.AddMessage(ctx, chainID, memory.Message{
			Role:       "assistant",
			Content:    response,
			Timestamp:  time.Now().UTC(),
			TokenUsage: actualPromptTokens + actualCompletionTokens,
		})
		if err != nil {
			// Log error but don't fail the execution
			fmt.Printf("Warning: failed to add response to memory: %v\n", err)
		}
	}
	
	// Log cost information
	fmt.Printf("Agent %s executed request. Cost: $%.6f (Prompt: %d tokens, Completion: %d tokens)\n", 
		a.ID, cost, actualPromptTokens, actualCompletionTokens)
	
	return response, nil
}

// ExecuteWithTools executes a request that may involve tool usage
func (a *WorkerAgent) ExecuteWithTools(ctx context.Context, request string, allowedTools []string) (string, error) {
	// First, check the cost of using tools
	var totalToolCost float64
	for _, toolName := range allowedTools {
		totalToolCost += a.ToolRegistry.GetToolCost(toolName)
	}
	
	// Check if we can afford tool usage
	if !a.CostManager.CanAfford(CostTypeTool, totalToolCost) {
		return "", fmt.Errorf("tool budget exceeded: estimated cost $%.4f exceeds available budget", totalToolCost)
	}
	
	// Execute with cost-aware LLM
	response, err := a.CostAwareExecute(ctx, request)
	if err != nil {
		return "", err
	}
	
	// Parse response for tool calls (simplified for demo)
	// In reality, we'd parse the response to find tool invocations
	
	// Example tool usage
	if len(allowedTools) > 0 {
		toolName := allowedTools[0]
		toolInput := `{"query": "example"}`
		
		// Execute tool with cost tracking
		result, cost, err := a.ToolRegistry.ExecuteToolWithCostTracking(ctx, toolName, toolInput)
		if err != nil {
			return "", fmt.Errorf("tool execution failed: %w", err)
		}
		
		// Record tool cost
		a.CostManager.RecordToolCost(toolName, map[string]string{
			"agent_id": a.ID,
			"input":    toolInput,
			"output":   result.Output[:100], // Truncated for metadata
			"cost":     fmt.Sprintf("%.4f", cost),
		})
		
		// Incorporate tool result into response
		response = fmt.Sprintf("%s\n\nTool Result (%s):\n%s", response, toolName, result.Output)
	}
	
	return response, nil
}

// GetCurrentCosts returns a summary of current costs
func (a *WorkerAgent) GetCurrentCosts() map[string]float64 {
	report := a.CostManager.GetCostReport()
	
	costs := make(map[string]float64)
	if totalCosts, ok := report["total_costs"].(map[string]float64); ok {
		costs = totalCosts
	}
	
	// Add budget usage percentages
	if budgets, ok := report["budgets"].(map[string]float64); ok {
		for costType, budget := range budgets {
			if budget > 0 && costs[costType] > 0 {
				percentage := (costs[costType] / budget) * 100
				costs[costType+"_budget_used"] = percentage
			}
		}
	}
	
	return costs
}