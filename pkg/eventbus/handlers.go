// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package eventbus

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// HandlerRegistry manages event handlers with ordering and dependencies
type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string][]*RegisteredHandler
}

// RegisteredHandler wraps a handler with metadata
type RegisteredHandler struct {
	ID           string
	Handler      EventHandler
	Priority     int
	Dependencies []string
	Timeout      time.Duration
	MaxRetries   int
	Metadata     map[string]interface{}
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string][]*RegisteredHandler),
	}
}

// Register registers a handler for an event type
func (hr *HandlerRegistry) Register(eventType string, handler *RegisteredHandler) error {
	if handler == nil || handler.Handler == nil {
		return gerror.New(gerror.ErrCodeValidation, "handler is required", nil)
	}

	if handler.ID == "" {
		handler.ID = generateHandlerID()
	}

	if handler.Timeout == 0 {
		handler.Timeout = 30 * time.Second
	}

	hr.mu.Lock()
	defer hr.mu.Unlock()

	// Check for circular dependencies
	if err := hr.checkDependencies(handler.ID, handler.Dependencies, make(map[string]bool)); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCircularDependency, "dependency check failed")
	}

	// Add handler to registry
	handlers := hr.handlers[eventType]
	handlers = append(handlers, handler)

	// Sort by priority
	hr.sortHandlers(handlers)
	hr.handlers[eventType] = handlers

	return nil
}

// Unregister removes a handler
func (hr *HandlerRegistry) Unregister(eventType, handlerID string) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()

	handlers, exists := hr.handlers[eventType]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "no handlers for event type", nil).
			WithDetails("event_type", eventType)
	}

	for i, h := range handlers {
		if h.ID == handlerID {
			hr.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			return nil
		}
	}

	return gerror.New(gerror.ErrCodeNotFound, "handler not found", nil).
		WithDetails("handler_id", handlerID)
}

// GetHandlers returns handlers for an event type
func (hr *HandlerRegistry) GetHandlers(eventType string) []*RegisteredHandler {
	hr.mu.RLock()
	defer hr.mu.RUnlock()

	handlers := hr.handlers[eventType]
	if handlers == nil {
		return nil
	}

	// Return a copy to prevent modification
	result := make([]*RegisteredHandler, len(handlers))
	copy(result, handlers)
	return result
}

// ExecuteHandlers executes handlers for an event
func (hr *HandlerRegistry) ExecuteHandlers(ctx context.Context, event Event) error {
	handlers := hr.GetHandlers(event.GetType())
	if len(handlers) == 0 {
		return nil
	}

	// Group handlers by dependency level
	levels := hr.groupByDependencyLevel(handlers)

	// Execute each level sequentially
	for _, level := range levels {
		if err := hr.executeLevel(ctx, event, level); err != nil {
			return err
		}
	}

	return nil
}

// checkDependencies checks for circular dependencies
func (hr *HandlerRegistry) checkDependencies(handlerID string, deps []string, visited map[string]bool) error {
	if visited[handlerID] {
		return gerror.New(gerror.ErrCodeCircularDependency, "circular dependency detected", nil).
			WithDetails("handler_id", handlerID)
	}

	visited[handlerID] = true

	for _, dep := range deps {
		// Find dependent handler
		found := false
		for _, handlers := range hr.handlers {
			for _, h := range handlers {
				if h.ID == dep {
					found = true
					if err := hr.checkDependencies(dep, h.Dependencies, visited); err != nil {
						return err
					}
					break
				}
			}
			if found {
				break
			}
		}
	}

	delete(visited, handlerID)
	return nil
}

// sortHandlers sorts handlers by priority (higher priority first)
func (hr *HandlerRegistry) sortHandlers(handlers []*RegisteredHandler) {
	for i := 0; i < len(handlers); i++ {
		for j := i + 1; j < len(handlers); j++ {
			if handlers[j].Priority > handlers[i].Priority {
				handlers[i], handlers[j] = handlers[j], handlers[i]
			}
		}
	}
}

// groupByDependencyLevel groups handlers by their dependency level
func (hr *HandlerRegistry) groupByDependencyLevel(handlers []*RegisteredHandler) [][]*RegisteredHandler {
	// Build dependency graph
	depGraph := make(map[string][]string)
	handlerMap := make(map[string]*RegisteredHandler)

	for _, h := range handlers {
		handlerMap[h.ID] = h
		for _, dep := range h.Dependencies {
			depGraph[dep] = append(depGraph[dep], h.ID)
		}
	}

	// Topological sort
	var levels [][]*RegisteredHandler
	processed := make(map[string]bool)

	for {
		level := make([]*RegisteredHandler, 0)

		for _, h := range handlers {
			if processed[h.ID] {
				continue
			}

			// Check if all dependencies are processed
			canProcess := true
			for _, dep := range h.Dependencies {
				if !processed[dep] {
					canProcess = false
					break
				}
			}

			if canProcess {
				level = append(level, h)
			}
		}

		if len(level) == 0 {
			break
		}

		// Mark as processed
		for _, h := range level {
			processed[h.ID] = true
		}

		levels = append(levels, level)
	}

	return levels
}

// executeLevel executes handlers at the same dependency level
func (hr *HandlerRegistry) executeLevel(ctx context.Context, event Event, handlers []*RegisteredHandler) error {
	if len(handlers) == 0 {
		return nil
	}

	// Execute handlers concurrently within the same level
	var wg sync.WaitGroup
	errCh := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		go func(h *RegisteredHandler) {
			defer wg.Done()

			// Create timeout context
			handlerCtx, cancel := context.WithTimeout(ctx, h.Timeout)
			defer cancel()

			// Execute with retries
			var err error
			for attempt := 0; attempt <= h.MaxRetries; attempt++ {
				err = h.Handler(handlerCtx, event)
				if err == nil {
					return
				}

				// Don't retry on context cancellation
				if handlerCtx.Err() != nil {
					break
				}

				// Exponential backoff
				if attempt < h.MaxRetries {
					time.Sleep(time.Duration(1<<attempt) * 100 * time.Millisecond)
				}
			}

			if err != nil {
				errCh <- gerror.Wrap(err, gerror.ErrCodeInternal, "handler failed").
					WithDetails("handler_id", h.ID).
					WithDetails("attempts", h.MaxRetries+1)
			}
		}(handler)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "handler execution failed", nil).
			WithDetails("error_count", len(errors)).
			WithDetails("errors", errors)
	}

	return nil
}

// generateHandlerID generates a unique handler ID
func generateHandlerID() string {
	return fmt.Sprintf("handler_%d_%d", time.Now().UnixNano(), rand.Int63())
}

// HandlerChain allows chaining multiple handlers together
type HandlerChain struct {
	handlers []EventHandler
}

// NewHandlerChain creates a new handler chain
func NewHandlerChain(handlers ...EventHandler) *HandlerChain {
	return &HandlerChain{
		handlers: handlers,
	}
}

// Add adds a handler to the chain
func (hc *HandlerChain) Add(handler EventHandler) *HandlerChain {
	hc.handlers = append(hc.handlers, handler)
	return hc
}

// Handle executes the handler chain
func (hc *HandlerChain) Handle(ctx context.Context, event Event) error {
	for i, handler := range hc.handlers {
		if err := handler(ctx, event); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "handler chain failed").
				WithDetails("handler_index", i)
		}
	}
	return nil
}

// MiddlewareHandler wraps a handler with middleware
type MiddlewareHandler struct {
	handler    EventHandler
	middleware []Middleware
}

// Middleware defines the middleware interface
type Middleware func(EventHandler) EventHandler

// NewMiddlewareHandler creates a new middleware handler
func NewMiddlewareHandler(handler EventHandler, middleware ...Middleware) *MiddlewareHandler {
	return &MiddlewareHandler{
		handler:    handler,
		middleware: middleware,
	}
}

// Handle executes the handler with middleware
func (mh *MiddlewareHandler) Handle(ctx context.Context, event Event) error {
	// Apply middleware in reverse order
	handler := mh.handler
	for i := len(mh.middleware) - 1; i >= 0; i-- {
		handler = mh.middleware[i](handler)
	}

	return handler(ctx, event)
}

// LoggingMiddleware adds logging to event handling
func LoggingMiddleware(logger interface{ Infof(string, ...interface{}) }) Middleware {
	return func(next EventHandler) EventHandler {
		return func(ctx context.Context, event Event) error {
			start := time.Now()
			logger.Infof("Handling event: type=%s, id=%s", event.GetType(), event.GetID())

			err := next(ctx, event)

			logger.Infof("Event handled: type=%s, id=%s, duration=%s, error=%v",
				event.GetType(), event.GetID(), time.Since(start), err)

			return err
		}
	}
}

// MetricsMiddleware adds metrics collection
func MetricsMiddleware(metrics interface {
	RecordEventHandled(string, time.Duration, error)
}) Middleware {
	return func(next EventHandler) EventHandler {
		return func(ctx context.Context, event Event) error {
			start := time.Now()
			err := next(ctx, event)
			metrics.RecordEventHandled(event.GetType(), time.Since(start), err)
			return err
		}
	}
}

// RecoveryMiddleware adds panic recovery
func RecoveryMiddleware() Middleware {
	return func(next EventHandler) EventHandler {
		return func(ctx context.Context, event Event) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = gerror.New(gerror.ErrCodePanic, "handler panicked", nil).
						WithDetails("panic", fmt.Sprintf("%v", r)).
						WithDetails("event_type", event.GetType())
				}
			}()

			return next(ctx, event)
		}
	}
}
