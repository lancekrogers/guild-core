package collectors

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestNewChatCollector(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	
	collector, err := NewChatCollector(meter)
	require.NoError(t, err)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.meter)
}

func TestChatCollector_RecordSessionCreated(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewChatCollector(meter)
	require.NoError(t, err)

	tests := []struct {
		name       string
		sessionID  string
		userID     string
		sessionType string
	}{
		{
			name:       "normal session",
			sessionID:  "session-123",
			userID:     "user-456",
			sessionType: "interactive",
		},
		{
			name:       "api session",
			sessionID:  "api-session-789",
			userID:     "api-user-012",
			sessionType: "api",
		},
		{
			name:       "empty session ID",
			sessionID:  "",
			userID:     "user-123",
			sessionType: "interactive",
		},
		{
			name:       "anonymous user",
			sessionID:  "session-anon",
			userID:     "",
			sessionType: "anonymous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				collector.RecordSessionStart(ctx, tt.sessionID, tt.userID)
			})
		})
	}
}

func TestChatCollector_RecordSessionEnd(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewChatCollector(meter)
	require.NoError(t, err)

	// Create sessions first
	sessionID := "test-session"
	collector.RecordSessionStart(ctx, sessionID, "user-123")

	// End the session
	assert.NotPanics(t, func() {
		collector.RecordSessionEnd(ctx, sessionID, "user-123", 5*time.Second)
	})

	// End non-existent session
	assert.NotPanics(t, func() {
		collector.RecordSessionEnd(ctx, "non-existent", "user-123", 1*time.Second)
	})

	// End empty session ID
	assert.NotPanics(t, func() {
		collector.RecordSessionEnd(ctx, "", "user-123", 0)
	})
}

func TestChatCollector_RecordMessage(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewChatCollector(meter)
	require.NoError(t, err)

	tests := []struct {
		name        string
		sessionID   string
		messageType string
		provider    string
	}{
		{
			name:        "user message",
			sessionID:   "session-123",
			messageType: "user",
			provider:    "openai",
		},
		{
			name:        "system message",
			sessionID:   "session-123",
			messageType: "system",
			provider:    "anthropic",
		},
		{
			name:        "assistant message",
			sessionID:   "session-123",
			messageType: "assistant",
			provider:    "ollama",
		},
		{
			name:        "empty provider",
			sessionID:   "session-456",
			messageType: "user",
			provider:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				collector.RecordMessage(ctx, tt.sessionID, tt.messageType, tt.provider)
			})
		})
	}
}


func TestChatCollector_RecordResponse(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewChatCollector(meter)
	require.NoError(t, err)

	tests := []struct {
		name         string
		sessionID    string
		responseType string
		duration     time.Duration
	}{
		{
			name:         "fast response",
			sessionID:    "session-123",
			responseType: "completion",
			duration:     100 * time.Millisecond,
		},
		{
			name:         "slow response",
			sessionID:    "session-456",
			responseType: "completion",
			duration:     5 * time.Second,
		},
		{
			name:         "streaming response",
			sessionID:    "session-789",
			responseType: "streaming",
			duration:     2 * time.Second,
		},
		{
			name:         "error response",
			sessionID:    "session-error",
			responseType: "error",
			duration:     50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				collector.RecordResponse(ctx, tt.sessionID, "openai", tt.duration, 100, true)
			})
		})
	}
}



func TestChatCollector_ConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	provider := sdkmetric.NewMeterProvider()
	meter := provider.Meter("test")
	collector, err := NewChatCollector(meter)
	require.NoError(t, err)

	// Create multiple sessions concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			sessionID := "session-" + string(rune('0'+id))
			userID := "user-" + string(rune('0'+id))

			// Create session
			collector.RecordSessionStart(ctx, sessionID, userID)

			// Send messages
			for j := 0; j < 5; j++ {
				collector.RecordMessage(ctx, sessionID, "user", "openai")
				collector.RecordResponse(ctx, sessionID, "openai", time.Duration(100+j*50)*time.Millisecond, 200+j*20, true)
			}

			// Record suggestion usage
			collector.RecordSuggestion(ctx, sessionID, "code-completion", id%2 == 0)

			// Some sessions record streaming latency
			if id%3 == 0 {
				collector.RecordStreamingLatency(ctx, sessionID, id, time.Duration(10+id)*time.Millisecond)
			}

			// End session
			collector.RecordSessionEnd(ctx, sessionID, userID, time.Duration(id+1)*time.Second)
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should complete without race conditions
}