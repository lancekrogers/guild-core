// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"

	"github.com/lancekrogers/guild-core/pkg/providers/interfaces"
)

// ToolAgent extends Agent with tool execution capabilities
type ToolAgent interface {
	Agent

	// ExecuteWithTools executes the agent with available tools
	// Returns the response text and any tool calls the agent wants to make
	ExecuteWithTools(ctx context.Context, input string, availableTools []interfaces.ToolDefinition) (response string, toolCalls []interfaces.ToolCall, err error)

	// ContinueWithToolResult continues execution after a tool has been executed
	// Takes the tool call ID and the result, returns additional response
	ContinueWithToolResult(ctx context.Context, toolCallID string, result string) (string, error)
}

// BaseToolAgent provides a base implementation of ToolAgent
type BaseToolAgent struct {
	*BaseAgent
	provider interfaces.AIProviderWithTools
}

// NewBaseToolAgent creates a new base tool agent
func NewBaseToolAgent(config BaseAgentConfig, provider interfaces.AIProviderWithTools) *BaseToolAgent {
	return &BaseToolAgent{
		BaseAgent: NewBaseAgent(config),
		provider:  provider,
	}
}

// ExecuteWithTools implements tool-enabled execution
func (a *BaseToolAgent) ExecuteWithTools(ctx context.Context, input string, availableTools []interfaces.ToolDefinition) (string, []interfaces.ToolCall, error) {
	// Build messages with system prompt and user input
	messages := []interfaces.ChatMessage{
		{
			Role:    "system",
			Content: a.config.SystemPrompt,
		},
		{
			Role:    "user",
			Content: input,
		},
	}

	// Create request with tools
	req := interfaces.ChatRequestWithTools{
		ChatRequest: interfaces.ChatRequest{
			Model:       a.config.Model,
			Messages:    messages,
			MaxTokens:   a.config.MaxTokens,
			Temperature: a.config.Temperature,
		},
		Tools:      availableTools,
		ToolChoice: interfaces.ToolChoiceAuto,
	}

	// Execute with provider
	resp, err := a.provider.ChatCompletionWithTools(ctx, req)
	if err != nil {
		return "", nil, err
	}

	// Extract response and tool calls
	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, resp.ToolCalls, nil
	}

	return "", nil, nil
}

// ContinueWithToolResult continues the conversation after tool execution
func (a *BaseToolAgent) ContinueWithToolResult(ctx context.Context, toolCallID string, result string) (string, error) {
	// For now, just acknowledge the tool result
	// In a full implementation, this would continue the conversation
	return "I've received the tool result and will continue based on that information.", nil
}
