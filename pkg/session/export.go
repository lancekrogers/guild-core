// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// SessionExporter handles session export functionality
type SessionExporter struct {
	serializer *SessionSerializer
	formatter  *ExportFormatter
	analytics  *SessionAnalytics
}

// ExportFormat defines the available export formats
type ExportFormat int

const (
	ExportFormatJSON ExportFormat = iota
	ExportFormatMarkdown
	ExportFormatHTML
	ExportFormatPDF
)

// String returns the string representation of an ExportFormat
func (ef ExportFormat) String() string {
	switch ef {
	case ExportFormatJSON:
		return "json"
	case ExportFormatMarkdown:
		return "markdown"
	case ExportFormatHTML:
		return "html"
	case ExportFormatPDF:
		return "pdf"
	default:
		return "unknown"
	}
}

// ExportOptions defines options for session export
type ExportOptions struct {
	Format           ExportFormat `json:"format"`
	IncludeMetadata  bool         `json:"include_metadata"`
	IncludeContext   bool         `json:"include_context"`
	IncludeAnalytics bool         `json:"include_analytics"`
	IncludeReasoning bool         `json:"include_reasoning"`
	DateRange        *DateRange   `json:"date_range,omitempty"`
	AgentFilter      []string     `json:"agent_filter,omitempty"`
	Theme            string       `json:"theme,omitempty"`
	CustomCSS        string       `json:"custom_css,omitempty"`
	SyntaxHighlight  bool         `json:"syntax_highlight"`
	LineNumbers      bool         `json:"line_numbers"`
	Title            string       `json:"title,omitempty"`
}

// DateRange defines a time range for filtering
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ExportData contains the data to be exported
type ExportData struct {
	Session          *Session               `json:"session"`
	Messages         []Message              `json:"messages"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Analytics        *AnalyticsData         `json:"analytics,omitempty"`
	ReasoningMetrics *ReasoningMetrics      `json:"reasoning_metrics,omitempty"`
	CSS              string                 `json:"css,omitempty"`
	CustomCSS        string                 `json:"custom_css,omitempty"`
}

// NewSessionExporter creates a new session exporter
func NewSessionExporter() *SessionExporter {
	return &SessionExporter{
		serializer: NewSessionSerializer(),
		formatter:  NewExportFormatter(),
		analytics:  nil, // Can be set later if analytics are needed
	}
}

// WithAnalytics sets the analytics provider for the exporter
func (se *SessionExporter) WithAnalytics(analytics *SessionAnalytics) *SessionExporter {
	se.analytics = analytics
	return se
}

// Export exports a session in the specified format
func (se *SessionExporter) Export(session *Session, opts ExportOptions) ([]byte, error) {
	// Filter messages if needed
	messages := se.filterMessages(session.Messages, opts)

	// Create export data
	exportData := &ExportData{
		Session:  session,
		Messages: messages,
		Metadata: se.buildMetadata(session, opts),
	}

	// Add analytics if requested
	if opts.IncludeAnalytics && se.analytics != nil {
		analytics, err := se.analytics.AnalyzeSession(context.Background(), session)
		if err == nil {
			exportData.Analytics = analytics
			if opts.IncludeReasoning && analytics.ReasoningMetrics != nil {
				exportData.ReasoningMetrics = analytics.ReasoningMetrics
			}
		}
	}

	// Format based on type
	switch opts.Format {
	case ExportFormatJSON:
		return se.exportJSON(exportData)
	case ExportFormatMarkdown:
		return se.exportMarkdown(exportData, opts)
	case ExportFormatHTML:
		return se.exportHTML(exportData, opts)
	case ExportFormatPDF:
		return se.exportPDF(exportData, opts)
	}

	return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported export format", nil)
}

// filterMessages filters messages based on export options
func (se *SessionExporter) filterMessages(messages []Message, opts ExportOptions) []Message {
	var filtered []Message

	for _, msg := range messages {
		// Date range filter
		if opts.DateRange != nil {
			if msg.Timestamp.Before(opts.DateRange.Start) || msg.Timestamp.After(opts.DateRange.End) {
				continue
			}
		}

		// Agent filter
		if len(opts.AgentFilter) > 0 {
			found := false
			for _, agent := range opts.AgentFilter {
				if msg.Agent == agent {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, msg)
	}

	return filtered
}

// buildMetadata creates metadata for the export
func (se *SessionExporter) buildMetadata(session *Session, opts ExportOptions) map[string]interface{} {
	if !opts.IncludeMetadata {
		return nil
	}

	metadata := map[string]interface{}{
		"export_time":   time.Now(),
		"export_format": opts.Format.String(),
		"session_id":    session.ID,
		"session_start": session.StartTime,
		"session_end":   session.LastActiveTime,
		"message_count": len(session.Messages),
	}

	if opts.IncludeContext {
		metadata["context"] = session.Context
		metadata["state"] = session.State
	}

	return metadata
}

// exportJSON exports session data as JSON
func (se *SessionExporter) exportJSON(data *ExportData) ([]byte, error) {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to marshal JSON")
	}
	return output, nil
}

// exportMarkdown exports session data as Markdown
func (se *SessionExporter) exportMarkdown(data *ExportData, opts ExportOptions) ([]byte, error) {
	var md strings.Builder

	// Header
	title := opts.Title
	if title == "" {
		title = "Guild Chat Export"
	}
	md.WriteString(fmt.Sprintf("# %s\n\n", title))

	// Session Information
	if data.Metadata != nil {
		md.WriteString("## Session Information\n\n")
		md.WriteString(fmt.Sprintf("- **Session ID**: %s\n", data.Session.ID))
		md.WriteString(fmt.Sprintf("- **Campaign**: %s\n", data.Session.CampaignID))
		md.WriteString(fmt.Sprintf("- **Started**: %s\n", data.Session.StartTime.Format("2006-01-02 15:04:05")))
		md.WriteString(fmt.Sprintf("- **Duration**: %s\n",
			data.Session.LastActiveTime.Sub(data.Session.StartTime).Round(time.Second)))
		md.WriteString(fmt.Sprintf("- **Messages**: %d\n", len(data.Messages)))

		if opts.IncludeContext && data.Session.Context.WorkingDirectory != "" {
			md.WriteString(fmt.Sprintf("- **Working Directory**: `%s`\n", data.Session.Context.WorkingDirectory))
		}
		if opts.IncludeContext && data.Session.Context.GitBranch != "" {
			md.WriteString(fmt.Sprintf("- **Git Branch**: `%s`\n", data.Session.Context.GitBranch))
		}

		md.WriteString("\n")
	}

	// Analytics Summary
	if opts.IncludeAnalytics && data.Analytics != nil {
		md.WriteString("## Analytics Summary\n\n")
		md.WriteString(fmt.Sprintf("- **Total Tokens**: %d\n", data.Analytics.TokenUsage.Total))
		md.WriteString(fmt.Sprintf("- **Reasoning Tokens**: %d (%.1f%%)\n",
			data.Analytics.TokenUsage.Reasoning,
			data.Analytics.TokenUsage.ReasoningRatio*100))
		md.WriteString(fmt.Sprintf("- **Productivity Score**: %.1f%%\n", data.Analytics.ProductivityScore))
		md.WriteString(fmt.Sprintf("- **Task Completion Rate**: %.1f%%\n",
			data.Analytics.TaskMetrics.CompletionRate*100))
		md.WriteString("\n")
	}

	// Reasoning Metrics
	if opts.IncludeReasoning && data.ReasoningMetrics != nil {
		md.WriteString("## Reasoning Analysis\n\n")
		md.WriteString(fmt.Sprintf("- **Reasoning Efficiency**: %.1f%%\n",
			data.ReasoningMetrics.ReasoningEfficiency*100))
		md.WriteString(fmt.Sprintf("- **Decision Quality**: %.1f%%\n",
			data.ReasoningMetrics.DecisionQuality*100))
		md.WriteString(fmt.Sprintf("- **Average Reasoning Depth**: %.1f\n",
			data.ReasoningMetrics.ReasoningDepth))
		md.WriteString(fmt.Sprintf("- **Pattern Complexity**: %.1f\n",
			data.ReasoningMetrics.PatternComplexity))

		// Agent reasoning styles
		if len(data.ReasoningMetrics.AgentReasoningStyles) > 0 {
			md.WriteString("\n### Agent Reasoning Styles\n\n")
			for agentID, style := range data.ReasoningMetrics.AgentReasoningStyles {
				md.WriteString(fmt.Sprintf("**%s**:\n", agentID))
				md.WriteString(fmt.Sprintf("- Average Depth: %.1f\n", style.AverageDepth))
				md.WriteString(fmt.Sprintf("- Consistency: %.1f%%\n", style.ConsistencyScore*100))
				md.WriteString(fmt.Sprintf("- Adaptability: %.1f%%\n", style.AdaptabilityScore*100))
			}
		}
		md.WriteString("\n")
	}

	// Active Agents
	if len(data.Session.State.ActiveAgents) > 0 {
		md.WriteString("## Active Agents\n\n")
		for agentID, state := range data.Session.State.ActiveAgents {
			md.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", state.Name, agentID, state.Status))
		}
		md.WriteString("\n")
	}

	// Messages
	md.WriteString("## Conversation\n\n")

	for i, msg := range data.Messages {
		// Agent header with timestamp
		md.WriteString(fmt.Sprintf("### %s _%s_\n\n",
			strings.Title(msg.Agent),
			msg.Timestamp.Format("15:04:05")))

		// Content with proper markdown formatting
		content := se.formatMarkdownContent(msg.Content, opts)
		md.WriteString(content)
		md.WriteString("\n\n")

		// Attachments
		if len(msg.Attachments) > 0 {
			md.WriteString("**Attachments:**\n")
			for _, att := range msg.Attachments {
				md.WriteString(fmt.Sprintf("- [%s](%s) (%s)\n",
					att.Name, att.Path, formatFileSize(att.Size)))
			}
			md.WriteString("\n")
		}

		// Add separator between messages (except last)
		if i < len(data.Messages)-1 {
			md.WriteString("---\n\n")
		}
	}

	// Footer
	md.WriteString("---\n")
	md.WriteString(fmt.Sprintf("_Exported on %s by Guild Framework_\n",
		time.Now().Format("2006-01-02 15:04:05")))

	return []byte(md.String()), nil
}

// formatMarkdownContent formats message content for Markdown export
func (se *SessionExporter) formatMarkdownContent(content string, opts ExportOptions) string {
	if !opts.SyntaxHighlight {
		return content
	}

	// Enhanced code block detection and formatting
	lines := strings.Split(content, "\n")
	var result strings.Builder
	inCodeBlock := false
	codeLanguage := ""

	for _, line := range lines {
		// Detect code block start
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				// Starting code block
				inCodeBlock = true
				codeLanguage = strings.TrimPrefix(line, "```")
				result.WriteString(fmt.Sprintf("```%s\n", codeLanguage))
			} else {
				// Ending code block
				inCodeBlock = false
				result.WriteString("```\n")
			}
			continue
		}

		// Add line numbers if requested and in code block
		if inCodeBlock && opts.LineNumbers {
			// This is simplified - a full implementation would track line numbers
			result.WriteString(line + "\n")
		} else {
			result.WriteString(line + "\n")
		}
	}

	return result.String()
}

// exportHTML exports session data as HTML
func (se *SessionExporter) exportHTML(data *ExportData, opts ExportOptions) ([]byte, error) {
	tmplStr := se.getHTMLTemplate(opts)

	tmpl, err := template.New("export").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("15:04:05")
		},
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"formatDuration": func(start, end time.Time) string {
			return end.Sub(start).Round(time.Second).String()
		},
		"agentClass": func(agent string) string {
			return fmt.Sprintf("agent-%s", strings.ToLower(agent))
		},
		"formatContent": func(content string) template.HTML {
			return template.HTML(se.formatHTMLContent(content, opts))
		},
		"safeCSS": func(css string) template.CSS {
			return template.CSS(css)
		},
	}).Parse(tmplStr)

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to parse HTML template")
	}

	// Add CSS to export data
	data.CSS = `
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; margin: 40px; }
		.header { border-bottom: 2px solid #eee; padding-bottom: 20px; margin-bottom: 30px; }
		.message { margin-bottom: 20px; padding: 15px; border-left: 4px solid #ddd; }
		.agent-user { border-left-color: #007acc; }
		.agent-assistant { border-left-color: #28a745; }
		.timestamp { color: #666; font-size: 0.9em; }
		pre { background: #f8f9fa; padding: 15px; border-radius: 4px; overflow-x: auto; }
	`

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to execute HTML template")
	}

	return buf.Bytes(), nil
}

// getHTMLTemplate returns the HTML template string
func (se *SessionExporter) getHTMLTemplate(opts ExportOptions) string {
	baseTemplate := `<!DOCTYPE html>
<html>
<head>
    <title>{{if .Session}}Guild Chat Export - {{.Session.ID}}{{else}}Guild Chat Export{{end}}</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        {{safeCSS .CSS}}
    </style>
    {{if .CustomCSS}}<style>{{safeCSS .CustomCSS}}</style>{{end}}
</head>
<body>
    <div class="container">
        <header>
            <h1>Guild Chat Export</h1>
            {{if .Session}}
            <div class="session-info">
                <p><strong>Session ID:</strong> {{.Session.ID}}</p>
                <p><strong>Campaign:</strong> {{.Session.CampaignID}}</p>
                <p><strong>Started:</strong> {{formatDate .Session.StartTime}} {{formatTime .Session.StartTime}}</p>
                <p><strong>Duration:</strong> {{formatDuration .Session.StartTime .Session.LastActiveTime}}</p>
                <p><strong>Messages:</strong> {{len .Messages}}</p>
            </div>
            {{end}}
        </header>

        <main>
            {{range .Messages}}
            <div class="message {{agentClass .Agent}}">
                <div class="message-header">
                    <span class="agent">{{.Agent}}</span>
                    <span class="timestamp">{{formatTime .Timestamp}}</span>
                </div>
                <div class="message-content">{{formatContent .Content}}</div>
                {{if .Attachments}}
                <div class="attachments">
                    <h4>Attachments:</h4>
                    <ul>
                    {{range .Attachments}}
                        <li><a href="{{.Path}}">{{.Name}}</a> ({{.Type}})</li>
                    {{end}}
                    </ul>
                </div>
                {{end}}
            </div>
            {{end}}
        </main>

        <footer>
            <p>Exported on {{formatDate .Metadata.export_time}} {{formatTime .Metadata.export_time}} by Guild Framework</p>
        </footer>
    </div>
</body>
</html>`

	return baseTemplate
}

// formatHTMLContent formats content for HTML display
func (se *SessionExporter) formatHTMLContent(content string, opts ExportOptions) string {
	// Basic HTML escaping and formatting
	content = template.HTMLEscapeString(content)

	// Convert newlines to <br>
	content = strings.ReplaceAll(content, "\n", "<br>")

	// Enhanced code block formatting for HTML
	if opts.SyntaxHighlight {
		content = se.highlightCodeBlocks(content)
	}

	return content
}

// highlightCodeBlocks adds syntax highlighting to code blocks
func (se *SessionExporter) highlightCodeBlocks(content string) string {
	// Simplified syntax highlighting - a full implementation would use a proper syntax highlighter
	lines := strings.Split(content, "<br>")
	var result strings.Builder
	inCodeBlock := false

	for _, line := range lines {
		if strings.Contains(line, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				result.WriteString(`<div class="code-block"><pre><code>`)
			} else {
				inCodeBlock = false
				result.WriteString(`</code></pre></div>`)
			}
			continue
		}

		if inCodeBlock {
			result.WriteString(line + "\n")
		} else {
			result.WriteString(line + "<br>")
		}
	}

	return result.String()
}

// exportPDF exports session data as PDF (placeholder implementation)
func (se *SessionExporter) exportPDF(data *ExportData, opts ExportOptions) ([]byte, error) {
	// For a complete implementation, this would use a PDF generation library
	// like gofpdf or convert HTML to PDF using a tool like wkhtmltopdf

	// For now, return an error indicating this feature needs external dependencies
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "PDF export requires external dependencies. Please use HTML export and convert using a PDF tool.", nil)
}

// SessionImporter handles session import functionality
type SessionImporter struct {
	manager   *SessionManager
	validator *ImportValidator
}

// NewSessionImporter creates a new session importer
func NewSessionImporter(manager *SessionManager) *SessionImporter {
	return &SessionImporter{
		manager:   manager,
		validator: NewImportValidator(),
	}
}

// Import imports session data from the specified format
func (si *SessionImporter) Import(ctx context.Context, data []byte, format ExportFormat) (*Session, error) {
	// Parse based on format
	var importData *ExportData
	var err error

	switch format {
	case ExportFormatJSON:
		importData, err = si.parseJSON(data)
	case ExportFormatMarkdown:
		importData, err = si.parseMarkdown(data)
	default:
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "unsupported import format", nil)
	}

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to parse import data")
	}

	// Validate import data
	if err := si.validator.Validate(importData); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "import data validation failed")
	}

	// Create new session with imported data
	session := &Session{
		ID:         generateSessionID(),
		UserID:     getCurrentUserID(),
		CampaignID: importData.Session.CampaignID,
		StartTime:  time.Now(),
		Messages:   importData.Messages,
		State: SessionState{
			ActiveAgents: make(map[string]AgentState),
			Variables:    make(map[string]interface{}),
			Status:       SessionStatusActive,
		},
		Context: SessionContext{
			WorkingDirectory: "/tmp", // Default working directory
		},
		Metadata: map[string]interface{}{
			"imported":      true,
			"import_source": format.String(),
			"import_time":   time.Now(),
			"original_id":   importData.Session.ID,
		},
	}

	// Copy original metadata if available
	if importData.Session.Metadata != nil {
		for k, v := range importData.Session.Metadata {
			session.Metadata[k] = v
		}
	}

	// Save imported session
	if err := si.manager.SaveSession(ctx, session); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save imported session")
	}

	return session, nil
}

// parseJSON parses JSON import data
func (si *SessionImporter) parseJSON(data []byte) (*ExportData, error) {
	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to unmarshal JSON")
	}
	return &exportData, nil
}

// parseMarkdown parses Markdown import data (simplified implementation)
func (si *SessionImporter) parseMarkdown(data []byte) (*ExportData, error) {
	// This is a simplified implementation - a full parser would be more robust
	content := string(data)

	// Extract basic session information from markdown
	session := &Session{
		ID:        "imported-session",
		StartTime: time.Now(),
	}

	// Parse messages from markdown (very basic parsing)
	var messages []Message
	sections := strings.Split(content, "### ")

	for _, section := range sections[1:] { // Skip first empty section
		lines := strings.Split(section, "\n")
		if len(lines) < 2 {
			continue
		}

		// Extract agent and timestamp from header
		header := lines[0]
		parts := strings.Split(header, " _")
		agent := strings.TrimSpace(parts[0])

		// Extract content
		contentLines := lines[2:] // Skip header and empty line
		content := strings.Join(contentLines, "\n")
		content = strings.TrimSpace(content)

		if content != "" && agent != "" {
			msg := Message{
				ID:        generateMessageID(),
				Agent:     agent,
				Content:   content,
				Timestamp: time.Now(),
				Type:      MessageTypeAgent,
			}
			messages = append(messages, msg)
		}
	}

	return &ExportData{
		Session:  session,
		Messages: messages,
	}, nil
}

// ImportValidator validates imported session data
type ImportValidator struct{}

// NewImportValidator creates a new import validator
func NewImportValidator() *ImportValidator {
	return &ImportValidator{}
}

// Validate validates import data
func (iv *ImportValidator) Validate(data *ExportData) error {
	if data == nil {
		return gerror.New(gerror.ErrCodeValidation, "import data is nil", nil)
	}

	if data.Session == nil {
		return gerror.New(gerror.ErrCodeValidation, "session data is missing", nil)
	}

	if data.Session.ID == "" {
		return gerror.New(gerror.ErrCodeValidation, "session ID is required", nil)
	}

	// Validate messages
	for i, msg := range data.Messages {
		if msg.ID == "" {
			return gerror.New(gerror.ErrCodeValidation, fmt.Sprintf("message %d is missing ID", i), nil)
		}
		if msg.Content == "" {
			return gerror.New(gerror.ErrCodeValidation, fmt.Sprintf("message %d is missing content", i), nil)
		}
	}

	return nil
}

// ExportFormatter handles formatting for different export types
type ExportFormatter struct{}

// NewExportFormatter creates a new export formatter
func NewExportFormatter() *ExportFormatter {
	return &ExportFormatter{}
}

// ExportSession exports a single session
func (se *SessionExporter) ExportSession(session *Session, format ExportFormat, options ExportOptions) ([]byte, error) {
	options.Format = format
	return se.Export(session, options)
}

// ExportSessions exports multiple sessions
func (se *SessionExporter) ExportSessions(sessions []*Session, format ExportFormat, options ExportOptions) ([]byte, error) {
	if len(sessions) == 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "no sessions to export", nil)
	}

	if len(sessions) == 1 {
		return se.ExportSession(sessions[0], format, options)
	}

	// For multiple sessions, create a combined export
	var allMessages []Message
	var combinedMetadata = make(map[string]interface{})

	for _, session := range sessions {
		allMessages = append(allMessages, session.Messages...)
		// Combine metadata from all sessions
		for k, v := range session.Metadata {
			combinedMetadata[k] = v
		}
	}

	// Create a virtual combined session
	combinedSession := &Session{
		ID:             "combined-export",
		UserID:         sessions[0].UserID,
		StartTime:      sessions[0].StartTime,
		LastActiveTime: sessions[len(sessions)-1].LastActiveTime,
		Messages:       allMessages,
		Metadata:       combinedMetadata,
	}

	return se.ExportSession(combinedSession, format, options)
}

// ImportSession imports a session from data
func (se *SessionExporter) ImportSession(ctx context.Context, data []byte, format ExportFormat) (*Session, error) {
	importer := NewSessionImporter(nil) // TODO: pass proper manager
	return importer.Import(ctx, data, format)
}

// ValidateImportData validates import data
func (se *SessionExporter) ValidateImportData(data []byte, format ExportFormat) error {
	// For now, just check basic format validity
	if len(data) == 0 {
		return gerror.New(gerror.ErrCodeInvalidInput, "import data is empty", nil)
	}

	switch format {
	case ExportFormatJSON:
		var temp map[string]interface{}
		return json.Unmarshal(data, &temp)
	case ExportFormatMarkdown:
		// Basic markdown validation - just check it's not empty
		if len(strings.TrimSpace(string(data))) == 0 {
			return gerror.New(gerror.ErrCodeInvalidInput, "markdown data is empty", nil)
		}
		return nil
	default:
		return gerror.New(gerror.ErrCodeInvalidInput, "unsupported format for validation", nil)
	}
}

// GetSupportedFormats returns supported export formats
func (se *SessionExporter) GetSupportedFormats() []ExportFormat {
	return []ExportFormat{
		ExportFormatJSON,
		ExportFormatMarkdown,
		ExportFormatHTML,
		ExportFormatPDF,
	}
}

// GetFormatCapabilities returns capabilities for a format
func (se *SessionExporter) GetFormatCapabilities(format ExportFormat) FormatCapabilities {
	switch format {
	case ExportFormatJSON:
		return FormatCapabilities{
			SupportsMetadata:    true,
			SupportsAttachments: true,
			SupportsFormatting:  false,
			MaxFileSize:         10 * 1024 * 1024, // 10MB
			Extensions:          []string{".json"},
		}
	case ExportFormatMarkdown:
		return FormatCapabilities{
			SupportsMetadata:    true,
			SupportsAttachments: true,
			SupportsFormatting:  true,
			MaxFileSize:         5 * 1024 * 1024, // 5MB
			Extensions:          []string{".md", ".markdown"},
		}
	case ExportFormatHTML:
		return FormatCapabilities{
			SupportsMetadata:    true,
			SupportsAttachments: true,
			SupportsFormatting:  true,
			MaxFileSize:         20 * 1024 * 1024, // 20MB
			Extensions:          []string{".html", ".htm"},
		}
	case ExportFormatPDF:
		return FormatCapabilities{
			SupportsMetadata:    true,
			SupportsAttachments: false,
			SupportsFormatting:  true,
			MaxFileSize:         50 * 1024 * 1024, // 50MB
			Extensions:          []string{".pdf"},
		}
	default:
		return FormatCapabilities{}
	}
}

// Helper functions

func getCurrentUserID() string {
	// In a real implementation, this would get the current user ID
	return "current_user"
}

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
