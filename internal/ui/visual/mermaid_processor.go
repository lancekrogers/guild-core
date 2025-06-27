// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package visual

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// MermaidProcessor handles Mermaid diagram detection and rendering
type MermaidProcessor struct {
	enableCLI     bool
	outputDir     string
	enablePreview bool
	maxWidth      int
	maxHeight     int

	// ASCII art generation settings
	asciiWidth  int
	asciiHeight int

	// Styling
	diagramStyle lipgloss.Style
	titleStyle   lipgloss.Style
	errorStyle   lipgloss.Style
}

// MermaidDiagram represents a detected Mermaid diagram
type MermaidDiagram struct {
	Type          string
	Title         string
	Content       string
	StartIndex    int
	EndIndex      int
	IsValid       bool
	Error         string
	GeneratedPath string
	ASCIIPreview  string
}

// NewMermaidProcessor creates a new Mermaid diagram processor
func NewMermaidProcessor() *MermaidProcessor {
	return &MermaidProcessor{
		enableCLI:     true,
		outputDir:     os.TempDir(),
		enablePreview: true,
		maxWidth:      800,
		maxHeight:     600,
		asciiWidth:    80,
		asciiHeight:   40,

		diagramStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")). // Purple
			Padding(1).
			Margin(1, 0),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")). // Purple
			Bold(true),

		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")). // Red
			Bold(true),
	}
}

// ProcessContent detects and processes Mermaid diagrams in content
func (mp *MermaidProcessor) ProcessContent(content string) (string, []MermaidDiagram, error) {
	// Detect Mermaid diagrams
	diagrams := mp.detectMermaidDiagrams(content)

	// Process each diagram
	processedDiagrams := make([]MermaidDiagram, 0, len(diagrams))
	processedContent := content

	// Process from end to start to maintain string indices
	for i := len(diagrams) - 1; i >= 0; i-- {
		diagram := diagrams[i]

		// Process the diagram
		processedDiagram := mp.processMermaidDiagram(diagram)
		processedDiagrams = append([]MermaidDiagram{processedDiagram}, processedDiagrams...)

		// Replace in content
		if processedDiagram.IsValid {
			replacement := mp.generateDiagramReplacement(processedDiagram)
			processedContent = processedContent[:diagram.StartIndex] + replacement + processedContent[diagram.EndIndex:]
		}
	}

	return processedContent, processedDiagrams, nil
}

// detectMermaidDiagrams finds Mermaid diagrams in content
func (mp *MermaidProcessor) detectMermaidDiagrams(content string) []MermaidDiagram {
	var diagrams []MermaidDiagram

	// Pattern for fenced Mermaid blocks
	mermaidRegex := regexp.MustCompile("(?s)```mermaid\\s*\\n?(.*?)```")
	matches := mermaidRegex.FindAllStringSubmatch(content, -1)
	indices := mermaidRegex.FindAllStringIndex(content, -1)

	for i, match := range matches {
		if len(match) >= 2 {
			diagramContent := strings.TrimSpace(match[1])
			diagramType := mp.detectDiagramType(diagramContent)
			title := mp.extractDiagramTitle(diagramContent)

			diagrams = append(diagrams, MermaidDiagram{
				Type:       diagramType,
				Title:      title,
				Content:    diagramContent,
				StartIndex: indices[i][0],
				EndIndex:   indices[i][1],
			})
		}
	}

	return diagrams
}

// processMermaidDiagram processes a single Mermaid diagram
func (mp *MermaidProcessor) processMermaidDiagram(diagram MermaidDiagram) MermaidDiagram {
	// Validate Mermaid syntax
	if !mp.validateMermaidSyntax(diagram.Content) {
		diagram.IsValid = false
		diagram.Error = "Invalid Mermaid syntax"
		return diagram
	}

	diagram.IsValid = true

	// Generate image if CLI is available and enabled
	if mp.enableCLI {
		imagePath, err := mp.generateMermaidImage(diagram)
		if err == nil {
			diagram.GeneratedPath = imagePath

			// Generate ASCII preview from image
			if mp.enablePreview {
				asciiPreview, err := mp.generateASCIIFromImage(imagePath)
				if err == nil {
					diagram.ASCIIPreview = asciiPreview
				}
			}
		} else {
			// Fall back to ASCII representation
			diagram.ASCIIPreview = mp.generateASCIIRepresentation(diagram)
		}
	} else {
		// Generate ASCII representation directly
		diagram.ASCIIPreview = mp.generateASCIIRepresentation(diagram)
	}

	return diagram
}

// generateDiagramReplacement creates a replacement string for a Mermaid diagram
func (mp *MermaidProcessor) generateDiagramReplacement(diagram MermaidDiagram) string {
	var parts []string

	// Title
	title := diagram.Title
	if title == "" {
		title = fmt.Sprintf("%s Diagram", strings.Title(diagram.Type))
	}
	parts = append(parts, mp.titleStyle.Render(fmt.Sprintf("📊 %s", title)))

	// ASCII preview or representation
	if diagram.ASCIIPreview != "" {
		parts = append(parts, "```")
		parts = append(parts, diagram.ASCIIPreview)
		parts = append(parts, "```")
	}

	// View instructions
	if diagram.GeneratedPath != "" {
		parts = append(parts, fmt.Sprintf("*Full diagram: `open %s`*", diagram.GeneratedPath))
	}

	// Live edit instruction
	parts = append(parts, "*Edit: Use `/mermaid` command for live preview*")

	return mp.diagramStyle.Render(strings.Join(parts, "\n"))
}

// validateMermaidSyntax performs basic Mermaid syntax validation
func (mp *MermaidProcessor) validateMermaidSyntax(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}

	// Check for basic Mermaid diagram types
	validStarts := []string{
		"graph", "flowchart", "sequenceDiagram", "classDiagram",
		"stateDiagram", "erDiagram", "pie", "gantt", "gitgraph",
		"journey", "requirement", "mindmap", "timeline",
	}

	for _, start := range validStarts {
		if strings.HasPrefix(content, start) {
			return true
		}
	}

	return false
}

// detectDiagramType detects the type of Mermaid diagram
func (mp *MermaidProcessor) detectDiagramType(content string) string {
	content = strings.TrimSpace(strings.ToLower(content))

	if strings.HasPrefix(content, "graph") {
		return "graph"
	}
	if strings.HasPrefix(content, "flowchart") {
		return "flowchart"
	}
	if strings.HasPrefix(content, "sequencediagram") {
		return "sequence"
	}
	if strings.HasPrefix(content, "classdiagram") {
		return "class"
	}
	if strings.HasPrefix(content, "statediagram") {
		return "state"
	}
	if strings.HasPrefix(content, "erdiagram") {
		return "er"
	}
	if strings.HasPrefix(content, "pie") {
		return "pie"
	}
	if strings.HasPrefix(content, "gantt") {
		return "gantt"
	}
	if strings.HasPrefix(content, "gitgraph") {
		return "gitgraph"
	}
	if strings.HasPrefix(content, "journey") {
		return "journey"
	}
	if strings.HasPrefix(content, "mindmap") {
		return "mindmap"
	}
	if strings.HasPrefix(content, "timeline") {
		return "timeline"
	}

	return "unknown"
}

// extractDiagramTitle extracts title from Mermaid diagram
func (mp *MermaidProcessor) extractDiagramTitle(content string) string {
	// Look for title directive
	titleRegex := regexp.MustCompile(`(?m)^\s*title\s+(.+)$`)
	if matches := titleRegex.FindStringSubmatch(content); len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	// Look for graph title in graph definitions
	graphTitleRegex := regexp.MustCompile(`(?m)^(?:graph|flowchart)\s+\w+\s*\[\s*"([^"]+)"\s*\]`)
	if matches := graphTitleRegex.FindStringSubmatch(content); len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// generateMermaidImage generates an image from Mermaid content using mermaid-cli
func (mp *MermaidProcessor) generateMermaidImage(diagram MermaidDiagram) (string, error) {
	// Check if mermaid CLI is available
	if _, err := exec.LookPath("mmdc"); err != nil {
		return "", gerror.New(gerror.ErrCodeExternal, "mermaid-cli (mmdc) not found", err)
	}

	// Create temporary files
	timestamp := time.Now().Format("20060102-150405")
	tempFile := filepath.Join(mp.outputDir, fmt.Sprintf("mermaid-%s.mmd", timestamp))
	outputFile := filepath.Join(mp.outputDir, fmt.Sprintf("mermaid-%s.png", timestamp))

	// Write Mermaid content to file
	if err := os.WriteFile(tempFile, []byte(diagram.Content), 0644); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write Mermaid file")
	}
	defer os.Remove(tempFile)

	// Generate image using mermaid-cli
	cmd := exec.Command("mmdc", "-i", tempFile, "-o", outputFile, "--width", fmt.Sprintf("%d", mp.maxWidth), "--height", fmt.Sprintf("%d", mp.maxHeight))

	if output, err := cmd.CombinedOutput(); err != nil {
		return "", gerror.New(gerror.ErrCodeExternal, fmt.Sprintf("mermaid-cli failed: %s", string(output)), err)
	}

	return outputFile, nil
}

// generateASCIIFromImage converts a generated image to ASCII art
func (mp *MermaidProcessor) generateASCIIFromImage(imagePath string) (string, error) {
	// Use image processor to generate ASCII art
	imageProcessor := NewImageProcessor()
	imageProcessor.SetASCIISize(mp.asciiWidth, mp.asciiHeight)

	return imageProcessor.generateASCIIArt(imagePath)
}

// generateASCIIRepresentation creates a simple ASCII representation of the diagram
func (mp *MermaidProcessor) generateASCIIRepresentation(diagram MermaidDiagram) string {
	switch diagram.Type {
	case "graph", "flowchart":
		return mp.generateFlowchartASCII(diagram.Content)
	case "sequence":
		return mp.generateSequenceASCII(diagram.Content)
	case "pie":
		return mp.generatePieASCII(diagram.Content)
	case "gantt":
		return mp.generateGanttASCII(diagram.Content)
	default:
		return mp.generateGenericASCII(diagram.Content, diagram.Type)
	}
}

// generateFlowchartASCII creates a simple ASCII flowchart
func (mp *MermaidProcessor) generateFlowchartASCII(content string) string {
	lines := strings.Split(content, "\n")
	var ascii []string

	ascii = append(ascii, "┌─────────────────────────────────────┐")
	ascii = append(ascii, "│           FLOWCHART DIAGRAM         │")
	ascii = append(ascii, "├─────────────────────────────────────┤")

	// Extract nodes and connections
	nodeRegex := regexp.MustCompile(`(\w+)\s*\[\s*"([^"]+)"\s*\]`)
	connectionRegex := regexp.MustCompile(`(\w+)\s*-->\s*(\w+)`)

	nodes := make(map[string]string)
	connections := make([]string, 0)

	for _, line := range lines {
		if matches := nodeRegex.FindStringSubmatch(line); len(matches) >= 3 {
			nodes[matches[1]] = matches[2]
		}
		if matches := connectionRegex.FindStringSubmatch(line); len(matches) >= 3 {
			connections = append(connections, fmt.Sprintf("%s → %s", matches[1], matches[2]))
		}
	}

	// Display nodes
	for id, label := range nodes {
		ascii = append(ascii, fmt.Sprintf("│ [%s] %s", id, label))
	}

	ascii = append(ascii, "│")

	// Display connections
	for _, conn := range connections {
		ascii = append(ascii, fmt.Sprintf("│ %s", conn))
	}

	ascii = append(ascii, "└─────────────────────────────────────┘")

	return strings.Join(ascii, "\n")
}

// generateSequenceASCII creates a simple ASCII sequence diagram
func (mp *MermaidProcessor) generateSequenceASCII(content string) string {
	lines := strings.Split(content, "\n")
	var ascii []string

	ascii = append(ascii, "┌─────────────────────────────────────┐")
	ascii = append(ascii, "│         SEQUENCE DIAGRAM            │")
	ascii = append(ascii, "├─────────────────────────────────────┤")

	participantRegex := regexp.MustCompile(`participant\s+(\w+)(?:\s+as\s+(.+))?`)
	messageRegex := regexp.MustCompile(`(\w+)\s*->>?\s*(\w+)\s*:\s*(.+)`)

	participants := make(map[string]string)
	messages := make([]string, 0)

	for _, line := range lines {
		if matches := participantRegex.FindStringSubmatch(line); len(matches) >= 2 {
			name := matches[1]
			label := name
			if len(matches) >= 3 && matches[2] != "" {
				label = matches[2]
			}
			participants[name] = label
		}
		if matches := messageRegex.FindStringSubmatch(line); len(matches) >= 4 {
			messages = append(messages, fmt.Sprintf("%s → %s: %s", matches[1], matches[2], matches[3]))
		}
	}

	// Display participants
	for _, label := range participants {
		ascii = append(ascii, fmt.Sprintf("│ │ %s │", label))
	}

	ascii = append(ascii, "│")

	// Display messages
	for _, msg := range messages {
		ascii = append(ascii, fmt.Sprintf("│ %s", msg))
	}

	ascii = append(ascii, "└─────────────────────────────────────┘")

	return strings.Join(ascii, "\n")
}

// generatePieASCII creates a simple ASCII pie chart representation
func (mp *MermaidProcessor) generatePieASCII(content string) string {
	var ascii []string

	ascii = append(ascii, "┌─────────────────────────────────────┐")
	ascii = append(ascii, "│            PIE CHART                │")
	ascii = append(ascii, "├─────────────────────────────────────┤")
	ascii = append(ascii, "│                                     │")
	ascii = append(ascii, "│         ████████████                │")
	ascii = append(ascii, "│      ████           ████             │")
	ascii = append(ascii, "│    ████     ●───────  ████           │")
	ascii = append(ascii, "│   ████               ████            │")
	ascii = append(ascii, "│    ████             ████             │")
	ascii = append(ascii, "│      ████         ████               │")
	ascii = append(ascii, "│         ████████████                 │")
	ascii = append(ascii, "│                                     │")

	// Extract data points
	dataRegex := regexp.MustCompile(`"([^"]+)"\s*:\s*(\d+(?:\.\d+)?)`)
	matches := dataRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			ascii = append(ascii, fmt.Sprintf("│ ● %s: %s", match[1], match[2]))
		}
	}

	ascii = append(ascii, "└─────────────────────────────────────┘")

	return strings.Join(ascii, "\n")
}

// generateGanttASCII creates a simple ASCII Gantt chart
func (mp *MermaidProcessor) generateGanttASCII(content string) string {
	var ascii []string

	ascii = append(ascii, "┌─────────────────────────────────────┐")
	ascii = append(ascii, "│           GANTT CHART               │")
	ascii = append(ascii, "├─────────────────────────────────────┤")
	ascii = append(ascii, "│ Task 1    ██████████                │")
	ascii = append(ascii, "│ Task 2        ████████              │")
	ascii = append(ascii, "│ Task 3             ██████████       │")
	ascii = append(ascii, "│ Task 4                    ████████  │")
	ascii = append(ascii, "├─────────────────────────────────────┤")
	ascii = append(ascii, "│ Week 1 | Week 2 | Week 3 | Week 4   │")
	ascii = append(ascii, "└─────────────────────────────────────┘")

	return strings.Join(ascii, "\n")
}

// generateGenericASCII creates a generic ASCII representation
func (mp *MermaidProcessor) generateGenericASCII(content, diagramType string) string {
	var ascii []string

	title := fmt.Sprintf("%s DIAGRAM", strings.ToUpper(diagramType))

	ascii = append(ascii, "┌─────────────────────────────────────┐")
	ascii = append(ascii, fmt.Sprintf("│ %-35s │", title))
	ascii = append(ascii, "├─────────────────────────────────────┤")
	ascii = append(ascii, "│                                     │")
	ascii = append(ascii, "│   Use mermaid-cli for full render   │")
	ascii = append(ascii, "│                                     │")
	ascii = append(ascii, "│   npm install -g @mermaid-js/       │")
	ascii = append(ascii, "│   mermaid-cli                       │")
	ascii = append(ascii, "│                                     │")
	ascii = append(ascii, "└─────────────────────────────────────┘")

	return strings.Join(ascii, "\n")
}

// SetOutputDirectory sets the directory for generated diagrams
func (mp *MermaidProcessor) SetOutputDirectory(dir string) {
	mp.outputDir = dir
}

// SetImageSize sets the size for generated images
func (mp *MermaidProcessor) SetImageSize(width, height int) {
	mp.maxWidth = width
	mp.maxHeight = height
}

// SetASCIISize sets the size for ASCII previews
func (mp *MermaidProcessor) SetASCIISize(width, height int) {
	mp.asciiWidth = width
	mp.asciiHeight = height
}

// ToggleCLI enables or disables Mermaid CLI usage
func (mp *MermaidProcessor) ToggleCLI(enabled bool) {
	mp.enableCLI = enabled
}

// TogglePreview enables or disables ASCII previews
func (mp *MermaidProcessor) TogglePreview(enabled bool) {
	mp.enablePreview = enabled
}
