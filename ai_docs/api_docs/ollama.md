# Ollama API Integration

This document explains how to integrate with the Ollama API in Guild.

## Overview

Ollama is an open-source framework for running large language models locally. Guild integrates with Ollama to provide access to a variety of models that can run on the user's local machine, eliminating the need for API keys and reducing costs.

## Installation

Before using Ollama with Guild, users need to install Ollama on their system:

1. **Install Ollama**:

   - Visit [ollama.ai](https://ollama.ai) and download the appropriate version
   - For macOS: Install from the downloaded DMG
   - For Linux: `curl -fsSL https://ollama.ai/install.sh | sh`
   - For Windows: Follow instructions on the Ollama website

2. **Download Models**:

   - After installation, download models with: `ollama pull <model-name>`
   - Example: `ollama pull llama3:8b` or `ollama pull mistral`

3. **Start Ollama Service**:
   - Ollama should automatically run as a service
   - Verify with: `ollama list` to see installed models

## Configuration

```yaml
providers:
  ollama:
    base_url: http://localhost:11434
    default_model: llama3:8b
```

## Provider Implementation

```go
// pkg/providers/ollama/client.go
package ollama

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

// Client implements the Provider interface for Ollama
type Client struct {
	baseURL     string
	client      *http.Client
	defaultModel string
}

// NewClient creates a new Ollama client
func NewClient(baseURL string, defaultModel string) (*Client, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	if defaultModel == "" {
		defaultModel = "llama3:8b"
	}

	return &Client{
		baseURL:     baseURL,
		defaultModel: defaultModel,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Type returns the provider type
func (c *Client) Type() providers.ProviderType {
	return providers.ProviderOllama
}

// Models returns the available models
func (c *Client) Models() []string {
	// Call the Ollama API to get the list of locally available models
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/tags", c.baseURL), nil)
	if err != nil {
		return []string{c.defaultModel}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return []string{c.defaultModel}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{c.defaultModel}
	}

	var modelResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelResp); err != nil {
		return []string{c.defaultModel}
	}

	// Extract model names
	models := make([]string, 0, len(modelResp.Models))
	for _, model := range modelResp.Models {
		models = append(models, model.Name)
	}

	return models
}

// Generate produces text from a prompt
func (c *Client) Generate(ctx context.Context, req providers.GenerateRequest) (providers.GenerateResponse, error) {
	// Set default model if not provided
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}

	// Create Ollama request
	ollamaReq := CompletionRequest{
		Model:       model,
		Prompt:      req.Prompt,
		System:      req.SystemPrompt,
		Format:      "json", // Request JSON format for better parsing
		Temperature: req.Temperature,
		Stream:      false,
	}

	// Set max tokens if provided
	if req.MaxTokens > 0 {
		ollamaReq.NumPredict = req.MaxTokens
	}

	// Convert to JSON
	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/generate", c.baseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

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
		return providers.GenerateResponse{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var completionResp CompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return providers.GenerateResponse{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Return result
	return providers.GenerateResponse{
		Text:         completionResp.Response,
		TokensUsed:   completionResp.EvalCount + completionResp.PromptEvalCount,
		FinishReason: completionResp.Done,
		Raw:          completionResp,
	}, nil
}

// GenerateStream produces a stream of tokens
func (c *Client) GenerateStream(ctx context.Context, req providers.GenerateRequest) (<-chan providers.GenerateResponseChunk, error) {
	// Set default model if not provided
	model := req.Model
	if model == "" {
		model = c.defaultModel
	}

	// Create Ollama request
	ollamaReq := CompletionRequest{
		Model:       model,
		Prompt:      req.Prompt,
		System:      req.SystemPrompt,
		Temperature: req.Temperature,
		Stream:      true,
	}

	// Set max tokens if provided
	if req.MaxTokens > 0 {
		ollamaReq.NumPredict = req.MaxTokens
	}

	// Convert to JSON
	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/generate", c.baseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

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

		// Create JSON decoder
		decoder := json.NewDecoder(resp.Body)

		// Process stream
		var fullText string
		for {
			// Decode next chunk
			var streamResp CompletionResponse
			if err := decoder.Decode(&streamResp); err != nil {
				if err != io.EOF {
					outputCh <- providers.GenerateResponseChunk{
						Error: fmt.Errorf("stream error: %w", err),
					}
				}
				break
			}

			// Send chunk
			outputCh <- providers.GenerateResponseChunk{
				Text: streamResp.Response,
			}

			// Accumulate text
			fullText += streamResp.Response

			// Check for end of stream
			if streamResp.Done == "true" {
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
	return true
}

// Cost returns the estimated cost for a request
func (c *Client) Cost(req providers.GenerateRequest) float64 {
	// Ollama is free, so the cost is always 0
	return 0.0
}

// Models API types

// ModelInfo contains information about a model
type ModelInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// ModelsResponse contains a list of available models
type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
}

// Generation API types

// CompletionRequest is the request body for the Ollama API
type CompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	System      string  `json:"system,omitempty"`
	Template    string  `json:"template,omitempty"`
	Context     []int   `json:"context,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	Format      string  `json:"format,omitempty"`
}

// CompletionResponse is the response from the Ollama API
type CompletionResponse struct {
	Model          string  `json:"model"`
	Response       string  `json:"response"`
	Done           string  `json:"done"`
	Context        []int   `json:"context,omitempty"`
	PromptEvalCount int     `json:"prompt_eval_count,omitempty"`
	EvalCount      int     `json:"eval_count,omitempty"`
	TotalDuration  int64   `json:"total_duration,omitempty"`
	LoadDuration   int64   `json:"load_duration,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalDuration   int64   `json:"eval_duration,omitempty"`
}

// Helper function to check if Ollama is available
func (c *Client) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/tags", c.baseURL), nil)
	if err != nil {
		return false
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
```

## Available Models

Ollama supports a variety of models that can be downloaded and run locally. Here are some popular options:

| Model      | Description                      | Parameters | Quantization | Disk Size |
| ---------- | -------------------------------- | ---------- | ------------ | --------- |
| llama3:8b  | Latest open-source Llama 3 model | 8B         | Q4_K_M       | ~4.7GB    |
| llama3:70b | Largest Llama 3 model            | 70B        | Q4_K_M       | ~42GB     |
| mistral    | Mistral 7B model                 | 7B         | Q4_K_M       | ~4.1GB    |
| phi3       | Microsoft Phi-3 mini             | 3.8B       | Q4_K_M       | ~2.3GB    |
| gemma:2b   | Google Gemma 2B model            | 2B         | Q4_K_M       | ~1.4GB    |
| gemma:7b   | Google Gemma 7B model            | 7B         | Q4_K_M       | ~4.8GB    |
| codellama  | Code-specialized Llama model     | 7B         | Q4_K_M       | ~4.1GB    |
| orca-mini  | Small but capable model          | 3B         | Q4_K_M       | ~2.0GB    |

Users can get the full list of available models with:

```bash
ollama list
```

To download and install a model:

```bash
ollama pull <model-name>
```

## API Reference

### Generate API

The main API endpoint for text generation is `/api/generate`:

```json
{
  "model": "llama3:8b",
  "prompt": "Write a function to calculate Fibonacci numbers in Go",
  "system": "You are a helpful coding assistant.",
  "stream": false,
  "num_predict": 1024,
  "temperature": 0.7
}
```

### Response Format

Standard response format:

````json
{
  "model": "llama3:8b",
  "response": "Here's a function to calculate Fibonacci numbers in Go:\n\n```go\nfunc fibonacci(n int) int {\n    if n <= 1 {\n        return n\n    }\n    return fibonacci(n-1) + fibonacci(n-2)\n}\n```",
  "done": "true",
  "context": [1, 2, 3, ...],
  "prompt_eval_count": 12,
  "eval_count": 227,
  "total_duration": 580424253
}
````

Streaming response format (multiple JSON objects, one per line):

````json
{"model":"llama3:8b","response":"Here","done":"false","context":[...]}
{"model":"llama3:8b","response":"'s","done":"false"}
{"model":"llama3:8b","response":" a","done":"false"}
...
{"model":"llama3:8b","response":"}","done":"false"}
{"model":"llama3:8b","response":"```","done":"true"}
````

## System Prompts and Templates

Ollama supports system prompts to guide model behavior:

```go
// Example system prompts for Ollama models
var systemPrompts = map[string]string{
	"code": `You are an expert programmer. Write clear, efficient, and well-documented code.
Follow best practices for the programming language you're using.
Only provide working code that fully addresses the user's request.`,

	"planner": `You are a planning assistant. Break down complex tasks into manageable steps.
Be thorough and consider all aspects of the task.
Organize the steps in a logical order and explain dependencies.`,

	"qa": `You are a quality assurance expert. Analyze code for bugs, efficiency issues, and best practices.
Suggest specific improvements and explain why they're important.
Be thorough but constructive in your feedback.`,
}
```

## Performance Considerations

When using Ollama, consider these performance factors:

1. **Hardware Requirements**:

   - CPU: Modern multi-core processor (8+ cores recommended for larger models)
   - RAM: 8GB minimum, 16GB+ recommended (32GB+ for larger models)
   - GPU: NVIDIA GPU with 8GB+ VRAM for acceleration
   - Disk: 10GB+ free space for model storage

2. **Model Selection**:

   - Smaller models (2B-7B) run well on most systems
   - Medium models (8B-13B) need decent hardware
   - Large models (30B-70B) require high-end hardware or GPU

3. **First-Time Latency**:

   - First request to a model has higher latency (model loading)
   - Subsequent requests are faster

4. **Optimizations**:
   - Use `context` parameter to maintain conversation state efficiently
   - Choose appropriate quantization levels (Q4_K_M is balanced)

## Command-Line Interface

Guild should provide commands to manage Ollama:

```go
// pkg/cli/commands/ollama.go
package commands

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/your-username/guild/pkg/providers/ollama"
)

// OllamaCmd returns the ollama management command
func OllamaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ollama",
		Short: "Manage Ollama models and instance",
		Long:  `Commands for managing Ollama models and the Ollama instance.`,
	}

	cmd.AddCommand(ollamaCheckCmd())
	cmd.AddCommand(ollamaListCmd())
	cmd.AddCommand(ollamaPullCmd())

	return cmd
}

// ollamaCheckCmd checks if Ollama is running
func ollamaCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if Ollama is available",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := ollama.NewClient("http://localhost:11434", "")
			if err != nil {
				fmt.Println("Error creating Ollama client:", err)
				return
			}

			if client.IsAvailable() {
				fmt.Println("✅ Ollama is running and available")
			} else {
				fmt.Println("❌ Ollama is not available. Make sure it's installed and running.")
				fmt.Println("   Install from: https://ollama.ai")
			}
		},
	}

	return cmd
}

// ollamaListCmd lists available models
func ollamaListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available Ollama models",
		Run: func(cmd *cobra.Command, args []string) {
			// Use the system command for better formatting
			ollamaCmd := exec.Command("ollama", "list")
			output, err := ollamaCmd.CombinedOutput()
			if err != nil {
				fmt.Println("Error listing models:", err)
				return
			}

			fmt.Println(string(output))
		},
	}

	return cmd
}

// ollamaPullCmd pulls a model
func ollamaPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull [model]",
		Short: "Pull an Ollama model",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			model := args[0]

			fmt.Printf("Pulling model %s...\n", model)
			ollamaCmd := exec.Command("ollama", "pull", model)
			ollamaCmd.Stdout = cmd.OutOrStdout()
			ollamaCmd.Stderr = cmd.ErrOrStderr()

			if err := ollamaCmd.Run(); err != nil {
				fmt.Println("Error pulling model:", err)
				return
			}

			fmt.Printf("Model %s pulled successfully\n", model)
		},
	}

	return cmd
}
```

## Usage Examples

### Basic Usage

```go
// Create Ollama client
ollamaClient, err := ollama.NewClient("http://localhost:11434", "llama3:8b")
if err != nil {
	log.Fatalf("Failed to create Ollama client: %v", err)
}

// Check if Ollama is available
if !ollamaClient.IsAvailable() {
	log.Fatalf("Ollama is not available. Please install and start Ollama service.")
}

// Generate text
ctx := context.Background()
resp, err := ollamaClient.Generate(ctx, providers.GenerateRequest{
	Model:        "llama3:8b",
	Prompt:       "Explain how goroutines work in Go",
	SystemPrompt: "You are a helpful programming assistant",
	MaxTokens:    1024,
	Temperature:  0.7,
})

if err != nil {
	log.Fatalf("Failed to generate text: %v", err)
}

fmt.Println(resp.Text)
```

### Streaming Response

```go
// Generate streaming response
ctx := context.Background()
stream, err := ollamaClient.GenerateStream(ctx, providers.GenerateRequest{
	Model:        "llama3:8b",
	Prompt:       "Write a haiku about programming",
	SystemPrompt: "You are a creative writing assistant",
	Temperature:  0.9,
})

if err != nil {
	log.Fatalf("Failed to start stream: %v", err)
}

// Process stream
for chunk := range stream {
	if chunk.Error != nil {
		log.Printf("Stream error: %v", chunk.Error)
		break
	}

	// Print chunk
	fmt.Print(chunk.Text)

	// Check if final
	if chunk.IsFinal {
		fmt.Println("\n--- End of response ---")
	}
}
```

### Fallback to API Models

Implement a fallback mechanism when Ollama is not available:

```go
// Try to create Ollama client
ollamaClient, err := ollama.NewClient("http://localhost:11434", "")
if err != nil || !ollamaClient.IsAvailable() {
	log.Println("Ollama not available, falling back to API provider")
	// Create OpenAI or Anthropic client instead
	client, err = openai.NewClient(apiKey)
	if err != nil {
		log.Fatalf("Failed to create fallback provider: %v", err)
	}
} else {
	client = ollamaClient
}

// Use client as normal
resp, err := client.Generate(ctx, req)
```

## Error Handling

Common errors and their solutions:

| Error              | Cause                      | Solution                       |
| ------------------ | -------------------------- | ------------------------------ |
| Connection refused | Ollama service not running | Start Ollama service           |
| Model not found    | Model not downloaded       | Run `ollama pull <model>`      |
| Out of memory      | Model too large for system | Use a smaller model or add RAM |
| Timeout            | Request taking too long    | Adjust client timeout settings |

## Best Practices

1. **Model Management**:

   - Pre-download required models during installation
   - Check model availability before generating
   - Prefer smaller models for routine tasks

2. **Resource Usage**:

   - Monitor memory and CPU usage when running large models
   - Implement cool-down periods between requests
   - Shut down models not in use with `ollama rm <model>` to free resources

3. **Request Optimization**:

   - Keep prompts clear and concise
   - Use appropriate temperature settings (0.7 is a good default)
   - Pass conversation context to maintain coherence

4. **Error Resilience**:
   - Implement retry mechanisms with backoff
   - Provide graceful fallbacks to API providers
   - Log performance metrics to identify bottlenecks

## Related Documentation

- [Ollama GitHub Repository](https://github.com/ollama/ollama)
- [Ollama API Documentation](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [../patterns/prompt_chain_memory.md](../patterns/prompt_chain_memory.md)
- [../architecture/agent_lifecycle.md](../architecture/agent_lifecycle.md)
