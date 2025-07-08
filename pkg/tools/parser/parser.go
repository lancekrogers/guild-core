// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
	jsonparser "github.com/lancekrogers/guild/pkg/tools/parser/json"
	xmlparser "github.com/lancekrogers/guild/pkg/tools/parser/xml"
)

// RobustParser implements ResponseParser with proper format detection and validation
type RobustParser struct {
	config      *parserConfig
	detectorReg *DetectorRegistry
	parsers     map[ProviderFormat]FormatParser
	mu          sync.RWMutex

	// Metrics
	parseAttempts  int64
	parseSuccesses int64
	parseFailures  int64
}

// NewResponseParser creates a new robust parser with all provider support
func NewResponseParser(opts ...ParserOption) ResponseParser {
	config := &parserConfig{
		maxInputSize:     10 * 1024 * 1024, // 10MB default
		enableFuzzyMatch: true,
		strictValidation: false,
		timeout:          5 * time.Second,
		customDetectors:  []FormatDetector{},
		customParsers:    make(map[ProviderFormat]FormatParser),
	}

	for _, opt := range opts {
		opt(config)
	}

	p := &RobustParser{
		config:      config,
		detectorReg: NewDetectorRegistry(),
		parsers:     make(map[ProviderFormat]FormatParser),
	}

	// Register default detectors and parsers
	p.registerDefaults()

	// Add custom detectors
	for _, detector := range config.customDetectors {
		p.detectorReg.Register(detector)
	}

	// Add custom parsers
	for format, parser := range config.customParsers {
		p.parsers[format] = parser
	}

	return p
}

// registerDefaults sets up the default detectors and parsers
func (p *RobustParser) registerDefaults() {
	// Register JSON detector and parser for OpenAI
	jsonDetector := jsonparser.NewDetector()
	p.detectorReg.Register(jsonDetector)
	p.parsers[ProviderFormatOpenAI] = jsonparser.NewParser()

	// Register XML detector and parser for Anthropic
	xmlDetector := xmlparser.NewDetector()
	p.detectorReg.Register(xmlDetector)
	p.parsers[ProviderFormatAnthropic] = xmlparser.NewParser()

	// Add other format support as needed
}

// ExtractToolCalls parses tool calls from any supported format
func (p *RobustParser) ExtractToolCalls(response string) ([]ToolCall, error) {
	ctx := context.Background()
	return p.ExtractWithContext(ctx, response)
}

// ExtractWithContext parses with context support for cancellation and timeouts
func (p *RobustParser) ExtractWithContext(ctx context.Context, response string) ([]ToolCall, error) {
	p.parseAttempts++

	// Apply timeout from config
	ctx, cancel := context.WithTimeout(ctx, p.config.timeout)
	defer cancel()

	logger := observability.GetLogger(ctx).
		WithComponent("parser").
		WithOperation("ExtractToolCalls")

	// Input validation
	if response == "" {
		p.parseFailures++
		return []ToolCall{}, nil // Empty response is not an error
	}

	responseBytes := []byte(response)
	if len(responseBytes) > p.config.maxInputSize {
		p.parseFailures++
		return nil, gerror.New(gerror.ErrCodeValidation, "input too large", nil).
			WithComponent("parser").
			WithOperation("ExtractToolCalls").
			WithDetails("size", len(responseBytes)).
			WithDetails("max_size", p.config.maxInputSize)
	}

	// Normalize input
	normalized := p.normalizeInput(responseBytes)

	// Detect format
	format, confidence, err := p.detectFormat(ctx, normalized)
	if err != nil {
		logger.Debug("No tool calls detected",
			"error", err,
			"response_preview", truncate(string(normalized), 200),
		)
		p.parseFailures++
		return []ToolCall{}, nil // No tool calls found is not an error
	}

	logger.Info("Format detected",
		"format", format,
		"confidence", confidence,
	)

	// Get parser
	p.mu.RLock()
	parser, exists := p.parsers[format]
	p.mu.RUnlock()

	if !exists {
		p.parseFailures++
		return nil, gerror.New(gerror.ErrCodeNotFound, "no parser for format", nil).
			WithComponent("parser").
			WithOperation("ExtractToolCalls").
			WithDetails("format", string(format))
	}

	// Validate if strict mode
	if p.config.strictValidation {
		validation := parser.Validate(normalized)
		if !validation.Valid {
			logger.Warn("Validation failed",
				"errors", validation.Errors,
				"warnings", validation.Warnings,
			)
			// Continue parsing anyway - might be recoverable
		}
	}

	// Parse tool calls
	calls, err := parser.Parse(ctx, normalized)
	if err != nil {
		p.parseFailures++
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "parsing failed").
			WithComponent("parser").
			WithOperation("ExtractToolCalls").
			WithDetails("format", string(format))
	}

	// Post-process and validate
	calls = p.postProcess(calls)

	p.parseSuccesses++
	return calls, nil
}

// DetectFormat identifies the format with confidence score
func (p *RobustParser) DetectFormat(response string) (ProviderFormat, float64, error) {
	ctx := context.Background()
	return p.detectFormat(ctx, []byte(response))
}

// detectFormat internal implementation with context
func (p *RobustParser) detectFormat(ctx context.Context, input []byte) (ProviderFormat, float64, error) {
	format, confidence, err := p.detectorReg.DetectFormat(ctx, input)
	if err != nil {
		return ProviderFormatUnknown, 0, err
	}

	// Require minimum confidence
	if confidence < 0.5 {
		return ProviderFormatUnknown, confidence,
			gerror.New(gerror.ErrCodeValidation, "confidence too low", nil).
				WithComponent("parser").
				WithOperation("detectFormat").
				WithDetails("confidence", fmt.Sprintf("%.2f", confidence))
	}

	return format, confidence, nil
}

// normalizeInput cleans and normalizes input data
func (p *RobustParser) normalizeInput(input []byte) []byte {
	// Remove BOM if present
	input = bytes.TrimPrefix(input, []byte{0xEF, 0xBB, 0xBF})

	// Trim excessive whitespace
	input = bytes.TrimSpace(input)

	// Handle common encoding issues would go here

	return input
}

// postProcess validates and normalizes parsed tool calls
func (p *RobustParser) postProcess(calls []ToolCall) []ToolCall {
	processed := make([]ToolCall, 0, len(calls))

	for i, call := range calls {
		// Ensure ID is set
		if call.ID == "" {
			call.ID = fmt.Sprintf("call_%d_%d", time.Now().Unix(), i)
		}

		// Ensure type is set
		if call.Type == "" {
			call.Type = "function"
		}

		// Validate function name
		if call.Function.Name == "" {
			continue // Skip invalid calls
		}

		// Ensure arguments is valid JSON
		if len(call.Function.Arguments) == 0 {
			call.Function.Arguments = json.RawMessage("{}")
		} else {
			// Validate JSON
			var check interface{}
			if err := json.Unmarshal(call.Function.Arguments, &check); err != nil {
				// Try to fix common issues
				fixed := p.tryFixJSON(call.Function.Arguments)
				if fixed != nil {
					call.Function.Arguments = fixed
				} else {
					continue // Skip if unfixable
				}
			}
		}

		processed = append(processed, call)
	}

	return processed
}

// tryFixJSON attempts to fix common JSON issues
func (p *RobustParser) tryFixJSON(data json.RawMessage) json.RawMessage {
	// Try to parse as-is first
	var test interface{}
	if err := json.Unmarshal(data, &test); err == nil {
		return data
	}

	// Common fixes
	// 1. Single quotes to double quotes
	if bytes.Contains(data, []byte("'")) {
		fixed := bytes.Replace(data, []byte("'"), []byte(`"`), -1)
		if err := json.Unmarshal(fixed, &test); err == nil {
			return fixed
		}
	}

	// 2. Unquoted keys
	// This would require more complex parsing

	return nil
}

// truncate safely truncates a string for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// DetectorRegistry manages format detectors
type DetectorRegistry struct {
	detectors []FormatDetector
	mu        sync.RWMutex

	// Metrics
	detectionAttempts  int64
	detectionSuccesses int64
	detectionFailures  int64

	// Observable parser for metrics
	observable *ObservableParser
}

// NewDetectorRegistry creates a new detector registry
func NewDetectorRegistry() *DetectorRegistry {
	return &DetectorRegistry{
		detectors: make([]FormatDetector, 0),
	}
}

// Register adds a format detector
func (r *DetectorRegistry) Register(detector FormatDetector) error {
	if detector == nil {
		return gerror.New(gerror.ErrCodeValidation, "detector cannot be nil", nil).
			WithComponent("parser").
			WithOperation("DetectorRegistry.Register")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.detectors = append(r.detectors, detector)
	return nil
}

// DetectFormat finds the best matching format
func (r *DetectorRegistry) DetectFormat(ctx context.Context, input []byte) (ProviderFormat, float64, error) {
	r.mu.RLock()
	detectors := make([]FormatDetector, len(r.detectors))
	copy(detectors, r.detectors)
	r.mu.RUnlock()

	r.detectionAttempts++

	if len(detectors) == 0 {
		r.detectionFailures++
		return ProviderFormatUnknown, 0, gerror.New(gerror.ErrCodeNotFound, "no detectors registered", nil)
	}

	// Run detectors and find best match
	type result struct {
		detection DetectionResult
		err       error
	}

	results := make([]result, 0, len(detectors))

	for _, detector := range detectors {
		if !detector.CanParse(input) {
			continue
		}

		detection, err := detector.Detect(ctx, input)
		if err == nil && detection.Confidence > 0 {
			results = append(results, result{detection: detection})
		}
	}

	if len(results) == 0 {
		r.detectionFailures++
		return ProviderFormatUnknown, 0, gerror.New(gerror.ErrCodeNotFound, "no format detected", nil)
	}

	// Find highest confidence
	best := results[0]
	for _, r := range results[1:] {
		if r.detection.Confidence > best.detection.Confidence {
			best = r
		}
	}

	r.detectionSuccesses++
	return best.detection.Format, best.detection.Confidence, nil
}
