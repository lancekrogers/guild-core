// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package json

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/guild-framework/guild-core/pkg/tools/parser/types"
)

// Detector detects OpenAI-style JSON tool call format with confidence scoring
type Detector struct {
	// Magic bytes for JSON detection
	jsonStartBytes [][]byte
}

// NewDetector creates a new JSON format detector
func NewDetector() *Detector {
	return &Detector{
		// Common JSON start patterns
		jsonStartBytes: [][]byte{
			[]byte("{"),
			[]byte("["),
			[]byte(`{"tool_calls"`),
			[]byte(`{ "tool_calls"`),
			[]byte(`{"function_call"`),
			[]byte(`{ "function_call"`),
		},
	}
}

// Format returns the format this detector handles
func (d *Detector) Format() types.ProviderFormat {
	return types.ProviderFormatOpenAI
}

// CanParse performs quick pre-check without full parsing
func (d *Detector) CanParse(input []byte) bool {
	if len(input) == 0 {
		return false
	}

	// Quick magic byte check
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 {
		return false
	}

	// Check JSON start characters
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return true
	}

	// Check for tool-related keywords
	inputStr := string(input)
	return strings.Contains(inputStr, `"tool_calls"`) ||
		strings.Contains(inputStr, `"function_call"`) ||
		strings.Contains(inputStr, `"tools"`) ||
		(strings.Contains(inputStr, `"type"`) && strings.Contains(inputStr, `"function"`))
}

// Detect analyzes input and returns detection result with confidence score
func (d *Detector) Detect(ctx context.Context, input []byte) (types.DetectionResult, error) {
	result := types.DetectionResult{
		Format:     types.ProviderFormatOpenAI,
		Confidence: 0,
		Metadata:   make(map[string]interface{}),
	}

	// Validate UTF-8
	if !utf8.Valid(input) {
		result.Metadata["error"] = "invalid UTF-8"
		return result, nil
	}

	// Extract JSON with location tracking
	jsonData, location := d.extractJSON(input)
	if jsonData == nil {
		return result, nil
	}

	result.Metadata["extraction_location"] = location
	result.Metadata["original_size"] = len(input)
	result.Metadata["json_size"] = len(jsonData)

	// Validate JSON structure
	if !json.Valid(jsonData) {
		result.Confidence = 0.1
		result.Metadata["json_valid"] = false
		return result, nil
	}

	// Parse and analyze structure
	var doc interface{}
	decoder := json.NewDecoder(bytes.NewReader(jsonData))
	decoder.UseNumber() // Preserve number precision

	if err := decoder.Decode(&doc); err != nil {
		result.Confidence = 0.2
		result.Metadata["parse_error"] = err.Error()
		return result, nil
	}

	// Check for extra data after JSON
	if decoder.More() {
		result.Metadata["has_trailing_data"] = true
		result.Confidence = max(0, result.Confidence-0.1)
	}

	// Analyze structure for OpenAI patterns
	confidence := d.analyzeOpenAIStructure(doc)
	result.Confidence = confidence

	// Add structure metadata
	switch v := doc.(type) {
	case map[string]interface{}:
		result.Metadata["root_type"] = "object"
		result.Metadata["has_tool_calls"] = v["tool_calls"] != nil
		result.Metadata["has_function_call"] = v["function_call"] != nil
	case []interface{}:
		result.Metadata["root_type"] = "array"
		result.Metadata["array_length"] = len(v)
	}

	return result, nil
}

// extractJSON finds and extracts JSON from mixed content
func (d *Detector) extractJSON(input []byte) ([]byte, string) {
	// Strategy 1: Check if entire input is valid JSON
	trimmed := bytes.TrimSpace(input)
	if json.Valid(trimmed) {
		return trimmed, "entire_input"
	}

	// Strategy 2: Find JSON boundaries in mixed content
	if jsonData, ok := d.extractJSONFromMixed(input); ok {
		return jsonData, "mixed_content"
	}

	// Strategy 3: Extract from code blocks
	if jsonData, ok := d.extractFromCodeBlock(input); ok {
		return jsonData, "code_block"
	}

	// Strategy 4: Try to extract JSON object/array with streaming parser
	if jsonData, ok := d.extractWithStreamingParser(input); ok {
		return jsonData, "streaming_extraction"
	}

	return nil, ""
}

// extractJSONFromMixed extracts JSON from text with conversational content
func (d *Detector) extractJSONFromMixed(input []byte) ([]byte, bool) {
	// Find potential JSON start positions
	starts := d.findJSONStarts(input)
	if len(starts) == 0 {
		return nil, false
	}

	// Try each potential start position
	for _, start := range starts {
		if end := d.findJSONEnd(input[start:]); end > 0 {
			candidate := input[start : start+end]
			if json.Valid(candidate) {
				return candidate, true
			}
		}
	}

	return nil, false
}

// findJSONStarts finds potential JSON start positions
func (d *Detector) findJSONStarts(input []byte) []int {
	var positions []int

	// Look for object starts
	for i := 0; i < len(input); i++ {
		if input[i] == '{' {
			// Check if it might be a tool call object
			if i+20 < len(input) {
				ahead := string(input[i:min(i+50, len(input))])
				if strings.Contains(ahead, `"tool_calls"`) ||
					strings.Contains(ahead, `"function_call"`) ||
					strings.Contains(ahead, `"function"`) ||
					strings.Contains(ahead, `"choices"`) ||
					strings.Contains(ahead, `"type"`) {
					positions = append(positions, i)
				}
			} else {
				positions = append(positions, i)
			}
		} else if input[i] == '[' {
			positions = append(positions, i)
		}
	}

	return positions
}

// findJSONEnd finds the end of a JSON structure using a state machine
func (d *Detector) findJSONEnd(input []byte) int {
	if len(input) == 0 {
		return -1
	}

	// Simple state machine for JSON parsing
	depth := 0
	inString := false
	escaped := false

	startChar := input[0]
	if startChar != '{' && startChar != '[' {
		return -1
	}

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' && !escaped {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return -1
}

// extractFromCodeBlock extracts JSON from markdown code blocks
func (d *Detector) extractFromCodeBlock(input []byte) ([]byte, bool) {
	s := string(input)

	// Pattern 1: ```json blocks
	if start := strings.Index(s, "```json"); start != -1 {
		start += 7 // len("```json")
		if end := strings.Index(s[start:], "```"); end != -1 {
			candidate := []byte(strings.TrimSpace(s[start : start+end]))
			if json.Valid(candidate) {
				return candidate, true
			}
		}
	}

	// Pattern 2: Generic ``` blocks with JSON
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
			if json.Valid(candidate) && d.looksLikeToolCalls(candidate) {
				return candidate, true
			}
		}

		codeStart = blockStart
	}

	return nil, false
}

// extractWithStreamingParser uses a streaming JSON decoder for extraction
func (d *Detector) extractWithStreamingParser(input []byte) ([]byte, bool) {
	// Try to find JSON using a streaming approach
	maxOffset := int64(len(input))

	for offset := int64(0); offset < maxOffset; offset++ {
		reader := bytes.NewReader(input[offset:])
		decoder := json.NewDecoder(reader)

		var value interface{}
		err := decoder.Decode(&value)

		if err == nil {
			// Successfully decoded JSON
			endPos := offset + decoder.InputOffset()
			return input[offset:endPos], true
		}

		// If we can't decode from this position, try the next
		if err == io.EOF || offset+1 >= maxOffset {
			break
		}
	}

	return nil, false
}

// analyzeOpenAIStructure examines JSON structure for OpenAI tool call patterns
func (d *Detector) analyzeOpenAIStructure(doc interface{}) float64 {
	switch v := doc.(type) {
	case map[string]interface{}:
		return d.analyzeOpenAIObject(v)
	case []interface{}:
		return d.analyzeToolCallArray(v)
	default:
		return 0.1
	}
}

// analyzeOpenAIObject analyzes an object for OpenAI patterns
func (d *Detector) analyzeOpenAIObject(obj map[string]interface{}) float64 {
	confidence := 0.0

	// Check for single tool call format (e.g., {"id": "...", "type": "function", "function": {...}})
	if _, hasID := obj["id"]; hasID {
		if objType, hasType := obj["type"].(string); hasType && objType == "function" {
			if funcObj, hasFunc := obj["function"].(map[string]interface{}); hasFunc {
				if funcObj["name"] != nil && funcObj["arguments"] != nil {
					confidence = max(confidence, 0.9) // High confidence for single tool call
				}
			}
		}
	}

	// Check for tool_calls field (strongest indicator)
	if toolCalls, exists := obj["tool_calls"]; exists {
		if arr, ok := toolCalls.([]interface{}); ok {
			toolCallConf := d.analyzeToolCallArray(arr)
			confidence = max(confidence, toolCallConf)
		} else {
			confidence = max(confidence, 0.3) // Has field but wrong type
		}
	}

	// Check for function_call field (older format)
	if funcCall, exists := obj["function_call"]; exists {
		if fc, ok := funcCall.(map[string]interface{}); ok {
			if fc["name"] != nil && fc["arguments"] != nil {
				confidence = max(confidence, 0.8)
			}
		}
	}

	// Check for function field (even older format)
	if funcObj, exists := obj["function"]; exists {
		if fc, ok := funcObj.(map[string]interface{}); ok {
			if fc["name"] != nil && fc["arguments"] != nil {
				confidence = max(confidence, 0.8)
			}
		}
	}

	// Check for OpenAI chat completion format with choices
	if choices, exists := obj["choices"]; exists {
		if choicesArr, ok := choices.([]interface{}); ok && len(choicesArr) > 0 {
			// Check first choice for message with tool_calls
			if choice, ok := choicesArr[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					// Recursively analyze the message
					msgConf := d.analyzeOpenAIObject(msg)
					confidence = max(confidence, msgConf*0.9) // Slightly lower confidence for nested
				}
			}
		}
	}

	// Check for assistant message with tool calls
	if role, _ := obj["role"].(string); role == "assistant" {
		if obj["tool_calls"] != nil || obj["function_call"] != nil {
			confidence = max(confidence, 0.9)
		}
	}

	// Check nested structures
	for _, value := range obj {
		if nested := d.analyzeOpenAIStructure(value); nested > 0.7 {
			confidence = max(confidence, nested*0.9)
		}
	}

	return confidence
}

// analyzeToolCallArray analyzes an array of tool calls
func (d *Detector) analyzeToolCallArray(arr []interface{}) float64 {
	if len(arr) == 0 {
		return 0.7 // Empty array is valid
	}

	validCalls := 0
	totalScore := 0.0

	for _, item := range arr {
		call, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		score := 0.0

		// Required fields check
		if call["id"] != nil {
			score += 0.25
		}
		if callType, _ := call["type"].(string); callType == "function" {
			score += 0.25
		}
		if function, ok := call["function"].(map[string]interface{}); ok {
			if function["name"] != nil {
				score += 0.25
			}
			if function["arguments"] != nil {
				score += 0.25
			}
		}

		if score >= 0.75 {
			validCalls++
		}
		totalScore += score
	}

	if len(arr) > 0 {
		avgScore := totalScore / float64(len(arr))
		if validCalls == len(arr) {
			return minFloat(0.95, avgScore)
		}
		return minFloat(0.85, avgScore)
	}

	return 0.5
}

// looksLikeToolCalls checks if JSON contains tool call patterns
func (d *Detector) looksLikeToolCalls(data []byte) bool {
	s := string(data)
	return strings.Contains(s, `"tool_calls"`) ||
		strings.Contains(s, `"function_call"`) ||
		strings.Contains(s, `"function"`) ||
		(strings.Contains(s, `"name"`) && strings.Contains(s, `"arguments"`))
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
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
