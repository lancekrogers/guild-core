package commission_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/guild-ventures/guild-core/pkg/prompts/layered/commission"
)

func TestManagerRefinementPrompt(t *testing.T) {
	prompt := commission.ManagerRefinementPrompt

	t.Run("ContainsEssentialElements", func(t *testing.T) {
		// Check for Guild terminology
		assert.Contains(t, prompt, "Guild Master")
		assert.Contains(t, prompt, "artisan")
		assert.Contains(t, prompt, "Workshop Board")
		assert.Contains(t, prompt, "commission")

		// Check for structural guidance
		assert.Contains(t, prompt, "hierarchical")
		assert.Contains(t, prompt, "markdown files")
		assert.Contains(t, prompt, "directory structure")

		// Check for task formatting rules
		assert.Contains(t, prompt, "Tasks Generated")
		assert.Contains(t, prompt, "Priority:")
		assert.Contains(t, prompt, "Estimate:")
		assert.Contains(t, prompt, "Dependencies:")
		assert.Contains(t, prompt, "Capabilities:")
		assert.Contains(t, prompt, "Description:")

		// Check for categories
		assert.Contains(t, prompt, "ARCH:")
		assert.Contains(t, prompt, "AUTH:")
		assert.Contains(t, prompt, "API:")
		assert.Contains(t, prompt, "UI:")
		assert.Contains(t, prompt, "DATA:")
	})

	t.Run("HasProperStructure", func(t *testing.T) {
		// Check major sections exist
		assert.Contains(t, prompt, "## Your Role")
		assert.Contains(t, prompt, "## Output Structure")
		assert.Contains(t, prompt, "## Task Formatting Rules")
		assert.Contains(t, prompt, "## Markdown File Structure")
		assert.Contains(t, prompt, "## Guidelines")
		assert.Contains(t, prompt, "## Categories for Task IDs")
		assert.Contains(t, prompt, "## Important Notes")
	})

	t.Run("IncludesExamples", func(t *testing.T) {
		// Check for directory structure example
		assert.Contains(t, prompt, "README.md")
		assert.Contains(t, prompt, "backend/")
		assert.Contains(t, prompt, "frontend/")
		assert.Contains(t, prompt, "infrastructure/")

		// Check for task example
		assert.Contains(t, prompt, "AUTH-001: Implement JWT token generation")
		assert.Contains(t, prompt, "Priority: high")
		assert.Contains(t, prompt, "Estimate: 4h")
	})
}

func TestTaskFormatTemplate(t *testing.T) {
	template := commission.TaskFormatTemplate

	t.Run("ContainsAllFields", func(t *testing.T) {
		assert.Contains(t, template, "{{.Category}}")
		assert.Contains(t, template, "{{.Number}}")
		assert.Contains(t, template, "{{.Title}}")
		assert.Contains(t, template, "{{.Priority}}")
		assert.Contains(t, template, "{{.Estimate}}")
		assert.Contains(t, template, "{{.Dependencies}}")
		assert.Contains(t, template, "{{.Capabilities}}")
		assert.Contains(t, template, "{{.Description}}")
	})

	t.Run("HasCorrectFormat", func(t *testing.T) {
		// Should start with task marker
		assert.True(t, strings.HasPrefix(template, "**Tasks Generated**:"))

		// Should have proper indentation markers
		assert.Contains(t, template, "- {{.Category}}-{{.Number}}:")
		assert.Contains(t, template, "  - Priority:")
		assert.Contains(t, template, "  - Estimate:")
		assert.Contains(t, template, "  - Dependencies:")
		assert.Contains(t, template, "  - Capabilities:")
		assert.Contains(t, template, "  - Description:")
	})
}

func TestDomainPrompts(t *testing.T) {
	t.Run("WebAppDomainPrompt", func(t *testing.T) {
		prompt := commission.WebAppDomainPrompt

		assert.Contains(t, prompt, "Web Applications")
		assert.Contains(t, prompt, "Frontend Structure")
		assert.Contains(t, prompt, "Backend Structure")
		assert.Contains(t, prompt, "state management")
		assert.Contains(t, prompt, "API endpoint")
		assert.Contains(t, prompt, "authentication")
	})

	t.Run("CLIToolDomainPrompt", func(t *testing.T) {
		prompt := commission.CLIToolDomainPrompt

		assert.Contains(t, prompt, "CLI Tools")
		assert.Contains(t, prompt, "Command Structure")
		assert.Contains(t, prompt, "subcommands")
		assert.Contains(t, prompt, "flags")
		assert.Contains(t, prompt, "Shell completion")
		assert.Contains(t, prompt, "Configuration file")
	})

	t.Run("LibraryDomainPrompt", func(t *testing.T) {
		prompt := commission.LibraryDomainPrompt

		assert.Contains(t, prompt, "Libraries")
		assert.Contains(t, prompt, "API Design")
		assert.Contains(t, prompt, "public APIs")
		assert.Contains(t, prompt, "backwards compatibility")
		assert.Contains(t, prompt, "Documentation generation")
		assert.Contains(t, prompt, "Versioning strategy")
	})

	t.Run("MicroserviceDomainPrompt", func(t *testing.T) {
		prompt := commission.MicroserviceDomainPrompt

		assert.Contains(t, prompt, "Microservices")
		assert.Contains(t, prompt, "Service Design")
		assert.Contains(t, prompt, "service boundaries")
		assert.Contains(t, prompt, "inter-service communication")
		assert.Contains(t, prompt, "Circuit breaker")
		assert.Contains(t, prompt, "Distributed tracing")
	})
}

func TestPromptsConsistency(t *testing.T) {
	// Ensure all prompts use consistent formatting
	prompts := map[string]string{
		"Manager":      commission.ManagerRefinementPrompt,
		"WebApp":       commission.WebAppDomainPrompt,
		"CLI":          commission.CLIToolDomainPrompt,
		"Library":      commission.LibraryDomainPrompt,
		"Microservice": commission.MicroserviceDomainPrompt,
	}

	for name, prompt := range prompts {
		t.Run(name+"_Formatting", func(t *testing.T) {
			// Check for proper markdown heading format
			lines := strings.Split(prompt, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "#") {
					// Ensure space after # symbols
					if len(line) > 1 && line[1] != ' ' && line[1] != '#' {
						t.Errorf("Improper heading format: %s", line)
					}
				}
			}
		})
	}
}

func TestPromptLength(t *testing.T) {
	// Ensure prompts are not too long for typical context windows
	t.Run("ManagerPromptLength", func(t *testing.T) {
		prompt := commission.ManagerRefinementPrompt
		// Rough estimate: 1 token ≈ 4 characters
		tokenEstimate := len(prompt) / 4

		// Should fit comfortably in most context windows
		assert.Less(t, tokenEstimate, 4000, "Manager prompt might be too long for some models")
	})
}

func TestPromptCompleteness(t *testing.T) {
	prompt := commission.ManagerRefinementPrompt

	t.Run("CoversAllAspects", func(t *testing.T) {
		// Essential aspects that should be covered
		aspects := []string{
			"commission",              // Input understanding
			"hierarchical",           // Structure requirement
			"markdown",               // Output format
			"Tasks Generated",        // Task formatting
			"Dependencies",           // Relationship tracking
			"Capabilities",           // Agent matching
			"Priority",              // Task prioritization
			"Estimate",              // Time planning
			"Testing Considerations", // Quality assurance
			"artisan",               // Guild terminology
		}

		for _, aspect := range aspects {
			assert.Contains(t, prompt, aspect, "Missing important aspect: %s", aspect)
		}
	})
}
