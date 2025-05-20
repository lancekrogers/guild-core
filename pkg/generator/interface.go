package generator

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/objective"
)

// ObjectiveGenerator defines the interface for objective-related content generation
type ObjectiveGenerator interface {
	// GenerateObjective creates a new objective from a description
	GenerateObjective(ctx context.Context, description string) (*objective.Objective, error)

	// GenerateAIDocs generates AI documentation based on an objective
	GenerateAIDocs(ctx context.Context, obj *objective.Objective, additionalContext string) (map[string]string, error)

	// GenerateSpecs generates technical specifications based on an objective
	GenerateSpecs(ctx context.Context, obj *objective.Objective, additionalContext string) (map[string]string, error)

	// SuggestImprovements suggests improvements to an objective
	SuggestImprovements(ctx context.Context, obj *objective.Objective) (string, error)
}

// DocsGenerator defines the interface for AI documentation generation
type DocsGenerator interface {
	// GenerateDocumentation generates documentation for a specific topic
	GenerateDocumentation(ctx context.Context, topic string, references []string) (string, error)

	// EnhanceDocumentation improves existing documentation
	EnhanceDocumentation(ctx context.Context, existing string, feedback string) (string, error)
}

// SpecsGenerator defines the interface for technical specification generation
type SpecsGenerator interface {
	// GenerateSpecification generates a technical specification
	GenerateSpecification(ctx context.Context, topic string, requirements []string) (string, error)

	// ValidateSpecification checks a specification for completeness
	ValidateSpecification(ctx context.Context, spec string) ([]string, error)
}
