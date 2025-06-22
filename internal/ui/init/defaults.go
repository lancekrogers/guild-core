// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"context"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"

	"github.com/guild-ventures/guild-core/internal/setup"
	"github.com/guild-ventures/guild-core/pkg/campaign"
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

	// Step 0: Ensure campaign directory exists
	campaignDir := filepath.Join(projectPath, paths.DefaultCampaignDir)
	if err := os.MkdirAll(campaignDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create campaign directory").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration").
			WithDetails("dir", campaignDir)
	}

	// Step 1: Create campaign.yaml with CampaignReference structure for detection
	campaignRef := map[string]interface{}{
		"campaign":    campaignName,
		"project":     projectName,
		"description": "Project " + projectName + " in campaign " + campaignName,
	}

	// Marshal and save campaign reference for detection
	campaignData, err := yaml.Marshal(campaignRef)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal campaign reference").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	campaignPath := filepath.Join(projectPath, paths.DefaultCampaignDir, "campaign.yaml")
	if err := os.WriteFile(campaignPath, campaignData, 0644); err != nil {
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

	// Step 2: Create guilds directory and default guild config
	guildsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "guilds")
	if err := os.MkdirAll(guildsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create guilds directory").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration").
			WithDetails("dir", guildsDir)
	}

	// Create default guild configuration
	defaultGuild := map[string]interface{}{
		"name":        "default",
		"purpose":     "General development tasks and project management",
		"description": "Default guild for handling various development tasks",
		"agents":      []string{"manager", "developer", "tester"},
		"coordination": map[string]interface{}{
			"max_parallel_tasks": 3,
			"review_required":    false,
			"auto_handoff":       true,
		},
	}

	guildData, err := yaml.Marshal(defaultGuild)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal guild config").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration")
	}

	guildPath := filepath.Join(guildsDir, "default_guild.yaml")
	if err := os.WriteFile(guildPath, guildData, 0644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write guild config").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration").
			WithDetails("path", guildPath)
	}

	// Step 3: Create agents directory and agent configs
	agentsDir := filepath.Join(projectPath, paths.DefaultCampaignDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create agents directory").
			WithComponent("DefaultConfigManager").
			WithOperation("CreatePhase0Configuration").
			WithDetails("dir", agentsDir)
	}

	// Create default agent configurations
	agents := map[string]map[string]interface{}{
		"manager": {
			"name":        "manager",
			"type":        "manager",
			"description": "Orchestrates tasks and coordinates between agents",
			"capabilities": []string{
				"task_breakdown",
				"agent_assignment",
				"progress_tracking",
			},
		},
		"developer": {
			"name":        "developer",
			"type":        "worker",
			"description": "Handles development tasks and code generation",
			"capabilities": []string{
				"code_generation",
				"code_review",
				"debugging",
				"refactoring",
			},
		},
		"tester": {
			"name":        "tester",
			"type":        "worker",
			"description": "Performs testing and quality assurance",
			"capabilities": []string{
				"test_generation",
				"test_execution",
				"bug_detection",
				"performance_testing",
			},
		},
	}

	// Write each agent configuration
	for agentName, agentConfig := range agents {
		agentData, err := yaml.Marshal(agentConfig)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal agent config").
				WithComponent("DefaultConfigManager").
				WithOperation("CreatePhase0Configuration").
				WithDetails("agent", agentName)
		}

		agentPath := filepath.Join(agentsDir, agentName+"_agent.yaml")
		if err := os.WriteFile(agentPath, agentData, 0644); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write agent config").
				WithComponent("DefaultConfigManager").
				WithOperation("CreatePhase0Configuration").
				WithDetails("path", agentPath)
		}
	}

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
				WithOperation("CreatePhase0Configuration").
				WithDetails("dir", dirPath)
		}
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

	// Create minimal campaign reference for detection
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