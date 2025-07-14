// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// EventTracer traces event flow through the system
type EventTracer struct {
	mu     sync.RWMutex
	traces map[string]*EventTrace
	config *TracerConfig
}

// TracerConfig configures the event tracer
type TracerConfig struct {
	MaxTraces     int
	TraceTTL      time.Duration
	EnableMetrics bool
	BufferSize    int
}

// EventTrace tracks an event's journey
type EventTrace struct {
	EventID   string
	EventType string
	StartTime time.Time
	Steps     []TraceStep
	Metadata  map[string]interface{}
	mu        sync.RWMutex
}

// TraceStep represents a step in event processing
type TraceStep struct {
	Timestamp time.Time
	Component string
	Action    string
	Duration  time.Duration
	Success   bool
	Error     string
	Metadata  map[string]interface{}
}

// EventAnalyzer analyzes event patterns and performance
type EventAnalyzer struct {
	mu      sync.RWMutex
	samples map[string]*EventSample
	stats   *AnalysisStats
	config  *AnalyzerConfig
}

// AnalyzerConfig configures the event analyzer
type AnalyzerConfig struct {
	SampleWindow    time.Duration
	MaxSamples      int
	EnableProfiling bool
	Percentiles     []float64
}

// EventSample contains event analysis data
type EventSample struct {
	EventType string
	Count     int64
	TotalTime time.Duration
	MinTime   time.Duration
	MaxTime   time.Duration
	Errors    int64
	LastSeen  time.Time
	Durations []time.Duration
}

// AnalysisStats contains analysis statistics
type AnalysisStats struct {
	TotalEvents    int64
	UniqueTypes    int
	AverageLatency time.Duration
	ErrorRate      float64
	ThroughputRPS  float64
	LastUpdate     time.Time
}

// EventReplayer replays events for testing and debugging
type EventReplayer struct {
	mu       sync.RWMutex
	sessions map[string]*ReplaySession
	config   *ReplayerConfig
}

// ReplayerConfig configures the event replayer
type ReplayerConfig struct {
	MaxSessions   int
	DefaultSpeed  float64
	BufferSize    int
	EnableMetrics bool
}

// ReplaySession manages event replay
type ReplaySession struct {
	ID        string
	Events    []events.CoreEvent
	Position  int
	Speed     float64
	Running   bool
	StartTime time.Time
	EndTime   *time.Time
	Handler   events.EventHandler
	mu        sync.RWMutex
}

// DefaultTracerConfig returns default tracer configuration
func DefaultTracerConfig() *TracerConfig {
	return &TracerConfig{
		MaxTraces:     10000,
		TraceTTL:      24 * time.Hour,
		EnableMetrics: true,
		BufferSize:    1000,
	}
}

// DefaultAnalyzerConfig returns default analyzer configuration
func DefaultAnalyzerConfig() *AnalyzerConfig {
	return &AnalyzerConfig{
		SampleWindow:    time.Hour,
		MaxSamples:      100000,
		EnableProfiling: true,
		Percentiles:     []float64{50, 90, 95, 99},
	}
}

// DefaultReplayerConfig returns default replayer configuration
func DefaultReplayerConfig() *ReplayerConfig {
	return &ReplayerConfig{
		MaxSessions:   10,
		DefaultSpeed:  1.0,
		BufferSize:    10000,
		EnableMetrics: true,
	}
}

// NewEventTracer creates a new event tracer
func NewEventTracer(config *TracerConfig) *EventTracer {
	if config == nil {
		config = DefaultTracerConfig()
	}

	return &EventTracer{
		traces: make(map[string]*EventTrace),
		config: config,
	}
}

// StartTrace starts tracing an event
func (t *EventTracer) StartTrace(event events.CoreEvent) *EventTrace {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if we need to cleanup old traces
	if len(t.traces) >= t.config.MaxTraces {
		t.cleanup()
	}

	trace := &EventTrace{
		EventID:   event.GetID(),
		EventType: event.GetType(),
		StartTime: time.Now(),
		Steps:     make([]TraceStep, 0),
		Metadata:  copyMetadata(event.GetMetadata()),
	}

	t.traces[event.GetID()] = trace
	return trace
}

// GetTrace returns a trace by event ID
func (t *EventTracer) GetTrace(eventID string) (*EventTrace, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	trace, exists := t.traces[eventID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "trace not found", nil).
			WithDetails("event_id", eventID)
	}

	return trace, nil
}

// cleanup removes old traces
func (t *EventTracer) cleanup() {
	cutoff := time.Now().Add(-t.config.TraceTTL)

	for id, trace := range t.traces {
		if trace.StartTime.Before(cutoff) {
			delete(t.traces, id)
		}
	}
}

// AddStep adds a step to the trace
func (tr *EventTrace) AddStep(component, action string, duration time.Duration, success bool, err error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	step := TraceStep{
		Timestamp: time.Now(),
		Component: component,
		Action:    action,
		Duration:  duration,
		Success:   success,
		Metadata:  make(map[string]interface{}),
	}

	if err != nil {
		step.Error = err.Error()
	}

	tr.Steps = append(tr.Steps, step)
}

// GetDuration returns the total trace duration
func (tr *EventTrace) GetDuration() time.Duration {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	if len(tr.Steps) == 0 {
		return 0
	}

	lastStep := tr.Steps[len(tr.Steps)-1]
	return lastStep.Timestamp.Sub(tr.StartTime)
}

// ToJSON converts the trace to JSON
func (tr *EventTrace) ToJSON() ([]byte, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	return json.MarshalIndent(tr, "", "  ")
}

// NewEventAnalyzer creates a new event analyzer
func NewEventAnalyzer(config *AnalyzerConfig) *EventAnalyzer {
	if config == nil {
		config = DefaultAnalyzerConfig()
	}

	return &EventAnalyzer{
		samples: make(map[string]*EventSample),
		stats:   &AnalysisStats{},
		config:  config,
	}
}

// RecordEvent records an event for analysis
func (a *EventAnalyzer) RecordEvent(event events.CoreEvent, duration time.Duration, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	eventType := event.GetType()
	sample, exists := a.samples[eventType]
	if !exists {
		sample = &EventSample{
			EventType: eventType,
			MinTime:   duration,
			MaxTime:   duration,
			Durations: make([]time.Duration, 0),
		}
		a.samples[eventType] = sample
	}

	// Update sample
	sample.Count++
	sample.TotalTime += duration
	sample.LastSeen = time.Now()

	if duration < sample.MinTime {
		sample.MinTime = duration
	}
	if duration > sample.MaxTime {
		sample.MaxTime = duration
	}

	if err != nil {
		sample.Errors++
	}

	// Store duration for percentile calculation
	if len(sample.Durations) < a.config.MaxSamples {
		sample.Durations = append(sample.Durations, duration)
	}

	// Update global stats
	a.stats.TotalEvents++
	a.stats.UniqueTypes = len(a.samples)
	a.stats.LastUpdate = time.Now()
}

// GetStats returns analysis statistics
func (a *EventAnalyzer) GetStats() *AnalysisStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := *a.stats
	return &stats
}

// GetSample returns analysis data for an event type
func (a *EventAnalyzer) GetSample(eventType string) (*EventSample, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sample, exists := a.samples[eventType]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "sample not found", nil).
			WithDetails("event_type", eventType)
	}

	// Return a copy
	copy := *sample
	return &copy, nil
}

// GetPercentiles calculates percentiles for an event type
func (a *EventAnalyzer) GetPercentiles(eventType string) (map[float64]time.Duration, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	sample, exists := a.samples[eventType]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "sample not found", nil).
			WithDetails("event_type", eventType)
	}

	if len(sample.Durations) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no duration data available", nil)
	}

	// Sort durations
	durations := make([]time.Duration, len(sample.Durations))
	copy(durations, sample.Durations)
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	// Calculate percentiles
	percentiles := make(map[float64]time.Duration)
	for _, p := range a.config.Percentiles {
		index := int(float64(len(durations)-1) * p / 100.0)
		percentiles[p] = durations[index]
	}

	return percentiles, nil
}

// NewEventReplayer creates a new event replayer
func NewEventReplayer(config *ReplayerConfig) *EventReplayer {
	if config == nil {
		config = DefaultReplayerConfig()
	}

	return &EventReplayer{
		sessions: make(map[string]*ReplaySession),
		config:   config,
	}
}

// CreateSession creates a new replay session
func (r *EventReplayer) CreateSession(events []events.CoreEvent, handler events.EventHandler) (*ReplaySession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.sessions) >= r.config.MaxSessions {
		return nil, gerror.New(gerror.ErrCodeResourceExhausted, "max sessions exceeded", nil).
			WithDetails("max_sessions", r.config.MaxSessions)
	}

	sessionID := fmt.Sprintf("replay_%d", time.Now().UnixNano())
	session := &ReplaySession{
		ID:      sessionID,
		Events:  events,
		Speed:   r.config.DefaultSpeed,
		Handler: handler,
	}

	r.sessions[sessionID] = session
	return session, nil
}

// StartReplay starts replaying events
func (r *EventReplayer) StartReplay(sessionID string) error {
	r.mu.RLock()
	session, exists := r.sessions[sessionID]
	r.mu.RUnlock()

	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "session not found", nil).
			WithDetails("session_id", sessionID)
	}

	return session.Start()
}

// StopReplay stops replaying events
func (r *EventReplayer) StopReplay(sessionID string) error {
	r.mu.RLock()
	session, exists := r.sessions[sessionID]
	r.mu.RUnlock()

	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "session not found", nil).
			WithDetails("session_id", sessionID)
	}

	return session.Stop()
}

// ReplaySession methods

// Start starts the replay session
func (s *ReplaySession) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Running {
		return gerror.New(gerror.ErrCodeAlreadyExists, "session is already running", nil)
	}

	s.Running = true
	s.StartTime = time.Now()

	// Start replay goroutine
	go s.replay()

	return nil
}

// Stop stops the replay session
func (s *ReplaySession) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Running {
		return nil
	}

	s.Running = false
	now := time.Now()
	s.EndTime = &now

	return nil
}

// replay performs the actual event replay
func (s *ReplaySession) replay() {
	ctx := context.Background()

	for s.Position < len(s.Events) {
		s.mu.RLock()
		if !s.Running {
			s.mu.RUnlock()
			break
		}

		event := s.Events[s.Position]
		speed := s.Speed
		handler := s.Handler
		s.mu.RUnlock()

		// Execute handler
		if handler != nil {
			if err := handler(ctx, event); err != nil {
				// Log error but continue
				continue
			}
		}

		// Update position
		s.mu.Lock()
		s.Position++
		s.mu.Unlock()

		// Apply speed adjustment
		if speed > 0 && speed != 1.0 {
			delay := time.Duration(float64(100*time.Millisecond) / speed)
			time.Sleep(delay)
		} else if speed == 0 {
			// Pause
			for {
				s.mu.RLock()
				if !s.Running || s.Speed > 0 {
					s.mu.RUnlock()
					break
				}
				s.mu.RUnlock()
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	// Mark as complete
	s.mu.Lock()
	s.Running = false
	if s.EndTime == nil {
		now := time.Now()
		s.EndTime = &now
	}
	s.mu.Unlock()
}

// GetStatus returns session status
func (s *ReplaySession) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := map[string]interface{}{
		"id":       s.ID,
		"running":  s.Running,
		"position": s.Position,
		"total":    len(s.Events),
		"speed":    s.Speed,
	}

	if !s.StartTime.IsZero() {
		status["start_time"] = s.StartTime
	}

	if s.EndTime != nil {
		status["end_time"] = *s.EndTime
		status["duration"] = s.EndTime.Sub(s.StartTime)
	}

	if len(s.Events) > 0 {
		status["progress"] = float64(s.Position) / float64(len(s.Events))
	}

	return status
}

// EventDumper dumps events to various formats
type EventDumper struct {
	format string
}

// NewEventDumper creates a new event dumper
func NewEventDumper(format string) *EventDumper {
	return &EventDumper{format: format}
}

// Dump dumps events to a writer
func (d *EventDumper) Dump(writer io.Writer, events []events.CoreEvent) error {
	switch strings.ToLower(d.format) {
	case "json":
		return d.dumpJSON(writer, events)
	case "csv":
		return d.dumpCSV(writer, events)
	case "text":
		return d.dumpText(writer, events)
	default:
		return gerror.New(gerror.ErrCodeValidation, "unsupported format", nil).
			WithDetails("format", d.format)
	}
}

// dumpJSON dumps events as JSON
func (d *EventDumper) dumpJSON(writer io.Writer, events []events.CoreEvent) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	for _, event := range events {
		eventData := map[string]interface{}{
			"id":        event.GetID(),
			"type":      event.GetType(),
			"source":    event.GetSource(),
			"timestamp": event.GetTimestamp(),
			"data":      event.GetData(),
			"metadata":  event.GetMetadata(),
		}

		if err := encoder.Encode(eventData); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to encode event")
		}
	}

	return nil
}

// dumpCSV dumps events as CSV
func (d *EventDumper) dumpCSV(writer io.Writer, events []events.CoreEvent) error {
	// Write header
	if _, err := writer.Write([]byte("id,type,source,timestamp,data\n")); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write CSV header")
	}

	// Write events
	for _, event := range events {
		dataJSON, _ := json.Marshal(event.GetData())
		line := fmt.Sprintf("%s,%s,%s,%s,%s\n",
			event.GetID(),
			event.GetType(),
			event.GetSource(),
			event.GetTimestamp().Format(time.RFC3339),
			string(dataJSON),
		)

		if _, err := writer.Write([]byte(line)); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write CSV line")
		}
	}

	return nil
}

// dumpText dumps events as human-readable text
func (d *EventDumper) dumpText(writer io.Writer, events []events.CoreEvent) error {
	for i, event := range events {
		if i > 0 {
			if _, err := writer.Write([]byte("\n---\n\n")); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write separator")
			}
		}

		text := fmt.Sprintf("Event #%d\n", i+1)
		text += fmt.Sprintf("  ID: %s\n", event.GetID())
		text += fmt.Sprintf("  Type: %s\n", event.GetType())
		text += fmt.Sprintf("  Source: %s\n", event.GetSource())
		text += fmt.Sprintf("  Timestamp: %s\n", event.GetTimestamp().Format(time.RFC3339))

		if data := event.GetData(); len(data) > 0 {
			text += "  Data:\n"
			for k, v := range data {
				text += fmt.Sprintf("    %s: %v\n", k, v)
			}
		}

		if metadata := event.GetMetadata(); len(metadata) > 0 {
			text += "  Metadata:\n"
			for k, v := range metadata {
				text += fmt.Sprintf("    %s: %v\n", k, v)
			}
		}

		if _, err := writer.Write([]byte(text)); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write event text")
		}
	}

	return nil
}
