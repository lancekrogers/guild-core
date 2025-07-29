// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package xml

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/tools/parser/types"
)

// Parser parses Anthropic-style XML tool calls
type Parser struct {
	// Options
	strictMode bool
	lenient    bool
}

// NewParser creates a new XML format parser
func NewParser() types.FormatParser {
	return &Parser{
		strictMode: false,
		lenient:    true, // Be forgiving with XML parsing
	}
}

// Format returns the format this parser handles
func (p *Parser) Format() types.ProviderFormat {
	return types.ProviderFormatAnthropic
}

// FunctionCalls represents the root XML structure
type FunctionCalls struct {
	XMLName xml.Name `xml:"function_calls"`
	Invokes []Invoke `xml:"invoke"`
	Tools   []Tool   `xml:"tool"`
}

// Invoke represents a function invocation
type Invoke struct {
	Name       string      `xml:"name,attr"`
	Parameters []Parameter `xml:"parameter"`
}

// Tool represents an alternative function invocation format
type Tool struct {
	Name   string  `xml:"name,attr"`
	Params []Param `xml:"param"`
}

// Parameter represents a function parameter
type Parameter struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// Param represents an alternative parameter format
type Param struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

// Parse extracts tool calls from XML input
func (p *Parser) Parse(ctx context.Context, input []byte) ([]types.ToolCall, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("parser").
		WithOperation("XMLParser.Parse")

	// Try structured parsing first
	calls, err := p.parseStructuredXML(input)
	if err == nil {
		return calls, nil
	}

	logger.Debug("Structured parse failed, trying streaming parser",
		"error", err,
	)

	// Try streaming parser for incomplete/malformed XML
	calls, streamErr := p.parseStreamingXML(input)
	if streamErr == nil && len(calls) > 0 {
		return calls, nil
	}

	logger.Debug("Streaming parse failed, trying extraction",
		"error", streamErr,
	)

	// Try to extract XML from mixed content
	detector := NewDetector()
	xmlData, location := detector.extractXML(input)
	if xmlData == nil {
		// If we had any success with streaming parser, return those results
		if len(calls) > 0 {
			return calls, nil
		}
		return []types.ToolCall{}, gerror.New(gerror.ErrCodeNotFound, "no valid XML found", nil).
			WithComponent("parser").
			WithOperation("XMLParser.Parse")
	}

	logger.Debug("Extracted XML",
		"location", location,
		"size", len(xmlData),
	)

	// Parse extracted XML
	calls, err = p.parseStructuredXML(xmlData)
	if err != nil {
		// Fall back to streaming parser if structured fails
		logger.Debug("Structured parse of extracted XML failed, trying streaming",
			"error", err,
		)
		return p.parseStreamingXML(xmlData)
	}

	return calls, nil
}

// parseStructuredXML uses the standard XML unmarshaler
func (p *Parser) parseStructuredXML(input []byte) ([]types.ToolCall, error) {
	var functionCalls FunctionCalls

	decoder := xml.NewDecoder(bytes.NewReader(input))
	decoder.Strict = !p.lenient

	if err := decoder.Decode(&functionCalls); err != nil {
		return nil, err
	}

	// Convert to standard format
	calls := []types.ToolCall{}
	for i, invoke := range functionCalls.Invokes {
		// Skip invokes without names
		if invoke.Name == "" {
			continue
		}
		call := p.convertInvokeToToolCall(invoke, i)
		calls = append(calls, call)
	}
	
	// Also convert tools (alternative format)
	for i, tool := range functionCalls.Tools {
		// Skip tools without names
		if tool.Name == "" {
			continue
		}
		call := p.convertToolToToolCall(tool, len(functionCalls.Invokes)+i)
		calls = append(calls, call)
	}

	return calls, nil
}

// parseStreamingXML uses streaming parser for more lenient parsing
func (p *Parser) parseStreamingXML(input []byte) ([]types.ToolCall, error) {
	decoder := xml.NewDecoder(bytes.NewReader(input))
	decoder.Strict = false

	calls := []types.ToolCall{}
	var currentInvoke *Invoke
	var currentParam *Parameter
	inFunctionCalls := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			if len(calls) > 0 {
				// Return what we parsed so far
				return calls, nil
			}
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "XML parsing failed").
				WithComponent("parser").
				WithOperation("parseStreamingXML")
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "function_calls":
				inFunctionCalls = true

			case "invoke", "tool": // Support both invoke and tool tags
				if inFunctionCalls {
					// Start new invoke
					currentInvoke = &Invoke{
						Parameters: []Parameter{},
					}

					// Get name attribute
					for _, attr := range t.Attr {
						if attr.Name.Local == "name" {
							currentInvoke.Name = attr.Value
							break
						}
					}
				}

			case "parameter", "param": // Support variations
				if currentInvoke != nil {
					currentParam = &Parameter{}

					// Get name attribute
					for _, attr := range t.Attr {
						if attr.Name.Local == "name" {
							currentParam.Name = attr.Value
							break
						}
					}
				}
			}

		case xml.CharData:
			// Capture parameter value
			if currentParam != nil {
				// Append to existing value (XML might send text in chunks)
				currentParam.Value += string(t)
			}

		case xml.EndElement:
			switch t.Name.Local {
			case "function_calls":
				inFunctionCalls = false

			case "invoke", "tool":
				if currentInvoke != nil && currentInvoke.Name != "" {
					call := p.convertInvokeToToolCall(*currentInvoke, len(calls))
					calls = append(calls, call)
					currentInvoke = nil
				}

			case "parameter", "param":
				if currentParam != nil && currentInvoke != nil {
					if currentParam.Name != "" { // Only add named parameters
						// Trim the parameter value before adding
						currentParam.Value = strings.TrimSpace(currentParam.Value)
						currentInvoke.Parameters = append(currentInvoke.Parameters, *currentParam)
					}
					currentParam = nil
				}
			}
		}
	}

	if len(calls) == 0 {
		return []types.ToolCall{}, gerror.New(gerror.ErrCodeNotFound, "no tool calls found in XML", nil).
			WithComponent("parser").
			WithOperation("parseStreamingXML")
	}

	return calls, nil
}

// convertInvokeToToolCall converts XML invoke to standard tool call
func (p *Parser) convertInvokeToToolCall(invoke Invoke, index int) types.ToolCall {
	call := types.ToolCall{
		ID:       fmt.Sprintf("call_%d_%d", time.Now().Unix(), index),
		Type:     "function",
		Metadata: make(map[string]interface{}),
	}

	call.Function.Name = invoke.Name

	// Convert parameters to JSON
	args := make(map[string]interface{})
	for _, param := range invoke.Parameters {
		// Trim the parameter value
		trimmedValue := strings.TrimSpace(param.Value)
		
		// Try to parse parameter value as JSON first
		var value interface{}
		if err := json.Unmarshal([]byte(trimmedValue), &value); err == nil {
			args[param.Name] = value
		} else {
			// Use as string if not valid JSON
			args[param.Name] = trimmedValue
		}
	}

	// Marshal to JSON
	if data, err := json.Marshal(args); err == nil {
		call.Function.Arguments = data
	} else {
		call.Function.Arguments = json.RawMessage("{}")
	}

	// Store original XML structure in metadata
	call.Metadata["xml_parameters"] = invoke.Parameters

	return call
}

// convertToolToToolCall converts XML tool to standard tool call
func (p *Parser) convertToolToToolCall(tool Tool, index int) types.ToolCall {
	call := types.ToolCall{
		ID:       fmt.Sprintf("call_%d_%d", time.Now().Unix(), index),
		Type:     "function",
		Metadata: make(map[string]interface{}),
	}

	call.Function.Name = tool.Name

	// Convert params to JSON
	args := make(map[string]interface{})
	for _, param := range tool.Params {
		// Trim the parameter value
		trimmedValue := strings.TrimSpace(param.Value)
		
		// Try to parse parameter value as JSON first
		var value interface{}
		if err := json.Unmarshal([]byte(trimmedValue), &value); err == nil {
			args[param.Name] = value
		} else {
			// Use as string if not valid JSON
			args[param.Name] = trimmedValue
		}
	}

	// Marshal to JSON
	if data, err := json.Marshal(args); err == nil {
		call.Function.Arguments = data
	} else {
		call.Function.Arguments = json.RawMessage("{}")
	}

	// Store original XML structure in metadata
	call.Metadata["xml_params"] = tool.Params

	return call
}

// Validate checks if input conforms to expected XML schema
func (p *Parser) Validate(input []byte) types.ValidationResult {
	result := types.ValidationResult{
		Valid:      true,
		Errors:     []types.ValidationError{},
		Warnings:   []string{},
		SchemaUsed: "anthropic_xml_v1",
		Metadata:   make(map[string]interface{}),
	}

	// Try to parse first
	decoder := xml.NewDecoder(bytes.NewReader(input))
	decoder.Strict = !p.lenient

	var functionCalls FunctionCalls
	if err := decoder.Decode(&functionCalls); err != nil {
		result.Valid = false
		
		// Provide more specific error messages based on the content
		s := string(input)
		if !strings.Contains(s, "<function_calls>") {
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    "$",
				Message: "Missing <function_calls> root element",
				Code:    "missing_root",
			})
		} else if !strings.Contains(s, "</function_calls>") {
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    "$",
				Message: "Missing </function_calls> closing tag",
				Code:    "missing_closing_tag",
			})
		} else {
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    "$",
				Message: "XML parsing failed: " + err.Error(),
				Code:    "parse_error",
			})
		}
		return result
	}

	// Validate structure
	if len(functionCalls.Invokes) == 0 && len(functionCalls.Tools) == 0 {
		result.Warnings = append(result.Warnings, "No invoke or tool elements found")
	}

	for i, invoke := range functionCalls.Invokes {
		invokePath := fmt.Sprintf("$.invoke[%d]", i)

		// Check name
		if invoke.Name == "" {
			result.Valid = false
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    invokePath + ".name",
				Message: "Invoke missing name attribute",
				Code:    "missing_attribute",
			})
		}

		// Check parameters
		for j, param := range invoke.Parameters {
			paramPath := fmt.Sprintf("%s.parameter[%d]", invokePath, j)

			if param.Name == "" {
				result.Valid = false
				result.Errors = append(result.Errors, types.ValidationError{
					Path:    paramPath + ".name",
					Message: "Parameter missing name attribute",
					Code:    "missing_attribute",
				})
			}

			if param.Value == "" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Empty parameter value at %s", paramPath))
			}
		}
	}

	// Also validate tools
	for i, tool := range functionCalls.Tools {
		toolPath := fmt.Sprintf("$.tool[%d]", i)

		// Check name
		if tool.Name == "" {
			result.Valid = false
			result.Errors = append(result.Errors, types.ValidationError{
				Path:    toolPath + ".name",
				Message: "Tool missing name attribute",
				Code:    "missing_attribute",
			})
		}

		// Check params
		for j, param := range tool.Params {
			paramPath := fmt.Sprintf("%s.param[%d]", toolPath, j)

			if param.Name == "" {
				result.Valid = false
				result.Errors = append(result.Errors, types.ValidationError{
					Path:    paramPath + ".name",
					Message: "Param missing name attribute",
					Code:    "missing_attribute",
				})
			}

			if param.Value == "" {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Empty param value at %s", paramPath))
			}
		}
	}

	result.Metadata["invoke_count"] = len(functionCalls.Invokes)
	result.Metadata["tool_count"] = len(functionCalls.Tools)

	return result
}
