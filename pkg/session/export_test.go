// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func createTestSession() *Session {
	return &Session{
		ID:         "test-session-123",
		UserID:     "test-user",
		CampaignID: "test-campaign",
		StartTime:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
		LastActiveTime: time.Date(2025, 1, 1, 12, 30, 0, 0, time.UTC),
		State: SessionState{
			ActiveAgents: map[string]AgentState{
				"elena": {
					ID:           "elena",
					Name:         "Elena",
					Status:       "active",
					LastActivity: time.Now(),
					Context:      map[string]interface{}{"mode": "helpful"},
					TaskQueue:    []string{"task-1"},
				},
				"marcus": {
					ID:           "marcus",
					Name:         "Marcus",
					Status:       "idle",
					LastActivity: time.Now(),
					Context:      map[string]interface{}{"specialization": "development"},
					TaskQueue:    []string{},
				},
			},
			CurrentView:    "chat",
			ScrollPosition: 150,
			Variables:      map[string]interface{}{"theme": "dark", "debug": true},
			Status:         SessionStatusActive,
		},
		Messages: []Message{
			{
				ID:        "msg-1",
				Agent:     "user",
				Content:   "Hello, I need help with a Go project",
				Timestamp: time.Date(2025, 1, 1, 10, 5, 0, 0, time.UTC),
				Type:      MessageTypeUser,
				Metadata:  map[string]interface{}{"input_method": "typing"},
			},
			{
				ID:        "msg-2",
				Agent:     "elena",
				Content:   "I'd be happy to help! What specific aspect of your Go project would you like assistance with?",
				Timestamp: time.Date(2025, 1, 1, 10, 5, 30, 0, time.UTC),
				Type:      MessageTypeAgent,
				Metadata:  map[string]interface{}{"tokens": 25, "model": "claude-3-sonnet"},
			},
			{
				ID:        "msg-3",
				Agent:     "user",
				Content:   "I'm trying to implement session management. Here's my code:\n```go\nfunc SaveSession(s *Session) error {\n    return nil\n}\n```",
				Timestamp: time.Date(2025, 1, 1, 10, 7, 0, 0, time.UTC),
				Type:      MessageTypeUser,
				Attachments: []Attachment{
					{Name: "session.go", Path: "/tmp/session.go", Type: "text/plain", Size: 1024},
				},
			},
			{
				ID:        "msg-4",
				Agent:     "marcus",
				Content:   "Great! I can help you implement proper session persistence. Let me review your code and suggest improvements.",
				Timestamp: time.Date(2025, 1, 1, 10, 8, 0, 0, time.UTC),
				Type:      MessageTypeAgent,
				Metadata:  map[string]interface{}{"tokens": 22, "model": "claude-3-sonnet"},
			},
		},
		Context: SessionContext{
			WorkingDirectory: "/home/user/guild-project",
			GitBranch:        "feature/session-management",
			OpenFiles:        []string{"session.go", "session_test.go", "README.md"},
			RunningTasks:     []string{"test-session-persistence"},
		},
		Metadata: map[string]interface{}{
			"version":     "1.0",
			"created_by":  "guild-chat",
			"project":     "session-management",
		},
	}
}

func TestCraftSessionExportJSON(t *testing.T) {
	exporter := NewSessionExporter()
	session := createTestSession()

	opts := ExportOptions{
		Format:          ExportFormatJSON,
		IncludeMetadata: true,
		IncludeContext:  true,
	}

	// Test JSON export
	data, err := exporter.Export(session, opts)
	if err != nil {
		t.Fatalf("Failed to export session as JSON: %v", err)
	}

	// Verify it's valid JSON
	var exportData ExportData
	err = json.Unmarshal(data, &exportData)
	if err != nil {
		t.Fatalf("Exported JSON is invalid: %v", err)
	}

	// Verify session data
	if exportData.Session.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, exportData.Session.ID)
	}

	if len(exportData.Messages) != len(session.Messages) {
		t.Errorf("Expected %d messages, got %d", len(session.Messages), len(exportData.Messages))
	}

	// Verify metadata is included
	if exportData.Metadata == nil {
		t.Error("Metadata should be included when IncludeMetadata is true")
	}
}

func TestCraftSessionExportMarkdown(t *testing.T) {
	exporter := NewSessionExporter()
	session := createTestSession()

	opts := ExportOptions{
		Format:          ExportFormatMarkdown,
		IncludeMetadata: true,
		IncludeContext:  true,
		SyntaxHighlight: true,
		Title:           "Test Session Export",
	}

	// Test Markdown export
	data, err := exporter.Export(session, opts)
	if err != nil {
		t.Fatalf("Failed to export session as Markdown: %v", err)
	}

	content := string(data)

	// Verify markdown structure
	if !strings.Contains(content, "# Test Session Export") {
		t.Error("Expected custom title in markdown")
	}

	if !strings.Contains(content, "## Session Information") {
		t.Error("Expected session information section")
	}

	if !strings.Contains(content, "## Active Agents") {
		t.Error("Expected active agents section")
	}

	if !strings.Contains(content, "## Conversation") {
		t.Error("Expected conversation section")
	}

	// Verify session details are included
	if !strings.Contains(content, session.ID) {
		t.Error("Session ID should be in markdown export")
	}

	if !strings.Contains(content, session.CampaignID) {
		t.Error("Campaign ID should be in markdown export")
	}

	// Verify code blocks are preserved
	if !strings.Contains(content, "```go") {
		t.Error("Code blocks should be preserved in markdown")
	}

	// Verify agent messages are included
	for _, msg := range session.Messages {
		if !strings.Contains(content, msg.Content) {
			t.Errorf("Message content should be included: %s", msg.Content[:50])
		}
	}

	// Verify attachments are listed
	if !strings.Contains(content, "**Attachments:**") {
		t.Error("Attachments section should be included for messages with attachments")
	}
}

func TestCraftSessionExportHTML(t *testing.T) {
	exporter := NewSessionExporter()
	session := createTestSession()

	opts := ExportOptions{
		Format:          ExportFormatHTML,
		IncludeMetadata: true,
		SyntaxHighlight: true,
		Theme:           "dark",
	}

	// Test HTML export
	data, err := exporter.Export(session, opts)
	if err != nil {
		t.Fatalf("Failed to export session as HTML: %v", err)
	}

	content := string(data)

	// Verify HTML structure
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("Expected proper HTML document structure")
	}

	if !strings.Contains(content, "<html>") {
		t.Error("Expected HTML tag")
	}

	if !strings.Contains(content, "<head>") {
		t.Error("Expected HTML head section")
	}

	if !strings.Contains(content, "<body>") {
		t.Error("Expected HTML body section")
	}

	// Verify session information is included
	if !strings.Contains(content, session.ID) {
		t.Error("Session ID should be in HTML export")
	}

	// Verify messages are structured properly
	if !strings.Contains(content, `class="message`) {
		t.Error("Expected message CSS classes")
	}

	if !strings.Contains(content, `class="agent"`) {
		t.Error("Expected agent CSS classes")
	}

	// Verify timestamp formatting
	if !strings.Contains(content, "10:05:00") {
		t.Error("Expected formatted timestamps")
	}
}

func TestCraftSessionExportFiltering(t *testing.T) {
	exporter := NewSessionExporter()
	session := createTestSession()

	// Test date range filtering
	opts := ExportOptions{
		Format: ExportFormatJSON,
		DateRange: &DateRange{
			Start: time.Date(2025, 1, 1, 10, 6, 0, 0, time.UTC),
			End:   time.Date(2025, 1, 1, 10, 8, 0, 0, time.UTC),
		},
	}

	data, err := exporter.Export(session, opts)
	if err != nil {
		t.Fatalf("Failed to export with date filtering: %v", err)
	}

	var exportData ExportData
	json.Unmarshal(data, &exportData)

	// Should only include messages within the date range
	expectedMessages := 2 // msg-3 and msg-4
	if len(exportData.Messages) != expectedMessages {
		t.Errorf("Expected %d messages after date filtering, got %d", expectedMessages, len(exportData.Messages))
	}

	// Test agent filtering
	opts = ExportOptions{
		Format:      ExportFormatJSON,
		AgentFilter: []string{"elena"},
	}

	data, err = exporter.Export(session, opts)
	if err != nil {
		t.Fatalf("Failed to export with agent filtering: %v", err)
	}

	json.Unmarshal(data, &exportData)

	// Should only include messages from elena
	expectedMessages = 1 // Only msg-2
	if len(exportData.Messages) != expectedMessages {
		t.Errorf("Expected %d messages after agent filtering, got %d", expectedMessages, len(exportData.Messages))
	}

	if exportData.Messages[0].Agent != "elena" {
		t.Errorf("Expected message from elena, got %s", exportData.Messages[0].Agent)
	}
}

func TestCraftSessionImportJSON(t *testing.T) {
	ctx := context.Background()
	store := newMockSessionStore()
	manager := NewSessionManager(store)
	importer := NewSessionImporter(manager)

	// Create test export data
	session := createTestSession()
	exporter := NewSessionExporter()
	opts := ExportOptions{
		Format:          ExportFormatJSON,
		IncludeMetadata: true,
		IncludeContext:  true,
	}

	exportedData, err := exporter.Export(session, opts)
	if err != nil {
		t.Fatalf("Failed to export session for import test: %v", err)
	}

	// Test import
	importedSession, err := importer.Import(ctx, exportedData, ExportFormatJSON)
	if err != nil {
		t.Fatalf("Failed to import session: %v", err)
	}

	// Verify imported session
	if importedSession.CampaignID != session.CampaignID {
		t.Errorf("Expected campaign ID %s, got %s", session.CampaignID, importedSession.CampaignID)
	}

	if len(importedSession.Messages) != len(session.Messages) {
		t.Errorf("Expected %d messages, got %d", len(session.Messages), len(importedSession.Messages))
	}

	// Verify import metadata
	if importedSession.Metadata["imported"] != true {
		t.Error("Expected imported flag to be set")
	}

	if importedSession.Metadata["import_source"] != "json" {
		t.Error("Expected import source to be 'json'")
	}

	if importedSession.Metadata["original_id"] != session.ID {
		t.Error("Expected original_id to be preserved")
	}
}

func TestCraftSessionImportMarkdown(t *testing.T) {
	ctx := context.Background()
	store := newMockSessionStore()
	manager := NewSessionManager(store)
	importer := NewSessionImporter(manager)

	// Create test markdown data
	markdownData := `# Guild Chat Export

## Session Information

- **Session ID**: test-session-123
- **Campaign**: test-campaign

## Conversation

### Elena _10:05:30_

I'd be happy to help! What specific aspect of your Go project would you like assistance with?

### User _10:07:00_

I'm trying to implement session management. Here's my code:

` + "```go" + `
func SaveSession(s *Session) error {
    return nil
}
` + "```" + `

### Marcus _10:08:00_

Great! I can help you implement proper session persistence.
`

	// Test import
	importedSession, err := importer.Import(ctx, []byte(markdownData), ExportFormatMarkdown)
	if err != nil {
		t.Fatalf("Failed to import markdown session: %v", err)
	}

	// Verify basic session properties
	if importedSession == nil {
		t.Fatal("Imported session is nil")
	}

	// Verify messages were parsed
	if len(importedSession.Messages) == 0 {
		t.Error("Expected messages to be parsed from markdown")
	}

	// Verify import metadata
	if importedSession.Metadata["imported"] != true {
		t.Error("Expected imported flag to be set")
	}

	if importedSession.Metadata["import_source"] != "markdown" {
		t.Error("Expected import source to be 'markdown'")
	}
}

func TestCraftExportValidation(t *testing.T) {
	validator := NewImportValidator()

	// Test valid data
	validData := &ExportData{
		Session: &Session{
			ID:         "valid-session",
			UserID:     "user",
			CampaignID: "campaign",
		},
		Messages: []Message{
			{
				ID:      "msg-1",
				Content: "Hello",
				Agent:   "test",
			},
		},
	}

	err := validator.Validate(validData)
	if err != nil {
		t.Errorf("Valid data should pass validation: %v", err)
	}

	// Test nil data
	err = validator.Validate(nil)
	if err == nil {
		t.Error("Nil data should fail validation")
	}

	// Test missing session
	invalidData := &ExportData{
		Session: nil,
	}

	err = validator.Validate(invalidData)
	if err == nil {
		t.Error("Missing session should fail validation")
	}

	// Test missing session ID
	invalidData = &ExportData{
		Session: &Session{
			ID: "",
		},
	}

	err = validator.Validate(invalidData)
	if err == nil {
		t.Error("Missing session ID should fail validation")
	}

	// Test message with missing ID
	invalidData = &ExportData{
		Session: &Session{
			ID: "valid",
		},
		Messages: []Message{
			{
				ID:      "",
				Content: "Hello",
			},
		},
	}

	err = validator.Validate(invalidData)
	if err == nil {
		t.Error("Message with missing ID should fail validation")
	}

	// Test message with missing content
	invalidData = &ExportData{
		Session: &Session{
			ID: "valid",
		},
		Messages: []Message{
			{
				ID:      "msg-1",
				Content: "",
			},
		},
	}

	err = validator.Validate(invalidData)
	if err == nil {
		t.Error("Message with missing content should fail validation")
	}
}

func TestCraftRoundTripExportImport(t *testing.T) {
	ctx := context.Background()
	store := newMockSessionStore()
	manager := NewSessionManager(store)

	// Original session
	originalSession := createTestSession()

	// Export and import cycle
	exporter := NewSessionExporter()
	importer := NewSessionImporter(manager)

	opts := ExportOptions{
		Format:          ExportFormatJSON,
		IncludeMetadata: true,
		IncludeContext:  true,
	}

	// Export
	exportedData, err := exporter.Export(originalSession, opts)
	if err != nil {
		t.Fatalf("Failed to export session: %v", err)
	}

	// Import
	importedSession, err := importer.Import(ctx, exportedData, ExportFormatJSON)
	if err != nil {
		t.Fatalf("Failed to import session: %v", err)
	}

	// Compare key properties (IDs will be different due to import process)
	if importedSession.CampaignID != originalSession.CampaignID {
		t.Errorf("Campaign ID mismatch: expected %s, got %s", originalSession.CampaignID, importedSession.CampaignID)
	}

	if len(importedSession.Messages) != len(originalSession.Messages) {
		t.Errorf("Message count mismatch: expected %d, got %d", len(originalSession.Messages), len(importedSession.Messages))
	}

	// Compare message content
	for i, originalMsg := range originalSession.Messages {
		if i >= len(importedSession.Messages) {
			break
		}
		importedMsg := importedSession.Messages[i]

		if importedMsg.Content != originalMsg.Content {
			t.Errorf("Message %d content mismatch", i)
		}

		if importedMsg.Agent != originalMsg.Agent {
			t.Errorf("Message %d agent mismatch: expected %s, got %s", i, originalMsg.Agent, importedMsg.Agent)
		}
	}
}