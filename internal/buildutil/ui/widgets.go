// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// internal/buildutil/ui/widgets.go
package ui

import (
	"fmt"
	"strings"
	"sync"
)

var (
	screenMutex sync.Mutex
	progressMsg string
)

// Section renders a boxed heading with rounded borders
func Section(title string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	width := TermWidth()

	bar := strings.Repeat("─", width-2)

	fmt.Println()
	if ColourEnabled() {
		fmt.Printf("%s┌%s┐%s\n", Cyan, bar, Reset)
		fmt.Printf("%s│%s%s%s│%s\n", Cyan, Reset, Center(Bold+"🏰 "+title+Reset, width-2), Cyan, Reset)
		fmt.Printf("%s└%s┘%s\n", Cyan, bar, Reset)
	} else {
		fmt.Printf("┌%s┐\n", bar)
		fmt.Printf("│%s│\n", Center("🏰 "+title, width-2))
		fmt.Printf("└%s┘\n", bar)
	}
}

// Progress redraws an in-place progress bar
func Progress(curr, total int, msg string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		// Clear the current line completely, then print new progress
		fmt.Printf("\r%s[%d/%d] %s", ClearLine, curr, total, msg)
	} else {
		// Simple format without ANSI codes
		fmt.Printf("\r[%d/%d] %s", curr, total, msg)
	}

	// Store message for clearing later
	progressMsg = msg
}

// Status prints a status indicator with checkmark or X
func Status(label string, ok bool) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	symbol := "✗"
	color := Red
	if ok {
		symbol = "✓"
		color = Green
	}

	if ColourEnabled() {
		fmt.Printf("  %s%s%s %s\n", color, symbol, Reset, label)
	} else {
		fmt.Printf("  %s %s\n", symbol, label)
	}
}

// ClearProgress clears any in-progress output
func ClearProgress() {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if progressMsg != "" {
		if ColourEnabled() {
			fmt.Printf("\r%s\n", ClearLine) // Clear the line and move to next line
		} else {
			fmt.Println() // Just move to next line
		}
		progressMsg = ""
	}
}

// Task displays a task in progress
func Task(action, description string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		fmt.Printf("  %s[%s]%s %s... ", Yellow, action, Reset, description)
	} else {
		fmt.Printf("  [%s] %s... ", action, description)
	}
}

// TaskPass marks the current task as passed
func TaskPass() {
	if ColourEnabled() {
		fmt.Printf("%s✓%s\n", Green, Reset)
	} else {
		fmt.Println("✓")
	}
}

// TaskFail marks the current task as failed
func TaskFail() {
	if ColourEnabled() {
		fmt.Printf("%s✗%s\n", Red, Reset)
	} else {
		fmt.Println("✗")
	}
}

// Success prints a success message
func Success(msg string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		fmt.Printf("%s✓ %s%s\n", Green, msg, Reset)
	} else {
		fmt.Printf("✓ %s\n", msg)
	}
}

// Warning prints a warning message
func Warning(msg string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	if ColourEnabled() {
		fmt.Printf("%s⚠ %s%s\n", Yellow, msg, Reset)
	} else {
		fmt.Printf("⚠ %s\n", msg)
	}
}

// SummaryCard displays a final status card
func SummaryCard(title string, rows [][]string, totalTime string, success bool) {
	SummaryCardWithStatus(title, rows, totalTime, success, "", "")
}

// SummaryCardWithStatus displays a final status card with custom status messages
func SummaryCardWithStatus(title string, rows [][]string, totalTime string, success bool, successMsg, failMsg string) {
	screenMutex.Lock()
	defer screenMutex.Unlock()

	width := TermWidth()

	bar := strings.Repeat("═", width-2)

	// Top border
	fmt.Println()
	if ColourEnabled() {
		fmt.Printf("%s╔%s╗%s\n", Cyan, bar, Reset)
		fmt.Printf("%s║%s%s%s║%s\n", Cyan, Reset, Center(Bold+"🏰 "+title+Reset, width-2), Cyan, Reset)
		fmt.Printf("%s╠%s╣%s\n", Cyan, bar, Reset)
	} else {
		fmt.Printf("╔%s╗\n", bar)
		fmt.Printf("║%s║\n", Center("🏰 "+title, width-2))
		fmt.Printf("╠%s╣\n", bar)
	}

	// Calculate column widths
	if len(rows) > 0 {
		colWidths := make([]int, len(rows[0]))
		for _, row := range rows {
			for i, cell := range row {
				if cellLen := visualLength(cell); cellLen > colWidths[i] {
					colWidths[i] = cellLen
				}
			}
		}

		// Render rows
		for i, row := range rows {
			if ColourEnabled() {
				fmt.Printf("%s║%s ", Cyan, Reset)
			} else {
				fmt.Print("║ ")
			}

			// Calculate total content width needed
			totalColWidth := 0
			for _, cw := range colWidths {
				totalColWidth += cw
			}

			// Account for spaces between columns (2 spaces per column except last)
			spaceBetweenCols := (len(colWidths) - 1) * 2
			contentWidth := totalColWidth + spaceBetweenCols

			// Calculate available space and distribute extra to last column
			availableSpace := width - 4 // "║ " at start, " ║" at end
			extraSpace := availableSpace - contentWidth
			if extraSpace > 0 && len(colWidths) > 0 {
				colWidths[len(colWidths)-1] += extraSpace
			}

			// Render each cell with proper spacing
			for j, cell := range row {
				fmt.Print(cell)
				padding := colWidths[j] - visualLength(cell)
				if j < len(row)-1 {
					padding += 2 // Add space between columns except for last column
				}
				fmt.Print(strings.Repeat(" ", padding))
			}

			if ColourEnabled() {
				fmt.Printf(" %s║%s\n", Cyan, Reset)
			} else {
				fmt.Println(" ║")
			}

			// Separator after header
			if i == 0 {
				if ColourEnabled() {
					fmt.Printf("%s╠%s╣%s\n", Cyan, bar, Reset)
				} else {
					fmt.Printf("╠%s╣\n", bar)
				}
			}
		}
	}

	// Time summary
	if ColourEnabled() {
		fmt.Printf("%s╠%s╣%s\n", Cyan, bar, Reset)
		fmt.Printf("%s║%s%s%s║%s\n", Cyan, Reset, Center("Total Time: "+totalTime, width-2), Cyan, Reset)
	} else {
		fmt.Printf("╠%s╣\n", bar)
		fmt.Printf("║%s║\n", Center("Total Time: "+totalTime, width-2))
	}

	// Status
	if ColourEnabled() {
		fmt.Printf("%s╠%s╣%s\n", Cyan, bar, Reset)
	} else {
		fmt.Printf("╠%s╣\n", bar)
	}

	// Use custom messages if provided, otherwise default to build messages
	var statusText string
	if success {
		if successMsg != "" {
			statusText = successMsg
		} else {
			statusText = "✓ BUILD SUCCESSFUL"
		}
	} else {
		if failMsg != "" {
			statusText = failMsg
		} else {
			statusText = "✗ BUILD FAILED"
		}
	}

	if ColourEnabled() {
		statusColor := Green
		if !success {
			statusColor = Red
		}
		statusLine := Center(Bold+statusColor+statusText+Reset, width-2)
		fmt.Printf("%s║%s%s%s║%s\n", Cyan, Reset, statusLine, Cyan, Reset)
	} else {
		fmt.Printf("║%s║\n", Center(statusText, width-2))
	}

	// Bottom border
	if ColourEnabled() {
		fmt.Printf("%s╚%s╝%s\n\n", Cyan, bar, Reset)
	} else {
		fmt.Printf("╚%s╝\n\n", bar)
	}
}
