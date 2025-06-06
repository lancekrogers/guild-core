// Package commission provides prompt loading for commission-related prompts
package commission

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed markdown/*.md
var promptFS embed.FS

// LoadPrompts loads all objective-related prompts as templates
func LoadPrompts() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Read all markdown files in the markdown directory
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
		templateName := "commission." + baseName

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