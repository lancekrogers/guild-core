// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package testutil

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// CommandResult holds the result of running a command
type CommandResult struct {
	Stdout   string
	Stderr   string
	Error    error
	ExitCode int
}

// RunGuildCommand runs a guild command in the given directory
func RunGuildCommand(t *testing.T, workDir string, args ...string) *CommandResult {
	t.Helper()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Build guild binary path
	guildBinary := "guild"
	if binPath := os.Getenv("GUILD_BINARY"); binPath != "" {
		guildBinary = binPath
	}
	
	cmd := exec.CommandContext(ctx, guildBinary, args...)
	cmd.Dir = workDir
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	result := &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Error:  err,
	}
	
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
	}
	
	t.Logf("Command: guild %s", strings.Join(args, " "))
	if result.Error != nil {
		t.Logf("Error: %v", result.Error)
		t.Logf("Stderr: %s", result.Stderr)
	}
	
	return result
}

// ProjectContextExtensions adds command execution to ProjectContext
type ProjectContextExtensions struct {
	*ProjectContext
}

// RunGuild runs a guild command in the project context
func (p *ProjectContextExtensions) RunGuild(args ...string) *CommandResult {
	return RunGuildCommand(p.t, p.RootPath, args...)
}

// ExtendProjectContext adds command execution capabilities to a ProjectContext
func ExtendProjectContext(ctx *ProjectContext) *ProjectContextExtensions {
	return &ProjectContextExtensions{ProjectContext: ctx}
}