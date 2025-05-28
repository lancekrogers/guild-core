package testing_test

import (
	"context"
	"os"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/providers/base"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/pkg/providers/openai"
	providertesting "github.com/guild-ventures/guild-core/pkg/providers/testing"
)

// Example: Testing OpenAI provider with mock server
func ExampleTestOpenAIWithMockServer(t *testing.T) {
	// 1. Create mock HTTP server
	mock := providertesting.NewMockHTTPServer()
	defer mock.Close()

	// 2. Create OpenAI client pointing to mock server
	client := openai.NewClient("test-api-key")
	
	// Override the base URL to point to mock server
	// This would require adding a method to override the base URL
	// For now, we'll create it directly:
	client = &openai.Client{
		OpenAICompatibleProvider: base.NewOpenAICompatibleProvider(
			"openai",
			"test-api-key", 
			mock.URL + "/v1",
			nil,
			interfaces.ProviderCapabilities{
				MaxTokens:      128000,
				ContextWindow:  128000,
				SupportsVision: true,
				SupportsTools:  true,
				SupportsStream: true,
				Models: []interfaces.ModelInfo{
					{
						ID:            "gpt-4.1-mini",
						Name:          "GPT-4.1 Mini",
						ContextWindow: 1000000,
						MaxOutput:     32768,
						InputCost:     1.0,
						OutputCost:    4.0,
					},
				},
			},
		),
	}

	// 3. Test chat completion
	ctx := context.Background()
	req := interfaces.ChatRequest{
		Model: "gpt-4.1-mini",
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	// 4. Verify response
	if len(resp.Choices) == 0 {
		t.Fatal("No choices in response")
	}

	// 5. Verify request was made correctly
	lastReq := mock.GetLastRequest()
	if lastReq.Path != "/v1/chat/completions" {
		t.Errorf("Wrong path: %s", lastReq.Path)
	}
}

// Example: Testing with different error scenarios
func ExampleTestErrorScenarios(t *testing.T) {
	testCases := []struct {
		name          string
		setupMock     func(*providertesting.MockHTTPServer)
		expectedError string
	}{
		{
			name: "AuthError",
			setupMock: func(m *providertesting.MockHTTPServer) {
				// Mock will return 401 for empty auth
			},
			expectedError: "401",
		},
		{
			name: "RateLimitError", 
			setupMock: func(m *providertesting.MockHTTPServer) {
				// Would need to add rate limit response to mock
			},
			expectedError: "429",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := providertesting.NewMockHTTPServer()
			defer mock.Close()

			if tc.setupMock != nil {
				tc.setupMock(mock)
			}

			// Test with appropriate client setup...
		})
	}
}

// Example: Integration test pattern (with environment variable)
func ExampleIntegrationTest(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Check for API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	// Create real client
	client := openai.NewClient(apiKey)

	// Run limited test with real API
	ctx := context.Background()
	req := interfaces.ChatRequest{
		Model: openai.GPT41Mini,
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Say 'test' and nothing else"},
		},
		MaxTokens:   10,
		Temperature: 0,
	}

	resp, err := client.ChatCompletion(ctx, req)
	if err != nil {
		t.Fatalf("Real API call failed: %v", err)
	}

	t.Logf("Real API response: %+v", resp)
}