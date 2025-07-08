// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	
	"github.com/lancekrogers/guild/pkg/observability"
)

var (
	// globalMetrics is a singleton instance of parser metrics
	globalMetrics *ParserMetrics
	metricsOnce   sync.Once
)

// ParserMetrics contains all parser-specific metrics
type ParserMetrics struct {
	// Detection metrics
	detectionAttempts   *prometheus.CounterVec
	detectionSuccesses  *prometheus.CounterVec
	detectionFailures   *prometheus.CounterVec
	detectionDuration   *prometheus.HistogramVec
	detectionConfidence *prometheus.HistogramVec
	
	// Parsing metrics
	parseAttempts     *prometheus.CounterVec
	parseSuccesses    *prometheus.CounterVec
	parseFailures     *prometheus.CounterVec
	parseDuration     *prometheus.HistogramVec
	parseErrors       *prometheus.CounterVec
	
	// Tool call metrics
	toolCallsExtracted *prometheus.CounterVec
	toolCallsPerParse  *prometheus.HistogramVec
	
	// Format-specific metrics
	formatDistribution *prometheus.CounterVec
	
	// Input metrics
	inputSize          *prometheus.HistogramVec
	extractionLocation *prometheus.CounterVec
	
	// Validation metrics
	validationAttempts *prometheus.CounterVec
	validationFailures *prometheus.CounterVec
	validationWarnings *prometheus.CounterVec
}

// InitParserMetrics initializes parser-specific metrics
func InitParserMetrics(registry prometheus.Registerer) *ParserMetrics {
	metricsOnce.Do(func() {
		globalMetrics = createParserMetrics(registry)
	})
	return globalMetrics
}

// createParserMetrics creates a new ParserMetrics instance
func createParserMetrics(registry prometheus.Registerer) *ParserMetrics {
	m := &ParserMetrics{}
	
	// Detection metrics
	m.detectionAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "detection_attempts_total",
			Help:      "Total number of format detection attempts",
		},
		[]string{"detector"},
	)
	
	m.detectionSuccesses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "detection_successes_total",
			Help:      "Total number of successful format detections",
		},
		[]string{"detector", "format"},
	)
	
	m.detectionFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "detection_failures_total",
			Help:      "Total number of failed format detections",
		},
		[]string{"detector", "reason"},
	)
	
	m.detectionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "detection_duration_seconds",
			Help:      "Duration of format detection operations",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 10), // 0.1ms to ~100ms
		},
		[]string{"detector", "format"},
	)
	
	m.detectionConfidence = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "detection_confidence",
			Help:      "Confidence scores for format detection (0-1)",
			Buckets:   []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.99, 1.0},
		},
		[]string{"detector", "format"},
	)
	
	// Parsing metrics
	m.parseAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "parse_attempts_total",
			Help:      "Total number of parsing attempts",
		},
		[]string{"format"},
	)
	
	m.parseSuccesses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "parse_successes_total",
			Help:      "Total number of successful parses",
		},
		[]string{"format"},
	)
	
	m.parseFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "parse_failures_total",
			Help:      "Total number of failed parses",
		},
		[]string{"format", "reason"},
	)
	
	m.parseDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "parse_duration_seconds",
			Help:      "Duration of parsing operations",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"format"},
	)
	
	m.parseErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "parse_errors_total",
			Help:      "Total number of parse errors by type",
		},
		[]string{"format", "error_type"},
	)
	
	// Tool call metrics
	m.toolCallsExtracted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "tool_calls_extracted_total",
			Help:      "Total number of tool calls extracted",
		},
		[]string{"format", "tool_name"},
	)
	
	m.toolCallsPerParse = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "tool_calls_per_parse",
			Help:      "Number of tool calls extracted per parse operation",
			Buckets:   []float64{0, 1, 2, 3, 4, 5, 10, 20, 50},
		},
		[]string{"format"},
	)
	
	// Format distribution
	m.formatDistribution = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "format_distribution_total",
			Help:      "Distribution of detected formats",
		},
		[]string{"format"},
	)
	
	// Input metrics
	m.inputSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "input_size_bytes",
			Help:      "Size of parser input in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8), // 100B to ~10MB
		},
		[]string{"format"},
	)
	
	m.extractionLocation = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "extraction_location_total",
			Help:      "Location where tool calls were extracted from",
		},
		[]string{"location"},
	)
	
	// Validation metrics
	m.validationAttempts = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "validation_attempts_total",
			Help:      "Total number of validation attempts",
		},
		[]string{"format"},
	)
	
	m.validationFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "validation_failures_total",
			Help:      "Total number of validation failures",
		},
		[]string{"format", "error_code"},
	)
	
	m.validationWarnings = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "guild",
			Subsystem: "parser",
			Name:      "validation_warnings_total",
			Help:      "Total number of validation warnings",
		},
		[]string{"format"},
	)
	
	// Register all metrics
	registry.MustRegister(
		m.detectionAttempts, m.detectionSuccesses, m.detectionFailures,
		m.detectionDuration, m.detectionConfidence,
		m.parseAttempts, m.parseSuccesses, m.parseFailures,
		m.parseDuration, m.parseErrors,
		m.toolCallsExtracted, m.toolCallsPerParse,
		m.formatDistribution,
		m.inputSize, m.extractionLocation,
		m.validationAttempts, m.validationFailures, m.validationWarnings,
	)
	
	return m
}

// ObservableParser wraps a parser with metrics collection
type ObservableParser struct {
	parser  ResponseParser
	metrics *ParserMetrics
}

// NewObservableParser creates a parser with metrics collection
func NewObservableParser(parser ResponseParser) *ObservableParser {
	// Get the global metrics registry
	registry := observability.GetMetrics()
	if registry == nil {
		// If no global registry, create a new one
		registry = observability.InitGlobalMetrics(nil)
	}
	
	// Initialize parser metrics using the prometheus default registry
	metrics := InitParserMetrics(prometheus.DefaultRegisterer)
	
	return &ObservableParser{
		parser:  parser,
		metrics: metrics,
	}
}

// ExtractToolCalls wraps the parser with metrics collection
func (o *ObservableParser) ExtractToolCalls(response string) ([]ToolCall, error) {
	ctx := context.Background()
	return o.ExtractWithContext(ctx, response)
}

// ExtractWithContext wraps the parser with metrics collection and context
func (o *ObservableParser) ExtractWithContext(ctx context.Context, response string) ([]ToolCall, error) {
	start := time.Now()
	
	// Record input size
	o.metrics.inputSize.WithLabelValues("unknown").Observe(float64(len(response)))
	
	// Detect format first
	format, confidence, err := o.parser.DetectFormat(response)
	if err != nil {
		o.metrics.parseFailures.WithLabelValues(string(format), "detection_failed").Inc()
		return nil, err
	}
	
	// Record format detection metrics
	o.metrics.formatDistribution.WithLabelValues(string(format)).Inc()
	o.metrics.detectionConfidence.WithLabelValues("combined", string(format)).Observe(confidence)
	
	// Record parse attempt
	o.metrics.parseAttempts.WithLabelValues(string(format)).Inc()
	
	// Parse tool calls
	calls, err := o.parser.ExtractWithContext(ctx, response)
	
	// Record duration
	duration := time.Since(start)
	o.metrics.parseDuration.WithLabelValues(string(format)).Observe(duration.Seconds())
	
	if err != nil {
		o.metrics.parseFailures.WithLabelValues(string(format), "parse_error").Inc()
		o.metrics.parseErrors.WithLabelValues(string(format), getErrorType(err)).Inc()
		return nil, err
	}
	
	// Record success metrics
	o.metrics.parseSuccesses.WithLabelValues(string(format)).Inc()
	o.metrics.toolCallsPerParse.WithLabelValues(string(format)).Observe(float64(len(calls)))
	
	// Record individual tool calls
	for _, call := range calls {
		o.metrics.toolCallsExtracted.WithLabelValues(string(format), call.Function.Name).Inc()
	}
	
	return calls, nil
}

// DetectFormat wraps format detection with metrics
func (o *ObservableParser) DetectFormat(response string) (ProviderFormat, float64, error) {
	start := time.Now()
	
	format, confidence, err := o.parser.DetectFormat(response)
	
	// Record detection duration
	duration := time.Since(start)
	if err == nil {
		o.metrics.detectionDuration.WithLabelValues("combined", string(format)).Observe(duration.Seconds())
		o.metrics.detectionSuccesses.WithLabelValues("combined", string(format)).Inc()
	} else {
		o.metrics.detectionFailures.WithLabelValues("combined", "error").Inc()
	}
	
	return format, confidence, err
}

// RecordDetection records detection metrics for individual detectors
func (o *ObservableParser) RecordDetection(detector string, format ProviderFormat, confidence float64, duration time.Duration, err error) {
	o.metrics.detectionAttempts.WithLabelValues(detector).Inc()
	
	if err == nil {
		o.metrics.detectionSuccesses.WithLabelValues(detector, string(format)).Inc()
		o.metrics.detectionDuration.WithLabelValues(detector, string(format)).Observe(duration.Seconds())
		o.metrics.detectionConfidence.WithLabelValues(detector, string(format)).Observe(confidence)
	} else {
		o.metrics.detectionFailures.WithLabelValues(detector, getErrorType(err)).Inc()
	}
}

// RecordValidation records validation metrics
func (o *ObservableParser) RecordValidation(format ProviderFormat, result ValidationResult) {
	o.metrics.validationAttempts.WithLabelValues(string(format)).Inc()
	
	if !result.Valid {
		for _, err := range result.Errors {
			o.metrics.validationFailures.WithLabelValues(string(format), err.Code).Inc()
		}
	}
	
	for range result.Warnings {
		o.metrics.validationWarnings.WithLabelValues(string(format)).Inc()
	}
}

// RecordExtractionLocation records where tool calls were extracted from
func (o *ObservableParser) RecordExtractionLocation(location string) {
	o.metrics.extractionLocation.WithLabelValues(location).Inc()
}

// getErrorType categorizes errors for metrics
func getErrorType(err error) string {
	if err == nil {
		return "none"
	}
	
	// Check for common error types
	switch {
	case isTimeoutError(err):
		return "timeout"
	case isValidationError(err):
		return "validation"
	case isParseError(err):
		return "parse"
	case isFormatError(err):
		return "format"
	default:
		return "unknown"
	}
}

// Error type detection helpers
func isTimeoutError(err error) bool {
	// Check for context deadline exceeded or timeout errors
	return false // Implement based on actual error types
}

func isValidationError(err error) bool {
	// Check for validation errors
	return false // Implement based on actual error types
}

func isParseError(err error) bool {
	// Check for parsing errors
	return false // Implement based on actual error types
}

func isFormatError(err error) bool {
	// Check for format errors
	return false // Implement based on actual error types
}