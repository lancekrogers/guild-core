// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"github.com/lancekrogers/guild-core/pkg/config"
)

// PresetType defines the type of preset collection
type PresetType string

const (
	PresetTypeDemo        PresetType = "demo"
	PresetTypeDevelopment PresetType = "development"
	PresetTypeProduction  PresetType = "production"
	PresetTypeMinimal     PresetType = "minimal"
)

// PresetCategory defines the project category for targeted presets
type PresetCategory string

const (
	PresetCategoryWeb     PresetCategory = "web"
	PresetCategoryAPI     PresetCategory = "api"
	PresetCategoryCLI     PresetCategory = "cli"
	PresetCategoryData    PresetCategory = "data"
	PresetCategoryGeneral PresetCategory = "general"
)

// PresetCollection contains a collection of related agent configurations
type PresetCollection struct {
	ID          string
	Name        string
	Description string
	Type        PresetType
	Category    PresetCategory
	Agents      []config.AgentConfig
	Reasoning   []string
	MinModels   int // Minimum models required for this preset
}

// PresetRecommendation contains recommendations for preset selection
type PresetRecommendation struct {
	Collection *PresetCollection
	Confidence float64 // 0.0-1.0 confidence score
	Reasoning  []string
	Compatible bool // Whether current providers support this preset
}

// ProviderCapabilities contains analysis of provider capabilities
type ProviderCapabilities struct {
	HasLocal       bool
	HasCloud       bool
	HasHighEnd     bool
	HasCheap       bool
	ModelCount     int
	BestManager    ModelSelection
	BestWorker     ModelSelection
	BestSpecialist ModelSelection
}
