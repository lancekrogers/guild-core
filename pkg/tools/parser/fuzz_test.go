// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build go1.18
// +build go1.18

package parser

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	jsonparser "github.com/guild-framework/guild-core/pkg/tools/parser/json"
	xmlparser "github.com/guild-framework/guild-core/pkg/tools/parser/xml"
)

// FuzzParser_ExtractToolCalls fuzzes the main parser entry point
func FuzzParser_ExtractToolCalls(f *testing.F) {
	parser := NewResponseParser()

	// Add seed corpus
	seeds := []string{
		`{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "test", "arguments": "{}"}}]}`,
		`<function_calls><invoke name="test"><parameter name="arg">value</parameter></invoke></function_calls>`,
		`{"id": "single", "type": "function", "function": {"name": "single", "arguments": "{}"}}`,
		`Mixed content with {"tool_calls": []} embedded`,
		``,
		`{`,
		`<`,
		`{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "test", "arguments": "{\"nested\": {\"deep\": true}}"}}]}`,
		string([]byte{0xFF, 0xFE, 0xFD}), // Invalid UTF-8
		`{"tool_calls": ` + `[` + `{"id": "x", "type": "function", "function": {"name": "y", "arguments": "{}"}}` + `]` + `}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// The parser should never panic
		calls, err := parser.ExtractToolCalls(input)

		// Basic invariants
		if err == nil {
			// If no error, calls should not be nil
			if calls == nil {
				t.Fatal("ExtractToolCalls returned nil calls with nil error")
			}

			// Validate each call
			for i, call := range calls {
				// ID should be set (either from input or generated)
				if call.ID == "" {
					t.Errorf("Call %d has empty ID", i)
				}

				// Type should be set
				if call.Type == "" {
					t.Errorf("Call %d has empty Type", i)
				}

				// Function name should be set
				if call.Function.Name == "" {
					t.Errorf("Call %d has empty Function.Name", i)
				}

				// Arguments should be valid JSON (even if empty)
				if len(call.Function.Arguments) > 0 {
					// Should be valid JSON
					var check interface{}
					if err := json.Unmarshal(call.Function.Arguments, &check); err != nil {
						t.Errorf("Call %d has invalid JSON arguments: %v", i, err)
					}
				}
			}
		}
	})
}

// FuzzParser_DetectFormat fuzzes format detection
func FuzzParser_DetectFormat(f *testing.F) {
	parser := NewResponseParser()

	// Add seed corpus
	seeds := []string{
		`{"tool_calls": []}`,
		`<function_calls></function_calls>`,
		`function_calls invoke parameter`,
		`{"mixed": "<xml>"}`,
		`<mixed>{"json": true}</mixed>`,
		``,
		string([]byte{0x00, 0x01, 0x02}),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		format, confidence, err := parser.DetectFormat(input)

		// Invariants
		if err == nil {
			// Format should be valid
			switch format {
			case ProviderFormatOpenAI, ProviderFormatAnthropic, ProviderFormatUnknown:
				// Valid formats
			default:
				t.Fatalf("Invalid format returned: %s", format)
			}

			// Confidence should be in valid range
			if confidence < 0 || confidence > 1 {
				t.Fatalf("Invalid confidence: %f", confidence)
			}

			// If format is unknown, confidence should be low
			if format == ProviderFormatUnknown && confidence > 0.5 {
				t.Errorf("Unknown format with high confidence: %f", confidence)
			}
		}
	})
}

// FuzzJSONParser_Parse fuzzes the JSON parser directly
func FuzzJSONParser_Parse(f *testing.F) {
	parser := jsonparser.NewParser() // JSON parser

	// Add JSON-specific seeds
	seeds := [][]byte{
		[]byte(`{"id": "test", "type": "function", "function": {"name": "test", "arguments": "{}"}}`),
		[]byte(`{"tool_calls": []}`),
		[]byte(`{"function": {"name": "x", "arguments": null}}`),
		[]byte(`{"id": "test", "type": "function", "function": {"name": "test", "arguments": "{\"a\": \"b\"}"}}`),
		[]byte(`[{"id": "1"}, {"id": "2"}]`),
		[]byte(`{}`),
		[]byte(`[]`),
		[]byte(`null`),
		[]byte(`"string"`),
		[]byte(`123`),
		[]byte(`true`),
		[]byte(`{{{{{`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		ctx := context.Background()
		calls, err := parser.Parse(ctx, input)

		// Should not panic
		if err == nil && calls == nil {
			t.Fatal("Parse returned nil calls with nil error")
		}

		// Validate calls if successful
		if err == nil {
			for _, call := range calls {
				if call.Function.Name == "" {
					t.Error("Parsed call with empty function name")
				}
			}
		}
	})
}

// FuzzXMLParser_Parse fuzzes the XML parser directly
func FuzzXMLParser_Parse(f *testing.F) {
	parser := xmlparser.NewParser() // XML parser

	// Add XML-specific seeds
	seeds := [][]byte{
		[]byte(`<function_calls><invoke name="test"/></function_calls>`),
		[]byte(`<function_calls></function_calls>`),
		[]byte(`<invoke name="test"><parameter name="arg">value</parameter></invoke>`),
		[]byte(`<?xml version="1.0"?><function_calls/>`),
		[]byte(`<function_calls>text content</function_calls>`),
		[]byte(`<<<<`),
		[]byte(`&lt;&gt;&amp;`),
		[]byte(`<function_calls xmlns="http://example.com"/>`),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		ctx := context.Background()
		calls, err := parser.Parse(ctx, input)

		// Should not panic
		if err == nil && calls == nil {
			t.Fatal("Parse returned nil calls with nil error")
		}

		// Validate calls if successful
		if err == nil {
			for _, call := range calls {
				if call.Function.Name == "" {
					t.Error("Parsed call with empty function name")
				}
			}
		}
	})
}

// FuzzParser_LargeInputs tests with potentially large inputs
func FuzzParser_LargeInputs(f *testing.F) {
	parser := NewResponseParser()

	// Add some large seeds
	f.Add(strings.Repeat(`{"id": "x", "type": "function", "function": {"name": "y", "arguments": "{}"}}`, 100))
	f.Add(strings.Repeat(`<invoke name="test"><parameter name="p">v</parameter></invoke>`, 100))

	f.Fuzz(func(t *testing.T, input string) {
		// Limit input size to prevent OOM
		if len(input) > 10*1024*1024 { // 10MB
			input = input[:10*1024*1024]
		}

		// Should handle large inputs without crashing
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, _ = parser.ExtractWithContext(ctx, input)
		// Just ensure it doesn't panic or hang
	})
}

// FuzzParser_MaliciousInputs tests potentially malicious inputs
func FuzzParser_MaliciousInputs(f *testing.F) {
	parser := NewResponseParser()

	// Add potentially malicious seeds
	seeds := []string{
		// Deeply nested JSON
		`{"a": {"b": {"c": {"d": {"e": {"f": {"g": {"h": {"i": {"j": {}}}}}}}}}}}`,
		// Repeated keys
		`{"id": "1", "id": "2", "id": "3"}`,
		// Very long strings
		`{"id": "` + strings.Repeat("x", 10000) + `"}`,
		// Unicode edge cases
		"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f",
		// XML bombs (simplified)
		`<!DOCTYPE foo [<!ENTITY bar "baz">]><function_calls>&bar;</function_calls>`,
		// Script injection attempts
		`<script>alert('xss')</script>{"tool_calls": []}`,
		`{"function": {"name": "<script>alert('xss')</script>", "arguments": "{}"}}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Parser should safely handle any input
		calls, _ := parser.ExtractToolCalls(input)

		// If calls are returned, they should be safe
		for _, call := range calls {
			// Function names should not contain script tags
			if strings.Contains(call.Function.Name, "<script>") {
				t.Error("Script tag in function name")
			}

			// Arguments should be valid JSON
			if len(call.Function.Arguments) > 0 {
				var check interface{}
				json.Unmarshal(call.Function.Arguments, &check)
				// Even if unmarshal fails, it shouldn't cause security issues
			}
		}
	})
}
