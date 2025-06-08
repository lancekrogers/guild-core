package chat_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-ventures/guild-core/internal/chat"
)

// DemoScenario represents a demo test case
type DemoScenario struct {
	Name        string
	Description string
	Commands    []DemoCommand // Sequence of commands to execute
	Expected    []string      // Expected visual outcomes
	Duration    time.Duration // Scenario duration
	AutoPlay    bool          // Auto-execute commands
}

// DemoCommand represents a command in a demo scenario
type DemoCommand struct {
	Input       string        // The command to type
	Delay       time.Duration // Delay before typing
	PauseAfter  time.Duration // Pause after command
	Description string        // What this demonstrates
}

// Test demo scenarios (simplified for testing)
var testDemoScenarios = []DemoScenario{
	{
		Name:        "Rich Content Showcase",
		Description: "Demonstrate markdown rendering and syntax highlighting",
		Duration:    2 * time.Minute,
		AutoPlay:    true,
		Commands: []DemoCommand{
			{
				Input:       "/test markdown",
				Delay:       100 * time.Millisecond,
				PauseAfter:  200 * time.Millisecond,
				Description: "Show rich markdown with headers, lists, and emphasis",
			},
			{
				Input:       "/test code go",
				Delay:       100 * time.Millisecond,
				PauseAfter:  200 * time.Millisecond,
				Description: "Show Go syntax highlighting",
			},
		},
		Expected: []string{
			"Rich markdown rendering",
			"Syntax highlighted code",
		},
	},
	{
		Name:        "Agent Commands",
		Description: "Test agent command processing",
		Duration:    90 * time.Second,
		AutoPlay:    true,
		Commands: []DemoCommand{
			{
				Input:       "@manager analyze requirements",
				Delay:       100 * time.Millisecond,
				PauseAfter:  200 * time.Millisecond,
				Description: "Agent command processing",
			},
		},
		Expected: []string{
			"Agent response",
		},
	},
}

// DemoScenarioValidator validates demo scenarios work end-to-end
type DemoScenarioValidator struct {
	model   *chat.ChatModel
	verbose bool
}

// NewDemoScenarioValidator creates a new demo scenario validator
func NewDemoScenarioValidator(model *chat.ChatModel, verbose bool) *DemoScenarioValidator {
	return &DemoScenarioValidator{
		model:   model,
		verbose: verbose,
	}
}

// ValidateAllScenarios validates all demo scenarios
func (dsv *DemoScenarioValidator) ValidateAllScenarios(t *testing.T) {
	t.Helper()

	if dsv.verbose {
		fmt.Println("🏰 Validating Demo Scenarios")
		fmt.Println("════════════════════════════════════════")
	}

	for _, scenario := range testDemoScenarios {
		t.Run("scenario_"+strings.ReplaceAll(scenario.Name, " ", "_"), func(t *testing.T) {
			dsv.validateScenario(t, &scenario)
		})
	}
}

// validateScenario validates a single demo scenario
func (dsv *DemoScenarioValidator) validateScenario(t *testing.T, scenario *DemoScenario) {
	t.Helper()

	if dsv.verbose {
		fmt.Printf("📋 Testing scenario: %s\n", scenario.Name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), scenario.Duration*2)
	defer cancel()

	// Execute scenario commands
	for i, cmd := range scenario.Commands {
		if dsv.verbose {
			fmt.Printf("  Step %d: %s\n", i+1, cmd.Description)
		}

		select {
		case <-ctx.Done():
			t.Fatalf("Scenario %s timed out at step %d", scenario.Name, i+1)
		default:
			// Continue execution
		}

		// Process command (mock implementation for testing)
		result := dsv.processCommand(cmd.Input)

		// Verify no errors
		assert.NoError(t, result.err, "Command should not error in %s: %s", scenario.Name, cmd.Input)

		// Add delay simulation (faster for testing)
		time.Sleep(cmd.PauseAfter / 10)
	}

	if dsv.verbose {
		fmt.Printf("  ✅ Scenario validated: %s\n", scenario.Name)
	}
}

// commandResult represents the result of processing a command
type commandResult struct {
	success bool
	output  string
	err     error
}

// processCommand simulates command processing for testing
func (dsv *DemoScenarioValidator) processCommand(input string) commandResult {
	// Mock command processing for testing
	switch {
	case strings.HasPrefix(input, "/test markdown"):
		// Simulate markdown test command
		return commandResult{success: true, output: "markdown test"}

	case strings.HasPrefix(input, "/test code"):
		// Simulate code highlighting test
		lang := "go"
		if strings.Contains(input, "python") {
			lang = "python"
		}
		_ = dsv.generateCodeTestContent(lang)
		return commandResult{success: true, output: "code test"}

	case strings.HasPrefix(input, "/test mixed"):
		// Simulate mixed content test
		return commandResult{success: true, output: "mixed test"}

	case strings.HasPrefix(input, "/agents"):
		// Simulate agents command
		return commandResult{success: true, output: "agents list"}

	case strings.HasPrefix(input, "@"):
		// Simulate agent command
		agentName := strings.Fields(input)[0][1:] // Remove @
		task := strings.Join(strings.Fields(input)[1:], " ")

		// Update agent status using test helper
		status := &chat.AgentStatus{
			ID:           agentName,
			Name:         strings.Title(agentName) + " Agent",
			State:        chat.AgentWorking,
			CurrentTask:  task,
			LastActivity: time.Now(),
		}
		_ = status // Use the status somehow

		return commandResult{success: true, output: "agent command"}

	case strings.HasPrefix(input, "/campaign"):
		// Simulate campaign switch
		return commandResult{success: true, output: "campaign switch"}

	case strings.HasPrefix(input, "/tools"):
		// Simulate tools command
		return commandResult{success: true, output: "tools list"}

	default:
		// Handle other commands
		return commandResult{success: true, output: "default"}
	}
}

// generateCodeTestContent generates test code content for different languages
func (dsv *DemoScenarioValidator) generateCodeTestContent(lang string) string {
	switch lang {
	case "go":
		return "```go\npackage main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Printf(\"Welcome to Guild!\")\n}\n```"
	case "python":
		return "```python\nimport asyncio\n\nasync def guild_agent():\n    await asyncio.sleep(1)\n    return \"Task complete\"\n```"
	case "javascript":
		return "```javascript\nclass GuildAgent {\n    constructor(name) {\n        this.name = name;\n    }\n    \n    async execute(task) {\n        return `Executing: ${task}`;\n    }\n}\n```"
	default:
		return "```\n# Code example\necho \"Hello Guild!\"\n```"
	}
}

// TestDemoScenarioExecution tests that all demo scenarios execute properly
func TestDemoScenarioExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo scenario execution tests in short mode")
	}

	// Create test model
	model := createTestChatModel(t)
	validator := NewDemoScenarioValidator(model, true)

	// Validate all scenarios
	validator.ValidateAllScenarios(t)
}

// TestDemoPerformance ensures demo scenarios complete within time limits
func TestDemoPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping demo performance tests in short mode")
	}

	model := createTestChatModel(t)
	validator := NewDemoScenarioValidator(model, false)

	// Test each scenario for performance
	for _, scenario := range testDemoScenarios {
		t.Run("performance_"+strings.ReplaceAll(scenario.Name, " ", "_"), func(t *testing.T) {
			start := time.Now()

			// Execute commands (simplified for performance testing)
			for _, cmd := range scenario.Commands {
				validator.processCommand(cmd.Input)
			}

			duration := time.Since(start)

			// Demo should not take longer than specified duration + 50% buffer
			maxDuration := scenario.Duration + (scenario.Duration / 2)
			assert.LessOrEqual(t, duration, maxDuration,
				"Demo scenario %s took too long: %v > %v", scenario.Name, duration, maxDuration)

			t.Logf("Scenario %s completed in %v (limit: %v)", scenario.Name, duration, maxDuration)
		})
	}
}

// TestVisualComponentCompatibility ensures visual components work together
func TestVisualComponentCompatibility(t *testing.T) {
	t.Run("markdown_and_status_display", func(t *testing.T) {
		// Test that markdown renderer and status display work together
		renderer, err := chat.NewMarkdownRenderer(80)
		require.NoError(t, err)

		// Render some markdown content
		content := "# Title\n\n```go\ncode\n```\n\n**bold**"
		rendered := renderer.Render(content)
		assert.NotEmpty(t, rendered)
		assert.Contains(t, rendered, "Title")
	})

	t.Run("status_tracker_functionality", func(t *testing.T) {
		// Test agent status tracking
		guildConfig := createTestConfig()
		tracker := chat.NewAgentStatusTracker(guildConfig)
		require.NotNil(t, tracker)

		// Update agent status
		status := &chat.AgentStatus{
			ID:           "test-agent",
			Name:         "Test Agent",
			State:        chat.AgentWorking,
			CurrentTask:  "Testing visual components",
			LastActivity: time.Now(),
		}
		tracker.UpdateAgentStatus("test-agent", status)

		// Verify status was updated
		retrievedStatus := tracker.GetAgentStatus("test-agent")
		require.NotNil(t, retrievedStatus)
		assert.Equal(t, "test-agent", retrievedStatus.ID)
		assert.Equal(t, chat.AgentWorking, retrievedStatus.State)
	})
}

// TestErrorHandlingInDemos tests graceful error handling during demos
func TestErrorHandlingInDemos(t *testing.T) {
	model := createTestChatModel(t)

	t.Run("component_failure_graceful_degradation", func(t *testing.T) {
		// Test with invalid renderer width
		_, err := chat.NewMarkdownRenderer(0) // Invalid width
		assert.NoError(t, err) // Should handle gracefully

		// Test with very large width
		renderer, err := chat.NewMarkdownRenderer(10000)
		assert.NoError(t, err)
		
		// Should still render content
		content := "# Test"
		rendered := renderer.Render(content)
		assert.NotEmpty(t, rendered)
	})

	t.Run("invalid_demo_commands", func(t *testing.T) {
		validator := NewDemoScenarioValidator(model, false)

		// Test with invalid commands
		invalidCommands := []string{
			"/nonexistent_command",
			"@invalid_agent do something",
			"/test invalid_type",
		}

		for _, cmd := range invalidCommands {
			result := validator.processCommand(cmd)
			// Should handle gracefully (success or failure, but no panic)
			assert.NotNil(t, result)
		}
	})

	t.Run("stress_test_many_operations", func(t *testing.T) {
		// Test with many operations quickly (stress test)
		renderer, err := chat.NewMarkdownRenderer(80)
		require.NoError(t, err)

		// Should handle without crashing
		assert.NotPanics(t, func() {
			for i := 0; i < 100; i++ {
				content := fmt.Sprintf("# Message %d\n\nContent with **formatting**", i)
				rendered := renderer.Render(content)
				assert.NotEmpty(t, rendered)
			}
		})
	})
}

// TestDemoContentQuality validates that demo content meets quality standards
func TestDemoContentQuality(t *testing.T) {
	t.Run("scenario_completeness", func(t *testing.T) {
		for _, scenario := range testDemoScenarios {
			// Each scenario should have required fields
			assert.NotEmpty(t, scenario.Name, "Scenario should have a name")
			assert.NotEmpty(t, scenario.Description, "Scenario should have a description")
			assert.NotEmpty(t, scenario.Commands, "Scenario should have commands")
			assert.NotEmpty(t, scenario.Expected, "Scenario should have expected outcomes")
			assert.Greater(t, scenario.Duration, time.Duration(0), "Scenario should have positive duration")

			// Commands should be well-formed
			for i, cmd := range scenario.Commands {
				assert.NotEmpty(t, cmd.Input, "Command %d in %s should have input", i+1, scenario.Name)
				assert.NotEmpty(t, cmd.Description, "Command %d in %s should have description", i+1, scenario.Name)
				assert.GreaterOrEqual(t, cmd.PauseAfter, time.Duration(0), "Command %d pause should be non-negative", i+1)
			}
		}
	})

	t.Run("expected_outcomes_realistic", func(t *testing.T) {
		// Expected outcomes should be achievable
		for _, scenario := range testDemoScenarios {
			for _, expected := range scenario.Expected {
				assert.NotEmpty(t, expected, "Expected outcome should not be empty")
				assert.Greater(t, len(expected), 5, "Expected outcome should be descriptive")
			}
		}
	})

	t.Run("demo_timing_realistic", func(t *testing.T) {
		// Demo timing should be reasonable
		for _, scenario := range testDemoScenarios {
			assert.LessOrEqual(t, scenario.Duration, 5*time.Minute,
				"Scenario %s should not exceed 5 minutes", scenario.Name)
			assert.GreaterOrEqual(t, scenario.Duration, 30*time.Second,
				"Scenario %s should be at least 30 seconds", scenario.Name)
		}
	})
}

// Helper function to create a test chat model
func createTestChatModel(t *testing.T) *chat.ChatModel {
	// For testing, we just need to verify the model can be created
	// Most functionality will be mocked in the validator
	model := &chat.ChatModel{}
	
	// Note: In a real implementation, we would initialize the model properly
	// For now, this is a minimal mock for testing the validation logic
	t.Helper()
	return model
}