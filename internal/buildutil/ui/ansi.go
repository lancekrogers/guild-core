// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// internal/buildutil/ui/ansi.go
package ui

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

var noColor bool

// ANSI color codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Cyan   = "\033[36m"
	// Cursor control
	HideCursor = "\033[?25l"
	ShowCursor = "\033[?25h"
	ClearLine  = "\033[2K"
	MoveUp     = "\033[A"
)

// Init initializes the UI package based on environment
func Init(noColorFlag bool) {
	noColor = noColorFlag || os.Getenv("NO_COLOR") != "" || os.Getenv("CI") != "" || !isatty()
}

// ColourEnabled returns true if colors should be used
func ColourEnabled() bool {
	return !noColor
}

// Color wraps text with ANSI color code if enabled
func Color(text, code string) string {
	if noColor {
		return text
	}
	return code + text + Reset
}

// isatty checks if stdout is a terminal
func isatty() bool {
	// Simplified isatty check
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (terminal)
	return fi.Mode()&os.ModeCharDevice != 0
}

// TIOCGWINSZ is the ioctl command to get window size
// Value is specific to macOS/Darwin
const TIOCGWINSZ = 0x40087468

// TermWidth returns the terminal width
func TermWidth() int {
	// Try environment variable first
	if cols := os.Getenv("COLUMNS"); cols != "" {
		var width int
		fmt.Sscanf(cols, "%d", &width)
		if width > 0 {
			return width
		}
	}

	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}
	ws := &winsize{}

	// Try stdout first
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(os.Stdout.Fd()),
		uintptr(TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)),
	)

	// If stdout fails, try stderr
	if (err != 0 || ws.Col == 0) && isatty() {
		_, _, err = syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(os.Stderr.Fd()),
			uintptr(TIOCGWINSZ),
			uintptr(unsafe.Pointer(ws)),
		)
	}

	if err != 0 || ws.Col == 0 {
		return 80
	}
	return int(ws.Col)
}

// Center centers text to given width
func Center(text string, width int) string {
	textLen := visualLength(text)
	if textLen >= width {
		return text
	}
	leftPad := (width - textLen) / 2
	rightPad := width - textLen - leftPad
	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}

// visualLength calculates the visual width of text, accounting for:
// - ANSI escape codes (which take 0 visual space)
// - Unicode characters like emojis (which may take 2 visual spaces)
func visualLength(text string) int {
	// First strip ANSI codes
	cleaned := stripANSI(text)

	// Count visual width
	width := 0
	for len(cleaned) > 0 {
		r, size := utf8.DecodeRuneInString(cleaned)
		if r == utf8.RuneError {
			width++
		} else {
			// Most emojis and wide characters take 2 spaces
			if r >= 0x1F000 {
				width += 2
			} else {
				width++
			}
		}
		cleaned = cleaned[size:]
	}
	return width
}

// stripANSI removes ANSI escape codes from text
func stripANSI(text string) string {
	// Simple implementation - could be improved with regex
	result := text
	for _, code := range []string{Reset, Bold, Red, Green, Yellow, Cyan} {
		result = strings.ReplaceAll(result, code, "")
	}
	return result
}
