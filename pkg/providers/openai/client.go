package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/blockhead-consulting/Guild/pkg/providers"
)

const (
	defaultAPI     = "https://api.openai.com/v1/chat/completions"
	defaultModel   = "gpt-4-turbo"
	defaultTimeout = 30 * time.Second
)

// OpenAIClient implements the LLMClient interface for OpenAI's models
type OpenAIClient struct {
	apiKey     string
	apiURL     string
	modelName  string
	httpClient *http.Client
	maxTokens  int
}

// OpenAI API request structure
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI API response structure
type openaiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openaiMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient creates a new OpenAI client
func NewClient(apiKey string, opts ...ClientOption) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey cannot be empty")
	}

	client := &OpenAIClient{
		apiKey:     apiKey,
		apiURL:     defaultAPI,
		modelName:  defaultModel,
		httpClient: &http.Client{Timeout: defaultTimeout},
		maxTokens:  8192, // Default for GPT-4
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// ClientOption defines a function that configures the OpenAI client
type ClientOption func(*OpenAIClient)

// WithModel sets the model to use
func WithModel(model string) ClientOption {
	return func(c *OpenAIClient) {
		c.modelName = model

		// Update max tokens based on model
		switch model {
		case "gpt-4-turbo", "gpt-4-0125-preview":
			c.maxTokens = 128000
		case "gpt-4":
			c.maxTokens = 8192
		case "gpt-3.5-turbo":
			c.maxTokens = 16385
		}
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *OpenAIClient) {
		c.httpClient.Timeout = timeout
	}
}

// Complete implements the LLMClient interface
func (c *OpenAIClient) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	// Convert to OpenAI's format
	messages := []openaiMessage{
		{Role: "user", Content: req.Prompt},
	}

	// Add system message if provided
	if systemPrompt, ok := req.Options["system"]; ok {
		messages = append([]openaiMessage{{Role: "system", Content: systemPrompt}}, messages...)
	}

	oaiReq := openaiRequest{
		Model:       c.modelName,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stop:        req.StopTokens,
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.apiURL,
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

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
	var oaiResp openaiResponse
	if err := json.Unmarshal(body, &oaiResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Check if we have any choices
	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	// Convert to generic response
	return &providers.CompletionResponse{
		Text:         oaiResp.Choices[0].Message.Content,
		TokensInput:  oaiResp.Usage.PromptTokens,
		TokensOutput: oaiResp.Usage.CompletionTokens,
		TokensUsed:   oaiResp.Usage.TotalTokens,
		FinishReason: oaiResp.Choices[0].FinishReason,
		ModelUsed:    oaiResp.Model,
	}, nil
}

// GetName returns the provider name
func (c *OpenAIClient) GetName() string {
	return "openai"
}

// GetModelInfo returns information about the model
func (c *OpenAIClient) GetModelInfo() map[string]string {
	return map[string]string{
		"name":         c.modelName,
		"provider":     "OpenAI",
		"capabilities": "text generation, instruction following, chat",
	}
}

// GetModelList returns available models
func (c *OpenAIClient) GetModelList(ctx context.Context) ([]string, error) {
	// Here you would typically call the OpenAI API to get available models
	// For simplicity, we'll return a static list
	return []string{
		"gpt-4-turbo",
		"gpt-4-0125-preview",
		"gpt-4",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-instruct",
	}, nil
}

// GetMaxTokens returns the maximum context size for the model
func (c *OpenAIClient) GetMaxTokens() int {
	return c.maxTokens
}

