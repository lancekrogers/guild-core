// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// googleParser handles Google/Gemini tool call format
type googleParser struct{}

// NewGoogleParser creates a new Google format parser
func NewGoogleParser() ToolCallParser {
	return &googleParser{}
}

// ParseResponse parses Google's format for tool calls
func (p *googleParser) ParseResponse(response string) ([]ToolCall, error) {
	// Google uses a slightly different format
	var googleResp struct {
		FunctionCalls []struct {
			Name string          `json:"name"`
			Args json.RawMessage `json:"args"`
		} `json:"functionCalls"`
	}
	
	if err := json.Unmarshal([]byte(response), &googleResp); err == nil && len(googleResp.FunctionCalls) > 0 {
		var toolCalls []ToolCall
		for i, fc := range googleResp.FunctionCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:   generateID(i),
				Type: "function",
				Function: Function{
					Name:      fc.Name,
					Arguments: fc.Args,
				},
			})
		}
		return toolCalls, nil
	}
	
	return nil, ErrNoToolCalls
}

// HasToolCalls checks if the response contains Google-style tool calls
func (p *googleParser) HasToolCalls(response string) bool {
	return strings.Contains(response, "functionCall") || strings.Contains(response, "function_call")
}

// SupportedFormat returns the Google format
func (p *googleParser) SupportedFormat() ProviderFormat {
	return FormatGoogleAI
}

// googleFormatter formats tools for Google's API
type googleFormatter struct{}

// NewGoogleFormatter creates a new Google formatter
func NewGoogleFormatter() ToolFormatter {
	return &googleFormatter{}
}

// FormatToolDefinitions converts to Google's format
func (f *googleFormatter) FormatToolDefinitions(tools []ToolDefinition) interface{} {
	// Google uses a slightly different structure
	var googleTools []map[string]interface{}
	
	for _, tool := range tools {
		googleTool := map[string]interface{}{
			"functionDeclarations": []map[string]interface{}{
				{
					"name":        tool.Function.Name,
					"description": tool.Function.Description,
					"parameters":  json.RawMessage(tool.Function.Parameters),
				},
			},
		}
		googleTools = append(googleTools, googleTool)
	}
	
	return googleTools
}

// FormatToolResult formats a result for Google
func (f *googleFormatter) FormatToolResult(result *ToolResult) string {
	if !result.Success {
		return "Error: " + result.Error
	}
	
	switch v := result.Output.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

// SupportedFormat returns the Google format
func (f *googleFormatter) SupportedFormat() ProviderFormat {
	return FormatGoogleAI
}

// generateID creates a unique ID for a tool call
func generateID(index int) string {
	return fmt.Sprintf("call_%d_%d", time.Now().Unix(), index)
}