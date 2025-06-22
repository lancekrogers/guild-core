// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// AdaptPresetForProviders adapts a preset collection to work with available providers
func (ap *AgentPresets) AdaptPresetForProviders(ctx context.Context, collection *PresetCollection, providers []ConfiguredProvider) (*PresetCollection, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("AdaptPresetForProviders")
	}

	if collection == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "preset collection is required", nil).
			WithComponent("AgentPresets").
			WithOperation("AdaptPresetForProviders")
	}

	if len(providers) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no providers available for adaptation", nil).
			WithComponent("AgentPresets").
			WithOperation("AdaptPresetForProviders")
	}

	// Create adapted copy of the collection
	adapted := &PresetCollection{
		ID:          collection.ID + "-adapted",
		Name:        collection.Name + " (Adapted)",
		Description: collection.Description,
		Type:        collection.Type,
		Category:    collection.Category,
		Reasoning:   append([]string{"Adapted for available providers"}, collection.Reasoning...),
		MinModels:   collection.MinModels,
	}

	// Adapt each agent configuration
	for _, agent := range collection.Agents {
		adaptedAgent, err := ap.adaptAgentForProviders(ctx, agent, providers)
		if err != nil {
			return nil, gerror.Wrapf(err, gerror.ErrCodeInternal, "failed to adapt agent '%s'", agent.ID).
				WithComponent("AgentPresets").
				WithOperation("AdaptPresetForProviders")
		}
		adapted.Agents = append(adapted.Agents, *adaptedAgent)
	}

	return adapted, nil
}

// adaptAgentForProviders adapts an agent configuration to available providers
func (ap *AgentPresets) adaptAgentForProviders(ctx context.Context, agent config.AgentConfig, providers []ConfiguredProvider) (*config.AgentConfig, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("adaptAgentForProviders")
	}

	adapted := agent // Copy the agent

	// If agent already has a specific provider that's available, keep it
	if agent.Provider != "auto" {
		for _, provider := range providers {
			if provider.Name == agent.Provider {
				// Check if the model is available
				for _, model := range provider.Models {
					if model.Name == agent.Model {
						return &adapted, nil // Already compatible
					}
				}
			}
		}
	}

	// Need to adapt - find best matching provider and model
	caps := ap.analyzeProviderCapabilities(providers)

	var selectedModel ModelSelection
	switch agent.Type {
	case "manager":
		selectedModel = caps.BestManager
	case "specialist":
		selectedModel = caps.BestSpecialist
	default:
		selectedModel = caps.BestWorker
	}

	if selectedModel.Provider == "" {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no suitable model found for agent type '%s'", agent.Type).
			WithComponent("AgentPresets").
			WithOperation("adaptAgentForProviders")
	}

	adapted.Provider = selectedModel.Provider
	adapted.Model = selectedModel.Model.Name
	adapted.CostMagnitude = selectedModel.Model.CostMagnitude
	adapted.ContextWindow = selectedModel.Model.ContextWindow

	return &adapted, nil
}
