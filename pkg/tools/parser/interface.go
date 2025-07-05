// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ToolCall represents a standardized tool invocation request
// We use OpenAI's format as the internal standard since it's becoming industry standard
type ToolCall struct {
	ID       string    `json:"id"`
	Type     string    `json:"type"` // "function" for function calls
	Function Function  `json:"function"`
}

// Function represents the function to call
type Function struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"` // JSON string of arguments
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
	ID       string          `json:"tool_call_id"`
	Content  string          `json:"content"`
	Success  bool            `json:"success"`
	Error    string          `json:"error,omitempty"`
	Duration time.Duration   `json:"duration,omitempty"`
	Output   interface{}     `json:"output,omitempty"` // Raw output data
}

// ToolDefinition represents a tool that can be called by an LLM
// Using OpenAI's format as the standard
type ToolDefinition struct {
	Type     string                 `json:"type"` // "function"
	Function FunctionDefinition     `json:"function"`
}

// FunctionDefinition defines a function that can be called
type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// ResponseParser extracts tool calls from LLM responses
type ResponseParser interface {
	// ExtractToolCalls finds and extracts tool calls from a response
	// Returns nil if no tool calls are found
	ExtractToolCalls(response string) ([]ToolCall, error)
	
	// ContainsToolCalls quickly checks if response has tool calls
	ContainsToolCalls(response string) bool
}

// ToolExecutor executes parsed tool calls
type ToolExecutor interface {
	// Execute runs a single tool call
	Execute(ctx context.Context, call ToolCall) (*ToolResult, error)
	
	// ExecuteBatch runs multiple tool calls (potentially in parallel)
	ExecuteBatch(ctx context.Context, calls []ToolCall) ([]*ToolResult, error)
	
	// GetAvailableTools returns all registered tools as definitions
	GetAvailableTools() []ToolDefinition
}

// ProviderAdapter adapts between provider-specific formats and our standard format
type ProviderAdapter interface {
	// ToProviderFormat converts our standard tool definitions to provider-specific format
	ToProviderFormat(tools []ToolDefinition) interface{}
	
	// FromProviderResponse extracts tool calls from provider-specific response format
	FromProviderResponse(response interface{}) ([]ToolCall, error)
	
	// FormatResult converts our standard result to provider-specific format
	FormatResult(result *ToolResult) interface{}
}

// ProviderFormat represents different LLM provider response formats
type ProviderFormat string

const (
	FormatOpenAI     ProviderFormat = "openai"
	FormatAnthropic  ProviderFormat = "anthropic"
	FormatOllama     ProviderFormat = "ollama"
	FormatMistral    ProviderFormat = "mistral"
	FormatGoogleAI   ProviderFormat = "google"
	FormatAutoDetect ProviderFormat = "auto"
)

// ToolCallParser parses tool calls from provider responses
type ToolCallParser interface {
	// ParseResponse extracts tool calls from a response string
	ParseResponse(response string) ([]ToolCall, error)
	
	// HasToolCalls checks if the response contains tool calls
	HasToolCalls(response string) bool
	
	// SupportedFormat returns the provider format this parser handles
	SupportedFormat() ProviderFormat
}

// ToolFormatter formats tool definitions for providers
type ToolFormatter interface {
	// FormatToolDefinitions converts tools to provider-specific format
	FormatToolDefinitions(tools []ToolDefinition) interface{}
	
	// FormatToolResult formats a result for the provider
	FormatToolResult(result *ToolResult) string
	
	// SupportedFormat returns the provider format this formatter handles
	SupportedFormat() ProviderFormat
}

// UniversalParser can handle multiple provider formats
type UniversalParser interface {
	ToolCallParser
	
	// ParseWithFormat parses using a specific format
	ParseWithFormat(response string, format ProviderFormat) ([]ToolCall, error)
	
	// DetectFormat attempts to detect the format from the response
	DetectFormat(response string) (ProviderFormat, error)
	
	// RegisterParser adds a parser for a specific format
	RegisterParser(format ProviderFormat, parser ToolCallParser) error
}

// UniversalFormatter can format for multiple providers
type UniversalFormatter interface {
	ToolFormatter
	
	// FormatWithProvider formats tools for a specific provider
	FormatWithProvider(tools []ToolDefinition, format ProviderFormat) interface{}
	
	// RegisterFormatter adds a formatter for a specific format
	RegisterFormatter(format ProviderFormat, formatter ToolFormatter) error
}

// Common errors
var (
	ErrNoToolCalls     = errors.New("no tool calls found in response")
	ErrInvalidFormat   = errors.New("invalid response format")
	ErrParserNotFound  = errors.New("parser not found for format")
	ErrInvalidToolCall = errors.New("invalid tool call structure")
)