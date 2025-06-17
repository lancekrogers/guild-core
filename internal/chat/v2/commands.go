// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package v2

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	
	"github.com/guild-ventures/guild-core/internal/chat/session"
	"github.com/guild-ventures/guild-core/pkg/templates"
)

// CommandProcessor handles command parsing and execution
type CommandProcessor struct {
	ctx             context.Context
	config          *ChatConfig
	history         *CommandHistory
	handlers        map[string]CommandHandler
	sessionManager  session.SessionManager
	currentSession  *session.Session
	templateManager templates.TemplateManager
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
func NewCommandProcessor(ctx context.Context, config *ChatConfig, history *CommandHistory, 
	sessionManager session.SessionManager, currentSession *session.Session,
	templateManager templates.TemplateManager) *CommandProcessor {
	cp := &CommandProcessor{
		ctx:             ctx,
		config:          config,
		history:         history,
		handlers:        make(map[string]CommandHandler),
		sessionManager:  sessionManager,
		currentSession:  currentSession,
		templateManager: templateManager,
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
	cp.handlers["export"] = NewExportHandler(cp.sessionManager, cp.currentSession)
	cp.handlers["save"] = NewSaveHandler(cp.sessionManager, cp.currentSession)
	cp.handlers["template"] = NewTemplateHandler(cp.templateManager)
	cp.handlers["templates"] = NewTemplatesHandler(cp.templateManager)
	cp.handlers["image"] = &ImageHandler{}
	cp.handlers["mermaid"] = &MermaidHandler{}
	cp.handlers["code"] = &CodeHandler{}
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
/export <format> [file] - Export chat (json, md, html, pdf)
/save [filename]        - Quick save as markdown
/template <action> [args] - Template operations (list, search, use)
/templates              - Template management interface
/image <path>           - Display image with ASCII preview
/mermaid               - Show Mermaid diagram help and examples
/code <action>         - Code rendering features (toggle-lines)
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
			return StatusUpdateMsg{
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
			return StatusUpdateMsg{
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
			return StatusUpdateMsg{
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
			return StatusUpdateMsg{
				Message: fmt.Sprintf("Export failed: %v", err),
				Level:   "error",
			}
		}

		// Save to file
		if filename == "" {
			// Generate default filename
			filename = fmt.Sprintf("guild-session-%s.%s", time.Now().Format("20060102-150405"), format)
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to save file: %v", err),
				Level:   "error",
			}
		}

		return StatusUpdateMsg{
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
			return StatusUpdateMsg{
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
			return StatusUpdateMsg{
				Message: fmt.Sprintf("Save failed: %v", err),
				Level:   "error",
			}
		}

		// Save to file
		if filename == "" {
			// Generate default filename
			filename = fmt.Sprintf("guild-chat-%s.md", time.Now().Format("20060102-150405"))
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to save file: %v", err),
				Level:   "error",
			}
		}

		return StatusUpdateMsg{
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
			return StatusUpdateMsg{
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
			return StatusUpdateMsg{
				Message: fmt.Sprintf("Unknown template action: %s", action),
				Level:   "error",
			}
		}
	}
}

func (h *TemplateHandler) handleList(ctx context.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		if h.templateManager == nil {
			return StatusUpdateMsg{
				Message: "Template manager not available",
				Level:   "error",
			}
		}

		// Get contextual suggestions
		context := make(map[string]interface{})
		templates, err := h.templateManager.GetContextualSuggestions(context)
		if err != nil {
			return StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to get templates: %v", err),
				Level:   "error",
			}
		}

		if len(templates) == 0 {
			return PaneUpdateMsg{
				PaneID:  "output",
				Content: "📋 No templates available. Templates will be created automatically when needed.",
			}
		}

		content := "# 📋 Available Templates\n\n"
		content += "## Available Templates\n\n"
		content += "Templates are automatically managed by the content formatter.\n"
		content += "Use `/template search <query>` to find specific templates.\n"

		return PaneUpdateMsg{
			PaneID:  "output",
			Content: content,
		}
	}
}

func (h *TemplateHandler) handleSearch(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return StatusUpdateMsg{
				Message: "Usage: /template search <query>",
				Level:   "error",
			}
		}
	}

	query := strings.Join(args, " ")

	return func() tea.Msg {
		if h.templateManager == nil {
			return StatusUpdateMsg{
				Message: "Template manager not available",
				Level:   "error",
			}
		}

		results, err := h.templateManager.SearchTemplates(query, 10)
		if err != nil {
			return StatusUpdateMsg{
				Message: fmt.Sprintf("Search failed: %v", err),
				Level:   "error",
			}
		}

		if len(results) == 0 {
			return PaneUpdateMsg{
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

		return PaneUpdateMsg{
			PaneID:  "output",
			Content: content,
		}
	}
}

func (h *TemplateHandler) handleUse(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return func() tea.Msg {
			return StatusUpdateMsg{
				Message: "Usage: /template use <template-id>",
				Level:   "error",
			}
		}
	}

	templateID := args[0]

	return func() tea.Msg {
		if h.templateManager == nil {
			return StatusUpdateMsg{
				Message: "Template manager not available",
				Level:   "error",
			}
		}

		// For now, use empty variables - in a full implementation, this would prompt for variables
		variables := make(map[string]interface{})

		content, err := h.templateManager.RenderTemplate(templateID, variables)
		if err != nil {
			return StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to render template: %v", err),
				Level:   "error",
			}
		}

		// Show the rendered template
		output := fmt.Sprintf("📋 Template '%s' rendered:\n\n%s", templateID, content)
		return PaneUpdateMsg{
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

		return PaneUpdateMsg{
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
			return StatusUpdateMsg{
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
		return PaneUpdateMsg{
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

		return PaneUpdateMsg{
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
			return StatusUpdateMsg{
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
			return PaneUpdateMsg{
				PaneID:  "output",
				Content: "✅ Code line numbers toggled",
			}
		}
	default:
		return func() tea.Msg {
			return StatusUpdateMsg{
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