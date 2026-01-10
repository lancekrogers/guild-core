// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package ui provides UI-specific error codes for Guild Framework
//
// This package defines domain-specific error codes for UI components,
// following the ErrCodeUI<Domain><Detail> naming convention.
//
// All error codes are documented in docs/error-codes.md
package ui

import "github.com/lancekrogers/guild-core/pkg/gerror"

// UI Theme Error Codes
const (
	// Theme management errors
	ErrCodeUIThemeNotFound     = gerror.ErrorCode("UI_THEME_NOT_FOUND")     // Theme does not exist
	ErrCodeUIThemeInvalid      = gerror.ErrorCode("UI_THEME_INVALID")       // Theme configuration invalid
	ErrCodeUIThemeLoadFailed   = gerror.ErrorCode("UI_THEME_LOAD_FAILED")   // Failed to load theme file
	ErrCodeUIThemeApplyFailed  = gerror.ErrorCode("UI_THEME_APPLY_FAILED")  // Failed to apply theme
	ErrCodeUIThemeExportFailed = gerror.ErrorCode("UI_THEME_EXPORT_FAILED") // Failed to export theme
)

// UI Animation Error Codes
const (
	// Animation system errors
	ErrCodeUIAnimationNotFound    = gerror.ErrorCode("UI_ANIMATION_NOT_FOUND")    // Animation preset not found
	ErrCodeUIAnimationInvalid     = gerror.ErrorCode("UI_ANIMATION_INVALID")      // Animation configuration invalid
	ErrCodeUIAnimationStartFailed = gerror.ErrorCode("UI_ANIMATION_START_FAILED") // Failed to start animation
	ErrCodeUIAnimationStopFailed  = gerror.ErrorCode("UI_ANIMATION_STOP_FAILED")  // Failed to stop animation
	ErrCodeUIAnimationTimeout     = gerror.ErrorCode("UI_ANIMATION_TIMEOUT")      // Animation timed out
	ErrCodeUITimelineInvalid      = gerror.ErrorCode("UI_TIMELINE_INVALID")       // Timeline configuration invalid
	ErrCodeUITimelineCreateFailed = gerror.ErrorCode("UI_TIMELINE_CREATE_FAILED") // Failed to create timeline
)

// UI Shortcuts Error Codes
const (
	// Keyboard shortcut errors
	ErrCodeUIShortcutNotFound       = gerror.ErrorCode("UI_SHORTCUT_NOT_FOUND")       // Shortcut not registered
	ErrCodeUIShortcutInvalid        = gerror.ErrorCode("UI_SHORTCUT_INVALID")         // Shortcut configuration invalid
	ErrCodeUIShortcutConflict       = gerror.ErrorCode("UI_SHORTCUT_CONFLICT")        // Shortcut key conflict
	ErrCodeUIShortcutRegisterFailed = gerror.ErrorCode("UI_SHORTCUT_REGISTER_FAILED") // Failed to register shortcut
	ErrCodeUIContextNotFound        = gerror.ErrorCode("UI_CONTEXT_NOT_FOUND")        // Shortcut context not found
	ErrCodeUIContextInvalid         = gerror.ErrorCode("UI_CONTEXT_INVALID")          // Shortcut context invalid

	// Command palette errors
	ErrCodeUIPaletteSearchFailed   = gerror.ErrorCode("UI_PALETTE_SEARCH_FAILED")   // Command palette search failed
	ErrCodeUIPaletteCommandInvalid = gerror.ErrorCode("UI_PALETTE_COMMAND_INVALID") // Command configuration invalid
)

// UI Component Error Codes
const (
	// Component rendering errors
	ErrCodeUIComponentInvalid      = gerror.ErrorCode("UI_COMPONENT_INVALID")       // Component configuration invalid
	ErrCodeUIComponentRenderFailed = gerror.ErrorCode("UI_COMPONENT_RENDER_FAILED") // Component rendering failed
	ErrCodeUIComponentSizeInvalid  = gerror.ErrorCode("UI_COMPONENT_SIZE_INVALID")  // Component size invalid
	ErrCodeUIComponentStateInvalid = gerror.ErrorCode("UI_COMPONENT_STATE_INVALID") // Component state invalid

	// Button component errors
	ErrCodeUIButtonInvalid        = gerror.ErrorCode("UI_BUTTON_INVALID")         // Button configuration invalid
	ErrCodeUIButtonVariantInvalid = gerror.ErrorCode("UI_BUTTON_VARIANT_INVALID") // Button variant invalid

	// Modal component errors
	ErrCodeUIModalInvalid     = gerror.ErrorCode("UI_MODAL_INVALID")      // Modal configuration invalid
	ErrCodeUIModalSizeInvalid = gerror.ErrorCode("UI_MODAL_SIZE_INVALID") // Modal size invalid

	// Agent badge errors
	ErrCodeUIBadgeInvalid       = gerror.ErrorCode("UI_BADGE_INVALID")         // Badge configuration invalid
	ErrCodeUIBadgeAgentNotFound = gerror.ErrorCode("UI_BADGE_AGENT_NOT_FOUND") // Agent not found for badge

	// Progress bar errors
	ErrCodeUIProgressInvalid      = gerror.ErrorCode("UI_PROGRESS_INVALID")       // Progress configuration invalid
	ErrCodeUIProgressValueInvalid = gerror.ErrorCode("UI_PROGRESS_VALUE_INVALID") // Progress value out of range

	// Chat message errors
	ErrCodeUIMessageInvalid     = gerror.ErrorCode("UI_MESSAGE_INVALID")      // Message configuration invalid
	ErrCodeUIMessageTypeInvalid = gerror.ErrorCode("UI_MESSAGE_TYPE_INVALID") // Message type invalid
)

// UI General Error Codes
const (
	// General UI errors
	ErrCodeUIContextCancelled = gerror.ErrorCode("UI_CONTEXT_CANCELLED")  // Context was cancelled
	ErrCodeUITimeout          = gerror.ErrorCode("UI_TIMEOUT")            // Operation timed out
	ErrCodeUIResourceNotFound = gerror.ErrorCode("UI_RESOURCE_NOT_FOUND") // UI resource not found
	ErrCodeUIResourceInvalid  = gerror.ErrorCode("UI_RESOURCE_INVALID")   // UI resource invalid
	ErrCodeUIInitFailed       = gerror.ErrorCode("UI_INIT_FAILED")        // UI initialization failed
	ErrCodeUIShutdownFailed   = gerror.ErrorCode("UI_SHUTDOWN_FAILED")    // UI shutdown failed
)
