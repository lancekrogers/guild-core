// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package testing provides test utilities and cleanup patterns for teatest
package testing

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/guild-framework/guild-core/internal/teatest"
)

// TeaTestCleanup provides a pattern for proper teatest cleanup
type TeaTestCleanup struct {
	mu      sync.Mutex
	cleanup []func()
}

// NewTeaTestCleanup creates a new cleanup manager
func NewTeaTestCleanup() *TeaTestCleanup {
	return &TeaTestCleanup{
		cleanup: make([]func(), 0),
	}
}

// Add registers a cleanup function
func (tc *TeaTestCleanup) Add(f func()) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cleanup = append(tc.cleanup, f)
}

// RunAll executes all cleanup functions
func (tc *TeaTestCleanup) RunAll() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Run in reverse order (LIFO)
	for i := len(tc.cleanup) - 1; i >= 0; i-- {
		if tc.cleanup[i] != nil {
			tc.cleanup[i]()
		}
	}
	tc.cleanup = tc.cleanup[:0]
}

// SafeTeaTest wraps teatest with proper cleanup and signal handling
func SafeTeaTest(t *testing.T, model tea.Model, opts ...teatest.TestOption) *teatest.TestModel {
	t.Helper()

	// Create cleanup manager
	cleanup := NewTeaTestCleanup()

	// Save terminal state before test
	oldState := saveTerminalState()
	cleanup.Add(func() {
		restoreTerminalState(oldState)
	})

	// Set up signal handler to restore terminal on interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cleanup.Add(cancel)

	// Start signal handler
	go func() {
		select {
		case <-sigChan:
			// Restore terminal state immediately on signal
			restoreTerminalState(oldState)
			cancel()
		case <-ctx.Done():
			// Normal cleanup path
		}
	}()

	// Register test cleanup
	t.Cleanup(func() {
		// Stop signal handler
		signal.Stop(sigChan)
		close(sigChan)

		// Run all cleanup functions
		cleanup.RunAll()

		// Give terminal a moment to settle
		time.Sleep(50 * time.Millisecond)
	})

	// Create the test model with cleanup-aware options
	allOpts := append([]teatest.TestOption{
		teatest.WithInitialTermSize(80, 24),
	}, opts...)

	tm := teatest.NewTestModel(t, model, allOpts...)

	// Ensure WaitFinished is called on cleanup
	cleanup.Add(func() {
		// Use a short timeout to avoid hanging tests
		done := make(chan struct{})
		go func() {
			tm.WaitFinished(t, teatest.WithFinalTimeout(100*time.Millisecond))
			close(done)
		}()

		select {
		case <-done:
			// Finished normally
		case <-time.After(200 * time.Millisecond):
			// Force quit if still running
			tm.Send(tea.QuitMsg{})
		}
	})

	return tm
}

// TerminalState represents saved terminal state
type TerminalState struct {
	// In a real implementation, this would save actual terminal state
	// For now, it's a placeholder
	saved bool
}

// saveTerminalState captures current terminal state
func saveTerminalState() *TerminalState {
	// In a real implementation, this would:
	// 1. Save terminal mode (raw vs cooked)
	// 2. Save cursor visibility
	// 3. Save alternate screen state
	// 4. Save any other terminal settings
	return &TerminalState{saved: true}
}

// restoreTerminalState restores terminal to saved state
func restoreTerminalState(state *TerminalState) {
	if state == nil || !state.saved {
		return
	}

	// In a real implementation, this would:
	// 1. Exit alternate screen if needed
	// 2. Show cursor
	// 3. Reset terminal mode
	// 4. Clear any partial escape sequences

	// For now, we'll use ANSI escape sequences to reset common issues
	os.Stdout.Write([]byte("\033[?1049l")) // Exit alternate screen
	os.Stdout.Write([]byte("\033[?25h"))   // Show cursor
	os.Stdout.Write([]byte("\033[0m"))     // Reset all attributes
	os.Stdout.Write([]byte("\033[H"))      // Move cursor to home
	os.Stdout.Write([]byte("\033[J"))      // Clear screen
}

// Example test using SafeTeaTest
func ExampleSafeTeaTest(t *testing.T) {
	// Create a simple test model
	model := &simpleModel{}

	// Use SafeTeaTest instead of direct teatest
	tm := SafeTeaTest(t, model)

	// Wait for initial render
	teatest.WaitFor(
		t,
		tm.Output(),
		func(bts []byte) bool {
			return len(bts) > 0
		},
		teatest.WithCheckInterval(50*time.Millisecond),
		teatest.WithDuration(time.Second),
	)

	// Send quit command
	tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	// SafeTeaTest will handle cleanup automatically
}

// simpleModel is a minimal tea.Model for testing
type simpleModel struct {
	quitting bool
}

func (m *simpleModel) Init() tea.Cmd {
	return nil
}

func (m *simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *simpleModel) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}
	return tea.NewView("Press any key to quit\n")
}
