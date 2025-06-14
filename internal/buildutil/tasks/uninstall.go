// internal/buildutil/tasks/uninstall.go
package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/guild-ventures/guild-core/internal/buildutil/ui"
)

// Uninstall removes the guild binary from the user's Go bin directory
func Uninstall(verbose bool) error {
	ui.Section("Uninstalling Guild Framework")

	// Determine Go bin directory
	goBin := getGoBinPath()
	
	// Handle Windows .exe extension
	binaryName := "guild"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	
	destPath := filepath.Join(goBin, binaryName)
	
	// Check if binary exists
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		ui.Warning(fmt.Sprintf("Guild is not installed at %s", destPath))
		return nil
	}
	
	// Remove the binary
	ui.Task("Uninstalling", fmt.Sprintf("Removing %s", destPath))
	
	if err := os.Remove(destPath); err != nil {
		ui.TaskFail()
		return fmt.Errorf("failed to remove %s: %w", destPath, err)
	}
	
	ui.TaskPass()
	
	fmt.Println("")
	ui.Success("Guild uninstalled successfully!")
	
	return nil
}