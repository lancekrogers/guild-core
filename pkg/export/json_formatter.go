// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"encoding/json"
)

// JSONFormatter implements Formatter for JSON export
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

// Format exports content as JSON
func (f *JSONFormatter) Format(ctx context.Context, content ExportContent) ([]byte, error) {
	// Create export structure
	export := struct {
		Metadata ExportMetadata `json:"metadata"`
		Messages []ChatMessage  `json:"messages"`
		Stats    ExportStats    `json:"stats"`
	}{
		Metadata: content.Metadata,
		Messages: f.getSelectedMessages(content),
	}

	// Calculate stats
	export.Stats = f.calculateStats(export.Messages)

	// Marshal with indentation for readability
	if f.shouldPrettyPrint(content.Options) {
		return json.MarshalIndent(export, "", "  ")
	}

	return json.Marshal(export)
}

// ExportStats provides statistics about the export
type ExportStats struct {
	MessageCount         int            `json:"message_count"`
	UserMessages         int            `json:"user_messages"`
	AssistantMessages    int            `json:"assistant_messages"`
	SystemMessages       int            `json:"system_messages"`
	TotalCharacters      int            `json:"total_characters"`
	AverageMessageLength float64        `json:"average_message_length"`
	RoleCounts           map[string]int `json:"role_counts"`
}

// GetMimeType returns the MIME type for JSON
func (f *JSONFormatter) GetMimeType() string {
	return "application/json"
}

// GetFileExtension returns the file extension for JSON
func (f *JSONFormatter) GetFileExtension() string {
	return "json"
}

// GetOptions returns available options for JSON export
func (f *JSONFormatter) GetOptions() []ExportOption {
	return []ExportOption{
		{
			Key:         "pretty_print",
			Name:        "Pretty Print",
			Description: "Format JSON with indentation for readability",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
		{
			Key:         "include_stats",
			Name:        "Include Statistics",
			Description: "Include message statistics in the export",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
		{
			Key:         "include_metadata",
			Name:        "Include Metadata",
			Description: "Include message metadata fields",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
		{
			Key:         "compact_messages",
			Name:        "Compact Messages",
			Description: "Remove empty or null fields from messages",
			Type:        "boolean",
			Default:     false,
			Required:    false,
		},
	}
}

// ValidateOptions validates JSON-specific options
func (f *JSONFormatter) ValidateOptions(options ExportOptions) error {
	// JSON is very flexible - no strict validation needed
	return nil
}

// getSelectedMessages returns the messages based on selection
func (f *JSONFormatter) getSelectedMessages(content ExportContent) []ChatMessage {
	if content.Selection == nil {
		return content.Messages
	}

	if len(content.Selection.MessageIDs) > 0 {
		// Select by message IDs
		selected := make([]ChatMessage, 0)
		idSet := make(map[string]bool)
		for _, id := range content.Selection.MessageIDs {
			idSet[id] = true
		}

		for _, msg := range content.Messages {
			if idSet[msg.ID] {
				selected = append(selected, msg)
			}
		}
		return selected
	}

	// Select by range
	start := content.Selection.StartIndex
	end := content.Selection.EndIndex
	if start < 0 {
		start = 0
	}
	if end >= len(content.Messages) {
		end = len(content.Messages) - 1
	}

	return content.Messages[start : end+1]
}

// shouldPrettyPrint determines if JSON should be formatted with indentation
func (f *JSONFormatter) shouldPrettyPrint(options ExportOptions) bool {
	if options.FormatSpecific == nil {
		return true // Default to pretty print
	}

	prettyPrint, exists := options.FormatSpecific["pretty_print"]
	if !exists {
		return true
	}

	if b, ok := prettyPrint.(bool); ok {
		return b
	}

	return true
}

// calculateStats calculates statistics about the messages
func (f *JSONFormatter) calculateStats(messages []ChatMessage) ExportStats {
	stats := ExportStats{
		MessageCount: len(messages),
		RoleCounts:   make(map[string]int),
	}

	totalChars := 0

	for _, msg := range messages {
		// Count by role
		stats.RoleCounts[msg.Role]++

		// Count specific roles
		switch msg.Role {
		case "user":
			stats.UserMessages++
		case "assistant":
			stats.AssistantMessages++
		case "system":
			stats.SystemMessages++
		}

		// Character count
		totalChars += len(msg.Content)
	}

	stats.TotalCharacters = totalChars

	if len(messages) > 0 {
		stats.AverageMessageLength = float64(totalChars) / float64(len(messages))
	}

	return stats
}
