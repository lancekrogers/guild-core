// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"strings"

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

	return &Wizard{
		config:         config,
		detectors:      detectors,
		providerConfig: providerConfig,
		modelConfig:    modelConfig,
		agentConfig:    agentConfig,
	}, nil
}

// Run executes the setup wizard
func (w *Wizard) Run(ctx context.Context) error {
	// Step 1: Detect available providers
	if !w.config.QuickMode {
		fmt.Println("🔍 Detecting available providers...")
	}

	detection, err := w.detectors.DetectProviders(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect providers").
			WithComponent("setup").
			WithOperation("Run")
	}

	if !w.config.QuickMode {
		w.displayDetectionResults(detection)
	}

	// Step 2: Select providers to configure
	selectedProviders, err := w.selectProviders(ctx, detection)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to select providers").
			WithComponent("setup").
			WithOperation("Run")
	}

	if len(selectedProviders) == 0 {
		return gerror.New(gerror.ErrCodeValidation, "no providers selected", nil).
			WithComponent("setup").
			WithOperation("Run")
	}

	// Step 3: Configure each selected provider
	configuredProviders := make([]ConfiguredProvider, 0, len(selectedProviders))
	for _, provider := range selectedProviders {
		configured, err := w.configureProvider(ctx, provider)
		if err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to configure provider %s", provider.Name).
				WithComponent("setup").
				WithOperation("Run")
		}
		if configured != nil {
			configuredProviders = append(configuredProviders, *configured)
		}
	}

	// Step 4: Create agent configurations
	agents, err := w.createAgents(ctx, configuredProviders)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agents").
			WithComponent("setup").
			WithOperation("Run")
	}

	// Step 5: Save configuration
	if err := w.saveConfiguration(ctx, configuredProviders, agents); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to save configuration").
			WithComponent("setup").
			WithOperation("Run")
	}

	// Step 6: Display summary
	w.displaySummary(configuredProviders, agents)

	return nil
}

// displayDetectionResults shows what providers were detected
func (w *Wizard) displayDetectionResults(detection *DetectionResult) {
	if len(detection.Available) == 0 {
		fmt.Println("❌ No providers detected")
		return
	}

	fmt.Printf("✅ Detected %d provider(s):\n", len(detection.Available))
	for _, provider := range detection.Available {
		fmt.Printf("  • %s", provider.Name)
		if provider.HasCredentials {
			fmt.Print(" (credentials available)")
		}
		if provider.IsLocal {
			fmt.Print(" (local)")
		}
		fmt.Println()
	}
	fmt.Println()
}

// selectProviders handles provider selection
func (w *Wizard) selectProviders(ctx context.Context, detection *DetectionResult) ([]DetectedProvider, error) {
	// If specific provider requested, use only that
	if w.config.ProviderOnly != "" {
		for _, provider := range detection.Available {
			if strings.EqualFold(provider.Name, w.config.ProviderOnly) {
				return []DetectedProvider{provider}, nil
			}
		}
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "requested provider %s not found", w.config.ProviderOnly).
			WithComponent("setup").
			WithOperation("selectProviders")
	}

	// Quick mode: use all available providers
	if w.config.QuickMode {
		return detection.Available, nil
	}

	// Interactive mode: let user select
	fmt.Println("🤖 Select providers to configure:")
	fmt.Println("   (Enter numbers separated by spaces, or 'all' for all providers)")
	fmt.Println()

	for i, provider := range detection.Available {
		fmt.Printf("  %d) %s", i+1, provider.Name)
		if provider.HasCredentials {
			fmt.Print(" ✅")
		}
		if provider.IsLocal {
			fmt.Print(" 🏠")
		}
		fmt.Println()
	}

	fmt.Print("\nSelection: ")
	var input string
	fmt.Scanln(&input)

	if strings.TrimSpace(input) == "all" {
		return detection.Available, nil
	}

	// Parse selection (simplified for now - in real implementation would be more robust)
	if input == "1" && len(detection.Available) > 0 {
		return []DetectedProvider{detection.Available[0]}, nil
	}

	// Default to first available provider
	return []DetectedProvider{detection.Available[0]}, nil
}

// configureProvider configures a specific provider
func (w *Wizard) configureProvider(ctx context.Context, provider DetectedProvider) (*ConfiguredProvider, error) {
	if !w.config.QuickMode {
		fmt.Printf("\n⚙️  Configuring %s...\n", provider.Name)
	}

	// Validate provider configuration
	validated, err := w.providerConfig.ValidateProvider(ctx, provider)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeValidation, "failed to validate provider %s", provider.Name).
			WithComponent("setup").
			WithOperation("configureProvider")
	}

	if !validated.IsValid {
		if !w.config.QuickMode {
			fmt.Printf("❌ %s validation failed: %s\n", provider.Name, validated.Error)
		}
		return nil, nil
	}

	// Get available models for this provider
	models, err := w.modelConfig.GetModelsForProvider(ctx, provider.Name)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to get models for provider %s", provider.Name).
			WithComponent("setup").
			WithOperation("configureProvider")
	}

	// Select model(s) for this provider
	selectedModels, err := w.selectModels(ctx, provider, models)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to select models for provider %s", provider.Name).
			WithComponent("setup").
			WithOperation("configureProvider")
	}

	if !w.config.QuickMode {
		fmt.Printf("✅ %s configured with %d model(s)\n", provider.Name, len(selectedModels))
	}

	return &ConfiguredProvider{
		Name:     provider.Name,
		Type:     provider.Type,
		Models:   selectedModels,
		Settings: validated.Settings,
	}, nil
}

// selectModels handles model selection for a provider
func (w *Wizard) selectModels(ctx context.Context, provider DetectedProvider, models []ModelInfo) ([]ModelInfo, error) {
	if len(models) == 0 {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no models available for provider %s", provider.Name).
			WithComponent("setup").
			WithOperation("selectModels")
	}

	// Quick mode: use recommended models
	if w.config.QuickMode {
		var recommended []ModelInfo
		for _, model := range models {
			if model.Recommended {
				recommended = append(recommended, model)
			}
		}
		if len(recommended) > 0 {
			return recommended, nil
		}
		// Fallback to first model
		return []ModelInfo{models[0]}, nil
	}

	// Interactive mode: show models with costs
	fmt.Printf("\n💰 Available models for %s:\n", provider.Name)
	for i, model := range models {
		fmt.Printf("  %d) %s", i+1, model.Name)
		if model.CostPerInputToken > 0 {
			fmt.Printf(" ($%.4f/1K input, $%.4f/1K output)", 
				model.CostPerInputToken*1000, model.CostPerOutputToken*1000)
		}
		if model.Recommended {
			fmt.Print(" ⭐ Recommended")
		}
		fmt.Println()
	}

	// For now, just return the first model (in real implementation, parse user input)
	return []ModelInfo{models[0]}, nil
}

// createAgents creates default agent configurations
func (w *Wizard) createAgents(ctx context.Context, providers []ConfiguredProvider) ([]config.AgentConfig, error) {
	if !w.config.QuickMode {
		fmt.Println("\n🤖 Creating default agent configurations...")
	}

	agents, err := w.agentConfig.CreateDefaultAgents(ctx, providers)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create default agents").
			WithComponent("setup").
			WithOperation("createAgents")
	}

	if !w.config.QuickMode {
		fmt.Printf("✅ Created %d agent(s)\n", len(agents))
	}

	return agents, nil
}

// saveConfiguration saves the final configuration
func (w *Wizard) saveConfiguration(ctx context.Context, providers []ConfiguredProvider, agents []config.AgentConfig) error {
	if !w.config.QuickMode {
		fmt.Println("\n💾 Saving configuration...")
	}

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
					Path: ".guild/memory.db",
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

	// Update provider configurations
	for _, provider := range providers {
		switch provider.Name {
		case "openai":
			guildConfig.Providers.OpenAI = config.ProviderSettings{
				BaseURL:  provider.Settings["base_url"],
				Settings: provider.Settings,
			}
		case "anthropic":
			guildConfig.Providers.Anthropic = config.ProviderSettings{
				BaseURL:  provider.Settings["base_url"],
				Settings: provider.Settings,
			}
		case "ollama":
			guildConfig.Providers.Ollama = config.ProviderSettings{
				BaseURL:  provider.Settings["base_url"],
				Settings: provider.Settings,
			}
		case "claude_code":
			guildConfig.Providers.ClaudeCode = config.ProviderSettings{
				BaseURL:  provider.Settings["base_url"],
				Settings: provider.Settings,
			}
		case "deepseek":
			guildConfig.Providers.DeepSeek = config.ProviderSettings{
				BaseURL:  provider.Settings["base_url"],
				Settings: provider.Settings,
			}
		}
	}

	// Save the configuration
	if err := config.SaveGuildConfig(w.config.ProjectPath, guildConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save guild config").
			WithComponent("setup").
			WithOperation("saveConfiguration")
	}

	if !w.config.QuickMode {
		fmt.Println("✅ Configuration saved")
	}

	return nil
}

// displaySummary shows the final setup summary
func (w *Wizard) displaySummary(providers []ConfiguredProvider, agents []config.AgentConfig) {
	if w.config.QuickMode {
		return
	}

	fmt.Println("\n📋 Setup Summary")
	fmt.Println("═══════════════")

	fmt.Printf("🔌 Providers: %d\n", len(providers))
	for _, provider := range providers {
		fmt.Printf("  • %s (%d models)\n", provider.Name, len(provider.Models))
	}

	fmt.Printf("\n🤖 Agents: %d\n", len(agents))
	for _, agent := range agents {
		fmt.Printf("  • %s (%s) - %s/%s\n", 
			agent.Name, agent.Type, agent.Provider, agent.Model)
	}
}

// IsProjectSetup checks if a project is already set up
func IsProjectSetup(ctx context.Context, projectPath string) (bool, error) {
	config, err := config.LoadGuildConfig(projectPath)
	if err != nil {
		return false, nil // Not setup if config doesn't exist
	}

	// Check if providers and agents are configured
	hasProviders := config.Providers.OpenAI.BaseURL != "" ||
		config.Providers.Anthropic.BaseURL != "" ||
		config.Providers.Ollama.BaseURL != "" ||
		config.Providers.ClaudeCode.BaseURL != "" ||
		len(config.Providers.OpenAI.Settings) > 0 ||
		len(config.Providers.Anthropic.Settings) > 0 ||
		len(config.Providers.Ollama.Settings) > 0 ||
		len(config.Providers.ClaudeCode.Settings) > 0

	hasAgents := len(config.Agents) > 0

	return hasProviders && hasAgents, nil
}

// GetSetupStatus returns the current setup status
func GetSetupStatus(ctx context.Context, projectPath string) (*SetupStatus, error) {
	guildConfig, err := config.LoadGuildConfig(projectPath)
	if err != nil {
		return &SetupStatus{
			IsConfigured: false,
			Providers:    []ProviderStatus{},
			Agents:       []AgentStatus{},
		}, nil
	}

	status := &SetupStatus{
		IsConfigured: true,
		Providers:    []ProviderStatus{},
		Agents:       []AgentStatus{},
	}

	// Check provider status
	providers := []struct {
		name     string
		settings config.ProviderSettings
	}{
		{"openai", guildConfig.Providers.OpenAI},
		{"anthropic", guildConfig.Providers.Anthropic},
		{"ollama", guildConfig.Providers.Ollama},
		{"claude_code", guildConfig.Providers.ClaudeCode},
		{"deepseek", guildConfig.Providers.DeepSeek},
	}

	for _, p := range providers {
		if len(p.settings.Settings) > 0 || p.settings.BaseURL != "" {
			status.Providers = append(status.Providers, ProviderStatus{
				Name:      p.name,
				Available: true,
				Models:    1, // Simplified
			})
		}
	}

	// Check agent status
	for _, agent := range guildConfig.Agents {
		status.Agents = append(status.Agents, AgentStatus{
			Name:     agent.Name,
			Type:     agent.Type,
			Provider: agent.Provider,
			Model:    agent.Model,
		})
	}

	return status, nil
}

// SetupStatus represents the current setup status
type SetupStatus struct {
	IsConfigured bool
	Providers    []ProviderStatus
	Agents       []AgentStatus
}

// ProviderStatus represents a provider's status
type ProviderStatus struct {
	Name      string
	Available bool
	Models    int
}

// AgentStatus represents an agent's status
type AgentStatus struct {
	Name     string
	Type     string
	Provider string
	Model    string
}

// ConfiguredProvider represents a configured provider
type ConfiguredProvider struct {
	Name     string
	Type     string
	Models   []ModelInfo
	Settings map[string]string
}