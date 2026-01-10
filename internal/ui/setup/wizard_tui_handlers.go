// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/lancekrogers/guild-core/internal/setup"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// handleKeyPress handles key press events based on current state
func (m *WizardTUIModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.state == stateCompletion || m.state == stateError {
			return m, tea.Quit
		}
		// Show confirmation dialog in other states
		return m, tea.Quit

	case "?":
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}

	switch m.state {
	case stateWelcome:
		return m.handleWelcomeKeys(msg)
	case stateProviderSelection:
		return m.handleProviderSelectionKeys(msg)
	case stateModelSelection:
		return m.handleModelSelectionKeys(msg)
	case stateAgentCreation:
		return m.handleAgentCreationKeys(msg)
	case stateCompletion:
		return m.handleCompletionKeys(msg)
	case stateError:
		return m.handleErrorKeys(msg)
	default:
		return m, nil
	}
}

// handleWelcomeKeys handles key presses on the welcome screen
func (m *WizardTUIModel) handleWelcomeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.state = stateProviderDetection
		m.currentStep = 1
		return m, m.detectProviders()
	}
	return m, nil
}

// handleProviderSelectionKeys handles key presses on the provider selection screen
func (m *WizardTUIModel) handleProviderSelectionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Get selected providers and move to model selection
		selected := m.getSelectedProviders()
		if len(selected) == 0 {
			// No providers selected, show error
			m.statusMessage = "Please select at least one provider"
			return m, nil
		}
		m.selectedProviders = selected
		m.currentProviderIndex = 0
		m.currentProvider = &m.selectedProviders[0]
		m.state = stateModelSelection
		m.currentStep = 3
		return m, nil

	case "space":
		// Toggle provider selection
		item, ok := m.list.SelectedItem().(ProviderItem)
		if ok {
			item.Selected = !item.Selected
			// Update the item in the list
			m.list.SetItem(m.list.Index(), item)
		}
		return m, nil

	case "esc":
		// Go back to welcome
		m.state = stateWelcome
		m.currentStep = 0
		return m, nil
	}

	// Let the list handle other keys
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// handleModelSelectionKeys handles key presses on the model selection screen
func (m *WizardTUIModel) handleModelSelectionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Configure current provider and move to next
		return m, m.configureCurrentProvider()

	case "esc":
		// Go back to provider selection
		m.state = stateProviderSelection
		m.currentStep = 2
		return m, nil
	}
	return m, nil
}

// handleAgentCreationKeys handles key presses on the agent creation screen
func (m *WizardTUIModel) handleAgentCreationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Create agents and move to completion
		return m, m.createAgents()

	case "esc":
		// Go back to model selection
		if len(m.selectedProviders) > 0 {
			m.state = stateModelSelection
			m.currentStep = 3
		} else {
			m.state = stateProviderSelection
			m.currentStep = 2
		}
		return m, nil
	}
	return m, nil
}

// handleCompletionKeys handles key presses on the completion screen
func (m *WizardTUIModel) handleCompletionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m, tea.Quit
	}
	return m, nil
}

// handleErrorKeys handles key presses on the error screen
func (m *WizardTUIModel) handleErrorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Go back to previous state
		if m.currentStep > 0 {
			m.currentStep--
		}
		m.state = WizardState(m.currentStep)
		m.errorMessage = ""
		return m, nil
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

// Command generators

// detectProviders starts provider detection
func (m *WizardTUIModel) detectProviders() tea.Cmd {
	return func() tea.Msg {
		providers, err := m.wizard.DetectProviders(m.ctx)
		return providerDetectedMsg{
			providers: providers,
			err:       err,
		}
	}
}

// updateProviderList updates the list with detected providers
func (m *WizardTUIModel) updateProviderList() tea.Cmd {
	items := make([]list.Item, len(m.detectedProviders))
	for i, provider := range m.detectedProviders {
		items[i] = ProviderItem{
			DetectedProvider: provider,
			Selected:         false,
		}
	}
	return m.list.SetItems(items)
}

// configureCurrentProvider configures the current provider
func (m *WizardTUIModel) configureCurrentProvider() tea.Cmd {
	return func() tea.Msg {
		if m.currentProvider == nil {
			return providerConfiguredMsg{
				err: gerror.New(gerror.ErrCodeInternal, "no current provider", nil),
			}
		}

		configured, err := m.wizard.ConfigureProvider(m.ctx, *m.currentProvider)
		if err != nil {
			return providerConfiguredMsg{err: err}
		}

		return providerConfiguredMsg{
			provider: *configured,
			err:      nil,
		}
	}
}

// handleNextProvider moves to the next provider or to agent creation
func (m *WizardTUIModel) handleNextProvider() tea.Cmd {
	m.currentProviderIndex++
	if m.currentProviderIndex < len(m.selectedProviders) {
		// Configure next provider
		m.currentProvider = &m.selectedProviders[m.currentProviderIndex]
		return nil
	} else {
		// All providers configured, move to agent creation
		m.state = stateAgentCreation
		m.currentStep = 4
		return nil
	}
}

// createAgents creates the agents
func (m *WizardTUIModel) createAgents() tea.Cmd {
	return func() tea.Msg {
		m.state = stateProgress
		m.currentStep = 5

		agents, err := m.wizard.CreateAgents(m.ctx, m.configuredProviders)
		if err != nil {
			return agentsCreatedMsg{err: err}
		}

		return agentsCreatedMsg{
			agents: agents,
			err:    nil,
		}
	}
}

// getSelectedProviders returns the currently selected providers
func (m *WizardTUIModel) getSelectedProviders() []setup.DetectedProvider {
	var selected []setup.DetectedProvider
	for i := 0; i < len(m.list.Items()); i++ {
		if item, ok := m.list.Items()[i].(ProviderItem); ok && item.Selected {
			selected = append(selected, item.DetectedProvider)
		}
	}
	return selected
}

// saveConfiguration saves the wizard configuration
func (m *WizardTUIModel) saveConfiguration() tea.Cmd {
	return func() tea.Msg {
		err := m.wizard.SaveConfiguration(m.ctx, m.configuredProviders, m.createdAgents)
		return configSavedMsg{err: err}
	}
}

// ProviderItem represents a provider in the list
type ProviderItem struct {
	setup.DetectedProvider
	Selected bool
}

// Title returns the title for the list item
func (p ProviderItem) Title() string {
	status := ""
	if p.HasCredentials {
		status += "✅ "
	}
	if p.IsLocal {
		status += "🏠 "
	}

	selected := ""
	if p.Selected {
		selected = "☑️ "
	} else {
		selected = "☐ "
	}

	return selected + status + p.Name
}

// Description returns the description for the list item
func (p ProviderItem) Description() string {
	desc := p.DetectedProvider.Notes
	if desc == "" {
		desc = p.DetectedProvider.Type
	}
	if p.HasCredentials {
		desc += " (credentials detected)"
	}
	if p.IsLocal {
		desc += " (local)"
	}
	return desc
}

// FilterValue returns the value to filter on
func (p ProviderItem) FilterValue() string {
	return p.Name
}

// ProviderDelegate renders provider list items
type ProviderDelegate struct{}

// NewProviderDelegate creates a new provider delegate
func NewProviderDelegate() ProviderDelegate {
	return ProviderDelegate{}
}

// Height returns the height of list items
func (d ProviderDelegate) Height() int {
	return 2
}

// Spacing returns the spacing between list items
func (d ProviderDelegate) Spacing() int {
	return 1
}

// Update handles updates to list items
func (d ProviderDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders a list item
func (d ProviderDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	p, ok := item.(ProviderItem)
	if !ok {
		return
	}

	title := p.Title()
	desc := p.Description()

	if index == m.Index() {
		// Selected item styling
		title = titleStyle.Render(title)
		desc = "  " + desc
		fmt.Fprintf(w, "%s\n%s", title, desc)
	} else {
		// Normal item styling
		desc = "  " + desc
		fmt.Fprintf(w, "%s\n%s", title, desc)
	}
}
