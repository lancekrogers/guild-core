// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/providers"
)

// ProviderService monitors AI provider status and health
type ProviderService struct {
	ctx         context.Context
	guildConfig *config.GuildConfig

	// Provider status
	providerStatus map[string]ProviderStatus

	// Monitoring
	checkInterval time.Duration
	lastCheck     time.Time

	// Statistics
	totalRequests map[string]int
	totalErrors   map[string]int
	avgLatency    map[string]time.Duration
}

// ProviderStatus represents the status of an AI provider
type ProviderStatus struct {
	Name       string
	Type       string // openai, anthropic, ollama, etc.
	Available  bool
	LastCheck  time.Time
	ErrorCount int
	Latency    time.Duration
	RateLimit  *RateLimit
	Cost       *CostInfo
	Models     []string
}

// RateLimit represents rate limiting information
type RateLimit struct {
	RequestsPerMinute int
	RequestsRemaining int
	ResetTime         time.Time
}

// CostInfo represents cost tracking information
type CostInfo struct {
	InputTokensUsed  int64
	OutputTokensUsed int64
	TotalCost        float64
	Currency         string
}

// NewProviderService creates a new provider service
func NewProviderService(ctx context.Context, guildConfig *config.GuildConfig) (*ProviderService, error) {
	if guildConfig == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "guild config cannot be nil", nil).
			WithComponent("services.provider").
			WithOperation("NewProviderService")
	}

	return &ProviderService{
		ctx:            ctx,
		guildConfig:    guildConfig,
		providerStatus: make(map[string]ProviderStatus),
		checkInterval:  30 * time.Second,
		lastCheck:      time.Now(),
		totalRequests:  make(map[string]int),
		totalErrors:    make(map[string]int),
		avgLatency:     make(map[string]time.Duration),
	}, nil
}

// Start initializes the provider service
func (ps *ProviderService) Start() tea.Cmd {
	return func() tea.Msg {
		// Initialize provider status from config
		// TODO: Fix when config.Providers is properly typed
		// For now, initialize with mock providers
		mockProviders := []string{"openai", "anthropic", "ollama"}
		for _, providerName := range mockProviders {
			ps.providerStatus[providerName] = ProviderStatus{
				Name:      providerName,
				Type:      providerName,
				Available: false, // Will be checked
				LastCheck: time.Now(),
				Models:    []string{"default"},
			}
		}

		// Perform initial health check
		return ps.checkAllProviders()
	}
}

// CheckProviderHealth checks the health of a specific provider
func (ps *ProviderService) CheckProviderHealth(providerName string) tea.Cmd {
	return func() tea.Msg {
		provider, exists := ps.providerStatus[providerName]
		if !exists {
			return ProviderServiceErrorMsg{
				Operation: "check_health",
				Error: gerror.Newf(gerror.ErrCodeNotFound, "provider not found: %s", providerName).
					WithComponent("services.provider").
					WithOperation("CheckProviderHealth"),
			}
		}

		// Perform health check
		startTime := time.Now()
		isHealthy, err := ps.performHealthCheck(providerName)
		latency := time.Since(startTime)

		// Update status
		provider.Available = isHealthy
		provider.LastCheck = time.Now()
		provider.Latency = latency

		if err != nil {
			provider.ErrorCount++
			ps.totalErrors[providerName]++
		} else {
			ps.totalRequests[providerName]++
		}

		ps.providerStatus[providerName] = provider

		if err != nil {
			return ProviderHealthCheckFailedMsg{
				ProviderName: providerName,
				Error:        err,
				Latency:      latency,
			}
		}

		return ProviderHealthCheckSuccessMsg{
			ProviderName: providerName,
			Latency:      latency,
		}
	}
}

// CheckAllProviders checks the health of all configured providers
func (ps *ProviderService) CheckAllProviders() tea.Cmd {
	return func() tea.Msg {
		return ps.checkAllProviders()
	}
}

// StartPeriodicChecks starts periodic health checks
func (ps *ProviderService) StartPeriodicChecks() tea.Cmd {
	return tea.Tick(ps.checkInterval, func(t time.Time) tea.Msg {
		return HealthCheckRequestMsg{Timestamp: t}
	})
}

// GetProviderStatus returns the status of a specific provider
func (ps *ProviderService) GetProviderStatus(providerName string) (ProviderStatus, error) {
	status, exists := ps.providerStatus[providerName]
	if !exists {
		return ProviderStatus{}, gerror.Newf(gerror.ErrCodeNotFound, "provider not found: %s", providerName).
			WithComponent("services.provider").
			WithOperation("GetProviderStatus")
	}

	return status, nil
}

// GetAllProviderStatus returns the status of all providers
func (ps *ProviderService) GetAllProviderStatus() map[string]ProviderStatus {
	// Return a copy to prevent external modification
	statusCopy := make(map[string]ProviderStatus)
	for k, v := range ps.providerStatus {
		statusCopy[k] = v
	}
	return statusCopy
}

// GetAvailableProviders returns a list of currently available providers
func (ps *ProviderService) GetAvailableProviders() []string {
	var available []string
	for name, status := range ps.providerStatus {
		if status.Available {
			available = append(available, name)
		}
	}
	return available
}

// GetProviderSummary returns a summary of provider health
func (ps *ProviderService) GetProviderSummary() ProviderSummary {
	total := len(ps.providerStatus)
	available := len(ps.GetAvailableProviders())

	var totalRequests, totalErrors int
	var avgLatency time.Duration

	for _, count := range ps.totalRequests {
		totalRequests += count
	}
	for _, count := range ps.totalErrors {
		totalErrors += count
	}

	if len(ps.avgLatency) > 0 {
		var totalLatency time.Duration
		for _, latency := range ps.avgLatency {
			totalLatency += latency
		}
		avgLatency = totalLatency / time.Duration(len(ps.avgLatency))
	}

	return ProviderSummary{
		TotalProviders:       total,
		AvailableProviders:   available,
		UnavailableProviders: total - available,
		TotalRequests:        totalRequests,
		TotalErrors:          totalErrors,
		SuccessRate:          calculateSuccessRate(totalRequests, totalErrors),
		AverageLatency:       avgLatency,
		LastCheck:            ps.lastCheck,
	}
}

// SetCheckInterval sets the interval for periodic health checks
func (ps *ProviderService) SetCheckInterval(interval time.Duration) {
	ps.checkInterval = interval
}

// UpdateProviderCosts updates cost tracking for a provider
func (ps *ProviderService) UpdateProviderCosts(providerName string, inputTokens, outputTokens int64, cost float64) {
	if status, exists := ps.providerStatus[providerName]; exists {
		if status.Cost == nil {
			status.Cost = &CostInfo{
				Currency: "USD", // Default currency
			}
		}

		status.Cost.InputTokensUsed += inputTokens
		status.Cost.OutputTokensUsed += outputTokens
		status.Cost.TotalCost += cost

		ps.providerStatus[providerName] = status
	}
}

// GetStats returns statistics about the provider service
func (ps *ProviderService) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	summary := ps.GetProviderSummary()
	stats["total_providers"] = summary.TotalProviders
	stats["available_providers"] = summary.AvailableProviders
	stats["unavailable_providers"] = summary.UnavailableProviders
	stats["total_requests"] = summary.TotalRequests
	stats["total_errors"] = summary.TotalErrors
	stats["success_rate"] = summary.SuccessRate
	stats["average_latency"] = summary.AverageLatency.String()
	stats["last_check"] = summary.LastCheck.Format(time.RFC3339)
	stats["check_interval"] = ps.checkInterval.String()

	// Individual provider stats
	providerStats := make(map[string]interface{})
	for name, status := range ps.providerStatus {
		providerStats[name] = map[string]interface{}{
			"type":        status.Type,
			"available":   status.Available,
			"last_check":  status.LastCheck.Format(time.RFC3339),
			"error_count": status.ErrorCount,
			"latency":     status.Latency.String(),
			"models":      status.Models,
		}

		if status.Cost != nil {
			providerStats[name].(map[string]interface{})["cost"] = map[string]interface{}{
				"input_tokens":  status.Cost.InputTokensUsed,
				"output_tokens": status.Cost.OutputTokensUsed,
				"total_cost":    status.Cost.TotalCost,
				"currency":      status.Cost.Currency,
			}
		}
	}
	stats["providers"] = providerStats

	return stats
}

// checkAllProviders performs health checks on all providers
func (ps *ProviderService) checkAllProviders() tea.Msg {
	results := make([]ProviderHealthResult, 0)

	for providerName := range ps.providerStatus {
		startTime := time.Now()
		isHealthy, err := ps.performHealthCheck(providerName)
		latency := time.Since(startTime)

		// Update status
		status := ps.providerStatus[providerName]
		status.Available = isHealthy
		status.LastCheck = time.Now()
		status.Latency = latency

		if err != nil {
			status.ErrorCount++
			ps.totalErrors[providerName]++
		} else {
			ps.totalRequests[providerName]++
		}

		ps.providerStatus[providerName] = status

		results = append(results, ProviderHealthResult{
			ProviderName: providerName,
			IsHealthy:    isHealthy,
			Latency:      latency,
			Error:        err,
		})
	}

	ps.lastCheck = time.Now()

	return AllProvidersCheckedMsg{
		Results: results,
	}
}

// performHealthCheck performs an actual health check on a provider
func (ps *ProviderService) performHealthCheck(providerName string) (bool, error) {
	// Get provider config - simplified for now
	// TODO: Use actual config.ProviderConfig when available
	var providerType string
	if status, exists := ps.providerStatus[providerName]; exists {
		providerType = status.Type
	}

	if providerType == "" {
		return false, gerror.Newf(gerror.ErrCodeNotFound, "provider config not found: %s", providerName).
			WithComponent("services.provider").
			WithOperation("performHealthCheck")
	}

	// Perform provider-specific health check
	switch providerType {
	case "openai":
		return ps.checkOpenAIHealth(providerName)
	case "anthropic":
		return ps.checkAnthropicHealth(providerName)
	case "ollama":
		return ps.checkOllamaHealth(providerName)
	default:
		return ps.checkGenericHealth(providerName)
	}
}

// checkOpenAIHealth checks OpenAI provider health
func (ps *ProviderService) checkOpenAIHealth(providerName string) (bool, error) {
	baseURL := providers.EndpointOpenAI
	if ps.guildConfig != nil && ps.guildConfig.Providers.OpenAI.BaseURL != "" {
		baseURL = ps.guildConfig.Providers.OpenAI.BaseURL
	}
	url := strings.TrimSuffix(baseURL, "/") + "/models"

	ctx, cancel := context.WithTimeout(ps.ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeConnection, "request failed").
			WithComponent("services.provider").
			WithOperation("checkOpenAIHealth")
	}

	if apiKey := os.Getenv(providers.EnvOpenAIKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return false, gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "timeout").
				WithComponent("services.provider").
				WithOperation("checkOpenAIHealth")
		}
		return false, gerror.Wrap(err, gerror.ErrCodeConnection, "health request failed").
			WithComponent("services.provider").
			WithOperation("checkOpenAIHealth")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, gerror.Newf(gerror.ErrCodeProviderAPI, "status %d", resp.StatusCode).
			WithComponent("services.provider").
			WithOperation("checkOpenAIHealth")
	}

	return true, nil
}

// checkAnthropicHealth checks Anthropic provider health
func (ps *ProviderService) checkAnthropicHealth(providerName string) (bool, error) {
	baseURL := providers.EndpointAnthropic
	if ps.guildConfig != nil && ps.guildConfig.Providers.Anthropic.BaseURL != "" {
		baseURL = ps.guildConfig.Providers.Anthropic.BaseURL
	}
	url := strings.TrimSuffix(baseURL, "/") + "/v1/models"

	ctx, cancel := context.WithTimeout(ps.ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeConnection, "request failed").
			WithComponent("services.provider").
			WithOperation("checkAnthropicHealth")
	}
	if apiKey := os.Getenv(providers.EnvAnthropicKey); apiKey != "" {
		req.Header.Set("x-api-key", apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return false, gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "timeout").
				WithComponent("services.provider").
				WithOperation("checkAnthropicHealth")
		}
		return false, gerror.Wrap(err, gerror.ErrCodeConnection, "health request failed").
			WithComponent("services.provider").
			WithOperation("checkAnthropicHealth")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, gerror.Newf(gerror.ErrCodeProviderAPI, "status %d", resp.StatusCode).
			WithComponent("services.provider").
			WithOperation("checkAnthropicHealth")
	}

	return true, nil
}

// checkOllamaHealth checks Ollama provider health
func (ps *ProviderService) checkOllamaHealth(providerName string) (bool, error) {
	baseURL := providers.EndpointOllama
	if ps.guildConfig != nil && ps.guildConfig.Providers.Ollama.BaseURL != "" {
		baseURL = ps.guildConfig.Providers.Ollama.BaseURL
	}
	url := strings.TrimSuffix(baseURL, "/") + "/api/tags"

	ctx, cancel := context.WithTimeout(ps.ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeConnection, "request failed").
			WithComponent("services.provider").
			WithOperation("checkOllamaHealth")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return false, gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "timeout").
				WithComponent("services.provider").
				WithOperation("checkOllamaHealth")
		}
		return false, gerror.Wrap(err, gerror.ErrCodeConnection, "health request failed").
			WithComponent("services.provider").
			WithOperation("checkOllamaHealth")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, gerror.Newf(gerror.ErrCodeProviderAPI, "status %d", resp.StatusCode).
			WithComponent("services.provider").
			WithOperation("checkOllamaHealth")
	}

	return true, nil
}

// checkGenericHealth performs a generic health check
func (ps *ProviderService) checkGenericHealth(providerName string) (bool, error) {
	// Basic validation
	if providerName == "" {
		return false, gerror.New(gerror.ErrCodeInvalidInput, "provider name not configured", nil).
			WithComponent("services.provider").
			WithOperation("checkGenericHealth")
	}

	return true, nil
}

// calculateSuccessRate calculates the success rate percentage
func calculateSuccessRate(totalRequests, totalErrors int) float64 {
	if totalRequests == 0 {
		return 0.0
	}

	successful := totalRequests - totalErrors
	return (float64(successful) / float64(totalRequests)) * 100.0
}

// FormatStatus returns a human-readable status string for a provider
func (ps *ProviderService) FormatStatus(providerName string) string {
	status, err := ps.GetProviderStatus(providerName)
	if err != nil {
		return "❓ Unknown"
	}

	if status.Available {
		return fmt.Sprintf("🟢 Available (latency: %s)", status.Latency.Round(time.Millisecond))
	}

	return fmt.Sprintf("🔴 Unavailable (errors: %d)", status.ErrorCount)
}

// Data structures for provider service

// ProviderSummary represents an overview of all providers
type ProviderSummary struct {
	TotalProviders       int
	AvailableProviders   int
	UnavailableProviders int
	TotalRequests        int
	TotalErrors          int
	SuccessRate          float64
	AverageLatency       time.Duration
	LastCheck            time.Time
}

// ProviderHealthResult represents the result of a health check
type ProviderHealthResult struct {
	ProviderName string
	IsHealthy    bool
	Latency      time.Duration
	Error        error
}

// Message types for provider service communication

// ProviderServiceStartedMsg indicates the provider service has started
type ProviderServiceStartedMsg struct {
	ProviderCount int
}

// ProviderServiceErrorMsg represents a provider service error
type ProviderServiceErrorMsg struct {
	Operation string
	Error     error
}

// ProviderHealthCheckSuccessMsg indicates a successful health check
type ProviderHealthCheckSuccessMsg struct {
	ProviderName string
	Latency      time.Duration
}

// ProviderHealthCheckFailedMsg indicates a failed health check
type ProviderHealthCheckFailedMsg struct {
	ProviderName string
	Error        error
	Latency      time.Duration
}

// AllProvidersCheckedMsg indicates all providers have been checked
type AllProvidersCheckedMsg struct {
	Results []ProviderHealthResult
}

// HealthCheckRequestMsg triggers a health check
type HealthCheckRequestMsg struct {
	Timestamp time.Time
}
