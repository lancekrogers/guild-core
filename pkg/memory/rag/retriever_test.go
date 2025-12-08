// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"
	"strings"
	"testing"

	"github.com/guild-framework/guild-core/pkg/memory/vector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test NewRetrieverWithStore
func TestNewRetrieverWithStore(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "successful creation with defaults",
			config: Config{
				MaxResults:   5,
				ChunkSize:    1000,
				ChunkOverlap: 100,
			},
		},
		{
			name: "zero values get defaults",
			config: Config{
				MaxResults:   0, // Should become 5
				ChunkSize:    0, // Should become 1000
				ChunkOverlap: 0, // Should become 200
			},
		},
		{
			name: "with corpus config",
			config: Config{
				MaxResults:      10,
				ChunkSize:       500,
				ChunkOverlap:    50,
				UseCorpus:       true,
				CorpusPath:      "/test/corpus",
				CorpusMaxSizeMB: 1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockVectorStore{}

			retriever := NewRetrieverWithStore(mockStore, tt.config)

			assert.NotNil(t, retriever)
			assert.NotNil(t, retriever.chunker)
			assert.Equal(t, mockStore, retriever.vectorStore)

			// Check defaults were applied
			if tt.config.MaxResults == 0 {
				assert.Equal(t, 5, retriever.Config.MaxResults)
			}
			if tt.config.ChunkSize == 0 {
				assert.Equal(t, 1000, retriever.Config.ChunkSize)
			}
			if tt.config.ChunkOverlap == 0 {
				assert.Equal(t, 200, retriever.Config.ChunkOverlap)
			}
		})
	}
}

// Test RetrieveContext
func TestRetriever_RetrieveContext(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		config     RetrievalConfig
		setupMocks func(*mockVectorStore)
		wantErr    bool
		wantCount  int
	}{
		{
			name:  "successful retrieval",
			query: "How to implement authentication?",
			config: RetrievalConfig{
				MaxResults:      10,
				MinScore:        0.5,
				IncludeMetadata: true,
			},
			setupMocks: func(store *mockVectorStore) {
				// Mock search results via QueryEmbeddings
				matches := []vector.EmbeddingMatch{
					{
						ID:       "doc1",
						Text:     "Authentication can be implemented using JWT tokens...",
						Score:    0.9,
						Source:   "auth.md",
						Metadata: map[string]interface{}{"type": "documentation"},
					},
					{
						ID:     "doc2",
						Text:   "OAuth2 is another authentication method...",
						Score:  0.8,
						Source: "oauth.md",
					},
				}
				store.On("QueryEmbeddings", mock.Anything, "How to implement authentication?", 20).
					Return(matches, nil)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:  "filter by min score",
			query: "test query",
			config: RetrievalConfig{
				MaxResults: 10,
				MinScore:   0.7,
			},
			setupMocks: func(store *mockVectorStore) {
				matches := []vector.EmbeddingMatch{
					{
						ID:    "doc1",
						Text:  "High relevance content",
						Score: 0.9,
					},
					{
						ID:    "doc2",
						Text:  "Low relevance content",
						Score: 0.5, // Below min score
					},
				}
				store.On("QueryEmbeddings", mock.Anything, "test query", 20).
					Return(matches, nil)
			},
			wantErr:   false,
			wantCount: 1, // Only one result above min score
		},
		{
			name:  "empty query",
			query: "",
			config: RetrievalConfig{
				MaxResults: 10,
			},
			setupMocks: func(store *mockVectorStore) {
				// No mocks needed - should fail validation
			},
			wantErr: true,
		},
		{
			name:  "vector store error",
			query: "test query",
			config: RetrievalConfig{
				MaxResults: 10,
			},
			setupMocks: func(store *mockVectorStore) {
				store.On("QueryEmbeddings", mock.Anything, "test query", 20).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockVectorStore{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockStore)
			}

			retriever := &Retriever{
				Config: Config{
					MaxResults: 10,
				},
				vectorStore: mockStore,
			}

			ctx := context.Background()
			results, err := retriever.RetrieveContext(ctx, tt.query, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
				assert.Len(t, results.Results, tt.wantCount)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

// Test AddDocument
func TestRetriever_AddDocument(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		content string
		source  string
		wantErr bool
	}{
		{
			name:    "successful document addition",
			id:      "doc1",
			content: "This is a test document about Go programming.",
			source:  "test.md",
			wantErr: false,
		},
		{
			name:    "empty document ID",
			id:      "",
			content: "Content",
			source:  "test.md",
			wantErr: true,
		},
		{
			name:    "empty content",
			id:      "doc1",
			content: "",
			source:  "test.md",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockVectorStore{}

			// For successful cases, expect SaveEmbedding to be called for each chunk
			if !tt.wantErr && tt.content != "" {
				mockStore.On("SaveEmbedding", mock.Anything, mock.MatchedBy(func(emb vector.Embedding) bool {
					return strings.HasPrefix(emb.ID, tt.id+"_chunk_") &&
						emb.Source == tt.source &&
						emb.Text != ""
				})).Return(nil).Maybe() // Maybe because chunking might create 0 or more chunks
			}

			chunker := newChunker(ChunkerConfig{
				ChunkSize:    100,
				ChunkOverlap: 20,
				Strategy:     ChunkByParagraph,
			})

			retriever := &Retriever{
				Config: Config{
					ChunkSize:    100,
					ChunkOverlap: 20,
				},
				vectorStore: mockStore,
				chunker:     chunker,
			}

			ctx := context.Background()
			err := retriever.AddDocument(ctx, tt.id, tt.content, tt.source)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockStore.AssertExpectations(t)
		})
	}
}

// Test EnhancePrompt
func TestRetriever_EnhancePrompt(t *testing.T) {
	tests := []struct {
		name         string
		prompt       string
		config       RetrievalConfig
		setupMocks   func(*mockVectorStore)
		wantEnhanced bool
		wantErr      bool
	}{
		{
			name:   "successful prompt enhancement",
			prompt: "How to handle errors in Go?",
			config: RetrievalConfig{
				MaxResults: 10,
				MinScore:   0.5,
			},
			setupMocks: func(store *mockVectorStore) {
				matches := []vector.EmbeddingMatch{
					{
						ID:     "doc1",
						Text:   "In Go, errors are handled using the error interface...",
						Score:  0.9,
						Source: "go-errors.md",
					},
				}
				store.On("QueryEmbeddings", mock.Anything, "How to handle errors in Go?", 20).
					Return(matches, nil)
			},
			wantEnhanced: true,
			wantErr:      false,
		},
		{
			name:   "no relevant context found",
			prompt: "Random query with no matches",
			config: RetrievalConfig{
				MaxResults: 10,
				MinScore:   0.5,
			},
			setupMocks: func(store *mockVectorStore) {
				// Return empty results
				store.On("QueryEmbeddings", mock.Anything, "Random query with no matches", 20).
					Return([]vector.EmbeddingMatch{}, nil)
			},
			wantEnhanced: false,
			wantErr:      false,
		},
		{
			name:   "vector store error",
			prompt: "test query",
			config: RetrievalConfig{
				MaxResults: 10,
			},
			setupMocks: func(store *mockVectorStore) {
				store.On("QueryEmbeddings", mock.Anything, "test query", 20).
					Return(nil, assert.AnError)
			},
			wantEnhanced: false,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockVectorStore{}

			if tt.setupMocks != nil {
				tt.setupMocks(mockStore)
			}

			retriever := &Retriever{
				Config: Config{
					MaxResults: 10,
				},
				vectorStore: mockStore,
			}

			ctx := context.Background()
			enhanced, err := retriever.EnhancePrompt(ctx, tt.prompt, tt.config)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantEnhanced {
					assert.Contains(t, enhanced, "# Context")
					assert.Contains(t, enhanced, tt.prompt)
				} else {
					assert.Equal(t, tt.prompt, enhanced)
				}
			}

			mockStore.AssertExpectations(t)
		})
	}
}

// Test RemoveDocument
func TestRetriever_RemoveDocument(t *testing.T) {
	retriever := &Retriever{}

	ctx := context.Background()
	err := retriever.RemoveDocument(ctx, "doc1")

	// RemoveDocument is not yet implemented, should return error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
}

// Test Close
func TestRetriever_Close(t *testing.T) {
	mockStore := &mockVectorStore{}

	// Mock the close operation
	mockStore.On("Close").Return(nil)

	retriever := &Retriever{
		vectorStore: mockStore,
	}

	err := retriever.Close()

	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

// Mock implementations use the ones from rag_agent_test.go to avoid duplication
