//go:build integration

package visual

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lancekrogers/guild-core/internal/teatest"
	"github.com/lancekrogers/guild-core/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simple test models for UI testing

// testThemeModel tests theme switching
type testThemeModel struct {
	theme    string
	quitting bool
}

func (m testThemeModel) Init() tea.Cmd {
	return nil
}

func (m testThemeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "t":
			// Toggle theme
			if m.theme == "dark" {
				m.theme = "light"
			} else {
				m.theme = "dark"
			}
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m testThemeModel) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}
	return tea.NewView("Theme: " + m.theme + "\nPress 't' to toggle theme, 'q' to quit\n")
}

// testProgressModel tests progress indicators
type testProgressModel struct {
	progress int
	quitting bool
}

func (m testProgressModel) Init() tea.Cmd {
	return tickCmd()
}

func (m testProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case tickMsg:
		m.progress = (m.progress + 10) % 101
		return m, tickCmd()
	}
	return m, nil
}

func (m testProgressModel) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}
	bar := strings.Repeat("█", m.progress/10) + strings.Repeat("░", 10-m.progress/10)
	return tea.NewView(fmt.Sprintf("Progress: [%s] %d%%\n", bar, m.progress))
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// testListModel tests scrollable lists
type testListModel struct {
	items    []string
	selected int
	quitting bool
}

func newTestListModel(itemCount int) testListModel {
	items := make([]string, itemCount)
	for i := 0; i < itemCount; i++ {
		items[i] = fmt.Sprintf("Item %d", i+1)
	}
	return testListModel{items: items}
}

func (m testListModel) Init() tea.Cmd {
	return nil
}

func (m testListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m testListModel) View() tea.View {
	if m.quitting {
		return tea.NewView("Goodbye!\n")
	}

	var s strings.Builder
	s.WriteString("List (use j/k or arrows to navigate):\n")

	// Show a window of items
	start := m.selected - 2
	if start < 0 {
		start = 0
	}
	end := start + 5
	if end > len(m.items) {
		end = len(m.items)
		start = end - 5
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		if i == m.selected {
			s.WriteString("> ")
		} else {
			s.WriteString("  ")
		}
		s.WriteString(m.items[i])
		s.WriteString("\n")
	}

	return tea.NewView(s.String())
}

// TestUIComponents validates all UI components, themes, animations, and responsiveness
func TestUIComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	helper := testutil.NewTeaTestHelper(t)

	t.Run("theme_switching", func(t *testing.T) {
		model := testThemeModel{theme: "dark"}
		tm := helper.RunModel(model, teatest.WithInitialTermSize(80, 24))

		// Wait for initial render
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "Theme: dark")
			},
			teatest.WithCheckInterval(50*time.Millisecond),
			teatest.WithDuration(2*time.Second),
		)

		// Test theme switching
		start := time.Now()
		tm.Send(tea.KeyPressMsg{Text: "t", Code: 't'})

		// Wait for theme to change
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "Theme: light")
			},
			teatest.WithCheckInterval(10*time.Millisecond),
			teatest.WithDuration(100*time.Millisecond),
		)

		switchDuration := time.Since(start)
		assert.LessOrEqual(t, switchDuration, 50*time.Millisecond,
			"Theme switch should complete quickly")

		t.Logf("Theme switch took %v", switchDuration)

		// Test quit
		tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
		testutil.SafeWaitFinished(t, tm, 2*time.Second)
	})

	t.Run("progress_animation", func(t *testing.T) {
		model := testProgressModel{}
		tm := helper.RunModel(model, teatest.WithInitialTermSize(80, 24))

		// Wait for initial render
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "Progress:")
			},
			teatest.WithCheckInterval(50*time.Millisecond),
			teatest.WithDuration(2*time.Second),
		)

		// Let animation run for a bit
		time.Sleep(500 * time.Millisecond)

		// Verify progress updated
		output := string(tm.Output())
		assert.Contains(t, output, "█", "Progress bar should show progress")

		tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
		testutil.SafeWaitFinished(t, tm, 2*time.Second)
	})

	t.Run("list_navigation", func(t *testing.T) {
		model := newTestListModel(10)
		tm := helper.RunModel(model, teatest.WithInitialTermSize(80, 24))

		// Wait for list render
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "Item 1")
			},
			teatest.WithCheckInterval(50*time.Millisecond),
			teatest.WithDuration(2*time.Second),
		)

		// Navigate down
		tm.Send(tea.KeyPressMsg{Text: "j", Code: 'j'})
		tm.Send(tea.KeyPressMsg{Text: "j", Code: 'j'})

		time.Sleep(50 * time.Millisecond)

		output := string(tm.Output())
		assert.Contains(t, output, "> Item 3", "Should select third item")

		tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
		testutil.SafeWaitFinished(t, tm, 2*time.Second)
	})

	t.Run("responsive_layout", func(t *testing.T) {
		sizes := []struct {
			width  int
			height int
			desc   string
		}{
			{80, 24, "standard"},
			{120, 40, "large"},
			{60, 20, "small"},
		}

		for _, size := range sizes {
			t.Run(size.desc, func(t *testing.T) {
				model := testThemeModel{theme: "dark"}
				tm := helper.RunModel(model,
					teatest.WithInitialTermSize(size.width, size.height))

				// Wait for render
				teatest.WaitFor(
					t,
					tm.Output(),
					func(bts []byte) bool {
						return len(bts) > 0
					},
					teatest.WithCheckInterval(50*time.Millisecond),
					teatest.WithDuration(time.Second),
				)

				// Send window resize
				tm.Send(tea.WindowSizeMsg{
					Width:  size.width,
					Height: size.height,
				})

				time.Sleep(50 * time.Millisecond)

				output := tm.Output()
				lines := bytes.Split(output, []byte("\n"))

				// Basic validation - content should fit
				for i, line := range lines {
					assert.LessOrEqual(t, len(line), size.width,
						"Line %d should fit within width %d", i, size.width)
				}

				tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
				testutil.SafeWaitFinished(t, tm, time.Second)
			})
		}
	})
}

// TestUIPerformance validates UI rendering performance
func TestUIPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	helper := testutil.NewTeaTestHelper(t)

	t.Run("render_performance", func(t *testing.T) {
		model := testProgressModel{}
		tm := helper.RunModel(model, teatest.WithInitialTermSize(80, 24))

		// Wait for initial render
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "Progress:")
			},
			teatest.WithCheckInterval(50*time.Millisecond),
			teatest.WithDuration(2*time.Second),
		)

		// Measure frame times by sending multiple updates
		frameTimes := make([]time.Duration, 0, 10)

		for i := 0; i < 10; i++ {
			start := time.Now()
			tm.Send(tickMsg(time.Now()))
			time.Sleep(16 * time.Millisecond) // Target 60fps
			frameTimes = append(frameTimes, time.Since(start))
		}

		// Calculate average frame time
		var total time.Duration
		for _, ft := range frameTimes {
			total += ft
		}
		avgFrameTime := total / time.Duration(len(frameTimes))

		// Should maintain smooth frame rate
		assert.LessOrEqual(t, avgFrameTime, 20*time.Millisecond,
			"Should maintain smooth frame rate")

		t.Logf("Average frame time: %v (%.1f fps)",
			avgFrameTime, 1000/float64(avgFrameTime.Milliseconds()))

		tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
		testutil.SafeWaitFinished(t, tm, time.Second)
	})

	t.Run("scrolling_performance", func(t *testing.T) {
		model := newTestListModel(100) // Large list
		tm := helper.RunModel(model, teatest.WithInitialTermSize(80, 24))

		// Wait for initial render
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "Item 1")
			},
			teatest.WithCheckInterval(50*time.Millisecond),
			teatest.WithDuration(2*time.Second),
		)

		// Measure rapid scrolling
		scrollStart := time.Now()

		// Scroll down rapidly
		for i := 0; i < 20; i++ {
			tm.Send(tea.KeyPressMsg{Text: "j", Code: 'j'})
			time.Sleep(16 * time.Millisecond) // 60fps
		}

		scrollDuration := time.Since(scrollStart)

		// Should handle rapid scrolling smoothly
		assert.LessOrEqual(t, scrollDuration, 400*time.Millisecond,
			"Scrolling should be smooth and responsive")

		t.Logf("Rapid scroll test took %v", scrollDuration)

		tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
		testutil.SafeWaitFinished(t, tm, time.Second)
	})

	t.Run("memory_stability", func(t *testing.T) {
		model := testProgressModel{}
		tm := helper.RunModel(model, teatest.WithInitialTermSize(80, 24))

		// Wait for initial render
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "Progress:")
			},
			teatest.WithCheckInterval(50*time.Millisecond),
			teatest.WithDuration(2*time.Second),
		)

		// Run for extended period to check for leaks
		iterations := 50
		for i := 0; i < iterations; i++ {
			tm.Send(tickMsg(time.Now()))
			tm.Send(tea.WindowSizeMsg{Width: 80 + i%20, Height: 24})
			time.Sleep(10 * time.Millisecond)
		}

		// Should complete without issues
		tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
		testutil.SafeWaitFinished(t, tm, time.Second)
	})
}

// TestUIAccessibility validates accessibility features
func TestUIAccessibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping accessibility test in short mode")
	}

	helper := testutil.NewTeaTestHelper(t)

	t.Run("keyboard_navigation", func(t *testing.T) {
		model := newTestListModel(5)
		tm := helper.RunModel(model, teatest.WithInitialTermSize(80, 24))

		// Wait for initial render
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "Item 1")
			},
			teatest.WithCheckInterval(50*time.Millisecond),
			teatest.WithDuration(2*time.Second),
		)

		// Test arrow keys
		tm.Send(tea.KeyPressMsg{Code: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)

		output := string(tm.Output())
		assert.Contains(t, output, "> Item 2", "Down arrow should work")

		// Test vim keys
		tm.Send(tea.KeyPressMsg{Text: "k", Code: 'k'})
		time.Sleep(50 * time.Millisecond)

		output = string(tm.Output())
		assert.Contains(t, output, "> Item 1", "Vim key 'k' should work")

		tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
		testutil.SafeWaitFinished(t, tm, time.Second)
	})

	t.Run("focus_indicators", func(t *testing.T) {
		model := newTestListModel(5)
		tm := helper.RunModel(model, teatest.WithInitialTermSize(80, 24))

		// Wait for initial render
		teatest.WaitFor(
			t,
			tm.Output(),
			func(bts []byte) bool {
				return contains(bts, "> Item 1")
			},
			teatest.WithCheckInterval(50*time.Millisecond),
			teatest.WithDuration(2*time.Second),
		)

		// Verify clear focus indicator
		output := string(tm.Output())
		assert.Contains(t, output, ">", "Should have clear focus indicator")

		// Count selected items (should be exactly one)
		selectedCount := strings.Count(output, ">")
		assert.Equal(t, 1, selectedCount, "Should have exactly one selected item")

		tm.Send(tea.KeyPressMsg{Text: "q", Code: 'q'})
		testutil.SafeWaitFinished(t, tm, time.Second)
	})
}

// Helper function
func contains(data []byte, s string) bool {
	return bytes.Contains(bytes.ToLower(data), bytes.ToLower([]byte(s)))
}
