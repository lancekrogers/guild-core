// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// InitState represents different stages of the initialization process
type InitState int

const (
	StateWelcome InitState = iota
	StateCampaignInput
	StateProjectInput
	StateConfirmation
	StateInitializing
	StateSetupWizard
	StateDemoQuestion
	StateDemoSelection
	StateValidating
	StateComplete
	StateError
)

// State update handlers

func (m *InitTUIModelV2) updateProjectInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	if k, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(k, keys.Enter):
			m.projectName = m.inputs["project"].Value()
			if m.projectName == "" {
				m.projectName = m.inputs["project"].Placeholder
			}
			m.state = StateConfirmation
			project := m.inputs["project"]
			project.Blur()
			m.inputs["project"] = project
			return m, nil
		case key.Matches(k, keys.Quit):
			return m, tea.Quit
		}
	}
	
	m.inputs["project"], cmd = m.inputs["project"].Update(msg)
	return m, cmd
}

func (m *InitTUIModelV2) updateConfirmation(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "y", "Y", "enter":
			m.state = StateInitializing
			return m, tea.Batch(
				m.spinner.Tick,
				m.doInitialization(),
			)
		case "n", "N":
			m.state = StateCampaignInput
			campaign := m.inputs["campaign"]
			campaign.SetValue(m.campaignName)
			cmd := campaign.Focus()
			m.inputs["campaign"] = campaign
			return m, cmd
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *InitTUIModelV2) updateInitializing(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initProgressMsg:
		m.progress.SetPercent(msg.percent)
		if msg.phase == "complete" {
			m.state = StateDemoQuestion
			return m, nil
		}
		return m, m.spinner.Tick
		
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
		
	case errMsg:
		m.err = msg.err
		m.state = StateError
		return m, nil
	}
	
	return m, nil
}

func (m *InitTUIModelV2) updateDemoQuestion(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "y", "Y":
			m.state = StateDemoSelection
			return m, nil
		case "n", "N", "enter":
			m.state = StateValidating
			return m, tea.Batch(
				m.spinner.Tick,
				m.doValidation(),
			)
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *InitTUIModelV2) updateDemoSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(k, keys.Up):
			if m.selectedDemo > 0 {
				m.selectedDemo--
			}
		case key.Matches(k, keys.Down):
			if m.selectedDemo < len(m.demoOptions)-1 {
				m.selectedDemo++
			}
		case key.Matches(k, keys.Enter):
			m.demoType = m.demoOptions[m.selectedDemo]
			m.state = StateValidating
			return m, tea.Batch(
				m.spinner.Tick,
				m.createDemoCommission(),
				m.doValidation(),
			)
		case key.Matches(k, keys.Quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *InitTUIModelV2) updateValidating(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case validationResultsMsg:
		m.validationResults = msg.results
		m.state = StateComplete
		return m, nil
		
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
		
	case errMsg:
		m.err = msg.err
		m.state = StateError
		return m, nil
		
	case warnMsg:
		// Log warning but continue
		return m, nil
	}
	
	return m, nil
}

func (m *InitTUIModelV2) updateComplete(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.Type == tea.KeyEnter || key.Matches(k, keys.Quit) {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *InitTUIModelV2) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.Type == tea.KeyEnter || key.Matches(k, keys.Quit) {
			return m, tea.Quit
		}
	}
	return m, nil
}

// Render methods for each state

func (m *InitTUIModelV2) renderCampaignInput() string {
	title := m.styles.RenderHeader(
		"Campaign Configuration",
		"Choose a name for your development campaign",
	)
	
	help := `A campaign organizes related projects and guilds.
Think of it as your overarching development initiative.`
	
	helpRendered, _ := m.renderer.Render(help)
	
	inputBox := m.styles.InputBox.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.Label.Render("Campaign Name"),
			m.inputs["campaign"].View(),
		),
	)
	
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		m.styles.Section.Render(helpRendered),
		inputBox,
		m.renderHelp(),
	)
}

func (m *InitTUIModelV2) renderProjectInput() string {
	title := m.styles.RenderHeader(
		"Project Configuration",
		"Name your project within the campaign",
	)
	
	help := `A project is a specific codebase or application.
Multiple projects can exist within a single campaign.`
	
	helpRendered, _ := m.renderer.Render(help)
	
	inputBox := m.styles.InputBox.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.Label.Render("Project Name"),
			m.inputs["project"].View(),
		),
	)
	
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		m.styles.Section.Render(helpRendered),
		inputBox,
		m.renderHelp(),
	)
}

func (m *InitTUIModelV2) renderConfirmation() string {
	title := m.styles.RenderHeader(
		"Confirm Your Settings",
		"Review your guild configuration",
	)
	
	settings := m.styles.Section.Border(m.styles.BorderStyle).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.RenderLabelValue("Campaign", m.campaignName),
			m.styles.RenderLabelValue("Project", m.projectName),
			m.styles.RenderLabelValue("Location", m.config.ProjectPath),
		),
	)
	
	prompt := m.styles.Info.Render("Continue with these settings? (Y/n)")
	
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		settings,
		"",
		prompt,
	)
}

func (m *InitTUIModelV2) renderInitializing() string {
	title := m.styles.RenderHeader(
		"Forging Your Guild",
		"Setting up your development environment",
	)
	
	status := lipgloss.JoinVertical(
		lipgloss.Center,
		m.spinner.View()+" Initializing project structure...",
		"",
		m.progress.View(),
	)
	
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		m.styles.Section.Render(status),
	)
}

func (m *InitTUIModelV2) renderDemoQuestion() string {
	title := m.styles.RenderHeader(
		"Demo Commission",
		"Start with a sample quest?",
	)
	
	content := `Would you like to create a demo commission?

This will give you a ready-to-use example showcasing
Guild's capabilities with your chosen technologies.

Create demo commission? (Y/n)`
	
	rendered, _ := m.renderer.Render(content)
	
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		m.styles.Section.Render(rendered),
	)
}

func (m *InitTUIModelV2) renderDemoSelection() string {
	// Use the demo renderer if available
	demos := GetDemoInfo()
	renderer, err := NewDemoRenderer(m.width-4, m.styles)
	if err == nil {
		return renderer.RenderDemoSelection(demos, m.selectedDemo)
	}
	
	// Fallback to simple rendering
	title := m.styles.RenderHeader(
		"Choose Your Quest",
		"Select a demo commission",
	)
	
	var items []string
	for i, demo := range demos {
		prefix := "  "
		style := m.styles.DemoItem
		if i == m.selectedDemo {
			prefix = "▶ "
			style = m.styles.DemoItemSelected
		}
		items = append(items, style.Render(fmt.Sprintf("%s%s", prefix, demo.Title)))
	}
	
	list := lipgloss.JoinVertical(lipgloss.Left, items...)
	
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		m.styles.Section.Render(list),
		m.renderHelp(),
	)
}

func (m *InitTUIModelV2) renderValidating() string {
	title := m.styles.RenderHeader(
		"Validating Setup",
		"Ensuring everything is properly configured",
	)
	
	status := lipgloss.JoinVertical(
		lipgloss.Center,
		m.spinner.View()+" Running validation checks...",
	)
	
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		m.styles.Section.Render(status),
	)
}

func (m *InitTUIModelV2) renderComplete() string {
	// Use demo renderer for rich results
	renderer, _ := NewDemoRenderer(m.width-4, m.styles)
	
	title := m.styles.RenderHeader(
		"Guild Successfully Established!",
		"Your development environment is ready",
	)
	
	// Summary
	summary := m.styles.Section.Border(m.styles.BorderStyle).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.styles.RenderSuccess("Campaign: "+m.campaignName),
			m.styles.RenderSuccess("Project: "+m.projectName),
			m.styles.RenderSuccess("Location: "+m.config.ProjectPath),
			m.styles.RenderSuccess("Database: Initialized"),
			m.styles.RenderSuccess("Daemon: Ready to start"),
		),
	)
	
	// Validation results (if any)
	var validation string
	if len(m.validationResults) > 0 && renderer != nil {
		validation = renderer.RenderValidationResults(m.validationResults)
	}
	
	// Next steps
	nextSteps := m.styles.Section.Render(`🚀 **Try these commands:**

    guild chat              # Start chatting with your agents
    guild status            # Check system status  
    guild commission list   # See your commissions

Ready to start? Run: **guild chat**`)
	
	nextStepsRendered, _ := m.renderer.Render(nextSteps)
	
	sections := []string{title, summary}
	if validation != "" {
		sections = append(sections, validation)
	}
	sections = append(sections, nextStepsRendered)
	sections = append(sections, m.styles.Info.Render("Press Enter to exit..."))
	
	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}

func (m *InitTUIModelV2) renderError() string {
	title := m.styles.RenderHeader(
		"Initialization Failed",
		"An error occurred during setup",
	)
	
	errMsg := m.styles.RenderError(m.err.Error())
	
	// Try to provide helpful context
	var help string
	if gerror.Is(m.err, gerror.ErrCodeCancelled) {
		help = "The operation was cancelled. You can run 'guild init' again to retry."
	} else if gerror.Is(m.err, gerror.ErrCodeStorage) {
		help = "There was a file system issue. Check permissions and disk space."
	} else if gerror.Is(m.err, gerror.ErrCodeValidation) {
		help = "Configuration validation failed. Check your settings and try again."
	}
	
	sections := []string{title, errMsg}
	if help != "" {
		sections = append(sections, "", m.styles.Info.Render(help))
	}
	sections = append(sections, "", m.styles.Info.Render("Press Enter to exit..."))
	
	return lipgloss.JoinVertical(lipgloss.Center, sections...)
}