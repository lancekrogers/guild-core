package main

import (
	"fmt"
	"time"
)

// DemoScenario represents a demo test case
type DemoScenario struct {
	Name         string
	Description  string
	Commands     []DemoCommand // Sequence of commands to execute
	Expected     []string      // Expected visual outcomes
	Duration     time.Duration // Scenario duration
	AutoPlay     bool         // Auto-execute commands
}

// DemoCommand represents a command in a demo scenario
type DemoCommand struct {
	Input       string        // The command to type
	Delay       time.Duration // Delay before typing
	PauseAfter  time.Duration // Pause after command
	Description string        // What this demonstrates
}

// DemoScenarios contains all demo scenarios to showcase Guild's visual superiority
var DemoScenarios = []DemoScenario{
	{
		Name:        "Rich Content Showcase",
		Description: "Demonstrate markdown rendering and syntax highlighting",
		Duration:    2 * time.Minute,
		AutoPlay:    true,
		Commands: []DemoCommand{
			{
				Input:       "/test markdown",
				Delay:       1 * time.Second,
				PauseAfter:  3 * time.Second,
				Description: "Show rich markdown with headers, lists, and emphasis",
			},
			{
				Input:       "/test code go",
				Delay:       500 * time.Millisecond,
				PauseAfter:  3 * time.Second,
				Description: "Show Go syntax highlighting with medieval agent example",
			},
			{
				Input:       "/test code python",
				Delay:       500 * time.Millisecond,
				PauseAfter:  3 * time.Second,
				Description: "Show Python syntax highlighting with async agent",
			},
			{
				Input:       "/test mixed",
				Delay:       500 * time.Millisecond,
				PauseAfter:  4 * time.Second,
				Description: "Show combined markdown and code content",
			},
		},
		Expected: []string{
			"Rich markdown rendering with proper formatting",
			"Syntax highlighted code blocks",
			"Professional visual presentation",
			"Clear visual hierarchy",
		},
	},
	{
		Name:        "Command Experience",
		Description: "Demonstrate auto-completion and history navigation",
		Duration:    90 * time.Second,
		AutoPlay:    false, // Manual interaction needed
		Commands: []DemoCommand{
			{
				Input:       "/he",
				Delay:       1 * time.Second,
				PauseAfter:  1 * time.Second,
				Description: "Type /he then press TAB to complete to /help",
			},
			{
				Input:       "@man",
				Delay:       500 * time.Millisecond,
				PauseAfter:  1 * time.Second,
				Description: "Type @man then TAB to complete to @manager",
			},
			{
				Input:       "/prompt ",
				Delay:       500 * time.Millisecond,
				PauseAfter:  2 * time.Second,
				Description: "Show subcommand completions after space",
			},
			{
				Input:       "↑",
				Delay:       500 * time.Millisecond,
				PauseAfter:  1 * time.Second,
				Description: "Press UP arrow to show previous command",
			},
			{
				Input:       "ctrl+r test",
				Delay:       500 * time.Millisecond,
				PauseAfter:  2 * time.Second,
				Description: "Search command history for 'test'",
			},
		},
		Expected: []string{
			"Tab completion popup with suggestions",
			"Intelligent agent name completion",
			"Command history navigation",
			"Fuzzy search through history",
			"Professional completion interface",
		},
	},
	{
		Name:        "Multi-Agent Coordination",
		Description: "Show multiple agents working together with visual status",
		Duration:    3 * time.Minute,
		AutoPlay:    true,
		Commands: []DemoCommand{
			{
				Input:       "/agents",
				Delay:       1 * time.Second,
				PauseAfter:  3 * time.Second,
				Description: "Show all available agents with status indicators",
			},
			{
				Input:       "@manager analyze this e-commerce codebase for improvements",
				Delay:       500 * time.Millisecond,
				PauseAfter:  2 * time.Second,
				Description: "Manager agent starts analysis (status: 🤔 Thinking)",
			},
			{
				Input:       "@developer implement user authentication feature",
				Delay:       2 * time.Second,
				PauseAfter:  2 * time.Second,
				Description: "Developer agent starts work (status: ⚙️ Working)",
			},
			{
				Input:       "@reviewer check the authentication implementation",
				Delay:       2 * time.Second,
				PauseAfter:  3 * time.Second,
				Description: "Reviewer agent performs code review (status: 🔍 Reviewing)",
			},
			{
				Input:       "ctrl+s",
				Delay:       1 * time.Second,
				PauseAfter:  3 * time.Second,
				Description: "Show real-time agent status panel",
			},
		},
		Expected: []string{
			"Visual agent status indicators",
			"Real-time status updates",
			"Multi-agent coordination display",
			"Professional agent management UI",
			"Clear task assignment visibility",
		},
	},
	{
		Name:        "Professional Polish Comparison",
		Description: "Complete Guild experience showcasing superiority over competitors",
		Duration:    4 * time.Minute,
		AutoPlay:    true,
		Commands: []DemoCommand{
			{
				Input:       "/campaign e-commerce",
				Delay:       1 * time.Second,
				PauseAfter:  2 * time.Second,
				Description: "Switch to e-commerce demo campaign",
			},
			{
				Input:       "@all Let's build a complete e-commerce REST API with user management, product catalog, and order processing",
				Delay:       1 * time.Second,
				PauseAfter:  3 * time.Second,
				Description: "Broadcast complex task to all agents",
			},
			{
				Input:       "/tools status",
				Delay:       3 * time.Second,
				PauseAfter:  2 * time.Second,
				Description: "Show active tool executions with progress bars",
			},
			{
				Input:       "ctrl+g",
				Delay:       2 * time.Second,
				PauseAfter:  2 * time.Second,
				Description: "Switch to global view showing all agent activity",
			},
			{
				Input:       "/test mixed",
				Delay:       1 * time.Second,
				PauseAfter:  3 * time.Second,
				Description: "Show rich content rendering during active development",
			},
		},
		Expected: []string{
			"Smooth multi-agent orchestration",
			"Rich visual feedback throughout",
			"Professional tool execution display",
			"Superior UX compared to plain-text tools",
			"Immediate 'wow' factor on first use",
		},
	},
}

// DemoRunner executes demo scenarios for recording or live demonstration
type DemoRunner struct {
	model    *ChatModel
	scenario *DemoScenario
	paused   bool
}

// NewDemoRunner creates a new demo runner
func NewDemoRunner(model *ChatModel) *DemoRunner {
	return &DemoRunner{
		model:  model,
		paused: false,
	}
}

// RunScenario executes a specific demo scenario
func (dr *DemoRunner) RunScenario(scenario *DemoScenario) error {
	dr.scenario = scenario

	// Show scenario introduction
	introMsg := fmt.Sprintf(`
🏰 Starting Demo: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
%s

Duration: %v | Mode: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`,
		scenario.Name,
		scenario.Description,
		scenario.Duration,
		map[bool]string{true: "Auto-play", false: "Manual"}[scenario.AutoPlay],
	)

	dr.model.addSystemMessage(introMsg)

	// Execute commands
	for i, cmd := range scenario.Commands {
		if dr.paused {
			// Wait for unpause
			continue
		}

		// Show command description
		if cmd.Description != "" {
			descMsg := fmt.Sprintf("📋 Step %d: %s", i+1, cmd.Description)
			dr.model.addSystemMessage(descMsg)
		}

		// Delay before typing
		time.Sleep(cmd.Delay)

		// Simulate typing or execute command
		if scenario.AutoPlay {
			dr.executeCommand(cmd.Input)
		} else {
			dr.showCommandPrompt(cmd.Input)
		}

		// Pause after command
		time.Sleep(cmd.PauseAfter)
	}

	// Show completion message
	completionMsg := fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Demo Complete: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Expected outcomes achieved:
%s
`, scenario.Name, formatExpectedOutcomes(scenario.Expected))

	dr.model.addSystemMessage(completionMsg)

	return nil
}

// executeCommand automatically executes a command
func (dr *DemoRunner) executeCommand(input string) {
	// Simulate typing effect
	dr.model.input.SetValue(input)
	dr.model.input.Focus()

	// Small delay to show the typed command
	time.Sleep(300 * time.Millisecond)

	// Process the command
	dr.model.handleSendMessage()
}

// showCommandPrompt shows command for manual execution
func (dr *DemoRunner) showCommandPrompt(input string) {
	promptMsg := fmt.Sprintf("👉 Type: %s", input)
	dr.model.addSystemMessage(promptMsg)
}

// Pause pauses the demo execution
func (dr *DemoRunner) Pause() {
	dr.paused = true
}

// Resume resumes the demo execution
func (dr *DemoRunner) Resume() {
	dr.paused = false
}

// formatExpectedOutcomes formats expected outcomes as a bullet list
func formatExpectedOutcomes(outcomes []string) string {
	result := ""
	for _, outcome := range outcomes {
		result += fmt.Sprintf("  • %s\n", outcome)
	}
	return result
}

// GetDemoScenarioByName returns a demo scenario by name
func GetDemoScenarioByName(name string) *DemoScenario {
	for i := range DemoScenarios {
		if DemoScenarios[i].Name == name {
			return &DemoScenarios[i]
		}
	}
	return nil
}

// ListDemoScenarios returns a formatted list of available scenarios
func ListDemoScenarios() string {
	result := "🏰 Available Demo Scenarios:\n\n"
	for i, scenario := range DemoScenarios {
		result += fmt.Sprintf("%d. %s\n   %s\n   Duration: %v\n\n",
			i+1, scenario.Name, scenario.Description, scenario.Duration)
	}
	return result
}
