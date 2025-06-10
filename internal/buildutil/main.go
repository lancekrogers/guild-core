package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Println("Usage: go run tools/buildtool/simple.go [build|test|clean]")
		os.Exit(1)
	}

	switch flag.Arg(0) {
	case "build":
		build(*verbose)
	case "test":
		test(*verbose)
	case "clean":
		clean(*verbose)
	default:
		fmt.Printf("Unknown command: %s\n", flag.Arg(0))
		os.Exit(1)
	}
}

func build(verbose bool) {
	fmt.Println("\n🔨 Building Guild...")
	
	// Clean first
	fmt.Println("  → Cleaning old artifacts...")
	os.RemoveAll("bin")
	
	// Create bin directory
	os.MkdirAll("bin", 0755)
	
	// Run go generate
	fmt.Println("  → Running code generation...")
	runCmd("go", "generate", "./...")
	
	// Build
	fmt.Println("  → Compiling binary...")
	if err := runCmd("go", "build", "-o", "bin/guild", "./cmd/guild"); err != nil {
		fmt.Println("  ❌ Build failed!")
		os.Exit(1)
	}
	
	fmt.Println("  ✅ Build complete: bin/guild")
	fmt.Println()
}

func test(verbose bool) {
	fmt.Println("\n🧪 Running tests...")
	
	packages := []string{
		"./pkg/agent/...",
		"./pkg/memory/...",
		"./pkg/orchestrator/...",
		"./pkg/providers/...",
	}
	
	failed := 0
	for _, pkg := range packages {
		shortName := strings.TrimPrefix(pkg, "./")
		fmt.Printf("  → Testing %s...", shortName)
		
		cmd := exec.Command("go", "test", "-short", pkg)
		if !verbose {
			cmd.Stdout = nil
			cmd.Stderr = nil
		}
		
		if err := cmd.Run(); err != nil {
			fmt.Println(" ❌")
			failed++
		} else {
			fmt.Println(" ✅")
		}
	}
	
	if failed > 0 {
		fmt.Printf("\n  ❌ %d packages failed\n", failed)
		os.Exit(1)
	} else {
		fmt.Println("\n  ✅ All tests passed!")
	}
	fmt.Println()
}

func clean(verbose bool) {
	fmt.Println("\n🧹 Cleaning...")
	
	items := []string{"bin/", "*.test", "coverage.out", ".test-*"}
	
	for _, item := range items {
		fmt.Printf("  → Removing %s...\n", item)
		if strings.Contains(item, "*") {
			// Use shell for wildcards
			exec.Command("sh", "-c", "rm -rf "+item).Run()
		} else {
			os.RemoveAll(item)
		}
	}
	
	fmt.Println("  ✅ Clean complete!")
	fmt.Println()
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}