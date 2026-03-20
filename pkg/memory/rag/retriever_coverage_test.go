// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lancekrogers/guild-core/pkg/corpus"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/memory/vector"
	"github.com/stretchr/testify/assert"
)

// Simple mock vector store for enhanced testing
type simpleMockVectorStore struct {
	embeddings map[string][]vector.Embedding
	shouldFail bool
}

func newSimpleMockVectorStore() *simpleMockVectorStore {
	return &simpleMockVectorStore{
		embeddings: make(map[string][]vector.Embedding),
	}
}

func (m *simpleMockVectorStore) SaveEmbedding(ctx context.Context, embedding vector.Embedding) error {
	if m.shouldFail {
		return fmt.Errorf("mock error")
	}
	if m.embeddings == nil {
		m.embeddings = make(map[string][]vector.Embedding)
	}
	// Use source as collection name for testing
	collection := "default"
	if embedding.Source != "" {
		collection = embedding.Source
	}
	m.embeddings[collection] = append(m.embeddings[collection], embedding)
	return nil
}

func (m *simpleMockVectorStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]vector.EmbeddingMatch, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("mock error")
	}

	// Return mock results
	results := []vector.EmbeddingMatch{}
	for _, embeddings := range m.embeddings {
		for i, emb := range embeddings {
			if i < limit {
				results = append(results, vector.EmbeddingMatch{
					ID:       emb.ID,
					Text:     emb.Text,
					Source:   emb.Source,
					Score:    0.8,
					Metadata: emb.Metadata,
				})
			}
		}
	}
	return results, nil
}

func (m *simpleMockVectorStore) QueryCollection(ctx context.Context, collectionName, query string, limit int) ([]vector.EmbeddingMatch, error) {
	if m.shouldFail {
		return nil, errors.New("mock query error")
	}
	return []vector.EmbeddingMatch{}, nil
}

func (m *simpleMockVectorStore) DeleteEmbedding(ctx context.Context, id string) error {
	if m.shouldFail {
		return errors.New("mock delete error")
	}
	return nil
}

func (m *simpleMockVectorStore) Close() error {
	return nil
}

// Test newRetriever with different configurations
func TestNewRetriever_AllPaths(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		embedder  vector.Embedder
		config    Config
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Nil embedder",
			embedder:  nil,
			config:    Config{CollectionName: "test"},
			wantError: true,
			errorMsg:  "embedder cannot be nil",
		},
		{
			name:      "Valid with default config",
			embedder:  &MockEmbedder{},
			config:    Config{},
			wantError: false,
		},
		{
			name:     "Valid with paragraph strategy",
			embedder: &MockEmbedder{},
			config: Config{
				CollectionName: "test",
				ChunkStrategy:  "paragraph",
				ChunkSize:      2000,
				ChunkOverlap:   200,
			},
			wantError: false,
		},
		{
			name:     "Valid with fixed strategy",
			embedder: &MockEmbedder{},
			config: Config{
				CollectionName: "test",
				ChunkStrategy:  "fixed",
				ChunkSize:      1000,
				ChunkOverlap:   100,
			},
			wantError: false,
		},
		{
			name:     "Valid with markdown_header strategy",
			embedder: &MockEmbedder{},
			config: Config{
				CollectionName: "test",
				ChunkStrategy:  "markdown_header",
				ChunkSize:      3000,
				ChunkOverlap:   300,
			},
			wantError: false,
		},
		{
			name:     "With corpus config",
			embedder: &MockEmbedder{},
			config: Config{
				CollectionName: "test",
				UseCorpus:      true,
				CorpusPath:     "/test/path",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retriever, err := newRetriever(ctx, tt.embedder, tt.config)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, retriever)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, retriever)
				assert.NotNil(t, retriever.vectorStore)
				assert.NotNil(t, retriever.chunker)
				assert.Equal(t, tt.embedder, retriever.embedder)

				// Clean up
				if retriever != nil {
					retriever.Close()
				}
			}
		})
	}
}

// Test RetrieveContext with various scenarios
func TestRetriever_RetrieveContext_Advanced(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	config := Config{
		CollectionName: "test_retrieve",
		ChunkSize:      100,
		ChunkOverlap:   20,
	}

	// Use mock vector store instead of ChromemGo for predictable results
	mockStore := newSimpleMockVectorStore()
	retriever := &Retriever{
		Config:      config,
		vectorStore: mockStore,
		embedder:    embedder,
		chunker: newChunker(ChunkerConfig{
			ChunkSize:    config.ChunkSize,
			ChunkOverlap: config.ChunkOverlap,
		}),
	}

	// Add test documents
	docs := []struct {
		id      string
		content string
		source  string
	}{
		{"doc1", "Machine learning algorithms", "ml.txt"},
		{"doc2", "Deep learning neural networks", "dl.txt"},
		{"doc3", "Natural language processing", "nlp.txt"},
	}

	for _, doc := range docs {
		err := retriever.AddDocument(ctx, doc.id, doc.content, doc.source)
		assert.NoError(t, err)
	}

	// Test different retrieval configurations
	tests := []struct {
		name     string
		query    string
		config   RetrievalConfig
		wantDocs int
	}{
		{
			name:  "Basic retrieval",
			query: "learning",
			config: RetrievalConfig{
				MaxResults: 10,
				MinScore:   0.0,
			},
			wantDocs: 3, // Mock returns all documents
		},
		{
			name:  "With high min score",
			query: "learning",
			config: RetrievalConfig{
				MaxResults: 10,
				MinScore:   0.9,
			},
			wantDocs: 0, // No results above 0.9
		},
		{
			name:  "Limited results",
			query: "processing",
			config: RetrievalConfig{
				MaxResults: 1,
				MinScore:   0.0,
			},
			wantDocs: 1,
		},
		{
			name:  "Empty query",
			query: "",
			config: RetrievalConfig{
				MaxResults: 10,
				MinScore:   0.0,
			},
			wantDocs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := retriever.RetrieveContext(ctx, tt.query, tt.config)

			if tt.query == "" {
				// Empty query should return an error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "query cannot be empty")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
				assert.Len(t, results.Results, tt.wantDocs)
			}
		})
	}
}

// Test AddDocument error paths
func TestRetriever_AddDocument_Errors(t *testing.T) {
	ctx := context.Background()

	// Test with nil vector store
	retriever := &Retriever{
		vectorStore: nil,
		chunker: newChunker(ChunkerConfig{
			ChunkSize:    100,
			ChunkOverlap: 20,
		}),
	}

	err := retriever.AddDocument(ctx, "doc1", "content", "test.txt")
	assert.Error(t, err)

	// Test with failing vector store
	mockStore := &simpleMockVectorStore{shouldFail: true}
	retriever.vectorStore = mockStore
	retriever.embedder = &MockEmbedder{}
	retriever.Config.CollectionName = "test"

	err = retriever.AddDocument(ctx, "doc2", "content", "test.txt")
	assert.Error(t, err)
}

// Test Close method with error handling
func TestRetriever_Close_ErrorHandling(t *testing.T) {
	retriever := &Retriever{}

	// Close with nil vector store
	err := retriever.Close()
	assert.NoError(t, err)

	// Close with mock vector store
	retriever.vectorStore = newSimpleMockVectorStore()
	err = retriever.Close()
	assert.NoError(t, err)
}

// Test searchCorpus error scenarios
func TestRetriever_SearchCorpus_Errors(t *testing.T) {
	ctx := context.Background()

	// Test without corpus config
	retriever := &Retriever{
		Config: Config{
			UseCorpus: true,
		},
		corpusConfig: nil,
	}

	results, err := retriever.searchCorpus(ctx, "test", 5)
	assert.NoError(t, err)
	assert.Nil(t, results) // Returns nil when no corpus config

	// Test with invalid corpus path - corpus.List returns empty list
	retriever.corpusConfig = &corpus.Config{
		CorpusPath: "/invalid/path/that/does/not/exist",
	}

	results, err = retriever.searchCorpus(ctx, "test", 5)
	assert.NoError(t, err)
	assert.Empty(t, results) // No documents found in invalid path
}

// Test enhanceRequestWithRAG error handling
func TestEnhanceRequestWithRAG_Errors(t *testing.T) {
	ctx := context.Background()

	// Test with retriever that fails
	mockStore := &simpleMockVectorStore{shouldFail: true}
	retriever := &Retriever{
		vectorStore: mockStore,
		embedder:    &MockEmbedder{},
		Config: Config{
			CollectionName: "test",
		},
	}

	wrapper := &AgentWrapper{
		retriever: retriever,
		config:    DefaultConfig(),
	}

	result, err := wrapper.enhanceRequestWithRAG(ctx, "test request")
	assert.Error(t, err)
	assert.Empty(t, result)
	assert.True(t, gerror.Is(err, gerror.ErrCodeStorage))
}

// Test AgentWrapper EnhancePrompt error handling
func TestAgentWrapper_EnhancePrompt_Errors(t *testing.T) {
	ctx := context.Background()

	// Test with failing retriever
	mockStore := &simpleMockVectorStore{shouldFail: true}
	retriever := &Retriever{
		vectorStore: mockStore,
		embedder:    &MockEmbedder{},
		Config: Config{
			CollectionName: "test",
		},
	}

	wrapper := &AgentWrapper{
		retriever: retriever,
		config:    DefaultConfig(),
	}

	_, err := wrapper.EnhancePrompt(ctx, "prompt", "query", RetrievalConfig{
		MaxResults: 3,
	})
	assert.Error(t, err)
	assert.True(t, gerror.Is(err, gerror.ErrCodeStorage))
}

// Test example usage functions (for coverage)
func TestExampleUsageFunctions(t *testing.T) {
	// These are example functions that demonstrate usage
	// We'll test them to improve coverage

	// Note: These functions typically have complex setup requirements
	// and interact with external services, so we'll skip them in unit tests
	t.Skip("Example usage functions are for documentation purposes")
}
