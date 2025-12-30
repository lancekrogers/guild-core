// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package testing provides test utilities for teatest with proper terminal cleanup
package testing

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest"
)

// TeaTestHelper provides utilities for testing with teatest
type TeaTestHelper struct {
	t             *testing.T
	terminalSaved bool
}

// NewTeaTestHelper creates a new test helper
func NewTeaTestHelper(t *testing.T) *TeaTestHelper {
	t.Helper()
	return &TeaTestHelper{t: t}
}

// RunTeaTest runs a teatest with proper cleanup
func (h *TeaTestHelper) RunTeaTest(model tea.Model, testFunc func(*teatest.TestModel)) {
	h.t.Helper()

	// Save terminal state before test
	h.saveTerminal()

	// Ensure cleanup happens
	defer h.restoreTerminal()

	// Create test model
	tm := teatest.NewTestModel(
		h.t,
		model,
		teatest.WithInitialTermSize(80, 24),
	)

	// Ensure the test model is properly terminated
	defer func() {
		// Try to gracefully quit first
		tm.Send(tea.QuitMsg{})

		// Wait briefly for quit to process
		done := make(chan struct{})
		go func() {
			tm.WaitFinished(h.t, teatest.WithFinalTimeout(100*time.Millisecond))
			close(done)
		}()

		select {
		case <-done:
			// Finished gracefully
		case <-time.After(200 * time.Millisecond):
			// Force cleanup if needed
			h.forceTerminalReset()
		}
	}()

	// Run the actual test
	testFunc(tm)
}

// saveTerminal saves the current terminal state
func (h *TeaTestHelper) saveTerminal() {
	// Execute stty to save terminal settings (Unix-like systems)
	if runtime.GOOS != "windows" {
		if output, err := exec.Command("stty", "-g").Output(); err == nil {
			// Store the settings (in a real implementation, we'd save this)
			h.t.Logf("Saved terminal state: %s", string(output))
			h.terminalSaved = true
		}
	}
}

// restoreTerminal restores the terminal to a clean state
func (h *TeaTestHelper) restoreTerminal() {
	// Always perform basic cleanup
	h.cleanupTerminal()

	// Additional restoration for Unix-like systems
	if runtime.GOOS != "windows" && h.terminalSaved {
		// Reset terminal to sane defaults
		exec.Command("stty", "sane").Run()
	}
}

// cleanupTerminal performs basic terminal cleanup using ANSI escape sequences
func (h *TeaTestHelper) cleanupTerminal() {
	// Exit alternate screen buffer
	fmt.Fprint(os.Stdout, "\033[?1049l")

	// Show cursor
	fmt.Fprint(os.Stdout, "\033[?25h")

	// Reset all text attributes
	fmt.Fprint(os.Stdout, "\033[0m")

	// Clear any remaining content
	fmt.Fprint(os.Stdout, "\033[2J")

	// Move cursor to top-left
	fmt.Fprint(os.Stdout, "\033[H")

	// Flush output
	os.Stdout.Sync()
}

// forceTerminalReset performs a more aggressive terminal reset
func (h *TeaTestHelper) forceTerminalReset() {
	h.t.Logf("Forcing terminal reset due to incomplete teatest cleanup")

	// Send interrupt sequences
	fmt.Fprint(os.Stdout, "\033c")   // Full reset
	fmt.Fprint(os.Stdout, "\033[!p") // Soft reset

	// Cleanup
	h.cleanupTerminal()

	// On Unix-like systems, use reset command
	if runtime.GOOS != "windows" {
		if err := exec.Command("reset").Run(); err != nil {
			// If reset fails, try tput
			exec.Command("tput", "reset").Run()
		}
	}
}

// TestModelWithCleanup wraps a tea.Model to ensure cleanup on quit
type TestModelWithCleanup struct {
	inner   tea.Model
	cleanup func()
}

// NewTestModelWithCleanup creates a model that runs cleanup on quit
func NewTestModelWithCleanup(model tea.Model, cleanup func()) *TestModelWithCleanup {
	return &TestModelWithCleanup{
		inner:   model,
		cleanup: cleanup,
	}
}

func (m *TestModelWithCleanup) Init() tea.Cmd {
	return m.inner.Init()
}

func (m *TestModelWithCleanup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.inner.Update(msg)

	// Check if we're quitting
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok && m.cleanup != nil {
			m.cleanup()
		}
	}

	// Update inner model reference
	if newInner, ok := model.(tea.Model); ok && newInner != m.inner {
		m.inner = newInner
	}

	return m, cmd
}

func (m *TestModelWithCleanup) View() string {
	return m.inner.View()
}

// WaitForOutput is a helper that waits for specific output with cleanup
func WaitForOutput(t *testing.T, tm *teatest.TestModel, match func([]byte) bool, timeout time.Duration) {
	t.Helper()

	teatest.WaitFor(
		t,
		tm.Output(),
		match,
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(timeout),
	)
}

// SendAndWait sends a message and waits briefly for processing
func SendAndWait(tm *teatest.TestModel, msg tea.Msg, wait time.Duration) {
	tm.Send(msg)
	time.Sleep(wait)
}

// Usage example for test files:
//
// func TestMyComponent(t *testing.T) {
//     helper := NewTeaTestHelper(t)
//     model := NewMyModel()
//
//     helper.RunTeaTest(model, func(tm *teatest.TestModel) {
//         // Wait for initial render
//         WaitForOutput(t, tm, func(b []byte) bool {
//             return bytes.Contains(b, []byte("Ready"))
//         }, 2*time.Second)
//
//         // Send input
//         SendAndWait(tm, tea.KeyMsg{Type: tea.KeyEnter}, 100*time.Millisecond)
//
//         // Test will cleanup automatically
//     })
// }
