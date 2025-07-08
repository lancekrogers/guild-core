// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package retrieval

import (
	"context"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// VectorSearchStrategy implements vector-based similarity search
type VectorSearchStrategy struct {
	vectorStore VectorStore
	weight      float64
}

// NewVectorSearchStrategy creates a new vector search strategy
func NewVectorSearchStrategy(vectorStore VectorStore, weight float64) *VectorSearchStrategy {
	return &VectorSearchStrategy{
		vectorStore: vectorStore,
		weight:      weight,
	}
}

// Name returns the strategy name
func (vss *VectorSearchStrategy) Name() string {
	return "vector_search"
}

// Weight returns the strategy weight for result combination
func (vss *VectorSearchStrategy) Weight() float64 {
	return vss.weight
}

// Retrieve performs vector similarity search
func (vss *VectorSearchStrategy) Retrieve(ctx context.Context, query Query) ([]Document, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("VectorSearchStrategy").
		WithOperation("Retrieve")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("VectorSearchStrategy").
			WithOperation("Retrieve")
	}

	// Enhance query with context
	enhancedQuery := vss.enhanceQuery(query)

	// Vector search - request more results than max to allow for post-processing
	searchLimit := query.MaxResults * 2
	if searchLimit < 10 {
		searchLimit = 10
	}

	results, err := vss.vectorStore.Search(ctx, enhancedQuery, searchLimit)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "vector search failed").
			WithComponent("VectorSearchStrategy").
			WithOperation("Retrieve")
	}

	// Convert to documents
	docs := make([]Document, 0, len(results))
	for _, r := range results {
		doc := Document{
			ID:       r.DocID,
			Content:  r.Content,
			Score:    r.Score,
			Metadata: r.Metadata,
		}

		// Add strategy metadata
		if doc.Metadata == nil {
			doc.Metadata = make(map[string]interface{})
		}
		doc.Metadata["strategy"] = "vector_search"
		doc.Metadata["vector_score"] = r.Score

		docs = append(docs, doc)
	}

	logger.Debug("Vector search completed", "results_count", len(docs))
	return docs, nil
}

// enhanceQuery adds contextual information to improve vector search
func (vss *VectorSearchStrategy) enhanceQuery(query Query) string {
	enhanced := query.Text

	// Add current files context
	if len(query.Context.CurrentFiles) > 0 {
		enhanced = enhanced + " files: " + strings.Join(query.Context.CurrentFiles, " ")
	}

	// Add tags context
	if len(query.Context.Tags) > 0 {
		enhanced = enhanced + " tags: " + strings.Join(query.Context.Tags, " ")
	}

	// Add recent user message context (last 2 user messages)
	userMessages := make([]string, 0)
	for i := len(query.Context.MessageHistory) - 1; i >= 0 && len(userMessages) < 2; i-- {
		msg := query.Context.MessageHistory[i]
		if msg.Role == "user" {
			userMessages = append(userMessages, msg.Content)
		}
	}

	// Add user messages in reverse order (most recent first)
	for i := len(userMessages) - 1; i >= 0; i-- {
		enhanced = enhanced + " " + userMessages[i]
	}

	return enhanced
}

// KeywordSearchStrategy implements keyword-based document search
type KeywordSearchStrategy struct {
	indexer KeywordIndexer
	weight  float64
}

// NewKeywordSearchStrategy creates a new keyword search strategy
func NewKeywordSearchStrategy(indexer KeywordIndexer, weight float64) *KeywordSearchStrategy {
	return &KeywordSearchStrategy{
		indexer: indexer,
		weight:  weight,
	}
}

// Name returns the strategy name
func (kss *KeywordSearchStrategy) Name() string {
	return "keyword_search"
}

// Weight returns the strategy weight for result combination
func (kss *KeywordSearchStrategy) Weight() float64 {
	return kss.weight
}

// Retrieve performs keyword-based search
func (kss *KeywordSearchStrategy) Retrieve(ctx context.Context, query Query) ([]Document, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("KeywordSearchStrategy").
		WithOperation("Retrieve")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("KeywordSearchStrategy").
			WithOperation("Retrieve")
	}

	// Extract keywords from query
	keywords := kss.extractKeywords(query.Text)
	if len(keywords) == 0 {
		logger.Debug("No keywords extracted from query")
		return []Document{}, nil
	}

	// Search index
	docs, err := kss.indexer.Search(ctx, keywords, query.MaxResults)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "keyword search failed").
			WithComponent("KeywordSearchStrategy").
			WithOperation("Retrieve")
	}

	// Add strategy metadata
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]interface{})
		}
		docs[i].Metadata["strategy"] = "keyword_search"
		docs[i].Metadata["keywords"] = keywords
	}

	logger.Debug("Keyword search completed", "keywords", keywords, "results_count", len(docs))

	return docs, nil
}

// extractKeywords extracts important keywords from query text
func (kss *KeywordSearchStrategy) extractKeywords(text string) []string {
	// Simple keyword extraction - split on whitespace and filter
	words := strings.Fields(strings.ToLower(text))

	// Filter out common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
		"this": true, "that": true, "these": true, "those": true, "i": true, "you": true,
		"he": true, "she": true, "it": true, "we": true, "they": true, "how": true,
		"what": true, "when": true, "where": true, "why": true, "go": true,
	}

	keywords := make([]string, 0, len(words))
	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:")

		// Skip short words and stop words
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// GraphTraversalStrategy implements knowledge graph-based retrieval
type GraphTraversalStrategy struct {
	knowledgeGraph KnowledgeGraph
	weight         float64
}

// NewGraphTraversalStrategy creates a new graph traversal strategy
func NewGraphTraversalStrategy(knowledgeGraph KnowledgeGraph, weight float64) *GraphTraversalStrategy {
	return &GraphTraversalStrategy{
		knowledgeGraph: knowledgeGraph,
		weight:         weight,
	}
}

// Name returns the strategy name
func (gts *GraphTraversalStrategy) Name() string {
	return "graph_traversal"
}

// Weight returns the strategy weight for result combination
func (gts *GraphTraversalStrategy) Weight() float64 {
	return gts.weight
}

// Retrieve performs graph-based traversal to find related documents
func (gts *GraphTraversalStrategy) Retrieve(ctx context.Context, query Query) ([]Document, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("GraphTraversalStrategy").
		WithOperation("Retrieve")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GraphTraversalStrategy").
			WithOperation("Retrieve")
	}

	// Find entry nodes based on query
	entryNodes, err := gts.knowledgeGraph.FindEntryNodes(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to find entry nodes").
			WithComponent("GraphTraversalStrategy").
			WithOperation("Retrieve")
	}

	if len(entryNodes) == 0 {
		logger.Debug("No entry nodes found for query")
		return []Document{}, nil
	}

	// Traverse graph to find related nodes (2 hops)
	related, err := gts.knowledgeGraph.TraverseRelated(ctx, entryNodes, 2)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "graph traversal failed").
			WithComponent("GraphTraversalStrategy").
			WithOperation("Retrieve")
	}

	// Convert nodes to documents
	docs := gts.nodesToDocuments(related)

	// Add strategy metadata
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]interface{})
		}
		docs[i].Metadata["strategy"] = "graph_traversal"
		docs[i].Metadata["entry_nodes_count"] = len(entryNodes)
		docs[i].Metadata["traversal_hops"] = 2
	}

	logger.Debug("Graph traversal completed", "entry_nodes", len(entryNodes), "related_nodes", len(related), "results_count", len(docs))

	return docs, nil
}

// nodesToDocuments converts graph nodes to retrievable documents
func (gts *GraphTraversalStrategy) nodesToDocuments(nodes []GraphNode) []Document {
	docs := make([]Document, 0, len(nodes))

	for _, node := range nodes {
		doc := Document{
			ID:       node.ID,
			Content:  node.Content,
			Score:    1.0, // Base score, will be adjusted by ranking
			Metadata: node.Metadata,
		}

		// Add node type to metadata
		if doc.Metadata == nil {
			doc.Metadata = make(map[string]interface{})
		}
		doc.Metadata["node_type"] = node.Type

		docs = append(docs, doc)
	}

	return docs
}
