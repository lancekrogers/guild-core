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

// MouseEvent represents a mouse event
type MouseEvent struct {
	Type      MouseEventType
	Button    MouseButton
	X, Y      int
	Modifiers Modifiers
}

// MouseEventType represents the type of mouse event
type MouseEventType int

const (
	MousePress MouseEventType = iota
	MouseRelease
	MouseMove
	MouseDrag
	MouseWheel
)

// MouseButton represents a mouse button
type MouseButton int

const (
	MouseButtonNone MouseButton = iota
	MouseButtonLeft
	MouseButtonMiddle
	MouseButtonRight
	MouseWheelUp
	MouseWheelDown
)

// Modifiers represents keyboard modifiers
type Modifiers int

const (
	ModifierNone  Modifiers = 0
	ModifierShift Modifiers = 1 << iota
	ModifierCtrl
	ModifierAlt
	ModifierMeta
)

// MouseDetector detects mouse support in terminals
type MouseDetector struct {
	platform string
	isSSH    bool
	isCI     bool
	isPTY    bool
}

// NewMouseDetector creates a new mouse detector
func NewMouseDetector() *MouseDetector {
	return &MouseDetector{
		platform: runtime.GOOS,
		isSSH:    os.Getenv("SSH_CONNECTION") != "",
		isCI:     isCI(),
		isPTY:    isTerminalPTY(),
	}
}

// Detect determines if the terminal supports mouse events
func (md *MouseDetector) Detect(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during mouse detection")
	}

	// Force disable mouse if requested
	if os.Getenv("GUILD_FORCE_NO_MOUSE") == "1" {
		return false, nil
	}

	// Force enable mouse if requested
	if os.Getenv("GUILD_FORCE_MOUSE") == "1" {
		return true, nil
	}

	// CI environments typically don't support mouse
	if md.isCI {
		return false, nil
	}

	// SSH sessions often don't properly support mouse
	if md.isSSH {
		return false, nil
	}

	// Must be a TTY
	if !md.isPTY {
		return false, nil
	}

	// Check terminal capabilities
	return md.detectMouseSupport(), nil
}

// detectMouseSupport checks for mouse support in various terminals
func (md *MouseDetector) detectMouseSupport() bool {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	// Known terminals with good mouse support
	switch termProgram {
	case "iTerm.app", "Terminal.app", "vscode", "Hyper":
		return true
	}

	// Windows Terminal
	if md.platform == "windows" && os.Getenv("WT_SESSION") != "" {
		return true
	}

	// ConEmu
	if md.platform == "windows" && os.Getenv("ConEmuPID") != "" {
		return true
	}

	// VTE-based terminals (GNOME Terminal, etc.)
	if os.Getenv("VTE_VERSION") != "" {
		return true
	}

	// Konsole
	if os.Getenv("KONSOLE_VERSION") != "" {
		return true
	}

	// Alacritty
	if os.Getenv("ALACRITTY_SOCKET") != "" || term == "alacritty" {
		return true
	}

	// Kitty
	if term == "xterm-kitty" {
		return true
	}

	// Most xterm-compatible terminals support mouse
	if strings.Contains(term, "xterm") {
		return true
	}

	// Screen and tmux can support mouse if configured
	if strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux") {
		return md.checkScreenTmuxMouse()
	}

	return false
}

// checkScreenTmuxMouse checks if screen/tmux has mouse support enabled
func (md *MouseDetector) checkScreenTmuxMouse() bool {
	// This is a simplified check - in a real implementation you'd
	// query the actual screen/tmux configuration

	// Check for tmux mouse mode
	if os.Getenv("TMUX") != "" {
		// In a full implementation, you'd check tmux show-options -g mouse
		return true // Assume modern tmux has mouse support
	}

	// Screen mouse support is less reliable
	return false
}

// MouseTracker manages mouse event tracking
type MouseTracker struct {
	enabled   bool
	tracking  MouseTrackingMode
	lastEvent *MouseEvent
	callbacks []MouseCallback
}

// MouseTrackingMode represents different mouse tracking modes
type MouseTrackingMode int

const (
	MouseTrackingNone   MouseTrackingMode = iota
	MouseTrackingNormal                   // Normal button tracking
	MouseTrackingButton                   // Button event tracking
	MouseTrackingAny                      // Track all mouse events
	MouseTrackingFocus                    // Focus event tracking
)

// MouseCallback is called when a mouse event occurs
type MouseCallback func(event MouseEvent)

// NewMouseTracker creates a new mouse tracker
func NewMouseTracker() *MouseTracker {
	return &MouseTracker{
		callbacks: make([]MouseCallback, 0),
	}
}

// Enable enables mouse tracking with the specified mode
func (mt *MouseTracker) Enable(mode MouseTrackingMode) error {
	if mt.enabled {
		return gerror.New(gerror.ErrCodeConflict, "mouse tracking already enabled", nil)
	}

	var sequence string
	switch mode {
	case MouseTrackingNormal:
		sequence = "\x1b[?1000h" // Normal button tracking
	case MouseTrackingButton:
		sequence = "\x1b[?1002h" // Button event tracking
	case MouseTrackingAny:
		sequence = "\x1b[?1003h" // Any event tracking
	case MouseTrackingFocus:
		sequence = "\x1b[?1004h" // Focus tracking
	default:
		return gerror.New(gerror.ErrCodeValidation, "invalid mouse tracking mode", nil)
	}

	// Enable SGR mouse mode for better coordinate handling
	sequence += "\x1b[?1006h"

	// Write to stdout to enable mouse tracking
	_, err := os.Stdout.WriteString(sequence)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to enable mouse tracking")
	}

	mt.enabled = true
	mt.tracking = mode

	return nil
}

// Disable disables mouse tracking
func (mt *MouseTracker) Disable() error {
	if !mt.enabled {
		return nil
	}

	// Disable all mouse tracking modes
	sequence := "\x1b[?1006l\x1b[?1004l\x1b[?1003l\x1b[?1002l\x1b[?1000l"

	_, err := os.Stdout.WriteString(sequence)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to disable mouse tracking")
	}

	mt.enabled = false
	mt.tracking = MouseTrackingNone

	return nil
}

// AddCallback adds a mouse event callback
func (mt *MouseTracker) AddCallback(callback MouseCallback) {
	mt.callbacks = append(mt.callbacks, callback)
}

// ParseMouseEvent parses a mouse event from an escape sequence
func (mt *MouseTracker) ParseMouseEvent(sequence string) (*MouseEvent, error) {
	// Handle SGR mouse format: \x1b[<button;x;y;M or m
	if strings.HasPrefix(sequence, "\x1b[<") && (strings.HasSuffix(sequence, "M") || strings.HasSuffix(sequence, "m")) {
		return mt.parseSGRMouse(sequence)
	}

	// Handle normal mouse format: \x1b[Mbxy
	if strings.HasPrefix(sequence, "\x1b[M") && len(sequence) == 6 {
		return mt.parseNormalMouse(sequence)
	}

	return nil, gerror.New(gerror.ErrCodeValidation, "unrecognized mouse sequence", nil)
}

// parseSGRMouse parses SGR mouse format
func (mt *MouseTracker) parseSGRMouse(sequence string) (*MouseEvent, error) {
	// Remove prefix and suffix
	data := sequence[3 : len(sequence)-1]
	isRelease := strings.HasSuffix(sequence, "m")

	parts := strings.Split(data, ";")
	if len(parts) != 3 {
		return nil, gerror.New(gerror.ErrCodeValidation, "invalid SGR mouse sequence", nil)
	}

	button, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid button code")
	}

	x, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid x coordinate")
	}

	y, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid y coordinate")
	}

	event := &MouseEvent{
		X: x - 1, // Convert to 0-based coordinates
		Y: y - 1,
	}

	// Parse button and modifiers
	event.Modifiers = Modifiers((button & 28) >> 2)
	buttonCode := button & 3

	// Handle wheel events
	if button&64 != 0 {
		if buttonCode == 0 {
			event.Button = MouseWheelUp
		} else {
			event.Button = MouseWheelDown
		}
		event.Type = MouseWheel
		return event, nil
	}

	// Handle motion events
	if button&32 != 0 {
		if mt.lastEvent != nil && (mt.lastEvent.Type == MousePress || mt.lastEvent.Type == MouseDrag) {
			event.Type = MouseDrag
		} else {
			event.Type = MouseMove
		}
		event.Button = MouseButton(buttonCode + 1)
		mt.lastEvent = event
		return event, nil
	}

	// Handle button press/release
	if isRelease {
		event.Type = MouseRelease
	} else {
		event.Type = MousePress
	}

	switch buttonCode {
	case 0:
		event.Button = MouseButtonLeft
	case 1:
		event.Button = MouseButtonMiddle
	case 2:
		event.Button = MouseButtonRight
	default:
		event.Button = MouseButtonNone
	}

	mt.lastEvent = event
	return event, nil
}

// parseNormalMouse parses normal mouse format
func (mt *MouseTracker) parseNormalMouse(sequence string) (*MouseEvent, error) {
	if len(sequence) != 6 {
		return nil, gerror.New(gerror.ErrCodeValidation, "invalid normal mouse sequence length", nil)
	}

	button := int(sequence[3]) - 32
	x := int(sequence[4]) - 32
	y := int(sequence[5]) - 32

	event := &MouseEvent{
		X: x - 1, // Convert to 0-based coordinates
		Y: y - 1,
	}

	// Parse modifiers
	event.Modifiers = Modifiers((button & 28) >> 2)

	// Parse button
	buttonCode := button & 3
	switch buttonCode {
	case 0:
		event.Button = MouseButtonLeft
		event.Type = MousePress
	case 1:
		event.Button = MouseButtonMiddle
		event.Type = MousePress
	case 2:
		event.Button = MouseButtonRight
		event.Type = MousePress
	case 3:
		event.Type = MouseRelease
		event.Button = MouseButtonNone
	}

	// Handle motion
	if button&32 != 0 {
		event.Type = MouseMove
	}

	// Handle wheel
	if button&64 != 0 {
		event.Type = MouseWheel
		if buttonCode == 0 {
			event.Button = MouseWheelUp
		} else {
			event.Button = MouseWheelDown
		}
	}

	mt.lastEvent = event
	return event, nil
}

// EmitEvent emits a mouse event to all callbacks
func (mt *MouseTracker) EmitEvent(event MouseEvent) {
	for _, callback := range mt.callbacks {
		callback(event)
	}
}

// String returns a string representation of the mouse event
func (me MouseEvent) String() string {
	var eventType string
	switch me.Type {
	case MousePress:
		eventType = "press"
	case MouseRelease:
		eventType = "release"
	case MouseMove:
		eventType = "move"
	case MouseDrag:
		eventType = "drag"
	case MouseWheel:
		eventType = "wheel"
	}

	var button string
	switch me.Button {
	case MouseButtonLeft:
		button = "left"
	case MouseButtonMiddle:
		button = "middle"
	case MouseButtonRight:
		button = "right"
	case MouseWheelUp:
		button = "wheel-up"
	case MouseWheelDown:
		button = "wheel-down"
	default:
		button = "none"
	}

	modifiers := []string{}
	if me.Modifiers&ModifierShift != 0 {
		modifiers = append(modifiers, "shift")
	}
	if me.Modifiers&ModifierCtrl != 0 {
		modifiers = append(modifiers, "ctrl")
	}
	if me.Modifiers&ModifierAlt != 0 {
		modifiers = append(modifiers, "alt")
	}
	if me.Modifiers&ModifierMeta != 0 {
		modifiers = append(modifiers, "meta")
	}

	modStr := ""
	if len(modifiers) > 0 {
		modStr = fmt.Sprintf(" +%s", strings.Join(modifiers, "+"))
	}

	return fmt.Sprintf("%s %s at (%d,%d)%s", eventType, button, me.X, me.Y, modStr)
}
