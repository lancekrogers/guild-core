package terminal

import (
	"fmt"
	"strings"
	"sync"
)

// BoxStyle represents different box drawing styles
type BoxStyle int

const (
	// BoxStyleSingle uses single-line box characters
	BoxStyleSingle BoxStyle = iota
	// BoxStyleDouble uses double-line box characters
	BoxStyleDouble
	// BoxStyleRounded uses rounded corner box characters
	BoxStyleRounded
	// BoxStyleBold uses bold/heavy box characters
	BoxStyleBold
	// BoxStyleASCII uses ASCII characters for compatibility
	BoxStyleASCII
)

// BoxChars contains all characters needed to draw boxes
type BoxChars struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	Cross       string
	TeeTop      string
	TeeBottom   string
	TeeLeft     string
	TeeRight    string
}

// SpinnerStyle represents different spinner animation styles
type SpinnerStyle int

const (
	// SpinnerDots classic dots spinner
	SpinnerDots SpinnerStyle = iota
	// SpinnerLine line spinner
	SpinnerLine
	// SpinnerArrows arrow spinner
	SpinnerArrows
	// SpinnerBraille braille pattern spinner
	SpinnerBraille
	// SpinnerASCII ASCII-safe spinner
	SpinnerASCII
)

// ColorScheme defines a color palette for the terminal
type ColorScheme struct {
	Primary   string
	Secondary string
	Success   string
	Warning   string
	Error     string
	Info      string
	Muted     string
	Reset     string
}

// Renderer provides terminal rendering capabilities
type Renderer interface {
	// Box returns box drawing characters for the given style
	Box(style BoxStyle) BoxChars

	// ProgressBar renders a progress bar
	ProgressBar(width int, percent float64) string

	// Spinner returns a spinner frame
	Spinner(style SpinnerStyle, frame int) string

	// Colors returns the color scheme
	Colors() ColorScheme

	// ClearLine returns the escape sequence to clear a line
	ClearLine() string

	// MoveCursor returns the escape sequence to move cursor
	MoveCursor(x, y int) string

	// HideCursor returns the escape sequence to hide cursor
	HideCursor() string

	// ShowCursor returns the escape sequence to show cursor
	ShowCursor() string

	// Bold returns text formatted as bold
	Bold(text string) string

	// Italic returns text formatted as italic
	Italic(text string) string

	// Underline returns text formatted as underlined
	Underline(text string) string

	// Hyperlink creates a hyperlink if supported
	Hyperlink(url, text string) string
}

// rendererRegistry manages available renderers
type rendererRegistry struct {
	mu        sync.RWMutex
	renderers map[string]func(Capabilities) Renderer
}

// Register adds a new renderer to the registry
func (rr *rendererRegistry) Register(name string, factory func(Capabilities) Renderer) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.renderers[name] = factory
}

var registry = &rendererRegistry{
	renderers: make(map[string]func(Capabilities) Renderer),
}

func init() {
	// Register default renderers
	registry.Register("rich", func(caps Capabilities) Renderer {
		return NewRichRenderer(caps)
	})

	registry.Register("standard", func(caps Capabilities) Renderer {
		return NewStandardRenderer(caps)
	})

	registry.Register("fallback", func(caps Capabilities) Renderer {
		return NewFallbackRenderer()
	})
}

// SelectRenderer chooses the best renderer based on capabilities
func SelectRenderer(caps Capabilities) Renderer {
	if caps.SupportsRichUI() {
		return NewRichRenderer(caps)
	}

	if caps.SupportsColor() {
		return NewStandardRenderer(caps)
	}

	return NewFallbackRenderer()
}

// RichRenderer provides full Unicode and color rendering
type RichRenderer struct {
	caps   Capabilities
	colors ColorScheme
}

// NewRichRenderer creates a renderer for feature-rich terminals
func NewRichRenderer(caps Capabilities) *RichRenderer {
	r := &RichRenderer{
		caps: caps,
	}

	// Setup rich color scheme
	if caps.TrueColor {
		r.colors = ColorScheme{
			Primary:   "\x1b[38;2;88;166;255m",  // Bright blue
			Secondary: "\x1b[38;2;255;184;108m", // Orange
			Success:   "\x1b[38;2;142;215;108m", // Green
			Warning:   "\x1b[38;2;255;221;89m",  // Yellow
			Error:     "\x1b[38;2;255;101;101m", // Red
			Info:      "\x1b[38;2;120;219;255m", // Cyan
			Muted:     "\x1b[38;2;128;128;128m", // Gray
			Reset:     "\x1b[0m",
		}
	} else if caps.Colors >= Extended256 {
		r.colors = ColorScheme{
			Primary:   "\x1b[38;5;33m",  // Blue
			Secondary: "\x1b[38;5;214m", // Orange
			Success:   "\x1b[38;5;82m",  // Green
			Warning:   "\x1b[38;5;226m", // Yellow
			Error:     "\x1b[38;5;196m", // Red
			Info:      "\x1b[38;5;51m",  // Cyan
			Muted:     "\x1b[38;5;244m", // Gray
			Reset:     "\x1b[0m",
		}
	} else {
		r.colors = ColorScheme{
			Primary:   "\x1b[34m", // Blue
			Secondary: "\x1b[33m", // Yellow
			Success:   "\x1b[32m", // Green
			Warning:   "\x1b[33m", // Yellow
			Error:     "\x1b[31m", // Red
			Info:      "\x1b[36m", // Cyan
			Muted:     "\x1b[90m", // Bright black
			Reset:     "\x1b[0m",
		}
	}

	return r
}

func (r *RichRenderer) Box(style BoxStyle) BoxChars {
	switch style {
	case BoxStyleRounded:
		return BoxChars{
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "╰",
			BottomRight: "╯",
			Horizontal:  "─",
			Vertical:    "│",
			Cross:       "┼",
			TeeTop:      "┬",
			TeeBottom:   "┴",
			TeeLeft:     "├",
			TeeRight:    "┤",
		}
	case BoxStyleDouble:
		return BoxChars{
			TopLeft:     "╔",
			TopRight:    "╗",
			BottomLeft:  "╚",
			BottomRight: "╝",
			Horizontal:  "═",
			Vertical:    "║",
			Cross:       "╬",
			TeeTop:      "╦",
			TeeBottom:   "╩",
			TeeLeft:     "╠",
			TeeRight:    "╣",
		}
	case BoxStyleBold:
		return BoxChars{
			TopLeft:     "┏",
			TopRight:    "┓",
			BottomLeft:  "┗",
			BottomRight: "┛",
			Horizontal:  "━",
			Vertical:    "┃",
			Cross:       "╋",
			TeeTop:      "┳",
			TeeBottom:   "┻",
			TeeLeft:     "┣",
			TeeRight:    "┫",
		}
	default: // BoxStyleSingle
		return BoxChars{
			TopLeft:     "┌",
			TopRight:    "┐",
			BottomLeft:  "└",
			BottomRight: "┘",
			Horizontal:  "─",
			Vertical:    "│",
			Cross:       "┼",
			TeeTop:      "┬",
			TeeBottom:   "┴",
			TeeLeft:     "├",
			TeeRight:    "┤",
		}
	}
}

func (r *RichRenderer) ProgressBar(width int, percent float64) string {
	if width <= 0 {
		return ""
	}

	// Ensure percent is between 0 and 1
	if percent < 0 {
		percent = 0
	} else if percent > 1 {
		percent = 1
	}

	filled := int(float64(width) * percent)

	// Use block characters for smooth progress
	blocks := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}

	// Calculate partial block
	remainder := (float64(width) * percent) - float64(filled)
	partialIdx := int(remainder * float64(len(blocks)-1))

	bar := strings.Repeat("█", filled)

	if filled < width && partialIdx > 0 {
		bar += blocks[partialIdx]
		filled++
	}

	if filled < width {
		bar += strings.Repeat(" ", width-filled)
	}

	return fmt.Sprintf("[%s%s%s]", r.colors.Primary, bar, r.colors.Reset)
}

func (r *RichRenderer) Spinner(style SpinnerStyle, frame int) string {
	spinners := map[SpinnerStyle][]string{
		SpinnerDots:    {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		SpinnerLine:    {"─", "\\", "│", "/"},
		SpinnerArrows:  {"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
		SpinnerBraille: {"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		SpinnerASCII:   {"-", "\\", "|", "/"},
	}

	frames, ok := spinners[style]
	if !ok {
		frames = spinners[SpinnerDots]
	}

	return frames[frame%len(frames)]
}

func (r *RichRenderer) Colors() ColorScheme {
	return r.colors
}

func (r *RichRenderer) ClearLine() string {
	return "\x1b[2K\r"
}

func (r *RichRenderer) MoveCursor(x, y int) string {
	if x == 0 && y == 0 {
		return ""
	}

	var seq strings.Builder

	if y < 0 {
		seq.WriteString(fmt.Sprintf("\x1b[%dA", -y))
	} else if y > 0 {
		seq.WriteString(fmt.Sprintf("\x1b[%dB", y))
	}

	if x < 0 {
		seq.WriteString(fmt.Sprintf("\x1b[%dD", -x))
	} else if x > 0 {
		seq.WriteString(fmt.Sprintf("\x1b[%dC", x))
	}

	return seq.String()
}

func (r *RichRenderer) HideCursor() string {
	return "\x1b[?25l"
}

func (r *RichRenderer) ShowCursor() string {
	return "\x1b[?25h"
}

func (r *RichRenderer) Bold(text string) string {
	return fmt.Sprintf("\x1b[1m%s\x1b[22m", text)
}

func (r *RichRenderer) Italic(text string) string {
	return fmt.Sprintf("\x1b[3m%s\x1b[23m", text)
}

func (r *RichRenderer) Underline(text string) string {
	return fmt.Sprintf("\x1b[4m%s\x1b[24m", text)
}

func (r *RichRenderer) Hyperlink(url, text string) string {
	if r.caps.Hyperlinks {
		return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, text)
	}
	return text
}

// StandardRenderer provides basic color and ASCII rendering
type StandardRenderer struct {
	caps   Capabilities
	colors ColorScheme
}

// NewStandardRenderer creates a renderer for standard terminals
func NewStandardRenderer(caps Capabilities) *StandardRenderer {
	return &StandardRenderer{
		caps: caps,
		colors: ColorScheme{
			Primary:   "\x1b[34m", // Blue
			Secondary: "\x1b[33m", // Yellow
			Success:   "\x1b[32m", // Green
			Warning:   "\x1b[33m", // Yellow
			Error:     "\x1b[31m", // Red
			Info:      "\x1b[36m", // Cyan
			Muted:     "\x1b[90m", // Bright black
			Reset:     "\x1b[0m",
		},
	}
}

func (s *StandardRenderer) Box(style BoxStyle) BoxChars {
	// Use ASCII for standard renderer
	return BoxChars{
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
		Horizontal:  "-",
		Vertical:    "|",
		Cross:       "+",
		TeeTop:      "+",
		TeeBottom:   "+",
		TeeLeft:     "+",
		TeeRight:    "+",
	}
}

func (s *StandardRenderer) ProgressBar(width int, percent float64) string {
	if width <= 0 {
		return ""
	}

	filled := int(float64(width) * percent)
	bar := strings.Repeat("=", filled)

	if filled < width {
		bar += ">"
		bar += strings.Repeat(" ", width-filled-1)
	}

	return fmt.Sprintf("[%s]", bar)
}

func (s *StandardRenderer) Spinner(style SpinnerStyle, frame int) string {
	// Always use ASCII spinner for standard renderer
	frames := []string{"-", "\\", "|", "/"}
	return frames[frame%len(frames)]
}

func (s *StandardRenderer) Colors() ColorScheme {
	if s.caps.Colors == NoColor {
		return ColorScheme{} // No colors
	}
	return s.colors
}

func (s *StandardRenderer) ClearLine() string {
	return "\r" + strings.Repeat(" ", 80) + "\r"
}

func (s *StandardRenderer) MoveCursor(x, y int) string {
	// Basic ANSI cursor movement
	return ""
}

func (s *StandardRenderer) HideCursor() string {
	return ""
}

func (s *StandardRenderer) ShowCursor() string {
	return ""
}

func (s *StandardRenderer) Bold(text string) string {
	if s.caps.Colors > NoColor {
		return fmt.Sprintf("\x1b[1m%s\x1b[0m", text)
	}
	return text
}

func (s *StandardRenderer) Italic(text string) string {
	return text // No italic in standard renderer
}

func (s *StandardRenderer) Underline(text string) string {
	return text // No underline in standard renderer
}

func (s *StandardRenderer) Hyperlink(url, text string) string {
	return text // No hyperlinks in standard renderer
}

// FallbackRenderer provides minimal ASCII-only rendering
type FallbackRenderer struct{}

// NewFallbackRenderer creates a renderer for minimal terminals
func NewFallbackRenderer() *FallbackRenderer {
	return &FallbackRenderer{}
}

func (f *FallbackRenderer) Box(style BoxStyle) BoxChars {
	return BoxChars{
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
		Horizontal:  "-",
		Vertical:    "|",
		Cross:       "+",
		TeeTop:      "+",
		TeeBottom:   "+",
		TeeLeft:     "+",
		TeeRight:    "+",
	}
}

func (f *FallbackRenderer) ProgressBar(width int, percent float64) string {
	if width <= 0 {
		return ""
	}

	filled := int(float64(width) * percent)
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)

	return fmt.Sprintf("[%s] %d%%", bar, int(percent*100))
}

func (f *FallbackRenderer) Spinner(style SpinnerStyle, frame int) string {
	frames := []string{"-", "\\", "|", "/"}
	return frames[frame%len(frames)]
}

func (f *FallbackRenderer) Colors() ColorScheme {
	return ColorScheme{} // No colors
}

func (f *FallbackRenderer) ClearLine() string {
	return "\r" + strings.Repeat(" ", 80) + "\r"
}

func (f *FallbackRenderer) MoveCursor(x, y int) string {
	return ""
}

func (f *FallbackRenderer) HideCursor() string {
	return ""
}

func (f *FallbackRenderer) ShowCursor() string {
	return ""
}

func (f *FallbackRenderer) Bold(text string) string {
	return text
}

func (f *FallbackRenderer) Italic(text string) string {
	return text
}

func (f *FallbackRenderer) Underline(text string) string {
	return text
}

func (f *FallbackRenderer) Hyperlink(url, text string) string {
	return text
}
