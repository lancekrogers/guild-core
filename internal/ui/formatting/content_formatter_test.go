package formatting

import (
	"io"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestOptimizeContentLength_DoesNotCorruptANSI(t *testing.T) {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Italic(true)

	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString(style.Render("Hello world "))
	}
	content := b.String()

	cf := &ContentFormatter{
		maxContentLength: 50,
		showMoreEnabled:  true,
	}

	truncated := cf.OptimizeContentLength(content)
	if !utf8.ValidString(truncated) {
		t.Fatalf("expected OptimizeContentLength to return valid UTF-8")
	}

	stripped := ansi.Strip(truncated)
	if strings.Contains(stripped, "[0m") || strings.Contains(stripped, "[38;") {
		t.Fatalf("expected ANSI escapes to be intact; saw escape fragments in stripped output: %q", stripped)
	}

	if !strings.Contains(stripped, "... (") {
		t.Fatalf("expected show-more indicator to be present, got: %q", stripped)
	}
}

func TestFormatMessage_PanicDoesNotWriteToStdout(t *testing.T) {
	// Guard against corrupting TUIs: libraries should not print to stdout on errors/panics.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	cf := &ContentFormatter{
		// Intentionally nil to trigger a panic in ProcessContent.
		markdownRenderer: nil,
		showMoreEnabled:  false,
		maxContentLength: 0,
	}

	_ = cf.FormatMessage("system", "# Title", nil)

	_ = w.Close()
	out, _ := io.ReadAll(r)
	if len(out) != 0 {
		t.Fatalf("expected no stdout output, got %q", string(out))
	}
}
