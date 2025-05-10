package mocks

import (
	"context"
	"fmt"
	"strings"

	"github.com/blockhead-consulting/guild/pkg/providers/interfaces"
)

// MockLLMClient implements the LLMClient interface for testing
type MockLLMClient struct {
	// Name of the provider
	Name string

	// Model information
	ModelInfo map[string]string

	// Available models
	AvailableModels []string

	// Max tokens for the model
	MaxTokenLimit int

	// Completion responses mapped by prompt prefixes
	CompletionResponses map[string]interfaces.CompletionResponse

	// Default response when no match is found
	DefaultResponse interfaces.CompletionResponse

	// Error to return (if not nil)
	Error error

	// Keep track of requests for verification
	CompletionRequests []*interfaces.CompletionRequest
}

// NewMockLLMClient creates a new mock LLM client with default values
func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		Name:                "mock_provider",
		ModelInfo:           map[string]string{"model": "mock-model", "version": "1.0"},
		AvailableModels:     []string{"mock-model-small", "mock-model-medium", "mock-model-large"},
		MaxTokenLimit:       4096,
		CompletionResponses: make(map[string]interfaces.CompletionResponse),
		DefaultResponse: interfaces.CompletionResponse{
			Text:         "This is a mock response",
			TokensUsed:   10,
			TokensInput:  5,
			TokensOutput: 5,
			FinishReason: "stop",
			ModelUsed:    "mock-model",
		},
		CompletionRequests: make([]*interfaces.CompletionRequest, 0),
	}
}

// WithName sets the name of the mock provider
func (m *MockLLMClient) WithName(name string) *MockLLMClient {
	m.Name = name
	return m
}

// WithError sets an error to be returned by the mock client
func (m *MockLLMClient) WithError(err error) *MockLLMClient {
	m.Error = err
	return m
}

// WithModelInfo sets the model info for the mock client
func (m *MockLLMClient) WithModelInfo(info map[string]string) *MockLLMClient {
	m.ModelInfo = info
	return m
}

// WithMaxTokens sets the max tokens for the mock client
func (m *MockLLMClient) WithMaxTokens(tokens int) *MockLLMClient {
	m.MaxTokenLimit = tokens
	return m
}

// WithDefaultResponse sets the default response for the mock client
func (m *MockLLMClient) WithDefaultResponse(resp interfaces.CompletionResponse) *MockLLMClient {
	m.DefaultResponse = resp
	return m
}

// AddResponse adds a response for a specific prompt prefix
func (m *MockLLMClient) AddResponse(promptPrefix string, response interfaces.CompletionResponse) *MockLLMClient {
	m.CompletionResponses[promptPrefix] = response
	return m
}

// Complete implements the LLMClient interface
func (m *MockLLMClient) Complete(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	// Store the request for later verification
	m.CompletionRequests = append(m.CompletionRequests, req)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue with normal processing
	}

	// Return error if set
	if m.Error != nil {
		return nil, m.Error
	}

	// Try to find a matching response based on prompt prefix
	for prefix, response := range m.CompletionResponses {
		if strings.HasPrefix(req.Prompt, prefix) {
			return &response, nil
		}
	}

	// Use default response if no match found
	response := m.DefaultResponse
	return &response, nil
}

// GetName implements the LLMClient interface
func (m *MockLLMClient) GetName() string {
	return m.Name
}

// GetModelInfo implements the LLMClient interface
func (m *MockLLMClient) GetModelInfo() map[string]string {
	return m.ModelInfo
}

// GetModelList implements the LLMClient interface
func (m *MockLLMClient) GetModelList(ctx context.Context) ([]string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue with normal processing
	}

	// Return error if set
	if m.Error != nil {
		return nil, m.Error
	}

	return m.AvailableModels, nil
}

// GetMaxTokens implements the LLMClient interface
func (m *MockLLMClient) GetMaxTokens() int {
	return m.MaxTokenLimit
}

// CreateEmbedding implements the LLMClient interface
func (m *MockLLMClient) CreateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue with normal processing
	}

	// Return error if set
	if m.Error != nil {
		return nil, m.Error
	}

	// Return mock embedding
	return &interfaces.EmbeddingResponse{
		Embedding:  make([]float32, 1536), // Default dimension
		Dimensions: 1536,
		Model:      "mock-embedding-model",
		TokensUsed: len(req.Text) / 4, // Rough estimation
	}, nil
}

// CreateEmbeddings implements the LLMClient interface
func (m *MockLLMClient) CreateEmbeddings(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Continue with normal processing
	}

	// Return error if set
	if m.Error != nil {
		return nil, m.Error
	}

	// Return mock embeddings
	embeddings := make([][]float32, len(req.Texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, 1536)
	}

	return &interfaces.EmbeddingResponse{
		Embeddings: embeddings,
		Dimensions: 1536,
		Model:      "mock-embedding-model",
		TokensUsed: len(strings.Join(req.Texts, " ")) / 4, // Rough estimation
	}, nil
}

// GetEmbeddingDimension implements the LLMClient interface
func (m *MockLLMClient) GetEmbeddingDimension(model string) int {
	return 1536 // Default OpenAI embedding dimension
}

// GetLastRequest returns the last request received by the mock client
func (m *MockLLMClient) GetLastRequest() (*interfaces.CompletionRequest, error) {
	if len(m.CompletionRequests) == 0 {
		return nil, fmt.Errorf("no requests received")
	}
	return m.CompletionRequests[len(m.CompletionRequests)-1], nil
}

// ClearRequests clears the stored requests
func (m *MockLLMClient) ClearRequests() {
	m.CompletionRequests = make([]*interfaces.CompletionRequest, 0)
}