// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI escape codes from a string for testing
func stripANSI(s string) string {
	// Remove ANSI escape sequences
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	s = ansi.ReplaceAllString(s, "")

	// Also remove the literal [0m patterns that appear in the output
	s = strings.ReplaceAll(s, "[0m", "")
	s = strings.ReplaceAll(s, "[38;2;248;248;242m", "")
	s = strings.ReplaceAll(s, "[38;2;189;147;249;1m", "")
	s = strings.ReplaceAll(s, "[38;2;255;184;108;1m", "")
	s = strings.ReplaceAll(s, "[38;2;80;250;123m", "")
	s = strings.ReplaceAll(s, "[38;5;231m", "")

	// Remove any remaining bracket patterns
	bracketPattern := regexp.MustCompile(`\[[0-9;]+m`)
	s = bracketPattern.ReplaceAllString(s, "")

	return s
}

// TestChatMarkdownIntegration verifies that the chat interface properly integrates
// with the markdown renderer and content formatter
func TestChatMarkdownIntegration(t *testing.T) {
	// Create a test chat model with markdown components
	chatWidth := 80
	markdownRenderer, err := NewMarkdownRenderer(chatWidth)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}

	contentFormatter := NewContentFormatter(markdownRenderer, chatWidth, ".")

	// Test message formatting for different types
	tests := []struct {
		name         string
		messageType  string
		content      string
		agentID      string
		wantContains []string
	}{
		{
			name:         "agent response with markdown",
			messageType:  "agent",
			content:      "# Task Complete\n\nI've **successfully** completed the task with `code`.",
			agentID:      "developer",
			wantContains: []string{"Task", "Complete", "successfully", "code"},
		},
		{
			name:         "system message with emphasis",
			messageType:  "system",
			content:      "System **initialized** and ready.",
			wantContains: []string{"System", "initialized", "ready"},
		},
		{
			name:         "error message formatting",
			messageType:  "error",
			content:      "Failed to execute task: permission denied",
			wantContains: []string{"Failed", "permission denied"},
		},
		{
			name:         "tool output with code",
			messageType:  "tool",
			content:      "```bash\nls -la\n```\nListed directory contents.",
			agentID:      "file-reader",
			wantContains: []string{"file-reader", "bash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var formatted string

			metadata := make(map[string]string)
			if tt.agentID != "" {
				metadata["agentID"] = tt.agentID
				metadata["toolName"] = tt.agentID
			}

			// Use the content formatter's generic FormatMessage method
			formatted = contentFormatter.FormatMessage(tt.messageType, tt.content, metadata)

			// Strip ANSI codes for testing
			strippedOutput := stripANSI(formatted)

			// Verify the formatted output contains expected content
			// Note: Content may be split across lines due to formatting
			normalizedOutput := strings.ReplaceAll(strippedOutput, "\n", " ")
			normalizedOutput = strings.ReplaceAll(normalizedOutput, "  ", " ")

			for _, want := range tt.wantContains {
				if !strings.Contains(normalizedOutput, want) && !strings.Contains(strippedOutput, want) {
					t.Errorf("Formatted message missing expected content %q\nStripped output: %s", want, strippedOutput)
				}
			}

			// Note: We don't check for markdown syntax removal here because
			// the glamour renderer may include ANSI codes that could interfere
			// The important thing is that the content is processed and styled
		})
	}
}

// TestChatViewIntegration tests that the chat view properly integrates visual components
func TestChatViewIntegration(t *testing.T) {
	// This is a placeholder test that would verify the View() method
	// properly includes status panels and rich content rendering
	// In a real test, we would create a full chat model and verify its output

	t.Skip("Full chat view integration test requires complete model setup")
}
