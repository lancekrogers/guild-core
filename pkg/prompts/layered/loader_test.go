// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lancekrogers/guild-core/pkg/prompts/layered"
)

func TestLoader(t *testing.T) {
	t.Run("LoadDefaults", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()
		loader := layered.NewLoader(registry)

		err := loader.LoadDefaults()
		require.NoError(t, err)

		// Check manager prompts loaded
		managerPrompt, err := registry.GetPrompt("manager", "default")
		require.NoError(t, err)
		assert.Contains(t, managerPrompt, "Guild Master")

		// Check domain-specific manager prompts
		webAppPrompt, err := registry.GetPrompt("manager", "web-app")
		require.NoError(t, err)
		assert.Contains(t, webAppPrompt, "Guild Master")
		assert.Contains(t, webAppPrompt, "Web Applications")

		// Check developer prompts loaded
		devPrompt, err := registry.GetPrompt("developer", "default")
		require.NoError(t, err)
		assert.Contains(t, devPrompt, "Code Artisan")

		// Check specialized developer prompts
		backendPrompt, err := registry.GetPrompt("developer", "backend")
		require.NoError(t, err)
		assert.Contains(t, backendPrompt, "Backend Specialization")

		// Check reviewer prompts loaded
		reviewPrompt, err := registry.GetPrompt("reviewer", "default")
		require.NoError(t, err)
		assert.Contains(t, reviewPrompt, "Quality Inspector")

		// Check templates loaded
		taskTemplate, err := registry.GetTemplate("task-format")
		require.NoError(t, err)
		assert.Contains(t, taskTemplate, "{{.Category}}")
	})

	t.Run("LoadManagerPrompts", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()
		loader := layered.NewLoader(registry)

		// Use reflection to test private method
		// In production, we'd test this through LoadDefaults
		err := loader.LoadDefaults()
		require.NoError(t, err)

		// Verify all manager domains are loaded
		domains := []string{"default", "web-app", "cli-tool", "library", "microservice"}
		for _, domain := range domains {
			prompt, err := registry.GetPrompt("manager", domain)
			require.NoError(t, err, "Failed to get manager prompt for domain: %s", domain)
			assert.NotEmpty(t, prompt)
			assert.Contains(t, prompt, "Guild Master")

			// Domain-specific prompts should have additional content
			if domain != "default" {
				assert.Contains(t, prompt, "Additional Guidelines")
			}
		}
	})

	t.Run("LoadDeveloperPrompts", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()
		loader := layered.NewLoader(registry)

		err := loader.LoadDefaults()
		require.NoError(t, err)

		// Check base developer prompt
		defaultPrompt, err := registry.GetPrompt("developer", "default")
		require.NoError(t, err)
		assert.Contains(t, defaultPrompt, "Code Artisan")
		assert.Contains(t, defaultPrompt, "Workshop Board")

		// Check specializations
		specializations := map[string]string{
			"backend":   "Backend Specialization",
			"frontend":  "Frontend Specialization",
			"fullstack": "Fullstack Specialization",
		}

		for spec, expectedContent := range specializations {
			prompt, err := registry.GetPrompt("developer", spec)
			require.NoError(t, err, "Failed to get developer prompt for: %s", spec)
			assert.Contains(t, prompt, "Code Artisan")
			assert.Contains(t, prompt, expectedContent)
		}
	})

	t.Run("LoadReviewerPrompts", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()
		loader := layered.NewLoader(registry)

		err := loader.LoadDefaults()
		require.NoError(t, err)

		// Check base reviewer prompt
		defaultPrompt, err := registry.GetPrompt("reviewer", "default")
		require.NoError(t, err)
		assert.Contains(t, defaultPrompt, "Quality Inspector")
		assert.Contains(t, defaultPrompt, "Review Criteria")

		// Check specializations
		specializations := map[string]string{
			"code-quality": "Code Quality Focus",
			"security":     "Security Focus",
			"performance":  "Performance Focus",
		}

		for spec, expectedContent := range specializations {
			prompt, err := registry.GetPrompt("reviewer", spec)
			require.NoError(t, err, "Failed to get reviewer prompt for: %s", spec)
			assert.Contains(t, prompt, "Quality Inspector")
			assert.Contains(t, prompt, expectedContent)
		}
	})

	t.Run("LoadTemplates", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()
		loader := layered.NewLoader(registry)

		err := loader.LoadDefaults()
		require.NoError(t, err)

		// Check all templates are loaded
		templates := []string{
			"task-format",
			"markdown-file",
			"review-comment",
			"task-context",
		}

		for _, templateName := range templates {
			template, err := registry.GetTemplate(templateName)
			require.NoError(t, err, "Failed to get template: %s", templateName)
			assert.NotEmpty(t, template)
			assert.Contains(t, template, "{{") // All templates should have placeholders
		}

		// Check specific template content
		markdownTemplate, _ := registry.GetTemplate("markdown-file")
		assert.Contains(t, markdownTemplate, "## Overview")
		assert.Contains(t, markdownTemplate, "## Requirements")
		assert.Contains(t, markdownTemplate, "## Tasks Generated")

		reviewTemplate, _ := registry.GetTemplate("review-comment")
		assert.Contains(t, reviewTemplate, "Review Decision")
		assert.Contains(t, reviewTemplate, "{{if .Issues}}")

		taskTemplate, _ := registry.GetTemplate("task-context")
		assert.Contains(t, taskTemplate, "Task Context")
		assert.Contains(t, taskTemplate, "Success Criteria")
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with a registry that always returns errors
		errorRegistry := &errorTestRegistry{}
		loader := layered.NewLoader(errorRegistry)

		err := loader.LoadDefaults()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load manager prompts")
	})
}

// errorTestRegistry is a registry that always returns errors
type errorTestRegistry struct{}

func (r *errorTestRegistry) RegisterPrompt(role, domain, prompt string) error {
	return assert.AnError
}

func (r *errorTestRegistry) RegisterTemplate(name, template string) error {
	return assert.AnError
}

func (r *errorTestRegistry) GetPrompt(role, domain string) (string, error) {
	return "", assert.AnError
}

func (r *errorTestRegistry) GetTemplate(name string) (string, error) {
	return "", assert.AnError
}

func TestLoaderCompleteness(t *testing.T) {
	// Ensure all expected roles and domains are covered
	registry := layered.NewMemoryRegistry()
	loader := layered.NewLoader(registry)

	err := loader.LoadDefaults()
	require.NoError(t, err)

	// Define expected coverage
	expectedRoles := map[string][]string{
		"manager":   {"default", "web-app", "cli-tool", "library", "microservice"},
		"developer": {"default", "backend", "frontend", "fullstack"},
		"reviewer":  {"default", "code-quality", "security", "performance"},
	}

	// Verify all expected prompts exist
	for role, domains := range expectedRoles {
		for _, domain := range domains {
			_, err := registry.GetPrompt(role, domain)
			assert.NoError(t, err, "Missing prompt for role=%s, domain=%s", role, domain)
		}
	}

	// Verify templates
	expectedTemplates := []string{
		"task-format",
		"markdown-file",
		"review-comment",
		"task-context",
	}

	for _, template := range expectedTemplates {
		_, err := registry.GetTemplate(template)
		assert.NoError(t, err, "Missing template: %s", template)
	}
}
