// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// universalParser implements UniversalParser to handle multiple provider formats
type universalParser struct {
	parsers map[ProviderFormat]ToolCallParser
	mu      sync.RWMutex
}

// NewUniversalParser creates a new universal parser with default parsers
func NewUniversalParser() UniversalParser {
	up := &universalParser{
		parsers: make(map[ProviderFormat]ToolCallParser),
	}
	
	// Register default parsers
	up.RegisterParser(FormatOpenAI, NewOpenAIParser())
	up.RegisterParser(FormatAnthropic, NewAnthropicParser())
	up.RegisterParser(FormatOllama, NewOllamaParser())
	up.RegisterParser(FormatMistral, NewMistralParser())
	up.RegisterParser(FormatGoogleAI, NewGoogleParser())
	
	return up
}

// ParseResponse attempts to parse with auto-detection
func (up *universalParser) ParseResponse(response string) ([]ToolCall, error) {
	format, err := up.DetectFormat(response)
	if err != nil {
		// If no tool calls were detected, return the original error
		if gerror.Is(err, ErrNoToolCalls) {
			return nil, err
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect format").
			WithComponent("UniversalParser").
			WithOperation("ParseResponse")
	}
	
	return up.ParseWithFormat(response, format)
}

// ParseWithFormat parses using a specific format
func (up *universalParser) ParseWithFormat(response string, format ProviderFormat) ([]ToolCall, error) {
	up.mu.RLock()
	parser, exists := up.parsers[format]
	up.mu.RUnlock()
	
	if !exists {
		return nil, gerror.Wrap(ErrParserNotFound, gerror.ErrCodeNotFound, "parser not found").
			WithComponent("UniversalParser").
			WithOperation("ParseWithFormat").
			WithDetails("format", string(format))
	}
	
	return parser.ParseResponse(response)
}

// HasToolCalls checks if any parser detects tool calls
func (up *universalParser) HasToolCalls(response string) bool {
	up.mu.RLock()
	defer up.mu.RUnlock()
	
	// Try each parser to see if any detect tool calls
	for _, parser := range up.parsers {
		if parser.HasToolCalls(response) {
			return true
		}
	}
	
	return false
}

// DetectFormat attempts to detect the format from the response
func (up *universalParser) DetectFormat(response string) (ProviderFormat, error) {
	trimmed := strings.TrimSpace(response)
	
	// Check for Anthropic XML format
	if strings.Contains(trimmed, "<function_calls>") || strings.Contains(trimmed, "<invoke name=") {
		return FormatAnthropic, nil
	}
	
	// Check for OpenAI JSON format - but only if it's valid JSON
	if strings.Contains(trimmed, `"tool_calls"`) || strings.Contains(trimmed, `"function_call"`) {
		// Quick check if it looks like JSON
		if json.Valid([]byte(trimmed)) {
			return FormatOpenAI, nil
		}
	}
	
	// Check for Ollama format (similar to OpenAI but may have variations)
	if strings.Contains(trimmed, `"tools"`) && strings.Contains(trimmed, `"function"`) {
		return FormatOllama, nil
	}
	
	// Check for Google/Gemini format
	if strings.Contains(trimmed, `"functionCall"`) || strings.Contains(trimmed, `"function_call"`) {
		return FormatGoogleAI, nil
	}
	
	// Check for Mistral format
	if strings.Contains(trimmed, `"tool_calls"`) && strings.Contains(trimmed, `"mistral"`) {
		return FormatMistral, nil
	}
	
	// Try each parser to see if any can handle it
	up.mu.RLock()
	defer up.mu.RUnlock()
	
	for format, parser := range up.parsers {
		if parser.HasToolCalls(response) {
			return format, nil
		}
	}
	
	return "", gerror.Wrap(ErrNoToolCalls, gerror.ErrCodeNotFound, "no recognizable tool call format detected").
		WithComponent("UniversalParser").
		WithOperation("DetectFormat")
}

// SupportedFormat returns auto since this handles all formats
func (up *universalParser) SupportedFormat() ProviderFormat {
	return FormatAutoDetect
}

// RegisterParser adds a parser for a specific format
func (up *universalParser) RegisterParser(format ProviderFormat, parser ToolCallParser) error {
	if parser == nil {
		return gerror.New(gerror.ErrCodeValidation, "parser cannot be nil", nil).
			WithComponent("UniversalParser").
			WithOperation("RegisterParser")
	}
	
	up.mu.Lock()
	defer up.mu.Unlock()
	
	up.parsers[format] = parser
	return nil
}

// universalFormatter implements UniversalFormatter to handle multiple provider formats
type universalFormatter struct {
	formatters map[ProviderFormat]ToolFormatter
	mu         sync.RWMutex
}

// NewUniversalFormatter creates a new universal formatter with default formatters
func NewUniversalFormatter() UniversalFormatter {
	uf := &universalFormatter{
		formatters: make(map[ProviderFormat]ToolFormatter),
	}
	
	// Register default formatters
	uf.RegisterFormatter(FormatOpenAI, NewOpenAIFormatter())
	uf.RegisterFormatter(FormatAnthropic, NewAnthropicFormatter())
	uf.RegisterFormatter(FormatOllama, NewOllamaFormatter())
	uf.RegisterFormatter(FormatMistral, NewMistralFormatter())
	uf.RegisterFormatter(FormatGoogleAI, NewGoogleFormatter())
	
	return uf
}

// FormatToolDefinitions uses auto-detection (returns generic format)
func (uf *universalFormatter) FormatToolDefinitions(tools []ToolDefinition) interface{} {
	// Return a generic format that includes all common fields
	// Specific providers will use FormatWithProvider
	return tools
}

// FormatToolResult formats a result generically
func (uf *universalFormatter) FormatToolResult(result *ToolResult) string {
	if !result.Success {
		return "Error: " + result.Error
	}
	
	// Convert output to string representation
	switch v := result.Output.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		// Use JSON for complex types
		data, _ := json.Marshal(v)
		return string(data)
	}
}

// FormatWithProvider formats tools for a specific provider
func (uf *universalFormatter) FormatWithProvider(tools []ToolDefinition, format ProviderFormat) interface{} {
	uf.mu.RLock()
	formatter, exists := uf.formatters[format]
	uf.mu.RUnlock()
	
	if !exists {
		// Return generic format if no specific formatter
		return tools
	}
	
	return formatter.FormatToolDefinitions(tools)
}

// SupportedFormat returns auto since this handles all formats
func (uf *universalFormatter) SupportedFormat() ProviderFormat {
	return FormatAutoDetect
}

// RegisterFormatter adds a formatter for a specific format
func (uf *universalFormatter) RegisterFormatter(format ProviderFormat, formatter ToolFormatter) error {
	if formatter == nil {
		return gerror.New(gerror.ErrCodeValidation, "formatter cannot be nil", nil).
			WithComponent("UniversalFormatter").
			WithOperation("RegisterFormatter")
	}
	
	uf.mu.Lock()
	defer uf.mu.Unlock()
	
	uf.formatters[format] = formatter
	return nil
}