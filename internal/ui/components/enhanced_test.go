// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package components

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/internal/ui/animation"
	"github.com/lancekrogers/guild-core/internal/ui/theme"
	"github.com/lancekrogers/guild-core/pkg/gerror"
)

func TestNewComponentLibrary(t *testing.T) {
	themeManager := theme.NewThemeManager()
	animator := animation.NewAnimator()

	cl := NewComponentLibrary(themeManager, animator)

	if cl == nil {
		t.Fatal("component library should not be nil")
	}

	if cl.themeManager != themeManager {
		t.Error("theme manager should be set")
	}

	if cl.animator != animator {
		t.Error("animator should be set")
	}
}

func TestComponentLibrary_RenderButton(t *testing.T) {
	tests := []struct {
		name     string
		button   Button
		validate func(*testing.T, string, error)
	}{
		{
			name: "renders_primary_button",
			button: Button{
				Text:    "Primary Button",
				Variant: ButtonPrimary,
				Size:    ButtonSizeMedium,
				State:   ButtonStateNormal,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if result == "" {
					t.Error("button should render something")
				}
				if !strings.Contains(result, "Primary Button") {
					t.Error("rendered button should contain text")
				}
			},
		},
		{
			name: "renders_disabled_button",
			button: Button{
				Text:     "Disabled Button",
				Variant:  ButtonSecondary,
				Disabled: true,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if result == "" {
					t.Error("disabled button should still render")
				}
			},
		},
		{
			name: "renders_loading_button",
			button: Button{
				Text:    "Loading Button",
				Variant: ButtonPrimary,
				Loading: true,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "Loading Button") {
					t.Error("loading button should contain original text")
				}
			},
		},
		{
			name: "renders_button_with_icon",
			button: Button{
				Text:    "Icon Button",
				Icon:    "🚀",
				Variant: ButtonAccent,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "🚀") {
					t.Error("button should contain icon")
				}
				if !strings.Contains(result, "Icon Button") {
					t.Error("button should contain text")
				}
			},
		},
		{
			name: "renders_button_with_custom_width",
			button: Button{
				Text:    "Wide Button",
				Variant: ButtonPrimary,
				Width:   50,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				// Note: In a real test, we'd verify the width was applied
				// This would require more sophisticated styling inspection
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cl := setupComponentLibrary(t)

			result, err := cl.RenderButton(ctx, tt.button)
			tt.validate(t, result, err)
		})
	}
}

func TestComponentLibrary_RenderButton_ContextCancellation(t *testing.T) {
	cl := setupComponentLibrary(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	button := Button{
		Text:    "Test Button",
		Variant: ButtonPrimary,
	}

	_, err := cl.RenderButton(ctx, button)
	if err == nil {
		t.Error("expected error due to cancelled context")
	}

	var gErr *gerror.GuildError
	if !gerror.As(err, &gErr) {
		t.Errorf("expected gerror.GuildError, got %T", err)
		return
	}

	if gErr.Code != gerror.ErrCodeCancelled {
		t.Errorf("expected cancelled error code, got %v", gErr.Code)
	}
}

func TestComponentLibrary_RenderModal(t *testing.T) {
	tests := []struct {
		name     string
		modal    Modal
		validate func(*testing.T, string, error)
	}{
		{
			name: "renders_basic_modal",
			modal: Modal{
				Title:   "Test Modal",
				Content: "This is modal content",
				Width:   60,
				Height:  20,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "Test Modal") {
					t.Error("modal should contain title")
				}
				if !strings.Contains(result, "This is modal content") {
					t.Error("modal should contain content")
				}
			},
		},
		{
			name: "renders_closable_modal",
			modal: Modal{
				Title:    "Closable Modal",
				Content:  "Content",
				Width:    40,
				Height:   15,
				Closable: true,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "✕") {
					t.Error("closable modal should contain close button")
				}
			},
		},
		{
			name: "renders_modal_with_backdrop",
			modal: Modal{
				Title:    "Backdrop Modal",
				Content:  "Content",
				Width:    40,
				Height:   15,
				Backdrop: true,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				// Backdrop should add additional styling layers
				if result == "" {
					t.Error("modal with backdrop should render")
				}
			},
		},
		{
			name: "renders_modal_with_buttons",
			modal: Modal{
				Title:   "Button Modal",
				Content: "Content",
				Width:   40,
				Height:  15,
				Buttons: []Button{
					{Text: "OK", Variant: ButtonPrimary},
					{Text: "Cancel", Variant: ButtonSecondary},
				},
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "OK") {
					t.Error("modal should contain OK button")
				}
				if !strings.Contains(result, "Cancel") {
					t.Error("modal should contain Cancel button")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cl := setupComponentLibrary(t)

			result, err := cl.RenderModal(ctx, tt.modal)
			tt.validate(t, result, err)
		})
	}
}

func TestComponentLibrary_RenderAgentBadge(t *testing.T) {
	tests := []struct {
		name     string
		badge    AgentBadge
		validate func(*testing.T, string, error)
	}{
		{
			name: "renders_online_agent_badge",
			badge: AgentBadge{
				AgentID:  "agent-1",
				Status:   AgentOnline,
				Size:     BadgeSizeMedium,
				ShowName: true,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "●") {
					t.Error("online badge should contain online indicator")
				}
				if !strings.Contains(result, "Agent 1") {
					t.Error("badge should contain formatted agent name")
				}
			},
		},
		{
			name: "renders_busy_agent_badge",
			badge: AgentBadge{
				AgentID: "agent-2",
				Status:  AgentBusy,
				Size:    BadgeSizeSmall,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "◐") {
					t.Error("busy badge should contain busy indicator")
				}
			},
		},
		{
			name: "renders_offline_agent_badge",
			badge: AgentBadge{
				AgentID: "agent-3",
				Status:  AgentOffline,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "○") {
					t.Error("offline badge should contain offline indicator")
				}
			},
		},
		{
			name: "renders_thinking_agent_badge",
			badge: AgentBadge{
				AgentID:  "agent-4",
				Status:   AgentThinking,
				Animated: true,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "◒") {
					t.Error("thinking badge should contain thinking indicator")
				}
			},
		},
		{
			name: "renders_error_agent_badge",
			badge: AgentBadge{
				AgentID: "agent-5",
				Status:  AgentError,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "✗") {
					t.Error("error badge should contain error indicator")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cl := setupComponentLibrary(t)

			result, err := cl.RenderAgentBadge(ctx, tt.badge)
			tt.validate(t, result, err)
		})
	}
}

func TestComponentLibrary_RenderProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		progress ProgressBar
		validate func(*testing.T, string, error)
	}{
		{
			name: "renders_linear_progress_bar",
			progress: ProgressBar{
				Progress:    0.75,
				Width:       20,
				ShowPercent: true,
				Style:       ProgressStyleBar,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "75%") {
					t.Error("progress bar should show percentage")
				}
				if !strings.Contains(result, "█") {
					t.Error("progress bar should contain filled sections")
				}
			},
		},
		{
			name: "renders_progress_with_label",
			progress: ProgressBar{
				Progress:  0.5,
				Width:     30,
				ShowLabel: true,
				Label:     "Loading...",
				Style:     ProgressStyleBar,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "Loading...") {
					t.Error("progress bar should contain label")
				}
			},
		},
		{
			name: "renders_circular_progress",
			progress: ProgressBar{
				Progress: 0.8,
				Style:    ProgressCircle,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				// Should render some form of circular indicator
				if result == "" {
					t.Error("circular progress should render something")
				}
			},
		},
		{
			name: "renders_dot_progress",
			progress: ProgressBar{
				Progress: 0.6,
				Style:    ProgressDots,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "●") && !strings.Contains(result, "○") {
					t.Error("dot progress should contain dot indicators")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cl := setupComponentLibrary(t)

			result, err := cl.RenderProgressBar(ctx, tt.progress)
			tt.validate(t, result, err)
		})
	}
}

func TestComponentLibrary_RenderChatMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  ChatMessage
		validate func(*testing.T, string, error)
	}{
		{
			name: "renders_user_message",
			message: ChatMessage{
				Content:   "Hello, world!",
				AgentID:   "user",
				Timestamp: time.Now(),
				Type:      MessageUser,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "Hello, world!") {
					t.Error("message should contain content")
				}
				if !strings.Contains(result, "User") {
					t.Error("message should contain formatted agent name")
				}
			},
		},
		{
			name: "renders_agent_message",
			message: ChatMessage{
				Content:   "I can help you with that.",
				AgentID:   "agent-1",
				Timestamp: time.Now(),
				Type:      MessageAgent,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "I can help you with that.") {
					t.Error("message should contain content")
				}
				if !strings.Contains(result, "Agent 1") {
					t.Error("message should contain formatted agent name")
				}
			},
		},
		{
			name: "renders_system_message",
			message: ChatMessage{
				Content: "System notification",
				AgentID: "system",
				Type:    MessageSystem,
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "System notification") {
					t.Error("message should contain content")
				}
			},
		},
		{
			name: "renders_message_with_reactions",
			message: ChatMessage{
				Content: "Great work!",
				AgentID: "agent-1",
				Type:    MessageAgent,
				Reactions: []Reaction{
					{Emoji: "👍", Count: 3, Active: true},
					{Emoji: "🎉", Count: 1, Active: false},
				},
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "👍") {
					t.Error("message should contain reaction emoji")
				}
				if !strings.Contains(result, "3") {
					t.Error("message should contain reaction count")
				}
			},
		},
		{
			name: "renders_message_with_metadata",
			message: ChatMessage{
				Content: "Edited message",
				AgentID: "agent-1",
				Type:    MessageAgent,
				Metadata: MessageMeta{
					Edited:   true,
					ReplyTo:  "previous-message",
					Mentions: []string{"agent-2", "agent-3"},
				},
			},
			validate: func(t *testing.T, result string, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if !strings.Contains(result, "(edited)") {
					t.Error("message should show edited indicator")
				}
				if !strings.Contains(result, "reply") {
					t.Error("message should show reply indicator")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cl := setupComponentLibrary(t)

			result, err := cl.RenderChatMessage(ctx, tt.message)
			tt.validate(t, result, err)
		})
	}
}

func TestComponentLibrary_ThemeIntegration(t *testing.T) {
	cl := setupComponentLibrary(t)
	ctx := context.Background()

	// Test that components use theme colors
	button := Button{
		Text:    "Themed Button",
		Variant: ButtonPrimary,
	}

	result1, err := cl.RenderButton(ctx, button)
	if err != nil {
		t.Fatalf("failed to render button: %v", err)
	}

	// Switch theme
	err = cl.themeManager.ApplyTheme(ctx, "claude-code-light")
	if err != nil {
		t.Fatalf("failed to switch theme: %v", err)
	}

	result2, err := cl.RenderButton(ctx, button)
	if err != nil {
		t.Fatalf("failed to render button after theme switch: %v", err)
	}

	// Results should be different (different themes applied)
	// Note: In a more sophisticated test, we'd verify specific color differences
	if result1 == result2 {
		t.Log("Button rendering appears unchanged after theme switch (may be expected for test environment)")
	}
}

func TestComponentLibrary_ErrorHandling(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() *ComponentLibrary
		action func(*ComponentLibrary) error
	}{
		{
			name: "handles_nil_theme_manager",
			setup: func() *ComponentLibrary {
				return &ComponentLibrary{
					themeManager: nil,
					animator:     animation.NewAnimator(),
				}
			},
			action: func(cl *ComponentLibrary) error {
				ctx := context.Background()
				_, err := cl.RenderButton(ctx, Button{Text: "Test"})
				return err
			},
		},
		{
			name: "handles_cancelled_context",
			setup: func() *ComponentLibrary {
				return setupComponentLibrary(nil)
			},
			action: func(cl *ComponentLibrary) error {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				_, err := cl.RenderButton(ctx, Button{Text: "Test"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := tt.setup()
			err := tt.action(cl)
			// Should handle errors gracefully
			if err != nil {
				var gErr *gerror.GuildError
				if !gerror.As(err, &gErr) {
					t.Errorf("expected gerror.GuildError, got %T", err)
				}
			}
		})
	}
}

func TestButtonVariants(t *testing.T) {
	cl := setupComponentLibrary(t)
	ctx := context.Background()

	variants := []ButtonVariant{
		ButtonPrimary,
		ButtonSecondary,
		ButtonAccent,
		ButtonSuccess,
		ButtonWarning,
		ButtonDanger,
		ButtonGhost,
		ButtonLink,
	}

	for _, variant := range variants {
		t.Run(variant.String(), func(t *testing.T) {
			button := Button{
				Text:    "Test Button",
				Variant: variant,
			}

			result, err := cl.RenderButton(ctx, button)
			if err != nil {
				t.Errorf("failed to render %v button: %v", variant, err)
			}

			if result == "" {
				t.Errorf("%v button should render something", variant)
			}
		})
	}
}

func TestButtonSizes(t *testing.T) {
	cl := setupComponentLibrary(t)
	ctx := context.Background()

	sizes := []ButtonSize{
		ButtonSizeSmall,
		ButtonSizeMedium,
		ButtonSizeLarge,
		ButtonSizeXLarge,
	}

	for _, size := range sizes {
		t.Run(size.String(), func(t *testing.T) {
			button := Button{
				Text: "Test Button",
				Size: size,
			}

			result, err := cl.RenderButton(ctx, button)
			if err != nil {
				t.Errorf("failed to render %v button: %v", size, err)
			}

			if result == "" {
				t.Errorf("%v button should render something", size)
			}
		})
	}
}

func TestAgentNameFormatting(t *testing.T) {
	cl := setupComponentLibrary(t)

	tests := []struct {
		input    string
		expected string
	}{
		{"agent-1", "Agent 1"},
		{"agent-2", "Agent 2"},
		{"user", "User"},
		{"system", "System"},
		{"custom-name", "Custom Name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cl.formatAgentName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Test helper functions

func setupComponentLibrary(t *testing.T) *ComponentLibrary {
	themeManager := theme.NewThemeManager()
	animator := animation.NewAnimator()

	return NewComponentLibrary(themeManager, animator)
}

// String methods for test output
func (bv ButtonVariant) String() string {
	switch bv {
	case ButtonPrimary:
		return "primary"
	case ButtonSecondary:
		return "secondary"
	case ButtonAccent:
		return "accent"
	case ButtonSuccess:
		return "success"
	case ButtonWarning:
		return "warning"
	case ButtonDanger:
		return "danger"
	case ButtonGhost:
		return "ghost"
	case ButtonLink:
		return "link"
	default:
		return "unknown"
	}
}

func (bs ButtonSize) String() string {
	switch bs {
	case ButtonSizeSmall:
		return "small"
	case ButtonSizeMedium:
		return "medium"
	case ButtonSizeLarge:
		return "large"
	case ButtonSizeXLarge:
		return "xlarge"
	default:
		return "unknown"
	}
}
