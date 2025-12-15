// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/tools"
)

// GlobTool provides file pattern matching capabilities equivalent to Claude Code's Glob tool
type GlobTool struct {
	*tools.BaseTool
	basePath string // Base path to restrict operations to
}

// GlobToolInput represents the input for glob pattern matching
type GlobToolInput struct {
	Pattern     string   `json:"pattern"`                // Glob pattern (e.g., "**/*.go", "src/**/*.js")
	Path        string   `json:"path,omitempty"`         // Optional directory to search in (defaults to current directory)
	Exclude     []string `json:"exclude,omitempty"`      // Optional exclusion patterns
	IncludeDirs bool     `json:"include_dirs,omitempty"` // Include directories in results (default: false)
}

// FileMatch represents a matched file with metadata
type FileMatch struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	IsDir        bool      `json:"is_dir"`
}

// GlobResult represents the output of glob pattern matching
type GlobResult struct {
	Files     []FileMatch `json:"files"`
	Count     int         `json:"count"`
	Pattern   string      `json:"pattern"`
	SearchDir string      `json:"search_dir"`
}

// NewGlobTool creates a new glob pattern matching tool
func NewGlobTool(basePath string) *GlobTool {
	if basePath == "" {
		// Default to current directory if none provided
		basePath, _ = os.Getwd()
	}

	// Ensure the base path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		os.MkdirAll(basePath, 0o755)
	}

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Glob pattern to match files (e.g., '**/*.go', 'src/**/*.js', '*.md')",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Optional directory to search in (defaults to current directory)",
			},
			"exclude": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Optional exclusion patterns",
			},
			"include_dirs": map[string]interface{}{
				"type":        "boolean",
				"description": "Include directories in results (default: false)",
			},
		},
		"required": []string{"pattern"},
	}

	examples := []string{
		`{"pattern": "**/*.go"}`,
		`{"pattern": "src/**/*.js", "path": "./my-project"}`,
		`{"pattern": "*.md", "exclude": ["node_modules/**", ".git/**"]}`,
		`{"pattern": "**/*.{ts,tsx}", "path": "src"}`,
		`{"pattern": "test/**/*_test.go", "exclude": ["**/mock_*"]}`,
	}

	baseTool := tools.NewBaseTool(
		"glob",
		"Fast file pattern matching tool that supports glob patterns like '**/*.js' or 'src/**/*.ts'. Returns matching file paths sorted by modification time.",
		schema,
		"filesystem",
		false,
		examples,
	)

	return &GlobTool{
		BaseTool: baseTool,
		basePath: basePath,
	}
}

// Execute runs the glob tool with the given input
func (t *GlobTool) Execute(ctx context.Context, input string) (*tools.ToolResult, error) {
	var params GlobToolInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid input").
			WithComponent("glob_tool").
			WithOperation("execute")
	}

	// Validate pattern
	if params.Pattern == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "pattern is required", nil).
			WithComponent("glob_tool").
			WithOperation("execute")
	}

	// Determine search directory
	searchDir := t.basePath
	if params.Path != "" {
		searchDir = t.sanitizePath(params.Path)
		if searchDir == "" {
			return nil, gerror.Newf(gerror.ErrCodeInvalidInput, "invalid path: %s", params.Path).
				WithComponent("glob_tool").
				WithOperation("execute")
		}
	}

	// Perform glob matching
	matches, err := t.findMatches(searchDir, params.Pattern, params.Exclude, params.IncludeDirs)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find matches").
			WithComponent("glob_tool").
			WithOperation("execute")
	}

	// Sort by modification time (newest first, like Claude Code)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ModTime.After(matches[j].ModTime)
	})

	// Prepare result
	result := GlobResult{
		Files:     matches,
		Count:     len(matches),
		Pattern:   params.Pattern,
		SearchDir: searchDir,
	}

	// Convert to JSON for output
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal result").
			WithComponent("glob_tool").
			WithOperation("execute")
	}

	metadata := map[string]string{
		"pattern":    params.Pattern,
		"search_dir": searchDir,
		"count":      fmt.Sprintf("%d", len(matches)),
	}

	return tools.NewToolResult(string(output), metadata, nil, nil), nil
}

// sanitizePath ensures the path doesn't escape the base path
func (t *GlobTool) sanitizePath(path string) string {
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

// findMatches performs the actual glob pattern matching
func (t *GlobTool) findMatches(searchDir, pattern string, excludePatterns []string, includeDirs bool) ([]FileMatch, error) {
	var matches []FileMatch

	// Check if search directory exists
	if _, err := os.Stat(searchDir); os.IsNotExist(err) {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "search directory does not exist: %s", searchDir).
			WithComponent("glob_tool").
			WithOperation("find_matches")
	}

	// Handle different types of patterns
	if strings.Contains(pattern, "**") {
		// Recursive pattern - use filepath.Walk
		err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Skip directories that can't be read
				return nil
			}

			// Skip the root directory itself
			if path == searchDir {
				return nil
			}

			// Get relative path for pattern matching
			relPath, err := filepath.Rel(searchDir, path)
			if err != nil {
				return nil
			}

			// Convert to forward slashes for consistent pattern matching
			relPath = filepath.ToSlash(relPath)

			// Check if file matches the pattern
			matched, err := t.matchPattern(relPath, pattern)
			if err != nil {
				return nil
			}

			if matched {
				// Check exclusion patterns
				excluded := false
				for _, excludePattern := range excludePatterns {
					if excludeMatched, _ := t.matchPattern(relPath, excludePattern); excludeMatched {
						excluded = true
						break
					}
				}

				if !excluded {
					// Filter out directories unless explicitly requested
					if !info.IsDir() || includeDirs {
						matches = append(matches, FileMatch{
							Path:         path,
							RelativePath: relPath,
							Size:         info.Size(),
							ModTime:      info.ModTime(),
							IsDir:        info.IsDir(),
						})
					}
				}
			}

			return nil
		})
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to walk directory").
				WithComponent("glob_tool").
				WithOperation("find_matches")
		}
	} else {
		// Simple pattern - use filepath.Glob
		globPattern := filepath.Join(searchDir, pattern)
		paths, err := filepath.Glob(globPattern)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to match pattern").
				WithComponent("glob_tool").
				WithOperation("find_matches")
		}

		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				continue // Skip files that can't be stat'd
			}

			relPath, err := filepath.Rel(searchDir, path)
			if err != nil {
				continue
			}

			// Convert to forward slashes for consistency
			relPath = filepath.ToSlash(relPath)

			// Check exclusion patterns
			excluded := false
			for _, excludePattern := range excludePatterns {
				if excludeMatched, _ := t.matchPattern(relPath, excludePattern); excludeMatched {
					excluded = true
					break
				}
			}

			if !excluded {
				// Filter out directories unless explicitly requested
				if !info.IsDir() || includeDirs {
					matches = append(matches, FileMatch{
						Path:         path,
						RelativePath: relPath,
						Size:         info.Size(),
						ModTime:      info.ModTime(),
						IsDir:        info.IsDir(),
					})
				}
			}
		}
	}

	return matches, nil
}

// matchPattern checks if a path matches a glob pattern with ** support
func (t *GlobTool) matchPattern(path, pattern string) (bool, error) {
	// Convert to forward slashes for consistent matching
	pattern = filepath.ToSlash(pattern)
	path = filepath.ToSlash(path)

	// Handle simple patterns without **
	if !strings.Contains(pattern, "**") {
		matched, err := filepath.Match(pattern, path)
		return matched, err
	}

	// For patterns with **, use a simpler approach
	return t.matchDoublestar(pattern, path), nil
}

// matchDoublestar implements a simplified doublestar matching
func (t *GlobTool) matchDoublestar(pattern, path string) bool {
	// Split pattern by **
	parts := strings.Split(pattern, "**")

	// If pattern starts with **, remove empty first part
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
		return t.matchDoublestarFromEnd(parts, path)
	}

	// If pattern ends with **, remove empty last part
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
		return t.matchDoublestarFromStart(parts, path)
	}

	// Pattern has ** in the middle
	return t.matchDoublestarMiddle(parts, path)
}

// matchDoublestarFromEnd matches when pattern starts with **
func (t *GlobTool) matchDoublestarFromEnd(parts []string, path string) bool {
	if len(parts) == 0 {
		return true // ** matches everything
	}

	// Try to match the suffix
	suffix := strings.Join(parts, "**")
	// Remove leading / from suffix
	if strings.HasPrefix(suffix, "/") {
		suffix = suffix[1:]
	}

	// Try all possible positions in the path
	pathParts := strings.Split(path, "/")
	suffixParts := strings.Split(suffix, "/")

	for i := 0; i <= len(pathParts)-len(suffixParts); i++ {
		match := true
		for j, suffixPart := range suffixParts {
			if strings.Contains(suffixPart, "**") {
				// Recursively handle more **
				subPath := strings.Join(pathParts[i+j:], "/")
				if !t.matchDoublestar(suffixPart, subPath) {
					match = false
					break
				}
				// If this matches, the rest should match too
				return true
			} else {
				matched, _ := filepath.Match(suffixPart, pathParts[i+j])
				if !matched {
					match = false
					break
				}
			}
		}
		if match {
			return true
		}
	}

	return false
}

// matchDoublestarFromStart matches when pattern ends with **
func (t *GlobTool) matchDoublestarFromStart(parts []string, path string) bool {
	if len(parts) == 0 {
		return true // ** matches everything
	}

	// Match the prefix
	prefix := strings.Join(parts, "**")
	// Remove trailing / from prefix
	if strings.HasSuffix(prefix, "/") {
		prefix = prefix[:len(prefix)-1]
	}

	prefixParts := strings.Split(prefix, "/")
	pathParts := strings.Split(path, "/")

	if len(prefixParts) > len(pathParts) {
		return false
	}

	for i, prefixPart := range prefixParts {
		if strings.Contains(prefixPart, "**") {
			// Recursively handle more **
			subPath := strings.Join(pathParts[i:], "/")
			return t.matchDoublestar(prefixPart, subPath)
		} else {
			matched, _ := filepath.Match(prefixPart, pathParts[i])
			if !matched {
				return false
			}
		}
	}

	return true
}

// matchDoublestarMiddle matches when pattern has ** in the middle
func (t *GlobTool) matchDoublestarMiddle(parts []string, path string) bool {
	if len(parts) < 2 {
		return t.matchDoublestarFromStart(parts, path)
	}

	// For simplicity, just check if the first and last parts match
	// This handles most common cases like "src/**/*.js"

	firstPart := parts[0]
	lastPart := parts[len(parts)-1]

	// Remove trailing / from first part
	if strings.HasSuffix(firstPart, "/") {
		firstPart = firstPart[:len(firstPart)-1]
	}

	// Remove leading / from last part
	if strings.HasPrefix(lastPart, "/") {
		lastPart = lastPart[1:]
	}

	pathParts := strings.Split(path, "/")

	// Check if path starts with first part
	if firstPart != "" {
		firstParts := strings.Split(firstPart, "/")
		if len(firstParts) > len(pathParts) {
			return false
		}

		for i, part := range firstParts {
			matched, _ := filepath.Match(part, pathParts[i])
			if !matched {
				return false
			}
		}
	}

	// Check if path ends with last part
	if lastPart != "" {
		lastParts := strings.Split(lastPart, "/")
		if len(lastParts) > len(pathParts) {
			return false
		}

		for i, part := range lastParts {
			pathIndex := len(pathParts) - len(lastParts) + i
			matched, _ := filepath.Match(part, pathParts[pathIndex])
			if !matched {
				return false
			}
		}
	}

	return true
}
