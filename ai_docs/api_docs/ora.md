# Ora API Integration

This document explains how to integrate with the Ora API in Guild.

## Overview

Ora provides Resilient Model Services (RMS) that offer managed access to a variety of large language models with built-in redundancy, retries, and cost optimization. Guild integrates with Ora to provide reliable access to LLMs while minimizing cost and latency.

## Key Benefits

1. **Reliability**: Automatic retries and fallbacks between model providers
2. **Cost Optimization**: Intelligent routing to most cost-effective models
3. **Unified API**: Access to multiple providers through a single interface
4. **Model Selection**: Automatic selection of appropriate models for each task

## Authentication

1. **API Key**

   - Get an API key from [Ora.ai](https://ora.ai)
   - Store in `.env` file or environment variable: `ORA_API_KEY`
   - Never hardcode API keys in source code

2. **Configuration**
   ```yaml
   providers:
     ora:
       api_key: ${ORA_API_KEY}
       base_url: https://api.ora.ai/v1
   ```

## Provider Implementation

```go
// pkg/providers/ora/client.go
package ora

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/your-username/guild/pkg/providers"
)

// Client implements the Provider interface for Ora
type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewClient creates a new Ora client
func NewClient(apiKey, baseURL string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if baseURL == "" {
		baseURL = "https://api.ora.ai/v1"
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Type returns the provider type
func (c *Client) Type() providers.ProviderType {
	return providers.ProviderOra
}

// Models returns the available models
func (c *Client) Models() []string {
	// Ora supports these model tiers
	return []string{
		"ora-fast",     // Fastest models
		"ora-balanced", // Balance of speed and quality
		"ora-best",     // Best quality models
		"gpt-3.5-turbo",
		"gpt-4",
		"claude-3-opus",
		"claude-3-sonnet",
		"claude-3-haiku",
	}
}

// Generate produces text from a prompt
func (c *Client) Generate(ctx context.Context, req providers.GenerateRequest) (providers.GenerateResponse, error) {
	// Set default model if not provided
	model := req.Model
	if model == "" {
		model = "ora-balanced"
	}

	// Convert providers.GenerateRequest to Ora request format
	messages := []Message{
		{
			Role:    "user",
			Content: req.Prompt,
		},
	}

	// Add system message if provided
	if req.SystemPrompt != "" {
		messages = append([]Message{
			{
				Role:    "system",
				Content: req.SystemPrompt,
			},
		}, messages...)
	}

	// Create Ora request
	oraReq := ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: req.Temperature,
	}

	// Set max tokens if provided
	if req.MaxTokens > 0 {
		oraReq.MaxTokens = req.MaxTokens
	}

	// Add stop sequences if provided
	if len(req.StopSequences) > 0 {
		oraReq.Stop = req.StopSequences
	}

	// Convert to JSON
	jsonData, err := json.Marshal(oraReq)
	if err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/chat/completions", c.baseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if err := json.Unmarshal(body, &errorResp); err != nil {
			return providers.GenerateResponse{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}
		return providers.GenerateResponse{}, fmt.Errorf("API error (%d): %s - %s", resp.StatusCode, errorResp.Error.Type, errorResp.Error.Message)
	}

	// Parse response
	var completionResp ChatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Verify we have at least one choice
	if len(completionResp.Choices) == 0 {
		return providers.GenerateResponse{}, fmt.Errorf("no completions returned")
	}

	// Return result
	return providers.GenerateResponse{
		Text:         completionResp.Choices[0].Message.Content,
		TokensUsed:   completionResp.Usage.TotalTokens,
		FinishReason: completionResp.Choices[0].FinishReason,
		Raw:          completionResp,
	}, nil
}

// GenerateStream produces a stream of tokens
func (c *Client) GenerateStream(ctx context.Context, req providers.GenerateRequest) (<-chan providers.GenerateResponseChunk, error) {
	// Set default model if not provided
	model := req.Model
	if model == "" {
		model = "ora-balanced"
	}

	// Convert providers.GenerateRequest to Ora request format
	messages := []Message{
		{
			Role:    "user",
			Content: req.Prompt,
		},
	}

	// Add system message if provided
	if req.SystemPrompt != "" {
		messages = append([]Message{
			{
				Role:    "system",
				Content: req.SystemPrompt,
			},
		}, messages...)
	}

	// Create Ora request
	oraReq := ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		Temperature: req.Temperature,
		Stream:      true,
	}

	// Set max tokens if provided
	if req.MaxTokens > 0 {
		oraReq.MaxTokens = req.MaxTokens
	}

	// Add stop sequences if provided
	if len(req.StopSequences) > 0 {
		oraReq.Stop = req.StopSequences
	}

	// Convert to JSON
	jsonData, err := json.Marshal(oraReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/chat/completions", c.baseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	httpReq.Header.Set("Accept", "text/event-stream")

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Create output channel
	outputCh := make(chan providers.GenerateResponseChunk)

	// Process stream in goroutine
	go func() {
		defer close(outputCh)
		defer resp.Body.Close()

		// Create scanner for SSE
		scanner := bufio.NewScanner(resp.Body)

		// Process events
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines
			if line == "" {
				continue
			}

			// Check for data prefix
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			// Extract data
			data := strings.TrimPrefix(line, "data: ")

			// Check for stream end
			if data == "[DONE]" {
				outputCh <- providers.GenerateResponseChunk{
					IsFinal: true,
				}
				break
			}

			// Parse chunk
			var chunk ChatCompletionChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				outputCh <- providers.GenerateResponseChunk{
					Error: fmt.Errorf("failed to parse chunk: %w", err),
				}
				break
			}

			// Skip empty choices
			if len(chunk.Choices) == 0 {
				continue
			}

			// Send content delta
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				outputCh <- providers.GenerateResponseChunk{
					Text: content,
				}
			}

			// Check for finish
			if chunk.Choices[0].FinishReason != "" {
				outputCh <- providers.GenerateResponseChunk{
					IsFinal: true,
				}
				break
			}
		}

		// Check for scanner errors
		if err := scanner.Err(); err != nil {
			outputCh <- providers.GenerateResponseChunk{
				Error: fmt.Errorf("scanner error: %w", err),
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
	// Note: Ora may have different pricing, this is just an approximation
	costPerToken := 0.0

	// Set default model if not provided
	model := req.Model
	if model == "" {
		model = "ora-balanced"
	}

	switch model {
	case "ora-best", "gpt-4", "claude-3-opus":
		costPerToken = 30.0 / 1000000 // $0.03 per 1K tokens (approximation)
	case "ora-balanced", "claude-3-sonnet":
		costPerToken = 15.0 / 1000000 // $0.015 per 1K tokens (approximation)
	case "ora-fast", "gpt-3.5-turbo", "claude-3-haiku":
		costPerToken = 3.0 / 1000000 // $0.003 per 1K tokens (approximation)
	default:
		costPerToken = 15.0 / 1000000 // Default to balanced tier pricing
	}

	// Estimate token count (very rough approximation)
	promptTokens := len(req.Prompt) / 4
	if req.SystemPrompt != "" {
		promptTokens += len(req.SystemPrompt) / 4
	}

	// Add output tokens
	outputTokens := 0
	if req.MaxTokens > 0 {
		outputTokens = req.MaxTokens
	} else {
		// Default output tokens if not specified
		outputTokens = promptTokens
	}

	totalTokens := promptTokens + outputTokens
	return float64(totalTokens) * costPerToken
}

// Request/Response Types

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest is the request body for the Ora API
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	N           int       `json:"n,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// ChatCompletionResponse is the response from the Ora API
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage contains token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk is a streaming response chunk
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

// ChunkChoice represents a streaming choice
type ChunkChoice struct {
	Index        int         `json:"index"`
	Delta        ChunkDelta  `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

// ChunkDelta contains the content delta
type ChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ErrorResponse represents an error from the API
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}
```

## Model Selection

Ora provides three tiers of models:

| Model Tier     | Description                  | Use Cases                         | Response Time  |
| -------------- | ---------------------------- | --------------------------------- | -------------- |
| `ora-fast`     | Optimized for speed          | RAG, classification, simple Q&A   | Fast (< 1s)    |
| `ora-balanced` | Balance of speed and quality | General purpose, content creation | Medium (1-3s)  |
| `ora-best`     | Highest quality responses    | Complex reasoning, creative tasks | Slower (3-5s+) |

Ora also supports direct access to specific models:

| Model             | Provider  | Strengths                            |
| ----------------- | --------- | ------------------------------------ |
| `gpt-4`           | OpenAI    | Coding, reasoning, general knowledge |
| `gpt-3.5-turbo`   | OpenAI    | Fast responses, general tasks        |
| `claude-3-opus`   | Anthropic | Long contexts, instructions, safety  |
| `claude-3-sonnet` | Anthropic | Good balance of speed and quality    |
| `claude-3-haiku`  | Anthropic | Fast responses, routine tasks        |

## Automatic Fallback

One of Ora's key features is automatic fallback between providers:

```go
// Example: Ora will automatically handle fallbacks between providers
resp, err := oraClient.Generate(ctx, providers.GenerateRequest{
	Model:       "ora-best", // Will try to use the best available model
	Prompt:      "Explain quantum computing in simple terms",
	MaxTokens:   1000,
	Temperature: 0.7,
})

// If the primary model fails, Ora will try alternatives automatically
if err != nil {
	// This will only happen if all fallback options have failed
	log.Printf("All model attempts failed: %v", err)
}
```

## Request Optimization

### Model Selection Strategy

```go
// Example: Select model based on task complexity
func selectModel(task Task) string {
	switch task.Complexity {
	case ComplexityHigh:
		return "ora-best" // Use highest quality for complex tasks
	case ComplexityMedium:
		return "ora-balanced" // Use balanced for medium complexity
	case ComplexityLow:
		return "ora-fast" // Use fastest for simple tasks
	default:
		return "ora-balanced" // Default to balanced
	}
}

// Example usage
model := selectModel(task)
resp, err := oraClient.Generate(ctx, providers.GenerateRequest{
	Model:  model,
	Prompt: task.Description,
})
```

### Streaming for Better UX

```go
// Example: Streaming response for better user experience
stream, err := oraClient.GenerateStream(ctx, providers.GenerateRequest{
	Model:  "ora-balanced",
	Prompt: "Generate a step-by-step guide for learning Go programming",
})

if err != nil {
	log.Fatalf("Failed to start stream: %v", err)
}

// Process stream
var fullText strings.Builder
for chunk := range stream {
	if chunk.Error != nil {
		log.Printf("Stream error: %v", chunk.Error)
		break
	}

	// Update UI with new content
	fmt.Print(chunk.Text)
	fullText.WriteString(chunk.Text)

	// Check if final
	if chunk.IsFinal {
		fmt.Println("\n--- End of response ---")
	}
}

// Store full response
response := fullText.String()
```

## Error Handling

Common errors and their solutions:

| Error                     | Cause               | Solution                              |
| ------------------------- | ------------------- | ------------------------------------- |
| 401 Unauthorized          | Invalid API key     | Check environment variables           |
| 429 Too Many Requests     | Rate limit exceeded | Implement backoff                     |
| 400 Bad Request           | Invalid parameters  | Check model name and parameter values |
| 500 Internal Server Error | Server-side issue   | Retry with backoff                    |
| 504 Gateway Timeout       | Request timeout     | Reduce request complexity or size     |

## Retry Strategy

Ora has built-in retries, but you can implement additional resilience:

```go
// retry performs an operation with exponential backoff
func retry(ctx context.Context, maxRetries int, operation func() error) error {
	backoff := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		err := operation()
		if err == nil {
			return nil
		}

		// Check for non-retryable errors
		if !isRetryable(err) {
			return err
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Increase backoff exponentially
			backoff = time.Duration(float64(backoff) * 1.5)
		}
	}

	return fmt.Errorf("exceeded maximum retries")
}

// isRetryable determines if an error should be retried
func isRetryable(err error) bool {
	// Retry on rate limiting and server errors
	return strings.Contains(err.Error(), "429") ||
		strings.Contains(err.Error(), "500") ||
		strings.Contains(err.Error(), "503") ||
		strings.Contains(err.Error(), "504")
}
```

## Cost Management

Ora provides cost optimization, but you can implement additional controls:

```go
// Token budget enforcement
type TokenBudget struct {
	MaxTokens    int
	UsedTokens   int
	AlertPercent float64
}

// TrackUsage updates token usage and returns true if budget exceeded
func (b *TokenBudget) TrackUsage(used int) bool {
	b.UsedTokens += used

	// Alert when reaching threshold
	if float64(b.UsedTokens) / float64(b.MaxTokens) >= b.AlertPercent {
		log.Printf("Warning: Token budget at %.1f%% (%d/%d tokens)",
			float64(b.UsedTokens) / float64(b.MaxTokens) * 100,
			b.UsedTokens, b.MaxTokens)
	}

	return b.UsedTokens >= b.MaxTokens
}

// Example usage
budget := &TokenBudget{
	MaxTokens:    100000, // 100K tokens
	AlertPercent: 0.8,    // Alert at 80%
}

resp, err := oraClient.Generate(ctx, req)
if err != nil {
	return err
}

// Track usage
if budget.TrackUsage(resp.TokensUsed) {
	log.Println("Token budget exceeded, restricting further requests")
	// Implement restrictions
}
```

## Best Practices

1. **Use Model Tiers**

   - Use `ora-fast` for simple tasks that need quick responses
   - Use `ora-balanced` for most general tasks
   - Reserve `ora-best` for complex reasoning or creative tasks

2. **Optimize Prompts**

   - Keep prompts clear and concise
   - Use system prompts to guide model behavior
   - Structure prompts with clear expectations

3. **Handle Streaming Properly**

   - Always close response channels and connections
   - Implement proper error handling for stream interruptions
   - Consider UI updates for streaming responses

4. **Manage Costs**

   - Monitor token usage
   - Set and enforce budgets
   - Use lower tier models for initial drafts

5. **Implement Graceful Degradation**
   - Prepare fallback logic for when Ora service is unavailable
   - Consider using local models as ultimate fallback

## Tool Integration

Ora supports function calling similar to OpenAI:

```go
// Example: Function calling with Ora
functionDefinitions := []map[string]interface{}{
	{
		"name": "get_weather",
		"description": "Get the weather in a location",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type": "string",
					"description": "The city and country",
				},
				"unit": map[string]interface{}{
					"type": "string",
					"enum": []string{"celsius", "fahrenheit"},
				},
			},
			"required": []string{"location"},
		},
	},
}

// Add to additional params
additionalParams := map[string]interface{}{
	"functions": functionDefinitions,
	"function_call": "auto",
}

// Create request
req := providers.GenerateRequest{
	Model:           "ora-best",
	Prompt:          "What's the weather in London?",
	AdditionalParams: additionalParams,
}

// Parse function calls from response
// Implementation depends on how function calls are structured in the response
```

## Related Documentation

- [Ora API Documentation](https://docs.ora.io/doc/resilient-model-services-rms/ora-api)
- [Ora Pricing](https://ora.ai/pricing)
- [../patterns/prompt_chain_memory.md](../patterns/prompt_chain_memory.md)
- [../architecture/agent_lifecycle.md](../architecture/agent_lifecycle.md)
