// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package layered_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/pkg/prompts/layered"
)

func TestDefaultManager(t *testing.T) {
	ctx := context.Background()

	t.Run("GetSystemPrompt", func(t *testing.T) {
		registry := newTestRegistry()
		formatter := &testFormatter{}
		manager := layered.NewDefaultManager(registry, formatter)

		// Register prompts
		err := registry.RegisterPrompt("manager", "web-app", "You are a web app Guild Master")
		require.NoError(t, err)
		err = registry.RegisterPrompt("manager", "default", "You are a default Guild Master")
		require.NoError(t, err)

		// Test getting specific prompt
		prompt, err := manager.GetSystemPrompt(ctx, "manager", "web-app")
		require.NoError(t, err)
		assert.Equal(t, "You are a web app Guild Master", prompt)

		// Test fallback to default
		prompt, err = manager.GetSystemPrompt(ctx, "manager", "cli-tool")
		require.NoError(t, err)
		assert.Equal(t, "You are a default Guild Master", prompt)

		// Test no prompt found
		_, err = manager.GetSystemPrompt(ctx, "unknown", "unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt not found")
	})

	t.Run("GetTemplate", func(t *testing.T) {
		registry := newTestRegistry()
		formatter := &testFormatter{}
		manager := layered.NewDefaultManager(registry, formatter)

		// Register template
		err := registry.RegisterTemplate("task-format", "Task template content")
		require.NoError(t, err)

		// Test getting template
		template, err := manager.GetTemplate(ctx, "task-format")
		require.NoError(t, err)
		assert.Equal(t, "Task template content", template)

		// Test template not found
		_, err = manager.GetTemplate(ctx, "unknown")
		assert.ErrorIs(t, err, layered.ErrTemplateNotFound)
	})

	t.Run("FormatContext", func(t *testing.T) {
		registry := newTestRegistry()
		formatter := &testFormatter{}
		manager := layered.NewDefaultManager(registry, formatter)

		testCtx := &testContext{
			commissionID:    "test-001",
			commissionTitle: "Test Commission",
		}

		// Test formatting context
		result, err := manager.FormatContext(ctx, testCtx)
		require.NoError(t, err)
		assert.Equal(t, "<context>xml</context>", result)
	})

	t.Run("ListRoles", func(t *testing.T) {
		manager := layered.NewDefaultManager(nil, nil)

		roles, err := manager.ListRoles(ctx)
		require.NoError(t, err)
		assert.Contains(t, roles, "manager")
		assert.Contains(t, roles, "developer")
		assert.Contains(t, roles, "reviewer")
		assert.Contains(t, roles, "architect")
		assert.Contains(t, roles, "tester")
		assert.Contains(t, roles, "documenter")
	})

	t.Run("ListDomains", func(t *testing.T) {
		manager := layered.NewDefaultManager(nil, nil)

		// Test manager domains
		domains, err := manager.ListDomains(ctx, "manager")
		require.NoError(t, err)
		assert.Contains(t, domains, "default")
		assert.Contains(t, domains, "web-app")
		assert.Contains(t, domains, "cli-tool")
		assert.Contains(t, domains, "library")

		// Test developer domains
		domains, err = manager.ListDomains(ctx, "developer")
		require.NoError(t, err)
		assert.Contains(t, domains, "default")
		assert.Contains(t, domains, "backend")
		assert.Contains(t, domains, "frontend")

		// Test unknown role
		domains, err = manager.ListDomains(ctx, "unknown")
		require.NoError(t, err)
		assert.Equal(t, []string{"default"}, domains)
	})

	t.Run("NilRegistry", func(t *testing.T) {
		manager := layered.NewDefaultManager(nil, &testFormatter{})

		// Test that methods handle nil registry gracefully
		_, err := manager.GetSystemPrompt(ctx, "manager", "web-app")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registry not initialized")

		_, err = manager.GetTemplate(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registry not initialized")
	})

	t.Run("NilFormatter", func(t *testing.T) {
		manager := layered.NewDefaultManager(newTestRegistry(), nil)

		testCtx := &testContext{}
		_, err := manager.FormatContext(ctx, testCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "formatter not initialized")
	})

	t.Run("SetRegistry", func(t *testing.T) {
		manager := layered.NewDefaultManager(nil, nil)
		registry := newTestRegistry()

		// Initially should fail
		_, err := manager.GetSystemPrompt(ctx, "manager", "web-app")
		assert.Error(t, err)

		// Set registry
		manager.SetRegistry(registry)
		registry.RegisterPrompt("manager", "web-app", "Test prompt")

		// Now should work
		prompt, err := manager.GetSystemPrompt(ctx, "manager", "web-app")
		require.NoError(t, err)
		assert.Equal(t, "Test prompt", prompt)
	})

	t.Run("SetFormatter", func(t *testing.T) {
		manager := layered.NewDefaultManager(nil, nil)
		formatter := &testFormatter{}

		// Initially should fail
		testCtx := &testContext{}
		_, err := manager.FormatContext(ctx, testCtx)
		assert.Error(t, err)

		// Set formatter
		manager.SetFormatter(formatter)

		// Now should work
		result, err := manager.FormatContext(ctx, testCtx)
		require.NoError(t, err)
		assert.Equal(t, "<context>xml</context>", result)
	})
}

func TestDefaultManagerConcurrency(t *testing.T) {
	// Test that the manager is thread-safe
	registry := newTestRegistry()
	formatter := &testFormatter{}
	manager := layered.NewDefaultManager(registry, formatter)

	// Register some prompts
	registry.RegisterPrompt("manager", "web-app", "Web app prompt")
	registry.RegisterPrompt("developer", "backend", "Backend prompt")
	registry.RegisterTemplate("task-format", "Task template")

	ctx := context.Background()
	done := make(chan bool)

	// Run multiple goroutines accessing the manager
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Perform various operations
			_, err := manager.GetSystemPrompt(ctx, "manager", "web-app")
			assert.NoError(t, err)

			_, err = manager.GetTemplate(ctx, "task-format")
			assert.NoError(t, err)

			_, err = manager.ListRoles(ctx)
			assert.NoError(t, err)

			testCtx := &testContext{}
			_, err = manager.FormatContext(ctx, testCtx)
			assert.NoError(t, err)

			// Modify registry and formatter
			if id%2 == 0 {
				manager.SetRegistry(registry)
				manager.SetFormatter(formatter)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
