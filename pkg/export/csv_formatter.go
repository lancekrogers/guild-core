// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
)

// CSVFormatter implements Formatter for CSV export
type CSVFormatter struct{}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter() *CSVFormatter {
	return &CSVFormatter{}
}

// Format exports content as CSV
func (f *CSVFormatter) Format(ctx context.Context, content ExportContent) ([]byte, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	
	// Write header row
	headers := f.getHeaders(content.Options)
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV headers: %w", err)
	}
	
	// Write data rows
	messages := f.getSelectedMessages(content)
	for _, msg := range messages {
		row := f.messageToRow(msg, content.Options)
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}
	
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}
	
	return []byte(builder.String()), nil
}

// GetMimeType returns the MIME type for CSV
func (f *CSVFormatter) GetMimeType() string {
	return "text/csv"
}

// GetFileExtension returns the file extension for CSV
func (f *CSVFormatter) GetFileExtension() string {
	return "csv"
}

// GetOptions returns available options for CSV export
func (f *CSVFormatter) GetOptions() []ExportOption {
	return []ExportOption{
		{
			Key:         "include_timestamps",
			Name:        "Include Timestamps",
			Description: "Include message timestamps as a column",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
		{
			Key:         "include_metadata",
			Name:        "Include Metadata",
			Description: "Include message metadata as additional columns",
			Type:        "boolean",
			Default:     false,
			Required:    false,
		},
		{
			Key:         "truncate_content",
			Name:        "Truncate Content",
			Description: "Maximum characters for content field (0 = no limit)",
			Type:        "number",
			Default:     0,
			Required:    false,
		},
		{
			Key:         "escape_newlines",
			Name:        "Escape Newlines",
			Description: "Replace newlines with \\n for single-line cells",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
	}
}

// ValidateOptions validates CSV-specific options
func (f *CSVFormatter) ValidateOptions(options ExportOptions) error {
	return nil
}

// getSelectedMessages returns the messages based on selection
func (f *CSVFormatter) getSelectedMessages(content ExportContent) []ChatMessage {
	if content.Selection == nil {
		return content.Messages
	}
	
	if len(content.Selection.MessageIDs) > 0 {
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

// getHeaders returns the CSV column headers based on options
func (f *CSVFormatter) getHeaders(options ExportOptions) []string {
	headers := []string{"ID", "Role", "Content"}
	
	if f.shouldIncludeTimestamps(options) {
		headers = append(headers, "Timestamp")
	}
	
	if f.shouldIncludeMetadata(options) {
		headers = append(headers, "Metadata")
	}
	
	return headers
}

// messageToRow converts a message to a CSV row
func (f *CSVFormatter) messageToRow(msg ChatMessage, options ExportOptions) []string {
	row := []string{
		msg.ID,
		msg.Role,
		f.formatContent(msg.Content, options),
	}
	
	if f.shouldIncludeTimestamps(options) {
		row = append(row, msg.Timestamp.Format("2006-01-02 15:04:05"))
	}
	
	if f.shouldIncludeMetadata(options) {
		metadata := f.formatMetadata(msg.Metadata)
		row = append(row, metadata)
	}
	
	return row
}

// formatContent formats message content for CSV
func (f *CSVFormatter) formatContent(content string, options ExportOptions) string {
	result := content
	
	// Escape newlines if requested
	if f.shouldEscapeNewlines(options) {
		result = strings.ReplaceAll(result, "\n", "\\n")
		result = strings.ReplaceAll(result, "\r", "\\r")
	}
	
	// Truncate if requested
	maxLength := f.getTruncateLength(options)
	if maxLength > 0 && len(result) > maxLength {
		result = result[:maxLength-3] + "..."
	}
	
	return result
}

// formatMetadata formats message metadata for CSV
func (f *CSVFormatter) formatMetadata(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return ""
	}
	
	parts := make([]string, 0, len(metadata))
	for key, value := range metadata {
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
	}
	
	return strings.Join(parts, ";")
}

// Helper methods for option checking

func (f *CSVFormatter) shouldIncludeTimestamps(options ExportOptions) bool {
	if options.FormatSpecific == nil {
		return true
	}
	
	if include, exists := options.FormatSpecific["include_timestamps"]; exists {
		if b, ok := include.(bool); ok {
			return b
		}
	}
	
	return options.IncludeTimestamps
}

func (f *CSVFormatter) shouldIncludeMetadata(options ExportOptions) bool {
	if options.FormatSpecific == nil {
		return false
	}
	
	if include, exists := options.FormatSpecific["include_metadata"]; exists {
		if b, ok := include.(bool); ok {
			return b
		}
	}
	
	return options.IncludeMetadata
}

func (f *CSVFormatter) shouldEscapeNewlines(options ExportOptions) bool {
	if options.FormatSpecific == nil {
		return true
	}
	
	if escape, exists := options.FormatSpecific["escape_newlines"]; exists {
		if b, ok := escape.(bool); ok {
			return b
		}
	}
	
	return true
}

func (f *CSVFormatter) getTruncateLength(options ExportOptions) int {
	if options.FormatSpecific == nil {
		return 0
	}
	
	if length, exists := options.FormatSpecific["truncate_content"]; exists {
		if l, ok := length.(float64); ok {
			return int(l)
		}
		if l, ok := length.(int); ok {
			return l
		}
	}
	
	return 0
}