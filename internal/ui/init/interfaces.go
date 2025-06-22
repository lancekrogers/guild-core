// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"context"

	"github.com/guild-ventures/guild-core/internal/setup"
)

// ConfigurationManager handles all configuration operations
type ConfigurationManager interface {
	CreatePhase0Configuration(ctx context.Context, projectPath, campaignName, projectName string) error
	IntegrateWithPhase0Config(ctx context.Context, projectPath, campaignName, projectName string) error
	CreateCampaignReference(ctx context.Context, projectPath, campaignName, projectName string) error
}

// ProjectInitializer handles project initialization
type ProjectInitializer interface {
	InitializeProject(ctx context.Context, projectPath string) error
	IsProjectInitialized(projectPath string) bool
}

// DemoGenerator handles demo commission generation
type DemoGenerator interface {
	GenerateCommission(ctx context.Context, demoType setup.DemoCommissionType) (string, error)
	GetAvailableTypes() []setup.DemoCommissionType
	GetDemoDescription(demoType setup.DemoCommissionType) string
}

// Validator handles post-initialization validation
type Validator interface {
	Validate(ctx context.Context) error
	HasFailures() bool
	GetResults() []ValidationResult
}

// ValidationResult represents a single validation check result
type ValidationResult struct {
	Name    string
	Passed  bool
	Message string
}

// DaemonManager handles daemon-related operations
type DaemonManager interface {
	SaveSocketRegistry(projectPath, campaignName string) error
}
