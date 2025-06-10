package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorWhite  = "\033[97m"
)

// Color helper functions
func green(s string) string  { return colorGreen + colorBold + s + colorReset }
func red(s string) string    { return colorRed + colorBold + s + colorReset }
func yellow(s string) string { return colorYellow + s + colorReset }
func blue(s string) string   { return colorBlue + colorBold + s + colorReset }
func purple(s string) string { return colorPurple + colorBold + s + colorReset }
func gray(s string) string   { return colorGray + s + colorReset }
func white(s string) string  { return colorWhite + colorBold + s + colorReset }

// Icons
const (
	iconCheck   = "✓"
	iconCross   = "✗"
	iconArrow   = "→"
	iconRocket  = "🚀"
	iconBuild   = "🔨"
	iconTest    = "🧪"
	iconClean   = "🧹"
	iconShield  = "🛡"
	iconGear    = "⚙"
	iconPackage = "📦"
)

// BuildTool handles all build operations with visual feedback
type BuildTool struct {
	verbose bool
	noColor bool
	skipVet bool
	width   int
	mu      sync.Mutex
}

// Task represents a build task
type Task struct {
	Name        string
	Description string
	Icon        string
	Action      func() error
	Progress    int
}

// ProgressBar renders a progress bar
func (bt *BuildTool) ProgressBar(percent int, message string) {
	if bt.noColor {
		fmt.Printf("%d%% %s\n", percent, message)
		return
	}

	width := 40
	filled := percent * width / 100
	empty := width - filled

	// Clear line and return to start
	fmt.Print("\033[2K\r")

	// Draw progress bar
	fmt.Print(gray("["))
	for i := 0; i < filled; i++ {
		fmt.Print(green("█"))
	}
	for i := 0; i < empty; i++ {
		fmt.Print(gray("░"))
	}
	fmt.Print(gray("] "))

	// Show percentage and message
	fmt.Printf("%s %s", white(fmt.Sprintf("%3d%%", percent)), yellow(message))

	if percent == 100 {
		fmt.Println()
	}
}

// Box draws a nice box around content
func (bt *BuildTool) Box(title string, content []string) {
	if bt.noColor {
		fmt.Printf("\n=== %s ===\n", title)
		for _, line := range content {
			fmt.Println(line)
		}
		fmt.Println()
		return
	}

	width := 60
	topLine := "┌" + strings.Repeat("─", width-2) + "┐"
	bottomLine := "└" + strings.Repeat("─", width-2) + "┘"
	divider := "├" + strings.Repeat("─", width-2) + "┤"

	fmt.Println()
	fmt.Println(blue(topLine))
	
	// Title
	titlePadded := fmt.Sprintf(" %s %s ", iconRocket, title)
	titleLen := len(title) + 4 // icon + spaces
	padding := (width - titleLen - 2) / 2
	fmt.Printf("%s%s%s%s%s\n",
		blue("│"),
		strings.Repeat(" ", padding),
		purple(titlePadded),
		strings.Repeat(" ", width-padding-titleLen-2),
		blue("│"),
	)
	
	if len(content) > 0 {
		fmt.Println(blue(divider))
		
		// Content
		for _, line := range content {
			lineLen := len(line)
			if lineLen > width-4 {
				line = line[:width-7] + "..."
				lineLen = width - 4
			}
			fmt.Printf("%s  %s%s%s\n",
				blue("│"),
				line,
				strings.Repeat(" ", width-lineLen-4),
				blue("│"),
			)
		}
	}
	
	fmt.Println(blue(bottomLine))
}

// StatusCard shows a status card with pass/fail
func (bt *BuildTool) StatusCard(title string, status bool) {
	if bt.noColor {
		if status {
			fmt.Printf("[PASS] %s\n", title)
		} else {
			fmt.Printf("[FAIL] %s\n", title)
		}
		return
	}

	icon := iconCheck
	colorFunc := green
	if !status {
		icon = iconCross
		colorFunc = red
	}

	bt.Box("", []string{
		colorFunc(fmt.Sprintf("%s %s", icon, title)),
	})
}

// RunCommand executes a command with progress feedback
func (bt *BuildTool) RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	
	if bt.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Capture output for error reporting
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Show error output
		if stderr.Len() > 0 {
			fmt.Println(red("\nError output:"))
			fmt.Print(stderr.String())
		}
	}
	
	return err
}

// Build handles the build process with visual feedback
func (bt *BuildTool) Build() error {
	bt.Box("Guild Framework Build", []string{
		"Building the future of AI agent orchestration",
		"Version: dev",
	})

	tasks := []Task{
		{
			Name:        "Clean",
			Description: "Cleaning build artifacts",
			Icon:        iconClean,
			Action: func() error {
				return os.RemoveAll("bin")
			},
		},
		{
			Name:        "Dependencies",
			Description: "Checking dependencies",
			Icon:        iconPackage,
			Action: func() error {
				return bt.RunCommand("go", "mod", "download")
			},
		},
		{
			Name:        "Generate",
			Description: "Running code generation",
			Icon:        iconGear,
			Action: func() error {
				return bt.RunCommand("go", "generate", "./...")
			},
		},
		{
			Name:        "Vet",
			Description: "Running go vet",
			Icon:        iconShield,
			Action: func() error {
				if bt.skipVet {
					// Skip vet check
					return nil
				}
				return bt.RunCommand("go", "vet", "./...")
			},
		},
		{
			Name:        "Build",
			Description: "Building guild binary",
			Icon:        iconBuild,
			Action: func() error {
				os.MkdirAll("bin", 0755)
				return bt.RunCommand("go", "build", "-o", "bin/guild", "./cmd/guild")
			},
		},
	}

	// Execute tasks with progress
	for i, task := range tasks {
		progress := (i * 100) / len(tasks)
		bt.ProgressBar(progress, fmt.Sprintf("%s %s", task.Icon, task.Description))
		
		err := task.Action()
		if err != nil {
			bt.ProgressBar(progress, fmt.Sprintf("%s %s - FAILED", iconCross, task.Description))
			fmt.Println()
			return fmt.Errorf("%s failed: %w", task.Name, err)
		}
		
		time.Sleep(100 * time.Millisecond) // Visual feedback
	}

	bt.ProgressBar(100, fmt.Sprintf("%s Build complete!", iconCheck))
	fmt.Println()

	// Final status
	bt.StatusCard("Build Completed Successfully", true)
	
	// Show binary info
	if info, err := os.Stat("bin/guild"); err == nil {
		bt.Box("Build Output", []string{
			fmt.Sprintf("Binary: bin/guild"),
			fmt.Sprintf("Size: %.2f MB", float64(info.Size())/1024/1024),
			fmt.Sprintf("Mode: %s", info.Mode()),
		})
	}

	return nil
}

// Test runs tests with visual dashboard
func (bt *BuildTool) Test() error {
	bt.Box("Guild Framework Test Suite", []string{
		"Running comprehensive test coverage",
		"This may take a few minutes...",
	})

	// Discover packages
	fmt.Println()
	fmt.Println(yellow("Discovering packages..."))
	
	cmd := exec.Command("go", "list", "./...")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	packages := []string{}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		pkg := scanner.Text()
		// Skip vendor and integration tests for unit tests
		if !strings.Contains(pkg, "/vendor/") && 
		   !strings.Contains(pkg, "/integration/") &&
		   !strings.Contains(pkg, "/examples/") {
			packages = append(packages, pkg)
		}
	}

	fmt.Printf("Found %d packages to test\n\n", len(packages))

	// Test each package with progress
	passed := 0
	failed := 0
	
	for i, pkg := range packages {
		progress := ((i + 1) * 100) / len(packages)
		shortPkg := strings.TrimPrefix(pkg, "github.com/guild-ventures/guild-core/")
		
		bt.ProgressBar(progress, fmt.Sprintf("Testing %s", shortPkg))
		
		// Run test without creating .test files
		cmd := exec.Command("go", "test", "-short", "-count=1", pkg)
		err := cmd.Run()
		
		if err != nil {
			failed++
			bt.ProgressBar(progress, fmt.Sprintf("%s %s - FAILED", iconCross, shortPkg))
			fmt.Println()
		} else {
			passed++
		}
		
		time.Sleep(50 * time.Millisecond) // Visual feedback
	}

	fmt.Println()

	// Summary
	total := passed + failed
	success := failed == 0

	summaryLines := []string{
		fmt.Sprintf("Total Packages: %d", total),
		green(fmt.Sprintf("Passed: %d", passed)),
	}
	
	if failed > 0 {
		summaryLines = append(summaryLines, red(fmt.Sprintf("Failed: %d", failed)))
	}
	
	coverage := float64(passed) / float64(total) * 100
	summaryLines = append(summaryLines, fmt.Sprintf("Coverage: %.1f%%", coverage))

	bt.Box("Test Results Summary", summaryLines)
	bt.StatusCard("All Tests Passed", success)

	if !success {
		return fmt.Errorf("%d tests failed", failed)
	}

	return nil
}

// Integration runs integration tests
func (bt *BuildTool) Integration() error {
	bt.Box("Integration Test Suite", []string{
		"Running end-to-end integration tests",
		"Validating component interactions",
	})

	// Run integration tests specifically
	fmt.Println()
	bt.ProgressBar(0, "Setting up test environment...")
	time.Sleep(500 * time.Millisecond)

	bt.ProgressBar(30, "Running integration tests...")
	
	cmd := exec.Command("go", "test", "-tags=integration", "./integration/...", "-v")
	
	if bt.verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	
	err := cmd.Run()
	
	if err != nil {
		bt.ProgressBar(30, "Integration tests - FAILED")
		fmt.Println()
		bt.StatusCard("Integration Tests Failed", false)
		return err
	}

	bt.ProgressBar(100, "Integration tests complete!")
	fmt.Println()
	bt.StatusCard("Integration Tests Passed", true)
	
	return nil
}

// Clean removes all build artifacts
func (bt *BuildTool) Clean() error {
	bt.Box("Cleaning Build Artifacts", []string{
		"Removing generated files and binaries",
	})

	items := []struct {
		path string
		desc string
	}{
		{"bin/", "Binary directory"},
		{"*.test", "Test binaries"},
		{".test-*", "Test artifacts"},
		{"coverage.out", "Coverage reports"},
		{".cache/", "Build cache"},
	}

	for i, item := range items {
		progress := ((i + 1) * 100) / len(items)
		bt.ProgressBar(progress, fmt.Sprintf("Removing %s", item.desc))
		
		// Use filepath.Glob for wildcards
		if strings.Contains(item.path, "*") {
			matches, _ := filepath.Glob(item.path)
			for _, match := range matches {
				os.RemoveAll(match)
			}
		} else {
			os.RemoveAll(item.path)
		}
		
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println()
	bt.StatusCard("Cleanup Complete", true)
	
	return nil
}

func main() {
	var (
		verbose = flag.Bool("v", false, "Verbose output")
		noColor = flag.Bool("no-color", false, "Disable colored output")
		skipVet = flag.Bool("skip-vet", false, "Skip go vet check")
		help    = flag.Bool("h", false, "Show help")
	)

	flag.Parse()

	if *help || flag.NArg() == 0 {
		showHelp()
		return
	}

	// Check if in CI environment
	if os.Getenv("CI") == "true" {
		*noColor = true
	}

	bt := &BuildTool{
		verbose: *verbose,
		noColor: *noColor,
		skipVet: *skipVet,
	}

	var err error
	
	switch flag.Arg(0) {
	case "build":
		err = bt.Build()
	case "test":
		err = bt.Test()
	case "integration":
		err = bt.Integration()
	case "clean":
		err = bt.Clean()
	case "all":
		// Run everything
		if err = bt.Clean(); err == nil {
			if err = bt.Build(); err == nil {
				if err = bt.Test(); err == nil {
					err = bt.Integration()
				}
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", flag.Arg(0))
		showHelp()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s Error: %v\n", red(iconCross), err)
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println("Guild Build Tool")
	fmt.Println()
	fmt.Println("Usage: go run tools/buildtool/main.go [flags] <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build        Build the Guild CLI")
	fmt.Println("  test         Run unit tests")
	fmt.Println("  integration  Run integration tests")
	fmt.Println("  clean        Clean build artifacts")
	fmt.Println("  all          Run clean, build, test, and integration")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -v           Verbose output")
	fmt.Println("  -no-color    Disable colored output")
	fmt.Println("  -h           Show this help")
}