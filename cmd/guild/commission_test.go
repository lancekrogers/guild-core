// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommissionCommandHelp(t *testing.T) {
	// Create a new command
	cmd := &cobra.Command{Use: "guild"}
	cmd.AddCommand(commissionCmd)

	// Capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Test help
	cmd.SetArgs([]string{"commission", "--help"})
	err := cmd.Execute()
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Commission specialized artisans to collaborate on complex tasks")
	assert.Contains(t, output, "guild commission [description]")
}

func TestCommissionCommandValidation(t *testing.T) {
	// Create a test directory
	tempDir, err := os.MkdirTemp("", "guild-commission-cli-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a basic guild.yaml
	guildDir := filepath.Join(tempDir, ".campaign")
	err = os.MkdirAll(guildDir, 0755)
	require.NoError(t, err)

	guildYAML := `name: Test Guild
version: 1.0.0
agents:
  - id: manager-1
    name: Guild Manager
    type: manager
    provider: mock
    model: test-model
    capabilities: [planning, coordination]
`
	err = os.WriteFile(filepath.Join(guildDir, "guild.yaml"), []byte(guildYAML), 0644)
	require.NoError(t, err)

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test commission command without description
	cmd := &cobra.Command{Use: "guild"}
	cmd.AddCommand(commissionCmd)

	// Capture both stdout and stderr
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Test without description - should show help since it has subcommands
	cmd.SetArgs([]string{"commission"})
	err = cmd.Execute()

	// Should succeed and show help
	assert.NoError(t, err)

	output := buf.String()
	// Should show help text with usage and available commands
	assert.Contains(t, output, "Commission specialized artisans")
	assert.Contains(t, output, "Available Commands:")
}

// TestCommissionListCommand tests the commission list command
// TODO: This test requires full Guild infrastructure setup including database migrations
// Skip for now until we have proper test infrastructure setup
func TestCommissionListCommand(t *testing.T) {
	t.Skip("Skipping commission list test - requires full Guild infrastructure setup")
}

// TestCommissionWorkshopCommand tests the workshop command
// TODO: This test requires full Guild infrastructure setup including database migrations
// Skip for now until we have proper test infrastructure setup
func TestCommissionWorkshopCommand(t *testing.T) {
	t.Skip("Skipping workshop command test - requires full Guild infrastructure setup")
}

// TestCommissionExecutionSetup tests the setupGuildComponents function
// TODO: This test requires full Guild infrastructure setup including database migrations
// Skip for now until we have proper test infrastructure setup
func TestCommissionExecutionSetup(t *testing.T) {
	t.Skip("Skipping execution setup test - requires full Guild infrastructure setup")
}
