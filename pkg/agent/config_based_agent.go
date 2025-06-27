// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package agent

import (
	"context"
	"strings"

	"github.com/lancekrogers/guild/pkg/commission"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/memory"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/providers"
	"github.com/lancekrogers/guild/pkg/tools"
)

// ConfigBasedAgent represents an agent created with the enhanced configuration system
type ConfigBasedAgent struct {
	// Core identification
	id        string
	name      string
	agentType string
	role      string
	specialty string
	backstory string

	// Dependencies
	llmClient         providers.LLMClient
	memoryManager     memory.ChainManager
	toolRegistry      tools.Registry
	commissionManager commission.CommissionManager
	costManager       CostManagerInterface

	// Enhanced features
	toolFilter        *ToolFilter
	contextManager    *ContextManager
	config            *config.EnhancedAgentConfig
	providerSelection *ProviderSelection

	// Agent attributes
	capabilities []string
	prompts      map[string]string
	metadata     map[string]interface{}
}

// GetID returns the agent's unique identifier
func (ca *ConfigBasedAgent) GetID() string {
	return ca.id
}

// GetName returns the agent's name
func (ca *ConfigBasedAgent) GetName() string {
	return ca.name
}

// GetType returns the agent's type
func (ca *ConfigBasedAgent) GetType() string {
	return ca.agentType
}

// Execute executes a task with the enhanced agent capabilities
func (ca *ConfigBasedAgent) Execute(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ConfigBasedAgent").
			WithOperation("Execute")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ConfigBasedAgent")
	ctx = observability.WithOperation(ctx, "Execute")

	logger.InfoContext(ctx, "Executing task",
		"agent_id", ca.id,
		"agent_type", ca.agentType,
		"prompt_length", len(prompt))

	// Add the user prompt to context
	if err := ca.contextManager.AddMessage(ctx, "user", prompt, 5); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to add message to context").
			WithComponent("ConfigBasedAgent").
			WithOperation("Execute").
			WithDetails("agent_id", ca.id)
	}

	// Build the complete prompt with context
	messages := ca.contextManager.GetMessages()
	contextPrompt := ca.buildPromptFromMessages(messages)

	// Execute with LLM client
	response, err := ca.llmClient.Complete(ctx, contextPrompt)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "LLM execution failed").
			WithComponent("ConfigBasedAgent").
			WithOperation("Execute").
			WithDetails("agent_id", ca.id)
	}

	// Add the response to context
	if err := ca.contextManager.AddMessage(ctx, "assistant", response, 5); err != nil {
		logger.WarnContext(ctx, "Failed to add response to context", "agent_id", ca.id, "error", err)
	}

	// Update cost tracking (simplified - would need token counts from LLM client)
	ca.contextManager.UpdateCost(ctx, len(contextPrompt)/4, len(response)/4)

	logger.InfoContext(ctx, "Task execution completed",
		"agent_id", ca.id,
		"response_length", len(response))

	return response, nil
}

// ExecuteWithTools executes a task with tool access control
func (ca *ConfigBasedAgent) ExecuteWithTools(ctx context.Context, prompt string, availableTools []string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ConfigBasedAgent").
			WithOperation("ExecuteWithTools")
	}

	logger := observability.GetLogger(ctx)
	ctx = observability.WithComponent(ctx, "ConfigBasedAgent")
	ctx = observability.WithOperation(ctx, "ExecuteWithTools")

	logger.InfoContext(ctx, "Executing task with tools",
		"agent_id", ca.id,
		"available_tools_count", len(availableTools))

	// Filter tools based on agent's access control
	allowedTools := ca.toolFilter.FilterTools(ctx, availableTools)

	logger.InfoContext(ctx, "Tools filtered for agent",
		"agent_id", ca.id,
		"requested_tools", len(availableTools),
		"allowed_tools", len(allowedTools))

	// Execute with the filtered tools
	// For now, we'll just execute normally and mention available tools in the prompt
	toolsPrompt := prompt
	if len(allowedTools) > 0 {
		toolsList := ""
		for _, tool := range allowedTools {
			toolsList += "- " + tool + "\n"
		}
		toolsPrompt = prompt + "\n\nAvailable tools:\n" + toolsList
	}

	return ca.Execute(ctx, toolsPrompt)
}

// HasCapability checks if the agent has a specific capability
func (ca *ConfigBasedAgent) HasCapability(capability string) bool {
	return ca.config.HasCapability(capability)
}

// GetCapabilities returns all agent capabilities
func (ca *ConfigBasedAgent) GetCapabilities() []string {
	capabilities := make([]string, len(ca.capabilities))
	copy(capabilities, ca.capabilities)
	return capabilities
}

// GetAllowedTools returns tools the agent is allowed to use
func (ca *ConfigBasedAgent) GetAllowedTools(ctx context.Context) ([]string, error) {
	return ca.toolFilter.GetAllowedTools(ctx)
}

// GetAgentInfo returns comprehensive information about the agent
func (ca *ConfigBasedAgent) GetAgentInfo(ctx context.Context) map[string]interface{} {
	contextSummary := ca.contextManager.GetContextSummary(ctx)
	toolSummary := ca.toolFilter.GetToolAccessSummary(ctx)

	return map[string]interface{}{
		"id":             ca.id,
		"name":           ca.name,
		"type":           ca.agentType,
		"role":           ca.role,
		"specialty":      ca.specialty,
		"backstory":      ca.backstory,
		"capabilities":   ca.capabilities,
		"provider":       ca.providerSelection.Provider,
		"model":          ca.providerSelection.Model,
		"cost_magnitude": ca.providerSelection.CostProfile.Magnitude,
		"context":        contextSummary,
		"tools":          toolSummary,
		"languages":      ca.config.Languages,
		"frameworks":     ca.config.Frameworks,
		"prompts":        ca.prompts,
		"metadata":       ca.metadata,
	}
}

// UpdateToolAccess updates the agent's tool access configuration
func (ca *ConfigBasedAgent) UpdateToolAccess(ctx context.Context, newToolConfig config.ToolAccessConfig) error {
	return ca.toolFilter.UpdateToolAccess(ctx, newToolConfig)
}

// ResetContext clears the agent's context window
func (ca *ConfigBasedAgent) ResetContext(ctx context.Context) {
	ca.contextManager.Reset(ctx)
}

// GetContextSummary returns the current context summary
func (ca *ConfigBasedAgent) GetContextSummary(ctx context.Context) map[string]interface{} {
	return ca.contextManager.GetContextSummary(ctx)
}

// buildPromptFromMessages builds a complete prompt from context messages
func (ca *ConfigBasedAgent) buildPromptFromMessages(messages []ContextMessage) string {
	var promptBuilder strings.Builder

	// Add system prompt if available
	if systemPrompt, exists := ca.prompts["system"]; exists {
		promptBuilder.WriteString("System: " + systemPrompt + "\n\n")
	} else if ca.backstory != "" {
		promptBuilder.WriteString("System: You are " + ca.name + ". " + ca.backstory + "\n\n")
	}

	// Add conversation history
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			promptBuilder.WriteString("System: " + msg.Content + "\n\n")
		case "user":
			promptBuilder.WriteString("Human: " + msg.Content + "\n\n")
		case "assistant":
			promptBuilder.WriteString("Assistant: " + msg.Content + "\n\n")
		}
	}

	promptBuilder.WriteString("Assistant: ")
	return promptBuilder.String()
}

// GetConfig returns the agent's configuration (read-only)
func (ca *ConfigBasedAgent) GetConfig() *config.EnhancedAgentConfig {
	// Return a copy to prevent modification
	configCopy := *ca.config
	return &configCopy
}

// GetProviderSelection returns the provider selection information
func (ca *ConfigBasedAgent) GetProviderSelection() *ProviderSelection {
	// Return a copy to prevent modification
	selectionCopy := *ca.providerSelection
	return &selectionCopy
}

// ValidateToolAccess validates that the agent can use a specific tool
func (ca *ConfigBasedAgent) ValidateToolAccess(ctx context.Context, toolName string) error {
	return ca.toolFilter.ValidateToolAccess(ctx, toolName)
}

// EstimateTaskCost estimates the cost of executing a task
func (ca *ConfigBasedAgent) EstimateTaskCost(ctx context.Context, prompt string) (float64, error) {
	// Estimate tokens for the prompt
	promptTokens := len(prompt) / 4 // Simple estimation
	responseTokens := 500           // Estimated response length

	promptCost := float64(promptTokens) / 1000.0 * ca.providerSelection.CostProfile.PromptCostPer1K
	responseCost := float64(responseTokens) / 1000.0 * ca.providerSelection.CostProfile.OutputCostPer1K

	return promptCost + responseCost, nil
}

// CanAffordTask checks if the agent can afford to execute a task within budget
func (ca *ConfigBasedAgent) CanAffordTask(ctx context.Context, prompt string) (bool, error) {
	estimatedCost, err := ca.EstimateTaskCost(ctx, prompt)
	if err != nil {
		return false, err
	}

	return ca.costManager.CanAfford(CostTypeLLM, estimatedCost), nil
}
