package observability

import (
	"context"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ExampleAgentExecution shows how to integrate observability into agent execution
func ExampleAgentExecution(ctx context.Context) error {
	// Ensure request context
	ctx = EnsureRequestContext(ctx)

	// Get logger and add context
	logger := GetLogger(ctx).
		WithComponent("agent").
		WithOperation("execute_task")

	// Start a trace span
	ctx, span := StartAgentSpan(ctx, "agent-123", "execute_task")
	defer span.End()

	// Get metrics
	metrics := GetMetrics()

	logger.InfoContext(ctx, "Starting agent task execution",
		"agent_id", "agent-123",
		"task_id", "task-456",
	)

	start := time.Now()

	// Simulate task execution
	err := executeTask(ctx)

	duration := time.Since(start)

	if err != nil {
		// Create structured error
		gerr := gerror.Wrap(err, gerror.ErrCodeAgentFailed, "agent task execution failed").
			WithComponent("agent").
			WithOperation("execute_task").
			WithDetails("agent_id", "agent-123").
			WithDetails("task_id", "task-456").
			FromContext(ctx)

		// Log error with context
		logger.WithError(gerr).ErrorContext(ctx, "Task execution failed")

		// Record error in trace
		RecordError(ctx, gerr)

		// Record metrics
		metrics.RecordAgentTask("agent-123", "worker", "failed")
		metrics.RecordError(string(gerr.Code), "agent", "execute_task")

		return gerr
	}

	// Success logging and metrics
	logger.InfoContext(ctx, "Task execution completed successfully",
		"duration_ms", duration.Milliseconds(),
	)

	metrics.RecordAgentTask("agent-123", "worker", "success")
	metrics.RecordAgentTaskDuration("agent-123", "worker", duration)

	return nil
}

// ExampleStorageOperation shows how to integrate observability into storage operations
func ExampleStorageOperation(ctx context.Context) error {
	logger := GetLogger(ctx).WithComponent("storage")
	metrics := GetMetrics()

	// Start storage span
	ctx, span := StartStorageSpan(ctx, "create", "tasks")
	defer span.End()

	start := time.Now()

	// Simulate storage operation
	err := performStorageOperation(ctx)

	duration := time.Since(start)

	if err != nil {
		// Check for specific errors
		if gerror.Is(err, gerror.ErrNotFound) {
			logger.WarnContext(ctx, "Resource not found",
				"table", "tasks",
				"operation", "create",
			)
			metrics.RecordStorageError("create", "tasks", "not_found")
		} else {
			logger.WithError(err).ErrorContext(ctx, "Storage operation failed")
			metrics.RecordStorageError("create", "tasks", "unknown")
		}

		RecordError(ctx, err)
		return err
	}

	// Record success metrics
	metrics.RecordStorageOperation("create", "tasks", "success")
	metrics.RecordStorageDuration("create", "tasks", duration)

	return nil
}

// ExampleProviderCall shows how to integrate observability into provider calls
func ExampleProviderCall(ctx context.Context) error {
	logger := GetLogger(ctx).WithComponent("provider")
	metrics := GetMetrics()

	// Start provider span
	ctx, span := StartProviderSpan(ctx, "openai", "completion")
	defer span.End()

	// Set span attributes
	SetSpanAttributes(ctx, map[string]interface{}{
		"provider.model":       "gpt-4",
		"provider.max_tokens":  1000,
		"provider.temperature": 0.7,
	})

	start := time.Now()

	// Simulate provider call
	response, err := callProvider(ctx)

	duration := time.Since(start)

	if err != nil {
		// Handle retryable errors
		if gerror.IsRetryable(err) {
			logger.WarnContext(ctx, "Provider call failed (retryable)",
				"provider", "openai",
				"model", "gpt-4",
				"duration_ms", duration.Milliseconds(),
			)
			metrics.RecordProviderError("openai", "gpt-4", "retryable")
		} else {
			logger.WithError(err).ErrorContext(ctx, "Provider call failed")
			metrics.RecordProviderError("openai", "gpt-4", "permanent")
		}

		RecordError(ctx, err)
		return err
	}

	// Record success metrics
	logger.InfoContext(ctx, "Provider call completed",
		"provider", "openai",
		"model", "gpt-4",
		"prompt_tokens", response.PromptTokens,
		"completion_tokens", response.CompletionTokens,
		"duration_ms", duration.Milliseconds(),
	)

	metrics.RecordProviderRequest("openai", "gpt-4", "success")
	metrics.RecordProviderDuration("openai", "gpt-4", duration)
	metrics.RecordProviderTokens("openai", "gpt-4", "prompt", response.PromptTokens)
	metrics.RecordProviderTokens("openai", "gpt-4", "completion", response.CompletionTokens)
	metrics.RecordProviderCost("openai", "gpt-4", response.Cost)

	return nil
}

// Helper functions for examples
func executeTask(ctx context.Context) error {
	// Simulate task execution
	select {
	case <-ctx.Done():
		return gerror.New(gerror.ErrCodeCancelled, "task cancelled", ctx.Err())
	case <-time.After(100 * time.Millisecond):
		// Simulate random failure
		if time.Now().Unix()%10 == 0 {
			return gerror.New(gerror.ErrCodeInternal, "simulated task failure", nil).
				WithComponent("example").
				WithOperation("executeTask")
		}
		return nil
	}
}

func performStorageOperation(ctx context.Context) error {
	// Simulate storage operation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
		// Simulate random not found
		if time.Now().Unix()%20 == 0 {
			return gerror.ErrNotFound
		}
		return nil
	}
}

type providerResponse struct {
	PromptTokens     int
	CompletionTokens int
	Cost             float64
}

func callProvider(ctx context.Context) (*providerResponse, error) {
	// Simulate provider call
	select {
	case <-ctx.Done():
		return nil, gerror.New(gerror.ErrCodeTimeout, "provider call timed out", ctx.Err())
	case <-time.After(200 * time.Millisecond):
		// Simulate random failure
		if time.Now().Unix()%15 == 0 {
			return nil, gerror.New(gerror.ErrCodeProviderTimeout, "provider timeout", nil)
		}
		return &providerResponse{
			PromptTokens:     150,
			CompletionTokens: 50,
			Cost:             0.0025,
		}, nil
	}
}
