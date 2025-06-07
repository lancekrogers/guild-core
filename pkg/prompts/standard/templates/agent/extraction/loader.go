package extraction

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed *.md
var promptFiles embed.FS

// LoadPrompt loads a prompt template by name
func LoadPrompt(name string) (string, error) {
	// Ensure .md extension
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}

	content, err := promptFiles.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("failed to load prompt %s: %w", name, err)
	}

	// Skip the YAML frontmatter
	contentStr := string(content)
	if strings.HasPrefix(contentStr, "---") {
		// Find the end of frontmatter
		endIndex := strings.Index(contentStr[3:], "---")
		if endIndex > 0 {
			// Skip past the frontmatter
			contentStr = strings.TrimSpace(contentStr[endIndex+6:])
		}
	}

	return contentStr, nil
}

// LoadPromptByPath loads a prompt template by its base name
func LoadPromptByPath(path string) (string, error) {
	// Extract just the filename
	name := filepath.Base(path)
	return LoadPrompt(name)
}

// GetAvailablePrompts returns a list of available prompt templates
func GetAvailablePrompts() ([]string, error) {
	entries, err := promptFiles.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt directory: %w", err)
	}

	var prompts []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			prompts = append(prompts, strings.TrimSuffix(entry.Name(), ".md"))
		}
	}

	return prompts, nil
}
