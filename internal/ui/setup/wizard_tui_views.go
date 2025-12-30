// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderWelcome renders the welcome screen
func (m *WizardTUIModel) renderWelcome() string {
	title := headerStyle.Render("🏰 Welcome to Guild Framework!")

	content := lipgloss.JoinVertical(lipgloss.Left,
		"Welcome to the Guild Framework setup wizard!",
		"",
		"This wizard will help you:",
		"• Detect available AI providers",
		"• Configure API credentials",
		"• Select optimal models for your needs",
		"• Create your team of specialized AI agents",
		"",
		warningStyle.Render("Setup time: ~2 minutes"),
		"",
		"Press Enter to begin your guild setup journey...",
	)

	box := boxStyle.Render(content)

	help := m.help.View(m.keys)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		box,
		help,
	)
}

// renderProviderDetection renders the provider detection screen
func (m *WizardTUIModel) renderProviderDetection() string {
	title := headerStyle.Render("🔍 Detecting AI Providers...")

	content := lipgloss.JoinVertical(lipgloss.Left,
		"Scanning your system for available AI providers...",
		"",
		"• Checking for API keys in environment variables",
		"• Detecting local AI services (Ollama, etc.)",
		"• Validating provider configurations",
		"",
		"This may take a few moments...",
	)

	// Show a simple spinner while detecting
	spinner := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	frame := len(m.detectedProviders) % len(spinner)
	spinnerChar := string([]rune(spinner)[frame])

	statusLine := fmt.Sprintf("%s Detecting providers...", spinnerChar)

	box := boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, content, "", statusLine))

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		box,
	)
}

// renderProviderSelection renders the provider selection screen
func (m *WizardTUIModel) renderProviderSelection() string {
	title := headerStyle.Render("📚 Select AI Providers")

	var content strings.Builder
	content.WriteString("Select the AI providers you want to configure:\n\n")

	if len(m.detectedProviders) == 0 {
		content.WriteString(warningStyle.Render("No providers detected. You may need to:"))
		content.WriteString("\n• Set API keys in environment variables")
		content.WriteString("\n• Install local services like Ollama")
		content.WriteString("\n• Check your network connection")

		box := boxStyle.Render(content.String())
		help := m.help.View(m.keys)

		return lipgloss.JoinVertical(lipgloss.Left, title, box, help)
	}

	// Show the list of providers
	listView := m.list.View()

	instructions := lipgloss.JoinVertical(lipgloss.Left,
		"Instructions:",
		"• Use ↑/↓ to navigate",
		"• Press Space to toggle selection",
		"• Press Enter to continue with selected providers",
		"• Providers with ✅ have valid credentials",
		"• Providers with 🏠 run locally on your machine",
	)

	instructionBox := boxStyle.Render(instructions)

	help := m.help.View(m.keys)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		listView,
		instructionBox,
		help,
	)
}

// renderModelSelection renders the model selection screen for the current provider
func (m *WizardTUIModel) renderModelSelection() string {
	if m.currentProvider == nil {
		return "Error: No current provider"
	}

	title := headerStyle.Render(fmt.Sprintf("💰 Select Models for %s", m.currentProvider.Name))

	// This would show model selection list similar to provider selection
	content := fmt.Sprintf("Configuring models for %s...\n\n", m.currentProvider.Name)
	content += "Available models:\n"

	// In real implementation, this would show actual model list
	content += "• Recommended models are marked with ⭐\n"
	content += "• Cost information shown per 1,000 tokens\n"
	content += "• Select models that fit your use case\n"

	box := boxStyle.Render(content)
	help := m.help.View(m.keys)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		box,
		help,
	)
}

// renderAgentCreation renders the agent creation screen
func (m *WizardTUIModel) renderAgentCreation() string {
	title := headerStyle.Render("🤖 Creating Your Guild of Agents")

	content := lipgloss.JoinVertical(lipgloss.Left,
		"Setting up your team of specialized AI agents...",
		"",
		"Agent Presets Available:",
		"• Demo: Minimal Setup - Perfect for 30-second demos",
		"• Development Team - Full-stack development focus",
		"• Content Creation - Writing and media focus",
		"• Data Analysis - Research and analytics focus",
		"",
		"Creating agents based on your selected providers...",
	)

	box := boxStyle.Render(content)
	help := m.help.View(m.keys)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		box,
		help,
	)
}

// renderProgress renders the progress screen
func (m *WizardTUIModel) renderProgress() string {
	title := headerStyle.Render("⚙️ Configuring Your Guild")

	progressView := m.progressBar.ViewAs(float64(m.progressCurrent) / float64(m.progressMax))

	content := lipgloss.JoinVertical(lipgloss.Left,
		m.statusMessage,
		"",
		progressView,
		"",
		fmt.Sprintf("Step %d of %d", m.progressCurrent, m.progressMax),
	)

	box := boxStyle.Render(content)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		box,
	)
}

// renderCompletion renders the completion screen
func (m *WizardTUIModel) renderCompletion() string {
	title := headerStyle.Render("🎉 Guild Setup Complete!")

	var content strings.Builder
	content.WriteString(successStyle.Render("Your guild has been successfully established!"))
	content.WriteString("\n\n")

	// Provider summary
	content.WriteString(fmt.Sprintf("Providers Configured: %d\n", len(m.configuredProviders)))
	for _, provider := range m.configuredProviders {
		content.WriteString(fmt.Sprintf("  • %s (%d models)\n", provider.Name, len(provider.Models)))
	}
	content.WriteString("\n")

	// Agent summary
	content.WriteString(fmt.Sprintf("Agents Created: %d\n", len(m.createdAgents)))
	for _, agent := range m.createdAgents {
		content.WriteString(fmt.Sprintf("  • %s (%s)\n", agent.Name, agent.Type))
	}
	content.WriteString("\n")

	// Next steps
	content.WriteString("Next Steps:\n")
	content.WriteString("  1. Run 'guild chat' to start coordinating agents\n")
	content.WriteString("  2. Use 'guild commission create' for new objectives\n")
	content.WriteString("  3. View progress with 'guild kanban view'\n")
	content.WriteString("\n")
	content.WriteString("Press Enter to exit the wizard...")

	box := boxStyle.Render(content.String())

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		box,
	)
}

// renderError renders the error screen
func (m *WizardTUIModel) renderError() string {
	title := headerStyle.Render("❌ Setup Error")

	content := lipgloss.JoinVertical(lipgloss.Left,
		errorStyle.Render("An error occurred during setup:"),
		"",
		m.errorMessage,
		"",
		"Press 'q' to quit or 'esc' to go back and try again.",
	)

	box := boxStyle.Render(content)
	help := m.help.View(m.keys)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		box,
		help,
	)
}

// renderStepIndicator renders a visual step indicator
func (m *WizardTUIModel) renderStepIndicator() string {
	steps := []string{"Welcome", "Detection", "Selection", "Models", "Agents", "Complete"}

	var indicators []string
	for i, step := range steps {
		if i < m.currentStep {
			indicators = append(indicators, successStyle.Render("✓ "+step))
		} else if i == m.currentStep {
			indicators = append(indicators, warningStyle.Render("▶ "+step))
		} else {
			indicators = append(indicators, "○ "+step)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, indicators...)
}
