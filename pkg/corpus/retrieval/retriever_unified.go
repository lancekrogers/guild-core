// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package retrieval

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/observability"
)

// UnifiedRetriever wraps RetrieverImpl to use the unified event system
type UnifiedRetriever struct {
	*RetrieverImpl
	unifiedEventBus events.EventBus
}

// NewUnifiedRetriever creates a new retriever with unified event bus
func NewUnifiedRetriever(ctx context.Context, vectorStore VectorStore, metadataStore MetadataStore, ranker *ResultRanker, unifiedEventBus events.EventBus) (Retriever, error) {
	// Create base retriever
	baseRetriever, err := NewRetriever(ctx, vectorStore, metadataStore, ranker)
	if err != nil {
		return nil, err
	}

	// Cast to implementation to access internal fields
	retrieverImpl, ok := baseRetriever.(*RetrieverImpl)
	if !ok {
		// Fallback - return base retriever
		return baseRetriever, nil
	}

	return &UnifiedRetriever{
		RetrieverImpl:   retrieverImpl,
		unifiedEventBus: unifiedEventBus,
	}, nil
}

// Retrieve performs multi-strategy retrieval and publishes events to unified bus
func (ur *UnifiedRetriever) Retrieve(ctx context.Context, query Query) ([]RankedDocument, error) {
	// Start timing
	startTime := time.Now()

	// Use base implementation
	results, err := ur.RetrieverImpl.Retrieve(ctx, query)

	// Calculate duration
	duration := time.Since(startTime)

	// Publish event to unified bus
	if ur.unifiedEventBus != nil {
		eventData := map[string]interface{}{
			"query":         query.Text,
			"results_count": len(results),
			"strategies":    len(ur.strategies),
			"duration_ms":   duration.Milliseconds(),
			"max_results":   query.MaxResults,
			"min_score":     query.MinScore,
			"success":       err == nil,
		}

		// Add context information if available
		if query.Context.TaskID != "" {
			eventData["task_id"] = query.Context.TaskID
		}
		if query.Context.AgentID != "" {
			eventData["agent_id"] = query.Context.AgentID
		}
		if len(query.Context.Tags) > 0 {
			eventData["tags"] = query.Context.Tags
		}
		if len(query.Filters) > 0 {
			eventData["filters_count"] = len(query.Filters)
		}

		// Add error information if retrieval failed
		if err != nil {
			eventData["error"] = err.Error()
		}

		event := events.NewBaseEvent(
			uuid.New().String(),
			"retrieval.completed",
			"corpus.retriever",
			eventData,
		)

		// Publish event
		if publishErr := ur.unifiedEventBus.Publish(ctx, event); publishErr != nil {
			logger := observability.GetLogger(ctx).
				WithComponent("UnifiedRetriever").
				WithOperation("Retrieve")
			logger.WithError(publishErr).Warn("Failed to publish retrieval event to unified bus")
		}
	}

	return results, err
}

// SetEventBus overrides to prevent setting legacy event bus
func (ur *UnifiedRetriever) SetEventBus(eventBus EventBus) {
	// No-op: we only use the unified event bus
}

// UnifiedEventBusAdapter adapts unified events.EventBus to corpus EventBus interface
type UnifiedEventBusAdapter struct {
	eventBus events.EventBus
}

// NewUnifiedEventBusAdapter creates a new adapter
func NewUnifiedEventBusAdapter(eventBus events.EventBus) EventBus {
	return &UnifiedEventBusAdapter{
		eventBus: eventBus,
	}
}

// Publish adapts corpus Event to unified event system
func (a *UnifiedEventBusAdapter) Publish(ctx context.Context, event Event) error {
	if a.eventBus == nil {
		return nil
	}

	// Create unified event
	unifiedEvent := events.NewBaseEvent(
		uuid.New().String(),
		event.Type,
		event.Source,
		event.Data,
	)

	// BaseEvent includes timestamp in its creation

	return a.eventBus.Publish(ctx, unifiedEvent)
}
