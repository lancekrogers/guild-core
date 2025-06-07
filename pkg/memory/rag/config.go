// Package rag provides configuration structures for the Retrieval-Augmented Generation system.
// These configurations control how documents are chunked, stored, and retrieved.
package rag

// Config defines configuration for the RAG system.
// This configuration is used when creating a new Retriever instance.
type Config struct {
	// CollectionName is the name of the vector store collection.
	// Different collections can be used to separate different types of content.
	// Default: "rag_embeddings"
	CollectionName string

	// ChunkSize is the size of text chunks in characters (not tokens).
	// Larger chunks provide more context but may be less precise.
	// Default: 1000
	ChunkSize int

	// ChunkOverlap is the number of characters to overlap between chunks.
	// Overlap helps maintain context across chunk boundaries.
	// Default: 200
	ChunkOverlap int

	// ChunkStrategy defines how documents are chunked.
	// Options: "paragraph", "sentence", "fixed", "markdown"
	// Default: "paragraph"
	ChunkStrategy string

	// MaxResults is the default maximum number of results to return.
	// This can be overridden per query.
	// Default: 5
	MaxResults int

	// VectorStorePath is the path to persist the vector store.
	// If empty, the vector store will be in-memory only.
	VectorStorePath string

	// UseCorpus enables corpus integration for retrieval.
	// When true, the retriever will also search corpus documents.
	UseCorpus bool

	// CorpusPath is the path to the corpus directory.
	// Required if UseCorpus is true.
	CorpusPath string

	// CorpusMaxSizeMB is the maximum size of the corpus in megabytes.
	// Used to enforce storage limits.
	// Default: 1000 (1GB)
	CorpusMaxSizeMB int
}

// RetrievalConfig defines configuration for retrieval operations.
// This configuration is passed to the RetrieveContext method.
type RetrievalConfig struct {
	// Query is the search query (deprecated - pass as parameter instead)
	Query string

	// MaxResults is the maximum number of results to return.
	// Overrides the default from Config.
	MaxResults int

	// MinScore is the minimum similarity score required for results.
	// Range: 0.0 to 1.0, where 1.0 is a perfect match.
	// Default: 0.0 (no minimum)
	MinScore float32

	// IncludeMetadata indicates whether to include metadata in results.
	// Metadata can include document IDs, chunk indices, tags, etc.
	IncludeMetadata bool

	// UseCorpus indicates whether to include corpus documents in search.
	// Overrides the default from Config.
	UseCorpus bool

	// DisableVectorSearch disables vector similarity search.
	// Useful when you only want corpus results.
	DisableVectorSearch bool
}

// DefaultConfig returns a Config with sensible defaults.
// This is a good starting point for most use cases.
func DefaultConfig() Config {
	return Config{
		CollectionName:  "rag_embeddings",
		ChunkSize:       1000,
		ChunkOverlap:    200,
		ChunkStrategy:   "paragraph",
		MaxResults:      5,
		UseCorpus:       false,
		CorpusMaxSizeMB: 1000,
	}
}

// DefaultRetrievalConfig returns a RetrievalConfig with sensible defaults.
func DefaultRetrievalConfig() RetrievalConfig {
	return RetrievalConfig{
		MaxResults:      5,
		MinScore:        0.0,
		IncludeMetadata: false,
		UseCorpus:       false,
	}
}
