// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiFormatExporter_SupportedFormats(t *testing.T) {
	exporter := NewMultiFormatExporter()
	
	formats := exporter.SupportedFormats()
	assert.NotEmpty(t, formats)
	
	expectedFormats := []ExportFormat{
		FormatMarkdown,
		FormatHTML,
		FormatJSON,
		FormatPlainText,
		FormatCSV,
	}
	
	for _, expected := range expectedFormats {
		assert.Contains(t, formats, expected)
	}
}

func TestMultiFormatExporter_Export_AllFormats(t *testing.T) {
	exporter := NewMultiFormatExporter()
	ctx := context.Background()
	
	content := createTestContent()
	
	formats := exporter.SupportedFormats()
	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			data, err := exporter.Export(ctx, content, format)
			assert.NoError(t, err)
			assert.NotEmpty(t, data)
			
			// Basic content validation
			dataStr := string(data)
			
			// CSV format doesn't include title in content, only message data
			if format != FormatCSV {
				assert.Contains(t, dataStr, "Test Chat Session")
			}
			assert.Contains(t, dataStr, "Hello, how can I help?")
		})
	}
}

func TestMultiFormatExporter_ValidateContent(t *testing.T) {
	exporter := NewMultiFormatExporter()
	
	tests := []struct {
		name        string
		content     ExportContent
		expectError bool
	}{
		{
			name:        "valid content",
			content:     createTestContent(),
			expectError: false,
		},
		{
			name: "empty messages",
			content: ExportContent{
				Metadata: ExportMetadata{Title: "Test"},
				Messages: []ChatMessage{},
			},
			expectError: true,
		},
		{
			name: "no title",
			content: ExportContent{
				Metadata: ExportMetadata{},
				Messages: []ChatMessage{
					{ID: "1", Role: "user", Content: "test"},
				},
			},
			expectError: true,
		},
		{
			name: "invalid selection range",
			content: ExportContent{
				Metadata: ExportMetadata{Title: "Test"},
				Messages: []ChatMessage{
					{ID: "1", Role: "user", Content: "test"},
				},
				Selection: &ContentSelection{
					StartIndex: 5,
					EndIndex:   10,
				},
			},
			expectError: true,
		},
		{
			name: "empty message content",
			content: ExportContent{
				Metadata: ExportMetadata{Title: "Test"},
				Messages: []ChatMessage{
					{ID: "1", Role: "user", Content: ""},
				},
			},
			expectError: true,
		},
		{
			name: "empty message role",
			content: ExportContent{
				Metadata: ExportMetadata{Title: "Test"},
				Messages: []ChatMessage{
					{ID: "1", Role: "", Content: "test"},
				},
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exporter.ValidateContent(tt.content)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMultiFormatExporter_ExportToResult(t *testing.T) {
	exporter := NewMultiFormatExporter()
	ctx := context.Background()
	
	content := createTestContent()
	
	result, err := exporter.ExportToResult(ctx, content, FormatJSON)
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	assert.Equal(t, FormatJSON, result.Format)
	assert.Equal(t, "application/json", result.MimeType)
	assert.True(t, strings.HasSuffix(result.Filename, ".json"))
	assert.Greater(t, result.Size, int64(0))
	assert.NotZero(t, result.ExportedAt)
	
	// Check metadata
	assert.Contains(t, result.Metadata, "message_count")
	assert.Contains(t, result.Metadata, "title")
}

func TestMultiFormatExporter_UnsupportedFormat(t *testing.T) {
	exporter := NewMultiFormatExporter()
	ctx := context.Background()
	
	content := createTestContent()
	
	_, err := exporter.Export(ctx, content, "unsupported")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestMultiFormatExporter_GetFormatOptions(t *testing.T) {
	exporter := NewMultiFormatExporter()
	
	options := exporter.GetFormatOptions(FormatMarkdown)
	assert.NotEmpty(t, options)
	
	// Check that we have expected options
	optionKeys := make([]string, len(options))
	for i, opt := range options {
		optionKeys[i] = opt.Key
	}
	
	assert.Contains(t, optionKeys, "include_metadata")
	assert.Contains(t, optionKeys, "include_timestamps")
}

func TestMultiFormatExporter_MessageSelection(t *testing.T) {
	exporter := NewMultiFormatExporter()
	ctx := context.Background()
	
	content := createTestContent()
	
	// Test range selection
	content.Selection = &ContentSelection{
		StartIndex: 0,
		EndIndex:   0, // Only first message
	}
	
	data, err := exporter.Export(ctx, content, FormatJSON)
	require.NoError(t, err)
	
	// Should only contain first message
	dataStr := string(data)
	assert.Contains(t, dataStr, "Hello, how can I help?")
	assert.NotContains(t, dataStr, "I need help with my project")
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		title     string
		extension string
		expected  string
	}{
		{
			title:     "Test Chat Session",
			extension: "md",
			expected:  "Test_Chat_Session.md",
		},
		{
			title:     "Chat/Session\\With:Special*Characters",
			extension: "txt",
			expected:  "Chat-Session-With-Special-Characters.txt",
		},
		{
			title:     "",
			extension: "json",
			expected:  "export_", // Should start with export_ and contain timestamp
		},
		{
			title:     strings.Repeat("a", 100),
			extension: "html",
			expected:  strings.Repeat("a", 50) + ".html",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := generateFilename(tt.title, tt.extension)
			
			if tt.title == "" {
				assert.True(t, strings.HasPrefix(result, tt.expected))
				assert.True(t, strings.HasSuffix(result, "."+tt.extension))
			} else if len(tt.title) > 50 {
				assert.Equal(t, tt.expected, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Helper function to create test content
func createTestContent() ExportContent {
	return ExportContent{
		Metadata: ExportMetadata{
			Title:       "Test Chat Session",
			Description: "A test chat session for export testing",
			Author:      "Test User",
			Campaign:    "test-campaign",
			ExportedAt:  time.Now(),
			Version:     "1.0",
			Tags:        []string{"test", "export"},
		},
		Messages: []ChatMessage{
			{
				ID:        "msg1",
				Role:      "assistant",
				Content:   "Hello, how can I help?",
				Timestamp: time.Now().Add(-10 * time.Minute),
				Metadata:  map[string]interface{}{"model": "test-model"},
			},
			{
				ID:        "msg2",
				Role:      "user",
				Content:   "I need help with my project",
				Timestamp: time.Now().Add(-5 * time.Minute),
			},
			{
				ID:        "msg3",
				Role:      "assistant",
				Content:   "I'd be happy to help! What kind of project are you working on?",
				Timestamp: time.Now(),
			},
		},
		Options: ExportOptions{
			IncludeMetadata:   true,
			IncludeTimestamps: true,
		},
	}
}