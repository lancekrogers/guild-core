package chat

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/guild-ventures/guild-core/pkg/config"
	pb "github.com/guild-ventures/guild-core/pkg/grpc/pb/guild/v1"
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
	
	help := helpStyle.Render("🏰 Guild Chat Commands:\n\n")
	
	commands := []struct {
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
		{"/exit, /quit, /q", "Exit chat"},
		{"@agent message", "Send message to specific agent"},
		{"@all message", "Broadcast to all agents"},
		{"ctrl+h", "Show help"},
		{"ctrl+p", "Toggle prompt management"},
		{"ctrl+a", "Toggle agent status"},
		{"ctrl+g", "Toggle global stream"},
		{"tab", "Auto-complete commands"},
		{"↑/↓", "Navigate history"},
	}
	
	for _, cmd := range commands {
		help += fmt.Sprintf("%s - %s\n", 
			commandStyle.Render(cmd.cmd), 
			helpStyle.Render(cmd.desc))
	}
	
	return help
}

// getStatusText returns current guild status
func (m ChatModel) getStatusText() string {
	status := fmt.Sprintf("📊 Guild Status\n")
	status += fmt.Sprintf("Campaign: %s\n", m.campaignID)
	status += fmt.Sprintf("Session: %s\n", m.sessionID)
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
		ID string
		Name string
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