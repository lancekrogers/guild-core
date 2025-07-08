// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scoring

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lancekrogers/guild/pkg/corpus/retrieval"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// RelevanceEngine implements advanced relevance scoring for documents
type RelevanceEngine struct {
	embedder     Embedder
	tokenizer    Tokenizer
	domainModels map[string]DomainModel
}

// Embedder interface for generating text embeddings
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Tokenizer interface for text tokenization
type Tokenizer interface {
	Tokenize(text string) []string
	CountTokens(text string) int
}

// DomainModel represents a domain-specific scoring model
type DomainModel interface {
	Score(ctx context.Context, doc retrieval.Document, context Context) float64
	Domain() string
}

// Context represents the context for relevance calculation
type Context struct {
	Query         string                 `json:"query"`
	Domain        string                 `json:"domain"`
	RequiresFresh bool                   `json:"requires_fresh"`
	UserID        string                 `json:"user_id"`
	TaskID        string                 `json:"task_id"`
	CurrentFiles  []string               `json:"current_files"`
	Tags          []string               `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// RelevanceScore represents a detailed relevance score breakdown
type RelevanceScore struct {
	Overall    float64            `json:"overall"`
	Components map[string]float64 `json:"components"`
	Reasoning  string             `json:"reasoning"`
}

// NewRelevanceEngine creates a new relevance engine
func NewRelevanceEngine(embedder Embedder, tokenizer Tokenizer) *RelevanceEngine {
	return &RelevanceEngine{
		embedder:     embedder,
		tokenizer:    tokenizer,
		domainModels: make(map[string]DomainModel),
	}
}

// RegisterDomainModel registers a domain-specific scoring model
func (re *RelevanceEngine) RegisterDomainModel(model DomainModel) error {
	if model == nil {
		return gerror.New(gerror.ErrCodeValidation, "domain model cannot be nil", nil).
			WithComponent("RelevanceEngine").
			WithOperation("RegisterDomainModel")
	}

	re.domainModels[model.Domain()] = model
	return nil
}

// CalculateRelevance computes a comprehensive relevance score for a document
func (re *RelevanceEngine) CalculateRelevance(ctx context.Context, doc retrieval.Document, context Context) (RelevanceScore, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("RelevanceEngine").
		WithOperation("CalculateRelevance")

	if err := ctx.Err(); err != nil {
		return RelevanceScore{}, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("RelevanceEngine").
			WithOperation("CalculateRelevance")
	}

	score := RelevanceScore{
		Overall:    0.0,
		Components: make(map[string]float64),
	}

	var reasoning strings.Builder

	// Calculate semantic similarity
	semantic, err := re.semanticSimilarity(ctx, doc, context)
	if err != nil {
		logger.WithError(err).Warn("Failed to calculate semantic similarity")
		semantic = 0.0
	}
	score.Components["semantic"] = semantic
	reasoning.WriteString("Semantic similarity: ")
	reasoning.WriteString(formatFloat(semantic))
	reasoning.WriteString("; ")

	// Calculate keyword relevance
	keyword := re.keywordRelevance(doc, context)
	score.Components["keyword"] = keyword
	reasoning.WriteString("Keyword relevance: ")
	reasoning.WriteString(formatFloat(keyword))
	reasoning.WriteString("; ")

	// Calculate temporal relevance
	temporal := re.temporalRelevance(doc, context)
	score.Components["temporal"] = temporal
	reasoning.WriteString("Temporal relevance: ")
	reasoning.WriteString(formatFloat(temporal))
	reasoning.WriteString("; ")

	// Calculate domain-specific relevance
	domainScore := 0.0
	if model := re.domainModels[context.Domain]; model != nil {
		domainScore = model.Score(ctx, doc, context)
		score.Components["domain"] = domainScore
		reasoning.WriteString("Domain relevance: ")
		reasoning.WriteString(formatFloat(domainScore))
		reasoning.WriteString("; ")
	}

	// Calculate citation relevance
	citation := re.citationRelevance(doc, context)
	score.Components["citation"] = citation
	reasoning.WriteString("Citation relevance: ")
	reasoning.WriteString(formatFloat(citation))

	// Calculate weighted overall score
	weights := map[string]float64{
		"semantic": 0.35,
		"keyword":  0.20,
		"temporal": 0.15,
		"domain":   0.20,
		"citation": 0.10,
	}

	for component, value := range score.Components {
		if weight, exists := weights[component]; exists {
			score.Overall += value * weight
		}
	}

	score.Reasoning = reasoning.String()

	logger.Debug("Relevance calculation completed", "overall_score", score.Overall, "components", score.Components)

	return score, nil
}

// semanticSimilarity calculates semantic similarity using embeddings
func (re *RelevanceEngine) semanticSimilarity(ctx context.Context, doc retrieval.Document, context Context) (float64, error) {
	if re.embedder == nil {
		return 0.0, nil
	}

	// Get document embedding (from cache or compute)
	docEmbed, err := re.getOrComputeEmbedding(ctx, doc)
	if err != nil {
		return 0.0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get document embedding").
			WithComponent("RelevanceEngine").
			WithOperation("semanticSimilarity")
	}

	// Get query embedding
	queryEmbed, err := re.embedder.Embed(ctx, context.Query)
	if err != nil {
		return 0.0, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get query embedding").
			WithComponent("RelevanceEngine").
			WithOperation("semanticSimilarity")
	}

	// Calculate cosine similarity
	similarity := cosineSimilarity(docEmbed, queryEmbed)
	return float64(similarity), nil
}

// getOrComputeEmbedding retrieves or computes document embedding
func (re *RelevanceEngine) getOrComputeEmbedding(ctx context.Context, doc retrieval.Document) ([]float32, error) {
	// Check if embedding is cached in metadata
	if embedding, ok := doc.Metadata["embedding"].([]float32); ok {
		return embedding, nil
	}

	// Compute new embedding
	return re.embedder.Embed(ctx, doc.Content)
}

// keywordRelevance calculates relevance based on keyword matching
func (re *RelevanceEngine) keywordRelevance(doc retrieval.Document, context Context) float64 {
	// Tokenize query and document
	var queryTokens []string
	var docTokens []string

	if re.tokenizer != nil {
		queryTokens = re.tokenizer.Tokenize(strings.ToLower(context.Query))
		docTokens = re.tokenizer.Tokenize(strings.ToLower(doc.Content))
	} else {
		// Fallback to simple word splitting
		queryTokens = strings.Fields(strings.ToLower(context.Query))
		docTokens = strings.Fields(strings.ToLower(doc.Content))
	}

	if len(queryTokens) == 0 {
		return 0.0
	}

	// Simple overlap-based scoring: count of query terms found / total query terms
	docTokenSet := make(map[string]bool)
	for _, token := range docTokens {
		docTokenSet[token] = true
	}

	matchCount := 0
	for _, token := range queryTokens {
		if docTokenSet[token] {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(queryTokens))
}

// temporalRelevance calculates relevance based on document age
func (re *RelevanceEngine) temporalRelevance(doc retrieval.Document, context Context) float64 {
	// Get document last modified time
	var lastModified time.Time
	if lm, ok := doc.Metadata["last_modified"].(time.Time); ok {
		lastModified = lm
	} else if lmStr, ok := doc.Metadata["last_modified"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, lmStr); err == nil {
			lastModified = parsed
		} else {
			// If we can't parse the time, assume it's recent
			return 0.7
		}
	} else {
		// No timestamp available, use neutral score
		return 0.5
	}

	age := time.Since(lastModified)

	// Different decay rates for different contexts
	decayRate := 168.0 // Default: weekly decay (hours)
	if context.RequiresFresh {
		decayRate = 24.0 // Daily decay for fresh content
	}

	// Exponential decay function
	relevance := math.Exp(-age.Hours() / decayRate)
	return relevance
}

// citationRelevance calculates relevance based on how often the document is referenced
func (re *RelevanceEngine) citationRelevance(doc retrieval.Document, context Context) float64 {
	// Look for citation count in metadata
	if citationCount, ok := doc.Metadata["citation_count"].(float64); ok {
		// Normalize citation count using log scaling
		if citationCount > 0 {
			return math.Log(citationCount+1) / math.Log(100) // Log scale, max at 100 citations
		}
	}

	// Look for view count as proxy
	if viewCount, ok := doc.Metadata["view_count"].(float64); ok {
		// Normalize view count
		return math.Min(viewCount/50.0, 1.0) // Max at 50 views
	}

	// Look for usage frequency
	if usageFreq, ok := doc.Metadata["usage_frequency"].(float64); ok {
		return math.Min(usageFreq, 1.0)
	}

	// Default neutral score
	return 0.5
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dotProduct, normA, normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// formatFloat formats a float64 to 3 decimal places
func formatFloat(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", f), "0"), ".")
}

// SimpleTokenizer implements basic tokenization
type SimpleTokenizer struct{}

// Tokenize splits text into tokens
func (st *SimpleTokenizer) Tokenize(text string) []string {
	return strings.Fields(text)
}

// CountTokens counts the number of tokens in text
func (st *SimpleTokenizer) CountTokens(text string) int {
	return len(strings.Fields(text))
}

// GolangDomainModel implements domain-specific scoring for Go projects
type GolangDomainModel struct{}

// Domain returns the domain name
func (gdm *GolangDomainModel) Domain() string {
	return "golang"
}

// Score calculates domain-specific relevance for Go projects
func (gdm *GolangDomainModel) Score(ctx context.Context, doc retrieval.Document, context Context) float64 {
	score := 0.0
	content := strings.ToLower(doc.Content)

	// Boost for Go-specific terms
	goTerms := []string{"golang", "go", "package", "func", "interface", "struct", "goroutine", "channel"}
	for _, term := range goTerms {
		if strings.Contains(content, term) {
			score += 0.1
		}
	}

	// Boost for Go file extensions in current files
	for _, file := range context.CurrentFiles {
		if strings.HasSuffix(strings.ToLower(file), ".go") {
			score += 0.2
			break
		}
	}

	// Boost for testing-related content if task involves testing
	if strings.Contains(strings.ToLower(context.Query), "test") {
		testTerms := []string{"testing", "test", "assert", "mock", "benchmark"}
		for _, term := range testTerms {
			if strings.Contains(content, term) {
				score += 0.15
				break
			}
		}
	}

	return math.Min(score, 1.0)
}
