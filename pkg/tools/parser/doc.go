// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

/*
Package parser provides a robust tool call extraction system for parsing
function calls from Large Language Model (LLM) responses.

The parser supports multiple provider formats including OpenAI's JSON format
and Anthropic's XML format, with automatic format detection and confidence
scoring. It's designed for production use with comprehensive error handling,
observability, and performance optimization.

# Basic Usage

The simplest way to use the parser is to create a ResponseParser and call
ExtractToolCalls:

	parser := parser.NewResponseParser()
	calls, err := parser.ExtractToolCalls(llmResponse)
	if err != nil {
		// Handle error
	}
	
	for _, call := range calls {
		fmt.Printf("Function: %s\n", call.Function.Name)
		// Process the tool call
	}

# Format Detection

The parser automatically detects the format of tool calls in the response:

	format, confidence, err := parser.DetectFormat(response)
	if err != nil {
		// No tool calls detected
		return
	}
	
	switch format {
	case parser.ProviderFormatOpenAI:
		// OpenAI JSON format detected
	case parser.ProviderFormatAnthropic:
		// Anthropic XML format detected
	}

# Advanced Configuration

The parser can be configured with various options:

	parser := parser.NewResponseParser(
		parser.WithMaxInputSize(10 * 1024 * 1024),  // 10MB max input
		parser.WithTimeout(30 * time.Second),        // 30s timeout
		parser.WithStrictValidation(true),           // Enable strict validation
		parser.WithEnableFuzzyMatch(true),           // Enable fuzzy matching
	)

# Context Support

For better control over parsing operations, use context:

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	calls, err := parser.ExtractWithContext(ctx, response)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			// Parsing timed out
		}
	}

# Observability

The parser includes comprehensive observability features:

	// Create an instrumented parser with metrics and tracing
	parser := parser.InstrumentParser(baseParser)
	
	// Create a monitored parser with health checks and alerting
	monitoredParser := parser.NewMonitoredParser(parser, "v1.0.0")
	defer monitoredParser.Stop()
	
	// Check health
	health := monitoredParser.GetHealth()
	if health.Status != parser.HealthStatusHealthy {
		// Handle degraded parser
	}
	
	// Get active alerts
	alerts := monitoredParser.GetAlerts()

# Custom Formats

To add support for a custom format, implement the FormatDetector and
FormatParser interfaces:

	type CustomDetector struct{}
	
	func (d *CustomDetector) Format() ProviderFormat {
		return "custom"
	}
	
	func (d *CustomDetector) CanParse(input []byte) bool {
		// Quick check if this format might apply
	}
	
	func (d *CustomDetector) Detect(ctx context.Context, input []byte) (DetectionResult, error) {
		// Detailed detection with confidence scoring
	}
	
	// Then register with the parser
	parser := parser.NewResponseParser(
		parser.WithCustomDetector(customDetector),
		parser.WithCustomParser("custom", customParser),
	)

# Error Handling

The parser uses gerror for rich error information:

	calls, err := parser.ExtractToolCalls(response)
	if err != nil {
		if gerr, ok := err.(*gerror.Error); ok {
			fmt.Printf("Error: %s (code: %s)\n", gerr.Message(), gerr.Code())
			fmt.Printf("Component: %s, Operation: %s\n", 
				gerr.Component(), gerr.Operation())
		}
	}

# Thread Safety

All parser types are safe for concurrent use. A single parser instance
can be shared across multiple goroutines:

	parser := parser.NewResponseParser()
	
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(response string) {
			defer wg.Done()
			calls, _ := parser.ExtractToolCalls(response)
			// Process calls
		}(responses[i])
	}
	wg.Wait()

# Performance Considerations

For optimal performance:

1. Reuse parser instances instead of creating new ones
2. Use context with timeouts for large inputs
3. Enable streaming mode for very large responses
4. Monitor metrics to identify bottlenecks

The parser is optimized for common cases while remaining robust for edge
cases and malformed inputs.
*/
package parser