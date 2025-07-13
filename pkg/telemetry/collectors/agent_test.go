package collectors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestNewAgentCollector(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")

	collector, err := NewAgentCollector(meter)
	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.meter)
	assert.NotNil(t, collector.executionTime)
	assert.NotNil(t, collector.taskCompletions)
	assert.NotNil(t, collector.taskFailures)
	assert.NotNil(t, collector.concurrentExecutions)
	assert.NotNil(t, collector.toolInvocations)
	assert.NotNil(t, collector.contextSwitches)
	assert.NotNil(t, collector.memoryRetrieval)
}

func TestAgentCollector_RecordExecution(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewAgentCollector(meter)
	require.NoError(t, err)

	tests := []struct {
		name      string
		agentName string
		taskType  string
		duration  time.Duration
		success   bool
	}{
		{
			name:      "successful execution",
			agentName: "test-agent",
			taskType:  "code-generation",
			duration:  100 * time.Millisecond,
			success:   true,
		},
		{
			name:      "failed execution",
			agentName: "test-agent",
			taskType:  "code-review",
			duration:  50 * time.Millisecond,
			success:   false,
		},
		{
			name:      "long execution",
			agentName: "slow-agent",
			taskType:  "analysis",
			duration:  5 * time.Second,
			success:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			assert.NotPanics(t, func() {
				collector.RecordExecution(ctx, tt.agentName, tt.taskType, tt.duration, tt.success)
			})
		})
	}
}

func TestAgentCollector_RecordConcurrent(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewAgentCollector(meter)
	require.NoError(t, err)

	// Test concurrent start/end
	agentName := "concurrent-agent"

	// Start multiple concurrent executions
	for i := 0; i < 5; i++ {
		collector.RecordConcurrentStart(ctx, agentName)
	}

	// End some executions
	for i := 0; i < 3; i++ {
		collector.RecordConcurrentEnd(ctx, agentName)
	}

	// Should handle empty agent name
	assert.NotPanics(t, func() {
		collector.RecordConcurrentStart(ctx, "")
		collector.RecordConcurrentEnd(ctx, "")
	})
}

func TestAgentCollector_RecordToolInvocation(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewAgentCollector(meter)
	require.NoError(t, err)

	tests := []struct {
		agentName string
		toolName  string
	}{
		{"agent1", "file-reader"},
		{"agent2", "code-analyzer"},
		{"agent1", "test-runner"},
		{"", "empty-agent"},
		{"agent3", ""},
	}

	for _, tt := range tests {
		assert.NotPanics(t, func() {
			collector.RecordToolInvocation(ctx, tt.agentName, tt.toolName)
		})
	}
}

func TestAgentCollector_RecordContextSwitch(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewAgentCollector(meter)
	require.NoError(t, err)

	tests := []struct {
		name        string
		agentName   string
		fromContext string
		toContext   string
	}{
		{
			name:        "normal switch",
			agentName:   "agent1",
			fromContext: "file-analysis",
			toContext:   "code-generation",
		},
		{
			name:        "empty contexts",
			agentName:   "agent2",
			fromContext: "",
			toContext:   "new-context",
		},
		{
			name:        "same context",
			agentName:   "agent3",
			fromContext: "context1",
			toContext:   "context1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				collector.RecordContextSwitch(ctx, tt.agentName, tt.fromContext, tt.toContext)
			})
		})
	}
}

func TestAgentCollector_RecordMemoryRetrieval(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewAgentCollector(meter)
	require.NoError(t, err)

	tests := []struct {
		name          string
		agentName     string
		retrievalType string
		duration      time.Duration
		itemCount     int
	}{
		{
			name:          "vector search",
			agentName:     "agent1",
			retrievalType: "vector",
			duration:      10 * time.Millisecond,
			itemCount:     25,
		},
		{
			name:          "sql query",
			agentName:     "agent2",
			retrievalType: "sql",
			duration:      5 * time.Millisecond,
			itemCount:     100,
		},
		{
			name:          "cache hit",
			agentName:     "agent3",
			retrievalType: "cache",
			duration:      1 * time.Millisecond,
			itemCount:     1,
		},
		{
			name:          "no results",
			agentName:     "agent4",
			retrievalType: "vector",
			duration:      15 * time.Millisecond,
			itemCount:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				collector.RecordMemoryRetrieval(ctx, tt.agentName, tt.retrievalType, tt.duration, tt.itemCount)
			})
		})
	}
}

func TestAgentCollector_ConcurrentRecording(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewAgentCollector(meter)
	require.NoError(t, err)

	// Run multiple goroutines recording metrics concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			agentName := "agent-" + string(rune('0'+id))

			// Record various metrics
			collector.RecordExecution(ctx, agentName, "task", time.Duration(id)*time.Millisecond, id%2 == 0)
			collector.RecordConcurrentStart(ctx, agentName)
			collector.RecordToolInvocation(ctx, agentName, "tool"+string(rune('0'+id)))
			collector.RecordContextSwitch(ctx, agentName, "ctx1", "ctx2")
			collector.RecordMemoryRetrieval(ctx, agentName, "type", time.Duration(id)*time.Millisecond, id*10)
			collector.RecordConcurrentEnd(ctx, agentName)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without race conditions
}
