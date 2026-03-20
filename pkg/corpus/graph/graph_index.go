// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package graph

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// GraphIndex provides efficient searching and indexing for the knowledge graph
type GraphIndex struct {
	mu              sync.RWMutex
	textIndex       map[string][]string   // word -> node IDs
	nodeContent     map[string]string     // node ID -> content
	typeIndex       map[NodeType][]string // node type -> node IDs
	confidenceIndex map[string]float64    // node ID -> confidence
	inverseDocFreq  map[string]float64    // word -> IDF score
	totalNodes      int
}

// NewGraphIndex creates a new graph index
func NewGraphIndex(ctx context.Context) (*GraphIndex, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.graph").
			WithOperation("NewGraphIndex")
	}

	return &GraphIndex{
		textIndex:       make(map[string][]string),
		nodeContent:     make(map[string]string),
		typeIndex:       make(map[NodeType][]string),
		confidenceIndex: make(map[string]float64),
		inverseDocFreq:  make(map[string]float64),
		totalNodes:      0,
	}, nil
}

// UpdateNode updates the index with a new or modified node
func (gi *GraphIndex) UpdateNode(ctx context.Context, node *KnowledgeNode) error {
	if ctx.Err() != nil {
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.graph").
			WithOperation("UpdateNode")
	}

	gi.mu.Lock()
	defer gi.mu.Unlock()

	// Remove existing entries for this node
	gi.removeNodeFromIndex(node.ID)

	// Add node to content index
	gi.nodeContent[node.ID] = node.Content
	gi.confidenceIndex[node.ID] = node.Confidence

	// Add to type index
	gi.typeIndex[node.Type] = append(gi.typeIndex[node.Type], node.ID)

	// Add to text index
	words := gi.extractWords(node.Content)
	for _, word := range words {
		gi.textIndex[word] = append(gi.textIndex[word], node.ID)
	}

	gi.totalNodes++

	// Update IDF scores periodically
	if gi.totalNodes%10 == 0 {
		gi.updateInverseDocumentFrequency()
	}

	return nil
}

// RemoveNode removes a node from the index
func (gi *GraphIndex) RemoveNode(ctx context.Context, nodeID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	gi.mu.Lock()
	defer gi.mu.Unlock()

	gi.removeNodeFromIndex(nodeID)
	gi.totalNodes--

	return nil
}

// Search performs a text search and returns ranked results
func (gi *GraphIndex) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.graph").
			WithOperation("Search")
	}

	gi.mu.RLock()
	defer gi.mu.RUnlock()

	if query == "" {
		return []SearchResult{}, nil
	}

	// Extract query words
	queryWords := gi.extractWords(query)
	if len(queryWords) == 0 {
		return []SearchResult{}, nil
	}

	// Calculate TF-IDF scores for candidate nodes
	candidates := make(map[string]float64)

	for _, word := range queryWords {
		if nodeIDs, exists := gi.textIndex[word]; exists {
			idf := gi.getIDF(word)

			for _, nodeID := range nodeIDs {
				content := gi.nodeContent[nodeID]
				tf := gi.calculateTermFrequency(word, content)
				score := tf * idf

				// Boost score by node confidence
				confidence := gi.confidenceIndex[nodeID]
				score *= (0.5 + confidence*0.5)

				candidates[nodeID] += score
			}
		}
	}

	// Convert to sorted results
	var results []SearchResult
	for nodeID, score := range candidates {
		results = append(results, SearchResult{
			NodeID:    nodeID,
			Score:     score,
			Highlight: gi.generateHighlight(gi.nodeContent[nodeID], queryWords),
		})
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// SearchByType searches for nodes of specific types
func (gi *GraphIndex) SearchByType(ctx context.Context, nodeTypes []NodeType, limit int) ([]SearchResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	gi.mu.RLock()
	defer gi.mu.RUnlock()

	var results []SearchResult
	seen := make(map[string]bool)

	for _, nodeType := range nodeTypes {
		if nodeIDs, exists := gi.typeIndex[nodeType]; exists {
			for _, nodeID := range nodeIDs {
				if !seen[nodeID] {
					seen[nodeID] = true
					confidence := gi.confidenceIndex[nodeID]
					results = append(results, SearchResult{
						NodeID: nodeID,
						Score:  confidence, // Use confidence as score
					})
				}
			}
		}
	}

	// Sort by score (confidence) descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetSimilarNodes finds nodes similar to the given node
func (gi *GraphIndex) GetSimilarNodes(ctx context.Context, nodeID string, limit int) ([]SearchResult, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	gi.mu.RLock()
	defer gi.mu.RUnlock()

	content, exists := gi.nodeContent[nodeID]
	if !exists {
		return []SearchResult{}, nil
	}

	// Use the node's content as the search query
	return gi.Search(ctx, content, limit+1) // +1 to account for the node itself
}

// Helper methods

func (gi *GraphIndex) removeNodeFromIndex(nodeID string) {
	// Remove from content and confidence indices
	content, hasContent := gi.nodeContent[nodeID]
	delete(gi.nodeContent, nodeID)
	delete(gi.confidenceIndex, nodeID)

	// Remove from text index
	if hasContent {
		words := gi.extractWords(content)
		for _, word := range words {
			if nodeIDs, exists := gi.textIndex[word]; exists {
				// Remove nodeID from the slice
				for i, id := range nodeIDs {
					if id == nodeID {
						gi.textIndex[word] = append(nodeIDs[:i], nodeIDs[i+1:]...)
						break
					}
				}
				// Remove empty entries
				if len(gi.textIndex[word]) == 0 {
					delete(gi.textIndex, word)
				}
			}
		}
	}

	// Remove from type index
	for nodeType, nodeIDs := range gi.typeIndex {
		for i, id := range nodeIDs {
			if id == nodeID {
				gi.typeIndex[nodeType] = append(nodeIDs[:i], nodeIDs[i+1:]...)
				break
			}
		}
		// Remove empty entries
		if len(gi.typeIndex[nodeType]) == 0 {
			delete(gi.typeIndex, nodeType)
		}
	}
}

func (gi *GraphIndex) extractWords(text string) []string {
	// Simple word extraction - could be enhanced with stemming, stop words, etc.
	text = strings.ToLower(text)
	words := strings.FieldsFunc(text, func(c rune) bool {
		return !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'))
	})

	// Filter out very short words and common stop words
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "by": true, "for": true, "from": true, "has": true, "he": true,
		"in": true, "is": true, "it": true, "its": true, "of": true, "on": true,
		"that": true, "the": true, "to": true, "was": true, "will": true, "with": true,
	}

	var filtered []string
	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

func (gi *GraphIndex) calculateTermFrequency(term, content string) float64 {
	words := gi.extractWords(content)
	if len(words) == 0 {
		return 0.0
	}

	count := 0
	for _, word := range words {
		if word == term {
			count++
		}
	}

	return float64(count) / float64(len(words))
}

func (gi *GraphIndex) getIDF(term string) float64 {
	if idf, exists := gi.inverseDocFreq[term]; exists {
		return idf
	}

	// Calculate IDF on demand if not cached
	documentsWithTerm := len(gi.textIndex[term])
	if documentsWithTerm == 0 || gi.totalNodes == 0 {
		return 0.0
	}

	// IDF = log(total documents / documents containing term)
	idf := math.Log(float64(gi.totalNodes) / float64(documentsWithTerm))
	gi.inverseDocFreq[term] = idf
	return idf
}

func (gi *GraphIndex) updateInverseDocumentFrequency() {
	// Recalculate IDF for all terms
	for term := range gi.textIndex {
		documentsWithTerm := len(gi.textIndex[term])
		if documentsWithTerm > 0 && gi.totalNodes > 0 {
			gi.inverseDocFreq[term] = math.Log(float64(gi.totalNodes) / float64(documentsWithTerm))
		}
	}
}

func (gi *GraphIndex) generateHighlight(content string, queryWords []string) string {
	if len(queryWords) == 0 {
		return ""
	}

	contentLower := strings.ToLower(content)

	// Find the first occurrence of any query word
	for _, word := range queryWords {
		if idx := strings.Index(contentLower, word); idx != -1 {
			// Extract context around the word
			start := idx - 30
			if start < 0 {
				start = 0
			}
			end := idx + len(word) + 30
			if end > len(content) {
				end = len(content)
			}

			highlight := content[start:end]
			if start > 0 {
				highlight = "..." + highlight
			}
			if end < len(content) {
				highlight = highlight + "..."
			}

			return highlight
		}
	}

	// Fallback: return first 60 characters
	if len(content) > 60 {
		return content[:60] + "..."
	}
	return content
}

// GetIndexStats returns statistics about the index
func (gi *GraphIndex) GetIndexStats(ctx context.Context) (*IndexStats, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	gi.mu.RLock()
	defer gi.mu.RUnlock()

	stats := &IndexStats{
		TotalNodes:   gi.totalNodes,
		TotalWords:   len(gi.textIndex),
		TypeCounts:   make(map[string]int),
		AverageWords: 0,
	}

	// Count nodes by type
	for nodeType, nodeIDs := range gi.typeIndex {
		stats.TypeCounts[nodeType.String()] = len(nodeIDs)
	}

	// Calculate average words per node
	if gi.totalNodes > 0 {
		totalWordCount := 0
		for _, content := range gi.nodeContent {
			totalWordCount += len(gi.extractWords(content))
		}
		stats.AverageWords = float64(totalWordCount) / float64(gi.totalNodes)
	}

	return stats, nil
}

// ClearIndex removes all entries from the index
func (gi *GraphIndex) ClearIndex(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	gi.mu.Lock()
	defer gi.mu.Unlock()

	gi.textIndex = make(map[string][]string)
	gi.nodeContent = make(map[string]string)
	gi.typeIndex = make(map[NodeType][]string)
	gi.confidenceIndex = make(map[string]float64)
	gi.inverseDocFreq = make(map[string]float64)
	gi.totalNodes = 0

	return nil
}

// IndexStats represents statistics about the graph index
type IndexStats struct {
	TotalNodes   int            `json:"total_nodes"`
	TotalWords   int            `json:"total_words"`
	TypeCounts   map[string]int `json:"type_counts"`
	AverageWords float64        `json:"average_words"`
}
