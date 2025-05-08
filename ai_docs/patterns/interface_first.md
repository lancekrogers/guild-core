# Interface-First Development

This document explains the interface-first development approach used in the Guild framework.

## Overview

Guild follows an interface-first development pattern, defining clear contracts between components before implementing concrete types. This approach enables:

1. **Loose coupling** between components
2. **Easier testing** through mocks
3. **Flexible implementation** choices
4. **Clear API boundaries** for developers

## Core Principles

### 1. Define Interfaces Before Implementation

Every major component in Guild starts with an interface definition:

```go
// pkg/providers/interface.go
package providers

import (
	"context"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Type returns the provider type
	Type() ProviderType

	// Models returns the available models
	Models() []string

	// Generate produces text from a prompt
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)

	// GenerateStream produces a stream of tokens
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan GenerateResponseChunk, error)

	// IsLocal returns true if this is a local provider
	IsLocal() bool

	// Cost returns the estimated cost for a request
	Cost(req GenerateRequest) float64
}
```

### 2. Keep Interfaces Focused and Small

Interfaces should have a clear responsibility and minimal method set:

```go
// Good: Focused interface
type TaskStorage interface {
	// Save persists a task
	Save(ctx context.Context, task Task) error

	// Get retrieves a task by ID
	Get(ctx context.Context, id string) (Task, error)

	// Update modifies an existing task
	Update(ctx context.Context, task Task) error

	// Delete removes a task
	Delete(ctx context.Context, id string) error
}

// Bad: Unfocused interface with too many responsibilities
type TaskSystem interface {
	Save(ctx context.Context, task Task) error
	Get(ctx context.Context, id string) (Task, error)
	Update(ctx context.Context, task Task) error
	Delete(ctx context.Context, id string) error
	AssignTaskToAgent(ctx context.Context, taskID, agentID string) error
	ExecuteTask(ctx context.Context, taskID string) (Result, error)
	NotifyCompletion(ctx context.Context, taskID string) error
	GenerateReportForTask(ctx context.Context, taskID string) (string, error)
}
```

### 3. Use Composition of Interfaces

Break complex interfaces into smaller ones that can be composed:

```go
// Small, focused interfaces
type Reader interface {
	Read(ctx context.Context, id string) ([]byte, error)
}

type Writer interface {
	Write(ctx context.Context, id string, data []byte) error
}

type Deleter interface {
	Delete(ctx context.Context, id string) error
}

// Composed interface
type Storage interface {
	Reader
	Writer
	Deleter
}
```

### 4. Accept Interfaces, Return Structs

Functions should accept interfaces and return concrete types:

```go
// Good: Accept interface, return concrete type
func NewBoltKanban(store memory.Store) (*BoltKanban, error) {
	// Implementation...
}

// Usage
store := memory.NewBoltDBStore("kanban.db")
kanban, err := NewBoltKanban(store)

// Bad: Accepting concrete type
func NewBoltKanban(store *BoltDBStore) (*BoltKanban, error) {
	// Implementation...
}
```

## Implementation Examples

### Provider Interface

```go
// pkg/providers/interface.go
package providers

import (
	"context"
)

// ProviderType identifies the LLM provider
type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderDeepseek  ProviderType = "deepseek"
	ProviderOra       ProviderType = "ora"
	ProviderOllama    ProviderType = "ollama"
)

// Provider defines the interface for LLM providers
type Provider interface {
	// Type returns the provider type
	Type() ProviderType

	// Models returns the available models
	Models() []string

	// Generate produces text from a prompt
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)

	// GenerateStream produces a stream of tokens
	GenerateStream(ctx context.Context, req GenerateRequest) (<-chan GenerateResponseChunk, error)

	// IsLocal returns true if this is a local provider
	IsLocal() bool

	// Cost returns the estimated cost for a request
	Cost(req GenerateRequest) float64
}

// GenerateRequest contains parameters for text generation
type GenerateRequest struct {
	// Model is the specific model to use
	Model string

	// Prompt is the input text
	Prompt string

	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int

	// Temperature controls randomness (0.0-1.0)
	Temperature float64

	// SystemPrompt is an optional system instruction
	SystemPrompt string

	// StopSequences are strings that stop generation
	StopSequences []string

	// AdditionalParams contains provider-specific parameters
	AdditionalParams map[string]interface{}
}

// GenerateResponse contains the result of text generation
type GenerateResponse struct {
	// Text is the generated text
	Text string

	// TokensUsed is the total tokens consumed
	TokensUsed int

	// FinishReason explains why generation stopped
	FinishReason string

	// Raw contains the raw provider response
	Raw interface{}
}

// GenerateResponseChunk contains a partial response
type GenerateResponseChunk struct {
	// Text is the generated text chunk
	Text string

	// IsFinal indicates the last chunk
	IsFinal bool

	// Error contains any error that occurred
	Error error
}

// ProviderConfig defines configuration for a provider
type ProviderConfig struct {
	// Type is the provider type
	Type ProviderType

	// Model is the specific model to use
	Model string

	// APIKey is the authentication key
	APIKey string

	// BaseURL is an optional custom endpoint
	BaseURL string

	// AdditionalParams contains provider-specific parameters
	AdditionalParams map[string]interface{}
}

// Factory creates providers
type Factory interface {
	// Create returns a provider for a configuration
	Create(config ProviderConfig) (Provider, error)

	// CreateWithAPIKey returns a provider with an API key
	CreateWithAPIKey(providerType ProviderType, model string, apiKey string) (Provider, error)
}
```

### OpenAI Implementation

```go
// pkg/providers/openai/client.go
package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/your-username/guild/pkg/providers"
)

// Client implements the Provider interface for OpenAI
type Client struct {
	client *openai.Client
	apiKey string
}

// NewClient creates a new OpenAI client
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	return &Client{
		client: openai.NewClient(apiKey),
		apiKey: apiKey,
	}, nil
}

// Type returns the provider type
func (c *Client) Type() providers.ProviderType {
	return providers.ProviderOpenAI
}

// Models returns the available models
func (c *Client) Models() []string {
	return []string{
		"gpt-4",
		"gpt-4-turbo",
		"gpt-3.5-turbo",
	}
}

// Generate produces text from a prompt
func (c *Client) Generate(ctx context.Context, req providers.GenerateRequest) (providers.GenerateResponse, error) {
	// Map to OpenAI request
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Prompt,
		},
	}

	// Create request
	openAIReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature),
	}

	// Add stop sequences if provided
	if len(req.StopSequences) > 0 {
		openAIReq.Stop = req.StopSequences
	}

	// Send request
	resp, err := c.client.CreateChatCompletion(ctx, openAIReq)
	if err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to generate completion: %w", err)
	}

	// Check for empty response
	if len(resp.Choices) == 0 {
		return providers.GenerateResponse{}, errors.New("no completion choices returned")
	}

	// Return response
	return providers.GenerateResponse{
		Text:         resp.Choices[0].Message.Content,
		TokensUsed:   resp.Usage.TotalTokens,
		FinishReason: string(resp.Choices[0].FinishReason),
		Raw:          resp,
	}, nil
}

// GenerateStream produces a stream of tokens
func (c *Client) GenerateStream(ctx context.Context, req providers.GenerateRequest) (<-chan providers.GenerateResponseChunk, error) {
	// Map to OpenAI request
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Prompt,
		},
	}

	// Create request
	openAIReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature),
		Stream:      true,
	}

	// Add stop sequences if provided
	if len(req.StopSequences) > 0 {
		openAIReq.Stop = req.StopSequences
	}

	// Create stream
	stream, err := c.client.CreateChatCompletionStream(ctx, openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create completion stream: %w", err)
	}

	// Create output channel
	outputCh := make(chan providers.GenerateResponseChunk)

	// Process stream in goroutine
	go func() {
		defer close(outputCh)
		defer stream.Close()

		for {
			// Get next response
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				// End of stream
				outputCh <- providers.GenerateResponseChunk{
					IsFinal: true,
				}
				return
			}

			if err != nil {
				// Stream error
				outputCh <- providers.GenerateResponseChunk{
					Error: err,
				}
				return
			}

			// Check for empty response
			if len(resp.Choices) == 0 {
				continue
			}

			// Send chunk
			outputCh <- providers.GenerateResponseChunk{
				Text: resp.Choices[0].Delta.Content,
			}
		}
	}()

	return outputCh, nil
}

// IsLocal returns true if this is a local provider
func (c *Client) IsLocal() bool {
	return false
}

// Cost returns the estimated cost for a request
func (c *Client) Cost(req providers.GenerateRequest) float64 {
	// Simplified cost estimation based on model and tokens
	tokenCost := 0.0
	switch {
	case strings.HasPrefix(req.Model, "gpt-4"):
		// GPT-4 pricing (approximate)
		tokenCost = 0.00003
	case strings.HasPrefix(req.Model, "gpt-3.5"):
		// GPT-3.5 pricing (approximate)
		tokenCost = 0.000002
	default:
		// Unknown model
		return 0.0
	}

	// Estimate token count (very rough approximation)
	promptTokens := len(req.Prompt) / 4
	if req.SystemPrompt != "" {
		promptTokens += len(req.SystemPrompt) / 4
	}

	// Add output tokens
	totalTokens := promptTokens
	if req.MaxTokens > 0 {
		totalTokens += req.MaxTokens
	} else {
		// Default output tokens if not specified
		totalTokens += promptTokens
	}

	return float64(totalTokens) * tokenCost
}
```

### Factory Implementation

```go
// pkg/providers/factory.go
package providers

import (
	"fmt"

	"github.com/your-username/guild/pkg/providers/anthropic"
	"github.com/your-username/guild/pkg/providers/ollama"
	"github.com/your-username/guild/pkg/providers/openai"
)

// ProviderFactoryImpl implements the Provider Factory
type ProviderFactoryImpl struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactoryImpl {
	return &ProviderFactoryImpl{}
}

// Create returns a provider for a configuration
func (f *ProviderFactoryImpl) Create(config ProviderConfig) (Provider, error) {
	switch config.Type {
	case ProviderOpenAI:
		return openai.NewClient(config.APIKey)
	case ProviderAnthropic:
		return anthropic.NewClient(config.APIKey, config.BaseURL)
	case ProviderOllama:
		return ollama.NewClient(config.BaseURL)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// CreateWithAPIKey returns a provider with an API key
func (f *ProviderFactoryImpl) CreateWithAPIKey(providerType ProviderType, model string, apiKey string) (Provider, error) {
	config := ProviderConfig{
		Type:   providerType,
		Model:  model,
		APIKey: apiKey,
	}
	return f.Create(config)
}
```

## Testing with Interfaces

Interfaces make testing much easier through mocks:

```go
// pkg/providers/mock/provider.go
package mock

import (
	"context"

	"github.com/your-username/guild/pkg/providers"
)

// Provider implements the Provider interface for testing
type Provider struct {
	TypeFunc           func() providers.ProviderType
	ModelsFunc         func() []string
	GenerateFunc       func(ctx context.Context, req providers.GenerateRequest) (providers.GenerateResponse, error)
	GenerateStreamFunc func(ctx context.Context, req providers.GenerateRequest) (<-chan providers.GenerateResponseChunk, error)
	IsLocalFunc        func() bool
	CostFunc           func(req providers.GenerateRequest) float64
}

// Type calls the mock TypeFunc
func (m *Provider) Type() providers.ProviderType {
	if m.TypeFunc != nil {
		return m.TypeFunc()
	}
	return providers.ProviderType("mock")
}

// Models calls the mock ModelsFunc
func (m *Provider) Models() []string {
	if m.ModelsFunc != nil {
		return m.ModelsFunc()
	}
	return []string{"mock-model"}
}

// Generate calls the mock GenerateFunc
func (m *Provider) Generate(ctx context.Context, req providers.GenerateRequest) (providers.GenerateResponse, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, req)
	}
	return providers.GenerateResponse{
		Text:       "Mock response",
		TokensUsed: 10,
	}, nil
}

// GenerateStream calls the mock GenerateStreamFunc
func (m *Provider) GenerateStream(ctx context.Context, req providers.GenerateRequest) (<-chan providers.GenerateResponseChunk, error) {
	if m.GenerateStreamFunc != nil {
		return m.GenerateStreamFunc(ctx, req)
	}

	ch := make(chan providers.GenerateResponseChunk, 1)
	ch <- providers.GenerateResponseChunk{
		Text:    "Mock response",
		IsFinal: true,
	}
	close(ch)

	return ch, nil
}

// IsLocal calls the mock IsLocalFunc
func (m *Provider) IsLocal() bool {
	if m.IsLocalFunc != nil {
		return m.IsLocalFunc()
	}
	return true
}

// Cost calls the mock CostFunc
func (m *Provider) Cost(req providers.GenerateRequest) float64 {
	if m.CostFunc != nil {
		return m.CostFunc(req)
	}
	return 0.0
}
```

### Testing Example

```go
// pkg/agent/agent_test.go
package agent_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/your-username/guild/pkg/agent"
	"github.com/your-username/guild/pkg/kanban"
	"github.com/your-username/guild/pkg/providers"
	"github.com/your-username/guild/pkg/providers/mock"
)

func TestAgentExecution(t *testing.T) {
	// Create mock provider
	mockProvider := &mock.Provider{
		GenerateFunc: func(ctx context.Context, req providers.GenerateRequest) (providers.GenerateResponse, error) {
			// Assert on request parameters
			assert.Contains(t, req.Prompt, "Generate a hello world program")

			// Return mock response
			return providers.GenerateResponse{
				Text: `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
				TokensUsed: 50,
			}, nil
		},
	}

	// Create mock board
	mockBoard := &kanban.MockBoard{}

	// Create agent
	agent := agent.NewBasicAgent("test-agent", mockProvider, mockBoard)

	// Create task
	task := kanban.Task{
		ID:          "task-1",
		Title:       "Hello World",
		Description: "Generate a hello world program in Go",
		Status:      kanban.StatusToDo,
	}

	// Execute task
	ctx := context.Background()
	result, err := agent.Execute(ctx, task)

	// Assert no error
	assert.NoError(t, err)

	// Assert result
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "package main")
	assert.Contains(t, result.Output, "Hello, World!")
	assert.Equal(t, 50, result.TokensUsed)
}
```

## Best Practices

1. **Define interfaces in separate files**

   - Keep interface definitions separate from implementations
   - Place interfaces at package level for visibility

2. **Focus on behavior, not structure**

   - Interfaces should describe actions, not attributes
   - Start interface names with verbs when appropriate (e.g., `Reader`, not `HasRead`)

3. **Only define what you need**

   - Don't add methods you don't currently use
   - Keep interfaces minimal and cohesive

4. **Use embedding for common interfaces**

   - Embed standard interfaces where possible (e.g., `io.Reader`, `io.Writer`)
   - Create hierarchies of interfaces through embedding

5. **Ensure interface compatibility**
   - Use compiler checks to verify implementations satisfy interfaces:
     ```go
     var _ providers.Provider = (*openai.Client)(nil)
     ```

## Related Documentation

- [Go Interfaces](https://golang.org/doc/effective_go.html#interfaces)
- [../patterns/go_concurrency.md](../patterns/go_concurrency.md)
- [../patterns/prompt_chain_memory.md](../patterns/prompt_chain_memory.md)
