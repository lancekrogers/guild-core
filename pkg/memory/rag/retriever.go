// Package rag implements Retrieval-Augmented Generation for the Guild framework.
// The retriever component is responsible for finding relevant context from the
// vector store and corpus to enhance LLM prompts with appropriate information.
package rag

import (
	"context"
	"fmt"
	"strings"
	
	"github.com/guild-ventures/guild-core/pkg/corpus"
	"github.com/guild-ventures/guild-core/pkg/memory/vector"
)

// Retriever provides methods for retrieving relevant context from both
// vector embeddings and the corpus system. It integrates with the Guild
// framework's knowledge management systems to provide context-aware responses.
//
// The retriever supports:
// - Vector similarity search across embeddings
// - Corpus document search and retrieval
// - Configurable chunking strategies
// - Score-based filtering
// - Metadata enrichment
type Retriever struct {
	// Config holds the retriever configuration
	Config      Config
	
	// vectorStore is the underlying vector database for embeddings
	vectorStore vector.VectorStore
	
	// embedder generates embeddings for queries
	embedder    vector.Embedder
	
	// chunker breaks documents into searchable chunks
	chunker     *Chunker
	
	// corpusConfig is the corpus system configuration (optional)
	corpusConfig *corpus.Config
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

// NewRetriever creates a new Retriever with the given configuration.
// The retriever integrates with both the vector store and corpus systems
// to provide comprehensive context retrieval.
//
// Example:
//   config := rag.Config{
//       ChunkSize: 1000,
//       MaxResults: 5,
//       UseCorpus: true,
//   }
//   retriever, err := rag.NewRetriever(ctx, embedder, config)
func NewRetriever(ctx context.Context, embedder vector.Embedder, config Config) (*Retriever, error) {
	// Validate embedder
	if embedder == nil {
		return nil, fmt.Errorf("embedder cannot be nil")
	}
	
	// Apply default config values
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1000
	}
	
	if config.ChunkOverlap <= 0 {
		config.ChunkOverlap = 200
	}
	
	if config.MaxResults <= 0 {
		config.MaxResults = 5
	}
	
	// Create chunker based on strategy
	chunkerStrategy := ChunkByParagraph
	if config.ChunkStrategy != "" {
		// Map string strategy to chunker constant
		switch strings.ToLower(config.ChunkStrategy) {
		case "sentence":
			chunkerStrategy = ChunkBySentence
		case "fixed":
			chunkerStrategy = ChunkByFixed
		case "markdown":
			chunkerStrategy = ChunkByMarkdown
		default:
			chunkerStrategy = ChunkByParagraph
		}
	}
	
	chunker := NewChunker(ChunkerConfig{
		ChunkSize:    config.ChunkSize,
		ChunkOverlap: config.ChunkOverlap,
		Strategy:     chunkerStrategy,
	})
	
	// Create vector store config
	vsConfig := vector.Config{
		Embedder:         embedder,
		DefaultDimension: 1536, // Default for modern embeddings
		DefaultCollection: "rag_embeddings",
	}
	
	// Apply vector store persistence if configured
	if config.VectorStorePath != "" {
		vsConfig.PersistencePath = config.VectorStorePath
	}
	
	// Create vector store
	vectorStore, err := vector.NewChromemStore(vsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}
	
	// Create retriever
	retriever := &Retriever{
		Config:      config,
		vectorStore: vectorStore,
		embedder:    embedder,
		chunker:     chunker,
	}
	
	// Configure corpus integration if enabled
	if config.UseCorpus && config.CorpusPath != "" {
		corpusConfig := &corpus.Config{
			Location:  config.CorpusPath,
			MaxSizeMB: config.CorpusMaxSizeMB,
		}
		retriever.corpusConfig = corpusConfig
	}
	
	return retriever, nil
}

// RetrieveContext gets relevant context for a query by searching both
// vector embeddings and corpus documents. Results are merged and ranked
// by relevance score.
//
// The retrieval process:
// 1. Search vector store for similar embeddings
// 2. Optionally search corpus documents if enabled
// 3. Merge and rank results by score
// 4. Apply score threshold filtering
// 5. Return top N results
func (r *Retriever) RetrieveContext(ctx context.Context, query string, config RetrievalConfig) (*SearchResults, error) {
	// Validate query
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	
	// Use default max results if not specified
	if config.MaxResults <= 0 {
		config.MaxResults = r.Config.MaxResults
	}
	
	// Initialize results
	results := &SearchResults{
		Query:   query,
		Results: make([]SearchResult, 0),
	}
	
	// Get vector store results
	if !config.DisableVectorSearch {
		matches, err := r.vectorStore.QueryEmbeddings(ctx, query, config.MaxResults*2) // Get extra for filtering
		if err != nil {
			// Log error but continue with corpus search if available
			// This makes the system more resilient
			if r.corpusConfig == nil || !config.UseCorpus {
				return nil, fmt.Errorf("failed to query vector store: %w", err)
			}
		} else {
			// Convert vector matches to search results
			for _, match := range matches {
				// Apply minimum score filter
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
		}
	}
	
	// Search corpus if enabled and configured
	if config.UseCorpus && r.corpusConfig != nil {
		corpusResults, err := r.searchCorpus(ctx, query, config.MaxResults)
		if err != nil {
			// Log but don't fail - corpus search is supplementary
			// In production, you'd log this error
		} else {
			// Merge corpus results with vector results
			results.Results = append(results.Results, corpusResults...)
		}
	}
	
	// Sort all results by score (highest first)
	r.sortResultsByScore(results.Results)
	
	// Limit to requested number of results
	if len(results.Results) > config.MaxResults {
		results.Results = results.Results[:config.MaxResults]
	}
	
	return results, nil
}

// searchCorpus searches the corpus documents for relevant content.
// This provides an additional source of context beyond vector embeddings.
func (r *Retriever) searchCorpus(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if r.corpusConfig == nil {
		return nil, nil
	}
	
	// List corpus documents
	docs, err := corpus.List(*r.corpusConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list corpus documents: %w", err)
	}
	
	results := make([]SearchResult, 0)
	queryLower := strings.ToLower(query)
	
	// Simple keyword matching for now
	// TODO: Implement more sophisticated corpus search
	for _, docPath := range docs {
		doc, err := corpus.Load(docPath)
		if err != nil {
			continue
		}
		
		// Calculate relevance score based on keyword matches
		score := r.calculateCorpusScore(doc, queryLower)
		if score > 0 {
			result := SearchResult{
				Content: doc.Body,
				Source:  fmt.Sprintf("corpus:%s", doc.Title),
				Score:   score,
				Metadata: map[string]interface{}{
					"title": doc.Title,
					"tags":  doc.Tags,
					"path":  docPath,
				},
			}
			results = append(results, result)
		}
	}
	
	// Sort corpus results by score
	r.sortResultsByScore(results)
	
	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}
	
	return results, nil
}

// calculateCorpusScore calculates a relevance score for a corpus document.
// Higher scores indicate better matches.
func (r *Retriever) calculateCorpusScore(doc *corpus.CorpusDoc, queryLower string) float32 {
	score := float32(0.0)
	
	// Title match is highly relevant
	if strings.Contains(strings.ToLower(doc.Title), queryLower) {
		score += 0.5
	}
	
	// Tag matches are moderately relevant
	for _, tag := range doc.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			score += 0.3
			break
		}
	}
	
	// Body content matches
	bodyLower := strings.ToLower(doc.Body)
	matches := strings.Count(bodyLower, queryLower)
	if matches > 0 {
		// Normalize by document length to avoid bias toward longer documents
		normalizedScore := float32(matches) / float32(len(doc.Body)) * 100
		score += normalizedScore
		
		// Cap the body score contribution
		if score > 1.0 {
			score = 1.0
		}
	}
	
	return score
}

// sortResultsByScore sorts search results by score in descending order
func (r *Retriever) sortResultsByScore(results []SearchResult) {
	// Simple insertion sort for small result sets
	for i := 1; i < len(results); i++ {
		key := results[i]
		j := i - 1
		
		for j >= 0 && results[j].Score < key.Score {
			results[j+1] = results[j]
			j--
		}
		results[j+1] = key
	}
}

// AddDocument adds a document to the vector store after chunking.
// This is used to index new content for retrieval.
//
// Example:
//   err := retriever.AddDocument(ctx, "doc1", "Guild framework documentation...", "docs")
func (r *Retriever) AddDocument(ctx context.Context, id, content, source string) error {
	// Validate inputs
	if id == "" {
		return fmt.Errorf("document ID cannot be empty")
	}
	if content == "" {
		return fmt.Errorf("document content cannot be empty")
	}
	
	// Chunk the document
	chunks := r.chunker.Chunk(content)
	
	// Add each chunk to the vector store
	for i, chunk := range chunks {
		embedding := vector.Embedding{
			ID:     fmt.Sprintf("%s_chunk_%d", id, i),
			Text:   chunk,
			Source: source,
			Metadata: map[string]interface{}{
				"document_id": id,
				"chunk_index": i,
				"total_chunks": len(chunks),
			},
		}
		
		if err := r.vectorStore.SaveEmbedding(ctx, embedding); err != nil {
			return fmt.Errorf("failed to save chunk %d: %w", i, err)
		}
	}
	
	return nil
}

// AddCorpusDocument indexes a corpus document in the vector store.
// This allows corpus documents to be found through vector similarity search
// in addition to keyword search.
func (r *Retriever) AddCorpusDocument(ctx context.Context, doc *corpus.CorpusDoc) error {
	if doc == nil {
		return fmt.Errorf("corpus document cannot be nil")
	}
	
	// Create a searchable representation of the document
	searchableContent := fmt.Sprintf("%s\n\n%s", doc.Title, doc.Body)
	
	// Add tags to the content for better searchability
	if len(doc.Tags) > 0 {
		searchableContent = fmt.Sprintf("Tags: %s\n\n%s", 
			strings.Join(doc.Tags, ", "), searchableContent)
	}
	
	// Generate document ID
	docID := fmt.Sprintf("corpus_%s_%s", doc.GuildID, doc.AgentID)
	if doc.Title != "" {
		// Use sanitized title as part of ID
		sanitizedTitle := strings.ReplaceAll(strings.ToLower(doc.Title), " ", "_")
		docID = fmt.Sprintf("corpus_%s", sanitizedTitle)
	}
	
	// Add document with corpus-specific metadata
	return r.AddDocument(ctx, docID, searchableContent, "corpus")
}

// EnhancePrompt adds retrieved context to a prompt for better LLM responses.
// This is the main integration point with the agent system.
//
// Example:
//   enhanced, err := retriever.EnhancePrompt(ctx, "How do agents communicate?", config)
func (r *Retriever) EnhancePrompt(ctx context.Context, prompt string, config RetrievalConfig) (string, error) {
	// Retrieve relevant context
	results, err := r.RetrieveContext(ctx, prompt, config)
	if err != nil {
		return prompt, err // Return original prompt on error
	}
	
	// If no results, return original prompt
	if len(results.Results) == 0 {
		return prompt, nil
	}
	
	// Build enhanced prompt with context
	var builder strings.Builder
	
	// Add context header
	builder.WriteString("# Context\n")
	builder.WriteString("The following information may be relevant to your query:\n\n")
	
	// Add each result as context
	for i, result := range results.Results {
		builder.WriteString(fmt.Sprintf("## Source %d: %s (Relevance: %.2f)\n", 
			i+1, result.Source, result.Score))
		builder.WriteString(result.Content)
		builder.WriteString("\n\n")
	}
	
	// Add original prompt
	builder.WriteString("# Query\n")
	builder.WriteString(prompt)
	
	return builder.String(), nil
}

// Close closes the retriever and its resources.
// This ensures proper cleanup of the vector store and any open connections.
func (r *Retriever) Close() error {
	if r.vectorStore != nil {
		return r.vectorStore.Close()
	}
	return nil
}