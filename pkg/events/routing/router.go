// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package routing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/eventbus"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// Router provides advanced event routing capabilities
type Router struct {
	mu         sync.RWMutex
	routes     map[string]*Route
	middleware []eventbus.Middleware
	bus        events.EventBus
	metrics    *RouterMetrics
}

// Route defines an event route with rules and handlers
type Route struct {
	ID          string
	Name        string
	Description string
	Rules       []RoutingRule
	Handler     events.EventHandler
	Transform   TransformFunc
	Priority    int
	Enabled     bool
	Metadata    map[string]interface{}
}

// RoutingRule defines when a route should be activated
type RoutingRule interface {
	Matches(ctx context.Context, event events.CoreEvent) (bool, error)
}

// TransformFunc transforms an event before routing
type TransformFunc func(ctx context.Context, event events.CoreEvent) (events.CoreEvent, error)

// RouterMetrics tracks routing metrics
type RouterMetrics struct {
	mu           sync.RWMutex
	routeMatches map[string]int64
	routeMisses  map[string]int64
	routeErrors  map[string]int64
}

// NewRouter creates a new event router
func NewRouter(bus events.EventBus) *Router {
	return &Router{
		routes: make(map[string]*Route),
		bus:    bus,
		metrics: &RouterMetrics{
			routeMatches: make(map[string]int64),
			routeMisses:  make(map[string]int64),
			routeErrors:  make(map[string]int64),
		},
	}
}

// AddRoute adds a route to the router
func (r *Router) AddRoute(route *Route) error {
	if route == nil {
		return gerror.New(gerror.ErrCodeValidation, "route is required", nil)
	}

	if route.ID == "" {
		route.ID = generateRouteID()
	}

	if route.Handler == nil {
		return gerror.New(gerror.ErrCodeValidation, "route handler is required", nil)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.routes[route.ID]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "route already exists", nil).
			WithDetails("route_id", route.ID)
	}

	r.routes[route.ID] = route
	return nil
}

// RemoveRoute removes a route
func (r *Router) RemoveRoute(routeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.routes[routeID]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "route not found", nil).
			WithDetails("route_id", routeID)
	}

	delete(r.routes, routeID)
	return nil
}

// UseMiddleware adds middleware to the router
func (r *Router) UseMiddleware(middleware ...eventbus.Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware...)
}

// Route routes an event through matching routes
func (r *Router) Route(ctx context.Context, event events.CoreEvent) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	r.mu.RLock()
	routes := r.getMatchingRoutes(ctx, event)
	middleware := r.middleware
	r.mu.RUnlock()

	if len(routes) == 0 {
		// No matching routes, use default bus
		return r.bus.Publish(ctx, event)
	}

	// Execute routes in priority order
	for _, route := range routes {
		if !route.Enabled {
			continue
		}

		// Apply transformation if defined
		routeEvent := event
		if route.Transform != nil {
			transformed, err := route.Transform(ctx, event)
			if err != nil {
				r.recordError(route.ID)
				return gerror.Wrap(err, gerror.ErrCodeInternal, "event transformation failed").
					WithDetails("route_id", route.ID)
			}
			routeEvent = transformed
		}

		// Build handler with middleware
		handler := route.Handler
		for i := len(middleware) - 1; i >= 0; i-- {
			handler = middleware[i](handler)
		}

		// Execute handler
		if err := handler(ctx, routeEvent); err != nil {
			r.recordError(route.ID)
			return gerror.Wrap(err, gerror.ErrCodeInternal, "route handler failed").
				WithDetails("route_id", route.ID)
		}

		r.recordMatch(route.ID)
	}

	return nil
}

// getMatchingRoutes returns routes that match the event
func (r *Router) getMatchingRoutes(ctx context.Context, event events.CoreEvent) []*Route {
	var matches []*Route

	for _, route := range r.routes {
		if r.matchesRoute(ctx, event, route) {
			matches = append(matches, route)
		}
	}

	// Sort by priority
	sortRoutesByPriority(matches)
	return matches
}

// matchesRoute checks if an event matches a route
func (r *Router) matchesRoute(ctx context.Context, event events.CoreEvent, route *Route) bool {
	// If no rules, route matches all events
	if len(route.Rules) == 0 {
		return true
	}

	// Check all rules (AND logic)
	for _, rule := range route.Rules {
		matches, err := rule.Matches(ctx, event)
		if err != nil || !matches {
			r.recordMiss(route.ID)
			return false
		}
	}

	return true
}

// recordMatch records a route match
func (r *Router) recordMatch(routeID string) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	r.metrics.routeMatches[routeID]++
}

// recordMiss records a route miss
func (r *Router) recordMiss(routeID string) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	r.metrics.routeMisses[routeID]++
}

// recordError records a route error
func (r *Router) recordError(routeID string) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()
	r.metrics.routeErrors[routeID]++
}

// GetMetrics returns router metrics
func (r *Router) GetMetrics() map[string]interface{} {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	metrics := make(map[string]interface{})
	metrics["route_matches"] = copyMap(r.metrics.routeMatches)
	metrics["route_misses"] = copyMap(r.metrics.routeMisses)
	metrics["route_errors"] = copyMap(r.metrics.routeErrors)

	return metrics
}

// Helper functions

func generateRouteID() string {
	return fmt.Sprintf("route_%d", time.Now().UnixNano())
}

func sortRoutesByPriority(routes []*Route) {
	for i := 0; i < len(routes); i++ {
		for j := i + 1; j < len(routes); j++ {
			if routes[j].Priority > routes[i].Priority {
				routes[i], routes[j] = routes[j], routes[i]
			}
		}
	}
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	result := make(map[K]V, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
