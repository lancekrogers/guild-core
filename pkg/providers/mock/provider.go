// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mock

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Provider is a mock AI provider for testing
type Provider struct {
	mu              sync.Mutex
	responses       map[string]string // Predefined responses by prompt (legacy)
	errors          map[string]error  // Predefined errors by prompt
	calls           []CallRecord      // Record of all calls
	defaultResponse string            // Default response if no match
	delay           time.Duration     // Simulated latency
	capabilities    interfaces.ProviderCapabilities
	yamlResponses   ResponseSet // YAML-based responses
	enabled         bool        // Whether mock provider is enabled
}

// CallRecord records details of each API call
type CallRecord struct {
	Timestamp time.Time
	Method    string
	Request   interface{}
	Response  interface{}
	Error     error
}

// NewProvider creates a new mock provider
func NewProvider() (*Provider, error) {
	// Only enable if environment variable is set
	enabled := os.Getenv("GUILD_MOCK_PROVIDER") == "true"

	provider := &Provider{
		responses:       make(map[string]string),
		errors:          make(map[string]error),
		calls:           make([]CallRecord, 0),
		defaultResponse: "Mock response",
		enabled:         enabled,
		capabilities: interfaces.ProviderCapabilities{
			MaxTokens:          4096,
			ContextWindow:      8192,
			SupportsVision:     false,
			SupportsTools:      false,
			SupportsStream:     true,
			SupportsEmbeddings: true,
			Models: []interfaces.ModelInfo{
				{
					ID:            "mock-model-v1",
					Name:          "Mock Model v1",
					ContextWindow: 8192,
					MaxOutput:     4096,
					InputCost:     0,
					OutputCost:    0,
				},
				{
					ID:            "mock-model-fast",
					Name:          "Mock Model Fast",
					ContextWindow: 4096,
					MaxOutput:     2048,
					InputCost:     0,
					OutputCost:    0,
				},
				{
					ID:            "mock-model-smart",
					Name:          "Mock Model Smart",
					ContextWindow: 16384,
					MaxOutput:     8192,
					InputCost:     0,
					OutputCost:    0,
				},
			},
		},
	}

	if enabled {
		// Load YAML responses
		yamlResponses, err := loadResponses()
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load mock responses").
				WithComponent("providers").
				WithOperation("NewProvider")
		}
		provider.yamlResponses = yamlResponses
	}

	return provider, nil
}

// SetResponse sets a predefined response for a specific prompt
func (p *Provider) SetResponse(prompt string, response string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses[prompt] = response
}

// SetError sets a predefined error for a specific prompt
func (p *Provider) SetError(prompt string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errors[prompt] = err
}

// SetDefaultResponse sets the default response for unmatched prompts
func (p *Provider) SetDefaultResponse(response string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultResponse = response
}

// SetDelay sets simulated latency for all calls
func (p *Provider) SetDelay(delay time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.delay = delay
}

// GetCalls returns all recorded calls
func (p *Provider) GetCalls() []CallRecord {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]CallRecord{}, p.calls...)
}

// ResetCalls clears the call history
func (p *Provider) ResetCalls() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = make([]CallRecord, 0)
}

// ChatCompletion implements the AIProvider interface
func (p *Provider) ChatCompletion(ctx context.Context, req interfaces.ChatRequest) (*interfaces.ChatResponse, error) {
	if !p.enabled {
		return nil, gerror.New(gerror.ErrCodeProvider, "mock provider not enabled", nil).
			WithComponent("providers").
			WithOperation("ChatCompletion")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Extract user message for matching
	userMessage := ""
	allMessages := make([]string, 0, len(req.Messages))
	for _, msg := range req.Messages {
		allMessages = append(allMessages, msg.Content)
		if msg.Role == "user" {
			userMessage = msg.Content
		}
	}
	fullContext := strings.ToLower(strings.Join(allMessages, " "))

	// Record the call
	call := CallRecord{
		Timestamp: time.Now(),
		Method:    "ChatCompletion",
		Request:   req,
	}

	// Check for predefined error
	if err, exists := p.errors[userMessage]; exists {
		call.Error = err
		p.calls = append(p.calls, call)
		return nil, err
	}

	// Try to find YAML-based response first
	var responseContent string
	var delay time.Duration
	tokens := 50 // Default token count

	if yamlResponse := p.findYAMLResponse(fullContext); yamlResponse != nil {
		// Use YAML response
		responseContent = strings.Join(yamlResponse.Messages, "")
		if yamlResponse.Delay > 0 {
			delay = time.Duration(yamlResponse.Delay) * time.Millisecond
		} else {
			delay = p.delay
		}
		if yamlResponse.Tokens > 0 {
			tokens = yamlResponse.Tokens
		}
	} else if resp, exists := p.responses[userMessage]; exists {
		// Use legacy string response
		responseContent = resp
		delay = p.delay
	} else {
		// Use default response
		responseContent = p.getGuildDefaultResponse()
		delay = p.delay
	}

	// Simulate latency with context awareness
	if delay > 0 {
		select {
		case <-time.After(delay):
			// Delay completed
		case <-ctx.Done():
			// Context cancelled or timed out
			return nil, ctx.Err()
		}
	}

	// Build response
	result := &interfaces.ChatResponse{
		ID:    fmt.Sprintf("mock-%d", time.Now().Unix()),
		Model: req.Model,
		Choices: []interfaces.ChatChoice{
			{
				Index: 0,
				Message: interfaces.ChatMessage{
					Role:    "assistant",
					Content: responseContent,
				},
				FinishReason: "stop",
			},
		},
		Usage: interfaces.UsageInfo{
			PromptTokens:     len(userMessage) / 4, // Rough estimate
			CompletionTokens: tokens,
			TotalTokens:      len(userMessage)/4 + tokens,
		},
	}

	call.Response = result
	p.calls = append(p.calls, call)

	return result, nil
}

// findYAMLResponse searches for a matching YAML response pattern
func (p *Provider) findYAMLResponse(fullContext string) *Response {
	// Search for matching patterns
	for _, resp := range p.yamlResponses.Responses {
		for _, pattern := range resp.Patterns {
			if strings.Contains(fullContext, strings.ToLower(pattern)) {
				return &resp
			}
		}
	}
	return nil
}

// getGuildDefaultResponse returns a Guild-specific default response
func (p *Provider) getGuildDefaultResponse() string {
	return `I understand your request. Let me help you with that.

Based on the context, I'll proceed with the implementation.

**Analysis:**
- Task type: Development request
- Complexity: Standard
- Approach: Systematic implementation

I'll work through this step by step to ensure quality results.

*Note: This is a mock response for testing purposes.*`
}

// StreamChatCompletion implements streaming (returns simple mock stream)
func (p *Provider) StreamChatCompletion(ctx context.Context, req interfaces.ChatRequest) (interfaces.ChatStream, error) {
	// Get the full response first
	resp, err := p.ChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return a mock stream that yields the response in chunks
	return &mockStream{
		content: resp.Choices[0].Message.Content,
		chunks:  splitIntoChunks(resp.Choices[0].Message.Content, 10),
		index:   0,
	}, nil
}

// CreateEmbedding implements the AIProvider interface
func (p *Provider) CreateEmbedding(ctx context.Context, req interfaces.EmbeddingRequest) (*interfaces.EmbeddingResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	call := CallRecord{
		Timestamp: time.Now(),
		Method:    "CreateEmbedding",
		Request:   req,
	}

	// Create mock embeddings
	embeddings := make([]interfaces.Embedding, len(req.Input))
	for i := range req.Input {
		// Create a simple mock embedding
		embedding := make([]float64, 384) // Common embedding size
		for j := range embedding {
			embedding[j] = float64(i+j) * 0.01 // Deterministic values
		}
		embeddings[i] = interfaces.Embedding{
			Index:     i,
			Embedding: embedding,
		}
	}

	result := &interfaces.EmbeddingResponse{
		Model:      req.Model,
		Embeddings: embeddings,
		Usage: interfaces.UsageInfo{
			PromptTokens: len(req.Input) * 10,
			TotalTokens:  len(req.Input) * 10,
		},
	}

	call.Response = result
	p.calls = append(p.calls, call)

	return result, nil
}

// GetCapabilities returns provider capabilities
func (p *Provider) GetCapabilities() interfaces.ProviderCapabilities {
	return p.capabilities
}

// Complete implements the LLMClient interface for compatibility with older systems
func (p *Provider) Complete(ctx context.Context, prompt string) (string, error) {
	// Create a simple request using the prompt
	req := interfaces.ChatRequest{
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: prompt},
		},
		Model: "mock-model",
	}

	// Use the existing ChatCompletion implementation
	resp, err := p.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", gerror.New(gerror.ErrCodeProviderAPI, "no response choices available", nil).
			WithComponent("providers").
			WithOperation("Complete").
			WithDetails("provider", "mock")
	}

	return resp.Choices[0].Message.Content, nil
}

// mockStream implements ChatStream for testing
type mockStream struct {
	content string
	chunks  []string
	index   int
}

func (s *mockStream) Next() (interfaces.ChatStreamChunk, error) {
	if s.index >= len(s.chunks) {
		return interfaces.ChatStreamChunk{}, io.EOF
	}

	chunk := interfaces.ChatStreamChunk{
		Delta: interfaces.ChatMessage{
			Content: s.chunks[s.index],
		},
	}

	s.index++

	if s.index >= len(s.chunks) {
		chunk.FinishReason = "stop"
	}

	return chunk, nil
}

func (s *mockStream) Close() error {
	return nil
}

// splitIntoChunks splits a string into chunks of specified size
func splitIntoChunks(s string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

// Builder provides a fluent interface for configuring the mock
type Builder struct {
	provider *Provider
}

// NewBuilder creates a new mock provider builder
func NewBuilder() (*Builder, error) {
	provider, err := NewProvider()
	if err != nil {
		return nil, err
	}
	return &Builder{
		provider: provider,
	}, nil
}

// WithResponse adds a response mapping
func (b *Builder) WithResponse(prompt, response string) *Builder {
	b.provider.SetResponse(prompt, response)
	return b
}

// WithError adds an error mapping
func (b *Builder) WithError(prompt string, err error) *Builder {
	b.provider.SetError(prompt, err)
	return b
}

// WithDefaultResponse sets the default response
func (b *Builder) WithDefaultResponse(response string) *Builder {
	b.provider.SetDefaultResponse(response)
	return b
}

// WithDelay sets the simulated latency
func (b *Builder) WithDelay(delay time.Duration) *Builder {
	b.provider.SetDelay(delay)
	return b
}

// Build returns the configured mock provider
func (b *Builder) Build() *Provider {
	return b.provider
}

// NewProviderForTesting creates a new mock provider for testing (legacy interface)
func NewProviderForTesting() *Provider {
	// Create a provider without environment variable check for testing
	provider := &Provider{
		responses:       make(map[string]string),
		errors:          make(map[string]error),
		calls:           make([]CallRecord, 0),
		defaultResponse: "Mock response",
		enabled:         true, // Always enabled for testing
		capabilities: interfaces.ProviderCapabilities{
			MaxTokens:          4096,
			ContextWindow:      8192,
			SupportsVision:     false,
			SupportsTools:      false,
			SupportsStream:     true,
			SupportsEmbeddings: true,
			Models: []interfaces.ModelInfo{
				{
					ID:            "mock-model",
					Name:          "Mock Model",
					ContextWindow: 8192,
					MaxOutput:     4096,
					InputCost:     0,
					OutputCost:    0,
				},
			},
		},
	}

	// Try to load YAML responses for testing
	if yamlResponses, err := loadResponses(); err == nil {
		provider.yamlResponses = yamlResponses
	}

	return provider
}
