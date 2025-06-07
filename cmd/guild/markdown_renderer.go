package main

import (
	"strings"
	"regexp"
	
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// MarkdownRenderer provides rich content rendering for Guild chat
type MarkdownRenderer struct {
	renderer   *glamour.TermRenderer
	width      int
	codeStyle  lipgloss.Style
	formatter  chroma.Formatter
	style      *chroma.Style
}

// NewMarkdownRenderer creates a new markdown renderer with medieval theming
func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	// Create glamour renderer with medieval-themed styling
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4), // Account for borders
		glamour.WithEmoji(),
	)
	if err != nil {
		// Fallback to basic renderer if glamour fails
		renderer, err = glamour.NewTermRenderer(
			glamour.WithWordWrap(width-4),
		)
		if err != nil {
			return nil, err
		}
	}

	// Create chroma formatter for syntax highlighting
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Get("terminal")
	}
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Use a dark style that works well in terminals (medieval theme)
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Get("native")
	}
	if style == nil {
		style = styles.Fallback
	}

	// Medieval-themed style for code blocks
	codeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")). // Medieval purple
		Padding(0, 1).
		Margin(1, 0).
		MaxWidth(width - 6) // Ensure proper wrapping

	return &MarkdownRenderer{
		renderer:  renderer,
		width:     width,
		codeStyle: codeStyle,
		formatter: formatter,
		style:     style,
	}, nil
}

// Render processes content with markdown and syntax highlighting
func (m *MarkdownRenderer) Render(content string) string {
	// Quick check if content needs markdown processing
	if !m.needsMarkdownProcessing(content) {
		return content
	}
	
	// First, extract and process code blocks separately for better syntax highlighting
	processedContent := m.processCodeBlocks(content)
	
	// Then render the markdown
	rendered, err := m.renderer.Render(processedContent)
	if err != nil {
		// Fallback to original content if rendering fails
		return content
	}
	
	return rendered
}

// needsMarkdownProcessing checks if content contains markdown elements
func (m *MarkdownRenderer) needsMarkdownProcessing(content string) bool {
	// Quick heuristics to avoid unnecessary processing
	markdownIndicators := []string{"```", "#", "*", "_", "[", "`", "1.", "-"}
	for _, indicator := range markdownIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}

// processCodeBlocks extracts code blocks and applies syntax highlighting
func (m *MarkdownRenderer) processCodeBlocks(content string) string {
	// Regex to match fenced code blocks
	codeBlockRegex := regexp.MustCompile("```(\\w+)?\\n([\\s\\S]*?)```")
	
	return codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := codeBlockRegex.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		
		language := parts[1]
		code := parts[2]
		
		// Apply syntax highlighting
		highlighted := m.highlightCode(code, language)
		
		// Wrap in styled border
		return m.codeStyle.Render(highlighted)
	})
}

// highlightCode applies syntax highlighting to code content
func (m *MarkdownRenderer) highlightCode(code, language string) string {
	// Handle empty code
	if strings.TrimSpace(code) == "" {
		return code
	}
	
	// Get lexer for the language
	var lexer chroma.Lexer
	if language != "" {
		lexer = lexers.Get(language)
		// Try common language aliases
		if lexer == nil {
			aliases := map[string]string{
				"golang": "go",
				"js": "javascript",
				"py": "python",
				"sh": "bash",
				"yml": "yaml",
			}
			if alias, exists := aliases[language]; exists {
				lexer = lexers.Get(alias)
			}
		}
	}
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	
	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code // Fallback to original code
	}
	
	// Format with syntax highlighting
	var buf strings.Builder
	err = m.formatter.Format(&buf, m.style, iterator)
	if err != nil {
		return code // Fallback to original code
	}
	
	return buf.String()
}

// RenderInlineCode applies highlighting to inline code snippets
func (m *MarkdownRenderer) RenderInlineCode(code string) string {
	inlineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("8")).   // Dark gray background
		Foreground(lipgloss.Color("15")).  // Bright white text
		Padding(0, 1)
	
	return inlineStyle.Render(code)
}

// DetectAndRenderContent automatically detects content type and applies appropriate rendering
func (m *MarkdownRenderer) DetectAndRenderContent(content string) string {
	// Check if content looks like markdown
	hasMarkdown := strings.Contains(content, "```") || 
	              strings.Contains(content, "#") ||
	              strings.Contains(content, "*") ||
	              strings.Contains(content, "_") ||
	              strings.Contains(content, "[") ||
	              strings.Contains(content, "`")
	
	if hasMarkdown {
		return m.Render(content)
	}
	
	// For plain text, just return as-is (already styled by lipgloss)
	return content
}