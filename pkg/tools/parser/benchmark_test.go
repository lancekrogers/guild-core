// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// Benchmark data
var (
	smallJSON = `{"id": "call_123", "type": "function", "function": {"name": "test", "arguments": "{}"}}`
	
	mediumJSON = `{"tool_calls": [
		{"id": "call_1", "type": "function", "function": {"name": "search", "arguments": "{\"query\": \"golang benchmarks\", \"limit\": 10}"}},
		{"id": "call_2", "type": "function", "function": {"name": "analyze", "arguments": "{\"data\": [1,2,3,4,5], \"method\": \"mean\"}"}}
	]}`
	
	smallXML = `<function_calls><invoke name="test"><parameter name="arg">value</parameter></invoke></function_calls>`
	
	mediumXML = `<function_calls>
		<invoke name="search">
			<parameter name="query">golang benchmarks</parameter>
			<parameter name="limit">10</parameter>
		</invoke>
		<invoke name="analyze">
			<parameter name="data">[1,2,3,4,5]</parameter>
			<parameter name="method">mean</parameter>
		</invoke>
	</function_calls>`
)

// BenchmarkParser_JSON tests JSON parsing performance
func BenchmarkParser_JSON_Small(b *testing.B) {
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(smallJSON)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 1 {
			b.Fatalf("expected 1 call, got %d", len(calls))
		}
	}
}

func BenchmarkParser_JSON_Medium(b *testing.B) {
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(mediumJSON)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 2 {
			b.Fatalf("expected 2 calls, got %d", len(calls))
		}
	}
}

func BenchmarkParser_JSON_Large(b *testing.B) {
	// Generate large JSON with many calls
	var calls []string
	for i := 0; i < 50; i++ {
		calls = append(calls, fmt.Sprintf(`{
			"id": "call_%d",
			"type": "function",
			"function": {
				"name": "process_%d",
				"arguments": "{\"index\": %d, \"data\": \"%s\"}"
			}
		}`, i, i, i, strings.Repeat("x", 100)))
	}
	largeJSON := fmt.Sprintf(`{"tool_calls": [%s]}`, strings.Join(calls, ","))
	
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(largeJSON)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 50 {
			b.Fatalf("expected 50 calls, got %d", len(calls))
		}
	}
}

// BenchmarkParser_XML tests XML parsing performance
func BenchmarkParser_XML_Small(b *testing.B) {
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(smallXML)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 1 {
			b.Fatalf("expected 1 call, got %d", len(calls))
		}
	}
}

func BenchmarkParser_XML_Medium(b *testing.B) {
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(mediumXML)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 2 {
			b.Fatalf("expected 2 calls, got %d", len(calls))
		}
	}
}

func BenchmarkParser_XML_Large(b *testing.B) {
	// Generate large XML with many calls
	var invokes []string
	for i := 0; i < 50; i++ {
		invokes = append(invokes, fmt.Sprintf(`
		<invoke name="process_%d">
			<parameter name="index">%d</parameter>
			<parameter name="data">%s</parameter>
		</invoke>`, i, i, strings.Repeat("x", 100)))
	}
	largeXML := fmt.Sprintf(`<function_calls>%s</function_calls>`, strings.Join(invokes, ""))
	
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(largeXML)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 50 {
			b.Fatalf("expected 50 calls, got %d", len(calls))
		}
	}
}

// BenchmarkParser_MixedContent tests parsing with surrounding text
func BenchmarkParser_MixedContent_JSON(b *testing.B) {
	mixed := fmt.Sprintf("%s\n\n%s\n\n%s", 
		strings.Repeat("This is some context before the JSON. ", 100),
		mediumJSON,
		strings.Repeat("This is some context after the JSON. ", 100))
	
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(mixed)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 2 {
			b.Fatalf("expected 2 calls, got %d", len(calls))
		}
	}
}

func BenchmarkParser_MixedContent_XML(b *testing.B) {
	mixed := fmt.Sprintf("%s\n\n%s\n\n%s", 
		strings.Repeat("This is some context before the XML. ", 100),
		mediumXML,
		strings.Repeat("This is some context after the XML. ", 100))
	
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(mixed)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 2 {
			b.Fatalf("expected 2 calls, got %d", len(calls))
		}
	}
}

// BenchmarkParser_FormatDetection tests format detection performance
func BenchmarkParser_FormatDetection_JSON(b *testing.B) {
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		format, confidence, err := parser.DetectFormat(mediumJSON)
		if err != nil {
			b.Fatal(err)
		}
		if format != ProviderFormatOpenAI {
			b.Fatalf("expected OpenAI format, got %s", format)
		}
		if confidence < 0.5 {
			b.Fatalf("confidence too low: %f", confidence)
		}
	}
}

func BenchmarkParser_FormatDetection_XML(b *testing.B) {
	parser := NewResponseParser()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		format, confidence, err := parser.DetectFormat(mediumXML)
		if err != nil {
			b.Fatal(err)
		}
		if format != ProviderFormatAnthropic {
			b.Fatalf("expected Anthropic format, got %s", format)
		}
		if confidence < 0.5 {
			b.Fatalf("confidence too low: %f", confidence)
		}
	}
}

// BenchmarkParser_Concurrent tests concurrent parsing performance
func BenchmarkParser_Concurrent(b *testing.B) {
	parser := NewResponseParser()
	inputs := []string{smallJSON, mediumJSON, smallXML, mediumXML}
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			input := inputs[i%len(inputs)]
			calls, err := parser.ExtractToolCalls(input)
			if err != nil {
				b.Fatal(err)
			}
			if len(calls) == 0 {
				b.Fatal("no calls extracted")
			}
			i++
		}
	})
}

// BenchmarkParser_WithContext tests context overhead
func BenchmarkParser_WithContext(b *testing.B) {
	parser := NewResponseParser()
	ctx := context.Background()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractWithContext(ctx, mediumJSON)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 2 {
			b.Fatalf("expected 2 calls, got %d", len(calls))
		}
	}
}

// Memory allocation benchmarks
func BenchmarkParser_Allocations_JSON(b *testing.B) {
	parser := NewResponseParser()
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = parser.ExtractToolCalls(mediumJSON)
	}
}

func BenchmarkParser_Allocations_XML(b *testing.B) {
	parser := NewResponseParser()
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = parser.ExtractToolCalls(mediumXML)
	}
}

// Comparative benchmarks
func BenchmarkParser_Compare_SimpleVsComplex(b *testing.B) {
	parser := NewResponseParser()
	
	simple := `{"id": "c1", "type": "function", "function": {"name": "test", "arguments": "{}"}}`
	complex := `{"tool_calls": [{"id": "c1", "type": "function", "function": {"name": "test", "arguments": "{}"}}]}`
	
	b.Run("Simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = parser.ExtractToolCalls(simple)
		}
	})
	
	b.Run("Complex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = parser.ExtractToolCalls(complex)
		}
	})
}

// BenchmarkParser_RealWorld tests with realistic inputs
func BenchmarkParser_RealWorld_ChatGPT(b *testing.B) {
	parser := NewResponseParser()
	
	// Realistic ChatGPT response
	input := `I'll help you analyze that data. Let me first fetch the current information and then process it.

{"tool_calls": [{"id": "call_JlbpwFCg7t8R1HXuBOxtaVxE", "type": "function", "function": {"name": "fetch_data", "arguments": "{\"source\": \"api\", \"filters\": {\"date_from\": \"2024-01-01\", \"date_to\": \"2024-12-31\", \"status\": \"active\"}, \"fields\": [\"id\", \"name\", \"value\", \"timestamp\"]}"}}]}

Now I'll process this data to generate the insights you're looking for.`
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(input)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 1 {
			b.Fatalf("expected 1 call, got %d", len(calls))
		}
	}
}

func BenchmarkParser_RealWorld_Claude(b *testing.B) {
	parser := NewResponseParser()
	
	// Realistic Claude response
	input := `I understand you need help with data analysis. Let me fetch and process that information for you.

<function_calls>
<invoke name="fetch_data">
<parameter name="source">api</parameter>
<parameter name="filters">{"date_from": "2024-01-01", "date_to": "2024-12-31", "status": "active"}</parameter>
<parameter name="fields">["id", "name", "value", "timestamp"]</parameter>
</invoke>
</function_calls>

I'm now retrieving the data based on your criteria. Once I have it, I'll analyze it and provide you with the insights.`
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		calls, err := parser.ExtractToolCalls(input)
		if err != nil {
			b.Fatal(err)
		}
		if len(calls) != 1 {
			b.Fatalf("expected 1 call, got %d", len(calls))
		}
	}
}