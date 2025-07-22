package terminal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// CapabilityCheck represents a function that checks for a specific capability
type CapabilityCheck func(ctx context.Context) (bool, error)

// CapabilitySet manages a set of terminal capabilities with lazy evaluation
type CapabilitySet struct {
	mu          sync.RWMutex
	caps        Capabilities
	checks      map[string]CapabilityCheck
	evaluated   map[string]bool
	forceValues map[string]bool
}

// NewCapabilitySet creates a new capability set with default checks
func NewCapabilitySet() *CapabilitySet {
	cs := &CapabilitySet{
		checks:      make(map[string]CapabilityCheck),
		evaluated:   make(map[string]bool),
		forceValues: make(map[string]bool),
	}

	// Register default capability checks
	cs.registerDefaultChecks()

	return cs
}

// registerDefaultChecks sets up the standard capability checks
func (cs *CapabilitySet) registerDefaultChecks() {
	// Color capability checks
	cs.RegisterCheck("color.16", func(ctx context.Context) (bool, error) {
		detector := NewDetector()
		caps, err := detector.Detect(ctx)
		if err != nil {
			return false, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect capabilities")
		}
		return caps.Colors >= Basic16, nil
	})

	cs.RegisterCheck("color.256", func(ctx context.Context) (bool, error) {
		detector := NewDetector()
		caps, err := detector.Detect(ctx)
		if err != nil {
			return false, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect capabilities")
		}
		return caps.Colors >= Extended256, nil
	})

	cs.RegisterCheck("color.truecolor", func(ctx context.Context) (bool, error) {
		detector := NewDetector()
		caps, err := detector.Detect(ctx)
		if err != nil {
			return false, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect capabilities")
		}
		return caps.TrueColor, nil
	})

	// Feature capability checks
	cs.RegisterCheck("unicode", func(ctx context.Context) (bool, error) {
		detector := NewDetector()
		caps, err := detector.Detect(ctx)
		if err != nil {
			return false, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect capabilities")
		}
		return caps.Unicode, nil
	})

	cs.RegisterCheck("mouse", func(ctx context.Context) (bool, error) {
		detector := NewDetector()
		caps, err := detector.Detect(ctx)
		if err != nil {
			return false, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect capabilities")
		}
		return caps.Mouse, nil
	})

	cs.RegisterCheck("hyperlinks", func(ctx context.Context) (bool, error) {
		detector := NewDetector()
		caps, err := detector.Detect(ctx)
		if err != nil {
			return false, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to detect capabilities")
		}
		return caps.Hyperlinks, nil
	})
}

// RegisterCheck registers a new capability check
func (cs *CapabilitySet) RegisterCheck(name string, check CapabilityCheck) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.checks[name] = check
	cs.evaluated[name] = false
}

// Has checks if a capability is available
func (cs *CapabilitySet) Has(ctx context.Context, capability string) (bool, error) {
	cs.mu.RLock()

	// Check if forced
	if forced, ok := cs.forceValues[capability]; ok {
		cs.mu.RUnlock()
		return forced, nil
	}

	// Check if already evaluated
	if cs.evaluated[capability] {
		cs.mu.RUnlock()
		return cs.getCapabilityValue(capability), nil
	}

	// Get the check function
	check, ok := cs.checks[capability]
	cs.mu.RUnlock()

	if !ok {
		return false, gerror.New(gerror.ErrCodeValidation, "unknown capability: "+capability, nil)
	}

	// Evaluate the capability
	result, err := check(ctx)
	if err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeInternal, "capability check failed for: "+capability)
	}

	// Store the result
	cs.mu.Lock()
	cs.setCapabilityValue(capability, result)
	cs.evaluated[capability] = true
	cs.mu.Unlock()

	return result, nil
}

// Force sets a capability to a specific value, overriding detection
func (cs *CapabilitySet) Force(capability string, value bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.forceValues[capability] = value
}

// Reset clears all forced values and re-enables detection
func (cs *CapabilitySet) Reset() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.forceValues = make(map[string]bool)
	cs.evaluated = make(map[string]bool)
}

// getCapabilityValue retrieves a capability value (must be called with lock held)
func (cs *CapabilitySet) getCapabilityValue(capability string) bool {
	switch capability {
	case "color.16":
		return cs.caps.Colors >= Basic16
	case "color.256":
		return cs.caps.Colors >= Extended256
	case "color.truecolor":
		return cs.caps.TrueColor
	case "unicode":
		return cs.caps.Unicode
	case "mouse":
		return cs.caps.Mouse
	case "hyperlinks":
		return cs.caps.Hyperlinks
	case "images":
		return cs.caps.Images
	case "cursor.shape":
		return cs.caps.CursorShape
	case "screen.alternate":
		return cs.caps.AlternateScreen
	default:
		return false
	}
}

// setCapabilityValue sets a capability value (must be called with lock held)
func (cs *CapabilitySet) setCapabilityValue(capability string, value bool) {
	switch capability {
	case "unicode":
		cs.caps.Unicode = value
	case "mouse":
		cs.caps.Mouse = value
	case "hyperlinks":
		cs.caps.Hyperlinks = value
	case "images":
		cs.caps.Images = value
	case "cursor.shape":
		cs.caps.CursorShape = value
	case "screen.alternate":
		cs.caps.AlternateScreen = value
	}
}

// GetAll returns all current capabilities
func (cs *CapabilitySet) GetAll() Capabilities {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.caps
}

// SupportsColor checks if the terminal supports any color
func (c Capabilities) SupportsColor() bool {
	return c.Colors > NoColor
}

// SupportsExtendedColor checks if the terminal supports 256+ colors
func (c Capabilities) SupportsExtendedColor() bool {
	return c.Colors >= Extended256
}

// SupportsRichUI checks if the terminal supports rich UI elements
func (c Capabilities) SupportsRichUI() bool {
	return c.Unicode && c.SupportsExtendedColor() && c.Mouse
}

// SupportsInteractivity checks if the terminal supports interactive features
func (c Capabilities) SupportsInteractivity() bool {
	return c.Mouse && c.Size && c.AlternateScreen
}

// Summary returns a human-readable summary of capabilities
func (c Capabilities) Summary() string {
	features := []string{}

	if c.Colors > NoColor {
		features = append(features, fmt.Sprintf("colors:%s", c.Colors))
	}

	if c.Unicode {
		features = append(features, "unicode")
	}

	if c.Mouse {
		features = append(features, "mouse")
	}

	if c.Hyperlinks {
		features = append(features, "hyperlinks")
	}

	if c.Images {
		features = append(features, "images")
	}

	if len(features) == 0 {
		return "minimal"
	}

	return fmt.Sprintf("%v", features)
}

// EnvironmentOverrides applies environment variable overrides to capabilities
func (c *Capabilities) EnvironmentOverrides() {
	// Check for forced capabilities via environment
	if os.Getenv("GUILD_FORCE_UNICODE") == "1" {
		c.Unicode = true
	} else if os.Getenv("GUILD_FORCE_UNICODE") == "0" || os.Getenv("GUILD_FORCE_NO_UNICODE") == "1" {
		c.Unicode = false
	}

	if os.Getenv("GUILD_FORCE_MOUSE") == "1" {
		c.Mouse = true
	} else if os.Getenv("GUILD_FORCE_MOUSE") == "0" || os.Getenv("GUILD_FORCE_NO_MOUSE") == "1" {
		c.Mouse = false
	}

	if os.Getenv("GUILD_FORCE_COLOR") != "" {
		switch os.Getenv("GUILD_FORCE_COLOR") {
		case "0", "none":
			c.Colors = NoColor
		case "1":
			// Force at least basic color support
			if c.Colors < Basic16 {
				c.Colors = Basic16
			}
		case "16":
			c.Colors = Basic16
		case "256":
			c.Colors = Extended256
		case "16m", "truecolor", "24bit":
			c.Colors = TrueColor24Bit
			c.TrueColor = true
		}
	}

	// Check for forced true color
	if os.Getenv("GUILD_FORCE_TRUE_COLOR") == "1" {
		c.Colors = TrueColor24Bit
		c.TrueColor = true
	}
}

// RequiresLevel checks if a capability meets a minimum level
type RequiresLevel interface {
	RequiresColor(min ColorSupport) bool
	RequiresUnicode() bool
	RequiresMouse() bool
	RequiresRichUI() bool
}

// Requires returns a requirements checker for these capabilities
func (c Capabilities) Requires() RequiresLevel {
	return capabilityRequirements{c}
}

type capabilityRequirements struct {
	caps Capabilities
}

func (cr capabilityRequirements) RequiresColor(min ColorSupport) bool {
	return cr.caps.Colors >= min
}

func (cr capabilityRequirements) RequiresUnicode() bool {
	return cr.caps.Unicode
}

func (cr capabilityRequirements) RequiresMouse() bool {
	return cr.caps.Mouse
}

func (cr capabilityRequirements) RequiresRichUI() bool {
	return cr.caps.SupportsRichUI()
}

// MarshalJSON implements json.Marshaler
func (c Capabilities) MarshalJSON() ([]byte, error) {
	type Alias Capabilities
	return []byte(fmt.Sprintf(`{
		"colors": "%s",
		"unicode": %t,
		"mouse": %t,
		"size": %t,
		"trueColor": %t,
		"hyperlinks": %t,
		"images": %t,
		"cursorShape": %t,
		"alternateScreen": %t,
		"sixel": %t,
		"kitty": %t,
		"iterm2": %t
	}`, c.Colors, c.Unicode, c.Mouse, c.Size, c.TrueColor,
		c.Hyperlinks, c.Images, c.CursorShape, c.AlternateScreen,
		c.Sixel, c.Kitty, c.ITerm2)), nil
}

// Merge combines two capability sets, taking the maximum/best capability from each
func (c Capabilities) Merge(other Capabilities) Capabilities {
	merged := Capabilities{
		Colors:          c.Colors,
		Unicode:         c.Unicode || other.Unicode,
		Mouse:           c.Mouse || other.Mouse,
		Size:            c.Size || other.Size,
		TrueColor:       c.TrueColor || other.TrueColor,
		Hyperlinks:      c.Hyperlinks || other.Hyperlinks,
		Images:          c.Images || other.Images,
		CursorShape:     c.CursorShape || other.CursorShape,
		AlternateScreen: c.AlternateScreen || other.AlternateScreen,
		Sixel:           c.Sixel || other.Sixel,
		Kitty:           c.Kitty || other.Kitty,
		ITerm2:          c.ITerm2 || other.ITerm2,
	}

	// Take the better color support
	if other.Colors > merged.Colors {
		merged.Colors = other.Colors
	}

	return merged
}

// String returns a string representation of the capabilities
func (c Capabilities) String() string {
	var parts []string
	
	parts = append(parts, fmt.Sprintf("Colors: %s", c.Colors))
	
	if c.Unicode {
		parts = append(parts, "Unicode: true")
	}
	
	if c.Mouse {
		parts = append(parts, "Mouse: true")
	}
	
	if c.Size {
		parts = append(parts, "Size: true")
	}
	
	if c.TrueColor {
		parts = append(parts, "TrueColor: true")
	}
	
	if c.Hyperlinks {
		parts = append(parts, "Hyperlinks: true")
	}
	
	if c.Images {
		parts = append(parts, "Images: true")
	}
	
	if c.CursorShape {
		parts = append(parts, "CursorShape: true")
	}
	
	if c.AlternateScreen {
		parts = append(parts, "AlternateScreen: true")
	}
	
	if c.Sixel {
		parts = append(parts, "Sixel: true")
	}
	
	if c.Kitty {
		parts = append(parts, "Kitty: true")
	}
	
	if c.ITerm2 {
		parts = append(parts, "ITerm2: true")
	}
	
	return strings.Join(parts, ", ")
}

// Copy creates a deep copy of the capabilities
func (c Capabilities) Copy() Capabilities {
	return Capabilities{
		Colors:          c.Colors,
		Unicode:         c.Unicode,
		Mouse:           c.Mouse,
		Size:            c.Size,
		TrueColor:       c.TrueColor,
		Hyperlinks:      c.Hyperlinks,
		Images:          c.Images,
		CursorShape:     c.CursorShape,
		AlternateScreen: c.AlternateScreen,
		Sixel:           c.Sixel,
		Kitty:           c.Kitty,
		ITerm2:          c.ITerm2,
	}
}
