package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/observability"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Latest Anthropic Claude models as of May 2025
const (
	// Claude 4 Series
	Claude4Opus   = "claude-4-opus"   // $15/$75 per million tokens, 200K context
	Claude4Sonnet = "claude-4-sonnet" // $3/$15 per million tokens, 200K context

	// Claude 3.5 Series
	Claude35Haiku = "claude-3-haiku-20241022" // $1/$5 per million tokens, 200K context
)

// Client implements the AIProvider interface for Anthropic
type Client struct {
	apiKey       string
	baseURL      string
	client       *http.Client
	capabilities interfaces.ProviderCapabilities
}

// NewClient creates a new Anthropic client
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	capabilities := interfaces.ProviderCapabilities{
		MaxTokens:      200000,
		ContextWindow:  200000,
		SupportsVision: true,
		SupportsTools:  true,
		SupportsStream: true,
		Models: []interfaces.ModelInfo{
			{
				ID:            Claude4Opus,
				Name:          "Claude 4 Opus",
				ContextWindow: 200000,
				MaxOutput:     32768,
				InputCost:     15.0,
				OutputCost:    75.0,
			},
			{
				ID:            Claude4Sonnet,
				Name:          "Claude 4 Sonnet",
				ContextWindow: 200000,
				MaxOutput:     64000,
				InputCost:     3.0,
				OutputCost:    15.0,
			},
			{
				ID:            Claude35Haiku,
				Name:          "Claude 3.5 Haiku",
				ContextWindow: 200000,
				MaxOutput:     8192,
				InputCost:     1.0,
				OutputCost:    5.0,
			},
		},
	}

	return &Client{
		apiKey:       apiKey, // pragma: allowlist secret
		baseURL:      "https://api.anthropic.com/v1",
		client:       &http.Client{Timeout: 2 * time.Minute},
		capabilities: capabilities,
	}
}

// ChatCompletion implements the AIProvider interface
func (c *Client) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("providers.anthropic").
		WithOperation("ChatCompletion").
		With("model", req.Model).
		With("message_count", len(req.Messages))

	// Convert messages to Anthropic format
	anthropicMessages := make([]map[string]interface{}, 0)
	systemPrompt := ""

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// Anthropic uses a separate system parameter
			systemPrompt = msg.Content
		} else {
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}
	}

	start := time.Now()
	logger.InfoContext(ctx, "Starting Anthropic chat completion request",
		"max_tokens", req.MaxTokens,
		"temperature", req.Temperature,
		"has_system_prompt", systemPrompt != "",
		"message_count", len(anthropicMessages),
	)

	// Build Anthropic request
	anthropicReq := map[string]interface{}{
		"model":      req.Model,
		"messages":   anthropicMessages,
		"max_tokens": 4096, // Default if not specified
	}

	if systemPrompt != "" {
		anthropicReq["system"] = systemPrompt
	}

	if req.MaxTokens > 0 {
		anthropicReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		anthropicReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		anthropicReq["top_p"] = req.TopP
	}
	if len(req.Stop) > 0 {
		anthropicReq["stop_sequences"] = req.Stop
	}

	// Make request
	respBody, err := c.makeRequest(ctx, "messages", anthropicReq)
	if err != nil {
		duration := time.Since(start)
		logger.WithError(err).ErrorContext(ctx, "Anthropic API request failed",
			"duration_ms", duration.Milliseconds(),
		)
		return nil, err
	}

	// Parse Anthropic response
	var anthropicResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProviderAPI, "failed to parse Anthropic response").
			WithComponent("providers").
			WithOperation("ChatCompletion").
			WithDetails("provider", "anthropic")
	}

	// Convert to our format
	content := ""
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	response := &interfaces.ChatResponse{
		ID:    anthropicResp.ID,
		Model: anthropicResp.Model,
		Choices: []interfaces.ChatChoice{
			{
				Index: 0,
				Message: interfaces.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: anthropicResp.StopReason,
			},
		},
		Usage: interfaces.UsageInfo{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}

	duration := time.Since(start)
	logger.InfoContext(ctx, "Anthropic chat completion succeeded",
		"duration_ms", duration.Milliseconds(),
		"input_tokens", anthropicResp.Usage.InputTokens,
		"output_tokens", anthropicResp.Usage.OutputTokens,
		"total_tokens", anthropicResp.Usage.InputTokens+anthropicResp.Usage.OutputTokens,
		"stop_reason", anthropicResp.StopReason,
		"response_length", len(content),
	)

	return response, nil
}

// StreamChatCompletion implements streaming for Anthropic
func (c *Client) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	// TODO: Implement Anthropic streaming (uses SSE format)
	return nil, gerror.New(gerror.ErrCodeProvider, "streaming not yet implemented for Anthropic", nil).
		WithComponent("providers").
		WithOperation("StreamChatCompletion").
		WithDetails("provider", "anthropic")
}

// CreateEmbedding implements the AIProvider interface
func (c *Client) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Note: Anthropic doesn't provide embeddings API
	// You would need to use a different provider for embeddings
	return nil, gerror.New(gerror.ErrCodeProvider, "Anthropic does not support embeddings - use OpenAI or another provider", nil).
		WithComponent("providers").
		WithOperation("CreateEmbedding").
		WithDetails("provider", "anthropic").
		WithDetails("capability", "embeddings")
}

// GetCapabilities returns provider capabilities
func (c *Client) GetCapabilities() interfaces.ProviderCapabilities {
	return c.capabilities
}

// makeRequest makes an HTTP request to the Anthropic API
func (c *Client) makeRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + "/" + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	// Anthropic uses different headers
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

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

// parseError parses Anthropic API error responses
func (c *Client) parseError(statusCode int, body []byte) error {
	err := &interfaces.ProviderError{
		Provider:   "anthropic",
		StatusCode: statusCode,
	}

	// Try to parse Anthropic error format
	var errorResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
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
	case "coding":
		return Claude4Opus // Best for coding
	case "reasoning":
		return Claude4Opus // Best reasoning
	case "cost-efficient":
		return Claude35Haiku // Most cost-efficient
	case "general":
		return Claude4Sonnet // Balanced
	default:
		return Claude4Sonnet // Default
	}
}

// Legacy LLMClient interface support
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	req := interfaces.ChatRequest{
		Model: Claude4Sonnet, // Default model
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
