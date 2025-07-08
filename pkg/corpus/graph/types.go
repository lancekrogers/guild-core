// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package graph

import (
	"time"
)

// NodeType represents different types of knowledge nodes
type NodeType int

const (
	NodeConcept NodeType = iota
	NodeEntity
	NodePattern
	NodeSolution
	NodeConstraint
	NodeDecision
	NodePreference
	NodeContext
)

// String returns the string representation of the node type
func (nt NodeType) String() string {
	switch nt {
	case NodeConcept:
		return "concept"
	case NodeEntity:
		return "entity"
	case NodePattern:
		return "pattern"
	case NodeSolution:
		return "solution"
	case NodeConstraint:
		return "constraint"
	case NodeDecision:
		return "decision"
	case NodePreference:
		return "preference"
	case NodeContext:
		return "context"
	default:
		return "unknown"
	}
}

// EdgeType represents different types of relationships between nodes
type EdgeType int

const (
	EdgeRelatedTo EdgeType = iota
	EdgeDependsOn
	EdgeSupersedes
	EdgeContradicts
	EdgeImplements
	EdgeUses
	EdgeMentions
	EdgeContains
)

// String returns the string representation of the edge type
func (et EdgeType) String() string {
	switch et {
	case EdgeRelatedTo:
		return "related_to"
	case EdgeDependsOn:
		return "depends_on"
	case EdgeSupersedes:
		return "supersedes"
	case EdgeContradicts:
		return "contradicts"
	case EdgeImplements:
		return "implements"
	case EdgeUses:
		return "uses"
	case EdgeMentions:
		return "mentions"
	case EdgeContains:
		return "contains"
	default:
		return "unknown"
	}
}

// KnowledgeNode represents a node in the knowledge graph
type KnowledgeNode struct {
	ID         string                 `json:"id"`
	Type       NodeType               `json:"type"`
	Content    string                 `json:"content"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Confidence float64                `json:"confidence"`
}

// Edge represents a relationship between two nodes
type Edge struct {
	ID         string                 `json:"id"`
	From       string                 `json:"from"`
	To         string                 `json:"to"`
	Type       EdgeType               `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Weight     float64                `json:"weight"`
	CreatedAt  time.Time              `json:"created_at"`
}

// GraphQuery represents a query against the knowledge graph
type GraphQuery struct {
	Text          string     `json:"text,omitempty"`
	NodeTypes     []NodeType `json:"node_types,omitempty"`
	EdgeTypes     []EdgeType `json:"edge_types,omitempty"`
	MinConfidence float64    `json:"min_confidence"`
	MinWeight     float64    `json:"min_weight"`
	MaxDepth      int        `json:"max_depth"`
	Limit         int        `json:"limit"`
	TimeRange     time.Time  `json:"time_range,omitempty"`
}

// GraphStats represents statistics about the knowledge graph
type GraphStats struct {
	NodeCount int            `json:"node_count"`
	EdgeCount int            `json:"edge_count"`
	NodeTypes map[string]int `json:"node_types"`
	EdgeTypes map[string]int `json:"edge_types"`
}

// SearchResult represents a search result from the graph index
type SearchResult struct {
	NodeID    string  `json:"node_id"`
	Score     float64 `json:"score"`
	Highlight string  `json:"highlight,omitempty"`
}

// TraversalPath represents a path through the graph during traversal
type TraversalPath struct {
	Nodes []string `json:"nodes"`
	Edges []string `json:"edges"`
	Score float64  `json:"score"`
}

// ClusterInfo represents information about a cluster of related nodes
type ClusterInfo struct {
	ID       string   `json:"id"`
	NodeIDs  []string `json:"node_ids"`
	Label    string   `json:"label"`
	Density  float64  `json:"density"`
	Strength float64  `json:"strength"`
}

// GraphMetrics represents various metrics about the graph structure
type GraphMetrics struct {
	Density            float64            `json:"density"`
	AverageClusterings float64            `json:"average_clustering"`
	ShortestPaths      map[string]float64 `json:"shortest_paths"`
	Centrality         map[string]float64 `json:"centrality"`
	Communities        []ClusterInfo      `json:"communities"`
}

// NodeMetrics represents metrics for individual nodes
type NodeMetrics struct {
	InDegree   int     `json:"in_degree"`
	OutDegree  int     `json:"out_degree"`
	Centrality float64 `json:"centrality"`
	Clustering float64 `json:"clustering"`
	PageRank   float64 `json:"page_rank"`
	Importance float64 `json:"importance"`
}

// QueryBuilder helps build complex graph queries
type QueryBuilder struct {
	query GraphQuery
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: GraphQuery{
			MinConfidence: 0.0,
			MinWeight:     0.0,
			MaxDepth:      5,
			Limit:         100,
		},
	}
}

// WithText sets the text search term
func (qb *QueryBuilder) WithText(text string) *QueryBuilder {
	qb.query.Text = text
	return qb
}

// WithNodeTypes sets the node type filter
func (qb *QueryBuilder) WithNodeTypes(types ...NodeType) *QueryBuilder {
	qb.query.NodeTypes = types
	return qb
}

// WithEdgeTypes sets the edge type filter
func (qb *QueryBuilder) WithEdgeTypes(types ...EdgeType) *QueryBuilder {
	qb.query.EdgeTypes = types
	return qb
}

// WithMinConfidence sets the minimum confidence threshold
func (qb *QueryBuilder) WithMinConfidence(confidence float64) *QueryBuilder {
	qb.query.MinConfidence = confidence
	return qb
}

// WithMinWeight sets the minimum edge weight threshold
func (qb *QueryBuilder) WithMinWeight(weight float64) *QueryBuilder {
	qb.query.MinWeight = weight
	return qb
}

// WithMaxDepth sets the maximum traversal depth
func (qb *QueryBuilder) WithMaxDepth(depth int) *QueryBuilder {
	qb.query.MaxDepth = depth
	return qb
}

// WithLimit sets the maximum number of results
func (qb *QueryBuilder) WithLimit(limit int) *QueryBuilder {
	qb.query.Limit = limit
	return qb
}

// WithTimeRange sets the time range filter
func (qb *QueryBuilder) WithTimeRange(since time.Time) *QueryBuilder {
	qb.query.TimeRange = since
	return qb
}

// Build returns the constructed query
func (qb *QueryBuilder) Build() GraphQuery {
	return qb.query
}

// Predefined query builders for common use cases

// FindSolutions creates a query to find solution nodes
func FindSolutions() *QueryBuilder {
	return NewQueryBuilder().WithNodeTypes(NodeSolution).WithMinConfidence(0.7)
}

// FindDecisions creates a query to find decision nodes
func FindDecisions() *QueryBuilder {
	return NewQueryBuilder().WithNodeTypes(NodeDecision).WithMinConfidence(0.6)
}

// FindPatterns creates a query to find pattern nodes
func FindPatterns() *QueryBuilder {
	return NewQueryBuilder().WithNodeTypes(NodePattern).WithMinConfidence(0.5)
}

// FindRelated creates a query to find nodes related by specific edge types
func FindRelated(edgeTypes ...EdgeType) *QueryBuilder {
	return NewQueryBuilder().WithEdgeTypes(edgeTypes...).WithMaxDepth(3)
}

// FindRecent creates a query to find recent knowledge
func FindRecent(since time.Time) *QueryBuilder {
	return NewQueryBuilder().WithTimeRange(since).WithMinConfidence(0.5)
}

// FindHighConfidence creates a query to find high-confidence knowledge
func FindHighConfidence() *QueryBuilder {
	return NewQueryBuilder().WithMinConfidence(0.8).WithMinWeight(0.7)
}

// GraphExport represents the graph in an exportable format
type GraphExport struct {
	Nodes      []*KnowledgeNode       `json:"nodes"`
	Edges      []*Edge                `json:"edges"`
	Metadata   map[string]interface{} `json:"metadata"`
	ExportedAt time.Time              `json:"exported_at"`
	Version    string                 `json:"version"`
}

// ValidationResult represents the result of graph validation
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Issues      []ValidationIssue `json:"issues"`
	Warnings    []string          `json:"warnings"`
	NodeCount   int               `json:"node_count"`
	EdgeCount   int               `json:"edge_count"`
	Orphaned    []string          `json:"orphaned_nodes"`
	Duplicates  []string          `json:"duplicate_nodes"`
	ValidatedAt time.Time         `json:"validated_at"`
}

// ValidationIssue represents a specific validation issue
type ValidationIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	NodeID      string `json:"node_id,omitempty"`
	EdgeID      string `json:"edge_id,omitempty"`
}

// GraphDiff represents differences between two graph states
type GraphDiff struct {
	AddedNodes   []*KnowledgeNode `json:"added_nodes"`
	RemovedNodes []*KnowledgeNode `json:"removed_nodes"`
	UpdatedNodes []*KnowledgeNode `json:"updated_nodes"`
	AddedEdges   []*Edge          `json:"added_edges"`
	RemovedEdges []*Edge          `json:"removed_edges"`
	Timestamp    time.Time        `json:"timestamp"`
}

// Recommendation represents a knowledge recommendation
type Recommendation struct {
	Type      string    `json:"type"`
	NodeID    string    `json:"node_id"`
	Score     float64   `json:"score"`
	Reason    string    `json:"reason"`
	Context   []string  `json:"context"`
	CreatedAt time.Time `json:"created_at"`
}
