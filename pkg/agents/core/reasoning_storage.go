// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package core

import (
	"context"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// ReasoningChain represents a complete reasoning chain from an agent
type ReasoningChain struct {
	ID         string                 `json:"id" db:"id"`
	AgentID    string                 `json:"agent_id" db:"agent_id"`
	SessionID  string                 `json:"session_id" db:"session_id"`
	RequestID  string                 `json:"request_id" db:"request_id"`
	Content    string                 `json:"content" db:"content"`
	Reasoning  string                 `json:"reasoning" db:"reasoning"`
	Confidence float64                `json:"confidence" db:"confidence"`
	TaskType   string                 `json:"task_type" db:"task_type"`
	Success    bool                   `json:"success" db:"success"`
	TokensUsed int                    `json:"tokens_used" db:"tokens_used"`
	Duration   time.Duration          `json:"duration" db:"duration"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	Metadata   map[string]interface{} `json:"metadata" db:"metadata"`
}

// ReasoningQuery defines search parameters for reasoning chains
type ReasoningQuery struct {
	AgentID       string
	SessionID     string
	TaskType      string
	MinConfidence float64
	MaxConfidence float64
	StartTime     time.Time
	EndTime       time.Time
	Limit         int
	Offset        int
	OrderBy       string // "created_at", "confidence", "duration"
	Ascending     bool
}

// ReasoningStats represents aggregated statistics
type ReasoningStats struct {
	TotalChains       int64                  `json:"total_chains"`
	AvgConfidence     float64                `json:"avg_confidence"`
	AvgDuration       time.Duration          `json:"avg_duration"`
	SuccessRate       float64                `json:"success_rate"`
	ConfidenceDistrib map[string]int         `json:"confidence_distribution"`
	TaskTypeDistrib   map[string]int         `json:"task_type_distribution"`
	TimeDistrib       map[string]int         `json:"time_distribution"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// ReasoningPattern represents a learned reasoning pattern
type ReasoningPattern struct {
	ID          string                 `json:"id" db:"id"`
	Pattern     string                 `json:"pattern" db:"pattern"`
	TaskType    string                 `json:"task_type" db:"task_type"`
	Occurrences int                    `json:"occurrences" db:"occurrences"`
	AvgSuccess  float64                `json:"avg_success" db:"avg_success"`
	Examples    []string               `json:"examples" db:"examples"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
}

// ReasoningStorage defines the interface for persisting reasoning chains
type ReasoningStorage interface {
	// Store saves a reasoning chain
	Store(ctx context.Context, chain *ReasoningChain) error

	// Get retrieves a reasoning chain by ID
	Get(ctx context.Context, id string) (*ReasoningChain, error)

	// Query searches for reasoning chains based on criteria
	Query(ctx context.Context, query *ReasoningQuery) ([]*ReasoningChain, error)

	// GetStats returns aggregated statistics
	GetStats(ctx context.Context, agentID string, startTime, endTime time.Time) (*ReasoningStats, error)

	// GetPatterns returns learned reasoning patterns
	GetPatterns(ctx context.Context, taskType string, limit int) ([]*ReasoningPattern, error)

	// UpdatePattern updates or creates a reasoning pattern
	UpdatePattern(ctx context.Context, pattern *ReasoningPattern) error

	// Delete removes old reasoning chains
	Delete(ctx context.Context, beforeTime time.Time) (int64, error)

	// Close closes the storage connection
	Close(ctx context.Context) error
}

// ReasoningAnalyzer provides analytics on reasoning chains
type ReasoningAnalyzer interface {
	// AnalyzeConfidenceCorrelation analyzes correlation between confidence and success
	AnalyzeConfidenceCorrelation(ctx context.Context, chains []*ReasoningChain) (float64, error)

	// IdentifyPatterns identifies common reasoning patterns
	IdentifyPatterns(ctx context.Context, chains []*ReasoningChain) ([]*ReasoningPattern, error)

	// CompareAgents compares reasoning performance between agents
	CompareAgents(ctx context.Context, agentIDs []string, startTime, endTime time.Time) (map[string]*ReasoningStats, error)

	// GenerateInsights generates actionable insights from reasoning data
	GenerateInsights(ctx context.Context, stats *ReasoningStats) ([]string, error)
}

// ReasoningStorageConfig configures the reasoning storage
type ReasoningStorageConfig struct {
	RetentionDays     int    `yaml:"retention_days" default:"30"`
	MaxChainsPerQuery int    `yaml:"max_chains_per_query" default:"1000"`
	EnableCompression bool   `yaml:"enable_compression" default:"true"`
	StorageBackend    string `yaml:"storage_backend" default:"sqlite"` // sqlite, postgres, memory
}

// Validate validates the storage configuration
func (c ReasoningStorageConfig) Validate() error {
	if c.RetentionDays < 0 {
		return gerror.New(gerror.ErrCodeValidation, "retention days cannot be negative", nil).
			WithComponent("reasoning_storage").
			WithOperation("Validate")
	}

	if c.MaxChainsPerQuery < 1 {
		return gerror.New(gerror.ErrCodeValidation, "max chains per query must be at least 1", nil).
			WithComponent("reasoning_storage").
			WithOperation("Validate")
	}

	validBackends := map[string]bool{
		"sqlite":   true,
		"postgres": true,
		"memory":   true,
	}

	if !validBackends[c.StorageBackend] {
		return gerror.Newf(gerror.ErrCodeValidation, "invalid storage backend: %s", c.StorageBackend).
			WithComponent("reasoning_storage").
			WithOperation("Validate")
	}

	return nil
}

// ReasoningEvent represents an event in the reasoning system
type ReasoningEvent struct {
	Type      string                 `json:"type"`
	AgentID   string                 `json:"agent_id"`
	ChainID   string                 `json:"chain_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// ReasoningEventType defines types of reasoning events
type ReasoningEventType string

const (
	EventReasoningStarted   ReasoningEventType = "reasoning_started"
	EventReasoningCompleted ReasoningEventType = "reasoning_completed"
	EventReasoningFailed    ReasoningEventType = "reasoning_failed"
	EventPatternIdentified  ReasoningEventType = "pattern_identified"
	EventInsightGenerated   ReasoningEventType = "insight_generated"
)

// ReasoningObserver defines the interface for observing reasoning events
type ReasoningObserver interface {
	// OnReasoningEvent is called when a reasoning event occurs
	OnReasoningEvent(ctx context.Context, event *ReasoningEvent) error
}

// ReasoningCache provides fast access to recent reasoning chains
type ReasoningCache interface {
	// Put adds a reasoning chain to the cache
	Put(ctx context.Context, chain *ReasoningChain) error

	// Get retrieves a reasoning chain from cache
	Get(ctx context.Context, id string) (*ReasoningChain, error)

	// GetRecent retrieves recent chains for an agent
	GetRecent(ctx context.Context, agentID string, limit int) ([]*ReasoningChain, error)

	// Invalidate removes entries from cache
	Invalidate(ctx context.Context, agentID string) error

	// Stats returns cache statistics
	Stats(ctx context.Context) (map[string]interface{}, error)
}
