// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer name for the parser package
const tracerName = "github.com/guild-framework/guild-core/pkg/tools/parser"

// getTracer returns the tracer for the parser package
func getTracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// TracedParser wraps a parser with distributed tracing
type TracedParser struct {
	parser ResponseParser
	tracer trace.Tracer
}

// NewTracedParser creates a parser with distributed tracing
func NewTracedParser(parser ResponseParser) *TracedParser {
	return &TracedParser{
		parser: parser,
		tracer: getTracer(),
	}
}

// ExtractToolCalls wraps the parser with tracing
func (t *TracedParser) ExtractToolCalls(response string) ([]ToolCall, error) {
	ctx := context.Background()
	return t.ExtractWithContext(ctx, response)
}

// ExtractWithContext wraps the parser with tracing and context
func (t *TracedParser) ExtractWithContext(ctx context.Context, response string) ([]ToolCall, error) {
	ctx, span := t.tracer.Start(ctx, "parser.ExtractToolCalls",
		trace.WithAttributes(
			attribute.Int("input.size", len(response)),
		),
	)
	defer span.End()

	// Detect format with tracing
	format, confidence, err := t.detectFormatWithTracing(ctx, response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "format detection failed")
		return nil, err
	}

	span.SetAttributes(
		attribute.String("format.detected", string(format)),
		attribute.Float64("format.confidence", confidence),
	)

	// Parse with tracing
	calls, err := t.parser.ExtractWithContext(ctx, response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parsing failed")
		return nil, err
	}

	// Record successful parse
	span.SetAttributes(
		attribute.Int("tool_calls.count", len(calls)),
	)

	// Add tool call details as events
	for i, call := range calls {
		span.AddEvent(fmt.Sprintf("tool_call_%d", i),
			trace.WithAttributes(
				attribute.String("tool.name", call.Function.Name),
				attribute.String("tool.id", call.ID),
				attribute.String("tool.type", call.Type),
			),
		)
	}

	span.SetStatus(codes.Ok, "")
	return calls, nil
}

// DetectFormat wraps format detection with tracing
func (t *TracedParser) DetectFormat(response string) (ProviderFormat, float64, error) {
	ctx := context.Background()
	return t.detectFormatWithTracing(ctx, response)
}

// detectFormatWithTracing performs format detection with tracing
func (t *TracedParser) detectFormatWithTracing(ctx context.Context, response string) (ProviderFormat, float64, error) {
	ctx, span := t.tracer.Start(ctx, "parser.DetectFormat",
		trace.WithAttributes(
			attribute.Int("input.size", len(response)),
		),
	)
	defer span.End()

	format, confidence, err := t.parser.DetectFormat(response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "detection failed")
		return format, confidence, err
	}

	span.SetAttributes(
		attribute.String("format", string(format)),
		attribute.Float64("confidence", confidence),
	)
	span.SetStatus(codes.Ok, "")

	return format, confidence, nil
}

// TracedDetector wraps a format detector with tracing
type TracedDetector struct {
	detector FormatDetector
	tracer   trace.Tracer
}

// NewTracedDetector creates a detector with tracing
func NewTracedDetector(detector FormatDetector) *TracedDetector {
	return &TracedDetector{
		detector: detector,
		tracer:   getTracer(),
	}
}

// Format returns the format this detector handles
func (t *TracedDetector) Format() ProviderFormat {
	return t.detector.Format()
}

// CanParse performs quick pre-check with tracing
func (t *TracedDetector) CanParse(input []byte) bool {
	ctx := context.Background()
	ctx, span := t.tracer.Start(ctx, "detector.CanParse",
		trace.WithAttributes(
			attribute.String("detector.format", string(t.detector.Format())),
			attribute.Int("input.size", len(input)),
		),
	)
	defer span.End()

	canParse := t.detector.CanParse(input)
	span.SetAttributes(
		attribute.Bool("can_parse", canParse),
	)

	return canParse
}

// Detect analyzes input with tracing
func (t *TracedDetector) Detect(ctx context.Context, input []byte) (DetectionResult, error) {
	ctx, span := t.tracer.Start(ctx, "detector.Detect",
		trace.WithAttributes(
			attribute.String("detector.format", string(t.detector.Format())),
			attribute.Int("input.size", len(input)),
		),
	)
	defer span.End()

	result, err := t.detector.Detect(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "detection failed")
		return result, err
	}

	span.SetAttributes(
		attribute.String("detected.format", string(result.Format)),
		attribute.Float64("detected.confidence", result.Confidence),
	)

	// Add metadata as events
	for key, value := range result.Metadata {
		span.AddEvent("metadata",
			trace.WithAttributes(
				attribute.String("key", key),
				attribute.String("value", fmt.Sprintf("%v", value)),
			),
		)
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// TracedFormatParser wraps a format parser with tracing
type TracedFormatParser struct {
	parser FormatParser
	tracer trace.Tracer
}

// NewTracedFormatParser creates a format parser with tracing
func NewTracedFormatParser(parser FormatParser) *TracedFormatParser {
	return &TracedFormatParser{
		parser: parser,
		tracer: getTracer(),
	}
}

// Format returns the format this parser handles
func (t *TracedFormatParser) Format() ProviderFormat {
	return t.parser.Format()
}

// Parse extracts tool calls with tracing
func (t *TracedFormatParser) Parse(ctx context.Context, input []byte) ([]ToolCall, error) {
	ctx, span := t.tracer.Start(ctx, "format_parser.Parse",
		trace.WithAttributes(
			attribute.String("parser.format", string(t.parser.Format())),
			attribute.Int("input.size", len(input)),
		),
	)
	defer span.End()

	calls, err := t.parser.Parse(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "parsing failed")
		return nil, err
	}

	span.SetAttributes(
		attribute.Int("tool_calls.count", len(calls)),
	)

	// Record tool calls as events
	for i, call := range calls {
		span.AddEvent(fmt.Sprintf("parsed_tool_%d", i),
			trace.WithAttributes(
				attribute.String("name", call.Function.Name),
				attribute.String("id", call.ID),
			),
		)
	}

	span.SetStatus(codes.Ok, "")
	return calls, nil
}

// Validate checks input with tracing
func (t *TracedFormatParser) Validate(input []byte) ValidationResult {
	ctx := context.Background()
	ctx, span := t.tracer.Start(ctx, "format_parser.Validate",
		trace.WithAttributes(
			attribute.String("parser.format", string(t.parser.Format())),
			attribute.Int("input.size", len(input)),
		),
	)
	defer span.End()

	result := t.parser.Validate(input)

	span.SetAttributes(
		attribute.Bool("valid", result.Valid),
		attribute.Int("errors.count", len(result.Errors)),
		attribute.Int("warnings.count", len(result.Warnings)),
		attribute.String("schema", result.SchemaUsed),
	)

	// Record errors as events
	for _, err := range result.Errors {
		span.AddEvent("validation_error",
			trace.WithAttributes(
				attribute.String("path", err.Path),
				attribute.String("message", err.Message),
				attribute.String("code", err.Code),
			),
		)
	}

	// Record warnings as events
	for _, warning := range result.Warnings {
		span.AddEvent("validation_warning",
			trace.WithAttributes(
				attribute.String("message", warning),
			),
		)
	}

	if result.Valid {
		span.SetStatus(codes.Ok, "validation passed")
	} else {
		span.SetStatus(codes.Error, "validation failed")
	}

	return result
}

// InstrumentParser adds both metrics and tracing to a parser
func InstrumentParser(parser ResponseParser) ResponseParser {
	// Add tracing first
	traced := NewTracedParser(parser)

	// Then add metrics
	observable := NewObservableParser(traced)

	return observable
}
