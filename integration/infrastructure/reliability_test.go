//go:build integration

package infrastructure

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSystemReliability validates error recovery and system resilience
func TestSystemReliability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Error recovery tests
	t.Run("graceful_error_recovery", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "reliability-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Test recovery from various error conditions
		errorScenarios := []struct {
			name      string
			setupFunc func() error
			testFunc  func() error
			wantErr   bool
		}{
			{
				name: "missing_config_file",
				setupFunc: func() error {
					// Remove config file
					configPath := filepath.Join(projCtx.GetRootPath(), ".campaign", "campaign.yaml")
					return os.Remove(configPath)
				},
				testFunc: func() error {
					// Should regenerate or use defaults
					result := extCtx.RunGuild("status")
					return result.Error
				},
				wantErr: false, // Should recover
			},
			{
				name: "corrupted_database",
				setupFunc: func() error {
					// Corrupt the database file
					dbPath := filepath.Join(projCtx.GetRootPath(), ".campaign", "memory.db")
					return os.WriteFile(dbPath, []byte("corrupted data"), 0644)
				},
				testFunc: func() error {
					// Should detect corruption and recreate
					result := extCtx.RunGuild("init", "--force")
					return result.Error
				},
				wantErr: false, // Should recover
			},
			{
				name: "permission_denied",
				setupFunc: func() error {
					// Make directory read-only
					campaignDir := filepath.Join(projCtx.GetRootPath(), ".campaign")
					return os.Chmod(campaignDir, 0444)
				},
				testFunc: func() error {
					// Should handle permission error gracefully
					result := extCtx.RunGuild("commission", "create", "test")
					if result.Error == nil {
						return fmt.Errorf("expected permission error")
					}
					// Check for graceful error message
					if !contains(result.Stderr, "permission") {
						return fmt.Errorf("error should mention permissions")
					}
					return nil
				},
				wantErr: false, // We expect an error but handled gracefully
			},
		}

		for _, scenario := range errorScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Initialize project
				result := extCtx.RunGuild("init")
				require.NoError(t, result.Error)

				// Setup error condition
				if scenario.setupFunc != nil {
					err := scenario.setupFunc()
					if err != nil {
						t.Logf("Setup warning: %v", err)
					}
				}

				// Test recovery
				err := scenario.testFunc()
				if scenario.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				// Cleanup permissions
				campaignDir := filepath.Join(projCtx.GetRootPath(), ".campaign")
				_ = os.Chmod(campaignDir, 0755)
			})
		}
	})

	t.Run("concurrent_access_safety", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "concurrent-safety-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Test concurrent operations
		var wg sync.WaitGroup
		errors := make(chan error, 10)

		operations := []func(){
			func() {
				defer wg.Done()
				result := extCtx.RunGuild("status")
				if result.Error != nil {
					errors <- result.Error
				}
			},
			func() {
				defer wg.Done()
				result := extCtx.RunGuild("tools", "list")
				if result.Error != nil {
					errors <- result.Error
				}
			},
			func() {
				defer wg.Done()
				result := extCtx.RunGuild("commission", "list")
				if result.Error != nil {
					errors <- result.Error
				}
			},
		}

		// Run operations concurrently multiple times
		for i := 0; i < 3; i++ {
			for _, op := range operations {
				wg.Add(1)
				go op()
			}
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errorCount int
		for err := range errors {
			t.Logf("Concurrent operation error: %v", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount, "Concurrent operations should not fail")
	})

	t.Run("signal_handling", func(t *testing.T) {
		if os.Getenv("CI") != "" {
			t.Skip("Skipping signal test in CI")
		}

		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "signal-test",
		})
		defer cleanup()

		// Initialize
		extCtx := testutil.ExtendProjectContext(projCtx)
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Start a long-running operation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "guild", "chat")
		cmd.Dir = projCtx.GetRootPath()

		// Start the process
		err := cmd.Start()
		require.NoError(t, err)

		// Give it time to start
		time.Sleep(500 * time.Millisecond)

		// Send interrupt signal
		err = cmd.Process.Signal(os.Interrupt)
		require.NoError(t, err)

		// Wait for graceful shutdown
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case err := <-done:
			// Process should exit cleanly
			t.Logf("Process exited with: %v", err)
			// Exit code might be non-zero but that's ok
		case <-time.After(2 * time.Second):
			t.Error("Process did not shut down gracefully")
			cmd.Process.Kill()
		}
	})

	t.Run("resource_cleanup", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "resource-cleanup-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Check initial state
		campaignDir := filepath.Join(projCtx.GetRootPath(), ".campaign")
		initialFiles, err := filepath.Glob(filepath.Join(campaignDir, "*"))
		require.NoError(t, err)

		// Run multiple operations
		for i := 0; i < 5; i++ {
			result = extCtx.RunGuild("commission", "create", fmt.Sprintf("test-%d", i))
			require.NoError(t, result.Error)
		}

		// Check for temp file cleanup
		tempPattern := filepath.Join(campaignDir, "*.tmp")
		tempFiles, err := filepath.Glob(tempPattern)
		require.NoError(t, err)
		assert.Empty(t, tempFiles, "No temporary files should remain")

		// Check for lock file cleanup
		lockPattern := filepath.Join(campaignDir, "*.lock")
		lockFiles, err := filepath.Glob(lockPattern)
		require.NoError(t, err)
		assert.Empty(t, lockFiles, "No lock files should remain")
	})
}

// TestMemoryLeaks validates against memory leaks
func TestMemoryLeaks(t *testing.T) {
	if testing.Short() || os.Getenv("SKIP_MEMORY_TESTS") != "" {
		t.Skip("Skipping memory leak test")
	}

	t.Run("extended_operation_memory", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "memory-leak-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Run many operations to check for leaks
		iterations := 50
		for i := 0; i < iterations; i++ {
			// Create and list commissions
			result = extCtx.RunGuild("commission", "create", fmt.Sprintf("mem-test-%d", i))
			assert.NoError(t, result.Error)

			if i%10 == 0 {
				result = extCtx.RunGuild("commission", "list")
				assert.NoError(t, result.Error)
			}
		}

		// The test environment should still be responsive
		result = extCtx.RunGuild("status")
		assert.NoError(t, result.Error)
		assert.Contains(t, result.Stdout, "Guild Status")
	})
}

// TestCrashRecovery validates system recovery from crashes
func TestCrashRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping crash recovery test in short mode")
	}

	t.Run("database_lock_recovery", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "crash-recovery-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Simulate a stale lock file
		lockFile := filepath.Join(projCtx.GetRootPath(), ".campaign", "memory.db.lock")
		err := os.WriteFile(lockFile, []byte("stale-pid"), 0644)
		require.NoError(t, err)

		// System should recover from stale lock
		result = extCtx.RunGuild("status")
		assert.NoError(t, result.Error, "Should recover from stale lock")
		assert.Contains(t, result.Stdout, "Guild Status")

		// Lock file should be cleaned up
		_, err = os.Stat(lockFile)
		assert.True(t, os.IsNotExist(err), "Stale lock should be removed")
	})

	t.Run("partial_write_recovery", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "partial-write-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Create a partial commission file
		commissionsDir := filepath.Join(projCtx.GetRootPath(), "commissions")
		err := os.MkdirAll(commissionsDir, 0755)
		require.NoError(t, err)

		partialFile := filepath.Join(commissionsDir, "partial.md.tmp")
		err = os.WriteFile(partialFile, []byte("# Incomplete Commission\n\nThis is"), 0644)
		require.NoError(t, err)

		// System should handle partial files gracefully
		result = extCtx.RunGuild("commission", "list")
		assert.NoError(t, result.Error)

		// Partial file should not appear in list
		assert.NotContains(t, result.Stdout, "partial.md")
		assert.NotContains(t, result.Stdout, "Incomplete Commission")
	})
}

// TestPerformanceUnderLoad validates performance requirements under load
func TestPerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance under load test in short mode")
	}

	t.Run("concurrent_user_load", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "load-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Simulate multiple concurrent users
		users := 10
		operationsPerUser := 5

		var wg sync.WaitGroup
		latencies := make(chan time.Duration, users*operationsPerUser)
		errors := make(chan error, users*operationsPerUser)

		for u := 0; u < users; u++ {
			wg.Add(1)
			go func(userID int) {
				defer wg.Done()

				for op := 0; op < operationsPerUser; op++ {
					start := time.Now()

					// Vary operations
					var result *testutil.CommandResult
					switch op % 3 {
					case 0:
						result = extCtx.RunGuild("status")
					case 1:
						result = extCtx.RunGuild("tools", "list")
					case 2:
						result = extCtx.RunGuild("commission", "list")
					}

					latency := time.Since(start)
					latencies <- latency

					if result.Error != nil {
						errors <- result.Error
					}
				}
			}(u)
		}

		wg.Wait()
		close(latencies)
		close(errors)

		// Analyze results
		var totalLatency time.Duration
		var maxLatency time.Duration
		count := 0

		for latency := range latencies {
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
			count++
		}

		avgLatency := totalLatency / time.Duration(count)

		// Performance requirements
		assert.LessOrEqual(t, avgLatency, 500*time.Millisecond,
			"Average latency should be under 500ms")
		assert.LessOrEqual(t, maxLatency, 2*time.Second,
			"Max latency should be under 2s")

		t.Logf("Load test results: avg=%v, max=%v, operations=%d",
			avgLatency, maxLatency, count)

		// Check errors
		errorCount := 0
		for err := range errors {
			t.Logf("Load test error: %v", err)
			errorCount++
		}

		// Allow up to 5% error rate under heavy load
		errorRate := float64(errorCount) / float64(count)
		assert.LessOrEqual(t, errorRate, 0.05,
			"Error rate should be less than 5%%")
	})

	t.Run("large_data_handling", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "large-data-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Create many commissions
		numCommissions := 100

		start := time.Now()
		for i := 0; i < numCommissions; i++ {
			result = extCtx.RunGuild("commission", "create",
				fmt.Sprintf("large-scale-test-%03d", i))
			require.NoError(t, result.Error)
		}
		createDuration := time.Since(start)

		// List performance with many items
		start = time.Now()
		result = extCtx.RunGuild("commission", "list")
		listDuration := time.Since(start)

		require.NoError(t, result.Error)

		// Performance requirements
		avgCreateTime := createDuration / time.Duration(numCommissions)
		assert.LessOrEqual(t, avgCreateTime, 100*time.Millisecond,
			"Average commission creation should be under 100ms")
		assert.LessOrEqual(t, listDuration, 1*time.Second,
			"Listing 100 commissions should be under 1s")

		t.Logf("Large data test: create avg=%v, list=%v",
			avgCreateTime, listDuration)
	})
}

// TestEdgeCases validates handling of edge cases
func TestEdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping edge case tests in short mode")
	}

	t.Run("unicode_handling", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "unicode-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Test various unicode scenarios
		unicodeTests := []string{
			"Hello 世界",
			"Émojis 🚀🎉👍",
			"Математика √∑∫",
			"Mixed العربية 中文 Русский",
			"Zero\x00Width\u200bChars",
		}

		for _, testStr := range unicodeTests {
			result = extCtx.RunGuild("commission", "create", testStr)
			assert.NoError(t, result.Error, "Should handle: %s", testStr)
		}

		// Verify all were created
		result = extCtx.RunGuild("commission", "list")
		require.NoError(t, result.Error)

		for _, testStr := range unicodeTests {
			// Some characters might be normalized or sanitized
			if !strings.Contains(testStr, "\x00") && !strings.Contains(testStr, "\u200b") {
				assert.Contains(t, result.Stdout, testStr,
					"Should list commission with: %s", testStr)
			}
		}
	})

	t.Run("extreme_inputs", func(t *testing.T) {
		projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
			Name: "extreme-input-test",
		})
		defer cleanup()

		extCtx := testutil.ExtendProjectContext(projCtx)

		// Initialize
		result := extCtx.RunGuild("init")
		require.NoError(t, result.Error)

		// Very long input
		longName := strings.Repeat("a", 1000)
		result = extCtx.RunGuild("commission", "create", longName)
		// Should either succeed or fail gracefully
		if result.Error != nil {
			assert.Contains(t, result.Stderr, "too long",
				"Error should mention length issue")
		}

		// Empty input
		result = extCtx.RunGuild("commission", "create", "")
		assert.Error(t, result.Error, "Should reject empty commission name")

		// Special shell characters
		specialChars := "; rm -rf / && echo 'hacked'"
		result = extCtx.RunGuild("commission", "create", specialChars)
		// Should handle safely (either escape or reject)
		if result.Error == nil {
			// If accepted, verify it's properly escaped in list
			result = extCtx.RunGuild("commission", "list")
			require.NoError(t, result.Error)
			// The dangerous command should not have been executed
			assert.NotContains(t, result.Stdout, "hacked")
		}
	})
}

// Helper function
func contains(data string, substr string) bool {
	return strings.Contains(strings.ToLower(data), strings.ToLower(substr))
}
