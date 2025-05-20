package providers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/providers/mocks"
)

// TestLLMClientImplementation tests that the mock client properly implements the LLMClient interface
func TestLLMClientImplementation(t *testing.T) {
	var _ providers.LLMClient = &mocks.MockLLMClient{}
}

// TestComplete tests the Complete method of the LLMClient interface
func TestComplete(t *testing.T) {
	testCases := []struct {
		name          string
		prompt        string
		mockSetup     func(*mocks.MockLLMClient)
		ctxSetup      func() (context.Context, context.CancelFunc)
		expectedText  string
		expectError   bool
		errorContains string
	}{
		{
			name:   "basic completion",
			prompt: "Hello, world!",
			mockSetup: func(m *mocks.MockLLMClient) {
				m.WithDefaultResponse(providers.CompletionResponse{
					Text:         "Hi there!",
					TokensUsed:   10,
					TokensInput:  5,
					TokensOutput: 5,
					FinishReason: "stop",
					ModelUsed:    "mock-model",
				})
			},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			expectedText: "Hi there!",
			expectError:  false,
		},
		{
			name:   "prefix-specific response",
			prompt: "Translate: Hello in French",
			mockSetup: func(m *mocks.MockLLMClient) {
				m.AddResponse("Translate:", providers.CompletionResponse{
					Text:         "Bonjour",
					TokensUsed:   10,
					TokensInput:  5,
					TokensOutput: 5,
					FinishReason: "stop",
					ModelUsed:    "mock-model",
				})
			},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			expectedText: "Bonjour",
			expectError:  false,
		},
		{
			name:   "error response",
			prompt: "Will fail",
			mockSetup: func(m *mocks.MockLLMClient) {
				m.WithError(errors.New("mock error"))
			},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			expectedText:  "",
			expectError:   true,
			errorContains: "mock error",
		},
		{
			name:   "canceled context",
			prompt: "Will be canceled",
			mockSetup: func(m *mocks.MockLLMClient) {
				// Default setup is fine
			},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			expectedText:  "",
			expectError:   true,
			errorContains: "canceled",
		},
		{
			name:   "timeout context",
			prompt: "Will timeout",
			mockSetup: func(m *mocks.MockLLMClient) {
				// Default setup is fine
			},
			ctxSetup: func() (context.Context, context.CancelFunc) {
				// Create a context that's already timed out
				return context.WithTimeout(context.Background(), -1*time.Millisecond)
			},
			expectedText:  "",
			expectError:   true,
			errorContains: "deadline",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock client
			mockClient := mocks.NewMockLLMClient()
			tc.mockSetup(mockClient)

			// Setup context
			ctx, cancel := tc.ctxSetup()
			defer cancel()

			// Create request
			req := &providers.CompletionRequest{
				Prompt:      tc.prompt,
				MaxTokens:   100,
				Temperature: 0.7,
			}

			// Call Complete
			resp, err := mockClient.Complete(ctx, req)

			// Check for expected error
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tc.errorContains)
				}
				if tc.errorContains != "" && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					if err.Error() != tc.errorContains {
						t.Fatalf("expected error containing '%s', got '%s'", tc.errorContains, err.Error())
					}
				}
				return
			}

			// Check for unexpected error
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check response
			if resp.Text != tc.expectedText {
				t.Errorf("expected text '%s', got '%s'", tc.expectedText, resp.Text)
			}

			// Verify request was stored
			lastReq, err := mockClient.GetLastRequest()
			if err != nil {
				t.Fatalf("failed to get last request: %v", err)
			}

			if lastReq.Prompt != tc.prompt {
				t.Errorf("expected stored prompt '%s', got '%s'", tc.prompt, lastReq.Prompt)
			}
		})
	}
}

// TestGetName tests the GetName method of the LLMClient interface
func TestGetName(t *testing.T) {
	mockClient := mocks.NewMockLLMClient().WithName("test-provider")
	name := mockClient.GetName()
	
	if name != "test-provider" {
		t.Errorf("expected name 'test-provider', got '%s'", name)
	}
}

// TestGetModelInfo tests the GetModelInfo method of the LLMClient interface
func TestGetModelInfo(t *testing.T) {
	expectedInfo := map[string]string{
		"model":   "test-model",
		"version": "2.0",
	}
	
	mockClient := mocks.NewMockLLMClient().WithModelInfo(expectedInfo)
	info := mockClient.GetModelInfo()
	
	if len(info) != len(expectedInfo) {
		t.Errorf("expected %d info items, got %d", len(expectedInfo), len(info))
	}
	
	for k, v := range expectedInfo {
		if info[k] != v {
			t.Errorf("expected info['%s'] = '%s', got '%s'", k, v, info[k])
		}
	}
}

// TestGetModelList tests the GetModelList method of the LLMClient interface
func TestGetModelList(t *testing.T) {
	expectedModels := []string{"model-a", "model-b", "model-c"}
	
	mockClient := mocks.NewMockLLMClient()
	mockClient.AvailableModels = expectedModels
	
	models, err := mockClient.GetModelList(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	if len(models) != len(expectedModels) {
		t.Errorf("expected %d models, got %d", len(expectedModels), len(models))
	}
	
	for i, model := range expectedModels {
		if models[i] != model {
			t.Errorf("expected model[%d] = '%s', got '%s'", i, model, models[i])
		}
	}
	
	// Test with error
	mockClient.WithError(errors.New("model list error"))
	
	_, err = mockClient.GetModelList(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	
	if err.Error() != "model list error" {
		t.Errorf("expected error 'model list error', got '%s'", err.Error())
	}
	
	// Test with canceled context
	mockClient = mocks.NewMockLLMClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	_, err = mockClient.GetModelList(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context, got nil")
	}
	
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got '%s'", err.Error())
	}
}

// TestGetMaxTokens tests the GetMaxTokens method of the LLMClient interface
func TestGetMaxTokens(t *testing.T) {
	expectedTokens := 8192
	
	mockClient := mocks.NewMockLLMClient().WithMaxTokens(expectedTokens)
	tokens := mockClient.GetMaxTokens()
	
	if tokens != expectedTokens {
		t.Errorf("expected %d tokens, got %d", expectedTokens, tokens)
	}
}