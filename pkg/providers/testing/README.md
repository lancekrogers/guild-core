# Provider Testing Framework

This package provides a comprehensive testing framework for AI provider implementations in the Guild framework.

## Overview

The testing framework offers three main approaches:

1. **Mock Provider** - Pure Go implementation for unit tests
2. **HTTP Mock Server** - Simulates real API responses for integration tests  
3. **Provider Test Suite** - Reusable test cases for all providers

## Usage

### 1. Mock Provider (Simplest)

```go
import "github.com/guild-ventures/guild-core/pkg/providers/mock"

// Create a mock provider with predefined responses
provider := mock.NewBuilder().
    WithResponse("Hello", "Hi there!").
    WithError("error prompt", errors.New("simulated error")).
    WithDefaultResponse("Default response").
    Build()

// Use it like any other provider
resp, err := provider.ChatCompletion(ctx, req)

// Verify calls
calls := provider.GetCalls()
```

### 2. HTTP Mock Server (For OpenAI-Compatible Providers)

```go
import providertesting "github.com/guild-ventures/guild-core/pkg/providers/testing"

// Create mock HTTP server
mock := providertesting.NewMockHTTPServer()
defer mock.Close()

// Create provider pointing to mock
client := openai.NewClient("test-key", WithBaseURL(mock.URL))

// Make requests - mock will return realistic responses
resp, err := client.ChatCompletion(ctx, req)

// Verify HTTP requests
lastReq := mock.GetLastRequest()
```

### 3. Provider Test Suite (Comprehensive Testing)

```go
// Run standard tests for any provider
suite := providertesting.NewProviderTestSuite(t, provider, providertesting.TestConfig{
    ProviderName: "MyProvider",
    TestModel:    "model-id",
    SkipLive:     true, // Set to false for live API tests
    LiveAPIKey:   os.Getenv("MY_API_KEY"),
})

suite.RunBasicTests() // Runs capabilities, completion, and error tests
```

## Testing Patterns

### Unit Tests (No External Dependencies)

```go
func TestMyFeature(t *testing.T) {
    // Use mock provider
    provider := mock.NewProvider()
    provider.SetResponse("specific prompt", "expected response")
    
    // Test your code
    result := myFunction(provider)
    
    // Verify behavior
    calls := provider.GetCalls()
    if len(calls) != 1 {
        t.Error("Expected one API call")
    }
}
```

### Integration Tests (With Mock HTTP)

```go
func TestOpenAIIntegration(t *testing.T) {
    mock := providertesting.NewMockHTTPServer()
    defer mock.Close()
    
    // Your provider with mock URL
    provider := createProviderWithURL(mock.URL)
    
    // Test actual HTTP communication
    // Mock server returns OpenAI-compatible responses
}
```

### Live API Tests (Optional)

```go
func TestLiveAPI(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping live API test")
    }
    
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("API key not set")
    }
    
    provider := openai.NewClient(apiKey)
    // Run limited tests against real API
}
```

## Test Coverage Requirements

Each provider should have tests for:

1. **Interface Implementation** - Compile-time check
2. **Capabilities** - Correct model info, limits, features
3. **Chat Completion** - Basic request/response
4. **Error Handling** - Auth errors, rate limits, invalid requests
5. **Model-Specific Features** - Provider-specific functionality
6. **Cost Information** - Pricing data accuracy

## Running Tests

```bash
# Run all provider tests
go test ./pkg/providers/...

# Run with mock servers only (fast)
go test -short ./pkg/providers/...

# Run specific provider tests
go test ./pkg/providers/openai

# Run with live APIs (requires API keys)
OPENAI_API_KEY=sk-... go test ./pkg/providers/openai

# Run with coverage
go test -cover ./pkg/providers/...
```

## Environment Variables for Live Tests

- `OPENAI_API_KEY` - OpenAI API key
- `ANTHROPIC_API_KEY` - Anthropic API key  
- `DEEPSEEK_API_KEY` - DeepSeek API key
- `DEEPINFRA_TOKEN` - DeepInfra API token
- `ORA_API_KEY` - Ora API key
- `OLLAMA_HOST` - Ollama server URL (default: http://localhost:11434)

## Best Practices

1. **Always test with mocks first** - Don't rely on external APIs for unit tests
2. **Use the test suite** - Ensures consistent testing across providers
3. **Test error conditions** - Auth failures, rate limits, network errors
4. **Verify capabilities** - Each provider should accurately report features
5. **Keep live tests minimal** - Only test basic connectivity with real APIs
6. **Record API calls** - Use mock provider's call recording for verification

## Adding a New Provider

1. Implement the `interfaces.AIProvider` interface
2. Create a test file using the standard pattern
3. Use the test suite for basic coverage
4. Add provider-specific tests as needed
5. Update `all_providers_test.go` to include the new provider

Example test file structure:

```go
package myprovider

import (
    "testing"
    "github.com/guild-ventures/guild-core/pkg/providers/testing"
)

func TestMyProvider(t *testing.T) {
    // Create provider (with mock if needed)
    provider := NewClient("test-key")
    
    // Run standard suite
    suite := testing.NewProviderTestSuite(t, provider, testing.TestConfig{
        ProviderName: "MyProvider",
        TestModel:    "default-model",
        SkipLive:     true,
    })
    suite.RunBasicTests()
}

func TestMyProviderSpecificFeatures(t *testing.T) {
    // Add provider-specific tests
}
```