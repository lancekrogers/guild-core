package memory

import (
	"context"
	"time"
)

// Store represents a persistent storage system for agent data.
//
// The Guild Framework uses BoltDB as its implementation, which provides:
//   - MVCC (Multi-Version Concurrency Control): Readers don't block writers
//   - Unlimited concurrent readers for excellent read performance
//   - Sequential writes (single writer) with high throughput (44k writes/sec)
//   - ACID transactions ensuring data consistency
//
// This design is optimal for Guild's agent coordination patterns:
//   - Read-heavy workload (checking status, retrieving context)
//   - Small write transactions (task updates, message appends)
//   - Event-driven coordination reduces database contention
//   - Proven to handle 1000+ concurrent connections in production
//
// Note: Do NOT replace BoltDB with client-server databases like PostgreSQL.
// The embedded nature and performance characteristics are essential to Guild's
// single-machine, developer-focused architecture.
type Store interface {
	// Put stores a value with the given key
	Put(ctx context.Context, bucket, key string, value []byte) error

	// Get retrieves a value by key
	Get(ctx context.Context, bucket, key string) ([]byte, error)

	// Delete removes a value by key
	Delete(ctx context.Context, bucket, key string) error

	// List returns all keys in a bucket
	List(ctx context.Context, bucket string) ([]string, error)

	// ListKeys returns keys with the given prefix in a bucket
	ListKeys(ctx context.Context, bucket, prefix string) ([]string, error)

	// Close closes the store
	Close() error
}

// PromptChain represents a sequence of prompts and responses
type PromptChain struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	TaskID    string    `json:"task_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
}

// Message represents a single message in a prompt chain
type Message struct {
	Role      string    `json:"role"`       // "system", "user", "assistant", or "tool"
	Content   string    `json:"content"`    // The message content
	Name      string    `json:"name,omitempty"` // Name of the tool for tool messages
	Timestamp time.Time `json:"timestamp"`  // When the message was added
	TokenUsage int      `json:"token_usage,omitempty"` // Tokens used for this message
}

// ChainManager manages prompt chains
type ChainManager interface {
	// CreateChain creates a new prompt chain
	CreateChain(ctx context.Context, agentID, taskID string) (string, error)

	// GetChain retrieves a chain by ID
	GetChain(ctx context.Context, chainID string) (*PromptChain, error)

	// AddMessage adds a message to a chain
	AddMessage(ctx context.Context, chainID string, message Message) error

	// GetChainsByAgent retrieves all chains for an agent
	GetChainsByAgent(ctx context.Context, agentID string) ([]*PromptChain, error)

	// GetChainsByTask retrieves all chains for a task
	GetChainsByTask(ctx context.Context, taskID string) ([]*PromptChain, error)

	// BuildContext builds a context from chains for an agent and task
	BuildContext(ctx context.Context, agentID, taskID string, maxTokens int) ([]Message, error)

	// DeleteChain deletes a chain
	DeleteChain(ctx context.Context, chainID string) error
}

// Document represents a document in the corpus
type Document struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// CorpusManager manages the research corpus
type CorpusManager interface {
	// AddDocument adds a document to the corpus
	AddDocument(ctx context.Context, doc *Document) (string, error)

	// GetDocument retrieves a document by ID
	GetDocument(ctx context.Context, docID string) (*Document, error)

	// UpdateDocument updates a document
	UpdateDocument(ctx context.Context, doc *Document) error

	// DeleteDocument deletes a document
	DeleteDocument(ctx context.Context, docID string) error

	// SearchDocuments searches documents by text
	SearchDocuments(ctx context.Context, query string, limit int) ([]*Document, error)

	// ListDocuments lists all documents with optional filters
	ListDocuments(ctx context.Context, filters map[string]string, limit, offset int) ([]*Document, error)
}

// ErrNotFound is returned when a requested item is not found
var ErrNotFound = StoreError{Message: "item not found"}

// StoreError represents an error from the storage system
type StoreError struct {
	Message string
}

// Error implements the error interface
func (e StoreError) Error() string {
	return e.Message
}
