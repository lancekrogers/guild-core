package components

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExternalCommandResult represents the result of executing an external command
type ExternalCommandResult struct {
	Command   string
	Output    string
	Success   bool
	Error     error
	ExitCode  int
}

// ExecuteExternalCommand runs an external guild command and returns the result
func ExecuteExternalCommand(cmdStr string) ExternalCommandResult {
	// Get the current executable path
	exePath, err := os.Executable()
	if err != nil {
		return ExternalCommandResult{
			Command:  cmdStr,
			Output:   "",
			Success:  false,
			Error:    fmt.Errorf("failed to get executable path: %w", err),
			ExitCode: -1,
		}
	}
	
	// Get the directory containing the current executable
	exeDir := filepath.Dir(exePath)
	
	// Construct the path to the guild binary
	guildBin := filepath.Join(exeDir, "guild")
	
	// Split command into args
	args := strings.Fields(cmdStr)
	
	// Create command
	cmd := exec.Command(guildBin, args...)
	
	// Set current directory as working directory
	cmd.Dir, _ = os.Getwd()
	
	// Capture output
	output, err := cmd.CombinedOutput()
	
	// Check for errors
	exitCode := 0
	success := true
	if err != nil {
		success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	
	return ExternalCommandResult{
		Command:  cmdStr,
		Output:   string(output),
		Success:  success,
		Error:    err,
		ExitCode: exitCode,
	}
}