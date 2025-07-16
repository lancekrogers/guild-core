// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package interfaces

import (
	"context"
	"time"
)

// ReasoningBlock represents a unit of reasoning extracted from LLM responses
type ReasoningBlock struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // e.g., "thinking", "planning", "analysis"
	Content     string                 `json:"content"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
	TokenCount  int                    `json:"token_count"`
	Depth       int                    `json:"depth"`       // Nesting level
	ParentID    string                 `json:"parent_id,omitempty"`
	Children    []string               `json:"children,omitempty"`
	Confidence  float64                `json:"confidence,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ReasoningProvider extends AIProvider with reasoning extraction capabilities
type ReasoningProvider interface {
	AIProvider
	// ExtractReasoning extracts reasoning blocks from a response
	ExtractReasoning(ctx context.Context, response *ChatResponse) ([]*ReasoningBlock, error)
	// SupportsReasoningExtraction indicates if the provider supports reasoning extraction
	SupportsReasoningExtraction() bool
}

// StreamingReasoningProvider supports reasoning extraction from streaming responses
type StreamingReasoningProvider interface {
	ReasoningProvider
	// StreamChatCompletionWithReasoning streams chat completion with reasoning extraction
	StreamChatCompletionWithReasoning(ctx context.Context, req ChatRequest) (ReasoningStream, error)
}

// ReasoningStream represents a streaming response with reasoning extraction
type ReasoningStream interface {
	ChatStream
	// ReasoningChannel returns a channel for reasoning blocks
	ReasoningChannel() <-chan *ReasoningBlock
	// ErrorChannel returns a channel for errors
	ErrorChannel() <-chan error
}

// ReasoningCapabilities extends ProviderCapabilities with reasoning support
type ReasoningCapabilities struct {
	ProviderCapabilities
	SupportsReasoningExtraction bool
	ReasoningFormats            []ReasoningFormat
}

// ReasoningFormat describes a reasoning format supported by a provider
type ReasoningFormat struct {
	Name        string // e.g., "thinking_tags", "chain_of_thought", "o1_reasoning"
	Description string
	Examples    []string
}

// ReasoningMetrics tracks reasoning extraction performance
type ReasoningMetrics struct {
	BlocksExtracted   int
	TokensInReasoning int
	ExtractionTimeMs  int64
	SuccessRate       float64
}
