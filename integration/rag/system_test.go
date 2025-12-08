//go:build disabled

//  This test file uses outdated APIs and needs to be rewritten to match current RAG implementation

package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/guild-framework/guild-core/internal/testutil"
	"github.com/guild-framework/guild-core/pkg/memory/rag"
	"github.com/guild-framework/guild-core/pkg/memory/vector"
	"github.com/guild-framework/guild-core/pkg/providers/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRAGSystem validates comprehensive RAG functionality
func TestRAGSystem(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name:       "rag-system-test",
		WithCorpus: true,
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Run("corpus_creation", func(t *testing.T) {
		// Create test documents
		documents := []struct {
			filename string
			content  string
		}{
			{
				"api_guide.md",
				`# API Development Guide
				
## REST API Best Practices
- Use proper HTTP methods (GET, POST, PUT, DELETE)
- Return appropriate status codes
- Include pagination for list endpoints
- Version your APIs

## Authentication
All API requests require authentication using Bearer tokens.
`,
			},
			{
				"database_design.md",
				`# Database Design Principles

## Normalization
Follow database normalization rules to reduce redundancy.

## Indexing Strategy
- Index foreign keys
- Index columns used in WHERE clauses
- Avoid over-indexing
`,
			},
			{
				"security_guide.md",
				`# Security Best Practices

## Input Validation
Always validate and sanitize user input.

## Authentication & Authorization
- Use strong password policies
- Implement proper session management
- Follow principle of least privilege
`,
			},
		}

		// Write documents to corpus
		corpusDir := filepath.Join(projCtx.GetRootPath(), "corpus")
		err := os.MkdirAll(corpusDir, 0755)
		require.NoError(t, err)

		for _, doc := range documents {
			path := filepath.Join(corpusDir, doc.filename)
			err := os.WriteFile(path, []byte(doc.content), 0644)
			require.NoError(t, err)
		}

		// Verify corpus created
		files, err := os.ReadDir(corpusDir)
		require.NoError(t, err)
		assert.Equal(t, len(documents), len(files))
	})

	t.Run("indexing_performance", func(t *testing.T) {
		// Create larger corpus for performance testing
		numDocs := 50
		corpusDir := filepath.Join(projCtx.GetRootPath(), "large_corpus")
		err := os.MkdirAll(corpusDir, 0755)
		require.NoError(t, err)

		// Generate documents
		for i := 0; i < numDocs; i++ {
			content := fmt.Sprintf(`# Document %d

This is test document number %d containing various technical content.
It includes information about programming, databases, security, and more.
The purpose is to test RAG indexing performance with multiple documents.

## Section 1
Lorem ipsum dolor sit amet, consectetur adipiscing elit.

## Section 2  
Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
`, i, i)

			path := filepath.Join(corpusDir, fmt.Sprintf("doc_%03d.md", i))
			err := os.WriteFile(path, []byte(content), 0644)
			require.NoError(t, err)
		}

		// Measure indexing time
		start := time.Now()

		// Simulate indexing (in real implementation, this would call RAG indexer)
		factory := rag.NewFactory()
		ragSystem, err := factory.CreateRAGAgent(ctx, rag.Config{
			ChunkSize:    500,
			ChunkOverlap: 50,
		})
		require.NoError(t, err)

		// Index documents
		for i := 0; i < numDocs; i++ {
			path := filepath.Join(corpusDir, fmt.Sprintf("doc_%03d.md", i))
			content, err := os.ReadFile(path)
			require.NoError(t, err)

			err = ragSystem.IndexDocument(ctx, path, string(content))
			assert.NoError(t, err)
		}

		duration := time.Since(start)
		docsPerSecond := float64(numDocs) / duration.Seconds()

		// Performance requirement: >= 10 docs/second
		assert.GreaterOrEqual(t, docsPerSecond, 10.0,
			"RAG indexing should process at least 10 docs/second, got %.2f", docsPerSecond)

		t.Logf("Indexed %d documents in %v (%.2f docs/sec)", numDocs, duration, docsPerSecond)
	})

	t.Run("search_accuracy", func(t *testing.T) {
		// Test search functionality
		queries := []struct {
			query    string
			expected []string
		}{
			{
				query:    "API authentication",
				expected: []string{"api_guide.md", "security_guide.md"},
			},
			{
				query:    "database indexing",
				expected: []string{"database_design.md"},
			},
			{
				query:    "security validation",
				expected: []string{"security_guide.md"},
			},
		}

		factory := rag.NewFactory()
		ragSystem, err := factory.CreateRAGAgent(ctx, rag.Config{})
		require.NoError(t, err)

		// Index test corpus
		corpusDir := filepath.Join(projCtx.GetRootPath(), "corpus")
		files, err := os.ReadDir(corpusDir)
		require.NoError(t, err)

		for _, file := range files {
			if filepath.Ext(file.Name()) == ".md" {
				path := filepath.Join(corpusDir, file.Name())
				content, err := os.ReadFile(path)
				require.NoError(t, err)

				err = ragSystem.IndexDocument(ctx, file.Name(), string(content))
				require.NoError(t, err)
			}
		}

		// Test searches
		for _, tc := range queries {
			t.Run(tc.query, func(t *testing.T) {
				results, err := ragSystem.Search(ctx, tc.query, 5)
				require.NoError(t, err)

				// Verify expected documents appear in results
				foundDocs := make(map[string]bool)
				for _, result := range results {
					foundDocs[result.DocumentID] = true
				}

				for _, expectedDoc := range tc.expected {
					assert.True(t, foundDocs[expectedDoc],
						"Expected %s in results for query '%s'", expectedDoc, tc.query)
				}
			})
		}
	})

	t.Run("incremental_updates", func(t *testing.T) {
		factory := rag.NewFactory()
		ragSystem, err := factory.CreateRAGAgent(ctx, rag.Config{})
		require.NoError(t, err)

		// Index initial document
		doc1 := "initial_doc.md"
		content1 := "This is the initial document content about testing."
		err = ragSystem.IndexDocument(ctx, doc1, content1)
		require.NoError(t, err)

		// Search should find it
		results, err := ragSystem.Search(ctx, "initial testing", 5)
		require.NoError(t, err)
		assert.Len(t, results, 1)

		// Update document
		content2 := "This is the updated document content about testing and validation."
		err = ragSystem.UpdateDocument(ctx, doc1, content2)
		require.NoError(t, err)

		// Search for new content
		results, err = ragSystem.Search(ctx, "validation", 5)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].Content, "validation")
	})

	t.Run("concurrent_indexing", func(t *testing.T) {
		factory := rag.NewFactory()
		ragSystem, err := factory.CreateRAGAgent(ctx, rag.Config{})
		require.NoError(t, err)

		// Index multiple documents concurrently
		numDocs := 20
		var wg sync.WaitGroup
		errors := make(chan error, numDocs)

		for i := 0; i < numDocs; i++ {
			wg.Add(1)
			go func(docNum int) {
				defer wg.Done()

				docID := fmt.Sprintf("concurrent_doc_%d.md", docNum)
				content := fmt.Sprintf("Concurrent document %d with test content.", docNum)

				if err := ragSystem.IndexDocument(ctx, docID, content); err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		var errorCount int
		for err := range errors {
			t.Logf("Concurrent indexing error: %v", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount, "Concurrent indexing should not produce errors")

		// Verify all documents indexed
		results, err := ragSystem.Search(ctx, "concurrent", 25)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), numDocs, "All concurrent documents should be indexed")
	})

	t.Run("memory_efficiency", func(t *testing.T) {
		// Test memory usage with large documents
		factory := rag.NewFactory()
		ragSystem, err := factory.CreateRAGAgent(ctx, rag.Config{
			ChunkSize:    1000,
			ChunkOverlap: 100,
		})
		require.NoError(t, err)

		// Create a large document (1MB)
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte('A' + (i % 26))
		}

		// Index large document
		err = ragSystem.IndexDocument(ctx, "large_doc.txt", string(largeContent))
		assert.NoError(t, err, "Should handle large documents efficiently")

		// Search should still work
		results, err := ragSystem.Search(ctx, "ABCDEF", 5)
		require.NoError(t, err)
		assert.NotEmpty(t, results, "Should find content in large document")
	})
}
