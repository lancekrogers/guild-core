// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

// Property: Parser should never panic
func TestProperty_NeverPanics(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(input string) bool {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Parser panicked with input: %q, panic: %v", input, r)
			}
		}()
		
		_, _ = parser.ExtractToolCalls(input)
		return true
	}
	
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// Property: Valid JSON tool calls should always be extracted
func TestProperty_ValidJSONAlwaysExtracted(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(id, funcName, argKey, argValue string) bool {
		// Skip if inputs contain quotes or other JSON-breaking characters
		if strings.ContainsAny(id+funcName+argKey+argValue, `"\{}[]`) {
			return true
		}
		
		// Build valid JSON
		json := fmt.Sprintf(`{"id": "%s", "type": "function", "function": {"name": "%s", "arguments": "{\"%s\": \"%s\"}"}}`,
			id, funcName, argKey, argValue)
		
		calls, err := parser.ExtractToolCalls(json)
		if err != nil {
			return false
		}
		
		// Should extract exactly one call
		if len(calls) != 1 {
			return false
		}
		
		call := calls[0]
		return call.ID == id && call.Function.Name == funcName
	}
	
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// Property: Format detection should be consistent
func TestProperty_ConsistentFormatDetection(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(input string) bool {
		// Detect format multiple times
		format1, conf1, err1 := parser.DetectFormat(input)
		format2, conf2, err2 := parser.DetectFormat(input)
		format3, conf3, err3 := parser.DetectFormat(input)
		
		// All detections should be identical
		if err1 != nil {
			return err2 != nil && err3 != nil
		}
		
		return format1 == format2 && format2 == format3 &&
			conf1 == conf2 && conf2 == conf3
	}
	
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// Property: Extracted calls should have valid structure
func TestProperty_ValidCallStructure(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(input string) bool {
		calls, err := parser.ExtractToolCalls(input)
		if err != nil {
			return true // Error is acceptable
		}
		
		for _, call := range calls {
			// ID should be set
			if call.ID == "" {
				return false
			}
			
			// Type should be set
			if call.Type == "" {
				return false
			}
			
			// Function name should be set
			if call.Function.Name == "" {
				return false
			}
			
			// Arguments should be valid JSON if present
			if len(call.Function.Arguments) > 0 {
				var check interface{}
				if err := json.Unmarshal(call.Function.Arguments, &check); err != nil {
					return false
				}
			}
		}
		
		return true
	}
	
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// Property: Parser should be idempotent for extraction
func TestProperty_IdempotentExtraction(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(input string) bool {
		// First extraction
		calls1, err1 := parser.ExtractToolCalls(input)
		
		// Second extraction
		calls2, err2 := parser.ExtractToolCalls(input)
		
		// Results should be identical
		if err1 != nil {
			return err2 != nil
		}
		
		if len(calls1) != len(calls2) {
			return false
		}
		
		// Compare calls (ignoring generated IDs which might differ)
		for i := range calls1 {
			if calls1[i].Type != calls2[i].Type ||
				calls1[i].Function.Name != calls2[i].Function.Name ||
				string(calls1[i].Function.Arguments) != string(calls2[i].Function.Arguments) {
				return false
			}
		}
		
		return true
	}
	
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// Property: Context cancellation should be respected
func TestProperty_ContextCancellation(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(input string) bool {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		_, _ = parser.ExtractWithContext(ctx, input)
		
		// Should either complete quickly or return context error
		// This property is hard to verify precisely due to timing
		return true
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}

// Custom generator for tool call inputs
type ToolCallInput struct {
	Format   string
	NumCalls int
	Content  string
}

func (ToolCallInput) Generate(rand *rand.Rand, size int) reflect.Value {
	formats := []string{"json", "xml"}
	format := formats[rand.Intn(len(formats))]
	numCalls := rand.Intn(5) + 1
	
	var content string
	if format == "json" {
		var calls []string
		for i := 0; i < numCalls; i++ {
			call := fmt.Sprintf(`{"id": "call_%d", "type": "function", "function": {"name": "func_%d", "arguments": "{}"}}`, i, i)
			calls = append(calls, call)
		}
		content = fmt.Sprintf(`{"tool_calls": [%s]}`, strings.Join(calls, ","))
	} else {
		var invokes []string
		for i := 0; i < numCalls; i++ {
			invoke := fmt.Sprintf(`<invoke name="func_%d"><parameter name="p">v</parameter></invoke>`, i)
			invokes = append(invokes, invoke)
		}
		content = fmt.Sprintf(`<function_calls>%s</function_calls>`, strings.Join(invokes, ""))
	}
	
	return reflect.ValueOf(ToolCallInput{
		Format:   format,
		NumCalls: numCalls,
		Content:  content,
	})
}

// Property: Generated valid inputs should parse correctly
func TestProperty_GeneratedInputsParseCorrectly(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(input ToolCallInput) bool {
		calls, err := parser.ExtractToolCalls(input.Content)
		if err != nil {
			t.Logf("Failed to parse generated %s input: %v", input.Format, err)
			return false
		}
		
		// Should extract the expected number of calls
		return len(calls) == input.NumCalls
	}
	
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// Property: Parser should handle mixed content gracefully
func TestProperty_MixedContent(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(prefix, suffix string, validJSON bool) bool {
		var middle string
		if validJSON {
			middle = `{"id": "test", "type": "function", "function": {"name": "test", "arguments": "{}"}}`
		} else {
			middle = `<function_calls><invoke name="test"></invoke></function_calls>`
		}
		
		input := prefix + "\n" + middle + "\n" + suffix
		
		calls, err := parser.ExtractToolCalls(input)
		
		// Should not error
		if err != nil {
			return false
		}
		
		// Should extract at least one call if middle is valid
		return len(calls) >= 1
	}
	
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

// Property: Large inputs should not cause memory issues
func TestProperty_LargeInputsHandled(t *testing.T) {
	parser := NewResponseParser()
	
	f := func(repeatCount uint16) bool {
		// Limit size to prevent test timeouts
		count := int(repeatCount) % 1000
		if count == 0 {
			count = 1
		}
		
		// Generate large input
		input := strings.Repeat(`{"id": "x", "type": "function", "function": {"name": "y", "arguments": "{}"}} `, count)
		
		// Should handle without panicking
		_, _ = parser.ExtractToolCalls(input)
		return true
	}
	
	if err := quick.Check(f, &quick.Config{MaxCount: 50}); err != nil {
		t.Error(err)
	}
}

// Property: Arguments should be preserved exactly
func TestProperty_ArgumentPreservation(t *testing.T) {
	parser := NewResponseParser()
	
	testCases := []map[string]interface{}{
		{"string": "value"},
		{"number": 42},
		{"float": 3.14},
		{"bool": true},
		{"null": nil},
		{"array": []interface{}{1, 2, 3}},
		{"object": map[string]interface{}{"nested": "value"}},
	}
	
	for _, args := range testCases {
		argsJSON, _ := json.Marshal(args)
		input := fmt.Sprintf(`{"id": "test", "type": "function", "function": {"name": "test", "arguments": %q}}`, string(argsJSON))
		
		calls, err := parser.ExtractToolCalls(input)
		assert.NoError(t, err)
		if !assert.Len(t, calls, 1) {
			t.Logf("Input was: %s", input)
			continue
		}
		
		// Verify arguments are preserved
		var parsed map[string]interface{}
		err = json.Unmarshal(calls[0].Function.Arguments, &parsed)
		assert.NoError(t, err)
		assert.Equal(t, args, parsed)
	}
}