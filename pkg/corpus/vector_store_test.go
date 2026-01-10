// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/memory/vector"
)

// mockEmbedder implements vector.Embedder for testing
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	// Create embeddings based on keywords in the text for more realistic testing
	embedding := make([]float32, 768)

	// Initialize with small base values
	for i := range embedding {
		embedding[i] = 0.1
	}

	// Add specific features for common keywords to simulate semantic similarity
	keywords := map[string]int{
		"golang":      50,
		"testing":     150,
		"patterns":    250,
		"database":    350,
		"guide":       450,
		"document":    550,
		"content":     650,
		"example":     700,
		"programming": 100,
		"strategies":  200,
	}

	textLower := strings.ToLower(text)
	for keyword, offset := range keywords {
		if strings.Contains(textLower, keyword) {
			// Add consistent boost for each keyword presence
			for i := 0; i < 15; i++ {
				if offset+i < len(embedding) {
					embedding[offset+i] += 50.0 // Consistent strong signal
				}
			}
		}
	}

	return embedding, nil
}

func setupTestVectorStore(t *testing.T) *CorpusVectorStore {
	// Create in-memory ChromemGo store
	config := vector.Config{
		Embedder:          &mockEmbedder{},
		DefaultCollection: "test_corpus",
	}

	chromemStore, err := vector.NewChromemStore(config)
	require.NoError(t, err)

	vsConfig := VectorStoreConfig{
		ChromemStore:   chromemStore,
		CollectionName: "test_corpus",
		ChunkSize:      100,
		ChunkOverlap:   20,
		Strategy:       ChunkingStrategyRecursive,
		MaxConcurrency: 1, // Use sequential processing for testing
	}

	store, err := NewCorpusVectorStore(vsConfig)
	require.NoError(t, err)

	return store
}

func TestNewCorpusVectorStore(t *testing.T) {
	t.Run("Missing ChromemStore", func(t *testing.T) {
		config := VectorStoreConfig{}
		_, err := NewCorpusVectorStore(config)
		assert.Error(t, err)
	})

	t.Run("Valid configuration", func(t *testing.T) {
		store := setupTestVectorStore(t)
		assert.NotNil(t, store)
		assert.Equal(t, "test_corpus", store.collectionName)
		assert.Equal(t, 100, store.chunkSize)
		assert.Equal(t, 20, store.chunkOverlap)
	})

	t.Run("Default values", func(t *testing.T) {
		chromemStore, _ := vector.NewChromemStore(vector.Config{Embedder: &mockEmbedder{}})
		config := VectorStoreConfig{
			ChromemStore: chromemStore,
		}
		store, err := NewCorpusVectorStore(config)
		require.NoError(t, err)

		assert.Equal(t, CorpusCollectionName, store.collectionName)
		assert.Equal(t, DefaultChunkSize, store.chunkSize)
		assert.Equal(t, DefaultChunkOverlap, store.chunkOverlap)
		assert.Equal(t, ChunkingStrategyRecursive, store.strategy)
		assert.Equal(t, 4, store.maxConcurrency)
	})
}

func TestChunkingStrategies(t *testing.T) {
	store := setupTestVectorStore(t)

	testDoc := &ScannedDocument{
		ID:   "test-doc",
		Path: "/test/doc.md",
		Type: ContentTypeMarkdown,
		Content: `# Test Document

This is the first paragraph with some content that should be chunked properly.

This is the second paragraph. It contains more text to ensure we have enough content for multiple chunks when using a small chunk size.

This is the third paragraph with additional content.`,
		Metadata: DocumentMetadata{
			Title: "Test Document",
		},
	}

	t.Run("Recursive chunking", func(t *testing.T) {
		store.strategy = ChunkingStrategyRecursive
		chunks, err := store.chunkDocument(context.Background(), testDoc)
		require.NoError(t, err)
		assert.Greater(t, len(chunks), 1) // Should create multiple chunks

		// Verify chunk properties
		for i, chunk := range chunks {
			assert.Equal(t, testDoc.ID, chunk.DocumentID)
			assert.Equal(t, testDoc.Path, chunk.DocumentPath)
			assert.Equal(t, i, chunk.ChunkIndex)
			assert.NotEmpty(t, chunk.Content)
			assert.Contains(t, chunk.ID, "chunk")
		}
	})

	t.Run("No chunking", func(t *testing.T) {
		store.strategy = ChunkingStrategyNone
		chunks, err := store.chunkDocument(context.Background(), testDoc)
		require.NoError(t, err)
		assert.Len(t, chunks, 1) // Should create single chunk
		assert.Equal(t, testDoc.Content, chunks[0].Content)
	})

	t.Run("Paragraph chunking", func(t *testing.T) {
		store.strategy = ChunkingStrategyParagraph
		chunks, err := store.chunkDocument(context.Background(), testDoc)
		require.NoError(t, err)

		// With small chunk size, should split paragraphs
		assert.GreaterOrEqual(t, len(chunks), 2)
	})
}

func TestMarkdownChunking(t *testing.T) {
	store := setupTestVectorStore(t)
	store.chunkSize = 200 // Larger chunks for this test

	markdownDoc := &ScannedDocument{
		ID:   "md-doc",
		Path: "/test/guide.md",
		Type: ContentTypeMarkdown,
		Content: `# Main Title

## Section 1

This is content in section 1. It has enough text to potentially be its own chunk.

## Section 2

This is content in section 2. It also has substantial content that could form a chunk.

### Subsection 2.1

Additional content in a subsection.

## Section 3

Final section with more content.`,
	}

	chunks, err := store.chunkMarkdown(markdownDoc)
	require.NoError(t, err)
	assert.Greater(t, len(chunks), 1)

	// Verify that chunks preserve structure
	for _, chunk := range chunks {
		assert.NotEmpty(t, chunk.Content)
		// Check that chunks don't start mid-sentence
		assert.NotEqual(t, " ", string(chunk.Content[0]))
	}
}

func TestCodeChunking(t *testing.T) {
	store := setupTestVectorStore(t)
	store.chunkSize = 150

	goDoc := &ScannedDocument{
		ID:   "go-doc",
		Path: "/test/main.go",
		Type: ContentTypeGo,
		Content: `package main

import "fmt"

func hello() {
    fmt.Println("Hello, World!")
}

func calculate(a, b int) int {
    result := a + b
    return result
}

func main() {
    hello()
    sum := calculate(5, 3)
    fmt.Printf("Sum: %d\n", sum)
}`,
	}

	chunks, err := store.chunkCode(goDoc)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(chunks), 2) // Should split at function boundaries

	// Verify functions are kept together when possible
	for _, chunk := range chunks {
		if strings.Contains(chunk.Content, "func hello") {
			assert.Contains(t, chunk.Content, "fmt.Println")
		}
	}
}

func TestIndexDocument(t *testing.T) {
	ctx := context.Background()
	store := setupTestVectorStore(t)

	doc := &ScannedDocument{
		ID:   "index-test",
		Path: "/test/index.md",
		Type: ContentTypeMarkdown,
		Content: `# Test Document

This is a test document for indexing. It contains enough content to create multiple chunks when processed.

## Section 1

More content here to ensure proper chunking behavior.`,
		Metadata: DocumentMetadata{
			Title:         "Test Document",
			WordCount:     20,
			LastModified:  time.Now(),
			ExtractedTags: []string{"test", "example"},
		},
		LastModified: time.Now(),
		Checksum:     "abc123",
	}

	err := store.IndexDocument(ctx, doc)
	assert.NoError(t, err)

	// Verify by searching
	results, err := store.SearchDocuments(ctx, "test document", 10)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Check result properties
	found := false
	for _, result := range results {
		if result.DocumentID == doc.ID {
			found = true
			assert.Equal(t, doc.Path, result.DocumentPath)
			assert.Equal(t, doc.Metadata.Title, result.Title)
			assert.NotEmpty(t, result.MatchedChunks)
			break
		}
	}
	assert.True(t, found, "Document should be found in search results")
}

func TestIndexDocuments(t *testing.T) {
	ctx := context.Background()
	store := setupTestVectorStore(t)

	docs := []*ScannedDocument{
		{
			ID:      "doc1",
			Path:    "/test/doc1.md",
			Type:    ContentTypeMarkdown,
			Content: "First document content about golang programming",
			Metadata: DocumentMetadata{
				Title: "Document 1",
			},
		},
		{
			ID:      "doc2",
			Path:    "/test/doc2.md",
			Type:    ContentTypeMarkdown,
			Content: "Second document content about testing strategies",
			Metadata: DocumentMetadata{
				Title: "Document 2",
			},
		},
		{
			ID:      "doc3",
			Path:    "/test/doc3.md",
			Type:    ContentTypeMarkdown,
			Content: "Third document content about golang testing",
			Metadata: DocumentMetadata{
				Title: "Document 3",
			},
		},
	}

	err := store.IndexDocuments(ctx, docs)
	assert.NoError(t, err)

	// Search for golang - should find doc1 and doc3
	results, err := store.SearchDocuments(ctx, "golang", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)

	// Search for testing - should find doc2 and doc3
	results, err = store.SearchDocuments(ctx, "testing", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)
}

func TestSearchDocuments(t *testing.T) {
	ctx := context.Background()
	store := setupTestVectorStore(t)

	// Index test documents
	docs := []*ScannedDocument{
		{
			ID:      "search1",
			Path:    "/docs/guide.md",
			Type:    ContentTypeMarkdown,
			Content: "Guild framework provides powerful agent orchestration capabilities",
			Metadata: DocumentMetadata{
				Title: "Guild Guide",
			},
		},
		{
			ID:      "search2",
			Path:    "/docs/agents.md",
			Type:    ContentTypeMarkdown,
			Content: "Agents in Guild can communicate and collaborate on complex tasks",
			Metadata: DocumentMetadata{
				Title: "Agent Documentation",
			},
		},
	}

	err := store.IndexDocuments(ctx, docs)
	require.NoError(t, err)

	t.Run("Search with results", func(t *testing.T) {
		results, err := store.SearchDocuments(ctx, "agent", 10)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Verify results are sorted by score
		for i := 1; i < len(results); i++ {
			assert.GreaterOrEqual(t, results[i-1].Score, results[i].Score)
		}
	})

	t.Run("Empty query", func(t *testing.T) {
		_, err := store.SearchDocuments(ctx, "", 10)
		assert.Error(t, err)
	})

	t.Run("Limit results", func(t *testing.T) {
		results, err := store.SearchDocuments(ctx, "guild", 1)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 1)
	})
}

func TestChunkMetadata(t *testing.T) {
	store := setupTestVectorStore(t)

	doc := &ScannedDocument{
		ID:   "meta-test",
		Path: "/test/meta.md",
		Type: ContentTypeMarkdown,
		Metadata: DocumentMetadata{
			Title:         "Metadata Test",
			Description:   "Testing metadata preservation",
			Language:      "en",
			ExtractedTags: []string{"test", "metadata"},
		},
	}

	metadata := store.createChunkMetadata(doc, 0)

	assert.Equal(t, store.collectionName, metadata["collection"])
	assert.Equal(t, doc.ID, metadata["document_id"])
	assert.Equal(t, doc.Path, metadata["document_path"])
	assert.Equal(t, string(doc.Type), metadata["content_type"])
	assert.Equal(t, 0, metadata["chunk_index"])
	assert.Equal(t, doc.Metadata.Title, metadata["title"])
	assert.Equal(t, doc.Metadata.Description, metadata["description"])
	assert.Equal(t, "test,metadata", metadata["tags"])
	assert.Equal(t, doc.Metadata.Language, metadata["language"])
	assert.NotEmpty(t, metadata["indexed_at"])
}

func TestConcurrentIndexing(t *testing.T) {
	ctx := context.Background()
	store := setupTestVectorStore(t)
	store.maxConcurrency = 3 // Test with limited concurrency

	// Create 10 documents
	var docs []*ScannedDocument
	for i := 0; i < 10; i++ {
		docs = append(docs, &ScannedDocument{
			ID:      fmt.Sprintf("concurrent-%d", i),
			Path:    fmt.Sprintf("/test/doc%d.md", i),
			Type:    ContentTypeMarkdown,
			Content: fmt.Sprintf("Document %d content with unique text for searching", i),
			Metadata: DocumentMetadata{
				Title: fmt.Sprintf("Document %d", i),
			},
		})
	}

	// Index concurrently
	err := store.IndexDocuments(ctx, docs)
	assert.NoError(t, err)

	// Verify all documents are indexed by searching for a common term
	results, err := store.SearchDocuments(ctx, "concurrent", 10)
	require.NoError(t, err)
	assert.Len(t, results, 10) // Should find all 10 documents

	// Verify we can find specific documents by their unique parts
	uniqueResults, err := store.SearchDocuments(ctx, "concurrent-0", 10)
	require.NoError(t, err)
	assert.NotEmpty(t, uniqueResults)

	// Check that all document IDs are represented
	foundIDs := make(map[string]bool)
	for _, result := range results {
		foundIDs[result.DocumentID] = true
	}
	assert.Len(t, foundIDs, 10) // Should have 10 unique document IDs
}

func TestEdgeCases(t *testing.T) {
	ctx := context.Background()
	store := setupTestVectorStore(t)

	t.Run("Very long document", func(t *testing.T) {
		// Create document that exceeds max chunks
		content := strings.Repeat("This is a long sentence. ", 1000)
		doc := &ScannedDocument{
			ID:      "long-doc",
			Path:    "/test/long.txt",
			Type:    ContentTypeText,
			Content: content,
		}

		store.chunkSize = 100 // Small chunks
		chunks, err := store.chunkDocument(ctx, doc)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(chunks), MaxChunksPerDocument)
	})

	t.Run("Empty document", func(t *testing.T) {
		doc := &ScannedDocument{
			ID:      "empty-doc",
			Path:    "/test/empty.md",
			Type:    ContentTypeMarkdown,
			Content: "",
		}

		chunks, err := store.chunkDocument(ctx, doc)
		require.NoError(t, err)
		assert.Empty(t, chunks) // Should not create chunks for empty content
	})

	t.Run("Whitespace only", func(t *testing.T) {
		doc := &ScannedDocument{
			ID:      "whitespace-doc",
			Path:    "/test/whitespace.md",
			Type:    ContentTypeMarkdown,
			Content: "   \n\n   \t\t   ",
		}

		chunks, err := store.chunkDocument(ctx, doc)
		require.NoError(t, err)
		assert.Empty(t, chunks) // Should not create chunks for whitespace
	})
}
