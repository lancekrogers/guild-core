// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PlainTextFormatter implements Formatter for plain text export
type PlainTextFormatter struct{}

// NewPlainTextFormatter creates a new plain text formatter
func NewPlainTextFormatter() *PlainTextFormatter {
	return &PlainTextFormatter{}
}

// Format exports content as plain text
func (f *PlainTextFormatter) Format(ctx context.Context, content ExportContent) ([]byte, error) {
	var builder strings.Builder

	// Write header
	f.writeHeader(&builder, content.Metadata, content.Options)

	// Write messages
	messages := f.getSelectedMessages(content)
	for i, msg := range messages {
		f.writeMessage(&builder, msg, i, content.Options)
	}

	// Write footer if metadata enabled
	if content.Options.IncludeMetadata {
		f.writeFooter(&builder, content.Metadata)
	}

	return []byte(builder.String()), nil
}

// GetMimeType returns the MIME type for plain text
func (f *PlainTextFormatter) GetMimeType() string {
	return "text/plain"
}

// GetFileExtension returns the file extension for plain text
func (f *PlainTextFormatter) GetFileExtension() string {
	return "txt"
}

// GetOptions returns available options for plain text export
func (f *PlainTextFormatter) GetOptions() []ExportOption {
	return []ExportOption{
		{
			Key:         "include_metadata",
			Name:        "Include Metadata",
			Description: "Include export metadata in the document",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
		{
			Key:         "include_timestamps",
			Name:        "Include Timestamps",
			Description: "Include message timestamps",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
		{
			Key:         "line_width",
			Name:        "Line Width",
			Description: "Maximum characters per line (0 = no limit)",
			Type:        "number",
			Default:     80,
			Required:    false,
		},
		{
			Key:         "separator_style",
			Name:        "Separator Style",
			Description: "Style for message separators",
			Type:        "select",
			Default:     "dashes",
			Options:     []string{"dashes", "equals", "dots", "stars", "none"},
			Required:    false,
		},
	}
}

// ValidateOptions validates plain text-specific options
func (f *PlainTextFormatter) ValidateOptions(options ExportOptions) error {
	return nil
}

// getSelectedMessages returns the messages based on selection
func (f *PlainTextFormatter) getSelectedMessages(content ExportContent) []ChatMessage {
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

// writeHeader writes the document header
func (f *PlainTextFormatter) writeHeader(builder *strings.Builder, metadata ExportMetadata, options ExportOptions) {
	lineWidth := f.getLineWidth(options)

	// Title
	builder.WriteString(f.centerText(metadata.Title, lineWidth))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("=", min(len(metadata.Title), lineWidth)))
	builder.WriteString("\n\n")

	// Description
	if metadata.Description != "" {
		builder.WriteString(f.wrapText(metadata.Description, lineWidth))
		builder.WriteString("\n\n")
	}

	// Metadata
	if options.IncludeMetadata {
		builder.WriteString("EXPORT INFORMATION\n")
		builder.WriteString(strings.Repeat("-", 18))
		builder.WriteString("\n")

		if metadata.Author != "" {
			builder.WriteString(fmt.Sprintf("Author: %s\n", metadata.Author))
		}
		if metadata.Campaign != "" {
			builder.WriteString(fmt.Sprintf("Campaign: %s\n", metadata.Campaign))
		}
		builder.WriteString(fmt.Sprintf("Exported: %s\n", metadata.ExportedAt.Format("2006-01-02 15:04:05 MST")))
		if metadata.Version != "" {
			builder.WriteString(fmt.Sprintf("Version: %s\n", metadata.Version))
		}
		if len(metadata.Tags) > 0 {
			builder.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(metadata.Tags, ", ")))
		}

		builder.WriteString("\n")
		builder.WriteString(strings.Repeat("=", lineWidth))
		builder.WriteString("\n\n")
	}
}

// writeMessage writes a single message
func (f *PlainTextFormatter) writeMessage(builder *strings.Builder, msg ChatMessage, index int, options ExportOptions) {
	lineWidth := f.getLineWidth(options)

	// Message separator (except for first message)
	if index > 0 {
		separator := f.getSeparator(options, lineWidth)
		builder.WriteString(separator)
		builder.WriteString("\n\n")
	}

	// Role header
	roleText := f.formatRole(msg.Role)
	builder.WriteString(roleText)
	builder.WriteString("\n")

	// Timestamp if enabled
	if options.IncludeTimestamps {
		builder.WriteString(fmt.Sprintf("Time: %s\n", msg.Timestamp.Format("2006-01-02 15:04:05 MST")))
	}

	builder.WriteString("\n")

	// Message content
	content := f.wrapText(msg.Content, lineWidth)
	builder.WriteString(content)
	builder.WriteString("\n\n")
}

// writeFooter writes the document footer
func (f *PlainTextFormatter) writeFooter(builder *strings.Builder, metadata ExportMetadata) {
	lineWidth := f.getLineWidth(ExportOptions{})

	builder.WriteString(strings.Repeat("=", lineWidth))
	builder.WriteString("\n")
	builder.WriteString(f.centerText("Generated by Guild Framework", lineWidth))
	builder.WriteString("\n")
	builder.WriteString(f.centerText(time.Now().Format("2006-01-02 15:04:05 MST"), lineWidth))
	builder.WriteString("\n")
}

// Helper methods

func (f *PlainTextFormatter) getLineWidth(options ExportOptions) int {
	if options.FormatSpecific == nil {
		return 80
	}

	if width, exists := options.FormatSpecific["line_width"]; exists {
		if w, ok := width.(float64); ok && w > 0 {
			return int(w)
		}
		if w, ok := width.(int); ok && w > 0 {
			return w
		}
	}

	return 80
}

func (f *PlainTextFormatter) getSeparator(options ExportOptions, width int) string {
	style := "dashes"

	if options.FormatSpecific != nil {
		if s, exists := options.FormatSpecific["separator_style"]; exists {
			if str, ok := s.(string); ok {
				style = str
			}
		}
	}

	char := "-"
	switch style {
	case "equals":
		char = "="
	case "dots":
		char = "."
	case "stars":
		char = "*"
	case "none":
		return ""
	}

	return strings.Repeat(char, width)
}

func (f *PlainTextFormatter) formatRole(role string) string {
	switch strings.ToLower(role) {
	case "user":
		return "USER"
	case "assistant":
		return "ASSISTANT"
	case "system":
		return "SYSTEM"
	default:
		return strings.ToUpper(role)
	}
}

func (f *PlainTextFormatter) centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}

	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}

func (f *PlainTextFormatter) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var result strings.Builder
	currentLine := ""

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			result.WriteString(currentLine + "\n")
			currentLine = word
		}
	}

	if currentLine != "" {
		result.WriteString(currentLine)
	}

	return result.String()
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
