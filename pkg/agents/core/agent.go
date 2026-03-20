// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild-core/pkg/commission"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/interfaces"
	"github.com/lancekrogers/guild-core/pkg/memory"
	"github.com/lancekrogers/guild-core/pkg/observability"
	"github.com/lancekrogers/guild-core/pkg/providers"
	"github.com/lancekrogers/guild-core/pkg/tools"
)

// Agent is an alias to the shared interface to avoid circular dependencies
type Agent = interfaces.Agent

// GuildArtisan is the primary agent interface
type GuildArtisan interface {
	Agent

	// GetToolRegistry returns the tool registry
	GetToolRegistry() tools.Registry

	// GetCommissionManager returns the commission manager
	GetCommissionManager() commission.CommissionManager

	// GetLLMClient returns the LLM client
	GetLLMClient() providers.LLMClient

	// GetMemoryManager returns the memory manager
	GetMemoryManager() memory.ChainManager
}

// WorkerAgent is a standard worker agent
type WorkerAgent struct {
	ID                string
	Name              string
	LLMClient         providers.LLMClient
	MemoryManager     memory.ChainManager
	ToolRegistry      tools.Registry
	CommissionManager commission.CommissionManager
	CostManager       CostManagerInterface

	// Context metadata
	capabilities []string
	description  string

	// Reasoning support
	reasoningExtractor *ReasoningExtractor
	reasoningStorage   ReasoningStorage
}

// newWorkerAgent creates a new worker agent (private constructor)
func newWorkerAgent(id, name string, llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface,
) *WorkerAgent {
	return &WorkerAgent{
		ID:                id,
		Name:              name,
		LLMClient:         llmClient,
		MemoryManager:     memoryManager,
		ToolRegistry:      toolRegistry,
		CommissionManager: commissionManager,
		CostManager:       costManager,
	}
}

// Execute runs a task with full tool support
func (a *WorkerAgent) Execute(ctx context.Context, request string) (string, error) {
	// Call the new ExecuteWithReasoning method and return just the content for backward compatibility
	response, err := a.ExecuteWithReasoning(ctx, request)
	if err != nil {
		return "", err
	}
	return response.Content, nil
}

// ExecuteWithReasoning runs a task and returns structured response with reasoning
func (a *WorkerAgent) ExecuteWithReasoning(ctx context.Context, request string) (*AgentResponse, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "agent execution cancelled").
			WithComponent("agent").
			WithOperation("Execute").
			WithDetails("agent_id", a.ID)
	}

	// Initialize observability
	logger := observability.GetLogger(ctx).
		WithComponent("agent").
		WithOperation("ExecuteWithReasoning").
		With("agent_id", a.ID, "agent_name", a.Name)

	// Add agent context for tracing
	ctx = context.WithValue(ctx, "agent_id", a.ID)

	// Start operation logging with timing
	start := time.Now()
	logger.InfoContext(ctx, "Starting agent execution with reasoning",
		"request_length", len(request),
		"has_cost_manager", a.CostManager != nil,
		"has_tools", a.ToolRegistry != nil,
	)

	var rawResponse string
	var err error
	var cost float64
	var toolsUsed []string

	// If we have a cost-aware implementation, use it
	if a.LLMClient != nil && a.CostManager != nil {
		logger.DebugContext(ctx, "Using cost-aware execution")
		rawResponse, err = a.CostAwareExecute(ctx, request)
		if a.CostManager != nil {
			cost = a.CostManager.GetTotalCost()
		}
	} else {
		// Otherwise, execute with tools if available
		if a.LLMClient == nil {
			err = gerror.New(gerror.ErrCodeValidation, "no LLM client configured", nil).
				WithComponent("agent").
				WithOperation("ExecuteWithReasoning").
				WithDetails("agent_id", a.ID)
			logger.ErrorContext(ctx, "No LLM client configured")
			return nil, err
		}

		// If we have tools available, create a task executor for tool-enabled execution
		if a.ToolRegistry != nil {
			logger.DebugContext(ctx, "Using tool-enabled execution",
				"available_tools", len(a.ToolRegistry.ListTools()))
			rawResponse, err = a.executeWithTools(ctx, request)
			// TODO: Extract actual tools used from response
		} else {
			// Fall back to simple LLM execution without tools
			logger.DebugContext(ctx, "Using simple LLM execution")
			rawResponse, err = a.LLMClient.Complete(ctx, request)
			if err != nil {
				err = gerror.Wrap(err, gerror.ErrCodeProvider, "LLM completion failed").
					WithComponent("agent").
					WithOperation("ExecuteWithReasoning").
					WithDetails("agent_id", a.ID)
			}
		}
	}

	// Log execution results with timing
	duration := time.Since(start)
	if err != nil {
		logger.WithError(err).ErrorContext(ctx, "Agent execution failed",
			"duration_ms", duration.Milliseconds(),
			"request_length", len(request),
		)
		return nil, err
	}

	// Extract reasoning using enhanced extractor if available
	var response *AgentResponse
	var extractErr error

	if a.reasoningExtractor != nil {
		// Use enhanced extractor with full context support
		response, extractErr = a.reasoningExtractor.ExtractReasoning(ctx, rawResponse)
		if extractErr != nil {
			logger.WithError(extractErr).WarnContext(ctx, "Enhanced reasoning extraction failed, falling back to simple extraction")
			// Fall back to simple extraction
			cleanContent, reasoning, confidence := ExtractReasoning(rawResponse)
			response = &AgentResponse{
				Content:    cleanContent,
				Reasoning:  reasoning,
				Confidence: confidence,
			}
		}
	} else {
		// Use simple extraction
		cleanContent, reasoning, confidence := ExtractReasoning(rawResponse)
		response = &AgentResponse{
			Content:    cleanContent,
			Reasoning:  reasoning,
			Confidence: confidence,
		}
	}

	// Log reasoning extraction
	if response.Reasoning != "" {
		logger.DebugContext(ctx, "Extracted reasoning from response",
			"reasoning_length", len(response.Reasoning),
			"confidence", response.Confidence,
			"enhanced_extractor", a.reasoningExtractor != nil,
		)

		// Store reasoning if storage is available
		if a.reasoningStorage != nil {
			go a.storeReasoningChain(context.Background(), response, duration)
		}
	}

	// Add additional metadata
	response.ToolsUsed = toolsUsed
	response.Cost = cost
	if response.Metadata == nil {
		response.Metadata = make(map[string]interface{})
	}
	response.Metadata["agent_id"] = a.ID
	response.Metadata["agent_name"] = a.Name
	response.Metadata["duration_ms"] = duration.Milliseconds()

	logger.InfoContext(ctx, "Agent execution completed successfully",
		"duration_ms", duration.Milliseconds(),
		"response_length", len(response.Content),
		"request_length", len(request),
		"has_reasoning", response.Reasoning != "",
		"confidence", response.Confidence,
	)

	// Log performance metrics for monitoring
	logger.Duration("agent.execute_with_reasoning", duration,
		"agent_id", a.ID,
		"success", true,
		"response_size", len(response.Content),
		"has_reasoning", response.Reasoning != "",
	)

	return response, nil
}

// GetID returns the agent's ID
func (a *WorkerAgent) GetID() string {
	return a.ID
}

// GetName returns the agent's name
func (a *WorkerAgent) GetName() string {
	return a.Name
}

// GetToolRegistry returns the tool registry
func (a *WorkerAgent) GetToolRegistry() tools.Registry {
	return a.ToolRegistry
}

// GetCommissionManager returns the commission manager
func (a *WorkerAgent) GetCommissionManager() commission.CommissionManager {
	return a.CommissionManager
}

// GetLLMClient returns the LLM client
func (a *WorkerAgent) GetLLMClient() providers.LLMClient {
	return a.LLMClient
}

// GetMemoryManager returns the memory manager
func (a *WorkerAgent) GetMemoryManager() memory.ChainManager {
	return a.MemoryManager
}

// SetCapabilities sets the agent's capabilities
func (a *WorkerAgent) SetCapabilities(capabilities []string) {
	a.capabilities = capabilities
}

// GetCapabilities returns the agent's capabilities
func (a *WorkerAgent) GetCapabilities() []string {
	return a.capabilities
}

// GetType returns the agent's type
func (a *WorkerAgent) GetType() string {
	return "worker"
}

// SetDescription sets the agent's description
func (a *WorkerAgent) SetDescription(description string) {
	a.description = description
}

// GetDescription returns the agent's description
func (a *WorkerAgent) GetDescription() string {
	return a.description
}

// HasCapability checks if the agent has a specific capability
func (a *WorkerAgent) HasCapability(capability string) bool {
	for _, cap := range a.capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// SetCostBudget sets the budget for a specific cost type
func (a *WorkerAgent) SetCostBudget(costType CostType, amount float64) {
	a.CostManager.SetBudget(costType, amount)
}

// GetCostReport returns a report of all costs incurred by the agent
func (a *WorkerAgent) GetCostReport() map[string]interface{} {
	return a.CostManager.GetCostReport()
}

// executeWithTools executes a task with tool awareness and basic tool execution
func (a *WorkerAgent) executeWithTools(ctx context.Context, request string) (string, error) {
	// Initialize observability for tool execution
	logger := observability.GetLogger(ctx).
		WithComponent("agent").
		WithOperation("executeWithTools").
		With("agent_id", a.ID, "agent_name", a.Name)

	// Start tool execution timing
	start := time.Now()
	logger.InfoContext(ctx, "Starting tool-enabled execution",
		"request_length", len(request),
		"has_tool_registry", a.ToolRegistry != nil,
	)

	// Get available tools for context
	var toolContext string
	var availableTools []string

	if a.ToolRegistry != nil {
		toolStart := time.Now()
		availableTools = a.ToolRegistry.ListTools()
		toolListDuration := time.Since(toolStart)

		logger.DebugContext(ctx, "Retrieved available tools",
			"tool_count", len(availableTools),
			"tool_list_duration_ms", toolListDuration.Milliseconds(),
		)

		if len(availableTools) > 0 {
			toolDescriptions := make([]string, 0, len(availableTools))
			var toolErrors []string

			for _, toolName := range availableTools {
				tool, err := a.ToolRegistry.GetTool(toolName)
				if err == nil {
					toolDescriptions = append(toolDescriptions, fmt.Sprintf("- %s: %s", toolName, tool.Description()))
				} else {
					toolErrors = append(toolErrors, toolName)
					logger.WarnContext(ctx, "Failed to get tool description",
						"tool_name", toolName,
						"error", err.Error(),
					)
				}
			}

			if len(toolErrors) > 0 {
				logger.WarnContext(ctx, "Some tools unavailable",
					"unavailable_tools", toolErrors,
					"available_tools", len(toolDescriptions),
				)
			}

			if len(toolDescriptions) > 0 {
				toolContext = "\n\nAvailable tools:\n" + strings.Join(toolDescriptions, "\n")
				toolContext += "\n\nYou can reference these tools in your response and I can execute them if needed."

				logger.DebugContext(ctx, "Tool context prepared",
					"tool_context_length", len(toolContext),
					"tool_descriptions_count", len(toolDescriptions),
				)
			}
		} else {
			logger.DebugContext(ctx, "No tools available in registry")
		}
	} else {
		logger.DebugContext(ctx, "No tool registry configured")
	}

	// Create enhanced prompt with tool context
	enhancedRequest := request + toolContext
	logger.DebugContext(ctx, "Enhanced request prepared",
		"enhanced_request_length", len(enhancedRequest),
		"tool_context_added", len(toolContext) > 0,
	)

	// Execute with LLM
	llmStart := time.Now()
	response, err := a.LLMClient.Complete(ctx, enhancedRequest)
	llmDuration := time.Since(llmStart)

	// Log execution results with timing
	duration := time.Since(start)
	if err != nil {
		logger.WithError(err).ErrorContext(ctx, "Tool-enabled execution failed",
			"duration_ms", duration.Milliseconds(),
			"llm_duration_ms", llmDuration.Milliseconds(),
			"request_length", len(request),
			"enhanced_request_length", len(enhancedRequest),
			"available_tools", len(availableTools),
		)
		return "", gerror.Wrap(err, gerror.ErrCodeProvider, "LLM completion failed with tool context").
			WithComponent("agent").
			WithOperation("executeWithTools").
			WithDetails("agent_id", a.ID)
	}

	logger.InfoContext(ctx, "Tool-enabled execution completed successfully",
		"duration_ms", duration.Milliseconds(),
		"llm_duration_ms", llmDuration.Milliseconds(),
		"response_length", len(response),
		"request_length", len(request),
		"enhanced_request_length", len(enhancedRequest),
		"available_tools", len(availableTools),
	)

	// Log performance metrics for monitoring
	logger.Duration("agent.execute_with_tools", duration,
		"agent_id", a.ID,
		"success", true,
		"response_size", len(response),
		"tool_count", len(availableTools),
		"llm_duration_ms", llmDuration.Milliseconds(),
	)

	// TODO: Parse the response for tool calls and execute them
	// For now, we're just providing tool awareness to the LLM
	// Future enhancement: Parse response for tool execution requests and execute them
	logger.DebugContext(ctx, "Tool execution parsing not yet implemented - returning LLM response")

	return response, nil
}

// storeReasoningChain stores reasoning chain asynchronously
func (a *WorkerAgent) storeReasoningChain(ctx context.Context, response *AgentResponse, duration time.Duration) {
	if a.reasoningStorage == nil || response.Reasoning == "" {
		return
	}

	logger := observability.GetLogger(ctx).
		WithComponent("agent").
		WithOperation("storeReasoningChain").
		With("agent_id", a.ID)

	chain := &ReasoningChain{
		ID:         fmt.Sprintf("%s-%d", a.ID, time.Now().UnixNano()),
		AgentID:    a.ID,
		Content:    response.Content,
		Reasoning:  response.Reasoning,
		Confidence: response.Confidence,
		Success:    true,
		Duration:   duration,
		CreatedAt:  time.Now(),
		Metadata:   response.Metadata,
	}

	if err := a.reasoningStorage.Store(ctx, chain); err != nil {
		logger.WithError(err).ErrorContext(ctx, "Failed to store reasoning chain")
	} else {
		logger.DebugContext(ctx, "Reasoning chain stored successfully",
			"chain_id", chain.ID,
			"confidence", chain.Confidence)
	}
}

// SetReasoningExtractor sets the reasoning extractor for the agent
func (a *WorkerAgent) SetReasoningExtractor(extractor *ReasoningExtractor) {
	a.reasoningExtractor = extractor
}

// SetReasoningStorage sets the reasoning storage for the agent
func (a *WorkerAgent) SetReasoningStorage(storage ReasoningStorage) {
	a.reasoningStorage = storage
}

// ManagerAgent is a coordinator agent
type ManagerAgent struct {
	WorkerAgent
}

// newManagerAgent creates a new manager agent (private constructor)
func newManagerAgent(id, name string, llmClient providers.LLMClient,
	memoryManager memory.ChainManager,
	toolRegistry tools.Registry,
	commissionManager commission.CommissionManager,
	costManager CostManagerInterface,
) *ManagerAgent {
	worker := newWorkerAgent(id, name, llmClient, memoryManager, toolRegistry, commissionManager, costManager)

	return &ManagerAgent{
		WorkerAgent: *worker,
	}
}
