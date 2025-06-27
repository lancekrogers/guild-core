// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/paths"
	"github.com/lancekrogers/guild/pkg/project"
)

// DemoValidator performs pre-flight checks for demo environment
type DemoValidator struct {
	errors   []string
	warnings []string
	verbose  bool
}

// DemoValidationResult represents the overall validation status
type DemoValidationResult struct {
	Passed   bool
	Errors   []string
	Warnings []string
	Duration time.Duration
}

// NewDemoValidator creates a new demo validator
func NewDemoValidator(verbose bool) *DemoValidator {
	return &DemoValidator{
		errors:   make([]string, 0),
		warnings: make([]string, 0),
		verbose:  verbose,
	}
}

// ValidateEnvironment performs comprehensive demo environment validation
func (dv *DemoValidator) ValidateEnvironment() (*DemoValidationResult, error) {
	start := time.Now()

	fmt.Printf("🏰 Guild Demo Environment Validator\n")
	fmt.Printf("═══════════════════════════════════════════\n\n")

	// Core checks
	dv.checkTerminalEnvironment()
	dv.checkGuildProject()
	dv.checkNetworkPorts()
	dv.checkDemoFiles()
	dv.checkVisualSupport()
	dv.checkRecordingTools()

	duration := time.Since(start)

	// Print results
	dv.printValidationSummary(duration)

	result := &DemoValidationResult{
		Passed:   len(dv.errors) == 0,
		Errors:   dv.errors,
		Warnings: dv.warnings,
		Duration: duration,
	}

	if len(dv.errors) > 0 {
		return result, gerror.New(gerror.ErrCodeValidation, "demo environment validation failed", nil).
			WithComponent("cli").
			WithOperation("demo.validate").
			WithDetails("error_count", strconv.Itoa(len(dv.errors)))
	}

	return result, nil
}

// checkTerminalEnvironment validates terminal size and capabilities
func (dv *DemoValidator) checkTerminalEnvironment() {
	fmt.Printf("📐 Checking terminal environment...\n")

	// Check if running in terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		dv.warnings = append(dv.warnings, "Not running in interactive terminal - some features may be limited")
	}

	// Check terminal size
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		dv.warnings = append(dv.warnings, "Could not determine terminal size")
	} else {
		if dv.verbose {
			fmt.Printf("  Terminal size: %dx%d\n", width, height)
		}

		if width < 120 {
			dv.warnings = append(dv.warnings, fmt.Sprintf("Terminal width (%d) smaller than recommended (120)", width))
		}
		if height < 40 {
			dv.warnings = append(dv.warnings, fmt.Sprintf("Terminal height (%d) smaller than recommended (40)", height))
		}
	}

	// Check color support
	colorterm := os.Getenv("COLORTERM")
	term_program := os.Getenv("TERM_PROGRAM")

	if colorterm == "truecolor" || term_program == "iTerm.app" {
		if dv.verbose {
			fmt.Printf("  ✅ Full color support detected\n")
		}
	} else {
		dv.warnings = append(dv.warnings, "Terminal may not support full colors - set COLORTERM=truecolor")
	}

	fmt.Printf("  ✅ Terminal environment checked\n\n")
}

// checkGuildProject validates Guild project setup
func (dv *DemoValidator) checkGuildProject() {
	fmt.Printf("🏗️  Checking Guild project setup...\n")

	// Check if in Guild project
	projCtx, err := project.GetContext()
	if err != nil {
		dv.errors = append(dv.errors, "Not in a Guild project - run 'guild init' first")
		return
	}

	if dv.verbose {
		fmt.Printf("  Project root: %s\n", projCtx.GetRootPath())
	}

	// Check .guild directory
	guildDir := filepath.Join(projCtx.GetRootPath(), paths.DefaultCampaignDir)
	if !dv.fileExists(guildDir) {
		dv.errors = append(dv.errors, "Guild not initialized - run 'guild init'")
		return
	}

	// Check database
	dbPath := filepath.Join(guildDir, "memory.db")
	if !dv.fileExists(dbPath) {
		dv.warnings = append(dv.warnings, "Database not found - may need to run some commands first")
	}

	// Check guild.yaml
	configPath := filepath.Join(guildDir, "guild.yaml")
	if !dv.fileExists(configPath) {
		dv.errors = append(dv.errors, "Guild configuration not found - check guild.yaml")
	}

	fmt.Printf("  ✅ Guild project validated\n\n")
}

// checkNetworkPorts validates that required ports are available
func (dv *DemoValidator) checkNetworkPorts() {
	fmt.Printf("🌐 Checking network ports...\n")

	ports := []int{50051, 50052} // gRPC ports

	for _, port := range ports {
		if dv.isPortInUse(port) {
			if port == 50051 {
				dv.errors = append(dv.errors, fmt.Sprintf("Port %d (primary gRPC) already in use", port))
			} else {
				dv.warnings = append(dv.warnings, fmt.Sprintf("Port %d (backup gRPC) already in use", port))
			}
		} else if dv.verbose {
			fmt.Printf("  ✅ Port %d available\n", port)
		}
	}

	fmt.Printf("  ✅ Network ports checked\n\n")
}

// checkDemoFiles validates that required demo files exist
func (dv *DemoValidator) checkDemoFiles() {
	fmt.Printf("📁 Checking demo files...\n")

	// Get project context for relative paths
	projCtx, err := project.GetContext()
	if err != nil {
		dv.errors = append(dv.errors, "Cannot check demo files - project context unavailable")
		return
	}

	rootPath := projCtx.GetRootPath()

	requiredFiles := []struct {
		path        string
		description string
		required    bool
	}{
		{"examples/commissions/task-management-api.md", "Example commission file", true},
		{"examples/config/enhanced_guild.yaml", "Enhanced guild configuration", false},
		{"docs/demo-chat.md", "Demo documentation", false},
		{".guild/guild.yaml", "Guild configuration", true},
	}

	for _, file := range requiredFiles {
		fullPath := filepath.Join(rootPath, file.path)
		if !dv.fileExists(fullPath) {
			if file.required {
				dv.errors = append(dv.errors, fmt.Sprintf("Missing required file: %s", file.path))
			} else {
				dv.warnings = append(dv.warnings, fmt.Sprintf("Missing optional file: %s (%s)", file.path, file.description))
			}
		} else if dv.verbose {
			fmt.Printf("  ✅ Found: %s\n", file.path)
		}
	}

	fmt.Printf("  ✅ Demo files checked\n\n")
}

// checkVisualSupport validates visual components and rendering
func (dv *DemoValidator) checkVisualSupport() {
	fmt.Printf("🎨 Checking visual support...\n")

	// Check TERM variable
	termVar := os.Getenv("TERM")
	if termVar == "" {
		dv.warnings = append(dv.warnings, "TERM environment variable not set")
	} else if dv.verbose {
		fmt.Printf("  TERM: %s\n", termVar)
	}

	// Check for truecolor support
	if dv.supportsColor() {
		if dv.verbose {
			fmt.Printf("  ✅ Color support available\n")
		}
	} else {
		dv.warnings = append(dv.warnings, "Limited color support - visual features may be degraded")
	}

	// Check terminal capabilities
	if dv.hasUnicodeSupport() {
		if dv.verbose {
			fmt.Printf("  ✅ Unicode support available\n")
		}
	} else {
		dv.warnings = append(dv.warnings, "Limited Unicode support - visual elements may not display correctly")
	}

	fmt.Printf("  ✅ Visual support checked\n\n")
}

// checkRecordingTools validates demo recording capabilities
func (dv *DemoValidator) checkRecordingTools() {
	fmt.Printf("🎥 Checking recording tools...\n")

	tools := []struct {
		name        string
		command     string
		description string
		required    bool
	}{
		{"asciinema", "asciinema", "Terminal recording", false},
		{"agg", "agg", "GIF conversion", false},
		{"ffmpeg", "ffmpeg", "Video processing", false},
	}

	for _, tool := range tools {
		if dv.commandExists(tool.command) {
			if dv.verbose {
				fmt.Printf("  ✅ %s available\n", tool.name)
			}
		} else {
			if tool.required {
				dv.errors = append(dv.errors, fmt.Sprintf("Required recording tool missing: %s", tool.name))
			} else {
				dv.warnings = append(dv.warnings, fmt.Sprintf("Optional recording tool missing: %s (%s)", tool.name, tool.description))
			}
		}
	}

	fmt.Printf("  ✅ Recording tools checked\n\n")
}

// Helper methods

func (dv *DemoValidator) fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (dv *DemoValidator) isPortInUse(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return true // Port is in use
	}
	ln.Close()
	return false
}

func (dv *DemoValidator) commandExists(command string) bool {
	_, err := os.Stat(fmt.Sprintf("/usr/bin/%s", command))
	if err == nil {
		return true
	}
	_, err = os.Stat(fmt.Sprintf("/usr/local/bin/%s", command))
	if err == nil {
		return true
	}
	_, err = os.Stat(fmt.Sprintf("/opt/homebrew/bin/%s", command))
	if err == nil {
		return true
	}
	return false
}

func (dv *DemoValidator) supportsColor() bool {
	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		return true
	}

	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram == "iTerm.app" || termProgram == "vscode" {
		return true
	}

	term := os.Getenv("TERM")
	return strings.Contains(term, "256color") || strings.Contains(term, "color")
}

func (dv *DemoValidator) hasUnicodeSupport() bool {
	lang := os.Getenv("LANG")
	lc_all := os.Getenv("LC_ALL")

	return strings.Contains(lang, "UTF-8") || strings.Contains(lc_all, "UTF-8")
}

func (dv *DemoValidator) printValidationSummary(duration time.Duration) {
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("📊 Validation Summary\n")
	fmt.Printf("═══════════════════════════════════════════\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Errors: %d\n", len(dv.errors))
	fmt.Printf("Warnings: %d\n", len(dv.warnings))
	fmt.Printf("\n")

	if len(dv.errors) > 0 {
		fmt.Printf("❌ Errors (must fix before demo):\n")
		for i, err := range dv.errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
		fmt.Printf("\n")
	}

	if len(dv.warnings) > 0 {
		fmt.Printf("⚠️  Warnings (recommend fixing):\n")
		for i, warn := range dv.warnings {
			fmt.Printf("  %d. %s\n", i+1, warn)
		}
		fmt.Printf("\n")
	}

	if len(dv.errors) == 0 {
		fmt.Printf("✅ Demo environment ready!\n")
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("1. Run demo scenarios: ./guild demo run\n")
		fmt.Printf("2. Start recording: ./scripts/demo-recording/record-demo.sh\n")
		fmt.Printf("3. Test chat interface: ./guild chat\n")
	} else {
		fmt.Printf("❌ Demo environment not ready\n")
		fmt.Printf("\nFix the errors above before proceeding with demo\n")
	}

	fmt.Printf("═══════════════════════════════════════════\n")
}

// Validation helper functions

// CheckAPIKeys validates that required API keys are available
func (dv *DemoValidator) CheckAPIKeys() {
	fmt.Printf("🔑 Checking API keys...\n")

	keys := []struct {
		env      string
		provider string
		required bool
	}{
		{"OPENAI_API_KEY", "OpenAI", false},
		{"ANTHROPIC_API_KEY", "Anthropic", false},
		{"DEEPSEEK_API_KEY", "DeepSeek", false},
	}

	hasAnyKey := false
	for _, key := range keys {
		if os.Getenv(key.env) != "" {
			hasAnyKey = true
			if dv.verbose {
				fmt.Printf("  ✅ %s configured\n", key.provider)
			}
		} else {
			if key.required {
				dv.errors = append(dv.errors, fmt.Sprintf("Required API key missing: %s", key.env))
			} else if dv.verbose {
				fmt.Printf("  ⚪ %s not configured\n", key.provider)
			}
		}
	}

	if !hasAnyKey {
		dv.warnings = append(dv.warnings, "No API keys configured - using mock provider only")
	}

	fmt.Printf("  ✅ API keys checked\n\n")
}

// CheckPerformance runs basic performance checks
func (dv *DemoValidator) CheckPerformance() {
	fmt.Printf("⚡ Checking performance...\n")

	// Test file I/O speed
	start := time.Now()
	tmpFile := "/tmp/guild_perf_test.txt"

	err := os.WriteFile(tmpFile, []byte("performance test"), 0644)
	if err == nil {
		_, err = os.ReadFile(tmpFile)
		os.Remove(tmpFile)
	}

	ioTime := time.Since(start)
	if ioTime > 10*time.Millisecond {
		dv.warnings = append(dv.warnings, fmt.Sprintf("Slow file I/O detected (%v) - may affect demo smoothness", ioTime))
	} else if dv.verbose {
		fmt.Printf("  ✅ File I/O performance: %v\n", ioTime)
	}

	fmt.Printf("  ✅ Performance checked\n\n")
}

// demoCheckCmd provides the demo-check command
var demoCheckCmd = &cobra.Command{
	Use:   "demo-check",
	Short: "Validate environment for Guild demo",
	Long: `Validate that the environment is properly configured for running Guild demos.

This command checks:
- Terminal size and color support
- Guild project initialization
- Network port availability
- Required demo files
- Visual rendering capabilities
- Recording tool availability

Use --verbose for detailed output.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		apiKeys, _ := cmd.Flags().GetBool("api-keys")
		performance, _ := cmd.Flags().GetBool("performance")

		validator := NewDemoValidator(verbose)

		// Run additional checks if requested
		if apiKeys {
			validator.CheckAPIKeys()
		}

		if performance {
			validator.CheckPerformance()
		}

		result, err := validator.ValidateEnvironment()
		if err != nil {
			return err
		}

		// Set exit code based on validation result
		if !result.Passed {
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	// Add demo-check command to root
	rootCmd.AddCommand(demoCheckCmd)

	// Add flags
	demoCheckCmd.Flags().BoolP("verbose", "v", false, "Show detailed validation output")
	demoCheckCmd.Flags().Bool("api-keys", false, "Check API key configuration")
	demoCheckCmd.Flags().Bool("performance", false, "Run performance checks")
}
