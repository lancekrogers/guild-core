// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model
func (m ChatModel) View() string {
	if !m.ready {
		return "Initializing Guild Chat..."
	}

	// Check for integration feature flags
	if m.integrationFlags != nil && m.integrationFlags["enhanced_view"] {
		return m.RenderIntegratedView()
	}

	var s strings.Builder

	// Show different content based on view mode
	switch m.viewMode {
	case chatModePrompt:
		s.WriteString(m.getPromptManagementView())
	case chatModeStatus:
		s.WriteString(m.getAgentStatusView())
	case chatModeGlobal:
		s.WriteString(m.getGlobalStreamView())
	case chatModeFuzzyFinder:
		if m.fuzzyFinder != nil {
			return m.fuzzyFinder.View()
		}
		s.WriteString("Initializing fuzzy finder...")
	case chatModeGlobalSearch:
		if m.globalSearch != nil {
			return m.globalSearch.View()
		}
		s.WriteString("Initializing global search...")
	default:
		// Normal chat view
		s.WriteString(m.viewport.View())
		s.WriteString("\n")

		// Show completion suggestions if available
		if m.showingCompletion && len(m.completionResults) > 0 {
			s.WriteString(m.renderCompletionSuggestions())
			s.WriteString("\n")
		}

		// Show command palette if open
		if m.commandPalette != nil && m.commandPalette.IsOpen() {
			s.WriteString(m.commandPalette.View())
			s.WriteString("\n")
		}

		// Input area with campaign and vim mode indicators
		inputLabel := fmt.Sprintf("📜 %s", m.getCampaignDisplay())

		// Add vim mode indicator if enabled
		if m.vimModeEnabled && m.vimState != nil {
			vimIndicator := m.vimState.GetModeIndicator()
			inputLabel += " " + vimIndicator
		}

		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Render(inputLabel))
		s.WriteString("\n")
		s.WriteString(m.input.View())
	}

	// Help line
	if m.err != nil {
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(fmt.Sprintf("Error: %v", m.err)))
	} else {
		s.WriteString("\n")
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		// Build dynamic help text based on platform
		helpText := fmt.Sprintf("%s: send • %s: newline • %s: quit • %s: help • %s: vim mode • %s/%s: copy/paste",
			m.keys.Submit.Help().Key,
			m.keys.NewLine.Help().Key,
			m.keys.Quit.Help().Key,
			m.keys.Help.Help().Key,
			m.keys.ToggleVimMode.Help().Key,
			m.keys.Copy.Help().Key,
			m.keys.Paste.Help().Key)
		s.WriteString(helpStyle.Render(helpText))
	}

	return s.String()
}

// updateMessagesView updates the viewport with formatted messages
func (m ChatModel) updateMessagesView() {
	var content strings.Builder

	for i, msg := range m.messages {
		if i > 0 {
			content.WriteString("\n\n")
		}

		// Use rich content formatter if available, otherwise fall back to safe formatting
		var formattedContent string
		if m.contentFormatter != nil {
			// Convert message type to string for content formatter
			msgTypeStr := ""
			switch msg.Type {
			case msgAgent:
				msgTypeStr = "agent"
			case msgUser:
				msgTypeStr = "user"
			case msgSystem:
				msgTypeStr = "system"
			case msgError:
				msgTypeStr = "error"
			case msgAgentThinking:
				msgTypeStr = "thinking"
			case msgAgentWorking:
				msgTypeStr = "working"
			case msgToolComplete:
				msgTypeStr = "tool"
			}

			// Use the rich content formatter
			if msgTypeStr == "agent" {
				formattedContent = m.contentFormatter.FormatAgentResponse(msg.Content, msg.AgentID)
			} else if msgTypeStr == "user" {
				formattedContent = m.contentFormatter.FormatUserMessage(msg.Content)
			} else if msgTypeStr == "system" {
				formattedContent = m.contentFormatter.FormatSystemMessage(msg.Content)
			} else if msgTypeStr == "error" {
				formattedContent = m.contentFormatter.FormatErrorMessage(msg.Content)
			} else {
				// For thinking, working, tool output, use agent response formatter
				formattedContent = m.contentFormatter.FormatAgentResponse(msg.Content, msg.AgentID)
			}
		} else {
			// Fall back to safe formatting if content formatter not available
			formattedContent = m.safeFormatContent(msg.Type, msg.Content, msg.AgentID)
		}

		content.WriteString(formattedContent)
	}

	m.viewport.SetContent(content.String())

	// Auto-scroll to bottom for new messages
	m.viewport.GotoBottom()
}

// getPromptManagementView returns the prompt management interface
func (m ChatModel) getPromptManagementView() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("33")).
		MarginBottom(1)

	s.WriteString(titleStyle.Render("📜 Layered Prompt Management"))
	s.WriteString("\n\n")

	// Show current prompt layers
	s.WriteString("Active Layers:\n")
	for i, layer := range m.promptLayers {
		s.WriteString(fmt.Sprintf("  %d. %s\n", i+1, layer))
	}

	if len(m.promptLayers) == 0 {
		s.WriteString("  No custom layers configured\n")
	}

	s.WriteString("\n")
	s.WriteString("Commands:\n")
	s.WriteString("  /prompt list              - List all available prompts\n")
	s.WriteString("  /prompt get <layer>       - Get specific layer content\n")
	s.WriteString("  /prompt set <layer> <text> - Set layer content\n")
	s.WriteString("  /prompt delete <layer>    - Remove a layer\n")
	s.WriteString("\n")
	s.WriteString("Press Ctrl+P to return to chat")

	return s.String()
}

// getAgentStatusView returns the agent status monitoring interface
func (m ChatModel) getAgentStatusView() string {
	if m.statusDisplay != nil {
		return m.statusDisplay.RenderStatusPanel()
	}

	// Fallback if status display not initialized
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("141")).
		MarginBottom(1)

	s.WriteString(titleStyle.Render("🏰 Guild Agent Status"))
	s.WriteString("\n\n")

	// Mock agent status for now
	s.WriteString("Active Agents:\n")
	s.WriteString("  🟢 @manager - Guild Master (idle)\n")
	s.WriteString("  🟡 @coder - Code Artisan (working on task_123)\n")
	s.WriteString("  🔴 @reviewer - Review Artisan (blocked)\n")
	s.WriteString("\n")
	s.WriteString("Active Tools: 2\n")
	s.WriteString("Total Cost: $0.042\n")
	s.WriteString("\n")
	s.WriteString("Press Ctrl+A to return to chat")

	return s.String()
}

// getGlobalStreamView returns the global activity stream view
func (m ChatModel) getGlobalStreamView() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	s.WriteString(titleStyle.Render("🌐 Global Activity Stream"))
	s.WriteString("\n\n")

	// Show recent activity from all agents
	s.WriteString("Recent Activity:\n")
	s.WriteString("  [14:23:01] @manager: Planning task decomposition...\n")
	s.WriteString("  [14:23:05] @coder: Started implementing feature X\n")
	s.WriteString("  [14:23:12] @reviewer: Reviewing pull request #42\n")
	s.WriteString("  [14:23:15] Tool 'file-reader' executed by @coder\n")
	s.WriteString("  [14:23:18] @tester: Running test suite...\n")
	s.WriteString("\n")
	s.WriteString("Press Ctrl+G to return to chat")

	return s.String()
}

// getCampaignDisplay returns the campaign name for display
func (m ChatModel) getCampaignDisplay() string {
	var display string

	if m.campaignID == "" {
		display = "Guild Chat"
	} else {
		display = fmt.Sprintf("Campaign: %s", m.campaignID)
	}

	// Add selected guild if available
	if m.selectedGuild != "" {
		display += fmt.Sprintf(" | Guild: %s", m.selectedGuild)
	}

	// Add session info if available
	if m.currentSession != nil {
		display += fmt.Sprintf(" | Session: %s", m.currentSession.Name)
	}

	return display
}

// renderCompletionSuggestions renders the completion popup with medieval theming
func (m ChatModel) renderCompletionSuggestions() string {
	if len(m.completionResults) == 0 {
		return ""
	}

	// Create a bordered box for completions with medieval styling
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")). // Medieval gold
		Padding(0, 1).
		MaxWidth(60)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("220")). // Bright gold
		MarginBottom(1)

	// Add decorative elements
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Render("─────────────────────────")

	var items []string
	items = append(items, titleStyle.Render("⚔️ Guild Suggestions"))
	items = append(items, separator)

	// Render completion items with type-specific icons and medieval styling
	for i, result := range m.completionResults {
		icon := m.getCompletionIcon(result.Metadata["type"])

		// Define styles for selected vs unselected items
		var nameStyle, descStyle lipgloss.Style

		if i == m.completionIndex {
			// Selected item - highlight with medieval purple theme
			nameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("230")). // Light yellow
				Background(lipgloss.Color("63"))   // Purple background
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")). // Light yellow
				Background(lipgloss.Color("63"))   // Purple background
		} else {
			// Unselected items
			nameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")) // Gold
			descStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")) // Dark gray
		}

		// Format item with proper spacing and medieval icons
		name := nameStyle.Render(result.Content)
		description := descStyle.Render(result.Metadata["description"])

		// Create the full item line
		var itemLine string
		if i == m.completionIndex {
			// Selected item gets special formatting
			itemLine = fmt.Sprintf(" %s %s", icon, name)
			if description != "" {
				itemLine += "\n   " + description
			}
		} else {
			itemLine = fmt.Sprintf(" %s %s", icon, name)
			if description != "" {
				itemLine += " - " + description
			}
		}

		items = append(items, itemLine)
	}

	// Add navigation hint with medieval styling
	items = append(items, separator)
	hintStyle := lipgloss.NewStyle().
		Italic(true).
		Foreground(lipgloss.Color("241"))
	items = append(items, hintStyle.Render("↹ Navigate • ↵ Select • ⎋ Cancel"))

	return boxStyle.Render(strings.Join(items, "\n"))
}

// getCompletionIcon returns an icon based on completion type
func (m ChatModel) getCompletionIcon(completionType string) string {
	switch completionType {
	case "command":
		return "⚔️" // Sword for commands
	case "agent":
		return "🛡️" // Shield for agents
	case "file":
		return "📜" // Scroll for files
	case "task":
		return "🎯" // Target for tasks
	case "argument":
		return "🗝️" // Key for arguments
	case "history":
		return "📚" // Books for history
	case "suggestion":
		return "✨" // Sparkles for suggestions
	default:
		return "•" // Default bullet
	}
}

// RenderIntegratedView renders the chat view with all visual enhancements
func (m ChatModel) RenderIntegratedView() string {
	var s strings.Builder

	// Use enhanced markdown rendering if available
	if m.markdownRenderer != nil {
		// Render with markdown support
		var content strings.Builder
		for i, msg := range m.messages {
			if i > 0 {
				content.WriteString("\n\n")
			}

			// Use content formatter for rich rendering
			if m.contentFormatter != nil {
				switch msg.Type {
				case msgAgent:
					content.WriteString(m.contentFormatter.FormatAgentResponse(msg.Content, msg.AgentID))
				case msgSystem:
					content.WriteString(m.contentFormatter.FormatSystemMessage(msg.Content))
				case msgError:
					content.WriteString(m.contentFormatter.FormatErrorMessage(msg.Content))
				default:
					content.WriteString(m.contentFormatter.FormatUserMessage(msg.Content))
				}
			} else {
				content.WriteString(m.safeFormatContent(msg.Type, msg.Content, msg.AgentID))
			}
		}
		m.viewport.SetContent(content.String())
	}

	// Status bar with agent indicators
	if m.agentIndicators != nil && m.viewMode == chatModeStatus {
		s.WriteString(m.renderAgentActivityBar())
		s.WriteString("\n")
	}

	// Main viewport
	s.WriteString(m.viewport.View())
	s.WriteString("\n")

	// Completion suggestions with enhanced styling
	if m.showingCompletion && len(m.completionResults) > 0 {
		s.WriteString(m.renderEnhancedCompletions())
		s.WriteString("\n")
	}

	// Enhanced input area
	s.WriteString(m.renderEnhancedInputArea())

	return s.String()
}

// renderAgentActivityBar renders a status bar showing agent activity
func (m ChatModel) renderAgentActivityBar() string {
	if m.agentIndicators == nil {
		return ""
	}

	var agents []string

	// Get active animations
	animations := m.agentIndicators.GetActiveAnimations()

	for agentID := range animations {
		indicator := m.agentIndicators.GetCurrentIndicator(agentID)
		context := m.agentIndicators.GetAnimationContext(agentID)

		agentStr := fmt.Sprintf("%s %s", indicator, agentID)
		if context != "" && context != "thinking" && context != "working" {
			agentStr += fmt.Sprintf(" (%s)", context)
		}

		agents = append(agents, agentStr)
	}

	if len(agents) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("No active agents")
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true).
		Render("Active: " + strings.Join(agents, " | "))
}

// renderEnhancedCompletions renders completions with visual enhancements
func (m ChatModel) renderEnhancedCompletions() string {
	// Reuse the existing completion rendering with potential enhancements
	return m.renderCompletionSuggestions()
}

// renderEnhancedInputArea renders the input area with visual indicators
func (m ChatModel) renderEnhancedInputArea() string {
	var s strings.Builder

	// Campaign/context indicator with styling
	campaignStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true)

	contextLine := fmt.Sprintf("📜 %s", m.getCampaignDisplay())

	// Add cost indicator if tracking
	if m.costConsent != nil && len(m.costConsent) > 0 {
		contextLine += " | 💰 Cost tracking enabled"
	}

	s.WriteString(campaignStyle.Render(contextLine))
	s.WriteString("\n")
	s.WriteString(m.input.View())

	// Enhanced help line
	s.WriteString("\n")
	if m.err != nil {
		s.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Render(fmt.Sprintf("❌ Error: %v", m.err)))
	} else {
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

		helpText := "^C quit • ^H help • ^P prompts • ^A agents"
		if m.showingCompletion {
			helpText += " • ↹ complete"
		}

		s.WriteString(helpStyle.Render(helpText))
	}

	return s.String()
}
