package corpus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildGraph(t *testing.T) {
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

	// Create test documents with links
	docs := []CorpusDoc{
		{
			Title:     "Document 1",
			Body:      "This links to [[Document 2]] and [[Document 3]].",
			Tags:      []string{"tag1", "tag2"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Document 2",
			Body:      "This links to [[Document 3]].",
			Tags:      []string{"tag2"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Document 3",
			Body:      "This links back to [[Document 1]].",
			Tags:      []string{"tag1", "tag3"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Document 4",
			Body:      "This has no links.",
			Tags:      []string{"tag4"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Save the documents
	for i := range docs {
		err = Save(&docs[i], cfg)
		if err != nil {
			t.Fatalf("Failed to save document %d: %v", i, err)
		}
	}

	// Build the graph
	graph, err := BuildGraph(cfg)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Verify the graph
	if len(graph.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(graph.Nodes))
	}

	// Count the edges
	expectedEdges := 4 // Document 1 -> 2, 1 -> 3, 2 -> 3, 3 -> 1
	if len(graph.Edges) != expectedEdges {
		t.Errorf("Expected %d edges, got %d", expectedEdges, len(graph.Edges))
	}

	// Verify tag relationships
	expectedTagLinks := map[string][]string{
		"tag1": {"Document 1", "Document 3"},
		"tag2": {"Document 1", "Document 2"},
		"tag3": {"Document 3"},
		"tag4": {"Document 4"},
	}

	for tag, expectedDocs := range expectedTagLinks {
		if !contains(graph.Tags, tag) {
			t.Errorf("Tag %s not found in graph tags", tag)
			continue
		}

		for _, docTitle := range expectedDocs {
			found := false
			for _, edge := range graph.TagLinks {
				if edge.Tag == tag && edge.Document == docTitle {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected tag link from %s to %s not found", tag, docTitle)
			}
		}
	}

	// Test GetBacklinks
	backlinks := graph.GetBacklinks("Document 3")
	if len(backlinks) != 2 {
		t.Errorf("Expected 2 backlinks for Document 3, got %d", len(backlinks))
	}

	expectedBacklinks := []string{"Document 1", "Document 2"}
	for _, expectedBacklink := range expectedBacklinks {
		if !contains(backlinks, expectedBacklink) {
			t.Errorf("Expected backlink %s not found", expectedBacklink)
		}
	}

	// Test GetDocumentsWithTag
	docsWithTag := graph.GetDocumentsWithTag("tag1")
	if len(docsWithTag) != 2 {
		t.Errorf("Expected 2 documents with tag1, got %d", len(docsWithTag))
	}

	expectedDocsWithTag := []string{"Document 1", "Document 3"}
	for _, expectedDoc := range expectedDocsWithTag {
		if !contains(docsWithTag, expectedDoc) {
			t.Errorf("Expected document %s not found in docs with tag1", expectedDoc)
		}
	}
}

// Helper function to check if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}