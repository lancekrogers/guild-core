// Package rag interfaces define contracts for the Retrieval-Augmented Generation system.
package rag

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
)

// RetrieverInterface defines the contract for retrieving relevant context
// from both vector embeddings and corpus documents.
type RetrieverInterface interface {
	// RetrieveContext gets relevant context for a query by searching both
	// vector embeddings and corpus documents.
	RetrieveContext(ctx context.Context, query string, config RetrievalConfig) (*SearchResults, error)

	// AddDocument adds a document to the vector store after chunking.
	AddDocument(ctx context.Context, id, content, source string) error

	// AddCorpusDocument indexes a corpus document in the vector store.
	AddCorpusDocument(ctx context.Context, doc *corpus.CorpusDoc) error

	// EnhancePrompt adds retrieved context to a prompt for better LLM responses.
	EnhancePrompt(ctx context.Context, prompt string, config RetrievalConfig) (string, error)

	// RemoveDocument removes a document and all its chunks from the vector store.
	RemoveDocument(ctx context.Context, documentID string) error

	// Close closes the retriever and its resources.
	Close() error
}

// ChunkerInterface defines the contract for document chunking strategies.
type ChunkerInterface interface {
	// ChunkDocument breaks a document into smaller chunks for processing.
	ChunkDocument(content string) []string

	// ChunkWithMetadata chunks a document and returns chunks with metadata.
	ChunkWithMetadata(content string) []ChunkWithMeta

	// GetConfig returns the chunker configuration.
	GetConfig() ChunkerConfig
}

// FactoryInterface creates RAG components using the registry pattern.
type FactoryInterface interface {
	// GetRetriever returns a configured retriever instance.
	GetRetriever() RetrieverInterface

	// GetEmbedder returns the embedder used by this factory.
	GetEmbedder() vector.Embedder

	// Close closes the factory and all its resources.
	Close() error
}

// Registry provides access to RAG components through the registry pattern.
type Registry interface {
	// GetRetriever returns a retriever instance for the given configuration.
	GetRetriever(config Config) (RetrieverInterface, error)

	// GetChunker returns a chunker instance for the given configuration.
	GetChunker(config ChunkerConfig) (ChunkerInterface, error)

	// GetFactory returns a factory instance for the given configuration.
	GetFactory(ctx context.Context, embedder vector.Embedder, config Config) (FactoryInterface, error)

	// RegisterRetriever registers a custom retriever implementation.
	RegisterRetriever(name string, factory func(ctx context.Context, embedder vector.Embedder, config Config) (RetrieverInterface, error)) error

	// RegisterChunker registers a custom chunker implementation.
	RegisterChunker(name string, factory func(config ChunkerConfig) (ChunkerInterface, error)) error
}

// Ensure concrete types implement interfaces
var (
	_ RetrieverInterface = (*Retriever)(nil)
	_ ChunkerInterface   = (*Chunker)(nil)
	_ FactoryInterface   = (*Factory)(nil)
)
