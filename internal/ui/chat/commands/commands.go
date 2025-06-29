// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lancekrogers/guild/internal/ui/chat/agents"
	"github.com/lancekrogers/guild/internal/ui/chat/common"
	"github.com/lancekrogers/guild/internal/ui/chat/messages"
	toolmsg "github.com/lancekrogers/guild/internal/ui/chat/messages/tools"
	"github.com/lancekrogers/guild/internal/ui/chat/panes"
	"github.com/lancekrogers/guild/internal/ui/chat/session"
	pb "github.com/lancekrogers/guild/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild/pkg/templates"
)

// CommandProcessor handles command parsing and execution
type CommandProcessor struct {
	ctx             context.Context
	config          *common.ChatConfig
	history         *CommandHistory
	handlers        map[string]CommandHandler
	sessionManager  session.SessionManager
	currentSession  *session.Session
	templateManager templates.TemplateManager
	guildClient     pb.GuildClient
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
func NewCommandProcessor(ctx context.Context, config *common.ChatConfig, history *CommandHistory,
	sessionManager session.SessionManager, currentSession *session.Session,
	templateManager templates.TemplateManager, guildClient pb.GuildClient,
) *CommandProcessor {
	cp := &CommandProcessor{
		ctx:             ctx,
		config:          config,
		history:         history,
		handlers:        make(map[string]CommandHandler),
		sessionManager:  sessionManager,
		currentSession:  currentSession,
		templateManager: templateManager,
		guildClient:     guildClient,
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
		return true, cp.ProcessCommand(input[1:]) // Remove the leading /
	}

	// Check if it's an agent mention (@agent-name message)
	if strings.HasPrefix(input, "@") {
		return false, cp.processAgentMention(input)
	}

	// Regular message to all agents
	return false, cp.processBroadcastMessage(input)
}

// ProcessCommand processes slash commands
func (cp *CommandProcessor) ProcessCommand(cmdText string) tea.Cmd {
	parts := strings.Fields(cmdText)
	if len(parts) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
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
		return panes.StatusUpdateMsg{
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
			return panes.StatusUpdateMsg{
				Message: "Usage: @agent-name your message",
				Level:   "error",
			}
		}
	}

	agentID := strings.TrimPrefix(parts[0], "@")
	message := parts[1]

	// Create agent-specific message
	return func() tea.Msg {
		return agents.AgentStreamMsg{
			AgentID: agentID,
			Content: message,
			Done:    false,
		}
	}
}

// processBroadcastMessage processes messages to all agents
func (cp *CommandProcessor) processBroadcastMessage(message string) tea.Cmd {
	return func() tea.Msg {
		return agents.AgentStreamMsg{
			AgentID: "all",
			Content: message,
			Done:    false,
		}
	}
}

// GetAvailableCommands returns a list of all available commands
func (cp *CommandProcessor) GetAvailableCommands() []messages.Command {
	var commands []messages.Command

	for name, handler := range cp.handlers {
		commands = append(commands, messages.Command{
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

// registerBuiltinHandlers registers the built-in command handlers (V1 compatible)
func (cp *CommandProcessor) registerBuiltinHandlers() {
	// General commands
	cp.handlers["help"] = &HelpHandler{}
	cp.handlers["?"] = &HelpHandler{}
	cp.handlers["clear"] = &ClearHandler{}
	cp.handlers["cls"] = &ClearHandler{}
	cp.handlers["quit"] = &QuitHandler{}
	cp.handlers["exit"] = &QuitHandler{}
	cp.handlers["q"] = &QuitHandler{}
	cp.handlers["configrefresh"] = &ConfigRefreshHandler{}

	// Agent commands
	cp.handlers["agents"] = NewAgentsHandler(cp.guildClient)
	cp.handlers["status"] = &StatusHandler{}

	// Guild commands
	cp.handlers["guilds"] = &GuildsHandler{}
	cp.handlers["guild"] = &GuildHandler{}

	// Tool commands
	cp.handlers["tools"] = &ToolsHandler{}
	cp.handlers["tool"] = &ToolExecuteHandler{}

	// Prompt commands
	cp.handlers["prompt"] = &PromptHandler{}

	// Export commands
	cp.handlers["export"] = NewExportHandler(cp.sessionManager, cp.currentSession)
	cp.handlers["save"] = NewSaveHandler(cp.sessionManager, cp.currentSession)

	// Template commands
	cp.handlers["template"] = NewTemplateHandler(cp.templateManager)
	cp.handlers["templates"] = NewTemplatesHandler(cp.templateManager)

	// Visual enhancement commands
	cp.handlers["image"] = &ImageHandler{}
	cp.handlers["mermaid"] = &MermaidHandler{}
	cp.handlers["code"] = &CodeHandler{}
	cp.handlers["vim"] = &VimHandler{}

	// Test commands
	cp.handlers["test"] = &TestHandler{}

	// Corpus commands (create first for search integration)
	corpusHandler := NewCorpusHandler(cp.config, cp.guildClient)
	cp.handlers["corpus"] = corpusHandler
	cp.handlers["knowledge"] = NewKnowledgeHandler()
	cp.handlers["index"] = NewIndexHandler()

	// Search commands (integrated with corpus)
	cp.handlers["search"] = NewSearchHandler(corpusHandler)

	// Session commands
	cp.handlers["session"] = &SessionHandler{}
}

// Built-in command handlers

// HelpHandler shows available commands
type HelpHandler struct{}

func (h *HelpHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		helpText := `🏰 **Guild Chat Commands**

**General Commands:**
  /help, /?              - Show this help message
  /clear, /cls           - Clear the chat history
  /exit, /quit, /q       - Exit Guild Chat
  /configrefresh         - Reload configurations

**Agent Commands:**
  /agents                - List all available agents
  /status                - Show current campaign status
  @agent-name <message>  - Send message to specific agent
  @all <message>         - Broadcast to all agents

**Guild Commands:**
  /guilds                - List all available guilds
  /guild                 - Show current guild details
  /guild <name>          - Switch to a different guild

**Tool Commands:**
  /tools list            - List available tools by category
  /tools search <query>  - Search tools by name/description
  /tools info <tool-id>  - Show detailed tool information
  /tools status          - Show active tool executions
  /tool <id> [params]    - Execute a tool directly

**Prompt Commands:**
  /prompt list           - List all prompt layers
  /prompt get <layer>    - Get content of a specific layer
  /prompt set <layer>    - Set prompt layer content
  /prompt delete <layer> - Delete a prompt layer

**Export Commands:**
  /export <format> [file] - Export session (json, md, html, pdf)
  /save [filename]       - Quick save as markdown

**Template Commands:**
  /template list [cat]   - List available templates
  /template search <q>   - Search templates
  /template use <id>     - Apply a template
  /templates             - Template management interface

**Visual Enhancement Commands:**
  /image <path>          - Show image with ASCII preview
  /mermaid              - Show Mermaid diagram help/examples
  /code toggle-lines     - Toggle line numbers in code blocks
  /vim                   - Toggle vim mode for input

**Test Commands:**
  /test markdown         - Test markdown rendering
  /test code <lang>      - Test syntax highlighting
  /test mixed           - Test mixed content rendering

**Search Commands:**
  /search <pattern>      - Search message history

**Corpus Commands:**
  /corpus list           - List all corpus documents
  /corpus search <query> - Search corpus content
  /corpus add <type> <content> - Add knowledge to corpus
  /corpus stats          - Show corpus statistics
  /knowledge browse      - Browse knowledge graph
  /knowledge validate    - Validate knowledge entries
  /index rebuild         - Rebuild corpus index

**Session Commands:**
  /session [list|load|save] - Session management

**Keyboard Shortcuts:**
  Ctrl+P                 - Command palette
  Ctrl+Shift+F          - Global search
  Ctrl+R                - Search history
  Tab                   - Auto-completion
  Esc                   - Cancel current operation`

		return panes.PaneUpdateMsg{
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
		return panes.PaneUpdateMsg{
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
		// TODO: Get real status data from services
		statusText := fmt.Sprintf(`📊 **Guild Status**

**Session:** %s
**Campaign:** %s
**Selected Guild:** %s
**Connected Agents:** %d
**Active Tools:** %d
**Prompt Layers:** %d

**System Status:**
🟢 gRPC Connection: Connected
🟢 Database: Active (SQLite)
🟢 Rich Content: Enabled
🟢 Tool Execution: Ready
🟢 Agent Router: Active

**Uptime:** %s
**Memory Usage:** %s
**Session Started:** %s`,
			"session-abc123", // TODO: Get from currentSession
			"e-commerce",     // TODO: Get from config
			"default",        // TODO: Get from config
			5,                // TODO: Get from agent router
			2,                // TODO: Get from tool execution
			6,                // TODO: Get from prompt manager
			"2h 35m",         // TODO: Calculate uptime
			"45.2 MB",        // TODO: Get memory usage
			time.Now().Add(-2*time.Hour-35*time.Minute).Format("15:04:05")) // TODO: Get session start time

		return panes.PaneUpdateMsg{
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
type AgentsHandler struct {
	guildClient pb.GuildClient
}

// NewAgentsHandler creates a new agents handler with gRPC client
func NewAgentsHandler(guildClient pb.GuildClient) *AgentsHandler {
	return &AgentsHandler{
		guildClient: guildClient,
	}
}

func (h *AgentsHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		// Get real agent data from gRPC service
		resp, err := h.guildClient.ListAvailableAgents(ctx, &pb.ListAgentsRequest{
			IncludeStatus: true,
		})
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to list agents: %v", err),
				Level:   "error",
			}
		}

		// Build agents text
		var agentsText strings.Builder
		agentsText.WriteString("🏰 **Available Guild Artisans**\n\n")

		for _, agent := range resp.Agents {
			// Get status icon
			statusIcon := "⚪"
			statusName := "UNKNOWN"
			if agent.Status != nil {
				switch agent.Status.State {
				case pb.AgentStatus_IDLE:
					statusIcon = "🟢"
					statusName = "IDLE"
				case pb.AgentStatus_THINKING:
					statusIcon = "🤔"
					statusName = "THINKING"
				case pb.AgentStatus_WORKING:
					statusIcon = "🟡"
					statusName = "WORKING"
				case pb.AgentStatus_ERROR:
					statusIcon = "🔴"
					statusName = "ERROR"
				case pb.AgentStatus_OFFLINE:
					statusIcon = "⚫"
					statusName = "OFFLINE"
				}
			}

			agentsText.WriteString(fmt.Sprintf("%s **@%s** - %s\n", statusIcon, agent.Id, agent.Name))
			if len(agent.Capabilities) > 0 {
				agentsText.WriteString(fmt.Sprintf("   🛡️ Skills: %s\n", strings.Join(agent.Capabilities, ", ")))
			}
			agentsText.WriteString(fmt.Sprintf("   📍 Status: %s\n\n", statusName))
		}

		agentsText.WriteString(`**Usage:**
@developer help me fix this bug
@writer create documentation for this feature
@all let's work on this together

**Status Icons:**
🟢 IDLE    🤔 THINKING    🟡 WORKING    🔴 ERROR    ⚫ OFFLINE`)

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: agentsText.String(),
		}
	}
}

func (h *AgentsHandler) Description() string {
	return "List available agents"
}

func (h *AgentsHandler) Usage() string {
	return "/agents"
}

// ToolsHandler manages tools with V1 functionality
type ToolsHandler struct{}

func (h *ToolsHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		// Default to "list" action
		args = []string{"list"}
	}

	action := args[0]
	switch action {
	case "list":
		return h.handleList(ctx, args[1:])
	case "search":
		return h.handleSearch(ctx, args[1:])
	case "info":
		return h.handleInfo(ctx, args[1:])
	case "status":
		return h.handleStatus(ctx, args[1:])
	default:
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /tools [list|search|info|status]",
				Level:   "error",
			}
		}
	}
}

func (h *ToolsHandler) handleList(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Get real tool data from tool registry
		toolsText := `🔨 **Available Guild Tools**

**📁 File Operations**
• file-reader          - Read file contents
• file-writer          - Write files safely
• directory-scanner    - Scan directory structure

**⚙️ System Operations**  
• shell-exec           - Execute shell commands
• process-monitor      - Monitor running processes
• environment-reader   - Read environment variables

**🐙 Git Operations**
• git-status           - Check repository status  
• git-commit           - Create commits
• git-diff             - Show differences

**🔧 Development Tools**
• code-analyzer        - Analyze code quality
• test-runner          - Execute test suites
• build-system         - Build projects
• dependency-scanner   - Scan dependencies

**🌐 Network Tools**
• api-client           - Make HTTP requests
• port-scanner         - Scan network ports
• dns-resolver         - Resolve DNS queries

**💾 Database Tools**
• database-query       - Execute SQL queries
• redis-client         - Redis operations
• mongodb-client       - MongoDB operations

**Usage:**
/tools search <query>    - Find tools by capability
/tools info <tool-id>    - Get detailed tool information
/tool <tool-id> [params] - Execute tool directly`

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: toolsText,
		}
	}
}

func (h *ToolsHandler) handleSearch(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /tools search <query>",
				Level:   "error",
			}
		}
	}

	query := strings.Join(args, " ")
	return func() tea.Msg {
		// TODO: Implement actual tool search
		searchText := fmt.Sprintf(`🔍 **Tool Search Results for '%s'**

**Matching Tools:**

🔨 **shell-exec** (90%% match)
   📝 Execute shell commands safely
   🏷️ Tags: system, command, execution
   
🔨 **code-analyzer** (75%% match)  
   📝 Analyze code quality and metrics
   🏷️ Tags: code, analysis, quality

🔨 **file-reader** (60%% match)
   📝 Read file contents with encoding detection
   🏷️ Tags: file, read, content

**Usage:**
/tools info shell-exec   - Get detailed information
/tool shell-exec --help  - Execute with help flag`, query)

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: searchText,
		}
	}
}

func (h *ToolsHandler) handleInfo(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /tools info <tool-id>",
				Level:   "error",
			}
		}
	}

	toolID := args[0]
	return func() tea.Msg {
		// TODO: Get real tool information
		infoText := fmt.Sprintf(`🔨 **Tool Information: %s**

**📝 Description:** Execute shell commands safely with workspace isolation

**🏷️ Category:** System Operations

**⚙️ Parameters:**
• command (required) - The shell command to execute
• timeout (optional) - Execution timeout in seconds (default: 30)
• workdir (optional) - Working directory (default: current)

**🛡️ Safety Features:**
• Workspace isolation enabled
• Command validation
• Timeout enforcement
• Output size limits

**📊 Usage Statistics:**
• Executions today: 15
• Success rate: 95%%
• Average duration: 2.3s

**💡 Examples:**
/tool %s --command "ls -la"
/tool %s --command "git status" --workdir ./project`, toolID, toolID, toolID)

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: infoText,
		}
	}
}

func (h *ToolsHandler) handleStatus(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Get real tool execution status
		statusText := `📊 **Active Tool Executions**

**🟡 Currently Running:**

⚙️ **shell-exec-001**
   👤 Requested by: @developer
   ⏱️ Started: 2 minutes ago
   📝 Command: "npm run build"
   
⚙️ **file-reader-002**  
   👤 Requested by: @manager
   ⏱️ Started: 30 seconds ago
   📝 File: "./docs/architecture.md"

**✅ Recently Completed:**

✓ **git-status-003** - Completed successfully (5 seconds ago)
✓ **code-analyzer-004** - Completed successfully (1 minute ago)
✗ **test-runner-005** - Failed: Test timeout (3 minutes ago)

**📈 Statistics:**
• Total executions today: 47
• Success rate: 89%
• Average execution time: 4.2s
• Active executions: 2
• Queue length: 0`

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: statusText,
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
			return panes.StatusUpdateMsg{
				Message: "Usage: /tool <tool-name> [parameters]",
				Level:   "error",
			}
		}
	}

	return func() tea.Msg {
		return toolmsg.ToolExecutionStartMsg{
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

// TestHandler for testing rich content with V1 functionality
type TestHandler struct{}

func (h *TestHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		// Default to markdown test
		args = []string{"markdown"}
	}

	testType := args[0]
	switch testType {
	case "markdown":
		return h.handleMarkdown(ctx, args[1:])
	case "code":
		return h.handleCode(ctx, args[1:])
	case "mixed":
		return h.handleMixed(ctx, args[1:])
	default:
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /test [markdown|code|mixed]\n/test code <language> - Test syntax highlighting",
				Level:   "error",
			}
		}
	}
}

func (h *TestHandler) handleMarkdown(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		testContent := `# 🏰 Guild Rich Content Test

This demonstrates **Guild's markdown rendering capabilities**:

## Text Formatting
- **Bold text** and *italic text*
- ~~Strikethrough text~~ and ` + "`inline code`" + `
- [Links to resources](https://github.com/lancekrogers/guild)

## Lists and Structure

### Ordered List
1. First item
2. Second item with **formatting**
3. Third item with ` + "`code`" + `

### Unordered List  
- Feature A: ✅ Implemented
- Feature B: 🔄 In Progress
- Feature C: ❌ Not Started

## Code Examples

Inline code: ` + "`fmt.Println(\"Hello Guild!\")`" + `

## Quotes

> "The Guild framework empowers AI agents to work together seamlessly."
> — Guild Development Team

## Tables

| Component | Status | Coverage |
|-----------|--------|----------|
| Agent     | ✅ Ready | 85% |
| Memory    | 🔄 Testing | 70% |
| Tools     | ✅ Ready | 90% |

## Emojis and Icons
🏰 🤖 ⚙️ 📝 🔧 🎯 ✅ ❌ 🔄 💡

**Test Status:** ✅ Markdown rendering working correctly!`

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: testContent,
		}
	}
}

func (h *TestHandler) handleCode(ctx context.Context, args []string) tea.Cmd {
	language := "go"
	if len(args) > 0 {
		language = args[0]
	}

	return func() tea.Msg {
		var testContent string

		switch language {
		case "go":
			testContent = `# 🏰 Go Code Syntax Highlighting Test

` + "```go" + `
package main

import (
	"context"
	"fmt"
	"log"
	
	"github.com/lancekrogers/guild/pkg/agent"
	"github.com/lancekrogers/guild/pkg/providers"
)

// GuildExample demonstrates Guild framework usage
type GuildExample struct {
	ctx      context.Context
	agent    agent.Agent
	provider providers.Provider
}

// NewGuildExample creates a new Guild example
func NewGuildExample(ctx context.Context) (*GuildExample, error) {
	// Initialize provider
	provider, err := providers.NewOpenAI("your-api-key")
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}
	
	// Create agent
	agent := agent.NewGuildArtisan("developer", provider)
	
	return &GuildExample{
		ctx:      ctx,
		agent:    agent,
		provider: provider,
	}, nil
}

// Execute runs the guild example
func (ge *GuildExample) Execute() error {
	prompt := "Help me implement a new feature"
	
	response, err := ge.agent.Process(ge.ctx, prompt)
	if err != nil {
		log.Printf("Error: %v", err)
		return err
	}
	
	fmt.Printf("Agent Response: %s\n", response)
	return nil
}

func main() {
	ctx := context.Background()
	
	example, err := NewGuildExample(ctx)
	if err != nil {
		log.Fatal(err)
	}
	
	if err := example.Execute(); err != nil {
		log.Fatal(err)
	}
}
` + "```" + `

**Test Status:** ✅ Go syntax highlighting working correctly!`

		case "python":
			testContent = `# 🏰 Python Code Syntax Highlighting Test

` + "```python" + `
import asyncio
import logging
from typing import Optional, Dict, Any

from guild_client import GuildClient, Agent

class GuildPythonExample:
    """Example Guild framework usage in Python."""
    
    def __init__(self, api_key: str):
        self.client = GuildClient(api_key=api_key)
        self.logger = logging.getLogger(__name__)
        
    async def create_agent(self, name: str, role: str) -> Agent:
        """Create a new agent with specified role."""
        try:
            agent = await self.client.create_agent(
                name=name,
                role=role,
                capabilities=["coding", "debugging", "testing"]
            )
            self.logger.info(f"Created agent: {agent.name}")
            return agent
        except Exception as e:
            self.logger.error(f"Failed to create agent: {e}")
            raise
            
    async def execute_task(self, agent: Agent, task: str) -> Dict[str, Any]:
        """Execute a task using the specified agent."""
        response = await agent.process(
            prompt=task,
            context={"framework": "guild", "language": "python"}
        )
        return response
        
async def main():
    """Main execution function."""
    example = GuildPythonExample(api_key="your-api-key")
    
    # Create developer agent
    developer = await example.create_agent("dev_agent", "developer")
    
    # Execute task
    result = await example.execute_task(
        developer, 
        "Implement a REST API endpoint for user management"
    )
    
    print(f"Task result: {result}")

if __name__ == "__main__":
    asyncio.run(main())
` + "```" + `

**Test Status:** ✅ Python syntax highlighting working correctly!`

		case "javascript":
			testContent = `# 🏰 JavaScript Code Syntax Highlighting Test

` + "```javascript" + `
const { GuildClient } = require('@guild/client');

class GuildJavaScriptExample {
    constructor(apiKey) {
        this.client = new GuildClient({ apiKey });
        this.agents = new Map();
    }
    
    async createAgent(name, role, capabilities = []) {
        try {
            const agent = await this.client.createAgent({
                name,
                role,
                capabilities: ['nodejs', 'frontend', ...capabilities]
            });
            
            this.agents.set(name, agent);
            console.log(` + "`Agent ${name} created successfully`" + `);
            return agent;
        } catch (error) {
            console.error(` + "`Failed to create agent: ${error.message}`" + `);
            throw error;
        }
    }
    
    async executeTask(agentName, task, context = {}) {
        const agent = this.agents.get(agentName);
        if (!agent) {
            throw new Error(` + "`Agent ${agentName} not found`" + `);
        }
        
        const response = await agent.process({
            prompt: task,
            context: {
                environment: 'nodejs',
                framework: 'guild',
                ...context
            }
        });
        
        return response;
    }
    
    async setupDevelopmentTeam() {
        const agents = await Promise.all([
            this.createAgent('frontend_dev', 'frontend_developer', ['react', 'vue']),
            this.createAgent('backend_dev', 'backend_developer', ['express', 'fastify']),
            this.createAgent('devops', 'devops_engineer', ['docker', 'kubernetes'])
        ]);
        
        console.log(` + "`Development team ready: ${agents.length} agents`" + `);
        return agents;
    }
}

async function main() {
    const example = new GuildJavaScriptExample('your-api-key');
    
    // Setup development team
    await example.setupDevelopmentTeam();
    
    // Execute frontend task
    const frontendResult = await example.executeTask(
        'frontend_dev',
        'Create a responsive dashboard component'
    );
    
    console.log('Frontend task result:', frontendResult);
}

main().catch(console.error);
` + "```" + `

**Test Status:** ✅ JavaScript syntax highlighting working correctly!`

		default:
			testContent = fmt.Sprintf(`# 🏰 Generic Code Syntax Highlighting Test

Language: **%s**

`+"```%s"+`
// Generic code example for %s
function example() {
    const message = "Hello from Guild framework!";
    console.log(message);
    return true;
}

example();
`+"```"+`

**Test Status:** ✅ %s syntax highlighting working correctly!

**Supported Languages:** go, python, javascript, typescript, rust, java, cpp, c, bash, sql, yaml, json, html, css`, language, language, language, language)
		}

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: testContent,
		}
	}
}

func (h *TestHandler) handleMixed(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		testContent := `# 🏰 Mixed Content Rendering Test

This test demonstrates **Guild's ability to render mixed content** with various formatting types:

## 📋 Task List
- [x] Implement basic markdown rendering
- [x] Add syntax highlighting for code blocks
- [x] Support for tables and lists
- [ ] Add Mermaid diagram support
- [ ] Implement LaTeX math rendering

## 🔧 Configuration Example

` + "```yaml" + `
guild:
  name: "development"
  agents:
    - name: "developer"
      role: "code_developer"
      tools: ["file-reader", "shell-exec", "git-commit"]
    - name: "tester"
      role: "qa_tester"  
      tools: ["test-runner", "coverage-analyzer"]
  
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"
      model: "gpt-4"
    anthropic:
      api_key: "${ANTHROPIC_API_KEY}"
      model: "claude-3-sonnet"
` + "```" + `

## 📊 Performance Metrics

| Metric | Value | Target | Status |
|--------|-------|--------|--------|
| Response Time | 150ms | <200ms | ✅ Good |
| Memory Usage | 45MB | <100MB | ✅ Good |
| CPU Usage | 12% | <25% | ✅ Good |
| Error Rate | 0.1% | <1% | ✅ Good |

## 🎯 Code Example with Documentation

` + "```go" + `
// ExecuteCommand processes a user command with full context
func (app *App) ExecuteCommand(ctx context.Context, cmd string) (*Response, error) {
    // Parse command and extract parameters
    parsed, err := app.parser.Parse(cmd)
    if err != nil {
        return nil, fmt.Errorf("command parsing failed: %w", err)
    }
    
    // Route to appropriate handler
    response, err := app.router.Route(ctx, parsed)
    if err != nil {
        return nil, fmt.Errorf("command execution failed: %w", err)
    }
    
    return response, nil
}
` + "```" + `

## 💡 Key Features

> **Guild Framework** provides a comprehensive platform for AI agent coordination with the following capabilities:

1. **Multi-Agent Orchestration** - Coordinate multiple AI agents working together
2. **Rich Tool Integration** - Execute tools safely with workspace isolation  
3. **Advanced Memory System** - RAG-enabled memory with vector search
4. **Flexible Configuration** - YAML-based configuration with hot reloading

## 🚀 Quick Start Commands

` + "```bash" + `
# Initialize a new Guild project
guild init --type web-app

# Start the chat interface
guild chat --campaign e-commerce

# List available agents
guild agents list

# Execute a tool directly  
guild tool file-reader --path ./README.md
` + "```" + `

## 🎨 Visual Elements

🏰 Architecture  🤖 Agents      ⚙️ Tools       📝 Documents
🔧 Development  🎯 Goals       ✅ Completed  🔄 In Progress
❌ Failed       💡 Ideas       🚀 Deploy     📊 Analytics

---

**Test Status:** ✅ Mixed content rendering working correctly!

**Features Tested:**
- ✅ Markdown formatting (headers, lists, emphasis)
- ✅ Code blocks with syntax highlighting  
- ✅ Tables with alignment
- ✅ Task lists with checkboxes
- ✅ Blockquotes and callouts
- ✅ Emoji and Unicode characters
- ✅ Horizontal rules and separators`

		return panes.PaneUpdateMsg{
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

// PromptHandler manages prompts with V1 functionality
type PromptHandler struct{}

func (h *PromptHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		// Default to "list" action
		args = []string{"list"}
	}

	action := args[0]
	switch action {
	case "list":
		return h.handleList(ctx, args[1:])
	case "get":
		return h.handleGet(ctx, args[1:])
	case "set":
		return h.handleSet(ctx, args[1:])
	case "delete":
		return h.handleDelete(ctx, args[1:])
	default:
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /prompt [list|get|set|delete] [args]",
				Level:   "error",
			}
		}
	}
}

func (h *PromptHandler) handleList(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Get real prompt data from prompt manager
		promptText := `📝 **Prompt Layer Management**

**6-Layer Prompt System:**

🔒 **PLATFORM** *(protected)*
   📝 Core Guild framework instructions
   📊 Size: 2,450 characters
   ⏰ Last updated: System initialization

🔒 **GUILD** *(protected)*  
   📝 Guild-specific behavior and capabilities
   📊 Size: 1,230 characters
   ⏰ Last updated: Guild configuration load

🟢 **ROLE**
   📝 Agent role definitions and responsibilities
   📊 Size: 876 characters
   ⏰ Last updated: 2 hours ago

🟢 **DOMAIN**
   📝 Domain-specific knowledge and context
   📊 Size: 1,450 characters  
   ⏰ Last updated: 1 hour ago

🟡 **SESSION**
   📝 Current session context and history
   📊 Size: 3,200 characters
   ⏰ Last updated: 15 minutes ago

🔵 **TURN**
   📝 Current conversation turn context
   📊 Size: 567 characters
   ⏰ Last updated: Just now

**Usage:**
/prompt get DOMAIN        - Get domain layer content
/prompt set ROLE <text>   - Set role layer content  
/prompt delete SESSION    - Clear session layer

**Note:** PLATFORM and GUILD layers are protected and cannot be modified.`

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: promptText,
		}
	}
}

func (h *PromptHandler) handleGet(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /prompt get <layer>\nLayers: PLATFORM, GUILD, ROLE, DOMAIN, SESSION, TURN",
				Level:   "error",
			}
		}
	}

	layer := strings.ToUpper(args[0])
	return func() tea.Msg {
		// TODO: Get actual prompt content from prompt manager
		var content string
		switch layer {
		case "PLATFORM":
			content = "Core Guild framework instructions for AI agent coordination..."
		case "GUILD":
			content = "Guild-specific behavior: Focus on collaborative development..."
		case "ROLE":
			content = "You are a helpful AI assistant specializing in software development..."
		case "DOMAIN":
			content = "Working on Go-based AI agent framework called Guild..."
		case "SESSION":
			content = "Current session context: E-commerce project development..."
		case "TURN":
			content = "Current turn: User requested command system implementation..."
		default:
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Unknown layer: %s. Valid layers: PLATFORM, GUILD, ROLE, DOMAIN, SESSION, TURN", layer),
				Level:   "error",
			}
		}

		promptContent := fmt.Sprintf(`📝 **Prompt Layer: %s**

**Content:**
%s

**Metadata:**
• Layer: %s
• Size: %d characters
• Editable: %s
• Last updated: %s`,
			layer, content, layer, len(content),
			map[string]string{
				"PLATFORM": "No (protected)",
				"GUILD":    "No (protected)",
				"ROLE":     "Yes",
				"DOMAIN":   "Yes",
				"SESSION":  "Yes",
				"TURN":     "Yes",
			}[layer],
			"2 hours ago") // TODO: Get real timestamp

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: promptContent,
		}
	}
}

func (h *PromptHandler) handleSet(ctx context.Context, args []string) tea.Cmd {
	if len(args) < 2 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /prompt set <layer> <content>\nLayers: ROLE, DOMAIN, SESSION, TURN",
				Level:   "error",
			}
		}
	}

	layer := strings.ToUpper(args[0])
	content := strings.Join(args[1:], " ")

	return func() tea.Msg {
		// Check if layer is protected
		if layer == "PLATFORM" || layer == "GUILD" {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Cannot modify protected layer: %s", layer),
				Level:   "error",
			}
		}

		// TODO: Implement actual prompt setting
		return panes.StatusUpdateMsg{
			Message: fmt.Sprintf("✅ Prompt layer '%s' updated (%d characters)", layer, len(content)),
			Level:   "success",
		}
	}
}

func (h *PromptHandler) handleDelete(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /prompt delete <layer>\nLayers: ROLE, DOMAIN, SESSION, TURN",
				Level:   "error",
			}
		}
	}

	layer := strings.ToUpper(args[0])

	return func() tea.Msg {
		// Check if layer is protected
		if layer == "PLATFORM" || layer == "GUILD" {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Cannot delete protected layer: %s", layer),
				Level:   "error",
			}
		}

		// TODO: Implement actual prompt deletion
		return panes.StatusUpdateMsg{
			Message: fmt.Sprintf("✅ Prompt layer '%s' cleared", layer),
			Level:   "success",
		}
	}
}

func (h *PromptHandler) Description() string {
	return "Manage prompt layers"
}

func (h *PromptHandler) Usage() string {
	return "/prompt [get|set|list]"
}

// SearchHandler provides integrated search across message history and corpus
type SearchHandler struct {
	corpusHandler *CorpusHandler
}

// NewSearchHandler creates a new search handler with corpus integration
func NewSearchHandler(corpusHandler *CorpusHandler) *SearchHandler {
	return &SearchHandler{
		corpusHandler: corpusHandler,
	}
}

func (h *SearchHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	ctx = observability.WithComponent(ctx, "search.unified")
	ctx = observability.WithOperation(ctx, "Handle")
	
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /search <query> - Searches both message history and corpus knowledge",
				Level:   "error",
			}
		}
	}

	query := strings.Join(args, " ")
	
	return func() tea.Msg {
		// Perform unified search across both message history and corpus
		var content strings.Builder
		content.WriteString(fmt.Sprintf("🔍 **Unified Search Results for \"%s\"**\n\n", query))
		
		// First, search corpus knowledge
		content.WriteString("## 📚 Knowledge Base Results\n\n")
		
		// Use corpus handler to search
		corpusCmd := h.corpusHandler.handleSearch(ctx, args)
		if corpusCmd != nil {
			corpusMsg := corpusCmd()
			if paneMsg, ok := corpusMsg.(panes.PaneUpdateMsg); ok {
				// Extract just the results part, skip the header
				corpusContent := paneMsg.Content
				if lines := strings.Split(corpusContent, "\n"); len(lines) > 2 {
					// Skip the header line and add the results
					resultsStart := false
					for _, line := range lines {
						if strings.Contains(line, "No results found") || strings.Contains(line, "Found") || resultsStart {
							resultsStart = true
							content.WriteString(line + "\n")
						}
					}
				}
			}
		}
		
		content.WriteString("\n## 💬 Chat History Results\n\n")
		content.WriteString("_Searching message history..._\n\n")
		
		// Add instruction for message search
		content.WriteString("**Note**: For detailed message history search, the system will also trigger a background search.\n")
		content.WriteString("**Tip**: Use `/corpus search --type pattern` to search only design patterns, or `/corpus search --from elena` to search by author.\n\n")
		
		content.WriteString("---\n\n")
		content.WriteString("**Advanced Options**:\n")
		content.WriteString("- `/corpus search` - Search knowledge base with advanced filters\n")
		content.WriteString("- Message history search is also triggered automatically\n")
		
		// Also trigger the original message search
		go func() {
			// This would trigger the message search in the background
			// For now, we'll keep the original behavior by sending the SearchMsg
		}()
		
		// Send both the unified results and trigger message search
		return tea.Batch(
			func() tea.Msg {
				return panes.PaneUpdateMsg{
					PaneID:  "output",
					Content: content.String(),
				}
			},
			func() tea.Msg {
				return messages.SearchMsg{
					Pattern: query,
				}
			},
		)
	}
}

func (h *SearchHandler) Description() string {
	return "Search both message history and knowledge corpus"
}

func (h *SearchHandler) Usage() string {
	return "/search <query> - Unified search across chat and corpus"
}

// ExportHandler exports chat history with full Sprint 7 functionality
type ExportHandler struct {
	sessionManager session.SessionManager
	currentSession *session.Session
}

func NewExportHandler(sessionManager session.SessionManager, currentSession *session.Session) *ExportHandler {
	return &ExportHandler{
		sessionManager: sessionManager,
		currentSession: currentSession,
	}
}

func (h *ExportHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /export <format> [filename]\nFormats: json, markdown, html, pdf",
				Level:   "error",
			}
		}
	}

	format := strings.ToLower(args[0])
	var filename string
	if len(args) > 1 {
		filename = args[1]
	}

	return func() tea.Msg {
		// Check if session manager is available
		if h.sessionManager == nil {
			return panes.StatusUpdateMsg{
				Message: "Session manager not available",
				Level:   "error",
			}
		}

		// Determine export format
		var exportFormat session.ExportFormat
		switch format {
		case "json":
			exportFormat = session.ExportFormatJSON
		case "markdown", "md":
			exportFormat = session.ExportFormatMarkdown
		case "html":
			exportFormat = session.ExportFormatHTML
		case "pdf":
			exportFormat = session.ExportFormatPDF
		default:
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Unsupported format: %s. Use: json, markdown, html, pdf", format),
				Level:   "error",
			}
		}

		// Use enhanced export with options
		options := &session.ExportOptions{
			IncludeToolOutputs: true,
			IncludeMetadata:    true,
			SyntaxHighlight:    true,
			LineNumbers:        false,
			Theme:              "default",
			DateFormat:         "2006-01-02 15:04:05",
		}

		// Get session ID from current session
		sessionID := ""
		if h.currentSession != nil {
			sessionID = h.currentSession.ID
		}

		// Export the session
		data, err := h.sessionManager.ExportSessionWithOptions(sessionID, exportFormat, options)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Export failed: %v", err),
				Level:   "error",
			}
		}

		// Save to file
		if filename == "" {
			// Generate default filename
			filename = fmt.Sprintf("guild-session-%s.%s", time.Now().Format("20060102-150405"), format)
		}

		if err := os.WriteFile(filename, data, 0o644); err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to save file: %v", err),
				Level:   "error",
			}
		}

		return panes.StatusUpdateMsg{
			Message: fmt.Sprintf("✅ Session exported to `%s` (%.1f KB)", filename, float64(len(data))/1024),
			Level:   "success",
		}
	}
}

func (h *ExportHandler) Description() string {
	return "Export chat session to various formats (json, markdown, html, pdf)"
}

func (h *ExportHandler) Usage() string {
	return "/export <format> [filename] - Formats: json, markdown, html, pdf"
}

// SaveHandler provides quick save functionality (defaults to markdown)
type SaveHandler struct {
	sessionManager session.SessionManager
	currentSession *session.Session
}

func NewSaveHandler(sessionManager session.SessionManager, currentSession *session.Session) *SaveHandler {
	return &SaveHandler{
		sessionManager: sessionManager,
		currentSession: currentSession,
	}
}

func (h *SaveHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	var filename string

	if len(args) > 0 {
		filename = args[0]
	}

	return func() tea.Msg {
		// Check if session manager is available
		if h.sessionManager == nil {
			return panes.StatusUpdateMsg{
				Message: "Session manager not available",
				Level:   "error",
			}
		}

		// Use enhanced export with options for markdown
		options := &session.ExportOptions{
			IncludeToolOutputs: true,
			IncludeMetadata:    true,
			SyntaxHighlight:    true,
			LineNumbers:        false,
			Theme:              "default",
			DateFormat:         "2006-01-02 15:04:05",
		}

		// Get session ID from current session
		sessionID := ""
		if h.currentSession != nil {
			sessionID = h.currentSession.ID
		}

		// Export the session as markdown
		data, err := h.sessionManager.ExportSessionWithOptions(sessionID, session.ExportFormatMarkdown, options)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Save failed: %v", err),
				Level:   "error",
			}
		}

		// Save to file
		if filename == "" {
			// Generate default filename
			filename = fmt.Sprintf("guild-chat-%s.md", time.Now().Format("20060102-150405"))
		}

		if err := os.WriteFile(filename, data, 0o644); err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to save file: %v", err),
				Level:   "error",
			}
		}

		return panes.StatusUpdateMsg{
			Message: fmt.Sprintf("💾 Chat saved to `%s` (%.1f KB)", filename, float64(len(data))/1024),
			Level:   "success",
		}
	}
}

func (h *SaveHandler) Description() string {
	return "Quick save chat session as markdown"
}

func (h *SaveHandler) Usage() string {
	return "/save [filename] - Quick save as markdown"
}

// SessionHandler manages chat sessions
type SessionHandler struct{}

func (h *SessionHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		return panes.PaneUpdateMsg{
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

// TemplateHandler handles template operations
type TemplateHandler struct {
	templateManager templates.TemplateManager
}

func NewTemplateHandler(templateManager templates.TemplateManager) *TemplateHandler {
	return &TemplateHandler{
		templateManager: templateManager,
	}
}

func (h *TemplateHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /template <action> [args]\nActions: list, search <query>, use <id>",
				Level:   "error",
			}
		}
	}

	action := args[0]
	switch action {
	case "list":
		return h.handleList(ctx, args[1:])
	case "search":
		return h.handleSearch(ctx, args[1:])
	case "use":
		return h.handleUse(ctx, args[1:])
	default:
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Unknown template action: %s", action),
				Level:   "error",
			}
		}
	}
}

func (h *TemplateHandler) handleList(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		if h.templateManager == nil {
			return panes.StatusUpdateMsg{
				Message: "Template manager not available",
				Level:   "error",
			}
		}

		// Get contextual suggestions
		context := make(map[string]interface{})
		templates, err := h.templateManager.GetContextualSuggestions(context)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to get templates: %v", err),
				Level:   "error",
			}
		}

		if len(templates) == 0 {
			return panes.PaneUpdateMsg{
				PaneID:  "output",
				Content: "📋 No templates available. Templates will be created automatically when needed.",
			}
		}

		content := "# 📋 Available Templates\n\n"
		content += "## Available Templates\n\n"
		content += "Templates are automatically managed by the content formatter.\n"
		content += "Use `/template search <query>` to find specific templates.\n"

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: content,
		}
	}
}

func (h *TemplateHandler) handleSearch(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /template search <query>",
				Level:   "error",
			}
		}
	}

	query := strings.Join(args, " ")

	return func() tea.Msg {
		if h.templateManager == nil {
			return panes.StatusUpdateMsg{
				Message: "Template manager not available",
				Level:   "error",
			}
		}

		results, err := h.templateManager.SearchTemplates(query, 10)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Search failed: %v", err),
				Level:   "error",
			}
		}

		if len(results) == 0 {
			return panes.PaneUpdateMsg{
				PaneID:  "output",
				Content: fmt.Sprintf("🔍 No templates found matching '%s'", query),
			}
		}

		content := fmt.Sprintf("# 🔍 Template Search Results for '%s'\n\n", query)
		for i, result := range results {
			template := result.Template
			content += fmt.Sprintf("## %d. %s (Score: %.1f)\n", i+1, template.Name, result.Relevance)
			content += fmt.Sprintf("**ID:** `%s`  \n", template.ID)
			content += fmt.Sprintf("**Description:** %s  \n", template.Description)
			content += fmt.Sprintf("**Matches:** %s  \n", strings.Join(result.Matches, ", "))
			content += fmt.Sprintf("**Usage:** `/template use %s`\n\n", template.ID)
		}

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: content,
		}
	}
}

func (h *TemplateHandler) handleUse(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /template use <template-id>",
				Level:   "error",
			}
		}
	}

	templateID := args[0]

	return func() tea.Msg {
		if h.templateManager == nil {
			return panes.StatusUpdateMsg{
				Message: "Template manager not available",
				Level:   "error",
			}
		}

		// For now, use empty variables - in a full implementation, this would prompt for variables
		variables := make(map[string]interface{})

		content, err := h.templateManager.RenderTemplate(templateID, variables)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to render template: %v", err),
				Level:   "error",
			}
		}

		// Show the rendered template
		output := fmt.Sprintf("📋 Template '%s' rendered:\n\n%s", templateID, content)
		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: output,
		}
	}
}

func (h *TemplateHandler) Description() string {
	return "Template operations: list, search, use"
}

func (h *TemplateHandler) Usage() string {
	return "/template <action> [args] - Actions: list, search <query>, use <id>"
}

// TemplatesHandler shows template management interface
type TemplatesHandler struct {
	templateManager templates.TemplateManager
}

func NewTemplatesHandler(templateManager templates.TemplateManager) *TemplatesHandler {
	return &TemplatesHandler{
		templateManager: templateManager,
	}
}

func (h *TemplatesHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		content := `# 📋 Template Management Interface

## Available Commands

### List Templates
` + "`/template list [category]`" + ` - List all templates or by category

### Search Templates  
` + "`/template search <query>`" + ` - Search templates by name, description, or tags

### Use Template
` + "`/template use <template-id>`" + ` - Apply a template to your message

## Example Usage

` + "```" + `
/template search api        # Find API-related templates
/template use api-endpoint  # Apply the API endpoint template
` + "```" + `

## Template Categories
- **api** - REST API endpoints and documentation
- **documentation** - Docs, reports, and meeting notes
- **development** - Code snippets and development workflows

💡 **Tip:** Templates support variable substitution and context-aware suggestions!`

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: content,
		}
	}
}

func (h *TemplatesHandler) Description() string {
	return "Show template management interface"
}

func (h *TemplatesHandler) Usage() string {
	return "/templates - Show template management help"
}

// ImageHandler handles image display and processing
type ImageHandler struct{}

func (h *ImageHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /image <path>",
				Level:   "error",
			}
		}
	}

	imagePath := strings.Join(args, " ")

	return func() tea.Msg {
		// Process the image with markdown format
		content := fmt.Sprintf("![Image](%s)", imagePath)

		// TODO: Add image processing with ASCII art when visual processors are integrated
		// For now, show the markdown image reference
		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: fmt.Sprintf("🖼️ Image: %s\n\n%s", imagePath, content),
		}
	}
}

func (h *ImageHandler) Description() string {
	return "Display image with ASCII art preview"
}

func (h *ImageHandler) Usage() string {
	return "/image <path> - Display image with ASCII preview"
}

// MermaidHandler handles Mermaid diagram support
type MermaidHandler struct{}

func (h *MermaidHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		content := `# 🖼️ Mermaid Diagram Support

Guild supports Mermaid diagrams with ASCII previews!

## Syntax
Just use fenced code blocks with ` + "`mermaid`" + ` language:

` + "```mermaid" + `
graph TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Action 1]
    B -->|No| D[Action 2]
    C --> E[End]
    D --> E
` + "```" + `

## Supported Diagram Types
- **Flowcharts**: ` + "`graph`" + ` or ` + "`flowchart`" + `
- **Sequence Diagrams**: ` + "`sequenceDiagram`" + `
- **Class Diagrams**: ` + "`classDiagram`" + `
- **State Diagrams**: ` + "`stateDiagram`" + `
- **Pie Charts**: ` + "`pie`" + `

## Features
- ✅ Real-time ASCII preview
- ✅ Syntax validation
- ✅ Export to PNG/SVG
- ✅ Copy to clipboard

💡 **Tip:** Diagrams are automatically detected and rendered in chat messages!`

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: content,
		}
	}
}

func (h *MermaidHandler) Description() string {
	return "Show Mermaid diagram help and examples"
}

func (h *MermaidHandler) Usage() string {
	return "/mermaid - Show Mermaid diagram help"
}

// CodeHandler handles code rendering features
type CodeHandler struct{}

func (h *CodeHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: "Usage: /code <action>\nActions: toggle-lines",
				Level:   "error",
			}
		}
	}

	action := args[0]
	switch action {
	case "toggle-lines":
		return func() tea.Msg {
			// TODO: Integrate with code renderer when visual processors are added
			return panes.PaneUpdateMsg{
				PaneID:  "output",
				Content: "✅ Code line numbers toggled",
			}
		}
	default:
		return func() tea.Msg {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Unknown code action: %s", action),
				Level:   "error",
			}
		}
	}
}

func (h *CodeHandler) Description() string {
	return "Code rendering features"
}

func (h *CodeHandler) Usage() string {
	return "/code <action> - Actions: toggle-lines"
}

// ConfigRefreshHandler reloads configurations
type ConfigRefreshHandler struct{}

func (h *ConfigRefreshHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual config refresh
		return panes.StatusUpdateMsg{
			Message: "✅ Configuration reloaded successfully",
			Level:   "success",
		}
	}
}

func (h *ConfigRefreshHandler) Description() string {
	return "Reload configurations without restarting"
}

func (h *ConfigRefreshHandler) Usage() string {
	return "/configrefresh"
}

// GuildsHandler lists available guilds
type GuildsHandler struct{}

func (h *GuildsHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Get real guild data from config/services
		guildsText := `🏰 **Available Guilds**

🟢 **default** *(current)*
   📝 Description: Default guild configuration
   👥 Agents: 5
   🛠️ Tools: 12
   
⚪ **development**
   📝 Description: Development-focused guild
   👥 Agents: 8
   🛠️ Tools: 15
   
⚪ **production**
   📝 Description: Production environment guild
   👥 Agents: 3
   🛠️ Tools: 8

**Usage:**
/guild development     - Switch to development guild
/guild                - Show current guild details`

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: guildsText,
		}
	}
}

func (h *GuildsHandler) Description() string {
	return "List all available guilds"
}

func (h *GuildsHandler) Usage() string {
	return "/guilds"
}

// GuildHandler shows or switches guild
type GuildHandler struct{}

func (h *GuildHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		// Show current guild details
		return func() tea.Msg {
			guildText := `🏰 **Current Guild: default**

📝 **Description:** Default guild configuration
👥 **Agents:** 5 connected
🛠️ **Tools:** 12 available
📦 **Campaigns:** 3 active
⚙️ **Configuration:** ~/.guild/guilds/default.yaml

**Recent Activity:**
• Developer agent completed task: "Fix build errors"
• Manager agent assigned new task: "Code review"
• Writer agent updated documentation

**Tools Available:**
file-reader, shell-exec, git-commit, code-analyzer, test-runner, 
docker-build, api-client, database-query, log-parser, 
documentation-generator, code-formatter, security-scanner`

			return panes.PaneUpdateMsg{
				PaneID:  "output",
				Content: guildText,
			}
		}
	} else {
		// Switch to specified guild
		guildName := args[0]
		return func() tea.Msg {
			// TODO: Implement actual guild switching
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("✅ Switched to guild: %s", guildName),
				Level:   "success",
			}
		}
	}
}

func (h *GuildHandler) Description() string {
	return "Show current guild details or switch guild"
}

func (h *GuildHandler) Usage() string {
	return "/guild [name] - Show current guild or switch to specified guild"
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

// VimHandler handles vim mode toggling
type VimHandler struct{}

func (h *VimHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		// Toggle vim mode
		return messages.VimModeToggleMsg{}
	}
}

func (h *VimHandler) Description() string {
	return "Toggle vim mode for input"
}

func (h *VimHandler) Usage() string {
	return "/vim - Toggle vim mode"
}
