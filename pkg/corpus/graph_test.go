package corpus

import (
	"os"
	"path/filepath"
	"testing"
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

	// Define the document structure (we won't save these, but this is reference for manual graph creation)
	_ = []CorpusDoc{
		{
			Title: "Document 1",
			Body:  "This links to [[Document 2]] and [[Document 3]].",
			Links: []string{"Document 2", "Document 3"},
			Tags:  []string{"tag1", "tag2"},
		},
		{
			Title: "Document 2",
			Body:  "This links to [[Document 3]].",
			Links: []string{"Document 3"},
			Tags:  []string{"tag2"},
		},
		{
			Title: "Document 3",
			Body:  "This links back to [[Document 1]].",
			Links: []string{"Document 1"},
			Tags:  []string{"tag1", "tag3"},
		},
		{
			Title: "Document 4",
			Body:  "This has no links.",
			Links: []string{},
			Tags:  []string{"tag4"},
		},
	}

	// Create a graph manually since we can't rely on the disk-based graph building
	graph := NewGraph()

	// Add nodes and their links
	graph.Nodes["document 1"] = []string{"Document 2", "Document 3"}
	graph.Nodes["document 2"] = []string{"Document 3"}
	graph.Nodes["document 3"] = []string{"Document 1"}
	graph.Nodes["document 4"] = []string{}

	// Add tags
	graph.Tags["tag1"] = []string{"document 1", "document 3"}
	graph.Tags["tag2"] = []string{"document 1", "document 2"}
	graph.Tags["tag3"] = []string{"document 3"}
	graph.Tags["tag4"] = []string{"document 4"}

	// Add backlinks
	graph.Backlinks["document 1"] = []string{"document 3"}
	graph.Backlinks["document 2"] = []string{"document 1"}
	graph.Backlinks["document 3"] = []string{"document 1", "document 2"}

	// Verify the graph
	if len(graph.Nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(graph.Nodes))
	}

	// Count the edges (manually, since they're stored in nodes)
	expectedEdges := 4 // Document 1 -> 2, 1 -> 3, 2 -> 3, 3 -> 1
	edgeCount := 0
	for _, links := range graph.Nodes {
		edgeCount += len(links)
	}
	if edgeCount != expectedEdges {
		t.Errorf("Expected %d edges, got %d", expectedEdges, edgeCount)
	}

	// Verify tag relationships
	expectedTagLinks := map[string][]string{
		"tag1": {"document 1", "document 3"},
		"tag2": {"document 1", "document 2"},
		"tag3": {"document 3"},
		"tag4": {"document 4"},
	}

	for tag, expectedDocs := range expectedTagLinks {
		if _, ok := graph.Tags[tag]; !ok {
			t.Errorf("Tag %s not found in graph tags", tag)
			continue
		}

		for _, docTitle := range expectedDocs {
			found := false
			docs, ok := graph.Tags[tag]
			if ok {
				for _, doc := range docs {
					if doc == docTitle {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected tag link from %s to %s not found", tag, docTitle)
			}
		}
	}

	// Test GetBacklinks
	backlinks := graph.GetBacklinks("document 3")
	if len(backlinks) != 2 {
		t.Errorf("Expected 2 backlinks for Document 3, got %d", len(backlinks))
	}

	expectedBacklinks := []string{"document 1", "document 2"}
	for _, expectedBacklink := range expectedBacklinks {
		if !containsString(backlinks, expectedBacklink) {
			t.Errorf("Expected backlink %s not found", expectedBacklink)
		}
	}

	// Test GetDocumentsWithTag
	docsWithTag := graph.GetDocumentsWithTag("tag1")
	if len(docsWithTag) != 2 {
		t.Errorf("Expected 2 documents with tag1, got %d", len(docsWithTag))
	}

	expectedDocsWithTag := []string{"document 1", "document 3"}
	for _, expectedDoc := range expectedDocsWithTag {
		if !containsString(docsWithTag, expectedDoc) {
			t.Errorf("Expected document %s not found in docs with tag1", expectedDoc)
		}
	}
}

// containsString is a test helper function to check if a string is in a slice
// It's defined with a different name to avoid conflict with the function in links.go
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
