// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package retrieval

import (
	"context"
	"time"
)

// Document represents a retrievable document with metadata
type Document struct {
	ID       string                 `json:"id"`
	Content  string                 `json:"content"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// RankedDocument represents a document with final ranking scores
type RankedDocument struct {
	Document
	FinalScore   float64            `json:"final_score"`
	ScoreDetails map[string]float64 `json:"score_details"`
}

// Query represents a retrieval query with context
type Query struct {
	Text       string       `json:"text"`
	Context    QueryContext `json:"context"`
	Filters    []Filter     `json:"filters"`
	MaxResults int          `json:"max_results"`
	MinScore   float64      `json:"min_score"`
}

// QueryContext provides contextual information for enhanced retrieval
type QueryContext struct {
	TaskID         string    `json:"task_id"`
	AgentID        string    `json:"agent_id"`
	MessageHistory []Message `json:"message_history"`
	CurrentFiles   []string  `json:"current_files"`
	Tags           []string  `json:"tags"`
}

// Message represents a chat message in the context
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Filter represents a filter for document retrieval
type Filter struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
	Op    FilterOp    `json:"op"`
}

// FilterOp represents filter operations
type FilterOp string

const (
	FilterOpEquals      FilterOp = "equals"
	FilterOpContains    FilterOp = "contains"
	FilterOpGreaterThan FilterOp = "gt"
	FilterOpLessThan    FilterOp = "lt"
)

// RetrievalStrategy defines an interface for different retrieval approaches
type RetrievalStrategy interface {
	Name() string
	Retrieve(ctx context.Context, query Query) ([]Document, error)
	Weight() float64
}

// VectorStore interface for vector-based retrieval
type VectorStore interface {
	Search(ctx context.Context, query string, limit int) ([]VectorResult, error)
}

// VectorResult represents a vector search result
type VectorResult struct {
	DocID    string                 `json:"doc_id"`
	Content  string                 `json:"content"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

// KeywordIndexer interface for keyword-based search
type KeywordIndexer interface {
	Search(ctx context.Context, keywords []string, limit int) ([]Document, error)
	Index(ctx context.Context, doc Document) error
}

// KnowledgeGraph interface for graph-based traversal
type KnowledgeGraph interface {
	FindEntryNodes(ctx context.Context, query Query) ([]GraphNode, error)
	TraverseRelated(ctx context.Context, nodes []GraphNode, hops int) ([]GraphNode, error)
}

// GraphNode represents a node in the knowledge graph
type GraphNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// Retriever interface for document retrieval
type Retriever interface {
	Retrieve(ctx context.Context, query Query) ([]RankedDocument, error)
	AddStrategy(strategy RetrievalStrategy) error
	SetEventBus(eventBus EventBus)
	GetStrategies() []RetrievalStrategy
	Close() error
}

// EventBus interface for system integration
type EventBus interface {
	Publish(ctx context.Context, event Event) error
}

// Event represents a system event
type Event struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Embedder interface for text embeddings
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// MetadataStore interface for document metadata
type MetadataStore interface {
	GetDocument(ctx context.Context, id string) (*Document, error)
	ListDocuments(ctx context.Context, filters []Filter) ([]Document, error)
	UpdateMetadata(ctx context.Context, id string, metadata map[string]interface{}) error
}
