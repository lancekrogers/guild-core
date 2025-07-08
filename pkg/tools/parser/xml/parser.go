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
}

// Invoke represents a function invocation
type Invoke struct {
	Name       string      `xml:"name,attr"`
	Parameters []Parameter `xml:"parameter"`
}

// Parameter represents a function parameter
type Parameter struct {
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

	logger.Debug("Structured parse failed, trying extraction",
		"error", err,
	)

	// Try to extract XML from mixed content
	detector := NewDetector()
	xmlData, location := detector.extractXML(input)
	if xmlData == nil {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no valid XML found", nil).
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
	var calls []types.ToolCall
	for i, invoke := range functionCalls.Invokes {
		call := p.convertInvokeToToolCall(invoke, i)
		calls = append(calls, call)
	}

	return calls, nil
}

// parseStreamingXML uses streaming parser for more lenient parsing
func (p *Parser) parseStreamingXML(input []byte) ([]types.ToolCall, error) {
	decoder := xml.NewDecoder(bytes.NewReader(input))
	decoder.Strict = false

	var calls []types.ToolCall
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
				currentParam.Value = strings.TrimSpace(string(t))
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
						currentInvoke.Parameters = append(currentInvoke.Parameters, *currentParam)
					}
					currentParam = nil
				}
			}
		}
	}

	if len(calls) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no tool calls found in XML", nil).
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
		// Try to parse parameter value as JSON first
		var value interface{}
		if err := json.Unmarshal([]byte(param.Value), &value); err == nil {
			args[param.Name] = value
		} else {
			// Use as string if not valid JSON
			args[param.Name] = param.Value
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

// Validate checks if input conforms to expected XML schema
func (p *Parser) Validate(input []byte) types.ValidationResult {
	result := types.ValidationResult{
		Valid:      true,
		Errors:     []types.ValidationError{},
		Warnings:   []string{},
		SchemaUsed: "anthropic_xml_v1",
		Metadata:   make(map[string]interface{}),
	}

	// Check for function_calls wrapper
	s := string(input)
	if !strings.Contains(s, "<function_calls>") {
		result.Valid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    "$",
			Message: "Missing <function_calls> root element",
			Code:    "missing_root",
		})
	}

	if !strings.Contains(s, "</function_calls>") {
		result.Valid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    "$",
			Message: "Missing </function_calls> closing tag",
			Code:    "missing_closing_tag",
		})
	}

	// Try to parse
	decoder := xml.NewDecoder(bytes.NewReader(input))
	decoder.Strict = !p.lenient

	var functionCalls FunctionCalls
	if err := decoder.Decode(&functionCalls); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, types.ValidationError{
			Path:    "$",
			Message: "XML parsing failed: " + err.Error(),
			Code:    "parse_error",
		})
		return result
	}

	// Validate structure
	if len(functionCalls.Invokes) == 0 {
		result.Warnings = append(result.Warnings, "No invoke elements found")
	}

	for i, invoke := range functionCalls.Invokes {
		invokePath := fmt.Sprintf("$.invoke[%d]", i)

		// Check name
		if invoke.Name == "" {
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

	result.Metadata["invoke_count"] = len(functionCalls.Invokes)

	return result
}
