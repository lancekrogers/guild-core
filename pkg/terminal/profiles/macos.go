package profiles

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/terminal"
)

// MacOSProfileDetector detects macOS-specific terminal profiles
type MacOSProfileDetector struct {
	detector *terminal.Detector
}

// NewMacOSProfileDetector creates a macOS profile detector
func NewMacOSProfileDetector() *MacOSProfileDetector {
	return &MacOSProfileDetector{
		detector: terminal.NewDetector(),
	}
}

// DetectProfile detects the specific macOS terminal in use
func (mpd *MacOSProfileDetector) DetectProfile(ctx context.Context) (*terminal.Profile, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during macOS profile detection")
	}

	termProgram := os.Getenv("TERM_PROGRAM")
	termVersion := os.Getenv("TERM_PROGRAM_VERSION")

	switch termProgram {
	case "iTerm.app":
		return mpd.createITerm2Profile(termVersion), nil
	case "Apple_Terminal":
		return mpd.createTerminalAppProfile(termVersion), nil
	case "vscode":
		return mpd.createVSCodeProfile(), nil
	case "Hyper":
		return mpd.createHyperProfile(), nil
	}

	// Check for other terminals by TERM environment
	term := os.Getenv("TERM")
	switch {
	case term == "xterm-kitty":
		return mpd.createKittyProfile(), nil
	case term == "alacritty":
		return mpd.createAlacrittyProfile(), nil
	case strings.Contains(term, "tmux"):
		return mpd.createTmuxProfile(), nil
	case strings.Contains(term, "screen"):
		return mpd.createScreenProfile(), nil
	}

	// Default macOS profile
	return mpd.createDefaultMacOSProfile(), nil
}

// createITerm2Profile creates a profile for iTerm2
func (mpd *MacOSProfileDetector) createITerm2Profile(version string) *terminal.Profile {
	profile := &terminal.Profile{
		Name:        "iterm2",
		Description: "iTerm2",
		Priority:    100,
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
			ITerm2:          true,
		},
		Overrides: map[string]string{
			"clear_screen":     "\x1b[2J\x1b[H",
			"title":            "\x1b]0;%s\x07",
			"hyperlink":        "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
			"iterm2_image":     "\x1b]1337;File=%s\x07",
			"iterm2_badge":     "\x1b]1337;SetBadgeFormat=%s\x07",
			"iterm2_mark":      "\x1b]1337;SetMark\x07",
			"cursor_block":     "\x1b]50;CursorShape=0\x07",
			"cursor_underline": "\x1b]50;CursorShape=1\x07",
			"cursor_bar":       "\x1b]50;CursorShape=2\x07",
		},
	}

	// Enhanced capabilities for newer versions
	if version != "" {
		if ver, err := parseVersion(version); err == nil {
			if ver >= 3.0 {
				profile.Capabilities.Sixel = true
			}
			if ver >= 3.3 {
				// Enhanced image protocol support
				profile.Overrides["iterm2_inline"] = "\x1b]1337;File=inline=1:%s\x07"
			}
		}
	}

	return profile
}

// createTerminalAppProfile creates a profile for Terminal.app
func (mpd *MacOSProfileDetector) createTerminalAppProfile(version string) *terminal.Profile {
	return &terminal.Profile{
		Name:        "terminal-app",
		Description: "macOS Terminal.app",
		Priority:    90,
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
			"bell":         "\x07",
		},
	}
}

// createVSCodeProfile creates a profile for VS Code on macOS
func (mpd *MacOSProfileDetector) createVSCodeProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "vscode-macos",
		Description: "VS Code Integrated Terminal (macOS)",
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

// createHyperProfile creates a profile for Hyper terminal
func (mpd *MacOSProfileDetector) createHyperProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "hyper",
		Description: "Hyper Terminal",
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

// createKittyProfile creates a profile for Kitty terminal
func (mpd *MacOSProfileDetector) createKittyProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "kitty-macos",
		Description: "Kitty Terminal (macOS)",
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
			"clear_screen":  "\x1b[2J\x1b[H",
			"title":         "\x1b]0;%s\x07",
			"hyperlink":     "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
			"kitty_image":   "\x1b_G%s\x1b\\",
			"kitty_unicode": "\x1b_G%s\x1b\\",
		},
	}
}

// createAlacrittyProfile creates a profile for Alacritty
func (mpd *MacOSProfileDetector) createAlacrittyProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "alacritty-macos",
		Description: "Alacritty (macOS)",
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

// createTmuxProfile creates a profile for tmux on macOS
func (mpd *MacOSProfileDetector) createTmuxProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "tmux-macos",
		Description: "tmux (macOS)",
		Priority:    70,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Extended256,
			Unicode:         true,
			Mouse:           false, // Mouse support depends on tmux config
			Size:            true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// createScreenProfile creates a profile for GNU Screen on macOS
func (mpd *MacOSProfileDetector) createScreenProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "screen-macos",
		Description: "GNU Screen (macOS)",
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

// createDefaultMacOSProfile creates a default macOS profile
func (mpd *MacOSProfileDetector) createDefaultMacOSProfile() *terminal.Profile {
	caps, _ := mpd.detector.Detect(context.Background())

	return &terminal.Profile{
		Name:         "macos-default",
		Description:  "Default macOS Terminal",
		Priority:     50,
		Capabilities: caps,
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// parseVersion parses a version string to a float
func parseVersion(version string) (float64, error) {
	// Extract major.minor from version string
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, gerror.New(gerror.ErrCodeValidation, "invalid version format", nil)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid major version")
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid minor version")
	}

	return float64(major) + float64(minor)/10.0, nil
}

// MacOSColorScheme provides macOS-specific color schemes
type MacOSColorScheme struct {
	DarkMode bool
}

// GetMacOSColorScheme returns a color scheme for macOS
func GetMacOSColorScheme(darkMode bool) map[string]string {
	if darkMode {
		return map[string]string{
			"background": "\x1b[48;2;30;30;30m",    // Dark background
			"foreground": "\x1b[38;2;236;236;236m", // Light text
			"primary":    "\x1b[38;2;10;132;255m",  // System Blue
			"accent":     "\x1b[38;2;191;90;242m",  // System Purple
			"success":    "\x1b[38;2;50;215;75m",   // System Green
			"warning":    "\x1b[38;2;255;214;10m",  // System Yellow
			"error":      "\x1b[38;2;255;105;97m",  // System Red
			"muted":      "\x1b[38;2;142;142;147m", // System Gray
		}
	}

	return map[string]string{
		"background": "\x1b[48;2;255;255;255m", // Light background
		"foreground": "\x1b[38;2;0;0;0m",       // Dark text
		"primary":    "\x1b[38;2;0;122;255m",   // System Blue
		"accent":     "\x1b[38;2;175;82;222m",  // System Purple
		"success":    "\x1b[38;2;52;199;89m",   // System Green
		"warning":    "\x1b[38;2;255;204;0m",   // System Yellow
		"error":      "\x1b[38;2;255;59;48m",   // System Red
		"muted":      "\x1b[38;2;142;142;147m", // System Gray
	}
}

// MacOSKeyMappings provides macOS-specific key mappings
func MacOSKeyMappings() map[string]string {
	return map[string]string{
		"copy":       "Cmd+C",
		"paste":      "Cmd+V",
		"cut":        "Cmd+X",
		"undo":       "Cmd+Z",
		"redo":       "Cmd+Shift+Z",
		"select_all": "Cmd+A",
		"find":       "Cmd+F",
		"save":       "Cmd+S",
		"new":        "Cmd+N",
		"open":       "Cmd+O",
		"close":      "Cmd+W",
		"quit":       "Cmd+Q",
		"fullscreen": "Ctrl+Cmd+F",
		"refresh":    "Cmd+R",
		"new_tab":    "Cmd+T",
		"next_tab":   "Cmd+Shift+]",
		"prev_tab":   "Cmd+Shift+[",
		"split_pane": "Cmd+D",
	}
}

// DetectMacOSVersion attempts to detect macOS version
func DetectMacOSVersion() string {
	// This would typically use system calls to get the actual version
	// For now, return a simplified version
	return "macOS"
}

// ITerm2Features returns iTerm2-specific features
func ITerm2Features() map[string]bool {
	return map[string]bool{
		"tabs":              true,
		"splits":            true,
		"profiles":          true,
		"themes":            true,
		"background":        true,
		"transparency":      true,
		"ligatures":         true,
		"emoji":             true,
		"powerline":         true,
		"images":            true,
		"badges":            true,
		"marks":             true,
		"triggers":          true,
		"hotkey_window":     true,
		"shell_integration": true,
	}
}

// TerminalAppFeatures returns Terminal.app-specific features
func TerminalAppFeatures() map[string]bool {
	return map[string]bool{
		"tabs":          true,
		"profiles":      true,
		"themes":        true,
		"transparency":  true,
		"background":    true,
		"emoji":         true,
		"accessibility": true,
	}
}

// DetectSystemAppearance detects if macOS is in dark mode
func DetectSystemAppearance() string {
	// This would typically check the system appearance setting
	// For now, check environment or return default
	if os.Getenv("MACOS_APPEARANCE") == "dark" {
		return "dark"
	}
	return "light"
}
