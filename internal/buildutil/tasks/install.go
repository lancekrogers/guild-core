// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// internal/buildutil/tasks/install.go
package tasks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/guild-ventures/guild-core/internal/buildutil/ui"
)

// Install installs the guild binary to the user's Go bin directory
func Install(verbose bool) error {
	ui.Section("Installing Guild Framework")

	// Determine Go bin directory
	goBin := getGoBinPath()

	// Handle Windows .exe extension
	binaryName := "guild"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	sourcePath := filepath.Join("bin", "guild")
	destPath := filepath.Join(goBin, binaryName)

	// Check if source binary exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("guild binary not found at %s. Run 'make build' first", sourcePath)
	}

	// Create Go bin directory if it doesn't exist
	if err := os.MkdirAll(goBin, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", goBin, err)
	}

	// Copy the binary
	ui.Task("Installing", fmt.Sprintf("Copying to %s", destPath))

	input, err := os.ReadFile(sourcePath)
	if err != nil {
		ui.TaskFail()
		return fmt.Errorf("failed to read source binary: %w", err)
	}

	err = os.WriteFile(destPath, input, 0755)
	if err != nil {
		ui.TaskFail()
		return fmt.Errorf("failed to write to %s: %w", destPath, err)
	}

	ui.TaskPass()

	// Code sign the binary on macOS to prevent Gatekeeper issues
	if runtime.GOOS == "darwin" {
		ui.Task("Signing", "guild binary for macOS security")

		cmd := exec.Command("codesign", "--force", "--sign", "-", destPath)
		if err := cmd.Run(); err != nil {
			ui.TaskFail()
			return fmt.Errorf("failed to sign binary for macOS: %w", err)
		}

		ui.TaskPass()
	}

	// Check if Go bin is in PATH
	pathEnv := os.Getenv("PATH")
	inPath := false
	for _, p := range strings.Split(pathEnv, string(os.PathListSeparator)) {
		if p == goBin {
			inPath = true
			break
		}
	}

	fmt.Println("")
	ui.Success("Guild installed successfully!")
	fmt.Println("")

	if inPath {
		fmt.Printf("✓ %s is already in your PATH\n", goBin)
		fmt.Println("  You can now run: guild")
	} else {
		ui.Warning(fmt.Sprintf("WARNING: %s is not in your PATH", goBin))
		fmt.Println("")
		fmt.Println("To add it to your PATH, add one of these lines to your shell config:")
		fmt.Println("")

		// Provide shell-specific instructions
		shell := os.Getenv("SHELL")
		if strings.Contains(shell, "bash") {
			fmt.Println("  # For bash (~/.bashrc or ~/.bash_profile):")
			fmt.Printf("  export PATH=\"$PATH:%s\"\n", goBin)
		} else if strings.Contains(shell, "zsh") {
			fmt.Println("  # For zsh (~/.zshrc):")
			fmt.Printf("  export PATH=\"$PATH:%s\"\n", goBin)
		} else if strings.Contains(shell, "fish") {
			fmt.Println("  # For fish (~/.config/fish/config.fish):")
			fmt.Printf("  set -gx PATH $PATH %s\n", goBin)
		} else {
			// Generic instructions
			fmt.Println("  # For bash (~/.bashrc or ~/.bash_profile):")
			fmt.Printf("  export PATH=\"$PATH:%s\"\n", goBin)
			fmt.Println("")
			fmt.Println("  # For zsh (~/.zshrc):")
			fmt.Printf("  export PATH=\"$PATH:%s\"\n", goBin)
			fmt.Println("")
			fmt.Println("  # For fish (~/.config/fish/config.fish):")
			fmt.Printf("  set -gx PATH $PATH %s\n", goBin)
		}

		if runtime.GOOS == "windows" {
			fmt.Println("")
			fmt.Println("  # For Windows:")
			fmt.Println("  Add the following directory to your PATH environment variable:")
			fmt.Printf("  %s\n", goBin)
			fmt.Println("  (System Properties → Environment Variables → Path → Edit)")
		}

		fmt.Println("")
		fmt.Println("Then reload your shell or run: source ~/.bashrc (or appropriate config file)")
	}

	return nil
}

// getGoBinPath returns the Go bin directory path
func getGoBinPath() string {
	// First check GOPATH environment variable
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin")
	}

	// Fall back to go env GOPATH
	cmd := exec.Command("go", "env", "GOPATH")
	output, err := cmd.Output()
	if err == nil {
		gopath := strings.TrimSpace(string(output))
		if gopath != "" {
			return filepath.Join(gopath, "bin")
		}
	}

	// Last resort: use home directory
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, "go", "bin")
	}

	// This shouldn't happen, but return a sensible default
	return filepath.Join(".", "go", "bin")
}
