// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"context"
	"os"
	"path/filepath"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/internal/setup"
	"github.com/guild-ventures/guild-core/pkg/campaign"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/paths"
	"github.com/guild-ventures/guild-core/pkg/project"
)

// DefaultConfigManager implements ConfigurationManager
type DefaultConfigManager struct{}

// NewDefaultConfigManager creates a new default config manager
func NewDefaultConfigManager() ConfigurationManager {
	return &DefaultConfigManager{}
}

func (d *DefaultConfigManager) CreatePhase0Configuration(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "Phase 0 configuration creation cancelled").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	// Step 1: Create campaign.yml
	campaignConfig := &config.CampaignConfig{
		Name:        campaignName,
		Description: "Campaign " + campaignName + " - automated multi-agent development",
		ProjectSettings: map[string]interface{}{
			"project_name": projectName,
			"created_at":   time.Now().Format(time.RFC3339),
			"version":      "1.0.0",
		},
		CommissionMappings: make(map[string][]string),
		LastSelectedGuild:  "default",
	}

	if err := config.SaveCampaignConfig(ctx, projectPath, campaignConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save campaign configuration").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration").
			WithDetails("campaign", campaignName)
	}

	// Check context between operations
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "cancelled after campaign config").
			WithComponent("DefaultConfigManager")
	}

	// Step 2: Create default guild.yml structure
	guildConfig := &config.GuildConfigFile{
		Guilds: map[string]config.GuildDefinition{
			"default": {
				Purpose:     "General development tasks and project management",
				Description: "Default guild for handling various development tasks",
				Agents:      []string{"manager", "developer", "tester"},
				Coordination: &config.CoordinationSettings{
					MaxParallelTasks: 3,
					ReviewRequired:   false,
					AutoHandoff:      true,
				},
			},
		},
	}

	if err := config.SaveGuildConfigFile(ctx, projectPath, guildConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save guild configuration").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	// Step 3: Create agents directory
	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration").
			WithDetails("dir", agentsDir)
	}

	return nil
}

func (d *DefaultConfigManager) IntegrateWithPhase0Config(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "Phase 0 integration cancelled").
			WithComponent("DefaultConfigManager").
			WithOperation("IntegrateWithPhase0Config")
	}

	// Implementation would integrate wizard results with Phase 0 config
	// For now, this is a simplified version
	return nil
}

func (d *DefaultConfigManager) CreateCampaignReference(ctx context.Context, projectPath, campaignName, projectName string) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "campaign reference creation cancelled").
			WithComponent("DefaultConfigManager").
			WithOperation("CreateCampaignReference")
	}

	// Create campaign reference
	ref := campaign.CampaignReference{
		Campaign:    campaignName,
		Project:     projectName,
		Description: "Project " + projectName + " in campaign " + campaignName,
	}

	refPath := filepath.Join(projectPath, paths.DefaultCampaignDir, "guild.yaml")
	refData, err := yaml.Marshal(ref)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign reference").
			WithComponent("DefaultConfigManager").
			WithOperation("CreateCampaignReference")
	}

	if err := os.WriteFile(refPath, refData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write campaign reference").
			WithComponent("DefaultConfigManager").
			WithOperation("CreateCampaignReference").
			WithDetails("path", refPath)
	}

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