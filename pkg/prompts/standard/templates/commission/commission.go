// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package commission provides prompt loading for commission-related prompts
package commission

import (
	"embed"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

//go:embed markdown/*.md
var promptFS embed.FS

// LoadPrompts loads all commission-related prompts as templates
func LoadPrompts() (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	// Read all markdown files in the markdown directory
	entries, err := fs.ReadDir(promptFS, "markdown")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error reading prompt directory").
			WithComponent("prompts").
			WithOperation("LoadPrompts")
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
			return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "error reading prompt file").
				WithComponent("prompts").
				WithOperation("LoadPrompts").
				WithDetails("file", entry.Name())
		}

		// Create template from markdown content
		tmpl, err := template.New(templateName).Parse(string(content))
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "error parsing template").
				WithComponent("prompts").
				WithOperation("LoadPrompts").
				WithDetails("template_name", templateName)
		}

		templates[templateName] = tmpl
	}

	return templates, nil
}
