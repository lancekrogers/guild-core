// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package providers includes auto-detection capabilities for LLM providers.
// This module can automatically detect Claude Code CLI and Ollama service availability,
// enabling seamless provider selection and fallback strategies.
package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/providers/interfaces"
)

// DetectionResult contains information about a detected provider
type DetectionResult struct {
	Provider     ProviderType `json:"provider"`
	Available    bool         `json:"available"`
	Version      string       `json:"version,omitempty"`
	Path         string       `json:"path,omitempty"`
	Endpoint     string       `json:"endpoint,omitempty"`
	Error        string       `json:"error,omitempty"`
	DetectedAt   time.Time    `json:"detected_at"`
	Confidence   float64      `json:"confidence"` // 0.0 - 1.0
	Capabilities []string     `json:"capabilities,omitempty"`
}

// AutoDetector handles automatic detection of available providers
type AutoDetector struct {
	timeout      time.Duration
	httpClient   *http.Client
	capabilities map[ProviderType][]string
}

// NewAutoDetector creates a new provider auto-detector
func NewAutoDetector(timeout time.Duration) *AutoDetector {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &AutoDetector{
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		capabilities: map[ProviderType][]string{
			ProviderClaudeCode: {
				"chat", "streaming", "mcp-tools", "file-processing",
				"code-generation", "code-review", "debugging",
			},
			ProviderOllama: {
				"chat", "streaming", "embeddings", "local-models",
				"vision", "custom-models",
			},
		},
	}
}

// DetectAll detects all available providers
func (d *AutoDetector) DetectAll(ctx context.Context) ([]DetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before detection").
			WithComponent("providers").
			WithOperation("DetectAll")
	}

	var results []DetectionResult

	// Detect Claude Code CLI
	claudeResult, err := d.detectClaudeCode(ctx)
	if err != nil {
		// Log error but continue detection
		claudeResult = DetectionResult{
			Provider:   ProviderClaudeCode,
			Available:  false,
			Error:      err.Error(),
			DetectedAt: time.Now(),
			Confidence: 0.0,
		}
	}
	results = append(results, claudeResult)

	// Detect Ollama service
	ollamaResult, err := d.detectOllama(ctx)
	if err != nil {
		// Log error but continue detection
		ollamaResult = DetectionResult{
			Provider:   ProviderOllama,
			Available:  false,
			Error:      err.Error(),
			DetectedAt: time.Now(),
			Confidence: 0.0,
		}
	}
	results = append(results, ollamaResult)

	return results, nil
}

// DetectClaudeCode detects Claude Code CLI availability
func (d *AutoDetector) DetectClaudeCode(ctx context.Context) (*DetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before detection").
			WithComponent("providers").
			WithOperation("DetectClaudeCode")
	}

	result, err := d.detectClaudeCode(ctx)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DetectOllama detects Ollama service availability
func (d *AutoDetector) DetectOllama(ctx context.Context) (*DetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before detection").
			WithComponent("providers").
			WithOperation("DetectOllama")
	}

	result, err := d.detectOllama(ctx)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// detectClaudeCode performs Claude Code CLI detection
func (d *AutoDetector) detectClaudeCode(ctx context.Context) (DetectionResult, error) {
	result := DetectionResult{
		Provider:   ProviderClaudeCode,
		DetectedAt: time.Now(),
	}

	// Simple detection using 'which claude'
	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "which", "claude")
	output, err := cmd.Output()
	if err != nil {
		// Claude CLI not found - this is expected, not an error
		result.Available = false
		result.Error = "Claude CLI not found (which claude failed)"
		result.Confidence = 0.0
		// Don't wrap with gerror - this is an expected case, not an error
		return result, nil
	}

	// Claude found
	claudePath := strings.TrimSpace(string(output))
	if claudePath != "" {
		result.Available = true
		result.Path = claudePath
		result.Version = "Claude CLI"
		result.Confidence = 1.0
		result.Capabilities = d.capabilities[ProviderClaudeCode]
		return result, nil
	}

	result.Available = false
	result.Error = "Claude CLI not found in PATH"
	result.Confidence = 0.0

	return result, nil
}

// detectOllama performs Ollama service detection
func (d *AutoDetector) detectOllama(ctx context.Context) (DetectionResult, error) {
	result := DetectionResult{
		Provider:   ProviderOllama,
		DetectedAt: time.Now(),
	}

	// Try common Ollama endpoints
	endpoints := d.getOllamaEndpoints()

	for _, endpoint := range endpoints {
		if err := ctx.Err(); err != nil {
			return result, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during detection").
				WithComponent("providers").
				WithOperation("detectOllama")
		}

		// Test HTTP connectivity to Ollama API
		if available, version, err := d.testOllamaEndpoint(ctx, endpoint); err == nil && available {
			result.Available = true
			result.Endpoint = endpoint
			result.Version = version
			result.Confidence = d.calculateOllamaConfidence(endpoint, version)
			result.Capabilities = d.capabilities[ProviderOllama]
			return result, nil
		}
	}

	result.Available = false
	result.Error = "Ollama service not running on standard ports"
	result.Confidence = 0.0

	return result, nil
}

// getClaudeCodeCandidates returns potential Claude Code binary locations
func (d *AutoDetector) getClaudeCodeCandidates() []string {
	candidates := []string{
		"claude", // Standard PATH lookup - this is the correct Claude CLI binary
	}

	// Add platform-specific paths
	switch runtime.GOOS {
	case "darwin":
		candidates = append(candidates,
			"/usr/local/bin/claude",
			"/opt/homebrew/bin/claude",
			"/Applications/Claude.app/Contents/MacOS/claude",
		)
	case "linux":
		candidates = append(candidates,
			"/usr/local/bin/claude",
			"/usr/bin/claude",
			"~/.local/bin/claude",
		)
	case "windows":
		candidates = append(candidates,
			"claude.exe",
			"C:\\Program Files\\Claude\\claude.exe",
			"C:\\Program Files (x86)\\Claude\\claude.exe",
		)
	}

	return candidates
}

// getOllamaEndpoints returns potential Ollama service endpoints
func (d *AutoDetector) getOllamaEndpoints() []string {
	return []string{
		"http://localhost:11434", // Default Ollama port
		"http://127.0.0.1:11434", // Explicit localhost
		"http://0.0.0.0:11434",   // All interfaces
		"http://localhost:11435", // Alternative port
		"http://localhost:8080",  // Some configurations
	}
}

// testClaudeCodeBinary tests if a Claude Code binary is available and working
func (d *AutoDetector) testClaudeCodeBinary(ctx context.Context, path string) (bool, string, error) {
	// Create a context with timeout for the command
	cmdCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	// Try to get version information
	cmd := exec.CommandContext(cmdCtx, path, "--version")

	output, err := cmd.Output()
	if err != nil {
		// Try alternative version command
		cmd = exec.CommandContext(cmdCtx, path, "version")
		output, err = cmd.Output()
		if err != nil {
			return false, "", gerror.Wrap(err, gerror.ErrCodeExternal, "failed to execute claude binary").
				WithComponent("providers").
				WithOperation("testClaudeCodeBinary").
				WithDetails("path", path)
		}
	}

	version := strings.TrimSpace(string(output))

	// Validate that this looks like Claude Code output
	if d.isValidClaudeCodeOutput(version) {
		return true, version, nil
	}

	return false, "", gerror.New(gerror.ErrCodeValidation, "unexpected claude binary output", nil).
		WithComponent("providers").
		WithOperation("testClaudeCodeBinary").
		WithDetails("path", path).
		WithDetails("output", version)
}

// testOllamaEndpoint tests if an Ollama endpoint is available and responding
func (d *AutoDetector) testOllamaEndpoint(ctx context.Context, endpoint string) (bool, string, error) {
	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/version", nil)
	if err != nil {
		return false, "", gerror.Wrap(err, gerror.ErrCodeExternal, "failed to create HTTP request").
			WithComponent("providers").
			WithOperation("testOllamaEndpoint").
			WithDetails("endpoint", endpoint)
	}

	// Execute request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return false, "", gerror.Wrap(err, gerror.ErrCodeExternal, "failed to connect to ollama endpoint").
			WithComponent("providers").
			WithOperation("testOllamaEndpoint").
			WithDetails("endpoint", endpoint)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log close error but don't override main error
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return false, "", gerror.Newf(gerror.ErrCodeExternal, "ollama endpoint returned status %d", resp.StatusCode).
			WithComponent("providers").
			WithOperation("testOllamaEndpoint").
			WithDetails("endpoint", endpoint).
			WithDetails("status_code", resp.StatusCode)
	}

	// Try to parse version from response
	var versionResp struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		// Version parsing failed, but service is available
		return true, "unknown", nil
	}

	if versionResp.Version == "" {
		return true, "unknown", nil
	}

	return true, versionResp.Version, nil
}

// isValidClaudeCodeOutput validates Claude Code binary output
func (d *AutoDetector) isValidClaudeCodeOutput(output string) bool {
	output = strings.ToLower(output)
	return strings.Contains(output, "claude") ||
		strings.Contains(output, "anthropic") ||
		strings.Contains(output, "version")
}

// calculateClaudeCodeConfidence calculates confidence for Claude Code detection
func (d *AutoDetector) calculateClaudeCodeConfidence(path, version string) float64 {
	confidence := 0.5 // Base confidence for working binary

	// Higher confidence for standard installation paths
	if path == "claude" {
		confidence += 0.3
	}

	// Higher confidence for version containing "claude"
	if strings.Contains(strings.ToLower(version), "claude") {
		confidence += 0.2
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// calculateOllamaConfidence calculates confidence for Ollama detection
func (d *AutoDetector) calculateOllamaConfidence(endpoint, version string) float64 {
	confidence := 0.6 // Base confidence for working endpoint

	// Higher confidence for standard port
	if strings.Contains(endpoint, ":11434") {
		confidence += 0.3
	}

	// Higher confidence for valid version
	if version != "unknown" && version != "" {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetBestProvider returns the best available provider based on detection results
func (d *AutoDetector) GetBestProvider(ctx context.Context, preferredProviders []ProviderType) (*DetectionResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before provider selection").
			WithComponent("providers").
			WithOperation("GetBestProvider")
	}

	// Detect all providers
	results, err := d.DetectAll(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeProvider, "failed to detect providers").
			WithComponent("providers").
			WithOperation("GetBestProvider")
	}

	// Filter available providers
	var available []DetectionResult
	for _, result := range results {
		if result.Available {
			available = append(available, result)
		}
	}

	if len(available) == 0 {
		return nil, gerror.New(gerror.ErrCodeProvider, "no providers available", nil).
			WithComponent("providers").
			WithOperation("GetBestProvider")
	}

	// Return preferred provider if available
	for _, preferred := range preferredProviders {
		for _, result := range available {
			if result.Provider == preferred {
				return &result, nil
			}
		}
	}

	// Return provider with highest confidence
	best := available[0]
	for _, result := range available[1:] {
		if result.Confidence > best.Confidence {
			best = result
		}
	}

	return &best, nil
}

// CreateClientFromDetection creates a client from detection result
func (d *AutoDetector) CreateClientFromDetection(result DetectionResult) (interfaces.AIProvider, error) {
	if !result.Available {
		return nil, gerror.New(gerror.ErrCodeProvider, "cannot create client from unavailable provider", nil).
			WithComponent("providers").
			WithOperation("CreateClientFromDetection").
			WithDetails("provider", string(result.Provider))
	}

	switch result.Provider {
	case ProviderClaudeCode:
		// Note: Claude Code uses different interface (LLMClient), not AIProvider
		return nil, gerror.New(gerror.ErrCodeNotImplemented, "Claude Code uses LLMClient interface, not AIProvider", nil).
			WithComponent("providers").
			WithOperation("CreateClientFromDetection").
			WithDetails("provider", string(result.Provider))

	case ProviderOllama:
		// Import would be needed here: "github.com/guild-ventures/guild-core/pkg/providers/ollama"
		// For now, return instruction to use factory
		return nil, gerror.New(gerror.ErrCodeNotImplemented, "use FactoryV2.CreateAIProvider for Ollama client creation", nil).
			WithComponent("providers").
			WithOperation("CreateClientFromDetection").
			WithDetails("provider", string(result.Provider)).
			WithDetails("endpoint", result.Endpoint)

	default:
		return nil, gerror.New(gerror.ErrCodeProvider, "unsupported provider for auto-detection", nil).
			WithComponent("providers").
			WithOperation("CreateClientFromDetection").
			WithDetails("provider", string(result.Provider))
	}
}

// ValidateProvider validates that a provider is properly configured and available
func (d *AutoDetector) ValidateProvider(ctx context.Context, providerType ProviderType) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before validation").
			WithComponent("providers").
			WithOperation("ValidateProvider")
	}

	var result DetectionResult
	var err error

	switch providerType {
	case ProviderClaudeCode:
		result, err = d.detectClaudeCode(ctx)
	case ProviderOllama:
		result, err = d.detectOllama(ctx)
	default:
		return gerror.New(gerror.ErrCodeProvider, "provider validation not supported", nil).
			WithComponent("providers").
			WithOperation("ValidateProvider").
			WithDetails("provider", string(providerType))
	}

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeProvider, "provider validation failed").
			WithComponent("providers").
			WithOperation("ValidateProvider").
			WithDetails("provider", string(providerType))
	}

	if !result.Available {
		return gerror.New(gerror.ErrCodeProvider, "provider not available", nil).
			WithComponent("providers").
			WithOperation("ValidateProvider").
			WithDetails("provider", string(providerType)).
			WithDetails("error", result.Error)
	}

	return nil
}
