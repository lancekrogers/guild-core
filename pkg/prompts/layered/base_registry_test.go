// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
)

func TestMemoryRegistry(t *testing.T) {
	t.Run("RegisterAndGetPrompt", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Test successful registration
		err := registry.RegisterPrompt("manager", "web-app", "Web app manager prompt")
		require.NoError(t, err)

		// Test retrieval
		prompt, err := registry.GetPrompt("manager", "web-app")
		require.NoError(t, err)
		assert.Equal(t, "Web app manager prompt", prompt)

		// Test non-existent prompt
		_, err = registry.GetPrompt("unknown", "unknown")
		assert.ErrorIs(t, err, layered.ErrPromptNotFound)
	})

	t.Run("RegisterPromptValidation", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Test empty role
		err := registry.RegisterPrompt("", "domain", "prompt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "role cannot be empty")

		// Test empty domain
		err = registry.RegisterPrompt("role", "", "prompt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "domain cannot be empty")

		// Test empty prompt
		err = registry.RegisterPrompt("role", "domain", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt cannot be empty")
	})

	t.Run("RegisterAndGetTemplate", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Test successful registration
		err := registry.RegisterTemplate("task-format", "Task format template")
		require.NoError(t, err)

		// Test retrieval
		template, err := registry.GetTemplate("task-format")
		require.NoError(t, err)
		assert.Equal(t, "Task format template", template)

		// Test non-existent template
		_, err = registry.GetTemplate("unknown")
		assert.ErrorIs(t, err, layered.ErrTemplateNotFound)
	})

	t.Run("RegisterTemplateValidation", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Test empty name
		err := registry.RegisterTemplate("", "template")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template name cannot be empty")

		// Test empty template
		err = registry.RegisterTemplate("name", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template cannot be empty")
	})

	t.Run("ListPrompts", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Register multiple prompts
		registry.RegisterPrompt("manager", "web-app", "prompt1")
		registry.RegisterPrompt("manager", "cli-tool", "prompt2")
		registry.RegisterPrompt("developer", "backend", "prompt3")

		// List prompts
		keys := registry.ListPrompts()
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "manager:web-app")
		assert.Contains(t, keys, "manager:cli-tool")
		assert.Contains(t, keys, "developer:backend")
	})

	t.Run("ListTemplates", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Register multiple templates
		registry.RegisterTemplate("task-format", "template1")
		registry.RegisterTemplate("context-format", "template2")
		registry.RegisterTemplate("output-format", "template3")

		// List templates
		names := registry.ListTemplates()
		assert.Len(t, names, 3)
		assert.Contains(t, names, "task-format")
		assert.Contains(t, names, "context-format")
		assert.Contains(t, names, "output-format")
	})

	t.Run("Clear", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Add some data
		registry.RegisterPrompt("manager", "web-app", "prompt")
		registry.RegisterTemplate("task-format", "template")

		// Verify data exists
		_, err := registry.GetPrompt("manager", "web-app")
		require.NoError(t, err)
		_, err = registry.GetTemplate("task-format")
		require.NoError(t, err)

		// Clear registry
		registry.Clear()

		// Verify data is gone
		_, err = registry.GetPrompt("manager", "web-app")
		assert.ErrorIs(t, err, layered.ErrPromptNotFound)
		_, err = registry.GetTemplate("task-format")
		assert.ErrorIs(t, err, layered.ErrTemplateNotFound)

		// Verify lists are empty
		assert.Empty(t, registry.ListPrompts())
		assert.Empty(t, registry.ListTemplates())
	})

	t.Run("OverwritePrompt", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Register initial prompt
		err := registry.RegisterPrompt("manager", "web-app", "Original prompt")
		require.NoError(t, err)

		// Overwrite with new prompt
		err = registry.RegisterPrompt("manager", "web-app", "Updated prompt")
		require.NoError(t, err)

		// Verify updated value
		prompt, err := registry.GetPrompt("manager", "web-app")
		require.NoError(t, err)
		assert.Equal(t, "Updated prompt", prompt)
	})

	t.Run("OverwriteTemplate", func(t *testing.T) {
		registry := layered.NewMemoryRegistry()

		// Register initial template
		err := registry.RegisterTemplate("task-format", "Original template")
		require.NoError(t, err)

		// Overwrite with new template
		err = registry.RegisterTemplate("task-format", "Updated template")
		require.NoError(t, err)

		// Verify updated value
		template, err := registry.GetTemplate("task-format")
		require.NoError(t, err)
		assert.Equal(t, "Updated template", template)
	})
}

func TestMemoryRegistryConcurrency(t *testing.T) {
	registry := layered.NewMemoryRegistry()

	// Number of concurrent operations
	numGoroutines := 100
	numOperations := 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Run concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Register prompts
				role := fmt.Sprintf("role%d", id%5)
				domain := fmt.Sprintf("domain%d", j)
				prompt := fmt.Sprintf("prompt-%d-%d", id, j)
				err := registry.RegisterPrompt(role, domain, prompt)
				assert.NoError(t, err)

				// Register templates
				templateName := fmt.Sprintf("template%d-%d", id, j)
				template := fmt.Sprintf("template content %d-%d", id, j)
				err = registry.RegisterTemplate(templateName, template)
				assert.NoError(t, err)

				// Read operations
				_, _ = registry.GetPrompt(role, domain)
				_, _ = registry.GetTemplate(templateName)

				// List operations
				_ = registry.ListPrompts()
				_ = registry.ListTemplates()

				// Occasionally clear (only first few goroutines)
				if id < 5 && j == numOperations-1 {
					registry.Clear()
				}
			}
		}(i)
	}

	wg.Wait()

	// The registry should still be functional after concurrent access
	err := registry.RegisterPrompt("final", "test", "Final prompt")
	assert.NoError(t, err)

	prompt, err := registry.GetPrompt("final", "test")
	assert.NoError(t, err)
	assert.Equal(t, "Final prompt", prompt)
}

// Benchmark tests
func BenchmarkRegistryRegisterPrompt(b *testing.B) {
	registry := layered.NewMemoryRegistry()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		registry.RegisterPrompt("manager", "web-app", "Benchmark prompt")
	}
}

func BenchmarkRegistryGetPrompt(b *testing.B) {
	registry := layered.NewMemoryRegistry()
	registry.RegisterPrompt("manager", "web-app", "Benchmark prompt")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = registry.GetPrompt("manager", "web-app")
	}
}

func BenchmarkRegistryConcurrentRead(b *testing.B) {
	registry := layered.NewMemoryRegistry()
	// Pre-populate with prompts
	for i := 0; i < 100; i++ {
		registry.RegisterPrompt(fmt.Sprintf("role%d", i), "domain", "prompt")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			role := fmt.Sprintf("role%d", i%100)
			_, _ = registry.GetPrompt(role, "domain")
			i++
		}
	})
}
