package terminal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorSupport_String(t *testing.T) {
	tests := []struct {
		cs   ColorSupport
		want string
	}{
		{NoColor, "no-color"},
		{Basic16, "16-color"},
		{Extended256, "256-color"},
		{TrueColor24Bit, "true-color"},
		{ColorSupport(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cs.String())
		})
	}
}

func TestCapabilities_SupportsColor(t *testing.T) {
	tests := []struct {
		name string
		caps Capabilities
		want bool
	}{
		{
			name: "no color",
			caps: Capabilities{Colors: NoColor},
			want: false,
		},
		{
			name: "basic color",
			caps: Capabilities{Colors: Basic16},
			want: true,
		},
		{
			name: "256 color",
			caps: Capabilities{Colors: Extended256},
			want: true,
		},
		{
			name: "true color",
			caps: Capabilities{Colors: TrueColor24Bit},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.caps.SupportsColor())
		})
	}
}

func TestCapabilities_SupportsRichUI(t *testing.T) {
	tests := []struct {
		name string
		caps Capabilities
		want bool
	}{
		{
			name: "minimal terminal",
			caps: Capabilities{
				Colors:  NoColor,
				Unicode: false,
				Mouse:   false,
			},
			want: false,
		},
		{
			name: "basic terminal",
			caps: Capabilities{
				Colors:  Basic16,
				Unicode: false,
				Mouse:   false,
			},
			want: false,
		},
		{
			name: "rich terminal without mouse",
			caps: Capabilities{
				Colors:  Extended256,
				Unicode: true,
				Mouse:   false,
			},
			want: false,
		},
		{
			name: "rich terminal with all features",
			caps: Capabilities{
				Colors:  Extended256,
				Unicode: true,
				Mouse:   true,
			},
			want: true,
		},
		{
			name: "true color terminal",
			caps: Capabilities{
				Colors:    TrueColor24Bit,
				Unicode:   true,
				Mouse:     true,
				TrueColor: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.caps.SupportsRichUI())
		})
	}
}

func TestCapabilities_Merge(t *testing.T) {
	tests := []struct {
		name  string
		base  Capabilities
		other Capabilities
		want  Capabilities
	}{
		{
			name: "merge with empty",
			base: Capabilities{
				Colors:  Extended256,
				Unicode: true,
				Mouse:   true,
			},
			other: Capabilities{},
			want: Capabilities{
				Colors:  Extended256,
				Unicode: true,
				Mouse:   true,
			},
		},
		{
			name: "upgrade colors",
			base: Capabilities{
				Colors:  Basic16,
				Unicode: true,
			},
			other: Capabilities{
				Colors: TrueColor24Bit,
			},
			want: Capabilities{
				Colors:  TrueColor24Bit,
				Unicode: true,
			},
		},
		{
			name: "merge features",
			base: Capabilities{
				Colors:  Extended256,
				Unicode: false,
				Mouse:   false,
			},
			other: Capabilities{
				Unicode:    true,
				Mouse:      true,
				Hyperlinks: true,
			},
			want: Capabilities{
				Colors:     Extended256,
				Unicode:    true,
				Mouse:      true,
				Hyperlinks: true,
			},
		},
		{
			name: "merge all features",
			base: Capabilities{
				Colors: Basic16,
			},
			other: Capabilities{
				Colors:          TrueColor24Bit,
				Unicode:         true,
				Mouse:           true,
				Size:            true,
				TrueColor:       true,
				Hyperlinks:      true,
				Images:          true,
				CursorShape:     true,
				AlternateScreen: true,
				Sixel:           true,
				Kitty:           true,
				ITerm2:          true,
			},
			want: Capabilities{
				Colors:          TrueColor24Bit,
				Unicode:         true,
				Mouse:           true,
				Size:            true,
				TrueColor:       true,
				Hyperlinks:      true,
				Images:          true,
				CursorShape:     true,
				AlternateScreen: true,
				Sixel:           true,
				Kitty:           true,
				ITerm2:          true,
			},
		},
		{
			name: "don't downgrade colors",
			base: Capabilities{
				Colors: TrueColor24Bit,
			},
			other: Capabilities{
				Colors: Basic16,
			},
			want: Capabilities{
				Colors: TrueColor24Bit,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.base.Merge(tt.other)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestCapabilities_String(t *testing.T) {
	tests := []struct {
		name string
		caps Capabilities
		want []string // Expected strings to contain
	}{
		{
			name: "minimal capabilities",
			caps: Capabilities{
				Colors: NoColor,
			},
			want: []string{"Colors: no-color"},
		},
		{
			name: "basic capabilities",
			caps: Capabilities{
				Colors:  Basic16,
				Unicode: true,
			},
			want: []string{"Colors: 16-color", "Unicode: true"},
		},
		{
			name: "full capabilities",
			caps: Capabilities{
				Colors:          TrueColor24Bit,
				Unicode:         true,
				Mouse:           true,
				Size:            true,
				TrueColor:       true,
				Hyperlinks:      true,
				Images:          true,
				CursorShape:     true,
				AlternateScreen: true,
				Sixel:           true,
				Kitty:           true,
				ITerm2:          true,
			},
			want: []string{
				"Colors: true-color",
				"Unicode: true",
				"Mouse: true",
				"Size: true",
				"TrueColor: true",
				"Hyperlinks: true",
				"Images: true",
				"CursorShape: true",
				"AlternateScreen: true",
				"Sixel: true",
				"Kitty: true",
				"ITerm2: true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.caps.String()
			for _, expected := range tt.want {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestCapabilities_Copy(t *testing.T) {
	original := Capabilities{
		Colors:          TrueColor24Bit,
		Unicode:         true,
		Mouse:           true,
		Size:            true,
		TrueColor:       true,
		Hyperlinks:      true,
		Images:          true,
		CursorShape:     true,
		AlternateScreen: true,
		Sixel:           true,
		Kitty:           true,
		ITerm2:          true,
	}

	copied := original.Copy()
	
	// Verify all fields are copied
	assert.Equal(t, original, copied)
	
	// Modify the copy
	copied.Colors = NoColor
	copied.Unicode = false
	
	// Original should be unchanged
	assert.Equal(t, TrueColor24Bit, original.Colors)
	assert.True(t, original.Unicode)
}

func TestCapabilities_LazyEvaluation(t *testing.T) {
	// Test that capabilities support lazy evaluation patterns
	caps := Capabilities{
		Colors:  Extended256,
		Unicode: true,
	}

	// These methods should be safe to call multiple times
	for i := 0; i < 10; i++ {
		assert.True(t, caps.SupportsColor())
		assert.False(t, caps.SupportsRichUI()) // No mouse
	}

	// Add mouse support
	caps.Mouse = true
	assert.True(t, caps.SupportsRichUI())
}

func TestCapabilities_MinimalTerminal(t *testing.T) {
	// Test behavior with minimal terminal capabilities
	caps := Capabilities{}
	
	assert.False(t, caps.SupportsColor())
	assert.False(t, caps.SupportsRichUI())
	assert.Equal(t, NoColor, caps.Colors)
	assert.False(t, caps.Unicode)
	assert.False(t, caps.Mouse)
	assert.False(t, caps.TrueColor)
}

func TestCapabilities_FeatureDetection(t *testing.T) {
	tests := []struct {
		name     string
		caps     Capabilities
		features map[string]bool
	}{
		{
			name: "basic terminal",
			caps: Capabilities{
				Colors:  Basic16,
				Unicode: false,
			},
			features: map[string]bool{
				"color":      true,
				"rich_ui":    false,
				"true_color": false,
				"images":     false,
			},
		},
		{
			name: "modern terminal",
			caps: Capabilities{
				Colors:     TrueColor24Bit,
				Unicode:    true,
				Mouse:      true,
				TrueColor:  true,
				Hyperlinks: true,
			},
			features: map[string]bool{
				"color":      true,
				"rich_ui":    true,
				"true_color": true,
				"hyperlinks": true,
			},
		},
		{
			name: "iTerm2 specific",
			caps: Capabilities{
				Colors:  TrueColor24Bit,
				Unicode: true,
				Mouse:   true,
				Images:  true,
				ITerm2:  true,
			},
			features: map[string]bool{
				"color":   true,
				"rich_ui": true,
				"images":  true,
				"iterm2":  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if val, ok := tt.features["color"]; ok {
				assert.Equal(t, val, tt.caps.SupportsColor())
			}
			if val, ok := tt.features["rich_ui"]; ok {
				assert.Equal(t, val, tt.caps.SupportsRichUI())
			}
			if val, ok := tt.features["true_color"]; ok {
				assert.Equal(t, val, tt.caps.TrueColor)
			}
			if val, ok := tt.features["hyperlinks"]; ok {
				assert.Equal(t, val, tt.caps.Hyperlinks)
			}
			if val, ok := tt.features["images"]; ok {
				assert.Equal(t, val, tt.caps.Images)
			}
			if val, ok := tt.features["iterm2"]; ok {
				assert.Equal(t, val, tt.caps.ITerm2)
			}
		})
	}
}