// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// manager implements SessionManager with high-level operations
type manager struct {
	store SessionStore
	mu    sync.RWMutex

	// Active message streams
	streams map[string]*messageStream
}

// NewManager creates a new session manager
func NewManager(store SessionStore) SessionManager {
	return &manager{
		store:   store,
		streams: make(map[string]*messageStream),
	}
}

// NewSession creates a new chat session
func (m *manager) NewSession(name string, campaignID *string) (*Session, error) {
	ctx := context.Background()

	session := &Session{
		ID:         uuid.New().String(),
		Name:       name,
		CampaignID: campaignID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"version":    "1.0",
			"created_by": "guild-chat",
		},
	}

	if err := m.store.CreateSession(ctx, session); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create new session")
	}

	return session, nil
}

// LoadSession loads an existing session
func (m *manager) LoadSession(id string) (*Session, error) {
	ctx := context.Background()

	session, err := m.store.GetSession(ctx, id)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load session")
	}

	return session, nil
}

// SaveSession updates an existing session
func (m *manager) SaveSession(session *Session) error {
	ctx := context.Background()

	session.UpdatedAt = time.Now()
	if err := m.store.UpdateSession(ctx, session); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save session")
	}

	return nil
}

// ForkSession creates a new session branching from an existing one
func (m *manager) ForkSession(sourceID string, newName string) (*Session, error) {
	ctx := context.Background()

	// Load source session
	source, err := m.store.GetSession(ctx, sourceID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load source session")
	}

	// Create new session
	forked := &Session{
		ID:         uuid.New().String(),
		Name:       newName,
		CampaignID: source.CampaignID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"version":     "1.0",
			"created_by":  "guild-chat",
			"forked_from": sourceID,
			"forked_at":   time.Now().Format(time.RFC3339),
		},
	}

	if err := m.store.CreateSession(ctx, forked); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create forked session")
	}

	// Copy messages from source
	messages, err := m.store.GetMessages(ctx, sourceID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load source messages")
	}

	for _, msg := range messages {
		// Create new message with new ID
		forkedMsg := &Message{
			ID:        uuid.New().String(),
			SessionID: forked.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
			ToolCalls: msg.ToolCalls,
			Metadata:  msg.Metadata,
		}

		if err := m.store.SaveMessage(ctx, forkedMsg); err != nil {
			// Log error but continue copying
			continue
		}
	}

	return forked, nil
}

// AppendMessage adds a new message to a session
func (m *manager) AppendMessage(sessionID string, role MessageRole, content string, toolCalls []ToolCall) (*Message, error) {
	ctx := context.Background()

	message := &Message{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
		ToolCalls: toolCalls,
	}

	if err := m.store.SaveMessage(ctx, message); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to append message")
	}

	return message, nil
}

// StreamMessage creates a message stream for progressive content
func (m *manager) StreamMessage(sessionID string, role MessageRole) (MessageStream, error) {
	streamID := uuid.New().String()

	stream := &messageStream{
		id:        streamID,
		sessionID: sessionID,
		role:      role,
		content:   &bytes.Buffer{},
		manager:   m,
		createdAt: time.Now(),
	}

	m.mu.Lock()
	m.streams[streamID] = stream
	m.mu.Unlock()

	return stream, nil
}

// GetContext retrieves recent messages for context
func (m *manager) GetContext(sessionID string, messageCount int) ([]*Message, error) {
	ctx := context.Background()

	// Get all messages (they're already ordered by created_at ASC)
	messages, err := m.store.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get context messages")
	}

	// Return last N messages
	if len(messages) > messageCount {
		return messages[len(messages)-messageCount:], nil
	}

	return messages, nil
}

// ClearContext removes all messages from a session
func (m *manager) ClearContext(sessionID string) error {
	ctx := context.Background()

	// Get all messages
	messages, err := m.store.GetMessages(ctx, sessionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get messages for clearing")
	}

	// Delete each message
	for _, msg := range messages {
		if err := m.store.DeleteMessage(ctx, msg.ID); err != nil {
			// Log error but continue
			continue
		}
	}

	return nil
}

// ExportSession exports a session in the specified format
func (m *manager) ExportSession(sessionID string, format ExportFormat) ([]byte, error) {
	// Use default options
	defaultOptions := &ExportOptions{
		IncludeToolOutputs: true,
		IncludeMetadata:    false,
		SyntaxHighlight:    true,
		LineNumbers:        false,
		DateFormat:         "2006-01-02 15:04:05",
		Theme:              "default",
	}
	return m.ExportSessionWithOptions(sessionID, format, defaultOptions)
}

// ExportSessionWithOptions exports a session with custom options
func (m *manager) ExportSessionWithOptions(sessionID string, format ExportFormat, options *ExportOptions) ([]byte, error) {
	ctx := context.Background()

	// Load session
	session, err := m.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load session for export")
	}

	// Load messages
	messages, err := m.store.GetMessages(ctx, sessionID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load messages for export")
	}

	// Apply default options if not provided
	if options == nil {
		options = &ExportOptions{
			IncludeToolOutputs: true,
			IncludeMetadata:    false,
			SyntaxHighlight:    true,
			LineNumbers:        false,
			DateFormat:         "2006-01-02 15:04:05",
			Theme:              "default",
		}
	}

	switch format {
	case ExportFormatJSON:
		return m.exportJSONWithOptions(session, messages, options)
	case ExportFormatMarkdown:
		return m.exportMarkdownWithOptions(session, messages, options)
	case ExportFormatHTML:
		return m.exportHTMLWithOptions(session, messages, options)
	case ExportFormatPDF:
		return m.exportPDF(session, messages, options)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unsupported export format: %s", format), nil)
	}
}

// ImportSession imports a session from exported data
func (m *manager) ImportSession(data []byte, format ExportFormat) (*Session, error) {
	switch format {
	case ExportFormatJSON:
		return m.importJSON(data)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unsupported import format: %s", format), nil)
	}
}

// messageStream implements MessageStream for streaming content
type messageStream struct {
	id        string
	sessionID string
	role      MessageRole
	content   *bytes.Buffer
	toolCalls []ToolCall
	manager   *manager
	createdAt time.Time
	mu        sync.Mutex
	closed    bool
}

// Write appends content to the stream
func (s *messageStream) Write(chunk string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return gerror.New(gerror.ErrCodeInvalidInput, "stream is closed", nil)
	}

	s.content.WriteString(chunk)
	return nil
}

// SetToolCalls sets the tool calls for the message
func (s *messageStream) SetToolCalls(toolCalls []ToolCall) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return gerror.New(gerror.ErrCodeInvalidInput, "stream is closed", nil)
	}

	s.toolCalls = toolCalls
	return nil
}

// Close finalizes the stream and saves the message
func (s *messageStream) Close() (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "stream already closed", nil)
	}

	s.closed = true

	// Remove from active streams
	s.manager.mu.Lock()
	delete(s.manager.streams, s.id)
	s.manager.mu.Unlock()

	// Create and save message
	message := &Message{
		ID:        uuid.New().String(),
		SessionID: s.sessionID,
		Role:      s.role,
		Content:   s.content.String(),
		CreatedAt: s.createdAt,
		ToolCalls: s.toolCalls,
	}

	ctx := context.Background()
	if err := s.manager.store.SaveMessage(ctx, message); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save streamed message")
	}

	return message, nil
}

// Export helpers

func (m *manager) exportJSON(session *Session, messages []*Message) ([]byte, error) {
	export := map[string]interface{}{
		"session":     session,
		"messages":    messages,
		"exported_at": time.Now().Format(time.RFC3339),
		"version":     "1.0",
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON export")
	}

	return data, nil
}

func (m *manager) exportJSONWithOptions(session *Session, messages []*Message, options *ExportOptions) ([]byte, error) {
	// Filter messages based on options
	filteredMessages := m.filterMessages(messages, options)

	export := map[string]interface{}{
		"session":     session,
		"messages":    filteredMessages,
		"exported_at": time.Now().Format(time.RFC3339),
		"version":     "2.0",
		"options":     options,
	}

	if options.IncludeMetadata {
		export["metadata"] = map[string]interface{}{
			"total_messages":    len(messages),
			"filtered_messages": len(filteredMessages),
			"export_settings":   options,
		}
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal JSON export")
	}

	return data, nil
}

func (m *manager) exportMarkdown(session *Session, messages []*Message) ([]byte, error) {
	defaultOptions := &ExportOptions{
		IncludeToolOutputs: true,
		IncludeMetadata:    false,
		SyntaxHighlight:    true,
		LineNumbers:        false,
		DateFormat:         "2006-01-02 15:04:05",
	}
	return m.exportMarkdownWithOptions(session, messages, defaultOptions)
}

func (m *manager) exportMarkdownWithOptions(session *Session, messages []*Message, options *ExportOptions) ([]byte, error) {
	var buf bytes.Buffer

	// Custom title or default
	title := session.Name
	if options.Title != "" {
		title = options.Title
	}

	// Header
	fmt.Fprintf(&buf, "# %s\n\n", title)
	fmt.Fprintf(&buf, "**Session ID:** %s  \n", session.ID)
	fmt.Fprintf(&buf, "**Created:** %s  \n", session.CreatedAt.Format(options.DateFormat))
	fmt.Fprintf(&buf, "**Updated:** %s  \n", session.UpdatedAt.Format(options.DateFormat))

	if session.CampaignID != nil {
		fmt.Fprintf(&buf, "**Campaign:** %s  \n", *session.CampaignID)
	}

	if options.IncludeMetadata && len(session.Metadata) > 0 {
		fmt.Fprintf(&buf, "\n**Metadata:**\n")
		for k, v := range session.Metadata {
			fmt.Fprintf(&buf, "- **%s:** %v\n", k, v)
		}
	}

	fmt.Fprintf(&buf, "\n---\n\n")

	// Filter messages
	filteredMessages := m.filterMessages(messages, options)

	// Messages
	for i, msg := range filteredMessages {
		roleTitle := strings.Title(string(msg.Role))
		if msg.Role == RoleAssistant {
			roleTitle = "🤖 Assistant"
		} else if msg.Role == RoleUser {
			roleTitle = "👤 User"
		} else if msg.Role == RoleSystem {
			roleTitle = "⚙️ System"
		} else if msg.Role == RoleTool {
			roleTitle = "🔧 Tool"
		}

		fmt.Fprintf(&buf, "## %s\n", roleTitle)
		fmt.Fprintf(&buf, "*%s*\n\n", msg.CreatedAt.Format(options.DateFormat))

		// Process content with syntax highlighting
		content := msg.Content
		if options.SyntaxHighlight {
			content = m.enhanceMarkdownCodeBlocks(content, options.LineNumbers)
		}

		fmt.Fprintf(&buf, "%s\n\n", content)

		// Tool calls
		if len(msg.ToolCalls) > 0 && options.IncludeToolOutputs {
			fmt.Fprintf(&buf, "**🔧 Tool Calls:**\n")
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&buf, "- **%s**", tc.Function.Name)
				if len(tc.Function.Arguments) > 0 {
					args, _ := json.MarshalIndent(tc.Function.Arguments, "  ", "  ")
					fmt.Fprintf(&buf, "\n  ```json\n  %s\n  ```", string(args))
				}
				if tc.Result != nil {
					if tc.Result.Error != nil {
						fmt.Fprintf(&buf, "\n  **Error:** %s", *tc.Result.Error)
					} else {
						fmt.Fprintf(&buf, "\n  **Result:**\n  ```\n  %s\n  ```", tc.Result.Content)
					}
				}
				fmt.Fprintf(&buf, "\n")
			}
			fmt.Fprintf(&buf, "\n")
		}

		// Metadata
		if options.IncludeMetadata && len(msg.Metadata) > 0 {
			fmt.Fprintf(&buf, "**Metadata:**\n")
			for k, v := range msg.Metadata {
				fmt.Fprintf(&buf, "- **%s:** %v\n", k, v)
			}
			fmt.Fprintf(&buf, "\n")
		}

		if i < len(filteredMessages)-1 {
			fmt.Fprintf(&buf, "---\n\n")
		}
	}

	// Footer metadata
	if options.IncludeMetadata {
		fmt.Fprintf(&buf, "\n---\n")
		fmt.Fprintf(&buf, "\n*Exported on %s*  \n", time.Now().Format(options.DateFormat))
		fmt.Fprintf(&buf, "*Total Messages: %d | Included: %d*\n", len(messages), len(filteredMessages))
	}

	return buf.Bytes(), nil
}

func (m *manager) exportHTML(session *Session, messages []*Message) ([]byte, error) {
	defaultOptions := &ExportOptions{
		IncludeToolOutputs: true,
		IncludeMetadata:    false,
		SyntaxHighlight:    true,
		LineNumbers:        false,
		DateFormat:         "2006-01-02 15:04:05",
		Theme:              "default",
	}
	return m.exportHTMLWithOptions(session, messages, defaultOptions)
}

func (m *manager) exportHTMLWithOptions(session *Session, messages []*Message, options *ExportOptions) ([]byte, error) {
	var buf bytes.Buffer

	// Custom title or default
	title := session.Name
	if options.Title != "" {
		title = options.Title
	}

	// HTML header with enhanced styling
	fmt.Fprintf(&buf, `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
    <style>
        %s
    </style>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.8.0/styles/%s.min.css">
    <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.8.0/highlight.min.js"></script>
    <script>hljs.highlightAll();</script>
</head>
<body>
`, title, m.getHTMLStyles(options), m.getThemeName(options.Theme))

	// Session info
	fmt.Fprintf(&buf, "<div class='header'>\n")
	fmt.Fprintf(&buf, "<h1>%s</h1>\n", title)
	fmt.Fprintf(&buf, "<div class='session-info'>\n")
	fmt.Fprintf(&buf, "<p><strong>Session ID:</strong> %s<br>\n", session.ID)
	fmt.Fprintf(&buf, "<strong>Created:</strong> %s<br>\n", session.CreatedAt.Format(options.DateFormat))
	fmt.Fprintf(&buf, "<strong>Updated:</strong> %s</p>\n", session.UpdatedAt.Format(options.DateFormat))

	if session.CampaignID != nil {
		fmt.Fprintf(&buf, "<p><strong>Campaign:</strong> %s</p>\n", *session.CampaignID)
	}

	if options.IncludeMetadata && len(session.Metadata) > 0 {
		fmt.Fprintf(&buf, "<div class='metadata'>\n<strong>Metadata:</strong>\n<ul>\n")
		for k, v := range session.Metadata {
			fmt.Fprintf(&buf, "<li><strong>%s:</strong> %v</li>\n", k, v)
		}
		fmt.Fprintf(&buf, "</ul>\n</div>\n")
	}

	fmt.Fprintf(&buf, "</div>\n")
	fmt.Fprintf(&buf, "</div>\n")

	// Filter messages
	filteredMessages := m.filterMessages(messages, options)

	// Messages
	fmt.Fprintf(&buf, "<div class='messages'>\n")
	for _, msg := range filteredMessages {
		roleClass := string(msg.Role)
		roleIcon := m.getRoleIcon(msg.Role)

		fmt.Fprintf(&buf, `<div class="message %s">`, roleClass)
		fmt.Fprintf(&buf, `<div class="message-header">`)
		fmt.Fprintf(&buf, `<span class="role">%s %s</span>`, roleIcon, strings.Title(string(msg.Role)))
		fmt.Fprintf(&buf, `<span class="timestamp">%s</span>`, msg.CreatedAt.Format(options.DateFormat))
		fmt.Fprintf(&buf, `</div>`)

		// Content with syntax highlighting
		content := msg.Content
		if options.SyntaxHighlight {
			content = m.enhanceHTMLCodeBlocks(content, options.LineNumbers)
		} else {
			content = strings.ReplaceAll(content, "\n", "<br>")
		}

		fmt.Fprintf(&buf, `<div class="content">%s</div>`, content)

		// Tool calls
		if len(msg.ToolCalls) > 0 && options.IncludeToolOutputs {
			fmt.Fprintf(&buf, `<div class="tool-calls">`)
			fmt.Fprintf(&buf, `<strong>🔧 Tool Calls:</strong>`)
			for _, tc := range msg.ToolCalls {
				fmt.Fprintf(&buf, `<div class="tool-call">`)
				fmt.Fprintf(&buf, `<strong>%s</strong>`, tc.Function.Name)
				if len(tc.Function.Arguments) > 0 {
					args, _ := json.MarshalIndent(tc.Function.Arguments, "", "  ")
					fmt.Fprintf(&buf, `<pre><code class="language-json">%s</code></pre>`, string(args))
				}
				if tc.Result != nil {
					if tc.Result.Error != nil {
						fmt.Fprintf(&buf, `<div class="error">Error: %s</div>`, *tc.Result.Error)
					} else {
						fmt.Fprintf(&buf, `<div class="result"><strong>Result:</strong><pre><code>%s</code></pre></div>`, tc.Result.Content)
					}
				}
				fmt.Fprintf(&buf, `</div>`)
			}
			fmt.Fprintf(&buf, `</div>`)
		}

		// Metadata
		if options.IncludeMetadata && len(msg.Metadata) > 0 {
			fmt.Fprintf(&buf, `<div class="message-metadata">`)
			fmt.Fprintf(&buf, `<strong>Metadata:</strong>`)
			fmt.Fprintf(&buf, `<ul>`)
			for k, v := range msg.Metadata {
				fmt.Fprintf(&buf, `<li><strong>%s:</strong> %v</li>`, k, v)
			}
			fmt.Fprintf(&buf, `</ul>`)
			fmt.Fprintf(&buf, `</div>`)
		}

		fmt.Fprintf(&buf, "</div>\n")
	}
	fmt.Fprintf(&buf, "</div>\n")

	// Footer
	if options.IncludeMetadata {
		fmt.Fprintf(&buf, "<div class='footer'>\n")
		fmt.Fprintf(&buf, "<hr>\n")
		fmt.Fprintf(&buf, "<p><em>Exported on %s</em><br>\n", time.Now().Format(options.DateFormat))
		fmt.Fprintf(&buf, "<em>Total Messages: %d | Included: %d</em></p>\n", len(messages), len(filteredMessages))
		fmt.Fprintf(&buf, "</div>\n")
	}

	// HTML footer
	fmt.Fprintf(&buf, "</body>\n</html>\n")

	return buf.Bytes(), nil
}

// Import helpers

func (m *manager) importJSON(data []byte) (*Session, error) {
	var export struct {
		Session  *Session   `json:"session"`
		Messages []*Message `json:"messages"`
		Version  string     `json:"version"`
	}

	if err := json.Unmarshal(data, &export); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to unmarshal JSON import")
	}

	if export.Version != "1.0" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, fmt.Sprintf("unsupported import version: %s", export.Version), nil)
	}

	ctx := context.Background()

	// Create new session with new ID
	newSession := &Session{
		ID:         uuid.New().String(),
		Name:       fmt.Sprintf("%s (imported)", export.Session.Name),
		CampaignID: export.Session.CampaignID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata: map[string]interface{}{
			"imported_from":    export.Session.ID,
			"imported_at":      time.Now().Format(time.RFC3339),
			"original_created": export.Session.CreatedAt.Format(time.RFC3339),
		},
	}

	if err := m.store.CreateSession(ctx, newSession); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create imported session")
	}

	// Import messages
	for _, msg := range export.Messages {
		newMsg := &Message{
			ID:        uuid.New().String(),
			SessionID: newSession.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt,
			ToolCalls: msg.ToolCalls,
			Metadata:  msg.Metadata,
		}

		if err := m.store.SaveMessage(ctx, newMsg); err != nil {
			// Log error but continue importing
			continue
		}
	}

	return newSession, nil
}

// PDF export using pandoc
func (m *manager) exportPDF(session *Session, messages []*Message, options *ExportOptions) ([]byte, error) {
	// Check if pandoc is available
	if _, err := exec.LookPath("pandoc"); err != nil {
		return nil, gerror.New(gerror.ErrCodeExternal, "pandoc is required for PDF export but not found in PATH", nil)
	}

	// Generate markdown first
	markdownData, err := m.exportMarkdownWithOptions(session, messages, options)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to generate markdown for PDF export")
	}

	// Create temporary files
	tempDir, err := os.MkdirTemp("", "guild-export-*")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create temp directory")
	}
	defer os.RemoveAll(tempDir)

	markdownFile := filepath.Join(tempDir, "session.md")
	pdfFile := filepath.Join(tempDir, "session.pdf")

	// Write markdown to file
	if err := os.WriteFile(markdownFile, markdownData, 0o644); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write markdown file")
	}

	// Prepare pandoc command
	args := []string{
		markdownFile,
		"-o", pdfFile,
		"--pdf-engine=xelatex",
		"--highlight-style=github",
		"-V", "geometry:margin=1in",
		"-V", "fontsize=11pt",
		"-V", "mainfont=DejaVu Sans",
		"-V", "monofont=DejaVu Sans Mono",
	}

	// Add custom CSS if provided
	if options.CustomCSS != "" {
		cssFile := filepath.Join(tempDir, "style.css")
		if err := os.WriteFile(cssFile, []byte(options.CustomCSS), 0o644); err == nil {
			args = append(args, "--css", cssFile)
		}
	}

	// Execute pandoc
	cmd := exec.Command("pandoc", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, gerror.New(gerror.ErrCodeExternal, fmt.Sprintf("pandoc failed: %s", string(output)), err)
	}

	// Read generated PDF
	pdfData, err := os.ReadFile(pdfFile)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read generated PDF")
	}

	return pdfData, nil
}

// Helper functions

// filterMessages filters messages based on export options
func (m *manager) filterMessages(messages []*Message, options *ExportOptions) []*Message {
	filtered := make([]*Message, 0, len(messages))

	for _, msg := range messages {
		// Skip tool messages if not including tool outputs
		if msg.Role == RoleTool && !options.IncludeToolOutputs {
			continue
		}

		filtered = append(filtered, msg)
	}

	return filtered
}

// enhanceMarkdownCodeBlocks adds syntax highlighting hints to markdown code blocks
func (m *manager) enhanceMarkdownCodeBlocks(content string, lineNumbers bool) string {
	// Enhanced regex to capture code blocks with optional language
	codeBlockRegex := regexp.MustCompile("(?s)```(\\w*)?\\n?(.*?)```")

	return codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract language and code
		parts := codeBlockRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		lang := parts[1]
		code := parts[2]

		// Default language detection
		if lang == "" {
			lang = m.detectLanguage(code)
		}

		// Add line numbers if requested
		if lineNumbers {
			lines := strings.Split(code, "\n")
			numberedLines := make([]string, len(lines))
			for i, line := range lines {
				numberedLines[i] = fmt.Sprintf("%3d | %s", i+1, line)
			}
			code = strings.Join(numberedLines, "\n")
		}

		return fmt.Sprintf("```%s\n%s\n```", lang, code)
	})
}

// enhanceHTMLCodeBlocks adds syntax highlighting to HTML code blocks
func (m *manager) enhanceHTMLCodeBlocks(content string, lineNumbers bool) string {
	// Convert markdown to HTML with syntax highlighting
	codeBlockRegex := regexp.MustCompile("(?s)```(\\w*)?\\n?(.*?)```")

	content = codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := codeBlockRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		lang := parts[1]
		code := parts[2]

		if lang == "" {
			lang = m.detectLanguage(code)
		}

		// Add line numbers class if requested
		lineNumClass := ""
		if lineNumbers {
			lineNumClass = " line-numbers"
		}

		return fmt.Sprintf(`<pre><code class="language-%s%s">%s</code></pre>`, lang, lineNumClass, code)
	})

	// Convert other markdown elements
	content = strings.ReplaceAll(content, "\n", "<br>")

	return content
}

// detectLanguage performs simple language detection based on content
func (m *manager) detectLanguage(code string) string {
	code = strings.TrimSpace(strings.ToLower(code))

	// Simple heuristics for language detection
	if strings.Contains(code, "package main") || strings.Contains(code, "func ") {
		return "go"
	}
	if strings.Contains(code, "def ") || strings.Contains(code, "import ") {
		return "python"
	}
	if strings.Contains(code, "function ") || strings.Contains(code, "const ") || strings.Contains(code, "let ") {
		return "javascript"
	}
	if strings.Contains(code, "class ") && strings.Contains(code, "public ") {
		return "java"
	}
	if strings.Contains(code, "{") && strings.Contains(code, "}") {
		return "json"
	}
	if strings.Contains(code, "<") && strings.Contains(code, ">") {
		return "xml"
	}
	if strings.Contains(code, "SELECT ") || strings.Contains(code, "select ") {
		return "sql"
	}

	return "text"
}

// getRoleIcon returns an emoji icon for message roles
func (m *manager) getRoleIcon(role MessageRole) string {
	switch role {
	case RoleUser:
		return "👤"
	case RoleAssistant:
		return "🤖"
	case RoleSystem:
		return "⚙️"
	case RoleTool:
		return "🔧"
	default:
		return "💬"
	}
}

// getThemeName converts theme name to highlight.js theme
func (m *manager) getThemeName(theme string) string {
	switch theme {
	case "dark":
		return "github-dark"
	case "monokai":
		return "monokai"
	case "solarized":
		return "solarized-light"
	default:
		return "github"
	}
}

// getHTMLStyles returns CSS styles based on options
func (m *manager) getHTMLStyles(options *ExportOptions) string {
	// Default styles
	baseStyles := `
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
            max-width: 900px; 
            margin: 0 auto; 
            padding: 20px; 
            line-height: 1.6;
            color: #333;
        }
        .header { margin-bottom: 30px; }
        .header h1 { color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px; }
        .session-info { background: #f8f9fa; padding: 15px; border-radius: 8px; margin: 15px 0; }
        .messages { margin-top: 20px; }
        .message { 
            margin-bottom: 25px; 
            padding: 20px; 
            border-radius: 12px; 
            border: 1px solid #e1e5e9;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .message.user { background: linear-gradient(135deg, #e3f2fd 0%, #bbdefb 100%); border-color: #2196f3; }
        .message.assistant { background: linear-gradient(135deg, #f5f5f5 0%, #eeeeee 100%); border-color: #757575; }
        .message.system { background: linear-gradient(135deg, #fff3e0 0%, #ffcc80 100%); border-color: #ff9800; }
        .message.tool { background: linear-gradient(135deg, #e8f5e9 0%, #c8e6c9 100%); border-color: #4caf50; }
        .message-header { 
            display: flex; 
            justify-content: space-between; 
            align-items: center; 
            margin-bottom: 12px;
            padding-bottom: 8px;
            border-bottom: 1px solid rgba(0,0,0,0.1);
        }
        .role { font-weight: bold; font-size: 1.1em; }
        .timestamp { font-size: 0.9em; color: #666; font-style: italic; }
        .content { 
            margin-top: 12px; 
            white-space: pre-wrap; 
            font-size: 14px;
        }
        .tool-calls { 
            margin-top: 15px; 
            padding: 12px; 
            background: rgba(255,255,255,0.7); 
            border-radius: 6px;
            border: 1px solid rgba(0,0,0,0.1);
        }
        .tool-call { margin: 8px 0; }
        .error { color: #d32f2f; font-weight: bold; }
        .result { margin-top: 8px; }
        .metadata, .message-metadata { 
            margin-top: 12px; 
            font-size: 0.9em; 
            color: #666; 
            background: rgba(255,255,255,0.5);
            padding: 8px;
            border-radius: 4px;
        }
        .footer { 
            margin-top: 40px; 
            text-align: center; 
            color: #666; 
            font-size: 0.9em;
        }
        pre { 
            background: #f4f4f4; 
            padding: 12px; 
            border-radius: 6px; 
            overflow-x: auto; 
            border: 1px solid #ddd;
        }
        code { 
            background: #f4f4f4; 
            padding: 2px 4px; 
            border-radius: 3px; 
            font-family: 'Monaco', 'Consolas', monospace;
        }
        pre code { background: none; padding: 0; }
    `

	// Merge with custom CSS if provided
	if options.CustomCSS != "" {
		return baseStyles + "\n" + options.CustomCSS
	}

	return baseStyles
}
