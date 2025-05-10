package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/blockhead-consulting/guild/pkg/providers/interfaces"
)

const (
	defaultAPI     = "https://api.anthropic.com/v1/messages"
	defaultModel   = "claude-3-opus-20240229"
	defaultTimeout = 30 * time.Second
)

// AnthropicClient implements the LLMClient interface for Anthropic's Claude models
type AnthropicClient struct {
	apiKey     string
	apiURL     string
	modelName  string
	httpClient *http.Client
	maxTokens  int
}

// Anthropic API request structure
type anthropicRequest struct {
	Model       string               `json:"model"`
	Messages    []anthropicMessage   `json:"messages"`
	MaxTokens   int                  `json:"max_tokens"`
	Temperature float64              `json:"temperature,omitempty"`
	Stop        []string             `json:"stop_sequences,omitempty"`
	System      string               `json:"system,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Anthropic API response structure
type anthropicResponse struct {
	Content []anthropicContent `json:"content"`
	Model   string             `json:"model"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	StopReason string `json:"stop_reason"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewClient creates a new Anthropic client
func NewClient(apiKey string, opts ...ClientOption) (*AnthropicClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}

	client := &AnthropicClient{
		apiKey:     apiKey,
		apiURL:     defaultAPI,
		modelName:  defaultModel,
		httpClient: &http.Client{Timeout: defaultTimeout},
		maxTokens:  200000, // Claude-3 Opus context window
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// ClientOption defines a function that configures the Anthropic client
type ClientOption func(*AnthropicClient)

// WithModel sets the model to use
func WithModel(model string) ClientOption {
	return func(c *AnthropicClient) {
		c.modelName = model
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *AnthropicClient) {
		c.httpClient.Timeout = timeout
	}
}

// Complete implements the LLMClient interface
func (c *AnthropicClient) Complete(ctx context.Context, req *interfaces.CompletionRequest) (*interfaces.CompletionResponse, error) {
	// Build Anthropic request
	prompt := req.Prompt
	
	// Simple prompt conversion (in practice you'd use a more sophisticated conversion)
	messages := []anthropicMessage{
		{Role: "user", Content: prompt},
	}

	aReq := anthropicRequest{
		Model:       c.modelName,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stop:        req.StopTokens,
	}

	// Set system prompt if provided in options
	if systemPrompt, ok := req.Options["system"]; ok {
		aReq.System = systemPrompt
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(aReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.apiURL,
		strings.NewReader(string(reqBody)),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Api-Key", c.apiKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var aResp anthropicResponse
	if err := json.Unmarshal(body, &aResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Extract text from content blocks
	var combinedText string
	for _, content := range aResp.Content {
		if content.Type == "text" {
			combinedText += content.Text
		}
	}

	// Convert to generic response
	return &interfaces.CompletionResponse{
		Text:         combinedText,
		TokensInput:  aResp.Usage.InputTokens,
		TokensOutput: aResp.Usage.OutputTokens,
		TokensUsed:   aResp.Usage.InputTokens + aResp.Usage.OutputTokens,
		FinishReason: aResp.StopReason,
		ModelUsed:    aResp.Model,
	}, nil
}

// GetName returns the provider name
func (c *AnthropicClient) GetName() string {
	return "anthropic"
}

// GetModelInfo returns information about the model
func (c *AnthropicClient) GetModelInfo() map[string]string {
	return map[string]string{
		"name":         c.modelName,
		"provider":     "Anthropic",
		"capabilities": "text generation, instruction following, chat",
	}
}

// GetModelList returns available models
func (c *AnthropicClient) GetModelList(ctx context.Context) ([]string, error) {
	// In practice, you'd fetch this from Anthropic's API
	// This is a static list for demonstration
	return []string{
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
		"claude-2.1",
		"claude-2.0",
		"claude-instant-1.2",
	}, nil
}

// GetMaxTokens returns the maximum context size for the model
func (c *AnthropicClient) GetMaxTokens() int {
	return c.maxTokens
}

// CreateEmbedding creates an embedding for the given text
func (c *AnthropicClient) CreateEmbedding(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Stub implementation - Anthropic doesn't have an embedding API yet
	return &interfaces.EmbeddingResponse{
		Embedding:  make([]float32, 1024), // Stub dimension
		Dimensions: 1024,
		Model:      c.modelName + "-embedding",
		TokensUsed: len(req.Text) / 4, // Rough estimation
	}, nil
}

// CreateEmbeddings creates embeddings for multiple texts
func (c *AnthropicClient) CreateEmbeddings(ctx context.Context, req *interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Stub implementation - Anthropic doesn't have an embedding API yet
	embeddings := make([][]float32, len(req.Texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, 1024)
	}

	return &interfaces.EmbeddingResponse{
		Embeddings: embeddings,
		Dimensions: 1024,
		Model:      c.modelName + "-embedding",
		TokensUsed: len(strings.Join(req.Texts, " ")) / 4, // Rough estimation
	}, nil
}

// GetEmbeddingDimension returns the dimension of embeddings from this provider
func (c *AnthropicClient) GetEmbeddingDimension(model string) int {
	return 1024 // Stub dimension
}