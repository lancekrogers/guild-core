package rag

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lancerogers/guild/pkg/corpus"
	"github.com/lancerogers/guild/pkg/memory/vector"
)

// RetrievalConfig defines configuration options for the retriever
type RetrievalConfig struct {
	// MaxResults is the maximum number of results to return
	MaxResults int

	// MinScore is the minimum similarity score required (0-1)
	MinScore float32

	// IncludeCorpus determines whether to include documents from the corpus
	IncludeCorpus bool

	// IncludePrompts determines whether to include previous prompts/responses
	IncludePrompts bool

	// ChunkSize is the size of chunks to break documents into
	ChunkSize int

	// ChunkOverlap is the overlap between chunks
	ChunkOverlap int

	// ChunkStrategy is the strategy for chunking documents
	ChunkStrategy ChunkStrategy
}

// DefaultRetrievalConfig returns a default retrieval configuration
func DefaultRetrievalConfig() RetrievalConfig {
	return RetrievalConfig{
		MaxResults:     5,
		MinScore:       0.7,
		IncludeCorpus:  true,
		IncludePrompts: true,
		ChunkSize:      1000,
		ChunkOverlap:   100,
		ChunkStrategy:  ChunkByParagraph,
	}
}

// Retriever provides retrieval-augmented generation capabilities
type Retriever struct {
	vectorStore  vector.VectorStore
	corpusConfig corpus.Config
	chunker      *Chunker
}

// NewRetriever creates a new RAG retriever
func NewRetriever(vectorStore vector.VectorStore, corpusConfig corpus.Config) *Retriever {
	return &Retriever{
		vectorStore:  vectorStore,
		corpusConfig: corpusConfig,
		chunker:      NewChunker(), // Use default chunker
	}
}

// WithChunker configures the retriever with a custom chunker
func (r *Retriever) WithChunker(chunker *Chunker) *Retriever {
	r.chunker = chunker
	return r
}

// WithCorpusConfig configures the retriever with a corpus config
func (r *Retriever) WithCorpusConfig(corpusConfig corpus.Config) *Retriever {
	r.corpusConfig = corpusConfig
	return r
}

// Document represents a document to be indexed
type Document struct {
	// ID is a unique identifier for the document
	ID string

	// Title is the document title
	Title string

	// Content is the document content
	Content string

	// Source is where the document came from
	Source string

	// Metadata contains additional information about the document
	Metadata map[string]interface{}
}

// IndexDocument indexes a document into the vector store
func (r *Retriever) IndexDocument(ctx context.Context, doc Document) error {
	// Apply chunking if the document is too large
	chunks := r.chunker.ChunkDocumentWithMetadata(doc.Content, doc.Source, doc.Metadata)
	
	// Store each chunk as a separate embedding
	for i, chunk := range chunks {
		// Create a unique ID for each chunk
		chunkID := doc.ID
		if i > 0 {
			chunkID = fmt.Sprintf("%s_chunk_%d", doc.ID, i)
		}
		
		// Add document title and other metadata
		chunkMetadata := chunk.Metadata
		if chunkMetadata == nil {
			chunkMetadata = make(map[string]interface{})
		}
		
		chunkMetadata["title"] = doc.Title
		chunkMetadata["source"] = chunk.Source
		chunkMetadata["chunk_index"] = i
		chunkMetadata["chunk_count"] = len(chunks)
		
		// Create the embedding
		embedding := vector.Embedding{
			ID:        chunkID,
			Text:      chunk.Content,
			Source:    chunk.Source,
			Timestamp: time.Now(),
			Metadata:  chunkMetadata,
		}
		
		// Store in vector store
		if err := r.vectorStore.SaveEmbedding(ctx, embedding); err != nil {
			return fmt.Errorf("failed to save embedding for chunk %d: %w", i, err)
		}
	}
	
	return nil
}

// SearchResults represents the results of a search
type SearchResults struct {
	// Matches are the matching documents
	Matches []SearchMatch

	// FormattedContext is a formatted string of the search results
	FormattedContext string
}

// SearchMatch represents a single search match
type SearchMatch struct {
	// ID is the document ID
	ID string

	// Content is the matched content
	Content string

	// Title is the document title (if available)
	Title string

	// Source is where the content came from
	Source string

	// Score is the similarity score (0-1)
	Score float32

	// Metadata contains additional information about the match
	Metadata map[string]interface{}
}

// RetrieveContext gets relevant context for a query
func (r *Retriever) RetrieveContext(ctx context.Context, query string, config RetrievalConfig) (*SearchResults, error) {
	// Use default config if none provided
	if config.MaxResults == 0 {
		config = DefaultRetrievalConfig()
	}

	// Collect matches from different sources
	var allMatches []SearchMatch

	// Search vector store
	vectorMatches, err := r.searchVectorStore(ctx, query, config.MaxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to query vector store: %w", err)
	}
	allMatches = append(allMatches, vectorMatches...)

	// Search corpus if enabled
	if config.IncludeCorpus {
		corpusMatches, err := r.searchCorpus(ctx, query, config.MaxResults)
		if err == nil {
			allMatches = append(allMatches, corpusMatches...)
		}
	}

	// Filter by minimum score
	if config.MinScore > 0 {
		var filteredMatches []SearchMatch
		for _, match := range allMatches {
			if match.Score >= config.MinScore {
				filteredMatches = append(filteredMatches, match)
			}
		}
		allMatches = filteredMatches
	}

	// Sort by score (highest first)
	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Score > allMatches[j].Score
	})

	// Limit to max results
	if len(allMatches) > config.MaxResults {
		allMatches = allMatches[:config.MaxResults]
	}

	// Format the results as a string
	formattedContext := formatSearchResults(allMatches)

	return &SearchResults{
		Matches:          allMatches,
		FormattedContext: formattedContext,
	}, nil
}

// searchVectorStore searches the vector store for relevant content
func (r *Retriever) searchVectorStore(ctx context.Context, query string, limit int) ([]SearchMatch, error) {
	matches, err := r.vectorStore.QueryEmbeddings(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	// Convert to SearchMatch format
	var searchMatches []SearchMatch
	for _, match := range matches {
		title := ""
		if t, ok := match.Metadata["title"].(string); ok {
			title = t
		}

		searchMatches = append(searchMatches, SearchMatch{
			ID:       match.ID,
			Content:  match.Text,
			Title:    title,
			Source:   match.Source,
			Score:    match.Score,
			Metadata: match.Metadata,
		})
	}

	return searchMatches, nil
}

// searchCorpus searches the corpus for relevant content
func (r *Retriever) searchCorpus(ctx context.Context, query string, limit int) ([]SearchMatch, error) {
	// List all corpus docs
	docs, err := corpus.List(r.corpusConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to list corpus documents: %w", err)
	}
	
	// Simple keyword matching (for a basic implementation)
	// In a real implementation, we might want to also use semantic search via embeddings
	queryLower := strings.ToLower(query)
	var matches []SearchMatch
	
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
			
			// Create metadata
			metadata := map[string]interface{}{
				"title": fullDoc.Title,
				"tags":  fullDoc.Tags,
				"type":  "corpus",
			}
			
			match := SearchMatch{
				ID:       fullDoc.FilePath,
				Content:  fullDoc.Body,
				Title:    fullDoc.Title,
				Source:   "Corpus: " + fullDoc.Title,
				Score:    score,
				Metadata: metadata,
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

// formatSearchResults formats search results into a readable string
func formatSearchResults(matches []SearchMatch) string {
	var builder strings.Builder
	builder.WriteString("# Relevant Context\n\n")

	for i, match := range matches {
		// Add source and title if available
		sourceInfo := match.Source
		if match.Title != "" && !strings.Contains(sourceInfo, match.Title) {
			sourceInfo = match.Title + " (" + sourceInfo + ")"
		}
		
		builder.WriteString(fmt.Sprintf("## Source %d: %s (Relevance: %.2f)\n\n", 
			i+1, sourceInfo, match.Score))
		
		// Add content
		builder.WriteString(match.Content)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

// EnhancePrompt adds retrieved context to a prompt
func (r *Retriever) EnhancePrompt(ctx context.Context, prompt, query string, config RetrievalConfig) (string, error) {
	// Get relevant context
	results, err := r.RetrieveContext(ctx, query, config)
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
`, results.FormattedContext, prompt)

	return enhanced, nil
}