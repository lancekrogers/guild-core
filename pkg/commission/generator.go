package commission

import (
	"context"
	"fmt"
)

// Generator is responsible for generating objectives
type Generator struct {
	manager *Manager
}

// newGenerator creates a new objective generator (private constructor)
func newGenerator(manager *Manager) *Generator {
	return &Generator{
		manager: manager,
	}
}

// DefaultGeneratorFactory creates a generator factory for registry use
func DefaultGeneratorFactory(manager *Manager) *Generator {
	return newGenerator(manager)
}

// GenerateFromPrompt generates an objective from a user prompt
func (g *Generator) GenerateFromPrompt(ctx context.Context, prompt string) (*Commission, error) {
	// In a real implementation, this would use an LLM to generate a commission
	// For now, we'll create a simple commission based on the prompt
	obj := &Commission{
		Title:       prompt,
		Description: "Auto-generated objective",
		Status:      CommissionStatusDraft,
		Parts:       []*CommissionPart{},
	}

	return obj, nil
}

// SaveGeneratedObjective saves a generated objective
func (g *Generator) SaveGeneratedCommission(ctx context.Context, obj *Commission) error {
	if obj == nil {
		return fmt.Errorf("objective is nil")
	}

	// Save the objective using the manager
	return g.manager.SaveObjective(ctx, obj)
}
