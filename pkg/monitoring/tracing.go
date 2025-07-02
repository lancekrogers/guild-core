// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TracingIntegration provides distributed tracing capabilities
type TracingIntegration struct {
	tracer   Tracer
	monitor  *PerformanceMonitor
	config   *TracingConfig
	spans    map[string]*Span
	mu       sync.RWMutex
}

// TracingConfig configures tracing behavior
type TracingConfig struct {
	Enabled        bool    `json:"enabled"`
	SamplingRate   float64 `json:"sampling_rate"`
	MaxSpans       int     `json:"max_spans"`
	FlushInterval  time.Duration `json:"flush_interval"`
	ExporterURL    string  `json:"exporter_url"`
}

// DefaultTracingConfig returns default tracing configuration
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:       true,
		SamplingRate:  1.0, // Sample 100% for demo
		MaxSpans:      10000,
		FlushInterval: time.Second * 30,
	}
}

// Tracer interface for distributed tracing
type Tracer interface {
	StartSpan(ctx context.Context, operationName string) (*Span, context.Context)
	FinishSpan(span *Span)
}

// Span represents a tracing span
type Span struct {
	TraceID       string                 `json:"trace_id"`
	SpanID        string                 `json:"span_id"`
	ParentSpanID  string                 `json:"parent_span_id,omitempty"`
	OperationName string                 `json:"operation_name"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty"`
	Duration      time.Duration          `json:"duration"`
	Tags          map[string]interface{} `json:"tags"`
	Logs          []LogEntry             `json:"logs"`
	Error         bool                   `json:"error"`
}

// LogEntry represents a log entry within a span
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Fields    map[string]interface{} `json:"fields"`
}

// NewTracingIntegration creates a new tracing integration
func NewTracingIntegration() *TracingIntegration {
	return &TracingIntegration{
		tracer: NewSimpleTracer(),
		config: DefaultTracingConfig(),
		spans:  make(map[string]*Span),
	}
}

// TraceOperation traces an operation with automatic instrumentation
func (ti *TracingIntegration) TraceOperation(ctx context.Context, operation string, fn func(context.Context) error) error {
	if !ti.config.Enabled {
		return fn(ctx)
	}

	span, ctx := ti.tracer.StartSpan(ctx, operation)
	defer ti.tracer.FinishSpan(span)

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	// Record metrics
	if ti.monitor != nil {
		ti.monitor.RecordOperation(operation, duration, err)
	}

	// Add span tags
	span.SetTag("duration_ms", duration.Milliseconds())
	if err != nil {
		span.SetTag("error", true)
		span.SetTag("error_message", err.Error())
		span.Error = true
	}

	return err
}

// RecordOperation records operation metrics for tracing
func (ti *TracingIntegration) RecordOperation(operation string, duration time.Duration, err error) {
	if !ti.config.Enabled {
		return
	}

	// Create a synthetic span for the operation
	span := &Span{
		TraceID:       generateTraceID(),
		SpanID:        generateSpanID(),
		OperationName: operation,
		StartTime:     time.Now().Add(-duration),
		Duration:      duration,
		Tags:          make(map[string]interface{}),
		Logs:          make([]LogEntry, 0),
		Error:         err != nil,
	}

	endTime := span.StartTime.Add(duration)
	span.EndTime = &endTime

	span.Tags["duration_ms"] = duration.Milliseconds()
	if err != nil {
		span.Tags["error"] = true
		span.Tags["error_message"] = err.Error()
	}

	ti.recordSpan(span)
}

// recordSpan records a span
func (ti *TracingIntegration) recordSpan(span *Span) {
	ti.mu.Lock()
	defer ti.mu.Unlock()

	ti.spans[span.SpanID] = span

	// Clean up old spans if needed
	if len(ti.spans) > ti.config.MaxSpans {
		ti.cleanupOldSpans()
	}
}

// cleanupOldSpans removes old spans to prevent memory leaks
func (ti *TracingIntegration) cleanupOldSpans() {
	cutoff := time.Now().Add(-time.Hour) // Keep spans for 1 hour

	for id, span := range ti.spans {
		if span.StartTime.Before(cutoff) {
			delete(ti.spans, id)
		}
	}
}

// GetSpans returns spans matching criteria
func (ti *TracingIntegration) GetSpans(filter *SpanFilter) []*Span {
	ti.mu.RLock()
	defer ti.mu.RUnlock()

	var spans []*Span
	for _, span := range ti.spans {
		if filter == nil || filter.Matches(span) {
			spans = append(spans, span)
		}
	}

	return spans
}

// SpanFilter filters spans based on criteria
type SpanFilter struct {
	OperationName string        `json:"operation_name,omitempty"`
	MinDuration   time.Duration `json:"min_duration,omitempty"`
	MaxDuration   time.Duration `json:"max_duration,omitempty"`
	ErrorsOnly    bool          `json:"errors_only"`
	Since         *time.Time    `json:"since,omitempty"`
}

// Matches checks if a span matches the filter criteria
func (sf *SpanFilter) Matches(span *Span) bool {
	if sf.OperationName != "" && span.OperationName != sf.OperationName {
		return false
	}

	if sf.MinDuration > 0 && span.Duration < sf.MinDuration {
		return false
	}

	if sf.MaxDuration > 0 && span.Duration > sf.MaxDuration {
		return false
	}

	if sf.ErrorsOnly && !span.Error {
		return false
	}

	if sf.Since != nil && span.StartTime.Before(*sf.Since) {
		return false
	}

	return true
}

// SetTag sets a tag on the span
func (s *Span) SetTag(key string, value interface{}) {
	if s.Tags == nil {
		s.Tags = make(map[string]interface{})
	}
	s.Tags[key] = value
}

// LogFields logs structured data to the span
func (s *Span) LogFields(fields map[string]interface{}) {
	logEntry := LogEntry{
		Timestamp: time.Now(),
		Fields:    fields,
	}
	s.Logs = append(s.Logs, logEntry)
}

// SimpleTracer provides a basic tracing implementation
type SimpleTracer struct {
	mu sync.RWMutex
}

// NewSimpleTracer creates a new simple tracer
func NewSimpleTracer() *SimpleTracer {
	return &SimpleTracer{}
}

// StartSpan starts a new span
func (st *SimpleTracer) StartSpan(ctx context.Context, operationName string) (*Span, context.Context) {
	span := &Span{
		TraceID:       generateTraceID(),
		SpanID:        generateSpanID(),
		OperationName: operationName,
		StartTime:     time.Now(),
		Tags:          make(map[string]interface{}),
		Logs:          make([]LogEntry, 0),
	}

	// Check for parent span in context
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		span.TraceID = parentSpan.TraceID
		span.ParentSpanID = parentSpan.SpanID
	}

	// Add span to context
	ctx = ContextWithSpan(ctx, span)

	return span, ctx
}

// FinishSpan finishes a span
func (st *SimpleTracer) FinishSpan(span *Span) {
	if span.EndTime == nil {
		endTime := time.Now()
		span.EndTime = &endTime
		span.Duration = endTime.Sub(span.StartTime)
	}
}

// Context utilities for span management
type spanKey struct{}

// ContextWithSpan returns a context with the span
func ContextWithSpan(ctx context.Context, span *Span) context.Context {
	return context.WithValue(ctx, spanKey{}, span)
}

// SpanFromContext extracts a span from context
func SpanFromContext(ctx context.Context) *Span {
	if span, ok := ctx.Value(spanKey{}).(*Span); ok {
		return span
	}
	return nil
}

// MetricsExporter exports metrics to external systems
type MetricsExporter interface {
	Start(ctx context.Context) error
	Export(metrics *Metrics) error
	Stop() error
}

// PrometheusExporter exports metrics to Prometheus
type PrometheusExporter struct {
	endpoint string
	interval time.Duration
	running  bool
	mu       sync.RWMutex
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(endpoint string, interval time.Duration) *PrometheusExporter {
	return &PrometheusExporter{
		endpoint: endpoint,
		interval: interval,
	}
}

// Start starts the Prometheus exporter
func (pe *PrometheusExporter) Start(ctx context.Context) error {
	pe.mu.Lock()
	if pe.running {
		pe.mu.Unlock()
		return fmt.Errorf("exporter already running")
	}
	pe.running = true
	pe.mu.Unlock()

	ticker := time.NewTicker(pe.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pe.mu.Lock()
			pe.running = false
			pe.mu.Unlock()
			return ctx.Err()
		case <-ticker.C:
			// Export metrics (placeholder implementation)
			// In real implementation, this would format and send metrics to Prometheus
		}
	}
}

// Export exports metrics to Prometheus
func (pe *PrometheusExporter) Export(metrics *Metrics) error {
	// Placeholder implementation
	// In real implementation, this would format metrics for Prometheus
	return nil
}

// Stop stops the Prometheus exporter
func (pe *PrometheusExporter) Stop() error {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.running = false
	return nil
}

// JaegerExporter exports traces to Jaeger
type JaegerExporter struct {
	endpoint string
	running  bool
	mu       sync.RWMutex
}

// NewJaegerExporter creates a new Jaeger exporter
func NewJaegerExporter(endpoint string) *JaegerExporter {
	return &JaegerExporter{
		endpoint: endpoint,
	}
}

// Start starts the Jaeger exporter
func (je *JaegerExporter) Start(ctx context.Context) error {
	je.mu.Lock()
	if je.running {
		je.mu.Unlock()
		return fmt.Errorf("exporter already running")
	}
	je.running = true
	je.mu.Unlock()

	// Placeholder implementation
	// In real implementation, this would start a background process to export traces
	return nil
}

// Export exports traces to Jaeger
func (je *JaegerExporter) Export(metrics *Metrics) error {
	// Placeholder implementation
	// In real implementation, this would send traces to Jaeger
	return nil
}

// Stop stops the Jaeger exporter
func (je *JaegerExporter) Stop() error {
	je.mu.Lock()
	defer je.mu.Unlock()
	je.running = false
	return nil
}

// Helper functions for ID generation
func generateTraceID() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}

func generateSpanID() string {
	return fmt.Sprintf("span-%d", time.Now().UnixNano())
}