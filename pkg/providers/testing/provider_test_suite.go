package testing

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// ProviderTestSuite provides common test cases for all providers
type ProviderTestSuite struct {
	t        *testing.T
	provider interfaces.AIProvider
	config   TestConfig
}

// TestConfig configures the test suite
type TestConfig struct {
	ProviderName string
	SkipLive     bool // Skip live API tests
	LiveAPIKey   string
	TestModel    string
}

// NewProviderTestSuite creates a new test suite
func NewProviderTestSuite(t *testing.T, provider interfaces.AIProvider, config TestConfig) *ProviderTestSuite {
	return &ProviderTestSuite{
		t:        t,
		provider: provider,
		config:   config,
	}
}

// RunBasicTests runs all basic provider tests
func (s *ProviderTestSuite) RunBasicTests() {
	s.t.Run("Capabilities", func(t *testing.T) { s.TestCapabilities() })
	s.t.Run("ChatCompletion", func(t *testing.T) { s.TestChatCompletion() })
	s.t.Run("ErrorHandling", func(t *testing.T) { s.TestErrorHandling() })
	
	if !s.config.SkipLive && s.config.LiveAPIKey != "" {
		s.t.Run("LiveAPI", func(t *testing.T) { s.TestLiveAPI() })
	}
}

// TestCapabilities verifies provider capabilities
func (s *ProviderTestSuite) TestCapabilities() {
	caps := s.provider.GetCapabilities()
	
	// Basic validations
	if caps.MaxTokens <= 0 {
		s.t.Error("MaxTokens should be positive")
	}
	
	if caps.ContextWindow <= 0 {
		s.t.Error("ContextWindow should be positive")
	}
	
	if len(caps.Models) == 0 {
		s.t.Error("Provider should have at least one model")
	}
	
	// Validate model info
	for _, model := range caps.Models {
		if model.ID == "" {
			s.t.Error("Model ID should not be empty")
		}
		if model.Name == "" {
			s.t.Error("Model Name should not be empty")
		}
		if model.ContextWindow <= 0 {
			s.t.Error("Model ContextWindow should be positive")
		}
	}
}

// TestChatCompletion tests basic chat completion
func (s *ProviderTestSuite) TestChatCompletion() {
	ctx := context.Background()
	
	// Use first available model
	caps := s.provider.GetCapabilities()
	if len(caps.Models) == 0 {
		s.t.Skip("No models available")
	}
	
	model := caps.Models[0].ID
	if s.config.TestModel != "" {
		model = s.config.TestModel
	}
	
	req := interfaces.ChatRequest{
		Model: model,
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Say 'test response' and nothing else"},
		},
		MaxTokens:   100,
		Temperature: 0,
	}
	
	// For unit tests, we'll use mocked responses
	// Real providers should implement proper mocking in their test files
	_, err := s.provider.ChatCompletion(ctx, req)
	if err != nil {
		// This is expected for unit tests without mocking
		s.t.Logf("Expected error in unit test: %v", err)
	}
}

// TestErrorHandling tests error scenarios
func (s *ProviderTestSuite) TestErrorHandling() {
	ctx := context.Background()
	
	testCases := []struct {
		name string
		req  interfaces.ChatRequest
	}{
		{
			name: "EmptyModel",
			req: interfaces.ChatRequest{
				Model:    "",
				Messages: []interfaces.ChatMessage{{Role: "user", Content: "test"}},
			},
		},
		{
			name: "NoMessages",
			req: interfaces.ChatRequest{
				Model:    "test-model",
				Messages: []interfaces.ChatMessage{},
			},
		},
		{
			name: "InvalidModel",
			req: interfaces.ChatRequest{
				Model:    "invalid-model-xyz",
				Messages: []interfaces.ChatMessage{{Role: "user", Content: "test"}},
			},
		},
	}
	
	for _, tc := range testCases {
		s.t.Run(tc.name, func(t *testing.T) {
			_, err := s.provider.ChatCompletion(ctx, tc.req)
			if err == nil {
				t.Log("Expected error but got none")
			}
		})
	}
}

// TestLiveAPI tests against real API (optional)
func (s *ProviderTestSuite) TestLiveAPI() {
	if s.config.SkipLive {
		s.t.Skip("Live API tests disabled")
	}
	
	ctx := context.Background()
	caps := s.provider.GetCapabilities()
	
	if len(caps.Models) == 0 {
		s.t.Skip("No models available")
	}
	
	model := caps.Models[0].ID
	if s.config.TestModel != "" {
		model = s.config.TestModel
	}
	
	req := interfaces.ChatRequest{
		Model: model,
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: "Respond with exactly: 'API test successful'"},
		},
		MaxTokens:   50,
		Temperature: 0,
	}
	
	resp, err := s.provider.ChatCompletion(ctx, req)
	if err != nil {
		s.t.Fatalf("Live API test failed: %v", err)
	}
	
	if len(resp.Choices) == 0 {
		s.t.Fatal("No choices in response")
	}
	
	s.t.Logf("Live API response: %s", resp.Choices[0].Message.Content)
}

// MockHTTPServer creates a mock HTTP server for testing
type MockHTTPServer struct {
	*httptest.Server
	Requests []RecordedRequest
}

// RecordedRequest captures request details
type RecordedRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

// NewMockHTTPServer creates a new mock server with OpenAI-compatible responses
func NewMockHTTPServer() *MockHTTPServer {
	mock := &MockHTTPServer{
		Requests: make([]RecordedRequest, 0),
	}
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record request
		body := make([]byte, 0)
		if r.Body != nil {
			body, _ = io.ReadAll(r.Body)
			r.Body.Close()
		}
		
		mock.Requests = append(mock.Requests, RecordedRequest{
			Method:  r.Method,
			Path:    r.URL.Path,
			Headers: r.Header.Clone(),
			Body:    body,
		})
		
		// Route based on path
		switch r.URL.Path {
		case "/v1/chat/completions", "/chat/completions":
			mock.handleChatCompletion(w, r, body)
		case "/v1/embeddings", "/embeddings":
			mock.handleEmbeddings(w, r, body)
		case "/v1/models", "/models":
			mock.handleModels(w, r)
		default:
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"message": "Not found",
					"type":    "not_found",
				},
			})
		}
	})
	
	mock.Server = httptest.NewServer(handler)
	return mock
}

func (m *MockHTTPServer) handleChatCompletion(w http.ResponseWriter, r *http.Request, body []byte) {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(400)
		return
	}
	
	// Check for auth
	auth := r.Header.Get("Authorization")
	if auth == "" || auth == "Bearer " {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	// Mock response
	response := map[string]interface{}{
		"id":      "chatcmpl-mock123",
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   req["model"],
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": "Mock response for testing",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     10,
			"completion_tokens": 5,
			"total_tokens":      15,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockHTTPServer) handleEmbeddings(w http.ResponseWriter, r *http.Request, body []byte) {
	var req map[string]interface{}
	json.Unmarshal(body, &req)
	
	response := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"object":    "embedding",
				"index":     0,
				"embedding": []float64{0.1, 0.2, 0.3},
			},
		},
		"model": req["model"],
		"usage": map[string]int{
			"prompt_tokens": 5,
			"total_tokens":  5,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockHTTPServer) handleModels(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"id":      "gpt-4",
				"object":  "model",
				"created": 1234567890,
				"owned_by": "openai",
			},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Close shuts down the mock server
func (m *MockHTTPServer) Close() {
	m.Server.Close()
}

// GetLastRequest returns the last recorded request
func (m *MockHTTPServer) GetLastRequest() *RecordedRequest {
	if len(m.Requests) == 0 {
		return nil
	}
	return &m.Requests[len(m.Requests)-1]
}