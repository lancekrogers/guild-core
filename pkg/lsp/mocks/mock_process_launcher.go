// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package mocks

import (
	"context"

	"github.com/guild-framework/guild-core/pkg/lsp"
)

// MockProcessLauncher is a mock implementation of ProcessLauncherInterface
type MockProcessLauncher struct {
	// Mock client to return
	MockClient *MockLSPClient

	// Error injection
	LaunchError error

	// Call tracking
	LaunchCalls int
	LastCommand string
	LastArgs    []string
	LastWorkDir string
}

// NewMockProcessLauncher creates a new mock process launcher
func NewMockProcessLauncher() *MockProcessLauncher {
	return &MockProcessLauncher{
		MockClient: NewMockLSPClient(),
	}
}

// LaunchServer returns the mock client instead of launching a real process
func (m *MockProcessLauncher) LaunchServer(ctx context.Context, command string, args []string, workDir string) (lsp.ClientInterface, error) {
	m.LaunchCalls++
	m.LastCommand = command
	m.LastArgs = args
	m.LastWorkDir = workDir

	if m.LaunchError != nil {
		return nil, m.LaunchError
	}

	// Simulate starting the mock client
	if err := m.MockClient.Start(ctx); err != nil {
		return nil, err
	}

	return m.MockClient, nil
}

// SetLaunchError sets an error to be returned by LaunchServer
func (m *MockProcessLauncher) SetLaunchError(err error) {
	m.LaunchError = err
}
