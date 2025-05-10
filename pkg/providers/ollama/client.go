package ollama

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
	defaultAPI     = "http://localhost:11434/api/generate"
	defaultModel   = "llama3"
	defaultTimeout = 60 * time.Second
)

// OllamaClient implements the LLMClient interface for Ollama API
type OllamaClient struct {
	apiURL     string
	modelName  string
	httpClient *http.Client
	maxTokens  int
}

// Ollama API request structure
type ollamaRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Stream      bool    `json:"stream,omitempty"`
	Options     options `json:"options,omitempty"`
	System      string  `json:"system,omitempty"`
	Context     []int   `json:"context,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

type options struct {
	NumPredict int `json:"num_predict,omitempty"`
}

// Ollama API response structure
type ollamaResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// NewClient creates a new Ollama client
func NewClient(opts ...ClientOption) (*OllamaClient, error) {
	client := &OllamaClient{
		apiURL:     defaultAPI,
		modelName:  defaultModel,
		httpClient: &http.Client{Timeout: defaultTimeout},
		maxTokens:  4096, // Default context size for many Ollama models
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// ClientOption defines a function that configures the Ollama client
type ClientOption func(*OllamaClient)

// WithModel sets the model to use
func WithModel(model string) ClientOption {
	return func(c *OllamaClient) {
		c.modelName = model

		// Adjust context window based on model
		// This is a simplified version - in reality you'd want to query the model info
		switch model {
		case "llama3:70b":
			c.maxTokens = 8192
		case "llama3", "llama3:8b":
			c.maxTokens = 8192
		case "mistral", "mistral:7b":
			c.maxTokens = 8192
		case "mixtral", "mixtral:8x7b":
			c.maxTokens = 32768
		}
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *OllamaClient) {
		c.httpClient.Timeout = timeout
	}
}

// WithEndpoint sets a custom API endpoint
func WithEndpoint(endpoint string) ClientOption {
	return func(c *OllamaClient) {
		c.apiURL = endpoint
	}
}

// Complete implements the LLMClient interface
func (c *OllamaClient) Complete(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	// Convert to Ollama request format
	ollamaReq := ollamaRequest{
		Model:  c.modelName,
		Prompt: req.Prompt,
		Options: options{
			NumPredict: req.MaxTokens,
		},
		Temperature: req.Temperature,
		Stream:      false,
	}

	// Add system prompt if provided
	if systemPrompt, ok := req.Options["system"]; ok {
		ollamaReq.System = systemPrompt
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(ollamaReq)
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
	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Ollama doesn't provide token counts, so we'll estimate
	// This is a very rough approximation
	promptTokens := len(req.Prompt) / 4
	responseTokens := len(ollamaResp.Response) / 4

	// Convert to generic response
	return &providers.CompletionResponse{
		Text:         ollamaResp.Response,
		TokensInput:  promptTokens,
		TokensOutput: responseTokens,
		TokensUsed:   promptTokens + responseTokens,
		FinishReason: "stop", // Ollama doesn't provide this explicitly
		ModelUsed:    ollamaResp.Model,
	}, nil
}

// GetName returns the provider name
func (c *OllamaClient) GetName() string {
	return "ollama"
}

// GetModelInfo returns information about the model
func (c *OllamaClient) GetModelInfo() map[string]string {
	return map[string]string{
		"name":         c.modelName,
		"provider":     "Ollama",
		"capabilities": "text generation, local inference",
	}
}

// GetModelList returns available models
func (c *OllamaClient) GetModelList(ctx context.Context) ([]string, error) {
	// In a real implementation, we'd query the Ollama API at /api/tags
	// For now, return a static list of common models
	return []string{
		"llama3",
		"llama3:8b",
		"llama3:70b",
		"mistral",
		"mistral:7b",
		"mixtral",
		"mixtral:8x7b",
		"phi3",
		"phi3:mini",
		"codellama",
		"codellama:7b",
		"codellama:13b",
		"codellama:34b",
	}, nil
}

// GetMaxTokens returns the maximum context size for the model
func (c *OllamaClient) GetMaxTokens() int {
	return c.maxTokens
}

