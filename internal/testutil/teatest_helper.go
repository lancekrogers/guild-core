// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package testutil

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TeaTestHelper provides safe teatest execution with proper cleanup
type TeaTestHelper struct {
	tb          testing.TB
	testModel   *teatest.TestModel
	cleanupDone bool
	done        chan struct{}
}

// NewTeaTestHelper creates a helper that ensures proper terminal cleanup
func NewTeaTestHelper(tb testing.TB) *TeaTestHelper {
	tb.Helper()

	helper := &TeaTestHelper{
		tb:   tb,
		done: make(chan struct{}),
	}

	// Register cleanup immediately
	tb.Cleanup(func() {
		helper.Cleanup()
	})

	// Handle interrupt signals to ensure cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			helper.Cleanup()
			os.Exit(1)
		case <-helper.done:
			// Test completed normally
		}
	}()

	return helper
}

// RunModel runs a Bubble Tea model with automatic cleanup
func (h *TeaTestHelper) RunModel(model tea.Model, options ...teatest.TestOption) *teatest.TestModel {
	h.tb.Helper()

	// Default options for safety
	defaultOpts := []teatest.TestOption{
		teatest.WithInitialTermSize(80, 24),
	}
	options = append(defaultOpts, options...)

	// Create test model
	h.testModel = teatest.NewTestModel(h.tb, model, options...)

	return h.testModel
}

// Cleanup ensures terminal is properly restored
func (h *TeaTestHelper) Cleanup() {
	if h.cleanupDone {
		return
	}
	h.cleanupDone = true

	// Signal that we're done
	close(h.done)

	// Restore terminal state
	restoreTerminal()

	// Give terminal time to restore
	time.Sleep(10 * time.Millisecond)
}

// restoreTerminal forcefully restores terminal to a sane state
func restoreTerminal() {
	// Exit alternate screen buffer
	os.Stdout.WriteString("\033[?1049l")

	// Show cursor
	os.Stdout.WriteString("\033[?25h")

	// Reset all attributes
	os.Stdout.WriteString("\033[0m")

	// Reset terminal mode (this fixes the box drawing issues)
	os.Stdout.WriteString("\033c")

	// Ensure output is flushed
	os.Stdout.Sync()
}

// SafeWaitFinished waits for the model to finish with a timeout
func SafeWaitFinished(tb testing.TB, tm *teatest.TestModel, timeout time.Duration) {
	tb.Helper()

	done := make(chan struct{})
	go func() {
		tm.WaitFinished(tb, teatest.WithFinalTimeout(timeout))
		close(done)
	}()

	select {
	case <-done:
		// Normal completion
	case <-time.After(timeout):
		tb.Fatalf("Test timed out after %v", timeout)
	}
}

// SendQuit sends the quit message and waits safely
func SendQuit(tb testing.TB, tm *teatest.TestModel) {
	tb.Helper()

	tm.Send(tea.QuitMsg{})
	SafeWaitFinished(tb, tm, 2*time.Second)
}
