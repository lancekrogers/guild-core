// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lancekrogers/guild-core/pkg/mcp/protocol"
)

// loggingMiddleware logs requests and responses
func loggingMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
		start := time.Now()

		log.Printf("MCP Request: ID=%s Method=%s", msg.ID, msg.Method)

		response, err := next(ctx, msg)

		duration := time.Since(start)
		if err != nil {
			log.Printf("MCP Error: ID=%s Method=%s Duration=%v Error=%v",
				msg.ID, msg.Method, duration, err)
		} else {
			log.Printf("MCP Response: ID=%s Method=%s Duration=%v",
				msg.ID, msg.Method, duration)
		}

		return response, err
	}
}

// recoveryMiddleware recovers from panics
func recoveryMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, msg *protocol.MCPMessage) (response *protocol.MCPMessage, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("MCP Panic recovered: ID=%s Method=%s Panic=%v",
					msg.ID, msg.Method, r)

				errData, _ := json.Marshal(fmt.Sprintf("panic: %v", r))
				err = &protocol.Error{
					Code:    protocol.ErrorCodeInternal,
					Message: "Internal server error",
					Data:    json.RawMessage(errData),
				}
				response = nil
			}
		}()

		return next(ctx, msg)
	}
}

// timeoutMiddleware adds request timeout
func timeoutMiddleware(timeout time.Duration) func(HandlerFunc) HandlerFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Use a channel to handle timeout vs completion
			type result struct {
				response *protocol.MCPMessage
				err      error
			}

			resultCh := make(chan result, 1)

			go func() {
				response, err := next(ctx, msg)
				resultCh <- result{response, err}
			}()

			select {
			case <-ctx.Done():
				errData, _ := json.Marshal(fmt.Sprintf("timeout after %v", timeout))
				return nil, &protocol.Error{
					Code:    protocol.ErrorCodeTimeout,
					Message: "Request timeout",
					Data:    json.RawMessage(errData),
				}
			case res := <-resultCh:
				return res.response, res.err
			}
		}
	}
}

// authMiddleware provides JWT authentication
func authMiddleware(jwtSecret string) func(HandlerFunc) HandlerFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
			// Skip auth for system methods
			if msg.Method == "system.ping" || msg.Method == "system.info" {
				return next(ctx, msg)
			}

			// Extract token from metadata
			var token string
			if auth, ok := msg.Metadata.CustomFields["authorization"]; ok {
				token = auth
			}

			if token == "" {
				return nil, &protocol.Error{
					Code:    protocol.ErrorCodeAuthFailed,
					Message: "Authentication required",
				}
			}

			// Validate JWT token (simplified - in production use proper JWT library)
			if !isValidToken(token, jwtSecret) {
				return nil, &protocol.Error{
					Code:    protocol.ErrorCodeAuthFailed,
					Message: "Invalid token",
				}
			}

			// Add user info to context
			ctx = context.WithValue(ctx, "user_id", extractUserID(token))

			return next(ctx, msg)
		}
	}
}

// rateLimitMiddleware limits requests per client
func rateLimitMiddleware(limit int, window time.Duration) func(HandlerFunc) HandlerFunc {
	// Simple in-memory rate limiter (in production use Redis or similar)
	clients := make(map[string][]time.Time)

	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
			// Extract client ID
			clientID := "unknown"
			if userID := ctx.Value("user_id"); userID != nil {
				clientID = fmt.Sprintf("%v", userID)
			}

			now := time.Now()

			// Get client's request history
			requests := clients[clientID]

			// Remove old requests outside the window
			validRequests := requests[:0]
			for _, reqTime := range requests {
				if now.Sub(reqTime) < window {
					validRequests = append(validRequests, reqTime)
				}
			}

			// Check rate limit
			if len(validRequests) >= limit {
				dataMap := map[string]interface{}{
					"limit":  limit,
					"window": window.String(),
				}
				dataBytes, _ := json.Marshal(dataMap)
				return nil, &protocol.Error{
					Code:    protocol.TooManyRequests,
					Message: "Rate limit exceeded",
					Data:    json.RawMessage(dataBytes),
				}
			}

			// Add current request
			validRequests = append(validRequests, now)
			clients[clientID] = validRequests

			return next(ctx, msg)
		}
	}
}

// metricsMiddleware collects metrics
func metricsMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
		start := time.Now()

		response, err := next(ctx, msg)

		duration := time.Since(start)

		// Record metrics (simplified - in production use proper metrics library)
		recordMetric("mcp_request_duration", duration.Seconds(), map[string]string{
			"method": msg.Method,
			"status": getStatus(err),
		})

		recordMetric("mcp_request_count", 1, map[string]string{
			"method": msg.Method,
			"status": getStatus(err),
		})

		return response, err
	}
}

// tracingMiddleware adds distributed tracing
func tracingMiddleware(next HandlerFunc) HandlerFunc {
	return func(ctx context.Context, msg *protocol.MCPMessage) (*protocol.MCPMessage, error) {
		// Extract trace ID from metadata
		traceID := msg.ID // Use message ID as trace ID for simplicity
		if msg.Metadata.TraceID != "" {
			traceID = msg.Metadata.TraceID
		} else if tid, ok := msg.Metadata.CustomFields["trace-id"]; ok {
			traceID = tid
		}

		// Add trace context
		ctx = context.WithValue(ctx, "trace_id", traceID)
		ctx = context.WithValue(ctx, "span_id", generateSpanID())

		// Start span
		span := startSpan(ctx, msg.Method)
		defer span.finish()

		response, err := next(ctx, msg)
		// Record span result
		if err != nil {
			span.setError(err)
		}

		return response, err
	}
}

// Helper functions

func isValidToken(token, secret string) bool {
	// Simplified token validation
	// In production, use proper JWT validation
	return token != "" && secret != ""
}

func extractUserID(token string) string {
	// Simplified user ID extraction
	// In production, decode JWT and extract user ID
	return "user123"
}

func getStatus(err error) string {
	if err == nil {
		return "success"
	}
	if mcpErr, ok := err.(*protocol.Error); ok {
		return fmt.Sprintf("error_%d", mcpErr.Code)
	}
	return "error"
}

func recordMetric(name string, value float64, tags map[string]string) {
	// Simplified metrics recording
	// In production, use proper metrics library like Prometheus
	log.Printf("Metric: %s=%f tags=%v", name, value, tags)
}

func generateSpanID() string {
	// Generate unique span ID
	return fmt.Sprintf("span-%d", time.Now().UnixNano())
}

type span struct {
	traceID string
	spanID  string
	name    string
	start   time.Time
	err     error
}

func startSpan(ctx context.Context, name string) *span {
	return &span{
		traceID: fmt.Sprintf("%v", ctx.Value("trace_id")),
		spanID:  fmt.Sprintf("%v", ctx.Value("span_id")),
		name:    name,
		start:   time.Now(),
	}
}

func (s *span) setError(err error) {
	s.err = err
}

func (s *span) finish() {
	duration := time.Since(s.start)
	status := "ok"
	if s.err != nil {
		status = "error"
	}

	log.Printf("Span: trace=%s span=%s name=%s duration=%v status=%s",
		s.traceID, s.spanID, s.name, duration, status)
}
