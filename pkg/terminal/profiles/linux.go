package profiles

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/terminal"
)

// LinuxProfileDetector detects Linux-specific terminal profiles
type LinuxProfileDetector struct {
	detector *terminal.Detector
}

// NewLinuxProfileDetector creates a Linux profile detector
func NewLinuxProfileDetector() *LinuxProfileDetector {
	return &LinuxProfileDetector{
		detector: terminal.NewDetector(),
	}
}

// DetectProfile detects the specific Linux terminal in use
func (lpd *LinuxProfileDetector) DetectProfile(ctx context.Context) (*terminal.Profile, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Linux profile detection")
	}

	// Check for specific terminal programs
	if profile := lpd.detectByProgram(); profile != nil {
		return profile, nil
	}

	// Check by VTE version (GNOME Terminal family)
	if profile := lpd.detectVTETerminal(); profile != nil {
		return profile, nil
	}

	// Check by TERM environment variable
	if profile := lpd.detectByTerm(); profile != nil {
		return profile, nil
	}

	// Check for desktop environment specific terminals
	if profile := lpd.detectByDesktopEnvironment(); profile != nil {
		return profile, nil
	}

	// Default Linux profile
	return lpd.createDefaultLinuxProfile(), nil
}

// detectByProgram detects terminal by program name or specific environment
func (lpd *LinuxProfileDetector) detectByProgram() *terminal.Profile {
	termProgram := os.Getenv("TERM_PROGRAM")

	switch termProgram {
	case "vscode":
		return lpd.createVSCodeProfile()
	}

	// Check for specific terminal indicators
	if os.Getenv("KONSOLE_VERSION") != "" {
		return lpd.createKonsoleProfile()
	}

	if os.Getenv("ALACRITTY_SOCKET") != "" || os.Getenv("TERM") == "alacritty" {
		return lpd.createAlacrittyProfile()
	}

	if os.Getenv("TERM") == "xterm-kitty" {
		return lpd.createKittyProfile()
	}

	if os.Getenv("TERMINOLOGY") != "" {
		return lpd.createTerminologyProfile()
	}

	if os.Getenv("TILIX_ID") != "" {
		return lpd.createTilixProfile()
	}

	return nil
}

// detectVTETerminal detects VTE-based terminals (GNOME Terminal family)
func (lpd *LinuxProfileDetector) detectVTETerminal() *terminal.Profile {
	vteVersion := os.Getenv("VTE_VERSION")
	if vteVersion == "" {
		return nil
	}

	// Check for specific VTE-based terminals
	if os.Getenv("GNOME_TERMINAL_SERVICE") != "" {
		return lpd.createGnomeTerminalProfile(vteVersion)
	}

	// Generic VTE terminal
	return lpd.createVTEProfile(vteVersion)
}

// detectByTerm detects terminal by TERM environment variable
func (lpd *LinuxProfileDetector) detectByTerm() *terminal.Profile {
	term := os.Getenv("TERM")

	switch {
	case strings.Contains(term, "tmux"):
		return lpd.createTmuxProfile()
	case strings.HasPrefix(term, "screen"):
		return lpd.createScreenProfile()
	case strings.Contains(term, "rxvt"):
		return lpd.createRxvtProfile()
	case strings.Contains(term, "st-"):
		return lpd.createStProfile()
	case term == "linux":
		return lpd.createLinuxConsoleProfile()
	}

	return nil
}

// detectByDesktopEnvironment detects terminal by desktop environment
func (lpd *LinuxProfileDetector) detectByDesktopEnvironment() *terminal.Profile {
	desktopSession := os.Getenv("DESKTOP_SESSION")
	xdgCurrentDesktop := os.Getenv("XDG_CURRENT_DESKTOP")

	// KDE Plasma
	if strings.Contains(desktopSession, "plasma") || strings.Contains(xdgCurrentDesktop, "KDE") {
		return lpd.createKonsoleProfile()
	}

	// XFCE
	if strings.Contains(desktopSession, "xfce") || strings.Contains(xdgCurrentDesktop, "XFCE") {
		return lpd.createXfceTerminalProfile()
	}

	// MATE
	if strings.Contains(xdgCurrentDesktop, "MATE") {
		return lpd.createMateTerminalProfile()
	}

	return nil
}

// createGnomeTerminalProfile creates a profile for GNOME Terminal
func (lpd *LinuxProfileDetector) createGnomeTerminalProfile(vteVersion string) *terminal.Profile {
	profile := &terminal.Profile{
		Name:        "gnome-terminal",
		Description: "GNOME Terminal",
		Priority:    90,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
			"hyperlink":    "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
		},
	}

	// Enhanced features for newer VTE versions
	if vteVersion != "" {
		if version, err := strconv.Atoi(vteVersion); err == nil {
			if version >= 5000 {
				// VTE 0.50+ has better hyperlink support
				profile.Capabilities.Hyperlinks = true
			}
		}
	}

	return profile
}

// createVTEProfile creates a generic VTE-based terminal profile
func (lpd *LinuxProfileDetector) createVTEProfile(vteVersion string) *terminal.Profile {
	return &terminal.Profile{
		Name:        "vte-terminal",
		Description: "VTE-based Terminal",
		Priority:    80,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
			"hyperlink":    "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
		},
	}
}

// createKonsoleProfile creates a profile for KDE Konsole
func (lpd *LinuxProfileDetector) createKonsoleProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "konsole",
		Description: "KDE Konsole",
		Priority:    85,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
			"bell":         "\x07",
		},
	}
}

// createAlacrittyProfile creates a profile for Alacritty
func (lpd *LinuxProfileDetector) createAlacrittyProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "alacritty-linux",
		Description: "Alacritty (Linux)",
		Priority:    95,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
			"hyperlink":    "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
		},
	}
}

// createKittyProfile creates a profile for Kitty terminal
func (lpd *LinuxProfileDetector) createKittyProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "kitty-linux",
		Description: "Kitty Terminal (Linux)",
		Priority:    95,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			Images:          true,
			CursorShape:     true,
			AlternateScreen: true,
			Kitty:           true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
			"hyperlink":    "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
			"kitty_image":  "\x1b_G%s\x1b\\",
		},
	}
}

// createTerminologyProfile creates a profile for Enlightenment Terminology
func (lpd *LinuxProfileDetector) createTerminologyProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "terminology",
		Description: "Enlightenment Terminology",
		Priority:    80,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Images:          true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// createTilixProfile creates a profile for Tilix terminal
func (lpd *LinuxProfileDetector) createTilixProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "tilix",
		Description: "Tilix Terminal",
		Priority:    85,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
			"hyperlink":    "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
		},
	}
}

// createXfceTerminalProfile creates a profile for XFCE Terminal
func (lpd *LinuxProfileDetector) createXfceTerminalProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "xfce4-terminal",
		Description: "XFCE Terminal",
		Priority:    75,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Extended256,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// createMateTerminalProfile creates a profile for MATE Terminal
func (lpd *LinuxProfileDetector) createMateTerminalProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "mate-terminal",
		Description: "MATE Terminal",
		Priority:    75,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Extended256,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// createVSCodeProfile creates a profile for VS Code on Linux
func (lpd *LinuxProfileDetector) createVSCodeProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "vscode-linux",
		Description: "VS Code Integrated Terminal (Linux)",
		Priority:    90,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
			"hyperlink":    "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
		},
	}
}

// createTmuxProfile creates a profile for tmux
func (lpd *LinuxProfileDetector) createTmuxProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "tmux-linux",
		Description: "tmux (Linux)",
		Priority:    70,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Extended256,
			Unicode:         true,
			Mouse:           false, // Depends on tmux configuration
			Size:            true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// createScreenProfile creates a profile for GNU Screen
func (lpd *LinuxProfileDetector) createScreenProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "screen-linux",
		Description: "GNU Screen (Linux)",
		Priority:    60,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Extended256,
			Unicode:         true,
			Mouse:           false,
			Size:            true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
		},
	}
}

// createRxvtProfile creates a profile for rxvt terminals
func (lpd *LinuxProfileDetector) createRxvtProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "rxvt",
		Description: "rxvt Terminal",
		Priority:    50,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Extended256,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// createStProfile creates a profile for st (simple terminal)
func (lpd *LinuxProfileDetector) createStProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "st",
		Description: "st (simple terminal)",
		Priority:    70,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Extended256,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// createLinuxConsoleProfile creates a profile for Linux console
func (lpd *LinuxProfileDetector) createLinuxConsoleProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "linux-console",
		Description: "Linux Console (TTY)",
		Priority:    30,
		Capabilities: terminal.Capabilities{
			Colors:  terminal.Basic16,
			Unicode: false, // Limited Unicode support in console
			Size:    true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"bell":         "\x07",
		},
	}
}

// createDefaultLinuxProfile creates a default Linux profile
func (lpd *LinuxProfileDetector) createDefaultLinuxProfile() *terminal.Profile {
	caps, _ := lpd.detector.Detect(context.Background())

	return &terminal.Profile{
		Name:         "linux-default",
		Description:  "Default Linux Terminal",
		Priority:     40,
		Capabilities: caps,
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// LinuxDistributionInfo provides information about the Linux distribution
type LinuxDistributionInfo struct {
	Name    string
	Version string
	ID      string
}

// DetectLinuxDistribution attempts to detect the Linux distribution
func DetectLinuxDistribution() LinuxDistributionInfo {
	// This would typically read /etc/os-release or similar files
	// For now, return basic information
	return LinuxDistributionInfo{
		Name:    "Linux",
		Version: "Unknown",
		ID:      "linux",
	}
}

// LinuxKeyMappings provides Linux-specific key mappings
func LinuxKeyMappings() map[string]string {
	return map[string]string{
		"copy":       "Ctrl+Shift+C",
		"paste":      "Ctrl+Shift+V",
		"cut":        "Ctrl+Shift+X",
		"undo":       "Ctrl+Z",
		"redo":       "Ctrl+Y",
		"select_all": "Ctrl+A",
		"find":       "Ctrl+F",
		"save":       "Ctrl+S",
		"new":        "Ctrl+N",
		"open":       "Ctrl+O",
		"close":      "Ctrl+W",
		"quit":       "Ctrl+Q",
		"fullscreen": "F11",
		"refresh":    "F5",
		"new_tab":    "Ctrl+Shift+T",
		"next_tab":   "Ctrl+Page_Down",
		"prev_tab":   "Ctrl+Page_Up",
		"split_h":    "Ctrl+Shift+H",
		"split_v":    "Ctrl+Shift+V",
	}
}

// GnomeTerminalFeatures returns GNOME Terminal specific features
func GnomeTerminalFeatures() map[string]bool {
	return map[string]bool{
		"tabs":         true,
		"profiles":     true,
		"transparency": true,
		"background":   true,
		"emoji":        true,
		"hyperlinks":   true,
		"search":       true,
		"zoom":         true,
	}
}

// KonsoleFeatures returns Konsole specific features
func KonsoleFeatures() map[string]bool {
	return map[string]bool{
		"tabs":         true,
		"splits":       true,
		"profiles":     true,
		"themes":       true,
		"transparency": true,
		"background":   true,
		"emoji":        true,
		"search":       true,
		"bookmarks":    true,
		"monitor":      true,
	}
}
