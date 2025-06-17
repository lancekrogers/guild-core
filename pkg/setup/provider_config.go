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
	switch provider.Name {
	case "claude_code":
		return pc.validateClaudeCode(ctx, provider)
	case "ollama":
		return pc.validateOllama(ctx, provider)
	case "openai":
		return pc.validateOpenAI(ctx, provider)
	case "anthropic":
		return pc.validateAnthropic(ctx, provider)
	case "deepseek":
		return pc.validateDeepSeek(ctx, provider)
	case "deepinfra":
		return pc.validateDeepInfra(ctx, provider)
	case "ora":
		return pc.validateOra(ctx, provider)
	default:
		return nil, gerror.Newf(gerror.ErrCodeValidation, "unsupported provider: %s", provider.Name).
			WithComponent("setup").
			WithOperation("ValidateProvider")
	}
}

// validateClaudeCode validates Claude Code configuration
func (pc *ProviderConfig) validateClaudeCode(ctx context.Context, provider DetectedProvider) (*ProviderValidation, error) {
	// Claude Code is always valid if detected
	settings := map[string]string{
		"type":        "claude_code",
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
	settings := map[string]string{
		"type": "ollama",
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
	models, err := pc.getOllamaModels(ctx, provider.Endpoint)
	if err != nil {
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
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    "OPENAI_API_KEY environment variable not set",
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  "Set your OpenAI API key: export OPENAI_API_KEY=sk-...",
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
		"type":     "openai",
		"base_url": "https://api.openai.com/v1",
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
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    "ANTHROPIC_API_KEY environment variable not set",
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  "Set your Anthropic API key: export ANTHROPIC_API_KEY=sk-ant-...",
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
		"type":     "anthropic",
		"base_url": "https://api.anthropic.com",
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
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    "DEEPSEEK_API_KEY environment variable not set",
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  "Set your DeepSeek API key: export DEEPSEEK_API_KEY=<your-key>",
		}, nil
	}

	settings := map[string]string{
		"type":     "deepseek",
		"base_url": "https://api.deepseek.com",
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
	apiKey := os.Getenv("DEEPINFRA_API_KEY")
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    "DEEPINFRA_API_KEY environment variable not set",
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  "Set your DeepInfra API key: export DEEPINFRA_API_KEY=<your-key>",
		}, nil
	}

	settings := map[string]string{
		"type":     "deepinfra",
		"base_url": "https://api.deepinfra.com/v1/openai",
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
	apiKey := os.Getenv("ORA_API_KEY")
	if apiKey == "" {
		return &ProviderValidation{
			IsValid:  false,
			Error:    "ORA_API_KEY environment variable not set",
			Settings: map[string]string{},
			Models:   []string{},
			Warning:  "Set your Ora API key: export ORA_API_KEY=<your-key>",
		}, nil
	}

	settings := map[string]string{
		"type":     "ora",
		"base_url": "https://api.ora.sh/v1",
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

// getOllamaModels retrieves the list of available models from Ollama
func (pc *ProviderConfig) getOllamaModels(ctx context.Context, endpoint string) ([]string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create request").
			WithComponent("setup").
			WithOperation("getOllamaModels")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to Ollama").
			WithComponent("setup").
			WithOperation("getOllamaModels")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, gerror.Newf(gerror.ErrCodeConnection, "Ollama API returned status: %d", resp.StatusCode).
			WithComponent("setup").
			WithOperation("getOllamaModels")
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

	// Test with a simple completion
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err = client.Complete(testCtx, "Hello, respond with just 'OK'")
	latency := time.Since(start)

	if err != nil {
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

// GetProviderRecommendations returns recommendations for provider setup
func (pc *ProviderConfig) GetProviderRecommendations(ctx context.Context, providers []DetectedProvider) (*ProviderRecommendations, error) {
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

// ProviderRecommendations contains setup recommendations
type ProviderRecommendations struct {
	Primary     string
	Secondary   string
	Local       string
	Reasoning   []string
	Suggestions []string
}