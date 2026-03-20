// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package e2e

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPerformanceMetrics(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Startup Time", func(t *testing.T) {
		// Measure cold start
		start := time.Now()
		result := env.RunGuild("version")
		coldStart := time.Since(start)

		result.AssertSuccess(t)
		assert.Less(t, coldStart, 3*time.Second,
			"Cold start too slow: %v", coldStart)

		// Measure help command start
		start = time.Now()
		result = env.RunGuild("help")
		helpStart := time.Since(start)

		result.AssertSuccess(t)
		assert.Less(t, helpStart, 5*time.Second,
			"Help command too slow: %v", helpStart)

		t.Logf("Performance metrics - Cold start: %v, Help: %v", coldStart, helpStart)
	})

	t.Run("Memory Usage", func(t *testing.T) {
		var m runtime.MemStats

		// Baseline memory
		runtime.GC()
		runtime.ReadMemStats(&m)
		baseline := m.Alloc

		// Run some commands
		env.RunGuild("help").AssertSuccess(t)
		env.RunGuild("version").AssertSuccess(t)

		// Check memory after
		runtime.GC()
		runtime.ReadMemStats(&m)
		used := m.Alloc - baseline

		// Memory usage should be reasonable for simple commands
		// Allow up to 50MB for CLI operations
		maxMemory := uint64(50 * 1024 * 1024)
		assert.Less(t, used, maxMemory,
			"Memory usage too high: %d MB", used/1024/1024)

		t.Logf("Memory usage: %d KB", used/1024)
	})

	t.Run("Command Execution Speed", func(t *testing.T) {
		// Initialize project first
		env.RunGuild("init").AssertSuccess(t)

		// Test various command speeds
		commands := []struct {
			name    string
			cmd     []string
			maxTime time.Duration
		}{
			{"status", []string{"status"}, 5 * time.Second},
			{"config", []string{"config", "show"}, 3 * time.Second},
			{"help", []string{"help"}, 3 * time.Second},
			{"agent_list", []string{"agent", "list"}, 5 * time.Second},
			{"campaign_list", []string{"campaign", "list"}, 5 * time.Second},
		}

		for _, cmd := range commands {
			t.Run(cmd.name, func(t *testing.T) {
				start := time.Now()
				result := env.RunGuild(cmd.cmd...)
				duration := time.Since(start)

				result.AssertSuccess(t)
				assert.Less(t, duration, cmd.maxTime,
					"Command %s took too long: %v (max: %v)",
					strings.Join(cmd.cmd, " "), duration, cmd.maxTime)

				t.Logf("Command '%s' took: %v", strings.Join(cmd.cmd, " "), duration)
			})
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	// Test that multiple guild commands can run concurrently without issues
	const numConcurrent = 5

	results := make(chan *CommandResult, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			env := NewTestEnvironment(t)
			// Each goroutine runs in its own environment
			result := env.RunGuild("version")
			results <- result
		}(i)
	}

	// Collect results
	for i := 0; i < numConcurrent; i++ {
		select {
		case result := <-results:
			result.AssertSuccess(t)
			result.AssertContains(t, "Guild")
			result.AssertFasterThan(t, 10*time.Second)
		case <-time.After(30 * time.Second):
			t.Fatalf("Concurrent operation %d timed out", i)
		}
	}
}

func TestScalabilityLimits(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Large Command Line Args", func(t *testing.T) {
		// Test with very long arguments
		longString := strings.Repeat("a", 1000)

		result := env.RunGuild("init", longString)
		// Should either succeed or fail gracefully
		if result.ExitCode != 0 {
			result.AssertStderrContains(t, "invalid")
		}
		result.AssertNotContains(t, "panic")
	})

	t.Run("Many Quick Commands", func(t *testing.T) {
		// Run many quick commands in sequence
		const numCommands = 20

		start := time.Now()
		for i := 0; i < numCommands; i++ {
			result := env.RunGuild("version")
			result.AssertSuccess(t)
		}
		totalTime := time.Since(start)

		avgTime := totalTime / numCommands
		assert.Less(t, avgTime, 1*time.Second,
			"Average command time degraded: %v", avgTime)

		t.Logf("Executed %d commands in %v (avg: %v)",
			numCommands, totalTime, avgTime)
	})
}

func TestResourceConstraints(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Low Memory Simulation", func(t *testing.T) {
		// We can't actually limit memory, but we can test behavior
		// under simulated constraints by running many operations

		var results []*CommandResult

		// Run commands until we see consistent performance
		for i := 0; i < 10; i++ {
			result := env.RunGuild("help")
			result.AssertSuccess(t)
			results = append(results, result)
		}

		// Check that performance doesn't degrade significantly
		firstTime := results[0].Duration
		lastTime := results[len(results)-1].Duration

		// Last command shouldn't be more than 3x slower than first
		if firstTime > 0 {
			ratio := float64(lastTime) / float64(firstTime)
			assert.Less(t, ratio, 3.0,
				"Performance degraded from %v to %v", firstTime, lastTime)
		}
	})

	t.Run("File System Pressure", func(t *testing.T) {
		// Create many files and then run commands
		for i := 0; i < 100; i++ {
			env.CreateFile(
				strings.Repeat("x", 10)+"_file_"+string(rune(i))+".txt",
				"test content")
		}

		// Commands should still work with many files present
		result := env.RunGuild("help")
		result.AssertSuccess(t)
		result.AssertFasterThan(t, 10*time.Second)
	})
}

func BenchmarkCommands(b *testing.B) {
	// Only run benchmarks if explicitly requested
	if testing.Short() {
		b.Skip("Skipping benchmarks in short mode")
	}

	env := NewTestEnvironment(&testing.T{})

	commands := []struct {
		name string
		args []string
	}{
		{"version", []string{"version"}},
		{"help", []string{"help"}},
		{"help_detailed", []string{"help", "init"}},
	}

	for _, cmd := range commands {
		b.Run(cmd.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := env.RunGuild(cmd.args...)
				if result.ExitCode != 0 {
					b.Fatalf("Command failed: %v", result.Error)
				}
			}
		})
	}
}

func BenchmarkProjectOperations(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmarks in short mode")
	}

	env := NewTestEnvironment(&testing.T{})

	// Initialize project once
	result := env.RunGuild("init")
	if result.ExitCode != 0 {
		b.Fatalf("Failed to initialize project: %v", result.Error)
	}

	operations := []struct {
		name string
		args []string
	}{
		{"status", []string{"status"}},
		{"config_show", []string{"config", "show"}},
		{"agent_list", []string{"agent", "list"}},
	}

	for _, op := range operations {
		b.Run(op.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := env.RunGuild(op.args...)
				if result.ExitCode != 0 {
					b.Fatalf("Operation failed: %v", result.Error)
				}
			}
		})
	}
}

func TestPerformanceRegression(t *testing.T) {
	// Test to catch performance regressions
	env := NewTestEnvironment(t)

	// Baseline performance targets (these should be updated as the system evolves)
	performanceTargets := map[string]time.Duration{
		"version":    1 * time.Second,
		"help":       2 * time.Second,
		"init":       5 * time.Second,
		"status":     3 * time.Second,
		"agent_list": 3 * time.Second,
	}

	// Test each target
	for cmdName, maxDuration := range performanceTargets {
		t.Run(cmdName+"_performance", func(t *testing.T) {
			var result *CommandResult

			switch cmdName {
			case "version":
				result = env.RunGuild("version")
			case "help":
				result = env.RunGuild("help")
			case "init":
				result = env.RunGuild("init")
			case "status":
				env.RunGuild("init").AssertSuccess(t)
				result = env.RunGuild("status")
			case "agent_list":
				env.RunGuild("init").AssertSuccess(t)
				result = env.RunGuild("agent", "list")
			}

			result.AssertSuccess(t)
			result.AssertFasterThan(t, maxDuration)

			t.Logf("Command '%s' completed in %v (target: %v)",
				cmdName, result.Duration, maxDuration)
		})
	}
}

func TestMemoryLeaks(t *testing.T) {
	env := NewTestEnvironment(t)

	// Test for potential memory leaks by running many operations
	var initialMem, finalMem runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&initialMem)

	// Run many operations
	for i := 0; i < 50; i++ {
		env.RunGuild("version").AssertSuccess(t)
		env.RunGuild("help").AssertSuccess(t)

		// Force GC every 10 iterations
		if i%10 == 0 {
			runtime.GC()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&finalMem)

	memoryIncrease := finalMem.Alloc - initialMem.Alloc

	// Memory increase should be minimal (less than 10MB)
	maxIncrease := uint64(10 * 1024 * 1024)
	assert.Less(t, memoryIncrease, maxIncrease,
		"Potential memory leak detected: %d KB increase",
		memoryIncrease/1024)

	t.Logf("Memory increase after 100 operations: %d KB",
		memoryIncrease/1024)
}
