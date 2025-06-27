// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// AgentPresets manages pre-configured agent collections for quick setup
type AgentPresets struct {
	presets map[string]*PresetCollection
}

// NewAgentPresets creates a new agent preset manager
func NewAgentPresets(ctx context.Context) (*AgentPresets, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during preset creation").
			WithComponent("AgentPresets").
			WithOperation("NewAgentPresets")
	}

	ap := &AgentPresets{
		presets: make(map[string]*PresetCollection),
	}

	// Initialize built-in presets
	if err := ap.initializePresets(ctx); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize presets").
			WithComponent("AgentPresets").
			WithOperation("NewAgentPresets")
	}

	return ap, nil
}

// Preset returns a preset collection by ID
func (ap *AgentPresets) Preset(ctx context.Context, id string) (*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("Preset")
	}

	collection, exists := ap.presets[id]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "preset '%s' not found", id).
			WithComponent("AgentPresets").
			WithOperation("Preset").
			WithDetails("preset_id", id).
			WithDetails("available_presets", ap.ListPresets(ctx))
	}

	return collection, nil
}

// GetPreset returns a preset collection by ID (deprecated: use Preset instead)
func (ap *AgentPresets) GetPreset(ctx context.Context, id string) (*PresetCollection, error) {
	return ap.Preset(ctx, id)
}

// ListPresets returns all available preset collections
func (ap *AgentPresets) ListPresets(ctx context.Context) []string {
	if err := ctx.Err(); err != nil {
		return []string{}
	}

	presets := make([]string, 0, len(ap.presets))
	for id := range ap.presets {
		presets = append(presets, id)
	}
	return presets
}

// PresetsByType returns preset collections of a specific type
func (ap *AgentPresets) PresetsByType(ctx context.Context, presetType PresetType) ([]*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("PresetsByType")
	}

	collections := make([]*PresetCollection, 0)
	for _, collection := range ap.presets {
		if collection.Type == presetType {
			collections = append(collections, collection)
		}
	}

	return collections, nil
}

// GetPresetsByType returns preset collections of a specific type (deprecated: use PresetsByType instead)
func (ap *AgentPresets) GetPresetsByType(ctx context.Context, presetType PresetType) ([]*PresetCollection, error) {
	return ap.PresetsByType(ctx, presetType)
}

// PresetsByCategory returns preset collections for a specific project category
func (ap *AgentPresets) PresetsByCategory(ctx context.Context, category PresetCategory) ([]*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("PresetsByCategory")
	}

	collections := make([]*PresetCollection, 0)
	for _, collection := range ap.presets {
		if collection.Category == category {
			collections = append(collections, collection)
		}
	}

	return collections, nil
}

// GetPresetsByCategory returns preset collections for a specific project category (deprecated: use PresetsByCategory instead)
func (ap *AgentPresets) GetPresetsByCategory(ctx context.Context, category PresetCategory) ([]*PresetCollection, error) {
	return ap.PresetsByCategory(ctx, category)
}

// DemoPreset returns a demo-optimized preset for quick demonstrations
func (ap *AgentPresets) DemoPreset(ctx context.Context, providers []ConfiguredProvider) (*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("DemoPreset")
	}

	// Try to get the best demo preset based on available providers
	recommendations, err := ap.RecommendPresets(ctx, providers, &ProjectContext{
		ProjectType: "demo",
		Language:    "go",
	})
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get recommendations for demo").
			WithComponent("AgentPresets").
			WithOperation("DemoPreset")
	}

	// Find the best demo preset
	for _, rec := range recommendations {
		if rec.Collection.Type == PresetTypeDemo && rec.Compatible {
			return ap.AdaptPresetForProviders(ctx, rec.Collection, providers)
		}
	}

	// Fallback to minimal demo if no optimal demo found
	collection, exists := ap.presets["demo-minimal"]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no demo presets available", nil).
			WithComponent("AgentPresets").
			WithOperation("DemoPreset")
	}

	return ap.AdaptPresetForProviders(ctx, collection, providers)
}

// GetDemoPreset returns a demo-optimized preset for quick demonstrations (deprecated: use DemoPreset instead)
func (ap *AgentPresets) GetDemoPreset(ctx context.Context, providers []ConfiguredProvider) (*PresetCollection, error) {
	return ap.DemoPreset(ctx, providers)
}

// initializePresets sets up all built-in preset collections
func (ap *AgentPresets) initializePresets(ctx context.Context) error {
	presets := []*PresetCollection{
		ap.createDemoMinimalPreset(),
		ap.createDemoComprehensivePreset(),
		ap.createDevelopmentTeamPreset(),
		ap.createProductionTeamPreset(),
		ap.createWebDevelopmentPreset(),
		ap.createAPIDevelopmentPreset(),
		ap.createCLIToolPreset(),
		ap.createDataAnalysisPreset(),
		ap.createClaudeCodeOptimizedPreset(),
		ap.createOllamaLocalPreset(),
		ap.createMultiProviderPreset(),
	}

	for _, preset := range presets {
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during preset initialization").
				WithComponent("AgentPresets").
				WithOperation("initializePresets")
		}
		ap.presets[preset.ID] = preset
	}

	return nil
}
