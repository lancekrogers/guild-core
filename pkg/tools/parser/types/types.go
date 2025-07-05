// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package types

import (
	"context"
	"encoding/json"
)

// ProviderFormat represents different LLM provider tool call formats
type ProviderFormat string

const (
	ProviderFormatOpenAI    ProviderFormat = "openai"
	ProviderFormatAnthropic ProviderFormat = "anthropic" 
	ProviderFormatUnknown   ProviderFormat = "unknown"
)

// ToolCall represents a standardized tool call structure
type ToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function FunctionCall           `json:"function"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// FunctionCall represents the function being called
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// DetectionResult contains format detection results with confidence scoring
type DetectionResult struct {
	Format     ProviderFormat         `json:"format"`
	Confidence float64                `json:"confidence"` // 0.0 to 1.0
	Metadata   map[string]interface{} `json:"metadata"`
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid      bool               `json:"valid"`
	Errors     []ValidationError  `json:"errors"`
	Warnings   []string           `json:"warnings"`
	SchemaUsed string             `json:"schema_used"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Path    string      `json:"path"`
	Message string      `json:"message"`
	Code    string      `json:"code"`
	Value   interface{} `json:"value,omitempty"`
}

// FormatDetector identifies and scores tool call formats
type FormatDetector interface {
	// Detect analyzes input and returns detection result with confidence
	Detect(ctx context.Context, input []byte) (DetectionResult, error)
	
	// CanParse performs quick pre-check without full parsing
	CanParse(input []byte) bool
	
	// Format returns the format this detector handles
	Format() ProviderFormat
}

// FormatParser extracts and validates tool calls from a specific format
type FormatParser interface {
	// Parse extracts tool calls from input
	Parse(ctx context.Context, input []byte) ([]ToolCall, error)
	
	// Validate checks if input conforms to expected schema
	Validate(input []byte) ValidationResult
	
	// Format returns the format this parser handles
	Format() ProviderFormat
}

// ResponseParser is the main interface for parsing tool calls from any format
type ResponseParser interface {
	// ExtractToolCalls parses tool calls from any supported format
	ExtractToolCalls(response string) ([]ToolCall, error)
	
	// ExtractWithContext parses with context support for cancellation and timeouts
	ExtractWithContext(ctx context.Context, response string) ([]ToolCall, error)
	
	// DetectFormat identifies the format with confidence score
	DetectFormat(response string) (ProviderFormat, float64, error)
}