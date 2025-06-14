package visual

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// CodeRenderer handles enhanced code display with syntax highlighting and features
type CodeRenderer struct {
	showLineNumbers bool
	enableDiffMode  bool
	enableFolding   bool
	tabSize         int
	maxWidth        int
	
	// Styling
	lineNumberStyle  lipgloss.Style
	codeStyle        lipgloss.Style
	addedLineStyle   lipgloss.Style
	removedLineStyle lipgloss.Style
	contextLineStyle lipgloss.Style
	foldedStyle      lipgloss.Style
	
	// Language-specific highlighting
	keywordColors    map[string]lipgloss.Color
	stringColors     map[string]lipgloss.Color
	commentColors    map[string]lipgloss.Color
}

// CodeBlock represents a processed code block
type CodeBlock struct {
	Language     string
	Content      string
	StartLine    int
	EndLine      int
	IsDiff       bool
	HasFolded    bool
	LineNumbers  bool
	ProcessedContent string
}

// DiffLine represents a line in a diff
type DiffLine struct {
	Number   int
	Content  string
	Type     DiffLineType
	Original string
}

// DiffLineType represents the type of diff line
type DiffLineType int

const (
	DiffLineContext DiffLineType = iota
	DiffLineAdded
	DiffLineRemoved
	DiffLineModified
)

// NewCodeRenderer creates a new enhanced code renderer
func NewCodeRenderer() *CodeRenderer {
	return &CodeRenderer{
		showLineNumbers: false, // Default off, can be toggled
		enableDiffMode:  true,
		enableFolding:   true,
		tabSize:         4,
		maxWidth:        120,
		
		lineNumberStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(4).
			Align(lipgloss.Right),
		
		codeStyle: lipgloss.NewStyle().
			Padding(0, 1),
		
		addedLineStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("22")).  // Dark green
			Foreground(lipgloss.Color("120")), // Bright green
		
		removedLineStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("52")).  // Dark red
			Foreground(lipgloss.Color("196")), // Bright red
		
		contextLineStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")), // Gray
		
		foldedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true),
		
		keywordColors: map[string]lipgloss.Color{
			"go":         lipgloss.Color("33"),  // Yellow
			"python":     lipgloss.Color("34"),  // Green
			"javascript": lipgloss.Color("226"), // Bright yellow
			"java":       lipgloss.Color("208"), // Orange
			"rust":       lipgloss.Color("130"), // Dark orange
			"cpp":        lipgloss.Color("27"),  // Blue
		},
		
		stringColors: map[string]lipgloss.Color{
			"default": lipgloss.Color("82"), // Bright green
		},
		
		commentColors: map[string]lipgloss.Color{
			"default": lipgloss.Color("240"), // Dark gray
		},
	}
}

// ProcessCodeBlocks detects and enhances code blocks in content
func (cr *CodeRenderer) ProcessCodeBlocks(content string) string {
	// Find all code blocks
	codeBlocks := cr.detectCodeBlocks(content)
	
	// Process from end to start to maintain indices
	processedContent := content
	for i := len(codeBlocks) - 1; i >= 0; i-- {
		block := codeBlocks[i]
		enhanced := cr.enhanceCodeBlock(block)
		
		// Replace in content
		start := strings.Index(processedContent, block.Content)
		if start >= 0 {
			end := start + len(block.Content)
			processedContent = processedContent[:start] + enhanced + processedContent[end:]
		}
	}
	
	return processedContent
}

// detectCodeBlocks finds code blocks in content
func (cr *CodeRenderer) detectCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock
	
	// Pattern for fenced code blocks
	fencedRegex := regexp.MustCompile("(?s)```(\\w*)?\\n?(.*?)```")
	matches := fencedRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			language := match[1]
			code := match[2]
			
			block := CodeBlock{
				Language:    language,
				Content:     match[0], // Full match including fences
				IsDiff:      cr.isDiffContent(code),
				LineNumbers: cr.showLineNumbers,
			}
			
			// Detect language if not specified
			if language == "" {
				block.Language = cr.detectLanguage(code)
			}
			
			blocks = append(blocks, block)
		}
	}
	
	// Pattern for indented code blocks
	indentedRegex := regexp.MustCompile("(?m)^(    .+)$")
	indentedMatches := indentedRegex.FindAllString(content, -1)
	
	for _, match := range indentedMatches {
		// Skip if already part of a fenced block
		isInFenced := false
		for _, block := range blocks {
			if strings.Contains(block.Content, match) {
				isInFenced = true
				break
			}
		}
		
		if !isInFenced {
			code := strings.TrimPrefix(match, "    ")
			block := CodeBlock{
				Language:    cr.detectLanguage(code),
				Content:     match,
				IsDiff:      cr.isDiffContent(code),
				LineNumbers: cr.showLineNumbers,
			}
			blocks = append(blocks, block)
		}
	}
	
	return blocks
}

// enhanceCodeBlock applies syntax highlighting and features to a code block
func (cr *CodeRenderer) enhanceCodeBlock(block CodeBlock) string {
	// Extract actual code content
	code := cr.extractCodeContent(block.Content)
	
	if block.IsDiff && cr.enableDiffMode {
		return cr.renderDiff(code, block.Language)
	}
	
	return cr.renderCode(code, block.Language, block.LineNumbers)
}

// extractCodeContent extracts the actual code from the full block content
func (cr *CodeRenderer) extractCodeContent(blockContent string) string {
	// Remove fenced code block markers
	fencedRegex := regexp.MustCompile("(?s)```\\w*\\n?(.*?)```")
	if matches := fencedRegex.FindStringSubmatch(blockContent); len(matches) >= 2 {
		return matches[1]
	}
	
	// Remove indentation for indented blocks
	lines := strings.Split(blockContent, "\n")
	var codeLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "    ") {
			codeLines = append(codeLines, strings.TrimPrefix(line, "    "))
		}
	}
	
	return strings.Join(codeLines, "\n")
}

// renderCode renders code with syntax highlighting and line numbers
func (cr *CodeRenderer) renderCode(code, language string, showLineNumbers bool) string {
	lines := strings.Split(code, "\n")
	
	// Apply code folding if enabled and content is long
	if cr.enableFolding && len(lines) > 30 {
		lines = cr.applyCodeFolding(lines)
	}
	
	var renderedLines []string
	
	for i, line := range lines {
		lineNum := i + 1
		
		// Apply syntax highlighting
		highlightedLine := cr.applySyntaxHighlighting(line, language)
		
		// Add line numbers if enabled
		if showLineNumbers {
			lineNumStr := cr.lineNumberStyle.Render(fmt.Sprintf("%4d", lineNum))
			highlightedLine = fmt.Sprintf("%s │ %s", lineNumStr, highlightedLine)
		}
		
		renderedLines = append(renderedLines, highlightedLine)
	}
	
	// Wrap in code block styling
	codeContent := strings.Join(renderedLines, "\n")
	
	// Add language label
	if language != "" {
		header := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render(fmt.Sprintf("// %s", language))
		codeContent = fmt.Sprintf("%s\n%s", header, codeContent)
	}
	
	return cr.codeStyle.Render(codeContent)
}

// renderDiff renders diff content with appropriate highlighting
func (cr *CodeRenderer) renderDiff(code, language string) string {
	diffLines := cr.parseDiffLines(code)
	var renderedLines []string
	
	for _, diffLine := range diffLines {
		var style lipgloss.Style
		var prefix string
		
		switch diffLine.Type {
		case DiffLineAdded:
			style = cr.addedLineStyle
			prefix = "+ "
		case DiffLineRemoved:
			style = cr.removedLineStyle
			prefix = "- "
		case DiffLineModified:
			style = cr.addedLineStyle // Treat as added for now
			prefix = "~ "
		default:
			style = cr.contextLineStyle
			prefix = "  "
		}
		
		// Apply syntax highlighting to the content
		highlightedContent := cr.applySyntaxHighlighting(diffLine.Content, language)
		
		// Render with diff styling
		line := style.Render(fmt.Sprintf("%s%s", prefix, highlightedContent))
		renderedLines = append(renderedLines, line)
	}
	
	// Add diff header
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Bold(true).
		Render("📝 Code Changes:")
	
	return fmt.Sprintf("%s\n%s", header, strings.Join(renderedLines, "\n"))
}

// parseDiffLines parses diff content into structured diff lines
func (cr *CodeRenderer) parseDiffLines(code string) []DiffLine {
	lines := strings.Split(code, "\n")
	var diffLines []DiffLine
	
	for i, line := range lines {
		diffLine := DiffLine{
			Number:   i + 1,
			Content:  line,
			Original: line,
			Type:     DiffLineContext,
		}
		
		// Detect diff line types
		if strings.HasPrefix(line, "+") {
			diffLine.Type = DiffLineAdded
			diffLine.Content = strings.TrimPrefix(line, "+")
		} else if strings.HasPrefix(line, "-") {
			diffLine.Type = DiffLineRemoved
			diffLine.Content = strings.TrimPrefix(line, "-")
		} else if strings.HasPrefix(line, "~") || strings.HasPrefix(line, "!") {
			diffLine.Type = DiffLineModified
			diffLine.Content = strings.TrimPrefix(strings.TrimPrefix(line, "~"), "!")
		}
		
		diffLines = append(diffLines, diffLine)
	}
	
	return diffLines
}

// applySyntaxHighlighting applies basic syntax highlighting to a line
func (cr *CodeRenderer) applySyntaxHighlighting(line, language string) string {
	// Basic syntax highlighting patterns
	
	// Highlight strings
	stringRegex := regexp.MustCompile(`"([^"\\]|\\.)*"|'([^'\\]|\\.)*'|` + "`" + `([^` + "`" + `\\]|\\.)*` + "`")
	line = stringRegex.ReplaceAllStringFunc(line, func(match string) string {
		return lipgloss.NewStyle().Foreground(cr.stringColors["default"]).Render(match)
	})
	
	// Highlight comments
	commentPatterns := map[string]*regexp.Regexp{
		"go":         regexp.MustCompile(`//.*$|/\*.*?\*/`),
		"python":     regexp.MustCompile(`#.*$`),
		"javascript": regexp.MustCompile(`//.*$|/\*.*?\*/`),
		"java":       regexp.MustCompile(`//.*$|/\*.*?\*/`),
		"rust":       regexp.MustCompile(`//.*$|/\*.*?\*/`),
		"cpp":        regexp.MustCompile(`//.*$|/\*.*?\*/`),
	}
	
	if commentRegex, exists := commentPatterns[language]; exists {
		line = commentRegex.ReplaceAllStringFunc(line, func(match string) string {
			return lipgloss.NewStyle().Foreground(cr.commentColors["default"]).Render(match)
		})
	}
	
	// Highlight keywords (basic patterns)
	keywordPatterns := map[string][]string{
		"go":         {"func", "var", "const", "type", "package", "import", "if", "else", "for", "range", "return"},
		"python":     {"def", "class", "import", "from", "if", "else", "elif", "for", "while", "return", "try", "except"},
		"javascript": {"function", "var", "let", "const", "if", "else", "for", "while", "return", "class", "import", "export"},
		"java":       {"public", "private", "protected", "class", "interface", "if", "else", "for", "while", "return", "try", "catch"},
		"rust":       {"fn", "let", "mut", "struct", "enum", "impl", "if", "else", "for", "while", "return", "match"},
	}
	
	if keywords, exists := keywordPatterns[language]; exists {
		keywordColor := cr.keywordColors[language]
		if keywordColor == "" {
			keywordColor = cr.keywordColors["go"] // Default
		}
		
		for _, keyword := range keywords {
			pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
			keywordRegex := regexp.MustCompile(pattern)
			line = keywordRegex.ReplaceAllStringFunc(line, func(match string) string {
				return lipgloss.NewStyle().Foreground(keywordColor).Bold(true).Render(match)
			})
		}
	}
	
	return line
}

// applyCodeFolding applies code folding to long content
func (cr *CodeRenderer) applyCodeFolding(lines []string) []string {
	if len(lines) <= 30 {
		return lines
	}
	
	// Show first 10 lines, folded indicator, last 10 lines
	var result []string
	
	// First 10 lines
	result = append(result, lines[:10]...)
	
	// Folded indicator
	foldedCount := len(lines) - 20
	foldedLine := cr.foldedStyle.Render(fmt.Sprintf("... %d lines folded ...", foldedCount))
	result = append(result, foldedLine)
	
	// Last 10 lines
	result = append(result, lines[len(lines)-10:]...)
	
	return result
}

// isDiffContent detects if content is a diff
func (cr *CodeRenderer) isDiffContent(code string) bool {
	lines := strings.Split(code, "\n")
	diffLineCount := 0
	
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "@@") {
			diffLineCount++
		}
	}
	
	// Consider it a diff if more than 20% of lines are diff markers
	return float64(diffLineCount)/float64(len(lines)) > 0.2
}

// detectLanguage detects programming language from code content
func (cr *CodeRenderer) detectLanguage(code string) string {
	code = strings.TrimSpace(strings.ToLower(code))
	
	// Language detection heuristics
	patterns := map[string][]string{
		"go": {
			"package main", "func ", "import (", ":=", "make(", "append(",
		},
		"python": {
			"def ", "import ", "from ", "__init__", "self.", "print(",
		},
		"javascript": {
			"function ", "const ", "let ", "var ", "=&gt;", "console.log",
		},
		"java": {
			"public class", "public static", "private ", "System.out",
		},
		"rust": {
			"fn ", "let mut", "impl ", "match ", "Result&lt;", "Option&lt;",
		},
		"cpp": {
			"#include", "std::", "int main", "using namespace",
		},
		"sql": {
			"select ", "from ", "where ", "insert ", "update ", "delete ",
		},
		"json": {
			"{", "}", "[", "]", "\":", "\",",
		},
		"yaml": {
			"---", "- ", ": |", ": &gt;",
		},
		"xml": {
			"&lt;?xml", "&lt;/", "&gt;",
		},
		"bash": {
			"#!/bin/bash", "echo ", "if [", "then", "fi",
		},
	}
	
	for lang, keywords := range patterns {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(code, keyword) {
				score++
			}
		}
		
		// If we find enough matches, return this language
		if score >= 2 {
			return lang
		}
	}
	
	return "text"
}

// ToggleLineNumbers toggles line number display
func (cr *CodeRenderer) ToggleLineNumbers() {
	cr.showLineNumbers = !cr.showLineNumbers
}

// SetMaxWidth sets the maximum width for code rendering
func (cr *CodeRenderer) SetMaxWidth(width int) {
	cr.maxWidth = width
}

// SetTabSize sets the tab size for code formatting
func (cr *CodeRenderer) SetTabSize(size int) {
	cr.tabSize = size
}

// GetLanguageStats returns statistics about detected languages in processed content
func (cr *CodeRenderer) GetLanguageStats(content string) map[string]int {
	blocks := cr.detectCodeBlocks(content)
	stats := make(map[string]int)
	
	for _, block := range blocks {
		stats[block.Language]++
	}
	
	return stats
}