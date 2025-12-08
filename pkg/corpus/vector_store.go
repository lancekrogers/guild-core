// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package corpus

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/memory/vector"
)

const (
	// CorpusCollectionName is the default collection name for corpus documents
	CorpusCollectionName = "corpus_documents"

	// DefaultChunkSize is the default size for document chunks
	DefaultChunkSize = 1000

	// DefaultChunkOverlap is the default overlap between chunks
	DefaultChunkOverlap = 200

	// MaxChunksPerDocument limits chunks to prevent memory issues
	MaxChunksPerDocument = 100
)

// ChunkingStrategy defines how documents are split into chunks
type ChunkingStrategy string

const (
	ChunkingStrategyRecursive ChunkingStrategy = "recursive"
	ChunkingStrategySentence  ChunkingStrategy = "sentence"
	ChunkingStrategyParagraph ChunkingStrategy = "paragraph"
	ChunkingStrategyNone      ChunkingStrategy = "none"
)

// DocumentChunk represents a chunk of a document with metadata
type DocumentChunk struct {
	ID           string
	DocumentID   string
	DocumentPath string
	Content      string
	ChunkIndex   int
	StartOffset  int
	EndOffset    int
	Metadata     map[string]interface{}
}

// VectorStoreConfig configures the corpus vector store
type VectorStoreConfig struct {
	// ChromemStore is the underlying vector store
	ChromemStore *vector.ChromemStore

	// CollectionName for corpus documents
	CollectionName string

	// ChunkSize for splitting documents
	ChunkSize int

	// ChunkOverlap between consecutive chunks
	ChunkOverlap int

	// ChunkingStrategy to use
	Strategy ChunkingStrategy

	// MaxConcurrency for parallel processing
	MaxConcurrency int
}

// CorpusVectorStore manages vector embeddings for corpus documents
type CorpusVectorStore struct {
	store          *vector.ChromemStore
	collectionName string
	chunkSize      int
	chunkOverlap   int
	strategy       ChunkingStrategy
	maxConcurrency int
	mu             sync.RWMutex
}

// NewCorpusVectorStore creates a new corpus vector store
func NewCorpusVectorStore(config VectorStoreConfig) (*CorpusVectorStore, error) {
	if config.ChromemStore == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "chromem store is required", nil).
			WithComponent("corpus.vector").
			WithOperation("NewCorpusVectorStore")
	}

	// Set defaults
	if config.CollectionName == "" {
		config.CollectionName = CorpusCollectionName
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = DefaultChunkSize
	}
	if config.ChunkOverlap <= 0 {
		config.ChunkOverlap = DefaultChunkOverlap
	}
	if config.ChunkOverlap >= config.ChunkSize {
		config.ChunkOverlap = config.ChunkSize / 5 // Max 20% overlap
	}
	if config.Strategy == "" {
		config.Strategy = ChunkingStrategyRecursive
	}
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 4
	}

	// Ensure collection exists
	_, err := config.ChromemStore.GetCollection(config.CollectionName)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get/create collection").
			WithComponent("corpus.vector").
			WithOperation("NewCorpusVectorStore").
			WithDetails("collection", config.CollectionName)
	}

	return &CorpusVectorStore{
		store:          config.ChromemStore,
		collectionName: config.CollectionName,
		chunkSize:      config.ChunkSize,
		chunkOverlap:   config.ChunkOverlap,
		strategy:       config.Strategy,
		maxConcurrency: config.MaxConcurrency,
	}, nil
}

// IndexDocument adds a document to the vector store with chunking
func (vs *CorpusVectorStore) IndexDocument(ctx context.Context, doc *ScannedDocument) error {
	if doc == nil {
		return gerror.New(gerror.ErrCodeInvalidInput, "document cannot be nil", nil).
			WithComponent("corpus.vector").
			WithOperation("IndexDocument")
	}

	// Check context
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.vector").
			WithOperation("IndexDocument")
	}

	// Create chunks based on strategy
	chunks, err := vs.chunkDocument(ctx, doc)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to chunk document").
			WithComponent("corpus.vector").
			WithOperation("IndexDocument").
			WithDetails("document_id", doc.ID)
	}

	// Index chunks in parallel
	return vs.indexChunks(ctx, chunks)
}

// IndexDocuments indexes multiple documents in parallel
func (vs *CorpusVectorStore) IndexDocuments(ctx context.Context, documents []*ScannedDocument) error {
	if len(documents) == 0 {
		return nil
	}

	// Create error channel for collecting errors
	errChan := make(chan error, len(documents))

	// Use semaphore for concurrency control
	sem := make(chan struct{}, vs.maxConcurrency)
	var wg sync.WaitGroup

	for _, doc := range documents {
		wg.Add(1)
		go func(d *ScannedDocument) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check context
			if err := ctx.Err(); err != nil {
				errChan <- err
				return
			}

			if err := vs.IndexDocument(ctx, d); err != nil {
				errChan <- gerror.Wrap(err, gerror.ErrCodeInternal, "failed to index document").
					WithDetails("path", d.Path)
			}
		}(doc)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal,
			fmt.Sprintf("failed to index %d documents", len(errors)), nil).
			WithComponent("corpus.vector").
			WithOperation("IndexDocuments").
			WithDetails("error_count", len(errors))
	}

	return nil
}

// SearchDocuments performs semantic search across corpus documents
func (vs *CorpusVectorStore) SearchDocuments(ctx context.Context, query string, limit int) ([]DocumentSearchResult, error) {
	if query == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "query cannot be empty", nil).
			WithComponent("corpus.vector").
			WithOperation("SearchDocuments")
	}

	if limit <= 0 {
		limit = 10
	}

	// Search in the corpus collection with adaptive limit
	// Try with limit*2 first for better deduplication, then fall back to smaller limits
	var matches []vector.EmbeddingMatch
	var err error

	searchLimits := []int{limit * 2, limit, max(limit/2, 1)}
	for _, searchLimit := range searchLimits {
		matches, err = vs.store.QueryCollection(ctx, vs.collectionName, query, searchLimit)
		if err == nil {
			break
		}
		// Check if this is the specific "nResults must be <= the number of documents" error
		if !strings.Contains(err.Error(), "nResults must be") {
			// This is a different error, don't retry
			break
		}
	}

	// If all attempts failed due to limit being too high, try with progressively smaller limits
	if err != nil && strings.Contains(err.Error(), "nResults must be") {
		for tryLimit := max(limit/2, 1); tryLimit >= 1; tryLimit-- {
			matches, err = vs.store.QueryCollection(ctx, vs.collectionName, query, tryLimit)
			if err == nil {
				break
			}
			if !strings.Contains(err.Error(), "nResults must be") {
				break
			}
		}
	}

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to query vector store").
			WithComponent("corpus.vector").
			WithOperation("SearchDocuments")
	}

	// Group chunks by document and calculate aggregate scores
	documentMap := make(map[string]*DocumentSearchResult)

	for _, match := range matches {
		docID, _ := match.Metadata["document_id"].(string)
		if docID == "" {
			continue
		}

		if existing, ok := documentMap[docID]; ok {
			// Update with higher score
			if match.Score > existing.Score {
				existing.Score = match.Score
			}
			existing.MatchedChunks = append(existing.MatchedChunks, ChunkMatch{
				ChunkID:     match.ID,
				ChunkIndex:  getIntFromMetadata(match.Metadata, "chunk_index"),
				Score:       match.Score,
				Content:     match.Text,
				StartOffset: getIntFromMetadata(match.Metadata, "start_offset"),
				EndOffset:   getIntFromMetadata(match.Metadata, "end_offset"),
			})
		} else {
			// Create new result
			result := &DocumentSearchResult{
				DocumentID:   docID,
				DocumentPath: getStringFromMetadata(match.Metadata, "document_path"),
				Title:        getStringFromMetadata(match.Metadata, "title"),
				Score:        match.Score,
				ContentType:  ContentType(getStringFromMetadata(match.Metadata, "content_type")),
				MatchedChunks: []ChunkMatch{{
					ChunkID:     match.ID,
					ChunkIndex:  getIntFromMetadata(match.Metadata, "chunk_index"),
					Score:       match.Score,
					Content:     match.Text,
					StartOffset: getIntFromMetadata(match.Metadata, "start_offset"),
					EndOffset:   getIntFromMetadata(match.Metadata, "end_offset"),
				}},
			}
			documentMap[docID] = result
		}
	}

	// Convert map to slice and sort by score
	results := make([]DocumentSearchResult, 0, len(documentMap))
	for _, result := range documentMap {
		results = append(results, *result)
	}

	// Sort by score (highest first)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// RemoveDocument removes all chunks for a document from the vector store
func (vs *CorpusVectorStore) RemoveDocument(ctx context.Context, documentID string) error {
	// Note: ChromemGo doesn't support deletion yet, so we'll track this for future implementation
	return gerror.New(gerror.ErrCodeInternal, "document removal not yet supported", nil).
		WithComponent("corpus.vector").
		WithOperation("RemoveDocument").
		WithDetails("document_id", documentID)
}

// chunkDocument splits a document into chunks based on the configured strategy
func (vs *CorpusVectorStore) chunkDocument(ctx context.Context, doc *ScannedDocument) ([]DocumentChunk, error) {
	switch vs.strategy {
	case ChunkingStrategyNone:
		return vs.chunkNone(doc)
	case ChunkingStrategySentence:
		return vs.chunkBySentence(doc)
	case ChunkingStrategyParagraph:
		return vs.chunkByParagraph(doc)
	case ChunkingStrategyRecursive:
		fallthrough
	default:
		return vs.chunkRecursive(doc)
	}
}

// chunkNone returns the entire document as a single chunk
func (vs *CorpusVectorStore) chunkNone(doc *ScannedDocument) ([]DocumentChunk, error) {
	chunk := DocumentChunk{
		ID:           fmt.Sprintf("%s_chunk_0", doc.ID),
		DocumentID:   doc.ID,
		DocumentPath: doc.Path,
		Content:      doc.Content,
		ChunkIndex:   0,
		StartOffset:  0,
		EndOffset:    len(doc.Content),
		Metadata:     vs.createChunkMetadata(doc, 0),
	}
	return []DocumentChunk{chunk}, nil
}

// chunkRecursive implements recursive text splitting with overlap
func (vs *CorpusVectorStore) chunkRecursive(doc *ScannedDocument) ([]DocumentChunk, error) {
	chunks := []DocumentChunk{}
	content := doc.Content

	// For code files, try to split at natural boundaries
	if doc.Type == ContentTypeGo {
		return vs.chunkCode(doc)
	}

	// For markdown, preserve structure
	if doc.Type == ContentTypeMarkdown {
		return vs.chunkMarkdown(doc)
	}

	// Generic recursive chunking
	currentOffset := 0
	chunkIndex := 0

	for currentOffset < len(content) && chunkIndex < MaxChunksPerDocument {
		// Calculate chunk boundaries
		endOffset := currentOffset + vs.chunkSize
		if endOffset > len(content) {
			endOffset = len(content)
		}

		// Try to find a good split point (paragraph, sentence, or word boundary)
		if endOffset < len(content) {
			// Look for paragraph break
			if idx := strings.LastIndex(content[currentOffset:endOffset], "\n\n"); idx > 0 {
				endOffset = currentOffset + idx
			} else if idx := strings.LastIndex(content[currentOffset:endOffset], ". "); idx > 0 {
				endOffset = currentOffset + idx + 1
			} else if idx := strings.LastIndex(content[currentOffset:endOffset], " "); idx > 0 {
				endOffset = currentOffset + idx
			}
		}

		// Create chunk
		chunkContent := content[currentOffset:endOffset]
		if strings.TrimSpace(chunkContent) != "" {
			chunk := DocumentChunk{
				ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
				DocumentID:   doc.ID,
				DocumentPath: doc.Path,
				Content:      chunkContent,
				ChunkIndex:   chunkIndex,
				StartOffset:  currentOffset,
				EndOffset:    endOffset,
				Metadata:     vs.createChunkMetadata(doc, chunkIndex),
			}
			chunks = append(chunks, chunk)
			chunkIndex++
		}

		// Move to next chunk with overlap
		currentOffset = endOffset - vs.chunkOverlap
		if currentOffset < 0 {
			currentOffset = endOffset
		}
	}

	return chunks, nil
}

// chunkMarkdown preserves markdown structure
func (vs *CorpusVectorStore) chunkMarkdown(doc *ScannedDocument) ([]DocumentChunk, error) {
	chunks := []DocumentChunk{}
	lines := strings.Split(doc.Content, "\n")

	currentChunk := strings.Builder{}
	currentOffset := 0
	chunkStartOffset := 0
	chunkIndex := 0

	for i, line := range lines {
		lineWithNewline := line
		if i < len(lines)-1 {
			lineWithNewline += "\n"
		}

		// Check if adding this line would exceed chunk size
		if currentChunk.Len()+len(lineWithNewline) > vs.chunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			if strings.TrimSpace(currentChunk.String()) != "" {
				chunk := DocumentChunk{
					ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
					DocumentID:   doc.ID,
					DocumentPath: doc.Path,
					Content:      currentChunk.String(),
					ChunkIndex:   chunkIndex,
					StartOffset:  chunkStartOffset,
					EndOffset:    currentOffset,
					Metadata:     vs.createChunkMetadata(doc, chunkIndex),
				}
				chunks = append(chunks, chunk)
				chunkIndex++
			}

			// Start new chunk with overlap
			currentChunk.Reset()

			// Add context from previous chunk if this isn't a heading
			if !strings.HasPrefix(line, "#") && i > 0 {
				// Look back for context
				contextLines := []string{}
				for j := i - 1; j >= 0 && len(strings.Join(contextLines, "\n")) < vs.chunkOverlap; j-- {
					if strings.HasPrefix(lines[j], "#") {
						contextLines = append([]string{lines[j]}, contextLines...)
						break
					}
					contextLines = append([]string{lines[j]}, contextLines...)
				}
				for _, contextLine := range contextLines {
					currentChunk.WriteString(contextLine)
					currentChunk.WriteString("\n")
				}
			}

			chunkStartOffset = currentOffset
		}

		currentChunk.WriteString(lineWithNewline)
		currentOffset += len(lineWithNewline)
	}

	// Add final chunk
	if currentChunk.Len() > 0 && strings.TrimSpace(currentChunk.String()) != "" {
		chunk := DocumentChunk{
			ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
			DocumentID:   doc.ID,
			DocumentPath: doc.Path,
			Content:      currentChunk.String(),
			ChunkIndex:   chunkIndex,
			StartOffset:  chunkStartOffset,
			EndOffset:    currentOffset,
			Metadata:     vs.createChunkMetadata(doc, chunkIndex),
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// chunkCode preserves code structure
func (vs *CorpusVectorStore) chunkCode(doc *ScannedDocument) ([]DocumentChunk, error) {
	// For Go code, try to split at function boundaries
	chunks := []DocumentChunk{}
	lines := strings.Split(doc.Content, "\n")

	currentChunk := strings.Builder{}
	currentOffset := 0
	chunkStartOffset := 0
	chunkIndex := 0
	inFunction := false

	for i, line := range lines {
		lineWithNewline := line
		if i < len(lines)-1 {
			lineWithNewline += "\n"
		}

		// Detect function boundaries
		if strings.HasPrefix(strings.TrimSpace(line), "func ") {
			// If we're in a function and found a new one, save the current chunk
			if inFunction && currentChunk.Len() > 0 {
				chunk := DocumentChunk{
					ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
					DocumentID:   doc.ID,
					DocumentPath: doc.Path,
					Content:      currentChunk.String(),
					ChunkIndex:   chunkIndex,
					StartOffset:  chunkStartOffset,
					EndOffset:    currentOffset,
					Metadata:     vs.createChunkMetadata(doc, chunkIndex),
				}
				chunks = append(chunks, chunk)
				chunkIndex++

				currentChunk.Reset()
				chunkStartOffset = currentOffset
			}
			inFunction = true
		}

		// Check size limit
		if currentChunk.Len()+len(lineWithNewline) > vs.chunkSize && currentChunk.Len() > 0 {
			chunk := DocumentChunk{
				ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
				DocumentID:   doc.ID,
				DocumentPath: doc.Path,
				Content:      currentChunk.String(),
				ChunkIndex:   chunkIndex,
				StartOffset:  chunkStartOffset,
				EndOffset:    currentOffset,
				Metadata:     vs.createChunkMetadata(doc, chunkIndex),
			}
			chunks = append(chunks, chunk)
			chunkIndex++

			currentChunk.Reset()
			chunkStartOffset = currentOffset
			inFunction = false
		}

		currentChunk.WriteString(lineWithNewline)
		currentOffset += len(lineWithNewline)
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunk := DocumentChunk{
			ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
			DocumentID:   doc.ID,
			DocumentPath: doc.Path,
			Content:      currentChunk.String(),
			ChunkIndex:   chunkIndex,
			StartOffset:  chunkStartOffset,
			EndOffset:    currentOffset,
			Metadata:     vs.createChunkMetadata(doc, chunkIndex),
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// chunkBySentence splits at sentence boundaries
func (vs *CorpusVectorStore) chunkBySentence(doc *ScannedDocument) ([]DocumentChunk, error) {
	// Simple sentence splitting - could be improved with NLP
	sentences := strings.Split(doc.Content, ". ")
	chunks := []DocumentChunk{}

	currentChunk := strings.Builder{}
	currentOffset := 0
	chunkStartOffset := 0
	chunkIndex := 0

	for i, sentence := range sentences {
		if i < len(sentences)-1 {
			sentence += ". "
		}

		if currentChunk.Len()+len(sentence) > vs.chunkSize && currentChunk.Len() > 0 {
			chunk := DocumentChunk{
				ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
				DocumentID:   doc.ID,
				DocumentPath: doc.Path,
				Content:      currentChunk.String(),
				ChunkIndex:   chunkIndex,
				StartOffset:  chunkStartOffset,
				EndOffset:    currentOffset,
				Metadata:     vs.createChunkMetadata(doc, chunkIndex),
			}
			chunks = append(chunks, chunk)
			chunkIndex++

			currentChunk.Reset()
			chunkStartOffset = currentOffset
		}

		currentChunk.WriteString(sentence)
		currentOffset += len(sentence)
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunk := DocumentChunk{
			ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
			DocumentID:   doc.ID,
			DocumentPath: doc.Path,
			Content:      currentChunk.String(),
			ChunkIndex:   chunkIndex,
			StartOffset:  chunkStartOffset,
			EndOffset:    currentOffset,
			Metadata:     vs.createChunkMetadata(doc, chunkIndex),
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// chunkByParagraph splits at paragraph boundaries
func (vs *CorpusVectorStore) chunkByParagraph(doc *ScannedDocument) ([]DocumentChunk, error) {
	paragraphs := strings.Split(doc.Content, "\n\n")
	chunks := []DocumentChunk{}

	currentChunk := strings.Builder{}
	currentOffset := 0
	chunkStartOffset := 0
	chunkIndex := 0

	for i, paragraph := range paragraphs {
		if i < len(paragraphs)-1 {
			paragraph += "\n\n"
		}

		if currentChunk.Len()+len(paragraph) > vs.chunkSize && currentChunk.Len() > 0 {
			chunk := DocumentChunk{
				ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
				DocumentID:   doc.ID,
				DocumentPath: doc.Path,
				Content:      currentChunk.String(),
				ChunkIndex:   chunkIndex,
				StartOffset:  chunkStartOffset,
				EndOffset:    currentOffset,
				Metadata:     vs.createChunkMetadata(doc, chunkIndex),
			}
			chunks = append(chunks, chunk)
			chunkIndex++

			currentChunk.Reset()
			chunkStartOffset = currentOffset
		}

		currentChunk.WriteString(paragraph)
		currentOffset += len(paragraph)
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunk := DocumentChunk{
			ID:           fmt.Sprintf("%s_chunk_%d", doc.ID, chunkIndex),
			DocumentID:   doc.ID,
			DocumentPath: doc.Path,
			Content:      currentChunk.String(),
			ChunkIndex:   chunkIndex,
			StartOffset:  chunkStartOffset,
			EndOffset:    currentOffset,
			Metadata:     vs.createChunkMetadata(doc, chunkIndex),
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// indexChunks adds chunks to the vector store
func (vs *CorpusVectorStore) indexChunks(ctx context.Context, chunks []DocumentChunk) error {
	for _, chunk := range chunks {
		embedding := vector.Embedding{
			ID:        chunk.ID,
			Text:      chunk.Content,
			Source:    "corpus",
			Timestamp: time.Now(),
			Metadata:  chunk.Metadata,
		}

		if err := vs.store.SaveEmbedding(ctx, embedding); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save chunk embedding").
				WithComponent("corpus.vector").
				WithOperation("indexChunks").
				WithDetails("chunk_id", chunk.ID)
		}
	}

	return nil
}

// createChunkMetadata creates metadata for a chunk
func (vs *CorpusVectorStore) createChunkMetadata(doc *ScannedDocument, chunkIndex int) map[string]interface{} {
	metadata := map[string]interface{}{
		"collection":     vs.collectionName,
		"document_id":    doc.ID,
		"document_path":  doc.Path,
		"content_type":   string(doc.Type),
		"chunk_index":    chunkIndex,
		"chunk_strategy": string(vs.strategy),
		"indexed_at":     time.Now().Format(time.RFC3339),
	}

	// Add document metadata
	if doc.Metadata.Title != "" {
		metadata["title"] = doc.Metadata.Title
	}
	if doc.Metadata.Description != "" {
		metadata["description"] = doc.Metadata.Description
	}
	if len(doc.Metadata.ExtractedTags) > 0 {
		metadata["tags"] = strings.Join(doc.Metadata.ExtractedTags, ",")
	}
	if doc.Metadata.Language != "" {
		metadata["language"] = doc.Metadata.Language
	}

	return metadata
}

// Helper functions

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func getStringFromMetadata(metadata map[string]interface{}, key string) string {
	if val, ok := metadata[key].(string); ok {
		return val
	}
	return ""
}

func getIntFromMetadata(metadata map[string]interface{}, key string) int {
	if val, ok := metadata[key].(int); ok {
		return val
	}
	if val, ok := metadata[key].(float64); ok {
		return int(val)
	}
	return 0
}

// DocumentSearchResult represents a search result with matched chunks
type DocumentSearchResult struct {
	DocumentID    string
	DocumentPath  string
	Title         string
	Score         float32
	ContentType   ContentType
	MatchedChunks []ChunkMatch
}

// ChunkMatch represents a matched chunk within a document
type ChunkMatch struct {
	ChunkID     string
	ChunkIndex  int
	Score       float32
	Content     string
	StartOffset int
	EndOffset   int
}
