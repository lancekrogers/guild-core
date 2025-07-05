// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package interfaces

import (
	"context"
	"encoding/json"
)

// ToolDefinition represents a tool that can be called by the AI
type ToolDefinition struct {
	Type     string              `json:"type"` // "function"
	Function FunctionDefinition  `json:"function"`
}

// FunctionDefinition defines a function that can be called
type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// ToolCall represents a tool invocation in a response
type ToolCall struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"` // "function"
	Function Function  `json:"function"`
}

// Function represents the function to call
type Function struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"` // JSON string of arguments
}

// ToolChoice controls which tools can be called
type ToolChoice string

const (
	ToolChoiceAuto     ToolChoice = "auto"     // Model decides
	ToolChoiceNone     ToolChoice = "none"     // No tools
	ToolChoiceRequired ToolChoice = "required" // Must use a tool
)

// ChatRequestWithTools extends ChatRequest with tool support
type ChatRequestWithTools struct {
	ChatRequest
	Tools      []ToolDefinition `json:"tools,omitempty"`
	ToolChoice interface{}      `json:"tool_choice,omitempty"` // ToolChoice or specific tool
}

// ChatResponseWithTools extends ChatResponse with tool calls
type ChatResponseWithTools struct {
	ChatResponse
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChatMessageWithToolCalls extends ChatMessage for tool responses
type ChatMessageWithToolCalls struct {
	ChatMessage
	ToolCalls   []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID  string     `json:"tool_call_id,omitempty"`  // For tool responses
}

// AIProviderWithTools extends AIProvider with tool support
type AIProviderWithTools interface {
	AIProvider
	ChatCompletionWithTools(ctx context.Context, req ChatRequestWithTools) (*ChatResponseWithTools, error)
	StreamChatCompletionWithTools(ctx context.Context, req ChatRequestWithTools) (ChatStreamWithTools, error)
}

// ChatStreamWithTools extends ChatStream for tool support
type ChatStreamWithTools interface {
	ChatStream
	NextWithTools() (ChatStreamChunkWithTools, error)
}

// ChatStreamChunkWithTools extends ChatStreamChunk with tool calls
type ChatStreamChunkWithTools struct {
	ChatStreamChunk
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}