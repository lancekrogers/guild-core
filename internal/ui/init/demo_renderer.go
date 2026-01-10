// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package init

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour/v2"
	"charm.land/lipgloss/v2"

	"github.com/lancekrogers/guild-core/internal/setup"
)

// DemoRenderer handles rich rendering of demo commissions
type DemoRenderer struct {
	renderer *glamour.TermRenderer
	styles   *Styles
	width    int
}

// NewDemoRenderer creates a new demo renderer
func NewDemoRenderer(width int, styles *Styles) (*DemoRenderer, error) {
	// Use auto style for theme detection
	renderer, err := glamour.NewTermRenderer(
		glamour.WithEnvironmentConfig(),
		glamour.WithWordWrap(width-4),
	)
	if err != nil {
		return nil, err
	}

	return &DemoRenderer{
		renderer: renderer,
		styles:   styles,
		width:    width,
	}, nil
}

// RenderDemoSelection creates a rich demo selection view
func (dr *DemoRenderer) RenderDemoSelection(demos []DemoInfo, selected int) string {
	var sections []string

	// Header
	header := dr.styles.RenderHeader(
		"Choose Your Quest",
		"Select a demo commission to begin your journey",
	)
	sections = append(sections, header)

	// Demo list with preview
	for i, demo := range demos {
		isSelected := i == selected
		sections = append(sections, dr.renderDemoItem(demo, isSelected))
	}

	// Instructions
	instructions := dr.styles.Help.Render(
		"↑/↓ Navigate • Enter Select • Tab Preview • Esc Cancel",
	)
	sections = append(sections, instructions)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// RenderDemoPreview shows a rich preview of the selected demo
func (dr *DemoRenderer) RenderDemoPreview(demo DemoInfo) string {
	// Create markdown content for the preview
	preview := fmt.Sprintf(`# %s

%s

## 🎯 What You'll Build

%s

## 🛠️ Technologies

%s

## ⏱️ Estimated Time

%s to complete with Guild's assistance.
`, demo.Title, demo.Description, demo.Preview, demo.TechStack, demo.Duration)

	// Render with glamour
	rendered, _ := dr.renderer.Render(preview)

	// Add border and styling
	return dr.styles.Container.
		Width(dr.width - 4).
		Render(rendered)
}

// Helper to render individual demo items
func (dr *DemoRenderer) renderDemoItem(demo DemoInfo, selected bool) string {
	icon := dr.getDemoIcon(demo.Type)

	var style lipgloss.Style
	prefix := "  "
	if selected {
		style = dr.styles.DemoItemSelected
		prefix = "▶ "
	} else {
		style = dr.styles.DemoItem
	}

	// Title line
	title := style.Render(fmt.Sprintf("%s%s %s", prefix, icon, demo.Title))

	// Description line (only show if selected)
	if selected {
		desc := dr.styles.DemoDescription.Render(demo.ShortDesc)
		return lipgloss.JoinVertical(lipgloss.Left, title, desc)
	}

	return title
}

// getDemoIcon returns an appropriate icon for each demo type
func (dr *DemoRenderer) getDemoIcon(demoType setup.DemoCommissionType) string {
	icons := map[setup.DemoCommissionType]string{
		setup.DemoTypeAPIService:    "🔌",  // API
		setup.DemoTypeWebApp:        "🌐",  // Web
		setup.DemoTypeCLITool:       "⚡",  // CLI
		setup.DemoTypeDataAnalysis:  "📊",  // Data
		setup.DemoTypeMicroservices: "🏗️", // Microservices
		setup.DemoTypeAI:            "🤖",  // AI
	}

	if icon, ok := icons[demoType]; ok {
		return icon
	}
	return "📋" // Default
}

// RenderValidationResults creates a rich validation report
func (dr *DemoRenderer) RenderValidationResults(results []ValidationResult) string {
	var md strings.Builder

	// Count passes and failures
	passed := 0
	failed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	// Summary
	md.WriteString("# 🏰 Guild Initialization Report\n\n")

	if failed == 0 {
		md.WriteString("✅ **All checks passed!** Your guild is ready for action.\n\n")
	} else {
		md.WriteString(fmt.Sprintf("⚠️  **%d checks passed, %d need attention.**\n\n", passed, failed))
	}

	// Details
	md.WriteString("## Validation Results\n\n")

	for _, result := range results {
		icon := "✅"
		if !result.Passed {
			icon = "❌"
		}

		md.WriteString(fmt.Sprintf("%s **%s**\n", icon, result.Name))
		if result.Message != "" {
			md.WriteString(fmt.Sprintf("   %s\n", result.Message))
		}
		md.WriteString("\n")
	}

	// Render with glamour
	rendered, _ := dr.renderer.Render(md.String())

	return dr.styles.Section.Render(rendered)
}

// DemoInfo contains rich information about a demo
type DemoInfo struct {
	Type        setup.DemoCommissionType
	Title       string
	ShortDesc   string
	Description string
	Preview     string
	TechStack   string
	Duration    string
}

// GetDemoInfo returns rich information for all demos
func GetDemoInfo() []DemoInfo {
	return []DemoInfo{
		{
			Type:        setup.DemoTypeAPIService,
			Title:       "RESTful Task Management API",
			ShortDesc:   "Production-ready REST API with auth & testing",
			Description: "Build a comprehensive task management API that demonstrates modern backend practices with authentication, testing, and documentation.",
			Preview:     "A fully-featured REST API with JWT authentication, PostgreSQL persistence, comprehensive testing, and OpenAPI documentation.",
			TechStack:   "Go • PostgreSQL • JWT • OpenAPI • Docker",
			Duration:    "2-3 hours",
		},
		{
			Type:        setup.DemoTypeWebApp,
			Title:       "Real-Time Analytics Dashboard",
			ShortDesc:   "Modern web dashboard with live data visualization",
			Description: "Create an interactive analytics dashboard showcasing real-time data updates, responsive design, and beautiful visualizations.",
			Preview:     "A React-based dashboard with WebSocket connections, interactive charts, and a polished UI that updates in real-time.",
			TechStack:   "React • TypeScript • WebSockets • D3.js • Tailwind",
			Duration:    "3-4 hours",
		},
		{
			Type:        setup.DemoTypeCLITool,
			Title:       "Smart File Organizer CLI",
			ShortDesc:   "AI-powered CLI tool for intelligent file management",
			Description: "Develop a powerful CLI tool that uses AI to organize files, detect duplicates, and maintain a clean file system.",
			Preview:     "A feature-rich CLI with progress bars, concurrent processing, and smart categorization powered by local AI models.",
			TechStack:   "Go • Cobra • Bubble Tea • SQLite • AI/ML",
			Duration:    "2-3 hours",
		},
		{
			Type:        setup.DemoTypeDataAnalysis,
			Title:       "Sales Analytics Pipeline",
			ShortDesc:   "End-to-end data pipeline with ML insights",
			Description: "Build a complete data pipeline that ingests sales data, performs analysis, and generates predictive insights.",
			Preview:     "An automated pipeline with data validation, transformation, ML-based forecasting, and beautiful report generation.",
			TechStack:   "Python • Apache Beam • TensorFlow • PostgreSQL",
			Duration:    "3-4 hours",
		},
		{
			Type:        setup.DemoTypeMicroservices,
			Title:       "E-Commerce Platform",
			ShortDesc:   "Cloud-native microservices architecture",
			Description: "Design and implement a scalable e-commerce platform using microservices, event-driven architecture, and modern DevOps.",
			Preview:     "A distributed system with service mesh, event streaming, circuit breakers, and comprehensive observability.",
			TechStack:   "Go • Kubernetes • Kafka • gRPC • Istio",
			Duration:    "4-5 hours",
		},
		{
			Type:        setup.DemoTypeAI,
			Title:       "Content Recommendation Engine",
			ShortDesc:   "ML-powered personalization system",
			Description: "Create an intelligent recommendation system that learns from user behavior and delivers personalized content.",
			Preview:     "A sophisticated ML pipeline with real-time inference, A/B testing, and continuous learning capabilities.",
			TechStack:   "Python • PyTorch • Redis • FastAPI • Docker",
			Duration:    "4-5 hours",
		},
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func uintPtr(u uint) *uint {
	return &u
}
