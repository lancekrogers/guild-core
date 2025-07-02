// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// cleanup-sprint-refs removes all sprint references from guild-core codebase
//
// This command implements the final task of launch coordination, cleaning up
// all internal sprint planning references from the codebase to prepare for
// production release.
//
// The tool performs intelligent cleanup of:
//   - Sprint numbering in comments and documentation
//   - development phase references
//   - Sprint task implementation mentions
//   - Development planning terminology
//
// Usage:
//
//	# Clean all sprint references from codebase
//	cleanup-sprint-refs
//
//	# Dry run to see what would be changed
//	cleanup-sprint-refs --dry-run
//
//	# Clean specific file patterns only
//	cleanup-sprint-refs --include="*.go,*.md"
//
//	# Exclude specific directories
//	cleanup-sprint-refs --exclude="vendor/,testdata/"
package main

import (
	"context"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"go.uber.org/zap"
)

// SprintCleanup removes all sprint references from guild-core codebase
type SprintCleanup struct {
	logger       *zap.Logger
	fileCount    int
	changeCount  int
	patterns     []*CleanupPattern
	excludeDirs  []string
	includeFiles []string
	dryRun       bool
	verbose      bool
	mu           sync.Mutex
}

// CleanupPattern defines a pattern and its replacement strategy
type CleanupPattern struct {
	Pattern     *regexp.Regexp
	Description string
	Replacer    func(string) string
}

// CleanupResult tracks the results of cleanup operations
type CleanupResult struct {
	FilesProcessed int            `json:"files_processed"`
	FilesChanged   int            `json:"files_changed"`
	TotalChanges   int            `json:"total_changes"`
	PatternMatches map[string]int `json:"pattern_matches"`
	ChangedFiles   []string       `json:"changed_files"`
	Errors         []string       `json:"errors"`
	Duration       time.Duration  `json:"duration"`
}

func main() {
	var (
		dryRun    = flag.Bool("dry-run", false, "Show what would be changed without making changes")
		verbose   = flag.Bool("verbose", false, "Verbose logging")
		include   = flag.String("include", "", "Comma-separated file patterns to include (e.g., '*.go,*.md')")
		exclude   = flag.String("exclude", "", "Comma-separated directory patterns to exclude (e.g., 'vendor/,testdata/')")
		targetDir = flag.String("dir", ".", "Target directory to clean (default: current directory)")
	)
	flag.Parse()

	// Initialize logger
	logger, err := initializeLogger(*verbose)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	ctx := context.Background()
	logger.Info("Starting sprint reference cleanup",
		zap.String("target_dir", *targetDir),
		zap.Bool("dry_run", *dryRun),
		zap.String("include", *include),
		zap.String("exclude", *exclude))

	// Initialize cleanup tool
	cleanup := NewSprintCleanup(logger, *dryRun, *verbose)

	// Configure include/exclude patterns
	if *include != "" {
		cleanup.includeFiles = strings.Split(*include, ",")
	}
	if *exclude != "" {
		cleanup.excludeDirs = strings.Split(*exclude, ",")
	}

	// Run cleanup
	result, err := cleanup.CleanupCodebase(ctx, *targetDir)
	if err != nil {
		logger.Fatal("Sprint cleanup failed", zap.Error(err))
	}

	// Print results
	cleanup.PrintSummary(result)

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}

// NewSprintCleanup creates a new sprint cleanup instance
func NewSprintCleanup(logger *zap.Logger, dryRun, verbose bool) *SprintCleanup {
	patterns := []*CleanupPattern{
		{
			Pattern:     regexp.MustCompile(`(?i)sprint\s*[0-9]+\.?[0-9]*`),
			Description: "Sprint numbers (e.g., 'performance optimization', 'performance optimization')",
			Replacer: func(match string) string {
				lower := strings.ToLower(match)
				switch {
				case strings.Contains(lower, "performance optimization"):
					return "performance optimization"
				case strings.Contains(lower, "production enhancement"):
					return "production enhancement"
				case strings.Contains(lower, "performance optimization"):
					return "launch preparation"
				default:
					return "development phase"
				}
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)pv[-_\s]*sprint[0-9]*\.?[0-9]*`),
			Description: "development iteration references (e.g., 'development iteration', 'pv_development phase')",
			Replacer: func(match string) string {
				return "development iteration"
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)product\s*vision\s*sprint[0-9]*\.?[0-9]*`),
			Description: "development phase references",
			Replacer: func(match string) string {
				return "development phase"
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)this\s+implements.*sprint.*[0-9]+.*task`),
			Description: "Implementation task references",
			Replacer: func(match string) string {
				return "This implements the required functionality"
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)feature requirements`),
			Description: "Sprint requirements references",
			Replacer: func(match string) string {
				return "feature requirements"
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)identified in the requirements]+`),
			Description: "Sprint identification references",
			Replacer: func(match string) string {
				return "identified in the requirements"
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)from the development requirements`),
			Description: "Sprint task source references",
			Replacer: func(match string) string {
				return "from the development requirements"
			},
		},
		{
			Pattern:     regexp.MustCompile(`(?i)agent\s*[0-9]+.*sprint.*[0-9]+`),
			Description: "Agent sprint references",
			Replacer: func(match string) string {
				lower := strings.ToLower(match)
				switch {
				case strings.Contains(lower, "agent 1"):
					return "UI polish component"
				case strings.Contains(lower, "agent 2"):
					return "integration architecture component"
				case strings.Contains(lower, "agent 3"):
					return "performance validation component"
				case strings.Contains(lower, "agent 4"):
					return "launch coordination component"
				default:
					return "development component"
				}
			},
		},
	}

	return &SprintCleanup{
		logger:       logger.Named("sprint-cleanup"),
		patterns:     patterns,
		excludeDirs:  []string{"vendor/", ".git/", "build/", "dist/", "node_modules/"},
		includeFiles: []string{"*.go", "*.md", "*.yaml", "*.yml", "*.json", "*.sql", "*.toml"},
		dryRun:       dryRun,
		verbose:      verbose,
	}
}

// CleanupCodebase processes all files in the target directory
func (sc *SprintCleanup) CleanupCodebase(ctx context.Context, targetDir string) (*CleanupResult, error) {
	startTime := time.Now()
	result := &CleanupResult{
		PatternMatches: make(map[string]int),
		ChangedFiles:   make([]string, 0),
		Errors:         make([]string, 0),
	}

	sc.logger.Info("Starting codebase cleanup", zap.String("target_dir", targetDir))

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			sc.addError(result, fmt.Sprintf("Error accessing %s: %v", path, err))
			return nil // Continue processing other files
		}

		// Skip directories
		if info.IsDir() {
			// Check if this directory should be excluded
			relPath, _ := filepath.Rel(targetDir, path)
			for _, excludeDir := range sc.excludeDirs {
				if strings.HasPrefix(relPath+"/", excludeDir) || relPath == strings.TrimSuffix(excludeDir, "/") {
					sc.logger.Debug("Skipping excluded directory", zap.String("path", path))
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file should be processed
		if !sc.shouldProcessFile(path) {
			return nil
		}

		// Process the file
		if err := sc.processFile(ctx, path, result); err != nil {
			sc.addError(result, fmt.Sprintf("Error processing %s: %v", path, err))
		}

		result.FilesProcessed++
		return nil
	})

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to walk directory tree")
	}

	result.Duration = time.Since(startTime)

	sc.logger.Info("Codebase cleanup completed",
		zap.Int("files_processed", result.FilesProcessed),
		zap.Int("files_changed", result.FilesChanged),
		zap.Int("total_changes", result.TotalChanges),
		zap.Duration("duration", result.Duration))

	return result, nil
}

// shouldProcessFile determines if a file should be processed
func (sc *SprintCleanup) shouldProcessFile(path string) bool {
	// Check include patterns
	if len(sc.includeFiles) > 0 {
		for _, pattern := range sc.includeFiles {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				return true
			}
		}
		return false
	} else {
		// Default include patterns
		ext := strings.ToLower(filepath.Ext(path))
		validExts := []string{".go", ".md", ".yaml", ".yml", ".json", ".sql", ".toml", ".txt"}
		for _, validExt := range validExts {
			if ext == validExt {
				return true
			}
		}
		return false
	}
}

// processFile processes a single file for sprint reference cleanup
func (sc *SprintCleanup) processFile(ctx context.Context, filePath string, result *CleanupResult) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to read file")
	}

	originalContent := string(content)
	modifiedContent := originalContent
	fileChanged := false
	changesMade := 0

	// Apply cleanup patterns
	for _, pattern := range sc.patterns {
		matches := pattern.Pattern.FindAllString(modifiedContent, -1)
		if len(matches) > 0 {
			result.PatternMatches[pattern.Description] += len(matches)

			if sc.verbose {
				sc.logger.Debug("Found pattern matches",
					zap.String("file", filePath),
					zap.String("pattern", pattern.Description),
					zap.Int("matches", len(matches)))
			}

			// Apply replacements
			newContent := pattern.Pattern.ReplaceAllStringFunc(modifiedContent, pattern.Replacer)
			if newContent != modifiedContent {
				modifiedContent = newContent
				fileChanged = true
				changesMade += len(matches)
			}
		}
	}

	// Additional cleanup for specific Go patterns
	if strings.HasSuffix(filePath, ".go") {
		modifiedContent = sc.cleanupGoSpecificPatterns(modifiedContent)
		if modifiedContent != originalContent && !fileChanged {
			fileChanged = true
			changesMade++
		}
	}

	// Write back if changes were made and not dry run
	if fileChanged && modifiedContent != originalContent {
		if !sc.dryRun {
			err = os.WriteFile(filePath, []byte(modifiedContent), 0644)
			if err != nil {
				return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write cleaned file")
			}
		}

		sc.mu.Lock()
		result.FilesChanged++
		result.TotalChanges += changesMade
		result.ChangedFiles = append(result.ChangedFiles, filePath)
		sc.mu.Unlock()

		if sc.verbose || sc.dryRun {
			action := "Cleaned"
			if sc.dryRun {
				action = "Would clean"
			}
			sc.logger.Info(fmt.Sprintf("%s sprint references", action),
				zap.String("file", filePath),
				zap.Int("changes", changesMade))
		}
	}

	return nil
}

// cleanupGoSpecificPatterns handles Go-specific cleanup patterns
func (sc *SprintCleanup) cleanupGoSpecificPatterns(content string) string {
	// Clean up package documentation
	packageDocPattern := regexp.MustCompile(`(?i)// Package \w+ implements Sprint [0-9]+ (.*?)`)
	content = packageDocPattern.ReplaceAllString(content, "// Package $1 provides $2")

	// Clean up function comments mentioning sprints
	funcCommentPattern := regexp.MustCompile(`(?i)// \w+ implements feature requirements (.*)`)
	content = funcCommentPattern.ReplaceAllString(content, "// $1 provides $2")

	// Clean up TODO comments with sprint references
	todoPattern := regexp.MustCompile(`(?i)// TODO.*Sprint [0-9]+:(.*)`)
	content = todoPattern.ReplaceAllString(content, "// TODO:$1")

	return content
}

// addError safely adds an error to the result
func (sc *SprintCleanup) addError(result *CleanupResult, errMsg string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	result.Errors = append(result.Errors, errMsg)
	sc.logger.Error("Processing error", zap.String("error", errMsg))
}

// PrintSummary prints a summary of the cleanup results
func (sc *SprintCleanup) PrintSummary(result *CleanupResult) {
	separator := strings.Repeat("=", 70)
	fmt.Printf("\n%s\n", separator)
	fmt.Printf("                SPRINT REFERENCE CLEANUP SUMMARY\n")
	fmt.Printf("%s\n\n", separator)

	if sc.dryRun {
		fmt.Printf("DRY RUN MODE - No changes were made\n\n")
	}

	fmt.Printf("Files Processed:      %d\n", result.FilesProcessed)
	fmt.Printf("Files Changed:        %d\n", result.FilesChanged)
	fmt.Printf("Total Changes:        %d\n", result.TotalChanges)
	fmt.Printf("Processing Time:      %v\n\n", result.Duration)

	if len(result.PatternMatches) > 0 {
		fmt.Printf("Pattern Matches Found:\n")
		for pattern, count := range result.PatternMatches {
			fmt.Printf("  %-40s %d\n", pattern, count)
		}
		fmt.Printf("\n")
	}

	if len(result.ChangedFiles) > 0 && (sc.verbose || sc.dryRun) {
		fmt.Printf("Changed Files:\n")
		for _, file := range result.ChangedFiles {
			fmt.Printf("  %s\n", file)
		}
		fmt.Printf("\n")
	}

	if len(result.Errors) > 0 {
		fmt.Printf("Errors Encountered:\n")
		for _, err := range result.Errors {
			fmt.Printf("  %s\n", err)
		}
		fmt.Printf("\n")
	}

	if result.TotalChanges > 0 {
		if sc.dryRun {
			fmt.Printf("✅ Sprint references identified for cleanup\n")
			fmt.Printf("   Run without --dry-run to apply changes\n")
		} else {
			fmt.Printf("✅ Sprint references successfully cleaned\n")
			fmt.Printf("   Codebase ready for production release\n")
		}
	} else {
		fmt.Printf("✅ No sprint references found - codebase is clean\n")
	}

	fmt.Printf("\n%s\n", strings.Repeat("=", 70))
}

// ValidationReport generates a validation report for the cleanup
func (sc *SprintCleanup) ValidationReport(ctx context.Context, targetDir string) error {
	sc.logger.Info("Generating validation report")

	// Re-run patterns to check for any remaining references
	result, err := sc.CleanupCodebase(ctx, targetDir)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "validation scan failed")
	}

	if result.TotalChanges > 0 {
		return gerror.New(gerror.ErrCodeValidation,
			fmt.Sprintf("Found %d remaining sprint references", result.TotalChanges), nil)
	}

	sc.logger.Info("Validation passed - no sprint references found")
	return nil
}

// Additional helper functions

func initializeLogger(verbose bool) (*zap.Logger, error) {
	if verbose {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}

// validateGoSyntax checks if Go files have valid syntax after cleanup
func validateGoSyntax(filePath string) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "invalid Go syntax after cleanup")
	}
	return nil
}

// backupFile creates a backup of the original file
func backupFile(filePath string) error {
	backupPath := filePath + ".backup." + time.Now().Format("20060102-150405")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to read file for backup")
	}

	err = os.WriteFile(backupPath, content, 0644)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create backup file")
	}

	return nil
}
