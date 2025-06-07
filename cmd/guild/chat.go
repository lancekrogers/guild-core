package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/guild-ventures/guild-core/pkg/config"
	guildcontext "github.com/guild-ventures/guild-core/pkg/context"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	guildgrpc "github.com/guild-ventures/guild-core/pkg/grpc"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/spf13/cobra"
)

var (
	chatCampaignID string
	chatSessionID  string
	grpcAddress    string
)

// chatCmd represents the chat command group
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interactive chat with Guild agents",
	Long: `Launch an interactive chat interface to communicate with Guild agents.

The Guild Chat interface allows you to:
- Send messages to specific agents using @agent-name
- Broadcast to all agents using @all
- Manage layered prompts with /prompt commands
- View agent status and task progress
- Control campaign execution in real-time`,
	RunE: runChat,
}

func init() {
	// Add chat command to root
	rootCmd.AddCommand(chatCmd)

	// Add flags
	chatCmd.Flags().StringVar(&chatCampaignID, "campaign", "", "Campaign ID to connect to")
	chatCmd.Flags().StringVar(&chatSessionID, "session", "", "Chat session ID (auto-generated if not provided)")
	chatCmd.Flags().StringVar(&grpcAddress, "grpc-address", "localhost:50051", "gRPC server address")
}

// runChat launches the Guild Chat TUI
func runChat(cmd *cobra.Command, args []string) error {
	// Get project context
	projCtx, err := project.GetContext()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get project context").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Load guild configuration
	guildConfig, err := config.LoadGuildConfig(projCtx.GetRootPath())
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load guild config").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Generate session ID if not provided
	if chatSessionID == "" {
		chatSessionID = fmt.Sprintf("session-%d", time.Now().Unix())
	}

	// Start embedded gRPC server if using default address
	var serverCtx context.Context
	var serverCancel context.CancelFunc
	if grpcAddress == "localhost:50051" {
		serverCtx, serverCancel = context.WithCancel(context.Background())
		defer serverCancel() // Ensure cleanup happens regardless of early returns
		
		if err := startEmbeddedServer(serverCtx, projCtx, guildConfig); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start embedded gRPC server").
				WithComponent("cli").
				WithOperation("chat.run")
		}

		// Give server time to start
		time.Sleep(500 * time.Millisecond)
	}

	// Connect to gRPC server
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeConnection, "failed to connect to gRPC server").
			WithComponent("cli").
			WithOperation("chat.run").
			WithDetails("address", grpcAddress)
	}
	defer conn.Close()

	// Create gRPC clients
	guildClient := pb.NewGuildClient(conn)
	promptClient := promptspb.NewPromptServiceClient(conn)

	// Initialize registry
	reg := registry.NewComponentRegistry()
	registryConfig := registry.Config{
		Agents: registry.AgentConfigYaml{
			DefaultType: "worker",
		},
	}
	if err := reg.Initialize(context.Background(), registryConfig); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to initialize registry").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	// Initialize chat model
	model := newChatModel(guildConfig, chatCampaignID, chatSessionID, conn, guildClient, promptClient, reg)

	// Start the TUI
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to run chat interface").
			WithComponent("cli").
			WithOperation("chat.run")
	}

	return nil
}

// ChatModel represents the Guild Chat TUI state
type ChatModel struct {
	// Configuration
	guildConfig *config.GuildConfig
	campaignID  string
	sessionID   string

	// UI Components
	input    textarea.Model
	messages viewport.Model
	help     help.Model

	// State
	keymap       chatKeyMap
	width        int
	height       int
	ready        bool
	messageLog   []chatMessage
	currentAgent string                   // Current focused agent ("" = global view)
	agentStreams map[string][]chatMessage // agent ID -> messages
	globalStream []chatMessage            // All messages for global view
	chatMode     chatViewMode             // Current view mode
	promptLayers map[string]string        // layer -> content

	// Tool execution tracking
	toolExecutions map[string]*toolExecution // execution ID -> execution
	activeTools    []string                  // ordered list of active tool IDs

	// gRPC clients
	guildClient  pb.GuildClient
	promptClient promptspb.PromptServiceClient
	grpcConn     *grpc.ClientConn

	// Context and registry
	ctx      context.Context
	cancel   context.CancelFunc
	registry registry.ComponentRegistry

	// Medieval theming
	borderStyle lipgloss.Style
	headerStyle lipgloss.Style
	inputStyle  lipgloss.Style
	agentStyle  lipgloss.Style
	systemStyle lipgloss.Style
	errorStyle  lipgloss.Style
	toolStyle   lipgloss.Style

	// Rich content rendering
	markdownRenderer *MarkdownRenderer
	contentFormatter *ContentFormatter

	// Command completion and history
	completionEngine *CompletionEngine
	commandHistory   *CommandHistory
	commandProcessor *CommandProcessor

	// Completion state
	showingCompletion bool
	completionResults []CompletionResult
	completionIndex   int

	// Agent status system (Agent 3)
	statusTracker   *AgentStatusTracker
	statusDisplay   *StatusDisplay
	agentIndicators *AgentIndicators
	showAgentStatus bool

	// Integration state (Agent 4)
	demoMode         bool
	integrationFlags map[string]bool
}

// toolExecution tracks an ongoing tool execution
type toolExecution struct {
	ID         string
	ToolID     string
	ToolName   string
	AgentID    string
	StartTime  time.Time
	EndTime    *time.Time
	Status     string
	Progress   float64
	Parameters map[string]string
	Result     string
	Error      string
	Cost       float64
}

// chatMessage represents a message in the chat
type chatMessage struct {
	Timestamp time.Time
	Sender    string // "user", "system", or agent ID
	AgentID   string // Specific agent ID for agent-related messages
	Content   string
	Type      messageType
}

// chatViewMode represents the current chat view mode
type chatViewMode int

const (
	globalView chatViewMode = iota // View all agents and system messages
	agentView                      // View specific agent conversation
)

type messageType int

const (
	msgUser messageType = iota
	msgAgent
	msgSystem
	msgPrompt
	msgAgentThinking
	msgAgentWorking
	msgError
	msgToolStart
	msgToolProgress
	msgToolComplete
	msgToolError
	msgToolAuth
)

// Agent 2: Auto-Completion System Implementation

// CompletionResult represents a single completion suggestion
type CompletionResult struct {
	Text        string // The completion text
	Description string // Human-readable description
	Type        string // Type: "command", "agent", "file", etc.
}

// CompletionEngine provides intelligent command and agent auto-completion
type CompletionEngine struct {
	guildConfig *config.GuildConfig
	projectRoot string
	commands    map[string]Command
	taskIDs     []string // Cache of task IDs for completion
	registry    registry.ComponentRegistry // Access to kanban for task completion
}

// NewCompletionEngine creates a new completion engine with full functionality
func NewCompletionEngine(guildConfig *config.GuildConfig, projectRoot string) *CompletionEngine {
	engine := &CompletionEngine{
		guildConfig: guildConfig,
		projectRoot: projectRoot,
		commands:    make(map[string]Command),
	}

	// Register built-in commands with medieval theming
	engine.registerCommands()
	return engine
}

// registerCommands registers all available commands for completion
func (ce *CompletionEngine) registerCommands() {
	commands := []Command{
		{Name: "/help", Description: "Show available commands", Usage: "/help"},
		{Name: "/status", Description: "Show campaign status", Usage: "/status"},
		{Name: "/agents", Description: "List available agents", Usage: "/agents"},
		{Name: "/prompt", Description: "Manage layered prompts", Usage: "/prompt [get|set|list|delete]"},
		{Name: "/tools", Description: "Manage Guild tools", Usage: "/tools [list|info|search|status]"},
		{Name: "/tool", Description: "Execute a tool directly", Usage: "/tool <tool-id> [params]"},
		{Name: "/test", Description: "Test rich content features", Usage: "/test [markdown|code|mixed]"},
		{Name: "/exit", Description: "Exit Guild Chat", Usage: "/exit"},
		{Name: "/quit", Description: "Exit Guild Chat", Usage: "/quit"},
	}

	for _, cmd := range commands {
		ce.commands[cmd.Name] = cmd
	}
}

// Complete provides intelligent completion suggestions
func (ce *CompletionEngine) Complete(input string, cursorPos int) []CompletionResult {
	var results []CompletionResult

	// Handle different completion types based on input context
	if strings.HasPrefix(input, "/") {
		// Command completion
		results = append(results, ce.completeCommands(input)...)
	} else if strings.HasPrefix(input, "@") {
		// Agent mention completion
		results = append(results, ce.completeAgents(input)...)
	} else if strings.Contains(input, "--") {
		// Command argument completion
		results = append(results, ce.completeArguments(input)...)
	} else if strings.Contains(strings.ToLower(input), "task") {
		// Task ID completion when context suggests tasks
		results = append(results, ce.completeTaskIDs(input)...)
	} else {
		// File path completion or general text
		results = append(results, ce.completeFilePaths(input)...)
	}

	// If no results, provide helpful suggestions based on context
	if len(results) == 0 {
		results = ce.getHelpfulSuggestions(input)
	}

	return results
}

// completeCommands suggests command completions with smart sorting
func (ce *CompletionEngine) completeCommands(input string) []CompletionResult {
	var results []CompletionResult

	// First try exact prefix match
	for cmdName, cmd := range ce.commands {
		if strings.HasPrefix(cmdName, input) {
			results = append(results, CompletionResult{
				Text:        cmdName,
				Description: cmd.Description,
				Type:        "command",
			})
		}
	}

	// If no exact matches, try fuzzy matching
	if len(results) == 0 {
		for cmdName, cmd := range ce.commands {
			if fuzzyMatch(cmdName, input) {
				results = append(results, CompletionResult{
					Text:        cmdName,
					Description: cmd.Description,
					Type:        "command",
				})
			}
		}
	}

	// Sort by relevance (exact matches first, then by length)
	sort.Slice(results, func(i, j int) bool {
		// Exact prefix matches come first
		iExact := strings.HasPrefix(results[i].Text, input)
		jExact := strings.HasPrefix(results[j].Text, input)
		if iExact != jExact {
			return iExact
		}
		// Then sort by length (shorter = more relevant)
		return len(results[i].Text) < len(results[j].Text)
	})

	return results
}

// completeAgents suggests agent completions
func (ce *CompletionEngine) completeAgents(input string) []CompletionResult {
	var results []CompletionResult

	// Add @all for broadcast
	if fuzzyMatch("@all", input) {
		results = append(results, CompletionResult{
			Text:        "@all",
			Description: "Broadcast to all agents",
			Type:        "agent",
		})
	}

	// Add configured agents
	if ce.guildConfig != nil {
		for _, agent := range ce.guildConfig.Agents {
			agentMention := "@" + agent.ID
			if fuzzyMatch(agentMention, input) {
				results = append(results, CompletionResult{
					Text:        agentMention,
					Description: fmt.Sprintf("%s - %s", agent.Name, strings.Join(agent.Capabilities, ", ")),
					Type:        "agent",
				})
			}
		}
	}

	return results
}

// completeArguments suggests command argument completions
func (ce *CompletionEngine) completeArguments(input string) []CompletionResult {
	var results []CompletionResult

	// Common argument patterns
	args := []CompletionResult{
		{Text: "--path", Description: "File or directory path", Type: "argument"},
		{Text: "--layer", Description: "Prompt layer name", Type: "argument"},
		{Text: "--text", Description: "Text content", Type: "argument"},
		{Text: "--timeout", Description: "Timeout in seconds", Type: "argument"},
		{Text: "--command", Description: "Shell command to execute", Type: "argument"},
	}

	for _, arg := range args {
		if fuzzyMatch(arg.Text, strings.ToLower(input)) {
			results = append(results, arg)
		}
	}

	return results
}

// completeFilePaths suggests file path completions (basic implementation)
func (ce *CompletionEngine) completeFilePaths(input string) []CompletionResult {
	// For now, suggest common Guild paths
	paths := []CompletionResult{
		{Text: ".guild/", Description: "Guild configuration directory", Type: "path"},
		{Text: "README.md", Description: "Project readme file", Type: "file"},
		{Text: "guild.yaml", Description: "Guild configuration file", Type: "file"},
	}

	var results []CompletionResult
	for _, path := range paths {
		if fuzzyMatch(path.Text, input) {
			results = append(results, path)
		}
	}

	return results
}

// completeTaskIDs suggests task ID completions
func (ce *CompletionEngine) completeTaskIDs(input string) []CompletionResult {
	var results []CompletionResult

	// Mock task IDs for now - in real implementation would fetch from kanban
	mockTasks := []struct {
		id     string
		title  string
		status string
	}{
		{"BE-001", "Setup API Gateway", "in_progress"},
		{"BE-002", "Implement Auth Service", "todo"},
		{"BE-003", "Payment Integration", "blocked"},
		{"FE-001", "Design Landing Page", "review"},
		{"FE-002", "Build Shopping Cart", "done"},
	}

	// Extract any partial task ID from input
	words := strings.Fields(input)
	var taskPattern string
	for _, word := range words {
		if strings.Contains(strings.ToUpper(word), "BE-") || strings.Contains(strings.ToUpper(word), "FE-") {
			taskPattern = strings.ToUpper(word)
			break
		}
	}

	for _, task := range mockTasks {
		if taskPattern == "" || fuzzyMatch(task.id, taskPattern) {
			statusIcon := "📋"
			switch task.status {
			case "in_progress":
				statusIcon = "🔨"
			case "blocked":
				statusIcon = "🚫"
			case "review":
				statusIcon = "👀"
			case "done":
				statusIcon = "✅"
			}

			results = append(results, CompletionResult{
				Text:        task.id,
				Description: fmt.Sprintf("%s %s (%s)", statusIcon, task.title, task.status),
				Type:        "task",
			})
		}
	}

	return results
}

// getHelpfulSuggestions provides context-aware suggestions when no matches found
func (ce *CompletionEngine) getHelpfulSuggestions(input string) []CompletionResult {
	var suggestions []CompletionResult

	// Analyze input to provide helpful suggestions
	lowerInput := strings.ToLower(input)

	// Suggest commands based on keywords
	if strings.Contains(lowerInput, "help") {
		suggestions = append(suggestions, CompletionResult{
			Text:        "/help",
			Description: "Show available commands",
			Type:        "suggestion",
		})
	}

	if strings.Contains(lowerInput, "agent") || strings.Contains(lowerInput, "who") {
		suggestions = append(suggestions, CompletionResult{
			Text:        "/agents",
			Description: "List available agents",
			Type:        "suggestion",
		})
	}

	if strings.Contains(lowerInput, "status") || strings.Contains(lowerInput, "progress") {
		suggestions = append(suggestions, CompletionResult{
			Text:        "/status",
			Description: "Show campaign and agent status",
			Type:        "suggestion",
		})
	}

	if strings.Contains(lowerInput, "task") || strings.Contains(lowerInput, "work") {
		suggestions = append(suggestions, CompletionResult{
			Text:        "/tools status",
			Description: "Show active tool executions",
			Type:        "suggestion",
		})
	}

	// If still no suggestions, provide general help
	if len(suggestions) == 0 {
		suggestions = append(suggestions, CompletionResult{
			Text:        "/help",
			Description: "Type /help to see available commands",
			Type:        "suggestion",
		})
		suggestions = append(suggestions, CompletionResult{
			Text:        "@",
			Description: "Type @ to see available agents",
			Type:        "suggestion",
		})
	}

	return suggestions
}

// UpdateTaskCache updates the cached task IDs for completion
func (ce *CompletionEngine) UpdateTaskCache(taskIDs []string) {
	ce.taskIDs = taskIDs
}

// GetSmartAgentSuggestions provides context-aware agent suggestions based on message content
func (ce *CompletionEngine) GetSmartAgentSuggestions(messageContent string) []CompletionResult {
	var results []CompletionResult

	if ce.guildConfig == nil {
		return results
	}

	// Analyze message content for keywords
	lowerContent := strings.ToLower(messageContent)

	type agentScore struct {
		agent config.AgentConfig
		score int
		reason string
	}

	var scoredAgents []agentScore

	for _, agent := range ce.guildConfig.Agents {
		score := 0
		reasons := []string{}

		// Check capabilities match
		for _, capability := range agent.Capabilities {
			capLower := strings.ToLower(capability)
			if strings.Contains(lowerContent, capLower) {
				score += 10
				reasons = append(reasons, fmt.Sprintf("has %s capability", capability))
			}
		}

		// Check for related keywords
		if strings.Contains(lowerContent, "frontend") && agent.Type == "developer" && agentHasCapability(agent, "UI") {
			score += 5
			reasons = append(reasons, "frontend specialist")
		}

		if strings.Contains(lowerContent, "api") && agentHasCapability(agent, "API") {
			score += 5
			reasons = append(reasons, "API expert")
		}

		if strings.Contains(lowerContent, "database") && agentHasCapability(agent, "Database") {
			score += 5
			reasons = append(reasons, "database specialist")
		}

		if score > 0 {
			reason := strings.Join(reasons, ", ")
			scoredAgents = append(scoredAgents, agentScore{
				agent: agent,
				score: score,
				reason: reason,
			})
		}
	}

	// Sort by score
	sort.Slice(scoredAgents, func(i, j int) bool {
		return scoredAgents[i].score > scoredAgents[j].score
	})

	// Build suggestions
	for _, sa := range scoredAgents {
		results = append(results, CompletionResult{
			Text:        "@" + sa.agent.ID,
			Description: fmt.Sprintf("%s - %s (Suggested: %s)", sa.agent.Name, strings.Join(sa.agent.Capabilities, ", "), sa.reason),
			Type:        "agent_suggestion",
		})
	}

	return results
}

// Helper to check if agent has capability
func agentHasCapability(agent config.AgentConfig, capability string) bool {
	capLower := strings.ToLower(capability)
	for _, cap := range agent.Capabilities {
		if strings.ToLower(cap) == capLower {
			return true
		}
	}
	return false
}

// CommandHistory manages command history with search and navigation
type CommandHistory struct {
	historyFile string
	commands    []string
	currentPos  int
	maxSize     int
}

// NewCommandHistory creates a new command history manager
func NewCommandHistory(historyFile string) *CommandHistory {
	ch := &CommandHistory{
		historyFile: historyFile,
		commands:    make([]string, 0),
		currentPos:  -1,
		maxSize:     1000,
	}

	ch.loadHistory()
	return ch
}

// Add adds a command to history
func (ch *CommandHistory) Add(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	// Remove duplicates
	ch.removeDuplicate(command)

	// Add to end
	ch.commands = append(ch.commands, command)

	// Maintain size limit
	if len(ch.commands) > ch.maxSize {
		ch.commands = ch.commands[1:]
	}

	// Reset position
	ch.currentPos = len(ch.commands)

	// Save to disk
	ch.saveHistory()
}

// Previous returns the previous command in history
func (ch *CommandHistory) Previous() string {
	if len(ch.commands) == 0 {
		return ""
	}

	if ch.currentPos > 0 {
		ch.currentPos--
	}

	if ch.currentPos < len(ch.commands) {
		return ch.commands[ch.currentPos]
	}

	return ""
}

// Next returns the next command in history
func (ch *CommandHistory) Next() string {
	if len(ch.commands) == 0 {
		return ""
	}

	if ch.currentPos < len(ch.commands)-1 {
		ch.currentPos++
		return ch.commands[ch.currentPos]
	}

	// At end of history
	ch.currentPos = len(ch.commands)
	return ""
}

// Search performs fuzzy search on command history
func (ch *CommandHistory) Search(term string) []string {
	var matches []string
	term = strings.ToLower(term)

	// Search from most recent to oldest
	for i := len(ch.commands) - 1; i >= 0; i-- {
		cmd := ch.commands[i]
		if fuzzyMatch(strings.ToLower(cmd), term) {
			matches = append(matches, cmd)

			// Limit results to prevent overwhelming UI
			if len(matches) >= 10 {
				break
			}
		}
	}

	return matches
}

// GetRecent returns the most recent N commands
func (ch *CommandHistory) GetRecent(count int) []string {
	if len(ch.commands) == 0 {
		return []string{}
	}

	start := len(ch.commands) - count
	if start < 0 {
		start = 0
	}

	// Return copy to avoid modification
	recent := make([]string, len(ch.commands[start:]))
	copy(recent, ch.commands[start:])

	// Reverse to show most recent first
	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}

	return recent
}

// Command represents a Guild command with completion information
type Command struct {
	Name         string
	Description  string
	Usage        string
	Handler      func(args []string) string
	Completion   func(args []string) []CompletionResult
}

// CommandProcessor handles command processing and completion integration
type CommandProcessor struct {
	completionEngine *CompletionEngine
	commandHistory   *CommandHistory
	guildConfig      *config.GuildConfig
	commands         map[string]Command
}

// NewCommandProcessor creates a new command processor
func NewCommandProcessor(completionEngine *CompletionEngine, commandHistory *CommandHistory, guildConfig *config.GuildConfig) *CommandProcessor {
	return &CommandProcessor{
		completionEngine: completionEngine,
		commandHistory:   commandHistory,
		guildConfig:      guildConfig,
		commands:         make(map[string]Command),
	}
}

// Helper functions

// fuzzyMatch performs intelligent fuzzy matching
func fuzzyMatch(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	text = strings.ToLower(text)
	pattern = strings.ToLower(pattern)

	// First check if it's a simple substring
	if strings.Contains(text, pattern) {
		return true
	}

	// Check if all characters in pattern appear in order in text
	textIdx := 0
	for _, patternChar := range pattern {
		found := false
		for textIdx < len(text) {
			if rune(text[textIdx]) == patternChar {
				found = true
				textIdx++
				break
			}
			textIdx++
		}
		if !found {
			return false
		}
	}

	return true
}

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
func (ch *CommandHistory) loadHistory() {
	// Create directory if it doesn't exist
	dir := filepath.Dir(ch.historyFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Log error but continue with empty history
		return
	}

	// Read history file
	data, err := os.ReadFile(ch.historyFile)
	if err != nil {
		// File doesn't exist yet, that's OK
		if !os.IsNotExist(err) {
			// Log error but continue
		}
		return
	}

	// Parse history (one command per line)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			ch.commands = append(ch.commands, line)
		}
	}

	// Limit history size
	if len(ch.commands) > ch.maxSize {
		ch.commands = ch.commands[len(ch.commands)-ch.maxSize:]
	}

	// Set position to end
	ch.currentPos = len(ch.commands)
}

// saveHistory saves command history to file
func (ch *CommandHistory) saveHistory() {
	// Build history content
	var content strings.Builder
	for _, cmd := range ch.commands {
		content.WriteString(cmd)
		content.WriteString("\n")
	}

	// Create directory if needed
	dir := filepath.Dir(ch.historyFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Log error but continue
		return
	}

	// Write to file
	if err := os.WriteFile(ch.historyFile, []byte(content.String()), 0644); err != nil {
		// Log error but continue
		return
	}
}

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
func (ch *CommandHistory) removeDuplicate(command string) {
	for i, cmd := range ch.commands {
		if cmd == command {
			ch.commands = append(ch.commands[:i], ch.commands[i+1:]...)
			if ch.currentPos > i {
				ch.currentPos--
			}
			break
		}
	}
}

// Custom tea.Msg types for agent streaming
type agentStreamMsg struct {
	agentID   string
	fragment  string
	complete  bool
	timestamp time.Time
}

type agentStatusMsg struct {
	agentID string
	status  pb.AgentStatus_State
	task    string
}

type agentErrorMsg struct {
	agentID string
	error   error
}

// Tool execution messages
type toolExecutionStartMsg struct {
	executionID string
	toolID      string
	toolName    string
	agentID     string
	parameters  map[string]string
	cost        float64
}

type toolExecutionProgressMsg struct {
	executionID string
	progress    float64
	message     string
}

type toolExecutionCompleteMsg struct {
	executionID string
	result      string
	cost        float64
}

type toolExecutionErrorMsg struct {
	executionID string
	error       error
}

type toolAuthRequiredMsg struct {
	executionID   string
	permissions   []string
	estimatedCost float64
}

// Agent status update messages (Agent 3)
type AgentStatusUpdateMsg struct {
	AgentID string
	Status  *AgentStatus
	State   AgentState
}

// chatKeyMap defines keyboard shortcuts for Guild Chat
type chatKeyMap struct {
	Send       key.Binding
	Quit       key.Binding
	Help       key.Binding
	PromptView key.Binding
	AgentList  key.Binding
	StatusView key.Binding
	GlobalView key.Binding // Switch to global view
	AgentFocus key.Binding // Focus on specific agent

	// Completion and history
	Tab        key.Binding // Tab completion
	Up         key.Binding // History up / completion up
	Down       key.Binding // History down / completion down
	CtrlR      key.Binding // Search history
	Escape     key.Binding // Cancel completion/search
}

// newChatKeyMap creates the default key mappings
func newChatKeyMap() chatKeyMap {
	return chatKeyMap{
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "ctrl+d"),
			key.WithHelp("ctrl+c", "exit chat"),
		),
		Help: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "toggle help"),
		),
		PromptView: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p", "view prompts"),
		),
		AgentList: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "list agents"),
		),
		StatusView: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "show status"),
		),
		GlobalView: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("ctrl+g", "global view"),
		),
		AgentFocus: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("ctrl+f", "focus agent"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "auto-complete"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "history/completion up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "history/completion down"),
		),
		CtrlR: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "search history"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k chatKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k chatKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Send, k.Tab, k.Up, k.Down},
		{k.PromptView, k.AgentList, k.StatusView},
		{k.GlobalView, k.AgentFocus, k.CtrlR},
		{k.Help, k.Escape, k.Quit},
	}
}

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

	// Create styles with medieval theming
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")) // Purple

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")). // Bright blue
		Bold(true).
		Padding(0, 1)

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("36")) // Cyan

	agentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("34")). // Green
		Bold(true)

	systemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")). // Yellow
		Italic(true)

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")). // Red
		Bold(true)

	toolStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("208")). // Orange
		Bold(true)

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())
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
	commandProcessor := NewCommandProcessor(completionEngine, commandHistory, guildConfig)

	// Initialize agent status systems (Agent 3)
	statusTracker := NewAgentStatusTracker(guildConfig)
	statusDisplay := NewStatusDisplay(statusTracker, chatWidth/4, chatWidth/3)
	agentIndicators := NewAgentIndicators()

	// Start the status tracking systems
	statusTracker.StartTracking()
	agentIndicators.SetupDefaultAnimations()

	return ChatModel{
		guildConfig:    guildConfig,
		campaignID:     campaignID,
		sessionID:      sessionID,
		input:          ta,
		messages:       vp,
		help:           help,
		keymap:         newChatKeyMap(),
		messageLog:     []chatMessage{},
		currentAgent:   "", // Start in global view
		agentStreams:   make(map[string][]chatMessage),
		globalStream:   []chatMessage{},
		chatMode:       globalView,
		promptLayers:   make(map[string]string),
		toolExecutions: make(map[string]*toolExecution),
		activeTools:    []string{},
		guildClient:    guildClient,
		promptClient:   promptClient,
		grpcConn:       grpcConn,
		ctx:            ctx,
		cancel:         cancel,
		registry:       registry,
		borderStyle:      borderStyle,
		headerStyle:      headerStyle,
		inputStyle:       inputStyle,
		agentStyle:       agentStyle,
		systemStyle:      systemStyle,
		errorStyle:       errorStyle,
		toolStyle:        toolStyle,
		markdownRenderer: markdownRenderer,
		contentFormatter: contentFormatter,
		completionEngine: completionEngine,
		commandHistory:   commandHistory,
		commandProcessor: commandProcessor,
		showingCompletion: false,
		completionResults: []CompletionResult{},
		completionIndex:   0,

		// Agent status system (Agent 3)
		statusTracker:   statusTracker,
		statusDisplay:   statusDisplay,
		agentIndicators: agentIndicators,
		showAgentStatus: true, // Default to showing status

		// Integration state (Agent 4)
		demoMode:         false,
		integrationFlags: make(map[string]bool),
	}
}

// Init implements tea.Model
func (m ChatModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.listenForAgentUpdates(),
	)
}

// listenForAgentUpdates listens for streaming agent responses
func (m ChatModel) listenForAgentUpdates() tea.Cmd {
	return func() tea.Msg {
		// This would start a gRPC stream in a real implementation
		// For now, return nil to avoid blocking
		return nil
	}
}

// Update implements tea.Model
func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.input, tiCmd = m.input.Update(msg)
	m.messages, vpCmd = m.messages.Update(msg)

	switch msg := msg.(type) {
	case agentStreamMsg:
		// Handle streaming agent response
		m.handleAgentStream(msg)
		m.updateMessagesView()
		return m, nil

	case agentStatusMsg:
		// Handle agent status update
		m.handleAgentStatus(msg)
		m.updateMessagesView()
		return m, nil

	case agentErrorMsg:
		// Handle agent error
		m.handleAgentError(msg)
		m.updateMessagesView()
		return m, nil

	case AgentStatusUpdateMsg:
		// Handle status system update (Agent 3)
		m.handleAgentStatusUpdate(msg)
		return m, nil

	case toolExecutionStartMsg:
		// Handle tool execution start
		m.handleToolExecutionStart(msg)
		m.updateMessagesView()
		return m, nil

	case toolExecutionProgressMsg:
		// Handle tool execution progress
		m.handleToolExecutionProgress(msg)
		m.updateMessagesView()
		return m, nil

	case toolExecutionCompleteMsg:
		// Handle tool execution completion
		m.handleToolExecutionComplete(msg)
		m.updateMessagesView()
		return m, nil

	case toolExecutionErrorMsg:
		// Handle tool execution error
		m.handleToolExecutionError(msg)
		m.updateMessagesView()
		return m, nil

	case toolAuthRequiredMsg:
		// Handle tool authorization request
		m.handleToolAuthRequired(msg)
		m.updateMessagesView()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			// First time setup
			m.ready = true
		}

		// Update component sizes
		headerHeight := 3
		helpHeight := 3
		inputHeight := 5

		// Calculate dimensions for status panel (Agent 3)
		statusPanelWidth := msg.Width / 4 // 25% of screen width
		mainChatWidth := msg.Width - statusPanelWidth - 8 // Remaining space minus padding

		m.messages.Width = mainChatWidth
		m.messages.Height = msg.Height - headerHeight - inputHeight - helpHeight

		m.input.SetWidth(mainChatWidth)

		// Update status display dimensions (Agent 3)
		if m.statusDisplay != nil {
			m.statusDisplay.SetDimensions(statusPanelWidth, msg.Height - headerHeight)
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.Send):
			return m.handleSendMessage()

		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keymap.Help):
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, m.keymap.PromptView):
			return m.handlePromptView()

		case key.Matches(msg, m.keymap.AgentList):
			return m.handleAgentList()

		case key.Matches(msg, m.keymap.StatusView):
			return m.handleStatusView()

		case key.Matches(msg, m.keymap.GlobalView):
			return m.handleGlobalView()

		case key.Matches(msg, m.keymap.AgentFocus):
			return m.handleAgentFocus()

		case key.Matches(msg, m.keymap.Tab):
			return m.handleTabCompletion()

		case key.Matches(msg, m.keymap.Up):
			return m.handleUpKey()

		case key.Matches(msg, m.keymap.Down):
			return m.handleDownKey()

		case key.Matches(msg, m.keymap.CtrlR):
			return m.handleSearchHistory()

		case key.Matches(msg, m.keymap.Escape):
			return m.handleEscape()
		}
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

// handleAgentStream handles streaming agent response fragments
func (m *ChatModel) handleAgentStream(msg agentStreamMsg) {
	// Find or create agent message
	found := false
	for i := len(m.messageLog) - 1; i >= 0; i-- {
		if m.messageLog[i].Type == msgAgent && m.messageLog[i].Sender == msg.agentID && !msg.complete {
			// Append to existing message
			m.messageLog[i].Content += msg.fragment
			found = true
			break
		}
	}

	if !found {
		// Create new agent message
		agentMsg := chatMessage{
			Timestamp: msg.timestamp,
			Sender:    msg.agentID,
			AgentID:   msg.agentID, // Associate with specific agent
			Content:   msg.fragment,
			Type:      msgAgent,
		}
		m.addMessage(agentMsg)
	}
}

// handleAgentStatus handles agent status updates
func (m *ChatModel) handleAgentStatus(msg agentStatusMsg) {
	var statusText string
	switch msg.status {
	case pb.AgentStatus_THINKING:
		statusText = fmt.Sprintf("%s is thinking...", msg.agentID)
	case pb.AgentStatus_WORKING:
		statusText = fmt.Sprintf("%s is working on: %s", msg.agentID, msg.task)
	case pb.AgentStatus_IDLE:
		statusText = fmt.Sprintf("%s is ready", msg.agentID)
	default:
		statusText = fmt.Sprintf("%s status: %s", msg.agentID, msg.status.String())
	}

	// Add status message
	statusMsg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		AgentID:   msg.agentID, // Associate with specific agent
		Content:   statusText,
		Type:      msgSystem,
	}
	m.addMessage(statusMsg)
}

// handleAgentError handles agent error messages
func (m *ChatModel) handleAgentError(msg agentErrorMsg) {
	errorMsg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		AgentID:   msg.agentID, // Associate with specific agent
		Content:   fmt.Sprintf("Error from %s: %v", msg.agentID, msg.error),
		Type:      msgError,
	}
	m.addMessage(errorMsg)
}

// handleAgentStatusUpdate handles status system updates (Agent 3)
func (m *ChatModel) handleAgentStatusUpdate(msg AgentStatusUpdateMsg) {
	// Update the status tracker
	if m.statusTracker != nil && msg.Status != nil {
		m.statusTracker.UpdateAgentStatus(msg.AgentID, msg.Status)
	}

	// Update agent indicators based on state
	if m.agentIndicators != nil {
		m.agentIndicators.UpdateAnimation(msg.AgentID, msg.State)
	}

	// Update display to reflect changes
	if m.statusDisplay != nil {
		m.statusDisplay.Update()
		// If the changed agent is currently selected, keep it selected
		if msg.AgentID == m.statusDisplay.GetSelectedAgent() {
			m.statusDisplay.SelectAgent(msg.AgentID)
		}
	}
}

// View implements tea.Model
func (m ChatModel) View() string {
	if !m.ready {
		return "Preparing Guild Chat Chamber..."
	}

	// Header with view mode
	viewInfo := "Global View"
	if m.chatMode == agentView && m.currentAgent != "" {
		viewInfo = fmt.Sprintf("Agent: %s", m.currentAgent)
	}

	header := m.headerStyle.Render(fmt.Sprintf(
		"🏰 Guild Chat | %s | Campaign: %s | Session: %s | Agents: %d",
		viewInfo,
		m.getCampaignDisplay(),
		m.sessionID[:8],
		len(m.guildConfig.Agents),
	))

	// Messages area
	messagesView := m.borderStyle.Render(m.messages.View())

	// Input area
	inputView := m.inputStyle.Render(m.input.View())

	// Help area
	helpView := m.help.View(m.keymap)

	// Agent status panel (Agent 3)
	var statusPanel string
	if m.showAgentStatus && m.statusDisplay != nil {
		statusPanel = m.statusDisplay.RenderStatusPanel()
	}

	// Create main content area (messages + input + help)
	mainChatArea := lipgloss.JoinVertical(
		lipgloss.Left,
		messagesView,
		inputView,
		helpView,
	)

	// Combine main chat with status panel horizontally if status panel exists
	var contentArea string
	if statusPanel != "" {
		contentArea = lipgloss.JoinHorizontal(
			lipgloss.Top,
			mainChatArea,
			statusPanel,
		)
	} else {
		contentArea = mainChatArea
	}

	// Combine header with content area
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		contentArea,
	)
}

// handleSendMessage processes user input
func (m ChatModel) handleSendMessage() (ChatModel, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return m, nil
	}

	// Add to command history
	if m.commandHistory != nil {
		m.commandHistory.Add(input)
	}

	// Clear input
	m.input.Reset()

	// Add user message to log
	userMsg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "user",
		Content:   input,
		Type:      msgUser,
	}
	m.addMessage(userMsg)

	// Process the message
	response := m.processMessage(input)

	// Add response to log
	responseMsg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		Content:   response,
		Type:      msgSystem,
	}
	m.addMessage(responseMsg)

	// Update viewport with new messages
	m.updateMessagesView()

	return m, nil
}

// processMessage handles different types of user input
func (m ChatModel) processMessage(input string) string {
	// Handle commands
	if strings.HasPrefix(input, "/") {
		return m.handleCommand(input)
	}

	// Handle agent mentions
	if strings.HasPrefix(input, "@") {
		return m.handleAgentMention(input)
	}

	// Default response for general messages
	return "I don't understand that command. Type /help for available commands."
}

// handleCommand processes slash commands
func (m ChatModel) handleCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "Invalid command format."
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/help":
		return m.getHelpText()

	case "/status":
		return m.getStatusText()

	case "/agents":
		return m.getAgentsText()

	case "/prompt":
		return m.handlePromptCommand(args)

	case "/tools":
		return m.handleToolCommand(args)

	case "/tool":
		return m.handleToolExecuteCommand(args)

	case "/test":
		return m.handleTestCommand(args)

	case "/exit", "/quit":
		return "Use Ctrl+C to exit Guild Chat."

	default:
		return fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd)
	}
}

// handleAgentMention processes agent mentions
func (m ChatModel) handleAgentMention(input string) string {
	parts := strings.SplitN(input, " ", 2)
	if len(parts) < 2 {
		return "Please include a message after the agent mention."
	}

	agentID := strings.TrimPrefix(parts[0], "@")
	message := parts[1]

	if agentID == "all" {
		// TODO: Implement broadcast to all agents
		return fmt.Sprintf("Broadcasting to all agents: %s\n(Note: Broadcast not yet implemented)", message)
	}

	// Check if agent exists locally first
	agentFound := false
	for _, agent := range m.guildConfig.Agents {
		if agent.ID == agentID {
			agentFound = true
			break
		}
	}

	if !agentFound {
		return fmt.Sprintf("Agent '%s' not found. Use /agents to see available agents.", agentID)
	}

	// Call gRPC service to send message to agent
	_ = &pb.AgentMessageRequest{
		AgentId:    agentID,
		Message:    message,
		SessionId:  m.sessionID,
		CampaignId: m.campaignID,
		Context: map[string]string{
			"chat_mode": "interactive",
			"user":      "chat_user",
		},
	}

	// Switch to agent view mode
	m.currentAgent = agentID
	m.chatMode = agentView

	// Start streaming conversation with agent
	go m.streamAgentConversation(agentID, message)

	// Return immediate acknowledgment
	agentName := agentID
	for _, agent := range m.guildConfig.Agents {
		if agent.ID == agentID {
			agentName = agent.Name
			break
		}
	}

	return fmt.Sprintf("Switching to %s conversation...", m.agentStyle.Render(agentName))
}

// streamAgentConversation handles bidirectional streaming with an agent
func (m *ChatModel) streamAgentConversation(agentID, message string) {
	// Create streaming client
	ctx, cancel := context.WithTimeout(m.ctx, 2*time.Minute)
	defer cancel()

	stream, err := m.guildClient.StreamAgentConversation(ctx)
	if err != nil {
		// Send error message through channel
		go func() {
			// In a real implementation, we'd use a tea.Cmd to send this
			_ = agentErrorMsg{
				agentID: agentID,
				error:   gerror.Wrap(err, gerror.ErrCodeConnection, "failed to create stream").
				WithComponent("cli").
				WithOperation("chat.streamAgentConversation"),
			}
		}()
		return
	}
	defer stream.CloseSend()

	// Send initial message
	initialReq := &pb.AgentStreamRequest{
		Request: &pb.AgentStreamRequest_Message{
			Message: &pb.AgentMessageRequest{
				AgentId:    agentID,
				Message:    message,
				SessionId:  m.sessionID,
				CampaignId: m.campaignID,
				Context: map[string]string{
					"chat_mode": "interactive",
					"user":      "chat_user",
				},
			},
		},
	}

	if err := stream.Send(initialReq); err != nil {
		// Send error through channel
		return
	}

	// Read responses
	for {
		resp, err := stream.Recv()
		if err != nil {
			// Stream ended or error
			return
		}

		switch response := resp.Response.(type) {
		case *pb.AgentStreamResponse_Fragment:
			// Handle response fragment
			// In real implementation, send through tea.Cmd
			_ = agentStreamMsg{
				agentID:   response.Fragment.AgentId,
				fragment:  response.Fragment.Content,
				complete:  response.Fragment.IsComplete,
				timestamp: time.Unix(response.Fragment.Timestamp, 0),
			}

		case *pb.AgentStreamResponse_Status:
			// Handle status update
			_ = agentStatusMsg{
				agentID: agentID,
				status:  response.Status.State,
				task:    response.Status.CurrentTask,
			}

		case *pb.AgentStreamResponse_Event:
			// Handle events (thinking, working, etc)
			// Could map to status messages
		}
	}
}

// handlePromptCommand processes prompt-related commands
func (m ChatModel) handlePromptCommand(args []string) string {
	if len(args) == 0 {
		return "Usage: /prompt [get|set|list|delete] [options...]"
	}

	switch args[0] {
	case "list":
		return m.getPromptLayersText()

	case "get":
		if len(args) < 3 {
			return "Usage: /prompt get --layer <layer-name>"
		}
		// TODO: Implement actual prompt retrieval
		return fmt.Sprintf("Getting prompt layer: %s\n(Note: Layered prompt system integration pending)", args[2])

	case "set":
		if len(args) < 4 {
			return "Usage: /prompt set --layer <layer-name> --text <content>"
		}
		// TODO: Implement actual prompt setting
		return fmt.Sprintf("Setting prompt layer: %s\n(Note: Layered prompt system integration pending)", args[2])

	case "delete":
		if len(args) < 3 {
			return "Usage: /prompt delete --layer <layer-name>"
		}
		// TODO: Implement actual prompt deletion
		return fmt.Sprintf("Deleting prompt layer: %s\n(Note: Layered prompt system integration pending)", args[2])

	default:
		return fmt.Sprintf("Unknown prompt command: %s", args[0])
	}
}

// updateMessagesView refreshes the messages viewport
func (m ChatModel) updateMessagesView() {
	var content strings.Builder

	// Get messages for current view mode
	messages := m.getCurrentMessages()

	for _, msg := range messages {
		timestamp := msg.Timestamp.Format("15:04:05")

		switch msg.Type {
		case msgUser:
			// Format user message with potential markdown
			formattedContent := msg.Content
			if m.contentFormatter != nil {
				formattedContent = m.contentFormatter.FormatUserMessage(msg.Content)
			}
			content.WriteString(fmt.Sprintf("[%s] You: %s\n", timestamp, formattedContent))

		case msgAgent:
			// Format agent message with rich content rendering
			formattedContent := msg.Content
			if m.contentFormatter != nil {
				formattedContent = m.contentFormatter.FormatAgentResponse(msg.Content, msg.AgentID)
			}

			// Show agent attribution differently based on view mode
			if m.chatMode == globalView {
				styled := m.agentStyle.Render(fmt.Sprintf("[%s]", msg.Sender))
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, formattedContent))
			} else {
				// In agent view, just show the agent name without brackets
				styled := m.agentStyle.Render(msg.Sender)
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, formattedContent))
			}

		case msgSystem:
			// Format system message with rich content rendering
			formattedContent := msg.Content
			if m.contentFormatter != nil {
				formattedContent = m.contentFormatter.FormatSystemMessage(msg.Content)
			}

			// Show agent attribution for system messages in global view
			if m.chatMode == globalView && msg.AgentID != "" {
				styled := m.systemStyle.Render(fmt.Sprintf("[Guild:%s]", msg.AgentID))
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, formattedContent))
			} else {
				styled := m.systemStyle.Render("[Guild]")
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, formattedContent))
			}

		case msgPrompt:
			content.WriteString(fmt.Sprintf("[%s] 🧠 Prompt: %s\n", timestamp, msg.Content))

		case msgError:
			// Format error message with rich content and emphasis
			formattedContent := msg.Content
			if m.contentFormatter != nil {
				formattedContent = m.contentFormatter.FormatErrorMessage(msg.Content)
			} else {
				styled := m.errorStyle.Render("[Error]")
				formattedContent = fmt.Sprintf("%s: %s", styled, msg.Content)
			}
			content.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, formattedContent))

		case msgAgentThinking:
			// Format thinking message with rich content
			formattedContent := msg.Content
			if m.contentFormatter != nil {
				formattedContent = m.contentFormatter.FormatThinkingMessage(msg.Content, msg.AgentID)
			} else {
				formattedContent = fmt.Sprintf("🤔 %s", msg.Content)
			}
			content.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, formattedContent))

		case msgAgentWorking:
			// Format working message with rich content
			formattedContent := msg.Content
			if m.contentFormatter != nil {
				formattedContent = m.contentFormatter.FormatWorkingMessage(msg.Content, msg.AgentID)
			} else {
				formattedContent = fmt.Sprintf("⚙️ %s", msg.Content)
			}
			content.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, formattedContent))

		case msgToolStart:
			// Format tool message with rich content
			formattedContent := msg.Content
			if m.contentFormatter != nil {
				toolName := "Tool"
				if msg.AgentID != "" {
					toolName = msg.AgentID
				}
				formattedContent = m.contentFormatter.FormatToolOutput(msg.Content, toolName)
			}

			// Show agent attribution for tool messages in global view
			if m.chatMode == globalView && msg.AgentID != "" {
				styled := m.toolStyle.Render(fmt.Sprintf("[Tool:%s]", msg.AgentID))
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, formattedContent))
			} else {
				styled := m.toolStyle.Render("[Tool]")
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, formattedContent))
			}

		case msgToolProgress:
			if m.chatMode == globalView && msg.AgentID != "" {
				styled := m.toolStyle.Render(fmt.Sprintf("[Tool:%s]", msg.AgentID))
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, msg.Content))
			} else {
				styled := m.toolStyle.Render("[Tool]")
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, msg.Content))
			}

		case msgToolComplete:
			if m.chatMode == globalView && msg.AgentID != "" {
				styled := m.toolStyle.Render(fmt.Sprintf("[Tool:%s]", msg.AgentID))
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, msg.Content))
			} else {
				styled := m.toolStyle.Render("[Tool]")
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, msg.Content))
			}

		case msgToolError:
			if m.chatMode == globalView && msg.AgentID != "" {
				styled := m.errorStyle.Render(fmt.Sprintf("[Tool:%s Error]", msg.AgentID))
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, msg.Content))
			} else {
				styled := m.errorStyle.Render("[Tool Error]")
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, msg.Content))
			}

		case msgToolAuth:
			if m.chatMode == globalView && msg.AgentID != "" {
				styled := m.toolStyle.Render(fmt.Sprintf("[Tool:%s Auth]", msg.AgentID))
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, msg.Content))
			} else {
				styled := m.toolStyle.Render("[Tool Auth]")
				content.WriteString(fmt.Sprintf("[%s] %s: %s\n", timestamp, styled, msg.Content))
			}
		}
	}

	m.messages.SetContent(content.String())
	m.messages.GotoBottom()
}

// Helper methods for command responses

func (m ChatModel) getCampaignDisplay() string {
	if m.campaignID == "" {
		return "none"
	}
	return m.campaignID
}

func (m ChatModel) getHelpText() string {
	return `🏰 Guild Chat Commands:

Agent Communication:
  @agent-name <message>  - Send message to specific agent
  @all <message>         - Broadcast to all agents

Campaign Management:
  /status               - Show campaign and agent status
  /agents               - List available agents

Tool Management:
  /tools list           - List all available tools
  /tools info <tool-id> - Show detailed tool information
  /tools search <capability>  - Find tools by capability
  /tools status         - Show active tool executions
  /tool <tool-id> [params]  - Execute a tool directly

Layered Prompt System:
  /prompt list          - Show active prompt layers
  /prompt get --layer <name>  - View specific prompt layer
  /prompt set --layer <name> --text <content>  - Update prompt layer
  /prompt delete --layer <name>  - Remove prompt layer

Rich Content Testing:
  /test markdown        - Demonstrate markdown features
  /test code <language> - Show syntax highlighting (go, python, js, etc.)
  /test mixed           - Show combined markdown and code

General:
  /help                 - Show this help message
  /exit                 - Exit Guild Chat (or use Ctrl+C)

View Management:
  Ctrl+G               - Switch to global view (all agents)
  Ctrl+F               - Focus on specific agent
  @agent-name <msg>    - Switch to agent conversation

Keyboard Shortcuts:
  Ctrl+P               - Quick prompt view
  Ctrl+A               - Quick agent list
  Ctrl+S               - Quick status view
  Ctrl+H               - Toggle help
  Ctrl+C               - Exit`
}

func (m ChatModel) getStatusText() string {
	return fmt.Sprintf(`🏰 Guild Status:

Campaign: %s
Session:  %s
Guild:    %s

Agents:   %d configured
          (Note: Real-time status not yet implemented)

Prompt Layers:
  📋 Platform: Active
  🏰 Guild:    Active
  👷 Role:     Active (per agent)
  👤 Session:  %s
  💬 Turn:     None

Memory: Ready
Storage: Connected`,
		m.getCampaignDisplay(),
		m.sessionID[:8],
		m.guildConfig.Name,
		len(m.guildConfig.Agents),
		m.sessionID[:8],
	)
}

func (m ChatModel) getAgentsText() string {
	var content strings.Builder
	content.WriteString("🏰 Available Guild Agents:\n\n")

	// Get agent status from gRPC server
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	req := &pb.ListAgentsRequest{
		CampaignId:    m.campaignID,
		IncludeStatus: true,
	}

	resp, err := m.guildClient.ListAvailableAgents(ctx, req)
	if err != nil {
		// Fallback to local config if gRPC fails
		for i, agent := range m.guildConfig.Agents {
			status := "⚪ Unknown" // Unknown status
			content.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, agent.ID, agent.Name))
			content.WriteString(fmt.Sprintf("   Type: %s | Provider: %s\n", agent.Type, agent.Provider))
			content.WriteString(fmt.Sprintf("   Status: %s (gRPC unavailable)\n", status))
			content.WriteString(fmt.Sprintf("   Capabilities: %s\n", strings.Join(agent.Capabilities, ", ")))
			content.WriteString("\n")
		}
	} else {
		// Use gRPC response for real-time status
		for i, agent := range resp.Agents {
			statusIcon := getStatusIcon(agent.Status)
			statusText := getStatusText(agent.Status)
			content.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, agent.Id, agent.Name))
			content.WriteString(fmt.Sprintf("   Type: %s | Provider: %s\n", agent.Type, agent.Metadata["provider"]))
			content.WriteString(fmt.Sprintf("   Status: %s %s\n", statusIcon, statusText))
			content.WriteString(fmt.Sprintf("   Capabilities: %s\n", strings.Join(agent.Capabilities, ", ")))
			if agent.Status != nil && agent.Status.CurrentTask != "" {
				content.WriteString(fmt.Sprintf("   Current Task: %s\n", agent.Status.CurrentTask))
			}
			content.WriteString("\n")
		}
	}

	content.WriteString("Use @agent-id to send messages to specific agents.")
	return content.String()
}

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
func getStatusText(status *pb.AgentStatus) string {
	if status == nil {
		return "Unknown"
	}
	switch status.State {
	case pb.AgentStatus_IDLE:
		return "Idle"
	case pb.AgentStatus_THINKING:
		return "Thinking"
	case pb.AgentStatus_WORKING:
		return "Working"
	case pb.AgentStatus_WAITING:
		return "Waiting"
	case pb.AgentStatus_ERROR:
		return "Error"
	case pb.AgentStatus_OFFLINE:
		return "Offline"
	default:
		return "Unknown"
	}
}

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

func (m ChatModel) handlePromptView() (ChatModel, tea.Cmd) {
	// Add prompt view message
	msg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		Content:   m.getPromptLayersText(),
		Type:      msgPrompt,
	}
	m.addMessage(msg)
	m.updateMessagesView()

	return m, nil
}

func (m ChatModel) handleAgentList() (ChatModel, tea.Cmd) {
	// Add agent list message
	msg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		Content:   m.getAgentsText(),
		Type:      msgSystem,
	}
	m.addMessage(msg)
	m.updateMessagesView()

	return m, nil
}

func (m ChatModel) handleStatusView() (ChatModel, tea.Cmd) {
	// Toggle agent status panel visibility (Agent 3)
	m.showAgentStatus = !m.showAgentStatus

	// Add status message
	statusContent := m.getStatusText()
	if m.showAgentStatus {
		statusContent += "\n\n🏰 Agent status panel is now visible (Ctrl+S to toggle)"
	} else {
		statusContent += "\n\n📊 Agent status panel is now hidden (Ctrl+S to toggle)"
	}

	msg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		Content:   statusContent,
		Type:      msgSystem,
	}
	m.addMessage(msg)
	m.updateMessagesView()

	return m, nil
}

// addMessage adds a message to appropriate streams (global + agent-specific)
func (m *ChatModel) addMessage(msg chatMessage) {
	// Always add to legacy messageLog for compatibility
	m.messageLog = append(m.messageLog, msg)

	// Add to global stream
	m.globalStream = append(m.globalStream, msg)

	// Add to agent-specific stream if applicable
	// Determine the agent ID for this message
	var agentID string
	if msg.AgentID != "" {
		agentID = msg.AgentID
	} else if msg.Type == msgAgent && msg.Sender != "system" && msg.Sender != "user" {
		// For agent messages without explicit AgentID, use the Sender
		agentID = msg.Sender
	}

	// Add to agent stream if we have a valid agent ID
	if agentID != "" && agentID != "system" && agentID != "user" {
		if _, exists := m.agentStreams[agentID]; !exists {
			m.agentStreams[agentID] = []chatMessage{}
		}
		m.agentStreams[agentID] = append(m.agentStreams[agentID], msg)
	}
}

// getCurrentMessages returns the appropriate message stream for current view
func (m *ChatModel) getCurrentMessages() []chatMessage {
	if m.chatMode == agentView && m.currentAgent != "" {
		// Return agent-specific messages
		if agentMessages, exists := m.agentStreams[m.currentAgent]; exists {
			return agentMessages
		}
		return []chatMessage{} // Empty if no messages for this agent
	}

	// Return global messages
	return m.globalStream
}

// handleGlobalView switches to global view mode
func (m ChatModel) handleGlobalView() (ChatModel, tea.Cmd) {
	m.chatMode = globalView
	m.currentAgent = ""
	m.updateMessagesView()

	// Add system message about view switch
	msg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		Content:   "🌍 Switched to Global View - seeing all agents and system messages",
		Type:      msgSystem,
	}
	m.addMessage(msg)
	m.updateMessagesView()

	return m, nil
}

// handleAgentFocus prompts user to select an agent to focus on
func (m ChatModel) handleAgentFocus() (ChatModel, tea.Cmd) {
	// For now, show instructions - in a full implementation this could open a selection UI
	msg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
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

	for i, agent := range m.guildConfig.Agents {
		messageCount := 0
		if agentMessages, exists := m.agentStreams[agent.ID]; exists {
			messageCount = len(agentMessages)
		}

		content.WriteString(fmt.Sprintf("  %d. @%s (%s) - %d messages\n",
			i+1, agent.ID, agent.Name, messageCount))
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
func (m ChatModel) getToolListText() string {
	// Try to get tools from tool registry first
	toolRegistry := m.registry.Tools()
	if toolRegistry != nil {
		toolNames := toolRegistry.ListTools()
		if len(toolNames) > 0 {
			var content strings.Builder
			content.WriteString("🔨 Available Tools:\n\n")

			for i, toolName := range toolNames {
				content.WriteString(fmt.Sprintf("%d. %s\n", i+1, m.toolStyle.Render(toolName)))

				// Try to get tool info
				if tool, err := toolRegistry.GetTool(toolName); err == nil {
					if toolWithSchema, ok := tool.(interface{ Schema() map[string]interface{} }); ok {
						schema := toolWithSchema.Schema()
						if desc, exists := schema["description"]; exists {
							content.WriteString(fmt.Sprintf("   Description: %s\n", desc))
						}
					}
				}
				content.WriteString("\n")
			}

			content.WriteString("Use '/tools info <tool-id>' for detailed information.")
			return content.String()
		}
	}

	// Fallback to mock data
	return `🔨 Available Tools:

1. file-reader
   Description: Read file contents
   Capabilities: file-system, text-processing
   Cost: Free

2. shell-exec
   Description: Execute shell commands
   Capabilities: system, execution
   Cost: Low

3. web-scraper
   Description: Scrape web content
   Capabilities: web, data-extraction
   Cost: Medium

4. corpus-search
   Description: Search project corpus
   Capabilities: search, knowledge
   Cost: Low

Use '/tools info <tool-id>' for detailed information.`
}

// getToolInfoText returns detailed information about a specific tool
func (m ChatModel) getToolInfoText(toolID string) string {
	// Try tool registry first
	toolRegistry := m.registry.Tools()
	if toolRegistry != nil {
		tool, err := toolRegistry.GetTool(toolID)
		if err == nil {
			var content strings.Builder
			content.WriteString(fmt.Sprintf("🔨 Tool Information: %s\n\n", m.toolStyle.Render(toolID)))
			content.WriteString(fmt.Sprintf("Name: %s\n", toolID))

			// Try to get schema and description
			if toolWithSchema, ok := tool.(interface{ Schema() map[string]interface{} }); ok {
				schema := toolWithSchema.Schema()
				if desc, exists := schema["description"]; exists {
					content.WriteString(fmt.Sprintf("Description: %s\n", desc))
				}

				content.WriteString("\nParameters:\n")
				if params, exists := schema["parameters"]; exists {
					content.WriteString(fmt.Sprintf("  %v\n", params))
				} else {
					content.WriteString("  See tool documentation for parameter details\n")
				}
			}

			// Try to get examples
			if toolWithExamples, ok := tool.(interface{ Examples() []string }); ok {
				examples := toolWithExamples.Examples()
				if len(examples) > 0 {
					content.WriteString("\nExamples:\n")
					for _, example := range examples {
						content.WriteString(fmt.Sprintf("  %s\n", example))
					}
				}
			}

			return content.String()
		}
	}

	// Fallback to mock data for common tools
	switch toolID {
	case "file-reader":
		return `🔨 Tool Information: file-reader

Name: File Reader
Description: Read and analyze file contents
Version: 1.0.0
Capabilities: file-system, text-processing
Cost: Free

Parameters:
  --path (required): Path to file
  --encoding (optional): File encoding (default: utf-8)

Examples:
  /tool file-reader --path ./README.md
  /tool file-reader --path ./src/main.go --encoding utf-8`

	case "shell-exec":
		return `🔨 Tool Information: shell-exec

Name: Shell Executor
Description: Execute shell commands safely
Version: 1.0.0
Capabilities: system, execution
Cost: Low (based on execution time)

Parameters:
  --command (required): Command to execute
  --timeout (optional): Timeout in seconds (default: 30)
  --workdir (optional): Working directory

Examples:
  /tool shell-exec --command "ls -la"
  /tool shell-exec --command "git status" --workdir ./project`

	default:
		return fmt.Sprintf("Tool '%s' not found. Use '/tools list' to see available tools.", toolID)
	}
}

// searchToolsText searches for tools by capability
func (m ChatModel) searchToolsText(capability string) string {
	// Try tool registry search
	toolRegistry := m.registry.Tools()
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

				content.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, m.toolStyle.Render(toolName), desc))
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
		content.WriteString(fmt.Sprintf("%d. %s\n", i+1, m.toolStyle.Render(tool)))
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

	for i, execID := range m.activeTools {
		if execution, exists := m.toolExecutions[execID]; exists {
			duration := time.Since(execution.StartTime)
			if execution.EndTime != nil {
				duration = execution.EndTime.Sub(execution.StartTime)
			}

			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, m.toolStyle.Render(execution.ToolName)))
			content.WriteString(fmt.Sprintf("   ID: %s\n", execution.ID))
			content.WriteString(fmt.Sprintf("   Agent: %s\n", execution.AgentID))
			content.WriteString(fmt.Sprintf("   Status: %s\n", execution.Status))
			content.WriteString(fmt.Sprintf("   Duration: %v\n", duration.Round(time.Millisecond)))
			if execution.Progress > 0 {
				content.WriteString(fmt.Sprintf("   Progress: %.1f%%\n", execution.Progress*100))
			}
			if execution.Cost > 0 {
				content.WriteString(fmt.Sprintf("   Cost: $%.4f\n", execution.Cost))
			}
			content.WriteString("\n")
		}
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
		ToolID:     toolID,
		ToolName:   toolID, // Will be updated with real name
		AgentID:    "chat-user",
		StartTime:  time.Now(),
		Status:     "starting",
		Progress:   0.0,
		Parameters: params,
	}

	// Add to tracking
	m.toolExecutions[execID] = execution
	m.activeTools = append(m.activeTools, execID)

	// TODO: Implement actual tool execution via registry
	// For now, return mock response
	go m.simulateToolExecution(execID)

	return fmt.Sprintf("🔨 Started tool execution: %s\nExecution ID: %s\nParameters: %v\n\nUse '/tools status' to monitor progress.",
		m.toolStyle.Render(toolID), execID, params)
}

// simulateToolExecution simulates tool execution (placeholder)
func (m *ChatModel) simulateToolExecution(execID string) {
	// This would be replaced with actual tool execution
	execution := m.toolExecutions[execID]
	if execution == nil {
		return
	}

	// Simulate progress updates
	for i := 0; i <= 100; i += 25 {
		time.Sleep(500 * time.Millisecond)
		execution.Progress = float64(i) / 100.0
		execution.Status = "running"

		// TODO: Send progress update via tea.Cmd
	}

	// Complete execution
	now := time.Now()
	execution.EndTime = &now
	execution.Status = "completed"
	execution.Result = "Tool execution completed successfully"
	execution.Cost = 0.001 // Mock cost

	// Remove from active tools
	for i, id := range m.activeTools {
		if id == execID {
			m.activeTools = append(m.activeTools[:i], m.activeTools[i+1:]...)
			break
		}
	}

	// TODO: Send completion message via tea.Cmd
}

// formatActiveToolExecutions formats the active tool executions for display
func (m ChatModel) formatActiveToolExecutions() string {
	if len(m.activeTools) == 0 {
		return "  None\n"
	}

	var content strings.Builder
	for _, execID := range m.activeTools {
		if execution, exists := m.toolExecutions[execID]; exists {
			progress := ""
			if execution.Progress > 0 {
				progress = fmt.Sprintf(" (%.0f%%)", execution.Progress*100)
			}
			content.WriteString(fmt.Sprintf("  %s - %s%s\n",
				execution.ToolName, execution.Status, progress))
		}
	}

	return content.String()
}

// Tool execution message handlers

// handleToolExecutionStart handles the start of a tool execution
func (m *ChatModel) handleToolExecutionStart(msg toolExecutionStartMsg) {
	// Create or update tool execution record
	execution := &toolExecution{
		ID:         msg.executionID,
		ToolID:     msg.toolID,
		ToolName:   msg.toolName,
		AgentID:    msg.agentID,
		StartTime:  time.Now(),
		Status:     "starting",
		Progress:   0.0,
		Parameters: msg.parameters,
		Cost:       msg.cost,
	}

	m.toolExecutions[msg.executionID] = execution

	// Add to active tools if not already present
	found := false
	for _, id := range m.activeTools {
		if id == msg.executionID {
			found = true
			break
		}
	}
	if !found {
		m.activeTools = append(m.activeTools, msg.executionID)
	}

	// Create a chat message for tool start
	var paramStr strings.Builder
	if len(msg.parameters) > 0 {
		paramStr.WriteString(" with parameters:")
		for key, value := range msg.parameters {
			paramStr.WriteString(fmt.Sprintf("\n   %s: %s", key, value))
		}
	}

	costInfo := ""
	if msg.cost > 0 {
		costInfo = fmt.Sprintf(" (estimated cost: $%.4f)", msg.cost)
	}

	toolMsg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		AgentID:   msg.agentID, // Associate with specific agent
		Content: fmt.Sprintf("🔨 Started tool execution: %s by %s%s%s",
			m.toolStyle.Render(msg.toolName),
			m.agentStyle.Render(msg.agentID),
			paramStr.String(),
			costInfo),
		Type: msgToolStart,
	}
	m.addMessage(toolMsg)
}

// handleToolExecutionProgress handles progress updates for tool execution
func (m *ChatModel) handleToolExecutionProgress(msg toolExecutionProgressMsg) {
	execution, exists := m.toolExecutions[msg.executionID]
	if !exists {
		return // Execution not found, ignore
	}

	// Update execution progress
	execution.Progress = msg.progress
	execution.Status = "running"

	// Create progress bar
	progressBar := m.createProgressBar(msg.progress)

	// Add progress message (limit frequency to avoid spam)
	lastProgress := execution.Progress
	if msg.progress-lastProgress >= 0.1 || msg.progress >= 1.0 {
		progressMsg := chatMessage{
			Timestamp: time.Now(),
			Sender:    "system",
			AgentID:   execution.AgentID, // Associate with specific agent
			Content: fmt.Sprintf("🔨 %s progress: %s %.1f%% - %s",
				m.toolStyle.Render(execution.ToolName),
				progressBar,
				msg.progress*100,
				msg.message),
			Type: msgToolProgress,
		}
		m.addMessage(progressMsg)
	}
}

// handleToolExecutionComplete handles completion of tool execution
func (m *ChatModel) handleToolExecutionComplete(msg toolExecutionCompleteMsg) {
	execution, exists := m.toolExecutions[msg.executionID]
	if !exists {
		return // Execution not found, ignore
	}

	// Update execution status
	now := time.Now()
	execution.EndTime = &now
	execution.Status = "completed"
	execution.Result = msg.result
	execution.Progress = 1.0
	execution.Cost = msg.cost // Final cost

	// Remove from active tools
	for i, id := range m.activeTools {
		if id == msg.executionID {
			m.activeTools = append(m.activeTools[:i], m.activeTools[i+1:]...)
			break
		}
	}

	// Calculate duration
	duration := time.Since(execution.StartTime)

	// Format final cost
	costInfo := ""
	if msg.cost > 0 {
		costInfo = fmt.Sprintf(" (final cost: $%.4f)", msg.cost)
	}

	// Create completion message
	completionMsg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		AgentID:   execution.AgentID, // Associate with specific agent
		Content: fmt.Sprintf("🔨 %s completed in %v%s\n\nResult:\n%s",
			m.toolStyle.Render(execution.ToolName),
			duration.Round(time.Millisecond),
			costInfo,
			msg.result),
		Type: msgToolComplete,
	}
	m.addMessage(completionMsg)
}

// handleToolExecutionError handles errors during tool execution
func (m *ChatModel) handleToolExecutionError(msg toolExecutionErrorMsg) {
	execution, exists := m.toolExecutions[msg.executionID]
	if !exists {
		return // Execution not found, ignore
	}

	// Update execution status
	now := time.Now()
	execution.EndTime = &now
	execution.Status = "failed"
	execution.Error = msg.error.Error()

	// Remove from active tools
	for i, id := range m.activeTools {
		if id == msg.executionID {
			m.activeTools = append(m.activeTools[:i], m.activeTools[i+1:]...)
			break
		}
	}

	// Calculate duration
	duration := time.Since(execution.StartTime)

	// Create error message
	errorMsg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		AgentID:   execution.AgentID, // Associate with specific agent
		Content: fmt.Sprintf("🔨 %s failed after %v\n\nError: %v",
			m.toolStyle.Render(execution.ToolName),
			duration.Round(time.Millisecond),
			msg.error),
		Type: msgToolError,
	}
	m.addMessage(errorMsg)
}

// handleToolAuthRequired handles authorization requests for tool execution
func (m *ChatModel) handleToolAuthRequired(msg toolAuthRequiredMsg) {
	execution, exists := m.toolExecutions[msg.executionID]
	if !exists {
		return // Execution not found, ignore
	}

	// Update execution status
	execution.Status = "awaiting_authorization"

	// Format permissions list
	permsList := strings.Join(msg.permissions, ", ")

	// Format estimated cost
	costInfo := ""
	if msg.estimatedCost > 0 {
		costInfo = fmt.Sprintf("\nEstimated cost: $%.4f", msg.estimatedCost)
	}

	// Create authorization request message
	authMsg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		AgentID:   execution.AgentID, // Associate with specific agent
		Content: fmt.Sprintf("🔒 %s requires authorization\n\nPermissions needed: %s%s\n\nReply with /authorize %s to proceed or /cancel %s to abort.",
			m.toolStyle.Render(execution.ToolName),
			permsList,
			costInfo,
			msg.executionID,
			msg.executionID),
		Type: msgToolAuth,
	}
	m.addMessage(authMsg)
}

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

` + "```%s" + `
// Sample code in %s
function example() {
    console.log("This is %s code with syntax highlighting!");
    return "syntax-highlighted";
}
` + "```" + `

**Try these languages**: go, python, javascript, sql, bash, yaml`, language, language, language, language)
	}
}

// handleTestCommand processes /test commands for demonstrating rich content rendering
func (m ChatModel) handleTestCommand(args []string) string {
	if len(args) == 0 {
		return m.getTestHelpText()
	}

	switch args[0] {
	case "markdown":
		return m.generateMarkdownTestContent()

	case "code":
		if len(args) < 2 {
			return "Usage: /test code <language>\nExample: /test code go"
		}
		return m.generateCodeTestContent(args[1])

	case "mixed":
		return m.generateMixedTestContent()

	default:
		return fmt.Sprintf("Unknown test command: %s. Use /test to see available options.", args[0])
	}
}

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
func (m ChatModel) handleTabCompletion() (ChatModel, tea.Cmd) {
	// Get current input text and cursor position
	input := m.input.Value()
	cursorPos := len(input) // Use input length as cursor position for now

	// If already showing completion, cycle to next option
	if m.showingCompletion && len(m.completionResults) > 0 {
		m.completionIndex = (m.completionIndex + 1) % len(m.completionResults)
		return m, nil
	}

	// Get completion suggestions
	completions := m.completionEngine.Complete(input, cursorPos)
	if len(completions) == 0 {
		// No completions available - just insert tab spaces
		return m, nil
	}

	// Store completion results and show popup
	m.completionResults = completions
	m.completionIndex = 0
	m.showingCompletion = true

	// If only one completion, auto-apply it
	if len(completions) == 1 {
		completion := completions[0]
		m.input.SetValue(completion.Text)
		m.input.CursorEnd()
		m.showingCompletion = false
		m.completionResults = nil
	}

	return m, nil
}

// handleUpKey handles up arrow for history navigation or completion navigation
func (m ChatModel) handleUpKey() (ChatModel, tea.Cmd) {
	// If showing completion popup, navigate up in completion list
	if m.showingCompletion && len(m.completionResults) > 0 {
		if m.completionIndex > 0 {
			m.completionIndex--
		} else {
			m.completionIndex = len(m.completionResults) - 1 // Wrap to bottom
		}
		return m, nil
	}

	// Otherwise, navigate command history
	if previousCommand := m.commandHistory.Previous(); previousCommand != "" {
		m.input.SetValue(previousCommand)
		m.input.CursorEnd()
	}

	return m, nil
}

// handleDownKey handles down arrow for history navigation or completion navigation
func (m ChatModel) handleDownKey() (ChatModel, tea.Cmd) {
	// If showing completion popup, navigate down in completion list
	if m.showingCompletion && len(m.completionResults) > 0 {
		if m.completionIndex < len(m.completionResults)-1 {
			m.completionIndex++
		} else {
			m.completionIndex = 0 // Wrap to top
		}
		return m, nil
	}

	// Otherwise, navigate command history
	if nextCommand := m.commandHistory.Next(); nextCommand != "" {
		m.input.SetValue(nextCommand)
		m.input.CursorEnd()
	}

	return m, nil
}

// handleSearchHistory handles Ctrl+R for fuzzy history search
func (m ChatModel) handleSearchHistory() (ChatModel, tea.Cmd) {
	// Get current input as search term
	searchTerm := m.input.Value()

	// If input is empty, show recent commands
	if searchTerm == "" {
		recent := m.commandHistory.GetRecent(5)
		if len(recent) > 0 {
			// Show first recent command
			m.input.SetValue(recent[0])
			m.input.CursorEnd()
		}
		return m, nil
	}

	// Perform fuzzy search
	results := m.commandHistory.Search(searchTerm)
	if len(results) > 0 {
		// Set first result
		m.input.SetValue(results[0])
		m.input.CursorEnd()

		// Store as completion results for cycling
		m.completionResults = make([]CompletionResult, len(results))
		for i, result := range results {
			m.completionResults[i] = CompletionResult{
				Text:        result,
				Description: "History match",
				Type:        "history",
			}
		}
		m.completionIndex = 0
		m.showingCompletion = true
	}

	return m, nil
}

// handleEscape handles escape key for canceling completion or search
func (m ChatModel) handleEscape() (ChatModel, tea.Cmd) {
	// Cancel completion popup
	if m.showingCompletion {
		m.showingCompletion = false
		m.completionResults = nil
		m.completionIndex = 0
		return m, nil
	}

	// Clear current input as fallback
	m.input.Reset()
	return m, nil
}

// Integration methods (Agent 4)

// InitializeAllComponents initializes and validates all chat components
func (m *ChatModel) InitializeAllComponents() error {
	// Initialize rich content rendering
	if m.markdownRenderer == nil {
		renderer, err := NewMarkdownRenderer(m.width)
		if err != nil {
			// Log error but continue - graceful degradation
			m.integrationFlags["markdown_failed"] = true
		} else {
			m.markdownRenderer = renderer
			m.integrationFlags["markdown_enabled"] = true
		}
	}

	// Initialize content formatter
	if m.contentFormatter == nil {
		m.contentFormatter = NewContentFormatter(m.markdownRenderer, m.width)
		m.integrationFlags["formatter_enabled"] = true
	}

	// Initialize completion engine
	if m.completionEngine == nil {
		m.completionEngine = NewCompletionEngine(m.guildConfig, ".")
		m.integrationFlags["completion_enabled"] = true
	}

	// Initialize command history
	if m.commandHistory == nil {
		m.commandHistory = NewCommandHistory(".guild/chat_history.txt")
		m.integrationFlags["history_enabled"] = true
	}

	// Initialize status tracker
	if m.statusTracker == nil {
		m.statusTracker = NewAgentStatusTracker(m.guildConfig)
		m.statusTracker.StartTracking()
		m.integrationFlags["status_tracking_enabled"] = true
	}

	// Initialize status display
	if m.statusDisplay == nil && m.statusTracker != nil {
		m.statusDisplay = NewStatusDisplay(m.statusTracker, m.width/4, m.width/3)
		m.integrationFlags["status_display_enabled"] = true
	}

	// Initialize agent indicators
	if m.agentIndicators == nil {
		m.agentIndicators = NewAgentIndicators()
		m.agentIndicators.SetupDefaultAnimations()
		m.integrationFlags["indicators_enabled"] = true
	}

	// Validate all components
	return m.ValidateAllComponents()
}

// ValidateAllComponents ensures all components are properly configured
func (m *ChatModel) ValidateAllComponents() error {
	// Check critical components
	if m.guildConfig == nil {
		return fmt.Errorf("guild configuration is not loaded")
	}

	if m.ctx == nil {
		return fmt.Errorf("context is not initialized")
	}

	// Log component status
	enabledCount := 0
	for feature, enabled := range m.integrationFlags {
		if enabled {
			enabledCount++
		} else {
			// Log warning for disabled features
			if feature == "markdown_failed" {
				// This is okay - we have graceful degradation
				continue
			}
		}
	}

	// Ensure minimum components are enabled
	if enabledCount < 3 {
		return fmt.Errorf("insufficient components enabled: only %d of minimum 3", enabledCount)
	}

	return nil
}

// HandleIntegratedKeyInput processes keyboard input with all components integrated
func (m ChatModel) HandleIntegratedKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle completion-aware input first
	if m.showingCompletion {
		switch {
		case key.Matches(msg, m.keymap.Tab):
			// Cycle through completions
			if len(m.completionResults) > 0 {
				m.completionIndex = (m.completionIndex + 1) % len(m.completionResults)
				// Apply current completion
				m.input.SetValue(m.completionResults[m.completionIndex].Text)
				m.input.CursorEnd()
			}
			return m, nil

		case key.Matches(msg, m.keymap.Escape):
			// Cancel completion
			m.showingCompletion = false
			m.completionResults = nil
			m.completionIndex = 0
			return m, nil

		case key.Matches(msg, m.keymap.Send):
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
func (m *ChatModel) ProcessIntegratedMessage(input string) string {
	// Add to command history
	if m.commandHistory != nil {
		m.commandHistory.Add(input)
	}

	// Process with command processor if available
	if m.commandProcessor != nil {
		// Command processor would handle advanced processing
		// For now, use the existing processMessage
	}

	return m.processMessage(input)
}

// RenderIntegratedView renders the chat view with all visual enhancements
func (m *ChatModel) RenderIntegratedView() string {
	if !m.ready {
		return "Initializing Guild Chat Chamber..."
	}

	// Build integrated view with all components
	var sections []string

	// Header with agent status indicators
	header := m.renderIntegratedHeader()
	sections = append(sections, header)

	// Messages area with rich content
	messagesView := m.borderStyle.Render(m.messages.View())
	sections = append(sections, messagesView)

	// Status panel if enabled
	if m.showAgentStatus && m.statusDisplay != nil {
		statusPanel := m.statusDisplay.RenderStatusPanel()
		sections = append(sections, statusPanel)
	}

	// Input area with completion popup
	inputView := m.renderIntegratedInput()
	sections = append(sections, inputView)

	// Help area
	helpView := m.help.View(m.keymap)
	sections = append(sections, helpView)

	// Combine all sections
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderIntegratedHeader creates header with live agent status
func (m *ChatModel) renderIntegratedHeader() string {
	// Get active agent count
	activeCount := 0
	if m.statusTracker != nil {
		activeAgents := m.statusTracker.GetActiveAgents()
		activeCount = len(activeAgents)
	}

	// Build header with status
	viewInfo := "Global View"
	if m.chatMode == agentView && m.currentAgent != "" {
		viewInfo = fmt.Sprintf("Agent: %s", m.currentAgent)
	}

	// Add live status indicators
	statusIndicator := ""
	if activeCount > 0 {
		statusIndicator = fmt.Sprintf(" | %d agents active", activeCount)
	}

	return m.headerStyle.Render(fmt.Sprintf(
		"🏰 Guild Chat | %s | Campaign: %s | Session: %s%s",
		viewInfo,
		m.getCampaignDisplay(),
		m.sessionID[:8],
		statusIndicator,
	))
}

// renderIntegratedInput creates input area with completion popup
func (m *ChatModel) renderIntegratedInput() string {
	inputView := m.inputStyle.Render(m.input.View())

	// Add completion popup if active
	if m.showingCompletion && len(m.completionResults) > 0 {
		completionView := m.renderCompletionPopup()
		// Stack completion above input
		return lipgloss.JoinVertical(lipgloss.Left, completionView, inputView)
	}

	return inputView
}

// renderCompletionPopup creates the auto-completion popup
func (m *ChatModel) renderCompletionPopup() string {
	var items []string

	for i, result := range m.completionResults {
		// Highlight current selection
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if i == m.completionIndex {
			style = style.Bold(true).Foreground(lipgloss.Color("212"))
		}

		// Format item
		item := fmt.Sprintf("%s - %s", result.Text, result.Description)
		items = append(items, style.Render(item))
	}

	// Create popup box
	popup := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(0, 1).
		Render(strings.Join(items, "\n"))

	return popup
}

// addSystemMessage adds a system message to the chat
func (m *ChatModel) addSystemMessage(content string) {
	msg := chatMessage{
		Timestamp: time.Now(),
		Sender:    "system",
		Content:   content,
		Type:      msgSystem,
	}
	m.addMessage(msg)
	m.updateMessagesView()
}

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
