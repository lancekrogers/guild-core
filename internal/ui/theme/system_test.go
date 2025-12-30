// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package theme

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/guild-framework/guild-core/internal/ui"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

func TestNewThemeManager(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, *ThemeManager)
	}{
		{
			name: "creates_manager_with_default_themes",
			validate: func(t *testing.T, tm *ThemeManager) {
				if tm == nil {
					t.Fatal("theme manager should not be nil")
				}

				themes := tm.ListThemes()
				if len(themes) == 0 {
					t.Error("should have default themes")
				}

				// Should have built-in themes
				hasLight := false
				hasDark := false
				for _, theme := range themes {
					if theme == "claude-code-light" {
						hasLight = true
					}
					if theme == "claude-code-dark" {
						hasDark = true
					}
				}

				if !hasLight {
					t.Error("should have claude-code-light theme")
				}
				if !hasDark {
					t.Error("should have claude-code-dark theme")
				}
			},
		},
		{
			name: "sets_default_current_theme",
			validate: func(t *testing.T, tm *ThemeManager) {
				current := tm.GetCurrentTheme()
				if current == nil {
					t.Error("should have a current theme set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewThemeManager()
			tt.validate(t, tm)
		})
	}
}

func TestThemeManager_ApplyTheme(t *testing.T) {
	tests := []struct {
		name      string
		themeName string
		setup     func(*ThemeManager)
		wantErr   bool
		errCode   gerror.ErrorCode
	}{
		{
			name:      "applies_existing_theme",
			themeName: "claude-code-dark",
			setup:     nil,
			wantErr:   false,
		},
		{
			name:      "fails_for_nonexistent_theme",
			themeName: "nonexistent-theme",
			setup:     nil,
			wantErr:   true,
			errCode:   ui.ErrCodeUIThemeNotFound,
		},
		{
			name:      "applies_custom_theme",
			themeName: "custom-theme",
			setup: func(tm *ThemeManager) {
				customTheme := &Theme{
					Name:        "custom-theme",
					DisplayName: "Custom Theme",
					Version:     "1.0.0",
					Author:      "Test",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				tm.themes["custom-theme"] = customTheme
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tm := NewThemeManager()

			if tt.setup != nil {
				tt.setup(tm)
			}

			err := tm.ApplyTheme(ctx, tt.themeName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					return
				}

				if tt.errCode != "" {
					var gErr *gerror.GuildError
					if !gerror.As(err, &gErr) {
						t.Errorf("expected gerror.GuildError, got %T", err)
						return
					}

					if gErr.Code != tt.errCode {
						t.Errorf("expected error code %v, got %v", tt.errCode, gErr.Code)
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
					return
				}

				current := tm.GetCurrentTheme()
				if current == nil {
					t.Error("current theme should not be nil after applying")
					return
				}

				if current.Name != tt.themeName {
					t.Errorf("expected current theme name %v, got %v", tt.themeName, current.Name)
				}
			}
		})
	}
}

func TestThemeManager_ApplyTheme_ContextCancellation(t *testing.T) {
	tm := NewThemeManager()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := tm.ApplyTheme(ctx, "claude-code-dark")
	if err == nil {
		t.Error("expected error due to cancelled context")
	}
}

func TestThemeManager_GetComponent(t *testing.T) {
	tests := []struct {
		name          string
		componentName string
		setup         func(*ThemeManager)
		wantEmpty     bool
	}{
		{
			name:          "returns_button_primary_style",
			componentName: "button.primary",
			setup:         nil,
			wantEmpty:     false,
		},
		{
			name:          "returns_empty_for_unknown_component",
			componentName: "unknown.component",
			setup:         nil,
			wantEmpty:     true,
		},
		{
			name:          "returns_empty_when_no_current_theme",
			componentName: "button.primary",
			setup: func(tm *ThemeManager) {
				tm.currentTheme = nil
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewThemeManager()

			if tt.setup != nil {
				tt.setup(tm)
			}

			style := tm.GetComponent(tt.componentName)

			// Check if style is effectively empty
			rendered := style.Render("test")
			isEmpty := rendered == "test" // No styling applied

			if tt.wantEmpty && !isEmpty {
				t.Error("expected empty style but got styled result")
			}
			if !tt.wantEmpty && isEmpty {
				t.Error("expected styled result but got empty style")
			}
		})
	}
}

func TestThemeManager_GetAgentStyle(t *testing.T) {
	tests := []struct {
		name    string
		agentID string
		setup   func(*ThemeManager)
	}{
		{
			name:    "returns_style_for_known_agent",
			agentID: "agent-1",
			setup:   nil,
		},
		{
			name:    "returns_default_style_for_unknown_agent",
			agentID: "unknown-agent",
			setup:   nil,
		},
		{
			name:    "returns_empty_style_when_no_theme",
			agentID: "agent-1",
			setup: func(tm *ThemeManager) {
				tm.currentTheme = nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := NewThemeManager()

			if tt.setup != nil {
				tt.setup(tm)
			}

			style := tm.GetAgentStyle(tt.agentID)

			// Should return a valid style (even if empty)
			rendered := style.Render("test")
			if rendered == "" {
				t.Error("agent style should render something")
			}
		})
	}
}

func TestThemeManager_LoadThemeFromFile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (string, func())
		wantErr bool
		errCode gerror.ErrorCode
	}{
		{
			name: "loads_valid_theme_file",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				themeFile := filepath.Join(tmpDir, "test-theme.json")

				themeJSON := `{
					"name": "test-theme",
					"display_name": "Test Theme",
					"version": "1.0.0",
					"author": "Test Author",
					"colors": {
						"primary": {"base": "#FF0000", "light": "#FF3333", "dark": "#CC0000", "inverse": "#FFFFFF"}
					},
					"created_at": "2025-01-01T00:00:00Z",
					"updated_at": "2025-01-01T00:00:00Z"
				}`

				err := os.WriteFile(themeFile, []byte(themeJSON), 0o644)
				if err != nil {
					t.Fatalf("failed to create test theme file: %v", err)
				}

				return themeFile, func() {}
			},
			wantErr: false,
		},
		{
			name: "fails_for_nonexistent_file",
			setup: func() (string, func()) {
				return "/nonexistent/theme.json", func() {}
			},
			wantErr: true,
			errCode: gerror.ErrCodeIO,
		},
		{
			name: "fails_for_invalid_json",
			setup: func() (string, func()) {
				tmpDir := t.TempDir()
				themeFile := filepath.Join(tmpDir, "invalid-theme.json")

				err := os.WriteFile(themeFile, []byte("invalid json"), 0o644)
				if err != nil {
					t.Fatalf("failed to create invalid theme file: %v", err)
				}

				return themeFile, func() {}
			},
			wantErr: true,
			errCode: gerror.ErrCodeParsing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tm := NewThemeManager()

			filePath, cleanup := tt.setup()
			defer cleanup()

			err := tm.LoadThemeFromFile(ctx, filePath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					return
				}

				if tt.errCode != "" {
					var gErr *gerror.GuildError
					if !gerror.As(err, &gErr) {
						t.Errorf("expected gerror.GuildError, got %T", err)
						return
					}

					if gErr.Code != tt.errCode {
						t.Errorf("expected error code %v, got %v", tt.errCode, gErr.Code)
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestThemeManager_ExportTheme(t *testing.T) {
	tests := []struct {
		name      string
		themeName string
		setup     func(*ThemeManager)
		wantErr   bool
		errCode   gerror.ErrorCode
	}{
		{
			name:      "exports_existing_theme",
			themeName: "claude-code-dark",
			setup:     nil,
			wantErr:   false,
		},
		{
			name:      "fails_for_nonexistent_theme",
			themeName: "nonexistent-theme",
			setup:     nil,
			wantErr:   true,
			errCode:   ui.ErrCodeUIThemeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tm := NewThemeManager()

			if tt.setup != nil {
				tt.setup(tm)
			}

			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "exported-theme.json")

			err := tm.ExportTheme(ctx, tt.themeName, outputPath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					return
				}

				if tt.errCode != "" {
					var gErr *gerror.GuildError
					if !gerror.As(err, &gErr) {
						t.Errorf("expected gerror.GuildError, got %T", err)
						return
					}

					if gErr.Code != tt.errCode {
						t.Errorf("expected error code %v, got %v", tt.errCode, gErr.Code)
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
					return
				}

				// Verify file was created
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Error("exported theme file should exist")
				}
			}
		})
	}
}

func TestThemeManager_ThreadSafety(t *testing.T) {
	tm := NewThemeManager()
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Test concurrent theme applications
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				themeName := "claude-code-dark"
				if j%2 == 0 {
					themeName = "claude-code-light"
				}

				// Apply theme
				tm.ApplyTheme(ctx, themeName)

				// Get current theme
				tm.GetCurrentTheme()

				// Get component style
				tm.GetComponent("button.primary")

				// Get agent style
				tm.GetAgentStyle("agent-1")

				// List themes
				tm.ListThemes()
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is consistent
	current := tm.GetCurrentTheme()
	if current == nil {
		t.Error("current theme should not be nil after concurrent operations")
	}
}

func TestThemeManager_AddObserver(t *testing.T) {
	tm := NewThemeManager()

	called := false
	observer := &testThemeObserver{
		onThemeChanged: func(oldTheme, newTheme *Theme) error {
			called = true
			return nil
		},
	}

	tm.AddObserver(observer)

	ctx := context.Background()
	err := tm.ApplyTheme(ctx, "claude-code-light")
	if err != nil {
		t.Fatalf("failed to apply theme: %v", err)
	}

	if !called {
		t.Error("observer should have been called")
	}
}

func TestThemeManager_ObserverError(t *testing.T) {
	tm := NewThemeManager()

	observer := &testThemeObserver{
		onThemeChanged: func(oldTheme, newTheme *Theme) error {
			return gerror.New(gerror.ErrCodeInternal, "observer error", nil)
		},
	}

	tm.AddObserver(observer)

	ctx := context.Background()
	// Should still succeed even if observer fails
	err := tm.ApplyTheme(ctx, "claude-code-light")
	if err != nil {
		t.Errorf("theme application should succeed even with observer error: %v", err)
	}
}

// Test helper
type testThemeObserver struct {
	onThemeChanged func(oldTheme, newTheme *Theme) error
}

func (t *testThemeObserver) OnThemeChanged(oldTheme, newTheme *Theme) error {
	if t.onThemeChanged != nil {
		return t.onThemeChanged(oldTheme, newTheme)
	}
	return nil
}

func TestBuiltinThemes(t *testing.T) {
	tm := NewThemeManager()

	tests := []struct {
		name      string
		themeName string
	}{
		{"claude_code_light_exists", "claude-code-light"},
		{"claude_code_dark_exists", "claude-code-dark"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tm.ApplyTheme(ctx, tt.themeName)
			if err != nil {
				t.Errorf("built-in theme %s should exist: %v", tt.themeName, err)
			}

			theme := tm.GetCurrentTheme()
			if theme == nil {
				t.Errorf("theme %s should be loaded", tt.themeName)
				return
			}

			if theme.Name != tt.themeName {
				t.Errorf("expected theme name %s, got %s", tt.themeName, theme.Name)
			}

			// Verify essential theme properties
			if theme.DisplayName == "" {
				t.Error("theme should have display name")
			}

			if theme.Version == "" {
				t.Error("theme should have version")
			}

			if theme.Author == "" {
				t.Error("theme should have author")
			}
		})
	}
}

func TestThemeColors(t *testing.T) {
	tm := NewThemeManager()
	ctx := context.Background()

	err := tm.ApplyTheme(ctx, "claude-code-dark")
	if err != nil {
		t.Fatalf("failed to apply theme: %v", err)
	}

	theme := tm.GetCurrentTheme()
	if theme == nil {
		t.Fatal("theme should not be nil")
	}

	// Test color scheme completeness
	colors := theme.Colors

	if colors.Primary.Base == "" {
		t.Error("primary color should be set")
	}

	if colors.Background.Base == "" {
		t.Error("background color should be set")
	}

	if colors.Text.Primary == "" {
		t.Error("text primary color should be set")
	}

	// Test dynamic agent color generation
	// Agent colors start empty and are generated on demand
	if colors.AgentColors == nil {
		t.Error("agent colors map should be initialized")
	}

	// Test that agent color generation works
	agentStyle := tm.GetAgentStyle("test-agent")
	// Check if style has any content by checking if it has background color
	if agentStyle.String() == lipgloss.NewStyle().String() {
		t.Error("agent style should not be empty")
	}

	// Verify the agent color was generated and stored
	if _, exists := colors.AgentColors["test-agent"]; !exists {
		t.Error("agent color should have been generated and stored")
	}

	// Verify generated color has required fields
	if testColor, exists := colors.AgentColors["test-agent"]; exists {
		if testColor.Base == "" {
			t.Error("generated agent color should have base color")
		}
		if testColor.Light == "" {
			t.Error("generated agent color should have light color")
		}
		if testColor.Dark == "" {
			t.Error("generated agent color should have dark color")
		}
	}
}
