// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package theme provides comprehensive theme management for Guild Framework UI
//
// This package implements the theme management requirements identified in performance optimization,
// Agent 1 task, providing:
//   - Centralized theme management with hot-swapping capabilities
//   - Claude Code visual parity with professional styling
//   - Agent-specific color coding for multi-agent identification
//   - Comprehensive component styling system
//
// The package follows Guild's architectural patterns:
//   - Context-first error handling with gerror
//   - Interface-driven design for testability
//   - Registry pattern for theme plugins
//   - Observability integration
//
// Example usage:
//
//	// Initialize theme manager
//	manager := NewThemeManager()
//
//	// Apply Claude Code theme
//	err := manager.ApplyTheme(ctx, "claude-code-light")
//
//	// Get styled component
//	buttonStyle := manager.GetComponent("button").Primary()
//
//	// Create agent-specific styling
//	agentStyle := manager.GetAgentStyle("agent-1")
package theme

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/lancekrogers/guild-core/internal/ui"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"go.uber.org/zap"
)

// Package version for compatibility tracking
const (
	Version     = "1.0.0"
	APIVersion  = "v1"
	PackageName = "theme"
)

// ThemeManager provides centralized theme management for Guild UI
type ThemeManager struct {
	currentTheme *Theme
	themes       map[string]*Theme
	registry     ThemeRegistry
	observers    []ThemeObserver
	mu           sync.RWMutex
	logger       *zap.Logger
}

// Theme represents a complete UI theme configuration
type Theme struct {
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	Version     string      `json:"version"`
	Author      string      `json:"author"`
	Colors      ColorScheme `json:"colors"`
	Typography  Typography  `json:"typography"`
	Spacing     Spacing     `json:"spacing"`
	Components  Components  `json:"components"`
	Animations  Animations  `json:"animations"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ColorScheme defines the complete color palette
type ColorScheme struct {
	// Primary brand colors
	Primary   Color `json:"primary"`   // Guild brand primary
	Secondary Color `json:"secondary"` // Guild brand secondary
	Accent    Color `json:"accent"`    // Interactive elements

	// Semantic colors
	Success Color `json:"success"` // Success states
	Warning Color `json:"warning"` // Warning states
	Error   Color `json:"error"`   // Error states
	Info    Color `json:"info"`    // Information states

	// Neutral colors
	Background Color `json:"background"` // Main background
	Surface    Color `json:"surface"`    // Card/panel backgrounds
	Border     Color `json:"border"`     // Border colors
	Text       Text  `json:"text"`       // Text color variants

	// Agent-specific colors (Guild feature) - dynamically generated
	AgentColors map[string]Color `json:"agent_colors"` // Per-agent identification, auto-generated
}

// Color represents a color with various shades
type Color struct {
	Base    string `json:"base"`    // Base color
	Light   string `json:"light"`   // Lighter variant
	Dark    string `json:"dark"`    // Darker variant
	Muted   string `json:"muted"`   // Muted variant
	Inverse string `json:"inverse"` // Inverse/contrast color
}

// Text represents text color variants
type Text struct {
	Primary   string `json:"primary"`   // Primary text
	Secondary string `json:"secondary"` // Secondary text
	Muted     string `json:"muted"`     // Muted text
	Disabled  string `json:"disabled"`  // Disabled text
	Inverse   string `json:"inverse"`   // Inverse text
	Link      string `json:"link"`      // Link text
}

// Typography defines font styles and sizes
type Typography struct {
	FontFamily  string          `json:"font_family"`
	Scale       TypographyScale `json:"scale"`
	Weights     FontWeights     `json:"weights"`
	LineHeights LineHeights     `json:"line_heights"`
}

// TypographyScale defines the font size scale
type TypographyScale struct {
	XS   int `json:"xs"`   // Extra small
	SM   int `json:"sm"`   // Small
	Base int `json:"base"` // Base size
	LG   int `json:"lg"`   // Large
	XL   int `json:"xl"`   // Extra large
	XXL  int `json:"xxl"`  // Double extra large
}

// FontWeights defines available font weights
type FontWeights struct {
	Light  int `json:"light"`  // Light weight
	Normal int `json:"normal"` // Normal weight
	Medium int `json:"medium"` // Medium weight
	Bold   int `json:"bold"`   // Bold weight
	Heavy  int `json:"heavy"`  // Heavy weight
}

// LineHeights defines line height variants
type LineHeights struct {
	Tight  float64 `json:"tight"`  // Tight line height
	Normal float64 `json:"normal"` // Normal line height
	Loose  float64 `json:"loose"`  // Loose line height
}

// Spacing defines spacing scale
type Spacing struct {
	XS  int `json:"xs"`  // Extra small spacing
	SM  int `json:"sm"`  // Small spacing
	MD  int `json:"md"`  // Medium spacing
	LG  int `json:"lg"`  // Large spacing
	XL  int `json:"xl"`  // Extra large spacing
	XXL int `json:"xxl"` // Double extra large spacing
}

// Components defines styled components
type Components struct {
	Button ButtonStyles `json:"button"`
	Input  InputStyles  `json:"input"`
	Modal  ModalStyles  `json:"modal"`
	Chat   ChatStyles   `json:"chat"`
	Kanban KanbanStyles `json:"kanban"`
	Agent  AgentStyles  `json:"agent"`
}

// ButtonStyles defines button styling variants
type ButtonStyles struct {
	Primary   ComponentStyle `json:"primary"`
	Secondary ComponentStyle `json:"secondary"`
	Accent    ComponentStyle `json:"accent"`
	Success   ComponentStyle `json:"success"`
	Warning   ComponentStyle `json:"warning"`
	Error     ComponentStyle `json:"error"`
	Ghost     ComponentStyle `json:"ghost"`
}

// InputStyles defines input styling variants
type InputStyles struct {
	Default  ComponentStyle `json:"default"`
	Focus    ComponentStyle `json:"focus"`
	Error    ComponentStyle `json:"error"`
	Success  ComponentStyle `json:"success"`
	Disabled ComponentStyle `json:"disabled"`
}

// ModalStyles defines modal styling
type ModalStyles struct {
	Overlay ComponentStyle `json:"overlay"`
	Content ComponentStyle `json:"content"`
	Header  ComponentStyle `json:"header"`
	Footer  ComponentStyle `json:"footer"`
}

// ChatStyles defines chat-specific styling
type ChatStyles struct {
	Message     ComponentStyle `json:"message"`
	UserMessage ComponentStyle `json:"user_message"`
	AIMessage   ComponentStyle `json:"ai_message"`
	SystemMsg   ComponentStyle `json:"system_message"`
	Input       ComponentStyle `json:"input"`
	Toolbar     ComponentStyle `json:"toolbar"`
}

// KanbanStyles defines kanban board styling
type KanbanStyles struct {
	Board  ComponentStyle `json:"board"`
	Column ComponentStyle `json:"column"`
	Card   ComponentStyle `json:"card"`
	Header ComponentStyle `json:"header"`
}

// AgentStyles defines agent-specific styling
type AgentStyles struct {
	Badge      ComponentStyle `json:"badge"`
	Avatar     ComponentStyle `json:"avatar"`
	Status     ComponentStyle `json:"status"`
	Background ComponentStyle `json:"background"`
}

// ComponentStyle defines styling for a component
type ComponentStyle struct {
	Background   string            `json:"background"`
	Foreground   string            `json:"foreground"`
	Border       string            `json:"border"`
	BorderRadius int               `json:"border_radius"`
	Padding      map[string]int    `json:"padding"`
	Margin       map[string]int    `json:"margin"`
	Typography   ComponentTypog    `json:"typography"`
	States       map[string]string `json:"states"` // hover, active, disabled
}

// ComponentTypog defines typography for components
type ComponentTypog struct {
	FontSize      int     `json:"font_size"`
	FontWeight    int     `json:"font_weight"`
	LineHeight    float64 `json:"line_height"`
	LetterSpacing float64 `json:"letter_spacing"`
}

// Animations defines animation settings
type Animations struct {
	Enabled     bool              `json:"enabled"`
	Duration    AnimationDuration `json:"duration"`
	Easing      AnimationEasing   `json:"easing"`
	Transitions AnimationTypes    `json:"transitions"`
}

// AnimationDuration defines animation timing
type AnimationDuration struct {
	Fast   int `json:"fast"`   // Fast animations
	Normal int `json:"normal"` // Normal speed
	Slow   int `json:"slow"`   // Slow animations
}

// AnimationEasing defines easing functions
type AnimationEasing struct {
	Linear    string `json:"linear"`
	EaseIn    string `json:"ease_in"`
	EaseOut   string `json:"ease_out"`
	EaseInOut string `json:"ease_in_out"`
}

// AnimationTypes defines enabled animation types
type AnimationTypes struct {
	Fade     bool `json:"fade"`
	Slide    bool `json:"slide"`
	Scale    bool `json:"scale"`
	Rotation bool `json:"rotation"`
}

// ThemeRegistry manages theme registration and discovery
type ThemeRegistry interface {
	RegisterTheme(theme *Theme) error
	GetTheme(name string) (*Theme, error)
	ListThemes() []string
	UnregisterTheme(name string) error
}

// ThemeObserver receives theme change notifications
type ThemeObserver interface {
	OnThemeChanged(oldTheme, newTheme *Theme) error
}

// NewThemeManager creates a new theme manager with built-in themes
func NewThemeManager() *ThemeManager {
	logger, _ := zap.NewDevelopment()

	tm := &ThemeManager{
		themes:    make(map[string]*Theme),
		observers: make([]ThemeObserver, 0),
		logger:    logger.Named("theme-manager"),
	}

	// Register built-in themes
	tm.registerBuiltinThemes()

	// Set default theme
	if theme, exists := tm.themes["claude-code-light"]; exists {
		tm.currentTheme = theme
	}

	return tm
}

// ApplyTheme applies a theme by name
func (tm *ThemeManager) ApplyTheme(ctx context.Context, themeName string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("theme-manager").
			WithOperation("ApplyTheme")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	theme, exists := tm.themes[themeName]
	if !exists {
		return gerror.New(ui.ErrCodeUIThemeNotFound, fmt.Sprintf("theme '%s' not found", themeName), nil).
			WithComponent("theme-manager").
			WithOperation("ApplyTheme")
	}

	oldTheme := tm.currentTheme
	tm.currentTheme = theme

	// Notify observers
	for _, observer := range tm.observers {
		if err := observer.OnThemeChanged(oldTheme, theme); err != nil {
			tm.logger.Warn("Theme observer failed", zap.Error(err))
		}
	}

	tm.logger.Info("Theme applied successfully",
		zap.String("theme", themeName),
		zap.String("previous_theme", getThemeName(oldTheme)))

	return nil
}

// GetCurrentTheme returns the currently active theme
func (tm *ThemeManager) GetCurrentTheme() *Theme {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.currentTheme
}

// GetComponent returns styled component by name
func (tm *ThemeManager) GetComponent(componentName string) lipgloss.Style {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.currentTheme == nil {
		return lipgloss.NewStyle()
	}

	// Convert theme components to lipgloss styles
	switch componentName {
	case "button.primary":
		return tm.componentToStyle(tm.currentTheme.Components.Button.Primary)
	case "button.secondary":
		return tm.componentToStyle(tm.currentTheme.Components.Button.Secondary)
	case "input.default":
		return tm.componentToStyle(tm.currentTheme.Components.Input.Default)
	case "chat.message":
		return tm.componentToStyle(tm.currentTheme.Components.Chat.Message)
	default:
		return lipgloss.NewStyle()
	}
}

// GetAgentStyle returns agent-specific styling, generating colors dynamically
func (tm *ThemeManager) GetAgentStyle(agentID string) lipgloss.Style {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.currentTheme == nil {
		return lipgloss.NewStyle()
	}

	// Get or generate agent color
	agentColor := tm.getOrGenerateAgentColor(agentID)

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(agentColor.Base)).
		Background(lipgloss.Color(agentColor.Light)).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(agentColor.Dark))
}

// getOrGenerateAgentColor gets existing agent color or generates a new one
func (tm *ThemeManager) getOrGenerateAgentColor(agentID string) Color {
	// Check if color already exists
	if agentColor, exists := tm.currentTheme.Colors.AgentColors[agentID]; exists {
		return agentColor
	}

	// Generate new color based on agent ID
	agentColor := tm.generateAgentColor(agentID)

	// Store for future use
	if tm.currentTheme.Colors.AgentColors == nil {
		tm.currentTheme.Colors.AgentColors = make(map[string]Color)
	}
	tm.currentTheme.Colors.AgentColors[agentID] = agentColor

	return agentColor
}

// generateAgentColor generates a unique color for an agent based on their ID
func (tm *ThemeManager) generateAgentColor(agentID string) Color {
	// Use a deterministic hash-based approach for consistent colors
	hash := tm.hashString(agentID)

	// Generate hue based on hash (0-360 degrees)
	hue := float64(hash % 360)

	// Use theme-appropriate saturation and lightness
	var saturation, lightness float64
	if tm.isDarkTheme() {
		saturation = 0.7
		lightness = 0.6
	} else {
		saturation = 0.6
		lightness = 0.5
	}

	baseColor := tm.hslToHex(hue, saturation, lightness)
	lightColor := tm.hslToHex(hue, saturation*0.8, lightness+0.2)
	darkColor := tm.hslToHex(hue, saturation*1.2, lightness-0.2)

	return Color{
		Base:    baseColor,
		Light:   lightColor,
		Dark:    darkColor,
		Muted:   tm.hslToHex(hue, saturation*0.5, lightness),
		Inverse: tm.getContrastColor(baseColor),
	}
}

// hashString creates a simple hash from a string
func (tm *ThemeManager) hashString(s string) uint32 {
	hash := uint32(2166136261)
	for _, b := range []byte(s) {
		hash ^= uint32(b)
		hash *= 16777619
	}
	return hash
}

// isDarkTheme determines if current theme is dark
func (tm *ThemeManager) isDarkTheme() bool {
	if tm.currentTheme == nil {
		return false
	}
	// Simple heuristic: check if background is darker than text
	return tm.currentTheme.Colors.Background.Base < tm.currentTheme.Colors.Text.Primary
}

// hslToHex converts HSL values to hex color
func (tm *ThemeManager) hslToHex(h, s, l float64) string {
	h = h / 360.0

	var r, g, b float64

	if s == 0 {
		r, g, b = l, l, l
	} else {
		hue2rgb := func(p, q, t float64) float64 {
			if t < 0 {
				t += 1
			}
			if t > 1 {
				t -= 1
			}
			if t < 1.0/6.0 {
				return p + (q-p)*6*t
			}
			if t < 1.0/2.0 {
				return q
			}
			if t < 2.0/3.0 {
				return p + (q-p)*(2.0/3.0-t)*6
			}
			return p
		}

		var q float64
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q

		r = hue2rgb(p, q, h+1.0/3.0)
		g = hue2rgb(p, q, h)
		b = hue2rgb(p, q, h-1.0/3.0)
	}

	return fmt.Sprintf("#%02x%02x%02x",
		int(r*255+0.5),
		int(g*255+0.5),
		int(b*255+0.5))
}

// getContrastColor returns black or white based on background
func (tm *ThemeManager) getContrastColor(bgColor string) string {
	if tm.isDarkTheme() {
		return "#ffffff"
	}
	return "#000000"
}

// AddObserver adds a theme change observer
func (tm *ThemeManager) AddObserver(observer ThemeObserver) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.observers = append(tm.observers, observer)
}

// LoadThemeFromFile loads a theme from a JSON file
func (tm *ThemeManager) LoadThemeFromFile(ctx context.Context, filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to read theme file").
			WithComponent("theme-manager").
			WithOperation("LoadThemeFromFile").
			WithDetails("filepath", filepath)
	}

	var theme Theme
	if err := json.Unmarshal(data, &theme); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to parse theme JSON").
			WithComponent("theme-manager").
			WithOperation("LoadThemeFromFile").
			WithDetails("filepath", filepath)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.themes[theme.Name] = &theme

	tm.logger.Info("Theme loaded from file",
		zap.String("theme", theme.Name),
		zap.String("filepath", filepath))

	return nil
}

// ExportTheme exports a theme to a JSON file
func (tm *ThemeManager) ExportTheme(ctx context.Context, themeName, outputPath string) error {
	tm.mu.RLock()
	theme, exists := tm.themes[themeName]
	tm.mu.RUnlock()

	if !exists {
		return gerror.New(ui.ErrCodeUIThemeNotFound, fmt.Sprintf("theme '%s' not found", themeName), nil).
			WithComponent("theme-manager").
			WithOperation("ExportTheme")
	}

	data, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to serialize theme").
			WithComponent("theme-manager").
			WithOperation("ExportTheme")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create output directory").
			WithComponent("theme-manager").
			WithOperation("ExportTheme")
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write theme file").
			WithComponent("theme-manager").
			WithOperation("ExportTheme")
	}

	tm.logger.Info("Theme exported successfully",
		zap.String("theme", themeName),
		zap.String("output_path", outputPath))

	return nil
}

// ListThemes returns all available theme names
func (tm *ThemeManager) ListThemes() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	themes := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		themes = append(themes, name)
	}
	return themes
}

// componentToStyle converts ComponentStyle to lipgloss.Style
func (tm *ThemeManager) componentToStyle(cs ComponentStyle) lipgloss.Style {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color(cs.Background)).
		Foreground(lipgloss.Color(cs.Foreground)).
		BorderForeground(lipgloss.Color(cs.Border))

	// Add padding if specified
	if cs.Padding != nil {
		if top, ok := cs.Padding["top"]; ok {
			style = style.PaddingTop(top)
		}
		if bottom, ok := cs.Padding["bottom"]; ok {
			style = style.PaddingBottom(bottom)
		}
		if left, ok := cs.Padding["left"]; ok {
			style = style.PaddingLeft(left)
		}
		if right, ok := cs.Padding["right"]; ok {
			style = style.PaddingRight(right)
		}
	}

	return style
}

// registerBuiltinThemes registers built-in Claude Code themes
func (tm *ThemeManager) registerBuiltinThemes() {
	// Claude Code Light Theme
	claudeLight := &Theme{
		Name:        "claude-code-light",
		DisplayName: "Claude Code Light",
		Version:     "1.0.0",
		Author:      "Guild Framework",
		Colors: ColorScheme{
			Primary:    Color{Base: "#2563eb", Light: "#60a5fa", Dark: "#1d4ed8", Muted: "#94a3b8", Inverse: "#ffffff"},
			Secondary:  Color{Base: "#64748b", Light: "#94a3b8", Dark: "#475569", Muted: "#cbd5e1", Inverse: "#ffffff"},
			Accent:     Color{Base: "#7c3aed", Light: "#a78bfa", Dark: "#5b21b6", Muted: "#c4b5fd", Inverse: "#ffffff"},
			Success:    Color{Base: "#059669", Light: "#34d399", Dark: "#047857", Muted: "#a7f3d0", Inverse: "#ffffff"},
			Warning:    Color{Base: "#d97706", Light: "#fbbf24", Dark: "#92400e", Muted: "#fde68a", Inverse: "#ffffff"},
			Error:      Color{Base: "#dc2626", Light: "#f87171", Dark: "#991b1b", Muted: "#fca5a5", Inverse: "#ffffff"},
			Info:       Color{Base: "#0891b2", Light: "#22d3ee", Dark: "#0e7490", Muted: "#a5f3fc", Inverse: "#ffffff"},
			Background: Color{Base: "#ffffff", Light: "#f8fafc", Dark: "#f1f5f9", Muted: "#e2e8f0", Inverse: "#1e293b"},
			Surface:    Color{Base: "#f8fafc", Light: "#ffffff", Dark: "#f1f5f9", Muted: "#e2e8f0", Inverse: "#1e293b"},
			Border:     Color{Base: "#e2e8f0", Light: "#f1f5f9", Dark: "#cbd5e1", Muted: "#94a3b8", Inverse: "#475569"},
			Text: Text{
				Primary:   "#1e293b",
				Secondary: "#475569",
				Muted:     "#64748b",
				Disabled:  "#94a3b8",
				Inverse:   "#ffffff",
				Link:      "#2563eb",
			},
			AgentColors: make(map[string]Color), // Will be populated dynamically
		},
		Typography: Typography{
			FontFamily:  "SF Mono, Monaco, Consolas, monospace",
			Scale:       TypographyScale{XS: 12, SM: 14, Base: 16, LG: 18, XL: 20, XXL: 24},
			Weights:     FontWeights{Light: 300, Normal: 400, Medium: 500, Bold: 600, Heavy: 700},
			LineHeights: LineHeights{Tight: 1.2, Normal: 1.5, Loose: 1.8},
		},
		Spacing: Spacing{XS: 2, SM: 4, MD: 8, LG: 16, XL: 24, XXL: 32},
		Animations: Animations{
			Enabled:     true,
			Duration:    AnimationDuration{Fast: 150, Normal: 300, Slow: 500},
			Easing:      AnimationEasing{Linear: "linear", EaseIn: "ease-in", EaseOut: "ease-out", EaseInOut: "ease-in-out"},
			Transitions: AnimationTypes{Fade: true, Slide: true, Scale: true, Rotation: false},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set component styles for Claude Code Light
	claudeLight.Components = Components{
		Button: ButtonStyles{
			Primary: ComponentStyle{
				Background:   "#2563eb",
				Foreground:   "#ffffff",
				Border:       "#1d4ed8",
				BorderRadius: 6,
				Padding:      map[string]int{"top": 2, "bottom": 2, "left": 4, "right": 4},
				Typography:   ComponentTypog{FontSize: 14, FontWeight: 500, LineHeight: 1.5},
				States:       map[string]string{"hover": "#1d4ed8", "active": "#1e40af"},
			},
		},
		Input: InputStyles{
			Default: ComponentStyle{
				Background:   "#ffffff",
				Foreground:   "#1e293b",
				Border:       "#e2e8f0",
				BorderRadius: 6,
				Padding:      map[string]int{"top": 2, "bottom": 2, "left": 3, "right": 3},
				Typography:   ComponentTypog{FontSize: 14, FontWeight: 400, LineHeight: 1.5},
			},
		},
		Chat: ChatStyles{
			Message: ComponentStyle{
				Background:   "#f8fafc",
				Foreground:   "#1e293b",
				Border:       "#e2e8f0",
				BorderRadius: 8,
				Padding:      map[string]int{"top": 3, "bottom": 3, "left": 4, "right": 4},
				Typography:   ComponentTypog{FontSize: 14, FontWeight: 400, LineHeight: 1.6},
			},
		},
	}

	tm.themes[claudeLight.Name] = claudeLight

	// Claude Code Dark Theme
	claudeDark := &Theme{
		Name:        "claude-code-dark",
		DisplayName: "Claude Code Dark",
		Version:     "1.0.0",
		Author:      "Guild Framework",
		Colors: ColorScheme{
			Primary:    Color{Base: "#3b82f6", Light: "#60a5fa", Dark: "#2563eb", Muted: "#64748b", Inverse: "#1e293b"},
			Secondary:  Color{Base: "#6b7280", Light: "#9ca3af", Dark: "#4b5563", Muted: "#374151", Inverse: "#f9fafb"},
			Accent:     Color{Base: "#8b5cf6", Light: "#a78bfa", Dark: "#7c3aed", Muted: "#6366f1", Inverse: "#1e1b4b"},
			Background: Color{Base: "#0f172a", Light: "#1e293b", Dark: "#020617", Muted: "#334155", Inverse: "#ffffff"},
			Surface:    Color{Base: "#1e293b", Light: "#334155", Dark: "#0f172a", Muted: "#475569", Inverse: "#f8fafc"},
			Text: Text{
				Primary:   "#f8fafc",
				Secondary: "#cbd5e1",
				Muted:     "#94a3b8",
				Disabled:  "#64748b",
				Inverse:   "#1e293b",
				Link:      "#60a5fa",
			},
			AgentColors: make(map[string]Color), // Will be populated dynamically
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tm.themes[claudeDark.Name] = claudeDark
}

// Helper function to get theme name safely
func getThemeName(theme *Theme) string {
	if theme == nil {
		return "none"
	}
	return theme.Name
}
