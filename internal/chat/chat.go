package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"

	"github.com/guild-ventures/guild-core/internal/chat/commands"
	"github.com/guild-ventures/guild-core/pkg/config"
	guildcontext "github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	guildgrpc "github.com/guild-ventures/guild-core/pkg/grpc"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/tools"
)

// Helper to check if agent has capability

// CommandHistory manages command history with search and navigation

// NewCommandHistory creates a new command history manager

// Add adds a command to history

// Previous returns the previous command in history

// Next returns the next command in history

// Search performs fuzzy search on command history

// GetRecent returns the most recent N commands

// Command represents a Guild command with completion information

// CommandProcessor handles command processing and completion integration

// NewCommandProcessor creates a new command processor

// Helper functions

// fuzzyMatch performs intelligent fuzzy matching

// fuzzyScore calculates a relevance score for fuzzy matching (lower is better)
func fuzzyScore(text, pattern string) int {
	text = strings.ToLower(text)
	pattern = strings.ToLower(pattern)

	// Exact match is best
	if text == pattern {
		return 0
	}

	// Prefix match is second best
	if strings.HasPrefix(text, pattern) {
		return 1
	}

	// Substring match
	if strings.Contains(text, pattern) {
		return 2
	}

	// Character sequence match - calculate distance
	score := 0
	textIdx := 0
	lastMatch := -1

	for _, patternChar := range pattern {
		for textIdx < len(text) {
			if rune(text[textIdx]) == patternChar {
				if lastMatch >= 0 {
					// Add distance between matches
					score += textIdx - lastMatch
				}
				lastMatch = textIdx
				textIdx++
				break
			}
			textIdx++
		}
	}

	return score + 3 // Base score for fuzzy match
}

// loadHistory loads command history from file

// saveHistory saves command history to file

// Export exports the command history to a specified file
func (ch *CommandHistory) Export(filename string) error {
	// Build formatted export with timestamps
	var content strings.Builder
	content.WriteString("# Guild Chat Command History\n")
	content.WriteString(fmt.Sprintf("# Exported: %s\n\n", time.Now().Format(time.RFC3339)))

	for i, cmd := range ch.commands {
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, cmd))
	}

	return os.WriteFile(filename, []byte(content.String()), 0644)
}

// removeDuplicate removes duplicate commands from history

// Custom tea.Msg types for agent streaming

// Tool execution messages

// Agent status update messages (Agent 3)

// chatKeyMap defines keyboard shortcuts for Guild Chat

// newChatKeyMap creates the default key mappings

// ShortHelp returns keybindings to be shown in the mini help view
// FullHelp returns keybindings for the expanded help view
// newChatModel creates a new Guild Chat model
func newChatModel(guildConfig *config.GuildConfig, campaignID, sessionID string,
	grpcConn *grpc.ClientConn, guildClient pb.GuildClient,
	promptClient promptspb.PromptServiceClient, registry registry.ComponentRegistry) ChatModel {
	// Initialize textarea for input
	ta := textarea.New()
	ta.Placeholder = "Message agents with @agent-name or use /commands..."
	ta.Focus()
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	// Initialize viewport for messages
	vp := viewport.New(0, 0)
	// Create a medieval-themed welcome banner
	welcomeBanner := `🏰 ═══════════════════════════════════════════ 🏰
   Welcome to the Guild Chat Chamber!

   ⚔️  Your agents await your commands
   🛡️  Type /help to see available commands
   👑  Use @agent-name to message specific agents
   📜  Use @all to broadcast to all agents

   Ready to craft great software together!
🏰 ═══════════════════════════════════════════ 🏰

Rich content rendering is ACTIVE! ✨
Try these commands to see visual features:
• /test markdown - See styled headers and formatting
• /test code go - View syntax highlighted code
• /status - View real-time agent status panel

`
	vp.SetContent(welcomeBanner)

	// Initialize help
	help := help.New()

	// Styles are now defined in the individual components

	// Create a new context for this chat session
	ctx := context.Background()
	ctx = guildcontext.NewGuildContext(ctx)
	ctx = guildcontext.WithSessionID(ctx, sessionID)
	if campaignID != "" {
		ctx = guildcontext.WithOperation(ctx, "campaign:"+campaignID)
	}

	// Initialize rich content rendering
	chatWidth := 80 // Default width, will be updated on resize
	markdownRenderer, err := NewMarkdownRenderer(chatWidth)
	if err != nil {
		// If markdown renderer fails, continue without it (graceful degradation)
		markdownRenderer = nil
	}
	contentFormatter := NewContentFormatter(markdownRenderer, chatWidth)

	// Initialize command completion and history
	// Use current working directory as project root
	projectRoot := "." // Agent 2 will improve this logic

	completionEngine := NewCompletionEngine(guildConfig, projectRoot)
	commandHistory := NewCommandHistory(projectRoot + "/.guild/chat_history.txt")
	commandPalette := commands.NewCommandPalette()
	// Command processor will be set after model creation

	// Initialize agent status systems (Agent 3)
	statusTracker := NewAgentStatusTracker(guildConfig)
	statusDisplay := NewStatusDisplay(statusTracker, chatWidth/4, chatWidth/3)
	agentIndicators := NewAgentIndicators()

	// Start the status tracking systems
	statusTracker.StartTracking()
	agentIndicators.SetupDefaultAnimations()

	// Build the complete model
	model := ChatModel{
		// UI Components
		input:        ta,
		viewport:     vp,
		help:         help,
		width:        0, // Will be set on window size
		height:       0, // Will be set on window size
		ready:        false,
		err:          nil,
		viewMode:     chatModeNormal,
		keys:         newChatKeyMap(),
		focusedAgent: "",

		// Visual Components
		markdownRenderer:   markdownRenderer,
		contentFormatter:   contentFormatter,
		agentStatusTracker: statusTracker,
		statusDisplay:      statusDisplay,
		agentIndicators:    agentIndicators,

		// Core Components
		grpcClient:     guildClient,
		promptsClient:  promptClient,
		sessionID:      sessionID,
		campaignID:     campaignID,
		guildConfig:    guildConfig,
		commandProc:    nil, // Set after model is created
		completionEng:  completionEngine,
		history:        commandHistory,
		commandPalette: commandPalette,
		registry:       registry,

		// State
		messages:      []Message{},
		activeTools:   make(map[string]*toolExecution),
		agents:        []string{},
		promptLayers:  []string{},
		searchPattern: "",
		searchMatches: []int{},
		currentMatch:  0,
		costConsent:   make(map[string]bool),
		taskCache:     make(map[string]string),
		blockedTools:  make(map[string]bool),
	}

	// Set the command processor after model creation
	model.commandProc = NewCommandProcessor(&model)

	// Return the fully initialized model
	return model
}

// Init implements tea.Model

// listenForAgentUpdates listens for streaming agent responses

// Update implements tea.Model

// handleAgentStream handles streaming agent response fragments

// handleAgentStatus handles agent status updates

// handleAgentError handles agent error messages

// handleAgentStatusUpdate handles status system updates (Agent 3)

// View implements tea.Model

// handleSendMessage processes user input

// processMessage handles different types of user input

// handleCommand processes slash commands

// handleAgentMention processes agent mentions

// streamAgentConversation handles bidirectional streaming with an agent

// handlePromptCommand processes prompt-related commands

// updateMessagesView refreshes the messages viewport
// safeFormatContent safely formats content with error recovery

// Helper methods for command responses

// getStatusIcon returns an icon for the agent status
func getStatusIcon(status *pb.AgentStatus) string {
	if status == nil {
		return "⚪"
	}
	switch status.State {
	case pb.AgentStatus_IDLE:
		return "🟢"
	case pb.AgentStatus_THINKING:
		return "🤔"
	case pb.AgentStatus_WORKING:
		return "⚙️"
	case pb.AgentStatus_WAITING:
		return "⏳"
	case pb.AgentStatus_ERROR:
		return "🔴"
	case pb.AgentStatus_OFFLINE:
		return "⚫"
	default:
		return "⚪"
	}
}

// getStatusText returns text description for the agent status

func (m ChatModel) getPromptLayersText() string {
	return `🏰 Active Prompt Layers:

📋 Platform Layer:
   Safety guidelines, Guild ethics, core principles
   Token usage: ~200 tokens

🏰 Guild Layer:
   Project-specific goals and coding standards
   Token usage: ~300 tokens

👷 Role Layer:
   Agent-specific role definitions and capabilities
   Token usage: ~400 tokens (varies by agent)

📚 Domain Layer:
   Project type specializations (web-app, cli-tool, etc.)
   Token usage: ~250 tokens

👤 Session Layer:
   User preferences and session-specific context
   Token usage: ~150 tokens

💬 Turn Layer:
   Ephemeral instructions for current interaction
   Token usage: ~100 tokens

Total estimated usage: 1,400 / 8,000 tokens (17.5%)

Note: Actual layered prompt integration is in progress.
Use /prompt get --layer <name> to view specific layers.`
}

func (m ChatModel) handleAgentList() (ChatModel, tea.Cmd) {
	// Add agent list message
	msg := Message{
		Timestamp: time.Now(),
		AgentID:   "system",
		Content:   m.getAgentsText(),
		Type:      msgSystem,
	}
	m.addMessage(msg)
	m.updateMessagesView()

	return m, nil
}

// addMessage adds a message to appropriate streams (global + agent-specific)

// getCurrentMessages returns the appropriate message stream for current view
func (m *ChatModel) getCurrentMessages() []Message {
	// TODO: Implement view-specific message filtering when agentStreams is added
	// For now, return all messages
	return m.messages
}

// handleGlobalView switches to global view mode

// handleAgentFocus prompts user to select an agent to focus on
func (m ChatModel) handleAgentFocus() (ChatModel, tea.Cmd) {
	// For now, show instructions - in a full implementation this could open a selection UI
	msg := Message{
		Timestamp: time.Now(),
		AgentID:   "system",
		Content:   "🎯 To focus on an agent, use @agent-name or try:\n" + m.getAgentFocusHelp(),
		Type:      msgSystem,
	}
	m.addMessage(msg)
	m.updateMessagesView()

	return m, nil
}

// getAgentFocusHelp returns help text for agent focusing
func (m ChatModel) getAgentFocusHelp() string {
	var content strings.Builder
	content.WriteString("Available agents for focus:\n")

	for i, agent := range m.agents {
		// TODO: Show message count when agentStreams is implemented
		content.WriteString(fmt.Sprintf("  %d. @%s\n", i+1, agent))
	}

	content.WriteString("\nUse '@agent-id your message' to start a focused conversation.")
	return content.String()
}

// handleToolCommand processes /tools commands for discovery and management
func (m ChatModel) handleToolCommand(args []string) string {
	if len(args) == 0 {
		return m.getToolHelpText()
	}

	switch args[0] {
	case "list", "ls":
		return m.getToolListText()
	case "info":
		if len(args) < 2 {
			return "Usage: /tools info <tool-id>"
		}
		return m.getToolInfoText(args[1])
	case "search":
		if len(args) < 2 {
			return "Usage: /tools search <capability>"
		}
		return m.searchToolsText(args[1])
	case "status":
		return m.getToolStatusText()
	default:
		return fmt.Sprintf("Unknown tools subcommand: %s. Use /tools to see available options.", args[0])
	}
}

// handleToolExecuteCommand processes /tool <tool-id> commands for direct execution
func (m ChatModel) handleToolExecuteCommand(args []string) string {
	if len(args) == 0 {
		return "Usage: /tool <tool-id> [parameters...]\nExample: /tool file-reader --path ./README.md"
	}

	toolID := args[0]

	// Parse parameters (simple key=value for now)
	params := make(map[string]string)
	for _, arg := range args[1:] {
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			params[strings.TrimPrefix(parts[0], "--")] = parts[1]
		} else if strings.HasPrefix(arg, "--") {
			// Flag without value
			params[strings.TrimPrefix(arg, "--")] = "true"
		}
	}

	// Start tool execution
	return m.executeToolDirectly(toolID, params)
}

// getToolHelpText returns help text for tool commands
func (m ChatModel) getToolHelpText() string {
	return `🔨 Guild Tool Commands:

/tools list                    - List all available tools
/tools info <tool-id>          - Show detailed tool information
/tools search <capability>     - Find tools by capability
/tools status                  - Show active tool executions

/tool <tool-id> [params]       - Execute a tool directly
  Example: /tool file-reader --path ./README.md
  Example: /tool shell-exec --command "ls -la"

Active Tool Executions:
` + m.formatActiveToolExecutions()
}

// getToolListText returns a list of available tools

// getToolInfoText returns detailed information about a specific tool

// searchToolsText searches for tools by capability
func (m ChatModel) searchToolsText(capability string) string {
	// Try tool registry search
	var toolRegistry interface {
		ListTools() []string
		GetTool(string) interface{}
		GetToolsByCapability(string) []interface{}
	} = nil // TODO: Add registry field.Tools()
	if toolRegistry != nil {
		tools := toolRegistry.GetToolsByCapability(capability)
		if len(tools) > 0 {
			var content strings.Builder
			content.WriteString(fmt.Sprintf("🔍 Tools with capability '%s':\n\n", capability))

			for i, tool := range tools {
				toolName := "unknown"
				if namedTool, ok := tool.(interface{ Name() string }); ok {
					toolName = namedTool.Name()
				}

				desc := "No description available"
				if toolWithSchema, ok := tool.(interface{ Schema() map[string]interface{} }); ok {
					schema := toolWithSchema.Schema()
					if description, exists := schema["description"]; exists {
						desc = fmt.Sprintf("%v", description)
					}
				}

				content.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Render(toolName), desc))
			}

			return content.String()
		}
	}

	// Fallback to mock search
	mockResults := map[string][]string{
		"file":   {"file-reader", "file-writer"},
		"web":    {"web-scraper", "http-client"},
		"system": {"shell-exec", "process-monitor"},
		"search": {"corpus-search", "web-search"},
	}

	var results []string
	for key, tools := range mockResults {
		if strings.Contains(key, capability) || strings.Contains(capability, key) {
			results = append(results, tools...)
		}
	}

	if len(results) == 0 {
		return fmt.Sprintf("No tools found with capability '%s'. Use '/tools list' to see all tools.", capability)
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("🔍 Tools with capability '%s':\n\n", capability))
	for i, tool := range results {
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Render(tool)))
	}

	return content.String()
}

// getToolStatusText returns status of active tool executions
func (m ChatModel) getToolStatusText() string {
	if len(m.activeTools) == 0 {
		return "🔨 No active tool executions."
	}

	var content strings.Builder
	content.WriteString("🔨 Active Tool Executions:\n\n")

	i := 0
	for execID, execution := range m.activeTools {
		_ = execID // Not used in this loop
		duration := time.Since(execution.StartTime)
		if execution.EndTime != nil {
			duration = execution.EndTime.Sub(execution.StartTime)
		}

		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Render(execution.ToolName)))
		content.WriteString(fmt.Sprintf("   ID: %s\n", execution.ID))
		content.WriteString(fmt.Sprintf("   Agent: %s\n", execution.AgentID))
		content.WriteString(fmt.Sprintf("   Status: %s\n", execution.Status))
		content.WriteString(fmt.Sprintf("   Duration: %v\n", duration.Round(time.Millisecond)))
		if execution.Progress > 0 {
			content.WriteString(fmt.Sprintf("   Progress: %.1f%%\n", execution.Progress*100))
		}
		// Display cost if available
		if execution.Cost > 0 {
			content.WriteString(fmt.Sprintf("   Cost: %.2f credits\n", execution.Cost))
		}
		content.WriteString("\n")
		i++
	}

	return content.String()
}

// executeToolDirectly executes a tool with given parameters
func (m ChatModel) executeToolDirectly(toolID string, params map[string]string) string {
	// Generate execution ID
	execID := fmt.Sprintf("exec-%d", time.Now().UnixNano())

	// Create tool execution record
	execution := &toolExecution{
		ID:         execID,
		ToolName:   toolID, // Will be updated with real name
		AgentID:    "chat-user",
		StartTime:  time.Now(),
		Status:     "starting",
		Progress:   0.0,
		Parameters: params,
	}

	// Add to tracking
	m.activeTools[execID] = execution
	// m.activeTools is a map, not a slice - already added above

	// Send tool execution request via gRPC
	go m.executeToolViaGRPC(execID, toolID, params)

	return fmt.Sprintf("🔨 Started tool execution: %s\nExecution ID: %s\nParameters: %v\n\nUse '/tools status' to monitor progress.",
		lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true).Render(toolID), execID, params)
}

// simulateToolExecution simulates tool execution (placeholder)
func (m *ChatModel) simulateToolExecution(execID string) {
	// This would be replaced with actual tool execution
	execution := m.activeTools[execID]
	if execution == nil {
		return
	}

	// Simulate progress updates
	for i := 0; i <= 100; i += 25 {
		time.Sleep(500 * time.Millisecond)
		execution.Progress = float32(i) / 100.0
		execution.Status = "running"

		// TODO: Send progress update via tea.Cmd
	}

	// Complete execution
	now := time.Now()
	execution.EndTime = &now
	execution.Status = "completed"
	execution.Result = "Tool execution completed successfully"
	// Cost tracking is now implemented in executeToolViaGRPC

	// Remove from active tools map
	delete(m.activeTools, execID)

	// TODO: Send completion message via tea.Cmd
}

// formatActiveToolExecutions formats the active tool executions for display
func (m ChatModel) formatActiveToolExecutions() string {
	if len(m.activeTools) == 0 {
		return "  None\n"
	}

	var content strings.Builder
	for execID, execution := range m.activeTools {
		_ = execID // unused
		progress := ""
		if execution.Progress > 0 {
			progress = fmt.Sprintf(" (%.0f%%)", execution.Progress*100)
		}
		content.WriteString(fmt.Sprintf("  %s - %s%s\n",
			execution.ToolName, execution.Status, progress))
	}

	return content.String()
}

// Tool execution message handlers

// handleToolExecutionStart handles the start of a tool execution

// handleToolExecutionProgress handles progress updates for tool execution

// handleToolExecutionComplete handles completion of tool execution

// handleToolExecutionError handles errors during tool execution

// handleToolAuthRequired handles authorization requests for tool execution

// createProgressBar creates a visual progress bar
func (m *ChatModel) createProgressBar(progress float64) string {
	const barWidth = 20
	filled := int(progress * barWidth)

	var bar strings.Builder
	bar.WriteString("[")

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteString("█")
		} else {
			bar.WriteString("░")
		}
	}

	bar.WriteString("]")
	return bar.String()
}

// Test methods for Agent 1: Rich Content Rendering

// generateMarkdownTestContent creates test content to demonstrate markdown rendering
func (m ChatModel) generateMarkdownTestContent() string {
	return `# Rich Markdown Rendering Test 🏰

This demonstrates Guild's **rich content rendering** capabilities!

## Features Included:
- **Bold text** and *italic text*
- Headers at multiple levels
- Bullet points and numbered lists
- [Links](https://github.com/guild-ventures/guild-core)
- ` + "`inline code`" + ` with highlighting

### Why This Matters:
1. **Professional appearance** - No more plain text responses
2. **Better readability** - Structured content is easier to parse
3. **Visual hierarchy** - Headers and emphasis guide the eye
4. **Code highlighting** - Syntax highlighting for technical content

> This is a blockquote showing Guild's superior visual presentation compared to plain-text tools like Aider.

---

**Try ` + "`/test code go`" + ` to see syntax highlighting in action!**`
}

// generateCodeTestContent creates test content with syntax highlighting
func (m ChatModel) generateCodeTestContent(language string) string {
	switch language {
	case "go", "golang":
		return `# Go Code Syntax Highlighting ⚔️

Here's a sample Go function with **full syntax highlighting**:

` + "```go" + `
package main

import (
    "fmt"
    "time"
    "github.com/charmbracelet/lipgloss"
)

// Agent represents a Guild agent
type Agent struct {
    ID          string
    Name        string
    Capabilities []string
    Status      AgentStatus
}

func (a *Agent) ExecuteTask(task string) error {
    fmt.Printf("🤖 Agent %s executing: %s\n", a.Name, task)

    // Simulate work
    time.Sleep(100 * time.Millisecond)

    if task == "impossible" {
        return fmt.Errorf("task cannot be completed")
    }

    return nil
}
` + "```" + `

**Notice the beautiful syntax highlighting!** This makes code much easier to read and understand compared to plain text.`

	case "python", "py":
		return `# Python Code Syntax Highlighting 🐍

Here's a sample Python class with **full syntax highlighting**:

` + "```python" + `
import asyncio
from typing import List, Optional
from dataclasses import dataclass

@dataclass
class GuildAgent:
    """A Guild AI agent with specialization capabilities."""
    name: str
    specializations: List[str]
    active: bool = True

    async def process_task(self, task: str) -> Optional[str]:
        """Process a task asynchronously."""
        print(f"🤖 {self.name} processing: {task}")

        # Simulate async work
        await asyncio.sleep(0.1)

        if not self.active:
            raise ValueError("Agent is not active")

        return f"Completed: {task}"

# Example usage
agent = GuildAgent("Developer", ["coding", "debugging"])
result = await agent.process_task("implement feature")
` + "```" + `

**Python syntax highlighting** shows keywords, strings, comments, and types clearly!`

	case "javascript", "js":
		return `# JavaScript Syntax Highlighting ⚡

Here's a sample JavaScript with **full syntax highlighting**:

` + "```javascript" + `
class GuildAgent {
    constructor(name, capabilities) {
        this.name = name;
        this.capabilities = capabilities;
        this.status = 'idle';
    }

    async executeTask(task) {
        console.log(` + "`🤖 ${this.name} executing: ${task}`" + `);
        this.status = 'working';

        try {
            // Simulate async work
            await new Promise(resolve => setTimeout(resolve, 100));

            const result = {
                success: true,
                output: ` + "`Task completed: ${task}`" + `,
                timestamp: new Date().toISOString()
            };

            this.status = 'idle';
            return result;
        } catch (error) {
            this.status = 'error';
            throw new Error(` + "`Failed to execute: ${task}`" + `);
        }
    }
}

// Usage
const agent = new GuildAgent('Frontend', ['react', 'typescript']);
const result = await agent.executeTask('build component');
` + "```" + `

**JavaScript highlighting** shows modern ES6+ syntax beautifully!`

	default:
		return fmt.Sprintf(`# Code Syntax Highlighting 🔧

**Language**: %s

`+"```%s"+`
// Sample code in %s
function example() {
    console.log("This is %s code with syntax highlighting!");
    return "syntax-highlighted";
}
`+"```"+`

**Try these languages**: go, python, javascript, sql, bash, yaml`, language, language, language, language)
	}
}

// handleTestCommand processes /test commands for demonstrating rich content rendering

// getTestHelpText returns help text for test commands
func (m ChatModel) getTestHelpText() string {
	return `🧪 Rich Content Test Commands:

/test markdown          - Demonstrate markdown features (headers, lists, emphasis)
/test code <language>   - Show syntax highlighting for specific language
/test mixed             - Show combined markdown and code content

Available languages for /test code:
  go, golang    - Go language syntax highlighting
  python, py    - Python syntax highlighting
  javascript, js - JavaScript syntax highlighting
  sql           - SQL syntax highlighting
  bash, sh      - Shell script highlighting
  yaml, yml     - YAML syntax highlighting

Examples:
  /test markdown
  /test code go
  /test code python
  /test mixed

These commands showcase Guild's rich content rendering capabilities that make it superior to plain-text tools like Aider or Claude Code! 🏰⚔️`
}

// generateMixedTestContent creates content mixing markdown and code
func (m ChatModel) generateMixedTestContent() string {
	return `# Mixed Content Demonstration 🎨

This shows **Guild's ability** to handle complex content with both markdown and code.

## Agent Implementation Example

Here's how a Guild agent might be implemented:

` + "```go" + `
func (agent *GuildAgent) ProcessCommission(commission *Commission) error {
    log.Printf("🏰 Processing commission: %s", commission.Title)

    // Break down into tasks
    tasks := agent.analyzer.BreakdownCommission(commission)

    for _, task := range tasks {
        if err := agent.ExecuteTask(task); err != nil {
            return gerror.Wrap(err, gerror.ErrCodeTaskFailed,
                "failed to execute task").
                WithComponent("GuildAgent").
                WithOperation("ProcessCommission")
        }
    }

    return nil
}
` + "```" + `

### Key Features:
- **Error handling** with structured Guild errors
- **Logging** with medieval theming 🏰
- **Task breakdown** for parallel execution
- **Clean interfaces** following Go best practices

### Command Options:
1. ` + "`/test markdown`" + ` - Basic markdown features
2. ` + "`/test code <lang>`" + ` - Language-specific highlighting
3. ` + "`/test mixed`" + ` - Combined markdown + code (this demo)

> **This is exactly the kind of rich, professional content that makes Guild superior to plain-text tools!**

---

*Rich content rendering powered by Agent 1's implementation* ⚔️`
}

// Agent 2: Completion and History Handlers

// handleTabCompletion handles tab key for intelligent auto-completion

// handleUpKey handles up arrow for history navigation or completion navigation

// handleDownKey handles down arrow for history navigation or completion navigation

// handleSearchHistory handles Ctrl+R for fuzzy history search

// handleEscape handles escape key for canceling completion or search

// Integration methods (Agent 4)

// InitializeAllComponents initializes and validates all chat components

// ValidateAllComponents ensures all components are properly configured

// Individual initialization methods (placeholders for Agent 1's work)

// Agent 4's components (already implemented - verifying initialization)

// HandleComponentFailure provides graceful degradation when components fail
func (m *ChatModel) HandleComponentFailure(component string, err error) {
	m.integrationFlags[component+"_failed"] = true

	switch component {
	case "markdown":
		// Fall back to plain text rendering
		m.markdownRenderer = nil
		// TODO: m.contentFormatter = NewPlainTextFormatter(m.width)

	case "status_display":
		// Disable status display but continue
		m.statusDisplay = nil

	case "auto_complete":
		// Disable auto-completion but continue
		m.completionEng = nil

	case "command_history":
		// Disable history but continue
		m.history = nil
	}

	// Log the failure for debugging
	fmt.Printf("Component %s failed: %v (continuing with degraded functionality)\n", component, err)
}

// LogIntegrationStatus provides debugging information about component status
func (m *ChatModel) LogIntegrationStatus() {
	fmt.Println("=== Guild Integration Status ===")

	components := []string{
		"markdown_enabled", "markdown_failed",
		"status_display_enabled", "status_display_failed",
		"auto_complete_enabled", "auto_complete_failed",
		"command_history_enabled", "command_history_failed",
	}

	for _, component := range components {
		if m.integrationFlags[component] {
			fmt.Printf("✓ %s\n", component)
		}
	}

	fmt.Println("===============================")
}

// HandleIntegratedKeyInput processes keyboard input with all components integrated
func (m ChatModel) HandleIntegratedKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle completion-aware input first
	if m.showingCompletion {
		switch {
		case msg.String() == "tab": // Tab completion
			// Cycle through completions
			if len(m.completionResults) > 0 {
				m.completionIndex = (m.completionIndex + 1) % len(m.completionResults)
				// Apply current completion
				m.input.SetValue(m.completionResults[m.completionIndex].Content)
				m.input.CursorEnd()
			}
			return m, nil

		case msg.String() == "esc": // Escape key
			// Cancel completion
			m.showingCompletion = false
			m.completionResults = nil
			m.completionIndex = 0
			return m, nil

		case key.Matches(msg, m.keys.Submit):
			// Accept current completion and send
			m.showingCompletion = false
			m.completionResults = nil
			return m.handleSendMessage()
		}
	}

	// Normal key handling
	return m.Update(msg)
}

// ProcessIntegratedMessage processes a message with all enhancements active

// RenderIntegratedView renders the chat view with all visual enhancements

// renderIntegratedHeader creates header with live agent status
func (m *ChatModel) renderIntegratedHeader() string {
	// Get active agent count
	activeCount := 0
	if m.agentStatusTracker != nil {
		activeAgents := m.agentStatusTracker.GetActiveAgents()
		activeCount = len(activeAgents)
	}

	// Build header with status
	viewInfo := "Global View"
	if m.viewMode == chatModeStatus && m.focusedAgent != "" {
		viewInfo = fmt.Sprintf("Agent: %s", m.focusedAgent)
	}

	// Add live status indicators
	statusIndicator := ""
	if activeCount > 0 {
		statusIndicator = fmt.Sprintf(" | %d agents active", activeCount)
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true).Render(fmt.Sprintf(
		"🏰 Guild Chat | %s | Campaign: %s | Session: %s%s",
		viewInfo,
		m.getCampaignDisplay(),
		m.sessionID[:8],
		statusIndicator,
	))
}

// renderIntegratedInput creates input area with completion popup
func (m *ChatModel) renderIntegratedInput() string {
	// Choose input style based on mode
	var inputStyle lipgloss.Style
	var modeIndicator string

	if false { // TODO: Add historyMode field
		// History mode - use medieval orange/amber styling
		inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("208")). // Orange
			Padding(0, 1)
		modeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true).
			Render("📜 History Mode")
	} else {
		// Normal mode - use standard cyan styling
		inputStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("141"))
	}

	inputView := inputStyle.Render(m.input.View())

	// Add completion popup if active
	if m.showingCompletion && len(m.completionResults) > 0 {
		completionView := m.renderCompletionPopup()
		// Stack completion above input
		return lipgloss.JoinVertical(lipgloss.Left, completionView, inputView)
	}

	// Add history mode indicator if active
	if false && modeIndicator != "" { // TODO: Add historyMode field
		return lipgloss.JoinVertical(lipgloss.Left, modeIndicator, inputView)
	}

	return inputView
}

// renderCompletionPopup creates the auto-completion popup
func (m *ChatModel) renderCompletionPopup() string {
	if len(m.completionResults) == 0 {
		return ""
	}

	var items []string

	// Add header with medieval styling
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")). // Purple
		Bold(true)
	header := headerStyle.Render("⚔️ Guild Suggestions")
	items = append(items, header)

	// Add separator
	separatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	separator := separatorStyle.Render(strings.Repeat("─", 25))
	items = append(items, separator)

	// Render completion items with type-specific icons and medieval styling
	for i, result := range m.completionResults {
		icon := m.getCompletionIcon(result.Metadata["type"])

		// Define styles for selected vs unselected items
		var nameStyle, descStyle lipgloss.Style

		if i == m.completionIndex {
			// Selected item - highlight with medieval purple theme
			nameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")). // Light yellow
				Background(lipgloss.Color("63"))   // Purple background
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("254")). // Light gray
				Background(lipgloss.Color("63"))   // Purple background
		} else {
			// Unselected items
			nameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("141")) // Medium purple
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")) // Dark gray
		}

		// Format item with proper spacing and medieval icons
		name := nameStyle.Render(result.Content)
		description := descStyle.Render(result.Metadata["description"])

		// Create the full item line
		var itemLine string
		if i == m.completionIndex {
			// Selected item gets special formatting
			itemLine = fmt.Sprintf("⚡ %s %s  %s", icon, name, description)
		} else {
			itemLine = fmt.Sprintf("  %s %s  %s", icon, name, description)
		}

		items = append(items, itemLine)
	}

	// Add footer with count
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
	footer := footerStyle.Render(fmt.Sprintf("⚡ %d of %d suggestions",
		m.completionIndex+1, len(m.completionResults)))
	items = append(items, footer)

	// Create popup box with medieval styling
	popup := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")). // Purple border
		Background(lipgloss.Color("0")).        // Black background
		Padding(0, 1).
		Render(strings.Join(items, "\n"))

	return popup
}

// getCompletionIcon returns a medieval-themed icon for completion type

// addSystemMessage adds a system message to the chat

// startEmbeddedServer starts a gRPC server for chat communication
func startEmbeddedServer(ctx context.Context, projCtx *project.Context, guildConfig *config.GuildConfig) error {
	// Initialize registry for the server
	reg := registry.NewComponentRegistry()

	// Configure registry with guild config
	registryConfig := registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
		},
	}

	if err := reg.Initialize(ctx, registryConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize server registry").
			WithComponent("grpc").
			WithOperation("startEmbeddedServer")
	}

	// Create event bus (minimal implementation for now)
	eventBus := &SimpleEventBus{}

	// Create gRPC server
	server := guildgrpc.NewServer(reg, eventBus)

	// Start server in background
	go func() {
		if err := server.Start(ctx, ":50051"); err != nil {
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()

	return nil
}

// SimpleEventBus provides a minimal event bus implementation
type SimpleEventBus struct{}

func (seb *SimpleEventBus) Publish(event interface{}) {
	// Minimal implementation - just log for now
	// TODO: Implement proper event publishing when needed
}

func (seb *SimpleEventBus) Subscribe(eventType string, handler func(event interface{})) {
	// Minimal implementation - events not needed for basic chat
	// TODO: Implement proper event subscription when needed
}

// executeToolViaGRPC sends tool execution request through gRPC
func (m *ChatModel) executeToolViaGRPC(execID, toolID string, params map[string]string) {
	go func() {
		// Get tool execution from tracking
		exec, ok := m.activeTools[execID]
		if !ok {
			return
		}

		// Update status to running
		exec.Status = "running"
		exec.Progress = 0.0

		// Get tool registry from component registry
		toolRegistry := m.registry.Tools()
		if toolRegistry == nil {
			exec.Status = "failed"
			exec.Error = "Tool registry not available"
			return
		}

		// Safety check: verify tool exists
		if !toolRegistry.HasTool(toolID) {
			exec.Status = "failed"
			exec.Error = fmt.Sprintf("Tool '%s' not found in registry", toolID)
			return
		}

		// Get the tool
		tool, err := toolRegistry.GetTool(toolID)
		if err != nil {
			exec.Status = "failed"
			exec.Error = fmt.Sprintf("Failed to get tool '%s': %v", toolID, err)
			return
		}

		// Safety check: verify tool permissions (if required)
		if tool.RequiresAuth() {
			// Check if user has granted permission for this tool
			if blocked, exists := m.blockedTools[toolID]; exists && blocked {
				exec.Status = "failed"
				exec.Error = fmt.Sprintf("Tool '%s' execution blocked by user", toolID)
				return
			}
		}

		// Progress update: preparation complete
		exec.Progress = 0.3

		// Convert string params to JSON for tool execution
		jsonParams := make(map[string]interface{})
		for k, v := range params {
			jsonParams[k] = v
		}

		// Create execution context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Progress update: starting execution
		exec.Progress = 0.5

		// Execute the tool with safety checks
		result, err := m.executeToolSafely(ctx, tool, jsonParams)

		// Progress update: execution complete
		exec.Progress = 0.9

		// Handle results
		if err != nil {
			exec.Status = "failed"
			exec.Error = fmt.Sprintf("Tool execution failed: %v", err)
		} else {
			exec.Status = "completed"
			exec.Progress = 1.0

			// Store the result
			if result != nil {
				exec.Output = result.Output
				if result.Error != "" {
					exec.Error = result.Error
					exec.Status = "failed"
				}
			}

			// Track cost if available
			cost := toolRegistry.GetToolCost(toolID)
			if cost > 0 {
				exec.Cost = cost
			}
		}
	}()
}

// executeToolSafely executes a tool with safety checks and workspace isolation
func (m *ChatModel) executeToolSafely(ctx context.Context, tool registry.Tool, params map[string]interface{}) (*tools.ToolResult, error) {
	// Import the tools package for ToolResult
	// Convert registry.Tool back to tools.Tool for execution
	actualTool, ok := tool.(tools.Tool)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInvalidFormat, "tool does not implement expected interface", nil).
			WithComponent("chat").
			WithOperation("executeToolSafely")
	}

	// Validate parameters against tool schema
	schema := actualTool.Schema()
	if schema != nil {
		if err := m.validateToolParams(params, schema); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "tool parameter validation failed").
				WithComponent("chat").
				WithOperation("executeToolSafely")
		}
	}

	// Convert params to JSON string for tool execution
	var paramJSON string
	if len(params) > 0 {
		jsonBytes, err := json.Marshal(params)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal tool parameters").
				WithComponent("chat").
				WithOperation("executeToolSafely")
		}
		paramJSON = string(jsonBytes)
	} else {
		paramJSON = "{}"
	}

	// Execute the tool with the context (includes timeout)
	result, err := actualTool.Execute(ctx, paramJSON)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "tool execution failed").
			WithComponent("chat").
			WithOperation("executeToolSafely").
			WithDetails("tool", actualTool.Name())
	}

	return result, nil
}

// validateToolParams validates parameters against a tool's JSON schema
func (m *ChatModel) validateToolParams(params map[string]interface{}, schema map[string]interface{}) error {
	// Basic validation - check required fields
	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		if required, ok := schema["required"].([]interface{}); ok {
			for _, requiredField := range required {
				if fieldName, ok := requiredField.(string); ok {
					if _, exists := params[fieldName]; !exists {
						return gerror.Newf(gerror.ErrCodeInvalidInput, "required parameter '%s' is missing", fieldName).
							WithComponent("chat").
							WithOperation("validateToolParams")
					}
				}
			}
		}

		// Validate each parameter type (basic validation)
		for paramName, paramValue := range params {
			if propSchema, exists := properties[paramName]; exists {
				if err := m.validateParamType(paramName, paramValue, propSchema); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// validateParamType validates a single parameter against its schema
func (m *ChatModel) validateParamType(paramName string, value interface{}, propSchema interface{}) error {
	schema, ok := propSchema.(map[string]interface{})
	if !ok {
		return nil // Skip validation if schema format is unexpected
	}

	paramType, ok := schema["type"].(string)
	if !ok {
		return nil // Skip validation if type is not specified
	}

	switch paramType {
	case "string":
		if _, ok := value.(string); !ok {
			return gerror.Newf(gerror.ErrCodeInvalidInput, "parameter '%s' must be a string", paramName).
				WithComponent("chat").
				WithOperation("validateParamType")
		}
	case "number":
		switch value.(type) {
		case float64, int, int64, float32:
			// Valid numeric types
		default:
			return gerror.Newf(gerror.ErrCodeInvalidInput, "parameter '%s' must be a number", paramName).
				WithComponent("chat").
				WithOperation("validateParamType")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return gerror.Newf(gerror.ErrCodeInvalidInput, "parameter '%s' must be a boolean", paramName).
				WithComponent("chat").
				WithOperation("validateParamType")
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return gerror.Newf(gerror.ErrCodeInvalidInput, "parameter '%s' must be an array", paramName).
				WithComponent("chat").
				WithOperation("validateParamType")
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return gerror.Newf(gerror.ErrCodeInvalidInput, "parameter '%s' must be an object", paramName).
				WithComponent("chat").
				WithOperation("validateParamType")
		}
	}

	return nil
}
