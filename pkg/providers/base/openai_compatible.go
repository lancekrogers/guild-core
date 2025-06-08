package base

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// OpenAICompatibleProvider implements AIProvider for OpenAI-compatible APIs
// This is exported to allow provider packages to embed it
type OpenAICompatibleProvider struct {
	name         string
	apiKey       string
	baseURL      string
	modelMap     map[string]string
	client       *http.Client
	capabilities interfaces.ProviderCapabilities
}

// NewOpenAICompatibleProvider creates a new OpenAI-compatible provider
func NewOpenAICompatibleProvider(name, apiKey, baseURL string, modelMap map[string]string, capabilities interfaces.ProviderCapabilities) *OpenAICompatibleProvider {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &OpenAICompatibleProvider{
		name:     name,
		apiKey:   apiKey,
		baseURL:  baseURL,
		modelMap: modelMap,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
		capabilities: capabilities,
	}
}

// ChatCompletion implements the AIProvider interface
func (p *OpenAICompatibleProvider) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	// Map model if needed
	model := req.Model
	if mapped, ok := p.modelMap[model]; ok {
		model = mapped
	}

	// Build OpenAI-format request
	openAIReq := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
		"stream":   false,
	}

	if req.MaxTokens > 0 {
		openAIReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		openAIReq["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		openAIReq["top_p"] = req.TopP
	}
	if len(req.Stop) > 0 {
		openAIReq["stop"] = req.Stop
	}

	// Add provider-specific options
	for k, v := range req.Options {
		openAIReq[k] = v
	}

	// Make request
	respBody, err := p.makeRequest(ctx, "chat/completions", openAIReq)
	if err != nil {
		return nil, err
	}

	// Parse response
	var openAIResp struct {
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

	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProviderAPI, "failed to parse response").
			WithComponent("providers").
			WithOperation("ChatCompletion").
			WithDetails("provider", p.name)
	}

	// Convert to our format
	choices := make([]interfaces.ChatChoice, len(openAIResp.Choices))
	for i, c := range openAIResp.Choices {
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
		ID:      openAIResp.ID,
		Model:   openAIResp.Model,
		Choices: choices,
		Usage: interfaces.UsageInfo{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}, nil
}

// StreamChatCompletion implements streaming for OpenAI-compatible APIs
func (p *OpenAICompatibleProvider) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	// Map model if needed
	model := req.Model
	if mapped, ok := p.modelMap[model]; ok {
		model = mapped
	}

	// Build request with stream enabled
	openAIReq := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
		"stream":   true,
	}

	if req.MaxTokens > 0 {
		openAIReq["max_tokens"] = req.MaxTokens
	}
	if req.Temperature > 0 {
		openAIReq["temperature"] = req.Temperature
	}

	data, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, p.parseError(resp.StatusCode, body)
	}

	return &openAIStream{
		reader:   resp.Body,
		scanner:  nil, // Will be initialized on first read
		provider: p.name,
	}, nil
}

// CreateEmbedding implements the AIProvider interface
func (p *OpenAICompatibleProvider) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	// Build request
	embReq := map[string]interface{}{
		"model": req.Model,
		"input": req.Input,
	}

	respBody, err := p.makeRequest(ctx, "embeddings", embReq)
	if err != nil {
		return nil, err
	}

	// Parse response
	var embResp struct {
		Model string `json:"model"`
		Data  []struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProviderAPI, "failed to parse embedding response").
			WithComponent("providers").
			WithOperation("CreateEmbedding").
			WithDetails("provider", p.name)
	}

	// Convert to our format
	embeddings := make([]interfaces.Embedding, len(embResp.Data))
	for i, e := range embResp.Data {
		embeddings[i] = interfaces.Embedding{
			Index:     e.Index,
			Embedding: e.Embedding,
		}
	}

	return &interfaces.EmbeddingResponse{
		Model:      embResp.Model,
		Embeddings: embeddings,
		Usage: interfaces.UsageInfo{
			PromptTokens: embResp.Usage.PromptTokens,
			TotalTokens:  embResp.Usage.TotalTokens,
		},
	}, nil
}

// GetCapabilities returns provider capabilities
func (p *OpenAICompatibleProvider) GetCapabilities() interfaces.ProviderCapabilities {
	return p.capabilities
}

// makeRequest makes an HTTP request to the API
func (p *OpenAICompatibleProvider) makeRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp.StatusCode, body)
	}

	return body, nil
}

// parseError parses API error responses
func (p *OpenAICompatibleProvider) parseError(statusCode int, body []byte) error {
	err := &interfaces.ProviderError{
		Provider:   p.name,
		StatusCode: statusCode,
	}

	// Try to parse OpenAI-style error
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
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

// openAIStream implements ChatStream for OpenAI SSE format
type openAIStream struct {
	reader   io.ReadCloser
	scanner  *json.Decoder
	provider string
}

func (s *openAIStream) Next() (interfaces.ChatStreamChunk, error) {
	// TODO: Implement SSE parsing for streaming
	// For now, return EOF to indicate no streaming support
	return interfaces.ChatStreamChunk{}, io.EOF
}

func (s *openAIStream) Close() error {
	return s.reader.Close()
}
