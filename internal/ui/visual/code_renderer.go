// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package visual

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
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
	keywordColors map[string]color.Color
	stringColors  map[string]color.Color
	commentColors map[string]color.Color
}

// CodeBlock represents a processed code block
type CodeBlock struct {
	Language         string
	Content          string
	StartLine        int
	EndLine          int
	IsDiff           bool
	HasFolded        bool
	LineNumbers      bool
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

		keywordColors: map[string]color.Color{
			"go":         lipgloss.Color("33"),  // Yellow
			"python":     lipgloss.Color("34"),  // Green
			"javascript": lipgloss.Color("226"), // Bright yellow
			"typescript": lipgloss.Color("39"),  // Blue
			"java":       lipgloss.Color("208"), // Orange
			"rust":       lipgloss.Color("130"), // Dark orange
			"cpp":        lipgloss.Color("27"),  // Blue
			"c":          lipgloss.Color("27"),  // Blue
			"ruby":       lipgloss.Color("196"), // Red
			"php":        lipgloss.Color("99"),  // Purple
			"swift":      lipgloss.Color("214"), // Orange
			"kotlin":     lipgloss.Color("99"),  // Purple
			"sql":        lipgloss.Color("75"),  // Light blue
			"shell":      lipgloss.Color("154"), // Light green
			"yaml":       lipgloss.Color("226"), // Yellow
			"json":       lipgloss.Color("39"),  // Blue
		},

		stringColors: map[string]color.Color{
			"default": lipgloss.Color("82"),  // Bright green
			"sql":     lipgloss.Color("196"), // Red for SQL strings
			"regex":   lipgloss.Color("214"), // Orange for regex
		},

		commentColors: map[string]color.Color{
			"default": lipgloss.Color("240"), // Dark gray
			"doc":     lipgloss.Color("244"), // Lighter gray for doc comments
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

// applySyntaxHighlighting applies sophisticated syntax highlighting to a line
func (cr *CodeRenderer) applySyntaxHighlighting(line, language string) string {
	// Track highlighted segments to avoid overlapping
	type segment struct {
		start, end int
		style      lipgloss.Style
		text       string
	}
	var segments []segment

	// Helper to add segment without overlap
	addSegment := func(start, end int, style lipgloss.Style, text string) {
		for _, s := range segments {
			if (start >= s.start && start < s.end) || (end > s.start && end <= s.end) {
				return // Skip overlapping segments
			}
		}
		segments = append(segments, segment{start, end, style, text})
	}

	// 1. Numbers (integers, floats, hex, binary)
	numberRegex := regexp.MustCompile(`\b(0x[a-fA-F0-9]+|0b[01]+|\d+\.?\d*[eE]?[+-]?\d*)\b`)
	for _, match := range numberRegex.FindAllStringIndex(line, -1) {
		numberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141")) // Purple
		addSegment(match[0], match[1], numberStyle, line[match[0]:match[1]])
	}

	// 2. Strings with proper escape handling
	stringPatterns := []string{
		`"(?:[^"\\]|\\.)*"`,     // Double quotes
		`'(?:[^'\\]|\\.)*'`,     // Single quotes
		"`(?:[^`\\\\]|\\\\.)*`", // Backticks
	}

	for _, pattern := range stringPatterns {
		stringRegex := regexp.MustCompile(pattern)
		for _, match := range stringRegex.FindAllStringIndex(line, -1) {
			stringStyle := lipgloss.NewStyle().Foreground(cr.stringColors["default"])
			addSegment(match[0], match[1], stringStyle, line[match[0]:match[1]])
		}
	}

	// 3. Comments (must come after strings to avoid false positives)
	commentPatterns := map[string][]*regexp.Regexp{
		"go": {
			regexp.MustCompile(`//.*$`),
			regexp.MustCompile(`/\*.*?\*/`),
		},
		"python": {
			regexp.MustCompile(`#.*$`),
			regexp.MustCompile(`'''[\s\S]*?'''`),
			regexp.MustCompile(`"""[\s\S]*?"""`),
		},
		"javascript": {
			regexp.MustCompile(`//.*$`),
			regexp.MustCompile(`/\*.*?\*/`),
		},
		"java": {
			regexp.MustCompile(`//.*$`),
			regexp.MustCompile(`/\*.*?\*/`),
			regexp.MustCompile(`/\*\*.*?\*/`), // Javadoc
		},
		"rust": {
			regexp.MustCompile(`//.*$`),
			regexp.MustCompile(`/\*.*?\*/`),
			regexp.MustCompile(`///.*$`), // Doc comments
		},
		"cpp": {
			regexp.MustCompile(`//.*$`),
			regexp.MustCompile(`/\*.*?\*/`),
		},
		"sql": {
			regexp.MustCompile(`--.*$`),
			regexp.MustCompile(`/\*.*?\*/`),
		},
	}

	if patterns, exists := commentPatterns[language]; exists {
		commentStyle := lipgloss.NewStyle().
			Foreground(cr.commentColors["default"]).
			Italic(true)

		for _, commentRegex := range patterns {
			for _, match := range commentRegex.FindAllStringIndex(line, -1) {
				addSegment(match[0], match[1], commentStyle, line[match[0]:match[1]])
			}
		}
	}

	// 4. Keywords and built-in types
	keywordPatterns := map[string][]string{
		"go": {
			"func", "var", "const", "type", "package", "import", "if", "else",
			"for", "range", "return", "defer", "go", "select", "case", "default",
			"switch", "fallthrough", "break", "continue", "goto", "interface",
			"struct", "map", "chan", "nil", "true", "false",
		},
		"python": {
			"def", "class", "import", "from", "if", "else", "elif", "for",
			"while", "return", "try", "except", "finally", "raise", "with",
			"as", "pass", "break", "continue", "lambda", "yield", "global",
			"nonlocal", "assert", "del", "in", "is", "and", "or", "not",
			"True", "False", "None",
		},
		"javascript": {
			"function", "var", "let", "const", "if", "else", "for", "while",
			"return", "class", "import", "export", "default", "from", "async",
			"await", "new", "this", "super", "extends", "static", "try",
			"catch", "finally", "throw", "typeof", "instanceof", "in", "of",
			"true", "false", "null", "undefined",
		},
		"java": {
			"public", "private", "protected", "class", "interface", "extends",
			"implements", "if", "else", "for", "while", "return", "try",
			"catch", "finally", "throw", "throws", "new", "this", "super",
			"static", "final", "abstract", "synchronized", "volatile",
			"transient", "native", "strictfp", "package", "import", "void",
			"boolean", "byte", "char", "short", "int", "long", "float",
			"double", "true", "false", "null",
		},
		"rust": {
			"fn", "let", "mut", "struct", "enum", "impl", "trait", "if",
			"else", "for", "while", "loop", "return", "match", "use", "mod",
			"pub", "crate", "self", "super", "static", "const", "unsafe",
			"async", "await", "dyn", "move", "ref", "type", "where", "as",
			"break", "continue", "extern", "in", "true", "false",
		},
		"sql": {
			"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER",
			"OUTER", "ON", "AS", "INSERT", "INTO", "VALUES", "UPDATE",
			"SET", "DELETE", "CREATE", "TABLE", "ALTER", "DROP", "INDEX",
			"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "NOT", "NULL",
			"UNIQUE", "DEFAULT", "CHECK", "CONSTRAINT", "AND", "OR", "IN",
			"BETWEEN", "LIKE", "ORDER", "BY", "GROUP", "HAVING", "UNION",
			"DISTINCT", "LIMIT", "OFFSET", "CASE", "WHEN", "THEN", "ELSE", "END",
		},
	}

	// 5. Types and built-in functions
	typePatterns := map[string][]string{
		"go": {
			"string", "int", "int8", "int16", "int32", "int64", "uint",
			"uint8", "uint16", "uint32", "uint64", "float32", "float64",
			"complex64", "complex128", "byte", "rune", "bool", "error",
			"uintptr", "any", "comparable",
		},
		"python": {
			"str", "int", "float", "bool", "list", "dict", "set", "tuple",
			"bytes", "bytearray", "complex", "frozenset", "range", "type",
			"object", "property", "staticmethod", "classmethod",
		},
		"javascript": {
			"Array", "Object", "String", "Number", "Boolean", "Function",
			"Symbol", "Date", "RegExp", "Error", "Map", "Set", "WeakMap",
			"WeakSet", "Promise", "Proxy", "Reflect",
		},
	}

	// 6. Function/method calls
	funcCallRegex := regexp.MustCompile(`\b(\w+)\s*\(`)
	for _, match := range funcCallRegex.FindAllStringSubmatch(line, -1) {
		if len(match) > 1 {
			funcName := match[1]
			funcIndex := strings.Index(line, funcName+"(")
			if funcIndex >= 0 {
				funcStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("87")) // Light blue
				addSegment(funcIndex, funcIndex+len(funcName), funcStyle, funcName)
			}
		}
	}

	// Apply keywords
	if keywords, exists := keywordPatterns[language]; exists {
		keywordColor := cr.keywordColors[language]
		if keywordColor == nil {
			keywordColor = lipgloss.Color("33") // Default yellow
		}
		keywordStyle := lipgloss.NewStyle().Foreground(keywordColor).Bold(true)

		for _, keyword := range keywords {
			pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(keyword))
			keywordRegex := regexp.MustCompile(pattern)
			for _, match := range keywordRegex.FindAllStringIndex(line, -1) {
				addSegment(match[0], match[1], keywordStyle, line[match[0]:match[1]])
			}
		}
	}

	// Apply types
	if types, exists := typePatterns[language]; exists {
		typeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Blue
		for _, typeName := range types {
			pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(typeName))
			typeRegex := regexp.MustCompile(pattern)
			for _, match := range typeRegex.FindAllStringIndex(line, -1) {
				addSegment(match[0], match[1], typeStyle, line[match[0]:match[1]])
			}
		}
	}

	// 7. Operators and punctuation
	operatorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // Orange
	operatorRegex := regexp.MustCompile(`[+\-*/%=<>!&|^~?:]+`)
	for _, match := range operatorRegex.FindAllStringIndex(line, -1) {
		addSegment(match[0], match[1], operatorStyle, line[match[0]:match[1]])
	}

	// Sort segments by start position
	for i := 0; i < len(segments)-1; i++ {
		for j := i + 1; j < len(segments); j++ {
			if segments[i].start > segments[j].start {
				segments[i], segments[j] = segments[j], segments[i]
			}
		}
	}

	// Build the highlighted line
	var result string
	lastEnd := 0
	for _, seg := range segments {
		if seg.start > lastEnd {
			result += line[lastEnd:seg.start]
		}
		result += seg.style.Render(seg.text)
		lastEnd = seg.end
	}
	if lastEnd < len(line) {
		result += line[lastEnd:]
	}

	return result
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

// ProcessInlineCode detects and highlights inline code snippets
func (cr *CodeRenderer) ProcessInlineCode(content string) string {
	// Pattern for inline code (backticks)
	inlineCodeRegex := regexp.MustCompile("`([^`]+)`")

	inlineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("75")).
		Padding(0, 1)

	return inlineCodeRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Extract code without backticks
		code := match[1 : len(match)-1]

		// Apply simple highlighting for common patterns
		highlighted := cr.highlightInlineCode(code)

		return inlineStyle.Render(highlighted)
	})
}

// highlightInlineCode applies simple highlighting to inline code
func (cr *CodeRenderer) highlightInlineCode(code string) string {
	// Detect if it's a function call
	if strings.Contains(code, "(") && strings.Contains(code, ")") {
		funcRegex := regexp.MustCompile(`(\w+)\(`)
		code = funcRegex.ReplaceAllStringFunc(code, func(match string) string {
			funcName := match[:len(match)-1]
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("87")).
				Bold(true).
				Render(funcName) + "("
		})
	}

	// Highlight common keywords
	commonKeywords := []string{"func", "def", "class", "type", "var", "const", "let"}
	for _, keyword := range commonKeywords {
		if strings.HasPrefix(code, keyword+" ") {
			parts := strings.SplitN(code, " ", 2)
			if len(parts) == 2 {
				keywordStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("33")).
					Bold(true)
				code = keywordStyle.Render(parts[0]) + " " + parts[1]
			}
		}
	}

	return code
}

// RenderWithTheme renders code with a specific color theme
func (cr *CodeRenderer) RenderWithTheme(code, language, theme string) string {
	// Save current colors
	oldKeywordColors := cr.keywordColors
	oldStringColors := cr.stringColors
	oldCommentColors := cr.commentColors

	// Apply theme
	switch theme {
	case "monokai":
		cr.keywordColors = map[string]color.Color{
			"default": lipgloss.Color("197"), // Pink
		}
		cr.stringColors = map[string]color.Color{
			"default": lipgloss.Color("226"), // Yellow
		}
		cr.commentColors = map[string]color.Color{
			"default": lipgloss.Color("242"), // Gray
		}
	case "solarized":
		cr.keywordColors = map[string]color.Color{
			"default": lipgloss.Color("33"), // Yellow
		}
		cr.stringColors = map[string]color.Color{
			"default": lipgloss.Color("37"), // Cyan
		}
		cr.commentColors = map[string]color.Color{
			"default": lipgloss.Color("244"), // Gray
		}
	case "dracula":
		cr.keywordColors = map[string]color.Color{
			"default": lipgloss.Color("212"), // Pink
		}
		cr.stringColors = map[string]color.Color{
			"default": lipgloss.Color("226"), // Yellow
		}
		cr.commentColors = map[string]color.Color{
			"default": lipgloss.Color("103"), // Purple gray
		}
	}

	// Render code
	result := cr.renderCode(code, language, cr.showLineNumbers)

	// Restore colors
	cr.keywordColors = oldKeywordColors
	cr.stringColors = oldStringColors
	cr.commentColors = oldCommentColors

	return result
}

// GetSupportedLanguages returns a list of supported languages
func (cr *CodeRenderer) GetSupportedLanguages() []string {
	languages := make([]string, 0, len(cr.keywordColors))
	for lang := range cr.keywordColors {
		languages = append(languages, lang)
	}
	return languages
}

// GetSupportedThemes returns a list of supported color themes
func (cr *CodeRenderer) GetSupportedThemes() []string {
	return []string{"default", "monokai", "solarized", "dracula"}
}
