// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/internal/chat/session"
)

// ExportFormat represents different export formats
type ExportFormat string

const (
	FormatMarkdown ExportFormat = "markdown"
	FormatHTML     ExportFormat = "html"
	FormatJSON     ExportFormat = "json"
	FormatText     ExportFormat = "text"
)

// SessionExporter provides simple session export functionality
type SessionExporter struct {
	session  *session.Session
	messages []*session.Message
}

// NewSessionExporter creates a new session exporter
func NewSessionExporter(sess *session.Session, msgs []*session.Message) *SessionExporter {
	return &SessionExporter{
		session:  sess,
		messages: msgs,
	}
}

// ExportToMarkdown exports the session to markdown format
func (e *SessionExporter) ExportToMarkdown() string {
	var buf strings.Builder
	
	// Header
	fmt.Fprintf(&buf, "# 🏰 Guild Chat Session: %s\n\n", e.session.Name)
	fmt.Fprintf(&buf, "**Date**: %s\n", e.session.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&buf, "**Messages**: %d\n\n", len(e.messages))
	
	// Messages
	for i, msg := range e.messages {
		// Message header
		roleIcon := e.getRoleIcon(string(msg.Role))
		roleName := e.getRoleName(string(msg.Role))
		
		fmt.Fprintf(&buf, "### %s %s\n\n", roleIcon, roleName)
		fmt.Fprintf(&buf, "%s\n\n", msg.Content)
		
		// Tool calls
		if len(msg.ToolCalls) > 0 {
			var toolNames []string
			for _, tc := range msg.ToolCalls {
				toolNames = append(toolNames, tc.Function.Name)
			}
			fmt.Fprintf(&buf, "**🔧 Tools**: %s\n\n", strings.Join(toolNames, ", "))
		}
		
		// Separator
		if i < len(e.messages)-1 {
			fmt.Fprintf(&buf, "---\n\n")
		}
	}
	
	// Footer
	fmt.Fprintf(&buf, "\n---\n*Exported from Guild Framework on %s*\n", 
		time.Now().Format("2006-01-02 15:04:05"))
	
	return buf.String()
}

// ExportToHTML exports the session to HTML format
func (e *SessionExporter) ExportToHTML() string {
	var buf strings.Builder
	
	fmt.Fprintf(&buf, `<!DOCTYPE html>
<html>
<head>
    <title>Guild Chat Session - %s</title>
    <style>
        body { font-family: -apple-system, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .message { margin: 20px 0; padding: 15px; border-radius: 8px; }
        .user-message { background: #e3f2fd; margin-left: 40px; }
        .assistant-message { background: #f5f5f5; margin-right: 40px; }
        .header { font-weight: bold; margin-bottom: 10px; }
    </style>
</head>
<body>
    <h1>🏰 %s</h1>
    <p><strong>Date:</strong> %s</p>
    <p><strong>Messages:</strong> %d</p>
`, e.session.Name, e.session.Name, e.session.CreatedAt.Format("2006-01-02 15:04:05"), len(e.messages))
	
	for _, msg := range e.messages {
		roleClass := fmt.Sprintf("%s-message", string(msg.Role))
		roleIcon := e.getRoleIcon(string(msg.Role))
		roleName := e.getRoleName(string(msg.Role))
		
		fmt.Fprintf(&buf, `    <div class="message %s">
        <div class="header">%s %s</div>
        <div>%s</div>
    </div>
`, roleClass, roleIcon, roleName, msg.Content)
	}
	
	fmt.Fprintf(&buf, `</body>
</html>`)
	
	return buf.String()
}

// getRoleIcon returns an icon for a role
func (e *SessionExporter) getRoleIcon(role string) string {
	switch role {
	case "user":
		return "👤"
	case "assistant":
		return "🤖"
	case "system":
		return "⚙️"
	case "tool":
		return "🔧"
	default:
		return "💬"
	}
}

// getRoleName returns a display name for a role
func (e *SessionExporter) getRoleName(role string) string {
	switch role {
	case "user":
		return "User"
	case "assistant":
		return "Assistant"
	case "system":
		return "System"
	case "tool":
		return "Tool"
	default:
		return strings.Title(role)
	}
}