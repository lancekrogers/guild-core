// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/corpus"
)

func TestCorpusTool(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "corpus-tool-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := corpus.Config{
		CorpusPath:     tempDir,
		ActivitiesPath: filepath.Join(tempDir, ".activities"),
		MaxSizeBytes:   1024 * 1024,
	}

	// Create activities directory
	err = os.MkdirAll(cfg.ActivitiesPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create activities directory: %v", err)
	}

	// Create a corpus tool
	tool := NewCorpusTool(cfg)

	// Check tool metadata
	if tool.Name() != "corpus" {
		t.Errorf("Expected tool name 'corpus', got '%s'", tool.Name())
	}

	if tool.Category() != "knowledge" {
		t.Errorf("Expected category 'knowledge', got '%s'", tool.Category())
	}

	if tool.RequiresAuth() {
		t.Error("Tool should not require auth")
	}

	// Test the tool schema
	schema := tool.Schema()
	if schema["type"] != "object" {
		t.Errorf("Expected schema type 'object', got '%v'", schema["type"])
	}

	// Check that the schema has the required properties
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected properties to be a map")
	}

	requiredProperties := []string{"action", "title", "content", "tags", "query"}
	for _, prop := range requiredProperties {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Expected schema to have property '%s'", prop)
		}
	}

	// Test save document
	saveInput := Input{
		Action:  "save",
		Title:   "Test Document",
		Content: "This is a test document.",
		Tags:    []string{"test", "document"},
		Source:  "unit test",
		GuildID: "test-guild",
		AgentID: "test-agent",
	}

	saveInputJSON, err := json.Marshal(saveInput)
	if err != nil {
		t.Fatalf("Failed to marshal save input: %v", err)
	}

	ctx := context.Background()
	saveResult, err := tool.Execute(ctx, string(saveInputJSON))
	if err != nil {
		t.Fatalf("Failed to execute save: %v", err)
	}

	if !saveResult.Success {
		t.Errorf("Expected save to succeed, got error: %s", saveResult.Error)
	}

	if !strings.Contains(saveResult.Output, "saved successfully") {
		t.Errorf("Expected save output to contain 'saved successfully', got '%s'", saveResult.Output)
	}

	// Test list documents
	listInput := Input{
		Action: "list",
	}

	listInputJSON, err := json.Marshal(listInput)
	if err != nil {
		t.Fatalf("Failed to marshal list input: %v", err)
	}

	listResult, err := tool.Execute(ctx, string(listInputJSON))
	if err != nil {
		t.Fatalf("Failed to execute list: %v", err)
	}

	if !listResult.Success {
		t.Errorf("Expected list to succeed, got error: %s", listResult.Error)
	}

	if !strings.Contains(listResult.Output, "Test Document") {
		t.Errorf("Expected list output to contain 'Test Document', got '%s'", listResult.Output)
	}

	// Test load document
	loadInput := Input{
		Action: "load",
		Title:  "Test Document",
	}

	loadInputJSON, err := json.Marshal(loadInput)
	if err != nil {
		t.Fatalf("Failed to marshal load input: %v", err)
	}

	loadResult, err := tool.Execute(ctx, string(loadInputJSON))
	if err != nil {
		t.Fatalf("Failed to execute load: %v", err)
	}

	if !loadResult.Success {
		t.Errorf("Expected load to succeed, got error: %s", loadResult.Error)
	}

	if !strings.Contains(loadResult.Output, "This is a test document.") {
		t.Errorf("Expected load output to contain document content, got '%s'", loadResult.Output)
	}

	// Test search documents
	searchInput := Input{
		Action: "search",
		Query:  "test",
	}

	searchInputJSON, err := json.Marshal(searchInput)
	if err != nil {
		t.Fatalf("Failed to marshal search input: %v", err)
	}

	searchResult, err := tool.Execute(ctx, string(searchInputJSON))
	if err != nil {
		t.Fatalf("Failed to execute search: %v", err)
	}

	if !searchResult.Success {
		t.Errorf("Expected search to succeed, got error: %s", searchResult.Error)
	}

	if !strings.Contains(searchResult.Output, "Test Document") {
		t.Errorf("Expected search output to contain 'Test Document', got '%s'", searchResult.Output)
	}

	// Test graph
	graphInput := Input{
		Action: "graph",
	}

	graphInputJSON, err := json.Marshal(graphInput)
	if err != nil {
		t.Fatalf("Failed to marshal graph input: %v", err)
	}

	graphResult, err := tool.Execute(ctx, string(graphInputJSON))
	if err != nil {
		t.Fatalf("Failed to execute graph: %v", err)
	}

	if !graphResult.Success {
		t.Errorf("Expected graph to succeed, got error: %s", graphResult.Error)
	}

	if !strings.Contains(graphResult.Output, "Corpus Graph") {
		t.Errorf("Expected graph output to contain 'Corpus Graph', got '%s'", graphResult.Output)
	}

	// Test delete document
	deleteInput := Input{
		Action: "delete",
		Title:  "Test Document",
	}

	deleteInputJSON, err := json.Marshal(deleteInput)
	if err != nil {
		t.Fatalf("Failed to marshal delete input: %v", err)
	}

	deleteResult, err := tool.Execute(ctx, string(deleteInputJSON))
	if err != nil {
		t.Fatalf("Failed to execute delete: %v", err)
	}

	if !deleteResult.Success {
		t.Errorf("Expected delete to succeed, got error: %s", deleteResult.Error)
	}

	if !strings.Contains(deleteResult.Output, "deleted successfully") {
		t.Errorf("Expected delete output to contain 'deleted successfully', got '%s'", deleteResult.Output)
	}

	// Verify the document was deleted
	listResult, err = tool.Execute(ctx, string(listInputJSON))
	if err != nil {
		t.Fatalf("Failed to execute list after delete: %v", err)
	}

	if strings.Contains(listResult.Output, "Test Document") {
		t.Errorf("Expected document to be deleted, but it still appears in the list")
	}
}

func TestCorpusToolInvalidInput(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "corpus-tool-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := corpus.Config{
		CorpusPath:     tempDir,
		ActivitiesPath: filepath.Join(tempDir, ".activities"),
		MaxSizeBytes:   1024 * 1024,
	}

	// Create a corpus tool
	tool := NewCorpusTool(cfg)

	// Test invalid JSON
	ctx := context.Background()
	invalidResult, err := tool.Execute(ctx, "invalid json")
	if err != nil {
		t.Fatalf("Execute should not return error for invalid input: %v", err)
	}

	if invalidResult.Success {
		t.Error("Expected failure for invalid JSON")
	}

	// Test unknown action
	unknownInput := Input{
		Action: "unknown",
	}

	unknownInputJSON, err := json.Marshal(unknownInput)
	if err != nil {
		t.Fatalf("Failed to marshal unknown input: %v", err)
	}

	unknownResult, err := tool.Execute(ctx, string(unknownInputJSON))
	if err != nil {
		t.Fatalf("Execute should not return error for unknown action: %v", err)
	}

	if unknownResult.Success {
		t.Error("Expected failure for unknown action")
	}

	// Test missing required parameters
	missingInput := Input{
		Action: "save",
		// Missing title
		Content: "Content without a title",
	}

	missingInputJSON, err := json.Marshal(missingInput)
	if err != nil {
		t.Fatalf("Failed to marshal missing input: %v", err)
	}

	missingResult, err := tool.Execute(ctx, string(missingInputJSON))
	if err != nil {
		t.Fatalf("Execute should not return error for missing parameters: %v", err)
	}

	if missingResult.Success {
		t.Error("Expected failure for missing required parameters")
	}
}
