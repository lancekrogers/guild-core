package objective

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/providers"
)

// Generator handles LLM-based generation of commissions and related documents
type Generator struct {
	client    providers.LLMClient
	promptMgr *prompts.PromptManager
}

// newGenerator creates a new commission generator (private constructor)
func newGenerator(client providers.LLMClient) (*Generator, error) {
	// Create prompt manager
	pm, err := prompts.NewPromptManager()
	if err != nil {
		return nil, fmt.Errorf("error creating prompt manager: %w", err)
	}

	return &Generator{
		client:    client,
		promptMgr: pm,
	}, nil
}

// DefaultGeneratorFactory creates a commission generator factory for registry use
func DefaultGeneratorFactory(client providers.LLMClient) (*Generator, error) {
	return newGenerator(client)
}

// GenerateCommission creates a new commission from a description
func (g *Generator) GenerateCommission(ctx context.Context, description string) (*commission.Commission, error) {
	// Prepare data for template
	data := map[string]interface{}{
		"Description": description,
	}

	// Render the prompt
	prompt, err := g.promptMgr.RenderPrompt("commission.creation", data)
	if err != nil {
		return nil, fmt.Errorf("error rendering prompt: %w", err)
	}

	// Call the LLM using the simplified interface
	response, err := g.client.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling LLM: %w", err)
	}

	// Create a temporary file to parse
	tempFile := filepath.Join(os.TempDir(), "temp_commission.md")
	if err := os.WriteFile(tempFile, []byte(response), 0644); err != nil {
		return nil, fmt.Errorf("error writing temp file: %w", err)
	}
	defer os.Remove(tempFile)

	// Parse the commission from the response
	obj, err := commission.ParseFile(tempFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing commission: %w", err)
	}

	return obj, nil
}

// GenerateAIDocs generates AI documentation based on a commission
func (g *Generator) GenerateAIDocs(ctx context.Context, obj *commission.Commission, additionalContext string) (map[string]string, error) {
	// Prepare data for template
	data := map[string]interface{}{
		"Commission":        obj.Format(), // Method to format commission as markdown
		"AdditionalContext": additionalContext,
	}

	// Render the prompt
	prompt, err := g.promptMgr.RenderPrompt("commission.ai_docs_gen", data)
	if err != nil {
		return nil, fmt.Errorf("error rendering prompt: %w", err)
	}

	// Call the LLM using the simplified interface
	response, err := g.client.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("error calling LLM: %w", err)
	}

	// Parse the response into multiple markdown documents
	docs := parseMultipleMarkdownDocs(response)
	return docs, nil
}

// GenerateSpecs generates technical specifications based on a commission
func (g *Generator) GenerateSpecs(ctx context.Context, obj *commission.Commission, additionalContext string) (map[string]string, error) {
	// Similar implementation to GenerateAIDocs, but using the "commission.specs_gen" prompt
	// ...

	return nil, nil // Placeholder
}

// SuggestImprovements suggests improvements to a commission
func (g *Generator) SuggestImprovements(ctx context.Context, obj *commission.Commission) (string, error) {
	// Similar implementation using "commission.suggestion" prompt
	// ...

	return "", nil // Placeholder
}

// Helper function to parse multiple markdown documents from a single response
func parseMultipleMarkdownDocs(text string) map[string]string {
	docs := make(map[string]string)

	// Simple implementation - look for markdown file headers
	// In a real implementation, you'd want a more robust parser
	sections := strings.Split(text, "```markdown")

	for i, section := range sections {
		if i == 0 {
			continue // Skip the intro text
		}

		// Extract the content and clean it up
		content := strings.Split(section, "```")[0]
		content = strings.TrimSpace(content)

		// Try to determine a filename from the first heading
		lines := strings.Split(content, "\n")
		filename := "document.md"

		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				// Convert the heading to a filename
				heading := strings.TrimPrefix(line, "# ")
				heading = strings.TrimSpace(heading)
				filename = strings.ToLower(heading)
				filename = strings.ReplaceAll(filename, " ", "_")
				filename = filename + ".md"
				break
			}
		}

		docs[filename] = content
	}

	return docs
}
