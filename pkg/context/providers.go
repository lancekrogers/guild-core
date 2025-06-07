package context

import (
	"context"
	"fmt"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// ProviderClient represents a context-aware LLM provider client
type ProviderClient interface {
	// Complete generates a completion with full context support
	Complete(ctx context.Context, prompt string) (string, error)

	// CreateCompletion creates a completion with detailed request/response
	CreateCompletion(ctx context.Context, req CompletionRequest) (CompletionResponse, error)

	// GetProviderInfo returns information about this provider
	GetProviderInfo() ProviderInfo
}

// CompletionRequest represents a context-aware completion request
type CompletionRequest struct {
	Prompt          string                 `json:"prompt"`
	Model           string                 `json:"model,omitempty"`
	MaxTokens       int                    `json:"max_tokens,omitempty"`
	Temperature     float64                `json:"temperature,omitempty"`
	TopP            float64                `json:"top_p,omitempty"`
	Stop            []string               `json:"stop,omitempty"`
	Stream          bool                   `json:"stream,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`

	// Context-specific fields
	RequestID       string                 `json:"request_id,omitempty"`
	SessionID       string                 `json:"session_id,omitempty"`
	AgentID         string                 `json:"agent_id,omitempty"`
	Operation       string                 `json:"operation,omitempty"`
	CostBudget      float64                `json:"cost_budget,omitempty"`
	TimeoutSeconds  int                    `json:"timeout_seconds,omitempty"`
}

// CompletionResponse represents a context-aware completion response
type CompletionResponse struct {
	ID              string                 `json:"id"`
	Object          string                 `json:"object"`
	Created         int64                  `json:"created"`
	Model           string                 `json:"model"`
	Content         string                 `json:"content"`
	FinishReason    string                 `json:"finish_reason"`
	Usage           UsageInfo              `json:"usage"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`

	// Context-specific fields
	RequestID       string                 `json:"request_id,omitempty"`
	ProcessingTime  time.Duration          `json:"processing_time,omitempty"`
	CostUSD         float64                `json:"cost_usd,omitempty"`
	Provider        string                 `json:"provider,omitempty"`
}

// UsageInfo contains token usage statistics
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProviderInfo contains information about a provider
type ProviderInfo struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Models       []string `json:"models"`
	Capabilities []string `json:"capabilities"`
	Version      string   `json:"version,omitempty"`
}

// ==============================================================================
// Context-Aware Provider Operations
// ==============================================================================

// CompleteWithProvider performs a completion using a specific provider from context
func CompleteWithProvider(ctx context.Context, providerName, prompt string) (string, error) {
	// Get provider from context
	provider, err := GetProviderFromContext(ctx, providerName)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get provider").WithComponent("context").WithOperation("CompleteWithProvider").WithDetails("provider_name", providerName)
	}

	// Try to cast to our context-aware interface
	if contextProvider, ok := provider.(ProviderClient); ok {
		return contextProvider.Complete(ctx, prompt)
	}

	// Fallback to basic LLM client interface
	if llmClient, ok := provider.(interface {
		Complete(context.Context, string) (string, error)
	}); ok {
		return llmClient.Complete(ctx, prompt)
	}

	return "", gerror.Newf(gerror.ErrCodeInvalidInput, "provider '%s' does not implement required completion interface", providerName).WithComponent("context").WithOperation("CompleteWithProvider")
}

// CompleteWithDefaultProvider performs a completion using the default provider from context
func CompleteWithDefaultProvider(ctx context.Context, prompt string) (string, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("CompleteWithDefaultProvider")
	}

	// Get default provider
	provider, err := registry.Providers().GetDefaultProvider()
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get default provider").WithComponent("context").WithOperation("CompleteWithDefaultProvider")
	}

	// Try to cast to our context-aware interface
	if contextProvider, ok := provider.(ProviderClient); ok {
		return contextProvider.Complete(ctx, prompt)
	}

	// Fallback to basic LLM client interface
	if llmClient, ok := provider.(interface {
		Complete(context.Context, string) (string, error)
	}); ok {
		return llmClient.Complete(ctx, prompt)
	}

	return "", gerror.New(gerror.ErrCodeInvalidInput, "default provider does not implement required completion interface", nil).WithComponent("context").WithOperation("CompleteWithDefaultProvider")
}

// CreateCompletionWithProvider creates a detailed completion request
func CreateCompletionWithProvider(ctx context.Context, providerName string, req CompletionRequest) (CompletionResponse, error) {
	// Enhance request with context information
	req = enhanceCompletionRequest(ctx, req)

	// Get provider from context
	provider, err := GetProviderFromContext(ctx, providerName)
	if err != nil {
		return CompletionResponse{}, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get provider").WithComponent("context").WithOperation("CreateCompletionWithProvider").WithDetails("provider_name", providerName)
	}

	// Try to cast to our context-aware interface
	if contextProvider, ok := provider.(ProviderClient); ok {
		return contextProvider.CreateCompletion(ctx, req)
	}

	return CompletionResponse{}, gerror.Newf(gerror.ErrCodeInvalidInput, "provider '%s' does not implement context-aware completion interface", providerName).WithComponent("context").WithOperation("CreateCompletionWithProvider")
}

// enhanceCompletionRequest adds context information to the completion request
func enhanceCompletionRequest(ctx context.Context, req CompletionRequest) CompletionRequest {
	if req.RequestID == "" {
		req.RequestID = GetRequestID(ctx)
	}
	if req.SessionID == "" {
		req.SessionID = GetSessionID(ctx)
	}
	if req.AgentID == "" {
		req.AgentID = GetAgentID(ctx)
	}
	if req.Operation == "" {
		req.Operation = GetOperation(ctx)
	}

	// Add cost budget if available
	if costInfo := GetCostInfo(ctx); costInfo != nil && req.CostBudget == 0 {
		req.CostBudget = costInfo.Budget - costInfo.Used
	}

	// Add timeout if available from context deadline
	if deadline, ok := ctx.Deadline(); ok {
		timeout := time.Until(deadline)
		if timeout > 0 && req.TimeoutSeconds == 0 {
			req.TimeoutSeconds = int(timeout.Seconds())
		}
	}

	return req
}

// ==============================================================================
// Provider Selection and Routing
// ==============================================================================

// SelectBestProvider chooses the best provider for a given task type
func SelectBestProvider(ctx context.Context, taskType string, requirements map[string]interface{}) (string, error) {
	registry, err := GetRegistryProvider(ctx)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to get registry from context").WithComponent("context").WithOperation("SelectBestProvider")
	}

	// Get all available providers
	providers := registry.Providers().ListProviders()
	if len(providers) == 0 {
		return "", gerror.New(gerror.ErrCodeNotFound, "no providers available", nil).WithComponent("context").WithOperation("SelectBestProvider")
	}

	// Simple selection logic - in production this could be more sophisticated
	// considering factors like cost, performance, model capabilities, etc.

	switch taskType {
	case "coding", "code-review", "debugging":
		// Prefer Claude Code for coding tasks
		for _, provider := range providers {
			if provider == "claudecode" {
				return provider, nil
			}
		}
		// Fallback to Claude or OpenAI for coding
		for _, provider := range providers {
			if provider == "anthropic" || provider == "openai" {
				return provider, nil
			}
		}

	case "reasoning", "analysis":
		// Prefer Claude or GPT-4 for reasoning tasks
		for _, provider := range providers {
			if provider == "anthropic" || provider == "openai" {
				return provider, nil
			}
		}

	case "fast", "simple":
		// Prefer faster models
		for _, provider := range providers {
			if provider == "anthropic" || provider == "google" {
				return provider, nil
			}
		}

	case "local", "private":
		// Prefer local models
		for _, provider := range providers {
			if provider == "ollama" {
				return provider, nil
			}
		}
	}

	// Default to first available provider
	return providers[0], nil
}

// RouteToProvider routes a request to the most appropriate provider
func RouteToProvider(ctx context.Context, taskType, prompt string, requirements map[string]interface{}) (string, error) {
	// Select best provider
	providerName, err := SelectBestProvider(ctx, taskType, requirements)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to select provider").WithComponent("context").WithOperation("RouteToProvider")
	}

	// Create enhanced context
	ctx = WithProvider(ctx, providerName)
	ctx = WithOperation(ctx, fmt.Sprintf("completion-%s", taskType))

	// Perform completion
	return CompleteWithProvider(ctx, providerName, prompt)
}

// ==============================================================================
// Provider Monitoring and Observability
// ==============================================================================

// ProviderMetrics contains metrics for provider operations
type ProviderMetrics struct {
	ProviderName    string        `json:"provider_name"`
	RequestCount    int64         `json:"request_count"`
	SuccessCount    int64         `json:"success_count"`
	ErrorCount      int64         `json:"error_count"`
	AvgLatency      time.Duration `json:"avg_latency"`
	TotalCost       float64       `json:"total_cost"`
	TotalTokens     int64         `json:"total_tokens"`
	LastUsed        time.Time     `json:"last_used"`
}

// TrackProviderUsage records provider usage metrics in context
func TrackProviderUsage(ctx context.Context, providerName string, latency time.Duration, cost float64, tokens int, err error) {
	// This could write to a metrics store, logger, or context metadata
	logger := GetLogger(ctx)
	if logger != nil {
		fields := []interface{}{
			"provider", providerName,
			"latency_ms", latency.Milliseconds(),
			"cost_usd", cost,
			"tokens", tokens,
		}

		if err != nil {
			fields = append(fields, "error", err.Error())
			logger.Error("Provider request failed", fields...)
		} else {
			logger.Info("Provider request completed", fields...)
		}
	}
}

// WithProviderMetrics adds provider metrics tracking to context
func WithProviderMetrics(ctx context.Context, metrics *ProviderMetrics) context.Context {
	return context.WithValue(ctx, "guild.provider_metrics", metrics)
}

// GetProviderMetrics retrieves provider metrics from context
func GetProviderMetrics(ctx context.Context) *ProviderMetrics {
	if metrics, ok := ctx.Value("guild.provider_metrics").(*ProviderMetrics); ok {
		return metrics
	}
	return nil
}
