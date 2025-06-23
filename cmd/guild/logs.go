// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/guild-ventures/guild-core/pkg/paths"
)

// logsCmd represents the logs command for viewing Guild debug logs
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Guild debug logs",
	Long:  `View and manage Guild debug logs for troubleshooting.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// logsViewCmd shows recent Guild logs
var logsViewCmd = &cobra.Command{
	Use:   "view [lines]",
	Short: "View recent log entries",
	Long:  `View recent Guild log entries for debugging.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lines := 20 // default
		if len(args) > 0 {
			if n, err := fmt.Sscanf(args[0], "%d", &lines); err != nil || n != 1 {
				fmt.Fprintf(os.Stderr, "Invalid number of lines: %s\n", args[0])
				os.Exit(1)
			}
		}
		
		viewLogs(lines)
	},
}

// logsListCmd lists available log files
var logsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available log files",
	Long:  `List all available Guild log files.`,
	Run: func(cmd *cobra.Command, args []string) {
		listLogFiles()
	},
}

// logsClearCmd clears log files
var logsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear log files",
	Long:  `Clear Guild log files (keeps today's log).`,
	Run: func(cmd *cobra.Command, args []string) {
		clearLogs()
	},
}

func init() {
	// Register logs command and subcommands
	rootCmd.AddCommand(logsCmd)
	logsCmd.AddCommand(logsViewCmd)
	logsCmd.AddCommand(logsListCmd)
	logsCmd.AddCommand(logsClearCmd)
}

func viewLogs(lines int) {
	guildDir, err := paths.GetGuildConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get Guild directory: %v\n", err)
		os.Exit(1)
	}
	
	latestLog := filepath.Join(guildDir, "logs", "latest.log")
	
	// Check if log file exists
	if _, err := os.Stat(latestLog); os.IsNotExist(err) {
		fmt.Println("No log files found. Try running a Guild command to generate logs.")
		fmt.Printf("Expected log location: %s\n", latestLog)
		return
	}
	
	// Read the log file
	content, err := os.ReadFile(latestLog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read log file: %v\n", err)
		os.Exit(1)
	}
	
	if len(content) == 0 {
		fmt.Println("Log file is empty.")
		return
	}
	
	// Split into lines and show last N lines
	logLines := []string{}
	current := ""
	for _, b := range content {
		if b == '\n' {
			if current != "" {
				logLines = append(logLines, current)
				current = ""
			}
		} else {
			current += string(b)
		}
	}
	if current != "" {
		logLines = append(logLines, current)
	}
	
	start := 0
	if len(logLines) > lines {
		start = len(logLines) - lines
	}
	
	fmt.Printf("📋 Last %d log entries:\n\n", len(logLines)-start)
	for i := start; i < len(logLines); i++ {
		fmt.Println(logLines[i])
	}
	
	fmt.Printf("\n💡 Log file: %s\n", latestLog)
}

func listLogFiles() {
	guildDir, err := paths.GetGuildConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get Guild directory: %v\n", err)
		os.Exit(1)
	}
	
	logsDir := filepath.Join(guildDir, "logs")
	
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read logs directory: %v\n", err)
		fmt.Printf("Expected logs directory: %s\n", logsDir)
		return
	}
	
	if len(entries) == 0 {
		fmt.Println("No log files found.")
		return
	}
	
	fmt.Printf("📂 Guild log files in %s:\n\n", logsDir)
	
	var logFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && (entry.Name() == "latest.log" || filepath.Ext(entry.Name()) == ".log") {
			logFiles = append(logFiles, entry)
		}
	}
	
	// Sort by modification time (newest first)
	sort.Slice(logFiles, func(i, j int) bool {
		infoI, _ := logFiles[i].Info()
		infoJ, _ := logFiles[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})
	
	for _, entry := range logFiles {
		info, _ := entry.Info()
		size := info.Size()
		modTime := info.ModTime().Format("2006-01-02 15:04:05")
		
		sizeStr := fmt.Sprintf("%d bytes", size)
		if size > 1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
		}
		
		fmt.Printf("  %s  %s  %s\n", entry.Name(), modTime, sizeStr)
	}
}

func clearLogs() {
	guildDir, err := paths.GetGuildConfigDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get Guild directory: %v\n", err)
		os.Exit(1)
	}
	
	logsDir := filepath.Join(guildDir, "logs")
	
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		fmt.Printf("No logs directory found at %s\n", logsDir)
		return
	}
	
	today := time.Now().Format("2006-01-02")
	todayLog := fmt.Sprintf("guild-%s.log", today)
	
	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		// Don't delete today's log or the latest symlink
		if name == todayLog || name == "latest.log" {
			continue
		}
		
		filePath := filepath.Join(logsDir, name)
		if err := os.Remove(filePath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to delete %s: %v\n", name, err)
		} else {
			fmt.Printf("Deleted %s\n", name)
			deleted++
		}
	}
	
	if deleted == 0 {
		fmt.Println("No old log files to delete.")
	} else {
		fmt.Printf("Deleted %d old log files.\n", deleted)
	}
}