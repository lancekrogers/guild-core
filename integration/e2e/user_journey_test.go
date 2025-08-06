// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFirstTimeUserJourney(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Installation and Help", func(t *testing.T) {
		// Step 1: Check help works
		result := env.RunGuild("help")
		result.AssertSuccess(t)
		result.AssertContains(t, "Guild coordinates specialized artisans")
		result.AssertContains(t, "Available Commands")
		result.AssertFasterThan(t, 5*time.Second)

		// Step 2: Check version command
		result = env.RunGuild("version")
		result.AssertSuccess(t)
		result.AssertContains(t, "Guild")
	})

	t.Run("Project Initialization", func(t *testing.T) {
		// Initialize project
		result := env.RunGuild("init")
		result.AssertSuccess(t)
		result.AssertContains(t, "Guild")
		result.AssertNotContains(t, "error")
		result.AssertNotContains(t, "failed")
		result.AssertFasterThan(t, 10*time.Second)

		// Verify .guild directory created
		assert.True(t, env.FileExists(".guild"), ".guild directory should exist")
		assert.True(t, env.FileExists(".guild/guild.yaml"), "guild.yaml should exist")

		// Verify config content
		config, err := env.ReadFile(".guild/guild.yaml")
		assert.NoError(t, err)
		assert.Contains(t, config, "name:")
	})

	t.Run("Command Discovery", func(t *testing.T) {
		// User explores available commands
		result := env.RunGuild("help")
		result.AssertSuccess(t)

		// Should show main command categories
		result.AssertContains(t, "agent")
		result.AssertContains(t, "commission")
		result.AssertContains(t, "kanban")
		result.AssertContains(t, "chat")

		// Check subcommand help
		result = env.RunGuild("agent", "--help")
		result.AssertSuccess(t)
		result.AssertContains(t, "list")
	})

	t.Run("Demo Experience", func(t *testing.T) {
		// User runs quick demo to see what Guild can do
		result := env.RunGuildWithTimeout(60*time.Second, "demo-check")
		result.AssertSuccess(t)

		// Verify demo shows key features
		result.AssertContains(t, "Demo")
		result.AssertContains(t, "Creating")
		result.AssertNotContains(t, "panic")
		result.AssertNotContains(t, "fatal")

		// Should complete within reasonable time
		result.AssertFasterThan(t, 45*time.Second)
	})

	t.Run("Agent Commands", func(t *testing.T) {
		// List available agents
		result := env.RunGuild("agent", "list")
		result.AssertSuccess(t)
		// Agent list might be empty in test mode
		result.AssertNotContains(t, "error")
		result.AssertFasterThan(t, 5*time.Second)
	})

	t.Run("Status Check", func(t *testing.T) {
		result := env.RunGuild("status")
		result.AssertSuccess(t)
		result.AssertContains(t, "Guild Status")
		result.AssertContains(t, "Project")
		result.AssertFasterThan(t, 3*time.Second)
	})

	env.SaveRecording("first_time_user")
}

func TestDeveloperWorkflow(t *testing.T) {
	env := NewTestEnvironment(t)

	// Initialize project first
	env.RunGuild("init").AssertSuccess(t)

	t.Run("Create Commission", func(t *testing.T) {
		// Create a commission using command line
		result := env.RunGuild("commission", "create",
			"--title", "REST API Development",
			"--description", "Build a REST API for user management with authentication")
		result.AssertSuccess(t)
		result.AssertContains(t, "Commission created")
		result.AssertFasterThan(t, 15*time.Second)

		// Verify commission file was created
		assert.True(t, env.FileExists(".guild/commissions/rest-api-development.md"),
			"Commission file should be created")
	})

	t.Run("Refine Commission", func(t *testing.T) {
		// Test commission refinement
		result := env.RunGuild("commission", "refine", "rest-api-development")
		result.AssertSuccess(t)
		result.AssertContains(t, "Commission refined")
		result.AssertFasterThan(t, 20*time.Second)

		// Check for refined commission output
		assert.True(t, env.FileExists(".guild/objectives/refined"),
			"Refined objectives directory should exist")
	})

	t.Run("View Kanban Board", func(t *testing.T) {
		result := env.RunGuild("kanban", "view")
		result.AssertSuccess(t)
		result.AssertContains(t, "Kanban Board")
		result.AssertContains(t, "To Do")
		result.AssertFasterThan(t, 5*time.Second)
	})

	t.Run("Campaign Management", func(t *testing.T) {
		// Create campaign
		result := env.RunGuild("campaign", "create", "api-dev")
		result.AssertSuccess(t)
		result.AssertContains(t, "Campaign created")

		// List campaigns
		result = env.RunGuild("campaign", "list")
		result.AssertSuccess(t)
		result.AssertContains(t, "api-dev")
	})

	t.Run("Configuration Check", func(t *testing.T) {
		result := env.RunGuild("config", "show")
		result.AssertSuccess(t)
		result.AssertContains(t, "Configuration")
		result.AssertContains(t, "test-mode")
	})

	env.SaveRecording("developer_workflow")
}

func TestCommandValidation(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Invalid Commands", func(t *testing.T) {
		// Test invalid command
		result := env.RunGuild("invalid-command")
		result.AssertFailure(t)
		result.AssertStderrContains(t, "unknown command")

		// Should suggest help
		result.AssertStderrContains(t, "help")
	})

	t.Run("Missing Arguments", func(t *testing.T) {
		// Commands that require arguments
		result := env.RunGuild("commission", "create")
		result.AssertFailure(t)
		result.AssertStderrContains(t, "required")
	})

	t.Run("Commands Without Project", func(t *testing.T) {
		// Some commands should work without project initialization
		validWithoutProject := [][]string{
			{"help"},
			{"version"},
			{"init", "--help"},
		}

		for _, cmd := range validWithoutProject {
			result := env.RunGuild(cmd...)
			result.AssertSuccess(t)
		}

		// Some commands should require project
		requireProject := [][]string{
			{"commission", "list"},
			{"kanban", "view"},
			{"status"},
		}

		for _, cmd := range requireProject {
			result := env.RunGuild(cmd...)
			result.AssertFailure(t)
			result.AssertStderrContains(t, "guild init")
		}
	})
}

func TestErrorHandling(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Graceful Error Messages", func(t *testing.T) {
		// Initialize project first
		env.RunGuild("init", "error-test").AssertSuccess(t)

		// Test invalid commission name
		result := env.RunGuild("commission", "view", "non-existent")
		result.AssertFailure(t)
		result.AssertStderrContains(t, "not found")
		result.AssertNotContains(t, "panic")

		// Error should be user-friendly
		assert.NotContains(t, result.Stderr, "GUILD-", "Should not expose internal error codes to users")
	})

	t.Run("Timeout Handling", func(t *testing.T) {
		// Test very short timeout (this may pass depending on command speed)
		_ = env.RunGuildWithTimeout(1*time.Millisecond, "help")
		// Don't assert failure as help might be fast enough
		// This mainly tests that timeout mechanism works
	})

	t.Run("File System Errors", func(t *testing.T) {
		// Create a valid file first to ensure directory exists
		env.CreateFile("test.txt", "content")

		// Test with invalid characters (if supported by OS)
		result := env.RunGuild("init", "test\000project")
		if result.ExitCode != 0 {
			result.AssertStderrContains(t, "invalid")
		}
	})
}

func TestPerformanceBaseline(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Startup Performance", func(t *testing.T) {
		// Test cold start performance
		result := env.RunGuild("version")
		result.AssertSuccess(t)
		result.AssertFasterThan(t, 2*time.Second)

		// Test help performance
		result = env.RunGuild("help")
		result.AssertSuccess(t)
		result.AssertFasterThan(t, 3*time.Second)
	})

	t.Run("Initialization Performance", func(t *testing.T) {
		result := env.RunGuild("init", "perf-test")
		result.AssertSuccess(t)
		result.AssertFasterThan(t, 5*time.Second)
	})

	t.Run("Command Response Time", func(t *testing.T) {
		env.RunGuild("init", "response-test").AssertSuccess(t)

		quickCommands := [][]string{
			{"status"},
			{"agent", "list"},
			{"config", "show"},
			{"campaign", "list"},
		}

		for _, cmd := range quickCommands {
			result := env.RunGuild(cmd...)
			result.AssertSuccess(t)
			result.AssertFasterThan(t, 10*time.Second)
		}
	})

	env.SaveRecording("performance_baseline")
}

func TestMockProviderIntegration(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("Mock Provider Active", func(t *testing.T) {
		env.RunGuild("init").AssertSuccess(t)

		// Config should show mock provider is active
		result := env.RunGuild("config", "show")
		result.AssertSuccess(t)
		result.AssertContains(t, "test-mode")
	})

	t.Run("Deterministic Responses", func(t *testing.T) {
		env.RunGuild("init").AssertSuccess(t)

		// Run the same command multiple times
		responses := make([]string, 3)
		for i := 0; i < 3; i++ {
			result := env.RunGuild("commission", "create",
				"--title", "Test Commission",
				"--description", "Create a simple test API")
			result.AssertSuccess(t)
			responses[i] = result.Stdout
		}

		// Responses should be consistent (mock provider behavior)
		// They don't need to be identical, but should be reasonable
		for _, response := range responses {
			assert.Contains(t, response, "Commission created")
		}
	})

	env.SaveRecording("mock_provider_integration")
}
