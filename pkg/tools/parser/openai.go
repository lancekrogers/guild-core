// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"encoding/json"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// openAIParser handles OpenAI's JSON tool call format
type openAIParser struct{}

// NewOpenAIParser creates a new OpenAI format parser
func NewOpenAIParser() ToolCallParser {
	return &openAIParser{}
}

// OpenAI response structures
type openAIResponse struct {
	Choices []struct {
		Message struct {
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"message"`
		Delta struct {
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	ToolCalls []openAIToolCall `json:"tool_calls"` // Direct format
}

// openAIToolCall represents the raw tool call from OpenAI response
type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string      `json:"name"`
		Arguments interface{} `json:"arguments"` // Can be string or object
	} `json:"function"`
}

// ParseResponse parses OpenAI's JSON format for tool calls
func (p *openAIParser) ParseResponse(response string) ([]ToolCall, error) {
	if response == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "empty response", nil).
			WithComponent("OpenAIParser").
			WithOperation("ParseResponse")
	}

	var resp openAIResponse
	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to parse JSON response").
			WithComponent("OpenAIParser").
			WithOperation("ParseResponse").
			WithDetails("responseLength", len(response))
	}

	// Check direct tool_calls field first
	if len(resp.ToolCalls) > 0 {
		return p.convertAndValidateToolCalls(resp.ToolCalls)
	}

	// Check choices format
	if len(resp.Choices) > 0 {
		// Check message format
		if len(resp.Choices[0].Message.ToolCalls) > 0 {
			return p.convertAndValidateToolCalls(resp.Choices[0].Message.ToolCalls)
		}
		// Check delta format (streaming)
		if len(resp.Choices[0].Delta.ToolCalls) > 0 {
			return p.convertAndValidateToolCalls(resp.Choices[0].Delta.ToolCalls)
		}
	}

	return nil, gerror.Wrap(ErrNoToolCalls, gerror.ErrCodeNotFound, "no tool calls found").
		WithComponent("OpenAIParser").
		WithOperation("ParseResponse")
}

// convertAndValidateToolCalls converts OpenAI format to standard format and validates
func (p *openAIParser) convertAndValidateToolCalls(calls []openAIToolCall) ([]ToolCall, error) {
	if len(calls) == 0 {
		return nil, gerror.Wrap(ErrNoToolCalls, gerror.ErrCodeValidation, "empty tool calls array").
			WithComponent("OpenAIParser").
			WithOperation("convertAndValidateToolCalls")
	}

	valid := make([]ToolCall, 0, len(calls))
	
	for i, call := range calls {
		// Validate required fields
		if call.Function.Name == "" {
			continue // Skip invalid calls
		}
		
		// Ensure ID is set
		if call.ID == "" {
			return nil, gerror.New(gerror.ErrCodeValidation, "tool call missing ID", nil).
				WithComponent("OpenAIParser").
				WithOperation("convertAndValidateToolCalls").
				WithDetails("index", i).
				WithDetails("function", call.Function.Name)
		}
		
		// Convert arguments to json.RawMessage
		var args json.RawMessage
		switch v := call.Function.Arguments.(type) {
		case string:
			// Arguments is a JSON string
			args = json.RawMessage(v)
		case []byte:
			// Arguments is bytes
			args = json.RawMessage(v)
		case nil:
			// No arguments
			args = json.RawMessage("{}")
		default:
			// Arguments is an object - marshal it
			data, err := json.Marshal(v)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "failed to marshal arguments").
					WithComponent("OpenAIParser").
					WithOperation("convertAndValidateToolCalls").
					WithDetails("function", call.Function.Name)
			}
			args = data
		}
		
		// Validate arguments is valid JSON
		if len(args) > 0 {
			var test interface{}
			if err := json.Unmarshal(args, &test); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid arguments JSON").
					WithComponent("OpenAIParser").
					WithOperation("convertAndValidateToolCalls").
					WithDetails("function", call.Function.Name).
					WithDetails("arguments", string(args))
			}
		} else {
			args = json.RawMessage("{}")
		}
		
		// Create standard tool call
		stdCall := ToolCall{
			ID:   call.ID,
			Type: call.Type,
			Function: Function{
				Name:      call.Function.Name,
				Arguments: args,
			},
		}
		
		// Ensure type is set
		if stdCall.Type == "" {
			stdCall.Type = "function"
		}
		
		valid = append(valid, stdCall)
	}
	
	if len(valid) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no valid tool calls after validation", nil).
			WithComponent("OpenAIParser").
			WithOperation("validateToolCalls").
			WithDetails("originalCount", len(calls))
	}
	
	return valid, nil
}

// HasToolCalls checks if the response contains OpenAI-style tool calls
func (p *openAIParser) HasToolCalls(response string) bool {
	if response == "" || !json.Valid([]byte(response)) {
		return false
	}
	
	// Quick check for tool call indicators
	return strings.Contains(response, `"tool_calls"`) || strings.Contains(response, `"function_call"`)
}

// SupportedFormat returns the OpenAI format
func (p *openAIParser) SupportedFormat() ProviderFormat {
	return FormatOpenAI
}

// openAIFormatter formats tools for OpenAI's API
type openAIFormatter struct{}

// NewOpenAIFormatter creates a new OpenAI formatter
func NewOpenAIFormatter() ToolFormatter {
	return &openAIFormatter{}
}

// FormatToolDefinitions returns tools in OpenAI's expected format
func (f *openAIFormatter) FormatToolDefinitions(tools []ToolDefinition) interface{} {
	// Validate tools before returning
	validTools := make([]ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		if tool.Function.Name == "" {
			continue
		}
		// Ensure parameters is valid JSON Schema
		if len(tool.Function.Parameters) == 0 {
			tool.Function.Parameters = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		validTools = append(validTools, tool)
	}
	return validTools
}

// FormatToolResult formats a result for OpenAI
func (f *openAIFormatter) FormatToolResult(result *ToolResult) string {
	// OpenAI expects the content as a string
	if !result.Success {
		return "Error: " + result.Error
	}
	
	// Use Content if available, otherwise format Output
	if result.Content != "" {
		return result.Content
	}
	
	switch v := result.Output.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		// Convert to JSON for complex types
		data, err := json.Marshal(v)
		if err != nil {
			return "Error formatting result: " + err.Error()
		}
		return string(data)
	}
}

// SupportedFormat returns the OpenAI format
func (f *openAIFormatter) SupportedFormat() ProviderFormat {
	return FormatOpenAI
}