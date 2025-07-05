// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package anthropic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/providers/interfaces"
	"github.com/lancekrogers/guild/pkg/tools/parser"
)

// Internal parser instance
var responseParser = parser.NewResponseParser()

// ChatCompletionWithTools implements tool-enabled chat completion for Anthropic
func (c *Client) ChatCompletionWithTools(ctx context.Context, req interfaces.ChatRequestWithTools) (*interfaces.ChatResponseWithTools, error) {
	// Check context early
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "chat completion cancelled").
			WithComponent("providers.anthropic").
			WithOperation("ChatCompletionWithTools")
	}

	logger := observability.GetLogger(ctx).
		WithComponent("providers.anthropic").
		WithOperation("ChatCompletionWithTools").
		With("model", req.Model).
		With("message_count", len(req.Messages)).
		With("tool_count", len(req.Tools))

	// Convert messages to Anthropic format
	anthropicMessages := make([]map[string]interface{}, 0)
	systemPrompt := ""

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			// For now, we'll handle regular messages only
			// TODO: Add support for tool response messages
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	start := time.Now()
	logger.InfoContext(ctx, "Starting Anthropic tool-enabled chat completion",
		"has_tools", len(req.Tools) > 0,
		"tool_choice", req.ToolChoice,
	)

	// Build Anthropic request
	anthropicReq := map[string]interface{}{
		"model":      req.Model,
		"messages":   anthropicMessages,
		"max_tokens": 4096,
	}

	if systemPrompt != "" {
		anthropicReq["system"] = systemPrompt
	}

	if req.MaxTokens > 0 {
		anthropicReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		anthropicTools := c.convertToolsToAnthropicFormat(req.Tools)
		anthropicReq["tools"] = anthropicTools
		
		// Handle tool choice
		if req.ToolChoice != nil {
			switch tc := req.ToolChoice.(type) {
			case interfaces.ToolChoice:
				switch tc {
				case interfaces.ToolChoiceAuto:
					anthropicReq["tool_choice"] = map[string]string{"type": "auto"}
				case interfaces.ToolChoiceNone:
					// Don't include tools if none
					delete(anthropicReq, "tools")
				case interfaces.ToolChoiceRequired:
					anthropicReq["tool_choice"] = map[string]string{"type": "any"}
				}
			case string:
				// Specific tool name
				anthropicReq["tool_choice"] = map[string]string{"type": "tool", "name": tc}
			}
		}
	}

	// Make request
	respBody, err := c.makeRequest(ctx, "messages", anthropicReq)
	if err != nil {
		duration := time.Since(start)
		logger.WithError(err).ErrorContext(ctx, "Anthropic API request failed",
			"duration_ms", duration.Milliseconds(),
		)
		return nil, err
	}

	// Parse Anthropic response
	var anthropicResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text,omitempty"`
			ID    string          `json:"id,omitempty"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProviderAPI, "failed to parse Anthropic response").
			WithComponent("providers.anthropic").
			WithOperation("ChatCompletionWithTools")
	}

	// Convert to our format
	content := ""
	var toolCalls []interfaces.ToolCall

	for _, c := range anthropicResp.Content {
		switch c.Type {
		case "text":
			content += c.Text
		case "tool_use":
			// Anthropic returns tool calls in the content
			toolCall := interfaces.ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: interfaces.Function{
					Name:      c.Name,
					Arguments: c.Input,
				},
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	// Also check if there are tool calls in the text content (XML format)
	if len(toolCalls) == 0 && content != "" {
		// Try to parse tool calls from the content
		if parsedCalls, err := responseParser.ExtractToolCalls(content); err == nil && len(parsedCalls) > 0 {
			// Convert parser format to interface format
			for _, pc := range parsedCalls {
				toolCall := interfaces.ToolCall{
					ID:       pc.ID,
					Type:     pc.Type,
					Function: interfaces.Function{
						Name:      pc.Function.Name,
						Arguments: pc.Function.Arguments,
					},
				}
				toolCalls = append(toolCalls, toolCall)
			}
		}
	}

	response := &interfaces.ChatResponseWithTools{
		ChatResponse: interfaces.ChatResponse{
			ID:    anthropicResp.ID,
			Model: anthropicResp.Model,
			Choices: []interfaces.ChatChoice{
				{
					Index: 0,
					Message: interfaces.ChatMessage{
						Role:    "assistant",
						Content: content,
					},
					FinishReason: anthropicResp.StopReason,
				},
			},
			Usage: interfaces.UsageInfo{
				PromptTokens:     anthropicResp.Usage.InputTokens,
				CompletionTokens: anthropicResp.Usage.OutputTokens,
				TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
			},
			FinishReason: anthropicResp.StopReason,
		},
		ToolCalls: toolCalls,
	}

	duration := time.Since(start)
	logger.InfoContext(ctx, "Anthropic tool-enabled chat completion successful",
		"duration_ms", duration.Milliseconds(),
		"tool_calls_count", len(toolCalls),
		"tokens_used", response.Usage.TotalTokens,
	)

	return response, nil
}

// convertToolsToAnthropicFormat converts standard tool definitions to Anthropic's format
func (c *Client) convertToolsToAnthropicFormat(tools []interfaces.ToolDefinition) []map[string]interface{} {
	anthropicTools := make([]map[string]interface{}, len(tools))
	
	for i, tool := range tools {
		anthropicTool := map[string]interface{}{
			"name":        tool.Function.Name,
			"description": tool.Function.Description,
		}
		
		// Parse and include the parameters schema
		if len(tool.Function.Parameters) > 0 {
			var schema map[string]interface{}
			if err := json.Unmarshal(tool.Function.Parameters, &schema); err == nil {
				anthropicTool["input_schema"] = schema
			}
		}
		
		anthropicTools[i] = anthropicTool
	}
	
	return anthropicTools
}

// StreamChatCompletionWithTools implements streaming with tool support
func (c *Client) StreamChatCompletionWithTools(ctx context.Context, req interfaces.ChatRequestWithTools) (interfaces.ChatStreamWithTools, error) {
	// For now, Anthropic doesn't support streaming with tools in the same way
	// We'll implement this when Anthropic adds proper streaming tool support
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "streaming with tools not yet implemented for Anthropic", nil).
		WithComponent("providers.anthropic").
		WithOperation("StreamChatCompletionWithTools")
}