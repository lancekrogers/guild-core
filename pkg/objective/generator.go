package objective

import (
	"context"
	"fmt"
)

// Generator is responsible for generating objectives
type Generator struct {
	manager *Manager
}

// NewGenerator creates a new objective generator
func NewGenerator(manager *Manager) *Generator {
	return &Generator{
		manager: manager,
	}
}

// GenerateFromPrompt generates an objective from a user prompt
func (g *Generator) GenerateFromPrompt(ctx context.Context, prompt string) (*Objective, error) {
	// In a real implementation, this would use an LLM to generate an objective
	// For now, we'll create a simple objective based on the prompt
	obj := &Objective{
		Title:       prompt,
		Description: "Auto-generated objective",
		Status:      ObjectiveStatusDraft,
		Parts:       []*ObjectivePart{},
	}

	return obj, nil
}

// SaveGeneratedObjective saves a generated objective
func (g *Generator) SaveGeneratedObjective(ctx context.Context, obj *Objective) error {
	if obj == nil {
		return fmt.Errorf("objective is nil")
	}

	// Save the objective using the manager
	return g.manager.SaveObjective(ctx, obj)
}