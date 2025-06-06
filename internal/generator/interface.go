package generator

import (
	"context"

	"github.com/guild-ventures/guild-core/internal/commission"
)

// CommissionGenerator defines the interface for commission-related content generation
type CommissionGenerator interface {
	// GenerateCommission creates a new commission from a description
	GenerateCommission(ctx context.Context, description string) (*commission.Commission, error)

	// GenerateAIDocs generates AI documentation based on a commission
	GenerateAIDocs(ctx context.Context, obj *commission.Commission, additionalContext string) (map[string]string, error)

	// GenerateSpecs generates technical specifications based on a commission
	GenerateSpecs(ctx context.Context, obj *commission.Commission, additionalContext string) (map[string]string, error)

	// SuggestImprovements suggests improvements to a commission
	SuggestImprovements(ctx context.Context, obj *commission.Commission) (string, error)
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
