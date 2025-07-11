package features

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// ColorSupport represents the level of color support
type ColorSupport int

const (
	NoColor ColorSupport = iota
	Basic16
	Extended256
	TrueColor24Bit
)

// ColorDetector detects terminal color capabilities
type ColorDetector struct {
	term      string
	colorTerm string
	platform  string
	isSSH     bool
	isPTY     bool
	isCI      bool
}

// NewColorDetector creates a new color detector
func NewColorDetector() *ColorDetector {
	return &ColorDetector{
		term:      os.Getenv("TERM"),
		colorTerm: os.Getenv("COLORTERM"),
		platform:  runtime.GOOS,
		isSSH:     os.Getenv("SSH_CONNECTION") != "",
		isPTY:     isTerminalPTY(),
		isCI:      isCI(),
	}
}

// Detect determines the color support level
func (cd *ColorDetector) Detect(ctx context.Context) (ColorSupport, error) {
	if err := ctx.Err(); err != nil {
		return NoColor, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during color detection")
	}

	// Respect NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return NoColor, nil
	}

	// Force color if requested
	if forceColor := os.Getenv("FORCE_COLOR"); forceColor != "" {
		return cd.parseForceColor(forceColor), nil
	}

	// Not a TTY and not forcing color
	if !cd.isPTY && os.Getenv("FORCE_COLOR") == "" {
		return NoColor, nil
	}

	// Check for dumb terminal
	if cd.term == "dumb" {
		return NoColor, nil
	}

	// CI environments often support color
	if cd.isCI {
		return cd.detectCIColor(), nil
	}

	// True color detection
	if cd.hasTrueColorSupport() {
		return TrueColor24Bit, nil
	}

	// 256 color detection
	if cd.has256ColorSupport() {
		return Extended256, nil
	}

	// Basic color detection
	if cd.hasBasicColorSupport() {
		return Basic16, nil
	}

	return NoColor, nil
}

// parseForceColor parses the FORCE_COLOR environment variable
func (cd *ColorDetector) parseForceColor(value string) ColorSupport {
	switch value {
	case "0", "false":
		return NoColor
	case "1", "true":
		return Basic16
	case "2":
		return Extended256
	case "3":
		return TrueColor24Bit
	default:
		// Default to 256 colors if forcing
		return Extended256
	}
}

// hasTrueColorSupport checks for 24-bit color support
func (cd *ColorDetector) hasTrueColorSupport() bool {
	// Check COLORTERM environment variable
	if cd.colorTerm == "truecolor" || cd.colorTerm == "24bit" {
		return true
	}

	// Windows Terminal supports true color
	if cd.platform == "windows" && os.Getenv("WT_SESSION") != "" {
		return true
	}

	// Check for known terminals with true color support
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "iTerm.app", "vscode", "Hyper":
		return true
	}

	// Check TERM values that indicate true color
	trueColorTerms := []string{
		"xterm-kitty",
		"alacritty",
		"konsole",
		"gnome-terminal",
	}

	for _, term := range trueColorTerms {
		if strings.Contains(cd.term, term) {
			return true
		}
	}

	// VTE version check for true color
	if vteVersion := os.Getenv("VTE_VERSION"); vteVersion != "" {
		if version, err := strconv.Atoi(vteVersion); err == nil && version >= 3600 {
			return true
		}
	}

	return false
}

// has256ColorSupport checks for 256 color support
func (cd *ColorDetector) has256ColorSupport() bool {
	// Direct TERM indication
	if strings.Contains(cd.term, "256color") || strings.Contains(cd.term, "256") {
		return true
	}

	// Screen and tmux often support 256 colors
	if strings.HasPrefix(cd.term, "screen") || strings.HasPrefix(cd.term, "tmux") {
		return true
	}

	// Most modern xterm variants support 256 colors
	if strings.Contains(cd.term, "xterm") && cd.term != "xterm" {
		return true
	}

	// Check for known 256-color terminals
	color256Terms := []string{
		"rxvt-unicode",
		"konsole",
		"gnome-terminal",
		"st-256color",
	}

	for _, term := range color256Terms {
		if strings.Contains(cd.term, term) {
			return true
		}
	}

	return false
}

// hasBasicColorSupport checks for basic 16 color support
func (cd *ColorDetector) hasBasicColorSupport() bool {
	// Most terminals support basic colors unless explicitly disabled
	if cd.term == "" || cd.term == "dumb" {
		return false
	}

	// Common color-supporting terminals
	colorTerms := []string{
		"xterm", "screen", "tmux", "linux", "cygwin", "putty",
		"rxvt", "konsole", "gnome", "terminal",
	}

	termLower := strings.ToLower(cd.term)
	for _, colorTerm := range colorTerms {
		if strings.Contains(termLower, colorTerm) {
			return true
		}
	}

	return false
}

// detectCIColor detects color support in CI environments
func (cd *ColorDetector) detectCIColor() ColorSupport {
	// GitHub Actions supports true color
	if os.Getenv("GITHUB_ACTIONS") != "" {
		return TrueColor24Bit
	}

	// GitLab CI supports 256 colors
	if os.Getenv("GITLAB_CI") != "" {
		return Extended256
	}

	// Most modern CI systems support at least 256 colors
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "TRAVIS", "CIRCLECI",
		"BUILDKITE", "DRONE", "JENKINS_URL",
	}

	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return Extended256
		}
	}

	return Basic16
}

// ColorPalette represents a color palette
type ColorPalette struct {
	Colors map[string]string
	Reset  string
}

// NewColorPalette creates a color palette based on support level
func NewColorPalette(support ColorSupport) *ColorPalette {
	palette := &ColorPalette{
		Colors: make(map[string]string),
		Reset:  "\x1b[0m",
	}

	switch support {
	case TrueColor24Bit:
		palette.Colors = map[string]string{
			"black":   "\x1b[38;2;0;0;0m",
			"red":     "\x1b[38;2;255;101;101m",
			"green":   "\x1b[38;2;142;215;108m",
			"yellow":  "\x1b[38;2;255;221;89m",
			"blue":    "\x1b[38;2;88;166;255m",
			"magenta": "\x1b[38;2;255;121;198m",
			"cyan":    "\x1b[38;2;120;219;255m",
			"white":   "\x1b[38;2;255;255;255m",
			"gray":    "\x1b[38;2;128;128;128m",
			"orange":  "\x1b[38;2;255;184;108m",
		}

	case Extended256:
		palette.Colors = map[string]string{
			"black":   "\x1b[38;5;0m",
			"red":     "\x1b[38;5;196m",
			"green":   "\x1b[38;5;82m",
			"yellow":  "\x1b[38;5;226m",
			"blue":    "\x1b[38;5;33m",
			"magenta": "\x1b[38;5;201m",
			"cyan":    "\x1b[38;5;51m",
			"white":   "\x1b[38;5;15m",
			"gray":    "\x1b[38;5;244m",
			"orange":  "\x1b[38;5;214m",
		}

	case Basic16:
		palette.Colors = map[string]string{
			"black":   "\x1b[30m",
			"red":     "\x1b[31m",
			"green":   "\x1b[32m",
			"yellow":  "\x1b[33m",
			"blue":    "\x1b[34m",
			"magenta": "\x1b[35m",
			"cyan":    "\x1b[36m",
			"white":   "\x1b[37m",
			"gray":    "\x1b[90m",
			"orange":  "\x1b[33m", // Fallback to yellow
		}

	default: // NoColor
		palette.Reset = ""
	}

	return palette
}

// Color applies a color to text
func (cp *ColorPalette) Color(color, text string) string {
	if colorCode, ok := cp.Colors[color]; ok {
		return colorCode + text + cp.Reset
	}
	return text
}

// Bold applies bold formatting
func (cp *ColorPalette) Bold(text string) string {
	if cp.Reset == "" {
		return text
	}
	return "\x1b[1m" + text + "\x1b[22m"
}

// Dim applies dim formatting
func (cp *ColorPalette) Dim(text string) string {
	if cp.Reset == "" {
		return text
	}
	return "\x1b[2m" + text + "\x1b[22m"
}

// Italic applies italic formatting
func (cp *ColorPalette) Italic(text string) string {
	if cp.Reset == "" {
		return text
	}
	return "\x1b[3m" + text + "\x1b[23m"
}

// Underline applies underline formatting
func (cp *ColorPalette) Underline(text string) string {
	if cp.Reset == "" {
		return text
	}
	return "\x1b[4m" + text + "\x1b[24m"
}

// Background applies background color
func (cp *ColorPalette) Background(color, text string) string {
	if cp.Reset == "" {
		return text
	}

	var bgCode string
	switch color {
	case "black":
		bgCode = "\x1b[40m"
	case "red":
		bgCode = "\x1b[41m"
	case "green":
		bgCode = "\x1b[42m"
	case "yellow":
		bgCode = "\x1b[43m"
	case "blue":
		bgCode = "\x1b[44m"
	case "magenta":
		bgCode = "\x1b[45m"
	case "cyan":
		bgCode = "\x1b[46m"
	case "white":
		bgCode = "\x1b[47m"
	default:
		return text
	}

	return bgCode + text + cp.Reset
}

// RGB creates a true color RGB sequence
func (cp *ColorPalette) RGB(r, g, b int, text string) string {
	if cp.Reset == "" {
		return text
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s%s", r, g, b, text, cp.Reset)
}

// isTerminalPTY checks if stdout is a terminal
func isTerminalPTY() bool {
	// This would use golang.org/x/term in a real implementation
	// For now, simplified check
	return os.Getenv("TERM") != ""
}

// isCI checks if running in a CI environment
func isCI() bool {
	ciVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_NUMBER",
		"JENKINS_URL", "TRAVIS", "CIRCLECI", "GITHUB_ACTIONS",
		"GITLAB_CI", "BUILDKITE", "DRONE", "TEAMCITY_VERSION",
	}

	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}

	return false
}
