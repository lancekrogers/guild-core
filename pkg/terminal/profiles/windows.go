package profiles

import (
	"context"
	"os"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/terminal"
)

// WindowsProfileDetector detects Windows-specific terminal profiles
type WindowsProfileDetector struct {
	detector *terminal.Detector
}

// NewWindowsProfileDetector creates a Windows profile detector
func NewWindowsProfileDetector() *WindowsProfileDetector {
	return &WindowsProfileDetector{
		detector: terminal.NewDetector(),
	}
}

// DetectProfile detects the specific Windows terminal in use
func (wpd *WindowsProfileDetector) DetectProfile(ctx context.Context) (*terminal.Profile, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Windows profile detection")
	}

	// Windows Terminal
	if os.Getenv("WT_SESSION") != "" {
		return wpd.createWindowsTerminalProfile(), nil
	}

	// ConEmu
	if os.Getenv("ConEmuPID") != "" {
		return wpd.createConEmuProfile(), nil
	}

	// VS Code integrated terminal
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return wpd.createVSCodeProfile(), nil
	}

	// PowerShell ISE
	if os.Getenv("PSISE") != "" {
		return wpd.createPowerShellISEProfile(), nil
	}

	// Git Bash / MSYS2
	if strings.Contains(strings.ToLower(os.Getenv("MSYSTEM")), "msys") {
		return wpd.createGitBashProfile(), nil
	}

	// Cygwin
	if os.Getenv("CYGWIN") != "" {
		return wpd.createCygwinProfile(), nil
	}

	// Windows Subsystem for Linux
	if wpd.isWSL() {
		return wpd.createWSLProfile(), nil
	}

	// Legacy Command Prompt
	if wpd.isLegacyCMD() {
		return wpd.createLegacyCMDProfile(), nil
	}

	// Default Windows profile
	return wpd.createDefaultWindowsProfile(), nil
}

// createWindowsTerminalProfile creates a profile for Windows Terminal
func (wpd *WindowsProfileDetector) createWindowsTerminalProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "windows-terminal",
		Description: "Microsoft Windows Terminal",
		Priority:    100,
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
			"bell":         "\x07",
		},
	}
}

// createConEmuProfile creates a profile for ConEmu
func (wpd *WindowsProfileDetector) createConEmuProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "conemu",
		Description: "ConEmu Terminal",
		Priority:    85,
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

// createVSCodeProfile creates a profile for VS Code integrated terminal
func (wpd *WindowsProfileDetector) createVSCodeProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "vscode-windows",
		Description: "VS Code Integrated Terminal (Windows)",
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
			"hyperlink":    "\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\",
		},
	}
}

// createPowerShellISEProfile creates a profile for PowerShell ISE
func (wpd *WindowsProfileDetector) createPowerShellISEProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "powershell-ise",
		Description: "PowerShell Integrated Scripting Environment",
		Priority:    60,
		Capabilities: terminal.Capabilities{
			Colors:  terminal.Basic16,
			Unicode: true,
			Size:    true,
		},
		Overrides: map[string]string{
			"clear_screen": "Clear-Host",
		},
	}
}

// createGitBashProfile creates a profile for Git Bash/MSYS2
func (wpd *WindowsProfileDetector) createGitBashProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "git-bash",
		Description: "Git Bash / MSYS2",
		Priority:    75,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Extended256,
			Unicode:         true,
			Mouse:           false, // Limited mouse support
			Size:            true,
			AlternateScreen: true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "\x1b]0;%s\x07",
		},
	}
}

// createCygwinProfile creates a profile for Cygwin
func (wpd *WindowsProfileDetector) createCygwinProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "cygwin",
		Description: "Cygwin Terminal",
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
		},
	}
}

// createWSLProfile creates a profile for Windows Subsystem for Linux
func (wpd *WindowsProfileDetector) createWSLProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "wsl",
		Description: "Windows Subsystem for Linux",
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
		},
	}
}

// createLegacyCMDProfile creates a profile for legacy Command Prompt
func (wpd *WindowsProfileDetector) createLegacyCMDProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "cmd",
		Description: "Windows Command Prompt (Legacy)",
		Priority:    20,
		Capabilities: terminal.Capabilities{
			Colors:  terminal.Basic16,
			Unicode: false, // Limited Unicode support in legacy CMD
			Size:    true,
		},
		Overrides: map[string]string{
			"clear_screen": "cls",
			"title":        "title %s",
		},
	}
}

// createDefaultWindowsProfile creates a default Windows profile
func (wpd *WindowsProfileDetector) createDefaultWindowsProfile() *terminal.Profile {
	caps, _ := wpd.detector.Detect(context.Background())

	return &terminal.Profile{
		Name:         "windows-default",
		Description:  "Default Windows Terminal",
		Priority:     50,
		Capabilities: caps,
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
		},
	}
}

// isWSL detects if running under Windows Subsystem for Linux
func (wpd *WindowsProfileDetector) isWSL() bool {
	// Check for WSL environment variables
	if os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}

	// Check for WSL in WSLENV
	if os.Getenv("WSLENV") != "" {
		return true
	}

	// Check kernel release for Microsoft
	return false // Simplified check
}

// isLegacyCMD detects legacy Command Prompt
func (wpd *WindowsProfileDetector) isLegacyCMD() bool {
	// Check if we're in a basic console without modern features
	term := os.Getenv("TERM")

	// If TERM is not set or is basic, and we're not in any other modern terminal
	if term == "" || term == "dumb" {
		return true
	}

	return false
}

// WindowsColorScheme provides Windows-specific color schemes
type WindowsColorScheme struct {
	LightMode bool
}

// GetWindowsColorScheme returns a color scheme for Windows
func GetWindowsColorScheme(lightMode bool) map[string]string {
	if lightMode {
		return map[string]string{
			"background": "\x1b[48;2;255;255;255m", // White
			"foreground": "\x1b[38;2;0;0;0m",       // Black
			"primary":    "\x1b[38;2;0;120;215m",   // Windows Blue
			"accent":     "\x1b[38;2;0;103;192m",   // Darker Blue
			"success":    "\x1b[38;2;16;124;16m",   // Green
			"warning":    "\x1b[38;2;152;104;0m",   // Orange
			"error":      "\x1b[38;2;196;43;28m",   // Red
			"muted":      "\x1b[38;2;96;96;96m",    // Gray
		}
	}

	return map[string]string{
		"background": "\x1b[48;2;12;12;12m",    // Dark
		"foreground": "\x1b[38;2;204;204;204m", // Light Gray
		"primary":    "\x1b[38;2;86;156;214m",  // Light Blue
		"accent":     "\x1b[38;2;156;220;254m", // Cyan
		"success":    "\x1b[38;2;78;201;176m",  // Teal
		"warning":    "\x1b[38;2;220;220;170m", // Yellow
		"error":      "\x1b[38;2;244;71;71m",   // Red
		"muted":      "\x1b[38;2;128;128;128m", // Gray
	}
}

// WindowsKeyMappings provides Windows-specific key mappings
func WindowsKeyMappings() map[string]string {
	return map[string]string{
		"copy":       "Ctrl+C",
		"paste":      "Ctrl+V",
		"cut":        "Ctrl+X",
		"undo":       "Ctrl+Z",
		"redo":       "Ctrl+Y",
		"select_all": "Ctrl+A",
		"find":       "Ctrl+F",
		"save":       "Ctrl+S",
		"new":        "Ctrl+N",
		"open":       "Ctrl+O",
		"close":      "Ctrl+W",
		"quit":       "Alt+F4",
		"fullscreen": "F11",
		"refresh":    "F5",
	}
}

// DetectWindowsVersion attempts to detect Windows version
func DetectWindowsVersion() string {
	// This is a simplified implementation
	// In practice, you'd use Windows API calls to get version info

	if os.Getenv("WT_SESSION") != "" {
		return "Windows 10+" // Windows Terminal requires Windows 10+
	}

	return "Unknown"
}

// WindowsTerminalFeatures returns Windows Terminal specific features
func WindowsTerminalFeatures() map[string]bool {
	return map[string]bool{
		"tabs":          true,
		"panes":         true,
		"profiles":      true,
		"themes":        true,
		"background":    true,
		"transparency":  true,
		"ligatures":     true,
		"emoji":         true,
		"powerline":     true,
		"cascadia_code": true,
	}
}
