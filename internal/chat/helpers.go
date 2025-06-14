package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/internal/ui"
	"github.com/guild-ventures/guild-core/pkg/config"
)

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

// safeFormatContent safely formats content based on message type
func (m ChatModel) safeFormatContent(msgType messageType, content string, agentID string) string {
	// Basic styling without advanced formatters
	var style lipgloss.Style
	var prefix string

	switch msgType {
	case msgUser:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
		prefix = "You"
	case msgAgent:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
		if agentID != "" {
			prefix = fmt.Sprintf("@%s", agentID)
		} else {
			prefix = "Agent"
		}
	case msgSystem:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
		prefix = "System"
	case msgError:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
		prefix = "Error"
	case msgToolStart:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
		prefix = "Tool"
	case msgToolProgress:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Italic(true)
		prefix = "Tool Progress"
	case msgToolComplete:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
		prefix = "Tool Complete"
	case msgAgentThinking:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Italic(true)
		prefix = "Thinking"
	case msgAgentWorking:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
		prefix = "Working"
	default:
		style = lipgloss.NewStyle()
		prefix = "Message"
	}

	// Format the message
	header := style.Render(fmt.Sprintf("[%s]", prefix))
	return fmt.Sprintf("%s %s", header, content)
}

// getHelpText returns formatted help text
func (m ChatModel) getHelpText() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	commandStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	platformStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)

	// Add platform information header
	var help string
	if m.keyAdapter != nil {
		help = platformStyle.Render(m.keyAdapter.GetPlatformHelpText()) + "\n\n"
	}
	
	help += helpStyle.Render("🏰 Guild Chat Commands:\n\n")

	// Text commands
	textCommands := []struct {
		cmd  string
		desc string
	}{
		{"/help, /h", "Show this help message"},
		{"/status, /s", "Show current guild status"},
		{"/agents, /a", "List available agents"},
		{"/prompt, /p", "Manage prompt layers"},
		{"/tools, /t", "List available tools"},
		{"/clear, /c", "Clear chat history"},
		{"/test", "Test visual features"},
		{"/sessions", "List all sessions"},
		{"/session new [name]", "Create new session"},
		{"/session switch <id>", "Switch to different session"},
		{"/session rename <name>", "Rename current session"},
		{"/session export [format]", "Export current session"},
		{"/exit, /quit, /q", "Exit chat"},
		{"@agent message", "Send message to specific agent"},
		{"@all message", "Broadcast to all agents"},
	}

	help += helpStyle.Render("Text Commands:\n")
	for _, cmd := range textCommands {
		help += fmt.Sprintf("  %s - %s\n",
			commandStyle.Render(cmd.cmd),
			helpStyle.Render(cmd.desc))
	}

	// Keyboard shortcuts
	help += "\n" + helpStyle.Render("Keyboard Shortcuts:\n")
	
	// Get platform-specific keybindings
	shortcuts := []struct {
		binding key.Binding
		desc    string
	}{
		{m.keys.Submit, "Submit message"},
		{m.keys.NewLine, "Insert new line"},
		{m.keys.Quit, "Quit chat"},
		{m.keys.Help, "Toggle help"},
		{m.keys.Prompt, "Prompt management"},
		{m.keys.Status, "Agent status"},
		{m.keys.Global, "Global view"},
		{m.keys.Clear, "Clear chat"},
		{m.keys.CommandPalette, "Command palette"},
		{m.keys.Copy, "Copy text"},
		{m.keys.Paste, "Paste text"},
		{m.keys.Search, "Search"},
		{m.keys.FuzzyFinder, "Fuzzy file finder"},
		{m.keys.GlobalSearch, "Global search"},
		{m.keys.ToggleVimMode, "Toggle Vim mode"},
	}

	for _, shortcut := range shortcuts {
		help += fmt.Sprintf("  %s - %s\n",
			commandStyle.Render(shortcut.binding.Help().Key),
			helpStyle.Render(shortcut.desc))
	}

	// Navigation
	help += "\n" + helpStyle.Render("Navigation:\n")
	navKeys := []struct {
		binding key.Binding
		desc    string
	}{
		{m.keys.ScrollUp, "Scroll up"},
		{m.keys.ScrollDown, "Scroll down"},
		{m.keys.PageUp, "Page up"},
		{m.keys.PageDown, "Page down"},
		{m.keys.Home, "Go to start"},
		{m.keys.End, "Go to end"},
		{m.keys.PrevHistory, "Previous command"},
		{m.keys.NextHistory, "Next command"},
	}

	for _, nav := range navKeys {
		help += fmt.Sprintf("  %s - %s\n",
			commandStyle.Render(nav.binding.Help().Key),
			helpStyle.Render(nav.desc))
	}

	return help
}

// getStatusText returns current guild status
func (m ChatModel) getStatusText() string {
	status := fmt.Sprintf("📊 Guild Status\n")
	status += fmt.Sprintf("Campaign: %s\n", m.campaignID)
	status += fmt.Sprintf("Session: %s\n", m.sessionID)
	
	// Add session details if available
	if m.currentSession != nil {
		status += fmt.Sprintf("Session Name: %s\n", m.currentSession.Name)
		status += fmt.Sprintf("Session Created: %s\n", m.currentSession.CreatedAt.Format("2006-01-02 15:04"))
	}
	
	status += fmt.Sprintf("Messages: %d\n", len(m.messages))
	status += fmt.Sprintf("Active Tools: %d\n", len(m.activeTools))

	if m.agentStatusTracker != nil {
		activeAgents := 0
		for _, status := range m.agentStatusTracker.agents {
			if status.State != AgentOffline {
				activeAgents++
			}
		}
		status += fmt.Sprintf("Active Agents: %d\n", activeAgents)
	}

	return status
}

// getAgentsText returns formatted list of agents
func (m ChatModel) getAgentsText() string {
	// Try to get agents from the gRPC client
	result := "🛡️ Available Agents:\n\n"

	agentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	capStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	// For now, use mock agents until we have proper agent listing
	mockAgents := []struct {
		ID           string
		Name         string
		Capabilities []string
	}{
		{"manager", "Guild Master", []string{"planning", "coordination"}},
		{"developer", "Code Artisan", []string{"coding", "testing"}},
		{"reviewer", "Review Artisan", []string{"review", "quality"}},
	}

	for _, agent := range mockAgents {
		// Get current status if available
		statusIcon := "⚫" // Default offline
		statusText := "offline"

		if m.agentStatusTracker != nil {
			if status := m.agentStatusTracker.GetAgentStatus(agent.ID); status != nil {
				switch status.State {
				case AgentIdle:
					statusIcon = "🟢"
					statusText = "idle"
				case AgentThinking:
					statusIcon = "🟡"
					statusText = "thinking"
				case AgentWorking:
					statusIcon = "🟠"
					statusText = "working"
				case AgentBlocked:
					statusIcon = "🔴"
					statusText = "blocked"
				}
			}
		}

		result += fmt.Sprintf("%s %s - %s (%s)\n",
			statusIcon,
			agentStyle.Render(fmt.Sprintf("@%s", agent.ID)),
			agent.Name,
			statusText)

		if len(agent.Capabilities) > 0 {
			result += fmt.Sprintf("   %s\n",
				capStyle.Render(fmt.Sprintf("Capabilities: %s",
					strings.Join(agent.Capabilities, ", "))))
		}
		result += "\n"
	}

	return result
}

// getToolListText returns formatted list of available tools
func (m ChatModel) getToolListText() string {
	// This would query the tool registry
	// For now, return mock data
	toolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	result := "🔨 Available Tools:\n\n"

	tools := []struct {
		name string
		desc string
		caps []string
	}{
		{
			name: "file-reader",
			desc: "Read and analyze files",
			caps: []string{"read", "search", "parse"},
		},
		{
			name: "code-writer",
			desc: "Generate and modify code",
			caps: []string{"write", "refactor", "test"},
		},
		{
			name: "web-scraper",
			desc: "Extract data from websites",
			caps: []string{"fetch", "parse", "extract"},
		},
		{
			name: "shell-executor",
			desc: "Execute shell commands",
			caps: []string{"run", "pipe", "script"},
		},
	}

	for _, tool := range tools {
		result += fmt.Sprintf("%s - %s\n",
			toolStyle.Render(tool.name),
			descStyle.Render(tool.desc))
		if len(tool.caps) > 0 {
			result += fmt.Sprintf("   Capabilities: %s\n",
				descStyle.Render(strings.Join(tool.caps, ", ")))
		}
		result += "\n"
	}

	return result
}

// getToolInfoText returns detailed info about a specific tool
func (m ChatModel) getToolInfoText(toolID string) string {
	// This would query the tool registry for specific tool
	// For now, return mock data
	return fmt.Sprintf("🔨 Tool: %s\n\nDetailed information about %s...", toolID, toolID)
}

// searchToolsByCapability searches tools by capability
func (m ChatModel) searchToolsByCapability(capability string) string {
	// This would search the tool registry
	// For now, return mock results
	return fmt.Sprintf("🔍 Tools with capability '%s':\n\n- file-reader\n- code-writer", capability)
}

// getActiveToolsStatus returns status of active tool executions
func (m ChatModel) getActiveToolsStatus() string {
	if len(m.activeTools) == 0 {
		return "No tools currently executing"
	}

	result := fmt.Sprintf("⚙️ Active Tool Executions (%d):\n\n", len(m.activeTools))

	for agentID, tool := range m.activeTools {
		elapsed := time.Since(tool.StartTime)
		result += fmt.Sprintf("• %s by @%s\n", tool.ToolName, agentID)
		result += fmt.Sprintf("  Status: %s\n", tool.Status)
		result += fmt.Sprintf("  Duration: %s\n", elapsed.Round(time.Second))

		if float64(tool.Progress) > 0 {
			// Create progress bar
			width := 20
			filled := int(float64(width) * float64(tool.Progress))
			bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
			result += fmt.Sprintf("  Progress: [%s] %.0f%%\n", bar, float64(tool.Progress)*100)
		}
		result += "\n"
	}

	return result
}

// Tool execution message handlers
func (m *ChatModel) handleToolExecutionStart(msg toolExecutionStartMsg) {
	// Track the tool execution
	m.activeTools[msg.executionID] = &toolExecution{
		ID:        msg.executionID,
		ToolName:  msg.toolName,
		AgentID:   msg.agentID,
		Status:    "starting",
		StartTime: time.Now(),
		Progress:  0,
	}

	// Add message
	m.messages = append(m.messages, Message{
		Type:      msgToolStart,
		Content:   fmt.Sprintf("🔨 %s: Starting %s", msg.agentID, msg.toolName),
		AgentID:   msg.agentID,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"execution_id": msg.executionID,
			"tool_name":    msg.toolName,
		},
	})
	m.updateMessagesView()
}

func (m *ChatModel) handleToolExecutionProgress(msg toolExecutionProgressMsg) {
	if tool, exists := m.activeTools[msg.executionID]; exists {
		tool.Status = "running"
		tool.Progress = msg.progress

		// Update or add progress message
		progressContent := fmt.Sprintf("⚙️ Execution %s: %.0f%%",
			msg.executionID, float64(msg.progress)*100)

		// Find and update last progress message for this tool
		updated := false
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Type == msgToolProgress &&
				m.messages[i].Metadata["execution_id"] == msg.executionID {
				m.messages[i].Content = progressContent
				updated = true
				break
			}
		}

		if !updated {
			m.messages = append(m.messages, Message{
				Type:      msgToolProgress,
				Content:   progressContent,
				AgentID:   tool.AgentID,
				Timestamp: time.Now(),
				Metadata: map[string]string{
					"tool_id": msg.executionID,
				},
			})
		}
		m.updateMessagesView()
	}
}

// Fixed handlers are in helpers_fixed.go

// handleFuzzyFinder initializes and shows the fuzzy file finder
func (m *ChatModel) handleFuzzyFinder() (tea.Model, tea.Cmd) {
	// Get current working directory
	workingDir := "."

	// Initialize fuzzy finder if not already created
	if m.fuzzyFinder == nil {
		m.fuzzyFinder = ui.NewFuzzyFinderModel(workingDir,
			ui.WithFileSelectedCallback(func(filePath string) {
				// Add message about file selection
				m.messages = append(m.messages, Message{
					Type:      msgSystem,
					Content:   fmt.Sprintf("📁 Selected file: %s", filePath),
					Timestamp: time.Now(),
				})
				m.updateMessagesView()
				m.viewMode = chatModeNormal
			}),
			ui.WithCloseCallback(func() {
				m.viewMode = chatModeNormal
			}),
		)
	}

	// Switch to fuzzy finder mode
	m.viewMode = chatModeFuzzyFinder
	return m, m.fuzzyFinder.Init()
}

// handleGlobalSearch initializes and shows the global search
func (m *ChatModel) handleGlobalSearch() (tea.Model, tea.Cmd) {
	// Get current working directory
	workingDir := "."

	// Initialize global search if not already created
	if m.globalSearch == nil {
		m.globalSearch = ui.NewGlobalSearchModel(workingDir,
			ui.WithLocationSelectedCallback(func(filePath string, line int, column int) {
				// Add message about location selection
				m.messages = append(m.messages, Message{
					Type:      msgSystem,
					Content:   fmt.Sprintf("🔍 Jump to: %s:%d:%d", filePath, line, column),
					Timestamp: time.Now(),
				})
				m.updateMessagesView()
				m.viewMode = chatModeNormal
			}),
			ui.WithGlobalSearchCloseCallback(func() {
				m.viewMode = chatModeNormal
			}),
		)
	}

	// Switch to global search mode
	m.viewMode = chatModeGlobalSearch
	return m, m.globalSearch.Init()
}

// handleNewLine handles Shift+Enter for multiline input
func (m ChatModel) handleNewLine() (tea.Model, tea.Cmd) {
	// The textarea component already has built-in support for multiline
	// We just need to insert a newline at the current position
	currentValue := m.input.Value()
	
	// Get line info to find cursor position
	lineInfo := m.input.LineInfo()
	currentLine := m.input.Line()
	
	// Calculate the absolute position in the text
	lines := strings.Split(currentValue, "\n")
	absPos := 0
	for i := 0; i < currentLine && i < len(lines); i++ {
		absPos += len(lines[i]) + 1 // +1 for newline
	}
	absPos += lineInfo.CharOffset
	
	// Insert newline at cursor position
	before := ""
	if absPos <= len(currentValue) {
		before = currentValue[:absPos]
	}
	after := ""
	if absPos < len(currentValue) {
		after = currentValue[absPos:]
	}
	
	newValue := before + "\n" + after
	m.input.SetValue(newValue)
	
	// The textarea will handle cursor positioning
	
	return m, nil
}

// handleToggleVimMode toggles vim mode on/off
func (m ChatModel) handleToggleVimMode() (tea.Model, tea.Cmd) {
	m.vimModeEnabled = !m.vimModeEnabled
	
	if m.vimModeEnabled {
		// Initialize vim state if not already created
		if m.vimState == nil {
			m.vimState = NewVimState()
			m.vimKeys = newVimKeyMap()
		}
		// Switch to normal mode and blur input
		m.vimState.Mode = ModeNormal
		m.input.Blur()
		
		// Add status message
		msg := Message{
			Type:      msgSystem,
			Content:   "⚔️ Vim mode ENABLED - Press 'i' to enter insert mode, 'esc' to return to normal mode",
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, msg)
	} else {
		// Disable vim mode and focus input
		m.input.Focus()
		
		// Add status message
		msg := Message{
			Type:      msgSystem,
			Content:   "🖱️ Vim mode DISABLED - Normal input mode restored",
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, msg)
	}
	
	m.updateMessagesView()
	return m, nil
}

// handleCopy handles copying selected text to clipboard
func (m ChatModel) handleCopy() (tea.Model, tea.Cmd) {
	// Get text from input area
	textToCopy := m.input.Value()
	
	// If input is empty, try to copy the last message instead
	if textToCopy == "" && len(m.messages) > 0 {
		// Find the last non-system message
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Type != msgSystem {
				textToCopy = m.messages[i].Content
				break
			}
		}
	}
	
	if textToCopy == "" {
		// Add message indicating nothing to copy
		msg := Message{
			Type:      msgSystem,
			Content:   "📋 Nothing to copy",
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, msg)
		m.updateMessagesView()
		return m, nil
	}
	
	// Store in internal clipboard for paste operation
	m.clipboard = textToCopy
	
	msg := Message{
		Type:      msgSystem,
		Content:   fmt.Sprintf("📋 Copied %d characters to clipboard", len(textToCopy)),
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, msg)
	m.updateMessagesView()
	
	return m, nil
}

// handlePaste handles pasting text from clipboard
func (m ChatModel) handlePaste() (tea.Model, tea.Cmd) {
	// Check if we have anything in the internal clipboard
	if m.clipboard == "" {
		msg := Message{
			Type:      msgSystem,
			Content:   "📋 Nothing to paste",
			Timestamp: time.Now(),
		}
		m.messages = append(m.messages, msg)
		m.updateMessagesView()
		return m, nil
	}
	
	// Get current input value
	currentValue := m.input.Value()
	
	// Get line info to find cursor position
	lineInfo := m.input.LineInfo()
	currentLine := m.input.Line()
	
	// Calculate the absolute position in the text
	lines := strings.Split(currentValue, "\n")
	absPos := 0
	for i := 0; i < currentLine && i < len(lines); i++ {
		absPos += len(lines[i]) + 1 // +1 for newline
	}
	absPos += lineInfo.CharOffset
	
	// Insert clipboard content at cursor position
	before := ""
	if absPos <= len(currentValue) {
		before = currentValue[:absPos]
	}
	after := ""
	if absPos < len(currentValue) {
		after = currentValue[absPos:]
	}
	
	newValue := before + m.clipboard + after
	m.input.SetValue(newValue)
	
	msg := Message{
		Type:      msgSystem,
		Content:   fmt.Sprintf("📋 Pasted %d characters", len(m.clipboard)),
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, msg)
	m.updateMessagesView()
	
	return m, nil
}
