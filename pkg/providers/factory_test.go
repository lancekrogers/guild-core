package providers_test

import (
	"os"
	"testing"

	"github.com/blockhead-consulting/guild/pkg/providers"
	"github.com/blockhead-consulting/guild/pkg/providers/anthropic"
	"github.com/blockhead-consulting/guild/pkg/providers/ollama"
	"github.com/blockhead-consulting/guild/pkg/providers/openai"
)

// TestNewFactory tests the creation of a new factory
func TestNewFactory(t *testing.T) {
	factory := providers.NewFactory()

	if factory == nil {
		t.Fatal("expected non-nil factory")
	}

	// Default provider should be OpenAI
	client, err := factory.GetDefaultClient()
	if err == nil {
		// This should fail without credentials, but if by chance it succeeds,
		// verify it's the right type
		if _, ok := client.(*openai.Client); !ok {
			t.Errorf("expected default client to be OpenAI client, got %T", client)
		}
	}
}

// TestRegisterProvider tests registering a provider configuration
func TestRegisterProvider(t *testing.T) {
	factory := providers.NewFactory()

	// Register a provider config
	config := providers.ProviderConfig{
		Type:   providers.ProviderOpenAI,
		ApiKey: "test-key",
		Model:  "test-model",
	}

	err := factory.RegisterProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get the client
	client, err := factory.GetClient(providers.ProviderOpenAI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// Check that it's the right type
	if _, ok := client.(*openai.Client); !ok {
		t.Errorf("expected OpenAI client, got %T", client)
	}

	// Check model info
	info := client.GetModelInfo()
	if info["model"] != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", info["model"])
	}
}

// TestSetDefaultProvider tests setting the default provider
func TestSetDefaultProvider(t *testing.T) {
	factory := providers.NewFactory()

	// Register providers
	openaiConfig := providers.ProviderConfig{
		Type:   providers.ProviderOpenAI,
		ApiKey: "test-key-openai",
		Model:  "test-model-openai",
	}
	err := factory.RegisterProvider(openaiConfig)
	if err != nil {
		t.Fatalf("unexpected error registering OpenAI: %v", err)
	}

	anthropicConfig := providers.ProviderConfig{
		Type:   providers.ProviderAnthropic,
		ApiKey: "test-key-anthropic",
		Model:  "test-model-anthropic",
	}
	err = factory.RegisterProvider(anthropicConfig)
	if err != nil {
		t.Fatalf("unexpected error registering Anthropic: %v", err)
	}

	// Set default provider to Anthropic
	factory.SetDefaultProvider(providers.ProviderAnthropic)

	// Get default client
	client, err := factory.GetDefaultClient()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that it's the right type
	if _, ok := client.(*anthropic.Client); !ok {
		t.Errorf("expected Anthropic client, got %T", client)
	}
}

// TestGetClient tests getting a client for a specific provider type
func TestGetClient(t *testing.T) {
	factory := providers.NewFactory()

	// Register a provider config
	ollamaConfig := providers.ProviderConfig{
		Type:   providers.ProviderOllama,
		ApiURL: "http://localhost:11434",
		Model:  "llama2",
	}
	err := factory.RegisterProvider(ollamaConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Get the client
	client, err := factory.GetClient(providers.ProviderOllama)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that it's the right type
	if _, ok := client.(*ollama.Client); !ok {
		t.Errorf("expected Ollama client, got %T", client)
	}

	// Check model info
	info := client.GetModelInfo()
	if info["model"] != "llama2" {
		t.Errorf("expected model 'llama2', got '%s'", info["model"])
	}

	// Request the same provider type again - should get the same client (cached)
	client2, err := factory.GetClient(providers.ProviderOllama)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if client != client2 {
		t.Error("expected to get the same client instance on second call")
	}
}

// TestEnvironmentFallback tests falling back to environment variables
func TestEnvironmentFallback(t *testing.T) {
	factory := providers.NewFactory()

	// Set environment variables
	oldOpenAI := os.Getenv("OPENAI_API_KEY")
	oldAnthropic := os.Getenv("ANTHROPIC_API_KEY")

	defer func() {
		// Restore environment
		os.Setenv("OPENAI_API_KEY", oldOpenAI)
		os.Setenv("ANTHROPIC_API_KEY", oldAnthropic)
	}()

	// Set test values
	os.Setenv("OPENAI_API_KEY", "test-env-key-openai")
	os.Setenv("ANTHROPIC_API_KEY", "test-env-key-anthropic")

	// Try to get OpenAI client without registering
	client, err := factory.GetClient(providers.ProviderOpenAI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that it's the right type
	if _, ok := client.(*openai.Client); !ok {
		t.Errorf("expected OpenAI client, got %T", client)
	}

	// Try to get Anthropic client without registering
	client, err = factory.GetClient(providers.ProviderAnthropic)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that it's the right type
	if _, ok := client.(*anthropic.Client); !ok {
		t.Errorf("expected Anthropic client, got %T", client)
	}
}

// TestUnknownProviderType tests handling unknown provider types
func TestUnknownProviderType(t *testing.T) {
	factory := providers.NewFactory()

	// Try to register an unknown provider type
	config := providers.ProviderConfig{
		Type:   "unknown",
		ApiKey: "test-key",
	}
	err := factory.RegisterProvider(config)
	if err != nil {
		t.Fatalf("unexpected error registering unknown provider: %v", err)
	}

	// Try to get client for unknown provider
	_, err = factory.GetClient("unknown")
	if err == nil {
		t.Fatal("expected error for unknown provider type, got nil")
	}

	if err.Error() != "unknown provider type: unknown" {
		t.Errorf("expected error 'unknown provider type: unknown', got '%s'", err.Error())
	}
}

// TestCloseAll tests closing all clients
func TestCloseAll(t *testing.T) {
	factory := providers.NewFactory()

	// Register and get a client
	config := providers.ProviderConfig{
		Type:   providers.ProviderOpenAI,
		ApiKey: "test-key",
	}
	err := factory.RegisterProvider(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = factory.GetClient(providers.ProviderOpenAI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close all clients
	factory.CloseAll()

	// Get client again - should create a new one
	client1, err := factory.GetClient(providers.ProviderOpenAI)
	if err != nil {
		t.Fatalf("unexpected error after CloseAll: %v", err)
	}

	if client1 == nil {
		t.Fatal("expected non-nil client after CloseAll")
	}
}

