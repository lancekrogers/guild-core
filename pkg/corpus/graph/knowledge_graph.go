// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package graph

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/corpus/extraction"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// KnowledgeGraph provides a graph-based representation of extracted knowledge
type KnowledgeGraph struct {
	mu    sync.RWMutex
	nodes map[string]*KnowledgeNode
	edges map[string][]*Edge
	index *GraphIndex
}

// NewKnowledgeGraph creates a new knowledge graph
func NewKnowledgeGraph(ctx context.Context) (*KnowledgeGraph, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.graph").
			WithOperation("NewKnowledgeGraph")
	}

	index, err := NewGraphIndex(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create graph index").
			WithComponent("corpus.graph").
			WithOperation("NewKnowledgeGraph")
	}

	return &KnowledgeGraph{
		nodes: make(map[string]*KnowledgeNode),
		edges: make(map[string][]*Edge),
		index: index,
	}, nil
}

// AddKnowledge adds extracted knowledge to the graph
func (kg *KnowledgeGraph) AddKnowledge(ctx context.Context, knowledge extraction.ExtractedKnowledge) error {
	if ctx.Err() != nil {
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.graph").
			WithOperation("AddKnowledge")
	}

	kg.mu.Lock()
	defer kg.mu.Unlock()

	// Create or update the main knowledge node
	createdAt := knowledge.Timestamp
	if createdAt.IsZero() {
		createdAt = time.Now() // Use current time if timestamp is missing
	}

	node := &KnowledgeNode{
		ID:         knowledge.ID,
		Type:       kg.mapKnowledgeType(knowledge.Type),
		Content:    knowledge.Content,
		Properties: kg.buildNodeProperties(knowledge),
		CreatedAt:  createdAt,
		UpdatedAt:  time.Now(),
		Confidence: knowledge.Confidence,
	}

	kg.nodes[node.ID] = node

	// Create entity nodes and edges
	for _, entity := range knowledge.Entities {
		entityNode := kg.getOrCreateEntityNode(entity)
		edge := &Edge{
			From:       node.ID,
			To:         entityNode.ID,
			Type:       EdgeMentions,
			Properties: map[string]interface{}{"entity_type": entity.Type},
			Weight:     entity.Confidence,
			CreatedAt:  time.Now(),
		}
		kg.addEdge(edge)
	}

	// Create relation edges
	for _, relation := range knowledge.Relations {
		subjectNode := kg.getOrCreateConceptNode(relation.Subject)
		objectNode := kg.getOrCreateConceptNode(relation.Object)

		edge := &Edge{
			From:       subjectNode.ID,
			To:         objectNode.ID,
			Type:       kg.mapRelationType(relation.Predicate),
			Properties: map[string]interface{}{"predicate": relation.Predicate},
			Weight:     relation.Confidence,
			CreatedAt:  time.Now(),
		}
		kg.addEdge(edge)

		// Link the knowledge node to the relation
		knowledgeToSubject := &Edge{
			From:      node.ID,
			To:        subjectNode.ID,
			Type:      EdgeContains,
			Weight:    0.5,
			CreatedAt: time.Now(),
		}
		kg.addEdge(knowledgeToSubject)
	}

	// Find and link related knowledge
	relatedNodes, err := kg.findRelatedKnowledge(ctx, knowledge)
	if err == nil {
		for _, relatedNode := range relatedNodes {
			edge := &Edge{
				From:      node.ID,
				To:        relatedNode.ID,
				Type:      EdgeRelatedTo,
				Weight:    kg.calculateRelationWeight(knowledge, relatedNode),
				CreatedAt: time.Now(),
			}
			kg.addEdge(edge)
		}
	}

	// Update the index
	if err := kg.index.UpdateNode(ctx, node); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update graph index").
			WithComponent("corpus.graph").
			WithOperation("AddKnowledge").
			WithDetails("node_id", node.ID)
	}

	return nil
}

// Query performs a graph query and returns matching nodes
func (kg *KnowledgeGraph) Query(ctx context.Context, query GraphQuery) ([]*KnowledgeNode, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.graph").
			WithOperation("Query")
	}

	kg.mu.RLock()
	defer kg.mu.RUnlock()

	// Find starting nodes based on query
	startNodes, err := kg.findStartNodes(ctx, query)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find start nodes").
			WithComponent("corpus.graph").
			WithOperation("Query")
	}

	if len(startNodes) == 0 {
		return []*KnowledgeNode{}, nil
	}

	// Traverse the graph from start nodes
	visited := make(map[string]bool)
	var results []*KnowledgeNode

	for _, startNode := range startNodes {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		kg.traverse(ctx, startNode, query, visited, &results)
	}

	// Rank and filter results
	rankedResults := kg.rankResults(results, query)

	// Apply limit
	if query.Limit > 0 && len(rankedResults) > query.Limit {
		rankedResults = rankedResults[:query.Limit]
	}

	return rankedResults, nil
}

// traverse performs depth-first traversal of the graph
func (kg *KnowledgeGraph) traverse(ctx context.Context, node *KnowledgeNode, query GraphQuery,
	visited map[string]bool, results *[]*KnowledgeNode,
) {
	if ctx.Err() != nil || visited[node.ID] {
		return
	}

	visited[node.ID] = true

	// Check if current node matches query criteria
	if kg.matchesQuery(node, query) {
		*results = append(*results, node)
	}

	// Follow edges based on query parameters
	edges := kg.edges[node.ID]
	for _, edge := range edges {
		// Check edge type filter
		if query.EdgeTypes != nil && !kg.containsEdgeType(query.EdgeTypes, edge.Type) {
			continue
		}

		// Check minimum weight
		if edge.Weight < query.MinWeight {
			continue
		}

		// Check maximum depth
		if query.MaxDepth == 0 {
			// MaxDepth 0 means no traversal - only return start nodes
			continue
		}
		if query.MaxDepth > 0 && kg.getDepthFromQuery(query, edge) > query.MaxDepth {
			continue
		}

		// Continue traversal
		if nextNode, exists := kg.nodes[edge.To]; exists {
			kg.traverse(ctx, nextNode, query, visited, results)
		}
	}
}

// findStartNodes identifies nodes to begin traversal from using OR semantics for combining filters
func (kg *KnowledgeGraph) findStartNodes(ctx context.Context, query GraphQuery) ([]*KnowledgeNode, error) {
	if ctx.Err() != nil {
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("corpus.graph").
			WithOperation("findStartNodes")
	}

	candidateMap := make(map[string]*candidateNode)

	// Collect text-based candidates
	if query.Text != "" {
		if err := kg.addTextCandidates(ctx, query.Text, candidateMap); err != nil {
			// Log error but continue - don't fail the entire query for text search issues
		}
	}

	// Collect type-based candidates (OR semantics, not AND)
	if query.NodeTypes != nil {
		kg.addTypeCandidates(query.NodeTypes, candidateMap)
	}

	// Convert candidates to slice and apply additional filters
	var startNodes []*KnowledgeNode
	for _, candidate := range candidateMap {
		node := candidate.node

		// Apply confidence filter
		if node.Confidence < query.MinConfidence {
			continue
		}

		// Apply time range filter
		if !query.TimeRange.IsZero() && node.CreatedAt.Before(query.TimeRange) {
			continue
		}

		startNodes = append(startNodes, node)
	}

	// If no candidates found, apply progressive fallback strategy
	if len(startNodes) == 0 {
		startNodes = kg.applyFallbackStrategy(ctx, query)
	}

	return startNodes, nil
}

// candidateNode tracks potential starting nodes with metadata for ranking
type candidateNode struct {
	node      *KnowledgeNode
	textScore float64
	typeMatch bool
	source    string // "text", "type", "fallback"
}

// addTextCandidates adds nodes that match text criteria
func (kg *KnowledgeGraph) addTextCandidates(ctx context.Context, text string, candidates map[string]*candidateNode) error {
	indexResults, err := kg.index.Search(ctx, text, 50) // Increased limit for better recall
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "text search failed").
			WithComponent("corpus.graph").
			WithOperation("addTextCandidates")
	}

	for _, result := range indexResults {
		if node, exists := kg.nodes[result.NodeID]; exists {
			if existing, found := candidates[result.NodeID]; found {
				// Update existing candidate with better text score
				if result.Score > existing.textScore {
					existing.textScore = result.Score
				}
			} else {
				candidates[result.NodeID] = &candidateNode{
					node:      node,
					textScore: result.Score,
					typeMatch: false,
					source:    "text",
				}
			}
		}
	}

	return nil
}

// addTypeCandidates adds nodes that match type criteria
func (kg *KnowledgeGraph) addTypeCandidates(nodeTypes []NodeType, candidates map[string]*candidateNode) {
	for _, node := range kg.nodes {
		if kg.containsNodeType(nodeTypes, node.Type) {
			if existing, found := candidates[node.ID]; found {
				// Mark existing candidate as also matching type
				existing.typeMatch = true
			} else {
				candidates[node.ID] = &candidateNode{
					node:      node,
					textScore: 0.0,
					typeMatch: true,
					source:    "type",
				}
			}
		}
	}
}

// applyFallbackStrategy implements progressive fallback when no candidates found
func (kg *KnowledgeGraph) applyFallbackStrategy(ctx context.Context, query GraphQuery) []*KnowledgeNode {
	var fallbackNodes []*KnowledgeNode

	// Strategy 1: High-confidence nodes
	if len(fallbackNodes) == 0 {
		for _, node := range kg.nodes {
			if node.Confidence >= 0.8 {
				fallbackNodes = append(fallbackNodes, node)
			}
		}
	}

	// Strategy 2: Any nodes matching partial criteria (relaxed confidence)
	if len(fallbackNodes) == 0 && query.NodeTypes != nil {
		for _, node := range kg.nodes {
			if kg.containsNodeType(query.NodeTypes, node.Type) && node.Confidence >= 0.5 {
				fallbackNodes = append(fallbackNodes, node)
			}
		}
	}

	// Strategy 3: Recent nodes (last resort)
	if len(fallbackNodes) == 0 {
		cutoff := time.Now().Add(-24 * time.Hour)
		for _, node := range kg.nodes {
			if node.CreatedAt.After(cutoff) {
				fallbackNodes = append(fallbackNodes, node)
			}
		}
	}

	return fallbackNodes
}

// matchesQuery checks if a node matches the query criteria using OR semantics for primary filters
func (kg *KnowledgeGraph) matchesQuery(node *KnowledgeNode, query GraphQuery) bool {
	// Always apply confidence and time range filters (these are constraints, not search criteria)
	if node.Confidence < query.MinConfidence {
		return false
	}

	if !query.TimeRange.IsZero() && node.CreatedAt.Before(query.TimeRange) {
		return false
	}

	// Use OR semantics for text and type filters when both are specified
	hasTextFilter := query.Text != ""
	hasTypeFilter := query.NodeTypes != nil

	if hasTextFilter && hasTypeFilter {
		// OR logic: match if either text OR type matches
		textMatches := false
		typeMatches := kg.containsNodeType(query.NodeTypes, node.Type)

		if hasTextFilter {
			contentLower := strings.ToLower(node.Content)
			// Split query text into words and check if any word matches (more flexible than full string match)
			queryWords := strings.Fields(strings.ToLower(query.Text))
			for _, word := range queryWords {
				if strings.Contains(contentLower, word) {
					textMatches = true
					break
				}
			}
		}

		return textMatches || typeMatches
	} else if hasTypeFilter {
		// Type filter only
		return kg.containsNodeType(query.NodeTypes, node.Type)
	} else if hasTextFilter {
		// Text filter only
		contentLower := strings.ToLower(node.Content)
		queryWords := strings.Fields(strings.ToLower(query.Text))
		for _, word := range queryWords {
			if strings.Contains(contentLower, word) {
				return true
			}
		}
		return false
	}

	// No text or type filters specified
	return true
}

// rankResults ranks query results by relevance
func (kg *KnowledgeGraph) rankResults(results []*KnowledgeNode, query GraphQuery) []*KnowledgeNode {
	// Create a slice of result pairs for sorting
	type scoredResult struct {
		node  *KnowledgeNode
		score float64
	}

	var scoredResults []scoredResult

	for _, node := range results {
		score := kg.calculateRelevanceScore(node, query)
		scoredResults = append(scoredResults, scoredResult{node: node, score: score})
	}

	// Sort by score descending
	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].score > scoredResults[j].score
	})

	// Extract sorted nodes
	var rankedNodes []*KnowledgeNode
	for _, result := range scoredResults {
		rankedNodes = append(rankedNodes, result.node)
	}

	return rankedNodes
}

// calculateRelevanceScore calculates relevance score for ranking
func (kg *KnowledgeGraph) calculateRelevanceScore(node *KnowledgeNode, query GraphQuery) float64 {
	score := node.Confidence * 0.3 // Base confidence score

	// Text relevance
	if query.Text != "" {
		textScore := kg.calculateTextRelevance(node.Content, query.Text)
		score += textScore * 0.4
	}

	// Recency bonus
	daysSinceCreation := time.Since(node.CreatedAt).Hours() / 24
	if daysSinceCreation < 30 {
		score += (30 - daysSinceCreation) / 30 * 0.2
	}

	// Connection count bonus (well-connected nodes are more important)
	connectionCount := len(kg.edges[node.ID])
	score += float64(connectionCount) / 100 * 0.1

	return score
}

// calculateTextRelevance calculates text relevance score
func (kg *KnowledgeGraph) calculateTextRelevance(content, queryText string) float64 {
	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(queryText)

	// Exact match bonus
	if strings.Contains(contentLower, queryLower) {
		return 1.0
	}

	// Word match scoring
	queryWords := strings.Fields(queryLower)
	contentWords := strings.Fields(contentLower)

	matches := 0
	for _, queryWord := range queryWords {
		for _, contentWord := range contentWords {
			if queryWord == contentWord {
				matches++
				break
			}
		}
	}

	if len(queryWords) > 0 {
		return float64(matches) / float64(len(queryWords))
	}

	return 0.0
}

// Helper methods

func (kg *KnowledgeGraph) mapKnowledgeType(extractionType extraction.KnowledgeType) NodeType {
	switch extractionType {
	case extraction.KnowledgeDecision:
		return NodeDecision
	case extraction.KnowledgeSolution:
		return NodeSolution
	case extraction.KnowledgePattern:
		return NodePattern
	case extraction.KnowledgePreference:
		return NodePreference
	case extraction.KnowledgeConstraint:
		return NodeConstraint
	case extraction.KnowledgeContext:
		return NodeContext
	default:
		return NodeConcept
	}
}

func (kg *KnowledgeGraph) mapRelationType(predicate string) EdgeType {
	switch strings.ToLower(predicate) {
	case "uses", "using", "utilizes":
		return EdgeUses
	case "depends_on", "requires", "needs":
		return EdgeDependsOn
	case "replaces", "replacing", "substitutes":
		return EdgeSupersedes
	case "implements", "implementing", "extends":
		return EdgeImplements
	case "integrates_with", "connects_to":
		return EdgeRelatedTo
	default:
		return EdgeRelatedTo
	}
}

func (kg *KnowledgeGraph) buildNodeProperties(knowledge extraction.ExtractedKnowledge) map[string]interface{} {
	properties := make(map[string]interface{})

	// Copy metadata
	for key, value := range knowledge.Metadata {
		properties[key] = value
	}

	// Add knowledge-specific properties
	sourceType := knowledge.Source.Type
	if sourceType == "" {
		sourceType = "unknown" // Graceful fallback for missing source
	}
	properties["source_type"] = sourceType
	properties["entity_count"] = len(knowledge.Entities)
	properties["relation_count"] = len(knowledge.Relations)
	properties["confidence"] = knowledge.Confidence

	if knowledge.Source.SessionID != "" {
		properties["session_id"] = knowledge.Source.SessionID
	}

	return properties
}

func (kg *KnowledgeGraph) getOrCreateEntityNode(entity extraction.Entity) *KnowledgeNode {
	nodeID := fmt.Sprintf("entity_%s_%s", entity.Type, strings.ReplaceAll(entity.Name, " ", "_"))

	if existing, exists := kg.nodes[nodeID]; exists {
		return existing
	}

	node := &KnowledgeNode{
		ID:      nodeID,
		Type:    NodeEntity,
		Content: entity.Name,
		Properties: map[string]interface{}{
			"entity_type": entity.Type,
			"confidence":  entity.Confidence,
		},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Confidence: entity.Confidence,
	}

	kg.nodes[nodeID] = node
	return node
}

func (kg *KnowledgeGraph) getOrCreateConceptNode(concept string) *KnowledgeNode {
	nodeID := fmt.Sprintf("concept_%s", strings.ReplaceAll(concept, " ", "_"))

	if existing, exists := kg.nodes[nodeID]; exists {
		return existing
	}

	node := &KnowledgeNode{
		ID:         nodeID,
		Type:       NodeConcept,
		Content:    concept,
		Properties: map[string]interface{}{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Confidence: 0.7, // Default confidence for concepts
	}

	kg.nodes[nodeID] = node
	return node
}

func (kg *KnowledgeGraph) addEdge(edge *Edge) {
	edgeID := fmt.Sprintf("%s-%s-%s", edge.From, edge.Type.String(), edge.To)
	edge.ID = edgeID

	// Add to forward edges
	kg.edges[edge.From] = append(kg.edges[edge.From], edge)
}

func (kg *KnowledgeGraph) findRelatedKnowledge(ctx context.Context, knowledge extraction.ExtractedKnowledge) ([]*KnowledgeNode, error) {
	var related []*KnowledgeNode

	// Find nodes with similar entities
	for _, entity := range knowledge.Entities {
		entityNodeID := fmt.Sprintf("entity_%s_%s", entity.Type, strings.ReplaceAll(entity.Name, " ", "_"))

		// Find edges from this entity to other knowledge nodes
		if edges, exists := kg.edges[entityNodeID]; exists {
			for _, edge := range edges {
				if edge.Type == EdgeMentions {
					if node, exists := kg.nodes[edge.To]; exists && node.ID != knowledge.ID {
						related = append(related, node)
					}
				}
			}
		}
	}

	return related, nil
}

func (kg *KnowledgeGraph) calculateRelationWeight(knowledge extraction.ExtractedKnowledge, relatedNode *KnowledgeNode) float64 {
	// Base weight from confidence
	weight := (knowledge.Confidence + relatedNode.Confidence) / 2

	// Increase weight for same type
	if kg.mapKnowledgeType(knowledge.Type) == relatedNode.Type {
		weight += 0.1
	}

	// Increase weight for recent knowledge
	timeDiff := time.Since(relatedNode.CreatedAt)
	if timeDiff < 24*time.Hour {
		weight += 0.1
	}

	return weight
}

// Utility methods

func (kg *KnowledgeGraph) containsNodeType(types []NodeType, nodeType NodeType) bool {
	for _, t := range types {
		if t == nodeType {
			return true
		}
	}
	return false
}

func (kg *KnowledgeGraph) containsEdgeType(types []EdgeType, edgeType EdgeType) bool {
	for _, t := range types {
		if t == edgeType {
			return true
		}
	}
	return false
}

func (kg *KnowledgeGraph) getDepthFromQuery(query GraphQuery, edge *Edge) int {
	// Simplified depth calculation - in a real implementation,
	// you'd track depth during traversal
	return 1
}

// GetStats returns statistics about the knowledge graph
func (kg *KnowledgeGraph) GetStats(ctx context.Context) (*GraphStats, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	kg.mu.RLock()
	defer kg.mu.RUnlock()

	stats := &GraphStats{
		NodeCount: len(kg.nodes),
		EdgeCount: 0,
		NodeTypes: make(map[string]int),
		EdgeTypes: make(map[string]int),
	}

	// Count nodes by type
	for _, node := range kg.nodes {
		stats.NodeTypes[node.Type.String()]++
	}

	// Count edges and edge types
	for _, edges := range kg.edges {
		stats.EdgeCount += len(edges)
		for _, edge := range edges {
			stats.EdgeTypes[edge.Type.String()]++
		}
	}

	return stats, nil
}
