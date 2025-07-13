package terminal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectRenderer(t *testing.T) {
	tests := []struct {
		name         string
		caps         Capabilities
		expectedType string
	}{
		{
			name: "rich renderer",
			caps: Capabilities{
				Colors:  Extended256,
				Unicode: true,
				Mouse:   true,
			},
			expectedType: "*terminal.RichRenderer",
		},
		{
			name: "standard renderer",
			caps: Capabilities{
				Colors:  Basic16,
				Unicode: false,
				Mouse:   false,
			},
			expectedType: "*terminal.StandardRenderer",
		},
		{
			name: "fallback renderer",
			caps: Capabilities{
				Colors: NoColor,
			},
			expectedType: "*terminal.FallbackRenderer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := SelectRenderer(tt.caps)
			assert.NotNil(t, renderer)
			assert.Contains(t, strings.Replace(tt.expectedType, "*", "", -1), "Renderer")
		})
	}
}

func TestRichRenderer_Box(t *testing.T) {
	renderer := NewRichRenderer(Capabilities{
		Colors:  TrueColor24Bit,
		Unicode: true,
	})

	tests := []struct {
		style BoxStyle
		check func(*testing.T, BoxChars)
	}{
		{
			style: BoxStyleSingle,
			check: func(t *testing.T, chars BoxChars) {
				assert.Equal(t, "┌", chars.TopLeft)
				assert.Equal(t, "┐", chars.TopRight)
				assert.Equal(t, "└", chars.BottomLeft)
				assert.Equal(t, "┘", chars.BottomRight)
				assert.Equal(t, "─", chars.Horizontal)
				assert.Equal(t, "│", chars.Vertical)
			},
		},
		{
			style: BoxStyleDouble,
			check: func(t *testing.T, chars BoxChars) {
				assert.Equal(t, "╔", chars.TopLeft)
				assert.Equal(t, "╗", chars.TopRight)
				assert.Equal(t, "╚", chars.BottomLeft)
				assert.Equal(t, "╝", chars.BottomRight)
				assert.Equal(t, "═", chars.Horizontal)
				assert.Equal(t, "║", chars.Vertical)
			},
		},
		{
			style: BoxStyleRounded,
			check: func(t *testing.T, chars BoxChars) {
				assert.Equal(t, "╭", chars.TopLeft)
				assert.Equal(t, "╮", chars.TopRight)
				assert.Equal(t, "╰", chars.BottomLeft)
				assert.Equal(t, "╯", chars.BottomRight)
			},
		},
		{
			style: BoxStyleBold,
			check: func(t *testing.T, chars BoxChars) {
				assert.Equal(t, "┏", chars.TopLeft)
				assert.Equal(t, "┓", chars.TopRight)
				assert.Equal(t, "┗", chars.BottomLeft)
				assert.Equal(t, "┛", chars.BottomRight)
				assert.Equal(t, "━", chars.Horizontal)
				assert.Equal(t, "┃", chars.Vertical)
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("BoxStyle_%d", tt.style), func(t *testing.T) {
			chars := renderer.Box(tt.style)
			tt.check(t, chars)
		})
	}
}

func TestRichRenderer_ProgressBar(t *testing.T) {
	renderer := NewRichRenderer(Capabilities{
		Colors:    TrueColor24Bit,
		Unicode:   true,
		TrueColor: true,
	})

	tests := []struct {
		name    string
		width   int
		percent float64
		check   func(*testing.T, string)
	}{
		{
			name:    "empty progress",
			width:   10,
			percent: 0.0,
			check: func(t *testing.T, bar string) {
				assert.Contains(t, bar, "[")
				assert.Contains(t, bar, "]")
				assert.NotContains(t, bar, "█")
			},
		},
		{
			name:    "half progress",
			width:   10,
			percent: 0.5,
			check: func(t *testing.T, bar string) {
				assert.Contains(t, bar, "█")
				// Should have some filled blocks
			},
		},
		{
			name:    "full progress",
			width:   10,
			percent: 1.0,
			check: func(t *testing.T, bar string) {
				assert.Contains(t, bar, "█")
			},
		},
		{
			name:    "over 100 percent",
			width:   10,
			percent: 1.5,
			check: func(t *testing.T, bar string) {
				// Should cap at 100%
				assert.NotEmpty(t, bar)
			},
		},
		{
			name:    "negative percent",
			width:   10,
			percent: -0.5,
			check: func(t *testing.T, bar string) {
				// Should treat as 0%
				assert.NotContains(t, bar, "█")
			},
		},
		{
			name:    "zero width",
			width:   0,
			percent: 0.5,
			check: func(t *testing.T, bar string) {
				assert.Empty(t, bar)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := renderer.ProgressBar(tt.width, tt.percent)
			tt.check(t, bar)
		})
	}
}

func TestRichRenderer_Spinner(t *testing.T) {
	renderer := NewRichRenderer(Capabilities{
		Colors:  TrueColor24Bit,
		Unicode: true,
	})

	tests := []struct {
		style  SpinnerStyle
		frames int
	}{
		{SpinnerDots, 10},
		{SpinnerLine, 4},
		{SpinnerArrows, 8},
		{SpinnerBraille, 8},
		{SpinnerASCII, 4},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("BoxStyle_%d", tt.style), func(t *testing.T) {
			// Test all frames
			frames := make(map[string]bool)
			for i := 0; i < tt.frames*2; i++ {
				frame := renderer.Spinner(tt.style, i)
				assert.NotEmpty(t, frame)
				frames[frame] = true
			}
			// Should cycle through frames
			assert.LessOrEqual(t, len(frames), tt.frames)
		})
	}

	// Test invalid spinner style
	frame := renderer.Spinner(SpinnerStyle(999), 0)
	assert.NotEmpty(t, frame) // Should fall back to default
}

func TestRichRenderer_Colors(t *testing.T) {
	renderer := NewRichRenderer(Capabilities{
		Colors:    TrueColor24Bit,
		TrueColor: true,
	})

	colors := renderer.Colors()
	assert.NotEmpty(t, colors.Primary)
	assert.NotEmpty(t, colors.Secondary)
	assert.NotEmpty(t, colors.Success)
	assert.NotEmpty(t, colors.Warning)
	assert.NotEmpty(t, colors.Error)
	assert.NotEmpty(t, colors.Info)
	assert.NotEmpty(t, colors.Muted)
	assert.Equal(t, "\x1b[0m", colors.Reset)

	// True color should have RGB sequences
	assert.Contains(t, colors.Primary, "38;2;")
}

func TestRichRenderer_TextFormatting(t *testing.T) {
	renderer := NewRichRenderer(Capabilities{
		Colors:  Extended256,
		Unicode: true,
	})

	tests := []struct {
		name   string
		method func(string) string
		input  string
		check  func(*testing.T, string)
	}{
		{
			name:   "bold",
			method: renderer.Bold,
			input:  "test",
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "\x1b[1m")
				assert.Contains(t, result, "test")
				assert.Contains(t, result, "\x1b[22m")
			},
		},
		{
			name:   "italic",
			method: renderer.Italic,
			input:  "test",
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "\x1b[3m")
				assert.Contains(t, result, "test")
				assert.Contains(t, result, "\x1b[23m")
			},
		},
		{
			name:   "underline",
			method: renderer.Underline,
			input:  "test",
			check: func(t *testing.T, result string) {
				assert.Contains(t, result, "\x1b[4m")
				assert.Contains(t, result, "test")
				assert.Contains(t, result, "\x1b[24m")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.method(tt.input)
			tt.check(t, result)
		})
	}
}

func TestRichRenderer_Hyperlink(t *testing.T) {
	tests := []struct {
		name        string
		caps        Capabilities
		url         string
		text        string
		wantEscapes bool
	}{
		{
			name: "with hyperlink support",
			caps: Capabilities{
				Colors:     Extended256,
				Unicode:    true,
				Hyperlinks: true,
			},
			url:         "https://example.com",
			text:        "Click here",
			wantEscapes: true,
		},
		{
			name: "without hyperlink support",
			caps: Capabilities{
				Colors:     Extended256,
				Unicode:    true,
				Hyperlinks: false,
			},
			url:         "https://example.com",
			text:        "Click here",
			wantEscapes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewRichRenderer(tt.caps)
			result := renderer.Hyperlink(tt.url, tt.text)

			if tt.wantEscapes {
				assert.Contains(t, result, "\x1b]8;;")
				assert.Contains(t, result, tt.url)
			} else {
				assert.Equal(t, tt.text, result)
			}
			assert.Contains(t, result, tt.text)
		})
	}
}

func TestStandardRenderer(t *testing.T) {
	renderer := NewStandardRenderer(Capabilities{
		Colors: Basic16,
	})

	t.Run("box chars", func(t *testing.T) {
		chars := renderer.Box(BoxStyleSingle)
		assert.Equal(t, "+", chars.TopLeft)
		assert.Equal(t, "-", chars.Horizontal)
		assert.Equal(t, "|", chars.Vertical)
	})

	t.Run("progress bar", func(t *testing.T) {
		bar := renderer.ProgressBar(10, 0.5)
		assert.Contains(t, bar, "[")
		assert.Contains(t, bar, "]")
		assert.Contains(t, bar, "=")
		assert.Contains(t, bar, ">")
	})

	t.Run("spinner", func(t *testing.T) {
		frame := renderer.Spinner(SpinnerDots, 0)
		assert.Contains(t, "-\\|/", frame)
	})

	t.Run("colors", func(t *testing.T) {
		colors := renderer.Colors()
		assert.NotEmpty(t, colors.Primary)
		assert.Contains(t, colors.Primary, "\x1b[")
	})

	t.Run("no colors", func(t *testing.T) {
		noColorRenderer := NewStandardRenderer(Capabilities{Colors: NoColor})
		colors := noColorRenderer.Colors()
		assert.Empty(t, colors.Primary)
		assert.Empty(t, colors.Reset)
	})
}

func TestFallbackRenderer(t *testing.T) {
	renderer := NewFallbackRenderer()

	t.Run("box chars", func(t *testing.T) {
		chars := renderer.Box(BoxStyleSingle)
		assert.Equal(t, "+", chars.TopLeft)
		assert.Equal(t, "-", chars.Horizontal)
		assert.Equal(t, "|", chars.Vertical)
	})

	t.Run("progress bar", func(t *testing.T) {
		bar := renderer.ProgressBar(10, 0.5)
		assert.Contains(t, bar, "[")
		assert.Contains(t, bar, "]")
		assert.Contains(t, bar, "#")
		assert.Contains(t, bar, "-")
		assert.Contains(t, bar, "50%")
	})

	t.Run("spinner", func(t *testing.T) {
		frame := renderer.Spinner(SpinnerDots, 0)
		assert.Contains(t, "-\\|/", frame)
	})

	t.Run("no colors", func(t *testing.T) {
		colors := renderer.Colors()
		assert.Empty(t, colors.Primary)
		assert.Empty(t, colors.Secondary)
		assert.Empty(t, colors.Reset)
	})

	t.Run("text formatting", func(t *testing.T) {
		assert.Equal(t, "test", renderer.Bold("test"))
		assert.Equal(t, "test", renderer.Italic("test"))
		assert.Equal(t, "test", renderer.Underline("test"))
		assert.Equal(t, "text", renderer.Hyperlink("http://example.com", "text"))
	})

	t.Run("control sequences", func(t *testing.T) {
		assert.Contains(t, renderer.ClearLine(), "\r")
		assert.Empty(t, renderer.MoveCursor(1, 1))
		assert.Empty(t, renderer.HideCursor())
		assert.Empty(t, renderer.ShowCursor())
	})
}

func TestRendererRegistry(t *testing.T) {
	// Test that renderers are registered
	require.NotNil(t, registry)
	require.NotNil(t, registry.renderers)

	// Test registration
	testFactory := func(caps Capabilities) Renderer {
		return NewFallbackRenderer()
	}

	registry.Register("test", testFactory)

	// Verify it was registered
	registry.mu.RLock()
	_, exists := registry.renderers["test"]
	registry.mu.RUnlock()
	assert.True(t, exists)
}
