# AI Providers Package

This package provides a unified interface for interacting with multiple AI/LLM providers.

## Status

✅ **Production Ready Providers:**
- **OpenAI** - Full API support with GPT-4.1, GPT-4o, O3 models
- **Anthropic** - Full API support with Claude 4 models
- **DeepSeek** - OpenAI-compatible API with Chat and Reasoner models
- **DeepInfra** - OpenAI-compatible API with Llama, Mistral, Qwen models
- **Ollama** - Local model support with streaming
- **Ora** - API support for DeepSeek models
- **Mock** - Full-featured mock provider for testing

⚠️ **Legacy/Pending Update:**
- **Google** - Needs update to new AIProvider interface
- **Claude Code** - MCP-based, different purpose

## Key Features

### 1. Unified Interface
All providers implement the `AIProvider` interface:
```go
type AIProvider interface {
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    StreamChatCompletion(ctx context.Context, req ChatRequest) (ChatStream, error)
    CreateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
    GetCapabilities() ProviderCapabilities
}
```

### 2. Proper Context Handling
- All providers properly propagate context for cancellation/timeout
- HTTP requests use `http.NewRequestWithContext()`
- No context is dropped in the call chain

### 3. Real API Implementations
- No stubs - all providers make actual API calls
- Proper error handling with typed errors
- Rate limit and retry support

### 4. Comprehensive Testing
- Mock provider for unit tests (zero dependencies)
- HTTP mock server for integration tests
- Reusable test suite for all providers
- Context cancellation/timeout tests

## Usage

### Basic Usage
```go
// Create a provider
client := openai.NewClient("your-api-key")

// Make a request
resp, err := client.ChatCompletion(ctx, interfaces.ChatRequest{
    Model: openai.GPT41Mini,
    Messages: []interfaces.ChatMessage{
        {Role: "user", Content: "Hello!"},
    },
})
```

### Using the Factory
```go
factory := providers.NewFactoryV2()
provider, err := factory.CreateAIProvider(providers.ProviderOpenAI, "api-key")
```

### Testing with Mock Provider
```go
// Create mock with predefined responses
mock := mock.NewBuilder().
    WithResponse("Hello", "Hi there!").
    WithError("error", errors.New("simulated error")).
    Build()

// Use like any provider
resp, err := mock.ChatCompletion(ctx, req)

// Verify calls
calls := mock.GetCalls()
```

## Provider Comparison

| Provider | Context Window | Streaming | Vision | Tools | Cost (per M tokens) |
|----------|---------------|-----------|--------|-------|---------------------|
| OpenAI | 1M | ✅ | ✅ | ✅ | $1-$30 |
| Anthropic | 200K | ✅ | ✅ | ✅ | $3-$75 |
| DeepSeek | 64K | ✅ | ❌ | ✅ | $0.07-$2.19 |
| DeepInfra | 131K | ✅ | ❌ | ✅ | $0.06-$0.79 |
| Ollama | Varies | ✅ | ✅* | ❌ | Free (local) |
| Ora | 64K | ✅ | ❌ | ✅ | $0.1-$2 |

*Some Ollama models support vision

## Environment Variables

- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key
- `DEEPSEEK_API_KEY` - DeepSeek API key
- `DEEPINFRA_TOKEN` - DeepInfra API token
- `ORA_API_KEY` - Ora API key
- `OLLAMA_HOST` - Ollama server URL (default: http://localhost:11434)

## Testing

```bash
# Run all tests with mocks (fast)
go test -short ./pkg/providers/...

# Run specific provider tests
go test ./pkg/providers/openai

# Run with live APIs (requires API keys)
OPENAI_API_KEY=sk-... go test ./pkg/providers/openai

# Run with coverage
go test -cover ./pkg/providers/...
```

## Adding a New Provider

1. Create a new package in `pkg/providers/yourprovider/`
2. Implement the `AIProvider` interface
3. Add provider type to `interfaces/provider.go`
4. Update the factory in `factory_v2.go`
5. Create tests using the testing framework
6. Update this README

## Architecture Notes

- **Base implementations** - OpenAI-compatible providers share code via `base/openai_compatible.go`
- **Interface segregation** - Types are defined in `interfaces/` to avoid circular dependencies
- **Context-first** - All methods accept context as first parameter
- **Error types** - Typed errors with retry information
- **Capabilities** - Each provider reports its features and model information