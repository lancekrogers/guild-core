//go:build windows
// +build windows

package daemon

import (
	"fmt"
	"os/exec"
	"syscall"
)

// setupSysProcAttr configures Windows-specific process attributes for background execution
func setupSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// stopProcess uses Windows-specific commands to terminate a process
func stopProcess(pid int) error {
	kill := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", pid))
	return kill.Run()
}

// isProcessRunningByPID checks if a process with the given PID is running on Windows
func isProcessRunningByPID(pid int) bool {
	// On Windows, we need to use different approach
	// This is a placeholder for proper Windows implementation
	process, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	syscall.CloseHandle(process)
	return true
}