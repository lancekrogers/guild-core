package terminal

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Position represents a cursor position in the terminal
type Position struct {
	X, Y int
}

// Cell represents a single terminal cell
type Cell struct {
	Char       rune
	Foreground string
	Background string
	Bold       bool
	Italic     bool
	Underline  bool
}

// Emulator provides terminal emulation for testing
type Emulator struct {
	mu           sync.RWMutex
	width        int
	height       int
	buffer       [][]Cell
	cursor       Position
	savedCursor  Position
	profile      *Profile
	altScreen    bool
	altBuffer    [][]Cell
	scrollTop    int
	scrollBottom int
	insertMode   bool
	wrapMode     bool
	originMode   bool
	sequences    []string
	output       bytes.Buffer
}

// NewEmulator creates a new terminal emulator
func NewEmulator(width, height int, profile *Profile) *Emulator {
	if profile == nil {
		// Default minimal profile for testing
		profile = &Profile{
			Name:        "test-emulator",
			Description: "Test Terminal Emulator",
			Capabilities: Capabilities{
				Colors:          Extended256,
				Unicode:         true,
				Mouse:           true,
				Size:            true,
				AlternateScreen: true,
			},
		}
	}

	e := &Emulator{
		width:        width,
		height:       height,
		profile:      profile,
		scrollBottom: height - 1,
		wrapMode:     true,
	}

	e.initializeBuffer()
	return e
}

// initializeBuffer initializes the terminal buffer
func (e *Emulator) initializeBuffer() {
	e.buffer = make([][]Cell, e.height)
	for y := 0; y < e.height; y++ {
		e.buffer[y] = make([]Cell, e.width)
		for x := 0; x < e.width; x++ {
			e.buffer[y][x] = Cell{Char: ' '}
		}
	}
}

// Write implements io.Writer interface for the emulator
func (e *Emulator) Write(p []byte) (n int, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.output.Write(p)
	return e.parseInput(p)
}

// parseInput parses and processes input data
func (e *Emulator) parseInput(data []byte) (int, error) {
	input := string(data)
	processed := 0

	for i := 0; i < len(input); i++ {
		char := rune(input[i])

		// Handle ANSI escape sequences
		if char == '\x1b' && i+1 < len(input) {
			seqLen, err := e.parseEscapeSequence(input[i:])
			if err != nil {
				// If parse fails, treat as regular character
				e.writeChar(char)
				processed++
			} else {
				e.sequences = append(e.sequences, input[i:i+seqLen])
				i += seqLen - 1
				processed += seqLen
			}
		} else {
			// Regular character
			switch char {
			case '\r':
				e.cursor.X = 0
			case '\n':
				e.newline()
			case '\b':
				if e.cursor.X > 0 {
					e.cursor.X--
				}
			case '\t':
				e.tab()
			case '\x07': // Bell
				// Ignore bell in emulator
			default:
				e.writeChar(char)
			}
			processed++
		}
	}

	return processed, nil
}

// parseEscapeSequence parses ANSI escape sequences
func (e *Emulator) parseEscapeSequence(input string) (int, error) {
	if len(input) < 2 {
		return 0, gerror.New(gerror.ErrCodeInvalidFormat, "incomplete escape sequence", nil)
	}

	switch input[1] {
	case '[':
		return e.parseCSI(input)
	case ']':
		return e.parseOSC(input)
	case '(':
		return e.parseCharsetSelect(input)
	case 'M':
		// Reverse index
		e.reverseIndex()
		return 2, nil
	case 'D':
		// Index
		e.index()
		return 2, nil
	case 'E':
		// Next line
		e.nextLine()
		return 2, nil
	case 'c':
		// Reset
		e.reset()
		return 2, nil
	case '7':
		// Save cursor
		e.saveCursor()
		return 2, nil
	case '8':
		// Restore cursor
		e.restoreCursor()
		return 2, nil
	default:
		return 0, gerror.New(gerror.ErrCodeValidation, "unknown escape sequence", nil)
	}
}

// parseCSI parses Control Sequence Introducer sequences
func (e *Emulator) parseCSI(input string) (int, error) {
	if len(input) < 3 {
		return 0, gerror.New(gerror.ErrCodeInvalidFormat, "incomplete CSI sequence", nil)
	}

	// Find the end of the CSI sequence
	end := 2
	for end < len(input) {
		char := input[end]
		if (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || char == '@' {
			break
		}
		end++
	}

	if end >= len(input) {
		return 0, gerror.New(gerror.ErrCodeInvalidFormat, "unterminated CSI sequence", nil)
	}

	params := strings.TrimSuffix(input[2:end], string(input[end]))
	command := input[end]

	e.executeCSI(command, params)
	return end + 1, nil
}

// executeCSI executes CSI commands
func (e *Emulator) executeCSI(command byte, params string) {
	switch command {
	case 'A': // Cursor up
		n := e.parseParam(params, 1)
		e.cursorUp(n)
	case 'B': // Cursor down
		n := e.parseParam(params, 1)
		e.cursorDown(n)
	case 'C': // Cursor forward
		n := e.parseParam(params, 1)
		e.cursorForward(n)
	case 'D': // Cursor backward
		n := e.parseParam(params, 1)
		e.cursorBackward(n)
	case 'H', 'f': // Cursor position
		e.setCursorPosition(params)
	case 'J': // Erase in display
		n := e.parseParam(params, 0)
		e.eraseInDisplay(n)
	case 'K': // Erase in line
		n := e.parseParam(params, 0)
		e.eraseInLine(n)
	case 'S': // Scroll up
		n := e.parseParam(params, 1)
		e.scrollUp(n)
	case 'T': // Scroll down
		n := e.parseParam(params, 1)
		e.scrollDown(n)
	case 'm': // Select Graphic Rendition
		e.setSGR(params)
	case 'r': // Set scrolling region
		e.setScrollingRegion(params)
	case 's': // Save cursor position
		e.saveCursor()
	case 'u': // Restore cursor position
		e.restoreCursor()
	}
}

// parseOSC parses Operating System Command sequences
func (e *Emulator) parseOSC(input string) (int, error) {
	// Find string terminator
	end := strings.Index(input[2:], "\x07")
	if end == -1 {
		end = strings.Index(input[2:], "\x1b\\")
		if end == -1 {
			return 0, gerror.New(gerror.ErrCodeInvalidFormat, "unterminated OSC sequence", nil)
		}
		end += 4 // Include terminator
	} else {
		end += 3 // Include terminator
	}

	// OSC sequences like title setting are ignored in emulator
	return end, nil
}

// parseCharsetSelect parses charset selection sequences
func (e *Emulator) parseCharsetSelect(input string) (int, error) {
	if len(input) < 3 {
		return 0, gerror.New(gerror.ErrCodeInvalidFormat, "incomplete charset sequence", nil)
	}
	// Ignore charset selection in emulator
	return 3, nil
}

// parseParam parses numeric parameters from CSI sequences
func (e *Emulator) parseParam(params string, defaultValue int) int {
	if params == "" {
		return defaultValue
	}

	// Simple parsing - take first number
	parts := strings.Split(params, ";")
	if len(parts) > 0 && parts[0] != "" {
		var n int
		fmt.Sscanf(parts[0], "%d", &n)
		if n == 0 {
			return defaultValue
		}
		return n
	}

	return defaultValue
}

// writeChar writes a character at the current cursor position
func (e *Emulator) writeChar(char rune) {
	if e.cursor.Y >= 0 && e.cursor.Y < e.height &&
		e.cursor.X >= 0 && e.cursor.X < e.width {
		e.buffer[e.cursor.Y][e.cursor.X] = Cell{Char: char}
	}

	e.cursor.X++
	if e.cursor.X >= e.width {
		if e.wrapMode {
			e.cursor.X = 0
			e.newline()
		} else {
			e.cursor.X = e.width - 1
		}
	}
}

// newline moves cursor to next line
func (e *Emulator) newline() {
	e.cursor.Y++
	if e.cursor.Y > e.scrollBottom {
		e.scrollUp(1)
		e.cursor.Y = e.scrollBottom
	}
}

// tab moves cursor to next tab stop
func (e *Emulator) tab() {
	e.cursor.X = ((e.cursor.X / 8) + 1) * 8
	if e.cursor.X >= e.width {
		e.cursor.X = e.width - 1
	}
}

// Cursor movement methods
func (e *Emulator) cursorUp(n int) {
	e.cursor.Y -= n
	if e.cursor.Y < e.scrollTop {
		e.cursor.Y = e.scrollTop
	}
}

func (e *Emulator) cursorDown(n int) {
	e.cursor.Y += n
	if e.cursor.Y > e.scrollBottom {
		e.cursor.Y = e.scrollBottom
	}
}

func (e *Emulator) cursorForward(n int) {
	e.cursor.X += n
	if e.cursor.X >= e.width {
		e.cursor.X = e.width - 1
	}
}

func (e *Emulator) cursorBackward(n int) {
	e.cursor.X -= n
	if e.cursor.X < 0 {
		e.cursor.X = 0
	}
}

func (e *Emulator) setCursorPosition(params string) {
	parts := strings.Split(params, ";")
	row, col := 1, 1

	if len(parts) >= 1 && parts[0] != "" {
		fmt.Sscanf(parts[0], "%d", &row)
	}
	if len(parts) >= 2 && parts[1] != "" {
		fmt.Sscanf(parts[1], "%d", &col)
	}

	e.cursor.Y = row - 1
	e.cursor.X = col - 1

	// Clamp to bounds
	if e.cursor.Y < 0 {
		e.cursor.Y = 0
	}
	if e.cursor.Y >= e.height {
		e.cursor.Y = e.height - 1
	}
	if e.cursor.X < 0 {
		e.cursor.X = 0
	}
	if e.cursor.X >= e.width {
		e.cursor.X = e.width - 1
	}
}

// Erase methods
func (e *Emulator) eraseInDisplay(mode int) {
	switch mode {
	case 0: // Erase from cursor to end of screen
		e.eraseInLine(0)
		for y := e.cursor.Y + 1; y < e.height; y++ {
			for x := 0; x < e.width; x++ {
				e.buffer[y][x] = Cell{Char: ' '}
			}
		}
	case 1: // Erase from beginning of screen to cursor
		for y := 0; y < e.cursor.Y; y++ {
			for x := 0; x < e.width; x++ {
				e.buffer[y][x] = Cell{Char: ' '}
			}
		}
		e.eraseInLine(1)
	case 2: // Erase entire screen
		e.clear()
	}
}

func (e *Emulator) eraseInLine(mode int) {
	if e.cursor.Y < 0 || e.cursor.Y >= e.height {
		return
	}

	switch mode {
	case 0: // Erase from cursor to end of line
		for x := e.cursor.X; x < e.width; x++ {
			e.buffer[e.cursor.Y][x] = Cell{Char: ' '}
		}
	case 1: // Erase from beginning of line to cursor
		for x := 0; x <= e.cursor.X && x < e.width; x++ {
			e.buffer[e.cursor.Y][x] = Cell{Char: ' '}
		}
	case 2: // Erase entire line
		for x := 0; x < e.width; x++ {
			e.buffer[e.cursor.Y][x] = Cell{Char: ' '}
		}
	}
}

// Scroll methods
func (e *Emulator) scrollUp(n int) {
	for i := 0; i < n; i++ {
		// Move lines up
		for y := e.scrollTop; y < e.scrollBottom; y++ {
			e.buffer[y] = e.buffer[y+1]
		}
		// Clear bottom line
		for x := 0; x < e.width; x++ {
			e.buffer[e.scrollBottom][x] = Cell{Char: ' '}
		}
	}
}

func (e *Emulator) scrollDown(n int) {
	for i := 0; i < n; i++ {
		// Move lines down
		for y := e.scrollBottom; y > e.scrollTop; y-- {
			e.buffer[y] = e.buffer[y-1]
		}
		// Clear top line
		for x := 0; x < e.width; x++ {
			e.buffer[e.scrollTop][x] = Cell{Char: ' '}
		}
	}
}

// Other terminal operations
func (e *Emulator) reverseIndex() {
	if e.cursor.Y <= e.scrollTop {
		e.scrollDown(1)
	} else {
		e.cursor.Y--
	}
}

func (e *Emulator) index() {
	if e.cursor.Y >= e.scrollBottom {
		e.scrollUp(1)
	} else {
		e.cursor.Y++
	}
}

func (e *Emulator) nextLine() {
	e.cursor.X = 0
	e.index()
}

func (e *Emulator) reset() {
	e.clear()
	e.cursor = Position{0, 0}
	e.scrollTop = 0
	e.scrollBottom = e.height - 1
}

func (e *Emulator) clear() {
	for y := 0; y < e.height; y++ {
		for x := 0; x < e.width; x++ {
			e.buffer[y][x] = Cell{Char: ' '}
		}
	}
}

func (e *Emulator) saveCursor() {
	e.savedCursor = e.cursor
}

func (e *Emulator) restoreCursor() {
	e.cursor = e.savedCursor
}

func (e *Emulator) setSGR(params string) {
	// Simplified SGR handling - just parse for testing
	// In a full implementation, this would set colors and formatting
}

func (e *Emulator) setScrollingRegion(params string) {
	parts := strings.Split(params, ";")
	top, bottom := 1, e.height

	if len(parts) >= 1 && parts[0] != "" {
		fmt.Sscanf(parts[0], "%d", &top)
	}
	if len(parts) >= 2 && parts[1] != "" {
		fmt.Sscanf(parts[1], "%d", &bottom)
	}

	e.scrollTop = top - 1
	e.scrollBottom = bottom - 1

	// Clamp to bounds
	if e.scrollTop < 0 {
		e.scrollTop = 0
	}
	if e.scrollBottom >= e.height {
		e.scrollBottom = e.height - 1
	}
}

// Screenshot returns a visual representation of the terminal buffer
func (e *Emulator) Screenshot() string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result strings.Builder
	for y := 0; y < e.height; y++ {
		for x := 0; x < e.width; x++ {
			result.WriteRune(e.buffer[y][x].Char)
		}
		if y < e.height-1 {
			result.WriteRune('\n')
		}
	}
	return result.String()
}

// GetCell returns the cell at the specified position
func (e *Emulator) GetCell(x, y int) (Cell, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if y >= 0 && y < e.height && x >= 0 && x < e.width {
		return e.buffer[y][x], true
	}
	return Cell{}, false
}

// GetLine returns a line as a string
func (e *Emulator) GetLine(y int) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if y < 0 || y >= e.height {
		return ""
	}

	var line strings.Builder
	for x := 0; x < e.width; x++ {
		line.WriteRune(e.buffer[y][x].Char)
	}
	return strings.TrimRight(line.String(), " ")
}

// GetCursor returns the current cursor position
func (e *Emulator) GetCursor() Position {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cursor
}

// GetSize returns the terminal size
func (e *Emulator) GetSize() (int, int) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.width, e.height
}

// GetSequences returns all captured escape sequences
func (e *Emulator) GetSequences() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	sequences := make([]string, len(e.sequences))
	copy(sequences, e.sequences)
	return sequences
}

// ClearSequences clears the captured sequences
func (e *Emulator) ClearSequences() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sequences = e.sequences[:0]
}

// TestHelper provides helper methods for testing
type TestHelper struct {
	emulator *Emulator
}

// NewTestHelper creates a test helper with an emulator
func NewTestHelper(width, height int, profile *Profile) *TestHelper {
	return &TestHelper{
		emulator: NewEmulator(width, height, profile),
	}
}

// Render renders content and returns a screenshot
func (th *TestHelper) Render(content string) string {
	th.emulator.Write([]byte(content))
	return th.emulator.Screenshot()
}

// ExpectLine checks if a line contains expected content
func (th *TestHelper) ExpectLine(lineNum int, expected string) bool {
	actual := th.emulator.GetLine(lineNum)
	return strings.Contains(actual, expected)
}

// ExpectSequence checks if a specific escape sequence was captured
func (th *TestHelper) ExpectSequence(pattern string) bool {
	sequences := th.emulator.GetSequences()
	regex := regexp.MustCompile(pattern)

	for _, seq := range sequences {
		if regex.MatchString(seq) {
			return true
		}
	}
	return false
}

// Reset resets the emulator state
func (th *TestHelper) Reset() {
	th.emulator.reset()
	th.emulator.ClearSequences()
}

// CompatibilityTester tests terminal compatibility
type CompatibilityTester struct {
	profiles map[string]*Profile
	results  map[string]CompatibilityResult
}

// CompatibilityResult represents the result of a compatibility test
type CompatibilityResult struct {
	Profile    string
	Passed     int
	Failed     int
	Errors     []string
	Screenshot string
	Duration   time.Duration
}

// NewCompatibilityTester creates a new compatibility tester
func NewCompatibilityTester() *CompatibilityTester {
	return &CompatibilityTester{
		profiles: make(map[string]*Profile),
		results:  make(map[string]CompatibilityResult),
	}
}

// AddProfile adds a profile to test
func (ct *CompatibilityTester) AddProfile(profile *Profile) {
	ct.profiles[profile.Name] = profile
}

// TestRender tests rendering across all profiles
func (ct *CompatibilityTester) TestRender(ctx context.Context, content string, validators []func(string) bool) error {
	for name, profile := range ct.profiles {
		start := time.Now()

		helper := NewTestHelper(80, 24, profile)
		screenshot := helper.Render(content)

		result := CompatibilityResult{
			Profile:    name,
			Screenshot: screenshot,
			Duration:   time.Since(start),
		}

		// Run validators
		for _, validator := range validators {
			if validator(screenshot) {
				result.Passed++
			} else {
				result.Failed++
				result.Errors = append(result.Errors, "Validation failed")
			}
		}

		ct.results[name] = result

		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeCancelled, "compatibility testing cancelled")
		}
	}

	return nil
}

// GetResults returns all test results
func (ct *CompatibilityTester) GetResults() map[string]CompatibilityResult {
	return ct.results
}
