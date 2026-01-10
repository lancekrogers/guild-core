package features

import (
	"context"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"golang.org/x/term"
)

// InputMode represents different terminal input modes
type InputMode int

const (
	InputModeCooked InputMode = iota // Normal line-buffered input
	InputModeRaw                     // Raw character input
	InputModeCbreak                  // Character input with signal processing
)

// InputDetector detects terminal input capabilities
type InputDetector struct {
	platform string
	isSSH    bool
	isCI     bool
	isPTY    bool
}

// NewInputDetector creates a new input detector
func NewInputDetector() *InputDetector {
	return &InputDetector{
		platform: runtime.GOOS,
		isSSH:    os.Getenv("SSH_CONNECTION") != "",
		isCI:     isCI(),
		isPTY:    isTerminalPTY(),
	}
}

// Detect determines input capabilities
func (id *InputDetector) Detect(ctx context.Context) (InputCapabilities, error) {
	if err := ctx.Err(); err != nil {
		return InputCapabilities{}, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during input detection")
	}

	caps := InputCapabilities{
		SupportsRaw:       id.isPTY,
		SupportsCbreak:    id.isPTY,
		SupportsEcho:      id.isPTY,
		SupportsPaste:     id.detectPasteSupport(),
		SupportsAltScreen: id.detectAltScreenSupport(),
	}

	// CI environments usually don't support interactive input
	if id.isCI {
		caps.SupportsRaw = false
		caps.SupportsCbreak = false
		caps.SupportsPaste = false
	}

	// SSH might have limited capabilities
	if id.isSSH {
		caps.SupportsPaste = false
	}

	return caps, nil
}

// detectPasteSupport checks for bracketed paste support
func (id *InputDetector) detectPasteSupport() bool {
	if !id.isPTY {
		return false
	}

	// Most modern terminals support bracketed paste
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "iTerm.app", "vscode", "Hyper":
		return true
	}

	// Windows Terminal
	if id.platform == "windows" && os.Getenv("WT_SESSION") != "" {
		return true
	}

	// VTE-based terminals
	if os.Getenv("VTE_VERSION") != "" {
		return true
	}

	// xterm and derivatives
	term := os.Getenv("TERM")
	if strings.Contains(term, "xterm") {
		return true
	}

	return false
}

// detectAltScreenSupport checks for alternate screen buffer support
func (id *InputDetector) detectAltScreenSupport() bool {
	if !id.isPTY || id.isCI {
		return false
	}

	term := os.Getenv("TERM")
	return term != "dumb" && term != ""
}

// InputCapabilities represents input-related capabilities
type InputCapabilities struct {
	SupportsRaw       bool
	SupportsCbreak    bool
	SupportsEcho      bool
	SupportsPaste     bool
	SupportsAltScreen bool
}

// InputManager manages terminal input modes and state
type InputManager struct {
	mu           sync.RWMutex
	originalMode *term.State
	currentMode  InputMode
	stdinFd      int
	initialized  bool
}

// NewInputManager creates a new input manager
func NewInputManager() *InputManager {
	return &InputManager{
		stdinFd: int(os.Stdin.Fd()),
	}
}

// Initialize initializes the input manager and saves current state
func (im *InputManager) Initialize() error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if im.initialized {
		return gerror.New(gerror.ErrCodeConflict, "input manager already initialized", nil)
	}

	// Save the current terminal state
	if term.IsTerminal(im.stdinFd) {
		state, err := term.GetState(im.stdinFd)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get terminal state")
		}
		im.originalMode = state
		im.currentMode = InputModeCooked
	}

	im.initialized = true
	return nil
}

// SetMode sets the input mode
func (im *InputManager) SetMode(mode InputMode) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if !im.initialized {
		return gerror.New(gerror.ErrCodeInternal, "input manager not initialized", nil)
	}

	if !term.IsTerminal(im.stdinFd) {
		return gerror.New(gerror.ErrCodeValidation, "not a terminal", nil)
	}

	switch mode {
	case InputModeRaw:
		_, err := term.MakeRaw(im.stdinFd)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set raw mode")
		}
	case InputModeCbreak:
		// For cbreak mode, we want character input but keep signal processing
		// This is a simplified implementation
		_, err := term.MakeRaw(im.stdinFd)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to set cbreak mode")
		}
	case InputModeCooked:
		if im.originalMode != nil {
			err := term.Restore(im.stdinFd, im.originalMode)
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to restore cooked mode")
			}
		}
	default:
		return gerror.New(gerror.ErrCodeValidation, "unknown input mode", nil)
	}

	im.currentMode = mode
	return nil
}

// GetMode returns the current input mode
func (im *InputManager) GetMode() InputMode {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.currentMode
}

// Restore restores the original terminal state
func (im *InputManager) Restore() error {
	im.mu.Lock()
	defer im.mu.Unlock()

	if !im.initialized {
		return nil
	}

	if im.originalMode != nil && term.IsTerminal(im.stdinFd) {
		err := term.Restore(im.stdinFd, im.originalMode)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to restore terminal state")
		}
	}

	im.initialized = false
	return nil
}

// EnableBracketedPaste enables bracketed paste mode
func (im *InputManager) EnableBracketedPaste() error {
	_, err := os.Stdout.WriteString("\x1b[?2004h")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to enable bracketed paste")
	}
	return nil
}

// DisableBracketedPaste disables bracketed paste mode
func (im *InputManager) DisableBracketedPaste() error {
	_, err := os.Stdout.WriteString("\x1b[?2004l")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to disable bracketed paste")
	}
	return nil
}

// EnableAlternateScreen enables the alternate screen buffer
func (im *InputManager) EnableAlternateScreen() error {
	_, err := os.Stdout.WriteString("\x1b[?1049h")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to enable alternate screen")
	}
	return nil
}

// DisableAlternateScreen disables the alternate screen buffer
func (im *InputManager) DisableAlternateScreen() error {
	_, err := os.Stdout.WriteString("\x1b[?1049l")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to disable alternate screen")
	}
	return nil
}

// KeyEvent represents a keyboard event
type KeyEvent struct {
	Type      KeyEventType
	Key       Key
	Rune      rune
	Modifiers Modifiers
}

// KeyEventType represents the type of key event
type KeyEventType int

const (
	KeyPress KeyEventType = iota
	KeyRelease
)

// Key represents special keys
type Key int

const (
	KeyNone Key = iota
	KeyBackspace
	KeyTab
	KeyEnter
	KeyEscape
	KeySpace
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyDelete
	KeyInsert
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// InputParser parses input sequences into events
type InputParser struct {
	buffer []byte
}

// NewInputParser creates a new input parser
func NewInputParser() *InputParser {
	return &InputParser{
		buffer: make([]byte, 0, 256),
	}
}

// ParseKey parses a key event from raw input
func (ip *InputParser) ParseKey(data []byte) (*KeyEvent, error) {
	if len(data) == 0 {
		return nil, gerror.New(gerror.ErrCodeValidation, "empty input data", nil)
	}

	event := &KeyEvent{
		Type: KeyPress,
	}

	// Single byte characters
	if len(data) == 1 {
		b := data[0]

		switch b {
		case 8, 127: // Backspace
			event.Key = KeyBackspace
		case 9: // Tab
			event.Key = KeyTab
		case 13: // Enter
			event.Key = KeyEnter
		case 27: // Escape
			event.Key = KeyEscape
		case 32: // Space
			event.Key = KeySpace
		default:
			if b < 32 {
				// Control character
				event.Modifiers = ModifierCtrl
				event.Rune = rune(b + 'a' - 1)
			} else {
				event.Rune = rune(b)
			}
		}

		return event, nil
	}

	// Multi-byte sequences (escape sequences)
	if data[0] == 27 { // ESC
		return ip.parseEscapeSequence(data[1:])
	}

	// UTF-8 characters
	if len(data) > 1 {
		runes := []rune(string(data))
		if len(runes) > 0 {
			event.Rune = runes[0]
			return event, nil
		}
	}

	return nil, gerror.New(gerror.ErrCodeValidation, "unrecognized input sequence", nil)
}

// parseEscapeSequence parses ANSI escape sequences
func (ip *InputParser) parseEscapeSequence(data []byte) (*KeyEvent, error) {
	if len(data) == 0 {
		return &KeyEvent{Type: KeyPress, Key: KeyEscape}, nil
	}

	event := &KeyEvent{Type: KeyPress}

	// Function keys and arrows
	if len(data) >= 2 && data[0] == '[' {
		switch data[1] {
		case 'A':
			event.Key = KeyUp
		case 'B':
			event.Key = KeyDown
		case 'C':
			event.Key = KeyRight
		case 'D':
			event.Key = KeyLeft
		case 'H':
			event.Key = KeyHome
		case 'F':
			event.Key = KeyEnd
		default:
			// Extended sequences like [1~, [2~, etc.
			if len(data) >= 3 && data[2] == '~' {
				switch data[1] {
				case '1':
					event.Key = KeyHome
				case '2':
					event.Key = KeyInsert
				case '3':
					event.Key = KeyDelete
				case '4':
					event.Key = KeyEnd
				case '5':
					event.Key = KeyPageUp
				case '6':
					event.Key = KeyPageDown
				}
			}
		}
	}

	// Alt + key
	if len(data) == 1 && data[0] >= 32 && data[0] <= 126 {
		event.Modifiers = ModifierAlt
		event.Rune = rune(data[0])
	}

	return event, nil
}

// IsPasteStart checks if data indicates start of bracketed paste
func (ip *InputParser) IsPasteStart(data []byte) bool {
	return len(data) >= 6 && string(data[:6]) == "\x1b[200~"
}

// IsPasteEnd checks if data indicates end of bracketed paste
func (ip *InputParser) IsPasteEnd(data []byte) bool {
	return len(data) >= 6 && string(data[:6]) == "\x1b[201~"
}

// String returns a string representation of the key event
func (ke KeyEvent) String() string {
	if ke.Key != KeyNone {
		keyName := ""
		switch ke.Key {
		case KeyBackspace:
			keyName = "Backspace"
		case KeyTab:
			keyName = "Tab"
		case KeyEnter:
			keyName = "Enter"
		case KeyEscape:
			keyName = "Escape"
		case KeySpace:
			keyName = "Space"
		case KeyUp:
			keyName = "Up"
		case KeyDown:
			keyName = "Down"
		case KeyLeft:
			keyName = "Left"
		case KeyRight:
			keyName = "Right"
		case KeyHome:
			keyName = "Home"
		case KeyEnd:
			keyName = "End"
		case KeyPageUp:
			keyName = "PageUp"
		case KeyPageDown:
			keyName = "PageDown"
		case KeyDelete:
			keyName = "Delete"
		case KeyInsert:
			keyName = "Insert"
		default:
			keyName = "Unknown"
		}

		if ke.Modifiers != ModifierNone {
			return ke.Modifiers.String() + "+" + keyName
		}
		return keyName
	}

	if ke.Rune != 0 {
		if ke.Modifiers != ModifierNone {
			return ke.Modifiers.String() + "+" + string(ke.Rune)
		}
		return string(ke.Rune)
	}

	return "None"
}

// String returns a string representation of modifiers
func (m Modifiers) String() string {
	var parts []string
	if m&ModifierCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if m&ModifierAlt != 0 {
		parts = append(parts, "Alt")
	}
	if m&ModifierShift != 0 {
		parts = append(parts, "Shift")
	}
	if m&ModifierMeta != 0 {
		parts = append(parts, "Meta")
	}
	return strings.Join(parts, "+")
}
