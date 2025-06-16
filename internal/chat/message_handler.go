// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/guild-ventures/guild-core/internal/chat/session"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
	promptspb "github.com/guild-ventures/guild-core/pkg/grpc/pb/prompts/v1"
)

// handleSendMessage processes sending a message
func (m ChatModel) handleSendMessage() (ChatModel, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return m, nil
	}

	// Clear input
	m.input.SetValue("")
	m.input.CursorStart()

	// Hide completions
	m.showingCompletion = false
	m.completionResults = nil

	// Add to history
	if m.history != nil {
		m.history.Add(input)
	}

	// Process with command processor if available
	if m.commandProc != nil {
		// Command processor would handle advanced processing
		// For now, use the existing processMessage
	}

	return m.processMessage(input)
}

// ProcessIntegratedMessage processes a message with all enhancements active
func (m *ChatModel) ProcessIntegratedMessage(input string) tea.Cmd {
	// Add user message with rich formatting
	userMsg := Message{
		Type:      msgUser,
		Content:   input,
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)

	// Check for commands
	if strings.HasPrefix(input, "/") {
		result := m.handleCommand(input)
		m.messages = append(m.messages, Message{
			Type:      msgSystem,
			Content:   result,
			Timestamp: time.Now(),
		})
		m.updateMessagesView()
		return nil
	}

	// Check for agent mentions with enhanced routing
	if strings.HasPrefix(input, "@") {
		m.handleAgentMention(input)
		return nil
	}

	// Default: broadcast to all agents with visual feedback
	m.messages = append(m.messages, Message{
		Type:      msgSystem,
		Content:   "📡 Broadcasting to all Guild agents...",
		Timestamp: time.Now(),
	})

	m.updateMessagesView()
	return nil
}

// processMessage handles message processing
func (m ChatModel) processMessage(input string) (ChatModel, tea.Cmd) {
	// Check if integration features are enabled
	if m.integrationFlags != nil && m.integrationFlags["integrated_processing"] {
		return m, m.ProcessIntegratedMessage(input)
	}

	// Add user message
	userMsg := Message{
		Type:      msgUser,
		Content:   input,
		Timestamp: time.Now(),
	}
	m.messages = append(m.messages, userMsg)

	// Save message to session if available
	if m.sessionManager != nil && m.currentSession != nil {
		go func() {
			_, err := m.sessionManager.AppendMessage(m.currentSession.ID, session.RoleUser, input, nil)
			if err != nil {
				log.Printf("Failed to save user message to session: %v", err)
			}
		}()
	}

	// Check for commands
	if strings.HasPrefix(input, "/") {
		result := m.handleCommand(input)
		m.messages = append(m.messages, Message{
			Type:      msgSystem,
			Content:   result,
			Timestamp: time.Now(),
		})
		m.updateMessagesView()

		// Check if we should quit after processing the command
		if m.shouldQuit {
			return m, tea.Quit
		}
		return m, nil
	}

	// Check for agent mentions
	if strings.HasPrefix(input, "@") {
		result := m.handleAgentMention(input)
		if result != "" {
			m.messages = append(m.messages, Message{
				Type:      msgSystem,
				Content:   result,
				Timestamp: time.Now(),
			})
			m.updateMessagesView()
		}
		return m, nil
	}

	// Default: broadcast to all agents
	m.messages = append(m.messages, Message{
		Type:      msgSystem,
		Content:   "Broadcasting message to all agents...",
		Timestamp: time.Now(),
	})
	m.updateMessagesView()

	// Send to all agents
	go m.streamAgentConversation("all", input)

	return m, nil
}

// handleCommand processes slash commands
func (m ChatModel) handleCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "Invalid command"
	}

	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	// Note: command processor returns tea.Cmd, this function needs string
	// For now, use fallback handling until integration is complete

	// Fallback to basic command handling
	switch cmd {
	case "/help", "/h":
		return m.getHelpText()
	case "/status", "/s":
		return m.getStatusText()
	case "/agents", "/a":
		return m.getAgentsText()
	case "/prompt", "/p":
		return m.handlePromptCommand(args)
	case "/tools", "/t":
		return m.handleToolsCommand(args)
	case "/test":
		return m.handleTestCommand(args)
	case "/sessions":
		return m.handleSessionsCommand(args)
	case "/session":
		return m.handleSessionCommand(args)
	case "/clear", "/c":
		m.messages = []Message{}
		m.updateMessagesView()
		return "Chat cleared"
	case "/exit", "/quit", "/q":
		// Set a flag to trigger quit in the main update loop
		m.shouldQuit = true
		return "Exiting chat..."
	default:
		return fmt.Sprintf("Unknown command: %s", cmd)
	}
}

// handleAgentMention processes @agent mentions
func (m ChatModel) handleAgentMention(input string) string {
	parts := strings.SplitN(input, " ", 2)
	if len(parts) < 2 {
		return "Usage: @agent_id message"
	}

	agentID := strings.TrimPrefix(parts[0], "@")
	message := parts[1]

	// Check for special @all mention
	if agentID == "all" {
		go m.streamAgentConversation("all", message)
		return ""
	}

	// Validate agent exists
	ctx := context.Background()
	resp, err := m.grpcClient.ListAvailableAgents(ctx, &pb.ListAgentsRequest{})
	if err != nil {
		return fmt.Sprintf("Failed to list agents: %v", err)
	}

	agentFound := false
	for _, agent := range resp.Agents {
		if agent.Id == agentID {
			agentFound = true
			break
		}
	}

	if !agentFound {
		return fmt.Sprintf("Unknown agent: @%s", agentID)
	}

	// Send message to specific agent
	go m.streamAgentConversation(agentID, message)
	return ""
}

// streamAgentConversation handles streaming conversation with an agent
func (m *ChatModel) streamAgentConversation(agentID, message string) {
	ctx := context.Background()

	// Create stream
	stream, err := m.grpcClient.StreamAgentConversation(ctx)
	if err != nil {
		log.Printf("Failed to create stream: %v", err)
		m.addErrorMessage(fmt.Sprintf("Failed to create stream: %v", err))
		return
	}
	defer stream.CloseSend()

	// Send initial request
	req := &pb.AgentStreamRequest{
		Request: &pb.AgentStreamRequest_Message{
			Message: &pb.AgentMessageRequest{
				AgentId:    agentID,
				Message:    message,
				SessionId:  m.sessionID,
				CampaignId: m.campaignID,
			},
		},
	}

	if err := stream.Send(req); err != nil {
		log.Printf("Failed to send request: %v", err)
		m.addErrorMessage(fmt.Sprintf("Failed to send request: %v", err))
		return
	}

	// Start thinking animation
	if m.agentIndicators != nil {
		m.agentIndicators.SetThinkingAnimation(agentID)
	}

	// Receive responses
	var currentContent strings.Builder
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("Stream error: %v", err)
				m.addErrorMessage(fmt.Sprintf("Stream error: %v", err))
			}
			break
		}

		switch response := resp.Response.(type) {
		case *pb.AgentStreamResponse_Fragment:
			// Handle message fragment
			fragment := response.Fragment
			if fragment != nil {
				// Accumulate content
				currentContent.WriteString(fragment.Content)

				// Update agent status
				if m.agentStatusTracker != nil {
					status := m.agentStatusTracker.GetAgentStatus(fragment.AgentId)
					if status != nil {
						status.State = AgentWorking
						status.CurrentTask = "Generating response..."
						m.agentStatusTracker.UpdateAgentStatus(fragment.AgentId, status)
					}
				}
			}

		case *pb.AgentStreamResponse_Status:
			// Handle status update
			agentStatus := response.Status
			if agentStatus != nil && m.agentStatusTracker != nil {
				// Convert proto status to internal status
				// This needs proper implementation based on the AgentStatus structure
			}

		case *pb.AgentStreamResponse_Event:
			// Handle stream event
			event := response.Event
			if event != nil {
				// Handle different event types based on StreamEvent structure
				// This needs implementation based on the actual event type
			}

		default:
			// Unknown response type
		}
	}
}

// handlePromptCommand handles /prompt subcommands
func (m ChatModel) handlePromptCommand(args []string) string {
	if len(args) == 0 {
		return `Prompt commands:
  /prompt list              - List all prompt layers
  /prompt get <layer>       - Get specific layer content
  /prompt set <layer> <text> - Set layer content
  /prompt delete <layer>    - Delete a layer`
	}

	ctx := context.Background()
	subCmd := args[0]

	switch subCmd {
	case "list":
		resp, err := m.promptsClient.ListPromptLayers(ctx, &promptspb.ListPromptLayersRequest{
			SessionId: m.sessionID,
		})
		if err != nil {
			return fmt.Sprintf("Failed to list prompts: %v", err)
		}

		if len(resp.Prompts) == 0 {
			return "No prompt layers configured"
		}

		var result strings.Builder
		result.WriteString("📜 Prompt Layers:\n")
		for _, prompt := range resp.Prompts {
			result.WriteString(fmt.Sprintf("  - %s (priority: %d)\n",
				prompt.Layer.String(), prompt.Priority))
		}
		return result.String()

	case "get":
		if len(args) < 2 {
			return "Usage: /prompt get <layer>"
		}
		// Implementation for getting a specific layer
		return fmt.Sprintf("Getting prompt layer: %s", args[1])

	case "set":
		if len(args) < 3 {
			return "Usage: /prompt set <layer> <content>"
		}
		// Implementation for setting a layer
		return fmt.Sprintf("Setting prompt layer: %s", args[1])

	case "delete":
		if len(args) < 2 {
			return "Usage: /prompt delete <layer>"
		}
		// Implementation for deleting a layer
		return fmt.Sprintf("Deleting prompt layer: %s", args[1])

	default:
		return fmt.Sprintf("Unknown prompt subcommand: %s", subCmd)
	}
}

// handleToolsCommand handles /tools subcommands
func (m ChatModel) handleToolsCommand(args []string) string {
	if len(args) == 0 {
		return m.getToolListText()
	}

	subCmd := args[0]
	switch subCmd {
	case "list":
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
		return m.searchToolsByCapability(args[1])
	case "status":
		return m.getActiveToolsStatus()
	default:
		return fmt.Sprintf("Unknown tools subcommand: %s", subCmd)
	}
}

// handleTestCommand handles test commands for visual features
func (m ChatModel) handleTestCommand(args []string) string {
	if len(args) == 0 {
		return m.getTestHelp()
	}

	testType := args[0]
	switch testType {
	case "markdown":
		return m.testMarkdownRendering()
	case "code":
		return m.testCodeHighlighting()
	case "mixed":
		return m.testMixedContent()
	case "agents":
		go m.testAgentAnimations()
		return "Testing agent animations..."
	case "completion":
		go m.testCompletionSystem()
		return "Testing completion system..."
	default:
		return fmt.Sprintf("Unknown test type: %s", testType)
	}
}

// Helper methods for adding messages
func (m *ChatModel) addMessage(msg Message) {
	m.messages = append(m.messages, msg)
	m.updateMessagesView()
}

func (m *ChatModel) addSystemMessage(content string) {
	m.addMessage(Message{
		Type:      msgSystem,
		Content:   content,
		Timestamp: time.Now(),
	})
}

func (m *ChatModel) addErrorMessage(content string) {
	m.addMessage(Message{
		Type:      msgError,
		Content:   content,
		Timestamp: time.Now(),
	})
}

func (m *ChatModel) addAgentMessage(agentID, content string) {
	m.addMessage(Message{
		Type:      msgAgent,
		Content:   content,
		AgentID:   agentID,
		Timestamp: time.Now(),
	})
}

func (m *ChatModel) addToolExecutionMessage(toolExec *pb.GuildToolExecution) {
	if toolExec == nil {
		return
	}

	msg := Message{
		Type:      msgToolStart,
		Content:   fmt.Sprintf("🔨 Executing tool: %s", toolExec.ToolName),
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"tool_id":   toolExec.ToolId,
			"tool_name": toolExec.ToolName,
		},
	}

	// Add parameters if any
	if len(toolExec.Parameters) > 0 {
		var params []string
		for k, v := range toolExec.Parameters {
			params = append(params, fmt.Sprintf("%s=%s", k, v))
		}
		msg.Content += fmt.Sprintf(" [%s]", strings.Join(params, ", "))
	}

	m.addMessage(msg)
}

// handleSessionsCommand handles /sessions command
func (m ChatModel) handleSessionsCommand(args []string) string {
	if m.sessionManager == nil {
		return "Session persistence is not available"
	}

	// For now, show current session info
	result := fmt.Sprintf("📚 Current Session:\n")
	if m.currentSession != nil {
		result += fmt.Sprintf("ID: %s\n", m.currentSession.ID)
		result += fmt.Sprintf("Name: %s\n", m.currentSession.Name)
		result += fmt.Sprintf("Created: %s\n", m.currentSession.CreatedAt.Format("2006-01-02 15:04"))
		if m.currentSession.CampaignID != nil {
			result += fmt.Sprintf("Campaign: %s\n", *m.currentSession.CampaignID)
		}
	} else {
		result += "No active session\n"
	}

	return result
}

// handleSessionCommand handles /session subcommands
func (m ChatModel) handleSessionCommand(args []string) string {
	if m.sessionManager == nil {
		return "Session persistence is not available"
	}

	if len(args) == 0 {
		return `Session commands:
  /session new [name]     - Create new session
  /session switch <id>    - Switch to different session
  /session rename <name>  - Rename current session
  /session export [format] - Export current session (json, markdown, html)`
	}

	subCmd := args[0]
	switch subCmd {
	case "new":
		var name string
		if len(args) > 1 {
			name = strings.Join(args[1:], " ")
		} else {
			name = fmt.Sprintf("Chat Session %s", time.Now().Format("2006-01-02 15:04"))
		}

		newSession, err := m.sessionManager.NewSession(name, m.currentSession.CampaignID)
		if err != nil {
			return fmt.Sprintf("Failed to create new session: %v", err)
		}

		// Note: We can't actually switch the session in the model from here
		// as we return a string, not a new model
		return fmt.Sprintf("Created new session: %s (ID: %s)\nNote: Restart chat to use the new session", newSession.Name, newSession.ID[:8])

	case "rename":
		if len(args) < 2 {
			return "Usage: /session rename <new name>"
		}

		newName := strings.Join(args[1:], " ")
		m.currentSession.Name = newName

		if err := m.sessionManager.SaveSession(m.currentSession); err != nil {
			return fmt.Sprintf("Failed to rename session: %v", err)
		}

		return fmt.Sprintf("Session renamed to: %s", newName)

	case "export":
		format := session.ExportFormatJSON
		if len(args) > 1 {
			switch args[1] {
			case "markdown", "md":
				format = session.ExportFormatMarkdown
			case "html":
				format = session.ExportFormatHTML
			case "json":
				format = session.ExportFormatJSON
			default:
				return fmt.Sprintf("Unknown export format: %s (use json, markdown, or html)", args[1])
			}
		}

		data, err := m.sessionManager.ExportSession(m.currentSession.ID, format)
		if err != nil {
			return fmt.Sprintf("Failed to export session: %v", err)
		}

		// Save to file
		filename := fmt.Sprintf("session_%s_%s.%s",
			m.currentSession.ID[:8],
			time.Now().Format("20060102_150405"),
			string(format))

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Sprintf("Failed to save export: %v", err)
		}

		return fmt.Sprintf("Session exported to: %s", filename)

	case "switch":
		if len(args) < 2 {
			return "Usage: /session switch <session-id>"
		}

		// Note: We can't actually switch sessions from here
		return "Session switching requires restarting chat with --session flag"

	default:
		return fmt.Sprintf("Unknown session subcommand: %s", subCmd)
	}
}
