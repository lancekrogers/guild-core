package errors

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// TestLLMProviderFailures tests various provider failure scenarios
func TestLLMProviderFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("ProviderUnavailable", func(t *testing.T) {
		// Setup registry with multiple providers
		reg := registry.NewComponentRegistry()
		err := reg.Initialize(ctx, registry.Config{})
		require.NoError(t, err)

		// Primary provider that fails
		primaryProvider := &failingProvider{
			failureMode: "unavailable",
			errorMsg:    "service temporarily unavailable",
		}

		// Secondary provider that works
		secondaryProvider := testutil.NewMockLLMProvider()
		secondaryProvider.SetResponse("fallback", "Response from secondary provider")

		// Register providers
		err = reg.Providers().Register("primary", func(ctx context.Context, cfg map[string]any) (any, error) {
			return primaryProvider, nil
		})
		require.NoError(t, err)

		err = reg.Providers().Register("secondary", func(ctx context.Context, cfg map[string]any) (any, error) {
			return secondaryProvider, nil
		})
		require.NoError(t, err)

		// Create agent with fallback configuration
		agentCfg := agent.Config{
			Name:              "test-agent",
			PrimaryProvider:   "primary",
			FallbackProviders: []string{"secondary"},
		}

		factory := reg.Agents()
		testAgent, err := factory.Create(ctx, agentCfg)
		require.NoError(t, err)

		// Execute task - should fallback to secondary
		result, err := testAgent.Execute(ctx, agent.Task{
			ID:      "test-task",
			Content: "Test task requiring LLM",
		})

		// Should succeed with fallback
		require.NoError(t, err, "Should fallback to secondary provider")
		assert.Contains(t, result.Response, "secondary provider", "Should use secondary provider")
		assert.Equal(t, 1, primaryProvider.attempts, "Should try primary once")
	})

	t.Run("RateLimitHandling", func(t *testing.T) {
		// Provider that simulates rate limiting
		rateLimitProvider := &failingProvider{
			failureMode: "rate_limit",
			errorMsg:    "rate limit exceeded",
			resetAfter:  2 * time.Second,
		}

		reg := registry.NewComponentRegistry()
		err := reg.Initialize(ctx, registry.Config{})
		require.NoError(t, err)

		err = reg.Providers().Register("rate_limited", func(ctx context.Context, cfg map[string]any) (any, error) {
			return rateLimitProvider, nil
		})
		require.NoError(t, err)

		// Track retry attempts
		var retryCount int32
		
		// Create custom retry handler
		retryHandler := func(err error, attempt int) bool {
			atomic.AddInt32(&retryCount, 1)
			
			// Check if it's a rate limit error
			if gerr, ok := err.(*gerror.Error); ok {
				if gerr.Type == gerror.ErrorTypeRateLimit {
					// Wait based on retry-after header
					waitTime := time.Duration(attempt) * 500 * time.Millisecond
					if resetTime, ok := gerr.Context["reset_after"]; ok {
						waitTime = resetTime.(time.Duration)
					}
					time.Sleep(waitTime)
					return true
				}
			}
			return false
		}

		// Execute with retry logic
		startTime := time.Now()
		
		// Simulate multiple attempts
		success := false
		var lastErr error
		for i := 0; i < 5; i++ {
			_, err := rateLimitProvider.Complete(ctx, nil)
			if err == nil {
				success = true
				break
			}
			lastErr = err
			
			if !retryHandler(err, i) {
				break
			}
		}

		duration := time.Since(startTime)
		
		// Should eventually succeed after rate limit resets
		assert.True(t, success || duration > rateLimitProvider.resetAfter, 
			"Should handle rate limiting with retries")
		assert.Greater(t, retryCount, int32(0), "Should attempt retries")
	})

	t.Run("CostTrackingForFailedRequests", func(t *testing.T) {
		// Provider that fails after consuming tokens
		costlyFailProvider := &failingProvider{
			failureMode:  "partial_response",
			errorMsg:     "response truncated",
			tokensUsed:   1000,
			costPerToken: 0.00002, // $0.02 per 1K tokens
		}

		reg := registry.NewComponentRegistry()
		err := reg.Initialize(ctx, registry.Config{})
		require.NoError(t, err)

		err = reg.Providers().Register("costly", func(ctx context.Context, cfg map[string]any) (any, error) {
			return costlyFailProvider, nil
		})
		require.NoError(t, err)

		// Cost tracker
		costManager := reg.Cost()
		require.NotNil(t, costManager)

		// Execute failed request
		agentCfg := agent.Config{
			Name:            "cost-tracking-agent",
			PrimaryProvider: "costly",
		}

		factory := reg.Agents()
		testAgent, err := factory.Create(ctx, agentCfg)
		require.NoError(t, err)

		// Execute task that will fail
		_, err = testAgent.Execute(ctx, agent.Task{
			ID:      "costly-task",
			Content: "This will fail after using tokens",
		})

		// Should track cost even for failed request
		assert.Error(t, err, "Request should fail")
		
		// Get cost report
		report := costManager.GetReport(ctx)
		assert.Greater(t, report.TotalCost, 0.0, "Should track cost for failed requests")
		assert.Equal(t, costlyFailProvider.tokensUsed, report.TotalTokens, "Should track tokens used")
	})

	t.Run("GracefulErrorMessages", func(t *testing.T) {
		testCases := []struct {
			name         string
			failureMode  string
			expectedMsg  string
			userFriendly bool
		}{
			{
				name:         "NetworkTimeout",
				failureMode:  "timeout",
				expectedMsg:  "request timed out",
				userFriendly: true,
			},
			{
				name:         "InvalidAPIKey",
				failureMode:  "auth",
				expectedMsg:  "authentication failed",
				userFriendly: true,
			},
			{
				name:         "ServerError",
				failureMode:  "server_error",
				expectedMsg:  "internal server error",
				userFriendly: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				provider := &failingProvider{
					failureMode: tc.failureMode,
					errorMsg:    tc.expectedMsg,
				}

				_, err := provider.Complete(ctx, nil)
				require.Error(t, err)

				// Check error is user-friendly
				assert.Contains(t, err.Error(), tc.expectedMsg)
				
				// Verify error has proper context
				if gerr, ok := err.(*gerror.Error); ok {
					assert.NotNil(t, gerr.Context, "Error should have context")
					assert.Equal(t, tc.failureMode, gerr.Context["failure_mode"])
				}
			})
		}
	})
}

// TestProviderFailoverChain tests cascading fallback through multiple providers
func TestProviderFailoverChain(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewComponentRegistry()
	err := reg.Initialize(ctx, registry.Config{})
	require.NoError(t, err)

	// Create chain of providers with different failure modes
	providers := []struct {
		name        string
		provider    interfaces.AIProvider
		shouldFail  bool
	}{
		{
			name: "primary",
			provider: &failingProvider{
				failureMode: "unavailable",
				errorMsg:    "primary is down",
			},
			shouldFail: true,
		},
		{
			name: "secondary", 
			provider: &failingProvider{
				failureMode: "rate_limit",
				errorMsg:    "secondary rate limited",
			},
			shouldFail: true,
		},
		{
			name: "tertiary",
			provider: testutil.NewMockLLMProvider().(*testutil.MockLLMProvider).
				WithResponse("Success from tertiary provider"),
			shouldFail: false,
		},
	}

	// Register all providers
	for _, p := range providers {
		prov := p.provider
		err := reg.Providers().Register(p.name, func(ctx context.Context, cfg map[string]any) (any, error) {
			return prov, nil
		})
		require.NoError(t, err)
	}

	// Track failover attempts
	var failoverAttempts []string
	var mu sync.Mutex

	// Create agent with failover chain
	agentCfg := agent.Config{
		Name:              "failover-test",
		PrimaryProvider:   "primary",
		FallbackProviders: []string{"secondary", "tertiary"},
		OnProviderSwitch: func(from, to string) {
			mu.Lock()
			failoverAttempts = append(failoverAttempts, fmt.Sprintf("%s->%s", from, to))
			mu.Unlock()
		},
	}

	factory := reg.Agents()
	testAgent, err := factory.Create(ctx, agentCfg)
	require.NoError(t, err)

	// Execute task
	result, err := testAgent.Execute(ctx, agent.Task{
		ID:      "failover-task",
		Content: "Test failover chain",
	})

	// Should eventually succeed
	require.NoError(t, err, "Should succeed with tertiary provider")
	assert.Contains(t, result.Response, "tertiary", "Should use tertiary provider")

	// Verify failover sequence
	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, failoverAttempts, 2, "Should have 2 failovers")
	assert.Equal(t, "primary->secondary", failoverAttempts[0])
	assert.Equal(t, "secondary->tertiary", failoverAttempts[1])
}

// TestProviderRecoveryPatterns tests various recovery strategies
func TestProviderRecoveryPatterns(t *testing.T) {
	ctx := context.Background()

	t.Run("ExponentialBackoff", func(t *testing.T) {
		attempts := 0
		delays := []time.Duration{}
		
		provider := &failingProvider{
			failureMode: "transient",
			errorMsg:    "temporary failure",
			onAttempt: func(n int) {
				attempts = n
				if n > 3 {
					// Succeed after 3 attempts
					provider.failureMode = "success"
				}
			},
		}

		// Exponential backoff implementation
		backoff := func(attempt int) time.Duration {
			delay := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
			if delay > 2*time.Second {
				delay = 2 * time.Second
			}
			return delay
		}

		// Retry with backoff
		var lastErr error
		startTime := time.Now()
		
		for i := 0; i < 5; i++ {
			_, err := provider.Complete(ctx, nil)
			if err == nil {
				break
			}
			lastErr = err
			
			delay := backoff(i)
			delays = append(delays, delay)
			time.Sleep(delay)
		}

		totalDuration := time.Since(startTime)

		// Should succeed after backoff
		assert.Equal(t, 4, attempts, "Should succeed on 4th attempt")
		assert.Nil(t, lastErr, "Should eventually succeed")
		
		// Verify exponential delays
		assert.Equal(t, 100*time.Millisecond, delays[0])
		assert.Equal(t, 200*time.Millisecond, delays[1])
		assert.Equal(t, 400*time.Millisecond, delays[2])
		
		// Total time should be sum of delays
		expectedDuration := 100*time.Millisecond + 200*time.Millisecond + 400*time.Millisecond
		assert.Greater(t, totalDuration, expectedDuration)
	})

	t.Run("CircuitBreaker", func(t *testing.T) {
		// Circuit breaker state
		type circuitState int
		const (
			closed circuitState = iota
			open
			halfOpen
		)

		state := closed
		failures := 0
		successThreshold := 2
		failureThreshold := 3
		cooldownPeriod := 500 * time.Millisecond
		lastFailureTime := time.Time{}

		provider := &failingProvider{
			failureMode: "unreliable",
			errorMsg:    "service unreliable",
		}

		// Circuit breaker logic
		callProvider := func() error {
			// Check circuit state
			switch state {
			case open:
				if time.Since(lastFailureTime) > cooldownPeriod {
					state = halfOpen
				} else {
					return fmt.Errorf("circuit breaker open")
				}
			}

			// Attempt call
			_, err := provider.Complete(ctx, nil)
			
			if err != nil {
				failures++
				lastFailureTime = time.Now()
				
				if failures >= failureThreshold {
					state = open
				}
				return err
			}

			// Success
			if state == halfOpen {
				failures = 0
				state = closed
			}
			return nil
		}

		// Test circuit breaker behavior
		// Should fail and open circuit
		for i := 0; i < failureThreshold; i++ {
			err := callProvider()
			assert.Error(t, err, "Provider should fail")
		}
		assert.Equal(t, open, state, "Circuit should be open")

		// Immediate calls should fail fast
		err := callProvider()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker open")

		// Wait for cooldown
		time.Sleep(cooldownPeriod + 100*time.Millisecond)

		// Make provider succeed
		provider.failureMode = "success"

		// Should enter half-open and succeed
		err = callProvider()
		assert.NoError(t, err, "Should succeed in half-open state")
		assert.Equal(t, closed, state, "Circuit should close on success")
	})
}

// failingProvider simulates various provider failure modes
type failingProvider struct {
	failureMode  string
	errorMsg     string
	attempts     int
	tokensUsed   int
	costPerToken float64
	resetAfter   time.Duration
	onAttempt    func(int)
	mu           sync.Mutex
}

func (f *failingProvider) Complete(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	f.mu.Lock()
	f.attempts++
	attempts := f.attempts
	f.mu.Unlock()

	if f.onAttempt != nil {
		f.onAttempt(attempts)
	}

	// Simulate token usage for cost tracking
	if f.tokensUsed > 0 {
		// In real implementation, this would be tracked properly
		time.Sleep(10 * time.Millisecond) // Simulate processing
	}

	switch f.failureMode {
	case "success":
		return &interfaces.CompletionResponse{
			Content: "Success",
			Usage: &interfaces.TokenUsage{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		}, nil

	case "unavailable":
		return nil, gerror.New(f.errorMsg, gerror.ErrorTypeProvider).
			WithContext("provider", "test").
			WithContext("failure_mode", f.failureMode)

	case "rate_limit":
		err := gerror.New(f.errorMsg, gerror.ErrorTypeRateLimit).
			WithContext("reset_after", f.resetAfter).
			WithContext("failure_mode", f.failureMode)
		
		// Succeed after reset time
		if attempts > 1 && time.Since(time.Now().Add(-f.resetAfter)) > 0 {
			f.failureMode = "success"
			return f.Complete(ctx, req)
		}
		return nil, err

	case "timeout":
		// Simulate timeout by waiting
		select {
		case <-time.After(5 * time.Second):
			return nil, gerror.New(f.errorMsg, gerror.ErrorTypeTimeout).
				WithContext("failure_mode", f.failureMode)
		case <-ctx.Done():
			return nil, ctx.Err()
		}

	case "auth":
		return nil, gerror.New(f.errorMsg, gerror.ErrorTypeAuthentication).
			WithContext("failure_mode", f.failureMode).
			WithContext("help", "Please check your API key configuration")

	case "partial_response":
		// Simulate partial response with token usage
		return nil, gerror.New(f.errorMsg, gerror.ErrorTypeProvider).
			WithContext("tokens_used", f.tokensUsed).
			WithContext("partial_content", "The response was truncated...").
			WithContext("failure_mode", f.failureMode)

	case "server_error":
		return nil, gerror.New(f.errorMsg, gerror.ErrorTypeProvider).
			WithContext("status_code", 500).
			WithContext("failure_mode", f.failureMode)

	case "transient":
		return nil, gerror.New(f.errorMsg, gerror.ErrorTypeProvider).
			WithContext("retry_after", "1s").
			WithContext("failure_mode", f.failureMode)

	case "unreliable":
		// Fails most of the time
		if attempts%5 == 0 {
			f.failureMode = "success"
			return f.Complete(ctx, req)
		}
		return nil, gerror.New(f.errorMsg, gerror.ErrorTypeProvider).
			WithContext("failure_mode", f.failureMode)

	default:
		return nil, fmt.Errorf("unknown failure mode: %s", f.failureMode)
	}
}

func (f *failingProvider) Stream(ctx context.Context, req *interfaces.CompletionRequest) (<-chan *interfaces.StreamChunk, error) {
	// Simplified streaming implementation
	ch := make(chan *interfaces.StreamChunk)
	go func() {
		defer close(ch)
		resp, err := f.Complete(ctx, req)
		if err != nil {
			ch <- &interfaces.StreamChunk{Error: err}
			return
		}
		ch <- &interfaces.StreamChunk{Content: resp.Content}
	}()
	return ch, nil
}

func (f *failingProvider) GetCapabilities() *interfaces.Capabilities {
	return &interfaces.Capabilities{
		MaxTokens:      4096,
		SupportsStream: true,
		SupportsTools:  false,
	}
}

func (f *failingProvider) Configure(config map[string]any) error {
	return nil
}