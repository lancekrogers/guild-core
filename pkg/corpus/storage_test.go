package corpus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "corpus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration
	cfg := Config{
		CorpusPath:     tempDir,
		ActivitiesPath: filepath.Join(tempDir, ".activities"),
		MaxSizeBytes:   1024 * 1024,
	}

	// Create activities directory
	err = os.MkdirAll(cfg.ActivitiesPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create activities directory: %v", err)
	}

	// Create a test document
	doc := CorpusDoc{
		Title:     "Test Document",
		Source:    "Unit Test",
		Tags:      []string{"test", "document"},
		Body:      "This is a test document for the corpus system.",
		GuildID:   "test-guild",
		AgentID:   "test-agent",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save the document
	err = Save(&doc, cfg)
	if err != nil {
		t.Fatalf("Failed to save document: %v", err)
	}

	// Verify the document was saved
	if doc.FilePath == "" {
		t.Fatalf("Document FilePath was not set")
	}

	// Check if the file exists
	_, err = os.Stat(doc.FilePath)
	if os.IsNotExist(err) {
		t.Fatalf("Document file was not created: %v", err)
	}

	// Load the document
	loadedDoc, err := Load(doc.FilePath)
	if err != nil {
		t.Fatalf("Failed to load document: %v", err)
	}

	// Verify the loaded document
	if loadedDoc.Title != doc.Title {
		t.Errorf("Expected title %s, got %s", doc.Title, loadedDoc.Title)
	}

	if loadedDoc.Source != doc.Source {
		t.Errorf("Expected source %s, got %s", doc.Source, loadedDoc.Source)
	}

	if loadedDoc.Body != doc.Body {
		t.Errorf("Expected body %s, got %s", doc.Body, loadedDoc.Body)
	}

	if loadedDoc.GuildID != doc.GuildID {
		t.Errorf("Expected guildID %s, got %s", doc.GuildID, loadedDoc.GuildID)
	}

	if loadedDoc.AgentID != doc.AgentID {
		t.Errorf("Expected agentID %s, got %s", doc.AgentID, loadedDoc.AgentID)
	}

	if len(loadedDoc.Tags) != len(doc.Tags) {
		t.Errorf("Expected %d tags, got %d", len(doc.Tags), len(loadedDoc.Tags))
	}

	for i, tag := range doc.Tags {
		if loadedDoc.Tags[i] != tag {
			t.Errorf("Expected tag %s, got %s", tag, loadedDoc.Tags[i])
		}
	}

	// Test document listing
	docs, err := List(cfg)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	// Test document deletion
	err = Delete(doc.FilePath)
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Verify the file was deleted
	_, err = os.Stat(doc.FilePath)
	if !os.IsNotExist(err) {
		t.Errorf("Document file was not deleted")
	}

	// Test listing after deletion
	docs, err = List(cfg)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("Expected 0 documents, got %d", len(docs))
	}
}

func TestMaxSizeConstraint(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "corpus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test configuration with very small size limit
	cfg := Config{
		CorpusPath:     tempDir,
		ActivitiesPath: filepath.Join(tempDir, ".activities"),
		MaxSizeBytes:   1, // Just 1 byte (much smaller than document content)
	}

	// Create activities directory
	err = os.MkdirAll(cfg.ActivitiesPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create activities directory: %v", err)
	}

	// Create a document larger than the size limit
	doc := CorpusDoc{
		Title:     "Large Document",
		Body:      "This document is larger than the allowed size.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Attempt to save the document
	err = Save(&doc, cfg)
	if err == nil {
		t.Errorf("Expected error for document exceeding size limit, but got none")
	}
}