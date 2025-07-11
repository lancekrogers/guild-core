package features

import (
	"context"
	"os"
	"runtime"
	"strings"
	"unicode"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// UnicodeDetector detects terminal Unicode support
type UnicodeDetector struct {
	platform string
	isSSH    bool
	isCI     bool
}

// NewUnicodeDetector creates a new Unicode detector
func NewUnicodeDetector() *UnicodeDetector {
	return &UnicodeDetector{
		platform: runtime.GOOS,
		isSSH:    os.Getenv("SSH_CONNECTION") != "",
		isCI:     isCI(),
	}
}

// Detect determines if the terminal supports Unicode
func (ud *UnicodeDetector) Detect(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled during Unicode detection")
	}

	// Force disable Unicode if requested
	if os.Getenv("GUILD_FORCE_ASCII") == "1" {
		return false, nil
	}

	// Force enable Unicode if requested
	if os.Getenv("GUILD_FORCE_UNICODE") == "1" {
		return true, nil
	}

	// Platform-specific detection
	switch ud.platform {
	case "windows":
		return ud.detectWindowsUnicode(), nil
	case "darwin":
		return ud.detectMacOSUnicode(), nil
	default:
		return ud.detectLinuxUnicode(), nil
	}
}

// detectWindowsUnicode checks Unicode support on Windows
func (ud *UnicodeDetector) detectWindowsUnicode() bool {
	// Windows Terminal supports Unicode
	if os.Getenv("WT_SESSION") != "" {
		return true
	}

	// ConEmu supports Unicode
	if os.Getenv("ConEmuPID") != "" {
		return true
	}

	// VS Code terminal supports Unicode
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return true
	}

	// Check for UTF-8 code page
	// In a real implementation, you'd use Windows API to check the console code page
	// For now, we'll check environment variables
	if strings.Contains(strings.ToLower(os.Getenv("LANG")), "utf-8") {
		return true
	}

	// Modern Windows 10+ with UTF-8 support
	// This is a simplified check - in production you'd use proper Windows API calls
	return false
}

// detectMacOSUnicode checks Unicode support on macOS
func (ud *UnicodeDetector) detectMacOSUnicode() bool {
	// macOS terminals generally support Unicode well
	termProgram := os.Getenv("TERM_PROGRAM")

	switch termProgram {
	case "Apple_Terminal", "iTerm.app", "vscode", "Hyper":
		return true
	}

	// Check locale
	if ud.checkLocaleUTF8() {
		return true
	}

	// SSH might not properly propagate locale
	if ud.isSSH {
		return false
	}

	// Default to true for macOS in most cases
	return true
}

// detectLinuxUnicode checks Unicode support on Linux
func (ud *UnicodeDetector) detectLinuxUnicode() bool {
	// Check locale settings first
	if ud.checkLocaleUTF8() {
		return true
	}

	// Check for known Unicode-supporting terminals
	term := os.Getenv("TERM")
	unicodeTerms := []string{
		"xterm-256color",
		"gnome-terminal",
		"konsole",
		"alacritty",
		"kitty",
	}

	for _, ut := range unicodeTerms {
		if strings.Contains(term, ut) {
			return true
		}
	}

	// VTE-based terminals support Unicode
	if os.Getenv("VTE_VERSION") != "" {
		return true
	}

	// SSH sessions might not have proper locale
	if ud.isSSH {
		return false
	}

	// CI environments might not support Unicode
	if ud.isCI {
		return false
	}

	return false
}

// checkLocaleUTF8 checks if the locale settings indicate UTF-8 support
func (ud *UnicodeDetector) checkLocaleUTF8() bool {
	localeVars := []string{"LC_ALL", "LC_CTYPE", "LANG"}

	for _, v := range localeVars {
		locale := strings.ToLower(os.Getenv(v))
		if strings.Contains(locale, "utf-8") || strings.Contains(locale, "utf8") {
			return true
		}
	}

	return false
}

// UnicodeCharsets provides collections of Unicode characters organized by category
type UnicodeCharsets struct {
	BoxDrawing   BoxDrawingChars
	Symbols      SymbolChars
	Arrows       ArrowChars
	Bullets      BulletChars
	Mathematical MathChars
	Emoji        EmojiChars
}

// BoxDrawingChars contains Unicode box drawing characters
type BoxDrawingChars struct {
	Light struct {
		Horizontal  string
		Vertical    string
		TopLeft     string
		TopRight    string
		BottomLeft  string
		BottomRight string
		Cross       string
		TeeUp       string
		TeeDown     string
		TeeLeft     string
		TeeRight    string
	}
	Heavy struct {
		Horizontal  string
		Vertical    string
		TopLeft     string
		TopRight    string
		BottomLeft  string
		BottomRight string
		Cross       string
		TeeUp       string
		TeeDown     string
		TeeLeft     string
		TeeRight    string
	}
	Double struct {
		Horizontal  string
		Vertical    string
		TopLeft     string
		TopRight    string
		BottomLeft  string
		BottomRight string
		Cross       string
		TeeUp       string
		TeeDown     string
		TeeLeft     string
		TeeRight    string
	}
	Rounded struct {
		TopLeft     string
		TopRight    string
		BottomLeft  string
		BottomRight string
	}
}

// SymbolChars contains common Unicode symbols
type SymbolChars struct {
	Checkmark    string
	Cross        string
	Warning      string
	Info         string
	Bullet       string
	Diamond      string
	Star         string
	Heart        string
	Ellipsis     string
	MiddleDot    string
	Infinity     string
	Copyright    string
	Trademark    string
	RegisteredTM string
}

// ArrowChars contains Unicode arrow characters
type ArrowChars struct {
	Up          string
	Down        string
	Left        string
	Right       string
	UpDown      string
	LeftRight   string
	UpLeft      string
	UpRight     string
	DownLeft    string
	DownRight   string
	DoubleUp    string
	DoubleDown  string
	DoubleLeft  string
	DoubleRight string
}

// BulletChars contains Unicode bullet characters
type BulletChars struct {
	Circle       string
	Square       string
	Triangle     string
	Diamond      string
	Star         string
	Hyphen       string
	Asterisk     string
	Plus         string
	RightPointer string
}

// MathChars contains mathematical Unicode symbols
type MathChars struct {
	PlusMinus     string
	Multiply      string
	Divide        string
	LessEqual     string
	GreaterEqual  string
	NotEqual      string
	Approximately string
	Infinity      string
	Sum           string
	Product       string
	Integral      string
	Delta         string
	Pi            string
	Sqrt          string
}

// EmojiChars contains commonly used emoji
type EmojiChars struct {
	Success    string
	Error      string
	Warning    string
	Info       string
	Question   string
	Fire       string
	Star       string
	Heart      string
	ThumbsUp   string
	ThumbsDown string
	Celebrate  string
	Rocket     string
}

// NewUnicodeCharsets creates a new Unicode character set
func NewUnicodeCharsets() *UnicodeCharsets {
	uc := &UnicodeCharsets{}

	// Box drawing characters
	uc.BoxDrawing.Light.Horizontal = "─"
	uc.BoxDrawing.Light.Vertical = "│"
	uc.BoxDrawing.Light.TopLeft = "┌"
	uc.BoxDrawing.Light.TopRight = "┐"
	uc.BoxDrawing.Light.BottomLeft = "└"
	uc.BoxDrawing.Light.BottomRight = "┘"
	uc.BoxDrawing.Light.Cross = "┼"
	uc.BoxDrawing.Light.TeeUp = "┴"
	uc.BoxDrawing.Light.TeeDown = "┬"
	uc.BoxDrawing.Light.TeeLeft = "┤"
	uc.BoxDrawing.Light.TeeRight = "├"

	uc.BoxDrawing.Heavy.Horizontal = "━"
	uc.BoxDrawing.Heavy.Vertical = "┃"
	uc.BoxDrawing.Heavy.TopLeft = "┏"
	uc.BoxDrawing.Heavy.TopRight = "┓"
	uc.BoxDrawing.Heavy.BottomLeft = "┗"
	uc.BoxDrawing.Heavy.BottomRight = "┛"
	uc.BoxDrawing.Heavy.Cross = "╋"
	uc.BoxDrawing.Heavy.TeeUp = "┻"
	uc.BoxDrawing.Heavy.TeeDown = "┳"
	uc.BoxDrawing.Heavy.TeeLeft = "┫"
	uc.BoxDrawing.Heavy.TeeRight = "┣"

	uc.BoxDrawing.Double.Horizontal = "═"
	uc.BoxDrawing.Double.Vertical = "║"
	uc.BoxDrawing.Double.TopLeft = "╔"
	uc.BoxDrawing.Double.TopRight = "╗"
	uc.BoxDrawing.Double.BottomLeft = "╚"
	uc.BoxDrawing.Double.BottomRight = "╝"
	uc.BoxDrawing.Double.Cross = "╬"
	uc.BoxDrawing.Double.TeeUp = "╩"
	uc.BoxDrawing.Double.TeeDown = "╦"
	uc.BoxDrawing.Double.TeeLeft = "╣"
	uc.BoxDrawing.Double.TeeRight = "╠"

	uc.BoxDrawing.Rounded.TopLeft = "╭"
	uc.BoxDrawing.Rounded.TopRight = "╮"
	uc.BoxDrawing.Rounded.BottomLeft = "╰"
	uc.BoxDrawing.Rounded.BottomRight = "╯"

	// Symbols
	uc.Symbols.Checkmark = "✓"
	uc.Symbols.Cross = "✗"
	uc.Symbols.Warning = "⚠"
	uc.Symbols.Info = "ℹ"
	uc.Symbols.Bullet = "•"
	uc.Symbols.Diamond = "◆"
	uc.Symbols.Star = "★"
	uc.Symbols.Heart = "♥"
	uc.Symbols.Ellipsis = "…"
	uc.Symbols.MiddleDot = "·"
	uc.Symbols.Infinity = "∞"
	uc.Symbols.Copyright = "©"
	uc.Symbols.Trademark = "™"
	uc.Symbols.RegisteredTM = "®"

	// Arrows
	uc.Arrows.Up = "↑"
	uc.Arrows.Down = "↓"
	uc.Arrows.Left = "←"
	uc.Arrows.Right = "→"
	uc.Arrows.UpDown = "↕"
	uc.Arrows.LeftRight = "↔"
	uc.Arrows.UpLeft = "↖"
	uc.Arrows.UpRight = "↗"
	uc.Arrows.DownLeft = "↙"
	uc.Arrows.DownRight = "↘"
	uc.Arrows.DoubleUp = "⇑"
	uc.Arrows.DoubleDown = "⇓"
	uc.Arrows.DoubleLeft = "⇐"
	uc.Arrows.DoubleRight = "⇒"

	// Bullets
	uc.Bullets.Circle = "•"
	uc.Bullets.Square = "▪"
	uc.Bullets.Triangle = "▸"
	uc.Bullets.Diamond = "◆"
	uc.Bullets.Star = "✦"
	uc.Bullets.Hyphen = "‐"
	uc.Bullets.Asterisk = "∗"
	uc.Bullets.Plus = "⊕"
	uc.Bullets.RightPointer = "▶"

	// Mathematical symbols
	uc.Mathematical.PlusMinus = "±"
	uc.Mathematical.Multiply = "×"
	uc.Mathematical.Divide = "÷"
	uc.Mathematical.LessEqual = "≤"
	uc.Mathematical.GreaterEqual = "≥"
	uc.Mathematical.NotEqual = "≠"
	uc.Mathematical.Approximately = "≈"
	uc.Mathematical.Infinity = "∞"
	uc.Mathematical.Sum = "∑"
	uc.Mathematical.Product = "∏"
	uc.Mathematical.Integral = "∫"
	uc.Mathematical.Delta = "Δ"
	uc.Mathematical.Pi = "π"
	uc.Mathematical.Sqrt = "√"

	// Emoji
	uc.Emoji.Success = "✅"
	uc.Emoji.Error = "❌"
	uc.Emoji.Warning = "⚠️"
	uc.Emoji.Info = "ℹ️"
	uc.Emoji.Question = "❓"
	uc.Emoji.Fire = "🔥"
	uc.Emoji.Star = "⭐"
	uc.Emoji.Heart = "❤️"
	uc.Emoji.ThumbsUp = "👍"
	uc.Emoji.ThumbsDown = "👎"
	uc.Emoji.Celebrate = "🎉"
	uc.Emoji.Rocket = "🚀"

	return uc
}

// IsDisplayable checks if a character can be displayed
func IsDisplayable(r rune) bool {
	return unicode.IsPrint(r) || unicode.IsSpace(r)
}

// Width estimates the display width of a string
func Width(s string) int {
	width := 0
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Mn, r): // Combining marks
			// Don't add width for combining characters
		case unicode.Is(unicode.Me, r): // Enclosing marks
			// Don't add width for enclosing marks
		case unicode.Is(unicode.Cf, r): // Format characters
			// Don't add width for format characters
		case r >= 0x1100 && r <= 0x115F: // Hangul Jamo
			width += 2
		case r >= 0x2E80 && r <= 0x9FFF: // CJK
			width += 2
		case r >= 0xAC00 && r <= 0xD7AF: // Hangul Syllables
			width += 2
		case r >= 0xF900 && r <= 0xFAFF: // CJK Compatibility Ideographs
			width += 2
		case r >= 0xFE10 && r <= 0xFE19: // Vertical Forms
			width += 2
		case r >= 0xFE30 && r <= 0xFE6F: // CJK Compatibility Forms
			width += 2
		case r >= 0xFF00 && r <= 0xFF60: // Fullwidth Forms
			width += 2
		case r >= 0xFFE0 && r <= 0xFFE6: // Fullwidth Forms
			width += 2
		case r >= 0x20000 && r <= 0x2FFFD: // CJK Extension B
			width += 2
		case r >= 0x30000 && r <= 0x3FFFD: // CJK Extension C
			width += 2
		default:
			width += 1
		}
	}
	return width
}

// Truncate truncates a string to a maximum display width
func Truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	width := 0
	var result []rune

	for _, r := range s {
		charWidth := 1
		if r >= 0x1100 && r <= 0x115F || // Hangul Jamo
			r >= 0x2E80 && r <= 0x9FFF || // CJK
			r >= 0xAC00 && r <= 0xD7AF || // Hangul Syllables
			r >= 0xF900 && r <= 0xFAFF || // CJK Compatibility
			r >= 0xFE30 && r <= 0xFE6F || // CJK Compatibility Forms
			r >= 0xFF00 && r <= 0xFF60 || // Fullwidth
			r >= 0xFFE0 && r <= 0xFFE6 { // Fullwidth
			charWidth = 2
		}

		if width+charWidth > maxWidth {
			break
		}

		result = append(result, r)
		width += charWidth
	}

	return string(result)
}
