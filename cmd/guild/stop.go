package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/guild-ventures/guild-core/internal/daemon"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop all Guild servers and processes",
	Long: `Terminate all running Guild servers and chat sessions.
	
This command will:
- Stop the Guild gRPC server (guild serve)
- Terminate all active chat sessions (guild chat)
- Kill any other guild-related processes`,
	RunE: runStop,
}

var (
	forceKill bool
)

func init() {
	stopCmd.Flags().BoolVarP(&forceKill, "force", "f", false, "Force kill processes (SIGKILL instead of SIGTERM)")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	fmt.Println("🛑 Stopping Guild processes...")

	// First try to stop the daemon if it's managed by our daemon package
	if daemon.IsRunning() {
		fmt.Print("  • Stopping managed Guild server... ")
		if err := daemon.Stop(); err != nil {
			fmt.Printf("❌ %v\n", err)
		} else {
			fmt.Println("✅")
		}
	}

	// Then look for any other guild processes
	processes, err := findGuildProcesses()
	if err != nil {
		return fmt.Errorf("failed to find guild processes: %w", err)
	}

	if len(processes) == 0 && !daemon.IsRunning() {
		fmt.Println("⚠️  No Guild processes found running")
		return nil
	}

	// Display found processes
	fmt.Printf("\nFound %d Guild process(es):\n", len(processes))
	for _, proc := range processes {
		fmt.Printf("  • PID %s: %s\n", proc.pid, proc.command)
	}

	// Kill each process
	killedCount := 0
	for _, proc := range processes {
		if err := killProcess(proc.pid, forceKill); err != nil {
			fmt.Printf("❌ Failed to stop PID %s: %v\n", proc.pid, err)
		} else {
			fmt.Printf("✅ Stopped PID %s\n", proc.pid)
			killedCount++
		}
	}

	// Summary
	fmt.Printf("\n✨ Stopped %d of %d process(es)\n", killedCount, len(processes))

	// Check if gRPC server port is still in use
	if isPortInUse("9090") {
		fmt.Println("⚠️  Port 9090 may still be in use. You might need to wait a moment or use --force")
	}

	return nil
}

type guildProcess struct {
	pid     string
	command string
}

func findGuildProcesses() ([]guildProcess, error) {
	var processes []guildProcess

	switch runtime.GOOS {
	case "darwin", "linux":
		// Use ps to find guild processes
		cmd := exec.Command("ps", "aux")
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// Look for guild processes but skip our own stop command
			if strings.Contains(line, "guild") && !strings.Contains(line, "guild stop") {
				// Check if it's actually a guild command (serve, chat, etc.)
				if strings.Contains(line, "guild serve") || 
				   strings.Contains(line, "guild chat") ||
				   strings.Contains(line, "./guild serve") ||
				   strings.Contains(line, "./guild chat") ||
				   strings.Contains(line, "bin/guild") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						// Extract full command for display
						cmdStart := strings.Index(line, fields[10])
						if cmdStart > 0 {
							processes = append(processes, guildProcess{
								pid:     fields[1],
								command: strings.TrimSpace(line[cmdStart:]),
							})
						}
					}
				}
			}
		}
	case "windows":
		// Use tasklist for Windows
		cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq guild.exe")
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}

		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i < 3 { // Skip header lines
				continue
			}
			if strings.Contains(line, "guild.exe") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					processes = append(processes, guildProcess{
						pid:     fields[1],
						command: "guild.exe",
					})
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return processes, nil
}

func killProcess(pid string, force bool) error {
	switch runtime.GOOS {
	case "darwin", "linux":
		signal := "-TERM"
		if force {
			signal = "-KILL"
		}
		cmd := exec.Command("kill", signal, pid)
		return cmd.Run()
	case "windows":
		flag := ""
		if force {
			flag = "/F"
		}
		cmd := exec.Command("taskkill", flag, "/PID", pid)
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func isPortInUse(port string) bool {
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("lsof", "-i", ":"+port)
		output, _ := cmd.Output()
		return len(output) > 0
	case "windows":
		cmd := exec.Command("netstat", "-an")
		output, _ := cmd.Output()
		return strings.Contains(string(output), ":"+port)
	default:
		return false
	}
}