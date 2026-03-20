// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package rag

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"

	"github.com/lancekrogers/guild-core/pkg/memory/vector"
)

// processDocumentBatch processes a batch of documents for indexing
func (f *RAGTestFramework) processDocumentBatch(documents []*vector.Document, config IndexConfig) ([]*vector.Document, int, error) {
	processedDocs := make([]*vector.Document, 0)
	totalChunks := 0

	for _, doc := range documents {
		// Split document into chunks
		chunks := f.splitIntoChunks(doc.Content, config.ChunkSize, config.ChunkOverlap)

		for i, chunk := range chunks {
			if len(strings.TrimSpace(chunk)) < 50 { // Skip very small chunks
				continue
			}

			// Generate embedding for chunk (simulated)
			embedding := f.generateEmbedding(chunk, config.EmbeddingModel)

			// Create processed document
			processedDoc := &vector.Document{
				ID:        fmt.Sprintf("%s-chunk-%d", doc.ID, i),
				Content:   chunk,
				Embedding: embedding,
				Metadata: map[string]interface{}{
					"parent_doc_id": doc.ID,
					"chunk_index":   i,
					"file_path":     getMetadataString(doc.Metadata.(map[string]interface{}), "file_path"),
					"language":      getMetadataString(doc.Metadata.(map[string]interface{}), "language"),
					"size":          len(chunk),
					"chunk_type":    f.classifyChunk(chunk),
				},
			}

			// Add keywords if enabled
			if config.EnableKeywords {
				metadata := processedDoc.Metadata.(map[string]interface{})
				metadata["keywords"] = f.extractKeywords(chunk)
			}

			// Add summary if enabled
			if config.EnableSummaries {
				metadata := processedDoc.Metadata.(map[string]interface{})
				metadata["summary"] = f.generateSummary(chunk)
			}

			processedDocs = append(processedDocs, processedDoc)
			totalChunks++
		}
	}

	return processedDocs, totalChunks, nil
}

// splitIntoChunks splits text into overlapping chunks
func (f *RAGTestFramework) splitIntoChunks(text string, chunkSize, overlap int) []string {
	if len(text) <= chunkSize {
		return []string{text}
	}

	chunks := make([]string, 0)
	start := 0

	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at word boundaries
		if end < len(text) {
			for i := end; i > start+chunkSize/2; i-- {
				if text[i] == ' ' || text[i] == '\n' || text[i] == '.' {
					end = i
					break
				}
			}
		}

		chunk := text[start:end]
		chunks = append(chunks, chunk)

		// Move start position with overlap
		start = end - overlap
		if start <= 0 {
			start = end
		}
	}

	return chunks
}

// generateEmbedding generates vector embedding for text (simulated)
func (f *RAGTestFramework) generateEmbedding(text, model string) []float32 {
	// Simulate different embedding models with different dimensions
	var dimensions int
	switch model {
	case "sentence-transformers/all-MiniLM-L6-v2":
		dimensions = 384
	case "text-embedding-ada-002":
		dimensions = 1536
	default:
		dimensions = 768
	}

	embedding := make([]float32, dimensions)

	// Generate pseudo-realistic embeddings based on text content
	hash := f.simpleHash(text)
	rand.Seed(int64(hash))

	for i := 0; i < dimensions; i++ {
		embedding[i] = float32(rand.NormFloat64())
	}

	// Normalize the vector
	norm := float32(0)
	for _, val := range embedding {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))

	for i := range embedding {
		embedding[i] /= norm
	}

	return embedding
}

// generateQueryEmbedding generates embedding for search query
func (f *RAGTestFramework) generateQueryEmbedding(query string) []float32 {
	return f.generateEmbedding(query, "sentence-transformers/all-MiniLM-L6-v2")
}

// classifyChunk classifies the type of content chunk
func (f *RAGTestFramework) classifyChunk(chunk string) string {
	chunk = strings.ToLower(chunk)

	if strings.Contains(chunk, "func ") || strings.Contains(chunk, "function") {
		return "code"
	} else if strings.Contains(chunk, "# ") || strings.Contains(chunk, "## ") {
		return "documentation"
	} else if strings.Contains(chunk, "test") && strings.Contains(chunk, "assert") {
		return "test"
	} else if strings.Contains(chunk, "import") || strings.Contains(chunk, "package") {
		return "header"
	} else if strings.Contains(chunk, "todo") || strings.Contains(chunk, "fixme") {
		return "comment"
	}

	return "content"
}

// extractKeywords extracts keywords from text
func (f *RAGTestFramework) extractKeywords(text string) []string {
	// Simple keyword extraction using common programming terms
	keywords := []string{}

	commonTerms := []string{
		"function", "func", "method", "class", "interface", "struct",
		"variable", "const", "type", "import", "package", "module",
		"test", "assert", "error", "context", "handler", "service",
		"database", "query", "api", "endpoint", "request", "response",
		"cache", "memory", "performance", "optimization", "security",
	}

	textLower := strings.ToLower(text)
	for _, term := range commonTerms {
		if strings.Contains(textLower, term) {
			keywords = append(keywords, term)
		}
	}

	// Add camelCase and PascalCase identifiers
	re := regexp.MustCompile(`\b[A-Z][a-z]+(?:[A-Z][a-z]*)*\b`)
	matches := re.FindAllString(text, -1)
	for _, match := range matches {
		if len(match) > 3 { // Only include meaningful identifiers
			keywords = append(keywords, strings.ToLower(match))
		}
	}

	return keywords
}

// generateSummary generates a summary of the chunk
func (f *RAGTestFramework) generateSummary(chunk string) string {
	// Simple summarization by taking first and last sentences
	lines := strings.Split(chunk, "\n")
	nonEmptyLines := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	if len(nonEmptyLines) == 0 {
		return "Empty content"
	} else if len(nonEmptyLines) == 1 {
		return nonEmptyLines[0]
	} else if len(nonEmptyLines) <= 3 {
		return strings.Join(nonEmptyLines, " ")
	}

	// Take first and last meaningful lines
	summary := nonEmptyLines[0]
	if len(nonEmptyLines) > 1 {
		summary += " ... " + nonEmptyLines[len(nonEmptyLines)-1]
	}

	if len(summary) > 150 {
		summary = summary[:147] + "..."
	}

	return summary
}

// extractRelevantContent extracts relevant content around query matches
func (f *RAGTestFramework) extractRelevantContent(content, query string, windowSize int) string {
	queryTerms := f.extractQueryTerms(query)
	if len(queryTerms) == 0 {
		// Return first part of content if no query terms
		if len(content) <= windowSize {
			return content
		}
		return content[:windowSize] + "..."
	}

	// Find the best match position
	bestPos := -1
	maxMatches := 0

	for i := 0; i <= len(content)-windowSize; i += 20 {
		window := content[i : i+windowSize]
		matches := 0
		for _, term := range queryTerms {
			if strings.Contains(strings.ToLower(window), strings.ToLower(term)) {
				matches++
			}
		}
		if matches > maxMatches {
			maxMatches = matches
			bestPos = i
		}
	}

	if bestPos == -1 {
		bestPos = 0
	}

	end := bestPos + windowSize
	if end > len(content) {
		end = len(content)
	}

	result := content[bestPos:end]
	if bestPos > 0 {
		result = "..." + result
	}
	if end < len(content) {
		result = result + "..."
	}

	return result
}

// extractQueryTerms extracts meaningful terms from query
func (f *RAGTestFramework) extractQueryTerms(query string) []string {
	// Remove common stop words and extract meaningful terms
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "how": true, "what": true, "where": true, "when": true,
		"why": true, "is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
	}

	words := strings.Fields(strings.ToLower(query))
	terms := make([]string, 0)

	for _, word := range words {
		// Clean the word
		word = strings.Trim(word, ".,!?;:")
		if len(word) < 3 || stopWords[word] {
			continue
		}
		terms = append(terms, word)
	}

	return terms
}

// findMatchedTerms finds which query terms match in the content
func (f *RAGTestFramework) findMatchedTerms(content, query string) []string {
	queryTerms := f.extractQueryTerms(query)
	matched := make([]string, 0)
	contentLower := strings.ToLower(content)

	for _, term := range queryTerms {
		if strings.Contains(contentLower, strings.ToLower(term)) {
			matched = append(matched, term)
		}
	}

	return matched
}

// applyReranking applies reranking to improve relevance
func (f *RAGTestFramework) applyReranking(result SearchResult, query string) float64 {
	baseScore := result.Score

	// Boost score based on matched terms
	matchedTerms := result.Metadata["matched_terms"].([]string)
	queryTerms := result.Metadata["query_terms"].([]string)

	if len(queryTerms) > 0 {
		matchRatio := float64(len(matchedTerms)) / float64(len(queryTerms))
		baseScore = baseScore * (0.7 + 0.3*matchRatio)
	}

	// Boost based on content type
	if chunkType, ok := result.Metadata["chunk_type"].(string); ok {
		switch chunkType {
		case "code":
			if strings.Contains(strings.ToLower(query), "function") ||
				strings.Contains(strings.ToLower(query), "implementation") {
				baseScore *= 1.1
			}
		case "documentation":
			if strings.Contains(strings.ToLower(query), "how") ||
				strings.Contains(strings.ToLower(query), "usage") {
				baseScore *= 1.1
			}
		}
	}

	// Ensure score doesn't exceed 1.0
	if baseScore > 1.0 {
		baseScore = 1.0
	}

	return baseScore
}

// simpleHash creates a simple hash of text for seeding random generators
func (f *RAGTestFramework) simpleHash(text string) uint32 {
	hash := uint32(5381)
	for _, c := range text {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}

// MockVectorStore implements VectorStore for testing
type MockVectorStore struct {
	documents  map[string]*vector.Document
	embeddings map[string][]float32
}

// NewMockVectorStore creates a new mock vector store
func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		documents:  make(map[string]*vector.Document),
		embeddings: make(map[string][]float32),
	}
}

// Add adds a document to the store
func (m *MockVectorStore) Add(ctx context.Context, doc *vector.Document) error {
	m.documents[doc.ID] = doc
	if doc.Embedding != nil {
		m.embeddings[doc.ID] = doc.Embedding
	}
	return nil
}

// AddBatch adds multiple documents to the store
func (m *MockVectorStore) AddBatch(ctx context.Context, docs []*vector.Document) error {
	for _, doc := range docs {
		if err := m.Add(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

// Search performs vector similarity search
func (m *MockVectorStore) Search(ctx context.Context, query []float32, k int, filter map[string]interface{}) ([]*vector.Document, error) {
	// Simple similarity search implementation
	type docScore struct {
		doc   *vector.Document
		score float64
	}

	scores := make([]docScore, 0)

	for _, doc := range m.documents {
		embedding := m.embeddings[doc.ID]
		if embedding == nil {
			continue
		}

		// Calculate cosine similarity
		similarity := m.cosineSimilarity(query, embedding)
		scores = append(scores, docScore{doc: doc, score: similarity})
	}

	// Sort by score
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score < scores[j].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Return top k
	if k > len(scores) {
		k = len(scores)
	}

	results := make([]*vector.Document, k)
	for i := 0; i < k; i++ {
		results[i] = scores[i].doc
	}

	return results, nil
}

// SimilaritySearch performs similarity search with scores
func (m *MockVectorStore) SimilaritySearch(ctx context.Context, queryText string, k int, threshold float64) ([]*vector.Document, []float64, error) {
	// For testing, we'll do a simple text-based similarity
	type docScore struct {
		doc   *vector.Document
		score float64
	}

	scores := make([]docScore, 0)
	queryLower := strings.ToLower(queryText)

	for _, doc := range m.documents {
		// Simple text similarity based on common words
		contentLower := strings.ToLower(doc.Content)
		score := m.calculateTextSimilarity(queryLower, contentLower)

		if score >= threshold {
			scores = append(scores, docScore{doc: doc, score: score})
		}
	}

	// Sort by score
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score < scores[j].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Return top k
	if k > len(scores) {
		k = len(scores)
	}

	docs := make([]*vector.Document, k)
	scoreValues := make([]float64, k)
	for i := 0; i < k; i++ {
		docs[i] = scores[i].doc
		scoreValues[i] = scores[i].score
	}

	return docs, scoreValues, nil
}

// Update updates a document
func (m *MockVectorStore) Update(ctx context.Context, docID string, doc *vector.Document) error {
	m.documents[docID] = doc
	if doc.Embedding != nil {
		m.embeddings[docID] = doc.Embedding
	}
	return nil
}

// Delete deletes a document
func (m *MockVectorStore) Delete(ctx context.Context, docID string) error {
	delete(m.documents, docID)
	delete(m.embeddings, docID)
	return nil
}

// Count returns the number of documents
func (m *MockVectorStore) Count() int {
	return len(m.documents)
}

// GetStats returns store statistics
func (m *MockVectorStore) GetStats() VectorStoreStats {
	return VectorStoreStats{
		TotalDocuments: len(m.documents),
		TotalVectors:   len(m.embeddings),
		IndexSize:      int64(len(m.documents) * 1024), // Estimate
		MemoryUsage:    int64(len(m.documents) * 2048), // Estimate
	}
}

// cosineSimilarity calculates cosine similarity between two vectors
func (m *MockVectorStore) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// calculateTextSimilarity calculates simple text similarity
func (m *MockVectorStore) calculateTextSimilarity(query, content string) float64 {
	queryWords := strings.Fields(query)
	contentWords := strings.Fields(content)

	if len(queryWords) == 0 {
		return 0.0
	}

	matches := 0
	for _, qWord := range queryWords {
		for _, cWord := range contentWords {
			if qWord == cWord {
				matches++
				break
			}
		}
	}

	// Add some randomness for testing variety
	baseScore := float64(matches) / float64(len(queryWords))
	noise := (rand.Float64() - 0.5) * 0.2 // ±10% noise

	return math.Max(0.0, math.Min(1.0, baseScore+noise))
}

// getMetadataString safely extracts a string value from metadata
func getMetadataString(metadata map[string]interface{}, key string) string {
	if val, ok := metadata[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
