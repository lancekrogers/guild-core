package openai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/blockhead-consulting/guild/pkg/providers/interfaces"
	"github.com/blockhead-consulting/guild/pkg/providers/openai"
)

// TestOpenAIClientImplementation tests that the OpenAI client implements the LLMClient interface
func TestOpenAIClientImplementation(t *testing.T) {
	var _ providers.LLMClient = &openai.Client{}
}

// TestNewClient tests the creation of a new OpenAI client
func TestNewClient(t *testing.T) {
	// Test with empty API key
	_, err := openai.NewClient("")
	if err == nil {
		t.Error("expected error with empty API key, got nil")
	}

	// Test with valid API key
	client, err := openai.NewClient("test-api-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// Test with model option
	client, err = openai.NewClient("test-api-key", openai.WithModel("gpt-4"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info := client.GetModelInfo()
	if info["name"] != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got '%s'", info["name"])
	}

	// Test with timeout option
	timeout := 60 * time.Second
	client, err = openai.NewClient(
		"test-api-key",
		openai.WithTimeout(timeout),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Timeout is a client setting, so we can't easily check it
}

// setupMockServer creates a mock HTTP server for testing
func setupMockServer(t *testing.T) *httptest.Server {
	// Read the mock response from testdata
	respData, err := os.ReadFile("../testdata/openai_completion_response.json")
	if err != nil {
		t.Fatalf("Failed to read mock response: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify auth header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got '%s'", authHeader)
		}

		// Verify content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		// Write the mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respData)
	}))

	return server
}

// TestComplete tests the Complete method
func TestComplete(t *testing.T) {
	// Setup mock server
	server := setupMockServer(t)
	defer server.Close()

	// Create client with mock server URL
	client, err := openai.NewClient(
		"test-api-key",
		openai.WithEndpoint(server.URL), // Add this option to the client
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create completion request
	req := &interfaces.CompletionRequest{
		Prompt:      "Test prompt",
		MaxTokens:   100,
		Temperature: 0.7,
		StopTokens:  []string{"\n"},
		Options: map[string]string{
			"system": "You are a helpful assistant.",
		},
	}

	// Call Complete
	resp, err := client.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Verify response
	expectedText := "This is a test response from the mock OpenAI API."
	if resp.Text != expectedText {
		t.Errorf("Expected text '%s', got '%s'", expectedText, resp.Text)
	}

	if resp.TokensUsed != 30 {
		t.Errorf("Expected 30 tokens used, got %d", resp.TokensUsed)
	}

	if resp.TokensInput != 10 {
		t.Errorf("Expected 10 input tokens, got %d", resp.TokensInput)
	}

	if resp.TokensOutput != 20 {
		t.Errorf("Expected 20 output tokens, got %d", resp.TokensOutput)
	}

	if resp.FinishReason != "stop" {
		t.Errorf("Expected finish reason 'stop', got '%s'", resp.FinishReason)
	}

	if resp.ModelUsed != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", resp.ModelUsed)
	}
}

// TestGetName tests the GetName method
func TestGetName(t *testing.T) {
	client, err := openai.NewClient("test-api-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	name := client.GetName()
	if name != "openai" {
		t.Errorf("Expected name 'openai', got '%s'", name)
	}
}

// TestGetModelInfo tests the GetModelInfo method
func TestGetModelInfo(t *testing.T) {
	client, err := openai.NewClient("test-api-key", openai.WithModel("gpt-4"))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	info := client.GetModelInfo()

	if info["name"] != "gpt-4" {
		t.Errorf("Expected model name 'gpt-4', got '%s'", info["name"])
	}

	if info["provider"] != "OpenAI" {
		t.Errorf("Expected provider 'OpenAI', got '%s'", info["provider"])
	}

	if _, exists := info["capabilities"]; !exists {
		t.Error("Expected capabilities info, not found")
	}
}

// TestGetModelList tests the GetModelList method
func TestGetModelList(t *testing.T) {
	client, err := openai.NewClient("test-api-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	models, err := client.GetModelList(context.Background())
	if err != nil {
		t.Fatalf("GetModelList failed: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected at least one model, got empty list")
	}

	// Check for presence of common models
	expectedModels := []string{"gpt-4", "gpt-3.5-turbo"}
	for _, expected := range expectedModels {
		found := false
		for _, model := range models {
			if model == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model '%s' not found in list", expected)
		}
	}
}

// TestGetMaxTokens tests the GetMaxTokens method
func TestGetMaxTokens(t *testing.T) {
	testCases := []struct {
		name        string
		model       string
		expectedMax int
	}{
		{
			name:        "gpt-4",
			model:       "gpt-4",
			expectedMax: 8192,
		},
		{
			name:        "gpt-4-turbo",
			model:       "gpt-4-turbo",
			expectedMax: 128000,
		},
		{
			name:        "gpt-3.5-turbo",
			model:       "gpt-3.5-turbo",
			expectedMax: 16385,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := openai.NewClient("test-api-key", openai.WithModel(tc.model))
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			max := client.GetMaxTokens()
			if max != tc.expectedMax {
				t.Errorf("Expected max tokens %d, got %d", tc.expectedMax, max)
			}
		})
	}
}

// TestCompleteWithError tests error handling in the Complete method
func TestCompleteWithError(t *testing.T) {
	// Setup mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Test error message",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	// Create client with mock server URL
	client, err := openai.NewClient(
		"test-api-key",
		openai.WithEndpoint(server.URL),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Call Complete
	_, err = client.Complete(context.Background(), &interfaces.CompletionRequest{
		Prompt: "Test prompt",
	})

	// Verify error
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestCompleteWithContextCancellation tests context cancellation handling
func TestCompleteWithContextCancellation(t *testing.T) {
	// Setup mock server with a delay
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep to simulate a slow response
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	client, err := openai.NewClient(
		"test-api-key",
		openai.WithEndpoint(server.URL),
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Create a context and cancel it immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Call Complete with canceled context
	_, err = client.Complete(ctx, &interfaces.CompletionRequest{
		Prompt: "Test prompt",
	})

	// Verify error
	if err == nil {
		t.Error("Expected error due to canceled context, got nil")
	}
}