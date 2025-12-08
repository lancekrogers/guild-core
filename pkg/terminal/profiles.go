package terminal

import (
	"context"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Profile represents a terminal profile with known capabilities
type Profile struct {
	Name         string
	Description  string
	Capabilities Capabilities
	Overrides    map[string]string
	Renderer     Renderer
	Priority     int // Higher priority profiles are checked first
}

// ProfileDetector detects and manages terminal profiles
type ProfileDetector struct {
	mu       sync.RWMutex
	profiles map[string]*Profile
	detected *Profile
	detector *Detector
}

// NewProfileDetector creates a new profile detector
func NewProfileDetector() *ProfileDetector {
	pd := &ProfileDetector{
		profiles: make(map[string]*Profile),
		detector: NewDetector(),
	}

	// Register default profiles
	pd.registerDefaultProfiles()

	return pd
}

// registerDefaultProfiles registers all known terminal profiles
func (pd *ProfileDetector) registerDefaultProfiles() {
	// Windows Terminal
	pd.Register(&Profile{
		Name:        "windows-terminal",
		Description: "Windows Terminal",
		Priority:    100,
		Capabilities: Capabilities{
			Colors:          TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
	})

	// iTerm2
	pd.Register(&Profile{
		Name:        "iterm2",
		Description: "iTerm2 on macOS",
		Priority:    100,
		Capabilities: Capabilities{
			Colors:          TrueColor24Bit,
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
	})

	// macOS Terminal.app
	pd.Register(&Profile{
		Name:        "terminal-app",
		Description: "macOS Terminal.app",
		Priority:    90,
		Capabilities: Capabilities{
			Colors:          Extended256,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			CursorShape:     true,
			AlternateScreen: true,
		},
	})

	// VS Code integrated terminal
	pd.Register(&Profile{
		Name:        "vscode",
		Description: "Visual Studio Code Terminal",
		Priority:    95,
		Capabilities: Capabilities{
			Colors:          TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
	})

	// Kitty
	pd.Register(&Profile{
		Name:        "kitty",
		Description: "Kitty terminal emulator",
		Priority:    100,
		Capabilities: Capabilities{
			Colors:          TrueColor24Bit,
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
	})

	// GNOME Terminal / VTE-based terminals
	pd.Register(&Profile{
		Name:        "gnome-terminal",
		Description: "GNOME Terminal and VTE-based terminals",
		Priority:    80,
		Capabilities: Capabilities{
			Colors:          TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
	})

	// Konsole
	pd.Register(&Profile{
		Name:        "konsole",
		Description: "KDE Konsole",
		Priority:    80,
		Capabilities: Capabilities{
			Colors:          TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			CursorShape:     true,
			AlternateScreen: true,
		},
	})

	// ConEmu
	pd.Register(&Profile{
		Name:        "conemu",
		Description: "ConEmu on Windows",
		Priority:    85,
		Capabilities: Capabilities{
			Colors:          Extended256,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			CursorShape:     true,
			AlternateScreen: true,
		},
	})

	// Alacritty
	pd.Register(&Profile{
		Name:        "alacritty",
		Description: "Alacritty GPU-accelerated terminal",
		Priority:    95,
		Capabilities: Capabilities{
			Colors:          TrueColor24Bit,
			Unicode:         true,
			Mouse:           true,
			Size:            true,
			TrueColor:       true,
			Hyperlinks:      true,
			CursorShape:     true,
			AlternateScreen: true,
		},
	})

	// SSH minimal profile
	pd.Register(&Profile{
		Name:        "ssh-minimal",
		Description: "Minimal SSH session",
		Priority:    20,
		Capabilities: Capabilities{
			Colors:  Basic16,
			Unicode: false,
			Mouse:   false,
			Size:    true,
		},
	})

	// Screen/tmux profile
	pd.Register(&Profile{
		Name:        "screen-tmux",
		Description: "GNU Screen or tmux",
		Priority:    30,
		Capabilities: Capabilities{
			Colors:          Extended256,
			Unicode:         true,
			Mouse:           false,
			Size:            true,
			AlternateScreen: true,
		},
	})

	// CI environment profile
	pd.Register(&Profile{
		Name:        "ci-environment",
		Description: "Continuous Integration environment",
		Priority:    10,
		Capabilities: Capabilities{
			Colors:  NoColor,
			Unicode: false,
			Mouse:   false,
			Size:    false,
		},
	})

	// Dumb terminal profile
	pd.Register(&Profile{
		Name:        "dumb",
		Description: "Dumb terminal",
		Priority:    1,
		Capabilities: Capabilities{
			Colors:  NoColor,
			Unicode: false,
			Mouse:   false,
			Size:    false,
		},
	})
}

// Register adds a new profile to the detector
func (pd *ProfileDetector) Register(profile *Profile) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	// Set renderer if not provided
	if profile.Renderer == nil {
		profile.Renderer = SelectRenderer(profile.Capabilities)
	}

	pd.profiles[profile.Name] = profile
}

// Detect determines the current terminal profile
func (pd *ProfileDetector) Detect(ctx context.Context) (*Profile, error) {
	// Check context
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during profile detection")
	}

	pd.mu.Lock()
	defer pd.mu.Unlock()

	// Return cached result if available
	if pd.detected != nil {
		return pd.detected, nil
	}

	// Check for explicit profile override
	if override := os.Getenv("GUILD_TERMINAL_PROFILE"); override != "" {
		if profile, ok := pd.profiles[override]; ok {
			pd.detected = profile
			return profile, nil
		}
	}

	// Detect based on environment
	profile := pd.detectProfile()

	// If no specific profile matched, create a detected profile
	if profile == nil {
		caps, err := pd.detector.Detect(ctx)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect capabilities")
		}

		profile = &Profile{
			Name:         "detected",
			Description:  "Auto-detected terminal",
			Capabilities: caps,
			Renderer:     SelectRenderer(caps),
		}
	}

	pd.detected = profile
	return profile, nil
}

// detectProfile attempts to match a known profile
func (pd *ProfileDetector) detectProfile() *Profile {
	// Sort profiles by priority
	var candidates []*Profile
	for _, p := range pd.profiles {
		candidates = append(candidates, p)
	}

	// Sort by priority (highest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].Priority > candidates[i].Priority {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Check each profile in priority order
	for _, profile := range candidates {
		if pd.matchesProfile(profile) {
			return profile
		}
	}

	return nil
}

// matchesProfile checks if the current environment matches a profile
func (pd *ProfileDetector) matchesProfile(profile *Profile) bool {
	switch profile.Name {
	case "windows-terminal":
		return runtime.GOOS == "windows" && os.Getenv("WT_SESSION") != ""

	case "iterm2":
		return runtime.GOOS == "darwin" && os.Getenv("TERM_PROGRAM") == "iTerm.app"

	case "terminal-app":
		return runtime.GOOS == "darwin" && os.Getenv("TERM_PROGRAM") == "Apple_Terminal"

	case "vscode":
		return os.Getenv("TERM_PROGRAM") == "vscode"

	case "kitty":
		return os.Getenv("TERM") == "xterm-kitty"

	case "gnome-terminal":
		return os.Getenv("VTE_VERSION") != "" ||
			os.Getenv("GNOME_TERMINAL_SERVICE") != ""

	case "konsole":
		return os.Getenv("KONSOLE_VERSION") != "" ||
			os.Getenv("KONSOLE_DBUS_SERVICE") != ""

	case "conemu":
		return runtime.GOOS == "windows" && os.Getenv("ConEmuPID") != ""

	case "alacritty":
		return os.Getenv("ALACRITTY_SOCKET") != "" ||
			(os.Getenv("TERM") == "alacritty")

	case "ssh-minimal":
		return pd.detector.isSSH && !pd.hasRichTerminalFeatures()

	case "screen-tmux":
		term := os.Getenv("TERM")
		return strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux")

	case "ci-environment":
		return pd.detector.isCI

	case "dumb":
		return os.Getenv("TERM") == "dumb"

	default:
		return false
	}
}

// hasRichTerminalFeatures checks for indicators of rich terminal features
func (pd *ProfileDetector) hasRichTerminalFeatures() bool {
	// Check for true color support
	if colorTerm := os.Getenv("COLORTERM"); colorTerm == "truecolor" || colorTerm == "24bit" {
		return true
	}

	// Check for 256 color support
	if term := os.Getenv("TERM"); strings.Contains(term, "256color") {
		return true
	}

	// Check for known rich terminal programs
	termProgram := os.Getenv("TERM_PROGRAM")
	richPrograms := []string{"iTerm.app", "vscode", "Hyper"}
	for _, prog := range richPrograms {
		if termProgram == prog {
			return true
		}
	}

	return false
}

// GetProfile returns a profile by name
func (pd *ProfileDetector) GetProfile(name string) (*Profile, bool) {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	profile, ok := pd.profiles[name]
	return profile, ok
}

// ListProfiles returns all registered profiles
func (pd *ProfileDetector) ListProfiles() []*Profile {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	var profiles []*Profile
	for _, p := range pd.profiles {
		profiles = append(profiles, p)
	}

	return profiles
}

// Reset clears the detected profile cache
func (pd *ProfileDetector) Reset() {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	pd.detected = nil
}

// ApplyProfile applies a specific profile
func (pd *ProfileDetector) ApplyProfile(name string) error {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	profile, ok := pd.profiles[name]
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "unknown profile: "+name, nil)
	}

	pd.detected = profile
	return nil
}
