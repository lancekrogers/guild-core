// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package debug

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild/pkg/events"
)

// Mock event filter
type mockEventFilter struct {
	matchFunc func(event events.CoreEvent) bool
	name      string
}

func (f *mockEventFilter) Match(event events.CoreEvent) bool {
	if f.matchFunc != nil {
		return f.matchFunc(event)
	}
	return true
}

func (f *mockEventFilter) Name() string {
	return f.name
}

// Mock event hook
type mockEventHook struct {
	onEventFunc func(ctx context.Context, event events.CoreEvent, session *DebugSession) error
}

func (h *mockEventHook) OnEvent(ctx context.Context, event events.CoreEvent, session *DebugSession) error {
	if h.onEventFunc != nil {
		return h.onEventFunc(ctx, event, session)
	}
	return nil
}

func TestInspector_CreateSession(t *testing.T) {
	inspector := NewInspector(nil)

	// Test creating session
	filters := []EventFilter{&mockEventFilter{name: "test-filter"}}
	hooks := []EventHook{&mockEventHook{}}

	session, err := inspector.CreateSession("Test Session", filters, hooks)
	require.NoError(t, err)
	assert.Equal(t, "Test Session", session.Name)
	assert.Len(t, session.Filters, 1)
	assert.Len(t, session.Hooks, 1)
	assert.True(t, session.IsActive())

	// Test getting session
	retrieved, err := inspector.GetSession(session.ID)
	require.NoError(t, err)
	assert.Equal(t, session.ID, retrieved.ID)

	// Test closing session
	err = inspector.CloseSession(session.ID)
	require.NoError(t, err)
	assert.False(t, session.IsActive())
}

func TestInspector_MaxSessions(t *testing.T) {
	config := &InspectorConfig{
		MaxSessions: 2,
		BufferSize:  100,
	}
	inspector := NewInspector(config)

	// Create max sessions
	_, err := inspector.CreateSession("Session 1", nil, nil)
	require.NoError(t, err)

	_, err = inspector.CreateSession("Session 2", nil, nil)
	require.NoError(t, err)

	// Try to create one more (should fail)
	session3, err := inspector.CreateSession("Session 3", nil, nil)
	assert.Error(t, err)
	assert.Nil(t, session3)
	if err != nil {
		assert.Contains(t, err.Error(), "max sessions exceeded")
	}
}

func TestInspector_InspectEvent(t *testing.T) {
	inspector := NewInspector(nil)
	ctx := context.Background()

	// Start inspector
	err := inspector.Start(ctx)
	require.NoError(t, err)

	var capturedEvent events.CoreEvent
	var capturedSession *DebugSession

	hook := &mockEventHook{
		onEventFunc: func(ctx context.Context, event events.CoreEvent, session *DebugSession) error {
			capturedEvent = event
			capturedSession = session
			return nil
		},
	}

	// Create session with filter
	filter := &mockEventFilter{
		matchFunc: func(event events.CoreEvent) bool {
			return event.GetType() == "test.event"
		},
		name: "type-filter",
	}

	session, err := inspector.CreateSession("Test Session", []EventFilter{filter}, []EventHook{hook})
	require.NoError(t, err)

	// Test matching event
	event1 := events.NewBaseEvent("evt1", "test.event", "test", nil)
	err = inspector.InspectEvent(ctx, event1)
	assert.NoError(t, err)
	assert.Equal(t, event1, capturedEvent)
	assert.Equal(t, session, capturedSession)

	// Test non-matching event
	capturedEvent = nil
	event2 := events.NewBaseEvent("evt2", "other.event", "test", nil)
	err = inspector.InspectEvent(ctx, event2)
	assert.NoError(t, err)
	assert.Nil(t, capturedEvent) // Should not be captured

	// Stop inspector
	err = inspector.Stop()
	assert.NoError(t, err)
}

func TestInspector_Metrics(t *testing.T) {
	config := &InspectorConfig{
		EnableMetrics: true,
		BufferSize:    100,
		MaxSessions:   10,
	}
	inspector := NewInspector(config)
	ctx := context.Background()

	err := inspector.Start(ctx)
	require.NoError(t, err)

	// Create session
	_, err = inspector.CreateSession("Test", nil, nil)
	require.NoError(t, err)

	// Inspect some events
	event := events.NewBaseEvent("evt1", "test.event", "test", nil)
	err = inspector.InspectEvent(ctx, event)
	assert.NoError(t, err)

	// Check metrics
	metrics := inspector.GetMetrics()
	assert.Equal(t, int64(1), metrics.EventsInspected)
	assert.Equal(t, 1, metrics.SessionsActive)
	assert.Equal(t, int64(1), metrics.SessionsTotal)

	err = inspector.Stop()
	assert.NoError(t, err)
}

func TestEventTracer_StartTrace(t *testing.T) {
	tracer := NewEventTracer(nil)
	event := events.NewBaseEvent("evt1", "test.event", "test", nil)

	// Start trace
	trace := tracer.StartTrace(event)
	assert.Equal(t, event.GetID(), trace.EventID)
	assert.Equal(t, event.GetType(), trace.EventType)
	assert.False(t, trace.StartTime.IsZero())

	// Get trace
	retrieved, err := tracer.GetTrace(event.GetID())
	require.NoError(t, err)
	assert.Equal(t, trace, retrieved)

	// Test non-existent trace
	_, err = tracer.GetTrace("non-existent")
	assert.Error(t, err)
}

func TestEventTrace_AddStep(t *testing.T) {
	tracer := NewEventTracer(nil)
	event := events.NewBaseEvent("evt1", "test.event", "test", nil)
	trace := tracer.StartTrace(event)

	// Add step
	trace.AddStep("component1", "process", 100*time.Millisecond, true, nil)

	assert.Len(t, trace.Steps, 1)
	step := trace.Steps[0]
	assert.Equal(t, "component1", step.Component)
	assert.Equal(t, "process", step.Action)
	assert.Equal(t, 100*time.Millisecond, step.Duration)
	assert.True(t, step.Success)
	assert.Empty(t, step.Error)

	// Add step with error
	testErr := assert.AnError
	trace.AddStep("component2", "validate", 50*time.Millisecond, false, testErr)

	assert.Len(t, trace.Steps, 2)
	step = trace.Steps[1]
	assert.Equal(t, "component2", step.Component)
	assert.False(t, step.Success)
	assert.Equal(t, testErr.Error(), step.Error)
}

func TestEventTrace_ToJSON(t *testing.T) {
	tracer := NewEventTracer(nil)
	event := events.NewBaseEvent("evt1", "test.event", "test", nil)
	trace := tracer.StartTrace(event)
	trace.AddStep("component1", "process", 100*time.Millisecond, true, nil)

	jsonData, err := trace.ToJSON()
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "evt1")
	assert.Contains(t, string(jsonData), "test.event")
	assert.Contains(t, string(jsonData), "component1")
}

func TestEventAnalyzer_RecordEvent(t *testing.T) {
	analyzer := NewEventAnalyzer(nil)
	event := events.NewBaseEvent("evt1", "test.event", "test", nil)

	// Record event
	analyzer.RecordEvent(event, 100*time.Millisecond, nil)

	// Get sample
	sample, err := analyzer.GetSample("test.event")
	require.NoError(t, err)
	assert.Equal(t, int64(1), sample.Count)
	assert.Equal(t, 100*time.Millisecond, sample.TotalTime)
	assert.Equal(t, 100*time.Millisecond, sample.MinTime)
	assert.Equal(t, 100*time.Millisecond, sample.MaxTime)
	assert.Equal(t, int64(0), sample.Errors)

	// Record event with error
	analyzer.RecordEvent(event, 200*time.Millisecond, assert.AnError)

	sample, err = analyzer.GetSample("test.event")
	require.NoError(t, err)
	assert.Equal(t, int64(2), sample.Count)
	assert.Equal(t, 300*time.Millisecond, sample.TotalTime)
	assert.Equal(t, 100*time.Millisecond, sample.MinTime)
	assert.Equal(t, 200*time.Millisecond, sample.MaxTime)
	assert.Equal(t, int64(1), sample.Errors)
}

func TestEventAnalyzer_GetPercentiles(t *testing.T) {
	config := &AnalyzerConfig{
		Percentiles: []float64{50, 90, 99},
		MaxSamples:  1000,
	}
	analyzer := NewEventAnalyzer(config)
	event := events.NewBaseEvent("evt1", "test.event", "test", nil)

	// Record events with various durations
	durations := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		30 * time.Millisecond,
		40 * time.Millisecond,
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		100 * time.Millisecond,
	}

	for _, duration := range durations {
		analyzer.RecordEvent(event, duration, nil)
	}

	// Get percentiles
	percentiles, err := analyzer.GetPercentiles("test.event")
	require.NoError(t, err)

	assert.Contains(t, percentiles, 50.0)
	assert.Contains(t, percentiles, 90.0)
	assert.Contains(t, percentiles, 99.0)

	// 50th percentile should be around 50ms
	assert.GreaterOrEqual(t, percentiles[50.0], 40*time.Millisecond)
	assert.LessOrEqual(t, percentiles[50.0], 60*time.Millisecond)
}

func TestEventReplayer_CreateSession(t *testing.T) {
	replayer := NewEventReplayer(nil)

	testEvents := []events.CoreEvent{
		events.NewBaseEvent("evt1", "test.event", "test", nil),
		events.NewBaseEvent("evt2", "test.event", "test", nil),
	}

	var handledEvents []events.CoreEvent
	handler := func(ctx context.Context, event events.CoreEvent) error {
		handledEvents = append(handledEvents, event)
		return nil
	}

	// Create session
	session, err := replayer.CreateSession(testEvents, handler)
	require.NoError(t, err)
	assert.Len(t, session.Events, 2)
	assert.Equal(t, 1.0, session.Speed)
	assert.False(t, session.Running)

	// Start replay
	err = replayer.StartReplay(session.ID)
	assert.NoError(t, err)
	assert.True(t, session.Running)

	// Wait for replay to complete
	time.Sleep(200 * time.Millisecond)

	// Check events were handled
	assert.Len(t, handledEvents, 2)

	// Stop replay
	err = replayer.StopReplay(session.ID)
	assert.NoError(t, err)
}

func TestEventReplayer_MaxSessions(t *testing.T) {
	config := &ReplayerConfig{
		MaxSessions: 1,
	}
	replayer := NewEventReplayer(config)

	testEvents := []events.CoreEvent{
		events.NewBaseEvent("evt1", "test.event", "test", nil),
	}
	handler := func(ctx context.Context, event events.CoreEvent) error { return nil }

	// Create first session
	_, err := replayer.CreateSession(testEvents, handler)
	require.NoError(t, err)

	// Try to create second session (should fail)
	_, err = replayer.CreateSession(testEvents, handler)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max sessions exceeded")
}

func TestReplaySession_Speed(t *testing.T) {
	replayer := NewEventReplayer(nil)

	testEvents := []events.CoreEvent{
		events.NewBaseEvent("evt1", "test.event", "test", nil),
		events.NewBaseEvent("evt2", "test.event", "test", nil),
	}

	var timestamps []time.Time
	handler := func(ctx context.Context, event events.CoreEvent) error {
		timestamps = append(timestamps, time.Now())
		return nil
	}

	session, err := replayer.CreateSession(testEvents, handler)
	require.NoError(t, err)

	// Set high speed
	session.Speed = 10.0

	start := time.Now()
	err = replayer.StartReplay(session.ID)
	assert.NoError(t, err)

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	duration := time.Since(start)
	assert.Less(t, duration, 200*time.Millisecond) // Should be fast
}

func TestEventDumper_DumpJSON(t *testing.T) {
	dumper := NewEventDumper("json")

	testEvents := []events.CoreEvent{
		events.NewBaseEvent("evt1", "test.event", "test", map[string]interface{}{
			"message": "test message",
		}),
	}

	var output strings.Builder
	err := dumper.Dump(&output, testEvents)
	assert.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, "evt1")
	assert.Contains(t, result, "test.event")
	assert.Contains(t, result, "test message")
}

func TestEventDumper_DumpCSV(t *testing.T) {
	dumper := NewEventDumper("csv")

	testEvents := []events.CoreEvent{
		events.NewBaseEvent("evt1", "test.event", "test", map[string]interface{}{
			"count": 42,
		}),
	}

	var output strings.Builder
	err := dumper.Dump(&output, testEvents)
	assert.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, "id,type,source,timestamp,data")
	assert.Contains(t, result, "evt1")
	assert.Contains(t, result, "test.event")
}

func TestEventDumper_DumpText(t *testing.T) {
	dumper := NewEventDumper("text")

	testEvents := []events.CoreEvent{
		events.NewBaseEvent("evt1", "test.event", "test", map[string]interface{}{
			"message": "hello world",
		}),
	}

	var output strings.Builder
	err := dumper.Dump(&output, testEvents)
	assert.NoError(t, err)

	result := output.String()
	assert.Contains(t, result, "Event #1")
	assert.Contains(t, result, "ID: evt1")
	assert.Contains(t, result, "Type: test.event")
	assert.Contains(t, result, "message: hello world")
}

func TestEventDumper_UnsupportedFormat(t *testing.T) {
	dumper := NewEventDumper("xml")

	testEvents := []events.CoreEvent{
		events.NewBaseEvent("evt1", "test.event", "test", nil),
	}

	var output strings.Builder
	err := dumper.Dump(&output, testEvents)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestEventBuffer(t *testing.T) {
	buffer := NewEventBuffer(3)

	// Add events
	snapshot1 := EventSnapshot{
		Event:     events.NewBaseEvent("evt1", "test.event", "test", nil),
		Timestamp: time.Now(),
		Source:    "test",
	}
	buffer.Add(snapshot1)

	snapshot2 := EventSnapshot{
		Event:     events.NewBaseEvent("evt2", "test.event", "test", nil),
		Timestamp: time.Now().Add(time.Second),
		Source:    "test",
	}
	buffer.Add(snapshot2)

	// Get events
	retrievedEvents := buffer.GetEvents(10)
	assert.Len(t, retrievedEvents, 2)
	assert.Equal(t, "evt2", retrievedEvents[0].Event.GetID()) // Most recent first
	assert.Equal(t, "evt1", retrievedEvents[1].Event.GetID())

	// Add more events (should wrap around)
	snapshot3 := EventSnapshot{
		Event:     events.NewBaseEvent("evt3", "test.event", "test", nil),
		Timestamp: time.Now().Add(2 * time.Second),
		Source:    "test",
	}
	buffer.Add(snapshot3)

	snapshot4 := EventSnapshot{
		Event:     events.NewBaseEvent("evt4", "test.event", "test", nil),
		Timestamp: time.Now().Add(3 * time.Second),
		Source:    "test",
	}
	buffer.Add(snapshot4)

	// Should have evt4, evt3, evt2 (evt1 was overwritten)
	retrievedEvents = buffer.GetEvents(10)
	assert.Len(t, retrievedEvents, 3)
	assert.Equal(t, "evt4", retrievedEvents[0].Event.GetID())
	assert.Equal(t, "evt3", retrievedEvents[1].Event.GetID())
	assert.Equal(t, "evt2", retrievedEvents[2].Event.GetID())
}
