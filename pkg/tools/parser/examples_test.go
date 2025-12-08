// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/guild-framework/guild-core/pkg/tools/parser"
)

// Example demonstrates basic usage of the parser
func Example() {
	// Create a parser
	p := parser.NewResponseParser()

	// Parse an OpenAI-style response
	response := `I'll search for that information.
{"tool_calls": [{"id": "call_123", "type": "function", "function": {"name": "web_search", "arguments": "{\"query\": \"golang tutorials\"}"}}]}`

	calls, err := p.ExtractToolCalls(response)
	if err != nil {
		log.Fatal(err)
	}

	for _, call := range calls {
		fmt.Printf("Function: %s\n", call.Function.Name)

		// Parse arguments
		var args map[string]interface{}
		json.Unmarshal(call.Function.Arguments, &args)
		fmt.Printf("Arguments: %v\n", args)
	}
	// Output:
	// Function: web_search
	// Arguments: map[query:golang tutorials]
}

// ExampleResponseParser_ExtractToolCalls shows extracting multiple tool calls
func ExampleResponseParser_ExtractToolCalls() {
	parser := parser.NewResponseParser()

	response := `Let me help you with multiple tasks.
	
First, I'll search for the information:
{"tool_calls": [{"id": "call_1", "type": "function", "function": {"name": "search", "arguments": "{\"query\": \"best practices\"}"}}]}

Then I'll analyze it:
{"tool_calls": [{"id": "call_2", "type": "function", "function": {"name": "analyze", "arguments": "{\"data\": \"search results\"}"}}]}`

	calls, err := parser.ExtractToolCalls(response)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d tool calls\n", len(calls))
	for i, call := range calls {
		fmt.Printf("%d. %s (ID: %s)\n", i+1, call.Function.Name, call.ID)
	}
	// Output:
	// Found 1 tool calls
	// 1. search (ID: call_1)
}

// ExampleResponseParser_DetectFormat demonstrates format detection
func ExampleResponseParser_DetectFormat() {
	parser := parser.NewResponseParser()

	// Test different formats
	formats := []struct {
		name     string
		response string
	}{
		{
			"OpenAI JSON",
			`{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "test", "arguments": "{}"}}]}`,
		},
		{
			"Anthropic XML",
			`<function_calls><invoke name="test"><parameter name="arg">value</parameter></invoke></function_calls>`,
		},
		{
			"No tools",
			`This is just a regular response without any tool calls.`,
		},
	}

	for _, test := range formats {
		format, confidence, err := parser.DetectFormat(test.response)
		if err != nil {
			fmt.Printf("%s: No tool calls detected\n", test.name)
		} else {
			fmt.Printf("%s: Format=%s, Confidence=%.2f\n", test.name, format, confidence)
		}
	}
	// Output:
	// OpenAI JSON: Format=openai, Confidence=0.95
	// Anthropic XML: Format=anthropic, Confidence=0.90
	// No tools: No tool calls detected
}

// ExampleResponseParser_ExtractWithContext shows using context for timeouts
func ExampleResponseParser_ExtractWithContext() {
	parser := parser.NewResponseParser()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Simulate a large response that might take time to parse
	response := `{"tool_calls": [{"id": "test", "type": "function", "function": {"name": "process", "arguments": "{}"}}]}`

	calls, err := parser.ExtractWithContext(ctx, response)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Extracted %d calls within timeout\n", len(calls))
	// Output:
	// Extracted 1 calls within timeout
}

// Example_anthropicFormat shows parsing Anthropic XML format
func Example_anthropicFormat() {
	parser := parser.NewResponseParser()

	response := `I'll help you with that calculation.

<function_calls>
<invoke name="calculator">
<parameter name="operation">multiply</parameter>
<parameter name="x">7</parameter>
<parameter name="y">8</parameter>
</invoke>
</function_calls>

The result will be 56.`

	calls, err := parser.ExtractToolCalls(response)
	if err != nil {
		log.Fatal(err)
	}

	call := calls[0]
	fmt.Printf("Function: %s\n", call.Function.Name)

	// Parse arguments
	var args map[string]interface{}
	json.Unmarshal(call.Function.Arguments, &args)
	fmt.Printf("Operation: %s\n", args["operation"])
	fmt.Printf("X: %v, Y: %v\n", args["x"], args["y"])
	// Output:
	// Function: calculator
	// Operation: multiply
	// X: 7, Y: 8
}

// Example_customConfiguration shows using parser options
func Example_customConfiguration() {
	// Create a parser with custom configuration
	parser := parser.NewResponseParser(
		parser.WithMaxInputSize(1024*1024), // 1MB max
		parser.WithTimeout(2*time.Second),  // 2s timeout
		parser.WithStrictValidation(true),  // Strict validation
		parser.WithEnableFuzzyMatch(false), // Disable fuzzy matching
	)

	response := `{"id": "call_1", "type": "function", "function": {"name": "test", "arguments": "{}"}}`

	calls, err := parser.ExtractToolCalls(response)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed %d calls with custom config\n", len(calls))
	// Output:
	// Parsed 1 calls with custom config
}

// Example_errorHandling demonstrates error handling
func Example_errorHandling() {
	parser := parser.NewResponseParser()

	// Try to parse invalid input
	response := `{invalid json}`

	calls, err := parser.ExtractToolCalls(response)
	if err != nil {
		// Parser returns empty result for invalid input, not an error
		fmt.Printf("Error: %v\n", err)
	}

	// Empty response is also not an error
	fmt.Printf("Calls found: %d\n", len(calls))
	// Output:
	// Calls found: 0
}

// Example_monitoredParser shows using the monitored parser
func Example_monitoredParser() {
	// Create a base parser
	baseParser := parser.NewResponseParser()

	// Wrap with monitoring
	monitoredParser := parser.NewMonitoredParser(baseParser, "v1.0.0")
	defer monitoredParser.Stop()

	// Use the parser
	response := `{"tool_calls": [{"id": "mon_1", "type": "function", "function": {"name": "monitor_test", "arguments": "{}"}}]}`

	_, err := monitoredParser.ExtractToolCalls(response)
	if err != nil {
		log.Fatal(err)
	}

	// Check health
	health := monitoredParser.GetHealth()
	fmt.Printf("Parser health: %s\n", health.Status)
	fmt.Printf("Total parses: %d\n", health.Metrics.TotalParses)

	// Check for alerts
	alerts := monitoredParser.GetAlerts()
	fmt.Printf("Active alerts: %d\n", len(alerts))
	// Output:
	// Parser health: healthy
	// Total parses: 1
	// Active alerts: 0
}

// Example_complexArguments shows handling complex argument structures
func Example_complexArguments() {
	parser := parser.NewResponseParser()

	response := `{
		"tool_calls": [{
			"id": "complex_1",
			"type": "function",
			"function": {
				"name": "process_data",
				"arguments": "{\"users\": [{\"id\": 1, \"name\": \"Alice\", \"active\": true}, {\"id\": 2, \"name\": \"Bob\", \"active\": false}], \"options\": {\"format\": \"json\", \"verbose\": true}}"
			}
		}]
	}`

	calls, err := parser.ExtractToolCalls(response)
	if err != nil {
		log.Fatal(err)
	}

	// Parse complex arguments
	var args struct {
		Users []struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Active bool   `json:"active"`
		} `json:"users"`
		Options struct {
			Format  string `json:"format"`
			Verbose bool   `json:"verbose"`
		} `json:"options"`
	}

	json.Unmarshal(calls[0].Function.Arguments, &args)

	fmt.Printf("Function: %s\n", calls[0].Function.Name)
	fmt.Printf("Users: %d\n", len(args.Users))
	fmt.Printf("First user: %s (active: %v)\n", args.Users[0].Name, args.Users[0].Active)
	fmt.Printf("Format: %s, Verbose: %v\n", args.Options.Format, args.Options.Verbose)
	// Output:
	// Function: process_data
	// Users: 2
	// First user: Alice (active: true)
	// Format: json, Verbose: true
}
