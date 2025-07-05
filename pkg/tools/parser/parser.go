// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package parser

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// responseParser implements ResponseParser with full context support and proper error handling
type responseParser struct {
	universal UniversalParser
	mu        sync.RWMutex
}

// NewResponseParser creates a new response parser with all provider support
func NewResponseParser() ResponseParser {
	return &responseParser{
		universal: NewUniversalParser(),
	}
}

// ExtractToolCalls extracts tool calls from a response with auto-detection
func (p *responseParser) ExtractToolCalls(response string) ([]ToolCall, error) {
	if response == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "empty response", nil).
			WithComponent("ResponseParser").
			WithOperation("ExtractToolCalls")
	}

	p.mu.RLock()
	universal := p.universal
	p.mu.RUnlock()

	calls, err := universal.ParseResponse(response)
	if err != nil {
		// Check if the error is because no tool calls were found
		if gerror.Is(err, ErrNoToolCalls) {
			return nil, nil // No tool calls is not an error
		}
		
		// Check if it's a wrapped "not found" error
		var gErr *gerror.GuildError
		if errors.As(err, &gErr) && gErr.Code == gerror.ErrCodeNotFound {
			return nil, nil
		}
		
		// Check the cause chain for ErrNoToolCalls
		cause := err
		for cause != nil {
			if gerror.Is(cause, ErrNoToolCalls) {
				return nil, nil
			}
			if gErr, ok := cause.(*gerror.GuildError); ok {
				if gErr.Code == gerror.ErrCodeNotFound {
					return nil, nil
				}
				cause = gErr.Cause
			} else {
				cause = errors.Unwrap(cause)
			}
		}
		
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse response").
			WithComponent("ResponseParser").
			WithOperation("ExtractToolCalls").
			WithDetails("responseLength", len(response))
	}

	return calls, nil
}

// ContainsToolCalls checks if response has tool calls
func (p *responseParser) ContainsToolCalls(response string) bool {
	if response == "" {
		return false
	}
	
	p.mu.RLock()
	universal := p.universal
	p.mu.RUnlock()
	
	return universal.HasToolCalls(response)
}

// ExtractToolCallsWithContext provides context-aware extraction with cancellation support
func (p *responseParser) ExtractToolCallsWithContext(ctx context.Context, response string) ([]ToolCall, error) {
	// Check context at the start
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCanceled, "context canceled before extraction").
			WithComponent("ResponseParser").
			WithOperation("ExtractToolCallsWithContext")
	}

	// Create result channel
	type result struct {
		calls []ToolCall
		err   error
	}
	resultCh := make(chan result, 1)

	// Run extraction in goroutine
	go func() {
		start := time.Now()
		calls, err := p.ExtractToolCalls(response)
		duration := time.Since(start)
		
		// Add timing info to error if present
		if err != nil {
			if ge, ok := err.(*gerror.GuildError); ok {
				ge.WithDetails("extractionDuration", duration.String())
			}
		}
		
		resultCh <- result{calls: calls, err: err}
	}()

	// Wait for result or context cancellation
	select {
	case <-ctx.Done():
		return nil, gerror.Wrap(ctx.Err(), gerror.ErrCodeCanceled, "extraction canceled").
			WithComponent("ResponseParser").
			WithOperation("ExtractToolCallsWithContext")
	case res := <-resultCh:
		return res.calls, res.err
	}
}