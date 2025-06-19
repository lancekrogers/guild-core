// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
	reader         *bufio.Reader // For interactive input
	inputTimeout   time.Duration // Timeout for user input
	ui             *WizardUI     // UI helper for enhanced display
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
		reader:         bufio.NewReader(os.Stdin),
		inputTimeout:   30 * time.Second, // Default 30 second timeout for user input
	}
	
	// Create UI helper
	wizard.ui = NewWizardUI(wizard)
	
	return wizard, nil
}

// Run executes the setup wizard
func (w *Wizard) Run(ctx context.Context) error {
	// Check context early
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before wizard execution").
			WithComponent("SetupWizard").
			WithOperation("Run")
	}

	// Show welcome screen
	w.ui.ShowWelcomeScreen()

	// Step 1: Detect available providers
	w.ui.ShowSection("Provider Detection")
	if !w.config.QuickMode {
		fmt.Println("🔍 Detecting available providers...")
	}

	detection, err := w.detectors.Providers(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect providers").
			WithComponent("SetupWizard").
			WithOperation("Run")
	}

	if !w.config.QuickMode {
		w.displayDetectionResults(detection)
	}

	// Check context between steps
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled after provider detection").
			WithComponent("SetupWizard").
			WithOperation("Run")
	}

	// Step 2: Select providers to configure
	w.ui.ShowSection("Provider Selection")
	selectedProviders, err := w.selectProviders(ctx, detection)
	if err != nil {
		w.ui.ShowError(err, "Provider selection failed")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to select providers").
			WithComponent("SetupWizard").
			WithOperation("Run")
	}

	if len(selectedProviders) == 0 {
		w.ui.ShowError(gerror.New(gerror.ErrCodeValidation, "no providers selected", nil), "Setup cannot continue")
		return gerror.New(gerror.ErrCodeValidation, "no providers selected", nil).
			WithComponent("SetupWizard").
			WithOperation("Run")
	}

	// Step 3: Configure each selected provider
	w.ui.ShowSection("Provider Configuration")
	configuredProviders := make([]ConfiguredProvider, 0, len(selectedProviders))
	
	// Show progress bar for provider configuration
	progress := w.ui.ShowProgressBar(ctx, "Configuring providers", len(selectedProviders))
	configuredCount := 0
	
	for _, provider := range selectedProviders {
		// Check context in loop
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider configuration").
				WithComponent("SetupWizard").
				WithOperation("Run").
				WithDetails("provider", provider.Name)
		}
		configured, err := w.configureProvider(ctx, provider)
		if err != nil {
			return gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to configure provider %s", provider.Name).
				WithComponent("SetupWizard").
				WithOperation("Run")
		}
		if configured != nil {
			configuredProviders = append(configuredProviders, *configured)
		}
		
		// Update progress
		configuredCount++
		select {
		case progress <- configuredCount:
		default:
		}
	}
	close(progress)

	// Check context before agent creation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before agent creation").
			WithComponent("SetupWizard").
			WithOperation("Run")
	}

	// Step 4: Create agent configurations
	w.ui.ShowSection("Agent Creation")
	agents, err := w.createAgents(ctx, configuredProviders)
	if err != nil {
		w.ui.ShowError(err, "Agent creation failed")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create agents").
			WithComponent("SetupWizard").
			WithOperation("Run")
	}

	// Step 5: Save configuration
	w.ui.ShowSection("Saving Configuration")
	if err := w.saveConfiguration(ctx, configuredProviders, agents); err != nil {
		w.ui.ShowError(err, "Configuration save failed")
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to save configuration").
			WithComponent("SetupWizard").
			WithOperation("Run")
	}

	// Step 6: Display summary
	w.ui.ShowCompletionSummary(configuredProviders, agents)

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
	// Check context at start
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider selection").
			WithComponent("SetupWizard").
			WithOperation("selectProviders")
	}

	// If specific provider requested, use only that
	if w.config.ProviderOnly != "" {
		for _, provider := range detection.Available {
			if strings.EqualFold(provider.Name, w.config.ProviderOnly) {
				return []DetectedProvider{provider}, nil
			}
		}
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "requested provider %s not found", w.config.ProviderOnly).
			WithComponent("SetupWizard").
			WithOperation("selectProviders")
	}

	// Quick mode: use all available providers
	if w.config.QuickMode {
		return detection.Available, nil
	}

	// Interactive mode: let user select
	w.ui.ShowProviderSelectionHelp()
	fmt.Println("🤖 Select providers to configure:")
	fmt.Println()

	for i, provider := range detection.Available {
		fmt.Printf("  %d) %s", i+1, provider.Name)
		if provider.HasCredentials {
			fmt.Print(" ✅ (credentials detected)")
		}
		if provider.IsLocal {
			fmt.Print(" 🏠 (local)")
		}
		fmt.Println()
	}

	fmt.Print("\nSelection (default: all): ")
	
	// Read user input with timeout
	input, err := w.readLineWithTimeout(ctx, w.inputTimeout)
	if err != nil {
		// If timeout or cancellation, use all providers
		if gerror.Is(err, gerror.ErrCodeTimeout) || gerror.Is(err, gerror.ErrCodeCancelled) {
			fmt.Println("\n⌚ Using all detected providers (timeout/cancelled)")
			return detection.Available, nil
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read user input").
			WithComponent("SetupWizard").
			WithOperation("selectProviders")
	}

	// Handle empty input (default to all)
	input = strings.TrimSpace(input)
	if input == "" || strings.EqualFold(input, "all") {
		return detection.Available, nil
	}

	// Parse selection
	selected := []DetectedProvider{}
	parts := strings.Fields(input)
	
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			fmt.Printf("⚠️  Invalid selection '%s', skipping\n", part)
			continue
		}
		
		if num < 1 || num > len(detection.Available) {
			fmt.Printf("⚠️  Selection %d out of range, skipping\n", num)
			continue
		}
		
		// Add to selected (avoiding duplicates)
		provider := detection.Available[num-1]
		alreadySelected := false
		for _, s := range selected {
			if s.Name == provider.Name {
				alreadySelected = true
				break
			}
		}
		if !alreadySelected {
			selected = append(selected, provider)
		}
	}

	// If no valid selections, default to all
	if len(selected) == 0 {
		fmt.Println("⚠️  No valid selections, using all detected providers")
		return detection.Available, nil
	}

	return selected, nil
}

// configureProvider configures a specific provider
func (w *Wizard) configureProvider(ctx context.Context, provider DetectedProvider) (*ConfiguredProvider, error) {
	// Check context at start
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during provider configuration").
			WithComponent("SetupWizard").
			WithOperation("configureProvider").
			WithDetails("provider", provider.Name)
	}

	if !w.config.QuickMode {
		fmt.Printf("\n⚙️  Configuring %s...\n", provider.Name)
	}

	// Validate provider configuration
	validated, err := w.providerConfig.ValidateProvider(ctx, provider)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeValidation, "failed to validate provider %s", provider.Name).
			WithComponent("SetupWizard").
			WithOperation("configureProvider")
	}

	if !validated.IsValid {
		w.ui.ShowWarning(fmt.Sprintf("%s validation failed: %s", provider.Name, validated.Error))
		return nil, nil
	}

	// Check context before model retrieval
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before model retrieval").
			WithComponent("SetupWizard").
			WithOperation("configureProvider").
			WithDetails("provider", provider.Name)
	}

	// Get available models for this provider
	models, err := w.modelConfig.GetModelsForProvider(ctx, provider.Name)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to get models for provider %s", provider.Name).
			WithComponent("SetupWizard").
			WithOperation("configureProvider")
	}

	// Select model(s) for this provider
	selectedModels, err := w.selectModels(ctx, provider, models)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to select models for provider %s", provider.Name).
			WithComponent("SetupWizard").
			WithOperation("configureProvider")
	}

	w.ui.ShowSuccess(fmt.Sprintf("%s configured with %d model(s)", provider.Name, len(selectedModels)))

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
			WithComponent("SetupWizard").
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
	w.ui.ShowModelSelectionHelp(provider.Name)
	fmt.Printf("💰 Available models for %s:\n", provider.Name)
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
		fmt.Println("\n🤖 Creating agent configurations...")
	}

	// Detect project context
	projectContext, err := w.detectors.DetectProjectContext(ctx)
	if err != nil {
		// If detection fails, continue with default context
		projectContext = &ProjectContext{
			ProjectType: "general",
			Language:    "go",
		}
	}

	// Quick mode: use intelligent preset selection
	if w.config.QuickMode {
		return w.createAgentsFromPresets(ctx, providers, projectContext)
	}

	// Interactive mode: offer preset choices
	return w.createAgentsInteractive(ctx, providers, projectContext)
}

// createAgentsFromPresets creates agents using intelligent preset selection for quick mode
func (w *Wizard) createAgentsFromPresets(ctx context.Context, providers []ConfiguredProvider, projectContext *ProjectContext) ([]config.AgentConfig, error) {
	// Get recommendations
	recommendations, err := w.agentPresets.RecommendPresets(ctx, providers, projectContext)
	if err != nil {
		// Fallback to legacy method if presets fail
		return w.agentConfig.CreateDefaultAgents(ctx, providers)
	}

	if len(recommendations) == 0 {
		// Fallback to legacy method
		return w.agentConfig.CreateDefaultAgents(ctx, providers)
	}

	// Use the highest-confidence recommendation
	bestRec := recommendations[0]
	if !bestRec.Compatible || bestRec.Confidence < 0.3 {
		// Fallback to legacy method if no good recommendations
		return w.agentConfig.CreateDefaultAgents(ctx, providers)
	}

	// Adapt the preset for available providers
	adapted, err := w.agentPresets.AdaptPresetForProviders(ctx, bestRec.Collection, providers)
	if err != nil {
		// Fallback to legacy method
		return w.agentConfig.CreateDefaultAgents(ctx, providers)
	}

	if !w.config.QuickMode {
		fmt.Printf("✅ Created %d agent(s) using preset '%s'\n", len(adapted.Agents), adapted.Name)
	}

	return adapted.Agents, nil
}

// createAgentsInteractive creates agents with user interaction for preset selection
func (w *Wizard) createAgentsInteractive(ctx context.Context, providers []ConfiguredProvider, projectContext *ProjectContext) ([]config.AgentConfig, error) {
	// Get recommendations
	recommendations, err := w.agentPresets.RecommendPresets(ctx, providers, projectContext)
	if err != nil {
		fmt.Printf("⚠️  Failed to get preset recommendations: %v\n", err)
		fmt.Println("Falling back to default agent creation...")
		return w.agentConfig.CreateDefaultAgents(ctx, providers)
	}

	if len(recommendations) == 0 {
		fmt.Println("No suitable presets found, using default agent creation...")
		return w.agentConfig.CreateDefaultAgents(ctx, providers)
	}

	// Display recommendations
	w.ui.ShowPresetSelectionHelp()
	fmt.Println("📋 Recommended Agent Presets:")
	fmt.Println()

	validRecs := make([]*PresetRecommendation, 0)
	for i, rec := range recommendations {
		if rec.Compatible && rec.Confidence > 0.2 {
			validRecs = append(validRecs, rec)
			confidence := int(rec.Confidence * 100)
			fmt.Printf("  %d) %s (%d%% match)\n", i+1, rec.Collection.Name, confidence)
			fmt.Printf("     %s\n", rec.Collection.Description)
			if len(rec.Reasoning) > 0 {
				fmt.Printf("     💡 %s\n", rec.Reasoning[0])
			}
			fmt.Printf("     👥 %d agents\n", len(rec.Collection.Agents))
			fmt.Println()
			
			if len(validRecs) >= 5 {
				break // Limit to top 5 recommendations
			}
		}
	}

	// Add fallback options
	fmt.Printf("  %d) Use default agent creation (legacy)\n", len(validRecs)+1)
	fmt.Printf("  %d) Manual preset selection\n", len(validRecs)+2)

	// Get user choice
	fmt.Print("\nSelect preset (1-" + fmt.Sprintf("%d", len(validRecs)+2) + "): ")
	// For demo purposes, automatically select the best recommendation
	selectedIndex := 0
	if len(validRecs) > 0 {
		selected := validRecs[selectedIndex]
		
		// Adapt the preset for available providers
		adapted, err := w.agentPresets.AdaptPresetForProviders(ctx, selected.Collection, providers)
		if err != nil {
			fmt.Printf("⚠️  Failed to adapt preset: %v\n", err)
			fmt.Println("Falling back to default agent creation...")
			return w.agentConfig.CreateDefaultAgents(ctx, providers)
		}

		fmt.Printf("✅ Created %d agent(s) using preset '%s'\n", len(adapted.Agents), adapted.Name)
		
		// Show brief agent summary
		fmt.Println("\n👥 Agent Team:")
		for _, agent := range adapted.Agents {
			fmt.Printf("  • %s (%s) - %s\n", agent.Name, agent.Type, agent.Description)
		}

		return adapted.Agents, nil
	}

	// Fallback to default
	return w.agentConfig.CreateDefaultAgents(ctx, providers)
}

// GetDemoQuickSetup creates a demo-optimized configuration for the "30-second demo" requirement
func (w *Wizard) GetDemoQuickSetup(ctx context.Context, providers []ConfiguredProvider) ([]config.AgentConfig, error) {
	// Get the optimal demo preset
	demoPreset, err := w.agentPresets.GetDemoPreset(ctx, providers)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get demo preset").
			WithComponent("setup").
			WithOperation("GetDemoQuickSetup")
	}

	if !w.config.QuickMode {
		fmt.Printf("🎭 Created demo team: %s\n", demoPreset.Name)
		fmt.Printf("   %s\n", demoPreset.Description)
		fmt.Printf("   👥 %d agents ready for demonstration\n", len(demoPreset.Agents))
	}

	return demoPreset.Agents, nil
}

// CreatePresetBasedSetup creates agents using a specific preset ID
func (w *Wizard) CreatePresetBasedSetup(ctx context.Context, providers []ConfiguredProvider, presetID string) ([]config.AgentConfig, error) {
	// Get the specified preset
	preset, err := w.agentPresets.GetPreset(ctx, presetID)
	if err != nil {
		return nil, gerror.Wrapf(err, gerror.ErrCodeNotFound, "preset '%s' not found", presetID).
			WithComponent("setup").
			WithOperation("CreatePresetBasedSetup")
	}

	// Adapt for available providers
	adapted, err := w.agentPresets.AdaptPresetForProviders(ctx, preset, providers)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to adapt preset for providers").
			WithComponent("setup").
			WithOperation("CreatePresetBasedSetup")
	}

	if !w.config.QuickMode {
		fmt.Printf("✅ Created %d agent(s) using preset '%s'\n", len(adapted.Agents), adapted.Name)
	}

	return adapted.Agents, nil
}

// saveConfiguration saves the final configuration
func (w *Wizard) saveConfiguration(ctx context.Context, providers []ConfiguredProvider, agents []config.AgentConfig) error {
	if !w.config.QuickMode {
		done := make(chan struct{})
		go w.displayProgressIndicator("Saving configuration", done)
		defer func() {
			close(done)
			time.Sleep(100 * time.Millisecond) // Let progress indicator finish
		}()
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

	w.ui.ShowSuccess("Configuration saved successfully")

	return nil
}

// displaySummary shows the final setup summary (deprecated - use UI component)
func (w *Wizard) displaySummary(providers []ConfiguredProvider, agents []config.AgentConfig) {
	w.ui.ShowCompletionSummary(providers, agents)
}

// readLineWithTimeout reads a line from stdin with a timeout
func (w *Wizard) readLineWithTimeout(ctx context.Context, timeout time.Duration) (string, error) {
	// Create a channel to receive the result
	type result struct {
		line string
		err  error
	}
	resultCh := make(chan result, 1)

	// Start goroutine to read input
	go func() {
		line, err := w.reader.ReadString('\n')
		if err != nil {
			resultCh <- result{err: err}
			return
		}
		// Trim the newline
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		resultCh <- result{line: line}
	}()

	// Create timeout context if not already limited
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Wait for result or timeout
	select {
	case <-timeoutCtx.Done():
		if ctx.Err() != nil {
			// Original context was cancelled
			return "", gerror.Wrap(ctx.Err(), gerror.ErrCodeCancelled, "context cancelled while reading input").
				WithComponent("SetupWizard").
				WithOperation("readLineWithTimeout")
		}
		// Timeout occurred
		return "", gerror.New(gerror.ErrCodeTimeout, "input timeout exceeded", nil).
			WithComponent("SetupWizard").
			WithOperation("readLineWithTimeout").
			WithDetails("timeout", timeout.String())
	case res := <-resultCh:
		if res.err != nil {
			return "", gerror.Wrap(res.err, gerror.ErrCodeInternal, "failed to read input").
				WithComponent("SetupWizard").
				WithOperation("readLineWithTimeout")
		}
		return res.line, nil
	}
}

// displayProgressIndicator shows progress for long-running operations
func (w *Wizard) displayProgressIndicator(message string, done <-chan struct{}) {
	if w.config.QuickMode {
		return
	}

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	for {
		select {
		case <-done:
			fmt.Printf("\r%s... Done! ✓\n", message)
			return
		default:
			fmt.Printf("\r%s %s...", spinner[i%len(spinner)], message)
			i++
			time.Sleep(100 * time.Millisecond)
		}
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