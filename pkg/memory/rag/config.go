package rag

// Config defines configuration for the RAG system
type Config struct {
	// CollectionName is the name of the vector store collection
	CollectionName string
	
	// ChunkSize is the size of text chunks in tokens
	ChunkSize int
	
	// ChunkOverlap is the number of tokens to overlap between chunks
	ChunkOverlap int
	
	// MaxResults is the maximum number of results to return from the vector store
	MaxResults int
}

// RetrievalConfig defines configuration for retrieval operations
type RetrievalConfig struct {
	// Query is the search query
	Query string
	
	// MaxResults is the maximum number of results to return
	MaxResults int
	
	// MinScore is the minimum similarity score required for results (0.0-1.0)
	MinScore float32
	
	// IncludeMetadata indicates whether to include metadata in results
	IncludeMetadata bool
	
	// UseCorpus indicates whether to include corpus documents in search
	UseCorpus bool
}
