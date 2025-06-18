// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/internal/chat/commands"
	"github.com/guild-ventures/guild-core/internal/chat/session"
	"github.com/guild-ventures/guild-core/internal/ui"
	"github.com/guild-ventures/guild-core/pkg/config"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
	"github.com/guild-ventures/guild-core/pkg/registry"
)

// chatViewMode represents different view modes in the chat interface
type chatViewMode int

const (
	chatModeNormal chatViewMode = iota
	chatModePrompt
	chatModeStatus
	chatModeGlobal
	chatModeFuzzyFinder
	chatModeGlobalSearch
)

// messageType represents different types of messages in the chat
type messageType int

const (
	msgUser messageType = iota
	msgAgent
	msgSystem
	msgError
	msgToolStart
	msgToolProgress
	msgToolComplete
	msgAgentThinking
	msgAgentWorking
	msgPrompt
	msgToolError
	msgToolAuth
)

// chatMessage represents a single message in the chat history
type Message struct {
	Type      messageType
	Content   string
	AgentID   string
	Timestamp time.Time
	Metadata  map[string]string
}

// toolExecution represents an active tool execution
type toolExecution struct {
	ID          string
	ToolName    string
	AgentID     string
	StartTime   time.Time
	EndTime     *time.Time
	Status      string
	Progress    float32
	Result      string
	Output      string
	Error       string
	Cost        float64
	Parameters  map[string]string
	WorkspaceID string
}

// CompletionResult represents the result of a completion request
type CompletionResult struct {
	Content  string
	AgentID  string
	Metadata map[string]string
}

// Command represents a command in the command palette
type Command struct {
	Name        string
	Description string
	Category    string
	Action      func(m *ChatModel) tea.Cmd
	Shortcut    string
}

// Message types for tea.Cmd communication
type agentStreamMsg struct {
	agentID string
	content string
	done    bool
}

type agentStatusMsg struct {
	agentID string
	status  string
}

type agentErrorMsg struct {
	agentID string
	err     error
}

type toolExecutionStartMsg struct {
	executionID string
	toolName    string
	agentID     string
	parameters  map[string]string
}

type toolExecutionProgressMsg struct {
	executionID string
	progress    float32
}

type toolExecutionCompleteMsg struct {
	executionID string
	result      string
}

type toolExecutionErrorMsg struct {
	executionID string
	err         error
}

type toolAuthRequiredMsg struct {
	toolName string
	authURL  string
	message  string
}

// AgentStatusUpdateMsg represents an agent status update
type AgentStatusUpdateMsg struct {
	AgentID string
	Status  *AgentStatus
	Event   *ActivityEvent
}

// chatKeyMap defines all key bindings for the chat interface
type chatKeyMap struct {
	Submit         key.Binding
	NewLine        key.Binding
	Quit           key.Binding
	Help           key.Binding
	Prompt         key.Binding
	Status         key.Binding
	Global         key.Binding
	ScrollUp       key.Binding
	ScrollDown     key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	Home           key.Binding
	End            key.Binding
	PrevHistory    key.Binding
	NextHistory    key.Binding
	Clear          key.Binding
	ToggleViewMode key.Binding
	CommandPalette key.Binding
	CancelTool     key.Binding
	RetryCost      key.Binding
	AcceptCost     key.Binding
	SelectNext     key.Binding
	SelectPrev     key.Binding
	Copy           key.Binding
	Paste          key.Binding
	Search         key.Binding
	NextMatch      key.Binding
	PrevMatch      key.Binding
	FuzzyFinder    key.Binding
	GlobalSearch   key.Binding
	ToggleVimMode  key.Binding
}

// ChatModel represents the main chat application state
type ChatModel struct {
	// UI Components
	input        textarea.Model
	viewport     viewport.Model
	help         help.Model
	width        int
	height       int
	ready        bool
	err          error
	viewMode     chatViewMode
	keys         chatKeyMap
	keyAdapter   *KeybindingAdapter
	focusedAgent string

	// Vim mode support
	vimState       *VimState
	vimKeys        vimKeyMap
	vimModeEnabled bool

	// Visual Components
	markdownRenderer   *MarkdownRenderer
	contentFormatter   *ContentFormatter
	agentStatusTracker *AgentStatusTracker
	statusDisplay      *StatusDisplay
	agentIndicators    *AgentIndicators

	// Core Components
	grpcClient     pb.GuildClient
	promptsClient  promptspb.PromptServiceClient
	sessionID      string
	campaignID     string
	selectedGuild  string
	guildConfig    *config.GuildConfig
	commandProc    *CommandProcessor
	completionEng  *CompletionEngine
	history        *CommandHistory
	commandPalette *commands.CommandPalette
	registry       registry.ComponentRegistry
	sessionManager session.SessionManager
	currentSession *session.Session

	// State
	messages      []Message
	activeTools   map[string]*toolExecution
	agents        []string
	promptLayers  []string
	searchPattern string
	searchMatches []int
	currentMatch  int
	costConsent   map[string]bool
	taskCache     map[string]string
	blockedTools  map[string]bool

	// Vim mode additional state
	cursorX         int
	cursorY         int
	showLineNumbers bool
	wrapLines       bool

	// Completion state
	showingCompletion bool
	completionResults []CompletionResult
	completionIndex   int

	// Search components
	fuzzyFinder  *ui.FuzzyFinderModel
	globalSearch *ui.GlobalSearchModel

	// Integration flags
	integrationFlags map[string]bool

	// Additional state
	shouldQuit bool
	clipboard  string
}

// Test messages
type testRichContentMsg struct {
	content  string
	agentID  string
	metadata map[string]string
}

type completionResultMsg struct {
	results []CompletionResult
}
