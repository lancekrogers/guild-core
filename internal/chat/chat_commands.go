package chat

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/internal/chat/commands"
	"github.com/guild-ventures/guild-core/pkg/chat/session"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
)

// CommandProcessor handles command execution for the chat interface
type CommandProcessor struct {
	model      *ChatModel
	commands   map[string]func(args []string) tea.Cmd
	mu         sync.RWMutex
	suggestion *commands.SuggestionEngine
}

// NewCommandProcessor creates a new command processor
func NewCommandProcessor(model *ChatModel) *CommandProcessor {
	cp := &CommandProcessor{
		model:      model,
		commands:   make(map[string]func(args []string) tea.Cmd),
		suggestion: commands.NewSuggestionEngine(),
	}

	// Register all commands
	cp.registerCommands()

	// Update suggestion engine with available commands
	commandList := make([]string, 0, len(cp.commands))
	for cmd := range cp.commands {
		commandList = append(commandList, cmd)
	}
	cp.suggestion.UpdateCommands(commandList)

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
	
	// Export commands
	cp.RegisterCommand("export", cp.handleExport)
	cp.RegisterCommand("save", cp.handleExport)
	
	// Template commands
	cp.RegisterCommand("template", cp.handleTemplate)
	cp.RegisterCommand("templates", cp.handleTemplates)
	
	// Visual enhancement commands
	cp.RegisterCommand("image", cp.handleImage)
	cp.RegisterCommand("mermaid", cp.handleMermaid)
	cp.RegisterCommand("code", cp.handleCode)
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
				Content: cp.suggestion.GetSmartErrorMessage("unknown_command", cmd),
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

**Export Commands:**
  /export <format>       - Export session (json, markdown, html, pdf)
  /export <format> <file> - Export to specific file
  /save                  - Quick save to markdown

**Template Commands:**
  /template list         - List available templates
  /template search <query> - Search templates
  /template use <id>     - Apply template
  /templates             - Show template management interface

**Visual Commands:**
  /image <path>          - Show image with ASCII preview
  /mermaid               - Render Mermaid diagram
  /code toggle-lines     - Toggle line numbers in code blocks

**Test Commands:**
  /test markdown         - Test markdown rendering
  /test code <language>  - Test syntax highlighting
  /test mixed            - Test mixed content rendering

**Keyboard Shortcuts:**
  - Enter: Send message
  - Shift+Enter: Insert new line
  - Ctrl+Q: Quit application  
  - Ctrl+Alt+V: Toggle vim mode
  - Ctrl+Shift+C/V: Copy/Paste
  - Tab: Auto-completion
  - Ctrl+O: Fuzzy file finder
  - Ctrl+Shift+F: Global search
  - Ctrl+R: Search command history
  - Ctrl+P: Prompt management interface
  - Ctrl+A: Agent status view

**Image Support:**
  - Drag & drop images or paste paths for previews
  - Support for PNG, JPG, GIF, SVG formats`

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
			// Update suggestion engine with valid layers
			cp.suggestion.UpdatePromptLayers([]string{"PLATFORM", "GUILD", "ROLE", "DOMAIN", "SESSION", "TURN"})
			return Message{
				Type:    msgError,
				Content: cp.suggestion.GetSmartErrorMessage("unknown_layer", layer),
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
	return func() tea.Msg {
		// For now, simulate prompt setting
		// TODO: Implement actual gRPC call to set prompt layer

		// Validate layer
		validLayers := []string{"PLATFORM", "GUILD", "ROLE", "DOMAIN", "SESSION", "TURN"}
		layerUpper := strings.ToUpper(layer)

		isValid := false
		for _, valid := range validLayers {
			if layerUpper == valid {
				isValid = true
				break
			}
		}

		if !isValid {
			cp.suggestion.UpdatePromptLayers(validLayers)
			return Message{
				Type:    msgError,
				Content: cp.suggestion.GetSmartErrorMessage("unknown_layer", layer),
			}
		}

		// Simulate success
		return Message{
			Type: msgSystem,
			Content: fmt.Sprintf("✅ **Prompt Layer Updated**\n\nLayer: %s\nContent: %s\n\n*Note: This is a simulation. Actual gRPC implementation pending.*",
				layerUpper, content),
		}
	}
}

func (cp *CommandProcessor) handlePromptDelete(layer string) tea.Cmd {
	return func() tea.Msg {
		// For now, simulate prompt deletion
		// TODO: Implement actual gRPC call to delete prompt layer

		// Validate layer
		validLayers := []string{"PLATFORM", "GUILD", "ROLE", "DOMAIN", "SESSION", "TURN"}
		layerUpper := strings.ToUpper(layer)

		isValid := false
		for _, valid := range validLayers {
			if layerUpper == valid {
				isValid = true
				break
			}
		}

		if !isValid {
			cp.suggestion.UpdatePromptLayers(validLayers)
			return Message{
				Type:    msgError,
				Content: cp.suggestion.GetSmartErrorMessage("unknown_layer", layer),
			}
		}

		// Check if it's a protected layer
		protectedLayers := []string{"PLATFORM", "GUILD"}
		for _, protected := range protectedLayers {
			if layerUpper == protected {
				return Message{
					Type:    msgError,
					Content: fmt.Sprintf("❌ Cannot delete protected layer '%s'. Only SESSION and TURN layers can be deleted.", layerUpper),
				}
			}
		}

		// Simulate success
		return Message{
			Type:    msgSystem,
			Content: fmt.Sprintf("✅ **Prompt Layer Deleted**\n\nLayer: %s has been removed.\n\n*Note: This is a simulation. Actual gRPC implementation pending.*", layerUpper),
		}
	}
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
	return func() tea.Msg {
		// For now, return a list of commonly available tools
		// TODO: Integrate with actual tool registry when available
		toolsList := `🔧 **Available Guild Tools**

**File Operations:**
  📄 **file_read** - Read contents of a file
  📝 **file_write** - Write or create files
  📁 **file_list** - List directory contents

**Code Analysis:**
  🔍 **code_search** - Search for patterns in code
  📊 **code_analyze** - Analyze code structure
  🧪 **test_run** - Execute test suites

**Version Control:**
  🌿 **git_status** - Check repository status
  📦 **git_commit** - Create commits
  🔄 **git_diff** - View changes

**Development Tools:**
  🛠️ **build** - Build the project
  🚀 **deploy** - Deploy to environments
  📋 **lint** - Run code linters

Use /tool <name> to execute a specific tool.
Use /tools search <query> to find tools by capability.`

		return Message{
			Type:    msgSystem,
			Content: toolsList,
		}
	}
}

func (cp *CommandProcessor) handleToolsSearch(query string) tea.Cmd {
	return func() tea.Msg {
		// Simulate tool search functionality
		// TODO: Integrate with actual tool registry search

		// Define mock tools for search
		tools := []struct {
			name        string
			category    string
			description string
			icon        string
		}{
			{"file_read", "File Operations", "Read contents of a file", "📄"},
			{"file_write", "File Operations", "Write or create files", "📝"},
			{"file_list", "File Operations", "List directory contents", "📁"},
			{"code_search", "Code Analysis", "Search for patterns in code", "🔍"},
			{"code_analyze", "Code Analysis", "Analyze code structure", "📊"},
			{"test_run", "Code Analysis", "Execute test suites", "🧪"},
			{"git_status", "Version Control", "Check repository status", "🌿"},
			{"git_commit", "Version Control", "Create commits", "📦"},
			{"git_diff", "Version Control", "View changes", "🔄"},
			{"build", "Development", "Build the project", "🛠️"},
			{"deploy", "Development", "Deploy to environments", "🚀"},
			{"lint", "Development", "Run code linters", "📋"},
		}

		// Search tools
		var matches []string
		queryLower := strings.ToLower(query)

		for _, tool := range tools {
			if strings.Contains(strings.ToLower(tool.name), queryLower) ||
				strings.Contains(strings.ToLower(tool.description), queryLower) ||
				strings.Contains(strings.ToLower(tool.category), queryLower) {
				matches = append(matches, fmt.Sprintf("%s **%s** - %s",
					tool.icon, tool.name, tool.description))
			}
		}

		// Build result message
		var content strings.Builder
		content.WriteString(fmt.Sprintf("🔍 **Tool Search Results for '%s'**\n\n", query))

		if len(matches) == 0 {
			content.WriteString("No tools found matching your query.\n")
			content.WriteString("Try searching for: file, code, git, build, test")
		} else {
			content.WriteString(fmt.Sprintf("Found %d matching tools:\n\n", len(matches)))
			for _, match := range matches {
				content.WriteString("  ")
				content.WriteString(match)
				content.WriteString("\n")
			}
			content.WriteString("\nUse /tool <name> to execute a specific tool.")
		}

		return Message{
			Type:    msgSystem,
			Content: content.String(),
		}
	}
}

func (cp *CommandProcessor) handleToolInfo(toolID string) tea.Cmd {
	return func() tea.Msg {
		// Simulate tool info functionality
		// TODO: Integrate with actual tool registry

		// Define mock tool information
		toolInfo := map[string]struct {
			name        string
			category    string
			description string
			usage       string
			parameters  []string
			examples    []string
			icon        string
		}{
			"file_read": {
				name:        "file_read",
				category:    "File Operations",
				description: "Read contents of a file",
				usage:       "/tool file_read --path <file_path>",
				parameters:  []string{"--path: Path to the file to read"},
				examples:    []string{"/tool file_read --path README.md", "/tool file_read --path src/main.go"},
				icon:        "📄",
			},
			"code_search": {
				name:        "code_search",
				category:    "Code Analysis",
				description: "Search for patterns in code",
				usage:       "/tool code_search --pattern <regex> --path <directory>",
				parameters:  []string{"--pattern: Regular expression to search", "--path: Directory to search in"},
				examples:    []string{"/tool code_search --pattern 'func.*Error' --path ./pkg"},
				icon:        "🔍",
			},
			"git_status": {
				name:        "git_status",
				category:    "Version Control",
				description: "Check repository status",
				usage:       "/tool git_status",
				parameters:  []string{},
				examples:    []string{"/tool git_status"},
				icon:        "🌿",
			},
		}

		// Look up tool
		tool, exists := toolInfo[toolID]
		if !exists {
			// Update suggestion engine and return error
			toolNames := make([]string, 0, len(toolInfo))
			for name := range toolInfo {
				toolNames = append(toolNames, name)
			}
			cp.suggestion.UpdateTools(toolNames)

			return Message{
				Type:    msgError,
				Content: cp.suggestion.GetSmartErrorMessage("unknown_tool", toolID),
			}
		}

		// Build detailed info
		var content strings.Builder
		content.WriteString(fmt.Sprintf("%s **Tool Information: %s**\n\n", tool.icon, tool.name))
		content.WriteString(fmt.Sprintf("**Category:** %s\n", tool.category))
		content.WriteString(fmt.Sprintf("**Description:** %s\n\n", tool.description))

		content.WriteString("**Usage:**\n```\n")
		content.WriteString(tool.usage)
		content.WriteString("\n```\n\n")

		if len(tool.parameters) > 0 {
			content.WriteString("**Parameters:**\n")
			for _, param := range tool.parameters {
				content.WriteString(fmt.Sprintf("  • %s\n", param))
			}
			content.WriteString("\n")
		}

		if len(tool.examples) > 0 {
			content.WriteString("**Examples:**\n")
			for _, example := range tool.examples {
				content.WriteString(fmt.Sprintf("  ```\n  %s\n  ```\n", example))
			}
		}

		content.WriteString("\n💡 **Tip:** Use tab completion after /tool for available tools.")

		return Message{
			Type:    msgSystem,
			Content: content.String(),
		}
	}
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

	// Get tool registry from the model
	toolRegistry := cp.model.registry.Tools()
	if toolRegistry == nil {
		return cp.errorMessage("Tool registry not available")
	}

	// Safety check: verify tool exists
	if !toolRegistry.HasTool(toolID) {
		return cp.errorMessage(fmt.Sprintf("Tool '%s' not found in registry", toolID))
	}

	// Get the tool
	tool, err := toolRegistry.GetTool(toolID)
	if err != nil {
		return cp.errorMessage(fmt.Sprintf("Failed to get tool '%s': %v", toolID, err))
	}

	// Safety check: verify tool permissions (if required)
	if tool.RequiresAuth() {
		// Check if user has granted permission for this tool
		if blocked, exists := cp.model.blockedTools[toolID]; exists && blocked {
			return cp.errorMessage(fmt.Sprintf("Tool '%s' execution blocked by user", toolID))
		}
	}

	// Parse parameters from remaining args
	params := make(map[string]interface{})
	if len(args) > 1 {
		// Simple parameter parsing: key=value format
		for _, arg := range args[1:] {
			if strings.Contains(arg, "=") {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					params[parts[0]] = parts[1]
				}
			}
		}
	}

	// Execute tool directly (synchronously for command interface)
	return func() tea.Msg {
		// Create execution context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Execute the tool with safety checks
		result, err := cp.model.executeToolSafely(ctx, tool, params)
		if err != nil {
			return Message{
				Type:      msgSystem,
				Content:   fmt.Sprintf("❌ Tool execution failed: %v", err),
				AgentID:   "system",
				Timestamp: time.Now(),
			}
		}

		// Format successful result
		var content strings.Builder
		content.WriteString(fmt.Sprintf("🔨 Tool '%s' completed successfully\n\n", toolID))

		if result != nil {
			if result.Output != "" {
				content.WriteString("📤 Output:\n")
				content.WriteString(result.Output)
				content.WriteString("\n")
			}

			if len(result.Metadata) > 0 {
				content.WriteString("\n📊 Metadata:\n")
				for key, value := range result.Metadata {
					content.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
				}
			}

			if result.Error != "" {
				content.WriteString(fmt.Sprintf("\n⚠️ Tool reported error: %s\n", result.Error))
			}
		}

		// Track cost if available
		cost := toolRegistry.GetToolCost(toolID)
		if cost > 0 {
			content.WriteString(fmt.Sprintf("\n💰 Cost: %.2f credits\n", cost))
		}

		return Message{
			Type:      msgToolComplete,
			Content:   content.String(),
			AgentID:   "system",
			Timestamp: time.Now(),
		}
	}
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
            return gerror.Wrap(err, gerror.ErrCodeOperationFailed, fmt.Sprintf("step %d failed", i)).
                WithComponent("chat").
                WithOperation("GuildAgent.executeTask")
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

// Export command handlers

func (cp *CommandProcessor) handleExport(args []string) tea.Cmd {
	if len(args) == 0 {
		return cp.errorMessage("Usage: /export <format> [filename]\nFormats: json, markdown, html, pdf")
	}
	
	format := strings.ToLower(args[0])
	var filename string
	if len(args) > 1 {
		filename = args[1]
	}
	
	return func() tea.Msg {
		// Get session manager
		sessionManager := cp.model.sessionManager
		if sessionManager == nil {
			return Message{
				Type:    msgError,
				Content: "Session manager not available",
			}
		}
		
		// Export the session
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
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Unsupported format: %s. Use: json, markdown, html, pdf", format),
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
		
		data, err := sessionManager.ExportSessionWithOptions(cp.model.sessionID, exportFormat, options)
		if err != nil {
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Export failed: %v", err),
			}
		}
		
		// Save to file if filename provided
		if filename != "" {
			if err := os.WriteFile(filename, data, 0644); err != nil {
				return Message{
					Type:    msgError,
					Content: fmt.Sprintf("Failed to save file: %v", err),
				}
			}
			
			return Message{
				Type:    msgSystem,
				Content: fmt.Sprintf("✅ Session exported to `%s` (%.1f KB)", filename, float64(len(data))/1024),
			}
		} else {
			// Generate default filename
			defaultFilename := fmt.Sprintf("guild-session-%s.%s", time.Now().Format("20060102-150405"), format)
			if err := os.WriteFile(defaultFilename, data, 0644); err != nil {
				return Message{
					Type:    msgError,
					Content: fmt.Sprintf("Failed to save file: %v", err),
				}
			}
			
			return Message{
				Type:    msgSystem,
				Content: fmt.Sprintf("✅ Session exported to `%s` (%.1f KB)", defaultFilename, float64(len(data))/1024),
			}
		}
	}
}

// Template command handlers

func (cp *CommandProcessor) handleTemplate(args []string) tea.Cmd {
	if len(args) == 0 {
		return cp.errorMessage("Usage: /template <action> [args]\nActions: list, search <query>, use <id>")
	}
	
	action := args[0]
	switch action {
	case "list":
		return cp.handleTemplateList(args[1:])
	case "search":
		return cp.handleTemplateSearch(args[1:])
	case "use":
		return cp.handleTemplateUse(args[1:])
	default:
		return cp.errorMessage(fmt.Sprintf("Unknown template action: %s", action))
	}
}

func (cp *CommandProcessor) handleTemplates(args []string) tea.Cmd {
	// Show template management interface
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

		return Message{
			Type:    msgSystem,
			Content: content,
		}
	}
}

func (cp *CommandProcessor) handleTemplateList(args []string) tea.Cmd {
	return func() tea.Msg {
		formatter := cp.model.contentFormatter
		if formatter == nil {
			return Message{
				Type:    msgError,
				Content: "Content formatter not available",
			}
		}
		
		// Get contextual suggestions (acts like a list)
		context := make(map[string]interface{})
		templates, err := formatter.GetTemplateSuggestions(context)
		if err != nil {
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Failed to get templates: %v", err),
			}
		}
		
		if len(templates) == 0 {
			return Message{
				Type:    msgSystem,
				Content: "No templates available. Templates will be created automatically when needed.",
			}
		}
		
		var content strings.Builder
		content.WriteString("# 📋 Available Templates\n\n")
		
		// Show basic template listing
		content.WriteString("## Available Templates\n\n")
		content.WriteString("Templates are automatically managed by the content formatter.\n")
		content.WriteString("Use `/template search <query>` to find specific templates.\n")
		
		return Message{
			Type:    msgSystem,
			Content: content.String(),
		}
	}
}

func (cp *CommandProcessor) handleTemplateSearch(args []string) tea.Cmd {
	if len(args) == 0 {
		return cp.errorMessage("Usage: /template search <query>")
	}
	
	query := strings.Join(args, " ")
	
	return func() tea.Msg {
		formatter := cp.model.contentFormatter
		if formatter == nil {
			return Message{
				Type:    msgError,
				Content: "Content formatter not available",
			}
		}
		
		results, err := formatter.SearchTemplates(query, 10)
		if err != nil {
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Search failed: %v", err),
			}
		}
		
		if len(results) == 0 {
			return Message{
				Type:    msgSystem,
				Content: fmt.Sprintf("No templates found matching '%s'", query),
			}
		}
		
		var content strings.Builder
		content.WriteString(fmt.Sprintf("# 🔍 Template Search Results for '%s'\n\n", query))
		
		for i, result := range results {
			template := result.Template
			content.WriteString(fmt.Sprintf("## %d. %s (Score: %.1f)\n", i+1, template.Name, result.Relevance))
			content.WriteString(fmt.Sprintf("**ID:** `%s`  \n", template.ID))
			content.WriteString(fmt.Sprintf("**Description:** %s  \n", template.Description))
			content.WriteString(fmt.Sprintf("**Matches:** %s  \n", strings.Join(result.Matches, ", ")))
			content.WriteString(fmt.Sprintf("**Usage:** `/template use %s`\n\n", template.ID))
		}
		
		return Message{
			Type:    msgSystem,
			Content: content.String(),
		}
	}
}

func (cp *CommandProcessor) handleTemplateUse(args []string) tea.Cmd {
	if len(args) == 0 {
		return cp.errorMessage("Usage: /template use <template-id>")
	}
	
	templateID := args[0]
	
	return func() tea.Msg {
		formatter := cp.model.contentFormatter
		if formatter == nil {
			return Message{
				Type:    msgError,
				Content: "Content formatter not available",
			}
		}
		
		// For now, use empty variables - in a full implementation, this would prompt for variables
		variables := make(map[string]interface{})
		
		content, err := formatter.RenderTemplate(templateID, variables)
		if err != nil {
			return Message{
				Type:    msgError,
				Content: fmt.Sprintf("Failed to render template: %v", err),
			}
		}
		
		// Insert the rendered template into the input area
		// This would typically update the model's input field
		return Message{
			Type:    msgSystem,
			Content: fmt.Sprintf("📋 Template '%s' rendered:\n\n%s", templateID, content),
		}
	}
}

// Visual enhancement command handlers

func (cp *CommandProcessor) handleImage(args []string) tea.Cmd {
	if len(args) == 0 {
		return cp.errorMessage("Usage: /image <path>")
	}
	
	imagePath := strings.Join(args, " ")
	
	return func() tea.Msg {
		// Process the image using the content formatter
		content := fmt.Sprintf("![Image](%s)", imagePath)
		
		formatter := cp.model.contentFormatter
		if formatter != nil {
			content = formatter.processEnhancedContent(content)
		}
		
		return Message{
			Type:    msgSystem,
			Content: content,
		}
	}
}

func (cp *CommandProcessor) handleMermaid(args []string) tea.Cmd {
	return func() tea.Msg {
		// Show Mermaid help and example
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
- **Gantt Charts**: ` + "`gantt`" + `

## Features
- 📺 ASCII preview in terminal
- 🖼️ Full diagram generation (with mermaid-cli)
- 🎨 Automatic diagram type detection

💡 **Tip:** Install mermaid-cli for full diagram generation: ` + "`npm install -g @mermaid-js/mermaid-cli`" + ``

		return Message{
			Type:    msgSystem,
			Content: content,
		}
	}
}

func (cp *CommandProcessor) handleCode(args []string) tea.Cmd {
	if len(args) == 0 {
		return cp.errorMessage("Usage: /code <action>\nActions: toggle-lines")
	}
	
	action := args[0]
	switch action {
	case "toggle-lines":
		return func() tea.Msg {
			formatter := cp.model.contentFormatter
			if formatter != nil {
				formatter.ToggleCodeLineNumbers()
				return Message{
					Type:    msgSystem,
					Content: "✅ Code line numbers toggled",
				}
			}
			return Message{
				Type:    msgError,
				Content: "Content formatter not available",
			}
		}
	default:
		return cp.errorMessage(fmt.Sprintf("Unknown code action: %s", action))
	}
}
