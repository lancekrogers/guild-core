package corpus

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCorpusBasics(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "corpus-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test corpus config
	cfg := Config{
		CorpusPath:      filepath.Join(tempDir, "corpus"),
		ActivitiesPath:  filepath.Join(tempDir, "activities"),
		MaxSizeBytes:    1024 * 1024 * 10, // 10MB
		DefaultTags:     []string{"test"},
		DefaultCategory: "general",
		Location:        filepath.Join(tempDir, "corpus"), // For backward compatibility
		MaxSizeMB:       10,                               // For backward compatibility
	}

	// Ensure the corpus directory exists
	err = os.MkdirAll(cfg.CorpusPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create corpus directory: %v", err)
	}

	// Create a test document
	doc := NewCorpusDoc(
		"Test Document",
		"Test Source",
		"# Test Document\n\nThis is a test document with [[Test Link]].",
		"test-guild",
		"test-agent",
		[]string{"test", "document"},
	)

	// Save the document
	ctx := context.Background()
	err = Save(ctx, doc, cfg)
	if err != nil {
		t.Fatalf("Failed to save document: %v", err)
	}

	// Check that the document was saved
	if doc.FilePath == "" {
		t.Error("Document FilePath should not be empty")
	}

	// Load the document
	loadedDoc, err := Load(ctx, doc.FilePath)
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

	if len(loadedDoc.Links) != 1 || loadedDoc.Links[0] != "Test Link" {
		t.Errorf("Expected links [Test Link], got %v", loadedDoc.Links)
	}

	// List all documents
	docs, err := List(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("Expected 1 document, got %d", len(docs))
	}

	// Delete the document
	err = Delete(ctx, doc.FilePath)
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Check that the document was deleted
	_, err = os.Stat(doc.FilePath)
	if !os.IsNotExist(err) {
		t.Error("Document file should not exist after deletion")
	}

	// List all documents after deletion
	docs, err = List(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to list documents after deletion: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("Expected 0 documents after deletion, got %d", len(docs))
	}
}
