// pkg/ui/objective/components/preview.go
package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DocumentPreview creates a preview of referenced documents
type DocumentPreview struct {
	documents map[string]string
	width     int
	height    int
}

// NewDocumentPreview creates a new document preview
func NewDocumentPreview(width, height int) *DocumentPreview {
	return &DocumentPreview{
		documents: make(map[string]string),
		width:     width,
		height:    height,
	}
}

// SetDocuments updates the documents to preview
func (dp *DocumentPreview) SetDocuments(docs map[string]string) {
	dp.documents = docs
}

// View renders the document preview
func (dp *DocumentPreview) View() string {
	var b strings.Builder

	// Style definitions
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#0000FF")).
		Padding(0, 1)

	contentStyle := lipgloss.NewStyle().
		Width(dp.width).
		MaxHeight(dp.height / len(dp.documents))

	// Render each document
	for path, content := range dp.documents {
		b.WriteString(titleStyle.Render(" " + path + " "))
		b.WriteString("\n\n")

		// Truncate content if needed to fit screen
		lines := strings.Split(content, "\n")
		maxLines := dp.height/len(dp.documents) - 5
		if len(lines) > maxLines {
			displayedLines := append(lines[:maxLines-1], "...", "")
			content = strings.Join(displayedLines, "\n")
		}

		b.WriteString(contentStyle.Render(content))
		b.WriteString("\n\n")
	}

	return b.String()
}
