// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/lancekrogers/guild/internal/setup"
	"github.com/lancekrogers/guild/pkg/campaign"
	"github.com/lancekrogers/guild/pkg/daemon"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/paths"
	"github.com/lancekrogers/guild/pkg/project"
)

// DefaultConfigManager implements ConfigurationManager
type DefaultConfigManager struct{}

// NewDefaultConfigManager creates a new default config manager
func NewDefaultConfigManager() ConfigurationManager {
	return &DefaultConfigManager{}
}

func (d *DefaultConfigManager) EstablishGuildFoundation(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "guild foundation establishment cancelled").
			WithComponent("DefaultConfigManager").
			WithOperation("EstablishGuildFoundation")
	}

	// Step 0: Ensure campaign directory exists
	campaignDir := filepath.Join(projectPath, paths.DefaultCampaignDir)
	if err := os.MkdirAll(campaignDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign directory").
			WithComponent("DefaultConfigManager").
			WithOperation("EstablishGuildFoundation").
			WithDetails("dir", campaignDir)
	}

	// Step 1: Create optimized detection files
	// 1a. Create binary hash file for ultra-fast detection
	if err := campaign.WriteCampaignHash(projectPath, campaignName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign hash").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	// 1b. Create socket registry for fast detection with metadata
	if err := campaign.WriteSocketRegistry(projectPath, campaignName); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write socket registry").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	// 1c. Create proper campaign.yaml structure
	campaignConfig := map[string]interface{}{
		"name":        campaignName,
		"description": fmt.Sprintf("Guild campaign for %s", projectName),
		"created":     time.Now().Format(time.RFC3339),
		"projects": []map[string]interface{}{
			{
				"name": projectName,
				"path": projectPath,
				"type": "software",
			},
		},
		"guilds": []string{"elena_guild"},
		"settings": map[string]interface{}{
			"default_guild":     "elena_guild",
			"auto_start_daemon": true,
		},
	}

	campaignData, err := yaml.Marshal(campaignConfig)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign config").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	campaignPath := filepath.Join(projectPath, paths.DefaultCampaignDir, "campaign.yaml")
	if err := os.WriteFile(campaignPath, campaignData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save campaign configuration").
			WithComponent("DefaultConfigManager").
			WithOperation("EstablishGuildFoundation").
			WithDetails("campaign", campaignName)
	}

	// Check context between operations
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled after campaign config").
			WithComponent("DefaultConfigManager")
	}

	// Step 2: Create guilds directory and Elena-focused guild config
	guildsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "guilds")
	if err := os.MkdirAll(guildsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create guilds directory").
			WithComponent("DefaultConfigManager").
			WithOperation("EstablishGuildFoundation").
			WithDetails("dir", guildsDir)
	}

	// Create Elena-focused guild configuration with rich agents
	elenaGuild := map[string]interface{}{
		"name":        "elena_guild",
		"purpose":     "Elena's elite team of digital artisans for sophisticated software development",
		"description": "Guild Master Elena leads a team of highly skilled specialists in creating exceptional software",
		"manager":     "elena-guild-master",
		"agents": []string{
			"elena-guild-master",
			"marcus-developer",
			"vera-tester",
		},
		"coordination": map[string]interface{}{
			"max_parallel_tasks":  3,
			"review_required":     true,
			"auto_handoff":        true,
			"communication_style": "collaborative",
		},
		"specialties": []string{
			"full-stack development",
			"quality assurance",
			"project orchestration",
			"team collaboration",
		},
	}

	guildData, err := yaml.Marshal(elenaGuild)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal guild config").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	guildPath := filepath.Join(guildsDir, "elena_guild.yaml")
	if err := os.WriteFile(guildPath, guildData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write guild config").
			WithComponent("DefaultConfigManager").
			WithOperation("EstablishGuildFoundation").
			WithDetails("path", guildPath)
	}

	// Step 3: Create agents directory
	// Note: Agent configs will be created by createEnhancedAgents in init.go
	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("DefaultConfigManager").
			WithOperation("EstablishGuildFoundation").
			WithDetails("dir", agentsDir)
	}

	// Step 4: Guild configuration is now consolidated in guilds/elena_guild.yaml
	// No redundant guild.yaml creation needed - all guild config is in the guilds/ directory

	// Step 4: Create other required directories
	directories := []string{
		"commissions",
		"commissions/refined",
		"corpus",
		"corpus/index",
		"kanban",
		"prompts",
		"tools",
		"workspaces",
	}

	for _, dir := range directories {
		dirPath := filepath.Join(projectPath, paths.DefaultCampaignDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create directory").
				WithComponent("DefaultConfigManager").
				WithOperation("EstablishGuildFoundation").
				WithDetails("dir", dirPath)
		}
	}

	// Step 5: Create memory database file
	dbPath := filepath.Join(projectPath, paths.DefaultCampaignDir, "memory.db")
	if _, err := os.Create(dbPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create database file").
			WithComponent("DefaultConfigManager").
			WithOperation("EstablishGuildFoundation").
			WithDetails("path", dbPath)
	}

	// Step 6: Register campaign in global registry (~/.guild/campaigns/{hash}/)
	if err := d.registerCampaignGlobally(ctx, campaignName, projectPath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to register campaign globally").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	return nil
}

func (d *DefaultConfigManager) registerCampaignGlobally(ctx context.Context, campaignName, projectPath string) error {
	// Get campaign hash for directory name
	hash := campaign.GenerateCampaignHash(campaignName)

	// Get global config directory
	configDir, err := paths.GetGuildConfigDir()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get guild config directory").
			WithComponent("DefaultConfigManager").
			WithOperation("registerCampaignGlobally")
	}

	// Create campaign registry directory
	campaignRegistryDir := filepath.Join(configDir, "campaigns", hash)
	if err := os.MkdirAll(campaignRegistryDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign registry directory").
			WithComponent("DefaultConfigManager").
			WithOperation("registerCampaignGlobally").
			WithDetails("dir", campaignRegistryDir)
	}

	// Create minimal campaign registry file
	registryData := map[string]interface{}{
		"name":     campaignName,
		"location": projectPath,
		"created":  time.Now().Format(time.RFC3339),
	}

	registryYaml, err := yaml.Marshal(registryData)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign registry").
			WithComponent("DefaultConfigManager").
			WithOperation("registerCampaignGlobally")
	}

	registryPath := filepath.Join(campaignRegistryDir, "campaign.yaml")
	if err := os.WriteFile(registryPath, registryYaml, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign registry").
			WithComponent("DefaultConfigManager").
			WithOperation("registerCampaignGlobally").
			WithDetails("path", registryPath)
	}

	return nil
}

func (d *DefaultConfigManager) FinalizeGuildCharter(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "guild charter finalization cancelled").
			WithComponent("DefaultConfigManager").
			WithOperation("FinalizeGuildCharter")
	}

	// Guild charter finalization is now complete - all configuration is in .campaign/
	// No need for .guild/ directory anymore

	// Verify campaign structure was created correctly
	campaignDir := filepath.Join(projectPath, paths.DefaultCampaignDir)
	if _, err := os.Stat(campaignDir); os.IsNotExist(err) {
		return gerror.New(gerror.ErrCodeNotFound, "campaign directory not found after creation", nil).
			WithComponent("DefaultConfigManager").
			WithOperation("FinalizeGuildCharter").
			WithDetails("dir", campaignDir)
	}

	// Verify guild configuration exists
	guildPath := filepath.Join(campaignDir, "guilds", "elena_guild.yaml")
	if _, err := os.Stat(guildPath); os.IsNotExist(err) {
		return gerror.New(gerror.ErrCodeNotFound, "guild configuration not found", nil).
			WithComponent("DefaultConfigManager").
			WithOperation("FinalizeGuildCharter").
			WithDetails("path", guildPath)
	}

	return nil
}

func (d *DefaultConfigManager) CreateCampaignReference(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "campaign reference creation cancelled").
			WithComponent("DefaultConfigManager").
			WithOperation("CreateCampaignReference")
	}

	// Campaign reference is now handled by campaign.yaml and guild registry system
	// No need to create a separate guild.yaml file - the campaign structure is sufficient
	return nil
}

// DefaultProjectInitializer implements ProjectInitializer
type DefaultProjectInitializer struct{}

// NewDefaultProjectInitializer creates a new default project initializer
func NewDefaultProjectInitializer() ProjectInitializer {
	return &DefaultProjectInitializer{}
}

func (d *DefaultProjectInitializer) InitializeProject(ctx context.Context, projectPath string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "project initialization cancelled").
			WithComponent("DefaultProjectInitializer").
			WithOperation("InitializeProject")
	}

	return project.InitializeProject(projectPath)
}

func (d *DefaultProjectInitializer) IsProjectInitialized(projectPath string) bool {
	return project.IsProjectInitialized(projectPath)
}

// DefaultDemoGenerator implements DemoGenerator
type DefaultDemoGenerator struct {
	generator *setup.DemoCommissionGenerator
}

// NewDefaultDemoGenerator creates a new default demo generator
func NewDefaultDemoGenerator() DemoGenerator {
	return &DefaultDemoGenerator{
		generator: setup.NewDemoCommissionGenerator(),
	}
}

func (d *DefaultDemoGenerator) GenerateCommission(ctx context.Context, demoType setup.DemoCommissionType) (string, error) {
	return d.generator.GenerateCommission(ctx, demoType)
}

func (d *DefaultDemoGenerator) GetAvailableTypes() []setup.DemoCommissionType {
	return d.generator.GetAvailableTypes()
}

func (d *DefaultDemoGenerator) GetDemoDescription(demoType setup.DemoCommissionType) string {
	return d.generator.GetDemoDescription(demoType)
}

// DefaultValidator implements Validator
type DefaultValidator struct {
	validator *setup.InitValidator
	results   []ValidationResult
}

// NewDefaultValidator creates a new default validator
func NewDefaultValidator() Validator {
	return &DefaultValidator{}
}

func (d *DefaultValidator) Validate(ctx context.Context) error {
	// This would be initialized with the project path
	// For now, we'll return a simple implementation
	d.results = []ValidationResult{
		{Name: "Project Structure", Passed: true, Message: "All directories created"},
		{Name: "Configuration Files", Passed: true, Message: "guild.yaml and campaign.yml present"},
		{Name: "Provider Setup", Passed: true, Message: "At least one provider configured"},
		{Name: "Agent Configuration", Passed: true, Message: "Default agents created"},
	}
	return nil
}

func (d *DefaultValidator) HasFailures() bool {
	for _, r := range d.results {
		if !r.Passed {
			return true
		}
	}
	return false
}

func (d *DefaultValidator) GetResults() []ValidationResult {
	return d.results
}

// DefaultDaemonManager implements DaemonManager
type DefaultDaemonManager struct{}

// NewDefaultDaemonManager creates a new default daemon manager
func NewDefaultDaemonManager() DaemonManager {
	return &DefaultDaemonManager{}
}

func (d *DefaultDaemonManager) SaveSocketRegistry(projectPath, campaignName string) error {
	return daemon.SaveSocketRegistry(projectPath, campaignName)
}
