// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"time"

	"github.com/guild-framework/guild-core/pkg/events"
)

// Event types for reasoning system
const (
	EventReasoningExtracted        = "reasoning.extracted"
	EventReasoningFailed           = "reasoning.failed"
	EventCircuitBreakerOpen        = "reasoning.circuit_breaker.open"
	EventCircuitBreakerClosed      = "reasoning.circuit_breaker.closed"
	EventCircuitBreakerHalfOpen    = "reasoning.circuit_breaker.half_open"
	EventCircuitBreakerStateChange = "reasoning.circuit_breaker.state_change"
	EventRateLimitExceeded         = "reasoning.rate_limit.exceeded"
	EventReasoningStreamStart      = "reasoning.stream.start"
	EventReasoningStreamEnd        = "reasoning.stream.end"
	EventReasoningBlockProcessed   = "reasoning.block.processed"
)

// ReasoningExtractedEvent is emitted when reasoning blocks are successfully extracted
type ReasoningExtractedEvent struct {
	events.BaseEvent
	AgentID         string           `json:"agent_id"`
	Provider        string           `json:"provider"`
	Blocks          []ReasoningBlock `json:"blocks"`
	TokensExtracted int              `json:"tokens_extracted"`
	Duration        time.Duration    `json:"duration"`
	Timestamp       time.Time        `json:"timestamp"`
}

// ReasoningFailedEvent is emitted when reasoning extraction fails
type ReasoningFailedEvent struct {
	events.BaseEvent
	AgentID   string    `json:"agent_id"`
	Provider  string    `json:"provider"`
	Error     error     `json:"error"`
	ErrorCode string    `json:"error_code"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// CircuitBreakerStateChangeEvent is emitted when circuit breaker changes state
type CircuitBreakerStateChangeEvent struct {
	events.BaseEvent
	Provider  string       `json:"provider"`
	From      CircuitState `json:"from_state"`
	To        CircuitState `json:"to_state"`
	Reason    string       `json:"reason"`
	Timestamp time.Time    `json:"timestamp"`
}

// RateLimitExceededEvent is emitted when rate limit is hit
type RateLimitExceededEvent struct {
	events.BaseEvent
	AgentID    string    `json:"agent_id"`
	LimitType  string    `json:"limit_type"` // "global" or "agent"
	Limit      int       `json:"limit"`
	Current    int       `json:"current"`
	RetryAfter time.Time `json:"retry_after"`
	Timestamp  time.Time `json:"timestamp"`
}

// ReasoningStreamStartEvent marks the beginning of streaming extraction
type ReasoningStreamStartEvent struct {
	events.BaseEvent
	AgentID   string    `json:"agent_id"`
	Provider  string    `json:"provider"`
	StreamID  string    `json:"stream_id"`
	Timestamp time.Time `json:"timestamp"`
}

// ReasoningStreamEndEvent marks the end of streaming extraction
type ReasoningStreamEndEvent struct {
	events.BaseEvent
	AgentID     string        `json:"agent_id"`
	Provider    string        `json:"provider"`
	StreamID    string        `json:"stream_id"`
	BlocksCount int           `json:"blocks_count"`
	TotalTokens int           `json:"total_tokens"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	Error       error         `json:"error,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
}

// ReasoningBlockProcessedEvent is emitted for each processed reasoning block
type ReasoningBlockProcessedEvent struct {
	events.BaseEvent
	AgentID      string         `json:"agent_id"`
	Provider     string         `json:"provider"`
	StreamID     string         `json:"stream_id"`
	Block        ReasoningBlock `json:"block"`
	ProcessingMS int64          `json:"processing_ms"`
	Timestamp    time.Time      `json:"timestamp"`
}

// EventHandler defines the interface for handling reasoning events
type EventHandler interface {
	HandleReasoningExtracted(event *ReasoningExtractedEvent) error
	HandleReasoningFailed(event *ReasoningFailedEvent) error
	HandleCircuitBreakerStateChange(event *CircuitBreakerStateChangeEvent) error
	HandleRateLimitExceeded(event *RateLimitExceededEvent) error
}

// EventSubscriber helps subscribe to reasoning events
type EventSubscriber struct {
	eventBus events.EventBus
	handlers map[string]events.SubscriptionID
}

// NewEventSubscriber creates a new event subscriber
func NewEventSubscriber(eventBus events.EventBus) *EventSubscriber {
	return &EventSubscriber{
		eventBus: eventBus,
		handlers: make(map[string]events.SubscriptionID),
	}
}

// Subscribe sets up subscriptions for all reasoning events
func (s *EventSubscriber) Subscribe(ctx context.Context, handler EventHandler) error {
	// Subscribe to reasoning extracted events
	subID, err := s.eventBus.Subscribe(ctx, EventReasoningExtracted, func(ctx context.Context, event events.CoreEvent) error {
		if e, ok := event.(*ReasoningExtractedEvent); ok {
			return handler.HandleReasoningExtracted(e)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.handlers[EventReasoningExtracted] = subID

	// Subscribe to reasoning failed events
	subID, err = s.eventBus.Subscribe(ctx, EventReasoningFailed, func(ctx context.Context, event events.CoreEvent) error {
		if e, ok := event.(*ReasoningFailedEvent); ok {
			return handler.HandleReasoningFailed(e)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.handlers[EventReasoningFailed] = subID

	// Subscribe to circuit breaker state changes
	subID, err = s.eventBus.Subscribe(ctx, EventCircuitBreakerStateChange, func(ctx context.Context, event events.CoreEvent) error {
		if e, ok := event.(*CircuitBreakerStateChangeEvent); ok {
			return handler.HandleCircuitBreakerStateChange(e)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.handlers[EventCircuitBreakerStateChange] = subID

	// Subscribe to rate limit exceeded events
	subID, err = s.eventBus.Subscribe(ctx, EventRateLimitExceeded, func(ctx context.Context, event events.CoreEvent) error {
		if e, ok := event.(*RateLimitExceededEvent); ok {
			return handler.HandleRateLimitExceeded(e)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.handlers[EventRateLimitExceeded] = subID

	return nil
}

// Unsubscribe removes all subscriptions
func (s *EventSubscriber) Unsubscribe(ctx context.Context) error {
	for _, subID := range s.handlers {
		if err := s.eventBus.Unsubscribe(ctx, subID); err != nil {
			return err
		}
	}
	s.handlers = make(map[string]events.SubscriptionID)
	return nil
}
