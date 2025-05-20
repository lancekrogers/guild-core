package rag

import (
	"context"
	"fmt"
	
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
)

// Retriever provides methods for retrieving relevant context
type Retriever struct {
	Config      Config
	vectorStore vector.VectorStore
	embedder    vector.Embedder
	chunker     *Chunker
}

// SearchResult represents a search result
type SearchResult struct {
	Content  string
	Source   string
	Score    float32
	Metadata map[string]interface{}
}

// SearchResults contains search results and metadata
type SearchResults struct {
	Results []SearchResult
	Query   string
}

// NewRetriever creates a new Retriever
func NewRetriever(ctx context.Context, embedder vector.Embedder, config Config) (*Retriever, error) {
	// Default config values
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1000
	}
	
	if config.ChunkOverlap <= 0 {
		config.ChunkOverlap = 200
	}
	
	if config.MaxResults <= 0 {
		config.MaxResults = 5
	}
	
	// Create chunker
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:    config.ChunkSize,
		ChunkOverlap: config.ChunkOverlap,
		Strategy:     ChunkByParagraph,
	})
	
	// Create vector store config
	vsConfig := vector.Config{
		Embedder:         embedder,
		DefaultDimension: 1536, // Default for modern embeddings
	}
	
	// Create vector store
	vectorStore, err := vector.NewChromemStore(vsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}
	
	return &Retriever{
		Config:      config,
		vectorStore: vectorStore,
		embedder:    embedder,
		chunker:     chunker,
	}, nil
}

// RetrieveContext gets relevant context for a query
func (r *Retriever) RetrieveContext(ctx context.Context, query string, config RetrievalConfig) (*SearchResults, error) {
	// Use default max results if not specified
	if config.MaxResults <= 0 {
		config.MaxResults = r.Config.MaxResults
	}
	
	// Get vector store results
	matches, err := r.vectorStore.QueryEmbeddings(ctx, query, config.MaxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to query vector store: %w", err)
	}
	
	// Convert matches to search results
	results := &SearchResults{
		Query: query,
	}
	
	for _, match := range matches {
		if match.Score < config.MinScore {
			continue
		}
		
		result := SearchResult{
			Content: match.Text,
			Source:  match.Source,
			Score:   match.Score,
		}
		
		// Add metadata if requested
		if config.IncludeMetadata {
			result.Metadata = match.Metadata
		}
		
		results.Results = append(results.Results, result)
	}
	
	return results, nil
}

// Close closes the retriever and its resources
func (r *Retriever) Close() error {
	if r.vectorStore != nil {
		return r.vectorStore.Close()
	}
	return nil
}