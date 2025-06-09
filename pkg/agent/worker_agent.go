package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/guild-ventures/guild-core/pkg/observability"
)

// CostAwareExecute is an enhanced execution method that tracks costs
func (a *WorkerAgent) CostAwareExecute(ctx context.Context, request string) (string, error) {
	// Initialize observability for cost-aware execution
	logger := observability.GetLogger(ctx).
		WithComponent("agent").
		WithOperation("CostAwareExecute").
		With("agent_id", a.ID, "agent_name", a.Name)

	// Start execution timing
	start := time.Now()
	logger.InfoContext(ctx, "Starting cost-aware execution",
		"request_length", len(request),
		"has_llm_client", a.LLMClient != nil,
		"has_cost_manager", a.CostManager != nil,
		"has_memory_manager", a.MemoryManager != nil,
	)

	// Check if we have a valid LLM client
	if a.LLMClient == nil {
		logger.ErrorContext(ctx, "No LLM client configured")
		return "", gerror.New(gerror.ErrCodeValidation, "no LLM client configured", nil).
			WithComponent("agent").
			WithOperation("CostAwareExecute").
			WithDetails("agent_id", a.ID)
	}

	// Check if we have a valid cost manager
	if a.CostManager == nil {
		logger.ErrorContext(ctx, "No cost manager configured")
		return "", gerror.New(gerror.ErrCodeValidation, "no cost manager configured", nil).
			WithComponent("agent").
			WithOperation("CostAwareExecute").
			WithDetails("agent_id", a.ID)
	}

	// Estimate cost for this request
	// Rough estimation: 1 character ≈ 0.25 tokens
	estimatedPromptTokens := len(request) / 4
	estimatedCompletionTokens := 500 // Assume a moderate response

	// Get model from LLM client (this would need to be added to the interface)
	model := "gpt-3.5-turbo" // Default model, would get from config

	// Estimate cost
	totalEstimatedTokens := estimatedPromptTokens + estimatedCompletionTokens
	estimatedCost := a.CostManager.EstimateLLMCost(model, totalEstimatedTokens)

	logger.DebugContext(ctx, "Cost estimation completed",
		"estimated_prompt_tokens", estimatedPromptTokens,
		"estimated_completion_tokens", estimatedCompletionTokens,
		"total_estimated_tokens", totalEstimatedTokens,
		"estimated_cost", estimatedCost,
		"model", model,
	)

	// Check if we can afford it
	if !a.CostManager.CanAfford(CostTypeLLM, estimatedCost) {
		logger.ErrorContext(ctx, "LLM budget exceeded",
			"estimated_cost", estimatedCost,
			"cost_type", CostTypeLLM,
		)
		return "", gerror.Newf(gerror.ErrCodeResourceLimit, "LLM budget exceeded: estimated cost $%.4f exceeds available budget", estimatedCost).
			WithComponent("agent").
			WithOperation("CostAwareExecute").
			WithDetails("agent_id", a.ID).
			WithDetails("estimated_cost", estimatedCost)
	}

	// Create or get memory chain
	var chainID string
	var err error
	var memoryStart time.Time

	if a.MemoryManager != nil {
		memoryStart = time.Now()
		// Create a new chain for this execution
		chainID, err = a.MemoryManager.CreateChain(ctx, a.ID, "task-"+time.Now().Format("20060102150405"))
		memoryCreateDuration := time.Since(memoryStart)

		if err != nil {
			logger.WithError(err).ErrorContext(ctx, "Failed to create memory chain",
				"memory_create_duration_ms", memoryCreateDuration.Milliseconds(),
			)
			return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create memory chain").
				WithComponent("agent").
				WithOperation("CostAwareExecute").
				WithDetails("agent_id", a.ID)
		}

		logger.DebugContext(ctx, "Memory chain created",
			"chain_id", chainID,
			"memory_create_duration_ms", memoryCreateDuration.Milliseconds(),
		)

		// Add the request to memory
		memoryAddStart := time.Now()
		err = a.MemoryManager.AddMessage(ctx, chainID, memory.Message{
			Role:      "user",
			Content:   request,
			Timestamp: time.Now().UTC(),
		})
		memoryAddDuration := time.Since(memoryAddStart)

		if err != nil {
			logger.WithError(err).ErrorContext(ctx, "Failed to add request to memory",
				"chain_id", chainID,
				"memory_add_duration_ms", memoryAddDuration.Milliseconds(),
			)
			return "", gerror.Wrap(err, gerror.ErrCodeStorage, "failed to add request to memory").
				WithComponent("agent").
				WithOperation("CostAwareExecute").
				WithDetails("agent_id", a.ID).
				WithDetails("chain_id", chainID)
		}

		logger.DebugContext(ctx, "Request added to memory",
			"chain_id", chainID,
			"memory_add_duration_ms", memoryAddDuration.Milliseconds(),
		)
	} else {
		logger.DebugContext(ctx, "No memory manager configured - proceeding without memory")
	}

	// Execute the request with the LLM
	llmStart := time.Now()
	response, err := a.LLMClient.Complete(ctx, request)
	llmDuration := time.Since(llmStart)

	if err != nil {
		logger.WithError(err).ErrorContext(ctx, "LLM execution failed",
			"llm_duration_ms", llmDuration.Milliseconds(),
			"model", model,
		)
		return "", gerror.Wrap(err, gerror.ErrCodeProvider, "LLM execution failed").
			WithComponent("agent").
			WithOperation("CostAwareExecute").
			WithDetails("agent_id", a.ID).
			WithDetails("model", model)
	}

	// Calculate actual cost (would get actual token counts from response)
	// In reality, the response would include usage information
	actualPromptTokens := len(request) / 4
	actualCompletionTokens := len(response) / 4

	// Record the cost
	costStart := time.Now()
	cost := a.CostManager.RecordLLMCost(model, actualPromptTokens, actualCompletionTokens, map[string]string{
		"agent_id":  a.ID,
		"chain_id":  chainID,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	costDuration := time.Since(costStart)

	logger.DebugContext(ctx, "Cost recorded",
		"actual_prompt_tokens", actualPromptTokens,
		"actual_completion_tokens", actualCompletionTokens,
		"total_tokens", actualPromptTokens+actualCompletionTokens,
		"actual_cost", cost,
		"cost_record_duration_ms", costDuration.Milliseconds(),
	)

	// Add response to memory
	if a.MemoryManager != nil && chainID != "" {
		memoryResponseStart := time.Now()
		err = a.MemoryManager.AddMessage(ctx, chainID, memory.Message{
			Role:       "assistant",
			Content:    response,
			Timestamp:  time.Now().UTC(),
			TokenUsage: actualPromptTokens + actualCompletionTokens,
		})
		memoryResponseDuration := time.Since(memoryResponseStart)

		if err != nil {
			// Log error but don't fail the execution
			logger.WithError(err).WarnContext(ctx, "Failed to add response to memory",
				"chain_id", chainID,
				"memory_response_duration_ms", memoryResponseDuration.Milliseconds(),
			)
		} else {
			logger.DebugContext(ctx, "Response added to memory",
				"chain_id", chainID,
				"memory_response_duration_ms", memoryResponseDuration.Milliseconds(),
			)
		}
	}

	// Log execution completion with comprehensive timing
	duration := time.Since(start)
	logger.InfoContext(ctx, "Cost-aware execution completed successfully",
		"duration_ms", duration.Milliseconds(),
		"llm_duration_ms", llmDuration.Milliseconds(),
		"response_length", len(response),
		"request_length", len(request),
		"actual_cost", cost,
		"actual_tokens", actualPromptTokens+actualCompletionTokens,
		"chain_id", chainID,
	)

	// Log performance metrics for monitoring
	logger.Duration("agent.cost_aware_execute", duration,
		"agent_id", a.ID,
		"success", true,
		"response_size", len(response),
		"cost", cost,
		"tokens", actualPromptTokens+actualCompletionTokens,
		"llm_duration_ms", llmDuration.Milliseconds(),
	)

	return response, nil
}

// ExecuteWithTools executes a request that may involve tool usage
func (a *WorkerAgent) ExecuteWithTools(ctx context.Context, request string, allowedTools []string) (string, error) {
	// Initialize observability for tool execution
	logger := observability.GetLogger(ctx).
		WithComponent("agent").
		WithOperation("ExecuteWithTools").
		With("agent_id", a.ID, "agent_name", a.Name)

	// Start execution timing
	start := time.Now()
	logger.InfoContext(ctx, "Starting execution with tools",
		"request_length", len(request),
		"allowed_tools_count", len(allowedTools),
		"allowed_tools", allowedTools,
		"has_cost_manager", a.CostManager != nil,
		"has_tool_registry", a.ToolRegistry != nil,
	)

	// For now, we'll use a fixed cost per tool
	const estimatedCostPerTool = 0.001
	totalToolCost := float64(len(allowedTools)) * estimatedCostPerTool

	logger.DebugContext(ctx, "Tool cost estimation",
		"estimated_cost_per_tool", estimatedCostPerTool,
		"total_tool_cost", totalToolCost,
	)

	// Check if we can afford tool usage
	if a.CostManager != nil && !a.CostManager.CanAfford(CostTypeTool, totalToolCost) {
		logger.ErrorContext(ctx, "Tool budget exceeded",
			"total_tool_cost", totalToolCost,
			"allowed_tools_count", len(allowedTools),
		)
		return "", gerror.Newf(gerror.ErrCodeResourceLimit, "tool budget exceeded: estimated cost $%.4f exceeds available budget", totalToolCost).
			WithComponent("agent").
			WithOperation("ExecuteWithTools").
			WithDetails("agent_id", a.ID).
			WithDetails("total_tool_cost", totalToolCost).
			WithDetails("allowed_tools", allowedTools)
	}

	// Execute with cost-aware LLM
	llmStart := time.Now()
	response, err := a.CostAwareExecute(ctx, request)
	llmDuration := time.Since(llmStart)

	if err != nil {
		logger.WithError(err).ErrorContext(ctx, "Cost-aware execution failed",
			"llm_duration_ms", llmDuration.Milliseconds(),
		)
		return "", err
	}

	logger.DebugContext(ctx, "Cost-aware execution completed",
		"llm_duration_ms", llmDuration.Milliseconds(),
		"response_length", len(response),
	)

	// Parse response for tool calls (simplified for demo)
	// In reality, we'd parse the response to find tool invocations

	// Example tool usage
	if len(allowedTools) > 0 {
		toolName := allowedTools[0]
		toolInput := `{"query": "example"}`

		logger.DebugContext(ctx, "Executing tool",
			"tool_name", toolName,
			"tool_input", toolInput,
		)

		// Execute tool with cost tracking
		// Get the tool
		toolGetStart := time.Now()
		tool, err := a.ToolRegistry.GetTool(toolName)
		toolGetDuration := time.Since(toolGetStart)

		if err != nil {
			logger.WithError(err).ErrorContext(ctx, "Tool not found",
				"tool_name", toolName,
				"tool_get_duration_ms", toolGetDuration.Milliseconds(),
			)
			return "", gerror.Wrap(err, gerror.ErrCodeAgent, "tool not found").
				WithComponent("agent").
				WithOperation("ExecuteWithTools").
				WithDetails("agent_id", a.ID).
				WithDetails("tool_name", toolName)
		}

		logger.DebugContext(ctx, "Tool retrieved successfully",
			"tool_name", toolName,
			"tool_get_duration_ms", toolGetDuration.Milliseconds(),
		)

		// Execute the tool
		toolExecStart := time.Now()
		result, err := tool.Execute(ctx, toolInput)
		toolExecDuration := time.Since(toolExecStart)

		if err != nil {
			logger.WithError(err).ErrorContext(ctx, "Tool execution failed",
				"tool_name", toolName,
				"tool_input", toolInput,
				"tool_exec_duration_ms", toolExecDuration.Milliseconds(),
			)
			return "", gerror.Wrap(err, gerror.ErrCodeAgent, "tool execution failed").
				WithComponent("agent").
				WithOperation("ExecuteWithTools").
				WithDetails("agent_id", a.ID).
				WithDetails("tool_name", toolName).
				WithDetails("tool_input", toolInput)
		}

		logger.DebugContext(ctx, "Tool executed successfully",
			"tool_name", toolName,
			"tool_exec_duration_ms", toolExecDuration.Milliseconds(),
			"tool_output_length", len(result.Output),
		)

		// Track the cost
		cost := estimatedCostPerTool

		// Record tool cost
		if a.CostManager != nil {
			costTrackStart := time.Now()
			if err := a.CostManager.TrackCost(CostTypeTool, cost); err != nil {
				costTrackDuration := time.Since(costTrackStart)
				logger.WithError(err).ErrorContext(ctx, "Failed to track tool cost",
					"tool_name", toolName,
					"cost", cost,
					"cost_track_duration_ms", costTrackDuration.Milliseconds(),
				)
				return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to track tool cost").
					WithComponent("WorkerAgent").
					WithOperation("processTools").
					WithDetails("tool_name", toolName)
			}
			costTrackDuration := time.Since(costTrackStart)

			logger.DebugContext(ctx, "Tool cost tracked",
				"tool_name", toolName,
				"cost", cost,
				"cost_track_duration_ms", costTrackDuration.Milliseconds(),
			)
		}

		// Incorporate tool result into response
		originalLength := len(response)
		response = fmt.Sprintf("%s\n\nTool Result (%s):\n%s", response, toolName, result.Output)

		logger.DebugContext(ctx, "Tool result incorporated",
			"tool_name", toolName,
			"original_response_length", originalLength,
			"enhanced_response_length", len(response),
			"tool_result_length", len(result.Output),
		)
	} else {
		logger.DebugContext(ctx, "No tools to execute - returning LLM response only")
	}

	// Log completion with comprehensive timing
	duration := time.Since(start)
	logger.InfoContext(ctx, "Tool execution completed successfully",
		"duration_ms", duration.Milliseconds(),
		"llm_duration_ms", llmDuration.Milliseconds(),
		"final_response_length", len(response),
		"request_length", len(request),
		"tools_executed", len(allowedTools),
	)

	// Log performance metrics for monitoring
	logger.Duration("agent.execute_with_tools_full", duration,
		"agent_id", a.ID,
		"success", true,
		"response_size", len(response),
		"tools_count", len(allowedTools),
		"llm_duration_ms", llmDuration.Milliseconds(),
	)

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
