// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lancekrogers/guild/pkg/tools/parser/types"
)

// Re-export types from the types package for convenience
type (
	ProviderFormat      = types.ProviderFormat
	ToolCall           = types.ToolCall
	FunctionCall       = types.FunctionCall
	DetectionResult    = types.DetectionResult
	ValidationResult   = types.ValidationResult
	ValidationError    = types.ValidationError
	FormatDetector     = types.FormatDetector
	FormatParser       = types.FormatParser
	ResponseParser     = types.ResponseParser
)

// Re-export constants
const (
	ProviderFormatOpenAI    = types.ProviderFormatOpenAI
	ProviderFormatAnthropic = types.ProviderFormatAnthropic
	ProviderFormatUnknown   = types.ProviderFormatUnknown
)


// ToolDefinition represents a tool that can be called
type ToolDefinition struct {
	Type     string              `json:"type"`
	Function FunctionDefinition  `json:"function"`
}

// FunctionDefinition defines a callable function
type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// ToolResult represents the result of tool execution
type ToolResult struct {
	ID       string                 `json:"id"`
	Success  bool                   `json:"success"`
	Content  string                 `json:"content,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Output   map[string]interface{} `json:"output,omitempty"`
	Duration time.Duration          `json:"duration,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExecutor executes tool calls
type ToolExecutor interface {
	// Execute runs a single tool call
	Execute(ctx context.Context, call ToolCall) (*ToolResult, error)
	
	// ExecuteBatch runs multiple tool calls, potentially in parallel
	ExecuteBatch(ctx context.Context, calls []ToolCall) ([]*ToolResult, error)
	
	// GetAvailableTools returns all registered tools as standardized definitions
	GetAvailableTools() []ToolDefinition
}


// ParserOption configures parser behavior
type ParserOption func(*parserConfig)

// parserConfig holds parser configuration
type parserConfig struct {
	maxInputSize      int
	enableFuzzyMatch  bool
	strictValidation  bool
	timeout           time.Duration
	customDetectors   []FormatDetector
	customParsers     map[ProviderFormat]FormatParser
}

// WithMaxInputSize sets the maximum input size to process
func WithMaxInputSize(size int) ParserOption {
	return func(c *parserConfig) {
		c.maxInputSize = size
	}
}

// WithStrictValidation enables strict schema validation
func WithStrictValidation(strict bool) ParserOption {
	return func(c *parserConfig) {
		c.strictValidation = strict
	}
}

// WithTimeout sets the parsing timeout
func WithTimeout(timeout time.Duration) ParserOption {
	return func(c *parserConfig) {
		c.timeout = timeout
	}
}

// WithCustomDetector adds a custom format detector
func WithCustomDetector(detector FormatDetector) ParserOption {
	return func(c *parserConfig) {
		c.customDetectors = append(c.customDetectors, detector)
	}
}

// WithCustomParser adds a custom format parser
func WithCustomParser(format ProviderFormat, parser FormatParser) ParserOption {
	return func(c *parserConfig) {
		if c.customParsers == nil {
			c.customParsers = make(map[ProviderFormat]FormatParser)
		}
		c.customParsers[format] = parser
	}
}