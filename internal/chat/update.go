package chat

import (
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/pkg/chat/session"
)

// Update implements tea.Model
func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 3 // Leave room for input
		m.input.SetWidth(msg.Width - 2)
		m.ready = true

		// Update visual components with new dimensions
		if m.statusDisplay != nil {
			m.statusDisplay.SetDimensions(msg.Width, msg.Height)
		}
		if m.contentFormatter != nil {
			m.contentFormatter.SetWidth(msg.Width)
		}

	case tea.KeyMsg:
		// Handle command palette navigation first if open
		if m.commandPalette != nil && m.commandPalette.IsOpen() {
			return m.handleCommandPaletteKey(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Submit):
			return m.handleSendMessage()

		case key.Matches(msg, m.keys.Help):
			m.viewMode = chatModeNormal
			helpMsg := m.getHelpText()
			m.messages = append(m.messages, Message{
				Type:      msgSystem,
				Content:   helpMsg,
				Timestamp: time.Now(),
			})
			m.updateMessagesView()
			return m, nil

		case key.Matches(msg, m.keys.Prompt):
			if m.viewMode == chatModePrompt {
				m.viewMode = chatModeNormal
			} else {
				m.viewMode = chatModePrompt
			}
			return m, nil

		case key.Matches(msg, m.keys.Status):
			if m.viewMode == chatModeStatus {
				m.viewMode = chatModeNormal
			} else {
				m.viewMode = chatModeStatus
			}
			return m, nil

		case key.Matches(msg, m.keys.Global):
			if m.viewMode == chatModeGlobal {
				m.viewMode = chatModeNormal
			} else {
				m.viewMode = chatModeGlobal
			}
			return m, nil

		case key.Matches(msg, m.keys.ScrollUp):
			m.viewport.LineUp(1)

		case key.Matches(msg, m.keys.ScrollDown):
			m.viewport.LineDown(1)

		case key.Matches(msg, m.keys.PrevHistory):
			return m.handleUpKey()

		case key.Matches(msg, m.keys.NextHistory):
			return m.handleDownKey()

		case key.Matches(msg, m.keys.Clear):
			m.messages = []Message{}
			m.updateMessagesView()
			return m, nil

		case key.Matches(msg, m.keys.CommandPalette):
			return m.handleCommandPalette()

		case msg.String() == "tab":
			return m.handleTabCompletion()

		default:
			// Handle integrated completion features if enabled
			if m.showingCompletion {
				return m.handleCompletionKey(msg)
			}
		}

	// Agent streaming messages
	case agentStreamMsg:
		m.handleAgentStream(msg)
		return m, nil

	case agentStatusMsg:
		m.handleAgentStatus(msg)
		return m, nil

	case agentErrorMsg:
		m.handleAgentError(msg)
		return m, nil

	// Tool execution messages
	case toolExecutionStartMsg:
		m.handleToolExecutionStart(msg)
		return m, nil

	case toolExecutionProgressMsg:
		m.handleToolExecutionProgress(msg)
		return m, nil

	case toolExecutionCompleteMsg:
		m.handleToolExecutionComplete(msg)
		return m, nil

	case toolExecutionErrorMsg:
		m.handleToolExecutionError(msg)
		return m, nil

	case toolAuthRequiredMsg:
		m.handleToolAuthRequired(msg)
		return m, nil

	// Agent status updates (Agent 3)
	case AgentStatusUpdateMsg:
		m.handleAgentStatusUpdate(msg)
		return m, nil

	// Test messages (Agent 1)
	case testRichContentMsg:
		m.handleTestRichContent(msg)
		return m, nil

	case completionResultMsg:
		m.handleCompletionResult(msg)
		return m, nil

	case Message:
		m.messages = append(m.messages, msg)
		m.updateMessagesView()
		return m, nil
	}

	// Handle text input updates
	if m.viewMode == chatModeNormal {
		m.input, tiCmd = m.input.Update(msg)
	}

	// Update viewport
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

// handleAgentStream handles streaming agent responses
func (m *ChatModel) handleAgentStream(msg agentStreamMsg) {
	// Find the last message from this agent
	var lastMsgIndex = -1
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].AgentID == msg.agentID && m.messages[i].Type == msgAgent {
			lastMsgIndex = i
			break
		}
	}

	if msg.done {
		// Stream is complete
		if lastMsgIndex >= 0 {
			// Final update to ensure complete content
			m.messages[lastMsgIndex].Content = msg.content
			
			// Save complete agent message to session
			if m.sessionManager != nil && m.currentSession != nil {
				go func() {
					_, err := m.sessionManager.AppendMessage(m.currentSession.ID, session.RoleAssistant, msg.content, nil)
					if err != nil {
						log.Printf("Failed to save agent message to session: %v", err)
					}
				}()
			}
		}
		delete(m.activeTools, msg.agentID) // Clean up any streaming state
	} else {
		// Streaming update
		if lastMsgIndex >= 0 {
			// Append to existing message
			m.messages[lastMsgIndex].Content = msg.content
		} else {
			// Create new message
			m.messages = append(m.messages, Message{
				Type:      msgAgent,
				Content:   msg.content,
				AgentID:   msg.agentID,
				Timestamp: time.Now(),
			})
		}
	}
	m.updateMessagesView()
}

// handleAgentStatus handles agent status updates
func (m *ChatModel) handleAgentStatus(msg agentStatusMsg) {
	// Update agent status in the UI
	statusMsg := fmt.Sprintf("Agent %s status: %s", msg.agentID, msg.status)

	// Update status display
	if m.agentStatusTracker != nil {
		status := m.agentStatusTracker.GetAgentStatus(msg.agentID)
		if status != nil {
			// Map string status to AgentState
			switch msg.status {
			case "thinking":
				status.State = AgentThinking
			case "working":
				status.State = AgentWorking
			case "idle":
				status.State = AgentIdle
			default:
				status.State = AgentOffline
			}
			m.agentStatusTracker.UpdateAgentStatus(msg.agentID, status)
		}
	}

	log.Printf("Agent status update: %s", statusMsg)
}

// handleAgentError handles agent error messages
func (m *ChatModel) handleAgentError(msg agentErrorMsg) {
	errorMsg := fmt.Sprintf("Error from agent %s: %v", msg.agentID, msg.err)
	m.messages = append(m.messages, Message{
		Type:      msgError,
		Content:   errorMsg,
		AgentID:   msg.agentID,
		Timestamp: time.Now(),
	})
	m.updateMessagesView()
}

// handleAgentStatusUpdate handles agent status update messages from Agent 3
func (m *ChatModel) handleAgentStatusUpdate(msg AgentStatusUpdateMsg) {
	if m.agentStatusTracker != nil && msg.Status != nil {
		m.agentStatusTracker.UpdateAgentStatus(msg.AgentID, msg.Status)
	}

	if msg.Event != nil {
		// Log the activity event
		log.Printf("Agent activity: %s - %s", msg.AgentID, msg.Event.Description)
	}

	// Update UI if in status view mode
	if m.viewMode == chatModeStatus {
		// Trigger a view refresh to show updated status
		m.updateMessagesView()
	}
}
