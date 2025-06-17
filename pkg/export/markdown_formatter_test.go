// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownFormatter_Format(t *testing.T) {
	formatter := NewMarkdownFormatter()
	ctx := context.Background()
	
	content := createTestContent()
	
	data, err := formatter.Format(ctx, content)
	require.NoError(t, err)
	
	result := string(data)
	
	// Check document structure
	assert.Contains(t, result, "# Test Chat Session")
	assert.Contains(t, result, "## Export Information")
	assert.Contains(t, result, "## 🤖 Assistant")
	assert.Contains(t, result, "## 👤 User")
	
	// Check content
	assert.Contains(t, result, "Hello, how can I help?")
	assert.Contains(t, result, "I need help with my project")
	
	// Check metadata table
	assert.Contains(t, result, "| Author | Test User |")
	assert.Contains(t, result, "| Campaign | test-campaign |")
	
	// Check separators
	assert.Contains(t, result, "---")
}

func TestMarkdownFormatter_FormatWithoutMetadata(t *testing.T) {
	formatter := NewMarkdownFormatter()
	ctx := context.Background()
	
	content := createTestContent()
	content.Options.IncludeMetadata = false
	
	data, err := formatter.Format(ctx, content)
	require.NoError(t, err)
	
	result := string(data)
	
	// Should still have title and messages
	assert.Contains(t, result, "# Test Chat Session")
	assert.Contains(t, result, "Hello, how can I help?")
	
	// Should not have export information table
	assert.NotContains(t, result, "## Export Information")
	assert.NotContains(t, result, "| Author | Test User |")
}

func TestMarkdownFormatter_FormatWithoutTimestamps(t *testing.T) {
	formatter := NewMarkdownFormatter()
	ctx := context.Background()
	
	content := createTestContent()
	content.Options.IncludeTimestamps = false
	
	data, err := formatter.Format(ctx, content)
	require.NoError(t, err)
	
	result := string(data)
	
	// Should have content but no message timestamps
	assert.Contains(t, result, "Hello, how can I help?")
	
	// Should not contain message timestamp format patterns
	// Look for timestamps in the format "*YYYY-MM-DD HH:MM:SS*"
	messageTimestampPattern := "*" + time.Now().Format("2006-01-02")
	assert.NotContains(t, result, messageTimestampPattern, "Message timestamps should not be present")
}

func TestMarkdownFormatter_MessageSelection(t *testing.T) {
	formatter := NewMarkdownFormatter()
	ctx := context.Background()
	
	content := createTestContent()
	
	// Select only first message
	content.Selection = &ContentSelection{
		StartIndex: 0,
		EndIndex:   0,
	}
	
	data, err := formatter.Format(ctx, content)
	require.NoError(t, err)
	
	result := string(data)
	
	// Should contain first message
	assert.Contains(t, result, "Hello, how can I help?")
	
	// Should not contain other messages
	assert.NotContains(t, result, "I need help with my project")
	assert.NotContains(t, result, "I'd be happy to help!")
}

func TestMarkdownFormatter_MessageSelectionByIDs(t *testing.T) {
	formatter := NewMarkdownFormatter()
	ctx := context.Background()
	
	content := createTestContent()
	
	// Select specific messages by ID
	content.Selection = &ContentSelection{
		MessageIDs: []string{"msg1", "msg3"},
	}
	
	data, err := formatter.Format(ctx, content)
	require.NoError(t, err)
	
	result := string(data)
	
	// Should contain selected messages
	assert.Contains(t, result, "Hello, how can I help?")
	assert.Contains(t, result, "I'd be happy to help!")
	
	// Should not contain unselected message
	assert.NotContains(t, result, "I need help with my project")
}

func TestMarkdownFormatter_FormatRole(t *testing.T) {
	formatter := NewMarkdownFormatter()
	
	tests := []struct {
		role     string
		expected string
	}{
		{"user", "👤 User"},
		{"assistant", "🤖 Assistant"},
		{"system", "⚙️  System"},
		{"custom", "📝 Custom"},
	}
	
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			result := formatter.formatRole(tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMarkdownFormatter_FormatContent(t *testing.T) {
	formatter := NewMarkdownFormatter()
	options := ExportOptions{}
	
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple text",
			content:  "Hello world",
			expected: "Hello world",
		},
		{
			name:     "already formatted code",
			content:  "```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
			expected: "```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
		},
		{
			name:     "code-like content",
			content:  "func main() {\n    fmt.Println(\"hello\")\n}",
			expected: "```\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.formatContent(tt.content, options)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMarkdownFormatter_GetOptions(t *testing.T) {
	formatter := NewMarkdownFormatter()
	
	options := formatter.GetOptions()
	assert.NotEmpty(t, options)
	
	// Check for expected options
	optionKeys := make([]string, len(options))
	for i, opt := range options {
		optionKeys[i] = opt.Key
	}
	
	expectedKeys := []string{
		"include_metadata",
		"include_timestamps",
		"style",
		"code_fence_style",
	}
	
	for _, key := range expectedKeys {
		assert.Contains(t, optionKeys, key)
	}
}

func TestMarkdownFormatter_ValidateOptions(t *testing.T) {
	formatter := NewMarkdownFormatter()
	
	// Markdown is flexible, so validation should always pass
	options := ExportOptions{
		FormatSpecific: map[string]interface{}{
			"invalid_option": "invalid_value",
		},
	}
	
	err := formatter.ValidateOptions(options)
	assert.NoError(t, err)
}

func TestMarkdownFormatter_MimeTypeAndExtension(t *testing.T) {
	formatter := NewMarkdownFormatter()
	
	assert.Equal(t, "text/markdown", formatter.GetMimeType())
	assert.Equal(t, "md", formatter.GetFileExtension())
}