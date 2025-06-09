package agent

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test Execute method with different scenarios
func TestWorkerAgent_Execute_Comprehensive(t *testing.T) {
	tests := []struct {
		name        string
		setupAgent  func() *WorkerAgent
		request     string
		wantErr     bool
		errContains string
	}{
		{
			name: "successful execution with mock LLM",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:        "test-agent",
					Name:      "Test Agent",
					LLMClient: &mockLLMClient{response: "test response"},
				}
			},
			request: "test request",
			wantErr: false,
		},
		{
			name: "execution with empty request",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:        "test-agent",
					Name:      "Test Agent",
					LLMClient: &mockLLMClient{response: "response to empty"},
				}
			},
			request: "",
			wantErr: false, // Empty request should still work
		},
		{
			name: "execution with very long request",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:        "test-agent",
					Name:      "Test Agent",
					LLMClient: &mockLLMClient{response: "response to long request"},
				}
			},
			request: generateLongString(10000), // 10k characters
			wantErr: false,
		},
		{
			name: "LLM client returns error",
			setupAgent: func() *WorkerAgent {
				return &WorkerAgent{
					ID:        "test-agent",
					Name:      "Test Agent",
					LLMClient: &mockLLMClient{shouldError: true, errorMsg: "LLM error"},
				}
			},
			request:     "test request",
			wantErr:     true,
			errContains: "LLM error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := tt.setupAgent()
			ctx := context.Background()

			response, err := agent.Execute(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, response)
			}
		})
	}
}

// Test Execute with context cancellation
func TestWorkerAgent_Execute_ContextCancellation(t *testing.T) {
	agent := &WorkerAgent{
		ID:        "test-agent",
		Name:      "Test Agent",
		LLMClient: &mockLLMClient{delay: 100 * time.Millisecond, response: "delayed response"},
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := agent.Execute(ctx, "test request")
	duration := time.Since(start)

	// Should error due to context deadline
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	
	// Should not take the full delay time
	assert.Less(t, duration, 50*time.Millisecond)
}

// Test Execute with different context values
func TestWorkerAgent_Execute_ContextValues(t *testing.T) {
	agent := &WorkerAgent{
		ID:        "test-agent",
		Name:      "Test Agent",
		LLMClient: &mockLLMClient{response: "context test response"},
	}

	// Test with context containing values
	type contextKey string
	key := contextKey("test-key")
	ctx := context.WithValue(context.Background(), key, "test-value")

	response, err := agent.Execute(ctx, "test request")
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
}

// Test concurrent execution
func TestWorkerAgent_Execute_Concurrent(t *testing.T) {
	agent := &WorkerAgent{
		ID:        "test-agent",
		Name:      "Test Agent",
		LLMClient: &mockLLMClient{response: "concurrent response"},
	}

	const numGoroutines = 10
	responses := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)
	
	// Channel to coordinate goroutines
	done := make(chan int, numGoroutines)

	// Launch multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			ctx := context.Background()
			resp, err := agent.Execute(ctx, "concurrent request")
			responses[index] = resp
			errors[index] = err
			done <- index
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all executions succeeded
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i], "Goroutine %d should not error", i)
		assert.NotEmpty(t, responses[i], "Goroutine %d should have response", i)
	}
}

// Enhanced mock LLM client with more features
type mockLLMClient struct {
	response    string
	shouldError bool
	errorMsg    string
	delay       time.Duration
	callCount   int
}

func (m *mockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	m.callCount++
	
	// Simulate delay if specified
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
			// Continue
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	
	if m.shouldError {
		if m.errorMsg != "" {
			return "", &mockError{msg: m.errorMsg}
		}
		return "", &mockError{msg: "LLM error"}
	}
	
	return m.response, nil
}

// mockError implements error interface
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// Helper function to generate long strings for testing
func generateLongString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

// Test Execute method error paths
func TestWorkerAgent_Execute_ErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		agent      *WorkerAgent
		request    string
		expectErr  bool
		errPattern string
	}{
		{
			name: "nil LLM client",
			agent: &WorkerAgent{
				ID:        "test-agent",
				Name:      "Test Agent",
				LLMClient: nil,
			},
			request:    "test request",
			expectErr:  true,
			errPattern: "no LLM client configured",
		},
		{
			name: "valid execution",
			agent: &WorkerAgent{
				ID:        "test-agent",
				Name:      "Test Agent",
				LLMClient: &mockLLMClient{response: "success"},
			},
			request:   "test request",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			response, err := tt.agent.Execute(ctx, tt.request)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errPattern != "" {
					assert.Contains(t, err.Error(), tt.errPattern)
				}
				assert.Empty(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, response)
			}
		})
	}
}