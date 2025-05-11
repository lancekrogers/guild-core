# Implement RAG System with Chromem-go

This document outlines how to implement the Retrieval-Augmented Generation (RAG) system for Guild using the Chromem-go embeddable vector database.

## Overview

Guild's RAG system enhances LLM prompts with relevant context retrieved from stored documents, embeddings, and agent interactions. We're implementing this using Chromem-go, a pure Go embeddable vector database with zero dependencies.

## Why Chromem-go

After careful evaluation, we've selected Chromem-go for these key reasons:
- Pure Go implementation (no CGO, no external dependencies)
- Embeddable directly in the application (no separate service to maintain)
- Simple, Chroma-like API that's easy to work with
- Sufficient performance for Guild's typical use cases
- Aligns with Guild's philosophy of minimizing external dependencies

## Implementation Steps

### 1. Add Chromem-go Dependency

```bash
go get github.com/philippgille/chromem-go
```

### 2. Create Vector Store Interface Implementation

Create a file at `pkg/memory/vector/chromem.go`:

```go
package vector

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"

	"github.com/lancerogers/guild/pkg/memory"
)

// ChromemStore implements the VectorStore interface using Chromem-go
type ChromemStore struct {
	db       *chromem.DB
	embedder Embedder
}

// Config contains Chromem configuration
type Config struct {
	Embedder         Embedder
	PersistencePath  string // Optional path for persistence
	DefaultDimension int    // Default dimension for vectors
}

// NewChromemStore creates a new Chromem store
func NewChromemStore(config Config) (*ChromemStore, error) {
	// Set defaults
	if config.DefaultDimension == 0 {
		config.DefaultDimension = 1536 // Default for OpenAI embeddings
	}

	// Configure options
	opts := []chromem.Option{
		chromem.WithDefaultDimension(config.DefaultDimension),
	}

	// Add persistence if path provided
	if config.PersistencePath != "" {
		opts = append(opts, chromem.WithPersistence(config.PersistencePath))
	}

	// Create DB
	db, err := chromem.NewDB(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Chromem DB: %w", err)
	}

	return &ChromemStore{
		db:       db,
		embedder: config.Embedder,
	}, nil
}

// SaveEmbedding stores a vector embedding
func (s *ChromemStore) SaveEmbedding(ctx context.Context, embedding memory.Embedding) error {
	// Generate ID if not provided
	if embedding.ID == "" {
		embedding.ID = uuid.New().String()
	}

	// Generate vector if not provided
	vector := embedding.Vector
	if len(vector) == 0 && s.embedder != nil {
		var err error
		vector, err = s.embedder.Embed(ctx, embedding.Text)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}
	}

	// Create collection if it doesn't exist
	collection := "default"
	if src, ok := embedding.Metadata["collection"]; ok {
		if colName, ok := src.(string); ok && colName != "" {
			collection = colName
		}
	}

	// Create metadata
	metadata := map[string]any{
		"text":      embedding.Text,
		"source":    embedding.Source,
		"timestamp": embedding.Timestamp.Format(time.RFC3339),
	}

	// Add custom metadata
	for k, v := range embedding.Metadata {
		if k != "collection" { // Skip collection as we already used it
			metadata[k] = v
		}
	}

	// Add the embedding
	err := s.db.UpsertEmbedding(ctx, collection, embedding.ID, vector, metadata)
	if err != nil {
		return fmt.Errorf("failed to upsert embedding: %w", err)
	}

	return nil
}

// QueryEmbeddings performs a similarity search
func (s *ChromemStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]memory.EmbeddingMatch, error) {
	// Generate query vector
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Set limit
	if limit <= 0 {
		limit = 10
	}

	// Search (in all collections)
	results, err := s.db.QueryAllCollections(ctx, vector, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}

	// Convert results
	matches := make([]memory.EmbeddingMatch, 0, len(results))
	for _, result := range results {
		match := memory.EmbeddingMatch{
			ID:       result.ID,
			Score:    result.Score,
			Metadata: make(map[string]interface{}),
		}

		// Extract fields from metadata
		if text, ok := result.Metadata["text"].(string); ok {
			match.Text = text
		}
		if source, ok := result.Metadata["source"].(string); ok {
			match.Source = source
		}
		if ts, ok := result.Metadata["timestamp"].(string); ok {
			timestamp, err := time.Parse(time.RFC3339, ts)
			if err == nil {
				match.Timestamp = timestamp
			}
		}

		// Extract other metadata
		for k, v := range result.Metadata {
			if k == "text" || k == "source" || k == "timestamp" {
				continue
			}
			match.Metadata[k] = v
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// QueryCollection performs a similarity search within a specific collection
func (s *ChromemStore) QueryCollection(ctx context.Context, collection, query string, limit int) ([]memory.EmbeddingMatch, error) {
	// Generate query vector
	vector, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Set limit
	if limit <= 0 {
		limit = 10
	}

	// Search
	results, err := s.db.Query(ctx, collection, vector, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Convert results
	matches := make([]memory.EmbeddingMatch, 0, len(results))
	for _, result := range results {
		match := memory.EmbeddingMatch{
			ID:       result.ID,
			Score:    result.Score,
			Metadata: make(map[string]interface{}),
		}

		// Extract fields from metadata
		if text, ok := result.Metadata["text"].(string); ok {
			match.Text = text
		}
		if source, ok := result.Metadata["source"].(string); ok {
			match.Source = source
		}
		if ts, ok := result.Metadata["timestamp"].(string); ok {
			timestamp, err := time.Parse(time.RFC3339, ts)
			if err == nil {
				match.Timestamp = timestamp
			}
		}

		// Extract other metadata
		for k, v := range result.Metadata {
			if k == "text" || k == "source" || k == "timestamp" {
				continue
			}
			match.Metadata[k] = v
		}

		matches = append(matches, match)
	}

	return matches, nil
}

// DeleteEmbedding removes an embedding
func (s *ChromemStore) DeleteEmbedding(ctx context.Context, collection, id string) error {
	err := s.db.DeleteEmbedding(ctx, collection, id)
	if err != nil {
		return fmt.Errorf("failed to delete embedding: %w", err)
	}
	return nil
}

// ListCollections returns all collections
func (s *ChromemStore) ListCollections(ctx context.Context) ([]string, error) {
	collections, err := s.db.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return collections, nil
}

// Close closes the database
func (s *ChromemStore) Close() error {
	return s.db.Close()
}
```

### 3. Implement the RAG System

Create a file at `pkg/memory/rag/rag.go`:

```go
package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/lancerogers/guild/pkg/corpus"
	"github.com/lancerogers/guild/pkg/memory"
)

// Retriever provides retrieval-augmented generation capabilities
type Retriever struct {
	vectorStore memory.VectorStore
	corpusConfig corpus.Config
}

// RetrievalConfig defines configuration options for the retriever
type RetrievalConfig struct {
	MaxResults     int
	MinScore       float32
	IncludeCorpus  bool
	IncludePrompts bool
}

// DefaultRetrievalConfig returns a default configuration
func DefaultRetrievalConfig() RetrievalConfig {
	return RetrievalConfig{
		MaxResults:     5,
		MinScore:       0.7,
		IncludeCorpus:  true,
		IncludePrompts: true,
	}
}

// NewRetriever creates a new RAG retriever
func NewRetriever(vectorStore memory.VectorStore, corpusConfig corpus.Config) *Retriever {
	return &Retriever{
		vectorStore: vectorStore,
		corpusConfig: corpusConfig,
	}
}

// RetrieveContext gets relevant context for a query
func (r *Retriever) RetrieveContext(ctx context.Context, query string, config RetrievalConfig) (string, error) {
	// Use default config if needed
	if config.MaxResults == 0 {
		config = DefaultRetrievalConfig()
	}

	// Search for relevant content from embeddings
	matches, err := r.vectorStore.QueryEmbeddings(ctx, query, config.MaxResults)
	if err != nil {
		return "", fmt.Errorf("failed to query embeddings: %w", err)
	}

	// Filter by minimum score
	if config.MinScore > 0 {
		filteredMatches := matches[:0]
		for _, match := range matches {
			if match.Score >= config.MinScore {
				filteredMatches = append(filteredMatches, match)
			}
		}
		matches = filteredMatches
	}

	// Sort by score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// If including corpus documents, search and merge with embeddings
	if config.IncludeCorpus {
		corpusMatches, err := r.searchCorpus(ctx, query, config.MaxResults)
		if err == nil && len(corpusMatches) > 0 {
			// Merge and re-sort
			allMatches := append(matches, corpusMatches...)
			sort.Slice(allMatches, func(i, j int) bool {
				return allMatches[i].Score > allMatches[j].Score
			})
			
			// Limit to max results
			if len(allMatches) > config.MaxResults {
				allMatches = allMatches[:config.MaxResults]
			}
			
			matches = allMatches
		}
	}

	// Build context string
	var builder strings.Builder
	builder.WriteString("# Relevant Context\n\n")

	for i, match := range matches {
		// Add metadata for source and relevance
		source := match.Source
		if source == "" {
			source = "Unknown"
		}
		
		builder.WriteString(fmt.Sprintf("## Source %d: %s (Relevance: %.2f)\n\n", 
			i+1, source, match.Score))
		
		// Add content
		builder.WriteString(match.Text)
		builder.WriteString("\n\n")
	}

	return builder.String(), nil
}

// searchCorpus searches the corpus documents and returns them as embedding matches
func (r *Retriever) searchCorpus(ctx context.Context, query string, limit int) ([]memory.EmbeddingMatch, error) {
	// List all corpus docs
	docs, err := corpus.List(r.corpusConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list corpus documents: %w", err)
	}
	
	// Simple keyword matching (for a basic implementation)
	// In a real implementation, we might want to also use semantic search via embeddings
	queryLower := strings.ToLower(query)
	var matches []memory.EmbeddingMatch
	
	for _, doc := range docs {
		// Load full document contents
		fullDoc, err := corpus.Load(doc.FilePath)
		if err != nil {
			continue
		}
		
		// Simple keyword matching in title, tags, and content
		titleMatch := strings.Contains(strings.ToLower(fullDoc.Title), queryLower)
		contentMatch := strings.Contains(strings.ToLower(fullDoc.Body), queryLower)
		
		tagMatch := false
		for _, tag := range fullDoc.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				tagMatch = true
				break
			}
		}
		
		if titleMatch || contentMatch || tagMatch {
			// Calculate a simple score based on match location
			// This is a very basic scoring approach
			score := float32(0.7) // Base score
			if titleMatch {
				score += 0.2 // Title matches are more relevant
			}
			if tagMatch {
				score += 0.1 // Tag matches are somewhat relevant
			}
			
			// Create an embedding match from corpus document
			match := memory.EmbeddingMatch{
				ID:       fullDoc.FilePath,
				Text:     fullDoc.Body,
				Source:   "Corpus: " + fullDoc.Title,
				Score:    score,
				Timestamp: fullDoc.UpdatedAt,
				Metadata: map[string]interface{}{
					"title": fullDoc.Title,
					"tags":  fullDoc.Tags,
				},
			}
			
			matches = append(matches, match)
		}
	}
	
	// Sort by score and limit results
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})
	
	if len(matches) > limit {
		matches = matches[:limit]
	}
	
	return matches, nil
}

// EnhancePrompt adds retrieved context to a prompt
func (r *Retriever) EnhancePrompt(ctx context.Context, prompt, query string, config RetrievalConfig) (string, error) {
	// Retrieve relevant context
	context, err := r.RetrieveContext(ctx, query, config)
	if err != nil {
		// Log the error but continue with original prompt
		return prompt, nil
	}

	// Combine context with original prompt
	enhanced := fmt.Sprintf(`
# Retrieved Context
%s

# Original Prompt
%s
`, context, prompt)

	return enhanced, nil
}

// EnhanceAgent adds RAG capabilities to an agent
func (r *Retriever) EnhanceAgent(agent Agent) Agent {
	// This would wrap the agent with RAG capabilities
	// Implementation will depend on the Agent interface
	return &RAGAgent{
		agent:     agent,
		retriever: r,
	}
}

// RAGAgent wraps an agent with RAG capabilities
type RAGAgent struct {
	agent     Agent
	retriever *Retriever
}

// Define the Agent interface methods for the RAGAgent
// This is a placeholder - actual implementation will depend on the Agent interface
```

### 4. Implement the Chunker for Document Processing

Create a file at `pkg/memory/rag/chunker.go`:

```go
package rag

import (
	"fmt"
	"strings"
)

// Chunker handles document chunking for RAG
type Chunker struct {
	ChunkSize     int
	ChunkOverlap  int
	SplitStrategy string
}

// NewChunker creates a new document chunker
func NewChunker(chunkSize, chunkOverlap int, strategy string) *Chunker {
	if chunkSize <= 0 {
		chunkSize = 1000 // Default chunk size
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 10 // Default 10% overlap
	}
	if strategy == "" {
		strategy = "paragraph" // Default strategy
	}

	return &Chunker{
		ChunkSize:     chunkSize,
		ChunkOverlap:  chunkOverlap,
		SplitStrategy: strategy,
	}
}

// ChunkDocument breaks a document into chunks
func (c *Chunker) ChunkDocument(text string) []string {
	switch c.SplitStrategy {
	case "paragraph":
		return c.chunkByParagraph(text)
	case "sentence":
		return c.chunkBySentence(text)
	case "fixed":
		return c.chunkByFixedSize(text)
	case "markdown":
		return c.chunkByMarkdownSection(text)
	default:
		return c.chunkByParagraph(text)
	}
}

// chunkByParagraph chunks text by paragraphs
func (c *Chunker) chunkByParagraph(text string) []string {
	// Split by double newlines (paragraphs)
	paragraphs := strings.Split(text, "\n\n")
	
	var chunks []string
	var currentChunk strings.Builder
	currentSize := 0
	
	for _, para := range paragraphs {
		paraSize := len(para)
		
		// If paragraph is too large on its own, split it further
		if paraSize > c.ChunkSize {
			// Add current chunk if not empty
			if currentSize > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentSize = 0
			}
			
			// Split large paragraph into fixed-size chunks
			paraChunks := c.chunkByFixedSize(para)
			chunks = append(chunks, paraChunks...)
			continue
		}
		
		// If adding this paragraph would exceed chunk size, start a new chunk
		if currentSize + paraSize > c.ChunkSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentChunk.WriteString(para)
			currentSize = paraSize
		} else {
			// Add to current chunk
			if currentSize > 0 {
				currentChunk.WriteString("\n\n")
				currentSize += 2
			}
			currentChunk.WriteString(para)
			currentSize += paraSize
		}
	}
	
	// Add the last chunk if not empty
	if currentSize > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}

// Other chunking methods (implementation details omitted for brevity)
func (c *Chunker) chunkBySentence(text string) []string {
	// Implementation would split by sentence boundaries
	// This is a placeholder - actual implementation would be more sophisticated
	return c.chunkByFixedSize(text)
}

func (c *Chunker) chunkByFixedSize(text string) []string {
	var chunks []string
	
	// For simple fixed-size chunking
	textRunes := []rune(text)
	textLen := len(textRunes)
	
	for i := 0; i < textLen; i += c.ChunkSize - c.ChunkOverlap {
		end := i + c.ChunkSize
		if end > textLen {
			end = textLen
		}
		
		chunk := string(textRunes[i:end])
		chunks = append(chunks, chunk)
		
		// Break if we've reached the end
		if end == textLen {
			break
		}
	}
	
	return chunks
}

func (c *Chunker) chunkByMarkdownSection(text string) []string {
	// Split by markdown headers
	lines := strings.Split(text, "\n")
	var chunks []string
	var currentChunk strings.Builder
	currentSize := 0
	
	for _, line := range lines {
		lineSize := len(line)
		
		// If line is a header, start a new chunk (unless current is empty)
		if strings.HasPrefix(line, "#") {
			if currentSize > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentChunk.WriteString(line)
				currentSize = lineSize
			} else {
				currentChunk.WriteString(line)
				currentSize = lineSize
			}
		} else if currentSize + lineSize > c.ChunkSize {
			// If adding this line would exceed chunk size, start a new chunk
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentChunk.WriteString(line)
			currentSize = lineSize
		} else {
			// Add to current chunk
			if currentSize > 0 {
				currentChunk.WriteString("\n")
				currentSize++
			}
			currentChunk.WriteString(line)
			currentSize += lineSize
		}
	}
	
	// Add the last chunk if not empty
	if currentSize > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	
	return chunks
}
```

### 5. Create Factory for Vector Store Implementation

Create a file at `pkg/memory/vector/factory.go`:

```go
package vector

import (
	"errors"
	"fmt"
	"os"

	"github.com/lancerogers/guild/pkg/memory"
)

// StoreType defines the type of vector store to use
type StoreType string

const (
	StoreTypeChromem StoreType = "chromem"
	StoreTypeQdrant  StoreType = "qdrant"
)

// FactoryConfig contains configuration for the vector store factory
type FactoryConfig struct {
	StoreType        StoreType
	EmbedderType     string
	ChromemConfig    ChromemConfig
	QdrantConfig     QdrantConfig
	OpenAIApiKey     string
	AnthropicApiKey  string
}

// ChromemConfig contains Chromem-specific configuration
type ChromemConfig struct {
	PersistencePath  string
	DefaultDimension int
}

// QdrantConfig contains Qdrant-specific configuration
type QdrantConfig struct {
	Address    string
	Collection string
	VectorSize uint64
	UseSSL     bool
}

// Factory creates vector stores
type Factory struct {
	config FactoryConfig
}

// NewFactory creates a new vector store factory
func NewFactory(config FactoryConfig) *Factory {
	// Set defaults
	if config.StoreType == "" {
		config.StoreType = StoreTypeChromem // Default to Chromem
	}
	if config.EmbedderType == "" {
		config.EmbedderType = "openai" // Default to OpenAI
	}
	if config.OpenAIApiKey == "" {
		config.OpenAIApiKey = os.Getenv("OPENAI_API_KEY")
	}
	if config.AnthropicApiKey == "" {
		config.AnthropicApiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	// Set Chromem defaults
	if config.ChromemConfig.DefaultDimension == 0 {
		config.ChromemConfig.DefaultDimension = 1536
	}

	// Set Qdrant defaults
	if config.QdrantConfig.Address == "" {
		config.QdrantConfig.Address = "localhost:6334"
	}
	if config.QdrantConfig.Collection == "" {
		config.QdrantConfig.Collection = "guild_embeddings"
	}
	if config.QdrantConfig.VectorSize == 0 {
		config.QdrantConfig.VectorSize = 1536
	}

	return &Factory{
		config: config,
	}
}

// CreateVectorStore creates a vector store based on configuration
func (f *Factory) CreateVectorStore() (memory.VectorStore, error) {
	// Create embedder
	embedder, err := f.createEmbedder()
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Create store based on type
	switch f.config.StoreType {
	case StoreTypeChromem:
		config := Config{
			Embedder:        embedder,
			PersistencePath: f.config.ChromemConfig.PersistencePath,
			DefaultDimension: f.config.ChromemConfig.DefaultDimension,
		}
		return NewChromemStore(config)

	case StoreTypeQdrant:
		config := QdrantConfig{
			Address:    f.config.QdrantConfig.Address,
			Collection: f.config.QdrantConfig.Collection,
			VectorSize: f.config.QdrantConfig.VectorSize,
			UseSSL:     f.config.QdrantConfig.UseSSL,
			Embedder:   embedder,
		}
		return NewQdrantStore(config)

	default:
		return nil, errors.New("unsupported vector store type")
	}
}

// createEmbedder creates an embedder based on configuration
func (f *Factory) createEmbedder() (Embedder, error) {
	switch f.config.EmbedderType {
	case "openai":
		if f.config.OpenAIApiKey == "" {
			return nil, errors.New("OpenAI API key is required")
		}
		return NewOpenAIEmbedder(f.config.OpenAIApiKey, "")

	case "anthropic":
		if f.config.AnthropicApiKey == "" {
			return nil, errors.New("Anthropic API key is required")
		}
		// Implement Anthropic embedder when their embedding API is available
		return nil, errors.New("Anthropic embeddings not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported embedder type: %s", f.config.EmbedderType)
	}
}
```

### 6. Update the VectorStore Interface

Update the file at `pkg/memory/vector/interface.go`:

```go
package vector

import (
	"context"

	"github.com/lancerogers/guild/pkg/memory"
)

// VectorStore provides storage and retrieval of vector embeddings
type VectorStore interface {
	// SaveEmbedding stores a vector embedding
	SaveEmbedding(ctx context.Context, embedding memory.Embedding) error

	// QueryEmbeddings performs a similarity search
	QueryEmbeddings(ctx context.Context, query string, limit int) ([]memory.EmbeddingMatch, error)

	// Close closes the store
	Close() error
}

// Embedder generates embeddings from text
type Embedder interface {
	// Embed generates an embedding from text
	Embed(ctx context.Context, text string) ([]float32, error)
}
```

## Integration with Agent System

The RAG system integrates with agents through a wrapper that enhances prompts with relevant context:

```go
// Example integration with an agent
type RAGAgent struct {
	agent     Agent          // Original agent
	retriever *rag.Retriever // RAG retriever
	config    rag.RetrievalConfig
}

// ExecutePrompt enhances the prompt with context before delegating to the wrapped agent
func (a *RAGAgent) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	// Enhance the prompt with relevant context
	enhancedPrompt, err := a.retriever.EnhancePrompt(ctx, prompt, prompt, a.config)
	if err != nil {
		// Log the error but continue with original prompt
		return a.agent.ExecutePrompt(ctx, prompt)
	}
	
	// Execute the enhanced prompt
	return a.agent.ExecutePrompt(ctx, enhancedPrompt)
}
```

## Configuration

Update your application configuration to initialize the RAG system:

```go
// In your main.go or initialization code
func initializeRAG() (*rag.Retriever, error) {
	// Create vector store factory
	factory := vector.NewFactory(vector.FactoryConfig{
		StoreType: vector.StoreTypeChromem,
		EmbedderType: "openai",
		ChromemConfig: vector.ChromemConfig{
			PersistencePath: "./data/vector_store",
			DefaultDimension: 1536,
		},
	})
	
	// Create vector store
	vectorStore, err := factory.CreateVectorStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}
	
	// Get corpus config
	corpusConfig := corpus.DefaultConfig()
	
	// Create retriever
	retriever := rag.NewRetriever(vectorStore, corpusConfig)
	
	return retriever, nil
}
```

## Testing

Create tests to verify the RAG implementation:

```go
// pkg/memory/rag/rag_test.go
package rag_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancerogers/guild/pkg/corpus"
	"github.com/lancerogers/guild/pkg/memory"
	"github.com/lancerogers/guild/pkg/memory/rag"
	"github.com/lancerogers/guild/pkg/memory/vector"
)

func TestRetriever_RetrieveContext(t *testing.T) {
	// Create a mock vector store
	vectorStore := &mockVectorStore{
		embeddings: []memory.EmbeddingMatch{
			{
				ID:    "1",
				Text:  "Guild is a framework for orchestrating AI agents.",
				Source: "documentation",
				Score: 0.9,
			},
			{
				ID:    "2",
				Text:  "Agents use tools to perform tasks.",
				Source: "documentation",
				Score: 0.8,
			},
		},
	}
	
	// Create retriever
	retriever := rag.NewRetriever(vectorStore, corpus.DefaultConfig())
	
	// Test context retrieval
	ctx := context.Background()
	config := rag.DefaultRetrievalConfig()
	
	result, err := retriever.RetrieveContext(ctx, "How do agents work?", config)
	
	// Assert
	require.NoError(t, err)
	assert.Contains(t, result, "Guild is a framework")
	assert.Contains(t, result, "Agents use tools")
}

// mockVectorStore implements the VectorStore interface for testing
type mockVectorStore struct {
	embeddings []memory.EmbeddingMatch
}

func (m *mockVectorStore) SaveEmbedding(ctx context.Context, embedding memory.Embedding) error {
	return nil
}

func (m *mockVectorStore) QueryEmbeddings(ctx context.Context, query string, limit int) ([]memory.EmbeddingMatch, error) {
	return m.embeddings, nil
}

func (m *mockVectorStore) Close() error {
	return nil
}
```

## Best Practices

1. **Chunking Strategy**: Choose chunking strategies based on document type:
   - Markdown: Split by headers
   - Code: Split by functions or classes
   - General text: Split by paragraphs

2. **Embedding Caching**: Cache embeddings for frequently used text to reduce API costs

3. **Performance Monitoring**: Track retrieval latency and relevance metrics

4. **Progressive Enhancement**: Fall back gracefully if RAG system encounters errors

5. **Collection Organization**: Use separate collections for different document types (agent memories, corpus documents, etc.)