// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock tool for testing health checks
type mockHealthTool struct {
	*BaseTool
	healthFunc      func() error
	healthCallCount int32
	mu              sync.Mutex
}

func (m *mockHealthTool) HealthCheck() error {
	atomic.AddInt32(&m.healthCallCount, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.healthFunc != nil {
		return m.healthFunc()
	}
	return nil
}

func (m *mockHealthTool) getCallCount() int32 {
	return atomic.LoadInt32(&m.healthCallCount)
}

func TestHealthChecker_CheckHealth(t *testing.T) {
	t.Run("healthy tool", func(t *testing.T) {
		checker := NewHealthChecker()
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("test-tool", "Test Tool", nil, "test", false, nil),
		}

		err := checker.CheckHealth(context.Background(), tool)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), tool.getCallCount())
	})

	t.Run("unhealthy tool", func(t *testing.T) {
		checker := NewHealthChecker()
		expectedErr := errors.New("tool is broken")
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("broken-tool", "Broken Tool", nil, "test", false, nil),
			healthFunc: func() error { return expectedErr },
		}

		err := checker.CheckHealth(context.Background(), tool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tool is broken")
	})

	t.Run("caching prevents multiple checks", func(t *testing.T) {
		checker := NewHealthChecker()
		checker.SetCheckInterval(1 * time.Hour) // Long cache duration
		
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("cached-tool", "Cached Tool", nil, "test", false, nil),
		}

		// First check
		err := checker.CheckHealth(context.Background(), tool)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), tool.getCallCount())

		// Second check should use cache
		err = checker.CheckHealth(context.Background(), tool)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), tool.getCallCount()) // No additional calls
	})

	t.Run("stale cache triggers new check", func(t *testing.T) {
		checker := NewHealthChecker()
		checker.SetCheckInterval(10 * time.Millisecond) // Short cache duration
		
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("stale-cache-tool", "Stale Cache Tool", nil, "test", false, nil),
		}

		// First check
		err := checker.CheckHealth(context.Background(), tool)
		assert.NoError(t, err)
		assert.Equal(t, int32(1), tool.getCallCount())

		// Wait for cache to expire
		time.Sleep(20 * time.Millisecond)

		// Should trigger new check
		err = checker.CheckHealth(context.Background(), tool)
		assert.NoError(t, err)
		assert.Equal(t, int32(2), tool.getCallCount())
	})

	t.Run("consecutive failures tracking", func(t *testing.T) {
		checker := NewHealthChecker()
		checker.SetCheckInterval(1 * time.Millisecond) // Very short cache for testing
		
		failCount := 0
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("flaky-tool", "Flaky Tool", nil, "test", false, nil),
			healthFunc: func() error {
				failCount++
				return errors.New("fail")
			},
		}

		// Multiple failures
		for i := 0; i < 3; i++ {
			_ = checker.CheckHealth(context.Background(), tool)
			time.Sleep(2 * time.Millisecond) // Ensure cache expires
		}

		// Check status
		status, exists := checker.GetHealthStatus("flaky-tool")
		assert.True(t, exists)
		assert.False(t, status.Healthy)
		assert.Equal(t, 3, status.ConsecutiveFails)
	})

	t.Run("health check timeout", func(t *testing.T) {
		checker := NewHealthChecker()
		checker.timeout = 50 * time.Millisecond
		
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("slow-tool", "Slow Tool", nil, "test", false, nil),
			healthFunc: func() error {
				time.Sleep(100 * time.Millisecond) // Longer than timeout
				return nil
			},
		}

		ctx := context.Background()
		err := checker.CheckHealth(ctx, tool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
	})

	t.Run("context cancellation", func(t *testing.T) {
		checker := NewHealthChecker()
		
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("cancel-tool", "Cancel Tool", nil, "test", false, nil),
			healthFunc: func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := checker.CheckHealth(ctx, tool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestHealthChecker_HealthReport(t *testing.T) {
	checker := NewHealthChecker()
	
	// Add some tools with different statuses
	tools := []struct {
		name    string
		healthy bool
		err     error
	}{
		{"tool1", true, nil},
		{"tool2", true, nil},
		{"tool3", false, errors.New("broken")},
		{"tool4", true, nil},
		{"tool5", false, errors.New("unavailable")},
	}

	for _, tc := range tools {
		tool := &mockHealthTool{
			BaseTool: NewBaseTool(tc.name, tc.name, nil, "test", false, nil),
			healthFunc: func() error { return tc.err },
		}
		// Prime the cache
		_ = checker.CheckHealth(context.Background(), tool)
	}

	report := checker.GenerateHealthReport()
	assert.Equal(t, 5, report.TotalTools)
	assert.Equal(t, 3, report.HealthyTools)
	assert.Equal(t, 2, report.UnhealthyTools)
	assert.Len(t, report.Tools, 5)
}

func TestBackgroundHealthChecker(t *testing.T) {
	t.Run("periodic health checks", func(t *testing.T) {
		registry := NewToolRegistry()
		
		// Register some tools
		tool1 := &mockHealthTool{
			BaseTool: NewBaseTool("bg-tool1", "BG Tool 1", nil, "test", false, nil),
		}
		tool2 := &mockHealthTool{
			BaseTool: NewBaseTool("bg-tool2", "BG Tool 2", nil, "test", false, nil),
		}
		
		err := registry.RegisterTool(tool1)
		require.NoError(t, err)
		err = registry.RegisterTool(tool2)
		require.NoError(t, err)

		// Create background checker with short interval
		bgChecker := NewBackgroundHealthChecker(registry, 50*time.Millisecond)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		bgChecker.Start(ctx)
		defer bgChecker.Stop()

		// Wait for at least one check cycle
		time.Sleep(100 * time.Millisecond)

		// Verify tools were checked
		assert.Greater(t, tool1.getCallCount(), int32(0))
		assert.Greater(t, tool2.getCallCount(), int32(0))
	})

	t.Run("stop gracefully", func(t *testing.T) {
		registry := NewToolRegistry()
		bgChecker := NewBackgroundHealthChecker(registry, 1*time.Hour)
		
		ctx := context.Background()
		bgChecker.Start(ctx)
		
		// Should not block
		done := make(chan struct{})
		go func() {
			bgChecker.Stop()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("Stop() blocked for too long")
		}
	})
}

func TestHealthCheckMiddleware(t *testing.T) {
	checker := NewHealthChecker()
	
	t.Run("allows execution when healthy", func(t *testing.T) {
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("middleware-tool", "Middleware Tool", nil, "test", false, nil),
		}
		
		// Wrap with middleware
		middleware := HealthCheckMiddleware(checker)
		wrapped := middleware(tool)
		
		// Should execute successfully
		result, err := wrapped.Execute(context.Background(), "{}")
		// Note: BaseTool.Execute returns error, but middleware should still call it
		assert.Error(t, err) // Expected from BaseTool
		assert.Nil(t, result)
		assert.Equal(t, int32(1), tool.getCallCount()) // Health check was called
	})

	t.Run("blocks execution when unhealthy", func(t *testing.T) {
		tool := &mockHealthTool{
			BaseTool: NewBaseTool("unhealthy-middleware-tool", "Unhealthy Tool", nil, "test", false, nil),
			healthFunc: func() error { return errors.New("unhealthy") },
		}
		
		// Wrap with middleware
		middleware := HealthCheckMiddleware(checker)
		wrapped := middleware(tool)
		
		// Should fail health check
		result, err := wrapped.Execute(context.Background(), "{}")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "health check failed")
		assert.Nil(t, result)
	})
}

func TestHealthStatus_ResponseTime(t *testing.T) {
	checker := NewHealthChecker()
	
	tool := &mockHealthTool{
		BaseTool: NewBaseTool("timing-tool", "Timing Tool", nil, "test", false, nil),
		healthFunc: func() error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	}

	err := checker.CheckHealth(context.Background(), tool)
	assert.NoError(t, err)

	status, exists := checker.GetHealthStatus("timing-tool")
	assert.True(t, exists)
	assert.True(t, status.Healthy)
	assert.Greater(t, status.ResponseTime.Milliseconds(), int64(40)) // Should be at least 50ms
	assert.Less(t, status.ResponseTime.Milliseconds(), int64(100))   // But not too long
}

func TestHealthChecker_ClearCache(t *testing.T) {
	checker := NewHealthChecker()
	
	// Add a tool to cache
	tool := &mockHealthTool{
		BaseTool: NewBaseTool("cache-clear-tool", "Cache Clear Tool", nil, "test", false, nil),
	}
	
	err := checker.CheckHealth(context.Background(), tool)
	assert.NoError(t, err)
	
	// Verify it's cached
	status, exists := checker.GetHealthStatus("cache-clear-tool")
	assert.True(t, exists)
	assert.NotNil(t, status)
	
	// Clear cache
	checker.ClearCache()
	
	// Should no longer exist
	status, exists = checker.GetHealthStatus("cache-clear-tool")
	assert.False(t, exists)
	assert.Nil(t, status)
}