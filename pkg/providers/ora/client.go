package ora

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Ora models
const (
	// DeepSeek Models
	DeepSeekV3 = "deepseek-v3"
	DeepSeekR1 = "deepseek-r1"
	
	// Other Models (if supported)
	GPT4Turbo = "gpt-4-turbo"
	Claude3 = "claude-3"
)

// Client implements the AIProvider interface for Ora
type Client struct {
	apiKey       string
	baseURL      string
	client       *http.Client
	capabilities interfaces.ProviderCapabilities
}

// NewClient creates a new Ora client
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("ORA_API_KEY")
	}

	capabilities := interfaces.ProviderCapabilities{
		MaxTokens:      64000,
		ContextWindow:  64000,
		SupportsVision: false,
		SupportsTools:  true,
		SupportsStream: true,
		Models: []interfaces.ModelInfo{
			{
				ID:            DeepSeekV3,
				Name:          "DeepSeek V3",
				ContextWindow: 64000,
				MaxOutput:     8192,
				InputCost:     0.1,  // Estimated
				OutputCost:    1.0,  // Estimated
			},
			{
				ID:            DeepSeekR1,
				Name:          "DeepSeek R1",
				ContextWindow: 64000,
				MaxOutput:     8192,
				InputCost:     0.5,
				OutputCost:    2.0,
			},
		},
	}

	return &Client{
		apiKey:       apiKey,
		baseURL:      "https://api.ora.ai/v1",
		client:       &http.Client{Timeout: 2 * time.Minute},
		capabilities: capabilities,
	}
}

// ChatCompletion implements the AIProvider interface
func (c *Client) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	// Build Ora request
	oraReq := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}

	if req.MaxTokens > 0 {
		oraReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		oraReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		oraReq["top_p"] = req.TopP
	}
	if len(req.Stop) > 0 {
		oraReq["stop"] = req.Stop
	}

	// Make request
	respBody, err := c.makeRequest(ctx, "chat/completions", oraReq)
	if err != nil {
		return nil, err
	}

	// Parse response (assuming OpenAI-compatible format)
	var oraResp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &oraResp); err != nil {
		return nil, fmt.Errorf("failed to parse Ora response: %w", err)
	}

	// Convert to our format
	choices := make([]interfaces.ChatChoice, len(oraResp.Choices))
	for i, c := range oraResp.Choices {
		choices[i] = interfaces.ChatChoice{
			Index: c.Index,
			Message: interfaces.ChatMessage{
				Role:    c.Message.Role,
				Content: c.Message.Content,
			},
			FinishReason: c.FinishReason,
		}
	}

	return &interfaces.ChatResponse{
		ID:    oraResp.ID,
		Model: oraResp.Model,
		Choices: choices,
		Usage: interfaces.UsageInfo{
			PromptTokens:     oraResp.Usage.PromptTokens,
			CompletionTokens: oraResp.Usage.CompletionTokens,
			TotalTokens:      oraResp.Usage.TotalTokens,
		},
	}, nil
}

// StreamChatCompletion implements streaming for Ora
func (c *Client) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	// TODO: Implement streaming if Ora supports it
	return nil, fmt.Errorf("streaming not yet implemented for Ora")
}

// CreateEmbedding implements the AIProvider interface
func (c *Client) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Check if Ora supports embeddings
	return nil, fmt.Errorf("embeddings not supported by Ora - use another provider")
}

// GetCapabilities returns provider capabilities
func (c *Client) GetCapabilities() interfaces.ProviderCapabilities {
	return c.capabilities
}

// makeRequest makes an HTTP request to the Ora API
func (c *Client) makeRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
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
		return nil, c.parseError(resp.StatusCode, body)
	}

	return body, nil
}

// parseError parses Ora API error responses
func (c *Client) parseError(statusCode int, body []byte) error {
	err := &interfaces.ProviderError{
		Provider:   "ora",
		StatusCode: statusCode,
	}

	// Try to parse error response
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if json.Unmarshal(body, &errorResp) == nil && errorResp.Error.Message != "" {
		err.Message = errorResp.Error.Message
		err.Type = errorResp.Error.Type
	} else {
		err.Message = string(body)
	}

	// Determine if retryable based on status code
	switch statusCode {
	case 401, 403:
		err.Type = interfaces.ErrorTypeAuth
		err.Retryable = false
	case 429:
		err.Type = interfaces.ErrorTypeRateLimit
		err.Retryable = true
	case 500, 502, 503, 504:
		err.Type = interfaces.ErrorTypeServer
		err.Retryable = true
	default:
		if err.Type == "" {
			err.Type = interfaces.ErrorTypeUnknown
		}
		err.Retryable = false
	}

	return err
}

// GetRecommendedModel returns a recommended model for a given use case
func GetRecommendedModel(useCase string) string {
	switch useCase {
	case "reasoning":
		return DeepSeekR1
	case "general", "cost-efficient":
		return DeepSeekV3
	default:
		return DeepSeekV3
	}
}

// Legacy LLMClient interface support
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	req := interfaces.ChatRequest{
		Model: DeepSeekV3, // Default model
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