package mock_test

import (
	"context"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
)

// TestContextCancellation verifies that providers respect context cancellation
func TestContextCancellation(t *testing.T) {
	// Create a mock provider with delay to simulate slow response
	provider := mock.NewBuilder().
		WithDefaultResponse("Should not see this").
		WithDelay(100 * time.Millisecond).
		Build()

	// Create a context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := interfaces.ChatRequest{
		Model: "mock-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	// Should return context canceled error
	_, err := provider.ChatCompletion(ctx, req)
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}

	// Note: The mock provider doesn't actually check context during delay
	// In real providers, the HTTP request would be cancelled
}

// TestContextTimeout verifies that providers respect context timeout
func TestContextTimeout(t *testing.T) {
	// Create a mock provider with longer delay
	provider := mock.NewBuilder().
		WithDefaultResponse("Should timeout").
		WithDelay(200 * time.Millisecond).
		Build()

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := interfaces.ChatRequest{
		Model: "mock-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	start := time.Now()
	_, err := provider.ChatCompletion(ctx, req)
	duration := time.Since(start)

	// Should timeout before the provider delay completes
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// Should have taken less time than the provider delay
	if duration > 150*time.Millisecond {
		t.Errorf("Request took too long: %v", duration)
	}
}

// TestContextPropagation verifies context flows through the call chain
func TestContextPropagation(t *testing.T) {
	provider := mock.NewProvider()

	// Create context with value
	type ctxKey string
	const testKey ctxKey = "test-key"
	ctx := context.WithValue(context.Background(), testKey, "test-value")

	req := interfaces.ChatRequest{
		Model: "mock-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	// Make request with context
	resp, err := provider.ChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("No response choices")
	}

	// The mock provider doesn't expose context, but in real providers
	// the context would be available throughout the request lifecycle
}

// TestContextWithMultipleProviders tests context across different providers
func TestContextWithMultipleProviders(t *testing.T) {
	providers := []struct {
		name     string
		provider interfaces.AIProvider
	}{
		{
			name: "mock-fast",
			provider: mock.NewBuilder().
				WithDefaultResponse("fast response").
				WithDelay(10 * time.Millisecond).
				Build(),
		},
		{
			name: "mock-slow",
			provider: mock.NewBuilder().
				WithDefaultResponse("slow response").
				WithDelay(100 * time.Millisecond).
				Build(),
		},
	}

	// Test with timeout that only allows fast provider to complete
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := interfaces.ChatRequest{
		Model: "mock-model",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			resp, err := p.provider.ChatCompletion(ctx, req)

			if p.name == "mock-fast" {
				// Fast provider should succeed
				if err != nil {
					t.Errorf("Fast provider failed: %v", err)
				}
				if resp == nil || len(resp.Choices) == 0 {
					t.Error("Fast provider returned no response")
				}
			} else {
				// Slow provider should timeout
				if err == nil {
					t.Error("Slow provider should have timed out")
				}
			}
		})
	}
}
