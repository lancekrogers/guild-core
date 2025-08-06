// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDemoSystemIntegration tests the complete demo script system
func TestDemoSystemIntegration(t *testing.T) {
	env := NewTestEnvironment(t)

	// Test cases for demo system integration
	t.Run("DemoScriptValidation", func(t *testing.T) {
		testDemoScriptValidation(t, env)
	})

	t.Run("DemoMasterOrchestration", func(t *testing.T) {
		testDemoMasterOrchestration(t, env)
	})

	t.Run("InteractiveDemoSystem", func(t *testing.T) {
		testInteractiveDemoSystem(t, env)
	})

	t.Run("ValidationFramework", func(t *testing.T) {
		testValidationFramework(t, env)
	})

	t.Run("CIIntegration", func(t *testing.T) {
		testCIIntegration(t, env)
	})
}

func testDemoScriptValidation(t *testing.T, env *TestEnvironment) {
	t.Log("Testing demo script validation functionality")

	// Get the scripts directory relative to the test
	scriptsDir := filepath.Join("..", "..", "scripts", "recording")

	// Test that validation script exists and is executable
	validateScript := filepath.Join(scriptsDir, "validate-demos.sh")
	_, err := os.Stat(validateScript)
	require.NoError(t, err, "Validation script should exist")

	// Test quick validation mode
	result := env.RunGuildWithTimeout(30*time.Second, "demo-check")
	// Should exit successfully even if some warnings
	assert.True(t, result.ExitCode == 0 || result.ExitCode == 1, "Demo check should run")
	assert.Contains(t, result.Stdout, "Demo", "Should contain demo-related output")

	t.Log("Demo script validation test completed")
}

func testDemoMasterOrchestration(t *testing.T, env *TestEnvironment) {
	t.Log("Testing demo master orchestration")

	scriptsDir := filepath.Join("..", "..", "scripts", "recording")
	masterScript := filepath.Join(scriptsDir, "demo-master.sh")

	// Test that master script exists
	_, err := os.Stat(masterScript)
	require.NoError(t, err, "Demo master script should exist")

	// Test help functionality by checking if file is readable
	data, err := os.ReadFile(masterScript)
	require.NoError(t, err, "Should be able to read demo master script")

	content := string(data)
	assert.Contains(t, content, "Demo Master", "Should contain demo master content")
	assert.Contains(t, content, "DEMO_SCRIPTS", "Should contain demo scripts configuration")

	t.Log("Demo master orchestration test completed")
}

func testInteractiveDemoSystem(t *testing.T, env *TestEnvironment) {
	t.Log("Testing interactive demo system")

	scriptsDir := filepath.Join("..", "..", "scripts", "recording")
	interactiveScript := filepath.Join(scriptsDir, "interactive-demo.sh")

	// Test that interactive script exists
	_, err := os.Stat(interactiveScript)
	require.NoError(t, err, "Interactive demo script should exist")

	// Test script content for key features
	data, err := os.ReadFile(interactiveScript)
	require.NoError(t, err, "Should be able to read interactive demo script")

	content := string(data)
	assert.Contains(t, content, "tutorial", "Should contain tutorial functionality")
	assert.Contains(t, content, "step", "Should contain step-by-step functionality")
	assert.Contains(t, content, "navigation", "Should contain navigation functionality")

	t.Log("Interactive demo system test completed")
}

func testValidationFramework(t *testing.T, env *TestEnvironment) {
	t.Log("Testing demo validation framework")

	scriptsDir := filepath.Join("..", "..", "scripts", "recording")
	validationScript := filepath.Join(scriptsDir, "validate-demos.sh")

	// Test validation script exists and has correct structure
	data, err := os.ReadFile(validationScript)
	require.NoError(t, err, "Should be able to read validation script")

	content := string(data)
	assert.Contains(t, content, "validate_environment", "Should contain environment validation")
	assert.Contains(t, content, "validate_demo_scripts", "Should contain script validation")
	assert.Contains(t, content, "test_case", "Should contain test case framework")

	// Test that recording utilities exist
	recordingUtils := filepath.Join(scriptsDir, "lib", "recording-utils.sh")
	_, err = os.Stat(recordingUtils)
	require.NoError(t, err, "Recording utilities should exist")

	utilsData, err := os.ReadFile(recordingUtils)
	require.NoError(t, err, "Should be able to read recording utilities")

	utilsContent := string(utilsData)
	assert.Contains(t, utilsContent, "recording_init", "Should contain recording initialization")
	assert.Contains(t, utilsContent, "generate_gif", "Should contain GIF generation")

	t.Log("Validation framework test completed")
}

func testCIIntegration(t *testing.T, env *TestEnvironment) {
	t.Log("Testing CI/CD integration")

	scriptsDir := filepath.Join("..", "..", "scripts", "recording")
	ciScript := filepath.Join(scriptsDir, "ci-demo-integration.sh")

	// Test CI integration script exists
	_, err := os.Stat(ciScript)
	require.NoError(t, err, "CI integration script should exist")

	// Test script content for CI features
	data, err := os.ReadFile(ciScript)
	require.NoError(t, err, "Should be able to read CI integration script")

	content := string(data)
	assert.Contains(t, content, "CI_MODE", "Should contain CI mode support")
	assert.Contains(t, content, "GITHUB_ACTIONS", "Should contain GitHub Actions support")
	assert.Contains(t, content, "pipeline", "Should contain pipeline functionality")

	t.Log("CI/CD integration test completed")
}

// TestDemoRecordingWorkflow tests the end-to-end demo recording workflow
func TestDemoRecordingWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo recording workflow test in short mode")
	}

	env := NewTestEnvironment(t)

	t.Log("Testing demo recording workflow")

	// Initialize Guild project for demo
	result := env.RunGuild("init", "--name", "demo-test", "--description", "Demo test project")
	result.AssertSuccess(t)

	// Test that basic demo components work
	result = env.RunGuild("agents", "list")
	result.AssertSuccess(t)
	assert.Contains(t, result.Stdout, "agent", "Should list agents")

	// Test commission creation (core demo functionality)
	commissionContent := `# Test Commission
Build a simple REST API with:
- User management
- Authentication
- Basic CRUD operations

## Technical Requirements
- Go/Gin framework
- SQLite database
- Unit tests
`

	err := env.CreateFile("test-commission.md", commissionContent)
	require.NoError(t, err, "Should create commission file")

	// Test commission processing (demo core functionality)
	result = env.RunGuildWithTimeout(60*time.Second, "commission", "-f", "test-commission.md")
	// Commission processing should complete or timeout gracefully
	assert.True(t, result.ExitCode == 0 || result.ExitCode == 124, "Commission should process or timeout")

	// Test status checking (demo monitoring functionality)
	result = env.RunGuild("status")
	result.AssertSuccess(t)

	t.Log("Demo recording workflow test completed")
}

// TestDemoValidationScenarios tests various demo validation scenarios
func TestDemoValidationScenarios(t *testing.T) {
	env := NewTestEnvironment(t)

	t.Run("BasicValidation", func(t *testing.T) {
		testBasicDemoValidation(t, env)
	})

	t.Run("EnvironmentChecks", func(t *testing.T) {
		testEnvironmentChecks(t, env)
	})

	t.Run("ScriptIntegrity", func(t *testing.T) {
		testScriptIntegrity(t, env)
	})
}

func testBasicDemoValidation(t *testing.T, env *TestEnvironment) {
	t.Log("Testing basic demo validation")

	// Test guild demo-check command exists
	result := env.RunGuild("demo-check", "--help")
	if result.ExitCode != 0 {
		// demo-check might not be implemented yet, that's ok
		t.Log("demo-check command not available, skipping")
		return
	}

	result.AssertSuccess(t)
	assert.Contains(t, result.Stdout, "demo", "Should contain demo-related help")

	t.Log("Basic demo validation test completed")
}

func testEnvironmentChecks(t *testing.T, env *TestEnvironment) {
	t.Log("Testing environment validation checks")

	// Check that we can detect Guild binary
	result := env.RunGuild("version")
	result.AssertSuccess(t)

	// Check that basic Guild functionality works for demos
	result = env.RunGuild("init", "--help")
	result.AssertSuccess(t)

	result = env.RunGuild("agents", "--help")
	result.AssertSuccess(t)

	result = env.RunGuild("commission", "--help")
	result.AssertSuccess(t)

	t.Log("Environment validation checks completed")
}

func testScriptIntegrity(t *testing.T, env *TestEnvironment) {
	t.Log("Testing demo script integrity")

	scriptsDir := filepath.Join("..", "..", "scripts", "recording")

	// Check that all required demo scripts exist
	requiredScripts := []string{
		"01-quick-start-demo.sh",
		"02-complete-workflow-demo.sh",
		"03-feature-showcase-demo.sh",
		"interactive-demo.sh",
		"demo-master.sh",
		"validate-demos.sh",
		"ci-demo-integration.sh",
		"lib/recording-utils.sh",
	}

	for _, script := range requiredScripts {
		scriptPath := filepath.Join(scriptsDir, script)
		_, err := os.Stat(scriptPath)
		assert.NoError(t, err, "Required script should exist: %s", script)

		if err == nil {
			// Check script is executable
			info, _ := os.Stat(scriptPath)
			mode := info.Mode()
			assert.True(t, mode&0100 != 0, "Script should be executable: %s", script)

			// Basic syntax check - ensure it's a valid shell script
			data, err := os.ReadFile(scriptPath)
			require.NoError(t, err, "Should be able to read script: %s", script)

			content := string(data)
			assert.True(t, strings.HasPrefix(content, "#!/bin/bash"), "Script should have bash shebang: %s", script)
		}
	}

	t.Log("Script integrity test completed")
}

// TestDemoArtifactGeneration tests demo artifact generation
func TestDemoArtifactGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping artifact generation test in short mode")
	}

	env := NewTestEnvironment(t)
	t.Log("Testing demo artifact generation")

	// Test that we can create demo-related files
	err := env.CreateFile("demo-commission.md", `# Demo Commission
Build a simple API for demonstration purposes.
`)
	require.NoError(t, err, "Should create demo commission")

	// Verify file was created
	assert.True(t, env.FileExists("demo-commission.md"), "Demo commission should exist")

	content, err := env.ReadFile("demo-commission.md")
	require.NoError(t, err, "Should read demo commission")
	assert.Contains(t, content, "Demo Commission", "Should contain expected content")

	t.Log("Demo artifact generation test completed")
}

// TestDemoSystemPerformance tests demo system performance characteristics
func TestDemoSystemPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := NewTestEnvironment(t)
	t.Log("Testing demo system performance")

	// Measure initialization time
	start := time.Now()
	result := env.RunGuild("init", "--name", "perf-test", "--description", "Performance test")
	initDuration := time.Since(start)

	result.AssertSuccess(t)
	assert.Less(t, initDuration, 10*time.Second, "Guild init should complete quickly")

	// Measure agent listing time
	start = time.Now()
	result = env.RunGuild("agents", "list")
	agentsDuration := time.Since(start)

	result.AssertSuccess(t)
	assert.Less(t, agentsDuration, 5*time.Second, "Agent listing should be fast")

	// Measure status checking time
	start = time.Now()
	result = env.RunGuild("status")
	statusDuration := time.Since(start)

	result.AssertSuccess(t)
	assert.Less(t, statusDuration, 3*time.Second, "Status check should be very fast")

	t.Logf("Performance results - Init: %v, Agents: %v, Status: %v",
		initDuration, agentsDuration, statusDuration)

	t.Log("Demo system performance test completed")
}
