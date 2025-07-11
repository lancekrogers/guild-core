package profiles

import (
	"context"
	"os"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/terminal"
)

// MinimalProfileDetector detects minimal terminal environments
type MinimalProfileDetector struct {
	detector *terminal.Detector
}

// NewMinimalProfileDetector creates a minimal profile detector
func NewMinimalProfileDetector() *MinimalProfileDetector {
	return &MinimalProfileDetector{
		detector: terminal.NewDetector(),
	}
}

// DetectProfile detects minimal terminal environments like SSH, CI, etc.
func (mpd *MinimalProfileDetector) DetectProfile(ctx context.Context) (*terminal.Profile, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during minimal profile detection")
	}

	// CI environments
	if mpd.isCI() {
		return mpd.createCIProfile(), nil
	}

	// SSH sessions
	if mpd.isSSH() {
		return mpd.createSSHProfile(), nil
	}

	// Dumb terminal
	if mpd.isDumbTerminal() {
		return mpd.createDumbProfile(), nil
	}

	// Serial console
	if mpd.isSerialConsole() {
		return mpd.createSerialConsoleProfile(), nil
	}

	// Limited TTY
	if mpd.isLimitedTTY() {
		return mpd.createLimitedTTYProfile(), nil
	}

	// Headless environment
	if mpd.isHeadless() {
		return mpd.createHeadlessProfile(), nil
	}

	// Container environment
	if mpd.isContainer() {
		return mpd.createContainerProfile(), nil
	}

	// Generic fallback minimal profile
	return mpd.createFallbackProfile(), nil
}

// isCI detects if running in a CI environment
func (mpd *MinimalProfileDetector) isCI() bool {
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER",
		"JENKINS_URL", "TRAVIS", "CIRCLECI", "GITHUB_ACTIONS",
		"GITLAB_CI", "BUILDKITE", "DRONE", "TEAMCITY_VERSION",
		"APPVEYOR", "CODEBUILD_BUILD_ID", "TF_BUILD",
	}

	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}

	return false
}

// isSSH detects if running in an SSH session
func (mpd *MinimalProfileDetector) isSSH() bool {
	return os.Getenv("SSH_CONNECTION") != "" ||
		os.Getenv("SSH_CLIENT") != "" ||
		os.Getenv("SSH_TTY") != ""
}

// isDumbTerminal detects dumb terminal
func (mpd *MinimalProfileDetector) isDumbTerminal() bool {
	return os.Getenv("TERM") == "dumb"
}

// isSerialConsole detects serial console
func (mpd *MinimalProfileDetector) isSerialConsole() bool {
	term := os.Getenv("TERM")
	return term == "vt100" || term == "vt102" || term == "vt220"
}

// isLimitedTTY detects limited TTY environments
func (mpd *MinimalProfileDetector) isLimitedTTY() bool {
	// Check for very basic terminals
	term := strings.ToLower(os.Getenv("TERM"))
	limitedTerms := []string{"ansi", "vt52", "unknown"}

	for _, lt := range limitedTerms {
		if term == lt {
			return true
		}
	}

	return false
}

// isHeadless detects headless environments
func (mpd *MinimalProfileDetector) isHeadless() bool {
	// No display
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		// But has a terminal
		if os.Getenv("TERM") != "" {
			return true
		}
	}

	return false
}

// isContainer detects container environments
func (mpd *MinimalProfileDetector) isContainer() bool {
	// Docker
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Kubernetes
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	// Container runtime indicators
	if os.Getenv("container") != "" {
		return true
	}

	return false
}

// createCIProfile creates a profile for CI environments
func (mpd *MinimalProfileDetector) createCIProfile() *terminal.Profile {
	// Determine CI-specific capabilities
	caps := terminal.Capabilities{
		Colors:  terminal.NoColor,
		Unicode: false,
		Mouse:   false,
		Size:    false,
	}

	// Some CI systems support color
	if mpd.ciSupportsColor() {
		caps.Colors = terminal.Basic16
	}

	return &terminal.Profile{
		Name:         "ci-environment",
		Description:  "Continuous Integration Environment",
		Priority:     10,
		Capabilities: caps,
		Overrides: map[string]string{
			"clear_screen": "",     // No screen clearing in CI
			"title":        "",     // No title setting
			"bell":         "",     // No bell
			"progress":     "text", // Text-only progress
		},
	}
}

// createSSHProfile creates a profile for SSH sessions
func (mpd *MinimalProfileDetector) createSSHProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "ssh-session",
		Description: "SSH Remote Session",
		Priority:    25,
		Capabilities: terminal.Capabilities{
			Colors:          terminal.Basic16,
			Unicode:         false, // Conservative for SSH
			Mouse:           false, // Usually not supported over SSH
			Size:            true,
			AlternateScreen: false, // Avoid in SSH sessions
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "", // Don't set title over SSH
			"bell":         "", // No bell over SSH
		},
	}
}

// createDumbProfile creates a profile for dumb terminals
func (mpd *MinimalProfileDetector) createDumbProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "dumb-terminal",
		Description: "Dumb Terminal",
		Priority:    5,
		Capabilities: terminal.Capabilities{
			Colors:  terminal.NoColor,
			Unicode: false,
			Mouse:   false,
			Size:    false,
		},
		Overrides: map[string]string{
			"clear_screen": "\n\n\n\n\n\n\n\n\n\n", // Just newlines
			"title":        "",
			"bell":         "",
			"progress":     "dots", // Simple dots progress
		},
	}
}

// createSerialConsoleProfile creates a profile for serial consoles
func (mpd *MinimalProfileDetector) createSerialConsoleProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "serial-console",
		Description: "Serial Console",
		Priority:    15,
		Capabilities: terminal.Capabilities{
			Colors:  terminal.NoColor,
			Unicode: false,
			Mouse:   false,
			Size:    true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "",
			"bell":         "\x07",
		},
	}
}

// createLimitedTTYProfile creates a profile for limited TTY
func (mpd *MinimalProfileDetector) createLimitedTTYProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "limited-tty",
		Description: "Limited TTY",
		Priority:    20,
		Capabilities: terminal.Capabilities{
			Colors:  terminal.Basic16,
			Unicode: false,
			Mouse:   false,
			Size:    true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "",
			"bell":         "\x07",
		},
	}
}

// createHeadlessProfile creates a profile for headless environments
func (mpd *MinimalProfileDetector) createHeadlessProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "headless",
		Description: "Headless Environment",
		Priority:    30,
		Capabilities: terminal.Capabilities{
			Colors:  terminal.Basic16,
			Unicode: false,
			Mouse:   false,
			Size:    true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "",
			"bell":         "",
		},
	}
}

// createContainerProfile creates a profile for container environments
func (mpd *MinimalProfileDetector) createContainerProfile() *terminal.Profile {
	return &terminal.Profile{
		Name:        "container",
		Description: "Container Environment",
		Priority:    35,
		Capabilities: terminal.Capabilities{
			Colors:  terminal.Extended256,
			Unicode: true,
			Mouse:   false,
			Size:    true,
		},
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "",
			"bell":         "",
		},
	}
}

// createFallbackProfile creates a generic fallback profile
func (mpd *MinimalProfileDetector) createFallbackProfile() *terminal.Profile {
	caps, _ := mpd.detector.Detect(context.Background())

	// Make it conservative
	caps.Mouse = false
	caps.Hyperlinks = false
	caps.Images = false

	if caps.Colors > terminal.Basic16 {
		caps.Colors = terminal.Basic16
	}

	return &terminal.Profile{
		Name:         "fallback-minimal",
		Description:  "Conservative Fallback",
		Priority:     1,
		Capabilities: caps,
		Overrides: map[string]string{
			"clear_screen": "\x1b[2J\x1b[H",
			"title":        "",
			"bell":         "",
		},
	}
}

// ciSupportsColor checks if the CI environment supports color
func (mpd *MinimalProfileDetector) ciSupportsColor() bool {
	// GitHub Actions supports color
	if os.Getenv("GITHUB_ACTIONS") != "" {
		return true
	}

	// GitLab CI supports color
	if os.Getenv("GITLAB_CI") != "" {
		return true
	}

	// Azure DevOps supports color
	if os.Getenv("TF_BUILD") != "" {
		return true
	}

	// Jenkins with AnsiColor plugin
	if os.Getenv("JENKINS_URL") != "" && os.Getenv("ANSI_COLOR") != "" {
		return true
	}

	// Explicitly forced color
	if os.Getenv("FORCE_COLOR") != "" && os.Getenv("FORCE_COLOR") != "0" {
		return true
	}

	return false
}

// MinimalFeatures returns minimal feature sets for different scenarios
type MinimalFeatures struct {
	ProgressIndicators []string
	StatusSymbols      map[string]string
	Formatting         map[string]string
}

// GetMinimalFeatures returns appropriate features for minimal environments
func GetMinimalFeatures(profile *terminal.Profile) MinimalFeatures {
	features := MinimalFeatures{
		StatusSymbols: make(map[string]string),
		Formatting:    make(map[string]string),
	}

	switch profile.Name {
	case "ci-environment":
		features.ProgressIndicators = []string{".", "#", "="}
		features.StatusSymbols = map[string]string{
			"success": "[OK]",
			"error":   "[FAIL]",
			"warning": "[WARN]",
			"info":    "[INFO]",
		}
		features.Formatting = map[string]string{
			"bold":      "",
			"underline": "",
			"italic":    "",
		}

	case "ssh-session":
		features.ProgressIndicators = []string{".", "o", "O", "0"}
		features.StatusSymbols = map[string]string{
			"success": "[+]",
			"error":   "[-]",
			"warning": "[!]",
			"info":    "[i]",
		}
		features.Formatting = map[string]string{
			"bold":      "\x1b[1m%s\x1b[0m",
			"underline": "",
			"italic":    "",
		}

	case "dumb-terminal":
		features.ProgressIndicators = []string{"."}
		features.StatusSymbols = map[string]string{
			"success": "OK",
			"error":   "ERROR",
			"warning": "WARNING",
			"info":    "INFO",
		}
		features.Formatting = map[string]string{
			"bold":      "",
			"underline": "",
			"italic":    "",
		}

	default:
		features.ProgressIndicators = []string{".", "o", "*", "#"}
		features.StatusSymbols = map[string]string{
			"success": "[+]",
			"error":   "[-]",
			"warning": "[!]",
			"info":    "[i]",
		}
		features.Formatting = map[string]string{
			"bold":      "\x1b[1m%s\x1b[0m",
			"underline": "\x1b[4m%s\x1b[0m",
			"italic":    "",
		}
	}

	return features
}

// SafeProgressBar creates a safe progress bar for minimal environments
func SafeProgressBar(width int, percent float64, chars []string) string {
	if width <= 0 || len(chars) == 0 {
		return ""
	}

	filled := int(float64(width) * percent)
	if filled > width {
		filled = width
	}

	char := chars[0]
	if len(chars) > 1 {
		char = chars[1]
	}

	bar := strings.Repeat(char, filled)
	if filled < width {
		bar += strings.Repeat(" ", width-filled)
	}

	return "[" + bar + "]"
}

// SafeOutput creates safe output for minimal environments
func SafeOutput(message string, profile *terminal.Profile) string {
	if profile.Capabilities.Unicode {
		return message
	}

	// Replace Unicode characters with ASCII equivalents
	replacements := map[string]string{
		"✓": "[OK]",
		"✗": "[X]",
		"→": "->",
		"←": "<-",
		"•": "*",
		"…": "...",
		"─": "-",
		"│": "|",
		"┌": "+",
		"┐": "+",
		"└": "+",
		"┘": "+",
	}

	result := message
	for unicode, ascii := range replacements {
		result = strings.ReplaceAll(result, unicode, ascii)
	}

	return result
}

// DetectMinimalCapabilities performs comprehensive minimal capability detection
func DetectMinimalCapabilities() terminal.Capabilities {
	caps := terminal.Capabilities{}

	// Very conservative detection for minimal environments
	if os.Getenv("NO_COLOR") != "" {
		caps.Colors = terminal.NoColor
	} else if isVeryBasicTerminal() {
		caps.Colors = terminal.NoColor
	} else {
		caps.Colors = terminal.Basic16
	}

	// Unicode is rarely reliable in minimal environments
	caps.Unicode = false

	// Mouse support is almost never available
	caps.Mouse = false

	// Size detection might work
	caps.Size = true

	// No advanced features
	caps.Hyperlinks = false
	caps.Images = false
	caps.CursorShape = false
	caps.AlternateScreen = false

	return caps
}

// isVeryBasicTerminal checks for extremely basic terminals
func isVeryBasicTerminal() bool {
	term := strings.ToLower(os.Getenv("TERM"))
	basicTerms := []string{"", "dumb", "unknown", "vt52"}

	for _, bt := range basicTerms {
		if term == bt {
			return true
		}
	}

	return false
}
