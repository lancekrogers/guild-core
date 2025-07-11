package collectors

import (
	"context"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ChatCollector collects chat interaction metrics
type ChatCollector struct {
	meter              metric.Meter
	messageCount       metric.Int64Counter
	responseTime       metric.Float64Histogram
	sessionDuration    metric.Float64Histogram
	activeSessions     metric.Int64UpDownCounter
	streamingLatency   metric.Float64Histogram
	suggestionAccepted metric.Int64Counter
	suggestionOffered  metric.Int64Counter
	errorResponses     metric.Int64Counter
}

// NewChatCollector creates a new chat metrics collector
func NewChatCollector(meter metric.Meter) (*ChatCollector, error) {
	cc := &ChatCollector{meter: meter}

	if err := cc.initializeMetrics(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize chat metrics")
	}

	return cc, nil
}

// initializeMetrics creates all chat metric instruments
func (cc *ChatCollector) initializeMetrics() error {
	var err error

	cc.messageCount, err = cc.meter.Int64Counter(
		"guild.chat.messages.total",
		metric.WithDescription("Total number of chat messages"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create message count counter")
	}

	cc.responseTime, err = cc.meter.Float64Histogram(
		"guild.chat.response.duration",
		metric.WithDescription("Time to generate chat responses"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create response time histogram")
	}

	cc.sessionDuration, err = cc.meter.Float64Histogram(
		"guild.chat.session.duration",
		metric.WithDescription("Duration of chat sessions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create session duration histogram")
	}

	cc.activeSessions, err = cc.meter.Int64UpDownCounter(
		"guild.chat.sessions.active",
		metric.WithDescription("Number of active chat sessions"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create active sessions counter")
	}

	cc.streamingLatency, err = cc.meter.Float64Histogram(
		"guild.chat.streaming.latency",
		metric.WithDescription("Latency of streaming responses"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create streaming latency histogram")
	}

	cc.suggestionAccepted, err = cc.meter.Int64Counter(
		"guild.chat.suggestions.accepted",
		metric.WithDescription("Number of accepted suggestions"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create suggestion accepted counter")
	}

	cc.suggestionOffered, err = cc.meter.Int64Counter(
		"guild.chat.suggestions.offered",
		metric.WithDescription("Number of suggestions offered"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create suggestion offered counter")
	}

	cc.errorResponses, err = cc.meter.Int64Counter(
		"guild.chat.errors.total",
		metric.WithDescription("Number of error responses"),
	)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create error responses counter")
	}

	return nil
}

// RecordMessage records a chat message
func (cc *ChatCollector) RecordMessage(ctx context.Context, sessionID string, messageType string, provider string) {
	attrs := []attribute.KeyValue{
		attribute.String("session.id", sessionID),
		attribute.String("message.type", messageType),
		attribute.String("provider", provider),
	}
	cc.messageCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordResponse records response generation metrics
func (cc *ChatCollector) RecordResponse(ctx context.Context, sessionID string, provider string, duration time.Duration, tokenCount int, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("session.id", sessionID),
		attribute.String("provider", provider),
		attribute.Int("token.count", tokenCount),
		attribute.Bool("success", success),
	}

	cc.responseTime.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if !success {
		cc.errorResponses.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordSessionStart records the start of a chat session
func (cc *ChatCollector) RecordSessionStart(ctx context.Context, sessionID string, userID string) {
	attrs := []attribute.KeyValue{
		attribute.String("session.id", sessionID),
		attribute.String("user.id", userID),
	}
	cc.activeSessions.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordSessionEnd records the end of a chat session
func (cc *ChatCollector) RecordSessionEnd(ctx context.Context, sessionID string, userID string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("session.id", sessionID),
		attribute.String("user.id", userID),
	}
	cc.activeSessions.Add(ctx, -1, metric.WithAttributes(attrs...))
	cc.sessionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordStreamingLatency records streaming response latency
func (cc *ChatCollector) RecordStreamingLatency(ctx context.Context, sessionID string, chunkIndex int, latency time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("session.id", sessionID),
		attribute.Int("chunk.index", chunkIndex),
	}
	cc.streamingLatency.Record(ctx, float64(latency.Milliseconds()), metric.WithAttributes(attrs...))
}

// RecordSuggestion records suggestion metrics
func (cc *ChatCollector) RecordSuggestion(ctx context.Context, sessionID string, suggestionType string, accepted bool) {
	attrs := []attribute.KeyValue{
		attribute.String("session.id", sessionID),
		attribute.String("suggestion.type", suggestionType),
	}

	cc.suggestionOffered.Add(ctx, 1, metric.WithAttributes(attrs...))

	if accepted {
		cc.suggestionAccepted.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}
