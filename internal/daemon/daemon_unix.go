// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build !windows
// +build !windows

package daemon

import (
	"os/exec"
	"syscall"
)

// setupSysProcAttr configures Unix-specific process attributes for daemonization
func setupSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session
	}
}

// isProcessRunningByPID checks if a process with the given PID is running on Unix systems
func isProcessRunningByPID(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}
