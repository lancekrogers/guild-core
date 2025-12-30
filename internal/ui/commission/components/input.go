// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package components provides UI components for the objective UI
package components

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// InputModel is a model for text input
type InputModel struct {
	textInput textinput.Model
	label     string
}

// NewInputModel creates a new input model
func NewInputModel(label string, placeholder string) InputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()

	return InputModel{
		textInput: ti,
		label:     label,
	}
}

// Init initializes the input model
func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update updates the input model
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the input model
func (m InputModel) View() string {
	return m.label + "\n" + m.textInput.View()
}

// Value returns the current input value
func (m InputModel) Value() string {
	return m.textInput.Value()
}

// SetValue sets the input value
func (m *InputModel) SetValue(value string) {
	m.textInput.SetValue(value)
}
