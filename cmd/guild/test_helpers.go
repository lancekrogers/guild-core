package main

import (
	"fmt"
	"testing"
	"time"
	"github.com/guild-ventures/guild-core/pkg/config"
)

func createTestConfig() *config.GuildConfig {
	return &config.GuildConfig{
		Name: "test-guild",
		Agents: []config.AgentConfig{
			{
				ID:   "manager",
				Name: "Test Manager",
				Type: "manager",
				Provider: "mock",
				Capabilities: []string{"planning", "coordination"},
			},
			{
				ID:   "developer",
				Name: "Test Developer",
				Type: "worker",
				Provider: "mock",
				Capabilities: []string{"coding", "testing"},
			},
		},
	}
}

func setupTestEnvironment(t *testing.T) (*AgentStatusTracker, *StatusDisplay, *MarkdownRenderer) {
	config := createTestConfig()
	tracker := NewAgentStatusTracker(config)
	display := NewStatusDisplay(tracker, 80, 24)
	renderer, err := NewMarkdownRenderer(80)
	if err != nil {
		t.Fatalf("Failed to create markdown renderer: %v", err)
	}
	return tracker, display, renderer
}

// Mock agent status for testing
func createMockAgentStatus(id, name string, state AgentState) *AgentStatus {
	return &AgentStatus{
		ID:        id,
		Name:      name,
		State:     state,
		StartTime: time.Now(),
	}
}

// Test data generators
func generateTestMarkdown() string {
	return `# Test Document

This is a test document with various **markdown** elements.

## Code Example

` + "```go" + `
func TestFunction() {
    fmt.Println("Hello, World!")
}
` + "```" + `

## Lists

- Item 1
- Item 2
- Item 3

## Links and Emphasis

Visit [Guild Framework](https://example.com) for more *information*.`
}

func generateTestCode(language string) string {
	switch language {
	case "go":
		return `package main

import "fmt"

func main() {
    fmt.Println("Hello from Go!")
}`
	case "python":
		return `def main():
    print("Hello from Python!")

if __name__ == "__main__":
    main()`
	case "javascript":
		return `function main() {
    console.log("Hello from JavaScript!");
}

main();`
	case "rust":
		return `fn main() {
    println!("Hello from Rust!");
}`
	default:
		return "// Code sample"
	}
}

// Visual validation helpers
func validateVisualOutput(t *testing.T, output string, expectedElements []string) {
	t.Helper()
	
	if output == "" {
		t.Error("Visual output is empty")
		return
	}
	
	for _, element := range expectedElements {
		if !containsString(output, element) {
			t.Errorf("Expected output to contain '%s', but it doesn't", element)
		}
	}
}

func validateNoVisualCorruption(t *testing.T, output string) {
	t.Helper()
	
	// Check for common visual corruption patterns
	corruptionPatterns := []string{
		"\x1b[0m\x1b[0m",  // Double escape sequences
		"\x00",            // Null bytes
		"\x1b[m\x1b[m",    // Malformed escapes
	}
	
	for _, pattern := range corruptionPatterns {
		if containsString(output, pattern) {
			t.Errorf("Visual corruption detected: found '%s' in output", pattern)
		}
	}
}

// Performance measurement helpers
func measureRenderTime(t *testing.T, name string, renderFunc func() string) (string, time.Duration) {
	t.Helper()
	
	start := time.Now()
	result := renderFunc()
	duration := time.Since(start)
	
	t.Logf("%s render time: %v", name, duration)
	
	return result, duration
}

// Agent simulation helpers
func simulateAgentActivity(tracker *AgentStatusTracker, agentID string, duration time.Duration) {
	states := []AgentState{
		AgentThinking,
		AgentWorking,
		AgentThinking, // Replace AgentReviewing which doesn't exist
		AgentWorking,  // Replace AgentCompleted which doesn't exist
	}
	
	ticker := time.NewTicker(duration / 4)
	defer ticker.Stop()
	
	for i, state := range states {
		tracker.UpdateAgentStatus(agentID, &AgentStatus{
			ID:          agentID,
			Name:        "Test Agent",
			State:       state,
			CurrentTask: "Task phase " + string(rune('A'+i)),
			Progress:    float64(i+1) / 4.0,
		})
		
		if i < len(states)-1 {
			<-ticker.C
		}
	}
}

// Helper function for string containment - renamed to avoid conflict
func containsString(str, substr string) bool {
	return len(str) >= len(substr) && containsStringAt(str, substr, 0)
}

func containsStringAt(str, substr string, start int) bool {
	if start+len(substr) > len(str) {
		return false
	}
	for i := 0; i < len(substr); i++ {
		if str[start+i] != substr[i] {
			return false
		}
	}
	return true
}

// Medieval theme validation
func validateMedievalTheme(t *testing.T, output string) {
	t.Helper()
	
	// Check for medieval-themed elements
	medievalElements := []string{
		"Guild",
		"Artisan",
		"Master",
		"⚔️",  // Sword
		"🛡️",  // Shield
		"🏰",  // Castle
		"👑",  // Crown
		"📜",  // Scroll
	}
	
	foundAny := false
	for _, element := range medievalElements {
		if containsString(output, element) {
			foundAny = true
			break
		}
	}
	
	if !foundAny {
		t.Log("Warning: No medieval theme elements found in output")
	}
}

// Color support detection mock
func mockDetectColorSupport() ColorSupport {
	return ColorSupport{
		TrueColor:  true,
		Color256:   true,
		BasicColor: true,
	}
}

type ColorSupport struct {
	TrueColor  bool
	Color256   bool
	BasicColor bool
}

// Terminal capability mocks
func mockTerminalCapabilities() TerminalCapabilities {
	return TerminalCapabilities{
		Width:          80,
		Height:         24,
		UnicodeSupport: true,
		ColorSupport:   mockDetectColorSupport(),
	}
}

type TerminalCapabilities struct {
	Width          int
	Height         int
	UnicodeSupport bool
	ColorSupport   ColorSupport
}

// Benchmark helpers
func setupBenchmarkEnvironment() (*AgentStatusTracker, *StatusDisplay, *MarkdownRenderer, *ContentFormatter) {
	config := createTestConfig()
	tracker := NewAgentStatusTracker(config)
	display := NewStatusDisplay(tracker, 80, 24)
	renderer, _ := NewMarkdownRenderer(80)
	formatter := NewContentFormatter(renderer, 80)
	
	// Pre-populate with test data
	for i := 0; i < 5; i++ {
		tracker.UpdateAgentStatus(fmt.Sprintf("agent-%d", i), &AgentStatus{
			ID:    fmt.Sprintf("agent-%d", i),
			Name:  fmt.Sprintf("Agent %d", i),
			State: AgentWorking,
		})
	}
	
	return tracker, display, renderer, formatter
}