package mock

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// Provider is a mock AI provider for testing
type Provider struct {
	mu              sync.Mutex
	responses       map[string]string              // Predefined responses by prompt
	errors          map[string]error               // Predefined errors by prompt
	calls           []CallRecord                   // Record of all calls
	defaultResponse string                         // Default response if no match
	delay           time.Duration                  // Simulated latency
	capabilities    interfaces.ProviderCapabilities
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
func NewProvider() *Provider {
	return &Provider{
		responses:       make(map[string]string),
		errors:          make(map[string]error),
		calls:           make([]CallRecord, 0),
		defaultResponse: "Mock response",
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
	p.mu.Lock()
	defer p.mu.Unlock()

	// Simulate latency with context awareness
	if p.delay > 0 {
		select {
		case <-time.After(p.delay):
			// Delay completed
		case <-ctx.Done():
			// Context cancelled or timed out
			return nil, ctx.Err()
		}
	}

	// Extract user message for matching
	userMessage := ""
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			userMessage = msg.Content
			break
		}
	}

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

	// Get response
	response := p.defaultResponse
	if resp, exists := p.responses[userMessage]; exists {
		response = resp
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
					Content: response,
				},
				FinishReason: "stop",
			},
		},
		Usage: interfaces.UsageInfo{
			PromptTokens:     len(userMessage) / 4, // Rough estimate
			CompletionTokens: len(response) / 4,
			TotalTokens:      (len(userMessage) + len(response)) / 4,
		},
	}

	call.Response = result
	p.calls = append(p.calls, call)

	return result, nil
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
func NewBuilder() *Builder {
	return &Builder{
		provider: NewProvider(),
	}
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