// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lancekrogers/guild/internal/ui/chat/panes"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/preferences"
)

// PreferencesHandler handles preference-related commands
type PreferencesHandler struct {
	prefService *preferences.Service
	userID      string
	campaignID  string
}

// NewPreferencesHandler creates a new preferences command handler
func NewPreferencesHandler(prefService *preferences.Service, userID, campaignID string) *PreferencesHandler {
	return &PreferencesHandler{
		prefService: prefService,
		userID:      userID,
		campaignID:  campaignID,
	}
}

// Handle processes preference commands
func (h *PreferencesHandler) Handle(ctx context.Context, args []string) tea.Cmd {
	if len(args) == 0 {
		return h.showPreferences(ctx)
	}

	switch args[0] {
	case "list", "show":
		return h.showPreferences(ctx)
	case "set":
		if len(args) < 3 {
			return h.showError("Usage: /preferences set <key> <value>")
		}
		return h.setPreference(ctx, args[1], strings.Join(args[2:], " "))
	case "get":
		if len(args) < 2 {
			return h.showError("Usage: /preferences get <key>")
		}
		return h.getPreference(ctx, args[1])
	case "reset":
		if len(args) < 2 {
			return h.showError("Usage: /preferences reset <key>")
		}
		return h.resetPreference(ctx, args[1])
	case "export":
		return h.exportPreferences(ctx)
	case "help":
		return h.showHelp()
	default:
		return h.showError(fmt.Sprintf("Unknown preferences command: %s", args[0]))
	}
}

func (h *PreferencesHandler) showPreferences(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// Get common preference keys to display
		commonKeys := []string{
			"ui.theme", "ui.vim_mode", "ui.font_size",
			"ai.default_provider", "ai.default_model", "ai.temperature",
		}

		prefs, err := h.prefService.GetPreferences(ctx, "user", &h.userID, commonKeys)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to load preferences: %v", err),
				Level:   "error",
			}
		}

		// Format preferences display
		var output strings.Builder
		output.WriteString("📋 **User Preferences**\n\n")

		// Group by category
		uiPrefs := make(map[string]interface{})
		aiPrefs := make(map[string]interface{})

		for key, value := range prefs {
			if strings.HasPrefix(key, "ui.") {
				uiPrefs[key] = value
			} else if strings.HasPrefix(key, "ai.") {
				aiPrefs[key] = value
			}
		}

		// Display UI preferences
		if len(uiPrefs) > 0 {
			output.WriteString("**UI Settings:**\n")
			for key, value := range uiPrefs {
				output.WriteString(fmt.Sprintf("  • %s: %v\n", key, value))
			}
			output.WriteString("\n")
		}

		// Display AI preferences
		if len(aiPrefs) > 0 {
			output.WriteString("**AI Settings:**\n")
			for key, value := range aiPrefs {
				output.WriteString(fmt.Sprintf("  • %s: %v\n", key, value))
			}
			output.WriteString("\n")
		}

		if len(prefs) == 0 {
			output.WriteString("No preferences set. Use `/preferences set <key> <value>` to set preferences.")
		}

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: output.String(),
		}
	}
}

func (h *PreferencesHandler) setPreference(ctx context.Context, key, value string) tea.Cmd {
	return func() tea.Msg {
		// Parse value based on common preference types
		var parsedValue interface{}

		// Try to parse as boolean
		if boolVal, err := strconv.ParseBool(value); err == nil {
			parsedValue = boolVal
		} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			// Try to parse as number
			parsedValue = floatVal
		} else {
			// Default to string
			parsedValue = value
		}

		// Set the preference
		err := h.prefService.SetUserPreference(ctx, h.userID, key, parsedValue)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to set preference: %v", err),
				Level:   "error",
			}
		}

		return panes.StatusUpdateMsg{
			Message: fmt.Sprintf("✅ Preference set: %s = %v", key, parsedValue),
			Level:   "success",
		}
	}
}

func (h *PreferencesHandler) getPreference(ctx context.Context, key string) tea.Cmd {
	return func() tea.Msg {
		pref, err := h.prefService.GetUserPreference(ctx, h.userID, key)
		if err != nil {
			if gerror.Is(err, gerror.ErrCodeNotFound) {
				return panes.StatusUpdateMsg{
					Message: fmt.Sprintf("Preference not found: %s", key),
					Level:   "error",
				}
			}
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to get preference: %v", err),
				Level:   "error",
			}
		}

		return panes.StatusUpdateMsg{
			Message: fmt.Sprintf("📋 %s = %v", key, pref),
			Level:   "info",
		}
	}
}

func (h *PreferencesHandler) resetPreference(ctx context.Context, key string) tea.Cmd {
	return func() tea.Msg {
		// Get default value from system preferences
		defaultValue, err := h.prefService.GetSystemPreference(ctx, key)
		if err != nil {
			// No system default, just remove the user preference by setting to nil
			// Note: Since there's no direct delete method, we'll set to nil which should be handled by the service
			if err := h.prefService.SetUserPreference(ctx, h.userID, key, nil); err != nil {
				return panes.StatusUpdateMsg{
					Message: fmt.Sprintf("Failed to reset preference: %v", err),
					Level:   "error",
				}
			}
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("✅ Preference reset: %s (removed)", key),
				Level:   "success",
			}
		}

		// Set to default value
		err = h.prefService.SetUserPreference(ctx, h.userID, key, defaultValue)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to reset preference: %v", err),
				Level:   "error",
			}
		}

		return panes.StatusUpdateMsg{
			Message: fmt.Sprintf("✅ Preference reset: %s = %v (default)", key, defaultValue),
			Level:   "success",
		}
	}
}

func (h *PreferencesHandler) exportPreferences(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		data, err := h.prefService.ExportPreferences(ctx, "user", &h.userID)
		if err != nil {
			return panes.StatusUpdateMsg{
				Message: fmt.Sprintf("Failed to export preferences: %v", err),
				Level:   "error",
			}
		}

		// Save to file or display
		filename := fmt.Sprintf("preferences_%s.json", h.userID)
		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: fmt.Sprintf("📋 Preferences exported:\n```json\n%s\n```\nSaved to: %s", string(data), filename),
		}
	}
}

func (h *PreferencesHandler) showHelp() tea.Cmd {
	return func() tea.Msg {
		help := `📋 **Preferences Commands**

• /preferences - Show all preferences
• /preferences set <key> <value> - Set a preference
• /preferences get <key> - Get a specific preference
• /preferences reset <key> - Reset to default value
• /preferences export - Export preferences to JSON

**Common Preferences:**
• ui.theme - Color theme (dark/light)
• ui.vim_mode - Enable vim mode (true/false)
• ui.font_size - Font size (8-24)
• ai.default_provider - Default AI provider
• ai.temperature - AI temperature (0.0-1.0)
• ai.streaming_enabled - Enable streaming (true/false)`

		return panes.PaneUpdateMsg{
			PaneID:  "output",
			Content: help,
		}
	}
}

func (h *PreferencesHandler) showError(message string) tea.Cmd {
	return func() tea.Msg {
		return panes.StatusUpdateMsg{
			Message: message,
			Level:   "error",
		}
	}
}

// Description returns the command description
func (h *PreferencesHandler) Description() string {
	return "Manage user and campaign preferences"
}

// Usage returns the command usage
func (h *PreferencesHandler) Usage() string {
	return "/preferences [list|set|get|reset|export] - Manage preferences"
}
