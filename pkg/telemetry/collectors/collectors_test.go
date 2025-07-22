package collectors

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
)

func TestSystemCollector(t *testing.T) {
	meter := noop.NewMeterProvider().Meter("test")
	collector := NewSystemCollector()

	// Test that the collector can be registered without error
	err := collector.Register(meter)
	if err != nil {
		t.Fatalf("failed to register system collector: %v", err)
	}

	// The actual observation methods are internal implementation details
	// and are tested through integration tests when metrics are collected
}

func TestAgentCollector(t *testing.T) {
	ctx := context.Background()
	meter := noop.NewMeterProvider().Meter("test")

	collector, err := NewAgentCollector(meter)
	if err != nil {
		t.Fatalf("failed to create agent collector: %v", err)
	}

	// Test all recording methods
	t.Run("RecordExecution", func(t *testing.T) {
		collector.RecordExecution(ctx, "test-agent", "analysis", 100*time.Millisecond, true)
		collector.RecordExecution(ctx, "test-agent", "generation", 200*time.Millisecond, false)
	})

	t.Run("RecordConcurrent", func(t *testing.T) {
		collector.RecordConcurrentStart(ctx, "test-agent")
		collector.RecordConcurrentEnd(ctx, "test-agent")
	})

	t.Run("RecordToolInvocation", func(t *testing.T) {
		collector.RecordToolInvocation(ctx, "test-agent", "file-reader")
		collector.RecordToolInvocation(ctx, "test-agent", "code-analyzer")
	})

	t.Run("RecordContextSwitch", func(t *testing.T) {
		collector.RecordContextSwitch(ctx, "test-agent", "planning", "execution")
	})

	t.Run("RecordMemoryRetrieval", func(t *testing.T) {
		collector.RecordMemoryRetrieval(ctx, "test-agent", "vector-search", 50*time.Millisecond, 10)
	})
}

func TestChatCollector(t *testing.T) {
	ctx := context.Background()
	meter := noop.NewMeterProvider().Meter("test")

	collector, err := NewChatCollector(meter)
	if err != nil {
		t.Fatalf("failed to create chat collector: %v", err)
	}

	sessionID := "session-123"
	userID := "user-456"

	t.Run("RecordMessage", func(t *testing.T) {
		collector.RecordMessage(ctx, sessionID, "user", "openai")
		collector.RecordMessage(ctx, sessionID, "assistant", "openai")
	})

	t.Run("RecordResponse", func(t *testing.T) {
		collector.RecordResponse(ctx, sessionID, "openai", 500*time.Millisecond, 150, true)
		collector.RecordResponse(ctx, sessionID, "anthropic", 1*time.Second, 0, false)
	})

	t.Run("RecordSession", func(t *testing.T) {
		collector.RecordSessionStart(ctx, sessionID, userID)
		time.Sleep(10 * time.Millisecond)
		collector.RecordSessionEnd(ctx, sessionID, userID, 10*time.Minute)
	})

	t.Run("RecordStreamingLatency", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			collector.RecordStreamingLatency(ctx, sessionID, i, time.Duration(10+i)*time.Millisecond)
		}
	})

	t.Run("RecordSuggestion", func(t *testing.T) {
		collector.RecordSuggestion(ctx, sessionID, "command", true)
		collector.RecordSuggestion(ctx, sessionID, "followup", false)
	})
}

func TestCommissionCollector(t *testing.T) {
	ctx := context.Background()
	meter := noop.NewMeterProvider().Meter("test")

	collector, err := NewCommissionCollector(meter)
	if err != nil {
		t.Fatalf("failed to create commission collector: %v", err)
	}

	commissionID := "comm-789"

	t.Run("RecordExecution", func(t *testing.T) {
		collector.RecordExecution(ctx, commissionID, "code-generation", 5*time.Second, true)
	})

	t.Run("RecordTaskGenerated", func(t *testing.T) {
		collector.RecordTaskGenerated(ctx, commissionID, "analysis", 3)
		collector.RecordTaskGenerated(ctx, commissionID, "implementation", 5)
	})

	t.Run("RecordTaskCompletion", func(t *testing.T) {
		collector.RecordTaskCompletion(ctx, commissionID, "task-1", true)
		collector.RecordTaskCompletion(ctx, commissionID, "task-2", false)
	})

	t.Run("RecordRefinement", func(t *testing.T) {
		collector.RecordRefinement(ctx, commissionID, "clarification")
	})

	t.Run("RecordComplexity", func(t *testing.T) {
		collector.RecordComplexity(ctx, commissionID, 7.5)
	})

	t.Run("RecordCommissionLifecycle", func(t *testing.T) {
		collector.RecordCommissionStart(ctx, commissionID)
		time.Sleep(10 * time.Millisecond)
		collector.RecordCommissionEnd(ctx, commissionID)
	})

	t.Run("RecordResourceUtilization", func(t *testing.T) {
		collector.RecordResourceUtilization(ctx, commissionID, "cpu", 75.5)
		collector.RecordResourceUtilization(ctx, commissionID, "memory", 60.2)
	})

	t.Run("ObserveQueueDepth", func(t *testing.T) {
		queueDepth := int64(5)
		err := collector.ObserveQueueDepth(func() int64 {
			return queueDepth
		})
		if err != nil {
			t.Errorf("failed to set queue depth observer: %v", err)
		}
	})
}

// Table-driven tests for edge cases
func TestCollectorEdgeCases(t *testing.T) {
	ctx := context.Background()
	meter := noop.NewMeterProvider().Meter("test")

	t.Run("AgentCollector empty values", func(t *testing.T) {
		collector, _ := NewAgentCollector(meter)

		// Empty agent name
		collector.RecordExecution(ctx, "", "task", 100*time.Millisecond, true)

		// Zero duration
		collector.RecordExecution(ctx, "agent", "task", 0, true)

		// Empty tool name
		collector.RecordToolInvocation(ctx, "agent", "")

		// Same context switch
		collector.RecordContextSwitch(ctx, "agent", "same", "same")

		// Zero items retrieved
		collector.RecordMemoryRetrieval(ctx, "agent", "search", 10*time.Millisecond, 0)
	})

	t.Run("ChatCollector edge cases", func(t *testing.T) {
		collector, _ := NewChatCollector(meter)

		// Empty session ID
		collector.RecordMessage(ctx, "", "user", "provider")

		// Zero token response
		collector.RecordResponse(ctx, "session", "provider", 100*time.Millisecond, 0, true)

		// Negative duration (should not happen but test resilience)
		collector.RecordSessionEnd(ctx, "session", "user", -1*time.Second)

		// Very high chunk index
		collector.RecordStreamingLatency(ctx, "session", 999999, 10*time.Millisecond)
	})

	t.Run("CommissionCollector edge cases", func(t *testing.T) {
		collector, _ := NewCommissionCollector(meter)

		// Empty commission ID
		collector.RecordExecution(ctx, "", "type", 1*time.Second, true)

		// Zero tasks generated
		collector.RecordTaskGenerated(ctx, "comm", "type", 0)

		// Negative complexity (invalid but test handling)
		collector.RecordComplexity(ctx, "comm", -1.0)

		// Over 100% utilization
		collector.RecordResourceUtilization(ctx, "comm", "cpu", 150.0)

		// Queue depth callback returning negative
		_ = collector.ObserveQueueDepth(func() int64 {
			return -1
		})
	})
}

// Benchmark tests
func BenchmarkAgentCollector(b *testing.B) {
	ctx := context.Background()
	meter := noop.NewMeterProvider().Meter("bench")
	collector, _ := NewAgentCollector(meter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordExecution(ctx, "bench-agent", "task", 100*time.Millisecond, true)
	}
}

func BenchmarkChatCollector(b *testing.B) {
	ctx := context.Background()
	meter := noop.NewMeterProvider().Meter("bench")
	collector, _ := NewChatCollector(meter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordResponse(ctx, "session", "provider", 200*time.Millisecond, 100, true)
	}
}

func BenchmarkCommissionCollector(b *testing.B) {
	ctx := context.Background()
	meter := noop.NewMeterProvider().Meter("bench")
	collector, _ := NewCommissionCollector(meter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordTaskCompletion(ctx, "comm", "task", i%2 == 0)
	}
}
