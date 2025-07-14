// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package debug

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// Inspector provides real-time event debugging capabilities
type Inspector struct {
	mu       sync.RWMutex
	config   *InspectorConfig
	filters  []EventFilter
	hooks    []EventHook
	buffers  map[string]*EventBuffer
	sessions map[string]*DebugSession
	metrics  *InspectorMetrics
	running  bool
}

// InspectorConfig configures the inspector
type InspectorConfig struct {
	BufferSize      int
	MaxSessions     int
	RetentionPeriod time.Duration
	EnableMetrics   bool
	EnableProfiling bool
	SamplingRate    float64 // 0.0 to 1.0
}

// EventFilter filters events for inspection
type EventFilter interface {
	Match(event events.CoreEvent) bool
	Name() string
}

// EventHook is called when events match filters
type EventHook interface {
	OnEvent(ctx context.Context, event events.CoreEvent, session *DebugSession) error
}

// EventBuffer stores events for inspection
type EventBuffer struct {
	events []EventSnapshot
	size   int
	index  int
	mu     sync.RWMutex
}

// EventSnapshot captures event state for debugging
type EventSnapshot struct {
	Event     events.CoreEvent
	Timestamp time.Time
	Source    string
	Metadata  map[string]interface{}
	Context   map[string]interface{}
}

// DebugSession tracks a debugging session
type DebugSession struct {
	ID         string
	Name       string
	Filters    []EventFilter
	Hooks      []EventHook
	StartTime  time.Time
	EndTime    *time.Time
	EventCount int64
	active     bool
	mu         sync.RWMutex
}

// InspectorMetrics tracks inspector performance
type InspectorMetrics struct {
	EventsInspected int64
	EventsFiltered  int64
	SessionsActive  int
	SessionsTotal   int64
	HooksExecuted   int64
	HookErrors      int64
	mu              sync.RWMutex
}

// DefaultInspectorConfig returns default configuration
func DefaultInspectorConfig() *InspectorConfig {
	return &InspectorConfig{
		BufferSize:      10000,
		MaxSessions:     100,
		RetentionPeriod: 24 * time.Hour,
		EnableMetrics:   true,
		EnableProfiling: false,
		SamplingRate:    1.0,
	}
}

// NewInspector creates a new event inspector
func NewInspector(config *InspectorConfig) *Inspector {
	if config == nil {
		config = DefaultInspectorConfig()
	}

	return &Inspector{
		config:   config,
		buffers:  make(map[string]*EventBuffer),
		sessions: make(map[string]*DebugSession),
		metrics:  &InspectorMetrics{},
	}
}

// Start starts the inspector
func (i *Inspector) Start(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.running {
		return gerror.New(gerror.ErrCodeAlreadyExists, "inspector is already running", nil)
	}

	i.running = true

	// Start cleanup goroutine
	go i.cleanup(ctx)

	return nil
}

// Stop stops the inspector
func (i *Inspector) Stop() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if !i.running {
		return nil
	}

	i.running = false

	// Close all sessions
	for _, session := range i.sessions {
		session.Close()
	}

	return nil
}

// InspectEvent inspects an event
func (i *Inspector) InspectEvent(ctx context.Context, event events.CoreEvent) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	i.mu.RLock()
	if !i.running {
		i.mu.RUnlock()
		return nil
	}

	// Apply sampling
	if i.config.SamplingRate < 1.0 && !i.shouldSample() {
		i.mu.RUnlock()
		return nil
	}

	sessions := make([]*DebugSession, 0, len(i.sessions))
	for _, session := range i.sessions {
		if session.IsActive() {
			sessions = append(sessions, session)
		}
	}
	i.mu.RUnlock()

	// Update metrics
	if i.config.EnableMetrics {
		i.metrics.mu.Lock()
		i.metrics.EventsInspected++
		i.metrics.mu.Unlock()
	}

	// Create snapshot
	snapshot := &EventSnapshot{
		Event:     event,
		Timestamp: time.Now(),
		Source:    event.GetSource(),
		Metadata:  copyMetadata(event.GetMetadata()),
		Context:   extractContext(ctx),
	}

	// Process sessions
	for _, session := range sessions {
		if err := i.processSession(ctx, session, snapshot); err != nil {
			// Log error but continue processing other sessions
			continue
		}
	}

	return nil
}

// CreateSession creates a new debug session
func (i *Inspector) CreateSession(name string, filters []EventFilter, hooks []EventHook) (*DebugSession, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if len(i.sessions) >= i.config.MaxSessions {
		return nil, gerror.New(gerror.ErrCodeResourceExhausted, "max sessions exceeded", nil).
			WithDetails("max_sessions", i.config.MaxSessions)
	}

	sessionID := generateSessionID()
	session := &DebugSession{
		ID:        sessionID,
		Name:      name,
		Filters:   filters,
		Hooks:     hooks,
		StartTime: time.Now(),
		active:    true,
	}

	i.sessions[sessionID] = session

	// Update metrics
	if i.config.EnableMetrics {
		i.metrics.mu.Lock()
		i.metrics.SessionsActive++
		i.metrics.SessionsTotal++
		i.metrics.mu.Unlock()
	}

	return session, nil
}

// GetSession returns a debug session
func (i *Inspector) GetSession(sessionID string) (*DebugSession, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	session, exists := i.sessions[sessionID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "session not found", nil).
			WithDetails("session_id", sessionID)
	}

	return session, nil
}

// CloseSession closes a debug session
func (i *Inspector) CloseSession(sessionID string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	session, exists := i.sessions[sessionID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "session not found", nil).
			WithDetails("session_id", sessionID)
	}

	session.Close()
	delete(i.sessions, sessionID)

	// Update metrics
	if i.config.EnableMetrics {
		i.metrics.mu.Lock()
		i.metrics.SessionsActive--
		i.metrics.mu.Unlock()
	}

	return nil
}

// GetActiveSessions returns all active sessions
func (i *Inspector) GetActiveSessions() []*DebugSession {
	i.mu.RLock()
	defer i.mu.RUnlock()

	sessions := make([]*DebugSession, 0, len(i.sessions))
	for _, session := range i.sessions {
		if session.IsActive() {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetMetrics returns inspector metrics
func (i *Inspector) GetMetrics() *InspectorMetrics {
	i.metrics.mu.RLock()
	defer i.metrics.mu.RUnlock()

	return &InspectorMetrics{
		EventsInspected: i.metrics.EventsInspected,
		EventsFiltered:  i.metrics.EventsFiltered,
		SessionsActive:  i.metrics.SessionsActive,
		SessionsTotal:   i.metrics.SessionsTotal,
		HooksExecuted:   i.metrics.HooksExecuted,
		HookErrors:      i.metrics.HookErrors,
	}
}

// processSession processes an event for a specific session
func (i *Inspector) processSession(ctx context.Context, session *DebugSession, snapshot *EventSnapshot) error {
	// Check filters
	matched := false
	if len(session.Filters) == 0 {
		matched = true
	} else {
		for _, filter := range session.Filters {
			if filter.Match(snapshot.Event) {
				matched = true
				break
			}
		}
	}

	if !matched {
		return nil
	}

	// Update session count
	session.mu.Lock()
	session.EventCount++
	session.mu.Unlock()

	// Execute hooks
	for _, hook := range session.Hooks {
		if err := hook.OnEvent(ctx, snapshot.Event, session); err != nil {
			if i.config.EnableMetrics {
				i.metrics.mu.Lock()
				i.metrics.HookErrors++
				i.metrics.mu.Unlock()
			}
			continue
		}

		if i.config.EnableMetrics {
			i.metrics.mu.Lock()
			i.metrics.HooksExecuted++
			i.metrics.mu.Unlock()
		}
	}

	return nil
}

// shouldSample determines if an event should be sampled
func (i *Inspector) shouldSample() bool {
	// Simple sampling implementation
	return true // TODO: Implement proper sampling
}

// cleanup periodically cleans up old data
func (i *Inspector) cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.performCleanup()
		}
	}
}

// performCleanup removes old data
func (i *Inspector) performCleanup() {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-i.config.RetentionPeriod)

	// Clean up old sessions
	for id, session := range i.sessions {
		if !session.IsActive() && session.StartTime.Before(cutoff) {
			delete(i.sessions, id)
		}
	}
}

// DebugSession methods

// IsActive returns true if the session is active
func (s *DebugSession) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// Close closes the debug session
func (s *DebugSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		now := time.Now()
		s.EndTime = &now
		s.active = false
	}
}

// GetInfo returns session information
func (s *DebugSession) GetInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info := map[string]interface{}{
		"id":          s.ID,
		"name":        s.Name,
		"start_time":  s.StartTime,
		"event_count": s.EventCount,
		"active":      s.active,
		"filters":     len(s.Filters),
		"hooks":       len(s.Hooks),
	}

	if s.EndTime != nil {
		info["end_time"] = *s.EndTime
		info["duration"] = s.EndTime.Sub(s.StartTime)
	}

	return info
}

// EventBuffer methods

// NewEventBuffer creates a new event buffer
func NewEventBuffer(size int) *EventBuffer {
	return &EventBuffer{
		events: make([]EventSnapshot, size),
		size:   size,
	}
}

// Add adds an event to the buffer
func (b *EventBuffer) Add(snapshot EventSnapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.events[b.index] = snapshot
	b.index = (b.index + 1) % b.size
}

// GetEvents returns events from the buffer
func (b *EventBuffer) GetEvents(limit int) []EventSnapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 || limit > b.size {
		limit = b.size
	}

	events := make([]EventSnapshot, 0, limit)
	for i := 0; i < limit && i < b.size; i++ {
		idx := (b.index - i - 1 + b.size) % b.size
		if !b.events[idx].Timestamp.IsZero() {
			events = append(events, b.events[idx])
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.After(events[j].Timestamp)
	})

	return events
}

// Helper functions

func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

func copyMetadata(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return nil
	}

	copy := make(map[string]interface{}, len(metadata))
	for k, v := range metadata {
		copy[k] = v
	}
	return copy
}

func extractContext(ctx context.Context) map[string]interface{} {
	// Extract useful context information
	info := make(map[string]interface{})

	if deadline, ok := ctx.Deadline(); ok {
		info["deadline"] = deadline
	}

	// TODO: Extract more context information as needed

	return info
}
