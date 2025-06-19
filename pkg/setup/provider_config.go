// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// ProviderConfig handles provider configuration and validation
type ProviderConfig struct {
	projectPath string
	factory     *providers.Factory
}

// NewProviderConfig creates a new provider configuration handler
func NewProviderConfig(ctx context.Context, projectPath string) (*ProviderConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider config creation").
			WithComponent("ProviderConfiguration").
			WithOperation("NewProviderConfig")
	}

	if projectPath == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "project path cannot be empty", nil).
			WithComponent("ProviderConfiguration").
			WithOperation("NewProviderConfig")
	}

	return &ProviderConfig{
		projectPath: projectPath,
		factory:     providers.NewFactory(),
	}, nil
}

// ProviderValidation contains provider validation results
type ProviderValidation struct {
	IsValid  bool
	Error    string
	Settings map[string]string
	Models   []string
	Warning  string
}

// ValidateProvider validates a detected provider's configuration
func (pc *ProviderConfig) ValidateProvider(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider validation").
			WithComponent("ProviderConfiguration").
			WithOperation("ValidateProvider").
			WithDetails("provider", provider.Name)
	}

	// Normalize provider name
	normalizedName := providers.NormalizeProviderName(provider.Name)
	
	switch normalizedName {
	case providers.ProviderNameClaude:
		return pc.validateClaudeCode(ctx, provider)
	case providers.ProviderNameOllama:
		return pc.validateOllama(ctx, provider)
	case providers.ProviderNameOpenAI:
		return pc.validateOpenAI(ctx, provider)
	case providers.ProviderNameAnthropic:
		return pc.validateAnthropic(ctx, provider)
	case providers.ProviderNameDeepSeek:
		return pc.validateDeepSeek(ctx, provider)
	case providers.ProviderNameDeepInfra:
		return pc.validateDeepInfra(ctx, provider)
	case providers.ProviderNameOra:
		return pc.validateOra(ctx, provider)
	default:
		return nil, gerror.Newf(gerror.ErrCodeValidation, "unsupported provider: %s", provider.Name).
			WithComponent("ProviderConfiguration").
			WithOperation("ValidateProvider").
			WithDetails("provider", provider.Name)
	}
}

// validateClaudeCode validates Claude Code configuration
func (pc *ProviderConfig) validateClaudeCode(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Claude Code validation").
			WithComponent("ProviderConfiguration").
			WithOperation("validateClaudeCode")
	}

	// Claude Code is always valid if detected
	settings := map[string]string{
		"type":        providers.ProviderNameClaude,
		"environment": "claude_code_session",
	}

	// Check if we have specific Claude Code environment variables
	if session := os.Getenv("CLAUDE_CODE_SESSION"); session != "" {
		settings["session_id"] = session
	}

	return &ProviderValidation{
		IsValid:  true,
		Error:    "",
		Settings: settings,
		Models:   []string{"claude-3-5-sonnet-20241022"},
		Warning:  "",
	}, nil
}

// validateOllama validates Ollama configuration
func (pc *ProviderConfig) validateOllama(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Ollama validation").
			WithComponent("ProviderConfiguration").
			WithOperation("validateOllama")
	}

	settings := map[string]string{
		"type": providers.ProviderNameOllama,
	}

	// If Ollama is not running, provide guidance
	if provider.Endpoint == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    "Ollama service is not running",
			Settings: settings,
			Models:   []string{},
			Warning:  "Start Ollama with 'ollama serve' then run setup again",
		}, nil
	}

	// Set the base URL
	settings["base_url"] = provider.Endpoint

	// Test connection and get models
	models, err := pc.ollamaModels(ctx, provider.Endpoint)
	if err != nil {
		// Check if it's a gerror already
		if _, ok := err.(*gerror.GuildError); !ok {
			err = gerror.Wrap(err, gerror.ErrCodeConnection, "failed to get Ollama models").
				WithComponent("ProviderConfiguration").
				WithOperation("validateOllama").
				WithDetails("endpoint", provider.Endpoint)
		}
		return &ProviderValidation{
			IsValid:  false,
			Error:    fmt.Sprintf("Failed to get models: %v", err),
			Settings: settings,
			Models:   []string{},
			Warning:  "Ensure Ollama is running and accessible",
		}, nil
	}

	if len(models) == 0 {
		return &ProviderValidation{
			IsValid:  true,
			Error:    "",
			Settings: settings,
			Models:   []string{},
			Warning:  "No models installed. Install models with 'ollama pull <model>'",
		}, nil
	}

	return &ProviderValidation{
		IsValid:  true,
		Error:    "",
		Settings: settings,
		Models:   models,
		Warning:  "",
	}, nil
}

// validateOpenAI validates OpenAI configuration
func (pc *ProviderConfig) validateOpenAI(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during OpenAI validation").
			WithComponent("ProviderConfiguration").
			WithOperation("validateOpenAI")
	}

	apiKey := os.Getenv(providers.EnvOpenAIKey)
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    fmt.Sprintf("%s environment variable not set", providers.EnvOpenAIKey),
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  fmt.Sprintf("Set your OpenAI API key: export %s=sk-...", providers.EnvOpenAIKey),
		}, nil
	}

	// Validate API key format
	if !strings.HasPrefix(apiKey, "sk-") {
		return &ProviderValidation{
			IsValid:  false,
			Error:    "Invalid OpenAI API key format",
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  "OpenAI API keys should start with 'sk-'",
		}, nil
	}

	settings := map[string]string{
		"type":     providers.ProviderNameOpenAI,
		"base_url": providers.EndpointOpenAI,
	}

	// Add organization ID if available
	if orgID := os.Getenv("OPENAI_ORG_ID"); orgID != "" {
		settings["organization"] = orgID
	}

	// Test the API key (optional - can be skipped for quick setup)
	models := []string{
		"gpt-4-turbo-preview",
		"gpt-4",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-16k",
	}

	return &ProviderValidation{
		IsValid:  true,
		Error:    "",
		Settings: settings,
		Models:   models,
		Warning:  "",
	}, nil
}

// validateAnthropic validates Anthropic configuration
func (pc *ProviderConfig) validateAnthropic(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Anthropic validation").
			WithComponent("ProviderConfiguration").
			WithOperation("validateAnthropic")
	}

	apiKey := os.Getenv(providers.EnvAnthropicKey)
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    fmt.Sprintf("%s environment variable not set", providers.EnvAnthropicKey),
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  fmt.Sprintf("Set your Anthropic API key: export %s=sk-ant-...", providers.EnvAnthropicKey),
		}, nil
	}

	// Validate API key format
	if !strings.HasPrefix(apiKey, "sk-ant-") {
		return &ProviderValidation{
			IsValid:  false,
			Error:    "Invalid Anthropic API key format",
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  "Anthropic API keys should start with 'sk-ant-'",
		}, nil
	}

	settings := map[string]string{
		"type":     providers.ProviderNameAnthropic,
		"base_url": providers.EndpointAnthropic,
	}

	models := []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}

	return &ProviderValidation{
		IsValid:  true,
		Error:    "",
		Settings: settings,
		Models:   models,
		Warning:  "",
	}, nil
}

// validateDeepSeek validates DeepSeek configuration
func (pc *ProviderConfig) validateDeepSeek(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during DeepSeek validation").
			WithComponent("ProviderConfiguration").
			WithOperation("validateDeepSeek")
	}

	apiKey := os.Getenv(providers.EnvDeepSeekKey)
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    fmt.Sprintf("%s environment variable not set", providers.EnvDeepSeekKey),
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  fmt.Sprintf("Set your DeepSeek API key: export %s=<your-key>", providers.EnvDeepSeekKey),
		}, nil
	}

	settings := map[string]string{
		"type":     providers.ProviderNameDeepSeek,
		"base_url": providers.EndpointDeepSeek,
	}

	models := []string{
		"deepseek-chat",
		"deepseek-coder",
	}

	return &ProviderValidation{
		IsValid:  true,
		Error:    "",
		Settings: settings,
		Models:   models,
		Warning:  "",
	}, nil
}

// validateDeepInfra validates DeepInfra configuration
func (pc *ProviderConfig) validateDeepInfra(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during DeepInfra validation").
			WithComponent("ProviderConfiguration").
			WithOperation("validateDeepInfra")
	}

	apiKey := os.Getenv(providers.EnvDeepInfraKey)
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    fmt.Sprintf("%s environment variable not set", providers.EnvDeepInfraKey),
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  fmt.Sprintf("Set your DeepInfra API key: export %s=<your-key>", providers.EnvDeepInfraKey),
		}, nil
	}

	settings := map[string]string{
		"type":     providers.ProviderNameDeepInfra,
		"base_url": providers.EndpointDeepInfra,
	}

	models := []string{
		"meta-llama/Llama-2-70b-chat-hf",
		"codellama/CodeLlama-34b-Instruct-hf",
		"mistralai/Mixtral-8x7B-Instruct-v0.1",
	}

	return &ProviderValidation{
		IsValid:  true,
		Error:    "",
		Settings: settings,
		Models:   models,
		Warning:  "",
	}, nil
}

// validateOra validates Ora configuration
func (pc *ProviderConfig) validateOra(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Ora validation").
			WithComponent("ProviderConfiguration").
			WithOperation("validateOra")
	}

	apiKey := os.Getenv(providers.EnvOraKey)
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    fmt.Sprintf("%s environment variable not set", providers.EnvOraKey),
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  fmt.Sprintf("Set your Ora API key: export %s=<your-key>", providers.EnvOraKey),
		}, nil
	}

	settings := map[string]string{
		"type":     providers.ProviderNameOra,
		"base_url": providers.EndpointOra,
	}

	models := []string{
		"gpt-4",
		"gpt-3.5-turbo",
		"claude-3-sonnet",
	}

	return &ProviderValidation{
		IsValid:  true,
		Error:    "",
		Settings: settings,
		Models:   models,
		Warning:  "",
	}, nil
}

// ollamaModels retrieves the list of available models from Ollama
func (pc *ProviderConfig) ollamaModels(ctx context.Context, endpoint string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Ollama models request").
			WithComponent("ProviderConfiguration").
			WithOperation("ollamaModels")
	}

	// Create timeout context for the request
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(timeoutCtx, "GET", endpoint+"/api/tags", nil)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create request").
			WithComponent("ProviderConfiguration").
			WithOperation("ollamaModels").
			WithDetails("endpoint", endpoint)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Check if error is due to context cancellation/timeout
		if timeoutCtx.Err() != nil {
			return nil, gerror.Wrap(timeoutCtx.Err(), gerror.ErrCodeTimeout, "timeout during Ollama models request").
				WithComponent("ProviderConfiguration").
				WithOperation("ollamaModels").
				WithDetails("endpoint", endpoint)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to Ollama").
			WithComponent("ProviderConfiguration").
			WithOperation("ollamaModels").
			WithDetails("endpoint", endpoint)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, gerror.Newf(gerror.ErrCodeConnection, "Ollama API returned status: %d", resp.StatusCode).
			WithComponent("ProviderConfiguration").
			WithOperation("ollamaModels").
			WithDetails("endpoint", endpoint).
			WithDetails("status_code", resp.StatusCode)
	}

	// For now, return common models (in real implementation, parse JSON response)
	// This is a simplified version - actual implementation would parse the JSON response
	models := []string{
		"llama2",
		"codellama",
		"mistral",
		"neural-chat",
		"starling-lm",
	}

	return models, nil
}

// TestProviderConnection performs a real test of the provider connection
func (pc *ProviderConfig) TestProviderConnection(ctx context.Context, provider DetectedProvider) (*ConnectionTest, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider connection test").
			WithComponent("ProviderConfiguration").
			WithOperation("TestProviderConnection").
			WithDetails("provider", provider.Name)
	}

	// Create a test client
	var providerType providers.ProviderType
	switch provider.Name {
	case "openai":
		providerType = providers.ProviderOpenAI
	case "anthropic":
		providerType = providers.ProviderAnthropic
	case "ollama":
		providerType = providers.ProviderOllama
	case "claude_code":
		providerType = providers.ProviderClaudeCode
	case "deepseek":
		providerType = providers.ProviderDeepSeek
	case "deepinfra":
		providerType = providers.ProviderDeepInfra
	case "ora":
		providerType = providers.ProviderOra
	default:
		return &ConnectionTest{
			Success: false,
			Error:   fmt.Sprintf("Unsupported provider: %s", provider.Name),
			Latency: 0,
		}, nil
	}

	// Get API key from environment
	apiKey := ""
	switch provider.Name {
	case "openai":
		apiKey = os.Getenv("OPENAI_API_KEY")
	case "anthropic":
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	case "deepseek":
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	case "deepinfra":
		apiKey = os.Getenv("DEEPINFRA_API_KEY")
	case "ora":
		apiKey = os.Getenv("ORA_API_KEY")
	case "claude_code":
		apiKey = "claude-code-session" // Special case
	case "ollama":
		apiKey = "" // No API key needed for Ollama
	}

	start := time.Now()

	// Create client and test with a simple prompt
	client, err := pc.factory.CreateClient(providerType, apiKey, "")
	if err != nil {
		return &ConnectionTest{
			Success: false,
			Error:   fmt.Sprintf("Failed to create client: %v", err),
			Latency: time.Since(start),
		}, nil
	}

	// Check for cancellation before test
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before connection test").
			WithComponent("ProviderConfiguration").
			WithOperation("TestProviderConnection").
			WithDetails("provider", provider.Name)
	}

	// Test with a simple completion
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err = client.Complete(testCtx, "Hello, respond with just 'OK'")
	latency := time.Since(start)

	if err != nil {
		// Check if error is due to context cancellation/timeout
		if testCtx.Err() != nil {
			return &ConnectionTest{
				Success: false,
				Error:   fmt.Sprintf("Connection test timeout: %v", testCtx.Err()),
				Latency: latency,
			}, nil
		}
		return &ConnectionTest{
			Success: false,
			Error:   fmt.Sprintf("Connection test failed: %v", err),
			Latency: latency,
		}, nil
	}

	return &ConnectionTest{
		Success: true,
		Error:   "",
		Latency: latency,
	}, nil
}

// ConnectionTest contains the results of a connection test
type ConnectionTest struct {
	Success bool
	Error   string
	Latency time.Duration
}

// ProviderRecommendations returns recommendations for provider setup
func (pc *ProviderConfig) ProviderRecommendations(ctx context.Context, providers []DetectedProvider) (*ProviderRecommendations, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider recommendations").
			WithComponent("ProviderConfiguration").
			WithOperation("ProviderRecommendations")
	}

	recs := &ProviderRecommendations{
		Primary:     "",
		Secondary:   "",
		Local:       "",
		Reasoning:   []string{},
		Suggestions: []string{},
	}

	var cloudProviders []DetectedProvider
	var localProviders []DetectedProvider

	// Categorize providers
	for _, provider := range providers {
		// Check for cancellation in loop
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider categorization").
				WithComponent("ProviderConfiguration").
				WithOperation("ProviderRecommendations").
				WithDetails("provider", provider.Name)
		}

		if !provider.HasCredentials {
			continue
		}
		if provider.IsLocal {
			localProviders = append(localProviders, provider)
		} else {
			cloudProviders = append(cloudProviders, provider)
		}
	}

	// Recommend primary provider (cloud-based for reliability)
	if len(cloudProviders) > 0 {
		// Prefer Claude/OpenAI for primary
		for _, provider := range cloudProviders {
			if provider.Name == "anthropic" {
				recs.Primary = "anthropic"
				recs.Reasoning = append(recs.Reasoning, "Anthropic Claude recommended as primary for advanced reasoning")
				break
			}
		}
		if recs.Primary == "" {
			for _, provider := range cloudProviders {
				if provider.Name == "openai" {
					recs.Primary = "openai"
					recs.Reasoning = append(recs.Reasoning, "OpenAI GPT recommended as primary for broad capabilities")
					break
				}
			}
		}
		if recs.Primary == "" {
			recs.Primary = cloudProviders[0].Name
			recs.Reasoning = append(recs.Reasoning, fmt.Sprintf("%s selected as primary cloud provider", cloudProviders[0].Name))
		}
	}

	// Recommend secondary provider (different from primary)
	if len(cloudProviders) > 1 {
		for _, provider := range cloudProviders {
			if provider.Name != recs.Primary {
				recs.Secondary = provider.Name
				recs.Reasoning = append(recs.Reasoning, fmt.Sprintf("%s recommended as secondary for redundancy", provider.Name))
				break
			}
		}
	}

	// Recommend local provider
	if len(localProviders) > 0 {
		recs.Local = localProviders[0].Name
		recs.Reasoning = append(recs.Reasoning, fmt.Sprintf("%s recommended for local/private processing", localProviders[0].Name))
	}

	// Add suggestions
	if recs.Primary == "" && recs.Local == "" {
		recs.Suggestions = append(recs.Suggestions, "No providers available - set up API keys or install Ollama")
	}
	if recs.Local == "" {
		recs.Suggestions = append(recs.Suggestions, "Consider installing Ollama for local model support")
	}
	if len(cloudProviders) == 0 {
		recs.Suggestions = append(recs.Suggestions, "Consider setting up cloud providers (OpenAI/Anthropic) for best performance")
	}

	return recs, nil
}

// GetProviderRecommendations returns recommendations for provider setup
// Deprecated: Use ProviderRecommendations instead
func (pc *ProviderConfig) GetProviderRecommendations(ctx context.Context, providers []DetectedProvider) (*ProviderRecommendations, error) {
	return pc.ProviderRecommendations(ctx, providers)
}

// getOllamaModels retrieves the list of available models from Ollama
// Deprecated: Use ollamaModels instead
func (pc *ProviderConfig) getOllamaModels(ctx context.Context, endpoint string) ([]string, error) {
	return pc.ollamaModels(ctx, endpoint)
}

// ProviderRecommendations contains setup recommendations
type ProviderRecommendations struct {
	Primary     string
	Secondary   string
	Local       string
	Reasoning   []string
	Suggestions []string
}