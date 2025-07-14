// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package eventbus

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// TopicRouter provides topic-based event routing
type TopicRouter struct {
	mu           sync.RWMutex
	topics       map[string]*Topic
	wildcardSubs map[string][]topicSubscription
}

// Topic represents an event topic with subscribers
type Topic struct {
	name        string
	subscribers []topicSubscription
	mu          sync.RWMutex
	eventFilter EventFilter
}

// topicSubscription wraps a handler with metadata
type topicSubscription struct {
	id       string
	handler  EventHandler
	filter   EventFilter
	metadata map[string]interface{}
}

// EventFilter defines filtering criteria for events
type EventFilter interface {
	// Match returns true if the event matches the filter
	Match(event Event) bool
}

// NewTopicRouter creates a new topic router
func NewTopicRouter() *TopicRouter {
	return &TopicRouter{
		topics:       make(map[string]*Topic),
		wildcardSubs: make(map[string][]topicSubscription),
	}
}

// CreateTopic creates a new topic
func (tr *TopicRouter) CreateTopic(name string, filter EventFilter) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, exists := tr.topics[name]; exists {
		return gerror.New(gerror.ErrCodeAlreadyExists, "topic already exists", nil).
			WithDetails("topic", name)
	}

	tr.topics[name] = &Topic{
		name:        name,
		subscribers: make([]topicSubscription, 0),
		eventFilter: filter,
	}

	return nil
}

// DeleteTopic deletes a topic
func (tr *TopicRouter) DeleteTopic(name string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, exists := tr.topics[name]; !exists {
		return gerror.New(gerror.ErrCodeNotFound, "topic not found", nil).
			WithDetails("topic", name)
	}

	delete(tr.topics, name)
	return nil
}

// Subscribe subscribes to a topic or pattern
func (tr *TopicRouter) Subscribe(pattern string, handler EventHandler, filter EventFilter) (string, error) {
	if handler == nil {
		return "", gerror.New(gerror.ErrCodeValidation, "handler is required", nil)
	}

	subscriptionID := generateSubscriptionID()
	sub := topicSubscription{
		id:       subscriptionID,
		handler:  handler,
		filter:   filter,
		metadata: make(map[string]interface{}),
	}

	// Check if it's a wildcard pattern
	if strings.Contains(pattern, "*") {
		tr.mu.Lock()
		tr.wildcardSubs[pattern] = append(tr.wildcardSubs[pattern], sub)
		tr.mu.Unlock()
		return subscriptionID, nil
	}

	// Regular topic subscription
	tr.mu.RLock()
	topic, exists := tr.topics[pattern]
	tr.mu.RUnlock()

	if !exists {
		// Auto-create topic if it doesn't exist
		if err := tr.CreateTopic(pattern, nil); err != nil && !strings.Contains(err.Error(), "already exists") {
			return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create topic")
		}

		tr.mu.RLock()
		topic = tr.topics[pattern]
		tr.mu.RUnlock()
	}

	topic.mu.Lock()
	topic.subscribers = append(topic.subscribers, sub)
	topic.mu.Unlock()

	return subscriptionID, nil
}

// Unsubscribe removes a subscription
func (tr *TopicRouter) Unsubscribe(subscriptionID string) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	// Check wildcard subscriptions
	for pattern, subs := range tr.wildcardSubs {
		for i, sub := range subs {
			if sub.id == subscriptionID {
				// Remove subscription
				tr.wildcardSubs[pattern] = append(subs[:i], subs[i+1:]...)
				if len(tr.wildcardSubs[pattern]) == 0 {
					delete(tr.wildcardSubs, pattern)
				}
				return nil
			}
		}
	}

	// Check topic subscriptions
	for _, topic := range tr.topics {
		topic.mu.Lock()
		for i, sub := range topic.subscribers {
			if sub.id == subscriptionID {
				// Remove subscription
				topic.subscribers = append(topic.subscribers[:i], topic.subscribers[i+1:]...)
				topic.mu.Unlock()
				return nil
			}
		}
		topic.mu.Unlock()
	}

	return gerror.New(gerror.ErrCodeNotFound, "subscription not found", nil).
		WithDetails("subscription_id", subscriptionID)
}

// Route routes an event to appropriate handlers
func (tr *TopicRouter) Route(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	eventType := event.GetType()
	handlers := make([]EventHandler, 0)

	tr.mu.RLock()

	// Find exact topic match
	if topic, exists := tr.topics[eventType]; exists {
		// Check topic filter
		if topic.eventFilter == nil || topic.eventFilter.Match(event) {
			topic.mu.RLock()
			for _, sub := range topic.subscribers {
				// Check subscriber filter
				if sub.filter == nil || sub.filter.Match(event) {
					handlers = append(handlers, sub.handler)
				}
			}
			topic.mu.RUnlock()
		}
	}

	// Find wildcard matches
	for pattern, subs := range tr.wildcardSubs {
		if matchesPattern(eventType, pattern) {
			for _, sub := range subs {
				if sub.filter == nil || sub.filter.Match(event) {
					handlers = append(handlers, sub.handler)
				}
			}
		}
	}

	tr.mu.RUnlock()

	// Execute handlers concurrently
	if len(handlers) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		go func(h events.EventHandler) {
			defer wg.Done()
			if err := h(ctx, event); err != nil {
				errors <- err
			}
		}(handler)
	}

	wg.Wait()
	close(errors)

	// Collect errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "handler errors occurred", nil).
			WithDetails("error_count", len(errs)).
			WithDetails("errors", errs)
	}

	return nil
}

// GetTopics returns all topics
func (tr *TopicRouter) GetTopics() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	topics := make([]string, 0, len(tr.topics))
	for name := range tr.topics {
		topics = append(topics, name)
	}
	return topics
}

// GetTopicStats returns statistics for a topic
func (tr *TopicRouter) GetTopicStats(topicName string) (map[string]interface{}, error) {
	tr.mu.RLock()
	topic, exists := tr.topics[topicName]
	tr.mu.RUnlock()

	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "topic not found", nil).
			WithDetails("topic", topicName)
	}

	topic.mu.RLock()
	defer topic.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["name"] = topic.name
	stats["subscriber_count"] = len(topic.subscribers)
	stats["has_filter"] = topic.eventFilter != nil

	return stats, nil
}

// matchesPattern checks if a string matches a wildcard pattern
func matchesPattern(str, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return str == pattern
	}

	// Check prefix
	if parts[0] != "" && !strings.HasPrefix(str, parts[0]) {
		return false
	}

	// Check suffix
	if parts[len(parts)-1] != "" && !strings.HasSuffix(str, parts[len(parts)-1]) {
		return false
	}

	// For now, simple implementation - could be enhanced
	return true
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	// In production, use a proper UUID generator
	return fmt.Sprintf("sub_%d_%d", time.Now().UnixNano(), rand.Int63())
}

// MetadataFilter filters events based on metadata
type MetadataFilter struct {
	RequiredKeys   []string
	RequiredValues map[string]interface{}
}

// Match checks if an event matches the metadata filter
func (mf *MetadataFilter) Match(event Event) bool {
	metadata := event.GetMetadata()
	if metadata == nil {
		return len(mf.RequiredKeys) == 0 && len(mf.RequiredValues) == 0
	}

	// Check required keys
	for _, key := range mf.RequiredKeys {
		if _, exists := metadata[key]; !exists {
			return false
		}
	}

	// Check required values
	for key, requiredValue := range mf.RequiredValues {
		if value, exists := metadata[key]; !exists || value != requiredValue {
			return false
		}
	}

	return true
}

// SourceFilter filters events based on source
type SourceFilter struct {
	AllowedSources []string
	DeniedSources  []string
}

// Match checks if an event matches the source filter
func (sf *SourceFilter) Match(event Event) bool {
	source := event.GetSource()

	// Check denied sources first
	for _, denied := range sf.DeniedSources {
		if source == denied || matchesPattern(source, denied) {
			return false
		}
	}

	// If no allowed sources specified, allow all (except denied)
	if len(sf.AllowedSources) == 0 {
		return true
	}

	// Check allowed sources
	for _, allowed := range sf.AllowedSources {
		if source == allowed || matchesPattern(source, allowed) {
			return true
		}
	}

	return false
}

// CompositeFilter combines multiple filters
type CompositeFilter struct {
	Filters []EventFilter
	All     bool // true = all must match, false = any must match
}

// Match checks if an event matches the composite filter
func (cf *CompositeFilter) Match(event Event) bool {
	if len(cf.Filters) == 0 {
		return true
	}

	if cf.All {
		// All filters must match
		for _, filter := range cf.Filters {
			if filter != nil && !filter.Match(event) {
				return false
			}
		}
		return true
	}

	// Any filter must match
	for _, filter := range cf.Filters {
		if filter != nil && filter.Match(event) {
			return true
		}
	}
	return false
}
