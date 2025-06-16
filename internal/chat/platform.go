// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"runtime"
)

// Platform represents the operating system
type Platform int

const (
	PlatformMacOS Platform = iota
	PlatformLinux
	PlatformWindows
	PlatformUnknown
)

// DetectPlatform returns the current operating system platform
func DetectPlatform() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMacOS
	case "linux":
		return PlatformLinux
	case "windows":
		return PlatformWindows
	default:
		return PlatformUnknown
	}
}

// String returns the string representation of the platform
func (p Platform) String() string {
	switch p {
	case PlatformMacOS:
		return "macOS"
	case PlatformLinux:
		return "Linux"
	case PlatformWindows:
		return "Windows"
	default:
		return "Unknown"
	}
}

// IsMacOS returns true if the platform is macOS
func (p Platform) IsMacOS() bool {
	return p == PlatformMacOS
}

// IsLinux returns true if the platform is Linux
func (p Platform) IsLinux() bool {
	return p == PlatformLinux
}

// IsWindows returns true if the platform is Windows
func (p Platform) IsWindows() bool {
	return p == PlatformWindows
}

// GetModifierKey returns the primary modifier key for the platform
func (p Platform) GetModifierKey() string {
	if p.IsMacOS() {
		return "alt"
	}
	return "ctrl"
}

// GetModifierDisplay returns the display string for the modifier key
func (p Platform) GetModifierDisplay() string {
	if p.IsMacOS() {
		return "⌥" // Option/Alt symbol
	}
	return "Ctrl"
}

// GetSecondaryModifierKey returns the secondary modifier key for the platform
func (p Platform) GetSecondaryModifierKey() string {
	if p.IsMacOS() {
		return "cmd"
	}
	return "alt"
}

// GetSecondaryModifierDisplay returns the display string for the secondary modifier key
func (p Platform) GetSecondaryModifierDisplay() string {
	if p.IsMacOS() {
		return "⌘" // Command symbol
	}
	return "Alt"
}
