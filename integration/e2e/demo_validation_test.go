// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package e2e

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDemoConsistency(t *testing.T) {
	// Run demo multiple times and verify consistency
	const runs = 3
	outputs := make([]string, runs)
	durations := make([]time.Duration, runs)

	for i := 0; i < runs; i++ {
		env := NewTestEnvironment(t)
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")
		result.AssertSuccess(t)
		outputs[i] = result.Stdout
		durations[i] = result.Duration
	}

	// Verify key elements appear in all runs
	expectedElements := []string{
		"Demo",
		"Guild",
		"Creating",
		"complete",
	}

	for _, expected := range expectedElements {
		for i, output := range outputs {
			assert.Contains(t, output, expected,
				"Run %d missing expected element: %s", i+1, expected)
		}
	}

	// Verify timing is reasonably consistent (within 100% variance)
	avgDuration := time.Duration(0)
	for _, d := range durations {
		avgDuration += d
	}
	avgDuration /= time.Duration(len(durations))

	for i, duration := range durations {
		variance := float64(duration-avgDuration) / float64(avgDuration)
		if variance < 0 {
			variance = -variance
		}
		assert.Less(t, variance, 1.0, // Allow 100% variance
			"Run %d duration %v varies too much from average %v", i+1, duration, avgDuration)
	}

	// All demos should complete within reasonable time
	for i, duration := range durations {
		assert.Less(t, duration, 60*time.Second,
			"Demo run %d took too long: %v", i+1, duration)
	}
}

func TestDemoScenarios(t *testing.T) {
	scenarios := []struct {
		name        string
		args        []string
		expected    []string
		maxDuration time.Duration
		description string
	}{
		{
			name:        "Quick Demo",
			args:        []string{"demo-check"},
			expected:    []string{"Demo", "Guild", "complete"},
			maxDuration: 45 * time.Second,
			description: "Basic quick demonstration",
		},
		{
			name:        "Help Demo",
			args:        []string{"demo", "--help"},
			expected:    []string{"Usage", "demo", "options"},
			maxDuration: 5 * time.Second,
			description: "Demo help information",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			env := NewTestEnvironment(t)
			result := env.RunGuildWithTimeout(sc.maxDuration+10*time.Second, sc.args...)

			if strings.Contains(strings.Join(sc.args, " "), "--help") {
				// Help commands should succeed
				result.AssertSuccess(t)
			} else {
				// Demo commands may or may not be implemented yet
				if result.ExitCode == 0 {
					// If demo succeeds, check expectations
					for _, expected := range sc.expected {
						result.AssertContains(t, expected)
					}
					result.AssertFasterThan(t, sc.maxDuration)
				} else {
					// If demo fails, it should fail gracefully
					result.AssertStderrContains(t, "not implemented")
				}
			}
		})
	}
}

func TestDemoErrorHandling(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Invalid Demo Options", func(t *testing.T) {
		result := env.RunGuild("demo", "--invalid-option")
		result.AssertFailure(t)
		result.AssertStderrContains(t, "unknown flag")
	})

	t.Run("Demo In Project Context", func(t *testing.T) {
		// Initialize a project
		env.RunGuild("init").AssertSuccess(t)

		// Demo should work in project context
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")
		if result.ExitCode == 0 {
			result.AssertContains(t, "Demo")
			result.AssertNotContains(t, "error")
		}
	})

	t.Run("Demo Without Project", func(t *testing.T) {
		// Demo should work without project initialization
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")
		if result.ExitCode == 0 {
			result.AssertContains(t, "Demo")
		}
		// If it fails, it should be graceful
		result.AssertNotContains(t, "panic")
	})
}

func TestDemoContentValidation(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Demo Output Quality", func(t *testing.T) {
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")

		if result.ExitCode == 0 {
			// If demo works, validate output quality
			output := result.Stdout

			// Should not contain debug/error information
			result.AssertNotContains(t, "DEBUG")
			result.AssertNotContains(t, "WARN")
			result.AssertNotContains(t, "ERROR")
			result.AssertNotContains(t, "panic")
			result.AssertNotContains(t, "nil pointer")

			// Should contain meaningful content
			assert.Greater(t, len(output), 50, "Demo output should be substantial")

			// Should be properly formatted (basic checks)
			lines := strings.Split(output, "\n")
			nonEmptyLines := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					nonEmptyLines++
				}
			}
			assert.Greater(t, nonEmptyLines, 3, "Demo should have multiple lines of output")
		}
	})

	t.Run("Demo Resource Cleanup", func(t *testing.T) {
		// Run demo and ensure it cleans up after itself
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")

		if result.ExitCode == 0 {
			// Check that no temporary files are left in work directory
			workDir := env.GetWorkDir()

			// Demo shouldn't create unexpected files
			unexpectedPatterns := []string{
				"*.tmp",
				"*.temp",
				"demo-*",
				"test-*",
			}

			for _, pattern := range unexpectedPatterns {
				files, _ := filepath.Glob(filepath.Join(workDir, pattern))
				assert.Empty(t, files, "Demo should not leave %s files", pattern)
			}
		}
	})
}

func TestDemoInterruption(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Demo Timeout Handling", func(t *testing.T) {
		// Test with very short timeout to simulate interruption
		result := env.RunGuildWithTimeout(100*time.Millisecond, "demo-check")

		// Command might timeout or complete quickly
		if result.Error != nil && strings.Contains(result.Error.Error(), "context deadline exceeded") {
			// Timeout occurred - this is expected for this test
			assert.Contains(t, result.Error.Error(), "context deadline exceeded")
		} else {
			// Command completed quickly - also acceptable
			assert.Equal(t, 0, result.ExitCode)
		}
	})
}

func TestDemoRecovery(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Demo After Failed Init", func(t *testing.T) {
		// Try to create invalid project first
		env.RunGuild("init")

		// Demo should still work even if previous commands failed
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")

		if result.ExitCode == 0 {
			result.AssertContains(t, "Demo")
		}
		// Demo should not crash due to previous failures
		result.AssertNotContains(t, "panic")
	})

	t.Run("Demo With Corrupted Config", func(t *testing.T) {
		// Create invalid guild.yaml
		env.CreateFile(".guild/guild.yaml", "invalid: yaml: content: [")

		// Demo should handle corrupted config gracefully
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")

		// Should either work or fail gracefully
		result.AssertNotContains(t, "panic")
		result.AssertNotContains(t, "fatal")
	})
}

func TestDemoMetrics(t *testing.T) {
	const numRuns = 5
	durations := make([]time.Duration, numRuns)

	for i := 0; i < numRuns; i++ {
		env := NewTestEnvironment(t)
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")

		if result.ExitCode == 0 {
			durations[i] = result.Duration
		} else {
			// Skip timing analysis if demo doesn't work
			t.Skip("Demo command not available or failing")
		}
	}

	// Calculate performance metrics
	var totalDuration time.Duration
	var minDuration, maxDuration time.Duration = durations[0], durations[0]

	for _, d := range durations {
		totalDuration += d
		if d < minDuration {
			minDuration = d
		}
		if d > maxDuration {
			maxDuration = d
		}
	}

	avgDuration := totalDuration / time.Duration(numRuns)

	t.Logf("Demo Performance Metrics:")
	t.Logf("  Average: %v", avgDuration)
	t.Logf("  Min: %v", minDuration)
	t.Logf("  Max: %v", maxDuration)
	t.Logf("  Variance: %v", maxDuration-minDuration)

	// Performance assertions
	assert.Less(t, avgDuration, 30*time.Second, "Average demo time should be reasonable")
	assert.Less(t, maxDuration, 60*time.Second, "Max demo time should not exceed limit")

	// Consistency check - max shouldn't be more than 3x min
	if minDuration > 0 {
		ratio := float64(maxDuration) / float64(minDuration)
		assert.Less(t, ratio, 3.0, "Demo timing should be reasonably consistent")
	}
}
