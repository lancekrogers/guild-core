// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package prompts

import (
	"context"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
	"github.com/lancekrogers/guild/pkg/prompts/standard"
)

// ManagerType defines the type of prompt manager
type ManagerType string

const (
	// TypeStandard represents the standard template-based prompt system
	TypeStandard ManagerType = "standard"

	// TypeLayered represents the advanced 6-layer prompt system
	TypeLayered ManagerType = "layered"
)

// ManagerConfig provides configuration for creating prompt managers
type ManagerConfig struct {
	// Type specifies which prompt manager to create
	Type ManagerType

	// TemplateDir is the directory containing prompt templates (for standard)
	TemplateDir string

	// LayeredConfig provides configuration for the layered system
	LayeredConfig *LayeredManagerConfig
}

// LayeredManagerConfig provides configuration specific to the layered prompt system
type LayeredManagerConfig struct {
	// DefaultPlatformPrompt is the default platform layer content
	DefaultPlatformPrompt string

	// DefaultGuildPrompt is the default guild layer content
	DefaultGuildPrompt string

	// StorePath is where to persist prompts (optional)
	StorePath string
}

// NewManager creates a prompt manager using the registry pattern
func NewManager(ctx context.Context, config ManagerConfig) (Manager, error) {
	registry := GetRegistry()
	return registry.CreateManager(ctx, config)
}

// newStandardManager creates a standard template-based prompt manager
func newStandardManager(ctx context.Context, config ManagerConfig) (Manager, error) {
	mgr, err := standard.NewPromptManager()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create standard prompt manager").
			WithComponent("prompts").
			WithOperation("newStandardManager")
	}

	// Load templates if directory specified
	if config.TemplateDir != "" {
		// TODO: Implement LoadFromDirectory in standard manager
		// if err := mgr.LoadFromDirectory(ctx, config.TemplateDir); err != nil {
		// 	return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load templates").
		// 		WithComponent("prompts").
		// 		WithOperation("newStandardManager").
		// 		WithDetails("template_dir", config.TemplateDir)
		// }
	}

	return mgr, nil
}

// newLayeredManager creates a layered prompt manager
func newLayeredManager(ctx context.Context, config ManagerConfig) (Manager, error) {
	if config.LayeredConfig == nil {
		config.LayeredConfig = &LayeredManagerConfig{}
	}

	// Create the layered manager
	mgr := layered.NewLayeredPromptManager()

	// Set default platform prompt if provided
	if config.LayeredConfig.DefaultPlatformPrompt != "" {
		if err := mgr.SetLayer(LayerPlatform, config.LayeredConfig.DefaultPlatformPrompt); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set platform layer").
				WithComponent("prompts").
				WithOperation("newLayeredManager")
		}
	}

	// Set default guild prompt if provided
	if config.LayeredConfig.DefaultGuildPrompt != "" {
		if err := mgr.SetLayer(LayerGuild, config.LayeredConfig.DefaultGuildPrompt); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set guild layer").
				WithComponent("prompts").
				WithOperation("newLayeredManager")
		}
	}

	return mgr, nil
}

// NewStandardManager is a convenience function to create a standard prompt manager
func NewStandardManager() (Manager, error) {
	registry := GetRegistry()
	return registry.CreateManager(context.Background(), ManagerConfig{
		Type: TypeStandard,
	})
}

// NewLayeredManager is a convenience function to create a layered prompt manager
func NewLayeredManager() (LayeredManager, error) {
	registry := GetRegistry()
	mgr, err := registry.CreateManager(context.Background(), ManagerConfig{
		Type: TypeLayered,
	})
	if err != nil {
		return nil, err
	}

	layeredMgr, ok := mgr.(LayeredManager)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeValidation, "manager is not a LayeredManager", nil).
			WithComponent("prompts").
			WithOperation("NewLayeredManager")
	}

	return layeredMgr, nil
}
