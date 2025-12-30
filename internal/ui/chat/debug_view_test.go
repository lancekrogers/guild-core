package chat

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/guild-framework/guild-core/pkg/config"
)

func escVisible(s string) string {
    b := strings.Builder{}
    for i := 0; i < len(s); i++ {
        switch s[i] {
        case '\x1b':
            b.WriteString("<ESC>")
        case '\n':
            b.WriteString("<NL>\n")
        case '\r':
            b.WriteString("<CR>")
        default:
            b.WriteByte(s[i])
        }
    }
    return b.String()
}

// Temporary debug helper to inspect the initial View() output without the renderer.
func TestDebugInitialView(t *testing.T) {
	if os.Getenv("GUILD_DEBUG_VIEW") == "" {
		t.Skip("debug only")
	}

	lipgloss.SetColorProfile(termenv.TrueColor)

    // isolate state
    _ = os.MkdirAll(".home", 0o755)
    os.Setenv("HOME", ".home")

	cfg := config.DefaultGuildTemplate()
	app := NewApp(context.Background(), cfg, nil)

	if wStr := os.Getenv("GUILD_DEBUG_WIDTH"); wStr != "" && app.config != nil {
		if w, err := strconv.Atoi(wStr); err == nil {
			app.config.Width = w
		}
	}
	if hStr := os.Getenv("GUILD_DEBUG_HEIGHT"); hStr != "" && app.config != nil {
		if h, err := strconv.Atoi(hStr); err == nil {
			app.config.Height = h
		}
	}

	// Optionally disable rich content to isolate ANSI issues.
	if os.Getenv("GUILD_DEBUG_PLAIN") != "" && app.config != nil {
		app.config.EnableRichContent = false
		app.config.MarkdownEnabled = false
	}

	if err := app.initializeComponents(); err != nil {
		t.Fatalf("init components: %v", err)
	}

	app.Init()
	view := app.View()

	fmt.Println("RAW:\n" + view)
	fmt.Println("VISIBLE:\n" + escVisible(view))

	// Also dump individual panes for debugging so we can see where corruption originates.
	outputView := app.outputPane.View()
	inputView := app.inputPane.View()
	statusView := app.statusPane.View()

	fmt.Println("OUTPUT PANE:\n" + escVisible(outputView))
	fmt.Println("INPUT PANE:\n" + escVisible(inputView))
	fmt.Println("STATUS PANE:\n" + escVisible(statusView))
}
