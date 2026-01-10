// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package reasoning

import (
	"context"
	"fmt"

	"github.com/lancekrogers/guild-core/pkg/events"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("guild.reasoning")

// TracedExtractor wraps an extractor with distributed tracing
type TracedExtractor struct {
	*Extractor
	circuitBreaker *CircuitBreaker
	rateLimiter    *RateLimiter
}

// NewTracedExtractor creates a new traced extractor
func NewTracedExtractor(extractor *Extractor, cb *CircuitBreaker, rl *RateLimiter) *TracedExtractor {
	return &TracedExtractor{
		Extractor:      extractor,
		circuitBreaker: cb,
		rateLimiter:    rl,
	}
}

// Extract performs extraction with distributed tracing
func (te *TracedExtractor) Extract(ctx context.Context, agentID, content string) ([]ReasoningBlock, error) {
	ctx, span := tracer.Start(ctx, "reasoning.extract",
		trace.WithAttributes(
			attribute.String("agent.id", agentID),
			attribute.Int("content.length", len(content)),
			attribute.String("extractor.version", "1.0"),
		),
	)
	defer span.End()

	// Circuit breaker check
	if te.circuitBreaker != nil {
		_, cbSpan := tracer.Start(ctx, "circuit_breaker.check")
		state := te.circuitBreaker.State()
		cbSpan.SetAttributes(
			attribute.String("state", state.String()),
			attribute.Bool("can_proceed", state != StateOpen),
		)
		cbSpan.End()

		if state == StateOpen {
			span.RecordError(ErrCircuitBreakerOpen)
			span.SetStatus(codes.Error, "circuit breaker open")
			return nil, ErrCircuitBreakerOpen
		}
	}

	// Rate limiter check
	if te.rateLimiter != nil {
		rlCtx, rlSpan := tracer.Start(ctx, "rate_limiter.check")
		err := te.rateLimiter.Allow(rlCtx, agentID)
		rlSpan.SetAttributes(
			attribute.String("agent.id", agentID),
			attribute.Bool("allowed", err == nil),
		)
		if err != nil {
			rlSpan.RecordError(err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "rate limit exceeded")
		}
		rlSpan.End()

		if err != nil {
			return nil, err
		}
	}

	// Actual extraction
	extractCtx, extractSpan := tracer.Start(ctx, "extract.blocks")
	blocks, err := te.Extractor.Extract(extractCtx, content)

	extractSpan.SetAttributes(
		attribute.Int("blocks.count", len(blocks)),
		attribute.Bool("error", err != nil),
	)

	if err != nil {
		extractSpan.RecordError(err)
		extractSpan.SetStatus(codes.Error, "extraction failed")
	} else {
		// Add block details
		for i, block := range blocks {
			extractSpan.AddEvent(fmt.Sprintf("block_%d_extracted", i),
				trace.WithAttributes(
					attribute.String("block.id", block.ID),
					attribute.String("block.type", block.Type),
					attribute.Int("block.tokens", block.TokenCount),
				),
			)
		}
	}
	extractSpan.End()

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "extraction failed")
	} else {
		span.SetStatus(codes.Ok, "extraction successful")
		span.SetAttributes(
			attribute.Int("blocks.extracted", len(blocks)),
		)
	}

	return blocks, err
}

// TracedRegistry wraps the registry with tracing
type TracedRegistry struct {
	*Registry
}

// NewTracedRegistry creates a new traced registry
func NewTracedRegistry(registry *Registry) *TracedRegistry {
	return &TracedRegistry{Registry: registry}
}

// Extract performs extraction with full tracing
func (tr *TracedRegistry) Extract(ctx context.Context, agentID, content string) ([]ReasoningBlock, error) {
	ctx, span := tracer.Start(ctx, "reasoning.registry.extract",
		trace.WithAttributes(
			attribute.String("agent.id", agentID),
			attribute.Int("content.length", len(content)),
		),
	)
	defer span.End()

	// Check if started
	tr.mu.RLock()
	started := tr.started
	tr.mu.RUnlock()

	if !started {
		err := ErrRegistryNotStarted
		span.RecordError(err)
		span.SetStatus(codes.Error, "registry not started")
		return nil, err
	}

	// Rate limiting span
	rlCtx, rlSpan := tracer.Start(ctx, "rate_limiter.wait")
	err := tr.rateLimiter.Wait(rlCtx, agentID)
	rlSpan.SetAttributes(
		attribute.String("agent.id", agentID),
		attribute.Bool("limited", err != nil),
	)
	if err != nil {
		rlSpan.RecordError(err)
		span.RecordError(err)
		span.SetStatus(codes.Error, "rate limit exceeded")
		rlSpan.End()
		return nil, err
	}
	rlSpan.End()

	// Retry loop with tracing
	var blocks []ReasoningBlock
	retryCtx, retrySpan := tracer.Start(rlCtx, "retry.execute")
	retrySpan.SetAttributes(
		attribute.Int("max_attempts", tr.retryer.config.MaxAttempts),
	)

	attempt := 0
	err = tr.retryer.Execute(retryCtx, func() error {
		attempt++
		attemptCtx, attemptSpan := tracer.Start(retryCtx, fmt.Sprintf("attempt_%d", attempt))
		attemptSpan.SetAttributes(
			attribute.Int("attempt.number", attempt),
		)

		// Circuit breaker execution
		cbErr := tr.circuitBreaker.Execute(attemptCtx, func() error {
			var extractErr error
			blocks, extractErr = tr.extractor.Extract(attemptCtx, content)
			return extractErr
		})

		if cbErr != nil {
			attemptSpan.RecordError(cbErr)
			attemptSpan.SetStatus(codes.Error, "attempt failed")
		} else {
			attemptSpan.SetStatus(codes.Ok, "attempt successful")
		}
		attemptSpan.End()

		return cbErr
	})

	retrySpan.SetAttributes(
		attribute.Int("attempts.total", attempt),
		attribute.Bool("success", err == nil),
	)
	retrySpan.End()

	if err != nil {
		// Dead letter queue span
		dlqCtx, dlqSpan := tracer.Start(retryCtx, "dead_letter.add")
		metadata := map[string]interface{}{
			"content_length": len(content),
			"provider":       "default",
			"trace_id":       span.SpanContext().TraceID().String(),
		}

		dlqErr := tr.deadLetter.Add(dlqCtx, agentID, content, err, attempt, metadata)
		dlqSpan.SetAttributes(
			attribute.Bool("added", dlqErr == nil),
		)
		if dlqErr != nil {
			dlqSpan.RecordError(dlqErr)
			tr.logger.Error("failed to add to dead letter queue",
				"error", dlqErr,
				"original_error", err,
				"agent_id", agentID)
		}
		dlqSpan.End()

		span.RecordError(err)
		span.SetStatus(codes.Error, "extraction failed after retries")
		return nil, err
	}

	span.SetStatus(codes.Ok, "extraction successful")
	span.SetAttributes(
		attribute.Int("blocks.count", len(blocks)),
		attribute.Int("retry.attempts", attempt),
	)

	return blocks, nil
}

// InstrumentEventBus adds tracing to event bus operations
func InstrumentEventBus(eventBus events.EventBus) events.EventBus {
	return &tracedEventBus{eventBus: eventBus}
}

type tracedEventBus struct {
	eventBus events.EventBus
}

func (t *tracedEventBus) Publish(ctx context.Context, event events.CoreEvent) error {
	eventType := event.GetType()
	ctx, span := tracer.Start(ctx, "event_bus.publish",
		trace.WithAttributes(
			attribute.String("event.type", eventType),
			attribute.String("event.id", event.GetID()),
		),
	)
	defer span.End()

	err := t.eventBus.Publish(ctx, event)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish failed")
	} else {
		span.SetStatus(codes.Ok, "published")
	}

	return err
}

func (t *tracedEventBus) Subscribe(ctx context.Context, eventType string, handler events.EventHandler) (events.SubscriptionID, error) {
	// Wrap the handler with tracing
	tracedHandler := func(ctx context.Context, event events.CoreEvent) error {
		ctx, span := tracer.Start(ctx, "event_bus.handle",
			trace.WithAttributes(
				attribute.String("event.type", eventType),
				attribute.String("event.id", event.GetID()),
			),
		)
		defer span.End()

		// Call original handler
		err := handler(ctx, event)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "handled")
		}

		return err
	}

	return t.eventBus.Subscribe(ctx, eventType, tracedHandler)
}

func (t *tracedEventBus) Unsubscribe(ctx context.Context, subscriptionID events.SubscriptionID) error {
	return t.eventBus.Unsubscribe(ctx, subscriptionID)
}

func (t *tracedEventBus) SubscribeAll(ctx context.Context, handler events.EventHandler) (events.SubscriptionID, error) {
	tracedHandler := func(ctx context.Context, event events.CoreEvent) error {
		ctx, span := tracer.Start(ctx, "event_bus.handle_all",
			trace.WithAttributes(
				attribute.String("event.type", event.GetType()),
				attribute.String("event.id", event.GetID()),
			),
		)
		defer span.End()

		err := handler(ctx, event)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "handled")
		}

		return err
	}

	return t.eventBus.SubscribeAll(ctx, tracedHandler)
}

func (t *tracedEventBus) PublishJSON(ctx context.Context, jsonEvent string) error {
	ctx, span := tracer.Start(ctx, "event_bus.publish_json")
	defer span.End()

	err := t.eventBus.PublishJSON(ctx, jsonEvent)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "published")
	}

	return err
}

func (t *tracedEventBus) Close(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "event_bus.close")
	defer span.End()

	err := t.eventBus.Close(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "closed")
	}

	return err
}

func (t *tracedEventBus) IsRunning() bool {
	return t.eventBus.IsRunning()
}

func (t *tracedEventBus) GetSubscriptionCount() int {
	return t.eventBus.GetSubscriptionCount()
}

// SpanFromContext is a helper to get the current span
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordError records an error on the current span
func RecordError(ctx context.Context, err error, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err, trace.WithAttributes(attrs...))
	span.SetStatus(codes.Error, err.Error())
}
