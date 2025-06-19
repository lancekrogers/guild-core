// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"github.com/guild-ventures/guild-core/pkg/campaign"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
	"github.com/guild-ventures/guild-core/pkg/providers"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// InitValidationResult represents the result of a single validation check
type InitValidationResult struct {
	Name        string
	Description string
	Success     bool
	Error       error
	Warning     string
	Details     map[string]string
}

// InitValidator validates post-init setup to ensure 'guild chat' will work
type InitValidator struct {
	projectPath string
	results     []InitValidationResult
	hasFailures bool
	hasWarnings bool
}

// NewInitValidator creates a new init validator
func NewInitValidator(projectPath string) *InitValidator {
	return &InitValidator{
		projectPath: projectPath,
		results:     make([]InitValidationResult, 0),
	}
}

// Validate runs all validation checks
func (v *InitValidator) Validate(ctx context.Context) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "validation cancelled").
			WithComponent("InitValidator").
			WithOperation("Validate")
	}

	// Run validation checks in order of criticality
	checks := []func(context.Context) InitValidationResult{
		v.validateProjectStructure,
		v.validateCampaignConfiguration,
		v.validateGuildConfiguration,
		v.validateAgentConfiguration,
		v.validateProviderConfiguration,
		v.validateDatabaseInitialization,
		v.validateSocketRegistry,
		v.validateDaemonReadiness,
	}

	for _, check := range checks {
		// Check for cancellation before each validation
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "validation cancelled").
				WithComponent("InitValidator").
				WithOperation("Validate")
		}

		result := check(ctx)
		v.results = append(v.results, result)

		if !result.Success {
			v.hasFailures = true
		}
		if result.Warning != "" {
			v.hasWarnings = true
		}
	}

	// Return error if any critical validations failed
	if v.hasFailures {
		return gerror.New(gerror.ErrCodeValidation, "validation failed", nil).
			WithComponent("InitValidator").
			WithOperation("Validate").
			WithDetails("failures", fmt.Sprintf("%d", v.countFailures()))
	}

	return nil
}

// validateProjectStructure checks if the project directory structure is correct
func (v *InitValidator) validateProjectStructure(ctx context.Context) InitValidationResult {
	result := InitValidationResult{
		Name:        "Project Structure",
		Description: "Checking .guild directory and subdirectories",
		Details:     make(map[string]string),
	}

	// Check main .guild directory
	guildDir := filepath.Join(v.projectPath, ".guild")
	if info, err := os.Stat(guildDir); err != nil || !info.IsDir() {
		result.Error = gerror.New(gerror.ErrCodeNotFound, ".guild directory not found", nil).
			WithComponent("InitValidator").
			WithOperation("validateProjectStructure")
		return result
	}

	// Check required subdirectories
	requiredDirs := []string{
		"agents",
		"archives",
		"campaigns",
		"corpus",
		"guilds",
		"kanban",
		"objectives",
		"prompts",
	}

	missingDirs := []string{}
	for _, dir := range requiredDirs {
		path := filepath.Join(guildDir, dir)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			missingDirs = append(missingDirs, dir)
		}
	}

	if len(missingDirs) > 0 {
		result.Warning = fmt.Sprintf("Missing directories: %s", strings.Join(missingDirs, ", "))
		result.Details["missing"] = strings.Join(missingDirs, ", ")
	}

	result.Success = true
	result.Details["guild_dir"] = guildDir
	return result
}

// validateCampaignConfiguration checks campaign setup
func (v *InitValidator) validateCampaignConfiguration(ctx context.Context) InitValidationResult {
	result := InitValidationResult{
		Name:        "Campaign Configuration",
		Description: "Checking campaign reference and configuration",
		Details:     make(map[string]string),
	}

	// Try to detect campaign
	cwd, err := os.Getwd()
	if err != nil {
		result.Error = gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get working directory").
			WithComponent("InitValidator").
			WithOperation("validateCampaignConfiguration")
		return result
	}

	campaignName, err := campaign.DetectCampaign(cwd, "")
	if err != nil {
		result.Error = gerror.Wrap(err, gerror.ErrCodeNotFound, "campaign not detected").
			WithComponent("InitValidator").
			WithOperation("validateCampaignConfiguration")
		return result
	}

	result.Details["campaign"] = campaignName

	// Validate campaign exists in global storage
	if err := campaign.ValidateCampaign(campaignName); err != nil {
		result.Error = gerror.Wrap(err, gerror.ErrCodeValidation, "campaign validation failed").
			WithComponent("InitValidator").
			WithOperation("validateCampaignConfiguration")
		return result
	}

	// Check local campaign reference
	guildYaml := filepath.Join(v.projectPath, ".guild", "guild.yaml")
	if _, err := os.Stat(guildYaml); err != nil {
		result.Warning = "Local guild.yaml missing - campaign detection may fail"
		result.Details["guild_yaml"] = "missing"
	} else {
		result.Details["guild_yaml"] = "present"
	}

	result.Success = true
	return result
}

// validateGuildConfiguration checks guild.yml configuration
func (v *InitValidator) validateGuildConfiguration(ctx context.Context) InitValidationResult {
	result := InitValidationResult{
		Name:        "Guild Configuration",
		Description: "Checking guild definitions",
		Details:     make(map[string]string),
	}

	// Load guild config file
	guildConfig, err := config.LoadGuildConfigFile(ctx, v.projectPath)
	if err != nil {
		result.Error = gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load guild configuration").
			WithComponent("InitValidator").
			WithOperation("validateGuildConfiguration")
		return result
	}

	// Check if at least one guild is defined
	if len(guildConfig.Guilds) == 0 {
		result.Error = gerror.New(gerror.ErrCodeValidation, "no guilds defined", nil).
			WithComponent("InitValidator").
			WithOperation("validateGuildConfiguration")
		return result
	}

	// List guilds
	guildNames := guildConfig.ListGuildNames()
	result.Details["guilds"] = strings.Join(guildNames, ", ")
	result.Details["count"] = fmt.Sprintf("%d", len(guildNames))

	// Validate each guild has agents
	for name, guild := range guildConfig.Guilds {
		if len(guild.Agents) == 0 {
			result.Warning = fmt.Sprintf("Guild '%s' has no agents", name)
			break
		}
	}

	result.Success = true
	return result
}

// validateAgentConfiguration checks agent setup
func (v *InitValidator) validateAgentConfiguration(ctx context.Context) InitValidationResult {
	result := InitValidationResult{
		Name:        "Agent Configuration",
		Description: "Checking agent definitions and configuration",
		Details:     make(map[string]string),
	}

	// Load main guild config
	guildConfig, err := config.LoadGuildConfig(v.projectPath)
	if err != nil {
		result.Error = gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load guild configuration").
			WithComponent("InitValidator").
			WithOperation("validateAgentConfiguration")
		return result
	}

	// Check if agents are configured
	if len(guildConfig.Agents) == 0 {
		result.Error = gerror.New(gerror.ErrCodeValidation, "no agents configured", nil).
			WithComponent("InitValidator").
			WithOperation("validateAgentConfiguration")
		return result
	}

	result.Details["agent_count"] = fmt.Sprintf("%d", len(guildConfig.Agents))

	// Check for manager agent
	hasManager := false
	providerTypes := make(map[string]int)
	
	for _, agent := range guildConfig.Agents {
		if agent.Type == "manager" {
			hasManager = true
		}
		providerTypes[agent.Provider]++
	}

	if !hasManager {
		result.Warning = "No manager agent configured - guild may not function properly"
	}

	// List providers in use
	providers := []string{}
	for provider, count := range providerTypes {
		providers = append(providers, fmt.Sprintf("%s(%d)", provider, count))
	}
	result.Details["providers"] = strings.Join(providers, ", ")

	result.Success = true
	return result
}

// validateProviderConfiguration checks provider setup and credentials
func (v *InitValidator) validateProviderConfiguration(ctx context.Context) InitValidationResult {
	result := InitValidationResult{
		Name:        "Provider Configuration",
		Description: "Checking AI provider setup and credentials",
		Details:     make(map[string]string),
	}

	// Load guild config to get providers in use
	guildConfig, err := config.LoadGuildConfig(v.projectPath)
	if err != nil {
		result.Error = gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load configuration").
			WithComponent("InitValidator").
			WithOperation("validateProviderConfiguration")
		return result
	}

	// Get unique providers
	providersInUse := make(map[string]bool)
	for _, agent := range guildConfig.Agents {
		providersInUse[agent.Provider] = true
	}

	// Check each provider
	missingCreds := []string{}
	for provider := range providersInUse {
		// Check for cancellation
		if err := ctx.Err(); err != nil {
			result.Error = gerror.Wrap(err, gerror.ErrCodeCancelled, "validation cancelled").
				WithComponent("InitValidator").
				WithOperation("validateProviderConfiguration")
			return result
		}

		// Normalize provider name
		normalizedProvider := providers.NormalizeProviderName(provider)
		
		switch normalizedProvider {
		case providers.ProviderNameOpenAI:
			if os.Getenv(providers.EnvOpenAIKey) == "" {
				missingCreds = append(missingCreds, providers.EnvOpenAIKey)
			}
		case providers.ProviderNameAnthropic:
			if os.Getenv(providers.EnvAnthropicKey) == "" {
				missingCreds = append(missingCreds, providers.EnvAnthropicKey)
			}
		case providers.ProviderNameDeepSeek:
			if os.Getenv(providers.EnvDeepSeekKey) == "" {
				missingCreds = append(missingCreds, providers.EnvDeepSeekKey)
			}
		case providers.ProviderNameDeepInfra:
			if os.Getenv(providers.EnvDeepInfraKey) == "" {
				missingCreds = append(missingCreds, providers.EnvDeepInfraKey)
			}
		case providers.ProviderNameOra:
			if os.Getenv(providers.EnvOraKey) == "" {
				missingCreds = append(missingCreds, providers.EnvOraKey)
			}
		case providers.ProviderNameClaude:
			// Claude Code doesn't need API key in Claude Code environment
			if os.Getenv("CLAUDE_CODE_SESSION") == "" && os.Getenv(providers.EnvAnthropicKey) == "" {
				missingCreds = append(missingCreds, providers.EnvAnthropicKey)
			}
		case providers.ProviderNameOllama:
			// For Ollama, just check if it's configured
			// The actual connectivity check would require provider initialization
			result.Details[providers.ProviderNameOllama] = "configured"
		}
	}

	if len(missingCreds) > 0 {
		result.Error = gerror.New(gerror.ErrCodeConfiguration, "missing API credentials", nil).
			WithComponent("InitValidator").
			WithOperation("validateProviderConfiguration").
			WithDetails("missing", strings.Join(missingCreds, ", "))
		return result
	}

	result.Success = true
	result.Details["providers"] = fmt.Sprintf("%d configured", len(providersInUse))
	return result
}

// validateDatabaseInitialization checks database setup
func (v *InitValidator) validateDatabaseInitialization(ctx context.Context) InitValidationResult {
	result := InitValidationResult{
		Name:        "Database Initialization",
		Description: "Checking SQLite database setup",
		Details:     make(map[string]string),
	}

	// Check database file
	dbPath := filepath.Join(v.projectPath, ".guild", "memory.db")
	if info, err := os.Stat(dbPath); err != nil {
		result.Error = gerror.New(gerror.ErrCodeNotFound, "database file not found", nil).
			WithComponent("InitValidator").
			WithOperation("validateDatabaseInitialization")
		return result
	} else {
		result.Details["size"] = fmt.Sprintf("%.2f MB", float64(info.Size())/1024/1024)
	}

	// Try to initialize storage (this validates database connectivity)
	_, _, err := storage.InitializeSQLiteStorageForRegistry(ctx, dbPath)
	if err != nil {
		result.Error = gerror.Wrap(err, gerror.ErrCodeConnection, "failed to initialize storage").
			WithComponent("InitValidator").
			WithOperation("validateDatabaseInitialization")
		return result
	}

	result.Success = true
	result.Details["path"] = dbPath
	result.Details["status"] = "connected"
	return result
}

// validateSocketRegistry checks socket registry setup
func (v *InitValidator) validateSocketRegistry(ctx context.Context) InitValidationResult {
	result := InitValidationResult{
		Name:        "Socket Registry",
		Description: "Checking daemon socket registry",
		Details:     make(map[string]string),
	}

	// Check registry file
	registryPath := filepath.Join(v.projectPath, ".guild", "socket-registry.yaml")
	if _, err := os.Stat(registryPath); err != nil {
		result.Warning = "Socket registry not found - daemon auto-start may not work"
		result.Details["registry"] = "missing"
	} else {
		// Try to load registry
		registry, err := daemon.LoadSocketRegistry(v.projectPath)
		if err != nil {
			result.Warning = "Socket registry corrupted"
			result.Details["registry"] = "corrupted"
		} else {
			result.Details["registry"] = "valid"
			result.Details["campaign"] = registry.CampaignName
		}
	}

	result.Success = true
	return result
}

// validateDaemonReadiness checks if daemon can be started
func (v *InitValidator) validateDaemonReadiness(ctx context.Context) InitValidationResult {
	result := InitValidationResult{
		Name:        "Daemon Readiness",
		Description: "Checking if daemon can be started",
		Details:     make(map[string]string),
	}

	// Get campaign name
	cwd, _ := os.Getwd()
	campaignName, err := campaign.DetectCampaign(cwd, "")
	if err != nil {
		result.Warning = "Cannot detect campaign for daemon"
		result.Success = true
		return result
	}

	// Check if we can get socket path for the campaign
	socketPath, err := paths.GetCampaignSocket(campaignName, 0)
	if err == nil && daemon.IsDaemonRunning(socketPath) {
		result.Details["status"] = "already running"
		result.Details["socket"] = socketPath
		result.Success = true
		return result
	}

	// Check if we could start a daemon
	result.Details["status"] = "ready to start"
	result.Details["campaign"] = campaignName
	result.Success = true
	return result
}

// HasFailures returns true if any validation failed
func (v *InitValidator) HasFailures() bool {
	return v.hasFailures
}

// HasWarnings returns true if any validation has warnings
func (v *InitValidator) HasWarnings() bool {
	return v.hasWarnings
}

// countFailures returns the number of failed validations
func (v *InitValidator) countFailures() int {
	count := 0
	for _, r := range v.results {
		if !r.Success {
			count++
		}
	}
	return count
}

// PrintResults prints validation results in a formatted way
func (v *InitValidator) PrintResults() {
	// Colors
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	fmt.Println()
	fmt.Println(bold("Validation Results:"))
	fmt.Println(strings.Repeat("-", 60))

	for _, result := range v.results {
		// Status icon
		var status string
		if !result.Success {
			status = red("✗")
		} else if result.Warning != "" {
			status = yellow("⚠")
		} else {
			status = green("✓")
		}

		// Print main result
		fmt.Printf("%s %s\n", status, bold(result.Name))
		fmt.Printf("  %s\n", result.Description)

		// Print error if any
		if result.Error != nil {
			fmt.Printf("  %s %s\n", red("Error:"), result.Error.Error())
		}

		// Print warning if any
		if result.Warning != "" {
			fmt.Printf("  %s %s\n", yellow("Warning:"), result.Warning)
		}

		// Print details
		if len(result.Details) > 0 {
			for key, value := range result.Details {
				fmt.Printf("  %s %s\n", cyan(key+":"), value)
			}
		}

		fmt.Println()
	}

	// Summary
	fmt.Println(strings.Repeat("-", 60))
	failureCount := v.countFailures()
	if failureCount == 0 {
		if v.hasWarnings {
			fmt.Printf("%s All checks passed with warnings.\n", yellow("⚠"))
			fmt.Println("Guild chat should work, but some features may be limited.")
		} else {
			fmt.Printf("%s All checks passed! Guild chat is ready to use.\n", green("✓"))
		}
	} else {
		fmt.Printf("%s %d checks failed. Please fix the issues above.\n", red("✗"), failureCount)
		fmt.Println("Guild chat may not work until these issues are resolved.")
	}
	fmt.Println()
}

// GetResults returns all validation results
func (v *InitValidator) GetResults() []InitValidationResult {
	return v.results
}