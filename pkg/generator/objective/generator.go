package objective

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/guild/internal/prompts"
	"github.com/yourusername/guild/pkg/objective"
	"github.com/yourusername/guild/pkg/providers"
)

// Generator handles LLM-based generation of objectives and related documents
type Generator struct {
	client    providers.LLMClient
	promptMgr *prompts.PromptManager
}

// NewGenerator creates a new objective generator
func NewGenerator(client providers.LLMClient) (*Generator, error) {
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

// GenerateObjective creates a new objective from a description
func (g *Generator) GenerateObjective(ctx context.Context, description string) (*objective.Objective, error) {
	// Prepare data for template
	data := map[string]interface{}{
		"Description": description,
	}

	// Render the prompt
	prompt, err := g.promptMgr.RenderPrompt("objective.creation", data)
	if err != nil {
		return nil, fmt.Errorf("error rendering prompt: %w", err)
	}

	// Call the LLM
	response, err := g.client.Complete(ctx, &providers.CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   2048,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("error calling LLM: %w", err)
	}

	// Create a temporary file to parse
	tempFile := filepath.Join(os.TempDir(), "temp_objective.md")
	if err := os.WriteFile(tempFile, []byte(response.Text), 0644); err != nil {
		return nil, fmt.Errorf("error writing temp file: %w", err)
	}
	defer os.Remove(tempFile)

	// Parse the objective from the response
	obj, err := objective.ParseFile(tempFile)
	if err != nil {
		return nil, fmt.Errorf("error parsing objective: %w", err)
	}

	return obj, nil
}

// GenerateAIDocs generates AI documentation based on an objective
func (g *Generator) GenerateAIDocs(ctx context.Context, obj *objective.Objective, additionalContext string) (map[string]string, error) {
	// Prepare data for template
	data := map[string]interface{}{
		"Objective":         obj.Format(), // Method to format objective as markdown
		"AdditionalContext": additionalContext,
	}

	// Render the prompt
	prompt, err := g.promptMgr.RenderPrompt("objective.ai_docs_gen", data)
	if err != nil {
		return nil, fmt.Errorf("error rendering prompt: %w", err)
	}

	// Call the LLM
	response, err := g.client.Complete(ctx, &providers.CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, fmt.Errorf("error calling LLM: %w", err)
	}

	// Parse the response into multiple markdown documents
	docs := parseMultipleMarkdownDocs(response.Text)
	return docs, nil
}

// GenerateSpecs generates technical specifications based on an objective
func (g *Generator) GenerateSpecs(ctx context.Context, obj *objective.Objective, additionalContext string) (map[string]string, error) {
	// Similar implementation to GenerateAIDocs, but using the "objective.specs_gen" prompt
	// ...

	return nil, nil // Placeholder
}

// SuggestImprovements suggests improvements to an objective
func (g *Generator) SuggestImprovements(ctx context.Context, obj *objective.Objective) (string, error) {
	// Similar implementation using "objective.suggestion" prompt
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
