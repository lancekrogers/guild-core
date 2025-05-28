package providers

import (
	"context"
	"fmt"
	"os"
	
	"github.com/guild-ventures/guild-core/pkg/providers/anthropic"
	"github.com/guild-ventures/guild-core/pkg/providers/deepinfra"
	"github.com/guild-ventures/guild-core/pkg/providers/deepseek"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
	"github.com/guild-ventures/guild-core/pkg/providers/ollama"
	"github.com/guild-ventures/guild-core/pkg/providers/openai"
	"github.com/guild-ventures/guild-core/pkg/providers/ora"
)

// FactoryV2 creates AI providers using the new AIProvider interface
type FactoryV2 struct{}

// NewFactoryV2 creates a new factory for AI providers
func NewFactoryV2() *FactoryV2 {
	return &FactoryV2{}
}

// CreateAIProvider creates a new AI provider based on the provider type
func (f *FactoryV2) CreateAIProvider(providerType ProviderType, apiKey string) (interfaces.AIProvider, error) {
	switch providerType {
	case ProviderOpenAI:
		return openai.NewClient(apiKey), nil
	case ProviderAnthropic:
		return anthropic.NewClient(apiKey), nil
	case ProviderDeepSeek:
		return deepseek.NewClient(apiKey), nil
	case ProviderDeepInfra:
		return deepinfra.NewClient(apiKey), nil
	case ProviderOllama:
		// For Ollama, apiKey is interpreted as baseURL if provided
		return ollama.NewClient(apiKey), nil
	case ProviderOra:
		return ora.NewClient(apiKey), nil
	case ProviderGoogle:
		// Google provider needs updating to implement AIProvider
		return nil, fmt.Errorf("Google provider not yet updated to AIProvider interface")
	case ProviderClaudeCode:
		// Claude Code is a special case
		return nil, fmt.Errorf("Claude Code provider not compatible with AIProvider interface")
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// CreateAIProviderFromConfig creates a provider from configuration map
func (f *FactoryV2) CreateAIProviderFromConfig(providerType ProviderType, config map[string]interface{}) (interfaces.AIProvider, error) {
	// Extract API key - try config first, then environment variable
	var apiKey string
	if key, exists := config["api_key"]; exists {
		if keyStr, ok := key.(string); ok {
			apiKey = keyStr
		}
	}
	
	// If no direct API key, try environment variable reference
	if apiKey == "" {
		if envVar, exists := config["api_key_env"]; exists {
			if envVarStr, ok := envVar.(string); ok {
				apiKey = os.Getenv(envVarStr)
			}
		}
	}

	// For Ollama, check for base URL
	if providerType == ProviderOllama {
		if baseURL, exists := config["base_url"]; exists {
			if urlStr, ok := baseURL.(string); ok {
				apiKey = urlStr // Use apiKey parameter as baseURL for Ollama
			}
		}
	}

	return f.CreateAIProvider(providerType, apiKey)
}

// CreateLLMClientAdapter creates an adapter that implements the LLMClient interface
// using the AIProvider interface for backward compatibility
func (f *FactoryV2) CreateLLMClientAdapter(provider interfaces.AIProvider) LLMClient {
	return &llmClientAdapter{provider: provider}
}

// llmClientAdapter adapts AIProvider to LLMClient interface
type llmClientAdapter struct {
	provider interfaces.AIProvider
}

func (a *llmClientAdapter) Complete(ctx context.Context, prompt string) (string, error) {
	// Use the first available model from capabilities
	capabilities := a.provider.GetCapabilities()
	model := ""
	if len(capabilities.Models) > 0 {
		model = capabilities.Models[0].ID
	}

	req := interfaces.ChatRequest{
		Model: model,
		Messages: []interfaces.ChatMessage{
			{Role: "user", Content: prompt},
		},
	}

	resp, err := a.provider.ChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from provider")
}

// GetProviderInfo returns information about available providers
func GetProviderInfo() map[ProviderType]string {
	return map[ProviderType]string{
		ProviderOpenAI:     "OpenAI (GPT-4.1, GPT-4o, O3)",
		ProviderAnthropic:  "Anthropic (Claude 4)",
		ProviderDeepSeek:   "DeepSeek (Chat, Reasoner)",
		ProviderDeepInfra:  "DeepInfra (Llama, Mistral, Qwen)",
		ProviderOllama:     "Ollama (Local models)",
		ProviderOra:        "Ora (DeepSeek models)",
		ProviderGoogle:     "Google (Gemini) - Legacy only",
		ProviderClaudeCode: "Claude Code (MCP) - Legacy only",
	}
}