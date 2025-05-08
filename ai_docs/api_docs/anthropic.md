# Anthropic API Integration

This document explains how to integrate with the Anthropic API in Guild.

## Overview

The Anthropic API provides access to Claude models like Claude 3 Opus, Claude 3 Sonnet, and Claude 3 Haiku. Guild integrates with these models through the Anthropic API client.

## Authentication

1. **API Key**

   - Get API key from Anthropic
   - Store in `.env` file or environment variable: `ANTHROPIC_API_KEY`
   - Never hardcode API keys in source code

2. **Configuration**

   ```yaml
   providers:
     anthropic:
       api_key: ${ANTHROPIC_API_KEY}
       base_url: https://api.anthropic.com/v1
   ```

## Provider Implementation

```go
// pkg/providers/anthropic/client.go
package anthropic

import (
 "context"
 "encoding/json"
 "fmt"
 "io"
 "net/http"
 "strings"
 "time"

 "github.com/your-username/guild/pkg/providers"
)

// Client implements the Provider interface for Anthropic's Claude API
type Client struct {
 apiKey  string
 baseURL string
 client  *http.Client
}

// NewClient creates a new Anthropic client
func NewClient(apiKey, baseURL string) (*Client, error) {
 if apiKey == "" {
  return nil, fmt.Errorf("API key is required")
 }

 if baseURL == "" {
  baseURL = "https://api.anthropic.com/v1"
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
 return providers.ProviderAnthropic
}

// Models returns the available models
func (c *Client) Models() []string {
 return []string{
  "claude-3-opus-20240229",
  "claude-3-sonnet-20240229",
  "claude-3-haiku-20240307",
 }
}

// Generate produces text from a prompt
func (c *Client) Generate(ctx context.Context, req providers.GenerateRequest) (providers.GenerateResponse, error) {
 // Build message
 messages := []Message{
  {
   Role:    "user",
   Content: req.Prompt,
  },
 }

 // Add system prompt if provided
 system := ""
 if req.SystemPrompt != "" {
  system = req.SystemPrompt
 }

 // Set default model if not provided
 model := req.Model
 if model == "" {
  model = "claude-3-sonnet-20240229"
 }

 // Set token limit if provided
 maxTokens := 1024
 if req.MaxTokens > 0 {
  maxTokens = req.MaxTokens
 }

 // Create request
 reqBody := CompletionRequest{
  Model:       model,
  Messages:    messages,
  System:      system,
  MaxTokens:   maxTokens,
  Temperature: req.Temperature,
 }

 // Convert to JSON
 jsonData, err := json.Marshal(reqBody)
 if err != nil {
  return providers.GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
 }

 // Create HTTP request
 httpReq, err := http.NewRequestWithContext(
  ctx,
  "POST",
  fmt.Sprintf("%s/messages", c.baseURL),
  strings.NewReader(string(jsonData)),
 )
 if err != nil {
  return providers.GenerateResponse{}, fmt.Errorf("failed to create request: %w", err)
 }

 // Set headers
 httpReq.Header.Set("Content-Type", "application/json")
 httpReq.Header.Set("x-api-key", c.apiKey)
 httpReq.Header.Set("anthropic-version", "2023-06-01")

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
  return providers.GenerateResponse{}, fmt.Errorf("API error (%d): %s - %s", resp.StatusCode, errorResp.Type, errorResp.Error.Message)
 }

 // Parse response
 var completionResp CompletionResponse
 if err := json.Unmarshal(body, &completionResp); err != nil {
  return providers.GenerateResponse{}, fmt.Errorf("failed to parse response: %w", err)
 }

 // Return result
 return providers.GenerateResponse{
  Text:         completionResp.Content[0].Text,
  TokensUsed:   completionResp.Usage.InputTokens + completionResp.Usage.OutputTokens,
  FinishReason: completionResp.StopReason,
  Raw:          completionResp,
 }, nil
}

// GenerateStream produces a stream of tokens
func (c *Client) GenerateStream(ctx context.Context, req providers.GenerateRequest) (<-chan providers.GenerateResponseChunk, error) {
 // Build message
 messages := []Message{
  {
   Role:    "user",
   Content: req.Prompt,
  },
 }

 // Add system prompt if provided
 system := ""
 if req.SystemPrompt != "" {
  system = req.SystemPrompt
 }

 // Set default model if not provided
 model := req.Model
 if model == "" {
  model = "claude-3-sonnet-20240229"
 }

 // Set token limit if provided
 maxTokens := 1024
 if req.MaxTokens > 0 {
  maxTokens = req.MaxTokens
 }

 // Create request
 reqBody := CompletionRequest{
  Model:       model,
  Messages:    messages,
  System:      system,
  MaxTokens:   maxTokens,
  Temperature: req.Temperature,
  Stream:      true,
 }

 // Convert to JSON
 jsonData, err := json.Marshal(reqBody)
 if err != nil {
  return nil, fmt.Errorf("failed to marshal request: %w", err)
 }

 // Create HTTP request
 httpReq, err := http.NewRequestWithContext(
  ctx,
  "POST",
  fmt.Sprintf("%s/messages", c.baseURL),
  strings.NewReader(string(jsonData)),
 )
 if err != nil {
  return nil, fmt.Errorf("failed to create request: %w", err)
 }

 // Set headers
 httpReq.Header.Set("Content-Type", "application/json")
 httpReq.Header.Set("x-api-key", c.apiKey)
 httpReq.Header.Set("anthropic-version", "2023-06-01")
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

  // Create stream reader
  reader := NewSSEReader(resp.Body)

  // Process events
  for {
   // Get next event
   event, err := reader.ReadEvent()
   if err != nil {
    if err != io.EOF {
     outputCh <- providers.GenerateResponseChunk{
      Error: fmt.Errorf("stream error: %w", err),
     }
    }
    break
   }

   // Skip non-data events
   if event.Event != "completion" {
    if event.Event == "error" {
     var errorResp ErrorResponse
     if err := json.Unmarshal([]byte(event.Data), &errorResp); err == nil {
      outputCh <- providers.GenerateResponseChunk{
       Error: fmt.Errorf("stream error: %s - %s", errorResp.Type, errorResp.Error.Message),
      }
     }
    }
    continue
   }

   // Parse data
   var streamResp StreamResponse
   if err := json.Unmarshal([]byte(event.Data), &streamResp); err != nil {
    outputCh <- providers.GenerateResponseChunk{
     Error: fmt.Errorf("failed to parse stream data: %w", err),
    }
    break
   }

   // Send chunk
   if streamResp.Type == "content_block_delta" && streamResp.Delta.Type == "text_delta" {
    outputCh <- providers.GenerateResponseChunk{
     Text: streamResp.Delta.Text,
    }
   }

   // Check for end of stream
   if streamResp.Type == "message_stop" {
    outputCh <- providers.GenerateResponseChunk{
     IsFinal: true,
    }
    break
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
 inputCost := 0.0
 outputCost := 0.0

 switch req.Model {
 case "claude-3-opus-20240229":
  inputCost = 15.0 / 1000000  // $15.00 per 1M input tokens
  outputCost = 75.0 / 1000000 // $75.00 per 1M output tokens
 case "claude-3-sonnet-20240229":
  inputCost = 3.0 / 1000000   // $3.00 per 1M input tokens
  outputCost = 15.0 / 1000000 // $15.00 per 1M output tokens
 case "claude-3-haiku-20240307":
  inputCost = 0.25 / 1000000  // $0.25 per 1M input tokens
  outputCost = 1.25 / 1000000 // $1.25 per 1M output tokens
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
 outputTokens := 0
 if req.MaxTokens > 0 {
  outputTokens = req.MaxTokens
 } else {
  // Default output tokens if not specified
  outputTokens = promptTokens
 }

 // Calculate total cost
 inputTokenCost := float64(promptTokens) * inputCost
 outputTokenCost := float64(outputTokens) * outputCost

 return inputTokenCost + outputTokenCost
}

// Message represents a message in a conversation
type Message struct {
 Role    string `json:"role"`
 Content string `json:"content"`
}

// CompletionRequest is the request body for the Messages API
type CompletionRequest struct {
 Model       string    `json:"model"`
 Messages    []Message `json:"messages"`
 System      string    `json:"system,omitempty"`
 MaxTokens   int       `json:"max_tokens,omitempty"`
 Temperature float64   `json:"temperature,omitempty"`
 Stream      bool      `json:"stream,omitempty"`
}

// ContentBlock represents a block of content in a response
type ContentBlock struct {
 Type string `json:"type"`
 Text string `json:"text"`
}

// Usage contains token usage information
type Usage struct {
 InputTokens  int `json:"input_tokens"`
 OutputTokens int `json:"output_tokens"`
}

// CompletionResponse is the response from the Messages API
type CompletionResponse struct {
 ID         string         `json:"id"`
 Type       string         `json:"type"`
 Role       string         `json:"role"`
 Content    []ContentBlock `json:"content"`
 Model      string         `json:"model"`
 StopReason string         `json:"stop_reason"`
 Usage      Usage          `json:"usage"`
}

// TextDelta represents a text delta in a stream response
type TextDelta struct {
 Type string `json:"type"`
 Text string `json:"text"`
}

// StreamResponse is a response chunk from a streaming request
type StreamResponse struct {
 Type       string    `json:"type"`
 Message    string    `json:"message"`
 Delta      TextDelta `json:"delta"`
 StopReason string    `json:"stop_reason,omitempty"`
}

// ErrorDetail contains details about an error
type ErrorDetail struct {
 Type    string `json:"type"`
 Message string `json:"message"`
}

// ErrorResponse is the error response from the API
type ErrorResponse struct {
 Type  string      `json:"type"`
 Error ErrorDetail `json:"error"`
}

// SSEEvent represents a server-sent event
type SSEEvent struct {
 Event string
 Data  string
}

// SSEReader reads server-sent events
type SSEReader struct {
 reader  io.Reader
 buffer  []byte
 lastPos int
}

// NewSSEReader creates a new SSE reader
func NewSSEReader(reader io.Reader) *SSEReader {
 return &SSEReader{
  reader:  reader,
  buffer:  make([]byte, 0),
  lastPos: 0,
 }
}

// ReadEvent reads the next event
func (r *SSEReader) ReadEvent() (*SSEEvent, error) {
 event := &SSEEvent{}
 inEvent := false
 eventDone := false

 for !eventDone {
  // Check if we need to read more data
  if r.lastPos >= len(r.buffer) {
   // Read more data
   newData := make([]byte, 1024)
   n, err := r.reader.Read(newData)
   if err != nil {
    if err == io.EOF && inEvent {
     // End of stream but we have a partial event
     eventDone = true
    } else {
     // Error or end of stream without an event
     return nil, err
    }
   }

   // Append new data to buffer
   if n > 0 {
    r.buffer = append(r.buffer, newData[:n]...)
   }
  }

  // Find next newline
  nlPos := bytes.IndexByte(r.buffer[r.lastPos:], '\n')
  if nlPos < 0 {
   // No complete line yet
   if !inEvent {
    // Not in an event, so we need more data
    r.lastPos = len(r.buffer)
    continue
   } else {
    // In an event, use all remaining data
    nlPos = len(r.buffer) - r.lastPos
    eventDone = true
   }
  } else {
   // Convert to absolute position
   nlPos += r.lastPos
  }

  // Extract line
  line := string(r.buffer[r.lastPos:nlPos])
  r.lastPos = nlPos + 1

  // Skip empty lines
  if line == "" {
   if inEvent {
    // Empty line terminates an event
    eventDone = true
   }
   continue
  }

  // Parse line
  inEvent = true
  if strings.HasPrefix(line, "event:") {
   event.Event = strings.TrimSpace(line[6:])
  } else if strings.HasPrefix(line, "data:") {
   if event.Data != "" {
    event.Data += "\n"
   }
   event.Data += strings.TrimSpace(line[5:])
  }
 }

 // Remove buffer data that we've processed
 if r.lastPos > 0 {
  r.buffer = r.buffer[r.lastPos:]
  r.lastPos = 0
 }

 return event, nil
}
```

## API Reference

### Models

Claude 3 family models:

| Model                    | Description                           | Input Cost       | Output Cost      | Context Length |
| ------------------------ | ------------------------------------- | ---------------- | ---------------- | -------------- |
| claude-3-opus-20240229   | Most powerful model for complex tasks | $15.00/1M tokens | $75.00/1M tokens | 200K tokens    |
| claude-3-sonnet-20240229 | Balance of intelligence and speed     | $3.00/1M tokens  | $15.00/1M tokens | 200K tokens    |
| claude-3-haiku-20240307  | Fastest and most efficient model      | $0.25/1M tokens  | $1.25/1M tokens  | 200K tokens    |

### Messages API

The Messages API is the primary interface for interacting with Claude models:

```json
{
  "model": "claude-3-opus-20240229",
  "messages": [
    {
      "role": "user",
      "content": "Hello, Claude!"
    }
  ],
  "system": "You are a helpful AI assistant that provides concise responses.",
  "max_tokens": 1024,
  "temperature": 0.7
}
```

### Response Format

Standard response format:

```json
{
  "id": "msg_012345abcdef",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I assist you today?"
    }
  ],
  "model": "claude-3-opus-20240229",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 5,
    "output_tokens": 9
  }
}
```

Streaming response format:

```
event: completion
data: {"type":"content_block_start","index":0,"content_block":{"type":"text"}}

event: completion
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: completion
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}

event: completion
data: {"type":"message_stop","message":{"id":"msg_012345abcdef","type":"message","role":"assistant","content":[{"type":"text","text":"Hello!"}],"model":"claude-3-opus-20240229","stop_reason":"end_turn"}}
```

## Error Handling

Common errors and their solutions:

| Error                     | Cause               | Solution                              |
| ------------------------- | ------------------- | ------------------------------------- |
| 401 Unauthorized          | Invalid API key     | Check environment variables           |
| 429 Too Many Requests     | Rate limit exceeded | Implement exponential backoff         |
| 400 Bad Request           | Invalid parameters  | Check model name and parameter values |
| 500 Internal Server Error | Server-side issue   | Retry with backoff                    |

## Retry Strategy

Implement an exponential backoff strategy for retrying failed requests:

```go
// retry performs an operation with exponential backoff
func retry(ctx context.Context, maxRetries int, operation func() error) error {
 var err error

 // Initial backoff: 1 second
 backoff := 1 * time.Second

 // Retry loop
 for i := 0; i < maxRetries; i++ {
  // Attempt operation
  err = operation()
  if err == nil {
   // Success
   return nil
  }

  // Check if we should retry
  if !isRetryable(err) {
   return err
  }

  // Check context cancellation
  select {
  case <-ctx.Done():
   return ctx.Err()
  case <-time.After(backoff):
   // Increase backoff for next retry
   backoff *= 2
  }
 }

 return fmt.Errorf("maximum retries exceeded: %w", err)
}

// isRetryable determines if an error should be retried
func isRetryable(err error) bool {
 // Retry on network errors and certain API errors
 if strings.Contains(err.Error(), "connection refused") ||
    strings.Contains(err.Error(), "timeout") ||
    strings.Contains(err.Error(), "429") ||
    strings.Contains(err.Error(), "500") {
  return true
 }
 return false
}
```

## System Prompts

System prompts help control Claude's behavior:

```go
// Example system prompts for different roles
var systemPrompts = map[string]string{
 "planner": `You are a planning agent responsible for breaking down complex tasks into smaller steps.
Always think step-by-step and create detailed plans with clear objectives.
Focus on creating actionable, concrete steps that can be executed by implementation agents.`,

 "coder": `You are a coding agent that writes clean, efficient, and well-documented code.
Follow best practices for the language you're using.
Always include error handling and use appropriate patterns.
Explain your implementation choices concisely.`,

 "reviewer": `You are a code review agent that provides constructive feedback.
Look for bugs, edge cases, and potential improvements.
Be thorough but respectful, focusing on the most important issues first.
Always suggest specific solutions when pointing out problems.`,
}
```

## Tool Use with Claude

Claude 3 models can use tools via the tool_use capability:

```json
{
  "model": "claude-3-opus-20240229",
  "messages": [
    {
      "role": "user",
      "content": "What's the weather in London?"
    }
  ],
  "tools": [
    {
      "name": "get_weather",
      "description": "Get the current weather for a location",
      "input_schema": {
        "type": "object",
        "properties": {
          "location": {
            "type": "string",
            "description": "The city and country"
          }
        },
        "required": ["location"]
      }
    }
  ],
  "tool_choice": "auto"
}
```

## Related Documentation

- [Anthropic API Documentation](https://docs.anthropic.com/claude/reference/getting-started-with-the-api)
- [../patterns/prompt_chain_memory.md](../patterns/prompt_chain_memory.md)
- [../architecture/agent_lifecycle.md](../architecture/agent_lifecycle.md)
