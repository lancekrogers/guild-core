package mocks

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/providers"
)

// MockLLMClient implements the providers.LLMClient interface for testing
type MockLLMClient struct {
	CompletionResponses map[string]providers.CompletionResponse
	DefaultResponse     providers.CompletionResponse
	Name                string
	ModelInfo           map[string]string
	Models              []string
	MaxTokens           int
	Error               error
	RequestHistory      []*providers.CompletionRequest
}

// NewMockLLMClient creates a new mock LLM client
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		CompletionResponses: make(map[string]providers.CompletionResponse),
		DefaultResponse: providers.CompletionResponse{
			Text:         `{"thoughts": "Task seems straightforward", "final_answer": "Task completed successfully"}`,
			TokensUsed:   10,
			TokensInput:  5,
			TokensOutput: 5,
			FinishReason: "stop",
			ModelUsed:    "mock-model",
		},
		Name: "mock-provider",
		ModelInfo: map[string]string{
			"name":    "mock-model",
			"version": "1.0",
		},
		Models:    []string{"mock-model"},
		MaxTokens: 4096,
	}
}

// WithError configures the mock client to return an error
func (m *MockLLMClient) WithError(err error) *MockLLMClient {
	m.Error = err
	return m
}

// WithResponse adds a response for a specific prompt prefix
func (m *MockLLMClient) WithResponse(promptPrefix string, response providers.CompletionResponse) *MockLLMClient {
	m.CompletionResponses[promptPrefix] = response
	return m
}

// WithDefaultResponse sets the default response
func (m *MockLLMClient) WithDefaultResponse(response providers.CompletionResponse) *MockLLMClient {
	m.DefaultResponse = response
	return m
}

// Complete implements the LLMClient.Complete method
func (m *MockLLMClient) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	// Store the request for verification
	m.RequestHistory = append(m.RequestHistory, req)

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return nil, m.Error
	}

	// Try to find a matching response
	for prefix, response := range m.CompletionResponses {
		if len(req.Prompt) >= len(prefix) && req.Prompt[:len(prefix)] == prefix {
			return &response, nil
		}
	}

	// Return default response
	return &m.DefaultResponse, nil
}

// GetName implements the LLMClient.GetName method
func (m *MockLLMClient) GetName() string {
	return m.Name
}

// GetModelInfo implements the LLMClient.GetModelInfo method
func (m *MockLLMClient) GetModelInfo() map[string]string {
	return m.ModelInfo
}

// GetModelList implements the LLMClient.GetModelList method
func (m *MockLLMClient) GetModelList(ctx context.Context) ([]string, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue execution
	}

	// Return error if configured
	if m.Error != nil {
		return nil, m.Error
	}

	return m.Models, nil
}

// GetMaxTokens implements the LLMClient.GetMaxTokens method
func (m *MockLLMClient) GetMaxTokens() int {
	return m.MaxTokens
}