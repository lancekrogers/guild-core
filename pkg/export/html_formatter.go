// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package export

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"
)

// HTMLFormatter implements Formatter for HTML export
type HTMLFormatter struct{}

// NewHTMLFormatter creates a new HTML formatter
func NewHTMLFormatter() *HTMLFormatter {
	return &HTMLFormatter{}
}

// Format exports content as HTML
func (f *HTMLFormatter) Format(ctx context.Context, content ExportContent) ([]byte, error) {
	var builder strings.Builder
	
	// Write HTML document structure
	f.writeHTMLHeader(&builder, content.Metadata)
	f.writeStyles(&builder, content.Options)
	builder.WriteString("</head><body>\n")
	
	// Write content
	f.writeContentHeader(&builder, content.Metadata)
	f.writeMessages(&builder, content)
	f.writeFooter(&builder, content.Metadata)
	
	builder.WriteString("</body></html>")
	
	return []byte(builder.String()), nil
}

// GetMimeType returns the MIME type for HTML
func (f *HTMLFormatter) GetMimeType() string {
	return "text/html"
}

// GetFileExtension returns the file extension for HTML
func (f *HTMLFormatter) GetFileExtension() string {
	return "html"
}

// GetOptions returns available options for HTML export
func (f *HTMLFormatter) GetOptions() []ExportOption {
	return []ExportOption{
		{
			Key:         "theme",
			Name:        "Theme",
			Description: "Color theme for the HTML document",
			Type:        "select",
			Default:     "guild",
			Options:     []string{"guild", "light", "dark", "minimal"},
			Required:    false,
		},
		{
			Key:         "include_timestamps",
			Name:        "Include Timestamps",
			Description: "Show message timestamps",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
		{
			Key:         "syntax_highlighting",
			Name:        "Syntax Highlighting",
			Description: "Enable syntax highlighting for code blocks",
			Type:        "boolean",
			Default:     true,
			Required:    false,
		},
	}
}

// ValidateOptions validates HTML-specific options
func (f *HTMLFormatter) ValidateOptions(options ExportOptions) error {
	return nil
}

// writeHTMLHeader writes the HTML document header
func (f *HTMLFormatter) writeHTMLHeader(builder *strings.Builder, metadata ExportMetadata) {
	builder.WriteString("<!DOCTYPE html>\n")
	builder.WriteString("<html lang=\"en\">\n<head>\n")
	builder.WriteString("<meta charset=\"UTF-8\">\n")
	builder.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	builder.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(metadata.Title)))
	
	if metadata.Description != "" {
		builder.WriteString(fmt.Sprintf("<meta name=\"description\" content=\"%s\">\n", html.EscapeString(metadata.Description)))
	}
	if metadata.Author != "" {
		builder.WriteString(fmt.Sprintf("<meta name=\"author\" content=\"%s\">\n", html.EscapeString(metadata.Author)))
	}
}

// writeStyles writes CSS styles
func (f *HTMLFormatter) writeStyles(builder *strings.Builder, options ExportOptions) {
	theme := f.getTheme(options)
	
	builder.WriteString("<style>\n")
	builder.WriteString(f.getCSS(theme))
	builder.WriteString("</style>\n")
}

// writeContentHeader writes the content header
func (f *HTMLFormatter) writeContentHeader(builder *strings.Builder, metadata ExportMetadata) {
	builder.WriteString("<div class=\"container\">\n")
	builder.WriteString(fmt.Sprintf("<h1>%s</h1>\n", html.EscapeString(metadata.Title)))
	
	if metadata.Description != "" {
		builder.WriteString(fmt.Sprintf("<p class=\"description\">%s</p>\n", html.EscapeString(metadata.Description)))
	}
	
	// Metadata info
	builder.WriteString("<div class=\"metadata\">\n")
	if metadata.Campaign != "" {
		builder.WriteString(fmt.Sprintf("<span class=\"campaign\">Campaign: %s</span>\n", html.EscapeString(metadata.Campaign)))
	}
	if metadata.Author != "" {
		builder.WriteString(fmt.Sprintf("<span class=\"author\">Author: %s</span>\n", html.EscapeString(metadata.Author)))
	}
	builder.WriteString(fmt.Sprintf("<span class=\"exported\">Exported: %s</span>\n", metadata.ExportedAt.Format("2006-01-02 15:04:05 MST")))
	builder.WriteString("</div>\n")
}

// writeMessages writes all messages
func (f *HTMLFormatter) writeMessages(builder *strings.Builder, content ExportContent) {
	messages := f.getSelectedMessages(content)
	
	builder.WriteString("<div class=\"messages\">\n")
	for _, msg := range messages {
		f.writeMessage(builder, msg, content.Options)
	}
	builder.WriteString("</div>\n")
}

// writeMessage writes a single message
func (f *HTMLFormatter) writeMessage(builder *strings.Builder, msg ChatMessage, options ExportOptions) {
	roleClass := f.getRoleClass(msg.Role)
	
	builder.WriteString(fmt.Sprintf("<div class=\"message %s\">\n", roleClass))
	
	// Message header
	builder.WriteString("<div class=\"message-header\">\n")
	builder.WriteString(fmt.Sprintf("<span class=\"role\">%s</span>\n", f.formatRole(msg.Role)))
	
	if options.IncludeTimestamps {
		builder.WriteString(fmt.Sprintf("<span class=\"timestamp\">%s</span>\n", msg.Timestamp.Format("2006-01-02 15:04:05")))
	}
	
	builder.WriteString("</div>\n")
	
	// Message content
	builder.WriteString("<div class=\"message-content\">\n")
	content := f.formatContent(msg.Content)
	builder.WriteString(content)
	builder.WriteString("</div>\n")
	
	builder.WriteString("</div>\n")
}

// writeFooter writes the document footer
func (f *HTMLFormatter) writeFooter(builder *strings.Builder, metadata ExportMetadata) {
	builder.WriteString("<div class=\"footer\">\n")
	builder.WriteString("<p>Generated by Guild Framework</p>\n")
	builder.WriteString(fmt.Sprintf("<p>%s</p>\n", time.Now().Format("2006-01-02 15:04:05 MST")))
	builder.WriteString("</div>\n")
	builder.WriteString("</div>\n") // Close container
}

// Helper methods

func (f *HTMLFormatter) getSelectedMessages(content ExportContent) []ChatMessage {
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

func (f *HTMLFormatter) getTheme(options ExportOptions) string {
	if options.FormatSpecific == nil {
		return "guild"
	}
	
	if theme, exists := options.FormatSpecific["theme"]; exists {
		if t, ok := theme.(string); ok {
			return t
		}
	}
	
	return "guild"
}

func (f *HTMLFormatter) getRoleClass(role string) string {
	return fmt.Sprintf("role-%s", strings.ToLower(role))
}

func (f *HTMLFormatter) formatRole(role string) string {
	switch strings.ToLower(role) {
	case "user":
		return "👤 User"
	case "assistant":
		return "🤖 Assistant"
	case "system":
		return "⚙️ System"
	default:
		return fmt.Sprintf("📝 %s", strings.Title(role))
	}
}

func (f *HTMLFormatter) formatContent(content string) string {
	// Basic HTML escaping
	escaped := html.EscapeString(content)
	
	// Convert newlines to <br>
	escaped = strings.ReplaceAll(escaped, "\n", "<br>\n")
	
	// Simple code block detection and formatting
	if strings.Contains(escaped, "```") {
		escaped = strings.ReplaceAll(escaped, "```", "<pre><code>")
		escaped = strings.ReplaceAll(escaped, "</code>", "</code></pre>")
	}
	
	return escaped
}

func (f *HTMLFormatter) getCSS(theme string) string {
	base := `
body {
	font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
	line-height: 1.6;
	margin: 0;
	padding: 20px;
}

.container {
	max-width: 800px;
	margin: 0 auto;
}

h1 {
	border-bottom: 2px solid #eee;
	padding-bottom: 10px;
}

.metadata {
	margin: 20px 0;
	padding: 10px;
	background: #f8f9fa;
	border-radius: 5px;
}

.metadata span {
	margin-right: 20px;
	font-size: 0.9em;
}

.messages {
	margin-top: 30px;
}

.message {
	margin-bottom: 20px;
	border: 1px solid #ddd;
	border-radius: 8px;
	overflow: hidden;
}

.message-header {
	padding: 10px 15px;
	background: #f1f3f4;
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.role {
	font-weight: bold;
}

.timestamp {
	font-size: 0.8em;
	color: #666;
}

.message-content {
	padding: 15px;
}

.role-user .message-header {
	background: #e3f2fd;
}

.role-assistant .message-header {
	background: #f3e5f5;
}

.role-system .message-header {
	background: #fff3e0;
}

pre {
	background: #f5f5f5;
	padding: 10px;
	border-radius: 4px;
	overflow-x: auto;
}

.footer {
	margin-top: 40px;
	padding-top: 20px;
	border-top: 1px solid #eee;
	text-align: center;
	color: #666;
	font-size: 0.9em;
}
`
	
	switch theme {
	case "dark":
		return base + `
body { background: #1a1a1a; color: #e0e0e0; }
.metadata { background: #2d2d2d; color: #e0e0e0; }
.message { border-color: #444; }
.message-header { background: #333; color: #e0e0e0; }
.role-user .message-header { background: #1e3a8a; }
.role-assistant .message-header { background: #7c2d12; }
.role-system .message-header { background: #a16207; }
pre { background: #2d2d2d; color: #e0e0e0; }
`
	case "minimal":
		return `
body { font-family: Georgia, serif; max-width: 600px; margin: 0 auto; padding: 20px; }
.message { border: none; margin-bottom: 30px; }
.message-header { background: none; padding: 0 0 5px 0; border-bottom: 1px solid #ccc; }
.message-content { padding: 10px 0; }
pre { border-left: 3px solid #ccc; padding-left: 15px; background: none; }
`
	default:
		return base
	}
}