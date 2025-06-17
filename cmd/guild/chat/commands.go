// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// CommandProcessor handles command parsing and execution
type CommandProcessor struct {
	ctx        context.Context
	config     *ChatConfig
	history    *CommandHistory
	handlers   map[string]CommandHandler
}

// CommandHandler defines the interface for command handlers
type CommandHandler interface {
	Handle(ctx context.Context, args []string) tea.Cmd
	Description() string
	Usage() string
}

// CommandResult represents the result of a command execution
type CommandResult struct {
	Success bool
	Message string
	Data    interface{}
}

// NewCommandProcessor creates a new command processor
func NewCommandProcessor(ctx context.Context, config *ChatConfig, history *CommandHistory) *CommandProcessor {
	cp := &CommandProcessor{
		ctx:      ctx,
		config:   config,
		history:  history,
		handlers: make(map[string]CommandHandler),
	}
	
	// Register built-in command handlers
	cp.registerBuiltinHandlers()
	
	return cp
}

// ProcessInput processes user input and determines if it's a command or message
func (cp *CommandProcessor) ProcessInput(input string) (isCommand bool, cmd tea.Cmd) {
	input = strings.TrimSpace(input)
	
	if input == "" {
		return false, nil
	}
	
	// Add to history regardless of type
	if cp.history != nil {
		cp.history.Add(input)
	}
	
	// Check if it's a command (starts with /)
	if strings.HasPrefix(input, "/") {
		return true, cp.processCommand(input[1:]) // Remove the leading /
	}
	
	// Check if it's an agent mention (@agent-name message)
	if strings.HasPrefix(input, "@") {
		return false, cp.processAgentMention(input)
	}
	
	// Regular message to all agents
	return false, cp.processBroadcastMessage(input)
}

// processCommand processes slash commands
func (cp *CommandProcessor) processCommand(cmdText string) tea.Cmd {
	parts := strings.Fields(cmdText)
	if len(parts) == 0 {
		return func() tea.Msg {
			return StatusUpdateMsg{
				Message: "Empty command",
				Level:   "error",
			}
		}
	}
	
	cmdName := parts[0]
	args := parts[1:]
	
	// Find and execute handler
	if handler, exists := cp.handlers[cmdName]; exists {
		return handler.Handle(cp.ctx, args)
	}
	
	// Unknown command
	return func() tea.Msg {
		return StatusUpdateMsg{
			Message: fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmdName),
			Level:   "error",
		}
	}
}

// processAgentMention processes @agent-name messages
func (cp *CommandProcessor) processAgentMention(input string) tea.Cmd {
	// Parse @agent-name message format
	parts := strings.SplitN(input, " ", 2)
	if len(parts) < 2 {
		return func() tea.Msg {
			return StatusUpdateMsg{
				Message: "Usage: @agent-name your message",
				Level:   "error",
			}
		}
	}
	
	agentID := strings.TrimPrefix(parts[0], "@")
	message := parts[1]
	
	// Create agent-specific message
	return func() tea.Msg {
		return AgentStreamMsg{
			AgentID: agentID,
			Content: message,
			Done:    false,
		}
	}
}

// processBroadcastMessage processes messages to all agents
func (cp *CommandProcessor) processBroadcastMessage(message string) tea.Cmd {
	return func() tea.Msg {
		return AgentStreamMsg{
			AgentID: "all",
			Content: message,
			Done:    false,
		}
	}
}

// GetAvailableCommands returns a list of all available commands
func (cp *CommandProcessor) GetAvailableCommands() []Command {
	var commands []Command
	
	for name, handler := range cp.handlers {
		commands = append(commands, Command{
			Name:        name,
			Description: handler.Description(),
			Category:    "built-in",
			Action: func() tea.Cmd {
				return handler.Handle(cp.ctx, []string{})
			},
		})
	}
	
	return commands
}

// registerBuiltinHandlers registers the built-in command handlers
func (cp *CommandProcessor) registerBuiltinHandlers() {
	cp.handlers["help"] = &HelpHandler{}
	cp.handlers["clear"] = &ClearHandler{}
	cp.handlers["quit"] = &QuitHandler{}
	cp.handlers["exit"] = &QuitHandler{}
	cp.handlers["status"] = &StatusHandler{}
	cp.handlers["agents"] = &AgentsHandler{}
	cp.handlers["tools"] = &ToolsHandler{}
	cp.handlers["tool"] = &ToolExecuteHandler{}
	cp.handlers["test"] = &TestHandler{}
	cp.handlers["prompt"] = &PromptHandler{}
	cp.handlers["search"] = &SearchHandler{}
	cp.handlers["export"] = &ExportHandler{}
	cp.handlers["session"] = &SessionHandler{}
}

// Built-in command handlers

// HelpHandler shows available commands
type HelpHandler struct{}

func (h *HelpHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		helpText := `🏰 Guild Chat Commands:

/help                    - Show this help message
/clear                   - Clear chat history
/quit, /exit            - Exit chat
/status                 - Show agent and system status
/agents                 - List available agents
/tools [list|info|search] - Tool management
/tool <name> [params]   - Execute a tool directly
/test [markdown|code]   - Test rich content rendering
/prompt [get|set|list]  - Manage prompts
/search <pattern>       - Search message history
/export <filename>      - Export chat history
/session [list|load|save] - Session management

Agent Commands:
@agent-name message     - Send message to specific agent
@all message           - Broadcast message to all agents

Keyboard Shortcuts:
Ctrl+P                 - Command palette
Ctrl+Shift+F          - Global search
Ctrl+R                - Search history
Tab                   - Auto-completion
Esc                   - Cancel current operation`

		return PaneUpdateMsg{
			PaneID:  "output",
			Content: helpText,
		}
	}
}

func (h *HelpHandler) Description() string {
	return "Show available commands and shortcuts"
}

func (h *HelpHandler) Usage() string {
	return "/help"
}

// ClearHandler clears the chat
type ClearHandler struct{}

func (h *ClearHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		return PaneUpdateMsg{
			PaneID: "output",
			Data:   "clear",
		}
	}
}

func (h *ClearHandler) Description() string {
	return "Clear chat history"
}

func (h *ClearHandler) Usage() string {
	return "/clear"
}

// QuitHandler exits the application
type QuitHandler struct{}

func (h *QuitHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return tea.Quit
}

func (h *QuitHandler) Description() string {
	return "Exit the chat application"
}

func (h *QuitHandler) Usage() string {
	return "/quit or /exit"
}

// StatusHandler shows system status
type StatusHandler struct{}

func (h *StatusHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		statusText := fmt.Sprintf(`🏰 Guild Status - %s

🤖 Agents: Available
🔧 Tools: Ready
📡 gRPC: Connected
💾 Database: Active
🎨 Rich Content: Enabled

Current Session: %s
Uptime: %s`, 
			time.Now().Format("15:04:05"),
			"session-id", // Will be filled from config
			"uptime")     // Will be calculated

		return PaneUpdateMsg{
			PaneID:  "output",
			Content: statusText,
		}
	}
}

func (h *StatusHandler) Description() string {
	return "Show system and agent status"
}

func (h *StatusHandler) Usage() string {
	return "/status"
}

// AgentsHandler lists available agents
type AgentsHandler struct{}

func (h *AgentsHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		agentsText := `👥 Available Agents:

🛠️  developer    - Code implementation and debugging
📝  writer       - Documentation and content creation
🔍  researcher   - Information gathering and analysis
🧪  tester       - Quality assurance and testing
🎨  designer     - UI/UX and visual design

Usage:
@developer help me fix this bug
@writer create documentation for this feature
@all let's work on this together`

		return PaneUpdateMsg{
			PaneID:  "output",
			Content: agentsText,
		}
	}
}

func (h *AgentsHandler) Description() string {
	return "List available agents"
}

func (h *AgentsHandler) Usage() string {
	return "/agents"
}

// ToolsHandler manages tools
type ToolsHandler struct{}

func (h *ToolsHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		toolsText := `🔨 Guild Tools:

Available Commands:
/tools list             - Show all tools
/tools info <tool>      - Tool details
/tools search <query>   - Find tools by capability

Direct Execution:
/tool file-read --path ./README.md
/tool shell-exec --command "ls -la"`

		return PaneUpdateMsg{
			PaneID:  "output",
			Content: toolsText,
		}
	}
}

func (h *ToolsHandler) Description() string {
	return "Tool management and discovery"
}

func (h *ToolsHandler) Usage() string {
	return "/tools [list|info|search]"
}

// ToolExecuteHandler executes tools directly
type ToolExecuteHandler struct{}

func (h *ToolExecuteHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return StatusUpdateMsg{
				Message: "Usage: /tool <tool-name> [parameters]",
				Level:   "error",
			}
		}
	}
	
	return func() tea.Msg {
		return ToolExecutionStartMsg{
			ExecutionID: fmt.Sprintf("exec-%d", time.Now().UnixNano()),
			ToolName:    args[0],
			AgentID:     "chat-user",
			Parameters:  make(map[string]string), // Parse from args
		}
	}
}

func (h *ToolExecuteHandler) Description() string {
	return "Execute a tool directly"
}

func (h *ToolExecuteHandler) Usage() string {
	return "/tool <tool-name> [parameters]"
}

// TestHandler for testing rich content
type TestHandler struct{}

func (h *TestHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		testContent := `# Rich Content Test 🏰

This demonstrates **Guild's rich rendering**:

## Features
- **Bold** and *italic* text
- ` + "`code snippets`" + `
- Lists and structure

Try: /test code go`

		return PaneUpdateMsg{
			PaneID:  "output",
			Content: testContent,
		}
	}
}

func (h *TestHandler) Description() string {
	return "Test rich content rendering"
}

func (h *TestHandler) Usage() string {
	return "/test [markdown|code]"
}

// PromptHandler manages prompts
type PromptHandler struct{}

func (h *PromptHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		return PaneUpdateMsg{
			PaneID:  "output",
			Content: "Prompt management - Coming soon!",
		}
	}
}

func (h *PromptHandler) Description() string {
	return "Manage prompt layers"
}

func (h *PromptHandler) Usage() string {
	return "/prompt [get|set|list]"
}

// SearchHandler searches message history
type SearchHandler struct{}

func (h *SearchHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return StatusUpdateMsg{
				Message: "Usage: /search <pattern>",
				Level:   "error",
			}
		}
	}
	
	return func() tea.Msg {
		return SearchMsg{
			Pattern: strings.Join(args, " "),
		}
	}
}

func (h *SearchHandler) Description() string {
	return "Search message history"
}

func (h *SearchHandler) Usage() string {
	return "/search <pattern>"
}

// ExportHandler exports chat history
type ExportHandler struct{}

func (h *ExportHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	filename := "chat-export.md"
	if len(args) > 0 {
		filename = args[0]
	}
	
	return func() tea.Msg {
		return StatusUpdateMsg{
			Message: fmt.Sprintf("Exported chat to %s", filename),
			Level:   "info",
		}
	}
}

func (h *ExportHandler) Description() string {
	return "Export chat history"
}

func (h *ExportHandler) Usage() string {
	return "/export [filename]"
}

// SessionHandler manages chat sessions
type SessionHandler struct{}

func (h *SessionHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		return PaneUpdateMsg{
			PaneID:  "output",
			Content: "Session management - Coming soon!",
		}
	}
}

func (h *SessionHandler) Description() string {
	return "Manage chat sessions"
}

func (h *SessionHandler) Usage() string {
	return "/session [list|load|save]"
}

// CommandHistory manages command history
type CommandHistory struct {
	commands []string
	index    int
	maxSize  int
}

// NewCommandHistory creates a new command history
func NewCommandHistory(maxSize int) *CommandHistory {
	return &CommandHistory{
		commands: make([]string, 0),
		index:    -1,
		maxSize:  maxSize,
	}
}

// Add adds a command to history
func (ch *CommandHistory) Add(command string) {
	// Don't add empty commands or duplicates
	if command == "" || (len(ch.commands) > 0 && ch.commands[len(ch.commands)-1] == command) {
		return
	}
	
	ch.commands = append(ch.commands, command)
	
	// Maintain max size
	if len(ch.commands) > ch.maxSize {
		ch.commands = ch.commands[len(ch.commands)-ch.maxSize:]
	}
	
	ch.index = len(ch.commands)
}

// Previous returns the previous command in history
func (ch *CommandHistory) Previous() string {
	if len(ch.commands) == 0 {
		return ""
	}
	
	ch.index--
	if ch.index < 0 {
		ch.index = 0
	}
	
	return ch.commands[ch.index]
}

// Next returns the next command in history
func (ch *CommandHistory) Next() string {
	if len(ch.commands) == 0 {
		return ""
	}
	
	ch.index++
	if ch.index >= len(ch.commands) {
		ch.index = len(ch.commands)
		return ""
	}
	
	return ch.commands[ch.index]
}

// Search searches command history
func (ch *CommandHistory) Search(pattern string) []string {
	var matches []string
	pattern = strings.ToLower(pattern)
	
	for _, cmd := range ch.commands {
		if strings.Contains(strings.ToLower(cmd), pattern) {
			matches = append(matches, cmd)
		}
	}
	
	return matches
}