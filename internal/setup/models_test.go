// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"testing"
)

func TestNewModelConfig(t *testing.T) {
	ctx := context.Background()

	modelConfig, err := NewModelConfig(ctx)
	if err != nil {
		t.Fatalf("Failed to create model config: %v", err)
	}

	if modelConfig == nil {
		t.Fatal("Model config is nil")
	}
	if modelConfig.models == nil {
		t.Fatal("Models map is nil")
	}

	// Check that some known providers are present
	expectedProviders := []string{"openai", "anthropic", "ollama", "claude_code"}
	for _, provider := range expectedProviders {
		if _, exists := modelConfig.models[provider]; !exists {
			t.Errorf("Expected provider '%s' to be present in models", provider)
		}
	}
}

func TestGetModelsForProvider(t *testing.T) {
	ctx := context.Background()
	modelConfig, _ := NewModelConfig(ctx)

	// Test OpenAI models
	models, err := modelConfig.GetModelsForProvider(ctx, "openai")
	if err != nil {
		t.Fatalf("Failed to get OpenAI models: %v", err)
	}
	if len(models) == 0 {
		t.Error("Expected OpenAI models to be present")
	}

	// Test invalid provider
	_, err = modelConfig.GetModelsForProvider(ctx, "invalid-provider")
	if err == nil {
		t.Error("Expected error for invalid provider")
	}
}

func TestGetRecommendedModels(t *testing.T) {
	ctx := context.Background()
	modelConfig, _ := NewModelConfig(ctx)

	models, err := modelConfig.GetRecommendedModels(ctx)
	if err != nil {
		t.Fatalf("Failed to get recommended models: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected some recommended models")
	}

	// Check that recommended models are actually marked as recommended
	for _, model := range models {
		if !model.Recommended {
			t.Errorf("Model '%s' in recommended list is not marked as recommended", model.Name)
		}
	}
}

func TestGetModelsByCapability(t *testing.T) {
	ctx := context.Background()
	modelConfig, _ := NewModelConfig(ctx)

	// Test coding capability
	models, err := modelConfig.GetModelsByCapability(ctx, "coding")
	if err != nil {
		t.Fatalf("Failed to get models by capability: %v", err)
	}
	if len(models) == 0 {
		t.Error("Expected models with coding capability")
	}

	// Test invalid capability
	_, err = modelConfig.GetModelsByCapability(ctx, "invalid-capability")
	if err == nil {
		t.Error("Expected error for invalid capability")
	}
}

func TestEstimateMonthlyCost(t *testing.T) {
	ctx := context.Background()
	modelConfig, _ := NewModelConfig(ctx)

	// Create a test model
	testModel := ModelInfo{
		Name:               "test-model",
		Provider:           "test",
		CostPerInputToken:  0.001,
		CostPerOutputToken: 0.002,
		CostMagnitude:      2,
	}

	estimate := modelConfig.EstimateMonthlyCost(ctx, testModel, 1000, 500)
	if estimate == nil {
		t.Fatal("Cost estimate is nil")
	}

	expectedInputCost := float64(1000*30) * 0.001
	expectedOutputCost := float64(500*30) * 0.002
	expectedTotal := expectedInputCost + expectedOutputCost

	if estimate.InputCost != expectedInputCost {
		t.Errorf("Expected input cost %f, got %f", expectedInputCost, estimate.InputCost)
	}
	if estimate.OutputCost != expectedOutputCost {
		t.Errorf("Expected output cost %f, got %f", expectedOutputCost, estimate.OutputCost)
	}
	if estimate.TotalCost != expectedTotal {
		t.Errorf("Expected total cost %f, got %f", expectedTotal, estimate.TotalCost)
	}
}

func TestGetCostComparison(t *testing.T) {
	ctx := context.Background()
	modelConfig, _ := NewModelConfig(ctx)

	comparison, err := modelConfig.GetCostComparison(ctx, "coding", 1000, 500)
	if err != nil {
		t.Fatalf("Failed to get cost comparison: %v", err)
	}

	if comparison == nil {
		t.Fatal("Cost comparison is nil")
	}
	if len(comparison.Estimates) == 0 {
		t.Error("Expected cost estimates")
	}
	if comparison.Capability != "coding" {
		t.Errorf("Expected capability 'coding', got '%s'", comparison.Capability)
	}

	// Check that estimates are sorted by cost (lowest first)
	for i := 1; i < len(comparison.Estimates); i++ {
		if comparison.Estimates[i].TotalCost < comparison.Estimates[i-1].TotalCost {
			t.Error("Cost estimates are not sorted correctly")
		}
	}
}

func TestGetModelRecommendations(t *testing.T) {
	ctx := context.Background()
	modelConfig, _ := NewModelConfig(ctx)

	recommendations, err := modelConfig.GetModelRecommendations(ctx, "coding", "low")
	if err != nil {
		t.Fatalf("Failed to get model recommendations: %v", err)
	}

	if recommendations == nil {
		t.Fatal("Recommendations is nil")
	}
	if recommendations.UseCase != "coding" {
		t.Errorf("Expected use case 'coding', got '%s'", recommendations.UseCase)
	}
	if recommendations.Budget != "low" {
		t.Errorf("Expected budget 'low', got '%s'", recommendations.Budget)
	}

	// Check that recommendations respect budget constraints
	for _, rec := range recommendations.Recommendations {
		if rec.Model.CostMagnitude > 1 { // "low" budget should have cost magnitude <= 1
			t.Errorf("Model '%s' exceeds low budget constraint (cost magnitude %d)", 
				rec.Model.Name, rec.Model.CostMagnitude)
		}
	}
}