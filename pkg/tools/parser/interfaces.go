// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"encoding/json"
	"time"

	"github.com/guild-framework/guild-core/pkg/tools/parser/types"
)

// Re-export types from the types package for convenience and backward compatibility.
// These type aliases allow users to import all types from the parser package
// without needing to import the types subpackage directly.

// ProviderFormat represents different LLM provider tool call formats.
// Each provider (OpenAI, Anthropic, etc.) may use a different format
// for encoding function calls in their responses.
type ProviderFormat = types.ProviderFormat

// ToolCall represents a standardized tool/function call extracted from
// an LLM response. This is the common format that all provider-specific
// formats are converted to, ensuring consistent handling across providers.
type ToolCall = types.ToolCall

// FunctionCall contains the details of a function to be called,
// including its name and JSON-encoded arguments. The arguments are
// stored as json.RawMessage to preserve the exact JSON structure.
type FunctionCall = types.FunctionCall

// DetectionResult contains the results of format detection, including
// the detected format, confidence score (0.0-1.0), and additional
// metadata about the detection process.
type DetectionResult = types.DetectionResult

// ValidationResult contains the results of validating input against
// a specific format schema, including any errors, warnings, and
// metadata about the validation process.
type ValidationResult = types.ValidationResult

// ValidationError represents a specific validation error found during
// format validation, including the path to the error, message, and
// error code for programmatic handling.
type ValidationError = types.ValidationError

// FormatDetector analyzes input to determine if it contains tool calls
// in a specific format and provides confidence scoring.
type FormatDetector = types.FormatDetector

// FormatParser extracts tool calls from input in a specific format
// and provides validation capabilities.
type FormatParser = types.FormatParser

// ResponseParser is the main interface for parsing tool calls from
// LLM responses, supporting automatic format detection and multiple
// provider formats.
type ResponseParser = types.ResponseParser

// Re-export constants
const (
	ProviderFormatOpenAI    = types.ProviderFormatOpenAI
	ProviderFormatAnthropic = types.ProviderFormatAnthropic
	ProviderFormatUnknown   = types.ProviderFormatUnknown
)

// ToolDefinition represents a tool that can be called by an LLM.
// It follows the OpenAI function calling schema for compatibility.
type ToolDefinition struct {
	// Type is the type of tool (usually "function")
	Type string `json:"type"`
	// Function contains the function definition
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition defines a callable function with its metadata.
// This structure is used to describe tools to LLMs so they know
// how to call them correctly.
type FunctionDefinition struct {
	// Name is the function name that will be used in tool calls
	Name string `json:"name"`
	// Description explains what the function does (shown to LLM)
	Description string `json:"description"`
	// Parameters is a JSON Schema defining the function parameters
	Parameters json.RawMessage `json:"parameters"`
}

// ToolResult represents the result of executing a tool call.
// It includes both success and error cases with detailed information
// about the execution.
type ToolResult struct {
	// ID matches the tool call ID that was executed
	ID string `json:"id"`
	// Success indicates if the tool execution succeeded
	Success bool `json:"success"`
	// Content is the main result text (for success cases)
	Content string `json:"content,omitempty"`
	// Error contains error message (for failure cases)
	Error string `json:"error,omitempty"`
	// Output contains structured output data
	Output map[string]interface{} `json:"output,omitempty"`
	// Duration is how long the execution took
	Duration time.Duration `json:"duration,omitempty"`
	// Metadata contains additional execution information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExecutor executes tool calls extracted by the parser.
// Implementations should handle tool registration, validation,
// and execution with proper error handling and timeouts.
type ToolExecutor interface {
	// Execute runs a single tool call and returns the result.
	// It should validate the tool exists and arguments are valid
	// before execution.
	Execute(ctx context.Context, call ToolCall) (*ToolResult, error)

	// ExecuteBatch runs multiple tool calls, potentially in parallel.
	// Implementations should respect context cancellation and may
	// limit concurrency for resource management.
	ExecuteBatch(ctx context.Context, calls []ToolCall) ([]*ToolResult, error)

	// GetAvailableTools returns all registered tools as standardized
	// definitions that can be passed to LLMs.
	GetAvailableTools() []ToolDefinition
}

// ParserOption configures parser behavior. Options can be passed to
// NewResponseParser to customize parsing behavior.
type ParserOption func(*parserConfig)

// parserConfig holds parser configuration
type parserConfig struct {
	maxInputSize     int
	enableFuzzyMatch bool
	strictValidation bool
	timeout          time.Duration
	customDetectors  []FormatDetector
	customParsers    map[ProviderFormat]FormatParser
}

// WithMaxInputSize sets the maximum input size to process.
// Inputs larger than this will be rejected with an error.
// Default is 10MB.
//
// Example:
//
//	parser := NewResponseParser(WithMaxInputSize(5 * 1024 * 1024)) // 5MB
func WithMaxInputSize(size int) ParserOption {
	return func(c *parserConfig) {
		c.maxInputSize = size
	}
}

// WithStrictValidation enables strict schema validation.
// When enabled, the parser will validate tool calls against
// strict schemas and may reject calls that don't conform exactly.
// Default is false (lenient mode).
//
// Example:
//
//	parser := NewResponseParser(WithStrictValidation(true))
func WithStrictValidation(strict bool) ParserOption {
	return func(c *parserConfig) {
		c.strictValidation = strict
	}
}

// WithTimeout sets the parsing timeout for operations.
// This prevents the parser from hanging on malformed inputs.
// Default is 5 seconds.
//
// Example:
//
//	parser := NewResponseParser(WithTimeout(10 * time.Second))
func WithTimeout(timeout time.Duration) ParserOption {
	return func(c *parserConfig) {
		c.timeout = timeout
	}
}

// WithEnableFuzzyMatch enables fuzzy matching for format detection.
// This can help detect formats in heavily mixed content but may
// increase false positives. Default is true.
//
// Example:
//
//	parser := NewResponseParser(WithEnableFuzzyMatch(false))
func WithEnableFuzzyMatch(enabled bool) ParserOption {
	return func(c *parserConfig) {
		c.enableFuzzyMatch = enabled
	}
}

// WithCustomDetector adds a custom format detector to the parser.
// This allows extending the parser to support new formats.
//
// Example:
//
//	detector := &MyCustomDetector{}
//	parser := NewResponseParser(WithCustomDetector(detector))
func WithCustomDetector(detector FormatDetector) ParserOption {
	return func(c *parserConfig) {
		c.customDetectors = append(c.customDetectors, detector)
	}
}

// WithCustomParser adds a custom format parser for a specific format.
// This allows extending the parser to handle new formats.
//
// Example:
//
//	customParser := &MyCustomParser{}
//	parser := NewResponseParser(
//	  WithCustomParser("myformat", customParser),
//	)
func WithCustomParser(format ProviderFormat, parser FormatParser) ParserOption {
	return func(c *parserConfig) {
		if c.customParsers == nil {
			c.customParsers = make(map[ProviderFormat]FormatParser)
		}
		c.customParsers[format] = parser
	}
}
