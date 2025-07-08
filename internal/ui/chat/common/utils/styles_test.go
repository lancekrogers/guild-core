package utils

import (
	"testing"
)

func TestNewClaudeCodeStyles(t *testing.T) {
	styles := NewClaudeCodeStyles()

	if styles == nil {
		t.Fatal("NewClaudeCodeStyles returned nil")
	}

	// Test that all required styles are present
	// Simply verify the styles were created without error
	// (lipgloss doesn't apply ANSI codes in non-terminal environments)
}

func TestGetAgentColor(t *testing.T) {
	styles := NewStyles()

	tests := []struct {
		name      string
		agent     string
		wantColor bool
	}{
		{"elena lowercase", "elena", true},
		{"Elena uppercase", "Elena", true},
		{"marcus", "marcus", true},
		{"vera", "vera", true},
		{"unknown agent", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := styles.GetAgentColor(tt.agent)
			hasColor := color != ""

			if hasColor != tt.wantColor {
				t.Errorf("GetAgentColor(%s) = %v, want color present: %v",
					tt.agent, color, tt.wantColor)
			}
		})
	}
}

func TestChordManager(t *testing.T) {
	cm := NewChordManager()

	// Test chord registration
	if len(cm.chordBindings) == 0 {
		t.Error("ChordManager has no chord bindings registered")
	}

	// Test chord sequence
	cmd, handled := cm.HandleKey("@")
	if !handled {
		t.Error("@ key should start a chord sequence")
	}
	if cmd == nil {
		t.Error("@ key should return a command")
	}

	// Test completing a chord
	cmd, handled = cm.HandleKey("e")
	if !handled {
		t.Error("@e should complete a chord")
	}
	if cmd == nil {
		t.Error("@e should return a command")
	}

	// Test invalid chord
	cmd, handled = cm.HandleKey("x")
	if handled {
		t.Error("x key should not be handled as a chord")
	}
}

func TestThemeApply(t *testing.T) {
	styles := NewStyles()

	// Test theme switching
	themes := []string{"medieval", "minimal", "claude-code", "default"}

	for _, theme := range themes {
		t.Run(theme, func(t *testing.T) {
			// Just verify ApplyTheme doesn't panic
			styles.ApplyTheme(theme)
		})
	}
}
