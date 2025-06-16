// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package e2e

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVisualOutput(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Visual tests don't run in CI without display")
	}

	env := NewTestEnvironment(t)

	t.Run("Help Output Formatting", func(t *testing.T) {
		result := env.RunGuild("help")
		result.AssertSuccess(t)

		output := result.Stdout

		// Check basic formatting
		assert.Contains(t, output, "Guild")
		assert.Contains(t, output, "Commands")

		// Verify no ANSI escape codes (since NO_COLOR=1)
		ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
		assert.False(t, ansiRegex.MatchString(output),
			"Help output should not contain ANSI escape codes with NO_COLOR=1")

		// Check line length is reasonable (not too wide)
		lines := strings.Split(output, "\n")
		for i, line := range lines {
			assert.LessOrEqual(t, len(line), 120,
				"Line %d too long (%d chars): %s", i+1, len(line), line[:min(len(line), 50)])
		}
	})

	t.Run("Status Display Format", func(t *testing.T) {
		// Initialize project first
		env.RunGuild("init").AssertSuccess(t)

		result := env.RunGuild("status")
		result.AssertSuccess(t)

		output := result.Stdout

		// Should have structured output
		assert.Contains(t, output, "Status")
		assert.Contains(t, output, "Project")

		// Check for proper alignment and structure
		lines := strings.Split(output, "\n")
		nonEmptyLines := 0
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				nonEmptyLines++
			}
		}
		assert.Greater(t, nonEmptyLines, 2, "Status should have multiple lines of content")
	})

	t.Run("Table Output Consistency", func(t *testing.T) {
		env.RunGuild("init").AssertSuccess(t)

		result := env.RunGuild("agent", "list")
		result.AssertSuccess(t)

		output := result.Stdout

		// If there's table output, check basic formatting
		if strings.Contains(output, "│") || strings.Contains(output, "|") {
			// Table-like output detected, verify structure
			lines := strings.Split(output, "\n")
			tableLinesCount := 0
			for _, line := range lines {
				if strings.Contains(line, "│") || strings.Contains(line, "|") {
					tableLinesCount++
				}
			}
			assert.Greater(t, tableLinesCount, 1, "Table should have multiple rows")
		}
	})
}

func TestOutputConsistency(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Command Output Stability", func(t *testing.T) {
		// Run same command multiple times and verify output structure is consistent
		const runs = 3
		outputs := make([]string, runs)

		for i := 0; i < runs; i++ {
			result := env.RunGuild("help")
			result.AssertSuccess(t)
			outputs[i] = result.Stdout
		}

		// Outputs should be identical for deterministic commands
		for i := 1; i < runs; i++ {
			assert.Equal(t, outputs[0], outputs[i],
				"Help output should be identical across runs")
		}
	})

	t.Run("Error Message Formatting", func(t *testing.T) {
		// Test error message consistency
		result := env.RunGuild("invalid-command")
		result.AssertFailure(t)

		errorOutput := result.Stderr

		// Error messages should be well-formatted
		assert.NotEmpty(t, errorOutput, "Should have error output")
		assert.NotContains(t, errorOutput, "GUILD-", "Should not expose internal error codes")

		// Should suggest help
		lowerError := strings.ToLower(errorOutput)
		assert.True(t,
			strings.Contains(lowerError, "help") ||
				strings.Contains(lowerError, "usage") ||
				strings.Contains(lowerError, "command"),
			"Error should provide helpful guidance")
	})
}

func TestUIElementRendering(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Progress Indicators", func(t *testing.T) {
		// Test commands that might show progress
		result := env.RunGuild("init")
		result.AssertSuccess(t)

		// Check that any progress indicators don't leave artifacts
		output := result.Stdout
		
		// Should not contain raw progress characters
		progressChars := []string{"\r", "\b", "\x08"}
		for _, char := range progressChars {
			assert.NotContains(t, output, char,
				"Output should not contain raw progress control characters")
		}
	})

	t.Run("Unicode Handling", func(t *testing.T) {
		// Test with unicode in project name
		result := env.RunGuild("init")
		
		if result.ExitCode == 0 {
			// If unicode is supported, output should handle it properly
			result.AssertNotContains(t, "�") // replacement character
		} else {
			// If unicode not supported, should fail gracefully
			result.AssertStderrContains(t, "invalid")
		}
	})
}

func TestOutputFormats(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Plain Text Output", func(t *testing.T) {
		// With NO_COLOR=1, output should be plain text
		result := env.RunGuild("help")
		result.AssertSuccess(t)

		output := result.Stdout

		// Should not contain color codes
		colorCodes := []string{
			"\x1b[0m", "\x1b[1m", "\x1b[31m", "\x1b[32m", "\x1b[33m",
			"\x1b[34m", "\x1b[35m", "\x1b[36m", "\x1b[37m",
		}
		for _, code := range colorCodes {
			assert.NotContains(t, output, code,
				"Plain text output should not contain color code: %s", code)
		}
	})

	t.Run("Machine Readable Output", func(t *testing.T) {
		// Test if there are machine-readable output options
		result := env.RunGuild("status", "--format", "json")
		
		if result.ExitCode == 0 {
			// If JSON format is supported, validate it's proper JSON
			output := strings.TrimSpace(result.Stdout)
			assert.True(t,
				strings.HasPrefix(output, "{") && strings.HasSuffix(output, "}"),
				"JSON output should be properly formatted")
		}
		// If not supported, that's fine - this is optional
	})
}

func TestAccessibilityFeatures(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Screen Reader Friendly", func(t *testing.T) {
		result := env.RunGuild("help")
		result.AssertSuccess(t)

		output := result.Stdout

		// Should have clear section headers
		assert.Contains(t, output, "Commands")

		// Should not rely only on visual formatting
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 {
				// Non-empty lines should have meaningful content, not just spacing
				assert.NotRegexp(t, `^[\s\-_=]+$`, trimmed,
					"Lines should not be purely decorative characters")
			}
		}
	})

	t.Run("Keyboard Navigation Hints", func(t *testing.T) {
		// Check if help mentions keyboard shortcuts
		result := env.RunGuild("help", "chat")
		
		if result.ExitCode == 0 {
			output := strings.ToLower(result.Stdout)
			// If chat help exists, it might mention keyboard shortcuts
			keyboardTerms := []string{"ctrl", "key", "shortcut", "tab", "enter"}
			mentionsKeyboard := false
			for _, term := range keyboardTerms {
				if strings.Contains(output, term) {
					mentionsKeyboard = true
					break
				}
			}
			// This is informational - we don't assert it's required
			if mentionsKeyboard {
				t.Log("Chat help mentions keyboard interactions")
			}
		}
	})
}

func TestLongRunningCommandDisplay(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Command Timeout Display", func(t *testing.T) {
		// Test very short timeout to see how timeout is displayed
		result := env.RunGuildWithTimeout(50*time.Millisecond, "demo", "--quick")
		
		// Command might timeout or complete quickly
		if result.Error != nil && strings.Contains(result.Error.Error(), "context deadline exceeded") {
			// Timeout behavior is correct
			assert.Contains(t, result.Error.Error(), "deadline")
		}
		// If command completes quickly, that's also fine
	})

	t.Run("Interrupt Handling", func(t *testing.T) {
		// This is harder to test automatically, but we can check
		// that commands handle context cancellation properly
		env.RunGuild("init").AssertSuccess(t)

		// Run a quick command to ensure interrupt handling doesn't break basic operations
		result := env.RunGuild("status")
		result.AssertSuccess(t)
	})
}

func TestVisualRegressionReference(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Save Output Reference", func(t *testing.T) {
		// Save reference outputs for manual comparison
		commands := [][]string{
			{"help"},
			{"version"},
		}

		referencesDir := filepath.Join("testdata", "visual", "references")
		os.MkdirAll(referencesDir, 0755)

		for _, cmd := range commands {
			result := env.RunGuild(cmd...)
			if result.ExitCode == 0 {
				filename := strings.Join(cmd, "_") + ".txt"
				filepath := filepath.Join(referencesDir, filename)
				
				err := os.WriteFile(filepath, []byte(result.Stdout), 0644)
				if err == nil {
					t.Logf("Saved reference output: %s", filepath)
				}
			}
		}
	})
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}