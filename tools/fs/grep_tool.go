// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package fs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// GrepTool provides file content search capabilities using regular expressions
type GrepTool struct {
	*tools.BaseTool
	basePath      string // Base path to restrict operations to
	maxFileSize   int64  // Maximum file size to read (default: 10MB)
	binaryChecker *binaryFileChecker
}

// GrepToolInput represents the input for grep pattern matching
type GrepToolInput struct {
	Pattern string `json:"pattern"`           // Regular expression pattern to search for
	Include string `json:"include,omitempty"` // File pattern to include (e.g., "*.js", "*.{ts,tsx}")
	Path    string `json:"path,omitempty"`    // Optional directory to search in (defaults to current directory)
}

// GrepMatch represents a file that contains matches
type GrepMatch struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	MatchCount   int       `json:"match_count"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
}

// GrepResult represents the output of grep pattern matching
type GrepResult struct {
	Files     []GrepMatch `json:"files"`
	Count     int         `json:"count"`
	Pattern   string      `json:"pattern"`
	Include   string      `json:"include,omitempty"`
	SearchDir string      `json:"search_dir"`
}

// binaryFileChecker helps detect binary files
type binaryFileChecker struct {
	// Common binary file extensions
	binaryExtensions map[string]bool
}

// newBinaryFileChecker creates a new binary file checker
func newBinaryFileChecker() *binaryFileChecker {
	return &binaryFileChecker{
		binaryExtensions: map[string]bool{
			".exe": true, ".dll": true, ".so": true, ".dylib": true,
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".ico": true,
			".mp3": true, ".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true,
			".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
			".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
			".o": true, ".a": true, ".lib": true, ".pyc": true, ".class": true,
			".db": true, ".sqlite": true, ".bin": true, ".dat": true,
		},
	}
}

// isBinary checks if a file is likely binary
func (b *binaryFileChecker) isBinary(path string) bool {
	// Check extension first
	ext := strings.ToLower(filepath.Ext(path))
	if b.binaryExtensions[ext] {
		return true
	}

	// For files without known extensions, check the first 512 bytes
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}

	// Check if mostly printable ASCII
	nonPrintable := 0
	for i := 0; i < n; i++ {
		if buf[i] < 32 && buf[i] != '\n' && buf[i] != '\r' && buf[i] != '\t' {
			nonPrintable++
		}
	}

	// If more than 30% non-printable, consider it binary
	return float64(nonPrintable)/float64(n) > 0.3
}

// NewGrepTool creates a new grep pattern matching tool
func NewGrepTool(basePath string) *GrepTool {
	if basePath == "" {
		// Default to current directory if none provided
		basePath, _ = os.Getwd()
	}

	// Ensure the base path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		os.MkdirAll(basePath, 0755)
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Regular expression pattern to search for in file contents",
			},
			"include": map[string]interface{}{
				"type":        "string",
				"description": "File pattern to include in the search (e.g., '*.js', '*.{ts,tsx}')",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Optional directory to search in (defaults to current directory)",
			},
		},
		"required": []string{"pattern"},
	}

	examples := []string{
		`{"pattern": "TODO.*fix"}`,
		`{"pattern": "function\\s+\\w+", "include": "*.js"}`,
		`{"pattern": "import.*from", "include": "*.{ts,tsx}", "path": "./src"}`,
		`{"pattern": "class\\s+\\w+\\s*{", "include": "*.java"}`,
		`{"pattern": "def\\s+test_", "include": "*.py", "path": "./tests"}`,
	}

	baseTool := tools.NewBaseTool(
		"grep",
		"Fast content search tool that searches file contents using regular expressions. Supports file filtering by pattern. Returns file paths with matches sorted by modification time.",
		schema,
		"filesystem",
		false,
		examples,
	)

	return &GrepTool{
		BaseTool:      baseTool,
		basePath:      basePath,
		maxFileSize:   10 * 1024 * 1024, // 10MB
		binaryChecker: newBinaryFileChecker(),
	}
}

// Execute runs the grep tool with the given input
func (t *GrepTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params GrepToolInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	// Validate pattern
	if params.Pattern == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "pattern is required", nil).
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	// Compile regex pattern
	regex, err := regexp.Compile(params.Pattern)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid regex pattern").
			WithComponent("grep_tool").
			WithOperation("execute").
			WithDetails("pattern", params.Pattern)
	}

	// Determine search directory
	searchDir := t.basePath
	if params.Path != "" {
		searchDir = t.sanitizePath(params.Path)
		if searchDir == "" {
			return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "invalid path: %s", params.Path).
				WithComponent("grep_tool").
				WithOperation("execute")
		}
	}

	// Find matching files
	matches, err := t.findMatches(ctx, searchDir, regex, params.Include)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find matches").
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	// Sort by modification time (newest first, like Claude Code)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ModTime.After(matches[j].ModTime)
	})

	// Prepare result
	result := GrepResult{
		Files:     matches,
		Count:     len(matches),
		Pattern:   params.Pattern,
		Include:   params.Include,
		SearchDir: searchDir,
	}

	// Convert to JSON for output
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("grep_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"pattern":    params.Pattern,
		"search_dir": searchDir,
		"count":      fmt.Sprintf("%d", len(matches)),
	}
	if params.Include != "" {
		metadata["include"] = params.Include
	}

	return tools.NewToolResult(string(output), metadata, nil, nil), nil
}

// sanitizePath ensures the path doesn't escape the base path
func (t *GrepTool) sanitizePath(path string) string {
	// Handle relative paths
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.basePath, path)
	}

	// Clean the path
	path = filepath.Clean(path)

	// Ensure the path is within the base path
	if !strings.HasPrefix(path, t.basePath) {
		return ""
	}

	return path
}

// findMatches searches for files containing the pattern
func (t *GrepTool) findMatches(ctx context.Context, searchDir string, regex *regexp.Regexp, includePattern string) ([]GrepMatch, error) {
	var matches []GrepMatch

	// Check if search directory exists
	if _, err := os.Stat(searchDir); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "search directory does not exist: %s", searchDir).
			WithComponent("grep_tool").
			WithOperation("find_matches")
	}

	// Walk the directory tree
	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			// Skip directories that can't be read
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip files larger than maxFileSize
		if info.Size() > t.maxFileSize {
			return nil
		}

		// Check if file is binary
		if t.binaryChecker.isBinary(path) {
			return nil
		}

		// Get relative path for pattern matching
		relPath, err := filepath.Rel(searchDir, path)
		if err != nil {
			return nil
		}

		// Convert to forward slashes for consistent pattern matching
		relPath = filepath.ToSlash(relPath)

		// Check include pattern if specified
		if includePattern != "" {
			matched, err := t.matchFilePattern(relPath, includePattern)
			if err != nil || !matched {
				return nil
			}
		}

		// Search file contents
		matchCount, err := t.searchFile(path, regex)
		if err != nil {
			// Skip files that can't be read
			return nil
		}

		if matchCount > 0 {
			matches = append(matches, GrepMatch{
				Path:         path,
				RelativePath: relPath,
				MatchCount:   matchCount,
				Size:         info.Size(),
				ModTime:      info.ModTime(),
			})
		}

		return nil
	})

	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to walk directory").
			WithComponent("grep_tool").
			WithOperation("find_matches")
	}

	return matches, nil
}

// searchFile searches a file for the regex pattern and returns match count
func (t *GrepTool) searchFile(path string, regex *regexp.Regexp) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	matchCount := 0
	scanner := bufio.NewScanner(file)

	// Set max buffer size to handle long lines
	const maxCapacity = 1024 * 1024 // 1MB per line
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := scanner.Text()
		if regex.MatchString(line) {
			matchCount++
		}
	}

	if err := scanner.Err(); err != nil {
		// If we hit a line too long, just skip the file
		return 0, nil
	}

	return matchCount, nil
}

// matchFilePattern checks if a file path matches the include pattern
func (t *GrepTool) matchFilePattern(path, pattern string) (bool, error) {
	// Handle brace expansion like *.{ts,tsx}
	if strings.Contains(pattern, "{") && strings.Contains(pattern, "}") {
		// Extract the parts
		prefix := pattern[:strings.Index(pattern, "{")]
		suffix := pattern[strings.Index(pattern, "}")+1:]
		braceContent := pattern[strings.Index(pattern, "{")+1 : strings.Index(pattern, "}")]

		// Split the brace content
		extensions := strings.Split(braceContent, ",")

		// Try each extension
		for _, ext := range extensions {
			fullPattern := prefix + strings.TrimSpace(ext) + suffix
			// Try matching just the filename
			if matched, _ := filepath.Match(fullPattern, filepath.Base(path)); matched {
				return true, nil
			}
			// Also try matching the full path for patterns with directories
			if strings.Contains(fullPattern, "/") {
				if matched, _ := filepath.Match(fullPattern, path); matched {
					return true, nil
				}
			}
		}
		return false, nil
	}

	// Simple glob pattern - try both filename and full path
	if matched, err := filepath.Match(pattern, filepath.Base(path)); matched || err == nil && matched {
		return matched, err
	}

	// Also try matching against the full path if pattern contains directory separator
	if strings.Contains(pattern, "/") {
		return filepath.Match(pattern, path)
	}

	return false, nil
}
