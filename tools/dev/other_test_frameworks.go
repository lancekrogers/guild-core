package dev

import (
	"fmt"
	"strings"
)

// JavaTestFramework implements JUnit testing framework (stub)
type JavaTestFramework struct{}

func (j *JavaTestFramework) Name() string {
	return "junit"
}

func (j *JavaTestFramework) Detect(path string) bool {
	return fileExists("pom.xml") || 
		   fileExists("build.gradle") || 
		   fileExists("build.gradle.kts") ||
		   len(findTestFiles(path, "*Test.java")) > 0
}

func (j *JavaTestFramework) BuildCommand(input TestRunnerInput) ([]string, error) {
	// Maven
	if fileExists("pom.xml") {
		cmd := []string{"mvn", "test"}
		if input.Verbose {
			cmd = append(cmd, "-X")
		}
		return cmd, nil
	}
	
	// Gradle
	if fileExists("build.gradle") || fileExists("build.gradle.kts") {
		cmd := []string{"./gradlew", "test"}
		if input.Verbose {
			cmd = append(cmd, "--info")
		}
		return cmd, nil
	}
	
	return nil, fmt.Errorf("no supported Java build tool found")
}

func (j *JavaTestFramework) ParseOutput(output string, exitCode int) (*TestResult, error) {
	// Basic parsing - could be enhanced with proper JUnit XML parsing
	lines := strings.Split(output, "\n")
	
	summary := TestSummary{
		Success: exitCode == 0,
	}
	
	// Count test results from Maven/Gradle output
	for _, line := range lines {
		if strings.Contains(line, "Tests run:") {
			// Maven format: Tests run: 5, Failures: 0, Errors: 0, Skipped: 0
			// Extract numbers...
		}
	}
	
	return &TestResult{
		Summary: summary,
		Output:  output,
		Metadata: map[string]string{
			"java_framework": "junit",
		},
	}, nil
}

func (j *JavaTestFramework) SupportsCoverage() bool {
	return true // Via JaCoCo
}

func (j *JavaTestFramework) SupportsParallel() bool {
	return true
}

// RubyTestFramework implements RSpec testing framework (stub)
type RubyTestFramework struct{}

func (r *RubyTestFramework) Name() string {
	return "rspec"
}

func (r *RubyTestFramework) Detect(path string) bool {
	return fileExists("Gemfile") ||
		   fileExists(".rspec") ||
		   len(findTestFiles(path, "*_spec.rb")) > 0
}

func (r *RubyTestFramework) BuildCommand(input TestRunnerInput) ([]string, error) {
	cmd := []string{"rspec"}
	
	if input.Verbose {
		cmd = append(cmd, "--format", "documentation")
	}
	
	if input.Pattern != "" {
		cmd = append(cmd, "--pattern", input.Pattern)
	}
	
	if input.Path != "" {
		cmd = append(cmd, input.Path)
	}
	
	return cmd, nil
}

func (r *RubyTestFramework) ParseOutput(output string, exitCode int) (*TestResult, error) {
	summary := TestSummary{
		Success: exitCode == 0,
	}
	
	return &TestResult{
		Summary: summary,
		Output:  output,
		Metadata: map[string]string{
			"ruby_framework": "rspec",
		},
	}, nil
}

func (r *RubyTestFramework) SupportsCoverage() bool {
	return true // Via SimpleCov
}

func (r *RubyTestFramework) SupportsParallel() bool {
	return true
}

// RustTestFramework implements Cargo test framework (stub)
type RustTestFramework struct{}

func (rt *RustTestFramework) Name() string {
	return "cargo"
}

func (rt *RustTestFramework) Detect(path string) bool {
	return fileExists("Cargo.toml")
}

func (rt *RustTestFramework) BuildCommand(input TestRunnerInput) ([]string, error) {
	cmd := []string{"cargo", "test"}
	
	if input.Verbose {
		cmd = append(cmd, "--verbose")
	}
	
	if input.Pattern != "" {
		cmd = append(cmd, input.Pattern)
	}
	
	return cmd, nil
}

func (rt *RustTestFramework) ParseOutput(output string, exitCode int) (*TestResult, error) {
	summary := TestSummary{
		Success: exitCode == 0,
	}
	
	return &TestResult{
		Summary: summary,
		Output:  output,
		Metadata: map[string]string{
			"rust_framework": "cargo",
		},
	}, nil
}

func (rt *RustTestFramework) SupportsCoverage() bool {
	return true // Via tarpaulin
}

func (rt *RustTestFramework) SupportsParallel() bool {
	return true
}