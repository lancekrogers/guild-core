package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Popular Ollama models
const (
	// Llama Models
	Llama33_70B = "llama3.3:70b"
	Llama31_8B  = "llama3.1:8b"
	Llama32Vision = "llama3.2-vision:11b"

	// Gemma Models
	Gemma2_27B = "gemma2:27b"
	Gemma2_9B  = "gemma2:9b"
	Gemma2_2B  = "gemma2:2b"

	// Other Models
	DeepSeekR1 = "deepseek-r1:70b"
	Phi4       = "phi4:14b"
	Qwen3_72B  = "qwen3:72b"
	Mistral    = "mistral:latest"
)

// Client implements the AIProvider interface for Ollama
type Client struct {
	baseURL      string
	client       *http.Client
	capabilities interfaces.ProviderCapabilities
}

// NewClient creates a new Ollama client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = os.Getenv("OLLAMA_HOST")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
	}

	capabilities := interfaces.ProviderCapabilities{
		MaxTokens:      128000, // Varies by model
		ContextWindow:  128000,
		SupportsVision: true,  // Some models support vision
		SupportsTools:  false, // Ollama doesn't support function calling yet
		SupportsStream: true,
		Models: []interfaces.ModelInfo{
			{
				ID:            Llama33_70B,
				Name:          "Llama 3.3 70B",
				ContextWindow: 131072,
				MaxOutput:     8192,
				InputCost:     0, // Free (local)
				OutputCost:    0,
			},
			{
				ID:            Llama31_8B,
				Name:          "Llama 3.1 8B",
				ContextWindow: 131072,
				MaxOutput:     8192,
				InputCost:     0,
				OutputCost:    0,
			},
			{
				ID:            Gemma2_9B,
				Name:          "Gemma 2 9B",
				ContextWindow: 8192,
				MaxOutput:     8192,
				InputCost:     0,
				OutputCost:    0,
			},
			{
				ID:            Mistral,
				Name:          "Mistral Latest",
				ContextWindow: 32768,
				MaxOutput:     8192,
				InputCost:     0,
				OutputCost:    0,
			},
		},
	}

	return &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Minute, // Longer timeout for local models
		},
		capabilities: capabilities,
	}
}

// ChatCompletion implements the AIProvider interface
func (c *Client) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	// Convert to Ollama format
	ollamaReq := map[string]interface{}{
		"model":  req.Model,
		"stream": false,
	}

	// Convert messages to Ollama format
	messages := make([]map[string]string, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	ollamaReq["messages"] = messages

	// Add options
	options := make(map[string]interface{})
	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		options["top_p"] = req.TopP
	}
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	}
	if len(req.Stop) > 0 {
		options["stop"] = req.Stop
	}
	if len(options) > 0 {
		ollamaReq["options"] = options
	}

	// Make request
	respBody, err := c.makeRequest(ctx, "/api/chat", ollamaReq)
	if err != nil {
		return nil, err
	}

	// Parse Ollama response
	var ollamaResp struct {
		Model              string `json:"model"`
		CreatedAt          string `json:"created_at"`
		Message            struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done               bool   `json:"done"`
		TotalDuration      int64  `json:"total_duration"`
		LoadDuration       int64  `json:"load_duration"`
		PromptEvalCount    int    `json:"prompt_eval_count"`
		PromptEvalDuration int64  `json:"prompt_eval_duration"`
		EvalCount          int    `json:"eval_count"`
		EvalDuration       int64  `json:"eval_duration"`
	}

	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProviderAPI, "failed to parse Ollama response").
			WithComponent("providers").
			WithOperation("ChatCompletion").
			WithDetails("provider", "ollama")
	}

	// Convert to our format
	return &interfaces.ChatResponse{
		ID:    fmt.Sprintf("ollama-%d", time.Now().Unix()),
		Model: ollamaResp.Model,
		Choices: []interfaces.ChatChoice{
			{
				Index: 0,
				Message: interfaces.ChatMessage{
					Role:    ollamaResp.Message.Role,
					Content: ollamaResp.Message.Content,
				},
				FinishReason: "stop",
			},
		},
		Usage: interfaces.UsageInfo{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
	}, nil
}

// StreamChatCompletion implements streaming for Ollama
func (c *Client) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	// Build request with streaming enabled
	ollamaReq := map[string]interface{}{
		"model":  req.Model,
		"stream": true,
	}

	// Convert messages
	messages := make([]map[string]string, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	ollamaReq["messages"] = messages

	data, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, gerror.Newf(gerror.ErrCodeProviderAPI, "ollama error %d: %s", resp.StatusCode, string(body)).
			WithComponent("providers").
			WithOperation("StreamChatCompletion").
			WithDetails("provider", "ollama").
			WithDetails("status_code", resp.StatusCode)
	}

	return &ollamaStream{
		reader:  resp.Body,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

// CreateEmbedding implements the AIProvider interface
func (c *Client) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Ollama embedding endpoint expects a single prompt
	if len(req.Input) == 0 {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "no input provided for embedding", nil).
			WithComponent("providers").
			WithOperation("CreateEmbedding").
			WithDetails("provider", "ollama")
	}

	embeddings := make([]interfaces.Embedding, len(req.Input))

	for i, input := range req.Input {
		ollamaReq := map[string]interface{}{
			"model":  req.Model,
			"prompt": input,
		}

		respBody, err := c.makeRequest(ctx, "/api/embeddings", ollamaReq)
		if err != nil {
			return nil, err
		}

		var embResp struct {
			Embedding []float64 `json:"embedding"`
		}

		if err := json.Unmarshal(respBody, &embResp); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeProviderAPI, "failed to parse embedding response").
				WithComponent("providers").
				WithOperation("CreateEmbedding").
				WithDetails("provider", "ollama")
		}

		embeddings[i] = interfaces.Embedding{
			Index:     i,
			Embedding: embResp.Embedding,
		}
	}

	return &interfaces.EmbeddingResponse{
		Model:      req.Model,
		Embeddings: embeddings,
		Usage: interfaces.UsageInfo{
			PromptTokens: len(strings.Join(req.Input, " ")), // Rough estimate
			TotalTokens:  len(strings.Join(req.Input, " ")),
		},
	}, nil
}

// GetCapabilities returns provider capabilities
func (c *Client) GetCapabilities() interfaces.ProviderCapabilities {
	return c.capabilities
}

// PullModel downloads a model from Ollama registry
func (c *Client) PullModel(ctx context.Context, model string) error {
	payload := map[string]string{"name": model}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/pull", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Stream progress updates
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var progress map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &progress); err == nil {
			// Could emit progress events here
			if status, ok := progress["status"].(string); ok {
				fmt.Printf("Pull progress: %s\n", status)
			}
		}
	}

	return scanner.Err()
}

// ListModels returns available models on the Ollama server
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// makeRequest makes an HTTP request to the Ollama API
func (c *Client) makeRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, gerror.Newf(gerror.ErrCodeProviderAPI, "ollama error %d: %s", resp.StatusCode, string(body)).
			WithComponent("providers").
			WithOperation("makeRequest").
			WithDetails("provider", "ollama").
			WithDetails("endpoint", endpoint).
			WithDetails("status_code", resp.StatusCode)
	}

	return body, nil
}

// ollamaStream implements ChatStream for Ollama
type ollamaStream struct {
	reader  io.ReadCloser
	scanner *bufio.Scanner
}

func (s *ollamaStream) Next() (interfaces.ChatStreamChunk, error) {
	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return interfaces.ChatStreamChunk{}, err
		}
		return interfaces.ChatStreamChunk{}, io.EOF
	}

	var chunk struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}

	if err := json.Unmarshal(s.scanner.Bytes(), &chunk); err != nil {
		return interfaces.ChatStreamChunk{}, err
	}

	return interfaces.ChatStreamChunk{
		Delta: interfaces.ChatMessage{
			Content: chunk.Message.Content,
		},
		FinishReason: func() string {
			if chunk.Done {
				return "stop"
			}
			return ""
		}(),
	}, nil
}

func (s *ollamaStream) Close() error {
	return s.reader.Close()
}

// GetRecommendedModel returns a recommended model for a given use case
func GetRecommendedModel(useCase string) string {
	switch useCase {
	case "coding":
		return Llama31_8B // Good for coding
	case "reasoning":
		return DeepSeekR1 // If available
	case "vision":
		return Llama32Vision
	case "fast":
		return Gemma2_2B // Smallest
	case "general":
		return Llama31_8B // Good balance
	default:
		return Mistral // Reliable default
	}
}

// Legacy LLMClient interface support
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	req := interfaces.ChatRequest{
		Model: Llama31_8B, // Default model
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := c.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", nil
}
