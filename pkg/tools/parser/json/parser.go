// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package json

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/tools/parser/types"
)

// Parser parses OpenAI-style JSON tool calls
type Parser struct {
	// Options
	strictMode      bool
	fixCommonErrors bool
}

// NewParser creates a new JSON format parser
func NewParser() types.FormatParser {
	return &Parser{
		strictMode:      false,
		fixCommonErrors: true,
	}
}

// Format returns the format this parser handles
func (p *Parser) Format() types.ProviderFormat {
	return types.ProviderFormatOpenAI
}

// Parse extracts tool calls from JSON input
func (p *Parser) Parse(ctx context.Context, input []byte) ([]types.ToolCall, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("parser").
		WithOperation("JSONParser.Parse")

	// Try to parse as complete JSON first
	calls, err := p.parseCompleteJSON(input)
	if err == nil {
		return calls, nil
	}

	logger.Debug("Complete JSON parse failed, trying extraction",
		"error", err,
	)

	// Try to extract JSON from mixed content
	detector := NewDetector()
	jsonData, location := detector.extractJSON(input)
	if jsonData == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no valid JSON found", nil).
			WithComponent("parser").
			WithOperation("JSONParser.Parse")
	}

	logger.Debug("Extracted JSON",
		"location", location,
		"size", len(jsonData),
	)

	// Parse extracted JSON
	calls, err = p.parseCompleteJSON(jsonData)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse extracted JSON").
			WithComponent("parser").
			WithOperation("JSONParser.Parse")
	}

	return calls, nil
}

// parseCompleteJSON parses a complete JSON document
func (p *Parser) parseCompleteJSON(input []byte) ([]types.ToolCall, error) {
	// Decode to preserve structure
	var doc interface{}
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber() // Preserve number precision

	if err := decoder.Decode(&doc); err != nil {
		// Try to fix common errors if enabled
		if p.fixCommonErrors {
			fixed := p.tryFixJSON(input)
			if fixed != nil {
				decoder = json.NewDecoder(bytes.NewReader(fixed))
				decoder.UseNumber()
				if err := decoder.Decode(&doc); err == nil {
					input = fixed
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Extract tool calls based on structure
	switch v := doc.(type) {
	case map[string]interface{}:
		return p.extractFromObject(v)
	case []interface{}:
		return p.extractFromArray(v)
	default:
		return nil, gerror.New(gerror.ErrCodeValidation, "unexpected JSON structure", nil).
			WithComponent("parser").
			WithOperation("parseCompleteJSON").
			WithDetails("type", fmt.Sprintf("%T", doc))
	}
}

// extractFromObject extracts tool calls from a JSON object
func (p *Parser) extractFromObject(obj map[string]interface{}) ([]types.ToolCall, error) {
	// Check if this object itself is a tool call
	if _, hasID := obj["id"]; hasID {
		if _, hasType := obj["type"]; hasType {
			if _, hasFunc := obj["function"]; hasFunc {
				// This is a single tool call
				if call := p.parseToolCall(obj); call != nil {
					return []types.ToolCall{*call}, nil
				}
			}
		}
	}

	// Check for tool_calls array
	if toolCallsRaw, exists := obj["tool_calls"]; exists {
		if toolCallsArr, ok := toolCallsRaw.([]interface{}); ok {
			return p.parseToolCallsArray(toolCallsArr)
		}
	}

	// Check for single function_call (older format)
	if funcCallRaw, exists := obj["function_call"]; exists {
		if funcCall, ok := funcCallRaw.(map[string]interface{}); ok {
			call := p.parseSingleFunctionCall(funcCall)
			if call != nil {
				return []types.ToolCall{*call}, nil
			}
		}
	}

	// Check for assistant message format
	if role, _ := obj["role"].(string); role == "assistant" {
		// Recursively check for tool calls in assistant message
		return p.extractFromObject(obj)
	}

	// Check nested structures
	for _, value := range obj {
		switch v := value.(type) {
		case map[string]interface{}:
			if calls, err := p.extractFromObject(v); err == nil && len(calls) > 0 {
				return calls, nil
			}
		case []interface{}:
			if calls, err := p.extractFromArray(v); err == nil && len(calls) > 0 {
				return calls, nil
			}
		}
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "no tool calls found in object", nil).
		WithComponent("parser").
		WithOperation("extractFromObject")
}

// extractFromArray extracts tool calls from a JSON array
func (p *Parser) extractFromArray(arr []interface{}) ([]types.ToolCall, error) {
	// Check if it's directly an array of tool calls
	if len(arr) > 0 {
		// Peek at first element
		if first, ok := arr[0].(map[string]interface{}); ok {
			// Check if it looks like a tool call
			if _, hasID := first["id"]; hasID {
				if _, hasFunc := first["function"]; hasFunc {
					return p.parseToolCallsArray(arr)
				}
			}
		}
	}

	// Otherwise, check each element for tool calls
	var allCalls []types.ToolCall
	for _, item := range arr {
		switch v := item.(type) {
		case map[string]interface{}:
			if calls, err := p.extractFromObject(v); err == nil {
				allCalls = append(allCalls, calls...)
			}
		}
	}

	if len(allCalls) > 0 {
		return allCalls, nil
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "no tool calls found in array", nil).
		WithComponent("parser").
		WithOperation("extractFromArray")
}

// parseToolCallsArray parses an array of tool call objects
func (p *Parser) parseToolCallsArray(arr []interface{}) ([]types.ToolCall, error) {
	var calls []types.ToolCall

	for i, item := range arr {
		callMap, ok := item.(map[string]interface{})
		if !ok {
			if p.strictMode {
				return nil, gerror.New(gerror.ErrCodeValidation, "invalid tool call type", nil).
					WithComponent("parser").
					WithOperation("parseToolCallsArray").
					WithDetails("index", i).
					WithDetails("type", fmt.Sprintf("%T", item))
			}
			continue
		}

		call := p.parseToolCall(callMap)
		if call != nil {
			calls = append(calls, *call)
		} else if p.strictMode {
			return nil, gerror.New(gerror.ErrCodeValidation, "invalid tool call structure", nil).
				WithComponent("parser").
				WithOperation("parseToolCallsArray").
				WithDetails("index", i)
		}
	}

	return calls, nil
}

// parseToolCall parses a single tool call object
func (p *Parser) parseToolCall(obj map[string]interface{}) *types.ToolCall {
	call := &types.ToolCall{
		Metadata: make(map[string]interface{}),
	}

	// Extract ID
	if id, ok := obj["id"].(string); ok {
		call.ID = id
	} else {
		// Generate ID if missing
		call.ID = fmt.Sprintf("call_%d", time.Now().UnixNano())
	}

	// Extract type
	if callType, ok := obj["type"].(string); ok {
		call.Type = callType
	} else {
		call.Type = "function" // Default
	}

	// Extract function details
	funcRaw, exists := obj["function"]
	if !exists {
		return nil
	}

	funcMap, ok := funcRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	// Extract function name
	if name, ok := funcMap["name"].(string); ok {
		call.Function.Name = name
	} else {
		return nil // Name is required
	}

	// Extract arguments
	if args := funcMap["arguments"]; args != nil {
		switch v := args.(type) {
		case string:
			// Arguments as JSON string
			call.Function.Arguments = json.RawMessage(v)
		case map[string]interface{}, []interface{}:
			// Arguments as object/array - marshal back to JSON
			data, err := json.Marshal(v)
			if err == nil {
				call.Function.Arguments = data
			}
		default:
			// Try to marshal whatever it is
			data, err := json.Marshal(v)
			if err == nil {
				call.Function.Arguments = data
			}
		}
	}

	// Ensure arguments is valid JSON
	if len(call.Function.Arguments) == 0 {
		call.Function.Arguments = json.RawMessage("{}")
	} else if !json.Valid(call.Function.Arguments) {
		// Try to fix
		if fixed := p.tryFixJSON(call.Function.Arguments); fixed != nil {
			call.Function.Arguments = fixed
		} else {
			call.Function.Arguments = json.RawMessage("{}")
		}
	}

	return call
}

// parseSingleFunctionCall parses older function_call format
func (p *Parser) parseSingleFunctionCall(funcCall map[string]interface{}) *types.ToolCall {
	call := &types.ToolCall{
		ID:       fmt.Sprintf("call_%d", time.Now().UnixNano()),
		Type:     "function",
		Metadata: make(map[string]interface{}),
	}

	// Extract name
	if name, ok := funcCall["name"].(string); ok {
		call.Function.Name = name
	} else {
		return nil
	}

	// Extract arguments (same logic as above)
	if args := funcCall["arguments"]; args != nil {
		switch v := args.(type) {
		case string:
			call.Function.Arguments = json.RawMessage(v)
		case map[string]interface{}, []interface{}:
			data, err := json.Marshal(v)
			if err == nil {
				call.Function.Arguments = data
			}
		}
	}

	if len(call.Function.Arguments) == 0 {
		call.Function.Arguments = json.RawMessage("{}")
	}

	return call
}

// Validate checks if input conforms to expected schema
func (p *Parser) Validate(input []byte) types.ValidationResult {
	result := types.ValidationResult{
		Valid:      true,
		Errors:     []types.ValidationError{},
		Warnings:   []string{},
		SchemaUsed: "openai_tool_calls_v1",
		Metadata:   make(map[string]interface{}),
	}

	// Basic JSON validation
	if !json.Valid(input) {
		result.Valid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    "$",
			Message: "Invalid JSON syntax",
			Code:    "invalid_json",
		})
		return result
	}

	// Parse and validate structure
	var doc interface{}
	if err := json.Unmarshal(input, &doc); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    "$",
			Message: err.Error(),
			Code:    "parse_error",
		})
		return result
	}

	// Validate based on structure type
	switch v := doc.(type) {
	case map[string]interface{}:
		p.validateObject(v, "$", &result)
	case []interface{}:
		p.validateArray(v, "$", &result)
	default:
		result.Valid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    "$",
			Message: "Expected object or array at root",
			Code:    "invalid_root_type",
		})
	}

	return result
}

// validateObject validates a JSON object for tool calls
func (p *Parser) validateObject(obj map[string]interface{}, path string, result *types.ValidationResult) {
	// Check for tool_calls
	if toolCalls, exists := obj["tool_calls"]; exists {
		if arr, ok := toolCalls.([]interface{}); ok {
			p.validateToolCallsArray(arr, path+".tool_calls", result)
		} else {
			result.Valid = false
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    path + ".tool_calls",
				Message: "tool_calls must be an array",
				Code:    "invalid_type",
			})
		}
	} else if funcCall, exists := obj["function_call"]; exists {
		// Validate function_call format
		if fc, ok := funcCall.(map[string]interface{}); ok {
			p.validateFunctionCall(fc, path+".function_call", result)
		}
	} else {
		result.Warnings = append(result.Warnings, "No tool_calls or function_call found")
	}
}

// validateArray validates an array of tool calls
func (p *Parser) validateArray(arr []interface{}, path string, result *types.ValidationResult) {
	if len(arr) == 0 {
		result.Warnings = append(result.Warnings, "Empty array")
		return
	}

	// Check if it's an array of tool calls
	for i, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			p.validateToolCall(obj, itemPath, result)
		}
	}
}

// validateToolCallsArray validates the tool_calls array
func (p *Parser) validateToolCallsArray(arr []interface{}, path string, result *types.ValidationResult) {
	for i, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			p.validateToolCall(obj, itemPath, result)
		} else {
			result.Valid = false
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    fmt.Sprintf("%s[%d]", path, i),
				Message: "Tool call must be an object",
				Code:    "invalid_type",
			})
		}
	}
}

// validateToolCall validates a single tool call
func (p *Parser) validateToolCall(obj map[string]interface{}, path string, result *types.ValidationResult) {
	// Check required fields
	if _, exists := obj["id"]; !exists {
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    path + ".id",
			Message: "Missing required field: id",
			Code:    "missing_required",
		})
	}

	if callType, exists := obj["type"]; exists {
		if t, ok := callType.(string); ok && t != "function" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Unexpected type at %s: %s", path+".type", t))
		}
	}

	// Validate function
	if function, exists := obj["function"]; exists {
		if funcObj, ok := function.(map[string]interface{}); ok {
			p.validateFunction(funcObj, path+".function", result)
		} else {
			result.Valid = false
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    path + ".function",
				Message: "function must be an object",
				Code:    "invalid_type",
			})
		}
	} else {
		result.Valid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    path + ".function",
			Message: "Missing required field: function",
			Code:    "missing_required",
		})
	}
}

// validateFunction validates function details
func (p *Parser) validateFunction(obj map[string]interface{}, path string, result *types.ValidationResult) {
	// Check name
	if name, exists := obj["name"]; exists {
		if n, ok := name.(string); ok {
			if n == "" {
				result.Errors = append(result.Errors, types.ValidationError{
					Path:    path + ".name",
					Message: "Function name cannot be empty",
					Code:    "empty_value",
				})
			}
		} else {
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    path + ".name",
				Message: "Function name must be a string",
				Code:    "invalid_type",
			})
		}
	} else {
		result.Valid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    path + ".name",
			Message: "Missing required field: name",
			Code:    "missing_required",
		})
	}

	// Check arguments
	if args, exists := obj["arguments"]; exists {
		switch v := args.(type) {
		case string:
			// Validate it's valid JSON
			if !json.Valid([]byte(v)) {
				result.Errors = append(result.Errors, types.ValidationError{
					Path:    path + ".arguments",
					Message: "Arguments string is not valid JSON",
					Code:    "invalid_json",
					Value:   v,
				})
			}
		case map[string]interface{}, []interface{}:
			// This is fine - already parsed
		default:
			result.Warnings = append(result.Warnings, fmt.Sprintf("Unexpected arguments type at %s: %T", path+".arguments", args))
		}
	}
}

// validateFunctionCall validates older function_call format
func (p *Parser) validateFunctionCall(obj map[string]interface{}, path string, result *types.ValidationResult) {
	p.validateFunction(obj, path, result)
}

// tryFixJSON attempts to fix common JSON errors
func (p *Parser) tryFixJSON(data json.RawMessage) json.RawMessage {
	s := string(data)

	// Try parsing as-is first
	var test interface{}
	if err := json.Unmarshal(data, &test); err == nil {
		return data
	}

	// Fix 1: Single quotes to double quotes
	fixed := strings.Replace(s, "'", `"`, -1)
	if json.Valid([]byte(fixed)) {
		return json.RawMessage(fixed)
	}

	// Fix 2: Remove trailing commas
	fixed = p.removeTrailingCommas(s)
	if json.Valid([]byte(fixed)) {
		return json.RawMessage(fixed)
	}

	// Fix 3: Escape unescaped quotes
	fixed = p.escapeUnescapedQuotes(s)
	if json.Valid([]byte(fixed)) {
		return json.RawMessage(fixed)
	}

	return nil
}

// removeTrailingCommas removes trailing commas from JSON
func (p *Parser) removeTrailingCommas(s string) string {
	// Simple regex-like approach
	result := s
	result = strings.Replace(result, ",]", "]", -1)
	result = strings.Replace(result, ",}", "}", -1)
	return result
}

// escapeUnescapedQuotes attempts to escape unescaped quotes
func (p *Parser) escapeUnescapedQuotes(s string) string {
	// This is complex and error-prone, so we'll just return the original
	// In production, you'd want a more sophisticated approach
	return s
}