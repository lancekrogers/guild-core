//go:build integration

package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lancekrogers/guild/internal/testutil"
	"github.com/lancekrogers/guild/pkg/gerror"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestChatInterface validates all chat commands and interactions
func TestChatInterface(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "chat-interface-test",
	})
	defer cleanup()

	extCtx := testutil.ExtendProjectContext(projCtx)

	t.Run("chat_commands", func(t *testing.T) {
		// Test help command
		result := extCtx.RunGuild("chat", "--help")
		require.NoError(t, result.Error)
		assert.Contains(t, result.Stdout, "chat")
		assert.Contains(t, result.Stdout, "Start a chat session")
	})

	t.Run("chat_initialization", func(t *testing.T) {
		// Initialize guild first
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Test status after init
		result = extCtx.RunGuild("status")
		require.NoError(t, result.Error)
		assert.Contains(t, result.Stdout, "Guild Status")
	})

	t.Run("commission_via_chat", func(t *testing.T) {
		// Create commission
		result := extCtx.RunGuild("commission", "create", "Test Commission")
		require.NoError(t, result.Error)

		// List commissions
		result = extCtx.RunGuild("commission", "list")
		require.NoError(t, result.Error)
		assert.Contains(t, result.Stdout, "Test Commission")
	})
}

// TestChatPerformance validates chat interface performance
func TestChatPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "chat-performance",
	})
	defer cleanup()

	extCtx := testutil.ExtendProjectContext(projCtx)

	// Initialize
	result := extCtx.RunGuild("init")
	require.NoError(t, result.Error)

	// Performance requirements
	requirements := []struct {
		name       string
		operation  func() *testutil.CommandResult
		maxLatency time.Duration
	}{
		{
			name: "command_response_time",
			operation: func() *testutil.CommandResult {
				return extCtx.RunGuild("status")
			},
			maxLatency: 200 * time.Millisecond,
		},
		{
			name: "help_command_time",
			operation: func() *testutil.CommandResult {
				return extCtx.RunGuild("help")
			},
			maxLatency: 100 * time.Millisecond,
		},
	}

	for _, req := range requirements {
		t.Run(req.name, func(t *testing.T) {
			// Warm up
			_ = req.operation()

			// Measure
			start := time.Now()
			result := req.operation()
			duration := time.Since(start)

			require.NoError(t, result.Error)
			assert.LessOrEqual(t, duration, req.maxLatency,
				"%s should complete within %v, took %v", req.name, req.maxLatency, duration)

			t.Logf("%s latency: %v", req.name, duration)
		})
	}
}

// TestChatResilience tests error recovery and edge cases
func TestChatResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilience test in short mode")
	}

	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "chat-resilience",
	})
	defer cleanup()

	extCtx := testutil.ExtendProjectContext(projCtx)

	// Initialize
	result := extCtx.RunGuild("init")
	require.NoError(t, result.Error)

	t.Run("invalid_command", func(t *testing.T) {
		result := extCtx.RunGuild("invalid-command")
		assert.Error(t, result.Error)
		assert.Contains(t, result.Stderr, "unknown command")
	})

	t.Run("empty_commission_title", func(t *testing.T) {
		result := extCtx.RunGuild("commission", "create", "")
		assert.Error(t, result.Error)
	})

	t.Run("special_characters", func(t *testing.T) {
		specialTitles := []string{
			"Test with emoji 🎉",
			"Test with unicode αβγ",
			"Test with quotes \"hello\"",
		}

		for _, title := range specialTitles {
			result := extCtx.RunGuild("commission", "create", title)
			// Should either succeed or fail gracefully
			if result.Error != nil {
				assert.NotContains(t, result.Stderr, "panic")
			}
		}
	})

	t.Run("concurrent_commands", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				result := extCtx.RunGuild("status")
				if result.Error != nil {
					errors <- result.Error
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errorCount int
		for err := range errors {
			t.Logf("Concurrent command error: %v", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount, "Concurrent commands should not fail")
	})
}
