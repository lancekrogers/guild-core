// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// RecommendPresets analyzes providers and project context to recommend optimal presets
func (ap *AgentPresets) RecommendPresets(ctx context.Context, providers []ConfiguredProvider, projectContext *ProjectContext) ([]*PresetRecommendation, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentPresets").
			WithOperation("RecommendPresets")
	}

	if len(providers) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "no providers available for recommendations", nil).
			WithComponent("AgentPresets").
			WithOperation("RecommendPresets")
	}

	recommendations := make([]*PresetRecommendation, 0)

	// Analyze provider capabilities
	analysis := ap.analyzeProviderCapabilities(providers)

	// Generate recommendations for each preset
	for _, collection := range ap.presets {
		recommendation := ap.evaluatePreset(ctx, collection, analysis, projectContext)
		if recommendation.Confidence > 0.1 { // Only include viable recommendations
			recommendations = append(recommendations, recommendation)
		}
	}

	// Sort by confidence (highest first)
	ap.sortRecommendationsByConfidence(recommendations)

	return recommendations, nil
}

// analyzeProviderCapabilities analyzes available providers and their capabilities
func (ap *AgentPresets) analyzeProviderCapabilities(providers []ConfiguredProvider) *ProviderCapabilities {
	caps := &ProviderCapabilities{}

	for _, provider := range providers {
		caps.ModelCount += len(provider.Models)

		for _, model := range provider.Models {
			// Analyze model characteristics
			if model.CostMagnitude == 0 {
				caps.HasLocal = true
			} else {
				caps.HasCloud = true
			}

			if model.CostMagnitude >= 5 {
				caps.HasHighEnd = true
			}

			if model.CostMagnitude <= 2 {
				caps.HasCheap = true
			}

			// Find best models for different roles
			selection := ModelSelection{
				Provider:      provider.Name,
				Model:         model,
				Available:     true,
				CostEffective: model.CostMagnitude <= 3,
			}

			if caps.BestManager.Provider == "" || ap.isBetterManagerModel(selection, caps.BestManager) {
				caps.BestManager = selection
			}

			if caps.BestWorker.Provider == "" || ap.isBetterWorkerModel(selection, caps.BestWorker) {
				caps.BestWorker = selection
			}

			if caps.BestSpecialist.Provider == "" || ap.isBetterSpecialistModel(selection, caps.BestSpecialist) {
				caps.BestSpecialist = selection
			}
		}
	}

	return caps
}

// evaluatePreset evaluates how well a preset fits the available providers and context
func (ap *AgentPresets) evaluatePreset(ctx context.Context, collection *PresetCollection, caps *ProviderCapabilities, projectContext *ProjectContext) *PresetRecommendation {
	recommendation := &PresetRecommendation{
		Collection: collection,
		Confidence: 0.0,
		Reasoning:  []string{},
		Compatible: false,
	}

	// Check basic compatibility
	if caps.ModelCount < collection.MinModels {
		recommendation.Reasoning = append(recommendation.Reasoning,
			fmt.Sprintf("Requires %d models, only %d available", collection.MinModels, caps.ModelCount))
		return recommendation
	}

	recommendation.Compatible = true
	baseConfidence := 0.5

	// Boost confidence for demo presets in demo context
	if projectContext != nil && projectContext.ProjectType == "demo" && collection.Type == PresetTypeDemo {
		baseConfidence += 0.3
		recommendation.Reasoning = append(recommendation.Reasoning, "Optimized for demo context")
	}

	// Boost confidence for category matches
	if projectContext != nil {
		categoryMatch := ap.getProjectCategory(projectContext)
		if categoryMatch == collection.Category {
			baseConfidence += 0.2
			recommendation.Reasoning = append(recommendation.Reasoning, "Category matches project type")
		}
	}

	// Adjust based on provider capabilities
	switch collection.Type {
	case PresetTypeDemo:
		if caps.HasCloud {
			baseConfidence += 0.1
			recommendation.Reasoning = append(recommendation.Reasoning, "Cloud models available for impressive demos")
		}
	case PresetTypeProduction:
		if caps.HasHighEnd {
			baseConfidence += 0.2
			recommendation.Reasoning = append(recommendation.Reasoning, "High-end models available for production quality")
		} else {
			baseConfidence -= 0.2
			recommendation.Reasoning = append(recommendation.Reasoning, "Limited high-end models for production use")
		}
	case PresetTypeMinimal:
		if caps.HasCheap || caps.HasLocal {
			baseConfidence += 0.2
			recommendation.Reasoning = append(recommendation.Reasoning, "Cost-effective models available")
		}
	}

	// Special handling for provider-specific presets
	switch collection.ID {
	case "claude-code-optimized":
		if ap.hasProvider(caps, "claude_code") {
			baseConfidence += 0.3
			recommendation.Reasoning = append(recommendation.Reasoning, "Claude Code detected")
		} else {
			baseConfidence -= 0.4
			recommendation.Reasoning = append(recommendation.Reasoning, "Requires Claude Code provider")
		}
	case "ollama-local":
		if caps.HasLocal {
			baseConfidence += 0.3
			recommendation.Reasoning = append(recommendation.Reasoning, "Local models detected")
		} else {
			baseConfidence -= 0.5
			recommendation.Reasoning = append(recommendation.Reasoning, "Requires local models")
		}
	}

	recommendation.Confidence = baseConfidence
	if recommendation.Confidence < 0 {
		recommendation.Confidence = 0
	}
	if recommendation.Confidence > 1 {
		recommendation.Confidence = 1
	}

	return recommendation
}

// Helper methods

func (ap *AgentPresets) sortRecommendationsByConfidence(recommendations []*PresetRecommendation) {
	// Simple bubble sort by confidence (descending)
	n := len(recommendations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if recommendations[j].Confidence < recommendations[j+1].Confidence {
				recommendations[j], recommendations[j+1] = recommendations[j+1], recommendations[j]
			}
		}
	}
}

func (ap *AgentPresets) getProjectCategory(context *ProjectContext) PresetCategory {
	if context == nil {
		return PresetCategoryGeneral
	}

	switch strings.ToLower(context.ProjectType) {
	case "web", "webapp", "website":
		return PresetCategoryWeb
	case "api", "microservice", "service":
		return PresetCategoryAPI
	case "cli", "tool", "command":
		return PresetCategoryCLI
	case "data", "analytics", "ml", "ai":
		return PresetCategoryData
	default:
		return PresetCategoryGeneral
	}
}

func (ap *AgentPresets) hasProvider(caps *ProviderCapabilities, providerName string) bool {
	// This is a simplified check - in a real implementation, you'd check the actual providers
	return caps.HasCloud || caps.HasLocal
}

func (ap *AgentPresets) isBetterManagerModel(a, b ModelSelection) bool {
	// Prefer models with larger context windows for managers
	if a.Model.ContextWindow > b.Model.ContextWindow {
		return true
	}
	// Prefer recommended models
	if a.Model.Recommended && !b.Model.Recommended {
		return true
	}
	return false
}

func (ap *AgentPresets) isBetterWorkerModel(a, b ModelSelection) bool {
	// Prefer cost-effective models for workers
	if a.CostEffective && !b.CostEffective {
		return true
	}
	// Among cost-effective models, prefer recommended
	if a.CostEffective == b.CostEffective && a.Model.Recommended && !b.Model.Recommended {
		return true
	}
	return false
}

func (ap *AgentPresets) isBetterSpecialistModel(a, b ModelSelection) bool {
	// For specialists, prefer higher-capability models
	if a.Model.Recommended && !b.Model.Recommended {
		return true
	}
	if a.Model.ContextWindow > b.Model.ContextWindow {
		return true
	}
	return false
}
