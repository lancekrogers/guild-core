// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// Config holds the setup wizard configuration
type Config struct {
	ProjectPath  string // Path to the Guild project
	QuickMode    bool   // Use quick setup with defaults
	Force        bool   // Force setup even if already configured
	ProviderOnly string // Setup only this provider
}

// Wizard manages the interactive setup process
type Wizard struct {
	config         *Config
	detectors      *Detectors
	providerConfig *ProviderConfig
	modelConfig    *ModelConfig
	agentConfig    *AgentConfig
	agentPresets   *AgentPresets
	registry       *ProviderRegistry
}

// NewWizard creates a new setup wizard
func NewWizard(ctx context.Context, config *Config) (*Wizard, error) {
	if config == nil {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "config is required", nil).
			WithComponent("setup").
			WithOperation("NewWizard")
	}

	// Initialize components
	detectors, err := NewDetectors(ctx, config.ProjectPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create detectors").
			WithComponent("setup").
			WithOperation("NewWizard")
	}

	providerConfig, err := NewProviderConfig(ctx, config.ProjectPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create provider config").
			WithComponent("setup").
			WithOperation("NewWizard")
	}

	modelConfig, err := NewModelConfig(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create model config").
			WithComponent("setup").
			WithOperation("NewWizard")
	}

	agentConfig, err := NewAgentConfig(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent config").
			WithComponent("setup").
			WithOperation("NewWizard")
	}

	agentPresets, err := NewAgentPresets(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agent presets").
			WithComponent("setup").
			WithOperation("NewWizard")
	}

	wizard := &Wizard{
		config:         config,
		detectors:      detectors,
		providerConfig: providerConfig,
		modelConfig:    modelConfig,
		agentConfig:    agentConfig,
		agentPresets:   agentPresets,
		registry:       NewProviderRegistry(),
	}

	return wizard, nil
}

// RunQuickMode executes the setup wizard in quick mode without TUI
func (w *Wizard) RunQuickMode(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before wizard execution").
			WithComponent("SetupWizard").
			WithOperation("RunQuickMode")
	}

	// Detect providers
	providers, err := w.DetectProviders(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect providers").
			WithComponent("SetupWizard").
			WithOperation("RunQuickMode")
	}

	// Select first available provider
	if len(providers) == 0 {
		return gerror.New(gerror.ErrCodeNotFound, "no providers detected", nil).
			WithComponent("SetupWizard").
			WithOperation("RunQuickMode")
	}

	selectedProvider := providers[0]
	configured, err := w.ConfigureProvider(ctx, selectedProvider)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to configure provider").
			WithComponent("SetupWizard").
			WithOperation("RunQuickMode")
	}

	// Create agents
	agents, err := w.CreateAgents(ctx, []ConfiguredProvider{*configured})
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agents").
			WithComponent("SetupWizard").
			WithOperation("RunQuickMode")
	}

	// Save configuration
	if err := w.SaveConfiguration(ctx, []ConfiguredProvider{*configured}, agents); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to save configuration").
			WithComponent("SetupWizard").
			WithOperation("RunQuickMode")
	}

	return nil
}

// DetectProviders is called by the TUI to detect available providers
func (w *Wizard) DetectProviders(ctx context.Context) ([]DetectedProvider, error) {
	detection, err := w.detectors.Providers(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect providers").
			WithComponent("SetupWizard").
			WithOperation("detectProviders")
	}

	var providers []DetectedProvider
	for _, p := range detection.Available {
		providers = append(providers, p)
	}

	return providers, nil
}

// ConfigureProvider is called by the TUI to configure a provider
func (w *Wizard) ConfigureProvider(ctx context.Context, provider DetectedProvider) (*ConfiguredProvider, error) {
	// Check context at start
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider configuration").
			WithComponent("SetupWizard").
			WithOperation("configureProvider").
			WithDetails("provider", provider.Name)
	}

	// Validate provider configuration
	validated, err := w.providerConfig.ValidateProvider(ctx, provider)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeValidation, "failed to validate provider %s", provider.Name).
			WithComponent("SetupWizard").
			WithOperation("configureProvider")
	}

	if !validated.IsValid {
		return nil, gerror.Newf(gerror.ErrCodeValidation, "%s validation failed: %s", provider.Name, validated.Error).
			WithComponent("SetupWizard").
			WithOperation("configureProvider")
	}

	// Get available models for this provider
	models, err := w.modelConfig.GetModelsForProvider(ctx, provider.Name)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to get models for provider %s", provider.Name).
			WithComponent("SetupWizard").
			WithOperation("configureProvider")
	}

	// For now, select recommended models automatically
	// TODO: Allow TUI to select models
	selectedModels := w.selectRecommendedModels(models)

	return &ConfiguredProvider{
		Name:     provider.Name,
		Type:     provider.Type,
		Models:   selectedModels,
		Settings: validated.Settings,
	}, nil
}

// selectRecommendedModels selects recommended models from the list
func (w *Wizard) selectRecommendedModels(models []ModelInfo) []ModelInfo {
	var selected []ModelInfo
	for _, model := range models {
		if model.Recommended {
			selected = append(selected, model)
		}
	}

	// If no recommended models, select the first one
	if len(selected) == 0 && len(models) > 0 {
		selected = append(selected, models[0])
	}

	return selected
}

// CreateAgents is called by the TUI to create agent configurations
func (w *Wizard) CreateAgents(ctx context.Context, providers []ConfiguredProvider) ([]config.AgentConfig, error) {
	// Detect project context
	projectContext, err := w.detectors.DetectProjectContext(ctx)
	if err != nil {
		// If detection fails, continue with default context
		projectContext = &ProjectContext{
			ProjectType: "general",
			Language:    "go",
		}
	}

	// Use intelligent preset selection for both QuickMode and regular mode
	// The TUI will handle user interaction if needed
	return w.createAgentsFromPresets(ctx, providers, projectContext)
}

// createAgentsFromPresets creates agents based on intelligent preset selection
func (w *Wizard) createAgentsFromPresets(ctx context.Context, providers []ConfiguredProvider, projectContext *ProjectContext) ([]config.AgentConfig, error) {
	// Get recommended presets based on providers and project context
	recommendations, err := w.agentPresets.RecommendPresets(ctx, providers, projectContext)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get preset recommendations").
			WithComponent("SetupWizard").
			WithOperation("createAgentsFromPresets")
	}

	if len(recommendations) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no suitable agent presets found", nil).
			WithComponent("SetupWizard").
			WithOperation("createAgentsFromPresets")
	}

	// Use the top recommendation
	recommendation := recommendations[0]

	// Adapt the preset for available providers
	adaptedPreset, err := w.agentPresets.AdaptPresetForProviders(ctx, recommendation.Collection, providers)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to adapt preset %s", recommendation.Collection.ID).
			WithComponent("SetupWizard").
			WithOperation("createAgentsFromPresets")
	}

	// Convert preset agents to config.AgentConfig
	var agents []config.AgentConfig
	for _, presetAgent := range adaptedPreset.Agents {
		agents = append(agents, presetAgent)
	}

	return agents, nil
}

// SaveConfiguration saves the wizard configuration
func (w *Wizard) SaveConfiguration(ctx context.Context, providers []ConfiguredProvider, agents []config.AgentConfig) error {
	// Load existing configuration if it exists
	var guildConfig *config.GuildConfig
	existingConfig, err := config.LoadGuildConfig(w.config.ProjectPath)
	if err != nil {
		// Create new configuration
		guildConfig = &config.GuildConfig{
			Name:        "My Guild",
			Description: "A team of specialized AI agents",
			Version:     "1.0.0",
			Manager: config.ManagerConfig{
				Default: "manager",
			},
			Storage: config.StorageConfig{
				Backend: "sqlite",
				SQLite: config.SQLiteConfig{
					Path: ".campaign/memory.db",
				},
			},
			Providers: config.ProvidersConfig{},
			Agents:    agents,
		}
	} else {
		// Update existing configuration
		guildConfig = existingConfig
		if w.config.Force {
			// Replace agents if force mode
			guildConfig.Agents = agents
		} else {
			// Merge agents
			guildConfig.Agents = append(guildConfig.Agents, agents...)
		}
	}

	// Update manager default if needed
	if guildConfig.Manager.Default != "" {
		// Check if the default manager exists in the agents list
		managerFound := false
		for _, agent := range guildConfig.Agents {
			if agent.ID == guildConfig.Manager.Default {
				managerFound = true
				break
			}
		}

		// If not found, try to find a manager agent
		if !managerFound {
			for _, agent := range guildConfig.Agents {
				if agent.Type == "manager" {
					guildConfig.Manager.Default = agent.ID
					managerFound = true
					break
				}
			}
		}

		// If still not found, clear the default
		if !managerFound {
			guildConfig.Manager.Default = ""
		}
	}

	// Update provider configurations using the registry
	for _, provider := range providers {
		settings := config.ProviderSettings{
			BaseURL:  provider.Settings["base_url"],
			Settings: provider.Settings,
		}

		// Use registry to apply settings dynamically
		if err := w.registry.ApplyProviderSettings(&guildConfig.Providers, provider.Name, settings); err != nil {
			// Log warning but continue - provider might be new/custom
			// The TUI should have already validated providers
			continue
		}
	}

	// Save the configuration
	if err := config.SaveGuildConfig(w.config.ProjectPath, guildConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save guild config").
			WithComponent("setup").
			WithOperation("saveConfiguration")
	}

	return nil
}

// IsProjectSetup checks if the project has been set up with providers
func IsProjectSetup(ctx context.Context, projectPath string) (bool, error) {
	// Check if guild config exists
	_, err := config.LoadGuildConfig(projectPath)
	if err != nil {
		// If config doesn't exist, project is not setup
		return false, nil
	}

	// TODO: Could check if providers are configured
	return true, nil
}

// ProviderStatus contains information about a configured provider
type ProviderStatus struct {
	Name      string
	Available bool
	Models    int
}

// AgentStatus contains information about a configured agent
type AgentStatus struct {
	Name     string
	Type     string
	Provider string
	Model    string
}

// SetupStatus contains information about the current setup state
type SetupStatus struct {
	IsConfigured bool
	Providers    []ProviderStatus
	Agents       []AgentStatus
	AgentCount   int
}

// GetSetupStatus returns the current setup status
func GetSetupStatus(ctx context.Context, projectPath string) (*SetupStatus, error) {
	status := &SetupStatus{
		IsConfigured: false,
		Providers:    []ProviderStatus{},
		Agents:       []AgentStatus{},
		AgentCount:   0,
	}

	// Try to load config
	guildConfig, err := config.LoadGuildConfig(projectPath)
	if err != nil {
		// Not configured
		return status, nil
	}

	status.IsConfigured = true

	// Use registry to check providers dynamically
	registry := NewProviderRegistry()
	status.Providers = registry.GetConfiguredProviders(&guildConfig.Providers)

	// Get agent details
	for _, agent := range guildConfig.Agents {
		status.Agents = append(status.Agents, AgentStatus{
			Name:     agent.Name,
			Type:     agent.Type,
			Provider: agent.Provider,
			Model:    agent.Model,
		})
	}

	// Count agents
	status.AgentCount = len(guildConfig.Agents)

	return status, nil
}
