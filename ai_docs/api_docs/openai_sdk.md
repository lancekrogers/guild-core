# OpenAI SDK Integration

This document explains how to integrate the OpenAI API in Guild.

## Overview

The OpenAI API provides access to models like GPT-4 and GPT-3.5. Guild integrates with these models through the OpenAI API client.

## Authentication

1. **API Key**

   - Get API key from OpenAI
   - Store in `.env` file or environment variable: `OPENAI_API_KEY`
   - Never hardcode API keys in source code

2. **Configuration**
   ```yaml
   providers:
     openai:
       api_key: ${OPENAI_API_KEY}
       organization: ${OPENAI_ORG_ID} # optional
   ```

## Provider Implementation

```go
// pkg/providers/openai/client.go
package openai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/your-username/guild/pkg/providers"
)

// Client implements the Provider interface for OpenAI
type Client struct {
	client *openai.Client
	config Config
}

// Config contains OpenAI configuration
type Config struct {
	APIKey       string
	Organization string
	BaseURL      string
}

// NewClient creates a new OpenAI client
func NewClient(config Config) (*Client, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	options := []openai.ClientOption{}

	if config.Organization != "" {
		options = append(options, openai.WithOrganization(config.Organization))
	}

	if config.BaseURL != "" {
		options = append(options, openai.WithBaseURL(config.BaseURL))
	}

	client := openai.NewClientWithOptions(config.APIKey, options...)

	return &Client{
		client: client,
		config: config,
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
	// Implementation details...
}

// GenerateStream produces a stream of tokens
func (c *Client) GenerateStream(ctx context.Context, req providers.GenerateRequest) (<-chan providers.GenerateResponseChunk, error) {
	// Implementation details...
}

// IsLocal returns true if this is a local provider
func (c *Client) IsLocal() bool {
	return false
}

// Cost returns the estimated cost for a request
func (c *Client) Cost(req providers.GenerateRequest) float64 {
	// Cost calculation based on model and tokens
}
```

## Model Parameters

| Parameter     | Description         | Default | Valid Range     |
| ------------- | ------------------- | ------- | --------------- |
| `model`       | Model name          | `gpt-4` | See models list |
| `temperature` | Randomness          | 0.7     | 0.0-2.0         |
| `max_tokens`  | Max response length | 1024    | 1-8192          |

## Function Calling

OpenAI models support function calling, which is useful for tool integration:

```go
func (c *Client) Generate(ctx context.Context, req providers.GenerateRequest) (providers.GenerateResponse, error) {
	messages := []openai.ChatMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Prompt,
		},
	}

	functions := []openai.FunctionDefinition{
		{
			Name:        "search_web",
			Description: "Search the web for information",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
				},
				"required": []string{"query"},
			},
		},
	}

	response, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     req.Model,
		Messages:  messages,
		Functions: functions,
	})

	// Handle response...
}
```

## Error Handling

Common errors and their solutions:

| Error                 | Cause               | Solution                    |
| --------------------- | ------------------- | --------------------------- |
| 401 Unauthorized      | Invalid API key     | Check environment variables |
| 429 Too Many Requests | Rate limit exceeded | Implement backoff           |
| 400 Bad Request       | Invalid input       | Check token limits          |

## Related Documentation

- [API Reference](https://platform.openai.com/docs/api-reference)
- [../patterns/prompt_chain_memory.md](../patterns/prompt_chain_memory.md)
