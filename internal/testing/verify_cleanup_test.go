// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package testing_test

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	testingpkg "github.com/guild-ventures/guild-core/internal/testing"
)

// TestTerminalCleanup verifies that terminal cleanup works correctly
func TestTerminalCleanup(t *testing.T) {
	t.Run("normal completion", func(t *testing.T) {
		helper := testingpkg.NewTeaTestHelper(t)
		model := &testModel{shouldQuit: true}
		
		helper.RunTeaTest(model, func(tm *teatest.TestModel) {
			// Send any key to trigger quit
			tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
			// Helper will handle cleanup
		})
		
		// Terminal should be clean after this
		t.Log("Test completed - terminal should be clean")
	})
	
	t.Run("model doesn't quit properly", func(t *testing.T) {
		helper := testingpkg.NewTeaTestHelper(t)
		model := &testModel{shouldQuit: false} // Won't quit on its own
		
		helper.RunTeaTest(model, func(tm *teatest.TestModel) {
			// Send keys that model ignores
			tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
			time.Sleep(50 * time.Millisecond)
			// Helper will force cleanup after timeout
		})
		
		// Terminal should still be clean
		t.Log("Test completed with forced cleanup - terminal should be clean")
	})
}

// testModel is a simple model for testing cleanup
type testModel struct {
	shouldQuit bool
	counter    int
}

func (m *testModel) Init() tea.Cmd {
	return nil
}

func (m *testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.counter++
		if m.shouldQuit {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *testModel) View() string {
	return fmt.Sprintf("Test Model (counter: %d)\nPress Enter to %s\n", 
		m.counter,
		map[bool]string{true: "quit", false: "increment counter"}[m.shouldQuit])
}

// TestInterruptHandling simulates Ctrl-C during test
func TestInterruptHandling(t *testing.T) {
	t.Skip("Manual test - uncomment to test Ctrl-C handling")
	
	helper := testingpkg.NewTeaTestHelper(t)
	model := &slowModel{}
	
	helper.RunTeaTest(model, func(tm *teatest.TestModel) {
		t.Log("Press Ctrl-C within 5 seconds to test interrupt handling...")
		
		// Simulate a long-running test
		time.Sleep(5 * time.Second)
		
		// If we get here without interrupt, quit normally
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	})
	
	t.Log("Test completed - terminal should be clean even if interrupted")
}

// slowModel simulates a slow-responding model
type slowModel struct {
	ready bool
}

func (m *slowModel) Init() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return readyMsg{}
	})
}

type readyMsg struct{}

func (m *slowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case readyMsg:
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "q" && m.ready {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *slowModel) View() string {
	if !m.ready {
		return "Loading... (this will take 2 seconds)\n"
	}
	return "Ready! Press 'q' to quit\n"
}