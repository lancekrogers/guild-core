// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// WizardUI provides enhanced UI components for the setup wizard
type WizardUI struct {
	wizard *Wizard
}

// NewWizardUI creates a new wizard UI helper
func NewWizardUI(wizard *Wizard) *WizardUI {
	return &WizardUI{
		wizard: wizard,
	}
}

// ShowWelcomeScreen displays the welcome message with ASCII art
func (ui *WizardUI) ShowWelcomeScreen() {
	if ui.wizard.config.QuickMode {
		return
	}

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  🏰 GUILD SETUP WIZARD 🏰                      ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("║                                                               ║")
	fmt.Println("║  Welcome to the Guild Framework!                              ║")
	fmt.Println("║                                                               ║")
	fmt.Println("║  This wizard will help you:                                   ║")
	fmt.Println("║  • Detect available AI providers                              ║")
	fmt.Println("║  • Configure API credentials                                  ║")
	fmt.Println("║  • Select optimal models for your needs                       ║")
	fmt.Println("║  • Create your team of specialized AI agents                  ║")
	fmt.Println("║                                                               ║")
	fmt.Println("║  Setup time: ~2 minutes                                       ║")
	fmt.Println("║                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// ShowProviderSelectionHelp displays help for provider selection
func (ui *WizardUI) ShowProviderSelectionHelp() {
	if ui.wizard.config.QuickMode {
		return
	}

	fmt.Println("📚 Provider Selection Help:")
	fmt.Println("   • Enter provider numbers separated by spaces (e.g., '1 3 5')")
	fmt.Println("   • Type 'all' to select all detected providers")
	fmt.Println("   • Press Enter to use all providers (recommended)")
	fmt.Println("   • Providers with ✅ have valid credentials")
	fmt.Println("   • Providers with 🏠 run locally on your machine")
	fmt.Println()
}

// ShowModelSelectionHelp displays help for model selection
func (ui *WizardUI) ShowModelSelectionHelp(providerName string) {
	if ui.wizard.config.QuickMode {
		return
	}

	fmt.Printf("📚 Model Selection Help for %s:\n", providerName)
	fmt.Println("   • Enter model numbers separated by spaces")
	fmt.Println("   • Type 'recommended' to use ⭐ marked models")
	fmt.Println("   • Press Enter to use recommended models")
	fmt.Println("   • Cost shown is per 1,000 tokens")
	fmt.Println()
}

// ShowPresetSelectionHelp displays help for preset selection
func (ui *WizardUI) ShowPresetSelectionHelp() {
	if ui.wizard.config.QuickMode {
		return
	}

	fmt.Println("📚 Agent Preset Selection Help:")
	fmt.Println("   • Presets are pre-configured teams optimized for specific tasks")
	fmt.Println("   • Match % shows compatibility with your providers")
	fmt.Println("   • Higher match % = better performance")
	fmt.Println("   • Press Enter to use the top recommendation")
	fmt.Println()
}

// ShowProgressBar displays a progress bar for long operations
func (ui *WizardUI) ShowProgressBar(ctx context.Context, message string, steps int) chan<- int {
	if ui.wizard.config.QuickMode {
		// Return a dummy channel for quick mode
		ch := make(chan int, steps)
		return ch
	}

	progress := make(chan int, 1)

	go func() {
		defer close(progress)
		
		width := 50
		current := 0

		fmt.Printf("\n%s\n", message)

		for {
			select {
			case <-ctx.Done():
				fmt.Println("\n⚠️  Operation cancelled")
				return
			case step, ok := <-progress:
				if !ok {
					return
				}
				current = step
				
				// Calculate progress
				percent := float64(current) / float64(steps) * 100
				filled := int(float64(width) * float64(current) / float64(steps))
				
				// Create progress bar
				bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
				
				// Display progress
				fmt.Printf("\r[%s] %3.0f%% (%d/%d)", bar, percent, current, steps)
				
				if current >= steps {
					fmt.Println(" ✅")
					return
				}
			}
		}
	}()

	return progress
}

// ConfirmAction asks for user confirmation
func (ui *WizardUI) ConfirmAction(ctx context.Context, message string, defaultYes bool) (bool, error) {
	if ui.wizard.config.QuickMode {
		return true, nil // Always confirm in quick mode
	}

	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}

	fmt.Printf("%s (%s): ", message, defaultStr)

	input, err := ui.wizard.readLineWithTimeout(ctx, ui.wizard.inputTimeout)
	if err != nil {
		// On timeout or error, use default
		if gerror.Is(err, gerror.ErrCodeTimeout) || gerror.Is(err, gerror.ErrCodeCancelled) {
			return defaultYes, nil
		}
		return false, err
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes, nil
	}

	return input == "y" || input == "yes", nil
}

// ShowError displays an error message with formatting
func (ui *WizardUI) ShowError(err error, context string) {
	if ui.wizard.config.QuickMode {
		// Minimal error display in quick mode
		fmt.Printf("❌ %s: %v\n", context, err)
		return
	}

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Printf("║ ❌ ERROR: %-51s ║\n", truncateString(context, 51))
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	
	// Wrap error message
	lines := wrapText(err.Error(), 61)
	for _, line := range lines {
		fmt.Printf("║ %-61s ║\n", line)
	}
	
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// ShowSuccess displays a success message
func (ui *WizardUI) ShowSuccess(message string) {
	if ui.wizard.config.QuickMode {
		fmt.Printf("✅ %s\n", message)
		return
	}

	fmt.Printf("\n✅ %s\n\n", message)
}

// ShowWarning displays a warning message
func (ui *WizardUI) ShowWarning(message string) {
	if ui.wizard.config.QuickMode {
		fmt.Printf("⚠️  %s\n", message)
		return
	}

	fmt.Printf("\n⚠️  Warning: %s\n\n", message)
}

// ShowInfo displays an informational message
func (ui *WizardUI) ShowInfo(message string) {
	if ui.wizard.config.QuickMode {
		return // Skip info messages in quick mode
	}

	fmt.Printf("ℹ️  %s\n", message)
}

// ShowSection displays a section header
func (ui *WizardUI) ShowSection(title string) {
	if ui.wizard.config.QuickMode {
		return
	}

	fmt.Println()
	fmt.Printf("═══ %s ═══\n", title)
	fmt.Println()
}

// ShowCompletionSummary displays a detailed completion summary
func (ui *WizardUI) ShowCompletionSummary(providers []ConfiguredProvider, agents []config.AgentConfig) {
	if ui.wizard.config.QuickMode {
		// Minimal summary in quick mode
		fmt.Printf("✅ Setup complete: %d providers, %d agents configured\n", 
			len(providers), len(agents))
		return
	}

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              🎉 GUILD SETUP COMPLETE! 🎉                      ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	
	// Providers summary
	fmt.Printf("║ Providers Configured: %-39d ║\n", len(providers))
	for _, provider := range providers {
		line := fmt.Sprintf("  • %s (%d models)", provider.Name, len(provider.Models))
		fmt.Printf("║ %-61s ║\n", line)
	}
	
	fmt.Println("║                                                               ║")
	
	// Agents summary
	fmt.Printf("║ Agents Created: %-45d ║\n", len(agents))
	for _, agent := range agents {
		line := fmt.Sprintf("  • %s (%s)", agent.Name, agent.Type)
		fmt.Printf("║ %-61s ║\n", truncateString(line, 61))
	}
	
	fmt.Println("║                                                               ║")
	fmt.Println("║ Next Steps:                                                   ║")
	fmt.Println("║   1. Run 'guild chat' to start coordinating agents           ║")
	fmt.Println("║   2. Use 'guild commission create' for new objectives         ║")
	fmt.Println("║   3. View progress with 'guild kanban view'                  ║")
	fmt.Println("║                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func wrapText(text string, width int) []string {
	var lines []string
	words := strings.Fields(text)
	
	var currentLine string
	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Word is longer than width, split it
				lines = append(lines, word[:width])
				currentLine = word[width:]
			}
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}
	
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	return lines
}