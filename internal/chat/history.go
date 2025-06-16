// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"bufio"
	"os"
	"strings"
)

// CommandHistory manages command history with search and navigation
type CommandHistory struct {
	historyFile string
	commands    []string
	currentPos  int
	maxSize     int
}

// NewCommandHistory creates a new command history manager
func NewCommandHistory(historyFile string) *CommandHistory {
	ch := &CommandHistory{
		historyFile: historyFile,
		commands:    make([]string, 0),
		currentPos:  -1,
		maxSize:     1000,
	}

	ch.loadHistory()
	return ch
}

// Add adds a command to history
func (ch *CommandHistory) Add(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	// Remove duplicates
	ch.removeDuplicate(command)

	// Add to end
	ch.commands = append(ch.commands, command)

	// Maintain size limit
	if len(ch.commands) > ch.maxSize {
		ch.commands = ch.commands[1:]
	}

	// Reset position
	ch.currentPos = len(ch.commands)

	// Save to disk
	ch.saveHistory()
}

// Previous returns the previous command in history
func (ch *CommandHistory) Previous() string {
	if len(ch.commands) == 0 {
		return ""
	}

	if ch.currentPos > 0 {
		ch.currentPos--
	}

	if ch.currentPos < len(ch.commands) {
		return ch.commands[ch.currentPos]
	}
	return ""
}

// Next returns the next command in history
func (ch *CommandHistory) Next() string {
	if len(ch.commands) == 0 {
		return ""
	}

	if ch.currentPos < len(ch.commands)-1 {
		ch.currentPos++
		return ch.commands[ch.currentPos]
	}

	// At the end, return empty
	ch.currentPos = len(ch.commands)
	return ""
}

// Search performs fuzzy search on command history
func (ch *CommandHistory) Search(term string) []string {
	if term == "" {
		return ch.GetRecent(10)
	}

	results := make([]string, 0)
	termLower := strings.ToLower(term)

	// Search from most recent to oldest
	for i := len(ch.commands) - 1; i >= 0; i-- {
		cmd := ch.commands[i]
		if strings.Contains(strings.ToLower(cmd), termLower) {
			results = append(results, cmd)
			if len(results) >= 10 { // Limit results
				break
			}
		}
	}

	return results
}

// GetRecent returns the most recent n commands
func (ch *CommandHistory) GetRecent(n int) []string {
	if n > len(ch.commands) {
		n = len(ch.commands)
	}

	start := len(ch.commands) - n
	result := make([]string, n)
	copy(result, ch.commands[start:])

	// Reverse to get most recent first
	for i := 0; i < len(result)/2; i++ {
		result[i], result[len(result)-1-i] = result[len(result)-1-i], result[i]
	}

	return result
}

// Clear clears the command history
func (ch *CommandHistory) Clear() {
	ch.commands = make([]string, 0)
	ch.currentPos = -1
	ch.saveHistory()
}

// removeDuplicate removes duplicate commands
func (ch *CommandHistory) removeDuplicate(command string) {
	newCommands := make([]string, 0, len(ch.commands))
	for _, cmd := range ch.commands {
		if cmd != command {
			newCommands = append(newCommands, cmd)
		}
	}
	ch.commands = newCommands
}

// loadHistory loads history from file
func (ch *CommandHistory) loadHistory() {
	file, err := os.Open(ch.historyFile)
	if err != nil {
		// File doesn't exist yet, that's ok
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			ch.commands = append(ch.commands, line)
		}
	}

	// Maintain size limit
	if len(ch.commands) > ch.maxSize {
		ch.commands = ch.commands[len(ch.commands)-ch.maxSize:]
	}

	ch.currentPos = len(ch.commands)
}

// saveHistory saves history to file
func (ch *CommandHistory) saveHistory() {
	// Create directory if needed
	dir := strings.TrimSuffix(ch.historyFile, "/chat_history.txt")
	os.MkdirAll(dir, 0755)

	file, err := os.Create(ch.historyFile)
	if err != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, cmd := range ch.commands {
		writer.WriteString(cmd + "\n")
	}
	writer.Flush()
}
