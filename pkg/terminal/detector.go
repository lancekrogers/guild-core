package terminal

import (
	"context"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
	"golang.org/x/term"
)

// ColorSupport represents the level of color support in the terminal
type ColorSupport int

const (
	// NoColor indicates no color support
	NoColor ColorSupport = iota
	// Basic16 supports 16 basic colors
	Basic16
	// Extended256 supports 256 colors
	Extended256
	// TrueColor24Bit supports 24-bit true color
	TrueColor24Bit
)

// Capabilities represents the detected capabilities of a terminal
type Capabilities struct {
	Colors          ColorSupport
	Unicode         bool
	Mouse           bool
	Size            bool
	TrueColor       bool
	Hyperlinks      bool
	Images          bool
	CursorShape     bool
	AlternateScreen bool
	Sixel           bool
	Kitty           bool
	ITerm2          bool
}

// Detector detects terminal capabilities using various environment cues
type Detector struct {
	mu          sync.RWMutex
	term        string
	colorTerm   string
	termProgram string
	platform    string
	isSSH       bool
	isPTY       bool
	isCI        bool
	forceColor  bool
	noColor     bool

	// Cached results
	capabilities *Capabilities
	cacheOnce    sync.Once
}

// NewDetector creates a new terminal detector with environment detection
func NewDetector() *Detector {
	return &Detector{
		term:        os.Getenv("TERM"),
		colorTerm:   os.Getenv("COLORTERM"),
		termProgram: os.Getenv("TERM_PROGRAM"),
		platform:    runtime.GOOS,
		isSSH:       os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "",
		isPTY:       term.IsTerminal(int(os.Stdout.Fd())),
		isCI:        isRunningInCI(),
		forceColor:  os.Getenv("FORCE_COLOR") != "" && os.Getenv("FORCE_COLOR") != "0",
		noColor:     os.Getenv("NO_COLOR") != "",
	}
}

// Detect performs terminal capability detection
func (d *Detector) Detect(ctx context.Context) (Capabilities, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return Capabilities{}, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during detection")
	}

	d.cacheOnce.Do(func() {
		caps := Capabilities{}

		// Detect color support
		caps.Colors = d.detectColorSupport()
		caps.TrueColor = caps.Colors == TrueColor24Bit

		// Detect Unicode support
		caps.Unicode = d.detectUnicodeSupport()

		// Detect mouse support
		caps.Mouse = d.detectMouseSupport()

		// Detect terminal size capability
		caps.Size = d.detectSizeSupport()

		// Detect hyperlink support
		caps.Hyperlinks = d.detectHyperlinkSupport()

		// Detect image protocol support
		caps.Images, caps.Sixel, caps.Kitty, caps.ITerm2 = d.detectImageSupport()

		// Detect cursor shape support
		caps.CursorShape = d.detectCursorShapeSupport()

		// Detect alternate screen support
		caps.AlternateScreen = d.detectAlternateScreenSupport()

		d.mu.Lock()
		d.capabilities = &caps
		d.mu.Unlock()
	})

	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.capabilities == nil {
		return Capabilities{}, gerror.New(gerror.ErrCodeInternal, "detection failed", nil)
	}

	return *d.capabilities, nil
}

// detectColorSupport determines the level of color support
func (d *Detector) detectColorSupport() ColorSupport {
	// Respect NO_COLOR environment variable
	if d.noColor {
		return NoColor
	}

	// Force color if requested
	if d.forceColor {
		// Assume at least 256 colors when forced
		if d.colorTerm == "truecolor" || d.colorTerm == "24bit" {
			return TrueColor24Bit
		}
		return Extended256
	}

	// Not a TTY and not forcing color
	if !d.isPTY && !d.forceColor {
		return NoColor
	}

	// Check for dumb terminal
	if d.term == "dumb" {
		return NoColor
	}

	// True color detection
	if d.colorTerm == "truecolor" || d.colorTerm == "24bit" {
		return TrueColor24Bit
	}

	// Windows Terminal supports true color
	if d.platform == "windows" && os.Getenv("WT_SESSION") != "" {
		return TrueColor24Bit
	}

	// Check for 256 color support
	if strings.Contains(d.term, "256color") || strings.Contains(d.term, "256") {
		return Extended256
	}

	// iTerm2 supports at least 256 colors
	if d.termProgram == "iTerm.app" {
		return Extended256
	}

	// Most modern terminals support at least 16 colors
	if d.term != "" && !strings.HasPrefix(d.term, "screen") {
		return Basic16
	}

	// Conservative default for screen/tmux
	if strings.HasPrefix(d.term, "screen") {
		return Basic16
	}

	return NoColor
}

// detectUnicodeSupport checks if the terminal supports Unicode
func (d *Detector) detectUnicodeSupport() bool {
	// Windows has special handling
	if d.platform == "windows" {
		return d.detectWindowsUnicode()
	}

	// Check locale settings
	lang := os.Getenv("LANG")
	lcAll := os.Getenv("LC_ALL")
	lcCtype := os.Getenv("LC_CTYPE")

	// Check if any locale setting indicates UTF-8
	for _, env := range []string{lcAll, lcCtype, lang} {
		if strings.Contains(strings.ToLower(env), "utf-8") ||
			strings.Contains(strings.ToLower(env), "utf8") {
			return true
		}
	}

	// Modern macOS terminals support Unicode
	if d.platform == "darwin" {
		return true
	}

	// SSH sessions might not properly propagate locale
	if d.isSSH {
		// Conservative approach for SSH
		return false
	}

	return false
}

// detectWindowsUnicode checks Unicode support on Windows
func (d *Detector) detectWindowsUnicode() bool {
	// Windows Terminal supports Unicode
	if os.Getenv("WT_SESSION") != "" {
		return true
	}

	// ConEmu supports Unicode
	if os.Getenv("ConEmuPID") != "" {
		return true
	}

	// Check Windows version for native Unicode support
	// Windows 10 1903+ has better Unicode support
	// This is a simplified check - in production you'd use Windows API
	return false
}

// detectMouseSupport checks if the terminal supports mouse events
func (d *Detector) detectMouseSupport() bool {
	// CI environments typically don't support mouse
	if d.isCI {
		return false
	}

	// SSH sessions often don't properly support mouse
	if d.isSSH {
		return false
	}

	// Not a TTY
	if !d.isPTY {
		return false
	}

	// Known terminals with mouse support
	switch d.termProgram {
	case "iTerm.app", "Terminal.app", "Hyper", "vscode":
		return true
	}

	// Windows Terminal supports mouse
	if d.platform == "windows" && os.Getenv("WT_SESSION") != "" {
		return true
	}

	// Most xterm-compatible terminals support mouse
	if strings.Contains(d.term, "xterm") {
		return true
	}

	return false
}

// detectSizeSupport checks if we can detect terminal size
func (d *Detector) detectSizeSupport() bool {
	// Must be a TTY to get size
	if !d.isPTY {
		return false
	}

	// Try to get terminal size
	_, _, err := term.GetSize(int(os.Stdout.Fd()))
	return err == nil
}

// detectHyperlinkSupport checks for OSC 8 hyperlink support
func (d *Detector) detectHyperlinkSupport() bool {
	// Known terminals with hyperlink support
	switch d.termProgram {
	case "iTerm.app", "Hyper", "vscode":
		return true
	}

	// Windows Terminal supports hyperlinks
	if d.platform == "windows" && os.Getenv("WT_SESSION") != "" {
		return true
	}

	// Recent GNOME Terminal versions
	if os.Getenv("VTE_VERSION") != "" {
		return true
	}

	return false
}

// detectImageSupport checks for various image protocols
func (d *Detector) detectImageSupport() (images, sixel, kitty, iterm2 bool) {
	// iTerm2 image protocol
	if d.termProgram == "iTerm.app" {
		return true, false, false, true
	}

	// Kitty graphics protocol
	if d.term == "xterm-kitty" {
		return true, false, true, false
	}

	// Sixel support (mlterm, xterm with sixel)
	if strings.Contains(d.term, "mlterm") || os.Getenv("SIXEL_SUPPORT") != "" {
		return true, true, false, false
	}

	return false, false, false, false
}

// detectCursorShapeSupport checks if terminal supports cursor shape changes
func (d *Detector) detectCursorShapeSupport() bool {
	// Most modern terminals support cursor shape
	switch d.termProgram {
	case "iTerm.app", "Terminal.app", "Hyper", "vscode":
		return true
	}

	// Windows Terminal
	if d.platform == "windows" && os.Getenv("WT_SESSION") != "" {
		return true
	}

	// VTE-based terminals (GNOME Terminal, etc.)
	if os.Getenv("VTE_VERSION") != "" {
		return true
	}

	return strings.Contains(d.term, "xterm")
}

// detectAlternateScreenSupport checks for alternate screen buffer support
func (d *Detector) detectAlternateScreenSupport() bool {
	// Not in CI
	if d.isCI {
		return false
	}

	// Most terminals except very basic ones support alternate screen
	return d.term != "dumb" && d.term != "" && d.isPTY
}

// isRunningInCI detects if we're running in a CI environment
func isRunningInCI() bool {
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

// String returns a string representation of color support
func (cs ColorSupport) String() string {
	switch cs {
	case NoColor:
		return "none"
	case Basic16:
		return "16"
	case Extended256:
		return "256"
	case TrueColor24Bit:
		return "16m"
	default:
		return "unknown"
	}
}
