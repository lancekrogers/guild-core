package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed markdown/*.md
var promptFS embed.FS

// PromptManager handles loading and rendering prompt templates
type PromptManager struct {
	templates map[string]*template.Template
}

// NewPromptManager creates a new prompt manager
func NewPromptManager() (*PromptManager, error) {
	templates, err := loadPrompts()
	if err != nil {
		return nil, err
	}

	return &PromptManager{
		templates: templates,
	}, nil
}

// RenderPrompt renders a prompt template with the given data
func (pm *PromptManager) RenderPrompt(name string, data interface{}) (string, error) {
	tmpl, exists := pm.templates[name]
	if !exists {
		return "", fmt.Errorf("prompt template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

// loadPrompts loads all objective-related prompts as templates
func loadPrompts() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Read all markdown files
	entries, err := fs.ReadDir(promptFS, "markdown")
	if err != nil {
		return nil, fmt.Errorf("error reading prompt directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Get filename without extension as the template name
		baseName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		templateName := "objective." + baseName

		// Read the markdown file
		content, err := fs.ReadFile(promptFS, filepath.Join("markdown", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("error reading prompt file %s: %w", entry.Name(), err)
		}

		// Create template from markdown content
		tmpl, err := template.New(templateName).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("error parsing template %s: %w", templateName, err)
		}

		templates[templateName] = tmpl
	}

	return templates, nil
}