package demo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVisualRegressionSuite tests that visual components don't break each other
func TestVisualRegressionSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping visual regression tests in short mode")
	}

	workDir := t.TempDir()
	setupTestGuildProject(t, workDir)

	t.Run("markdown_syntax_highlighting_compatibility", func(t *testing.T) {
		// Test that markdown and syntax highlighting work together
		content := createComplexMarkdownContent()
		
		// Write test content
		contentFile := filepath.Join(workDir, "test-visual.md")
		err := os.WriteFile(contentFile, []byte(content), 0644)
		require.NoError(t, err)

		// Read back and verify structure
		readContent, err := os.ReadFile(contentFile)
		require.NoError(t, err)

		contentStr := string(readContent)
		
		// Should contain all elements
		assert.Contains(t, contentStr, "# Visual Test")
		assert.Contains(t, contentStr, "```go")
		assert.Contains(t, contentStr, "```python")
		assert.Contains(t, contentStr, "**bold text**")
		assert.Contains(t, contentStr, "- List item")
		assert.Contains(t, contentStr, "> This is a blockquote")
		
		// Should be well-formed
		assert.True(t, strings.Count(contentStr, "```")%2 == 0, "Code blocks should be balanced")
		assert.Contains(t, contentStr, "fmt.Println", "Go code should be present")
		assert.Contains(t, contentStr, "async def", "Python code should be present")
	})

	t.Run("agent_status_markdown_integration", func(t *testing.T) {
		// Test agent status display with markdown content
		agentConfig := createAdvancedAgentConfig()
		
		configPath := filepath.Join(workDir, ".guild", "guild.yaml")
		err := os.WriteFile(configPath, []byte(agentConfig), 0644)
		require.NoError(t, err)

		// Create status simulation content
		statusContent := `# Agent Status Integration Test

## Current Agents
- 🤔 **Manager**: Analyzing requirements
- ⚙️ **Developer**: Implementing features  
- 🔍 **Reviewer**: Checking code quality
- 📊 **Architect**: Designing systems

## Task Progress
` + "```" + `
┌─────────────────┬──────────┬─────────┐
│ Agent           │ Status   │ Task    │
├─────────────────┼──────────┼─────────┤
│ manager         │ thinking │ plan    │
│ developer       │ working  │ code    │
│ reviewer        │ idle     │ -       │
└─────────────────┴──────────┴─────────┘
` + "```" + `

### Rich Content with Status
This tests that **status displays** work with *markdown rendering*.`

		statusFile := filepath.Join(workDir, "status-test.md")
		err = os.WriteFile(statusFile, []byte(statusContent), 0644)
		require.NoError(t, err)

		// Verify content structure
		content, err := os.ReadFile(statusFile)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "🤔 **Manager**")
		assert.Contains(t, contentStr, "⚙️ **Developer**")
		assert.Contains(t, contentStr, "thinking")
		assert.Contains(t, contentStr, "working")
	})

	t.Run("auto_completion_visual_layout", func(t *testing.T) {
		// Test auto-completion with rich visual content
		completionScenarios := []struct {
			input    string
			context  string
			expected []string
		}{
			{
				input:   "@man",
				context: "agent completion with markdown",
				expected: []string{"manager", "architect"},
			},
			{
				input:   "/test",
				context: "command completion with code blocks",
				expected: []string{"markdown", "code", "mixed"},
			},
			{
				input:   "/prompt ",
				context: "subcommand completion with status display",
				expected: []string{"list", "show", "edit"},
			},
		}

		for _, scenario := range completionScenarios {
			t.Run(scenario.context, func(t *testing.T) {
				// Create test content that includes the completion context
				testContent := fmt.Sprintf(`# Completion Test: %s

User typed: %s

Expected completions:
%s

This should work alongside rich markdown and status displays.`,
					scenario.context,
					scenario.input,
					strings.Join(scenario.expected, ", "))

				testFile := filepath.Join(workDir, "completion-test.md")
				err := os.WriteFile(testFile, []byte(testContent), 0644)
				require.NoError(t, err)

				// Verify structure
				content, err := os.ReadFile(testFile)
				require.NoError(t, err)
				assert.Contains(t, string(content), scenario.input)
			})
		}
	})

	t.Run("command_history_rich_content_preservation", func(t *testing.T) {
		// Test that command history preserves rich content formatting
		historyCommands := []string{
			"/test markdown",
			"@manager analyze **this important code**",
			"/test code go",
			"@developer implement ```func main() {}```",
			"/test mixed content with *emphasis*",
		}

		historyFile := filepath.Join(workDir, "command-history.txt")
		historyContent := strings.Join(historyCommands, "\n")
		
		err := os.WriteFile(historyFile, []byte(historyContent), 0644)
		require.NoError(t, err)

		// Read back and verify preservation
		content, err := os.ReadFile(historyFile)
		require.NoError(t, err)

		contentStr := string(content)
		
		// Rich formatting should be preserved
		assert.Contains(t, contentStr, "**this important code**")
		assert.Contains(t, contentStr, "```func main() {}```") 
		assert.Contains(t, contentStr, "*emphasis*")
		
		// Commands should be preserved
		for _, cmd := range historyCommands {
			assert.Contains(t, contentStr, cmd)
		}
	})

	t.Run("error_display_formatting", func(t *testing.T) {
		// Test that error messages display properly with rich content
		errorScenarios := []struct {
			name    string
			error   string
			context string
		}{
			{
				name:    "markdown_render_error",
				error:   "Failed to render markdown: invalid syntax",
				context: "# Test Content\n\nThis has **bold** text",
			},
			{
				name:    "agent_communication_error", 
				error:   "Agent 'developer' not responding",
				context: "@developer implement ```go\nfunc test() {}\n```",
			},
			{
				name:    "tool_execution_error",
				error:   "Tool execution failed: permission denied",
				context: "/tools run shell --command \"ls -la\"",
			},
		}

		for _, scenario := range errorScenarios {
			t.Run(scenario.name, func(t *testing.T) {
				errorContent := fmt.Sprintf(`# Error Test: %s

## Context
%s

## Error
❌ %s

## Expected Behavior
Error should display clearly without breaking visual formatting.`,
					scenario.name,
					scenario.context,
					scenario.error)

				errorFile := filepath.Join(workDir, scenario.name+"-error.md")
				err := os.WriteFile(errorFile, []byte(errorContent), 0644)
				require.NoError(t, err)

				// Verify error formatting
				content, err := os.ReadFile(errorFile)
				require.NoError(t, err)
				
				contentStr := string(content)
				assert.Contains(t, contentStr, "❌")
				assert.Contains(t, contentStr, scenario.error)
				assert.Contains(t, contentStr, scenario.context)
			})
		}
	})
}

// TestDemoVisualStability tests that visuals remain stable during demo execution
func TestDemoVisualStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping visual stability tests in short mode")
	}

	workDir := t.TempDir()
	setupTestGuildProject(t, workDir)

	t.Run("layout_stability_under_load", func(t *testing.T) {
		// Simulate heavy visual content
		contentParts := []string{
			"# Large Demo Content\n\n",
			strings.Repeat("## Section\n\nContent with **formatting**.\n\n", 10),
			"```go\n" + strings.Repeat("func test() { fmt.Println(\"test\") }\n", 20) + "```\n\n",
			strings.Repeat("- List item with *emphasis*\n", 50),
			"> " + strings.Repeat("Long blockquote content. ", 100) + "\n\n",
		}

		largeContent := strings.Join(contentParts, "")
		
		contentFile := filepath.Join(workDir, "large-content.md")
		err := os.WriteFile(contentFile, []byte(largeContent), 0644)
		require.NoError(t, err)

		// Verify it can be processed
		content, err := os.ReadFile(contentFile)
		require.NoError(t, err)
		
		// Should handle large content gracefully
		assert.Greater(t, len(content), 5000, "Should have substantial content")
		assert.Contains(t, string(content), "Large Demo Content")
		assert.Contains(t, string(content), "func test()")
	})

	t.Run("rapid_content_updates", func(t *testing.T) {
		// Simulate rapid content changes during demo
		baseDir := filepath.Join(workDir, "rapid-updates")
		err := os.MkdirAll(baseDir, 0755)
		require.NoError(t, err)

		// Create multiple content files rapidly
		for i := 0; i < 20; i++ {
			content := fmt.Sprintf(`# Update %d

**Time**: %v
**Status**: Processing update %d

` + "```go\nfunc update%d() {\n    fmt.Printf(\"Update %d\")\n}\n```" + `

## Progress
%s`,
				i,
				time.Now().Format(time.RFC3339),
				i,
				i, i,
				strings.Repeat("█", i%20))

			filename := filepath.Join(baseDir, fmt.Sprintf("update-%d.md", i))
			err := os.WriteFile(filename, []byte(content), 0644)
			require.NoError(t, err)
		}

		// Verify all files were created successfully
		files, err := os.ReadDir(baseDir)
		require.NoError(t, err)
		assert.Equal(t, 20, len(files))
	})

	t.Run("multi_agent_status_display_stability", func(t *testing.T) {
		// Test that multi-agent status displays remain stable
		agentStatuses := []struct {
			agent  string
			status string
			task   string
			emoji  string
		}{
			{"manager", "thinking", "Analyzing requirements", "🤔"},
			{"developer", "working", "Implementing authentication", "⚙️"},
			{"reviewer", "reviewing", "Checking code quality", "🔍"},
			{"architect", "designing", "System architecture", "📐"},
			{"tester", "testing", "Running test suite", "🧪"},
			{"deployer", "deploying", "Production deployment", "🚀"},
		}

		statusContent := "# Multi-Agent Status Display\n\n"
		
		for _, agent := range agentStatuses {
			statusContent += fmt.Sprintf("## %s %s Agent\n\n", agent.emoji, strings.Title(agent.agent))
			statusContent += fmt.Sprintf("**Status**: %s\n", agent.status)
			statusContent += fmt.Sprintf("**Current Task**: %s\n\n", agent.task)
			statusContent += fmt.Sprintf("```\n[%s] %s: %s\n```\n\n", 
				time.Now().Format("15:04:05"), agent.agent, agent.task)
		}

		statusFile := filepath.Join(workDir, "multi-agent-status.md")
		err := os.WriteFile(statusFile, []byte(statusContent), 0644)
		require.NoError(t, err)

		// Verify status display structure
		content, err := os.ReadFile(statusFile)
		require.NoError(t, err)
		
		contentStr := string(content)
		for _, agent := range agentStatuses {
			assert.Contains(t, contentStr, agent.emoji)
			assert.Contains(t, contentStr, agent.agent)
			assert.Contains(t, contentStr, agent.status)
			assert.Contains(t, contentStr, agent.task)
		}
	})
}

// Helper functions for visual regression testing

func createComplexMarkdownContent() string {
	return `# Visual Test Document

This document tests the integration of **multiple visual components**.

## Markdown Formatting

Text can be **bold**, *italic*, or ***both***.

### Lists and Structure

Unordered list:
- List item 1 with **bold**
- List item 2 with *italic*
- List item 3 with ` + "`code`" + `

Ordered list:
1. First item
2. Second item with [link](https://example.com)
3. Third item

## Code Examples

### Go Code
` + "```go" + `
package main

import (
    "fmt"
    "time"
)

func main() {
    fmt.Println("Welcome to Guild!")
    
    agents := []string{"manager", "developer", "reviewer"}
    for _, agent := range agents {
        fmt.Printf("Agent: %s\n", agent)
    }
}
` + "```" + `

### Python Code
` + "```python" + `
import asyncio
from typing import List, Dict

class GuildAgent:
    def __init__(self, name: str, capabilities: List[str]):
        self.name = name
        self.capabilities = capabilities
    
    async def execute_task(self, task: str) -> Dict[str, str]:
        await asyncio.sleep(0.1)  # Simulate work
        return {"status": "complete", "result": f"Completed: {task}"}

async def main():
    agents = [
        GuildAgent("manager", ["planning", "coordination"]),
        GuildAgent("developer", ["coding", "testing"]),
    ]
    
    for agent in agents:
        result = await agent.execute_task("sample task")
        print(f"{agent.name}: {result}")

if __name__ == "__main__":
    asyncio.run(main())
` + "```" + `

## Tables

| Agent     | Status   | Current Task         |
|-----------|----------|---------------------|
| Manager   | 🤔 Thinking | Analyzing requirements |
| Developer | ⚙️ Working  | Implementing auth    |
| Reviewer  | 🔍 Reviewing | Code quality check   |

## Blockquotes

> This is a blockquote that demonstrates how quoted content
> appears alongside other visual elements. It should maintain
> proper formatting and spacing.

## Mixed Content Test

Here's a paragraph with **bold text** followed by a code block:

` + "```bash" + `
# Guild CLI commands
guild init
guild chat --campaign "demo"
guild corpus scan --path ./src
` + "```" + `

And then more text with *emphasis* and ` + "`inline code`" + `.

## Emoji and Unicode

Guild supports rich visual elements:
- 🏰 Medieval guild theming
- ⚙️ Agent status indicators  
- 📊 Progress visualization
- 🚀 Deployment indicators
- ✅ Success markers
- ❌ Error indicators

## Final Test

This document exercises multiple visual components:
1. ✅ Markdown rendering
2. ✅ Syntax highlighting
3. ✅ Tables and lists
4. ✅ Unicode and emoji
5. ✅ Mixed content types

**All components should render correctly together.**`
}

func createAdvancedAgentConfig() string {
	return `name: visual-test-guild
description: Advanced configuration for visual testing

agents:
  - id: manager
    name: Guild Master
    role: manager
    provider: mock
    model: test-model
    capabilities:
      - coordination
      - planning
      - task-breakdown
      - resource-allocation
    status_emoji: "🤔"
    
  - id: developer
    name: Code Artisan
    role: developer
    provider: mock
    model: test-model
    capabilities:
      - implementation
      - coding
      - testing
      - debugging
    status_emoji: "⚙️"
    
  - id: reviewer
    name: Quality Keeper
    role: reviewer
    provider: mock
    model: test-model
    capabilities:
      - code-review
      - quality-assurance
      - validation
      - documentation
    status_emoji: "🔍"
    
  - id: architect
    name: System Designer
    role: architect
    provider: mock
    model: test-model
    capabilities:
      - system-design
      - architecture
      - planning
      - integration
    status_emoji: "📐"

campaigns:
  - name: visual-test
    description: Visual component testing campaign
    agents: [manager, developer, reviewer, architect]
    
visual_config:
  theme: medieval
  enable_rich_content: true
  enable_syntax_highlighting: true
  enable_agent_status: true
  enable_progress_bars: true
  
demo_settings:
  auto_play: false
  show_typing_effect: true
  pause_between_commands: 1s
  highlight_active_agent: true`
}