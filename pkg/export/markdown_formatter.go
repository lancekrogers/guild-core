// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MarkdownFormatter implements Formatter for Markdown export
type MarkdownFormatter struct{}

// NewMarkdownFormatter creates a new Markdown formatter
func NewMarkdownFormatter() *MarkdownFormatter {
	return &MarkdownFormatter{}
}

// Format exports content as Markdown
func (f *MarkdownFormatter) Format(ctx context.Context, content ExportContent) ([]byte, error) {
	var builder strings.Builder
	
	// Write header with metadata
	f.writeHeader(&builder, content.Metadata)
	
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

// GetMimeType returns the MIME type for Markdown
func (f *MarkdownFormatter) GetMimeType() string {
	return "text/markdown"
}

// GetFileExtension returns the file extension for Markdown
func (f *MarkdownFormatter) GetFileExtension() string {
	return "md"
}

// GetOptions returns available options for Markdown export
func (f *MarkdownFormatter) GetOptions() []ExportOption {
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
			Key:         "style",
			Name:        "Markdown Style",
			Description: "Choose Markdown formatting style",
			Type:        "select",
			Default:     "standard",
			Options:     []string{"standard", "github", "minimal"},
			Required:    false,
		},
		{
			Key:         "code_fence_style",
			Name:        "Code Fence Style",
			Description: "Style for code blocks",
			Type:        "select",
			Default:     "backticks",
			Options:     []string{"backticks", "indented"},
			Required:    false,
		},
	}
}

// ValidateOptions validates Markdown-specific options
func (f *MarkdownFormatter) ValidateOptions(options ExportOptions) error {
	// No strict validation needed for Markdown - it's very flexible
	return nil
}

// getSelectedMessages returns the messages based on selection
func (f *MarkdownFormatter) getSelectedMessages(content ExportContent) []ChatMessage {
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

// writeHeader writes the document header
func (f *MarkdownFormatter) writeHeader(builder *strings.Builder, metadata ExportMetadata) {
	builder.WriteString(fmt.Sprintf("# %s\n\n", metadata.Title))
	
	if metadata.Description != "" {
		builder.WriteString(fmt.Sprintf("%s\n\n", metadata.Description))
	}
	
	// Metadata table
	builder.WriteString("## Export Information\n\n")
	builder.WriteString("| Field | Value |\n")
	builder.WriteString("|-------|-------|\n")
	
	if metadata.Author != "" {
		builder.WriteString(fmt.Sprintf("| Author | %s |\n", metadata.Author))
	}
	if metadata.Campaign != "" {
		builder.WriteString(fmt.Sprintf("| Campaign | %s |\n", metadata.Campaign))
	}
	builder.WriteString(fmt.Sprintf("| Exported | %s |\n", metadata.ExportedAt.Format("2006-01-02 15:04:05 MST")))
	if metadata.Version != "" {
		builder.WriteString(fmt.Sprintf("| Version | %s |\n", metadata.Version))
	}
	
	if len(metadata.Tags) > 0 {
		builder.WriteString(fmt.Sprintf("| Tags | %s |\n", strings.Join(metadata.Tags, ", ")))
	}
	
	builder.WriteString("\n---\n\n")
}

// writeMessage writes a single message
func (f *MarkdownFormatter) writeMessage(builder *strings.Builder, msg ChatMessage, index int, options ExportOptions) {
	// Message header
	roleTitle := f.formatRole(msg.Role)
	builder.WriteString(fmt.Sprintf("## %s\n\n", roleTitle))
	
	// Timestamp if enabled
	if options.IncludeTimestamps {
		builder.WriteString(fmt.Sprintf("*%s*\n\n", msg.Timestamp.Format("2006-01-02 15:04:05 MST")))
	}
	
	// Message content
	content := f.formatContent(msg.Content, options)
	builder.WriteString(content)
	builder.WriteString("\n\n")
	
	// Separator between messages
	if index > 0 {
		builder.WriteString("---\n\n")
	}
}

// writeFooter writes the document footer
func (f *MarkdownFormatter) writeFooter(builder *strings.Builder, metadata ExportMetadata) {
	builder.WriteString("---\n\n")
	builder.WriteString("*This document was exported from Guild Framework*\n")
	builder.WriteString(fmt.Sprintf("*Generated on %s*\n", time.Now().Format("2006-01-02 15:04:05 MST")))
}

// formatRole formats the role name for display
func (f *MarkdownFormatter) formatRole(role string) string {
	switch strings.ToLower(role) {
	case "user":
		return "👤 User"
	case "assistant":
		return "🤖 Assistant"
	case "system":
		return "⚙️  System"
	default:
		return fmt.Sprintf("📝 %s", strings.Title(role))
	}
}

// formatContent formats message content with proper Markdown
func (f *MarkdownFormatter) formatContent(content string, options ExportOptions) string {
	// Basic content formatting
	result := content
	
	// Ensure code blocks are properly formatted
	if !strings.Contains(result, "```") && strings.Contains(result, "\n") {
		// Check if content looks like code (simple heuristic)
		lines := strings.Split(result, "\n")
		codeLines := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "{") || strings.Contains(line, "}") || 
			   strings.Contains(line, "func ") || strings.Contains(line, "import ") ||
			   strings.Contains(line, "def ") || strings.Contains(line, "class ") {
				codeLines++
			}
		}
		
		// If more than 30% of lines look like code, wrap in code block
		if codeLines > len(lines)/3 {
			result = fmt.Sprintf("```\n%s\n```", result)
		}
	}
	
	return result
}