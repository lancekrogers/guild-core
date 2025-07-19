// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// Common errors
var (
	ErrCircuitBreakerOpen = gerror.New("circuit breaker is open").
				WithCode(gerror.ErrCodeResourceExhausted).
				WithComponent("reasoning")

	ErrRateLimitExceeded = gerror.New("rate limit exceeded").
				WithCode(gerror.ErrCodeResourceExhausted).
				WithComponent("reasoning")

	ErrRegistryNotStarted = gerror.New("reasoning registry not started").
				WithCode(gerror.ErrCodeFailedPrecondition).
				WithComponent("reasoning")
)

// ReasoningBlock represents a unit of reasoning extracted from LLM responses
// This is the local type that mirrors interfaces.ReasoningBlock
type ReasoningBlock struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // e.g., "thinking", "planning", "analysis"
	Content    string                 `json:"content"`
	Timestamp  time.Time              `json:"timestamp"`
	Duration   time.Duration          `json:"duration"`
	TokenCount int                    `json:"token_count"`
	Depth      int                    `json:"depth"` // Nesting level
	ParentID   string                 `json:"parent_id,omitempty"`
	Children   []string               `json:"children,omitempty"`
	Confidence float64                `json:"confidence,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
