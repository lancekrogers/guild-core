package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
)

// CommandProcessor handles command execution for the chat interface
type CommandProcessor struct {
	model         *ChatModel
	commands      map[string]func(args []string) tea.Cmd
	mu            sync.RWMutex
}

// NewCommandProcessor creates a new command processor
func NewCommandProcessor(model *ChatModel) *CommandProcessor {
	cp := &CommandProcessor{
		model:    model,
		commands: make(map[string]func(args []string) tea.Cmd),
	}

	// Register all commands
	cp.registerCommands()
	return cp
}

// registerCommands registers all available commands
func (cp *CommandProcessor) registerCommands() {
	// Help command
	cp.RegisterCommand("help", cp.handleHelp)
	cp.RegisterCommand("?", cp.handleHelp)

	// Agent commands
	cp.RegisterCommand("agents", cp.handleAgents)
	cp.RegisterCommand("status", cp.handleStatus)

	// Prompt commands
	cp.RegisterCommand("prompt", cp.handlePrompt)

	// Tool commands
	cp.RegisterCommand("tools", cp.handleTools)
	cp.RegisterCommand("tool", cp.handleTool)

	// Test commands for rich content
	cp.RegisterCommand("test", cp.handleTest)

	// Exit commands
	cp.RegisterCommand("exit", cp.handleExit)
	cp.RegisterCommand("quit", cp.handleExit)
	cp.RegisterCommand("q", cp.handleExit)

	// Clear command
	cp.RegisterCommand("clear", cp.handleClear)
	cp.RegisterCommand("cls", cp.handleClear)
}

// RegisterCommand registers a new command handler
func (cp *CommandProcessor) RegisterCommand(name string, handler func(args []string) tea.Cmd) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.commands[name] = handler
}

// ProcessCommand processes a command and returns the appropriate tea.Cmd
func (cp *CommandProcessor) ProcessCommand(input string) tea.Cmd {
	// Parse command and arguments
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	// Remove leading slash if present
	cmd := strings.TrimPrefix(parts[0], "/")
	args := parts[1:]

	// Look up command handler
	cp.mu.RLock()
	handler, exists := cp.commands[cmd]
	cp.mu.RUnlock()

	if !exists {
		return func() tea.Msg {
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Unknown command: /%s. Type /help for available commands.", cmd),
			}
		}
	}

	// Execute command handler
	return handler(args)
}

// Command handlers

func (cp *CommandProcessor) handleHelp(args []string) tea.Cmd {
	helpText := `🏰 **Guild Chat Commands**

**General Commands:**
  /help, /?              - Show this help message
  /clear, /cls           - Clear the chat history
  /exit, /quit, /q       - Exit Guild Chat

**Agent Commands:**
  /agents                - List all available agents
  /status                - Show current campaign status
  @agent-name <message>  - Send message to specific agent
  @all <message>         - Broadcast to all agents

**Prompt Management:**
  /prompt list           - List all prompt layers
  /prompt get <layer>    - Get content of a prompt layer
  /prompt set <layer>    - Set content for a prompt layer
  /prompt delete <layer> - Delete a prompt layer

**Tool Commands:**
  /tools list            - List all available tools
  /tools search <query>  - Search for tools
  /tools info <tool-id>  - Get detailed tool information
  /tool <id> [params]    - Execute a tool directly

**Test Commands:**
  /test markdown         - Test markdown rendering
  /test code <language>  - Test syntax highlighting
  /test mixed            - Test mixed content rendering

**Tips:**
  - Use Tab for auto-completion
  - Use Ctrl+R to search command history
  - Use Ctrl+P for prompt management interface
  - Use Ctrl+A for agent status view`

	return func() tea.Msg {
		return Message{
			Type:    msgSystem,
			Content: helpText,
		}
	}
}

func (cp *CommandProcessor) handleAgents(args []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := cp.model.grpcClient.ListAvailableAgents(ctx, &pb.ListAgentsRequest{})
		if err != nil {
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Failed to list agents: %v", err),
			}
		}

		if len(resp.Agents) == 0 {
			return Message{
				Type:    msgSystem,
				Content: "No agents are currently available.",
			}
		}

		// Format agent list with medieval theming
		var content strings.Builder
		content.WriteString("🏰 **Available Guild Artisans**\n\n")

		for _, agent := range resp.Agents {
			statusIcon := "⚫" // Offline
			if agent.Status != nil {
				switch agent.Status.State {
				case pb.AgentStatus_IDLE:
					statusIcon = "🟢"
				case pb.AgentStatus_THINKING:
					statusIcon = "🤔"
				case pb.AgentStatus_WORKING:
					statusIcon = "🟡"
				case pb.AgentStatus_ERROR:
					statusIcon = "🔴"
				}
			}

			content.WriteString(fmt.Sprintf("%s **@%s** - %s\n", statusIcon, agent.Id, agent.Name))
			if len(agent.Capabilities) > 0 {
				content.WriteString(fmt.Sprintf("   🛡️ Skills: %s\n", strings.Join(agent.Capabilities, ", ")))
			}
			content.WriteString("\n")
		}

		return Message{
			Type:    msgSystem,
			Content: content.String(),
		}
	}
}

func (cp *CommandProcessor) handleStatus(args []string) tea.Cmd {
	return func() tea.Msg {
		// Show campaign and session status
		status := fmt.Sprintf(`📊 **Guild Status**

**Session:** %s
**Campaign:** %s
**Connected Agents:** %d
**Active Tools:** %d
**Prompt Layers:** %d

Use /agents to see detailed agent status.`,
			cp.model.sessionID,
			cp.model.campaignID,
			len(cp.model.agents),
			len(cp.model.activeTools),
			len(cp.model.promptLayers),
		)

		return Message{
			Type:    msgSystem,
			Content: status,
		}
	}
}

func (cp *CommandProcessor) handlePrompt(args []string) tea.Cmd {
	if len(args) == 0 {
		return cp.handleHelp([]string{"prompt"})
	}

	action := args[0]
	switch action {
	case "list":
		return cp.handlePromptList()
	case "get":
		if len(args) < 2 {
			return cp.errorMessage("Usage: /prompt get <layer>")
		}
		return cp.handlePromptGet(args[1])
	case "set":
		if len(args) < 2 {
			return cp.errorMessage("Usage: /prompt set <layer> <content>")
		}
		return cp.handlePromptSet(args[1], strings.Join(args[2:], " "))
	case "delete":
		if len(args) < 2 {
			return cp.errorMessage("Usage: /prompt delete <layer>")
		}
		return cp.handlePromptDelete(args[1])
	default:
		return cp.errorMessage(fmt.Sprintf("Unknown prompt action: %s", action))
	}
}

func (cp *CommandProcessor) handlePromptList() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := cp.model.promptsClient.ListPromptLayers(ctx, &promptspb.ListPromptLayersRequest{})
		if err != nil {
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Failed to list prompts: %v", err),
			}
		}

		if len(resp.Prompts) == 0 {
			return Message{
				Type:    msgSystem,
				Content: "No prompt layers configured.",
			}
		}

		var content strings.Builder
		content.WriteString("📜 **Prompt Layers**\n\n")

		for _, prompt := range resp.Prompts {
			layerName := prompt.Layer.String()
			content.WriteString(fmt.Sprintf("**%s** (Layer %d)\n", layerName, prompt.Layer))
			if prompt.ArtisanId != "" {
				content.WriteString(fmt.Sprintf("  Artisan: %s\n", prompt.ArtisanId))
			}
			content.WriteString(fmt.Sprintf("  Priority: %d\n", prompt.Priority))
			content.WriteString(fmt.Sprintf("  Version: %d\n", prompt.Version))
			content.WriteString("\n")
		}

		return Message{
			Type:    msgSystem,
			Content: content.String(),
		}
	}
}

func (cp *CommandProcessor) handlePromptGet(layer string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// Convert layer name to enum
		var layerEnum promptspb.PromptLayer
		switch strings.ToUpper(layer) {
		case "PLATFORM":
			layerEnum = promptspb.PromptLayer_PROMPT_LAYER_PLATFORM
		case "GUILD":
			layerEnum = promptspb.PromptLayer_PROMPT_LAYER_GUILD
		case "ROLE":
			layerEnum = promptspb.PromptLayer_PROMPT_LAYER_ROLE
		case "DOMAIN":
			layerEnum = promptspb.PromptLayer_PROMPT_LAYER_DOMAIN
		case "SESSION":
			layerEnum = promptspb.PromptLayer_PROMPT_LAYER_SESSION
		case "TURN":
			layerEnum = promptspb.PromptLayer_PROMPT_LAYER_TURN
		default:
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Unknown prompt layer: %s", layer),
			}
		}
		
		resp, err := cp.model.promptsClient.GetPromptLayer(ctx, &promptspb.GetPromptLayerRequest{
			Layer: layerEnum,
		})
		if err != nil {
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Failed to get prompt: %v", err),
			}
		}

		content := fmt.Sprintf("📜 **Prompt Layer: %s**\n\n```\n%s\n```",
			layer, resp.Prompt.Content)

		return Message{
			Type:    msgSystem,
			Content: content,
		}
	}
}

func (cp *CommandProcessor) handlePromptSet(layer, content string) tea.Cmd {
	// TODO: Implement prompt setting via gRPC
	return cp.errorMessage("Prompt setting not yet implemented")
}

func (cp *CommandProcessor) handlePromptDelete(layer string) tea.Cmd {
	// TODO: Implement prompt deletion via gRPC
	return cp.errorMessage("Prompt deletion not yet implemented")
}

func (cp *CommandProcessor) handleTools(args []string) tea.Cmd {
	if len(args) == 0 {
		args = []string{"list"}
	}

	action := args[0]
	switch action {
	case "list":
		return cp.handleToolsList()
	case "search":
		if len(args) < 2 {
			return cp.errorMessage("Usage: /tools search <query>")
		}
		return cp.handleToolsSearch(strings.Join(args[1:], " "))
	case "info":
		if len(args) < 2 {
			return cp.errorMessage("Usage: /tools info <tool-id>")
		}
		return cp.handleToolInfo(args[1])
	case "status":
		return cp.handleToolsStatus()
	default:
		return cp.errorMessage(fmt.Sprintf("Unknown tools action: %s", action))
	}
}

func (cp *CommandProcessor) handleToolsList() tea.Cmd {
	// TODO: Implement tool listing via registry
	return func() tea.Msg {
		return Message{
			Type:    msgSystem,
			Content: "🔧 **Available Tools**\n\nTool listing coming soon...",
		}
	}
}

func (cp *CommandProcessor) handleToolsSearch(query string) tea.Cmd {
	// TODO: Implement tool search
	return cp.errorMessage("Tool search not yet implemented")
}

func (cp *CommandProcessor) handleToolInfo(toolID string) tea.Cmd {
	// TODO: Implement tool info
	return cp.errorMessage("Tool info not yet implemented")
}

func (cp *CommandProcessor) handleToolsStatus() tea.Cmd {
	return func() tea.Msg {
		if len(cp.model.activeTools) == 0 {
			return Message{
				Type:    msgSystem,
				Content: "No tools are currently executing.",
			}
		}

		var content strings.Builder
		content.WriteString("⚙️ **Active Tool Executions**\n\n")

		for id, tool := range cp.model.activeTools {
			content.WriteString(fmt.Sprintf("**%s** (%s)\n", tool.ToolName, id))
			content.WriteString(fmt.Sprintf("  Agent: @%s\n", tool.AgentID))
			content.WriteString(fmt.Sprintf("  Status: %s\n", tool.Status))
			if tool.Progress > 0 {
				content.WriteString(fmt.Sprintf("  Progress: %.0f%%\n", tool.Progress*100))
			}
			content.WriteString("\n")
		}

		return Message{
			Type:    msgSystem,
			Content: content.String(),
		}
	}
}

func (cp *CommandProcessor) handleTool(args []string) tea.Cmd {
	if len(args) < 1 {
		return cp.errorMessage("Usage: /tool <tool-id> [parameters]")
	}

	toolID := args[0]
	_ = args[1:] // params - will be used when tool execution is implemented

	// TODO: Implement direct tool execution
	return cp.errorMessage(fmt.Sprintf("Direct tool execution for '%s' not yet implemented", toolID))
}

func (cp *CommandProcessor) handleTest(args []string) tea.Cmd {
	if len(args) == 0 {
		args = []string{"markdown"}
	}

	testType := args[0]
	switch testType {
	case "markdown":
		return cp.testMarkdown()
	case "code":
		lang := "go"
		if len(args) > 1 {
			lang = args[1]
		}
		return cp.testCode(lang)
	case "mixed":
		return cp.testMixed()
	default:
		return cp.errorMessage(fmt.Sprintf("Unknown test type: %s", testType))
	}
}

func (cp *CommandProcessor) testMarkdown() tea.Cmd {
	content := `# Markdown Rendering Test

This demonstrates Guild's **rich markdown rendering** capabilities.

## Features

- **Bold text** for emphasis
- *Italic text* for style
- ` + "`Code snippets`" + ` for technical content
- [Links](https://example.com) for references

### Lists

1. Numbered lists
2. With multiple items
   - Nested bullet points
   - For organization

### Quotes

> "The Guild stands ready to serve, with artisans skilled in every craft."
> 
> — Guild Master

### Tables

| Agent | Status | Specialization |
|-------|--------|----------------|
| Manager | 🟢 Online | Planning |
| Developer | 🟡 Busy | Implementation |
| Reviewer | 🟢 Online | Quality |`

	return func() tea.Msg {
		return Message{
			Type:    msgSystem,
			Content: content,
		}
	}
}

func (cp *CommandProcessor) testCode(language string) tea.Cmd {
	codeExamples := map[string]string{
		"go": `// Guild Agent Implementation
package agent

import (
    "context"
    "fmt"
)

type GuildArtisan struct {
    ID           string
    Name         string
    Capabilities []string
}

func (g *GuildArtisan) Execute(ctx context.Context, task Task) error {
    fmt.Printf("🛡️ %s executing task: %s\n", g.Name, task.ID)
    
    // Medieval-themed task execution
    for i, step := range task.Steps {
        if err := g.performStep(ctx, step); err != nil {
            return fmt.Errorf("step %d failed: %w", i, err)
        }
    }
    
    return nil
}`,
		"python": `# Guild Agent Implementation
import asyncio
from typing import List, Dict, Any

class GuildArtisan:
    """A skilled artisan in the Guild framework"""
    
    def __init__(self, id: str, name: str, capabilities: List[str]):
        self.id = id
        self.name = name
        self.capabilities = capabilities
        self.active_tasks = []
    
    async def execute_task(self, task: Dict[str, Any]) -> Dict[str, Any]:
        """Execute a task with medieval flair"""
        print(f"🛡️ {self.name} executing task: {task['id']}")
        
        # Simulate task execution
        for step in task.get('steps', []):
            await self._perform_step(step)
            await asyncio.sleep(0.1)
        
        return {
            'status': 'completed',
            'artisan': self.name,
            'task_id': task['id']
        }`,
		"javascript": `// Guild Agent Implementation
class GuildArtisan {
    constructor(id, name, capabilities) {
        this.id = id;
        this.name = name;
        this.capabilities = capabilities;
        this.activeTasks = new Map();
    }
    
    async executeTask(task) {
        console.log(` + "`🛡️ ${this.name} executing task: ${task.id}`" + `);
        
        // Medieval-themed task execution
        for (const [index, step] of task.steps.entries()) {
            try {
                await this.performStep(step);
            } catch (error) {
                throw new Error(` + "`Step ${index} failed: ${error.message}`" + `);
            }
        }
        
        return {
            status: 'completed',
            artisan: this.name,
            taskId: task.id
        };
    }
}`,
	}

	code, exists := codeExamples[language]
	if !exists {
		code = fmt.Sprintf("// No example available for %s\n// Try: go, python, or javascript", language)
		language = "text"
	}

	content := fmt.Sprintf("```%s\n%s\n```", language, code)

	return func() tea.Msg {
		return Message{
			Type:    msgSystem,
			Content: content,
		}
	}
}

func (cp *CommandProcessor) testMixed() tea.Cmd {
	content := `## Rich Content Demo

This demonstrates **mixed content rendering** with both markdown and code.

### Task Execution Flow

When a Guild artisan receives a task, the following process occurs:

` + "```go" + `
// Task execution pipeline
func (g *GuildOrchestrator) ExecuteCommission(commission Commission) error {
    // 1. Parse commission into tasks
    tasks := g.parseCommission(commission)
    
    // 2. Assign tasks to artisans
    for _, task := range tasks {
        artisan := g.selectArtisan(task)
        g.assignTask(artisan, task)
    }
    
    // 3. Monitor execution
    return g.monitorExecution()
}
` + "```" + `

### Visual Status Indicators

The Guild uses medieval-themed indicators:

- 🟢 **Online** - Artisan ready for tasks
- 🟡 **Busy** - Currently executing commission
- 🔴 **Error** - Requires guild master attention
- ⚫ **Offline** - Artisan unavailable

> **Note:** This rich rendering makes Guild superior to plain-text alternatives!`

	return func() tea.Msg {
		return Message{
			Type:    msgSystem,
			Content: content,
		}
	}
}

func (cp *CommandProcessor) handleExit(args []string) tea.Cmd {
	return tea.Quit
}

func (cp *CommandProcessor) handleClear(args []string) tea.Cmd {
	return func() tea.Msg {
		// Clear messages except for the welcome message
		if len(cp.model.messages) > 0 {
			cp.model.messages = cp.model.messages[:1]
		}
		return nil
	}
}

// Helper functions

func (cp *CommandProcessor) errorMessage(content string) tea.Cmd {
	return func() tea.Msg {
		return Message{
			Type:    msgError,
			Content: content,
		}
	}
}