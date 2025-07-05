// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// anthropicParser handles Anthropic's XML-based tool call format
type anthropicParser struct{}

// NewAnthropicParser creates a new Anthropic format parser
func NewAnthropicParser() ToolCallParser {
	return &anthropicParser{}
}

// Anthropic XML structures
type anthropicFunctionCalls struct {
	XMLName xml.Name              `xml:"function_calls"`
	Invokes []anthropicInvocation `xml:"invoke"`
}

type anthropicInvocation struct {
	Name       string               `xml:"name,attr"`
	Parameters []anthropicParameter `xml:"parameter"`
}

type anthropicParameter struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// ParseResponse parses Anthropic's XML format for tool calls
func (p *anthropicParser) ParseResponse(response string) ([]ToolCall, error) {
	if !p.HasToolCalls(response) {
		return nil, gerror.Wrap(ErrNoToolCalls, gerror.ErrCodeNotFound, "no tool calls in response").
			WithComponent("AnthropicParser").
			WithOperation("ParseResponse")
	}

	var allCalls []ToolCall
	
	// Find all function_calls blocks
	parts := strings.Split(response, "<function_calls>")
	
	for i := 1; i < len(parts); i++ {
		endIdx := strings.Index(parts[i], "</function_calls>")
		if endIdx == -1 {
			continue
		}
		
		xmlContent := "<function_calls>" + parts[i][:endIdx+len("</function_calls>")]
		
		var fc anthropicFunctionCalls
		if err := xml.Unmarshal([]byte(xmlContent), &fc); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to parse XML").
				WithComponent("AnthropicParser").
				WithOperation("ParseResponse").
				WithDetails("xmlContent", xmlContent)
		}
		
		// Convert to standard format
		for idx, invoke := range fc.Invokes {
			params := make(map[string]interface{})
			for _, p := range invoke.Parameters {
				// Keep parameters as strings - let the tool decide how to parse them
				params[p.Name] = p.Value
			}
			
			paramsJSON, err := json.Marshal(params)
			if err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal parameters").
					WithComponent("AnthropicParser").
					WithOperation("ParseResponse").
					WithDetails("function", invoke.Name)
			}
			
			allCalls = append(allCalls, ToolCall{
				ID:   fmt.Sprintf("call_%d_%d", time.Now().Unix(), i*100+idx),
				Type: "function",
				Function: Function{
					Name:      invoke.Name,
					Arguments: paramsJSON,
				},
			})
		}
	}
	
	if len(allCalls) == 0 {
		return nil, gerror.Wrap(ErrNoToolCalls, gerror.ErrCodeNotFound, "no valid tool calls found").
			WithComponent("AnthropicParser").
			WithOperation("ParseResponse")
	}
	
	return allCalls, nil
}

// HasToolCalls checks if the response contains Anthropic-style tool calls
func (p *anthropicParser) HasToolCalls(response string) bool {
	return strings.Contains(response, "<function_calls>") || 
		strings.Contains(response, "<invoke name=")
}

// SupportedFormat returns the Anthropic format
func (p *anthropicParser) SupportedFormat() ProviderFormat {
	return FormatAnthropic
}

// anthropicFormatter formats tools for Anthropic's API
type anthropicFormatter struct{}

// NewAnthropicFormatter creates a new Anthropic formatter
func NewAnthropicFormatter() ToolFormatter {
	return &anthropicFormatter{}
}

// FormatToolDefinitions converts to Anthropic's format
func (f *anthropicFormatter) FormatToolDefinitions(tools []ToolDefinition) interface{} {
	// Anthropic uses a specific structure for tools
	anthropicTools := make([]map[string]interface{}, 0, len(tools))
	
	for _, tool := range tools {
		// Parse the parameters JSON Schema
		var params interface{}
		if err := json.Unmarshal(tool.Function.Parameters, &params); err != nil {
			params = map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			}
		}
		
		anthropicTool := map[string]interface{}{
			"name":        tool.Function.Name,
			"description": tool.Function.Description,
			"input_schema": params,
		}
		
		anthropicTools = append(anthropicTools, anthropicTool)
	}
	
	return anthropicTools
}

// FormatToolResult formats a result for Anthropic
func (f *anthropicFormatter) FormatToolResult(result *ToolResult) string {
	if !result.Success {
		return fmt.Sprintf("Error executing tool: %s", result.Error)
	}
	
	// Anthropic expects text responses
	switch v := result.Output.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		// Convert to JSON for complex types
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("Error formatting result: %v", err)
		}
		return string(data)
	}
}

// SupportedFormat returns the Anthropic format
func (f *anthropicFormatter) SupportedFormat() ProviderFormat {
	return FormatAnthropic
}