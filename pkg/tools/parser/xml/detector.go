// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package xml

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"strings"

	"github.com/lancekrogers/guild/pkg/tools/parser/types"
)

// Detector detects Anthropic-style XML tool call format
type Detector struct {
	// XML patterns to look for
	xmlPatterns []string
}

// NewDetector creates a new XML format detector
func NewDetector() *Detector {
	return &Detector{
		xmlPatterns: []string{
			"<function_calls>",
			"</function_calls>",
			"<invoke",
			"</invoke>",
			"<parameter",
		},
	}
}

// Format returns the format this detector handles
func (d *Detector) Format() types.ProviderFormat {
	return types.ProviderFormatAnthropic
}

// CanParse quickly checks if input might contain XML tool calls
func (d *Detector) CanParse(input []byte) bool {
	if len(input) == 0 {
		return false
	}

	// Quick string check for XML patterns
	s := string(input)
	for _, pattern := range d.xmlPatterns {
		if strings.Contains(s, pattern) {
			return true
		}
	}

	return false
}

// Detect analyzes input and returns detection result with confidence
func (d *Detector) Detect(ctx context.Context, input []byte) (types.DetectionResult, error) {
	result := types.DetectionResult{
		Format:     types.ProviderFormatAnthropic,
		Confidence: 0,
		Metadata:   make(map[string]interface{}),
	}

	// Extract XML section
	xmlData, location := d.extractXML(input)
	if xmlData == nil {
		return result, nil
	}

	result.Metadata["extraction_location"] = location
	result.Metadata["original_size"] = len(input)
	result.Metadata["xml_size"] = len(xmlData)

	// Try to parse with XML decoder
	confidence := d.analyzeXMLStructure(xmlData)
	result.Confidence = confidence

	// Add parsing metadata
	if confidence > 0.5 {
		invokes, params := d.countXMLElements(xmlData)
		result.Metadata["invoke_count"] = invokes
		result.Metadata["parameter_count"] = params
	}

	return result, nil
}

// extractXML finds and extracts XML function calls from mixed content
func (d *Detector) extractXML(input []byte) ([]byte, string) {
	s := string(input)

	// Strategy 1: Look for complete function_calls block
	start := strings.Index(s, "<function_calls>")
	if start != -1 {
		end := strings.Index(s[start:], "</function_calls>")
		if end != -1 {
			xmlData := []byte(s[start : start+end+len("</function_calls>")])
			if d.isWellFormedXML(xmlData) {
				return xmlData, "complete_block"
			}
		}
	}

	// Strategy 2: Try to extract with more lenient boundaries
	if xmlData := d.extractWithBoundarySearch(input); xmlData != nil {
		return xmlData, "boundary_search"
	}

	// Strategy 3: Extract from code blocks
	if xmlData := d.extractFromCodeBlock(input); xmlData != nil {
		return xmlData, "code_block"
	}

	return nil, ""
}

// extractWithBoundarySearch finds XML with fuzzy boundary matching
func (d *Detector) extractWithBoundarySearch(input []byte) []byte {
	s := string(input)

	// Find all potential start positions
	starts := []int{}
	idx := 0
	for {
		pos := strings.Index(s[idx:], "<function_calls")
		if pos == -1 {
			break
		}
		starts = append(starts, idx+pos)
		idx = idx + pos + 1
	}

	// Try each start position
	for _, start := range starts {
		// Find the closing tag
		searchFrom := start + 16 // len("<function_calls>")
		end := strings.Index(s[searchFrom:], "</function_calls>")
		if end != -1 {
			candidate := []byte(s[start : searchFrom+end+len("</function_calls>")])
			if d.isWellFormedXML(candidate) {
				return candidate
			}
		}
	}

	return nil
}

// extractFromCodeBlock extracts XML from markdown code blocks
func (d *Detector) extractFromCodeBlock(input []byte) []byte {
	s := string(input)

	// Look for ```xml blocks
	if start := strings.Index(s, "```xml"); start != -1 {
		start += 6 // len("```xml")
		if end := strings.Index(s[start:], "```"); end != -1 {
			candidate := []byte(strings.TrimSpace(s[start : start+end]))
			if d.isWellFormedXML(candidate) && d.containsFunctionCalls(candidate) {
				return candidate
			}
		}
	}

	// Look for generic ``` blocks with XML
	codeStart := 0
	for {
		idx := strings.Index(s[codeStart:], "```")
		if idx == -1 {
			break
		}

		blockStart := codeStart + idx + 3
		// Skip language identifier
		if newline := strings.IndexByte(s[blockStart:], '\n'); newline != -1 {
			blockStart += newline + 1
		}

		if blockEnd := strings.Index(s[blockStart:], "```"); blockEnd != -1 {
			candidate := []byte(strings.TrimSpace(s[blockStart : blockStart+blockEnd]))
			if d.isWellFormedXML(candidate) && d.containsFunctionCalls(candidate) {
				return candidate
			}
		}

		codeStart = blockStart
	}

	return nil
}

// isWellFormedXML checks if data is well-formed XML
func (d *Detector) isWellFormedXML(data []byte) bool {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.Strict = false // Be lenient

	// Try to parse all tokens
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			return true
		}
		if err != nil {
			return false
		}
	}
}

// containsFunctionCalls checks if XML contains expected structure
func (d *Detector) containsFunctionCalls(data []byte) bool {
	s := string(data)
	return strings.Contains(s, "<function_calls") &&
		strings.Contains(s, "</function_calls>") &&
		(strings.Contains(s, "<invoke") || strings.Contains(s, "<tool"))
}

// analyzeXMLStructure analyzes XML and returns confidence score
func (d *Detector) analyzeXMLStructure(xmlData []byte) float64 {
	// Try to parse with streaming decoder for better error handling
	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	decoder.Strict = false

	inFunctionCalls := false
	invokeCount := 0
	paramCount := 0
	depth := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Partial parse still gives some confidence
			if invokeCount > 0 {
				return 0.5 + (0.1 * float64(min(invokeCount, 3)))
			}
			return 0.2
		}

		switch t := token.(type) {
		case xml.StartElement:
			depth++
			switch t.Name.Local {
			case "function_calls":
				inFunctionCalls = true
			case "invoke", "tool":
				if inFunctionCalls {
					invokeCount++
					// Check for name attribute
					for _, attr := range t.Attr {
						if attr.Name.Local == "name" && attr.Value != "" {
							// Bonus confidence for proper attributes
							invokeCount++
						}
					}
				}
			case "parameter":
				if inFunctionCalls {
					paramCount++
				}
			}
		case xml.EndElement:
			depth--
			if t.Name.Local == "function_calls" {
				inFunctionCalls = false
			}
		}

		// Sanity check on depth
		if depth > 10 {
			return 0.3 // Too deeply nested, suspicious
		}
	}

	// Calculate confidence based on structure
	if invokeCount == 0 {
		return 0.3 // Has structure but no invokes
	}

	// Base confidence
	confidence := 0.7

	// Bonus for multiple invokes (up to 3)
	confidence += 0.05 * float64(min(invokeCount, 3))

	// Bonus for parameters
	if paramCount > 0 {
		confidence += 0.1
	}

	// Bonus for proper ratio of params to invokes
	if invokeCount > 0 && paramCount/invokeCount >= 1 {
		confidence += 0.05
	}

	return minFloat(confidence, 0.95)
}

// countXMLElements counts invoke and parameter elements
func (d *Detector) countXMLElements(xmlData []byte) (invokes, params int) {
	s := string(xmlData)

	// Count invokes
	invokes = strings.Count(s, "<invoke")

	// Count parameters
	params = strings.Count(s, "<parameter")

	return
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
