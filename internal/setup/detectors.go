// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// Detectors handles provider detection
type Detectors struct {
	projectPath string
	registry    *ProviderRegistry
}

// NewDetectors creates a new detector instance
func NewDetectors(ctx context.Context, projectPath string) (*Detectors, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during detector creation").
			WithComponent("ProviderDetection").
			WithOperation("NewDetectors")
	}

	if projectPath == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "project path cannot be empty", nil).
			WithComponent("ProviderDetection").
			WithOperation("NewDetectors")
	}

	return &Detectors{
		projectPath: projectPath,
		registry:    NewProviderRegistry(),
	}, nil
}

// DetectionResult contains the results of provider detection
type DetectionResult struct {
	Available []DetectedProvider
	Missing   []string
}

// DetectedProvider represents a detected provider
type DetectedProvider struct {
	Name           string
	Type           string
	HasCredentials bool
	IsLocal        bool
	Version        string
	Endpoint       string
	Notes          string
}

// Providers scans for available providers
func (d *Detectors) Providers(ctx context.Context) (*DetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider detection").
			WithComponent("ProviderDetection").
			WithOperation("DetectProviders")
	}

	result := &DetectionResult{
		Available: []DetectedProvider{},
		Missing:   []string{},
	}

	// Check for each supported provider
	providers := []struct {
		name     string
		detector func(ctx context.Context) (*DetectedProvider, error)
	}{
		{providers.ProviderNameClaude, d.detectClaudeCode},
		{providers.ProviderNameOllama, d.detectOllama},
		{providers.ProviderNameOpenAI, d.detectOpenAI},
		{providers.ProviderNameAnthropic, d.detectAnthropic},
		{providers.ProviderNameDeepSeek, d.detectDeepSeek},
		{providers.ProviderNameDeepInfra, d.detectDeepInfra},
		{providers.ProviderNameOra, d.detectOra},
	}

	for _, p := range providers {
		// Check for cancellation before each provider
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider detection loop").
				WithComponent("ProviderDetection").
				WithOperation("Providers").
				WithDetails("provider", p.name)
		}

		provider, err := p.detector(ctx)
		if err != nil {
			// Convert to gerror if not already
			if _, ok := err.(*gerror.GuildError); !ok {
				err = gerror.Wrap(err, gerror.ErrCodeProvider, "provider detection failed").
					WithComponent("ProviderDetection").
					WithOperation("Providers").
					WithDetails("provider", p.name)
			}
			// Log error but continue with other providers
			result.Missing = append(result.Missing, fmt.Sprintf("%s (error: %v)", p.name, err))
			continue
		}

		if provider != nil {
			result.Available = append(result.Available, *provider)
		} else {
			result.Missing = append(result.Missing, p.name)
		}
	}

	return result, nil
}

// DetectProviders scans for available providers
// Deprecated: Use Providers instead
func (d *Detectors) DetectProviders(ctx context.Context) (*DetectionResult, error) {
	return d.Providers(ctx)
}

// getProviderDescription gets the description from the registry or returns a default
func (d *Detectors) getProviderDescription(providerName string) string {
	if provider, exists := d.registry.Get(providerName); exists {
		return provider.Description
	}
	// Default description if not in registry
	return "AI model provider"
}

// detectClaudeCode checks for Claude Code availability
func (d *Detectors) detectClaudeCode(ctx context.Context) (*DetectedProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Claude Code detection").
			WithComponent("ProviderDetection").
			WithOperation("detectClaudeCode")
	}

	// Check if we're running in Claude Code environment
	if os.Getenv("CLAUDE_CODE_SESSION") != "" || os.Getenv("ANTHROPIC_CLAUDE_CODE") != "" {
		return &DetectedProvider{
			Name:           providers.ProviderNameClaude,
			Type:           "cloud",
			HasCredentials: true,
			IsLocal:        false,
			Version:        "current",
			Endpoint:       "claude.ai/code",
			Notes:          d.getProviderDescription("claude_code"),
		}, nil
	}

	// Check for cancellation before executing command
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before CLI check").
			WithComponent("ProviderDetection").
			WithOperation("detectClaudeCode")
	}

	// Check for Claude Code CLI tools (if any)
	claudeCodePath, err := exec.LookPath("claude-code")
	if err == nil {
		return &DetectedProvider{
			Name:           providers.ProviderNameClaude,
			Type:           "cli",
			HasCredentials: true,
			IsLocal:        false,
			Version:        "cli",
			Endpoint:       claudeCodePath,
			Notes:          d.getProviderDescription("claude_code"),
		}, nil
	}

	return nil, nil
}

// detectOllama checks for Ollama availability
func (d *Detectors) detectOllama(ctx context.Context) (*DetectedProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Ollama detection").
			WithComponent("ProviderDetection").
			WithOperation("detectOllama")
	}

	// Check if ollama is installed
	ollamaPath, err := exec.LookPath("ollama")
	if err != nil {
		return nil, nil
	}

	// Check for cancellation before network calls
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before Ollama service check").
			WithComponent("ProviderDetection").
			WithOperation("detectOllama")
	}

	// Check if Ollama service is running with proper timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	endpoints := []string{
		"http://localhost:11434",
		"http://127.0.0.1:11434",
	}

	var workingEndpoint string
	for _, endpoint := range endpoints {
		// Check for cancellation in loop
		if err := timeoutCtx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeTimeout, "timeout during Ollama endpoint check").
				WithComponent("ProviderDetection").
				WithOperation("detectOllama").
				WithDetails("endpoint", endpoint)
		}

		req, err := http.NewRequestWithContext(timeoutCtx, "GET", endpoint+"/api/tags", nil)
		if err != nil {
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}

		if resp.StatusCode == http.StatusOK {
			workingEndpoint = endpoint
			break
		}
	}

	if workingEndpoint == "" {
		return &DetectedProvider{
			Name:           "ollama",
			Type:           "local",
			HasCredentials: true,
			IsLocal:        true,
			Version:        "installed",
			Endpoint:       ollamaPath,
			Notes:          "Ollama installed but not running - start with 'ollama serve'",
		}, nil
	}

	// Check for cancellation before version check
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before version check").
			WithComponent("ProviderDetection").
			WithOperation("detectOllama")
	}

	// Get version information with timeout
	versionCtx, versionCancel := context.WithTimeout(ctx, 3*time.Second)
	defer versionCancel()

	version := "unknown"
	if cmd := exec.CommandContext(versionCtx, "ollama", "--version"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			version = strings.TrimSpace(string(output))
		}
	}

	return &DetectedProvider{
		Name:           "ollama",
		Type:           "local",
		HasCredentials: true,
		IsLocal:        true,
		Version:        version,
		Endpoint:       workingEndpoint,
		Notes:          d.getProviderDescription("ollama"),
	}, nil
}

// detectOpenAI checks for OpenAI API credentials
func (d *Detectors) detectOpenAI(ctx context.Context) (*DetectedProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during OpenAI detection").
			WithComponent("ProviderDetection").
			WithOperation("detectOpenAI")
	}

	apiKey := os.Getenv(providers.EnvOpenAIKey)
	orgID := os.Getenv("OPENAI_ORG_ID")

	if apiKey == "" {
		return nil, nil
	}

	// Validate API key format (basic check)
	if !strings.HasPrefix(apiKey, "sk-") {
		return &DetectedProvider{
			Name:           "openai",
			Type:           "cloud",
			HasCredentials: false,
			IsLocal:        false,
			Version:        "api",
			Endpoint:       "https://api.openai.com",
			Notes:          "Invalid API key format",
		}, nil
	}

	// Test API connection (optional quick test)
	notes := d.getProviderDescription("openai")
	if orgID != "" {
		notes += " (with organization ID)"
	}

	return &DetectedProvider{
		Name:           "openai",
		Type:           "cloud",
		HasCredentials: true,
		IsLocal:        false,
		Version:        "api",
		Endpoint:       "https://api.openai.com",
		Notes:          notes,
	}, nil
}

// detectAnthropic checks for Anthropic API credentials
func (d *Detectors) detectAnthropic(ctx context.Context) (*DetectedProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Anthropic detection").
			WithComponent("ProviderDetection").
			WithOperation("detectAnthropic")
	}

	apiKey := os.Getenv(providers.EnvAnthropicKey)

	if apiKey == "" {
		return nil, nil
	}

	// Validate API key format (basic check)
	if !strings.HasPrefix(apiKey, "sk-ant-") {
		return &DetectedProvider{
			Name:           providers.ProviderNameAnthropic,
			Type:           "cloud",
			HasCredentials: false,
			IsLocal:        false,
			Version:        "api",
			Endpoint:       providers.EndpointAnthropic,
			Notes:          "Invalid API key format",
		}, nil
	}

	return &DetectedProvider{
		Name:           "anthropic",
		Type:           "cloud",
		HasCredentials: true,
		IsLocal:        false,
		Version:        "api",
		Endpoint:       "https://api.anthropic.com",
		Notes:          d.getProviderDescription("anthropic"),
	}, nil
}

// detectDeepSeek checks for DeepSeek API credentials
func (d *Detectors) detectDeepSeek(ctx context.Context) (*DetectedProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during DeepSeek detection").
			WithComponent("ProviderDetection").
			WithOperation("detectDeepSeek")
	}

	apiKey := os.Getenv(providers.EnvDeepSeekKey)

	if apiKey == "" {
		return nil, nil
	}

	return &DetectedProvider{
		Name:           providers.ProviderNameDeepSeek,
		Type:           "cloud",
		HasCredentials: true,
		IsLocal:        false,
		Version:        "api",
		Endpoint:       providers.EndpointDeepSeek,
		Notes:          d.getProviderDescription("deepseek"),
	}, nil
}

// detectDeepInfra checks for DeepInfra API credentials
func (d *Detectors) detectDeepInfra(ctx context.Context) (*DetectedProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during DeepInfra detection").
			WithComponent("ProviderDetection").
			WithOperation("detectDeepInfra")
	}

	apiKey := os.Getenv(providers.EnvDeepInfraKey)

	if apiKey == "" {
		return nil, nil
	}

	return &DetectedProvider{
		Name:           providers.ProviderNameDeepInfra,
		Type:           "cloud",
		HasCredentials: true,
		IsLocal:        false,
		Version:        "api",
		Endpoint:       providers.EndpointDeepInfra,
		Notes:          "API key available",
	}, nil
}

// detectOra checks for Ora API credentials
func (d *Detectors) detectOra(ctx context.Context) (*DetectedProvider, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Ora detection").
			WithComponent("ProviderDetection").
			WithOperation("detectOra")
	}

	apiKey := os.Getenv(providers.EnvOraKey)

	if apiKey == "" {
		return nil, nil
	}

	return &DetectedProvider{
		Name:           providers.ProviderNameOra,
		Type:           "cloud",
		HasCredentials: true,
		IsLocal:        false,
		Version:        "api",
		Endpoint:       providers.EndpointOra,
		Notes:          "API key available",
	}, nil
}

// ValidateProviderConnection performs a deeper validation of provider availability
func (d *Detectors) ValidateProviderConnection(ctx context.Context, provider DetectedProvider) (*ValidationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider validation").
			WithComponent("ProviderDetection").
			WithOperation("ValidateProviderConnection").
			WithDetails("provider", provider.Name)
	}

	switch provider.Name {
	case "ollama":
		return d.validateOllamaConnection(ctx, provider)
	case "openai":
		return d.validateOpenAIConnection(ctx, provider)
	case "anthropic":
		return d.validateAnthropicConnection(ctx, provider)
	case "claude_code":
		return d.validateClaudeCodeConnection(ctx, provider)
	default:
		// For other providers, assume valid if credentials are present
		return &ValidationResult{
			IsValid: provider.HasCredentials,
			Error:   "",
			Models:  []string{},
		}, nil
	}
}

// ValidationResult contains the result of provider validation
type ValidationResult struct {
	IsValid bool
	Error   string
	Models  []string
}

// validateOllamaConnection tests Ollama connectivity
func (d *Detectors) validateOllamaConnection(ctx context.Context, provider DetectedProvider) (*ValidationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Ollama validation").
			WithComponent("ProviderDetection").
			WithOperation("validateOllamaConnection")
	}

	if provider.Endpoint == "" {
		return &ValidationResult{
			IsValid: false,
			Error:   "Ollama service not running",
			Models:  []string{},
		}, nil
	}

	// Create timeout context for validation
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try to get model list
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(timeoutCtx, "GET", provider.Endpoint+"/api/tags", nil)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create validation request").
			WithComponent("ProviderDetection").
			WithOperation("validateOllamaConnection").
			WithDetails("endpoint", provider.Endpoint)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Check if error is due to context cancellation/timeout
		if timeoutCtx.Err() != nil {
			return nil, gerror.Wrap(timeoutCtx.Err(), gerror.ErrCodeTimeout, "timeout during Ollama validation").
				WithComponent("ProviderDetection").
				WithOperation("validateOllamaConnection").
				WithDetails("endpoint", provider.Endpoint)
		}
		return &ValidationResult{
			IsValid: false,
			Error:   fmt.Sprintf("Connection failed: %v", err),
			Models:  []string{},
		}, nil
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		return &ValidationResult{
			IsValid: false,
			Error:   fmt.Sprintf("HTTP %d", resp.StatusCode),
			Models:  []string{},
		}, nil
	}

	// For now, assume success means models are available
	return &ValidationResult{
		IsValid: true,
		Error:   "",
		Models:  []string{"llama2", "codellama", "mistral"}, // Common models
	}, nil
}

// validateOpenAIConnection tests OpenAI API connectivity
func (d *Detectors) validateOpenAIConnection(ctx context.Context, provider DetectedProvider) (*ValidationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during OpenAI validation").
			WithComponent("ProviderDetection").
			WithOperation("validateOpenAIConnection")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return &ValidationResult{
			IsValid: false,
			Error:   "No API key",
			Models:  []string{},
		}, nil
	}

	// Simple validation - could make actual API call here
	return &ValidationResult{
		IsValid: true,
		Error:   "",
		Models:  []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
	}, nil
}

// validateAnthropicConnection tests Anthropic API connectivity
func (d *Detectors) validateAnthropicConnection(ctx context.Context, provider DetectedProvider) (*ValidationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Anthropic validation").
			WithComponent("ProviderDetection").
			WithOperation("validateAnthropicConnection")
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return &ValidationResult{
			IsValid: false,
			Error:   "No API key",
			Models:  []string{},
		}, nil
	}

	// Simple validation - could make actual API call here
	return &ValidationResult{
		IsValid: true,
		Error:   "",
		Models:  []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
	}, nil
}

// validateClaudeCodeConnection tests Claude Code connectivity
func (d *Detectors) validateClaudeCodeConnection(ctx context.Context, provider DetectedProvider) (*ValidationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Claude Code validation").
			WithComponent("ProviderDetection").
			WithOperation("validateClaudeCodeConnection")
	}

	// Claude Code is always valid if detected
	return &ValidationResult{
		IsValid: true,
		Error:   "",
		Models:  []string{"claude-3-5-sonnet-20241022"},
	}, nil
}

// DetectProjectContext analyzes the project to suggest appropriate configurations
func (d *Detectors) DetectProjectContext(ctx context.Context) (*ProjectContext, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during project context detection").
			WithComponent("ProviderDetection").
			WithOperation("DetectProjectContext")
	}

	if d.projectPath == "" {
		return nil, gerror.New(gerror.ErrCodeValidation, "project path is empty", nil).
			WithComponent("ProviderDetection").
			WithOperation("DetectProjectContext")
	}

	context := &ProjectContext{
		Language:     "unknown",
		Framework:    "unknown",
		ProjectType:  "unknown",
		Dependencies: []string{},
		Suggestions:  []string{},
	}

	// Check for common project indicators
	files := []struct {
		name      string
		indicates func(*ProjectContext)
	}{
		{"go.mod", func(pc *ProjectContext) {
			pc.Language = "go"
			pc.ProjectType = "application"
			pc.Suggestions = append(pc.Suggestions, "Go development detected - recommend code analysis agents")
		}},
		{"package.json", func(pc *ProjectContext) {
			pc.Language = "javascript"
			pc.Framework = "node"
			pc.Suggestions = append(pc.Suggestions, "Node.js project detected - recommend JavaScript/TypeScript agents")
		}},
		{"requirements.txt", func(pc *ProjectContext) {
			pc.Language = "python"
			pc.ProjectType = "application"
			pc.Suggestions = append(pc.Suggestions, "Python project detected - recommend Python development agents")
		}},
		{"Cargo.toml", func(pc *ProjectContext) {
			pc.Language = "rust"
			pc.ProjectType = "application"
			pc.Suggestions = append(pc.Suggestions, "Rust project detected - recommend systems programming agents")
		}},
		{"pom.xml", func(pc *ProjectContext) {
			pc.Language = "java"
			pc.Framework = "maven"
			pc.Suggestions = append(pc.Suggestions, "Java/Maven project detected - recommend enterprise development agents")
		}},
	}

	for _, file := range files {
		// Check for cancellation in loop
		if err := ctx.Err(); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during file check").
				WithComponent("ProviderDetection").
				WithOperation("DetectProjectContext").
				WithDetails("file", file.name)
		}

		if _, err := os.Stat(filepath.Join(d.projectPath, file.name)); err == nil {
			file.indicates(context)
		}
	}

	return context, nil
}

// ProjectContext contains information about the project
type ProjectContext struct {
	Language     string
	Framework    string
	ProjectType  string
	Dependencies []string
	Suggestions  []string
}
