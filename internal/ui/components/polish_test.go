package components

import (
	"context"
	"testing"

	"github.com/guild-framework/guild-core/internal/ui/chat/common/utils"
	"go.uber.org/zap"
)

func TestLoadingIndicator(t *testing.T) {
	ctx := context.Background()
	theme := utils.NewClaudeCodeStyles()
	logger := zap.NewNop()

	tests := []struct {
		name  string
		style LoadingStyle
	}{
		{"Spinner", LoadingSpinner},
		{"Progress", LoadingProgress},
		{"Dots", LoadingDots},
		{"Pulse", LoadingPulse},
		{"Elastic", LoadingElastic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			li, err := NewLoadingIndicator(ctx, tt.style, "Loading...", theme, logger)
			if err != nil {
				t.Fatalf("NewLoadingIndicator failed: %v", err)
			}

			// Test initial render
			view, err := li.View(ctx)
			if err != nil {
				t.Fatalf("View failed: %v", err)
			}
			if view == "" {
				t.Errorf("LoadingIndicator.View() returned empty string for %s", tt.name)
			}

			// Test update
			err = li.Update(ctx)
			if err != nil {
				t.Fatalf("Update failed: %v", err)
			}

			viewAfterUpdate, err := li.View(ctx)
			if err != nil {
				t.Fatalf("View after update failed: %v", err)
			}
			if viewAfterUpdate == "" {
				t.Errorf("LoadingIndicator.View() returned empty after update for %s", tt.name)
			}

			// Test progress (for progress style)
			if tt.style == LoadingProgress {
				err = li.SetProgress(ctx, 0.5)
				if err != nil {
					t.Fatalf("SetProgress failed: %v", err)
				}

				progressView, err := li.View(ctx)
				if err != nil {
					t.Fatalf("View with progress failed: %v", err)
				}
				if progressView == "" {
					t.Error("LoadingIndicator with progress returned empty view")
				}
			}
		})
	}
}

func TestLoadingIndicatorWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	theme := utils.NewClaudeCodeStyles()
	logger := zap.NewNop()

	// Test creation with cancelled context
	_, err := NewLoadingIndicator(ctx, LoadingSpinner, "Test", theme, logger)
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

func TestStatusIndicator(t *testing.T) {
	ctx := context.Background()
	theme := utils.NewClaudeCodeStyles()
	logger := zap.NewNop()

	si, err := NewStatusIndicator(ctx, theme, logger)
	if err != nil {
		t.Fatalf("NewStatusIndicator failed: %v", err)
	}

	statuses := []struct {
		status  Status
		message string
		icon    string
	}{
		{StatusSuccess, "Operation completed", "✓"},
		{StatusError, "Operation failed", "✗"},
		{StatusWarning, "Check required", "⚠"},
		{StatusInfo, "Information", "ℹ"},
		{StatusLoading, "Processing", "⠋"}, // First frame of spinner
	}

	for _, s := range statuses {
		t.Run(s.message, func(t *testing.T) {
			err := si.SetStatus(ctx, s.status, s.message)
			if err != nil {
				t.Fatalf("SetStatus failed: %v", err)
			}

			view, err := si.View(ctx)
			if err != nil {
				t.Fatalf("View failed: %v", err)
			}

			if view == "" {
				t.Errorf("StatusIndicator.View() returned empty for %s", s.message)
			}

			// For loading status, test update
			if s.status == StatusLoading {
				err = si.Update(ctx)
				if err != nil {
					t.Fatalf("Update failed: %v", err)
				}
			}
		})
	}
}

func TestTooltip(t *testing.T) {
	ctx := context.Background()
	theme := utils.NewClaudeCodeStyles()
	logger := zap.NewNop()

	tooltip, err := NewTooltip(ctx, "Help text", TooltipAbove, theme, logger)
	if err != nil {
		t.Fatalf("NewTooltip failed: %v", err)
	}

	// Initially should not be visible
	view, err := tooltip.View(ctx)
	if err != nil {
		t.Fatalf("View failed: %v", err)
	}
	if view != "" {
		t.Error("Tooltip should not be visible initially")
	}

	// Show tooltip
	err = tooltip.Show(ctx)
	if err != nil {
		t.Fatalf("Show failed: %v", err)
	}

	// Still shouldn't be visible immediately (has delay)
	view, err = tooltip.View(ctx)
	if err != nil {
		t.Fatalf("View after Show failed: %v", err)
	}
	if view != "" {
		t.Error("Tooltip should not be visible immediately after Show()")
	}

	// Simulate timer completion
	tooltip.mu.Lock()
	tooltip.visible = true
	tooltip.mu.Unlock()

	view, err = tooltip.View(ctx)
	if err != nil {
		t.Fatalf("View when visible failed: %v", err)
	}
	if view == "" {
		t.Error("Tooltip should be visible after delay")
	}

	// Hide tooltip
	err = tooltip.Hide(ctx)
	if err != nil {
		t.Fatalf("Hide failed: %v", err)
	}

	view, err = tooltip.View(ctx)
	if err != nil {
		t.Fatalf("View after Hide failed: %v", err)
	}
	if view != "" {
		t.Error("Tooltip should not be visible after Hide()")
	}

	// Cleanup
	tooltip.Cleanup()
}

func TestHapticFeedback(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	hf, err := NewHapticFeedbackWithLogger(ctx, true, logger)
	if err != nil {
		t.Fatalf("NewHapticFeedbackWithLogger failed: %v", err)
	}

	// Test success feedback
	feedback, err := hf.Success(ctx)
	if err != nil {
		t.Fatalf("Success failed: %v", err)
	}
	if feedback == "" {
		t.Error("HapticFeedback.Success() returned empty when enabled")
	}

	// Test error feedback
	feedback, err = hf.Error(ctx)
	if err != nil {
		t.Fatalf("Error failed: %v", err)
	}
	if feedback == "" {
		t.Error("HapticFeedback.Error() returned empty when enabled")
	}

	// Test info feedback
	feedback, err = hf.Info(ctx)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if feedback == "" {
		t.Error("HapticFeedback.Info() returned empty when enabled")
	}

	// Test disabled feedback
	hfDisabled, err := NewHapticFeedbackWithLogger(ctx, false, logger)
	if err != nil {
		t.Fatalf("NewHapticFeedbackWithLogger (disabled) failed: %v", err)
	}

	feedback, err = hfDisabled.Success(ctx)
	if err != nil {
		t.Fatalf("Success (disabled) failed: %v", err)
	}
	if feedback != "" {
		t.Error("HapticFeedback.Success() should return empty when disabled")
	}
}

func TestAccessibilityManager(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	am, err := NewAccessibilityManager(ctx, logger)
	if err != nil {
		t.Fatalf("NewAccessibilityManager failed: %v", err)
	}

	// Test initial state
	info, err := am.GetAccessibilityInfo(ctx)
	if err != nil {
		t.Fatalf("GetAccessibilityInfo failed: %v", err)
	}
	if info == "" {
		t.Error("AccessibilityManager.GetAccessibilityInfo() returned empty")
	}

	// Enable features
	theme := utils.NewClaudeCodeStyles()
	err = am.EnableHighContrast(ctx, theme)
	if err != nil {
		t.Fatalf("EnableHighContrast failed: %v", err)
	}

	err = am.EnableReducedMotion(ctx, nil)
	if err != nil {
		t.Fatalf("EnableReducedMotion failed: %v", err)
	}

	err = am.EnableScreenReader(ctx)
	if err != nil {
		t.Fatalf("EnableScreenReader failed: %v", err)
	}

	err = am.SetFontSize(ctx, 18)
	if err != nil {
		t.Fatalf("SetFontSize failed: %v", err)
	}

	// Check updated info
	updatedInfo, err := am.GetAccessibilityInfo(ctx)
	if err != nil {
		t.Fatalf("GetAccessibilityInfo (updated) failed: %v", err)
	}
	if updatedInfo == info {
		t.Error("AccessibilityManager info should change after enabling features")
	}

	// Test getters
	if !am.IsHighContrastEnabled() {
		t.Error("High contrast should be enabled")
	}
	if !am.IsReducedMotionEnabled() {
		t.Error("Reduced motion should be enabled")
	}
	if !am.IsScreenReaderEnabled() {
		t.Error("Screen reader should be enabled")
	}
	if am.GetFontSize() != 18 {
		t.Error("Font size should be 18")
	}
}

func TestPolishedComponents(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	pc, err := NewPolishedComponents(ctx, logger)
	if err != nil {
		t.Fatalf("NewPolishedComponents failed: %v", err)
	}

	if pc == nil {
		t.Fatal("NewPolishedComponents returned nil")
	}

	if pc.theme == nil {
		t.Error("PolishedComponents theme is nil")
	}

	if pc.animator == nil {
		t.Error("PolishedComponents animator is nil")
	}

	if pc.feedback == nil {
		t.Error("PolishedComponents feedback is nil")
	}

	// Test shutdown
	err = pc.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Test with nil context
	_, err = NewPolishedComponents(nil, logger)
	if err == nil {
		t.Error("NewPolishedComponents should error with nil context")
	}
}
